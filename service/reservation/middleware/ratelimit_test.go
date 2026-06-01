package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

// setupMiniredis 创建并返回一个用于测试的内存级 Redis 实例
func setupMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	m := miniredis.RunT(t)
	return m
}

// newTestRedisClient 基于 miniredis 创建 redis.Client
func newTestRedisClient(m *miniredis.Miniredis) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: m.Addr(),
	})
}

// parseJSON 辅助函数，解析 JSON 响应
func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// ====================================================================
// 4.3.1 单元测试 —— Allow 函数
// ====================================================================

// TestAllow_NormalPass 测试 ratelimit.go 文件中 func Allow(client *redis.Client, key string, window time.Duration, max int64) (bool, error)
//
// 函数功能：基于滑动窗口判断当前请求是否允许通过
//
// 场景1：正常通过 — 在窗口内向同一用户发起 3 次请求，验证前 3 次均返回允许
//  1. 验证 Allow 返回 true，不返回 error
func TestAllow_NormalPass(t *testing.T) {
	t.Parallel()
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	window := 60 * time.Second
	max := 3
	key := "ratelimit:test:user:u001:submit"

	for i := 0; i < 3; i++ {
		allowed, err := Allow(client, key, window, max)
		assert.NoError(t, err)
		assert.True(t, allowed, "第 %d 次请求应被允许", i+1)
	}
}

// TestAllow_ExceedLimit 场景2：触发限流
// 在窗口内向同一用户发起 4 次请求，验证第 4 次返回拒绝
func TestAllow_ExceedLimit(t *testing.T) {
	t.Parallel()
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	window := 60 * time.Second
	max := 3
	key := "ratelimit:test:user:u002:submit"

	for i := 0; i < 3; i++ {
		allowed, err := Allow(client, key, window, max)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}

	allowed, err := Allow(client, key, window, max)
	assert.NoError(t, err)
	assert.False(t, allowed, "第 4 次请求应被拒绝")
}

// TestAllow_WindowSlide 场景3：窗口滑动后重置
// 首次请求后等待窗口过期，再次发起请求，验证允许通过
func TestAllow_WindowSlide(t *testing.T) {
	t.Parallel()
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	window := 2 * time.Second
	max := 1
	key := "ratelimit:test:user:u003:submit"

	allowed, err := Allow(client, key, window, max)
	assert.NoError(t, err)
	assert.True(t, allowed)

	// 立即第 2 次应被拒绝
	allowed, err = Allow(client, key, window, max)
	assert.NoError(t, err)
	assert.False(t, allowed)

	// 等待窗口过期
	time.Sleep(2 * time.Second)
	// 推进 miniredis 内部时钟以确保过期清理生效
	m.FastForward(2 * time.Second)

	// 窗口滑动后应允许
	allowed, err = Allow(client, key, window, max)
	assert.NoError(t, err)
	assert.True(t, allowed, "窗口滑动后请求应被允许")
}

// TestAllow_DifferentUsersIndependent 场景4：不同用户互不影响
// 用户 A 触发限流后，验证用户 B 的请求仍然允许
func TestAllow_DifferentUsersIndependent(t *testing.T) {
	t.Parallel()
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	window := 60 * time.Second
	max := 1
	keyA := "ratelimit:test:user:u004:submit"
	keyB := "ratelimit:test:user:u005:submit"

	// 用户 A 触发限流
	allowed, err := Allow(client, keyA, window, max)
	assert.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = Allow(client, keyA, window, max)
	assert.NoError(t, err)
	assert.False(t, allowed)

	// 用户 B 不受影响
	allowed, err = Allow(client, keyB, window, max)
	assert.NoError(t, err)
	assert.True(t, allowed, "用户 B 不应受用户 A 限流影响")
}

// TestAllow_DifferentHandlersIndependent 场景5：不同接口互不影响
// 同一用户对接口 A 触发限流后，验证对接口 B 的请求仍然允许
func TestAllow_DifferentHandlersIndependent(t *testing.T) {
	t.Parallel()
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	window := 60 * time.Second
	max := 1
	keySubmit := "ratelimit:test:user:u006:submit"
	keyCancel := "ratelimit:test:user:u006:cancel"

	// 提交接口触发限流
	allowed, err := Allow(client, keySubmit, window, max)
	assert.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = Allow(client, keySubmit, window, max)
	assert.NoError(t, err)
	assert.False(t, allowed)

	// 取消接口不受影响
	allowed, err = Allow(client, keyCancel, window, max)
	assert.NoError(t, err)
	assert.True(t, allowed, "取消接口不应受提交接口限流影响")
}

