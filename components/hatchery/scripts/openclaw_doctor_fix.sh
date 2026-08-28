#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u)"

# ========== 龙虾医院 一键修复脚本 ==========
# 功能：先备份 ~/.openclaw/openclaw.json 到 /tmp/（纯本地 cp），再执行 openclaw doctor --fix --yes

# ========== 路径配置 ==========
OPENCLAW_HOME="${OPENCLAW_STATE_DIR:-$HOME/.openclaw}"
CONFIG_FILE="${OPENCLAW_HOME}/openclaw.json"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_PATH="/tmp/openclaw.json.bak.${TIMESTAMP}"

echo "=== 龙虾医院 一键修复 ==="
echo "状态目录: $OPENCLAW_HOME"

# ========== 第一步：本地备份配置文件 ==========
echo "正在备份配置文件..."
if [ -f "$CONFIG_FILE" ]; then
    cp "$CONFIG_FILE" "$BACKUP_PATH"
    echo "✓ 配置文件已备份: $BACKUP_PATH"
else
    echo "⚠ 配置文件不存在，跳过备份"
fi

# ========== 第二步：执行修复 ==========
echo "正在执行 openclaw doctor --fix --yes ..."
if ! command -v openclaw &>/dev/null; then
    echo "✗ openclaw 命令不存在"
    exit 1
fi

openclaw doctor --fix --yes 2>&1

echo ""
echo "=== 一键修复完成 ==="
echo "如需回退，本地备份位于: $BACKUP_PATH"
