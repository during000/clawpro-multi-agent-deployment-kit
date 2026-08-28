#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:/usr/local/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# %INCLUDE% lib_acli_compat.sh

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
SCRIPT_NAME="set_channel_hermes"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# ===== 脚本级探测一次，后续复用 =====
_acli_mode="$(ensure_acli 2>>"$LOG_FILE")"

# ========== 工具函数 ==========
# update_env KEY VALUE FILE
#   在 .env 文件中覆盖写入 KEY=VALUE（已有 key 则替换，不重复追加）
update_env() {
    local k="$1" v="$2" f="$3"
    local escaped_v
    escaped_v="$(printf '%s' "$v" | sed 's/[\/&]/\\&/g')"
    if grep -q "^${k}=" "$f" 2>/dev/null; then
        sed -i "s/^${k}=.*/${k}=${escaped_v}/" "$f"
    else
        echo "${k}=${v}" >> "$f"
    fi
}

# restart_gateway
#   .env 写入后重启 hermes gateway，使配置生效
#   复用脚本开头的 _acli_mode；acli 存在时直接用 acli
restart_gateway() {
    echo "重启 hermes gateway..."
    if [ "$_acli_mode" = "acli" ]; then
        acli gateway restart 2>/dev/null
        echo "✓ acli gateway restart 完成"
        return
    fi
    if command -v harness >/dev/null 2>&1; then
        harness gateway restart 2>/dev/null && echo "✓ harness gateway restart 完成" && return
    fi
    for unit in hermes hermes-gateway harness-gateway; do
        if systemctl --user restart "$unit" 2>/dev/null; then
            echo "✓ systemctl --user restart $unit 完成"
            return
        fi
    done
    echo "⚠ gateway restart 失败（可能未注册为 systemd 服务），请手动重启"
}

