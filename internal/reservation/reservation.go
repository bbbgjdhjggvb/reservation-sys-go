package reservation

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	"reservation-sys/internal/event"
	"reservation-sys/internal/model"
	"reservation-sys/internal/reservation/store"

	"gorm.io/gorm"
)

// ========== 领域类型 ==========

// ParsedSlot 解析后的时间段
type ParsedSlot struct {
	StartTime time.Time
	EndTime   time.Time
}

// ========== Service ==========

// Service 预约业务服务
type Service struct {
	orders *store.OrderStore
	slots  *store.SlotStore
	bus    *event.Bus
}

// NewService 创建预约服务实例
func NewService(orders *store.OrderStore, slots *store.SlotStore, bus *event.Bus) *Service {
	return &Service{orders: orders, slots: slots, bus: bus}
}

// Submit 提交预约申请。
func (s *Service) Submit(openid string, slots []ParsedSlot, req *SubmitReq) (*model.ReservationOrder, error) {
	if len(slots) == 0 || len(slots) > 4 {
		return nil, fmt.Errorf("预约时段数量必须在1~4之间")
	}

	mergedSlots := mergeContinuousSlots(slots)
	if len(mergedSlots) == 0 {
		return nil, fmt.Errorf("合并后无有效时段")
	}
	log.Printf("[info][reservation/service] 原始时段数=%d, 合并后=%d", len(slots), len(mergedSlots))

	orderNo := generateOrderNo()

	order := &model.ReservationOrder{
		OrderNo:           orderNo,
		OpenID:            openid,
		ApplicantName:     req.ApplicantName,
		AlumniAssociation: req.AlumniAssociation,
		Year:              req.Year,
		Major:             req.Major,
		Reason:            req.Reason,
		Phone:             req.Phone,
		AttendeeCount:     req.AttendeeCount,
		TotalSlots:        len(mergedSlots),
		Status:            model.StatusPendingLevel1,
	}

	slotRecords := make([]model.ReservationSlot, len(mergedSlots))
	for i, slot := range mergedSlots {
		slotRecords[i] = model.ReservationSlot{
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
			Status:    model.StatusPendingLevel1,
		}
	}

	if err := s.orders.CreateOrderWithLock(order, slotRecords); err != nil {
		log.Printf("[error][reservation/service] 创建订单失败: %v", err)
		return nil, fmt.Errorf("创建预约失败: %v", err)
	}

	// 发布事件到进程内总线
	s.bus.Publish(event.OrderEvent{
		Type:      event.OrderCreated,
		OrderID:   order.ID,
		Timestamp: time.Now().Unix(),
	})

	return order, nil
}

// GetOrderByID 根据订单ID查询订单详情。
func (s *Service) GetOrderByID(id uint) (*model.ReservationOrder, error) {
	return s.orders.FindOrderByID(id)
}

// GetMyReservations 获取用户的预约列表。
func (s *Service) GetMyReservations(openid string) ([]*model.ReservationOrder, error) {
	return s.orders.FindOrdersByOpenID(openid)
}

// GetOccupiedSlots 获取指定日期的已占用时间段，并标记是否属于当前用户。
func (s *Service) GetOccupiedSlots(date string, openid string) ([]TimeSlotResp, error) {
	day, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return nil, fmt.Errorf("日期格式错误")
	}

	startOfDay := day
	endOfDay := day.Add(24 * time.Hour)

	slots, err := s.slots.FindSlotsWithOpenIDByTimeRange(startOfDay, endOfDay)
	if err != nil {
		log.Printf("[error][reservation/service] 查询占用时段失败: %v", err)
		return nil, err
	}

	result := make([]TimeSlotResp, 0, len(slots))
	for _, slot := range slots {
		status := "pending"
		if slot.Status == model.StatusApproved {
			status = "approved"
		}

		isMine := openid != "" && slot.OpenID == openid

		result = append(result, TimeSlotResp{
			StartTime: slot.StartTime.Format("2006-01-02 15:04"),
			EndTime:   slot.EndTime.Format("2006-01-02 15:04"),
			Status:    status,
			IsMine:    isMine,
		})
	}

	return result, nil
}

// Cancel 取消预约订单。
func (s *Service) Cancel(orderID uint, openid string) error {
	order, err := s.orders.FindOrderByID(orderID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("预约不存在")
		}
		log.Printf("[error][reservation/service] 查询订单失败: %v", err)
		return err
	}

	if order.OpenID != openid {
		return fmt.Errorf("无权操作此预约")
	}

	if order.Status != model.StatusPendingLevel1 &&
		order.Status != model.StatusPendingLevel2 &&
		order.Status != model.StatusApproved {
		return fmt.Errorf("当前状态无法取消")
	}

	if order.Status == model.StatusApproved {
		if len(order.Slots) == 0 {
			return fmt.Errorf("订单无时段信息")
		}
		earliestStart := order.Slots[0].StartTime
		for _, slot := range order.Slots[1:] {
			if slot.StartTime.Before(earliestStart) {
				earliestStart = slot.StartTime
			}
		}
		if time.Until(earliestStart) < 24*time.Hour {
			return fmt.Errorf("距预约开始不足24小时，无法取消")
		}
	}

	if err := s.orders.CancelOrder(orderID, openid); err != nil {
		return err
	}

	s.bus.Publish(event.OrderEvent{
		Type:      event.OrderCancelled,
		OrderID:   orderID,
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// ========== 辅助函数 ==========

// generateOrderNo 生成订单号，格式: R{时间戳14位}{4位随机hex}
func generateOrderNo() string {
	return fmt.Sprintf("R%s%04x", time.Now().Format("20060102150405"), rand.Uint32()%0xFFFF)
}

// mergeContinuousSlots 将同一天连续时间段合并。
func mergeContinuousSlots(slots []ParsedSlot) []ParsedSlot {
	if len(slots) <= 1 {
		return slots
	}

	sorted := make([]ParsedSlot, len(slots))
	copy(sorted, slots)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Before(sorted[j].StartTime)
	})

	merged := []ParsedSlot{sorted[0]}
	for i := 1; i < len(sorted); i++ {
		last := &merged[len(merged)-1]
		curr := sorted[i]
		if isSameDay(last.StartTime, curr.StartTime) && last.EndTime.Equal(curr.StartTime) {
			last.EndTime = curr.EndTime
		} else {
			merged = append(merged, curr)
		}
	}
	return merged
}

// isSameDay 判断两个时间是否在同一天。
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
