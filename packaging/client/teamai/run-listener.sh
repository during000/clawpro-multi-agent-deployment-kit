#!/usr/bin/env bash
set -euo pipefail

WORKSPACE=""
TOOL="codebuddy"

usage() {
  echo "Usage: bash run-listener.sh --workspace /absolute/workspace [--tool codebuddy]"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --workspace)
      WORKSPACE="${2:-}"
      shift 2
      ;;
    --tool)
      TOOL="${2:-}"
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

if [[ -z "$WORKSPACE" || "$WORKSPACE" != /* || ! -d "$WORKSPACE" ]]; then
  echo "--workspace must be an existing absolute directory." >&2
  exit 2
fi

if ! command -v teamai >/dev/null 2>&1; then
  echo "teamai is not installed." >&2
  exit 1
fi

exec teamai agent-task-listen --tool "$TOOL" --cwd "$WORKSPACE"
