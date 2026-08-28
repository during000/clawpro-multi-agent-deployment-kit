#!/bin/bash
#
# feishu_bot_creator_ace.sh
#
# 作用：LightClaw-ACE 飞书通道扫码接入（员工端）。
# 本 wrapper 调用 ACE 预装的 Python 脚本 feishu_bot_creator.py，
# 实际扫码与 bot 创建流程全部在 Python 侧处理，wrapper 仅负责：
#   1) 定位并执行 Python 脚本
#   2) stdout 直接透传 JSON Lines（Python 已原生输出标准格式）
#   3) 异常时补发 finish error 事件并清理浏览器僵尸进程
#
# ========== Python 脚本契约（见 scripts/vendor/lightclaw-ace/channel/feishu/feishu_bot_creator.py） ==========
#
# 入口：
#   python3 feishu_bot_creator.py init                              # 预装依赖（playwright + Chromium）
#   python3 feishu_bot_creator.py create [--avatar-url URL] [--greeting 欢迎词] [--platform feishu|lark]
#   python3 feishu_bot_creator.py cleanup [--platform feishu|lark]  # 清理僵尸 Chromium
#
# stdout 行格式（每行一条 JSON）：
#   {"action":"log"|"progress"|"show_qrcode"|"finish", "level":"info|success|warn|error", "step":..., "message":..., ...}
#
# show_qrcode 事件的 content 字段是 JSON 字符串：'{"qrlogin":{"token":"<token>"}}'
#   —— 前端需要自己用该 token 生成二维码图像（与 OpenClaw 的 URL/字符画不同）。
#
# 配置写入：Python 内部写 $LIGHTCLAW_BASE_DIR/lightclaw.json（默认 ~/.lightclaw/lightclaw.json），
#   wrapper 不应重复写。
#
# 超时：首次执行需装 playwright + Chromium，TAT 侧 timeout 应 >= 600s（与 openclaw 飞书一致）。
#
# ========== TAT 可选参数 ==========
#   {{greeting}}    - 前端欢迎词（空则用 Python 默认值："Hi，我是你刚刚使用 LightClaw 创建..."）
#   {{avatar_url}}  - 头像 URL（空则用 Python 默认头像）

set -uo pipefail

export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export PYTHONUNBUFFERED=1

# ⚠️ 关键：若用户自定义了 lightclaw 部署目录，需在此处通过环境变量同步，
# Python 脚本内部会读 LIGHTCLAW_BASE_DIR 写 lightclaw.json。默认 ~/.lightclaw 已由 Python 负责。
# export LIGHTCLAW_BASE_DIR="$HOME/.lightclaw"  # 如有定制部署，取消本行注释

# ACE 脚本未加入 hatchery rootRequiredTATScripts 白名单（controller/tat.go），
# 由 TAT 以实例普通用户身份执行，无权写 /var/log/clawpro，日志落到用户目录。
LOG_DIR="$HOME/.lightclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/feishu_bot_creator_ace.log"

# 行缓冲：让 SSE 下游尽快拿到每条 JSON Lines
exec 1> >(stdbuf -oL -eL cat)

# ========== JSON Lines 辅助（仅用于 wrapper 自身错误报告；Python 输出直接透传） ==========
emit() {
    local action="$1" level="$2" step="$3" message="$4"
    local esc_msg="${message//\\/\\\\}"
    esc_msg="${esc_msg//\"/\\\"}"
    esc_msg="${esc_msg//$'\n'/\\n}"
    printf '{"action":"%s","level":"%s","step":"%s","message":"%s"}\n' \
        "$action" "$level" "$step" "$esc_msg"
}

log_info()   { emit log      info   "$1" "$2"; }
log_warn()   { emit log      warn   "$1" "$2"; }
fin_error()  { emit finish   error  "$1" "$2"; }

# ========== 参数 ==========
GREETING="{{greeting}}"
if [ "$GREETING" = "{{greeting}}" ] || [ -z "$GREETING" ]; then
    GREETING=""
fi

AVATAR_URL="{{avatar_url}}"
if [ "$AVATAR_URL" = "{{avatar_url}}" ] || [ -z "$AVATAR_URL" ]; then
    AVATAR_URL=""
fi

log_info boot "ACE 飞书扫码流程启动"

# ========== 依赖检查 ==========
if ! command -v python3 >/dev/null 2>&1; then
    fin_error dependency "系统缺少 python3，请联系管理员安装"
    exit 1
fi

# ========== 定位 ACE Python 脚本（按优先级枚举，匹配 ACE 镜像可能的部署方式） ==========
#
# 真实路径取决于 ACE 镜像打包方式。以下 4 种候选按概率排序；上线前必须在真实 ACE
# 实例上验证，若均不命中需补充正确路径或 pip module。
#
# ⚠️ ACE 镜像工程师：请在此注释区维护镜像内的真实路径，以便版本迁移时追踪。
CANDIDATES=(
    "$HOME/lightclaw-dashboard/lightclaw-ace/channel/feishu/feishu_bot_creator.py"
    "/opt/lightclaw-ace/channel/feishu/feishu_bot_creator.py"
    "$HOME/.lightclaw/channel/feishu/feishu_bot_creator.py"
    "$HOME/.lightclaw/lightclaw-ace/channel/feishu/feishu_bot_creator.py"
    "/usr/local/lib/lightclaw-ace/channel/feishu/feishu_bot_creator.py"
)

