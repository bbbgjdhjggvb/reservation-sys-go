// internal/reservation/service_test.go
package reservation

import (
	"errors"
	"fmt"
	"testing"
	"time"

	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	layout = "2006-01-02 15:04:05"

	testSlot1 = ParsedSlot{
		StartTime: mustTime("2026-03-25 08:00:00"),
		EndTime:   mustTime("2026-03-25 10:00:00"),
	}
	testSlot2 = ParsedSlot{
		StartTime: mustTime("2026-03-25 13:00:00"),
		EndTime:   mustTime("2026-03-25 15:00:00"),
	}
	// 连续时段：紧接在 testSlot1 之后 (10:00-12:00)
	testSlot1Continuous = ParsedSlot{
		StartTime: mustTime("2026-03-25 10:00:00"),
		EndTime:   mustTime("2026-03-25 12:00:00"),
	}
)

func mustTime(s string) time.Time {
	t, _ := time.ParseInLocation(layout, s, time.Local)
	return t
}

// ========== Submit（批量提交） ==========
//
// 测试 service.go 文件中 func (s *ReservationService) Submit(openid string, slots []ParsedSlot, req *SubmitReq) (*SubmitResp, error)
//
// 函数功能：提交预约申请，支持多时段并自动合并连续时段
//
// 测试场景：
// 1. 正常提交单个时段 — 验证订单创建成功、ID回填
// 2. 正常提交多个时段(3个) — 验证 TotalSlots 等于 3
// 3. 第1个时段已被占用(原子检测)
// 4. 第2个时段已被占用(多时段场景)
// 5. 创建订单失败(DB错误)
// 6. 两个连续时段自动合并为1个存储
// 7. 三个时段其中两个连续，合并为2个存储

func TestReservationService_Submit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	req := &SubmitReq{
		ApplicantName:     "张三",
		AlumniAssociation: "计算机与软件学院校友会",
		Year:              2020,
		Major:             "计算机科学",
		Reason:            "举办活动",
		Phone:             "13800138000",
		AttendeeCount:     10,
	}

	tests := []struct {
		name      string
		openid    string
		slots     []ParsedSlot
		req       *SubmitReq
		mockSetup func()
		wantErr   bool
		errMsg    string
	}{
		{
			name:   "正常提交单个时段",
			openid: "test_openid_001",
			slots:  []ParsedSlot{testSlot1},
			req:    req,
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateOrderWithLock(gomock.Any(), gomock.Any()).
					Return(nil).Do(func(order *reservationdb.ReservationOrder, slots []reservationdb.ReservationSlot) {
					order.ID = 100
					assert.Equal(t, 1, len(slots))
				})
			},
			wantErr: false,
		},
		{
			name:   "正常提交多个时段(3个)",
			openid: "test_openid_002",
			slots: []ParsedSlot{
				testSlot1,
				testSlot2,
				{
					StartTime: mustTime("2026-03-26 08:00:00"),
					EndTime:   mustTime("2026-03-26 10:00:00"),
				},
			},
			req: req,
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateOrderWithLock(gomock.Any(), gomock.Any()).
					Return(nil).Do(func(order *reservationdb.ReservationOrder, slots []reservationdb.ReservationSlot) {
					order.ID = 101
					assert.Equal(t, 3, len(slots))
					assert.Equal(t, 3, order.TotalSlots)
				})
			},
			wantErr: false,
		},
		{
			name:   "第1个时段已被占用(原子检测)",
			openid: "test_openid_001",
			slots:  []ParsedSlot{testSlot1},
			req:    req,
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateOrderWithLock(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("第1个时间段已被预约"))
			},
			wantErr: true,
			errMsg:  "第1个时间段已被预约",
		},
		{
			name:   "第2个时段已被占用(多时段场景，原子检测)",
			openid: "test_openid_001",
			slots:  []ParsedSlot{testSlot1, testSlot2},
			req:    req,
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateOrderWithLock(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("第2个时间段已被预约"))
			},
			wantErr: true,
			errMsg:  "第2个时间段已被预约",
		},
		{
			name:   "创建订单失败(DB错误)",
			openid: "test_openid_001",
			slots:  []ParsedSlot{testSlot1},
			req:    req,
			mockSetup: func() {
				mockRepo.EXPECT().
					CreateOrderWithLock(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("创建预约失败: insert failed"))
			},
			wantErr: true,
			errMsg:  "创建预约失败",
		},
		{
			name:   "两个连续时段自动合并为1个存储",
			openid: "test_openid_003",
			// testSlot1(08:00-10:00) + testSlot1Continuous(10:00-12:00) → 合并为 08:00-12:00
			slots: []ParsedSlot{testSlot1, testSlot1Continuous},
			req:   req,
			mockSetup: func() {
				// 合并后只有1个时段，由 CreateOrderWithLock 内部处理冲突检测
				mockRepo.EXPECT().
					CreateOrderWithLock(gomock.Any(), gomock.Any()).
					Return(nil).Do(func(order *reservationdb.ReservationOrder, slots []reservationdb.ReservationSlot) {
					order.ID = 102
					assert.Equal(t, 1, len(slots), "合并后应只有1个slot记录")
					assert.Equal(t, mustTime("2026-03-25 08:00:00"), slots[0].StartTime)
					assert.Equal(t, mustTime("2026-03-25 12:00:00"), slots[0].EndTime)
					assert.Equal(t, 1, order.TotalSlots, "TotalSlots应为合并后的数量")
				})
			},
			wantErr: false,
		},
		{
			name:   "三个时段其中两个连续，合并为2个存储",
			openid: "test_openid_004",
			// slot1(08:00-10:00) + slot1Cont(10:00-12:00) 合并; slot2(13:00-15:00) 独立
			slots: []ParsedSlot{testSlot1, testSlot1Continuous, testSlot2},
			req:   req,
			mockSetup: func() {
				// 合并后有2个时段
				mockRepo.EXPECT().
					CreateOrderWithLock(gomock.Any(), gomock.Any()).
					Return(nil).Do(func(order *reservationdb.ReservationOrder, slots []reservationdb.ReservationSlot) {
					order.ID = 103
					assert.Equal(t, 2, len(slots), "合并后应只有2个slot记录")
					assert.Equal(t, 2, order.TotalSlots)
				})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			res, err := svc.Submit(tt.openid, tt.slots, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, tt.openid, res.OpenID)
				assert.NotEmpty(t, res.OrderNo)
				// TotalSlots 为合并后的时段数，不一定等于原始 slots 数量
				assert.Greater(t, res.TotalSlots, 0)
			}
		})
	}
}

