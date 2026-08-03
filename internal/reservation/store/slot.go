package store

import (
	"fmt"
	"time"

	"reservation-sys/internal/model"

	"gorm.io/gorm"
)

// SlotStore 时段数据访问
type SlotStore struct {
	db *gorm.DB
}

// NewSlotStore 创建时段仓库实例
func NewSlotStore(db *gorm.DB) *SlotStore {
	return &SlotStore{db: db}
}

// SlotWithOpenID 带订单归属信息的时段查询结果
type SlotWithOpenID struct {
	model.ReservationSlot
	OpenID string
}

// FindSlotsByTimeRange 查询指定时间范围内有交集的已占用时段。
func (s *SlotStore) FindSlotsByTimeRange(start, end time.Time) ([]model.ReservationSlot, error) {
	var slots []model.ReservationSlot
	err := s.db.Where("status IN ?", []int{model.StatusPendingLevel1, model.StatusPendingLevel2, model.StatusApproved}).
		Where("start_time < ? AND end_time > ?", end, start).
		Find(&slots).Error
	return slots, err
}

// FindSlotsWithOpenIDByTimeRange 查询时段并附带 open_id，供上层标记 is_mine。
func (s *SlotStore) FindSlotsWithOpenIDByTimeRange(start, end time.Time) ([]SlotWithOpenID, error) {
	var results []SlotWithOpenID

	err := s.db.Table("reservation_slots").
		Select("reservation_slots.*, reservation_orders.open_id").
		Joins("LEFT JOIN reservation_orders ON reservation_orders.id = reservation_slots.order_id").
		Where("reservation_slots.status IN ?",
			[]int{model.StatusPendingLevel1, model.StatusPendingLevel2, model.StatusApproved}).
		Where("reservation_slots.start_time < ? AND reservation_slots.end_time > ?", end, start).
		Find(&results).Error

	return results, err
}

// SetSlotPassword 设置已通过时段的门锁密码。
func (s *SlotStore) SetSlotPassword(slotID uint, password string) error {
	result := s.db.Model(&model.ReservationSlot{}).
		Where("id = ? AND status = ?", slotID, model.StatusApproved).
		Update("password", password)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("时段不存在或状态不允许设置密码")
	}
	return nil
}
