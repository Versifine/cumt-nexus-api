package middleware

import (
	"cumt-nexus-api/internal/logger"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery // 获取 ? 后面的参数

		// 挂起当前中间件，执行后续的 Controller 业务逻辑
		c.Next()

		// 业务逻辑执行完后，收集数据
		costMs := time.Since(start).Milliseconds()
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int64("latency_ms", costMs),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("error", c.Errors.String()))
		}

		base := logger.Log.Desugar()

		// 根据 HTTP 状态码映射日志级别，统一日志语义
		switch {
		case status >= 500:
			base.Error("HTTP 请求完成", fields...)
		case status >= 400:
			base.Warn("HTTP 请求完成", fields...)
		default:
			base.Info("HTTP 请求完成", fields...)
		}
	}
}
