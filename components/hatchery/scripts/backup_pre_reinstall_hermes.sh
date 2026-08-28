#!/bin/bash
set -euo pipefail

# 重装前备份 ~/.hermes/ + 插件外部保留路径 → hermes-state-<ts>.tgz。
# 排除 venv/logs/缓存/__pycache__ 等不可移植或可重装恢复的内容。

# ========== 路径配置 ==========
HERMES_HOME="${HERMES_STATE_DIR:-$HOME/.hermes}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
ARCHIVE_NAME="hermes-state-${TIMESTAMP}.tgz"
ARCHIVE_PATH="/tmp/${ARCHIVE_NAME}"

# ========== 解析参数 ==========
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help) echo "Usage: $0"; exit 0 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

echo "=== Hermes 重装前数据备份 ==="
echo "状态目录: $HERMES_HOME"
echo "压缩包路径: $ARCHIVE_PATH"

# 同时输出到终端（供 TAT RunScript 抓取）和落盘日志
LOG_DIR="${HERMES_HOME}/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
SCRIPT_NAME="backup_pre_reinstall_hermes"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "状态目录: $HERMES_HOME"
echo "压缩包路径: $ARCHIVE_PATH"

LOG_PREFIX="[backup-pre-reinstall-hermes]"
KEEP=3
UPGRADES_DIR="${HERMES_HOME}/upgrades"

# 打包前清理旧升级快照，保留最近 KEEP 个（YYYYMMDD_HHMMSS 字典序==时间序）
cleanup_old_upgrade_snapshots() {
    echo ""
    echo "${LOG_PREFIX} [step 0/3] 清理旧 upgrade 快照（保留最近 $KEEP 个）"

    if [ ! -d "$UPGRADES_DIR" ]; then
        echo "${LOG_PREFIX}   upgrades 目录不存在，跳过清理"
        return 0
    fi

    local total
    total=$(find "$UPGRADES_DIR" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l | tr -d ' ')
    if [ "${total:-0}" -le "$KEEP" ]; then
        echo "${LOG_PREFIX}   子目录共 ${total:-0} 个，未超过保留上限 $KEEP，无需清理"
        return 0
    fi

    echo "${LOG_PREFIX}   子目录共 $total 个，将清理较旧的 $((total - KEEP)) 个"

    local removed=0
    local d
    while IFS= read -r d; do
        [ -z "$d" ] && continue
        if rm -rf -- "$d" 2>/dev/null; then
            removed=$((removed + 1))
            echo "${LOG_PREFIX}   ✓ 已删除: $(basename -- "$d")"
        else
            echo "${LOG_PREFIX}   ⚠ 删除失败（忽略）: $d"
        fi
    done < <(find "$UPGRADES_DIR" -mindepth 1 -maxdepth 1 -type d 2>/dev/null \
        | sort -r | tail -n +"$((KEEP + 1))")

    echo "${LOG_PREFIX}   ✓ upgrades 清理完成，共删除 $removed 个旧目录"
}

