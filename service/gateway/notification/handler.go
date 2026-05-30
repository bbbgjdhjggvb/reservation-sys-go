package notification

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/service/gateway/auth"

	"github.com/gin-gonic/gin"
	"github.com/silenceper/wechat/v2/officialaccount"
	"github.com/silenceper/wechat/v2/officialaccount/message"
)

// NotificationHandler 结构体，持有 service 的引用
type NotificationHandler struct {
	svc *NotificationService
}

// NewNotificationHandler 创建通知处理器实例。
//
// 参数:
//   - svc: 通知服务实例
//
// 返回值:
//   - *NotificationHandler: 通知处理器实例
func NewNotificationHandler(svc *NotificationService) *NotificationHandler {
	return &NotificationHandler{
		svc: svc,
	}
}

// ProcessMessage 处理所有来自微信的消息入口（关注/取消关注事件、文本消息等）。
//
// 参数:
//   - oa: 微信公众号实例
//   - msg: 微信混合消息
//
// 返回值:
//   - *message.Reply: 回复消息（无需回复时返回 nil）
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

// NotifyHandler 发送微信模板消息通知（审核通过后通知用户门锁密码等信息）。
//
// 验证流程:
//  1. 校验管理员身份和角色（需一级管理员）
//  2. 解析订单 ID 并查询订单信息
//  3. 检查订单状态和密码是否已设置
//  4. 调用微信模板消息接口发送通知
//
//	@Summary		发送微信模板消息通知
//	@Description	一级管理员向已审核通过的订单用户发送微信模板消息（含门锁密码等信息）
//	@Tags			网关-通知
//	@Produce		json
//	@Param			id	path		string	true	"订单ID"
//	@Success		200	{object}	auth.AdminResp	"通知发送成功"
//	@Failure		400	{object}	auth.AdminResp	"参数错误或订单状态不符"
//	@Failure		401	{object}	auth.AdminResp	"未登录"
//	@Failure		403	{object}	auth.AdminResp	"权限不足"
//	@Failure		500	{object}	auth.AdminResp	"服务未配置或发送失败"
//	@Security		BearerAuth
//	@Router			/api/admin/order/{id}/notify [post]
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
