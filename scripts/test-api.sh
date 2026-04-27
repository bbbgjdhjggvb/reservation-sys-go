#!/bin/bash
# API 测试脚本 — 发送测试请求验证各接口
# 需要先通过 env.sh 准好环境和 Token 后再执行
# 使用方法: bash scripts/test-api.sh

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEST_DATA_DIR="$PROJECT_ROOT/.test-data"
BASE_URL="http://localhost:8081"
LOG_FILE="$TEST_DATA_DIR/test-result.log"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

info() { echo -e "${BLUE}[INFO]${NC} $1"; log "INFO  $1"; }
ok()   { echo -e "${GREEN}[OK]${NC} $1";   log "OK    $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; log "WARN  $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; log "FAIL  $1"; exit 1; }

# 写入日志文件
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

# 记录请求和响应到日志
log_request() {
    local method="$1"
    local url="$2"
    local body="${3:-}"
    log "------ REQUEST ------"
    log "$method $url"
    [ -n "$body" ] && log "Body: $body"
}

log_response() {
    local resp="$1"
    log "------ RESPONSE -----"
    log "$resp"
    log "---------------------"
}

# ============ 加载 Token ============
load_token() {
    if [ ! -f "$TEST_DATA_DIR/token" ]; then
        fail "Token 文件不存在，请先执行: bash scripts/env.sh token"
    fi
    TOKEN=$(cat "$TEST_DATA_DIR/token")
    ok "已加载 Token: ${TOKEN:0:40}..."
}

# ============ 检查服务可用性 ============
check_service() {
    if ! curl -s "$BASE_URL/" > /dev/null 2>&1; then
        fail "v2 服务未运行 ($BASE_URL)，请先执行: bash scripts/env.sh serve"
    fi
    ok "v2 服务可达 ($BASE_URL)"
}

# ============ 测试用例 ============

