#!/bin/bash
set -euo pipefail

# uninstall_skill_hermes.sh
# 从 Hermes 实例上卸载技能
#
# 参数（通过 TAT 模板变量注入）：
#   {{skill_slug}} - 技能 slug

# %INCLUDE% lib_acli_compat.sh

SKILL_SLUG="{{skill_slug}}"

# ===== 参数安全校验 =====

if [[ "${SKILL_SLUG}" == *"/"* ]] || [[ "${SKILL_SLUG}" == *"\\"* ]] || [[ "${SKILL_SLUG}" == *".."* ]]; then
    echo "Invalid skill slug: ${SKILL_SLUG}" >&2
    exit 1
fi

if [ -z "${SKILL_SLUG}" ]; then
    echo "skill_slug is empty" >&2
    exit 1
fi

# ===== 脚本级探测一次 =====
_acli_mode="$(ensure_acli 2>/dev/null)"

# ===== acli 路径 =====
# acli skills uninstall 输出：{"success":true,"data":{"name":"xxx","status":"uninstalled"}}
if [ "$_acli_mode" = "acli" ]; then
    if acli skills uninstall "$SKILL_SLUG" 2>/dev/null; then
        echo "Uninstalled skill: ${SKILL_SLUG} (acli)"
        exit 0
    fi
    echo "acli uninstall 未找到技能，尝试直接删除目录..."
fi

# ===== fallback: 直接删除目录 =====
SKILLS_DIR="$HOME/.hermes/skills"
INSTALL_DIR="${SKILLS_DIR}/${SKILL_SLUG}"
AGENTS_SKILLS_DIR="$HOME/.agents/skills"
AGENTS_INSTALL_DIR="${AGENTS_SKILLS_DIR}/${SKILL_SLUG}"

# 幂等处理：两个目录都不存在视为成功
if [ ! -d "${INSTALL_DIR}" ] && [ ! -d "${AGENTS_INSTALL_DIR}" ]; then
    echo "Skill not installed: ${SKILL_SLUG} (directory does not exist)"
    exit 0
fi

if [ -d "${INSTALL_DIR}" ]; then
    rm -rf "${INSTALL_DIR}"
    echo "Uninstalled skill: ${SKILL_SLUG} (removed ${INSTALL_DIR})"
fi

# Harness CLI 使用 ~/.agents/skills；同 slug 可能只存在于此目录或同时存在于两个目录。
if [ -d "${AGENTS_INSTALL_DIR}" ]; then
    rm -rf "${AGENTS_INSTALL_DIR}"
    echo "Uninstalled skill: ${SKILL_SLUG} (removed ${AGENTS_INSTALL_DIR})"
fi
