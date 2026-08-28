#!/bin/bash
# set_soul_ace.sh — 写入 Lightclaw ACE SOUL.md 并重启使生效
# 通过 TAT 下发，以 runtime_user 身份执行。
# Parameters (substituted by TAT before execution):
#   {{soul_b64}}  - base64 编码的 SOUL.md 内容
set -euo pipefail

mkdir -p "$HOME/.lightclaw/workspace"

SOUL_FILE="$HOME/.lightclaw/workspace/SOUL.md"
BACKUP_FILE="$HOME/.lightclaw/workspace/SOUL.md.default"

# 备份已有 SOUL.md
if [ -f "$SOUL_FILE" ]; then
    cp "$SOUL_FILE" "$BACKUP_FILE"
fi

# 解码 base64 内容并写入
printf '%s' '{{soul_b64}}' | base64 -d > "$SOUL_FILE"

# 重启 lightclaw 使 SOUL.md 生效
systemctl restart lightclaw 2>/dev/null || true

echo "ok"
