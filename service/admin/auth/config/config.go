// Package config 提供 admin 模块的配置管理
package config

import (
	"log"

	baseconfig "reservation-sys/pkg/config"
)

// Config admin 模块配置
type Config struct {
	Server baseconfig.ServerConfig `yaml:"server"`
	MySQL  baseconfig.MySQLConfig  `yaml:"mysql"`
	Redis  baseconfig.RedisConfig  `yaml:"redis"`
	JWT    baseconfig.JwtConfig    `yaml:"jwt"`
	GRPC   GRPCConfig              `yaml:"grpc"`
}

// GRPCConfig gRPC 客户端配置
type GRPCConfig struct {
	NotificationAddr string `yaml:"notification_addr"` // v1 通知服务地址
	ReservationAddr  string `yaml:"reservation_addr"`  // v2 预约服务地址
}

var cfg *Config

// Load 加载配置文件
func Load(path string) {
	cfg = &Config{}
	if err := baseconfig.LoadYAMLFile(path, cfg); err != nil {
		log.Fatalf("[admin/auth/config] 加载配置失败: %v", err)
	}
}

// Get 获取配置实例
func Get() *Config {
	if cfg == nil {
		panic("admin config not initialized")
	}
	return cfg
}

// GetMySQL 获取 MySQL 配置
func GetMySQL() *baseconfig.MySQLConfig {
	return &cfg.MySQL
}

// GetRedis 获取 Redis 配置
func GetRedis() *baseconfig.RedisConfig {
	return &cfg.Redis
}

// GetJWT 获取 JWT 配置
func GetJWT() *baseconfig.JwtConfig {
	return &cfg.JWT
}
