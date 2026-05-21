package httpserver

import (
	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if c.Writer.Written() {
					c.Abort()
					return
				}
				writeError(c, apperr.New(apperr.CodeInternal, "internal server error"))
				c.Abort()
			}
		}()

		c.Next()
	}
}

func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 {
			return
		}
		if c.Writer.Written() {
			return
		}
		lastErr := c.Errors.Last()
		if lastErr == nil {
			return
		}
		writeError(c, lastErr.Err)
		c.Abort()
	}
}
