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

// AdminAuthMiddleware 管理员认证中间件（基于 Admin JWT Token）。
//
// 验证流程:
//  1. 从 Authorization 头提取 Bearer Token
//  2. 解析并校验 Admin JWT Token
//  3. 将 AdminClaims 写入 gin.Context（key: "admin"）
//
// 未通过验证时直接返回 401 响应并中止请求链
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

// RoleMiddleware 角色校验中间件工厂。
//
// 参数:
//   - allowedRoles: 允许通过的角色ID列表（如 1=一级管理员, 2=二级管理员）
//
// 返回值:
//   - gin.HandlerFunc: Gin 中间件函数，角色不匹配时返回 403
//
// 注意: 必须在 AdminAuthMiddleware 之后使用（依赖 Context 中的 "admin" 字段）
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

// GetAdminInfo 从 gin.Context 中获取当前管理员信息。
//
// 参数:
//   - c: Gin 上下文（需经过 AdminAuthMiddleware 注入 "admin" 字段）
//
// 返回值:
//   - *jwt.AdminClaims: 管理员 JWT 声明信息
//   - bool: 是否成功获取（未登录或类型断言失败时返回 false）
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

// CORSMiddleware 根据运行模式动态生成 CORS 中间件。
//   - debug 模式：允许所有来源跨域访问（方便前端联调）
//   - release 模式：仅允许 allowOrigins 白名单域名，且允许携带凭证
//
// 参数:
//   - allowOrigins: release 模式下允许的来源域名列表
//
// 返回值:
//   - gin.HandlerFunc: CORS 中间件
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
