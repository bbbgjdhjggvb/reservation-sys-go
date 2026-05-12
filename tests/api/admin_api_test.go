package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"reservation-sys/pkg/constants"
	reservationdb "reservation-sys/pkg/reservationdb"
	"reservation-sys/pkg/jwt"
	adminauth "reservation-sys/service/admin/auth"
	adminreview "reservation-sys/service/admin/review"
	pb "reservation-sys/service/gateway/api/gen/account"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Admin JWT 在 reservation_api_test.go 的 TestMain 中初始化（需在此包中，
// 所以将 admin JWT init 也放到同一 TestMain）

// setupAdminAPI 创建完整的 admin API 测试环境。
func setupAdminAPI(t *testing.T) (*gomock.Controller, *adminauth.MockAccountServiceClient, *adminreview.MockRepository, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	mockAccount := adminauth.NewMockAccountServiceClient(ctrl)
	mockRepo := adminreview.NewMockRepository(ctrl)
	mockNotify := adminreview.NewMockNotificationServiceClient(ctrl)

	authSvc := adminauth.NewAdminAuthService(mockAccount)
	authHdl := adminauth.NewAdminAuthHandler(authSvc)

	reviewSvc := adminreview.NewReviewService(mockRepo)
	notifyHdl := adminreview.NewNotifyHandler(mockNotify, mockRepo)
	reviewHdl := adminreview.NewReviewHandler(reviewSvc, notifyHdl)

	r := gin.New()

	// 认证路由（无中间件）
	authGroup := r.Group("/api/admin/auth")
	{
		authGroup.POST("/login", authHdl.LoginHandler)
	}

	// 需认证的路由
	api := r.Group("/api/admin")
	api.Use(adminauth.AdminAuthMiddleware())
	{
		api.GET("/admin/info", authHdl.GetAdminInfoHandler)
		api.GET("/orders", reviewHdl.GetOrderListHandler)
		api.GET("/orders/:id", reviewHdl.GetOrderDetailHandler)

		// 一级管理员路由
		level1 := api.Group("/review")
		level1.Use(adminauth.RoleMiddleware(constants.RoleLevel1))
		{
			level1.POST("/level1/:id", reviewHdl.Level1ReviewHandler)
			level1.PUT("/level1/:id/slots/:slotID/password", reviewHdl.SetPasswordHandler)
			level1.POST("/level1/:id/notify", reviewHdl.NotifyHandler)
			level1.POST("/level1/:id/reject-notify", reviewHdl.RejectionNotifyHandler)
		}

		// 二级管理员路由
		level2 := api.Group("/review")
		level2.Use(adminauth.RoleMiddleware(constants.RoleLevel2))
		{
			level2.POST("/level2/:id", reviewHdl.Level2ReviewHandler)
		}
	}

	return ctrl, mockAccount, mockRepo, r
}

// generateAdminToken 生成管理员 JWT token。
func generateAdminToken(t *testing.T, adminID uint, username string, role int) string {
	t.Helper()
	token, err := jwt.GenerateAdminToken(adminID, username, role)
	require.NoError(t, err)
	return token
}

// ========== 管理员登录 API ==========

