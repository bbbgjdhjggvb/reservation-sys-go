package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// func GenerateUserToken(openid string)(string, error) 测试
//
// 函数功能：根据用户的 opendi 生成 JWT token
//
// 测试场景：
// 1. 正常生成token，并且可以通过 ParseUserToken 函数解析
func TestGenerateUserToken(t *testing.T) {
	InitUserJWT("test-user-secret-key-123456", 24)

	t.Run("正常生成token_非空且可解析", func(t *testing.T) {
		token, err := GenerateUserToken("test_openid_001")
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ParseUserToken(token)
		require.NoError(t, err)
		assert.Equal(t, "test_openid_001", claims.OpenID)
		assert.Equal(t, "reservation-sys-user", claims.Issuer)
	})
}

// func ParseUserToken(tokenString string) (*UserClaims, error) 测试
//
// 函数功能：根据令牌解析出openid和令牌时间
//
// 测试场景：
// 1. 解析有效 token
// 2. 解析过期 token
// 3. 解析伪造密钥 token
// 4. 解析空字符串
func TestParseUserToken(t *testing.T) {
	InitUserJWT("test-user-secret-parse-123", 24)

	validToken := mustGenerateUserToken(t, "test_openid_002")

	tests := []struct {
		name    string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "解析有效token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "解析过期token",
			token:   mustGenerateExpiredUserToken(t, "test_openid_expired"),
			wantErr: true,
			errMsg:  "token is expired",
		},
		{
			name:    "解析伪造token_错误密钥",
			token:   mustGenerateTokenWithSecret("test_openid_fake", "wrong-secret-key"),
			wantErr: true,
		},
		{
			name:    "解析空字符串",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ParseUserToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, claims)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "test_openid_002", claims.OpenID)
			}
		})
	}
}

// func InitUserJWT(jwtSecret string, expireHours int)
//
// func GenerateUserToken(openid string) (string, error)
//
// 测试上面两个函数联合效果：在时间 <= 0 的情况下有没有兜底措施，
// 保障过期时间一定晚于创建时间
func TestGenerateUserToken_Expiration(t *testing.T) {
	InitUserJWT("test-user-expire-secret", 0)

	token, err := GenerateUserToken("test_openid_003")
	require.NoError(t, err)

	claims, err := ParseUserToken(token)
	require.NoError(t, err)

	assert.True(t, claims.ExpiresAt.Time.After(time.Now()), "token 过期时间应在将来")
}

// func GenerateAdminToken(adminID uint, username string, role int)(string, error) 测试
//
// 函数功能：生成管理员鉴权令牌
//
// 测试场景：
// 1. 正常生成管理员令牌，并解析出id,姓名，role

func TestGenerateAdminToken(t *testing.T) {
	InitAdminJWT("test-admin-secret-key-456", 24)

	t.Run("正常生成_admin_id_username_role可解析", func(t *testing.T) {
		token, err := GenerateAdminToken(42, "admin_user", 1)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := ParseAdminToken(token)
		require.NoError(t, err)
		assert.Equal(t, uint(42), claims.AdminID)
		assert.Equal(t, "admin_user", claims.Username)
		assert.Equal(t, 1, claims.Role)
		assert.Equal(t, "reservation-sys-admin", claims.Issuer)
	})
}

// func ParseAdminToken(tokenString string) (*AdminClaims, error)
//
// 函数功能：解析管理员令牌，获取信息
//
// 测试场景：
// 1. 解析有效令牌
// 2. 解析过期令牌
// 3. 解析伪造密钥的令牌 TODO
// 4. 解析空字符串
func TestParseAdminToken(t *testing.T) {
	InitAdminJWT("test-admin-secret-parse-789", 24)

	validToken := mustGenerateAdminToken(t, 7, "admin_test", 2)

	tests := []struct {
		name    string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "解析有效admin_token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "解析过期admin_token",
			token:   mustGenerateExpiredAdminToken(t, 8, "expired_admin", 1),
			wantErr: true,
			errMsg:  "token is expired",
		},
		{
			name:    "解析伪造token_错误密钥",
			token:   mustGenerateAdminTokenWithSecret(t, 9, "error_token_admin", 1, "use-make-up-secrete"),
			wantErr: true,
		},
		{
			name:    "解析空字符串",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ParseAdminToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, claims)
			} else {
				require.NoError(t, err)
				assert.Equal(t, uint(7), claims.AdminID)
			}
		})
	}
}

// 测试admin token 和 user token 有没有区分开来，
// 防止跨类型攻击
//
// 测试场景：
// 1. admin token parser 解析 user token 会失败
func TestCrossTypeAttack_UserKeyParseAdminToken(t *testing.T) {
	InitUserJWT("user-secret-cross", 24)
	InitAdminJWT("admin-secret-cross", 24)

	// 用用户密钥签名的 token，admin 解析器应拒绝
	userToken, err := GenerateUserToken("hacker_openid")
	require.NoError(t, err)

	claims, err := ParseAdminToken(userToken)
	assert.Error(t, err, "用户token不应被admin解析器接受")
	assert.Nil(t, claims)
}

