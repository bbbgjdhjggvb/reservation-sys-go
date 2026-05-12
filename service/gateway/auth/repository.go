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

// Upsert 存在则更新，不存在则创建（原子操作）。
// 基于 OpenID 唯一索引，冲突时自动更新 nickname、status、last_login、updated_at。
//
// SQL: INSERT INTO users (openid, nickname, status, last_login, ...) VALUES (?, ?, ?, ?, ...)
//      ON DUPLICATE KEY UPDATE nickname=VALUES(nickname), status=VALUES(status), last_login=VALUES(last_login), updated_at=VALUES(updated_at);
func (r *userRepository) Upsert(user *User) error {
	// 使用 GORM 的 OnConflict 处理 OpenID 冲突时的自动更新
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "openid"}},
		DoUpdates: clause.AssignmentColumns([]string{"nickname", "status", "last_login", "updated_at"}),
	}).Create(user).Error
}

// UpdateStatus 仅更新用户的关注状态。
//
// SQL: UPDATE users SET status = ? WHERE openid = ?;
func (r *userRepository) UpdateStatus(openid string, status int) error {
	return r.db.Model(&User{}).
		Where("openid = ?", openid).
		Update("status", status).Error
}

// GetByOpenID 根据 OpenID 查找用户。
//
// SQL: SELECT * FROM users WHERE openid = ? LIMIT 1;
func (r *userRepository) GetByOpenID(openid string) (*User, error) {
	var user User
	err := r.db.Where("openid = ?", openid).First(&user).Error
	return &user, err
}

// =============================================
// 管理员数据访问
// =============================================

// AdminRepository 定义管理员数据访问接口
type AdminRepository interface {
	FindAdminByUsername(username string) (*Admin, error)
	UpdateAdminLoginTime(adminID uint) error
}

type adminRepository struct {
	db *gorm.DB
}

// NewAdminRepository 创建管理员仓库实例
func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

// FindAdminByUsername 根据用户名查找正常状态的管理员。
//
// SQL: SELECT * FROM admins WHERE username = ? AND status = 1 LIMIT 1;
func (r *adminRepository) FindAdminByUsername(username string) (*Admin, error) {
	var admin Admin
	err := r.db.Where("username = ? AND status = 1", username).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// UpdateAdminLoginTime 更新管理员最后登录时间。
//
// SQL: UPDATE admins SET last_login_at = NOW() WHERE id = ?;
func (r *adminRepository) UpdateAdminLoginTime(adminID uint) error {
	return r.db.Model(&Admin{}).Where("id = ?", adminID).Update("last_login_at", gorm.Expr("NOW()")).Error
}
