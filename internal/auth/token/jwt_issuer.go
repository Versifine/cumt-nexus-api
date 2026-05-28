package token

import (
	"fmt"
	"time"

	userdomain "github.com/Versifine/cumt-nexus-api/internal/user/domain"
	"github.com/golang-jwt/jwt/v5"
)

type JWTIssuer struct {
	secret string
	ttl    time.Duration
}
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJWTIssuer(secret string, ttl time.Duration) *JWTIssuer {
	return &JWTIssuer{
		secret: secret,
		ttl:    ttl,
	}
}

func (i *JWTIssuer) IssueAccessToken(userID userdomain.UserID, now time.Time) (value string, tokenType string, expiresIn time.Duration, err error) {
	claims := Claims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cumt-nexus-api",
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
