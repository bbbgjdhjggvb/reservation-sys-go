package notification

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"reservation-sys/service/gateway/auth"
	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/gin-gonic/gin"
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
			if err := h.svc.HandleSubscribe(oa, string(msg.FromUserName)); err != nil {
				log.Printf("[NotificationHandler][Fatal] HandleSubscribe failed: %v, openid: %s", err, msg.FromUserName)
			}
			return &message.Reply{
				MsgType: message.MsgTypeText,
				MsgData: message.NewText("欢迎关注场地预约系统！\n点击下方菜单即可开始预约。"),
			}

		case message.EventUnsubscribe:
			if err := h.svc.HandleUnsubscribe(string(msg.FromUserName)); err != nil {
				log.Printf("[NotificationHandler][Fatal] HandleUnsubscribe failed: %v, openid: %s", err, msg.FromUserName)
			}
			return nil

		default:
			log.Printf("[NotificationHandler][info] unhandled event type: %s, openid: %s", msg.Event, msg.FromUserName)
		}
	}

	// 普通文本消息
	if msg.MsgType == message.MsgTypeText {
		return &message.Reply{
			MsgType: message.MsgTypeText,
			MsgData: message.NewText("如有疑问，请咨询客服：1234567"),
		}
	}

	return nil
}

// ==================== 模板消息通知 Handler ====================

// NotifyHandler 发送微信通知（审核通过后通知用户）
func (h *NotificationHandler) NotifyHandler(c *gin.Context) {
	claims, exists := auth.GetAdminInfo(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, auth.AdminResp{Code: 401, Msg: "未登录"})
		return
	}

	if claims.Role != auth.RoleLevel1 {
		c.JSON(http.StatusForbidden, auth.AdminResp{Code: 403, Msg: "仅一级管理员可发送通知"})
		return
	}

	idStr := c.Param("id")
	orderID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "无效的订单ID"})
		return
	}

	// 通过注入的 OrderFetcher 获取订单信息
	if h.svc.orderFetcher == nil {
		c.JSON(http.StatusInternalServerError, auth.AdminResp{Code: 500, Msg: "订单查询服务未配置"})
		return
	}

	order, err := h.svc.orderFetcher(uint(orderID))
	if err != nil {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: err.Error()})
		return
	}

	// 检查是否已设置密码且已通过审核
	hasPassword := false
	for _, slot := range order.Slots {
		if slot.Password != "" {
			hasPassword = true
			break
		}
	}

	if order.Status != reservationdb.StatusApproved {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "仅审核通过的订单可发送通知"})
		return
	}

	if !hasPassword {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "请先设置门锁密码后再发送通知"})
		return
	}

	// 调用微信模板消息接口发送通知
	if err := h.svc.SendApprovalNotification(order); err != nil {
		log.Printf("[error][notification/notify] 发送通知失败: order_id=%d err=%v", orderID, err)
		c.JSON(http.StatusInternalServerError, auth.AdminResp{Code: 500, Msg: fmt.Sprintf("通知发送失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, auth.AdminResp{
		Code: 200,
		Msg:  fmt.Sprintf("通知已发送给用户（订单号: %s）", order.OrderNo),
		Data: nil,
	})

	log.Printf("[info][notification/notify] admin_id=%d order_id=%d order_no=%s openid=%s",
		claims.AdminID, orderID, order.OrderNo, order.OpenID)
}
