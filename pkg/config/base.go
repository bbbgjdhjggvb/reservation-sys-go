// Package config 提供通用的配置类型定义
// 各模块可独立使用或组合这些配置
package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Port             string   `yaml:"port"`
	GRPCPort         string   `yaml:"grpc_port"`
	Mode             string   `yaml:"mode"`
	CORSAllowOrigins []string `yaml:"cors_allow_origins"`
}

// MySQLConfig MySQL 数据库配置
type MySQLConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	DBName       string `yaml:"dbname"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

// RedisConfig Redis 缓存配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// JwtConfig JWT 配置
type JwtConfig struct {
	Secret     string `yaml:"secret"`
	ExpireTime int    `yaml:"expire_time"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	HandlerName string `yaml:"handler_name"`
	Dimension   string `yaml:"dimension"`
	WindowSec   int    `yaml:"window_sec"`
	MaxRequests int    `yaml:"max_requests"`
	FailOpen    bool   `yaml:"fail_open"`
}

// LoadYAMLFile 通用 YAML 文件加载函数
func LoadYAMLFile(path string, cfg any) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("[error][pkg/config/base] 无法读取配置文件 [%s]: %w", path, err)
	}

	if err := yaml.Unmarshal(file, cfg); err != nil {
		return fmt.Errorf("[error][pkg/config/base] 解析配置文件失败: %w", err)
	}

	log.Printf("[info][pkg/config/base] 配置文件加载成功: %s", path)
	return nil
}

// GetEnv 获取环境变量，若不存在则返回默认值
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
