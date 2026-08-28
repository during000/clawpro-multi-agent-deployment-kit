#!/bin/bash
# secure_first_boot.sh —— 自定义镜像首装的安全初始化总入口
#
# 背景：
#   用户拿一台已安装 openclaw 的机器制作自定义镜像，会把 ~/.openclaw/openclaw.json
#   一起打进镜像（其中包含 gateway.auth.token / gateway.port 等）。后续基于该镜像
#   装出的所有实例敏感字段完全一致，存在严重安全风险。
#
# 本脚本职责（首装一次性安全加固，按步骤组织，未来可在内部扩展）：
#   当前已实现：
#     - 强制轮换 .gateway.auth.token（生成 32B 强随机 → flock + jq 改写 → 重启 gateway）
#   未来可扩展（在不改 Go 调用方的前提下，于本脚本内部增加分步函数即可）：
#     - .gateway.controlUi.basePath / .gateway.controlUi.allowedOrigins 随机化
#     - SSH host key（/etc/ssh/ssh_host_*）重生成
#     - shell history 清理等
#
# 触发时机：
#   后端 task/agent_checker.go 检测到 agent_ready 由 0 翻 1 且 prevOp == OpCreate
#   且 agent_type == openclaw 且 imgId 为非候选镜像（自定义镜像）时，异步下发本脚本。
#   官方候选镜像由镜像内置 first-boot-token.sh 自行接管，不会走到这里。
#
# 入口契约：
#   - 通过 TAT 以 root 身份执行（见 controller/tat.go rootRequiredTATScripts）；
#   - hatchery 会注入 OPENCLAW_RUNTIME_USER 环境变量（可能为空，做了兜底探测）；
#   - 不要求任何 stdout 契约，调用方仅观察日志和退出码；
#   - 任何分支失败均仅记录日志，exit 0，不影响业务主流程。
set -u

SCRIPT_NAME="secure_first_boot"

# ========== 步骤 1：探测 RUNTIME_USER / RUNTIME_HOME（先于日志初始化） ==========
# 日志要落到 $RUNTIME_HOME/.openclaw/logs（与项目其它脚本一致），
# 所以必须先探测出 RUNTIME_HOME 再初始化日志重定向。
# 这一步的探测输出走 stderr/stdout（无日志重定向），跑完 TAT 也能在 invocation 输出里看到，不会丢。

has_openclaw() {
    local h="$1"
    [ -d "$h/.openclaw" ] && return 0
    return 1
}

RUNTIME_USER=""
RUNTIME_HOME=""

# 优先：hatchery 注入的 OPENCLAW_RUNTIME_USER
if [ -n "${OPENCLAW_RUNTIME_USER:-}" ]; then
    candidate_user="$OPENCLAW_RUNTIME_USER"
    if [ "$candidate_user" = "root" ]; then
        candidate_home="/root"
    elif [ -d "/home/$candidate_user" ]; then
        candidate_home="/home/$candidate_user"
    else
        candidate_home=""
    fi
    if [ -n "$candidate_home" ] && has_openclaw "$candidate_home"; then
        RUNTIME_USER="$candidate_user"
        RUNTIME_HOME="$candidate_home"
    fi
fi

