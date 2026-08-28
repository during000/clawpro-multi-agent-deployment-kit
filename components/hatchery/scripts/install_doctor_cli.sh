#!/bin/bash
set -euo pipefail

# install_doctor_cli.sh
# 从 SMH 下载 doctor-cli 二进制并安装到 /usr/local/bin/
#
# 参数（通过 TAT 模板变量注入）：
#   {{download_url}}  - SMH 下载 URL（含 access_token）

DOWNLOAD_URL="{{download_url}}"
INSTALL_PATH="/usr/local/bin/doctor-cli"

echo "Installing doctor-cli..."

# 检查 curl 是否可用
if ! command -v curl >/dev/null 2>&1; then
    echo "Missing curl" >&2
    exit 1
fi

# 下载
curl -fsSL --retry 3 --connect-timeout 10 --max-time 30 \
    -o "${INSTALL_PATH}" "${DOWNLOAD_URL}"

if [ ! -s "${INSTALL_PATH}" ]; then
    echo "Download failed: empty file" >&2
    exit 1
fi

chmod +x "${INSTALL_PATH}"

# 验证下载的是有效 ELF 可执行文件
if ! file "${INSTALL_PATH}" | grep -q "ELF"; then
    echo "ERROR: downloaded file is not a valid ELF binary" >&2
    file "${INSTALL_PATH}" >&2
    exit 1
fi

echo "doctor-cli installed successfully"
