// Package main 处理微信服务器发送的消息和鉴权
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"reservation-sys/pkg/jwt"
	"reservation-sys/pkg/platform"
	pb "reservation-sys/service/gateway/api/gen/account"
	notifpb "reservation-sys/service/gateway/api/gen/notification"
	"reservation-sys/service/gateway/auth"
	authconfig "reservation-sys/service/gateway/auth/config"
	"reservation-sys/service/gateway/notification"

	"github.com/gin-gonic/gin"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/message"
	"google.golang.org/grpc"
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
	return "service/gateway/configs/config_v1.yaml"
}

func main() {
	configPath := getConfigPath()
	authconfig.Load(configPath)
	cfg := authconfig.Get()

	gin.SetMode(cfg.Server.Mode)

	// 初始化认证数据库（home_xy：users, admins）
	db := platform.InitDB(authconfig.GetMySQL())

	if _, err := platform.InitRedis(authconfig.GetRedis()); err != nil {
		log.Fatalf("[error][platform]: Redis 初始化失败: %v", err)
	}

	jwtCfg := authconfig.GetJWT()
	jwt.InitUserJWT(jwtCfg.Secret, jwtCfg.ExpireTime)

	// 初始化微信 SDK
	wc := wechat.NewWechat()
	// Redis 缓存的一个作用就是存储 access_token
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	redisCache := cache.NewRedis(context.Background(), &cache.RedisOpts{
		Host:     redisAddr,
		Password: cfg.Redis.Password,
		Database: cfg.Redis.DB,
	})
	oa := wc.GetOfficialAccount(&offConfig.Config{
		AppID:     cfg.Wechat.AppID,
		AppSecret: cfg.Wechat.AppSecret,
		Token:     cfg.Wechat.Token,
		Cache:     redisCache,
	})

	// 初始化认证模块（用户 + 管理员）
	auth.InitModule(db, oa, cfg.Wechat.DefaultRedirect, cfg.Wechat.RedirectURLs)
	auth.InitAdminModule(db)

	// 初始化通知模块
	notification.InitModule(auth.GetUserAuthService(), oa, cfg.Wechat.TemplateID)

	notifyHdl := notification.GetNotificationHandler()

	// ========== 启动 gRPC 服务（通知服务 + 账号验证服务） ==========
	grpcLis, err := net.Listen("tcp", cfg.Server.GRPCPort)
	if err != nil {
		log.Fatalf("[error][grpc] 监听失败: %v", err)
	}
	grpcSrv := grpc.NewServer()

	// 注册通知服务
	notifpb.RegisterNotificationServiceServer(grpcSrv, notification.NewGRPCServer(oa, cfg.Wechat.TemplateID))

	// 注册账号验证服务
	adminRepo := auth.NewAdminRepository(db)
	pb.RegisterAccountServiceServer(grpcSrv, auth.NewAccountGRPCServer(adminRepo))

	go func() {
		log.Printf("[info][grpc] Gateway gRPC service started on %s (notification + account)", cfg.Server.GRPCPort)
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatalf("[error][grpc] gRPC serve failed: %v", err)
		}
	}()

	// ==== 启动 HTTP 服务 ====
	r := gin.Default()

	// 处理第一次服务器配置，微信发送的信息
	// 以及用户发送的信息，关注服务号，取消关注等事件
	/*
	 * 为什么要用 Any ?
	 * - 初次服务器配置,微信会发送 GET 请求
	 * - 用户平时发送消息、关注服务号等,发送的是 POST 请求
	 * TODO: 后续将进行拆分
	 */
	r.Any("/wx", func(c *gin.Context) {
		// SDK 自动解析微信发送过来的 XML 数据
		server := oa.GetServer(c.Request, c.Writer)
		// 设置注册时间处理函数
		server.SetMessageHandler(func(msg *message.MixMessage) *message.Reply {
			return notifyHdl.ProcessMessage(oa, msg)
		})
		server.Serve()
		server.Send()
	})

	api := r.Group("/api/v1")
	{
		api.GET("/auth/callback", auth.GetUserAuthHandler().WeChatCallBack)
	}

	log.Printf("[info] WeChat HTTP service started on port %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("[error] Server failed: %v", err)
	}
}
