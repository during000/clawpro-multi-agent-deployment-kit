#!/bin/bash
set -uo pipefail

# ========== OpenClaw Gateway WebSocket 健康检查脚本 ==========
# 功能：验证 Gateway 能否正常接受 WebSocket 连接。
# 与 check_openclaw_ready.sh 的区别：
#   - check_openclaw_ready.sh 检查 HTTP 层面端口是否响应（进程是否启动）
#   - 本脚本检查 WebSocket 层面是否真正可用（进程是否完全初始化）
#
# 背景：Gateway 启动分阶段：先 bind 端口（HTTP 可达），再初始化内部组件，
# 最后才能正常处理 WS 连接。如果只检查 HTTP 层面就认为就绪，
# 前端/preflight 尝试 WS 连接时可能收到 1006 abnormal closure。
#
# 输出：最后一行为 JSON，格式：
#   {"healthy": true}
#   {"healthy": false, "reason": "<原因>"}

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="check_gateway_ws_health"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"

# stdout 契约：仅最终 JSON 走真 stdout（fd 3）
exec 3>&1
exec >>"$LOG_FILE" 2>&1

echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

emit_result() {
    echo "$1"
    echo "$1" >&3
}

# ========== 加载 openclaw 运行环境 ==========
export PNPM_HOME="${PNPM_HOME:-$HOME/.local/share/pnpm}"
export PATH="$PNPM_HOME:$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"

CONFIG="$HOME/.openclaw/openclaw.json"

echo "=== Gateway WebSocket 健康检查 ==="

# ========== 前置检查：配置文件 ==========
if [ ! -f "$CONFIG" ]; then
    echo "✗ 配置文件不存在: $CONFIG"
    emit_result '{"healthy": false, "reason": "config file not found"}'
    exit 0
fi

# ========== 方式 1（首选）：openclaw gateway call status ==========
# `gateway call status` 是最轻量的 RPC：底层会建立 WebSocket 连接到 gateway，
# 然后调用 status 方法。命令成功（exit code 0）说明：
#   1. WebSocket 握手成功；
#   2. 鉴权通过；
#   3. gateway 已经可以处理 RPC 请求。
#
# 不使用 `gateway health`：health 命令会探测所有 channel，对 doctor 节点
# 这种短期实例既不必要也容易干扰（某个 channel unhealthy 不应阻塞 WS
# 健康判断）。我们只关心 WS 是否可达。
#
# 不使用 grep "healthy" 文本匹配：history bug —— "unhealthy" 也包含
# "healthy" 子串导致假阳性；且 health 命令在某些 channel 不健康时仍可能
# 返回 exit code 0，整体语义不清晰。
if command -v openclaw &>/dev/null; then
    echo "使用 openclaw gateway call status 进行 WS 探测"
    # 用 timeout 兜底，避免 RPC hang 死把整个脚本拖到 TAT 超时
    if STATUS_OUT=$(timeout 10 openclaw gateway call status --json --params '{}' 2>&1); then
        echo "✓ Gateway WebSocket 连接正常 (gateway call status 成功)"
        echo "status 输出: ${STATUS_OUT}"
        emit_result '{"healthy": true}'
        exit 0
    fi
    REASON=$(echo "${STATUS_OUT:-}" | head -3 | tr '\n' ' ' | sed 's/"/\\"/g')
    echo "✗ Gateway WebSocket 连接异常: ${REASON}"
    emit_result "{\"healthy\": false, \"reason\": \"gateway call status failed: ${REASON}\"}"
    exit 0
fi

# ========== 方式 2（兜底）：curl WebSocket 升级探测 ==========
# 仅在 openclaw 命令不可用时触发（理论上 doctor 节点必装 openclaw）。
echo "openclaw 命令不可用，尝试 curl WebSocket 升级探测"

# 获取端口
SERVICE_FILE="$HOME/.config/systemd/user/openclaw-gateway.service"
SERVICE_PORT=""
if [ -f "$SERVICE_FILE" ]; then
    SERVICE_PORT=$(grep -oP '(?<=--port )\d+' "$SERVICE_FILE" 2>/dev/null | head -1 || true)
fi
CONFIG_PORT=$(jq -r '.gateway.port // empty' "$CONFIG" 2>/dev/null || true)

if [ -n "$SERVICE_PORT" ]; then
    PORT="$SERVICE_PORT"
elif [ -n "$CONFIG_PORT" ]; then
    PORT="$CONFIG_PORT"
else
    echo "✗ 无法获取 gateway 端口"
    emit_result '{"healthy": false, "reason": "port not configured"}'
    exit 0
fi

# 读取 auth token
AUTH_TOKEN=$(jq -r '.gateway.auth.token // empty' "$CONFIG" 2>/dev/null || true)
if [ -z "$AUTH_TOKEN" ] || [ "$AUTH_TOKEN" = "null" ]; then
    # token 缺失 = gateway 配置尚未完整初始化，应判为 unhealthy 让外层重试。
    # 旧版本默认放行（视为健康）会导致后续真正建 WS 时失败。
    echo "✗ 无法读取 gateway auth token，gateway 配置可能未初始化完成"
    emit_result '{"healthy": false, "reason": "gateway auth token missing"}'
    exit 0
fi

# 发送 WebSocket 升级请求，检查是否收到 101
WS_RESP=$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 \
    -H "Upgrade: websocket" \
    -H "Connection: Upgrade" \
    -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
    -H "Sec-WebSocket-Version: 13" \
    "http://localhost:${PORT}/ws?token=${AUTH_TOKEN}" 2>/dev/null || true)
echo "WebSocket 升级探测 HTTP 状态码: ${WS_RESP:-无响应}"

# 101 = 升级成功，说明 WS 可用
if [ "$WS_RESP" = "101" ]; then
    echo "✓ Gateway WebSocket 升级成功 (HTTP 101)"
    emit_result '{"healthy": true}'
elif [ -n "$WS_RESP" ] && [ "$WS_RESP" != "000" ]; then
    echo "✗ Gateway WebSocket 升级失败 (HTTP $WS_RESP)"
    emit_result "{\"healthy\": false, \"reason\": \"ws upgrade returned HTTP ${WS_RESP}\"}"
else
    echo "✗ Gateway WebSocket 探测无响应"
    emit_result '{"healthy": false, "reason": "ws probe no response"}'
fi
exit 0
