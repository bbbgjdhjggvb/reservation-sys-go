// Package middleware 提供跨模块共用的 Gin 中间件。
// 统一了原 gateway、reservation、admin 三份重复的中间件实现。
package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// UserAuth 用户 JWT 认证中间件。
// 从 Authorization 头提取 Bearer Token，解析后校验用户 JWT，
// 通过后将 openid 写入 gin.Context（key: "openid"）。
func UserAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未授权，请从服务号进行订阅",
			})
			log.Printf("[info][middleware] 未授权访问")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Token格式错误，应为 Bearer {token}",
			})
			log.Printf("[info][middleware] Token格式错误: authHeader=%s", authHeader)
			c.Abort()
			return
		}

		claims, err := jwt.ParseUserToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  fmt.Sprintf("Token 无效或已过期: %v", err),
			})
			log.Printf("[info][middleware] Token无效或已过期: err=%v", err)
			c.Abort()
			return
		}

		log.Printf("[info][middleware] Token校验通过: openid=%v", claims.OpenID)
		c.Set("openid", claims.OpenID)
		c.Next()
	}
}
