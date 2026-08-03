// Package store 提供 review 模块的数据访问层。
package store

import (
	"fmt"

	"reservation-sys/internal/model"

	"gorm.io/gorm"
)

// OrderStore 订单查询数据访问（审核模块只读 + 状态更新）
type OrderStore struct {
	db *gorm.DB
}

// NewOrderStore 创建订单仓库实例
func NewOrderStore(db *gorm.DB) *OrderStore {
	return &OrderStore{db: db}
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

// ListOrders 分页查询订单列表（支持按状态筛选，预加载时段）。
func (s *OrderStore) ListOrders(statuses []int, page, pageSize int) ([]*model.ReservationOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var orders []*model.ReservationOrder
	var total int64

	query := s.db.Model(&model.ReservationOrder{})
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	query.Count(&total)

	err := s.db.Preload("Slots").
		Scopes(func(db *gorm.DB) *gorm.DB {
			if len(statuses) > 0 {
				return db.Where("status IN ?", statuses)
			}
			return db
		}).
		Order("created_at desc").
		Offset(offset).Limit(pageSize).
		Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// UpdateOrderStatus 审核更新订单状态（事务：订单+时段状态同步，乐观锁防并发）。
func (s *OrderStore) UpdateOrderStatus(orderID uint, fromStatus, toStatus int) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.ReservationOrder{}).
			Where("id = ? AND status = ?", orderID, fromStatus).
			Update("status", toStatus)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		tx.Model(&model.ReservationSlot{}).
			Where("order_id = ? AND status = ?", orderID, fromStatus).
			Update("status", toStatus)

		return nil
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("订单状态不匹配，无法执行此操作")
		}
		return fmt.Errorf("更新订单状态失败: %w", err)
	}
	return nil
}
