#!/usr/bin/env bash
#
# OpenClaw Recovery Script — 修复数据库损坏与配置清理
#
# 用于 OpenClaw 升级恢复流程中，自动诊断并修复常见问题：
#   - SQLite 数据库 "database disk image is malformed"
#   - 过时/缺失的 plugin 配置条目
#   - legacy config 兼容性 (plugins.allow / bundledDiscovery)
#
# 用法 (直接在 OpenClaw 服务器上执行):
#   sudo bash openclaw_recovery.sh           # 诊断 + 修复
#   sudo bash openclaw_recovery.sh --dry-run # 仅诊断
#   sudo bash openclaw_recovery.sh --no-doctor # 跳过 doctor --fix
#

set -euo pipefail

# 引入 SQLite 无损修复公共库（TAT 场景由 ExpandIncludes 内联；本地执行走下方 source 兜底）
# %INCLUDE% lib_sqlite_repair.sh
_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd 2>/dev/null || echo .)"
if [ -f "${_SCRIPT_DIR}/lib_sqlite_repair.sh" ]; then
    # shellcheck disable=SC1091
    . "${_SCRIPT_DIR}/lib_sqlite_repair.sh"
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

DRY_RUN=false
NO_DOCTOR=false
RESUME_MODE=false

for arg in "$@"; do
    case "$arg" in
        --dry-run)   DRY_RUN=true ;;
        --no-doctor) NO_DOCTOR=true ;;
        --resume)    RESUME_MODE=true ;;
        --help|-h)
            echo "用法: sudo bash openclaw_recovery.sh [--dry-run] [--no-doctor] [--resume]"
            echo "  --dry-run    仅诊断，不执行修复"
            echo "  --no-doctor  跳过 openclaw doctor --fix"
            echo "  --resume     升级恢复模式：修库 + doctor --fix --yes + 重启 gateway"
            exit 0
            ;;
    esac
done

# TAT 模板变量：resume=true 时启用升级恢复模式（由 Go 侧 reinstallAndRestore 在 malformed 时下发）
if [ "{{resume}}" = "true" ]; then
    RESUME_MODE=true
fi

# ── 检查 root 权限 ──────────────────────────────────────────

if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}请用 sudo 运行此脚本。${NC}"
    exit 1
fi

# ── 检查 sqlite3 依赖 ─────────────────────────────────────────
# DB 修复的四层策略（WAL checkpoint / .recover / .dump）全部依赖 sqlite3 命令。
# 缺失时诊断和修复都会失败，必须先确保安装。
if ! command -v sqlite3 &>/dev/null; then
    echo -e "${YELLOW}⚠ sqlite3 未安装，正在自动安装...${NC}"
    # 抑制 needrestart 自动重启服务（避免重启 tat_agent 导致 SSH/TAT 会话中断）
    export DEBIAN_FRONTEND=noninteractive
    if command -v apt-get &>/dev/null; then
        NEEDRESTART_MODE=l apt-get update -qq && NEEDRESTART_MODE=l apt-get install -y -qq --no-install-recommends sqlite3
    elif command -v yum &>/dev/null; then
        yum install -y sqlite
    elif command -v apk &>/dev/null; then
        apk add --no-cache sqlite
    else
        echo -e "${RED}✗ 无法自动安装 sqlite3，请手动安装后重试${NC}"
        echo -e "  apt-get install -y sqlite3  或  yum install -y sqlite"
        exit 1
    fi
    unset DEBIAN_FRONTEND
    if ! command -v sqlite3 &>/dev/null; then
        echo -e "${RED}✗ sqlite3 安装失败，请手动安装后重试${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ sqlite3 已安装${NC}"
fi

echo -e "${BLUE}OpenClaw Recovery Script${NC}"
echo "========================="
echo ""

