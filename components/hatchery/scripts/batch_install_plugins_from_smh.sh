#!/bin/bash
set -uo pipefail

# batch_install_plugins_from_smh.sh
# 从 SMH 批量下载插件 zip 包并安装到 OpenClaw 实例
#
# 参数（通过 TAT 模板变量注入）：
#   {{plugins_list}} - base64 编码的多行文本，每行 TAB 分隔：
#     download_url<TAB>plugin_slug<TAB>plugin_id<TAB>plugin_version<TAB>plugin_kind

PLUGINS_LIST_B64='{{plugins_list}}'
PLUGINS_LIST=$(echo "${PLUGINS_LIST_B64}" | base64 -d)
EXTENSIONS_DIR="$HOME/.openclaw/extensions"
CONFIG_FILE="$HOME/.openclaw/openclaw.json"
MAX_UNCOMPRESSED_BYTES=$((200 * 1024 * 1024))

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
    local json_array="[]"
    for entry in "${RESULTS[@]}"; do
        IFS='|' read -r slug version status message <<< "${entry}"
        if [ "${status}" = "success" ]; then
            ((success++))
        else
            ((failed++))
        fi
        json_array=$(echo "${json_array}" | jq --arg s "${slug}" --arg v "${version}" --arg st "${status}" --arg m "${message}" \
            '. + [{"slug":$s,"version":$v,"status":$st,"message":$m}]')
    done
    local json
    json=$(jq -n --argjson results "${json_array}" --argjson total "${total}" --argjson success "${success}" --argjson failed "${failed}" \
        '{"results":$results,"summary":{"total":$total,"success":$success,"failed":$failed}}')
    echo ""
    echo "========== BATCH INSTALL RESULTS =========="
    echo "${json}"
}

available_kb() {
    df -Pk "$1" | awk 'NR==2{print $4}'
}

encode_plugin_dir() {
    local pid="$1"
    if echo "$pid" | grep -q '/'; then
        echo "@$(echo -n "$pid" | md5sum | cut -c1-16)"
    else
        echo "$pid"
    fi
}

ensure_dependencies() {
    local need_install=false
    for cmd in curl unzip jq; do
        if ! command -v "${cmd}" >/dev/null 2>&1; then
            need_install=true
            break
        fi
    done
    if [ "${need_install}" = true ]; then
        local SUDO_CMD=""
        if [ "$(id -u)" -ne 0 ]; then
            if command -v sudo >/dev/null 2>&1; then
                SUDO_CMD="sudo"
            else
                echo "Error: Cannot install dependencies without root or sudo." >&2
                exit 1
            fi
        fi
        $SUDO_CMD apt-get update -qq && $SUDO_CMD apt-get install -y -qq curl unzip jq 2>/dev/null || \
        $SUDO_CMD yum install -y -q curl unzip jq 2>/dev/null || {
            echo "Failed to install dependencies" >&2
            exit 1
        }
    fi
}

ensure_dependencies

PLUGIN_URLS=()
PLUGIN_SLUGS=()
PLUGIN_IDS=()
PLUGIN_VERSIONS=()
PLUGIN_KINDS=()

