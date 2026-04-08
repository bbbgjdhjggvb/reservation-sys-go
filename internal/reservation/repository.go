// internal/reservation/repository.go
package reservation

import (
	"time"

	"gorm.io/gorm"
)

// ReservationRepository 定义数据访问接口，方便微服务拆分时替换实现
type ReservationRepository interface {
	Create(res *Reservation) error
	FindByID(id uint) (*Reservation, error)
	FindByOpenID(openid string) ([]*Reservation, error)
	FindByTimeRange(start, end time.Time) ([]*Reservation, error)
	UpdateStatus(id uint, status int) error
	Cancel(id uint, openid string) error
}

// reservationRepo 实现 ReservationRepository 接口
type reservationRepo struct {
	db *gorm.DB
}

// NewReservationRepository 创建仓库实例
func NewReservationRepository(db *gorm.DB) ReservationRepository {
	return &reservationRepo{db: db}
}

// Create 创建预约记录
func (r *reservationRepo) Create(res *Reservation) error {
	return r.db.Create(res).Error
}

// FindByID 根据ID查询预约
func (r *reservationRepo) FindByID(id uint) (*Reservation, error) {
	var res Reservation
	err := r.db.First(&res, id).Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// FindByOpenID 根据用户openid查询预约列表
func (r *reservationRepo) FindByOpenID(openid string) ([]*Reservation, error) {
	var reservations []*Reservation
	err := r.db.Where("open_id = ?", openid).Order("created_at desc").Find(&reservations).Error
	return reservations, err
}

// FindByTimeRange 查询指定时间范围内的预约（用于检查时间段占用）
func (r *reservationRepo) FindByTimeRange(start, end time.Time) ([]*Reservation, error) {
	var reservations []*Reservation
	// 查询与指定时间段有交集的预约
	err := r.db.Where("status IN ?", []int{StatusPending, StatusApproved}).
		Where("start_time < ? AND end_time > ?", end, start).
		Find(&reservations).Error
	return reservations, err
}

// UpdateStatus 更新预约状态
func (r *reservationRepo) UpdateStatus(id uint, status int) error {
	return r.db.Model(&Reservation{}).Where("id = ?", id).Update("status", status).Error
}

// Cancel 取消预约（仅限本人且状态为待审核或已通过）
func (r *reservationRepo) Cancel(id uint, openid string) error {
	result := r.db.Model(&Reservation{}).
		Where("id = ? AND open_id = ? AND status IN ?", id, openid, []int{StatusPending, StatusApproved}).
		Update("status", StatusCancelled)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
