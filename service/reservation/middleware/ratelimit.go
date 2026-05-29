// Package middleware 提供预约服务的中间件
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

//go:embed scripts/ratelimit.lua
var rateLimitLuaScript string

// RateLimitDimension 限流维度
const (
	RateLimitDimensionUser = "user"
	RateLimitDimensionIP   = "ip"
)

// rateLimitScript 滑动窗口限流 Lua 脚本，从 scripts/ratelimit.lua 嵌入
var rateLimitScript = redis.NewScript(rateLimitLuaScript)

// RateLimitConfig 限流配置
// Window: 时间窗口大小
// MaxRequests: 窗口内允许的最大请求数
// Dimension: 限流维度 (user / ip)
// KeyPrefix: Redis Key 前缀
// HandlerName: 接口标识，用于区分不同接口的限流桶
// FailOpen: Redis 故障时是否放行（保守降级）

// 比 BaseConfig 中的 RateLimitConfig 多了 KeyPrefix
type RateLimitConfig struct {
	Window      time.Duration
	MaxRequests int
	Dimension   string
	KeyPrefix   string
	HandlerName string
	FailOpen    bool
}

// RateLimitMiddleware 返回 Gin 限流中间件
// 参数:
//   - redisClient: Redis 客户端
//   - config: 限流配置
//
// 返回值:
//   - gin.HandlerFunc: Gin 中间件函数
func RateLimitMiddleware(redisClient *redis.Client, config *RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成限流 Redis Key
		key := generateLimitKey(c, config)

		// 判断是否允许请求通过
		allowed, err := Allow(redisClient, key, config.Window, config.MaxRequests)
		if err != nil {
			log.Printf("[error][ratelimit] Redis 操作失败: %v", err)
			if config.FailOpen {
				// 保守降级：Redis 故障时放行请求
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

// Allow 执行滑动窗口限流核心逻辑
// 使用 Redis Lua 脚本在服务端原子执行：清理过期记录、统计窗口内请求数、判断超限、写入记录、设置过期时间
// 参数:
//   - redisClient: Redis 客户端
//   - key: 限流 Redis Key
//   - window: 时间窗口大小
//   - max: 窗口内最大请求数
//
// 返回值:
//   - bool: 是否允许本次请求通过
//   - error: Redis 操作错误
func Allow(redisClient *redis.Client, key string, window time.Duration, max int) (bool, error) {
	ctx := redisClient.Context()
	now := time.Now().Unix()

	// 使用时间戳+随机数确保 Member 唯一
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

// generateLimitKey 根据限流维度和请求上下文生成 Redis Key
// 参数:
//   - c: Gin 上下文
//   - config: 限流配置
//
// 返回值:
//   - string: 限流 Redis Key，格式为 "prefix:dimension:identifier:handler"
func generateLimitKey(c *gin.Context, config *RateLimitConfig) string {
	var identifier string

	switch config.Dimension {
	case RateLimitDimensionUser:
		// 从认证中间件注入的上下文中获取 OpenID
		if openid, exists := c.Get("openid"); exists {
			identifier = openid.(string)
		} else {
			identifier = "anonymous"
		}
	case RateLimitDimensionIP:
		// 获取客户端真实 IP（优先从 X-Forwarded-For 头获取，兼容 Nginx 反向代理）
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
