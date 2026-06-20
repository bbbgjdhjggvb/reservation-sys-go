# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在此仓库中工作时提供指导。
这个项目是一个场地预约审核系统。用户量大约在 500 左右，很少有高突发流量。

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

**数据库表结构统一由 `deploy/mysql/init.sql` 管理**，不使用 GORM AutoMigrate。原因：
- AutoMigrate 无法创建数据库、用户、权限（`CREATE DATABASE` / `CREATE USER` / `GRANT`）
- AutoMigrate 无法插入种子数据（管理员账号等）
- 多副本启动时并发 DDL 存在风险
- 隐式 DDL 变更无法在 PR 中审查
- `init.sql` 显式、可审计、功能完整，适配 Docker `/docker-entrypoint-initdb.d/` 机制

Gateway 服务的 `home_xy` 数据库（users、admins 表）同样由 `init.sql` 管理，`auth.InitModule` 和 `auth.InitAdminModule` 仅创建服务实例，不执行表迁移。

## 测试

- 单元测试：放在对应包内，使用 `gomock` / `sqlmock` / `miniredis` + `testify/assert`，handler 测试用 `httptest.NewRecorder()` 配合 Gin
- 集成测试：放在 `./tests/integration/`，通过 Docker 容器使用真实 MySQL 和 Redis，测试完整 HTTP 请求链路（请求 → 中间件 → handler → service → 真实数据库 → 响应）
- Mock 文件命名：`mock_*.go`

## 文件组织

```
.
├── CLAUDE.md                          # 本文件
├── docker-compose.local.yaml          # 本地开发环境（MySQL + Redis）
├── docker-compose.e2e.yaml            # E2E 集成测试环境（完整服务栈 + nginx）
├── docker-compose.prod.yaml           # 生产环境
├── .env.example                       # 环境变量模板
│
├── deploy/                            # 部署配置
│   ├── mysql/init.sql                 # 数据库初始化（库/表/权限）
│   └── nginx/nginx.config             # Nginx 反向代理配置
│
├── scripts/                           # 运维脚本
│   ├── build.sh                       # 构建 Docker 镜像
│   ├── gen_proto.sh                   # 从 proto 生成 gRPC 代码
│   ├── generate_oauth_url.sh          # 生成微信 OAuth URL
│   └── e2e_test.sh                    # E2E 测试（docker-compose 生命周期 + go test）
│
├── pkg/                               # 共享 Go 包
│   ├── config/                        # 通用配置加载
│   ├── constants/                     # 管理员角色常量
│   ├── grpc/                          # gRPC 连接辅助
│   ├── jwt/                           # JWT 签发/验证
│   ├── platform/                      # DB/Redis 初始化
│   └── reservationdb/                 # 共享数据库层（模型 + Repository）
│
├── service/                           # Go 微服务
│   ├── gateway/                       # Gateway 服务（微信登录、模板消息）
│   │   ├── cmd/main.go
│   │   ├── api/proto/                 # gRPC proto 定义
│   │   ├── api/gen/                   # gRPC 生成代码
│   │   └── configs/                   # YAML 配置文件
│   ├── reservation/                   # 预约服务（用户端 API）
│   │   ├── cmd/main.go
│   │   ├── configs/
│   │   └── frontend/                  # 旧前端（已废弃，保留空目录）
│   ├── admin/                         # 审核管理服务（管理员端 API）
│   │   ├── cmd/main.go
│   │   ├── auth/                      # 认证模块
│   │   ├── review/                    # 审核模块
│   │   ├── configs/
│   │   └── frontend/                  # 旧前端（已废弃，保留空目录）
│   └── swagger/                       # Swagger 文档聚合
│
├── frontend/                          # Vue 前端（pnpm monorepo）
│   ├── packages/
│   │   ├── shared/                    # 共享类型定义（ORDER_STATUS_MAP、API 类型等）
│   │   ├── reservation/               # 用户端（预约日历、我的预约）
│   │   │   └── src/
│   │   │       ├── views/             # ReserveView、MyOrdersView
│   │   │       ├── components/        # CalendarGrid、OrderCard、SlotCell 等
│   │   │       ├── composables/       # useCalendar、useToast
│   │   │       ├── stores/            # auth（Pinia）
│   │   │       └── api/client.ts      # HTTP 请求封装
│   │   └── admin/                     # 管理员端（审核管理）
│   │       └── src/
│   │           ├── views/             # LoginView、DashboardView
│   │           ├── components/        # AdminOrderCard、ReviewModal、StatusTabs 等
│   │           ├── composables/       # useAdminOrders
│   │           ├── stores/            # admin（Pinia）
│   │           └── api/client.ts      # HTTP 请求封装
│   └── package.json
│
├── tests/                             # 集成测试
│   └── integration/                   # 通过 nginx 发送真实 HTTP 请求的 E2E 测试
│
├── doc/                               # 项目文档
│   ├── 需求/                          # 需求与方案设计文档
│   ├── 开发指导/                      # 开发指南、部署指南、文件结构说明
│   ├── 测试/                          # 测试文档
│   └── 学习/                          # 技术学习笔记
│
└── docs/                              # Swagger 生成文档（docs.go、swagger.json）
```

### 前端架构要点

- **构建工具**：Vite + Vue 3 + TypeScript + Pinia
- **前端路由**使用 `createWebHistory`，admin 的 base 为 `/admin/`，reservation 为 `/reservation/`
- **`shared/types.ts`** 是前后端状态码的权威映射，`ORDER_STATUS_MAP` 定义所有订单状态的中文显示文本
- 前端组件中的状态码判断 **必须** 与 `pkg/reservationdb/model.go` 保持一致（1-7），不一致会导致审核按钮不显示、列表筛选失效等问题
- 旧 `service/*/frontend/` 目录为空，前端已全部迁移至 `frontend/packages/`

## 订单状态机

订单状态码定义在 `pkg/reservationdb/model.go`，状态转换由 `service/admin/review/service.go`（审核）和 `service/reservation/service.go`（提交/取消）控制。

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
| 提交预约 | 用户 | — | 1 (待一级审核) | `reservation/service.go:Submit()` |
| 一级审核通过 | 一级管理员 | 1 | 2 (待二级审核) | `admin/review/service.go:Level1Review()` |
| 一级审核拒绝 | 一级管理员 | 1 | 3 (一级审核拒绝) | `admin/review/service.go:Level1Review()` |
| 二级审核通过 | 二级管理员 | 2 | 5 (审核通过) | `admin/review/service.go:Level2Review()` |
| 二级审核拒绝 | 二级管理员 | 2 | 4 (二级审核拒绝) | `admin/review/service.go:Level2Review()` |
| 用户取消 | 用户 | 1 | 6 (已取消) | `reservation/service.go:Cancel()` |
| 完成 | — | — | 7 (已完成) | 预留，暂无触发逻辑 |

---

## 项目约定

- Go module：`reservation-sys`
- 入口：`service/<名称>/cmd/main.go`，支持 `--config` 标志或 `CONFIG_PATH` 环境变量
- 配置：`service/<名称>/configs/config_v<N>.yaml`（提交），`config_v<N>.local.yaml`（gitignore，本地覆盖）
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