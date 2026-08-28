#!/usr/bin/env bash
set -euo pipefail

ARCHIVE_PATH="/tmp/agent-export.tgz"
CURRENT_STEP="initializing"
TAR_DIAG=""
VERIFY_DIAG=""
BUSINESS_MANIFEST_BEFORE=""
BUSINESS_MANIFEST_AFTER=""
RESP=""

[ -d "$AGENT_DIR" ] || { echo "✗ agent 目录不存在: $AGENT_DIR"; exit 1; }

LOG_DIR="$AGENT_DIR/logs"
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/migration_export_$(date '+%Y%m%d_%H%M%S')_$$.log"
touch "$LOG_FILE"
chmod 600 "$LOG_FILE"
exec > >(tee -a "$LOG_FILE") 2>&1

finish_export() {
  local status=$?
  trap - EXIT
  [ -z "${TAR_DIAG:-}" ] || rm -f "$TAR_DIAG"
  [ -z "${VERIFY_DIAG:-}" ] || rm -f "$VERIFY_DIAG"
  [ -z "${BUSINESS_MANIFEST_BEFORE:-}" ] || rm -f "$BUSINESS_MANIFEST_BEFORE"
  [ -z "${BUSINESS_MANIFEST_AFTER:-}" ] || rm -f "$BUSINESS_MANIFEST_AFTER"
  [ -z "${RESP:-}" ] || rm -f "$RESP"
  if [ "$status" -ne 0 ]; then
    rm -f "$ARCHIVE_PATH"
    echo "✗ 迁移导出失败（阶段: $CURRENT_STEP，退出码: $status）"
  fi
  echo "日志文件: $LOG_FILE"
  exit "$status"
}
trap finish_export EXIT

# 对未排除的业务树生成稳定指纹。目录记录路径和权限，文件/符号链接额外记录
# 大小、mtime、ctime、inode 和链接目标，用于识别业务数据的新增、删除、修改或替换。
# 目录不记录 mtime/ctime/size/inode，避免已排除的运行时子项改变其父目录元数据后误报。
build_included_business_manifest() {
  local output_file="$1"
  AGENT_DIR="$AGENT_DIR" EXCLUDED_PATHS_B64="$EXCLUDED_PATHS_B64" \
    python3 - "$output_file" <<'PY'
import base64
import json
import os
import stat
import sys

root = os.environ["AGENT_DIR"]
excluded = json.loads(base64.b64decode(os.environ["EXCLUDED_PATHS_B64"]))
excluded = {path.strip("/") for path in excluded if path.strip("/")}

def is_excluded(relative_path):
    parts = relative_path.split("/")
    for pattern in excluded:
        if "/" in pattern:
            if relative_path == pattern or relative_path.startswith(pattern + "/"):
                return True
        elif pattern in parts:
            return True
    return False

rows = []

def scan(directory, relative_directory=""):
    with os.scandir(directory) as entries:
        for entry in entries:
            relative_path = (
                entry.name
                if not relative_directory
                else relative_directory + "/" + entry.name
            )
            if is_excluded(relative_path):
                continue
            info = entry.stat(follow_symlinks=False)
            file_type = stat.S_IFMT(info.st_mode)
            mode = stat.S_IMODE(info.st_mode)
            if stat.S_ISDIR(info.st_mode):
                rows.append([relative_path, file_type, mode])
                scan(entry.path, relative_path)
            else:
                link_target = os.readlink(entry.path) if stat.S_ISLNK(info.st_mode) else ""
                rows.append([
                    relative_path,
                    file_type,
                    mode,
                    info.st_size,
                    info.st_mtime_ns,
                    info.st_ctime_ns,
                    info.st_ino,
                    link_target,
                ])

scan(root)
rows.sort(key=lambda row: row[0])
with open(sys.argv[1], "w", encoding="utf-8") as output:
    json.dump(rows, output, ensure_ascii=True, separators=(",", ":"))
PY
}

