// Package integration 提供端到端集成测试基础设施。
//
// 测试前需通过 docker-compose 启动完整服务栈，测试通过 nginx (localhost:80) 发送真实 HTTP 请求，
// 验证完整链路：
//
//	HTTP 请求 → nginx 反向代理 → 服务容器 → gRPC(服务间) → MySQL/Redis → 响应
//
// 运行方式（必须通过脚本，脚本负责启动/停止服务）:
//
//	bash scripts/e2e_test.sh              # 一键构建+测试+清理
//	bash scripts/e2e_test.sh -run TestXxx # 运行指定测试
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"reservation-sys/pkg/jwt"
	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	e2eBaseURL   = "http://localhost"
	e2eMySQLDSN  = "res_user:xSIn34sU7qQl31kQ3TVfcQ==@tcp(127.0.0.1:3307)/home_res?charset=utf8mb4&parseTime=True&loc=Local"
	e2eRedisAddr = "127.0.0.1:6380"
	e2eJWTSecret = "Y6Xoo746BoVCWFyFUVSqqboCfqo7QkC8A5CN7F9sOm0="
	e2eJWTExpire = 24
)

var (
	e2eDB    *gorm.DB
	e2eRedis *redis.Client
)

// 测试 能否正常连接到 nginx、mysql、redis
func TestMain(m *testing.M) {
	// 检查服务是否已启动（由 scripts/e2e_test.sh 负责）
	if !checkService(e2eBaseURL + "/health") {
		fmt.Println("[e2e] 服务未启动，请先运行: bash scripts/e2e_test.sh")
		os.Exit(1)
	}

	// 初始化 JWT（使用与线上相同的 secret）
	jwt.InitUserJWT(e2eJWTSecret, e2eJWTExpire)
	jwt.InitAdminJWT(e2eJWTSecret, e2eJWTExpire)

	// 连接 MySQL
	var err error
	e2eDB, err = gorm.Open(mysql.Open(e2eMySQLDSN), &gorm.Config{})
	if err != nil {
		fmt.Printf("[e2e] 连接 MySQL 失败: %v\n", err)
		os.Exit(1)
	}

	// 连接 Redis
	e2eRedis = redis.NewClient(&redis.Options{Addr: e2eRedisAddr})
	if err := e2eRedis.Ping(e2eRedis.Context()).Err(); err != nil {
		fmt.Printf("[e2e] 连接 Redis 失败: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	e2eRedis.Close()
	os.Exit(code)
}

// ---------- 服务就绪检测 ----------

func checkService(url string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ---------- 测试辅助 ----------

// skipIfNoDocker 在没有 Docker 时跳过集成测试。
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if err := checkDocker(); err != nil {
		t.Skip("Docker 不可用，跳过 E2E 测试")
	}
}

func checkDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return err
	}
	return exec.Command("docker", "info").Run()
}

// e2eHTTPClient 返回用于请求 nginx 的 HTTP 客户端。
func e2eHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// newRepo 返回连接 e2e MySQL 的 repository 和清理函数（截断所有表）。
func newRepo(t *testing.T) (reservationdb.Repository, func()) {
	t.Helper()
	skipIfNoDocker(t)
	repo := reservationdb.NewRepository(e2eDB)
	cleanup := func() {
		e2eDB.Exec("DELETE FROM review_records")
		e2eDB.Exec("DELETE FROM reservation_slots")
		e2eDB.Exec("DELETE FROM reservation_orders")
	}
	return repo, cleanup
}

// newRedisClient 返回连接 e2e Redis 的客户端和清库清理函数。
func newRedisClient(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	skipIfNoDocker(t)
	ctx := e2eRedis.Context()
	cleanup := func() {
		e2eRedis.FlushDB(ctx)
	}
	return e2eRedis, cleanup
}

// mustParseTime 解析时间字符串，失败时 panic。
func mustParseTime(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

// genUserToken 生成测试用用户 JWT。
func genUserToken(t *testing.T, openid string) string {
	t.Helper()
	token, err := jwt.GenerateUserToken(openid)
	if err != nil {
		t.Fatalf("生成用户 token 失败: %v", err)
	}
	return token
}

// genAdminToken 生成测试用管理员 JWT。
func genAdminToken(t *testing.T, adminID uint, username string, role int) string {
	t.Helper()
	token, err := jwt.GenerateAdminToken(adminID, username, role)
	if err != nil {
		t.Fatalf("生成管理员 token 失败: %v", err)
	}
	return token
}

// ---------- HTTP 请求辅助 ----------

// doRequest 向 nginx 发送 HTTP 请求并返回响应。
//
// 参数:
//   - method: HTTP 方法（GET/POST/PUT/DELETE）
//   - path: 请求路径（如 /api/reservation/reservation/submit）
//   - token: JWT Bearer Token，为空时不添加 Authorization 头
//   - body: 请求体 JSON 字符串，为空时无 body
//
// 返回值:
//   - *http.Response: HTTP 响应
//   - error: 请求失败时返回错误
func doRequest(method, path, token, body string) (*http.Response, error) {
	url := e2eBaseURL + path
	var reqBody io.Reader
	if body != "" {
		reqBody = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return e2eHTTPClient().Do(req)
}

// doRequestJSON 发送请求并将 JSON 响应解析到 v。
func doRequestJSON(t *testing.T, method, path, token, body string, v any) *http.Response {
	t.Helper()
	resp, err := doRequest(method, path, token, body)
	require.NoError(t, err, "HTTP 请求失败")
	if v != nil {
		defer resp.Body.Close()
		respBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "读取响应体失败")
		if len(respBytes) > 0 {
			require.NoError(t, json.Unmarshal(respBytes, v), "解析 JSON 响应失败")
		}
	}
	return resp
}

// assertOK 断言响应状态码为 200。
func assertOK(t *testing.T, resp *http.Response, msgAndArgs ...any) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("期望 200, 实际 %d, body: %s", resp.StatusCode, string(body))
	}
}

// e2eSlotCounter 为每个 createOrder 调用分配唯一时段，避免固定时间冲突。
var e2eSlotCounter atomic.Int64

// createOrder 通过 repository 直接创建订单（用于测试数据准备）。
func createOrder(t *testing.T, repo reservationdb.Repository, status int, openid string) uint {
	t.Helper()
	n := int(e2eSlotCounter.Add(1))
	start := time.Date(2026, 7, 1, 8, 0, 0, 0, time.Local).Add(time.Duration((n-1)*2) * time.Hour)
	end := start.Add(2 * time.Hour)
	order := &reservationdb.ReservationOrder{
		OrderNo:           fmt.Sprintf("R_E2E_%d_%d", status, time.Now().UnixNano()),
		OpenID:            openid,
		ApplicantName:     "测试用户",
		AlumniAssociation: "校友会",
		Year:              2020,
		Major:             "CS",
		Reason:            "E2E测试",
		Phone:             "13800138000",
		TotalSlots:        1,
		Status:            status,
	}
	slots := []reservationdb.ReservationSlot{
		{StartTime: start, EndTime: end, Status: status},
	}
	err := repo.CreateOrderWithLock(order, slots)
	require.NoError(t, err)
	return order.ID
}

// httpStatus 断言响应状态码为期望值。
func httpStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	require.Equal(t, expected, resp.StatusCode, "期望 %d, 实际 %d", expected, resp.StatusCode)
}

// readBody 读取响应体为字符串。
func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

// strContains 断言字符串包含子串。
func strContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("期望包含 %q, 实际: %s", substr, s)
	}
}
