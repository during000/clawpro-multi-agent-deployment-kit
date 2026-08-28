#!/bin/bash
#
# weixin_bot_creator_ace.sh
#
# 作用：LightClaw-ACE 微信通道扫码接入（员工端）。
# 契约（v4：与 openclaw weixin_bot_creator.sh 完全一致的字符画输出）：
#   stdout JSON Lines，hatchery HandleAutoChannel 原样透传 SSE 给前端。
#   show_qrcode 事件的 content 字段装 UTF8 字符画（由 lib_qr_render.sh 渲染）。
#
# 实现：调用 ACE lightclaw 本地已运行的 FastAPI
#   （host/port 通过 lib_ace_api.sh 五级发现 + 探活 + 规整，不 hardcode）
#   - POST /api/config/channels/weixin/qrcode/generate
#       → {"qrcode_token":"...", "qrcode_url":"https://..."}
#     拿到 URL 后作为裸 URL 输出给前端用 QRCodeCanvas 渲染
#   - POST /api/config/channels/weixin/qrcode/poll  body={"qrcode_token":"..."}
#       → {"status":"wait|scaned|success|error|confirmed_but_save_failed", ...}
#   - status=success 时 lightclaw 内部已把 account 写入 lightclaw.json
#     且 lightclaw 运行时会热加载（无需额外 systemctl restart）
#
#   注：所有 API 路由挂载在 /api 前缀下（_app.py: app.include_router(api_router, prefix="/api")）
#
# 超时：硬上限 10 分钟；poll 间隔 2 秒

set -uo pipefail

export PATH="$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export PYTHONUNBUFFERED=1

# ACE 脚本未加入 hatchery rootRequiredTATScripts 白名单（controller/tat.go），
# 由 TAT 以实例普通用户身份执行，无权写 /var/log/clawpro，日志落到用户目录。
LOG_DIR="$HOME/.lightclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/weixin_bot_creator_ace.log"

exec 1> >(stdbuf -oL -eL cat)

# 引入 ACE API helper + QR 字符画渲染 lib
# %INCLUDE% lib_ace_api.sh
# %INCLUDE% lib_qr_render.sh
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd 2>/dev/null || echo .)"
if [ -f "${SCRIPT_DIR}/lib_ace_api.sh" ]; then
    # shellcheck disable=SC1091
    . "${SCRIPT_DIR}/lib_ace_api.sh"
fi
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
    s="${s//$'\n'/\\n}"
    s="${s//$'\t'/\\t}"
    printf '%s' "$s"
}
log_event()  { emit log      "\"step\":\"$1\",\"level\":\"$2\",\"message\":\"$(json_escape "$3")\""; }
prog_event() { emit progress "\"step\":\"$1\",\"level\":\"$2\",\"message\":\"$(json_escape "$3")\""; }
# 与 weixin_bot_creator_hermes.sh 对齐：content 为裸 URL，前端用 QRCodeCanvas 渲染
qr_event()   {
    local url="$1"
    local escaped_url
    escaped_url="$(json_escape "$url")"
    emit show_qrcode "\"content\":\"${escaped_url}\""
}
fin_event()  { emit finish   "\"level\":\"$1\",\"message\":\"$(json_escape "$2")\""; }

log_event login info "ACE 微信扫码流程启动"

# 依赖
if ! command -v curl >/dev/null 2>&1; then
    fin_event error "curl 未安装"; exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
    fin_event error "jq 未安装"; exit 1
fi

# 1) 通用发现 API base（五级回退 + 探活，不 hardcode 端口）
BASE="$(resolve_ace_api_base)" || {
    fin_event error "未能发现存活的 ACE API 端口（尝试过 env / lastApi / 8088 均失败，请确认 lightclaw 服务已启动）"
    exit 1
}
log_event login info "ACE API base=${BASE}"

# 2) 生成二维码
GEN_RESP="$(curl -sS -X POST "${BASE}/api/config/channels/weixin/qrcode/generate" \
    -H 'Content-Type: application/json' \
    --connect-timeout 5 --max-time 30 2>> "$LOG_FILE" || true)"

if [ -z "$GEN_RESP" ]; then
    fin_event error "调用 ACE qrcode/generate 无响应"
    exit 1
fi
echo "[generate] $GEN_RESP" >> "$LOG_FILE" 2>/dev/null || true

QR_TOKEN="$(echo "$GEN_RESP" | jq -r '.qrcode_token // empty' 2>/dev/null)"
QR_URL="$(  echo "$GEN_RESP" | jq -r '.qrcode_url   // empty' 2>/dev/null)"

if [ -z "$QR_TOKEN" ] || [ -z "$QR_URL" ]; then
    MSG="$(echo "$GEN_RESP" | jq -r '.detail // .message // "unknown error"' 2>/dev/null || echo 'unknown error')"
    fin_event error "ACE 生成二维码失败：$MSG"
    exit 1
fi

log_event qrcode info "微信二维码 URL 已获取，渲染字符画中…"
qr_event "$QR_URL"
log_event qrcode info "字符画二维码已输出，请使用微信扫描"

# 3) 轮询
DEADLINE=$(( $(date +%s) + 600 ))   # 10 分钟上限
LAST_STATUS=""
while [ "$(date +%s)" -lt "$DEADLINE" ]; do
    sleep 2

    POLL_RESP="$(curl -sS -X POST "${BASE}/api/config/channels/weixin/qrcode/poll" \
        -H 'Content-Type: application/json' \
        -d "{\"qrcode_token\":\"$(json_escape "$QR_TOKEN")\"}" \
        --connect-timeout 5 --max-time 45 2>> "$LOG_FILE")"
    echo "[poll] $POLL_RESP" >> "$LOG_FILE" 2>/dev/null || true


    STATUS="$(echo "$POLL_RESP" | jq -r '.status // "wait"' 2>/dev/null)"
    [ -z "$STATUS" ] && STATUS="wait"

    if [ "$STATUS" != "$LAST_STATUS" ]; then
        case "$STATUS" in
            wait)
                prog_event scan info "等待扫码…"
                ;;
            scaned)
                prog_event scan info "已扫码，等待确认"
                ;;
            success)
                ACCT_ID="$(echo "$POLL_RESP" | jq -r '.account_id // ""' 2>/dev/null)"
                if [ -n "$ACCT_ID" ] && [ "$ACCT_ID" != "null" ]; then
                    prog_event save info "账号已写入 lightclaw.json：${ACCT_ID}"
                fi
                # ACE API 只写账号，不设 enabled=true，需用 CLI 补充
                lightclaw channels set weixin --set enabled=true >> "$LOG_FILE" 2>&1 || true
                fin_event success "微信接入成功"
                exit 0
                ;;
            confirmed_but_save_failed)
                MSG="$(echo "$POLL_RESP" | jq -r '.message // "QR confirmed but save failed"' 2>/dev/null)"
                fin_event error "扫码成功但 ACE 保存配置失败：${MSG}"
                exit 1
                ;;
            error)
                MSG="$(echo "$POLL_RESP" | jq -r '.message // "unknown"' 2>/dev/null)"
                fin_event error "ACE 扫码失败：${MSG}"
                exit 1
                ;;
            *)
                log_event poll info "ACE 状态=${STATUS}"
                ;;
        esac
        LAST_STATUS="$STATUS"
    fi
done

fin_event error "扫码超时（10 分钟未完成）"
exit 1
