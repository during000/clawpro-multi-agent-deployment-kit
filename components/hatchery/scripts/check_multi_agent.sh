#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

CONFIG="$HOME/.openclaw/openclaw.json"

if [ ! -s "$CONFIG" ]; then
  echo "CONFIG_NOT_FOUND: $CONFIG is missing or empty" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "JQ_NOT_FOUND: jq is required to parse $CONFIG" >&2
  exit 1
fi

if ! count="$(jq '(.agents.list // []) | if type == "array" then length else 0 end' "$CONFIG")"; then
  echo "CONFIG_PARSE_ERROR: failed to parse $CONFIG" >&2
  exit 1
fi

if ! [[ "$count" =~ ^[0-9]+$ ]]; then
  echo "COUNT_PARSE_ERROR: invalid count from $CONFIG: $count" >&2
  exit 1
fi

if (( count < 1 )); then
  count=1
fi

printf '{"count":%s}\n' "$count"
