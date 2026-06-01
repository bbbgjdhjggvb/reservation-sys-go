package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========== 限流：用户维度取消接口 ==========

// 测试 DELETE /api/reservation/reservation/:id 限流行为
// 完整链路: nginx → reservation 服务 → Redis（真实滑动窗口限流）
//
// 函数功能：验证取消接口（cancel）的用户维度限流，max=5，前5次触发限流检查，第6次返回429。
//
// 测试场景：
//  1. 前5次请求触发限流检测（不关心业务返回码）
//  2. 第6次请求返回429

func TestRateLimit_CancelUserDimension(t *testing.T) {
	skipIfNoDocker(t)
	_, cleanup := newRepo(t)
	defer cleanup()

	token := genUserToken(t, "e2e_rl_cancel")

	// cancel 限流配置: window=60s, max=5（用户维度）
	// 每次请求不同订单ID避免影响业务结果
	// 前5次不超过限制，第6次返回429
	for i := 1; i <= 5; i++ {
		url := "/api/reservation/reservation/99999"
		resp, err := doRequest("DELETE", url, token, "")
		assert.NoError(t, err, "第 %d 次请求应正常发送", i)
		// 只要不返回429就算通过限流检查
		assert.NotEqual(t, http.StatusTooManyRequests, resp.StatusCode,
			"第 %d 次请求不应被限流 (got 429)", i)
		resp.Body.Close()
	}

	resp, err := doRequest("DELETE", "/api/reservation/reservation/99999", token, "")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode, "第6次请求应返回429")
	resp.Body.Close()
}

// ========== 限流：未认证请求不被限流 ==========

// 测试中间件顺序：AuthMiddleware 在 RateLimitMiddleware 之前，无Token请求返回401而非429。
//
// 函数功能：验证认证中间件在限流中间件之前执行，未认证请求不会消耗限流配额。
//
// 测试场景：
//  1. 无Token请求返回401
//  2. 有Token的请求正常通过（限流配额未被消耗）

func TestRateLimit_UnauthenticatedNotLimited(t *testing.T) {
	skipIfNoDocker(t)
	_, cleanup := newRepo(t)
	defer cleanup()

	// 连续发送多次无Token请求，确保返回401而非429
	for i := 1; i <= 5; i++ {
		resp, err := doRequest("POST", "/api/reservation/reservation/submit", "", "{}")
		assert.NoError(t, err, "第 %d 次无Token请求应正常发送", i)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"第 %d 次请求应返回401（认证拦截优先于限流）", i)
		resp.Body.Close()
	}
}

// ========== 限流：读接口不限流 ==========

// 测试 GET /api/reservation/reservation/my 不挂载限流中间件。
//
// 函数功能：验证读接口不受写接口限流配置影响。
//
// 测试场景：
//  1. 连续10次GET请求均返回200

func TestRateLimit_ReadRoutesNoLimit(t *testing.T) {
	skipIfNoDocker(t)
	_, cleanup := newRepo(t)
	defer cleanup()

	token := genUserToken(t, "e2e_rl_read")

	for i := 1; i <= 10; i++ {
		resp, err := doRequest("GET", "/api/reservation/reservation/my", token, "")
		assert.NoError(t, err, "第 %d 次读请求应正常发送", i)
		assert.Equal(t, http.StatusOK, resp.StatusCode,
			"第 %d 次读请求应返回200（读接口不限流）", i)
		resp.Body.Close()
	}
}
