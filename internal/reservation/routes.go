package reservation

import (
	"reservation-sys/internal/permissions"
	"reservation-sys/internal/router"
	"reservation-sys/internal/sse"
)

// RegisterRoutes 向 Registrar 注册 reservation 模块的所有 API 端点。
func RegisterRoutes(r *router.Registrar, h *Handler, hub *sse.Hub) {
	// SSE 实时推送（公开端点）
	r.Register("GET", "/api/reservation/events", sse.Handler(hub), nil, nil)

	// 用户读接口
	r.Register("GET", "/api/reservation/reservation/my", h.GetMyReservations, nil, &permissions.UserAuth)
	r.Register("GET", "/api/reservation/reservation/occupied", h.GetOccupiedSlots, nil, &permissions.UserAuth)

	// 用户写接口（限流键对应 ratelimit 配置中的 handler_name）
	r.Register("POST", "/api/reservation/reservation/submit", h.Submit, router.Limiter("submit"), &permissions.UserAuth)
	r.Register("DELETE", "/api/reservation/reservation/:id", h.Cancel, router.Limiter("cancel"), &permissions.UserAuth)
}
