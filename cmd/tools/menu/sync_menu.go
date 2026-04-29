// 独立命令行工具：同步微信公众号菜单
//
// 用法:
//
//	go run cmd/tools/sync_menu.go
//
// 说明:
//
//	读取 menu.json 并同步到微信服务号，适用于测试阶段反复调整菜单
//	使用内存缓存存储 access_token，无需 Redis 依赖
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

	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/officialaccount"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
)

// 微信公众号配置（硬编码，无需外部配置文件）
var (
	appID     = "wx84d6833105361902"
	appSecret = "d37c9085aa2d446fcd60e8c4236fee3d"
	token     = "mytesttoken123"
	menuPath  = "configs/menu.json"
)

func main() {
	// 1. 初始化微信 SDK（使用内存缓存，无需 Redis）
	oa := initOfficialAccount(appID, appSecret, token)

	// 2. 同步菜单
	if err := createMenuFromJSON(oa, menuPath); err != nil {
		fmt.Fprintf(os.Stderr, "[error] 菜单同步失败: %v\n", err)
		os.Exit(1)
	}

	log.Printf("[info] 菜单同步成功: %s\n", menuPath)
}

func initOfficialAccount(appID, appSecret, token string) *officialaccount.OfficialAccount {
	wc := wechat.NewWechat()
	memoryCache := cache.NewMemory()
	return wc.GetOfficialAccount(&offConfig.Config{
		AppID:     appID,
		AppSecret: appSecret,
		Token:     token,
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
