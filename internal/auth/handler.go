package auth

import (
	"log"
	"net/http"

	"reservation-sys/internal/middleware"
	"reservation-sys/internal/model"
	"reservation-sys/pkg/config"
	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/silenceper/wechat/v2/officialaccount"
	"github.com/silenceper/wechat/v2/officialaccount/message"
)

// ========== DTO ==========

// LoginReq 管理员登录请求
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResp 管理员登录响应
type LoginResp struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	RealName string `json:"real_name"`
	Role     int    `json:"role"`
	RoleText string `json:"role_text"`
}

// AdminInfoResp 管理员信息响应
type AdminInfoResp struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	RealName string `json:"real_name"`
	Role     int    `json:"role"`
	RoleText string `json:"role_text"`
}

// ========== Handler ==========

// Handler 认证模块 HTTP 处理器
type Handler struct {
	svc *Service
	oa  *officialaccount.OfficialAccount
	cfg *config.WechatConfig
}

// NewHandler 创建认证处理器实例
func NewHandler(svc *Service, oa *officialaccount.OfficialAccount, cfg *config.WechatConfig) *Handler {
	return &Handler{svc: svc, oa: oa, cfg: cfg}
}

// ========== 微信消息处理 ==========

// WeChatHandler 处理微信服务器消息（GET 验证 + POST 接收消息/事件）。
// 使用微信公众号 SDK 的 GetServer 统一处理签名验证和消息分发。
func (h *Handler) WeChatHandler(c *gin.Context) {
	server := h.oa.GetServer(c.Request, c.Writer)
	server.SetMessageHandler(func(msg *message.MixMessage) *message.Reply {
		notifier := &Notifier{} // 仅用于 ProcessMessage 方法
		return notifier.ProcessMessage(h.svc, msg)
	})
	server.Serve()
	server.Send()
}

// WeChatCallBack 处理微信 OAuth 回调。
// 用 code 换取 openid，签发用户 JWT，重定向到前端页面。
func (h *Handler) WeChatCallBack(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		errcode := c.Query("errcode")
		if errcode != "" {
			errmsg := c.Query("errmsg")
			log.Printf("[info][auth/handler] 微信授权失败: errcode=%s, errmsg=%s", errcode, errmsg)
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"msg": "缺少 code 参数，从微信服务号进入预约界面",
		})
		return
	}

	log.Printf("[info][auth/handler] 收到微信回调请求，code: %s", code)

	openid, err := h.svc.LoginByCode(code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"msg": "微信授权失效"})
		return
	}
	log.Printf("[info][auth/handler] 微信授权成功，openid: %s", openid)

	// 签发用户 JWT
	token, err := jwt.GenerateUserToken(openid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "服务器内部错误，Token 生成失败"})
		return
	}

	// 根据 state 参数决定重定向目标
	state := c.Query("state")
	redirectURL := h.cfg.DefaultRedirect
	if url, ok := h.cfg.RedirectURLs[state]; ok && url != "" {
		redirectURL = url
	}
	redirectURL += "?token=" + token

	log.Printf("[info][auth/handler] 重定向到: %s", redirectURL)
	c.Redirect(http.StatusFound, redirectURL)
}

// ========== 管理员认证 Handler ==========

// AdminLogin 管理员登录接口。
func (h *Handler) AdminLogin(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数错误"})
		return
	}

	admin, token, err := h.svc.AdminLogin(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "登录成功",
		"data": &LoginResp{
			Token:    token,
			Username: admin.Username,
			RealName: admin.RealName,
			Role:     admin.Role,
			RoleText: model.RoleText(admin.Role),
		},
	})

	log.Printf("[info][auth/handler] admin=%s(%s) role=%d login success",
		admin.Username, admin.RealName, admin.Role)
}

// GetAdminInfo 获取当前管理员信息。
func (h *Handler) GetAdminInfo(c *gin.Context) {
	claims, exists := middleware.GetAdminInfo(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "success",
		"data": &AdminInfoResp{
			ID:       claims.AdminID,
			Username: claims.Username,
			Role:     claims.Role,
			RoleText: model.RoleText(claims.Role),
		},
	})
}
