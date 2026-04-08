// internal/reservation/service.go
package reservation

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// TimeSlot 时间段结构，用于前端展示已占用时段
type TimeSlot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"` // "pending" 或 "approved"
}

// ReservationService 预约业务服务
type ReservationService struct {
	repo ReservationRepository
}

// NewReservationService 创建服务实例
func NewReservationService(repo ReservationRepository) *ReservationService {
	return &ReservationService{repo: repo}
}

// Submit 提交预约申请
func (s *ReservationService) Submit(openid string, req *SubmitReq) (*Reservation, error) {
	// 解析时间
	layout := "2006-01-02 15:04:05"
	startTime, err1 := time.ParseInLocation(layout, req.StartTime, time.Local)
	endTime, err2 := time.ParseInLocation(layout, req.EndTime, time.Local)

	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("时间格式错误")
	}

	if !endTime.After(startTime) {
		return nil, fmt.Errorf("结束时间必须晚于开始时间")
	}

	// 检查时间段是否已被占用
	occupied, err := s.repo.FindByTimeRange(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("查询占用时段失败: %v", err)
	}

	if len(occupied) > 0 {
		return nil, fmt.Errorf("该时间段已被预约")
	}

	// 生成订单号：日期+随机数
	orderNo := generateOrderNo()

	// 创建预约记录
	res := &Reservation{
		OrderNo:         orderNo,
		OpenID:          openid,
		ApplicationName: req.ApplicantName,
		Reason:          req.Reason,
		Phone:           req.Phone,
		StartTime:       startTime,
		EndTime:         endTime,
		Status:          StatusPending,
	}

	if err := s.repo.Create(res); err != nil {
		return nil, fmt.Errorf("创建预约失败: %v", err)
	}

	return res, nil
}

// GetMyReservations 获取用户的预约列表
func (s *ReservationService) GetMyReservations(openid string) ([]*Reservation, error) {
	return s.repo.FindByOpenID(openid)
}

// GetOccupiedSlots 获取指定日期的已占用时间段
func (s *ReservationService) GetOccupiedSlots(date string) ([]TimeSlot, error) {
	// 解析日期，构建当天的起止时间
	day, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return nil, fmt.Errorf("日期格式错误")
	}

	startOfDay := day
	endOfDay := day.Add(24 * time.Hour)

	// 查询当天的预约
	reservations, err := s.repo.FindByTimeRange(startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	// 转换为时间段格式
	slots := make([]TimeSlot, 0, len(reservations))
	for _, res := range reservations {
		status := "pending"
		if res.Status == StatusApproved {
			status = "approved"
		}
		slots = append(slots, TimeSlot{
			StartTime: res.StartTime,
			EndTime:   res.EndTime,
			Status:    status,
		})
	}

	return slots, nil
}

// Cancel 取消预约
func (s *ReservationService) Cancel(id uint, openid string) error {
	// 检查预约是否存在且属于当前用户
	res, err := s.repo.FindByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("预约不存在")
		}
		return err
	}

	if res.OpenID != openid {
		return fmt.Errorf("无权操作此预约")
	}

	if res.Status != StatusPending && res.Status != StatusApproved {
		return fmt.Errorf("当前状态无法取消")
	}

	return s.repo.Cancel(id, openid)
}

// generateOrderNo 生成订单号
func generateOrderNo() string {
	return fmt.Sprintf("R%s%d", time.Now().Format("20060102150405"), time.Now().Nanosecond()%10000)
}
