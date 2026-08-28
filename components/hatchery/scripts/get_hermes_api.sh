#!/bin/bash
set -uo pipefail

# %INCLUDE% lib_acli_compat.sh

# ========== 获取 Hermes API Server 连接信息 ==========
# 功能：确保 Hermes .env 中 API Server 配置完整，配置变更或端口不可达时重启 gateway。
# 返回端口和 API Key 供 Hatchery 拼接连接地址。
#
# 输出（最后一行 JSON，通过 fd 3）：
#   {"port": 8642, "key": "xxxxxx"}
# 失败时输出：{"error": "错误描述"}

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.hermes/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="get_hermes_api"
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

echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "=== 获取 Hermes API Server 连接信息 ==="

# ===== 脚本级探测一次，后续复用 =====
_acli_mode="$(ensure_acli 2>>"$LOG_FILE")"

# ========== 加载环境 ==========
HERMES_HOME="${HERMES_HOME:-$HOME/.hermes}"
ENV_FILE="$HERMES_HOME/.env"
echo "HERMES_HOME: $HERMES_HOME"
echo "ENV_FILE: $ENV_FILE"

# ========== 辅助函数 ==========
generate_key() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 16
  else
    head -c 16 /dev/urandom | xxd -p 2>/dev/null || cat /proc/sys/kernel/random/uuid | tr -d '-'
  fi
}

# 安全读取 .env 中的值，文件不存在或 key 不存在时返回空字符串
read_env() {
  local key="$1"
  if [ ! -f "$ENV_FILE" ]; then
    echo ""
    return 0
  fi
  grep -E "^${key}=" "$ENV_FILE" 2>/dev/null | tail -1 | cut -d= -f2- | tr -d "'\"" || echo ""
}

tcp_probe() {
  if command -v timeout >/dev/null 2>&1; then
    timeout 1 bash -c "exec 9<>/dev/tcp/$1/$2" 2>/dev/null
  else
    ( exec 9<>"/dev/tcp/$1/$2" ) 2>/dev/null
  fi
}

# ========== [步骤 1/2] 读取/确保 API Server 配置 ==========
echo ""
echo ">>> [步骤 1/3] 读取 API Server 配置"

API_ENABLED=$(read_env "API_SERVER_ENABLED")
API_PORT=$(read_env "API_SERVER_PORT")
API_KEY=$(read_env "API_SERVER_KEY")
API_HOST=$(read_env "API_SERVER_HOST")

echo "当前配置: ENABLED=$API_ENABLED PORT=$API_PORT HOST=$API_HOST KEY=${API_KEY:+***已设置}"

# 确保 .env 文件存在
if [ ! -f "$ENV_FILE" ]; then
    mkdir -p "$HERMES_HOME"
    touch "$ENV_FILE"
    echo "✓ 创建 .env 文件"
fi

CHANGED=false

# 确保 API_SERVER_ENABLED=true
if [ "$API_ENABLED" != "true" ]; then
    sed -i '/^API_SERVER_ENABLED=/d' "$ENV_FILE"
    echo "API_SERVER_ENABLED=true" >> "$ENV_FILE"
    echo "✓ 设置 API_SERVER_ENABLED=true"
    CHANGED=true
fi

# 确保 API_SERVER_HOST=0.0.0.0
if [ "$API_HOST" != "0.0.0.0" ]; then
    sed -i '/^API_SERVER_HOST=/d' "$ENV_FILE"
    echo "API_SERVER_HOST=0.0.0.0" >> "$ENV_FILE"
    echo "✓ 设置 API_SERVER_HOST=0.0.0.0"
    CHANGED=true
fi

# 确保有 API_SERVER_PORT
if [ -z "$API_PORT" ] || [ "$API_PORT" = "0" ]; then
    API_PORT="8642"
    sed -i '/^API_SERVER_PORT=/d' "$ENV_FILE"
    echo "API_SERVER_PORT=$API_PORT" >> "$ENV_FILE"
    echo "✓ 设置 API_SERVER_PORT=$API_PORT"
    CHANGED=true
fi

# 确保有 API_SERVER_KEY
if [ -z "$API_KEY" ]; then
    API_KEY=$(generate_key)
    sed -i '/^API_SERVER_KEY=/d' "$ENV_FILE"
    echo "API_SERVER_KEY=$API_KEY" >> "$ENV_FILE"
    echo "✓ 生成并设置 API_SERVER_KEY"
    CHANGED=true
fi

if [ "$CHANGED" = "true" ]; then
    echo "⚠ 配置已变更，需要重启 gateway"
fi

# ========== [步骤 2/3] 确保 gateway 运行且端口可达 ==========
echo ""
echo ">>> [步骤 2/3] 确保 Gateway 运行中"

NEED_RESTART=false
if [ "$CHANGED" = "true" ]; then
    NEED_RESTART=true
elif ! tcp_probe 127.0.0.1 "$API_PORT"; then
    echo "⚠ API Server 端口 $API_PORT 不可达，需要重启"
    NEED_RESTART=true
else
    echo "✓ API Server 端口 $API_PORT 可达，无需重启"
fi

