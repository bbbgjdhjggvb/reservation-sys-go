// Package main 处理微信服务器发送的消息和鉴权
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"reservation-sys/internal/auth"
	authconfig "reservation-sys/internal/auth/config"
	"reservation-sys/internal/notification"
	"reservation-sys/internal/pkg/jwt"
	"reservation-sys/internal/platform"

	"github.com/gin-gonic/gin"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/message"
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
	return "configs/config_v1.yaml"
}

func main() {
	// 加载 auth 模块配置
	configPath := getConfigPath()
	authconfig.Load(configPath)
	cfg := authconfig.Get()

	// 设置 Gin 运行模式
	gin.SetMode(cfg.Server.Mode)

	// 初始化数据库
	db := platform.InitDB(authconfig.GetMySQL())

	// 初始化 Redis 缓存
	if _, err := platform.InitRedis(authconfig.GetRedis()); err != nil {
		log.Fatalf("[error][platform]: Redis 初始化失败: %v", err)
	}

	// 初始化 JWT
	jwtCfg := authconfig.GetJWT()
	jwt.Init(jwtCfg.Secret, jwtCfg.ExpireTime)

	// 初始化微信 SDK
	wc := wechat.NewWechat()
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

	// 初始化 auth 模块
	auth.InitModule(db, oa, cfg.Wechat.FrontendURL)

	authHdl := auth.GetUserAuthHandler()
	authSvc := auth.GetUserAuthService()

	notifySvc := notification.NewNotificationService(authSvc)
	notifyHdl := notification.NewNotificationHandler(notifySvc)

	// 初始化路由
	r := gin.Default()

	// 微信服务器消息入口
	r.Any("/wx", func(c *gin.Context) {
		server := oa.GetServer(c.Request, c.Writer)
		server.SetMessageHandler(func(msg *message.MixMessage) *message.Reply {
			return notifyHdl.ProcessMessage(oa, msg)
		})
		server.Serve()
		server.Send()
	})

	// API 路由
	api := r.Group("/api/v1")
	{
		// 微信服务器会在用户完成授权后重定向到该地址，并自动凭借 code 参数
		api.GET("/auth/callback", authHdl.WeChatCallBack)
	}

	log.Printf("[info] WeChat service started on port %s", cfg.Server.Port)
	if err := r.Run(cfg.Server.Port); err != nil {
		log.Fatalf("[error] Server failed: %v", err)
	}
}
