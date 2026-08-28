#!/bin/bash
set -euo pipefail

# uninstall_skill_ace.sh
# 从 Ace (LightClaw) 实例上卸载技能（删除技能目录）
#
# 与 uninstall_skill.sh 的差异：
#   - 安装目录: ~/.openclaw/workspace/skills → ${LIGHTCLAW_HOME}/workspace/skills
#
# 参数（通过 TAT 模板变量注入）：
#   {{skill_slug}} - 技能 slug

SKILL_SLUG="{{skill_slug}}"

# Ace 使用 LIGHTCLAW_HOME 环境变量确定安装目录
if [ -z "${LIGHTCLAW_HOME:-}" ]; then
    LIGHTCLAW_HOME="$HOME/.lightclaw"
fi

SKILLS_DIR="${LIGHTCLAW_HOME}/workspace/skills"
INSTALL_DIR="${SKILLS_DIR}/${SKILL_SLUG}"

# ===== 参数安全校验 =====

if [[ "${SKILL_SLUG}" == *"/"* ]] || [[ "${SKILL_SLUG}" == *"\\"* ]] || [[ "${SKILL_SLUG}" == *".."* ]]; then
    echo "Invalid skill slug: ${SKILL_SLUG}" >&2
    exit 1
fi

if [ -z "${SKILL_SLUG}" ]; then
    echo "skill_slug is empty" >&2
    exit 1
fi

# ===== 幂等处理：目录不存在视为成功 =====

if [ ! -d "${INSTALL_DIR}" ]; then
    echo "Skill not installed: ${SKILL_SLUG} (directory does not exist)"
    exit 0
fi

# ===== 执行卸载 =====

rm -rf "${INSTALL_DIR}"
echo "Uninstalled skill: ${SKILL_SLUG} (removed ${INSTALL_DIR})"
