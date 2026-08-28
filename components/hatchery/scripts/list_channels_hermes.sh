#!/bin/bash
#
# list_channels_hermes.sh
#
# 作用：列出 Hermes 实例当前启用/配置的通道。
# 契约：stdout 末行输出 JSON object，形如：
#   { "feishu": {"enabled": true, ...}, "weixin": {"enabled": false, ...} }
# 与 scripts/list_channels.sh（openclaw）保持同一形状，便于 controller 统一解析。
#
# 实现：优先调用 `harness channel list`（返回 {channels: [{key,label,effective,config_enabled,raw_config}...]}），
# 并把每个 entry 归一化为 {enabled: effective, config: raw_config}。
# fallback：harness 不存在/异常 → 输出 "{}"。
#
set -uo pipefail

export PATH="$HOME/.local/bin:$HOME/.cargo/bin:$HOME/.npm-global/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR="/run/user/$(id -u 2>/dev/null || echo 0)"

LOG_DIR="$HOME/.hermes/logs"
LOG_FILE="$LOG_DIR/list_channels_hermes.log"
mkdir -p "$LOG_DIR" 2>/dev/null || true

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> "$LOG_FILE" 2>/dev/null || true; }

# ========== 拉取/更新 harness CLI ==========
ensure_harness_cli() {
    local HARNESS_URL="https://agentone-1388575217.cos.ap-guangzhou.myqcloud.com/harness-linux-amd64"
    local HARNESS_BIN="$HOME/.local/bin/harness"
    mkdir -p "$(dirname "$HARNESS_BIN")" 2>/dev/null || true
    log "拉取 harness CLI: $HARNESS_URL"
    if curl -fsSL --connect-timeout 10 --max-time 60 --retry 2 --retry-delay 2 \
        "$HARNESS_URL" -o "${HARNESS_BIN}.tmp" 2>> "$LOG_FILE"; then
        chmod +x "${HARNESS_BIN}.tmp"
        mv -f "${HARNESS_BIN}.tmp" "$HARNESS_BIN"
        log "harness CLI 已更新: $HARNESS_BIN"
    else
        rm -f "${HARNESS_BIN}.tmp" 2>/dev/null || true
        if command -v harness >/dev/null 2>&1; then
            log "harness CLI 下载失败，沿用已有版本: $(command -v harness)"
        else
            log "harness CLI 下载失败且本地无已有版本"
            return 1
        fi
    fi
}

# 1) 依赖检查
ensure_harness_cli || true
if ! command -v harness >/dev/null 2>&1; then
    log "harness CLI not found, fallback to {}"
    echo '{}'
    exit 0
fi
if ! command -v jq >/dev/null 2>&1; then
    log "jq not found, fallback to {}"
    echo '{}'
    exit 0
fi

# 2) 调用 harness channel list，捕获输出
RAW="$(harness channel list 2>> "$LOG_FILE" || true)"
if [ -z "$RAW" ]; then
    log "harness channel list returned empty"
    RAW='{"channels":[]}'
fi

