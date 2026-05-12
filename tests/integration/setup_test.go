// Package integration 提供数据库集成测试基础设施。
// 使用 Docker MySQL 容器进行真实数据库测试。
// TestMain 管理单个 MySQL 容器的生命周期，所有测试共享该容器。
package integration

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	reservationdb "reservation-sys/pkg/reservationdb"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	testDBUser     = "res_user"
	testDBPassword = "12345678"
	testDBName     = "home_res"
)

var (
	testDBPort string
	testDSN    string
	cleanupFn  func()
)

func TestMain(m *testing.M) {
	// 随机端口避免冲突
	testDBPort = fmt.Sprintf("%d", 33070+rand.Intn(100))
	testDSN, cleanupFn = setupMySQLContainer()
	code := m.Run()
	cleanupFn()
	os.Exit(code)
}

// setupMySQLContainer 启动 Docker MySQL 容器，返回 DSN 和清理函数。
func setupMySQLContainer() (string, func()) {
	containerName := fmt.Sprintf("reservation-test-mysql-%d", time.Now().UnixNano())

	cmd := exec.Command("docker", "run", "-d", "--rm",
		"--name", containerName,
		"-e", "MYSQL_ROOT_PASSWORD=root123",
		"-p", testDBPort+":3306",
		"mysql:8.0",
		"--default-authentication-plugin=mysql_native_password",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("启动 MySQL 容器失败: %v, output: %s", err, string(output)))
	}
	containerID := strings.TrimSpace(string(output))

	cleanup := func() {
		exec.Command("docker", "stop", containerID).Run()
	}

	// 等待 MySQL 就绪
	rootDSN := fmt.Sprintf("root:root123@tcp(127.0.0.1:%s)/?charset=utf8mb4&parseTime=True&loc=Local", testDBPort)
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

	// 创建测试数据库和用户
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

	// 使用测试用户连接并建表
	userDSN := fmt.Sprintf("%s:%s@tcp(127.0.0.1:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		testDBUser, testDBPassword, testDBPort, testDBName)
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

// runInitSQL 创建测试需要的表结构。
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

// setupRepo 连接共享的 MySQL 容器，返回 repository 和清理函数（截断所有表）。
func setupRepo(t *testing.T) (reservationdb.Repository, func()) {
	t.Helper()
	skipIfNoDocker(t)

	db, err := gorm.Open(mysql.Open(testDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接测试数据库失败: %v", err)
	}

	repo := reservationdb.NewRepository(db)

	// 每个测试后清空数据，保留表结构
	cleanup := func() {
		db.Exec("DELETE FROM review_records")
		db.Exec("DELETE FROM reservation_slots")
		db.Exec("DELETE FROM reservation_orders")
	}

	return repo, cleanup
}

// skipIfNoDocker 在没有 Docker 时跳过集成测试。
func skipIfNoDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker 不可用，跳过集成测试")
	}
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skip("Docker 不可用，跳过集成测试")
	}
}

// 确保 os 被使用
var _ = os.Getenv
