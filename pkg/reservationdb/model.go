// Package reservationdb 提供预约+审核数据库（home_res）的共享模型和数据访问。
// 由 reservation 和 admin 服务共同使用，确保数据操作逻辑单一归属。
package reservationdb

import "time"

// =============================================
// 状态常量（订单、时段、审核共用）
// =============================================

const (
	// 基础状态
	StatusPending   = 0 // 待审核
	StatusApproved  = 1 // 已通过
	StatusRejected  = 2 // 已拒绝
	StatusCompleted = 3 // 已完成
	StatusCancelled = 4 // 已取消

	// 审核扩展状态
	StatusPendingLevel1  = 5 // 待一级审核
	StatusPendingLevel2  = 6 // 待二级审核（一级已通过）
	StatusRejectedLevel1 = 7 // 一级审核拒绝
	StatusRejectedLevel2 = 8 // 二级审核拒绝
	StatusApprovedFinal  = 9 // 终审通过（二级审核通过）
)

// StatusText 返回状态码对应的中文描述。
//
// 参数:
//   - code: 状态码（0~9）
//
// 返回值:
//   - string: 状态码对应的中文描述，未匹配时返回 "未知状态"
func StatusText(code int) string {
	switch code {
	case StatusPending:
		return "待审核"
	case StatusApproved:
		return "已通过"
	case StatusRejected:
		return "已拒绝"
	case StatusCompleted:
		return "已完成"
	case StatusCancelled:
		return "已取消"
	case StatusPendingLevel2:
		return "待二级审核"
	case StatusRejectedLevel1:
		return "一级驳回"
	case StatusRejectedLevel2:
		return "二级驳回"
	default:
		return "未知状态"
	}
}

// =============================================
// 预约订单表
// =============================================

// ReservationOrder 预约订单模型
// 对应数据库表: reservation_orders
//
// 字段说明:
//   - OrderNo: 订单号，格式 R{时间戳}{4位随机hex}，唯一索引
//   - OpenID: 微信用户唯一标识，普通索引
//   - TotalSlots: 预约时段数量，默认1
//   - Status: 订单状态，参见 StatusXxx 常量
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
	TotalSlots        int       `gorm:"type:tinyint unsigned;not null;default:1" json:"total_slots"`
	Status            int       `gorm:"type:tinyint;default:0" json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Slots []ReservationSlot `gorm:"foreignKey:OrderID" json:"slots,omitempty"`
}

// TableName 指定表名（reservation_orders）
func (ReservationOrder) TableName() string { return "reservation_orders" }

// =============================================
// 预约时段明细表
// =============================================

// ReservationSlot 预约时段明细模型
// 对应数据库表: reservation_slots
//
// 字段说明:
//   - OrderID: 关联的订单ID（外键 → reservation_orders.id）
//   - StartTime/EndTime: 预约起止时间
//   - Status: 时段状态，与订单状态同步
//   - Password: 审核通过后由管理员设置的门锁密码
type ReservationSlot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OrderID   uint      `gorm:"type:bigint unsigned;index;not null" json:"order_id"`
	StartTime time.Time `gorm:"not null;index" json:"start_time"`
	EndTime   time.Time `gorm:"not null" json:"end_time"`
	Status    int       `gorm:"type:tinyint;default:0" json:"status"`
	Password  string    `gorm:"type:varchar(20)" json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Order *ReservationOrder `gorm:"-" json:"-"`
}

// TableName 指定表名（reservation_slots）
func (ReservationSlot) TableName() string { return "reservation_slots" }

// =============================================
// 审核记录表
// =============================================

// ReviewRecord 审核记录模型
// 对应数据库表: review_records
//
// 字段说明:
//   - OrderID: 关联的订单ID（外键 → reservation_orders.id）
//   - ReviewerID: 审核人ID（关联 admins.id）
//   - ReviewerRole: 审核人角色（1:一级管理员, 2:二级管理员）
//   - Action: 审核动作（1:通过, 2:拒绝）
//   - Comment: 审核意见，最大500字
type ReviewRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	OrderID      uint      `gorm:"type:bigint unsigned;index;not null" json:"order_id"`
	ReviewerID   uint      `gorm:"type:bigint unsigned;not null" json:"reviewer_id"`
	ReviewerRole int       `gorm:"type:tinyint;not null" json:"reviewer_role"`
	Action       int       `gorm:"type:tinyint;not null" json:"action"`
	Comment      string    `gorm:"type:varchar(500)" json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
}

// TableName 指定表名（review_records）
func (ReviewRecord) TableName() string { return "review_records" }
