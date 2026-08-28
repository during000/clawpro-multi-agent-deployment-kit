#!/bin/bash
set -euo pipefail
export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)
GATEWAY_IP="{{gateway_ip}}"
GATEWAY_UI_PORT="{{gateway_ui_port}}"
CONFIG="$HOME/.openclaw/openclaw.json"
# 拼接 origin
ORIGIN="http://${GATEWAY_IP}:${GATEWAY_UI_PORT}"
# ========== 文件锁：避免并发重启服务触发 systemd start rate limit ==========
LOCK_FILE="/tmp/.openclaw_set_gateway_ui.lock"
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
    echo '{"error":"Gateway UI 配置操作正在进行中，请勿重复提交"}' >&1
    exit 1
fi

# ========== 日志系统初始化 ==========
# 重要：调用方（Go 后端 setOpenClawGatewayUI）会对 stdout 做 json.Unmarshal，
# 因此 stdout 必须保持为纯 JSON，不能混入任何日志文本。
# 做法：用 fd 3 保存原始 stdout 供最后输出 JSON 使用；
#       同时把脚本 stdout/stderr 全部重定向到日志文件，所有 echo 日志都只落盘，不进 stdout。
LOG_DIR="/var/log/clawpro"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR" 2>/dev/null || true
fi
SCRIPT_NAME="set_gateway_ui"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
# 用 fd 3 复制一份原始 stdout（供最后输出 JSON）
exec 3>&1
# 将脚本的 stdout/stderr 全部重定向到日志文件（追加）
# 若日志目录/文件不可写，则退回到 /dev/null，避免 exec 失败导致脚本整体崩溃
if { : >> "$LOG_FILE"; } 2>/dev/null; then
    exec >> "$LOG_FILE" 2>&1
else
    exec >/dev/null 2>&1
