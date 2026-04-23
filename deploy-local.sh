#!/bin/bash

set -e

IMAGE_NAME="reservation-sys"
IMAGE_TAG="test"

# 构建镜像
echo "构建 Docker 镜像..."
docker build -t ${IMAGE_NAME}:${IMAGE_TAG}

# 进行部署
echo "进行部署..."
docker-compose -f docker-compose.${IMAGE_TAG}.yaml up -d