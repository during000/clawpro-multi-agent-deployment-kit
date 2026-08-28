#!/bin/bash
set -euo pipefail

# install_plugin_from_smh.sh
# 从 SMH 下载插件 zip 包并安装到 OpenClaw 实例
#
# 参数（通过 TAT 模板变量注入）：
#   {{download_url}}    - SMH zip 下载 URL（含 access_token）
#   {{plugin_slug}}     - 插件 slug（Hatchery 管理标识）
#   {{plugin_id}}       - openclaw.plugin.json 中的 id
#   {{plugin_version}}  - 插件版本号
#   {{plugin_kind}}     - 插件类型（memory / context-engine / 空）

DOWNLOAD_URL="{{download_url}}"
PLUGIN_SLUG="{{plugin_slug}}"
PLUGIN_ID="{{plugin_id}}"
PLUGIN_VERSION="{{plugin_version}}"
PLUGIN_KIND="{{plugin_kind}}"
EXTENSIONS_DIR="$HOME/.openclaw/extensions"
CONFIG_FILE="$HOME/.openclaw/openclaw.json"
LOCK_FILE="/tmp/install-plugin-${PLUGIN_SLUG}.lock"
TMP_ZIP=""
STAGING_DIR=""
BACKUP_DIR=""
CLEANUP_BACKUP=1
LOCK_FD=""

# 安装目录编码（对齐 encodePluginInstallDirName）
if echo "$PLUGIN_ID" | grep -q '/'; then
    ENCODED_DIR="@$(echo -n "$PLUGIN_ID" | md5sum | cut -c1-16)"
else
    ENCODED_DIR="$PLUGIN_ID"
fi
INSTALL_DIR="${EXTENSIONS_DIR}/${ENCODED_DIR}"

