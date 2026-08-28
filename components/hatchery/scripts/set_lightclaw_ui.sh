#!/bin/bash
set -euo pipefail

GATEWAY_IP="{{gateway_ip}}"
GATEWAY_UI_PORT="{{gateway_ui_port}}"

# ========== 文件锁：避免并发操作导致进程冲突 ==========
LOCK_FILE="/tmp/.lightclaw_set_ui.lock"
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
    echo "LightClaw UI 配置操作正在进行中，请勿重复提交" >&2
    exit 1
fi

# ========== 1. 查找 lightclaw 命令及运行用户 ==========
LIGHTCLAW_BIN=""
LIGHTCLAW_USER=""

if command -v lightclaw >/dev/null 2>&1; then
    LIGHTCLAW_BIN="$(command -v lightclaw)"
    LIGHTCLAW_USER="$(id -un)"
elif [ -x "$HOME/.local/bin/lightclaw" ]; then
    LIGHTCLAW_BIN="$HOME/.local/bin/lightclaw"
    LIGHTCLAW_USER="$(id -un)"
elif [ -x "/usr/local/bin/lightclaw" ]; then
    LIGHTCLAW_BIN="/usr/local/bin/lightclaw"
    LIGHTCLAW_USER="$(id -un)"
else
    for user_home in /home/*; do
        if [ -x "$user_home/.local/bin/lightclaw" ]; then
            LIGHTCLAW_BIN="$user_home/.local/bin/lightclaw"
            LIGHTCLAW_USER="$(basename "$user_home")"
            break
        fi
    done
    [ -z "$LIGHTCLAW_BIN" ] && [ -x "/root/.local/bin/lightclaw" ] && LIGHTCLAW_BIN="/root/.local/bin/lightclaw" && LIGHTCLAW_USER="root"
fi

if [ -z "$LIGHTCLAW_BIN" ]; then
    echo "lightclaw not found" >&2
    exit 1
fi

# ========== 2. 读取当前 Password ==========
# lightclaw-login.txt 在安装用户的 HOME 下
if [ "$(id -un)" = "$LIGHTCLAW_USER" ]; then
    LIGHTCLAW_HOME="$HOME"
elif [ "$LIGHTCLAW_USER" = "root" ]; then
    LIGHTCLAW_HOME="/root"
else
    LIGHTCLAW_HOME="/home/$LIGHTCLAW_USER"
fi

LOGIN_FILE="$LIGHTCLAW_HOME/lightclaw-login.txt"
PASSWORD=""
if [ -f "$LOGIN_FILE" ]; then
    PASSWORD=$(sed -n 's/^Password: *//p' "$LOGIN_FILE" | tr -d '[:space:]' || true)
fi

# 如果没有 login.txt 或无法读取密码，则自动生成随机密码并重置
if [ -z "$PASSWORD" ]; then
    PASSWORD=$(head -c 100 /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 6 || true)
    if [ -z "$PASSWORD" ]; then
        # fallback: 用 $RANDOM 拼接
        PASSWORD=$(printf '%s%s%s' $RANDOM $RANDOM $RANDOM | cut -c1-6)
    fi
    if [ "$(id -un)" = "$LIGHTCLAW_USER" ]; then
        "$LIGHTCLAW_BIN" passwd --force-reset-password "$PASSWORD" >/dev/null 2>&1
    else
        su - "$LIGHTCLAW_USER" -c "'$LIGHTCLAW_BIN' passwd --force-reset-password '$PASSWORD'" >/dev/null 2>&1
    fi
fi

# ========== 3. 停止 lightclaw ==========
CURRENT_USER="$(id -un)"
if [ "$CURRENT_USER" = "$LIGHTCLAW_USER" ]; then
    "$LIGHTCLAW_BIN" stop >/dev/null 2>&1 || true
else
    su - "$LIGHTCLAW_USER" -c "$LIGHTCLAW_BIN stop" >/dev/null 2>&1 || true
fi
sleep 1

# ========== 4. 启动 lightclaw（使用管理后台分配的端口） ==========
# 用 setsid 创建新 session，避免 su 退出时后台进程被 SIGHUP 杀掉
# setsid 属于 util-linux，Ubuntu / TencentOS 均自带，兼容 dash/bash
if [ "$CURRENT_USER" = "$LIGHTCLAW_USER" ]; then
    setsid "$LIGHTCLAW_BIN" start --host 0.0.0.0 --port "$GATEWAY_UI_PORT" </dev/null >/dev/null 2>&1 &
else
    su - "$LIGHTCLAW_USER" -c "setsid $LIGHTCLAW_BIN start --host 0.0.0.0 --port $GATEWAY_UI_PORT </dev/null >/dev/null 2>&1 &"
fi

# ========== 5. 健康检查 ==========
HEALTHY=false
for i in $(seq 1 10); do
    sleep 2
    HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "http://localhost:${GATEWAY_UI_PORT}/" 2>/dev/null || true)
    if [ -n "$HTTP_CODE" ] && [ "$HTTP_CODE" != "000" ]; then
        HEALTHY=true
        break
    fi
done

if [ "$HEALTHY" != "true" ]; then
    echo "lightclaw health check failed after restart" >&2
    exit 1
fi

# ========== 6. 更新 lightclaw-login.txt ==========
NEW_URL="http://${GATEWAY_IP}:${GATEWAY_UI_PORT}"
cat > "$LOGIN_FILE" <<EOF
URL: ${NEW_URL}
Password: ${PASSWORD}
EOF

# ========== 7. 输出 JSON ==========
printf '{"port":%d,"password":"%s"}' "$GATEWAY_UI_PORT" "$PASSWORD"
