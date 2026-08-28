#!/bin/bash
set -euo pipefail
RUNTIME_USER="{{runtime_user}}"
if [ -z "$RUNTIME_USER" ] || [ "$RUNTIME_USER" = "{{runtime_user}}" ]; then
    RUNTIME_USER="${OPENCLAW_RUNTIME_USER:-root}"
fi
if [ "$RUNTIME_USER" != "root" ] && ! id "$RUNTIME_USER" >/dev/null 2>&1; then
    echo "WARN: runtime user '$RUNTIME_USER' 不存在，回退到 root"
    RUNTIME_USER="root"
fi
# resume-after-doctor: 跳过下载/解压，从 doctor 开始
RESUME_AFTER_DOCTOR=false
if [ "{{resume_after_doctor}}" = "true" ]; then
    RESUME_AFTER_DOCTOR=true
fi
TARGET_UID="$(id -u "$RUNTIME_USER" 2>/dev/null || id -u)"
TARGET_GID="$(id -g "$RUNTIME_USER" 2>/dev/null || id -g)"
TARGET_HOME=""
[ -r /etc/passwd ] && TARGET_HOME=$(awk -F: -v u="$RUNTIME_USER" '$1==u{print $6; exit}' /etc/passwd)
[ -z "$TARGET_HOME" ] && TARGET_HOME=$([ "$RUNTIME_USER" = "root" ] && echo "/root" || echo "/home/$RUNTIME_USER")
DEFAULT_OPENCLAW_HOME="${TARGET_HOME}/.openclaw"
export HOME="$TARGET_HOME"
SCRIPT_NAME="restore_post_reinstall"
LOG_DIR="${HOME}/.openclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || LOG_DIR="/tmp"
chmod 700 "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
touch "$LOG_FILE" 2>/dev/null || true
chmod 600 "$LOG_FILE" 2>/dev/null || true
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "日志文件: $LOG_FILE"

fail_exit() {
    local stage="$1" msg="$2"
    echo ""
    echo "✗ [FATAL] 阶段 [$stage] 失败：$msg"
    echo "RESTORE_FAILED:${stage}" > /tmp/restore_status 2>/dev/null || true
    exit 1
}

on_error() {
    local exit_code=$? lineno="${1:-?}"
    if [ "$exit_code" -ne 0 ]; then
        echo ""
        echo "✗ [FATAL] 脚本在第 ${lineno} 行非预期退出（exit=$exit_code）"
        if [ ! -s /tmp/restore_status ] || ! grep -q "^RESTORE_" /tmp/restore_status 2>/dev/null; then
            echo "RESTORE_FAILED:unexpected_error" > /tmp/restore_status 2>/dev/null || true
        fi
    fi
}
trap 'on_error "$LINENO"' ERR

run_as_runtime_user() {
    if [ "$(id -un)" = "$RUNTIME_USER" ]; then
        "$@"; return
    fi
    if command -v runuser >/dev/null 2>&1; then
        runuser -u "$RUNTIME_USER" -- "$@"; return
    fi
    su - "$RUNTIME_USER" -s /bin/bash -c "$(printf '%q ' "$@")"
}

user_systemctl() {
    if [ "$(id -un)" = "$RUNTIME_USER" ]; then
        XDG_RUNTIME_DIR="/run/user/${TARGET_UID}" systemctl --user "$@"; return
    fi
    run_as_runtime_user env XDG_RUNTIME_DIR="/run/user/${TARGET_UID}" systemctl --user "$@"
}

# 精确匹配 node/openclaw 进程，避免误杀 tee/tail/grep 及脚本自身
stop_openclaw_processes() {
    local self_pid=$$ ppid=$PPID
    local pattern='(^|/)node[^ ]* .*(openclaw-gateway|\.openclaw/.*\.(js|cjs|mjs)|/openclaw/dist/)|(^|/)openclaw +(doctor|plugins|gateway|service|start|stop)'
    local pids=""
    if [ "$(id -u)" -eq 0 ] && [ "$RUNTIME_USER" != "root" ]; then
        pids=$(pgrep -u "$RUNTIME_USER" -af "$pattern" 2>/dev/null | awk '{print $1}' || true)
    else
        pids=$(pgrep -af "$pattern" 2>/dev/null | awk '{print $1}' || true)
    fi
    local filtered=""
    for pid in $pids; do
        [ "$pid" = "$self_pid" ] && continue
        [ "$pid" = "$ppid" ] && continue
        filtered="$filtered $pid"
    done
    if [ -z "${filtered// /}" ]; then
        echo "  无运行中的 openclaw 业务进程"; return 0
    fi
    echo "  待终止 PID:$filtered"
    # shellcheck disable=SC2086
    kill -TERM $filtered 2>/dev/null || true
    sleep 2
    local still=""
    for pid in $filtered; do
        kill -0 "$pid" 2>/dev/null && still="$still $pid"
    done
    if [ -n "${still// /}" ]; then
        echo "  强制终止仍存活 PID:$still"
        # shellcheck disable=SC2086
        kill -KILL $still 2>/dev/null || true
    fi
    return 0
}

ARCHIVE_URL="{{url}}"
ARCHIVE_PATH=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --archive|--backup-dir)
            ARCHIVE_PATH="$2"; ARCHIVE_URL=""; shift 2 ;;
        --url)
            ARCHIVE_URL="$2"; shift 2 ;;
        --resume-after-doctor)
            RESUME_AFTER_DOCTOR=true; shift ;;
        -h|--help)
            echo "Usage: $0 (--archive <path.tgz> | --url <smh-url>) [--resume-after-doctor]"; exit 0 ;;
        *)
            echo "Unknown option: $1"; exit 1 ;;
    esac
done

if [ "$RESUME_AFTER_DOCTOR" != "true" ]; then
    if [ -z "$ARCHIVE_PATH" ] && [ -z "$ARCHIVE_URL" ]; then
        echo "Error: url 参数是必需的（TAT 模板变量 {{.url}} 或 --url 参数）"
        exit 1
    fi
