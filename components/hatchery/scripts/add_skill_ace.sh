#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化 ==========
# 注意：ACE 未加入 hatchery 的 rootRequiredTATScripts 白名单（controller/tat.go），
# 由 TAT 以实例普通用户身份执行，无权写 /var/log/clawpro。
SCRIPT_NAME="add_skill_ace"
LOG_DIR="$HOME/.lightclaw/logs"
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# Parameters:
#   {{skill_name}} - skill slug（唯一参数，与 openclaw add_skill.sh 对齐）
#
# 安装策略（对齐 openclaw add_skill.sh，从 clawhub 安装）：
#   1) clawhub CLI install（如果 clawhub 存在）
#   2) skillhub --dir install（skillhub CLI）
#   3) clawhub.ai API 直接下载（用 curl + jq 模拟 clawhub install，核心路径）
#   4) lightclaw HTTP API fallback
#   5) lightmake.site 公共源 zip 兜底

SKILL="{{skill_name}}"
LIGHTCLAW_PORT="${LIGHTCLAW_PORT:-80}"
CLAWHUB_API="https://clawhub.ai/api/v1"

# ========== 探测真正的 lightclaw 安装用户 ==========
# 如果以 root 身份执行但 root 下没有 .lightclaw，自动查找真正的安装用户。
# ACE 镜像默认安装在 agentuser 下，DB 里 runtime_user 可能还未正确写入。
resolve_lightclaw_home() {
    # 优先：当前用户下有 .lightclaw
    if [ -d "$HOME/.lightclaw" ]; then
        echo "$HOME"
        return
    fi
    # 以 root 执行时，扫描 /home/* 找安装了 lightclaw 的用户
    if [ "$(id -u)" = "0" ]; then
        for user_home in /home/*; do
            if [ -d "$user_home/.lightclaw" ]; then
                echo "$user_home"
                return
            fi
        done
    fi
    # 都没找到，回退到 $HOME（后续会 mkdir）
    echo "$HOME"
}

TARGET_HOME="$(resolve_lightclaw_home)"
LIGHTCLAW_HOME="${TARGET_HOME}/.lightclaw"
SKILLS_DIR="$LIGHTCLAW_HOME/workspace/skills"
CONFIG="$LIGHTCLAW_HOME/lightclaw.json"

if [ "$TARGET_HOME" != "$HOME" ]; then
    echo "ℹ 当前用户 $(whoami)，lightclaw 安装在 ${TARGET_HOME}，使用该目录"
fi

# 参数校验
if [ -z "$SKILL" ] || [[ "$SKILL" == *"{{"*"}}"* ]]; then
    echo "✗ skill_name 参数未正确替换或为空: '${SKILL}'"
    exit 1
fi

mkdir -p "$SKILLS_DIR"

echo "=== ACE install skill: ${SKILL} ==="
echo "  SKILLS_DIR: ${SKILLS_DIR}"

# ========== 写入 lightclaw.json 启用 skill ==========
enable_skill_in_config() {
    local slug="$1"
    if [ -f "${CONFIG}" ] && command -v jq >/dev/null 2>&1; then
        local config_tmp
        config_tmp=$(mktemp "/tmp/lightclaw-config-${slug}.XXXXXX.json")
        if jq --arg s "${slug}" \
            '.skills = (.skills // {})
             | .skills.entries = (.skills.entries // {})
             | .skills.entries[$s] = (.skills.entries[$s] // {}) + {enabled: true}' \
            "${CONFIG}" > "${config_tmp}"; then
            mv "${config_tmp}" "${CONFIG}"
        else
            rm -f "${config_tmp}" 2>/dev/null || true
            echo "⚠ jq 写入 lightclaw.json 失败，技能文件已就位但未自动启用" >&2
        fi
    fi
}

# ========== 从 clawhub.ai API 下载技能（模拟 clawhub install） ==========
# clawhub.ai API 结构：
#   GET /api/v1/skills/{slug}
#     → { skill: { slug, ... }, latestVersion: { version: "3.0.0", ... } }
#   GET /api/v1/skills/{slug}/versions/{version}
#     → { version: { files: [{ path, size, ... }, ...] } }
#   GET /api/v1/skills/{slug}/file?path=xxx&version=xxx
#     → 原始文件内容（raw text，非 JSON）
install_from_clawhub_api() {
    local slug="$1"
    local install_dir="${SKILLS_DIR}/${slug}"

    echo "ℹ 尝试从 clawhub.ai API 安装: ${slug}"

    if ! command -v jq >/dev/null 2>&1; then
        echo "⚠ jq 未安装，无法解析 clawhub API 响应" >&2
        return 1
    fi

    # 1) 获取技能详情，取最新版本号
    local detail_url="${CLAWHUB_API}/skills/${slug}"
    echo "  GET ${detail_url}"
    local detail_file
    detail_file=$(mktemp "/tmp/clawhub-detail-${slug}.XXXXXX.json")

    if ! curl -fsSL --connect-timeout 10 --max-time 30 --retry 2 --retry-delay 2 \
        -H "Accept: application/json" -H "User-Agent: lightclaw-ace/1.0" \
        -o "${detail_file}" "${detail_url}"; then
        rm -f "${detail_file}" 2>/dev/null || true
        echo "✗ clawhub API 技能详情请求失败" >&2
        return 1
    fi

    # 从 latestVersion.version 取版本号
    local latest_version
    latest_version=$(jq -r '.latestVersion.version // empty' "${detail_file}" 2>/dev/null || true)

    # fallback: skill.tags.latest
    if [ -z "$latest_version" ]; then
        latest_version=$(jq -r '.skill.tags.latest // empty' "${detail_file}" 2>/dev/null || true)
    fi

    rm -f "${detail_file}" 2>/dev/null || true

    if [ -z "$latest_version" ]; then
        echo "✗ 无法从 clawhub API 获取最新版本号" >&2
        return 1
    fi

    echo "  最新版本: ${latest_version}"

    # 2) 获取版本详情，取文件列表
    local version_url="${CLAWHUB_API}/skills/${slug}/versions/${latest_version}"
    echo "  GET ${version_url}"
    local version_file
    version_file=$(mktemp "/tmp/clawhub-ver-${slug}.XXXXXX.json")

    if ! curl -fsSL --connect-timeout 10 --max-time 30 --retry 2 --retry-delay 2 \
        -H "Accept: application/json" -H "User-Agent: lightclaw-ace/1.0" \
        -o "${version_file}" "${version_url}"; then
        rm -f "${version_file}" 2>/dev/null || true
        echo "✗ clawhub API 版本详情请求失败" >&2
        return 1
    fi

    # 提取文件路径列表
    local file_paths
    file_paths=$(jq -r '.version.files[]?.path // empty' "${version_file}" 2>/dev/null || true)
    rm -f "${version_file}" 2>/dev/null || true

    if [ -z "$file_paths" ]; then
        echo "✗ clawhub API 版本详情中没有文件列表" >&2
        return 1
    fi

    # 3) 逐个文件下载（file API 返回原始文本，直接保存）
    local staging_dir
    staging_dir=$(mktemp -d "${SKILLS_DIR}/.${slug}.staging.XXXXXX")
    local file_count=0

    while IFS= read -r fpath; do
        [ -z "$fpath" ] && continue
        # URL encode 文件路径
        local encoded_path
        encoded_path=$(printf '%s' "$fpath" | sed 's/ /%20/g; s/!/%21/g; s/#/%23/g; s/\$/%24/g; s/&/%26/g; s/'\''/%27/g; s/(/%28/g; s/)/%29/g; s/+/%2B/g; s/,/%2C/g; s/:/%3A/g; s/;/%3B/g; s/=/%3D/g; s/?/%3F/g; s/@/%40/g; s/\[/%5B/g; s/\]/%5D/g')
        local file_url="${CLAWHUB_API}/skills/${slug}/file?path=${encoded_path}&version=${latest_version}"
        local dest="${staging_dir}/${fpath}"
        mkdir -p "$(dirname "${dest}")"
        echo "  下载: ${fpath}"
        if curl -fsSL --connect-timeout 10 --max-time 30 \
            -H "User-Agent: lightclaw-ace/1.0" \
            -o "${dest}" "${file_url}"; then
            file_count=$((file_count + 1))
        else
            echo "  ⚠ 下载文件失败: ${fpath}" >&2
        fi
    done <<< "$file_paths"

    if [ "$file_count" -eq 0 ]; then
        echo "✗ 所有文件下载失败" >&2
        rm -rf "${staging_dir}" 2>/dev/null || true
        return 1
    fi

    # 4) 原子替换安装目录
    local backup_dir=""
    if [ -d "${install_dir}" ]; then
        backup_dir="${SKILLS_DIR}/.${slug}.backup.$$"
        mv "${install_dir}" "${backup_dir}"
    fi

    if mv "${staging_dir}" "${install_dir}"; then
        [ -n "${backup_dir}" ] && [ -d "${backup_dir}" ] && rm -rf "${backup_dir}"
    else
        echo "✗ 激活技能文件失败" >&2
        rm -rf "${staging_dir}" 2>/dev/null || true
        if [ -n "${backup_dir}" ] && [ -d "${backup_dir}" ] && [ ! -e "${install_dir}" ]; then
            mv "${backup_dir}" "${install_dir}" || true
        fi
        return 1
    fi

    enable_skill_in_config "${slug}"
    echo "✓ clawhub API 安装成功: ${slug}@${latest_version} → ${install_dir} (${file_count} 个文件)"
    return 0
}

# ========== zip 下载安装 ==========
install_from_zip() {
    local url="$1"
    local slug="$2"
    local install_dir="${SKILLS_DIR}/${slug}"

    echo "ℹ 尝试 zip 下载安装: ${slug}"

    local tmp_zip
    tmp_zip=$(mktemp "/tmp/skill-ace-${slug}.XXXXXX")
    echo "  下载 zip..."
    if ! curl -fsSL --connect-timeout 10 --max-time 60 --retry 2 --retry-delay 2 \
        -o "${tmp_zip}" "${url}"; then
        rm -f "${tmp_zip}" 2>/dev/null || true
        echo "✗ zip 下载失败" >&2
        return 1
    fi

    if [ ! -s "${tmp_zip}" ]; then
        rm -f "${tmp_zip}" 2>/dev/null || true
        echo "✗ zip 下载为空文件" >&2
        return 1
    fi

    if ! unzip -tq "${tmp_zip}" >/dev/null 2>&1; then
        rm -f "${tmp_zip}" 2>/dev/null || true
        echo "✗ zip 文件损坏" >&2
        return 1
    fi

    # Zip Slip 防护
    local unsafe
    unsafe=$(unzip -Z -1 "${tmp_zip}" | grep -E '(^/|(^|/)\.\.(/|$))' || true)
    if [[ -n "${unsafe}" ]]; then
        rm -f "${tmp_zip}" 2>/dev/null || true
        echo "✗ zip 中存在危险路径" >&2
        return 1
    fi

    local staging_dir
    staging_dir=$(mktemp -d "${SKILLS_DIR}/.${slug}.staging.XXXXXX")
    unzip -qo "${tmp_zip}" -d "${staging_dir}"
    rm -f "${tmp_zip}" 2>/dev/null || true

    # 如果解压后只有一个顶级目录，将其内容提升一级
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

    local backup_dir=""
    if [ -d "${install_dir}" ]; then
        backup_dir="${SKILLS_DIR}/.${slug}.backup.$$"
        mv "${install_dir}" "${backup_dir}"
    fi

    if mv "${staging_dir}" "${install_dir}"; then
        [ -n "${backup_dir}" ] && [ -d "${backup_dir}" ] && rm -rf "${backup_dir}"
    else
        echo "✗ 激活技能文件失败" >&2
        rm -rf "${staging_dir}" 2>/dev/null || true
        if [ -n "${backup_dir}" ] && [ -d "${backup_dir}" ] && [ ! -e "${install_dir}" ]; then
            mv "${backup_dir}" "${install_dir}" || true
        fi
        return 1
    fi

    enable_skill_in_config "${slug}"
    echo "✓ zip 安装成功: ${slug} → ${install_dir}"
    return 0
}

# ── 按优先级尝试安装 ──────────────────────────────────────────────

# Strategy 1: clawhub CLI install（openclaw 实例有此 CLI，ACE 通常没有）
if command -v clawhub >/dev/null 2>&1; then
    echo "尝试 clawhub install ${SKILL}..."
    stderr=$(clawhub install "$SKILL" --force 2>&1 >/dev/null) && {
        echo "✓ skill ${SKILL} 安装完成 (clawhub CLI)"
        exit 0
    } || true
    echo "$stderr" >&2
    echo "⚠ clawhub CLI install 失败，尝试下一种方式..."
fi

# Strategy 2: skillhub --dir install
if command -v skillhub >/dev/null 2>&1; then
    echo "尝试 skillhub --dir $SKILLS_DIR install ${SKILL}..."
    if skillhub --dir "$SKILLS_DIR" install "${SKILL}" --force 2>&1; then
        echo "✓ skill ${SKILL} 安装成功 (skillhub)"
        exit 0
    fi
    echo "⚠ skillhub 安装失败，尝试下一种方式..."
fi

# Strategy 3: clawhub.ai API 直接下载（核心路径，替代 clawhub CLI）
# ACE 没有 clawhub CLI，用 curl + jq 直接调 clawhub.ai API：
#   详情 → 版本号 → 文件列表 → 逐文件下载原始内容
if install_from_clawhub_api "${SKILL}"; then
    exit 0
fi
echo "⚠ clawhub API 安装失败，尝试下一种方式..."

# Strategy 4: lightclaw HTTP API fallback
API_URL="http://127.0.0.1:${LIGHTCLAW_PORT}/api/skillhub/install"
echo "调用 HTTP API: POST ${API_URL} (slug=${SKILL})"

RESPONSE=$(curl -sS -o - -w "\n%{http_code}" \
    --connect-timeout 5 --max-time 120 \
    -X POST "${API_URL}" \
    -H "Content-Type: application/json" \
    -d "{\"slug\":\"${SKILL}\",\"force\":true}" 2>&1) && {
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
        echo "✓ skill ${SKILL} 安装成功 (HTTP API)"
        exit 0
    fi
    echo "⚠ API 返回 HTTP ${HTTP_CODE}"
} || {
    echo "⚠ curl 调用失败: ${RESPONSE}"
}

# Strategy 5: lightmake.site 公共源直接下载 zip（最终兜底）
echo "尝试从公共源直接下载 zip..."
PUBLIC_DOWNLOAD_URL="https://lightmake.site/api/v1/download?slug=${SKILL}"
if install_from_zip "${PUBLIC_DOWNLOAD_URL}" "${SKILL}"; then
    exit 0
fi

echo "✗ skill ${SKILL} 安装失败（所有方式均失败）"
exit 1
