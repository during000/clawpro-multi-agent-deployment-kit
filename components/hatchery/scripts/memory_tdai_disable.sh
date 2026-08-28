#!/bin/bash
# memory_tdai_disable.sh — 禁用 memory-tencentdb 插件
# 支持 OpenClaw / Hermes 两种智能体类型。
# OpenClaw：Pro→OFF 时先导出 VDB 数据，然后 jq 改 openclaw.json + 重启 openclaw
# Hermes（Free→OFF）：移除 memory.provider 配置 + 重启 hermes
#
# Pro→OFF 时根据 hatchery 透传的 skip_export 决定是否导出 VDB 数据；
# 然后禁用插件、清理 VDB 配置、重启对应智能体。
#
# 数据导出策略（Pro→OFF）：
#   - skip_export=false → 走正常 export 流程，导出失败则 abort（保护数据）
#   - skip_export=true  → 跳过 export 直接清理（hatchery 已通过预检确认网络不通，
#                         此时 VDB 端不可能有"未导出的新数据"，跳过安全）
# 网络连通性预检由 hatchery 侧 precheck_vdb_connectivity.sh 完成，本脚本只负责执行。
set -uo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"
export NO_COLOR=1

# %INCLUDE% lib_acli_compat.sh

plugin_fullname="{{plugin}}"
clear_pro_config="{{clear_pro_config}}"
vdb_endpoint="{{vdb_endpoint}}"
vdb_database="{{vdb_database}}"
vdb_api_key="{{vdb_api_key}}"
vdb_username="{{vdb_username}}"
job_id="{{job_id}}"
agent_type="{{agent_type}}"
skip_export="{{skip_export}}"  # "true" 时跳过 VDB 数据导出，由 hatchery 根据连通性预检决定
plugin_root="{{plugin_root}}"  # 插件根目录（由 hatchery ResolveMemoryPluginRoot 探测传入）

log() { echo "[disable] $*"; }

# ==================== 智能体类型检测 ====================
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
  if [ -d "$HOME/.hermes" ]; then
    log "detected agent_type=hermes (by ~/.hermes directory)"
    echo "hermes"
    return
  fi
  log "detected agent_type=openclaw (default)"
  echo "openclaw"
}

if [ -z "$agent_type" ] || { [ "$agent_type" != "openclaw" ] && [ "$agent_type" != "hermes" ]; }; then
  agent_type=$(detect_agent_type)
fi
log "agent_type=$agent_type, plugin=$plugin_fullname, clear_pro_config=$clear_pro_config"

