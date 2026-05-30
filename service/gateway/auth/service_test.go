package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// ---------- UserAuthService.LoginByCode 测试 ----------
//
// 测试 service.go 文件中 func (s *UserAuthService) LoginByCode(code string) (string, error)
//
// 函数功能：通过微信 OAuth code 换取 openid

// TestUserAuthService_LoginByCode 测试通过 code 登录
//  1. 登录成功 — 验证返回 openid
//  2. OAuth失败 — 验证返回 error
func TestUserAuthService_LoginByCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOAuth := NewMockOAuthClient(ctrl)
	mockRepo := NewMockUserRepository(ctrl)

	svc := NewUserAuthService(mockRepo, mockOAuth)

	tests := []struct {
		name       string
		code       string
		mockSetup  func()
		wantOpenID string
		wantErr    bool
		errMsg     string
	}{
		{
			name: "登录成功",
			code: "valid_code",
			mockSetup: func() {
				mockOAuth.EXPECT().
					GetUserAccessToken("valid_code").
					Return(&OAuthAccessTokenResult{OpenID: "test_openid_123"}, nil)
			},
			wantOpenID: "test_openid_123",
			wantErr:    false,
		},
		{
			name: "OAuth失败",
			code: "invalid_code",
			mockSetup: func() {
				mockOAuth.EXPECT().
					GetUserAccessToken("invalid_code").
					Return(nil, errors.New("oauth error"))
			},
			wantOpenID: "",
			wantErr:    true,
			errMsg:     "oauth error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			openid, err := svc.LoginByCode(tt.code)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Empty(t, openid)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOpenID, openid)
			}
		})
	}
}

// ---------- UserAuthService.FindOrCreate 测试 ----------
//
// 测试 service.go 文件中 func (s *UserAuthService) FindOrCreate(openid string) (*User, error)
//
// 函数功能：根据 openid 查找用户，不存在则创建新用户

// TestUserAuthService_FindOrCreate 测试查找或创建用户
//  1. 创建用户成功 — 验证昵称、状态、登录时间正确
//  2. 获取用户信息失败(昵称为空) — 验证昵称为空但创建成功
//  3. 数据库Upsert失败 — 验证返回 error
func TestUserAuthService_FindOrCreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOAuth := NewMockOAuthClient(ctrl)
	mockRepo := NewMockUserRepository(ctrl)

	testOpenID := "test_openid_456"

	tests := []struct {
		name      string
		openid    string
		mockSetup func()
		wantErr   bool
		errMsg    string
		checkUser func(t *testing.T, user *User)
	}{
		{
			name:   "创建用户成功",
			openid: testOpenID,
			mockSetup: func() {
				mockOAuth.EXPECT().
					GetUserInfo(testOpenID).
					Return("TestUser")
				mockRepo.EXPECT().
					Upsert(gomock.Any()).
					Return(nil)
			},
			wantErr: false,
			checkUser: func(t *testing.T, user *User) {
				assert.Equal(t, testOpenID, user.OpenID)
				assert.Equal(t, "TestUser", user.Nickname)
				assert.Equal(t, 1, user.Status)
				assert.WithinDuration(t, time.Now(), user.LastLogin, time.Second)
			},
		},
		{
			name:   "获取用户信息失败",
			openid: testOpenID,
			mockSetup: func() {
				mockOAuth.EXPECT().
					GetUserInfo(testOpenID).
					Return("") // 微信API返回空昵称
				mockRepo.EXPECT().
					Upsert(gomock.Any()).
					Return(nil)
			},
			wantErr: false,
			checkUser: func(t *testing.T, user *User) {
				assert.Equal(t, testOpenID, user.OpenID)
				assert.Equal(t, "", user.Nickname) // 昵称为空
				assert.Equal(t, 1, user.Status)
			},
		},
		{
			name:   "数据库Upsert失败",
			openid: testOpenID,
			mockSetup: func() {
				mockOAuth.EXPECT().
					GetUserInfo(testOpenID).
					Return("TestUser")
				mockRepo.EXPECT().
					Upsert(gomock.Any()).
					Return(errors.New("db error"))
			},
			wantErr: true,
			errMsg:  "db error",
			checkUser: func(t *testing.T, user *User) {
				// 即使 Upsert 失败，service 也会返回构造的 user 对象
				assert.NotNil(t, user)
				assert.Equal(t, testOpenID, user.OpenID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			svc := NewUserAuthService(mockRepo, mockOAuth)
			user, err := svc.FindOrCreate(tt.openid)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				// 即使出错，user 也可能被返回（在 Upsert 之前构造的）
				if tt.checkUser != nil {
					tt.checkUser(t, user)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				if tt.checkUser != nil {
					tt.checkUser(t, user)
				}
			}
		})
	}
}

// 测试 service.go 文件中 func NewUserAuthServiceWithUserInfo 的自定义 UserInfoProvider 功能
//
// 函数功能：使用自定义 UserInfoProvider 替代微信 API 获取用户信息
//
// TestUserAuthService_FindOrCreate_WithCustomProvider 验证自定义提供者返回指定昵称
//  1. 验证自定义昵称被正确使用
func TestUserAuthService_FindOrCreate_WithCustomProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOAuth := NewMockOAuthClient(ctrl)
	mockRepo := NewMockUserRepository(ctrl)
	mockProvider := NewMockUserInfoProvider(ctrl)

	testOpenID := "test_openid_789"

	// Mock 自定义提供者返回特定昵称
	mockProvider.EXPECT().
		GetUserInfo(testOpenID).
		Return("CustomNickname")
	mockRepo.EXPECT().
		Upsert(gomock.Any()).
		Return(nil)

	svc := NewUserAuthServiceWithUserInfo(mockRepo, mockOAuth, mockProvider)
	user, err := svc.FindOrCreate(testOpenID)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "CustomNickname", user.Nickname)
	assert.Equal(t, testOpenID, user.OpenID)
}

// ---------- UserAuthService.SetStatus 测试 ----------
//
// 测试 service.go 文件中 func (s *UserAuthService) SetStatus(openid string, active bool) error
//
// 函数功能：设置用户激活状态（当前为占位符实现，始终返回 nil）

// TestUserAuthService_SetStatus 验证 SetStatus 不返回错误
//  1. 验证返回 nil
func TestUserAuthService_SetStatus(t *testing.T) {
	// 当前实现只是占位符，返回 nil
	svc := &UserAuthService{}
	err := svc.SetStatus("any_openid", true)
	assert.NoError(t, err)
}

// ---------- Repository 接口测试 ----------

func TestUserRepository_Interface(t *testing.T) {
	// 验证 userRepository 实现了 UserRepository 接口
	var _ UserRepository = (*userRepository)(nil)
}

func TestMockUserRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockUserRepository(ctrl)
	testUser := &User{OpenID: "test_123", Nickname: "Test"}

	// 测试 Upsert
	mockRepo.EXPECT().Upsert(testUser).Return(nil)
	err := mockRepo.Upsert(testUser)
	assert.NoError(t, err)

	// 测试 GetByOpenID
	mockRepo.EXPECT().GetByOpenID("test_123").Return(testUser, nil)
	user, err := mockRepo.GetByOpenID("test_123")
	assert.NoError(t, err)
	assert.Equal(t, testUser, user)

	// 测试 UpdateStatus
	mockRepo.EXPECT().UpdateStatus("test_123", 0).Return(nil)
	err = mockRepo.UpdateStatus("test_123", 0)
	assert.NoError(t, err)
}