fi

echo "=== OpenClaw 重装后数据恢复 ==="
echo "备份压缩包: ${ARCHIVE_URL:-$ARCHIVE_PATH}"

download_from_smh() {
    if [ -z "$ARCHIVE_URL" ]; then return; fi
    echo ""
    echo ">>> [步骤 0/7] 从 SMH 下载备份压缩包"
    local openclaw_home="$DEFAULT_OPENCLAW_HOME"
    local dl_ts
    dl_ts="$(date +%Y%m%d_%H%M%S)"
    local upgrade_dir="$openclaw_home/upgrades/$dl_ts"
    mkdir -p "$upgrade_dir"
    echo "从 SMH 下载备份包..."
    ARCHIVE_PATH="$upgrade_dir/openclaw-backup.tgz"
    # 优先内网下载，失败时剥掉 internal_domain=1 走公网
    if ! curl -fsSL -o "$ARCHIVE_PATH" "$ARCHIVE_URL"; then
        local public_url="$ARCHIVE_URL"
        public_url="${public_url//&internal_domain=1/}"
        public_url="${public_url//\?internal_domain=1&/\?}"
        public_url="${public_url//\?internal_domain=1/}"
        if [ "$public_url" != "$ARCHIVE_URL" ]; then
            echo "内网下载失败，回退到公网域名重试..."
            if ! curl -fsSL -o "$ARCHIVE_PATH" "$public_url"; then
                fail_exit "download" "下载备份包失败（内网+公网均失败）: $ARCHIVE_URL"
            fi
        else
            fail_exit "download" "下载备份包失败: $ARCHIVE_URL"
        fi
    fi
    if [ ! -s "$ARCHIVE_PATH" ]; then
        fail_exit "download" "下载文件为空: $ARCHIVE_PATH"
    fi
    if ! tar -tzf "$ARCHIVE_PATH" >/dev/null 2>&1; then
        fail_exit "download" "下载的压缩包损坏或格式不正确: $ARCHIVE_PATH"
    fi
    echo "✓ 下载完成: $ARCHIVE_PATH ($(du -sh "$ARCHIVE_PATH" | cut -f1))"
    echo "✓ 压缩包已保存到升级记录目录: $upgrade_dir"
}