# ==================== 分支入口 ====================
if [ "$agent_type" = "hermes" ]; then
  # ==================== Hermes 分支（Free/Pro → OFF） ====================

  HERMES_CONFIG="$HOME/.hermes/config.yaml"
  TDAI_INSTALL_DIR="$HOME/.memory-tencentdb/tdai-memory-openclaw-plugin"
  gateway_config="$TDAI_INSTALL_DIR/tdai-gateway.json"
  plugin_data_dir="$HOME/.memory-tencentdb/memory-tdai"

  # ---- D1. 幂等检查：memory: 段内 provider 未启用 → 已是 OFF ----
  # Pro→OFF 时 clear_pro_config=true，即使 provider 已移除也需要继续清理 Pro 配置
  _memory_provider_enabled=false
  if [ -f "$HERMES_CONFIG" ] && grep -q "^memory:" "$HERMES_CONFIG" 2>/dev/null; then
    if sed -n '/^memory:/,/^[a-zA-Z]/p' "$HERMES_CONFIG" | grep -q "provider:.*memory_tencentdb"; then
      _memory_provider_enabled=true
    fi
  fi
  if [ "$_memory_provider_enabled" = false ]; then
    if [ "$clear_pro_config" != "true" ]; then
      log "memory provider not enabled, nothing to disable"
      echo "memory_tdai already disabled"
      exit 0
    fi
    log "memory provider not enabled, but clear_pro_config=true, continue to export and clean up"
  fi

  # ---- D2. Pro→OFF：导出 VDB 数据到本地 ----
  if [ "$clear_pro_config" = "true" ]; then
    log "Pro→OFF: exporting VDB data to local ..."

    export_bin="$TDAI_INSTALL_DIR/bin/export-tencent-vdb.mjs"
    export_output_dir="$plugin_data_dir/vdb-export-$(date +%Y%m%d_%H%M%S)"

    if [ -f "$export_bin" ]; then
      log "calling export tool: $export_bin"
      log "output dir: $export_output_dir"

      export_args=(
        --url "$vdb_endpoint"
        --username "$vdb_username"
        --api-key "$vdb_api_key"
        --database "$vdb_database"
        -o "$export_output_dir"
      )

      if node "$export_bin" "${export_args[@]}" 2>&1; then
        log "VDB data export completed successfully"
        echo "$export_output_dir" > "$plugin_data_dir/.last_vdb_export_path"
      else
        export_exit=$?
        log "ERROR: VDB data export failed with exit code $export_exit, aborting disable"
        exit "$export_exit"
      fi
    else
      log "ERROR: export-tencent-vdb tool not found at $export_bin, aborting disable to avoid data loss"
      exit 5
    fi
  fi

  # ---- D3. 禁用 memory provider（只操作 memory: 段落） ----
  if [ -f "$HERMES_CONFIG" ] && grep -q "^memory:" "$HERMES_CONFIG" 2>/dev/null; then
    if sed -n '/^memory:/,/^[a-zA-Z]/p' "$HERMES_CONFIG" | grep -q "provider:.*memory_tencentdb"; then
      cp "$HERMES_CONFIG" "${HERMES_CONFIG}.bak.$(date +%Y%m%d%H%M%S)"
      sed -i '/^memory:/,/^[a-zA-Z]/{/^[[:space:]]*provider:[[:space:]]*memory_tencentdb/d}' "$HERMES_CONFIG"
      log "memory provider removed from config.yaml (within memory: section)"
    fi
  fi

  # ---- D4. 清理 Pro 配置（回退 storeBackend=sqlite） ----
  if [ "$clear_pro_config" = "true" ]; then
    CTL_SCRIPT="$TDAI_INSTALL_DIR/scripts/memory-tencentdb-ctl.sh"
    if [ -f "$CTL_SCRIPT" ]; then
      log "clearing Pro VDB config via ctl.sh"
      bash "$CTL_SCRIPT" --hermes config vdb-off --purge-creds 2>&1 || log "WARN: config vdb-off failed (non-fatal)"
    else
      log "WARN: ctl.sh not found ($CTL_SCRIPT), skip Pro config cleanup"
    fi
  fi

  # ---- D5. 停止 Gateway 进程 ----
  gateway_pid=$(ss -tlnp 2>/dev/null | grep ':8420 ' | sed -n 's/.*pid=\([0-9]*\).*/\1/p')
  if [ -n "$gateway_pid" ]; then
    log "killing Gateway process (pid=$gateway_pid) on port 8420..."
    kill "$gateway_pid" 2>/dev/null || true
    sleep 1
    kill -0 "$gateway_pid" 2>/dev/null && kill -9 "$gateway_pid" 2>/dev/null || true
    log "Gateway process stopped"
  else
    log "Gateway not running on port 8420, skip"
  fi

  # ---- D6. 重启 Hermes ----
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

  echo "memory_tdai disabled"
  exit 0
fi

# ==================== OpenClaw 分支（原有逻辑不变） ====================

plugin_id="${plugin_fullname##*/}"
if [ "$plugin_id" = "$plugin_fullname" ]; then
  plugin_id="${plugin_fullname#@*/}"
fi
config_file="$HOME/.openclaw/openclaw.json"
# OpenClaw 分支统一的本地路径变量。后续 step 5 / step 9 都会复用，
# 必须在 set -uo pipefail 下保证非空，避免 "unbound variable" 退出。
openclaw_home="$HOME/.openclaw"

# ========== 1. 前置检查 ==========
if ! command -v jq >/dev/null 2>&1; then
  log "ERROR: jq not found"
  exit 1
fi

# ========== 2. 读取当前状态 ==========
if [ ! -f "$config_file" ]; then
  log "config file not found, nothing to disable"
  echo "memory_tdai not installed, skip"
  exit 0
fi

installed=false
enabled=false

entry=$(jq -r --arg id "$plugin_id" '.plugins.entries[$id] // empty' "$config_file" 2>/dev/null)
if [ -n "$entry" ]; then
  installed=true
  if echo "$entry" | jq -e '.enabled == true' >/dev/null 2>&1; then
    enabled=true
  fi
fi
log "installed=$installed, enabled=$enabled, clear_pro_config=$clear_pro_config, skip_export=$skip_export"

# ========== 3. 未安装 → 幂等跳过 ==========
if [ "$installed" = false ]; then
  log "plugin not installed, nothing to disable"
  echo "memory_tdai not installed, skip"
  exit 0
fi

# ========== 4. 已禁用 → 幂等跳过（Pro→OFF 时仍需继续导出和清理配置） ==========
if [ "$enabled" = false ] && [ "$clear_pro_config" != "true" ]; then
  log "plugin already disabled, nothing to do"
  echo "memory_tdai already disabled"
  exit 0
fi
if [ "$enabled" = false ]; then
  log "plugin already disabled, but clear_pro_config=true, continue to export and clean up"
fi

