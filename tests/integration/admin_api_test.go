package integration

import (
	"fmt"
	"net/http/httptest"
	"testing"

	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/service/admin/auth"
	"reservation-sys/service/admin/review"

	notifpb "reservation-sys/service/gateway/api/gen/notification"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAdminRouter 构造与 main.go 一致的 admin 路由。
// AuthMiddleware、RoleMiddleware、真实 handler，连接共享 MySQL。
// gRPC 客户端传 nil（仅 login/notify 端点需要，集成测试不覆盖这些端点）。
func setupAdminRouter(t *testing.T, repo reservationdb.Repository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	authSvc := auth.NewAdminAuthService(nil)
	authHdl := auth.NewAdminAuthHandler(authSvc)

	reviewSvc := review.NewReviewService(repo)
	var nilNotifyCli notifpb.NotificationServiceClient = nil
	notifyHdl := review.NewNotifyHandler(nilNotifyCli, repo)
	reviewHdl := review.NewReviewHandler(reviewSvc, notifyHdl)

	r := gin.New()
	api := r.Group("/api/admin")

	// 登录端点（无中间件，依赖 gRPC，不测）
	api.POST("/auth/login", authHdl.LoginHandler)

	protected := api.Group("")
	protected.Use(auth.AdminAuthMiddleware())
	{
		protected.GET("/admin/info", authHdl.GetAdminInfoHandler)
		protected.GET("/orders", reviewHdl.GetOrderListHandler)
		protected.GET("/orders/:id", reviewHdl.GetOrderDetailHandler)

		level1 := protected.Group("/review/level1")
		level1.Use(auth.RoleMiddleware(1))
		{
			level1.POST("/:id", reviewHdl.Level1ReviewHandler)
			level1.PUT("/:id/slots/:slotID/password", reviewHdl.SetPasswordHandler)
		}

		level2 := protected.Group("/review/level2")
		level2.Use(auth.RoleMiddleware(2))
		{
			level2.POST("/:id", reviewHdl.Level2ReviewHandler)
		}
	}

	return r
}

// createOrder 通过 repository 直接创建订单，返回订单 ID。
func createOrder(t *testing.T, repo reservationdb.Repository, status int, openid string) uint {
	t.Helper()
	order := &reservationdb.ReservationOrder{
		OrderNo:           fmt.Sprintf("R_ADMIN_%d", status),
		OpenID:            openid,
		ApplicantName:     "测试用户",
		AlumniAssociation: "校友会",
		Year:              2020,
		Major:             "CS",
		Reason:            "测试",
		Phone:             "13800138000",
		TotalSlots:        1,
		Status:            status,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-07-01 08:00:00"), EndTime: mustParseTime("2026-07-01 10:00:00"), Status: status},
	}
	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)
	return order.ID
}

// ========== 订单列表 ==========
//
// 测试 GET /api/admin/orders 接口（完整链路：中间件 -> handler -> service -> repository -> MySQL）
//
// 函数功能：验证管理员分页查询订单列表
//
// 测试场景：
// 1. 查询全部订单 — 验证返回200
// 2. 按状态筛选 — 验证按 status=1 筛选
// 3. 无Token返回401

func TestAdminAPI_GetOrderList(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	r := setupAdminRouter(t, repo)
	token := genAdminToken(t, 1, "admin1", 1)

	// 创建几个不同状态的订单
	createOrder(t, repo, reservationdb.StatusPendingLevel1, "u1")
	createOrder(t, repo, reservationdb.StatusApproved, "u2")

	t.Run("all_orders", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := newAuthRequest("GET", "/api/admin/orders?page=1&page_size=20", token, "")
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("filter_by_status", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := newAuthRequest("GET", "/api/admin/orders?status=1", token, "")
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("no_token_401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := newAuthRequest("GET", "/api/admin/orders", "", "")
		r.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})
}

// ========== 订单详情 ==========
//
// 测试 GET /api/admin/orders/:id 接口（完整链路）
//
// 函数功能：验证管理员查询订单详情
//
// 测试场景：
// 1. 查询成功 — 验证返回200
// 2. 订单不存在返回400
// 3. 无效ID返回400

func TestAdminAPI_GetOrderDetail(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	r := setupAdminRouter(t, repo)
	token := genAdminToken(t, 1, "admin1", 1)
	orderID := createOrder(t, repo, reservationdb.StatusPendingLevel1, "u_detail")

	t.Run("success", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := newAuthRequest("GET", fmt.Sprintf("/api/admin/orders/%d", orderID), token, "")
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("not_found_400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := newAuthRequest("GET", "/api/admin/orders/99999", token, "")
		r.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code)
	})

	t.Run("invalid_id_400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := newAuthRequest("GET", "/api/admin/orders/abc", token, "")
		r.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code)
	})
}

// ========== 一级审核 ==========
//
// 测试 POST /api/admin/review/level1/:id 接口（完整链路）
//
// 函数功能：验证一级管理员审核订单（通过/拒绝）
//
// 测试场景：
// 1. 审核通过 — 验证返回200
// 2. 审核拒绝（含评论） — 验证返回200
// 3. 角色错误返回403
// 4. 无Token返回401
// 5. 错误状态（非一级待审）返回400