# 3) 归一化为 {key: {enabled, label, config}} 形状
#    harness 输出：{"channels":[{"key":"feishu","label":"...","effective":true,"config_enabled":false,"raw_config":{...}, ...}, ...]}
NORMALIZED="$(
    echo "$RAW" | jq -c '
      .channels // []
      | map({
          key: .key,
          value: {
            enabled: (.effective // false),
            config_enabled: (.config_enabled // false),
            label: (.label // ""),
            env_keys_present: (.env_keys_present // []),
            config: (.raw_config // {})
          }
        })
      | from_entries
    ' 2>> "$LOG_FILE"
)"

if [ -z "$NORMALIZED" ] || [ "$NORMALIZED" = "null" ]; then
    log "normalize failed"
    echo '{}'
    exit 0
fi

# 4) 修补：harness channel list 的 wecom 条目只检测 env key 是否存在，不读取值，
#    导致 bot 为空（null 或 {}）。前端 appliedChannels 从 wecomDetail.bot.botId / .bot.secret 取值，
#    空 bot 会导致 fieldValues 为空，企微通道不显示。
#    修补条件：wecom 不存在，或 wecom.bot 为 null/空对象
ENV_FILE="$HOME/.hermes/.env"
_needs_wecom_patch() {
    if ! echo "$NORMALIZED" | jq -e '.wecom' > /dev/null 2>&1; then
        return 0  # wecom 不存在
    fi
    # bot 为 null、不存在、或空对象 {} 时均需补丁
    if echo "$NORMALIZED" | jq -e '.wecom.bot == null or .wecom.bot == {} or (.wecom | has("bot") | not)' > /dev/null 2>&1; then
        return 0
    fi
    return 1  # wecom.bot 已有值，无需补丁
}
if _needs_wecom_patch; then
    WECOM_BOT_ID=""
    WECOM_SECRET=""
    if [ -f "$ENV_FILE" ]; then
        WECOM_BOT_ID="$(grep '^WECOM_BOT_ID=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
        WECOM_SECRET="$(grep '^WECOM_SECRET=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
    fi
    if [ -n "$WECOM_BOT_ID" ] && [ -n "$WECOM_SECRET" ]; then
        WECOM_ENABLED=true
    else
        WECOM_ENABLED=false
    fi
    NORMALIZED="$(echo "$NORMALIZED" | jq -c \
        --argjson enabled "$WECOM_ENABLED" \
        --arg bot_id "$WECOM_BOT_ID" \
        --arg secret "$WECOM_SECRET" \
        'if .wecom then
            .wecom.bot = (if $enabled then {"botId": $bot_id, "secret": $secret} else {} end) |
            .wecom.enabled = $enabled
        else
            . + {"wecom": {
                "enabled": $enabled,
                "config_enabled": false,
                "label": "WeCom",
                "env_keys_present": (if $enabled then ["WECOM_BOT_ID","WECOM_SECRET"] else [] end),
                "config": {},
                "bot": (if $enabled then {"botId": $bot_id, "secret": $secret} else {} end)
            }}
        end'
    )"
    log "wecom bot 补丁已注入 (enabled=$WECOM_ENABLED)"
fi

# 5) 修补 slack：从 .env 兜底补齐 Hermes Slack 状态。
SLACK_APP_TOKEN=""
SLACK_BOT_TOKEN=""
if [ -f "$ENV_FILE" ]; then
    SLACK_APP_TOKEN="$(grep '^SLACK_APP_TOKEN=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
    SLACK_BOT_TOKEN="$(grep '^SLACK_BOT_TOKEN=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
fi
SLACK_ENABLED=false
if [ -n "$SLACK_APP_TOKEN" ] && [ -n "$SLACK_BOT_TOKEN" ]; then
    SLACK_ENABLED=true
fi
NORMALIZED="$(echo "$NORMALIZED" | jq -c \
    --argjson enabled "$SLACK_ENABLED" \
    --arg app_token "$SLACK_APP_TOKEN" \
    --arg bot_token "$SLACK_BOT_TOKEN" \
    '.slack = ((.slack // {}) + {
        "enabled": $enabled,
        "config_enabled": $enabled,
        "label": "Slack",
        "env_keys_present": (if $enabled then ["SLACK_APP_TOKEN","SLACK_BOT_TOKEN"] else [] end),
        "config": (if $enabled then {"mode": "socket", "app_token": $app_token, "bot_token": $bot_token} else {} end),
        "mode": "socket",
        "app_token": $app_token,
        "bot_token": $bot_token
    })'
)"
log "slack env 补丁已注入 (enabled=$SLACK_ENABLED)"

# 6) 修补 discord：从 .env 兜底补齐 Hermes Discord 状态。
DISCORD_BOT_TOKEN=""
DISCORD_USER_ID=""
if [ -f "$ENV_FILE" ]; then
    DISCORD_BOT_TOKEN="$(grep '^DISCORD_BOT_TOKEN=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
    DISCORD_USER_ID="$(grep '^DISCORD_ALLOWED_USERS=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
fi
DISCORD_ENABLED=false
if [ -n "$DISCORD_BOT_TOKEN" ] && [ -n "$DISCORD_USER_ID" ]; then
    DISCORD_ENABLED=true
fi
NORMALIZED="$(echo "$NORMALIZED" | jq -c \
    --argjson enabled "$DISCORD_ENABLED" \
    --arg bot_token "$DISCORD_BOT_TOKEN" \
    --arg user_id "$DISCORD_USER_ID" \
    '.discord = ((.discord // {}) + {
        "enabled": $enabled,
        "config_enabled": $enabled,
        "label": "Discord",
        "env_keys_present": (if $enabled then ["DISCORD_BOT_TOKEN","DISCORD_ALLOWED_USERS"] else [] end),
        "config": (if $enabled then {"bot_token": $bot_token, "user_id": $user_id} else {} end),
        "bot_token": $bot_token,
        "user_id": $user_id
    })'
)"
log "discord env 补丁已注入 (enabled=$DISCORD_ENABLED)"

