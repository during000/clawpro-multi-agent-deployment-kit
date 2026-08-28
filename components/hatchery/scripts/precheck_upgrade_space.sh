#!/bin/bash
# OpenClaw 升级前存储空间预探测（只读，exit 恒为 0）。
# 输出 KV：
#   PRECHECK_SOURCE_KB / ESTIMATED_KB / REQUIRED_KB / HOME_AVAIL_KB / HOME_FS / RESULT / REASON
# exclude 规则与 backup_pre_reinstall.sh 一致。
set -u

export PATH="$HOME/.npm-global/bin:$PATH"

# ========== 日志设置 ==========
# 与 backup_pre_reinstall.sh / sync_gateway_port.sh 等脚本保持一致：
# 日志落到 ~/.openclaw/logs/ 下，方便在实例本地排查升级前空间预探测问题。
SCRIPT_NAME="precheck_upgrade_space"
LOG_DIR="${HOME}/.openclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || LOG_DIR="/tmp"
chmod 700 "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
touch "$LOG_FILE" 2>/dev/null || true
chmod 600 "$LOG_FILE" 2>/dev/null || true
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== $(date '+%Y-%m-%d %H:%M:%S') precheck_upgrade_space 开始 =========="

OPENCLAW_HOME="${OPENCLAW_STATE_DIR:-$HOME/.openclaw}"
CHECK_DIR="${HOME}"

# 压缩比经验值：openclaw 状态目录以 JSON/SQLite/文本日志为主，gzip 后一般 20~40%。
# 保守取 40%，避免低估。
COMPRESS_RATIO_PCT=40
# 生成过程冗余系数：150% = 压缩包 + 生成中的临时占用 + 用户新写日志的余量。
REQUIRED_MULTIPLIER_PCT=150

LOG_PREFIX="[precheck-upgrade-space]"

echo "${LOG_PREFIX} === OpenClaw 升级前存储空间预探测 ==="
echo "${LOG_PREFIX} 状态目录: ${OPENCLAW_HOME}"
echo "${LOG_PREFIX} 备份存放目录: ${CHECK_DIR}"

# ---------- 1) 状态目录大小估算（应用与 backup 脚本一致的 exclude 规则） ----------
if [ ! -d "${OPENCLAW_HOME}" ]; then
    echo "${LOG_PREFIX} ⚠ 状态目录不存在: ${OPENCLAW_HOME}，无法预估，降级放行"
    echo "PRECHECK_SOURCE_KB:0"
    echo "PRECHECK_ESTIMATED_KB:0"
    echo "PRECHECK_REQUIRED_KB:0"
    echo "PRECHECK_HOME_AVAIL_KB:0"
    echo "PRECHECK_HOME_FS:unknown"
    echo "PRECHECK_RESULT:unknown"
    echo "PRECHECK_REASON:openclaw_home_not_exist"
    exit 0
fi

# 用 du -sk + --exclude 估算目录大小（KB）。
# GNU du 支持 --exclude；BusyBox du 不支持，走 find 兜底排除已知大目录。
DU_METHOD="gnu"
SOURCE_KB=$(du -sk \
    --exclude='node_modules' \
    --exclude='.npm' \
    --exclude='npm-cache' \
    --exclude='.cache' \
    --exclude='*.log' \
    --exclude='tmp' \
    "${OPENCLAW_HOME}" 2>/dev/null | awk '{print $1}')
if [ -z "${SOURCE_KB:-}" ] || [ "${SOURCE_KB}" -eq 0 ] 2>/dev/null; then
    # BusyBox 兜底：用 find 排除已知大目录，避免把 node_modules 误算入导致拒绝。
    # 实际备份（backup_pre_reinstall.sh）也 exclude 这些目录，预探测应与之一致。
    DU_METHOD="busybox-find"
    SOURCE_KB=$(find "${OPENCLAW_HOME}" -mindepth 1 -maxdepth 1 \
        ! -name 'node_modules' \
        ! -name '.npm' \
        ! -name 'npm-cache' \
        ! -name '.cache' \
        ! -name 'tmp' \
        -exec du -sk {} + 2>/dev/null | awk '{sum+=$1} END {print sum+0}')
fi
if [ -z "${SOURCE_KB:-}" ]; then
    SOURCE_KB=0
fi
echo "${LOG_PREFIX} [step 1/4] 估算状态目录大小（排除 node_modules 等程序文件）"
echo "${LOG_PREFIX} [step 1/4] ✓ 目录大小估算完成: method=${DU_METHOD} source_kb=${SOURCE_KB}"

# ---------- 2) 估算压缩后大小 & 需要的总空间 ----------
echo "${LOG_PREFIX} [step 2/4] 计算压缩估及冗余需求"
ESTIMATED_KB=$(( SOURCE_KB * COMPRESS_RATIO_PCT / 100 ))
REQUIRED_KB=$(( ESTIMATED_KB * REQUIRED_MULTIPLIER_PCT / 100 ))
# 极小实例兜底：至少要求 10 MB，避免 SOURCE_KB 极小时 REQUIRED_KB 判 0
if [ "${REQUIRED_KB}" -lt 10240 ]; then
    REQUIRED_KB=10240
fi
echo "${LOG_PREFIX} [step 2/4] ✓ 压缩及冗余估算完成: compress_ratio=${COMPRESS_RATIO_PCT}% required_multiplier=${REQUIRED_MULTIPLIER_PCT}% estimated_kb=${ESTIMATED_KB} required_kb=${REQUIRED_KB}"

# ---------- 3) HOME 剩余空间 & 文件系统类型 ----------
echo "${LOG_PREFIX} [step 3/4] 查询 HOME 目录剩余空间及文件系统类型"
# df -Pk 输出 POSIX 格式，第 4 列为 Available（KB）
HOME_AVAIL_KB=$(df -Pk "${CHECK_DIR}" 2>/dev/null | awk 'NR==2 {print $4}')
if [ -z "${HOME_AVAIL_KB:-}" ]; then
    HOME_AVAIL_KB=0
fi

# 文件系统类型（ext4/xfs 等，HOME 目录一般位于持久化磁盘）
HOME_FS=$(df -PT "${CHECK_DIR}" 2>/dev/null | awk 'NR==2 {print $2}')
if [ -z "${HOME_FS:-}" ]; then
    HOME_FS="unknown"
fi
echo "${LOG_PREFIX} [step 3/4] ✓ HOME 空间查询完成: home_avail_kb=${HOME_AVAIL_KB} home_fs=${HOME_FS}"

# ---------- 4) 判定 ----------
echo "${LOG_PREFIX} [step 4/4] 执行空间充足性判定"
RESULT="ok"
REASON=""
if [ "${HOME_AVAIL_KB}" -lt "${REQUIRED_KB}" ]; then
    RESULT="insufficient"
    REASON="home_avail_lt_required"
fi
echo "${LOG_PREFIX} [step 4/4] ✓ 空间判定完成: result=${RESULT} reason=${REASON}"

# ---------- 5) 输出结构化 KV ----------
echo "PRECHECK_SOURCE_KB:${SOURCE_KB}"
echo "PRECHECK_ESTIMATED_KB:${ESTIMATED_KB}"
echo "PRECHECK_REQUIRED_KB:${REQUIRED_KB}"
echo "PRECHECK_HOME_AVAIL_KB:${HOME_AVAIL_KB}"
echo "PRECHECK_HOME_FS:${HOME_FS}"
echo "PRECHECK_RESULT:${RESULT}"
echo "PRECHECK_REASON:${REASON}"

echo "${LOG_PREFIX} === 探测完成，全部步骤通过 ==="
exit 0
