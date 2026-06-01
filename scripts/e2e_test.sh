#!/usr/bin/env bash
# E2E 集成测试脚本
# 职责: 管理 docker-compose 生命周期（启动 → 等待就绪 → 运行测试 → 清理）
#
# 用法:
#   bash scripts/e2e_test.sh              # 运行所有 E2E 测试
#   bash scripts/e2e_test.sh -v           # 详细输出
#   bash scripts/e2e_test.sh -run TestXxx # 运行指定测试
#
# 依赖: docker, go

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
COMPOSE_FILE="docker-compose.e2e.yaml"

cd "$PROJECT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[e2e]${NC} $*"; }
warn() { echo -e "${YELLOW}[e2e]${NC} $*"; }
err()  { echo -e "${RED}[e2e]${NC} $*"; }

cleanup() {
    log "清理 docker-compose 环境..."
    docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# 检查依赖
for cmd in docker go curl; do
    if ! command -v "$cmd" &>/dev/null; then
        err "未找到 $cmd，请先安装"
        exit 1
    fi
done

# 清理残留
log "清理残留容器..."
docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true

# 构建并启动
log "构建镜像并启动服务栈..."
if ! docker compose -f "$COMPOSE_FILE" up -d --build 2>&1; then
    err "docker-compose 启动失败"
    exit 1
fi

# 等待 MySQL 就绪
log "等待 MySQL 就绪..."
for i in $(seq 1 30); do
    if docker compose -f "$COMPOSE_FILE" exec -T mysql mysqladmin ping -h localhost --silent 2>/dev/null; then
        log "MySQL 就绪"
        break
    fi
    if [ "$i" -eq 30 ]; then
        err "MySQL 启动超时"
        docker compose -f "$COMPOSE_FILE" logs mysql --tail 20
        exit 1
    fi
    sleep 2
done

# 等待 nginx 就绪（nginx 依赖所有后端服务已启动）
log "等待 nginx 就绪（最多 60 秒）..."
DEADLINE=$(($(date +%s) + 60))
while [ $(date +%s) -lt $DEADLINE ]; do
    if curl -s -o /dev/null -w "%{http_code}" http://localhost/health 2>/dev/null | grep -q "200"; then
        log "nginx 就绪"
        break
    fi
    sleep 3
done

if [ $(date +%s) -ge $DEADLINE ]; then
    err "nginx 启动超时"
    docker compose -f "$COMPOSE_FILE" ps
    docker compose -f "$COMPOSE_FILE" logs nginx --tail 30 2>/dev/null || true
    exit 1
fi

# 等待后端服务就绪（nginx 返回 502 说明上游尚未准备好）
log "等待后端服务就绪（最多 90 秒）..."
DEADLINE=$(($(date +%s) + 90))
while [ $(date +%s) -lt $DEADLINE ]; do
    # 检查 reservation 读接口（不限流、无需 token）
    RESV_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost/api/reservation/reservation/occupied" 2>/dev/null)
    # 检查 admin 登录接口（会返回 4xx 表示服务可达，返回 502 表示不可达）
    ADMIN_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost/api/admin/auth/login" -X POST -H "Content-Type: application/json" -d '{}' 2>/dev/null)
    if [ "$RESV_CODE" != "502" ] && [ "$RESV_CODE" != "000" ] && [ "$ADMIN_CODE" != "502" ] && [ "$ADMIN_CODE" != "000" ]; then
        log "后端服务就绪 (reservation=$RESV_CODE, admin=$ADMIN_CODE)"
        break
    fi
    warn "等待后端服务... (reservation=$RESV_CODE, admin=$ADMIN_CODE)"
    sleep 3
done

if [ $(date +%s) -ge $DEADLINE ]; then
    err "后端服务启动超时"
    docker compose -f "$COMPOSE_FILE" ps
    docker compose -f "$COMPOSE_FILE" logs admin --tail 20 2>/dev/null || true
    docker compose -f "$COMPOSE_FILE" logs reservation --tail 20 2>/dev/null || true
    exit 1
fi

# 运行 E2E 测试（Go 测试代码只负责连接和测试，不管理容器）
log "运行 E2E 测试..."
TEST_ARGS="${*:--v -count=1}"
if go test ./tests/integration/... $TEST_ARGS; then
    log "E2E 测试全部通过"
else
    err "E2E 测试失败"
    exit 1
fi
