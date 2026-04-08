# auth 包单元测试指南

## 测试文件说明

auth 包包含以下测试文件：

1. **middleware_test.go** - 测试 JWT 认证中间件
   - `TestAuthMiddleware_NoAuthorizationHeader` - 测试缺少 Authorization 头
   - `TestAuthMiddleware_InvalidTokenFormat` - 测试各种无效的 token 格式
   - `TestAuthMiddleware_InvalidToken` - 测试伪造/无效的 token
   - `TestAuthMiddleware_ValidToken` - 测试有效的 token
   - `TestAuthMiddleware_AbortPreventsNextHandler` - 测试中间件拦截功能
   - `TestAuthMiddleware_ResponseIsJSON` - 测试响应格式

2. **handler_test.go** - 测试认证处理器
   - `TestWeChatCallBack_MissingCode` - 测试缺少 code 参数
   - `TestWeChatCallBack_EmptyCode` - 测试空的 code 参数
   - `TestUser_TableName` - 测试 User 模型的表名
   - `TestNewUserAuthHandler` - 测试 Handler 构造函数
   - `TestWeChatCallBack_ResponseIsJSON` - 测试响应格式

3. **testutil_test.go** - 测试工具
   - `TestMain` - 测试前的配置初始化

## 运行测试

在项目根目录执行：

```bash
# 运行 auth 包的所有测试
go test ./internal/auth/ -v

# 运行特定测试
go test ./internal/auth/ -v -run TestAuthMiddleware_ValidToken

# 查看测试覆盖率
go test ./internal/auth/ -cover
```

## 测试覆盖率

当前测试覆盖的代码路径：

### ✅ 已覆盖
- JWT 中间件的所有分支（缺少头、格式错误、无效 token、有效 token）
- WeChatCallBack 处理缺少/空 code 参数的场景
- JSON 响应格式验证
- Handler 和 Model 的基本功能

### ⚠️ 需要重构才能测试
- `WeChatCallBack` 中 `LoginByCode` 失败的分支（依赖微信 SDK）
- `LoginByCode` 成功的完整流程（需要 mock OAuth）

**重构建议**：将 `UserAuthService` 改为依赖接口而非具体类型，参考 `reservation` 包的 `ReservationRepository` 接口模式。

## 测试架构说明

### JWT 懒加载

为了避免包导入时的循环依赖问题，`jwt` 包采用了懒加载模式：

```go
var (
    secret     []byte
    secretOnce sync.Once
)

func getJWTSecret() []byte {
    secretOnce.Do(func() {
        cfg := config.GetConfig()
        if cfg == nil || cfg.Jwt.Secret == "" {
            secret = []byte("test-default-secret")
        } else {
            secret = []byte(cfg.Jwt.Secret)
        }
    })
    return secret
}
```

这样即使 `config` 在测试时还未初始化，也不会导致 `panic`。

### TestMain 初始化

`TestMain` 函数在所有测试执行前运行，用于：
1. 创建临时配置文件
2. 加载 JWT 配置
3. 确保 `jwt` 包在首次使用时能正确获取密钥

```go
func TestMain(m *testing.M) {
    tmpDir := os.TempDir()
    configPath := tmpDir + "/auth_test_config.yaml"

    testConfig := `jwt:
  secret: "test-secret-key"
  expire_time: 24`

    os.WriteFile(configPath, []byte(testConfig), 0644)
    config.LoadConfig(configPath)

    os.Exit(m.Run())
}
```

## 测试用例编写规范

### Table-Driven Tests

使用表格驱动测试来覆盖多个场景：

```go
tests := []struct {
    name       string
    authVal    string
    expectMsg  string
}{
    {
        name:      "无Bearer前缀",
        authVal:   "some-random-token",
        expectMsg: "Token格式错误",
    },
    // ... more cases
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // test logic
    })
}
```

### 使用 httptest

所有 HTTP Handler 和 Middleware 测试都使用 `httptest.ResponseRecorder`：

```go
w := httptest.NewRecorder()
req, _ := http.NewRequest("GET", "/protected", nil)
req.Header.Set("Authorization", "Bearer "+token)
r.ServeHTTP(w, req)

assert.Equal(t, http.StatusOK, w.Code)
```

### 断言库

使用 `github.com/stretchr/testify/assert` 提供清晰的断言：

```go
assert.Equal(t, expected, actual)
assert.NoError(t, err)
assert.Contains(t, str, substring)
```
