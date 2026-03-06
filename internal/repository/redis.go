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
		logger.Log.Error("连接 Redis 失败: %w", err)
		return fmt.Errorf("连接 Redis 失败: %w", err)
	} else {
		logger.Log.Info("Redis 连接成功")
	}

	return nil
}
