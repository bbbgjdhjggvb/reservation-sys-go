package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const testDefaultRedirect = "http://localhost:8081/reserve"

var testRedirectURLs = map[string]string{
	"reserve":  "http://localhost:8081/reserve",
	"myorders": "http://localhost:8081/myorders",
}

// ---------- WeChatCallBack Handler 测试 ----------
//
// 测试 handler.go 文件中 func (h *UserAuthHandler) WeChatCallBack(c *gin.Context)
//
// 函数功能：处理微信 OAuth 回调，用 code 换取 openid 并签发用户 JWT

// TestWeChatCallBack_MissingCode 测试缺少 code 参数时返回 400
// 场景：无 code 参数，应返回400并提示"缺少 code 参数"
func TestWeChatCallBack_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	hdl := NewUserAuthHandler(nil, testDefaultRedirect, testRedirectURLs)
	r.GET("/api/gateway/auth/callback", hdl.WeChatCallBack)

	// 不带 code 参数
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/gateway/auth/callback", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "缺少 code 参数，从微信服务号进入预约界面", resp["msg"])
}

// TestWeChatCallBack_EmptyCode 测试 code 参数为空字符串时返回 400
//  1. 验证返回400，msg 为"缺少 code 参数"
func TestWeChatCallBack_EmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	hdl := NewUserAuthHandler(nil, testDefaultRedirect, testRedirectURLs)
	r.GET("/api/gateway/auth/callback", hdl.WeChatCallBack)

	// code 参数为空字符串
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/gateway/auth/callback?code=", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "缺少 code 参数，从微信服务号进入预约界面", resp["msg"])
}

// TestWeChatCallBack_LoginByCodeFail 测试 OAuth code 换取失败时返回 401
//  1. Mock GetUserAccessToken 返回 error，验证返回401"微信授权失效"
func TestWeChatCallBack_LoginByCodeFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOAuth := NewMockOAuthClient(ctrl)
	mockRepo := NewMockUserRepository(ctrl)

	svc := NewUserAuthService(mockRepo, mockOAuth)
	hdl := NewUserAuthHandler(svc, testDefaultRedirect, testRedirectURLs)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/gateway/auth/callback", hdl.WeChatCallBack)

	// Mock LoginByCode 失败
	mockOAuth.EXPECT().
		GetUserAccessToken("invalid_code").
		Return(nil, errors.New("oauth error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/gateway/auth/callback?code=invalid_code", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "微信授权失效", resp["msg"])
}

// TestWeChatCallBack_Success 测试微信 OAuth 回调成功，重定向到前端页面
//  1. 验证返回 302 重定向
//  2. 验证 Location 头包含 token= 参数
//  3. 验证 Location 头包含默认重定向 URL
func TestWeChatCallBack_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOAuth := NewMockOAuthClient(ctrl)
	mockRepo := NewMockUserRepository(ctrl)

	svc := NewUserAuthService(mockRepo, mockOAuth)
	hdl := NewUserAuthHandler(svc, testDefaultRedirect, testRedirectURLs)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/gateway/auth/callback", hdl.WeChatCallBack)

	testOpenID := "test_openid_12345"

	// Mock OAuth 成功
	mockOAuth.EXPECT().
		GetUserAccessToken("valid_code").
		Return(&OAuthAccessTokenResult{OpenID: testOpenID}, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/gateway/auth/callback?code=valid_code", nil)
	r.ServeHTTP(w, req)

	// 应该重定向到前端页面
	assert.Equal(t, http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.Contains(t, location, "token=")
	assert.Contains(t, location, testDefaultRedirect)
}

// ---------- User 模型测试 ----------
//
// 测试 model.go 文件中 func (User) TableName() string
//
// 函数功能：返回 User 对应的数据库表名

// TestUser_TableName 验证表名为 "users"
//  1. 验证返回值为 "users"
func TestUser_TableName(t *testing.T) {
	user := User{}
	assert.Equal(t, "users", user.TableName())
}

// ---------- Handler 构造函数测试 ----------
//
// 测试 handler.go 文件中 func NewUserAuthHandler(svc *UserAuthService, defaultRedirect string, redirectURLs map[string]string) *UserAuthHandler
//
// 函数功能：创建 UserAuthHandler 实例

// TestNewUserAuthHandler 验证构造函数创建非 nil 实例，且 defaultRedirect 正确设置
//  1. 验证 hdl 不为 nil
//  2. 验证默认重定向 URL 正确
func TestNewUserAuthHandler(t *testing.T) {
	hdl := NewUserAuthHandler(nil, testDefaultRedirect, testRedirectURLs)
	assert.NotNil(t, hdl)
	assert.Nil(t, hdl.svc)
	assert.Equal(t, testDefaultRedirect, hdl.defaultRedirect)
}

// ---------- JSON 响应格式测试 ----------

// TestWeChatCallBack_ResponseIsJSON 验证错误响应体为合法 JSON 格式
//  1. 验证 Content-Type 为 application/json
//  2. 验证响应体为合法 JSON
func TestWeChatCallBack_ResponseIsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	hdl := NewUserAuthHandler(nil, testDefaultRedirect, testRedirectURLs)
	r.GET("/api/gateway/auth/callback", hdl.WeChatCallBack)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/gateway/auth/callback", nil)
	r.ServeHTTP(w, req)

	// 验证 Content-Type 为 JSON
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// 验证响应体可以解析为 JSON
	var body bytes.Buffer
	_, err := body.ReadFrom(w.Body)
	assert.NoError(t, err)
	assert.True(t, json.Valid(body.Bytes()), "响应体应为合法 JSON")
}
