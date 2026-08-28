#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化 ==========
# ACE 脚本未加入 hatchery 的 rootRequiredTATScripts 白名单（controller/tat.go），
# 由 TAT 以实例普通用户身份执行，无权写 /var/log/clawpro，日志落到用户目录。
SCRIPT_NAME="set_model_ace"
LOG_DIR="$HOME/.lightclaw/logs"
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# 将 stdout 和 stderr 同时输出到终端和日志文件
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# Parameters (substituted by TAT before execution, 与 set_model.sh 契约对齐):
#   {{valueb64}}  - base64(完整 provider JSON object)
#   {{provider}}  - provider key, e.g. hatchery-glm-4-plus, custom-my-model
#   {{model}}     - model id (lowercase)
#
# 与 openclaw set_model.sh 的差异：
#   - 配置文件: ~/.openclaw/openclaw.json → ~/.lightclaw/lightclaw.json
#   - activeLlm 分隔符: openclaw "provider/model" → lightclaw "provider:model"
#     （以实际 ~/.lightclaw/lightclaw.json 里的存量格式为准；ACE 官方
#      add_model.sh 历史上写 "provider/model"，但实际运行时内容为 ":"，
#      这里按存量格式写入，避免 LLM 解析失败）
#   - 重启命令: openclaw-gateway → lightclaw (systemctl 用户态 → 系统 systemctl)

# 先把 TAT 占位符落到 shell 变量，避免 value JSON 直接拼到 jq 命令行
# 被 shell 特殊字符破坏。
VALUE_B64='{{valueb64}}'
# base64 解码（与 openclaw set_model.sh 一致，避免 shell 元字符解析风险）
VALUE_JSON="$(printf '%s' "$VALUE_B64" | base64 -d)" || {
    echo "错误: valueb64 base64 解码失败，中止执行"; exit 1
}
PROVIDER='{{provider}}'
MODEL='{{model}}'

# 占位符替换失败保护（手工 bash 执行或 TAT 未启用 EnableParameter 时会出现字面量）
if [ -z "$PROVIDER" ] || [[ "$PROVIDER" == *"{{"*"}}"* ]] \
   || [ -z "$MODEL" ] || [[ "$MODEL" == *"{{"*"}}"* ]] \
   || [ -z "$VALUE_B64" ] || [[ "$VALUE_B64" == *"{{"*"}}"* ]]; then
    echo "✗ TAT 占位符未被替换: provider='${PROVIDER}' model='${MODEL}' value_b64='${VALUE_B64:0:40}...'"
    exit 1
fi

LIGHTCLAW_HOME="$HOME/.lightclaw"
CONFIG="$LIGHTCLAW_HOME/lightclaw.json"

mkdir -p "$LIGHTCLAW_HOME"

echo "=== ACE set_model: ${PROVIDER}/${MODEL} ==="

# Backup existing config (skip if file doesn't exist yet)
if [ -f "$CONFIG" ]; then
    cp "$CONFIG" "$CONFIG.bak.$(date +%Y-%m-%dT%H:%M:%S)"
    BASE_JSON="$(cat "$CONFIG")"
else
    BASE_JSON='{"models":{"providers":{},"activeLlm":""}}'
fi

# 写入 provider 配置 + 激活 activeLlm（使用冒号分隔符，与 lightclaw.json 存量格式对齐）
# 补全 isCustom / extraModels / chatModel / name 字段，与 lightclaw 实际 provider 格式对齐。
# isCustom 固定为 true：对 LightClaw ACE 来说，hatchery 代理模型与用户自定义模型均属 custom。
echo "$BASE_JSON" \
    | jq --arg provider "$PROVIDER" --arg model "$MODEL" \
         --argjson newval "$VALUE_JSON" \
        '.models.providers[$provider] = $newval
         | .models.providers[$provider].isCustom    = true
         | .models.providers[$provider].extraModels = (.models.providers[$provider].extraModels // [])
         | .models.providers[$provider].chatModel   = (.models.providers[$provider].chatModel // "")
         | .models.providers[$provider].name        = (.models.providers[$provider].name // $provider)
         | .models.activeLlm = ($provider + ":" + $model)' \
    > /tmp/lightclaw.json

cp /tmp/lightclaw.json "$CONFIG"
rm /tmp/lightclaw.json

echo "✓ provider ${PROVIDER} 已写入，activeLlm = ${PROVIDER}:${MODEL}"

# 重启 lightclaw（ACE 使用系统 systemctl，服务名 lightclaw）
echo "重启 lightclaw..."
lightclaw restart
echo "✓ lightclaw 已重启"

echo ""
echo "=== ACE set_model 完成 ==="
