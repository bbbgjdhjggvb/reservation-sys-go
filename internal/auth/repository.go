package auth

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRepository 定义用户数据访问接口
type UserRepository interface {
	Upsert(user *User) error
	UpdateStatus(openid string, status int) error
	GetByOpenID(openid string) (*User, error)
}

// userRepository 实现 UserRepository 接口
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库实例
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Upsert 存在则更新，不存在则创建 (原子操作)
func (r *userRepository) Upsert(user *User) error {
	// 使用 GORM 的 OnConflict 处理 OpenID 冲突时的自动更新
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "openid"}},
		DoUpdates: clause.AssignmentColumns([]string{"nickname", "status", "last_login", "updated_at"}),
	}).Create(user).Error
}

// UpdateStatus 仅更新用户的关注状态
func (r *userRepository) UpdateStatus(openid string, status int) error {
	return r.db.Model(&User{}).
		Where("openid = ?", openid).
		Update("status", status).Error
}

// GetByOpenID 根据 OpenID 查找用户
func (r *userRepository) GetByOpenID(openid string) (*User, error) {
	var user User
	err := r.db.Where("openid = ?", openid).First(&user).Error
	return &user, err
}
