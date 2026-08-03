// Package model 提供跨模块共享的 GORM 模型和状态常量。
// 所有 store 包依赖此包进行数据库操作。
package model

import "time"

// =============================================
// 用户表 (users)
// =============================================

// User 对应数据库中的 users 表
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OpenID    string    `gorm:"column:openid;type:varchar(100);not null;uniqueIndex" json:"openid"`
	Nickname  string    `gorm:"type:varchar(255)" json:"nickname"`
	Status    int       `gorm:"type:tinyint;default:1;comment:'1:正常,0:已取消关注'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastLogin time.Time `json:"last_login"`
}

// TableName 指定数据库表名
func (User) TableName() string {
	return "users"
}

// =============================================
// 管理员表 (admins)
// =============================================

// Admin 管理员模型
type Admin struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Username    string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Password    string    `gorm:"type:varchar(100);not null" json:"-"`
	RealName    string    `gorm:"type:varchar(50);not null" json:"real_name"`
	Role        int       `gorm:"type:tinyint;not null;default:1" json:"role"`
	Status      int       `gorm:"type:tinyint;not null;default:1" json:"status"`
	LastLoginAt time.Time `json:"last_login_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定管理员表名
func (Admin) TableName() string {
	return "admins"
}

// =============================================
// 管理员角色常量
// =============================================

const (
	RoleLevel1 = 1 // 一级管理员
	RoleLevel2 = 2 // 二级管理员
)

// RoleText 返回管理员角色对应的中文描述
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
