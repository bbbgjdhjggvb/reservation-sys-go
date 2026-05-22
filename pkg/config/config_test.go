package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/* 测试 func GetEnv(key, defaultValue string) string
 * 函数功能：通过 key 获取环境变量，如果不存在环境变量返回 defaultValue
 * 测试场景：
 * 1. 环境变量存在时返回其值
 * 2. 环境变量不存在时返回默认值
 * 3. 环境变量为空字符串时返回默认值
 */
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "环境变量存在时返回其值",
			key:          "TEST_CONFIG_ENV_EXISTS",
			defaultValue: "default",
			envValue:     "custom_value",
			setEnv:       true,
			expected:     "custom_value",
		},
		{
			name:         "环境变量不存在时返回默认值",
			key:          "TEST_CONFIG_ENV_MISSING",
			defaultValue: "fallback",
			envValue:     "",
			setEnv:       false,
			expected:     "fallback",
		},
		{
			name:         "环境变量为空字符串时返回默认值_空等同于不存在",
			key:          "TEST_CONFIG_ENV_EMPTY",
			defaultValue: "default_val",
			envValue:     "",
			setEnv:       true,
			expected:     "default_val",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 清理环境
			os.Unsetenv(tt.key)

			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			// 调用 GetEnv 函数
			got := GetEnv(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, got)
		})
	}
}

/* func LoadYAMLFile(path string, cfg any) 函数测试
 * 函数功能：从指定路径中加载 YAML 文件配置到任意结构体中
 * 测试场景：
 * 1. 正常解析 YAML 文件
 * 2. 空 YAML 文件，应该返回结构体的零值
 * 3. 部分字段的 TAML 文件，未定义的字段应该为零值
 * 4. 路径错误，应该返回 error
 * 5. 错误的 YAML 格式，应该返回 error
 */

type testConfig struct {
	Server   testServerConfig   `yaml:"server"`
	Database testDatabaseConfig `yaml:"database"`
}

type testServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type testDatabaseConfig struct {
	DSN      string `yaml:"dsn"`
	MaxConns int    `yaml:"max_conns"`
}

func TestLoadYAMLFile(t *testing.T) {
	t.Run("正常解析YAML文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "test_config.yaml")
		content := `
server:
  host: localhost
  port: 8080
database:
  dsn: "user:pass@tcp(127.0.0.1:3306)/testdb"
  max_conns: 100
`
		err := os.WriteFile(yamlPath, []byte(content), 0644)
		require.NoError(t, err)

		var cfg testConfig
		err = LoadYAMLFile(yamlPath, &cfg)
		require.NoError(t, err)

		assert.Equal(t, "localhost", cfg.Server.Host)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, "user:pass@tcp(127.0.0.1:3306)/testdb", cfg.Database.DSN)
		assert.Equal(t, 100, cfg.Database.MaxConns)
	})

	t.Run("空YAML文件_使用零值", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "empty_config.yaml")
		err := os.WriteFile(yamlPath, []byte(""), 0644)
		require.NoError(t, err)

		var cfg testConfig
		err = LoadYAMLFile(yamlPath, &cfg)
		require.NoError(t, err)

		assert.Equal(t, "", cfg.Server.Host)
		assert.Equal(t, 0, cfg.Server.Port)
	})

	t.Run("部分字段的YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "partial_config.yaml")
		content := `
server:
  port: 9090
`
		err := os.WriteFile(yamlPath, []byte(content), 0644)
		require.NoError(t, err)

		var cfg testConfig
		err = LoadYAMLFile(yamlPath, &cfg)
		require.NoError(t, err)

		assert.Equal(t, 9090, cfg.Server.Port)
		assert.Equal(t, "", cfg.Server.Host) // 未定义的字段应为零值
	})

	t.Run("错误路径_返回错误", func(t *testing.T) {
		err := LoadYAMLFile("/nonexistent/path/config.yaml", &testConfig{})
		require.Error(t, err)
	})

	t.Run("文件格式错误_返回错误", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(yamlPath, []byte("{{{invalid yaml}}}"), 0644)
		require.NoError(t, err)

		err = LoadYAMLFile(yamlPath, &testConfig{})
		require.Error(t, err)
	})
}
