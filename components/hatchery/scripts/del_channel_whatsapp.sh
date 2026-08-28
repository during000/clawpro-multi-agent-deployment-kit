#!/usr/bin/env bash
# 断开 WhatsApp 通道并清理凭证
# TAT timeout: 60s
set -euo pipefail

OC_JSON="$HOME/.openclaw/openclaw.json"
AUTH_DIR="$HOME/.openclaw/credentials/whatsapp/default"
TMP="/tmp/openclaw_wa_del_$$"

# 脚本中断时清理临时文件，避免 /tmp 残留
trap 'rm -f "$TMP"' EXIT

# 1. 删除凭证
rm -rf "$AUTH_DIR"

# 2. 从 openclaw.json 删除 channels.whatsapp，并禁用 whatsapp 插件
if [ -f "$OC_JSON" ] && command -v jq &>/dev/null; then
  jq 'del(.channels.whatsapp)
      | .plugins.allow = ((.plugins.allow // []) | map(select(. != "whatsapp")))
      | if (.plugins.entries.whatsapp? != null) then .plugins.entries.whatsapp.enabled = false else . end' \
    "$OC_JSON" > "$TMP" 2>/dev/null && mv "$TMP" "$OC_JSON"
fi

# 3. 重启 gateway
systemctl --user restart openclaw-gateway.service >/dev/null 2>&1 || true

echo '{"action":"finish","success":true,"message":"WhatsApp 通道已断开，凭证已清理"}'