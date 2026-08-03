// 微信 OAuth URL 生成工具
//
// 用于生成微信网页授权 URL 和菜单配置示例，支持命令行参数和交互式输入两种模式。
//
// 用法:
//
//	go run tools/oauth/main.go [选项]
//
// 选项:
//
//	-a, --appid     微信 AppID（默认: wx84d6833105361902）
//	-s, --server    服务器地址（IP 或域名，必填）
//	-t, --https     使用 HTTPS（默认: false）
//	-c, --scope     授权作用域 base 或 userinfo（默认: base）
//	-e, --state     State 参数（可选，默认为空）
//
// 不带 --server 参数时将进入交互式模式。
//
// ----------------------------------------------------------------------------------------------
//
// OAuth
//
// 1. OAuth 的出现是为了解决什么问题？
//
// 在 OAuth 之前，如果我想在一个应用中获取，另外一个服务的信息。比如我的一个视频剪辑软件，想要获取网盘的视频资源。
// 我就需要把我网盘的账号，密码告诉我的这个应用。这样的安全隐患很高。
//
// OAuth 就是允许第三方应用，在不获取用户账号，密码的前提下，获取用户保存在一个服务器上的信息或者资源。
//
// 2. 为什么我们的系统涉及到 OAuth？
//
// 在我们的系统中存在这样几个角色：想要预约的用户、微信这个应用、我们校友会的服务号。
// 我们的服务号需要用户授权，从而获取微信账号的一些信息。
//
// 3. OAuth 是怎么运行的
//
// 我们的服务号有一个 appid，这个 appid 即是我们服务号的唯一标识，也是微信自己通过某种算法算出的，标识这是一个微信应用的一个字符串。
// 只要服务号有这个 appid，微信程序就会信任我们这个服务号，就允许我们的服务号获取用户的账号信息。
//
// 微信自己有一个叫做“授权服务器”的东西。它专门用来处理授权事务。这个服务器的域名就是 https://open.weixin.qq.com/connect/oauth2/authorize。
// 我们要获取授权，就需要向这个服务器发送请求。请求参数中需要携带下面这些信息。
// 	- appid：服务号唯一标识。
// 	- redirect_uri：我们服务号背后的服务器，用来处理微信授权服务器发过来的信息，我们就是要在这个信息中，获取用户的账号信息。
// 	- response_type：固定值 code，表示获取授权码，access_token。微信授权服务器会向我们服务号的 redirect_uri 发送一个 code，表示授权码。这个 code 以请求参数的形式发送，需要我们自己去解析。
// 	- scope：snsapi_base 标识静默授权，用户感知不到这个授权过程，会直接跳转到 redirect_uri。
// 后面的请求参数就是拼接到 redirect_uri 后面的参数。相当于用户跳转的连接是 https://www.szuedf.org.cn/api/gateway/auth/callback?code=CODE&state=STATE。
// 我们从上面的请求链接中解析出 code。用这个 code 去获取微信用户的 openid。
//
// -----------------------------------------------------------------------------------------------
//
// 当我们把重定向连接作为参数时，里面的 "//" 和 url 冲突，需要被转换为其他字符
//
// -----------------------------------------------------------------------------------------------

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
)

const defaultAppID = "wx84d6833105361902"

func main() {
	var (
		appID     string
		server    string
		useHTTPS  bool
		scopeName string
		state     string
	)

	// 解析命令行参数
	appIDFlag := flag.String("a", "", "")
	serverFlag := flag.String("s", "", "")
	httpsFlag := flag.Bool("t", false, "")
	scopeFlag := flag.String("c", "", "")
	stateFlag := flag.String("e", "", "")

	// 支持长选项
	for i, arg := range os.Args {
		if i == 0 {
			continue
		}
		switch arg {
		case "--appid":
			if i+1 < len(os.Args) {
				*appIDFlag = os.Args[i+1]
			}
		case "--server":
			if i+1 < len(os.Args) {
				*serverFlag = os.Args[i+1]
			}
		case "--https":
			*httpsFlag = true
		case "--scope":
			if i+1 < len(os.Args) {
				*scopeFlag = os.Args[i+1]
			}
		case "--state":
			if i+1 < len(os.Args) {
				*stateFlag = os.Args[i+1]
			}
		}
	}
	flag.Parse()

	appID = *appIDFlag
	server = *serverFlag
	useHTTPS = *httpsFlag
	scopeName = *scopeFlag
	state = *stateFlag

	// 交互式模式：未提供 server 时提示输入
	if server == "" {
		fmt.Println("==========================================")
		fmt.Println("  微信菜单 OAuth URL 生成工具")
		fmt.Println("==========================================")
		fmt.Println()

		// AppID
		fmt.Printf("请输入微信 AppID [默认: %s]: ", defaultAppID)
		fmt.Scanln(&appID)
		if appID == "" {
			appID = defaultAppID
		}

		// 服务器地址
		fmt.Print("请输入服务器地址 (IP 或域名): ")
		fmt.Scanln(&server)
		if server == "" {
			fmt.Println("[错误] 服务器地址不能为空")
			os.Exit(1)
		}

		// HTTPS
		fmt.Print("使用 HTTPS? [y/N]: ")
		var httpsInput string
		fmt.Scanln(&httpsInput)
		useHTTPS = strings.EqualFold(httpsInput, "y") || strings.EqualFold(httpsInput, "yes")

		// 授权作用域
		fmt.Print("授权作用域 [1=snsapi_base, 2=snsapi_userinfo, 默认:1]: ")
		var scopeChoice string
		fmt.Scanln(&scopeChoice)
		if scopeChoice == "2" {
			scopeName = "snsapi_userinfo"
		} else {
			scopeName = "snsapi_base"
		}

		// State
		fmt.Print("State 参数 [可选，默认为空]: ")
		fmt.Scanln(&state)
	} else {
		// 命令行模式：设置默认值
		if appID == "" {
			appID = defaultAppID
		}
		switch scopeName {
		case "userinfo", "snsapi_userinfo":
			scopeName = "snsapi_userinfo"
		default:
			scopeName = "snsapi_base"
		}
	}

	// 构建回调 URL
	protocol := "http"
	if useHTTPS {
		protocol = "https"
	}
	callbackURL := fmt.Sprintf("%s://%s/api/gateway/auth/callback", protocol, server)
	encodedCallback := url.QueryEscape(callbackURL)

	// 构建 OAuth URL
	oauthURL := fmt.Sprintf(
		"https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=%s",
		appID, encodedCallback, scopeName,
	)
	if state != "" {
		oauthURL += "&state=" + url.QueryEscape(state)
	}
	oauthURL += "#wechat_redirect"

	// 构建菜单配置
	menuConfig := map[string]string{
		"type": "view",
		"name": "预约场地",
		"url":  oauthURL,
	}
	menuJSON, _ := json.MarshalIndent(menuConfig, "", "    ")

	// 输出结果
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("  生成结果")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Printf("📋 回调地址: %s\n", callbackURL)
	fmt.Printf("📋 编码后:   %s\n", encodedCallback)
	fmt.Println()
	fmt.Println("✅ 完整 OAuth URL:")
	fmt.Println()
	fmt.Println(oauthURL)
	fmt.Println()
	fmt.Println("📝 菜单配置示例:")
	fmt.Println()
	fmt.Println(string(menuJSON))
	fmt.Println()
	fmt.Println("==========================================")
}