fi
# 脚本异常退出时向 fd 3（原始 stdout）输出错误 JSON，
# 避免 Go 侧 json.Unmarshal 收到空字符串报 unexpected end of JSON input。
#
# 设计要点：
# 1) 不依赖 jq 做 escape（避免 jq 本身异常时陷入死循环），改用 bash 参数替换；
# 2) 通过全局标志 _JSON_EMITTED 保证同一次运行内 fd 3 上最多输出一次 JSON，
#    避免 trap EXIT 与显式 _fatal 重复输出导致 Go 侧 Unmarshal 失败；
# 3) 退出前 reap 后台备份子进程，避免 TAT 环境下 SIGHUP kill 产生不完整备份。
_JSON_EMITTED=0
_json_escape() {
  # 将入参转义为合法 JSON 字符串值（不含外层引号）
  local s="$1"
  s="${s//\\/\\\\}"   # \ -> \\
  s="${s//\"/\\\"}"   # " -> \"
  s="${s//$'\n'/\\n}"  # 换行
  s="${s//$'\r'/\\r}"  # 回车
  s="${s//$'\t'/\\t}"  # 制表
  printf '%s' "$s"
}
_emit_error_json() {
  [ "$_JSON_EMITTED" = "1" ] && return 0
  local msg="${1:-脚本异常退出}"
  printf '{"error":"%s"}\n' "$(_json_escape "$msg")" >&3
  _JSON_EMITTED=1
}
_reap_bak() {
  if [ -n "${BAK_PID:-}" ]; then
    wait "$BAK_PID" 2>/dev/null || true
  fi
}
_fatal() {
  local msg="${1:-脚本异常退出}"
  echo "✗ $msg"
  _reap_bak
  _emit_error_json "$msg"
  exit 1
}
# 兜底：若脚本因 set -e 在未显式调用 _fatal 的地方退出，
# 仍向 fd 3 输出一个错误 JSON，避免 Go 侧收到空 stdout。
_on_err() {
  local ec=$?
  local line="${BASH_LINENO[0]:-?}"
  echo "✗ 脚本在第 ${line} 行以退出码 ${ec} 异常终止"
  _reap_bak
  _emit_error_json "脚本在第 ${line} 行以退出码 ${ec} 异常终止，详见 ${LOG_FILE}"
}
trap _on_err ERR
# EXIT 兜底：正常路径下 JSON 已输出（_JSON_EMITTED=1），此处不会重复；
# 仅处理 ERR 未触发但脚本仍异常退出（例如 SIGPIPE）的极端情况。
trap '_emit_error_json "脚本在未输出正常结果前退出，详见 $LOG_FILE"' EXIT
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo "=== OpenClaw Gateway UI 配置 ==="
echo "GATEWAY_IP:      $GATEWAY_IP"
echo "GATEWAY_UI_PORT: $GATEWAY_UI_PORT"
echo "ORIGIN:          $ORIGIN"
echo "CONFIG:          $CONFIG"
# ---------- 轻量 TCP 探测：利用 bash 内建 /dev/tcp，无子进程开销 ----------
# 比反复 fork `openclaw gateway health` 快一个数量级
tcp_probe() {
  # $1=host $2=port
  # 在子 shell 内完成连接与关闭，通过退出码传递结果（0=成功，1=失败）。
  # 不在父 shell 中操作任何 fd，避免与 fd 3（原始 stdout 副本）冲突。
  # 用 timeout 包一层，避免极端情况下 /dev/tcp 卡住导致健康检查死循环；
  # 若系统无 timeout 命令，退回直接探测。
  if command -v timeout >/dev/null 2>&1; then
    timeout 1 bash -c "exec 9<>/dev/tcp/$1/$2" 2>/dev/null
  else
    ( exec 9<>"/dev/tcp/$1/$2" ) 2>/dev/null
  fi
}
# ---------- 设备审批异步化（fire-and-forget，不阻塞主流程） ----------
# 配置中已设置 dangerouslyDisableDeviceAuth=true，审批结果不影响面板访问，
# 因此无需等待 openclaw devices list/approve 返回，后台执行即可。
echo ""
echo ">>> [步骤 1/5] 异步触发设备审批（fire-and-forget）"
(
  REQUEST_ID=""
  if DEVICES_JSON=$(openclaw devices list --json 2>/dev/null); then
    REQUEST_ID=$(echo "$DEVICES_JSON" | jq -r '.pending[0].requestId // empty' 2>/dev/null || true)
  fi
  if [ -n "$REQUEST_ID" ]; then
    openclaw devices approve "$REQUEST_ID" >/dev/null 2>&1 || true
  fi
) >/dev/null 2>&1 &
disown 2>/dev/null || true
echo "✓ 设备审批任务已放入后台（不阻塞主流程）"
# ---------- 幂等短路：若配置已是目标状态且网关健康，直接跳过重启 ----------
# 优化点：把原先 5 次 jq 调用合并为 1 次（单次解析，TAB 分隔输出），
#         健康检查用 TCP 探测替代 openclaw gateway health 子进程。
echo ""
echo ">>> [步骤 2/5] 幂等检查：判断是否需要重新应用配置"
NEED_APPLY=true
if [ -f "$CONFIG" ]; then
  IFS=$'\t' read -r CUR_PORT CUR_BIND CUR_INSECURE CUR_DISABLE_DEV CUR_ORIGIN < <(
    jq -r '[
      (.gateway.port // ""),
      (.gateway.bind // ""),
      (.gateway.controlUi.allowInsecureAuth // ""),
      (.gateway.controlUi.dangerouslyDisableDeviceAuth // ""),
      (.gateway.controlUi.allowedOrigins[0] // "")
    ] | @tsv' "$CONFIG" 2>/dev/null || echo $'\t\t\t\t'
  )
  echo "当前配置: port=$CUR_PORT bind=$CUR_BIND allowInsecureAuth=$CUR_INSECURE dangerouslyDisableDeviceAuth=$CUR_DISABLE_DEV origin=$CUR_ORIGIN"
  if [ "$CUR_PORT" = "$GATEWAY_UI_PORT" ] \
     && [ "$CUR_BIND" = "lan" ] \
     && [ "$CUR_INSECURE" = "true" ] \
     && [ "$CUR_DISABLE_DEV" = "true" ] \
     && [ "$CUR_ORIGIN" = "$ORIGIN" ]; then
    # 配置完全匹配，再确认网关健康；健康则直接跳过 restart。
    if tcp_probe 127.0.0.1 "$GATEWAY_UI_PORT"; then
      NEED_APPLY=false
      echo "✓ 配置已为目标状态且网关端口 $GATEWAY_UI_PORT 健康，跳过重启"
    else
      echo "⚠ 配置已为目标状态，但端口 $GATEWAY_UI_PORT TCP 探测失败，需要重启"
    fi
  else
    echo "⚠ 配置与目标状态不一致，需要重新应用"
  fi
else
  echo "⚠ 配置文件不存在: $CONFIG，将执行完整写入流程"
fi

if [ "$NEED_APPLY" = "true" ]; then
  echo ""


  echo ""
  echo ">>> [步骤 3/5] 写入目标配置到 $CONFIG"
  # 修改配置：
  # 1. gateway.port = GATEWAY_UI_PORT（使用管理后台分配的端口）
  # 2. gateway.controlUi.allowedOrigins = [ORIGIN]
  # 3. gateway.bind = "lan"
  # 4. gateway.controlUi.allowInsecureAuth = true
  # 5. gateway.controlUi.dangerouslyDisableDeviceAuth = true
  jq --arg origin "$ORIGIN" --argjson port "$GATEWAY_UI_PORT" '
    .gateway.port = $port |
    .gateway.bind = "lan" |
    .gateway.controlUi.allowedOrigins = [$origin] |
    .gateway.controlUi.allowInsecureAuth = true |
    .gateway.controlUi.dangerouslyDisableDeviceAuth = true
  ' "$CONFIG" > /tmp/openclaw.json
  echo "✓ 新配置已生成到 /tmp/openclaw.json"
  # 备份异步化：cp 放后台，不阻塞主流程。
  # 注意：
  #   1) 时间戳避免使用 ':'，提升在 NFS/SMB/网盘等文件系统上的可移植性；
  #   2) 追加 PID 后缀，避免同一秒内多次调用产生同名备份冲突；
  #   3) mv 是 rename（原子），后台 cp 即便正在读取旧 inode 也安全——inode
  #      引用计数会让旧文件在 cp 完成后自动回收。
  BAK_FILE="${CONFIG}.bak.$(date +%Y%m%dT%H%M%S).$$"
  cp "$CONFIG" "$BAK_FILE" &
  BAK_PID=$!
  mv /tmp/openclaw.json "$CONFIG"
  echo "✓ 配置已生效（原文件异步备份到 $BAK_FILE，PID=$BAK_PID）"
  # ---------- 更新 systemd 服务文件中的端口（仅在需要时改写 + reload） ----------
  echo ""
  echo ">>> [步骤 4/5] 更新 systemd 用户服务文件端口并重启 gateway"
  GATEWAY_SERVICE="$HOME/.config/systemd/user/openclaw-gateway.service"
  if [ ! -f "$GATEWAY_SERVICE" ]; then
    _fatal "service 文件未找到: $GATEWAY_SERVICE"
  fi
  # 仅当 service 文件中的端口与目标不一致时才 sed + daemon-reload，
  # 可在"配置变了但 service 端口本就正确"的场景省掉一次 daemon-reload（几百 ms ~ 秒级）。
  # 端口匹配使用 ERE + 非数字/行尾 边界，避免 \b 在不同 grep 版本下行为不一致，
  # 并防止 port=80 误匹配 --port 8080 这类前缀重叠情况。
  if ! grep -Eq "Environment=OPENCLAW_GATEWAY_PORT=${GATEWAY_UI_PORT}([^0-9]|$)" "$GATEWAY_SERVICE" \
     || ! grep -Eq "gateway --port ${GATEWAY_UI_PORT}([^0-9]|$)" "$GATEWAY_SERVICE"; then
    sed -i \
        -e "s/Environment=OPENCLAW_GATEWAY_PORT=.*/Environment=OPENCLAW_GATEWAY_PORT=${GATEWAY_UI_PORT}/" \
        -e "s/gateway --port [0-9]*/gateway --port ${GATEWAY_UI_PORT}/" \
        "$GATEWAY_SERVICE"
    systemctl --user daemon-reload >/dev/null 2>&1
    echo "✓ service 文件端口已更新为 $GATEWAY_UI_PORT 并执行 daemon-reload"
  else
    echo "✓ service 文件端口已是目标值，跳过 sed 与 daemon-reload"
  fi

  # 重启网关
  echo "重启 openclaw-gateway..."
  if ! systemctl --user restart openclaw-gateway 2>/tmp/restart_err; then
    RESTART_ERR=$(cat /tmp/restart_err 2>/dev/null || true)
    # 抓取 systemd 诊断信息：既要 echo 到日志，也要拼进 _fatal 消息回传给 Go 侧。
    # 用 `|| true` 防止 set -e 在管道任一段失败时把整条命令视为 fatal。
    SVC_STATUS=$(systemctl --user status openclaw-gateway --no-pager 2>&1 | head -n 20 || true)
    SVC_JOURNAL=$(journalctl --user -u openclaw-gateway -n 20 --no-pager 2>&1 || true)
    echo "--- systemctl restart 失败，诊断信息 ---"
    echo "错误输出: $RESTART_ERR"
    echo "$SVC_STATUS"
    echo "$SVC_JOURNAL"
    # 多行诊断通过 \n 拼接进 _fatal 消息；_emit_error_json 会做 JSON 转义。
    # 限制 journal 截取行数（20 行）已能保证最终 JSON 大小可控（< 几 KB）。
    _fatal "systemctl --user restart openclaw-gateway 失败: ${RESTART_ERR}
--- systemctl status (head -n 20) ---
${SVC_STATUS}
--- journalctl -n 20 ---
${SVC_JOURNAL}"
  fi
  echo "✓ openclaw-gateway 已发送 restart 指令"

  # ---------- 健康检查：TCP 探测 + systemd 服务状态双重校验 ----------
  # 设计思路：
  #   1) Go 侧最终只通过 HTTP 访问 gateway 端口，因此 TCP 端口可连通即代表"可用"，
  #      不需要（也不应该）依赖 `openclaw gateway health` 的 CLI 输出格式，
  #      该 CLI 输出在不同版本间可能变化（大小写、格式、JSON 化等），容易产生误判；
  #   2) 为防"端口被其他进程抢占"的极端情况，用 systemctl --user is-active 辅助确认，
  #      只要服务处于 active 状态 + 本地端口可连通，即视为健康；
  #   3) TCP 探测是 bash 内建 /dev/tcp，无子进程开销，100ms 粒度快速收敛；
  #   4) 总最长等待 ~30s（300 × 100ms）。
  #      openclaw-gateway 冷启动涉及配置解析、设备通道建立、UI 初始化等，
  #      实测在高负载或首次启动场景下可能需要 10s+，给足 30s 预算避免误判。
  #      Go 侧 RunScript 超时为 120s，此处 30s 内完全安全。
  #   5) 若 systemctl 报告 failed，立即中止等待快速失败，无需干等 30s。
  echo ""
  echo ">>> [步骤 5/5] 健康检查（TCP 探测 + 服务状态，最长 ~30s）"
  HEALTHY=false
  ATTEMPTS=0
  LAST_SVC_STATE=""
  MAX_ATTEMPTS=300
  while [ "$ATTEMPTS" -lt "$MAX_ATTEMPTS" ]; do
    ATTEMPTS=$((ATTEMPTS + 1))
    # 每 10 次（~1s）检查一次服务状态，若已 failed 则快速失败
    if [ $((ATTEMPTS % 10)) -eq 0 ]; then
      CUR_SVC_STATE=$(systemctl --user is-active openclaw-gateway 2>/dev/null || echo "unknown")
      if [ "$CUR_SVC_STATE" = "failed" ]; then
        LAST_SVC_STATE="failed"
        echo "⚠ openclaw-gateway 服务状态为 failed，提前终止等待"
        break
      fi
    fi
    if tcp_probe 127.0.0.1 "$GATEWAY_UI_PORT"; then
      # TCP 通了，再确认 systemd 服务也处于 active 状态，排除端口抢占
      LAST_SVC_STATE=$(systemctl --user is-active openclaw-gateway 2>/dev/null || echo "unknown")
      if [ "$LAST_SVC_STATE" = "active" ]; then
        HEALTHY=true
        break
      fi
    fi
    sleep 0.1
  done
  if [ "$HEALTHY" != "true" ]; then
    # 失败时尽量给出可排查的信息：尝试次数、端口、服务状态、CLI 输出、systemd 状态、journalctl 近期日志
    LAST_HEALTH=$(openclaw gateway health 2>&1 || true)
    SVC_STATUS=$(systemctl --user status openclaw-gateway --no-pager 2>&1 | head -n 20 || true)
    SVC_JOURNAL=$(journalctl --user -u openclaw-gateway -n 20 --no-pager 2>&1 || true)
    echo "--- 健康检查失败调试信息 ---"
    echo "最近一次 systemctl is-active: ${LAST_SVC_STATE:-<未探测到 TCP 连通>}"
    echo "最近一次 openclaw gateway health 输出:"
    echo "$LAST_HEALTH"
    echo "--- systemctl --user status openclaw-gateway ---"
    echo "$SVC_STATUS"
    echo "--- journalctl --user -u openclaw-gateway 最近 20 行 ---"
    echo "$SVC_JOURNAL"
    _fatal "gateway 健康检查失败（已尝试 $ATTEMPTS 次，端口: $GATEWAY_UI_PORT，服务状态: ${LAST_SVC_STATE:-未就绪}）
--- openclaw gateway health ---
${LAST_HEALTH}
--- systemctl status (head -n 20) ---
${SVC_STATUS}
--- journalctl -n 20 ---
${SVC_JOURNAL}"
  fi
  echo "✓ gateway 健康检查通过（尝试 $ATTEMPTS 次后命中，端口: $GATEWAY_UI_PORT，服务状态: $LAST_SVC_STATE）"
  # 等待备份完成（通常早已完成，这里只是回收子进程，避免产生 zombie）
  wait "$BAK_PID" 2>/dev/null || true
  echo "✓ 配置备份子进程已回收"
else
  echo ""
  echo ">>> [步骤 3/5] 跳过配置写入（幂等短路命中）"
  echo ">>> [步骤 4/5] 跳过 systemd 更新与重启（幂等短路命中）"
  echo ">>> [步骤 5/5] 跳过健康检查（幂等短路命中）"
fi
# 输出关键信息（无论是否走 restart 分支，输出格式一致）
echo ""
echo "=== 关键配置输出 ==="
# 注意：调用方 json.Unmarshal 依赖该段输出为纯 JSON。
# 直接重定向到 fd 3（原始 stdout），不经过 tee，避免 JSON 被写入两个目标造成重复。
# 同时用 tee 将 JSON 追加到日志文件，便于事后排障。
# 先把 JSON 写到临时变量，便于做字段校验与精确的错误提示
RESULT_JSON=$(jq -c '{
  port: .gateway.port,
  basePath: (.gateway.controlUi.basePath // ""),
  authToken: (.gateway.auth.token // "")
}' "$CONFIG" 2>/dev/null || true)
if [ -z "$RESULT_JSON" ]; then
  _fatal "读取配置文件失败或 jq 解析异常: $CONFIG"
fi
# 关键字段校验：authToken 为空时 Go 侧会回报"脚本返回数据不完整"，
# 这里提前拦截并给出更具体的错误信息，方便排障。
RESULT_TOKEN=$(printf '%s' "$RESULT_JSON" | jq -r '.authToken')
RESULT_PORT=$(printf '%s' "$RESULT_JSON" | jq -r '.port')
if [ -z "$RESULT_TOKEN" ] || [ "$RESULT_TOKEN" = "null" ]; then
  _fatal "配置文件中 gateway.auth.token 为空，无法生成面板访问链接"
fi
if [ -z "$RESULT_PORT" ] || [ "$RESULT_PORT" = "null" ] || [ "$RESULT_PORT" = "0" ]; then
  _fatal "配置文件中 gateway.port 为空或为 0"
fi
# 同步写入日志 + fd 3（Go 侧 stdout）
echo "$RESULT_JSON" >> "$LOG_FILE"
printf '%s\n' "$RESULT_JSON" >&3
_JSON_EMITTED=1
echo ""
echo "=== Gateway UI 配置完成 ==="
echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="
# 正常结束：撤销 EXIT trap，避免 trap 再次触发错误 JSON 输出（虽然 _JSON_EMITTED=1 已兜底）
trap - EXIT ERR