cleanup() {
    [ -n "${TMP_ZIP}" ] && rm -f "${TMP_ZIP}" 2>/dev/null || true
    [ -n "${STAGING_DIR}" ] && rm -rf "${STAGING_DIR}" 2>/dev/null || true
    if [ "${CLEANUP_BACKUP}" -eq 1 ] && [ -n "${BACKUP_DIR}" ]; then
        rm -rf "${BACKUP_DIR}" 2>/dev/null || true
    fi
    if [ -n "${LOCK_FD}" ]; then
        rm -f "${LOCK_FILE}" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

available_kb() {
    df -Pk "$1" | awk 'NR==2{print $4}'
}

echo "Installing plugin: ${PLUGIN_SLUG}@${PLUGIN_VERSION} (id=${PLUGIN_ID})"

# ===== 参数安全校验 =====

if [[ "${PLUGIN_SLUG}" == *".."* ]]; then
    echo "Invalid plugin slug: ${PLUGIN_SLUG}" >&2
    exit 1
fi

if [[ "${PLUGIN_ID}" == *".."* ]]; then
    echo "Invalid plugin id: ${PLUGIN_ID}" >&2
    exit 1
fi

if [[ "${PLUGIN_VERSION}" == *"/"* ]] || [[ "${PLUGIN_VERSION}" == *"\\"* ]] || [[ "${PLUGIN_VERSION}" == *".."* ]]; then
    echo "Invalid plugin version: ${PLUGIN_VERSION}" >&2
    exit 1
fi

# 使用 flock 文件锁
exec 9> "${LOCK_FILE}"
if ! flock -n 9; then
    echo "Another installation of ${PLUGIN_SLUG} is already in progress" >&2
    exit 1
fi
LOCK_FD=9

# 检查 curl、unzip 和 jq（jq 用于 openclaw.json 配置写入）
NEED_INSTALL=false
for cmd in curl unzip jq; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
        NEED_INSTALL=true
        break
    fi
done
if [ "${NEED_INSTALL}" = true ]; then
    echo "Missing required dependencies (curl/unzip/jq). Attempting installation..." >&2
    SUDO_CMD=""
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

# 检查 /tmp 空间
TMP_AVAIL_KB=$(available_kb /tmp)
if [ "${TMP_AVAIL_KB}" -lt 65536 ]; then
    echo "Insufficient disk space on /tmp: ${TMP_AVAIL_KB}KB available" >&2
    exit 1
fi

mkdir -p "${EXTENSIONS_DIR}"
TMP_ZIP=$(mktemp "/tmp/plugin-${PLUGIN_SLUG}-${PLUGIN_VERSION}.XXXXXX")

echo "Downloading from SMH..."
CURL_RETRY_OPTS=(--retry 3 --retry-delay 2)
if curl --help all 2>/dev/null | grep -q -- '--retry-all-errors'; then
    CURL_RETRY_OPTS+=(--retry-all-errors)
fi
curl -fsSL --connect-timeout 10 --max-time 60 \
    "${CURL_RETRY_OPTS[@]}" \
    -o "${TMP_ZIP}" "${DOWNLOAD_URL}"
if [ ! -s "${TMP_ZIP}" ]; then
    echo "Download failed: empty file" >&2
    exit 1
fi
echo "Downloaded: $(du -h "${TMP_ZIP}" | cut -f1)"

# zip 完整性校验
if ! unzip -tq "${TMP_ZIP}" >/dev/null 2>&1; then
    echo "Download corrupted: invalid zip file" >&2
    exit 1
fi

# Zip Slip 防护
UNSAFE_ENTRIES=$(unzip -Z -1 "${TMP_ZIP}" | grep -E '(^/|(^|/)\.\.(/|$))' || true)
if [[ -n "${UNSAFE_ENTRIES}" ]]; then
    echo "Dangerous paths detected in archive" >&2
    exit 1
fi

# 解压大小检查
UNCOMPRESSED_BYTES=$(unzip -qql "${TMP_ZIP}" | awk '{sum += $1} END {print sum+0}')
MAX_UNCOMPRESSED_BYTES=$((200 * 1024 * 1024))
if [ "${UNCOMPRESSED_BYTES}" -le 0 ]; then
    echo "Invalid plugin package: archive is empty" >&2
    exit 1
fi
if [ "${UNCOMPRESSED_BYTES}" -gt "${MAX_UNCOMPRESSED_BYTES}" ]; then
    echo "Invalid plugin package: uncompressed size exceeds 200MB limit" >&2
    exit 1
fi

UNCOMPRESSED_KB=$(((UNCOMPRESSED_BYTES + 1023) / 1024))
TARGET_REQUIRED_KB=$((UNCOMPRESSED_KB + 10240))
TARGET_AVAIL_KB=$(available_kb "${EXTENSIONS_DIR}")
if [ "${TARGET_AVAIL_KB}" -lt "${TARGET_REQUIRED_KB}" ]; then
    echo "Insufficient disk space: ${TARGET_AVAIL_KB}KB available, need ${TARGET_REQUIRED_KB}KB" >&2
    exit 1
fi

# 解压到 staging
STAGING_DIR=$(mktemp -d "${EXTENSIONS_DIR}/.${PLUGIN_SLUG}.staging.XXXXXX")
unzip -qo "${TMP_ZIP}" -d "${STAGING_DIR}"

# 顶级目录提升
shopt -s dotglob nullglob
ENTRIES=("${STAGING_DIR}"/*)
if [ ${#ENTRIES[@]} -eq 1 ] && [ -d "${ENTRIES[0]}" ]; then
    NESTED_DIR="${ENTRIES[0]}"
    NESTED_ENTRIES=("${NESTED_DIR}"/*)
    if [ ${#NESTED_ENTRIES[@]} -gt 0 ]; then
        mv -- "${NESTED_ENTRIES[@]}" "${STAGING_DIR}/"
    fi
    rmdir "${NESTED_DIR}" 2>/dev/null || rm -rf "${NESTED_DIR}"
fi
shopt -u dotglob nullglob

# 校验锚点
MANIFEST_FOUND=$(find "${STAGING_DIR}" -maxdepth 1 -name "openclaw.plugin.json" -print -quit)
BUNDLE_FOUND=""
if [ -z "${MANIFEST_FOUND}" ]; then
    for dir in ".codex-plugin" ".claude-plugin" ".cursor-plugin"; do
        if [ -d "${STAGING_DIR}/${dir}" ]; then
            BUNDLE_FOUND="${dir}"
            break
        fi
    done
fi
if [ -z "${MANIFEST_FOUND}" ] && [ -z "${BUNDLE_FOUND}" ]; then
    echo "Invalid plugin package: neither openclaw.plugin.json nor bundle directory found" >&2
    echo "Archive contents:" >&2
    ls -la "${STAGING_DIR}/" >&2
    exit 1
fi

# 原子替换
if [ -e "${INSTALL_DIR}" ] && [ ! -d "${INSTALL_DIR}" ]; then
    echo "Install path exists but is not a directory: ${INSTALL_DIR}" >&2
    exit 1
fi

if [ -d "${INSTALL_DIR}" ]; then
    BACKUP_DIR="${EXTENSIONS_DIR}/.${PLUGIN_SLUG}.backup.$$"
    mv "${INSTALL_DIR}" "${BACKUP_DIR}"
    CLEANUP_BACKUP=0
fi

if mv "${STAGING_DIR}" "${INSTALL_DIR}"; then
    STAGING_DIR=""
else
    echo "Failed to activate new plugin files" >&2
    if [ -n "${BACKUP_DIR}" ] && [ -d "${BACKUP_DIR}" ] && [ ! -e "${INSTALL_DIR}" ]; then
        mv "${BACKUP_DIR}" "${INSTALL_DIR}" || true
        CLEANUP_BACKUP=1
    fi
    exit 1
fi

if [ -n "${BACKUP_DIR}" ] && [ -d "${BACKUP_DIR}" ]; then
    rm -rf "${BACKUP_DIR}"
fi
BACKUP_DIR=""
CLEANUP_BACKUP=1

echo "Plugin files installed to ${INSTALL_DIR}"

# ===== openclaw.json 配置写入 =====

# 确保配置文件存在
if [ ! -f "$CONFIG_FILE" ]; then
    mkdir -p "$(dirname "$CONFIG_FILE")"
    echo '{}' > "$CONFIG_FILE"
fi

# 检查 plugins.deny 黑名单，企业下发时自动移除
if jq -e --arg id "$PLUGIN_ID" '.plugins.deny // [] | index($id) != null' "$CONFIG_FILE" >/dev/null 2>&1; then
    echo "Warning: plugin $PLUGIN_ID is in deny list, removing for enterprise install" >&2
OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
    jq --arg id "$PLUGIN_ID" \
        '.plugins.deny = [.plugins.deny[]? | select(. != $id)]' \
        "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
fi

OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
cp "$CONFIG_FILE" "${CONFIG_FILE}.bak.$(date +%Y-%m-%dT%H:%M:%S)"

# 1. 确保 plugins 对象存在
jq '.plugins = (.plugins // {})' "$CONFIG_FILE" > "$OC_TMP" \
    && mv "$OC_TMP" "$CONFIG_FILE"

# 2. 写入 plugins.entries.{plugin_id}（保留已有 config）
OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
jq --arg id "$PLUGIN_ID" '
  .plugins.entries = (.plugins.entries // {}) |
  .plugins.entries[$id] = ((.plugins.entries[$id] // {}) + {"enabled": true}) |
  .plugins.entries[$id].config = (.plugins.entries[$id].config // {})
' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"

# 3. 写入 plugins.installs.{plugin_id}
OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
jq --arg id "$PLUGIN_ID" \
   --arg ver "$PLUGIN_VERSION" \
   --arg ipath "$INSTALL_DIR" \
   --arg ts "$(date -u +%Y-%m-%dT%H:%M:%S.000Z)" '
  .plugins.installs = (.plugins.installs // {}) |
  .plugins.installs[$id] = {
    "source": "archive",
    "installPath": $ipath,
    "version": $ver,
    "resolvedVersion": $ver,
    "installedAt": $ts
  }
' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"

# 4. 更新 plugins.allow（仅当已存在 allow 数组时追加）
if jq -e '.plugins.allow | type == "array"' "$CONFIG_FILE" >/dev/null 2>&1; then
  OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
  jq --arg id "$PLUGIN_ID" '
    .plugins.allow = (.plugins.allow | if index($id) then . else . + [$id] end)
  ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
fi

# 5. 处理独占槽位
if [ "$PLUGIN_KIND" = "memory" ]; then
    OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
    jq --arg id "$PLUGIN_ID" '
      .plugins.slots = (.plugins.slots // {}) |
      .plugins.slots.memory = $id
    ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
elif [ "$PLUGIN_KIND" = "context-engine" ]; then
    OC_TMP=$(mktemp /tmp/oc_tmp.XXXXXX)
    jq --arg id "$PLUGIN_ID" '
      .plugins.slots = (.plugins.slots // {}) |
      .plugins.slots.contextEngine = $id
    ' "$CONFIG_FILE" > "$OC_TMP" && mv "$OC_TMP" "$CONFIG_FILE"
fi

# 6. 重启 gateway
export XDG_RUNTIME_DIR=/run/user/$(id -u)
systemctl --user restart openclaw-gateway || true

echo "Plugin installed: ${PLUGIN_SLUG}@${PLUGIN_VERSION} (id=${PLUGIN_ID}) → ${INSTALL_DIR}"
