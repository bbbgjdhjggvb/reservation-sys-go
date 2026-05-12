// Token 生成工具
// 用于本地测试时生成用户 JWT Token，跳过微信 OAuth 流程
//
// 用法:
//
//	go run tools/jwt/gen_token.go                 # 使用默认 openid (test_openid_local_001)
//	go run tools/jwt/gen_token.go test_user_001   # 指定 openid
//
// 注意: JWT secret 需与配置文件中 jwt.secret 保持一致
package main

import (
	"fmt"
	"os"

	"reservation-sys/pkg/jwt"
)

func main() {
	openid := "test_openid_local_001"
	if len(os.Args) > 1 {
		openid = os.Args[1]
	}

	jwt.InitUserJWT("Y6Xoo746BoVCWFyFUVSqqboCfqo7QkC8A5CN7F9sOm0=", 24)

	token, err := jwt.GenerateUserToken(openid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成 token 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(token)
}
