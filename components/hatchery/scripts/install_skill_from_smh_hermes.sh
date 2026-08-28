#!/bin/bash
set -euo pipefail

# install_skill_from_smh_hermes.sh
# 从 SMH 下载技能 zip 包并安装到 Hermes 实例
#
# 参数（通过 TAT 模板变量注入，与 openclaw install_skill_from_smh.sh 契约完全一致）：
#   {{download_url}}  - SMH zip 下载 URL（含 access_token）
#   {{skill_slug}}    - 技能 slug
#   {{skill_version}} - 技能版本号
#
# 与 openclaw install_skill_from_smh.sh 的差异：
#   - 安装目录: ~/.openclaw/workspace/skills → ~/.hermes/skills
#   - 启用方式：不依赖 `openclaw` CLI
#     * 若存在 harness CLI：在安装完成后尝试 `harness skills reload` 让 gateway 热载
#       （harness 可能无 reload 子命令，失败不视为安装失败——依赖 harness 自身 watcher
#       或下次会话重载）
#   - 保留：flock 并发锁 / Zip Slip 防护 / 200MB 上限 / 磁盘空间预检 / 备份回滚
#   - 注意：openclaw 版会校验 SKILL.md，这里同样保留（harness 生态与 openclaw 对齐）

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:/usr/local/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"

# ========== 拉取/更新 harness CLI ==========
ensure_harness_cli() {
    local HARNESS_URL="https://agentone-1388575217.cos.ap-guangzhou.myqcloud.com/harness-linux-amd64"
    local HARNESS_BIN="$HOME/.local/bin/harness"
    mkdir -p "$(dirname "$HARNESS_BIN")" 2>/dev/null || true
    echo "ℹ 拉取 harness CLI: $HARNESS_URL"
    if curl -fsSL --connect-timeout 10 --max-time 60 --retry 2 --retry-delay 2 \
        "$HARNESS_URL" -o "${HARNESS_BIN}.tmp" 2>/dev/null; then
        chmod +x "${HARNESS_BIN}.tmp"
        mv -f "${HARNESS_BIN}.tmp" "$HARNESS_BIN"
        echo "✓ harness CLI 已更新: $HARNESS_BIN"
    else
        rm -f "${HARNESS_BIN}.tmp" 2>/dev/null || true
        if command -v harness >/dev/null 2>&1; then
            echo "⚠ harness CLI 下载失败，沿用已有版本: $(command -v harness)"
        else
            echo "ℹ harness CLI 下载失败且本地无已有版本（非关键路径，继续）"
            return 1
        fi
    fi
}

DOWNLOAD_URL="{{download_url}}"
SKILL_SLUG="{{skill_slug}}"
SKILL_VERSION="{{skill_version}}"
SKILLS_DIR="$HOME/.hermes/skills"
INSTALL_DIR="${SKILLS_DIR}/${SKILL_SLUG}"
LOCK_FILE="/tmp/install-skill-hermes-${SKILL_SLUG}.lock"
TMP_ZIP=""
STAGING_DIR=""
BACKUP_DIR=""
CLEANUP_BACKUP=1
LOCK_FD=""

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

echo "Installing skill (hermes): ${SKILL_SLUG}@${SKILL_VERSION}"

# ===== 参数安全校验 =====

if [[ "${SKILL_SLUG}" == *"/"* ]] || [[ "${SKILL_SLUG}" == *"\\"* ]] || [[ "${SKILL_SLUG}" == *".."* ]]; then
    echo "Invalid skill slug: ${SKILL_SLUG}" >&2
    exit 1
fi

if [[ "${SKILL_VERSION}" == *"/"* ]] || [[ "${SKILL_VERSION}" == *"\\"* ]] || [[ "${SKILL_VERSION}" == *".."* ]]; then
    echo "Invalid skill version: ${SKILL_VERSION}" >&2
    exit 1
fi

# flock 文件锁，避免并发安装同一技能
exec 9> "${LOCK_FILE}"
if ! flock -n 9; then
    echo "Another installation of ${SKILL_SLUG} is already in progress" >&2
    exit 1
fi
LOCK_FD=9

# 检查 curl 和 unzip 是否可用
if ! command -v curl >/dev/null 2>&1 || ! command -v unzip >/dev/null 2>&1; then
    echo "Missing curl or unzip. Attempting installation..." >&2
    SUDO_CMD=""
    if [ "$(id -u)" -ne 0 ]; then
        if command -v sudo >/dev/null 2>&1; then
            SUDO_CMD="sudo"
        else
            echo "Error: Cannot install dependencies without root or sudo. Please install curl and unzip manually." >&2
            exit 1
        fi
    fi
    $SUDO_CMD apt-get update -qq && $SUDO_CMD apt-get install -y -qq curl unzip 2>/dev/null || \
    $SUDO_CMD yum install -y -q curl unzip 2>/dev/null || {
        echo "Failed to install dependencies" >&2
        exit 1
    }
fi

# 下载前确认 /tmp 空间
TMP_AVAIL_KB=$(available_kb /tmp)
if [ "${TMP_AVAIL_KB}" -lt 65536 ]; then
    echo "Insufficient disk space on /tmp: ${TMP_AVAIL_KB}KB available (need at least 64MB before download)" >&2
    exit 1
