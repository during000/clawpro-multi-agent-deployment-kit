#!/bin/bash
# ensure_memory_plugin.sh — 确保记忆插件已安装、来源为 npm/archive、且版本不低于 min_version。
#
# 核心流程：
#   0. 进程互斥锁（flock）：避免 hatchery 重试 / 多 job 并发同时触发 ensure，导致并发改写 openclaw.json
#   1. npm view <plugin>@<dist_tag> version → 解析为具体版本号（latest / beta 等）
#   2. 未安装 → install_plugin（优先 openclaw 原生安装，429 降级走 npm tgz 本地安装）
#   3. 已安装但 source 非受管（非 npm/archive） → 备份配置 → uninstall → install_plugin → 恢复配置
#   4. 已安装且 source 为受管 → 按 dist_tag 分支检查版本：
#        - dist_tag != latest（开发调试）：current 必须严格等于 npm_target_ver
#        - dist_tag == latest（生产）：current >= min_version 即视为就绪
#   5. 不满足 → upgrade_plugin
#
# 安装降级策略（应对 ClawHub 429 限流）：
#   优先: openclaw plugins install @pkg@ver（走 ClawHub→npm 解析链）
#   降级: npm pack @pkg@ver 下载 tgz → openclaw plugins install ./xxx.tgz（跳过 ClawHub）
#
# 退出码约定：
#   0  — 成功
#   2  — 安装/升级失败
#   75 — 被互斥锁挡掉，可由 caller 退避后重试（EX_TEMPFAIL）
#
# 由管控通过 TAT 调用，作为 switch_free / switch_pro 的前置步骤。
set -uo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"
export NO_COLOR=1

plugin_fullname="{{plugin}}"
min_version="{{min_version}}"
dist_tag="{{dist_tag}}"

# dist_tag 兼容：旧调用方未传 → 占位符保持原样未被 TAT 替换 → 兜底 latest。
#
# 注意：判断"占位符未被替换"时不能直接写 `"$dist_tag" = "{{dist_tag}}"`，
# 因为脚本里所有出现 {{dist_tag}} 的地方都会被 TAT 字符串替换，导致条件永真。
# 这里通过字符串拼接构造一个不会被 TAT 识别的 marker。
_unsubst_marker="{{""dist_tag""}}"
if [ -z "$dist_tag" ] || [ "$dist_tag" = "$_unsubst_marker" ]; then
  dist_tag="latest"
fi
unset _unsubst_marker

# 提取短名：@scope/name → name
plugin_id="${plugin_fullname##*/}"
if [ "$plugin_id" = "$plugin_fullname" ]; then
  plugin_id="${plugin_fullname#@*/}"
fi

config_file="$HOME/.openclaw/openclaw.json"

log() { echo "[ensure_plugin] $*"; }

# ========== 0. 进程互斥锁 ==========
# 防止 hatchery dispatcher 在 job 退避重试 / 多 job 并发时同时触发多次 ensure，
# 引发 openclaw 的 ConfigMutationConflictError、临时目录 stale module 加载等问题。
#
# 行为：
#   - flock -w 60: 等待最多 60s（覆盖一次普通 install 的 P95 耗时 ~35s）
#   - 等不到锁 → exit 75 (EX_TEMPFAIL)，让 hatchery 走自身退避重试机制
#     （TAT 拿不到 exit code 数值，任何非 0 都报 FAILED，hatchery 会按 5s/30s/180s 退避重新调度）
#   - 锁文件放在 ~/.openclaw/.locks/ 而非 /tmp，避免 systemd-tmpfiles 周期清理与跨用户权限问题
#   - 写 holder 旁路文件，便于排障定位锁持有者
LOCK_DIR="$HOME/.openclaw/.locks"
mkdir -p "$LOCK_DIR" 2>/dev/null || true
LOCK_FILE="$LOCK_DIR/ensure_memory_plugin_${plugin_id}.lock"

# 用 fd 9 关联锁文件；进程退出（含被 kill -9）fd 关闭，锁自动释放，不会泄露
exec 9>"$LOCK_FILE"
if ! flock -w 60 9; then
  holder=$(cat "${LOCK_FILE}.holder" 2>/dev/null || echo "unknown")
  log "ERROR: another ensure_memory_plugin is running (holder=$holder), waited 60s with no luck"
  echo "BUSY: another ensure_memory_plugin instance is running, retry later"
  exit 75
fi

