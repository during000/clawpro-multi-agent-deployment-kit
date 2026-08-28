#!/bin/bash
set -euo pipefail

SKILLHUB="{{.SkillHub}}"

export XDG_RUNTIME_DIR=/run/user/$(id -u)

if [[ -n "$SKILLHUB" ]]; then
    # 自动探测新系统上的实际运行用户，不依赖 Go 侧传入的 {{.RuntimeUser}}
    # （重装后镜像可能切换运行用户，如 agentuser→ubuntu，旧值不可信）
    RUNTIME_USER=""
    for candidate_home in /home/*/; do
        candidate_home="${candidate_home%/}"
        candidate="$(basename "$candidate_home")"
        if ! id "$candidate" >/dev/null 2>&1; then
            continue
        fi
        if [ -d "${candidate_home}/.hermes" ] || [ -x "${candidate_home}/.local/bin/hermes" ] || [ -x "${candidate_home}/.local/bin/harness" ]; then
            RUNTIME_USER="$candidate"
            break
        fi
    done
    if [[ -z "$RUNTIME_USER" ]]; then
        if [ -d "/root/.hermes" ] || [ -x "/root/.local/bin/hermes" ]; then
            RUNTIME_USER="root"
        fi
    fi
    # 仍未找到：取 /home 下第一个真实用户；否则 root
    if [[ -z "$RUNTIME_USER" ]]; then
        for candidate_home in /home/*/; do
            candidate="$(basename "${candidate_home%/}")"
            if id "$candidate" >/dev/null 2>&1; then
                RUNTIME_USER="$candidate"
                break
            fi
        done
    fi
    if [[ -z "$RUNTIME_USER" ]]; then
        RUNTIME_USER="root"
    fi

    if [[ "$RUNTIME_USER" == "root" ]]; then
        TARGET_HOME="/root"
    else
        TARGET_HOME="/home/${RUNTIME_USER}"
    fi

    mkdir -p "${TARGET_HOME}/.config/skillhub"
    cat > "${TARGET_HOME}/.config/skillhub/config.json" << EOF
{
  "registry": "$SKILLHUB"
}
EOF
    if [[ "$RUNTIME_USER" != "root" ]] && id "$RUNTIME_USER" >/dev/null 2>&1; then
        chown -R "$RUNTIME_USER:$RUNTIME_USER" "${TARGET_HOME}/.config/skillhub"
    fi
fi