# enable_line_platform_in_config_yaml
#   幂等地在 ~/.hermes/config.yaml 中写入：
#       gateway:
#         platforms:
#           line:
#             enabled: true
#   优先用 yq（若存在），否则用 Python（hermes 环境必有）处理。
enable_line_platform_in_config_yaml() {
    local cfg="$HOME/.hermes/config.yaml"
    mkdir -p "$(dirname "$cfg")" 2>/dev/null || true
    touch "$cfg"

    if command -v yq >/dev/null 2>&1; then
        yq -i '.gateway.platforms.line.enabled = true' "$cfg"
        echo "✓ config.yaml 已写入 gateway.platforms.line.enabled=true (yq)"
        return 0
    fi

    # Python fallback：hermes 环境必有 Python，且能可靠处理 YAML 嵌套结构
    local py
    py="$(find_hermes_python)" || {
        echo "⚠ 未找到 yq/python，跳过 config.yaml 的 gateway.platforms.line 写入"
        return 0
    }

    # 尝试用 PyYAML；若不可用则回退到纯文本处理
    "$py" - "$cfg" <<'PYEOF'
import sys, os

cfg = sys.argv[1]
try:
    import yaml
    with open(cfg, 'r') as f:
        data = yaml.safe_load(f) or {}
    gw = data.setdefault('gateway', {}) if isinstance(data, dict) else {}
    if not isinstance(gw, dict):
        gw = {}
        data['gateway'] = gw
    plats = gw.setdefault('platforms', {}) if isinstance(gw, dict) else {}
    if not isinstance(plats, dict):
        plats = {}
        gw['platforms'] = plats
    line_cfg = plats.setdefault('line', {}) if isinstance(plats, dict) else {}
    if not isinstance(line_cfg, dict):
        line_cfg = {}
        plats['line'] = line_cfg
    line_cfg['enabled'] = True
    # 保留 key 顺序：gateway 在前，platforms 在内，line 在 platforms 内
    with open(cfg, 'w') as f:
        yaml.dump(data, f, default_flow_style=False, sort_keys=False, allow_unicode=True)
    print("✓ config.yaml 已写入 gateway.platforms.line.enabled=true (PyYAML)")
except ImportError:
    # 纯文本 fallback：不依赖任何第三方库
    with open(cfg, 'r') as f:
        content = f.read()

    # 幂等：已有 line platform 块则只确保 enabled: true
    import re
    # 检查 gateway.platforms.line 是否已存在
    gw_block_re = re.compile(r'(^gateway:\s*\n)(.*?)(?=^[a-zA-Z]|\Z)', re.MULTILINE | re.DOTALL)
    m = gw_block_re.search(content)
    line_block = "    line:\n      enabled: true\n"

    if m:
        gw_body = m.group(2)
        # 检查 platforms: 是否存在
        plats_re = re.compile(r'(^  platforms:\s*\n)(.*?)(?=^  [a-zA-Z]|^[a-zA-Z]|\Z)', re.MULTILINE | re.DOTALL)
        pm = plats_re.search(gw_body)
        if pm:
            plats_body = pm.group(2)
            # 检查 line: 是否已存在
            line_re = re.compile(r'(^    line:\s*\n)(.*?)(?=^    [a-zA-Z]|^  [a-zA-Z]|^[a-zA-Z]|\Z)', re.MULTILINE | re.DOTALL)
            lm = line_re.search(plats_body)
            if lm:
                # 已有 line 块，替换其内容为 enabled: true
                new_plats_body = plats_body[:lm.start()] + line_block + plats_body[lm.end():]
            else:
                # 在 platforms: 末尾追加 line 块
                new_plats_body = plats_body.rstrip('\n') + '\n' + line_block
            new_gw_body = gw_body[:pm.start()] + pm.group(1) + new_plats_body + gw_body[pm.end():]
        else:
            # gateway 块内无 platforms:，在 gateway: 后插入
            new_gw_body = "  platforms:\n" + line_block + gw_body
        content = content[:m.start()] + m.group(1) + new_gw_body + content[m.end():]
    else:
        # 无 gateway: 块，追加到文件末尾
        if content and not content.endswith('\n'):
            content += '\n'
        content += "gateway:\n  platforms:\n" + line_block

    with open(cfg, 'w') as f:
        f.write(content)
    print("✓ config.yaml 已写入 gateway.platforms.line.enabled=true (text fallback)")
PYEOF
}

find_hermes_python() {
    if [ -x "$HOME/.hermes/hermes-agent/venv/bin/python" ]; then
        echo "$HOME/.hermes/hermes-agent/venv/bin/python"
        return 0
    fi
    if command -v python >/dev/null 2>&1; then
        command -v python
        return 0
    fi
    if command -v python3 >/dev/null 2>&1; then
        command -v python3
        return 0
    fi
    return 1
}

run_install_cmd() {
    if command -v timeout >/dev/null 2>&1; then
        timeout 300 "$@"
    else
        "$@"
    fi
}

