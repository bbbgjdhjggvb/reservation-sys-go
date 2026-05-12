# Docker 基础教程

本文档面向项目新人，帮助你掌握 Docker 最基本的使用方法，足以应对日常开发和部署工作。

---

## 一、核心概念

在开始敲命令之前，先理解三个核心概念：

| 概念 | 类比 | 说明 |
|------|------|------|
| **镜像 (Image)** | 安装包/ISO | 只读模板，包含运行应用所需的代码、依赖、配置等。一个镜像可以创建多个容器 |
| **容器 (Container)** | 运行中的程序 | 镜像的运行实例，轻量级、隔离的运行环境。可启动、停止、删除 |
| **数据卷 (Volume)** | 外接硬盘 | 容器删除后数据会丢失，数据卷用于持久化存储（如数据库文件） |

> 简单理解：**镜像**是类，**容器**是实例，**数据卷**是持久化的外挂存储。

---

## 二、本项目架构

本项目采用微服务架构，通过 Docker Compose 编排。

`Docker-compose` 存在的目的: 简化多个容器部署的过程，一个文件替换多个命令。

**三个环境对应的 Compose 文件：**

| 环境 | 文件 | 用途 |
|------|------|------|
| 本地开发 | `docker-compose.local.yaml` | 本机开发调试，映射端口到宿主机 |
| 生产环境 | `docker-compose.prod.yaml` | 正式部署，使用预构建的镜像 |

---

## 三、镜像操作

### 1. 查看镜像

```sh
docker images

# 输出示例：
# REPOSITORY                TAG           IMAGE ID       CREATED        SIZE
# reservation-admin         test          3fa2839c0b87   4 days ago     26.2MB
# reservation-reservation   test          5015b67ae948   4 days ago     26.1MB
# reservation-gateway       test          7516ed4365ce   4 days ago     30.3MB
# nginx                     alpine        d5030d429039   6 weeks ago    62.2MB
# mysql                     8.0           a123df39bc7c   7 weeks ago    786MB
# redis                     7-alpine      aa189b5a1954   2 months ago   41.4MB
```

### 2. 删除镜像

```sh
# 按 名称:标签 删除
docker rmi reservation-admin:test

# 按 镜像ID 删除
docker rmi 3fa2839c0b87

# 如果镜像正在被容器使用，需先删除容器
```

### 3. 导出镜像

用于将构建好的镜像传到服务器上：

```sh
# 导出单个镜像
# docker save -o <输出文件名>.tar <镜像名:标签>
docker save -o reservation-admin.tar reservation-admin:test

# 导出多个镜像到一个文件
docker save -o reservation-sys.tar reservation-gateway:test reservation-reservation:test reservation-admin:test
```

### 4. 加载镜像

在目标服务器上导入之前导出的镜像：

```sh
# docker load -i <文件名>.tar
docker load -i reservation-sys.tar
```

### 5. 构建镜像
镜像的构建需要编写 Dockerfile

```sh
# 在项目根目录，根据 Dockerfile 构建镜像
# docker build -t <镜像名:标签> -f <Dockerfile路径> <构建上下文>
docker build -t reservation-admin:test -f service/admin/Dockerfile .
```

> **提示**：本项目通常通过 `docker-compose up --build` 自动构建，不需要手动执行 `docker build`。

---

## 四、容器操作

### 1. 查看容器

```sh
# 查看正在运行的容器
docker ps

# 查看所有容器（包括已停止的）
docker ps -a

# 只显示容器ID（配合其他命令使用）
docker ps -q
docker stop $(docker ps -q)
```

### 2. 启动/停止/重启容器

```sh
# 启动已停止的容器
docker start reservation-admin

# 停止正在运行的容器
docker stop reservation-admin

# 重启容器（常用于配置修改后生效）
docker restart reservation-admin

# 也可以使用容器ID
docker restart 66b4fa344e56
```

### 3. 删除容器

```sh
# 删除已停止的容器
docker rm reservation-admin

# 删除正在运行的容器（先停止再删除）
docker rm -f reservation-admin

# 删除所有已停止的容器
docker container prune
```

> **注意**：`docker rm` 删除容器，`docker rmi` 删除镜像，命令别搞混。

### 4. 进入容器内部

调试时经常需要进入容器查看文件或执行命令：

```sh
# docker exec -it <容器名> <Shell类型>
docker exec -it reservation-admin sh

# 如果容器内有 bash
docker exec -it reservation-admin bash

# 进入后可以执行常见命令
ls /app                    # 查看应用文件
cat /etc/resolv.conf       # 查看 DNS 配置
ping reservation-mysql     # 测试网络连通性
exit                       # 退出容器
```

> **说明**：本项目使用 Alpine Linux 基础镜像，默认只有 `sh`，没有 `bash`。

### 5. 在容器中执行单条命令（不进入交互模式）

