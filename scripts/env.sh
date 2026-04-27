#!/bin/bash
# 环境准备脚本 — 基础设施启停、服务管理、Token 生成、清理
# 使用方法: bash scripts/env.sh <command>
#   up       启动 MySQL + Redis (Docker)
#   down     停止 Docker 容器并清理临时文件
#   token    生成 JWT Token（保存到 .test-data/token）
#   serve    启动 v2 服务（后台运行）
#   stop     停止 v2 服务

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CONFIG_DEBUG="$PROJECT_ROOT/configs/config_v2.debug.yaml"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.local.yaml"
BASE_URL="http://localhost:8081"
TEST_DATA_DIR="$PROJECT_ROOT/.test-data"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
ok()   { echo -e "${GREEN}[OK]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }

# ============ 工具函数 ============

ensure_test_dir() {
    mkdir -p "$TEST_DATA_DIR"
}

check_prerequisites() {
    if ! command -v docker &> /dev/null; then
        fail "Docker 未安装或不在 PATH 中，请先安装 Docker"
    fi
    if ! docker info &> /dev/null; then
        fail "Docker 服务未运行，请先启动 Docker Desktop 或 Docker daemon"
    fi
    ok "前置检查通过 (Docker 已就绪)"
}

# ============ 命令: up — 启动基础设施 ============
cmd_up() {
    check_prerequisites
    ensure_test_dir

    info "启动 MySQL + Redis 容器..."
    docker-compose -f "$COMPOSE_FILE" up -d mysql redis

    info "等待 MySQL 就绪..."
    for i in $(seq 1 30); do
        if docker exec reservation-mysql mysqladmin ping -h localhost --silent 2>/dev/null; then
            ok "MySQL 已就绪"
            break
        fi
        [ "$i" -eq 30 ] && fail "MySQL 启动超时"
        sleep 2
    done

    info "等待 Redis 就绪..."
    for i in $(seq 1 15); do
        if docker exec reservation-redis redis-cli ping 2>/dev/null | grep -q PONG; then
            ok "Redis 已就绪"
            break
        fi
        [ "$i" -eq 15 ] && fail "Redis 启动超时"
        sleep 1
    done

    info "验证数据库表结构..."
    docker exec reservation-mysql mysql -ures_user -p12345678 --default-character-set=utf8mb4 home_xy \
        -e "SHOW TABLES;" 2>/dev/null && ok "数据库表创建成功"

    ok "环境准备完成 ✅"
}

# ============ 命令: down — 停止并清理 ============
cmd_down() {
    # 停止 v2 服务
    if [ -f "$TEST_DATA_DIR/v2.pid" ]; then
        info "停止 v2 服务 (PID: $(cat "$TEST_DATA_DIR/v2.pid"))..."
        kill "$(cat "$TEST_DATA_DIR/v2.pid")" 2>/dev/null || true
        rm -f "$TEST_DATA_DIR/v2.pid"
    fi

    info "停止 Docker 容器..."
    docker-compose -f "$COMPOSE_FILE" down 2>/dev/null

    ensure_test_dir
    rm -f "$TEST_DATA_DIR/token" "$TEST_DATA_DIR/reservation_id" "$TEST_DATA_DIR/v2.log"

    ok "清理完成 ✅"
}

# ============ 命令: token — 生成 JWT Token ============
cmd_token() {
    ensure_test_dir
    info "生成 JWT Token..."

    local raw_output
    raw_output=$(cd "$PROJECT_ROOT" && go run cmd/tools/jwt/gen_token.go 2>/dev/null)

    local token
    token=$(echo "$raw_output" | grep -oE 'eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+' | head -1)
    [ -z "$token" ] && token=$(echo "$raw_output" | grep -oE 'eyJ.*' | head -1)
    if [ -z "$token" ]; then
        fail "Token 生成失败，原始输出: $(echo "$raw_output" | head -5)"
    fi

    echo "$token" > "$TEST_DATA_DIR/token"
    ok "Token: ${token:0:40}... → $TEST_DATA_DIR/token"
    echo ""
    echo "$token"
}

# ============ 命令: serve — 启动 v2 服务 ============
cmd_serve() {
    ensure_test_dir

    info "检查 v2 服务是否已启动..."
    if curl -s "$BASE_URL/" > /dev/null 2>&1; then
        ok "v2 服务已在运行 ($BASE_URL)"
        return
    fi

    info "启动 v2 服务 (后台运行)..."
    cd "$PROJECT_ROOT"
    CONFIG_PATH="$CONFIG_DEBUG" nohup go run cmd/api/v2/main.go > "$TEST_DATA_DIR/v2.log" 2>&1 &
    local PID=$!
    echo "$PID" > "$TEST_DATA_DIR/v2.pid"
    info "服务日志: $TEST_DATA_DIR/v2.log (PID: $PID)"

    info "等待服务启动..."
    for i in $(seq 1 30); do
        if curl -s "$BASE_URL/" > /dev/null 2>&1; then
            ok "v2 服务启动成功 (PID: $PID) → $BASE_URL"
            return
        fi
        sleep 1
    done
    warn "v2 服务启动超时！查看日志: tail -f $TEST_DATA_DIR/v2.log"
    exit 1
}

# ============ 命令: stop — 仅停止 v2 服务 ============
cmd_stop() {
    if [ -f "$TEST_DATA_DIR/v2.pid" ]; then
        info "停止 v2 服务 (PID: $(cat "$TEST_DATA_DIR/v2.pid"))..."
        kill "$(cat "$TEST_DATA_DIR/v2.pid")" 2>/dev/null || true
        rm -f "$TEST_DATA_DIR/v2.pid"
        ok "v2 服务已停止"
    else
        warn "未找到运行的 v2 服务 PID 文件"
    fi
}

# ============ 主入口 ============
case "${1:-}" in
    up)      cmd_up ;;
    down)    cmd_down ;;
    token)   cmd_token ;;
    serve)   cmd_serve ;;
    stop)    cmd_stop ;;
    *)
        echo "环境准备脚本"
        echo ""
        echo "使用方法: bash scripts/env.sh <command>"
        echo ""
        echo "命令:"
        echo "  up       启动 MySQL + Redis (Docker)"
        echo "  down     停止容器 + 服务 + 清理临时文件"
        echo "  token    生成 JWT Token → .test-data/token"
        echo "  serve    启动 v2 服务 (后台，日志在 .test-data/v2.log)"
        echo "  stop     仅停止 v2 服务"
        ;;
esac
