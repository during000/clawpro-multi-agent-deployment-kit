#!/bin/bash
# get_env_ace.sh
# 读取 LightClaw ACE 当前所有环境变量。
# 契约（与 scripts/get_env.sh 对齐）：stdout 末行输出 JSON object：
#   {"KEY1": "value1", "KEY2": "value2"}
# 未配置任何 env 时输出 "{}"。
#
# 实现：解析 `lightclaw env list` 表格输出：
#     Key                             Value
#     ────────────────────────────────────────────
#     KEY1                            VALUE1
#     KEY2
#     KEY3                            VALUE3
#
# 说明：无值的 key 跳过（value 为空）；value 中不含空格（命令不支持）。

set -uo pipefail
export PATH="$HOME/.lightclaw/bin:$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u 2>/dev/null || echo 0)

LOG_DIR="${HOME}/.lightclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/get_env_ace.log"
exec 2>>"$LOG_FILE"

if ! command -v lightclaw >/dev/null 2>&1; then
    echo "{}"
    exit 0
fi

raw=$(lightclaw env list 2>>"$LOG_FILE" || true)
if [ -z "$raw" ]; then
    echo "{}"
    exit 0
fi

# 解析表格行。跳过表头（"Key"开头）、分隔线（─）、空行。
# 每行：第一个字段 = key，第二个及以后字段 = value（以空格合并）。
# 若一行只有 key 没有 value，跳过（表示该变量无值）。
if command -v jq >/dev/null 2>&1; then
    result=$(echo "$raw" | awk '
        # 跳过表头 / 分隔线 / 全空行
        /^[[:space:]]*(Key|─)/ { next }
        /^[[:space:]]*$/       { next }
        {
            # 去掉开头空白
            sub(/^[[:space:]]+/, "")
            # 第一个字段是 key
            key = $1
            # 其余字段合并为 value
            val = ""
            for (i = 2; i <= NF; i++) {
                val = (val == "" ? $i : val " " $i)
            }
            if (key != "" && val != "") {
                print key "\t" val
            }
        }
    ' | jq -Rc --slurp '
        split("\n")
        | map(select(length > 0))
        | map(
            split("\t") | {key: .[0], value: (.[1] // "")}
          )
        | from_entries
    ' 2>>"$LOG_FILE" || echo "{}")
    [ -z "$result" ] && result="{}"
    echo "$result"
else
    # jq 缺失：手拼降级
    out="{"
    first=1
    while IFS= read -r line; do
        line_trimmed="$(echo "$line" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
        [[ -z "$line_trimmed" || "$line_trimmed" == Key* || "$line_trimmed" == ─* ]] && continue
        key=$(echo "$line_trimmed" | awk '{print $1}')
        val=$(echo "$line_trimmed" | awk '{for(i=2;i<=NF;i++) printf "%s%s",$i,(i==NF?"":" "); print ""}')
        [ -z "$key" ] && continue
        [ -z "$val" ] && continue
        safe_val=$(printf '%s' "$val" | sed 's/\\/\\\\/g; s/"/\\"/g')
        if [ "$first" -eq 1 ]; then
            out="${out}\"${key}\":\"${safe_val}\""
            first=0
        else
            out="${out},\"${key}\":\"${safe_val}\""
        fi
    done <<EOL
$raw
EOL
    echo "${out}}"
fi
