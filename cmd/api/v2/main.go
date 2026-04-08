// Package main 预约系统的入口程序
package main

import (
	"log"
	"os"

	"reservation-sys/internal/auth"
	"reservation-sys/internal/pkg/jwt"
	"reservation-sys/internal/platform"
	"reservation-sys/internal/reservation"
	resconfig "reservation-sys/internal/reservation/config"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载 reservation 模块配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config_v2.yaml"
	}
	resconfig.Load(configPath)
	cfg := resconfig.Get()

	// 设置 Gin 运行模式
	gin.SetMode(cfg.Server.Mode)

	// 初始化数据库
	db := platform.InitDB(resconfig.GetMySQL())

	// 初始化 JWT（v2 服务需要验证 token）
	jwtCfg := resconfig.GetJWT()
	jwt.Init(jwtCfg.Secret, jwtCfg.ExpireTime)

	// 初始化 reservation 模块
	reservation.InitModule(db)

	resSvc := reservation.GetReservationService()
	resHdl := reservation.NewReservationHandler(resSvc)

	// 初始化路由
	r := gin.Default()

	// 静态资源和模板
	r.Static("/static", "./internal/reservation/frontend/static")
	r.LoadHTMLGlob("internal/reservation/frontend/*.html")

	// 页面路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	// 预约页面入口（从微信服务号重定向过来）
	r.GET("/reserve", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	// API 路由
	api := r.Group("/api/v2")
	{
		// 需要认证的路由
		protected := api.Group("")
		protected.Use(auth.AuthMiddleware())
		{
			protected.POST("/reservation/submit", resHdl.SubmitHandler)
			protected.GET("/reservation/my", resHdl.GetMyReservations)
			protected.GET("/reservation/occupied", resHdl.GetOccupiedSlots)
			protected.DELETE("/reservation/:id", resHdl.Cancel)
		}
	}

	log.Printf("[info] Reservation service started on port %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("[error] Server failed: %v", err)
	}
}
