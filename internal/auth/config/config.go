// Package config 提供 auth 模块的配置管理
package config

import (
	baseconfig "reservation-sys/internal/config"
)

// Config auth 模块配置
type Config struct {
	Server baseconfig.ServerConfig `yaml:"server"`
	MySQL  baseconfig.MySQLConfig  `yaml:"mysql"`
	Redis  baseconfig.RedisConfig  `yaml:"redis"`
	JWT    baseconfig.JwtConfig    `yaml:"jwt"`
	Wechat WechatConfig            `yaml:"wechat"`
}

// WechatConfig 微信相关配置
type WechatConfig struct {
	AppID          string `yaml:"app_id"`
	AppSecret      string `yaml:"app_secret"`
	Token          string `yaml:"token"`
	MenuConfigPath string `yaml:"menu_config_path"` // 菜单配置文件路径
	FrontendURL    string `yaml:"frontend_url"`     // OAuth 回调后重定向的前端URL
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
		panic("auth config not initialized")
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

// GetWechat 获取微信配置
func GetWechat() *WechatConfig {
	return &cfg.Wechat
}
