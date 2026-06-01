package integration

import (
	"fmt"
	"testing"

	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== 管理员登录（跨服务 gRPC） ==========

// 测试 POST /api/admin/auth/login 管理员登录接口
// 完整链路: nginx → admin 服务 → gRPC → gateway 服务 → MySQL(home_xy.admins)
//
// 函数功能：验证管理员登录的完整端到端流程，包含跨服务 gRPC 调用和 JWT 签发。
//
// 测试场景：
//  1. 登录成功 — 返回200，含 token 和管理员信息
//  2. 密码错误 — 返回401
//  3. 空请求体 — 返回400

func TestAdminAPI_Login(t *testing.T) {
	skipIfNoDocker(t)
	_, cleanup := newRepo(t)
	defer cleanup()

	t.Run("success", func(t *testing.T) {
		// 1. 返回200
		// 2. token 非空
		// 3. role_text 为"一级管理员"
		body := `{"username":"admin1","password":"admin123"}`
		var result loginRespWrapper
		resp := doRequestJSON(t, "POST", "/api/admin/auth/login", "", body, &result)
		httpStatus(t, resp, 200)
		assert.NotEmpty(t, result.Data.Token, "token 不应为空")
		assert.Equal(t, "一级管理员", result.Data.RoleText)
	})

	t.Run("wrong_password_401", func(t *testing.T) {
		// 1. 返回401
		body := `{"username":"admin1","password":"wrong"}`
		resp := doRequestJSON(t, "POST", "/api/admin/auth/login", "", body, nil)
		httpStatus(t, resp, 401)
		resp.Body.Close()
	})

	t.Run("empty_body_400", func(t *testing.T) {
		// 1. 返回400
		resp := doRequestJSON(t, "POST", "/api/admin/auth/login", "", "{}", nil)
		httpStatus(t, resp, 400)
		resp.Body.Close()
	})
}

type loginRespWrapper struct {
	Code int `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Role     int    `json:"role"`
		RoleText string `json:"role_text"`
	} `json:"data"`
}

// ========== 订单列表 ==========

// 测试 GET /api/admin/orders 获取订单列表接口
// 完整链路: nginx → admin 服务 → MySQL
//
// 函数功能：验证管理员分页查询订单列表的完整端到端流程。
//
// 测试场景：
//  1. 查询全部订单 — 返回200
//  2. 按状态筛选 — 返回200
//  3. 无Token返回401

func TestAdminAPI_GetOrderList(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	token := genAdminToken(t, 1, "admin1", 1)

	// 创建测试数据
	createOrder(t, repo, reservationdb.StatusPendingLevel1, "e2e_list_1")
	createOrder(t, repo, reservationdb.StatusApproved, "e2e_list_2")

	t.Run("all_orders", func(t *testing.T) {
		// 1. 返回200
		resp := doRequestJSON(t, "GET", "/api/admin/orders?page=1&page_size=20", token, "", nil)
		httpStatus(t, resp, 200)
		resp.Body.Close()
	})

	t.Run("filter_by_status", func(t *testing.T) {
		// 1. 返回200
		resp := doRequestJSON(t, "GET", "/api/admin/orders?status=1", token, "", nil)
		httpStatus(t, resp, 200)
		resp.Body.Close()
	})

	t.Run("no_token_401", func(t *testing.T) {
		// 1. 返回401
		resp := doRequestJSON(t, "GET", "/api/admin/orders", "", "", nil)
		httpStatus(t, resp, 401)
		resp.Body.Close()
	})
}

// ========== 订单详情 ==========

// 测试 GET /api/admin/orders/:id 获取订单详情接口
// 完整链路: nginx → admin 服务 → MySQL
//
// 函数功能：验证管理员查询订单详情的完整端到端流程。
//
// 测试场景：
//  1. 查询成功 — 返回200
//  2. 订单不存在返回400
//  3. 无效ID返回400

func TestAdminAPI_GetOrderDetail(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	token := genAdminToken(t, 1, "admin1", 1)
	orderID := createOrder(t, repo, reservationdb.StatusPendingLevel1, "e2e_detail")

	t.Run("success", func(t *testing.T) {
		// 1. 返回200
		url := fmt.Sprintf("/api/admin/orders/%d", orderID)
		resp := doRequestJSON(t, "GET", url, token, "", nil)
		httpStatus(t, resp, 200)
		resp.Body.Close()
	})

	t.Run("not_found_400", func(t *testing.T) {
		// 1. 返回400
		resp := doRequestJSON(t, "GET", "/api/admin/orders/99999", token, "", nil)
		httpStatus(t, resp, 400)
		resp.Body.Close()
	})

	t.Run("invalid_id_400", func(t *testing.T) {
		// 1. 返回400
		resp := doRequestJSON(t, "GET", "/api/admin/orders/abc", token, "", nil)
		httpStatus(t, resp, 400)
		resp.Body.Close()
	})
}

// ========== 一级审核 ==========

// 测试 POST /api/admin/review/level1/:id 一级审核接口
// 完整链路: nginx → admin 服务 → MySQL
//
// 函数功能：验证一级管理员审核订单（通过/拒绝）的完整端到端流程。
//
// 测试场景：
//  1. 审核通过 — 返回200，msg 包含"通过"
//  2. 审核拒绝（含评论） — 返回200
//  3. 角色错误返回403
//  4. 无Token返回401
//  5. 重复审核（状态已变更）返回400

func TestAdminAPI_Level1Review(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	orderID := createOrder(t, repo, reservationdb.StatusPendingLevel1, "e2e_l1")

	t.Run("approve", func(t *testing.T) {
		// 1. 返回200
		// 2. msg 包含"通过"
		token := genAdminToken(t, 1, "admin1", 1)
		url := fmt.Sprintf("/api/admin/review/level1/%d", orderID)
		resp := doRequestJSON(t, "POST", url, token, `{"action":1,"comment":"通过"}`, nil)
		httpStatus(t, resp, 200)
		bodyStr := readBody(t, resp)
		strContains(t, bodyStr, "通过")
	})

	t.Run("reject_with_comment", func(t *testing.T) {
		// 1. 创建新订单，状态为待一级审核
		// 2. 审核拒绝，返回200
		orderID2 := createOrder(t, repo, reservationdb.StatusPendingLevel1, "e2e_l1_r")
		token := genAdminToken(t, 1, "admin1", 1)
		url := fmt.Sprintf("/api/admin/review/level1/%d", orderID2)
		resp := doRequestJSON(t, "POST", url, token, `{"action":2,"comment":"资料不全"}`, nil)
		httpStatus(t, resp, 200)
		resp.Body.Close()
	})

	t.Run("wrong_role_403", func(t *testing.T) {
		// 1. 二级管理员尝试一级审核，返回403
		orderID3 := createOrder(t, repo, reservationdb.StatusPendingLevel1, "e2e_l1_wr")
		token := genAdminToken(t, 2, "admin2", 2)
		url := fmt.Sprintf("/api/admin/review/level1/%d", orderID3)
		resp := doRequestJSON(t, "POST", url, token, `{"action":1}`, nil)
		httpStatus(t, resp, 403)
		resp.Body.Close()
	})

	t.Run("no_token_401", func(t *testing.T) {
		// 1. 返回401
		url := fmt.Sprintf("/api/admin/review/level1/%d", orderID)
		resp := doRequestJSON(t, "POST", url, "", `{"action":1}`, nil)
		httpStatus(t, resp, 401)
		resp.Body.Close()
	})

	t.Run("wrong_status_400", func(t *testing.T) {
		// 1. 对已审核通过的订单再次审核，返回400
		url := fmt.Sprintf("/api/admin/review/level1/%d", orderID)
		token := genAdminToken(t, 1, "admin1", 1)
		resp := doRequestJSON(t, "POST", url, token, `{"action":1,"comment":"再次通过"}`, nil)
		httpStatus(t, resp, 400)
		resp.Body.Close()
	})
}

// ========== 二级审核 ==========

// 测试 POST /api/admin/review/level2/:id 二级审核接口
// 完整链路: nginx → admin 服务 → MySQL
//
// 函数功能：验证二级管理员终审订单（通过/拒绝）的完整端到端流程。
//
// 测试场景：
//  1. 审核通过 — 返回200
//  2. 角色错误返回403

func TestAdminAPI_Level2Review(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	orderID := createOrder(t, repo, reservationdb.StatusPendingLevel2, "e2e_l2")

	t.Run("approve", func(t *testing.T) {
		// 1. 返回200
		token := genAdminToken(t, 2, "admin2", 2)
		url := fmt.Sprintf("/api/admin/review/level2/%d", orderID)
		resp := doRequestJSON(t, "POST", url, token, `{"action":1,"comment":"终审通过"}`, nil)
		httpStatus(t, resp, 200)
		resp.Body.Close()
	})

	t.Run("wrong_role_403", func(t *testing.T) {
		// 1. 一级管理员尝试二级审核，返回403
		orderID2 := createOrder(t, repo, reservationdb.StatusPendingLevel2, "e2e_l2_wr")
		token := genAdminToken(t, 1, "admin1", 1)
		url := fmt.Sprintf("/api/admin/review/level2/%d", orderID2)
		resp := doRequestJSON(t, "POST", url, token, `{"action":1}`, nil)
		httpStatus(t, resp, 403)
		resp.Body.Close()
	})
}

// ========== 设置密码 ==========

// 测试 PUT /api/admin/review/level1/:id/slots/:slotID/password 设置门锁密码接口
// 完整链路: nginx → admin 服务 → MySQL
//
// 函数功能：验证一级管理员为已通过订单的时段设置门锁密码的完整端到端流程。
//
// 测试场景：
//  1. 设置成功 — 返回200，密码持久化到数据库
//  2. 订单未通过审核返回400

func TestAdminAPI_SetPassword(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	// 创建已通过终审的订单
	orderID := createOrder(t, repo, reservationdb.StatusApproved, "e2e_pwd")
	found, err := repo.FindOrderByID(orderID)
	require.NoError(t, err)
	require.Len(t, found.Slots, 1)
	slotID := found.Slots[0].ID

	t.Run("success", func(t *testing.T) {
		// 1. 返回200
		// 2. msg 包含"设置成功"
		token := genAdminToken(t, 1, "admin1", 1)
		url := fmt.Sprintf("/api/admin/review/level1/%d/slots/%d/password", orderID, slotID)
		resp := doRequestJSON(t, "PUT", url, token, `{"password":"654321"}`, nil)
		httpStatus(t, resp, 200)

		// 验证密码持久化到数据库
		found, _ := repo.FindOrderByID(orderID)
		assert.Equal(t, "654321", found.Slots[0].Password)
		resp.Body.Close()
	})

	t.Run("not_approved_400", func(t *testing.T) {
		// 1. 创建待审核订单，返回400
		orderID2 := createOrder(t, repo, reservationdb.StatusPendingLevel1, "e2e_pwd2")
		found2, _ := repo.FindOrderByID(orderID2)
		slotID2 := found2.Slots[0].ID

		token := genAdminToken(t, 1, "admin1", 1)
		url := fmt.Sprintf("/api/admin/review/level1/%d/slots/%d/password", orderID2, slotID2)
		resp := doRequestJSON(t, "PUT", url, token, `{"password":"123456"}`, nil)
		httpStatus(t, resp, 400)
		resp.Body.Close()
	})
}
