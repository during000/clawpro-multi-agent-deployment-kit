#!/bin/bash
# approve_device.sh —— 绕过 RPC 鉴权，直接给指定 device 的 operator token 写入 5 件套权限
#
# 主流程：
#   step 0: 兜底执行 `openclaw devices list`，让 gateway 在 loopback silent 路径下把
#           paired.json 落到磁盘（5.7 全新实例首次握手会触发 auto-approve）。
#   step 1: 停止 gateway，独占 paired.json，避免在线时 verifyDeviceToken 反向覆盖。
#   step 2: 备份 + 用 jq 原地修补 paired.json（scopes/approvedScopes/tokens.operator）。
#   step 3: 清理 pending.json 中该 device 的残留 pending 请求。
#   step 4: 启动 gateway，等待 active。
#   step 5: 校验最终 scopes。
#
# 上游契约（hatchery controller / TAT 依赖）：
#   stdout 最后一行：成功时为 `ok`，失败时为 `<错误消息>` 并 exit !=0。
#   所有诊断日志通过 log() 写到 stderr，再被 tee 复制到 ~/.openclaw/logs/approve_device.log，
#   避免污染上游对 stdout 的解析。
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化（对齐 list_skills.sh 风格） ==========
# 使用用户家目录，避免非 root 运行时无法写入 /var/log
LOG_DIR="${HOME}/.openclaw/logs"
mkdir -p "$LOG_DIR" 2>/dev/null || true
LOG_FILE="$LOG_DIR/approve_device.log"

# 关键：把整个进程的 stderr 永久重定向（追加）到日志文件，
# 这样后续所有 `>&2` 输出都不会污染 TAT 抓取的 stdout。
# 若日志文件不可写则降级为不重定向，避免 exec 失败导致脚本崩溃。
if { : >> "$LOG_FILE"; } 2>/dev/null; then
    exec 2>>"$LOG_FILE"
fi

echo "" >&2
echo "========== $(date '+%Y-%m-%d %H:%M:%S') approve_device 开始 ==========" >&2
trap 'echo "========== $(date "+%Y-%m-%d %H:%M:%S") approve_device 结束 ==========" >&2' EXIT

LOG_PREFIX="[approve-device]"

# 日志辅助函数：统一时间戳前缀，仅写 stderr（已被重定向到日志文件）
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ${LOG_PREFIX} $*" >&2
}

# ========== 参数与路径 ==========
DEVICE_ID="${1:-}"   # 不传就取 paired.json 里第一个 deviceId
DEVICES_DIR="$HOME/.openclaw/devices"
PAIRED_JSON="$DEVICES_DIR/paired.json"
PENDING_JSON="$DEVICES_DIR/pending.json"
PRIME_LIST_TIMEOUT=10        # step 0 兜底 devices list 单次超时(秒)
PRIME_POLL_INTERVAL=2       # step 0 轮询间隔(秒)
PRIME_POLL_TOTAL=60         # step 0 轮询总时长(秒，1 分钟)

TARGET='["operator.admin","operator.pairing","operator.read"]'

log "=== OpenClaw 设备审批脚本启动 ==="
log "运行用户: $(id -un) (uid=$(id -u))"
log "HOME=${HOME}"
log "DEVICES_DIR=${DEVICES_DIR}"
log "PAIRED_JSON=${PAIRED_JSON}"
log "PENDING_JSON=${PENDING_JSON}"
log "PRIME_LIST_TIMEOUT=${PRIME_LIST_TIMEOUT}s, PRIME_POLL_INTERVAL=${PRIME_POLL_INTERVAL}s, PRIME_POLL_TOTAL=${PRIME_POLL_TOTAL}s"
log "DEVICE_ID(入参)=${DEVICE_ID:-<未指定，将自动选 paired[0]>}"
log "TARGET_SCOPES=${TARGET}"

# ========== 前置依赖检查 ==========
command -v jq >/dev/null || { log "✗ jq 不可用，无法继续"; echo "jq required"; exit 1; }
log "依赖检查通过 (jq 可用)"

