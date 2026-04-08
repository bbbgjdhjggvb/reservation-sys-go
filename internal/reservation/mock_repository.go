// internal/reservation/mock_repository.go
/* Mock 介绍
 * # 什么是 Mock ?
 * ReservationService 依赖 ReservationRepository 接口来访问数据库。
 * 但在测试 Service 层时，我们不希望真正连接数据库，所以需要 mock 来进行模拟

 * # gomock 如何工作
 * gomock 通过两个配合的结构体实现 Mock 功能：
 * 1. MockReservationRepository（Mock实现）
      - 实现了 ReservationRepository 接口
      - 当测试代码调用方法时，它会拦截调用
      - 返回我们在测试中预设的返回值
 * 2. MockReservationRepositoryMockRecorder（调用记录器）
      - 用于在测试中"预设"方法应该如何被调用
      - 比如：预期某方法会被调用一次，并返回特定值

 * #使用示例
 * 1. 创建 Controller（管理整个 mock 生命周期）
 * ctrl := gomock.NewController(t)
 * defer ctrl.Finish()  // 测试结束时检查所有预期是否满足

 * 2. 创建 Mock 实例
 * mockRepo := NewMockReservationRepository(ctrl)

 * 3. 设置预期：当 FindByID(1) 被调用时，返回特定数据
 * mockRepo.EXPECT().
	 * FindByID(uint(1)).
	 * Return(&Reservation{ID: 1, OpenID: "test"}, nil)

 * 4. 使用 mock 进行测试
 * svc := NewReservationService(mockRepo)
 * result, err := svc.Cancel(1, "test")  // 内部会调用 mockRepo.FindByID
*/

package reservation

import (
	"reflect"
	"time"

	"github.com/golang/mock/gomock"
)

/* # Mock 结构体定义
 * MockReservationRepository 是 ReservationRepository 接口的 Mock 实现
 * 当调用 mockRepo.FindByID(1) 时，它不会查数据库，
 * 而是返回你在测试中通过 EXPECT().FindByID(1).Return(...) 设置的值。
 */
type MockReservationRepository struct {
	/* ctrl 是 gomock 的控制器，负责：
	 * 1. 管理所有 mock 对象的生命周期
	 * 2. 在测试结束时验证所有预期调用是否发生
	 * 3. 协调方法调用与预设返回值之间的匹配
	 */
	ctrl *gomock.Controller

	/* recorder 是调用记录器，用于在测试中"预设"方法的行为
	 * 比如：mockRepo.EXPECT().FindByID(1).Return(result, nil)
	 * 这里 EXPECT() 返回的就是 recorder
	 */
	recorder *MockReservationRepositoryMockRecorder
}

/*
	# MockReservationRepositoryMockRecorder 是方法调用的记录器

* 为什么需要两个结构体？
* 这是一种设计模式，将"Mock实现"和"预期设置"分离：

* - MockReservationRepository：被测代码实际调用的对象
* 例如：mockRepo.FindByID(1)  ← 这是调用 Mock 方法
* - MockReservationRepositoryMockRecorder：测试代码设置预期的对象
* 例如：mockRepo.EXPECT().FindByID(1).Return(...)  ← 这是设置预期
* recorder 中的方法与 Mock 结构体中的方法一一对应，
* 但 recorder 方法返回 *gomock.Call，用于链式调用设置返回值。
*/
type MockReservationRepositoryMockRecorder struct {
	// mock 是指向对应 Mock 实例的引用
	// recorder 需要通过它来访问 Controller 进行预期记录
	mock *MockReservationRepository
}

/* NewMockReservationRepository 创建新的 Mock 实例
 * 使用说明：
 * ctrl := gomock.NewController(t)
 * deffer ctrl.Finish()
 * mockRepo := NewMockReservationRepository(ctrl)
 */
func NewMockReservationRepository(ctrl *gomock.Controller) *MockReservationRepository {
	// 创建 Mock 实例，并注入控制器
	mock := &MockReservationRepository{ctrl: ctrl}
	// 创建对应的 recorder，并建立双向引用
	mock.recorder = &MockReservationRepositoryMockRecorder{mock}
	return mock
}

/* EXPECT 返回 Mock 记录器，用于设置方法调用预期，这是设置 Mock 行为的入口点。
 * 使用示例：
 *	mockRepo.EXPECT().FindByID(1).Return(&Reservation{ID: 1}, nil)
 *	│           │           │            │
 *	│           │           │            └── 设置返回值
 *	│           │           └── 指定方法名和参数匹配
 *	│           └── EXPECT() 返回 recorder
 *	└── Mock 实例
 */
func (m *MockReservationRepository) EXPECT() *MockReservationRepositoryMockRecorder {
	return m.recorder
}

// 下面是 Mock 方法的实现，用来模拟调用

/* MockReservationRepository.Create 和 ReservationRepository.Create 对应
 * Mock 方法的工作流程如下
 * 当测试代码调用 mockRepo.Create(res) 时
 * 1. m.ctrl.T.Helper() 标记这个函数为辅助函数
 * 2. m.ctrl.Call(m, "Create", res) 查找匹配的预期并执行
 * 3. 类型断言提取返回值: ret[0].(error)
 */
