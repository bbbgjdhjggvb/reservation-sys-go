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
	"reservation-sys/internal/pkg/jwt"

	"github.com/gin-gonic/gin"
)

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
	token, err := jwt.GenerateToken(openid)
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
