#!/usr/bin/env bash
set -euo pipefail

: "${DEPLOY_SSH_PRIVATE_KEY:?DEPLOY_SSH_PRIVATE_KEY is required}"
: "${DEPLOY_HOST:?DEPLOY_HOST is required}"
: "${DEPLOY_PORT:?DEPLOY_PORT is required}"
: "${DEPLOY_USER:?DEPLOY_USER is required}"
: "${DEPLOY_PATH:?DEPLOY_PATH is required}"

mkdir -p ~/.ssh
chmod 700 ~/.ssh
KEY_FILE="$HOME/.ssh/clawpro_demo_deploy_key"
KEY_CONTENT="${DEPLOY_SSH_PRIVATE_KEY//$'\\n'/$'\n'}"

printf '%s\n' "$KEY_CONTENT" | tr -d '\r' > "$KEY_FILE"
chmod 600 ~/.ssh/clawpro_demo_deploy_key

if ! ssh-keygen -y -f "$KEY_FILE" >/dev/null 2>&1; then
  first_line="$(head -n 1 "$KEY_FILE" || true)"
  if [[ "$first_line" == ssh-* ]]; then
    echo "DEPLOY_SSH_PRIVATE_KEY looks like a public key. Paste the private key content into the Stream credential." >&2
  elif [[ "$first_line" == "-----BEGIN "* ]]; then
    echo "DEPLOY_SSH_PRIVATE_KEY has a private-key header but ssh cannot parse it. Check that the full key, including BEGIN and END lines, was pasted into Stream." >&2
  else
    echo "DEPLOY_SSH_PRIVATE_KEY does not look like a valid SSH private key." >&2
  fi
  exit 1
fi

ssh-keyscan -p "$DEPLOY_PORT" "$DEPLOY_HOST" >> ~/.ssh/known_hosts 2>/dev/null

SSH_OPTS=(
  -i ~/.ssh/clawpro_demo_deploy_key
  -o IdentitiesOnly=yes
  -o HostkeyAlgorithms=+ssh-rsa
  -o PubkeyAcceptedKeyTypes=+ssh-rsa
  -p "$DEPLOY_PORT"
)

SCP_OPTS=(
  -i ~/.ssh/clawpro_demo_deploy_key
  -o IdentitiesOnly=yes
  -o HostkeyAlgorithms=+ssh-rsa
  -o PubkeyAcceptedKeyTypes=+ssh-rsa
  -P "$DEPLOY_PORT"
)

REMOTE="${DEPLOY_USER}@${DEPLOY_HOST}"
BACKUP_DIR="/data/home/${DEPLOY_USER}/deploy-backups"
WORK_DIR="/data/home/${DEPLOY_USER}/clawpro-demo-ci-work/openclaw-enterprise"
REMOTE_ARCHIVE="/tmp/clawpro-demo-src-${CI_PIPELINE_ID:-manual}-$(date +%Y%m%d%H%M%S).tgz"
LOCAL_ARCHIVE="$(mktemp -t clawpro-demo-src.XXXXXX.tgz)"
STAMP="$(date +%Y%m%d%H%M%S)"
TAR_OPTS=()

if tar --no-xattrs -cf /dev/null --files-from /dev/null >/dev/null 2>&1; then
  TAR_OPTS+=(--no-xattrs)
fi

if tar --no-mac-metadata -cf /dev/null --files-from /dev/null >/dev/null 2>&1; then
  TAR_OPTS+=(--no-mac-metadata)
fi

LC_ALL=C tar \
  "${TAR_OPTS[@]}" \
  --exclude='./.git' \
  --exclude='./node_modules' \
  --exclude='./dist' \
  -czf "$LOCAL_ARCHIVE" \
  .

scp "${SCP_OPTS[@]}" "$LOCAL_ARCHIVE" "$REMOTE:$REMOTE_ARCHIVE"
rm -f "$LOCAL_ARCHIVE"

ssh "${SSH_OPTS[@]}" "$REMOTE" "
  set -euo pipefail
  mkdir -p '$BACKUP_DIR'
  rm -rf '$WORK_DIR'
  mkdir -p '$WORK_DIR'
  tar -xzf '$REMOTE_ARCHIVE' -C '$WORK_DIR'
  rm -f '$REMOTE_ARCHIVE'

  cd '$WORK_DIR'
  pnpm install --frozen-lockfile
  pnpm build

  if [ ! -d '$WORK_DIR/dist/public' ]; then
    echo 'Build output not found: $WORK_DIR/dist/public' >&2
    exit 1
  fi

  sudo -n mkdir -p '$DEPLOY_PATH'
  if [ -d '$DEPLOY_PATH' ] && [ \"\$(find '$DEPLOY_PATH' -mindepth 1 -maxdepth 1 2>/dev/null | head -1)\" ]; then
    sudo -n tar -C '$DEPLOY_PATH' -czf '$BACKUP_DIR/clawprodemo-before-ci-$STAMP.tgz' .
  fi
  sudo -n chown -R '$DEPLOY_USER':users '$DEPLOY_PATH' '$BACKUP_DIR'

  rsync -az --delete '$WORK_DIR/dist/public/' '$DEPLOY_PATH/'

  sudo -n nginx -t
  sudo -n systemctl reload nginx
  curl -fsSI -H 'Host: clawprodemo.woa.com' http://127.0.0.1/ >/dev/null
  curl -fsSI -H 'Host: clawprodemo.woa.com' http://127.0.0.1/admin/member-management >/dev/null
"
