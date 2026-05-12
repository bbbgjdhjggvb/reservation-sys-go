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
	"reserve":   "http://localhost:8081/reserve",
	"myorders":  "http://localhost:8081/myorders",
}

// ---------- WeChatCallBack Handler 测试 ----------

func TestWeChatCallBack_MissingCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	hdl := NewUserAuthHandler(nil, testDefaultRedirect, testRedirectURLs)
	r.GET("/api/v1/auth/callback", hdl.WeChatCallBack)

	// 不带 code 参数
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auth/callback", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "缺少 code 参数，从微信服务号进入预约界面", resp["msg"])
}

func TestWeChatCallBack_EmptyCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	hdl := NewUserAuthHandler(nil, testDefaultRedirect, testRedirectURLs)
	r.GET("/api/v1/auth/callback", hdl.WeChatCallBack)

	// code 参数为空字符串
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auth/callback?code=", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "缺少 code 参数，从微信服务号进入预约界面", resp["msg"])
}

func TestWeChatCallBack_LoginByCodeFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOAuth := NewMockOAuthClient(ctrl)
	mockRepo := NewMockUserRepository(ctrl)

	svc := NewUserAuthService(mockRepo, mockOAuth)
	hdl := NewUserAuthHandler(svc, testDefaultRedirect, testRedirectURLs)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/auth/callback", hdl.WeChatCallBack)

	// Mock LoginByCode 失败
	mockOAuth.EXPECT().
		GetUserAccessToken("invalid_code").
		Return(nil, errors.New("oauth error"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auth/callback?code=invalid_code", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "微信授权失效", resp["msg"])
}

func TestWeChatCallBack_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOAuth := NewMockOAuthClient(ctrl)
	mockRepo := NewMockUserRepository(ctrl)

	svc := NewUserAuthService(mockRepo, mockOAuth)
	hdl := NewUserAuthHandler(svc, testDefaultRedirect, testRedirectURLs)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/auth/callback", hdl.WeChatCallBack)

	testOpenID := "test_openid_12345"

	// Mock OAuth 成功
	mockOAuth.EXPECT().
		GetUserAccessToken("valid_code").
		Return(&OAuthAccessTokenResult{OpenID: testOpenID}, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auth/callback?code=valid_code", nil)
	r.ServeHTTP(w, req)

	// 应该重定向到前端页面
	assert.Equal(t, http.StatusFound, w.Code)
	location := w.Header().Get("Location")
	assert.Contains(t, location, "token=")
	assert.Contains(t, location, testDefaultRedirect)
}

// ---------- User 模型测试 ----------

func TestUser_TableName(t *testing.T) {
	user := User{}
	assert.Equal(t, "users", user.TableName())
}

// ---------- Handler 构造函数测试 ----------

func TestNewUserAuthHandler(t *testing.T) {
	hdl := NewUserAuthHandler(nil, testDefaultRedirect, testRedirectURLs)
	assert.NotNil(t, hdl)
	assert.Nil(t, hdl.svc)
	assert.Equal(t, testDefaultRedirect, hdl.defaultRedirect)
}

// ---------- JSON 响应格式测试 ----------

func TestWeChatCallBack_ResponseIsJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	hdl := NewUserAuthHandler(nil, testDefaultRedirect, testRedirectURLs)
	r.GET("/api/v1/auth/callback", hdl.WeChatCallBack)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auth/callback", nil)
	r.ServeHTTP(w, req)

	// 验证 Content-Type 为 JSON
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// 验证响应体可以解析为 JSON
	var body bytes.Buffer
	_, err := body.ReadFrom(w.Body)
	assert.NoError(t, err)
	assert.True(t, json.Valid(body.Bytes()), "响应体应为合法 JSON")
}