func TestAdminAPI_Login(t *testing.T) {
	ctrl, mockAccount, _, r := setupAdminAPI(t)
	defer ctrl.Finish()

	t.Run("登录成功", func(t *testing.T) {
		mockAccount.EXPECT().VerifyAdmin(gomock.Any(), gomock.Any()).
			Return(&pb.VerifyAdminResp{
				Success:  true,
				AdminId:  1,
				Username: "admin1",
				RealName: "一级管理员",
				Role:     1,
				Message:  "success",
			}, nil)

		req := httptest.NewRequest("POST", "/api/admin/auth/login",
			strings.NewReader(`{"username":"admin1","password":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		var resp adminauth.AdminResp
		json.Unmarshal(w.Body.Bytes(), &resp)
		assert.Equal(t, 200, resp.Code)
		assert.Contains(t, resp.Msg, "登录成功")
	})

	t.Run("凭证错误", func(t *testing.T) {
		mockAccount.EXPECT().VerifyAdmin(gomock.Any(), gomock.Any()).
			Return(&pb.VerifyAdminResp{Success: false, Message: "用户名或密码错误"}, nil)

		req := httptest.NewRequest("POST", "/api/admin/auth/login",
			strings.NewReader(`{"username":"admin1","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})
}

// ========== 获取管理员信息 API ==========

func TestAdminAPI_GetAdminInfo(t *testing.T) {
	_, _, _, r := setupAdminAPI(t)

	token := generateAdminToken(t, 1, "admin1", 1)

	t.Run("成功获取信息", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/admin/info", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("无Token返回401", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/admin/info", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})
}

// ========== 订单列表 API ==========

func TestAdminAPI_GetOrderList(t *testing.T) {
	_, _, mockRepo, r := setupAdminAPI(t)

	token := generateAdminToken(t, 1, "admin1", 1)

	t.Run("获取全部订单", func(t *testing.T) {
		mockRepo.EXPECT().ListOrders([]int(nil), 1, 20).Return(
			[]*reservationdb.ReservationOrder{
				{
					ID: 1, OrderNo: "R001", OpenID: "u1",
					ApplicantName: "张三", Status: 0, TotalSlots: 1,
					Slots: []reservationdb.ReservationSlot{{ID: 1}},
				},
			}, int64(1), nil)

		req := httptest.NewRequest("GET", "/api/admin/orders?page=1&page_size=20", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

// ========== 审核流程 API ==========

func TestAdminAPI_Level1Review(t *testing.T) {
	_, _, mockRepo, r := setupAdminAPI(t)

	token := generateAdminToken(t, 1, "admin1", constants.RoleLevel1)

	t.Run("一级审核通过", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(&reservationdb.ReservationOrder{
			ID: 1, Status: reservationdb.StatusPendingLevel1,
		}, nil)
		mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel1, reservationdb.StatusPendingLevel2).Return(nil)
		mockRepo.EXPECT().CreateReviewRecord(gomock.Any()).Return(nil)

		req := httptest.NewRequest("POST", "/api/admin/review/level1/1",
			strings.NewReader(`{"action":1,"comment":"通过"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("非一级管理员访问被拒绝", func(t *testing.T) {
		token2 := generateAdminToken(t, 2, "admin2", constants.RoleLevel2)

		req := httptest.NewRequest("POST", "/api/admin/review/level1/1",
			strings.NewReader(`{"action":1,"comment":"通过"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token2)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
	})
}

func TestAdminAPI_Level2Review(t *testing.T) {
	_, _, mockRepo, r := setupAdminAPI(t)

	token := generateAdminToken(t, 2, "admin2", constants.RoleLevel2)

	t.Run("二级审核通过", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(&reservationdb.ReservationOrder{
			ID: 1, Status: reservationdb.StatusPendingLevel2,
		}, nil)
		mockRepo.EXPECT().UpdateOrderStatus(uint(1), reservationdb.StatusPendingLevel2, reservationdb.StatusApprovedFinal).Return(nil)
		mockRepo.EXPECT().CreateReviewRecord(gomock.Any()).Return(nil)

		req := httptest.NewRequest("POST", "/api/admin/review/level2/1",
			strings.NewReader(`{"action":1,"comment":"终审通过"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("非二级管理员访问被拒绝", func(t *testing.T) {
		token1 := generateAdminToken(t, 1, "admin1", constants.RoleLevel1)

		req := httptest.NewRequest("POST", "/api/admin/review/level2/1",
			strings.NewReader(`{"action":1}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token1)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
	})
}

// ========== 设置密码 API ==========

func TestAdminAPI_SetPassword(t *testing.T) {
	_, _, mockRepo, r := setupAdminAPI(t)

	token := generateAdminToken(t, 1, "admin1", constants.RoleLevel1)

	t.Run("设置密码成功", func(t *testing.T) {
		mockRepo.EXPECT().FindOrderByID(uint(1)).Return(&reservationdb.ReservationOrder{
			ID: 1, Status: reservationdb.StatusApprovedFinal,
		}, nil)
		mockRepo.EXPECT().SetSlotPassword(uint(10), "123456").Return(nil)

		req := httptest.NewRequest("PUT", "/api/admin/review/level1/1/slots/10/password",
			strings.NewReader(`{"password":"123456"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}

// 确保类型被使用
var _ = adminreview.Response{}
var _ = adminauth.AdminResp{}
