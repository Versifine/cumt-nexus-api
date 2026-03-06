package router

import (
	"cumt-nexus-api/internal/config"
	"cumt-nexus-api/internal/controller"
	"cumt-nexus-api/internal/logger"
	"cumt-nexus-api/internal/middleware"
	"cumt-nexus-api/internal/repository"
	"cumt-nexus-api/internal/service"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	setGinMode(config.Conf.App.Mode)
	bindGinDebugPrinter()

	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		logger.Log.Warnw("设置 Gin TrustedProxies 失败", "component", "gin", "error", err)
	}

	r.Use(middleware.GinLogger())
	r.Use(middleware.Recovery())
	r.Use(middleware.Cors())

	v1 := r.Group("/api/v1")
	{
		userRepo := repository.NewUserRepository()
		userSvc := service.NewUserService(userRepo)
		userCtrl := controller.NewUserController(userSvc)
		InitUserRouter(v1, userCtrl)
	}
	return r
}

func setGinMode(mode string) {
	switch strings.ToLower(mode) {
	case gin.ReleaseMode:
		gin.SetMode(gin.ReleaseMode)
	case gin.TestMode:
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}
}

func bindGinDebugPrinter() {
	gin.DebugPrintFunc = func(format string, values ...any) {
		msg := strings.TrimSpace(fmt.Sprintf(format, values...))
		logger.Log.Debugw("Gin 调试日志", "component", "gin", "message", msg)
	}

	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		logger.Log.Debugw("Gin 路由注册",
			"component", "gin",
			"method", httpMethod,
			"path", absolutePath,
			"handler", handlerName,
			"handlers", nuHandlers,
		)
	}
}
