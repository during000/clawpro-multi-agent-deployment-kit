#!/bin/bash
set -euo pipefail

# ========== 日志系统初始化 ==========
# 使用用户家目录，避免非 root 运行时无法写入 /var/log
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="feishu_bot_creator"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# 将 stdout 和 stderr 同时输出到终端和日志文件
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 通过 TAT 模板变量接收欢迎词，未传入时使用默认值
GREETING="{{greeting}}"
if [ "$GREETING" = "{{greeting}}" ] || [ -z "$GREETING" ]; then
  GREETING="Hi，我是你刚刚创建的机器人，你现在可以跟我聊天了！"
fi

# 记录脚本开始时间
SCRIPT_START=$(date +%s)
echo "[$(date '+%Y-%m-%d %H:%M:%S')] 脚本开始执行"

# ========== 版本判断：决定插件名 ==========
# openclaw >= 2026.5.28 起，飞书插件由原 "feishu" 重命名为 "openclaw-lark"
#   - plugins.allow / plugins enable / plugins.entries 全部用新名 openclaw-lark
#   - 同时移除 plugins.entries.feishu 老名残留（避免 doctor 重置或老插件冲突）
#   - channels.feishu 保持原名不动（Go 层 channel id 仍是 "feishu"，新插件读老 channel）
# 与 compat_plugins.sh::fix_lark_legacy_names 保持一致的判据。

# 获取 openclaw 版本号；获取失败输出空字符串。兼容 `--version` / `version` 两种 CLI。
_get_openclaw_version() {
    {
        set +o pipefail
        { openclaw --version; openclaw version; } 2>&1 \
            | grep -oP '[0-9]{4}\.[0-9]+\.[0-9]+' | head -1
    } || true
}

# 判断当前 openclaw 版本是否 >= 指定版本（三段语义版本号比较）
# 返回 0 表示满足；返回 1 表示不满足或无法获取当前版本。
_openclaw_version_ge() {
    local target="$1"
    [ -z "$target" ] && return 1
    local current
    current="$(_get_openclaw_version)"
    [ -z "$current" ] && return 1

    local cur_major cur_minor cur_patch tgt_major tgt_minor tgt_patch
    cur_major="$(echo "$current" | awk -F. '{print $1+0}')"
    cur_minor="$(echo "$current" | awk -F. '{print $2+0}')"
    cur_patch="$(echo "$current" | awk -F. '{print $3+0}')"
    tgt_major="$(echo "$target"  | awk -F. '{print $1+0}')"
    tgt_minor="$(echo "$target"  | awk -F. '{print $2+0}')"
    tgt_patch="$(echo "$target"  | awk -F. '{print $3+0}')"

    if [ "$cur_major" -gt "$tgt_major" ] 2>/dev/null; then return 0
    elif [ "$cur_major" -lt "$tgt_major" ] 2>/dev/null; then return 1
    fi
    if [ "$cur_minor" -gt "$tgt_minor" ] 2>/dev/null; then return 0
    elif [ "$cur_minor" -lt "$tgt_minor" ] 2>/dev/null; then return 1
    fi
    if [ "$cur_patch" -ge "$tgt_patch" ] 2>/dev/null; then return 0; fi
    return 1
}

OPENCLAW_VERSION="$(_get_openclaw_version)"
if [ -n "$OPENCLAW_VERSION" ] && _openclaw_version_ge "2026.5.28"; then
    PLUGIN_ID="openclaw-lark"
    echo "检测到 openclaw 版本 ${OPENCLAW_VERSION} >= 2026.5.28，使用新插件名: ${PLUGIN_ID}"
else
    PLUGIN_ID="feishu"
    echo "检测到 openclaw 版本 ${OPENCLAW_VERSION:-unknown} < 2026.5.28（或无法获取），使用旧插件名: ${PLUGIN_ID}"
fi

# 检查 ${PLUGIN_ID} 插件是否已经启用
PLUGIN_ENABLED=false
echo "检查 ${PLUGIN_ID} 插件状态..."

# 使用 openclaw plugins list --json 检查（JSON 格式输出，解析更稳健）
# 真实输出字段示例：{ "id": "openclaw-lark"|"feishu", "enabled": true, "activated": true, "status": "loaded", ... }
# 判定规则：enabled=true 或 status=loaded/enabled 视为已启用；status=disabled 视为未启用
if command -v openclaw >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    # 注意：当前脚本开启了 set -euo pipefail，管道中使用 head 会触发 SIGPIPE 导致整条管道失败
    # 这里用子 shell 暂时关闭 pipefail，同时用 || true 兜底，保证命令失败不会让脚本整体退出
    _plugin_info=$(
        set +o pipefail
        openclaw plugins list --json 2>/dev/null \
            | jq -r --arg id "$PLUGIN_ID" '.. | objects | select((.id // .name // "") == $id) | "\(.enabled // "")|\(.status // .state // "")"' 2>/dev/null \
            | head -n 1
    ) || true
    _plugin_enabled_field="${_plugin_info%%|*}"
    _plugin_status="${_plugin_info##*|}"
    if [ "$_plugin_enabled_field" = "true" ] || [ "$_plugin_status" = "loaded" ] || [ "$_plugin_status" = "enabled" ]; then
        PLUGIN_ENABLED=true
        echo "✓ 检测到 ${PLUGIN_ID} 插件已启用 (enabled=${_plugin_enabled_field}, status=${_plugin_status})，跳过插件配置步骤"
    elif [ -n "$_plugin_status" ] || [ -n "$_plugin_enabled_field" ]; then
        echo "  ${PLUGIN_ID} 插件当前 enabled=${_plugin_enabled_field}, status=${_plugin_status}，需要启用"
    else
        echo "  未从 JSON 输出中匹配到 ${PLUGIN_ID} 插件，需要启用"
    fi
