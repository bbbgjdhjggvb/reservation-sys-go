package review

import (
	"errors"
	"testing"
	"time"

	"reservation-sys/pkg/constants"
	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func makeTestOrder(id uint, status int) *reservationdb.ReservationOrder {
	return &reservationdb.ReservationOrder{
		ID:     id,
		Status: status,
	}
}

func TestReviewService_Level1Review(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	tests := []struct {
		name      string
		orderID   uint
		adminID   uint
		req       *ReviewActionReq
		mockSetup func()
		wantErr   bool
		errMsg    string
	}{
		{
			name:    "success_approve",
			orderID: 1, adminID: 1,
			req: &ReviewActionReq{Action: 1, Comment: "通过"},
			mockSetup: func() {
				mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusPendingLevel1), nil)
				mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel1, reservationdb.StatusPendingLevel2).Return(nil)
				mockRepo.EXPECT().CreateReviewRecord(gomock.Any()).Return(nil)
			},
		},
		{
			name:    "success_reject",
			orderID: 1, adminID: 1,
			req: &ReviewActionReq{Action: 2, Comment: "不通过"},
			mockSetup: func() {
				mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusPendingLevel1), nil)
				mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel1, reservationdb.StatusRejectedLevel1).Return(nil)
				mockRepo.EXPECT().CreateReviewRecord(gomock.Any()).Return(nil)
			},
		},
		{
			name:    "order_not_found",
			orderID: 99, adminID: 1,
			req:       &ReviewActionReq{Action: 1},
			mockSetup: func() {
				mockRepo.EXPECT().FindOrderByID(uint(99)).Return(nil, errors.New("record not found"))
			},
			wantErr: true,
			errMsg:  "订单不存在",
		},
		{
			name:    "wrong_status",
			orderID: 1, adminID: 1,
			req: &ReviewActionReq{Action: 1},
			mockSetup: func() {
				mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusApprovedFinal), nil)
			},
			wantErr: true,
			errMsg:  "不允许一级审核",
		},
		{
			name:    "optimistic_lock_fail",
			orderID: 1, adminID: 1,
			req: &ReviewActionReq{Action: 1},
			mockSetup: func() {
				mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusPendingLevel1), nil)
				mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel1, reservationdb.StatusPendingLevel2).
					Return(errors.New("订单状态不匹配，无法执行此操作"))
			},
			wantErr: true,
			errMsg:  "审核操作失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			err := svc.Level1Review(tt.adminID, tt.orderID, tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReviewService_Level2Review(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success_approve", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusPendingLevel2), nil)
		mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel2, reservationdb.StatusApprovedFinal).Return(nil)
		mockRepo.EXPECT().CreateReviewRecord(gomock.Any()).Return(nil)

		err := svc.Level2Review(2, 1, &ReviewActionReq{Action: 1, Comment: "终审通过"})
		assert.NoError(t, err)
	})

	t.Run("success_reject", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusPendingLevel2), nil)
		mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel2, reservationdb.StatusRejectedLevel2).Return(nil)
		mockRepo.EXPECT().CreateReviewRecord(gomock.Any()).Return(nil)

		err := svc.Level2Review(2, 1, &ReviewActionReq{Action: 2, Comment: "驳回"})
		assert.NoError(t, err)
	})

	t.Run("order_not_found", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(99)).Return(nil, errors.New("record not found"))

		err := svc.Level2Review(2, 99, &ReviewActionReq{Action: 1})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "订单不存在")
	})

	t.Run("wrong_status", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusPendingLevel1), nil)

		err := svc.Level2Review(2, 1, &ReviewActionReq{Action: 1})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不允许二级审核")
	})
}

func TestReviewService_SetPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusApprovedFinal), nil)
		mockRepo.EXPECT().SetSlotPassword(uint(10), "123456").Return(nil)

		err := svc.SetPassword(constants.RoleLevel1, 1, 10, "123456")
		assert.NoError(t, err)
	})

	t.Run("not_level1", func(t *testing.T) {
		err := svc.SetPassword(constants.RoleLevel2, 1, 10, "123456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "仅一级管理员可设置门锁密码")
	})

	t.Run("order_not_found", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(99)).Return(nil, errors.New("record not found"))

		err := svc.SetPassword(constants.RoleLevel1, 99, 10, "123456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "订单不存在")
	})

	t.Run("wrong_status", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusPendingLevel1), nil)

		err := svc.SetPassword(constants.RoleLevel1, 1, 10, "123456")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "仅审核通过的订单可设置门锁密码")
	})

	t.Run("repo_error", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusApprovedFinal), nil)
		mockRepo.EXPECT().SetSlotPassword(uint(10), "123456").Return(errors.New("db error"))

		err := svc.SetPassword(constants.RoleLevel1, 1, 10, "123456")
		assert.Error(t, err)
	})
}

