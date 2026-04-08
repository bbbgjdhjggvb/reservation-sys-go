/* dto：data tranform object 数据传输对象
 * 在前段的H5页面，用户点击提交之后将会传输一段json数据到后台
 * dto.go 定义传输的内容
 */
package reservation

type SubmitReq struct {
	ApplicantName     string `json:"applicant_name" binding:"required"`
	AlumniAssociation string `json:"alumni_association" binding:"required"`
	Reason            string `json:"reason" binding:"required,max=500"`
	Phone             string `json:"phone" binding:"required,len=11"`
	StartTime         string `json:"start_time" binding:"required"`
	EndTime           string `json:"end_time" binding:"required"`
}

// ReservationResp 预约响应
type ReservationResp struct {
	ID              uint   `json:"id"`
	OrderNo         string `json:"order_no"`
	ApplicationName string `json:"applicant_name"`
	Reason          string `json:"reason"`
	Phone           string `json:"phone"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	Status          int    `json:"status"`
	StatusText      string `json:"status_text"`
	CreatedAt       string `json:"created_at"`
}

// ToResp 转换为响应格式
func (r *Reservation) ToResp() *ReservationResp {
	statusText := map[int]string{
		StatusPending:   "待审核",
		StatusApproved:  "已通过",
		StatusRejected:  "已拒绝",
		StatusCompleted: "已完成",
		StatusCancelled: "已取消",
	}

	return &ReservationResp{
		ID:              r.ID,
		OrderNo:         r.OrderNo,
		ApplicationName: r.ApplicationName,
		Reason:          r.Reason,
		Phone:           r.Phone,
		StartTime:       r.StartTime.Format("2006-01-02 15:04"),
		EndTime:         r.EndTime.Format("2006-01-02 15:04"),
		Status:          r.Status,
		StatusText:      statusText[r.Status],
		CreatedAt:       r.CreatedAt.Format("2006-01-02 15:04"),
	}
}
