#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 防并发：抢不到锁立即退出，避免短时间内多次 restart gateway 撞 systemd start-limit
LOCK_FILE="/tmp/.openclaw_set_channel.lock"
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
    echo "✗ 另一个 set_channel 正在执行，请稍后重试" >&2
    exit 1
fi

# ========== 日志系统初始化 ==========
# 使用用户家目录，避免非 root 运行时无法写入 /var/log
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="set_channel"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# 将 stdout 和 stderr 同时输出到终端和日志文件
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

echo "=== 配置通道: {{channel}} ==="

# 清理上次运行可能遗留的临时文件，避免旧配置被误写入
rm -f /tmp/openclaw.json

if [[ "{{channel}}" == "qqbot" ]]; then
    # qqbot 是 openclaw 内置插件，已随主程序提供，无需重新下载/安装。
    # 历史问题：调用 `openclaw channels add --channel qqbot ...` 会触发
    #   ① CLI 内部 "Installed QQ Bot plugin" 流程；
    #   ② 与脚本前面 `openclaw plugins enable` / `jq 写 plugins.allow` 形成多次写 config，
    #   触发 ConfigMutationConflictError: config changed since last load。
    # 修复：完全绕开 CLI，全部用 jq 一次性写完 plugins.allow / plugins.entries.qqbot.enabled / channels.qqbot，
    #      然后由脚本末尾统一的 systemctl restart 让 gateway 重新加载配置生效。
    echo "配置 qqbot 通道（合并写入 plugins.allow / plugins.entries.qqbot / channels.qqbot）..."
    jq --arg id "qqbot" \
       --arg appid "{{app_id}}" \
       --arg secret "{{app_secret}}" \
        '.plugins.allow = ((.plugins.allow // []) | if index($id) then . else . + [$id] end)
         | .plugins.entries[$id] = ((.plugins.entries[$id] // {}) + {"enabled": true})
         | .channels.qqbot = {
             "enabled": true,
             "appId": $appid,
             "clientSecret": $secret
           }' \
        "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
    echo "✓ qqbot 通道配置已合并写入（未触发插件重新安装）"
fi

# 辅助函数：enable 插件（调用 openclaw plugins enable）
# 注意：不在此处写 openclaw.json，避免与后续 jq 写入产生 hash 冲突
# （ConfigMutationConflictError）。plugins.allow 的追加统一在各通道最终的 jq 写入中完成。
_enable_npm_plugin() {
    local plugin_id="$1"
    local label="$2"
    echo "启用 ${label} 插件..."
    openclaw plugins enable "${plugin_id}" || true
    echo "✓ ${label} 插件已启用"
}

# ========== 公共：openclaw 版本号获取与比较 ==========
# 用于 feishu → openclaw-lark 等按版本切换插件名的场景。
# 与 compat_plugins.sh / feishu_bot_creator.sh 中的实现保持一致。

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

if [[ "{{channel}}" == "wecom" ]]; then
    # wecom 插件三版检测：
    #   >= 2026.7.01: @sunnoy/wecom (sunnoy-wecom-* 目录) 优先 → 其次 wecom-openclaw-plugin → 兜底 @mocrane/wecom
    #   <  2026.7.01: wecom-openclaw-plugin (三个安装路径) → 兜底 @mocrane/wecom
    # @sunnoy/wecom 检测路径：npm/projects/sunnoy-wecom-<hash>/
    # wecom-openclaw-plugin 检测：
    #   1) extensions/wecom-openclaw-plugin
    #   2) npm/node_modules/@wecom/wecom-openclaw-plugin
    #   3) npm/projects/wecom-wecom-openclaw-plugin-<hash>/
    _wecom_openclaw_version="$(_get_openclaw_version)"
    _has_sunnoy_wecom=false
    compgen -G "$HOME/.openclaw/npm/projects/sunnoy-wecom-*" >/dev/null 2>&1 && _has_sunnoy_wecom=true

    if [ -d "$HOME/.openclaw/extensions/wecom-openclaw-plugin" ] || \
       [ -d "$HOME/.openclaw/npm/node_modules/@wecom/wecom-openclaw-plugin" ] || \
       compgen -G "$HOME/.openclaw/npm/projects/wecom-wecom-openclaw-plugin-*" >/dev/null; then
        _has_official_wecom=true
    else
        _has_official_wecom=false
    fi

    if [ -n "$_wecom_openclaw_version" ] && _openclaw_version_ge "2026.7.1"; then
        # >= 2026.7.01: @sunnoy/wecom 优先
        if $_has_sunnoy_wecom; then
            echo "配置 wecom 通道（@sunnoy/wecom 插件，扁平格式，版本 ${_wecom_openclaw_version}）..."
            jq '.plugins.allow = ((.plugins.allow // []) | if index("wecom") then . else . + ["wecom"] end)
                | .plugins.entries["wecom"] = ((.plugins.entries["wecom"] // {}) + {"enabled": true})
                | .channels.wecom.enabled = true
                | .channels.wecom.botId = "{{bot_id}}"
                | .channels.wecom.secret = "{{secret}}"
                | .channels.wecom.dmPolicy = "open"
                | .channels.wecom.welcomeMessage = "你好！我是 AI 助手"
                | .channels.wecom.sendThinkingMessage = true
                | del(.channels.wecom.bot)
                | del(.channels.wecom.connectionMode)
                | if (.plugins.entries // {} | has("wecom-openclaw-plugin")) then del(.plugins.entries["wecom-openclaw-plugin"]) else . end
                | .plugins.allow = ((.plugins.allow // []) | map(select(. != "wecom-openclaw-plugin")))' \
                "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
            echo "✓ wecom-openclaw-plugin 插件条目已清理"
            _enable_npm_plugin "wecom" "wecom"
        elif $_has_official_wecom; then
            echo "配置 wecom 通道（wecom-openclaw-plugin 插件，过渡期）..."
            jq '.plugins.allow = ((.plugins.allow // []) | if index("wecom-openclaw-plugin") then . else . + ["wecom-openclaw-plugin"] end)
                | .plugins.entries["wecom-openclaw-plugin"] = ((.plugins.entries["wecom-openclaw-plugin"] // {}) + {"enabled": true})
                | .channels.wecom.enabled = true
                | .channels.wecom.botId = "{{bot_id}}"
                | .channels.wecom.secret = "{{secret}}"
                | .channels.wecom.bot = {
                    "connectionMode": "websocket",
                    "streamPlaceholderContent": "正在思考...",
                    "welcomeText": "你好！我是 AI 助手",
                    "dm": {"policy": "open"}
                }
                | if (.plugins.entries // {} | has("wecom")) then del(.plugins.entries.wecom) else . end
                | .plugins.allow = ((.plugins.allow // []) | map(select(. != "wecom")))' \
                "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
            echo "✓ 老版 wecom 插件幽灵条目已清理"
            _enable_npm_plugin "wecom-openclaw-plugin" "wecom-openclaw-plugin"
        else
            echo "配置 wecom 通道（旧版 @mocrane/wecom 插件）..."
            jq '.plugins.allow = ((.plugins.allow // []) | if index("wecom") then . else . + ["wecom"] end)
                | .channels.wecom.enabled = true
                | .channels.wecom.bot = {
                  "connectionMode": "websocket",
                  "botId": "{{bot_id}}",
                  "secret": "{{secret}}",
                  "streamPlaceholderContent": "正在思考...",
                  "welcomeText": "你好！我是 AI 助手",
                  "dm": {"policy": "open"}
                }
                | if (.plugins.entries // {} | has("wecom-openclaw-plugin")) then del(.plugins.entries["wecom-openclaw-plugin"]) else . end
                | .plugins.entries["wecom"] = ((.plugins.entries["wecom"] // {}) + {"enabled": true})
                | .plugins.allow = ((.plugins.allow // []) | map(select(. != "wecom-openclaw-plugin")))' \
                "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
            echo "✓ wecom-openclaw-plugin 插件条目已清理"
            _enable_npm_plugin "wecom" "wecom"
        fi
    else
        # < 2026.7.01: 原逻辑（wecom-openclaw-plugin 优先）
        if $_has_official_wecom; then
            echo "配置 wecom 通道（新版插件）..."
            jq '.plugins.allow = ((.plugins.allow // []) | if index("wecom-openclaw-plugin") then . else . + ["wecom-openclaw-plugin"] end)
                | .plugins.entries["wecom-openclaw-plugin"] = ((.plugins.entries["wecom-openclaw-plugin"] // {}) + {"enabled": true})
                | .channels.wecom.enabled = true
                | .channels.wecom.botId = "{{bot_id}}"
                | .channels.wecom.secret = "{{secret}}"
                | .channels.wecom.bot = {
                    "connectionMode": "websocket",
                    "streamPlaceholderContent": "正在思考...",
                    "welcomeText": "你好！我是 AI 助手",
                    "dm": {"policy": "open"}
                }
                | if (.plugins.entries // {} | has("wecom")) then del(.plugins.entries.wecom) else . end
                | .plugins.allow = ((.plugins.allow // []) | map(select(. != "wecom")))' \
                "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
            echo "✓ 老版 wecom 插件幽灵条目已清理（plugins.entries.wecom 已删除，plugins.allow 中 wecom 已移除）"
            _enable_npm_plugin "wecom-openclaw-plugin" "wecom-openclaw-plugin"
        else
            echo "配置 wecom 通道（旧版插件）..."
            jq '.plugins.allow = ((.plugins.allow // []) | if index("wecom") then . else . + ["wecom"] end)
                | .channels.wecom.enabled = true
                | .channels.wecom.bot = {
                  "connectionMode": "websocket",
                  "botId": "{{bot_id}}",
                  "secret": "{{secret}}",
                  "streamPlaceholderContent": "正在思考...",
                  "welcomeText": "你好！我是 AI 助手",
                  "dm": {"policy": "open"}
                }
                | if (.plugins.entries // {} | has("wecom-openclaw-plugin")) then del(.plugins.entries["wecom-openclaw-plugin"]) else . end
                | .plugins.entries["wecom"] = ((.plugins.entries["wecom"] // {}) + {"enabled": true})
                | .plugins.allow = ((.plugins.allow // []) | map(select(. != "wecom-openclaw-plugin")))' \
                "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
            echo "✓ 新版 wecom-openclaw-plugin 插件已禁用"
            _enable_npm_plugin "wecom" "wecom"
        fi
    fi
fi

if [[ "{{channel}}" == "ddingtalk" ]]; then
    # 两个安装路径任一存在即视为新版钉钉插件已安装：
    #   1) npm/node_modules/@dingtalk-real-ai/dingtalk-connector              —— npm 全局 scoped 包形态
    #   2) npm/projects/dingtalk-real-ai-dingtalk-connector-<hash>/           —— 新版 npm 子工程隔离安装形态（目录名带哈希后缀，需用 glob 匹配）
    if [ -d "$HOME/.openclaw/npm/node_modules/@dingtalk-real-ai/dingtalk-connector" ] || \
       compgen -G "$HOME/.openclaw/npm/projects/dingtalk-real-ai-dingtalk-connector-*" >/dev/null; then
        # 新版插件：使用 dingtalk-connector
        echo "配置 dingtalk-connector 通道（新版插件）..."
        # plugins.allow 追加与通道配置合并为一次写入，避免 ConfigMutationConflictError
        jq '.plugins.allow = ((.plugins.allow // []) | if index("dingtalk-connector") then . else . + ["dingtalk-connector"] end)
            | .plugins.entries["dingtalk-connector"] = ((.plugins.entries["dingtalk-connector"] // {}) + {"enabled": true})
            | .channels["dingtalk-connector"] = {
              "enabled": true,
              "clientId": "{{client_id}}",
              "clientSecret": "{{client_secret}}"
            }' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
        _enable_npm_plugin "dingtalk-connector" "dingtalk-connector"
    else
        # 旧版插件：使用 ddingtalk
        echo "配置 ddingtalk 通道（旧版插件）..."
        # plugins.allow 追加与通道配置合并为一次写入，避免 ConfigMutationConflictError
        jq '.plugins.allow = ((.plugins.allow // []) | if index("ddingtalk") then . else . + ["ddingtalk"] end)
            | .plugins.entries["ddingtalk"] = ((.plugins.entries["ddingtalk"] // {}) + {"enabled": true})
            | .channels.ddingtalk = {
              "enabled": true,
              "clientId": "{{client_id}}",
              "clientSecret": "{{client_secret}}"
            }' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
        _enable_npm_plugin "ddingtalk" "ddingtalk"
    fi
fi

if [[ "{{channel}}" == "feishu" ]] || [[ "{{channel}}" == "lark" ]]; then
    # 版本判断：openclaw >= 2026.5.28 起，飞书插件由原 "feishu" 重命名为 "openclaw-lark"
    #   - plugins.allow / plugins.entries / plugins enable 全部用新名 openclaw-lark
    #   - 同时移除 plugins.entries.feishu 老名残留（避免 doctor 重置或老插件冲突）
    #   - channels.feishu 保持原名不动（Go 层 channel id 仍是 "feishu"，新插件读老 channel）
    # 与 compat_plugins.sh::fix_lark_legacy_names / feishu_bot_creator.sh 保持一致的判据。
    _feishu_openclaw_version="$(_get_openclaw_version)"
    if [ -n "$_feishu_openclaw_version" ] && _openclaw_version_ge "2026.5.28"; then
        _feishu_plugin_id="openclaw-lark"
        echo "检测到 openclaw 版本 ${_feishu_openclaw_version} >= 2026.5.28，使用新插件名: ${_feishu_plugin_id}"
    else
        _feishu_plugin_id="feishu"
        echo "检测到 openclaw 版本 ${_feishu_openclaw_version:-unknown} < 2026.5.28（或无法获取），使用旧插件名: ${_feishu_plugin_id}"
    fi

    # 注意：先 jq 写 config 再 openclaw plugins enable，避免 enable 触发的 Config overwrite
    # 与 jq 写入产生 hash 冲突（ConfigMutationConflictError）。
    echo "配置 feishu 通道（channel id 保持 'feishu' 不变）..."
    # plugins.allow 追加与通道配置合并为一次写入，避免 ConfigMutationConflictError
    # 新版（openclaw-lark）分支：额外移除 plugins.allow / plugins.entries 中的老名 "feishu"。
    if [ "$_feishu_plugin_id" = "openclaw-lark" ]; then
        jq --arg id "$_feishu_plugin_id" \
            '.plugins.allow = ((.plugins.allow // []) | map(select(. != "feishu")))
             | .plugins.allow = ((.plugins.allow // []) | if index($id) then . else . + [$id] end)
             | if (.plugins.entries // {} | has("feishu")) then del(.plugins.entries.feishu) else . end
             | .plugins.entries[$id] = ((.plugins.entries[$id] // {}) + {"enabled": true})
             | .channels["feishu"] = {
               "enabled": true,
               "appId": "{{app_id}}",
               "appSecret": "{{app_secret}}",
               "domain": "{{feishu_domain}}",
               "groupPolicy": "open",
               "dmPolicy": "open",
               "allowFrom": ["*"]
             }' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
        echo "✓ plugins.allow 已加入 ${_feishu_plugin_id}（同时移除老名 feishu）"
        echo "✓ plugins.entries.feishu 老名残留已清理"
    else
        jq --arg id "$_feishu_plugin_id" \
            '.plugins.allow = ((.plugins.allow // []) | if index($id) then . else . + [$id] end)
             | .plugins.entries[$id] = ((.plugins.entries[$id] // {}) + {"enabled": true})
             | .channels["feishu"] = {
               "enabled": true,
               "appId": "{{app_id}}",
               "appSecret": "{{app_secret}}",
               "domain": "{{feishu_domain}}",
               "groupPolicy": "open",
               "dmPolicy": "open",
               "allowFrom": ["*"]
             }' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
    fi
    openclaw plugins enable "$_feishu_plugin_id" || true
fi

if [[ "{{channel}}" == "slack" ]]; then
    echo "配置 slack 通道（Socket Mode）..."
    jq --arg app_token "{{app_token}}" \
       --arg bot_token "{{bot_token}}" \
        '.plugins.allow = ((.plugins.allow // []) | if index("slack") then . else . + ["slack"] end)
        | .plugins.entries["slack"] = ((.plugins.entries["slack"] // {}) + {"enabled": true})
        | .channels["slack"] = {
          "enabled": true,
          "mode": "socket",
          "appToken": $app_token,
          "botToken": $bot_token,
          "groupPolicy": "open",
          "dmPolicy": "open",
          "allowFrom": ["*"]
        }' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
    openclaw plugins enable slack || true
fi

if [[ "{{channel}}" == "discord" ]]; then
    echo "配置 discord 通道"
    jq --arg bot_token "{{bot_token}}" \
       --arg user_id "{{user_id}}"  \
        '.plugins.allow = ((.plugins.allow // []) | if index("discord") then . else . + ["discord"] end)
        | .plugins.entries["discord"] = ((.plugins.entries["discord"] // {}) + {"enabled": true})
        | .channels["discord"] = {
                "enabled": true,
                "token": $bot_token,
                "dmPolicy": "allowlist",
                "allowFrom": [$user_id]
            }
        ' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
    openclaw plugins enable discord || true
fi

if [[ "{{channel}}" == "wecom_app" ]]; then
    # .channels.wecom.agent 的字段在新旧插件中完全一致，
    # 但需要判断 enable 哪个插件。plugins.allow / plugins.entries 中只保留当前插件，
    # 移除另一个，避免两个 wecom 系列插件同时出现。
    _wecom_openclaw_version="$(_get_openclaw_version)"
    _has_sunnoy_wecom=false
    compgen -G "$HOME/.openclaw/npm/projects/sunnoy-wecom-*" >/dev/null 2>&1 && _has_sunnoy_wecom=true

    if [ -d "$HOME/.openclaw/extensions/wecom-openclaw-plugin" ] || \
       [ -d "$HOME/.openclaw/npm/node_modules/@wecom/wecom-openclaw-plugin" ] || \
       compgen -G "$HOME/.openclaw/npm/projects/wecom-wecom-openclaw-plugin-*" >/dev/null; then
        _has_official_wecom=true
    else
        _has_official_wecom=false
    fi

    if [ -n "$_wecom_openclaw_version" ] && _openclaw_version_ge "2026.7.1"; then
        # >= 2026.7.01: @sunnoy/wecom 优先
        if $_has_sunnoy_wecom; then
            _plugin_id="wecom"
            _other_plugin_id="wecom-openclaw-plugin"
        elif $_has_official_wecom; then
            _plugin_id="wecom-openclaw-plugin"
            _other_plugin_id="wecom"
        else
            _plugin_id="wecom"
            _other_plugin_id="wecom-openclaw-plugin"
        fi
    else
        # < 2026.7.01: 原逻辑
        if $_has_official_wecom; then
            _plugin_id="wecom-openclaw-plugin"
            _other_plugin_id="wecom"
        else
            _plugin_id="wecom"
            _other_plugin_id="wecom-openclaw-plugin"
        fi
    fi
    echo "配置 wecom_app 通道..."
    # 注意：必须在调用 `openclaw plugins enable` 之前先用 jq 把 channels / plugins.entries / plugins.allow
    # 一次性写完，避免 enable 触发的 Config overwrite 与后续 jq 写入产生 hash 冲突
    # （ConfigMutationConflictError）。enable 操作放到 jq 之后执行。
    jq --arg plugin_id "$_plugin_id" --arg other_id "$_other_plugin_id" \
        '.plugins.allow = ((.plugins.allow // []) | map(select(. != $other_id)))
        | .plugins.allow = ((.plugins.allow // []) | if index($plugin_id) then . else . + [$plugin_id] end)
        | if (.plugins.entries // {} | has($other_id)) then del(.plugins.entries[$other_id]) else . end
        | .plugins.entries[$plugin_id] = ((.plugins.entries[$plugin_id] // {}) + {"enabled": true})
        | .channels.wecom.enabled = true
        | .channels.wecom.agent = {
          "corpId": "{{corp_id}}",
          "corpSecret": "{{corp_secret}}",
          "agentId": "{{agent_id}}",
          "token": "{{token}}",
          "encodingAESKey": "{{encoding_aes_key}}",
          "welcomeText": "你好！我是 AI 助手",
          "dm": {"policy": "open"}
        }' \
        "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
    _enable_npm_plugin "$_plugin_id" "$_plugin_id"
fi

if [[ "{{channel}}" == "openclaw-weixin" ]]; then
    # 配置 openclaw-weixin 通道前，先 enable openclaw-weixin 插件
    _enable_npm_plugin "openclaw-weixin" "openclaw-weixin"
fi

if [[ "{{channel}}" == "whatsapp" ]]; then
    # 配置 whatsapp 通道前，先 enable whatsapp 插件
    _enable_npm_plugin "whatsapp" "whatsapp"
fi

if [[ "{{channel}}" == "lightclawbot" ]]; then
    # 配置 lightclawbot 通道前，先 enable lightclawbot 插件
    _enable_npm_plugin "lightclawbot" "lightclawbot"
fi

if [[ "{{channel}}" == "adp-openclaw" ]]; then
    # 配置 adp-openclaw 通道前，先 enable adp-openclaw 插件
    _enable_npm_plugin "adp-openclaw" "adp-openclaw"
fi

if [[ "{{channel}}" == "yuanbao" ]]; then
    # 配置 yuanbao 通道前，先 enable openclaw-plugin-yuanbao 插件
    _enable_npm_plugin "openclaw-plugin-yuanbao" "openclaw-plugin-yuanbao"
fi

if [[ "{{channel}}" == "msteams" ]]; then
    if openclaw plugins list --json 2>/dev/null \
        | sed -n '/^{/,/^}/p' \
        | jq -e '.plugins[] | select(.id == "msteams")' >/dev/null 2>&1; then
        echo "✓ Microsoft Teams 插件已安装，跳过安装"
    else
        echo "安装 Microsoft Teams 插件..."
        openclaw plugins install @openclaw/msteams
        echo "✓ Microsoft Teams 插件已安装"
    fi
    openclaw plugins enable msteams || true
    echo "✓ Microsoft Teams 插件已启用"

    echo "配置 Microsoft Teams 通道..."
    jq --arg appid "{{app_id}}" \
       --arg secret "{{app_secret}}" \
       --arg tenant "{{tenant_id}}" \
       --argjson port "{{webhook_port}}" \
       --arg path "{{webhook_path}}" \
        '.plugins.allow = ((.plugins.allow // []) | if index("msteams") then . else . + ["msteams"] end)
         | .plugins.entries["msteams"] = ((.plugins.entries["msteams"] // {}) + {"enabled": true})
         | .channels.msteams = {
          "enabled": true,
          "appId": $appid,
          "appPassword": $secret,
          "tenantId": $tenant,
          "webhook": {
            "port": $port,
            "path": $path
          },
          "dmPolicy": "open",
          "groupPolicy": "open",
          "allowFrom": ["*"]
        }' "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
    echo "✓ Microsoft Teams 通道配置已写入"
    echo "  Endpoint: {{teams_endpoint}}"
fi

# Custom channel: server config + user credentials are pre-built as channel_config JSON by Go layer
if [[ "{{is_custom}}" == "true" ]]; then
    echo "配置自定义通道 {{channel}}..."
    echo '{{channel_config}}' | jq --arg ch "{{channel}}" \
        '. as $cfg | input | .channels[$ch] = $cfg' \
        - "$HOME/.openclaw/openclaw.json" > /tmp/openclaw.json
fi

if [ -f /tmp/openclaw.json ]; then
    echo "备份并写入配置文件..."
    cp "$HOME/.openclaw/openclaw.json" "$HOME/.openclaw/openclaw.json.bak.$(date +%y-%m-%dT%H:%M:%S)"
    mv /tmp/openclaw.json "$HOME/.openclaw/openclaw.json"
    echo "✓ 通道 {{channel}} 配置已写入"
else
    echo "✓ 通道 {{channel}} 无需修改配置文件（仅启用插件）"
fi
echo "重启 gateway..."
systemctl --user restart openclaw-gateway
echo "✓ gateway 已重启"

echo ""
echo "=== 通道 {{channel}} 配置完成 ==="