if [ "$NEED_RESTART" = "true" ]; then
    echo "重启 hermes gateway..."

    if [ "$_acli_mode" = "acli" ]; then
        # acli 路径：失败直接报错退出
        acli gateway restart >>"$LOG_FILE" 2>&1 || _fatal "acli gateway restart 失败"
        echo "✓ acli gateway restart 完成"
    else
        # fallback: harness → systemd → 手动启动
        if command -v harness >/dev/null 2>&1; then
            if harness gateway restart >>"$LOG_FILE" 2>&1; then
                echo "✓ harness gateway restart 完成"
            else
                echo "⚠ harness gateway restart 失败，尝试 systemd"
                RESTARTED=false
                for unit in hermes hermes-gateway harness-gateway; do
                    if systemctl --user restart "$unit" 2>/dev/null; then
                        echo "✓ systemctl --user restart $unit 完成"
                        RESTARTED=true
                        break
                    fi
                done
                if [ "$RESTARTED" != "true" ]; then
                    echo "⚠ systemd 重启也失败，尝试手动重启"
                    HERMES_PID=$(pgrep -u "$(id -un)" -f "hermes_cli.main gateway" 2>/dev/null | head -1 || true)
                    if [ -n "$HERMES_PID" ]; then
                        kill "$HERMES_PID" 2>/dev/null || true
                        for i in $(seq 1 20); do
                            kill -0 "$HERMES_PID" 2>/dev/null || break
                            sleep 0.5
                        done
                    fi
                    if [ -x "$HERMES_HOME/hermes-agent/venv/bin/python" ]; then
                        setsid "$HERMES_HOME/hermes-agent/venv/bin/python" -m hermes_cli.main gateway run --replace \
                            </dev/null >>"$LOG_DIR/hermes_gateway.log" 2>&1 &
                        echo "✓ 手动启动 gateway (venv python)"
                    else
                        HERMES_BIN=""
                        command -v hermes >/dev/null 2>&1 && HERMES_BIN="$(command -v hermes)"
                        [ -z "$HERMES_BIN" ] && [ -x "$HOME/.local/bin/hermes" ] && HERMES_BIN="$HOME/.local/bin/hermes"
                        if [ -n "$HERMES_BIN" ]; then
                            setsid "$HERMES_BIN" gateway run --replace \
                                </dev/null >>"$LOG_DIR/hermes_gateway.log" 2>&1 &
                            echo "✓ 手动启动 gateway ($HERMES_BIN)"
                        else
                            _fatal "hermes 命令未找到，无法重启 gateway"
                        fi
                    fi
                fi
            fi
        else
            RESTARTED=false
            for unit in hermes hermes-gateway harness-gateway; do
                if systemctl --user restart "$unit" 2>/dev/null; then
                    echo "✓ systemctl --user restart $unit 完成"
                    RESTARTED=true
                    break
                fi
            done
            if [ "$RESTARTED" != "true" ]; then
                HERMES_PID=$(pgrep -u "$(id -un)" -f "hermes_cli.main gateway" 2>/dev/null | head -1 || true)
                if [ -n "$HERMES_PID" ]; then
                    kill "$HERMES_PID" 2>/dev/null || true
                    for i in $(seq 1 20); do
                        kill -0 "$HERMES_PID" 2>/dev/null || break
                        sleep 0.5
                    done
                fi
                if [ -x "$HERMES_HOME/hermes-agent/venv/bin/python" ]; then
                    setsid "$HERMES_HOME/hermes-agent/venv/bin/python" -m hermes_cli.main gateway run --replace \
                        </dev/null >>"$LOG_DIR/hermes_gateway.log" 2>&1 &
                    echo "✓ 手动启动 gateway (venv python)"
                else
                    HERMES_BIN=""
                    command -v hermes >/dev/null 2>&1 && HERMES_BIN="$(command -v hermes)"
                    [ -z "$HERMES_BIN" ] && [ -x "$HOME/.local/bin/hermes" ] && HERMES_BIN="$HOME/.local/bin/hermes"
                    if [ -n "$HERMES_BIN" ]; then
                        setsid "$HERMES_BIN" gateway run --replace \
                            </dev/null >>"$LOG_DIR/hermes_gateway.log" 2>&1 &
                        echo "✓ 手动启动 gateway ($HERMES_BIN)"
                    else
                        _fatal "hermes 命令未找到，无法重启 gateway"
                    fi
                fi
            fi
        fi
    fi

    # 等待端口就绪
    echo "等待 API Server 端口 $API_PORT 就绪..."
    HEALTHY=false
    for i in $(seq 1 60); do
        if tcp_probe 127.0.0.1 "$API_PORT"; then
            HEALTHY=true
            break
        fi
        sleep 1
    done

    if [ "$HEALTHY" != "true" ]; then
        _fatal "API Server 端口 $API_PORT 启动超时（等待 60 秒），请检查 $LOG_DIR/hermes_gateway.log"
    fi
    echo "✓ Gateway 重启完成，端口 $API_PORT 就绪（等待 ${i} 秒）"
fi

# ========== [步骤 3/3] 输出结果 ==========
echo ""
echo ">>> [步骤 3/3] 输出连接信息"

# 最终重新读取确保一致
API_PORT=$(read_env "API_SERVER_PORT")
API_KEY=$(read_env "API_SERVER_KEY")

if [ -z "$API_PORT" ] || [ -z "$API_KEY" ]; then
    _fatal "读取 API Server 配置失败"
fi

RESULT_JSON=$(printf '{"port":%s,"key":"%s"}' "$API_PORT" "$API_KEY")
echo "$RESULT_JSON" >> "$LOG_FILE"
printf '%s\n' "$RESULT_JSON" >&3
_JSON_EMITTED=1

echo ""
echo "=== 获取完成 ==="
echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="

trap - EXIT ERR