func (m *MockReservationRepository) Create(res *Reservation) error {
	// Helper() 标记此函数为测试辅助函数
	// 当测试失败时，错误信息会跳过此函数，直接指向调用者的代码行
	m.ctrl.T.Helper()

	/* Call 方法的工作：
	 * 1. 查找是否有匹配的预期（方法名 + 参数匹配）
	 * 2. 找到后返回预设的返回值
	 * 3. 记录此次调用（用于 Finish() 时验证）

	 * 返回值是 []interface{} 切片，包含所有返回值
	 */
	ret := m.ctrl.Call(m, "Create", res)

	// 类型断言：从切片中提取第一个返回值（error 类型）
	// Create 方法只有一个返回值，所以只取 ret[0]
	ret0, _ := ret[0].(error)
	return ret0
}

func (m *MockReservationRepository) FindByID(id uint) (*Reservation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByID", id)

	// 提取第一个返回值：*Reservation（指针类型）
	ret0, _ := ret[0].(*Reservation)
	// 提取第二个返回值：error
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByOpenID 是 FindByOpenID 方法的 Mock 实现
func (m *MockReservationRepository) FindByOpenID(openid string) ([]*Reservation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByOpenID", openid)

	// 注意：返回切片时的类型断言写法
	ret0, _ := ret[0].([]*Reservation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByTimeRange 是 FindByTimeRange 方法的 Mock 实现
func (m *MockReservationRepository) FindByTimeRange(start, end time.Time) ([]*Reservation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByTimeRange", start, end)
	ret0, _ := ret[0].([]*Reservation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateStatus 是 UpdateStatus 方法的 Mock 实现
func (m *MockReservationRepository) UpdateStatus(id uint, status int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStatus", id, status)
	ret0, _ := ret[0].(error)
	return ret0
}

// Cancel 是 Cancel 方法的 Mock 实现
func (m *MockReservationRepository) Cancel(id uint, openid string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cancel", id, openid)
	ret0, _ := ret[0].(error)
	return ret0
}

// 下面是 Recorder 方法的实现，用来设置预期输出

/* # Recoder 方法的作用
 * 告诉 mock ：
 * 1. 该方法应该被调用
 * 2. 参数应该是什么
 * 3. 应该返回什么值

 * # reflect.TypeOf 的作用
 * 用户获取方法的类型信息，让gomock知道
 * 1. 方法有几个参数
 * 2. 每个参数的类型
 * 3. 有几个返回值

 * # 使用示例
 * mockRepo.EXPECT().
 *		FindByID(gomock.Eq(uint(1))).      // 参数匹配器，预期参数为 1
 *     	Return(&Reservation{ID: 1}, nil).  // 设置返回值
 *	 	Times(1)                            // 设置调用次数（可选，默认是 1 次）

 * # 常用参数匹配器
 * 1. gomock.Any()    匹配任意值
 * 2. gomock.Eq(x)    匹配等于 x 的值
 * 3. gomock.Not(x)   匹配不等于 x 的值
 */

// Create 设置 Create 方法的调用预期，返回 *gomock.Call，支持链式调用设置返回值和调用次数等
func (mr *MockReservationRepositoryMockRecorder) Create(res any) *gomock.Call {
	// RecordCallWithMethodType 记录一个方法调用预期
	// 参数说明：
	//   - mr.mock: Mock 对象实例
	//   - "Create": 方法名（字符串）
	//   - reflect.TypeOf(...): 方法的类型信息
	//   - res: 方法参数
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"Create",
		/* 下面的语法介绍
		 * 1. (*MockReservationRepository)(nil) 创建一个 nil 指针，转换为 (*MockReservationRespository) 类型
		 * 2. .Create 获取 Create 方法的指针
		 * 3. reflect.TypeOf 获取 Create 方法的信息
		 */
		reflect.TypeOf((*MockReservationRepository)(nil).Create),
		res,
	)
}

// FindByID 设置 FindByID 方法的调用预期
func (mr *MockReservationRepositoryMockRecorder) FindByID(id any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"FindByID",
		reflect.TypeOf((*MockReservationRepository)(nil).FindByID),
		id,
	)
}

// FindByOpenID 设置 FindByOpenID 方法的调用预期
func (mr *MockReservationRepositoryMockRecorder) FindByOpenID(openid any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"FindByOpenID",
		reflect.TypeOf((*MockReservationRepository)(nil).FindByOpenID),
		openid,
	)
}

// FindByTimeRange 设置 FindByTimeRange 方法的调用预期
//
// 多个参数时，按顺序传入所有参数
func (mr *MockReservationRepositoryMockRecorder) FindByTimeRange(start, end any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"FindByTimeRange",
		reflect.TypeOf((*MockReservationRepository)(nil).FindByTimeRange),
		start, end, // 多个参数
	)
}

// UpdateStatus 设置 UpdateStatus 方法的调用预期
func (mr *MockReservationRepositoryMockRecorder) UpdateStatus(id, status any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"UpdateStatus",
		reflect.TypeOf((*MockReservationRepository)(nil).UpdateStatus),
		id, status,
	)
}

// Cancel 设置 Cancel 方法的调用预期
func (mr *MockReservationRepositoryMockRecorder) Cancel(id, openid any) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock,
		"Cancel",
		reflect.TypeOf((*MockReservationRepository)(nil).Cancel),
		id, openid,
	)
}
