package store

import (
	"reservation-sys/internal/model"

	"gorm.io/gorm"
)

// RecordStore 审核记录数据访问
type RecordStore struct {
	db *gorm.DB
}

// NewRecordStore 创建审核记录仓库实例
func NewRecordStore(db *gorm.DB) *RecordStore {
	return &RecordStore{db: db}
}

// CreateReviewRecord 创建审核记录。
// SQL: INSERT INTO review_records (order_id, reviewer_id, reviewer_role, action, comment) VALUES (?, ?, ?, ?, ?)
func (s *RecordStore) CreateReviewRecord(record *model.ReviewRecord) error {
	return s.db.Create(record).Error
}

// FindReviewRecordsByOrderID 根据订单ID查询审核记录（按创建时间正序）。
// SQL: SELECT * FROM review_records WHERE order_id = ? ORDER BY created_at ASC
func (s *RecordStore) FindReviewRecordsByOrderID(orderID uint) ([]model.ReviewRecord, error) {
	var records []model.ReviewRecord
	err := s.db.Where("order_id = ?", orderID).
		Order("created_at asc").
		Find(&records).Error
	return records, err
}
