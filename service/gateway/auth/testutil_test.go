package auth

import (
	"os"
	"testing"

	"reservation-sys/pkg/jwt"
)

// TestMain 在所有测试执行前运行，用于初始化 JWT
func TestMain(m *testing.M) {
	// 初始化 JWT（测试用密钥）
	jwt.InitUserJWT("test-secret-key-for-unit-test-do-not-use-in-production", 24)
	os.Exit(m.Run())
}
