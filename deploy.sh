#!/bin/bash
# 云服务器部署脚本

set -e

echo "=========================================="
echo "  预约系统 - 云服务器部署工具"
echo "=========================================="

# 配置
IMAGE_NAME="reservation-sys"
IMAGE_TAG="latest"
TAR_FILE="reservation-sys-deploy.tar.gz"

# 步骤 1: 构建镜像
echo ""
echo "[1/4] 构建 Docker 镜像..."
docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .

# 步骤 2: 导出镜像
echo ""
echo "[2/4] 导出镜像到文件..."
docker save -o ${IMAGE_NAME}.tar ${IMAGE_NAME}:${IMAGE_TAG}

# 步骤 4: 打包部署文件
echo ""
echo "[3/4] 打包部署文件..."
tar -czf ${TAR_FILE} \
    ${IMAGE_NAME}.tar \
    sync_menu \
    docker-compose.prod.yaml \
    .env.example \
    configs/ \
    deploy/ \
    internal/reservation/frontend/ \
    --exclude='*.log' \
    --exclude='configs/*.local.yaml'

# 清理临时文件
rm ${IMAGE_NAME}.tar sync_menu

# 步骤 6: 显示上传指引
echo ""
echo "[4/4] 打包完成！"
echo ""
echo "=========================================="
echo "  部署包已创建"
echo "=========================================="
echo ""
echo "📦 文件: ${TAR_FILE}"
echo "📏 大小: $(du -h ${TAR_FILE} | cut -f1)"
echo ""
echo "📋 包含文件:"
echo "  ✓ Docker 镜像"
echo "  ✓ 生产环境配置"
echo "  ✓ 部署配置文件"
echo "  ✓ 前端页面"
echo ""
echo "📖 部署说明已保存到: DEPLOY_PROD_README.txt"
echo ""
echo ""
echo "1. 上传部署包到云服务器："
echo "   scp ${TAR_FILE} user@your-server:/path/to/deploy/"
echo ""
echo "2. 在云服务器上解压："
echo "   tar -xzf ${TAR_FILE}"
echo ""
echo "3. 配置环境变量："
echo "   cp .env.example .env"
echo "   vim .env  # 修改配置"
echo ""
echo "4. 更新微信配置："
echo "   vim configs/config_v1.yaml  # 修改 frontend_url"
echo ""
echo "5. 启动服务："
echo "   docker-compose -f docker-compose.prod.yaml up -d"
echo ""
