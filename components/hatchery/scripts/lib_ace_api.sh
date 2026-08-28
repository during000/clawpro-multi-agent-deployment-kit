#!/bin/bash
#
# lib_ace_api.sh
#
# 作用：LightClaw-ACE 本地 FastAPI 端口/主机通用发现与探活。
# 被其他 *_ace.sh wrapper 以 `source lib_ace_api.sh` 方式引入。
#
# 对外导出两个命令：
#   resolve_ace_api_base         → stdout 打印成功的 base URL（如 http://127.0.0.1:8088）
#                                  失败时退出码非 0
#   ace_http POST /config/...    → 调 ACE API 的便捷 curl 包装
#
# 发现优先级（越靠前越权威）：
#   1. 环境变量  $LIGHTCLAW_API_HOST / $LIGHTCLAW_API_PORT  （运维/TAT 强制覆盖）
#   2. finnie 官方 python API：read_last_api()             （最稳）
#   3. jq 读 ~/.lightclaw/lightclaw.json 的 .lastApi.{host,port}
#   4. 兼容老 schema：.last_api.{host,port} 或顶层 .last_api_host/.last_api_port
#   5. 兜底 127.0.0.1:8088
#
# Host 规整：0.0.0.0 / 空 / :: → 127.0.0.1；IPv6 自动加 [ ]
# 探活：对候选 base 发 GET /config/openapi.json，收到任何 HTTP 响应即视为活

# shellcheck shell=bash

# ── 1. Host 规整 ──────────────────────────────────────────────────
_ace_normalize_host() {
    local h="${1:-}"
    case "$h" in
        ""|"0.0.0.0"|"::"|"*") echo "127.0.0.1" ;;
        *:*)                   echo "[$h]" ;;      # IPv6
        *)                     echo "$h" ;;
    esac
}

# ── 2. 候选源：返回 "host:port" 行 ────────────────────────────────
_ace_candidates() {
    local cfg="$HOME/.lightclaw/lightclaw.json"

    # 源 1：环境变量
    if [ -n "${LIGHTCLAW_API_HOST:-}${LIGHTCLAW_API_PORT:-}" ]; then
        echo "${LIGHTCLAW_API_HOST:-127.0.0.1}:${LIGHTCLAW_API_PORT:-8088}"
    fi

    # 源 2：finnie 官方 python API
    if command -v python3 >/dev/null 2>&1; then
        local py_out
        py_out="$(python3 - <<'PY' 2>/dev/null
try:
    from lightclaw.config.utils import read_last_api
    r = read_last_api()
    if r:
        print(f"{r[0]}:{r[1]}")
except Exception:
    pass
PY
        )"
        [ -n "$py_out" ] && echo "$py_out"
    fi

    # 源 3 & 4：jq 读配置文件
    if [ -f "$cfg" ] && command -v jq >/dev/null 2>&1; then
        jq -r '
          # new schema
          (.lastApi.host // .last_api.host // .last_api_host // "127.0.0.1") as $h
          | (.lastApi.port // .last_api.port // .last_api_port // 8088) as $p
          | "\($h):\($p)"
        ' "$cfg" 2>/dev/null
    fi

    # 源 5：兜底
    echo "127.0.0.1:8088"
}

# ── 3. 探活 ───────────────────────────────────────────────────────
_ace_probe() {
    # $1 = base URL；收到任何 HTTP 响应即判活（排除 000 连接失败）
    local base="$1"
    local code
    code="$(curl -sS -o /dev/null -m 3 --connect-timeout 2 \
            -w '%{http_code}' "${base}/config/openapi.json" 2>/dev/null || echo 000)"
    [ "$code" != "000" ] && [ -n "$code" ]
}

# ── 4. 对外：解析 base URL ────────────────────────────────────────
resolve_ace_api_base() {
    local seen=""
    while IFS= read -r hp; do
        [ -z "$hp" ] && continue
        local host="${hp%:*}"
        local port="${hp##*:}"
        host="$(_ace_normalize_host "$host")"

        # 端口合法性：先确保是纯数字，再校验范围 1~65535
        case "$port" in
            ''|*[!0-9]*) continue ;;
        esac
        if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
            continue
        fi

        local base="http://${host}:${port}"
        # 去重
        case " $seen " in *" $base "*) continue ;; esac
        seen="$seen $base"

        if _ace_probe "$base"; then
            echo "$base"
            return 0
        fi
    done < <(_ace_candidates)

    return 1
}

# ── 5. 对外：HTTP 调用便捷包装 ────────────────────────────────────
# 用法：ace_http POST /config/channels/weixin/qrcode/generate  [curl 额外参数...]
#   默认 Content-Type: application/json，用 -d 传 body
ace_http() {
    local method="$1"; shift
    local path="$1";   shift
    local base
    base="$(resolve_ace_api_base)" || {
        echo "resolve_ace_api_base failed" >&2
        return 1
    }
    curl -sS -X "$method" "${base}${path}" \
        -H 'Content-Type: application/json' \
        --connect-timeout 5 --max-time 45 \
        "$@"
}
