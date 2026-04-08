# 本项目的鉴权流程
微信OAuth → authHdl.WeChatCallBack → 生成 JWT → 前端携带 Bearer token → AuthMiddleware → c.Set("openid") → Handler