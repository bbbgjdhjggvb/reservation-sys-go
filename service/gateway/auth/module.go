package auth

import (
	"github.com/silenceper/wechat/v2/officialaccount"
	"gorm.io/gorm"
)

var (
	instance     *UserAuthService
	handler      *UserAuthHandler
	adminService *AdminAuthService
	adminHandler *AdminAuthHandler
)

// InitModule 初始化用户认证模块（用户 OAuth 登录 + JWT 签发）。
//
// 数据库表结构由 deploy/mysql/init.sql 管理，不使用 GORM AutoMigrate。
//
// 参数:
//   - db: GORM 数据库实例
//   - oa: 微信公众号实例（用于 OAuth 换取 openid）
//   - defaultRedirect: OAuth 回调后的默认重定向地址
//   - redirectURLs: state 到重定向地址的映射（用于多页面跳转）
func InitModule(db *gorm.DB, oa *officialaccount.OfficialAccount, defaultRedirect string, redirectURLs map[string]string) error {
	// 数据库表结构由 deploy/mysql/init.sql 管理，不使用 GORM AutoMigrate

	repo := NewUserRepository(db)
	oauth := NewWechatOAuthClient(oa)
	provider := NewWechatUserInfoProvider(oa)
	instance = NewUserAuthServiceWithUserInfo(repo, oauth, provider)
	handler = NewUserAuthHandler(instance, defaultRedirect, redirectURLs)
	return nil
}

// InitAdminModule 初始化管理员认证模块（管理员登录 + JWT 签发）。
//
// 数据库表结构由 deploy/mysql/init.sql 管理，不使用 GORM AutoMigrate。
//
// 参数:
//   - db: GORM 数据库实例
func InitAdminModule(db *gorm.DB) error {
	// 数据库表结构由 deploy/mysql/init.sql 管理，不使用 GORM AutoMigrate

	adminRepo := NewAdminRepository(db)
	adminService = NewAdminAuthService(adminRepo)
	adminHandler = NewAdminAuthHandler(adminService)
	return nil
}

// GetUserAuthService 获取用户认证服务实例。
//
// 返回值:
//   - *UserAuthService: 用户认证服务实例（未初始化时 panic）
func GetUserAuthService() *UserAuthService {
	if instance == nil {
		panic("auth module not initialized")
	}

	return instance
}

// GetUserAuthHandler 获取用户认证处理器实例。
//
// 返回值:
//   - *UserAuthHandler: 用户认证处理器实例（未初始化时 panic）
func GetUserAuthHandler() *UserAuthHandler {
	if handler == nil {
		panic("auth module not initialized")
	}

	return handler
}

// GetAdminAuthService 获取管理员认证服务实例。
//
// 返回值:
//   - *AdminAuthService: 管理员认证服务实例（未初始化时 panic）
func GetAdminAuthService() *AdminAuthService {
	if adminService == nil {
		panic("admin auth module not initialized")
	}
	return adminService
}

// GetAdminAuthHandler 获取管理员认证处理器实例。
//
// 返回值:
//   - *AdminAuthHandler: 管理员认证处理器实例（未初始化时 panic）
func GetAdminAuthHandler() *AdminAuthHandler {
	if adminHandler == nil {
		panic("admin auth module not initialized")
	}
	return adminHandler
}
