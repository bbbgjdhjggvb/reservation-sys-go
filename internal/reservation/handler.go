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

// SubmitHandler 处理预约提交请求（支持多时间段批量提交）
// @Summary      提交预约申请（支持多时段）
// @Description  用户提交场地预约申请，支持一次提交1~4个时间段，需要JWT认证
// @Tags         预约管理
// @Accept       json
// @Produce      json
// @Param        Authorization  header    string     true  "Bearer JWT令牌"  default(Bearer )
// @Param        body           body      SubmitReq  true  "预约提交请求（含多个时间段）"
// @Success      200            {object}  Response{data=OrderResp} "预约申请提交成功"
// @Failure      400            {object}  Response                        "请求参数错误"
// @Failure      401            {object}  Response                        "未授权"
// @Security     BearerAuth
// @Router       /reservation/submit [post]
func (h *ReservationHandler) SubmitHandler(c *gin.Context) {
	var req SubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[info][handler/Submit] 参数绑定失败: %v", err)
		badRequest(c, "表单填写有误，请检查")
		return
	}

	// 校验时段数量
	slotCount := len(req.Slots)
	if slotCount == 0 {
		badRequest(c, "请至少选择一个时间段")
		return
	}
	if slotCount > 4 {
		badRequest(c, "最多只能选择4个时间段")
		return
	}

	// 解析所有时间字符串
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

	// 获取用户身份
	openid, exists := getOpenID(c)
	if !exists {
		unauthorized(c, "未授权，请从微信服务号进入")
		return
	}
	log.Printf("[info][handler/Submit] openid=%s, slots=%d", openid, slotCount)

	// 调用业务层提交
	order, err := h.svc.Submit(openid, parsedSlots, &req)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	// 重新查询以加载关联的Slots（事务创建后GORM不会自动Preload）
	fullOrder, loadErr := h.svc.GetOrderByID(order.ID)
	if loadErr != nil {
		// 即使加载失败也不影响返回，直接用order本身转换
		okWithMsg(c, fmt.Sprintf("预约提交成功，共%d个时段，请等待审核", slotCount), order.ToOrderResp())
		return
	}

	okWithMsg(c, fmt.Sprintf("预约提交成功，共%d个时段，请等待审核", slotCount), fullOrder.ToOrderResp())
	log.Printf("[info][handler/Submit] orderNo=%s 提交成功", order.OrderNo)
}

// GetMyReservations 获取当前用户的预约列表
// @Summary      获取我的预约列表
// @Description  获取当前登录用户的所有预约订单（包含各时段明细），需要JWT认证
// @Tags         预约管理
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer JWT令牌"  default(Bearer )
// @Success      200            {object}  Response{data=[]OrderResp}  "查询成功"
// @Failure      401            {object}  Response                  "未授权"
// @Failure      500            {object}  Response                  "服务器内部错误"
// @Security     BearerAuth
// @Router       /reservation/my [get]
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
		list = append(list, o.ToOrderResp())
	}

	ok(c, list)
}

// GetOccupiedSlots 获取指定日期的已占用时间段
// @Summary      获取已占用时间段
// @Description  查询指定日期已被预约占用的时间段，需要JWT认证
// @Tags         预约管理
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer JWT令牌"  default(Bearer )
// @Param        date           query     string  false "查询日期(格式: 2026-01-01)"  example(2026-01-01)
// @Success      200            {object}  Response{data=[]TimeSlotResp}  "查询成功"
// @Failure      400            {object}  Response                       "请求参数错误"
// @Security     BearerAuth
// @Router       /reservation/occupied [get]
func (h *ReservationHandler) GetOccupiedSlots(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	slots, err := h.svc.GetOccupiedSlots(date)
	if err != nil {
		log.Printf("[error][handler/GetOccupiedSlots] 查询失败: %v", err)
		badRequest(c, err.Error())
		return
	}

	ok(c, slots)
}

// Cancel 取消预约订单（按orderID）
// @Summary      取消预约
// @Description  取消指定的预约订单（取消所有关联的时段），仅预约人本人可操作，需要JWT认证
// @Tags         预约管理
// @Produce      json
// @Param        Authorization  header    string  true  "Bearer JWT令牌"  default(Bearer )
// @Param        id             path      int     true  "订单ID"         example(1)
// @Success      200            {object}  Response "取消成功"
// @Failure      400            {object}  Response "请求参数错误"
// @Failure      401            {object}  Response "未授权"
// @Security     BearerAuth
// @Router       /reservation/{id} [delete]
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
