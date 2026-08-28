#!/bin/bash
set -uo pipefail
# 注意：不用 set -e，因为单个技能安装失败不应终止整个批量流程

# batch_install_skills_from_smh.sh
# 从 SMH 批量下载技能 zip 包并安装到 OpenClaw 实例
#
# 参数（通过 TAT 模板变量注入）：
#   {{skills_list}} - 多行文本，每行一个技能，字段用 TAB 分隔：
#     download_url<TAB>skill_slug<TAB>skill_version
#
# 示例（2 个技能）：
#   https://smh.example.com/skill-a.zip?token=xxx	my-skill	1.0.0
#   https://smh.example.com/skill-b.zip?token=yyy	other-skill	2.1.0
#
# 输出：最后一行为 JSON 格式的安装结果汇总（方便服务端解析）

SKILLS_LIST_B64='{{skills_list}}'
SKILLS_LIST=$(echo "${SKILLS_LIST_B64}" | base64 -d)
SKILLS_DIR="$HOME/.openclaw/workspace/skills"
MAX_UNCOMPRESSED_BYTES=$((200 * 1024 * 1024))  # 单个技能解压上限 200MB

# ===== 结果收集 =====
# 格式：每个元素为 "slug|version|status|message"
RESULTS=()

add_result() {
    local slug="$1" version="$2" status="$3" message="$4"
    # 转义 message 中的特殊字符（双引号、反斜杠、换行）
    message="${message//\\/\\\\}"
    message="${message//\"/\\\"}"
    message="${message//$'\n'/\\n}"
    RESULTS+=("${slug}|${version}|${status}|${message}")
}

