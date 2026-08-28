#!/bin/bash
set -euo pipefail

# %INCLUDE% lib_acli_compat.sh

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
SCRIPT_NAME="del_channel_hermes"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

# ===== 脚本级探测一次，后续复用 =====
_acli_mode="$(ensure_acli 2>>"$LOG_FILE")"

# delete_env KEY FILE
delete_env() {
    local k="$1" f="$2"
    if [ -f "$f" ]; then
        sed -i "/^${k}=/d" "$f"
    fi
}

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

# disable_line_platform_in_config_yaml
#   从 ~/.hermes/config.yaml 中删除 gateway.platforms.line 块（幂等，不存在则跳过）。
#   优先用 yq（若存在），否则用 Python（hermes 环境必有）处理。
disable_line_platform_in_config_yaml() {
    local cfg="$HOME/.hermes/config.yaml"
    [ -f "$cfg" ] || return 0

    if command -v yq >/dev/null 2>&1; then
        if yq -e '.gateway.platforms.line' "$cfg" >/dev/null 2>&1; then
            yq -i 'del(.gateway.platforms.line)' "$cfg"
            echo "✓ config.yaml 中 line platform 已删除 (yq)"
        fi
        return 0
    fi

    local py
    py="$(find_hermes_python)" || {
        echo "⚠ 未找到 yq/python，跳过 config.yaml 的 line platform 删除"
        return 0
    }

    "$py" - "$cfg" <<'PYEOF'
import sys, re

cfg = sys.argv[1]
try:
    import yaml
    with open(cfg, 'r') as f:
        data = yaml.safe_load(f) or {}
    changed = False
    if isinstance(data, dict) and isinstance(data.get('gateway'), dict):
        gw = data['gateway']
        if isinstance(gw.get('platforms'), dict):
            plats = gw['platforms']
            if 'line' in plats:
                del plats['line']
                changed = True
    if changed:
        with open(cfg, 'w') as f:
            yaml.dump(data, f, default_flow_style=False, sort_keys=False, allow_unicode=True)
        print("✓ config.yaml 中 line platform 已删除 (PyYAML)")
except ImportError:
    # 纯文本 fallback
    with open(cfg, 'r') as f:
        content = f.read()
    original = content
    # 在 gateway: 块内定位 platforms: 块，删除其中的 line: 子块
    gw_block_re = re.compile(r'(^gateway:\s*\n)(.*?)(?=^[a-zA-Z]|\Z)', re.MULTILINE | re.DOTALL)
    m = gw_block_re.search(content)
    if m:
        gw_body = m.group(2)
        plats_re = re.compile(r'(^  platforms:\s*\n)(.*?)(?=^  [a-zA-Z]|^[a-zA-Z]|\Z)', re.MULTILINE | re.DOTALL)
        pm = plats_re.search(gw_body)
        if pm:
            plats_body = pm.group(2)
            line_re = re.compile(r'^    line:\s*\n(?:^      [^\n]*\n)*', re.MULTILINE)
            new_plats_body = line_re.sub('', plats_body, count=1)
            if new_plats_body != plats_body:
                new_gw_body = gw_body[:pm.start()] + pm.group(1) + new_plats_body + gw_body[pm.end():]
                content = content[:m.start()] + m.group(1) + new_gw_body + content[m.end():]
    if content != original:
        with open(cfg, 'w') as f:
            f.write(content)
        print("✓ config.yaml 中 line platform 已删除 (text fallback)")
PYEOF
}

# Parameters (与 openclaw del_channel.sh 契约对齐):
#   {{channel}} - channel key（Hermes 白名单: feishu / weixin / qqbot / slack）
#
# 与 openclaw del_channel.sh 的差异：
#   - 不操作 ~/.openclaw/openclaw.json，直接调 `harness channel delete --platform KEY`
#   - harness 内部语义是"删除配置源"（而非写 synthetic enabled=false）
#   - 若配置真的变动，harness 内部自动重启 gateway
#   - 无 plugins disable 概念
#
# harness channel delete 输出 JSON:
#   {"platform":"feishu","changed":true,"restarted":true}

CHANNEL="{{channel}}"
# 前端契约使用 "openclaw-weixin"，脚本内部（harness CLI）使用 "weixin"，在入口处统一转换
[ "$CHANNEL" = "openclaw-weixin" ] && CHANNEL="weixin"
# 前端契约使用 "ddingtalk"，harness CLI 使用 "dingtalk"，在入口处统一转换
[ "$CHANNEL" = "ddingtalk" ] && CHANNEL="dingtalk"

