package middleware

import (
	"cumt-nexus-api/internal/logger"
	"cumt-nexus-api/pkg/response"
	"net"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				path := c.Request.URL.Path
				method := c.Request.Method
				clientIP := c.ClientIP()

				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				if brokenPipe {
					logger.Log.Desugar().Warn("客户端连接已断开",
						zap.String("component", "recovery"),
						zap.String("method", method),
						zap.String("path", path),
						zap.String("ip", clientIP),
						zap.Any("error", err),
					)
					c.Abort()
					return
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				logger.Log.Desugar().Error("请求发生 panic",
					zap.String("component", "recovery"),
					zap.String("method", method),
					zap.String("path", path),
					zap.String("ip", clientIP),
					zap.ByteString("request", httpRequest),
					zap.Any("panic", err),
					zap.Stack("stack"),
				)

				response.FailWithServer(c, "系统繁忙,请稍后再试")
				c.Abort()
			}
		}()

		c.Next()
	}
}
