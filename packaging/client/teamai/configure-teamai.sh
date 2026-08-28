#!/usr/bin/env bash
set -euo pipefail

ENDPOINT=""
TOKEN_FILE=""
WORKSPACE=""
PROJECT_ID=""

usage() {
  echo "Usage: bash configure-teamai.sh --endpoint https://clawpro.example.com --token-file /path/to/token --workspace /absolute/workspace --project-id PROJECT_ID"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --endpoint)
      ENDPOINT="${2:-}"
      shift 2
      ;;
    --token-file)
      TOKEN_FILE="${2:-}"
      shift 2
      ;;
    --workspace)
      WORKSPACE="${2:-}"
      shift 2
      ;;
    --project-id)
      PROJECT_ID="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ ! "$ENDPOINT" =~ ^https://[^/]+/?$ ]]; then
  echo "--endpoint must be an HTTPS origin, for example https://clawpro.example.com" >&2
  exit 2
fi

if [[ -z "$TOKEN_FILE" || ! -f "$TOKEN_FILE" ]]; then
  echo "--token-file must point to a readable file." >&2
  exit 2
fi

if [[ -z "$WORKSPACE" || "$WORKSPACE" != /* || ! -d "$WORKSPACE" ]]; then
  echo "--workspace must be an existing absolute directory." >&2
  exit 2
fi

if [[ -z "$PROJECT_ID" ]]; then
  echo "--project-id is required. Copy it from the ClawPro project page." >&2
  exit 2
fi

if ! command -v teamai >/dev/null 2>&1; then
  echo "teamai is not installed. Run install-teamai.sh first." >&2
  exit 1
fi

TOKEN="$(tr -d '\r\n' < "$TOKEN_FILE")"
if [[ -z "$TOKEN" ]]; then
  echo "Token file is empty." >&2
  exit 2
fi

cd "$WORKSPACE"
TEAMAI_API_TOKEN="$TOKEN" teamai init --http "${ENDPOINT%/}" --scope project --agent codebuddy --force
TEAMAI_API_TOKEN="$TOKEN" teamai bind-project --project-id "$PROJECT_ID"

echo "TeamAI project configuration completed for: $WORKSPACE"
echo "Start the listener with run-listener.sh."
