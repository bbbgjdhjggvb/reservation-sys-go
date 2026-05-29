# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此仓库中工作时提供指导。

## 常用命令

```sh
# 从 proto 生成 gRPC Go 代码
bash scripts/gen_proto.sh

# 构建所有 Docker 镜像并打包
bash scripts/build.sh

# 生成 url
bash scripts/generate_oauth_url.sh

# 直接运行单个服务
CONFIG_PATH=service/reservation/configs/config_v2.local.yaml go run service/reservation/cmd/main.go
CONFIG_PATH=service/gateway/configs/config_v1.local.yaml go run service/gateway/cmd/main.go
CONFIG_PATH=service/admin/configs/config_v3.yaml    go run service/admin/cmd/main.go

# 启动 Swagger 文档服务（端口 8083）
go run service/swagger/cmd/main.go

# 从注解重新生成 Swagger 文档
/home/fufu/go/bin/swag init -g service/swagger/cmd/main.go -o docs --parseDependency --parseInternal

# 运行单元测试
go test ./service/... ./pkg/... -v -count=1

# 运行集成测试（需要 Docker 环境）
go test ./tests/integration/... -v -count=1

# 运行单个测试用例
go test ./service/reservation/... -run TestSubmitHandler -v

# 重新生成 mock 文件
go generate ./...
```

## 架构

三个 Go 微服务位于 `service/`，共享代码位于 `pkg/`。HTTP 框架：Gin，ORM：GORM，服务间通信：gRPC。

### 服务概览

| 服务 | 目录 | 配置 | 数据库 | 职责 |
|---------|-----|------|-----|---------|
| Gateway (v1) | `service/gateway/` | config_v1 | `home_xy` | 微信 OAuth 登录、用户/管理员认证、通过 gRPC 推送模板消息 |
| Reservation (v2) | `service/reservation/` | config_v2 | `home_res` | 用户端：提交/取消/查询预约，提供前端页面 |
| Admin (v3) | `service/admin/` | config_v3 | `home_res` | 两级审核后台、门锁密码管理，通过 gRPC 调用 Gateway，提供前端页面 |
| Swagger | `service/swagger/` | :8083 | — | 聚合 API 文档，Swagger UI 展示 |

### 共享包 (`pkg/`)

| 包 | 职责 |
|---|------|
| `pkg/config/` | 通用配置结构体（Server、MySQL、Redis、JWT）+ YAML 加载器 |
| `pkg/platform/` | `InitDB()`（GORM/MySQL）、`InitRedis()`（go-redis） |
| `pkg/reservationdb/` | Repository 接口 + GORM 实现，操作 `home_res`（订单、时段、审核），v2 和 v3 共用 |
| `pkg/jwt/` | 用户/管理员 JWT（HMAC-SHA256），`sync.Once` 单例，使用前需 `InitUserJWT` / `InitAdminJWT` |
| `pkg/grpc/` | gRPC 客户端连接辅助 |
| `pkg/constants/` | 共享的状态/角色常量 |

### 请求流程

```
微信 → Gateway (/callback) → 换取 openid → 签发用户 JWT → 重定向到前端
前端 → Reservation API (Bearer token) → AuthMiddleware → handler → repository → MySQL
管理员 → Admin API (Bearer token) → AdminAuthMiddleware → RoleMiddleware → handler → repository + gRPC 调用 Gateway
```

### 数据库访问

Repository 接口定义在 `pkg/reservationdb/repository.go`，GORM 实现，通过 `GetRepository()` 单例访问（未初始化则 panic）。状态转换使用乐观锁（`WHERE status = fromStatus`），提交预约使用 `SELECT ... FOR UPDATE` 防重复。

## 测试

- 单元测试：放在对应包内，使用 `gomock` / `sqlmock` / `miniredis` + `testify/assert`，handler 测试用 `httptest.NewRecorder()` 配合 Gin
- 集成测试：放在 `./tests/integration/`，通过 Docker 容器使用真实 MySQL 和 Redis，测试完整 HTTP 请求链路（请求 → 中间件 → handler → service → 真实数据库 → 响应）
- Mock 文件命名：`mock_*.go`

## 项目约定

- Go module：`reservation-sys`
- 入口：`service/<名称>/cmd/main.go`，支持 `--config` 标志或 `CONFIG_PATH` 环境变量
- 配置：`service/<名称>/configs/config_v<N>.yaml`（提交），`config_v<N>.local.yaml`（gitignore，本地覆盖）
- 前端：静态 HTML 位于 `service/<名称>/frontend/`，由 Gin `LoadHTMLGlob` 提供，须采用深圳大学荔枝红风格
- Proto：`service/gateway/api/proto/`，生成代码在 `service/gateway/api/gen/` 并复制到 `service/admin/api/gen/`
- `.test-data/`：本地测试状态（token、预约 ID、日志），已 gitignore
- 本地开发：`docker-compose.local.yaml` 启动 MySQL + Redis；完整环境包括 gateway + reservation + admin + nginx
- MySQL 初始化与授权解耦：库/表/权限统一在 `deploy/mysql/init.sql` 管理，`docker-compose` 仅设 `MYSQL_ROOT_PASSWORD`，禁止设 `MYSQL_DATABASE`、`MYSQL_USER`、`MYSQL_PASSWORD`

## 注释约定

- Handler 层必须添加 swagger 注解
- 每个方法须说明目的、参数、返回值
- 每个 GORM 操作须添加对应的 SQL 注释

## 文档约定

- 图像用 mermaid 绘制，表格用 markdown 表格，不要用字符组合图像或表格

## 编码规范

- 包，库函数不应该在函数内部 fatal，而应该返回 error，让调用者决定是否 fatal