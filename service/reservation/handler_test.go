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
	"github.com/stretchr/testify/require"
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
//
// 测试 handler.go 文件中 func (h *ReservationHandler) SubmitHandler(c *gin.Context)
//
// 函数功能：处理用户提交预约申请，支持单时段和多时段
//
// 测试场景：
// 1. 提交单个时段成功
// 2. 提交多个时段(3个)成功
// 3. 参数错误：缺少必填字段
// 4. 参数错误：时间段格式不合法
// 5. 参数错误：超过4个时段
// 6. 未授权（无openid）

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
			AttendeeCount:     10,
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
				Status: reservationdb.StatusPendingLevel1, TotalSlots: 1,
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
			AttendeeCount:     5,
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
				Status: reservationdb.StatusPendingLevel1, TotalSlots: 3,
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
			AttendeeCount: 10,
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
			AttendeeCount:     10,
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
			Major: "CS", Reason: "测试", Phone: "13800138000", AttendeeCount: 10,
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
//
// 测试 handler.go 文件中 func (h *ReservationHandler) GetOccupiedSlots(c *gin.Context)
//
// 函数功能：根据日期查询已占用的时间段列表
//
// 测试场景：
// 1. 成功查询已占用时段
//  1. 验证 HTTP 状态码为 200
//  2. 验证调用 FindSlotsByTimeRange 返回的数据正确

func TestReservationHandler_GetOccupiedSlots(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.GET("/api/reservation/reservation/occupied", hdl.GetOccupiedSlots)

	t.Run("获取成功", func(t *testing.T) {
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID:        1,
						StartTime: time.Date(2026, 3, 25, 14, 0, 0, 0, time.Local),
						EndTime:   time.Date(2026, 3, 25, 16, 0, 0, 0, time.Local),
						Status:    reservationdb.StatusApproved,
					},
					OpenID: "other_user",
				},
			}, nil)

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/occupied?date=2026-03-25", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// ========== GetMyReservations ==========
//
// 测试 handler.go 文件中 func (h *ReservationHandler) GetMyReservations(c *gin.Context)
//
// 函数功能：查询当前用户的预约列表
//
// 测试场景：
// 1. 成功获取预约列表
//  1. 验证 HTTP 状态码为 200
//  2. 验证响应 code 为 200

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
				ID: 1, OpenID: "test_openid_001", TotalSlots: 2, Status: reservationdb.StatusPendingLevel1,
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
//
// 测试 handler.go 文件中 func (h *ReservationHandler) Cancel(c *gin.Context)
//
// 函数功能：取消预约订单
//
// 测试场景：
// 1. 取消成功
// 2. 无效ID(非数字)
// 3. 订单不存在
// 4. 无权操作(非本人)
// 5. 未授权（无openid）

func TestReservationHandler_Cancel(t *testing.T) {
	ctrl, mockRepo, hdl, r := setupTestHandler(t)
	defer ctrl.Finish()

	r.DELETE("/api/reservation/reservation/:id", func(c *gin.Context) {
		c.Set("openid", "test_openid_001")
		hdl.Cancel(c)
	})

	t.Run("取消成功", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).
			Return(&reservationdb.ReservationOrder{ID: 1, OpenID: "test_openid_001", Status: reservationdb.StatusPendingLevel1}, nil)
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
			Return(&reservationdb.ReservationOrder{ID: 2, OpenID: "other_openid", Status: reservationdb.StatusPendingLevel1}, nil)

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

// 测试 handler.go 文件中 GetMyReservations / Cancel 在无 Token 时返回 401
//
// 函数功能：验证未授权访问时返回 HTTP 401
//
// 测试场景：
// 1. GetMyReservations 无Token返回401
// 2. Cancel 无Token返回401
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

// 测试 handler.go 文件中 SubmitHandler 的参数校验逻辑
//
// 函数功能：验证各类非法参数输入时返回 HTTP 400 和正确的错误提示
//
// 测试场景：
// 1. 缺少必填字段(applicant_name)
// 2. 空请求体
// 3. 时间段结束时间不晚于开始时间
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
			Major: "CS", Reason: "测试", Phone: "13800138000", AttendeeCount: 10,
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

// 测试 handler.go 文件中 SubmitHandler 在 GetOrderByID 失败时的回退逻辑
//
// 函数功能：提交成功后回查订单失败时仍返回成功（部分数据）
//
// 测试场景：
// 1. Submit成功但GetOrderByID失败时返回部分数据
//  1. 验证 HTTP 状态码为 200
//  2. 验证响应 msg 包含"提交成功"
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
			AttendeeCount:     10,
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

// 测试 handler.go 文件中 GetMyReservations 数据库错误路径
//
// 函数功能：数据库查询失败时返回 HTTP 500
//
// 测试场景：
// 1. 数据库查询失败返回500
//  1. Mock FindOrdersByOpenID 返回 error
//  2. 验证 HTTP 状态码为 500
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

// 测试 handler.go 文件中 GetOccupiedSlots 错误路径
//
// 函数功能：验证日期格式错误和数据库查询失败时返回 HTTP 400
//
// 测试场景：
// 1. 日期格式错误返回400
// 2. 数据库查询失败返回400
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
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db error"))

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/occupied?date=2026-03-25", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ========== 业务逻辑错误测试 ==========

// 测试 handler.go 中 SubmitHandler 和 GetOccupiedSlots 等业务逻辑错误路径
//
// 函数功能：验证时段冲突、缺省日期、空列表等边界场景的处理
//
// 测试场景：
// 1. 提交时时间段已被占用(原子检测)
// 2. 查询已占用时段-缺省date参数使用今天
// 3. 查询我的预约-空列表
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
			Major: "CS", Reason: "测试占用检测", Phone: "13800138000", AttendeeCount: 10,
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
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{}, nil)

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

