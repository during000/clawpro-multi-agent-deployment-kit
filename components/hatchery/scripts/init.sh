#!/bin/bash
set -euo pipefail

SKILLHUB="{{.SkillHub}}"
RUNTIME_USER="{{.RuntimeUser}}"

export XDG_RUNTIME_DIR=/run/user/$(id -u)

if [[ -n "$SKILLHUB" ]]; then
    if [[ -z "$RUNTIME_USER" ]]; then
        for candidate_home in /home/*/; do
            candidate_home="${candidate_home%/}"
            candidate="$(basename "$candidate_home")"
            if [ -d "${candidate_home}/.openclaw" ] || [ -x "${candidate_home}/.local/bin/openclaw" ] || [ -x "${candidate_home}/.local/share/pnpm/openclaw" ]; then
                RUNTIME_USER="$candidate"
                break
            fi
        done
        if [[ -z "$RUNTIME_USER" ]]; then
            if [ -d "/root/.openclaw" ] || [ -x "/root/.local/bin/openclaw" ] || [ -x "/root/.local/share/pnpm/openclaw" ]; then
                RUNTIME_USER="root"
            fi
        fi
        if [[ -z "$RUNTIME_USER" ]]; then
            RUNTIME_USER="root"
        fi
    fi

    if [[ "$RUNTIME_USER" == "root" ]]; then
        TARGET_HOME="/root"
    else
        TARGET_HOME="/home/${RUNTIME_USER}"
    fi

    mkdir -p "${TARGET_HOME}/.config/clawhub"
    cat > "${TARGET_HOME}/.config/clawhub/config.json" << EOF
{
  "registry": "$SKILLHUB"
}
EOF
    if [[ "$RUNTIME_USER" != "root" ]] && id "$RUNTIME_USER" >/dev/null 2>&1; then
        chown -R "$RUNTIME_USER:$RUNTIME_USER" "${TARGET_HOME}/.config/clawhub"
    fi
fi
