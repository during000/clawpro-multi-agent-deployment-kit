#!/bin/bash
# set_env_ace.sh
# 为 LightClaw ACE 批量设置/删除环境变量。
# 契约（与 scripts/set_env.sh 对齐）：
#   - 入参：{{env_json}} - JSON object，string 值=设置，null 值=删除
#   - 成功 stdout 末行：  "ok"
#   - 失败：非零 exit code + stderr
#
# 实现：调用 `lightclaw env set KEY VALUE` 和 `lightclaw env delete KEY`。
# 注意 lightclaw 的 env 最终由 lightclaw.service 使用，设置后需 `lightclaw restart`。

set -euo pipefail
export PATH="$HOME/.lightclaw/bin:$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u 2>/dev/null || echo 0)

LOG_DIR="${HOME}/.lightclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/set_env_ace.log"
echo "========== $(date '+%Y-%m-%d %H:%M:%S') set_env_ace ==========" >>"$LOG_FILE"

ENV_JSON='{{env_json}}'

# 使用文件锁避免并发重启服务触发 systemd start rate limit
LOCK_FILE="/tmp/.ace_set_env.lock"
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
    echo "环境变量设置操作正在进行中，请勿重复提交" >&2
    exit 1
fi

if ! command -v lightclaw >/dev/null 2>&1; then
    echo "lightclaw command not found" >&2
    exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
    echo "jq not found" >&2
    exit 1
fi

# 校验入参是合法 JSON object
if ! echo "$ENV_JSON" | jq 'type == "object"' 2>>"$LOG_FILE" | grep -q "true"; then
    echo "env_json must be a JSON object" >&2
    exit 1
fi

# 遍历 key-value：string 值 → set；null 值 → delete
# 使用 jq -r 输出便于 bash 循环
#   格式："KEY\tACTION\tVALUE"
while IFS=$'\t' read -r key action val; do
    [ -z "$key" ] && continue
    case "$action" in
        set)
            echo "[env] set $key" >>"$LOG_FILE"
            if ! lightclaw env set "$key" "$val" >>"$LOG_FILE" 2>&1; then
                echo "lightclaw env set $key failed" >&2
                exit 1
            fi
            ;;
        delete)
            echo "[env] delete $key" >>"$LOG_FILE"
            # delete 失败不致命（可能 key 本就不存在）
            lightclaw env delete "$key" >>"$LOG_FILE" 2>&1 || true
            ;;
    esac
done < <(echo "$ENV_JSON" | jq -r '
    to_entries[]
    | if .value == null then
        [.key, "delete", ""]
      else
        [.key, "set", (.value | tostring)]
      end
    | @tsv
')

# 重启服务让 env 生效
echo "[env] restart lightclaw to apply env changes" >>"$LOG_FILE"
if ! lightclaw restart >>"$LOG_FILE" 2>&1; then
    echo "[env] lightclaw restart returned non-zero, but env changes saved" >>"$LOG_FILE"
fi

echo "ok"
