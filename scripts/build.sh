#!/bin/bash
# 构建、导出、打包部署文件
# 用法: bash scripts/build.sh

set -euo pipefail

# ---- 配置（按需修改）----
VERSION="latest"
IMAGE_NAME="reservation-server"
# ------------------------

cd "$(dirname "$0")/.."
PKG="reservation-deploy-${VERSION}"
DIR="dist/build/${PKG}"

# 1. 构建前端
echo "[1/4] 构建前端..."
cd frontend
pnpm install --frozen-lockfile
pnpm build:all
cd ..

# 2. 构建镜像
echo "[2/4] docker build..."
docker build -t "${IMAGE_NAME}":"${VERSION}" .

# 3. 导出镜像
echo "[3/4] 导出镜像..."
mkdir -p "${DIR}/images"
docker save -o "${DIR}/images/${IMAGE_NAME}.tar" "${IMAGE_NAME}":"${VERSION}"

# 4. 打包
echo "[4/4] 打包..."
mkdir -p "${DIR}/configs" "${DIR}/deploy/mysql" "${DIR}/deploy/nginx" "${DIR}/dist"
cp docker-compose.prod.yaml "${DIR}/docker-compose.yaml"
cp configs/config.yaml "${DIR}/configs/"
cp deploy/mysql/init.sql "${DIR}/deploy/mysql/"
cp deploy/nginx/nginx.config "${DIR}/deploy/nginx/"        # 生产配置（含 HTTPS）
cp -r dist/reservation "${DIR}/dist/" 2>/dev/null || true
cp -r dist/admin "${DIR}/dist/" 2>/dev/null || true

mkdir -p dist
cd dist/build
tar czf "../../${PKG}.tar.gz" "${PKG}"
cd ../..
rm -rf "${DIR}"

echo ""
echo "完成: dist/${PKG}.tar.gz"
echo "部署: 上传→解压→编辑configs/config.yaml→docker load -i images/reservation-server.tar→docker compose up -d"
