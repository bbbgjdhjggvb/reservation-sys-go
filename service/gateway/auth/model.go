package auth

import (
	"time"
)

// User 对应数据库中的 users 表
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OpenID    string    `gorm:"column:openid;type:varchar(100);not null;uniqueIndex" json:"openid"` // 微信唯一标识，加唯一索引
	Nickname  string    `gorm:"type:varchar(255)" json:"nickname"`                                  // 昵称
	Status    int       `gorm:"type:tinyint;default:1;comment:'1:正常,0:已取消关注'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastLogin time.Time `json:"last_login"` // 记录最后交互时间
}

// TableName 指定数据库表名，符合 Go 惯例
func (User) TableName() string {
	return "users"
}

// =============================================
// 管理员表
// =============================================

// Admin 管理员模型
type Admin struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Username    string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"` // 登录账号
	Password    string    `gorm:"type:varchar(100);not null" json:"-"`                   // 密码（bcrypt哈希）
	RealName    string    `gorm:"type:varchar(50);not null" json:"real_name"`            // 真实姓名
	Role        int       `gorm:"type:tinyint;not null;default:1" json:"role"`           // 1:一级管理员, 2:二级管理员
	Status      int       `gorm:"type:tinyint;not null;default:1" json:"status"`         // 1:正常, 0:禁用
	LastLoginAt time.Time `json:"last_login_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定管理员表名
func (Admin) TableName() string {
	return "admins"
}

// 管理员角色常量（保持向后兼容，推荐使用 internal/pkg/constants 包）
const (
	RoleLevel1 = 1 // 一级管理员
	RoleLevel2 = 2 // 二级管理员
)

// RoleText 返回角色中文描述
func RoleText(role int) string {
	switch role {
	case RoleLevel1:
		return "一级管理员"
	case RoleLevel2:
		return "二级管理员"
	default:
		return ""
	}
}
