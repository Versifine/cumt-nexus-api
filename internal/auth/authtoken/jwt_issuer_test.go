package authtoken

import (
	"errors"
	"testing"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTIssuerIssueAccessToken(t *testing.T) {
	secret := "test-secret"
	issuerName := "cumt-nexus-api"
	ttl := time.Hour
	now := time.Date(2030, 5, 28, 10, 30, 0, 0, time.UTC)
	userID := userdomain.NewGeneratedUserID()
	issuer := NewJWTIssuer(secret, issuerName, ttl)

	value, tokenType, expiresIn, err := issuer.IssueAccessToken(userID, now)
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}
	if value == "" {
		t.Fatal("expected token value to be non-empty")
	}
	if tokenType != "Bearer" {
		t.Fatalf("expected token type %q, got %q", "Bearer", tokenType)
	}
	if expiresIn != ttl {
		t.Fatalf("expected expires in %s, got %s", ttl, expiresIn)
	}

	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(value, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("parse issued token: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("expected issued token to be valid")
	}
	if claims.Subject != userID.String() {
		t.Fatalf("expected subject %q, got %q", userID.String(), claims.Subject)
	}
	if claims.UserID != userID.String() {
		t.Fatalf("expected user_id %q, got %q", userID.String(), claims.UserID)
	}
	if claims.Issuer != issuerName {
		t.Fatalf("expected issuer %q, got %q", issuerName, claims.Issuer)
	}
	if claims.IssuedAt == nil || !claims.IssuedAt.Time.Equal(now) {
		t.Fatalf("expected issued at %s, got %v", now, claims.IssuedAt)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(now.Add(ttl)) {
		t.Fatalf("expected expires at %s, got %v", now.Add(ttl), claims.ExpiresAt)
	}
}

func TestJWTIssuerParseAccessToken(t *testing.T) {
	secret := "test-secret"
	issuerName := "cumt-nexus-api"
	issuer := NewJWTIssuer(secret, issuerName, time.Hour)
	now := time.Now().UTC()
	userID := userdomain.NewGeneratedUserID()

	value, _, _, err := issuer.IssueAccessToken(userID, now)
	if err != nil {
		t.Fatalf("IssueAccessToken returned error: %v", err)
	}

	claims, err := issuer.ParseAccessToken(value)
	if err != nil {
		t.Fatalf("ParseAccessToken returned error: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("expected user id %q, got %q", userID.String(), claims.UserID.String())
	}
}

func TestJWTIssuerParseAccessTokenRejectsInvalidToken(t *testing.T) {
	secret := "test-secret"
	issuerName := "cumt-nexus-api"
	issuer := NewJWTIssuer(secret, issuerName, time.Hour)
	userID := userdomain.NewGeneratedUserID()
	now := time.Now().UTC()

	wrongSecretToken := mustSignToken(t, "other-secret", Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuerName,
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}, jwt.SigningMethodHS256)

	wrongIssuerToken := mustSignToken(t, secret, Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "other-issuer",
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}, jwt.SigningMethodHS256)

	expiredToken := mustSignToken(t, secret, Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuerName,
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)),
		},
	}, jwt.SigningMethodHS256)

	wrongMethodToken := mustSignToken(t, secret, Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuerName,
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}, jwt.SigningMethodHS512)

	invalidSubjectToken := mustSignToken(t, secret, Claims{
		UserID: "not-a-user-id",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuerName,
			Subject:   "not-a-user-id",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}, jwt.SigningMethodHS256)

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "blank token",
			token: "   ",
		},
		{
			name:  "malformed token",
			token: "not-a-jwt",
		},
		{
			name:  "wrong secret",
			token: wrongSecretToken,
		},
		{
			name:  "wrong issuer",
			token: wrongIssuerToken,
		},
		{
			name:  "expired token",
			token: expiredToken,
		},
		{
			name:  "wrong signing method",
			token: wrongMethodToken,
		},
		{
			name:  "invalid subject",
			token: invalidSubjectToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := issuer.ParseAccessToken(tt.token)
			if claims != nil {
				t.Fatalf("expected nil claims, got %#v", claims)
			}
			if !hasAppCode(err, apperr.CodeUnauthenticated) {
				t.Fatalf("expected unauthenticated error, got %v", err)
			}
		})
	}
}

func mustSignToken(t *testing.T, secret string, claims Claims, method jwt.SigningMethod) string {
	t.Helper()

	value, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString returned error: %v", err)
	}
	return value
}

func hasAppCode(err error, code apperr.Code) bool {
	if err == nil {
		return false
	}

	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Code() == code
}
