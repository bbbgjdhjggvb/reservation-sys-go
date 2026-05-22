package review

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"reservation-sys/pkg/constants"
	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/service/admin/auth"

	"github.com/gin-gonic/gin"
)

// ReviewHandler 处理审核相关的 HTTP 请求
type ReviewHandler struct {
	svc       *ReviewService
	notifyHdl *NotifyHandler
}

// NewReviewHandler 创建审核处理器实例
func NewReviewHandler(svc *ReviewService, notifyHdl *NotifyHandler) *ReviewHandler {
	return &ReviewHandler{svc: svc, notifyHdl: notifyHdl}
}

// --- 统一响应辅助方法 ---

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

// GetOrderListHandler 获取订单列表（分页，支持按状态筛选）。
//
//	@Summary		获取订单列表
//	@Description	分页查询所有预约订单，支持按多个状态筛选（传多个 status 参数取并集），不传则查全部
//	@Tags			管理员-审核
//	@Produce		json
//	@Param			page		query		int	false	"页码，默认1"
//	@Param			page_size	query		int	false	"每页条数，默认20，最大50"
//	@Param			status		query		[]int	false	"状态筛选（可传多个），不传则查全部"
//	@Success		200			{object}	Response{data=object{list=[]OrderResp,total=int,page=int,page_size=int}}	"订单列表"
//	@Failure		500			{object}	Response	"查询失败"
//	@Security		BearerAuth
//	@Router			/api/admin/orders [get]
func (h *ReviewHandler) GetOrderListHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	statusParams := c.QueryArray("status")

	var orders []*reservationdb.ReservationOrder
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
		log.Printf("[error][review/GetOrderList] 查询失败: %v", err)
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

// GetOrderDetailHandler 获取订单详情（含审核记录）。
//
//	@Summary		获取订单详情
//	@Description	根据订单ID查询订单完整信息，包含关联时段和审核操作流水记录
//	@Tags			管理员-审核
//	@Produce		json
//	@Param			id	path		int	true	"订单ID"
//	@Success		200	{object}	Response{data=object{order=OrderResp,review_records=[]ReviewRecordResp}}	"订单详情"
//	@Failure		400	{object}	Response	"订单不存在"
//	@Security		BearerAuth
//	@Router			/api/admin/orders/{id} [get]
func (h *ReviewHandler) GetOrderDetailHandler(c *gin.Context) {
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
			RoleText:     constants.RoleText(r.ReviewerRole),
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

// Level1ReviewHandler 一级审核操作接口。
//
//	@Summary		一级审核
//	@Description	一级管理员对订单进行审核（通过→待二级审核，拒绝→一级驳回），使用乐观锁防止并发冲突
//	@Tags			管理员-审核
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"订单ID"
//	@Param			body	body		ReviewActionReq	true	"审核操作（action: 1=通过, 2=拒绝）"
//	@Success		200		{object}	Response		"审核成功"
//	@Failure		400		{object}	Response		"参数/状态错误"
//	@Failure		401		{object}	Response		"未登录"
//	@Failure		403		{object}	Response		"权限不足（非一级管理员）"
//	@Security		BearerAuth
//	@Router			/api/admin/review/level1/{id} [post]
func (h *ReviewHandler) Level1ReviewHandler(c *gin.Context) {
	claims, exists := auth.GetAdminInfo(c)
	if !exists {
		unauthorized(c, "未登录")
		return
	}

	if claims.Role != constants.RoleLevel1 {
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

	log.Printf("[info][review/level1] admin_id=%d order_id=%d action=%s", claims.AdminID, id, actionText)
}

// Level2ReviewHandler 二级审核操作接口。
//
//	@Summary		二级审核
//	@Description	二级管理员对已通过一级审核的订单进行终审（通过→终审通过，拒绝→二级驳回），使用乐观锁防止并发冲突
//	@Tags			管理员-审核
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"订单ID"
//	@Param			body	body		ReviewActionReq	true	"审核操作（action: 1=通过, 2=拒绝）"
//	@Success		200		{object}	Response		"审核成功"
//	@Failure		400		{object}	Response		"参数/状态错误"
//	@Failure		401		{object}	Response		"未登录"
//	@Failure		403		{object}	Response		"权限不足（非二级管理员）"
//	@Security		BearerAuth
//	@Router			/api/admin/review/level2/{id} [post]
func (h *ReviewHandler) Level2ReviewHandler(c *gin.Context) {
	claims, exists := auth.GetAdminInfo(c)
	if !exists {
		unauthorized(c, "未登录")
		return
	}

	if claims.Role != constants.RoleLevel2 {
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

	log.Printf("[info][review/level2] admin_id=%d order_id=%d action=%s", claims.AdminID, id, actionText)
}

// SetPasswordHandler 设置门锁密码接口。
//
//	@Summary		设置门锁密码
//	@Description	一级管理员为终审通过的订单中指定时段设置门锁密码（最大20字符）
//	@Tags			管理员-审核
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"订单ID"
//	@Param			slotID	path		int				true	"时段ID"
//	@Param			body	body		SetPasswordReq	true	"门锁密码"
//	@Success		200		{object}	Response		"设置成功"
//	@Failure		400		{object}	Response		"参数/状态错误"
//	@Failure		401		{object}	Response		"未登录"
//	@Security		BearerAuth
//	@Router			/api/admin/review/level1/{id}/slots/{slotID}/password [put]
func (h *ReviewHandler) SetPasswordHandler(c *gin.Context) {
	claims, exists := auth.GetAdminInfo(c)
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

	log.Printf("[info][review/password] admin_id=%d order_id=%d slot_id=%d", claims.AdminID, orderID, slotID)
}

// NotifyHandler 发送审核通过通知（委托给 NotifyHandler 处理）。
//
//	@Summary		发送审核通过通知
//	@Description	一级管理员向终审通过且已设密码的订单用户发送微信模板消息通知
//	@Tags			管理员-通知
//	@Produce		json
//	@Param			id	path		int	true	"订单ID"
//	@Success		200	{object}	service_admin_auth.AdminResp	"通知发送成功"
//	@Failure		400	{object}	service_admin_auth.AdminResp	"订单不存在/状态不允许/未设密码"
//	@Failure		401	{object}	service_admin_auth.AdminResp	"未登录"
//	@Failure		403	{object}	service_admin_auth.AdminResp	"权限不足"
//	@Security		BearerAuth
//	@Router			/api/admin/review/level1/{id}/notify [post]
func (h *ReviewHandler) NotifyHandler(c *gin.Context) {
	h.notifyHdl.NotifyHandler(c)
}

// RejectionNotifyHandler 发送审核驳回通知（委托给 NotifyHandler 处理）。
//
//	@Summary		发送驳回通知
//	@Description	一级管理员向被驳回订单的用户发送微信模板消息，需附带驳回原因
//	@Tags			管理员-通知
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int								true	"订单ID"
//	@Param			body	body		RejectionNotifyReq				true	"驳回原因"
//	@Success		200		{object}	service_admin_auth.AdminResp	"驳回通知发送成功"
//	@Failure		400		{object}	service_admin_auth.AdminResp	"订单不存在/状态不允许"
//	@Failure		401		{object}	service_admin_auth.AdminResp	"未登录"
//	@Failure		403		{object}	service_admin_auth.AdminResp	"权限不足"
//	@Security		BearerAuth
//	@Router			/api/admin/review/level1/{id}/reject-notify [post]
func (h *ReviewHandler) RejectionNotifyHandler(c *gin.Context) {
	h.notifyHdl.RejectionNotifyHandler(c)
}
