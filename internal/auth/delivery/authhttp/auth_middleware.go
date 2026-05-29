package authhttp

import (
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/gin-gonic/gin"
)

type AccessTokenParser interface {
	ParseAccessToken(rawToken string) (*authtoken.AccessTokenClaims, error)
}

func RequireAuth(parser AccessTokenParser) gin.HandlerFunc {
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
		SetCurrentUserID(c, claims.UserID)
		c.Next()
	}
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
