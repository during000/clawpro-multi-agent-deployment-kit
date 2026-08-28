#!/bin/bash
set -euo pipefail

# write_doctor_identity.sh
# 将龙虾医生人设内容写入 OpenClaw workspace 的 IDENTITY.md
# 同时删除 BOOTSTRAP.md，跳过 OpenClaw 的引导流程。
#
# 参数（通过 TAT 模板变量注入）：
#   {{content}}  - Base64 编码的 IDENTITY.md 内容

CONTENT_B64="{{content}}"
WORKSPACE_DIR="$HOME/.openclaw/workspace"
IDENTITY_FILE="${WORKSPACE_DIR}/IDENTITY.md"
BOOTSTRAP_FILE="${WORKSPACE_DIR}/BOOTSTRAP.md"

mkdir -p "${WORKSPACE_DIR}"
echo "${CONTENT_B64}" | base64 -d > "${IDENTITY_FILE}"

# 删除 BOOTSTRAP.md，标记引导已完成
rm -f "${BOOTSTRAP_FILE}"

echo "IDENTITY.md written: $(wc -c < "${IDENTITY_FILE}") bytes, BOOTSTRAP.md removed"
