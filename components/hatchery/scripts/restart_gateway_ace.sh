#!/bin/bash
set -euo pipefail

export PATH="$HOME/.lightclaw/bin:$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

if command -v lightclaw >/dev/null 2>&1; then
    lightclaw restart
    echo "✓ lightclaw 已重启"
    exit 0
fi

systemctl restart lightclaw
echo "✓ systemctl restart lightclaw 完成"
