package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/service/reservation"
	"reservation-sys/service/reservation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupReservationRouter 构造与 main.go 一致的 reservation 路由。
// 包含 AuthMiddleware、RateLimitMiddleware 和真实 handler，连接到共享 MySQL/Redis。
func setupReservationRouter(t *testing.T, repo reservationdb.Repository, redisClient *redis.Client) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	svc := reservation.NewReservationService(repo)
	hdl := reservation.NewReservationHandler(svc)

	r := gin.New()
	api := r.Group("/api/reservation")

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/reservation/my", hdl.GetMyReservations)
		protected.GET("/reservation/occupied", hdl.GetOccupiedSlots)
	}

	// 写接口带宽松限流（不阻塞业务逻辑测试）
	rateLimits := []middleware.RateLimitConfig{
		{Window: 60 * time.Second, MaxRequests: 100, Dimension: middleware.RateLimitDimensionUser, KeyPrefix: "rl", HandlerName: "submit", FailOpen: true},
		{Window: 60 * time.Second, MaxRequests: 100, Dimension: middleware.RateLimitDimensionUser, KeyPrefix: "rl", HandlerName: "cancel", FailOpen: true},
	}
	submitGroup := protected.Group("")
	cancelGroup := protected.Group("")
	for i := range rateLimits {
		switch rateLimits[i].HandlerName {
		case "submit":
			submitGroup.Use(middleware.RateLimitMiddleware(redisClient, &rateLimits[i]))
		case "cancel":
			cancelGroup.Use(middleware.RateLimitMiddleware(redisClient, &rateLimits[i]))
		}
	}
	submitGroup.POST("/reservation/submit", hdl.SubmitHandler)
	cancelGroup.DELETE("/reservation/:id", hdl.Cancel)

	return r
}

func newAuthRequest(method, path, token string, body string) *http.Request {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

// ========== 提交预约 ==========
//
// 测试 POST /api/reservation/reservation/submit 提交预约接口（完整链路：中间件 -> handler -> service -> repository -> MySQL）
//
// 函数功能：验证预约提交的完整请求链路
//
// 测试场景：
// 1. 成功提交预约 — 验证返回200
// 2. 无Token返回401
// 3. 空请求体返回400
// 4. 超过4个时段返回400
// 5. 时段冲突返回400（含"已被预约"提示）

func TestReservationAPI_Submit(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	r := setupReservationRouter(t, repo, redisCli)
	token := genUserToken(t, "test_openid_submit")

	body := func(name string) string {
		return fmt.Sprintf(`{"applicant_name":"%s","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","slots":[{"start_time":"2026-06-01 08:00:00","end_time":"2026-06-01 10:00:00"}]}`, name)
	}

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("POST", "/api/reservation/reservation/submit", token, body("张三")))
		assert.Equal(t, 200, w.Code)
		var resp reservation.Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("no_token_401", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("POST", "/api/reservation/reservation/submit", "", body("张三")))
		assert.Equal(t, 401, w.Code)
	})

	t.Run("empty_body_400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("POST", "/api/reservation/reservation/submit", token, "{}"))
		assert.Equal(t, 400, w.Code)
	})

	t.Run("over_4_slots_400", func(t *testing.T) {
		b := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","slots":[
			{"start_time":"2026-06-01 08:00:00","end_time":"2026-06-01 10:00:00"},
			{"start_time":"2026-06-01 10:00:00","end_time":"2026-06-01 12:00:00"},
			{"start_time":"2026-06-01 13:00:00","end_time":"2026-06-01 15:00:00"},
			{"start_time":"2026-06-01 15:00:00","end_time":"2026-06-01 17:00:00"},
			{"start_time":"2026-06-02 08:00:00","end_time":"2026-06-02 10:00:00"}
		]}`
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("POST", "/api/reservation/reservation/submit", token, b))
		assert.Equal(t, 400, w.Code)
	})

	t.Run("conflict_400", func(t *testing.T) {
		// 先成功提交一个预约
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, newAuthRequest("POST", "/api/reservation/reservation/submit", token, body("李四")))
		assert.Equal(t, 200, w1.Code)

		// 同一时段再提交应冲突
		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, newAuthRequest("POST", "/api/reservation/reservation/submit", genUserToken(t, "other_user"), body("王五")))
		assert.Equal(t, 400, w2.Code)
		var resp reservation.Response
		json.Unmarshal(w2.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "已被预约")
	})
}

// ========== 查询我的预约 ==========
//
// 测试 GET /api/reservation/reservation/my 接口（完整链路）
//
// 函数功能：验证用户预约列表查询
//
// 测试场景：
// 1. 空列表 — 验证返回200
// 2. 有数据 — 验证返回包含"张三"
// 3. 无Token返回401

func TestReservationAPI_GetMyReservations(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	r := setupReservationRouter(t, repo, redisCli)
	token := genUserToken(t, "test_openid_my")

	t.Run("empty_list", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("GET", "/api/reservation/reservation/my", token, ""))
		assert.Equal(t, 200, w.Code)
		var resp reservation.Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("with_data", func(t *testing.T) {
		// 先提交一个预约
		body := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","slots":[{"start_time":"2026-06-01 14:00:00","end_time":"2026-06-01 16:00:00"}]}`
		w1 := httptest.NewRecorder()
		r.ServeHTTP(w1, newAuthRequest("POST", "/api/reservation/reservation/submit", token, body))
		assert.Equal(t, 200, w1.Code)

		w2 := httptest.NewRecorder()
		r.ServeHTTP(w2, newAuthRequest("GET", "/api/reservation/reservation/my", token, ""))
		assert.Equal(t, 200, w2.Code)
		var resp reservation.Response
		json.Unmarshal(w2.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
		data, _ := json.Marshal(resp.Data)
		assert.Contains(t, string(data), "张三")
	})

	t.Run("no_token_401", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("GET", "/api/reservation/reservation/my", "", ""))
		assert.Equal(t, 401, w.Code)
	})
}

