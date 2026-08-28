#!/bin/bash
# extract_migration_models.sh (OpenClaw)
# 从 openclaw.json 提取激活的可迁移模型，输出 JSON。
#
# 输出（stdout 最后一行）：
#   {"agent_type":"openclaw","models":[
#     {"role":"primary","model_id":"gpt-4o","model_name":"gpt-4o",
#      "base_url":"https://...","api_key":"sk-xxx",
#      "api_mode":"openai-completions","context_len":128000,"input_types":["text"]}
#   ]}
# 无可迁移模型时：{"agent_type":"openclaw","models":[]}

set -uo pipefail
export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

CONFIG="$HOME/.openclaw/openclaw.json"

if [ ! -f "$CONFIG" ] || ! command -v jq >/dev/null 2>&1; then
    echo '{"agent_type":"openclaw","models":[]}'
    exit 0
fi

CONFIG_JSON=$(cat "$CONFIG")

PRIMARY_REF=$(echo "$CONFIG_JSON" | jq -r '.agents.defaults.model.primary // ""')
FALLBACKS_JSON=$(echo "$CONFIG_JSON" | jq -c '.agents.defaults.model.fallbacks // []')

# 有序列表：primary ref 排第一，然后 fallbacks
ALL_REFS=$(echo "$FALLBACKS_JSON" | jq -c --arg p "$PRIMARY_REF" \
    '(if $p != "" then [$p] else [] end) + .')

RESULT_MODELS="[]"
SEEN_IDS=""

while IFS= read -r ref; do
    [ -z "$ref" ] || [ "$ref" = "null" ] && continue

    PROVIDER_KEY="${ref%%/*}"
    SLUG_MODEL_ID="${ref##*/}"
    [ -z "$PROVIDER_KEY" ] || [ -z "$SLUG_MODEL_ID" ] && continue

    PROVIDER_JSON=$(echo "$CONFIG_JSON" | jq -c --arg k "$PROVIDER_KEY" \
        '.models.providers[$k] // empty')
    [ -z "$PROVIDER_JSON" ] && continue

    # 优先从 provider JSON 中取原始 model_id（保真值），兜底用 ref 中的 slug 值
    MODEL_ID=$(echo "$PROVIDER_JSON" | jq -r '.models[0].id // ""')
    [ -z "$MODEL_ID" ] && MODEL_ID="$SLUG_MODEL_ID"

    BASE_URL=$(echo "$PROVIDER_JSON" | jq -r '.baseUrl // ""')
    API_KEY=$(echo "$PROVIDER_JSON"  | jq -r '.apiKey  // ""')
    { [ -z "$BASE_URL" ] || [ -z "$API_KEY" ]; } && continue

    # api_mode：本项目只支持 openai-completions / anthropic-messages
    RAW_MODE=$(echo "$PROVIDER_JSON" | jq -r '.api // ""')
    case "$RAW_MODE" in
        openai-completions|chat_completions)   API_MODE="openai-completions" ;;
        anthropic-messages|anthropic_messages) API_MODE="anthropic-messages" ;;
        *) continue ;;  # 不支持的模式，跳过
    esac

    # 去重（按 model_id）
    echo "$SEEN_IDS" | grep -qxF "$MODEL_ID" && continue
    SEEN_IDS="${SEEN_IDS}
${MODEL_ID}"

    ROLE="fallback"
    [ "$ref" = "$PRIMARY_REF" ] && ROLE="primary"

    CTX_LEN=$(echo "$PROVIDER_JSON" | jq -r '(.models[0].contextWindow // 128000) | tonumber? // 128000')

    # input_types：过滤只保留 text/image，无合法值则默认 ["text"]
    INPUT_TYPES=$(echo "$PROVIDER_JSON" | jq -c '
        (.models[0].input // ["text"])
        | map(select(. == "text" or . == "image"))
        | if length == 0 then ["text"] else . end')

    MODEL_JSON=$(jq -cn \
        --arg role "$ROLE" \
        --arg model_id "$MODEL_ID" \
        --arg base_url "$BASE_URL" \
        --arg api_key "$API_KEY" \
        --arg api_mode "$API_MODE" \
        --argjson ctx_len "$CTX_LEN" \
        --argjson input_types "$INPUT_TYPES" \
        '{role:$role,model_id:$model_id,model_name:$model_id,
          base_url:$base_url,api_key:$api_key,api_mode:$api_mode,
          context_len:$ctx_len,input_types:$input_types}')

    RESULT_MODELS=$(echo "$RESULT_MODELS" | jq -c --argjson m "$MODEL_JSON" '. + [$m]')
done < <(echo "$ALL_REFS" | jq -r '.[]')

jq -cn --argjson models "$RESULT_MODELS" '{agent_type:"openclaw",models:$models}'