echo "=== Hermes 删除通道: $CHANNEL ==="

if [ "$CHANNEL" = "slack" ]; then
    ENV_FILE="$HOME/.hermes/.env"
    delete_env SLACK_APP_TOKEN "$ENV_FILE"
    delete_env SLACK_BOT_TOKEN "$ENV_FILE"
    delete_env SLACK_ALLOWED_USERS "$ENV_FILE"
    delete_env SLACK_ALLOW_ALL_USERS "$ENV_FILE"
    echo "✓ Slack 通道配置已从 $ENV_FILE 删除"
    restart_gateway
    echo ""
    echo "=== 通道 $CHANNEL 删除完成 ==="
    exit 0
fi

if [ "$CHANNEL" = "discord" ]; then
    ENV_FILE="$HOME/.hermes/.env"
    delete_env DISCORD_BOT_TOKEN "$ENV_FILE"
    delete_env DISCORD_ALLOWED_USERS "$ENV_FILE"
    echo "✓ Discord 通道配置已从 $ENV_FILE 删除"
    restart_gateway
    echo ""
    echo "=== 通道 $CHANNEL 删除完成 ==="
    exit 0
fi

if [ "$CHANNEL" = "lark" ]; then
    ENV_FILE="$HOME/.hermes/.env"
    delete_env FEISHU_APP_ID    "$ENV_FILE"
    delete_env FEISHU_APP_SECRET "$ENV_FILE"
    delete_env FEISHU_DOMAIN     "$ENV_FILE"
    delete_env FEISHU_ALLOWED_USERS   "$ENV_FILE"
    delete_env FEISHU_ALLOW_ALL_USERS "$ENV_FILE"
    restart_gateway
    echo ""
    echo "=== 通道 $CHANNEL 删除完成 ==="
    exit 0
fi

if [ "$CHANNEL" = "msteams" ]; then
    ENV_FILE="$HOME/.hermes/.env"
    delete_env TEAMS_CLIENT_ID "$ENV_FILE"
    delete_env TEAMS_CLIENT_SECRET "$ENV_FILE"
    delete_env TEAMS_TENANT_ID "$ENV_FILE"
    delete_env TEAMS_PORT "$ENV_FILE"
    delete_env TEAMS_ALLOW_ALL_USERS "$ENV_FILE"
    echo "✓ Microsoft Teams 配置已从 $ENV_FILE 删除"
    restart_gateway
    echo ""
    echo "=== 通道 $CHANNEL 删除完成 ==="
    exit 0
fi

if [ "$CHANNEL" = "line" ]; then
    ENV_FILE="$HOME/.hermes/.env"
    delete_env LINE_CHANNEL_ACCESS_TOKEN "$ENV_FILE"
    delete_env LINE_CHANNEL_SECRET "$ENV_FILE"
    delete_env LINE_PORT "$ENV_FILE"
    delete_env LINE_ALLOW_ALL_USERS "$ENV_FILE"
    # 从 ~/.hermes/config.yaml 删除 gateway.platforms.line 块
    disable_line_platform_in_config_yaml
    echo "✓ LINE 配置已从 $ENV_FILE 删除"
    restart_gateway
    echo ""
    echo "=== 通道 $CHANNEL 删除完成 ==="
    exit 0
fi

if [ "$_acli_mode" = "acli" ]; then
    echo "执行: acli channel delete --platform $CHANNEL"
    acli channel delete --platform "$CHANNEL"
    echo "✓ 通道 $CHANNEL 删除完成 (acli)"
else
    # ===== fallback: harness 路径（仅 acli 不可用时） =====
    ensure_harness_cli || exit 1
    command -v harness >/dev/null 2>&1 || { echo "✗ harness CLI 不存在"; exit 1; }

    # ===== 白名单兜底 =====
    case "$CHANNEL" in
        feishu|weixin|qqbot|msteams|line)
            ;;
        *)
            echo "⚠ Hermes 不在白名单的通道: $CHANNEL（仍透传 harness CLI 尝试删除）"
            ;;
    esac

    echo "执行: harness channel delete --platform $CHANNEL"
    harness channel delete --platform "$CHANNEL"
    echo "✓ 通道 $CHANNEL 删除完成"
fi

echo ""
echo "=== 通道 $CHANNEL 删除完成 ==="
