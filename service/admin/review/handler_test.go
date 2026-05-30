package review

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"reservation-sys/pkg/jwt"
	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/service/admin/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// injectAdmin 注入管理员 claims 的中间件
func injectAdmin(adminID uint, role int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("admin", &jwt.AdminClaims{AdminID: adminID, Username: "admin1", Role: role})
		c.Next()
	}
}

func setupReviewTestHandler(t *testing.T) (*gomock.Controller, *MockRepository, *MockNotificationServiceClient, *ReviewHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	mockRepo := NewMockRepository(ctrl)
	mockNotify := NewMockNotificationServiceClient(ctrl)
	svc := NewReviewService(mockRepo)
	notifyHdl := NewNotifyHandler(mockNotify, mockRepo)
	hdl := NewReviewHandler(svc, notifyHdl)
	r := gin.New()
	return ctrl, mockRepo, mockNotify, hdl, r
}

// getOrder 创建测试用订单
func getOrder(id uint, status int) *reservationdb.ReservationOrder {
	return &reservationdb.ReservationOrder{
		ID:                id,
		OrderNo:           "R202605010000000001",
		OpenID:            "test_openid",
		ApplicantName:     "张三",
		AlumniAssociation: "计算机与软件学院校友会",
		Year:              2015,
		Major:             "软件工程",
		Phone:             "13800138000",
		Reason:            "测试",
		TotalSlots:        1,
		Status:            status,
		Slots: []reservationdb.ReservationSlot{
			{ID: 10, Status: status},
		},
	}
}

