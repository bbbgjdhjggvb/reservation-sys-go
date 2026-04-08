#!/bin/bash
# 生成微信菜单 OAuth URL 工具

echo "=========================================="
echo "  微信菜单 OAuth URL 生成工具"
echo "=========================================="
echo ""

# 读取配置
read -p "请输入微信 AppID [默认: wx84d6833105361902]: " APP_ID
APP_ID=${APP_ID:-wx84d6833105361902}

read -p "请输入服务器地址 (IP 或域名): " SERVER_ADDR

if [ -z "$SERVER_ADDR" ]; then
    echo "[错误] 服务器地址不能为空"
    exit 1
fi

read -p "使用 HTTPS? [y/N]: " USE_HTTPS
if [[ "$USE_HTTPS" =~ ^[Yy]$ ]]; then
    PROTOCOL="https"
else
    PROTOCOL="http"
fi

read -p "授权作用域 [1=snsapi_base, 2=snsapi_userinfo, 默认:1]: " SCOPE_CHOICE
case $SCOPE_CHOICE in
    2) SCOPE="snsapi_userinfo" ;;
    *) SCOPE="snsapi_base" ;;
esac

read -p "State 参数 [可选，默认为空]: " STATE

# 构建回调 URL
CALLBACK_URL="${PROTOCOL}://${SERVER_ADDR}/api/v1/auth/callback"

# URL 编码
ENCODED_CALLBACK=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$CALLBACK_URL', safe=''))")

# 构建完整 URL
if [ -n "$STATE" ]; then
    OAUTH_URL="https://open.weixin.qq.com/connect/oauth2/authorize?appid=${APP_ID}&redirect_uri=${ENCODED_CALLBACK}&response_type=code&scope=${SCOPE}&state=${STATE}#wechat_redirect"
else
    OAUTH_URL="https://open.weixin.qq.com/connect/oauth2/authorize?appid=${APP_ID}&redirect_uri=${ENCODED_CALLBACK}&response_type=code&scope=${SCOPE}#wechat_redirect"
fi

# 输出结果
echo ""
echo "=========================================="
echo "  生成结果"
echo "=========================================="
echo ""
echo "📋 回调地址: ${CALLBACK_URL}"
echo "📋 编码后:   ${ENCODED_CALLBACK}"
echo ""
echo "✅ 完整 OAuth URL:"
echo ""
echo "${OAUTH_URL}"
echo ""
echo "📝 菜单配置示例:"
echo ""
cat << EOF
{
    "type": "view",
    "name": "预约场地",
    "url": "${OAUTH_URL}"
}
EOF
echo ""
echo "=========================================="
