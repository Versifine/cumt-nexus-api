package authtoken

import (
	"fmt"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/golang-jwt/jwt/v5"
)

type JWTIssuer struct {
	secret string
	issuer string
	ttl    time.Duration
}
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}
type AccessTokenClaims struct {
	UserID   userdomain.UserID
	IssuedAt time.Time
}

func NewJWTIssuer(secret string, issuer string, ttl time.Duration) *JWTIssuer {
	return &JWTIssuer{
		secret: secret,
		issuer: issuer,
		ttl:    ttl,
	}
}

func (i *JWTIssuer) IssueAccessToken(userID userdomain.UserID, now time.Time) (value string, tokenType string, expiresIn time.Duration, err error) {
	claims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(i.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(i.secret))
	if err != nil {
		return "", "", 0, fmt.Errorf("sign access token: %w", err)
	}
	return tokenStr, "Bearer", i.ttl, nil
}

func (i *JWTIssuer) ParseAccessToken(rawToken string) (*AccessTokenClaims, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, apperr.New(apperr.CodeUnauthenticated, "invalid token")
	}
	var claims Claims
	token, err := jwt.ParseWithClaims(
		rawToken,
		&claims,
		func(t *jwt.Token) (any, error) {
			return []byte(i.secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(i.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, apperr.New(apperr.CodeUnauthenticated, "invalid token")
	}
	if token == nil || !token.Valid {
		return nil, apperr.New(apperr.CodeUnauthenticated, "invalid token")
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return nil, apperr.New(apperr.CodeUnauthenticated, "invalid token")
	}
	userID, err := userdomain.NewUserID(claims.Subject)
	if err != nil {
		return nil, apperr.New(apperr.CodeUnauthenticated, "invalid token")
	}

	return &AccessTokenClaims{
		UserID:   userID,
		IssuedAt: claims.IssuedAt.Time,
	}, nil
}
