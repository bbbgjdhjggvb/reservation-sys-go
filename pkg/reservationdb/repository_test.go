package reservationdb

import (
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// GORM 和 go-sqlmock结合：生成一个不连接真实数据库的虚拟 *sql.DB 连接，然后强行作为
// gorm 初始化的参数，让 gorm 误认为自己真的连接了真的数据库。
// 当 gorm 执行 SQL 时，会把 SQL 语句发给 sqlmock。
func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	// db 类型为 sql*DB，是 go 语言标准库中的数据库连接对象。
	//
	// mock 类型为 sqlmock.Sqlmock，是一个测试控制器，可以在测试用例中使用它
	// 指定“期望收到的SQL”以及"返回的模拟数据"
	//
	// sqlmock 默认使用 正则表达式模糊匹配器，而不是 全文本精确匹配。
	// 只要 gorm 生成的 SQL 语句包含 mock.ExpectQuery() 中写的正则片段，测试就通过。
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	// 通常，我们在初始化 GORM 时会传入一个 DNS(如root:pass@tcp(127.0.0.1:3306)/db) 。
	// db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})。
	//
	// mysql.New 代替了 mysql.Open，他可以直接指定虚拟的 db。
	// Conn 参数：告诉 mysql 不用自己去连接，而是直接使用虚拟 db。
	// SkipInitializeWithVersion 参数：告诉 mysql 不用自己去检查数据库版本，
	// 因为 Gorm 启动的时候，默认会自动向数据库发送一条 SELECT VERSION() 的SQL，
	// 但是没有实际的数据库，如果不跳过，肯定会报错。
	dialector := mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	return gormDB, mock
}

// 在进行数据库测试时，我们通常需要测试包含时间段（如 create_at，updated_at）
// 的 SQL 操作，这些时间段通常是由系统调用 time.Now() 动态生成的。
// 由于测试的时候，时间一直在流动，计算机执行指令有时间差，所以不能准确地比较。
//
// 为了解决上面的问题，go-sqlmock 提供了一个 argument 接口：
//
// type Argument interface {
// 		Match(driver.Value) bool
// }
//
// 任何实现了 Match 方法的结构体，都可以作为参数传入 .WithArgs() 中
//
// 由于时间是系统调用产生，精确度高，所以只进行类型断言。确保传入的参数为
// time.Time 类型

type anyTime struct{}

func (a anyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

// model.go 模型测试

// 测试 model.go 文件中 func StatusText(code int) string 函数
// 函数功能：将定义的常量转换为中文语言

func TestStatusText(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{StatusPendingLevel1, "等待一级审核"},
		{StatusPendingLevel2, "等待二级审核"},
		{StatusRejectedLevel1, "一级审核拒绝"},
		{StatusRejectedLevel2, "二级审核拒绝"},
		{StatusApproved, "审核通过"},
		{StatusCancelled, "订单已经取消"},
		{StatusCompleted, "订单已经完成"},
		{-1, "未知状态"},
		{99, "未知状态"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d_%s", tt.code, tt.expected), func(t *testing.T) {
			assert.Equal(t, tt.expected, StatusText(tt.code))
		})
	}
}

// 测试结构体 ReservationOrder TableName() 函数
func TestReservationOrder_TableName(t *testing.T) {
	assert.Equal(t, "reservation_orders", ReservationOrder{}.TableName())
}

// 测试结构体 ReservationSlot TableName() 函数
func TestReservationSlot_TableName(t *testing.T) {
	assert.Equal(t, "reservation_slots", ReservationSlot{}.TableName())
}

// 测试结构体 ReviewRecord TableName() 函数
func TestReviewRecord_TableName(t *testing.T) {
	assert.Equal(t, "review_records", ReviewRecord{}.TableName())
}

// repository.go 函数测试

func TestNewRepository(t *testing.T) {
	gormDB, _ := newMockDB(t)
	repo := NewRepository(gormDB)
	assert.NotNil(t, repo)

	// 用来断言一个对象是否实现了接口
	assert.Implements(t, (*Repository)(nil), repo)
}

