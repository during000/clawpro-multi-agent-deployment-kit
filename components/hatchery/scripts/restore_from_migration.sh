#!/bin/bash
set -euo pipefail

RUNTIME_USER="{{runtime_user}}"
if [ -z "$RUNTIME_USER" ] || [ "$RUNTIME_USER" = "{{runtime_user}}" ]; then
    RUNTIME_USER="${OPENCLAW_RUNTIME_USER:-root}"
fi

if [ "$RUNTIME_USER" != "root" ] && ! id "$RUNTIME_USER" >/dev/null 2>&1; then
    echo "WARN: runtime user '$RUNTIME_USER' 不存在，回退到 root"
    RUNTIME_USER="root"
fi

TARGET_UID="$(id -u "$RUNTIME_USER" 2>/dev/null || id -u)"
TARGET_GID="$(id -g "$RUNTIME_USER" 2>/dev/null || id -g)"
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

echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

ARCHIVE_URL="{{url}}"

if [ -z "$ARCHIVE_URL" ] || [ "$ARCHIVE_URL" = "{{url}}" ]; then
    echo "✗ url 参数未注入"; exit 1
fi

AGENT_TYPE="{{agent_type}}"
PRESERVED_PATHS="{{preserved_paths}}"

# 从备份恢复平台相关目录（打包时已排除，目标实例安装时自带，解压后需恢复本机版本）
restore_preserved_from_backup() {
    [ -z "$PRESERVED_PATHS" ] && return
    for p in $PRESERVED_PATHS; do
        if [ -d "$BACKUP_PATH/$p" ] && [ ! -e "$AGENT_HOME/$p" ]; then
            echo ">>> 从备份恢复平台目录: $p"
            cp -a "$BACKUP_PATH/$p" "$AGENT_HOME/$p"
        fi
    done
}

# 根据 agent type 确定 agent 目录
case "$AGENT_TYPE" in
    hermes)       AGENT_HOME="$TARGET_HOME/.hermes" ;;
    lightclawace) AGENT_HOME="$TARGET_HOME/.lightclaw" ;;
    *)            AGENT_HOME="$TARGET_HOME/.openclaw" ;;  # openclaw 及其他默认
esac

if [ ! -d "$AGENT_HOME" ]; then
    echo "✗ agent 目录不存在: $AGENT_HOME（agent_type=$AGENT_TYPE）"; exit 1
fi
echo "目标 agent 目录: $AGENT_HOME"

# agent_stop / agent_start：按 agent_type 分派正确的启停命令
agent_stop() {
    case "$AGENT_TYPE" in
        lightclawace)
            run_as_runtime_user lightclaw stop 2>/dev/null || true
            ;;
        hermes)
            run_as_runtime_user hermes gateway stop 2>/dev/null || true
            ;;
        *)
            run_as_runtime_user openclaw gateway stop 2>/dev/null || true
            ;;
    esac
}

agent_start() {
    case "$AGENT_TYPE" in
        lightclawace)
            run_as_runtime_user lightclaw restart 2>/dev/null || true
            ;;
        hermes)
            run_as_runtime_user hermes gateway restart 2>/dev/null || true
            ;;
        *)
            run_as_runtime_user openclaw gateway restart 2>/dev/null || true
            ;;
    esac
}

# 停止 gateway
echo ">>> 停止 agent gateway..."
agent_stop
sleep 2

# 备份现有目录（mv，秒级完成）
BACKUP_PATH="/tmp/agent-backup-$(date +%Y%m%d_%H%M%S)"
echo "PROGRESS:backing_up"
echo ">>> 移动现有 agent 目录到 $BACKUP_PATH ..."
mv "$AGENT_HOME" "$BACKUP_PATH" || true
echo "BACKUP_PATH:$BACKUP_PATH"
mkdir -p "$AGENT_HOME"

# trap：失败时从备份恢复，并重启 gateway
_RESTORE_DONE=0
cleanup_on_error() {
    if [ "$_RESTORE_DONE" -eq 1 ]; then
        return
    fi
    echo "✗ 迁移失败，尝试从备份恢复..."
    if [ -d "$BACKUP_PATH" ]; then
        rm -rf "$AGENT_HOME" 2>/dev/null || true
        mv "$BACKUP_PATH" "$AGENT_HOME" 2>/dev/null && \
            chown -R "${TARGET_UID}:${TARGET_GID}" "$AGENT_HOME" 2>/dev/null || true
        echo "✓ 已从备份恢复: $BACKUP_PATH"
    fi
    agent_start
}
trap cleanup_on_error ERR

# 下载迁移包
ARCHIVE_PATH="/tmp/agent-migration-$$.tgz"
echo "PROGRESS:downloading"
echo ">>> 下载迁移包..."
if ! curl -fsSL -L -o "$ARCHIVE_PATH" "$ARCHIVE_URL"; then
    echo "✗ 下载失败: $ARCHIVE_URL"; exit 1
fi
echo "✓ 下载完成 ($(du -sh "$ARCHIVE_PATH" | cut -f1))"

# 验证压缩包
echo ">>> 验证压缩包..."
if ! tar tf "$ARCHIVE_PATH" >/dev/null; then
    echo "✗ 压缩包损坏或格式不正确"; exit 1
fi
echo "✓ 压缩包验证通过"

# 解压覆盖
echo "PROGRESS:extracting"
echo ">>> 解压覆盖到 $AGENT_HOME ..."
tar xf "$ARCHIVE_PATH" -C "$AGENT_HOME" --strip-components=1
echo "✓ 解压完成"

# 从备份恢复平台相关目录（保留目标机器上的本机版本）
restore_preserved_from_backup

# 修复权限
echo ">>> 修复权限..."
chown -R "${TARGET_UID}:${TARGET_GID}" "$AGENT_HOME" || true
chmod 755 "$AGENT_HOME"
if [ -d "$AGENT_HOME/credentials" ]; then
    chmod 700 "$AGENT_HOME/credentials"
fi
echo "✓ 权限修复完成"

# 修复 openclaw.json 的 gateway.port：迁移包来自源实例，其端口可能与本机 systemd service 不一致。
# 以本机 systemd service 文件中的 --port 参数为准覆盖，确保 health check 和 CLI 使用同一端口。
CONFIG_JSON="$AGENT_HOME/openclaw.json"
SERVICE_FILE="$TARGET_HOME/.config/systemd/user/openclaw-gateway.service"
if [ -f "$SERVICE_FILE" ] && [ -f "$CONFIG_JSON" ] && command -v jq >/dev/null 2>&1; then
    ACTUAL_PORT=$(grep -oP '(?<=--port )\d+' "$SERVICE_FILE" 2>/dev/null | head -1 || true)
    if [ -n "$ACTUAL_PORT" ]; then
        TMP=$(mktemp)
        jq --argjson p "$ACTUAL_PORT" '.gateway.port = $p' "$CONFIG_JSON" > "$TMP" && mv "$TMP" "$CONFIG_JSON"
        chown "${TARGET_UID}:${TARGET_GID}" "$CONFIG_JSON" 2>/dev/null || true
        echo "✓ openclaw.json gateway.port 已修正为 $ACTUAL_PORT（来自 systemd service）"
    fi
fi

# 清理临时文件
rm -f "$ARCHIVE_PATH"

# 重启 gateway
echo "PROGRESS:restarting"
echo ">>> 重启 agent gateway..."
agent_start

_RESTORE_DONE=1
echo "RESTORE_DONE:1"
echo "========== 迁移恢复完成: $(date '+%Y-%m-%d %H:%M:%S') =========="
