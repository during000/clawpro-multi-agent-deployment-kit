#!/bin/bash
# remove_soul.sh — 移除 OpenClaw SOUL.md 并重启 Gateway 使生效
# 通过 TAT 下发，以 runtime_user 身份执行。
set -euo pipefail

SOUL_FILE="$HOME/.openclaw/workspace/SOUL.md"
BACKUP_FILE="$HOME/.openclaw/workspace/SOUL.md.default"

if [ -f "$BACKUP_FILE" ]; then
    # 恢复备份
    mv "$BACKUP_FILE" "$SOUL_FILE"
elif [ -f "$SOUL_FILE" ]; then
    # 无备份则删除
    rm -f "$SOUL_FILE"
fi

# 重启 gateway 使变更生效
systemctl --user restart openclaw-gateway || true

echo "ok"
