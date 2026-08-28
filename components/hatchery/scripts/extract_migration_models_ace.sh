#!/bin/bash
# extract_migration_models_ace.sh (LightclawACE)
# 从 ~/.lightclaw/lightclaw.json 提取当前激活的可迁移模型，输出 JSON。
#
# lightclaw.json 结构：
#   {
#     "models": {
#       "providers": {
#         "providerKey": {
#           "baseUrl": "...", "apiKey": "...",
#           "api": "anthropic-messages",   # 仅 anthropic 显式声明，否则为 openai-completions
#           "models": [{"id":"...", "input":["text","image"], ...}]
#         }
#       },
#       "activeLlm": "providerKey:modelId"  # 空表示未配置，直接返回 []
#     }
#   }

set -uo pipefail
export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

CONFIG="$HOME/.lightclaw/lightclaw.json"

if [ ! -f "$CONFIG" ] || ! command -v jq >/dev/null 2>&1; then
    echo '{"agent_type":"lightclawace","models":[]}'
    exit 0
fi

CONFIG_JSON=$(cat "$CONFIG")

ACTIVE_LLM=$(echo "$CONFIG_JSON" | jq -r '.models.activeLlm // ""')
if [ -z "$ACTIVE_LLM" ]; then
    echo '{"agent_type":"lightclawace","models":[]}'
    exit 0
fi

# activeLlm 格式：provider:model 或 provider/model
if echo "$ACTIVE_LLM" | grep -q ':'; then
    PROVIDER_KEY="${ACTIVE_LLM%%:*}"
    MODEL_ID="${ACTIVE_LLM##*:}"
else
    PROVIDER_KEY="${ACTIVE_LLM%%/*}"
    MODEL_ID="${ACTIVE_LLM##*/}"
fi
[ -z "$PROVIDER_KEY" ] || [ -z "$MODEL_ID" ] && { echo '{"agent_type":"lightclawace","models":[]}'; exit 0; }

PROVIDER_JSON=$(echo "$CONFIG_JSON" | jq -c --arg k "$PROVIDER_KEY" \
    '.models.providers[$k] // empty')
if [ -z "$PROVIDER_JSON" ]; then
    echo '{"agent_type":"lightclawace","models":[]}'
    exit 0
fi

BASE_URL=$(echo "$PROVIDER_JSON" | jq -r '.baseUrl // ""')
API_KEY=$(echo "$PROVIDER_JSON"  | jq -r '.apiKey  // ""')
if [ -z "$BASE_URL" ] || [ -z "$API_KEY" ]; then
    echo '{"agent_type":"lightclawace","models":[]}'
    exit 0
fi

# api_mode：本项目只支持 openai-completions / anthropic-messages
# api 字段不存在时默认 openai-completions（anthropic 会显式声明 api 字段）
RAW_MODE=$(echo "$PROVIDER_JSON" | jq -r '.api // ""')
case "$RAW_MODE" in
    openai-completions|chat_completions|"") API_MODE="openai-completions" ;;
    anthropic-messages|anthropic_messages)  API_MODE="anthropic-messages" ;;
    *)
        echo '{"agent_type":"lightclawace","models":[]}'
        exit 0
        ;;
esac

CTX_LEN=$(echo "$PROVIDER_JSON" | jq -r '(.models[0].contextWindow // 128000) | tonumber? // 128000')

# input_types：过滤只保留 text/image，无合法值则默认 ["text"]
INPUT_TYPES=$(echo "$PROVIDER_JSON" | jq -c '
    (.models[0].input // ["text"])
    | map(select(. == "text" or . == "image"))
    | if length == 0 then ["text"] else . end')

jq -cn \
    --arg model_id "$MODEL_ID" \
    --arg base_url "$BASE_URL" \
    --arg api_key "$API_KEY" \
    --arg api_mode "$API_MODE" \
    --argjson ctx_len "$CTX_LEN" \
    --argjson input_types "$INPUT_TYPES" \
    '{agent_type:"lightclawace",models:[{
        role:"primary",
        model_id:$model_id,
        model_name:$model_id,
        base_url:$base_url,
        api_key:$api_key,
        api_mode:$api_mode,
        context_len:$ctx_len,
        input_types:$input_types
    }]}'
