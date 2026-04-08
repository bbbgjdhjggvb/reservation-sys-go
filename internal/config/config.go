// Package config 提供全局配置管理（已废弃，建议使用各模块独立的配置）
//
// Deprecated: 请使用各模块独立的配置包：
//   - auth 模块: reservation-sys/internal/auth/config
//   - reservation 模块: reservation-sys/internal/reservation/config
//
// 保留此包是为了向后兼容，新代码请勿使用
package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// WechatConfig 微信相关配置（仅用于全局配置）
type WechatConfig struct {
	AppID          string `yaml:"app_id"`
	AppSecret      string `yaml:"app_secret"`
	Token          string `yaml:"token"`
	MenuConfigPath string `yaml:"menu_config_path"` // 菜单配置文件路径
	FrontendURL    string `yaml:"frontend_url"`     // 前端预约页面URL
}

// AppConfig 总配置结构体
// Deprecated: 请使用各模块独立的 Config
type AppConfig struct {
	Server ServerConfig `yaml:"server"`
	MySQL  MySQLConfig  `yaml:"mysql"`
	Redis  RedisConfig  `yaml:"redis"`
	Wechat WechatConfig `yaml:"wechat"`
	Jwt    JwtConfig    `yaml:"jwt"`
}

var cfg *AppConfig

// LoadConfig 从指定路径加载 YAML 配置文件
// Deprecated: 请使用各模块的 config.Load()
func LoadConfig(path string) {
	config := &AppConfig{}

	// 读取文件内容
	file, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("无法读取配置文件 [%s]: %v", path, err)
	}

	// 解析 YAML
	err = yaml.Unmarshal(file, config)
	if err != nil {
		log.Fatalf("解析配置文件失败: %v", err)
	}

	cfg = config
	log.Println("配置文件加载成功")
}

// GetConfig 获取全局配置实例
// Deprecated: 请使用各模块的 config.Get()
func GetConfig() *AppConfig {
	if cfg == nil {
		log.Fatalf("[error][config]: cfg don't init")
		return nil
	}

	return cfg
}
