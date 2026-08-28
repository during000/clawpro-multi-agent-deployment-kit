#!/bin/bash
# memory_tdai_switch_pro.sh — 切换到 Pro 模式
# 支持 OpenClaw / Hermes 两种智能体类型。
# 由管控通过 TAT 调用。
#
# 纯业务逻辑：
#   1. 前置检查（插件应已由 ensure_memory_plugin.sh 安装就绪）
#   2. 根据 previous_plan 决定路径：
#      - FREE → PRO：调用 migrate-sqlite-to-tcvdb（数据迁移 + 配置下发 + manifest 更新）
#      - OFF  → PRO：仅下发 VDB 配置 + 更新 manifest（跳过数据迁移）
#   3. 启用插件 + 重启
#
# 插件安装/升级由 ensure_memory_plugin.sh 负责，应在本脚本之前调用。
set -uo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"
export NO_COLOR=1

# %INCLUDE% lib_acli_compat.sh

plugin_fullname="{{plugin}}"
plugin_id="${plugin_fullname##*/}"
if [ "$plugin_id" = "$plugin_fullname" ]; then
  plugin_id="${plugin_fullname#@*/}"
fi

previous_plan="{{previous_plan}}"
vdb_endpoint="{{vdb_endpoint}}"
vdb_database="{{vdb_database}}"
vdb_api_key="{{vdb_api_key}}"
vdb_username="{{vdb_username}}"
embedding_model="{{embedding_model}}"
job_id="{{job_id}}"
agent_type="{{agent_type}}"
# Offload 参数（仅 OpenClaw Pro 使用，由 Go 层按地域映射传入）
offload_backend_url="{{offload_backend_url}}"
offload_user_id="{{offload_user_id}}"

log() { echo "[switch_pro] $*"; }

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

log "switch_pro started"
log "agent_type=$agent_type, plugin=$plugin_fullname, previous_plan=$previous_plan"
log "vdb_endpoint=$vdb_endpoint, vdb_database=$vdb_database, vdb_username=$vdb_username, embedding_model=$embedding_model"

