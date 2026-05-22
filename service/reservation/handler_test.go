// internal/reservation/handler_test.go
package reservation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTestHandler(t *testing.T) (*gomock.Controller, *MockReservationRepository, *ReservationHandler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)
	hdl := NewReservationHandler(svc)

	r := gin.New()
	return ctrl, mockRepo, hdl, r
}

// ========== SubmitHandler（支持多时段） ==========

func TestReservationHandler_Submit(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.POST("/api/reservation/reservation/submit", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.SubmitHandler(c)
	})

	t.Run("提交单个时段成功", func(t *testing.T) {
		body := SubmitReq{
			ApplicantName:     "张三",
			AlumniAssociation: "计算机与软件学院校友会",
			Year:              2020,
			Major:             "计算机科学",
			Reason:            "举办活动",
			Phone:             "13800138000",
			Slots: []TimeSlotReq{
				{StartTime: "2026-03-25 14:00:00", EndTime: "2026-03-25 16:00:00"},
			},
		}
		jsonBody, _ := json.Marshal(body)

		mockRepo.EXPECT().CreateOrderWithLock(gomock.Any(), gomock.Any()).
			Return(nil).Do(func(order *reservationdb.ReservationOrder, slots []reservationdb.ReservationSlot) { order.ID = 100 })
		mockRepo.EXPECT().FindOrderByID(uint(100)).
			Return(&reservationdb.ReservationOrder{
				ID: 100, OrderNo: "R1234567890", OpenID: "test_openid_001",
				Status: reservationdb.StatusPending, TotalSlots: 1,
				Slots: []reservationdb.ReservationSlot{{ID: 1}},
			}, nil)

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
	})

	t.Run("提交多个时段(3个)成功", func(t *testing.T) {
		body := SubmitReq{
			ApplicantName:     "张三",
			AlumniAssociation: "计算机与软件学院校友会",
			Year:              2020,
			Major:             "计算机科学",
			Reason:            "多时段会议",
			Phone:             "13800138000",
			Slots: []TimeSlotReq{
				{StartTime: "2026-03-25 08:00:00", EndTime: "2026-03-25 10:00:00"},
				{StartTime: "2026-03-25 13:00:00", EndTime: "2026-03-25 15:00:00"},
				{StartTime: "2026-03-26 08:00:00", EndTime: "2026-03-26 10:00:00"},
			},
		}
		jsonBody, _ := json.Marshal(body)

		mockRepo.EXPECT().CreateOrderWithLock(gomock.Any(), gomock.Any()).
			Return(nil).Do(func(order *reservationdb.ReservationOrder, slots []reservationdb.ReservationSlot) { order.ID = 200 })
		mockRepo.EXPECT().FindOrderByID(uint(200)).
			Return(&reservationdb.ReservationOrder{
				ID: 200, OrderNo: "R1234567890", OpenID: "test_openid_001",
				Status: reservationdb.StatusPending, TotalSlots: 3,
				Slots: []reservationdb.ReservationSlot{{ID: 1}, {ID: 2}, {ID: 3}},
			}, nil)

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("参数错误：缺少必填字段", func(t *testing.T) {
		body := map[string]string{"applicant_name": "张三"} // 缺少slots等必填
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("参数错误：时间段格式不合法", func(t *testing.T) {
		body := SubmitReq{
			ApplicantName: "张三",
			Reason:        "测试",
			Phone:         "13800138000",
			Slots: []TimeSlotReq{
				{StartTime: "invalid-time", EndTime: "2026-03-25 16:00:00"},
			},
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("参数错误：超过4个时段", func(t *testing.T) {
		slots := make([]TimeSlotReq, 5)
		for i := range slots {
			slots[i] = TimeSlotReq{
				StartTime: "2026-03-25 08:00:00",
				EndTime:   "2026-03-25 10:00:00",
			}
		}
		body := SubmitReq{
			ApplicantName:     "张三",
			AlumniAssociation: "计算机与软件学院校友会",
			Year:              2020,
			Major:             "CS",
			Reason:            "测试",
			Phone:             "13800138000",
			Slots:             slots,
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "填写有误")
	})

	t.Run("未授权（无openid）", func(t *testing.T) {
		r2 := gin.New()
		hdl2 := NewReservationHandler(NewReservationService(mockRepo))
		r2.POST("/api/reservation/reservation/submit", hdl2.SubmitHandler)

		body := SubmitReq{
			ApplicantName: "张三", AlumniAssociation: "某校友会", Year: 2020,
			Major: "CS", Reason: "测试", Phone: "13800138000",
			Slots: []TimeSlotReq{
				{StartTime: "2026-03-25 14:00:00", EndTime: "2026-03-25 16:00:00"},
			},
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ========== GetOccupiedSlots ==========

func TestReservationHandler_GetOccupiedSlots(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.GET("/api/reservation/reservation/occupied", hdl.GetOccupiedSlots)

	t.Run("获取成功", func(t *testing.T) {
		mockRepo.EXPECT().
			FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.ReservationSlot{
				{
					ID:        1,
					StartTime: time.Date(2026, 3, 25, 14, 0, 0, 0, time.Local),
					EndTime:   time.Date(2026, 3, 25, 16, 0, 0, 0, time.Local),
					Status:    reservationdb.StatusApproved,
				},
			}, nil)

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/occupied?date=2026-03-25", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ========== GetMyReservations ==========

func TestReservationHandler_GetMyReservations(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.GET("/api/reservation/reservation/my", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.GetMyReservations(c)
	})

	t.Run("获取成功", func(t *testing.T) {
		mockRepo.EXPECT().FindOrdersByOpenID("test_openid_001").Return([]*reservationdb.ReservationOrder{
			{
				ID: 1, OpenID: "test_openid_001", TotalSlots: 2, Status: reservationdb.StatusPending,
				Slots: []reservationdb.ReservationSlot{
					{ID: 10}, {ID: 11},
				},
			},
		}, nil)

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/my", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
	})
}

// ========== Cancel ==========

func TestReservationHandler_Cancel(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.DELETE("/api/reservation/reservation/:id", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.Cancel(c)
	})

	t.Run("取消成功", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).
			Return(&reservationdb.ReservationOrder{ID: 1, OpenID: "test_openid_001", Status: reservationdb.StatusPending}, nil)
		mockRepo.EXPECT().CancelOrder(uint(1), "test_openid_001").Return(nil)

		req, _ := http.NewRequest("DELETE", "/api/reservation/reservation/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
		assert.Contains(t, resp.Msg, "取消成功")
	})

	t.Run("无效ID(非数字)", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/reservation/reservation/invalid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "无效的预约ID")
	})

	t.Run("订单不存在", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(999)).
			Return(nil, gorm.ErrRecordNotFound)

		req, _ := http.NewRequest("DELETE", "/api/reservation/reservation/999", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "预约不存在")
	})

	t.Run("无权操作(非本人)", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(2)).
			Return(&reservationdb.ReservationOrder{ID: 2, OpenID: "other_openid", Status: reservationdb.StatusPending}, nil)

		req, _ := http.NewRequest("DELETE", "/api/reservation/reservation/2", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "无权操作")
	})

	t.Run("未授权（无openid）", func(t *testing.T) {
		r2 := gin.New()
		r2.DELETE("/api/reservation/reservation/:id", hdl.Cancel)

		req, _ := http.NewRequest("DELETE", "/api/reservation/reservation/1", nil)
		w := httptest.NewRecorder()
		r2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ========== 未授权访问测试（对应 shell 脚本的 test_unauthorized_access）==========

func TestReservationHandler_UnauthorizedAccess(t *testing.T) {
	ctrl, _, hdl, _ := setupTestHandler(t)
	defer ctrl.Finish()

	t.Run("GetMyReservations 无Token返回401", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/reservation/reservation/my", hdl.GetMyReservations)

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/my", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "未授权")
	})

	t.Run("Cancel 无Token返回401", func(t *testing.T) {
		r := gin.New()
		r.DELETE("/api/reservation/reservation/:id", hdl.Cancel)

		req, _ := http.NewRequest("DELETE", "/api/reservation/reservation/1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ========== 参数错误测试（对应 shell 脚本的 test_invalid_params）==========

func TestReservationHandler_ParamValidation(t *testing.T) {
	ctrl, _, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.POST("/api/reservation/reservation/submit", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.SubmitHandler(c)
	})

	t.Run("缺少必填字段(applicant_name)", func(t *testing.T) {
		body := map[string]any{"applicant_name": "缺字段测试"} // 缺少slots等必填
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("空请求体", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "填写有误")
	})

	t.Run("时间段结束时间不晚于开始时间", func(t *testing.T) {
		body := SubmitReq{
			ApplicantName: "张三", AlumniAssociation: "某校友会", Year: 2020,
			Major: "CS", Reason: "测试", Phone: "13800138000",
			Slots: []TimeSlotReq{
				{StartTime: "2026-06-01 16:00:00", EndTime: "2026-06-01 14:00:00"},
			},
		}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "结束时间必须晚于开始时间")
	})
}

// ========== SubmitHandler GetOrderByID 回退路径 ==========

func TestReservationHandler_Submit_GetOrderByIDFallback(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.POST("/api/reservation/reservation/submit", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.SubmitHandler(c)
	})

	t.Run("Submit成功但GetOrderByID失败时返回部分数据", func(t *testing.T) {
		body := SubmitReq{
			ApplicantName:     "张三",
			AlumniAssociation: "计算机与软件学院校友会",
			Year:              2020,
			Major:             "CS",
			Reason:            "测试",
			Phone:             "13800138000",
			Slots:             []TimeSlotReq{{StartTime: "2026-03-25 14:00:00", EndTime: "2026-03-25 16:00:00"}},
		}
		jsonBody, _ := json.Marshal(body)

		mockRepo.EXPECT().CreateOrderWithLock(gomock.Any(), gomock.Any()).
			Return(nil).Do(func(order *reservationdb.ReservationOrder, slots []reservationdb.ReservationSlot) { order.ID = 100 })
		mockRepo.EXPECT().FindOrderByID(uint(100)).Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
		assert.Contains(t, resp.Msg, "提交成功")
	})
}

// ========== GetMyReservations 数据库错误 ==========

func TestReservationHandler_GetMyReservations_DBError(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.GET("/api/reservation/reservation/my", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.GetMyReservations(c)
	})

	t.Run("数据库查询失败返回500", func(t *testing.T) {
		mockRepo.EXPECT().FindOrdersByOpenID("test_openid_001").Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/my", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// ========== GetOccupiedSlots 错误路径 ==========

func TestReservationHandler_GetOccupiedSlots_Error(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.GET("/api/reservation/reservation/occupied", hdl.GetOccupiedSlots)

	t.Run("日期格式错误返回400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/reservation/reservation/occupied?date=invalid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("数据库查询失败返回400", func(t *testing.T) {
		mockRepo.EXPECT().
			FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/occupied?date=2026-03-25", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ========== 业务逻辑错误测试 ==========

func TestReservationHandler_BusinessErrors(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.POST("/api/reservation/reservation/submit", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.SubmitHandler(c)
	})

	t.Run("提交时时间段已被占用(原子检测)", func(t *testing.T) {
		body := SubmitReq{
			ApplicantName: "张三", AlumniAssociation: "某校友会", Year: 2020,
			Major: "CS", Reason: "测试占用检测", Phone: "13800138000",
			Slots: []TimeSlotReq{
				{StartTime: "2026-07-01 09:00:00", EndTime: "2026-07-01 11:00:00"},
			},
		}
		jsonBody, _ := json.Marshal(body)

		mockRepo.EXPECT().
			CreateOrderWithLock(gomock.Any(), gomock.Any()).
			Return(fmt.Errorf("第1个时间段已被预约"))

		req, _ := http.NewRequest("POST", "/api/reservation/reservation/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Contains(t, resp.Msg, "已被预约")
	})

	t.Run("查询已占用时段-缺省date参数使用今天", func(t *testing.T) {
		r2 := gin.New()
		r2.GET("/api/reservation/reservation/occupied", hdl.GetOccupiedSlots)

		mockRepo.EXPECT().
			FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.ReservationSlot{}, nil)

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/occupied", nil)
		w := httptest.NewRecorder()
		r2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("查询我的预约-空列表", func(t *testing.T) {
		r2 := gin.New()
		r2.GET("/api/reservation/reservation/my", func(c *gin.Context) {
			c.Set("openid", "empty_user")
			hdl.GetMyReservations(c)
		})

		mockRepo.EXPECT().FindOrdersByOpenID("empty_user").Return([]*reservationdb.ReservationOrder{}, nil)

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/my", nil)
		w := httptest.NewRecorder()
		r2.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
		// 空列表应为 [] 或 null
		dataRaw, _ := json.Marshal(resp.Data)
		assert.Contains(t, string(dataRaw), "[")
	})
}
