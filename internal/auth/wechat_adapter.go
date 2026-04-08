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
	user, err := c.oa.GetUser().GetUserInfo(openid)
	if err != nil {
		return ""
	}
	return user.Nickname
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
	user, err := p.oa.GetUser().GetUserInfo(openid)
	if err != nil {
		return ""
	}
	return user.Nickname
}
