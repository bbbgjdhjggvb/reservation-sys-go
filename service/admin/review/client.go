package review

import (
	reservationdb "reservation-sys/pkg/reservationdb"
)

// ========== 预约订单 → HTTP 响应转换 ==========

// OrderResp 订单响应（用于 HTTP API）
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

// OrderToResp 将 ReservationOrder 转为 HTTP 响应
func OrderToResp(o *reservationdb.ReservationOrder, showPassword bool) *OrderResp {
	slots := make([]SlotResp, 0, len(o.Slots))
	for _, s := range o.Slots {
		slot := SlotResp{
			ID:         s.ID,
			StartTime:  s.StartTime.Format("2006-01-02 15:04"),
			EndTime:    s.EndTime.Format("2006-01-02 15:04"),
			Status:     s.Status,
			StatusText: reservationdb.StatusText(s.Status),
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
		TotalSlots:        o.TotalSlots,
		Status:            o.Status,
		StatusText:        reservationdb.StatusText(o.Status),
		CreatedAt:         o.CreatedAt.Format("2006-01-02 15:04"),
		Slots:             slots,
	}
}
