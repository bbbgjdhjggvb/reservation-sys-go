package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"reservation-sys/internal/auth"
	authStore "reservation-sys/internal/auth/store"

	"reservation-sys/internal/event"
	"reservation-sys/internal/middleware"
	"reservation-sys/internal/permissions"
	"reservation-sys/internal/platform"
	"reservation-sys/internal/reservation"
	resStore "reservation-sys/internal/reservation/store"
	"reservation-sys/internal/review"
	revStore "reservation-sys/internal/review/store"
	"reservation-sys/internal/router"
	"reservation-sys/internal/sse"

	"reservation-sys/pkg/config"
	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
)


func main() {
	// 1. 加载配置
	cfg := config.MustLoad(config.ConfigPath())
	gin.SetMode(cfg.Server.Mode)

	// 2. 初始化基础设施
	db, err := platform.InitDB(&cfg.MySQL)
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	redisClient, err := platform.InitRedis(&cfg.Redis)
	if err != nil {
		log.Fatalf("Redis 初始化失败: %v", err)
	}
	// 初始化 JWT
	jwt.InitUserJWT(cfg.JWT.Secret, cfg.JWT.UserExpireHours)
	jwt.InitAdminJWT(cfg.JWT.Secret, cfg.JWT.AdminExpireHours)

	// 3. 微信 SDK
	wc := wechat.NewWechat()
	redisCache := cache.NewRedis(context.Background(), &cache.RedisOpts{
		Host:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		Database: cfg.Redis.DB,
	})
	oa := wc.GetOfficialAccount(&offConfig.Config{
		AppID:     cfg.Wechat.AppID,
		AppSecret: cfg.Wechat.AppSecret,
		Token:     cfg.Wechat.Token,
		Cache:     redisCache,
	})

	// 4. 事件总线
	bus := event.NewBus()

	// === 组装 auth 模块 ===
	oauthClient := auth.NewWechatOAuthClient(oa)
	authSvc := auth.NewService(
		authStore.NewUserRepository(db),
		authStore.NewAdminRepository(db),
		oauthClient,
		&cfg.Wechat,
	)
	authH := auth.NewHandler(authSvc, oa, &cfg.Wechat)
	notifier := auth.NewNotifier(oa, cfg.Wechat.TemplateID)

	// === 组装 reservation 模块 ===
	slotStore := resStore.NewSlotStore(db)
	resSvc := reservation.NewService(
		resStore.NewOrderStore(db),
		slotStore,
		bus,
	)
	resH := reservation.NewHandler(resSvc)

	// === 组装 review 模块（跨模块接口注入） ===
	revSvc := review.NewService(
		revStore.NewOrderStore(db),
		revStore.NewRecordStore(db),
		slotStore, // reservation/store.SlotStore 实现了 review.SlotStore 接口
		notifier,  // auth.Notifier 实现了 review.Notifier 接口
		bus,
	)
	revH := review.NewHandler(revSvc)

	// === 共享组件 ===
	hub := sse.NewHub(bus)

	// 权限映射器：将 Permission 常量映射为中间件链
	permMapper := permissions.NewMiddlewareMapper(
		middleware.UserAuth(),
		middleware.AdminAuth(),
		middleware.RequireRole(1),
		middleware.RequireRole(2),
	)

	// 限流中间件映射表：根据配置为每个 handler_name 创建限流中间件
	rateLimiters := make(map[string]gin.HandlerFunc, len(cfg.RateLimit))
	for _, rlc := range cfg.RateLimit {
		rateLimiters[rlc.HandlerName] = middleware.RateLimit(redisClient, &middleware.RateLimitConfig{
			Window:      time.Duration(rlc.WindowSec) * time.Second,
			MaxRequests: rlc.MaxRequests,
			Dimension:   rlc.Dimension,
			KeyPrefix:   "rl",
			HandlerName: rlc.HandlerName,
			FailOpen:    rlc.FailOpen,
		})
	}

	// === 路由自注册 ===
	r := router.NewRegistrar(gin.Default(), permMapper, rateLimiters)
	r.Use(middleware.CORS(cfg.Server.CORSAllowOrigins))

	auth.RegisterRoutes(r, authH)
	reservation.RegisterRoutes(r, resH, hub)
	review.RegisterRoutes(r, authH, revH, hub)

	log.Printf("[main] Server starting on %s", cfg.Server.Port)
	if err := r.Engine().Run(cfg.Server.Port); err != nil {
		log.Fatalf("[main] Server failed: %v", err)
	}
}
