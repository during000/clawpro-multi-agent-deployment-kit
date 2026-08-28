#!/usr/bin/env bash
set -euo pipefail

if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  echo "Remote deployment requires root." >&2
  exit 1
fi

if [[ "$(uname -s)" != "Linux" || "$(uname -m)" != "x86_64" ]]; then
  echo "Only Linux x86_64 is supported by this release." >&2
  exit 1
fi

install_packages() {
  if command -v apt-get >/dev/null 2>&1; then
    export DEBIAN_FRONTEND=noninteractive
    apt-get update
    apt-get install -y docker.io python3 openssl curl ca-certificates coreutils tar
    return
  fi
  if command -v dnf >/dev/null 2>&1; then
    dnf install -y docker python3 openssl curl ca-certificates coreutils tar
    return
  fi
  if command -v yum >/dev/null 2>&1; then
    yum install -y docker python3 openssl curl ca-certificates coreutils tar
    return
  fi
  echo "No supported package manager found. Install Docker, Python 3, OpenSSL and curl manually." >&2
  exit 1
}

MISSING=0
for command_name in docker python3 openssl curl systemctl tar sha256sum; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    MISSING=1
  fi
done

if [[ "$MISSING" -eq 1 ]]; then
  install_packages
fi

for command_name in docker python3 openssl curl systemctl tar sha256sum; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Required command is still missing: $command_name" >&2
    exit 1
  fi
done

systemctl enable --now docker.service
docker info >/dev/null

echo "Remote server preflight passed."
