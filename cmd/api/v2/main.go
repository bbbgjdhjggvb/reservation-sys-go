// Package main 预约系统的入口程序
// @title           预约系统 API
// @version         2.0
// @description     校友场地预约系统的后端API服务
// @host            localhost:8081
// @BasePath		/api/v2
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     JWT Bearer令牌认证，格式: Bearer token
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

// getConfigPath 获取配置文件路径
// 优先级: 命令行参数 --config > 环境变量 CONFIG_PATH > 默认值
func getConfigPath() string {
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--config" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		return configPath
	}
	return "configs/config_v2.yaml"
}

func main() {
	// 加载 reservation 模块配置
	configPath := getConfigPath()
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

	// CORS 中间件（根据运行模式自动切换策略）
	r.Use(auth.CORSMiddleware(cfg.Server.CORSAllowOrigins))

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
