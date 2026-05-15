#!/bin/bash

TAG="latest"

echo "安装前端依赖并构建..."
cd frontend && pnpm install && pnpm build:all && cd ..

echo "清理旧文件..."
rm -f reservation-*.tar reservation-sys-test.tar.gz reservation.tar admin.tar gateway.tar

echo "构建 gateway 镜像..."
docker build -t reservation-gateway:${TAG} -f service/gateway/Dockerfile .

echo "构建 reservation 镜像..."
docker build -t reservation-reservation:${TAG} -f service/reservation/Dockerfile .

echo "构建 admin 镜像..."
docker build -t reservation-admin:${TAG} -f service/admin/Dockerfile .

echo "导出镜像文件..."
docker save -o gateway.tar reservation-gateway:${TAG}
docker save -o reservation.tar reservation-reservation:${TAG}
docker save -o admin.tar reservation-admin:${TAG}

echo "打包部署文件..."
tar -czf reservation-sys.tar.gz \
    gateway.tar \
    reservation.tar \
    admin.tar \
    .env.example \
    docker-compose.prod.yaml \
    service/gateway/configs/config_v1.yaml \
    service/reservation/configs/config_v2.yaml \
    service/admin/configs/config_v3.yaml \
    deploy/ \
    dist/

echo "完成！输出: reservation-sys.tar.gz"
