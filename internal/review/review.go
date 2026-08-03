package review

import (
	"context"
	"fmt"
	"log"
	"time"

	"reservation-sys/internal/event"
	"reservation-sys/internal/model"
	"reservation-sys/internal/review/store"
)

// ========== 跨模块接口（消费方定义） ==========

// Notifier 定义 review 模块需要的微信通知能力。
// auth 模块的 Notifier 类型隐式实现此接口。
type Notifier interface {
	SendApproval(order *model.ReservationOrder) error
	SendRejection(order *model.ReservationOrder, reason string) error
}

// SlotStore 定义 review 模块需要的时段操作能力。
// reservation/store.SlotStore 隐式实现此接口。
type SlotStore interface {
	SetSlotPassword(slotID uint, password string) error
}

// ========== Service ==========

// Service 审核业务服务
type Service struct {
	orders   *store.OrderStore
	records  *store.RecordStore
	slots    SlotStore
	notifier Notifier
	bus      *event.Bus
}

// NewService 创建审核服务实例
func NewService(
	orders *store.OrderStore,
	records *store.RecordStore,
	slots SlotStore,
	notifier Notifier,
	bus *event.Bus,
) *Service {
	return &Service{
		orders:   orders,
		records:  records,
		slots:    slots,
		notifier: notifier,
		bus:      bus,
	}
}

// Level1Review 一级管理员审核操作。
func (s *Service) Level1Review(adminID uint, orderID uint, req *ReviewActionReq) error {
	order, err := s.orders.FindOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("订单不存在")
	}

	if order.Status != model.StatusPendingLevel1 {
		return fmt.Errorf("当前订单状态不允许一级审核")
	}

	var targetStatus int
	if req.Action == 1 {
		targetStatus = model.StatusPendingLevel2
	} else {
		targetStatus = model.StatusRejectedLevel1
	}

	if err := s.orders.UpdateOrderStatus(orderID, model.StatusPendingLevel1, targetStatus); err != nil {
		return fmt.Errorf("审核操作失败: %v", err)
	}

	record := &model.ReviewRecord{
		OrderID:      orderID,
		ReviewerID:   adminID,
		ReviewerRole: model.RoleLevel1,
		Action:       req.Action,
		Comment:      req.Comment,
	}
	if err := s.records.CreateReviewRecord(record); err != nil {
		log.Printf("[error][review] 创建一级审核记录失败: %v", err)
	}

	s.bus.Publish(event.OrderEvent{
		Type:      event.OrderReviewed,
		OrderID:   orderID,
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// Level2Review 二级管理员审核操作。
func (s *Service) Level2Review(adminID uint, orderID uint, req *ReviewActionReq) error {
	order, err := s.orders.FindOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("订单不存在")
	}

	if order.Status != model.StatusPendingLevel2 {
		return fmt.Errorf("当前订单状态不允许二级审核（需先经一级审核通过）")
	}

	var targetStatus int
	if req.Action == 1 {
		targetStatus = model.StatusApproved
	} else {
		targetStatus = model.StatusRejectedLevel2
	}

	if err := s.orders.UpdateOrderStatus(orderID, model.StatusPendingLevel2, targetStatus); err != nil {
		return fmt.Errorf("审核操作失败: %v", err)
	}

	record := &model.ReviewRecord{
		OrderID:      orderID,
		ReviewerID:   adminID,
		ReviewerRole: model.RoleLevel2,
		Action:       req.Action,
		Comment:      req.Comment,
	}
	if err := s.records.CreateReviewRecord(record); err != nil {
		log.Printf("[error][review] 创建二级审核记录失败: %v", err)
	}

	s.bus.Publish(event.OrderEvent{
		Type:      event.OrderReviewed,
		OrderID:   orderID,
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// SetPassword 设置门锁密码（仅一级管理员可操作，仅审核通过的订单可设置）。
func (s *Service) SetPassword(adminRole int, orderID uint, slotID uint, password string) error {
	if adminRole != model.RoleLevel1 {
		return fmt.Errorf("仅一级管理员可设置门锁密码")
	}

	order, err := s.orders.FindOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("订单不存在")
	}

	if order.Status != model.StatusApproved {
		return fmt.Errorf("仅审核通过的订单可设置门锁密码")
	}

	if err := s.slots.SetSlotPassword(slotID, password); err != nil {
		return err
	}

	s.bus.Publish(event.OrderEvent{
		Type:      event.SlotUpdated,
		OrderID:   orderID,
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// GetOrderDetail 获取订单详情（含审核记录）。
func (s *Service) GetOrderDetail(orderID uint) (*model.ReservationOrder, []model.ReviewRecord, error) {
	order, err := s.orders.FindOrderByID(orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("订单不存在")
	}

	records, err := s.records.FindReviewRecordsByOrderID(orderID)
	if err != nil {
		log.Printf("[warning][review] 查询审核记录失败: %v", err)
		records = []model.ReviewRecord{}
	}

	return order, records, nil
}

// GetOrdersByStatuses 按多状态分页查询订单列表。
func (s *Service) GetOrdersByStatuses(statuses []int, page, pageSize int) ([]*model.ReservationOrder, int64, error) {
	return s.orders.ListOrders(statuses, page, pageSize)
}

// GetAllOrders 分页查询所有订单。
func (s *Service) GetAllOrders(page, pageSize int) ([]*model.ReservationOrder, int64, error) {
	return s.orders.ListOrders(nil, page, pageSize)
}

// GetOrderForNotify 获取用于通知的订单信息。
func (s *Service) GetOrderForNotify(orderID uint) (*model.ReservationOrder, error) {
	order, err := s.orders.FindOrderByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("订单不存在")
	}
	return order, nil
}

// SendApprovalNotify 发送审核通过通知（直接调用 Notifier）。
func (s *Service) SendApprovalNotify(ctx context.Context, orderID uint) error {
	order, err := s.orders.FindOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("订单不存在")
	}

	if order.Status != model.StatusApproved {
		return fmt.Errorf("仅审核通过的订单可发送通知")
	}

	hasPassword := false
	for _, slot := range order.Slots {
		if slot.Password != "" {
			hasPassword = true
			break
		}
	}
	if !hasPassword {
		return fmt.Errorf("请先设置门锁密码后再发送通知")
	}

	return s.notifier.SendApproval(order)
}

// SendRejectionNotify 发送审核驳回通知（直接调用 Notifier）。
func (s *Service) SendRejectionNotify(ctx context.Context, orderID uint, reason string) error {
	order, err := s.orders.FindOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("订单不存在")
	}

	if order.Status != model.StatusRejectedLevel1 && order.Status != model.StatusRejectedLevel2 {
		return fmt.Errorf("仅被驳回的订单可发送驳回通知")
	}

	return s.notifier.SendRejection(order, reason)
}
