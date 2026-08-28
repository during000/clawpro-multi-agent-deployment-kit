#!/bin/bash
set -euo pipefail

# Hermes 重装后数据恢复脚本。
# 从 SMH 下载备份包，解压覆盖新镜像的 ~/.hermes/，重建 venv（因备份时排除了不可移植的 venv），重启 gateway。
# 入参：{{url}}（SMH 下载 URL）、{{runtime_user}}（运行用户）。
# 失败输出 RESTORE_FAILED:<stage> 并 exit 非 0；成功输出 RESTORE_OK。

# ========== 运行用户解析 ==========
# 三层兜底：{{runtime_user}} → HERMES_RUNTIME_USER env → agentuser
RUNTIME_USER="{{runtime_user}}"
if [ -z "$RUNTIME_USER" ] || [ "$RUNTIME_USER" = "{{runtime_user}}" ]; then
    RUNTIME_USER="${HERMES_RUNTIME_USER:-agentuser}"
fi

# 直接读 /etc/passwd 校验用户存在性，避免 id 命令在 TAT 容器初期因 NSS/fork 紧张瞬时失败
_passwd_line=""
[ -r /etc/passwd ] && _passwd_line=$(awk -F: -v u="$RUNTIME_USER" '$1==u{print; exit}' /etc/passwd)
if [ -z "$_passwd_line" ] && [ "$RUNTIME_USER" != "root" ]; then
    _self_user="$(id -un 2>/dev/null || whoami 2>/dev/null || echo '')"
    if [ -n "$_self_user" ] && [ "$_self_user" != "$RUNTIME_USER" ]; then
        _self_line="$(awk -F: -v u="$_self_user" '$1==u{print; exit}' /etc/passwd 2>/dev/null || true)"
        if [ -n "$_self_line" ]; then
            echo "WARN: runtime user '$RUNTIME_USER' 不存在于 /etc/passwd，改用当前脚本执行身份 '$_self_user'"
            RUNTIME_USER="$_self_user"
            _passwd_line="$_self_line"
        else
            echo "WARN: runtime user '$RUNTIME_USER' 与当前身份 '$_self_user' 均不在 /etc/passwd，回退到 root"
            RUNTIME_USER="root"
            _passwd_line=$(awk -F: '$1=="root"{print; exit}' /etc/passwd 2>/dev/null || true)
        fi
    else
        echo "WARN: runtime user '$RUNTIME_USER' 不存在于 /etc/passwd，且 whoami/id -un 均失败，回退到 root"
        RUNTIME_USER="root"
        _passwd_line=$(awk -F: '$1=="root"{print; exit}' /etc/passwd 2>/dev/null || true)
    fi
fi

TARGET_UID=""
TARGET_GID=""
TARGET_HOME=""
if [ -n "$_passwd_line" ]; then
    TARGET_UID=$(printf '%s' "$_passwd_line" | awk -F: '{print $3}')
    TARGET_GID=$(printf '%s' "$_passwd_line" | awk -F: '{print $4}')
    TARGET_HOME=$(printf '%s' "$_passwd_line" | awk -F: '{print $6}')
fi
[ -z "$TARGET_UID" ] && TARGET_UID="$(id -u 2>/dev/null || echo 0)"
[ -z "$TARGET_GID" ] && TARGET_GID="$(id -g 2>/dev/null || echo 0)"
[ -z "$TARGET_HOME" ] && TARGET_HOME=$([ "$RUNTIME_USER" = "root" ] && echo "/root" || echo "/home/$RUNTIME_USER")
unset _passwd_line
HERMES_HOME="${TARGET_HOME}/.hermes"
export HOME="$TARGET_HOME"

# 日志直接落 ~/.hermes/logs/
SCRIPT_NAME="restore_post_reinstall_hermes"
LOG_DIR="${HERMES_HOME}/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"

# 保存原始 stdout/stderr（TAT 捕获管道），供退出时恢复
exec 3>&1 4>&2

# 用 FIFO + 后台 tee 替代 process substitution(> >(...))，确保日志可靠落盘。
# 问题：exec > >(tee ...) 的 tee 在主脚本 exit 后成为孤儿进程，
# TAT 可能在主进程退出后立即清理进程组(SIGKILL)，导致 tee 的文件写入
# 缓冲区未 flush —— TAT 控制台日志完整(直读 tee stdout 管道)，
# 但机器上日志文件在耗时操作(cp -an)后截断。
TEE_PID=""
LOG_FIFO="/tmp/.hermes_restore_log_fifo_$$"
rm -f "$LOG_FIFO" 2>/dev/null || true
if mkfifo "$LOG_FIFO" 2>/dev/null; then
    tee -a "$LOG_FILE" < "$LOG_FIFO" &
    TEE_PID=$!
    exec > "$LOG_FIFO" 2>&1
else
    LOG_FIFO=""
    exec > >(tee -a "$LOG_FILE") 2>&1
fi

echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "日志文件: $LOG_FILE"
echo "运行用户: $RUNTIME_USER (uid=$TARGET_UID, gid=$TARGET_GID)"
echo "Hermes 主目录: $HERMES_HOME"