# ───────────────────────────────────────────────────────
# step 0（前置）: 认证架构检测 —— OpenClaw 2026.7.1+ 采用 gateway "token" 认证
#   现网实测（2026.7.1）结论：
#     - 当 openclaw 版本 >= 2026.7.1 且 gateway.auth.mode == "token" 时，hatchery 后端
#       RPC 与用户 webchat 均通过 gateway.auth.token 认证并获得完整权限，
#       【不走 device-pairing、不依赖 paired.json】。
#     - 该架构下 gateway 不再为 loopback 的 `openclaw devices list` 落盘 paired.json
#       （5.7 的 silent auto-approve 落盘行为已移除）。本脚本后续 step 0~5 的 paired.json
#       裸改流程既无对象可改也无必要，只会白等 PRIME_POLL_TOTAL 秒后在 step 1 报
#       "no paired.json" 失败（clawpro bug：openclaw7.1 执行 approve_device.sh 失败）。
#   处理：命中"2026.7.1+ 且 token 认证"时，判定无需设备审批，按 stdout 契约输出 ok 后成功退出。
#   兜底：版本 < 2026.7.1 / 非 token 模式 / 无法探测版本时，一律保持原有 5.x
#         device-pairing 审批流程（step 0~5）不变，确保对存量 5.x 实例零影响。
# ───────────────────────────────────────────────────────
# 语义化版本比较：判断 $1 >= $2（点分数字，兼容 YYYY.M.D 与 semver）
version_ge() {
    [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V 2>/dev/null | head -n1)" = "$2" ]
}

OPENCLAW_CONFIG="$HOME/.openclaw/openclaw.json"
OC_VERSION=""
if command -v openclaw >/dev/null 2>&1; then
    OC_VERSION=$(openclaw --version 2>/dev/null | grep -oE '[0-9]{4}\.[0-9]+\.[0-9]+' | head -n1 || true)
fi
AUTH_MODE=""
if [ -f "$OPENCLAW_CONFIG" ]; then
    AUTH_MODE=$(jq -r '.gateway.auth.mode // ""' "$OPENCLAW_CONFIG" 2>/dev/null || echo "")
fi
log "认证架构探测: openclaw_version=${OC_VERSION:-<未知>} gateway.auth.mode=${AUTH_MODE:-<空>}"

if [ -n "$OC_VERSION" ] && version_ge "$OC_VERSION" "2026.7.1" && [ "$AUTH_MODE" = "token" ]; then
    log "✓ 命中 2026.7.1+ token 认证架构：设备鉴权由 gateway token 统一处理，无需 device-pairing 审批，跳过 step 0~5"
    # === stdout 契约（与 step 5 / step 1.5 保持一致）：最后一行为 ok ===
    echo "ok"
    log "=== 终态: ok (token-auth, skip device approve) openclaw_version=${OC_VERSION} ==="
    exit 0
fi
log "未命中 token 认证快速路径（version=${OC_VERSION:-<未知>}, auth_mode=${AUTH_MODE:-<空>}），进入原有 device-pairing 审批流程"

# ───────────────────────────────────────────────────────
# step 0: 兜底执行 `openclaw devices list`，触发 gateway 握手
#   目的：在全新实例上，loopback 的握手会走 OpenClaw 内部 silent
#         auto-approve 路径，把 paired.json / pending.json 写到磁盘，
#         避免下一步 `[ -f $PAIRED_JSON ]` 检查直接失败。
#   策略：轮询 PRIME_POLL_TOTAL 秒（默认 1 分钟），每 PRIME_POLL_INTERVAL
#         秒触发一次 `openclaw devices list`，单次 list 仍受 PRIME_LIST_TIMEOUT
#         保护；任意一次让 paired.json 落盘成功即立刻跳出，进入 step 1。
#         这样即使上游忘了等 openclaw-gateway 就绪，本脚本也能自愈。
#   注意：list 自身可能因 5.7 churn 而失败/超时，本步骤不阻断主流程，
#         真正的存在性判定仍由 step 1 的 [ -f ] 兜底。
# ───────────────────────────────────────────────────────
log ">>> [step 0/5] 兜底触发 \`openclaw devices list\`（轮询），确保 paired.json 落盘"
mkdir -p "$DEVICES_DIR" 2>/dev/null || true

if [ -f "$PAIRED_JSON" ]; then
    log "[prime] paired.json 已存在 (size=$(wc -c <"$PAIRED_JSON" 2>/dev/null || echo ?) bytes)，跳过 prime list"
elif ! command -v openclaw >/dev/null 2>&1; then
    log "[prime] openclaw CLI 不可用，跳过 prime list（依赖后续 step 1 报错）"
