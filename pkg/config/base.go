// Package config 提供通用的配置类型定义
// 各模块可独立使用或组合这些配置
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ServerConfig 服务器配置
type ServerConfig struct {
	Port             string   `yaml:"port"`
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
	Secret           string `yaml:"secret"`
	UserExpireHours  int    `yaml:"user_expire_hours"`  // 用户 JWT 过期小时数
	AdminExpireHours int    `yaml:"admin_expire_hours"` // 管理员 JWT 过期小时数
}

// WechatConfig 微信相关配置
type WechatConfig struct {
	AppID           string            `yaml:"app_id"`
	AppSecret       string            `yaml:"app_secret"`
	Token           string            `yaml:"token"`
	TemplateID      string            `yaml:"template_id"`      // 审核通知模板消息ID
	MenuConfigPath  string            `yaml:"menu_config_path"` // 菜单配置文件路径
	DefaultRedirect string            `yaml:"default_redirect"` // OAuth 回调默认重定向地址
	RedirectURLs    map[string]string `yaml:"redirect_urls"`    // state -> 重定向URL 映射表
}

// AppConfig 单体应用统一配置
type AppConfig struct {
	Server    ServerConfig      `yaml:"server"`
	MySQL     MySQLConfig       `yaml:"mysql"`
	Redis     RedisConfig       `yaml:"redis"`
	JWT       JwtConfig         `yaml:"jwt"`
	Wechat    WechatConfig      `yaml:"wechat"`
	RateLimit []RateLimitConfig `yaml:"ratelimit"`
}

// MustLoad 加载配置文件，若文件不存在则生成默认配置文件后加载。
func MustLoad(path string) *AppConfig {
	// 配置文件不存在时自动生成默认配置
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("[pkg/config] 配置文件 [%s] 不存在，正在生成默认配置...", path)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("[pkg/config] 创建配置目录 [%s] 失败: %v", dir, err)
		}
		cfg := DefaultConfig()
		data, err := yaml.Marshal(cfg)
		if err != nil {
			log.Fatalf("[pkg/config] 序列化默认配置失败: %v", err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			log.Fatalf("[pkg/config] 写入默认配置文件 [%s] 失败: %v", path, err)
		}
		log.Printf("[pkg/config] 默认配置文件已生成: %s，请根据实际环境修改配置", path)
	}

	cfg := &AppConfig{}
	if err := LoadYAMLFile(path, cfg); err != nil {
		log.Fatalf("[pkg/config] 加载配置失败: %v", err)
	}
	return cfg
}

// DefaultConfig 返回一份默认配置，供首次运行时自动生成配置文件使用。
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Port:             ":8080",
			Mode:             "debug",
			CORSAllowOrigins: []string{"http://localhost:5173", "http://localhost:5174"},
		},
		MySQL: MySQLConfig{
			Host:         "127.0.0.1",
			Port:         3306,
			User:         "res_user",
			Password:     "xSIn34sU7qQl31kQ3TVfcQ==",
			DBName:       "reservation_sys",
			MaxIdleConns: 10,
			MaxOpenConns: 50,
		},
		Redis: RedisConfig{
			Host:     "127.0.0.1",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		JWT: JwtConfig{
			Secret:           "Y6Xoo746BoVCWFyFUVSqqboCfqo7QkC8A5CN7F9sOm0=",
			UserExpireHours:  24,
			AdminExpireHours: 24,
		},
		Wechat: WechatConfig{
			AppID:           "wx84d6833105361902",
			AppSecret:       "d37c9085aa2d446fcd60e8c4236fee3d",
			Token:           "1yYfrKx0DNgyrglRsvMs9KQKHlp0ON3jhnjJUDI+mmg=",
			TemplateID:      "_vyv6J7c_YNBSnBJ6Este2P8k04s3gmmDL-Wr9thNJo",
			MenuConfigPath:  "tools/menu/menu.json",
			DefaultRedirect: "http://localhost:5173/",
			RedirectURLs: map[string]string{
				"reserve":  "http://localhost:5173/",
				"myorders": "http://localhost:5173/myorders",
			},
		},
		RateLimit: []RateLimitConfig{
			{HandlerName: "submit", Dimension: "user", WindowSec: 60, MaxRequests: 3, FailOpen: true},
			{HandlerName: "submit", Dimension: "ip", WindowSec: 60, MaxRequests: 10, FailOpen: true},
			{HandlerName: "cancel", Dimension: "user", WindowSec: 60, MaxRequests: 5, FailOpen: true},
		},
	}
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

// ConfigPath 解析配置文件路径，优先级: --config 标志 > CONFIG_PATH 环境变量 > 默认 config.yaml
func ConfigPath() string {
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--config" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	return "config.yaml"
}
