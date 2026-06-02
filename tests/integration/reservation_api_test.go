package integration

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========== 提交预约 ==========

// 测试 POST /api/reservation/reservation/submit 提交预约接口
// 完整链路: nginx → reservation 服务 → MySQL
//
// 函数功能：验证用户通过 HTTP 提交预约申请的完整端到端流程。
//
// 测试场景：
//  1. 成功提交预约 — 返回200，msg 包含"提交成功"
//  2. 无Token返回401
//  3. 空请求体返回400
//  4. 超过4个时段返回400
//  5. 时段冲突返回400（含"已被预约"提示）

func TestReservationAPI_Submit(t *testing.T) {
	skipIfNoDocker(t)
	_, cleanup := newRepo(t)
	defer cleanup()

	token := genUserToken(t, "e2e_submit")
	bodySuccess := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"E2E测试","phone":"13800138000","attendee_count":10,"slots":[{"start_time":"2026-06-01 08:00:00","end_time":"2026-06-01 10:00:00"}]}`

	t.Run("success", func(t *testing.T) {
		// 1. 返回200
		// 2. msg 包含"提交成功"
		resp := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", token, bodySuccess, nil)
		httpStatus(t, resp, 200)
		bodyStr := readBody(t, resp)
		strContains(t, bodyStr, "提交成功")
	})

	t.Run("no_token_401", func(t *testing.T) {
		// 1. 返回401
		resp := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", "", bodySuccess, nil)
		httpStatus(t, resp, 401)
	})

	t.Run("empty_body_400", func(t *testing.T) {
		// 1. 返回400
		resp := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", token, "{}", nil)
		httpStatus(t, resp, 400)
	})

	t.Run("over_4_slots_400", func(t *testing.T) {
		// 1. 返回400
		b := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","attendee_count":10,"slots":[
			{"start_time":"2026-06-01 08:00:00","end_time":"2026-06-01 10:00:00"},
			{"start_time":"2026-06-01 10:00:00","end_time":"2026-06-01 12:00:00"},
			{"start_time":"2026-06-01 13:00:00","end_time":"2026-06-01 15:00:00"},
			{"start_time":"2026-06-01 15:00:00","end_time":"2026-06-01 17:00:00"},
			{"start_time":"2026-06-02 08:00:00","end_time":"2026-06-02 10:00:00"}
		]}`
		resp := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", token, b, nil)
		httpStatus(t, resp, 400)
	})

	t.Run("conflict_400", func(t *testing.T) {
		// 1. 第一个请求返回200
		// 2. 第二个请求返回400，msg 包含"已被预约"
		// 使用独立用户和时段避免与 success 子测试冲突
		bodyConflict := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"E2E测试","phone":"13800138000","attendee_count":10,"slots":[{"start_time":"2026-06-02 08:00:00","end_time":"2026-06-02 10:00:00"}]}`
		tokenConflict := genUserToken(t, "e2e_conflict")
		resp1 := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", tokenConflict, bodyConflict, nil)
		httpStatus(t, resp1, 200)
		resp1.Body.Close()

		otherToken := genUserToken(t, "e2e_conflict_other")
		resp2 := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", otherToken, bodyConflict, nil)
		httpStatus(t, resp2, 400)
		bodyStr := readBody(t, resp2)
		strContains(t, bodyStr, "已被预约")
	})
}

// ========== 查询我的预约 ==========

// 测试 GET /api/reservation/reservation/my 获取我的预约列表接口
// 完整链路: nginx → reservation 服务 → MySQL
//
// 函数功能：验证用户查询自己预约列表的完整端到端流程。
//
// 测试场景：
//  1. 空列表 — 返回200
//  2. 有数据 — 返回200，data 包含"张三"
//  3. 无Token返回401

func TestReservationAPI_GetMyReservations(t *testing.T) {
	skipIfNoDocker(t)
	_, cleanup := newRepo(t)
	defer cleanup()

	token := genUserToken(t, "e2e_my")

	t.Run("empty_list", func(t *testing.T) {
		// 1. 返回200
		resp := doRequestJSON(t, "GET", "/api/reservation/reservation/my", token, "", nil)
		httpStatus(t, resp, 200)
		resp.Body.Close()
	})

	t.Run("with_data", func(t *testing.T) {
		// 1. 先提交预约，返回200
		// 2. 查询列表，返回200，data 包含"张三"
		body := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","attendee_count":10,"slots":[{"start_time":"2026-06-01 14:00:00","end_time":"2026-06-01 16:00:00"}]}`
		resp1 := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", token, body, nil)
		assertOK(t, resp1)
		resp1.Body.Close()

		resp2 := doRequestJSON(t, "GET", "/api/reservation/reservation/my", token, "", nil)
		httpStatus(t, resp2, 200)
		bodyStr := readBody(t, resp2)
		strContains(t, bodyStr, "张三")
	})

	t.Run("no_token_401", func(t *testing.T) {
		// 1. 返回401
		resp := doRequestJSON(t, "GET", "/api/reservation/reservation/my", "", "", nil)
		httpStatus(t, resp, 401)
		resp.Body.Close()
	})
}

// ========== 查询已占用时段 ==========

// 测试 GET /api/reservation/reservation/occupied 获取已占用时段接口
// 完整链路: nginx → reservation 服务 → MySQL
//
// 函数功能：验证按日期查询已占用时段的完整端到端流程。
//
// 测试场景：
//  1. 成功查询 — 返回200
//  2. 日期格式错误返回400
//  3. 无日期参数使用当天

