package auth

import (
	"fmt"
	"log"
	"net/http"
	"reservation-sys/pkg/jwt"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware 用户 JWT 认证中间件。
// 从 Authorization 头提取 Bearer Token，解析后校验用户 JWT，
// 通过后将 openid 写入 gin.Context（key: "openid"）。
// 未通过验证时返回 401 响应并中止请求链。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未授权，请从服务号进行订阅",
			})
			log.Printf("[info][auth][middleware]: 未授权访问")
			c.Abort() // 拦截请求，不再往下执行
			return
		}

		parts := strings.SplitN(authHeader, " ", 2) // 最多分割出两部分
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Token格式错误，应为 Bearer {token}",
			})
			log.Printf("[info][auth][middleware]: Token格式错误: authHeader=%s", authHeader)
			c.Abort()
			return
		}

		// 解析并校验 Token
		claims, err := jwt.ParseUserToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":  401,
				"msg":   fmt.Sprintf("Token 无效或已过期: %v", err),
			})
			log.Printf("[info][auth][middleware]: Token无效或已过期: token=%v, err=%v", parts[1], err)
			c.Abort()
			return
		}

		// 校验通过，解析出 OpenID,塞入上下文
		log.Printf("[info][auth][middleware]: Token校验通过: openid=%v", claims.OpenID)
		c.Set("openid", claims.OpenID)

		// 继续执行
		c.Next()
	}
}

// AdminAuthMiddleware 管理员认证中间件（基于 Admin JWT Token）
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, AdminResp{Code: 401, Msg: "未登录，请先登录"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, AdminResp{Code: 401, Msg: "Token格式错误"})
			c.Abort()
			return
		}

		claims, err := jwt.ParseAdminToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, AdminResp{Code: 401, Msg: fmt.Sprintf("Token无效或已过期: %v", err)})
			c.Abort()
			return
		}

		log.Printf("[info][auth/admin] admin_id=%d username=%s role=%d auth success", claims.AdminID, claims.Username, claims.Role)
		c.Set("admin", claims)
		c.Next()
	}
}

// RoleMiddleware 角色校验中间件工厂
func RoleMiddleware(allowedRoles ...int) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("admin")
		if !exists {
			c.JSON(http.StatusUnauthorized, AdminResp{Code: 401, Msg: "未登录"})
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

		c.JSON(http.StatusForbidden, AdminResp{Code: 403, Msg: "无权限操作"})
		c.Abort()
	}
}

// GetAdminInfo 从上下文中获取当前管理员信息
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

// CORSMiddleware 根据运行模式动态生成 CORS 中间件
//   - debug 模式：允许所有来源（方便 Swagger UI / 前端联调）
//   - release 模式：仅允许 allowOrigins 白名单域名
func CORSMiddleware(allowOrigins []string) gin.HandlerFunc {
	if gin.Mode() == gin.DebugMode {
		log.Println("[cors] debug模式：允许所有来源跨域访问")
		return cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: false,
			MaxAge:           12 * time.Hour,
		})
	}

	if len(allowOrigins) == 0 {
		allowOrigins = []string{}
	}
	log.Printf("[cors] release模式：允许来源 %v\n", allowOrigins)
	return cors.New(cors.Config{
		AllowOrigins:     allowOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
