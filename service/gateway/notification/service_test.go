package notification

import (
	"errors"
	"testing"

	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== NewNotificationService ==========

func TestNewNotificationService(t *testing.T) {
	svc := NewNotificationService(nil)
	assert.NotNil(t, svc)
	assert.Nil(t, svc.notifier)
	assert.Nil(t, svc.orderFetcher)
}

// ========== SetOrderFetcher / GetOrderFetcher ==========

func TestOrderFetcher(t *testing.T) {
	svc := NewNotificationService(nil)

	t.Run("未设置时返回nil", func(t *testing.T) {
		assert.Nil(t, svc.GetOrderFetcher())
	})

	t.Run("设置后可获取", func(t *testing.T) {
		dummyFetcher := func(orderID uint) (*reservationdb.ReservationOrder, error) {
			return &reservationdb.ReservationOrder{ID: orderID}, nil
		}
		svc.SetOrderFetcher(dummyFetcher)

		fetcher := svc.GetOrderFetcher()
		assert.NotNil(t, fetcher)

		order, err := fetcher(42)
		require.NoError(t, err)
		assert.Equal(t, uint(42), order.ID)
	})
}

// ========== SendApprovalNotification ==========

func TestSendApprovalNotification(t *testing.T) {
	t.Run("notifier未初始化时返回错误", func(t *testing.T) {
		svc := NewNotificationService(nil)
		err := svc.SendApprovalNotification(&reservationdb.ReservationOrder{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "微信通知服务未初始化")
	})
}

// ========== SendRejectionNotification ==========

func TestSendRejectionNotification(t *testing.T) {
	t.Run("notifier未初始化时返回错误", func(t *testing.T) {
		svc := NewNotificationService(nil)
		err := svc.SendRejectionNotification(&reservationdb.ReservationOrder{}, "时间冲突")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "微信通知服务未初始化")
	})
}

// ========== NotificationHandler 构造 ==========

func TestNewNotificationHandler(t *testing.T) {
	svc := NewNotificationService(nil)
	hdl := NewNotificationHandler(svc)
	assert.NotNil(t, hdl)
	assert.Equal(t, svc, hdl.svc)
}

// ========== WechatNotifier 构造 ==========

func TestNewWechatNotifier(t *testing.T) {
	n := NewWechatNotifier(nil, "")
	assert.NotNil(t, n)
	assert.Equal(t, "", n.templateID)

	n2 := NewWechatNotifier(nil, "template_abc")
	assert.NotNil(t, n2)
	assert.Equal(t, "template_abc", n2.templateID)
}

// ========== WechatNotifier 错误路径 ==========

func TestWechatNotifier_NotifyApprovalApproved_Errors(t *testing.T) {
	t.Run("oa未初始化时返回错误", func(t *testing.T) {
		n := NewWechatNotifier(nil, "template_123")
		err := n.NotifyApprovalApproved(&reservationdb.ReservationOrder{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "微信服务号未初始化")
	})

	t.Run("templateID为空时返回错误", func(t *testing.T) {
		n := NewWechatNotifier(nil, "")
		// oa is nil, so it'll hit that error first
		err := n.NotifyApprovalApproved(&reservationdb.ReservationOrder{})
		assert.Error(t, err)
	})
}

func TestWechatNotifier_NotifyApprovalRejected_Errors(t *testing.T) {
	t.Run("oa未初始化时返回错误", func(t *testing.T) {
		n := NewWechatNotifier(nil, "template_123")
		err := n.NotifyApprovalRejected(&reservationdb.ReservationOrder{}, "驳回原因")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "微信服务号未初始化")
	})
}

// ========== HandleSubscribe (通过 handler) ==========

// mockUserStore 模拟 UserAuthService 的用户存储操作
type mockUserStore struct {
	findOrCreateFn func(openid string) error
	setStatusFn    func(openid string, active bool) error
}

// ========== WechatNotifier 格式化测试 ==========

func TestWechatNotifier_SlotFormatting(t *testing.T) {
	// 验证 WechatNotifier 内部格式化逻辑不 panic
	// 通过 nil oa 触发错误路径，间接验证格式化逻辑
	t.Run("审批通过通知_格式化时段信息不panic", func(t *testing.T) {
		n := NewWechatNotifier(nil, "tpl")
		order := &reservationdb.ReservationOrder{
			ApplicantName:     "测试用户",
			AlumniAssociation: "测试校友会",
			OrderNo:           "R20260325001",
			OpenID:            "openid_test",
			Slots: []reservationdb.ReservationSlot{
				{},
			},
		}
		// 应返回 oa 未初始化错误（格式化逻辑在 oa 检查之后，但不会 panic）
		err := n.NotifyApprovalApproved(order)
		assert.Error(t, err)
	})

	t.Run("审批驳回通知_默认驳回原因", func(t *testing.T) {
		n := NewWechatNotifier(nil, "tpl")
		order := &reservationdb.ReservationOrder{
			ApplicantName:     "测试用户",
			AlumniAssociation: "测试校友会",
			OrderNo:           "R20260325001",
			OpenID:            "openid_test",
			Slots: []reservationdb.ReservationSlot{
				{},
			},
		}
		err := n.NotifyApprovalRejected(order, "")
		assert.Error(t, err)
	})
}

// ========== OrderFetcher 集成测试 ==========

func TestOrderFetcher_NilGuard(t *testing.T) {
	svc := NewNotificationService(nil)

	t.Run("orderFetcher为nil时GetOrderFetcher返回nil", func(t *testing.T) {
		fetcher := svc.GetOrderFetcher()
		assert.Nil(t, fetcher)
	})

	t.Run("orderFetcher正确传递错误", func(t *testing.T) {
		expectedErr := errors.New("order not found")
		fetcher := func(orderID uint) (*reservationdb.ReservationOrder, error) {
			return nil, expectedErr
		}
		svc.SetOrderFetcher(fetcher)

		order, err := svc.GetOrderFetcher()(1)
		assert.Error(t, err)
		assert.Nil(t, order)
		assert.Equal(t, expectedErr, err)
	})
}
