package auth

import (
	"fmt"
	"log"
	"time"

	"reservation-sys/internal/auth/store"
	"reservation-sys/internal/model"
	"reservation-sys/pkg/config"
	"reservation-sys/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

// ========== OAuth 接口 ==========

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

// ========== 认证服务 ==========

// Service 认证服务，聚合用户认证和管理员认证。
// 同时提供 Notifier 方法，供 review 模块通过 Notifier 接口调用。
type Service struct {
	users  store.UserRepository
	admins store.AdminRepository
	oauth  OAuthClient
	wechat *config.WechatConfig
}

// NewService 创建认证服务实例
func NewService(
	users store.UserRepository,
	admins store.AdminRepository,
	oauth OAuthClient,
	wechat *config.WechatConfig,
) *Service {
	return &Service{
		users:  users,
		admins: admins,
		oauth:  oauth,
		wechat: wechat,
	}
}

// ========== 用户认证 ==========

// FindOrCreate 根据 openid 查找用户，不存在则创建。
func (s *Service) FindOrCreate(openid string) (*model.User, error) {
	log.Printf("[debug][auth/service] FindOrCreate: openid=%s", openid)

	nickname := s.oauth.GetUserInfo(openid)

	user := &model.User{
		OpenID:    openid,
		Nickname:  nickname,
		Status:    1,
		LastLogin: time.Now(),
	}

	return user, s.users.Upsert(user)
}

// SetStatus 设置用户关注状态。
func (s *Service) SetStatus(openid string, active bool) error {
	status := 0
	if active {
		status = 1
	}
	return s.users.UpdateStatus(openid, status)
}

// LoginByCode 通过微信授权码换取用户 openid。
func (s *Service) LoginByCode(code string) (string, error) {
	result, err := s.oauth.GetUserAccessToken(code)
	if err != nil {
		log.Printf("[error][auth/service] 获取 access token 失败: %v", err)
		return "", fmt.Errorf("%w: %v", ErrInvalidCode, err)
	}
	log.Printf("[info][auth/service] 获取 openid 成功: %s", result.OpenID)
	return result.OpenID, nil
}

// ========== 管理员认证 ==========

// AdminLogin 管理员登录（验证凭证，签发 JWT）。
func (s *Service) AdminLogin(username, password string) (*model.Admin, string, error) {
	admin, err := s.admins.FindAdminByUsername(username)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	// 更新登录时间
	s.admins.UpdateAdminLoginTime(admin.ID)

	// 生成管理员 JWT Token
	token, err := jwt.GenerateAdminToken(admin.ID, admin.Username, admin.Role)
	if err != nil {
		return nil, "", fmt.Errorf("生成token失败")
	}

	return admin, token, nil
}

// HashPassword 对明文密码进行 bcrypt 哈希。
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