# 诊断信息：复盘 fallback 决策时用
echo ""
echo "========== 诊断信息 =========="
echo "[diag] whoami       : $(whoami 2>/dev/null || echo '<failed>')"
echo "[diag] id -un       : $(id -un 2>/dev/null || echo '<failed>')"
echo "[diag] id -u/-g     : $(id -u 2>/dev/null || echo '?')/$(id -g 2>/dev/null || echo '?')"
echo "[diag] HOME env     : ${HOME:-<unset>}"
echo "[diag] /etc/passwd 中 uid>=1000 的真人用户:"
awk -F: '$3>=1000 && $3<65534 {printf "         %s (uid=%s, home=%s, shell=%s)\n", $1, $3, $6, $7}' /etc/passwd 2>/dev/null || true
echo "[diag] /home 目录清单:"
ls -la /home 2>/dev/null | head -20 || true
echo "[diag] hermes 安装候选检查:"
for _cand in /root /home/agentuser /home/ubuntu /home/hermes; do
    if [ -d "$_cand" ]; then
        _has_bin="no"; _has_cfg="no"
        [ -x "$_cand/.local/bin/hermes" ] && _has_bin="yes"
        [ -f "$_cand/.hermes/config.yaml" ] && _has_cfg="yes"
        echo "         $_cand exists=yes bin=$_has_bin cfg=$_has_cfg"
    else
        echo "         $_cand exists=no"
    fi
done
echo "=============================="
echo ""

# ========== 失败汇报 ==========
fail_exit() {
    local stage="$1"
    local msg="$2"
    echo ""
    echo "✗ [FATAL] 阶段 [$stage] 失败：$msg"
    echo "RESTORE_FAILED:${stage}" > /tmp/restore_status 2>/dev/null || true
    exit 1
}

on_error() {
    local exit_code=$?
    local lineno="${1:-?}"
    if [ "$exit_code" -ne 0 ]; then
        echo ""
        echo "✗ [FATAL] 脚本在第 ${lineno} 行非预期退出（exit=$exit_code）"
        if [ ! -s /tmp/restore_status ] || ! grep -q "^RESTORE_" /tmp/restore_status 2>/dev/null; then
            echo "RESTORE_FAILED:unexpected_error" > /tmp/restore_status 2>/dev/null || true
        fi
    fi
}
trap 'on_error "$LINENO"' ERR

# 脚本退出时确保 tee flush 日志到文件（修复 process substitution tee 缓冲丢失问题）
_cleanup_log_tee() {
    if [ -n "${TEE_PID:-}" ]; then
        # 恢复原始 stdout/stderr（关闭 FIFO 写入端，让 tee 收到 EOF）
        exec 1>&3 2>&4 3>&- 4>&- 2>/dev/null || true
        # 不能只靠 wait：tee 写文件使用 stdio 块缓冲（非终端时默认 4KB/8KB），
        # 收到 EOF 后需时间 flush。若 TAT 在主进程 exit 后立即 SIGKILL 进程组，
        # tee 被杀导致块缓冲未落盘 → 日志在 cp -an 等耗时操作后截断。
        # sleep 给 tee 时间读取 FIFO 剩余数据并 flush；sync 强制文件系统落盘。
        sleep 1
        sync 2>/dev/null || true
        wait "$TEE_PID" 2>/dev/null || true
    fi
    [ -n "${LOG_FIFO:-}" ] && rm -f "$LOG_FIFO" 2>/dev/null || true
}

# restore_files 已将新镜像目录移到 FRESH_HERMES_BACKUP 后，任何失败都必须恢复它，
# 避免实例停在半恢复状态。外部保留路径的写入无法原子回滚，但核心 Hermes 状态目录可恢复。
rollback_fresh_hermes_home() {
    local status="$1"
    [ "$status" -ne 0 ] || return 0
    [ -n "${FRESH_HERMES_BACKUP:-}" ] && [ -d "$FRESH_HERMES_BACKUP" ] || return 0

    echo "[rollback] 恢复失败，回滚新镜像 Hermes 目录: $FRESH_HERMES_BACKUP → $HERMES_HOME"
    rm -rf "$HERMES_HOME" 2>/dev/null || true
    if mv "$FRESH_HERMES_BACKUP" "$HERMES_HOME" 2>/dev/null; then
        echo "[rollback] ✓ 新镜像 Hermes 目录已恢复"
    else
        echo "[rollback] ✗ 无法自动恢复 $FRESH_HERMES_BACKUP，请人工处理" >&2
    fi
}

on_exit() {
    local status=$?
    rollback_fresh_hermes_home "$status"
    _cleanup_log_tee
    return "$status"
}
trap 'on_exit' EXIT

run_as_runtime_user() {
    echo "[run] $(id -un) → $RUNTIME_USER: $*"
    if [ "$(id -un)" = "$RUNTIME_USER" ]; then
        "$@"
        return
    fi
    if [ "$(id -u)" -ne 0 ]; then
        if command -v sudo >/dev/null 2>&1 && sudo -n -u "$RUNTIME_USER" true >/dev/null 2>&1; then
            sudo -n -u "$RUNTIME_USER" -- "$@"
            return
        fi
        echo "[run_as_runtime_user] 当前非 root 且无 sudo 免密，以当前身份 $(id -un) 直接执行"
        "$@"
        return
    fi
    if command -v runuser >/dev/null 2>&1; then
        runuser -u "$RUNTIME_USER" -- "$@"
        return
    fi
    su - "$RUNTIME_USER" -s /bin/bash -c "$(printf '%q ' "$@")"
}

# ========== TAT 模板变量 ==========
ARCHIVE_URL="{{url}}"
UPGRADE_TS=""
FRESH_HERMES_BACKUP=""  # restore_files 将新镜像默认 .hermes 备份到此路径

# ========== 解析参数 ==========
ARCHIVE_PATH=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --archive|--backup-dir) ARCHIVE_PATH="$2"; ARCHIVE_URL=""; shift 2 ;;
        --url) ARCHIVE_URL="$2"; shift 2 ;;
        -h|--help) echo "Usage: $0 (--archive <path.tgz> | --url <smh-url>)"; exit 0 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

