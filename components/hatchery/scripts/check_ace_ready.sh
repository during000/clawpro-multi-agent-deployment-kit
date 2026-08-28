#!/bin/bash
set -uo pipefail

# ========== LightClaw-ACE 就绪状态检查脚本 ==========
# 功能：检查 lightclaw-ace 是否已安装完成并服务就绪
# 检查两个条件（均满足才认为就绪）：
#   1. ~/.lightclaw/lightclaw.json 文件存在
#   2. lightclaw 服务在 lastApi.port 上响应（或 systemctl is-active lightclaw = active）
#
# 输出：最后一行为 JSON：
#   {"ready": true}
#   {"ready": false, "reason": "<原因>"}

# ========== 日志系统初始化 ==========
# ACE 脚本未加入 hatchery rootRequiredTATScripts 白名单（controller/tat.go），
# 由 TAT 以实例普通用户身份执行，无权写 /var/log/clawpro，日志落到用户目录。
LOG_DIR="$HOME/.lightclaw/logs"
mkdir -p "$LOG_DIR"
SCRIPT_NAME="check_ace_ready"
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

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"

CONFIG="$HOME/.lightclaw/lightclaw.json"

echo "=== LightClaw-ACE 就绪状态检查 ==="
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

# ========== 检查二：从配置文件解析端口并做健康检查 ==========
echo ""
echo ">>> [检查 2/2] 验证 lightclaw lastApi 端口健康"
PORT=$(jq -r '.lastApi.port // empty' "$CONFIG" 2>/dev/null || true)
if [ -z "$PORT" ]; then
    echo "⚠ 无法从配置读取 lastApi.port，回退到 systemctl 检查"
    if systemctl is-active --quiet lightclaw 2>/dev/null; then
        echo "✓ systemctl is-active lightclaw = active"
        emit_result '{"ready": true}'
        exit 0
    fi
    echo "✗ systemctl is-active lightclaw != active"
    emit_result '{"ready": false, "reason": "port not configured and service inactive"}'
    exit 0
fi
echo "lightclaw lastApi 端口: $PORT"

HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "http://localhost:${PORT}/" 2>/dev/null || true)
echo "健康检查 HTTP 状态码: ${HTTP_CODE:-无响应}"

if [ -n "$HTTP_CODE" ] && [ "$HTTP_CODE" != "000" ]; then
    echo "✓ lightclaw 端口 $PORT 响应正常 (HTTP $HTTP_CODE)"
    emit_result '{"ready": true}'
    exit 0
fi

# HTTP 失败时再用 systemctl 兜底
if systemctl is-active --quiet lightclaw 2>/dev/null; then
    echo "⚠ HTTP 无响应，但 systemctl is-active lightclaw = active，视为就绪"
    emit_result '{"ready": true}'
    exit 0
fi

echo "✗ lightclaw 端口 $PORT 无响应 且 服务非 active 状态"
emit_result "{\"ready\": false, \"reason\": \"health check failed on port ${PORT}\"}"
exit 0
