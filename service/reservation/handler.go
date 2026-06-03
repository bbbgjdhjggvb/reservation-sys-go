package reservation

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ReservationHandler 处理预约相关的 HTTP 请求
type ReservationHandler struct {
	svc *ReservationService
}

// NewReservationHandler 创建预约处理器实例
func NewReservationHandler(svc *ReservationService) *ReservationHandler {
	return &ReservationHandler{svc: svc}
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

func internalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: msg})
}

// getOpenID 从上下文中获取当前用户的 openid
func getOpenID(c *gin.Context) (string, bool) {
	openid, exists := c.Get("openid")
	if !exists {
		return "", false
	}
	return openid.(string), true
}

// SubmitHandler 处理预约提交请求。
//
//	@Summary		提交预约申请
//	@Description	用户提交预约申请，包含申请人信息和时段列表（1~4个时段），同一天连续时段自动合并
//	@Tags			预约-用户端
//	@Accept			json
//	@Produce		json
//	@Param			body	body		SubmitReq	true	"预约申请信息"
//	@Success		200		{object}	Response{data=OrderResp}	"预约提交成功"
//	@Failure		400		{object}	Response					"参数错误/时段冲突"
//	@Failure		401		{object}	Response					"未授权"
//	@Security		BearerAuth
//	@Router			/api/reservation/reservation/submit [post]
func (h *ReservationHandler) SubmitHandler(c *gin.Context) {
	var req SubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[info][handler/Submit] 参数绑定失败: %v", err)
		badRequest(c, "表单填写有误，请检查")
		return
	}

	slotCount := len(req.Slots)
	if slotCount == 0 {
		badRequest(c, "请至少选择一个时间段")
		return
	}
	if slotCount > 4 {
		badRequest(c, "最多只能选择4个时间段")
		return
	}

	layout := "2006-01-02 15:04:05"
	parsedSlots := make([]ParsedSlot, slotCount)
	for i, slot := range req.Slots {
		st, err1 := time.ParseInLocation(layout, slot.StartTime, time.Local)
		et, err2 := time.ParseInLocation(layout, slot.EndTime, time.Local)
		if err1 != nil || err2 != nil {
			badRequest(c, fmt.Sprintf("第%d个时间段格式错误", i+1))
			return
		}
		if !et.After(st) {
			badRequest(c, fmt.Sprintf("第%d个时间段的结束时间必须晚于开始时间", i+1))
			return
		}
		parsedSlots[i] = ParsedSlot{StartTime: st, EndTime: et}
	}

	openid, exists := getOpenID(c)
	if !exists {
		unauthorized(c, "未授权，请从微信服务号进入")
		return
	}
	log.Printf("[info][handler/Submit] openid=%s, slots=%d", openid, slotCount)

	order, err := h.svc.Submit(openid, parsedSlots, &req)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	fullOrder, loadErr := h.svc.GetOrderByID(order.ID)
	if loadErr != nil {
		okWithMsg(c, fmt.Sprintf("预约提交成功，共%d个时段，请等待审核", slotCount), OrderToResp(order))
		return
	}

	okWithMsg(c, fmt.Sprintf("预约提交成功，共%d个时段，请等待审核", slotCount), OrderToResp(fullOrder))
	log.Printf("[info][handler/Submit] orderNo=%s 提交成功", order.OrderNo)
}

// GetMyReservations 获取当前用户的预约列表。
//
//	@Summary		获取我的预约列表
//	@Description	根据 JWT 中的 openid 查询当前用户的所有预约订单（按创建时间倒序）
//	@Tags			预约-用户端
//	@Produce		json
//	@Success		200	{object}	Response{data=[]OrderResp}	"预约列表"
//	@Failure		401	{object}	Response					"未授权"
//	@Failure		500	{object}	Response					"查询失败"
//	@Security		BearerAuth
//	@Router			/api/reservation/reservation/my [get]
func (h *ReservationHandler) GetMyReservations(c *gin.Context) {
	openid, exists := getOpenID(c)
	if !exists {
		unauthorized(c, "未授权")
		return
	}

	orders, err := h.svc.GetMyReservations(openid)
	if err != nil {
		internalError(c, "查询失败")
		return
	}

	list := make([]*OrderResp, 0, len(orders))
	for _, o := range orders {
		list = append(list, OrderToResp(o))
	}

	ok(c, list)
}

// GetOccupiedSlots 获取指定日期的已占用时间段。
//
//	@Summary		获取已占用时段
//	@Description	查询指定日期内已被预约（待审核/已通过）的时段，用于前端日历展示不可选状态。
//					认证用户会标记 is_mine 字段以区分自己的预约和他人的预约。
//	@Tags			预约-用户端
//	@Produce		json
//	@Param			date	query		string	false	"日期，格式 2006-01-02，默认当天"
//	@Success		200		{object}	Response{data=[]TimeSlotResp}	"已占用时段列表（含 is_mine 字段）"
//	@Failure		400		{object}	Response						"日期格式错误"
//	@Security		BearerAuth
//	@Router			/api/reservation/reservation/occupied [get]
func (h *ReservationHandler) GetOccupiedSlots(c *gin.Context) {
	// 获取查询日期参数，默认为当天
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 从上下文提取当前用户 openid
	// AuthMiddleware 确保此端点已通过认证；
	// 获取失败时传空字符串，服务层会将所有时段 is_mine 设为 false
	openid, _ := getOpenID(c)

	// 调用服务层查询，传入 openid 以标记 is_mine
	slots, err := h.svc.GetOccupiedSlots(date, openid)
	if err != nil {
		log.Printf("[error][handler/GetOccupiedSlots] 查询失败: %v", err)
		badRequest(c, err.Error())
		return
	}

	ok(c, slots)
}

// Cancel 取消预约订单。
//
//	@Summary		取消预约
//	@Description	用户取消自己的预约订单（仅待审核或已通过状态可取消），同时将关联时段标记为已取消
//	@Tags			预约-用户端
//	@Produce		json
//	@Param			id	path		int		true	"订单ID"
//	@Success		200	{object}	Response		"取消成功"
//	@Failure		400	{object}	Response		"订单不存在/状态不允许取消"
//	@Failure		401	{object}	Response		"未授权"
//	@Security		BearerAuth
//	@Router			/api/reservation/reservation/{id} [delete]
func (h *ReservationHandler) Cancel(c *gin.Context) {
	openid, exists := getOpenID(c)
	if !exists {
		unauthorized(c, "未授权")
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		badRequest(c, "无效的预约ID")
		return
	}

	if cancelErr := h.svc.Cancel(uint(id), openid); cancelErr != nil {
		badRequest(c, cancelErr.Error())
		return
	}

	okWithMsg(c, "取消成功", nil)
}
