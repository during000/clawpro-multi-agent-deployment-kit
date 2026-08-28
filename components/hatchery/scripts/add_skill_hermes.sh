#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:/usr/local/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# %INCLUDE% lib_acli_compat.sh

# ========== 日志系统初始化 ==========
# 注意：hermes add_skill 由 TAT 以实例普通用户身份执行，无权写 /var/log/clawpro，
# 日志统一落到用户家目录，避免 Permission denied。
SCRIPT_NAME="add_skill_hermes"
LOG_DIR="$HOME/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || LOG_DIR="/tmp"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# Parameters:
#   {{skill_name}} - skill slug（唯一参数，与 openclaw add_skill.sh 对齐）
#
# 安装策略（从 clawhub 安装，对齐 ACE/openclaw）：
#   1) clawhub CLI install（如果 clawhub 存在）
#   2) harness skills install
#   3) skillhub --dir install
#   4) clawhub.ai API 直接下载（curl + jq，核心路径）
#   5) lightmake.site 公共源 zip 兜底

SKILL="{{skill_name}}"
CLAWHUB_API="https://clawhub.ai/api/v1"

# ========== 探测真正的 hermes 安装用户 ==========
# add_skill_hermes.sh 在白名单里以 root 执行，$HOME=/root，
# 但 hermes 实际安装在业务账户下（v0.0.11 及以前为 agentuser，v0.0.12 起切到 ubuntu），
# 不能硬编码账户名，需要扫 /home/* 找到真正的安装目录。
resolve_hermes_home() {
    if [ -d "$HOME/.hermes" ]; then
        echo "$HOME"
        return
    fi
    if [ "$(id -u)" = "0" ]; then
        for user_home in /home/*; do
            if [ -d "$user_home/.hermes" ]; then
                echo "$user_home"
                return
            fi
        done
    fi
    echo "$HOME"
}

TARGET_HOME="$(resolve_hermes_home)"
SKILLS_DIR="${TARGET_HOME}/.hermes/skills"

if [ "$TARGET_HOME" != "$HOME" ]; then
    echo "ℹ 当前用户 $(whoami)，hermes 安装在 ${TARGET_HOME}，使用该目录"
fi

# 参数校验
if [ -z "$SKILL" ] || [[ "$SKILL" == *"{{"*"}}"* ]]; then
    echo "✗ skill_name 参数未正确替换或为空: '${SKILL}'"
    exit 1
fi

if [[ ! "$SKILL" =~ ^[a-zA-Z0-9_-]+$ ]]; then
    echo "✗ skill_name 参数包含非法字符，只允许字母、数字、连字符(-)和下划线(_): '${SKILL}'"
    exit 1
fi

mkdir -p "$SKILLS_DIR"

echo "=== Hermes install skill: ${SKILL} ==="
echo "  SKILLS_DIR: ${SKILLS_DIR}"

# ========== 拉取/更新 harness CLI ==========
ensure_harness_cli() {
    local HARNESS_URL="https://agentone-1388575217.cos.ap-guangzhou.myqcloud.com/harness-linux-amd64"
    local HARNESS_BIN="${TARGET_HOME}/.local/bin/harness"
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
            echo "✗ harness CLI 下载失败且本地无已有版本" >&2
            return 1
        fi
    fi
}

# ========== 确保 skillhub CLI 可用 ==========
ensure_skillhub_cli() {
    if command -v skillhub >/dev/null 2>&1; then
        echo "ℹ skillhub CLI 已存在: $(command -v skillhub)"
        return 0
    fi
    echo "ℹ skillhub CLI 未找到，开始安装..."
    timeout --kill-after=10 240 bash -c \
        'curl -fsSL https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/install/install.sh | bash' 2>&1 || true
    hash -r 2>/dev/null || true
    if command -v skillhub >/dev/null 2>&1; then
        echo "✓ skillhub CLI 安装成功: $(command -v skillhub)"
        return 0
    fi
    echo "✗ skillhub CLI 安装失败" >&2
    return 1
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
        -H "Accept: application/json" -H "User-Agent: hermes-agent/1.0" \
        -o "${detail_file}" "${detail_url}"; then
        rm -f "${detail_file}" 2>/dev/null || true
        echo "✗ clawhub API 技能详情请求失败" >&2
        return 1
    fi

    local latest_version
    latest_version=$(jq -r '.latestVersion.version // empty' "${detail_file}" 2>/dev/null || true)

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
        -H "Accept: application/json" -H "User-Agent: hermes-agent/1.0" \
        -o "${version_file}" "${version_url}"; then
        rm -f "${version_file}" 2>/dev/null || true
        echo "✗ clawhub API 版本详情请求失败" >&2
        return 1
    fi

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
        local encoded_path
        encoded_path=$(printf '%s' "$fpath" | sed 's/ /%20/g; s/!/%21/g; s/#/%23/g; s/\$/%24/g; s/&/%26/g; s/'\''/%27/g; s/(/%28/g; s/)/%29/g; s/+/%2B/g; s/,/%2C/g; s/:/%3A/g; s/;/%3B/g; s/=/%3D/g; s/?/%3F/g; s/@/%40/g; s/\[/%5B/g; s/\]/%5D/g')
        local file_url="${CLAWHUB_API}/skills/${slug}/file?path=${encoded_path}&version=${latest_version}"
        local dest="${staging_dir}/${fpath}"
        mkdir -p "$(dirname "${dest}")"
        echo "  下载: ${fpath}"
        if curl -fsSL --connect-timeout 10 --max-time 30 \
            -H "User-Agent: hermes-agent/1.0" \
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

    # 尝试 reload（非关键路径）
    if command -v harness >/dev/null 2>&1; then
        harness skills reload >/dev/null 2>&1 && echo "✓ harness skills reload triggered" || \
            echo "ℹ harness skills reload not available; relying on watcher"
    fi

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
    tmp_zip=$(mktemp "/tmp/skill-hermes-${slug}.XXXXXX")
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

    # 尝试 reload（非关键路径）
    if command -v harness >/dev/null 2>&1; then
        harness skills reload >/dev/null 2>&1 && echo "✓ harness skills reload triggered" || \
            echo "ℹ harness skills reload not available; relying on watcher"
    fi

    echo "✓ zip 安装成功: ${slug} → ${install_dir}"
    return 0
}

# ── 确保 CLI 工具可用 ─────────────────────────────────────────────

# ===== 脚本级探测一次，后续复用 =====
_acli_mode="$(ensure_acli 2>>"$LOG_FILE")"

# 用于在所有策略都失败后输出"Go 侧关键字"
_FAIL_KEYWORD=""

# ── 按优先级尝试安装 ──────────────────────────────────────────────

# Strategy 0: acli skills install --name（最高优先级）
if [ "$_acli_mode" = "acli" ]; then
    echo "尝试 acli skills install ${SKILL}..."
    _acli_out=""
    _acli_exit=0
    _acli_out="$(acli skills install --name "$SKILL" 2>&1)" || _acli_exit=$?
    echo "$_acli_out"
    if [ "$_acli_exit" -eq 0 ]; then
        echo "✓ skill ${SKILL} 安装完成 (acli)"
        exit 0
    fi
    # acli 失败，解析错误关键字，继续尝试下一种方式
    if command -v jq >/dev/null 2>&1; then
        _acli_err="$(printf '%s' "$_acli_out" | jq -r '.error.message // empty' 2>/dev/null || true)"
    else
        _acli_err="$_acli_out"
    fi
    if [ -n "${_acli_err:-}" ]; then
        if printf '%s' "$_acli_err" | grep -qiE 'not found|no such skill'; then
            _FAIL_KEYWORD="Skill not found"
        elif printf '%s' "$_acli_err" | grep -qiE 'rate limit'; then
            _FAIL_KEYWORD="Rate limit exceeded"
        fi
    fi
    echo "⚠ acli skills install 失败 (exit=$_acli_exit)，尝试下一种方式..."
fi

# acli 失败或不可用，准备 fallback CLI 工具
ensure_harness_cli || true
ensure_skillhub_cli || true

# Strategy 1: clawhub CLI install（如果存在）
if command -v clawhub >/dev/null 2>&1; then
    echo "尝试 clawhub install ${SKILL}..."
    stderr=$(clawhub install "$SKILL" --force 2>&1 >/dev/null) && {
        echo "✓ skill ${SKILL} 安装完成 (clawhub CLI)"
        exit 0
    } || true
    echo "$stderr" >&2
    echo "⚠ clawhub CLI install 失败，尝试下一种方式..."
fi

# Strategy 2: harness skills install
if command -v harness >/dev/null 2>&1; then
    echo "尝试 harness skills install ${SKILL}..."
    if harness skills install "$SKILL" 2>&1; then
        echo "✓ skill ${SKILL} 安装完成 (harness)"
        exit 0
    fi
    echo "⚠ harness skills install 失败，尝试下一种方式..."
fi

# Strategy 3: skillhub --dir install
if command -v skillhub >/dev/null 2>&1; then
    echo "尝试 skillhub --dir $SKILLS_DIR install ${SKILL}..."
    if skillhub --dir "$SKILLS_DIR" install "$SKILL" --force 2>&1; then
        echo "✓ skill ${SKILL} 安装完成 (skillhub)"
        if command -v harness >/dev/null 2>&1; then
            harness skills reload >/dev/null 2>&1 || true
        fi
        exit 0
    fi
    echo "⚠ skillhub install 失败，尝试下一种方式..."
fi

# Strategy 4: clawhub.ai API 直接下载（核心路径，替代 clawhub CLI）
# Hermes 没有 clawhub CLI，用 curl + jq 直接调 clawhub.ai API：
#   详情 → 版本号 → 文件列表 → 逐文件下载原始内容
if install_from_clawhub_api "${SKILL}"; then
    exit 0
fi
echo "⚠ clawhub API 安装失败，尝试下一种方式..."

# Strategy 5: lightmake.site 公共源直接下载 zip（最终兜底）
echo "尝试从公共源直接下载 zip..."
PUBLIC_DOWNLOAD_URL="https://lightmake.site/api/v1/download?slug=${SKILL}"
if install_from_zip "${PUBLIC_DOWNLOAD_URL}" "${SKILL}"; then
    exit 0
fi

echo "✗ skill ${SKILL} 安装失败（所有方式均失败）"
if [ -n "${_FAIL_KEYWORD:-}" ]; then
    if [ "$_FAIL_KEYWORD" = "Skill not found" ]; then
        echo "Skill not found: ${SKILL}" >&2
    else
        echo "$_FAIL_KEYWORD" >&2
    fi
fi
exit 1
