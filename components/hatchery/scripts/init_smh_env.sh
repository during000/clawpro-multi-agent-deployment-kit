#!/bin/bash
# init_smh_env.sh — 安装 SMH skill 并配置 SMH 相关环境变量
# 通过 TAT 下发到远程 CVM 上执行
#
# 参数（通过 TAT 模板变量注入）：
#   {{agent_type}}  - 智能体类型：openclaw | hermes | lightclawace（为空时默认 openclaw）
#   {{skill_name}}  - skill 名称，如 tencent-agent-storage
#   {{basePath}}    - SMH API 基础路径，如 https://api.tencentsmh.cn
#   {{libraryId}}   - SMH 媒体库 ID，如 smhxxx-xxxxx
#   {{spaceId}}     - SMH 空间 ID，如 space-xxxxx
#   {{accessToken}} - space_admin 权限的 Token
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"

log() { echo "[init_smh_env] $*"; }

# ========== 0. 解析参数 ==========
AGENT_TYPE="{{agent_type}}"
# 兼容历史：未传或为空时按 openclaw 处理
if [ -z "$AGENT_TYPE" ]; then
  AGENT_TYPE="openclaw"
fi

case "$AGENT_TYPE" in
  openclaw)
    SKILL_DIR="$HOME/.openclaw/workspace/skills"
    ;;
  hermes)
    SKILL_DIR="$HOME/.hermes/skills"
    ;;
  lightclawace)
    SKILL_DIR="$HOME/.lightclaw/workspace/skills"
    ;;
  *)
    log "ERROR: unsupported agent_type: $AGENT_TYPE"
    exit 1
    ;;
esac
log "agent_type=$AGENT_TYPE, skill_dir=$SKILL_DIR"

# ========== 1. 确保 skillhub 命令存在 ==========
if ! command -v skillhub >/dev/null 2>&1; then
  log "skillhub not found, installing via install.sh ..."
  if ! curl -fsSL https://skillhub.cn/install/install.sh | bash -s -- --no-skills; then
    log "ERROR: failed to install skillhub"
    exit 1
  fi
  # 安装脚本通常把二进制放到 ~/.local/bin 或 ~/.npm-global/bin，刷新 PATH
  export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:$PATH"
  hash -r 2>/dev/null || true
  if ! command -v skillhub >/dev/null 2>&1; then
    log "ERROR: skillhub still not found after install"
    exit 1
  fi
fi
log "skillhub ready: $(command -v skillhub)"

# ========== 2. 安装 skill ==========
mkdir -p "$SKILL_DIR"
log "Installing skill: {{skill_name}} -> $SKILL_DIR"
if ! skillhub --dir "$SKILL_DIR" install "{{skill_name}}" --force; then
  log "ERROR: failed to install skill {{skill_name}}"
  exit 1
fi
log "Skill installed successfully"

# ========== 3. 写入统一 env 文件 ~/.tencentAgentStorage/.env ==========
ENV_FILE="$HOME/.tencentAgentStorage/.env"
mkdir -p "$(dirname "$ENV_FILE")"

# 若文件不存在则创建（带分节头）
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
  # 使用 awk 进行精确匹配替换（仅匹配行首的 key=，不触碰注释）
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

# 如果是新建文件，确保结尾有换行（cat <<EOF 已保证），再写入 key
upsert_env "smh_basePath"    "{{basePath}}"    "$ENV_FILE"
upsert_env "smh_libraryId"   "{{libraryId}}"   "$ENV_FILE"
upsert_env "smh_spaceId"     "{{spaceId}}"     "$ENV_FILE"
upsert_env "smh_accessToken" "{{accessToken}}" "$ENV_FILE"

chmod 600 "$ENV_FILE" || true
log "SMH env written to $ENV_FILE"

log "SMH env initialized"
echo "smh_env initialized"
