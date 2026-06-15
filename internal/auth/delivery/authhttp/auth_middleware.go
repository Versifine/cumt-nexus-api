package authhttp

import (
	"context"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authcontext"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
	"github.com/gin-gonic/gin"
)

type AccessTokenParser interface {
	ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error)
}

type AccessTokenValidator interface {
	ValidateAccessToken(ctx context.Context, userID userdomain.UserID, issuedAt time.Time) error
}

func RequireAuth(parser AccessTokenParser, validators ...AccessTokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawToken, err := extractBearerToken(c)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		claims, err := parser.ParseAccessToken(rawToken)
		if err != nil || claims == nil || claims.UserID == "" {
			_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "invalid token"))
			c.Abort()
			return
		}
		if err := validateAccessToken(c.Request.Context(), claims, validators); err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		ctx := authcontext.WithCurrentUserID(c.Request.Context(), claims.UserID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func OptionalAuth(parser AccessTokenParser, validators ...AccessTokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(c.GetHeader("Authorization")) == "" {
			c.Next()
			return
		}

		rawToken, err := extractBearerToken(c)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		claims, err := parser.ParseAccessToken(rawToken)
		if err != nil || claims == nil || claims.UserID == "" {
			_ = c.Error(apperr.New(apperr.CodeUnauthenticated, "invalid token"))
			c.Abort()
			return
		}
		if err := validateAccessToken(c.Request.Context(), claims, validators); err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		ctx := authcontext.WithCurrentUserID(c.Request.Context(), claims.UserID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func validateAccessToken(ctx context.Context, claims *authtoken.AccessTokenClaims, validators []AccessTokenValidator) error {
	for _, validator := range validators {
		if validator == nil {
			continue
		}
		if err := validator.ValidateAccessToken(ctx, claims.UserID, claims.IssuedAt); err != nil {
			return err
		}
	}
	return nil
}

func extractBearerToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if strings.TrimSpace(authHeader) == "" {
		return "", apperr.New(apperr.CodeUnauthenticated, "invalid auth header")
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 {
		return "", apperr.New(apperr.CodeUnauthenticated, "invalid auth header")
	}

	scheme := parts[0]
	rawToken := parts[1]

	if !strings.EqualFold(scheme, "Bearer") {
		return "", apperr.New(apperr.CodeUnauthenticated, "invalid auth header")
	}

	if strings.TrimSpace(rawToken) == "" {
		return "", apperr.New(apperr.CodeUnauthenticated, "invalid auth header")
	}

	return rawToken, nil
}
