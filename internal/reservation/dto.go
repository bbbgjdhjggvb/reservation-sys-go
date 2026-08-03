package reservation

import "reservation-sys/internal/model"

// ========== 请求结构 ==========

// TimeSlotReq 单个时间段请求
type TimeSlotReq struct {
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

// SubmitReq 预约提交请求
type SubmitReq struct {
	ApplicantName     string        `json:"applicant_name" binding:"required"`
	AlumniAssociation string        `json:"alumni_association" binding:"required"`
	Year              int           `json:"year" binding:"required"`
	Major             string        `json:"major" binding:"required"`
	Reason            string        `json:"reason" binding:"required,max=500"`
	Phone             string        `json:"phone" binding:"required,len=11"`
	AttendeeCount     int           `json:"attendee_count" binding:"required,min=1"`
	Slots             []TimeSlotReq `json:"slots" binding:"required,min=1,max=4,dive"`
}

// ========== 响应结构 ==========

// SlotResp 单个时段响应
type SlotResp struct {
	ID         uint   `json:"id"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Status     int    `json:"status"`
	StatusText string `json:"status_text"`
	Password   string `json:"password,omitempty"`
}

// OrderResp 订单响应
type OrderResp struct {
	ID                uint       `json:"id"`
	OrderNo           string     `json:"order_no"`
	ApplicantName     string     `json:"applicant_name"`
	AlumniAssociation string     `json:"alumni_association"`
	Year              int        `json:"year"`
	Major             string     `json:"major"`
	Reason            string     `json:"reason"`
	Phone             string     `json:"phone"`
	AttendeeCount     int        `json:"attendee_count"`
	TotalSlots        int        `json:"total_slots"`
	Status            int        `json:"status"`
	StatusText        string     `json:"status_text"`
	CreatedAt         string     `json:"created_at"`
	Slots             []SlotResp `json:"slots"`
}

// TimeSlotResp 已占用时间段响应
type TimeSlotResp struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Status    string `json:"status"`
	IsMine    bool   `json:"is_mine"`
}

// Response 统一响应结构
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

// ========== 转换方法 ==========

// SlotToResp 将数据库模型转为 HTTP 响应 DTO。
func SlotToResp(s *model.ReservationSlot, showPassword ...bool) *SlotResp {
	resp := &SlotResp{
		ID:         s.ID,
		StartTime:  s.StartTime.Format("2006-01-02 15:04"),
		EndTime:    s.EndTime.Format("2006-01-02 15:04"),
		Status:     s.Status,
		StatusText: model.StatusText(s.Status),
	}
	if len(showPassword) > 0 && showPassword[0] && s.Password != "" {
		resp.Password = s.Password
	}
	return resp
}

// OrderToResp 将数据库模型转为 HTTP 响应 DTO。
func OrderToResp(o *model.ReservationOrder, showPassword ...bool) *OrderResp {
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
		StatusText:        model.StatusText(o.Status),
		CreatedAt:         o.CreatedAt.Format("2006-01-02 15:04"),
		Slots:             slots,
	}
}
