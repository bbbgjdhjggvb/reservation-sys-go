package auth

import (
	"github.com/silenceper/wechat/v2/officialaccount"
	"gorm.io/gorm"
)

var (
	instance    *UserAuthService
	handler     *UserAuthHandler
)

func InitModule(db *gorm.DB, oa *officialaccount.OfficialAccount, frontendURL string) {
	repo := NewUserRepository(db)
	oauth := NewWechatOAuthClient(oa)
	provider := NewWechatUserInfoProvider(oa)
	instance = NewUserAuthServiceWithUserInfo(repo, oauth, provider)
	handler = NewUserAuthHandler(instance, frontendURL)
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
