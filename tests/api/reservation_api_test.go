// Package api 提供完整的 HTTP API 测试（通过 httptest 模拟请求）。
// 测试从 HTTP 层验证请求/响应格式、状态码和业务逻辑。
package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/pkg/jwt"
	"reservation-sys/service/reservation"
	"reservation-sys/service/reservation/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	jwt.InitUserJWT("api-test-secret-key-do-not-use-in-production", 24)
	jwt.InitAdminJWT("admin-api-test-secret", 24)
	os.Exit(m.Run())
}

// generateTestToken 生成测试用 JWT token。
func generateTestToken(t *testing.T, openid string) string {
	t.Helper()
	token, err := jwt.GenerateUserToken(openid)
	require.NoError(t, err)
	return token
}

// setupReservationAPI 创建完整的 reservation API 测试环境（含中间件）。
func setupReservationAPI(t *testing.T) (*gomock.Controller, *reservation.MockReservationRepository, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	mockRepo := reservation.NewMockReservationRepository(ctrl)
	svc := reservation.NewReservationService(mockRepo)
	hdl := reservation.NewReservationHandler(svc)

	r := gin.New()

	api := r.Group("/api/reservation")
	api.Use(middleware.AuthMiddleware())
	{
		api.POST("/reservation/submit", hdl.SubmitHandler)
		api.GET("/reservation/my", hdl.GetMyReservations)
		api.DELETE("/reservation/:id", hdl.Cancel)
	}
	// GetOccupiedSlots 不需要认证
	r.GET("/api/reservation/reservation/occupied", hdl.GetOccupiedSlots)

	return ctrl, mockRepo, r
}

// ========== 提交预约 API ==========

func TestReservationAPI_Submit(t *testing.T) {
	ctrl, mockRepo, r := setupReservationAPI(t)
	defer ctrl.Finish()

	token := generateTestToken(t, "test_openid_001")

	t.Run("完整提交流程返回200", func(t *testing.T) {
		body := `{
			"applicant_name":"张三",
			"alumni_association":"计算机与软件学院校友会",
			"year":2020,
			"major":"软件工程",
			"reason":"举办技术讲座",
			"phone":"13800138000",
			"slots":[{"start_time":"2026-06-01 08:00:00","end_time":"2026-06-01 10:00:00"}]
		}`

		mockRepo.EXPECT().CreateOrderWithLock(gomock.Any(), gomock.Any()).
			Return(nil).Do(func(order *reservationdb.ReservationOrder, slots []reservationdb.ReservationSlot) {
				order.ID = 1
				order.OrderNo = "R202606010800000001"
			})
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(&reservationdb.ReservationOrder{
			ID: 1, OrderNo: "R202606010800000001", OpenID: "test_openid_001",
			ApplicantName: "张三", Status: reservationdb.StatusPending, TotalSlots: 1,
			Slots: []reservationdb.ReservationSlot{{ID: 10}},
		}, nil)

		req := httptest.NewRequest("POST", "/api/reservation/reservation/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var resp reservation.Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("无Token返回401", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/reservation/reservation/submit", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	t.Run("空body返回400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/reservation/reservation/submit", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("超4个时段返回400", func(t *testing.T) {
		body := `{
			"applicant_name":"张三","alumni_association":"校友会","year":2020,
			"major":"CS","reason":"测试","phone":"13800138000",
			"slots":[
				{"start_time":"2026-06-01 08:00:00","end_time":"2026-06-01 10:00:00"},
				{"start_time":"2026-06-01 10:00:00","end_time":"2026-06-01 12:00:00"},
				{"start_time":"2026-06-01 13:00:00","end_time":"2026-06-01 15:00:00"},
				{"start_time":"2026-06-01 15:00:00","end_time":"2026-06-01 17:00:00"},
				{"start_time":"2026-06-02 08:00:00","end_time":"2026-06-02 10:00:00"}
			]
		}`

		req := httptest.NewRequest("POST", "/api/reservation/reservation/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("时段冲突返回400", func(t *testing.T) {
		body := `{
			"applicant_name":"张三","alumni_association":"校友会","year":2020,
			"major":"CS","reason":"测试","phone":"13800138000",
			"slots":[{"start_time":"2026-06-01 08:00:00","end_time":"2026-06-01 10:00:00"}]
		}`

		mockRepo.EXPECT().CreateOrderWithLock(gomock.Any(), gomock.Any()).
			Return(fmt.Errorf("第1个时间段已被预约"))

		req := httptest.NewRequest("POST", "/api/reservation/reservation/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
		var resp reservation.Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "已被预约")
	})
}

// ========== 查询我的预约 API ==========

func TestReservationAPI_GetMyReservations(t *testing.T) {
	ctrl, mockRepo, r := setupReservationAPI(t)
	defer ctrl.Finish()

	token := generateTestToken(t, "test_openid_001")

	t.Run("成功返回订单列表", func(t *testing.T) {
		mockRepo.EXPECT().FindOrdersByOpenID("test_openid_001").Return([]*reservationdb.ReservationOrder{
			{
				ID: 1, OpenID: "test_openid_001", TotalSlots: 1,
				Status: reservationdb.StatusPending,
				Slots:  []reservationdb.ReservationSlot{{ID: 10}},
			},
		}, nil)

		req := httptest.NewRequest("GET", "/api/reservation/reservation/my", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var resp reservation.Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("无Token返回401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/reservation/reservation/my", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})
}

// ========== 查询已占用时段 API ==========

func TestReservationAPI_GetOccupiedSlots(t *testing.T) {
	ctrl, mockRepo, r := setupReservationAPI(t)
	defer ctrl.Finish()

	t.Run("成功返回已占用时段", func(t *testing.T) {
		mockRepo.EXPECT().
			FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.ReservationSlot{
				{ID: 1, StartTime: time.Date(2026, 5, 1, 8, 0, 0, 0, time.Local), EndTime: time.Date(2026, 5, 1, 10, 0, 0, 0, time.Local), Status: reservationdb.StatusPending},
			}, nil)

		req := httptest.NewRequest("GET", "/api/reservation/reservation/occupied?date=2026-05-01", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("日期格式错误返回400", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/reservation/reservation/occupied?date=invalid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})
}

// ========== 取消预约 API ==========

func TestReservationAPI_Cancel(t *testing.T) {
	ctrl, mockRepo, r := setupReservationAPI(t)
	defer ctrl.Finish()

	token := generateTestToken(t, "test_openid_001")

	t.Run("成功取消", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(&reservationdb.ReservationOrder{
			ID: 1, OpenID: "test_openid_001", Status: reservationdb.StatusPending,
		}, nil)
		mockRepo.EXPECT().CancelOrder(uint(1), "test_openid_001").Return(nil)

		req := httptest.NewRequest("DELETE", "/api/reservation/reservation/1", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var resp reservation.Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "取消成功")
	})

	t.Run("订单不存在返回400", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(999)).Return(nil, gorm.ErrRecordNotFound)

		req := httptest.NewRequest("DELETE", "/api/reservation/reservation/999", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
	})

	t.Run("无Token返回401", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/reservation/reservation/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})
}

// 确保 time 被使用
var _ = time.Now
