// Package router 提供路由注册器，封装 Gin Engine 供各模块自注册 API。
//
// 用法示例:
//
//	// 公开端点
//	r.Register("GET", "/wx", h.WeChatHandler, nil, nil)
//
//	// 需用户认证的端点
//	r.Register("GET", "/api/reservation/my", h.MyOrders, nil, &permissions.UserAuth)
//
//	// 需认证 + 限流的端点
//	r.Register("POST", "/api/reservation/submit", h.Submit, Limiter("submit"), &permissions.UserAuth)
package router

import (
	"reservation-sys/internal/permissions"

	"github.com/gin-gonic/gin"
)

// Registrar 路由注册器，封装 Gin Engine、权限映射器和限流中间件。
// 各模块的 routes.go 通过 Register() 向 Registrar 注册自己的 API。
type Registrar struct {
	engine       *gin.Engine
	perm         *permissions.MiddlewareMapper
	rateLimiters map[string]gin.HandlerFunc
}

// NewRegistrar 创建路由注册器。
func NewRegistrar(
	engine *gin.Engine,
	perm *permissions.MiddlewareMapper,
	limiters map[string]gin.HandlerFunc,
) *Registrar {
	if limiters == nil {
		limiters = make(map[string]gin.HandlerFunc)
	}
	return &Registrar{engine: engine, perm: perm, rateLimiters: limiters}
}

// Engine 返回底层 Gin Engine，用于启动 HTTP 服务。
func (r *Registrar) Engine() *gin.Engine { return r.engine }

// Use 添加全局中间件（如 CORS）。
func (r *Registrar) Use(handlers ...gin.HandlerFunc) {
	r.engine.Use(handlers...)
}

// Limiter 返回字符串 s 的指针，用于 Register 的 rateLimitKey 参数。
// Limiter("submit") → 启用 handler_name="submit" 对应的限流中间件。
func Limiter(s string) *string { return &s }

// Register 注册一个 API 端点。
// 可选参数均为指针，nil 表示不启用：rateLimitKey=nil 不限流，perm=nil 公开端点。
func (r *Registrar) Register(
	method, path string,
	handler gin.HandlerFunc,
	rateLimitKey *string,
	perm *permissions.Permission,
) {
	var chain []gin.HandlerFunc

	// 1. 权限中间件（先鉴权）
	if perm != nil {
		chain = append(chain, r.perm.MiddlewareFor(*perm)...)
	}

	// 2. 限流中间件（鉴权后限流）
	if rateLimitKey != nil {
		if limiter, ok := r.rateLimiters[*rateLimitKey]; ok {
			chain = append(chain, limiter)
		}
	}

	// 3. 业务 handler
	chain = append(chain, handler)
	r.engine.Handle(method, path, chain...)
}
