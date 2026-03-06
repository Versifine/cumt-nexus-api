package repository

import (
	"context"
	"cumt-nexus-api/internal/config"
	"cumt-nexus-api/internal/logger"
	"fmt"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis() error {
	c := config.Conf.Redis
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	RDB = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: c.Password,
		DB:       0, // 默认使用 DB 0
		PoolSize: 100,
	})

	ctx := context.Background()

	if err := RDB.Ping(ctx).Err(); err != nil {
		logger.Log.Errorw("Redis 心跳检测失败", "component", "redis", "addr", addr, "error", err)
		return fmt.Errorf("连接 Redis 失败: %w", err)
	}

	logger.Log.Infow("Redis 连接初始化成功",
		"component", "redis",
		"addr", addr,
		"db", 0,
		"pool_size", 100,
	)

	return nil
}
