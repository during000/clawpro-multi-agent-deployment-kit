#!/bin/bash
# ensure_memory_plugin_hermes.sh — Hermes 场景下的记忆插件就绪校验脚本。
# 与 OpenClaw 的 ensure_memory_plugin.sh 语义不同：
#   - 仅做文件存在性校验，不尝试自动安装/升级
#   - Hermes 插件生命周期由后端 install_hermes_tdai_gateway.sh 闭环管控
#   - 失败直接退出，不自愈（避免无效的 TAT 循环）
set -uo pipefail

export NO_COLOR=1

TDAI_DIR="$HOME/.memory-tencentdb/tdai-memory-openclaw-plugin"
BIN_PATH="$TDAI_DIR/bin/read-local-memory.mjs"
DATA_DIR="$HOME/.memory-tencentdb/memory-tdai"

log() { echo "[ensure_plugin_hermes] $*"; }

# ========== 1. 校验插件根目录 ==========
if [ ! -d "$TDAI_DIR" ]; then
  log "ERROR: Hermes plugin directory not found at $TDAI_DIR"
  log "HINT: plugin install may have failed, please re-trigger switch_free/switch_pro"
  exit 1
fi

# ========== 2. 校验 read-local-memory.mjs ==========
if [ ! -f "$BIN_PATH" ]; then
  log "ERROR: read-local-memory.mjs not found at $BIN_PATH"
  log "HINT: plugin may be corrupted or outdated, please re-trigger switch_free/switch_pro"
  exit 2
fi

# ========== 3. 校验数据目录（提示性，不强制失败） ==========
if [ ! -d "$DATA_DIR" ]; then
  log "WARN: data directory not found at $DATA_DIR (no conversations yet)"
fi

log "read-local-memory.mjs available"
echo "ok"
exit 0