restore_files() {
    local openclaw_home="$DEFAULT_OPENCLAW_HOME"
    echo ""
    echo ">>> [步骤 2/7] 解压恢复文件到 $openclaw_home、验证恢复结果并修复权限"
    echo "开始解压恢复文件..."
    mkdir -p "$openclaw_home"
    # 插件目录取并集：镜像已预装的加入 exclude 防备份老版本覆盖
    local _plugin_excludes_file="/tmp/openclaw_plugin_excludes_$$.txt"
    : > "$_plugin_excludes_file"
    _collect_plugin_excludes() {
        local rel_root="$1"
        local abs_root="$openclaw_home/$rel_root"
        [ -d "$abs_root" ] || return 0
        local sub
        while IFS= read -r -d '' sub; do
            local name
            name="$(basename "$sub")"
            case "$name" in .*) continue ;; esac
            # 双前缀兼容 GNU/BSD tar
            printf './%s/%s\n%s/%s\n' "$rel_root" "$name" "$rel_root" "$name" >> "$_plugin_excludes_file"
        done < <(find "$abs_root" -mindepth 1 -maxdepth 1 -type d -print0 2>/dev/null)
    }
    _collect_plugin_excludes "extensions"
    _collect_plugin_excludes "npm/node_modules"
    _collect_plugin_excludes "npm/projects"
    _collect_plugin_excludes ".npm/node_modules"
    _collect_plugin_excludes ".npm/projects"
    local _excludes_count=0
    if [ -s "$_plugin_excludes_file" ]; then
        _excludes_count=$(awk '/^\.\//' "$_plugin_excludes_file" | sort -u | wc -l | tr -d ' ')
    fi
    if [ "$_excludes_count" -gt 0 ]; then
        echo "镜像已预装 $_excludes_count 个插件目录，保留镜像版："
        awk '/^\.\//{ sub(/^\.\//, ""); print "  · " $0; if (++n >= 20) exit }' "$_plugin_excludes_file" || true
        [ "$_excludes_count" -gt 20 ] && echo "  · ...（其余 $((_excludes_count - 20)) 个省略）"
    else
        echo "镜像未检测到预装插件目录，备份将全量解压"
    fi
    local _installs_rel="plugins/installs.json"
    local _installs_disk="$openclaw_home/$_installs_rel"
    local _installs_image_bak="" _installs_archive_tmp=""
    local _do_installs_merge=false
    echo "评估 installs.json 镜像优先合并模式..."
    local _gate_ok=true _jq_err
    if ! command -v jq >/dev/null 2>&1; then
        echo "  ✗ jq 缺失，跳过（compat_installs_json.sh 兜底）"; _gate_ok=false
    elif ! _jq_err=$(echo '{}' | jq empty 2>&1); then
        echo "  ✗ jq 异常: ${_jq_err:-<空>} (path=$(command -v jq))"; _gate_ok=false
    fi
    if $_gate_ok && [ ! -f "$_installs_disk" ]; then
        echo "  ✗ 镜像版不存在: $_installs_disk"; _gate_ok=false
    fi
    if $_gate_ok && ! _jq_err=$(jq empty "$_installs_disk" 2>&1); then
        echo "  ✗ 镜像版解析失败: ${_jq_err:-<空>}"; _gate_ok=false
    fi
    if $_gate_ok; then
        local _arc_list="/tmp/openclaw_archive_list_chk_$$.txt"
        if ! tar -tzf "$ARCHIVE_PATH" > "$_arc_list" 2>/dev/null; then
            echo "  ✗ 无法列出压缩包: $ARCHIVE_PATH"; _gate_ok=false
        elif ! grep -Eq "(^|/)\\./?$_installs_rel\$|(^|/)$_installs_rel\$" "$_arc_list"; then
            echo "  ✗ 压缩包不含 $_installs_rel"; _gate_ok=false
        fi
        rm -f "$_arc_list"
    fi
    if $_gate_ok; then
        _do_installs_merge=true
        _installs_image_bak="/tmp/openclaw_installs_image_$$.json"
        _installs_archive_tmp="/tmp/openclaw_installs_archive_$$.json"
        if ! cp -p "$_installs_disk" "$_installs_image_bak" 2>/dev/null; then
            echo "  ✗ 备份镜像版失败"; _do_installs_merge=false
        fi
    fi
    if $_do_installs_merge; then
        echo "启用 installs.json 镜像优先合并模式"
        if ! tar -xzOf "$ARCHIVE_PATH" "./$_installs_rel" 2>/dev/null > "$_installs_archive_tmp" \
            || [ ! -s "$_installs_archive_tmp" ]; then
            tar -xzOf "$ARCHIVE_PATH" "$_installs_rel" 2>/dev/null > "$_installs_archive_tmp" || true
        fi
        if [ ! -s "$_installs_archive_tmp" ] || ! jq empty "$_installs_archive_tmp" >/dev/null 2>&1; then
            echo "⚠ 提取备份版 installs.json 失败，回退为常规覆盖"
            _do_installs_merge=false
            rm -f "$_installs_archive_tmp"
        fi
    fi
    local _tar_rc=0
    if $_do_installs_merge; then
        if [ -s "$_plugin_excludes_file" ]; then
            tar -xzf "$ARCHIVE_PATH" -C "$openclaw_home" \
                --exclude-from="$_plugin_excludes_file" \
                --exclude="./$_installs_rel" \
                --exclude="$_installs_rel" || _tar_rc=$?
        else
            tar -xzf "$ARCHIVE_PATH" -C "$openclaw_home" \
                --exclude="./$_installs_rel" \
                --exclude="$_installs_rel" || _tar_rc=$?
        fi
    else
        if [ -s "$_plugin_excludes_file" ]; then
            tar -xzf "$ARCHIVE_PATH" -C "$openclaw_home" \
                --exclude-from="$_plugin_excludes_file" || _tar_rc=$?
        else
            tar -xzf "$ARCHIVE_PATH" -C "$openclaw_home" || _tar_rc=$?
        fi
    fi
    if [ "$_tar_rc" -ne 0 ]; then
        fail_exit "restore_files" "tar 解压失败 (rc=$_tar_rc): $ARCHIVE_PATH → $openclaw_home"
    fi
    echo "✓ 文件已还原到: $openclaw_home"
    rm -f "$_plugin_excludes_file"
    if $_do_installs_merge; then
        # 防 BSD tar 忽略 --exclude
        if ! cmp -s "$_installs_image_bak" "$_installs_disk" 2>/dev/null; then
            cp -p "$_installs_image_bak" "$_installs_disk" || true
        fi
        local _installs_merged="/tmp/openclaw_installs_merged_$$.json"
        # installRecords 用 + 合并（备份打底、镜像同名覆盖）
        if jq -s '
            .[0] as $arc | .[1] as $img |
            $img
            | .installRecords = (($arc.installRecords // {}) + ($img.installRecords // {}))
        ' "$_installs_archive_tmp" "$_installs_image_bak" > "$_installs_merged" \
           && jq empty "$_installs_merged" >/dev/null 2>&1; then
            local _img_keys _arc_keys _merged_keys _added_keys
            _img_keys=$(jq -r '.installRecords // {} | keys | length' "$_installs_image_bak")
            _arc_keys=$(jq -r '.installRecords // {} | keys | length' "$_installs_archive_tmp")
            _merged_keys=$(jq -r '.installRecords // {} | keys | length' "$_installs_merged")
            _added_keys=$(jq -r --slurpfile img "$_installs_image_bak" \
                '.installRecords // {} | keys
                 | map(select(. as $k | ($img[0].installRecords // {}) | has($k) | not))
                 | join(",")' "$_installs_archive_tmp")
            mv "$_installs_merged" "$_installs_disk"
            echo "✓ installs.json 合并: 镜像=$_img_keys 备份=$_arc_keys 合并后=$_merged_keys"
            [ -n "$_added_keys" ] && echo "  · 从备份补入: $_added_keys"
        else
            echo "⚠ jq 合并失败，保留镜像版（compat_installs_json.sh 兜底）"
            rm -f "$_installs_merged"
        fi
        rm -f "$_installs_archive_tmp" "$_installs_image_bak"
    fi
    echo "验证恢复结果..."
    if [ -d "$openclaw_home" ]; then
        echo "✓ OpenClaw 状态目录已恢复: $openclaw_home"
        if command -v jq >/dev/null 2>&1 && [ -f "$openclaw_home/openclaw.json" ]; then
            if jq empty "$openclaw_home/openclaw.json" >/dev/null 2>&1; then
                echo "✓ 主配置文件语法验证通过"
            else
                fail_exit "restore_files" "主配置文件 openclaw.json 语法错误"
            fi
        fi
        if [ -d "$openclaw_home/credentials" ]; then
            local cred_count
            cred_count=$(find "$openclaw_home/credentials" -type f | wc -l)
            echo "✓ credentials/ 目录已恢复，包含 $cred_count 个凭证文件"
        else
            echo "⚠ credentials/ 目录未找到，渠道可能需要重新登录"
        fi
        if [ -d "$openclaw_home/agents" ]; then
            local agent_count
            agent_count=$(find "$openclaw_home/agents" -maxdepth 1 -mindepth 1 -type d | wc -l)
            echo "✓ agents/ 目录已恢复，包含 $agent_count 个 Agent"
        else
            echo "⚠ agents/ 目录未找到，Agent 状态可能丢失"
        fi
    else
        echo "⚠ OpenClaw 状态目录未找到: $openclaw_home"
    fi
    if [ -d "$openclaw_home" ]; then
        echo "修复 OpenClaw 目录权限..."
        if ! chown -R "${TARGET_UID}:${TARGET_GID}" "$openclaw_home" 2>/dev/null; then
            echo "⚠ chown 部分失败，继续执行"
        fi
        chmod 755 "$openclaw_home" || true
        if [ -d "$openclaw_home/credentials" ]; then
            chmod 700 "$openclaw_home/credentials" || true
            find "$openclaw_home/credentials" -type f -exec chmod 600 {} \; 2>/dev/null || true
        fi
        find "$openclaw_home" -name "*.json" -exec chmod 644 {} \; 2>/dev/null || true
        find "$openclaw_home" -name "*.db"   -exec chmod 644 {} \; 2>/dev/null || true
        echo "✓ 权限修复完成"
    fi
}

load_env() {
    export PNPM_HOME="${PNPM_HOME:-$HOME/.local/share/pnpm}"
    export PATH="$PNPM_HOME:$HOME/.npm-global/bin:$HOME/.local/bin:$PATH"
    export NVM_DIR="$HOME/.nvm"
    [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
    hash -r || true
}

run_doctor() {
    echo ""
    echo ">>> [步骤 3/7] 运行 doctor 修复服务和配置迁移"
    load_env
    echo "openclaw 路径: $(which openclaw || echo '未找到')"
    echo "openclaw 版本: $(openclaw --version 2>/dev/null || echo '未知')"
    if ! command -v openclaw >/dev/null 2>&1; then
        fail_exit "run_doctor" "openclaw 命令未找到，无法继续恢复（请检查镜像安装状态）"
    fi
    user_systemctl stop openclaw-gateway || true
    stop_openclaw_processes
    # 配置迁移（openclaw.json 与 clawdbot.json 双文件兼容）
    for cfg in "$HOME/.openclaw/openclaw.json" "$HOME/.clawdbot/clawdbot.json"; do
        if [ -f "$cfg" ]; then
            # wecom V1 flat 迁移为 V2 bot
            if jq -e '.channels.wecom.token and ((.channels.wecom.bot or .channels.wecom.agent or .channels.wecom.accounts) | not)' "$cfg" > /dev/null; then
                jq '.channels.wecom = {
                  enabled: (.channels.wecom.enabled // true),
                  bot: { token: .channels.wecom.token,
                         encodingAESKey: .channels.wecom.encodingAESKey,
                         streamPlaceholderContent: (.channels.wecom.streamPlaceholderContent // "正在思考..."),
                         welcomeText: (.channels.wecom.welcomeText // "你好！我是 AI 助手"),
                         dm: (.channels.wecom.dm // {"policy": "open"}) }
                }' "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
                echo "✓ wecom 配置已迁移为 V2 格式"
            fi
            # dingtalk → ddingtalk 重命名
            if jq -e '.channels.dingtalk and (.channels.ddingtalk | not)' "$cfg" > /dev/null; then
                jq '.channels.ddingtalk = .channels.dingtalk | del(.channels.dingtalk)' "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
                echo "✓ dingtalk → ddingtalk 配置已迁移"
            fi
        fi
    done
    echo "运行 openclaw doctor 修复服务和配置迁移..."
    DOCTOR_OUTPUT=""
    DOCTOR_RC=0
    DOCTOR_OUTPUT=$(openclaw doctor --fix --yes 2>&1) || DOCTOR_RC=$?
    echo "$DOCTOR_OUTPUT"
    if [ "$DOCTOR_RC" -eq 0 ]; then
        echo "✓ openclaw doctor 执行完成"
    else
        # 检测数据库损坏信号：Go 侧 needLocalDBRepair 据此自动下发 recovery 脚本
        if echo "$DOCTOR_OUTPUT" | grep -q "database disk image is malformed"; then
            echo ""
            echo "⚠ 检测到 SQLite 数据库损坏（malformed）"
            echo "  自动恢复：升级流程会自动下发 openclaw_recovery.sh --resume 修复"
            echo "  手动恢复：请执行  sudo bash openclaw_recovery.sh --resume"
        fi
        fail_exit "run_doctor" "openclaw doctor --fix --yes 执行失败 (rc=$DOCTOR_RC)"
    fi
    mkdir -p "$HOME/.openclaw/extensions"
    rm -f "$HOME/.openclaw/clawdbot.json"*
    rm -f "$HOME/.clawdbot/clawdbot.json"*
    rm -rf "$HOME/.clawdbot"
}

# 必须在解压之前装 gateway：install 会触发 Config overwrite 覆盖备份
stage_pre_restore() {
    echo ""
    echo ">>> [步骤 1/7] 验证备份压缩包 & 预安装 Gateway 服务（在解压备份前）"
    echo "验证备份压缩包..."
    if ! tar -tzf "$ARCHIVE_PATH" >/dev/null 2>&1; then
        fail_exit "pre_restore" "压缩包损坏或格式不正确: $ARCHIVE_PATH"
    fi
    echo "✓ 压缩包验证通过"
    local list_file="/tmp/openclaw_archive_list_$$.txt"
    tar -tzf "$ARCHIVE_PATH" > "$list_file" || true
    local total
    total=$(wc -l < "$list_file")
    echo "压缩包内容概览: 共 $total 个条目"
    echo "顶层目录:"
    awk -F/ '{print $1}' "$list_file" | sort -u | sed 's/^/  - /'
    rm -f "$list_file"
    load_env
    if ! command -v openclaw >/dev/null 2>&1; then
        echo "⚠ openclaw 命令未找到，跳过 Gateway 预安装步骤"
        return
    fi
    echo "安装 gateway 服务（预执行，避免覆盖备份配置）..."
    # --force 确保 unit 文件被创建；配置覆盖不影响（下一步解压备份会还原）
    if ! openclaw gateway install --force; then
        echo "⚠ openclaw gateway install --force 返回非零，继续执行（可能已经安装过）"
    fi
    echo "✓ gateway 服务安装完成"
    echo "停止 openclaw 业务进程..."
    user_systemctl stop openclaw-gateway || true
    stop_openclaw_processes
    echo "✓ 已停止，可安全解压备份"
}

check_channel_plugins() {
    local cfg="$HOME/.openclaw/openclaw.json"
    echo "校验所有插件与通道配置的一致性..."
    if [ ! -f "$cfg" ] || ! command -v jq >/dev/null 2>&1; then
        echo "⚠ 配置文件不存在或 jq 未安装，跳过插件校验"
        return
    fi
    local ALLOW_IDS=(
        "qqbot" "openclaw-qqbot" "ddingtalk" "dingtalk-connector"
        "wecom" "wecom-openclaw-plugin" "adp-openclaw" "yuanbao"
        "openclaw-plugin-yuanbao" "openclaw-weixin" "lightclawbot"
        "memory-tdai" "feishu" "openclaw-lark" "slack" "discord"
    )
    local allow_ids_json
    allow_ids_json=$(printf '%s\n' "${ALLOW_IDS[@]}" | jq -R . | jq -s .)
    jq --argjson ids "$allow_ids_json" '
        .plugins.allow = ((.plugins.allow // []) + ($ids - (.plugins.allow // [])))
    ' "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
    echo "✓ 所有预设插件 ID 已写入 plugins.allow"
    local _has_wecom_official=false
    [ -d "$HOME/.openclaw/extensions/wecom-openclaw-plugin" ] && _has_wecom_official=true
    _set_plugin_enabled() {
        local plugin_id="$1" enabled="$2"
        if ! jq -e --arg id "$plugin_id" '.plugins.entries[$id]' "$cfg" >/dev/null 2>&1; then
            echo "⏭ ${plugin_id} 未在 plugins.entries 中注册，跳过"
            return
        fi
        jq --arg id "$plugin_id" --argjson en "$enabled" \
            '.plugins.entries[$id].enabled = $en' "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
        echo "✓ ${plugin_id} 插件已$([ "$enabled" = "true" ] && echo 启用 || echo 禁用)"
    }
    # channel jq 路径 | action(native/entry) | plugin_id
    local mappings=(
        '.channels.feishu|native|feishu'
        '.channels.slack|native|slack'
        '.channels.discord|native|discord'
        '.channels.qqbot|native|openclaw-qqbot'
        '.channels.ddingtalk|entry|ddingtalk'
        '.channels["dingtalk-connector"]|entry|dingtalk-connector'
        '.channels["openclaw-weixin"]|entry|openclaw-weixin'
        '.channels.lightclawbot|entry|lightclawbot'
        '.channels["adp-openclaw"]|entry|adp-openclaw'
        '.channels.yuanbao|entry|openclaw-plugin-yuanbao'
    )
    local m ch_path action plugin_id configured
    for m in "${mappings[@]}"; do
        IFS='|' read -r ch_path action plugin_id <<< "$m"
        if jq -e "$ch_path" "$cfg" >/dev/null 2>&1; then configured=true; else configured=false; fi
        case "$action" in
            native)
                if $configured; then
                    openclaw plugins enable "$plugin_id" || true
                    echo "✓ ${plugin_id} 插件已启用"
                else
                    openclaw plugins disable "$plugin_id" || true
                    echo "✓ ${plugin_id} 插件已禁用"
                fi
                ;;
            entry)
                if $configured; then _set_plugin_enabled "$plugin_id" "true"
                else _set_plugin_enabled "$plugin_id" "false"; fi
                ;;
        esac
    done
    # wecom: 根据 official 插件是否存在决定 enable 哪个 plugin_id
    if jq -e '.channels.wecom' "$cfg" >/dev/null 2>&1; then
        if $_has_wecom_official; then
            _set_plugin_enabled "wecom-openclaw-plugin" "true"
            _set_plugin_enabled "wecom" "false"
        else
            _set_plugin_enabled "wecom" "true"
            _set_plugin_enabled "wecom-openclaw-plugin" "false"
        fi
    else
        _set_plugin_enabled "wecom-openclaw-plugin" "false"
        _set_plugin_enabled "wecom" "false"
    fi
    # 兜底：disable 无对应 channel 配置的孤儿启用项
    echo "兜底扫描：disable 无对应 channel 配置的孤儿启用项..."
    local _exp=()
    jq -e '.channels.qqbot'                  "$cfg" >/dev/null 2>&1 && _exp+=("qqbot" "openclaw-qqbot")
    jq -e '.channels.feishu'                 "$cfg" >/dev/null 2>&1 && _exp+=("feishu" "openclaw-lark")
    jq -e '.channels.slack'                  "$cfg" >/dev/null 2>&1 && _exp+=("slack")
    jq -e '.channels.discord'                "$cfg" >/dev/null 2>&1 && _exp+=("discord")
    jq -e '.channels.ddingtalk'              "$cfg" >/dev/null 2>&1 && _exp+=("ddingtalk")
    jq -e '.channels["dingtalk-connector"]'  "$cfg" >/dev/null 2>&1 && _exp+=("dingtalk-connector")
    jq -e '.channels["openclaw-weixin"]'     "$cfg" >/dev/null 2>&1 && _exp+=("openclaw-weixin")
    jq -e '.channels.lightclawbot'           "$cfg" >/dev/null 2>&1 && _exp+=("lightclawbot")
    jq -e '.channels["adp-openclaw"]'        "$cfg" >/dev/null 2>&1 && _exp+=("adp-openclaw")
    jq -e '.channels.yuanbao'                "$cfg" >/dev/null 2>&1 && _exp+=("yuanbao" "openclaw-plugin-yuanbao")
    if jq -e '.channels.wecom' "$cfg" >/dev/null 2>&1; then
        $_has_wecom_official && _exp+=("wecom-openclaw-plugin") || _exp+=("wecom")
    fi
    local _managed='["feishu","openclaw-lark","slack","discord","qqbot","openclaw-qqbot","ddingtalk","dingtalk-connector","openclaw-weixin","lightclawbot","adp-openclaw","yuanbao","openclaw-plugin-yuanbao","wecom","wecom-openclaw-plugin"]'
    local _exc=("memory-tdai" "memory-tencentdb" "clawpro-diagnostics-metrics-cls" "browser" "codex" "memory-core")
    local _user_kept
    _user_kept=$(jq -r --argjson m "$_managed" \
        '(.plugins.entries // {}) | keys | map(select(. as $k | $m | index($k) | not)) | .[]' \
        "$cfg" 2>/dev/null || true)
    if [ -n "$_user_kept" ]; then
        while IFS= read -r _uid; do [ -n "$_uid" ] && _exc+=("$_uid"); done <<< "$_user_kept"
        echo "保留用户自装插件 enabled 状态: $(echo "$_user_kept" | tr '\n' ' ')"
    fi
    jq '.plugins.entries // {}' "$cfg" > "/tmp/openclaw_entries_before_fix_$$.json" 2>/dev/null || true
    local _ej _xj
    _ej=$([ "${#_exp[@]}" -eq 0 ] && echo "[]" || printf '%s\n' "${_exp[@]}" | jq -R . | jq -s .)
    _xj=$(printf '%s\n' "${_exc[@]}" | jq -R . | jq -s .)
    local _td
    _td=$(jq -r --argjson e "$_ej" --argjson x "$_xj" '
        ((.plugins.entries // {}) | to_entries
            | map(select(.value.enabled == true
                and (.key as $k | $e | index($k) | not)
                and (.key as $k | $x | index($k) | not))) | .[].key),
        (. as $r | (.plugins.allow // [])
            | map(select((. as $k | ($r.plugins.entries // {}) | has($k) | not)
                and (. as $k | $e | index($k) | not)
                and (. as $k | $x | index($k) | not))) | .[])
    ' "$cfg" 2>/dev/null | awk 'NF && !seen[$0]++' || true)
    if [ -z "$_td" ]; then
        echo "✓ 无孤儿启用项需要清理"
    else
        local _id
        while IFS= read -r _id; do
            [ -z "$_id" ] && continue
            command -v openclaw >/dev/null 2>&1 && openclaw plugins disable "$_id" 2>/dev/null || true
            if jq -e --arg id "$_id" '.plugins.entries[$id]' "$cfg" >/dev/null 2>&1; then
                jq --arg id "$_id" '.plugins.entries[$id].enabled = false' "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
            else
                jq --arg id "$_id" '.plugins.entries[$id] = {"enabled": false}' "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
            fi
            echo "✓ ${_id} 插件已禁用（无对应 channel 配置）"
        done <<< "$_td"
    fi
}

migrate_wecom_plugin() {
    local cfg="$HOME/.openclaw/openclaw.json"
    echo ""
    echo "检测并迁移 wecom 插件配置格式"
    [ -f "$cfg" ] && command -v jq >/dev/null 2>&1 || { echo "配置文件或 jq 不可用，跳过"; return; }
    local ext_dir="$HOME/.openclaw/extensions" target_official
    if   [ -d "$ext_dir/wecom-openclaw-plugin" ]; then target_official=true;  echo "检测到 official wecom 插件目录"
    elif [ -d "$ext_dir/wecom" ];                 then target_official=false; echo "检测到 mocrane wecom 插件目录"
    else echo "未检测到 wecom 插件目录，跳过迁移"; return
    fi
    local config_is_official
    if   jq -e '.channels.wecom.botId or .channels.wecom.secret'         "$cfg" >/dev/null 2>&1; then config_is_official=true;  echo "wecom 配置格式: official"
    elif jq -e '.channels.wecom.bot.botId or .channels.wecom.bot.secret' "$cfg" >/dev/null 2>&1; then config_is_official=false; echo "wecom 配置格式: mocrane"
    else echo "未检测到 wecom bot 配置，跳过迁移"; return
    fi
    if [ "$config_is_official" = "$target_official" ]; then
        echo "✓ 配置格式与插件一致，无需迁移"
        return
    fi
    if ! $config_is_official && $target_official; then
        echo "▶ 正向迁移: mocrane → official"
        jq '
          .channels.wecom.botId = .channels.wecom.bot.botId? |
          .channels.wecom.secret = .channels.wecom.bot.secret? |
          .channels.wecom.connectionMode = (.channels.wecom.bot.connectionMode? // "websocket") |
          .channels.wecom.dmPolicy = (.channels.wecom.bot.dm.policy? // "open") |
          .channels.wecom.allowFrom = (.channels.wecom.bot.dm.allowFrom? // []) |
          .channels.wecom.bot |= del(.botId, .secret, .connectionMode, .dm) |
          if .channels.wecom.accounts then
            .channels.wecom.accounts |= with_entries(
              .value.botId = .value.bot.botId? |
              .value.secret = .value.bot.secret? |
              .value.connectionMode = (.value.bot.connectionMode? // "websocket") |
              .value.bot |= del(.botId, .secret, .connectionMode, .dm)
            )
          else . end
        ' "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
        echo "✓ 正向迁移完成"
    else
        echo "◀ 反向迁移: official → mocrane"
        jq '
          .channels.wecom.bot.botId = .channels.wecom.botId? |
          .channels.wecom.bot.secret = .channels.wecom.secret? |
          .channels.wecom.bot.connectionMode = (.channels.wecom.connectionMode? // "websocket") |
          .channels.wecom.bot.dm.policy = (.channels.wecom.dmPolicy? // "open") |
          .channels.wecom.bot.dm.allowFrom = (.channels.wecom.allowFrom? // []) |
          del(.channels.wecom.botId, .channels.wecom.secret,
              .channels.wecom.connectionMode, .channels.wecom.dmPolicy,
              .channels.wecom.allowFrom) |
          if .channels.wecom.accounts then
            .channels.wecom.accounts |= with_entries(
              .value.bot.botId = .value.botId? |
              .value.bot.secret = .value.secret? |
              .value.bot.connectionMode = (.value.connectionMode? // "websocket") |
              del(.value.botId, .value.secret, .value.connectionMode)
            )
          else . end
        ' "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
        echo "✓ 反向迁移完成"
    fi
}

stage_plugins() {
    echo ""
    echo ">>> [步骤 4/7] 适配镜像内置插件"
    load_env
    export NO_COLOR=1
    check_channel_plugins
    migrate_wecom_plugin
    echo "✓ 镜像内置插件适配完成"
}

stage_skills() {
    echo ""
    echo ">>> [步骤 5/7] 安装 skillhub"
    rm -rf ~/.openclaw/extensions/skillhub/index.ts
    rm -rf ~/.openclaw/extensions/skillhub/openclaw.plugin.json
    timeout --kill-after=10 240 bash -c 'curl -fsSL https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/install/install.sh | bash' || true
    if command -v skillhub; then
        for i in 1 2 3; do skillhub install --force web-tools-guide && break || sleep 2; done
    else
        echo "⚠ skillhub 安装失败或未找到，跳过 web-tools-guide 安装"
    fi
    echo "✓ skillhub 安装完成"
}

stage_browser_setup() {
    echo ""
    echo ">>> [步骤 6/7] 浏览器配置"
    local cfg="$HOME/.openclaw/openclaw.json"
    local cdp_port=9222
    local user_data_dir="$HOME/.openclaw/browser-existing-session"
    local service_name="lighthouse-chromium"
    local unit_file="/etc/systemd/system/${service_name}.service"
    local devtools_script="/usr/local/bin/update-devtools-port.sh"
    _wait_cdp() {
        local port="$1" retries="${2:-30}" interval="${3:-1}"
        echo "等待 CDP 端口 ${port}..."
        while [ "$retries" -gt 0 ]; do
            curl -s --max-time 2 "http://localhost:${port}/json/version" >/dev/null 2>&1 && {
                echo "✓ CDP 端口 ${port} 就绪"; return 0; }
            sleep "$interval"; retries=$((retries - 1))
        done
        echo "⚠ CDP 端口 ${port} 未响应，继续执行"; return 1
    }
    _write_devtools_port() {
        local port="$1" dir="$2" bid
        mkdir -p "$dir"
        bid=$(curl -s "http://localhost:${port}/json/version" \
            | grep webSocketDebuggerUrl 2>/dev/null \
            | sed 's/.*browser\///' | tr -d '"' | tr -d ' ')
        if [ -n "$bid" ] && [ "$bid" != "null" ]; then
            echo -e "${port}\n/devtools/browser/${bid}" > "${dir}/DevToolsActivePort"
            echo "✓ DevToolsActivePort 已更新 (browser ID: ${bid})"
        fi
    }
    _write_browser_cfg() {
        local chrome_bin="$1"
        [ -f "$cfg" ] && command -v jq >/dev/null 2>&1 || return 0
        echo "配置浏览器 existing-session 模式..."
        jq --arg bin "$chrome_bin" '.browser = {
            "enabled": true, "executablePath": $bin, "noSandbox": true,
            "defaultProfile": "user",
            "profiles": { "user": {
                "cdpUrl": "http://localhost:9222",
                "driver": "existing-session", "attachOnly": true, "color": "#4285F4"
            }}
        } | .tools.deny = [.tools.deny[]? | select(. != "browser" and . != "web_search" and . != "web_fetch")]' \
            "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
        echo "✓ 浏览器 existing-session 模式配置完成"
    }
    # VNC 模式感知
    local vnc_mode=false
    if [ -f "/etc/systemd/system/browser-vnc-chromium.service" ]; then
        if systemctl is-active --quiet browser-vnc-chromium 2>/dev/null; then
            vnc_mode=true
            echo "检测到 VNC Chrome 已运行，使用 VNC 模式"
        else
            echo "VNC unit 存在但服务未运行，尝试启动..."
            systemctl daemon-reload
            systemctl start browser-vnc-xvnc browser-vnc-websockify browser-vnc-openbox browser-vnc-session 2>/dev/null || true
            sleep 3
            systemctl start browser-vnc-chromium 2>/dev/null || true
            sleep 5
            if systemctl is-active --quiet browser-vnc-chromium 2>/dev/null; then
                vnc_mode=true
                echo "✓ VNC 服务已启动，使用 VNC 模式"
            else
                echo "⚠ VNC 服务启动失败，回退到 headless 模式"
            fi
        fi
    fi
    if [ "$vnc_mode" = true ]; then
        systemctl is-active --quiet "$service_name" 2>/dev/null && systemctl stop "$service_name" 2>/dev/null || true
        systemctl disable "$service_name" 2>/dev/null || true
        _wait_cdp "$cdp_port" 30 1 || true
        _write_devtools_port "$cdp_port" "$user_data_dir"
        _write_browser_cfg "/usr/bin/google-chrome-stable"
        echo "✓ 浏览器配置完成（VNC 模式）"
        return
    fi
    # 标准模式：headless Chrome
    echo "标准模式：使用 headless Chrome"
    local chromium_dir
    chromium_dir=$(ls -d "${HOME}/.cache/ms-playwright"/chromium-[0-9]* 2>/dev/null \
        | grep -v headless_shell | sort -V | tail -1 || true)
    if [ -z "$chromium_dir" ]; then
        echo "⚠ 未找到 Chromium，跳过浏览器配置（如需可手动: python3 -m playwright install chromium）"
        return
    fi
    local chrome_bin="${chromium_dir}/chrome-linux64/chrome"
    if [ ! -x "$chrome_bin" ]; then
        echo "⚠ Chromium 可执行文件不存在: $chrome_bin，跳过浏览器配置"
        return
    fi
    echo "Chromium binary: $chrome_bin"
    cat > "$devtools_script" << 'SCRIPT_EOF'
#!/usr/bin/env bash
CDP_PORT="${1:-9222}"
USER_DATA_DIR="${2:-$HOME/.openclaw/browser-existing-session}"
for i in $(seq 1 60); do
    curl -s --max-time 2 "http://localhost:${CDP_PORT}/json/version" >/dev/null && break
    sleep 0.5
done
BROWSER_ID=$(curl -s "http://localhost:${CDP_PORT}/json/version" \
    | grep webSocketDebuggerUrl | sed 's/.*browser\///' | tr -d '"' | tr -d ' ')
[ -z "$BROWSER_ID" ] && { echo "[ERROR] Failed to extract browser ID" >&2; exit 1; }
mkdir -p "${USER_DATA_DIR}"
echo -e "${CDP_PORT}\n/devtools/browser/${BROWSER_ID}" > "${USER_DATA_DIR}/DevToolsActivePort"
echo "[INFO] DevToolsActivePort updated (browser ID: ${BROWSER_ID})"
SCRIPT_EOF
    chmod +x "$devtools_script"
    mkdir -p "$user_data_dir"
    cat > "$unit_file" <<UNIT_EOF
[Unit]
Description=Chromium Headless (CDP port ${cdp_port})
After=network.target

[Service]
Type=simple
ExecStart=${chrome_bin} \\
    --remote-debugging-port=${cdp_port} \\
    --no-sandbox \\
    --headless=new \\
    --disable-gpu \\
    --window-size=1920,1080 \\
    --ozone-override-screen-size=1920,1080 \\
    --user-data-dir=${user_data_dir}
ExecStartPost=-${devtools_script} ${cdp_port} ${user_data_dir}
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT_EOF
    systemctl daemon-reload
    systemctl enable "$service_name" || true
    systemctl restart "$service_name" || echo "⚠ lighthouse-chromium 启动失败，继续执行"
    _wait_cdp "$cdp_port" 60 0.5 || true
    _write_browser_cfg "$chrome_bin"
    echo "✓ 浏览器配置完成（标准 headless 模式）"
}

stage_finalize() {
    echo ""
    echo ">>> [步骤 7/7] 收尾：创建软链接、修复配置、重启 gateway"
    local pnpm_bin="${PNPM_HOME:-$HOME/.local/share/pnpm}"
    if [ -x "$pnpm_bin/openclaw" ]; then
        echo "创建 /usr/local/bin 软链接..."
        for cmd in openclaw clawdbot; do
            cat > "/usr/local/bin/$cmd" << WRAPPER
#!/bin/sh
export XDG_RUNTIME_DIR=\${XDG_RUNTIME_DIR:-/run/user/\$(id -u)}
exec "$pnpm_bin/openclaw" "\$@"
WRAPPER
            chmod +x "/usr/local/bin/$cmd"
        done
        echo "✓ 软链接创建完成"
    fi
    local cfg="$HOME/.openclaw/openclaw.json"
    if [ -f "$cfg" ] && command -v jq; then
        jq '
          if .tools.profile == "messaging" then .tools.profile = "full" else . end
          | if (.gateway.mode // "") == "" then .gateway.mode = "local" else . end
          | if .plugins.allow and (["openclaw-weixin"] - .plugins.allow | length > 0) then .plugins.allow += ["openclaw-weixin"] else . end
        ' "$cfg" > "$cfg.tmp" && mv "$cfg.tmp" "$cfg"
        echo "✓ 配置修复完成（tools.profile / gateway.mode / plugins.allow）"
    fi
    # 2GB 机器：drop-in 限制 Node.js 老生代内存避 OOM
    local TOTAL_MEM_KB
    TOTAL_MEM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}' || echo "0")
    if [ "$TOTAL_MEM_KB" -gt 0 ] && [ "$TOTAL_MEM_KB" -lt 3000000 ]; then
        local DROPIN_DIR="$HOME/.config/systemd/user/openclaw-gateway.service.d"
        mkdir -p "$DROPIN_DIR"
        cat > "$DROPIN_DIR/env.conf" << 'ENVEOF'
[Service]
Environment="NODE_OPTIONS=--max-old-space-size=1800"
ENVEOF
        user_systemctl daemon-reload || true
        echo "✓ 已创建 drop-in env.conf (NODE_OPTIONS=--max-old-space-size=1800, MemTotal: ${TOTAL_MEM_KB}KB)"
    fi
    echo "重启 gateway..."
    if ! user_systemctl restart openclaw-gateway; then
        fail_exit "finalize" "openclaw-gateway 重启命令执行失败"
    fi
    if [ -d "$DEFAULT_OPENCLAW_HOME" ]; then
        chown -R "${TARGET_UID}:${TARGET_GID}" "$DEFAULT_OPENCLAW_HOME" 2>/dev/null || true
    fi
    echo "等待 gateway 进入 active 状态..."
    local active=false
    for i in 1 2 3 4 5 6 7 8 9 10; do
        user_systemctl is-active --quiet openclaw-gateway 2>/dev/null && { active=true; break; }
        sleep 1
    done
    if [ "$active" = true ]; then
        echo "✓ gateway 已重启并处于 active 状态"
    else
        echo "gateway 当前状态:"
        user_systemctl status openclaw-gateway --no-pager -l 2>/dev/null | head -20 || true
        fail_exit "finalize" "gateway 重启后未能进入 active 状态"
    fi
}

main() {
    if [ "$RESUME_AFTER_DOCTOR" = "true" ]; then
        echo "=== 恢复续跑模式（跳过下载/解压，从 doctor 开始执行配置迁移）==="
        load_env
        run_doctor
    else
        download_from_smh
        stage_pre_restore
        restore_files
        run_doctor
    fi
    stage_plugins
    stage_skills
    stage_browser_setup
    stage_finalize
    echo ""
    echo "=== 数据恢复完成 ==="
    echo "所有文件已按原始路径还原"
    echo "下一步: openclaw gateway start"
    echo "RESTORE_COMPLETED:/" > /tmp/restore_status
}

main "$@"
exit 0