package auth

import (
	"fmt"
	"log"
	"time"

	"reservation-sys/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

//go:generate mockgen -source=service.go -destination=mock_service.go -package=auth

// OAuthClient OAuth认证客户端接口
// 通过接口限制可以使用的方法，同时方便进行mock测试
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

// ==================== 用户认证服务 ====================

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

// FindOrCreate 根据 openid 查找用户，不存在则创建（Upsert 原子操作）。
//
// 参数:
//   - openid: 微信用户唯一标识
//
// 返回值:
//   - *User: 用户实体
//   - error: 数据库操作失败时返回错误
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

// SetStatus 设置用户关注状态（关注/取关）。
//
// 参数:
//   - openid: 微信用户唯一标识
//   - active: true=正常，false=已取消关注
//
// 返回值:
//   - error: 更新失败时返回错误（当前为空实现）
func (s *UserAuthService) SetStatus(openid string, active bool) error {
	// UPDATE users SET status = ? WHERE openid = ?
	return nil
}

// LoginByCode 通过微信授权码换取用户 openid。
//
// 参数:
//   - code: 微信 OAuth 授权码（由前端/微信回调获取）
//
// 返回值:
//   - string: 用户 openid
//   - error: 授权码无效或微信接口调用失败时返回错误
func (s *UserAuthService) LoginByCode(code string) (string, error) {
	result, err := s.oauth.GetUserAccessToken(code)
	if err != nil {
		log.Printf("[error][auth/service/LoginByCode]: failed to get access token: %v", err)
		return "", err
	}
	log.Printf("[info][auth/service/LoginByCode]: got openID %s", result.OpenID)

	return result.OpenID, nil
}

// ==================== 管理员认证服务 ====================

// AdminAuthService 管理员认证服务
type AdminAuthService struct {
	repo AdminRepository
}

// NewAdminAuthService 创建管理员认证服务实例
func NewAdminAuthService(repo AdminRepository) *AdminAuthService {
	return &AdminAuthService{repo: repo}
}

// Login 管理员登录（查询数据库验证凭证，生成 JWT Token）。
//
// 流程:
//  1. 根据用户名查询 admins 表（仅查正常状态账号）
//  2. 使用 bcrypt 比对密码哈希
//  3. 更新最后登录时间
//  4. 生成 Admin JWT Token
//
// 参数:
//   - username: 管理员用户名
//   - password: 管理员密码（明文）
//
// 返回值:
//   - *Admin: 管理员实体
//   - string: JWT Token 字符串
//   - error: 用户名不存在、密码错误、Token 生成失败时返回错误
func (s *AdminAuthService) Login(username, password string) (*Admin, string, error) {
	admin, err := s.repo.FindAdminByUsername(username)
	if err != nil {
		return nil, "", fmt.Errorf("用户名或密码错误")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(password)); err != nil {
		return nil, "", fmt.Errorf("用户名或密码错误")
	}

	// 更新登录时间
	s.repo.UpdateAdminLoginTime(admin.ID)

	// 生成管理员 JWT Token
	token, err := jwt.GenerateAdminToken(admin.ID, admin.Username, admin.Role)
	if err != nil {
		return nil, "", fmt.Errorf("生成token失败")
	}

	return admin, token, nil
}

// HashPassword 对明文密码进行 bcrypt 哈希（用于初始化管理员密码）。
//
// 参数:
//   - password: 明文密码
//
// 返回值:
//   - string: bcrypt 哈希字符串
//   - error: 哈希生成失败时返回错误
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}
