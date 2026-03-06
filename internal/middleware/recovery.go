package middleware

import (
	"cumt-nexus-api/internal/logger"
	"cumt-nexus-api/pkg/response"
	"net"
	"net/http/httputil"
	"os"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}
				if brokenPipe {
					logger.Log.Errorf("broken pipe: %v", err)
					c.Abort()
					return
				}
				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				path := c.Request.URL.Path
				logger.Log.Errorf("[Recovery from panic]\nPath: %s\nRequest: %s\nError: %v", path, string(httpRequest), err)
				for i := 1; ; i++ {
					_, file, line, ok := runtime.Caller(i)
					if !ok {
						break
					}
					logger.Log.Errorf("  %s:%d", file, line)
				}
				c.Abort()
				response.FailWithServer(c, "系统繁忙,请稍后再试")
			}
		}()
		c.Next()
	}
}
