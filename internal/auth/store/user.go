// Package store 提供 auth 模块的数据访问层。
package store

import (
	"reservation-sys/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRepository 定义用户数据访问接口
type UserRepository interface {
	Upsert(user *model.User) error
	UpdateStatus(openid string, status int) error
	GetByOpenID(openid string) (*model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Upsert 存在则更新，不存在则创建（基于 OpenID 唯一索引的原子操作）。
// SQL: INSERT INTO users (...) ON DUPLICATE KEY UPDATE ...
func (r *userRepository) Upsert(user *model.User) error {
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "openid"}},
		DoUpdates: clause.AssignmentColumns([]string{"nickname", "status", "last_login", "updated_at"}),
	}).Create(user).Error
}

// UpdateStatus 仅更新用户的关注状态。
// SQL: UPDATE users SET status = ? WHERE openid = ?
func (r *userRepository) UpdateStatus(openid string, status int) error {
	return r.db.Model(&model.User{}).
		Where("openid = ?", openid).
		Update("status", status).Error
}

// GetByOpenID 根据 OpenID 查找用户。
// SQL: SELECT * FROM users WHERE openid = ? LIMIT 1
func (r *userRepository) GetByOpenID(openid string) (*model.User, error) {
	var user model.User
	err := r.db.Where("openid = ?", openid).First(&user).Error
	return &user, err
}
