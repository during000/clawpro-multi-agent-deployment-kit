#!/bin/bash
# set_smh_token.sh — 刷新 SMH access token 到 ~/.tencentAgentStorage/.env
# 通过 TAT 下发到远程 CVM 上执行
#
# 参数（通过 TAT 模板变量注入）：
#   {{agent_type}}  - 智能体类型：openclaw | hermes | lightclawace（为空时默认 openclaw）
#   {{basePath}}    - SMH API 基础路径，如 https://api.tencentsmh.cn
#   {{libraryId}}   - SMH 媒体库 ID，如 smhxxx-xxxxx
#   {{spaceId}}     - SMH 空间 ID，如 space-xxxxx
#   {{accessToken}} - space_admin 权限的 Token
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"

log() { echo "[set_smh_token] $*"; }

# ========== 0. 解析参数 ==========
AGENT_TYPE="{{agent_type}}"
# 兼容历史：未传或为空时按 openclaw 处理
if [ -z "$AGENT_TYPE" ]; then
  AGENT_TYPE="openclaw"
fi

case "$AGENT_TYPE" in
  openclaw|hermes|lightclawace) ;;
  *)
    log "ERROR: unsupported agent_type: $AGENT_TYPE"
    exit 1
    ;;
esac
log "agent_type=$AGENT_TYPE"

# ========== 1. 写入统一 env 文件 ~/.tencentAgentStorage/.env ==========
ENV_FILE="$HOME/.tencentAgentStorage/.env"
mkdir -p "$(dirname "$ENV_FILE")"

if [ ! -f "$ENV_FILE" ]; then
  log "env file not found, creating: $ENV_FILE"
  cat > "$ENV_FILE" <<'EOF'
# ============================================================================
# TENCENT SMH CONFIGURATION
# ============================================================================
EOF
fi

# 使用 awk 幂等地 upsert 指定 key=value；不存在则追加，存在则原地替换
upsert_env() {
  local key="$1"
  local val="$2"
  local file="$3"
  local tmp
  tmp="$(mktemp)"
  # 使用字面量前缀比较，避免 key 被当作正则解析
  awk -v k="$key" -v v="$val" '
    BEGIN { found = 0; prefix = k "="; plen = length(prefix) }
    {
      if (substr($0, 1, plen) == prefix) {
        print k "=" v
        found = 1
      } else {
        print $0
      }
    }
    END {
      if (!found) {
        print k "=" v
      }
    }
  ' "$file" > "$tmp"
  mv "$tmp" "$file"
}

upsert_env "smh_basePath"    "{{basePath}}"    "$ENV_FILE"
upsert_env "smh_libraryId"   "{{libraryId}}"   "$ENV_FILE"
upsert_env "smh_spaceId"     "{{spaceId}}"     "$ENV_FILE"
upsert_env "smh_accessToken" "{{accessToken}}" "$ENV_FILE"

chmod 600 "$ENV_FILE" || true
log "SMH token written to $ENV_FILE"

log "SMH token injected"
echo "smh_token configured"