while IFS=$'\t' read -r url slug pid version kind; do
    [ -z "${url}" ] && continue
    [[ "${url}" == \#* ]] && continue
    PLUGIN_URLS+=("${url}")
    PLUGIN_SLUGS+=("${slug}")
    PLUGIN_IDS+=("${pid}")
    PLUGIN_VERSIONS+=("${version}")
    PLUGIN_KINDS+=("${kind}")
done <<< "${PLUGINS_LIST}"

PLUGIN_COUNT=${#PLUGIN_SLUGS[@]}
if [ "${PLUGIN_COUNT}" -le 0 ]; then
    echo '{"results":[],"summary":{"total":0,"success":0,"failed":0}}'
    exit 1
fi

echo "Batch installing ${PLUGIN_COUNT} plugin(s)..."

TMP_AVAIL_KB=$(available_kb /tmp)
if [ "${TMP_AVAIL_KB}" -lt 65536 ]; then
    echo '{"results":[],"summary":{"total":0,"success":0,"failed":0},"error":"Insufficient disk space on /tmp"}'
    exit 1
fi

mkdir -p "${EXTENSIONS_DIR}"

CURL_RETRY_OPTS=(--retry 3 --retry-delay 2)
if curl --help all 2>/dev/null | grep -q -- '--retry-all-errors'; then
    CURL_RETRY_OPTS+=(--retry-all-errors)
fi

install_single_plugin() {
    local download_url="$1"
    local plugin_slug="$2"
    local plugin_id="$3"
    local plugin_version="$4"
    local plugin_kind="$5"

    local encoded_dir
    encoded_dir=$(encode_plugin_dir "$plugin_id")
    local install_dir="${EXTENSIONS_DIR}/${encoded_dir}"
    local lock_file="/tmp/install-plugin-${plugin_slug}.lock"
    local tmp_zip=""
    local staging_dir=""
    local backup_dir=""
    local cleanup_backup=1
    local lock_fd=""

    skill_cleanup() {
        [ -n "${tmp_zip}" ] && rm -f "${tmp_zip}" 2>/dev/null || true
        [ -n "${staging_dir}" ] && rm -rf "${staging_dir}" 2>/dev/null || true
        if [ "${cleanup_backup}" -eq 1 ] && [ -n "${backup_dir}" ]; then
            rm -rf "${backup_dir}" 2>/dev/null || true
        fi
        if [ -n "${lock_fd}" ]; then
            eval "exec ${lock_fd}>&-" 2>/dev/null || true
            rm -f "${lock_file}" 2>/dev/null || true
        fi
    }

    echo ""
    echo "--- [${plugin_slug}@${plugin_version}] Starting installation ---"

    if [[ "${plugin_slug}" == *".."* ]]; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Invalid plugin slug"
        return
    fi

    if [[ "${plugin_id}" == *".."* ]]; then
        add_result "${plugin_id}" "${plugin_version}" "failed" "Invalid plugin id"
        return
    fi

    local fd=$((10 + CURRENT_INDEX))
    eval "exec ${fd}> '${lock_file}'"
    if ! flock -n "${fd}"; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Another installation in progress"
        return
    fi
    lock_fd="${fd}"

    tmp_zip=$(mktemp "/tmp/plugin-${plugin_slug}-${plugin_version}.XXXXXX")
    echo "[${plugin_slug}] Downloading from SMH..."
    if ! curl -fsSL --connect-timeout 10 --max-time 60 \
        "${CURL_RETRY_OPTS[@]}" \
        -o "${tmp_zip}" "${download_url}"; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Download failed"
        skill_cleanup
        return
    fi

    if [ ! -s "${tmp_zip}" ]; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Download failed: empty file"
        skill_cleanup
        return
    fi

    if ! unzip -tq "${tmp_zip}" >/dev/null 2>&1; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Invalid zip file"
        skill_cleanup
        return
    fi

    local unsafe_entries
    unsafe_entries=$(unzip -Z -1 "${tmp_zip}" | grep -E '(^/|(^|/)\.\.(/|$))' || true)
    if [[ -n "${unsafe_entries}" ]]; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Dangerous paths detected"
        skill_cleanup
        return
    fi

    local uncompressed_bytes
    uncompressed_bytes=$(unzip -qql "${tmp_zip}" | awk '{sum += $1} END {print sum+0}')
    if [ "${uncompressed_bytes}" -le 0 ]; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Archive is empty"
        skill_cleanup
        return
    fi
    if [ "${uncompressed_bytes}" -gt "${MAX_UNCOMPRESSED_BYTES}" ]; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Exceeds 200MB limit"
        skill_cleanup
        return
    fi

    staging_dir=$(mktemp -d "${EXTENSIONS_DIR}/.${plugin_slug}.staging.XXXXXX")
    if ! unzip -qo "${tmp_zip}" -d "${staging_dir}"; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Failed to extract zip"
        skill_cleanup
        return
    fi

    shopt -s dotglob nullglob
    local entries=("${staging_dir}"/*)
    if [ ${#entries[@]} -eq 1 ] && [ -d "${entries[0]}" ]; then
        local nested_dir="${entries[0]}"
        local nested_entries=("${nested_dir}"/*)
        if [ ${#nested_entries[@]} -gt 0 ]; then
            mv -- "${nested_entries[@]}" "${staging_dir}/"
        fi
        rmdir "${nested_dir}" 2>/dev/null || rm -rf "${nested_dir}"
    fi
    shopt -u dotglob nullglob

    # 校验锚点
    local manifest_found
    manifest_found=$(find "${staging_dir}" -maxdepth 1 -name "openclaw.plugin.json" -print -quit)
    local bundle_found=""
    if [ -z "${manifest_found}" ]; then
        for dir in ".codex-plugin" ".claude-plugin" ".cursor-plugin"; do
            if [ -d "${staging_dir}/${dir}" ]; then
                bundle_found="${dir}"
                break
            fi
        done
    fi
    if [ -z "${manifest_found}" ] && [ -z "${bundle_found}" ]; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "No openclaw.plugin.json or bundle dir found"
        skill_cleanup
        return
    fi

    if [ -e "${install_dir}" ] && [ ! -d "${install_dir}" ]; then
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Install path is not a directory"
        skill_cleanup
        return
    fi

    if [ -d "${install_dir}" ]; then
        backup_dir="${EXTENSIONS_DIR}/.${plugin_slug}.backup.$$"
        mv "${install_dir}" "${backup_dir}"
        cleanup_backup=0
    fi

    if mv "${staging_dir}" "${install_dir}"; then
        staging_dir=""
    else
        if [ -n "${backup_dir}" ] && [ -d "${backup_dir}" ] && [ ! -e "${install_dir}" ]; then
            mv "${backup_dir}" "${install_dir}" || true
            cleanup_backup=1
        fi
        add_result "${plugin_slug}" "${plugin_version}" "failed" "Failed to activate plugin files"
        skill_cleanup
        return
    fi

    if [ -n "${backup_dir}" ] && [ -d "${backup_dir}" ]; then
        rm -rf "${backup_dir}"
    fi
    backup_dir=""
    cleanup_backup=1

    echo "[${plugin_slug}] Installed: ${plugin_slug}@${plugin_version} → ${install_dir}"
    add_result "${plugin_slug}" "${plugin_version}" "success" "Installed successfully"

    # 记录成功安装的插件信息，用于后续统一配置写入
    INSTALLED_PLUGINS+=("${plugin_id}|${plugin_version}|${install_dir}|${plugin_kind}")

    [ -n "${tmp_zip}" ] && rm -f "${tmp_zip}" 2>/dev/null || true
    tmp_zip=""
    if [ -n "${lock_fd}" ]; then
        eval "exec ${lock_fd}>&-" 2>/dev/null || true
        rm -f "${lock_file}" 2>/dev/null || true
        lock_fd=""
    fi
}

# ===== 主循环 =====

for ((i = 0; i < PLUGIN_COUNT; i++)); do
    CURRENT_INDEX=${i}
    install_single_plugin "${PLUGIN_URLS[${i}]}" "${PLUGIN_SLUGS[${i}]}" "${PLUGIN_IDS[${i}]}" "${PLUGIN_VERSIONS[${i}]}" "${PLUGIN_KINDS[${i}]}"
done

# ===== 统一配置写入 =====

if [ ${#INSTALLED_PLUGINS[@]} -gt 0 ]; then
    if [ ! -f "$CONFIG_FILE" ]; then
        mkdir -p "$(dirname "$CONFIG_FILE")"
        echo '{}' > "$CONFIG_FILE"
    fi

    cp "$CONFIG_FILE" "${CONFIG_FILE}.bak.$(date +%Y-%m-%dT%H:%M:%S)"

    # 确保 plugins 对象存在
    OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
    jq '.plugins = (.plugins // {}) | .plugins.entries = (.plugins.entries // {}) | .plugins.installs = (.plugins.installs // {})' \
        "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"

    for entry in "${INSTALLED_PLUGINS[@]}"; do
        IFS='|' read -r pid pver ppath pkind <<< "${entry}"

        # 从 deny 列表中移除
        OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
        jq --arg id "$pid" \
            '.plugins.deny = [.plugins.deny[]? | select(. != $id)]' \
            "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"

        # 写入 entries
        OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
        jq --arg id "$pid" '
          .plugins.entries[$id] = ((.plugins.entries[$id] // {}) + {"enabled": true}) |
          .plugins.entries[$id].config = (.plugins.entries[$id].config // {})
        ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"

        # 写入 installs
        OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
        jq --arg id "$pid" --arg ver "$pver" --arg ipath "$ppath" \
           --arg ts "$(date -u +%Y-%m-%dT%H:%M:%S.000Z)" '
          .plugins.installs[$id] = {
            "source": "archive",
            "installPath": $ipath,
            "version": $ver,
            "resolvedVersion": $ver,
            "installedAt": $ts
          }
        ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"

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

    # 统一重启 gateway（只重启一次）
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    systemctl --user restart openclaw-gateway || true
fi

# ===== 输出结果 =====

output_results
