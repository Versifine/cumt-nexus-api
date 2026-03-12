package jwtx

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenEmpty   = errors.New("token is empty")
	ErrSecretEmpty  = errors.New("jwt secret is empty")
	ErrTokenInvalid = errors.New("token is invalid")
	ErrTokenExpired = errors.New("token is expired")
)

type UserClaims struct {
	UID  uint64 `json:"uid"`
	Role int8   `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成访问令牌。
func GenerateToken(userID uint64, role int8, secret string, expireHours int) (string, error) {
	if secret == "" {
		return "", ErrSecretEmpty
	}
	if expireHours <= 0 {
		expireHours = 24
	}

	now := time.Now()
	claims := UserClaims{
		UID:  userID,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cumt-nexus-api",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expireHours) * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign jwt failed: %w", err)
	}

	return signed, nil
}

// ParseToken 解析并校验访问令牌。
func ParseToken(tokenString, secret string) (*UserClaims, error) {
	if tokenString == "" {
		return nil, ErrTokenEmpty
	}
	if secret == "" {
		return nil, ErrSecretEmpty
	}

	claims := new(UserClaims)
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrTokenInvalid
			}
			return []byte(secret), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("parse jwt failed: %w", err)
	}
	if token == nil || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