```sh
# 查看容器内的环境变量
docker exec reservation-admin env

# 查看容器内的文件
docker exec reservation-admin ls /app/configs

# 测试容器间网络连通性
docker exec reservation-admin ping -c 3 reservation-mysql
```

---

## 五、查看日志

查看日志是排查问题最常用的手段：

```sh
# 查看全部日志
docker logs reservation-admin

# 实时跟踪最新日志（Ctrl + C 退出）
docker logs -f reservation-admin

# 查看最后 100 行
docker logs --tail 100 reservation-admin

# 实时跟踪 + 时间戳
docker logs -f -t --tail 50 reservation-admin

# 过滤特定关键词（如错误信息）
docker logs reservation-admin | grep "error"

# 同时查看多个容器的日志
docker logs reservation-admin 2>&1 | head -50
docker logs reservation-gateway 2>&1 | head -50
```

---

## 六、数据卷操作

数据卷用于持久化存储，即使容器被删除，数据卷中的数据依然保留。

```sh
# 列出所有数据卷
docker volume ls

# 查看数据卷详情（挂载路径等）
docker volume inspect reservation_sys_go_mysql_data

# 删除未使用的数据卷（危险！会丢失数据）
docker volume prune
```

**本项目的数据卷：**

| 数据卷名 | 用途 |
|----------|------|
| `mysql_data` | MySQL 数据持久化 |
| `redis_data` | Redis 数据持久化 |

> **注意**：如果需要重置数据库（清空所有数据），可以删除 `mysql_data` 数据卷后重新启动：
> ```sh
> docker-compose -f docker-compose.local.yaml down -v   # -v 会同时删除数据卷
> docker-compose -f docker-compose.local.yaml up -d --build
> ```

---

## 七、网络

Docker Compose 会自动创建网络，让同一项目下的容器可以通过服务名互相访问：

```sh
# 查看所有网络
docker network ls

# 查看网络详情（包含哪些容器）
docker network inspect reservation_sys_go_reservation-net
```

**本项目网络**：所有服务都在 `reservation-net` 网络中，容器间可通过服务名通信：
- `reservation-admin` 访问 Gateway：`gateway:8080`
- `reservation-admin` 访问 MySQL：`mysql:3306`
- `reservation-admin` 访问 Redis：`redis:6379`

> **关键点**：容器之间用 **Docker 服务名**（如 `mysql`）通信，不是用 `localhost` 或 `127.0.0.1`。

---

## 八、Docker Compose

Docker Compose 是管理多容器应用的工具，通过一个 YAML 文件定义所有服务。

### 1. 启动服务

```sh
# 本地开发：构建镜像 + 后台启动
docker-compose -f docker-compose.local.yaml up -d --build

# 参数说明：
#   -f <文件>     指定 compose 文件
#   up            创建并启动容器
#   -d            后台运行（detached mode）
#   --build       启动前重新构建镜像
```

### 2. 停止服务

```sh
# 停止并删除容器（保留数据卷）
docker-compose -f docker-compose.local.yaml down

# 停止并删除容器 + 数据卷（清空所有数据！）
docker-compose -f docker-compose.local.yaml down -v
```

### 3. 只重启某个服务

```sh
# 重新构建并重启单个服务（不需要全部重启）
docker-compose -f docker-compose.local.yaml up -d --build admin

# 只重启不重新构建
docker-compose -f docker-compose.local.yaml restart admin
```

### 4. 查看服务状态

```sh
# 查看所有服务的运行状态
docker-compose -f docker-compose.local.yaml ps

# 查看服务日志
docker-compose -f docker-compose.local.yaml logs admin

# 实时跟踪日志
docker-compose -f docker-compose.local.yaml logs -f admin
```

### 5. 进入服务容器

```sh
docker-compose -f docker-compose.local.yaml exec admin sh
```

### 6. 常用速查表

| 场景 | 命令 |
|------|------|
| 本地开发启动 | `docker-compose -f docker-compose.local.yaml up -d --build` |
| 测试环境部署 | `docker-compose -f docker-compose.test.yaml up -d --build` |
| 生产环境部署 | `docker-compose -f docker-compose.prod.yaml up -d` |
| 停止所有服务 | `docker-compose -f docker-compose.local.yaml down` |
| 重置数据库 | `docker-compose -f docker-compose.local.yaml down -v` |
| 只重启 admin | `docker-compose -f docker-compose.local.yaml up -d --build admin` |
| 查看 admin 日志 | `docker-compose -f docker-compose.local.yaml logs -f admin` |
| 进入 admin 容器 | `docker-compose -f docker-compose.local.yaml exec admin sh` |

---

## 九、日常开发流程

### 场景 1：首次拉取项目，启动本地环境

```sh
# 1. 启动所有服务
docker-compose -f docker-compose.local.yaml up -d --build

# 2. 查看所有服务是否正常运行
docker-compose -f docker-compose.local.yaml ps

# 3. 如果某个服务未启动，查看日志定位原因
docker-compose -f docker-compose.local.yaml logs <服务名>
```

