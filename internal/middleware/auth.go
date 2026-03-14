package middleware

import (
	"cumt-nexus-api/internal/config"
	"cumt-nexus-api/pkg/jwtx"
	"cumt-nexus-api/pkg/response"
	"strings"

	"github.com/gin-gonic/gin"
)

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		auth = strings.TrimSpace(auth)
		if !strings.HasPrefix(auth, "Bearer ") {
			response.Fail(c, 30001, "未登录或token无效")
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if token == "" {
			response.Fail(c, 30001, "未登录或token无效")
			c.Abort()
			return
		}
		claims, err := jwtx.ParseToken(token, config.Conf.JWT.Secret)
		if err != nil {
			response.Fail(c, 30001, "未登录或token无效")
			c.Abort()
			return
		}
		c.Set("user_id", claims.UID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
