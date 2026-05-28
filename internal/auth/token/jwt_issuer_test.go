package token

import (
	"testing"
	"time"

	userdomain "github.com/Versifine/cumt-nexus-api/internal/user/domain"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTIssuerIssueAccessToken(t *testing.T) {
	secret := "test-secret"
	ttl := time.Hour
	now := time.Date(2030, 5, 28, 10, 30, 0, 0, time.UTC)
	userID := userdomain.NewGeneratedUserID()
	issuer := NewJWTIssuer(secret, ttl)

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
	if claims.IssuedAt == nil || !claims.IssuedAt.Time.Equal(now) {
		t.Fatalf("expected issued at %s, got %v", now, claims.IssuedAt)
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.Time.Equal(now.Add(ttl)) {
		t.Fatalf("expected expires at %s, got %v", now.Add(ttl), claims.ExpiresAt)
	}
}
