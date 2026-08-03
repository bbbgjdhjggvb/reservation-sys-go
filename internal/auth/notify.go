package auth

import (
	"fmt"
	"log"
	"strings"

	"reservation-sys/internal/model"

	"github.com/silenceper/wechat/v2/officialaccount"
	"github.com/silenceper/wechat/v2/officialaccount/message"
)

// ========== 微信模板消息通知 ==========

// Notifier 微信模板消息推送服务。
// 实现 review 模块定义的 Notifier 接口。
type Notifier struct {
	oa         *officialaccount.OfficialAccount
	templateID string
}

// NewNotifier 创建微信通知服务
func NewNotifier(oa *officialaccount.OfficialAccount, templateID string) *Notifier {
	return &Notifier{
		oa:         oa,
		templateID: templateID,
	}
}

// SendApproval 审核通过后发送通知给用户
func (n *Notifier) SendApproval(order *model.ReservationOrder) error {
	if n.oa == nil {
		return fmt.Errorf("微信服务号未初始化")
	}
	if n.templateID == "" {
		return fmt.Errorf("微信模板消息ID未配置")
	}

	slotParts := make([]string, 0, len(order.Slots))
	for _, s := range order.Slots {
		line := fmt.Sprintf("%s~%s", s.StartTime.Format("01-02 15:04"), s.EndTime.Format("15:04"))
		if s.Password != "" {
			line += fmt.Sprintf(" 密码:%s", s.Password)
		}
		slotParts = append(slotParts, line)
	}
	slotsText := strings.Join(slotParts, "\n")

	tplMsg := &message.TemplateMessage{
		ToUser:     order.OpenID,
		TemplateID: n.templateID,
		Data: map[string]*message.TemplateDataItem{
			"first": {
				Value: "您的场地预约已审核通过！\n",
				Color: "#10B981",
			},
			"keyword1": {Value: order.ApplicantName},
			"keyword2": {Value: slotsText},
			"keyword3": {Value: order.AlumniAssociation},
			"remark": {
				Value: fmt.Sprintf("\n订单号: %s\n请凭门锁密码在预约时间段内使用场地。", order.OrderNo),
			},
		},
	}

	msgID, err := n.oa.GetTemplate().Send(tplMsg)
	if err != nil {
		log.Printf("[error][auth/notify] 发送模板消息失败: order_no=%s openid=%s err=%v",
			order.OrderNo, order.OpenID, err)
		return fmt.Errorf("发送微信通知失败: %v", err)
	}

	log.Printf("[info][auth/notify] 模板消息发送成功: order_no=%s openid=%s msgid=%d",
		order.OrderNo, order.OpenID, msgID)
	return nil
}

// SendRejection 审核驳回后发送通知给用户
func (n *Notifier) SendRejection(order *model.ReservationOrder, reason string) error {
	if n.oa == nil {
		return fmt.Errorf("微信服务号未初始化")
	}
	if n.templateID == "" {
		return fmt.Errorf("微信模板消息ID未配置")
	}

	slotParts := make([]string, 0, len(order.Slots))
	for _, s := range order.Slots {
		slotParts = append(slotParts, fmt.Sprintf("%s~%s",
			s.StartTime.Format("01-02 15:04"), s.EndTime.Format("15:04")))
	}
	slotsText := strings.Join(slotParts, "\n")

	if reason == "" {
		reason = "请咨询管理员了解详情"
	}

	tplMsg := &message.TemplateMessage{
		ToUser:     order.OpenID,
		TemplateID: n.templateID,
		Data: map[string]*message.TemplateDataItem{
			"first": {
				Value: "您的场地预约未通过审核。\n",
				Color: "#EF4444",
			},
			"keyword1": {Value: order.ApplicantName},
			"keyword2": {Value: slotsText},
			"keyword3": {Value: order.AlumniAssociation},
			"remark": {
				Value: fmt.Sprintf("\n驳回原因: %s\n如有疑问请联系管理员。", reason),
			},
		},
	}

	msgID, err := n.oa.GetTemplate().Send(tplMsg)
	if err != nil {
		log.Printf("[error][auth/notify] 发送驳回通知失败: order_no=%s openid=%s err=%v",
			order.OrderNo, order.OpenID, err)
		return fmt.Errorf("发送微信通知失败: %v", err)
	}

	log.Printf("[info][auth/notify] 驳回通知发送成功: order_no=%s openid=%s msgid=%d",
		order.OrderNo, order.OpenID, msgID)
	return nil
}

// ========== 微信消息处理 ==========

// ProcessMessage 处理所有来自微信的消息入口（关注/取消关注事件、文本消息等）。
func (n *Notifier) ProcessMessage(svc *Service, msg *message.MixMessage) *message.Reply {
	if msg == nil {
		log.Println("[auth/wechat] msg is nil")
		return nil
	}

	if msg.MsgType == message.MsgTypeEvent {
		switch msg.Event {
		case message.EventSubscribe:
			if _, err := svc.FindOrCreate(string(msg.FromUserName)); err != nil {
				log.Printf("[auth/wechat] HandleSubscribe failed: %v, openid: %s", err, msg.FromUserName)
			}
			return nil

		case message.EventUnsubscribe:
			if err := svc.SetStatus(string(msg.FromUserName), false); err != nil {
				log.Printf("[auth/wechat] HandleUnsubscribe failed: %v, openid: %s", err, msg.FromUserName)
			}
			return nil

		default:
			log.Printf("[auth/wechat] unhandled event: %s, openid: %s", msg.Event, msg.FromUserName)
		}
	}

	if msg.MsgType == message.MsgTypeText {
		return &message.Reply{
			MsgType: message.MsgTypeText,
			MsgData: message.NewText("如有疑问，请咨询客服：1234567"),
		}
	}

	return nil
}
