// 独立命令行工具：同步微信公众号菜单
//
// 用法:
//
//	# 方式1: 使用默认配置
//	go run cmd/tools/sync_menu.go
//
//	# 方式2: 指定配置文件
//	go run cmd/tools/sync_menu.go -config configs/config_sync_menu.local.yaml
//
//	# 方式3: 使用环境变量
//	CONFIG_PATH=configs/config_sync_menu.test.yaml go run cmd/tools/sync_menu.go
//
// 说明:
//
//	读取 menu.json 并同步到微信服务号，适用于测试阶段反复调整菜单
//	使用内存缓存存储 access_token，无需 Redis 依赖
//
// 配置文件优先级:
//  1. 命令行参数 -config
//  2. 环境变量 CONFIG_PATH
//  3. configs/config_sync_menu.yaml（默认）
//  4. configs/config_v1.yaml（降级）
//
// 菜单创建规范:
//  1. 最多 3 个一级菜单，每个一级菜单最多包含 5 个二级菜单
//  2. 一级菜单最多 4 个汉字，二级菜单最多 8 个汉字
//  3. 测试时，重新关注服务号可以刷新菜单
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	authconfig "reservation-sys/internal/auth/config"

	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/officialaccount"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
)

func main() {
	// 1. 加载配置
	configPath := getConfigPath()
	authconfig.Load(configPath)
	cfg := authconfig.Get()

	menuPath := cfg.Wechat.MenuConfigPath
	if menuPath == "" {
		log.Fatal("[error] 配置文件中缺少 menu_config_path 字段")
	}

	// 2. 初始化微信 SDK（使用内存缓存，无需 Redis）
	oa := initOfficialAccount(cfg)

	// 3. 同步菜单
	if err := createMenuFromJSON(oa, menuPath); err != nil {
		fmt.Fprintf(os.Stderr, "[error] 菜单同步失败: %v\n", err)
		os.Exit(1)
	}

	log.Printf("[info] 菜单同步成功: %s\n", menuPath)
}

// getConfigPath 获取配置文件路径
// 优先级: 命令行参数 > 环境变量 > 默认值
func getConfigPath() string {
	// 1. 检查命令行参数 -config
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "-config" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}

	// 2. 检查环境变量
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		return configPath
	}

	// 3. 默认配置文件（优先使用独立配置）
	if _, err := os.Stat("configs/config_sync_menu.yaml"); err == nil {
		return "configs/config_sync_menu.yaml"
	}

	// 4. 降级使用 v1 配置
	return "configs/config_v1.yaml"
}

func initOfficialAccount(cfg *authconfig.Config) *officialaccount.OfficialAccount {
	wc := wechat.NewWechat()
	// 使用内存缓存，无需 Redis 依赖
	memoryCache := cache.NewMemory()
	return wc.GetOfficialAccount(&offConfig.Config{
		AppID:     cfg.Wechat.AppID,
		AppSecret: cfg.Wechat.AppSecret,
		Token:     cfg.Wechat.Token,
		Cache:     memoryCache,
	})
}

// createMenuFromJSON 从 JSON 文件创建自定义菜单
func createMenuFromJSON(oa *officialaccount.OfficialAccount, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取菜单文件失败: %w", err)
	}

	// 校验 JSON 格式是否合法（至少包含 button 字段）
	var menu struct {
		Button []json.RawMessage `json:"button"`
	}
	if err := json.Unmarshal(data, &menu); err != nil {
		return fmt.Errorf("菜单 JSON 格式错误: %w", err)
	}

	menuClient := oa.GetMenu()
	if err := menuClient.SetMenuByJSON(string(data)); err != nil {
		return fmt.Errorf("微信 API 设置菜单失败: %w", err)
	}

	return nil
}