output_results() {
    local total=${#RESULTS[@]}
    local success=0
    local failed=0

    # 构建 JSON
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

available_kb() {
    df -Pk "$1" | awk 'NR==2{print $4}'
}

# ===== 前置校验 =====

ensure_dependencies() {
    local need_install=false
    for cmd in curl unzip; do
        if ! command -v "${cmd}" >/dev/null 2>&1; then
            need_install=true
            break
        fi
    done

    if [ "${need_install}" = true ]; then
        echo "Missing dependencies. Attempting installation..." >&2
        local SUDO_CMD=""
        if [ "$(id -u)" -ne 0 ]; then
            if command -v sudo >/dev/null 2>&1; then
                SUDO_CMD="sudo"
            else
                echo "Error: Cannot install dependencies without root or sudo." >&2
                exit 1
            fi
        fi
        $SUDO_CMD apt-get update -qq && $SUDO_CMD apt-get install -y -qq curl unzip 2>/dev/null || \
        $SUDO_CMD yum install -y -q curl unzip 2>/dev/null || {
            echo "Failed to install dependencies (curl, unzip)" >&2
            exit 1
        }
    fi
}

ensure_dependencies

# 解析输入：按行读取，每行 TAB 分隔为 download_url / skill_slug / skill_version
SKILL_URLS=()
SKILL_SLUGS=()
SKILL_VERSIONS=()

while IFS=$'\t' read -r url slug version; do
    # 跳过空行和注释行
    [ -z "${url}" ] && continue
    [[ "${url}" == \#* ]] && continue
    SKILL_URLS+=("${url}")
    SKILL_SLUGS+=("${slug}")
    SKILL_VERSIONS+=("${version}")
done <<< "${SKILLS_LIST}"

SKILL_COUNT=${#SKILL_SLUGS[@]}
if [ "${SKILL_COUNT}" -le 0 ]; then
    echo '{"results":[],"summary":{"total":0,"success":0,"failed":0},"error":"Invalid or empty skills_list"}'
    exit 1
fi

echo "Batch installing ${SKILL_COUNT} skill(s)..."

# 检查 /tmp 磁盘空间（批量场景需要更多空间）
TMP_AVAIL_KB=$(available_kb /tmp)
REQUIRED_TMP_KB=$((SKILL_COUNT * 65536))  # 每个技能预留 64MB
if [ "${TMP_AVAIL_KB}" -lt "${REQUIRED_TMP_KB}" ]; then
    echo "Warning: /tmp space may be tight: ${TMP_AVAIL_KB}KB available (ideal: ${REQUIRED_TMP_KB}KB for ${SKILL_COUNT} skills)" >&2
    # 不直接退出，至少保证有 64MB 给第一个技能
    if [ "${TMP_AVAIL_KB}" -lt 65536 ]; then
        echo '{"results":[],"summary":{"total":0,"success":0,"failed":0},"error":"Insufficient disk space on /tmp"}'
        exit 1
    fi
fi

mkdir -p "${SKILLS_DIR}"

# 构建 curl 重试参数（只检测一次）
CURL_RETRY_OPTS=(--retry 3 --retry-delay 2)
if curl --help all 2>/dev/null | grep -q -- '--retry-all-errors'; then
    CURL_RETRY_OPTS+=(--retry-all-errors)
fi

# ===== 逐个安装 =====

install_single_skill() {
    local download_url="$1"
    local skill_slug="$2"
    local skill_version="$3"

    local install_dir="${SKILLS_DIR}/${skill_slug}"
    local lock_file="/tmp/install-skill-${skill_slug}.lock"
    local tmp_zip=""
    local staging_dir=""
    local backup_dir=""
    local cleanup_backup=1
    local lock_fd=""

    # 单个技能的清理函数
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
    echo "--- [${skill_slug}@${skill_version}] Starting installation ---"

    # 参数安全校验
    if [[ "${skill_slug}" == *"/"* ]] || [[ "${skill_slug}" == *"\\"* ]] || [[ "${skill_slug}" == *".."* ]]; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Invalid skill slug"
        return
    fi

    if [[ "${skill_version}" == *"/"* ]] || [[ "${skill_version}" == *"\\"* ]] || [[ "${skill_version}" == *".."* ]]; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Invalid skill version"
        return
    fi

    # 并发锁（为每个技能分配独立的 fd：从 10 开始递增）
    local fd=$((10 + CURRENT_INDEX))
    eval "exec ${fd}> '${lock_file}'"
    if ! flock -n "${fd}"; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Another installation of ${skill_slug} is already in progress"
        return
    fi
    lock_fd="${fd}"

    # 下载 zip
    tmp_zip=$(mktemp "/tmp/skill-${skill_slug}-${skill_version}.XXXXXX")
    echo "[${skill_slug}] Downloading from SMH..."
    if ! curl -fsSL --connect-timeout 10 --max-time 30 \
        "${CURL_RETRY_OPTS[@]}" \
        -o "${tmp_zip}" "${download_url}"; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Download failed: curl error"
        skill_cleanup
        return
    fi

    if [ ! -s "${tmp_zip}" ]; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Download failed: empty file"
        skill_cleanup
        return
    fi
    echo "[${skill_slug}] Downloaded: $(du -h "${tmp_zip}" | cut -f1)"

    # zip 完整性校验
    if ! unzip -tq "${tmp_zip}" >/dev/null 2>&1; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Download corrupted: invalid zip file"
        skill_cleanup
        return
    fi

    # Zip Slip 防护
    local unsafe_entries
    unsafe_entries=$(unzip -Z -1 "${tmp_zip}" | grep -E '(^/|(^|/)\.\.(/|$))' || true)
    if [[ -n "${unsafe_entries}" ]]; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Dangerous paths detected in archive (possible Zip Slip attack)"
        skill_cleanup
        return
    fi

    # 解压大小检查
    local uncompressed_bytes
    uncompressed_bytes=$(unzip -qql "${tmp_zip}" | awk '{sum += $1} END {print sum+0}')
    if [ "${uncompressed_bytes}" -le 0 ]; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Invalid skill package: archive is empty"
        skill_cleanup
        return
    fi
    if [ "${uncompressed_bytes}" -gt "${MAX_UNCOMPRESSED_BYTES}" ]; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Invalid skill package: uncompressed size exceeds 200MB limit"
        skill_cleanup
        return
    fi

    # 目标磁盘空间检查
    local uncompressed_kb=$(((uncompressed_bytes + 1023) / 1024))
    local target_required_kb=$((uncompressed_kb + 10240))
    local target_avail_kb
    target_avail_kb=$(available_kb "${SKILLS_DIR}")
    if [ "${target_avail_kb}" -lt "${target_required_kb}" ]; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Insufficient disk space: ${target_avail_kb}KB available, need ${target_required_kb}KB"
        skill_cleanup
        return
    fi

    # 解压到 staging
    staging_dir=$(mktemp -d "${SKILLS_DIR}/.${skill_slug}.staging.XXXXXX")
    if ! unzip -qo "${tmp_zip}" -d "${staging_dir}"; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Failed to extract zip"
        skill_cleanup
        return
    fi

    # 顶级目录提升
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

    # 校验 SKILL.md
    local skill_md_found
    skill_md_found=$(find "${staging_dir}" -maxdepth 1 -iname "SKILL.md" -print -quit)
    if [ -z "${skill_md_found}" ]; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Invalid skill package: SKILL.md not found"
        skill_cleanup
        return
    fi

    # 安装路径检查
    if [ -e "${install_dir}" ] && [ ! -d "${install_dir}" ]; then
        add_result "${skill_slug}" "${skill_version}" "failed" "Install path exists but is not a directory: ${install_dir}"
        skill_cleanup
        return
    fi

    # 原子替换：备份旧版本 → mv staging → 清理
    if [ -d "${install_dir}" ]; then
        backup_dir="${SKILLS_DIR}/.${skill_slug}.backup.$$"
        mv "${install_dir}" "${backup_dir}"
        cleanup_backup=0
    fi

    if mv "${staging_dir}" "${install_dir}"; then
        staging_dir=""
    else
        # 安装失败，回滚
        if [ -n "${backup_dir}" ] && [ -d "${backup_dir}" ] && [ ! -e "${install_dir}" ]; then
            mv "${backup_dir}" "${install_dir}" || true
            cleanup_backup=1
        fi
        add_result "${skill_slug}" "${skill_version}" "failed" "Failed to activate new skill files"
        skill_cleanup
        return
    fi

    # 安装成功，清理备份
    if [ -n "${backup_dir}" ] && [ -d "${backup_dir}" ]; then
        rm -rf "${backup_dir}"
    fi
    backup_dir=""
    cleanup_backup=1

    echo "[${skill_slug}] Installed: ${skill_slug}@${skill_version} → ${install_dir}"
    add_result "${skill_slug}" "${skill_version}" "success" "Installed successfully"

    # 清理临时文件（锁在循环中保持，函数返回后释放）
    [ -n "${tmp_zip}" ] && rm -f "${tmp_zip}" 2>/dev/null || true
    tmp_zip=""
    if [ -n "${lock_fd}" ]; then
        eval "exec ${lock_fd}>&-" 2>/dev/null || true
        rm -f "${lock_file}" 2>/dev/null || true
        lock_fd=""
    fi
}

# ===== 主循环 =====

for ((i = 0; i < SKILL_COUNT; i++)); do
    CURRENT_INDEX=${i}
    DOWNLOAD_URL="${SKILL_URLS[${i}]}"
    SKILL_SLUG="${SKILL_SLUGS[${i}]}"
    SKILL_VERSION="${SKILL_VERSIONS[${i}]}"

    # 跳过无效条目
    if [ -z "${SKILL_SLUG}" ] || [ -z "${DOWNLOAD_URL}" ]; then
        add_result "${SKILL_SLUG:-unknown}" "${SKILL_VERSION:-unknown}" "failed" "Missing required fields (download_url or skill_slug)"
        continue
    fi

    install_single_skill "${DOWNLOAD_URL}" "${SKILL_SLUG}" "${SKILL_VERSION}"
done

# ===== 输出结果 =====

output_results
