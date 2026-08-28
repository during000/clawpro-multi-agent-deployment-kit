#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PACK_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ROOT="$(cd "$PACK_ROOT/../../.." && pwd)"
DATE_TAG="$(date +%Y-%m-%d)"
OUTPUT_DIR="${1:-$PROJECT_ROOT/dist}"
ZIP_NAME="clawpro-portable-design-skill-${DATE_TAG}.zip"

mkdir -p "$OUTPUT_DIR"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

cp -R "$PACK_ROOT" "$TMP_DIR/clawpro-portable-design-skill"

cd "$TMP_DIR"
zip -qr "$OUTPUT_DIR/$ZIP_NAME" clawpro-portable-design-skill

echo "$OUTPUT_DIR/$ZIP_NAME"

