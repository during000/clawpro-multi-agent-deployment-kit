#!/bin/bash
# memory_tdai_switch_free.sh — 启用 memory-tencentdb 插件（Free 模式）
# 支持 OpenClaw / Hermes 两种智能体类型。
# OpenClaw：openclaw plugins install + jq 改 openclaw.json + systemctl restart
# Hermes：确保 tdai-gateway.yaml + 同步 LLM 配置 + 启用 provider + 重启 hermes
set -uo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"
export NO_COLOR=1

# %INCLUDE% lib_acli_compat.sh

plugin_fullname="{{plugin}}"
supported_versions='{{supported_versions}}'
agent_type="{{agent_type}}"

log() { echo "[switch_free] $*"; }

# ==================== 智能体类型检测 ====================
# Go 传参优先；为空或无效时，脚本通过 --version 自动检测
detect_agent_type() {
  if command -v hermes >/dev/null 2>&1; then
    local ver
    ver=$(hermes --version 2>/dev/null | head -n1)
    if echo "$ver" | grep -qi "Hermes"; then
      log "detected agent_type=hermes, version=$ver"
      echo "hermes"
      return
    fi
  fi
  if command -v openclaw >/dev/null 2>&1; then
    local ver
    ver=$(openclaw --version 2>/dev/null | head -n1)
    if echo "$ver" | grep -qi "OpenClaw"; then
      log "detected agent_type=openclaw, version=$ver"
      echo "openclaw"
      return
    fi
  fi
  # 兜底：目录特征
  if [ -d "$HOME/.hermes" ]; then
    log "detected agent_type=hermes (by ~/.hermes directory)"
    echo "hermes"
    return
  fi
  # 默认 openclaw（向后兼容）
  log "detected agent_type=openclaw (default)"
  echo "openclaw"
}

if [ -z "$agent_type" ] || { [ "$agent_type" != "openclaw" ] && [ "$agent_type" != "hermes" ]; }; then
  agent_type=$(detect_agent_type)
fi
log "agent_type=$agent_type, plugin=$plugin_fullname"