// ========== GetMyReservations ==========
//
// 测试 service.go 文件中 func (s *ReservationService) GetMyReservations(openid string) ([]*MyReservationResp, error)
//
// 函数功能：查询当前用户的预约列表
//
// 测试场景：
// 1. 获取用户订单列表成功 — 验证返回 2 条，Slots 正确加载
// 2. 数据库错误 — 验证返回 error 和 nil

func TestReservationService_GetMyReservations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	t.Run("获取用户订单列表成功", func(t *testing.T) {
		expectedOrders := []*reservationdb.ReservationOrder{
			{
				ID:         1,
				OpenID:     "test_openid_001",
				Status:     reservationdb.StatusPendingLevel1,
				TotalSlots: 2,
				Slots: []reservationdb.ReservationSlot{
					{ID: 10, StartTime: testSlot1.StartTime, EndTime: testSlot1.EndTime},
					{ID: 11, StartTime: testSlot2.StartTime, EndTime: testSlot2.EndTime},
				},
			},
			{
				ID: 2, OpenID: "test_openid_001", Status: reservationdb.StatusApproved, TotalSlots: 1,
				Slots: []reservationdb.ReservationSlot{{ID: 20}},
			},
		}
		mockRepo.EXPECT().FindOrdersByOpenID("test_openid_001").Return(expectedOrders, nil)

		result, err := svc.GetMyReservations("test_openid_001")
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Len(t, result[0].Slots, 2)
	})

	t.Run("数据库错误", func(t *testing.T) {
		mockRepo.EXPECT().FindOrdersByOpenID("test_openid_001").Return(nil, errors.New("db error"))

		result, err := svc.GetMyReservations("test_openid_001")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// ========== Cancel ==========
//
// 测试 service.go 文件中 func (s *ReservationService) Cancel(orderID uint, openid string) error
//
// 函数功能：取消预约订单，校验权限和状态
//
// 测试场景：
// 1. 取消成功 — 验证不返回错误
// 2. 订单不存在 — 验证返回"不存在"错误
// 3. 无权操作他人订单 — 验证返回"无权操作"错误
// 4. 已完成的订单无法取消 — 验证返回"当前状态无法取消"错误

func TestReservationService_Cancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	t.Run("取消成功", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(&reservationdb.ReservationOrder{
			ID: 1, OpenID: "test_001", Status: reservationdb.StatusPendingLevel1,
		}, nil)
		mockRepo.EXPECT().CancelOrder(uint(1), "test_001").Return(nil)

		err := svc.Cancel(1, "test_001")
		assert.NoError(t, err)
	})

	t.Run("订单不存在", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(999)).
			Return(nil, gorm.ErrRecordNotFound)

		err := svc.Cancel(999, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不存在")
	})

	t.Run("无权操作他人订单", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).
			Return(&reservationdb.ReservationOrder{ID: 1, OpenID: "other_user", Status: reservationdb.StatusPendingLevel1}, nil)

		err := svc.Cancel(1, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无权操作")
	})

	t.Run("已完成的订单无法取消", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).
			Return(&reservationdb.ReservationOrder{ID: 1, OpenID: "test_001", Status: reservationdb.StatusCompleted}, nil)

		err := svc.Cancel(1, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "当前状态无法取消")
	})
}

