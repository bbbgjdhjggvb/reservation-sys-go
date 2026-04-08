package jwt

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWT 配置，支持模块化初始化
var (
	secret     []byte
	expireTime int = 24 // 默认过期时间（小时）
	secretOnce sync.Once
)

// Init 初始化 JWT 配置（各模块启动时调用）
func Init(jwtSecret string, expireHours int) {
	secretOnce.Do(func() {
		if jwtSecret != "" {
			secret = []byte(jwtSecret)
		} else {
			// 测试环境或配置未加载时的默认密钥
			secret = []byte("test-default-secret-do-not-use-in-production")
		}
		if expireHours > 0 {
			expireTime = expireHours
		}
	})
}

// 自定义荷载，里面存放敏感信息
type Claims struct {
	OpenID string `json:"openid"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT
func GenerateToken(openid string) (string, error) {
	nowTime := time.Now()
	expireDuration := time.Duration(expireTime) * time.Hour

	claims := Claims{
		OpenID: openid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(nowTime.Add(expireDuration)),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			Issuer:    "reservation-sys",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// ParseToken 解析并验证 JWT
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (any, error) {
			return secret, nil
		})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