# 打包完成后、Go 上传 SMH 前，将 tgz 复制到 upgrades/<ts>/ 留存
preserve_local_snapshot() {
    echo ""
    echo "${LOG_PREFIX} [step 2.5/3] 沉淀本次备份到 $UPGRADES_DIR"

    if [ ! -f "$ARCHIVE_PATH" ]; then
        echo "${LOG_PREFIX}   ⚠ ARCHIVE_PATH 不存在，跳过快照沉淀"
        return 0
    fi

    local upgrade_dir="${UPGRADES_DIR}/${TIMESTAMP}"

    if ! mkdir -p "$upgrade_dir" 2>/dev/null; then
        echo "${LOG_PREFIX}   ⚠ 无法创建 $upgrade_dir，跳过快照沉淀（不阻塞备份）"
        return 0
    fi

    local target="${upgrade_dir}/hermes-backup.tgz"
    # cp 而非 mv：Go 层仍需从 ARCHIVE_PATH 上传 SMH
    if ! cp -f -- "$ARCHIVE_PATH" "$target" 2>/dev/null; then
        echo "${LOG_PREFIX}   ⚠ 复制 $ARCHIVE_PATH → $target 失败，跳过快照沉淀"
        return 0
    fi

    local meta="${upgrade_dir}/upgrade.info"
    local archive_bytes
    archive_bytes=$(stat -c%s "$ARCHIVE_PATH" 2>/dev/null || stat -f%z "$ARCHIVE_PATH" 2>/dev/null || echo unknown)
    {
        echo "upgrade_ts=$TIMESTAMP"
        echo "archive_source=$ARCHIVE_PATH"
        echo "archive_size=$archive_bytes"
        echo "backup_at=$(date -Iseconds)"
        echo "user=$(whoami)"
        echo "hostname=$(hostname)"
        echo "hermes_home=$HERMES_HOME"
    } > "$meta" 2>/dev/null || true

    echo "${LOG_PREFIX}   ✓ 已沉淀本次备份: $target ($(du -sh "$target" | cut -f1))"
    echo "${LOG_PREFIX}   提示：如需人工回滚，可执行以下命令："
    echo "${LOG_PREFIX}     mkdir /tmp/hermes-manual-restore && tar -xzf $target -C /tmp/hermes-manual-restore"
    echo "${LOG_PREFIX}     cp -a /tmp/hermes-manual-restore/hermes_home/. ${HERMES_HOME}/"
    echo "${LOG_PREFIX}     # 若有外部保留路径，同步恢复"
    echo "${LOG_PREFIX}     for d in /tmp/hermes-manual-restore/*; do [ \"\$(basename \"\$d\")\" = hermes_home ] && continue; cp -a \"\$d\" \"\$HOME/\"; done"
}

# ========== 停服（避免 sqlite 并发写导致 tar 失败）==========
_GATEWAY_STOPPED=0

_restart_gateway_on_error() {
    if [[ "$_GATEWAY_STOPPED" -eq 1 ]]; then
        echo "⚠ 脚本异常，尝试重新拉起 Hermes Gateway..."
        for _u in hermes hermes-gateway harness-gateway; do
            systemctl --user start "$_u" 2>/dev/null && return
        done
        command -v acli >/dev/null 2>&1 && acli gateway restart 2>/dev/null || true
        command -v harness >/dev/null 2>&1 && harness gateway restart 2>/dev/null || true
    fi
}

stop_gateway() {
    echo "停止 Hermes Gateway..."
    local stopped=0

    # 优先 systemd user units，再 pkill 兜底
    for unit in hermes hermes-gateway harness-gateway; do
        if systemctl --user stop "$unit" 2>/dev/null; then
            echo "✓ systemctl --user stop $unit 成功"
            stopped=1
            break
        fi
    done

    if [ "$stopped" -eq 0 ]; then
        echo "systemctl --user stop 未生效，尝试 pkill 'hermes_cli.main gateway' 兜底"
        pkill -u "$(id -un)" -f 'hermes_cli\.main gateway' 2>/dev/null || true
        sleep 1
        if ! pgrep -u "$(id -un)" -f 'hermes_cli\.main gateway' >/dev/null 2>&1; then
            stopped=1
            echo "✓ pkill 后未发现残留进程，视为已停服"
        fi
    fi

    if [ "$stopped" -eq 1 ]; then
        _GATEWAY_STOPPED=1
        trap '_restart_gateway_on_error' ERR EXIT
        sleep 2
    else
        echo "⚠ 未能停服但也未发现残留进程，视为 gateway 已不在跑，继续备份"
    fi
}

# 插件外部保留路径（$HOME 相对路径）。
# 注意：此列表为 config/agent_plugin_preserve_paths.json 的硬编码副本（声明性配置），
# 脚本不动态读取 JSON。新增/修改保留路径时需同时更新此处和 JSON 文件。
declare -a PRESERVE_PATHS=(
    ".memory-tencentdb/memory-tdai"
)

