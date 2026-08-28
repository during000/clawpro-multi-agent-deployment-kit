#!/bin/bash
# remove_soul_ace.sh — 移除 Lightclaw ACE SOUL.md 并重启使生效
# 通过 TAT 下发，以 runtime_user 身份执行。
set -euo pipefail

SOUL_FILE="$HOME/.lightclaw/workspace/SOUL.md"
BACKUP_FILE="$HOME/.lightclaw/workspace/SOUL.md.default"

if [ -f "$BACKUP_FILE" ]; then
    mv "$BACKUP_FILE" "$SOUL_FILE"
elif [ -f "$SOUL_FILE" ]; then
    rm -f "$SOUL_FILE"
fi

# 重启 lightclaw 使变更生效
systemctl restart lightclaw 2>/dev/null || true

echo "ok"