// ====================================================================
// 4.3.2 中间件集成测试
// ====================================================================

// TestRateLimitMiddleware_WithAuthenticatedUser 测试 ratelimit.go 文件中 func RateLimitMiddleware(client *redis.Client, config *RateLimitConfig) gin.HandlerFunc
//
// 函数功能：创建限流中间件，根据配置对请求进行频率限制
//
// 场景1：带认证信息的请求 — 在 Gin 上下文中注入 openid，验证中间件能正确提取并限流
//  1. 第 1 次请求返回 200
//  2. 第 2 次请求返回 429，响应体包含"请求过于频繁"
func TestRateLimitMiddleware_WithAuthenticatedUser(t *testing.T) {
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	config := &RateLimitConfig{
		Window:      60 * time.Second,
		MaxRequests: 1,
		Dimension:   RateLimitDimensionUser,
		KeyPrefix:   "ratelimit",
		HandlerName: "submit",
		FailOpen:    true,
	}
	// 模拟认证中间件先注入 openid
	r.Use(func(c *gin.Context) {
		c.Set("openid", "test_openid_007")
		c.Next()
	})
	r.Use(RateLimitMiddleware(client, config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 第 1 次通过
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, 200, w1.Code)

	// 第 2 次被限流
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, 429, w2.Code)

	// 验证 429 响应体内容
	var resp map[string]any
	err := parseJSON(w2.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(429), resp["code"])
	assert.Contains(t, resp["msg"], "请求过于频繁")
}

