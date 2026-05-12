/*
 * NotificationService 依赖于 AuthService
 */
package notification

import (
	"fmt"

	"reservation-sys/service/gateway/auth"
	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/silenceper/wechat/v2/officialaccount"
)

// OrderFetcher 获取订单信息的函数类型（由调用方注入，避免循环依赖）。
// 使用函数类型而非直接依赖 Repository，是因为 notification 包不应感知数据库细节。
type OrderFetcher func(orderID uint) (*reservationdb.ReservationOrder, error)

type NotificationService struct {
	authService  *auth.UserAuthService
	notifier     *WechatNotifier
	orderFetcher OrderFetcher
}

func NewNotificationService(authService *auth.UserAuthService) *NotificationService {
	return &NotificationService{authService: authService}
}

// SetOrderFetcher 设置订单查询函数
func (s *NotificationService) SetOrderFetcher(fetcher OrderFetcher) {
	s.orderFetcher = fetcher
}

// GetOrderFetcher 获取订单查询函数
func (s *NotificationService) GetOrderFetcher() OrderFetcher {
	return s.orderFetcher
}

// HandleSubscribe 处理用户关注事件（创建或更新用户记录）。
//
// 参数:
//   - oa: 微信公众号实例（用于获取用户信息）
//   - openid: 关注用户的 openid
//
// 返回值:
//   - error: 用户创建/更新失败时返回错误
func (s *NotificationService) HandleSubscribe(oa *officialaccount.OfficialAccount, openid string) error {
	_, err := s.authService.FindOrCreate(openid)
	return err
}

// HandleUnsubscribe 处理用户取消关注事件（设置用户状态为不活跃）。
//
// 参数:
//   - openid: 取消关注用户的 openid
//
// 返回值:
//   - error: 状态更新失败时返回错误
func (s *NotificationService) HandleUnsubscribe(openid string) error {
	return s.authService.SetStatus(openid, false)
}

// SendApprovalNotification 发送审核通过通知给用户（委托 WechatNotifier 发送）。
//
// 参数:
//   - order: 订单实体（含时段和密码信息）
//
// 返回值:
//   - error: 微信通知服务未初始化或发送失败时返回错误
func (s *NotificationService) SendApprovalNotification(order *reservationdb.ReservationOrder) error {
	if s.notifier == nil {
		return fmt.Errorf("微信通知服务未初始化")
	}
	return s.notifier.NotifyApprovalApproved(order)
}

// SendRejectionNotification 发送审核驳回通知给用户（委托 WechatNotifier 发送）。
//
// 参数:
//   - order: 订单实体（含时段信息）
//   - reason: 驳回原因
//
// 返回值:
//   - error: 微信通知服务未初始化或发送失败时返回错误
func (s *NotificationService) SendRejectionNotification(order *reservationdb.ReservationOrder, reason string) error {
	if s.notifier == nil {
		return fmt.Errorf("微信通知服务未初始化")
	}
	return s.notifier.NotifyApprovalRejected(order, reason)
}
