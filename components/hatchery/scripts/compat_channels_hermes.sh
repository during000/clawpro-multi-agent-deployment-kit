#!/bin/bash
# =============================================================================
# Hermes 通道依赖兼容修复脚本
# -----------------------------------------------------------------------------
# 升级后恢复各通道的可选 Python 依赖（如 dingtalk、msteams 等不在核心依赖中的包）。
# 每个修复项独立为一个函数，新增通道：追加函数 + 在 main() 中调用即可。
#
# 设计原则（与 compat_plugins.sh 一致）：
#   - 幂等：重复执行结果一致
#   - 容错：单项失败不影响其他项，整体以 0 退出（不阻断升级主流程）
# =============================================================================

set -u  # 不开 -e：单项失败不终止全局

export PATH="$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin:$PATH"
export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"

# ========== 日志系统初始化 ==========
HERMES_HOME="${HOME}/.hermes"
LOG_DIR="${HERMES_HOME}/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
SCRIPT_NAME="compat_channels_hermes"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# 同时输出到终端（供 TAT RunScript 抓取）和落盘日志
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

LOG_PREFIX="[compat-channels-hermes]"
ENV_FILE="${HERMES_HOME}/.env"

# ========== 公共工具函数 ==========

# find_hermes_python: 定位 Hermes venv 中的 Python
find_hermes_python() {
    if [ -x "${HERMES_HOME}/hermes-agent/venv/bin/python" ]; then
        echo "${HERMES_HOME}/hermes-agent/venv/bin/python"
        return 0
    fi
    if command -v python3 >/dev/null 2>&1; then
        command -v python3
        return 0
    fi
    if command -v python >/dev/null 2>&1; then
        command -v python
        return 0
    fi
    return 1
}

# pip_install_pkg <pkg_spec> [<pkg_spec> ...]
# 在 Hermes venv 中安装 Python 包，优先清华镜像，回退 PyPI 官方源。
pip_install_pkg() {
    local py
    py="$(find_hermes_python)" || {
        echo "${LOG_PREFIX} ✗ 未找到 Hermes Python，跳过安装"
        return 1
    }

    local pkgs=("$@")
    echo "${LOG_PREFIX} Python: $py"
    echo "${LOG_PREFIX} 安装: ${pkgs[*]}"

    if command -v uv >/dev/null 2>&1; then
        if UV_HTTP_TIMEOUT=120 timeout 300 uv pip install --verbose --python "$py" \
            -i https://mirrors.aliyun.com/pypi/simple/ "${pkgs[@]}" 2>&1; then
            echo "${LOG_PREFIX} ✓ uv 安装成功"
            return 0
        fi
        if UV_HTTP_TIMEOUT=120 timeout 300 uv pip install --verbose --python "$py" \
            -i https://pypi.org/simple/ "${pkgs[@]}" 2>&1; then
            echo "${LOG_PREFIX} ✓ uv 安装成功（回退 PyPI）"
            return 0
        fi
    elif "$py" -m pip --version >/dev/null 2>&1; then
        if timeout 300 "$py" -m pip install --progress-bar on \
            -i https://mirrors.aliyun.com/pypi/simple/ "${pkgs[@]}" 2>&1; then
            echo "${LOG_PREFIX} ✓ pip 安装成功"
            return 0
        fi
        if timeout 300 "$py" -m pip install --progress-bar on \
            -i https://pypi.org/simple/ "${pkgs[@]}" 2>&1; then
            echo "${LOG_PREFIX} ✓ pip 安装成功（回退 PyPI）"
            return 0
        fi
    fi

    echo "${LOG_PREFIX} ✗ 安装失败: ${pkgs[*]}"
    return 1
}

# check_import: 验证 Python 包是否可 import
check_import() {
    local py module="$1"
    py="$(find_hermes_python)" || return 1
    "$py" -c "import ${module}" 2>/dev/null
}

# channel_configured: 检查 .env 中是否存在指定 key（非空）
channel_configured() {
    local key="$1"
    [ -f "$ENV_FILE" ] || return 1
    local val
    val="$(grep "^${key}=" "$ENV_FILE" 2>/dev/null | cut -d= -f2- | tr -d '[:space:]')"
    [ -n "$val" ]
}

