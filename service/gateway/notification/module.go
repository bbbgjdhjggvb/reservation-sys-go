package notification

import (
	"reservation-sys/service/gateway/auth"

	"github.com/silenceper/wechat/v2/officialaccount"
)

var (
	notificationService *NotificationService
	notificationHandler *NotificationHandler
	wechatNotifier      *WechatNotifier
)

// InitModule 初始化通知模块（完整版，由 v1 调用）
// 包含微信消息事件处理 + 模板消息推送服务
func InitModule(authService *auth.UserAuthService, oa *officialaccount.OfficialAccount, templateID string) {
	notificationService = NewNotificationService(authService)
	notificationHandler = NewNotificationHandler(notificationService)

	// 初始化微信模板消息推送（始终创建，templateID 为空时发送会返回明确错误）
	if oa != nil {
		wechatNotifier = NewWechatNotifier(oa, templateID)
		notificationService.notifier = wechatNotifier
	}
}

// InitNotifyModule 初始化通知推送模块（精简版，由 v3 调用）
// 仅初始化模板消息推送功能，不包含微信消息事件处理
func InitNotifyModule(oa *officialaccount.OfficialAccount, templateID string) {
	notificationService = NewNotificationService(nil)
	notificationHandler = NewNotificationHandler(notificationService)

	// 初始化微信模板消息推送（始终创建，templateID 为空时发送会返回明确错误）
	if oa != nil {
		wechatNotifier = NewWechatNotifier(oa, templateID)
		notificationService.notifier = wechatNotifier
	}
}

// SetOrderFetcher 设置订单查询函数（由 cmd 层注入，避免循环依赖）
func SetOrderFetcher(fetcher OrderFetcher) {
	if notificationService != nil {
		notificationService.SetOrderFetcher(fetcher)
	}
}

// GetNotificationService 获取通知服务实例
func GetNotificationService() *NotificationService {
	if notificationService == nil {
		panic("notification module not initialized")
	}
	return notificationService
}

// GetNotificationHandler 获取通知处理器实例
func GetNotificationHandler() *NotificationHandler {
	if notificationHandler == nil {
		panic("notification module not initialized")
	}
	return notificationHandler
}

// GetWechatNotifier 获取微信通知服务实例
func GetWechatNotifier() *WechatNotifier {
	return wechatNotifier
}
