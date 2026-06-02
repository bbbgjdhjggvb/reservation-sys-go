package reservation

import (
	reservationdb "reservation-sys/pkg/reservationdb"
)

// ========== 请求结构 ==========

// TimeSlotReq 单个时间段请求
type TimeSlotReq struct {
	StartTime string `json:"start_time" binding:"required" example:"2026-01-01 08:00:00"`
	EndTime   string `json:"end_time" binding:"required" example:"2026-01-01 10:00:00"`
}

// SubmitReq 预约提交请求
type SubmitReq struct {
	ApplicantName     string        `json:"applicant_name" binding:"required" example:"张三"`
	AlumniAssociation string        `json:"alumni_association" binding:"required" example:"土木与交通工程学院校友会"`
	Year              int           `json:"year" binding:"required" example:"2000"`
	Major             string        `json:"major" binding:"required" example:"建筑系"`
	Reason            string        `json:"reason" binding:"required,max=500" example:"智慧城市讲座"`
	Phone             string        `json:"phone" binding:"required,len=11" example:"13800138000"`
	AttendeeCount     int           `json:"attendee_count" binding:"required,min=1" example:"10"`
	Slots             []TimeSlotReq `json:"slots" binding:"required,min=1,max=4,dive"`
}

// ========== 响应结构 ==========

// SlotResp 单个时段响应
type SlotResp struct {
	ID         uint   `json:"id" example:"1"`
	StartTime  string `json:"start_time" example:"2026-01-01 08:00"`
	EndTime    string `json:"end_time" example:"2026-01-01 10:00"`
	Status     int    `json:"status" example:"0"`
	StatusText string `json:"status_text" example:"待审核"`
	Password   string `json:"password,omitempty" example:"123456"`
}

// OrderResp 订单响应
type OrderResp struct {
	ID                uint       `json:"id" example:"1"`
	OrderNo           string     `json:"order_no" example:"R202601010900001234"`
	ApplicantName     string     `json:"applicant_name" example:"张三"`
	AlumniAssociation string     `json:"alumni_association" example:"土木与交通工程学院校友会"`
	Year              int        `json:"year" example:"2000"`
	Major             string     `json:"major" example:"建筑系"`
	Reason            string     `json:"reason" example:"智慧城市讲座"`
	Phone             string     `json:"phone" example:"13800138000"`
	AttendeeCount     int        `json:"attendee_count" example:"10"`
	TotalSlots        int        `json:"total_slots" example:"2"`
	Status            int        `json:"status" example:"0"`
	StatusText        string     `json:"status_text" example:"待审核"`
	CreatedAt         string     `json:"created_at" example:"2026-01-01 08:30"`
	Slots             []SlotResp `json:"slots"`
}

// TimeSlotResp 已占用时间段响应
type TimeSlotResp struct {
	StartTime string `json:"start_time" example:"2026-01-01 09:00"`
	EndTime   string `json:"end_time" example:"2026-01-01 11:00"`
	Status    string `json:"status" example:"pending"`
}

// Response 统一响应结构
type Response struct {
	Code int    `json:"code" example:"200"`
	Msg  string `json:"msg" example:"success"`
	Data any    `json:"data"`
}

// ========== 转换方法 ==========

// SlotToResp 将数据库 ReservationSlot 模型转为 HTTP 响应 DTO。
//
// 参数:
//   - s: 数据库时段模型
//   - showPassword: 可选，为 true 且密码非空时在响应中包含密码字段
//
// 返回值:
//   - *SlotResp: HTTP 响应时段 DTO
func SlotToResp(s *reservationdb.ReservationSlot, showPassword ...bool) *SlotResp {
	resp := &SlotResp{
		ID:         s.ID,
		StartTime:  s.StartTime.Format("2006-01-02 15:04"),
		EndTime:    s.EndTime.Format("2006-01-02 15:04"),
		Status:     s.Status,
		StatusText: reservationdb.StatusText(s.Status),
	}
	if len(showPassword) > 0 && showPassword[0] && s.Password != "" {
		resp.Password = s.Password
	}
	return resp
}

// OrderToResp 将数据库 ReservationOrder 模型转为 HTTP 响应 DTO。
//
// 参数:
//   - o: 数据库订单模型（含关联时段）
//   - showPassword: 可选，为 true 时在时段响应中包含密码字段
//
// 返回值:
//   - *OrderResp: HTTP 响应订单 DTO
func OrderToResp(o *reservationdb.ReservationOrder, showPassword ...bool) *OrderResp {
	slots := make([]SlotResp, 0, len(o.Slots))
	for _, s := range o.Slots {
		slots = append(slots, *SlotToResp(&s, showPassword...))
	}
	return &OrderResp{
		ID:                o.ID,
		OrderNo:           o.OrderNo,
		ApplicantName:     o.ApplicantName,
		AlumniAssociation: o.AlumniAssociation,
		Year:              o.Year,
		Major:             o.Major,
		Reason:            o.Reason,
		Phone:             o.Phone,
		AttendeeCount:     o.AttendeeCount,
		TotalSlots:        o.TotalSlots,
		Status:            o.Status,
		StatusText:        reservationdb.StatusText(o.Status),
		CreatedAt:         o.CreatedAt.Format("2006-01-02 15:04"),
		Slots:             slots,
	}
}