else
    log "[prime] paired.json 不存在，开始轮询 prime list (单次 timeout=${PRIME_LIST_TIMEOUT}s, 间隔=${PRIME_POLL_INTERVAL}s, 总计=${PRIME_POLL_TOTAL}s)"
    prime_deadline=$(( $(date +%s) + PRIME_POLL_TOTAL ))
    prime_attempt=0
    prime_ok=0
    while :; do
        prime_attempt=$((prime_attempt + 1))
        prime_out=""
        prime_rc=0
        if command -v timeout >/dev/null 2>&1; then
            prime_out=$(timeout "${PRIME_LIST_TIMEOUT}s" openclaw devices list --json 2>&1) || prime_rc=$?
        else
            prime_out=$(openclaw devices list --json 2>&1) || prime_rc=$?
        fi
        log "[prime] attempt=${prime_attempt} 返回码: rc=${prime_rc}"
        # 限制日志条数，避免输出过长污染日志
        printf '%s\n' "$prime_out" | head -n 10 | sed "s|^|${LOG_PREFIX} [prime]   |" >&2
        case "$prime_rc" in
            0)   log "[prime] ✓ attempt=${prime_attempt} list 成功" ;;
            124) log "[prime] ⚠ attempt=${prime_attempt} list 超时 (${PRIME_LIST_TIMEOUT}s)" ;;
            *)   log "[prime] ⚠ attempt=${prime_attempt} list 失败 (rc=${prime_rc})" ;;
        esac

        if [ -f "$PAIRED_JSON" ]; then
            log "[prime] ✓ paired.json 已落盘 (attempt=${prime_attempt}, size=$(wc -c <"$PAIRED_JSON" 2>/dev/null || echo ?) bytes)"
            prime_ok=1
            break
        fi

        now_ts=$(date +%s)
        if [ "$now_ts" -ge "$prime_deadline" ]; then
            log "[prime] ⚠ 轮询总时长已达 ${PRIME_POLL_TOTAL}s 仍无 paired.json，继续主流程（step 1 将报错退出）"
            break
        fi
        remaining=$((prime_deadline - now_ts))
        log "[prime] paired.json 仍未出现，sleep ${PRIME_POLL_INTERVAL}s 后重试 (剩余 ${remaining}s)"
        sleep "$PRIME_POLL_INTERVAL"
    done
    log "[prime] 落盘探测: paired.json=$([ -f "$PAIRED_JSON" ] && echo yes || echo no), pending.json=$([ -f "$PENDING_JSON" ] && echo yes || echo no), prime_ok=${prime_ok}"
fi

# ───────────────────────────────────────────────────────
# step 1: 校验 paired.json 存在 + 解析 deviceId
# ───────────────────────────────────────────────────────
log ">>> [step 1/5] 校验 paired.json 存在 + 解析目标 deviceId"
if [ ! -f "$PAIRED_JSON" ]; then
    log "✗ paired.json 仍不存在: ${PAIRED_JSON}"
    echo "no paired.json"
    exit 1
fi
log "paired.json 存在 (size=$(wc -c <"$PAIRED_JSON" 2>/dev/null || echo ?) bytes)"

# 自动定位 deviceId（建议显式传）
if [ -z "$DEVICE_ID" ]; then
    DEVICE_ID=$(jq -r 'to_entries | .[0].key // empty' "$PAIRED_JSON")
    log "未传 DEVICE_ID，自动选取 paired[0]: ${DEVICE_ID:-<空>}"
fi
if [ -z "$DEVICE_ID" ]; then
    log "✗ paired.json 中无任何 device"
    echo "no deviceId"
    exit 1
fi
if ! jq -e --arg d "$DEVICE_ID" 'has($d)' "$PAIRED_JSON" >/dev/null; then
    log "✗ device ${DEVICE_ID} 不在 paired.json 中"
    echo "device $DEVICE_ID not found in paired.json"
    exit 1
fi
log "✓ 目标 device: ${DEVICE_ID}"