// ========== mergeContinuousSlots 连续时段合并 ==========
//
// 测试 service.go 文件中 func mergeContinuousSlots(slots []ParsedSlot) []ParsedSlot
//
// 函数功能：将连续的时段合并为一个长时段，支持乱序输入自动排序
//
// 测试场景：
// 1. 空切片 — 返回空切片
// 2. 单个时段不合并
// 3. 两个非连续时段不合并
// 4. 两个连续时段合并为一个
// 5. 三个时段前两个连续，合并为两个
// 6. 三个连续时段合并为一个
// 7. 乱序输入后自动排序再合并
// 8. 跨天的不连续时段不合并

func TestMergeContinuousSlots(t *testing.T) {
	tests := []struct {
		name     string
		input    []ParsedSlot
		expected []ParsedSlot
	}{
		{
			name:     "空切片",
			input:    []ParsedSlot{},
			expected: []ParsedSlot{},
		},
		{
			name:     "单个时段不合并",
			input:    []ParsedSlot{testSlot1},
			expected: []ParsedSlot{testSlot1},
		},
		{
			name:     "两个非连续时段不合并",
			input:    []ParsedSlot{testSlot1, testSlot2},
			expected: []ParsedSlot{testSlot1, testSlot2},
		},
		{
			name: "两个连续时段合并为一个",
			input: []ParsedSlot{
				testSlot1,           // 08:00-10:00
				testSlot1Continuous, // 10:00-12:00
			},
			expected: []ParsedSlot{
				{StartTime: mustTime("2026-03-25 08:00:00"), EndTime: mustTime("2026-03-25 12:00:00")},
			},
		},
		{
			name: "三个时段前两个连续，合并为两个",
			input: []ParsedSlot{
				testSlot1,           // 08:00-10:00
				testSlot1Continuous, // 10:00-12:00
				testSlot2,           // 13:00-15:00 (不连续)
			},
			expected: []ParsedSlot{
				{StartTime: mustTime("2026-03-25 08:00:00"), EndTime: mustTime("2026-03-25 12:00:00")},
				testSlot2,
			},
		},
		{
			name: "三个连续时段合并为一个",
			input: []ParsedSlot{
				{StartTime: mustTime("2026-03-25 09:00:00"), EndTime: mustTime("2026-03-25 10:00:00")},
				{StartTime: mustTime("2026-03-25 10:00:00"), EndTime: mustTime("2026-03-25 11:00:00")},
				{StartTime: mustTime("2026-03-25 11:00:00"), EndTime: mustTime("2026-03-25 12:00:00")},
			},
			expected: []ParsedSlot{
				{StartTime: mustTime("2026-03-25 09:00:00"), EndTime: mustTime("2026-03-25 12:00:00")},
			},
		},
		{
			name: "乱序输入后自动排序再合并",
			input: []ParsedSlot{
				testSlot1Continuous, // 10:00-12:00
				testSlot1,           // 08:00-10:00 (更早)
			},
			expected: []ParsedSlot{
				{StartTime: mustTime("2026-03-25 08:00:00"), EndTime: mustTime("2026-03-25 12:00:00")},
			},
		},
		{
			name: "跨天的不连续时段不合并",
			input: []ParsedSlot{
				{StartTime: mustTime("2026-03-25 22:00:00"), EndTime: mustTime("2026-03-26 02:00:00")},
				{StartTime: mustTime("2026-03-26 02:00:00"), EndTime: mustTime("2026-03-26 06:00:00")},
			},
			expected: []ParsedSlot{
				{StartTime: mustTime("2026-03-25 22:00:00"), EndTime: mustTime("2026-03-26 02:00:00")},
				{StartTime: mustTime("2026-03-26 02:00:00"), EndTime: mustTime("2026-03-26 06:00:00")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeContinuousSlots(tt.input)
			assert.Equal(t, len(tt.expected), len(result), "数量不一致")
			for i := range result {
				assert.Equal(t, tt.expected[i].StartTime, result[i].StartTime,
					"第%d个时段StartTime不一致", i+1)
				assert.Equal(t, tt.expected[i].EndTime, result[i].EndTime,
					"第%d个时段EndTime不一致", i+1)
			}
		})
	}
}

// ========== GetOrderByID ==========
//
// 测试 service.go 文件中 func (s *ReservationService) GetOrderByID(orderID uint) (*ReservationOrder, error)
//
// 函数功能：根据订单 ID 查询订单详情
//
// 测试场景：
// 1. 查询成功 — 验证返回订单对象正确
// 2. 订单不存在 — 验证返回 gorm.ErrRecordNotFound 错误

func TestReservationService_GetOrderByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	t.Run("success", func(t *testing.T) {
		expected := &reservationdb.ReservationOrder{ID: 1, OrderNo: "R001"}
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(expected, nil)

		order, err := svc.GetOrderByID(1)
		assert.NoError(t, err)
		assert.Equal(t, expected, order)
	})

	t.Run("not_found", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(999)).Return(nil, gorm.ErrRecordNotFound)

		order, err := svc.GetOrderByID(999)
		assert.Error(t, err)
		assert.Nil(t, order)
	})
}

