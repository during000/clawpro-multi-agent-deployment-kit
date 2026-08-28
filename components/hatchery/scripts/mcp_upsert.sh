#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# TAT 模板变量
SERVICE_ID="{{service_id}}"
CONFIG_B64="{{config_json_base64}}"   # 内层 config JSON，base64 编码以避免 TAT 参数转义问题

LOCK_FILE="/tmp/mcp_set.${SERVICE_ID}.lock"

cleanup() {
    rm -f "${LOCK_FILE}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# 单实例下同一 MCP 串行（实例侧保护；服务端另有分布式锁）
exec 9>"${LOCK_FILE}"
flock -n 9 || { echo "another mcp_set running for ${SERVICE_ID}" >&2; exit 10; }

# 解码 config（内层 JSON object，如 {"transportType":"...","url":"..."}）
CONFIG_JSON=$(echo "${CONFIG_B64}" | base64 -d 2>/dev/null)
if [ $? -ne 0 ] || [ -z "${CONFIG_JSON}" ]; then
    echo "base64 decode failed for config" >&2
    exit 2
fi

# 先移除旧配置（忽略不存在的情况），再写入新配置
openclaw mcp unset "${SERVICE_ID}" 2>/dev/null || true
openclaw mcp set "${SERVICE_ID}" "${CONFIG_JSON}"
# 不重启 openclaw-gateway — 由调用方决定是否额外执行 restart_gateway.sh
