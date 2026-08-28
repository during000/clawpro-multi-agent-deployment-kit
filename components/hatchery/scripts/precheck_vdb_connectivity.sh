#!/bin/bash
# precheck_vdb_connectivity.sh — Pro 切换前预检 CVM 与 VDB 的真实业务可达性
#
# 用途：
#   - handleSwitchToPro 在 allocate_database + persist_binding 后调用本脚本，
#     不通时返回非 0 → hatchery 判 NonRetryableError 直接终态 + 自动 rollback mem space
#   - handleSwitchToOff 在 disable 前调用本脚本，
#     不通时透传 skip_export=true 给 disable 脚本，避免 export 必然超时阻断 OFF 流程
#
# 探测策略：模拟真实业务请求
#   POST ${url}/database/list
#     Header: Authorization: Bearer account=<username>&api_key=<apiKey>
#     Body  : {}
#   通：HTTP 200（VDB 服务在线 + 鉴权信道工作）
#   不通：http_code=000（含 RST/超时/连接拒绝/无鉴权被 nginx 直接拒）
#         或 5xx（VDB 服务异常）
#
# 为什么必须带 auth：
#   腾讯云 VDB 网关对任何没带正确 Authorization 头的请求**直接 RST**
#   （GET / 、 POST 无 auth 、错 token 都立即 close 连接）
#   导致 curl 拿到 http_code=000，无法与"网络真不通"区分。
#   只有带正确鉴权才能拿到真实 HTTP 响应。
#
# 退出码：
#   0  = 通（HTTP 200，VDB 服务可用）
#   1  = 不通（含连接被拒/超时/RST/5xx 等）
#   2  = 探测工具不可用（无 curl 且无 python3）—— hatchery 视为通过（保守）
#   3  = 参数非法（vdb_endpoint/username/api_key 缺失）
#
# 标准输出：单行 JSON 摘要供 hatchery 解析
#   {"reachable":true,"host":"10.0.0.5","port":80,"probe":"curl","http_code":"200"}
#   {"reachable":false,"host":"10.0.0.5","port":80,"probe":"curl","http_code":"000"}

set -uo pipefail

export PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$HOME/.local/bin:$HOME/.npm-global/bin:$PATH"
export NO_COLOR=1

vdb_endpoint="{{vdb_endpoint}}"
vdb_username="{{vdb_username}}"
vdb_api_key="{{vdb_api_key}}"
vdb_database="{{vdb_database}}"
timeout_sec="{{timeout_sec}}"

# 兜底：若 hatchery 未传 timeout_sec 或传了 "{{timeout_sec}}" 字面量
case "$timeout_sec" in
  ''|*[!0-9]*) timeout_sec=5 ;;
esac

log() { echo "[precheck-vdb] $*"; }

# ========== 解析 host:port（仅用于日志和 JSON 摘要）==========
parse_host_port() {
  local raw="$1"
  local stripped="${raw#http://}"
  stripped="${stripped#https://}"
  stripped="${stripped%%/*}"
  local host="${stripped%%:*}"
  local port="${stripped#*:}"
  if [ "$port" = "$stripped" ] || [ -z "$port" ]; then
    if [[ "$raw" == https://* ]]; then
      port="443"
    else
      port="80"
    fi
  fi
  echo "$host" "$port"
}

# ========== 探测：POST /database/list 带 Bearer 鉴权 ==========
# 用 curl，输出 http_code 到全局变量 $http_code
probe_curl() {
  local base_url="$1"
  local user="$2"
  local key="$3"
  local to="$4"
  local code
  code=$(curl -s -o /dev/null \
    --connect-timeout "$to" --max-time "$to" \
    -X POST \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer account=${user}&api_key=${key}" \
    -d '{}' \
    -w "%{http_code}" \
    "${base_url}/database/list" 2>/dev/null)
  http_code="$code"
  case "$code" in
    000|"") return 1 ;;          # 没拿到 HTTP 响应 = 不通
    2*|4*) return 0 ;;           # 200/4xx 都说明 VDB 服务在响应（401/403 也算通）
    *) return 1 ;;               # 5xx / 3xx 视为不通
  esac
}

