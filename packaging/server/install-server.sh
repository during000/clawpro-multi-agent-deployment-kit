#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACKAGE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
INSTALL_DIR="/opt/clawpro-multi-agent"
DOMAIN=""
INIT_USER="admin"

usage() {
  echo "Usage: sudo bash server/install-server.sh --domain clawpro.example.com [--install-dir /opt/clawpro-multi-agent] [--admin-user admin]"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain)
      DOMAIN="${2:-}"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="${2:-}"
      shift 2
      ;;
    --admin-user)
      INIT_USER="${2:-}"
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

if [[ ${EUID:-$(id -u)} -ne 0 ]]; then
  echo "Run this installer as root." >&2
  exit 1
fi

if [[ -z "$DOMAIN" || ! "$DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]]; then
  echo "--domain is required and must be a valid host name." >&2
  exit 2
fi

if [[ ! "$INSTALL_DIR" =~ ^/[A-Za-z0-9._/-]+$ ]]; then
  echo "--install-dir must be an absolute path without spaces." >&2
  exit 2
fi

if [[ ! "$INIT_USER" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "--admin-user contains unsupported characters." >&2
  exit 2
fi

if [[ "$(uname -s)" != "Linux" || "$(uname -m)" != "x86_64" ]]; then
  echo "This bundle currently supports Linux x86_64 only." >&2
  exit 1
fi

for command_name in docker systemctl python3 openssl install sed curl; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Missing required command: $command_name" >&2
    exit 1
  fi
done

if [[ ! -f "$PACKAGE_DIR/server/bin/hatchery-linux-amd64" ]]; then
  echo "Missing Hatchery binary in deployment package." >&2
  exit 1
fi

if [[ ! -f "$PACKAGE_DIR/server/frontend/index.html" ]]; then
  echo "Missing frontend build in deployment package." >&2
  exit 1
fi

install -d -m 0755 \
  "$INSTALL_DIR/bin" \
  "$INSTALL_DIR/config" \
  "$INSTALL_DIR/data" \
  "$INSTALL_DIR/frontend" \
  "$INSTALL_DIR/orchestrator"

install -m 0755 "$PACKAGE_DIR/server/bin/hatchery-linux-amd64" "$INSTALL_DIR/bin/hatchery"
install -m 0755 "$PACKAGE_DIR/server/healthcheck.sh" "$INSTALL_DIR/bin/healthcheck"
cp -a "$PACKAGE_DIR/server/frontend/." "$INSTALL_DIR/frontend/"
cp -a "$PACKAGE_DIR/server/orchestrator/." "$INSTALL_DIR/orchestrator/"
install -d -m 0755 \
  "$INSTALL_DIR/orchestrator/.runtime" \
  "$INSTALL_DIR/orchestrator/runtime-workspaces" \
  "$INSTALL_DIR/orchestrator/real-agent-workspaces"

HATCHERY_ENV="$INSTALL_DIR/config/hatchery.env"
if [[ ! -f "$HATCHERY_ENV" ]]; then
  INIT_PASS="$(openssl rand -hex 18)"
  ADMIN_TOKEN="$(openssl rand -hex 32)"
  COOKIE_SECRET="$(openssl rand -hex 32)"
  install -m 0600 /dev/null "$HATCHERY_ENV"
  {
    printf 'HATCHERY_INIT_USER=%s\n' "$INIT_USER"
    printf 'HATCHERY_INIT_PASS=%s\n' "$INIT_PASS"
    printf 'HATCHERY_ADMIN_TOKEN=%s\n' "$ADMIN_TOKEN"
    printf 'HATCHERY_COOKIE_SECRET=%s\n' "$COOKIE_SECRET"
  } > "$HATCHERY_ENV"
fi
chmod 0600 "$HATCHERY_ENV"

ORCHESTRATOR_ENV="$INSTALL_DIR/config/orchestrator.env"
if [[ ! -f "$ORCHESTRATOR_ENV" ]]; then
  install -m 0600 /dev/null "$ORCHESTRATOR_ENV"
  {
    printf 'TEAMAI_REMOTE_MODE=1\n'
    printf 'TEAMAI_PUBLIC_ENDPOINT=https://%s\n' "$DOMAIN"
    printf 'TEAMAI_REMOTE_DEVICE_NAME=ClawPro-Remote-TeamAI\n'
    printf 'TEAMAI_REMOTE_IWIKI_AUTHORIZED=0\n'
  } > "$ORCHESTRATOR_ENV"
fi
chmod 0600 "$ORCHESTRATOR_ENV"

escape_sed() {
  printf '%s' "$1" | sed 's/[&|]/\\&/g'
}

ESCAPED_INSTALL_DIR="$(escape_sed "$INSTALL_DIR")"
ESCAPED_DOMAIN="$(escape_sed "$DOMAIN")"

sed -e "s|__INSTALL_DIR__|$ESCAPED_INSTALL_DIR|g" -e "s|__DOMAIN__|$ESCAPED_DOMAIN|g" \
  "$PACKAGE_DIR/server/nginx.conf.template" > "$INSTALL_DIR/config/nginx.conf"

for service_name in clawpro-hatchery clawpro-orchestrator clawpro-nginx; do
  sed -e "s|__INSTALL_DIR__|$ESCAPED_INSTALL_DIR|g" -e "s|__DOMAIN__|$ESCAPED_DOMAIN|g" \
    "$PACKAGE_DIR/server/systemd/${service_name}.service.template" \
    > "/etc/systemd/system/${service_name}.service"
done

systemctl daemon-reload
systemctl enable --now clawpro-hatchery.service clawpro-orchestrator.service clawpro-nginx.service

echo "ClawPro installed at: $INSTALL_DIR"
echo "Open: https://$DOMAIN"
echo "Initial credentials are stored at: $HATCHERY_ENV"
echo "The credential values were intentionally not printed."
echo "Run: sudo $INSTALL_DIR/bin/healthcheck"
