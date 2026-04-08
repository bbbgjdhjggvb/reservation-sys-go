package notification

import (
	"log"

	"github.com/silenceper/wechat/v2/officialaccount"
	"github.com/silenceper/wechat/v2/officialaccount/message"
)

// NotificationHandler 结构体，持有 service 的引用
type NotificationHandler struct {
	svc *NotificationService
}

// NewNotificationHandler 构造函数
func NewNotificationHandler(svc *NotificationService) *NotificationHandler {
	return &NotificationHandler{
		svc: svc,
	}
}

// ProcessMessage 处理所有来自微信的消息入口
func (h *NotificationHandler) ProcessMessage(oa *officialaccount.OfficialAccount, msg *message.MixMessage) *message.Reply {
	if msg == nil {
		log.Println("[NotificationHandler] msg is nil")
		return nil
	}

	if msg.MsgType == message.MsgTypeEvent {
		switch msg.Event {
		// 处理关注事件
		case message.EventSubscribe:
			/* msg.FromUserName为message.CDATA类型，在源代码中的定义为 type CDATA string
			 * 需要显示转为为string类型，因为HandleSubscribe接受的是string类型参数 */
			if err := h.svc.HandleSubscribe(oa, string(msg.FromUserName)); err != nil {
				log.Printf("[NotificationHandler][Fatal] HandleSubscribe failed: %v, openid: %s", err, msg.FromUserName)
			}
			return &message.Reply{
				MsgType: message.MsgTypeText,
				MsgData: message.NewText("欢迎关注场地预约系统！\n点击下方菜单即可开始预约。"),
			}

		case message.EventUnsubscribe:
			// 调用指挥官处理取消关注逻辑
			if err := h.svc.HandleUnsubscribe(string(msg.FromUserName)); err != nil {
				log.Printf("[NotificationHandler][Fatal] HandleUnsubscribe failed: %v, openid: %s", err, msg.FromUserName)
			}
			return nil

		default:
			// 记录未知事件类型
			log.Printf("[NotificationHandler][info] unhandled event type: %s, openid: %s", msg.Event, msg.FromUserName)
		}
	}

	// 2. 如果是普通文本消息 (未来可以扩展关键词回复)
	if msg.MsgType == message.MsgTypeText {
		// 比如：用户输入"帮助"，返回引导语
		return &message.Reply{
			MsgType: message.MsgTypeText,
			MsgData: message.NewText("如有疑问，请咨询客服：1234567"),
		}
	}

	return nil
}
