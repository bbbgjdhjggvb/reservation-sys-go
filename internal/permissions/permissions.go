// Package permissions 提供 API 权限枚举和中间件映射。
// 各模块在 routes.go 中通过 Register(..., &permissions.Xxx) 显式声明 API 所需权限。
package permissions

import "github.com/gin-gonic/gin"

// Permission 表示一组中间件链的标识符
type Permission int

// 使用 var 而非 const，以便取地址传给 Register
var (
	UserAuth     = Permission(1) // 用户 JWT
	AdminAuth    = Permission(2) // 管理员 JWT
	Level1Review = Permission(3) // 管理员 JWT + 一级审核角色
	Level2Review = Permission(4) // 管理员 JWT + 二级审核角色
)

// MiddlewareMapper 将 Permission 常量映射为对应的 Gin 中间件链
type MiddlewareMapper struct {
	userAuth     gin.HandlerFunc
	adminAuth    gin.HandlerFunc
	requireRole1 gin.HandlerFunc
	requireRole2 gin.HandlerFunc
}

// NewMiddlewareMapper 创建权限中间件映射器
func NewMiddlewareMapper(
	userAuth, adminAuth, requireRole1, requireRole2 gin.HandlerFunc,
) *MiddlewareMapper {
	return &MiddlewareMapper{
		userAuth:     userAuth,
		adminAuth:    adminAuth,
		requireRole1: requireRole1,
		requireRole2: requireRole2,
	}
}

// MiddlewareFor 返回 Permission 对应的中间件链
func (m *MiddlewareMapper) MiddlewareFor(p Permission) []gin.HandlerFunc {
	switch p {
	case UserAuth:
		return []gin.HandlerFunc{m.userAuth}
	case AdminAuth:
		return []gin.HandlerFunc{m.adminAuth}
	case Level1Review:
		return []gin.HandlerFunc{m.adminAuth, m.requireRole1}
	case Level2Review:
		return []gin.HandlerFunc{m.adminAuth, m.requireRole2}
	default:
		return nil
	}
}
