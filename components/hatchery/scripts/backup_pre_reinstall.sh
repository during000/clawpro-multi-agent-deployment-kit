#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"

# ========== 日志设置 ==========
# 与 restore_post_reinstall.sh / sync_gateway_port.sh 等脚本保持一致：
# 日志落到 ~/.openclaw/logs/ 下，方便在实例本地排查升级前备份问题。
SCRIPT_NAME="backup_pre_reinstall"
LOG_DIR="${HOME}/.openclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || LOG_DIR="/tmp"
chmod 700 "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
touch "$LOG_FILE" 2>/dev/null || true
chmod 600 "$LOG_FILE" 2>/dev/null || true
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== $(date '+%Y-%m-%d %H:%M:%S') backup_pre_reinstall 开始 =========="

# 引入 SQLite 无损修复公共库（TAT 场景由 ExpandIncludes 内联；本地执行走下方 source 兜底）
# %INCLUDE% lib_sqlite_repair.sh
_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd 2>/dev/null || echo .)"
if [ -f "${_SCRIPT_DIR}/lib_sqlite_repair.sh" ]; then
    # shellcheck disable=SC1091
    . "${_SCRIPT_DIR}/lib_sqlite_repair.sh"
fi

# ========== OpenClaw 重装前数据备份脚本 ==========
# 功能：备份 ~/.openclaw/ 状态目录，打包为压缩包，输出包路径和大小供调用方上传到 SMH

# ========== 路径配置 ==========
OPENCLAW_HOME="${OPENCLAW_STATE_DIR:-$HOME/.openclaw}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
ARCHIVE_NAME="openclaw-state-${TIMESTAMP}.tgz"
ARCHIVE_PATH="${HOME}/${ARCHIVE_NAME}"

# ========== 解析参数 ==========
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            echo "Usage: $0"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "=== OpenClaw 重装前数据备份 ==="
echo "状态目录: $OPENCLAW_HOME"
echo "压缩包路径: $ARCHIVE_PATH"

# ========== 第一步：停止 Gateway，避免文件变化 ==========
_GATEWAY_STOPPED=0

_restart_gateway_on_error() {
    if [[ "$_GATEWAY_STOPPED" -eq 1 ]]; then
        echo "⚠ 脚本异常，尝试重新拉起 OpenClaw Gateway..."
        openclaw gateway start 2>/dev/null || true
    fi
}

stop_gateway() {
    echo "停止 OpenClaw Gateway..."
    if command -v openclaw &>/dev/null; then
        openclaw gateway stop 2>/dev/null || true
        _GATEWAY_STOPPED=1
        trap '_restart_gateway_on_error' ERR
        sleep 2
    else
        echo "⚠ openclaw 命令不存在，跳过停止 Gateway"
    fi
}

# ========== 第一步补充：SQLite 无损修复 + 门禁 ==========
# 停 Gateway、确认 openclaw 进程退出后，对本地 SQLite 库做"无损优先修复"（见 lib_sqlite_repair.sh）：
#   - integrity ok / WAL 收敛 / .recover / .dump 成功 → 打包的是好库，从源头规避恢复端
#     doctor 的 "database disk image is malformed"，实现零数据丢失，且无需重装后 recovery。
#   - 真不可无损修复 → 输出 BACKUP_DB_UNRECOVERABLE 并以非零退出：
#     ★重装(ResetInstance)是不可逆的数据销毁点，任何抢救都必须在重装前完成★。
#     此时中止升级、保留原盘现场（原实例仍可用），交人工离线抢救，
#     绝不把坏库打包上传→重装→只能删库重建导致数据永久丢失。
# sqlite3 缺失时跳过修复（恢复端 openclaw_recovery.sh 兜底），不阻断升级。
ensure_healthy_sqlite() {
    echo "检查并无损修复本地 SQLite 以保证备份一致性..."

    # 等 openclaw 业务进程真正退出（最多 10s），而非仅 sleep 2
    local i
    for i in $(seq 1 10); do
        if ! pgrep -u "$(id -u)" -f 'openclaw.*gateway|\.openclaw/.*\.(c?js|mjs)' >/dev/null 2>&1; then
            break
        fi
        sleep 1
    done

    if ! command -v sqlite3 >/dev/null 2>&1; then
        echo "⚠ 未安装 sqlite3，跳过修复（恢复端 openclaw_recovery.sh 会兜底）"
        return 0
    fi

    local db found=0 unrecoverable=0 rc
    while IFS= read -r -d '' db; do
        head -c 16 "$db" 2>/dev/null | grep -q 'SQLite format 3' || continue
        found=1
        echo "  修复: $db"
        # 备份原库（保底留证，绝不丢弃原始数据）
        cp -p "$db" "${db}.prebackup.$(date +%s).bak" 2>/dev/null || true
        # set -e 下用 || rc=$? 吞掉非零返回，避免函数返回码直接触发脚本退出
        rc=0
        sqlite_lossless_repair "$db" || rc=$?
        if [ "$rc" -eq 1 ]; then
            echo "  ✗ 数据库无法无损修复: $db"
            unrecoverable=1
        fi
    done < <(find "$OPENCLAW_HOME" -type f \
                \( -name '*.sqlite' -o -name '*.sqlite3' -o -name '*.db' \) \
                ! -name '*.bak' ! -name '*.corrupted.*' ! -name '*.prebackup.*' -print0 2>/dev/null)

    if [ "$found" -eq 0 ]; then
        echo "  未发现 SQLite 文件，跳过"
        return 0
    fi

    if [ "$unrecoverable" -eq 1 ]; then
        # 门禁：存在不可无损修复的库 → 中止升级以保护数据。
        # 前面已 stop_gateway，这里把原实例的 gateway 拉起来恢复服务。
        echo "重新拉起 OpenClaw Gateway（升级已中止，原实例继续提供服务）..."
        openclaw gateway start 2>/dev/null || true
        echo "BACKUP_DB_UNRECOVERABLE"
        echo "✗ 检测到本地数据库损坏且无法无损修复，已中止升级以保护数据（原实例保持可用）。"
        exit 1
    fi

    echo "✓ SQLite 完整性检查/无损修复通过"
    return 0
}

