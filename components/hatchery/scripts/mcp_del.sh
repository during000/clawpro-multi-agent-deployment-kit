#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# TAT 模板变量
SERVICE_ID="{{service_id}}"

# 删除 MCP 配置；如果不存在则视为成功（幂等）
OUTPUT=$(openclaw mcp unset "${SERVICE_ID}" 2>&1) || {
    if echo "${OUTPUT}" | grep -qi "No MCP server named"; then
        echo "MCP ${SERVICE_ID} 不存在，视为已删除"
        exit 0
    fi
    echo "${OUTPUT}" >&2
    exit 1
}
# 不重启 openclaw-gateway — 由调用方决定是否额外执行 restart_gateway.sh