# 7) 修补 lark：从 .env 兜底补齐 Hermes Lark 状态。
#    Lark 复用 feishu 配置字段，通过 FEISHU_DOMAIN=lark 区分。
LARK_APP_ID=""
LARK_APP_SECRET=""
FEISHU_DOMAIN=""
if [ -f "$ENV_FILE" ]; then
    LARK_APP_ID="$(grep '^FEISHU_APP_ID=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
    LARK_APP_SECRET="$(grep '^FEISHU_APP_SECRET=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
    FEISHU_DOMAIN="$(grep '^FEISHU_DOMAIN=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
fi
if [ -n "$LARK_APP_ID" ] && [ -n "$LARK_APP_SECRET" ] && [ "$FEISHU_DOMAIN" = "lark" ]; then
    NORMALIZED="$(echo "$NORMALIZED" | jq -c \
        --argjson enabled "true" \
        --arg app_id "$LARK_APP_ID" \
        --arg app_secret "$LARK_APP_SECRET" \
        '.feishu = ((.feishu // {}) + {
            "enabled": $enabled,
            "config_enabled": $enabled,
            "label": "Lark",
            "env_keys_present": (if $enabled then ["FEISHU_APP_ID","FEISHU_APP_SECRET","FEISHU_DOMAIN"] else [] end),
            "config": (if $enabled then {"app_id": $app_id, "app_secret": $app_secret, "domain": "lark"} else {} end),
            "app_id": $app_id,
            "app_secret": $app_secret
        })'
    )"
fi


# 归一化 weixin → openclaw-weixin（前端契约统一使用 openclaw-weixin）
NORMALIZED="$(echo "$NORMALIZED" | jq -c 'if has("weixin") then .["openclaw-weixin"] = .weixin | del(.weixin) else . end' 2>/dev/null || echo "$NORMALIZED")"

# 归一化 dingtalk → ddingtalk（harness 使用 dingtalk，前端契约统一使用 ddingtalk）
NORMALIZED="$(echo "$NORMALIZED" | jq -c 'if has("dingtalk") then .ddingtalk = .dingtalk | del(.dingtalk) else . end' 2>/dev/null || echo "$NORMALIZED")"

# 修补 msteams：部分 Hermes/harness 版本尚未在 channel list 中暴露 Teams，
# 直接从 ~/.hermes/.env 检测 TEAMS_* 配置并合并到结果中。
if [ -f "$ENV_FILE" ]; then
    TEAMS_CLIENT_ID="$(grep '^TEAMS_CLIENT_ID=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
    TEAMS_TENANT_ID="$(grep '^TEAMS_TENANT_ID=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
    TEAMS_PORT="$(grep '^TEAMS_PORT=' "$ENV_FILE" | cut -d= -f2- | tr -d '[:space:]')"
    if [ -n "$TEAMS_CLIENT_ID" ] && [ -n "$TEAMS_TENANT_ID" ]; then
        [ -z "$TEAMS_PORT" ] && TEAMS_PORT="3978"
        NORMALIZED="$(echo "$NORMALIZED" | jq -c \
            --arg client_id "$TEAMS_CLIENT_ID" \
            --arg tenant_id "$TEAMS_TENANT_ID" \
            --arg port "$TEAMS_PORT" \
            '.msteams = {
                "enabled": true,
                "config_enabled": true,
                "label": "Microsoft Teams",
                "env_keys_present": ["TEAMS_CLIENT_ID", "TEAMS_CLIENT_SECRET", "TEAMS_TENANT_ID", "TEAMS_PORT", "TEAMS_ALLOW_ALL_USERS"],
                "config": {"client_id": $client_id, "tenant_id": $tenant_id, "port": $port}
            }' 2>/dev/null || echo "$NORMALIZED")"
        log "msteams env 补丁已注入"
    fi
fi

# 过滤掉内置 api_server（非用户可操作通道，部分 hermes 版本会自动启用）
NORMALIZED="$(echo "$NORMALIZED" | jq -c 'del(.api_server)' 2>/dev/null || echo "$NORMALIZED")"

echo "$NORMALIZED"
