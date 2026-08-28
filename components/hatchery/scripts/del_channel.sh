#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 防并发：抢不到锁立即退出，避免短时间内多次 restart gateway 撞 systemd start-limit
LOCK_FILE="/tmp/.openclaw_del_channel.lock"
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
    echo "✗ 另一个 del_channel 正在执行，请稍后重试" >&2
    exit 1
fi

# ========== 日志系统初始化 ==========
# 使用用户家目录，避免非 root 运行时无法写入 /var/log
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="del_channel"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# 将 stdout 和 stderr 同时输出到终端和日志文件
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

CHANNEL="{{channel}}"

# lark 和 飞书保持一致
if [[ "$CHANNEL" == "lark" ]]; then
    CHANNEL="feishu"
fi

echo "=== 删除通道: $CHANNEL ==="

# 微信通道需要额外清理 accounts.json 和账号文件
if [[ "$CHANNEL" == "openclaw-weixin" ]]; then
    WEIXIN_DIR="$HOME/.openclaw/openclaw-weixin"
    ACCOUNTS_FILE="$WEIXIN_DIR/accounts.json"

    if [ -f "$ACCOUNTS_FILE" ]; then
        # 删除每个账号对应的 json 文件和 sync 文件
        for ACCT in $(jq -r '.[]' "$ACCOUNTS_FILE" 2>/dev/null); do
            rm -f "$WEIXIN_DIR/accounts/${ACCT}.json"
            rm -f "$WEIXIN_DIR/accounts/${ACCT}.sync.json"
        done
        # 清空 accounts.json 为空数组
        echo '[]' > "$ACCOUNTS_FILE"
        echo "✓ 微信账号文件已清理"
    fi
fi

# 从 openclaw.json 中删除对应通道配置
echo "删除通道配置..."
# wecom_app 在 openclaw 配置中实际存储为 .channels.wecom.agent
if [ ! -f "$HOME/.openclaw/openclaw.json" ]; then
    echo "config file not found, skip channel deletion: $HOME/.openclaw/openclaw.json"
    exit 0
fi

if [[ "$CHANNEL" == "wecom_app" ]]; then
    if jq -e '.channels.wecom.agent' "$HOME/.openclaw/openclaw.json" >/dev/null 2>&1; then
        jq 'del(.channels.wecom.agent)' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
        cp "$HOME/.openclaw/openclaw.json" "$HOME/.openclaw/openclaw.json.bak.$(date +%y-%m-%dT%H:%M:%S)"
        mv /tmp/openclaw.json "$HOME/.openclaw/openclaw.json"
        echo "✓ wecom_app 通道配置已删除"
    else
        echo "⚠ wecom_app 通道配置不存在，跳过"
    fi
elif [[ "$CHANNEL" == "dingtalk-connector" ]]; then
    # 新版钉钉插件：通道配置存储为 dingtalk-connector，而非 ddingtalk
    if jq -e '.channels["dingtalk-connector"]' "$HOME/.openclaw/openclaw.json" >/dev/null 2>&1; then
        jq 'del(.channels["dingtalk-connector"])' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
        cp "$HOME/.openclaw/openclaw.json" "$HOME/.openclaw/openclaw.json.bak.$(date +%y-%m-%dT%H:%M:%S)"
        mv /tmp/openclaw.json "$HOME/.openclaw/openclaw.json"
        echo "✓ dingtalk-connector 通道配置已删除"
    else
        echo "⚠ dingtalk-connector 通道配置不存在，跳过"
    fi
elif [[ "$CHANNEL" == "msteams" ]]; then
    if jq -e '.channels["msteams"]' "$HOME/.openclaw/openclaw.json" >/dev/null 2>&1; then
        jq 'del(.channels["msteams"])' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
        cp "$HOME/.openclaw/openclaw.json" "$HOME/.openclaw/openclaw.json.bak.$(date +%y-%m-%dT%H:%M:%S)"
        mv /tmp/openclaw.json "$HOME/.openclaw/openclaw.json"
        echo "✓ msteams 通道配置已删除"
    else
        echo "⚠ msteams 通道配置不存在，跳过"
    fi