// ========== generateOrderNo ==========
//
// 测试 service.go 文件中 func generateOrderNo() string
//
// 函数功能：生成格式为 R + 14位时间 + 4位hex小写的订单号
//
// 测试场景：
// 1. 验证订单号长度为 19
// 2. 验证以 R 开头
// 3. 连续两次生成不应相同
func TestGenerateOrderNo(t *testing.T) {
	// 验证格式：R + 14位时间 YYYYMMDDHHmmss + 4位hex小写
	orderNo := generateOrderNo()
	assert.Len(t, orderNo, 1+14+4, "订单号长度应为19")
	assert.Equal(t, 'R', rune(orderNo[0]), "应以R开头")

	// 多次生成不应相同
	orderNo2 := generateOrderNo()
	assert.NotEqual(t, orderNo, orderNo2, "连续生成的订单号不应相同")
}

// ========== GetOccupiedSlots 错误路径 ==========
//
// 测试 service.go 文件中 func (s *ReservationService) GetOccupiedSlots(date string) ([]*OccupiedSlotResp, error)
//
// 函数功能：验证日期格式错误和数据库查询错误时的处理
//
// 测试场景：
// 1. 日期格式错误 — 验证返回"日期格式错误"error
// 2. 数据库查询错误 — 验证返回 error

func TestReservationService_GetOccupiedSlots_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	t.Run("invalid_date_format", func(t *testing.T) {
		result, err := svc.GetOccupiedSlots("not-a-date", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "日期格式错误")
		assert.Nil(t, result)
	})

	t.Run("db_error", func(t *testing.T) {
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("db connection error"))

		result, err := svc.GetOccupiedSlots("2026-03-25", "")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// ========== Cancel DB 错误路径 ==========
//
// 测试 service.go 文件中 Cancel 在非 RecordNotFound 类数据库错误时的处理
//
// 函数功能：验证数据库查询返回非 RecordNotFound 错误时正确传递错误
//
// 测试场景：
// 1. FindOrderByID返回非RecordNotFound错误 — 验证错误信息包含"connection timeout"
func TestReservationService_Cancel_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	t.Run("FindOrderByID返回非RecordNotFound错误", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).
			Return(nil, errors.New("connection timeout"))

		err := svc.Cancel(1, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection timeout")
	})
}

// ========== GetOccupiedSlots（返回格式验证） ==========
//
// 测试 service.go 文件中 GetOccupiedSlots 的返回格式
//
// 函数功能：验证返回的时间格式、状态映射和时段时间值的正确性
//
// 测试场景：
// 1. 时间格式为 string 类型且包含空格分隔符
// 2. 待审核时段状态映射为 "pending"
// 3. 已通过时段状态映射为 "approved"
// 4. 时段时间值精确匹配