# GNU tar 在归档期间若仅有已排除的日志/锁文件被创建或删除，仍可能因为
# agent 根目录元数据变化而返回 1。此兼容仅对 Hermes 开启，并只接受这一
# 条精确诊断；普通文件变化、权限/缺失或 fatal 错误仍必须失败。
is_only_agent_root_changed_warning() {
  local tar_status="$1"
  local diag_file="$2"
  local expected
  local line
  local matched=0

  expected="tar: $(basename "$AGENT_DIR"): file changed as we read it"
  [ "${ALLOW_AGENT_ROOT_CHANGE_WARNING:-0}" = "1" ] || return 1
  [ "$tar_status" -eq 1 ] || return 1
  while IFS= read -r line || [ -n "$line" ]; do
    [ -z "$line" ] && continue
    [ "$line" = "$expected" ] || return 1
    matched=1
  done < "$diag_file"
  [ "$matched" -eq 1 ]
}

echo "========== 迁移导出开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "Agent 目录: $AGENT_DIR"
echo "日志文件: $LOG_FILE"

CURRENT_STEP="packing"
TAR_DIAG=$(mktemp /tmp/agent-export-tar.XXXXXX)
if [ "${ALLOW_AGENT_ROOT_CHANGE_WARNING:-0}" = "1" ]; then
  BUSINESS_MANIFEST_BEFORE=$(mktemp /tmp/agent-export-business-before.XXXXXX)
  BUSINESS_MANIFEST_AFTER=$(mktemp /tmp/agent-export-business-after.XXXXXX)
  build_included_business_manifest "$BUSINESS_MANIFEST_BEFORE"
fi
rm -f "$ARCHIVE_PATH"
echo ">>> 打包 agent 目录..."
export LC_ALL=C
if command -v zstd >/dev/null 2>&1; then
  eval tar --use-compress-program="'zstd -T0 -1'" -cf "$ARCHIVE_PATH" \
    "$EXCLUDE_ARGS" \
    -C "$(dirname "$AGENT_DIR")" "$(basename "$AGENT_DIR")" \
    2>"$TAR_DIAG" &
else
  eval tar --use-compress-program="'gzip -1'" -cf "$ARCHIVE_PATH" \
    "$EXCLUDE_ARGS" \
    -C "$(dirname "$AGENT_DIR")" "$(basename "$AGENT_DIR")" \
    2>"$TAR_DIAG" &
fi
TAR_PID=$!
while kill -0 "$TAR_PID" 2>/dev/null; do
  sleep 5
  [ -f "$ARCHIVE_PATH" ] && echo ">>> 打包中... $(du -sh "$ARCHIVE_PATH" 2>/dev/null | cut -f1) 已写入"
done
TAR_STATUS=0
wait "$TAR_PID" || TAR_STATUS=$?
[ ! -s "$TAR_DIAG" ] || cat "$TAR_DIAG"
if [ "$TAR_STATUS" -ne 0 ]; then
  if is_only_agent_root_changed_warning "$TAR_STATUS" "$TAR_DIAG"; then
    build_included_business_manifest "$BUSINESS_MANIFEST_AFTER"
    if ! cmp -s "$BUSINESS_MANIFEST_BEFORE" "$BUSINESS_MANIFEST_AFTER"; then
      echo "✗ Hermes 业务数据在打包期间发生变化，拒绝上传可能不一致的归档"
      exit 1
    fi
    echo "⚠ Hermes 根目录在打包期间发生元数据变化；业务树指纹未变化，将在验包通过后继续"
  else
    echo "✗ 打包失败（tar 退出码: $TAR_STATUS）"
    exit "$TAR_STATUS"
  fi
fi

CURRENT_STEP="validating_archive"
VERIFY_DIAG=$(mktemp /tmp/agent-export-verify.XXXXXX)
echo ">>> 校验迁移归档..."
if ! tar tf "$ARCHIVE_PATH" >/dev/null 2>"$VERIFY_DIAG"; then
  echo "✗ 迁移归档校验失败"
  [ ! -s "$VERIFY_DIAG" ] || tail -n 20 "$VERIFY_DIAG"
  exit 1
