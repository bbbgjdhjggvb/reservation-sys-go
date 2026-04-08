# 部署选项说明

本项目支持三种部署环境，根据你的需求选择合适的方案。

---

## 📊 环境对比

| 特性 | 本地开发环境 | 云服务器测试环境 | 生产环境 |
|------|-------------|-----------------|---------|
| **配置文件** | `config_v1.local.yaml` | `config_v1.test.yaml` | `config_v1.yaml` |
| **Compose 文件** | `docker-compose.local.yaml` | `docker-compose.test.yaml` | `docker-compose.prod.yaml` |
| **部署脚本** | `start-local.sh` | `deploy-test.sh` | `deploy.sh` |
| **运行模式** | debug | debug | release |
| **数据库密码** | `12345678` | `test_pwd_2026` | 自定义强密码 |
| **MySQL 端口映射** | 3307:3306 | 不映射 | 不映射 |
| **Redis 端口映射** | 6380:6379 | 不映射 | 不映射 |
| **前端 URL** | ngrok URL | 服务器 IP | 正式域名 |
| **HTTPS** | ngrok 提供 | 可选 | 必须 |
| **适用场景** | 本地开发调试 | 云服务器测试 | 正式上线 |

---

## 🚀 快速选择指南

### 场景 1: 本地开发调试

```bash
# 使用本地开发环境
./start-local.sh

# 配合 ngrok 测试微信
ngrok http 80
```

**特点：**
- ✅ 端口映射到本地，方便数据库管理
- ✅ 支持 ngrok 代理测试微信
- ✅ debug 模式，日志详细

**文档：** `doc/本地开发指南.md`

---

### 场景 2: 云服务器测试

```bash
# 打包测试环境
./deploy-test.sh

# 上传到服务器
scp reservation-sys-test.tar.gz user@SERVER_IP:/root/

# 在服务器上部署
tar -xzf reservation-sys-test.tar.gz
docker load -i reservation-sys.tar
docker-compose -f docker-compose.test.yaml up -d
```

**特点：**
- ✅ 快速部署，适合测试
- ✅ 使用服务器 IP 或临时域名
- ✅ debug 模式方便排查问题
- ✅ 简化配置，快速迭代

**文档：** `TEST_DEPLOY_GUIDE.md`

---

### 场景 3: 正式生产部署

```bash
# 打包生产环境
./deploy.sh

# 配置域名和 SSL
# 修改配置为 release 模式
# 使用强密码策略

docker-compose -f docker-compose.prod.yaml up -d
```

**特点：**
- ✅ release 模式，性能优化
- ✅ 强密码策略
- ✅ 配置 HTTPS/SSL
- ✅ 使用正式域名

**文档：** `doc/云服务器部署指南.md`

---

## 🔧 配置文件说明

### 测试环境配置要点

`config_v1.test.yaml` 关键配置：

```yaml
server:
  mode: "debug"          # 保持 debug 方便排查

wechat:
  frontend_url: "http://YOUR_SERVER_IP/reserve"  # ⚠️ 需要修改

mysql:
  host: "mysql"          # Docker 服务名
  password: "test_pwd_2026"  # 测试环境密码

jwt:
  secret: "test_jwt_secret_key_2026_reservation_sys"
```

### 生产环境配置要点

`config_v1.yaml` 关键配置：

```yaml
server:
  mode: "release"        # 生产环境

wechat:
  frontend_url: "https://your-domain.com/reserve"  # 正式域名

mysql:
  host: "mysql"
  password: "${MYSQL_PASSWORD}"  # 从环境变量读取

jwt:
  secret: "${JWT_SECRET}"  # 从环境变量读取
```

---

## 📋 部署检查清单

### 测试环境部署检查

- [ ] 修改 `frontend_url` 为服务器 IP
- [ ] 云服务器安全组开放 80 端口
- [ ] 微信测试号配置 URL 和 Token
- [ ] 测试微信消息推送
- [ ] 测试预约功能

### 生产环境部署检查

- [ ] 配置正式域名
- [ ] 申请 SSL 证书
- [ ] 配置 HTTPS
- [ ] 修改数据库密码（强密码）
- [ ] 修改 JWT 密钥
- [ ] 配置微信公众号（正式号）
- [ ] 配置数据库备份
- [ ] 配置监控告警

---

## 🆘 常见问题

### Q1: 如何选择部署环境？

**A:**
- 本地开发 → 使用 `start-local.sh`
- 云服务器测试 → 使用 `deploy-test.sh`
- 正式上线 → 使用 `deploy.sh`

### Q2: 测试环境如何升级到生产环境？

**A:**
1. 修改配置文件模式为 `release`
2. 配置域名和 SSL
3. 修改数据库密码
4. 使用 `docker-compose.prod.yaml`

### Q3: 数据库密码在哪里修改？

**A:**
- 测试环境：`docker-compose.test.yaml` 和 `config_v1.test.yaml`
- 生产环境：`.env` 文件和 `docker-compose.prod.yaml`

### Q4: 如何查看运行日志？

**A:**
```bash
# 测试环境
docker-compose -f docker-compose.test.yaml logs -f

# 生产环境
docker-compose -f docker-compose.prod.yaml logs -f
```

---

## 📚 相关文档

- **本地开发**: `doc/本地开发指南.md`
- **测试环境**: `TEST_DEPLOY_GUIDE.md`
- **生产环境**: `doc/云服务器部署指南.md`
- **API 文档**: `doc/api.md`

---

## 🎯 下一步

根据你的需求选择合适的部署方式：

1. **继续本地开发** → 运行 `./start-local.sh`
2. **部署到云服务器测试** → 运行 `./deploy-test.sh`
3. **准备生产上线** → 查看生产环境部署指南
