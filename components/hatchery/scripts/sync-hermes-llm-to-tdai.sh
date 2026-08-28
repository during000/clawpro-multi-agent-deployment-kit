#!/usr/bin/env bash
#
# sync-hermes-llm-to-tdai.sh — 读取系统 Hermes 模型配置，同步到 TDAI Gateway
#
# 前置条件：
#   1. memory-tencentdb-ctl.sh 存在
#   2. ~/.hermes/config.yaml 存在且包含模型配置
#
# 用法（命令行）：
#   bash sync-hermes-llm-to-tdai.sh [--restart] [--dry-run] [--hermes]
#
# 用法（TAT 注入）：
#   由 Go 层通过 TAT 下发，参数通过 {{mode}} / {{restart}} 占位符传入。
#   命令行参数 $@ 仍然支持（手动运维场景）。
#
# 可选环境变量：
#   HERMES_HOME            hermes 主目录（默认 ~/.hermes）
#   MEMORY_TENCENTDB_ROOT  TDAI 统一根目录（默认 ~/.memory-tencentdb）
#

# TAT 模板参数（Go 层传入，手动执行时为原始 {{...}} 字面量，会被忽略）
_tat_mode="{{mode}}"
_tat_restart="{{restart}}"

set -euo pipefail

SCRIPT_NAME="sync-hermes-llm-to-tdai"
USER_HOME="${HOME:-$(eval echo "~$(whoami)")}"

# ============================================================
# 路径
# ============================================================

HERMES_HOME="${HERMES_HOME:-$USER_HOME/.hermes}"
HERMES_CONFIG="$HERMES_HOME/config.yaml"

MEMORY_TENCENTDB_ROOT="${MEMORY_TENCENTDB_ROOT:-$USER_HOME/.memory-tencentdb}"
TDAI_INSTALL_DIR="${TDAI_INSTALL_DIR:-$MEMORY_TENCENTDB_ROOT/tdai-memory-openclaw-plugin}"

CTL_SCRIPT="$TDAI_INSTALL_DIR/scripts/memory-tencentdb-ctl.sh"

# ============================================================
# 通用 helpers
# ============================================================

log()  { printf '[%s] %s\n' "$SCRIPT_NAME" "$*"; }
warn() { printf '[%s:warn] %s\n' "$SCRIPT_NAME" "$*" >&2; }
die()  { printf '[%s:error] %s\n' "$SCRIPT_NAME" "$*" >&2; exit "${2:-1}"; }

source_sync_envs() {
    if [[ -r /etc/profile.d/memory-tencentdb-env.sh ]]; then
        # shellcheck disable=SC1091
        source /etc/profile.d/memory-tencentdb-env.sh
    fi

    if [[ "${SYNC_MODE:-standalone}" == "hermes" ]]; then
        if [[ -r /etc/profile.d/hermes-env.sh ]]; then
            # shellcheck disable=SC1091
            source /etc/profile.d/hermes-env.sh
        fi
        if [[ -d "$HERMES_HOME/env.d" ]]; then
            local f
            for f in "$HERMES_HOME"/env.d/*.sh; do
                [[ -r "$f" ]] || continue
                # shellcheck disable=SC1090
                source "$f"
            done
        fi
    fi
}

listening_pids() {
    local port="$1"
    if command -v lsof >/dev/null 2>&1; then
        lsof -nP -iTCP:"$port" -sTCP:LISTEN -t 2>/dev/null || true
    elif command -v ss >/dev/null 2>&1; then
        ss -ltnpH "sport = :$port" 2>/dev/null \
            | sed -n 's/.*pid=\([0-9]\+\).*/\1/p' | sort -u
    fi
}

should_restart_gateway() {
    source_sync_envs
    local port="${MEMORY_TENCENTDB_GATEWAY_PORT:-8420}"
    local pids
    pids="$(listening_pids "$port")"
    if [[ -n "$pids" ]]; then
        log "检测到 Gateway 正在运行: port=$port pid=$pids，允许执行 --restart"
        return 0
    fi
    warn "未检测到 Gateway 进程（port=$port）；本次仅同步配置，不执行 --restart，也不会拉起新进程。"
    return 1
}

# ============================================================
# Step 1: 检查 memory-tencentdb-ctl.sh 是否存在
# ============================================================

if [[ ! -f "$CTL_SCRIPT" ]]; then
    # 也尝试旧路径
    _LEGACY_CTL="$USER_HOME/.memory-tencentdb/tdai-memory-openclaw-plugin/scripts/memory-tencentdb-ctl.sh"
    if [[ -f "$_LEGACY_CTL" ]]; then
        CTL_SCRIPT="$_LEGACY_CTL"
        warn "使用旧路径: $CTL_SCRIPT"
    else
        die "memory-tencentdb-ctl.sh 不存在，TDAI 未安装。退出。" 1
    fi
fi

log "找到 ctl 脚本: $CTL_SCRIPT"

# ============================================================
# Step 2: 检查 hermes config.yaml
# ============================================================

if [[ ! -f "$HERMES_CONFIG" ]]; then
    die "Hermes 配置文件不存在: $HERMES_CONFIG" 1
fi

log "读取 Hermes 配置: $HERMES_CONFIG"

# ============================================================
# Step 3: 从 hermes config.yaml 提取模型配置
# ============================================================

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || die "必需命令未找到: $1" 127
}