PY_SCRIPT=""
for cand in "${CANDIDATES[@]}"; do
    if [ -f "$cand" ]; then
        PY_SCRIPT="$cand"
        break
    fi
done

# fallback 1：PATH 内有同名可执行脚本
if [ -z "$PY_SCRIPT" ]; then
    PY_SCRIPT="$(command -v feishu_bot_creator.py 2>/dev/null || true)"
fi

# fallback 2：pip 安装形式（import module）
USE_MODULE=0
if [ -z "$PY_SCRIPT" ]; then
    if python3 -c "import lightclaw_ace.channel.feishu.feishu_bot_creator" 2>/dev/null; then
        USE_MODULE=1
    fi
fi

if [ -z "$PY_SCRIPT" ] && [ "$USE_MODULE" -eq 0 ]; then
    fin_error locate "未找到 ACE 飞书配置脚本 feishu_bot_creator.py，请联系管理员确认 ACE 镜像版本。已尝试路径：${CANDIDATES[*]}；以及 PATH 查询、python module 查询"
    exit 1
fi

if [ "$USE_MODULE" -eq 1 ]; then
    log_info locate "定位到 ACE 飞书脚本：python3 -m lightclaw_ace.channel.feishu.feishu_bot_creator"
else
    log_info locate "定位到 ACE 飞书脚本：$PY_SCRIPT"
fi

# ========== 僵尸清理（避免上次 TAT 被杀留下的 Chromium 进程占用 CDP 端口） ==========
cleanup_chromium() {
    if [ "$USE_MODULE" -eq 1 ]; then
        python3 -m lightclaw_ace.channel.feishu.feishu_bot_creator cleanup --platform feishu \
            >/dev/null 2>&1 || true
    else
        python3 "$PY_SCRIPT" cleanup --platform feishu >/dev/null 2>&1 || true
    fi
}

# 启动前预清理，防止上次残留
cleanup_chromium

# 退出时再次清理；即使 TAT 没超时也要释放浏览器资源
trap 'cleanup_chromium' EXIT INT TERM

# ========== 构造命令 ==========
PY_CMD=(python3 -u)
if [ "$USE_MODULE" -eq 1 ]; then
    PY_CMD+=(-m lightclaw_ace.channel.feishu.feishu_bot_creator)
else
    PY_CMD+=("$PY_SCRIPT")
fi
PY_CMD+=(create --platform feishu)

# 可选参数按需追加（避免传空串让 Python 误解析为 cmdline 选项）
if [ -n "$AVATAR_URL" ]; then
    PY_CMD+=(--avatar-url "$AVATAR_URL")
fi
if [ -n "$GREETING" ]; then
    PY_CMD+=(--greeting "$GREETING")
fi

log_info exec "开始执行 ACE 飞书 bot 创建流程（首次可能需 5~10 分钟安装 playwright 依赖）"

# ========== 执行 Python 并逐行透传 stdout ==========
#
# 契约：Python 原生输出标准 JSON Lines，wrapper 直接透传。
# 遇到非 JSON 行（极少见：playwright 装机过程某些 stderr 逃逸到 stdout）时，包装为 log 事件。
TMPDIR="$(mktemp -d -t ace-fs.XXXXXX)"
# 把清理扩展到临时目录
trap 'cleanup_chromium; rm -rf "$TMPDIR" 2>/dev/null || true' EXIT INT TERM
STDERR_FILE="$TMPDIR/stderr"

# 注意：pipe 两端默认各自子 shell，用 PIPESTATUS 读 Python 退出码
set +e
"${PY_CMD[@]}" 2> "$STDERR_FILE" | while IFS= read -r line; do
    [ -z "$line" ] && continue
    # Python 原生 JSON Lines（以 { 起始且含 "action"）直接透传
    if [[ "$line" == \{*\"action\"* ]]; then
        printf '%s\n' "$line"
        echo "[stdout] $line" >> "$LOG_FILE" 2>/dev/null || true
        continue
    fi
    # 非 JSON 行：包装为 log 事件（不污染 SSE 解析）
    log_info python "$line"
    echo "[stdout-raw] $line" >> "$LOG_FILE" 2>/dev/null || true
done
PY_RC=${PIPESTATUS[0]}


# 转存 stderr 到日志供 Ops 排查
if [ -s "$STDERR_FILE" ]; then
    while IFS= read -r eline; do
        echo "[stderr] $eline" >> "$LOG_FILE" 2>/dev/null || true
    done < "$STDERR_FILE"
fi

# ========== 退出处理 ==========
# Python 脚本成功时自己会输出 finish success，wrapper 不重复；
# 只在 Python 异常退出且没输出 finish 事件时补一条 finish error（保底，避免前端卡住）。
if [ "$PY_RC" -ne 0 ]; then
    # 最多取 stderr 尾部 500 字节作为错误说明
    STDERR_TAIL="$(tail -c 500 "$STDERR_FILE" 2>/dev/null || true)"
    fin_error python "ACE 飞书脚本退出码 $PY_RC；stderr: $STDERR_TAIL"
    exit "$PY_RC"
fi

exit 0
