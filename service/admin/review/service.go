package review

import (
	"fmt"
	"log"

	"reservation-sys/pkg/constants"
	reservationdb "reservation-sys/pkg/reservationdb"
)

// ReviewService 审核业务服务。
// 通过 pkg/reservationdb 共享包直接操作 home_res 数据库，
// 不再通过 gRPC 调用 Reservation 服务。
type ReviewService struct {
	repo reservationdb.Repository
}

// NewReviewService 创建审核服务实例。
//
// 参数:
//   - repo: 预约数据库仓库接口（操作 home_res 库）
//
// 返回值:
//   - *ReviewService: 审核服务实例
func NewReviewService(repo reservationdb.Repository) *ReviewService {
	return &ReviewService{repo: repo}
}

// Level1Review 一级管理员审核操作。
//
// 流程:
//  1. 查询订单，校验当前状态为"待一级审核"
//  2. 根据 action 决定目标状态：通过→待二级审核，拒绝→一级驳回
//  3. 调用 UpdateOrderStatus 乐观锁更新（订单+时段状态同步）
//  4. 创建审核记录
//
// 参数:
//   - adminID: 审核人ID
//   - orderID: 订单ID
//   - req: 审核请求（action: 1=通过, 2=拒绝; comment: 审核意见）
//
// 返回值:
//   - error: 订单不存在、状态不允许、审核操作失败时返回错误
func (s *ReviewService) Level1Review(adminID uint, orderID uint, req *ReviewActionReq) error {
	order, err := s.repo.FindOrderByID(orderID)
	if err != nil {
		log.Printf("[service/admin/review/Level1Review]: 订单id %d", orderID)
		return fmt.Errorf("订单不存在")
	}

	if order.Status != reservationdb.StatusPendingLevel1 {
		log.Printf("[service/admin/review/Level1Review]: 订单状态 %d", order.Status)
		return fmt.Errorf("当前订单状态不允许一级审核")
	}

	var targetStatus int
	if req.Action == 1 {
		targetStatus = reservationdb.StatusPendingLevel2
	} else {
		targetStatus = reservationdb.StatusRejectedLevel1
	}

	if err := s.repo.UpdateOrderStatus(orderID, reservationdb.StatusPendingLevel1, targetStatus); err != nil {
		return fmt.Errorf("审核操作失败: %v", err)
	}

	record := &reservationdb.ReviewRecord{
		OrderID:      orderID,
		ReviewerID:   adminID,
		ReviewerRole: constants.RoleLevel1,
		Action:       req.Action,
		Comment:      req.Comment,
	}
	if err := s.repo.CreateReviewRecord(record); err != nil {
		log.Printf("[error] 创建一级审核记录失败: %v", err)
	}

	return nil
}

// Level2Review 二级管理员审核操作。
//
// 流程:
//  1. 查询订单，校验当前状态为"待二级审核"
//  2. 根据 action 决定目标状态：通过→终审通过，拒绝→二级驳回
//  3. 调用 UpdateOrderStatus 乐观锁更新（订单+时段状态同步）
//  4. 创建审核记录
//
// 参数:
//   - adminID: 审核人ID
//   - orderID: 订单ID
//   - req: 审核请求（action: 1=通过, 2=拒绝; comment: 审核意见）
//
// 返回值:
//   - error: 订单不存在、状态不允许、审核操作失败时返回错误
func (s *ReviewService) Level2Review(adminID uint, orderID uint, req *ReviewActionReq) error {
	order, err := s.repo.FindOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("订单不存在")
	}

	if order.Status != reservationdb.StatusPendingLevel2 {
		return fmt.Errorf("当前订单状态不允许二级审核（需先经一级审核通过）")
	}

	var targetStatus int
	if req.Action == 1 {
		targetStatus = reservationdb.StatusApproved
	} else {
		targetStatus = reservationdb.StatusRejectedLevel2
	}

	if err := s.repo.UpdateOrderStatus(orderID, reservationdb.StatusPendingLevel2, targetStatus); err != nil {
		return fmt.Errorf("审核操作失败: %v", err)
	}

	record := &reservationdb.ReviewRecord{
		OrderID:      orderID,
		ReviewerID:   adminID,
		ReviewerRole: constants.RoleLevel2,
		Action:       req.Action,
		Comment:      req.Comment,
	}
	if err := s.repo.CreateReviewRecord(record); err != nil {
		log.Printf("[error] 创建二级审核记录失败: %v", err)
	}

	return nil
}

