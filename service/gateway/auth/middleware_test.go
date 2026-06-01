package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ---------- 辅助函数 ----------

// setupTestRouter 创建测试用的 Gin 路由
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

// performRequest 发送测试请求并返回响应
func performRequest(r *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// generateTestToken 生成测试用的 JWT Token
func generateTestToken(t *testing.T, openid string) string {
	t.Helper()
	token, err := jwt.GenerateUserToken(openid)
	if err != nil {
		t.Fatalf("生成测试 token 失败: %v", err)
	}
	return token
}

// ---------- AuthMiddleware 测试 ----------
//
// 测试 middleware.go 文件中 func AuthMiddleware() gin.HandlerFunc
//
// 函数功能：验证请求中的用户 Bearer Token，解析 openid 并注入 Gin 上下文

// TestAuthMiddleware_NoAuthorizationHeader 无 Authorization 头时返回 401
//  1. 验证状态码为 401，code 为 401，msg 为"未授权，请从服务号进行订阅"
func TestAuthMiddleware_NoAuthorizationHeader(t *testing.T) {
	r := setupTestRouter()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "ok"})
	})

	w := performRequest(r, "GET", "/protected", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(401), resp["code"])
	assert.Equal(t, "未授权，请从服务号进行订阅", resp["msg"])
}

// TestAuthMiddleware_InvalidTokenFormat 验证各种非法 Token 格式均被拒绝
//  1. 无Bearer前缀 — 返回401 "Token格式错误"
//  2. 只写Bearer无token — 返回401 "Token格式错误"
//  3. Basic前缀 — 返回401 "Token格式错误"
//  4. Bearer多余空格 — 返回401 "Token格式错误"（按格式正确但token无效处理）
func TestAuthMiddleware_InvalidTokenFormat(t *testing.T) {
	r := setupTestRouter()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "ok"})
	})

	tests := []struct {
		name      string
		authVal   string
		expectMsg string
	}{
		{
			name:      "无Bearer前缀",
			authVal:   "some-random-token",
			expectMsg: "Token格式错误",
		},
		{
			name:      "只写Bearer无token",
			authVal:   "Bearer",
			expectMsg: "Token格式错误",
		},
		{
			name:      "Basic前缀",
			authVal:   "Basic dXNlcjpwYXNz",
			expectMsg: "Token格式错误",
		},
		{
			name:      "Bearer前缀多了空格",
			authVal:   "Bearer  extra spaces",
			expectMsg: "Token 无效或已过期", // 格式正确但token无效
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(r, "GET", "/protected", map[string]string{
				"Authorization": tt.authVal,
			})

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, float64(401), resp["code"])
			assert.Contains(t, resp["msg"], tt.expectMsg)
		})
	}
}

// TestAuthMiddleware_InvalidToken 验证各种无效 Token 被拒绝
//  1. 伪造token — 返回401 "无效或已过期"
//  2. 空token — 返回401 "无效或已过期"
//  3. 错误格式JWT — 返回401 "无效或已过期"
func TestAuthMiddleware_InvalidToken(t *testing.T) {
	r := setupTestRouter()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "ok"})
	})

	tests := []struct {
		name    string
		authVal string
	}{
		{
			name:    "伪造的token",
			authVal: "Bearer fake-invalid-token-string",
		},
		{
			name:    "空token",
			authVal: "Bearer ",
		},
		{
			name:    "错误格式的JWT",
			authVal: "Bearer not.a.jwt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := performRequest(r, "GET", "/protected", map[string]string{
				"Authorization": tt.authVal,
			})

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, float64(401), resp["code"])
			assert.Contains(t, resp["msg"], "无效或已过期")
		})
	}
}

// TestAuthMiddleware_ValidToken 验证有效 Token 正确放行并注入 openid
//  1. 验证返回 200
//  2. 验证 openid 正确存在于上下文中
//  3. 验证响应体中的 openid 与 Token 中的一致
func TestAuthMiddleware_ValidToken(t *testing.T) {
	testOpenID := "test_openid_12345"
	token := generateTestToken(t, testOpenID)

	r := setupTestRouter()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		// 验证 openid 被正确注入到上下文
		openid, exists := c.Get("openid")
		assert.True(t, exists, "openid 应存在于上下文中")
		assert.Equal(t, testOpenID, openid)
		c.JSON(http.StatusOK, gin.H{"openid": openid})
	})

	w := performRequest(r, "GET", "/protected", map[string]string{
		"Authorization": "Bearer " + token,
	})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, testOpenID, resp["openid"])
}

// TestAuthMiddleware_AbortPreventsNextHandler 验证中间件拦截后后续 handler 不被执行
//  1. 验证返回 401
//  2. 验证 nextCalled 为 false
func TestAuthMiddleware_AbortPreventsNextHandler(t *testing.T) {
	// 不设置 Authorization，期望中间件拦截请求
	nextCalled := false

	r := setupTestRouter()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		nextCalled = true
		c.JSON(http.StatusOK, gin.H{"msg": "should not reach"})
	})

	w := performRequest(r, "GET", "/protected", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, nextCalled, "中间件应拦截请求，后续handler不应被执行")
}

// TestAuthMiddleware_ResponseIsJSON 验证错误响应格式为 JSON
//  1. 验证 Content-Type 为 application/json; charset=utf-8
func TestAuthMiddleware_ResponseIsJSON(t *testing.T) {
	r := setupTestRouter()
	r.GET("/protected", AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "ok"})
	})

	w := performRequest(r, "GET", "/protected", nil)

	// 验证返回的是 JSON 格式
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}