// 测试 handler.go 文件中 func (h *ReviewHandler) GetOrderListHandler(c *gin.Context)
//
// 函数功能：分页查询订单列表，支持按状态筛选
//
// 测试场景：
// 1. 查询全部订单 — 验证返回200
// 2. 按状态筛选 — 验证按 status=1 筛选
// 3. 数据库错误 — 验证返回500
// 4. 负数status退化为查询全部
func TestGetOrderListHandler(t *testing.T) {
	t.Run("all_orders", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.GET("/orders", injectAdmin(1, 1), hdl.GetOrderListHandler)

		orders := []*reservationdb.ReservationOrder{getOrder(1, reservationdb.StatusPendingLevel1)}
		mockRepo.EXPECT().ListOrders([]int(nil), 1, 20).Return(orders, int64(1), nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/orders?page=1&page_size=20", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("filtered_by_status", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.GET("/orders", injectAdmin(1, 1), hdl.GetOrderListHandler)

		mockRepo.EXPECT().ListOrders([]int{reservationdb.StatusPendingLevel1}, 1, 20).Return(nil, int64(0), nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/orders?status=1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("db_error", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.GET("/orders", injectAdmin(1, 1), hdl.GetOrderListHandler)

		mockRepo.EXPECT().ListOrders(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, int64(0), errors.New("db error"))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/orders", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 500, w.Code)
	})

	t.Run("all_negative_status_falls_back_to_all", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.GET("/orders", injectAdmin(1, 1), hdl.GetOrderListHandler)

		mockRepo.EXPECT().ListOrders([]int(nil), 1, 20).Return(nil, int64(0), nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/orders?status=-1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

// 测试 handler.go 文件中 func (h *ReviewHandler) GetOrderDetailHandler(c *gin.Context)
//
// 函数功能：查询订单详情（含审核记录）
//
// 测试场景：
// 1. 查询成功 — 验证返回200
// 2. 无效ID — 验证返回400
// 3. 订单不存在 — 验证返回400
func TestGetOrderDetailHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.GET("/orders/:id", injectAdmin(1, 1), hdl.GetOrderDetailHandler)

		order := getOrder(1, reservationdb.StatusPendingLevel1)
		records := []reservationdb.ReviewRecord{{ID: 1, OrderID: 1, ReviewerID: 1, Comment: "通过"}}
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(order, nil)
		mockRepo.EXPECT().FindReviewRecordsByOrderID(uint(1)).Return(records, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/orders/1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.GET("/orders/:id", hdl.GetOrderDetailHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/orders/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("not_found", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.GET("/orders/:id", hdl.GetOrderDetailHandler)

		mockRepo.EXPECT().FindOrderByID(uint(999)).Return(nil, errors.New("record not found"))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/orders/999", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})
}

// 测试 handler.go 文件中 func (h *ReviewHandler) Level1ReviewHandler(c *gin.Context)
//
// 函数功能：处理一级审核（通过/拒绝），仅角色1管理员可操作
//
// 测试场景：
// 1. 审核通过 — 验证状态从 PendingLevel1 变为 PendingLevel2
// 2. 未登录 — 验证返回401
// 3. 角色错误 — 角色2调用返回403
// 4. 无效ID — 验证返回400
// 5. 错误的请求体 — 验证返回400
// 6. 服务层错误 — 验证返回400
func TestLevel1ReviewHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.POST("/review/level1/:id", injectAdmin(1, 1), hdl.Level1ReviewHandler)

		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(getOrder(1, reservationdb.StatusPendingLevel1), nil)
		mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel1, reservationdb.StatusPendingLevel2).Return(nil)
		mockRepo.EXPECT().CreateReviewRecord(gomock.Any()).Return(nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1", strings.NewReader(`{"action":1,"comment":"通过"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("not_logged_in", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.POST("/review/level1/:id", hdl.Level1ReviewHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1", strings.NewReader(`{"action":1}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	t.Run("wrong_role", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.POST("/review/level1/:id", injectAdmin(2, 2), hdl.Level1ReviewHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1", strings.NewReader(`{"action":1}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.POST("/review/level1/:id", injectAdmin(1, 1), hdl.Level1ReviewHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/abc", strings.NewReader(`{"action":1}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("bad_body", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.POST("/review/level1/:id", injectAdmin(1, 1), hdl.Level1ReviewHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1", strings.NewReader(`{`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("service_error", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.POST("/review/level1/:id", injectAdmin(1, 1), hdl.Level1ReviewHandler)

		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(nil, errors.New("record not found"))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1", strings.NewReader(`{"action":1,"comment":"ok"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})
}

// 测试 handler.go 文件中 func (h *ReviewHandler) Level2ReviewHandler(c *gin.Context)
//
// 函数功能：处理二级审核（终审通过/拒绝），仅角色2管理员可操作
//
// 测试场景：
// 1. 审核通过 — 验证状态从 PendingLevel2 变为 Approved
// 2. 未登录 — 验证返回401
// 3. 角色错误 — 角色1调用返回403
func TestLevel2ReviewHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.POST("/review/level2/:id", injectAdmin(2, 2), hdl.Level2ReviewHandler)

		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(getOrder(1, reservationdb.StatusPendingLevel2), nil)
		mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel2, reservationdb.StatusApproved).Return(nil)
		mockRepo.EXPECT().CreateReviewRecord(gomock.Any()).Return(nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level2/1", strings.NewReader(`{"action":1,"comment":"通过"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("not_logged_in", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.POST("/review/level2/:id", hdl.Level2ReviewHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level2/1", strings.NewReader(`{"action":1}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	t.Run("wrong_role", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.POST("/review/level2/:id", injectAdmin(1, 1), hdl.Level2ReviewHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level2/1", strings.NewReader(`{"action":1}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
	})
}

// 测试 handler.go 文件中 func (h *ReviewHandler) SetPasswordHandler(c *gin.Context)
//
// 函数功能：为已通过终审的时段设置门锁动态密码，仅角色1管理员可操作
//
// 测试场景：
// 1. 设置成功 — 验证返回200
// 2. 未登录 — 验证返回401
// 3. 无效订单ID — 验证返回400
// 4. 无效时段ID — 验证返回400
// 5. 错误的请求体 — 验证返回400
func TestSetPasswordHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, mockRepo, _, hdl, r := setupReviewTestHandler(t)
		r.PUT("/review/level1/:id/slots/:slotID/password", injectAdmin(1, 1), hdl.SetPasswordHandler)

		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(getOrder(1, reservationdb.StatusApproved), nil)
		mockRepo.EXPECT().SetSlotPassword(uint(10), "123456").Return(nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/review/level1/1/slots/10/password", strings.NewReader(`{"password":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("not_logged_in", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.PUT("/review/level1/:id/slots/:slotID/password", hdl.SetPasswordHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/review/level1/1/slots/10/password", strings.NewReader(`{"password":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	t.Run("invalid_order_id", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.PUT("/review/level1/:id/slots/:slotID/password", injectAdmin(1, 1), hdl.SetPasswordHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/review/level1/abc/slots/10/password", strings.NewReader(`{"password":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("invalid_slot_id", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.PUT("/review/level1/:id/slots/:slotID/password", injectAdmin(1, 1), hdl.SetPasswordHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/review/level1/1/slots/xyz/password", strings.NewReader(`{"password":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("bad_body", func(t *testing.T) {
		_, _, _, hdl, r := setupReviewTestHandler(t)
		r.PUT("/review/level1/:id/slots/:slotID/password", injectAdmin(1, 1), hdl.SetPasswordHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/review/level1/1/slots/10/password", strings.NewReader(`{`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})
}

// 确保 auth 包被使用
var _ = auth.AdminResp{}