elif [[ "$CHANNEL" == "whatsapp" ]]; then
    if jq -e '.channels["whatsapp"]' "$HOME/.openclaw/openclaw.json" >/dev/null 2>&1; then
        jq 'del(.channels["whatsapp"])' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
        cp "$HOME/.openclaw/openclaw.json" "$HOME/.openclaw/openclaw.json.bak.$(date +%y-%m-%dT%H:%M:%S)"
        mv /tmp/openclaw.json "$HOME/.openclaw/openclaw.json"
        echo "✓ whatsapp 通道配置已删除"
    else
        echo "⚠ whatsapp 通道配置不存在，跳过"
    fi
elif [[ "$CHANNEL" == "ddingtalk" ]]; then
    # 旧版钉钉插件：通道配置存储为 ddingtalk
    if jq -e '.channels["ddingtalk"]' "$HOME/.openclaw/openclaw.json" >/dev/null 2>&1; then
        jq 'del(.channels["ddingtalk"])' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
        cp "$HOME/.openclaw/openclaw.json" "$HOME/.openclaw/openclaw.json.bak.$(date +%y-%m-%dT%H:%M:%S)"
        mv /tmp/openclaw.json "$HOME/.openclaw/openclaw.json"
        echo "✓ ddingtalk 通道配置已删除"
    else
        echo "⚠ ddingtalk 通道配置不存在，跳过"
    fi
elif jq -e ".channels[\"$CHANNEL\"]" "$HOME/.openclaw/openclaw.json" >/dev/null 2>&1; then
    jq "del(.channels[\"$CHANNEL\"])" "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
    cp "$HOME/.openclaw/openclaw.json" "$HOME/.openclaw/openclaw.json.bak.$(date +%y-%m-%dT%H:%M:%S)"
    mv /tmp/openclaw.json "$HOME/.openclaw/openclaw.json"
    echo "✓ $CHANNEL 通道配置已删除"
else
    echo "⚠ $CHANNEL 通道配置不存在，跳过"
fi

# 删除通道时，同步 disable 对应插件，避免未配置的插件处于 enable 状态影响通道使用
# channel → plugin 对应关系：
#   feishu          → feishu / openclaw-lark   （按 openclaw 版本切换，>= 2026.5.28 用新名）
#   slack           → slack                    （openclaw plugins disable）
#   discord         → discord                   (openclaw plugins disable)
#   qqbot           → openclaw-qqbot            （openclaw plugins disable）
#   wecom/wecom_app → wecom / wecom-openclaw-plugin（按安装路径自动识别 sunnoy/official/legacy）
#   ddingtalk       → ddingtalk / dingtalk-connector（按安装路径自动识别新旧版）
#   openclaw-weixin → openclaw-weixin            （openclaw plugins disable）
#   lightclawbot    → lightclawbot               （openclaw plugins disable）
#   adp-openclaw    → adp-openclaw               （openclaw plugins disable）
#   yuanbao         → openclaw-plugin-yuanbao    （openclaw plugins disable）
CFG="$HOME/.openclaw/openclaw.json"

# 辅助函数：disable 插件（调用 openclaw plugins disable）
_disable_npm_plugin() {
    local plugin_id="$1"
    local label="$2"
    echo "disable ${label} 插件..."
    openclaw plugins disable "${plugin_id}" || true
    echo "✓ ${label} 插件已禁用"
}

# ========== 公共：openclaw 版本号获取与比较 ==========
# 与 set_channel.sh / compat_plugins.sh / feishu_bot_creator.sh 中的实现保持一致。
# 用于 feishu → openclaw-lark 等按版本切换插件名的场景。

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

