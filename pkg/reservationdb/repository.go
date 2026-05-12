package reservationdb

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository 预约+审核数据库的统一数据访问接口。
// 涵盖 reservation 和 admin 服务所需的所有 home_res 数据库操作，
// 确保数据操作逻辑只在此处实现，避免散落在多个服务中。
type Repository interface {
	// --- 订单操作 ---

	// CreateOrderWithLock 在事务内创建订单并锁定时段，防止并发双重预约。
	// 参数:
	//   - order: 订单实体（ID 由数据库生成）
	//   - slots: 时段切片（OrderID 在方法内自动填充）
	// 返回值:
	//   - error: 时段冲突时返回 "第N个时间段已被预约"，创建失败返回包装错误
	CreateOrderWithLock(order *ReservationOrder, slots []ReservationSlot) error

	// FindOrderByID 根据订单ID查询订单详情（预加载关联时段）。
	// SQL: SELECT * FROM reservation_orders WHERE id = ?; SELECT * FROM reservation_slots WHERE order_id = ?;
	// 参数:
	//   - id: 订单主键ID
	// 返回值:
	//   - *ReservationOrder: 订单实体（含 Slots），未找到时返回 nil
	//   - error: 未找到时返回 gorm.ErrRecordNotFound
	FindOrderByID(id uint) (*ReservationOrder, error)

	// FindOrdersByOpenID 根据用户 openid 查询其所有预约订单（预加载时段，按创建时间倒序）。
	// SQL: SELECT * FROM reservation_orders WHERE open_id = ? ORDER BY created_at DESC;
	// 参数:
	//   - openid: 微信用户唯一标识
	// 返回值:
	//   - []*ReservationOrder: 订单列表，无记录时返回空切片
	FindOrdersByOpenID(openid string) ([]*ReservationOrder, error)

	// ListOrders 分页查询订单列表（支持按状态筛选，预加载时段）。
	// SQL: SELECT COUNT(*) FROM reservation_orders WHERE status IN (?);
	//      SELECT * FROM reservation_orders WHERE status IN (?) ORDER BY created_at DESC LIMIT ? OFFSET ?;
	// 参数:
	//   - statuses: 状态筛选列表，为空时查询全部
	//   - page: 页码（从1开始，<1 时自动修正为1）
	//   - pageSize: 每页条数（1~50，超出范围自动修正为20）
	// 返回值:
	//   - []*ReservationOrder: 当前页订单列表
	//   - int64: 符合条件的总记录数
	ListOrders(statuses []int, page, pageSize int) ([]*ReservationOrder, int64, error)

	// UpdateOrderStatus 审核更新订单状态（事务内同步更新订单+时段状态，乐观锁防并发）。
	// SQL: BEGIN;
	//      UPDATE reservation_orders SET status = ? WHERE id = ? AND status = ?;
	//      UPDATE reservation_slots SET status = ? WHERE order_id = ? AND status = ?;
	//      COMMIT;
	// 参数:
	//   - orderID: 订单ID
	//   - fromStatus: 期望的当前状态（乐观锁条件）
	//   - toStatus: 目标状态
	// 返回值:
	//   - error: 状态不匹配时返回 "订单状态不匹配，无法执行此操作"
	UpdateOrderStatus(orderID uint, fromStatus, toStatus int) error

	// CancelOrder 用户取消订单（事务内同时更新订单和时段状态为已取消）。
	// SQL: BEGIN;
	//      UPDATE reservation_orders SET status = 4 WHERE id = ? AND open_id = ? AND status IN (0, 1);
	//      UPDATE reservation_slots SET status = 4 WHERE order_id = ? AND status IN (0, 1);
	//      COMMIT;
	// 参数:
	//   - orderID: 订单ID
	//   - openid: 用户 openid（校验归属）
	// 返回值:
	//   - error: 订单不存在或不属于该用户时返回 gorm.ErrRecordNotFound
	CancelOrder(orderID uint, openid string) error

	// --- 时段操作 ---

	// FindSlotsByTimeRange 查询指定时间范围内有交集的已占用时段。
	// SQL: SELECT * FROM reservation_slots WHERE status IN (0, 1) AND start_time < ? AND end_time > ?;
	// 参数:
	//   - start: 范围起始时间
	//   - end: 范围结束时间
	// 返回值:
	//   - []ReservationSlot: 有交集的时段列表
	FindSlotsByTimeRange(start, end time.Time) ([]ReservationSlot, error)

	// UpdateSlotStatus 更新单个时段的状态。
	// SQL: UPDATE reservation_slots SET status = ? WHERE id = ?;
	// 参数:
	//   - slotID: 时段主键ID
	//   - status: 目标状态
	// 返回值:
	//   - error: 更新失败时返回数据库错误
	UpdateSlotStatus(slotID uint, status int) error

	// SetSlotPassword 设置已通过时段的门锁密码。
	// SQL: UPDATE reservation_slots SET password = ? WHERE id = ? AND status = 1;
	// 参数:
	//   - slotID: 时段主键ID
	//   - password: 门锁密码（明文，最大20字符）
	// 返回值:
	//   - error: 时段不存在或状态不允许时返回 "时段不存在或状态不允许设置密码"
	SetSlotPassword(slotID uint, password string) error

	// --- 审核记录操作 ---

	// CreateReviewRecord 创建审核记录。
	// SQL: INSERT INTO review_records (order_id, reviewer_id, reviewer_role, action, comment) VALUES (?, ?, ?, ?, ?);
	// 参数:
	//   - record: 审核记录实体
	// 返回值:
	//   - error: 插入失败时返回数据库错误
	CreateReviewRecord(record *ReviewRecord) error

	// FindReviewRecordsByOrderID 根据订单ID查询审核记录（按创建时间正序）。
	// SQL: SELECT * FROM review_records WHERE order_id = ? ORDER BY created_at ASC;
	// 参数:
	//   - orderID: 订单ID
	// 返回值:
	//   - []ReviewRecord: 审核记录列表，无记录时返回空切片
	FindReviewRecordsByOrderID(orderID uint) ([]ReviewRecord, error)
}

