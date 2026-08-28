#!/bin/bash
# remove_soul_hermes.sh — 移除 Hermes SOUL.md 并重启 Gateway 使生效
# 通过 TAT 下发，以 runtime_user 身份执行。
set -euo pipefail

# %INCLUDE% lib_acli_compat.sh

SOUL_FILE="$HOME/.hermes/SOUL.md"
BACKUP_FILE="$HOME/.hermes/SOUL.md.default"

if [ -f "$BACKUP_FILE" ]; then
    mv "$BACKUP_FILE" "$SOUL_FILE"
elif [ -f "$SOUL_FILE" ]; then
    rm -f "$SOUL_FILE"
fi

# ===== 脚本级探测一次 =====
_acli_mode="$(ensure_acli 2>/dev/null)"

# 重启 gateway 使变更生效
if [ "$_acli_mode" = "acli" ]; then
    acli gateway restart 2>/dev/null
    echo "ok"
    exit 0
fi

# fallback: harness gateway restart + systemctl（仅 acli 不可用时）
if command -v harness >/dev/null 2>&1 && harness gateway restart 2>/dev/null; then
    :
else
    for unit in hermes hermes-gateway harness-gateway; do
        if systemctl --user is-enabled "$unit" >/dev/null 2>&1 || systemctl --user is-active "$unit" >/dev/null 2>&1; then
            systemctl --user restart "$unit" 2>/dev/null || true
            break
        fi
    done
fi

echo "ok"
