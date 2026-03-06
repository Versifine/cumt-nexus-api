package main

import (
	"cumt-nexus-api/internal/config"
	"cumt-nexus-api/internal/logger"
	"cumt-nexus-api/internal/repository"
	"log"
)

func main() {
	// 1. 初始化 Viper 配置并加载 .env
	if err := config.InitConfig(); err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("日志初始化失败: %v", err)
	}
	defer logger.Log.Sync() // 确保日志缓冲区的内容被写入磁盘

	// 2. 初始化数据库连接
	if err := repository.InitDB(); err != nil {
		logger.Log.Fatalf("数据库连接失败: %v", err)
	}
	// 3. 初始化 Redis 连接
	if err := repository.InitRedis(); err != nil {
		logger.Log.Fatalf("Redis 连接失败: %v", err)
	}

}