# ==================== 分支入口 ====================
if [ "$agent_type" = "hermes" ]; then
  # ==================== Hermes 分支 ====================

  if ! command -v jq >/dev/null 2>&1; then
    log "ERROR: jq not found"
    exit 1
  fi

  # ---- HP0. 加载 Gateway 环境变量 ----
  if [ -f /etc/profile.d/memory-tencentdb-env.sh ]; then
    source /etc/profile.d/memory-tencentdb-env.sh
    log "Gateway env loaded"
  else
    log "WARN: /etc/profile.d/memory-tencentdb-env.sh not found"
  fi

  # ---- HP1. 检测插件是否已安装 ----
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

  HERMES_CONFIG="$HOME/.hermes/config.yaml"
  plugin_data_dir="$HOME/.memory-tencentdb/memory-tdai"
  CTL_SCRIPT="$TDAI_INSTALL_DIR/scripts/memory-tencentdb-ctl.sh"

  # ---- HP2. 清理旧 YAML 配置 ----
  old_yaml_config="$TDAI_INSTALL_DIR/tdai-gateway.yaml"
  if [ -f "$old_yaml_config" ]; then
    log "found legacy tdai-gateway.yaml, backing up..."
    mv "$old_yaml_config" "${old_yaml_config}.bak.$(date +%Y%m%d%H%M%S)"
  fi

  # ---- HP3. 数据迁移（根据 previous_plan） ----
  if [ "$previous_plan" = "FREE" ]; then
    # ── FREE → PRO：迁移 SQLite 数据到 TCVDB ──
    log "FREE → PRO: calling migrate-sqlite-to-tcvdb (with data migration)"

    migrate_args=(
      --plugin-data-dir "$plugin_data_dir"
      --openclaw-config-path "/dev/null"
      --no-apply-config
      --no-rewrite-manifest
      --tcvdb-url "$vdb_endpoint"
      --tcvdb-username "$vdb_username"
      --tcvdb-api-key "$vdb_api_key"
      --tcvdb-database "$vdb_database"
      --tcvdb-embedding-model "$embedding_model"
      --no-fail-if-target-nonempty
      --yes
    )

    if [ -n "$job_id" ] && [ "$job_id" != "{{job_id}}" ]; then
      migrate_args+=(--job-id "$job_id")
    fi

    if (cd "$TDAI_INSTALL_DIR" && node ./bin/migrate-sqlite-to-tcvdb.mjs "${migrate_args[@]}" 2>&1); then
      log "migration completed successfully"
    else
      migrate_exit=$?
      log "ERROR: migration failed with exit code $migrate_exit"
      exit "$migrate_exit"
    fi
  else
    log "$previous_plan → PRO: config-only mode (skip data migration)"
  fi

  # ---- HP4. 通过 ctl.sh 写入 VDB 配置 ----
  if [ -f "$CTL_SCRIPT" ]; then
    log "writing VDB config via ctl.sh"
    bash "$CTL_SCRIPT" --hermes config vdb \
      --url "$vdb_endpoint" \
      --username "$vdb_username" \
      --api-key "$vdb_api_key" \
      --database "$vdb_database" \
      --embedding-model "$embedding_model" 2>&1 || log "WARN: ctl config vdb failed (non-fatal)"

    # ---- HP5. 通过 ctl.sh 同步 LLM 配置 ----
    # LLM 信息从 ~/.hermes/config.yaml 读取，不依赖 hatchery 传参
    _llm_api_key=""
    _llm_base_url=""
    _llm_model=""
    if [ -f "$HERMES_CONFIG" ]; then
      _llm_api_key=$(sed -n '/^model:/,/^[a-z]/{/^[[:space:]]*api_key:/p}' "$HERMES_CONFIG" | head -1 | sed 's/^[[:space:]]*api_key:[[:space:]]*//' | tr -d '"' | tr -d "'")
      _llm_base_url=$(sed -n '/^model:/,/^[a-z]/{/^[[:space:]]*base_url:/p}' "$HERMES_CONFIG" | head -1 | sed 's/^[[:space:]]*base_url:[[:space:]]*//' | tr -d '"' | tr -d "'")
      _llm_model=$(sed -n '/^model:/,/^[a-z]/{/^[[:space:]]*default:/p}' "$HERMES_CONFIG" | head -1 | sed 's/^[[:space:]]*default:[[:space:]]*//' | tr -d '"' | tr -d "'")
    fi

    if [ -n "$_llm_api_key" ] && [ -n "$_llm_base_url" ] && [ -n "$_llm_model" ]; then
      log "syncing LLM config via ctl.sh: baseUrl=$_llm_base_url, model=$_llm_model"
      bash "$CTL_SCRIPT" --hermes config llm \
        --api-key "$_llm_api_key" \
        --base-url "$_llm_base_url" \
        --model "$_llm_model" 2>&1 || log "WARN: ctl config llm failed (non-fatal)"
    else
      log "WARN: hermes config.yaml missing LLM config, skip sync"
    fi

    # ---- HP6. 启用 memory provider ----
    log "enabling hermes memory provider via ctl.sh"
    bash "$CTL_SCRIPT" --hermes enable-hermes-memory 2>&1 || log "WARN: enable-hermes-memory failed (non-fatal)"
  else
    log "WARN: memory-tencentdb-ctl.sh not found ($CTL_SCRIPT), skip VDB/LLM/memory config"
  fi

  # ---- HP7. 更新 manifest.json ----
  manifest_file="$plugin_data_dir/manifest.json"
  mkdir -p "$plugin_data_dir"
  log "updating manifest at $manifest_file"

  if [ -f "$manifest_file" ]; then
    cp "$manifest_file" "${manifest_file}.migrate.bak"
    manifest_tmp="${manifest_file}.tmp"
    manifest_err=$(
      jq --arg url "$vdb_endpoint" --arg db "$vdb_database" '
        .store = {
          "type": "tcvdb",
          "tcvdbUrl": $url,
          "tcvdbDatabase": $db
        }
      ' "$manifest_file" > "$manifest_tmp" 2>&1
    )
    manifest_rc=$?
    if [ $manifest_rc -ne 0 ] || [ ! -s "$manifest_tmp" ]; then
      log "WARN: jq failed to update manifest (rc=$manifest_rc): ${manifest_err:-<empty>} (non-fatal)"
      rm -f "$manifest_tmp"
    else
      mv "$manifest_tmp" "$manifest_file"
    fi
  else
    cat > "$manifest_file" <<MANIFEST
{
  "version": 1,
  "createdAt": "$(date -u +%Y-%m-%dT%H:%M:%S.000Z)",
  "store": {
    "type": "tcvdb",
    "tcvdbUrl": "$vdb_endpoint",
    "tcvdbDatabase": "$vdb_database"
  },
  "seed": null
}
MANIFEST
  fi
  log "manifest updated"

  # ---- HP9. 杀掉旧 Gateway 进程（让新配置生效） ----
  gateway_pid=$(ss -tlnp 2>/dev/null | grep ':8420 ' | sed -n 's/.*pid=\([0-9]*\).*/\1/p')
  if [ -n "$gateway_pid" ]; then
    log "killing old Gateway process (pid=$gateway_pid) to reload config..."
    kill "$gateway_pid" 2>/dev/null || true
    sleep 1
    kill -0 "$gateway_pid" 2>/dev/null && kill -9 "$gateway_pid" 2>/dev/null || true
    log "old Gateway stopped"
  fi

  # ---- HP10. 重启 Hermes ----
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

  log "switch_pro completed (hermes)"
  echo "switch_pro done"
  exit 0
