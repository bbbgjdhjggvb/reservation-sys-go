// Package config 提供 auth 模块的配置管理
package config

import (
	"log"

	baseconfig "reservation-sys/pkg/config"
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
	AppID           string            `yaml:"app_id"`
	AppSecret       string            `yaml:"app_secret"`
	Token           string            `yaml:"token"`
	TemplateID      string            `yaml:"template_id"`      // 审核通知模板消息ID
	MenuConfigPath  string            `yaml:"menu_config_path"` // 菜单配置文件路径
	DefaultRedirect string            `yaml:"default_redirect"` // OAuth 回调默认重定向地址（state 未匹配时使用）
	RedirectURLs    map[string]string `yaml:"redirect_urls"`    // state -> 重定向URL 映射表
}

var cfg *Config

// Load 加载配置文件
func Load(path string) {
	cfg = &Config{}
	if err := baseconfig.LoadYAMLFile(path, cfg); err != nil {
		log.Fatalf("[gateway/auth/config] 加载配置失败: %v", err)
	}
}

// Get 获取配置实例
func Get() *Config {
	if cfg == nil {
		panic("auth config not initialized")
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
