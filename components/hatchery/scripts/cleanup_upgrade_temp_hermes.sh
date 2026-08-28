#!/bin/bash
# Hermes 升级完成后清理脚本。
# 由 Go 侧 runHermesUpgradePostHooks 在恢复+ready 后异步下发：
#   1. 清理 /tmp 下的临时压缩包和状态文件
#   2. 清理 ~/.hermes/upgrades/ 下的旧快照目录，仅保留最近 3 个
#   3. 清理恢复失败或旧版本遗留的 ~/.hermes.fresh.<timestamp> 临时目录
# 幂等、容错，任何子步骤失败都不影响其他步骤，始终 exit 0。

set -u

# 日志写入 ~/.hermes/logs/
LOG_DIR="${HOME}/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
SCRIPT_NAME="cleanup_upgrade_temp_hermes"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
if [ -w "$LOG_DIR" ] || [ -w "$LOG_FILE" ] 2>/dev/null; then
    exec > >(tee -a "$LOG_FILE") 2>&1
fi
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

LOG_PREFIX="[cleanup-upgrade-temp-hermes]"
KEEP=3
UPGRADES_DIR="${HOME}/.hermes/upgrades"

# 清理 /tmp 下临时压缩包和状态文件
cleanup_tmp_archives() {
    echo "${LOG_PREFIX} [step 1/2] 清理 /tmp 下的临时备份压缩包与状态文件"
    local removed=0
    # shellcheck disable=SC2206
    local patterns=(
        /tmp/hermes-restore-*.tgz
        /tmp/hermes-state-*.tgz
        /tmp/hermes_backup_*.info
        /tmp/backup_status
        /tmp/restore_status
        /tmp/backup_precheck_status
    )
    local f
    for f in "${patterns[@]}"; do
        [ -e "$f" ] || continue
        if rm -f -- "$f" 2>/dev/null; then
            removed=$((removed + 1))
            echo "${LOG_PREFIX}   ✓ 已删除: $f"
        else
            echo "${LOG_PREFIX}   ⚠ 删除失败（忽略）: $f"
        fi
    done
    [ "$removed" -eq 0 ] && echo "${LOG_PREFIX}   /tmp 下无残留临时文件" \
        || echo "${LOG_PREFIX}   ✓ 共清理 $removed 个 /tmp 临时文件"
}

# 仅保留 upgrades/ 下最近 KEEP 个子目录（YYYYMMDD_HHMMSS 字典序==时间序）
cleanup_old_upgrade_snapshots() {
    echo "${LOG_PREFIX} [step 2/2] 清理 $UPGRADES_DIR 下的旧快照（保留最近 $KEEP 个）"

    if [ ! -d "$UPGRADES_DIR" ]; then
        echo "${LOG_PREFIX}   目录不存在，跳过"
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

# 仅清理超过一天的同级临时目录，避免与正在执行的恢复脚本争用目录。
# 名称必须严格以 .hermes.fresh.<数字时间戳> 开头，防止误删用户其他目录。
cleanup_stale_fresh_homes() {
    echo "${LOG_PREFIX} [step 3/3] 清理超过一天的 Hermes 新镜像临时目录"

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
    done < <(find "$HOME" -mindepth 1 -maxdepth 1 -type d -name '.hermes.fresh.[0-9]*' -mtime +1 2>/dev/null)

    [ "$removed" -eq 0 ] && echo "${LOG_PREFIX}   无过期新镜像临时目录" \
        || echo "${LOG_PREFIX}   ✓ 共清理 $removed 个过期新镜像临时目录"
}

main() {
    cleanup_tmp_archives || true
    cleanup_old_upgrade_snapshots || true
    cleanup_stale_fresh_homes || true
    echo "${LOG_PREFIX} 全部清理完成"
}

main "$@"
exit 0