# =============================================================================
# 修复项：钉钉通道依赖
#   - 检测：DINGTALK_CLIENT_ID 在 .env 中存在且非空
#   - 安装：dingtalk-stream、alibabacloud-dingtalk、qrcode
#   - 验证：import dingtalk_stream
# =============================================================================
fix_ddingtalk_deps() {
    echo ""
    echo "${LOG_PREFIX} [fix_ddingtalk_deps] 开始执行钉钉通道依赖修复"

    if ! channel_configured "DINGTALK_CLIENT_ID"; then
        echo "${LOG_PREFIX} [fix_ddingtalk_deps] 钉钉通道未配置，跳过"
        return
    fi

    # 检查是否已安装（幂等）
    if check_import "dingtalk_stream"; then
        echo "${LOG_PREFIX} [fix_ddingtalk_deps] ✓ dingtalk-stream 已可导入，无需修复"
        return
    fi

    echo "${LOG_PREFIX} [fix_ddingtalk_deps] 钉钉通道已配置但依赖缺失，开始安装..."
    pip_install_pkg "dingtalk-stream" "alibabacloud-dingtalk" "qrcode"

    if check_import "dingtalk_stream"; then
        echo "${LOG_PREFIX} [fix_ddingtalk_deps] ✓ 钉钉依赖修复完成"
    else
        echo "${LOG_PREFIX} [fix_ddingtalk_deps] ⚠ 钉钉依赖安装后仍不可用（日志已记录，gateway 启动后可通过 acli 查看状态）"
    fi

    echo "${LOG_PREFIX} [fix_ddingtalk_deps] 完成"
}

# =============================================================================
# 修复项：Microsoft Teams 通道依赖
#   - 检测：TEAMS_CLIENT_ID 在 .env 中存在且非空
#   - 安装：microsoft-teams-apps
#   - 验证：import microsoft_teams
# =============================================================================
fix_msteams_deps() {
    echo ""
    echo "${LOG_PREFIX} [fix_msteams_deps] 开始执行 Microsoft Teams 通道依赖修复"

    if ! channel_configured "TEAMS_CLIENT_ID"; then
        echo "${LOG_PREFIX} [fix_msteams_deps] Microsoft Teams 通道未配置，跳过"
        return
    fi

    # 检查是否已安装（幂等）
    if check_import "microsoft_teams"; then
        echo "${LOG_PREFIX} [fix_msteams_deps] ✓ microsoft_teams 已可导入，无需修复"
        return
    fi

    echo "${LOG_PREFIX} [fix_msteams_deps] Microsoft Teams 通道已配置但依赖缺失，开始安装..."
    # 与 set_channel_hermes.sh::ensure_hermes_teams_deps 保持一致
    pip_install_pkg "microsoft-teams-apps==2.0.13.4"

    if check_import "microsoft_teams"; then
        echo "${LOG_PREFIX} [fix_msteams_deps] ✓ Microsoft Teams 依赖修复完成"
    else
        echo "${LOG_PREFIX} [fix_msteams_deps] ⚠ Microsoft Teams 依赖安装后仍不可用（日志已记录）"
    fi

    echo "${LOG_PREFIX} [fix_msteams_deps] 完成"
}

# =============================================================================
# 修复项：Slack 通道依赖
#   - 检测：SLACK_APP_TOKEN 在 .env 中存在且非空
#   - 安装：slack-sdk（如果不在核心依赖中）
#   - 验证：import slack_sdk
# =============================================================================
fix_slack_deps() {
    echo ""
    echo "${LOG_PREFIX} [fix_slack_deps] 开始执行 Slack 通道依赖修复"

    if ! channel_configured "SLACK_APP_TOKEN"; then
        echo "${LOG_PREFIX} [fix_slack_deps] Slack 通道未配置，跳过"
        return
    fi

    # slack-sdk 通常在核心依赖中，但仍做兜底检查
    if check_import "slack_sdk"; then
        echo "${LOG_PREFIX} [fix_slack_deps] ✓ slack_sdk 已可导入，无需修复"
        return
    fi

    echo "${LOG_PREFIX} [fix_slack_deps] Slack 通道已配置但依赖缺失，开始安装..."
    pip_install_pkg "slack-sdk"

    if check_import "slack_sdk"; then
        echo "${LOG_PREFIX} [fix_slack_deps] ✓ Slack 依赖修复完成"
    else
        echo "${LOG_PREFIX} [fix_slack_deps] ⚠ Slack 依赖安装后仍不可用"
    fi

    echo "${LOG_PREFIX} [fix_slack_deps] 完成"
}

# ========== 主执行逻辑 ==========
main() {
    echo "${LOG_PREFIX} 开始执行 Hermes 通道依赖兼容修复"
    echo "${LOG_PREFIX} ENV_FILE=${ENV_FILE}"
    echo "${LOG_PREFIX} HERMES_HOME=${HERMES_HOME}"

    if [ ! -f "$ENV_FILE" ]; then
        echo "${LOG_PREFIX} .env 文件不存在，跳过所有通道依赖修复"
        echo "${LOG_PREFIX} 所有通道依赖兼容修复执行完毕"
        echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="
        exit 0
    fi

    # 按需添加新的通道修复函数
    fix_ddingtalk_deps
    fix_msteams_deps
    fix_slack_deps

    echo ""
    echo "${LOG_PREFIX} 所有通道依赖兼容修复执行完毕"
    echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="
    echo ""
}

main "$@"

# 兼容脚本永远以 0 退出，避免阻断升级主流程；问题通过日志体现
exit 0
