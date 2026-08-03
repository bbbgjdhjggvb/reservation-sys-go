package middleware

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS 根据运行模式动态生成 CORS 中间件。
//   - debug 模式：允许所有来源（方便前端联调）
//   - release 模式：仅允许 allowOrigins 白名单域名
func CORS(allowOrigins []string) gin.HandlerFunc {
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
