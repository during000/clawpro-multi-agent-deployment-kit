#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 带时间戳的日志输出
log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }

# 仅对 openclaw 类型实例执行，其他类型跳过
if ! command -v openclaw >/dev/null 2>&1; then
    log "==> Skipped (non-openclaw agent type)."
    exit 0
fi

# 探测可用镜像源：优先 mirrors.tencentyun.com，回退 mirrors.tencent.com
detect_mirror() {
    local primary="https://mirrors.tencentyun.com"
    local fallback="https://mirrors.tencent.com"
    if curl -sf --connect-timeout 2 --max-time 3 "${primary}/" -o /dev/null 2>/dev/null; then
        echo "$primary"
    elif curl -sf --connect-timeout 2 --max-time 3 "${fallback}/" -o /dev/null 2>/dev/null; then
        echo "$fallback"
    else
        echo "$primary"
    fi
}

MIRROR_BASE=$(detect_mirror)
log "Using mirror: $MIRROR_BASE"

LOCKFILE="$HOME/.openclaw/openclaw.lock"
mkdir -p "$HOME/.openclaw"

(
# 先尝试非阻塞获取锁，失败则打印等待日志后再阻塞等待
if ! flock -n 200; then
    echo "[INFO] 🔒 openclaw.json 正被其他脚本修改，等待锁释放..."
    flock -x 200
    echo "[INFO] 🔓 锁已获取，继续执行"
fi

# Step 1: 卸载旧插件
log "==> Step 1: Uninstalling CLS diagnostics metrics onboard CLI..."
npx --yes --registry "${MIRROR_BASE}/npm/" clawpro-diagnostics-metrics-cls-onboard-cli uninstall --force
log "==> Step 1: Done."

# Step 2: 重新安装插件（携带 traceTopicId）
log "==> Step 2: Installing CLS diagnostics metrics onboard CLI..."
npx --yes --registry "${MIRROR_BASE}/npm/" clawpro-diagnostics-metrics-cls-onboard-cli install \
    --endpoint "{{region}}.cls.tencentyun.com" \
    --credentialMode cvmRole \
    --roleName "{{role_name}}" \
    --metricTopicId "{{metric_topic_id}}" \
    --traceTopicId "{{trace_topic_id}}" \
    --enableReport true \
    --traceEnabled true
log "==> Step 2: Done."

) 200>"$LOCKFILE"
