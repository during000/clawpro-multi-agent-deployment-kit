#!/bin/bash
# remove_smh_env.sh — 删除 SMH skill 并清除 SMH 相关环境变量
# 通过 TAT 下发到远程 CVM 上执行
#
# 参数（通过 TAT 模板变量注入）：
#   {{agent_type}}  - 智能体类型：openclaw | hermes | lightclawace（为空时默认 openclaw）
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"

SKILL_SLUG="tencent-agent-storage"

tmp=""
cleanup_tmp() {
  if [ -n "${tmp:-}" ] && [ -f "$tmp" ]; then
    rm -f "$tmp"
  fi
}
trap cleanup_tmp EXIT

log() { echo "[remove_smh_env] $*"; }

# ========== 0. 解析参数 ==========
AGENT_TYPE="{{agent_type}}"
# 兼容历史：未传或为空时按 openclaw 处理
if [ -z "$AGENT_TYPE" ]; then
  AGENT_TYPE="openclaw"
fi

case "$AGENT_TYPE" in
  openclaw)
    SKILL_DIR_BASE="$HOME/.openclaw/workspace/skills"
    ;;
  hermes)
    SKILL_DIR_BASE="$HOME/.hermes/skills"
    ;;
  lightclawace)
    SKILL_DIR_BASE="$HOME/.lightclaw/workspace/skills"
    ;;
  *)
    log "ERROR: unsupported agent_type: $AGENT_TYPE"
    exit 1
    ;;
esac

SKILL_DIR="$SKILL_DIR_BASE/$SKILL_SLUG"
log "agent_type=$AGENT_TYPE, skill_dir=$SKILL_DIR"

# ========== 1. 删除 SMH Skill ==========
skill_removed=false

# 1a. openclaw 下优先使用 clawhub uninstall（会同步清理 lockfile 状态）
if [ "$AGENT_TYPE" = "openclaw" ] && command -v clawhub >/dev/null 2>&1; then
  log "attempting clawhub uninstall $SKILL_SLUG ..."
  # clawhub uninstall 可能预期失败（如 skill 由 skillhub 安装，不在 lockfile 中），需要 fallback 到 rm -rf
  if clawhub uninstall "$SKILL_SLUG" 2>&1; then
    log "clawhub uninstall succeeded"
    skill_removed=true
  else
    log "WARN: clawhub uninstall failed (may not be in lockfile), will fallback to rm -rf"
  fi
else
  log "clawhub not used (agent_type=$AGENT_TYPE), will use rm -rf"
fi

# 1b. 兜底：无论 clawhub 是否成功，检查目录是否仍存在并强制删除
# 处理以下场景：
#   - clawhub 不可用
#   - clawhub uninstall 失败（如 skillhub 安装的，lockfile 中无记录）
#   - clawhub uninstall 成功但目录未被完全清理
#   - hermes / lightclawace 类型（无 clawhub）
if [ -d "$SKILL_DIR" ]; then
  log "skill directory still exists, removing: $SKILL_DIR"
  rm -rf "$SKILL_DIR"
  if [ -d "$SKILL_DIR" ]; then
    log "ERROR: failed to remove skill directory"
    exit 1
  fi
  log "skill directory removed"
  skill_removed=true
fi

if [ "$skill_removed" = true ]; then
  log "SMH skill removed successfully"
else
  log "SMH skill was not installed, skip"
fi

# ========== 2. 清除统一 env 文件 ~/.tencentAgentStorage/.env 中的 SMH 条目 ==========
ENV_FILE="$HOME/.tencentAgentStorage/.env"
if [ -f "$ENV_FILE" ]; then
  tmp="$(mktemp)"
  # 删除行首为 smh_basePath= / smh_libraryId= / smh_spaceId= / smh_accessToken= 的行
  awk '
    /^smh_basePath=/    { next }
    /^smh_libraryId=/   { next }
    /^smh_spaceId=/     { next }
    /^smh_accessToken=/ { next }
    { print }
  ' "$ENV_FILE" > "$tmp"
  mv "$tmp" "$ENV_FILE"
  tmp=""
  chmod 600 "$ENV_FILE" || true
  log "SMH env removed from $ENV_FILE"
else
  log "env file not found, skip: $ENV_FILE"
fi

# ========== 3. openclaw 兼容：清除 ~/.openclaw/openclaw.json 中的 SMH env ==========
if [ "$AGENT_TYPE" = "openclaw" ]; then
  CONFIG="$HOME/.openclaw/openclaw.json"
  if [ -f "$CONFIG" ]; then
    BASE_JSON="$(cat "$CONFIG")"
    if ! echo "$BASE_JSON" | jq empty 2>/dev/null; then
      log "ERROR: existing openclaw.json is not valid JSON"
      exit 1
    fi
    echo "$BASE_JSON" | jq '
      del(.env.smh_basePath, .env.smh_libraryId, .env.smh_spaceId, .env.smh_accessToken)
    ' > "$CONFIG"
    log "SMH env removed from $CONFIG (openclaw compatibility)"
  else
    log "openclaw.json not found, skip"
  fi
fi

log "SMH env removed"
echo "smh_env removed"
