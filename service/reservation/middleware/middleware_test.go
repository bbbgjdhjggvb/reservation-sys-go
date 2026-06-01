package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	jwt.InitUserJWT("test-user-secret-for-unit-test", 24)
	os.Exit(m.Run())
}

func generateValidUserToken(t *testing.T) string {
	t.Helper()
	token, err := jwt.GenerateUserToken("test_openid_001")
	assert.NoError(t, err)
	return token
}

// 测试 middleware.go 文件中的 func AuthMiddleware() gin.HandlerFunc
//
// 函数功能：验证请求中的 Bearer Token，解析 openid 并注入 Gin 上下文
//
// 测试场景：
// 1. 无 Authorization 头 — 返回 401 "未授权"
// 2. 格式错误（无 Bearer 前缀） — 返回 401 "Token格式错误"
// 3. 格式错误（错误前缀） — 返回 401 "Token格式错误"
// 4. Token 无效 — 返回 401 "Token 无效或已过期"
// 5. 有效 Token — 返回 200，openid 正确注入上下文
func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		openid, _ := c.Get("openid")
		c.JSON(200, gin.H{"openid": openid})
	})

	tests := []struct {
		name       string
		authHeader string
		wantCode   int
		wantMsg    string
	}{
		{
			name:     "no_header",
			wantCode: 401,
			wantMsg:  "未授权",
		},
		{
			name:       "bad_format_no_bearer",
			authHeader: "token123",
			wantCode:   401,
			wantMsg:    "Token格式错误",
		},
		{
			name:       "bad_format_wrong_prefix",
			authHeader: "Basic token123",
			wantCode:   401,
			wantMsg:    "Token格式错误",
		},
		{
			name:       "invalid_token",
			authHeader: "Bearer invalid.token.here",
			wantCode:   401,
			wantMsg:    "Token 无效或已过期",
		},
		{
			name:       "valid_token",
			authHeader: "Bearer " + generateValidUserToken(t),
			wantCode:   200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantMsg != "" {
				var resp map[string]any
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Contains(t, resp["msg"], tt.wantMsg)
			}
		})
	}
}