func TestReservationAPI_GetOccupiedSlots(t *testing.T) {
	skipIfNoDocker(t)
	_, cleanup := newRepo(t)
	defer cleanup()

	token := genUserToken(t, "e2e_occ")

	t.Run("success", func(t *testing.T) {
		// 1. 返回200
		resp := doRequestJSON(t, "GET", "/api/reservation/reservation/occupied?date=2026-06-01", token, "", nil)
		httpStatus(t, resp, 200)
		resp.Body.Close()
	})

	t.Run("bad_date_400", func(t *testing.T) {
		// 1. 返回400
		resp := doRequestJSON(t, "GET", "/api/reservation/reservation/occupied?date=invalid", token, "", nil)
		httpStatus(t, resp, 400)
		resp.Body.Close()
	})

	t.Run("no_date_uses_today", func(t *testing.T) {
		// 1. 返回200
		resp := doRequestJSON(t, "GET", "/api/reservation/reservation/occupied", token, "", nil)
		httpStatus(t, resp, 200)
		resp.Body.Close()
	})
}

// ========== 取消预约 ==========

// 测试 DELETE /api/reservation/reservation/:id 取消预约接口
// 完整链路: nginx → reservation 服务 → MySQL
//
// 函数功能：验证用户取消预约的完整端到端流程，包含权限校验。
//
// 测试场景：
//  1. 取消成功 — 返回200，msg 包含"取消成功"
//  2. 订单不存在返回400
//  3. 他人订单无权操作返回400
//  4. 无效ID返回400
//  5. 无Token返回401

func TestReservationAPI_Cancel(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	token := genUserToken(t, "e2e_cancel")
	body := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"测试","phone":"13800138000","attendee_count":10,"slots":[{"start_time":"2026-06-04 08:00:00","end_time":"2026-06-04 10:00:00"}]}`

	// 先提交预约获取订单 ID
	resp := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", token, body, nil)
	httpStatus(t, resp, 200)
	bodyStr := readBody(t, resp)
	strContains(t, bodyStr, "提交成功")

	// 从数据库查询已提交的订单 ID
	orders, err := repo.FindOrdersByOpenID("e2e_cancel")
	assert.NoError(t, err)
	assert.NotEmpty(t, orders, "应该能查到已提交的订单")
	orderID := orders[0].ID

	t.Run("success", func(t *testing.T) {
		// 1. 返回200
		// 2. msg 包含"取消成功"
		url := "/api/reservation/reservation/" + strconv.FormatUint(uint64(orderID), 10)
		resp := doRequestJSON(t, "DELETE", url, token, "", nil)
		httpStatus(t, resp, 200)
		bodyStr := readBody(t, resp)
		strContains(t, bodyStr, "取消成功")
	})

	t.Run("not_found_400", func(t *testing.T) {
		// 1. 返回400
		resp := doRequestJSON(t, "DELETE", "/api/reservation/reservation/99999", token, "", nil)
		httpStatus(t, resp, 400)
		resp.Body.Close()
	})

	t.Run("wrong_user_400", func(t *testing.T) {
		// 1. 他人 token 操作已取消的订单，返回400
		otherToken := genUserToken(t, "e2e_cancel_other")
		url := "/api/reservation/reservation/" + strconv.FormatUint(uint64(orderID), 10)
		resp := doRequestJSON(t, "DELETE", url, otherToken, "", nil)
		httpStatus(t, resp, 400)
		resp.Body.Close()
	})

	t.Run("invalid_id_400", func(t *testing.T) {
		// 1. 返回400
		resp := doRequestJSON(t, "DELETE", "/api/reservation/reservation/abc", token, "", nil)
		httpStatus(t, resp, 400)
		resp.Body.Close()
	})

	t.Run("no_token_401", func(t *testing.T) {
		// 1. 返回401
		resp := doRequestJSON(t, "DELETE", "/api/reservation/reservation/1", "", "", nil)
		httpStatus(t, resp, 401)
		resp.Body.Close()
	})
}

// ========== 限流验证（真实 nginx + reservation + Redis） ==========

// 测试 POST /api/reservation/reservation/submit 限流行为
// 完整链路: nginx → reservation 服务 → Redis（真实滑动窗口限流）
//
// 函数功能：验证用户维度限流在真实部署环境下的行为，每次请求使用不冲突的时段。
//
// 测试场景：
//  1. 前3次请求返回200（使用不同时段避免业务冲突）
//  2. 第4次请求返回429

func TestReservationAPI_RateLimit(t *testing.T) {
	skipIfNoDocker(t)
	_, cleanup := newRepo(t)
	defer cleanup()

	// 使用独立 openid 避免与配置中的限流冲突
	token := genUserToken(t, "e2e_ratelimit_integration")

	// 服务端配置 submit 限流为 window=60s, max=3（用户维度）
	// 每次请求使用不同时段避免业务冲突
	for i := 1; i <= 3; i++ {
		hour := 8 + i
		body := fmt.Sprintf(`{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"限流测试","phone":"13800138000","attendee_count":10,"slots":[{"start_time":"2026-06-03 %02d:00:00","end_time":"2026-06-03 %02d:00:00"}]}`, hour, hour+1)
		resp := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", token, body, nil)
		if resp.StatusCode != 200 {
			bodyStr := readBody(t, resp)
			t.Fatalf("第 %d 次请求应通过 (期望200), 实际 %d, body: %s", i, resp.StatusCode, bodyStr)
		}
		resp.Body.Close()
	}

	body := `{"applicant_name":"张三","alumni_association":"校友会","year":2020,"major":"CS","reason":"限流测试","phone":"13800138000","attendee_count":10,"slots":[{"start_time":"2026-06-03 14:00:00","end_time":"2026-06-03 15:00:00"}]}`
	resp := doRequestJSON(t, "POST", "/api/reservation/reservation/submit", token, body, nil)
	httpStatus(t, resp, 429)
	bodyStr := readBody(t, resp)
	strContains(t, bodyStr, "频繁")
}