type repository struct {
	db *gorm.DB
}

// NewRepository 创建数据仓库实例。
//
// 参数:
//   - db: 已初始化的 GORM 数据库连接（指向 home_res 库）
//
// 返回值:
//   - Repository: 数据仓库接口实例
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// =============================================
// 订单操作
// =============================================

// CreateOrderWithLock 原子化创建订单（事务内行锁，防止并发双重预约）。
//
// 流程:
//  1. 逐个检测时段冲突（SELECT ... FOR UPDATE 行锁）
//  2. 创建订单记录（INSERT reservation_orders）
//  3. 批量创建时段记录（INSERT reservation_slots）
//
// SQL:
//
//	BEGIN;
//	SELECT COUNT(*) FROM reservation_slots WHERE status IN (0,1) AND start_time < ? AND end_time > ? FOR UPDATE;
//	INSERT INTO reservation_orders (...) VALUES (...);
//	INSERT INTO reservation_slots (order_id, ...) VALUES (?, ...), (?, ...), ...;
//	COMMIT;
func (r *repository) CreateOrderWithLock(order *ReservationOrder, slots []ReservationSlot) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, slot := range slots {
			var count int64
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

		if err := tx.Create(order).Error; err != nil {
			return fmt.Errorf("创建订单失败: %w", err)
		}

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

// FindOrderByID 根据订单ID查询（预加载时段）。
//
// SQL: SELECT * FROM reservation_orders WHERE id = ?;
//      SELECT * FROM reservation_slots WHERE order_id = ?;
//
// 参数:
//   - id: 订单主键ID
//
// 返回值:
//   - *ReservationOrder: 订单实体（含关联时段）
//   - error: 未找到时返回 gorm.ErrRecordNotFound
func (r *repository) FindOrderByID(id uint) (*ReservationOrder, error) {
	var order ReservationOrder
	err := r.db.Preload("Slots").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// FindOrdersByOpenID 根据用户 openid 查询预约列表（预加载时段，按创建时间倒序）。
//
// SQL: SELECT * FROM reservation_orders WHERE open_id = ? ORDER BY created_at DESC;
//      SELECT * FROM reservation_slots WHERE order_id IN (?);
//
// 参数:
//   - openid: 微信用户唯一标识
//
// 返回值:
//   - []*ReservationOrder: 订单列表
//   - error: 查询失败时返回数据库错误
func (r *repository) FindOrdersByOpenID(openid string) ([]*ReservationOrder, error) {
	var orders []*ReservationOrder
	err := r.db.Preload("Slots").
		Where("open_id = ?", openid).
		Order("created_at desc").
		Find(&orders).Error
	return orders, err
}

// ListOrders 分页查询订单列表（支持按状态筛选，预加载时段）。
//
// SQL:
//
//	SELECT COUNT(*) FROM reservation_orders WHERE status IN (?);
//	SELECT * FROM reservation_orders WHERE status IN (?) ORDER BY created_at DESC LIMIT ? OFFSET ?;
//	SELECT * FROM reservation_slots WHERE order_id IN (?);
//
// 参数:
//   - statuses: 状态筛选列表，为空时不加 WHERE 条件
//   - page: 页码（从1开始，<1 时自动修正为1）
//   - pageSize: 每页条数（1~50，超出范围自动修正为20）
//
// 返回值:
//   - []*ReservationOrder: 当前页订单列表
//   - int64: 符合条件的总记录数
//   - error: 查询失败时返回数据库错误
func (r *repository) ListOrders(statuses []int, page, pageSize int) ([]*ReservationOrder, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var orders []*ReservationOrder
	var total int64

	query := r.db.Model(&ReservationOrder{})
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	query.Count(&total)

	err := r.db.Preload("Slots").
		Scopes(func(db *gorm.DB) *gorm.DB {
			if len(statuses) > 0 {
				return db.Where("status IN ?", statuses)
			}
			return db
		}).
		Order("created_at desc").
		Offset(offset).Limit(pageSize).
		Find(&orders).Error
	if err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// UpdateOrderStatus 审核更新订单状态（事务：订单+时段状态同步，乐观锁防并发）。
//
// 乐观锁机制：WHERE status = fromStatus 确保状态在读取和更新之间未被其他操作修改，
// RowsAffected == 0 表示状态已变更，拒绝本次操作。
//
// SQL:
//
//	BEGIN;
//	UPDATE reservation_orders SET status = ?, updated_at = NOW() WHERE id = ? AND status = ?;
//	UPDATE reservation_slots SET status = ?, updated_at = NOW() WHERE order_id = ? AND status = ?;
//	COMMIT;
//
// 参数:
//   - orderID: 订单ID
//   - fromStatus: 期望的当前状态（乐观锁条件）
//   - toStatus: 目标状态
//
// 返回值:
//   - error: 状态不匹配时返回 "订单状态不匹配，无法执行此操作"
func (r *repository) UpdateOrderStatus(orderID uint, fromStatus, toStatus int) error {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&ReservationOrder{}).
			Where("id = ? AND status = ?", orderID, fromStatus).
			Update("status", toStatus)

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		tx.Model(&ReservationSlot{}).
			Where("order_id = ? AND status = ?", orderID, fromStatus).
			Update("status", toStatus)

		return nil
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("订单状态不匹配，无法执行此操作")
		}
		return fmt.Errorf("更新订单状态失败: %w", err)
	}
	return nil
}