func TestReservationService_GetOccupiedSlots_Format(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	testDate := "2026-03-25"
	mockSlots := []reservationdb.SlotWithOpenID{
		{ReservationSlot: reservationdb.ReservationSlot{ID: 1, StartTime: mustTime("2026-03-25 08:00:00"), EndTime: mustTime("2026-03-25 10:00:00"), Status: reservationdb.StatusPendingLevel1}, OpenID: "user_a"},
		{ReservationSlot: reservationdb.ReservationSlot{ID: 2, StartTime: mustTime("2026-03-25 13:00:00"), EndTime: mustTime("2026-03-25 15:00:00"), Status: reservationdb.StatusApproved}, OpenID: "user_b"},
	}

	mockRepo.EXPECT().
		FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
		Return(mockSlots, nil)

	// 以 user_a 身份查询，验证 is_mine 标记
	result, err := svc.GetOccupiedSlots(testDate, "user_a")
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// 验证返回的是 string 类型时间，不是 time.Time 的 RFC3339 格式
	t.Run("time_format", func(t *testing.T) {
		for i, slot := range result {
			assert.IsType(t, "", slot.StartTime, "slot[%d].StartTime 应为 string", i)
			assert.IsType(t, "", slot.EndTime, "slot[%d].EndTime 应为 string", i)
			// 应为 "2006-01-02 15:04" 格式（带空格分隔日期和时间）
			assert.Contains(t, slot.StartTime, " ", "slot[%d].StartTIme 应包含空格分隔符", i)
			assert.Contains(t, slot.EndTime, " ", "slot[%d].EndTIme 应包含空格分隔符", i)
		}
	})

	t.Run("pending_status", func(t *testing.T) {
		assert.Equal(t, "pending", result[0].Status, "待审核时段应返回 'pending'")
		assert.True(t, result[0].IsMine, "user_a 的 pending 时段 is_mine 应为 true")
	})

	t.Run("approved_status", func(t *testing.T) {
		assert.Equal(t, "approved", result[1].Status, "已通过时段应返回 'approved'")
		assert.False(t, result[1].IsMine, "user_b 的 approved 时段 is_mine 应为 false")
	})

	t.Run("exact_time_values", func(t *testing.T) {
		assert.Equal(t, "2026-03-25 08:00", result[0].StartTime)
		assert.Equal(t, "2026-03-25 10:00", result[0].EndTime)
		assert.Equal(t, "2026-03-25 13:00", result[1].StartTime)
		assert.Equal(t, "2026-03-25 15:00", result[1].EndTime)
	})
}

