# 快速部署清单

## 📦 本地操作（在你的开发机器上）

### 1. 打包部署文件
```bash
./deploy.sh
```

### 2. 上传到云服务器
```bash
scp reservation-sys-deploy.tar.gz user@your-server-ip:/root/
```

---

## ☁️ 云服务器操作

### 1. 解压并部署
```bash
# SSH 登录
ssh user@your-server-ip

# 解压
cd /root
tar -xzf reservation-sys-deploy.tar.gz

# 导入镜像
docker load -i reservation-sys.tar
```

### 2. 配置环境
```bash
# 复制并编辑环境变量
cp .env.example .env
vim .env

# 修改微信配置（重要！）
vim configs/config_v1.yaml
# 将 frontend_url 改为: https://your-domain.com/reserve
```

### 3. 启动服务
```bash
docker-compose -f docker-compose.prod.yaml up -d
```

### 4. 验证服务
```bash
# 查看容器状态
docker-compose -f docker-compose.prod.yaml ps

# 查看日志
docker-compose -f docker-compose.prod.yaml logs -f

# 测试接口
curl http://localhost/wx
curl http://localhost/reserve
```

---

## 🔧 微信配置

### 在微信公众平台配置

**URL**: `https://your-domain.com/wx`  
**Token**: `mytesttoken123`

**网页授权域名**: `your-domain.com`

---

## 📋 部署文件清单

生成的 `reservation-sys-deploy.tar.gz` 包含：

```
├── reservation-sys.tar          # Docker 镜像
├── docker-compose.prod.yaml     # 生产环境编排文件
├── .env.example                 # 环境变量模板
├── configs/                     # 配置文件
│   ├── config_v1.yaml          # v1 服务配置
│   ├── config_v2.yaml          # v2 服务配置
│   └── menu.json               # 微信菜单配置
├── deploy/                      # 部署配置
│   ├── mysql/init.sql          # 数据库初始化
│   └── nginx/nginx.config      # Nginx 配置
└── internal/reservation/frontend/ # 前端页面
```

---

## 🎯 常用命令

```bash
# 启动
docker-compose -f docker-compose.prod.yaml up -d

# 停止
docker-compose -f docker-compose.prod.yaml down

# 重启
docker-compose -f docker-compose.prod.yaml restart

# 查看日志
docker-compose -f docker-compose.prod.yaml logs -f v1-service

# 进入容器
docker exec -it reservation-v1 sh

# 备份数据库
docker exec reservation-mysql mysqldump -u res_user -p home_xy > backup.sql

# 更新服务
docker-compose -f docker-compose.prod.yaml pull
docker-compose -f docker-compose.prod.yaml up -d
```

---

## ⚠️ 重要提示

1. **修改数据库密码** - 编辑 `.env` 文件
2. **配置域名** - 修改 `configs/config_v1.yaml` 中的 `frontend_url`
3. **开放端口** - 确保云服务器安全组开放 80/443 端口
4. **微信 Token** - 必须与配置文件中的 `wechat.token` 一致

---

## 🆘 故障排查

```bash
# 查看所有日志
docker-compose -f docker-compose.prod.yaml logs

# 测试数据库连接
docker exec reservation-mysql mysql -u res_user -p -e "SELECT 1"

# 测试 Redis 连接
docker exec reservation-redis redis-cli ping

# 检查网络
docker network inspect reservation_sys_go_reservation-net
```

---

## 📚 详细文档

查看 `doc/云服务器部署指南.md` 获取完整部署文档。
