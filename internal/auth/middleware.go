package auth

import (
	"net/http"
	"reservation-sys/internal/pkg/jwt"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未授权，请从服务号进行订阅",
			})
			c.Abort() // 拦截请求，不再往下执行
			return
		}

		parts := strings.SplitN(authHeader, " ", 2) // 最多分割出两部分
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Token格式错误，应为 Bearer {token}",
			})
			c.Abort()
			return
		}

		// 解析并校验 Token
		claims, err := jwt.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Token 无效或已过期，重新进入预约页面进行预约",
			})
			c.Abort()
			return
		}

		// 校验通过，解析出 OpenID,塞入上下文
		c.Set("openid", claims.OpenID)

		// 继续执行
		c.Next()
	}
}
