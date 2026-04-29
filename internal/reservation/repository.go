// internal/reservation/repository.go
package reservation

// import "fmt" needed for error formatting
import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReservationRepository 定义数据访问接口，方便微服务拆分时替换实现
type ReservationRepository interface {
	// --- 订单操作 ---
	CreateOrderWithLock(order *ReservationOrder, slots []ReservationSlot) error // 原子化：冲突检测(行锁)+创建订单+创建时段
	FindByOrderID(id uint) (*ReservationOrder, error)
	FindByOpenID(openid string) ([]*ReservationOrder, error) // 返回订单及其关联的Slots

	// --- 时段查询 ---
	FindSlotsByTimeRange(start, end time.Time) ([]ReservationSlot, error)

	// --- 状态更新 ---
	UpdateSlotStatus(slotID uint, status int) error
	CancelOrder(orderID uint, openid string) error // 原子化：事务内取消订单+所有时段

	// --- 兼容旧接口 ---
	FindByID(id uint) (*ReservationOrder, error)
}

// reservationRepo 实现 ReservationRepository 接口
type reservationRepo struct {
	db *gorm.DB
}

// NewReservationRepository 创建仓库实例
func NewReservationRepository(db *gorm.DB) ReservationRepository {
	return &reservationRepo{db: db}
}

// CreateOrderWithLock 原子化创建订单，解决 Check-Then-Act 数据竞争问题
func (r *reservationRepo) CreateOrderWithLock(order *ReservationOrder, slots []ReservationSlot) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, slot := range slots {
			var count int64
			/* 下面代码对应的 sql
			 * SELECT COUNT(*) FROM reservation_slots FOR UPDATE
			 * WHERE status IN (1,2)
			 * AND start_time < slot.EndTime
			 * AND end_time > slot.StartTime

			 * 这是一个事务内，REPEATABLE READ 隔离级别的 Current Read 查询
			 * 如果不加 FOR UPDATE，InnoDB 在进行查询使用的是快照读，不会加X锁
			 * X锁是读写锁，锁住后，其他事务不能进行读，也不能进行写
			 */
			err := tx.Model(&ReservationSlot{}).
				Where("status IN ?", []int{StatusPending, StatusApproved}).
				Where("start_time < ? AND end_time > ?", slot.EndTime, slot.StartTime).
				Clauses(clause.Locking{Strength: "UPDATE"}).
				Count(&count).Error
			if err != nil {
				return fmt.Errorf("检测第%d个时段冲突失败: %w", i+1, err)
			}
			if count > 0 {
				return fmt.Errorf("第%d个时间段已被预约", i+1)
			}
		}

		// 创建订单
		// 插入数据 InnoDB 会自动添加排他锁(Record Lock(X) + Insert Intention Lock)
		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("创建订单失败: %w", err)
		}

		// 批量创建时段记录
		for i := range slots {
			slots[i].OrderID = order.ID
		}
		if len(slots) > 0 {
			if err := tx.Create(&slots).Error; err != nil {
				return fmt.Errorf("创建时段失败: %w", err)
			}
		}

		return nil
	})
}

// FindByOrderID 根据订单ID查询（预加载时段）
func (r *reservationRepo) FindByOrderID(id uint) (*ReservationOrder, error) {
	var order ReservationOrder
	err := r.db.Preload("Slots").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// FindByID 向后兼容别名
func (r *reservationRepo) FindByID(id uint) (*ReservationOrder, error) {
	return r.FindByOrderID(id)
}

// FindByOpenID 根据用户openid查询预约列表（预加载时段，按创建时间倒序）
func (r *reservationRepo) FindByOpenID(openid string) ([]*ReservationOrder, error) {
	var orders []*ReservationOrder
	err := r.db.Preload("Slots").
		Where("open_id = ?", openid).
		Order("created_at desc").
		Find(&orders).Error
	return orders, err
}

// FindSlotsByTimeRange 查询指定时间范围内有交集的已占用时段
// 用于检查时间段冲突
func (r *reservationRepo) FindSlotsByTimeRange(start, end time.Time) ([]ReservationSlot, error) {
	var slots []ReservationSlot
	err := r.db.Where("status IN ?", []int{StatusPending, StatusApproved}).
		Where("start_time < ? AND end_time > ?", end, start).
		Find(&slots).Error
	return slots, err
}

// UpdateSlotStatus 更新单个时段的状态
func (r *reservationRepo) UpdateSlotStatus(slotID uint, status int) error {
	return r.db.Model(&ReservationSlot{}).Where("id = ?", slotID).Update("status", status).Error
}

// CancelOrder 原子化取消整个订单（事务内同时更新订单和所有时段状态）
func (r *reservationRepo) CancelOrder(orderID uint, openid string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 更新订单状态
		result := tx.Model(&ReservationOrder{}).
			Where("id = ? AND open_id = ? AND status IN ?",
				orderID, openid, []int{StatusPending, StatusApproved}).
			Update("status", StatusCancelled)

		if result.Error != nil {
			return fmt.Errorf("更新订单状态失败: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		// 更新该订单下所有待审核/已通过的时段状态
		slotResult := tx.Model(&ReservationSlot{}).
			Where("order_id = ? AND status IN ?", orderID, []int{StatusPending, StatusApproved}).
			Update("status", StatusCancelled)

		if slotResult.Error != nil {
			return fmt.Errorf("更新时段状态失败: %w", slotResult.Error)
		}

		return nil
	})
}
