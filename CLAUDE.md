# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此仓库中工作时提供指导。

## 常用命令

```sh
# 启动基础设施（MySQL + Redis 容器）
bash scripts/env.sh up

# 停止所有容器并清理
bash scripts/env.sh down

# 生成用于本地测试的用户 JWT（openid 默认为 test_openid_local_001）
bash scripts/env.sh token

# 后台启动 reservation 服务（日志输出到 .test-data/reservation.log）
bash scripts/env.sh serve

# 停止 reservation 服务
bash scripts/env.sh stop

# 运行完整的 API 集成测试（需先执行 env up + serve + token）
bash scripts/test-api.sh

# 从 proto 定义生成 gRPC Go 代码
bash scripts/gen_proto.sh

# 构建所有 Docker 镜像并打包用于部署
bash scripts/build.sh

# 直接运行单个服务（用于调试）
CONFIG_PATH=service/reservation/configs/config_v2.local.yaml go run service/reservation/cmd/main.go
CONFIG_PATH=service/gateway/configs/config_v1.local.yaml go run service/gateway/cmd/main.go
CONFIG_PATH=service/admin/configs/config_v3.yaml go run service/admin/cmd/main.go

# 启动 Swagger 文档服务（端口 8083）
go run service/swagger/cmd/main.go

# 从注解重新生成 Swagger 文档
/home/fufu/go/bin/swag init -g service/swagger/cmd/main.go -o docs --parseDependency --parseInternal

# 运行指定包的 Go 测试
go test ./service/reservation/... -v
go test ./service/gateway/auth/... -v

# 运行单个测试用例
go test ./service/reservation/... -run TestSubmitHandler -v

# 重新生成 mock 文件（如果存在 go:generate 指令）
go generate ./...
```

## 架构

三个 Go 微服务位于 `service/` 目录下，共享代码位于 `pkg/`。HTTP 框架：Gin。ORM：GORM。服务间通信：gRPC。

### 服务概览

| 服务 | 目录 | 端口 | 数据库 | 职责 |
|---------|-----|------|-----|---------|
| Gateway (v1) | `service/gateway/` | config_v1 | `home_xy` | 微信 OAuth 登录、用户/管理员认证、通过 gRPC 推送模板消息 |
| Reservation (v2) | `service/reservation/` | config_v2 | `home_res` | 用户端：提交/取消/查询预约，提供前端 HTML 页面 |
| Admin (v3) | `service/admin/` | config_v3 | `home_res` | 两级审核后台、门锁密码管理、通过 gRPC 调用 Gateway, 提供前端 HTML 页面|
| Swagger | `service/swagger/` | :8083 | — | 聚合全部 API 文档，通过 swagger UI 展示 |

### 共享包 (`pkg/`)

- `pkg/config/` — 通用配置结构体（Server、MySQL、Redis、JWT）+ YAML 加载器
- `pkg/platform/` — `InitDB()`（GORM/MySQL）和 `InitRedis()`（go-redis）
- `pkg/reservationdb/` — 共享的 Repository 接口 + GORM 实现，操作 `home_res` 数据库（订单、时段、审核记录）。v2 和 v3 均使用此包。
- `pkg/jwt/` — 独立的用户 JWT 和管理员 JWT（HMAC-SHA256，`sync.Once` 单例）。使用前需调用 `InitUserJWT` / `InitAdminJWT`。
- `pkg/grpc/` — gRPC 客户端连接辅助
- `pkg/constants/` — 共享的状态/角色常量

### 请求流程

```
微信 → Gateway (/callback 回调) → 用 code 换取 openid → 签发用户 JWT → 重定向到前端
前端 → Reservation API (Bearer token) → AuthMiddleware → handler → repository → MySQL
管理员后台 → Admin API (Bearer token) → AdminAuthMiddleware → RoleMiddleware → handler → repository + 通过 gRPC 调用 Gateway
```

### 配置加载模式

每个服务的 `cmd/main.go` 接受 `--config` 标志或读取 `CONFIG_PATH` 环境变量。配置结构体嵌入 `pkg/config` 中的共享类型，并添加服务特有的字段。

### 数据库访问模式

Repository 接口定义在 `pkg/reservationdb/repository.go`，使用 GORM 实现。通过 `reservationdb.GetRepository()` 单例访问（如果未初始化会 panic）。状态转换使用乐观锁（`WHERE status = fromStatus`）。提交预约使用 `SELECT ... FOR UPDATE` 防止重复预订。

### 测试

使用 `gomock`（github.com/golang/mock）生成 mock，使用 `testify/assert` 进行断言。Handler 测试使用 `httptest.NewRecorder()` 配合 Gin 引擎。Mock 文件命名为 `mock_*.go`。

## 项目约定

- Go module 名称：`reservation-sys`
- 服务入口：`service/<名称>/cmd/main.go`
- 配置文件：`service/<名称>/configs/config_v<N>.yaml`（已提交）和 `config_v<N>.local.yaml`（gitignore，本地覆盖）
- 前端：静态 HTML 位于 `service/<名称>/frontend/`，由 Gin 通过 `r.LoadHTMLGlob` 提供
- 前端设计必须有深圳大学的风格，采用荔枝红配色
- Proto 定义：`service/gateway/api/proto/`，生成的代码在 `service/gateway/api/gen/` 并复制到 `service/admin/api/gen/`
- `.test-data/` 存放本地测试状态（token、预约 ID、日志）—— 已加入 gitignore
- 本地开发使用 `docker-compose.local.yaml` 启动 MySQL + Redis；完整环境包括 gateway + reservation + admin + nginx
- MySQL 初始化与授权解耦：数据库创建、表创建、权限授予统一在 `deploy/mysql/init.sql` 中管理；`docker-compose*.yaml` 只设置 `MYSQL_ROOT_PASSWORD`，`MYSQL_USER`禁止设置 `MYSQL_DATABASE`、`MYSQL_USER`、`MYSQL_PASSWORD`，避免交叉管理

## 注释约定

- Handler 层必须添加 swagger 注解
- 每个方法必须要简单介绍方法的目的，要有参数，返回值的相关解释
- 每个 gorm 操作必须添加对应的 SQL 注释

