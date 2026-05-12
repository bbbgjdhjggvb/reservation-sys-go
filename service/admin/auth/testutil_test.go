package auth

import (
	"os"
	"testing"

	"reservation-sys/pkg/jwt"
)

func TestMain(m *testing.M) {
	jwt.InitAdminJWT("test-admin-secret-for-unit-test", 24)
	os.Exit(m.Run())
}
