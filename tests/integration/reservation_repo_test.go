package integration

import (
	"fmt"
	"testing"
	"time"

	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParseTime(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	return t
}

// TestCreateOrderWithLock_Success 验证事务内创建订单+时段
func TestCreateOrderWithLock_Success(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	order := &reservationdb.ReservationOrder{
		OrderNo:           "R202605011000000001",
		OpenID:            "test_openid",
		ApplicantName:     "张三",
		AlumniAssociation: "计算机与软件学院校友会",
		Year:              2020,
		Major:             "软件工程",
		Reason:            "测试",
		Phone:             "13800138000",
		TotalSlots:        1,
		Status:            reservationdb.StatusPending,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-01 08:00:00"), EndTime: mustParseTime("2026-05-01 10:00:00"), Status: reservationdb.StatusPending},
	}

	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)
	assert.NotZero(t, order.ID, "订单ID应由数据库生成")
	assert.Equal(t, order.ID, slots[0].OrderID, "时段OrderID应被自动填充")

	// 验证可查询到
	found, err := repo.FindOrderByID(order.ID)
	require.NoError(t, err)
	assert.Equal(t, "R202605011000000001", found.OrderNo)
	assert.Len(t, found.Slots, 1)
}

// TestCreateOrderWithLock_Conflict 验证时段冲突检测
func TestCreateOrderWithLock_Conflict(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	// 先创建一个已通过的预约占用 08:00-10:00
	order1 := &reservationdb.ReservationOrder{
		OrderNo: "R001", OpenID: "u1", ApplicantName: "A",
		AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13800138000",
		TotalSlots: 1, Status: reservationdb.StatusApproved,
	}
	slots1 := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-01 08:00:00"), EndTime: mustParseTime("2026-05-01 10:00:00"), Status: reservationdb.StatusApproved},
	}
	err := repo.CreateOrderWithLock(order1, slots1)
	require.NoError(t, err)

	// 再尝试预约重叠时段，应检测到冲突
	order2 := &reservationdb.ReservationOrder{
		OrderNo: "R002", OpenID: "u2", ApplicantName: "B",
		AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13900139000",
		TotalSlots: 1, Status: reservationdb.StatusPending,
	}
	slots2 := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-01 09:00:00"), EndTime: mustParseTime("2026-05-01 11:00:00"), Status: reservationdb.StatusPending},
	}
	err = repo.CreateOrderWithLock(order2, slots2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "已被预约")
}

// TestFindOrderByID 验证预加载时段
func TestFindOrderByID(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	order := &reservationdb.ReservationOrder{
		OrderNo: "R003", OpenID: "u1", ApplicantName: "测试",
		AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13800138000",
		TotalSlots: 2, Status: reservationdb.StatusPending,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-02 08:00:00"), EndTime: mustParseTime("2026-05-02 10:00:00"), Status: reservationdb.StatusPending},
		{StartTime: mustParseTime("2026-05-02 13:00:00"), EndTime: mustParseTime("2026-05-02 15:00:00"), Status: reservationdb.StatusPending},
	}
	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)

	found, err := repo.FindOrderByID(order.ID)
	require.NoError(t, err)
	assert.Len(t, found.Slots, 2)

	// 不存在的订单
	_, err = repo.FindOrderByID(99999)
	assert.Error(t, err)
}

// TestUpdateOrderStatus_OptimisticLock 验证乐观锁机制
func TestUpdateOrderStatus_OptimisticLock(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	order := &reservationdb.ReservationOrder{
		OrderNo: "R004", OpenID: "u1", ApplicantName: "测试",
		AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13800138000",
		TotalSlots: 1, Status: reservationdb.StatusPending,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-03 08:00:00"), EndTime: mustParseTime("2026-05-03 10:00:00"), Status: reservationdb.StatusPending},
	}
	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)

	// 第一次更新应成功：Pending(0) → Approved(1)
	err = repo.UpdateOrderStatus(order.ID, reservationdb.StatusPending, reservationdb.StatusApproved)
	assert.NoError(t, err)

	// 第二次用相同的 fromStatus 应失败（乐观锁），因为状态已经变成 Approved
	err = repo.UpdateOrderStatus(order.ID, reservationdb.StatusPending, reservationdb.StatusApproved)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "状态不匹配")

	// 验证订单状态确实已更新
	found, _ := repo.FindOrderByID(order.ID)
	assert.Equal(t, reservationdb.StatusApproved, found.Status)
}

// TestCancelOrder 验证取消订单同时更新订单+时段状态
func TestCancelOrder(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	order := &reservationdb.ReservationOrder{
		OrderNo: "R005", OpenID: "u1", ApplicantName: "测试",
		AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13800138000",
		TotalSlots: 1, Status: reservationdb.StatusPending,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-04 08:00:00"), EndTime: mustParseTime("2026-05-04 10:00:00"), Status: reservationdb.StatusPending},
	}
	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)

	err = repo.CancelOrder(order.ID, "u1")
	require.NoError(t, err)

	// 验证
	found, _ := repo.FindOrderByID(order.ID)
	assert.Equal(t, reservationdb.StatusCancelled, found.Status)
	assert.Equal(t, reservationdb.StatusCancelled, found.Slots[0].Status)
}