// CancelOrder 原子化取消订单（事务内同时更新订单和时段状态）。
//
// SQL:
//
//	BEGIN;
//	UPDATE reservation_orders SET status = 4, updated_at = NOW() WHERE id = ? AND open_id = ? AND status IN (0, 1);
//	UPDATE reservation_slots SET status = 4, updated_at = NOW() WHERE order_id = ? AND status IN (0, 1);
//	COMMIT;
//
// 参数:
//   - orderID: 订单ID
//   - openid: 用户 openid（校验归属，防止越权取消）
//
// 返回值:
//   - error: 订单不存在或不属于该用户时返回 gorm.ErrRecordNotFound
func (r *repository) CancelOrder(orderID uint, openid string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
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

		slotResult := tx.Model(&ReservationSlot{}).
			Where("order_id = ? AND status IN ?", orderID, []int{StatusPending, StatusApproved}).
			Update("status", StatusCancelled)

		if slotResult.Error != nil {
			return fmt.Errorf("更新时段状态失败: %w", slotResult.Error)
		}

		return nil
	})
}

// =============================================
// 时段操作
// =============================================

// FindSlotsByTimeRange 查询指定时间范围内有交集的已占用时段。
//
// SQL: SELECT * FROM reservation_slots WHERE status IN (0, 1) AND start_time < ? AND end_time > ?;
//
// 参数:
//   - start: 范围起始时间
//   - end: 范围结束时间
//
// 返回值:
//   - []ReservationSlot: 有交集的时段列表
//   - error: 查询失败时返回数据库错误
func (r *repository) FindSlotsByTimeRange(start, end time.Time) ([]ReservationSlot, error) {
	var slots []ReservationSlot
	err := r.db.Where("status IN ?", []int{StatusPending, StatusApproved}).
		Where("start_time < ? AND end_time > ?", end, start).
		Find(&slots).Error
	return slots, err
}