// ========== GetOccupiedSlots is_mine 归属标记 ==========
//
// 测试 handler.go 文件中
// func (h *ReservationHandler) GetOccupiedSlots(c *gin.Context)
//
// 函数功能：接收 HTTP 请求查询已占用时段，从 gin context 提取 openid 传递给服务层，
// 使服务层能够标记 is_mine 字段。
//
// 测试场景：
// 1. 认证用户查询 — 自己的时段 is_mine=true
//    - 目的：验证从 context 提取 openid 后正确传递到服务层，is_mine 生效
//    - 预期：HTTP 200，响应 data[0].is_mine 为 true
// 2. 认证用户查询 — 他人的时段 is_mine=false
//    - 目的：验证不同用户之间的 is_mine 隔离
//    - 预期：HTTP 200，响应 data[0].is_mine 为 false
// 3. 未认证用户查询 — 所有 is_mine=false
//    - 目的：验证 context 中无 openid 时，所有时段 is_mine 为 false
//    - 预期：HTTP 200，所有 is_mine 为 false
func TestReservationHandler_GetOccupiedSlots_IsMine(t *testing.T) {
	ctrl, mockRepo, hdl, _ := setupTestHandler(t)
	defer ctrl.Finish()

	t.Run("认证用户查询_自己的时段is_mine为true", func(t *testing.T) {
		// 准备：模拟当前用户 "user_me" 在数据库中有 1 个 pending 时段
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 1, StartTime: time.Date(2026, 3, 25, 8, 0, 0, 0, time.Local),
						EndTime: time.Date(2026, 3, 25, 10, 0, 0, 0, time.Local),
						Status:  reservationdb.StatusPendingLevel1,
					},
					OpenID: "user_me", // ← 与上下文中的 openid 一致
				},
			}, nil)

		// 创建带 openid 上下文的路由（模拟 AuthMiddleware 已注入 openid）
		r := gin.New()
		r.GET("/api/reservation/reservation/occupied", func(c *gin.Context) {
			c.Set("openid", "user_me") // ← 注入当前用户 openid
			hdl.GetOccupiedSlots(c)
		})

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/occupied?date=2026-03-25", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// 验证 HTTP 200 且 is_mine 为 true
		assert.Equal(t, http.StatusOK, w.Code)
		var resp Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.Code)

		slots, ok := resp.Data.([]interface{})
		require.True(t, ok)
		assert.Len(t, slots, 1)
		slot := slots[0].(map[string]interface{})
		assert.Equal(t, "pending", slot["status"])
		assert.Equal(t, true, slot["is_mine"],
			"自己的时段 is_mine 应为 true")
	})

	t.Run("认证用户查询_他人的时段is_mine为false", func(t *testing.T) {
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 2, StartTime: time.Date(2026, 3, 25, 10, 0, 0, 0, time.Local),
						EndTime: time.Date(2026, 3, 25, 12, 0, 0, 0, time.Local),
						Status:  reservationdb.StatusApproved,
					},
					OpenID: "other_user", // ← 与上下文中的 openid 不同
				},
			}, nil)

		r := gin.New()
		r.GET("/api/reservation/reservation/occupied", func(c *gin.Context) {
			c.Set("openid", "user_me") // ← 当前用户是 user_me
			hdl.GetOccupiedSlots(c)
		})

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/occupied?date=2026-03-25", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		slots := resp.Data.([]interface{})
		slot := slots[0].(map[string]interface{})
		assert.Equal(t, "approved", slot["status"])
		assert.Equal(t, false, slot["is_mine"],
			"他人的时段 is_mine 应为 false")
	})

	t.Run("未认证用户查询_context无openid_all_is_mine为false", func(t *testing.T) {
		// 目的：即使 AuthMiddleware 未注入 openid（异常情况），
		// handler 应安全降级，不 panic，且所有 is_mine 为 false
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 3, StartTime: time.Date(2026, 3, 25, 13, 0, 0, 0, time.Local),
						EndTime: time.Date(2026, 3, 25, 15, 0, 0, 0, time.Local),
						Status:  reservationdb.StatusApproved,
					},
					OpenID: "some_user", // ← 数据库中属于某个用户
				},
			}, nil)

		// 不设置 openid 上下文（模拟未认证场景）
		r := gin.New()
		r.GET("/api/reservation/reservation/occupied", hdl.GetOccupiedSlots)

		req, _ := http.NewRequest("GET", "/api/reservation/reservation/occupied?date=2026-03-25", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// 验证不会 panic，正常返回 200
		assert.Equal(t, http.StatusOK, w.Code)
		var resp Response
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)

		slots := resp.Data.([]interface{})
		slot := slots[0].(map[string]interface{})
		assert.Equal(t, false, slot["is_mine"],
			"未认证时 is_mine 必须为 false，防止泄露归属信息")
	})
}
