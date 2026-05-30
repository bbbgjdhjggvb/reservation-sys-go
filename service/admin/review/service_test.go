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

// 测试 service.go 文件中 func (s *ReviewService) Level1Review(adminID uint, orderID uint, req *ReviewActionReq) error
//
// 函数功能：执行一级审核（通过/拒绝），使用乐观锁更新订单状态
//
// 测试场景：
// 1. 审核通过 — 状态从 PendingLevel1 变为 PendingLevel2
// 2. 审核拒绝 — 状态从 PendingLevel1 变为 RejectedLevel1
// 3. 订单不存在 — 返回"订单不存在"错误
// 4. 状态不正确 — 返回"不允许一级审核"错误
// 5. 乐观锁失败 — 返回"审核操作失败"错误
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
			req: &ReviewActionReq{Action: 1},
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
				mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusApproved), nil)
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

// 测试 service.go 文件中 func (s *ReviewService) Level2Review(adminID uint, orderID uint, req *ReviewActionReq) error
//
// 函数功能：执行二级审核（终审通过/拒绝），使用乐观锁更新订单状态
//
// 测试场景：
// 1. 审核通过 — 状态从 PendingLevel2 变为 Approved
// 2. 审核拒绝 — 状态从 PendingLevel2 变为 RejectedLevel2
// 3. 订单不存在 — 返回"订单不存在"错误
// 4. 状态不正确 — 返回"不允许二级审核"错误
func TestReviewService_Level2Review(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success_approve", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusPendingLevel2), nil)
		mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel2, reservationdb.StatusApproved).Return(nil)
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

// 测试 service.go 文件中 func (s *ReviewService) SetPassword(role int, orderID uint, slotID uint, password string) error
//
// 函数功能：为已通过终审的时段设置门锁动态密码，仅角色1管理员可操作
//
// 测试场景：
// 1. 设置成功 — 验证不返回错误
// 2. 角色不正确 — 返回"仅一级管理员可设置门锁密码"
// 3. 订单不存在 — 返回"订单不存在"
// 4. 订单状态不正确 — 返回"仅审核通过的订单可设置门锁密码"
// 5. 数据库错误 — 返回 error
func TestReviewService_SetPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusApproved), nil)
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
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(makeTestOrder(1, reservationdb.StatusApproved), nil)
		mockRepo.EXPECT().SetSlotPassword(uint(10), "123456").Return(errors.New("db error"))

		err := svc.SetPassword(constants.RoleLevel1, 1, 10, "123456")
		assert.Error(t, err)
	})
}

// 测试 service.go 文件中 func (s *ReviewService) GetOrderDetail(orderID uint) (*ReservationOrder, []ReviewRecord, error)
//
// 函数功能：查询订单详情（含审核记录）
//
// 测试场景：
// 1. 查询成功 — 验证返回订单和审核记录
// 2. 订单不存在 — 返回"订单不存在"错误
// 3. 审核记录查询失败 — 返回空切片，不报错
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

// 测试 service.go 文件中 func (s *ReviewService) GetOrdersByStatuses(statuses []int, page, pageSize int) ([]*ReservationOrder, int64, error)
//
// 函数功能：按状态分页查询订单，自动修正非法 page 和 pageSize
//
// 测试场景：
// 1. 正常查询 — 验证返回订单列表和总数
// 2. page 为负数自动修正为1
// 3. pageSize 为0自动修正为20
// 4. pageSize 超过上限自动修正为20
func TestReviewService_GetOrdersByStatuses(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success", func(t *testing.T) {
		orders := []*reservationdb.ReservationOrder{makeTestOrder(1, reservationdb.StatusPendingLevel1)}
		mockRepo.EXPECT().ListOrders([]int{reservationdb.StatusPendingLevel1}, 1, 20).Return(orders, int64(1), nil)

		o, total, err := svc.GetOrdersByStatuses([]int{reservationdb.StatusPendingLevel1}, 1, 20)
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

// 测试 service.go 文件中 func (s *ReviewService) GetAllOrders(page, pageSize int) ([]*ReservationOrder, int64, error)
//
// 函数功能：分页查询全部订单（不限状态），自动修正非法参数
//
// 测试场景：
// 1. 正常查询 — 验证返回所有订单和总数
// 2. page 和 pageSize 越界自动修正
func TestReviewService_GetAllOrders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success", func(t *testing.T) {
		orders := []*reservationdb.ReservationOrder{makeTestOrder(1, reservationdb.StatusPendingLevel1), makeTestOrder(2, reservationdb.StatusApproved)}
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

// 测试 service.go 文件中 func (s *ReviewService) GetOrderForNotify(orderID uint) (*ReservationOrder, error)
//
// 函数功能：查询订单用于发送通知，是 GetOrderDetail 的简化版本
//
// 测试场景：
// 1. 查询成功 — 验证返回订单对象
// 2. 订单不存在 — 返回"订单不存在"错误
func TestReviewService_GetOrderForNotify(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockRepository(ctrl)
	svc := NewReviewService(mockRepo)

	t.Run("success", func(t *testing.T) {
		order := makeTestOrder(1, reservationdb.StatusApproved)
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
