#!/bin/bash
set -euo pipefail

GATEWAY_UI_PORT="{{gateway_ui_port}}"

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.hermes/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="set_hermes_ui"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"

# ========== stdout 契约：仅最终 JSON 走真 stdout ==========
exec 3>&1
exec >>"$LOG_FILE" 2>&1

echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "[set_hermes_ui] 准备开启 Hermes 面板，端口=${GATEWAY_UI_PORT}"

emit_result() {
    local json="$1"
    echo "[set_hermes_ui] 最终结果: ${json}"
    echo "${json}" >&3
}

# ========== 文件锁：flock -w 超时等待，避免并发冲突 ==========
# exec 200>&- 会在启动前主动释放锁，并发请求只需等前一个完成。-w 10 是 ~3x 余量。
LOCK_FILE="/tmp/.hermes_set_ui.lock"
exec 200>"$LOCK_FILE"
if ! flock -w 10 200; then
    echo "[set_hermes_ui] 等待文件锁超时（10s），前一个脚本未完成，放弃" >&2
    exit 1
fi
echo "[set_hermes_ui] 文件锁获取成功"

# ========== 幂等检查：端口已监听 Hermes 则直接返回 ==========
# 锁内检查避免竞态；通过响应体特征识别 Hermes，避免误判为其他 Web 服务。
HERMES_PROBE=$(curl -s --max-time 2 "http://127.0.0.1:${GATEWAY_UI_PORT}/" 2>/dev/null || true)
if echo "$HERMES_PROBE" | grep -qiE "hermes|<title>" 2>/dev/null; then
    echo "[set_hermes_ui] 端口 ${GATEWAY_UI_PORT} 已响应且识别为 Hermes 面板，跳过启动"
    emit_result "{\"port\":${GATEWAY_UI_PORT}}"
    exit 0
fi
if [ -n "$HERMES_PROBE" ]; then
    echo "[set_hermes_ui] 端口 ${GATEWAY_UI_PORT} 有响应但非 Hermes 面板，将重启"
else
    echo "[set_hermes_ui] 端口 ${GATEWAY_UI_PORT} 无响应，需要启动"
fi

# ========== 1. 查找 hermes 命令 ==========
HERMES_BIN=""

if command -v hermes >/dev/null 2>&1; then
    HERMES_BIN="$(command -v hermes)"
elif [ -x "$HOME/.local/bin/hermes" ]; then
    HERMES_BIN="$HOME/.local/bin/hermes"
elif [ -x "/usr/local/bin/hermes" ]; then
    HERMES_BIN="/usr/local/bin/hermes"
fi

if [ -z "$HERMES_BIN" ]; then
    echo "[set_hermes_ui] ERROR: hermes 命令未找到" >&2
    exit 1
fi
echo "[set_hermes_ui] 找到 hermes 命令: ${HERMES_BIN}"

# ========== 2. 停止已有的 hermes dashboard 进程 ==========
# 只匹配 "hermes dashboard --host"，避开 hermes-gateway 等常驻服务
#（它们的命令行是 hermes_cli.main gateway run，不含 "dashboard --host"）。
echo "[set_hermes_ui] 停止已有 hermes dashboard 进程..."
pkill -u "$(id -un)" -f "hermes dashboard --host" >/dev/null 2>&1 || true
sleep 1
# 兜底：如果目标端口仍被占用（且是本用户进程），强制释放
if command -v fuser >/dev/null 2>&1; then
    fuser -k -n tcp "${GATEWAY_UI_PORT}" >/dev/null 2>&1 || true
    sleep 1
fi
echo "[set_hermes_ui] 已停止已有进程（如有）"

# ========== 3. 启动 hermes dashboard ==========
mkdir -p "$HOME/.hermes/logs" 2>/dev/null || true
echo "[set_hermes_ui] 启动: ${HERMES_BIN} dashboard --host 0.0.0.0 --port ${GATEWAY_UI_PORT} --insecure"

# 释放文件锁 fd，避免被 setsid 子进程继承导致锁永久泄漏
exec 200>&-
echo "[set_hermes_ui] 文件锁已释放"

# PYTHONUNBUFFERED=1 去除 Python stdout 缓冲，实时写入日志便于排障。
PYTHONUNBUFFERED=1 setsid "$HERMES_BIN" dashboard --host 0.0.0.0 --port "$GATEWAY_UI_PORT" --insecure </dev/null >>"$HOME/.hermes/logs/hermes_dashboard.log" 2>&1 &

