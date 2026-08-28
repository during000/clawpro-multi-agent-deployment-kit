#!/bin/bash
#
# feishu_bot_creator_hermes.sh
#
# 作用：Hermes 飞书通道扫码接入（员工端）。
# 契约（v4：字符画输出，与 openclaw 对齐）：
#   stdout JSON Lines，show_qrcode 事件 content 字段装 UTF8 字符画。
#
# 可选 TAT 参数：
#   {{greeting}}   - 前端可提交欢迎词（harness CLI 暂不支持，忽略不报错）

set -uo pipefail

export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u 2>/dev/null || echo 0)"
export PYTHONUNBUFFERED=1

LOG_DIR="$HOME/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/feishu_bot_creator_hermes.log"

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

exec 1> >(stdbuf -oL -eL cat)

# 引入 QR 字符画渲染 lib
# %INCLUDE% lib_qr_render.sh
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd 2>/dev/null || echo .)"
if [ -f "${SCRIPT_DIR}/lib_qr_render.sh" ]; then
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
# qr_event：与 OpenClaw feishu_bot_creator.py 输出契约对齐
#   content = JSON 字符串 {"qrlogin":{"token":"<url>"}}
#   前端用 parsed.qrlogin.token 作为 QRCodeCanvas value，无需按 agent_type 分支
qr_event()   {
    local url="$1"
    local escaped_url
    escaped_url="$(json_escape "$url")"
    emit show_qrcode "\"content\":\"{\\\"qrlogin\\\":{\\\"token\\\":\\\"${escaped_url}\\\"}}\""
}
fin_event()  { emit finish   "\"level\":\"$1\",\"message\":\"$(json_escape "$2")\""; }

GREETING="{{greeting}}"
if [ "$GREETING" = "{{greeting}}" ] || [ -z "$GREETING" ]; then
    GREETING=""
fi

log_event login info "Hermes 飞书扫码流程启动"
[ -n "$GREETING" ] && log_event login info "（欢迎词 harness CLI 暂不支持，已忽略）"

ensure_harness_cli || true
if ! command -v harness >/dev/null 2>&1; then
    fin_event error "harness CLI 未安装"
    exit 1
fi

TMPDIR="$(mktemp -d -t hermes-fs.XXXXXX)"
trap 'rm -rf "$TMPDIR" 2>/dev/null || true' EXIT

STDOUT_PIPE="$TMPDIR/stdout"
STDERR_PIPE="$TMPDIR/stderr"
mkfifo "$STDOUT_PIPE" "$STDERR_PIPE"

harness gateway qr-url-fast --platform feishu --timeout 10m \
    > "$STDOUT_PIPE" 2> "$STDERR_PIPE" &
HARNESS_PID=$!

(
    while IFS= read -r line; do
        [ -z "$line" ] && continue
        case "$line" in
            *"QR code issued"*)
                prog_event issued info "飞书二维码已下发，等待扫码"
                ;;
            *"Configured Feishu"*|*"restarted gateway"*)
                prog_event config info "Hermes 已写入凭证并重启网关"
                ;;
            *"access_denied"*|*"expired_token"*|*"denied"*|*"expired"*)
                log_event harness warn "$line"
                ;;
            *)
                log_event harness info "$line"
                ;;
        esac
        echo "[stderr] $line" >> "$LOG_FILE" 2>/dev/null || true
    done < "$STDERR_PIPE"
) &
STDERR_PID=$!

QR_EMITTED=0
while IFS= read -r line; do
    line="$(echo "$line" | tr -d '\r')"
    [ -z "$line" ] && continue
    if [ "$QR_EMITTED" -eq 0 ] && [[ "$line" == http* ]]; then
        log_event qrcode info "飞书二维码 URL 已下发，渲染字符画中…"
        qr_event "$line"
        QR_EMITTED=1
        log_event qrcode info "字符画二维码已输出，请使用飞书/Lark 扫描"
    else
        log_event harness info "$line"
    fi
    echo "[stdout] $line" >> "$LOG_FILE" 2>/dev/null || true
done < "$STDOUT_PIPE"

wait "$HARNESS_PID"
HARNESS_RC=$?
wait "$STDERR_PID" 2>/dev/null || true

if [ "$HARNESS_RC" -eq 0 ]; then
    if [ "$QR_EMITTED" -eq 0 ]; then
        fin_event error "harness 未输出二维码 URL"
        exit 1
    fi
    fin_event success "飞书接入成功"
    exit 0
else
    fin_event error "harness 退出码 $HARNESS_RC（可能扫码超时或拒绝授权）"
    exit "$HARNESS_RC"
fi