// TestCancelOrder_WrongUser 验证非本人无法取消
func TestCancelOrder_WrongUser(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	order := &reservationdb.ReservationOrder{
		OrderNo: "R006", OpenID: "u1", ApplicantName: "测试",
		AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13800138000",
		TotalSlots: 1, Status: reservationdb.StatusPending,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-05 08:00:00"), EndTime: mustParseTime("2026-05-05 10:00:00"), Status: reservationdb.StatusPending},
	}
	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)

	err = repo.CancelOrder(order.ID, "other_user")
	assert.Error(t, err, "非本人取消应失败")
}

// TestSetSlotPassword 验证仅为已通过时段设置密码
func TestSetSlotPassword(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	order := &reservationdb.ReservationOrder{
		OrderNo: "R007", OpenID: "u1", ApplicantName: "测试",
		AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13800138000",
		TotalSlots: 1, Status: reservationdb.StatusPending,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-06 08:00:00"), EndTime: mustParseTime("2026-05-06 10:00:00"), Status: reservationdb.StatusPending},
	}
	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)

	// 时段状态为 Pending(0)，设置密码应失败
	err = repo.SetSlotPassword(slots[0].ID, "123456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不允许设置密码")

	// 更新为已通过
	err = repo.UpdateOrderStatus(order.ID, reservationdb.StatusPending, reservationdb.StatusApproved)
	require.NoError(t, err)

	// 现在设置密码应成功
	err = repo.SetSlotPassword(slots[0].ID, "123456")
	assert.NoError(t, err)

	// 验证密码
	found, _ := repo.FindOrderByID(order.ID)
	assert.Equal(t, "123456", found.Slots[0].Password)
}

// TestFindSlotsByTimeRange 验证时间段交集查询
func TestFindSlotsByTimeRange(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	// 创建 08:00-10:00 的已通过预约
	order := &reservationdb.ReservationOrder{
		OrderNo: "R008", OpenID: "u1", ApplicantName: "测试",
		AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13800138000",
		TotalSlots: 1, Status: reservationdb.StatusApproved,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-07 08:00:00"), EndTime: mustParseTime("2026-05-07 10:00:00"), Status: reservationdb.StatusApproved},
	}
	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)

	// 查询 07:00-09:00 的交集，应命中
	result, err := repo.FindSlotsByTimeRange(
		mustParseTime("2026-05-07 07:00:00"),
		mustParseTime("2026-05-07 09:00:00"),
	)
	require.NoError(t, err)
	assert.Len(t, result, 1, "有交集的时段应被返回")

	// 查询完全不重叠的范围，不应命中
	result, err = repo.FindSlotsByTimeRange(
		mustParseTime("2026-05-07 10:00:00"),
		mustParseTime("2026-05-07 12:00:00"),
	)
	require.NoError(t, err)
	assert.Empty(t, result, "不重叠的时段不应返回")
}

// TestCreateReviewRecord 验证审核记录创建
func TestCreateReviewRecord(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	order := &reservationdb.ReservationOrder{
		OrderNo: "R009", OpenID: "u1", ApplicantName: "测试",
		AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13800138000",
		TotalSlots: 1, Status: reservationdb.StatusPending,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: mustParseTime("2026-05-08 08:00:00"), EndTime: mustParseTime("2026-05-08 10:00:00"), Status: reservationdb.StatusPending},
	}
	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)

	record := &reservationdb.ReviewRecord{
		OrderID:      order.ID,
		ReviewerID:   1,
		ReviewerRole: 1,
		Action:       1,
		Comment:      "审核通过",
	}
	err = repo.CreateReviewRecord(record)
	require.NoError(t, err)
	assert.NotZero(t, record.ID)

	// 验证查询
	records, err := repo.FindReviewRecordsByOrderID(order.ID)
	require.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, "审核通过", records[0].Comment)
}

// TestListOrders 验证分页查询
func TestListOrders(t *testing.T) {
	repo, cleanup := setupRepo(t)
	defer cleanup()

	// 创建 3 个不同状态的订单
	for i := 0; i < 3; i++ {
		order := &reservationdb.ReservationOrder{
			OrderNo: fmt.Sprintf("R010_%d", i), OpenID: "u1", ApplicantName: "测试",
			AlumniAssociation: "校友会", Year: 2020, Major: "CS", Reason: "t", Phone: "13800138000",
			TotalSlots: 1, Status: i,
		}
		slots := []reservationdb.ReservationSlot{
			{StartTime: mustParseTime(fmt.Sprintf("2026-05-09 %02d:00:00", 8+i*4)), EndTime: mustParseTime(fmt.Sprintf("2026-05-09 %02d:00:00", 10+i*4)), Status: i},
		}
		err := repo.CreateOrderWithLock(order, slots)
		require.NoError(t, err)
	}

	// 查询所有
	orders, total, err := repo.ListOrders(nil, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, orders, 3)

	// 按状态筛选
	orders, total, err = repo.ListOrders([]int{0}, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, orders, 1)

	// 分页
	orders, total, err = repo.ListOrders(nil, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, orders, 1)
}