ensure_hermes_teams_deps() {
    local py
    py="$(find_hermes_python)" || {
        echo "✗ 未找到 Hermes Python，无法安装 Microsoft Teams 依赖"
        return 1
    }

    if "$py" -c 'import aiohttp, microsoft_teams' >/dev/null 2>&1; then
        echo "✓ Hermes Microsoft Teams Python 依赖已存在"
        return 0
    fi

    echo "安装 Hermes Microsoft Teams Python 依赖..."
    echo "  Python: $py"

    # microsoft-teams-apps 早期 alpha 要求 Python >=3.12；2.0.13.4 支持 Python 3.11，
    # 兼容当前 Hermes 0.14/0.16 常见运行环境。
    local pkg="microsoft-teams-apps==2.0.13.4"

    if command -v uv >/dev/null 2>&1; then
        if UV_HTTP_TIMEOUT=120 run_install_cmd uv pip install --verbose --python "$py" -i https://mirrors.aliyun.com/pypi/simple/ "$pkg"; then
            :
        elif UV_HTTP_TIMEOUT=120 run_install_cmd uv pip install --verbose --python "$py" -i https://pypi.org/simple/ "$pkg"; then
            :
        else
            echo "✗ uv 安装 $pkg 失败"
            return 1
        fi
    elif "$py" -m pip --version >/dev/null 2>&1; then
        if run_install_cmd "$py" -m pip install --progress-bar on -i https://mirrors.aliyun.com/pypi/simple/ "$pkg"; then
            :
        elif run_install_cmd "$py" -m pip install --progress-bar on -i https://pypi.org/simple/ "$pkg"; then
            :
        else
            echo "✗ pip 安装 $pkg 失败"
            return 1
        fi
    else
        echo "✗ Hermes Python 缺少 microsoft_teams，且未找到 uv / pip，无法自动安装"
        echo "  可手动执行: uv pip install --python $py \"$pkg\""
        return 1
    fi

    if "$py" -c 'import aiohttp, microsoft_teams' >/dev/null 2>&1; then
        echo "✓ Hermes Microsoft Teams Python 依赖安装完成"
        return 0
    fi
    echo "✗ Hermes Microsoft Teams Python 依赖安装后仍无法 import microsoft_teams"
    return 1
}

ensure_hermes_dingtalk_deps() {
    local py
    py="$(find_hermes_python)" || {
        echo "✗ 未找到 Hermes Python，无法安装 dingtalk 依赖"
        return 1
    }

    if "$py" -c 'import dingtalk_stream' >/dev/null 2>&1; then
        echo "✓ dingtalk-stream 依赖已存在"
        return 0
    fi

    echo "安装 dingtalk-stream 依赖（避免 gateway 启动时自动安装导致超时）..."
    echo "  Python: $py"

    if command -v uv >/dev/null 2>&1; then
        if UV_HTTP_TIMEOUT=120 run_install_cmd uv pip install --verbose --python "$py" \
            -i https://mirrors.aliyun.com/pypi/simple/ dingtalk-stream alibabacloud-dingtalk qrcode; then
            :
        elif UV_HTTP_TIMEOUT=120 run_install_cmd uv pip install --verbose --python "$py" \
            -i https://pypi.org/simple/ dingtalk-stream alibabacloud-dingtalk qrcode; then
            :
        else
            echo "✗ dingtalk 依赖安装失败"
            return 1
        fi
    elif "$py" -m pip --version >/dev/null 2>&1; then
        if run_install_cmd "$py" -m pip install --progress-bar on \
            -i https://mirrors.aliyun.com/pypi/simple/ dingtalk-stream alibabacloud-dingtalk qrcode; then
            :
        elif run_install_cmd "$py" -m pip install --progress-bar on \
            -i https://pypi.org/simple/ dingtalk-stream alibabacloud-dingtalk qrcode; then
            :
        else
            echo "✗ dingtalk 依赖安装失败"
            return 1
        fi
    else
        echo "✗ 未找到 uv / pip，无法安装 dingtalk 依赖"
        return 1
    fi

    if "$py" -c 'import dingtalk_stream' >/dev/null 2>&1; then
        echo "✓ dingtalk-stream 依赖安装完成"
        return 0
    fi
    echo "✗ dingtalk-stream 安装后仍无法 import"
    return 1
}

ensure_hermes_slack_deps() {
    local py
    py="$(find_hermes_python)" || {
        echo "⚠ 未找到 Hermes Python，跳过 slack 依赖检查"
        return 0  # slack-sdk 通常在核心依赖中，不阻断
    }

    if "$py" -c 'import slack_sdk' >/dev/null 2>&1; then
        echo "✓ slack_sdk 依赖已存在"
        return 0
    fi

    echo "安装 slack_sdk 依赖..."
    echo "  Python: $py"

    if command -v uv >/dev/null 2>&1; then
        UV_HTTP_TIMEOUT=120 run_install_cmd uv pip install --verbose --python "$py" \
            -i https://mirrors.aliyun.com/pypi/simple/ slack-sdk || {
            echo "⚠ slack-sdk 安装失败（gateway 启动后可能会自动重试）"
            return 0  # 不阻断
        }
    elif "$py" -m pip --version >/dev/null 2>&1; then
        run_install_cmd "$py" -m pip install --progress-bar on \
            -i https://mirrors.aliyun.com/pypi/simple/ slack-sdk || {
            echo "⚠ slack-sdk 安装失败（gateway 启动后可能会自动重试）"
            return 0
        }
    fi

    echo "✓ slack_sdk 依赖安装完成"
}