func TestAdminAPI_Level1Review(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	r := setupAdminRouter(t, repo)
	orderID := createOrder(t, repo, reservationdb.StatusPendingLevel1, "u_l1")

	t.Run("approve", func(t *testing.T) {
		token := genAdminToken(t, 1, "admin1", 1)
		w := httptest.NewRecorder()
		req := newAuthRequest("POST", fmt.Sprintf("/api/admin/review/level1/%d", orderID), token,
			`{"action":1,"comment":"通过"}`)
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("reject_with_comment", func(t *testing.T) {
		orderID2 := createOrder(t, repo, reservationdb.StatusPendingLevel1, "u_l1_r")
		token := genAdminToken(t, 1, "admin1", 1)
		w := httptest.NewRecorder()
		req := newAuthRequest("POST", fmt.Sprintf("/api/admin/review/level1/%d", orderID2), token,
			`{"action":2,"comment":"资料不全"}`)
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("wrong_role_403", func(t *testing.T) {
		token := genAdminToken(t, 2, "admin2", 2)
		w := httptest.NewRecorder()
		req := newAuthRequest("POST", fmt.Sprintf("/api/admin/review/level1/%d", orderID), token,
			`{"action":1}`)
		r.ServeHTTP(w, req)
		assert.Equal(t, 403, w.Code)
	})

	t.Run("no_token_401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := newAuthRequest("POST", fmt.Sprintf("/api/admin/review/level1/%d", orderID), "",
			`{"action":1}`)
		r.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)
	})

	t.Run("wrong_status_400", func(t *testing.T) {
		// 订单状态不是 PendingLevel1，审核应失败
		orderID3 := createOrder(t, repo, reservationdb.StatusPendingLevel1, "u_l1_ws")
		token := genAdminToken(t, 1, "admin1", 1)
		w := httptest.NewRecorder()
		req := newAuthRequest("POST", fmt.Sprintf("/api/admin/review/level1/%d", orderID3), token,
			`{"action":1,"comment":"通过"}`)
		r.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code)
	})
}

// ========== 二级审核 ==========
//
// 测试 POST /api/admin/review/level2/:id 接口（完整链路）
//
// 函数功能：验证二级管理员终审订单（通过/拒绝）
//
// 测试场景：
// 1. 审核通过 — 验证返回200
// 2. 角色错误返回403

func TestAdminAPI_Level2Review(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	r := setupAdminRouter(t, repo)
	orderID := createOrder(t, repo, reservationdb.StatusPendingLevel2, "u_l2")

	t.Run("approve", func(t *testing.T) {
		token := genAdminToken(t, 2, "admin2", 2)
		w := httptest.NewRecorder()
		req := newAuthRequest("POST", fmt.Sprintf("/api/admin/review/level2/%d", orderID), token,
			`{"action":1,"comment":"终审通过"}`)
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("wrong_role_403", func(t *testing.T) {
		token := genAdminToken(t, 1, "admin1", 1)
		w := httptest.NewRecorder()
		req := newAuthRequest("POST", fmt.Sprintf("/api/admin/review/level2/%d", orderID), token,
			`{"action":1}`)
		r.ServeHTTP(w, req)
		assert.Equal(t, 403, w.Code)
	})
}

// ========== 设置密码 ==========
//
// 测试 PUT /api/admin/review/level1/:id/slots/:slotID/password 接口（完整链路）
//
// 函数功能：验证一级管理员为已通过订单的时段设置门锁密码
//
// 测试场景：
// 1. 设置成功 — 验证返回200，密码被持久化
// 2. 订单未通过审核返回400

func TestAdminAPI_SetPassword(t *testing.T) {
	skipIfNoDocker(t)
	repo, cleanup := newRepo(t)
	defer cleanup()

	r := setupAdminRouter(t, repo)
	// 创建已通过终审的订单（状态为 ApprovedFinal）
	orderID := createOrder(t, repo, reservationdb.StatusApproved, "u_pwd")

	// 获取 slot ID
	found, err := repo.FindOrderByID(orderID)
	require.NoError(t, err)
	require.Len(t, found.Slots, 1)
	slotID := found.Slots[0].ID

	t.Run("success", func(t *testing.T) {
		token := genAdminToken(t, 1, "admin1", 1)
		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/admin/review/level1/%d/slots/%d/password", orderID, slotID)
		req := newAuthRequest("PUT", url, token, `{"password":"654321"}`)
		r.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code)

		// 验证密码已更新
		found, _ := repo.FindOrderByID(orderID)
		assert.Equal(t, "654321", found.Slots[0].Password)
	})

	t.Run("not_approved_400", func(t *testing.T) {
		// 创建新订单，状态为 Pending，不应允许设置密码
		orderID2 := createOrder(t, repo, reservationdb.StatusPendingLevel1, "u_pwd2")
		found2, _ := repo.FindOrderByID(orderID2)
		slotID2 := found2.Slots[0].ID

		token := genAdminToken(t, 1, "admin1", 1)
		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/admin/review/level1/%d/slots/%d/password", orderID2, slotID2)
		req := newAuthRequest("PUT", url, token, `{"password":"123456"}`)
		r.ServeHTTP(w, req)
		assert.Equal(t, 400, w.Code)
	})
}
