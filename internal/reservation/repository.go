// internal/reservation/repository.go
package reservation

import (
	"time"

	"gorm.io/gorm"
)

// ReservationRepository 定义数据访问接口，方便微服务拆分时替换实现
type ReservationRepository interface {
	// --- 订单操作 ---
	CreateOrder(order *ReservationOrder, slots []ReservationSlot) error // 事务：创建订单+时段
	FindByOrderID(id uint) (*ReservationOrder, error)
	FindByOpenID(openid string) ([]*ReservationOrder, error) // 返回订单及其关联的Slots

	// --- 时段冲突检测 ---
	FindSlotsByTimeRange(start, end time.Time) ([]ReservationSlot, error)

	// --- 状态更新 ---
	UpdateSlotStatus(slotID uint, status int) error
	CancelOrder(orderID uint, openid string) error // 取消整个订单（所有时段）

	// --- 兼容旧接口 ---
	FindByID(id uint) (*ReservationOrder, error)
}

// reservationRepo 实现 ReservationRepository 接口
type reservationRepo struct {
	db *gorm.DB
}

// NewReservationRepository 创建仓库实例
func NewReservationRepository(db *gorm.DB) ReservationRepository {
	return &reservationRepo{db: db}
}

// CreateOrder 事务性创建订单和关联的时段记录
func (r *reservationRepo) CreateOrder(order *ReservationOrder, slots []ReservationSlot) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		for i := range slots {
			slots[i].OrderID = order.ID
		}
		if len(slots) > 0 {
			if err := tx.Create(&slots).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindByOrderID 根据订单ID查询（预加载时段）
func (r *reservationRepo) FindByOrderID(id uint) (*ReservationOrder, error) {
	var order ReservationOrder
	err := r.db.Preload("Slots").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// FindByID 向后兼容别名
func (r *reservationRepo) FindByID(id uint) (*ReservationOrder, error) {
	return r.FindByOrderID(id)
}

// FindByOpenID 根据用户openid查询预约列表（预加载时段，按创建时间倒序）
func (r *reservationRepo) FindByOpenID(openid string) ([]*ReservationOrder, error) {
	var orders []*ReservationOrder
	err := r.db.Preload("Slots").
		Where("open_id = ?", openid).
		Order("created_at desc").
		Find(&orders).Error
	return orders, err
}

// FindSlotsByTimeRange 查询指定时间范围内有交集的已占用时段
// 用于检查时间段冲突
func (r *reservationRepo) FindSlotsByTimeRange(start, end time.Time) ([]ReservationSlot, error) {
	var slots []ReservationSlot
	err := r.db.Where("status IN ?", []int{StatusPending, StatusApproved}).
		Where("start_time < ? AND end_time > ?", end, start).
		Find(&slots).Error
	return slots, err
}

// UpdateSlotStatus 更新单个时段的状态
func (r *reservationRepo) UpdateSlotStatus(slotID uint, status int) error {
	return r.db.Model(&ReservationSlot{}).Where("id = ?", slotID).Update("status", status).Error
}

// CancelOrder 取消整个订单（将订单及所有时段状态设为已取消）
func (r *reservationRepo) CancelOrder(orderID uint, openid string) error {
	result := r.db.Model(&ReservationOrder{}).
		Where("id = ? AND open_id = ? AND status IN ?",
			orderID, openid, []int{StatusPending, StatusApproved}).
		Update("status", StatusCancelled)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	// 同时取消该订单下的所有待审核/已通过的时段
	r.db.Model(&ReservationSlot{}).
		Where("order_id = ? AND status IN ?", orderID, []int{StatusPending, StatusApproved}).
		Update("status", StatusCancelled)

	return nil
}
