#!/bin/bash
set -uo pipefail

# ========== 获取 OpenClaw Gateway WebSocket 连接信息 ==========
# 功能：确保 gateway 可从内网访问（bind=lan），使用管理后台分配的端口，然后读取 authToken。
# 供 Hatchery 拼接内网 WebSocket URL 返回给 SDK 调用方。
#
# 与 set_gateway_ui.sh 的区别：
#   - allowedOrigins 设为 ["*"]（SDK/WS 场景 Origin 不可预知，需放通所有）
#   - 不修改 systemd service 文件（除非需要重启时同步端口）
#   - 仅在必要时修改 gateway.port、gateway.bind、dangerouslyDisableDeviceAuth、allowedOrigins 和 systemd service 端口
#
# 参数（由 Go 程序注入）：
#   gateway_ui_port - 管理后台分配的端口号
#
# 输出（最后一行 JSON，通过 fd 3）：
#   {"port": 8080, "authToken": "xxxxxx"}
# 失败时输出：{"error": "错误描述"}

GATEWAY_UI_PORT="{{gateway_ui_port}}"

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="get_ws_url"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"

# stdout 契约：仅最终 JSON 走真 stdout（fd 3）
exec 3>&1
if { : >> "$LOG_FILE"; } 2>/dev/null; then
    exec >>"$LOG_FILE" 2>&1
else
    exec >/dev/null 2>&1
fi

_JSON_EMITTED=0
_json_escape() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  s="${s//$'\n'/\\n}"
  s="${s//$'\r'/\\r}"
  s="${s//$'\t'/\\t}"
  printf '%s' "$s"
}
_emit_error_json() {
  [ "$_JSON_EMITTED" = "1" ] && return 0
  local msg="${1:-脚本异常退出}"
  printf '{"error":"%s"}\n' "$(_json_escape "$msg")" >&3
  _JSON_EMITTED=1
}
_fatal() {
  local msg="${1:-脚本异常退出}"
  echo "✗ $msg"
  _emit_error_json "$msg"
  exit 1
}
_on_err() {
  local ec=$?
  local line="${BASH_LINENO[0]:-?}"
  echo "✗ 脚本在第 ${line} 行以退出码 ${ec} 异常终止"
  _emit_error_json "脚本在第 ${line} 行以退出码 ${ec} 异常终止，详见 ${LOG_FILE}"
}
trap _on_err ERR
trap '_emit_error_json "脚本在未输出正常结果前退出，详见 $LOG_FILE"' EXIT

# ========== 互斥锁（防止并发执行导致 gateway 反复重启）==========
LOCK_FILE="/tmp/openclaw_get_ws_url.lock"
exec {LOCK_FD}>"$LOCK_FILE"
if ! flock -n "$LOCK_FD"; then
    echo "⚠ 另一个 get_ws_url 脚本正在运行中，等待其完成（最多 60s）..."
    if flock -w 60 "$LOCK_FD"; then
        echo "✓ 获取锁成功，前序脚本已完成"
    else
        _fatal "获取互斥锁超时（60s），可能存在死锁，详见 $LOCK_FILE"
    fi
fi

echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "=== 获取 Gateway WebSocket 连接信息 ==="

# ========== 加载 openclaw 运行环境 ==========
export PNPM_HOME="${PNPM_HOME:-$HOME/.local/share/pnpm}"
export PATH="$PNPM_HOME:$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"

CONFIG="$HOME/.openclaw/openclaw.json"
echo "配置文件路径: $CONFIG"

# ========== 检查配置文件 ==========
if [ ! -f "$CONFIG" ]; then
    _fatal "配置文件不存在: $CONFIG，OpenClaw 可能未安装完成"
fi
echo "✓ 配置文件存在"

# ========== TCP 探测辅助函数 ==========
tcp_probe() {
  if command -v timeout >/dev/null 2>&1; then
    timeout 1 bash -c "exec 9<>/dev/tcp/$1/$2" 2>/dev/null
  else
    ( exec 9<>"/dev/tcp/$1/$2" ) 2>/dev/null
  fi
}

