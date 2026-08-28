#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化 ==========
# ACE 脚本未加入 hatchery rootRequiredTATScripts 白名单（controller/tat.go），
# 由 TAT 以实例普通用户身份执行，无权写 /var/log/clawpro，日志落到用户目录。
LOG_DIR="$HOME/.lightclaw/logs"
mkdir -p "$LOG_DIR"
SCRIPT_NAME="del_channel_ace"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# Parameters (与 openclaw del_channel.sh 契约对齐):
#   {{channel}} - channel key（ACE 白名单: feishu/weixin/qqbot/wecom；不支持 dingtalk）
#
# 与 openclaw del_channel.sh 的差异：
#   - 配置文件: ~/.openclaw/openclaw.json → ~/.lightclaw/lightclaw.json
#   - 内置 channel 执行 reset（置 enabled=false + 清凭据），不 del key
#     原因：lightclaw 用 Pydantic 模型，del key 下次加载会以默认值重建，reset 才能生效
#   - 重启命令: openclaw-gateway → lightclaw
#   - 无 plugins disable（ACE 无 plugin 体系）
#   - weixin 额外清理 ~/.lightclaw/weixin/{accounts,sync_state,context_tokens}

CHANNEL="{{channel}}"
# 前端契约使用 "openclaw-weixin"，脚本内部（lightclaw.json key）使用 "weixin"，在入口处统一转换
[ "$CHANNEL" = "openclaw-weixin" ] && CHANNEL="weixin"
CONFIG="$HOME/.lightclaw/lightclaw.json"
LIGHTCLAW_HOME="$HOME/.lightclaw"

echo "=== ACE 删除通道: $CHANNEL ==="

if [ ! -f "$CONFIG" ]; then
    echo "配置文件不存在，跳过: $CONFIG"
    exit 0
fi

# ===== reset 内置 channel 到默认态 =====
rm -f /tmp/lightclaw.json

case "$CHANNEL" in
    feishu)
        jq '.channels["feishu"] = {
          "enabled": false,
          "botPrefix": "",
          "appId": "",
          "appSecret": "",
          "encryptKey": "",
          "verificationToken": "",
          "mediaDir": "~/.lightclaw/media"
        }' "$CONFIG" > /tmp/lightclaw.json
        ;;
    weixin)
        jq '.channels["weixin"] = {
          "enabled": false,
          "botPrefix": "",
          "mediaDir": "~/.lightclaw/media",
          "showToolDetails": true,
          "accounts": []
        }' "$CONFIG" > /tmp/lightclaw.json
        ;;
    qqbot)
        jq '.channels["qqbot"] = {
          "enabled": false,
          "botPrefix": "",
          "appId": "",
          "clientSecret": "",
          "stt": {},
          "tts": {}
        }' "$CONFIG" > /tmp/lightclaw.json
        ;;
    wecom)
        # lightclaw.json 中 wecom 保持平铺格式（ACE 原生结构），
        # 归一化到 .bot 子对象的工作由 list_channels_ace.sh 在读取时完成。
        jq '.channels["wecom"] = {
          "enabled": false,
          "botPrefix": "",
          "botId": "",
          "secret": "",
          "name": "",
          "websocketUrl": "",
          "dmPolicy": "open",
          "allowFrom": [],
          "groupPolicy": "open",
          "groupAllowFrom": [],
          "groups": {},
          "sendThinkingMessage": true,
          "mediaDir": "~/.lightclaw/media",
          "mediaLocalRoots": []
        }' "$CONFIG" > /tmp/lightclaw.json
        ;;
    *)
        # 自定义/插件通道：直接删 key
        if jq -e ".channels[\"$CHANNEL\"]" "$CONFIG" >/dev/null 2>&1; then
            jq "del(.channels[\"$CHANNEL\"])" "$CONFIG" > /tmp/lightclaw.json
        else
            echo "⚠ 通道 $CHANNEL 不存在，跳过"
            exit 0
        fi
        ;;
esac

# ===== 写回配置 =====
if [ -f /tmp/lightclaw.json ]; then
    cp "$CONFIG" "$CONFIG.bak.$(date +%y-%m-%dT%H:%M:%S)"
    mv /tmp/lightclaw.json "$CONFIG"
    echo "✓ 通道 $CHANNEL 已 reset/删除"
fi

# ===== weixin 额外文件清理（与 ACE delete_channel.sh 对齐）=====
if [[ "$CHANNEL" == "weixin" ]]; then
    for sub in accounts sync_state context_tokens; do
        DIR="$LIGHTCLAW_HOME/weixin/$sub"
        if [ -d "$DIR" ]; then
            rm -rf "$DIR"
            echo "✓ 已清理 $DIR"
        fi
    done
fi

echo "重启 lightclaw..."
lightclaw restart
if [ $? -ne 0 ]; then
    echo "✗ lightclaw 重启失败"
    exit 1
else
    echo "✓ lightclaw 已重启"
fi


echo ""
echo "=== 通道 $CHANNEL 删除完成 ==="
