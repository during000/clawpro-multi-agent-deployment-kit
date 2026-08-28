#!/bin/bash
# set_soul.sh — 写入 OpenClaw SOUL.md 并重启 Gateway 使生效
# 通过 TAT 下发，以 runtime_user 身份执行。
# Parameters (substituted by TAT before execution):
#   {{soul_b64}}  - base64 编码的 SOUL.md 内容
set -euo pipefail

mkdir -p "$HOME/.openclaw/workspace"

SOUL_FILE="$HOME/.openclaw/workspace/SOUL.md"
BACKUP_FILE="$HOME/.openclaw/workspace/SOUL.md.default"

# 备份已有 SOUL.md
if [ -f "$SOUL_FILE" ]; then
    cp "$SOUL_FILE" "$BACKUP_FILE"
fi

# 解码 base64 内容并写入
printf '%s' '{{soul_b64}}' | base64 -d > "$SOUL_FILE"

# 重启 gateway 使 SOUL.md 生效
systemctl --user restart openclaw-gateway || true

echo "ok"
