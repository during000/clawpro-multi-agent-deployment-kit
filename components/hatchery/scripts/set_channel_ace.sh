#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化 ==========
# ACE 脚本未加入 hatchery rootRequiredTATScripts 白名单（controller/tat.go），
# 由 TAT 以实例普通用户身份执行，无权写 /var/log/clawpro，日志落到用户目录。
LOG_DIR="$HOME/.lightclaw/logs"
mkdir -p "$LOG_DIR"
SCRIPT_NAME="set_channel_ace"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# Parameters (与 openclaw set_channel.sh 契约对齐):
#   {{channel}}          - channel key（ACE 白名单: feishu/weixin/qqbot/wecom；不支持 dingtalk）
#   {{app_id}}           - feishu / qqbot
#   {{app_secret}}       - feishu / qqbot（注：ACE 的 qqbot 配置字段名是 clientSecret，
#                          但 hatchery/前端契约统一为 app_secret，脚本内部负责映射）
#   {{bot_id}}           - wecom
#   {{secret}}           - wecom
#   {{bot_name}}         - wecom (display name)
#   {{is_custom}}        - true/false
#   {{channel_config}}   - 自定义通道完整 JSON
#
# 与 openclaw set_channel.sh 的差异：
#   - 配置文件: ~/.openclaw/openclaw.json → ~/.lightclaw/lightclaw.json
#   - 无 openclaw plugins enable（ACE 无 plugin 体系，channel 直接在 .channels 下启用）
#   - 重启命令: openclaw-gateway → lightclaw
#   - weixin 仅写配置空壳，实际扫码由独立 weixin_bot_creator.py 完成
#   - ACE 通道白名单: feishu/weixin/qqbot/wecom（hatchery 侧 `AIChannel` 收录范围；dingtalk 不支持）

CHANNEL="{{channel}}"
# 前端契约使用 "openclaw-weixin"，脚本内部（lightclaw.json key）使用 "weixin"，在入口处统一转换
[ "$CHANNEL" = "openclaw-weixin" ] && CHANNEL="weixin"
CONFIG="$HOME/.lightclaw/lightclaw.json"

echo "=== ACE 配置通道: $CHANNEL ==="

if [ ! -f "$CONFIG" ]; then
    echo "✗ 配置文件不存在: $CONFIG"
    exit 1
fi

# 清理上次运行可能遗留的临时文件
rm -f /tmp/lightclaw.json

# ===== 白名单校验（双保险：Go 层已过滤，这里兜底）=====
case "$CHANNEL" in
    feishu|weixin|qqbot|wecom)
        ;;
    *)
        if [[ "{{is_custom}}" != "true" ]]; then
            echo "✗ ACE 不支持的通道类型: $CHANNEL"
            echo "   支持列表: feishu weixin qqbot wecom（dingtalk 不支持）"
            exit 1
        fi
        ;;
esac

# ===== 按 channel 分派配置写入 =====

if [[ "$CHANNEL" == "feishu" ]]; then
    echo "配置 feishu 通道..."
    jq '.channels["feishu"] = {
      "enabled": true,
      "botPrefix": "",
      "appId": "{{app_id}}",
      "appSecret": "{{app_secret}}",
      "encryptKey": "",
      "verificationToken": "",
      "mediaDir": "~/.lightclaw/media"
    }' "$CONFIG" > /tmp/lightclaw.json
fi

if [[ "$CHANNEL" == "weixin" ]]; then
    echo "配置 weixin 通道（空壳，实际扫码由独立流程触发）..."
    jq '.channels["weixin"] = {
      "enabled": true,
      "botPrefix": "",
      "mediaDir": "~/.lightclaw/media",
      "showToolDetails": true,
      "accounts": []
    }' "$CONFIG" > /tmp/lightclaw.json
fi

if [[ "$CHANNEL" == "qqbot" ]]; then
    # 字段映射：hatchery/前端契约为 {app_id, app_secret}（对齐 ChannelParams["qqbot"]），
    # 写入 lightclaw.json 时 app_secret 映射到 clientSecret 字段（ACE Python 端契约）。
    echo "配置 qqbot 通道..."
    jq '.channels["qqbot"] = {
      "enabled": true,
      "botPrefix": "",
      "appId": "{{app_id}}",
      "clientSecret": "{{app_secret}}",
      "stt": {},
      "tts": {}
    }' "$CONFIG" > /tmp/lightclaw.json
fi

if [[ "$CHANNEL" == "wecom" ]]; then
    echo "配置 wecom 通道..."
    # lightclaw.json 中 wecom 保持平铺格式（ACE 原生结构），
    # 归一化到 .bot 子对象的工作由 list_channels_ace.sh 在读取时完成。
    jq '.channels["wecom"] = {
      "enabled": true,
      "botPrefix": "",
      "botId": "{{bot_id}}",
      "secret": "{{secret}}",
      "name": "{{bot_name}}",
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
fi

# 自定义通道：由 Go 层预先组装完整 JSON
if [[ "{{is_custom}}" == "true" ]]; then
    echo "配置自定义通道 $CHANNEL..."
    echo '{{channel_config}}' | jq --arg ch "$CHANNEL" \
        '. as $cfg | input | .channels[$ch] = $cfg' \
        - "$CONFIG" > /tmp/lightclaw.json
fi

# ===== 写入配置文件 =====
if [ -f /tmp/lightclaw.json ]; then
    echo "备份并写入配置文件..."
    cp "$CONFIG" "$CONFIG.bak.$(date +%y-%m-%dT%H:%M:%S)"
    mv /tmp/lightclaw.json "$CONFIG"
    echo "✓ 通道 $CHANNEL 配置已写入"
else
    echo "⚠ 通道 $CHANNEL 无配置写入（未命中任何分支）"
fi

echo "重启 lightclaw..."
lightclaw restart
echo "✓ lightclaw 已重启"

echo ""
echo "=== 通道 $CHANNEL 配置完成 ==="