test_submit_reservation() {
    echo ""
    info "1. 提交预约 (POST /api/v2/reservation/submit) — 多时段格式"

    local body='{"applicant_name":"测试用户","alumni_association":"计算机与软件学院校友会","year":2020,"major":"软件工程","reason":"校友技术分享会本地测试","phone":"13800138000","slots":[{"start_time":"2026-05-01 14:00:00","end_time":"2026-05-01 16:00:00"},{"start_time":"2026-05-01 16:30:00","end_time":"2026-05-01 18:30:00"}]}'
    local url="$BASE_URL/api/v2/reservation/submit"
    log_request "POST" "$url" "$body"

    local resp
    resp=$(curl -s -X POST "$url" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "$body")

    echo "  响应: $resp"
    log_response "$resp"

    if echo "$resp" | grep -q '"code":200'; then
        ok "提交预约成功（多时段）"
        local res_id
        res_id=$(echo "$resp" | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')
        [ -n "$res_id" ] && ok "订单 ID: $res_id" && echo "$res_id" > "$TEST_DATA_DIR/reservation_id"
    else
        warn "时间段可能已被占用，换一个时间重试..."
        local body2='{"applicant_name":"测试用户","alumni_association":"计算机与软件学院校友会","year":2020,"major":"软件工程","reason":"校友技术分享会本地测试(重试)","phone":"13800138000","slots":[{"start_time":"2026-06-15 09:00:00","end_time":"2026-06-15 11:00:00"}]}'
        log_request "POST" "$url" "$body2"
        resp=$(curl -s -X POST "$url" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            -d "$body2")
        echo "  重试响应: $resp"
        log_response "$resp"
        if echo "$resp" | grep -q '"code":200'; then
            ok "提交预约成功（重试）"
            local res_id
            res_id=$(echo "$resp" | grep -o '"id":[0-9]*' | head -1 | grep -o '[0-9]*')
            [ -n "$res_id" ] && ok "订单 ID: $res_id" && echo "$res_id" > "$TEST_DATA_DIR/reservation_id"
        else
            warn "提交预约失败，查看日志: tail -f $TEST_DATA_DIR/v2.log"
        fi
    fi
}

test_get_my_reservations() {
    echo ""
    info "2. 查询我的预约 (GET /api/v2/reservation/my)"
    local url="$BASE_URL/api/v2/reservation/my"
    log_request "GET" "$url"

    local resp
    resp=$(curl -s "$url" -H "Authorization: Bearer $TOKEN")
    echo "  响应: $resp"
    log_response "$resp"

    if echo "$resp" | grep -q '"code":200'; then ok "查询我的预约成功"; else warn "查询我的预约失败"; fi
}

test_get_occupied_slots() {
    echo ""
    info "3. 查询已占用时段 (GET /api/v2/reservation/occupied?date=2026-05-01)"
    local url="$BASE_URL/api/v2/reservation/occupied?date=2026-05-01"
    log_request "GET" "$url"

    local resp
    resp=$(curl -s "$url" -H "Authorization: Bearer $TOKEN")
    echo "  响应: $resp"
    log_response "$resp"

    if echo "$resp" | grep -q '"code":200'; then ok "查询已占用时段成功"; else warn "查询已占用时段失败"; fi
}

test_cancel_reservation() {
    echo ""
    local res_id
    res_id=$(cat "$TEST_DATA_DIR/reservation_id" 2>/dev/null || echo "")
    if [ -z "$res_id" ]; then
        warn "4. 跳过取消预约（无有效预约 ID）"
        log "SKIP  4. 取消预约 (无有效预约 ID)"
        return
    fi

    info "4. 取消预约 (DELETE /api/v2/reservation/$res_id)"
    local url="$BASE_URL/api/v2/reservation/$res_id"
    log_request "DELETE" "$url"

    local resp
    resp=$(curl -s -X DELETE "$url" -H "Authorization: Bearer $TOKEN")
    echo "  响应: $resp"
    log_response "$resp"

    if echo "$resp" | grep -q '"code":200'; then ok "取消预约成功"; else warn "取消预约失败"; fi
}

test_unauthorized_access() {
    echo ""
    info "5. 未授权测试 (无 Token 请求)"
    local url="$BASE_URL/api/v2/reservation/my"
    log_request "GET" "$url" "(no token)"

    local resp
    resp=$(curl -s "$url")
    echo "  响应: $resp"
    log_response "$resp"

    if echo "$resp" | grep -q '401\|未授权\|unauthorized'; then ok "未授权拦截正常"; else warn "未授权拦截可能异常"; fi
}

test_invalid_params() {
    echo ""
    info "6. 参数错误测试 (缺少必填字段)"
    local body='{"applicant_name": "缺字段测试"}'
    local url="$BASE_URL/api/v2/reservation/submit"
    log_request "POST" "$url" "$body"

    local resp
    resp=$(curl -s -X POST "$url" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "$body")
    echo "  响应: $resp"
    log_response "$resp"

    if echo "$resp" | grep -q '400\|参数\|required\|Key'; then ok "参数校验正常拦截"; else warn "参数校验可能异常"; fi
}

# ============ 数据库验证 ============

verify_database() {
    echo ""
    echo "---"
    info "验证数据库记录"
    echo ""

    local db_result
    info "订单表最近 5 条:"
    db_result=$(docker exec reservation-mysql mysql -ures_user -p12345678 --default-character-set=utf8mb4 home_xy \
        -e "SELECT id, order_no, applicant_name, total_slots, status FROM reservation_orders ORDER BY id DESC LIMIT 5;" 2>/dev/null)
    echo "$db_result"
    log "DB reservation_orders: $db_result"

    echo ""
    info "时段明细表最近 5 条:"
    db_result=$(docker exec reservation-mysql mysql -ures_user -p12345678 --default-character-set=utf8mb4 home_xy \
        -e "SELECT id, order_id, start_time, end_time, status FROM reservation_slots ORDER BY id DESC LIMIT 5;" 2>/dev/null)
    echo "$db_result"
    log "DB reservation_slots: $db_result"

    echo ""
    info "用户表:"
    db_result=$(docker exec reservation-mysql mysql -ures_user -p12345678 --default-character-set=utf8mb4 home_xy \
        -e "SELECT id, openid, created_at FROM users LIMIT 5;" 2>/dev/null)
    echo "$db_result"
    log "DB users: $db_result"

    ok "数据库验证完成 ✅"
}

# ============ 主流程 ============
main() {
    mkdir -p "$TEST_DATA_DIR"

    # 初始化日志文件
    echo "" > "$LOG_FILE"
    log "============================"
    log "  API 测试开始"
    log "============================"

    echo "=========================================="
    echo "  API 接口测试"
    echo "=========================================="
    echo "  日志文件: $LOG_FILE"
    echo "=========================================="

    check_service
    load_token

    echo ""
    echo "=========================================="
    echo "  开始发送测试请求"
    echo "=========================================="

    test_submit_reservation
    test_get_my_reservations
    test_get_occupied_slots
    test_cancel_reservation
    test_unauthorized_access
    test_invalid_params

    echo ""
    echo "=========================================="
    ok "全部测试用例执行完毕"
    echo "  详细日志: $LOG_FILE"
    echo "=========================================="

    verify_database

    log "============================"
    log "  API 测试结束"
    log "============================"
}

main "$@"