# ========== 打包 ==========
create_archive() {
    echo "开始打包 Hermes 目录..."

    if [ ! -d "$HERMES_HOME" ]; then
        echo "✗ Hermes 状态目录不存在: $HERMES_HOME"
        exit 1
    fi

    echo "打包目录: $HERMES_HOME"

    # 创建 staging 目录
    STAGING_DIR="/tmp/hermes-backup-staging-${TIMESTAMP}"
    mkdir -p "${STAGING_DIR}/hermes_home"
    echo "${LOG_PREFIX} staging 目录: ${STAGING_DIR}"

    # 复制 .hermes 到 staging
    echo "${LOG_PREFIX} 复制 .hermes 内容到 staging..."
    cp -a "${HERMES_HOME}/." "${STAGING_DIR}/hermes_home/"

    # 清理不可移植/可重装恢复的内容
    # hermes-agent 整个目录排除：纯代码目录，无用户数据；新镜像出厂已含匹配的源码+venv。
    # 恢复旧源码会触发 venv 重建（pip install -e 需网络下载，耗时 5-15min），是恢复超时的主因。
    rm -rf "${STAGING_DIR}/hermes_home/upgrades" \
           "${STAGING_DIR}/hermes_home/hermes-agent" \
           "${STAGING_DIR}/hermes_home/.npm" \
           "${STAGING_DIR}/hermes_home/npm-cache" \
           "${STAGING_DIR}/hermes_home/.cache" \
           "${STAGING_DIR}/hermes_home/tmp" 2>/dev/null || true
    find "${STAGING_DIR}/hermes_home" -name "*.log" -delete 2>/dev/null || true
    find "${STAGING_DIR}/hermes_home" -name "__pycache__" -type d -exec rm -rf {} + 2>/dev/null || true
    find "${STAGING_DIR}/hermes_home" -name "*.pyc" -delete 2>/dev/null || true
    echo "${LOG_PREFIX} ✓ .hermes 内容已复制到 staging（已清理排除项）"

    # 复制外部保留路径
    for path in "${PRESERVE_PATHS[@]}"; do
        src="$HOME/$path"
        if [ -d "$src" ] || [ -f "$src" ]; then
            dst="${STAGING_DIR}/${path}"
            echo "${LOG_PREFIX} 保留外部路径: $src"
            mkdir -p "$(dirname "$dst")"
            cp -a "$src" "$dst"
        else
            echo "${LOG_PREFIX} 外部保留路径不存在，跳过: $src"
        fi
    done

    # 打包 staging 目录
    echo "${LOG_PREFIX} 打包 staging 目录..."
    set +e
    tar --warning=no-file-changed --warning=no-file-removed \
        -czf "$ARCHIVE_PATH" \
        -C "$STAGING_DIR" \
        .
    local tar_rc=$?
    set -e
    [ "$tar_rc" -ge 2 ] && echo "✗ tar 打包失败（rc=$tar_rc，fatal）" && rm -rf "$STAGING_DIR" && exit 1
    [ "$tar_rc" -eq 1 ] && echo "⚠ tar 报 warning（rc=1），已忽略"

    # 清理 staging
    rm -rf "$STAGING_DIR"

    local archive_bytes
    archive_bytes=$(stat -c%s "$ARCHIVE_PATH" 2>/dev/null || stat -f%z "$ARCHIVE_PATH" 2>/dev/null || du -b "$ARCHIVE_PATH" | cut -f1)
    echo "✓ 打包完成: $ARCHIVE_PATH ($(du -sh "$ARCHIVE_PATH" | cut -f1), ${archive_bytes} bytes)"
    echo "ARCHIVE_SIZE:${archive_bytes}"
}

# ========== 备份元数据（仅排障，不参与 Go 层解析）==========
create_metadata() {
    local meta_file="/tmp/hermes_backup_${TIMESTAMP}.info"
    local hermes_version=""
    command -v hermes >/dev/null 2>&1 && hermes_version="$(hermes --version 2>/dev/null | head -n1 || true)"
    cat > "$meta_file" << EOF
{
    "backup_time": "$(date -Iseconds)",
    "backup_type": "pre_reinstall",
    "agent_type": "hermes",
    "archive_path": "$ARCHIVE_PATH",
    "archive_name": "$ARCHIVE_NAME",
    "source_dir": "$HERMES_HOME",
    "user": "$(whoami)",
    "hostname": "$(hostname)",
    "hermes_version": "${hermes_version:-unknown}"
}
EOF
    echo "备份元数据: $meta_file"
}

# ========== 主流程 ==========
main() {
    cleanup_old_upgrade_snapshots
    stop_gateway
    create_archive
    preserve_local_snapshot
    create_metadata

    echo ""
    echo "${LOG_PREFIX} === 备份完成 ==="
    echo "压缩包: $ARCHIVE_PATH"
    echo ""

    echo "BACKUP_COMPLETED:$ARCHIVE_PATH" > /tmp/backup_status
    echo "BACKUP_DIR_PATH:$ARCHIVE_PATH"
}

main "$@"
exit 0