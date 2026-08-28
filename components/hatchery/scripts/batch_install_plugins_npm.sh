#!/bin/bash
set -uo pipefail

# batch_install_plugins_npm.sh
# 通过 npm 批量安装插件到 OpenClaw 实例
#
# 参数（通过 TAT 模板变量注入）：
#   {{plugins_list}} - base64 编码的多行文本，每行 TAB 分隔：
#     npm_package<TAB>plugin_id<TAB>plugin_version<TAB>plugin_kind

PLUGINS_LIST_B64='{{plugins_list}}'
PLUGINS_LIST=$(echo "${PLUGINS_LIST_B64}" | base64 -d)
CONFIG_FILE="$HOME/.openclaw/openclaw.json"

RESULTS=()
INSTALLED_PLUGINS=()

add_result() {
    local slug="$1" version="$2" status="$3" message="$4"
    message="${message//\\/\\\\}"
    message="${message//\"/\\\"}"
    message="${message//$'\n'/\\n}"
    RESULTS+=("${slug}|${version}|${status}|${message}")
}

output_results() {
    local total=${#RESULTS[@]}
    local success=0
    local failed=0
    local json='{"results":['
    local first=true
    for entry in "${RESULTS[@]}"; do
        IFS='|' read -r slug version status message <<< "${entry}"
        if [ "${status}" = "success" ]; then
            ((success++))
        else
            ((failed++))
        fi
        if [ "${first}" = true ]; then
            first=false
        else
            json+=','
        fi
        json+="{\"slug\":\"${slug}\",\"version\":\"${version}\",\"status\":\"${status}\",\"message\":\"${message}\"}"
    done
    json+="],"
    json+="\"summary\":{\"total\":${total},\"success\":${success},\"failed\":${failed}}"
    json+='}'
    echo ""
    echo "========== BATCH INSTALL RESULTS =========="
    echo "${json}"
}

# 检查 openclaw CLI 是否可用
OPENCLAW_BIN=""
if command -v openclaw >/dev/null 2>&1; then
    OPENCLAW_BIN="openclaw"
elif [ -x "$HOME/.openclaw/bin/openclaw" ]; then
    OPENCLAW_BIN="$HOME/.openclaw/bin/openclaw"
elif [ -x "/usr/local/bin/openclaw" ]; then
    OPENCLAW_BIN="/usr/local/bin/openclaw"
fi

if [ -z "${OPENCLAW_BIN}" ]; then
    echo "Error: openclaw CLI not found" >&2
    add_result "all" "0.0.0" "failed" "openclaw CLI not found"
    output_results
    exit 1
fi

# 解析输入
NPM_PACKAGES=()
PLUGIN_IDS=()
PLUGIN_VERSIONS=()
PLUGIN_KINDS=()

while IFS=$'\t' read -r npm_pkg pid version kind; do
    [ -z "${npm_pkg}" ] && continue
    [[ "${npm_pkg}" == \#* ]] && continue
    NPM_PACKAGES+=("${npm_pkg}")
    PLUGIN_IDS+=("${pid}")
    PLUGIN_VERSIONS+=("${version}")
    PLUGIN_KINDS+=("${kind}")
done <<< "${PLUGINS_LIST}"

PLUGIN_COUNT=${#NPM_PACKAGES[@]}
if [ "${PLUGIN_COUNT}" -le 0 ]; then
    echo '{"results":[],"summary":{"total":0,"success":0,"failed":0}}'
    exit 0
fi

echo "Batch installing ${PLUGIN_COUNT} npm plugin(s)..."

# 逐个安装
for ((i = 0; i < PLUGIN_COUNT; i++)); do
    npm_pkg="${NPM_PACKAGES[${i}]}"
    pid="${PLUGIN_IDS[${i}]}"
    pver="${PLUGIN_VERSIONS[${i}]}"
    pkind="${PLUGIN_KINDS[${i}]}"

    install_target="${npm_pkg}"
    if [ -n "${pver}" ]; then
        install_target="${npm_pkg}@${pver}"
    fi

    # npm 包名安全校验：仅允许 @scope/name 或 name 格式
    if ! echo "${npm_pkg}" | grep -qE '^@?[a-zA-Z0-9][-a-zA-Z0-9._]*(\/[a-zA-Z0-9][-a-zA-Z0-9._]*)?$'; then
        echo "[${pid}] Invalid npm package name: ${npm_pkg}" >&2
        add_result "${pid}" "${pver}" "failed" "Invalid npm package name format"
        continue
    fi

    echo ""
    echo "--- [${pid}@${pver}] Installing via npm: ${install_target} ---"

    # 使用 openclaw CLI 安装
    install_output=$("${OPENCLAW_BIN}" plugins install "${install_target}" 2>&1)
    install_exit=$?

    if [ ${install_exit} -ne 0 ]; then
        echo "[${pid}] Install failed: ${install_output}" >&2
        add_result "${pid}" "${pver}" "failed" "npm install failed: ${install_output}"
        continue
    fi

    # 启用插件
    enable_output=$("${OPENCLAW_BIN}" plugins enable "${pid}" 2>&1)
    enable_exit=$?

    if [ ${enable_exit} -ne 0 ]; then
        echo "[${pid}] Enable failed: ${enable_output}" >&2
        # 安装成功但启用失败，仍然标记为成功（插件已安装，只是未启用）
    fi

    echo "[${pid}] Installed successfully via npm"
    add_result "${pid}" "${pver}" "success" "Installed successfully"
    INSTALLED_PLUGINS+=("${pid}|${pver}|${pkind}")
done

# 统一处理 slots 和 allow
if [ ${#INSTALLED_PLUGINS[@]} -gt 0 ] && [ -f "$CONFIG_FILE" ]; then
    # 确保 jq 可用
    if command -v jq >/dev/null 2>&1; then
        for entry in "${INSTALLED_PLUGINS[@]}"; do
            IFS='|' read -r pid pver pkind <<< "${entry}"

            # 从 deny 列表中移除
            OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
            jq --arg id "$pid" \
                '.plugins.deny = [.plugins.deny[]? | select(. != $id)]' \
                "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"

            # 更新 allow
            if jq -e '.plugins.allow | type == "array"' "$CONFIG_FILE" >/dev/null 2>&1; then
                OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
                jq --arg id "$pid" '
                  .plugins.allow = (.plugins.allow | if index($id) then . else . + [$id] end)
                ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
            fi

            # 处理独占槽位
            if [ "$pkind" = "memory" ]; then
                OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
                jq --arg id "$pid" '
                  .plugins.slots = (.plugins.slots // {}) | .plugins.slots.memory = $id
                ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
            elif [ "$pkind" = "context-engine" ]; then
                OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
                jq --arg id "$pid" '
                  .plugins.slots = (.plugins.slots // {}) | .plugins.slots.contextEngine = $id
                ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
            fi
        done
    fi

    # 统一重启 gateway
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    systemctl --user restart openclaw-gateway || true
fi

# 输出结果
output_results
