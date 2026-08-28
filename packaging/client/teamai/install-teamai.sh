#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE="$(find "$SCRIPT_DIR" -maxdepth 1 -type f -name 'teamai-cli-*.tgz' -print -quit)"

if ! command -v node >/dev/null 2>&1 || ! command -v npm >/dev/null 2>&1; then
  echo "Node.js 22 and npm are required." >&2
  exit 1
fi

NODE_MAJOR="$(node -p 'Number(process.versions.node.split(".")[0])')"
if [[ "$NODE_MAJOR" -lt 22 ]]; then
  echo "Node.js 22 or newer is required; current version: $(node --version)" >&2
  exit 1
fi

if [[ -z "$PACKAGE" || ! -f "$PACKAGE" ]]; then
  echo "Missing TeamAI package in: $SCRIPT_DIR" >&2
  exit 1
fi

npm install --global "$PACKAGE" --ignore-scripts
teamai --version
echo "TeamAI installed. Continue with configure-teamai.sh."
