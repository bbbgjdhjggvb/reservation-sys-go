package auth

import (
	"time"
)

// User 对应数据库中的 users 表
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OpenID    string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"openid"` // 微信唯一标识，加唯一索引
	Nickname  string    `gorm:"type:varchar(255)" json:"nickname"`                    // 昵称
	Status    int       `gorm:"type:tinyint;default:1;comment:'1:正常,0:已取消关注'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LastLogin time.Time `json:"last_login"` // 记录最后交互时间
}

// TableName 指定数据库表名，符合 Go 惯例
func (User) TableName() string {
	return "users"
}
