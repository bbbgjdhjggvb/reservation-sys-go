// Package main Swagger 文档聚合服务。
//
// 本服务汇总 reservation、admin、gateway 三个子服务的 API 文档，
// 通过 swagger UI 在统一页面上展示所有接口。
//
//	@title						深圳大学校友会会议室预约系统 API
//	@version					1.0
//	@description				包含用户端预约、管理员审核、微信网关认证三个子系统的接口文档
//	@host						localhost:8083
//	@BasePath					/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				在请求头中添加 Authorization: Bearer <token>，用户端和管理员端使用不同的 JWT
package main

import (
	"log"
	"net/http"

	_ "reservation-sys/service/admin/auth"
	_ "reservation-sys/service/admin/review"
	_ "reservation-sys/service/gateway/auth"
	_ "reservation-sys/service/reservation"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "reservation-sys/docs"
)

func main() {
	r := gin.Default()

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 重定向根路径到 swagger UI
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	log.Printf("[info] Swagger 文档服务启动在端口 :8083")
	log.Printf("[info] 访问 http://localhost:8083/swagger/index.html")
	if err := r.Run(":8083"); err != nil {
		log.Fatalf("[error] Swagger 服务启动失败: %v", err)
	}
}
