package auth

import (
	"github.com/silenceper/wechat/v2/officialaccount"
)

// wechatOAuthClient 微信OAuth客户端适配器
type wechatOAuthClient struct {
	oa *officialaccount.OfficialAccount
}

// NewWechatOAuthClient 创建微信OAuth客户端
func NewWechatOAuthClient(oa *officialaccount.OfficialAccount) OAuthClient {
	return &wechatOAuthClient{oa: oa}
}

func (c *wechatOAuthClient) GetUserAccessToken(code string) (*OAuthAccessTokenResult, error) {
	oauth := c.oa.GetOauth()
	res, err := oauth.GetUserAccessToken(code)
	if err != nil {
		return nil, err
	}
	return &OAuthAccessTokenResult{OpenID: res.OpenID}, nil
}

func (c *wechatOAuthClient) GetUserInfo(openid string) string {
	return fetchWechatNickname(c.oa, openid)
}

// wechatUserInfoProvider 微信用户信息提供者
type wechatUserInfoProvider struct {
	oa *officialaccount.OfficialAccount
}

// NewWechatUserInfoProvider 创建微信用户信息提供者
func NewWechatUserInfoProvider(oa *officialaccount.OfficialAccount) UserInfoProvider {
	return &wechatUserInfoProvider{oa: oa}
}

func (p *wechatUserInfoProvider) GetUserInfo(openid string) string {
	return fetchWechatNickname(p.oa, openid)
}

// fetchWechatNickname 从微信API获取用户昵称（公共逻辑，避免重复）
func fetchWechatNickname(oa *officialaccount.OfficialAccount, openid string) string {
	user, err := oa.GetUser().GetUserInfo(openid)
	if err != nil {
		return ""
	}
	return user.Nickname
}