# Parameters (与 openclaw set_channel.sh 契约对齐):
#   {{channel}}        - channel key（Hermes 白名单: feishu / discord / wecom / weixin / qqbot / ddingtalk / slack）
#   {{app_id}}         - feishu / qqbot 用（feishu 对应 FEISHU_APP_ID）
#   {{app_secret}}     - feishu / qqbot 用（feishu 对应 FEISHU_APP_SECRET；
#                        qqbot harness CLI 参数名是 --client-secret，脚本内部负责映射）
#   {{bot_prefix}}     - 预留（Hermes harness CLI 目前不支持，脚本忽略）
#   {{app_token}}      - slack Socket Mode 用（SLACK_APP_TOKEN）
#   {{bot_token}}      - slack Socket Mode 用（SLACK_BOT_TOKEN）
#
# Hermes 通道分派逻辑：
#   feishu    → 手动配置：写入 ~/.hermes/.env（FEISHU_APP_ID/SECRET/DOMAIN），restart gateway
#   wecom     → 手动配置：写入 ~/.hermes/.env（WECOM_BOT_ID/WECOM_SECRET），restart gateway
#   slack     → 手动配置：写入 ~/.hermes/.env（SLACK_APP_TOKEN/SLACK_BOT_TOKEN），restart gateway
#   discord   → 手动配置：写入 ~/.hermes/.env（DISCORD_BOT_TOKEN/DISCORD_ALLOWED_USERS），restart gateway
#   ddingtalk → harness configure gateway dingtalk --client-id --client-secret
#   weixin    → 扫码流程：harness gateway qr-url-fast --platform weixin
#   qqbot     → harness configure gateway qqbot --app-id --client-secret
#   msteams   → 写入 ~/.hermes/.env（TEAMS_CLIENT_ID/SECRET/TENANT_ID/PORT/ALLOW_ALL_USERS），restart gateway
#
# 与 openclaw set_channel.sh 的差异：
#   - feishu/wecom 手动配置写 ~/.hermes/.env 而非 openclaw.json
#   - weixin 扫码走 SSE 流式，stdout 透传给 hatchery 后端转发前端
#   - 自定义通道（is_custom=true）Hermes 不支持，直接 fail

CHANNEL="{{channel}}"
# 前端契约使用 "openclaw-weixin"，脚本内部（harness CLI）使用 "weixin"，在入口处统一转换
[ "$CHANNEL" = "openclaw-weixin" ] && CHANNEL="weixin"
APP_ID="{{app_id}}"
APP_SECRET="{{app_secret}}"
# wecom 手动配置使用不同字段名（与前端 CHANNEL_CONFIG_MAP.wecom.fields 对齐）
BOT_ID="{{bot_id}}"
BOT_SECRET="{{secret}}"
# ddingtalk 使用 client_id / client_secret（与前端 CHANNEL_CONFIG_MAP.ddingtalk.fields 对齐）
CLIENT_ID="{{client_id}}"
CLIENT_SECRET="{{client_secret}}"
# slack 使用 Socket Mode 凭据
SLACK_APP_TOKEN="{{app_token}}"
SLACK_BOT_TOKEN="{{bot_token}}"
# msteams 使用 Azure Bot Framework 应用凭据
TENANT_ID="{{tenant_id}}"
WEBHOOK_PORT="{{webhook_port}}"
# discord 使用 bot token 和 user id
DISCORD_BOT_TOKEN="{{bot_token}}"
DISCORD_USER_ID="{{user_id}}"
# line 使用 LINE Messaging API 凭据
LINE_CHANNEL_TOKEN="{{channel_access_token}}"
LINE_CHANNEL_SECRET="{{channel_secret}}"

