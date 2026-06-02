// Package reservationdb 提供预约+审核数据库（home_res）的共享模型和数据访问。
// 由 reservation 和 admin 服务共同使用，确保数据操作逻辑单一归属。
package reservationdb

import "time"

// =============================================
// 状态常量（订单、时段、审核共用）
// =============================================

const (
	// 订单状态（按生命周期编号 1~7）
	StatusPendingLevel1  = 1 // 等待一级审核
	StatusPendingLevel2  = 2 // 等待二级审核
	StatusRejectedLevel1 = 3 // 一级审核拒绝
	StatusRejectedLevel2 = 4 // 二级审核拒绝
	StatusApproved       = 5 // 审核通过
	StatusCancelled      = 6 // 订单已经取消
	StatusCompleted      = 7 // 订单已经完成
)

// StatusText 返回状态码对应的中文描述。
//
// 参数:
//   - code: 状态码（1~7）
//
// 返回值:
//   - string: 状态码对应的中文描述，未匹配时返回 "未知状态"
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

// ReservationOrder 预约订单模型
// 对应数据库表: reservation_orders
//
// CREATE TABLE `reservation_orders` (
//
//		`id` INT UNSIGNED AUTO_INCREMENT,
//		`order_no` VARCHAR(50) NOT NULL,
//		`open_id`  VARCHAR(100) NOT NULL,
//	 	`applicant_name` VARCHAR(50) NOT NULL,
//		`alumni_association` VARCHAR(100) NOT NULL,
//		`year` INT NOT NULL,
//		`major` VARCHAR(30) NOT NULL,
//		`reason` VARCHAR(500) NOT NULL,
//		`phone` VARCHAR(20) NOT NULL,
//		`attendee_count` TINYINT UNSIGNED NOT NULL DEFAULT 1,
//		`total_slots` TINYINT UNSIGNED NOT NULL DEFAULT 1,
//		`status` TINYINT DEFAULT 1,
//		`created_at` DATETIME(3) NULL,
//		`updated_at` DATETIME(3) NULL,
//
//		PRIMARY KEY (`id`),
//		UNIQUE KEY `uni_reservation_orders_order_no` (`order_no`),
//		KEY `idx_reservation_orders_open_id` (`open_id`)
//
// )
type ReservationOrder struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	OrderNo           string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_no"`
	OpenID            string    `gorm:"column:open_id;type:varchar(100);index;not null" json:"openid"`
	ApplicantName     string    `gorm:"type:varchar(50);not null" json:"applicant_name"`             // 申请者姓名
	AlumniAssociation string    `gorm:"type:varchar(100);not null" json:"alumni_association"`        // 所属校友会
	Year              int       `gorm:"type:int;not null" json:"year"`                               // 年级
	Major             string    `gorm:"type:varchar(30);not null" json:"major"`                      // 专业
	Reason            string    `gorm:"type:varchar(500);not null" json:"reason"`                    // 申请理由（会议内容）
	Phone             string    `gorm:"type:varchar(20);not null" json:"phone"`                      // 手机号码
	AttendeeCount     int       `gorm:"type:tinyint unsigned;not null;default:1" json:"attendee_count"` // 会议人数
	TotalSlots        int       `gorm:"type:tinyint unsigned;not null;default:1" json:"total_slots"` // 预约的时间段数量
	Status            int       `gorm:"type:tinyint;default:1" json:"status"`                        // 预约请求的变化状态
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Slots []ReservationSlot `gorm:"foreignKey:OrderID" json:"slots,omitempty"` // 外键（一对多）
}

func (ReservationOrder) TableName() string { return "reservation_orders" }

// ReservationSlot 预约时段明细模型
// 对应数据库表: reservation_slots
//
// CREATE TABLE `reservation_slots` (
//
//		`id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
//		`order_id` BIGINT UNSIGNED NOT NULL,
//		`start_time` DATETIME(3) NOT NULL,
//		`status` TINYINT DEFAULT 1,
//		`end_time`	DATETIME(3) NOT NULL,
//		`password`	VARCHAR(20),
//		`created_at` DATETIME(3),
//		`updated_at` DATETIME(3),
//
//		PRIMARY KEY (`id`),
//		KEY `idx_reservation_slots_order_id` (`order_id`),
//	 	KEY `idx_reservation_slots_start_time` (`start_time`),
//		CONSTRAINT `fk_reservation_orders_slots`
//			FOREIGN KEY (`order_id`) REFERENCES	`reservation_orders` (`id`)
//			ON DELETE CASCADE ON UPDATE CASCADE
//
// )
type ReservationSlot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	OrderID   uint      `gorm:"type:int unsigned;index;not null" json:"order_id"`
	StartTime time.Time `gorm:"not null;index" json:"start_time"`     // 预约开始时间
	EndTime   time.Time `gorm:"not null" json:"end_time"`             // 预约结束时间
	Status    int       `gorm:"type:tinyint;default:1" json:"status"` // 预约状态
	Password  string    `gorm:"type:varchar(20)" json:"password"`     // 场地密码
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// gorm:"-"：告诉 GORM 操作数据库的使用，完全忽略这个字段。
	// json:"-"：告诉 json 操作标准库，当进行序列转换的时候，完全忽略这个字段。
	// 可以在查询 reservation_slot 记录的同时，将关联的 reservation_order 记录赋值给这个对象，
	// 有些业务处理需要从 reservation_slot 中获取 order 的相关信息。
	Order *ReservationOrder `gorm:"-" json:"-"`
}

func (ReservationSlot) TableName() string { return "reservation_slots" }

// ReviewRecord 审核记录模型
// 对应数据库表: review_records
//
// CREATE TABLE review_records (
//
//		`id` INT UNSIGNED NOT NULL,
//		`order_id` INT UNSIGNED NOT NULL,
//		`review_id` INT UNSIGNED NOT NULL,
//		`review_role` TINYINT NOT NULL,
//		`action` TINIINT NOT NULL,
//	 	`comment` VARCHAR(500),
//		`created_at` DATETIME(3)
//
// )
type ReviewRecord struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	OrderID      uint      `gorm:"type:int unsigned;index;not null" json:"order_id"`
	ReviewerID   uint      `gorm:"type:bigint unsigned;not null" json:"reviewer_id"` // 管理员id
	ReviewerRole int       `gorm:"type:tinyint;not null" json:"reviewer_role"`       // 管理员层级
	Action       int       `gorm:"type:tinyint;not null" json:"action"`              // 是同意还是拒绝
	Comment      string    `gorm:"type:varchar(500)" json:"comment"`                 // 同意或者拒绝的理由
	CreatedAt    time.Time `json:"created_at"`
}

func (ReviewRecord) TableName() string { return "review_records" }