# ========== 第二步：打包整个 openclaw 目录 ==========
# 打包 OPENCLAW_HOME 下所有文件，排除重装后自带的 node_modules 等程序文件
# 恢复时解压到新机器的 OPENCLAW_HOME 下即可完成覆盖
create_archive() {
    echo "开始打包 OpenClaw 目录..."

    if [ ! -d "$OPENCLAW_HOME" ]; then
        echo "✗ OpenClaw 状态目录不存在: $OPENCLAW_HOME"
        exit 1
    fi

    echo "打包目录: $OPENCLAW_HOME"

    # 以 OPENCLAW_HOME 为基准打包全部内容
    # 排除 node_modules、npm 缓存等重装后自带的程序文件，避免包体积过大
    # 排除 upgrades/ 快照目录（由恢复脚本写入，记录历次升级内容，无需重复备份）
    # todo 后续需要忽略 upgrades 文件夹 当前先保留用于临时观察
    tar -czf "$ARCHIVE_PATH" \
        -C "$OPENCLAW_HOME" \
        --exclude="node_modules" \
        --exclude=".npm" \
        --exclude="npm-cache" \
        --exclude=".cache" \
        --exclude="*.log" \
        --exclude="tmp" \
        --exclude="*.sqlite-wal" \
        --exclude="*.sqlite-shm" \
        --exclude="*.db-wal" \
        --exclude="*.db-shm" \
        .

    local archive_bytes
    archive_bytes=$(stat -c%s "$ARCHIVE_PATH" 2>/dev/null || stat -f%z "$ARCHIVE_PATH" 2>/dev/null || du -b "$ARCHIVE_PATH" | cut -f1)
    echo "✓ 打包完成: $ARCHIVE_PATH ($(du -sh "$ARCHIVE_PATH" | cut -f1), ${archive_bytes} bytes)"
    # 输出包大小（字节），供调用方检查 SMH 剩余空间
    echo "ARCHIVE_SIZE:${archive_bytes}"
}

# ========== 清理函数：删除本地压缩包 ==========
cleanup_on_failure() {
    echo "清理本地备份文件..."
    rm -f "$ARCHIVE_PATH" && echo "✓ 已删除压缩包: $ARCHIVE_PATH" || true
}

# ========== 第四步：写入备份元数据 ==========
create_metadata() {
    local meta_file="${HOME}/openclaw_backup_${TIMESTAMP}.info"
    cat > "$meta_file" << EOF
{
    "backup_time": "$(date -Iseconds)",
    "backup_type": "pre_reinstall",
    "archive_path": "$ARCHIVE_PATH",
    "archive_name": "$ARCHIVE_NAME",
    "source_dir": "$OPENCLAW_HOME",
    "user": "$(whoami)",
    "hostname": "$(hostname)",
    "openclaw_version": "$(openclaw --version 2>/dev/null | head -n1 || echo 'unknown')"
}
EOF
    echo "备份元数据: $meta_file"
}

# ========== 检查 openclaw.json 中 models.providers key 合法性 ==========
# 若 key 中包含非法字符（如 "/"），备份前直接报错，阻止后续升级流程
check_provider_keys() {
    local config_file="$OPENCLAW_HOME/openclaw.json"
    if [ ! -f "$config_file" ]; then
        echo "openclaw.json 不存在，跳过 provider key 检查"
        return 0
    fi

    # 提取 models.providers 下所有 key
    local invalid_keys
    invalid_keys=$(python3 - "$config_file" <<'PYEOF'
import sys, json
try:
    with open(sys.argv[1]) as f:
        cfg = json.load(f)
    providers = cfg.get("models", {}).get("providers", {})
    bad = [k for k in providers if "/" in k]
    if bad:
        print("\n".join(bad))
except Exception:
    pass  # 解析失败不阻断
PYEOF
)

    if [ -n "$invalid_keys" ]; then
        echo "✗ openclaw.json 中 models.providers 存在非法 key（不能包含 /）："
        echo "$invalid_keys"
        echo "请先修复配置后再升级。"
        exit 1
    fi

    echo "✓ models.providers key 检查通过"
}

# ========== 主执行逻辑 ==========
main() {
    check_provider_keys
    stop_gateway
    ensure_healthy_sqlite
    create_archive
    create_metadata

    echo ""
    echo "=== 备份完成 ==="
    echo "压缩包: $ARCHIVE_PATH"
    echo ""

    # 写入状态文件
    echo "BACKUP_COMPLETED:$ARCHIVE_PATH" > /tmp/backup_status

    # 输出压缩包路径（供 Go 代码读取并上传到 SMH）
    echo "BACKUP_DIR_PATH:$ARCHIVE_PATH"
}

main "$@"
exit 0
