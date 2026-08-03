package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// AdminAuth 管理员 JWT 认证中间件。
// 从 Authorization 头提取 Bearer Token，解析 Admin JWT，
// 通过后将 AdminClaims 写入 gin.Context（key: "admin"）。
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未登录，请先登录",
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Token格式错误",
			})
			c.Abort()
			return
		}

		claims, err := jwt.ParseAdminToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  fmt.Sprintf("Token无效或已过期: %v", err),
			})
			c.Abort()
			return
		}

		log.Printf("[info][middleware] admin_id=%d username=%s role=%d auth success",
			claims.AdminID, claims.Username, claims.Role)
		c.Set("admin", claims)
		c.Next()
	}
}

// GetAdminInfo 从 gin.Context 中获取当前管理员 JWT 声明信息。
func GetAdminInfo(c *gin.Context) (*jwt.AdminClaims, bool) {
	val, exists := c.Get("admin")
	if !exists {
		return nil, false
	}
	claims, ok := val.(*jwt.AdminClaims)
	if !ok {
		return nil, false
	}
	return claims, true
}
