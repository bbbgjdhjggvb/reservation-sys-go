// Package integration 提供数据库+Redis集成测试基础设施。
// TestMain 管理单个 MySQL 容器和单个 Redis 容器的生命周期，所有测试共享。
package integration

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"reservation-sys/pkg/jwt"
	reservationdb "reservation-sys/pkg/reservationdb"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	testDBUser     = "res_user"
	testDBPassword = "12345678"
	testDBName     = "home_res"
)

var (
	testMySQLDSN   string
	testRedisAddr  string
	mysqlCleanupFn func()
	redisCleanupFn func()
)

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	testMySQLDSN, mysqlCleanupFn = setupMySQLContainer()
	testRedisAddr, redisCleanupFn = setupRedisContainer()

	jwt.InitUserJWT("integration-test-user-secret", 24)
	jwt.InitAdminJWT("integration-test-admin-secret", 24)

	code := m.Run()

	if redisCleanupFn != nil {
		redisCleanupFn()
	}
	if mysqlCleanupFn != nil {
		mysqlCleanupFn()
	}
	os.Exit(code)
}

// ---------- MySQL ----------

func setupMySQLContainer() (string, func()) {
	if err := checkDocker(); err != nil {
		fmt.Printf("[integration] Docker 不可用 (%v)，跳过 MySQL 容器启动\n", err)
		return "", nil
	}

	port := fmt.Sprintf("%d", 33100+rand.Intn(100))
	containerName := fmt.Sprintf("reservation-test-mysql-%d", time.Now().UnixNano())

	cmd := exec.Command("docker", "run", "-d", "--rm",
		"--name", containerName,
		"-e", "MYSQL_ROOT_PASSWORD=root123",
		"-p", fmt.Sprintf("%s:3306", port),
		"mysql:8.0",
		"--default-authentication-plugin=mysql_native_password",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[integration] 启动 MySQL 容器失败: %v, output: %s\n", err, string(output))
		return "", nil
	}
	containerID := strings.TrimSpace(string(output))

	cleanup := func() { exec.Command("docker", "stop", containerID).Run() }

	rootDSN := fmt.Sprintf("root:root123@tcp(127.0.0.1:%s)/?charset=utf8mb4&parseTime=True&loc=Local", port)
	var db *gorm.DB
	for i := 0; i < 60; i++ {
		db, err = gorm.Open(mysql.Open(rootDSN), &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		cleanup()
		panic(fmt.Sprintf("MySQL 容器启动超时: %v", err))
	}

	setupSQLs := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", testDBName),
		fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s'", testDBUser, testDBPassword),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%%'", testDBName, testDBUser),
		"FLUSH PRIVILEGES",
	}
	for _, sql := range setupSQLs {
		if err := db.Exec(sql).Error; err != nil {
			cleanup()
			panic(fmt.Sprintf("执行初始化 SQL 失败: %s, err: %v", sql, err))
		}
	}

	userDSN := fmt.Sprintf("%s:%s@tcp(127.0.0.1:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		testDBUser, testDBPassword, port, testDBName)
	testDB, err := gorm.Open(mysql.Open(userDSN), &gorm.Config{})
	if err != nil {
		cleanup()
		panic(fmt.Sprintf("连接测试数据库失败: %v", err))
	}
	if err := runInitSQL(testDB); err != nil {
		cleanup()
		panic(fmt.Sprintf("创建表结构失败: %v", err))
	}

	return userDSN, cleanup
}

func runInitSQL(db *gorm.DB) error {
	sqls := []string{
		`CREATE TABLE IF NOT EXISTS reservation_orders (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			order_no VARCHAR(50) NOT NULL COMMENT '订单号',
			open_id VARCHAR(100) NOT NULL COMMENT '微信用户标识',
			applicant_name VARCHAR(50) NOT NULL COMMENT '申请人姓名',
			alumni_association VARCHAR(100) NOT NULL COMMENT '所属学院校友会',
			year INT NOT NULL COMMENT '入学年份',
			major VARCHAR(30) NOT NULL COMMENT '专业',
			reason VARCHAR(500) NOT NULL COMMENT '会议内容/预约理由',
			phone VARCHAR(20) NOT NULL COMMENT '联系电话',
			total_slots TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '预约时段数量',
			status TINYINT NOT NULL DEFAULT 0 COMMENT '整体状态',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY idx_orders_order_no (order_no),
			KEY idx_orders_open_id (open_id),
			KEY idx_orders_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS reservation_slots (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			order_id BIGINT UNSIGNED NOT NULL COMMENT '关联订单ID',
			start_time DATETIME NOT NULL COMMENT '开始时间',
			end_time DATETIME NOT NULL COMMENT '结束时间',
			status TINYINT NOT NULL DEFAULT 0 COMMENT '时段状态',
			password VARCHAR(20) DEFAULT NULL COMMENT '门锁动态密码',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_slots_order_id (order_id),
			KEY idx_slots_time_range (start_time, end_time),
			KEY idx_slots_status (status),
			CONSTRAINT fk_slots_order_id FOREIGN KEY (order_id) REFERENCES reservation_orders(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,

		`CREATE TABLE IF NOT EXISTS review_records (
			id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			order_id BIGINT UNSIGNED NOT NULL COMMENT '关联订单ID',
			reviewer_id BIGINT UNSIGNED NOT NULL COMMENT '审核人ID',
			reviewer_role TINYINT NOT NULL COMMENT '审核人角色',
			action TINYINT NOT NULL COMMENT '操作: 1-通过, 2-拒绝',
			comment VARCHAR(500) DEFAULT NULL COMMENT '审核意见',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			KEY idx_review_records_order_id (order_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}
	for _, s := range sqls {
		if err := db.Exec(s).Error; err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}
	return nil
}

// ---------- Redis ----------

func setupRedisContainer() (string, func()) {
	if err := checkDocker(); err != nil {
		fmt.Printf("[integration] Docker 不可用 (%v)，跳过 Redis 容器启动\n", err)
		return "", nil
	}

	port := fmt.Sprintf("%d", 63800+rand.Intn(100))
	containerName := fmt.Sprintf("reservation-test-redis-%d", time.Now().UnixNano())

	cmd := exec.Command("docker", "run", "-d", "--rm",
		"--name", containerName,
		"-p", fmt.Sprintf("%s:6379", port),
		"redis:7-alpine",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[integration] 启动 Redis 容器失败: %v, output: %s\n", err, string(output))
		return "", nil
	}
	containerID := strings.TrimSpace(string(output))

	cleanup := func() { exec.Command("docker", "stop", containerID).Run() }

	addr := fmt.Sprintf("127.0.0.1:%s", port)
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx := client.Context()
	for i := 0; i < 30; i++ {
		if err := client.Ping(ctx).Err(); err == nil {
			break
		}
		if i == 29 {
			cleanup()
			panic(fmt.Sprintf("Redis 容器启动超时: %v", err))
		}
		time.Sleep(time.Second)
	}
	client.Close()

	return addr, cleanup
}

// ---------- 测试辅助 ----------

// skipIfNoDocker 在没有 Docker 时跳过集成测试。
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if err := checkDocker(); err != nil {
		t.Skip("Docker 不可用，跳过集成测试")
	}
}

func checkDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return err
	}
	return exec.Command("docker", "info").Run()
}