echo "=== Hermes 配置通道: $CHANNEL ==="

# ===== 白名单校验（双保险：Go 层已过滤，这里兜底）=====
case "$CHANNEL" in
    feishu|discord|wecom|weixin|qqbot|ddingtalk|slack|msteams|lark|line)
        ;;
    *)
        echo "✗ Hermes 不支持的通道类型: $CHANNEL"
        echo "   支持列表: feishu wecom weixin qqbot ddingtalk slack msteams lark discord line"
        exit 1
        ;;
esac

_GLOBAL_ENV_FILE="$HOME/.hermes/.env"
mkdir -p "$(dirname "$_GLOBAL_ENV_FILE")"
touch "$_GLOBAL_ENV_FILE"

# ===== 按 channel 分派 =====
case "$CHANNEL" in
    feishu)
        if [ -z "$APP_ID" ] || [ -z "$APP_SECRET" ]; then
            echo "✗ feishu 配置缺失 app_id 或 app_secret"
            exit 1
        fi

        update_env FEISHU_APP_ID     "$APP_ID"     "$_GLOBAL_ENV_FILE"
        update_env FEISHU_APP_SECRET "$APP_SECRET" "$_GLOBAL_ENV_FILE"
        update_env FEISHU_DOMAIN     "feishu"      "$_GLOBAL_ENV_FILE"
        # gateway/run.py: 无 allowlist 且无 GATEWAY_ALLOW_ALL_USERS 时默认拒绝
        # 用平台级 allow-all 代替全局开关，避免污染其他通道
        update_env FEISHU_ALLOWED_USERS   "*"    "$_GLOBAL_ENV_FILE"
        update_env FEISHU_ALLOW_ALL_USERS "true" "$_GLOBAL_ENV_FILE"

        echo "✓ 飞书通道配置已写入 $_GLOBAL_ENV_FILE"
        echo "  FEISHU_APP_ID=$APP_ID"
        echo "  FEISHU_DOMAIN=feishu"
        echo "  FEISHU_ALLOW_ALL_USERS=true"
        restart_gateway
        ;;
    
    lark)
        if [ -z "$APP_ID" ] || [ -z "$APP_SECRET" ]; then
            echo "✗ lark require app_id or app_secret"
            exit 1
        fi

        update_env FEISHU_APP_ID     "$APP_ID"     "$_GLOBAL_ENV_FILE"
        update_env FEISHU_APP_SECRET "$APP_SECRET" "$_GLOBAL_ENV_FILE"
        update_env FEISHU_DOMAIN     "lark"      "$_GLOBAL_ENV_FILE"
        update_env FEISHU_ALLOWED_USERS   "*"    "$_GLOBAL_ENV_FILE"
        update_env FEISHU_ALLOW_ALL_USERS "true" "$_GLOBAL_ENV_FILE"

        echo "✓ Lark channel config has been written to $_GLOBAL_ENV_FILE"
        restart_gateway
        ;;

    wecom)
        if [ -z "$BOT_ID" ] || [ -z "$BOT_SECRET" ]; then
            echo "✗ wecom 配置缺失 bot_id 或 secret"
            exit 1
        fi

        update_env WECOM_BOT_ID    "$BOT_ID"     "$_GLOBAL_ENV_FILE"
        update_env WECOM_SECRET    "$BOT_SECRET" "$_GLOBAL_ENV_FILE"
        update_env WECOM_DM_POLICY "open"        "$_GLOBAL_ENV_FILE"
        # gateway/run.py: 无 allowlist 时默认拒绝，用平台级 * 代替全局开关
        update_env WECOM_ALLOWED_USERS   "*"    "$_GLOBAL_ENV_FILE"
        update_env WECOM_ALLOW_ALL_USERS "true" "$_GLOBAL_ENV_FILE"

        echo "✓ 企业微信通道配置已写入 $_GLOBAL_ENV_FILE"
        echo "  WECOM_BOT_ID=$BOT_ID"
        echo "  WECOM_DM_POLICY=open"
        echo "  WECOM_ALLOW_ALL_USERS=true"
        restart_gateway
        ;;

    slack)
        if [ -z "$SLACK_APP_TOKEN" ] || [ -z "$SLACK_BOT_TOKEN" ]; then
            echo "✗ slack 配置缺失 app_token 或 bot_token"
            exit 1
        fi

        ensure_hermes_slack_deps

        update_env SLACK_APP_TOKEN "$SLACK_APP_TOKEN" "$_GLOBAL_ENV_FILE"
        update_env SLACK_BOT_TOKEN "$SLACK_BOT_TOKEN" "$_GLOBAL_ENV_FILE"
        update_env SLACK_ALLOWED_USERS   "*"    "$_GLOBAL_ENV_FILE"
        update_env SLACK_ALLOW_ALL_USERS "true" "$_GLOBAL_ENV_FILE"

        echo "✓ Slack 通道配置已写入 $_GLOBAL_ENV_FILE"
        echo "  SLACK_ALLOWED_USERS=*"
        restart_gateway
        ;;

    discord)
        if [ -z "$DISCORD_BOT_TOKEN" ] || [ -z "$DISCORD_USER_ID" ]; then
            echo "✗ discord configuration missing bot_token or user_id"
            exit 1
        fi

        update_env DISCORD_BOT_TOKEN "$DISCORD_BOT_TOKEN" "$_GLOBAL_ENV_FILE"
        # Hermes Discord 不支持通配符匹配，所以必须指定用户 ID
        update_env DISCORD_ALLOWED_USERS "$DISCORD_USER_ID" "$_GLOBAL_ENV_FILE"

        echo "✓ Discord 通道配置已写入 $_GLOBAL_ENV_FILE"
        restart_gateway
        ;;

    ddingtalk)
        echo "配置 ddingtalk..."
        if [ -z "$CLIENT_ID" ] || [ -z "$CLIENT_SECRET" ]; then
            echo "✗ ddingtalk 配置缺失 client_id 或 client_secret"
            exit 1
        fi

        ensure_hermes_dingtalk_deps

        if [ "$_acli_mode" = "acli" ]; then
            acli channel connect \
                --platform dingtalk \
                --app-id "$CLIENT_ID" \
                --app-secret "$CLIENT_SECRET" \
                --allowed-users "*"
            echo "✓ ddingtalk 配置完成 (acli)"
        else
            ensure_harness_cli || exit 1
            command -v harness >/dev/null 2>&1 || { echo "✗ harness CLI 不存在"; exit 1; }
            harness configure gateway dingtalk \
                --client-id     "$CLIENT_ID" \
                --client-secret "$CLIENT_SECRET" \
                --allowed-users "*"
            echo "✓ ddingtalk 配置完成"
        fi
        ;;

    weixin)
        echo "启动微信扫码流程（stdout 直接输出 QR URL，hatchery SSE 透传前端）..."
        if [ "$_acli_mode" = "acli" ]; then
            exec acli gateway qr-url-fast --platform weixin --timeout 10m
        else
            ensure_harness_cli || exit 1
            command -v harness >/dev/null 2>&1 || { echo "✗ harness CLI 不存在"; exit 1; }
            exec harness gateway qr-url-fast --platform weixin --timeout 10m
        fi
        ;;

    msteams)
        echo "配置 Microsoft Teams..."
        if [ -z "$APP_ID" ] || [ -z "$APP_SECRET" ] || [ -z "$TENANT_ID" ]; then
            echo "✗ msteams 配置缺失 app_id、app_secret 或 tenant_id"
            exit 1
        fi
        if [ -z "$WEBHOOK_PORT" ] || [ "$WEBHOOK_PORT" = "<no value>" ]; then
            WEBHOOK_PORT="3978"
        fi

        ensure_hermes_teams_deps

        update_env TEAMS_CLIENT_ID        "$APP_ID"        "$_GLOBAL_ENV_FILE"
        update_env TEAMS_CLIENT_SECRET    "$APP_SECRET"    "$_GLOBAL_ENV_FILE"
        update_env TEAMS_TENANT_ID        "$TENANT_ID"     "$_GLOBAL_ENV_FILE"
        update_env TEAMS_PORT             "$WEBHOOK_PORT"  "$_GLOBAL_ENV_FILE"
        update_env TEAMS_ALLOW_ALL_USERS  "true"           "$_GLOBAL_ENV_FILE"

        echo "✓ Microsoft Teams 通道配置已写入 $_GLOBAL_ENV_FILE"
        echo "  TEAMS_CLIENT_ID=$APP_ID"
        echo "  TEAMS_TENANT_ID=$TENANT_ID"
        echo "  TEAMS_PORT=$WEBHOOK_PORT"
        echo "  Endpoint: {{teams_endpoint}}"
        restart_gateway
        ;;

    line)
        echo "配置 LINE..."
        if [ -z "$LINE_CHANNEL_TOKEN" ] || [ "$LINE_CHANNEL_TOKEN" = "<no value>" ]; then
            echo "✗ line 配置缺失 channel_token"
            exit 1
        fi
        if [ -z "$LINE_CHANNEL_SECRET" ] || [ "$LINE_CHANNEL_SECRET" = "<no value>" ]; then
            echo "✗ line 配置缺失 channel_secret"
            exit 1
        fi
        if [ -z "$WEBHOOK_PORT" ] || [ "$WEBHOOK_PORT" = "<no value>" ]; then
            WEBHOOK_PORT="8646"
        fi

        update_env LINE_CHANNEL_ACCESS_TOKEN  "$LINE_CHANNEL_TOKEN"   "$_GLOBAL_ENV_FILE"
        update_env LINE_CHANNEL_SECRET        "$LINE_CHANNEL_SECRET"  "$_GLOBAL_ENV_FILE"
        update_env LINE_PORT                  "$WEBHOOK_PORT"         "$_GLOBAL_ENV_FILE"
        update_env LINE_ALLOW_ALL_USERS       "true"                  "$_GLOBAL_ENV_FILE"

        # 在 ~/.hermes/config.yaml 中启用 gateway.platforms.line（幂等）
        enable_line_platform_in_config_yaml

        echo "✓ LINE 通道配置已写入 $_GLOBAL_ENV_FILE"
        echo "  LINE_PORT=$WEBHOOK_PORT"
        echo "  Endpoint: {{proxy_endpoint}}"
        restart_gateway
        ;;

    qqbot)
        echo "配置 qqbot..."
        if [ -z "$APP_ID" ] || [ -z "$APP_SECRET" ]; then
            echo "✗ qqbot 配置缺失 app_id 或 app_secret"
            exit 1
        fi
        if [ "$_acli_mode" = "acli" ]; then
            acli configure gateway qqbot \
                --app-id        "$APP_ID" \
                --client-secret "$APP_SECRET"
            echo "✓ qqbot 配置完成 (acli 内部已重启 gateway)"
        else
            ensure_harness_cli || exit 1
            command -v harness >/dev/null 2>&1 || { echo "✗ harness CLI 不存在"; exit 1; }
            harness configure gateway qqbot \
                --app-id        "$APP_ID" \
                --client-secret "$APP_SECRET"
            echo "✓ qqbot 配置完成（harness 内部已重启 gateway）"
        fi
        ;;
esac

echo ""
echo "=== 通道 $CHANNEL 配置完成 ==="
