#!/bin/bash
set -uo pipefail

# ========== OpenClaw 就绪状态检查脚本 ==========
# 功能：检查 openclaw 是否已安装完成并服务就绪
# 检查两个条件（均满足才认为就绪）：
#   1. ~/.openclaw/openclaw.json 文件存在
#   2. openclaw gateway 端口正常响应
#
# 输出：最后一行为 JSON，格式：
#   {"ready": true}
#   {"ready": false, "reason": "<原因>"}

# ========== 日志系统初始化 ==========
# 使用用户家目录，避免非 root 运行时无法写入 /var/log
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="check_openclaw_ready"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"

# ========== stdout 契约：仅最终 JSON 走真 stdout ==========
# 历史 bug（工蜂 CR 提出）：原写法 `exec > >(tee -a "$LOG_FILE") 2>&1`
# 把所有调试 echo 也打到 stdout，与最后一行 JSON 混染，导致
# hatchery/controller/openclaw.go:1953 `json.Unmarshal([]byte(output), ...)`
# 永远解析失败 → `HandleCheckAgentReady` 始终返回 `{"running":false}`。
#
# 新契约：
#   - fd 3 = 真正的 stdout（调用方 TAT 捕获目标）。exec 3>&1 先把它保存起来。
#   - 脚本主体的 stdout/stderr 全部追加进 LOG_FILE。
#   - 终态 JSON 专门通过 emit_result 写到 fd 3。
exec 3>&1
exec >>"$LOG_FILE" 2>&1

echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# emit_result：仅用于终态 JSON 输出。同步写 LOG_FILE 便于回溯"最终结论"，
# 并通过 fd 3 送回真 stdout 供 hatchery 解析。
emit_result() {
    echo "$1"
    echo "$1" >&3
}

# ========== 加载 openclaw 运行环境 ==========
export PNPM_HOME="${PNPM_HOME:-$HOME/.local/share/pnpm}"
export PATH="$PNPM_HOME:$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"

CONFIG="$HOME/.openclaw/openclaw.json"

echo "=== OpenClaw 就绪状态检查 ==="
echo "配置文件路径: $CONFIG"

# ========== 检查一：配置文件是否存在 ==========
echo ""
echo ">>> [检查 1/2] 验证配置文件"
if [ ! -f "$CONFIG" ]; then
    echo "✗ 配置文件不存在: $CONFIG"
    emit_result '{"ready": false, "reason": "config file not found"}'
    exit 0
fi
echo "✓ 配置文件存在: $CONFIG"

# ========== 检查二：验证 Gateway 进程就绪 ==========
echo ""
echo ">>> [检查 2/2] 验证 Gateway 就绪"

# 方式 1（首选）：openclaw gateway status --json
# 与 restart_doctor_gateway.sh 保持一致，无需手动解析端口。
if command -v openclaw &>/dev/null; then
    GW_STATUS=$(openclaw gateway status --json 2>/dev/null || true)
    echo "openclaw gateway status: ${GW_STATUS:-无输出}"
    if echo "$GW_STATUS" | grep -q '"running"'; then
        echo "✓ Gateway 正在运行 (openclaw gateway status)"
        emit_result '{"ready": true}'
        exit 0
    fi
    echo "openclaw gateway status 未报告 running，尝试端口检测"
else
    echo "openclaw 命令不可用，跳过 status 检查"
fi

# 方式 2（兜底）：curl 端口探测
# 端口获取优先级：
#   1. systemd service 文件中的 --port 参数
#   2. openclaw.json 中的 gateway.port
SERVICE_FILE="$HOME/.config/systemd/user/openclaw-gateway.service"
SERVICE_PORT=""
if [ -f "$SERVICE_FILE" ]; then
    SERVICE_PORT=$(grep -oP '(?<=--port )\d+' "$SERVICE_FILE" 2>/dev/null | head -1 || true)
fi

CONFIG_PORT=$(jq -r '.gateway.port // empty' "$CONFIG" 2>/dev/null || true)

if [ -n "$SERVICE_PORT" ]; then
    PORT="$SERVICE_PORT"
    echo "使用 systemd service 端口: $PORT"
elif [ -n "$CONFIG_PORT" ]; then
    PORT="$CONFIG_PORT"
    echo "使用配置文件端口: $PORT"
else
    echo "✗ 无法获取 gateway 端口"
    emit_result '{"ready": false, "reason": "port not configured and openclaw cli unavailable"}'
    exit 0
fi

HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "http://localhost:${PORT}/" 2>/dev/null || true)
echo "健康检查 HTTP 状态码: ${HTTP_CODE:-无响应}"

if [ -n "$HTTP_CODE" ] && [ "$HTTP_CODE" != "000" ]; then
    echo "✓ gateway 端口 $PORT 响应正常 (HTTP $HTTP_CODE)"
    emit_result '{"ready": true}'
    exit 0
fi

echo "✗ gateway 端口 $PORT 无响应"
emit_result "{\"ready\": false, \"reason\": \"health check failed on port ${PORT}\"}"
exit 0