if [ -z "$ARCHIVE_PATH" ] && [ -z "$ARCHIVE_URL" ]; then
    fail_exit "args" "url 参数是必需的（TAT 模板变量 {{url}} 或 --url 参数）"
fi

echo "=== Hermes 重装后数据恢复 ==="
# 下载 URL 包含短期访问令牌，日志仅保留不带查询参数的路径。
echo "备份压缩包: ${ARCHIVE_PATH:-${ARCHIVE_URL%%\?*}}"

# ========== 步骤 0：从 SMH 下载 ==========
# 先落到 /tmp（不能放 $HERMES_HOME/upgrades/，因为 restore_files 会 mv 掉整个 .hermes）
download_from_smh() {
    if [ -z "$ARCHIVE_URL" ]; then
        return
    fi
    echo ""
    echo ">>> [步骤 0/6] 从 SMH 下载备份压缩包"

    UPGRADE_TS="$(date +%Y%m%d_%H%M%S)"
    ARCHIVE_PATH="/tmp/hermes-restore-${UPGRADE_TS}.tgz"

    echo "下载到: $ARCHIVE_PATH"
    # 优先内网，失败回退公网
    if ! curl -fsSL --connect-timeout 15 --max-time 600 -o "$ARCHIVE_PATH" "$ARCHIVE_URL"; then
        local public_url="$ARCHIVE_URL"
        public_url="${public_url//&internal_domain=1/}"
        public_url="${public_url//\?internal_domain=1&/\?}"
        public_url="${public_url//\?internal_domain=1/}"
        if [ "$public_url" != "$ARCHIVE_URL" ]; then
            echo "内网下载失败，回退到公网域名重试..."
            curl -fsSL --connect-timeout 15 --max-time 600 -o "$ARCHIVE_PATH" "$public_url" \
                || fail_exit "download" "下载备份包失败（内网+公网均失败）: $ARCHIVE_URL"
        else
            fail_exit "download" "下载备份包失败: $ARCHIVE_URL"
        fi
    fi

    [ ! -s "$ARCHIVE_PATH" ] && fail_exit "download" "下载文件为空: $ARCHIVE_PATH"
    tar -tzf "$ARCHIVE_PATH" >/dev/null 2>&1 || fail_exit "download" "下载的压缩包损坏: $ARCHIVE_PATH"
    echo "✓ 下载完成: $ARCHIVE_PATH ($(du -sh "$ARCHIVE_PATH" | cut -f1))"
}

# ========== 步骤 1：停服 ==========
stop_hermes_gateway() {
    echo ""
    echo ">>> [步骤 1/6] 停止 Hermes Gateway"
    local stop_ok=0
    for unit in hermes hermes-gateway harness-gateway; do
        echo "[stop] 尝试 systemctl --user stop $unit"
        if run_as_runtime_user bash -lc "systemctl --user stop ${unit} 2>/dev/null"; then
            echo "✓ systemctl --user stop ${unit} 成功"
            stop_ok=1
            break
        fi
        echo "[stop] systemctl --user stop ${unit} 未成功"
    done

    if [ "$stop_ok" -eq 0 ]; then
        echo "[stop] systemctl --user stop 均未生效，尝试 pkill 'hermes_cli.main gateway' 兜底"
        run_as_runtime_user bash -lc "pkill -u \"\$(id -un)\" -f 'hermes_cli\\.main gateway' 2>/dev/null || true" || true
    fi

    sleep 2
    echo "[stop] 停服后残留进程检查:"
    ps -ef 2>/dev/null | grep -E 'hermes_cli\\.main|hermes-gateway|harness.*gateway' | grep -v grep || echo "  <无残留进程>"
    echo "✓ Gateway 停止指令已下发"
}