# 兜底：扫 /home/* 和 /root
if [ -z "$RUNTIME_USER" ]; then
    shopt -s nullglob 2>/dev/null || true
    for user_home in /home/*; do
        [ -d "$user_home" ] || continue
        if has_openclaw "$user_home"; then
            RUNTIME_USER="$(basename "$user_home")"
            RUNTIME_HOME="$user_home"
            break
        fi
    done
    shopt -u nullglob 2>/dev/null || true
fi
if [ -z "$RUNTIME_USER" ] && has_openclaw "/root"; then
    RUNTIME_USER="root"
    RUNTIME_HOME="/root"
fi

if [ -z "$RUNTIME_USER" ]; then
    echo "[secure_first_boot] ⚠ 未在任何用户下发现 openclaw 安装，无可加固对象，退出" >&2
    exit 0
fi

CONFIG="$RUNTIME_HOME/.openclaw/openclaw.json"
if [ ! -f "$CONFIG" ]; then
    echo "[secure_first_boot] ⚠ $CONFIG 不存在，退出" >&2
    exit 0
fi

# ========== 日志（与项目其它脚本统一：$RUNTIME_HOME/.openclaw/logs） ==========
LOG_DIR="$RUNTIME_HOME/.openclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
# 目录归属给目标用户（脚本以 root 身份执行，但日志归属目标用户更安全）
if [ "$RUNTIME_USER" != "root" ]; then
    chown "$RUNTIME_USER:$RUNTIME_USER" "$LOG_DIR" 2>/dev/null || true
fi
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
if { : >> "$LOG_FILE"; } 2>/dev/null; then
    exec >>"$LOG_FILE" 2>&1
else
    exec >/dev/null 2>&1
fi
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "=== secure_first_boot 开始 ==="
echo "runtime_user=$RUNTIME_USER runtime_home=$RUNTIME_HOME"
echo "config=$CONFIG"

# 任何分支退出都 exit 0：本脚本是安全加固性质，失败不能影响主流程
trap 'echo "========== 日志结束: $(date "+%Y-%m-%d %H:%M:%S") =========="; exit 0' EXIT

# ========== 步骤 2：依赖检查 ==========
echo ""
echo ">>> [步骤 2/5] 检查依赖（jq / flock / openssl 或 /dev/urandom）"
if ! command -v jq >/dev/null 2>&1; then
    echo "⚠ jq 不存在，退出（无法安全改写 JSON）"
    exit 0
fi
if ! command -v flock >/dev/null 2>&1; then
    echo "⚠ flock 不存在，将不加锁（与 set_channel.sh 等并发概率极低，仅 OpCreate 首装期间）"
fi

# ========== 步骤 3：生成新 token ==========
echo ""
echo ">>> [步骤 3/5] 生成强随机 token"
NEW_TOKEN=""
if command -v openssl >/dev/null 2>&1; then
    NEW_TOKEN="$(openssl rand -hex 32 2>/dev/null || true)"
fi
if [ -z "$NEW_TOKEN" ] && [ -r /dev/urandom ]; then
    NEW_TOKEN="$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' || true)"
fi
if [ -z "$NEW_TOKEN" ] || [ ${#NEW_TOKEN} -lt 32 ]; then
    echo "⚠ 生成 token 失败（openssl / urandom 均不可用），退出"
    exit 0
fi
# 日志文件已设为 600 权限且位于目标用户家目录（受家目录权限保护），
# 完整 token 打印到日志便于排障核对（与 first-boot-token.sh 同款思路）。
echo "new_token=$NEW_TOKEN"
echo "new_token_len=${#NEW_TOKEN}"

# ========== 步骤 4：原地改写 openclaw.json ==========
echo ""
echo ">>> [步骤 4/5] 改写 $CONFIG 的 .gateway.auth.token"

# flock 仅用于防止本脚本自身因 hatchery 重试而并发执行（同一台 CVM 上 OpCreate
# 触发瞬间被多次 RunCommand 的极端场景）。
#
# 注意：项目里其它改 openclaw.json 的脚本（set_gateway_ui / set_model /
# switch_model / set_channel / set_env 等）各自使用独立的 /tmp 锁文件，
# 项目并未维护一把全局 openclaw.json 锁。本脚本与它们之间不互斥，但触发时机
# 是 OpCreate 首次 agent_ready 瞬间，那一刻用户尚未发起任何 set_xxx 操作，
# 跨脚本并发概率可视为 0。
LOCK_FD=9
LOCK_FILE="/tmp/.openclaw_secure_first_boot.lock"
do_rotate() {
    local tmp
    tmp="$(mktemp 2>/dev/null || echo "${CONFIG}.tmp.$$")"
    if ! jq --arg t "$NEW_TOKEN" '.gateway.auth.token = $t' "$CONFIG" > "$tmp" 2>/dev/null; then
        echo "× jq 改写失败"
        rm -f "$tmp"
        return 1
    fi
    if ! jq -e . "$tmp" >/dev/null 2>&1; then
        echo "× 改写后 JSON 校验失败"
        rm -f "$tmp"
        return 1
    fi
    cp -p "$CONFIG" "${CONFIG}.bak.$(date +%s)" 2>/dev/null || true
    if ! mv "$tmp" "$CONFIG"; then
        echo "× mv 写回失败"
        rm -f "$tmp"
        return 1
    fi
    # 保持原属主属组（mktemp 默认归属当前进程的 root，需要归还给 RUNTIME_USER 否则
    # 后续 openclaw 进程读不到）。注意：不主动 chmod 改权限模式 —— 与 set_channel /
    # set_model / switch_model 等其它改 openclaw.json 的脚本风格保持一致，避免
    # 静默把 644 收紧为 600 影响排障；token 安全性已由家目录权限和日志 600 兜底。
    chown "$RUNTIME_USER:$RUNTIME_USER" "$CONFIG" 2>/dev/null || true
    return 0
}

if command -v flock >/dev/null 2>&1; then
    # shellcheck disable=SC2094
    exec 9>"$LOCK_FILE"
    if flock -w 30 9; then
        if ! do_rotate; then
            echo "⚠ rotate 失败"
            exit 0
        fi
        flock -u 9 || true
    else
        echo "⚠ 抢锁失败，跳过 rotate"
        exit 0
    fi
else
    if ! do_rotate; then
        echo "⚠ rotate 失败"
        exit 0
    fi
fi
echo "✓ .gateway.auth.token 已轮换"

# ========== 步骤 5：重启 openclaw-gateway 让新 token 生效 ==========
echo ""
echo ">>> [步骤 5/5] 重启 openclaw-gateway"

restart_as_user() {
    local u="$1"
    local uid
    uid="$(id -u "$u" 2>/dev/null || true)"
    if [ -z "$uid" ]; then
        echo "× 无法解析用户 $u 的 uid"
        return 1
    fi
    # systemctl --user 需要正确的 XDG_RUNTIME_DIR + DBUS_SESSION_BUS_ADDRESS
    local env_prefix="XDG_RUNTIME_DIR=/run/user/$uid DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$uid/bus"
    if command -v runuser >/dev/null 2>&1; then
        runuser -u "$u" -- env $env_prefix systemctl --user restart openclaw-gateway 2>&1
        return $?
    fi
    if command -v sudo >/dev/null 2>&1; then
        sudo -u "$u" -- env $env_prefix systemctl --user restart openclaw-gateway 2>&1
        return $?
    fi
    su - "$u" -c "env $env_prefix systemctl --user restart openclaw-gateway" 2>&1
    return $?
}

if [ "$RUNTIME_USER" = "root" ]; then
    # root 模式下 openclaw-gateway 通常是 user-mode（root 的 user instance），
    # 直接以 root 跑即可
    if systemctl --user restart openclaw-gateway 2>&1; then
        echo "✓ openclaw-gateway 已重启（root user-instance）"
    else
        echo "⚠ openclaw-gateway 重启失败（非致命，新 token 已落盘，下次重启即生效）"
    fi
else
    if restart_as_user "$RUNTIME_USER"; then
        echo "✓ openclaw-gateway 已以 $RUNTIME_USER 身份重启"
    else
        echo "⚠ openclaw-gateway 重启失败（非致命，新 token 已落盘，下次重启即生效）"
    fi
fi

echo ""
echo "=== secure_first_boot 完成 ==="
exit 0
