package review

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"reservation-sys/pkg/constants"
	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/service/admin/auth"

	pb "reservation-sys/service/gateway/api/gen/notification"

	"github.com/gin-gonic/gin"
)

// NotifyHandler 通过 gRPC 调用 Gateway 通知服务的 HTTP 处理器。
// 负责发送审核通过/驳回的微信模板消息通知，订单查询通过 pkg/reservationdb 直接操作数据库。
type NotifyHandler struct {
	notifyCli pb.NotificationServiceClient
	repo      reservationdb.Repository
}

// NewNotifyHandler 创建通知处理器。
//
// 参数:
//   - notifyCli: Gateway 通知服务的 gRPC 客户端
//   - repo: 预约数据库仓库接口（用于查询订单信息）
//
// 返回值:
//   - *NotifyHandler: 通知处理器实例
func NewNotifyHandler(notifyCli pb.NotificationServiceClient, repo reservationdb.Repository) *NotifyHandler {
	return &NotifyHandler{notifyCli: notifyCli, repo: repo}
}

// RejectionNotifyHandler 发送驳回通知接口。
// 请求: POST /api/admin/orders/:id/reject-notify
// 权限: 仅一级管理员
// 流程:
//  1. 从数据库查询订单信息
//  2. 校验订单状态为"已驳回"
//  3. 通过 gRPC 调用 Gateway 发送微信驳回模板消息
// 响应: 200 通知发送成功，400 订单不存在/状态不允许，500 通知发送失败
func (h *NotifyHandler) RejectionNotifyHandler(c *gin.Context) {
	claims, exists := auth.GetAdminInfo(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, auth.AdminResp{Code: 401, Msg: "未登录"})
		return
	}

	if claims.Role != constants.RoleLevel1 {
		c.JSON(http.StatusForbidden, auth.AdminResp{Code: 403, Msg: "仅一级管理员可发送驳回通知"})
		return
	}

	idStr := c.Param("id")
	orderID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "无效的订单ID"})
		return
	}

	var req RejectionNotifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "参数错误"})
		return
	}

	order, err := h.repo.FindOrderByID(uint(orderID))
	if err != nil {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "订单不存在"})
		return
	}

	if order.Status != reservationdb.StatusRejectedLevel1 && order.Status != reservationdb.StatusRejectedLevel2 {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "仅被驳回的订单可发送驳回通知"})
		return
	}

	// 通过 gRPC 调用 Gateway 发送驳回通知
	resp, err := h.notifyCli.SendRejectionNotification(c.Request.Context(), &pb.RejectionNotificationReq{
		Openid:            order.OpenID,
		ApplicantName:     order.ApplicantName,
		AlumniAssociation: order.AlumniAssociation,
		OrderNo:           order.OrderNo,
		Slots:             orderSlotsToNotify(order.Slots),
		Reason:            req.Reason,
	})
	if err != nil {
		log.Printf("[error][review/reject-notify] 发送驳回通知失败: order_id=%d err=%v", orderID, err)
		c.JSON(http.StatusInternalServerError, auth.AdminResp{Code: 500, Msg: fmt.Sprintf("驳回通知发送失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, auth.AdminResp{
		Code: 200,
		Msg:  fmt.Sprintf("驳回通知已发送给用户（订单号: %s）", order.OrderNo),
		Data: resp.Message,
	})

	log.Printf("[info][review/reject-notify] admin_id=%d order_id=%d order_no=%s openid=%s reason=%s",
		claims.AdminID, orderID, order.OrderNo, order.OpenID, req.Reason)
}

// NotifyHandler 发送审核通过通知接口。
// 请求: POST /api/admin/orders/:id/notify
// 权限: 仅一级管理员
// 流程:
//  1. 从数据库查询订单信息
//  2. 校验订单状态为"终审通过"且已设置门锁密码
//  3. 通过 gRPC 调用 Gateway 发送微信审核通过模板消息
// 响应: 200 通知发送成功，400 订单不存在/状态不允许/未设密码，500 通知发送失败
func (h *NotifyHandler) NotifyHandler(c *gin.Context) {
	claims, exists := auth.GetAdminInfo(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, auth.AdminResp{Code: 401, Msg: "未登录"})
		return
	}

	if claims.Role != constants.RoleLevel1 {
		c.JSON(http.StatusForbidden, auth.AdminResp{Code: 403, Msg: "仅一级管理员可发送通知"})
		return
	}

	idStr := c.Param("id")
	orderID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "无效的订单ID"})
		return
	}

	// 直接从数据库获取订单信息
	order, err := h.repo.FindOrderByID(uint(orderID))
	if err != nil {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "订单不存在"})
		return
	}

	if order.Status != reservationdb.StatusApproved {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "仅审核通过的订单可发送通知"})
		return
	}

	hasPassword := false
	for _, s := range order.Slots {
		if s.Password != "" {
			hasPassword = true
			break
		}
	}
	if !hasPassword {
		c.JSON(http.StatusBadRequest, auth.AdminResp{Code: 400, Msg: "请先设置门锁密码后再发送通知"})
		return
	}

	// 通过 gRPC 调用 Gateway 发送通知
	resp, err := h.notifyCli.SendApprovalNotification(c.Request.Context(), &pb.ApprovalNotificationReq{
		Openid:            order.OpenID,
		ApplicantName:     order.ApplicantName,
		AlumniAssociation: order.AlumniAssociation,
		OrderNo:           order.OrderNo,
		Slots:             orderSlotsToNotify(order.Slots),
	})
	if err != nil {
		log.Printf("[error][review/notify] 发送通知失败: order_id=%d err=%v", orderID, err)
		c.JSON(http.StatusInternalServerError, auth.AdminResp{Code: 500, Msg: fmt.Sprintf("通知发送失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, auth.AdminResp{
		Code: 200,
		Msg:  fmt.Sprintf("通知已发送给用户（订单号: %s）", order.OrderNo),
		Data: resp.Message,
	})

	log.Printf("[info][review/notify] admin_id=%d order_id=%d order_no=%s openid=%s",
		claims.AdminID, orderID, order.OrderNo, order.OpenID)
}

// orderSlotsToNotify 将时段信息转换为 gRPC 通知请求格式。
//
// 参数:
//   - slots: 数据库时段列表
//
// 返回值:
//   - []*pb.SlotInfo: gRPC 通知所需的时段信息切片（时间格式化为 "2006-01-02 15:04"）
func orderSlotsToNotify(slots []reservationdb.ReservationSlot) []*pb.SlotInfo {
	result := make([]*pb.SlotInfo, 0, len(slots))
	for _, s := range slots {
		result = append(result, &pb.SlotInfo{
			StartTime: s.StartTime.Format("2006-01-02 15:04"),
			EndTime:   s.EndTime.Format("2006-01-02 15:04"),
			Password:  s.Password,
		})
	}
	return result
}