// SetPassword 设置门锁密码（仅一级管理员可操作，仅审核通过的订单可设置）。
//
// 流程:
//  1. 校验操作人角色为一级管理员
//  2. 查询订单，校验状态为"终审通过"
//  3. 调用 SetSlotPassword 更新指定时段的密码字段
//
// 参数:
//   - adminRole: 当前管理员角色等级
//   - orderID: 订单ID
//   - slotID: 时段ID
//   - password: 门锁密码（明文，最大20字符）
//
// 返回值:
//   - error: 权限不足、订单状态不允许、设置失败时返回错误
func (s *ReviewService) SetPassword(adminRole int, orderID uint, slotID uint, password string) error {
	if adminRole != constants.RoleLevel1 {
		return fmt.Errorf("仅一级管理员可设置门锁密码")
	}

	order, err := s.repo.FindOrderByID(orderID)
	if err != nil {
		return fmt.Errorf("订单不存在")
	}

	if order.Status != reservationdb.StatusApproved {
		return fmt.Errorf("仅审核通过的订单可设置门锁密码")
	}

	return s.repo.SetSlotPassword(slotID, password)
}

// GetOrderDetail 获取订单详情（含审核记录）。
//
// 参数:
//   - orderID: 订单ID
//
// 返回值:
//   - *reservationdb.ReservationOrder: 订单实体（含时段）
//   - []reservationdb.ReviewRecord: 审核记录列表（查询失败时返回空切片而非错误）
//   - error: 订单不存在时返回错误
func (s *ReviewService) GetOrderDetail(orderID uint) (*reservationdb.ReservationOrder, []reservationdb.ReviewRecord, error) {
	order, err := s.repo.FindOrderByID(orderID)
	if err != nil {
		return nil, nil, fmt.Errorf("订单不存在")
	}

	records, err := s.repo.FindReviewRecordsByOrderID(orderID)
	if err != nil {
		log.Printf("[warning] 查询审核记录失败: %v", err)
		records = []reservationdb.ReviewRecord{}
	}

	return order, records, nil
}

// GetOrdersByStatuses 按多状态分页查询订单列表。
//
// 参数:
//   - statuses: 状态筛选列表
//   - page: 页码（从1开始，<1 时自动修正为1）
//   - pageSize: 每页条数（1~50，超出范围自动修正为20）
//
// 返回值:
//   - []*reservationdb.ReservationOrder: 当前页订单列表
//   - int64: 符合条件的总记录数
//   - error: 查询失败时返回错误
func (s *ReviewService) GetOrdersByStatuses(statuses []int, page, pageSize int) ([]*reservationdb.ReservationOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	return s.repo.ListOrders(statuses, page, pageSize)
}

// GetAllOrders 分页查询所有订单（不限状态）。
//
// 参数:
//   - page: 页码（从1开始）
//   - pageSize: 每页条数（1~50）
//
// 返回值:
//   - []*reservationdb.ReservationOrder: 当前页订单列表
//   - int64: 总记录数
//   - error: 查询失败时返回错误
func (s *ReviewService) GetAllOrders(page, pageSize int) ([]*reservationdb.ReservationOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	return s.repo.ListOrders(nil, page, pageSize)
}

// GetOrderForNotify 获取用于通知的订单信息（供 NotifyHandler 使用）。
//
// 参数:
//   - orderID: 订单ID
//
// 返回值:
//   - *reservationdb.ReservationOrder: 订单实体（含时段，含密码）
//   - error: 订单不存在时返回错误
func (s *ReviewService) GetOrderForNotify(orderID uint) (*reservationdb.ReservationOrder, error) {
	order, err := s.repo.FindOrderByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("订单不存在")
	}
	return order, nil
}