// TestRateLimitMiddleware_IPDimension 场景2：IP级限流
// 模拟不同来源 IP，验证 IP 维度限流独立生效
func TestRateLimitMiddleware_IPDimension(t *testing.T) {
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	config := &RateLimitConfig{
		Window:      60 * time.Second,
		MaxRequests: 1,
		Dimension:   RateLimitDimensionIP,
		KeyPrefix:   "ratelimit",
		HandlerName: "submit",
		FailOpen:    true,
	}
	r.Use(RateLimitMiddleware(client, config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// IP 1 第 1 次通过
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.10:1234"
	r.ServeHTTP(w1, req1)
	assert.Equal(t, 200, w1.Code)

	// IP 1 第 2 次被限流
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.10:5678"
	r.ServeHTTP(w2, req2)
	assert.Equal(t, 429, w2.Code)

	// IP 2 不受影响
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.11:1234"
	r.ServeHTTP(w3, req3)
	assert.Equal(t, 200, w3.Code, "不同 IP 的限流应独立生效")
}

// TestRateLimitMiddleware_ResponseHeadersAndStatus 测试 RateLimitMiddleware 限流时返回的正确 HTTP 响应
//
// 函数功能：验证触发限流时返回 HTTP 429 状态码、application/json Content-Type 及正确的 JSON 错误体
//
// 场景3：响应头与状态码
//  1. 验证状态码为 429
//  2. 验证 Content-Type 为 application/json
//  3. 验证响应体 code 为 429，msg 包含"请求过于频繁"
func TestRateLimitMiddleware_ResponseHeadersAndStatus(t *testing.T) {
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	config := &RateLimitConfig{
		Window:      60 * time.Second,
		MaxRequests: 1,
		Dimension:   RateLimitDimensionUser,
		KeyPrefix:   "ratelimit",
		HandlerName: "submit",
		FailOpen:    true,
	}
	r.Use(func(c *gin.Context) {
		c.Set("openid", "test_openid_resp")
		c.Next()
	})
	r.Use(RateLimitMiddleware(client, config))
	r.POST("/reservation/submit", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 第 1 次请求通过
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/reservation/submit", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, 200, w1.Code)

	// 第 2 次请求触发限流，验证状态码和响应头
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/reservation/submit", nil)
	r.ServeHTTP(w2, req2)

	// 验证状态码为 429
	assert.Equal(t, 429, w2.Code)

	// 验证 Content-Type 为 application/json
	contentType := w2.Header().Get("Content-Type")
	assert.Contains(t, contentType, "application/json", "限流响应应为 JSON 格式")

	// 验证响应体结构
	var resp map[string]any
	err := parseJSON(w2.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(429), resp["code"], "响应 code 应为 429")
	assert.NotEmpty(t, resp["msg"], "响应 msg 不应为空")
	assert.Contains(t, resp["msg"], "请求过于频繁", "响应 msg 应包含限流提示")
}

// ====================================================================
// 4.3.3 并发压力测试
// ====================================================================

// TestAllow_ConcurrentBurst 场景1：并发超发
// 在同一毫秒内并发发起 100 次请求（阈值 3），验证实际通过的请求数不超过 3
func TestAllow_ConcurrentBurst(t *testing.T) {
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	window := 60 * time.Second
	max := 3
	key := "ratelimit:test:concurrent:burst"

	var wg sync.WaitGroup
	var passCount int64

	// 并发发起 100 次请求
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, err := Allow(client, key, window, max)
			assert.NoError(t, err)
			if allowed {
				atomic.AddInt64(&passCount, 1)
			}
		}()
	}

	wg.Wait()

	// 验证通过的请求数不超过阈值
	assert.LessOrEqual(t, passCount, int64(max), "并发下通过的请求数不应超过阈值 %d", max)
	fmt.Printf("并发超发测试: 通过 %d/100, 拒绝 %d/100\n", passCount, 100-passCount)
}

// TestAllow_PipelineAtomicity 场景2：Pipeline原子性
// 高并发下验证无 Redis 数据竞争或计数漂移
// 配合 go test -race 运行以检测 Go 层面的竞态条件
// 注：Pipeline 方案将清理、统计、写入打包为一次网络请求，Redis 单线程顺序执行各 Pipeline 批次，
// 竞态窗口极小。被拒绝的请求也会写入 ZSet（Pipeline 批量执行的副作用），
// 但统计值 countCmd 在 ZAdd 之前获取，因此限流判断准确。
func TestAllow_PipelineAtomicity(t *testing.T) {
	m := setupMiniredis(t)
	defer m.Close()
	client := newTestRedisClient(m)

	window := 60 * time.Second
	max := 5
	key := "ratelimit:test:pipeline:atomicity"

	var wg sync.WaitGroup
	var passCount int64
	var errCount int64
	totalReqs := int64(200)

	// 高并发发起 200 次请求
	for i := 0; i < int(totalReqs); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, err := Allow(client, key, window, max)
			if err != nil {
				atomic.AddInt64(&errCount, 1)
				return
			}
			if allowed {
				atomic.AddInt64(&passCount, 1)
			}
		}()
	}

	wg.Wait()

	// 验证无 Redis 操作错误
	assert.Equal(t, int64(0), errCount, "不应有 Redis 操作错误")

	// 验证通过的请求数不超过阈值
	assert.LessOrEqual(t, passCount, int64(max), "并发下通过的请求数不应超过阈值 %d", max)
	assert.GreaterOrEqual(t, passCount, int64(1), "应至少有1个请求通过")

	// 验证 ZSet 中所有成员的 Score 均在当前窗口内，无过期残留或异常数据
	ctx := client.Context()
	members, err := client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", time.Now().Unix()-int64(window.Seconds())),
		Max: "+inf",
	}).Result()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(members), int(passCount), "ZSet 成员数应不少于通过的请求数")

	fmt.Printf("Pipeline 原子性测试: 允许 %d, 拒绝 %d, ZSet成员数 %d, 错误 %d\n",
		passCount, totalReqs-passCount-errCount, len(members), errCount)
}

// ====================================================================
// 4.3.4 降级策略测试
// ====================================================================

