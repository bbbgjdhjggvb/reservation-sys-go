# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此仓库中工作时提供指导。
这个项目是一个场地预约审核系统。用户量大约在 500 左右，很少有高突发流量。

## 常用命令

```sh
# 构建 Docker 镜像并打包部署文件
bash scripts/build.sh

# 直接运行（宿主机直连 Docker 中的 MySQL/Redis）
CONFIG_PATH=configs/config.local.yaml go run cmd/server/main.go

# Docker Compose 本地开发（完整环境）
docker compose -f docker-compose.local.yaml up -d --build

# Docker Compose E2E 测试
docker compose -f docker-compose.e2e.yaml up -d --build

# 运行单元测试
go test ./internal/... ./pkg/... -v -count=1

# 运行集成测试（需要 Docker 环境）
go test ./tests/integration/... -v -count=1

# 运行单个测试用例
go test ./internal/reservation/... -run TestSubmit -v

# 重新生成 mock 文件
go generate ./...

# 生成微信 OAuth 授权 URL
go run tools/oauth/main.go -s reserve
```

## 架构

单体应用，单一入口 `cmd/server/main.go`，监听 `:8080`。HTTP 框架：Gin，ORM：GORM。

重构将原来的 3 个微服务（gateway、reservation、admin）合并为单体，消除 gRPC 通信和 Redis Pub/Sub，改为进程内直接调用和进程内事件总线。**新代码在 `internal/`，旧代码在 `service/` 和 `pkg/` 保留但逐步废弃。**

### 模块组织（`internal/`）

| 模块 | 目录 | 职责 |
|------|------|------|
| auth | `internal/auth/` | 微信 OAuth 登录、用户/管理员认证、模板消息通知 |
| reservation | `internal/reservation/` | 用户端：提交/取消/查询预约 |
| review | `internal/review/` | 管理员端：两级审核、门锁密码管理 |
| model | `internal/model/` | 数据库模型、订单状态常量（1-7） |
| router | `internal/router/` | 路由注册器，封装 Gin Engine，支持权限和限流 |
| permissions | `internal/permissions/` | Permission 枚举（UserAuth/AdminAuth/Level1Review/Level2Review）→ 中间件链映射 |
| middleware | `internal/middleware/` | 认证、角色鉴权、CORS、限流中间件 |
| event | `internal/event/` | 进程内事件总线（替代 Redis Pub/Sub） |
| sse | `internal/sse/` | SSE 实时推送，Hub 订阅 event.Bus 广播 |
| platform | `internal/platform/` | DB（GORM/MySQL）和 Redis 初始化 |
| errors | `internal/errors/` | 共享错误变量 |

### 模块内三层分离

每个模块内部按 store → service → handler 分层：

| 层 | 目录/文件 | 职责 |
|----|-----------|------|
| store | `store/*.go` | 数据库 CRUD，GORM 实现 |
| service | `<模块名>.go` | 业务逻辑，构造函数注入依赖 |
| handler | `handler.go` | HTTP handler，参数校验，调用 service |
| routes | `routes.go` | 调用 `r.Register()` 自注册路由 |
| errors | `errors.go` | 模块专用错误变量 |
| dto | `dto.go` | 请求/响应结构体 |

### 共享包（`pkg/`，旧代码，仅维护不扩展）

| 包 | 职责 |
|---|------|
| `pkg/config/` | 统一配置结构体（Server、MySQL、Redis、JWT、Wechat、RateLimit）+ YAML 加载 + 默认配置生成 |
| `pkg/jwt/` | 用户/管理员 JWT（HMAC-SHA256），`sync.Once` 单例 |

### 请求流程

```
微信 → /wx (服务器验证/消息) 或 /api/gateway/auth/callback (OAuth 回调)
        → auth.Handler → auth.Service → auth/store → MySQL

用户前端 → /api/reservation/* (Bearer JWT) → UserAuth 中间件 → reservation.Handler → Service → store → MySQL

管理前端 → /api/admin/* (Bearer JWT) → AdminAuth 中间件 → review.Handler → Service → store + auth.Notifier → MySQL
```

### 路由注册

每个模块通过 `RegisterRoutes(r *router.Registrar, ...)` 注册路由。Registrar.Register 签名：

```go
// rateLimitKey=nil 不限流，perm=nil 公开端点
func (r *Registrar) Register(method, path string, handler gin.HandlerFunc, rateLimitKey *string, perm *permissions.Permission)
```

用法示例：
```go
r.Register("GET", "/wx", h.WeChatHandler, nil, nil)                           // 公开端点
r.Register("GET", "/api/reservation/my", h.MyOrders, nil, &permissions.UserAuth)  // 需认证
r.Register("POST", "/api/reservation/submit", h.Submit, router.Limiter("submit"), &permissions.UserAuth) // 认证+限流
```

### 权限系统

权限定义在 `internal/permissions/permissions.go`：

