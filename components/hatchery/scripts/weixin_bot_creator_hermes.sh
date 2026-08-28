#!/bin/bash
#
# weixin_bot_creator_hermes.sh
#
# 作用：Hermes 微信通道扫码接入（员工端）。
# 契约（v4：与 openclaw weixin_bot_creator.sh **完全一致的字符画输出**）：
#   stdout JSON Lines，hatchery HandleAutoChannel 解析 action 字段后
#   透传为 SSE 事件给前端。
#
# 事件清单（字段契约对齐 openclaw）：
#   {"action":"log",    "step":"...", "level":"info|warn", "message":"..."}
#   {"action":"show_qrcode", "content":"<UTF8 字符画>"}   ← 与 openclaw 完全一致
#   {"action":"progress","step":"...", "level":"info", "message":"..."}
#   {"action":"finish", "level":"success|error", "message":"..."}
#
# 实现：
#   1. 封装 `harness gateway qr-url-fast --platform weixin --timeout 10m`
#   2. stdout 首行 `http*` → 用 lib_qr_render.sh 的 qr_to_json_content 渲染成字符画 → show_qrcode
#   3. stderr 翻译为 log/progress 事件
#   4. 退出码 0 → finish success；非 0 → finish error
#   5. harness 内部已自动写入 env + 重启 gateway，wrapper 不 restart

set -uo pipefail

export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u 2>/dev/null || echo 0)"
export PYTHONUNBUFFERED=1

LOG_DIR="$HOME/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/weixin_bot_creator_hermes.log"

# ========== 拉取/更新 harness CLI（日志走 LOG_FILE，不污染 stdout JSON Lines）==========
ensure_harness_cli() {
    local HARNESS_URL="https://agentone-1388575217.cos.ap-guangzhou.myqcloud.com/harness-linux-amd64"
    local HARNESS_BIN="$HOME/.local/bin/harness"
    mkdir -p "$(dirname "$HARNESS_BIN")" 2>/dev/null || true
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 拉取 harness CLI: $HARNESS_URL" >> "$LOG_FILE" 2>/dev/null || true
    if curl -fsSL --connect-timeout 10 --max-time 60 --retry 2 --retry-delay 2 \
        "$HARNESS_URL" -o "${HARNESS_BIN}.tmp" 2>>"$LOG_FILE"; then
        chmod +x "${HARNESS_BIN}.tmp"
        mv -f "${HARNESS_BIN}.tmp" "$HARNESS_BIN"
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] harness CLI 已更新: $HARNESS_BIN" >> "$LOG_FILE" 2>/dev/null || true
    else
        rm -f "${HARNESS_BIN}.tmp" 2>/dev/null || true
        if command -v harness >/dev/null 2>&1; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] harness CLI 下载失败，沿用已有版本: $(command -v harness)" >> "$LOG_FILE" 2>/dev/null || true
        else
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] harness CLI 下载失败且本地无已有版本" >> "$LOG_FILE" 2>/dev/null || true
            return 1
        fi
    fi
}

# 强制行缓冲
exec 1> >(stdbuf -oL -eL cat)

# 引入 QR 字符画渲染 lib
# %INCLUDE% lib_qr_render.sh
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd 2>/dev/null || echo .)"
if [ -f "${SCRIPT_DIR}/lib_qr_render.sh" ]; then
    # 本地调试时直接 source（TAT 场景指令行已被替换掉，这里不会再执行）
    # shellcheck disable=SC1091
    . "${SCRIPT_DIR}/lib_qr_render.sh"
fi

emit() {
    local action="$1"; shift
    local body="$*"
    if [ -z "$body" ]; then
        printf '{"action":"%s"}\n' "$action"
    else
        printf '{"action":"%s",%s}\n' "$action" "$body"
    fi
}
json_escape() {
    local s="$1"
    s="${s//\\/\\\\}"
    s="${s//\"/\\\"}"
    printf '%s' "$s"
}
log_event()  { emit log      "\"step\":\"$1\",\"level\":\"$2\",\"message\":\"$(json_escape "$3")\""; }
prog_event() { emit progress "\"step\":\"$1\",\"level\":\"$2\",\"message\":\"$(json_escape "$3")\""; }
# 与 feishu_bot_creator_hermes.sh 对齐：content 为裸 URL，前端用 QRCodeCanvas 渲染
qr_event()   {
    local url="$1"
    local escaped_url
    escaped_url="$(json_escape "$url")"
    emit show_qrcode "\"content\":\"${escaped_url}\""
}
fin_event()  { emit finish   "\"level\":\"$1\",\"message\":\"$(json_escape "$2")\""; }

log_event login info "Hermes 微信扫码流程启动"

# 1) 拉取/更新 harness CLI + 依赖检查
ensure_harness_cli || true
if ! command -v harness >/dev/null 2>&1; then
    fin_event error "harness CLI 未安装"
    exit 1
fi

# 2) 命名管道分离 stdout/stderr
TMPDIR="$(mktemp -d -t hermes-wx.XXXXXX)"
trap 'rm -rf "$TMPDIR" 2>/dev/null || true' EXIT

STDOUT_PIPE="$TMPDIR/stdout"
STDERR_PIPE="$TMPDIR/stderr"
mkfifo "$STDOUT_PIPE" "$STDERR_PIPE"

# 3) 后台跑 harness
harness gateway qr-url-fast --platform weixin --timeout 10m \
    > "$STDOUT_PIPE" 2> "$STDERR_PIPE" &
HARNESS_PID=$!

# 4) 消费 stderr → log / progress 事件
(
    while IFS= read -r line; do
        [ -z "$line" ] && continue
        case "$line" in
            *"QR code scanned"*|*"scaned"*)
                prog_event scan info "用户已扫码，等待确认"
                ;;
            *"Configured Weixin"*|*"restarted gateway"*)
                prog_event config info "Hermes 已写入凭证并重启网关"
                ;;
            *)
                log_event harness info "$line"
                ;;
        esac
        echo "[stderr] $line" >> "$LOG_FILE" 2>/dev/null || true
    done < "$STDERR_PIPE"
) &
STDERR_PID=$!

# 5) 读 stdout 首行 URL → 渲染字符画 → show_qrcode
QR_EMITTED=0
while IFS= read -r line; do
    line="$(echo "$line" | tr -d '\r')"
    [ -z "$line" ] && continue
    if [ "$QR_EMITTED" -eq 0 ] && [[ "$line" == http* ]]; then
        log_event qrcode info "二维码 URL 已下发，渲染字符画中…"
        qr_event "$line"
        QR_EMITTED=1
        log_event qrcode info "字符画二维码已输出，请使用微信扫描"
    else
        log_event harness info "$line"
    fi
    echo "[stdout] $line" >> "$LOG_FILE" 2>/dev/null || true
done < "$STDOUT_PIPE"

# 6) 等 harness 退出
wait "$HARNESS_PID"
HARNESS_RC=$?
wait "$STDERR_PID" 2>/dev/null || true

if [ "$HARNESS_RC" -eq 0 ]; then
    if [ "$QR_EMITTED" -eq 0 ]; then
        fin_event error "harness 未输出二维码 URL"
        exit 1
    fi
    fin_event success "微信接入成功"
    exit 0
else
    fin_event error "harness 退出码 $HARNESS_RC（可能为扫码超时）"
    exit "$HARNESS_RC"
fi
