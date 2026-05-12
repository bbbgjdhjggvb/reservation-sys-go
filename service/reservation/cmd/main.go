// Package main 预约系统的入口程序
package main

import (
	"log"
	"os"

	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/service/reservation"
	resconfig "reservation-sys/service/reservation/config"
	"reservation-sys/service/reservation/middleware"
	"reservation-sys/pkg/jwt"
	"reservation-sys/pkg/platform"

	"github.com/gin-gonic/gin"
)

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--config" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		return configPath
	}
	return "service/reservation/configs/config_v2.yaml"
}

func main() {
	configPath := getConfigPath()
	resconfig.Load(configPath)
	cfg := resconfig.Get()

	gin.SetMode(cfg.Server.Mode)

	// 初始化预约数据库（home_res：reservation_orders, reservation_slots, review_records）
	db := platform.InitDB(resconfig.GetMySQL())

	jwtCfg := resconfig.GetJWT()
	jwt.InitUserJWT(jwtCfg.Secret, jwtCfg.ExpireTime)

	// 初始化共享预约数据库模块
	reservationdb.InitModule(db)

	// 初始化预约服务模块
	reservation.InitModule()

	resSvc := reservation.GetReservationService()
	resHdl := reservation.NewReservationHandler(resSvc)

	// ========== 启动 HTTP 服务（不再提供 gRPC） ==========
	r := gin.Default()
	r.Use(middleware.CORSMiddleware(cfg.Server.CORSAllowOrigins))

	r.LoadHTMLGlob("service/reservation/frontend/*.html")

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})
	r.GET("/reserve", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})
	r.GET("/myorders", func(c *gin.Context) {
		c.HTML(200, "myorders.html", nil)
	})

	api := r.Group("/api/v2")
	{
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.POST("/reservation/submit", resHdl.SubmitHandler)
			protected.GET("/reservation/my", resHdl.GetMyReservations)
			protected.GET("/reservation/occupied", resHdl.GetOccupiedSlots)
			protected.DELETE("/reservation/:id", resHdl.Cancel)
		}
	}

	log.Printf("[info] Reservation HTTP service started on port %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("[error] Server failed: %v", err)
	}
}
