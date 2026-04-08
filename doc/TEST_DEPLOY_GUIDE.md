# 云服务器测试环境部署指南

## 🎯 测试环境特点

- ✅ 使用 `debug` 模式，方便查看日志和排查问题
- ✅ 简化密码策略（但仍比开发环境更安全）
- ✅ 保持最小化配置，快速部署
- ✅ 支持 HTTP 测试（后续可升级 HTTPS）

---

## 📦 快速部署（5 步完成）

### 本地操作

```bash
# 1. 打包测试环境
chmod +x deploy-test.sh
./deploy-test.sh

# 2. 上传到云服务器
scp reservation-sys-test.tar.gz user@YOUR_SERVER_IP:/root/
```

### 云服务器操作

```bash
# 3. 解压并导入镜像
ssh user@YOUR_SERVER_IP
cd /root
tar -xzf reservation-sys-test.tar.gz
docker load -i reservation-sys.tar

# 4. 修改配置（重要！）
vim configs/config_v1.test.yaml
# 将 frontend_url 改为: http://YOUR_SERVER_IP/reserve

# 5. 启动服务
docker-compose -f docker-compose.test.yaml up -d
```

---

## 🔧 配置说明

### 测试环境配置文件

| 文件 | 说明 |
|------|------|
| `config_v1.test.yaml` | 微信服务测试配置 |
| `config_v2.test.yaml` | 预约服务测试配置 |
| `docker-compose.test.yaml` | 测试环境编排文件 |

### 关键配置项

#### 1. 修改服务器 IP

编辑 `configs/config_v1.test.yaml`：

```yaml
wechat:
  frontend_url: "http://YOUR_SERVER_IP/reserve"  # 改为你的服务器 IP
```

#### 2. 数据库配置（已预设）

```yaml
mysql:
  host: "mysql"           # Docker 服务名
  user: "res_user"
  password: "test_pwd_2026"  # 测试环境密码
```

#### 3. 微信 Token

```yaml
wechat:
  token: "mytesttoken123"  # 符合微信要求的格式
```

---

## 🌐 微信配置

### 在微信测试号平台填写

**接口配置信息：**
- **URL**: `http://YOUR_SERVER_IP/wx`
- **Token**: `mytesttoken123`

**网页授权域名（可选）：**
- 添加: `YOUR_SERVER_IP`

> ⚠️ 注意：测试环境使用 HTTP，如果微信要求 HTTPS，可以使用 ngrok 或配置 SSL 证书

---

## 📋 服务管理

### 常用命令

```bash
# 查看服务状态
docker-compose -f docker-compose.test.yaml ps

# 查看日志
docker-compose -f docker-compose.test.yaml logs -f

# 查看特定服务日志
docker-compose -f docker-compose.test.yaml logs -f v1-service

# 重启服务
docker-compose -f docker-compose.test.yaml restart

# 停止服务
docker-compose -f docker-compose.test.yaml down

# 停止并删除数据
docker-compose -f docker-compose.test.yaml down -v
```

---

## ✅ 验证测试

### 1. 检查容器状态

```bash
docker-compose -f docker-compose.test.yaml ps
```

应该看到 5 个容器都是 `Up` 状态。

### 2. 测试微信接口

```bash
# 本地测试
curl "http://localhost/wx?signature=test&timestamp=123&nonce=456&echostr=hello"

# 外网测试（从其他机器）
curl "http://YOUR_SERVER_IP/wx"
```

### 3. 测试预约页面

浏览器访问：`http://YOUR_SERVER_IP/reserve`

### 4. 查看日志

```bash
# 实时查看所有日志
docker-compose -f docker-compose.test.yaml logs -f

# 查看 v1 服务日志
docker-compose -f docker-compose.test.yaml logs -f v1-service
```

---

## 🐛 故障排查

### 问题 1: 容器无法启动

```bash
# 查看详细日志
docker-compose -f docker-compose.test.yaml logs v1-service

# 检查配置文件
docker exec reservation-v1 cat /app/configs/config_v1.test.yaml
```

### 问题 2: 数据库连接失败

```bash
# 检查 MySQL 是否就绪
docker exec reservation-mysql mysql -u res_user -ptest_pwd_2026 -e "SELECT 1"

# 查看数据表
docker exec reservation-mysql mysql -u res_user -ptest_pwd_2026 home_xy -e "SHOW TABLES"
```

### 问题 3: 微信验证失败

检查项：
1. ✅ Token 是否一致（`mytesttoken123`）
2. ✅ URL 是否可访问（`http://YOUR_SERVER_IP/wx`）
3. ✅ 服务器安全组是否开放 80 端口
4. ✅ 查看 v1 服务日志

```bash
docker-compose -f docker-compose.test.yaml logs v1-service | grep -i error
```

---

## 🔄 更新部署

### 代码更新后重新部署

```bash
# 1. 本地重新打包
./deploy-test.sh

# 2. 上传到服务器
scp reservation-sys-test.tar.gz user@YOUR_SERVER_IP:/root/

# 3. 服务器上更新
ssh user@YOUR_SERVER_IP
cd /root
tar -xzf reservation-sys-test.tar.gz
docker load -i reservation-sys.tar

# 4. 重启服务
docker-compose -f docker-compose.test.yaml down
docker-compose -f docker-compose.test.yaml up -d
```

---

## 🔒 安全建议

虽然是测试环境，仍建议：

1. **修改默认密码** - 编辑 `docker-compose.test.yaml` 中的数据库密码
2. **限制端口访问** - 云服务器安全组只允许必要的 IP 访问
3. **定期备份数据** - 导出数据库备份

```bash
# 备份数据库
docker exec reservation-mysql mysqldump -u res_user -ptest_pwd_2026 home_xy > backup_$(date +%Y%m%d).sql
```

---

## 📊 性能监控

### 查看资源使用

```bash
# 查看所有容器资源使用
docker stats

# 查看特定容器
docker stats reservation-v1 reservation-v2 reservation-mysql
```

---

## 🚀 升级到生产环境

测试完成后，升级到生产环境：

1. 使用 `config_v1.yaml` 和 `config_v2.yaml`
2. 修改 `mode: "release"`
3. 配置 HTTPS/SSL
4. 使用更强的密码策略
5. 配置域名和正式的微信配置

---

## 📞 需要帮助？

查看日志：
```bash
docker-compose -f docker-compose.test.yaml logs -f
```

检查配置：
```bash
docker exec reservation-v1 env | grep CONFIG
docker exec reservation-v1 cat /app/configs/config_v1.test.yaml
```
