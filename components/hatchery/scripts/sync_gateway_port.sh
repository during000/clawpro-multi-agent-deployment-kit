#!/bin/bash
set -euo pipefail

# ========== 运行用户解析 ==========
RUNTIME_USER="{{runtime_user}}"
if [ -z "$RUNTIME_USER" ] || [ "$RUNTIME_USER" = "{{runtime_user}}" ]; then
    RUNTIME_USER="${OPENCLAW_RUNTIME_USER:-root}"
fi

if [ "$RUNTIME_USER" != "root" ] && ! id "$RUNTIME_USER" >/dev/null 2>&1; then
    echo "WARN: runtime user '$RUNTIME_USER' 不存在，回退到 root"
    RUNTIME_USER="root"
fi

TARGET_UID="$(id -u "$RUNTIME_USER" 2>/dev/null || id -u)"
TARGET_HOME=""
if [ -r /etc/passwd ]; then
    while IFS=: read -r _pw_name _pw_x _pw_uid _pw_gid _pw_gecos _pw_dir _pw_shell; do
        if [ "$_pw_name" = "$RUNTIME_USER" ]; then
            TARGET_HOME="$_pw_dir"
            break
        fi
    done < /etc/passwd
    unset _pw_name _pw_x _pw_uid _pw_gid _pw_gecos _pw_dir _pw_shell
fi
if [ -z "$TARGET_HOME" ]; then
    if [ "$RUNTIME_USER" = "root" ]; then
        TARGET_HOME="/root"
    else
        TARGET_HOME="/home/$RUNTIME_USER"
    fi
fi
export HOME="$TARGET_HOME"

run_as_runtime_user() {
    if [ "$(id -un)" = "$RUNTIME_USER" ]; then
        "$@"
        return
    fi
    if command -v runuser >/dev/null 2>&1; then
        runuser -u "$RUNTIME_USER" -- "$@"
        return
    fi
    su - "$RUNTIME_USER" -s /bin/bash -c "$(printf '%q ' "$@")"
}

user_systemctl() {
    if [ "$(id -un)" = "$RUNTIME_USER" ]; then
        XDG_RUNTIME_DIR="/run/user/${TARGET_UID}" systemctl --user "$@"
        return
    fi
    run_as_runtime_user env XDG_RUNTIME_DIR="/run/user/${TARGET_UID}" systemctl --user "$@"
}

# ========== 日志设置 ==========
# 与 restore_post_reinstall.sh / set_channel.sh 等脚本保持一致：
# 日志落到 ~/.openclaw/logs/ 下，方便在实例本地排查升级后 gateway 端口问题。
SCRIPT_NAME="sync_gateway_port"
LOG_DIR="${HOME}/.openclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || LOG_DIR="/tmp"
chmod 700 "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
touch "$LOG_FILE" 2>/dev/null || true
chmod 600 "$LOG_FILE" 2>/dev/null || true
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== $(date '+%Y-%m-%d %H:%M:%S') sync_gateway_port 开始 =========="
echo "运行用户: $RUNTIME_USER (UID=$TARGET_UID, HOME=$TARGET_HOME)"

