#!/bin/bash
# list_skills.sh
# 列出 openclaw 实例已启用的 skills。
# 契约：stdout 只输出 JSON（jq 的过滤结果），不允许夹杂任何日志/诊断信息。
#       上游 controller (HandleSkillsList) 直接将 TAT 返回的 stdout 作为
#       application/json 响应体透传，因此 stderr 也必须重定向到日志文件，
#       否则 TAT 合并 stdout/stderr 后会污染 HTTP 响应。
#
# 与 list_skills_ace.sh / list_skills_hermes.sh 输出形状保持一致：
#   [{"slug": "...", "name": "...", "description": "...", "eligible": true, "can_uninstall": true}, ...]

set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化 ==========
# 使用用户家目录，避免非 root 运行时无法写入 /var/log
LOG_DIR="${HOME}/.openclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/list_skills.log"

# 关键：把整个进程的 stderr 永久重定向到日志文件，
# 避免 TAT 把 stderr 与 stdout 合并后污染 HTTP 响应中的 JSON。
exec 2>>"$LOG_FILE"

echo "" >&2
echo "========== $(date '+%Y-%m-%d %H:%M:%S') list_skills 开始 ==========" >&2

# 日志辅助函数：仅写 stderr（已经被重定向到日志文件），绝不进 stdout
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

# 创建临时文件存放 openclaw 输出，脚本退出时自动清理
tmp_output=$(mktemp -t openclaw_skills.XXXXXX)
slug_map=$(mktemp -t openclaw_skill_slug_map.XXXXXX)
log "临时文件: $tmp_output"
trap 'rm -f "$tmp_output" "$slug_map"' EXIT

log "执行 openclaw skills list --json ..."
# 只消费 CLI stdout；所有 stderr 已由文件级重定向写入日志。
openclaw skills list --json >"$tmp_output" || true
log "✓ openclaw 输出已写入临时文件 ($(wc -c <"$tmp_output") 字节)"

# 生成 JSON、gzip 压缩到临时文件，根据压缩后大小决定输出策略：
#   - ≤16KB：base64 单次输出（快路径，一次 TAT 调用）
#   - >16KB：输出 {"mode":"file","path":"...","size":...,"chunks":N}，
#     由 Go 侧通过多次 TAT 分块读取（突破 24KB 上限，容量无限制）
# 这一行是脚本中**唯一**写 stdout 的命令。
gzip_tmp=$(mktemp -t hatchery_skills_gz.XXXXXX)
log "gzip 临时文件: $gzip_tmp"
# OpenClaw 按 name 合并技能并处理重名，skills list 中每个 name 只会出现一次。
# 目录名是稳定 slug；SKILL.md 的 name 用于关联 CLI display name。
{
  for skill_dir in "$HOME/.openclaw/workspace/skills"/*/; do
    [ -f "$skill_dir/SKILL.md" ] || continue
    slug=$(basename "$skill_dir")
    display_name=$(sed -n 's/^name:[[:space:]]*//p' "$skill_dir/SKILL.md" | sed -n '1p')
    display_name=${display_name#\"}
    display_name=${display_name%\"}
    [ -n "$display_name" ] || display_name="$slug"
    jq -cn --arg name "$display_name" --arg slug "$slug" '{key: $name, value: $slug}'
  done
} | jq -s 'from_entries' > "$slug_map"

# 有安装目录的技能使用目录 slug 并允许卸载；仅 CLI 可见的内建技能不可卸载。
sed -n '/^{/,/^}/p' "$tmp_output" |
  jq --slurpfile slugs "$slug_map" '[
    .skills[]
    | select(.eligible == true)
    | . as $skill
    | ($slugs[0][$skill.name] // null) as $slug
    | {
        slug: ($slug // $skill.name),
        name: $skill.name,
        description: $skill.description,
        eligible: $skill.eligible,
        can_uninstall: ($slug != null)
      }
  ]' |
  gzip > "$gzip_tmp"
gz_size=$(wc -c < "$gzip_tmp" | tr -d ' ')
log "gzip 压缩后大小: ${gz_size} 字节"

if [ "$gz_size" -le 16000 ] && [ "$gz_size" -gt 0 ]; then
    # 快路径：压缩数据小，单次 base64 传输即可
    base64 < "$gzip_tmp"
    rm -f "$gzip_tmp"
    log "✓ 单次 base64 模式输出完成"
else
    # 分块路径：压缩数据过大，写元数据供 Go 侧分块读取
    chunks=$(( (gz_size + 15999) / 16000 ))
    echo "{\"mode\":\"file\",\"path\":\"$gzip_tmp\",\"size\":$gz_size,\"chunks\":$chunks}"
    log "✓ 分块传输模式: path=$gzip_tmp size=$gz_size chunks=$chunks"
fi

log "✓ skills 列表输出完成"
echo "========== $(date '+%Y-%m-%d %H:%M:%S') list_skills 结束 ==========" >&2
