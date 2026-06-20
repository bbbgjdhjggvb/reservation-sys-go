// Package main 预约系统的入口程序
package main

import (
	"log"
	"os"
	"time"

	"reservation-sys/pkg/events"
	"reservation-sys/pkg/jwt"
	"reservation-sys/pkg/platform"
	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/pkg/sse"
	"reservation-sys/service/reservation"
	resconfig "reservation-sys/service/reservation/config"
	"reservation-sys/service/reservation/middleware"

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
	svrCfg := resconfig.GetServer()

	gin.SetMode(svrCfg.Mode)

	// 初始化预约数据库（home_res：reservation_orders, reservation_slots, review_records）
	db, err := platform.InitDB(resconfig.GetMySQL())
	if err != nil {
		log.Fatalf("[reservation] 数据库初始化失败: %v", err)
	}

	// 初始化 Redis 客户端连接，给限流中间件使用
	redisCfg := resconfig.GetRedis()
	redisClient, err := platform.InitRedis(redisCfg)
	if err != nil {
		log.Fatalf("[reservation] Redis 初始化失败: %v", err)
	}

	jwtCfg := resconfig.GetJWT()
	jwt.InitUserJWT(jwtCfg.Secret, jwtCfg.ExpireTime)

	// 初始化共享预约数据库模块
	if err := reservationdb.InitModule(db); err != nil {
		log.Fatalf("[reservation] 预约数据库模块初始化失败: %v", err)
	}

	// 初始化预约服务模块
	reservation.InitModule()

	// ========== 初始化 SSE 实时推送 ==========
	// 1. 初始化全局事件发布器（Submit/Cancel 后发布事件到 Redis）
	events.InitPublisher(redisClient)

	// 2. 创建 Redis 事件订阅器（监听其他服务发布的事件）
	sseSubscriber := sse.NewEventSubscriber(redisClient)
	defer sseSubscriber.Close()

	// 3. 创建 SSE Hub（管理客户端连接池，转发 Redis 消息到 SSE 客户端）
	sseHub := sse.NewSSEHub(sseSubscriber)

	resSvc := reservation.GetReservationService()
	resHdl := reservation.NewReservationHandler(resSvc)

	// ========== 启动 HTTP 服务 ==========
	r := gin.Default()
	r.Use(middleware.CORSMiddleware(svrCfg.CORSAllowOrigins))

	api := r.Group("/api/reservation")
	{
		// SSE 端点（无需认证，推送不含敏感数据，参见文档 7.2.2）
		// 客户端通过此端点接收订单变更通知，收到通知后仍需通过 REST API 拉取完整数据
		api.GET("/events", sse.Handler(sseHub))

		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware())
		{
			// 读接口，无需限流
			protected.GET("/reservation/my", resHdl.GetMyReservations)
			protected.GET("/reservation/occupied", resHdl.GetOccupiedSlots)
		}

		// 写接口：按 HandlerName 分组，只挂对应维度的限流中间件
		submitGroup := protected.Group("")
		cancelGroup := protected.Group("")
		for _, rlCfg := range resconfig.GetRateLimits() {
			cfg := middleware.RateLimitConfig{
				Window:      time.Duration(rlCfg.WindowSec) * time.Second,
				MaxRequests: rlCfg.MaxRequests,
				Dimension:   rlCfg.Dimension,
				KeyPrefix:   "ratelimit",
				HandlerName: rlCfg.HandlerName,
				FailOpen:    rlCfg.FailOpen,
			}
			switch rlCfg.HandlerName {
			case "submit":
				submitGroup.Use(middleware.RateLimitMiddleware(redisClient, &cfg))
			case "cancel":
				cancelGroup.Use(middleware.RateLimitMiddleware(redisClient, &cfg))
			}
		}
		submitGroup.POST("/reservation/submit", resHdl.SubmitHandler)
		cancelGroup.DELETE("/reservation/:id", resHdl.Cancel)
	}

	log.Printf("[info] Reservation HTTP service started on port %s", svrCfg.Port)
	if err := r.Run(svrCfg.Port); err != nil {
		log.Fatalf("[error] Server failed: %v", err)
	}
}