### 场景 2：修改代码后重启服务

```sh
# 只重新构建并重启修改过的服务（如 admin）
docker-compose -f docker-compose.local.yaml up -d --build admin
```

### 场景 3：部署到测试服务器

```sh
# 方式一：在服务器上直接构建
docker-compose -f docker-compose.test.yaml up -d --build

# 方式二：本地构建 → 导出 → 传到服务器 → 加载 → 启动
# 1. 本地构建并导出
docker save -o reservation-sys-test.tar \
  reservation-gateway:test \
  reservation-reservation:test \
  reservation-admin:test

# 2. 上传到服务器
scp reservation-sys-test.tar user@server:/path/to/project/

# 3. 在服务器上加载镜像
docker load -i reservation-sys-test.tar

# 4. 启动服务（不需要 --build，因为镜像已存在）
docker-compose -f docker-compose.test.yaml up -d
```

### 场景 4：数据库变更，需要重置

```sh
# 停止服务并删除数据卷，然后重新启动
# 注意：这会清空所有数据！
docker-compose -f docker-compose.local.yaml down -v
docker-compose -f docker-compose.local.yaml up -d --build
```

---

## 十、问题排查

### 1. 服务启动失败

```sh
# 第一步：查看服务状态，确认哪个服务挂了
docker-compose -f docker-compose.local.yaml ps

# 第二步：查看失败服务的日志
docker-compose -f docker-compose.local.yaml logs <服务名>

# 第三步：如果日志不够详细，进入容器排查
docker exec -it <容器名> sh
```

### 2. 容器间网络不通

```sh
# 从 admin 容器 ping MySQL 容器
docker exec reservation-admin ping -c 3 reservation-mysql

# 从 admin 容器 ping Redis 容器
docker exec reservation-admin ping -c 3 reservation-redis

# 如果 ping 不通，检查是否在同一网络
docker network inspect reservation_sys_go_reservation-net
```

### 3. Gateway 连接 Redis 失败

```sh
# 1. 确认 Redis 容器是否健康
docker ps | grep redis

# 2. 从 Gateway 容器测试连接
docker exec reservation-gateway ping -c 3 reservation-redis

# 3. 检查配置文件中的 Redis 地址是否为服务名 `redis`，而非 `localhost`
```

### 4. MySQL 连接失败

```sh
# 1. 确认 MySQL 容器是否健康
docker ps | grep mysql

# 2. 检查 MySQL 是否完成初始化（首次启动较慢）
docker logs reservation-mysql 2>&1 | grep "ready for connections"

# 3. 从应用容器测试连接
docker exec reservation-admin ping -c 3 reservation-mysql
```

### 5. 端口被占用

```sh
# 查看端口占用
sudo lsof -i :80
# 或
sudo netstat -tlnp | grep :80

# 停止占用端口的程序，或修改 docker-compose 中的端口映射
```

### 6. 镜像构建失败

```sh
# 查看完整构建日志，不加 -d 可以看到实时输出
docker-compose -f docker-compose.local.yaml up --build

# 或单独构建某个镜像
docker build -t reservation-admin:test -f service/admin/Dockerfile . --no-cache
```

---

## 十一、常用命令速查

```sh
# ========== 镜像 ==========
docker images                           # 查看所有镜像
docker rmi <镜像名:标签>                 # 删除镜像
docker save -o <文件>.tar <镜像:标签>     # 导出镜像
docker load -i <文件>.tar                # 加载镜像

# ========== 容器 ==========
docker ps                               # 查看运行中的容器
docker ps -a                            # 查看所有容器
docker start <容器名>                    # 启动容器
docker stop <容器名>                     # 停止容器
docker restart <容器名>                  # 重启容器
docker rm <容器名>                      # 删除容器
docker exec -it <容器名> sh              # 进入容器

# ========== 日志 ==========
docker logs <容器名>                     # 查看日志
docker logs -f <容器名>                  # 实时跟踪日志
docker logs --tail 100 <容器名>          # 查看最后100行

# ========== 数据卷 ==========
docker volume ls                        # 查看数据卷
docker volume inspect <卷名>            # 查看数据卷详情

# ========== 网络 ==========
docker network ls                       # 查看网络
docker network inspect <网络名>          # 查看网络详情

# ========== Compose ==========
docker-compose -f <文件> up -d --build  # 构建并启动
docker-compose -f <文件> down           # 停止并删除
docker-compose -f <文件> down -v        # 停止并删除（含数据卷）
docker-compose -f <文件> ps             # 查看服务状态
docker-compose -f <文件> logs -f <服务>  # 查看服务日志
docker-compose -f <文件> exec <服务> sh  # 进入服务容器
docker-compose -f <文件> restart <服务>  # 重启服务
```
