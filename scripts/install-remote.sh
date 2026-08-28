#!/usr/bin/env bash
set -euo pipefail

DOMAIN=""
ARCHIVE=""
CHECKSUM=""
SKIP_PREREQUISITES=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain)
      DOMAIN="${2:-}"
      shift 2
      ;;
    --archive)
      ARCHIVE="${2:-}"
      shift 2
      ;;
    --checksum)
      CHECKSUM="${2:-}"
      shift 2
      ;;
    --skip-prerequisites)
      SKIP_PREREQUISITES=1
      shift
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  echo "Remote installation requires root." >&2
  exit 1
fi

if [[ -z "$DOMAIN" || ! "$DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]]; then
  echo "A valid --domain is required." >&2
  exit 2
fi

if [[ -z "$ARCHIVE" || ! -f "$ARCHIVE" || -z "$CHECKSUM" || ! -f "$CHECKSUM" ]]; then
  echo "The deployment archive or checksum is missing." >&2
  exit 2
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [[ "$SKIP_PREREQUISITES" -eq 0 ]]; then
  bash "$SCRIPT_DIR/remote-preflight.sh"
fi

cd "$(dirname "$ARCHIVE")"
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum -c "$(basename "$CHECKSUM")"
else
  shasum -a 256 -c "$(basename "$CHECKSUM")"
fi

ARCHIVE_ROOT="$(tar -tzf "$ARCHIVE" | sed -n '1s|/.*||p')"
if [[ -z "$ARCHIVE_ROOT" || ! "$ARCHIVE_ROOT" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "Unable to determine a safe archive root." >&2
  exit 1
fi

tar -xzf "$ARCHIVE"
INSTALLER="$(dirname "$ARCHIVE")/$ARCHIVE_ROOT/server/install-server.sh"
if [[ ! -f "$INSTALLER" ]]; then
  echo "Server installer is missing from the release archive." >&2
  exit 1
fi

bash "$INSTALLER" --domain "$DOMAIN"
/opt/clawpro-multi-agent/bin/healthcheck

echo "Remote ClawPro deployment completed: https://$DOMAIN/project-collaboration"