# ========== 探测：python3 fallback ==========
probe_python() {
  local base_url="$1"
  local user="$2"
  local key="$3"
  local to="$4"
  local code
  code=$(python3 - "$base_url" "$user" "$key" "$to" 2>/dev/null <<'PY'
import sys, json, urllib.request, urllib.error
base, user, key, to = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4])
url = base.rstrip('/') + '/database/list'
req = urllib.request.Request(
    url,
    method="POST",
    data=b'{}',
    headers={
        "Content-Type": "application/json",
        "Authorization": f"Bearer account={user}&api_key={key}",
    },
)
try:
    with urllib.request.urlopen(req, timeout=to) as resp:
        print(resp.status)
except urllib.error.HTTPError as e:
    # 4xx 也说明服务在响应
    print(e.code)
except Exception:
    print("000")
PY
)
  http_code="$code"
  case "$code" in
    000|"") return 1 ;;
    2*|4*) return 0 ;;
    *) return 1 ;;
  esac
}

# ========== 主流程 ==========
# 参数校验
missing=""
[ -z "$vdb_endpoint" ] && missing="${missing} vdb_endpoint"
[ -z "$vdb_username" ] && missing="${missing} vdb_username"
[ -z "$vdb_api_key" ] && missing="${missing} vdb_api_key"
if [ -n "$missing" ]; then
  log "ERROR: 缺少参数:$missing"
  echo "{\"reachable\":false,\"error\":\"missing_params\",\"missing\":\"$missing\"}"
  exit 3
fi

# database 仅用于 collection/list 之类的进阶探测，本脚本只用 /database/list 不强制要

read -r host port < <(parse_host_port "$vdb_endpoint")
if [ -z "$host" ] || [ -z "$port" ]; then
  log "ERROR: 无法从 endpoint 解析 host/port: $vdb_endpoint"
  echo "{\"reachable\":false,\"error\":\"unparseable_endpoint\",\"endpoint\":\"$vdb_endpoint\"}"
  exit 3
fi

# 去掉 endpoint 末尾的 /
base_url="${vdb_endpoint%/}"

probe_used=""
http_code=""
rc=2

if command -v curl >/dev/null 2>&1; then
  probe_used="curl"
  log "probing VDB API: POST ${base_url}/database/list (timeout=${timeout_sec}s, host=${host}:${port})"
  probe_curl "$base_url" "$vdb_username" "$vdb_api_key" "$timeout_sec"
  rc=$?
elif command -v python3 >/dev/null 2>&1; then
  probe_used="python3"
  log "probing VDB API via python3: POST ${base_url}/database/list (timeout=${timeout_sec}s)"
  probe_python "$base_url" "$vdb_username" "$vdb_api_key" "$timeout_sec"
  rc=$?
else
  log "WARN: 无 curl 且无 python3，无法预检"
  echo "{\"reachable\":false,\"host\":\"$host\",\"port\":$port,\"probe\":\"none\",\"error\":\"no_probe_tool\"}"
  exit 2
fi

case "$rc" in
  0)
    log "OK: VDB API reachable (probe=$probe_used, http_code=$http_code)"
    echo "{\"reachable\":true,\"host\":\"$host\",\"port\":$port,\"probe\":\"$probe_used\",\"http_code\":\"$http_code\"}"
    exit 0
    ;;
  1)
    log "FAIL: VDB API NOT reachable (probe=$probe_used, http_code=$http_code)"
    echo "{\"reachable\":false,\"host\":\"$host\",\"port\":$port,\"probe\":\"$probe_used\",\"http_code\":\"$http_code\"}"
    exit 1
    ;;
  *)
    log "WARN: 探测异常 rc=$rc"
    echo "{\"reachable\":false,\"host\":\"$host\",\"port\":$port,\"probe\":\"$probe_used\",\"error\":\"unexpected_rc:$rc\"}"
    exit 2
    ;;
esac