// ========== 查询已占用时段 ==========
//
// 测试 GET /api/reservation/reservation/occupied 接口（完整链路）
//
// 函数功能：验证按日期查询已占用时段
//
// 测试场景：
// 1. 成功查询 — 验证返回200
// 2. 日期格式错误返回400
// 3. 无日期参数使用今天

func TestReservationAPI_GetOccupiedSlots(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	r := setupReservationRouter(t, repo, redisCli)

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("GET", "/api/reservation/reservation/occupied?date=2026-06-01", genUserToken(t, "u1"), ""))
		assert.Equal(t, 200, w.Code)
	})

	t.Run("bad_date_400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("GET", "/api/reservation/reservation/occupied?date=invalid", genUserToken(t, "u1"), ""))
		assert.Equal(t, 400, w.Code)
	})

	t.Run("no_date_uses_today", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("GET", "/api/reservation/reservation/occupied", genUserToken(t, "u1"), ""))
		assert.Equal(t, 200, w.Code)
	})
}

// ========== 取消预约 ==========
//
// 测试 DELETE /api/reservation/reservation/:id 接口（完整链路）
//
// 函数功能：验证取消预约，包含权限校验
//
// 测试场景：
// 1. 取消成功 — 验证返回200，msg包含"取消成功"
// 2. 订单不存在返回400
// 3. 他人订单无权操作返回400
// 4. 无效ID返回400
// 5. 无Token返回401

func TestReservationAPI_Cancel(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	r := setupReservationRouter(t, repo, redisCli)
	token := genUserToken(t, "test_openid_cancel")

	// 先提交一个预约
	body := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","slots":[{"start_time":"2026-06-02 08:00:00","end_time":"2026-06-02 10:00:00"}]}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newAuthRequest("POST", "/api/reservation/reservation/submit", token, body))
	require.Equal(t, 200, w.Code)
	var submitResp reservation.Response
	json.Unmarshal(w.Body.Bytes(), &submitResp)
	data, _ := json.Marshal(submitResp.Data)
	var order map[string]any
	json.Unmarshal(data, &order)
	orderID := int(order["id"].(float64))

	t.Run("success", func(t *testing.T) {
		url := fmt.Sprintf("/api/reservation/reservation/%d", orderID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("DELETE", url, token, ""))
		assert.Equal(t, 200, w.Code)
		var resp reservation.Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "取消成功")
	})

	t.Run("not_found_400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("DELETE", "/api/reservation/reservation/99999", token, ""))
		assert.Equal(t, 400, w.Code)
	})

	t.Run("wrong_user_400", func(t *testing.T) {
		otherToken := genUserToken(t, "other_user_cancel")
		url := fmt.Sprintf("/api/reservation/reservation/%d", orderID)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("DELETE", url, otherToken, ""))
		assert.Equal(t, 400, w.Code)
	})

	t.Run("invalid_id_400", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("DELETE", "/api/reservation/reservation/abc", token, ""))
		assert.Equal(t, 400, w.Code)
	})

	t.Run("no_token_401", func(t *testing.T) {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("DELETE", "/api/reservation/reservation/1", "", ""))
		assert.Equal(t, 401, w.Code)
	})
}

// ========== 限流验证（真实 Redis） ==========
//
// 测试限流中间件在真实 Redis 环境下的行为（完整链路）
//
// 函数功能：验证用户维度限流，前2次通过，第3次返回429
//
// 测试场景：
// 1. 前2次请求返回200
// 2. 第3次请求返回429

func TestReservationAPI_RateLimit(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()
	redisCli, redisCleanup := newRedisClient(t)
	defer redisCleanup()

	// 构造严格限流的路由
	svc := reservation.NewReservationService(repo)
	hdl := reservation.NewReservationHandler(svc)

	r := gin.New()
	gin.SetMode(gin.TestMode)
	api := r.Group("/api/reservation")
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	submitGroup := protected.Group("")
	submitGroup.Use(middleware.RateLimitMiddleware(redisCli, &middleware.RateLimitConfig{
		Window: 60 * time.Second, MaxRequests: 2,
		Dimension: middleware.RateLimitDimensionUser, KeyPrefix: "rl",
		HandlerName: "submit", FailOpen: true,
	}))
	submitGroup.POST("/reservation/submit", hdl.SubmitHandler)

	token := genUserToken(t, "ratelimit_user")
	body := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","slots":[{"start_time":"2026-06-03 08:00:00","end_time":"2026-06-03 10:00:00"}]}`

	for i := 1; i <= 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newAuthRequest("POST", "/api/reservation/reservation/submit", token, body))
		assert.Equal(t, 200, w.Code, "第 %d 次应通过", i)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newAuthRequest("POST", "/api/reservation/reservation/submit", token, body))
	assert.Equal(t, 429, w.Code)
}
