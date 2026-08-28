#!/bin/bash
# get_version_info.sh
# 获取 openclaw 主程序版本 + 已安装插件版本，输出 JSON
# 输出格式：{"agent_version":"2026.3.28","agent_type":"openclaw","plugins":{"openclaw-qqbot":"1.6.7",...}}

set -uo pipefail
export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化 ==========
# 使用用户家目录，避免非 root 运行时无法写入 /var/log
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="get_version_info"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# 将 stdout 和 stderr 同时输出到终端和日志文件
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# ========== 1. 获取 openclaw 主程序版本 ==========
echo ""
echo ">>> [步骤 1/2] 获取 openclaw 主程序版本"

OPENCLAW_VERSION=""
CONFIG="$HOME/.openclaw/openclaw.json"

# 优先从 openclaw.json 中读取 meta.lastTouchedVersion（无需启动 Node.js，更快）
if [ -f "$CONFIG" ]; then
    OPENCLAW_VERSION=$(jq -r '.meta.lastTouchedVersion // empty' "$CONFIG" 2>/dev/null || true)
    if [ -n "$OPENCLAW_VERSION" ]; then
        echo "✓ 从 openclaw.json 读取版本: $OPENCLAW_VERSION"
    else
        echo "  openclaw.json 中无 meta.lastTouchedVersion，尝试执行命令获取"
    fi
else
    echo "  配置文件不存在: $CONFIG，尝试执行命令获取"
fi

# fallback：执行 openclaw --version（需启动 Node.js，较慢）
if [ -z "$OPENCLAW_VERSION" ]; then
    OPENCLAW_VERSION=$({ openclaw --version; openclaw version; } 2>&1 \
      | grep -oP '[0-9]{4}\.[0-9]+\.[0-9]+' | head -1 || true)
    if [ -n "$OPENCLAW_VERSION" ]; then
        echo "✓ 通过命令获取版本: $OPENCLAW_VERSION"
    else
        echo "✗ 无法获取 openclaw 版本，将输出空字符串"
    fi
fi

# ========== 2. 读取已安装插件版本 ==========
echo ""
echo ">>> [步骤 2/2] 读取已安装插件版本"

PLUGINS_JSON="{}"
EXTENSIONS_DIR="$HOME/.openclaw/extensions"

if [ -d "$EXTENSIONS_DIR" ]; then
    echo "  插件目录: $EXTENSIONS_DIR"
    plugin_entries=""
    plugin_count=0
    for pkg_json in "$EXTENSIONS_DIR"/*/package.json; do
        [ -f "$pkg_json" ] || continue
        # 取插件目录名作为 slug（与 openclaw.json 中的 installs key 一致）
        slug=$(basename "$(dirname "$pkg_json")")
        version=$(jq -r '.version // empty' "$pkg_json" 2>/dev/null || true)
        if [ -z "$version" ]; then
            echo "  ⚠ 跳过 $slug：无法读取版本"
            continue
        fi
        echo "  ✓ 插件: $slug @ $version"
        if [ -z "$plugin_entries" ]; then
            plugin_entries="\"${slug}\":\"${version}\""
        else
            plugin_entries="${plugin_entries},\"${slug}\":\"${version}\""
        fi
        plugin_count=$((plugin_count + 1))
    done
    PLUGINS_JSON="{${plugin_entries}}"
    echo "  共读取 $plugin_count 个插件"
else
    echo "  ⚠ 插件目录不存在: $EXTENSIONS_DIR"
fi

# ========== 3. 输出 JSON ==========
echo ""
echo ">>> [输出] 版本信息 JSON"
RESULT=$(printf '{"agent_version":"%s","agent_type":"openclaw","plugins":%s}' \
  "${OPENCLAW_VERSION}" \
  "${PLUGINS_JSON}")
echo "$RESULT"
