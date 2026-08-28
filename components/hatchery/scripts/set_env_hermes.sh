#!/bin/bash
# set_env_hermes.sh
# 为 Hermes 批量设置/删除环境变量。
# 契约（与 scripts/set_env.sh 对齐）：
#   - 入参 {{env_json}} - JSON object
#   - stdout "ok" / 非零 exit code + stderr
#
# 实现（双路径）：
#   - 首选：`harness env set KEY VALUE` / `harness env delete KEY`（若 harness CLI 支持）
#   - 兜底：直接写 systemd user drop-in（参考 openclaw set_env.sh 的做法，针对 hermes 的 unit）

set -euo pipefail
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u 2>/dev/null || echo 0)

LOG_DIR="${HOME}/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/set_env_hermes.log"
echo "========== $(date '+%Y-%m-%d %H:%M:%S') set_env_hermes ==========" >>"$LOG_FILE"

ENV_JSON='{{env_json}}'

# 使用文件锁避免并发重启服务触发 systemd start rate limit
LOCK_FILE="/tmp/.hermes_set_env.lock"
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
    echo "环境变量设置操作正在进行中，请勿重复提交" >&2
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    echo "jq not found" >&2
    exit 1
fi
if ! echo "$ENV_JSON" | jq 'type == "object"' 2>>"$LOG_FILE" | grep -q "true"; then
    echo "env_json must be a JSON object" >&2
    exit 1
fi

# ───────── 首选路径：harness env set/delete ─────────
if command -v harness >/dev/null 2>&1; then
    # 探测 `harness env` 子命令是否存在
    if harness env --help >/dev/null 2>&1; then
        echo "[env] using harness CLI" >>"$LOG_FILE"
        ok=1
        while IFS=$'\t' read -r key action val; do
            [ -z "$key" ] && continue
            case "$action" in
                set)
                    echo "[env] harness env set $key" >>"$LOG_FILE"
                    if ! harness env set "$key" "$val" >>"$LOG_FILE" 2>&1; then
                        ok=0
                        echo "harness env set $key failed" >&2
                        break
                    fi
                    ;;
                delete)
                    echo "[env] harness env delete $key" >>"$LOG_FILE"
                    harness env delete "$key" >>"$LOG_FILE" 2>&1 || true
                    ;;
            esac
        done < <(echo "$ENV_JSON" | jq -r '
            to_entries[]
            | if .value == null then [.key, "delete", ""]
              else [.key, "set", (.value | tostring)]
              end
            | @tsv
        ')
        if [ "$ok" -eq 1 ]; then
            # 尝试重启 hermes 让 env 生效
            harness gateway restart >>"$LOG_FILE" 2>&1 || true
            echo "ok"
            exit 0
        fi
        echo "[env] harness env path failed, falling back to systemd drop-in" >>"$LOG_FILE"
    fi
fi

# ───────── 兜底路径：写 systemd user drop-in ─────────
DROPIN_DIR="$HOME/.config/systemd/user/hermes.service.d"
ENV_FILE="$DROPIN_DIR/.hermes_env"
DROPIN_FILE="$DROPIN_DIR/env.conf"

CURRENT='{}'
if [ -f "$ENV_FILE" ]; then
    while IFS= read -r line || [ -n "$line" ]; do
        [[ -z "$line" || "$line" == \#* ]] && continue
        key="${line%%=*}"
        val="${line#*=}"
        CURRENT=$(echo "$CURRENT" | jq --arg k "$key" --arg v "$val" '.[$k] = $v')
    done < "$ENV_FILE"
fi

MERGED=$(echo "$CURRENT" | jq --argjson input "$ENV_JSON" '
    reduce ($input | to_entries[]) as $e (.;
        if $e.value == null then del(.[$e.key])
        else .[$e.key] = $e.value
        end
    )
')

mkdir -p "$DROPIN_DIR"
echo "$MERGED" | jq -r 'to_entries[] | "\(.key)=\(.value)"' > "$ENV_FILE"

if [ ! -f "$DROPIN_FILE" ]; then
    cat > "$DROPIN_FILE" << CONF
[Service]
EnvironmentFile=-${HOME}/.config/systemd/user/hermes.service.d/.hermes_env
CONF
fi

systemctl --user daemon-reload >>"$LOG_FILE" 2>&1 || true
# 尝试重启多个可能的 hermes unit 名
for unit in hermes hermes-gateway harness-gateway; do
    systemctl --user restart "$unit" >>"$LOG_FILE" 2>&1 && break || true
done

echo "ok"
