#!/bin/bash
# check_service_ace.sh
# 探测 LightClaw ACE 服务整体状态。契约：stdout JSON（单 object）：
#   {"gateway": <...>, "update": <...>, "channelSummary": <...>}
# 与 scripts/check_service.sh（openclaw）保持同一形状，便于 Go 层 / 前端统一消费。
#
# ACE 语义映射（与 openclaw 对齐）：
#   - gateway：{running: bool, api: "0.0.0.0:8088 reachable|unreachable", version: "<semver>"}
#     通过 `lightclaw status` 文本输出解析（CLI 目前无 --json）。
#   - update：ACE 无在线升级能力，返回 {available: false}。
#   - channelSummary：从 $HOME/.lightclaw/lightclaw.json 统计已启用 channel 数。

set -uo pipefail
export PATH="$HOME/.lightclaw/bin:$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u 2>/dev/null || echo 0)

LOG_DIR="${HOME}/.lightclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/check_service_ace.log"
exec 2>>"$LOG_FILE"
echo "" >>"$LOG_FILE"
echo "========== $(date '+%Y-%m-%d %H:%M:%S') check_service_ace ==========" >>"$LOG_FILE"

# ===== gateway 状态 =====
gateway_running=false
gateway_api=""
gateway_version=""

if command -v lightclaw >/dev/null 2>&1; then
    status_out=$(lightclaw status 2>>"$LOG_FILE" || true)
    # 典型输出：
    #   Service (lightclaw.service): running
    #   API  (0.0.0.0:8088): reachable (v0.1.1)
    if echo "$status_out" | grep -qiE "Service.*running"; then
        gateway_running=true
    fi
    gateway_api=$(echo "$status_out" | grep -iE "^API" | head -1 | sed 's/[[:space:]]\+/ /g')
    gateway_version=$(echo "$status_out" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 | sed 's/^v//')
fi

# ===== channel summary =====
total_channels=0
enabled_channels=0
CFG="$HOME/.lightclaw/lightclaw.json"
if [ -f "$CFG" ] && command -v jq >/dev/null 2>&1; then
    total_channels=$(jq -r '.channels | length' "$CFG" 2>>"$LOG_FILE" || echo 0)
    enabled_channels=$(jq -r '[.channels | to_entries[] | select(.value.enabled == true)] | length' "$CFG" 2>>"$LOG_FILE" || echo 0)
fi

# ===== 输出 =====
# 使用 jq 生成 JSON（更安全），不可用时回退手拼
if command -v jq >/dev/null 2>&1; then
    jq -nc \
        --argjson running "$gateway_running" \
        --arg api "$gateway_api" \
        --arg version "$gateway_version" \
        --argjson total "$total_channels" \
        --argjson enabled "$enabled_channels" \
        '{
            gateway: { running: $running, api: $api, version: $version },
            update: { available: false },
            channelSummary: { total: $total, enabled: $enabled }
        }'
else
    # jq 缺失：降级手拼（不推荐，但兜底）
    printf '{"gateway":{"running":%s,"api":"%s","version":"%s"},"update":{"available":false},"channelSummary":{"total":%s,"enabled":%s}}\n' \
        "$gateway_running" "$gateway_api" "$gateway_version" "$total_channels" "$enabled_channels"
fi
