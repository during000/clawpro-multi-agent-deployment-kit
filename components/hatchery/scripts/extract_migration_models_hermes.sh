#!/bin/bash
# extract_migration_models_hermes.sh (Hermes)
# 从 ~/.hermes/config.yaml 提取当前激活的可迁移模型，输出 JSON。
#
# config.yaml 格式（新版，harness model set 写入）：
#   model:
#     api_key: sk-xxx
#     api_mode: chat_completions    # harness 格式，需转换
#     base_url: https://...
#     default: gpt-4o               # 当前激活的 model id
#     provider: my_provider         # 引用 providers: 块中的 provider 名
#
# 注意：model: 块可能没有 api_key/api_mode（通过 provider: 引用 providers: 块中的凭证），
# 此时回退到 providers: 块按 provider 名解析。
#
# Hermes 无 primary/fallback，只有单个激活模型，作为 primary 输出。

set -uo pipefail
export PATH="$HOME/.local/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

CONFIG="$HOME/.hermes/config.yaml"

if [ ! -f "$CONFIG" ]; then
    echo '{"agent_type":"hermes","models":[]}'
    exit 0
fi

# 读取 model: 块下的字段
# 策略：找到 "^model:" 行，然后读取后续缩进行，直到遇到非缩进行为止
get_model_val() {
    local key="$1"
    awk '/^model:/{found=1; next}
         found && /^[^ ]/{found=0}
         found && /^[ \t]+'"$key"':/{
             sub(/^[ \t]+'"$key"':[ \t]*/, "")
             gsub(/^['"'"'"]|['"'"'"]$/, "")
             print; exit
         }' "$CONFIG" | tr -d '\r'
}

# 读取 providers: 块下指定 provider 的字段
# model: 块可能没有 api_key/api_mode（通过 provider: 引用），此时回退到 providers: 块解析
get_provider_val() {
    local provider="$1" key="$2"
    PROV="$provider" KEY="$key" awk '
        BEGIN { prov = ENVIRON["PROV"]; k = ENVIRON["KEY"] }
        /^providers:/ {in_block=1; indent=match($0, /[^ \t]/); next}
        in_block && match($0, /[^ \t]/) && RSTART <= indent {in_block=0}
        in_block && $0 ~ "^[ \t]+" prov ":" {
            in_prov=1; pindent=match($0, /[^ \t]/); next
        }
        in_prov && match($0, /[^ \t]/) && RSTART <= pindent {in_prov=0}
        in_prov && $0 ~ "^[ \t]+" k ":" {
            sub("^[ \t]+" k ":[ \t]*", "")
            gsub(/^['"'"'"]|['"'"'"]$/, "")
            print; exit
        }' "$CONFIG" | tr -d '\r'
}

BASE_URL=$(get_model_val "base_url")
API_KEY=$(get_model_val "api_key")
MODEL_ID=$(get_model_val "default")
RAW_MODE=$(get_model_val "api_mode")
PROVIDER=$(get_model_val "provider")

# model: 块的 api_key / api_mode 可能为空（hermes 切换自定义 provider 时不写这些字段）
# 此时从 providers: / custom_providers: 块按 provider 名解析
if [ -z "$API_KEY" ] && [ -n "$PROVIDER" ]; then
    API_KEY=$(get_provider_val "$PROVIDER" "api_key")
fi
if [ -z "$RAW_MODE" ] && [ -n "$PROVIDER" ]; then
    RAW_MODE=$(get_provider_val "$PROVIDER" "api_mode")
fi
# provider 的 base_url 仅在 model: 块没有时作为回退
if [ -z "$BASE_URL" ] && [ -n "$PROVIDER" ]; then
    BASE_URL=$(get_provider_val "$PROVIDER" "base_url")
fi

if [ -z "$BASE_URL" ] || [ -z "$API_KEY" ] || [ -z "$MODEL_ID" ]; then
    echo '{"agent_type":"hermes","models":[]}'
    exit 0
fi

# api_mode 转换：harness 格式 → 本项目格式，不支持的跳过
case "$RAW_MODE" in
    chat_completions|openai-completions)   API_MODE="openai-completions" ;;
    anthropic_messages|anthropic-messages) API_MODE="anthropic-messages" ;;
    *)
        echo '{"agent_type":"hermes","models":[]}'
        exit 0
        ;;
esac

# 确保 CTX_LEN 是整数（Hermes config 不含 context_length，默认 128000）
CTX_LEN="128000"

if ! command -v jq >/dev/null 2>&1; then
    echo '{"agent_type":"hermes","models":[]}'
    exit 0
fi

# Hermes 没有 input_types 配置，默认 text
jq -cn \
    --arg model_id "$MODEL_ID" \
    --arg base_url "$BASE_URL" \
    --arg api_key "$API_KEY" \
    --arg api_mode "$API_MODE" \
    --argjson ctx_len "${CTX_LEN}" \
    '{agent_type:"hermes",models:[{
        role:"primary",
        model_id:$model_id,
        model_name:$model_id,
        base_url:$base_url,
        api_key:$api_key,
        api_mode:$api_mode,
        context_len:$ctx_len,
        input_types:["text"]
    }]}'
