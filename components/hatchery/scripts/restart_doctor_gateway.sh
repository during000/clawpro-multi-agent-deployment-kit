#!/bin/bash
# 重启龙虾医生实例上的 Gateway，同步等待就绪（最多 30 秒）。
set -e

# 加载 openclaw 运行环境
export PNPM_HOME="${PNPM_HOME:-$HOME/.local/share/pnpm}"
export PATH="$PNPM_HOME:$HOME/.npm-global/bin:$PATH"

# 获取实际运行 Gateway 的用户 UID
RUNTIME_UID=$(id -u)
export XDG_RUNTIME_DIR="/run/user/${RUNTIME_UID}"

# 如果以 root 身份运行 systemctl --user，需确保 DBUS_SESSION_BUS_ADDRESS 可用
if [ "$RUNTIME_UID" = "0" ]; then
    export DBUS_SESSION_BUS_ADDRESS="unix:path=/run/user/0/bus"
fi

systemctl --user restart openclaw-gateway 2>&1 || true

# 等待 Gateway 重启完成
for i in $(seq 1 30); do
  sleep 1
  if openclaw gateway status --json 2>/dev/null | grep -q '"running"'; then
    echo "READY"
    exit 0
  fi
done

echo "TIMEOUT"
exit 1