need_cmd python3

# 使用 python3 安全解析 YAML，提取 model 段的 default / base_url / api_key
read_hermes_model_config() {
    python3 - "$HERMES_CONFIG" <<'PYEOF'
import sys, json

config_path = sys.argv[1]

# 尝试使用 PyYAML；如果不存在则手动逐行解析
try:
    import yaml
    with open(config_path, "r", encoding="utf-8") as f:
        cfg = yaml.safe_load(f) or {}
    model_section = cfg.get("model", {})
    result = {
        "model":    model_section.get("default", ""),
        "base_url": model_section.get("base_url", ""),
        "api_key":  model_section.get("api_key", ""),
    }
    print(json.dumps(result))
except ImportError:
    # PyYAML 不可用，用简单逐行解析提取 model 段
    import re
    with open(config_path, "r", encoding="utf-8") as f:
        lines = f.readlines()

    in_model = False
    result = {"model": "", "base_url": "", "api_key": ""}
    for line in lines:
        stripped = line.rstrip("\n")
        # 检测 model: 顶层段
        if re.match(r"^model:\s*$", stripped) or re.match(r"^model:\s*#", stripped):
            in_model = True
            continue
        # 进入新的顶层段则退出
        if in_model and re.match(r"^[A-Za-z_]", stripped):
            break
        if in_model:
            m = re.match(r"^\s+default:\s*(.+)", stripped)
            if m:
                result["model"] = m.group(1).strip().strip("'\"")
            m = re.match(r"^\s+base_url:\s*(.+)", stripped)
            if m:
                result["base_url"] = m.group(1).strip().strip("'\"")
            m = re.match(r"^\s+api_key:\s*(.+)", stripped)
            if m:
                result["api_key"] = m.group(1).strip().strip("'\"")
    print(json.dumps(result))
PYEOF
}

MODEL_JSON="$(read_hermes_model_config)"

HERMES_MODEL="$(printf '%s' "$MODEL_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["model"])')"
HERMES_BASE_URL="$(printf '%s' "$MODEL_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["base_url"])')"
HERMES_API_KEY="$(printf '%s' "$MODEL_JSON" | python3 -c 'import json,sys; print(json.load(sys.stdin)["api_key"])')"

# 校验必需字段
if [[ -z "$HERMES_MODEL" ]]; then
    die "Hermes 配置中 model.default 为空" 1
fi
if [[ -z "$HERMES_BASE_URL" ]]; then
    die "Hermes 配置中 model.base_url 为空" 1
fi
if [[ -z "$HERMES_API_KEY" ]]; then
    die "Hermes 配置中 model.api_key 为空" 1
fi

log "Hermes 模型配置:"
log "  model    = $HERMES_MODEL"
log "  base_url = $HERMES_BASE_URL"
log "  api_key  = <${#HERMES_API_KEY} chars>"

# ============================================================
# Step 4: 解析参数（TAT 占位符 + 命令行参数均支持）
# ============================================================

CTL_EXTRA_ARGS=()
SYNC_MODE="standalone"
REQUEST_RESTART=0

# 先从 TAT 占位符读取（被 TAT 替换后为实际值；手动执行时为 {{...}} 字面量，跳过）
if [[ "$_tat_mode" == "hermes" ]]; then
    SYNC_MODE="hermes"
    CTL_EXTRA_ARGS+=("--hermes")
elif [[ "$_tat_mode" == "standalone" ]]; then
    SYNC_MODE="standalone"
    CTL_EXTRA_ARGS+=("--standalone")
fi
if [[ "$_tat_restart" == "true" ]]; then
    REQUEST_RESTART=1
fi

# 命令行参数覆盖（手动运维场景）
for arg in "$@"; do
    case "$arg" in
        --restart)
            REQUEST_RESTART=1
            ;;
        --hermes)
            SYNC_MODE="hermes"
            CTL_EXTRA_ARGS+=("$arg")
            ;;
        --standalone)
            SYNC_MODE="standalone"
            CTL_EXTRA_ARGS+=("$arg")
            ;;
        --dry-run)
            CTL_EXTRA_ARGS+=("$arg")
            ;;
        *)
            warn "忽略未知参数: $arg"
            ;;
    esac
done

if [[ $REQUEST_RESTART -eq 1 ]]; then
    if should_restart_gateway; then
        CTL_RESTART_ARG="--restart"
    else
        log "已按新逻辑跳过 --restart。"
        CTL_RESTART_ARG=""
    fi
else
    CTL_RESTART_ARG=""
fi

# ============================================================
# Step 5: 调用 memory-tencentdb-ctl.sh config llm 同步配置
# ============================================================

log "调用 memory-tencentdb-ctl.sh config llm 同步配置到 TDAI ..."

bash "$CTL_SCRIPT" "${CTL_EXTRA_ARGS[@]+"${CTL_EXTRA_ARGS[@]}"}" \
    config llm \
    --api-key  "$HERMES_API_KEY" \
    --base-url "$HERMES_BASE_URL" \
    --model    "$HERMES_MODEL" \
    $CTL_RESTART_ARG

log "同步完成。"
log ""
log "如需立即重启 Gateway 使配置生效，请重新运行并追加 --restart："
log "  bash $0 --restart"
