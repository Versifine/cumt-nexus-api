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
		cost := time.Since(start)

		// 组装结构化的日志字段，方便后续对接日志收集系统
		// 使用 Desugar() 拿到原生的高性能 zap.Logger 来打强类型日志
		logger.Log.Desugar().Info("HTTP Request",
			zap.Int("status", c.Writer.Status()),                                 // 响应状态码
			zap.String("method", c.Request.Method),                               // GET/POST 等
			zap.String("path", path),                                             // 请求路径
			zap.String("query", query),                                           // 请求参数
			zap.String("ip", c.ClientIP()),                                       // 客户端真实 IP
			zap.String("user-agent", c.Request.UserAgent()),                      // 浏览器指纹
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()), // 如果内部有错误
			zap.Duration("cost", cost),                                           // 耗时
		)
	}
}
