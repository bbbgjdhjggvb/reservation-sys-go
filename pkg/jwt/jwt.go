package jwt

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ============================================================
// 用户 JWT（微信用户认证）
// ============================================================

var (
	userSecret     []byte
	userExpireTime int = 24 // 默认过期时间（小时）
	userSecretOnce sync.Once
)

// InitUserJWT 初始化用户 JWT 配置（各模块启动时调用）
func InitUserJWT(jwtSecret string, expireHours int) {
	userSecretOnce.Do(func() {
		if jwtSecret != "" {
			userSecret = []byte(jwtSecret)
		} else {
			// 测试环境或配置未加载时的默认密钥
			userSecret = []byte("test-default-secret-do-not-use-in-production")
		}
		if expireHours > 0 {
			userExpireTime = expireHours
		}
	})
}

// UserClaims 用户 JWT 荷载，存放微信用户的 OpenID
type UserClaims struct {
	OpenID string `json:"openid"`
	jwt.RegisteredClaims
}

// GenerateUserToken 生成用户 JWT Token
func GenerateUserToken(openid string) (string, error) {
	nowTime := time.Now()
	expireDuration := time.Duration(userExpireTime) * time.Hour

	claims := UserClaims{
		OpenID: openid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(expireDuration)),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			Issuer:    "reservation-sys-user",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(userSecret)
}

// ParseUserToken 解析并验证用户 JWT Token
func ParseUserToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&UserClaims{},
		func(token *jwt.Token) (any, error) {
			return userSecret, nil
		})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid user token")
}

// ============================================================
// 管理员 JWT（管理员后台认证）
// ============================================================

var (
	adminSecret     []byte
	adminExpireTime int = 24 // 默认过期时间（小时）
	adminSecretOnce sync.Once
)

// InitAdminJWT 初始化管理员 JWT 配置（各模块启动时调用）
func InitAdminJWT(jwtSecret string, expireHours int) {
	adminSecretOnce.Do(func() {
		if jwtSecret != "" {
			adminSecret = []byte(jwtSecret)
		} else {
			adminSecret = []byte("test-default-admin-secret-do-not-use-in-production")
		}
		if expireHours > 0 {
			adminExpireTime = expireHours
		}
	})
}

// AdminClaims 管理员 JWT 荷载，存放管理员身份信息
type AdminClaims struct {
	AdminID  uint   `json:"admin_id"`
	Username string `json:"username"`
	Role     int    `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAdminToken 生成管理员 JWT Token
func GenerateAdminToken(adminID uint, username string, role int) (string, error) {
	now := time.Now()
	claims := AdminClaims{
		AdminID:  adminID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(adminExpireTime) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "reservation-sys-admin",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(adminSecret)
}

// ParseAdminToken 解析并验证管理员 JWT Token
func ParseAdminToken(tokenString string) (*AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		return adminSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*AdminClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid admin token")
}
