package main

import (
	"cumt-nexus-api/internal/config"
	"cumt-nexus-api/internal/logger"
	"cumt-nexus-api/internal/repository"
	"cumt-nexus-api/internal/router"
	"fmt"
)

func main() {
	// 1. 初始化配置
	if err := config.InitConfig(); err != nil {
		logger.Log.Fatalw("配置初始化失败", "component", "bootstrap", "error", err)
	}

	// 2. 初始化日志系统
	if err := logger.InitLogger(); err != nil {
		logger.Log.Fatalw("日志系统初始化失败", "component", "bootstrap", "error", err)
	}
	defer logger.Sync()

	logger.Log.Infow("应用基础组件初始化完成", "mode", config.Conf.App.Mode, "port", config.Conf.App.Port)

	// 3. 初始化数据库连接
	if err := repository.InitDB(); err != nil {
		logger.Log.Fatalw("数据库连接初始化失败", "component", "mysql", "error", err)
	}
	if config.Conf.App.AutoMigrate {
		if err := repository.AutoMigrate(); err != nil {
			logger.Log.Fatalw("数据库自动迁移失败", "component", "mysql", "error", err)
		}
		logger.Log.Infow("数据库自动迁移完成", "component", "mysql")
	}

	// 4. 初始化 Redis 连接
	if err := repository.InitRedis(); err != nil {
		logger.Log.Fatalw("Redis 连接初始化失败", "component", "redis", "error", err)
	}

	// 5. 初始化 Gin 路由
	r := router.InitRouter()
	addr := fmt.Sprintf(":%d", config.Conf.App.Port)
	logger.Log.Infow("HTTP 服务启动中", "component", "http", "addr", addr, "mode", config.Conf.App.Mode)

	// 6. 启动服务器
	if err := r.Run(addr); err != nil {
		logger.Log.Fatalw("HTTP 服务启动失败", "component", "http", "addr", addr, "error", err)
	}
}
