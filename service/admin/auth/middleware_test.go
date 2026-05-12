package auth

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"reservation-sys/pkg/jwt"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupMiddlewareTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func generateValidAdminToken(t *testing.T) string {
	t.Helper()
	token, err := jwt.GenerateAdminToken(1, "admin1", 1)
	assert.NoError(t, err)
	return token
}

func TestAdminAuthMiddleware(t *testing.T) {
	r := setupMiddlewareTestRouter()
	r.Use(AdminAuthMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	tests := []struct {
		name       string
		authHeader string
		wantCode   int
		wantMsg    string
	}{
		{
			name:       "no_header",
			authHeader: "",
			wantCode:   401,
			wantMsg:    "未登录",
		},
		{
			name:       "bad_format_no_bearer",
			authHeader: "token123",
			wantCode:   401,
			wantMsg:    "Token格式错误",
		},
		{
			name:       "bad_format_wrong_prefix",
			authHeader: "Basic token123",
			wantCode:   401,
			wantMsg:    "Token格式错误",
		},
		{
			name:       "invalid_token",
			authHeader: "Bearer invalid.token.here",
			wantCode:   401,
			wantMsg:    "Token无效或已过期",
		},
		{
			name:       "valid_token",
			authHeader: "Bearer " + generateValidAdminToken(t),
			wantCode:   200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantMsg != "" {
				var resp AdminResp
				json.Unmarshal(w.Body.Bytes(), &resp)
				assert.Contains(t, resp.Msg, tt.wantMsg)
			}
		})
	}
}

func TestRoleMiddleware(t *testing.T) {
	t.Run("no_admin_in_context", func(t *testing.T) {
		r := setupMiddlewareTestRouter()
		r.Use(RoleMiddleware(1))
		r.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
	})

	t.Run("wrong_role_returns_403", func(t *testing.T) {
		r := setupMiddlewareTestRouter()
		r.Use(func(c *gin.Context) {
			c.Set("admin", &jwt.AdminClaims{AdminID: 2, Username: "admin2", Role: 2})
			c.Next()
		})
		r.Use(RoleMiddleware(1))
		r.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 403, w.Code)
	})

	t.Run("correct_role_passes", func(t *testing.T) {
		r := setupMiddlewareTestRouter()
		r.Use(func(c *gin.Context) {
			c.Set("admin", &jwt.AdminClaims{AdminID: 1, Username: "admin1", Role: 1})
			c.Next()
		})
		r.Use(RoleMiddleware(1))
		r.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("multiple_allowed_roles", func(t *testing.T) {
		r := setupMiddlewareTestRouter()
		r.Use(func(c *gin.Context) {
			c.Set("admin", &jwt.AdminClaims{AdminID: 2, Username: "admin2", Role: 2})
			c.Next()
		})
		r.Use(RoleMiddleware(1, 2))
		r.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}