# ========== 4. 验证启动成功：端口 listen + HTTP 响应 ==========
# 判据：端口 listen 且 HTTP 响应即成功。
# 自适应超时：检测到 "Building web UI" 则窗口 20s→180s，否则 20s 快速失败。
DASHBOARD_LOG="$HOME/.hermes/logs/hermes_dashboard.log"
MAX_WAIT=20
BUILDING_DETECTED=""
STARTED=""
LAST_STAGE=""  # 记录最后卡在哪一步：port_not_listen / http_no_response
i=0
while [ $i -lt $MAX_WAIT ]; do
    i=$((i+1))
    sleep 1
    # 检测到首次启动的前端构建标记 → 延长窗口到 180s
    if [ -z "$BUILDING_DETECTED" ] && [ -f "$DASHBOARD_LOG" ] \
       && grep -qiE "Building web UI|building.*web|npm.*install|webpack" "$DASHBOARD_LOG" 2>/dev/null; then
        BUILDING_DETECTED="1"
        MAX_WAIT=180
        echo "[set_hermes_ui] 检测到首次启动前端构建（Building web UI），延长等待窗口至 ${MAX_WAIT}s"
    fi
    # 每 10s 打一条 heartbeat（构建期间用户可见进度）
    if [ $((i % 10)) -eq 0 ]; then
        echo "[set_hermes_ui] 探活中... 已等待 ${i}s / ${MAX_WAIT}s${BUILDING_DETECTED:+ (前端构建中)}"
    fi
    # 端口 listen 检查（bash 内建 /dev/tcp，无需外部命令）
    if ! (echo >/dev/tcp/127.0.0.1/${GATEWAY_UI_PORT}) 2>/dev/null; then
        LAST_STAGE="port_not_listen"
        continue
    fi
    # HTTP 响应检查
    if curl -s -o /dev/null --max-time 2 "http://127.0.0.1:${GATEWAY_UI_PORT}/" 2>/dev/null; then
        STARTED="1"
        echo "[set_hermes_ui] 启动就绪（第 ${i}s）"
        break
    fi
    LAST_STAGE="http_no_response"
done

if [ -z "$STARTED" ]; then
    echo "[set_hermes_ui] ERROR: hermes dashboard 启动后 ${MAX_WAIT}s 内未能提供服务，卡在阶段: ${LAST_STAGE}"
    echo ""
    echo "--- 进程快照 (ps -ef | grep hermes) ---"
    ps -ef 2>/dev/null | grep -F "hermes" | grep -v grep || echo "(无匹配进程)"
    echo ""
    echo "--- 端口快照 (端口 ${GATEWAY_UI_PORT}) ---"
    if command -v ss >/dev/null 2>&1; then
        ss -tlnp 2>/dev/null | grep ":${GATEWAY_UI_PORT}" || echo "(端口未 listen)"
    elif command -v netstat >/dev/null 2>&1; then
        netstat -tlnp 2>/dev/null | grep ":${GATEWAY_UI_PORT}" || echo "(端口未 listen)"
    fi
    echo ""
    echo "--- hermes dashboard stderr/stdout (最后 50 行) ---"
    if [ -f "$HOME/.hermes/logs/hermes_dashboard.log" ]; then
        tail -n 50 "$HOME/.hermes/logs/hermes_dashboard.log" 2>/dev/null || true
    else
        echo "(dashboard 日志文件不存在)"
    fi
    echo "--- hermes dashboard 日志结束 ---"
    exit 1
fi
# 用端口占用者查 PID（比 pgrep 匹配命令行更可靠）
DASHBOARD_PID=""
if command -v ss >/dev/null 2>&1; then
    DASHBOARD_PID=$(ss -tlnp 2>/dev/null | grep ":${GATEWAY_UI_PORT}" | grep -oE 'pid=[0-9]+' | head -1 | cut -d= -f2)
fi
echo "[set_hermes_ui] Hermes Dashboard 启动成功，端口=${GATEWAY_UI_PORT}${DASHBOARD_PID:+, PID=${DASHBOARD_PID}}"

emit_result "{\"port\":${GATEWAY_UI_PORT}}"