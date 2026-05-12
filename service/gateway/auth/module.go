package auth

import (
	"reservation-sys/pkg/platform"

	"github.com/silenceper/wechat/v2/officialaccount"
	"gorm.io/gorm"
)

var (
	instance       *UserAuthService
	handler        *UserAuthHandler
	adminService   *AdminAuthService
	adminHandler   *AdminAuthHandler
)

// InitModule 初始化用户认证模块
func InitModule(db *gorm.DB, oa *officialaccount.OfficialAccount, defaultRedirect string, redirectURLs map[string]string) {
	// 自动迁移表结构
	platform.AutoMigrate(db, &User{})

	repo := NewUserRepository(db)
	oauth := NewWechatOAuthClient(oa)
	provider := NewWechatUserInfoProvider(oa)
	instance = NewUserAuthServiceWithUserInfo(repo, oauth, provider)
	handler = NewUserAuthHandler(instance, defaultRedirect, redirectURLs)
}

// InitAdminModule 初始化管理员认证模块
func InitAdminModule(db *gorm.DB) {
	platform.AutoMigrate(db, &Admin{})

	adminRepo := NewAdminRepository(db)
	adminService = NewAdminAuthService(adminRepo)
	adminHandler = NewAdminAuthHandler(adminService)
}

func GetUserAuthService() *UserAuthService {
	if instance == nil {
		panic("auth module not initialized")
	}

	return instance
}

func GetUserAuthHandler() *UserAuthHandler {
	if handler == nil {
		panic("auth module not initialized")
	}

	return handler
}

// GetAdminAuthService 获取管理员认证服务实例
func GetAdminAuthService() *AdminAuthService {
	if adminService == nil {
		panic("admin auth module not initialized")
	}
	return adminService
}

// GetAdminAuthHandler 获取管理员认证处理器实例
func GetAdminAuthHandler() *AdminAuthHandler {
	if adminHandler == nil {
		panic("admin auth module not initialized")
	}
	return adminHandler
}
