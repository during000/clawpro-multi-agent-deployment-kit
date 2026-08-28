#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

openclaw status --json  2>/dev/null | sed -n '/^{/,$p' | jq '{gateway, update, channelSummary}'
