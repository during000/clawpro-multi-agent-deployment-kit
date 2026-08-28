#!/usr/bin/env bash
set -euo pipefail

REPOSITORY="during000/clawpro-multi-agent-deployment-kit"
TAG="latest"
DESTINATION=""

usage() {
  echo "Usage: bash scripts/fetch-release.sh [--tag v2026.08.28-poc.1] [--destination /absolute/path]"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag)
      TAG="${2:-}"
      shift 2
      ;;
    --destination)
      DESTINATION="${2:-}"
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

if [[ ! "$TAG" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "Invalid release tag." >&2
  exit 2
fi

for command_name in gh shasum; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Missing required command: $command_name" >&2
    exit 1
  fi
done

gh auth status >/dev/null

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
if [[ -z "$DESTINATION" ]]; then
  DESTINATION="$REPO_DIR/.downloads/$TAG"
fi

if [[ "$DESTINATION" != /* ]]; then
  echo "--destination must be an absolute path." >&2
  exit 2
fi

mkdir -p "$DESTINATION"

DOWNLOAD_ARGS=(
  release download
  --repo "$REPOSITORY"
  --pattern 'clawpro-multi-agent-deployment-kit-*.tar.gz'
  --pattern 'clawpro-multi-agent-deployment-kit-*.tar.gz.sha256'
  --dir "$DESTINATION"
  --clobber
)

if [[ "$TAG" != "latest" ]]; then
  DOWNLOAD_ARGS+=("$TAG")
fi

gh "${DOWNLOAD_ARGS[@]}"

CHECKSUM_FILE="$(find "$DESTINATION" -maxdepth 1 -type f -name '*.tar.gz.sha256' -print -quit)"
if [[ -z "$CHECKSUM_FILE" ]]; then
  echo "Release checksum file was not downloaded." >&2
  exit 1
fi

cd "$DESTINATION"
shasum -a 256 -c "$(basename "$CHECKSUM_FILE")"

ARCHIVE="$(find "$DESTINATION" -maxdepth 1 -type f -name '*.tar.gz' -print -quit)"
if [[ -z "$ARCHIVE" ]]; then
  echo "Release archive was not downloaded." >&2
  exit 1
fi

printf '%s\n' "$ARCHIVE"
