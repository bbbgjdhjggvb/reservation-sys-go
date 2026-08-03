package review

import "reservation-sys/internal/model"

// ========== 请求结构 ==========

// ReviewActionReq 审核操作请求
type ReviewActionReq struct {
	Action  int    `json:"action" binding:"required,oneof=1 2"`
	Comment string `json:"comment" binding:"max=500"`
}

// SetPasswordReq 设置门锁密码请求
type SetPasswordReq struct {
	Password string `json:"password" binding:"required,max=20"`
}

// RejectionNotifyReq 驳回通知请求
type RejectionNotifyReq struct {
	Reason string `json:"reason" binding:"max=500"`
}

// ========== 响应结构 ==========

// ReviewRecordResp 审核记录响应
type ReviewRecordResp struct {
	ID           uint   `json:"id"`
	ReviewerName string `json:"reviewer_name"`
	ReviewerRole int    `json:"reviewer_role"`
	RoleText     string `json:"role_text"`
	Action       int    `json:"action"`
	ActionText   string `json:"action_text"`
	Comment      string `json:"comment"`
	CreatedAt    string `json:"created_at"`
}

// Response 统一响应结构
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

// OrderResp 订单响应
type OrderResp struct {
	ID                uint       `json:"id"`
	OrderNo           string     `json:"order_no"`
	OpenID            string     `json:"openid"`
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

// SlotResp 时段响应
type SlotResp struct {
	ID         uint   `json:"id"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Status     int    `json:"status"`
	StatusText string `json:"status_text"`
	Password   string `json:"password,omitempty"`
}

// ========== 转换方法 ==========

// OrderToResp 将 ReservationOrder 转为 HTTP 响应
func OrderToResp(o *model.ReservationOrder, showPassword bool) *OrderResp {
	slots := make([]SlotResp, 0, len(o.Slots))
	for _, s := range o.Slots {
		slot := SlotResp{
			ID:         s.ID,
			StartTime:  s.StartTime.Format("2006-01-02 15:04"),
			EndTime:    s.EndTime.Format("2006-01-02 15:04"),
			Status:     s.Status,
			StatusText: model.StatusText(s.Status),
		}
		if showPassword && s.Password != "" {
			slot.Password = s.Password
		}
		slots = append(slots, slot)
	}

	return &OrderResp{
		ID:                o.ID,
		OrderNo:           o.OrderNo,
		OpenID:            o.OpenID,
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

// ActionText 返回审核操作的中文描述
func ActionText(action int) string {
	if action == 1 {
		return "通过"
	}
	return "拒绝"
}
