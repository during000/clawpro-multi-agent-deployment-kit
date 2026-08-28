#!/bin/bash
set -euo pipefail

# install_skill_from_smh_ace.sh
# 从 SMH 下载技能 zip 包并安装到 LightClaw-ACE 实例
#
# 参数（通过 TAT 模板变量注入，与 openclaw install_skill_from_smh.sh 契约完全一致）：
#   {{download_url}}  - SMH zip 下载 URL（含 access_token）
#   {{skill_slug}}    - 技能 slug
#   {{skill_version}} - 技能版本号
#
# 与 openclaw install_skill_from_smh.sh 的差异：
#   - 安装目录: ~/.openclaw/workspace/skills → ~/.lightclaw/workspace/skills
#   - 启用方式：不调用 `openclaw skills enable`（lightclaw 无该 CLI）
#     改为 `jq` 原子写入 ~/.lightclaw/lightclaw.json 的 .skills.entries.<slug>.enabled = true
#   - 不主动重启 lightclaw（强行重启会中断正在进行的对话，单技能下发是热操作场景）
#     依赖 ACE 自身 skill watcher 或下次会话重载生效
#   - 不强制校验 SKILL.md（与 batch_install_skills_from_smh_ace.sh 宽松策略保持一致）
#   - 保留：flock 并发锁 / Zip Slip 防护 / 200MB 上限 / 磁盘空间预检 / 备份回滚

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:/usr/local/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"

DOWNLOAD_URL="{{download_url}}"
SKILL_SLUG="{{skill_slug}}"
SKILL_VERSION="{{skill_version}}"
LIGHTCLAW_HOME="$HOME/.lightclaw"
SKILLS_DIR="${LIGHTCLAW_HOME}/workspace/skills"
CONFIG="${LIGHTCLAW_HOME}/lightclaw.json"
INSTALL_DIR="${SKILLS_DIR}/${SKILL_SLUG}"
LOCK_FILE="/tmp/install-skill-ace-${SKILL_SLUG}.lock"
TMP_ZIP=""
STAGING_DIR=""
BACKUP_DIR=""
CONFIG_BACKUP=""
CLEANUP_BACKUP=1
LOCK_FD=""

cleanup() {
    [ -n "${TMP_ZIP}" ] && rm -f "${TMP_ZIP}" 2>/dev/null || true
    [ -n "${STAGING_DIR}" ] && rm -rf "${STAGING_DIR}" 2>/dev/null || true
    if [ "${CLEANUP_BACKUP}" -eq 1 ] && [ -n "${BACKUP_DIR}" ]; then
        rm -rf "${BACKUP_DIR}" 2>/dev/null || true
    fi
    # config 备份在成功写回后清理；失败路径下保留供手工回滚
    if [ -n "${LOCK_FD}" ]; then
        rm -f "${LOCK_FILE}" 2>/dev/null || true
    fi
}
trap cleanup EXIT INT TERM

available_kb() {
    df -Pk "$1" | awk 'NR==2{print $4}'
}

echo "Installing skill (ace): ${SKILL_SLUG}@${SKILL_VERSION}"

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