# ==================== 分支入口 ====================
if [ "$agent_type" = "hermes" ]; then
  # ==================== Hermes 分支 ====================

  # ---- H0. 加载 Gateway 环境变量 ----
  if [ -f /etc/profile.d/memory-tencentdb-env.sh ]; then
    source /etc/profile.d/memory-tencentdb-env.sh
    log "Gateway env loaded"
  else
    log "WARN: /etc/profile.d/memory-tencentdb-env.sh not found"
  fi

  # ---- H1. 版本校验（Hermes 版本） ----
  hermes_version=""
  if command -v hermes >/dev/null 2>&1; then
    hermes_version=$(hermes --version 2>/dev/null | head -n1 | sed -E 's/.*v([0-9]+\.[0-9]+\.[0-9]+).*/\1/')
  fi
  log "hermes_version=$hermes_version, supported_versions=$supported_versions"

  if [ -n "$supported_versions" ] && [ "$supported_versions" != "[]" ]; then
    if [ -z "$hermes_version" ]; then
      log "ERROR: UNSUPPORTED_VERSION - failed to detect hermes version"
      echo "UNSUPPORTED_VERSION: unable to detect hermes version, allowed=$supported_versions"
      exit 21
    fi
    if command -v jq >/dev/null 2>&1; then
      if ! echo "$supported_versions" | jq -e --arg v "$hermes_version" 'type == "array" and (map(. == $v) | any)' >/dev/null 2>&1; then
        log "ERROR: UNSUPPORTED_VERSION - current=$hermes_version, allowed=$supported_versions"
        echo "UNSUPPORTED_VERSION: current=$hermes_version, allowed=$supported_versions"
        exit 21
      fi
      log "hermes version check passed"
    else
      log "WARN: jq not found, skip version whitelist check"
    fi
  else
    log "no version whitelist configured, skip version check"
  fi

  # ---- H2. 检测插件是否已安装 ----
  HERMES_PLUGIN_DIR="$HOME/.hermes/hermes-agent/plugins/memory/memory_tencentdb"
  TDAI_INSTALL_DIR="$HOME/.memory-tencentdb/tdai-memory-openclaw-plugin"

  is_hermes_plugin_installed() {
    [ -L "$HERMES_PLUGIN_DIR" ] || return 1
    [ -d "$HERMES_PLUGIN_DIR" ] || return 1
    [ -f "$TDAI_INSTALL_DIR/src/gateway/server.ts" ] || return 1
    [ -d "$TDAI_INSTALL_DIR/node_modules" ] || return 1
    return 0
  }

  if ! is_hermes_plugin_installed; then
    log "ERROR: Hermes plugin not installed (expected Go backend to run install_hermes_tdai_gateway.sh first)"
    exit 6
  fi
  log "hermes plugin installed: OK"

  # ---- H3. 幂等检查：provider 已启用 ----
  HERMES_CONFIG="$HOME/.hermes/config.yaml"
  gateway_config="$TDAI_INSTALL_DIR/tdai-gateway.json"
  if [ -f "$HERMES_CONFIG" ] && grep -q "provider:.*memory_tencentdb" "$HERMES_CONFIG" 2>/dev/null; then
    if [ -f "$gateway_config" ]; then
      log "memory provider already enabled and gateway config exists, nothing to do"
      echo "memory_tdai already enabled"
      exit 0
    fi
  fi

  # ---- H3.5 清理旧 YAML 配置（全面迁移到 JSON） ----
  old_yaml_config="$TDAI_INSTALL_DIR/tdai-gateway.yaml"
  if [ -f "$old_yaml_config" ]; then
    log "found legacy tdai-gateway.yaml, migrating to JSON..."
    mv "$old_yaml_config" "${old_yaml_config}.bak.$(date +%Y%m%d%H%M%S)"
    log "old YAML config backed up"
  fi

  # ---- H4. 同步 LLM 配置到 TDAI Gateway ----
  # 调用插件提供的 memory-tencentdb-ctl.sh（随插件安装到 CVM）。
  # ctl.sh --hermes config llm 会：
  #   - 自动创建 tdai-gateway.json（如不存在）
  #   - 写入 LLM 配置
  #   - hermes 模式下额外写 env.d（供 systemd 继承）
  # LLM 信息从 ~/.hermes/config.yaml 读取，不依赖 hatchery 传参。
  CTL_SCRIPT="$TDAI_INSTALL_DIR/scripts/memory-tencentdb-ctl.sh"
  HERMES_CONFIG="$HOME/.hermes/config.yaml"

  if [ -f "$CTL_SCRIPT" ]; then
    # 从 hermes config.yaml 提取 LLM 配置
    api_key=""
    base_url=""
    default_model=""
    if [ -f "$HERMES_CONFIG" ]; then
      api_key=$(sed -n '/^model:/,/^[a-z]/{/^[[:space:]]*api_key:/p}' "$HERMES_CONFIG" | head -1 | sed 's/^[[:space:]]*api_key:[[:space:]]*//' | tr -d '"' | tr -d "'")
      base_url=$(sed -n '/^model:/,/^[a-z]/{/^[[:space:]]*base_url:/p}' "$HERMES_CONFIG" | head -1 | sed 's/^[[:space:]]*base_url:[[:space:]]*//' | tr -d '"' | tr -d "'")
      default_model=$(sed -n '/^model:/,/^[a-z]/{/^[[:space:]]*default:/p}' "$HERMES_CONFIG" | head -1 | sed 's/^[[:space:]]*default:[[:space:]]*//' | tr -d '"' | tr -d "'")
    fi

    if [ -n "$api_key" ] && [ -n "$base_url" ] && [ -n "$default_model" ]; then
      log "syncing LLM config via ctl.sh: baseUrl=$base_url, model=$default_model"
      bash "$CTL_SCRIPT" --hermes config llm \
        --api-key "$api_key" \
        --base-url "$base_url" \
        --model "$default_model" 2>&1 || log "WARN: ctl config llm failed (non-fatal)"
    else
      log "WARN: hermes config.yaml missing LLM config, skip sync"
    fi

    # ---- H4.5 确保 storeBackend=sqlite（Free 模式） ----
    # 清掉可能残留的 Pro 配置（tcvdb/bm25），确保 Free 模式用 sqlite
    log "ensuring storeBackend=sqlite for Free mode"
    bash "$CTL_SCRIPT" --hermes config vdb-off --purge-creds 2>&1 || log "WARN: config vdb-off failed (non-fatal)"

    # ---- H5. 启用 memory provider ----
    log "enabling hermes memory provider via ctl.sh"
    bash "$CTL_SCRIPT" --hermes enable-hermes-memory 2>&1 || log "WARN: enable-hermes-memory failed (non-fatal)"
  else
    log "WARN: memory-tencentdb-ctl.sh not found ($CTL_SCRIPT), skip LLM sync and memory enable"
  fi

  # ---- H8. 重启 Hermes ----
  # 优先 systemctl（环境变量通过 drop-in 注入），acli 作为兜底
  restarted=false
  for unit in hermes-gateway hermes harness-gateway; do
    if systemctl --user is-enabled "$unit" >/dev/null 2>&1; then
      log "restarting hermes via systemctl --user restart $unit..."
      systemctl --user restart "$unit" 2>&1 && restarted=true && break || true
    fi
  done
  if [ "$restarted" = false ]; then
    _acli_mode="$(ensure_acli 2>/dev/null)"
    if [ "$_acli_mode" = "acli" ]; then
      log "restarting hermes via acli gateway restart..."
      acli gateway restart 2>&1 || log "WARN: acli gateway restart failed"
    fi
  fi

  echo "memory_tdai switch_free done"
  exit 0
