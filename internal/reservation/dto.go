/* dto：data transfer object 数据传输对象
 * 前端H5页面提交预约时传输的JSON数据格式
 * 支持多时段批量提交（一次请求最多4个时间段）
 */
package reservation

// ========== 请求结构 ==========

// TimeSlotReq 单个时间段请求
type TimeSlotReq struct {
	StartTime string `json:"start_time" binding:"required" example:"2026-01-01 08:00:00"` // 开始时间
	EndTime   string `json:"end_time" binding:"required" example:"2026-01-01 10:00:00"`   // 结束时间
}

// SubmitReq 预约提交请求（支持多时间段）
type SubmitReq struct {
	ApplicantName     string         `json:"applicant_name" binding:"required" example:"张三"`
	AlumniAssociation string         `json:"alumni_association" binding:"required" example:"土木与交通工程学院校友会"`
	Year              int            `json:"year" binding:"required" example:"2000"`
	Major             string         `json:"major" binding:"required" example:"建筑系"`
	Reason            string         `json:"reason" binding:"required,max=500" example:"智慧城市讲座"`
	Phone             string         `json:"phone" binding:"required,len=11" example:"13800138000"`
	Slots             []TimeSlotReq  `json:"slots" binding:"required,min=1,max=4,dive"`       // 预约时间段列表（1~4个）
}

// ========== 响应结构 ==========

// SlotResp 单个时段响应
type SlotResp struct {
	ID        uint   `json:"id" example:"1"`
	StartTime string `json:"start_time" example:"2026-01-01 08:00"`
	EndTime   string `json:"end_time" example:"2026-01-01 10:00"`
	Status    int    `json:"status" example:"0"`
	StatusText string `json:"status_text" example:"待审核"`
}

// OrderResp 订单响应（包含时段明细）
type OrderResp struct {
	ID                uint      `json:"id" example:"1"`
	OrderNo           string    `json:"order_no" example:"R202601010900001234"`
	ApplicantName     string    `json:"applicant_name" example:"张三"`
	AlumniAssociation string    `json:"alumni_association" example:"土木与交通工程学院校友会"`
	Year              int       `json:"year" example:"2000"`
	Major             string    `json:"major" example:"建筑系"`
	Reason            string    `json:"reason" example:"智慧城市讲座"`
	Phone             string    `json:"phone" example:"13800138000"`
	TotalSlots        int       `json:"total_slots" example:"2"`
	Status            int       `json:"status" example:"0"`
	StatusText        string    `json:"status_text" example:"待审核"`
	CreatedAt         string    `json:"created_at" example:"2026-01-01 08:30"`
	Slots             []SlotResp `json:"slots"` // 时段明细列表
}

// TimeSlotResp 已占用时间段响应（前端日历展示用）
type TimeSlotResp struct {
	StartTime string `json:"start_time" example:"2026-01-01 09:00"`
	EndTime   string `json:"end_time" example:"2026-01-01 11:00"`
	Status    string `json:"status" example:"pending"` // pending待审核 / approved已通过
}

// SubmitResult 批量提交结果
type SubmitResult struct {
	OrderNo       string    `json:"order_no"`        // 生成的订单号
	SuccessCount  int       `json:"success_count"`   // 成功提交的时段数
	TotalCount    int       `json:"total_count"`     // 总时段数
	FailedSlots   []int     `json:"failed_slots,omitempty"` // 失败的时段索引(0-based)
}

// Response 统一响应结构
type Response struct {
	Code int    `json:"code" example:"200"`
	Msg  string `json:"msg" example:"success"`
	Data any    `json:"data"`
}

// ========== 转换方法 ==========

// ToSlotResp 将 ReservationSlot 转为 SlotResp
func (s *ReservationSlot) ToSlotResp() *SlotResp {
	return &SlotResp{
		ID:         s.ID,
		StartTime:  s.StartTime.Format("2006-01-02 15:04"),
		EndTime:    s.EndTime.Format("2006-01-02 15:04"),
		Status:     s.Status,
		StatusText: StatusText(s.Status),
	}
}

// ToOrderResp 将 ReservationOrder + Slots 转为 OrderResp
func (o *ReservationOrder) ToOrderResp() *OrderResp {
	slots := make([]SlotResp, 0, len(o.Slots))
	for _, s := range o.Slots {
		slots = append(slots, *s.ToSlotResp())
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
		TotalSlots:        o.TotalSlots,
		Status:            o.Status,
		StatusText:        StatusText(o.Status),
		CreatedAt:         o.CreatedAt.Format("2006-01-02 15:04"),
		Slots:             slots,
	}
}