# 拿到锁后立即记录持有者信息（pid + 启动时间 + 主机名），方便人工排障
echo "pid=$$ start=$(date -u +%FT%TZ) host=$(hostname 2>/dev/null || echo unknown)" > "${LOCK_FILE}.holder"
trap "rm -f '${LOCK_FILE}.holder'" EXIT
log "lock acquired (lockfile=$LOCK_FILE)"

# is_rate_limited — 检查命令输出是否包含 ClawHub 429 限流关键字
is_rate_limited() {
  echo "$1" | grep -qiE '429|[Rr]ate limit'
}

# npm_download_tgz — 通过 npm pack 下载指定版本的 tgz 到临时目录，返回文件路径。
# 完全走 npm registry，不经过 ClawHub，绕开限流。
# 用法: npm_download_tgz <package@version>
# 成功时 tgz 路径写入 stdout；失败返回非零。
npm_download_tgz() {
  local pkg_spec="$1"
  local tmp_dir="/tmp/_plugin_tgz_$$"
  mkdir -p "$tmp_dir"

  # 注意：本函数通过 stdout 返回 tgz 路径，所有日志必须走 stderr（>&2）
  log "[降级] npm pack ${pkg_spec} ..." >&2
  local pack_output
  # npm pack 会在当前目录下载 tgz，文件名打到 stdout
  pack_output=$(cd "$tmp_dir" && npm pack "$pkg_spec" 2>&1)
  local rc=$?

  if [ $rc -ne 0 ]; then
    log "[降级] npm pack 失败: $pack_output" >&2
    rm -rf "$tmp_dir"
    return 1
  fi

  # npm pack 输出最后一行是文件名（如 tencentdb-agent-memory-memory-tencentdb-0.2.3.tgz）
  local tgz_name
  tgz_name=$(echo "$pack_output" | tail -1)
  local tgz_path="${tmp_dir}/${tgz_name}"

  if [ ! -f "$tgz_path" ]; then
    log "[降级] tgz 文件未找到: $tgz_path (npm pack output: $pack_output)" >&2
    rm -rf "$tmp_dir"
    return 1
  fi

  log "[降级] tgz 下载成功: $tgz_path ($(du -h "$tgz_path" | cut -f1))" >&2
  echo "$tgz_path"
}

# install_plugin — 安装插件，优先走 openclaw 原生安装，429 降级走本地 tgz。
# 用法: install_plugin [--force]
# 调用前需确保 plugin_fullname、npm_latest_ver 已设置。
install_plugin() {
  local force_flag="${1:-}"
  local pkg_spec="${plugin_fullname}@${npm_latest_ver}"

  # --- 优先方案：openclaw 原生安装 ---
  log "installing ${pkg_spec} (openclaw native) ..."
  local output
  output=$(openclaw plugins install "${pkg_spec}" $force_flag 2>&1)
  local rc=$?
  echo "$output"

  if [ $rc -eq 0 ]; then
    return 0
  fi

  # --- 非限流错误：直接失败 ---
  if ! is_rate_limited "$output"; then
    return $rc
  fi

  # --- 降级方案：npm pack + 本地 tgz 安装 ---
  log "[降级] ClawHub 429 限流，切换到 npm tgz 本地安装"
  local tgz_path
  tgz_path=$(npm_download_tgz "$pkg_spec") || return 1

  log "[降级] openclaw plugins install ${tgz_path} ${force_flag} ..."
  output=$(openclaw plugins install "$tgz_path" $force_flag 2>&1)
  rc=$?
  echo "$output"

  # 清理 tgz 临时目录
  local tgz_dir
  tgz_dir=$(dirname "$tgz_path")
  rm -rf "$tgz_dir"

  return $rc
}

