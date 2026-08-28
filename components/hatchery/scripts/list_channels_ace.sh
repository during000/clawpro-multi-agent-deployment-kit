#!/bin/bash
#
# list_channels_ace.sh
#
# 作用：列出 LightClaw-ACE 实例当前 ~/.lightclaw/lightclaw.json 内的通道配置。
# 契约：stdout 末行输出 JSON object：
#   { "feishu": {"enabled": ..., ...}, "weixin": {...}, ... }
# 与 scripts/list_channels.sh（openclaw）保持同一形状。
#
# 实现：直接 jq 读取 .channels；不存在则输出 "{}"。
#
set -uo pipefail

export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u 2>/dev/null || echo 0)"

# ACE 脚本未加入 hatchery rootRequiredTATScripts 白名单（controller/tat.go），
# 由 TAT 以实例普通用户身份执行，无权写 /var/log/clawpro，日志落到用户目录。
LOG_DIR="$HOME/.lightclaw/logs"
LOG_FILE="$LOG_DIR/list_channels_ace.log"
mkdir -p "$LOG_DIR" 2>/dev/null || true

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> "$LOG_FILE" 2>/dev/null || true; }

CFG="$HOME/.lightclaw/lightclaw.json"

if [ ! -f "$CFG" ]; then
    log "config not found: $CFG"
    echo '{}'
    exit 0
fi
if ! command -v jq >/dev/null 2>&1; then
    log "jq not found"
    echo '{}'
    exit 0
fi

# 读取 .channels。lightclaw.json 的 channels 本身就是 {name: config} map，
# 但不同 channel 的**存储结构**与 hatchery/openclaw 前端契约不一致，
# 需要在脚本层做一次归一化，再交给上层统一处理。
#
# 已知差异（见 set_channel_ace.sh 对照）：
#   - wecom：lightclaw.json 中 wecom 保持 ACE 原生的平铺格式（botId/secret/... 直接在
#     .channels.wecom 根下），但 hatchery API（/api/openclaw/channels）返回给前端时
#     需要将 bot 相关字段归一化到 .bot 子对象（与 openclaw 前端契约对齐）。
#     → 归一化逻辑：用 jq 把根下的 bot 相关字段搬运到 .bot；仍保留 enabled / botPrefix /
#       mediaDir / mediaLocalRoots / groups / groupPolicy / groupAllowFrom / sendThinkingMessage
#       这类"通道级"字段在 wecom 根下不动。
#
# 其他 channel（feishu/weixin/qqbot/自定义）结构已经与契约一致，直接透传。
#
# 容错：jq 过滤失败时保底回退原始 .channels（避免 jq 表达式 bug 吞掉所有配置）。
RAW_CHANNELS="$(jq -c '.channels // {}' "$CFG" 2>> "$LOG_FILE")"

if [ -z "$RAW_CHANNELS" ] || [ "$RAW_CHANNELS" = "null" ]; then
    log "parse failed"
    echo '{}'
    exit 0
fi

# wecom bot 字段白名单：这些字段会被搬运到 .bot 子对象。
# 与 set_channel_ace.sh:104-119 写入的字段（除 enabled / botPrefix / mediaDir /
# mediaLocalRoots / groups / groupPolicy / groupAllowFrom / sendThinkingMessage
# 这些属于"通道级"而非"bot 级"的字段外）对齐。
#
# 注意：不使用"把 wecom 下除 enabled 外全部搬进 bot"的粗暴方式，因为
# groups/groupPolicy/groupAllowFrom/mediaDir/mediaLocalRoots/sendThinkingMessage
# 是整个 wecom 通道级的配置，与 bot 这一"机器人账号"维度正交。显式白名单
# 既能保持两层职责清晰，也能避免未来新增"通道级"字段时被误搬。
CHANNELS="$(
    echo "$RAW_CHANNELS" | jq -c '
      if .wecom == null or (.wecom | type) != "object" then
        .
      else
        .wecom as $w
        | ($w | to_entries | map(select(.key as $k | [
            "botId", "secret", "name", "websocketUrl",
            "dmPolicy", "allowFrom"
          ] | index($k))) | from_entries) as $botFields
        | .wecom = (
            ($w | with_entries(select(.key as $k | [
              "botId", "secret", "name", "websocketUrl",
              "dmPolicy", "allowFrom"
            ] | index($k) | not)))
            | .bot = (($w.bot // {}) + $botFields)
          )
      end
    ' 2>> "$LOG_FILE"
)"

# jq 失败兜底：任何非法输出（空/null）回退原始 channels，保证前端至少能显示"未归一化但完整" 的配置。
if [ -z "$CHANNELS" ] || [ "$CHANNELS" = "null" ]; then
    log "wecom shape normalize failed, falling back to raw channels"
    CHANNELS="$RAW_CHANNELS"
fi

# 归一化 weixin → openclaw-weixin（前端契约统一使用 openclaw-weixin）
CHANNELS="$(echo "$CHANNELS" | jq -c 'if has("weixin") then .["openclaw-weixin"] = .weixin | del(.weixin) else . end' 2>/dev/null || echo "$CHANNELS")"

echo "$CHANNELS"
