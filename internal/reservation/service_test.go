// internal/reservation/service_test.go
package reservation

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
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
			slots: []ParsedSlot{testSlot1},
			req:   req,
			mockSetup: func() {
				mockRepo.EXPECT().
					FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
					Return([]ReservationSlot{}, nil).Times(1)
				mockRepo.EXPECT().
					CreateOrder(gomock.Any(), gomock.Any()).
					Return(nil).Do(func(order *ReservationOrder, slots []ReservationSlot) {
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
					FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
					Return([]ReservationSlot{}, nil).Times(3)
				mockRepo.EXPECT().
					CreateOrder(gomock.Any(), gomock.Any()).
					Return(nil).Do(func(order *ReservationOrder, slots []ReservationSlot) {
						order.ID = 101
						assert.Equal(t, 3, len(slots))
						assert.Equal(t, 3, order.TotalSlots)
					})
			},
			wantErr: false,
		},
		{
			name:   "第1个时段已被占用",
			openid: "test_openid_001",
			slots: []ParsedSlot{testSlot1},
			req:   req,
			mockSetup: func() {
				mockRepo.EXPECT().
					FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
					Return([]ReservationSlot{{ID: 1}}, nil)
			},
			wantErr: true,
			errMsg:  "第1个时间段已被预约",
		},
		{
			name:   "第2个时段已被占用(多时段场景)",
			openid: "test_openid_001",
			slots: []ParsedSlot{testSlot1, testSlot2},
			req:   req,
			mockSetup: func() {
				mockRepo.EXPECT().
					FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
					Return([]ReservationSlot{}, nil). // 第1个空闲
					Times(1)
				mockRepo.EXPECT().
					FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
					Return([]ReservationSlot{{ID: 5}}, nil) // 第2个被占用
			},
			wantErr: true,
			errMsg:  "第2个时间段已被预约",
		},
		{
			name:   "查询占用时段失败(DB错误)",
			openid: "test_openid_001",
			slots: []ParsedSlot{testSlot1},
			req:   req,
			mockSetup: func() {
				mockRepo.EXPECT().
					FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("db error"))
			},
			wantErr: true,
			errMsg:  "查询第1个时段占用失败",
		},
		{
			name:   "创建订单失败(DB错误)",
			openid: "test_openid_001",
			slots: []ParsedSlot{testSlot1},
			req:   req,
			mockSetup: func() {
				mockRepo.EXPECT().
					FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
					Return([]ReservationSlot{}, nil)
				mockRepo.EXPECT().
					CreateOrder(gomock.Any(), gomock.Any()).
					Return(errors.New("insert failed"))
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
				// 合并后只有1个时段，所以 FindSlotsByTimeRange 只调用1次
				mockRepo.EXPECT().
					FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
					Return([]ReservationSlot{}, nil).Times(1)
				mockRepo.EXPECT().
					CreateOrder(gomock.Any(), gomock.Any()).
					Return(nil).Do(func(order *ReservationOrder, slots []ReservationSlot) {
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
					FindSlotsByTimeRange(gomock.Any(), gomock.Any()).
					Return([]ReservationSlot{}, nil).Times(2)
				mockRepo.EXPECT().
					CreateOrder(gomock.Any(), gomock.Any()).
					Return(nil).Do(func(order *ReservationOrder, slots []ReservationSlot) {
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

func TestReservationService_GetMyReservations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	t.Run("获取用户订单列表成功", func(t *testing.T) {
		expectedOrders := []*ReservationOrder{
			{
				ID:        1,
				OpenID:    "test_openid_001",
				Status:    StatusPending,
				TotalSlots: 2,
				Slots: []ReservationSlot{
					{ID: 10, StartTime: testSlot1.StartTime, EndTime: testSlot1.EndTime},
					{ID: 11, StartTime: testSlot2.StartTime, EndTime: testSlot2.EndTime},
				},
			},
			{
				ID: 2, OpenID: "test_openid_001", Status: StatusApproved, TotalSlots: 1,
				Slots: []ReservationSlot{{ID: 20}},
			},
		}
		mockRepo.EXPECT().FindByOpenID("test_openid_001").Return(expectedOrders, nil)

		result, err := svc.GetMyReservations("test_openid_001")
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Len(t, result[0].Slots, 2)
	})

	t.Run("数据库错误", func(t *testing.T) {
		mockRepo.EXPECT().FindByOpenID("test_openid_001").Return(nil, errors.New("db error"))

		result, err := svc.GetMyReservations("test_openid_001")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// ========== Cancel ==========

func TestReservationService_Cancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockReservationRepository(ctrl)
	svc := NewReservationService(mockRepo)

	t.Run("取消成功", func(t *testing.T) {
		mockRepo.EXPECT().FindByOrderID(uint(1)).Return(&ReservationOrder{
			ID: 1, OpenID: "test_001", Status: StatusPending,
		}, nil)
		mockRepo.EXPECT().CancelOrder(uint(1), "test_001").Return(nil)

		err := svc.Cancel(1, "test_001")
		assert.NoError(t, err)
	})

	t.Run("订单不存在", func(t *testing.T) {
		mockRepo.EXPECT().FindByOrderID(uint(999)).
			Return(nil, gorm.ErrRecordNotFound)

		err := svc.Cancel(999, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不存在")
	})

	t.Run("无权操作他人订单", func(t *testing.T) {
		mockRepo.EXPECT().FindByOrderID(uint(1)).
			Return(&ReservationOrder{ID: 1, OpenID: "other_user", Status: StatusPending}, nil)

		err := svc.Cancel(1, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "无权操作")
	})

	t.Run("已完成的订单无法取消", func(t *testing.T) {
		mockRepo.EXPECT().FindByOrderID(uint(1)).
			Return(&ReservationOrder{ID: 1, OpenID: "test_001", Status: StatusCompleted}, nil)

		err := svc.Cancel(1, "test_001")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "当前状态无法取消")
	})
}

// ========== mergeContinuousSlots 连续时段合并 ==========

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
				testSlot1,                           // 08:00-10:00
				testSlot1Continuous,                 // 10:00-12:00
			},
			expected: []ParsedSlot{
				{StartTime: mustTime("2026-03-25 08:00:00"), EndTime: mustTime("2026-03-25 12:00:00")},
			},
		},
		{
			name: "三个时段前两个连续，合并为两个",
			input: []ParsedSlot{
				testSlot1,                           // 08:00-10:00
				testSlot1Continuous,                 // 10:00-12:00
				testSlot2,                           // 13:00-15:00 (不连续)
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
				testSlot1Continuous,                 // 10:00-12:00
				testSlot1,                           // 08:00-10:00 (更早)
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
