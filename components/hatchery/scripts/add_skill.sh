#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

AGENT_ID="{{agent_id}}"
# agent_id 是可选参数；旧调用方可能没有在 TAT params 中传递该 key，
# 此时模板占位符会保留为字面量 {{agent_id}}，需要按空值处理。
if [[ "$AGENT_ID" == *"{{"*"}}"* ]]; then
    AGENT_ID=""
fi
if [ -n "$AGENT_ID" ]; then
    WORKSPACE_ROOT=$(realpath -m "$HOME/.openclaw/workspace")
    AGENT_WORKSPACE=$(realpath -m "${WORKSPACE_ROOT}/${AGENT_ID}")
    case "$AGENT_WORKSPACE" in
        "${WORKSPACE_ROOT}"/*) ;;
        *) echo "Invalid agent workspace: ${AGENT_ID}" >&2; exit 1 ;;
    esac
    if [ ! -d "$AGENT_WORKSPACE" ]; then
        echo "Agent workspace not found: ${AGENT_ID}" >&2
        exit 1
    fi
    AGENT_SKILLS_DIR="${AGENT_WORKSPACE}/skills"
    mkdir -p "$AGENT_SKILLS_DIR"
    skillhub --dir "$AGENT_SKILLS_DIR" install "{{skill_name}}" --force
    exit 0
fi

stderr=$(clawhub install "{{skill_name}}" --force 2>&1 >/dev/null) && exit 0 || true

echo "$stderr" >&2
# clawhub install 失败，尝试降级到 skillhub（不限定具体错误类型，
# 无论是 Rate limit exceeded、Request timed out 还是其他错误都走降级）
if command -v skillhub &>/dev/null; then
    echo "clawhub install failed, falling back to skillhub..." >&2
    skillhub --dir ~/.openclaw/workspace/skills install "{{skill_name}}" --force
else
    exit 1
fi
