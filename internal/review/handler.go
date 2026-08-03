package review

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"reservation-sys/internal/middleware"
	"reservation-sys/internal/model"

	"github.com/gin-gonic/gin"
)

// ========== Handler ==========

// Handler 审核模块 HTTP 处理器
type Handler struct {
	svc *Service
}

// NewHandler 创建审核处理器实例
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ========== 响应辅助 ==========

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{Code: 200, Msg: "success", Data: data})
}

func okWithMsg(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, Response{Code: 200, Msg: msg, Data: data})
}

func badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Response{Code: 400, Msg: msg})
}

func unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Response{Code: 401, Msg: msg})
}

func forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, Response{Code: 403, Msg: msg})
}

func internalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: msg})
}

// ========== Handler 方法 ==========

// GetOrderList 获取订单列表（分页，支持按状态筛选）。
func (h *Handler) GetOrderList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	statusParams := c.QueryArray("status")

	var orders []*model.ReservationOrder
	var total int64
	var err error

	if len(statusParams) > 0 {
		statuses := make([]int, 0, len(statusParams))
		allNeg := true
		for _, sp := range statusParams {
			s, e := strconv.Atoi(sp)
			if e != nil {
				continue
			}
			if s >= 0 {
				allNeg = false
			}
			statuses = append(statuses, s)
		}
		if allNeg {
			orders, total, err = h.svc.GetAllOrders(page, pageSize)
		} else {
			validStatuses := make([]int, 0, len(statuses))
			for _, s := range statuses {
				if s >= 0 {
					validStatuses = append(validStatuses, s)
				}
			}
			orders, total, err = h.svc.GetOrdersByStatuses(validStatuses, page, pageSize)
		}
	} else {
		orders, total, err = h.svc.GetAllOrders(page, pageSize)
	}

	if err != nil {
		log.Printf("[error][review/handler] 查询失败: %v", err)
		internalError(c, "查询失败")
		return
	}

	list := make([]*OrderResp, 0, len(orders))
	for _, o := range orders {
		list = append(list, OrderToResp(o, true))
	}

	ok(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetOrderDetail 获取订单详情（含审核记录）。
func (h *Handler) GetOrderDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		badRequest(c, "无效的订单ID")
		return
	}

	order, records, err := h.svc.GetOrderDetail(uint(id))
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	recordResps := make([]ReviewRecordResp, 0, len(records))
	for _, r := range records {
		recordResps = append(recordResps, ReviewRecordResp{
			ID:           r.ID,
			ReviewerName: fmt.Sprintf("管理员%d", r.ReviewerID),
			ReviewerRole: r.ReviewerRole,
			RoleText:     model.RoleText(r.ReviewerRole),
			Action:       r.Action,
			ActionText:   ActionText(r.Action),
			Comment:      r.Comment,
			CreatedAt:    r.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	ok(c, gin.H{
		"order":          OrderToResp(order, true),
		"review_records": recordResps,
	})
}

// Level1Review 一级审核操作。
func (h *Handler) Level1Review(c *gin.Context) {
	claims, exists := middleware.GetAdminInfo(c)
	if !exists {
		unauthorized(c, "未登录")
		return
	}

	if claims.Role != model.RoleLevel1 {
		forbidden(c, "仅一级管理员可进行一级审核")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		badRequest(c, "无效的订单ID")
		return
	}

	var req ReviewActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "参数错误")
		return
	}

	err = h.svc.Level1Review(claims.AdminID, uint(id), &req)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	actionText := "通过"
	if req.Action == 2 {
		actionText = "拒绝"
	}
	okWithMsg(c, fmt.Sprintf("一级审核%s成功", actionText), nil)
	log.Printf("[info][review/handler] admin_id=%d order_id=%d action=%s", claims.AdminID, id, actionText)
}

// Level2Review 二级审核操作。
func (h *Handler) Level2Review(c *gin.Context) {
	claims, exists := middleware.GetAdminInfo(c)
	if !exists {
		unauthorized(c, "未登录")
		return
	}

	if claims.Role != model.RoleLevel2 {
		forbidden(c, "仅二级管理员可进行二级审核")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		badRequest(c, "无效的订单ID")
		return
	}

	var req ReviewActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "参数错误")
		return
	}

	err = h.svc.Level2Review(claims.AdminID, uint(id), &req)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	actionText := "通过"
	if req.Action == 2 {
		actionText = "拒绝"
	}
	okWithMsg(c, fmt.Sprintf("二级审核%s成功", actionText), nil)
	log.Printf("[info][review/handler] admin_id=%d order_id=%d action=%s", claims.AdminID, id, actionText)
}

// SetPassword 设置门锁密码。
func (h *Handler) SetPassword(c *gin.Context) {
	claims, exists := middleware.GetAdminInfo(c)
	if !exists {
		unauthorized(c, "未登录")
		return
	}

	idStr := c.Param("id")
	orderID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		badRequest(c, "无效的订单ID")
		return
	}

	slotIDStr := c.Param("slotID")
	slotID, err := strconv.ParseUint(slotIDStr, 10, 64)
	if err != nil {
		badRequest(c, "无效的时段ID")
		return
	}

	var req SetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "参数错误")
		return
	}

	req.Password = strings.TrimSpace(req.Password)

	err = h.svc.SetPassword(claims.Role, uint(orderID), uint(slotID), req.Password)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	okWithMsg(c, "门锁密码设置成功", nil)
	log.Printf("[info][review/handler] admin_id=%d order_id=%d slot_id=%d", claims.AdminID, orderID, slotID)
}

// SendApprovalNotify 发送审核通过通知。
func (h *Handler) SendApprovalNotify(c *gin.Context) {
	claims, exists := middleware.GetAdminInfo(c)
	if !exists {
		unauthorized(c, "未登录")
		return
	}

	if claims.Role != model.RoleLevel1 {
		forbidden(c, "仅一级管理员可发送通知")
		return
	}

	idStr := c.Param("id")
	orderID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		badRequest(c, "无效的订单ID")
		return
	}

	if err := h.svc.SendApprovalNotify(c.Request.Context(), uint(orderID)); err != nil {
		badRequest(c, err.Error())
		return
	}

	okWithMsg(c, fmt.Sprintf("通知已发送给用户（订单ID: %d）", orderID), nil)
	log.Printf("[info][review/handler] admin_id=%d order_id=%d send approval notify",
		claims.AdminID, orderID)
}

// SendRejectionNotify 发送审核驳回通知。
func (h *Handler) SendRejectionNotify(c *gin.Context) {
	claims, exists := middleware.GetAdminInfo(c)
	if !exists {
		unauthorized(c, "未登录")
		return
	}

	if claims.Role != model.RoleLevel1 {
		forbidden(c, "仅一级管理员可发送驳回通知")
		return
	}

	idStr := c.Param("id")
	orderID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		badRequest(c, "无效的订单ID")
		return
	}

	var req RejectionNotifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "参数错误")
		return
	}

	if err := h.svc.SendRejectionNotify(c.Request.Context(), uint(orderID), req.Reason); err != nil {
		badRequest(c, err.Error())
		return
	}

	okWithMsg(c, fmt.Sprintf("驳回通知已发送给用户（订单ID: %d）", orderID), nil)
	log.Printf("[info][review/handler] admin_id=%d order_id=%d reason=%s",
		claims.AdminID, orderID, req.Reason)
}
