#!/usr/bin/env bash
set -euo pipefail

SOURCE_ROOT=""
OUTPUT_ROOT=""
SKIP_TESTS=0

usage() {
  echo "Usage: bash scripts/package-development.sh [--source-root /absolute/path] [--output-root /absolute/path] [--skip-tests]"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source-root)
      SOURCE_ROOT="${2:-}"
      shift 2
      ;;
    --output-root)
      OUTPUT_ROOT="${2:-}"
      shift 2
      ;;
    --skip-tests)
      SKIP_TESTS=1
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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
if [[ -z "$SOURCE_ROOT" ]]; then
  SOURCE_ROOT="$REPO_DIR"
fi
if [[ "$SOURCE_ROOT" != /* ]]; then
  echo "--source-root must be an absolute path." >&2
  exit 2
fi

if [[ -d "$SOURCE_ROOT/components" ]]; then
  COMPONENT_ROOT="$SOURCE_ROOT/components"
elif [[ -d "$SOURCE_ROOT/repos" ]]; then
  COMPONENT_ROOT="$SOURCE_ROOT/repos"
else
  echo "Source root must contain components/ (or legacy repos/)." >&2
  exit 2
fi

if [[ -z "$OUTPUT_ROOT" ]]; then
  OUTPUT_ROOT="$REPO_DIR/.local-releases"
fi

if [[ "$OUTPUT_ROOT" != /* ]]; then
  echo "--output-root must be an absolute path." >&2
  exit 2
fi

for component in clawpro hatchery teamai-cli orchestrator; do
  if [[ ! -d "$COMPONENT_ROOT/$component" ]]; then
    echo "Missing source component: $component" >&2
    exit 1
  fi
done

for command_name in node npm corepack go python3 tar shasum; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "Missing build dependency: $command_name" >&2
    exit 1
  fi
done

NODE_MAJOR="$(node -p 'Number(process.versions.node.split(".")[0])')"
if [[ "$NODE_MAJOR" -lt 22 ]]; then
  echo "Node.js 22 or newer is required." >&2
  exit 1
fi

CLAWPRO_DIR="$COMPONENT_ROOT/clawpro"
HATCHERY_DIR="$COMPONENT_ROOT/hatchery"
TEAMAI_DIR="$COMPONENT_ROOT/teamai-cli"
ORCHESTRATOR_DIR="$COMPONENT_ROOT/orchestrator"

if [[ ! -d "$CLAWPRO_DIR/node_modules" ]]; then
  (cd "$CLAWPRO_DIR" && npm ci)
fi
(cd "$CLAWPRO_DIR" && npm run build)

if [[ ! -d "$TEAMAI_DIR/node_modules" ]]; then
  (cd "$TEAMAI_DIR" && corepack pnpm install --frozen-lockfile)
fi
(cd "$TEAMAI_DIR" && corepack pnpm typecheck)
if [[ "$SKIP_TESTS" -eq 0 ]]; then
  (cd "$TEAMAI_DIR" && ./node_modules/.bin/vitest run src/__tests__/agent-task-executor.test.ts)
  (cd "$ORCHESTRATOR_DIR" && python3 -m unittest discover -p 'test_*.py')
  (cd "$HATCHERY_DIR" && go test ./controller ./model)
fi
(cd "$TEAMAI_DIR" && corepack pnpm build)

mkdir -p "$HATCHERY_DIR/build"
(cd "$HATCHERY_DIR" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags release -ldflags='-s -w' -o build/hatchery .)

STAMP="$(date '+%Y%m%d%H%M%S')"
BUNDLE_NAME="clawpro-multi-agent-deployment-kit-dev-$STAMP"
BUNDLE_DIR="$OUTPUT_ROOT/$BUNDLE_NAME"
ARCHIVE="$OUTPUT_ROOT/$BUNDLE_NAME.tar.gz"

if [[ -e "$BUNDLE_DIR" || -e "$ARCHIVE" ]]; then
  echo "Refusing to overwrite an existing local release: $BUNDLE_NAME" >&2
  exit 1
fi

mkdir -p "$BUNDLE_DIR" "$BUNDLE_DIR/source-info"
cp "$REPO_DIR/packaging/DEPLOYMENT_README.md" "$BUNDLE_DIR/README.md"
cp "$REPO_DIR/packaging/SECURITY.md" "$BUNDLE_DIR/SECURITY.md"
cp -a "$REPO_DIR/packaging/server" "$BUNDLE_DIR/server"
mkdir -p "$BUNDLE_DIR/server/frontend" "$BUNDLE_DIR/server/bin" "$BUNDLE_DIR/server/orchestrator"
cp -a "$CLAWPRO_DIR/dist/public/." "$BUNDLE_DIR/server/frontend/"
cp "$HATCHERY_DIR/build/hatchery" "$BUNDLE_DIR/server/bin/hatchery-linux-amd64"
cp -a "$ORCHESTRATOR_DIR/." "$BUNDLE_DIR/server/orchestrator/"
cp -a "$REPO_DIR/packaging/client" "$BUNDLE_DIR/client"
(cd "$TEAMAI_DIR" && npm pack --ignore-scripts --pack-destination "$BUNDLE_DIR/client/teamai")
if [[ -f "$SOURCE_ROOT/SOURCE_STATE.md" ]]; then
  cp "$SOURCE_ROOT/SOURCE_STATE.md" "$BUNDLE_DIR/source-info/SOURCE_STATE.md"
else
  printf '# Source state\n\nBuilt from %s\n' "$SOURCE_ROOT" > "$BUNDLE_DIR/source-info/SOURCE_STATE.md"
fi

chmod 0755 \
  "$BUNDLE_DIR/server/bin/hatchery-linux-amd64" \
  "$BUNDLE_DIR/server/install-server.sh" \
  "$BUNDLE_DIR/server/healthcheck.sh" \
  "$BUNDLE_DIR/client/teamai/"*.sh

{
  printf '{\n'
  printf '  "name": "clawpro-multi-agent-deployment-kit",\n'
  printf '  "version": "dev-%s",\n' "$STAMP"
  printf '  "target": "linux-amd64",\n'
  printf '  "source_layout": "components",\n'
  printf '  "source_root": "%s"\n' "$(basename "$SOURCE_ROOT")"
  printf '}\n'
} > "$BUNDLE_DIR/VERSION.json"

(
  cd "$BUNDLE_DIR"
  find . -type f ! -name SHA256SUMS -print0 | LC_ALL=C sort -z | xargs -0 shasum -a 256 > SHA256SUMS
)

mkdir -p "$OUTPUT_ROOT"
tar -czf "$ARCHIVE" -C "$OUTPUT_ROOT" "$BUNDLE_NAME"
(
  cd "$OUTPUT_ROOT"
  shasum -a 256 "$(basename "$ARCHIVE")" > "$(basename "$ARCHIVE").sha256"
  shasum -a 256 -c "$(basename "$ARCHIVE").sha256"
)

echo "ARCHIVE=$ARCHIVE"
echo "CHECKSUM=$ARCHIVE.sha256"