# ── 运行用户 & gateway 控制 ─────────────────────────────────
# 运行用户由 Go 侧通过 OPENCLAW_RUNTIME_USER 环境变量注入；手动执行默认 root。
# openclaw-gateway 是 systemd --user 级 unit（$HOME/.config/systemd/user/），
# 因此 stop/restart 必须走 `systemctl --user`，不能用 system 级 systemctl，
# 否则找不到 unit（root 分支尤其）会 fallback 到裸 `openclaw gateway start`，
# 产生不受 systemd 托管的游离 gateway 进程。
RUNTIME_USER="${OPENCLAW_RUNTIME_USER:-root}"
RUNTIME_UID="$(id -u "$RUNTIME_USER" 2>/dev/null || echo 0)"

gateway_ctl() {
    # $1: stop | start | restart
    local action="$1"
    if [ "$RUNTIME_USER" = "root" ]; then
        XDG_RUNTIME_DIR="/run/user/${RUNTIME_UID}" systemctl --user "$action" openclaw-gateway 2>/dev/null
    elif command -v runuser >/dev/null 2>&1; then
        runuser -u "$RUNTIME_USER" -- env XDG_RUNTIME_DIR="/run/user/${RUNTIME_UID}" systemctl --user "$action" openclaw-gateway 2>/dev/null
    else
        su "$RUNTIME_USER" -c "XDG_RUNTIME_DIR=/run/user/${RUNTIME_UID} systemctl --user $action openclaw-gateway" 2>/dev/null
    fi
}

# ── 定位 OpenClaw ───────────────────────────────────────────
# 优先按运行用户家目录定位，避免多用户/多残留目录时 /home/* 遍历选错。
OC_DATA=""
_ru_home=""
if [ "$RUNTIME_USER" = "root" ]; then
    _ru_home="/root"
else
    [ -r /etc/passwd ] && _ru_home=$(awk -F: -v u="$RUNTIME_USER" '$1==u{print $6; exit}' /etc/passwd)
    [ -z "$_ru_home" ] && _ru_home="/home/$RUNTIME_USER"
fi
if [[ -d "$_ru_home/.openclaw" ]]; then
    OC_DATA="$_ru_home/.openclaw"
