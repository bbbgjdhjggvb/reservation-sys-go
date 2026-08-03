package review

import (
	"reservation-sys/internal/auth"
	"reservation-sys/internal/permissions"
	"reservation-sys/internal/router"
	"reservation-sys/internal/sse"
)

// RegisterRoutes 向 Registrar 注册 review 模块的所有 API 端点。
func RegisterRoutes(r *router.Registrar, authH *auth.Handler, revH *Handler, hub *sse.Hub) {
	// 管理员认证（公开端点）
	r.Register("POST", "/api/admin/auth/login", authH.AdminLogin, nil, nil)

	// SSE 实时推送（公开端点）
	r.Register("GET", "/api/admin/events", sse.Handler(hub), nil, nil)

	// 管理员信息
	r.Register("GET", "/api/admin/admin/info", authH.GetAdminInfo, nil, &permissions.AdminAuth)

	// 订单管理
	r.Register("GET", "/api/admin/orders", revH.GetOrderList, nil, &permissions.AdminAuth)
	r.Register("GET", "/api/admin/orders/:id", revH.GetOrderDetail, nil, &permissions.AdminAuth)

	// 一级审核
	r.Register("POST", "/api/admin/review/level1/:id", revH.Level1Review, nil, &permissions.Level1Review)
	r.Register("PUT", "/api/admin/review/level1/:id/slots/:slotID/password", revH.SetPassword, nil, &permissions.Level1Review)
	r.Register("POST", "/api/admin/review/level1/:id/notify", revH.SendApprovalNotify, nil, &permissions.Level1Review)
	r.Register("POST", "/api/admin/review/level1/:id/reject-notify", revH.SendRejectionNotify, nil, &permissions.Level1Review)

	// 二级审核
	r.Register("POST", "/api/admin/review/level2/:id", revH.Level2Review, nil, &permissions.Level2Review)
}
