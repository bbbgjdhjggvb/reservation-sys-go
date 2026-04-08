// internal/reservation/handler_test.go
package reservation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func setupTestHandler(t *testing.T) (*gomock.Controller, *MockReservationRepository, *ReservationHandler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)
	hdl := NewReservationHandler(svc)

	r := gin.New()
	return ctrl, mockRepo, hdl, r
}

func TestReservationHandler_Submit(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.POST("/api/v1/reservation/submit", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.SubmitHandler(c)
	})

	t.Run("提交成功", func(t *testing.T) {
		body := SubmitReq{
			ApplicantName:     "张三",
			AlumniAssociation: "某某校友会",
			Reason:            "举办活动",
			Phone:             "13800138000",
			StartTime:         "2026-03-25 14:00:00",
			EndTime:           "2026-03-25 16:00:00",
		}
		jsonBody, _ := json.Marshal(body)

		mockRepo.EXPECT().
			FindByTimeRange(gomock.Any(), gomock.Any()).
			Return([]*Reservation{}, nil)
		mockRepo.EXPECT().
			Create(gomock.Any()).
			Return(nil)

		req, _ := http.NewRequest("POST", "/api/v1/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, float64(200), resp["code"])
	})

	t.Run("参数错误", func(t *testing.T) {
		body := map[string]string{
			"applicant_name": "张三",
			// 缺少必填字段
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/v1/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestReservationHandler_GetOccupiedSlots(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.GET("/api/v1/reservation/occupied", hdl.GetOccupiedSlots)

	t.Run("获取成功", func(t *testing.T) {
		mockRepo.EXPECT().
			FindByTimeRange(gomock.Any(), gomock.Any()).
			Return([]*Reservation{
				{
					ID:        1,
					StartTime: time.Date(2026, 3, 25, 14, 0, 0, 0, time.Local),
					EndTime:   time.Date(2026, 3, 25, 16, 0, 0, 0, time.Local),
					Status:    StatusApproved,
				},
			}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/reservation/occupied?date=2026-03-25", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestReservationHandler_GetMyReservations(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.GET("/api/v1/reservation/my", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.GetMyReservations(c)
	})

	t.Run("获取成功", func(t *testing.T) {
		mockRepo.EXPECT().
			FindByOpenID("test_openid_001").
			Return([]*Reservation{
				{ID: 1, OpenID: "test_openid_001"},
			}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/reservation/my", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestReservationHandler_Cancel(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.POST("/api/v1/reservation/cancel/:id", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.Cancel(c)
	})

	t.Run("取消成功", func(t *testing.T) {
		mockRepo.EXPECT().
			FindByID(uint(1)).
			Return(&Reservation{ID: 1, OpenID: "test_openid_001", Status: StatusPending}, nil)
		mockRepo.EXPECT().
			Cancel(uint(1), "test_openid_001").
			Return(nil)

		req, _ := http.NewRequest("POST", "/api/v1/reservation/cancel/1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("无效ID", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/v1/reservation/cancel/invalid", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
