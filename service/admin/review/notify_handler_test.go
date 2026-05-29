package review

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"reservation-sys/pkg/constants"
	"reservation-sys/pkg/jwt"
	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/service/admin/auth"
	pb "reservation-sys/service/gateway/api/gen/notification"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func setupNotifyTestHandler(t *testing.T) (*gomock.Controller, *MockRepository, *MockNotificationServiceClient, *NotifyHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	mockRepo := NewMockRepository(ctrl)
	mockNotify := NewMockNotificationServiceClient(ctrl)
	notifyHdl := NewNotifyHandler(mockNotify, mockRepo)
	r := gin.New()
	return ctrl, mockRepo, mockNotify, notifyHdl, r
}

func injectAdminLevel1() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("admin", &jwt.AdminClaims{AdminID: 1, Username: "admin1", Role: constants.RoleLevel1})
		c.Next()
	}
}

func makeRejectedOrder(status int) *reservationdb.ReservationOrder {
	return &reservationdb.ReservationOrder{
		ID:                1,
		OrderNo:           "R202605010000000001",
		OpenID:            "test_openid",
		ApplicantName:     "张三",
		AlumniAssociation: "计算机与软件学院校友会",
		Status:            status,
		Slots: []reservationdb.ReservationSlot{
			{StartTime: time.Date(2026, 5, 1, 8, 0, 0, 0, time.Local), EndTime: time.Date(2026, 5, 1, 10, 0, 0, 0, time.Local)},
		},
	}
}

func makeApprovedOrderWithPassword() *reservationdb.ReservationOrder {
	return &reservationdb.ReservationOrder{
		ID:                1,
		OrderNo:           "R202605010000000001",
		OpenID:            "test_openid",
		ApplicantName:     "张三",
		AlumniAssociation: "计算机与软件学院校友会",
		Status:            reservationdb.StatusApproved,
		Slots: []reservationdb.ReservationSlot{
			{
				StartTime: time.Date(2026, 5, 1, 8, 0, 0, 0, time.Local),
				EndTime:   time.Date(2026, 5, 1, 10, 0, 0, 0, time.Local),
				Password:  "123456",
				Status:    reservationdb.StatusApproved,
			},
		},
	}
}

func TestNotifyHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, mockRepo, mockNotify, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/notify", injectAdminLevel1(), notifyHdl.NotifyHandler)

		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeApprovedOrderWithPassword(), nil)
		mockNotify.EXPECT().SendApprovalNotification(gomock.Any(), gomock.Any()).Return(&pb.NotificationResp{Message: "ok"}, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/notify", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("not_logged_in", func(t *testing.T) {
		_, _, _, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/notify", notifyHdl.NotifyHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/notify", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	t.Run("wrong_role", func(t *testing.T) {
		_, _, _, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/notify", func(c *gin.Context) {
			c.Set("admin", &jwt.AdminClaims{AdminID: 2, Role: constants.RoleLevel2})
			c.Next()
		}, notifyHdl.NotifyHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/notify", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
	})

	t.Run("invalid_id", func(t *testing.T) {
		_, _, _, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/notify", injectAdminLevel1(), notifyHdl.NotifyHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/abc/notify", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("order_not_found", func(t *testing.T) {
		_, mockRepo, _, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/notify", injectAdminLevel1(), notifyHdl.NotifyHandler)

		mockRepo.EXPECT().FindOrderByID(uint(999)).Return(nil, errors.New("record not found"))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/999/notify", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("wrong_status", func(t *testing.T) {
		_, mockRepo, _, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/notify", injectAdminLevel1(), notifyHdl.NotifyHandler)

		order := makeApprovedOrderWithPassword()
		order.Status = reservationdb.StatusPendingLevel1
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(order, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/notify", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("no_password", func(t *testing.T) {
		_, mockRepo, _, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/notify", injectAdminLevel1(), notifyHdl.NotifyHandler)

		order := makeApprovedOrderWithPassword()
		order.Slots[0].Password = ""
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(order, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/notify", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("grpc_error", func(t *testing.T) {
		_, mockRepo, mockNotify, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/notify", injectAdminLevel1(), notifyHdl.NotifyHandler)

		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeApprovedOrderWithPassword(), nil)
		mockNotify.EXPECT().SendApprovalNotification(gomock.Any(), gomock.Any()).Return(nil, errors.New("grpc error"))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/notify", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 500, w.Code)
	})
}

func TestRejectionNotifyHandler(t *testing.T) {
	t.Run("success_level1_rejected", func(t *testing.T) {
		_, mockRepo, mockNotify, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/reject-notify", injectAdminLevel1(), notifyHdl.RejectionNotifyHandler)

		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeRejectedOrder(reservationdb.StatusRejectedLevel1), nil)
		mockNotify.EXPECT().SendRejectionNotification(gomock.Any(), gomock.Any()).Return(&pb.NotificationResp{Message: "ok"}, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/reject-notify", strings.NewReader(`{"reason":"场地冲突"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("success_level2_rejected", func(t *testing.T) {
		_, mockRepo, mockNotify, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/reject-notify", injectAdminLevel1(), notifyHdl.RejectionNotifyHandler)

		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeRejectedOrder(reservationdb.StatusRejectedLevel2), nil)
		mockNotify.EXPECT().SendRejectionNotification(gomock.Any(), gomock.Any()).Return(&pb.NotificationResp{Message: "ok"}, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/reject-notify", strings.NewReader(`{"reason":"不符合要求"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("not_logged_in", func(t *testing.T) {
		_, _, _, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/reject-notify", notifyHdl.RejectionNotifyHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/reject-notify", strings.NewReader(`{"reason":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	t.Run("order_not_found", func(t *testing.T) {
		_, mockRepo, _, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/reject-notify", injectAdminLevel1(), notifyHdl.RejectionNotifyHandler)

		mockRepo.EXPECT().FindOrderByID(uint(999)).Return(nil, errors.New("record not found"))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/999/reject-notify", strings.NewReader(`{"reason":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("wrong_status", func(t *testing.T) {
		_, mockRepo, _, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/reject-notify", injectAdminLevel1(), notifyHdl.RejectionNotifyHandler)

		order := makeRejectedOrder(reservationdb.StatusApproved)
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(order, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/reject-notify", strings.NewReader(`{"reason":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("grpc_error", func(t *testing.T) {
		_, mockRepo, mockNotify, notifyHdl, r := setupNotifyTestHandler(t)
		r.POST("/review/level1/:id/reject-notify", injectAdminLevel1(), notifyHdl.RejectionNotifyHandler)

		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeRejectedOrder(reservationdb.StatusRejectedLevel1), nil)
		mockNotify.EXPECT().SendRejectionNotification(gomock.Any(), gomock.Any()).Return(nil, errors.New("grpc error"))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/review/level1/1/reject-notify", strings.NewReader(`{"reason":"test"}`))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, 500, w.Code)
	})
}

func TestOrderSlotsToNotify(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		slots := []reservationdb.ReservationSlot{
			{StartTime: time.Date(2026, 5, 1, 8, 0, 0, 0, time.Local), EndTime: time.Date(2026, 5, 1, 10, 0, 0, 0, time.Local), Password: "123"},
			{StartTime: time.Date(2026, 5, 1, 10, 0, 0, 0, time.Local), EndTime: time.Date(2026, 5, 1, 12, 0, 0, 0, time.Local), Password: "456"},
		}
		result := orderSlotsToNotify(slots)
		assert.Len(t, result, 2)
		assert.Equal(t, "2026-05-01 08:00", result[0].StartTime)
		assert.Equal(t, "2026-05-01 10:00", result[0].EndTime)
		assert.Equal(t, "123", result[0].Password)
	})

	t.Run("empty", func(t *testing.T) {
		result := orderSlotsToNotify(nil)
		assert.Empty(t, result)
	})
}

// 确保类型被使用
var _ = json.NewDecoder
var _ = auth.AdminResp{}
var _ = errors.New
