# 预约系统（Reservation System）

基于微信公众号的场地预约系统，支持用户通过微信进行场地预约申请。

## 📖 项目简介

本项目是一个基于微信公众号的场地预约系统，采用微服务架构设计，包含以下核心功能：

- **微信服务（v1）**: 处理微信消息推送、OAuth 认证
- **预约服务（v2）**: 处理用户预约申请、查询、管理
- **管理服务（v3）**: 管理员审核预约（待开发）

## 🛠️ 技术栈

| 类别 | 技术 |
|------|------|
| 后端框架 | Gin |
| 数据库 | MySQL 8.0 + GORM |
| 缓存 | Redis 7 |
| 微信 SDK | silenceper/wechat |
| 容器化 | Docker + Docker Compose |
| 反向代理 | Nginx |
| 认证 | JWT |

## 🚀 快速开始

### 本地开发

```bash
# 1. 启动本地开发环境
./start-local.sh

# 2. 启动 ngrok（新终端）
ngrok http 80

# 3. 更新配置
vim configs/config_v1.local.yaml
# 将 frontend_url 改为 ngrok URL

# 4. 重启 v1 服务
docker-compose -f docker-compose.local.yaml restart v1-service
```

### 云服务器测试

```bash
# 1. 打包测试环境
./deploy-test.sh

# 2. 上传到服务器
scp reservation-sys-test.tar.gz user@your-server-ip:/root/

# 3. 部署
tar -xzf reservation-sys-test.tar.gz
docker load -i reservation-sys.tar
docker-compose -f docker-compose.test.yaml up -d
```

### 生产环境部署

```bash
# 1. 打包生产环境
./deploy.sh

# 2. 部署到服务器
# 详见 doc/部署指南.md
```

## 📁 项目结构

```
reservation_sys_go/
├── cmd/                    # 应用入口
│   ├── api/               # 服务主程序
│   │   ├── v1/           # 微信服务
│   │   ├── v2/           # 预约服务
│   │   └── v3/           # 管理服务（待开发）
│   └── tools/            # 工具程序
│       └── sync_menu.go  # 菜单同步工具
├── configs/               # 配置文件
│   ├── config_v1.*.yaml  # v1 服务配置
│   ├── config_v2.*.yaml  # v2 服务配置
│   ├── config_sync_menu.*.yaml  # 菜单工具配置
│   └── menu.json         # 微信菜单配置
├── deploy/                # 部署配置
│   ├── mysql/            # 数据库初始化
│   └── nginx/            # Nginx 配置
├── internal/              # 内部代码
│   ├── auth/             # 认证模块
│   ├── reservation/      # 预约模块
│   ├── notification/     # 消息通知模块
│   └── platform/         # 基础设施（DB、Redis）
├── doc/                   # 文档
├── docker-compose.*.yaml  # Docker Compose 配置
├── Dockerfile             # Docker 镜像构建
├── deploy.sh              # 生产环境部署脚本
├── deploy-test.sh         # 测试环境部署脚本
└── start-local.sh         # 本地开发启动脚本
```

## 📚 文档导航

| 文档 | 说明 |
|------|------|
| [部署指南](doc/部署指南.md) | 本地、测试、生产环境部署 |
| [开发指南](doc/开发指南.md) | API 文档、模块说明、测试 |
| [配置说明](doc/配置说明.md) | 配置文件详解 |
| [工具使用](doc/工具使用.md) | 菜单同步工具等工具说明 |
| [数据库说明](doc/数据库说明.md) | 数据库结构和常用命令 |

## 🌐 服务架构

```
微信服务器
    ↓
   Nginx (:80)
    ↓
    ├── v1 Service (:8080) → 微信消息、OAuth
    ├── v2 Service (:8081) → 预约业务
    └── v3 Service (:8082) → 管理审核（待开发）
    ↓
MySQL (:3306) + Redis (:6379)
```

## ⚙️ 环境要求

- Docker 20.10+
- Docker Compose 2.0+
- Go 1.24+（本地开发）
- ngrok（本地测试微信）

## 🔧 常用命令

```bash
# 本地开发
./start-local.sh                          # 启动本地环境
docker-compose -f docker-compose.local.yaml logs -f  # 查看日志

# 测试环境
./deploy-test.sh                          # 打包测试环境
docker-compose -f docker-compose.test.yaml ps        # 查看服务状态

# 生产环境
./deploy.sh                               # 打包生产环境
docker-compose -f docker-compose.prod.yaml up -d     # 启动服务

# 菜单同步
go run cmd/tools/sync_menu.go -config configs/config_sync_menu.local.yaml
```

## 📞 支持

如有问题，请查看：
- [故障排查](doc/部署指南.md#故障排查)
- [常见问题](doc/开发指南.md#常见问题)

## 📄 许可证

MIT License