# upgrade_plugin — 升级插件，根据当前 source 选择策略。
# source=npm → openclaw plugins update（若 429 则降级 tgz + install --force）
# source=archive → 直接走 npm tgz + install --force（update 不支持 archive 来源）
# 调用前需确保 plugin_fullname、npm_latest_ver、install_source 已设置。
upgrade_plugin() {
  local pkg_spec="${plugin_fullname}@${npm_latest_ver}"

  if [ "$install_source" = "archive" ]; then
    # archive 来源：update 不适用，走 tgz 重装
    log "source=archive, 走 npm tgz + install --force 升级"
    local tgz_path
    tgz_path=$(npm_download_tgz "$pkg_spec") || return 1

    log "[降级] openclaw plugins install ${tgz_path} --force ..."
    local output
    output=$(openclaw plugins install "$tgz_path" --force 2>&1)
    local rc=$?
    echo "$output"

    local tgz_dir
    tgz_dir=$(dirname "$tgz_path")
    rm -rf "$tgz_dir"
    return $rc
  fi

  # source=npm → 优先 openclaw plugins update
  log "updating ${pkg_spec} (openclaw native) ..."
  local output
  output=$(openclaw plugins update "${pkg_spec}" 2>&1)
  local rc=$?
  echo "$output"

  if [ $rc -eq 0 ]; then
    return 0
  fi

  # 非限流错误：直接失败
  if ! is_rate_limited "$output"; then
    return $rc
  fi

  # 降级：npm tgz + install --force
  log "[降级] ClawHub 429 限流，切换到 npm tgz + install --force 升级"
  local tgz_path
  tgz_path=$(npm_download_tgz "$pkg_spec") || return 1

  log "[降级] openclaw plugins install ${tgz_path} --force ..."
  output=$(openclaw plugins install "$tgz_path" --force 2>&1)
  rc=$?
  echo "$output"

  local tgz_dir
  tgz_dir=$(dirname "$tgz_path")
  rm -rf "$tgz_dir"
  return $rc
}

log "ensure_memory_plugin started"
log "plugin=$plugin_fullname, min_version=$min_version"

# ========== 1. 前置检查 ==========
if ! command -v jq >/dev/null 2>&1; then
  log "ERROR: jq not found"
  exit 1
fi
if ! command -v openclaw >/dev/null 2>&1; then
  log "ERROR: openclaw command not found"
  exit 1
fi
if ! command -v npm >/dev/null 2>&1; then
  log "ERROR: npm command not found"
  exit 1
fi

# ========== 2. 获取 npm 目标版本号 ==========
# 用 dist_tag 解析为具体版本号（如 beta→0.3.0-beta.1），
# 后续 install/update 均走 @<具体版本号>，避免每次都升级带来的反复重启。
#
# 注意：dist_tag=latest 时 NOT 直接取 npm latest 版本！
# 因为 npm latest 未来可能指向不兼容的新大版本（如 v1.1.0），
# 而生产环境用户只能使用 v1.0.x 系列，所以 dist_tag=latest 时
# 通过版本号过滤取 1.0.x 范围内的最高版本，而不是 latest tag 指向的版本。
# dist_tag=beta 时仍然严格跟随 beta tag 指向的最新预发布版本。
log "fetching version for dist-tag '$dist_tag' from npm ..."
npm_latest_ver=""

if [ "$dist_tag" = "latest" ]; then
  # 生产模式：取 1.0.x 最高版本，不跟随 npm latest tag（避免跳到 v1.1.0 等不兼容版本）
  log "dist_tag=latest, querying 1.0.x max version from npm ..."
  for attempt in 1 2 3; do
    npm_latest_ver=$(npm view "${plugin_fullname}" versions --json 2>/dev/null \
      | jq -r '[.[] | select(test("^1\\.0\\.[0-9]+$"))] | sort_by(. | split(".") | map(tonumber)) | last' 2>/dev/null || true)
    if [ -n "$npm_latest_ver" ] && [ "$npm_latest_ver" != "null" ]; then
      break
    fi
    log "npm view versions attempt $attempt failed, retrying ..."
    sleep 2
  done
  log "1.0.x max version from npm: $npm_latest_ver"
else
  # 非 latest（如 beta）：严格跟随对应 dist-tag 指向的版本
  for attempt in 1 2 3; do
    npm_latest_ver=$(npm view "${plugin_fullname}@${dist_tag}" version 2>/dev/null || true)
    if [ -n "$npm_latest_ver" ]; then
      break
    fi
    log "npm view attempt $attempt failed, retrying ..."
    sleep 2
  done
fi

if [ -z "$npm_latest_ver" ] || [ "$npm_latest_ver" = "null" ]; then
  log "ERROR: failed to get version for dist-tag '$dist_tag' from npm after 3 attempts"
  exit 2
fi
log "npm target version (dist-tag=$dist_tag): $npm_latest_ver"

# ========== 3. 版本比较函数 ==========
# semver_compare: 比较两个 semver 版本号
# 返回: 0=相等, 1=左>右, 2=左<右
semver_compare() {
  local ver1="$1" ver2="$2"
  if [ "$ver1" = "$ver2" ]; then
    return 0
  fi
  # 去掉 v 前缀
  ver1="${ver1#v}"
  ver2="${ver2#v}"

  local IFS='.'
  local -a v1=($ver1) v2=($ver2)

  for i in 0 1 2; do
    local n1="${v1[$i]:-0}"
    local n2="${v2[$i]:-0}"
    # 去掉 prerelease 部分（如 1.0.0-beta）
    n1="${n1%%-*}"
    n2="${n2%%-*}"
    if [ "$n1" -gt "$n2" ] 2>/dev/null; then
      return 1
    elif [ "$n1" -lt "$n2" ] 2>/dev/null; then
      return 2
    fi
  done
  return 0
}

