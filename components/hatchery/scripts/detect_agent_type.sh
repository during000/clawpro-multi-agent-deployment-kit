#!/bin/bash
# detect_agent_type.sh
# 检测当前实例上安装/运行的 agent 类型，供调用方决定后续使用哪个 get_version_info_*.sh。
#
# 输出契约：
#   - stdout 最后一行只打印一个 token，其值为以下之一：
#       openclaw       -- 当前机器为 openclaw agent
#       lightclawace   -- 当前机器为 LightClaw ACE agent
#       hermes         -- 当前机器为 Hermes agent
#       unknown        -- 以上均未检测到
#   - 所有诊断信息写入 ~/.hatchery/logs/detect_agent_type.log，不污染 stdout
#
# 判据（与各 get_version_info_*.sh 的实际依赖保持一致）：
#   hermes       : `hermes` 或 `harness` 命令可用 / $HOME/.hermes/ 目录存在
#   lightclawace : `lightclaw` 命令可用 / $HOME/.lightclaw/ 目录存在
#   openclaw     : `openclaw` 命令可用 / $HOME/.openclaw/openclaw.json 存在
#
# 注意：
#   1) 脚本经常以 root 被 TAT 下发执行，不能只看当前 $HOME；
#      需要同时扫描 /home/* 与 /root，避免把 agentuser 下的 hermes/ace 漏判为 openclaw。
#   2) openclaw 对“仅目录存在”不做强判定（历史残留目录较常见，易误判）。
#
# 同机并存多类 agent 时，按以下优先级返回：
#   hermes > lightclawace > openclaw
# 这一顺序与"新 agent 优先"的演进方向一致，可在后续需要时调整。

set -uo pipefail
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.lightclaw/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.hatchery/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
SCRIPT_NAME="detect_agent_type"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# stderr 也重定向到日志，保持 stdout 干净（最后一行必须是 token）
exec 2>>"$LOG_FILE"
echo "" >>"$LOG_FILE"
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') ==========" >>"$LOG_FILE"
echo "当前用户: $(id -un 2>/dev/null || echo unknown) ($(id -u 2>/dev/null || echo -1))" >>"$LOG_FILE"
echo "HOME: $HOME" >>"$LOG_FILE"

