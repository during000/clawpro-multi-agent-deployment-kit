#!/bin/bash
set -euo pipefail

export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"

if command -v acli >/dev/null 2>&1; then
    acli gateway restart
    echo "✓ acli gateway restart 完成"
    exit 0
fi

if command -v harness >/dev/null 2>&1; then
    harness gateway restart
    echo "✓ harness gateway restart 完成"
    exit 0
fi

for unit in hermes hermes-gateway harness-gateway; do
    if systemctl --user restart "$unit" 2>/dev/null; then
        echo "✓ systemctl --user restart $unit 完成"
        exit 0
    fi
done

echo "hermes gateway restart failed"
exit 1
