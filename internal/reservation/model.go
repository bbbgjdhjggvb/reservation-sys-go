package reservation

import "time"

// 预约状态常量（订单和时段共用）
const (
	StatusPending   = 0 // 待审核
	StatusApproved  = 1 // 已通过
	StatusRejected  = 2 // 已拒绝
	StatusCompleted = 3 // 已完成
	StatusCancelled = 4 // 已取消
)

// StatusText 返回状态码对应的中文描述
func StatusText(code int) string {
	texts := map[int]string{
		StatusPending:   "待审核",
		StatusApproved:  "已通过",
		StatusRejected:  "已拒绝",
		StatusCompleted: "已完成",
		StatusCancelled: "已取消",
	}
	return texts[code]
}

// =============================================
// 预约订单表：一次提交生成一个订单
// 存放申请人信息和共享字段
// =============================================
type ReservationOrder struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	OrderNo           string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_no"`
	OpenID            string    `gorm:"type:varchar(100);index;not null" json:"openid"`
	ApplicantName     string    `gorm:"type:varchar(50);not null" json:"applicant_name"`
	AlumniAssociation string    `gorm:"type:varchar(100);not null" json:"alumni_association"`
	Year              int       `gorm:"type:int;not null" json:"year"`
	Major             string    `gorm:"type:varchar(30);not null" json:"major"`
	Reason            string    `gorm:"type:varchar(500);not null" json:"reason"`
	Phone             string    `gorm:"type:varchar(20);not null" json:"phone"`
	TotalSlots        int       `gorm:"type:tinyint unsigned;not null;default:1" json:"total_slots"` // 预约时段数量
	Status            int       `gorm:"type:tinyint;default:0" json:"status"`                       // 整体状态
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// 关联：该订单下的所有时段（GORM 自动加载）
	Slots []ReservationSlot `gorm:"foreignKey:OrderID" json:"slots,omitempty"`
}

func (ReservationOrder) TableName() string {
	return "reservation_orders"
}

// =============================================
// 预约时段明细表：每个时间段一行
// 独立状态、独立的门锁密码
// =============================================
type ReservationSlot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OrderID   uint      `gorm:"type:bigint unsigned;index;not null" json:"order_id"`          // 关联订单ID
	StartTime time.Time `gorm:"not null;index" json:"start_time"`                             // 开始时间
	EndTime   time.Time `gorm:"not null" json:"end_time"`                                     // 结束时间
	Status    int       `gorm:"type:tinyint;default:0" json:"status"`                         // 时段状态
	Password  string    `gorm:"type:varchar(20)" json:"password"`                             // 门锁动态密码
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 反向关联（可选）
	Order *ReservationOrder `gorm:"-" json:"-"`
}

func (ReservationSlot) TableName() string {
	return "reservation_slots"
}

// =============================================
// 向后兼容：保留 Reservation 别名，指向 ReservationOrder
// =============================================

// Reservation 是 ReservationOrder 的别名，保持向后兼容
// Deprecated: 新代码请使用 ReservationOrder 和 ReservationSlot
type Reservation = ReservationOrder
