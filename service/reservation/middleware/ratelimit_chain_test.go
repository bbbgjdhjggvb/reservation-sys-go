package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupChainRouter 构造与 main.go 一致的 Gin 路由，中间件链为：
// mockAuthMiddleware（注入 openid） → 多层 RateLimitMiddleware（user+ip） → handler
//
// mockAuthMiddleware 从请求头 X-Test-Openid 读取 openid 并注入上下文，
// 模拟真实 AuthMiddleware 的行为，跳过 JWT 解析，聚焦限流链路验证。
func setupChainRouter(t *testing.T, redisClient *redis.Client, rateLimits []RateLimitConfig) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 模拟认证中间件：从测试头提取 openid 注入上下文
	mockAuth := func(c *gin.Context) {
		if openid := c.GetHeader("X-Test-Openid"); openid != "" {
			c.Set("openid", openid)
		}
		c.Next()
	}

	api := r.Group("/api/reservation")
	protected := api.Group("")
	protected.Use(mockAuth)

	// 按 HandlerName 分组，每个接口只挂对应维度的限流中间件
	submitGroup := protected.Group("")
	cancelGroup := protected.Group("")
	for i := range rateLimits {
		switch rateLimits[i].HandlerName {
		case "submit":
			submitGroup.Use(RateLimitMiddleware(redisClient, &rateLimits[i]))
		case "cancel":
			cancelGroup.Use(RateLimitMiddleware(redisClient, &rateLimits[i]))
		}
	}

	submitGroup.POST("/reservation/submit", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
	})
	cancelGroup.DELETE("/reservation/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
	})

	return r
}

// setupMiniredisClient 创建 miniredis 实例和对应的 redis.Client
func setupMiniredisClient(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	return m, client
}

// newRequest 创建带模拟认证头和可选 IP 的测试请求
func newRequest(method, path, openid, remoteAddr string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if openid != "" {
		req.Header.Set("X-Test-Openid", openid)
	}
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	return req
}

const (
	chainSubmitURL = "/api/reservation/reservation/submit"
	chainCancelURL = "/api/reservation/reservation/1"
	chainOpenid1   = "test_openid_chain_001"
	chainOpenid2   = "test_openid_chain_002"
)

// ====================================================================
// 中间件链路集成测试：验证完整的中间件链（mockAuth → RateLimit → handler）
// 使用 miniredis 模拟 Redis，无需外部依赖
// ====================================================================

// TestRateLimitChain_UserDimension 场景1：用户维度限流整条链路
func TestRateLimitChain_UserDimension(t *testing.T) {
	m, client := setupMiniredisClient(t)
	defer m.Close()

	rateLimits := []RateLimitConfig{
		{
			Window:      60 * time.Second,
			MaxRequests: 3,
			Dimension:   RateLimitDimensionUser,
			KeyPrefix:   "ratelimit",
			HandlerName: "submit",
			FailOpen:    true,
		},
	}

	r := setupChainRouter(t, client, rateLimits)

	for i := 1; i <= 3; i++ {
		w := httptest.NewRecorder()
		req := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "第 %d 次请求应返回 200", i)
	}

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, float64(429), resp["code"])
	assert.Contains(t, resp["msg"], "请求过于频繁")
}

// TestRateLimitChain_IPDimension 场景2：IP 维度限流，不同 IP 互不影响
func TestRateLimitChain_IPDimension(t *testing.T) {
	m, client := setupMiniredisClient(t)
	defer m.Close()

	rateLimits := []RateLimitConfig{
		{
			Window:      60 * time.Second,
			MaxRequests: 1,
			Dimension:   RateLimitDimensionIP,
			KeyPrefix:   "ratelimit",
			HandlerName: "submit",
			FailOpen:    true,
		},
	}

	r := setupChainRouter(t, client, rateLimits)

	w1 := httptest.NewRecorder()
	req1 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "10.0.0.1:12345")
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	req2 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "10.0.0.1:54321")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)

	w3 := httptest.NewRecorder()
	req3 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "10.0.0.2:12345")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code, "不同 IP 限流应独立生效")
}

// TestRateLimitChain_DifferentUsers 场景3：不同用户限流独立
func TestRateLimitChain_DifferentUsers(t *testing.T) {
	m, client := setupMiniredisClient(t)
	defer m.Close()

	rateLimits := []RateLimitConfig{
		{
			Window:      60 * time.Second,
			MaxRequests: 1,
			Dimension:   RateLimitDimensionUser,
			KeyPrefix:   "ratelimit",
			HandlerName: "submit",
			FailOpen:    true,
		},
	}

	r := setupChainRouter(t, client, rateLimits)

	w1 := httptest.NewRecorder()
	req1 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	req2 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)

	w3 := httptest.NewRecorder()
	req3 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid2, "")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code, "不同用户限流应独立生效")
}

