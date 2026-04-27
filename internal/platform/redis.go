package platform

import (
	"context"
	"fmt"
	"time"

	"reservation-sys/internal/config"

	"github.com/go-redis/redis/v8"
)

// 全局可用的 Redis 客户端
var RedisClient *redis.Client

// InitRedis 初始化 Redis 连接并检测连通性
func InitRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	// 拼接正确的地址格式，例如 "127.0.0.1:6379"
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password, // 没有密码则为空
		DB:       cfg.DB,       // 默认是 0
	})

	// 设置一个 5 秒超时的上下文，防止程序连不上 Redis 一直卡死
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping 一下，测试网络和密码是否正确
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("Redis 连接失败 (请检查IP、端口或密码): %w", err)
	}

	// 如果连接成功 RedisClient 必定不为nil
	RedisClient = client
	return client, nil
}