| 常量 | 中间件链 | 用途 |
|------|----------|------|
| `UserAuth` | UserAuth | 用户 JWT 认证 |
| `AdminAuth` | AdminAuth | 管理员 JWT 认证 |
| `Level1Review` | AdminAuth + RequireRole(1) | 一级管理员 |
| `Level2Review` | AdminAuth + RequireRole(2) | 二级管理员 |

### 配置

| 文件 | 使用场景 | MySQL host | mode |
|------|----------|------------|------|
| `configs/config.yaml` | Docker Compose（local/e2e/prod） | `mysql:3306` | release |
| `configs/config.local.yaml` | 宿主机直接 `go run` | `127.0.0.1:3307` | debug |

首次运行时若配置文件不存在，`MustLoad()` 自动生成默认配置（`DefaultConfig()`）并写入磁盘。
配置路径优先级：`--config` 标志 > `CONFIG_PATH` 环境变量 > `config.yaml`。

### 数据库

**数据库表结构统一由 `deploy/mysql/init.sql` 管理**，不使用 GORM AutoMigrate。原因：
- AutoMigrate 无法创建数据库、用户、权限（`CREATE DATABASE` / `CREATE USER` / `GRANT`）
- AutoMigrate 无法插入种子数据（管理员账号等）
- 多副本启动时并发 DDL 存在风险
- 隐式 DDL 变更无法在 PR 中审查
- `init.sql` 显式、可审计、功能完整，适配 Docker `/docker-entrypoint-initdb.d/` 机制

所有表在单一数据库 `reservation_sys` 中。

**种子管理员账号：**

| 账号 | 密码 | 角色 |
|------|------|------|
| `admin1` | `123456` | 一级管理员 |
| `admin2` | `123456` | 二级管理员 |

## 订单状态机

订单状态码定义在 `internal/model/reservation.go`，状态转换由 `internal/review/review.go`（审核）和 `internal/reservation/reservation.go`（提交/取消）控制。

### 状态码定义

| 常量 | 数值 | 含义 |
|------|------|------|
| `StatusPendingLevel1` | 1 | 等待一级审核 |
| `StatusPendingLevel2` | 2 | 等待二级审核 |
| `StatusRejectedLevel1` | 3 | 一级审核拒绝 |
| `StatusRejectedLevel2` | 4 | 二级审核拒绝 |
| `StatusApproved` | 5 | 审核通过 |
| `StatusCancelled` | 6 | 订单已取消 |
| `StatusCompleted` | 7 | 订单已完成（预留，暂无代码触发） |

### 转换规则

| 操作 | 触发方 | 源状态 | 目标状态 | 实现位置 |
|------|--------|--------|----------|----------|
| 提交预约 | 用户 | — | 1 | `reservation/reservation.go:Submit()` |
| 一级审核通过 | 一级管理员 | 1 | 2 | `review/review.go:Level1Review()` |
| 一级审核拒绝 | 一级管理员 | 1 | 3 | `review/review.go:Level1Review()` |
| 二级审核通过 | 二级管理员 | 2 | 5 | `review/review.go:Level2Review()` |
| 二级审核拒绝 | 二级管理员 | 2 | 4 | `review/review.go:Level2Review()` |
| 用户取消 | 用户 | 1 | 6 | `reservation/reservation.go:Cancel()` |
| 完成 | — | — | 7 | 预留，暂无触发逻辑 |

## 测试

- 单元测试：放在对应包内，使用 `gomock` / `sqlmock` / `miniredis` + `testify/assert`，handler 测试用 `httptest.NewRecorder()` 配合 Gin
- 集成测试：放在 `./tests/integration/`，通过 Docker 容器使用真实 MySQL 和 Redis，测试完整 HTTP 请求链路
- Mock 文件命名：`mock_*.go`

## 文件组织

