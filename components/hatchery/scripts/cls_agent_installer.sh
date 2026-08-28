#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 带时间戳的日志输出
log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }

# Pre-check: skip if loglistener is already installed and running
if systemctl is-active --quiet loglistener 2>/dev/null; then
    log "loglistener service is already running, skipping installation."
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

# Step 1: Download and install loglistener_operator
log "==> Step 1: Downloading loglistener_operator..."
rm -rf ./loglistener_operator && wget "${MIRROR_BASE}/install/cls/script/loglistener/loglistener_operator" && chmod +x ./loglistener_operator
log "==> Step 1: Done."

# Step 2: Install loglistener via operator
log "==> Step 2: Installing loglistener via operator..."
# Note: `yes` may exit with SIGPIPE (141) when the downstream process closes stdin,
# which is expected and not an error. We temporarily disable pipefail to handle this.
set +o pipefail
yes | ./loglistener_operator install -r {{region}} --auth_mode=role --role_name={{role_name}} --upload_cvm_metadata=true -l ClawPro_CLS_TenCentCloud
install_exit_code=${PIPESTATUS[1]}
set -o pipefail
if [ "$install_exit_code" -ne 0 ]; then
    log "==> ERROR: loglistener_operator install failed with exit code $install_exit_code"
    exit "$install_exit_code"
fi
log "==> Step 2: Done."

# Step 3: Install CLS diagnostics metrics onboard CLI
# 该步骤依赖 openclaw CLI（仅 openclaw 类型实例有），Hermes/ACE 实例跳过。
# 使用 flock 文件锁防止与 set_model.sh / switch_model.sh 并发修改 openclaw.json
log "==> Step 3: Installing CLS diagnostics metrics onboard CLI..."
if command -v openclaw >/dev/null 2>&1; then
    LOCKFILE="$HOME/.openclaw/openclaw.lock"
    mkdir -p "$HOME/.openclaw"
    (
    # 先尝试非阻塞获取锁，失败则打印等待日志后再阻塞等待
    if ! flock -n 200; then
        echo "[INFO] 🔒 openclaw.json 正被其他脚本修改，等待锁释放..."
        flock -x 200
        echo "[INFO] 🔓 锁已获取，继续执行"
    fi
    npx --yes --registry "${MIRROR_BASE}/npm/" clawpro-diagnostics-metrics-cls-onboard-cli install --endpoint "{{region}}.cls.tencentyun.com" --credentialMode cvmRole --roleName "{{role_name}}" --metricTopicId "{{metric_topic_id}}" --traceTopicId "{{trace_topic_id}}" --enableReport true --traceEnabled true
    ) 200>"$LOCKFILE"
    log "==> Step 3: Done."
else
    log "==> Step 3: Skipped (non-openclaw agent type)."
fi