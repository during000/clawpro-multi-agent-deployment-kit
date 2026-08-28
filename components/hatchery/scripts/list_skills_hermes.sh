#!/bin/bash
# list_skills_hermes.sh
# 列出 Hermes 已安装的 skills。
# 契约：stdout 末行输出 JSON 数组：
#   [{"slug": "xxx", "name": "xxx", "description": "", "eligible": true, "can_uninstall": true}, ...]
# 与 scripts/list_skills.sh（openclaw）的输出形状保持一致。
#
# 实现：直接扫描技能目录（兼容 ~/.hermes/skills/ 和 ~/.agents/skills/ 两个目录）。
# harness CLI 的 skills list 走的是 ~/.agents/skills/，
# 但 hatchery 脚本安装技能到 ~/.hermes/skills/，两者都需要扫。

set -uo pipefail
export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u 2>/dev/null || echo 0)

LOG_DIR="${HOME}/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/list_skills_hermes.log"
exec 2>>"$LOG_FILE"
echo "" >>"$LOG_FILE"
echo "========== $(date '+%Y-%m-%d %H:%M:%S') list_skills_hermes ==========" >>"$LOG_FILE"

# 注意：acli skills list 依赖 skillhub 注册表，而 hatchery 安装的技能不走 skillhub 注册，
# 所以不使用 acli，直接用目录扫描。

# ===== 原目录扫描兜底 =====

# 扫描目录收集技能名称（去重）
# 注意：不使用 declare -A 关联数组，因为 Bash 3.x 不支持。
# 改用换行分隔的字符串 + grep 去重，兼容所有 Bash 版本。
skill_names=""

scan_skills_dir() {
    local dir="$1"
    if [ ! -d "$dir" ]; then
        return
    fi
    for skill_dir in "$dir"/*/; do
        [ -d "$skill_dir" ] || continue
        local name
        name=$(basename "$skill_dir")
        # 跳过隐藏目录和临时目录（staging/backup）
        case "$name" in .*) continue ;; esac
        # 至少要有 SKILL.md 或任意文件才算有效技能
        if [ -f "$skill_dir/SKILL.md" ] || [ "$(ls -A "$skill_dir" 2>/dev/null)" ]; then
            # 去重：检查是否已存在
            if ! echo "$skill_names" | grep -qxF "$name"; then
                if [ -z "$skill_names" ]; then
                    skill_names="$name"
                else
                    skill_names="${skill_names}
${name}"
                fi
            fi
        fi
    done
}

# 扫描两个目录：hatchery 安装目录 + harness CLI 安装目录
scan_skills_dir "$HOME/.hermes/skills"
scan_skills_dir "$HOME/.agents/skills"

# 计算技能数量
if [ -z "$skill_names" ]; then
    skill_count=0
else
    skill_count=$(echo "$skill_names" | wc -l | tr -d ' ')
fi

echo "scanned dirs: ~/.hermes/skills/ ~/.agents/skills/, found $skill_count skills" >>"$LOG_FILE"

# 构建 JSON 数组
if [ -z "$skill_names" ]; then
    echo "[]" | gzip -c | base64
    exit 0
fi

# 公共函数：将 gzip 输出按大小决定 base64 单次还是分块元数据
# 参数：$1 = gzip 临时文件路径
output_gzip_file() {
    local gz_file="$1"
    local gz_size
    gz_size=$(wc -c < "$gz_file" | tr -d ' ')
    if [ "$gz_size" -le 16000 ] && [ "$gz_size" -gt 0 ]; then
        base64 < "$gz_file"
        rm -f "$gz_file"
    else
        local chunks=$(( (gz_size + 15999) / 16000 ))
        echo "{\"mode\":\"file\",\"path\":\"$gz_file\",\"size\":$gz_size,\"chunks\":$chunks}"
    fi
}
gzip_tmp=$(mktemp -t hermes_skills_gz.XXXXXX)

# 优先用 jq 构建 JSON（安全转义）
if command -v jq >/dev/null 2>&1; then
    json="[]"
    while IFS= read -r name; do
        [ -z "$name" ] && continue
        json=$(echo "$json" | jq --arg n "$name" '. + [{slug: $n, name: $n, description: "", eligible: true, can_uninstall: true}]')
    done <<EOF
$skill_names
EOF
    echo "$json" | jq -c '.' | gzip > "$gzip_tmp"
    output_gzip_file "$gzip_tmp"
else
    # jq 不可用时手动拼 JSON，通过 gzip+base64 压缩突破 TAT 24KB 限制
    {
    first=true
    printf '['
    while IFS= read -r name; do
        [ -z "$name" ] && continue
        if [ "$first" = true ]; then
            first=false
        else
            printf ','
        fi
        # 转义特殊字符（保守处理：先反斜杠再双引号，与 ACE 版本对齐）
        safe_name=$(printf '%s' "$name" | sed 's/\\/\\\\/g; s/"/\\"/g')
        printf '{"slug":"%s","name":"%s","description":"","eligible":true,"can_uninstall":true}' "$safe_name" "$safe_name"
    done <<EOF
$skill_names
EOF
    printf ']\n'
    } | gzip > "$gzip_tmp"
    output_gzip_file "$gzip_tmp"
fi
