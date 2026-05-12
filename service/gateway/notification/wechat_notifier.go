package notification

import (
	"fmt"
	"log"
	"strings"

	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/silenceper/wechat/v2/officialaccount"
	"github.com/silenceper/wechat/v2/officialaccount/message"
)

// WechatNotifier 微信模板消息推送服务
type WechatNotifier struct {
	oa         *officialaccount.OfficialAccount
	templateID string
}

// NewWechatNotifier 创建微信通知服务
func NewWechatNotifier(oa *officialaccount.OfficialAccount, templateID string) *WechatNotifier {
	return &WechatNotifier{
		oa:         oa,
		templateID: templateID,
	}
}

// NotifyApprovalApproved 审核通过后发送通知给用户
func (n *WechatNotifier) NotifyApprovalApproved(order *reservationdb.ReservationOrder) error {
	if n.oa == nil {
		return fmt.Errorf("微信服务号未初始化")
	}
	if n.templateID == "" {
		return fmt.Errorf("微信模板消息ID未配置")
	}

	// 构建时段与密码信息
	slotParts := make([]string, 0, len(order.Slots))
	for _, s := range order.Slots {
		line := fmt.Sprintf("%s~%s", s.StartTime.Format("01-02 15:04"), s.EndTime.Format("15:04"))
		if s.Password != "" {
			line += fmt.Sprintf(" 密码:%s", s.Password)
		}
		slotParts = append(slotParts, line)
	}
	slotsText := strings.Join(slotParts, "\n")

	// 构建模板消息数据
	tplMsg := &message.TemplateMessage{
		ToUser:     order.OpenID,
		TemplateID: n.templateID,
		Data: map[string]*message.TemplateDataItem{
			"first": {
				Value: "您的场地预约已审核通过！\n",
				Color: "#10B981",
			},
			"keyword1": {
				Value: order.ApplicantName,
			},
			"keyword2": {
				Value: slotsText,
			},
			"keyword3": {
				Value: order.AlumniAssociation,
			},
			"remark": {
				Value: fmt.Sprintf("\n订单号: %s\n请凭门锁密码在预约时间段内使用场地。", order.OrderNo),
			},
		},
	}

	msgID, err := n.oa.GetTemplate().Send(tplMsg)
	if err != nil {
		log.Printf("[error][notification/wechat] 发送模板消息失败: order_no=%s openid=%s err=%v", order.OrderNo, order.OpenID, err)
		return fmt.Errorf("发送微信通知失败: %v", err)
	}

	log.Printf("[info][notification/wechat] 模板消息发送成功: order_no=%s openid=%s msgid=%d", order.OrderNo, order.OpenID, msgID)
	return nil
}

// NotifyApprovalRejected 审核驳回后发送通知给用户
func (n *WechatNotifier) NotifyApprovalRejected(order *reservationdb.ReservationOrder, reason string) error {
	if n.oa == nil {
		return fmt.Errorf("微信服务号未初始化")
	}
	if n.templateID == "" {
		return fmt.Errorf("微信模板消息ID未配置")
	}

	// 构建时段信息
	slotParts := make([]string, 0, len(order.Slots))
	for _, s := range order.Slots {
		slotParts = append(slotParts, fmt.Sprintf("%s~%s", s.StartTime.Format("01-02 15:04"), s.EndTime.Format("15:04")))
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
			"keyword1": {
				Value: order.ApplicantName,
			},
			"keyword2": {
				Value: slotsText,
			},
			"keyword3": {
				Value: order.AlumniAssociation,
			},
			"remark": {
				Value: fmt.Sprintf("\n驳回原因: %s\n如有疑问请联系管理员。", reason),
			},
		},
	}

	msgID, err := n.oa.GetTemplate().Send(tplMsg)
	if err != nil {
		log.Printf("[error][notification/wechat] 发送驳回通知失败: order_no=%s openid=%s err=%v", order.OrderNo, order.OpenID, err)
		return fmt.Errorf("发送微信通知失败: %v", err)
	}

	log.Printf("[info][notification/wechat] 驳回通知发送成功: order_no=%s openid=%s msgid=%d", order.OrderNo, order.OpenID, msgID)
	return nil
}
