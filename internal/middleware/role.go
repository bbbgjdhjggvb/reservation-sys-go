package middleware

import (
	"net/http"

	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// RequireRole 角色校验中间件工厂。
// 返回一个中间件，仅允许指定角色的管理员通过。
// 必须在 AdminAuth 中间件之后使用。
func RequireRole(allowedRoles ...int) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("admin")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未登录",
			})
			c.Abort()
			return
		}
		claims := val.(*jwt.AdminClaims)

		for _, role := range allowedRoles {
			if claims.Role == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"code": 403,
			"msg":  "无权限操作",
		})
		c.Abort()
	}
}
