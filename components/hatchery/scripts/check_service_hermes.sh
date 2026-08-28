#!/bin/bash
# check_service_hermes.sh
# 探测 Hermes 服务整体状态。契约：stdout JSON（单 object）：
#   {"gateway": <...>, "update": <...>, "channelSummary": <...>}
# 与 scripts/check_service.sh（openclaw）同形状。
#
# Hermes 语义映射：
#   - gateway：通过 `harness gateway status`（JSON 输出）转换
#   - update：暂不支持在线升级，返回 {available: false}
#   - channelSummary：通过 `harness channel list` 统计

set -uo pipefail
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u 2>/dev/null || echo 0)

# %INCLUDE% lib_acli_compat.sh

LOG_DIR="${HOME}/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/check_service_hermes.log"
exec 2>>"$LOG_FILE"
echo "" >>"$LOG_FILE"
echo "========== $(date '+%Y-%m-%d %H:%M:%S') check_service_hermes ==========" >>"$LOG_FILE"

# ===== 脚本级探测一次，后续复用 =====
_acli_mode="$(ensure_acli 2>>"$LOG_FILE")"

gateway_running=false
gateway_api=""
gateway_version=""

# ===== acli 路径（信任 acli 结果） =====
# acli gateway status 输出：{"success":true,"data":{"running":bool,"pid":int|null,"port":int|null,"message":"..."}}
if [ "$_acli_mode" = "acli" ] && command -v jq >/dev/null 2>&1; then
    _acli_status="$(acli gateway status 2>>"$LOG_FILE" || true)"
    if [ -n "$_acli_status" ] && echo "$_acli_status" | jq empty >/dev/null 2>&1; then
        if [ "$(echo "$_acli_status" | jq -r '.data.running // false' 2>>"$LOG_FILE")" = "true" ]; then
            gateway_running=true
        fi
        gateway_api="$(echo "$_acli_status" | jq -r '.data.api // .data.endpoint // ""' 2>>"$LOG_FILE")"
        gateway_version="$(echo "$_acli_status" | jq -r '.data.version // ""' 2>>"$LOG_FILE")"
        echo "acli gateway status 解析成功 (running=$gateway_running)" >>"$LOG_FILE"
    else
        echo "acli gateway status 输出为空或非 JSON" >>"$LOG_FILE"
    fi
else
    # ===== fallback: harness 路径 =====
    if command -v harness >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
        status_json=$(harness gateway status 2>>"$LOG_FILE" || true)
        if [ -n "$status_json" ] && echo "$status_json" | jq empty >/dev/null 2>&1; then
            if [ "$(echo "$status_json" | jq -r '.running // false')" = "true" ]; then
                gateway_running=true
            fi
            gateway_api=$(echo "$status_json" | jq -r '.api // .endpoint // ""')
            gateway_version=$(echo "$status_json" | jq -r '.version // ""')
        fi
    fi
fi

# ===== channel summary =====
total_channels=0
enabled_channels=0

if [ "$_acli_mode" = "acli" ] && command -v jq >/dev/null 2>&1; then
    # acli channel status 输出：{"success":true,"data":{"gateway_running":bool,"platforms":[{"name":"x","status":"connected|disconnected"}]}}
    _acli_ch="$(acli channel status 2>>"$LOG_FILE" || true)"
    if [ -n "$_acli_ch" ] && echo "$_acli_ch" | jq empty >/dev/null 2>&1; then
        total_channels=$(echo "$_acli_ch" | jq -r '.data.platforms | length' 2>>"$LOG_FILE" || echo 0)
        enabled_channels=$(echo "$_acli_ch" | jq -r '[.data.platforms[] | select(.status == "connected")] | length' 2>>"$LOG_FILE" || echo 0)
        echo "acli channel status 解析成功 (total=$total_channels enabled=$enabled_channels)" >>"$LOG_FILE"
    fi
else
    if command -v harness >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
        ch_json=""
        for args in "--output json" "--format json" "-o json" ""; do
            # shellcheck disable=SC2086
            ch_json=$(harness channel list $args 2>>"$LOG_FILE" || true)
            if [ -n "$ch_json" ] && echo "$ch_json" | jq empty >/dev/null 2>&1; then
                break
            fi
            ch_json=""
        done
        if [ -n "$ch_json" ]; then
            total_channels=$(echo "$ch_json" | jq -r '(.channels // .) | length // 0' 2>>"$LOG_FILE" || echo 0)
            enabled_channels=$(echo "$ch_json" | jq -r '
                (.channels // .)
                | [.[] | select((.effective // .enabled // false) == true)]
                | length
            ' 2>>"$LOG_FILE" || echo 0)
        fi
    fi
fi

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
    printf '{"gateway":{"running":%s,"api":"%s","version":"%s"},"update":{"available":false},"channelSummary":{"total":%s,"enabled":%s}}\n' \
        "$gateway_running" "$gateway_api" "$gateway_version" "$total_channels" "$enabled_channels"
fi
