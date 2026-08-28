#!/bin/bash
set -euo pipefail

ENV_FILE="$HOME/.config/systemd/user/openclaw-gateway.service.d/.openclaw_env"

if [ ! -f "$ENV_FILE" ]; then
    echo '{}'
    exit 0
fi

# 解析 KEY=VALUE 行，输出 JSON 对象
# 使用 read 循环确保 value 中的 = 不被截断
RESULT='{}'
while IFS= read -r line || [ -n "$line" ]; do
    [[ -z "$line" || "$line" == \#* ]] && continue
    key="${line%%=*}"
    val="${line#*=}"
    RESULT=$(echo "$RESULT" | jq --arg k "$key" --arg v "$val" '.[$k] = $v')
done < "$ENV_FILE"

echo "$RESULT"