# ========== 同步 openclaw-gateway.service 端口 ==========
# 原因：openclaw gateway install 生成 service 文件时使用默认端口，
# 而备份恢复后 openclaw.json 中的 gateway.port 可能是用户自定义端口，两者不一致会导致
# gateway 进程监听的端口与 openclaw.json 中记录的端口不同，造成 WS 连接不可用。
main() {
    local cfg="$HOME/.openclaw/openclaw.json"
    local GATEWAY_SERVICE_FILE="$HOME/.config/systemd/user/openclaw-gateway.service"

    if [ ! -f "$cfg" ]; then
        echo "⚠ openclaw.json 不存在，跳过端口同步"
        exit 0
    fi
    if [ ! -f "$GATEWAY_SERVICE_FILE" ]; then
        echo "⚠ openclaw-gateway.service 不存在，跳过端口同步"
        exit 0
    fi
    if ! command -v jq >/dev/null 2>&1; then
        echo "⚠ jq 未安装，跳过端口同步"
        exit 0
    fi

    local JSON_PORT
    JSON_PORT=$(jq -r '.gateway.port // empty' "$cfg" 2>/dev/null || true)

    if [ -z "$JSON_PORT" ] || [ "$JSON_PORT" = "null" ] || [ "$JSON_PORT" = "0" ]; then
        echo "⚠ openclaw.json 中 gateway.port 为空，跳过 service 文件端口同步"
        exit 0
    fi

    # 只对 unit 中"实际存在"的端口写法做一致性判断：
    # 不同 openclaw 版本的 unit 模板可能只含 Environment 或只含 --port 之一，
    # 若对不存在的写法也要求匹配端口，会导致 UNIT_NEEDS_UPDATE 恒 true、每次升级都多做无谓 restart。
    local UNIT_NEEDS_UPDATE=false
    if grep -q "Environment=OPENCLAW_GATEWAY_PORT=" "$GATEWAY_SERVICE_FILE" \
       && ! grep -Eq "Environment=OPENCLAW_GATEWAY_PORT=${JSON_PORT}([^0-9]|$)" "$GATEWAY_SERVICE_FILE"; then
        UNIT_NEEDS_UPDATE=true
    fi
    if grep -Eq "gateway --port [0-9]+" "$GATEWAY_SERVICE_FILE" \
       && ! grep -Eq "gateway --port ${JSON_PORT}([^0-9]|$)" "$GATEWAY_SERVICE_FILE"; then
        UNIT_NEEDS_UPDATE=true
    fi
    if [ "$UNIT_NEEDS_UPDATE" = "true" ]; then
        sed -i \
            -e "s/Environment=OPENCLAW_GATEWAY_PORT=.*/Environment=OPENCLAW_GATEWAY_PORT=${JSON_PORT}/" \
            -e "s/gateway --port [0-9]*/gateway --port ${JSON_PORT}/" \
            "$GATEWAY_SERVICE_FILE"
        echo "✓ openclaw-gateway.service 端口配置已同步为 $JSON_PORT（来自 openclaw.json gateway.port）"
    else
        echo "✓ openclaw-gateway.service 端口配置已与 openclaw.json 一致（$JSON_PORT）"
    fi

    # 关键修复：unit 文件即使已经写对了端口，运行中的旧进程仍可能绑定着旧端口
    # （例如 unit 是解压备份后才生成正确端口，但进程从未随之重启过）。
    # 直接检测"期望端口是否已被监听"，比"抓第一个 node 监听端口再比较"更稳健：
    # 机器上可能有其它 node 服务（skillhub 等）在监听，抓错端口会误判需重启或误报不一致。
    _port_listening() {
        run_as_runtime_user bash -c "ss -tlnp 2>/dev/null | grep -Eq ':${JSON_PORT}[[:space:]]'"
    }

    local PORT_LISTENING=false
    if _port_listening; then PORT_LISTENING=true; fi

    if [ "$UNIT_NEEDS_UPDATE" = "true" ] || [ "$PORT_LISTENING" != "true" ]; then
        echo "  期望端口 $JSON_PORT 监听状态: ${PORT_LISTENING} → 需要重启 gateway 生效"
        user_systemctl daemon-reload || true
        user_systemctl restart openclaw-gateway || true

        # 轮询等待 gateway 起来监听目标端口（冷启动可能需要 4~10 秒，固定 sleep 2 会误报未监听）
        local WAIT_MAX=20
        local WAITED=0
        PORT_LISTENING=false
        while [ "$WAITED" -lt "$WAIT_MAX" ]; do
            if _port_listening; then PORT_LISTENING=true; break; fi
            sleep 1
            WAITED=$((WAITED + 1))
        done

        if [ "$PORT_LISTENING" = "true" ]; then
            echo "✓ gateway 已重启，期望端口 $JSON_PORT 已在监听（等待 ${WAITED}s）"
        else
            echo "⚠ gateway 重启后 ${WAIT_MAX}s 内期望端口 $JSON_PORT 仍未监听，请手动排查"
        fi
    else
        echo "✓ 期望端口已在监听（$JSON_PORT），无需重启"
    fi
}

main "$@"
exit 0
