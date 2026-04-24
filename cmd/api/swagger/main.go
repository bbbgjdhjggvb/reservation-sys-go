// Package main Swagger 文档独立服务
//
// 独立运行 Swagger 文档服务，与业务 API 服务解耦
// 默认监听 :8083，访问 http://localhost:8083/swagger/index.html 查看文档
package main

import (
	"log"
	"os"

	_ "reservation-sys/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Swagger 文档路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "swagger-doc"})
	})

	port := os.Getenv("SWAGGER_PORT")
	if port == "" {
		port = ":8083"
	}
	if port[0] != ':' {
		port = ":" + port
	}

	log.Printf("[info] Swagger docs service starting on http://localhost%s/swagger/index.html", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("[error] Swagger docs service failed: %v", err)
	}
}
