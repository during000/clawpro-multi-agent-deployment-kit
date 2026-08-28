#!/bin/bash
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)
export NO_COLOR=1

openclaw pairing approve {{module}} {{code}}