# ========== 通用探测辅助 ==========
# 枚举候选 home 目录：当前 HOME、/root、/home/*
iter_homes() {
    if [ -n "${HOME:-}" ] && [ -d "$HOME" ]; then
        echo "$HOME"
    fi
    if [ -d "/root" ] && [ "${HOME:-}" != "/root" ]; then
        echo "/root"
    fi
    for d in /home/*; do
        [ -d "$d" ] || continue
        [ "$d" = "${HOME:-}" ] && continue
        [ "$d" = "/root" ] && continue
        echo "$d"
    done
}

exists_exec_in_homes() {
    # $1: 相对路径，例如 ".local/bin/hermes"
    local rel="$1"
    local h p
    while IFS= read -r h; do
        [ -n "$h" ] || continue
        p="$h/$rel"
        if [ -x "$p" ]; then
            echo "  命中可执行: $p" >>"$LOG_FILE"
            return 0
        fi
    done < <(iter_homes)
    return 1
}

exists_dir_in_homes() {
    # $1: 相对路径，例如 ".hermes"
    local rel="$1"
    local h p
    while IFS= read -r h; do
        [ -n "$h" ] || continue
        p="$h/$rel"
        if [ -d "$p" ]; then
            echo "  命中目录: $p" >>"$LOG_FILE"
            return 0
        fi
    done < <(iter_homes)
    return 1
}

exists_file_in_homes() {
    # $1: 相对路径，例如 ".openclaw/openclaw.json"
    local rel="$1"
    local h p
    while IFS= read -r h; do
        [ -n "$h" ] || continue
        p="$h/$rel"
        if [ -f "$p" ]; then
            echo "  命中文件: $p" >>"$LOG_FILE"
            return 0
        fi
    done < <(iter_homes)
    return 1
}

# ========== 检测函数 ==========
# 注意：下列函数对"命令存在"与"目录/文件存在"同时考察，仅要求其一成立即视为安装。
#       这样即使 agent 被停用或 PATH 暂时丢失，只要痕迹仍在也能识别。

has_hermes() {
    # 命令层：hermes 或 harness 任一可用
    if command -v hermes >/dev/null 2>&1; then
        echo "  hermes: command -v hermes -> $(command -v hermes)" >>"$LOG_FILE"
        return 0
    fi
    if command -v harness >/dev/null 2>&1; then
        echo "  hermes: command -v harness -> $(command -v harness)" >>"$LOG_FILE"
        return 0
    fi
    # 路径 fallback（与 get_version_info_hermes.sh::locate_hermes_bin 一致）
    for p in "$HOME/.local/bin/hermes" "/usr/local/bin/hermes" "/root/.local/bin/hermes"; do
        if [ -x "$p" ]; then
            echo "  hermes: 找到可执行 $p" >>"$LOG_FILE"
            return 0
        fi
    done
    if exists_exec_in_homes ".local/bin/hermes"; then
        echo "  hermes: 命中 home 内 hermes 可执行" >>"$LOG_FILE"
        return 0
    fi
    if exists_exec_in_homes ".local/bin/harness"; then
        echo "  hermes: 命中 home 内 harness 可执行" >>"$LOG_FILE"
        return 0
    fi
    # 目录层
    if [ -d "$HOME/.hermes" ]; then
        echo "  hermes: 目录存在 $HOME/.hermes" >>"$LOG_FILE"
        return 0
    fi
    if exists_dir_in_homes ".hermes"; then
        echo "  hermes: 命中 home 内 .hermes 目录" >>"$LOG_FILE"
        return 0
    fi
    return 1
}

has_lightclawace() {
    # 命令层
    if command -v lightclaw >/dev/null 2>&1; then
        echo "  lightclawace: command -v lightclaw -> $(command -v lightclaw)" >>"$LOG_FILE"
        return 0
    fi
    # 路径 fallback（与 detect_ace_install.sh 一致）
    for p in "$HOME/.lightclaw/bin/lightclaw" "$HOME/.local/bin/lightclaw" "/usr/local/bin/lightclaw"; do
        if [ -x "$p" ]; then
            echo "  lightclawace: 找到可执行 $p" >>"$LOG_FILE"
            return 0
        fi
    done
    if exists_exec_in_homes ".lightclaw/bin/lightclaw"; then
        echo "  lightclawace: 命中 home 内 .lightclaw/bin/lightclaw" >>"$LOG_FILE"
        return 0
    fi
    if exists_exec_in_homes ".local/bin/lightclaw"; then
        echo "  lightclawace: 命中 home 内 .local/bin/lightclaw" >>"$LOG_FILE"
        return 0
    fi
    # 目录层
    if [ -d "$HOME/.lightclaw" ]; then
        echo "  lightclawace: 目录存在 $HOME/.lightclaw" >>"$LOG_FILE"
        return 0
    fi
    if exists_dir_in_homes ".lightclaw"; then
        echo "  lightclawace: 命中 home 内 .lightclaw 目录" >>"$LOG_FILE"
        return 0
    fi
    return 1
}

has_openclaw() {
    # 命令层
    if command -v openclaw >/dev/null 2>&1; then
        echo "  openclaw: command -v openclaw -> $(command -v openclaw)" >>"$LOG_FILE"
        return 0
    fi
    # 文件/目录层
    if [ -f "$HOME/.openclaw/openclaw.json" ]; then
        echo "  openclaw: 配置文件存在 $HOME/.openclaw/openclaw.json" >>"$LOG_FILE"
        return 0
    fi
    if [ -f "$HOME/.clawdbot/openclaw.json" ]; then
        echo "  openclaw: 配置文件存在 $HOME/.clawdbot/openclaw.json" >>"$LOG_FILE"
        return 0
    fi
    if exists_exec_in_homes ".local/bin/openclaw"; then
        echo "  openclaw: 命中 home 内 .local/bin/openclaw" >>"$LOG_FILE"
        return 0
    fi
    if exists_file_in_homes ".openclaw/openclaw.json"; then
        echo "  openclaw: 命中 home 内 .openclaw/openclaw.json" >>"$LOG_FILE"
        return 0
    fi
    if exists_file_in_homes ".clawdbot/openclaw.json"; then
        echo "  openclaw: 命中 home 内 .clawdbot/openclaw.json" >>"$LOG_FILE"
        return 0
    fi
    return 1
}

# ========== 执行检测（按优先级返回） ==========
echo ">>> 开始检测 agent 类型（优先级: hermes > lightclawace > openclaw）" >>"$LOG_FILE"

DETECTED="unknown"

# 同时记录全部命中，便于排查"多 agent 并存"的异常
HITS=""
if has_hermes; then HITS="${HITS}${HITS:+,}hermes"; fi
if has_lightclawace; then HITS="${HITS}${HITS:+,}lightclawace"; fi
if has_openclaw; then HITS="${HITS}${HITS:+,}openclaw"; fi
echo "  全部命中: [${HITS:-none}]" >>"$LOG_FILE"

case ",$HITS," in
    *,hermes,*)       DETECTED="hermes" ;;
    *,lightclawace,*) DETECTED="lightclawace" ;;
    *,openclaw,*)     DETECTED="openclaw" ;;
    *)                DETECTED="unknown" ;;
esac

echo "  最终判定: $DETECTED" >>"$LOG_FILE"
echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') ==========" >>"$LOG_FILE"

# stdout 最后一行仅输出 token
echo "$DETECTED"
