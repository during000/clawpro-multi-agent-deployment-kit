#!/usr/bin/env bash
set -euo pipefail

REMOTE=""
PORT="22"
DOMAIN=""
IDENTITY=""
TAG="latest"
SKIP_PREREQUISITES=0

usage() {
  echo "Usage: bash scripts/deploy-remote.sh --host root@server --port 22 --domain clawpro.example.com [--identity /absolute/key] [--tag TAG] [--skip-prerequisites]"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      REMOTE="${2:-}"
      shift 2
      ;;
    --port)
      PORT="${2:-}"
      shift 2
      ;;
    --domain)
      DOMAIN="${2:-}"
      shift 2
      ;;
    --identity)
      IDENTITY="${2:-}"
      shift 2
      ;;
    --tag)
      TAG="${2:-}"
      shift 2
      ;;
    --skip-prerequisites)
      SKIP_PREREQUISITES=1
      shift
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

if [[ ! "$REMOTE" =~ ^[A-Za-z0-9._-]+@[A-Za-z0-9.:-]+$ ]]; then
  echo "--host must look like root@server.example.com." >&2
  exit 2
fi

if [[ ! "$PORT" =~ ^[0-9]+$ || "$PORT" -lt 1 || "$PORT" -gt 65535 ]]; then
  echo "--port must be between 1 and 65535." >&2
  exit 2
fi

if [[ -z "$DOMAIN" || ! "$DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]]; then
  echo "--domain is required and must not include https:// or a path." >&2
  exit 2
fi

if [[ ! "$TAG" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "Invalid release tag." >&2
  exit 2
fi

if [[ -n "$IDENTITY" && ( "$IDENTITY" != /* || ! -f "$IDENTITY" ) ]]; then
  echo "--identity must be an existing absolute file path." >&2
  exit 2
fi

for command_name in ssh scp gh shasum; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Missing required command: $command_name" >&2
    exit 1
  fi
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DOWNLOAD_DIR="$REPO_DIR/.downloads/$TAG"

ARCHIVE="$(bash "$SCRIPT_DIR/fetch-release.sh" --tag "$TAG" --destination "$DOWNLOAD_DIR" | tail -n 1)"
CHECKSUM="$ARCHIVE.sha256"
if [[ ! -f "$ARCHIVE" || ! -f "$CHECKSUM" ]]; then
  echo "Release archive verification did not produce the expected files." >&2
  exit 1
fi

SSH_ARGS=(-p "$PORT" -o ServerAliveInterval=10 -o ServerAliveCountMax=6)
SCP_ARGS=(-P "$PORT" -o ServerAliveInterval=10 -o ServerAliveCountMax=6)
if [[ -n "$IDENTITY" ]]; then
  SSH_ARGS+=(-i "$IDENTITY")
  SCP_ARGS+=(-i "$IDENTITY")
fi

REMOTE_DIR="/root/clawpro-multi-agent-deploy/$TAG"
ssh "${SSH_ARGS[@]}" "$REMOTE" "mkdir -p '$REMOTE_DIR'"
scp "${SCP_ARGS[@]}" "$ARCHIVE" "$CHECKSUM" "$SCRIPT_DIR/remote-preflight.sh" "$SCRIPT_DIR/install-remote.sh" "$REMOTE:$REMOTE_DIR/"

REMOTE_ARGS=(
  bash "$REMOTE_DIR/install-remote.sh"
  --domain "$DOMAIN"
  --archive "$REMOTE_DIR/$(basename "$ARCHIVE")"
  --checksum "$REMOTE_DIR/$(basename "$CHECKSUM")"
)
if [[ "$SKIP_PREREQUISITES" -eq 1 ]]; then
  REMOTE_ARGS+=(--skip-prerequisites)
fi

ssh "${SSH_ARGS[@]}" "$REMOTE" "${REMOTE_ARGS[@]}"

echo "Deployment finished. Open: https://$DOMAIN/project-collaboration"
