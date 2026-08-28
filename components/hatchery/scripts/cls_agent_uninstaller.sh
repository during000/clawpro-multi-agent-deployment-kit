#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 探测可用镜像源：优先 mirrors.tencentyun.com（内网），回退 mirrors.tencent.com（外网）
detect_mirror() {
    local primary="https://mirrors.tencentyun.com"
    local fallback="https://mirrors.tencent.com"
    if curl -sf --connect-timeout 2 --max-time 3 "${primary}/" -o /dev/null 2>/dev/null; then
        echo "$primary"
    elif curl -sf --connect-timeout 2 --max-time 3 "${fallback}/" -o /dev/null 2>/dev/null; then
        echo "$fallback"
    else
        echo "$primary"
    fi
}

MIRROR_BASE=$(detect_mirror)
echo "Using mirror: $MIRROR_BASE"

# Pre-check: skip if loglistener is not running (already uninstalled or never installed)
if ! systemctl is-active --quiet loglistener 2>/dev/null; then
    echo "loglistener service is not running, skipping uninstallation."
    exit 0
fi

# Step 1: Stop loglistener service
systemctl stop loglistener

# Step 2: Uninstall loglistener via operator
# Note: `yes` may exit with SIGPIPE (141) when the downstream process closes stdin,
# which is expected and not an error. We temporarily disable pipefail to handle this.
set +o pipefail
yes | /opt/loglistener/tools/loglistener_operator uninstall
uninstall_exit_code=${PIPESTATUS[1]}
set -o pipefail
if [ "$uninstall_exit_code" -ne 0 ]; then
    echo "==> ERROR: loglistener_operator uninstall failed with exit code $uninstall_exit_code"
    exit "$uninstall_exit_code"
fi

# Step 3: Uninstall CLS diagnostics metrics onboard CLI
npx --yes --registry "${MIRROR_BASE}/npm/" clawpro-diagnostics-metrics-cls-onboard-cli uninstall --force
