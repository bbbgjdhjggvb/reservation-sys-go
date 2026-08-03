package model

import "time"

// =============================================
// 订单状态常量
// =============================================

const (
	StatusPendingLevel1  = 1 // 等待一级审核
	StatusPendingLevel2  = 2 // 等待二级审核
	StatusRejectedLevel1 = 3 // 一级审核拒绝
	StatusRejectedLevel2 = 4 // 二级审核拒绝
	StatusApproved       = 5 // 审核通过
	StatusCancelled      = 6 // 订单已经取消
	StatusCompleted      = 7 // 订单已经完成
)

// StatusText 返回状态码对应的中文描述
func StatusText(code int) string {
	switch code {
	case StatusPendingLevel1:
		return "等待一级审核"
	case StatusPendingLevel2:
		return "等待二级审核"
	case StatusRejectedLevel1:
		return "一级审核拒绝"
	case StatusRejectedLevel2:
		return "二级审核拒绝"
	case StatusApproved:
		return "审核通过"
	case StatusCancelled:
		return "订单已经取消"
	case StatusCompleted:
		return "订单已经完成"
	default:
		return "未知状态"
	}
}

// =============================================
// 预约订单表 (reservation_orders)
// =============================================

// ReservationOrder 预约订单模型
type ReservationOrder struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	OrderNo           string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_no"`
	OpenID            string    `gorm:"column:open_id;type:varchar(100);index;not null" json:"openid"`
	ApplicantName     string    `gorm:"type:varchar(50);not null" json:"applicant_name"`
	AlumniAssociation string    `gorm:"type:varchar(100);not null" json:"alumni_association"`
	Year              int       `gorm:"type:int;not null" json:"year"`
	Major             string    `gorm:"type:varchar(30);not null" json:"major"`
	Reason            string    `gorm:"type:varchar(500);not null" json:"reason"`
	Phone             string    `gorm:"type:varchar(20);not null" json:"phone"`
	AttendeeCount     int       `gorm:"type:tinyint unsigned;not null;default:1" json:"attendee_count"`
	TotalSlots        int       `gorm:"type:tinyint unsigned;not null;default:1" json:"total_slots"`
	Status            int       `gorm:"type:tinyint;default:1" json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Slots []ReservationSlot `gorm:"foreignKey:OrderID" json:"slots,omitempty"`
}

// TableName 指定订单表名
func (ReservationOrder) TableName() string { return "reservation_orders" }

// =============================================
// 预约时段表 (reservation_slots)
// =============================================

// ReservationSlot 预约时段明细模型
type ReservationSlot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OrderID   uint      `gorm:"type:int unsigned;index;not null" json:"order_id"`
	StartTime time.Time `gorm:"not null;index" json:"start_time"`
	EndTime   time.Time `gorm:"not null" json:"end_time"`
	Status    int       `gorm:"type:tinyint;default:1" json:"status"`
	Password  string    `gorm:"type:varchar(20)" json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Order *ReservationOrder `gorm:"-" json:"-"`
}

// TableName 指定时段表名
func (ReservationSlot) TableName() string { return "reservation_slots" }

// =============================================
// 审核记录表 (review_records)
// =============================================

// ReviewRecord 审核记录模型
type ReviewRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	OrderID      uint      `gorm:"type:int unsigned;index;not null" json:"order_id"`
	ReviewerID   uint      `gorm:"type:bigint unsigned;not null" json:"reviewer_id"`
	ReviewerRole int       `gorm:"type:tinyint;not null" json:"reviewer_role"`
	Action       int       `gorm:"type:tinyint;not null" json:"action"`
	Comment      string    `gorm:"type:varchar(500)" json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定审核记录表名
func (ReviewRecord) TableName() string { return "review_records" }
