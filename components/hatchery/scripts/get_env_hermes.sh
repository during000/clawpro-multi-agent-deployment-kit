#!/bin/bash
# get_env_hermes.sh
# 读取 Hermes 当前环境变量。
# 契约（与 scripts/get_env.sh 对齐）：stdout JSON object，未配置时 "{}"。
#
# 实现：优先走 `harness env list`；兜底读 systemd drop-in 文件。

set -uo pipefail
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

LOG_DIR="${HOME}/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/get_env_hermes.log"
exec 2>>"$LOG_FILE"

# ───────── 首选：harness env list ─────────
if command -v harness >/dev/null 2>&1 && harness env --help >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    raw=""
    for args in "--output json" "--format json" "-o json" ""; do
        # shellcheck disable=SC2086
        raw=$(harness env list $args 2>>"$LOG_FILE" || true)
        if [ -n "$raw" ] && echo "$raw" | jq empty >/dev/null 2>&1; then
            break
        fi
        raw=""
    done
    if [ -n "$raw" ]; then
        # 兼容 object 直出 / {env: {...}} / {items: [{key,value}]} 等几种形状
        normalized=$(echo "$raw" | jq -c '
            if type == "object" and (.env // false) then .env
            elif type == "object" and (.items // false) then
                (.items | map({key: (.key // .name), value: (.value // .val)}) | from_entries)
            elif type == "object" then .
            elif type == "array" then
                (. | map({key: (.key // .name), value: (.value // .val)}) | from_entries)
            else {}
            end
        ' 2>>"$LOG_FILE" || true)
        if [ -n "$normalized" ] && [ "$normalized" != "null" ]; then
            echo "$normalized"
            exit 0
        fi
    fi
fi

# ───────── 兜底：systemd user drop-in ─────────
ENV_FILE="$HOME/.config/systemd/user/hermes.service.d/.hermes_env"
if [ ! -f "$ENV_FILE" ]; then
    echo "{}"
    exit 0
fi

if command -v jq >/dev/null 2>&1; then
    RESULT='{}'
    while IFS= read -r line || [ -n "$line" ]; do
        [[ -z "$line" || "$line" == \#* ]] && continue
        key="${line%%=*}"
        val="${line#*=}"
        RESULT=$(echo "$RESULT" | jq --arg k "$key" --arg v "$val" '.[$k] = $v')
    done < "$ENV_FILE"
    echo "$RESULT"
else
    # jq 缺失的降级（不推荐，但兜底）
    out="{"
    first=1
    while IFS= read -r line || [ -n "$line" ]; do
        [[ -z "$line" || "$line" == \#* ]] && continue
        key="${line%%=*}"
        val="${line#*=}"
        safe_val=$(printf '%s' "$val" | sed 's/\\/\\\\/g; s/"/\\"/g')
        if [ "$first" -eq 1 ]; then
            out="${out}\"${key}\":\"${safe_val}\""
            first=0
        else
            out="${out},\"${key}\":\"${safe_val}\""
        fi
    done < "$ENV_FILE"
    echo "${out}}"
fi
