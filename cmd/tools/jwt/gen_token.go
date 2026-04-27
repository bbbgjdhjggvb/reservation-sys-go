// Token 生成工具
// 用于本地测试时生成 JWT Token，跳过微信 OAuth 流程
//
// 用法:
//
//	go run cmd/tools/jwt/gen_token.go               # 使用默认 openid
//	go run cmd/tools/jwt/gen_token.go test_user_001  # 指定 openid
//
// 注意: JWT secret 硬编码，需与 config_v2.debug.yaml 中的 jwt.secret 保持一致
package main

import (
	"fmt"
	"os"

	"reservation-sys/internal/pkg/jwt"
)

const defaultJWTSecret = "Y6Xoo746BoVCWFyFUVSqqboCfqo7QkC8A5CN7F9sOm0="

func main() {
	openid := "test_openid_local_001"
	if len(os.Args) > 1 {
		openid = os.Args[1]
	}

	// 必须和 config_v2.debug.yaml 中的 jwt.secret 一致
	jwt.Init(defaultJWTSecret, 24)

	token, err := jwt.GenerateToken(openid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成 token 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("========================================")
	fmt.Println("JWT Token 生成成功")
	fmt.Printf("OpenID: %s\n", openid)
	fmt.Println("========================================")
	fmt.Println(token)
	fmt.Println("========================================")
	fmt.Println("使用方式:")
	fmt.Println("  curl -H \"Authorization: Bearer <token>\" ...")
	fmt.Println("========================================")
}
