#!/bin/bash
# 云服务器测试环境部署脚本

set -e

echo "=========================================="
echo "  预约系统 - 测试环境部署"
echo "=========================================="

# 配置
IMAGE_NAME="reservation-sys"
IMAGE_TAG="test"
TAR_FILE="reservation-sys-test.tar.gz"

# 步骤 1: 构建镜像
echo ""
echo "[1/4] 构建 Docker 镜像..."
docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .

# 步骤 2: 导出镜像
echo ""
echo "[2/4] 导出镜像到文件..."
docker save -o ${IMAGE_NAME}.tar ${IMAGE_NAME}:${IMAGE_TAG}

# 步骤 3: 打包部署文件
echo ""
echo "[3/4] 打包测试环境文件..."
tar -czf ${TAR_FILE} \
    ${IMAGE_NAME}.tar \
    sync_menu \
    docker-compose.test.yaml \
    configs/config_v1.test.yaml \
    configs/config_v2.test.yaml \
    deploy/ \
    internal/reservation/frontend/

# 清理临时文件
rm ${IMAGE_NAME}.tar sync_menu


# 步骤 6: 显示结果
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
echo "  ✓ 菜单同步工具 (sync_menu)"
echo "  ✓ 测试环境配置文件"
echo "  ✓ Nginx 配置"
echo "  ✓ 数据库初始化脚本"
echo "  ✓ 前端页面"
echo ""