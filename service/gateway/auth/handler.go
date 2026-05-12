/* # 为什么要进行验证
 * 1. 防止有心人通过网页链接自己随意编造openid然后进行高频访问，搞垮服务器
 * 2. 限制用户必须通过微信服务号访问预约界面
 *
 * # 验证运行逻辑
 * 1. 用户在微信服务号点击预约按钮，微信服务器发给后台服务器一个 code
 * 2. 后端的authHdl.WeChatCallBack 用这个 code 换取用户的 openid
 * 3. 然后将利用 openid 生成一个 Token,然后重定位到预约的前端个网页
 * 4. 这个网页的url后面将会有这个 token,在用户填写完成信息点击提交后，会将这个填写的信息和这个token一起发送到后台
 * 5. 后台的预约处理模块就可以根据这个 token 来对用户的身份进行校验
 */

package auth

import (
	"log"
	"net/http"
	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
)

// ==================== 用户认证 Handler ====================

type UserAuthHandler struct {
	svc             *UserAuthService
	defaultRedirect string
	redirectURLs    map[string]string
}

func NewUserAuthHandler(svc *UserAuthService, defaultRedirect string, redirectURLs map[string]string) *UserAuthHandler {
	return &UserAuthHandler{
		svc:             svc,
		defaultRedirect: defaultRedirect,
		redirectURLs:    redirectURLs,
	}
}

// buildRedirectURL 根据 state 参数构建重定向地址，未匹配时使用默认地址
func (h *UserAuthHandler) buildRedirectURL(token, state string) string {
	if url, ok := h.redirectURLs[state]; ok && url != "" {
		return url + "?token=" + token
	}
	return h.defaultRedirect + "?token=" + token
}

// WeChatCallback 处理微信 OAuth 回调，用 code 换取 openid 并签发 JWT，重定向到前端页面。
//
//	@Summary		微信 OAuth 回调
//	@Description	微信服务号菜单点击后，微信服务器回调此接口，用 code 换取用户 openid，签发 JWT 后重定向到前端页面
//	@Tags			网关-认证
//	@Produce		json
//	@Param			code	query		string	true	"微信授权临时票据 code"
//	@Param			state	query		string	false	"重定向目标标识（映射到预设的页面URL）"
//	@Success		302		{string}	string	"重定向到前端页面（URL 带 token 参数）"
//	@Failure		400		{object}	object	"缺少 code 参数"
//	@Failure		401		{object}	object	"微信授权失效"
//	@Failure		500		{object}	object	"Token 生成失败"
//	@Router			/api/v1/auth/callback [get]
func (h *UserAuthHandler) WeChatCallBack(c *gin.Context) {
	// 获取微信重定向过来的 code 参数
	code := c.Query("code")
	if code == "" {
		errcode := c.Query("errcode")
		errmsg := c.Query("errmsg")
		if errcode != "" {
			log.Printf("[info][auth/handler/WeChatCallBack]: 微信授权失败: errocde=%s, errmsg=%s", errcode, errmsg)
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "缺少 code 参数，从微信服务号进入预约界面",
		})
		return
	}
	log.Printf("[info][auth/handler/WeChatCallBack]: 收到微信回调请求，code: %s", code)

	openid, err := h.svc.LoginByCode(code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"msg": "微信授权失效",
		})
		return
	}
	log.Printf("[info][auth/handler/WeChatCallBack]: 微信授权成功，获取用户openid: %s", openid)

	// 签发 JWT
	token, err := jwt.GenerateUserToken(openid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "服务器内部错误，Token 生成失效",
		})
		return
	}
	log.Printf("[info][auth/handler/WeChatCallBack]: 生成JWT Token成功: %s", token)

	// 根据 state 参数决定重定向目标页面（未匹配时使用默认地址）
	state := c.Query("state")
	redirectURL := h.buildRedirectURL(token, state)

	log.Printf("[info][auth/handler/WeChatCallBack]: 重定向到: %s", redirectURL)
	c.Redirect(http.StatusFound, redirectURL)
}

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

// LoginHandler 管理员登录接口（Gateway 端）。
//
//	@Summary		管理员登录（Gateway）
//	@Description	验证管理员用户名密码（直接查询数据库），签发 Admin JWT Token
//	@Tags			网关-管理员认证
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginReq	true	"管理员登录凭证"
//	@Success		200		{object}	AdminResp{data=LoginResp}	"登录成功"
//	@Failure		400		{object}	AdminResp	"参数错误"
//	@Failure		401		{object}	AdminResp	"凭证错误"
//	@Router			/api/v1/auth/admin/login [post]
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

// GetAdminInfoHandler 获取当前管理员信息（Gateway 端）。
//
//	@Summary		获取当前管理员信息（Gateway）
//	@Description	根据 JWT Token 返回当前登录管理员的详细信息
//	@Tags			网关-管理员认证
//	@Produce		json
//	@Success		200	{object}	AdminResp{data=AdminInfoResp}	"管理员信息"
//	@Failure		401	{object}	AdminResp	"未登录"
//	@Security		BearerAuth
//	@Router			/api/v1/auth/admin/info [get]
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