fi

# ==================== OpenClaw 分支（原有逻辑不变） ====================

config_file="$HOME/.openclaw/openclaw.json"
plugin_data_dir="$HOME/.openclaw/memory-tdai"

# ========== 1. 前置检查 ==========
if ! command -v jq >/dev/null 2>&1; then
  log "ERROR: jq not found"
  exit 1
fi
if ! command -v openclaw >/dev/null 2>&1; then
  log "ERROR: openclaw command not found"
  exit 1
fi
if [ ! -f "$config_file" ]; then
  log "ERROR: openclaw.json not found at $config_file"
  exit 1
fi

# ========== 1b. 获取 OpenClaw 版本号 ==========
# 版本格式为 2026.M.D（如 2026.4.23 / 2026.5.7）
# 优先从 openclaw.json 读取（快），fallback 到 openclaw --version
openclaw_version=""
if [ -f "$config_file" ]; then
  openclaw_version=$(jq -r '.meta.lastTouchedVersion // empty' "$config_file" 2>/dev/null)
fi
if [ -z "$openclaw_version" ]; then
  openclaw_version=$(openclaw --version 2>/dev/null | head -n 1 | sed -E 's/[^0-9]*([0-9]+(\.[0-9]+){1,3}).*/\1/')
fi
oc_major="$(echo "$openclaw_version" | awk -F. '{print $1}')"
oc_minor="$(echo "$openclaw_version" | awk -F. '{print $2}')"
log "openclaw_version=$openclaw_version (major=$oc_major, minor=$oc_minor)"

# ========== 2. 检查插件已安装（由 ensure_memory_plugin.sh 保证） ==========
openclaw_home="{{plugin_root}}"
if [ -z "$openclaw_home" ] || [ ! -d "$openclaw_home" ]; then
  log "ERROR: plugin_root is empty or directory not found: $openclaw_home"
  exit 2
fi

installed=false
if [ -f "$config_file" ]; then
  entry=$(jq -r --arg id "$plugin_id" '.plugins.entries[$id] // empty' "$config_file" 2>/dev/null)
  if [ -n "$entry" ]; then
    installed=true
  fi
fi
log "installed=$installed"

if [ "$installed" = false ]; then
  log "ERROR: plugin not installed. Run ensure_memory_plugin.sh first."
  exit 2
fi

# ========== 3. 检查 migrate 命令可用性 ==========
if ! (cd "$openclaw_home" && npm run migrate-sqlite-to-tcvdb -- --help >/dev/null 2>&1); then
  log "ERROR: migrate-sqlite-to-tcvdb 不可用，请确认 ensure_memory_plugin.sh 已成功执行"
  exit 3
fi
log "migrate-sqlite-to-tcvdb 已就绪"

# ========== 4. 根据 previous_plan 选择执行路径 ==========