func TestReviewService_GetOrderDetail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success", func(t *testing.T) {
		order := makeTestOrder(1, reservationdb.StatusPendingLevel1)
		records := []reservationdb.ReviewRecord{{ID: 1, OrderID: 1, Comment: "审核通过"}}
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(order, nil)
		mockRepo.EXPECT().FindReviewRecordsByOrderID(uint(1)).Return(records, nil)

		o, r, err := svc.GetOrderDetail(1)
		assert.NoError(t, err)
		assert.Equal(t, order, o)
		assert.Len(t, r, 1)
	})

	t.Run("order_not_found", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(99)).Return(nil, errors.New("record not found"))

		_, _, err := svc.GetOrderDetail(99)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "订单不存在")
	})

	t.Run("records_query_error_returns_empty", func(t *testing.T) {
		order := makeTestOrder(1, reservationdb.StatusPendingLevel1)
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(order, nil)
		mockRepo.EXPECT().FindReviewRecordsByOrderID(uint(1)).Return(nil, errors.New("db error"))

		o, r, err := svc.GetOrderDetail(1)
		assert.NoError(t, err)
		assert.Equal(t, order, o)
		assert.Empty(t, r)
	})
}

func TestReviewService_GetOrdersByStatuses(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success", func(t *testing.T) {
		orders := []*reservationdb.ReservationOrder{makeTestOrder(1, 0)}
		mockRepo.EXPECT().ListOrders([]int{0}, 1, 20).Return(orders, int64(1), nil)

		o, total, err := svc.GetOrdersByStatuses([]int{0}, 1, 20)
		assert.NoError(t, err)
		assert.Len(t, o, 1)
		assert.Equal(t, int64(1), total)
	})

	t.Run("page_fixup_negative", func(t *testing.T) {
		mockRepo.EXPECT().ListOrders(gomock.Any(), 1, 20).Return(nil, int64(0), nil)

		_, _, err := svc.GetOrdersByStatuses(nil, -1, 20)
		assert.NoError(t, err)
	})

	t.Run("page_size_fixup_zero", func(t *testing.T) {
		mockRepo.EXPECT().ListOrders(gomock.Any(), 1, 20).Return(nil, int64(0), nil)

		_, _, err := svc.GetOrdersByStatuses(nil, 1, 0)
		assert.NoError(t, err)
	})

	t.Run("page_size_fixup_exceeds_max", func(t *testing.T) {
		mockRepo.EXPECT().ListOrders(gomock.Any(), 1, 20).Return(nil, int64(0), nil)

		_, _, err := svc.GetOrdersByStatuses(nil, 1, 100)
		assert.NoError(t, err)
	})
}

func TestReviewService_GetAllOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success", func(t *testing.T) {
		orders := []*reservationdb.ReservationOrder{makeTestOrder(1, 0), makeTestOrder(2, 1)}
		mockRepo.EXPECT().ListOrders([]int(nil), 1, 20).Return(orders, int64(2), nil)

		o, total, err := svc.GetAllOrders(1, 20)
		assert.NoError(t, err)
		assert.Len(t, o, 2)
		assert.Equal(t, int64(2), total)
	})

	t.Run("page_fixup", func(t *testing.T) {
		mockRepo.EXPECT().ListOrders([]int(nil), 1, 20).Return(nil, int64(0), nil)

		_, _, err := svc.GetAllOrders(0, 200)
		assert.NoError(t, err)
	})
}

func TestReviewService_GetOrderForNotify(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success", func(t *testing.T) {
		order := makeTestOrder(1, reservationdb.StatusApprovedFinal)
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(order, nil)

		o, err := svc.GetOrderForNotify(1)
		assert.NoError(t, err)
		assert.Equal(t, order, o)
	})

	t.Run("not_found", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(99)).Return(nil, errors.New("record not found"))

		_, err := svc.GetOrderForNotify(99)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "订单不存在")
	})
}

// 确保 time 被使用
var _ = time.Now
