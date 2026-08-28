#!/bin/bash
set -euo pipefail

# install_skill_from_smh.sh
# 从 SMH 下载技能 zip 包并安装到 OpenClaw 实例
#
# 参数（通过 TAT 模板变量注入）：
#   {{download_url}}  - SMH zip 下载 URL（含 access_token）
#   {{skill_slug}}    - 技能 slug
#   {{skill_version}} - 技能版本号
#   {{agent_id}}      - 可选，OpenClaw 多 Agent ID

DOWNLOAD_URL="{{download_url}}"
SKILL_SLUG="{{skill_slug}}"
SKILL_VERSION="{{skill_version}}"
AGENT_ID="{{agent_id}}"
# agent_id 是可选参数；旧调用方可能没有在 TAT params 中传递该 key，
# 此时模板占位符会保留为字面量 {{agent_id}}，需要按空值处理。
if [[ "$AGENT_ID" == *"{{"*"}}"* ]]; then
    AGENT_ID=""
fi
WORKSPACE_ROOT=$(realpath -m "$HOME/.openclaw/workspace")
if [ -n "$AGENT_ID" ]; then
    AGENT_WORKSPACE=$(realpath -m "${WORKSPACE_ROOT}/${AGENT_ID}")
    case "$AGENT_WORKSPACE" in
        "${WORKSPACE_ROOT}"/*) ;;
        *) echo "Invalid agent workspace: ${AGENT_ID}" >&2; exit 1 ;;
    esac
    if [ ! -d "$AGENT_WORKSPACE" ]; then
        echo "Agent workspace not found: ${AGENT_ID}" >&2
        exit 1
    fi
    SKILLS_DIR="${AGENT_WORKSPACE}/skills"
else
    SKILLS_DIR="${WORKSPACE_ROOT}/skills"
fi
INSTALL_DIR="${SKILLS_DIR}/${SKILL_SLUG}"
LOCK_FILE="/tmp/install-skill-${SKILL_SLUG}.lock"
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
    # flock 会在进程结束时自动释放，这里只清理锁文件
    if [ -n "${LOCK_FD}" ]; then
        rm -f "${LOCK_FILE}" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

available_kb() {
    df -Pk "$1" | awk 'NR==2{print $4}'
}

echo "Installing skill: ${SKILL_SLUG}@${SKILL_VERSION}"

# ===== 参数安全校验 =====

if [[ "${SKILL_SLUG}" == *"/"* ]] || [[ "${SKILL_SLUG}" == *"\\"* ]] || [[ "${SKILL_SLUG}" == *".."* ]]; then
    echo "Invalid skill slug: ${SKILL_SLUG}" >&2
    exit 1
fi

if [[ "${SKILL_VERSION}" == *"/"* ]] || [[ "${SKILL_VERSION}" == *"\\"* ]] || [[ "${SKILL_VERSION}" == *".."* ]]; then
    echo "Invalid skill version: ${SKILL_VERSION}" >&2
    exit 1
fi

# 使用 flock 文件锁，避免 mkdir 锁的并发漏洞和僵尸锁问题
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

# 下载前先确认 /tmp 至少能放下 zip 包本身（服务端上传上限 50MB）
TMP_AVAIL_KB=$(available_kb /tmp)
if [ "${TMP_AVAIL_KB}" -lt 65536 ]; then
    echo "Insufficient disk space on /tmp: ${TMP_AVAIL_KB}KB available (need at least 64MB before download)" >&2
    exit 1
fi

mkdir -p "${SKILLS_DIR}"
TMP_ZIP=$(mktemp "/tmp/skill-${SKILL_SLUG}-${SKILL_VERSION}.XXXXXX")

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

# Zip Slip 防护：检查 ZIP 内是否包含危险路径
UNSAFE_ENTRIES=$(unzip -Z -1 "${TMP_ZIP}" | grep -E '(^/|(^|/)\.\.(/|$))' || true)
if [[ -n "${UNSAFE_ENTRIES}" ]]; then
    echo "Dangerous paths detected in archive (possible Zip Slip attack):" >&2
    echo "${UNSAFE_ENTRIES}" >&2
    exit 1
fi

# 计算解压后大小，使用 -qql 获取更干净的输出
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

# 直接解压到 Staging 目录，省去 /tmp 中转和 cp -a 操作
STAGING_DIR=$(mktemp -d "${SKILLS_DIR}/.${SKILL_SLUG}.staging.XXXXXX")
unzip -qo "${TMP_ZIP}" -d "${STAGING_DIR}"

# 如果解压后只有一个顶级目录，将其内容提升一级（兼容 zip 包内含/不含顶级目录）
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

# 校验技能包完整性：必须包含 SKILL.md（不区分大小写，与服务端校验逻辑一致）
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

echo "Skill installed: ${SKILL_SLUG}@${SKILL_VERSION} → ${INSTALL_DIR}"
