#!/usr/bin/env bash
set -euo pipefail

SERVICES=(clawpro-hatchery clawpro-orchestrator clawpro-nginx)

for service_name in "${SERVICES[@]}"; do
  service_ready=0
  for _ in $(seq 1 30); do
    if systemctl is-active --quiet "${service_name}.service"; then
      service_ready=1
      break
    fi
    sleep 2
  done
  if [[ "$service_ready" -ne 1 ]]; then
    echo "FAIL: ${service_name}.service is not active" >&2
    systemctl status "${service_name}.service" --no-pager >&2 || true
    exit 1
  fi
  echo "OK: ${service_name}.service"
done

http_ready=0
for _ in $(seq 1 30); do
  if curl --fail --silent http://127.0.0.1/health >/dev/null; then
    http_ready=1
    break
  fi
  sleep 2
done
if [[ "$http_ready" -ne 1 ]]; then
  echo "FAIL: http://127.0.0.1/health did not become ready" >&2
  exit 1
fi
echo "OK: http://127.0.0.1/health"

curl --fail --silent --show-error http://127.0.0.1/ | grep -qi '<!doctype html'
echo "OK: frontend index"

echo "ClawPro server health check passed."
