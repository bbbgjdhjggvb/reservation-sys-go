package store

import (
	"reservation-sys/internal/model"

	"gorm.io/gorm"
)

// AdminRepository 定义管理员数据访问接口
type AdminRepository interface {
	FindAdminByUsername(username string) (*model.Admin, error)
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
// SQL: SELECT * FROM admins WHERE username = ? AND status = 1 LIMIT 1
func (r *adminRepository) FindAdminByUsername(username string) (*model.Admin, error) {
	var admin model.Admin
	err := r.db.Where("username = ? AND status = 1", username).First(&admin).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

// UpdateAdminLoginTime 更新管理员最后登录时间。
// SQL: UPDATE admins SET last_login_at = NOW() WHERE id = ?
func (r *adminRepository) UpdateAdminLoginTime(adminID uint) error {
	return r.db.Model(&model.Admin{}).Where("id = ?", adminID).
		Update("last_login_at", gorm.Expr("NOW()")).Error
}