# version_gte: 判断 $1 >= $2
version_gte() {
  semver_compare "$1" "$2"
  local rc=$?
  [ $rc -eq 0 ] || [ $rc -eq 1 ]
}

# is_managed_source — 判断 install source 是否为受管来源（npm 或 archive）
is_managed_source() {
  [ "$1" = "npm" ] || [ "$1" = "archive" ]
}

# ========== 4. 检查安装状态 ==========
installed=false
install_source=""
current_version=""

if [ -f "$config_file" ]; then
  entry=$(jq -r --arg id "$plugin_id" '.plugins.entries[$id] // empty' "$config_file" 2>/dev/null)
  if [ -n "$entry" ]; then
    installed=true
  fi
  install_source=$(jq -r --arg id "$plugin_id" '.plugins.installs[$id].source // empty' "$config_file" 2>/dev/null)
  current_version=$(jq -r --arg id "$plugin_id" '.plugins.installs[$id].version // empty' "$config_file" 2>/dev/null)

  # 兼容 openclaw >= 4.25：installs 配置块位置调整，旧路径读取为空时使用安全默认值。
  # 此时插件版本一定 >= 0.3.4，source 一定为 npm。
  if [ "$installed" = true ] && [ -z "$install_source" ]; then
    install_source="npm"
    log "WARN: plugins.installs[$plugin_id].source not found, using default: npm (openclaw >= 4.25 compat)"
  fi
  if [ "$installed" = true ] && [ -z "$current_version" ]; then
    current_version="0.3.4"
    log "WARN: plugins.installs[$plugin_id].version not found, using default: 0.3.4 (openclaw >= 4.25 compat)"
  fi
fi
log "installed=$installed, source=$install_source, current_version=$current_version"

# ========== 5. 未安装 → 全新安装 ==========
if [ "$installed" = false ]; then
  ext_dir="$HOME/.openclaw/extensions/$plugin_id"

  if [ -d "$ext_dir" ]; then
    bak_dir="$HOME/.openclaw/extensions_bak/${plugin_id}.bak.$(date +%Y%m%d%H%M%S)"
    mkdir -p "$HOME/.openclaw/extensions_bak"
    log "found stale extension directory, moving to $bak_dir"
    mv "$ext_dir" "$bak_dir"
  fi

  if ! install_plugin; then
    log "ERROR: plugin install failed"
    exit 2
  fi
  log "plugin installed successfully (version: $npm_latest_ver)"
  echo "ok"
  exit 0
fi

