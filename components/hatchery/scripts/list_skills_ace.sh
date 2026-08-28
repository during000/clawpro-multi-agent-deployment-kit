#!/bin/bash
# list_skills_ace.sh
# 列出 LightClaw ACE 已启用的 skills（内建）。
# 契约：stdout 末行输出 JSON 数组：
#   [{"slug": "cron", "name": "cron", "description": "", "eligible": true, "can_uninstall": false}, ...]
# 与 scripts/list_skills.sh（openclaw）的输出形状保持一致，便于 Go 层统一消费。
#
# 实现：解析 `lightclaw skills list` 的文本输出（目前 lightclaw CLI 未提供 --json），
# 仅保留 "enabled" 状态的 skill。所有诊断信息走日志文件，stdout 保持干净。

set -uo pipefail
export PATH="$HOME/.lightclaw/bin:$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u 2>/dev/null || echo 0)

LOG_DIR="${HOME}/.lightclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/list_skills_ace.log"
exec 2>>"$LOG_FILE"
echo "" >>"$LOG_FILE"
echo "========== $(date '+%Y-%m-%d %H:%M:%S') list_skills_ace ==========" >>"$LOG_FILE"

if ! command -v lightclaw >/dev/null 2>&1; then
    echo "lightclaw command not found" >>"$LOG_FILE"
    echo "[]" | gzip -c | base64
    exit 0
fi

raw=$(lightclaw skills list 2>>"$LOG_FILE" || true)
if [ -z "$raw" ]; then
    echo "lightclaw skills list returned empty" >>"$LOG_FILE"
    echo "[]" | gzip -c | base64
    exit 0
fi

LIGHTCLAW_SKILLS_DIR="${LIGHTCLAW_HOME:-$HOME/.lightclaw}/workspace/skills"

resolve_skill_slug() {
    local display_name="$1"
    local skills_dir="$LIGHTCLAW_SKILLS_DIR"
    if [ -d "$skills_dir/$display_name" ]; then
        printf '%s' "$display_name"
        return
    fi
    for skill_dir in "$skills_dir"/*/; do
        [ -f "$skill_dir/SKILL.md" ] || continue
        local manifest_name
        manifest_name=$(sed -n 's/^name:[[:space:]]*//p' "$skill_dir/SKILL.md" | sed -n '1p')
        manifest_name=${manifest_name#\"}
        manifest_name=${manifest_name%\"}
        if [ "$manifest_name" = "$display_name" ]; then
            basename "$skill_dir"
            return
        fi
    done
    printf '%s' "$display_name"
}

# 解析表格输出，提取 enabled 行
# 格式：
#   ──────────
#     name                  source    ✓ enabled
#     name2                 builtin   ✗ disabled
#   ──────────
entries=""
while IFS= read -r line; do
    line_trimmed="$(echo "$line" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
    case "$line_trimmed" in
        ""|─*|Skill*|Total*|*──*) continue ;;
    esac
    name=$(echo "$line_trimmed" | awk '{print $1}')
    source=$(echo "$line_trimmed" | awk '{print $2}')
    status_rest=$(echo "$line_trimmed" | awk '{for(i=3;i<=NF;i++) printf $i" "; print ""}')
    if [ -n "$name" ] && echo "$status_rest" | grep -qi "enabled"; then
        # 转义 name 中的特殊字符（保守处理：反斜杠+双引号）
        safe_name=$(printf '%s' "$name" | sed 's/\\/\\\\/g; s/"/\\"/g')
        slug=$(resolve_skill_slug "$name")
        safe_slug=$(printf '%s' "$slug" | sed 's/\\/\\\\/g; s/"/\\"/g')
        if [ "$source" != "builtin" ] && [ -d "$LIGHTCLAW_SKILLS_DIR/$slug" ]; then
            can_uninstall=true
        else
            can_uninstall=false
        fi
        item="{\"slug\":\"${safe_slug}\",\"name\":\"${safe_name}\",\"description\":\"\",\"eligible\":true,\"can_uninstall\":${can_uninstall}}"
        if [ -z "$entries" ]; then
            entries="$item"
        else
            entries="${entries},${item}"
        fi
    fi
done <<EOL
$raw
EOL

# 生成 JSON、gzip 到临时文件，根据大小决定输出策略
gzip_tmp=$(mktemp -t ace_skills_gz.XXXXXX)
echo "[${entries}]" | gzip > "$gzip_tmp"
gz_size=$(wc -c < "$gzip_tmp" | tr -d ' ')
if [ "$gz_size" -le 16000 ] && [ "$gz_size" -gt 0 ]; then
    base64 < "$gzip_tmp"
    rm -f "$gzip_tmp"
else
    chunks=$(( (gz_size + 15999) / 16000 ))
    echo "{\"mode\":\"file\",\"path\":\"$gzip_tmp\",\"size\":$gz_size,\"chunks\":$chunks}"
fi