# ========== 读取当前配置 ==========
echo ""
echo ">>> [步骤 1/4] 读取当前 Gateway 配置"
IFS=$'\t' read -r CUR_PORT CUR_BIND CUR_DISABLE_DEV CUR_ORIGIN CUR_BASE_PATH CUR_TOKEN < <(
  jq -r '[
    (.gateway.port // "" | tostring),
    (.gateway.bind // ""),
    (.gateway.controlUi.dangerouslyDisableDeviceAuth | tostring),
    (.gateway.controlUi.allowedOrigins[0] // ""),
    (.gateway.controlUi.basePath // ""),
    (.gateway.auth.token // "")
  ] | @tsv' "$CONFIG" 2>/dev/null || echo $'\t\t\t\t\t'
)
echo "当前配置: port=$CUR_PORT bind=$CUR_BIND dangerouslyDisableDeviceAuth=$CUR_DISABLE_DEV allowedOrigins[0]=$CUR_ORIGIN basePath=$CUR_BASE_PATH"
echo "目标端口: $GATEWAY_UI_PORT"

if [ -z "$GATEWAY_UI_PORT" ] || [ "$GATEWAY_UI_PORT" = "0" ]; then
    _fatal "Go 注入的 gateway_ui_port 为空或为 0"
fi
if [ -z "$CUR_TOKEN" ] || [ "$CUR_TOKEN" = "null" ]; then
    _fatal "配置文件中 gateway.auth.token 为空"
fi

# ========== 确保 gateway 配置正确并可从内网访问 ==========
echo ""
echo ">>> [步骤 2/4] 确保 Gateway 配置正确并对内网可访问"
NEED_RESTART=false

if [ "$CUR_PORT" != "$GATEWAY_UI_PORT" ] || [ "$CUR_BIND" != "lan" ] || [ "$CUR_DISABLE_DEV" != "true" ] || [ "$CUR_ORIGIN" != "*" ]; then
    echo "⚠ 当前配置不满足要求（port=$CUR_PORT→$GATEWAY_UI_PORT, bind=$CUR_BIND→lan, disableDeviceAuth=$CUR_DISABLE_DEV→true, origin=$CUR_ORIGIN→*），需要调整"

    # 备份配置
    BAK_FILE="${CONFIG}.bak.$(date +%Y%m%dT%H%M%S).$$"
    cp "$CONFIG" "$BAK_FILE" &
    BAK_PID=$!

    # 修改 port、bind、dangerouslyDisableDeviceAuth 和 allowedOrigins
    # allowedOrigins=["*"]：SDK/WS 场景下客户端 Origin 不可预知（浏览器页面地址各异），
    # 必须放通所有 Origin，否则新版 OpenClaw Gateway 会严格校验并拒绝连接。
    jq --argjson port "$GATEWAY_UI_PORT" '
      .gateway.port = $port |
      .gateway.bind = "lan" |
      .gateway.controlUi.dangerouslyDisableDeviceAuth = true |
      .gateway.controlUi.allowedOrigins = ["*"]
    ' "$CONFIG" > /tmp/openclaw_ws.json
    mv /tmp/openclaw_ws.json "$CONFIG"
    echo "✓ 配置已更新: port=$GATEWAY_UI_PORT, bind=lan, dangerouslyDisableDeviceAuth=true, allowedOrigins=[*]"

    wait "$BAK_PID" 2>/dev/null || true
    NEED_RESTART=true
else
    echo "✓ 配置已满足要求，无需修改"
fi

# 检查 gateway 是否在目标端口运行
PORT_REACHABLE=false
if tcp_probe 127.0.0.1 "$GATEWAY_UI_PORT"; then
    echo "✓ gateway 端口 $GATEWAY_UI_PORT 可达"
    PORT_REACHABLE=true
else
    echo "⚠ gateway 端口 $GATEWAY_UI_PORT 不可达"

    # 检查服务当前状态，避免在启动/重启过程中重复重启
    CUR_SVC_STATE=$(systemctl --user is-active openclaw-gateway 2>/dev/null || echo "unknown")
    echo "当前服务状态: $CUR_SVC_STATE"

    if [ "$CUR_SVC_STATE" = "activating" ]; then
        echo "⚠ gateway 正在启动中(activating)，等待其就绪而非再次重启"
        # 防止在满足配置的情况下重置 NEED_RESTART
        if [ "$NEED_RESTART" != "true" ]; then
            NEED_WAIT=true
        fi
    elif [ "$CUR_SVC_STATE" = "active" ]; then
        echo "⚠ gateway 服务为 active 但端口不可达（异常状态），需要重启"
        NEED_RESTART=true
    else
        # inactive, failed, unknown 等情况
        echo "⚠ gateway 服务状态为 $CUR_SVC_STATE，需要重启"
        NEED_RESTART=true
    fi
fi

if [ "$NEED_RESTART" = "true" ] || [ "${NEED_WAIT:-false}" = "true" ]; then
    if [ "$NEED_RESTART" = "true" ]; then
        # 更新 systemd service 文件中的端口（同 set_gateway_ui.sh 逻辑）
        GATEWAY_SERVICE="$HOME/.config/systemd/user/openclaw-gateway.service"
        if [ -f "$GATEWAY_SERVICE" ]; then
            if ! grep -Eq "Environment=OPENCLAW_GATEWAY_PORT=${GATEWAY_UI_PORT}([^0-9]|$)" "$GATEWAY_SERVICE" \
               || ! grep -Eq "gateway --port ${GATEWAY_UI_PORT}([^0-9]|$)" "$GATEWAY_SERVICE"; then
                sed -i \
                    -e "s/Environment=OPENCLAW_GATEWAY_PORT=.*/Environment=OPENCLAW_GATEWAY_PORT=${GATEWAY_UI_PORT}/" \
                    -e "s/gateway --port [0-9]*/gateway --port ${GATEWAY_UI_PORT}/" \
                    "$GATEWAY_SERVICE"
                systemctl --user daemon-reload >/dev/null 2>&1
                echo "✓ service 文件端口已更新为 $GATEWAY_UI_PORT 并执行 daemon-reload"
            else
                echo "✓ service 文件端口已是目标值，跳过"
            fi
        else
            echo "⚠ service 文件不存在: $GATEWAY_SERVICE，跳过 systemd 端口更新"
        fi

        # 重启前再次检查服务状态：如果已在 activating，跳过 restart
        CUR_SVC_STATE=$(systemctl --user is-active openclaw-gateway 2>/dev/null || echo "unknown")
        if [ "$CUR_SVC_STATE" = "activating" ]; then
            echo "⚠ 服务已是 activating 状态，跳过 restart，直接等待就绪"
        else
            echo ">>> [步骤 3/4] 更新 systemd 端口并重启 gateway"
            systemctl --user restart openclaw-gateway >/dev/null 2>&1
            echo "✓ openclaw-gateway 已发送 restart 指令"
        fi
    else
        echo ">>> [步骤 3/4] gateway 正在启动中，等待就绪"
    fi

    # 健康检查（最长 ~30s）
    echo ""
    echo ">>> 等待 gateway 就绪..."
    HEALTHY=false
    ATTEMPTS=0
    MAX_ATTEMPTS=300
    LAST_SVC_STATE=""
    while [ "$ATTEMPTS" -lt "$MAX_ATTEMPTS" ]; do
        ATTEMPTS=$((ATTEMPTS + 1))
        if [ $((ATTEMPTS % 10)) -eq 0 ]; then
            CUR_SVC_STATE=$(systemctl --user is-active openclaw-gateway 2>/dev/null || echo "unknown")
            if [ "$CUR_SVC_STATE" = "failed" ]; then
                LAST_SVC_STATE="failed"
                echo "⚠ openclaw-gateway 服务状态为 failed，提前终止等待"
                break
            fi
        fi
        if tcp_probe 127.0.0.1 "$GATEWAY_UI_PORT"; then
            LAST_SVC_STATE=$(systemctl --user is-active openclaw-gateway 2>/dev/null || echo "unknown")
            if [ "$LAST_SVC_STATE" = "active" ]; then
                HEALTHY=true
                break
            fi
        fi
        sleep 0.1
    done

    if [ "$HEALTHY" != "true" ]; then
        echo "--- 健康检查失败调试信息 ---"
        echo "最近一次 systemctl is-active: ${LAST_SVC_STATE:-未探测到}"
        systemctl --user status openclaw-gateway --no-pager 2>&1 | head -n 20 || true
        journalctl --user -u openclaw-gateway -n 30 --no-pager 2>&1 || true
        _fatal "gateway 健康检查失败（端口: $GATEWAY_UI_PORT，服务状态: ${LAST_SVC_STATE:-未就绪}）"
    fi
    echo "✓ gateway 健康检查通过（尝试 $ATTEMPTS 次，端口: $GATEWAY_UI_PORT）"
else
    echo "✓ gateway 运行正常，无需重启"
fi

# ========== 输出结果 ==========
echo ""
echo ">>> [步骤 4/4] 输出连接信息"
RESULT_JSON=$(jq -c --argjson port "$GATEWAY_UI_PORT" '{
  port: $port,
  authToken: (.gateway.auth.token // ""),
  basePath: (.gateway.controlUi.basePath // "")
}' "$CONFIG" 2>/dev/null || true)

if [ -z "$RESULT_JSON" ]; then
    _fatal "读取配置文件失败或 jq 解析异常"
fi

echo "$RESULT_JSON" >> "$LOG_FILE"
printf '%s\n' "$RESULT_JSON" >&3
_JSON_EMITTED=1

echo ""
echo "=== 获取完成 ==="
echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="

trap - EXIT ERR
