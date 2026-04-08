package auth

import (
	"log"
	"time"
)

//go:generate mockgen -source=service.go -destination=mock_service.go -package=auth

// OAuthClient OAuth认证客户端接口
type OAuthClient interface {
	GetUserAccessToken(code string) (*OAuthAccessTokenResult, error)
	GetUserInfo(openid string) string
}

// OAuthAccessTokenResult OAuth访问令牌结果
type OAuthAccessTokenResult struct {
	OpenID string
}

// UserInfoProvider 用户信息提供者接口
type UserInfoProvider interface {
	GetUserInfo(openid string) string
}

// UserAuthService 提供纯粹的账号操作
type UserAuthService struct {
	repo             UserRepository
	oauth            OAuthClient
	userInfoProvider UserInfoProvider
}

// NewUserAuthService 创建用户认证服务
func NewUserAuthService(repo UserRepository, oauth OAuthClient) *UserAuthService {
	return &UserAuthService{
		repo:  repo,
		oauth: oauth,
	}
}

// NewUserAuthServiceWithUserInfo 创建带自定义用户信息提供者的服务（用于测试）
func NewUserAuthServiceWithUserInfo(repo UserRepository, oauth OAuthClient, provider UserInfoProvider) *UserAuthService {
	return &UserAuthService{
		repo:             repo,
		oauth:            oauth,
		userInfoProvider: provider,
	}
}

func (s *UserAuthService) FindOrCreate(openid string) (*User, error) {
	log.Printf("[debug][auth/service/FindOrCreate]: call wechat.GetUserInfo")

	var nickname string
	if s.userInfoProvider != nil {
		nickname = s.userInfoProvider.GetUserInfo(openid)
	} else {
		nickname = s.oauth.GetUserInfo(openid)
	}

	user := &User{
		OpenID:    openid,
		Nickname:  nickname,
		Status:    1,
		LastLogin: time.Now(),
	}

	return user, s.repo.Upsert(user)
}

func (s *UserAuthService) SetStatus(openid string, active bool) error {
	// UPDATE users SET status = ? WHERE openid = ?
	return nil
}

// LoginByCode 通过授权码登录
func (s *UserAuthService) LoginByCode(code string) (string, error) {
	result, err := s.oauth.GetUserAccessToken(code)
	if err != nil {
		return "", err
	}
	log.Printf("[info][auth/service/LoginByCode]: got openID %s", result.OpenID)

	return result.OpenID, nil
}
