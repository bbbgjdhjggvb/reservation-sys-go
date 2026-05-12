// Package main 管理员审核系统的入口程序
package main

import (
	"log"
	"os"

	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/pkg/grpc"
	"reservation-sys/pkg/jwt"
	"reservation-sys/pkg/platform"
	"reservation-sys/service/admin/auth"
	adminconfig "reservation-sys/service/admin/config"
	"reservation-sys/service/admin/review"

	pb "reservation-sys/service/gateway/api/gen/account"
	notifpb "reservation-sys/service/gateway/api/gen/notification"

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
	return "service/admin/configs/config_v3.yaml"
}

func main() {
	configPath := getConfigPath()
	adminconfig.Load(configPath)
	cfg := adminconfig.Get()

	gin.SetMode(cfg.Server.Mode)

	// 初始化预约数据库（home_res：reservation_orders, reservation_slots, review_records）
	db := platform.InitDB(adminconfig.GetMySQL())

	// 初始化共享预约数据库模块
	reservationdb.InitModule(db)

	// 初始化 JWT 配置（本地生成 admin token）
	jwtCfg := adminconfig.GetJWT()
	jwt.InitAdminJWT(jwtCfg.Secret, jwtCfg.ExpireTime)

	// ========== 连接 Gateway gRPC 服务（通知 + 账号验证） ==========
	gatewayConn := grpc.Connect(cfg.GRPC.GatewayAddr)

	// 账号验证客户端
	accountClient := pb.NewAccountServiceClient(gatewayConn)

	// 通知服务客户端
	notifyClient := notifpb.NewNotificationServiceClient(gatewayConn)

	// ========== 初始化认证模块（通过 gRPC 验证管理员凭证） ==========
	auth.InitModule(accountClient)

	// ========== 初始化审核模块 ==========
	review.InitModule(notifyClient)

	reviewSvc := review.GetReviewService()
	repo := reservationdb.GetRepository()
	notifyHdl := review.NewNotifyHandler(notifyClient, repo)
	hdl := review.NewReviewHandler(reviewSvc, notifyHdl)
	adminHdl := auth.GetAdminAuthHandler()

	// ========== 启动 HTTP 服务 ==========
	r := gin.Default()
	r.Use(auth.CORSMiddleware(cfg.Server.CORSAllowOrigins))

	r.LoadHTMLGlob("service/admin/frontend/*.html")

	r.GET("/admin", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.GET("/admin/dashboard", func(c *gin.Context) {
		c.HTML(200, "dashboard.html", nil)
	})

	api := r.Group("/api/admin")
	{
		api.POST("/auth/login", adminHdl.LoginHandler)

		protected := api.Group("")
		protected.Use(auth.AdminAuthMiddleware())
		{
			protected.GET("/admin/info", adminHdl.GetAdminInfoHandler)

			protected.GET("/orders", hdl.GetOrderListHandler)
			protected.GET("/orders/:id", hdl.GetOrderDetailHandler)

			level1 := protected.Group("/review/level1")
			level1.Use(auth.RoleMiddleware(1))
			{
				level1.POST("/:id", hdl.Level1ReviewHandler)
				level1.PUT("/:id/slots/:slotID/password", hdl.SetPasswordHandler)
				level1.POST("/:id/notify", hdl.NotifyHandler)
				level1.POST("/:id/reject-notify", hdl.RejectionNotifyHandler)
			}

			level2 := protected.Group("/review/level2")
			level2.Use(auth.RoleMiddleware(2))
			{
				level2.POST("/:id", hdl.Level2ReviewHandler)
			}
		}
	}

	log.Printf("[info] Review HTTP service started on port %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("[error] Server failed: %v", err)
	}
}
