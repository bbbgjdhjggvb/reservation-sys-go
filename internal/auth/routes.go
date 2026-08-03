package auth

import (
	"reservation-sys/internal/router"
)

// RegisterRoutes 向 Registrar 注册 auth 模块的网关认证 API 端点。
// 管理员认证路由（/api/admin/auth/*）由 review 模块统一注册。
func RegisterRoutes(r *router.Registrar, h *Handler) {
	// 微信消息入口（GET 用于服务器验证，POST 用于接收消息/事件）
	r.Register("GET", "/wx", h.WeChatHandler, nil, nil)
	r.Register("POST", "/wx", h.WeChatHandler, nil, nil)

	// 微信 OAuth 回调
	r.Register("GET", "/api/gateway/auth/callback", h.WeChatCallBack, nil, nil)
}