fi

# 配置飞书通道前，先将插件 ID 加入 plugins.allow，再 enable 插件（避免 Config overwrite 覆盖通道配置）
PLUGIN_CONFIG_START=$(date +%s)
if [ "$PLUGIN_ENABLED" = "false" ]; then
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] 开始配置 ${PLUGIN_ID} 插件..."

    # 将插件 ID 加入 plugins.allow（幂等：已存在则不重复追加）
    # openclaw >= 2026.5.28：还需要从 plugins.allow / plugins.entries 中移除老名 "feishu"，
    # 避免老名残留导致 doctor / Config overwrite 异常（与 compat_plugins.sh::fix_lark_legacy_names 一致）。
    _cfg="$HOME/.openclaw/openclaw.json"
    if [ -f "$_cfg" ]; then
        if [ "$PLUGIN_ID" = "openclaw-lark" ]; then
            jq --arg id "$PLUGIN_ID" \
                '.plugins.allow = ((.plugins.allow // []) | map(select(. != "feishu")))
                 | .plugins.allow = ((.plugins.allow // []) | if index($id) then . else . + [$id] end)
                 | if (.plugins.entries // {} | has("feishu")) then del(.plugins.entries.feishu) else . end' \
                "$_cfg" > /tmp/openclaw_allow.json && mv /tmp/openclaw_allow.json "$_cfg"
            echo "✓ plugins.allow 已加入 ${PLUGIN_ID}（同时移除老名 feishu）"
            echo "✓ plugins.entries.feishu 老名残留已清理"
        else
            jq --arg id "$PLUGIN_ID" \
                '.plugins.allow = ((.plugins.allow // []) | if index($id) then . else . + [$id] end)' \
                "$_cfg" > /tmp/openclaw_allow.json && mv /tmp/openclaw_allow.json "$_cfg"
            echo "✓ ${PLUGIN_ID} 已加入 plugins.allow"
        fi
    fi
    echo "启用 ${PLUGIN_ID} 插件..."
    openclaw plugins enable "$PLUGIN_ID" || true
    echo "✓ ${PLUGIN_ID} 插件已启用"
    echo "重启 gateway 使插件生效..."
    systemctl --user restart openclaw-gateway || true
    echo "✓ gateway 已重启"
fi

PLUGIN_CONFIG_END=$(date +%s)
PLUGIN_CONFIG_TIME=$((PLUGIN_CONFIG_END - PLUGIN_CONFIG_START))
echo "[$(date '+%Y-%m-%d %H:%M:%S')] 插件配置完成，耗时: ${PLUGIN_CONFIG_TIME}秒"

# 下载远程脚本，用 sed 在写入配置的字典里直接追加 dmPolicy/allowFrom 字段，再执行
DOWNLOAD_START=$(date +%s)
echo "[$(date '+%Y-%m-%d %H:%M:%S')] 开始下载远程脚本..."

curl -s https://blueprint-common-1325194254.cos.ap-guangzhou.myqcloud.com/openclaw/feishu_bot_creator.py \
  | sed 's/"groupPolicy": "open",/"groupPolicy": "open", "dmPolicy": "open", "allowFrom": ["*"],/' \
  | python3 -u - create \
      --platform "{{feishu_domain}}" \
      --avatar-url "https://clawpro-feishu-1251783334.cos.ap-guangzhou.myqcloud.com/avatar_3d_preview.png" \
      --greeting "$GREETING"

DOWNLOAD_END=$(date +%s)
DOWNLOAD_TIME=$((DOWNLOAD_END - DOWNLOAD_START))
echo "[$(date '+%Y-%m-%d %H:%M:%S')] 远程脚本执行完成，耗时: ${DOWNLOAD_TIME}秒"

# 记录脚本总耗时
SCRIPT_END=$(date +%s)
TOTAL_TIME=$((SCRIPT_END - SCRIPT_START))
echo "[$(date '+%Y-%m-%d %H:%M:%S')] 脚本执行完成，总耗时: ${TOTAL_TIME}秒"

echo ""
echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="