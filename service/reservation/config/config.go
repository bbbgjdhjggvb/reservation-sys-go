// Package config 提供 reservation 模块的配置管理
package config

import (
	baseconfig "reservation-sys/pkg/config"
)

// Config reservation 模块配置
type Config struct {
	Server baseconfig.ServerConfig `yaml:"server"`
	MySQL  baseconfig.MySQLConfig  `yaml:"mysql"`
	Redis  baseconfig.RedisConfig  `yaml:"redis"`
	JWT    baseconfig.JwtConfig    `yaml:"jwt"`
}

var cfg *Config

// Load 加载配置文件
func Load(path string) {
	cfg = &Config{}
	baseconfig.LoadYAMLFile(path, cfg)
}

// Get 获取配置实例
func Get() *Config {
	if cfg == nil {
		panic("reservation config not initialized")
	}
	return cfg
}

// GetServer 获取服务器配置
func GetServer() *baseconfig.ServerConfig {
	return &cfg.Server
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
