package platform

import (
	"context"
	"fmt"
	"time"

	"reservation-sys/pkg/config"

	"github.com/go-redis/redis/v8"
)

// InitRedis 初始化 Redis 连接并检测连通性
func InitRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("Redis 连接失败 (请检查IP、端口或密码): %w", err)
	}

	return client, nil
}