fi
echo "✓ 打包完成且归档校验通过 ($(du -sh "$ARCHIVE_PATH" | cut -f1))"

CURRENT_STEP="preparing_upload"
HEADERS=()
while IFS= read -r header_line; do
  HEADERS+=("$header_line")
done < <(printf '%s' "$PART_HEADERS_B64" | base64 -d | python3 -c "import json,sys; d=json.load(sys.stdin); [print(f'-H\n{k}: {v}') for k,v in d.items()]" 2>/dev/null || true)
FILE_SIZE=$(stat -c%s "$ARCHIVE_PATH" 2>/dev/null || stat -f%z "$ARCHIVE_PATH")
PART_SIZE=$((50 * 1024 * 1024))
TOTAL_PARTS=$(( (FILE_SIZE + PART_SIZE - 1) / PART_SIZE ))
[ "$TOTAL_PARTS" -lt 1 ] && TOTAL_PARTS=1

CURRENT_STEP="uploading"
echo ">>> 开始分块上传，共 $TOTAL_PARTS 块..."
UPLOADED_BYTES=0
for (( i=1; i<=TOTAL_PARTS; i++ )); do
  PART_URL="${PART_URL_TEMPLATE//\{partNumber\}/$i}"
  OFFSET=$(( (i - 1) * PART_SIZE ))
  REMAINING=$(( FILE_SIZE - OFFSET ))
  ACTUAL=$(( REMAINING < PART_SIZE ? REMAINING : PART_SIZE ))
  RESP=$(mktemp)
  CODE=$(dd if="$ARCHIVE_PATH" iflag=skip_bytes,count_bytes skip=$OFFSET count=$ACTUAL bs=4M 2>/dev/null \
    | curl -sS -o "$RESP" -w '%{http_code} %{speed_upload}' -X PUT -H "Content-Length: $ACTUAL" \
        ${HEADERS[@]+"${HEADERS[@]}"} --data-binary @- --retry 2 --retry-delay 3 "$PART_URL") || true
  HTTP_CODE=$(echo "$CODE" | awk '{print $1}')
  SPEED=$(echo "$CODE" | awk '{print $2}')
  if [ "${HTTP_CODE:-000}" -ge 200 ] && [ "${HTTP_CODE:-000}" -lt 300 ]; then
    UPLOADED_BYTES=$(( UPLOADED_BYTES + ACTUAL ))
    PERCENT=$(( UPLOADED_BYTES * 100 / FILE_SIZE ))
    SPEED_KB=$(awk "BEGIN {printf \"%.0f\", $SPEED/1024}")
    echo "✓ 分块 $i/$TOTAL_PARTS  已上传 ${PERCENT}%  速度 ${SPEED_KB} KB/s"
    rm -f "$RESP"; RESP=""
  else
    echo "✗ 上传分块 $i 失败 (HTTP $HTTP_CODE)"; cat "$RESP" || true
    rm -f "$RESP"; RESP=""
    exit 1
  fi
done

CURRENT_STEP="confirming_upload"
echo ">>> 确认上传..."
RESP=$(mktemp)
CODE=$(curl -sS -o "$RESP" -w '%{http_code}' -X POST \
  "$SMH_ENDPOINT/api/v1/file/$LIBRARY_ID/$SPACE_ID/$CONFIRM_KEY?confirm=1&conflict_resolution_strategy=overwrite&access_token=$ACCESS_TOKEN") || true
[ "${CODE:-000}" -ge 200 ] && [ "${CODE:-000}" -lt 300 ] || { echo "✗ 确认上传失败 (HTTP $CODE)"; cat "$RESP" || true; exit 1; }
rm -f "$RESP" "$ARCHIVE_PATH"; RESP=""

CURRENT_STEP="done"
echo -e "✓ 导出成功\n文件路径: $FILE_KEY"
echo "========== 迁移导出完成: $(date '+%Y-%m-%d %H:%M:%S') =========="
