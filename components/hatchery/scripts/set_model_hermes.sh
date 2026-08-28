#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:/usr/local/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 拉取/更新 harness CLI ==========
ensure_harness_cli() {
    local HARNESS_URL="https://agentone-1388575217.cos.ap-guangzhou.myqcloud.com/harness-linux-amd64"
    local HARNESS_BIN="$HOME/.local/bin/harness"
    mkdir -p "$(dirname "$HARNESS_BIN")" 2>/dev/null || true
    echo "ℹ 拉取 harness CLI: $HARNESS_URL"
    if curl -fsSL --connect-timeout 10 --max-time 60 --retry 2 --retry-delay 2 \
        "$HARNESS_URL" -o "${HARNESS_BIN}.tmp" 2>/dev/null; then
        chmod +x "${HARNESS_BIN}.tmp"
        mv -f "${HARNESS_BIN}.tmp" "$HARNESS_BIN"
        echo "✓ harness CLI 已更新: $HARNESS_BIN"
    else
        rm -f "${HARNESS_BIN}.tmp" 2>/dev/null || true
        if command -v harness >/dev/null 2>&1; then
            echo "⚠ harness CLI 下载失败，沿用已有版本: $(command -v harness)"
        else
            echo "✗ harness CLI 下载失败且本地无已有版本" >&2
            return 1
        fi
    fi
}

# ========== 日志系统初始化 ==========
LOG_DIR="$HOME/.hermes/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
SCRIPT_NAME="set_model_hermes"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# Parameters (substituted by TAT before execution, 与 openclaw set_model.sh 契约对齐):
#   {{valueb64}}  - base64(完整 provider JSON object)
#                   JSON 结构: {baseUrl, apiKey, auth, api, models:[{id,name,reasoning,input,contextWindow}]}
#   {{provider}}  - provider key, e.g. hatchery-glm-4-plus, custom-my-model
#   {{model}}     - model id (lowercase)
#
# 与 openclaw set_model.sh 的差异：
#   - 不直写配置文件，而是调用 `harness model set` CLI（harness 内部会原子写 ~/.hermes/config.yaml 并自动重启网关）
#   - 从 value JSON 中抽出 baseUrl / apiKey / api / contextWindow 转成 flag 传给 harness
#   - 重启网关由 harness model set 内部完成，无需外部再触发

VALUE_B64='{{valueb64}}'
# base64 解码（与 openclaw set_model.sh 一致，避免 shell 元字符解析风险）
VALUE_JSON="$(printf '%s' "$VALUE_B64" | base64 -d)" || {
    echo "错误: valueb64 base64 解码失败，中止执行"; exit 1
}
PROVIDER='{{provider}}'
MODEL='{{model}}'

echo "=== Hermes set_model: provider=${PROVIDER} model=${MODEL} ==="

# 依赖检查
ensure_harness_cli || exit 1
command -v harness >/dev/null 2>&1 || { echo "✗ harness CLI 不存在"; exit 1; }
command -v jq      >/dev/null 2>&1 || { echo "✗ jq 不存在"; exit 1; }

# 从 {{value}} JSON 抽字段
BASE_URL=$(echo "$VALUE_JSON" | jq -r '.baseUrl // empty')
API_KEY=$( echo "$VALUE_JSON" | jq -r '.apiKey  // empty')
API_MODE=$(echo "$VALUE_JSON" | jq -r '.api     // empty')
CTX_LEN=$( echo "$VALUE_JSON" | jq -r '.models[0].contextWindow // empty')

if [ -z "$MODEL" ]; then
    echo "✗ 模型 id 为空"; exit 1
fi
if [ -z "$BASE_URL" ]; then
    echo "✗ baseUrl 为空"; exit 1
fi
if [ -z "$API_KEY" ]; then
    echo "✗ apiKey 为空"; exit 1
fi

# harness model set --name 约束：
#   hermes 内置 provider（harness model set 原生识别的 --name 值）直接透传；
#   hatchery 代理模型或其他非内置 provider 一律强制为 "custom"。
#
# 内置 provider 列表（对应 hermes model 交互菜单中的选项，不含 "Cancel"）：
#   nous / openrouter / anthropic / openai-codex / xiaomi / qwen-oauth /
#   github-copilot / github-copilot-acp / huggingface / google-ai-studio /
#   deepseek / xai / zhipu / moonshot / moonshot-cn / minimax / minimax-cn /
#   dashscope / arcee / kilo / opencode-zen / opencode-go / vercel / custom
#
# hatchery 预配置模型的 provider 固定为 "hatchery"，不在内置列表中 → 走 custom。
HARNESS_NAME="$PROVIDER"
case "$PROVIDER" in
    nous|openrouter|anthropic|openai-codex|xiaomi|qwen-oauth|\
    github-copilot|github-copilot-acp|huggingface|google-ai-studio|\
    deepseek|xai|zhipu|moonshot|moonshot-cn|minimax|minimax-cn|\
    dashscope|arcee|kilo|opencode-zen|opencode-go|vercel|custom)
        echo "ℹ 内置 provider: --name=${PROVIDER}"
        ;;
    *)
        HARNESS_NAME="custom"
        echo "ℹ 非内置 provider: --name 强制设为 custom (原始 provider=${PROVIDER})"
        ;;
esac

# 组装 harness model set 参数
HARNESS_ARGS=( model set --model "$MODEL" --base-url "$BASE_URL" --api-key "$API_KEY" --name "$HARNESS_NAME" )

# api-mode 仅在 harness 支持的值集内才透传
# harness 支持: chat_completions / codex_responses / anthropic_messages / bedrock_converse
# 也接受别名: openai-completions → chat_completions, anthropic-messages → anthropic_messages
if [ -n "$API_MODE" ]; then
    case "$API_MODE" in
        chat_completions|codex_responses|anthropic_messages|bedrock_converse|openai-completions|anthropic-messages)
            HARNESS_ARGS+=( --api-mode "$API_MODE" )
            ;;
        *)
            echo "⚠ 未识别 api-mode=${API_MODE}，跳过"
            ;;
    esac
fi

# contextLength 仅在 >0 时传
if [ -n "$CTX_LEN" ] && [ "$CTX_LEN" != "null" ] && [ "$CTX_LEN" -gt 0 ] 2>/dev/null; then
    HARNESS_ARGS+=( --context-length "$CTX_LEN" )
fi

echo "执行: harness ${HARNESS_ARGS[*]}"
# harness model set 内部会自动重启 gateway
harness "${HARNESS_ARGS[@]}"

echo "✓ Hermes model 配置已更新（harness 内部已重启 gateway）"
echo ""
echo "=== Hermes set_model 完成 ==="
