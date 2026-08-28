#!/bin/bash
set -uo pipefail

# ========== Hermes 就绪状态检查脚本 ==========
# 功能：检查 Hermes 是否已安装完成并服务就绪
# 依赖：harness CLI 必须可用
# 检查项：
#   harness gateway status JSON 里 .running == true 即视为 ready
#
# 输出：最后一行为 JSON：
#   {"ready": true}
#   {"ready": false, "reason": "<原因>"}

# ========== 日志系统初始化 ==========
LOG_DIR="$HOME/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
SCRIPT_NAME="check_hermes_ready"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"

# ========== stdout 契约：仅最终 JSON 走真 stdout ==========
# 历史 bug（工蜂 CR #6/9）：原写法 `exec > >(tee -a "$LOG_FILE") 2>&1`
# 把所有调试 echo 也打到 stdout，与最后一行 JSON 混染，导致
# hatchery/controller/openclaw.go:1953 `json.Unmarshal([]byte(output), ...)`
# 永远解析失败 → `HandleCheckAgentReady` 始终返回 `{"running":false}`，
# 前端/监控看到"实例常年未就绪"的假象。
#
# 新契约：
#   - fd 3 = 真正的 stdout（调用方 TAT 捕获目标）。exec 3>&1 先把它保存起来。
#   - 脚本主体的 stdout/stderr 全部追加进 LOG_FILE（不再落到真 stdout）。
#   - 终态 JSON 专门通过 emit_result 写到 fd 3。
#
# 兼容性：LOG_FILE 仍保留全部调试日志，排障体验不退化。
exec 3>&1
exec >>"$LOG_FILE" 2>&1

echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# emit_result：仅用于终态 JSON 输出。
#   - 先写 LOG_FILE（保留"最终结论"便于回溯）
#   - 再写 fd 3（真 stdout），供 hatchery 解析
emit_result() {
    echo "$1"           # 当前 stdout 已重定向到 LOG_FILE
    echo "$1" >&3       # 送回真 stdout，给调用方解析
}

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:/usr/local/bin:$PATH"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"

# %INCLUDE% lib_acli_compat.sh

# ===== 脚本级探测一次，后续复用 =====
_acli_mode="$(ensure_acli_light 2>>"$LOG_FILE")"

# ===== acli 路径（信任 acli 结果，不再 fallback） =====
# acli gateway status 输出格式：{"success":true,"data":{"running":bool,"pid":int|null,"port":int|null,"message":"..."}}
if [ "$_acli_mode" = "acli" ]; then
    echo ""
    echo ">>> [acli 路径] acli gateway status"
    _acli_status="$(acli gateway status 2>/dev/null || true)"
    echo "acli gateway status raw: ${_acli_status:-<empty>}"
    if [ -z "$_acli_status" ]; then
        emit_result '{"ready": false, "reason": "acli gateway status returned empty"}'
        exit 0
    fi
    if command -v jq >/dev/null 2>&1; then
        _running="$(printf '%s' "$_acli_status" | jq -r '.data.running // false' 2>/dev/null)"
    else
        if printf '%s' "$_acli_status" | grep -q '"running"[[:space:]]*:[[:space:]]*true'; then
            _running="true"
        else
            _running="false"
        fi
    fi
    if [ "$_running" = "true" ]; then
        echo "✓ gateway running"
        emit_result '{"ready": true}'
    else
        echo "✗ gateway not running"
        emit_result '{"ready": false, "reason": "gateway not running (acli)"}'
    fi
    exit 0
fi

# ===== fallback: harness 路径（仅 acli 不可用时走此分支） =====
# ========== 拉取/更新 harness CLI ==========
ensure_harness_cli() {
    local HARNESS_URL="https://agentone-1388575217.cos.ap-guangzhou.myqcloud.com/harness-linux-amd64"
    local HARNESS_BIN="$HOME/.local/bin/harness"
    mkdir -p "$(dirname "$HARNESS_BIN")" 2>/dev/null || true
    echo "ℹ 拉取 harness CLI: $HARNESS_URL"
    if curl -fsSL --connect-timeout 10 --max-time 60 --retry 2 --retry-delay 2 \
        "$HARNESS_URL" -o "${HARNESS_BIN}.tmp" 2>/dev/null; then
        chmod +x "${HARNESS_BIN}.tmp"
        mv -f "${HARNESS_BIN}.tmp" "$HARNESS_BIN"
        echo "✓ harness CLI 已更新: $HARNESS_BIN"
    else
        rm -f "${HARNESS_BIN}.tmp" 2>/dev/null || true
        if command -v harness >/dev/null 2>&1; then
            echo "⚠ harness CLI 下载失败，沿用已有版本: $(command -v harness)"
        else
            echo "✗ harness CLI 下载失败且本地无已有版本" >&2
            return 1
        fi
    fi
}

echo "=== Hermes 就绪状态检查 ==="

# ========== 检查一：harness CLI 是否存在（优先拉取最新版） ==========
echo ""
echo ">>> [检查 1/2] 验证 harness CLI"
if ! ensure_harness_cli; then
    emit_result '{"ready": false, "reason": "harness CLI not found"}'
    exit 0
fi
if ! command -v harness >/dev/null 2>&1; then
    echo "✗ harness CLI 不存在"
    emit_result '{"ready": false, "reason": "harness CLI not found"}'
    exit 0
fi
echo "✓ harness CLI 存在: $(command -v harness)"

# ========== 检查二：harness gateway status ==========
echo ""
echo ">>> [检查 2/2] harness gateway status"
STATUS_JSON=$(harness gateway status 2>/dev/null || true)
if [ -z "$STATUS_JSON" ]; then
    echo "✗ harness gateway status 无输出"
    emit_result '{"ready": false, "reason": "harness gateway status returned empty"}'
    exit 0
fi
echo "gateway status raw: $STATUS_JSON"

if ! command -v jq >/dev/null 2>&1; then
    # 无 jq 时简单用 grep
    if echo "$STATUS_JSON" | grep -q '"running"[[:space:]]*:[[:space:]]*true'; then
        echo "✓ gateway running=true (grep 判定)"
        emit_result '{"ready": true}'
        exit 0
    fi
    echo "✗ gateway running != true"
    emit_result '{"ready": false, "reason": "gateway not running (no jq)"}'
    exit 0
fi

RUNNING=$(echo "$STATUS_JSON" | jq -r '.running // false')
INSTALLED=$(echo "$STATUS_JSON" | jq -r '.installed // false')
STATUS_FIELD=$(echo "$STATUS_JSON" | jq -r '.status // "unknown"')

echo "installed=$INSTALLED running=$RUNNING status=$STATUS_FIELD"

if [ "$RUNNING" = "true" ]; then
    echo "✓ Hermes gateway 就绪"
    emit_result '{"ready": true}'
    exit 0
fi

REASON="gateway not running (status=${STATUS_FIELD}, installed=${INSTALLED})"
echo "✗ $REASON"
emit_result "{\"ready\": false, \"reason\": \"${REASON}\"}"
exit 0
