#!/bin/bash
# get_version_info_ace.sh
# 获取 LightClaw ACE 主程序版本 + 内建技能列表，输出 JSON。
# 输出契约（与 openclaw 版一致）：
#   {"agent_version":"0.1.1","agent_type":"lightclawace","plugins":{"<skill_name>":"builtin",...}}
#
# ACE 特性：
#   - 主程序命令为 `lightclaw`（Python，通过 venv 启动），版本通过 `lightclaw --version` 获得
#     实际输出形如 "LightClaw v0.1.1"（从中抽 semver）
#   - ACE 没有 openclaw 那样的独立 extensions 目录；skills 内建在主程序内，
#     使用 `lightclaw skills list` 获取（人类可读输出，需要解析）
#   - 没有稳定的单条命令直接给出 skills JSON，故 plugins 字段填 skill_name -> "builtin"
#     （这与 openclaw 的 plugin 版本语义不完全一致；前端 BuildPluginVersionStatus 能接受该格式）

set -uo pipefail
export PATH="$HOME/.lightclaw/bin:$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u 2>/dev/null || echo 0)

# ========== 日志系统初始化 ==========
# 统一放到用户家目录，避免非 root 运行时 /var/log 无权
LOG_DIR="${HOME}/.lightclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
SCRIPT_NAME="get_version_info_ace"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# stderr 也重定向到日志，保持 stdout 干净（最后一行必须是合法 JSON）
exec 2>>"$LOG_FILE"
echo "" >>"$LOG_FILE"
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') ==========" >>"$LOG_FILE"

# ========== 1. 获取 ACE 主程序版本 ==========
echo ">>> [步骤 1/2] 获取 LightClaw ACE 主程序版本" >>"$LOG_FILE"

ACE_VERSION=""

# 优先级 1：调 `lightclaw --version`，输出形如 "LightClaw v0.1.1"
if command -v lightclaw >/dev/null 2>&1; then
    raw=$(lightclaw --version 2>>"$LOG_FILE" || true)
    # 提取 semver（支持 0.1.1 / 1.2.3 / 10.20.30 等）
    ACE_VERSION=$(echo "$raw" | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)
    if [ -n "$ACE_VERSION" ]; then
        echo "✓ 从 lightclaw --version 读取: $ACE_VERSION (raw=$raw)" >>"$LOG_FILE"
    fi
fi

# 优先级 2：直接调 venv python 模块读 __version__
if [ -z "$ACE_VERSION" ] && [ -x "$HOME/.lightclaw/venv/bin/python" ]; then
    ACE_VERSION=$("$HOME/.lightclaw/venv/bin/python" -c "import lightclaw; print(lightclaw.__version__)" 2>>"$LOG_FILE" | tr -d '[:space:]' || true)
    if [ -n "$ACE_VERSION" ]; then
        echo "✓ 从 venv python 读取: $ACE_VERSION" >>"$LOG_FILE"
    fi
fi

if [ -z "$ACE_VERSION" ]; then
    echo "✗ 无法获取 LightClaw ACE 版本" >>"$LOG_FILE"
fi

# ========== 2. 读取已启用 skills 列表 ==========
echo "" >>"$LOG_FILE"
echo ">>> [步骤 2/2] 读取已启用 skills 列表" >>"$LOG_FILE"

PLUGINS_JSON="{}"

# `lightclaw skills list` 输出为表格文本，带 ANSI 分隔线。格式示例：
#   ──────────────────────────────────────
#     Skill Name          Source    Status
#   ──────────────────────────────────────
#     cron                builtin   ✓ enabled
#     cvm-ai-doctor       builtin   ✓ enabled
#     ...
#   ──────────────────────────────────────
#     Total: 14 skills, 14 enabled, 0 disabled
#
# 解析策略：行首两个空格 + 单词 + 空格 + (builtin|custom|...) + 其余视为状态，
# 仅保留状态含 "enabled" 的行。
if command -v lightclaw >/dev/null 2>&1; then
    raw_list=$(lightclaw skills list 2>>"$LOG_FILE" || true)
    if [ -n "$raw_list" ]; then
        entries=""
        count=0
        while IFS= read -r line; do
            # 去掉首尾空白
            line_trimmed="$(echo "$line" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
            # 跳过分隔线 / 标题行 / 统计行
            case "$line_trimmed" in
                ""|─*|Skill*|Total*|*──*) continue ;;
            esac
            # 提取 3 个字段：name source status
            name=$(echo "$line_trimmed" | awk '{print $1}')
            source=$(echo "$line_trimmed" | awk '{print $2}')
            status_rest=$(echo "$line_trimmed" | awk '{for(i=3;i<=NF;i++) printf $i" "; print ""}')
            # 只收录 enabled
            if echo "$status_rest" | grep -qi "enabled"; then
                [ -z "$name" ] && continue
                # 用 source 当作版本值（"builtin" / "custom" 等）。前端主要消费 key 做列表；
                # 若后续需要版本号可在 lightclaw CLI 层加 JSON 输出后再回填。
                if [ -z "$entries" ]; then
                    entries="\"${name}\":\"${source}\""
                else
                    entries="${entries},\"${name}\":\"${source}\""
                fi
                count=$((count+1))
            fi
        done <<EOL
$raw_list
EOL
        PLUGINS_JSON="{${entries}}"
        echo "  共读取 $count 个已启用 skill" >>"$LOG_FILE"
    else
        echo "  ⚠ lightclaw skills list 输出为空" >>"$LOG_FILE"
    fi
else
    echo "  ⚠ lightclaw 命令不可用" >>"$LOG_FILE"
fi

# ========== 3. 输出 JSON ==========
# 严格遵守契约：stdout 最后一行必须是合法 JSON object。
# 其它诊断信息全部走 $LOG_FILE，避免污染 TAT 合并后的 output。
RESULT=$(printf '{"agent_version":"%s","agent_type":"lightclawace","plugins":%s}' \
  "${ACE_VERSION}" \
  "${PLUGINS_JSON}")
echo "$RESULT"
