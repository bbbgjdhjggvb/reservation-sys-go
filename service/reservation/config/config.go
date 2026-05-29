// Package config 提供 reservation 模块的配置管理
package config

import (
	"fmt"
	"log"

	baseconfig "reservation-sys/pkg/config"
)

// Config reservation 模块配置
type Config struct {
	Server    baseconfig.ServerConfig      `yaml:"server"`
	MySQL     baseconfig.MySQLConfig       `yaml:"mysql"`
	Redis     baseconfig.RedisConfig       `yaml:"redis"`
	RateLimit []baseconfig.RateLimitConfig `yaml:"rate_limit"`
	JWT       baseconfig.JwtConfig         `yaml:"jwt"`
}

var cfg *Config

// Load 加载配置文件，必须在调用任何 GetXxx 函数之前执行
func Load(path string) {
	cfg = &Config{}
	if err := baseconfig.LoadYAMLFile(path, cfg); err != nil {
		log.Fatalf("[reservation/config] 加载配置失败: %v", err)
	}
}

// requireLoad 在访问配置前校验 Load 已被调用
func requireLoad() {
	if cfg == nil {
		panic(fmt.Errorf(
			"[reservation/config] 配置尚未加载，请先调用 Load() 再访问配置",
		))
	}
}

// GetServer 获取服务器配置
func GetServer() *baseconfig.ServerConfig {
	requireLoad()
	return &cfg.Server
}

// GetMySQL 获取 MySQL 配置
func GetMySQL() *baseconfig.MySQLConfig {
	requireLoad()
	return &cfg.MySQL
}

// GetRedis 获取 Redis 配置
func GetRedis() *baseconfig.RedisConfig {
	requireLoad()
	return &cfg.Redis
}

// GetJWT 获取 JWT 配置
func GetJWT() *baseconfig.JwtConfig {
	requireLoad()
	return &cfg.JWT
}

// GetRateLimits 获取 rate_limit 配置列表
func GetRateLimits() []baseconfig.RateLimitConfig {
	requireLoad()
	return cfg.RateLimit
}
