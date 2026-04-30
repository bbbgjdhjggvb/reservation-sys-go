// internal/reservation/service.go
package reservation

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	"gorm.io/gorm"
)

// TimeSlot 时间段结构，用于前端展示已占用时段
type TimeSlot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"` // "pending" 或 "approved"
}

// ParsedSlot 解析后的时间段
type ParsedSlot struct {
	StartTime time.Time
	EndTime   time.Time
}

// ReservationService 预约业务服务
type ReservationService struct {
	repo ReservationRepository
}

// NewReservationService 创建服务实例
func NewReservationService(repo ReservationRepository) *ReservationService {
	return &ReservationService{repo: repo}
}

// Submit 批量提交预约申请（支持多时间段，自动合并同日连续时段）
func (s *ReservationService) Submit(openid string, slots []ParsedSlot, req *SubmitReq) (*ReservationOrder, error) {
	if len(slots) == 0 || len(slots) > 4 {
		return nil, fmt.Errorf("预约时段数量必须在1~4之间")
	}

	// 合并同一天的连续时间段
	mergedSlots := mergeContinuousSlots(slots)
	if len(mergedSlots) == 0 {
		return nil, fmt.Errorf("合并后无有效时段")
	}
	log.Printf("[info][service/Submit] 原始时段数=%d, 合并后=%d", len(slots), len(mergedSlots))

	orderNo := generateOrderNo()

	// 构建订单记录（TotalSlots 使用合并后的数量）
	order := &ReservationOrder{
		OrderNo:           orderNo,
		OpenID:            openid,
		ApplicantName:     req.ApplicantName,
		AlumniAssociation: req.AlumniAssociation,
		Year:              req.Year,
		Major:             req.Major,
		Reason:            req.Reason,
		Phone:             req.Phone,
		TotalSlots:        len(mergedSlots),
		Status:            StatusPending,
	}

	// 构建时段记录（使用合并后的时段）
	slotRecords := make([]ReservationSlot, len(mergedSlots))
	for i, slot := range mergedSlots {
		slotRecords[i] = ReservationSlot{
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
			Status:    StatusPending,
		}
	}

	// 原子化创建订单+冲突检测（事务内行锁，防止并发双重预约）
	if err := s.repo.CreateOrderWithLock(order, slotRecords); err != nil {
		log.Printf("[error][service/Submit] 创建订单失败: %v", err)
		return nil, fmt.Errorf("创建预约失败: %v", err)
	}

	return order, nil
}

// GetOrderByID 根据订单ID查询（预加载时段），供Handler在创建后重新查询
func (s *ReservationService) GetOrderByID(id uint) (*ReservationOrder, error) {
	return s.repo.FindByOrderID(id)
}

// GetMyReservations 获取用户的预约列表（含时段明细）
func (s *ReservationService) GetMyReservations(openid string) ([]*ReservationOrder, error) {
	return s.repo.FindByOpenID(openid)
}

// GetOccupiedSlots 获取指定日期的已占用时间段
func (s *ReservationService) GetOccupiedSlots(date string) ([]TimeSlotResp, error) {
	day, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return nil, fmt.Errorf("日期格式错误")
	}

	startOfDay := day
	endOfDay := day.Add(24 * time.Hour)

	slots, err := s.repo.FindSlotsByTimeRange(startOfDay, endOfDay)
	if err != nil {
		log.Printf("[error][service/GetOccupiedSlots] 查询占用时段失败: %v", err)
		return nil, err
	}

	result := make([]TimeSlotResp, 0, len(slots))
	for _, slot := range slots {
		status := "pending"
		if slot.Status == StatusApproved {
			status = "approved"
		}
		result = append(result, TimeSlotResp{
			StartTime: slot.StartTime.Format("2006-01-02 15:04"),
			EndTime:   slot.EndTime.Format("2006-01-02 15:04"),
			Status:    status,
		})
	}

	return result, nil
}

// Cancel 取消整个订单（按orderID）
func (s *ReservationService) Cancel(orderID uint, openid string) error {
	order, err := s.repo.FindByOrderID(orderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("预约不存在")
		}
		log.Printf("[error][service/Cancel] 查询订单失败: %v", err)
		return err
	}

	if order.OpenID != openid {
		return fmt.Errorf("无权操作此预约")
	}

	if order.Status != StatusPending && order.Status != StatusApproved {
		return fmt.Errorf("当前状态无法取消")
	}

	return s.repo.CancelOrder(orderID, openid)
}

// generateOrderNo 生成订单号（含随机熵，防止高并发碰撞）
func generateOrderNo() string {
	return fmt.Sprintf("R%s%04x", time.Now().Format("20060102150405"), rand.Uint32()%0xFFFF)
}

// mergeContinuousSlots 将同一天的连续时间段合并为一条记录
// 合并规则：
//   - 按 StartTime 升序排列
//   - 如果两个时段在同一天且前一个的 EndTime == 后一个的 StartTime（首尾相接），则合并
//   - 例如: [09:00-10:00, 10:00-11:00] → [09:00-11:00]
func mergeContinuousSlots(slots []ParsedSlot) []ParsedSlot {
	if len(slots) <= 1 {
		return slots
	}

	// 按开始时间升序排序
	sorted := make([]ParsedSlot, len(slots))
	copy(sorted, slots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Before(sorted[j].StartTime)
	})

	// 逐个合并连续时段
	merged := []ParsedSlot{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		last := &merged[len(merged)-1]
		curr := sorted[i]

		if isSameDay(last.StartTime, curr.StartTime) && last.EndTime.Equal(curr.StartTime) {
			// 同一天且首尾相接，合并为一段
			last.EndTime = curr.EndTime
			log.Printf("[info][mergeContinuousSlots] 合并时段: ...%s ~ %s + ...%s ~ %s → ...%s ~ %s",
				last.StartTime.Format("15:04"), last.StartTime.Format("15:04"),
				curr.StartTime.Format("15:04"), curr.EndTime.Format("15:04"),
				last.StartTime.Format("15:04"), last.EndTime.Format("15:04"))
		} else {
			merged = append(merged, curr)
		}
	}

	return merged
}

// isSameDay 判断两个时间是否在同一天
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
