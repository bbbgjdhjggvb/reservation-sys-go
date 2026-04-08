package reservation

import "time"

// 预约状态常量
const (
	StatusPending   = 0 // 待审核
	StatusApproved  = 1 // 已通过
	StatusRejected  = 2 // 已拒绝
	StatusCompleted = 3 // 已完成
	StatusCancelled = 4 // 已取消
)

type Reservation struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	OrderNo         string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_no"`               // 订单号
	OpenID          string    `gorm:"type:varchar(100);index;not null" json:"openid"`                      // 预约人标识
	ApplicationName string    `gorm:"type:varchar(50);not null" json:"application_name"`                   // 预约人名称
	Reason          string    `gorm:"type:varchar(500);not null" json:"reason"`                            // 预约理由
	Phone           string    `gorm:"type:varchar(20);not null" json:"phone"`                              // 电话号码
	Num             int       `gorm:"type:int;not null" json:"num"`                                        // 预约人数
	StartTime       time.Time `gorm:"not null" json:"start_time"`                                          // 预约开始时间
	EndTime         time.Time `gorm:"not null" json:"end_time"`                                            // 预约结束时间
	Status          int       `gorm:"type:tinyint;default:0;comment:'0:待审核,1:通过,2:拒绝,3:完成'" json:"status"` // 预约状态
	Password        string    `gorm:"type:varchar(20)" json:"password"`                                    // 门锁下发的动态密码
	CreatedAt       time.Time `json:"created_at"`                                                          // 预约时间
	UpdatedAt       time.Time `json:"updated_at"`                                                          // 预约状态变更时间
}

func (Reservation) TableName() string {
	return "reservations"
}
