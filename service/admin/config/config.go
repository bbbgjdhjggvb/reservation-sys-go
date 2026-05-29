// Package config 提供 admin 服务的配置管理
package config

import (
	"log"

	baseconfig "reservation-sys/pkg/config"
)

// Config admin 服务配置
type Config struct {
	Server baseconfig.ServerConfig `yaml:"server"`
	MySQL  baseconfig.MySQLConfig  `yaml:"mysql"`
	JWT    baseconfig.JwtConfig    `yaml:"jwt"`
	GRPC   GRPCConfig              `yaml:"grpc"`
}

// GRPCConfig gRPC 客户端配置（仅连接 Gateway）
type GRPCConfig struct {
	GatewayAddr string `yaml:"gateway_addr"` // Gateway gRPC 地址（通知 + 账号验证）
}

var cfg *Config

// Load 加载配置文件
func Load(path string) {
	cfg = &Config{}
	if err := baseconfig.LoadYAMLFile(path, cfg); err != nil {
		log.Fatalf("[admin/config] 加载配置失败: %v", err)
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

// GetJWT 获取 JWT 配置
func GetJWT() *baseconfig.JwtConfig {
	return &cfg.JWT
}