# 检查依赖：curl / unzip / jq（ACE 版额外需要 jq 写配置）
ensure_dependencies() {
    local missing=()
    for cmd in curl unzip jq; do
        command -v "${cmd}" >/dev/null 2>&1 || missing+=("${cmd}")
    done
    [ ${#missing[@]} -eq 0 ] && return 0

    echo "Missing: ${missing[*]}. Attempting installation..." >&2
    local SUDO_CMD=""
    if [ "$(id -u)" -ne 0 ]; then
        if command -v sudo >/dev/null 2>&1; then
            SUDO_CMD="sudo"
        else
            echo "Error: Cannot install dependencies without root or sudo. Please install ${missing[*]} manually." >&2
            return 1
        fi
    fi
    $SUDO_CMD apt-get update -qq && $SUDO_CMD apt-get install -y -qq "${missing[@]}" 2>/dev/null || \
    $SUDO_CMD yum install -y -q "${missing[@]}" 2>/dev/null || {
        echo "Failed to install dependencies: ${missing[*]}" >&2
        return 1
    }
}

ensure_dependencies

# 下载前确认 /tmp 空间
TMP_AVAIL_KB=$(available_kb /tmp)
if [ "${TMP_AVAIL_KB}" -lt 65536 ]; then
    echo "Insufficient disk space on /tmp: ${TMP_AVAIL_KB}KB available (need at least 64MB before download)" >&2
    exit 1
fi

mkdir -p "${SKILLS_DIR}"
TMP_ZIP=$(mktemp "/tmp/skill-ace-${SKILL_SLUG}-${SKILL_VERSION}.XXXXXX")

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

# 包内容基础校验：至少有一个文件（与 batch_install_skills_from_smh_ace.sh 对齐，
# 不强制要求 SKILL.md；ACE 生态对包结构更宽松）
shopt -s dotglob nullglob
FILE_COUNT=$(find "${STAGING_DIR}" -mindepth 1 -maxdepth 1 | wc -l)
shopt -u dotglob nullglob
if [ "${FILE_COUNT}" -eq 0 ]; then
    echo "Invalid skill package: extracted archive is empty" >&2
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

# ===== 写入 lightclaw.json：启用该 skill =====
# 策略：
#   - 若 config 不存在：仅打印告警（lightclaw 首次启动会自动生成），skill 文件已落盘不阻断
#   - 若 config 存在：jq 合并后原子写回（mv 保证 rename 原子性）；失败则回滚文件目录
if [ -f "${CONFIG}" ]; then
    CONFIG_BACKUP="${CONFIG}.bak.$$"
    cp -p "${CONFIG}" "${CONFIG_BACKUP}"

    CONFIG_TMP=$(mktemp "/tmp/lightclaw-config-${SKILL_SLUG}.XXXXXX.json")
    if jq --arg s "${SKILL_SLUG}" \
        '.skills = (.skills // {})
         | .skills.entries = (.skills.entries // {})
         | .skills.entries[$s] = (.skills.entries[$s] // {}) + {enabled: true}' \
        "${CONFIG}" > "${CONFIG_TMP}"; then
        # 原子替换
        if mv "${CONFIG_TMP}" "${CONFIG}"; then
            rm -f "${CONFIG_BACKUP}"
            CONFIG_BACKUP=""
        else
            echo "Failed to write ${CONFIG}; keeping backup at ${CONFIG_BACKUP}" >&2
            rm -f "${CONFIG_TMP}" 2>/dev/null || true
            # 回滚文件目录，保持"全有或全无"语义
            rm -rf "${INSTALL_DIR}"
            if [ -n "${BACKUP_DIR}" ] && [ -d "${BACKUP_DIR}" ]; then
                mv "${BACKUP_DIR}" "${INSTALL_DIR}" || true
                CLEANUP_BACKUP=1
            fi
            exit 1
        fi
    else
        echo "jq failed to merge ${CONFIG}; keeping backup at ${CONFIG_BACKUP}" >&2
        rm -f "${CONFIG_TMP}" 2>/dev/null || true
        # 回滚文件目录
        rm -rf "${INSTALL_DIR}"
        if [ -n "${BACKUP_DIR}" ] && [ -d "${BACKUP_DIR}" ]; then
            mv "${BACKUP_DIR}" "${INSTALL_DIR}" || true
            CLEANUP_BACKUP=1
        fi
        exit 1
    fi
else
    echo "warn: ${CONFIG} not found; skipped enabled flag (skill files ready, will be picked up on first launch)" >&2
fi

# 配置写入成功，清理备份目录
if [ -n "${BACKUP_DIR}" ] && [ -d "${BACKUP_DIR}" ]; then
    rm -rf "${BACKUP_DIR}"
fi
BACKUP_DIR=""
CLEANUP_BACKUP=1

# 注意：有意不重启 lightclaw——单技能热下发不应中断正在进行的对话。
# 依赖 ACE 自身的 skill watcher 或下次会话重新加载配置。
echo "Skill installed: ${SKILL_SLUG}@${SKILL_VERSION} → ${INSTALL_DIR}"
echo "ℹ lightclaw 未重启；新技能将在下次会话或 watcher 触发时生效"