fi

mkdir -p "${SKILLS_DIR}"
TMP_ZIP=$(mktemp "/tmp/skill-hermes-${SKILL_SLUG}-${SKILL_VERSION}.XXXXXX")

echo "Downloading from SMH..."
CURL_RETRY_OPTS=(--retry 3 --retry-delay 2)
if curl --help all 2>/dev/null | grep -q -- '--retry-all-errors'; then
    CURL_RETRY_OPTS+=(--retry-all-errors)
fi
curl -fsSL --connect-timeout 10 --max-time 30 \
    "${CURL_RETRY_OPTS[@]}" \
    -o "${TMP_ZIP}" "${DOWNLOAD_URL}"
if [ ! -s "${TMP_ZIP}" ]; then
    echo "Download failed: empty file" >&2
    exit 1
fi
echo "Downloaded: $(du -h "${TMP_ZIP}" | cut -f1)"

# 校验 zip 完整性
if ! unzip -tq "${TMP_ZIP}" >/dev/null 2>&1; then
    echo "Download corrupted: invalid zip file" >&2
    exit 1
fi

# Zip Slip 防护
UNSAFE_ENTRIES=$(unzip -Z -1 "${TMP_ZIP}" | grep -E '(^/|(^|/)\.\.(/|$))' || true)
if [[ -n "${UNSAFE_ENTRIES}" ]]; then
    echo "Dangerous paths detected in archive (possible Zip Slip attack):" >&2
    echo "${UNSAFE_ENTRIES}" >&2
    exit 1
fi

# 解压后大小限制
UNCOMPRESSED_BYTES=$(unzip -qql "${TMP_ZIP}" | awk '{sum += $1} END {print sum+0}')
MAX_UNCOMPRESSED_BYTES=$((200 * 1024 * 1024))
if [ "${UNCOMPRESSED_BYTES}" -le 0 ]; then
    echo "Invalid skill package: archive is empty" >&2
    exit 1
fi
if [ "${UNCOMPRESSED_BYTES}" -gt "${MAX_UNCOMPRESSED_BYTES}" ]; then
    echo "Invalid skill package: uncompressed size exceeds 200MB limit" >&2
    exit 1
fi

UNCOMPRESSED_KB=$(((UNCOMPRESSED_BYTES + 1023) / 1024))
TARGET_REQUIRED_KB=$((UNCOMPRESSED_KB + 10240))

TARGET_AVAIL_KB=$(available_kb "${SKILLS_DIR}")
if [ "${TARGET_AVAIL_KB}" -lt "${TARGET_REQUIRED_KB}" ]; then
    echo "Insufficient disk space on target filesystem: ${TARGET_AVAIL_KB}KB available (need about ${TARGET_REQUIRED_KB}KB)" >&2
    exit 1
fi

# 直接解压到 Staging 目录
STAGING_DIR=$(mktemp -d "${SKILLS_DIR}/.${SKILL_SLUG}.staging.XXXXXX")
unzip -qo "${TMP_ZIP}" -d "${STAGING_DIR}"

# 如果解压后只有一个顶级目录，将其内容提升一级
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

# 校验技能包完整性：必须包含 SKILL.md（与 openclaw 打包规范对齐）
SKILL_MD_FOUND=$(find "${STAGING_DIR}" -maxdepth 1 -iname "SKILL.md" -print -quit)
if [ -z "${SKILL_MD_FOUND}" ]; then
    echo "Invalid skill package: SKILL.md not found in archive" >&2
    echo "Archive contents:" >&2
    ls -la "${STAGING_DIR}/" >&2
    exit 1
fi

if [ -e "${INSTALL_DIR}" ] && [ ! -d "${INSTALL_DIR}" ]; then
    echo "Install path exists but is not a directory: ${INSTALL_DIR}" >&2
    exit 1
fi

# 备份旧版本以便回滚
if [ -d "${INSTALL_DIR}" ]; then
    BACKUP_DIR="${SKILLS_DIR}/.${SKILL_SLUG}.backup.$$"
    mv "${INSTALL_DIR}" "${BACKUP_DIR}"
    CLEANUP_BACKUP=0
fi

if mv "${STAGING_DIR}" "${INSTALL_DIR}"; then
    STAGING_DIR=""
else
    echo "Failed to activate new skill files" >&2
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

# ===== 尝试 reload（非关键路径，失败不影响安装结果） =====
# 说明：harness CLI 的 reload 子命令并未公开，若存在则调用以让 gateway 立即加载新技能；
# 否则依赖 harness 自身的 skill watcher 或下次会话自动重载。
ensure_harness_cli || true
if command -v harness >/dev/null 2>&1; then
    if harness skills reload >/dev/null 2>&1; then
        echo "✓ harness skills reload triggered"
    else
        echo "ℹ harness skills reload not available or failed; relying on watcher/next session"
    fi
else
    echo "ℹ harness CLI not found; skill will be picked up by watcher or next session"
fi

echo "Skill installed: ${SKILL_SLUG}@${SKILL_VERSION} → ${INSTALL_DIR}"