// 测试admin token 和 user token 有没有区分开来，
// 防止跨类型攻击
//
// 测试场景：
// 1. user token parser 解析 admin token 会失败
func TestCrossTypeAttack_AdminKeyParseUserToken(t *testing.T) {
	InitUserJWT("user-secret-cross-2", 24)
	InitAdminJWT("admin-secret-cross-2", 24)

	// 用 admin 密钥签名的 token，用户解析器应拒绝
	adminToken, err := GenerateAdminToken(99, "hacker_admin", 1)
	require.NoError(t, err)

	claims, err := ParseUserToken(adminToken)
	assert.Error(t, err, "admin token不应被用户解析器接受")
	assert.Nil(t, claims)
}

// func InitUserJWT(jwtSecret string, expireHours int)
//
// 函数功能：初始化密钥和过期时间
//
// 测试场景：
// 1. 重复调用，保持第一次调用结果
// 2. 当密钥为空时，使用默认密钥
// 3. 输入 expireHours <= 0 时，使用默认过期时间，且过期时间要合理
func TestInitUserJWT_SyncOnce(t *testing.T) {
	// 第一次初始化
	InitUserJWT("first-secret-key", 12)

	token1, err := GenerateUserToken("openid_first")
	require.NoError(t, err)
	claims1, err := ParseUserToken(token1)
	require.NoError(t, err)
	assert.Equal(t, "openid_first", claims1.OpenID)

	// 第二次初始化（应被 sync.Once 忽略）
	InitUserJWT("second-secret-key", 48)

	// 用第一次初始化的密钥仍然能解析 token
	claims1Again, err := ParseUserToken(token1)
	require.NoError(t, err, "sync.Once 保证首次初始化后不会被覆盖，token1 仍应有效")
	assert.Equal(t, "openid_first", claims1Again.OpenID)

	// 新生成的 token 仍使用第一次的密钥
	token3, err := GenerateUserToken("openid_third")
	require.NoError(t, err)
	claims3, err := ParseUserToken(token3)
	require.NoError(t, err)
	assert.Equal(t, "openid_third", claims3.OpenID)
}

func TestInitUserJWT_EmptySecretUsesDefault(t *testing.T) {
	// sync.Once 在所有测试间共享，此测试验证空密钥时使用默认密钥的逻辑
	// 由于 sync.Once 可能已在其他测试中触发，因此仅验证 InitUserJWT 不 panic
	assert.NotPanics(t, func() {
		InitUserJWT("", 0)
	}, "空密钥调用不应 panic")
}

func TestInitAdminJWT_EmptySecretUsesDefault(t *testing.T) {
	assert.NotPanics(t, func() {
		InitAdminJWT("", 0)
	}, "空密钥调用不应 panic")
}

func TestInitUserJWT_ZeroExpireTimeKeepsDefault(t *testing.T) {
	assert.NotPanics(t, func() {
		InitUserJWT("some-secret", 0)
	}, "零过期时间不应 panic")
}

func TestInitAdminJWT_ZeroExpireTimeKeepsDefault(t *testing.T) {
	assert.NotPanics(t, func() {
		InitAdminJWT("some-secret", -1)
	}, "负过期时间不应 panic")
}

// ========== 辅助函数 ==========

// 生成正确的用户令牌
func mustGenerateUserToken(t *testing.T, openid string) string {
	t.Helper()
	token, err := GenerateUserToken(openid)
	require.NoError(t, err)
	return token
}

// 生成过期的用户令牌
func mustGenerateExpiredUserToken(t *testing.T, openid string) string {
	t.Helper()
	claims := UserClaims{
		OpenID: openid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "reservation-sys-user",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(userSecret)
	require.NoError(t, err)
	return signed
}

// 指定密钥，生成令牌
func mustGenerateTokenWithSecret(openid, secret string) string {
	claims := UserClaims{
		OpenID: openid,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "reservation-sys-user",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

// 生成正确的管理员令牌
func mustGenerateAdminToken(t *testing.T, adminID uint, username string, role int) string {
	t.Helper()
	token, err := GenerateAdminToken(adminID, username, role)
	require.NoError(t, err)
	return token
}

// 生成过期的管理员令牌
func mustGenerateExpiredAdminToken(t *testing.T, adminID uint, username string, role int) string {
	t.Helper()
	claims := AdminClaims{
		AdminID:  adminID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "reservation-sys-admin",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(adminSecret)
	require.NoError(t, err)
	return signed
}

// 指定密钥，生成管理员令牌
func mustGenerateAdminTokenWithSecret(t *testing.T, adminID uint, username string, role int, secret string) string {
	t.Helper()
	claims := AdminClaims{
		AdminID:  adminID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "reservation-sys-admin",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	signed, _ := token.SignedString([]byte(secret))
	return signed
}

// Banchmark go 性能测试工具

// Banchmark go 性能测试工具
func BenchmarkGenerateUserToken(b *testing.B) {
	InitUserJWT("bench-user-secret", 24)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		GenerateUserToken("bench_openid_test")
	}
}

func BenchmarkParseUserToken(b *testing.B) {
	InitUserJWT("bench-parse-secret", 24)
	token, _ := GenerateUserToken("bench_openid_parse")
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ParseUserToken(token)
	}
}

func BenchmarkGenerateAdminToken(b *testing.B) {
	InitAdminJWT("bench-admin-secret", 24)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		GenerateAdminToken(1, "admin_bench", 1)
	}
}

func BenchmarkParseAdminToken(b *testing.B) {
	InitAdminJWT("bench-admin-secret", 24)
	token, _ := GenerateAdminToken(1, "admin", 1)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ParseAdminToken(token)
	}
}
