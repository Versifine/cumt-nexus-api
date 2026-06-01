package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("panic recovered",
					"request_id", RequestID(c),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"status", http.StatusInternalServerError,
					"panic", recovered,
				)
				if c.Writer.Written() {
					c.Abort()
					return
				}
				err := apperr.New(apperr.CodeInternal, "internal server error")
				writeError(c, err)

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

func RequestLoggerMiddleware(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		log.Info(
			"http request",
			"request_id", RequestID(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
		)
	}
}

const RequestIDKey = "request_id"
const RequestIDHeader = "X-Request-ID"

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)
		c.Next()
	}
}

func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := newOriginAllowList(allowedOrigins)

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin == "" || len(allowed) == 0 {
			c.Next()
			return
		}

		if allowed["*"] || allowed[origin] {
			c.Header("Access-Control-Allow-Origin", corsAllowOrigin(origin, allowed["*"]))
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Expose-Headers", RequestIDHeader)
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func newOriginAllowList(origins []string) map[string]bool {
	allowed := make(map[string]bool, len(origins))
	for _, origin := range origins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = true
		}
	}
	return allowed
}

func corsAllowOrigin(origin string, allowWildcard bool) string {
	if allowWildcard {
		return "*"
	}
	return origin
}

func generateRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}