// UpdateSlotStatus 更新单个时段的状态。
//
// SQL: UPDATE reservation_slots SET status = ?, updated_at = NOW() WHERE id = ?;
//
// 参数:
//   - slotID: 时段主键ID
//   - status: 目标状态
//
// 返回值:
//   - error: 更新失败时返回数据库错误
func (r *repository) UpdateSlotStatus(slotID uint, status int) error {
	return r.db.Model(&ReservationSlot{}).Where("id = ?", slotID).Update("status", status).Error
}

// SetSlotPassword 设置已通过时段的门锁密码。
//
// SQL: UPDATE reservation_slots SET password = ?, updated_at = NOW() WHERE id = ? AND status = 1;
//
// 参数:
//   - slotID: 时段主键ID
//   - password: 门锁密码（明文，最大20字符）
//
// 返回值:
//   - error: 时段不存在或状态不允许时返回 "时段不存在或状态不允许设置密码"
func (r *repository) SetSlotPassword(slotID uint, password string) error {
	result := r.db.Model(&ReservationSlot{}).
		Where("id = ? AND status = ?", slotID, StatusApproved).
		Update("password", password)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("时段不存在或状态不允许设置密码")
	}
	return nil
}

// =============================================
// 审核记录操作
// =============================================

// CreateReviewRecord 创建审核记录。
//
// SQL: INSERT INTO review_records (order_id, reviewer_id, reviewer_role, action, comment, created_at) VALUES (?, ?, ?, ?, ?, NOW());
//
// 参数:
//   - record: 审核记录实体（ID 和 CreatedAt 由数据库/自动填充）
//
// 返回值:
//   - error: 插入失败时返回数据库错误
func (r *repository) CreateReviewRecord(record *ReviewRecord) error {
	return r.db.Create(record).Error
}

// FindReviewRecordsByOrderID 根据订单ID查询审核记录（按创建时间正序）。
//
// SQL: SELECT * FROM review_records WHERE order_id = ? ORDER BY created_at ASC;
//
// 参数:
//   - orderID: 订单ID
//
// 返回值:
//   - []ReviewRecord: 审核记录列表，无记录时返回空切片
//   - error: 查询失败时返回数据库错误
func (r *repository) FindReviewRecordsByOrderID(orderID uint) ([]ReviewRecord, error) {
	var records []ReviewRecord
	err := r.db.Where("order_id = ?", orderID).
		Order("created_at asc").
		Find(&records).Error
	return records, err
}
