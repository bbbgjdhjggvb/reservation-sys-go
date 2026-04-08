# docker
## docker 镜像
```
# 查看镜像
docker images

# 打包镜像
docker save -o <image_name>.tar:<tag>
docker save -o reservation-sys.tar:latest

# 加载镜像
docker load -i <image_name>.tar
docker load -i reservation-sys.tar
```
## docker 容器
```
docker ps 
docker ps -a # 包括停止了对容器
```

## docker 数据卷
```
docker volume ls 
```

# docker-compose
什么是 docker-compose
一个文件解决多个容器配置，编排，统一部署，管理多个容器

```
# 启动服务
docker-compose  -f docker-compose.yaml up -d --build
# -d 参数，表示 detached ，后台运行
# --build 参数，表示构建镜像

# 停止服务
docker-compose -f docker-compose.yaml down
```

# docker 问题排查
## auth 模块 redis 初始化失败
1. 从 auth 容器中 ping redis 容器