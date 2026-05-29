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