// TestRateLimitChain_DifferentHandlers 场景4：不同接口限流独立
func TestRateLimitChain_DifferentHandlers(t *testing.T) {
	m, client := setupMiniredisClient(t)
	defer m.Close()

	rateLimits := []RateLimitConfig{
		{
			Window:      60 * time.Second,
			MaxRequests: 1,
			Dimension:   RateLimitDimensionUser,
			KeyPrefix:   "ratelimit",
			HandlerName: "submit",
			FailOpen:    true,
		},
		{
			Window:      60 * time.Second,
			MaxRequests: 1,
			Dimension:   RateLimitDimensionUser,
			KeyPrefix:   "ratelimit",
			HandlerName: "cancel",
			FailOpen:    true,
		},
	}

	r := setupChainRouter(t, client, rateLimits)

	w1 := httptest.NewRecorder()
	req1 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code, "submit 第1次应通过")

	w2 := httptest.NewRecorder()
	req2 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code, "submit 第2次应被限流")

	w3 := httptest.NewRecorder()
	req3 := newRequest(http.MethodDelete, chainCancelURL, chainOpenid1, "")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code, "cancel 不应受 submit 限流影响")

	w4 := httptest.NewRecorder()
	req4 := newRequest(http.MethodDelete, chainCancelURL, chainOpenid1, "")
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusTooManyRequests, w4.Code, "cancel 第2次应被限流")
}

// TestRateLimitChain_WindowSlide 场景5：窗口过期后自动恢复
func TestRateLimitChain_WindowSlide(t *testing.T) {
	m, client := setupMiniredisClient(t)
	defer m.Close()

	rateLimits := []RateLimitConfig{
		{
			Window:      2 * time.Second,
			MaxRequests: 1,
			Dimension:   RateLimitDimensionUser,
			KeyPrefix:   "ratelimit",
			HandlerName: "submit",
			FailOpen:    true,
		},
	}

	r := setupChainRouter(t, client, rateLimits)

	w1 := httptest.NewRecorder()
	req1 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	req2 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)

	time.Sleep(2 * time.Second)
	m.FastForward(2 * time.Second)

	w3 := httptest.NewRecorder()
	req3 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code, "窗口过期后请求应恢复通过")
}

// TestRateLimitChain_RedisDown_FailOpen 场景6：Redis 故障降级（保守模式）
func TestRateLimitChain_RedisDown_FailOpen(t *testing.T) {
	m, client := setupMiniredisClient(t)
	defer m.Close()

	rateLimits := []RateLimitConfig{
		{
			Window:      60 * time.Second,
			MaxRequests: 1,
			Dimension:   RateLimitDimensionUser,
			KeyPrefix:   "ratelimit",
			HandlerName: "submit",
			FailOpen:    true,
		},
	}

	r := setupChainRouter(t, client, rateLimits)

	w1 := httptest.NewRecorder()
	req1 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	m.Close()

	w2 := httptest.NewRecorder()
	req2 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code, "FailOpen=true 时 Redis 故障应放行请求")
}

// TestRateLimitChain_RedisDown_FailClosed 场景7：Redis 故障拒绝（安全模式）
func TestRateLimitChain_RedisDown_FailClosed(t *testing.T) {
	m, client := setupMiniredisClient(t)
	defer m.Close()

	rateLimits := []RateLimitConfig{
		{
			Window:      60 * time.Second,
			MaxRequests: 1,
			Dimension:   RateLimitDimensionUser,
			KeyPrefix:   "ratelimit",
			HandlerName: "submit",
			FailOpen:    false,
		},
	}

	r := setupChainRouter(t, client, rateLimits)

	w1 := httptest.NewRecorder()
	req1 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	m.Close()

	w2 := httptest.NewRecorder()
	req2 := newRequest(http.MethodPost, chainSubmitURL, chainOpenid1, "")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusInternalServerError, w2.Code, "FailOpen=false 时 Redis 故障应拒绝请求")
}

// TestRateLimitChain_Unauthenticated 场景8：未认证请求回退到 anonymous
func TestRateLimitChain_Unauthenticated(t *testing.T) {
	m, client := setupMiniredisClient(t)
	defer m.Close()

	rateLimits := []RateLimitConfig{
		{
			Window:      60 * time.Second,
			MaxRequests: 2,
			Dimension:   RateLimitDimensionUser,
			KeyPrefix:   "ratelimit",
			HandlerName: "submit",
			FailOpen:    true,
		},
	}

	r := setupChainRouter(t, client, rateLimits)

	for i := 1; i <= 2; i++ {
		w := httptest.NewRecorder()
		req := newRequest(http.MethodPost, chainSubmitURL, "", "")
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "未认证请求第 %d 次应通过", i)
	}

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPost, chainSubmitURL, "", "")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}
