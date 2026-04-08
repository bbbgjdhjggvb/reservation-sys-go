package reservation

import (
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

// SubmitHandler 处理预约提交请求
func (h *ReservationHandler) SubmitHandler(c *gin.Context) {
	var req SubmitReq
	// 绑定并校验前端传递的数据
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[info][reservation/handler/SubmitHandler]: 无法绑定表单")
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"msg":   "表单填写有误请检查",
			"error": err.Error(),
		})
		return
	}

	// 解析时间字符串
	// 前端传入的时间格式为 "2026-01-01 14:00:00"
	layout := "2006-01-02 15:04:05"
	startTime, err1 := time.ParseInLocation(layout, req.StartTime, time.Local)
	endTime, err2 := time.ParseInLocation(layout, req.EndTime, time.Local)
	if err1 != nil || err2 != nil || !endTime.After(startTime) {
		log.Printf("[info][reservation/handler/SubmitHandler]: 时间格式错误或者结束时间不晚于开始时间")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "时间格式错误或结束时间晚于开始时间",
		})
		return
	}

	// 获取当前用户的openid
	// 这个请求处理接口需要鉴权，OpenID 应该是我们从 JWT Token 中解析出来放在 Context 里面的
	openid, exists := c.Get("openid")
	if !exists {
		log.Printf("[info][reservation/handler/SubmitHandler]: Context 中没有 openid")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未授权，请从微信服务号里预约",
		})
		return
	}
	log.Printf("[info][reservation/handler/SubmitHandler]: 从 Context 中获取到 openid: %s", openid.(string))

	res, err := h.svc.Submit(openid.(string), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}

	// 返回成功信息
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "预约申请提交成功，请等待审核",
		"data": res.ToResp(),
	})
	log.Printf("[info][reservation/handler/SubmitHandler]: %s的预约提交成功", openid.(string))
}

// GetMyReservations 获取当前用户的预约列表
func (h *ReservationHandler) GetMyReservations(c *gin.Context) {
	openid, exists := c.Get("openid")
	if !exists {
		log.Printf("[error][reservation/handler/GetMyReservations]: Context 中没有 openid")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未授权",
		})
		return
	}
	log.Printf("[info][reservation/handler/GetMyReservations]: 从 Context 中获取到 openid: %s", openid.(string))

	reservations, err := h.svc.GetMyReservations(openid.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "查询失败",
		})
		return
	}

	// 转换为响应格式
	list := make([]*ReservationResp, 0, len(reservations))
	for _, r := range reservations {
		list = append(list, r.ToResp())
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": list,
	})
}

// GetOccupiedSlots 获取指定日期的已占用时间段
func (h *ReservationHandler) GetOccupiedSlots(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	slots, err := h.svc.GetOccupiedSlots(date)
	if err != nil {
		log.Printf("[error][reservation/handler/GetOccupiedSlots]: 查询占用时段失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": slots,
	})
}

// Cancel 取消预约
func (h *ReservationHandler) Cancel(c *gin.Context) {
	openid, exists := c.Get("openid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "未授权",
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "无效的预约ID",
		})
		return
	}

	if err := h.svc.Cancel(uint(id), openid.(string)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "取消成功",
	})
}