if [ "$previous_plan" = "FREE" ]; then
  # ── FREE → PRO：走完整 migrate（数据迁移 + 配置下发 + manifest） ──
  log "FREE → PRO: calling migrate-sqlite-to-tcvdb (with data migration)"

  migrate_args=(
    --plugin-data-dir "$plugin_data_dir"
    --openclaw-config-path "$config_file"
    --tcvdb-url "$vdb_endpoint"
    --tcvdb-username "$vdb_username"
    --tcvdb-api-key "$vdb_api_key"
    --tcvdb-database "$vdb_database"
    --tcvdb-embedding-model "$embedding_model"
    --no-fail-if-target-nonempty
  )

  if [ -n "$job_id" ] && [ "$job_id" != "{{job_id}}" ]; then
    migrate_args+=(--job-id "$job_id")
  fi

  if (cd "$openclaw_home" && npm run migrate-sqlite-to-tcvdb -- "${migrate_args[@]}" 2>&1); then
    log "migration completed successfully"
  else
    migrate_exit=$?
    log "ERROR: migration failed with exit code $migrate_exit"
    exit "$migrate_exit"
  fi

else
  # ── OFF → PRO（或 PRO 重装）：仅下发 VDB 配置 + 更新 manifest，跳过数据迁移 ──
  log "$previous_plan → PRO: config-only mode (skip data migration)"

  # 4a. VDB 配置暂存（稍后在 offload 之后统一写入 openclaw.json，避免被 setup-offload.sh 覆盖）
  config_patch=$(cat <<JSONEOF
{
  "storeBackend": "tcvdb",
  "tcvdb": {
    "url": "$vdb_endpoint",
    "username": "$vdb_username",
    "apiKey": "$vdb_api_key",
    "database": "$vdb_database",
    "alias": "",
    "embeddingModel": "$embedding_model",
    "timeout": 3000
  },
  "bm25": {
    "enabled": true,
    "language": "zh"
  }
}
JSONEOF
  )
  # 标记：OFF→PRO 路径需要在 offload 之后写 VDB config
  _off_to_pro_pending_vdb=true

  # 4b. 更新 manifest.json
  manifest_file="$plugin_data_dir/manifest.json"
  log "updating manifest at $manifest_file"

  mkdir -p "$plugin_data_dir"

  if [ -f "$manifest_file" ]; then
    cp "$manifest_file" "${manifest_file}.migrate.bak"
    manifest_tmp="${manifest_file}.tmp"
    manifest_err=$(
      jq --arg url "$vdb_endpoint" --arg db "$vdb_database" '
        .store = {
          "type": "tcvdb",
          "tcvdbUrl": $url,
          "tcvdbDatabase": $db
        }
      ' "$manifest_file" > "$manifest_tmp" 2>&1
    )
    manifest_rc=$?
    if [ $manifest_rc -ne 0 ] || [ ! -s "$manifest_tmp" ]; then
      log "ERROR: jq failed to update manifest (rc=$manifest_rc): ${manifest_err:-<empty>}"
      rm -f "$manifest_tmp"
      exit 4
    fi
    if ! mv "$manifest_tmp" "$manifest_file"; then
      log "ERROR: failed to replace manifest file with updated result"
      rm -f "$manifest_tmp"
      exit 4
    fi
  else
    cat > "$manifest_file" <<MANIFEST
{
  "version": 1,
  "createdAt": "$(date -u +%Y-%m-%dT%H:%M:%S.000Z)",
  "store": {
    "type": "tcvdb",
    "tcvdbUrl": "$vdb_endpoint",
    "tcvdbDatabase": "$vdb_database"
  },
  "seed": null
}
MANIFEST
  fi
  log "manifest updated"
fi

# ========== 5. 开启 Offload（仅 OpenClaw Pro） ==========
# 注意：offload 必须在写 VDB config / enabled 之前，因为 setup-offload.sh 会覆盖 openclaw.json。
# 等待 openclaw plugins install 的异步 persistPluginInstall 完成后再写配置。
# EnsureMemoryPlugin 调用的 `openclaw plugins install` 会异步执行 persistPluginInstall，
# 将安装记录写入 openclaw.json。如果我们在它完成之前写了 openclaw.json，
# 它后续执行时会基于旧内容覆盖我们的写入（先写后校验，覆盖已发生）。
# 这里等待足够长的时间确保异步操作完成。
log "waiting for async plugin install persistence to complete ..."
sleep 180
# 先暂时不开启offload功能
# log "Step 5: enable offload"
# if [ -n "$offload_backend_url" ]; then
#   OFFLOAD_SCRIPT="$openclaw_home/scripts/setup-offload.sh"
#   if [ -f "$OFFLOAD_SCRIPT" ]; then
#     log "开启 offload: backend=$offload_backend_url, user_id=$offload_user_id"
#     if bash "$OFFLOAD_SCRIPT" --enable \
#         --user-id "$offload_user_id" \
#         --backend-url "$offload_backend_url"; then
#       log "offload 开启成功"
#     else
#       log "WARN: offload 开启失败（非阻断，不影响 Pro 切换）"
#     fi
#   else
#     log "WARN: setup-offload.sh 不存在（$OFFLOAD_SCRIPT），跳过 offload 开启"
#   fi
# else
#   log "offload_backend_url 为空或未配置，跳过 offload 开启"
# fi

