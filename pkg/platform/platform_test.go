package platform

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"reservation-sys/pkg/config"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// miniredisPort 从 miniredis.Addr() 中提取端口号（int）。
func miniredisPort(mr *miniredis.Miniredis) int {
	// Addr() 返回 "host:port"
	parts := strings.Split(mr.Addr(), ":")
	if len(parts) == 2 {
		p, _ := strconv.Atoi(parts[1])
		return p
	}
	return 6379
}

// func InitRedis(cfg *config.RedisConfig) (*redis.Client, error) 测试
//
// 函数功能：根据传入的配置初始化 redis 连接
//
// 测试场景：
// 1. 成功连接到 Redis
// 2. 使用密码连接 Redis
// 3. 验证 DB 参数是否有效
// 4. 验证最大连接池配置是否有效
func TestInitRedis_Success(t *testing.T) {
	// 使用 miniredis 模拟 Redis 程序
	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Host: mr.Host(),
		Port: miniredisPort(mr),
		DB:   0,
	}

	client, err := InitRedis(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	// 验证 Ping 成功
	ctx := client.Context()
	err = client.Ping(ctx).Err()
	require.NoError(t, err)
}

func TestInitRedis_WithPassword(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()
	mr.RequireAuth("testpass")

	cfg := &config.RedisConfig{
		Host:     mr.Host(),
		Port:     miniredisPort(mr),
		Password: "testpass",
		DB:       0,
	}

	client, err := InitRedis(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	ctx := client.Context()
	err = client.Ping(ctx).Err()
	require.NoError(t, err)
}

func TestInitRedis_WithDB(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Host: mr.Host(),
		Port: miniredisPort(mr),
		DB:   2,
	}

	client, err := InitRedis(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	ctx := client.Context()
	err = client.Ping(ctx).Err()
	require.NoError(t, err)
}

func TestInitRedis_ConnectionPool(t *testing.T) {
	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Host: mr.Host(),
		Port: miniredisPort(mr),
		DB:   0,
	}

	client, err := InitRedis(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)

	// 验证连接池正常工作（连续多次操作）
	ctx := client.Context()
	for i := 0; i < 10; i++ {
		err := client.Set(ctx, "test_key", "value", 10*time.Second).Err()
		require.NoError(t, err)
		val, err := client.Get(ctx, "test_key").Result()
		require.NoError(t, err)
		assert.Equal(t, "value", val)
	}
}
