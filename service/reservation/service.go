package reservation

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	reservationdb "reservation-sys/pkg/reservationdb"

	"gorm.io/gorm"
)

// TimeSlot 时间段结构，用于前端展示已占用时段。
type TimeSlot struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

// ParsedSlot 解析后的时间段（从请求中解析并验证后的结构）。
type ParsedSlot struct {
	StartTime time.Time
	EndTime   time.Time
}

// ReservationService 预约业务服务
type ReservationService struct {
	repo reservationdb.Repository
}

// NewReservationService 创建预约服务实例。
//
// 参数:
//   - repo: 预约数据库仓库接口（操作 home_res 库）
//
// 返回值:
//   - *ReservationService: 预约服务实例
func NewReservationService(repo reservationdb.Repository) *ReservationService {
	return &ReservationService{repo: repo}
}

// Submit 批量提交预约申请。
//
// 流程:
//  1. 校验时段数量（1~4个）
//  2. 合并同一天连续的时段
//  3. 生成订单号
//  4. 在事务内创建订单+时段（行锁防并发）
//
// 参数:
//   - openid: 微信用户唯一标识
//   - slots: 已解析的时段列表
//   - req: 提交请求（含申请人信息）
//
// 返回值:
//   - *reservationdb.ReservationOrder: 创建成功的订单实体（含 ID）
//   - error: 时段数量不合法、时段冲突、创建失败时返回错误
func (s *ReservationService) Submit(openid string, slots []ParsedSlot, req *SubmitReq) (*reservationdb.ReservationOrder, error) {
	if len(slots) == 0 || len(slots) > 4 {
		return nil, fmt.Errorf("预约时段数量必须在1~4之间")
	}

	mergedSlots := mergeContinuousSlots(slots)
	if len(mergedSlots) == 0 {
		return nil, fmt.Errorf("合并后无有效时段")
	}
	log.Printf("[info][service/Submit] 原始时段数=%d, 合并后=%d", len(slots), len(mergedSlots))

	orderNo := generateOrderNo()

	order := &reservationdb.ReservationOrder{
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
		Status:            reservationdb.StatusPendingLevel1,
	}

	slotRecords := make([]reservationdb.ReservationSlot, len(mergedSlots))
	for i, slot := range mergedSlots {
		slotRecords[i] = reservationdb.ReservationSlot{
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
			Status:    reservationdb.StatusPendingLevel1,
		}
	}

	if err := s.repo.CreateOrderWithLock(order, slotRecords); err != nil {
		log.Printf("[error][service/Submit] 创建订单失败: %v", err)
		return nil, fmt.Errorf("创建预约失败: %v", err)
	}

	return order, nil
}

// GetOrderByID 根据订单ID查询订单详情（预加载时段）。
//
// 参数:
//   - id: 订单主键ID
//
// 返回值:
//   - *reservationdb.ReservationOrder: 订单实体（含时段）
//   - error: 未找到时返回 gorm.ErrRecordNotFound
func (s *ReservationService) GetOrderByID(id uint) (*reservationdb.ReservationOrder, error) {
	return s.repo.FindOrderByID(id)
}

// GetMyReservations 获取用户的预约列表（按创建时间倒序）。
//
// 参数:
//   - openid: 微信用户唯一标识
//
// 返回值:
//   - []*reservationdb.ReservationOrder: 该用户的订单列表
//   - error: 查询失败时返回错误
func (s *ReservationService) GetMyReservations(openid string) ([]*reservationdb.ReservationOrder, error) {
	return s.repo.FindOrdersByOpenID(openid)
}

// GetOccupiedSlots 获取指定日期的已占用时间段（供前端日历展示）。
//
// 参数:
//   - date: 日期字符串，格式 "2006-01-02"，为空时默认当天
//
// 返回值:
//   - []TimeSlotResp: 已占用时段列表（含起止时间和状态）
//   - error: 日期格式错误或查询失败时返回错误
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
		if slot.Status == reservationdb.StatusApproved {
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

// Cancel 取消预约订单（校验归属和状态后事务内更新订单+时段为已取消）。
//
// 参数:
//   - orderID: 订单ID
//   - openid: 用户 openid（校验归属，防止越权取消）
//
// 返回值:
//   - error: 订单不存在、无权操作、状态不允许取消时返回错误
func (s *ReservationService) Cancel(orderID uint, openid string) error {
	order, err := s.repo.FindOrderByID(orderID)
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

	if order.Status != reservationdb.StatusPendingLevel1 {
		return fmt.Errorf("当前状态无法取消")
	}

	return s.repo.CancelOrder(orderID, openid)
}

// generateOrderNo 生成订单号，格式: R{时间戳14位}{4位随机hex}。
// 示例: R20260504103000a1b2
func generateOrderNo() string {
	return fmt.Sprintf("R%s%04x", time.Now().Format("20060102150405"), rand.Uint32()%0xFFFF)
}

// mergeContinuousSlots 将同一天的连续时间段合并为一条记录。
//
// 合并条件: 同一天内，前一时段的结束时间 == 后一时段的开始时间。
// 合并后延长前一时段的 EndTime，丢弃后一时段。
//
// 参数:
//   - slots: 原始时段列表（无需预排序，方法内部会按 StartTime 排序）
//
// 返回值:
//   - []ParsedSlot: 合并后的时段列表
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
			log.Printf("[info][mergeContinuousSlots] 合并时段: ...%s ~ %s + ...%s ~ %s → ...%s ~ %s",
				last.StartTime.Format("15:04"), last.EndTime.Format("15:04"),
				curr.StartTime.Format("15:04"), curr.EndTime.Format("15:04"),
				last.StartTime.Format("15:04"), last.EndTime.Format("15:04"))
		} else {
			merged = append(merged, curr)
		}
	}

	return merged
}

// isSameDay 判断两个时间是否在同一天（同一时区）。
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