# ========== 步骤 2：解压恢复 ==========
# 性能优化要点（避免超时）：
#   1. 用 mv 替代 cp -a 备份新镜像 .hermes → 瞬间完成（同分区 rename），省去全量拷贝
#   2. tar 解压时排除 hermes_agent/ → 避免恢复旧源码（rebuild_venv 会从新镜像恢复）
#   3. chown -R 移到合并前 → 只覆盖恢复的备份文件，不含新镜像已正确属主的文件
restore_files() {
    echo ""
    echo ">>> [步骤 2/6] 解压恢复数据"
    local _step2_start; _step2_start="$(date +%s)"
    echo "[restore] 备份包: $ARCHIVE_PATH ($(du -sh "$ARCHIVE_PATH" 2>/dev/null | cut -f1))"

    # 备份新镜像默认 .hermes —— 用 mv 替代 cp -a（同分区 rename，瞬间完成）
    # 合并时用 cp -an 回补新镜像独有文件（不覆盖备份已恢复的内容）
    if [ -d "$HERMES_HOME" ]; then
        FRESH_HERMES_BACKUP="${HERMES_HOME}.fresh.$(date +%s)"
        echo "[restore] 移动新镜像 .hermes 到: $FRESH_HERMES_BACKUP (mv 替代 cp -a)"
        mv "$HERMES_HOME" "$FRESH_HERMES_BACKUP" || fail_exit "restore" "无法移动 $HERMES_HOME 到 $FRESH_HERMES_BACKUP"
        echo "[restore] 新镜像 .hermes 大小: $(du -sh "$FRESH_HERMES_BACKUP" 2>/dev/null | cut -f1)"
    else
        echo "[restore] $HERMES_HOME 不存在，跳过新镜像备份"
    fi

    # 解压到临时目录（新格式 staging 含 hermes_home/ + 外部保留路径）
    # 排除 hermes-agent/：备份已排除该目录，此处双保险（兼容旧格式备份包含 hermes-agent 的情况）。
    # 新镜像出厂已含匹配的 hermes-agent 源码+venv，恢复旧版源码是冗余且会触发 venv 重建。
    TMP_RESTORE="/tmp/hermes-restore-staging-$$"
    mkdir -p "$TMP_RESTORE"
    echo "[restore] 解压到: $TMP_RESTORE (排除 hermes-agent)"
    tar -xzf "$ARCHIVE_PATH" \
        --exclude='hermes_home/hermes-agent' \
        --exclude='hermes-agent' \
        -C "$TMP_RESTORE" 2>&1 || fail_exit "restore" "解压失败: $ARCHIVE_PATH -> $TMP_RESTORE"
    echo "[restore] 解压后文件列表:"
    find "$TMP_RESTORE" -maxdepth 2 -type f -o -type d | head -30 | while read -r _f; do echo "  $_f"; done
    echo "[restore]   共 $(find "$TMP_RESTORE" -type f | wc -l | tr -d ' ') 个文件"

    # 恢复 .hermes：新格式有 hermes_home/ → $HERMES_HOME；旧格式直接就是 .hermes 内容
    if [ -d "${TMP_RESTORE}/hermes_home" ]; then
        mkdir -p "$HERMES_HOME"
        cp -a "${TMP_RESTORE}/hermes_home/." "$HERMES_HOME/"
        echo "✓ hermes_home 已恢复（新格式）"
    else
        mkdir -p "$HERMES_HOME"
        cp -a "${TMP_RESTORE}/." "$HERMES_HOME/"
        echo "✓ 旧格式备份已恢复（兼容模式）"
    fi

    # 先修正恢复文件的属主（仅覆盖备份恢复的文件，不含后续合并的新镜像文件）
    chown -R "$TARGET_UID":"$TARGET_GID" "$HERMES_HOME" 2>/dev/null || true

    # 回补新镜像独有文件（备份同名已覆盖，新镜像新增的不在备份中需保留）
    # cp -an 保留新镜像文件原始属主（正确），无需再次 chown
    if [ -n "${FRESH_HERMES_BACKUP:-}" ] && [ -d "$FRESH_HERMES_BACKUP" ]; then
        echo "[restore] 回补新镜像独有文件（不覆盖备份已恢复的内容）..."
        cp -an "${FRESH_HERMES_BACKUP}/." "$HERMES_HOME/" 2>/dev/null || true
        echo "✓ 新镜像独有文件已合并"
    fi

    # 恢复外部保留路径（仅新格式 staging 有 hermes_home/ 时处理额外的顶级项）。
    # 注意：外部保留路径可能以 . 开头（如 .memory-tencentdb），bash 的 * 默认不匹配
    # dotfiles，必须 shopt -s dotglob 才能遍历到，否则隐藏目录会被静默跳过。
    if [ -d "${TMP_RESTORE}/hermes_home" ]; then
        local extra_count=0
        shopt -s dotglob
        for item in "${TMP_RESTORE}"/*; do
            [ ! -e "$item" ] && continue
            [ "$(basename "$item")" = "hermes_home" ] && continue
            local target="$HOME/$(basename "$item")"
            mkdir -p "$target"
            echo "[restore] 恢复外部保留路径: $(basename "$item") → $target"
            # 用 "$item/." 复制目录内容而非目录本身，避免目标已存在时嵌套
            # （新镜像可能已创建空 ~/.memory-tencentdb/，cp -a src dst 会变成 dst/src/...）
            cp -a "$item/." "$target/" 2>/dev/null || echo "[restore] ⚠ 恢复 $(basename "$item") 失败（忽略）"
            extra_count=$((extra_count + 1))
        done
        shopt -u dotglob
        if [ "$extra_count" -gt 0 ]; then
            echo "[restore] ✓ 外部保留路径已恢复（${extra_count} 项）"
        else
            echo "[restore] 无外部保留路径需恢复"
        fi
    fi

    rm -rf "$TMP_RESTORE"
    echo "✓ 解压完成，已修正属主为 $RUNTIME_USER (耗时: $(( ($(date +%s) - _step2_start) ))s)"
}

# ========== 步骤 3：验证/重建 venv ==========
# 全流程优化：
#   备份已排除 hermes-agent 整个目录（backup_pre_reinstall_hermes.sh），
#   restore_files 也排除了 hermes-agent（tar --exclude），
#   因此恢复后 hermes-agent 保持新镜像出厂状态（源码+venv 完全匹配，已 pip install -e）。
#
#   理论上不需要任何重建。但为应对边缘情况（旧格式备份/手工修改等），仍做验证：
#     1. 优先验证新镜像 venv 可直接 import hermes_cli → 跳过，秒级完成
#     2. 验证失败 → 从 FRESH_HERMES_BACKUP 恢复 hermes-agent（mv 瞬间完成）→ 再验证
#     3. 仍失败 → pip install --no-deps -e（无需网络，30s 级）
#     4. 最终兜底 → 完整重建（venv + pip install -e，需网络，仅极端情况）
rebuild_venv() {
    echo ""
    echo ">>> [步骤 3/6] 验证/重建 hermes-agent venv"
    local _step3_start; _step3_start="$(date +%s)"

    local agent_dir="${HERMES_HOME}/hermes-agent"

    # restore_files 用 mv 把新镜像 .hermes 移到了 FRESH_HERMES_BACKUP。
    # 如果恢复后 hermes-agent 不存在（新格式备份已排除），从 FRESH_HERMES_BACKUP 移回来。
    if [ ! -d "$agent_dir" ]; then
        echo "[rebuild_venv] hermes-agent 不在 $HERMES_HOME，从新镜像备份恢复"
        if [ -n "${FRESH_HERMES_BACKUP:-}" ] && [ -d "${FRESH_HERMES_BACKUP}/hermes-agent" ]; then
            # mv 瞬间完成（同分区 rename）
            if ! mv "${FRESH_HERMES_BACKUP}/hermes-agent" "${agent_dir}" 2>/dev/null; then
                echo "[rebuild_venv] mv 失败（可能跨分区），回退 cp -a"
                run_as_runtime_user cp -a "${FRESH_HERMES_BACKUP}/hermes-agent" "${agent_dir}"
            fi
            chown -R "$TARGET_UID":"$TARGET_GID" "${agent_dir}" 2>/dev/null || true
            echo "[rebuild_venv] ✓ hermes-agent 已从新镜像恢复"
        else
            fail_exit "venv" "hermes-agent 不存在且无法从新镜像目录恢复"
        fi
    else
        # hermes-agent 已存在（旧格式备份包含/边缘情况），需要确认是新镜像版本
        # 优先用 FRESH_HERMES_BACKUP 的新镜像版本替换
        if [ -n "${FRESH_HERMES_BACKUP:-}" ] && [ -d "${FRESH_HERMES_BACKUP}/hermes-agent" ]; then
            echo "[rebuild_venv] 用新镜像 hermes-agent 替换现有版本 (mv)"
            run_as_runtime_user rm -rf "${agent_dir}"
            if ! mv "${FRESH_HERMES_BACKUP}/hermes-agent" "${agent_dir}" 2>/dev/null; then
                echo "[rebuild_venv] mv 失败（可能跨分区），回退 cp -a"
                run_as_runtime_user cp -a "${FRESH_HERMES_BACKUP}/hermes-agent" "${agent_dir}"
            fi
            chown -R "$TARGET_UID":"$TARGET_GID" "${agent_dir}" 2>/dev/null || true
            echo "[rebuild_venv] ✓ hermes-agent 已替换为新镜像版本"
        fi
    fi

    if [ ! -d "$agent_dir" ]; then
        fail_exit "venv" "hermes-agent 目录不存在"
    fi

    # ===== 验证路径：新镜像出厂 venv 应可直接使用，无需 pip install =====
    local venv_python="${agent_dir}/venv/bin/python"
    local venv_pip="${agent_dir}/venv/bin/pip"

    _verify_venv_ready() {
        # 验证 venv python 可执行 + 能 import hermes_cli（新镜像已 pip install -e）
        [ -x "$venv_python" ] || return 1
        run_as_runtime_user "$venv_python" -c "import hermes_cli; print('ok')" 2>/dev/null
    }

    echo "[rebuild_venv] 验证新镜像 venv 是否可直接使用..."
    if _verify_venv_ready; then
        echo "[rebuild_venv] ✓ 新镜像 venv 已就绪，无需重建 (耗时: $(( $(date +%s) - _step3_start ))s)"
        return 0
    fi
    echo "[rebuild_venv] venv 验证失败，尝试 --no-deps 重链源码（无需网络）"

    # ===== 回退 1：pip install --no-deps -e（重链 egg-link，无需网络下载依赖） =====
    if [ -x "$venv_pip" ]; then
        if run_as_runtime_user "$venv_pip" install --no-deps -e "$agent_dir" 2>&1; then
            if _verify_venv_ready; then
                echo "[rebuild_venv] ✓ --no-deps 重链后 venv 就绪 (耗时: $(( $(date +%s) - _step3_start ))s)"
                chown -R "$TARGET_UID":"$TARGET_GID" "${agent_dir}/venv" 2>/dev/null || true
                return 0
            fi
        fi
        echo "[rebuild_venv] ⚠ --no-deps 重链后验证仍失败，回退完整重建"
    else
        echo "[rebuild_venv] ⚠ venv/bin/pip 不可用，回退完整重建"
    fi

    # ===== 回退 2：完整重建（创建 venv + pip install -e，需网络） =====
    echo "[rebuild_venv] 完整重建 venv (需网络下载依赖，可能耗时数分钟)..."

    # 清理旧 venv
    if [ -d "${agent_dir}/venv" ]; then
        run_as_runtime_user rm -rf "${agent_dir}/venv"
    fi

    # 确认系统 Python 可用
    local sys_python
    sys_python="$(command -v python3 2>/dev/null || command -v python 2>/dev/null || echo '')"
    if [ -z "$sys_python" ]; then
        fail_exit "venv" "系统未找到 Python，无法重建 hermes-agent venv"
    fi
    echo "[rebuild_venv] 系统 Python: $sys_python ($($sys_python --version 2>&1))"

    # 确保 python3-venv 可用
    _ensure_venv_module() {
        local tmp_venv="/tmp/.hermes_venv_check_$$"
        if $sys_python -m venv "$tmp_venv" 2>/dev/null && [ -x "$tmp_venv/bin/pip" ]; then
            rm -rf "$tmp_venv"
            return 0
        fi
        rm -rf "$tmp_venv" 2>/dev/null || true
        return 1
    }

    if ! _ensure_venv_module; then
        echo "[rebuild_venv] python3-venv 未安装，尝试 apt install..."
        local py_ver
        py_ver="$($sys_python -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")' 2>/dev/null || echo '')"
        local pkg="python3-venv"
        [ -n "$py_ver" ] && pkg="python3-venv python${py_ver}-venv"

        if [ "$(id -u)" -eq 0 ]; then
            apt-get update -qq 2>/dev/null || fail_exit "venv" "apt-get update 失败，无法安装 python venv 模块"
            apt-get install -y $pkg 2>&1 || fail_exit "venv" "apt-get install python venv 模块失败"
        elif command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
            sudo apt-get update -qq 2>/dev/null || fail_exit "venv" "sudo apt-get update 失败，无法安装 python venv 模块"
            sudo apt-get install -y $pkg 2>&1 || fail_exit "venv" "sudo apt-get install python venv 模块失败"
        else
            fail_exit "venv" "无 root 或免密 sudo 权限，无法安装 python venv 模块"
        fi
        _ensure_venv_module || fail_exit "venv" "python venv 模块安装后仍不可用"
        echo "[rebuild_venv] ✓ python3-venv 安装完成"
    fi

    if ! run_as_runtime_user "$sys_python" -m venv "${agent_dir}/venv"; then
        fail_exit "venv" "venv 创建失败"
    fi
    echo "[rebuild_venv] ✓ venv 已创建"

    local venv_pip_rebuild="${agent_dir}/venv/bin/pip"
    [ -x "$venv_pip_rebuild" ] || fail_exit "venv" "venv/bin/pip 不可用"

    echo "[rebuild_venv] pip install -e ${agent_dir}"
    if run_as_runtime_user "$venv_pip_rebuild" install -e "$agent_dir" 2>&1; then
        echo "[rebuild_venv] ✓ hermes-agent 安装完成"
    else
        echo "[rebuild_venv] pip install 失败，尝试 --no-deps 回退..."
        run_as_runtime_user "$venv_pip_rebuild" install --no-deps -e "$agent_dir" 2>&1 || fail_exit "venv" "pip 安装 hermes-agent 失败"
    fi

    _verify_venv_ready || fail_exit "venv" "venv 无法导入 hermes_cli"
    chown -R "$TARGET_UID":"$TARGET_GID" "${agent_dir}/venv" 2>/dev/null || true
    echo "[rebuild_venv] venv 属主已修正为 $RUNTIME_USER (总耗时: $(( $(date +%s) - _step3_start ))s)"
}

# ========== 步骤 4：归档 tgz 到 upgrades/<ts>/ ==========
# restore_files 已重建 ~/.hermes/，此处把 SMH 下载的 tgz 复制到 upgrades/ 留存
preserve_upgrade_snapshot() {
    if [ -z "${ARCHIVE_PATH:-}" ] || [ ! -f "$ARCHIVE_PATH" ]; then
        echo "[preserve] 未找到 ARCHIVE_PATH，跳过归档沉淀"
        return 0
    fi

    echo ""
    echo ">>> [步骤 4/6] 沉淀本次备份到 ~/.hermes/upgrades/"

    local ts="${UPGRADE_TS:-$(date +%Y%m%d_%H%M%S)}"
    local upgrade_dir="${HERMES_HOME}/upgrades/${ts}"

    mkdir -p "$upgrade_dir" 2>/dev/null || { echo "[preserve] ⚠ 无法创建 $upgrade_dir，跳过归档"; return 0; }

    local target="${upgrade_dir}/hermes-backup.tgz"
    echo "[preserve] 移动 $ARCHIVE_PATH → $target (mv 替代 cp，省去大文件拷贝)"
    # mv 替代 cp：tgz 在步骤 6 会被 cleanup_temp 删除，此处直接移动即可
    if ! mv -f -- "$ARCHIVE_PATH" "$target" 2>/dev/null; then
        echo "[preserve] mv 失败（可能跨分区），回退 cp"
        cp -f -- "$ARCHIVE_PATH" "$target" 2>/dev/null || { echo "[preserve] ⚠ 复制失败，跳过归档"; return 0; }
    fi
    echo "[preserve] 归档大小: $(du -sh "$target" 2>/dev/null | cut -f1)"

    local meta="${upgrade_dir}/upgrade.info"
    {
        echo "upgrade_ts=$ts"
        echo "archive_source=$ARCHIVE_PATH"
        echo "archive_size=$(stat -c%s "$target" 2>/dev/null || stat -f%z "$target" 2>/dev/null || echo unknown)"
        echo "restored_at=$(date -Iseconds)"
        echo "runtime_user=$RUNTIME_USER"
        echo "hermes_home=$HERMES_HOME"
    } > "$meta" 2>/dev/null || true

    chown -R "$TARGET_UID":"$TARGET_GID" "${HERMES_HOME}/upgrades" 2>/dev/null || true
    echo "✓ 已沉淀本次备份: $target"
    echo "  提示：如需人工回滚，可执行 tar -xzf $target -C $HERMES_HOME/"
}

# ========== 修正 /etc/acli 权限 ==========
# 重装后 machine_id 变化，acli 的 identity self-heal 需要写 /etc/acli/ 更新 identity.yaml。
# 但 /etc/acli/ 是镜像构建时 root 创建的（755），运行用户（如 ubuntu）无写权限，
# 导致 "identity self-heal: persist failed: permission denied"。
# 此处在首次调用 acli 前修正属主，确保 self-heal 能正常持久化。
fix_acli_identity_perms() {
    local acli_dir="/etc/acli"
    [ -d "$acli_dir" ] || return 0

    # 已有写权限则无需处理
    if [ -w "$acli_dir" ]; then
        return 0
    fi

    local dir_owner
    dir_owner="$(stat -c '%U:%G' "$acli_dir" 2>/dev/null || stat -f '%Su:%Sg' "$acli_dir" 2>/dev/null || echo '?')"
    echo "[acli-fix] /etc/acli 不可写（owner=$dir_owner, 当前用户=$RUNTIME_USER uid=$TARGET_UID），尝试修正属主..."

    if [ "$(id -u)" -eq 0 ]; then
        if chown -R "$TARGET_UID":"$TARGET_GID" "$acli_dir" 2>/dev/null; then
            echo "[acli-fix] ✓ /etc/acli 属主已修正为 $RUNTIME_USER (root 直接 chown)"
            return 0
        fi
    elif command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
        if sudo -n chown -R "$TARGET_UID":"$TARGET_GID" "$acli_dir" 2>/dev/null; then
            echo "[acli-fix] ✓ /etc/acli 属主已修正为 $RUNTIME_USER (sudo chown)"
            return 0
        fi
    fi

    echo "[acli-fix] ⚠ 无 root/sudo 权限，无法修正 /etc/acli 属主"
    echo "[acli-fix]   影响：acli 每次调用会报 identity self-heal permission denied"
    echo "[acli-fix]   缓解：gateway 仍可运行（in-memory refresh ok），但 identity 未持久化"
}

# ========== 步骤 4：启动 Gateway ==========
# 优先 restart，回退 start；启动后采样 status + 进程列表 + dump 日志
start_hermes_gateway() {
    echo ""
    echo ">>> [步骤 5/6] 启动 Hermes Gateway"

    echo "[diag] whoami=$(id -un) uid=$(id -u) gid=$(id -g)"
    echo "[diag] PATH=$PATH"
    echo "[diag] acli 路径(当前): $(command -v acli 2>/dev/null || echo '<not found>')"
    echo "[diag] harness 路径(当前): $(command -v harness 2>/dev/null || echo '<not found>')"
    echo "[diag] acli 路径($RUNTIME_USER 视角): $(run_as_runtime_user bash -lc 'command -v acli 2>/dev/null || echo "<not found>"' 2>&1 || true)"
    echo "[diag] harness 路径($RUNTIME_USER 视角): $(run_as_runtime_user bash -lc 'command -v harness 2>/dev/null || echo "<not found>"' 2>&1 || true)"

    # TAT 环境下 systemctl --user 依赖 D-Bus session；非登录 shell 可能缺失此变量，
    # 导致 systemctl --user stop/start 静默失败。此处按目标 uid 推导并导出。
    if [ -z "${DBUS_SESSION_BUS_ADDRESS:-}" ] && [ -n "${TARGET_UID:-}" ] && [ -S "/run/user/${TARGET_UID}/bus" ]; then
        export DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/${TARGET_UID}/bus"
        echo "[diag] 已设置 DBUS_SESSION_BUS_ADDRESS=${DBUS_SESSION_BUS_ADDRESS}"
    fi

    # 修正 /etc/acli 权限（重装后 machine_id 变化，acli 需 self-heal 更新 identity）
    fix_acli_identity_perms

    # 探测 CLI 支持的子命令：优先 restart，回退 start
    _detect_subcmd() {
        local cli="$1"
        # 缓存结果：同一 CLI 只探测一次
        local cache_var="_SUBCMD_CACHE_${cli//-/_}"
        if [ -n "${!cache_var:-}" ]; then
            printf '%s\n' "${!cache_var}"
            return 0
        fi
        local help_out
        help_out="$(run_as_runtime_user bash -lc "${cli} gateway --help 2>&1 || ${cli} gateway 2>&1" 2>&1 || true)"
        local cmds
        cmds="$(printf '%s\n' "$help_out" | awk '/Available Commands:/{flag=1;next} /^[A-Za-z]+:$/{flag=0} flag {print $1}')"
        local result=""
        printf '%s\n' "$cmds" | grep -qx 'restart' && { result="restart"; }
        [ -z "$result" ] && printf '%s\n' "$cmds" | grep -qx 'start' && { result="start"; }
        if [ -z "$result" ]; then
            run_as_runtime_user bash -lc "${cli} gateway restart --help >/dev/null 2>&1" && result="restart"
        fi
        [ -z "$result" ] && run_as_runtime_user bash -lc "${cli} gateway start --help >/dev/null 2>&1" && result="start"
        printf '%s\n' "$result"
        # 写缓存（eval 用于动态变量名赋值）
        eval "${cache_var}=\"${result}\""
    }

    _try_invoke() {
        local cli="$1"
        local sub
        sub="$(_detect_subcmd "$cli")"
        [ -z "$sub" ] && { echo "[start] $cli 不支持 restart/start 子命令，跳过"; return 1; }

        local max_try=2
        local attempt=1
        while [ "$attempt" -le "$max_try" ]; do
            echo "[start] 调用: $cli gateway $sub (user=$RUNTIME_USER, attempt=$attempt/$max_try)"
            if run_as_runtime_user bash -lc "$cli gateway $sub"; then
                echo "[start] $cli gateway $sub 退出码 0 (attempt=$attempt)"
                return 0
            fi
            echo "[start] $cli gateway $sub 退出码非 0 (attempt=$attempt)"
            if [ "$attempt" -lt "$max_try" ]; then
                echo "[start] 等待 5s 后重试..."
                sleep 5
            fi
            attempt=$((attempt + 1))
        done
        return 1
    }

    local invoked=0
    local start_tool=""

    if run_as_runtime_user bash -lc 'command -v acli >/dev/null 2>&1'; then
        _try_invoke acli && { invoked=1; start_tool="acli"; }
    else
        echo "[start] 跳过 acli：$RUNTIME_USER 视角下 acli 不可用"
    fi

    if [ "$invoked" -eq 0 ] && run_as_runtime_user bash -lc 'command -v harness >/dev/null 2>&1'; then
        _try_invoke harness && { invoked=1; start_tool="harness"; }
    fi

    if [ "$invoked" -eq 0 ]; then
        fail_exit "gateway" "acli/harness 均无法成功执行 gateway restart/start"
    fi
    echo "✓ Gateway 启动指令已下发（via $start_tool）"

    # 采样 status + 进程列表
    sleep 2
    local status_raw=""
    if run_as_runtime_user bash -lc 'command -v acli >/dev/null 2>&1'; then
        echo "[probe] acli gateway status:"
        status_raw="$(run_as_runtime_user bash -lc 'acli gateway status' 2>&1 || true)"
        echo "${status_raw:-<empty>}"
    elif run_as_runtime_user bash -lc 'command -v harness >/dev/null 2>&1'; then
        echo "[probe] harness gateway status:"
        status_raw="$(run_as_runtime_user bash -lc 'harness gateway status' 2>&1 || true)"
        echo "${status_raw:-<empty>}"
    else
        echo "[probe] 跳过 status 采样：未找到 acli/harness"
    fi

    local running="unknown"
    printf '%s' "$status_raw" | grep -q '"running"[[:space:]]*:[[:space:]]*true' && running="true"
    printf '%s' "$status_raw" | grep -q '"running"[[:space:]]*:[[:space:]]*false' && running="false"
    echo "[probe] running=$running"

    echo "[probe] 当前 hermes 相关进程列表:"
    ps -ef 2>/dev/null | grep -E 'hermes_cli\.main|hermes-gateway|harness.*gateway|acli.*gateway' | grep -v grep || echo "  <无匹配进程>"

    if [ "$running" != "true" ]; then
        echo "[probe] gateway 未处于 running 状态，dump 常见日志末尾:"
        local log_candidates=(
            "$HERMES_HOME/logs/gateway.log"
            "$HERMES_HOME/logs/gateway.err"
            "$HERMES_HOME/logs/hermes.log"
            "$HERMES_HOME/gateway.log"
        )
        local found_any=0
        for f in "${log_candidates[@]}"; do
            if [ -f "$f" ]; then
                found_any=1
                echo "----- tail -n 80 $f -----"
                tail -n 80 "$f" 2>&1 || true
                echo "----- end of $f -----"
            fi
        done
        [ "$found_any" -eq 0 ] && echo "[probe] 未找到任何 gateway 日志文件"
        if [ -d "$HERMES_HOME/logs" ]; then
            echo "[probe] $HERMES_HOME/logs 目录内容:"
            ls -la "$HERMES_HOME/logs" 2>&1 || true
        fi
        fail_exit "gateway" "gateway status 未返回 running=true"
    fi
}

# ========== 步骤 6：清理 /tmp 临时文件 ==========
cleanup_temp() {
    echo ""
    echo ">>> [步骤 6/6] 清理临时文件"
    if [ -n "$ARCHIVE_PATH" ] && [[ "$ARCHIVE_PATH" == /tmp/hermes-restore-*.tgz ]]; then
        rm -f "$ARCHIVE_PATH" && echo "✓ 已删除: $ARCHIVE_PATH" || echo "⚠ 删除失败: $ARCHIVE_PATH"
    else
        echo "  无需清理: $ARCHIVE_PATH (非 /tmp 临时文件)"
    fi
    # 清理可能残留的 venv 检查临时目录
    rm -rf /tmp/.hermes_venv_check_* 2>/dev/null || true
}

# ========== 主流程 ==========
# _step_timer：步骤计时工具，打印每步耗时便于定位超时瓶颈
_step_timer() {
    local label="$1"
    local start="$2"
    local now; now="$(date +%s)"
    echo "⏱  [计时] ${label}: $(( now - start ))s"
}

main() {
    local _t0; _t0="$(date +%s)"

    download_from_smh
    local _t1; _t1="$(date +%s)"; _step_timer "download_from_smh" "$_t0"

    stop_hermes_gateway
    local _t2; _t2="$(date +%s)"; _step_timer "stop_hermes_gateway" "$_t1"

    restore_files
    local _t3; _t3="$(date +%s)"; _step_timer "restore_files" "$_t2"

    rebuild_venv
    local _t4; _t4="$(date +%s)"; _step_timer "rebuild_venv" "$_t3"

    preserve_upgrade_snapshot
    local _t5; _t5="$(date +%s)"; _step_timer "preserve_upgrade_snapshot" "$_t4"

    start_hermes_gateway
    local _t6; _t6="$(date +%s)"; _step_timer "start_hermes_gateway" "$_t5"

    # 仅在 Gateway 已确认 running 后删除新镜像临时目录；失败路径会由 EXIT trap 回滚。
    if [ -n "${FRESH_HERMES_BACKUP:-}" ] && [ -d "$FRESH_HERMES_BACKUP" ]; then
        if rm -rf "$FRESH_HERMES_BACKUP"; then
            echo "✓ 已清理新镜像临时目录: $FRESH_HERMES_BACKUP"
            FRESH_HERMES_BACKUP=""
        else
            echo "⚠ 无法清理新镜像临时目录，保留供后续排障: $FRESH_HERMES_BACKUP"
        fi
    fi

    cleanup_temp
    _step_timer "cleanup_temp" "$_t6"
    _step_timer "总计" "$_t0"

    echo ""
    echo "=== 恢复完成 ==="
    echo "RESTORE_OK" > /tmp/restore_status
    echo "RESTORE_OK"
}

main "$@"
exit 0