# ========== 5. Pro→OFF：导出 VDB 数据到龙虾本地（hatchery 已判定 skip_export） ==========
if [ "$clear_pro_config" = "true" ] && [ "$skip_export" = "true" ]; then
  log "WARN: skip_export=true (hatchery 已预检 VDB 网络不通)，跳过 VDB 数据 export"
  log "WARN: remote VDB data on $vdb_endpoint/$vdb_database will be DELETED by subsequent DeleteMemSpace"
elif [ "$clear_pro_config" = "true" ]; then
  log "Pro→OFF: exporting VDB data to local ..."

  # export-tencent-vdb 工具路径（预编译 JS，不需要 tsx）
  export_bin="$plugin_root/bin/export-tencent-vdb.mjs"
  plugin_data_dir="$openclaw_home/memory-tdai"
  export_output_dir="$plugin_data_dir/vdb-export-$(date +%Y%m%d_%H%M%S)"

  if [ -f "$export_bin" ]; then
    log "calling export tool: $export_bin"
    log "output dir: $export_output_dir"

    export_args=(
      --url "$vdb_endpoint"
      --username "$vdb_username"
      --api-key "$vdb_api_key"
      --database "$vdb_database"
      -o "$export_output_dir"
    )

    if node "$export_bin" "${export_args[@]}" 2>&1; then
      log "VDB data export completed successfully"
      # 记录导出路径到状态文件，便于后续恢复或排查
      echo "$export_output_dir" > "$plugin_data_dir/.last_vdb_export_path"
    else
      export_exit=$?
      log "ERROR: VDB data export failed with exit code $export_exit, aborting disable"
      exit "$export_exit"
    fi
  else
    log "ERROR: export-tencent-vdb tool not found at $export_bin, aborting disable to avoid data loss"
    exit 5
  fi
fi

# ========== 6. 禁用插件（已禁用时跳过） ==========
if [ "$enabled" = true ]; then
  log "disabling plugin via jq ..."
  cp "$config_file" "${config_file}.bak.$(date +%Y%m%d%H%M%S)"

  jq --arg id "$plugin_id" '
    .plugins.entries[$id].enabled = false
  ' "$config_file" > /tmp/_openclaw_disable_tmp.json

  if [ ! -s /tmp/_openclaw_disable_tmp.json ]; then
    log "ERROR: jq output empty, config not modified"
    exit 3
  fi
  mv /tmp/_openclaw_disable_tmp.json "$config_file"
  log "config updated: enabled=false"

  # ========== 7. 验证禁用 ==========
  verify=$(jq -r --arg id "$plugin_id" '.plugins.entries[$id].enabled' "$config_file" 2>/dev/null)
  if [ "$verify" != "false" ]; then
    log "ERROR: verification failed, enabled=$verify"
    exit 4
  fi
else
  log "plugin already disabled, skip disable step"
fi

# ========== 8. 清理 Pro 配置（仅 Pro→OFF 时） ==========
# Pro 模式会用到 storeBackend / tcvdb / bm25；关闭时回退 storeBackend 并清理 tcvdb、bm25，保留 embedding
if [ "$clear_pro_config" = "true" ]; then
  log "clearing Pro VDB config from plugin settings ..."

  jq --arg id "$plugin_id" '
    .plugins.entries[$id].config.storeBackend = "sqlite" |
    del(.plugins.entries[$id].config.tcvdb) |
    del(.plugins.entries[$id].config.bm25)
  ' "$config_file" > /tmp/_openclaw_clear_pro_tmp.json

  if [ ! -s /tmp/_openclaw_clear_pro_tmp.json ]; then
    log "WARN: jq clear Pro config output empty, skip"
  else
    mv /tmp/_openclaw_clear_pro_tmp.json "$config_file"
    log "Pro config cleared: storeBackend=sqlite, tcvdb/bm25 removed"
  fi
fi

# ========== 9. 关闭 Offload（仅 OpenClaw，Pro→OFF 时清理 offload 配置） ==========
# 调用插件提供的 setup-offload.sh --disable（随插件安装到 CVM）。
if [ "$agent_type" = "openclaw" ] && [ -f "$config_file" ]; then
  OFFLOAD_SCRIPT="$plugin_root/scripts/setup-offload.sh"
  if [ -f "$OFFLOAD_SCRIPT" ]; then
    log "关闭 offload ..."
    if bash "$OFFLOAD_SCRIPT" --disable 2>&1; then
      log "offload 已关闭"
    else
      log "WARN: offload 关闭失败（非阻断）"
    fi
  else
    log "setup-offload.sh 不存在，跳过 offload 关闭"
  fi
fi

# ========== 10. 重启 gateway ==========
if command -v openclaw >/dev/null 2>&1; then
  log "restarting openclaw-gateway via openclaw gateway restart ..."
  openclaw gateway restart 2>&1 || log "WARN: gateway restart failed (non-fatal)"
else
  log "WARN: openclaw command not found, skip restart"
fi

echo "memory_tdai disabled"