# ========== 6. 启用插件 + 写入 VDB 配置（在 offload 之后，避免被 setup-offload.sh 覆盖） ==========
log "enabling plugin and writing final config"

# OFF→PRO 路径：setup-offload.sh 会覆盖 openclaw.json，所以 VDB config 需要在此时写入
if [ "${_off_to_pro_pending_vdb:-}" = "true" ]; then
  # 备份原配置
  backup_file="${config_file}.bak.$(date +%Y%m%d%H%M%S)"
  if ! cp "$config_file" "$backup_file"; then
    log "ERROR: failed to backup config file to $backup_file"
    exit 4
  fi
  log "writing VDB config to $config_file (after offload)"
  tmp_file="${config_file}.tmp"
  jq_stderr=$(
    jq --arg id "$plugin_id" --argjson patch "$config_patch" '
      .plugins.entries[$id].config = (
        (.plugins.entries[$id].config // {}) * $patch
      ) |
      .plugins.entries[$id].enabled = true
    ' "$config_file" > "$tmp_file" 2>&1
  )
  jq_rc=$?
  if [ $jq_rc -ne 0 ] || [ ! -s "$tmp_file" ]; then
    log "ERROR: jq failed to merge VDB config + enable (rc=$jq_rc): ${jq_stderr:-<empty>}"
    rm -f "$tmp_file"
    exit 4
  fi
  if ! mv "$tmp_file" "$config_file"; then
    log "ERROR: failed to replace config file"
    rm -f "$tmp_file"
    exit 4
  fi
  log "VDB config + plugin enabled written successfully"
else
  # FREE→PRO 路径：只需设 enabled=true（VDB config 由 migrate 工具写入）
  enable_tmp="${config_file}.tmp"
  enable_err=$(
    jq --arg id "$plugin_id" '
      .plugins.entries[$id].enabled = true
    ' "$config_file" > "$enable_tmp" 2>&1
  )
  enable_rc=$?
  if [ $enable_rc -ne 0 ] || [ ! -s "$enable_tmp" ]; then
    log "ERROR: jq failed to enable plugin (rc=$enable_rc): ${enable_err:-<empty>}"
    rm -f "$enable_tmp"
    exit 4
  fi
  if ! mv "$enable_tmp" "$config_file"; then
    log "ERROR: failed to replace config file while enabling plugin"
    rm -f "$enable_tmp"
    exit 4
  fi
  log "plugin enabled in config"
fi

# ========== 7. 重启 openclaw 使配置生效 ==========
# 4.x（minor<5）：openclaw gateway restart 会触发 applyPluginAutoEnable → replaceConfigFile，
#   覆盖我们刚写入的配置（bug #67557）。4.x 的 config file watcher 会自动检测
#   openclaw.json 变更并 reload，因此跳过重启即可。
# 5.x（minor>=5）：bug #67557 已修复，使用 openclaw gateway restart 热重载。
#   不能用 systemctl restart（冷重启），冷重启会触发 openclaw plugins install，
#   与脚本修改的 openclaw.json 产生 hash 冲突（ConfigMutationConflictError）。
if [ -n "$oc_minor" ] && [ "$oc_minor" -ge 5 ] 2>/dev/null; then
  if command -v openclaw >/dev/null 2>&1; then
    log "openclaw >= 2026.5.x, restarting via openclaw gateway restart ..."
    openclaw gateway restart 2>&1 || log "WARN: gateway restart failed (non-fatal)"
  else
    log "WARN: openclaw command not found, skip restart"
  fi
else
  log "openclaw < 2026.5.x (version=$openclaw_version), skip restart — relying on config file watcher to auto-reload"
fi

log "switch_pro completed"
echo "switch_pro done"