# ───────────────────────────────────────────────────────
# step 1.5: 幂等快速跳过 —— 若 operator.admin 已经存在，直接返回 ok
#   背景：approve_device.sh 会被多个入口反复触发（创建/重装/升级/取 token），
#         若每次都走 stop → patch → start，会反复打断 gateway 在线连接，造成无谓抖动。
#         这里在持久化前先读一次磁盘上的 scopes，若已有 operator.admin 则直接命中 stdout
#         契约（最终 scopes JSON + "ok"）退出，跳过 step 2~5。
#   注意：判定条件用 operator.admin 单一权限即可——TARGET 三件套是同一批写入的，
#         只要 admin 在，pairing/read 必然也在；反之若被人为篡改丢失，后续 step 仍会补齐。
# ───────────────────────────────────────────────────────
log ">>> [step 1.5/5] 幂等检查：operator.admin 是否已经写入"
existing_scopes=$(jq -c --arg dev "$DEVICE_ID" '.[$dev].tokens.operator.scopes // []' "$PAIRED_JSON" 2>/dev/null || echo '[]')
log "[idempotent] 当前 tokens.operator.scopes: ${existing_scopes}"
if jq -e --arg dev "$DEVICE_ID" '
    (.[$dev].tokens.operator.scopes // []) | index("operator.admin") != null
' "$PAIRED_JSON" >/dev/null 2>&1; then
    log "[idempotent] ✓ operator.admin 已存在，跳过 step 2~5（不重启 gateway）"
    # === stdout 契约（与 step 5 保持一致）===
    # 第一行：最终 scopes 数组（JSON）
    # 第二行：ok
    echo "$existing_scopes"
    echo "ok"
    log "=== 终态: ok device=${DEVICE_ID} (idempotent skip) ==="
    exit 0
fi
log "[idempotent] operator.admin 不存在，继续执行授权流程"

NOW_MS=$(($(date +%s%3N 2>/dev/null || echo $(($(date +%s) * 1000))) + 0))
log "时间戳: now_ms=${NOW_MS}"

# ───────────────────────────────────────────────────────
# 探测 systemd user unit（轮询 + 超时）
#   背景：升级场景下 root 的 systemd --user 实例可能仍在初始化，
#         `systemctl --user list-unit-files` 可能短暂返回空，
#         单次探测易误判为 "service 未找到" 导致脚本提前退出。
#   策略：复用 PRIME_POLL_INTERVAL / PRIME_POLL_TOTAL 作为轮询节奏；
#         任何一次命中即认定 unit 存在，否则超时后再报错退出。
# ───────────────────────────────────────────────────────
UNIT=""
unit_probe_max_attempts=$(( PRIME_POLL_TOTAL / PRIME_POLL_INTERVAL ))
log "[unit-probe] 开始轮询探测 openclaw-gateway.service (interval=${PRIME_POLL_INTERVAL}s, timeout=${PRIME_POLL_TOTAL}s, max_attempts=${unit_probe_max_attempts})"
unit_probe_attempt=0
unit_probe_start=$(date +%s)
while [ "$unit_probe_attempt" -lt "$unit_probe_max_attempts" ]; do
    unit_probe_attempt=$(( unit_probe_attempt + 1 ))
    if systemctl --user list-unit-files openclaw-gateway.service 2>/dev/null \
        | grep -q '^openclaw-gateway\.service'; then
        UNIT="openclaw-gateway.service"
        unit_probe_elapsed=$(( $(date +%s) - unit_probe_start ))
        log "[unit-probe] ✓ 探测到 systemd user unit: ${UNIT} (attempt=${unit_probe_attempt}/${unit_probe_max_attempts}, elapsed=${unit_probe_elapsed}s)"
        break
    fi
    if [ "$unit_probe_attempt" -lt "$unit_probe_max_attempts" ]; then
        log "[unit-probe] attempt ${unit_probe_attempt}/${unit_probe_max_attempts} 未命中，${PRIME_POLL_INTERVAL}s 后重试"
        sleep "$PRIME_POLL_INTERVAL"
    fi
done
if [ -z "$UNIT" ]; then
    unit_probe_elapsed=$(( $(date +%s) - unit_probe_start ))
    log "✗ openclaw-gateway.service 未找到 (轮询 ${unit_probe_attempt} 次，耗时 ${unit_probe_elapsed}s，超过 ${PRIME_POLL_TOTAL}s 阈值)"
    echo "openclaw-gateway.service not found"
    exit 1
fi

# ───────────────────────────────────────────────────────
# step 2: 停 gateway → 备份 → 原子修补 paired.json
# ───────────────────────────────────────────────────────
log ">>> [step 2/5] 停止 gateway 并原子修补 paired.json"
log "[stop] systemctl --user stop ${UNIT}"
systemctl --user stop "$UNIT"
for i in 1 2 3 4 5 6 7 8 9 10; do
    state=$(systemctl --user is-active "$UNIT" 2>/dev/null || true)
    if [ "$state" != "active" ]; then
        log "[stop] ✓ ${UNIT} 已停止 (state=${state}, 耗时 ${i}s)"
        break
    fi
    sleep 1
done

backup="$PAIRED_JSON.bak.$(date +%Y%m%d-%H%M%S)"
cp -p "$PAIRED_JSON" "$backup"
log "[backup] 已备份: ${backup}"

before_scopes=$(jq -c --arg dev "$DEVICE_ID" '.[$dev].tokens.operator.scopes // []' "$PAIRED_JSON" 2>/dev/null || echo '[]')
log "[before] 当前 tokens.operator.scopes: ${before_scopes}"

TMP="$PAIRED_JSON.tmp.$$"
jq --arg dev "$DEVICE_ID" \
   --argjson target "$TARGET" \
   --argjson now "$NOW_MS" '
  def merge_scopes(existing): (existing // []) + $target | unique;
  .[$dev].scopes              = merge_scopes(.[$dev].scopes)
  | .[$dev].approvedScopes    = merge_scopes(.[$dev].approvedScopes)
  | .[$dev].tokens.operator.scopes      = merge_scopes(.[$dev].tokens.operator.scopes)
  | .[$dev].tokens.operator.rotatedAtMs = $now
' "$PAIRED_JSON" > "$TMP"
log "[patch] jq 已生成新版本到 ${TMP}"

# schema 完整性校验：确保 operator token 的必需字段都在
if ! jq -e --arg dev "$DEVICE_ID" '
  .[$dev].tokens.operator
  | (.token != null and .role != null and (.scopes | type) == "array" and .createdAtMs != null)
' "$TMP" >/dev/null; then
    rm -f "$TMP"
    log "✗ operator token 完整性校验失败 (token/role/createdAtMs 缺失)，回滚并重启 gateway"
    echo "operator token incomplete (token/role/createdAtMs missing); abort"
    systemctl --user start "$UNIT" 2>&1 | sed "s|^|${LOG_PREFIX} [rollback-start]   |" >&2 || true
    exit 1
fi
log "[verify] ✓ 修补后 operator token schema 完整"

mv "$TMP" "$PAIRED_JSON"
log "[write] ✓ 已原子替换 paired.json"

# ───────────────────────────────────────────────────────
# step 3: 清理 pending.json 中该 device 的所有残留
# ───────────────────────────────────────────────────────
log ">>> [step 3/5] 清理 pending.json 中 device=${DEVICE_ID} 的残留 pending"
if [ -f "$PENDING_JSON" ]; then
    pending_before=$(jq -r --arg dev "$DEVICE_ID" '
      [to_entries[] | select(.value.deviceId == $dev)] | length
    ' "$PENDING_JSON" 2>/dev/null || echo "?")
    log "[pending] 清理前匹配 device 的 pending 数量: ${pending_before}"
    TMP2="$PENDING_JSON.tmp.$$"
    if jq --arg dev "$DEVICE_ID" 'with_entries(select(.value.deviceId != $dev))' \
        "$PENDING_JSON" > "$TMP2" 2>/dev/null; then
        mv "$TMP2" "$PENDING_JSON"
        log "[pending] ✓ 已清理"
    else
        rm -f "$TMP2"
        log "[pending] ⚠ jq 清理失败或 pending.json 为空，跳过"
    fi
else
    log "[pending] pending.json 不存在，跳过清理"
fi

# ───────────────────────────────────────────────────────
# step 4: 启动 gateway，让它从磁盘重新加载 paired.json
# ───────────────────────────────────────────────────────
log ">>> [step 4/5] 启动 gateway 并等待 active"
log "[start] systemctl --user restart ${UNIT}"
systemctl --user start "$UNIT"
gateway_active=0
for i in $(seq 1 20); do
    state=$(systemctl --user is-active "$UNIT" 2>/dev/null || true)
    if [ "$state" = "active" ]; then
        log "[start] ✓ ${UNIT} 已 active (耗时 ${i}s)"
        gateway_active=1
        break
    fi
    sleep 1
done
if [ "$gateway_active" -ne 1 ]; then
    log "[start] ⚠ ${UNIT} 在 20s 内未回到 active (当前 state=$(systemctl --user is-active "$UNIT" 2>/dev/null || true))"
fi

# ───────────────────────────────────────────────────────
# step 5: 校验最终 scopes（输出到 stdout 供上游解析）
# ───────────────────────────────────────────────────────
log ">>> [step 5/5] 校验最终 scopes"
after_scopes=$(jq -c --arg dev "$DEVICE_ID" '.[$dev].tokens.operator.scopes' "$PAIRED_JSON")
log "[after] 最终 tokens.operator.scopes: ${after_scopes}"
log "[diff]  before=${before_scopes}"
log "[diff]  after =${after_scopes}"

# === stdout 契约 ===
# 第一行：最终 scopes 数组（JSON）
# 第二行：ok
echo "$after_scopes"
echo "ok"

log "=== 终态: ok device=${DEVICE_ID} ==="
exit 0
