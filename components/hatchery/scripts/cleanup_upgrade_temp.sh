#!/bin/bash
# =============================================================================
# OpenClaw 升级完成后清理脚本
# -----------------------------------------------------------------------------
# 由 Go 侧 reinstallAndRestore 在数据恢复成功后异步下发执行，专职处理三件事：
#
#   1. 清理升级过程中产生的临时压缩包（防止占用机器存储）：
#      - ${HOME}/openclaw-state-*.tgz （backup_pre_reinstall 备份压缩包）
#      - ${HOME}/openclaw-backup.tgz    （早期 restore 临时位置）
#      - ${HOME}/openclaw_backup_*.info、/tmp/backup_status（备份状态文件）
#      - /tmp/openclaw-state-*.tgz      （早期 backup_pre_reinstall 残留，兼容历史）
#      - /tmp/openclaw-backup.tgz       （早期 restore 临时位置，兼容历史）
#      - /tmp/openclaw_backup_*.info    （早期备份状态文件，兼容历史）
#
#   2. 清理 ~/.openclaw/upgrades/ 下的旧快照目录，仅保留最近 3 个：
#      restore_post_reinstall.sh 的 download_from_smh 步骤会把恢复包写到
#      ~/.openclaw/upgrades/<YYYYMMDD_HHMMSS>/openclaw-backup.tgz 留存，
#      历次升级累积会让该目录持续膨胀，这里按字典序倒序仅保留最近 3 个。
#
#   3. 升级后对 ~/.openclaw/openclaw.json 做幂等字段适配：
#      - agents.defaults.mediaMaxMb = 200
#      新版本要求该字段的最小值为 200，旧配置文件升级后需要补齐/拉高，
#      已为 200 或更大时也保持幂等覆盖为 200（按当前需求强制为该值）。
#
# 设计原则：
#   - 幂等：重复执行结果一致
#   - 容错：任何子步骤失败都不影响其他步骤，最终始终以 0 退出
#           （清理失败也不能阻断升级主流程，Go 侧已按 warn 处理）
# -----------------------------------------------------------------------------
# 拆分原因：
#   restore_post_reinstall.sh 体积较大，已接近 TAT 命令下发体积上限，
#   不再追加新逻辑，独立脚本由 Go 侧单独通过 RunScript 下发。
# =============================================================================

set -u

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR" 2>/dev/null || true
fi
SCRIPT_NAME="cleanup_upgrade_temp"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
if [ -w "$LOG_DIR" ] || [ -w "$LOG_FILE" ] 2>/dev/null; then
    exec > >(tee -a "$LOG_FILE") 2>&1
fi
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

LOG_PREFIX="[cleanup-upgrade-temp]"
KEEP=3
UPGRADES_DIR="${HOME}/.openclaw/upgrades"

# -----------------------------------------------------------------------------
# 步骤 1：清理 /tmp 下的临时压缩包
# -----------------------------------------------------------------------------
cleanup_tmp_archives() {
    echo "${LOG_PREFIX} [step 1/3] 清理 HOME 及 /tmp 下的临时备份压缩包"
    local removed=0
    # shellcheck disable=SC2206
    local patterns=(
        ${HOME}/openclaw-state-*.tgz
        ${HOME}/openclaw-backup.tgz
        ${HOME}/openclaw_backup_*.info
        /tmp/openclaw-state-*.tgz
        /tmp/openclaw-backup.tgz
        /tmp/openclaw_backup_*.info
        /tmp/backup_status
    )
    local f
    for f in "${patterns[@]}"; do
        # glob 未匹配时 patterns 中会保留字面量，需要 -e 测试存在性过滤掉
        if [ -e "$f" ]; then
            if rm -f -- "$f" 2>/dev/null; then
                removed=$((removed + 1))
                echo "${LOG_PREFIX}   ✓ 已删除: $f"
            else
                echo "${LOG_PREFIX}   ⚠ 删除失败（忽略）: $f"
            fi
        fi
    done
    if [ "$removed" -eq 0 ]; then
        echo "${LOG_PREFIX}   无残留临时备份文件"
    else
        echo "${LOG_PREFIX}   ✓ 共清理 $removed 个临时文件"
    fi
}

# -----------------------------------------------------------------------------
# 步骤 2：仅保留 ~/.openclaw/upgrades/ 下最近 KEEP 个子目录
# -----------------------------------------------------------------------------
cleanup_old_upgrade_snapshots() {
    echo "${LOG_PREFIX} [step 2/3] 清理 $UPGRADES_DIR 下的旧快照（保留最近 $KEEP 个）"

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

    # 子目录按 download_from_smh 命名为 YYYYMMDD_HHMMSS，字典序 == 时间序，
    # 倒序后跳过前 KEEP 个，剩下的就是较旧的需要删除的目录。
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

# -----------------------------------------------------------------------------
# 步骤 3：升级适配 — 强制 ~/.openclaw/openclaw.json 中
#         agents.defaults.mediaMaxMb = 200
# -----------------------------------------------------------------------------
# 背景：
#   新版本要求 agents.defaults.mediaMaxMb 最小为 200，旧配置中可能为更小值
#   或缺失该字段。这里在升级完成后做一次幂等修正，确保升级后配置满足新版要求。
#
# 容错策略（与本脚本整体「失败不阻断升级主流程」原则一致）：
#   - 配置文件不存在：跳过（新装机器不在此处生成残缺配置）
#   - jq 不存在：    跳过并 warn（不视为升级失败）
#   - jq 处理失败 / 输出为空：跳过并 warn，绝不用空文件覆盖原配置
#   - 临时文件加 $$ 后缀，避免并发执行互相踩踏
# -----------------------------------------------------------------------------
patch_openclaw_config() {
    local cfg="${HOME}/.openclaw/openclaw.json"
    echo "${LOG_PREFIX} [step 3/3] 适配 openclaw.json: agents.defaults.mediaMaxMb=200"

    if [ ! -f "$cfg" ]; then
        echo "${LOG_PREFIX}   配置文件不存在，跳过：$cfg"
        return 0
    fi
    if ! command -v jq >/dev/null 2>&1; then
        echo "${LOG_PREFIX}   ⚠ 未安装 jq，跳过适配（忽略）"
        return 0
    fi

    local tmp="/tmp/oc.tmp.$$"
    if jq '.agents.defaults.mediaMaxMb = 200' "$cfg" > "$tmp" 2>/dev/null && [ -s "$tmp" ]; then
        if mv "$tmp" "$cfg" 2>/dev/null; then
            echo "${LOG_PREFIX}   ✓ 已写入 mediaMaxMb=200: $cfg"
        else
            rm -f "$tmp" 2>/dev/null || true
            echo "${LOG_PREFIX}   ⚠ 替换失败（忽略）: $cfg"
        fi
    else
        rm -f "$tmp" 2>/dev/null || true
        echo "${LOG_PREFIX}   ⚠ jq 处理失败或输出为空（忽略），原配置保持不变"
    fi
}

main() {
    cleanup_tmp_archives || true
    cleanup_old_upgrade_snapshots || true
    patch_openclaw_config || true
    echo "${LOG_PREFIX} 全部清理完成"
}

main "$@"
exit 0