// 测试 repository.go 文件中的 func FindOrderByID(id uint) (*ReservationOrder, error)
//
// 函数功能：根据订单的 ID 进行查询
//
// 测试场景：
// 1. 顶端存在且正确加载时间段
//  1. SQL 语句是否是
//     - SELECT * FROM `reservation_orders` WHERE `reservation_orders`.`id` = ? ORDER BY `reservation_orders`.`id` LIMIT ?
//     - SELECT * FROM `reservation_slots` WHERE `reservation_slots`.`order_id` = ?
//  2. 测试 order 对象是否正确加载上 slots[] 时间段切片
//
// 2. 测试订单不存在时，是否有返回 gorm.ErrRecordNotFound 错误
func TestFindOrderByID(t *testing.T) {
	t.Run("订单存在_含预加载时段", func(t *testing.T) {
		// 调用 newMockDB 函数，将 sqlmock 替换 gorm
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		// 假设数据库中有如下数据
		orderRows := sqlmock.NewRows([]string{"id", "order_no", "open_id", "applicant_name",
			"alumni_association", "year", "major", "reason", "phone", "total_slots", "status",
			"created_at", "updated_at"}).
			AddRow(1, "R20260325001", "openid_001", "张三", "校友会", 2020, "CS", "测试",
				"13800138000", 1, StatusPendingLevel1, time.Now(), time.Now())

		mock.ExpectQuery("SELECT \\* FROM `reservation_orders` WHERE `reservation_orders`.`id` = \\? ORDER BY `reservation_orders`.`id` LIMIT \\?").
			WithArgs(1, 1).
			WillReturnRows(orderRows)

		slotRows := sqlmock.NewRows([]string{"id", "order_id", "start_time", "end_time", "status", "password", "created_at", "updated_at"}).
			AddRow(1, 1, time.Now(), time.Now(), StatusPendingLevel1, "", time.Now(), time.Now()).
			AddRow(2, 1, time.Now(), time.Now(), StatusPendingLevel1, "", time.Now(), time.Now())
		mock.ExpectQuery("SELECT \\* FROM `reservation_slots` WHERE `reservation_slots`.`order_id` = \\?").
			WithArgs(1).
			WillReturnRows(slotRows)

		order, err := repo.FindOrderByID(1)
		require.NoError(t, err)
		assert.Equal(t, uint(1), order.ID)
		assert.Equal(t, "张三", order.ApplicantName)
		assert.Equal(t, 2, len(order.Slots))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("订单不存在_返回gorm.ErrRecordNotFound", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectQuery("SELECT \\* FROM `reservation_orders`").
			WithArgs(999, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		order, err := repo.FindOrderByID(999)
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// 测试 repository.go 文件中的 func FindOrdersByOpenID(openid string) ([]*ReservationOrder, error)
//
// 函数功能：根据 openid 查询订单列表
//
// 测试场景：
// 1. 正常返回列表
//  1. SQL 语句检测
//     - SELECT * FROM `reservation_orders` WHERE open_id = ? ORDER BY created_at desc
//     - SELECT * FROM `reservation_slots` WHERE `reservation_slots`.`order_id` IN (?,?)
//  2. 不用测试 order 对象是否已经加载上了 slots[] 时间段，因为上一个测试已经测试过了
//
// 2. 无订单时返回空列表
func TestFindOrdersByOpenID(t *testing.T) {
	t.Run("正常返回列表", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		orderRows := sqlmock.NewRows([]string{"id", "order_no", "open_id", "applicant_name",
			"alumni_association", "year", "major", "reason", "phone", "total_slots", "status",
			"created_at", "updated_at"}).
			AddRow(1, "R001", "openid_001", "张三", "校友会", 2020, "CS", "测试",
				"13800138000", 1, StatusPendingLevel1, time.Now(), time.Now()).
			AddRow(2, "R002", "openid_001", "张三", "校友会", 2020, "CS", "测试2",
				"13800138000", 2, StatusApproved, time.Now(), time.Now())

		mock.ExpectQuery("SELECT \\* FROM `reservation_orders` WHERE open_id = \\? ORDER BY created_at desc").
			WithArgs("openid_001").
			WillReturnRows(orderRows)

		mock.ExpectQuery("SELECT \\* FROM `reservation_slots` WHERE `reservation_slots`.`order_id` IN \\(\\?,\\?\\)").
			WithArgs(uint(1), uint(2)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "start_time", "end_time", "status", "password", "created_at", "updated_at"}))

		orders, err := repo.FindOrdersByOpenID("openid_001")
		require.NoError(t, err)
		assert.Len(t, orders, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("无订单时返回空列表", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectQuery("SELECT \\* FROM `reservation_orders` WHERE open_id = \\? ORDER BY created_at desc").
			WithArgs("empty_user").
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_no", "open_id", "applicant_name",
				"alumni_association", "year", "major", "reason", "phone", "total_slots", "status",
				"created_at", "updated_at"}))

		orders, err := repo.FindOrdersByOpenID("empty_user")
		require.NoError(t, err)
		assert.NotNil(t, orders)
		assert.Len(t, orders, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// 测试 repository.go 文件中的 func ListOrders(query *ReservationOrder, page, pageSize int) ([]*ReservationOrder, int64, error)
//
// 函数功能：进行分页查询，并且可以根据状态进行筛选
//
// 测试场景：
// 1. page 小于 1 时，自动修正为 1
// 2. pagesize 超出 1 到 50 范围，自动修正为 20
// 3. 是否有按状态进行筛选
func TestListOrders(t *testing.T) {
	t.Run("page小于1时自动修正为1", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		// COUNT
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `reservation_orders`").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))

		// page 修正为 1，pageSize=20（有效范围），offset=0, limit 被参数化
		mock.ExpectQuery("SELECT \\* FROM `reservation_orders` ORDER BY created_at desc LIMIT \\?").
			WithArgs(20).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_no", "open_id", "applicant_name",
				"alumni_association", "year", "major", "reason", "phone", "total_slots", "status",
				"created_at", "updated_at"}))

		orders, total, err := repo.ListOrders(nil, 0, 20)
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Len(t, orders, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pageSize超出1到50范围时自动修正为20", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `reservation_orders`").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
		mock.ExpectQuery("SELECT \\* FROM `reservation_orders` ORDER BY created_at desc LIMIT \\?").
			WithArgs(20).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_no", "open_id", "applicant_name",
				"alumni_association", "year", "major", "reason", "phone", "total_slots", "status",
				"created_at", "updated_at"}))

		orders, _, err := repo.ListOrders(nil, 1, 100)
		require.NoError(t, err)
		assert.Len(t, orders, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("按状态筛选", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `reservation_orders` WHERE status IN \\(\\?,\\?\\)").
			WithArgs(StatusPendingLevel1, StatusApproved).
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(2))

		orderRows := sqlmock.NewRows([]string{"id", "order_no", "open_id", "applicant_name",
			"alumni_association", "year", "major", "reason", "phone", "total_slots", "status",
			"created_at", "updated_at"}).
			AddRow(1, "R001", "o1", "A", "X", 2020, "CS", "r", "138", 1, StatusPendingLevel1, time.Now(), time.Now()).
			AddRow(2, "R002", "o2", "B", "Y", 2020, "CS", "r", "138", 1, StatusApproved, time.Now(), time.Now())

		mock.ExpectQuery("SELECT \\* FROM `reservation_orders` WHERE status IN \\(\\?,\\?\\) ORDER BY created_at desc LIMIT \\?").
			WithArgs(StatusPendingLevel1, StatusApproved, 20).
			WillReturnRows(orderRows)

		mock.ExpectQuery("SELECT \\* FROM `reservation_slots` WHERE `reservation_slots`.`order_id` IN \\(\\?,\\?\\)").
			WithArgs(uint(1), uint(2)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_id", "start_time", "end_time", "status", "password", "created_at", "updated_at"}))

		orders, total, err := repo.ListOrders([]int{StatusPendingLevel1, StatusApproved}, 1, 20)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, orders, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// repository.go 文件中的 func UpdateOrderStatus(orderID uint, fromStatus, toStatus int) error
//
// 函数功能：更新订单状态，采用乐观锁
//
// 测试场景：
// 1. 修改成功
// 2. 修改失败，返回错误中包含“订单状态不匹配”

func TestUpdateOrderStatus(t *testing.T) {
	t.Run("乐观锁成功_状态匹配", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `reservation_orders` SET `status`=\\?,`updated_at`=\\? WHERE id = \\? AND status = \\?").
			WithArgs(StatusApproved, anyTime{}, uint(1), StatusPendingLevel1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE `reservation_slots` SET `status`=\\?,`updated_at`=\\? WHERE order_id = \\? AND status = \\?").
			WithArgs(StatusApproved, anyTime{}, uint(1), StatusPendingLevel1).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.UpdateOrderStatus(1, StatusPendingLevel1, StatusApproved)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("乐观锁失败_状态不匹配", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `reservation_orders` SET `status`=\\?,`updated_at`=\\? WHERE id = \\? AND status = \\?").
			WithArgs(StatusApproved, anyTime{}, uint(1), StatusPendingLevel1).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		err := repo.UpdateOrderStatus(1, StatusPendingLevel1, StatusApproved)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "订单状态不匹配")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// 测试 repository.go 文件中的 func CancelOrder(orderID uint, openid string) error
//
// 函数功能：取消订单，只有订单处于一级审核状态，才能够取消
//
// 测试场景：
// 1. 正常取消：测试 SQL 语句是否正常
// 2. 订单不存在或者订单不属于用户：测试能否正常触发 error

func TestCancelOrder(t *testing.T) {
	t.Run("正常取消", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `reservation_orders` SET `status`=\\?,`updated_at`=\\? WHERE id = \\? AND open_id = \\? AND status = \\?").
			WithArgs(StatusCancelled, anyTime{}, uint(1), "openid_001", StatusPendingLevel1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE `reservation_slots` SET `status`=\\?,`updated_at`=\\? WHERE order_id = \\? AND status = \\?").
			WithArgs(StatusCancelled, anyTime{}, uint(1), StatusPendingLevel1).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := repo.CancelOrder(1, "openid_001")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("订单不存在或不属于该用户", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `reservation_orders` SET `status`=\\?,`updated_at`=\\? WHERE id = \\? AND open_id = \\? AND status = \\?").
			WithArgs(StatusCancelled, anyTime{}, uint(999), "wrong_user", StatusPendingLevel1).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		err := repo.CancelOrder(999, "wrong_user")
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ========== SetSlotPassword ==========

func TestSetSlotPassword(t *testing.T) {
	t.Run("已通过时段正常设置密码", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectExec("UPDATE `reservation_slots` SET `password`=\\?,`updated_at`=\\? WHERE id = \\? AND status = \\?").
			WithArgs("123456", anyTime{}, uint(10), StatusApproved).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.SetSlotPassword(10, "123456")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("非已通过状态时段返回错误", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectExec("UPDATE `reservation_slots` SET `password`=\\?,`updated_at`=\\? WHERE id = \\? AND status = \\?").
			WithArgs("pwd", anyTime{}, uint(10), StatusApproved).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.SetSlotPassword(10, "pwd")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "时段不存在或状态不允许设置密码")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ========== CreateReviewRecord + FindReviewRecordsByOrderID ==========

func TestReviewRecordOperations(t *testing.T) {
	t.Run("创建审核记录", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		// SkipDefaultTransaction=true 时 Create 不包裹事务，直接执行 INSERT
		mock.ExpectExec("INSERT INTO `review_records`").
			WillReturnResult(sqlmock.NewResult(1, 1))

		record := &ReviewRecord{
			OrderID:      1,
			ReviewerID:   10,
			ReviewerRole: 1,
			Action:       1,
			Comment:      "审核通过",
		}
		err := repo.CreateReviewRecord(record)
		require.NoError(t, err)
		assert.Equal(t, uint(1), record.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("查询审核记录", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "order_id", "reviewer_id", "reviewer_role", "action", "comment", "created_at"}).
			AddRow(1, 1, 10, 1, 1, "一级通过", now).
			AddRow(2, 1, 20, 2, 1, "二级通过", now)

		mock.ExpectQuery("SELECT \\* FROM `review_records` WHERE order_id = \\? ORDER BY created_at asc").
			WithArgs(uint(1)).
			WillReturnRows(rows)

		records, err := repo.FindReviewRecordsByOrderID(1)
		require.NoError(t, err)
		assert.Len(t, records, 2)
		assert.Equal(t, "一级通过", records[0].Comment)
		assert.Equal(t, "二级通过", records[1].Comment)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// ========== FindSlotsByTimeRange ==========

func TestFindSlotsByTimeRange(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := NewRepository(gormDB)

	start := time.Date(2026, 3, 25, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 3, 26, 0, 0, 0, 0, time.Local)

	rows := sqlmock.NewRows([]string{"id", "order_id", "start_time", "end_time", "status", "password", "created_at", "updated_at"}).
		AddRow(1, 1, time.Date(2026, 3, 25, 14, 0, 0, 0, time.Local),
			time.Date(2026, 3, 25, 16, 0, 0, 0, time.Local), StatusPendingLevel1, "", time.Now(), time.Now())

	// GORM 生成 SQL 格式: WHERE status IN (?,?) AND (start_time < ? AND end_time > ?)
	mock.ExpectQuery("SELECT \\* FROM `reservation_slots` WHERE status IN \\(\\?,\\?,\\?\\) AND \\(start_time < \\? AND end_time > \\?\\)").
		WithArgs(StatusPendingLevel1, StatusPendingLevel2, StatusApproved, end, start).
		WillReturnRows(rows)

	slots, err := repo.FindSlotsByTimeRange(start, end)
	require.NoError(t, err)
	assert.Len(t, slots, 1)
	assert.Equal(t, uint(1), slots[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ========== UpdateSlotStatus ==========

func TestUpdateSlotStatus(t *testing.T) {
	gormDB, mock := newMockDB(t)
	repo := NewRepository(gormDB)

	mock.ExpectExec("UPDATE `reservation_slots` SET `status`=\\?,`updated_at`=\\? WHERE id = \\?").
		WithArgs(StatusApproved, anyTime{}, uint(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateSlotStatus(5, StatusApproved)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ========== CreateOrderWithLock 原子操作 ==========

func TestCreateOrderWithLock(t *testing.T) {
	t.Run("无冲突时正常创建", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectBegin()
		// SELECT count(*)...FOR UPDATE — GORM 会加括号: AND (start_time < ? AND end_time > ?)
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `reservation_slots` WHERE status IN \\(\\?,\\?,\\?\\) AND \\(start_time < \\? AND end_time > \\?\\)").
			WithArgs(StatusPendingLevel1, StatusPendingLevel2, StatusApproved, anyTime{}, anyTime{}).
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
		mock.ExpectExec("INSERT INTO `reservation_orders`").
			WillReturnResult(sqlmock.NewResult(100, 1))
		mock.ExpectExec("INSERT INTO `reservation_slots`").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		slot := ReservationSlot{
			StartTime: time.Date(2026, 3, 25, 14, 0, 0, 0, time.Local),
			EndTime:   time.Date(2026, 3, 25, 16, 0, 0, 0, time.Local),
			Status:    StatusPendingLevel1,
		}
		order := &ReservationOrder{
			OpenID:        "openid_001",
			ApplicantName: "张三",
			TotalSlots:    1,
			Status:        StatusPendingLevel1,
		}

		err := repo.CreateOrderWithLock(order, []ReservationSlot{slot})
		require.NoError(t, err)
		assert.Equal(t, uint(100), order.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("时段冲突时返回错误", func(t *testing.T) {
		gormDB, mock := newMockDB(t)
		repo := NewRepository(gormDB)

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `reservation_slots` WHERE status IN \\(\\?,\\?,\\?\\) AND \\(start_time < \\? AND end_time > \\?\\)").
			WithArgs(StatusPendingLevel1, StatusPendingLevel2, StatusApproved, anyTime{}, anyTime{}).
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(1))
		mock.ExpectRollback()

		slot := ReservationSlot{
			StartTime: time.Date(2026, 3, 25, 14, 0, 0, 0, time.Local),
			EndTime:   time.Date(2026, 3, 25, 16, 0, 0, 0, time.Local),
		}
		order := &ReservationOrder{OpenID: "openid_001"}

		err := repo.CreateOrderWithLock(order, []ReservationSlot{slot})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "第1个时间段已被预约")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
