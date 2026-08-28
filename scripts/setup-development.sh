#!/usr/bin/env bash
set -euo pipefail

REPOSITORY="during000/clawpro-multi-agent-deployment-kit"
TAG="v2026.08.28-poc.1"
DESTINATION=""

usage() {
  echo "Usage: bash scripts/setup-development.sh [--tag TAG] [--destination /absolute/path]"
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
  echo "Invalid source release tag." >&2
  exit 2
fi

for command_name in gh shasum tar; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Missing required command: $command_name" >&2
    exit 1
  fi
done

gh auth status >/dev/null

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
if [[ -z "$DESTINATION" ]]; then
  DESTINATION="$REPO_DIR/workspace"
fi

if [[ "$DESTINATION" != /* ]]; then
  echo "--destination must be an absolute path." >&2
  exit 2
fi

if [[ -d "$DESTINATION" && -n "$(find "$DESTINATION" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
  echo "Development destination is not empty: $DESTINATION" >&2
  echo "Choose a new directory so existing work is not overwritten." >&2
  exit 1
fi

DOWNLOAD_DIR="$REPO_DIR/.downloads/sources/$TAG"
mkdir -p "$DOWNLOAD_DIR" "$DESTINATION"

gh release download "$TAG" \
  --repo "$REPOSITORY" \
  --pattern 'clawpro-multi-agent-source-*.tar.gz' \
  --pattern 'clawpro-multi-agent-source-*.tar.gz.sha256' \
  --dir "$DOWNLOAD_DIR" \
  --clobber

CHECKSUM="$(find "$DOWNLOAD_DIR" -maxdepth 1 -type f -name '*.tar.gz.sha256' -print -quit)"
ARCHIVE="$(find "$DOWNLOAD_DIR" -maxdepth 1 -type f -name '*.tar.gz' -print -quit)"
if [[ -z "$CHECKSUM" || -z "$ARCHIVE" ]]; then
  echo "Source archive or checksum was not downloaded." >&2
  exit 1
fi

cd "$DOWNLOAD_DIR"
shasum -a 256 -c "$(basename "$CHECKSUM")"
tar -xzf "$ARCHIVE" -C "$DESTINATION"

SOURCE_ROOT="$(find "$DESTINATION" -mindepth 1 -maxdepth 1 -type d -name 'clawpro-multi-agent-source-*' -print -quit)"
if [[ -z "$SOURCE_ROOT" ]]; then
  echo "Source archive did not contain the expected workspace." >&2
  exit 1
fi

echo "Development source is ready: $SOURCE_ROOT"
echo "Read: $SOURCE_ROOT/CODEBUDDY.md"
