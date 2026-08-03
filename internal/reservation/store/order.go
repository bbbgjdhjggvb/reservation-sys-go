// Package store 提供 reservation 模块的数据访问层。
package store

import (
	"fmt"

	"reservation-sys/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrderStore 订单数据访问
type OrderStore struct {
	db *gorm.DB
}

// NewOrderStore 创建订单仓库实例
func NewOrderStore(db *gorm.DB) *OrderStore {
	return &OrderStore{db: db}
}

// CreateOrderWithLock 在事务内创建订单并锁定时段，防止并发双重预约。
func (s *OrderStore) CreateOrderWithLock(order *model.ReservationOrder, slots []model.ReservationSlot) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i, slot := range slots {
			var count int64
			err := tx.Model(&model.ReservationSlot{}).
				Where("status IN ?", []int{model.StatusPendingLevel1, model.StatusPendingLevel2, model.StatusApproved}).
				Where("start_time < ? AND end_time > ?", slot.EndTime, slot.StartTime).
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Count(&count).Error
			if err != nil {
				return fmt.Errorf("检测第%d个时段冲突失败: %w", i+1, err)
			}
			if count > 0 {
				return fmt.Errorf("第%d个时间段已被预约", i+1)
			}
		}

		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("创建订单失败: %w", err)
		}

		for i := range slots {
			slots[i].OrderID = order.ID
		}
		if len(slots) > 0 {
			if err := tx.Create(&slots).Error; err != nil {
				return fmt.Errorf("创建时段失败: %w", err)
			}
		}

		return nil
	})
}

// FindOrderByID 根据订单ID查询订单详情（预加载时段）。
func (s *OrderStore) FindOrderByID(id uint) (*model.ReservationOrder, error) {
	var order model.ReservationOrder
	err := s.db.Preload("Slots").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// FindOrdersByOpenID 根据用户 openid 查询其所有预约订单。
func (s *OrderStore) FindOrdersByOpenID(openid string) ([]*model.ReservationOrder, error) {
	var orders []*model.ReservationOrder
	err := s.db.Preload("Slots").
		Where("open_id = ?", openid).
		Order("created_at desc").
		Find(&orders).Error
	return orders, err
}

// CancelOrder 取消订单（事务内同时更新订单和时段状态）。
func (s *OrderStore) CancelOrder(orderID uint, openid string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.ReservationOrder{}).
			Where("id = ? AND open_id = ? AND status IN ?",
				orderID, openid, []int{model.StatusPendingLevel1, model.StatusPendingLevel2, model.StatusApproved}).
			Update("status", model.StatusCancelled)

		if result.Error != nil {
			return fmt.Errorf("更新订单状态失败: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		slotResult := tx.Model(&model.ReservationSlot{}).
			Where("order_id = ? AND status IN ?", orderID,
				[]int{model.StatusPendingLevel1, model.StatusPendingLevel2, model.StatusApproved}).
			Update("status", model.StatusCancelled)

		if slotResult.Error != nil {
			return fmt.Errorf("更新时段状态失败: %w", slotResult.Error)
		}

		return nil
	})
}