fi

# ==================== OpenClaw 分支（原有逻辑不变） ====================

plugin_id="${plugin_fullname##*/}"
if [ "$plugin_id" = "$plugin_fullname" ]; then
  plugin_id="${plugin_fullname#@*/}"
fi
config_file="$HOME/.openclaw/openclaw.json"

# ========== 1. 前置检查 ==========
if ! command -v jq >/dev/null 2>&1; then
  log "ERROR: jq not found"
  exit 1
fi
if ! command -v openclaw >/dev/null 2>&1; then
  log "ERROR: openclaw command not found"
  exit 1
fi
log "openclaw=$(command -v openclaw), jq=$(command -v jq)"

# ========== 2. 版本白名单校验 ==========
openclaw_version=""
if [ -f "$config_file" ]; then
  openclaw_version=$(jq -r '.meta.lastTouchedVersion // empty' "$config_file" 2>/dev/null)
fi
if [ -z "$openclaw_version" ]; then
  openclaw_version=$(openclaw --version 2>/dev/null | head -n 1 | sed -E 's/[^0-9]*([0-9]+(\.[0-9]+){1,3}).*/\1/')
fi
log "openclaw_version=$openclaw_version, supported_versions=$supported_versions"

if [ -n "$supported_versions" ] && [ "$supported_versions" != "[]" ]; then
  if [ -z "$openclaw_version" ]; then
    log "ERROR: UNSUPPORTED_VERSION - failed to detect openclaw version"
    echo "UNSUPPORTED_VERSION: unable to detect openclaw version, allowed=$supported_versions"
    exit 21
  fi
  if ! echo "$supported_versions" | jq -e --arg v "$openclaw_version" 'type == "array" and (map(. == $v) | any)' >/dev/null 2>&1; then
    log "ERROR: UNSUPPORTED_VERSION - current=$openclaw_version, allowed=$supported_versions"
    echo "UNSUPPORTED_VERSION: current=$openclaw_version, allowed=$supported_versions"
    exit 21
  fi
  log "version check passed"
else
  log "no version whitelist configured, skip version check"
fi

# ========== 3. 读取当前状态 ==========
installed=false
enabled=false

if [ -f "$config_file" ]; then
  entry=$(jq -r --arg id "$plugin_id" '.plugins.entries[$id] // empty' "$config_file" 2>/dev/null)
  if [ -n "$entry" ]; then
    installed=true
    if echo "$entry" | jq -e '.enabled == true' >/dev/null 2>&1; then
      enabled=true
    fi
  fi
fi
log "installed=$installed, enabled=$enabled"

# ========== 4. 已启用 → 幂等跳过 ==========
if [ "$enabled" = true ]; then
  log "plugin already enabled, nothing to do"
  echo "memory_tdai already enabled"
  exit 0
fi

# ========== 5. 插件应已安装（由 ensure_memory_plugin.sh 保证） ==========
if [ "$installed" = false ]; then
  log "ERROR: plugin not installed. Run ensure_memory_plugin.sh first."
  exit 2
fi

# ========== 6. 启用插件（jq 直改 JSON） ==========
log "enabling plugin via jq ..."
cp "$config_file" "${config_file}.bak.$(date +%Y%m%d%H%M%S)"

jq --arg id "$plugin_id" '
  if .plugins.entries[$id] == null then
    .plugins.entries[$id] = {"enabled": true}
  else
    .plugins.entries[$id].enabled = true
  end |
  .plugins.allow = ((.plugins.allow // []) | if index($id) then . else . + [$id] end)
' "$config_file" > /tmp/_openclaw_enable_tmp.json

if [ ! -s /tmp/_openclaw_enable_tmp.json ]; then
  log "ERROR: jq output empty, config not modified"
  exit 3
fi
mv /tmp/_openclaw_enable_tmp.json "$config_file"
log "config updated: enabled=true"

# ========== 7. 验证 ==========
verify=$(jq -r --arg id "$plugin_id" '.plugins.entries[$id].enabled' "$config_file" 2>/dev/null)
if [ "$verify" != "true" ]; then
  log "ERROR: verification failed, enabled=$verify"
  exit 4
fi

# ========== 8. 重启 gateway ==========
if command -v systemctl >/dev/null 2>&1 && systemctl --user is-enabled openclaw-gateway >/dev/null 2>&1; then
  log "restarting openclaw-gateway ..."
  systemctl --user restart openclaw-gateway 2>&1 || log "WARN: gateway restart failed (non-fatal)"
else
  log "WARN: openclaw-gateway not available, skip restart"
fi

echo "memory_tdai switch_free done"