```
.
├── CLAUDE.md
├── Dockerfile                           # 多阶段构建（golang:1.24-alpine → alpine:3.21）
├── docker-compose.local.yaml            # 本地开发环境（MySQL + Redis + server + nginx，仅 HTTP）
├── docker-compose.e2e.yaml              # E2E 测试环境（完整服务栈，仅 HTTP）
├── docker-compose.prod.yaml             # 生产环境（含 HTTPS + SSL）
├── .env.example                         # 环境变量模板
│
├── cmd/server/                          # 程序入口
│   ├── main.go                          # 模块组装 + 启动
│   └── doc.go                           # 包文档（新手引导）
│
├── internal/                            # 私有应用代码（新架构）
│   ├── auth/                            # 认证模块
│   │   ├── auth.go                      # OAuth 登录 service
│   │   ├── wechat.go                    # 微信 OAuth 客户端
│   │   ├── notify.go                    # 模板消息通知
│   │   ├── handler.go
│   │   ├── routes.go
│   │   ├── errors.go
│   │   └── store/{user,admin}.go        # 用户/管理员数据访问
│   ├── reservation/                     # 预约模块（用户端）
│   │   ├── reservation.go               # 预约/取消业务逻辑
│   │   ├── handler.go
│   │   ├── routes.go
│   │   ├── dto.go
│   │   ├── errors.go
│   │   └── store/{order,slot}.go
│   ├── review/                          # 审核模块（管理员端）
│   │   ├── review.go                    # 两级审核业务逻辑
│   │   ├── handler.go
│   │   ├── routes.go
│   │   ├── dto.go
│   │   ├── errors.go
│   │   └── store/{order,record}.go
│   ├── model/                           # 数据库模型 + 状态常量
│   ├── router/                          # Registrar 路由注册器
│   ├── permissions/                     # Permission 枚举 + MiddlewareMapper
│   ├── middleware/                      # 中间件（auth, admin_auth, role, cors, ratelimit）
│   ├── event/                           # 进程内事件总线
│   ├── sse/                             # SSE Hub + HTTP handler
│   ├── platform/                        # DB/Redis 初始化
│   └── errors/                          # 共享错误变量
│
├── pkg/                                 # 共享 Go 包（旧架构，仅维护）
│   ├── config/base.go                   # 配置结构体 + MustLoad + DefaultConfig + ConfigPath
│   ├── jwt/                             # JWT 签发/验证
│   ├── platform/                        # 旧 DB/Redis 初始化（service/ 使用）
│   └── reservationdb/                   # 旧数据库层（service/ 使用）
│
├── service/                             # 旧微服务代码（保留编译通过，不新增功能）
├── frontend/                            # Vue 前端（pnpm monorepo）
│   └── packages/
│       ├── shared/                      # 共享类型（ORDER_STATUS_MAP 等）
│       ├── reservation/                 # 用户端 SPA
│       └── admin/                       # 管理员端 SPA
│
├── deploy/                              # 部署配置
│   ├── mysql/init.sql                   # 数据库初始化（库/表/种子数据）
│   └── nginx/
│       ├── nginx.config                 # 生产 Nginx（HTTPS + HTTP→HTTPS 重定向）
│       └── nginx.local.config           # 本地 Nginx（仅 HTTP）
│
├── configs/
│   ├── config.yaml                      # Docker 环境配置（mysql:3306）
│   └── config.local.yaml                # 宿主机直连配置（127.0.0.1:3307）
│
├── scripts/
│   ├── build.sh                         # 构建镜像 → 导出 tar → 打包
│   └── e2e_test.sh                      # E2E 测试脚本
│
├── tools/
│   ├── oauth/main.go                    # 微信 OAuth URL 生成工具
│   ├── menu/                            # 微信菜单配置
│   └── jwt/                             # JWT 工具
│
├── tests/integration/                   # 集成测试
├── doc/                                 # 项目文档
└── docs/                                # Swagger 生成文档
```

### 前端架构要点

- **构建工具**：Vite + Vue 3 + TypeScript + Pinia
- **前端路由**使用 `createWebHistory`，admin 的 base 为 `/admin/`，reservation 为 `/`
- **`shared/types.ts`** 是前后端状态码的权威映射，`ORDER_STATUS_MAP` 定义所有订单状态的中文显示文本
- 前端组件中的状态码判断 **必须** 与 `internal/model/reservation.go` 保持一致（1-7），不一致会导致审核按钮不显示、列表筛选失效等问题

### Nginx 配置说明

| 文件 | 使用场景 | 端口 | HTTPS |
|------|----------|------|-------|
| `nginx.config` | 生产（`docker-compose.prod.yaml` + `build.sh` 打包） | 80→443 重定向 + 443 SSL | 有 |
| `nginx.local.config` | 本地开发 + E2E 测试 | 80 直接服务 | 无 |

## 核心设计原则

1. **构造函数注入**：每个模块通过 `NewXxx(db, cfg, deps...)` 创建，不用全局单例
2. **消费者定义接口**：review 模块定义 `Notifier`、`SlotStore` 接口，auth/reservation 提供实现（DIP 依赖反转）
3. **路由自治注册**：每个模块通过 `RegisterRoutes(r, ...)` 向 Registrar 注册路由，main.go 只做组装
4. **进程内事件总线**：`event.Bus` 替代 Redis Pub/Sub，`Subscribe` channel 驱动 SSE Hub 广播
5. **模块内三层分离**：store（数据访问） → service（业务逻辑） → handler（HTTP）+ routes（路由注册）

## 项目约定

- Go module：`reservation-sys`
- 入口：`cmd/server/main.go`，支持 `--config` 标志或 `CONFIG_PATH` 环境变量
- 配置：`configs/config.yaml`（Docker）、`configs/config.local.yaml`（宿主机直连）
- `.test-data/`：本地测试状态（token、预约 ID、日志），已 gitignore
- MySQL 初始化与授权解耦：库/表/权限统一在 `deploy/mysql/init.sql` 管理，`docker-compose` 仅设 `MYSQL_ROOT_PASSWORD`

## 编码规范

- 包，库函数不应该在函数内部 fatal，而应该返回 error，让调用者决定是否 fatal
- 每个方法须说明目的、参数、返回值