# ========== 6. 已安装但 source 非受管 → 备份配置 → 重装 → 恢复配置 ==========
if ! is_managed_source "$install_source"; then
  log "source is '$install_source' (not npm/archive), need to reinstall"

  # 备份插件配置
  config_backup="/tmp/_plugin_config_bak_${plugin_id}_$$.json"
  jq --arg id "$plugin_id" '{
    entry: .plugins.entries[$id],
    install: .plugins.installs[$id],
    slots: .plugins.slots,
    allow: .plugins.allow
  }' "$config_file" > "$config_backup" 2>/dev/null || true
  log "plugin config backed up to $config_backup"

  # 卸载旧安装
  log "uninstalling plugin (keep files) ..."
  openclaw plugins uninstall "$plugin_id" --force 2>&1 || true

  # 重新安装（优先 openclaw 原生，429 降级 tgz）
  if ! install_plugin --force; then
    log "ERROR: reinstall failed"
    rm -f "$config_backup"
    exit 2
  fi

  # 恢复配置：恢复 entry.config（业务配置如 storeBackend/tcvdb）、slots、allow
  if [ -f "$config_backup" ] && [ -s "$config_backup" ]; then
    log "restoring plugin config from backup ..."
    backup_entry_config=$(jq '.entry.config // empty' "$config_backup" 2>/dev/null)
    backup_slots=$(jq '.slots // empty' "$config_backup" 2>/dev/null)
    backup_allow=$(jq '.allow // empty' "$config_backup" 2>/dev/null)

    tmp_file="${config_file}.tmp"
    restore_ok=true

    # 恢复 entry.config（保留安装后生成的其他字段如 enabled）
    if [ -n "$backup_entry_config" ] && [ "$backup_entry_config" != "null" ] && [ "$backup_entry_config" != "" ]; then
      jq_err=$(jq --arg id "$plugin_id" --argjson cfg "$backup_entry_config" '
        .plugins.entries[$id].config = (.plugins.entries[$id].config // {} | . * $cfg)
      ' "$config_file" > "$tmp_file" 2>&1)
      if [ $? -eq 0 ] && [ -s "$tmp_file" ]; then
        mv "$tmp_file" "$config_file"
      else
        log "WARN: failed to restore entry.config: $jq_err"
        rm -f "$tmp_file"
        restore_ok=false
      fi
    fi

    # 恢复 slots
    if [ -n "$backup_slots" ] && [ "$backup_slots" != "null" ] && [ "$backup_slots" != "" ]; then
      jq_err=$(jq --argjson slots "$backup_slots" '
        .plugins.slots = $slots
      ' "$config_file" > "$tmp_file" 2>&1)
      if [ $? -eq 0 ] && [ -s "$tmp_file" ]; then
        mv "$tmp_file" "$config_file"
      else
        log "WARN: failed to restore slots: $jq_err"
        rm -f "$tmp_file"
      fi
    fi

    # 恢复 allow
    if [ -n "$backup_allow" ] && [ "$backup_allow" != "null" ] && [ "$backup_allow" != "" ]; then
      jq_err=$(jq --argjson allow "$backup_allow" '
        .plugins.allow = $allow
      ' "$config_file" > "$tmp_file" 2>&1)
      if [ $? -eq 0 ] && [ -s "$tmp_file" ]; then
        mv "$tmp_file" "$config_file"
      else
        log "WARN: failed to restore allow: $jq_err"
        rm -f "$tmp_file"
      fi
    fi

    if [ "$restore_ok" = true ]; then
      log "plugin config restored successfully"
    else
      log "WARN: some config fields could not be restored, check logs above"
    fi
  fi

  rm -f "$config_backup"
  log "reinstalled (version: $npm_latest_ver), config restored"
  echo "ok"
  exit 0
fi

# ========== 7. source 为受管（npm/archive），检查版本 ==========
# 优先级（自上而下）：
#   1. dist_tag != latest（开发调试模式）：忽略 min_version 短路，严格跟随 npm_target_ver
#      → 否则 beta 升到 0.3.0-beta.2 时不会被自动同步上去
#   2. 生产模式（dist_tag=latest）：current_version >= min_version 则视为就绪
#   3. 兜底：current_version 与 npm_target_ver 严格相等才视为就绪
#
# 注意：min_version 占位符未替换的判断同样要用字符串拼接 marker，
# 避免 "$min_version" != "{{min_version}}" 中的 {{min_version}} 被 TAT 替换。
_unsubst_minver_marker="{{""min_version""}}"
if [ "$dist_tag" != "latest" ]; then
  if [ "$current_version" = "$npm_latest_ver" ]; then
    log "dist-tag=$dist_tag, current $current_version == target $npm_latest_ver, plugin is up to date"
    echo "ok"
    exit 0
  fi
  log "dist-tag=$dist_tag, current_version=$current_version, target=$npm_latest_ver, need upgrade (min_version short-circuit ignored)"
elif [ -n "$min_version" ] && [ "$min_version" != "$_unsubst_minver_marker" ] && [ -n "$current_version" ]; then
  if version_gte "$current_version" "$min_version"; then
    log "version $current_version >= min_version $min_version, plugin is up to date"
    echo "ok"
    exit 0
  fi
  log "version $current_version < min_version $min_version, need upgrade"
else
  # 无 min_version 要求或无法获取当前版本时，仍检查是否为最新
  if [ "$current_version" = "$npm_latest_ver" ]; then
    log "already at latest version $current_version"
    echo "ok"
    exit 0
  fi
  log "current_version=$current_version, npm_latest=$npm_latest_ver, need upgrade"
fi
unset _unsubst_minver_marker

# ========== 8. 升级到最新版 ==========
if ! upgrade_plugin; then
  log "ERROR: plugin upgrade failed"
  exit 2
fi

# 验证升级后的版本
new_version=""
if [ -f "$config_file" ]; then
  new_version=$(jq -r --arg id "$plugin_id" '.plugins.installs[$id].version // empty' "$config_file" 2>/dev/null)
  # 兼容 openclaw >= 4.25：旧路径读不到时回退 npm_latest_ver
  [ -z "$new_version" ] && new_version="$npm_latest_ver"
fi
log "plugin upgraded successfully (version: ${new_version:-unknown})"
echo "ok"