else
    for d in /root/.openclaw /home/*/.openclaw; do
        if [[ -d "$d" ]]; then
            OC_DATA="$d"
            break
        fi
    done
fi

if [[ -z "$OC_DATA" ]]; then
    echo -e "${RED}错误: 未找到 OpenClaw 数据目录 (.openclaw)${NC}"
    exit 1
fi

OC_CONFIG="$OC_DATA/openclaw.json"
OC_STATE="$OC_DATA/state"
OC_DB="$OC_STATE/openclaw.sqlite"

# 首选路径不存在时，按 SQLite 文件头 magic 自动发现真实数据库文件，
# 免疫文件名/后缀/子目录差异（.sqlite / .db / 无后缀均可命中），不改变既有流程。
if [[ ! -f "$OC_DB" ]]; then
    while IFS= read -r -d '' _f; do
        if head -c 16 "$_f" 2>/dev/null | grep -q 'SQLite format 3'; then
            OC_DB="$_f"
            OC_STATE="$(dirname "$_f")"
            break
        fi
    done < <(find "$OC_DATA" -type f \
                ! -name '*.bak' ! -name '*.corrupted.*' \
                ! -name '*-wal' ! -name '*-shm' -print0 2>/dev/null)
fi

# 找 openclaw 二进制
OC_BIN=""
if command -v openclaw &>/dev/null; then
    OC_BIN="$(command -v openclaw)"
elif [[ -f /root/.local/share/pnpm/openclaw ]]; then
    OC_BIN="/root/.local/share/pnpm/openclaw"
else
    # 尝试用 pnpm 全局路径
    for p in /root/.local/share/pnpm/openclaw /usr/local/bin/openclaw /usr/bin/openclaw; do
        if [[ -f "$p" ]]; then
            OC_BIN="$p"
            break
        fi
    done
fi

echo -e "binary : ${GREEN}${OC_BIN:-未找到(将尝试自动寻找)}${NC}"
echo -e "data   : ${GREEN}${OC_DATA}${NC}"
echo -e "config : ${GREEN}${OC_CONFIG}${NC}"
echo -e "db     : ${GREEN}${OC_DB}${NC}"
echo ""

# ── 诊断 ──────────────────────────────────────────────────────

echo -e "${BLUE}── 诊断 ──${NC}"

ISSUES=()

# 1. 检查数据库完整性
if [[ -f "$OC_DB" ]]; then
    INTEGRITY=$(sqlite3 "$OC_DB" "PRAGMA integrity_check;" 2>&1 || true)
    if [[ "$INTEGRITY" == "ok" ]]; then
        echo -e "  DB integrity: ${GREEN}✓ ok${NC}"
    else
        echo -e "  DB integrity: ${RED}✗ ${INTEGRITY}${NC}"
        ISSUES+=("db_corrupt")
    fi
else
    echo -e "  DB: ${YELLOW}文件不存在 (首次运行或已清理)${NC}"
fi

# 2. 检查配置文件
if [[ -f "$OC_CONFIG" ]]; then
    if ! python3 -c "import json; json.load(open('$OC_CONFIG'))" 2>/dev/null; then
        echo -e "  config: ${RED}✗ JSON 格式错误${NC}"
        ISSUES+=("config_invalid")
    else
        # 提取 plugins 配置
        PLUGIN_ALLOW=$(python3 -c "
import json
cfg = json.load(open('$OC_CONFIG'))
pl = cfg.get('plugins', {})
allow = pl.get('allow', [])
entries = pl.get('entries', {})
# enabled 但不在 allowlist 的插件
missing = [n for n in entries if n not in allow and entries[n].get('enabled', False)]
# disabled 的插件 (仅提示)
disabled = [n for n in entries if not entries[n].get('enabled', True)]
print('missing:', ' '.join(missing) if missing else '--')
print('disabled:', ' '.join(disabled) if disabled else '--')
" 2>/dev/null || true)

        MISSING=$(echo "$PLUGIN_ALLOW" | grep '^missing:' | sed 's/^missing: //')
        DISABLED=$(echo "$PLUGIN_ALLOW" | grep '^disabled:' | sed 's/^disabled: //')

        if [[ "$MISSING" != "--" && -n "$MISSING" ]]; then
            echo -e "  plugins not in allowlist: ${YELLOW}${MISSING}${NC}"
            ISSUES+=("plugin_allow_missing")
        fi
        if [[ "$DISABLED" != "--" && -n "$DISABLED" ]]; then
            echo -e "  disabled plugins (保留): ${YELLOW}${DISABLED}${NC}"
        fi
    fi
else
    echo -e "  config: ${RED}✗ 配置文件不存在${NC}"
    ISSUES+=("config_missing")
fi

# 3. WAL/SHM 残留检查
WAL_SHM=$(ls "${OC_DB}-wal" "${OC_DB}-shm" 2>/dev/null || true)
if [[ -n "$WAL_SHM" ]]; then
    echo -e "  WAL/SHM 残留文件存在"
fi

echo ""

if [[ ${#ISSUES[@]} -eq 0 ]]; then
    # resume 模式（升级 malformed 自愈）契约是"修库 + doctor --fix --yes + 重启 gateway"。
    # 即使本次复检未发现库/配置问题（可能 doctor 认定的库与本脚本发现的库不一致，
    # 或 WAL 已收敛），仍需继续执行 doctor + 重启 gateway，不能提前退出，
    # 否则后续续跑的 restore_post(resume_after_doctor) 若再次 malformed 将无第二次自愈。
    if $RESUME_MODE && ! $NO_DOCTOR && ! $DRY_RUN; then
        echo -e "${GREEN}✓ 未发现库/配置问题；resume 模式仍将执行 doctor --fix --yes + 重启 gateway${NC}"
    else
        echo -e "${GREEN}✓ 未发现问题，无需修复。${NC}"
        exit 0
    fi
fi

echo -e "发现 ${YELLOW}${#ISSUES[@]}${NC} 个问题: ${ISSUES[*]}"
echo ""

if $DRY_RUN; then
    echo -e "${YELLOW}[DRY RUN] 跳过修复步骤。${NC}"
    exit 0
fi

# ── 修复 ──────────────────────────────────────────────────────

echo -e "${BLUE}── 修复 ──${NC}"

TIMESTAMP=$(date +%s)
STEP=1
TOTAL_STEPS=0

# 计算总步数
# resume 模式下跳过配置清理（交给续跑的 restore_post step4 check_channel_plugins 统一处理），
# 故 plugin_allow_missing/config_invalid 不计入步数。config_missing 无对应修复分支，不计。
for issue in "${ISSUES[@]}"; do
    case "$issue" in
        db_corrupt) TOTAL_STEPS=$((TOTAL_STEPS + 1)) ;;
        plugin_allow_missing|config_invalid)
            $RESUME_MODE || TOTAL_STEPS=$((TOTAL_STEPS + 1)) ;;
    esac
done
if ! $NO_DOCTOR; then
    TOTAL_STEPS=$((TOTAL_STEPS + 1))
fi

# --- 修复 1: 数据库 ---

if [[ " ${ISSUES[*]} " =~ "db_corrupt" ]]; then
    echo ""
    echo -e "[${STEP}/${TOTAL_STEPS}] 修复数据库（无损恢复优先）..."

    # 停止可能占用数据库的 openclaw 进程（best-effort，避免边修边写又损坏）。
    # 先经 systemd --user 停服务（防止 Restart= 策略在 pkill 后立即把 gateway 拉起
    # 导致修库期间重新写库、二次损坏），再 pkill 兜底残留进程。
    gateway_ctl stop || true
    "${OC_BIN:-openclaw}" gateway stop >/dev/null 2>&1 || true
    pkill -f 'openclaw.*gateway' >/dev/null 2>&1 || true
    sleep 1

    # 清理残留锁文件（gateway 异常退出时 .locks/ 和 openclaw.lock 不会自动释放）
    rm -f "$OC_DATA"/.locks/*.lock 2>/dev/null || true
    rm -f "$OC_DATA"/openclaw.lock 2>/dev/null || true
    echo -e "  ${GREEN}✓ 已清理残留锁文件${NC}"

    # 备份损坏的数据库（保底留证，原始数据绝不直接丢弃）
    # 关键：-wal/-shm 也必须一起备份——WAL 里可能包含尚未 checkpoint 的最新事务数据，
    # 若只备份主库文件、后续兜底删库时把 -wal/-shm 直接删掉，这部分数据会永久丢失且无法补救。
    BAK_PATH="${OC_DB}.corrupted.${TIMESTAMP}.bak"
    cp -p "$OC_DB" "$BAK_PATH" 2>/dev/null || true
    [ -f "${OC_DB}-wal" ] && cp -p "${OC_DB}-wal" "${BAK_PATH}-wal" 2>/dev/null || true
    [ -f "${OC_DB}-shm" ] && cp -p "${OC_DB}-shm" "${BAK_PATH}-shm" 2>/dev/null || true
    echo -e "  ${GREEN}✓ 已备份坏库: ${BAK_PATH}（含 -wal/-shm 侧车文件，如存在）${NC}"

    RECOVERED=false

    # 步骤 ①②③：无损优先修复（WAL 收敛 → .recover → .dump），复用公共库 lib_sqlite_repair.sh，
    # 与备份端 backup_pre_reinstall.sh 共用同一份逻辑，避免两处修复策略漂移。
    # set -e 下用 || _repair_rc=$? 吞掉非零返回码，避免函数返回码直接触发脚本退出。
    _repair_rc=0
    sqlite_lossless_repair "$OC_DB" || _repair_rc=$?
    if [[ "$_repair_rc" -eq 0 ]]; then
        echo -e "  ${GREEN}✓ 数据库已无损修复（坏库副本保留于 ${BAK_PATH}）${NC}"
        RECOVERED=true
    elif [[ "$_repair_rc" -eq 2 ]]; then
        echo -e "  ${YELLOW}⚠ sqlite3 缺失无法修复（rc=2），保留原库不删，交下游处理${NC}"
        # 不设 RECOVERED=true，也不走兜底删库——sqlite3 缺失不代表库不可恢复
    else
        echo -e "  ${YELLOW}⚠ 无损/部分恢复均失败（rc=$_repair_rc），进入兜底重建${NC}"
    fi

    # 步骤 ④：兜底——确实救不回（rc=1），清除损坏库让 OpenClaw 重建空库（接受历史数据丢失，坏库副本已保留）
    # rc=2（sqlite3 缺失）时不删库，保留原库交下游处理
    if ! $RECOVERED && [[ "$_repair_rc" -ne 2 ]]; then
        rm -f "$OC_DB" "${OC_DB}-wal" "${OC_DB}-shm" 2>/dev/null || true
        echo -e "  ${RED}✗ 数据无法恢复（坏库副本保留于 ${BAK_PATH}）${NC}"
        echo -e "  ${GREEN}✓ 已清除损坏库，OpenClaw 将在下次启动时重建空库${NC}"
        # 输出机器可识别标记：Go 侧据此在升级完成通知中额外提示"历史数据未能恢复"
        echo "RECOVERY_DB_REBUILT_EMPTY (backup=${BAK_PATH})"
    fi

    STEP=$((STEP + 1))
fi

# --- 修复 2: 配置清理 ---

if { [[ " ${ISSUES[*]} " =~ "plugin_allow_missing" ]] || [[ " ${ISSUES[*]} " =~ "config_invalid" ]]; } && ! $RESUME_MODE; then
    echo ""
    echo -e "[${STEP}/${TOTAL_STEPS}] 清理配置文件..."

    # 备份原配置
    BAK_CFG="${OC_CONFIG}.bak.${TIMESTAMP}"
    cp "$OC_CONFIG" "$BAK_CFG"
    echo -e "  ${GREEN}✓ 配置已备份: ${BAK_CFG}${NC}"

    # 用 python3 修复 JSON 配置
    python3 -c "
import json, sys

with open('$OC_CONFIG') as f:
    cfg = json.load(f)

plugins = cfg.setdefault('plugins', {})
entries = plugins.get('entries', {})
allow = plugins.get('allow', [])

if not isinstance(allow, list):
    plugins['allow'] = []
    allow = plugins['allow']

# 添加所有 enabled 但不在 allowlist 的插件
for name in entries:
    if name not in allow and entries[name].get('enabled', False):
        allow.append(name)
        print(f'  + 添加 {name} 到 plugins.allow')

with open('$OC_CONFIG', 'w') as f:
    json.dump(cfg, f, indent=2, ensure_ascii=False)

print('  ✓ 配置文件已更新')
" 2>&1

    chmod 600 "$OC_CONFIG"

    STEP=$((STEP + 1))
fi

# --- 修复 3: openclaw doctor --fix ---

if ! $NO_DOCTOR; then
    echo ""
    echo -e "[${STEP}/${TOTAL_STEPS}] 运行 openclaw doctor --fix..."

    OC_CMD="${OC_BIN:-/root/.local/share/pnpm/openclaw}"

    # 尝试加载 nvm/pnpm 环境
    if [ -f /etc/profile.d/openclaw-env.sh ]; then
        source /etc/profile.d/openclaw-env.sh || true
    elif [ -f /root/.nvm/nvm.sh ]; then
        export NVM_DIR="/root/.nvm"
        source "$NVM_DIR/nvm.sh" 2>/dev/null || true
    fi

    # resume 模式用 --yes（非交互，TAT 环境）；手动模式保持 --fix
    if $RESUME_MODE; then
        DOCTOR_OUT=$("$OC_CMD" doctor --fix --yes 2>&1) || true
    else
        DOCTOR_OUT=$("$OC_CMD" doctor --fix 2>&1) || true
    fi

    if echo "$DOCTOR_OUT" | grep -q "database disk image is malformed"; then
        echo -e "  ${YELLOW}⚠ doctor --fix 仍报告数据库损坏，尝试无损修复...${NC}"
        # integrity_check 可能与 doctor 的检测不一致（integrity ok 但 doctor 仍报 malformed），
        # 此时之前诊断阶段未触发修库。这里兜底再试一次无损修复，修好后重跑 doctor。
        # 修库前先停 gateway（自动下发时 gateway 已停是 no-op；手动执行时 gateway 可能还在运行，
        # 不停会导致 mv 替换主库后 gateway 仍写旧 inode → 数据丢失/二次损坏）。
        gateway_ctl stop || true
        pkill -f 'openclaw.*gateway' >/dev/null 2>&1 || true
        sleep 1
        _repair_rc2=0
        sqlite_lossless_repair "$OC_DB" || _repair_rc2=$?
        if [ "$_repair_rc2" -eq 0 ]; then
            echo -e "  ${GREEN}✓ 无损修复成功，重新运行 doctor${NC}"
            DOCTOR_OUT=$("$OC_CMD" doctor --fix --yes 2>&1) || true
            if echo "$DOCTOR_OUT" | grep -q "database disk image is malformed"; then
                echo -e "  ${YELLOW}⚠ 修复后 doctor 仍报告数据库损坏${NC}"
            else
                echo -e "  ${GREEN}✓ 修复后 doctor 通过${NC}"
            fi
        else
            echo -e "  ${YELLOW}⚠ 无损修复失败（rc=$_repair_rc2），doctor 仍报告数据库损坏${NC}"
        fi
    elif echo "$DOCTOR_OUT" | grep -qi "Error"; then
        echo -e "  ${YELLOW}⚠ doctor --fix 可能有错误:${NC}"
        echo "$DOCTOR_OUT" | grep -i "Error\|error" | while read -r line; do
            echo -e "    ${line}"
        done
    else
        echo -e "  ${GREEN}✓ doctor --fix 执行完成${NC}"
    fi

    # 显示关键变化（grep 无匹配时返回 1，需 || true 避免 set -e 终止脚本）
    echo "$DOCTOR_OUT" | grep -E "Doctor changes|Set plugins|Added|Enabled|Installed" | while read -r line; do
        echo -e "    ${line}"
    done || true

    # doctor 后始终重启 gateway，让修好的库与配置生效
    # （无论 resume 还是手动模式，修完 DB + doctor 后都需要重启 gateway 才能加载新库）。
    # gateway 是 systemd --user 级 unit，统一走 gateway_ctl（systemctl --user）；
    # 仅在 systemd 路径彻底失败时才 fallback 到裸 openclaw gateway start。
    echo ""
    echo -e "  重启 gateway 加载修复后的库与配置..."
    gateway_ctl restart || "$OC_CMD" gateway start 2>/dev/null || true
    # 验证 gateway 是否在运行（避免重启失败被 || true 吞掉后误报成功）
    sleep 2
    if pgrep -f 'openclaw.*gateway' >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓ gateway 已重启${NC}"
    else
        echo -e "  ${YELLOW}⚠ gateway 重启后未检测到运行进程，续跑可能需手动重启${NC}"
    fi

    STEP=$((STEP + 1))
fi

# ── 验证 ──────────────────────────────────────────────────────

echo ""
echo -e "${BLUE}── 验证 ──${NC}"

# 1. 数据库
if [[ -f "$OC_DB" ]]; then
    INTEGRITY=$(sqlite3 "$OC_DB" "PRAGMA integrity_check;" 2>&1 || true)
    if [[ "$INTEGRITY" == "ok" ]]; then
        echo -e "  DB: ${GREEN}✓ ok${NC}"
    else
        echo -e "  DB: ${RED}✗ ${INTEGRITY}${NC}"
    fi
else
    echo -e "  DB: ${YELLOW}文件不存在 (OpenClaw 将在下次启动时重建)${NC}"
fi

# 2. 状态目录
if [[ -d "$OC_STATE" ]]; then
    echo ""
    echo "  状态目录:"
    ls -lh "$OC_DB"* 2>/dev/null | while read -r line; do
        echo "    $line"
    done || true
fi

echo ""
echo -e "${GREEN}── 完成 ──${NC}"
