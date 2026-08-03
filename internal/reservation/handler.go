package reservation

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	ierr "reservation-sys/internal/errors"

	"github.com/gin-gonic/gin"
)

// ========== Handler ==========

// Handler 预约模块 HTTP 处理器
type Handler struct {
	svc *Service
}

// NewHandler 创建预约处理器实例
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

func internalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, Response{Code: 500, Msg: msg})
}

func getOpenID(c *gin.Context) (string, bool) {
	openid, exists := c.Get("openid")
	if !exists {
		return "", false
	}
	return openid.(string), true
}

// ========== Handler 方法 ==========

// Submit 处理 POST /api/reservation/reservation/submit
func (h *Handler) Submit(c *gin.Context) {
	var req SubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[info][reservation/handler] 参数绑定失败: %v", err)
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
	log.Printf("[info][reservation/handler] openid=%s, slots=%d", openid, slotCount)

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
	log.Printf("[info][reservation/handler] orderNo=%s 提交成功", order.OrderNo)
}

// GetMyReservations 处理 GET /api/reservation/reservation/my
func (h *Handler) GetMyReservations(c *gin.Context) {
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

// GetOccupiedSlots 处理 GET /api/reservation/reservation/occupied
func (h *Handler) GetOccupiedSlots(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	openid, _ := getOpenID(c)

	slots, err := h.svc.GetOccupiedSlots(date, openid)
	if err != nil {
		log.Printf("[error][reservation/handler] 查询失败: %v", err)
		badRequest(c, err.Error())
		return
	}

	ok(c, slots)
}

// Cancel 处理 DELETE /api/reservation/reservation/:id
func (h *Handler) Cancel(c *gin.Context) {
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

// errorCode 将业务错误映射为 HTTP 状态码。
func errorCode(err error) int {
	switch {
	case errors.Is(err, ErrSlotOccupied):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidStatus):
		return http.StatusUnprocessableEntity
	case errors.Is(err, ErrDailyLimitReached):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrOrderNotFound),
		errors.Is(err, ierr.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ierr.ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ierr.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ierr.ErrForbidden):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
