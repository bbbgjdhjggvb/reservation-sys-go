package middleware

import (
	_ "embed"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

//go:embed ratelimit.lua
var rateLimitLuaScript string

// 限流维度常量
const (
	RateLimitDimensionUser = "user"
	RateLimitDimensionIP   = "ip"
)

// rateLimitScript 滑动窗口限流 Lua 脚本，从 ratelimit.lua 嵌入
var rateLimitScript = redis.NewScript(rateLimitLuaScript)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Window      time.Duration
	MaxRequests int
	Dimension   string
	KeyPrefix   string
	HandlerName string
	FailOpen    bool
}

// RateLimit 返回 Gin 限流中间件。
func RateLimit(redisClient *redis.Client, config *RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := generateLimitKey(c, config)

		allowed, err := allow(redisClient, key, config.Window, config.MaxRequests)
		if err != nil {
			log.Printf("[error][ratelimit] Redis 操作失败: %v", err)
			if config.FailOpen {
				c.Next()
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  "服务暂时不可用，请稍后重试",
			})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code": 429,
				"msg":  "请求过于频繁，请稍后重试",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow 执行滑动窗口限流核心逻辑。
func allow(redisClient *redis.Client, key string, window time.Duration, max int) (bool, error) {
	ctx := redisClient.Context()
	now := time.Now().Unix()
	member := fmt.Sprintf("%d:%d", now, rand.Intn(100000))

	result, err := rateLimitScript.Run(ctx, redisClient, []string{key},
		int64(window.Seconds()),
		max,
		now,
		member,
	).Int64()
	if err != nil {
		return false, fmt.Errorf("lua script exec failed: %w", err)
	}

	return result == 1, nil
}

// generateLimitKey 根据限流维度和请求上下文生成 Redis Key。
func generateLimitKey(c *gin.Context, config *RateLimitConfig) string {
	var identifier string

	switch config.Dimension {
	case RateLimitDimensionUser:
		if openid, exists := c.Get("openid"); exists {
			identifier = openid.(string)
		} else {
			identifier = "anonymous"
		}
	case RateLimitDimensionIP:
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = "unknown"
		}
		identifier = clientIP
	default:
		identifier = "default"
	}

	return fmt.Sprintf("%s:%s:%s:%s", config.KeyPrefix, config.Dimension, identifier, config.HandlerName)
}
