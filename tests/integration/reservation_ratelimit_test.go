package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"reservation-sys/service/reservation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

// buildRateLimitRouter 构造带限流中间件的简化路由。
func buildRateLimitRouter(t *testing.T, redisCli *redis.Client, rateLimits []middleware.RateLimitConfig) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	api := r.Group("/api/reservation")
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/reservation/my", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	submitGroup := protected.Group("")
	cancelGroup := protected.Group("")
	for i := range rateLimits {
		switch rateLimits[i].HandlerName {
		case "submit":
			submitGroup.Use(middleware.RateLimitMiddleware(redisCli, &rateLimits[i]))
		case "cancel":
			cancelGroup.Use(middleware.RateLimitMiddleware(redisCli, &rateLimits[i]))
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

// 测试限流中间件在真实 Redis 环境下的用户维度限流（完整链路：AuthMiddleware -> RateLimitMiddleware -> handler -> Redis）
//
// 函数功能：验证用户维度限流，max=3，前3次通过，第4次返回429
//
// TestRateLimit_UserDimension 前3次请求返回200，第4次返回429
//  1. 验证第4次响应 code 为 429
func TestRateLimit_UserDimension(t *testing.T) {
	skipIfNoDocker(t)
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	rateLimits := []middleware.RateLimitConfig{
		{Window: 60 * time.Second, MaxRequests: 3, Dimension: middleware.RateLimitDimensionUser, KeyPrefix: "rl", HandlerName: "submit", FailOpen: true},
	}
	r := buildRateLimitRouter(t, redisCli, rateLimits)
	token := genUserToken(t, "rl_user_001")

	for i := 1; i <= 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest(http.MethodPost, "/api/reservation/reservation/submit", token, ""))
		assert.Equal(t, http.StatusOK, w.Code, "第 %d 次应返回 200", i)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newAuthRequest(http.MethodPost, "/api/reservation/reservation/submit", token, ""))
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, float64(429), resp["code"])
}

// 测试 IP 维度限流，max=1，同一IP第2次返回429，不同IP不受影响
//
// 函数功能：验证 IP 维度的限流独立生效
//
//  1. 同一IP第2次请求返回429
//  2. 不同IP请求返回200
func TestRateLimit_IPDimension(t *testing.T) {
	skipIfNoDocker(t)
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	rateLimits := []middleware.RateLimitConfig{
		{Window: 60 * time.Second, MaxRequests: 1, Dimension: middleware.RateLimitDimensionIP, KeyPrefix: "rl", HandlerName: "submit", FailOpen: true},
	}
	r := buildRateLimitRouter(t, redisCli, rateLimits)
	token := genUserToken(t, "rl_ip_user")

	req1 := newAuthRequest(http.MethodPost, "/api/reservation/reservation/submit", token, "")
	req1.RemoteAddr = "10.0.1.1:12345"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	req2 := newAuthRequest(http.MethodPost, "/api/reservation/reservation/submit", token, "")
	req2.RemoteAddr = "10.0.1.1:54321"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)

	req3 := newAuthRequest(http.MethodPost, "/api/reservation/reservation/submit", token, "")
	req3.RemoteAddr = "10.0.1.2:12345"
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code, "不同 IP 限流应独立生效")
}

// 测试不同接口（submit/cancel）限流独立，submit 限流不影响 cancel
//
// 函数功能：验证不同 handler 的限流计数互相独立
//
//  1. submit 第2次返回429
//  2. cancel 第1次返回200
//  3. cancel 第2次返回429
func TestRateLimit_DifferentHandlers(t *testing.T) {
	skipIfNoDocker(t)
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	rateLimits := []middleware.RateLimitConfig{
		{Window: 60 * time.Second, MaxRequests: 1, Dimension: middleware.RateLimitDimensionUser, KeyPrefix: "rl", HandlerName: "submit", FailOpen: true},
		{Window: 60 * time.Second, MaxRequests: 1, Dimension: middleware.RateLimitDimensionUser, KeyPrefix: "rl", HandlerName: "cancel", FailOpen: true},
	}
	r := buildRateLimitRouter(t, redisCli, rateLimits)
	token := genUserToken(t, "rl_handler_user")

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, newAuthRequest(http.MethodPost, "/api/reservation/reservation/submit", token, ""))
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, newAuthRequest(http.MethodPost, "/api/reservation/reservation/submit", token, ""))
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)

	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, newAuthRequest(http.MethodDelete, "/api/reservation/reservation/1", token, ""))
	assert.Equal(t, http.StatusOK, w3.Code, "cancel 不应受 submit 限流影响")

	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, newAuthRequest(http.MethodDelete, "/api/reservation/reservation/1", token, ""))
	assert.Equal(t, http.StatusTooManyRequests, w4.Code)
}

// 测试未认证请求的行为：AuthMiddleware 返回 401，不会触发限流
//
// 函数功能：验证未认证用户不受限流影响（由 AuthMiddleware 拦截）
//
//  1. 无Token请求返回401，不触发429
func TestRateLimit_Unauthenticated(t *testing.T) {
	skipIfNoDocker(t)
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	rateLimits := []middleware.RateLimitConfig{
		{Window: 60 * time.Second, MaxRequests: 2, Dimension: middleware.RateLimitDimensionUser, KeyPrefix: "rl", HandlerName: "submit", FailOpen: true},
	}
	r := buildRateLimitRouter(t, redisCli, rateLimits)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newAuthRequest(http.MethodPost, "/api/reservation/reservation/submit", "", ""))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// 测试读接口（GET /my）不挂载限流中间件，多次请求均通过
//
// 函数功能：验证未限流的读接口不受限流影响
//
//  1. 连续 5 次请求均返回 200
func TestRateLimit_ReadRoutesNoLimit(t *testing.T) {
	skipIfNoDocker(t)
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	rateLimits := []middleware.RateLimitConfig{
		{Window: 60 * time.Second, MaxRequests: 1, Dimension: middleware.RateLimitDimensionUser, KeyPrefix: "rl", HandlerName: "submit", FailOpen: true},
	}
	r := buildRateLimitRouter(t, redisCli, rateLimits)
	token := genUserToken(t, "rl_read_user")

	for i := 1; i <= 5; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest(http.MethodGet, "/api/reservation/reservation/my", token, ""))
		assert.Equal(t, http.StatusOK, w.Code, "读接口第 %d 次应通过", i)
	}
}
