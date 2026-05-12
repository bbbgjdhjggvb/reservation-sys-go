package auth

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	pb "reservation-sys/service/gateway/api/gen/account"
	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func setupAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestLoginHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := NewMockAccountServiceClient(ctrl)
	svc := NewAdminAuthService(mockClient)
	hdl := NewAdminAuthHandler(svc)
	r := setupAuthTestRouter()
	r.POST("/login", hdl.LoginHandler)

	tests := []struct {
		name      string
		body      string
		mockSetup func()
		wantCode  int
		wantMsg   string
	}{
		{
			name: "success",
			body: `{"username":"admin1","password":"123456"}`,
			mockSetup: func() {
				mockClient.EXPECT().VerifyAdmin(gomock.Any(), gomock.Any()).
					Return(&pb.VerifyAdminResp{
						Success: true, AdminId: 1, Username: "admin1",
						RealName: "管理员", Role: 1, Message: "success",
					}, nil)
			},
			wantCode: 200,
			wantMsg:  "登录成功",
		},
		{
			name:     "bad_request_missing_body",
			body:     `{`,
			mockSetup: func() {},
			wantCode: 400,
			wantMsg:  "参数错误",
		},
		{
			name: "unauthorized_bad_credentials",
			body: `{"username":"admin1","password":"wrong"}`,
			mockSetup: func() {
				mockClient.EXPECT().VerifyAdmin(gomock.Any(), gomock.Any()).
					Return(&pb.VerifyAdminResp{Success: false, Message: "用户名或密码错误"}, nil)
			},
			wantCode: 401,
			wantMsg:  "用户名或密码错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			req := httptest.NewRequest("POST", "/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
			var resp AdminResp
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Contains(t, resp.Msg, tt.wantMsg)
		})
	}
}

func TestGetAdminInfoHandler(t *testing.T) {
	hdl := NewAdminAuthHandler(nil)

	t.Run("success", func(t *testing.T) {
		r := setupAuthTestRouter()
		// 用中间件注入 admin claims 模拟已认证状态
		r.Use(func(c *gin.Context) {
			c.Set("admin", &jwt.AdminClaims{AdminID: 1, Username: "admin1", Role: 1})
			c.Next()
		})
		r.GET("/info", hdl.GetAdminInfoHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/info", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var resp AdminResp
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, "success", resp.Msg)
	})

	t.Run("unauthorized_no_admin_in_context", func(t *testing.T) {
		r := setupAuthTestRouter()
		r.GET("/info", hdl.GetAdminInfoHandler)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/info", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})
}
