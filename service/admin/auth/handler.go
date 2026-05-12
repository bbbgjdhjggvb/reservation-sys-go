package auth

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ==================== 管理员认证 Handler ====================

// AdminAuthHandler 管理员认证处理器
type AdminAuthHandler struct {
	svc *AdminAuthService
}

// NewAdminAuthHandler 创建管理员认证处理器实例
func NewAdminAuthHandler(svc *AdminAuthService) *AdminAuthHandler {
	return &AdminAuthHandler{svc: svc}
}

// --- 管理员接口响应辅助方法 ---

// adminOk 成功响应
func adminOk(c *gin.Context, data any) {
	c.JSON(http.StatusOK, AdminResp{Code: 200, Msg: "success", Data: data})
}

// adminOkWithMsg 成功响应，带消息
func adminOkWithMsg(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, AdminResp{Code: 200, Msg: msg, Data: data})
}

// adminBadRequest 错误响应
func adminBadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, AdminResp{Code: 400, Msg: msg})
}

// adminUnauthorized 未授权的错误响应
func adminUnauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, AdminResp{Code: 401, Msg: msg})
}

// adminForbidden 权限不匹配的错误响应
func adminForbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, AdminResp{Code: 403, Msg: msg})
}

// adminInternalError 服务器内部错误的响应
func adminInternalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, AdminResp{Code: 500, Msg: msg})
}

// LoginHandler 管理员登录接口。
//
//	@Summary		管理员登录
//	@Description	验证管理员用户名密码（通过 gRPC 调用 Gateway），返回 JWT Token 和管理员信息
//	@Tags			管理员-认证
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginReq	true	"管理员登录凭证"
//	@Success		200		{object}	AdminResp{data=LoginResp}	"登录成功"
//	@Failure		400		{object}	AdminResp					"参数错误"
//	@Failure		401		{object}	AdminResp					"凭证错误"
//	@Router			/api/v3/auth/login [post]
func (h *AdminAuthHandler) LoginHandler(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		adminBadRequest(c, "参数错误")
		return
	}

	admin, token, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		adminUnauthorized(c, err.Error())
		return
	}

	adminOkWithMsg(c, "登录成功", &LoginResp{
		Token:    token,
		Username: admin.Username,
		RealName: admin.RealName,
		Role:     admin.Role,
		RoleText: RoleText(admin.Role),
	})

	log.Printf("[info][auth/admin/login] admin=%s(%s) role=%d login success", admin.Username, admin.RealName, admin.Role)
}

// GetAdminInfoHandler 获取当前管理员信息接口。
//
//	@Summary		获取当前管理员信息
//	@Description	根据 JWT Token 返回当前登录管理员的 ID、用户名、角色等信息
//	@Tags			管理员-认证
//	@Produce		json
//	@Success		200	{object}	AdminResp{data=AdminInfoResp}	"管理员信息"
//	@Failure		401	{object}	AdminResp						"未登录"
//	@Security		BearerAuth
//	@Router			/api/v3/admin/info [get]
func (h *AdminAuthHandler) GetAdminInfoHandler(c *gin.Context) {
	claims, exists := GetAdminInfo(c)
	if !exists {
		adminUnauthorized(c, "未登录")
		return
	}

	adminOk(c, &AdminInfoResp{
		ID:       claims.AdminID,
		Username: claims.Username,
		Role:     claims.Role,
		RoleText: RoleText(claims.Role),
	})
}
