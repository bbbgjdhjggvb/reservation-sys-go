# 配置文件总览

## 📁 配置文件结构

```
configs/
├── config_v1.local.yaml          # v1 服务 - 本地开发环境
├── config_v1.test.yaml           # v1 服务 - 测试环境
├── config_v1.yaml                # v1 服务 - 生产环境
├── config_v2.local.yaml          # v2 服务 - 本地开发环境
├── config_v2.test.yaml           # v2 服务 - 测试环境
├── config_v2.yaml                # v2 服务 - 生产环境
├── config_sync_menu.test.yaml    # 菜单同步工具 - 测试环境
└── menu.json                     # 微信菜单配置
```

---

## 🔧 配置文件分类

### 1. 服务配置

| 服务 | 本地开发 | 测试环境 | 生产环境 |
|------|---------|---------|---------|
| v1 - 微信服务 | `config_v1.local.yaml` | `config_v1.test.yaml` | `config_v1.yaml` |
| v2 - 预约服务 | `config_v2.local.yaml` | `config_v2.test.yaml` | `config_v2.yaml` |

### 2. 工具配置

| 工具 | 本地开发 | 测试环境 | 生产环境 |
|------|---------|---------|---------|
| 菜单同步工具 | `config_sync_menu.local.yaml` | `config_sync_menu.test.yaml` | `config_sync_menu.yaml` |

---

## 🎯 配置文件用途

### v1 服务配置

**用途**: 微信服务号相关功能
- 微信消息推送
- OAuth 认证
- 用户管理

**关键配置**:
- `wechat.app_id`: 微信 AppID
- `wechat.app_secret`: 微信 AppSecret
- `wechat.token`: 微信 Token
- `wechat.frontend_url`: 前端预约页面 URL

### v2 服务配置

**用途**: 预约业务逻辑
- 预约申请
- 预约查询
- 时间段管理

**关键配置**:
- `server.port`: 服务端口
- `jwt.secret`: JWT 密钥
- `mysql.*`: 数据库连接

### 菜单同步工具配置

**用途**: 独立同步微信菜单
- 无需启动完整服务
- 支持命令行参数配置
- 适合 CI/CD 集成

**关键配置**:
- `wechat.menu_config_path`: 菜单配置文件路径
- `redis.*`: Redis 连接（用于缓存 access_token）

---

## 📊 环境差异对比

### Redis 配置差异

| 环境 | Host | Port | 说明 |
|------|------|------|------|
| 本地开发 | `127.0.0.1` | `6380` | 映射到本地端口 |
| 测试环境 | `redis` | `6379` | Docker 服务名 |
| 生产环境 | `redis` | `6379` | Docker 服务名 |

### MySQL 配置差异

| 环境 | Host | Port | 说明 |
|------|------|------|------|
| 本地开发 | `mysql` | `3306` | Docker 服务名 |
| 测试环境 | `mysql` | `3306` | Docker 服务名 |
| 生产环境 | `mysql` | `3306` | Docker 服务名 |

---

## 🚀 使用指南

### 本地开发

```bash
# 启动服务
./start-local.sh

# 同步菜单
go run cmd/tools/sync_menu.go -config configs/config_sync_menu.local.yaml
```

### 测试环境

```bash
# 部署
./deploy-test.sh

# 同步菜单
docker exec reservation-v1 /app/v1 -tools sync-menu
```

### 生产环境

```bash
# 部署
./deploy.sh

# 同步菜单
docker exec reservation-v1 /app/v1 -tools sync-menu
```

---

## ⚙️ 配置优先级

### 服务配置

服务启动时会按以下顺序查找配置文件：

1. **环境变量**: `CONFIG_PATH`
2. **默认路径**: `configs/config_v1.yaml` (v1) 或 `configs/config_v2.yaml` (v2)

示例：
```bash
# 使用环境变量指定配置
CONFIG_PATH=configs/config_v1.test.yaml docker-compose up -d
```

### 菜单同步工具配置

工具启动时按以下顺序查找配置：

1. **命令行参数**: `-config` 指定的路径
2. **环境变量**: `CONFIG_PATH`
3. **默认配置**: `configs/config_sync_menu.yaml`
4. **降级配置**: `configs/config_v1.yaml`

示例：
```bash
# 方式1: 命令行参数
go run cmd/tools/sync_menu.go -config configs/config_sync_menu.local.yaml

# 方式2: 环境变量
CONFIG_PATH=configs/config_sync_menu.test.yaml go run cmd/tools/sync_menu.go

# 方式3: 默认配置
go run cmd/tools/sync_menu.go
```

---

## 🔒 安全建议

### 1. 密码管理

- ✅ **测试环境**: 使用固定密码（`test_pwd_2026`）
- ✅ **生产环境**: 使用环境变量（`.env` 文件）
- ❌ **不要**: 将生产环境密码硬编码在配置文件中

### 2. 配置文件管理

- ✅ **提交到 Git**: `config_v1.yaml`, `config_v2.yaml`, `config_sync_menu.yaml`
- ⚠️ **谨慎提交**: `config_*.local.yaml`（包含测试配置，可选择性提交）
- ❌ **不要提交**: `.env` 文件（包含敏感信息）

### 3. JWT 密钥

```yaml
# 测试环境可以使用固定密钥
jwt:
  secret: "test_jwt_secret_key_2026"

# 生产环境应使用强密钥
jwt:
  secret: "${JWT_SECRET}"  # 从环境变量读取
```

---

## 📚 相关文档

- [本地开发指南](doc/本地开发指南.md)
- [测试环境部署指南](TEST_DEPLOY_GUIDE.md)
- [生产环境部署指南](doc/云服务器部署指南.md)
- [菜单同步工具使用指南](doc/菜单同步工具使用指南.md)
- [API 接口文档](doc/api.md)

---

## 🆕 更新记录

### 2026-04-05
- ✨ 新增 `config_sync_menu.local.yaml` - 本地开发菜单同步配置
- ✨ 新增 `config_sync_menu.test.yaml` - 测试环境菜单同步配置
- ✨ 新增 `config_sync_menu.yaml` - 生产环境菜单同步配置
- 🔧 更新 `sync_menu.go` 支持多种配置加载方式
- 📝 新增菜单同步工具使用文档
