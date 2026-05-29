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
