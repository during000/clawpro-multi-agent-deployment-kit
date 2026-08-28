#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化 ==========
LOG_DIR="${HOME}/.openclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
SCRIPT_NAME="list_channels"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> "$LOG_FILE" 2>/dev/null || true; }

log ""
log "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
log "=== 列出通道配置 ==="

# 读取 openclaw.json 中的 channels
if [ ! -f "$HOME/.openclaw/openclaw.json" ]; then
    log "openclaw.json 不存在，返回空对象"
    echo '{}'
    exit 0
fi
log "读取 openclaw.json 中的 channels..."
if ! CHANNELS=$(jq '.channels // {}' "$HOME/.openclaw/openclaw.json" 2>/tmp/jq_err.txt); then
    JQ_ERR=$(cat /tmp/jq_err.txt 2>/dev/null || true)
    log "✗ openclaw.json 解析失败: $JQ_ERR"
    echo "openclaw.json 解析失败"
    exit 1
fi
log "✓ channels 读取完成"

# 检测微信通道：如果 accounts.json 存在且有已接入的账号，将 openclaw-weixin 合并到结果中
WEIXIN_ACCOUNTS="$HOME/.openclaw/openclaw-weixin/accounts.json"
if [ -f "$WEIXIN_ACCOUNTS" ]; then
    ACCOUNT_COUNT=$(jq 'length' "$WEIXIN_ACCOUNTS" 2>/dev/null || echo 0)
    log "检测到微信账号文件，账号数量: $ACCOUNT_COUNT"
    if [ "$ACCOUNT_COUNT" -gt 0 ]; then
        # 读取每个账号的详细信息
        ACCOUNTS_DIR="$HOME/.openclaw/openclaw-weixin/accounts"
        ACCOUNT_LIST=$(jq -r '.[]' "$WEIXIN_ACCOUNTS" 2>/dev/null || true)
        DETAILS="[]"
        for ACCT in $ACCOUNT_LIST; do
            ACCT_FILE="$ACCOUNTS_DIR/${ACCT}.json"
            if [ -f "$ACCT_FILE" ]; then
                DETAILS=$(echo "$DETAILS" | jq --arg name "$ACCT" --slurpfile info "$ACCT_FILE" '. + [{"name": $name} + $info[0]]')
                log "✓ 读取微信账号详情: $ACCT"
            fi
        done
        CHANNELS=$(echo "$CHANNELS" | jq --argjson details "$DETAILS" '.["openclaw-weixin"] = {"enabled": true, "accounts": $details}')
        log "✓ openclaw-weixin 通道已合并到结果中"
    fi
else
    log "未检测到微信账号文件，跳过 openclaw-weixin 通道"
fi

log "=== 通道列表获取完成 ==="
echo "$CHANNELS"