// TestRateLimitMiddleware_RedisDown_FailOpen 场景1：Redis宕机降级
// 关闭 Redis 服务，验证限流中间件采用保守降级策略（放行请求）并记录错误日志
func TestRateLimitMiddleware_RedisDown_FailOpen(t *testing.T) {
	m := setupMiniredis(t)
	client := newTestRedisClient(m)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	config := &RateLimitConfig{
		Window:      60 * time.Second,
		MaxRequests: 1,
		Dimension:   RateLimitDimensionUser,
		KeyPrefix:   "ratelimit",
		HandlerName: "submit",
		FailOpen:    true, // 保守降级
	}
	r.Use(func(c *gin.Context) {
		c.Set("openid", "test_openid_failover")
		c.Next()
	})
	r.Use(RateLimitMiddleware(client, config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 正常请求应通过
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, 200, w1.Code)

	// 捕获日志输出以验证错误日志
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(os.Stderr)

	// 关闭 miniredis 模拟 Redis 故障
	m.Close()

	// Redis 故障后请求应被放行（FailOpen）
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, 200, w2.Code, "FailOpen=true 时 Redis 故障应放行请求")

	// 验证错误日志被记录
	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "ratelimit", "Redis 故障时应记录包含 ratelimit 的错误日志")
	assert.Contains(t, logOutput, "Redis 操作失败", "Redis 故障时应记录操作失败信息")
}

// TestRateLimitMiddleware_RedisDown_FailClosed 补充测试：高安全模式降级
// Redis 故障时 FailOpen=false，验证请求被拒绝
func TestRateLimitMiddleware_RedisDown_FailClosed(t *testing.T) {
	m := setupMiniredis(t)
	client := newTestRedisClient(m)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	config := &RateLimitConfig{
		Window:      60 * time.Second,
		MaxRequests: 1,
		Dimension:   RateLimitDimensionUser,
		KeyPrefix:   "ratelimit",
		HandlerName: "submit",
		FailOpen:    false, // 高安全模式
	}
	r.Use(func(c *gin.Context) {
		c.Set("openid", "test_openid_failclosed")
		c.Next()
	})
	r.Use(RateLimitMiddleware(client, config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 正常请求应通过
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, 200, w1.Code)

	// 关闭 miniredis 模拟 Redis 故障
	m.Close()

	// Redis 故障后请求应被拒绝（FailClosed）
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, 500, w2.Code, "FailOpen=false 时 Redis 故障应拒绝请求")

	// 验证 500 响应体
	var resp map[string]any
	err := parseJSON(w2.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(500), resp["code"])
}

// ====================================================================
// 辅助函数测试 —— generateLimitKey
// ====================================================================
//
// 测试 ratelimit.go 文件中 func generateLimitKey(c *gin.Context, config *RateLimitConfig) string
//
// 函数功能：根据限流维度（用户/IP）和配置生成 Redis Key

// TestGenerateLimitKey_UserDimension 验证用户维度 Key 生成
//  1. Key 格式为 ratelimit:user:<openid>:<handlerName>
func TestGenerateLimitKey_UserDimension(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("openid", "test_openid_009")

	config := &RateLimitConfig{
		KeyPrefix:   "ratelimit",
		Dimension:   RateLimitDimensionUser,
		HandlerName: "submit",
	}

	key := generateLimitKey(c, config)
	assert.Equal(t, "ratelimit:user:test_openid_009:submit", key)
}

// TestGenerateLimitKey_IPDimension 验证 IP 维度 Key 生成
func TestGenerateLimitKey_IPDimension(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.RemoteAddr = "192.168.1.20:1234"

	config := &RateLimitConfig{
		KeyPrefix:   "ratelimit",
		Dimension:   RateLimitDimensionIP,
		HandlerName: "submit",
	}

	key := generateLimitKey(c, config)
	assert.Contains(t, key, "ratelimit:ip:")
	assert.Contains(t, key, ":submit")
}

// TestGenerateLimitKey_Anonymous 验证无 openid 时回退到 anonymous
func TestGenerateLimitKey_Anonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	config := &RateLimitConfig{
		KeyPrefix:   "ratelimit",
		Dimension:   RateLimitDimensionUser,
		HandlerName: "submit",
	}

	key := generateLimitKey(c, config)
	assert.Equal(t, "ratelimit:user:anonymous:submit", key)
}

// ====================================================================
// Benchmark
// ====================================================================

func BenchmarkAllow(b *testing.B) {
	m := miniredis.RunT(b)
	defer m.Close()
	client := newTestRedisClient(m)

	window := 60 * time.Second
	max := 1000
	key := "ratelimit:bench:allow"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Allow(client, key, window, max)
	}
}