# 辅助函数：检测当前安装的 wecom 插件版本（与 set_channel.sh 判据保持一致）
# 输出: "sunnoy" 表示 @sunnoy/wecom，"official" 表示 wecom-openclaw-plugin，"legacy" 表示 @mocrane/wecom
_detect_wecom_plugin_variant() {
    if compgen -G "$HOME/.openclaw/npm/projects/sunnoy-wecom-*" >/dev/null 2>&1; then
        echo "sunnoy"
    elif [ -d "$HOME/.openclaw/extensions/wecom-openclaw-plugin" ] || \
       [ -d "$HOME/.openclaw/npm/node_modules/@wecom/wecom-openclaw-plugin" ] || \
       compgen -G "$HOME/.openclaw/npm/projects/wecom-wecom-openclaw-plugin-*" >/dev/null 2>&1; then
        echo "official"
    else
        echo "legacy"
    fi
}

# 辅助函数：检测当前安装的 dingtalk 插件版本（与 set_channel.sh 判据保持一致）
# 输出: "new" 表示新版（dingtalk-connector），"old" 表示旧版（ddingtalk）
_detect_dingtalk_plugin_variant() {
    if [ -d "$HOME/.openclaw/npm/node_modules/@dingtalk-real-ai/dingtalk-connector" ] || \
       compgen -G "$HOME/.openclaw/npm/projects/dingtalk-real-ai-dingtalk-connector-*" >/dev/null 2>&1; then
        echo "new"
    else
        echo "old"
    fi
}

if [[ "$CHANNEL" == "feishu" ]]; then
    # 与 set_channel.sh 保持一致：openclaw >= 2026.5.28 起插件改名为 openclaw-lark
    _feishu_openclaw_version="$(_get_openclaw_version)"
    if [ -n "$_feishu_openclaw_version" ] && _openclaw_version_ge "2026.5.28"; then
        _feishu_plugin_id="openclaw-lark"
        echo "检测到 openclaw 版本 ${_feishu_openclaw_version} >= 2026.5.28，使用新插件名: ${_feishu_plugin_id}"
    else
        _feishu_plugin_id="feishu"
        echo "检测到 openclaw 版本 ${_feishu_openclaw_version:-unknown} < 2026.5.28（或无法获取），使用旧插件名: ${_feishu_plugin_id}"
    fi
    _disable_npm_plugin "$_feishu_plugin_id" "$_feishu_plugin_id"
elif [[ "$CHANNEL" == "slack" ]]; then
    _disable_npm_plugin "slack" "slack"
elif [[ "$CHANNEL" == "discord" ]]; then
    _disable_npm_plugin "discord" "discord"
elif [[ "$CHANNEL" == "qqbot" ]]; then
    _disable_npm_plugin "qqbot" "qqbot"
elif [[ "$CHANNEL" == "wecom" ]]; then
    # wecom bot 通道删除后，检查 wecom_app（.channels.wecom.agent）是否仍存在
    # 若两者都不存在，才 disable wecom 插件
    if ! jq -e '.channels.wecom.agent' "$CFG" >/dev/null 2>&1; then
        _wecom_variant="$(_detect_wecom_plugin_variant)"
        if [ "$_wecom_variant" = "official" ]; then
            _disable_npm_plugin "wecom-openclaw-plugin" "wecom-openclaw-plugin"
            # 同步清理另一版本可能残留的 plugins.entries（与 set_channel.sh 思路一致）
            if jq -e '.plugins.entries.wecom' "$CFG" >/dev/null 2>&1; then
                jq '.plugins.entries.wecom = ((.plugins.entries.wecom // {}) + {"enabled": false})' \
                    "$CFG" > /tmp/openclaw.json
                cp "$CFG" "$CFG.bak.$(date +%y-%m-%dT%H:%M:%S)"
                mv /tmp/openclaw.json "$CFG"
                echo "✓ 老版 wecom 插件残留 entries 已置 disabled"
            fi
        else
            # sunnoy 和 legacy 都使用 plugin_id "wecom"
            _disable_npm_plugin "wecom" "wecom"
            if jq -e '.plugins.entries["wecom-openclaw-plugin"]' "$CFG" >/dev/null 2>&1; then
                jq '.plugins.entries["wecom-openclaw-plugin"] = ((.plugins.entries["wecom-openclaw-plugin"] // {}) + {"enabled": false})' \
                    "$CFG" > /tmp/openclaw.json
                cp "$CFG" "$CFG.bak.$(date +%y-%m-%dT%H:%M:%S)"
                mv /tmp/openclaw.json "$CFG"
                echo "✓ wecom-openclaw-plugin 插件残留 entries 已置 disabled"
            fi
        fi
    else
        echo "wecom_app 通道仍存在，保持 wecom 插件启用状态"
    fi