// ========== GetOccupiedSlots is_mine 归属标记 ==========
//
// 测试 service.go 文件中
// func (s *ReservationService) GetOccupiedSlots(date string, openid string) ([]TimeSlotResp, error)
//
// 函数功能：获取指定日期的已占用时段，并通过比对 openid 标记 is_mine 字段，
// 区分"自己的预约"和"他人的预约"。
//
// 测试场景：
// 1. 自己的 pending 时段 is_mine 为 true
//    - 目的：验证当 slot.OpenID == 当前用户 openid 时，pending 状态的 is_mine 标记正确
//    - 预期：Status="pending", IsMine=true
// 2. 自己的 approved 时段 is_mine 为 true
//    - 目的：验证已通过状态下的 is_mine 标记正确
//    - 预期：Status="approved", IsMine=true
// 3. 他人的时段 is_mine 为 false
//    - 目的：验证当 slot.OpenID != 当前用户 openid 时，is_mine 为 false
//    - 预期：Status="pending" 或 "approved", IsMine=false
// 4. openid 为空字符串时所有 is_mine 为 false
//    - 目的：防御性测试 — 未登录或中间件异常时 openid 为空，不应误标记
//    - 预期：所有 IsMine=false，即使某些 slot 的 OpenID 也是空字符串
// 5. 混合场景 — 自己的 + 他人的 + 已通过的
//    - 目的：验证在同一天多个订单混合时 is_mine 标记互不干扰
//    - 预期：3 条记录，is_mine 分别为 true/false/true
func TestReservationService_GetOccupiedSlots_IsMine(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	testDate := "2026-03-25"
	currentUser := "current_user_openid"

	t.Run("自己的pending时段_is_mine为true", func(t *testing.T) {
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 1, StartTime: mustTime("2026-03-25 08:00:00"),
						EndTime: mustTime("2026-03-25 10:00:00"),
						Status:  reservationdb.StatusPendingLevel1,
					},
					OpenID: currentUser, // ← 与当前用户匹配
				},
			}, nil)

		result, err := svc.GetOccupiedSlots(testDate, currentUser)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "pending", result[0].Status)
		assert.True(t, result[0].IsMine, "自己的 pending 时段 is_mine 应为 true")
	})

	t.Run("自己的approved时段_is_mine为true", func(t *testing.T) {
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 2, StartTime: mustTime("2026-03-25 13:00:00"),
						EndTime: mustTime("2026-03-25 15:00:00"),
						Status:  reservationdb.StatusApproved,
					},
					OpenID: currentUser, // ← 与当前用户匹配
				},
			}, nil)

		result, err := svc.GetOccupiedSlots(testDate, currentUser)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "approved", result[0].Status)
		assert.True(t, result[0].IsMine, "自己的 approved 时段 is_mine 应为 true")
	})

	t.Run("他人的时段_is_mine为false", func(t *testing.T) {
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 3, StartTime: mustTime("2026-03-25 10:00:00"),
						EndTime: mustTime("2026-03-25 12:00:00"),
						Status:  reservationdb.StatusPendingLevel1,
					},
					OpenID: "other_user_openid", // ← 与当前用户不匹配
				},
			}, nil)

		result, err := svc.GetOccupiedSlots(testDate, currentUser)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "pending", result[0].Status)
		assert.False(t, result[0].IsMine, "他人的时段 is_mine 应为 false")
	})

	t.Run("openid为空字符串时所有is_mine为false", func(t *testing.T) {
		// 防御性测试：未登录用户或中间件异常未注入 openid 时，
		// 传入空字符串，即使 slot 的 OpenID 也是空字符串（异常数据），
		// 也不应该标记 is_mine=true（openid != "" 保护）
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 4, StartTime: mustTime("2026-03-25 08:00:00"),
						EndTime: mustTime("2026-03-25 10:00:00"),
						Status:  reservationdb.StatusPendingLevel1,
					},
					OpenID: "some_user",
				},
			}, nil)

		result, err := svc.GetOccupiedSlots(testDate, "")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.False(t, result[0].IsMine,
			"openid 为空时 is_mine 必须为 false，防止误标记")
	})

	t.Run("混合场景_自己的加他人的加已通过的", func(t *testing.T) {
		// 模拟真实场景：同一天有多个不同用户的订单
		mockRepo.EXPECT().
			FindSlotsWithOpenIDByTimeRange(gomock.Any(), gomock.Any()).
			Return([]reservationdb.SlotWithOpenID{
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 10, StartTime: mustTime("2026-03-25 08:00:00"),
						EndTime: mustTime("2026-03-25 10:00:00"),
						Status:  reservationdb.StatusPendingLevel1,
					},
					OpenID: currentUser, // 自己的 pending
				},
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 11, StartTime: mustTime("2026-03-25 10:00:00"),
						EndTime: mustTime("2026-03-25 12:00:00"),
						Status:  reservationdb.StatusPendingLevel1,
					},
					OpenID: "other_user", // 他人的 pending
				},
				{
					ReservationSlot: reservationdb.ReservationSlot{
						ID: 12, StartTime: mustTime("2026-03-25 13:00:00"),
						EndTime: mustTime("2026-03-25 15:00:00"),
						Status:  reservationdb.StatusApproved,
					},
					OpenID: currentUser, // 自己的 approved
				},
			}, nil)

		result, err := svc.GetOccupiedSlots(testDate, currentUser)
		require.NoError(t, err)
		assert.Len(t, result, 3)

		// 自己的 pending → is_mine=true, status=pending
		assert.Equal(t, "pending", result[0].Status)
		assert.True(t, result[0].IsMine, "slot[0] 是 current_user 的 pending，is_mine 应为 true")

		// 他人的 pending → is_mine=false, status=pending
		assert.Equal(t, "pending", result[1].Status)
		assert.False(t, result[1].IsMine, "slot[1] 是 other_user 的 pending，is_mine 应为 false")

		// 自己的 approved → is_mine=true, status=approved
		assert.Equal(t, "approved", result[2].Status)
		assert.True(t, result[2].IsMine, "slot[2] 是 current_user 的 approved，is_mine 应为 true")
	})
}
