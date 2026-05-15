// 微信服务号菜单同步工具
//
// 将本地菜单 JSON 配置文件推送到微信公众平台，支持同步、查询和删除操作。
//
// 用法:
//
//	go run tools/menu/main.go sync   [-c config.yaml] [-m menu.json]
//	go run tools/menu/main.go get    [-c config.yaml]
//	go run tools/menu/main.go delete [-c config.yaml]
//
// 配置文件需包含 wechat.app_id 和 wechat.app_secret 字段，格式与服务端 config_v1.yaml 一致。
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	baseconfig "reservation-sys/pkg/config"

	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/officialaccount"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
)

type wechatAuthConfig struct {
	Wechat struct {
		AppID     string `yaml:"app_id"`
		AppSecret string `yaml:"app_secret"`
	} `yaml:"wechat"`
}

func loadWechatConfig(path string) (appID, appSecret string) {
	cfg := &wechatAuthConfig{}
	baseconfig.LoadYAMLFile(path, cfg)
	if cfg.Wechat.AppID == "" || cfg.Wechat.AppSecret == "" {
		log.Fatal("配置文件中缺少 wechat.app_id 或 wechat.app_secret")
	}
	return cfg.Wechat.AppID, cfg.Wechat.AppSecret
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run tools/menu/main.go <sync|get|delete> [选项]")
		fmt.Println()
		fmt.Println("子命令:")
		fmt.Println("  sync   将本地菜单 JSON 推送到微信公众平台")
		fmt.Println("  get    从微信公众平台获取当前菜单配置")
		fmt.Println("  delete 删除微信公众平台上的所有菜单")
		fmt.Println()
		fmt.Println("选项:")
		fmt.Println("  -c, --config  配置文件路径（默认使用 CONFIG_PATH 环境变量，或 service/gateway/configs/config_v1.yaml）")
		fmt.Println("  -m, --menu    菜单 JSON 文件路径（仅 sync 子命令，默认 tools/menu/menu.json）")
		os.Exit(1)
	}

	cmd := os.Args[1]

	configPath := flagArg(2, "-c", "--config")
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}
	if configPath == "" {
		configPath = "service/gateway/configs/config_v1.yaml"
	}

	menuPath := "tools/menu/menu.json"
	if cmd == "sync" {
		if mp := flagArg(2, "-m", "--menu"); mp != "" {
			menuPath = mp
		}
	}

	appID, appSecret := loadWechatConfig(configPath)

	oa := officialaccount.NewOfficialAccount(&offConfig.Config{
		AppID:     appID,
		AppSecret: appSecret,
		Cache:     cache.NewMemory(),
	})
	m := oa.GetMenu()

	switch cmd {
	case "sync":
		menuJSON, err := os.ReadFile(menuPath)
		if err != nil {
			log.Fatalf("读取菜单文件失败: %v", err)
		}

		var raw json.RawMessage
		if err := json.Unmarshal(menuJSON, &raw); err != nil {
			log.Fatalf("菜单 JSON 格式无效: %v", err)
		}

		if err := m.SetMenuByJSON(string(menuJSON)); err != nil {
			log.Fatalf("同步菜单失败: %v", err)
		}
		fmt.Printf("菜单同步成功（来源: %s）\n", menuPath)

	case "get":
		res, err := m.GetMenu()
		if err != nil {
			log.Fatalf("获取菜单失败: %v", err)
		}
		pretty, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(pretty))

	case "delete":
		if err := m.DeleteMenu(); err != nil {
			log.Fatalf("删除菜单失败: %v", err)
		}
		fmt.Println("菜单已删除")

	default:
		log.Fatalf("未知子命令: %s（支持 sync / get / delete）", cmd)
	}
}

// flagArg 从 os.Args 中提取指定 flag 的值。
// 支持 --flag value 和 -flag value 两种形式。
func flagArg(start int, names ...string) string {
	for i := start; i < len(os.Args)-1; i++ {
		for _, n := range names {
			if os.Args[i] == n {
				return os.Args[i+1]
			}
		}
	}
	return ""
}
