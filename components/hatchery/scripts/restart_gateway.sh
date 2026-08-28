#!/bin/bash
set -euo pipefail

export XDG_RUNTIME_DIR=/run/user/$(id -u)

systemctl --user restart openclaw-gateway
