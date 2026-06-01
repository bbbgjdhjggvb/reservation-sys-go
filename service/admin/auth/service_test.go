package auth

import (
	"errors"
	"testing"

	pb "reservation-sys/service/gateway/api/gen/account"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// 测试 service.go 文件中 func (s *AdminAuthService) Login(username, password string) (*AdminInfo, string, error)
//
// 函数功能：调用 gRPC 验证管理员凭据，成功后签发 JWT
//
// 测试场景：
// 1. 登录成功 — 验证返回 AdminInfo 和 Token
// 2. gRPC 错误 — 验证返回"账号验证服务不可用"
// 3. 凭据无效 — 验证返回"用户名或密码错误"
func TestAdminAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockClient := NewMockAccountServiceClient(ctrl)
	svc := NewAdminAuthService(mockClient)

	tests := []struct {
		name      string
		username  string
		password  string
		mockSetup func()
		wantErr   bool
		errMsg    string
		checkResp func(*testing.T, *AdminInfo, string)
	}{
		{
			name:     "success",
			username: "admin1",
			password: "123456",
			mockSetup: func() {
				mockClient.EXPECT().VerifyAdmin(gomock.Any(), &pb.VerifyAdminReq{
					Username: "admin1",
					Password: "123456",
				}).Return(&pb.VerifyAdminResp{
					Success:  true,
					AdminId:  1,
					Username: "admin1",
					RealName: "管理员1",
					Role:     1,
					Message:  "success",
				}, nil)
			},
			wantErr: false,
			checkResp: func(t *testing.T, admin *AdminInfo, token string) {
				assert.NotEmpty(t, token)
				assert.Equal(t, uint(1), admin.ID)
				assert.Equal(t, "admin1", admin.Username)
				assert.Equal(t, "管理员1", admin.RealName)
				assert.Equal(t, 1, admin.Role)
			},
		},
		{
			name:     "grpc_error",
			username: "admin1",
			password: "wrong",
			mockSetup: func() {
				mockClient.EXPECT().VerifyAdmin(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("connection refused"))
			},
			wantErr: true,
			errMsg:  "账号验证服务不可用",
		},
		{
			name:     "invalid_credentials",
			username: "admin1",
			password: "wrong",
			mockSetup: func() {
				mockClient.EXPECT().VerifyAdmin(gomock.Any(), gomock.Any()).
					Return(&pb.VerifyAdminResp{
						Success: false,
						Message: "用户名或密码错误",
					}, nil)
			},
			wantErr: true,
			errMsg:  "用户名或密码错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			admin, token, err := svc.Login(tt.username, tt.password)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				tt.checkResp(t, admin, token)
			}
		})
	}
}
