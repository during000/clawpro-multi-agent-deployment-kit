#!/bin/bash
set -euo pipefail

# uninstall_plugin.sh
# 从 OpenClaw 实例上卸载插件（删除插件目录 + 清理配置）
#
# 参数（通过 TAT 模板变量注入）：
#   {{plugin_slug}}   - 插件 slug（Hatchery 管理标识）
#   {{plugin_id}}     - openclaw.plugin.json 中的 id
#   {{plugin_kind}}   - 插件类型（memory / context-engine / 空）

PLUGIN_SLUG="{{plugin_slug}}"
PLUGIN_ID="{{plugin_id}}"
PLUGIN_KIND="{{plugin_kind}}"
EXTENSIONS_DIR="$HOME/.openclaw/extensions"
CONFIG_FILE="$HOME/.openclaw/openclaw.json"

# 安装目录编码（对齐 encodePluginInstallDirName 和 install 脚本）
if echo "$PLUGIN_ID" | grep -q '/'; then
    ENCODED_DIR="@$(echo -n "$PLUGIN_ID" | md5sum | cut -c1-16)"
else
    ENCODED_DIR="$PLUGIN_ID"
fi
INSTALL_DIR="${EXTENSIONS_DIR}/${ENCODED_DIR}"

# ===== 参数安全校验 =====

if [[ "${PLUGIN_SLUG}" == *"/"* ]] || [[ "${PLUGIN_SLUG}" == *"\\"* ]] || [[ "${PLUGIN_SLUG}" == *".."* ]]; then
    echo "Invalid plugin slug: ${PLUGIN_SLUG}" >&2
    exit 1
fi

if [ -z "${PLUGIN_SLUG}" ]; then
    echo "plugin_slug is empty" >&2
    exit 1
fi

if [[ "${PLUGIN_ID}" == *".."* ]]; then
    echo "Invalid plugin id: ${PLUGIN_ID}" >&2
    exit 1
fi

if [ -z "${PLUGIN_ID}" ]; then
    echo "plugin_id is empty" >&2
    exit 1
fi

echo "Uninstalling plugin: ${PLUGIN_SLUG} (id=${PLUGIN_ID})"

# ===== 删除插件目录 =====

if [ -d "${INSTALL_DIR}" ]; then
    rm -rf "${INSTALL_DIR}"
    echo "Removed plugin directory: ${INSTALL_DIR}"
else
    echo "Plugin directory not found (already removed): ${INSTALL_DIR}"
fi

# 清理可能残留的 staging/backup 目录（install 中途失败时遗留）
for leftover in "${EXTENSIONS_DIR}/.${PLUGIN_SLUG}.staging."* "${EXTENSIONS_DIR}/.${PLUGIN_SLUG}.backup."*; do
    if [ -d "$leftover" ]; then
        rm -rf "$leftover"
        echo "Cleaned up leftover: $leftover"
    fi
done

# ===== 清理 openclaw.json 配置 =====

if [ -f "$CONFIG_FILE" ] && command -v jq >/dev/null 2>&1; then
    OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
    cp "$CONFIG_FILE" "${CONFIG_FILE}.bak.$(date +%Y-%m-%dT%H:%M:%S)"

    # 1. 删除 plugins.entries.{plugin_id}
    jq --arg id "$PLUGIN_ID" '
      if .plugins.entries then
        .plugins.entries |= del(.[$id])
      else . end
    ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"

    # 2. 删除 plugins.installs.{plugin_id}
    OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
    jq --arg id "$PLUGIN_ID" '
      if .plugins.installs then
        .plugins.installs |= del(.[$id])
      else . end
    ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"

    # 3. 从 plugins.allow 中移除
    if jq -e '.plugins.allow | type == "array"' "$CONFIG_FILE" >/dev/null 2>&1; then
        OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
        jq --arg id "$PLUGIN_ID" '
          .plugins.allow = [.plugins.allow[]? | select(. != $id)]
        ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
    fi

    # 4. 清理独占槽位（如果当前占用的是本插件）
    if [ "$PLUGIN_KIND" = "memory" ]; then
        if jq -e --arg id "$PLUGIN_ID" '.plugins.slots.memory == $id' "$CONFIG_FILE" >/dev/null 2>&1; then
            OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
            jq '.plugins.slots.memory = null' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
        fi
    elif [ "$PLUGIN_KIND" = "context-engine" ]; then
        if jq -e --arg id "$PLUGIN_ID" '.plugins.slots.contextEngine == $id' "$CONFIG_FILE" >/dev/null 2>&1; then
            OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
            jq '.plugins.slots.contextEngine = null' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
        fi
    fi

    echo "Cleaned up openclaw.json config"
else
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "Config file not found, skipping config cleanup"
    else
        echo "jq not available, skipping config cleanup (manual cleanup may be needed)"
    fi
fi

# ===== 重启 gateway =====

export XDG_RUNTIME_DIR=/run/user/$(id -u)
if systemctl --user restart openclaw-gateway 2>/dev/null; then
    echo "Gateway restarted"
else
    echo "Gateway restart skipped (systemctl not available or not user service)"
fi

echo "Uninstalled plugin: ${PLUGIN_SLUG} (id=${PLUGIN_ID})"