// newDB 连接共享 MySQL 容器，返回 GORM DB 实例。
func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	skipIfNoDocker(t)
	if testMySQLDSN == "" {
		t.Skip("MySQL 容器未启动")
	}
	db, err := gorm.Open(mysql.Open(testMySQLDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接测试数据库失败: %v", err)
	}
	return db
}

// newRepo 返回连接共享 MySQL 的 repository 和清理函数（截断所有表）。
func newRepo(t *testing.T) (reservationdb.Repository, func()) {
	t.Helper()
	db := newDB(t)
	repo := reservationdb.NewRepository(db)
	cleanup := func() {
		db.Exec("DELETE FROM review_records")
		db.Exec("DELETE FROM reservation_slots")
		db.Exec("DELETE FROM reservation_orders")
	}
	return repo, cleanup
}

// newRedisClient 连接共享 Redis 容器，返回客户端和清库清理函数。
func newRedisClient(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	skipIfNoDocker(t)
	if testRedisAddr == "" {
		t.Skip("Redis 容器未启动")
	}
	client := redis.NewClient(&redis.Options{Addr: testRedisAddr})
	ctx := client.Context()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("连接测试 Redis 失败: %v", err)
	}
	cleanup := func() {
		client.FlushDB(ctx)
		client.Close()
	}
	return client, cleanup
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