elif [[ "$CHANNEL" == "wecom_app" ]]; then
    # wecom_app 删除后，检查 wecom bot 通道是否仍存在
    # 注意：此时 .channels.wecom 对象（含 enabled 等顶层字段）可能仍残留，
    #       不能用 .channels.wecom 判断，必须精确判断 bot 通道配置。
    #       @sunnoy/wecom（>= 2026.7.01）使用扁平格式（botId 在顶层，无 bot 对象）；
    #       其他插件使用嵌套格式（.channels.wecom.bot）。
    #       两种格式任一存在即视为 wecom bot 通道仍在。
    # 若都不存在，才 disable wecom 插件
    if ! jq -e '.channels.wecom.bot or .channels.wecom.botId' "$CFG" >/dev/null 2>&1; then
        _wecom_variant="$(_detect_wecom_plugin_variant)"
        if [ "$_wecom_variant" = "official" ]; then
            _disable_npm_plugin "wecom-openclaw-plugin" "wecom-openclaw-plugin"
        else
            _disable_npm_plugin "wecom" "wecom"
        fi
    else
        echo "wecom 通道仍存在，保持 wecom 插件启用状态"
    fi
elif [[ "$CHANNEL" == "dingtalk-connector" ]]; then
    # 与 set_channel.sh 保持一致的路径判据
    _dingtalk_variant="$(_detect_dingtalk_plugin_variant)"
    if [ "$_dingtalk_variant" = "new" ]; then
        _disable_npm_plugin "dingtalk-connector" "dingtalk-connector"
    else
        # 路径上未检出新版插件却收到了 dingtalk-connector 通道删除请求，
        # 兜底仍按 dingtalk-connector 名 disable，避免遗漏。
        _disable_npm_plugin "dingtalk-connector" "dingtalk-connector"
    fi
elif [[ "$CHANNEL" == "ddingtalk" ]]; then
    _dingtalk_variant="$(_detect_dingtalk_plugin_variant)"
    if [ "$_dingtalk_variant" = "old" ]; then
        _disable_npm_plugin "ddingtalk" "ddingtalk"
    else
        # 路径上检出新版但收到旧名通道删除请求：两个名字都尝试 disable，确保彻底关闭
        _disable_npm_plugin "ddingtalk" "ddingtalk"
    fi
elif [[ "$CHANNEL" == "openclaw-weixin" ]]; then
    _disable_npm_plugin "openclaw-weixin" "openclaw-weixin"
elif [[ "$CHANNEL" == "lightclawbot" ]]; then
    _disable_npm_plugin "lightclawbot" "lightclawbot"
elif [[ "$CHANNEL" == "adp-openclaw" ]]; then
    _disable_npm_plugin "adp-openclaw" "adp-openclaw"
elif [[ "$CHANNEL" == "yuanbao" ]]; then
    _disable_npm_plugin "openclaw-plugin-yuanbao" "openclaw-plugin-yuanbao"
elif [[ "$CHANNEL" == "whatsapp" ]]; then
    _whatsapp_auth="$HOME/.openclaw/whatsapp-login"
    if [ -d "$_whatsapp_auth" ]; then
        rm -rf "$_whatsapp_auth"
        echo "whatsapp 登录信息已删除"
    fi
    _disable_npm_plugin "whatsapp" "whatsapp"
fi

echo "重启 gateway..."
systemctl --user restart openclaw-gateway
echo "✓ gateway 已重启"

echo ""
echo "=== 通道 $CHANNEL 删除完成 ==="
