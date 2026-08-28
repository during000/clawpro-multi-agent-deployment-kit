#!/usr/bin/env bash
# WhatsApp 配对码通道设置脚本 v3
# 模板变量：{{phone_number}}, {{dm_policy}}, {{self_chat_mode}}
# TAT timeout: 180s
#
# stdout 协议：JSON Lines，每行一个 JSON 对象，必须含 action 字段
#   action=progress          进度消息
#   action=show_pairing_code 配对码（code, expires_in）
#   action=finish            结束（success:bool, code?, message）
#
# 重要：stdout 只能输出 emit 函数产生的 JSON 行，
#       所有其他命令的 stdout 必须重定向到 /dev/null 或 stderr，
#       否则 Go 层 newJSONLinesHandler 无法正确解析。

set -euo pipefail

# ========== 模板变量 ==========
PHONE="{{phone_number}}"
DM_POLICY="{{dm_policy}}"
SELF_CHAT_MODE="{{self_chat_mode}}"

# ========== 常量 ==========
WA_LOGIN_DIR="/tmp/wa-login-$$"
AUTH_DIR="$HOME/.openclaw/credentials/whatsapp/default"
LOCK_FILE="/tmp/wa-pairing.lock"
OC_JSON="$HOME/.openclaw/openclaw.json"
AUDIT_LOG="/tmp/wa-pairing-audit.log"

# ========== 审计日志（写文件，不污染 stdout）==========
audit() {
  local msg="[$(date '+%Y-%m-%d %H:%M:%S')] $*"
  echo "$msg" >> "$AUDIT_LOG" 2>/dev/null || true
}

# ========== stdout JSON Lines 输出（对齐 action 协议）==========
emit_progress() {
  echo "{\"action\":\"progress\",\"message\":\"$1\"}"
}
emit_finish_ok() {
  echo "{\"action\":\"finish\",\"success\":true,\"message\":\"$1\"}"
}
emit_finish_err() {
  echo "{\"action\":\"finish\",\"success\":false,\"code\":\"$1\",\"message\":\"$2\"}"
}

audit "START phone=$PHONE dmPolicy=$DM_POLICY selfChatMode=$SELF_CHAT_MODE"

# ========== Node 版本检查 ==========
NODE_VERSION=$(node -v 2>/dev/null | sed 's/v//; s/\..*//')
if [ -z "$NODE_VERSION" ] || [ "$NODE_VERSION" -lt 20 ] 2>/dev/null; then
  emit_finish_err "NODE_TOO_OLD" "Node.js v${NODE_VERSION:-未安装} 过低，需要 >= 20"
  audit "FAIL Node version too old: v${NODE_VERSION:-missing}"
  exit 1
fi

# ========== jq 依赖检查 ==========
if ! command -v jq &>/dev/null; then
  audit "INFO installing jq..."
  apt-get install -y jq >/dev/null 2>&1 || yum install -y jq >/dev/null 2>&1 || true
fi
if ! command -v jq &>/dev/null; then
  emit_finish_err "DEPENDENCY" "jq 未安装且安装失败，请联系管理员"
  audit "FAIL jq not available"
  exit 1
fi

# ========== 手机号格式校验 ==========
# 规则：1-9 开头，后跟 6-14 位数字 → 总长 7-15 位（兼容国际区号）
if ! echo "$PHONE" | grep -qE '^[1-9][0-9]{6,14}$'; then
  emit_finish_err "INVALID_PHONE" "手机号格式无效：必须是7-15位纯数字（1-9开头，不含+号），如 85266803489"
  audit "FAIL invalid phone format: $PHONE"
  exit 1
fi

# ========== 并发互斥锁 ==========
exec 200>"$LOCK_FILE"
if ! flock -n 200; then
  emit_finish_err "BUSY" "已有配对流程在进行中，请等待完成后重试"
  audit "FAIL another pairing in progress"
  exit 1
fi
trap 'flock -u 200; rm -f "$LOCK_FILE"; rm -rf "$WA_LOGIN_DIR"' EXIT

# ========== 网络前置检测（快速失败）==========
emit_progress "检查网络连通性..."
if ! timeout 10 openssl s_client -connect g.whatsapp.net:443 -servername g.whatsapp.net -brief </dev/null 2>&1 | grep -q "Verification: OK"; then
  emit_finish_err "NETWORK" "无法连接 WhatsApp (g.whatsapp.net:443)，请确认实例网络配置"
  audit "FAIL network check failed for g.whatsapp.net:443"
  exit 1
fi

# ========== 0. 确保 WhatsApp 插件已启用 ==========
# 跳过耗时的 `openclaw plugins list` 扫描（要枚举 npm/projects 全部插件，~15s+）
# 改用轻量级方案：直接尝试启用，幂等操作
emit_progress "确保 WhatsApp 插件已启用..."
audit "INFO ensuring whatsapp plugin enabled..."
if ! timeout 30 openclaw plugins enable whatsapp >/dev/null 2>&1; then
  EN_ERR=$(timeout 30 openclaw plugins enable whatsapp 2>&1 | head -3)
  emit_finish_err "PLUGIN_DISABLED" "WhatsApp 插件启用失败: ${EN_ERR:-未知错误}"
  audit "FAIL plugin enable: $EN_ERR"
  exit 1
fi
audit "OK plugin enabled (or already enabled)"

# ========== 1. 准备临时工作目录 ==========
emit_progress "准备临时工作目录..."
rm -rf "$AUTH_DIR"
mkdir -p "$WA_LOGIN_DIR"
cd "$WA_LOGIN_DIR"

# ========== 2. 软链 baileys ==========
emit_progress "查找 baileys 库..."
BAILEYS_DIR=$(find "$HOME/.openclaw/npm/projects" -path "*@openclaw/whatsapp/node_modules/baileys" -type d 2>/dev/null | head -1)
if [ -z "$BAILEYS_DIR" ]; then
  emit_finish_err "PLUGIN_MISSING" "找不到 baileys 库，请确认 WhatsApp 插件已正确安装"
  audit "FAIL baileys not found in npm projects"
  exit 1
fi
mkdir -p node_modules
ln -sfn "$BAILEYS_DIR" node_modules/baileys
audit "OK baileys linked: $BAILEYS_DIR"

# ========== 4. 写入配对码脚本（内置版本 + Browsers.ubuntu + 515 自动重连） ==========
#
# 关键约束：TAT agent 对 stdout 输出大小有上限（约 16KB），超出后停止上报。
# baileys 库在 connection === 'open' 后会输出大量 sync 日志（每条数 KB），
# 会撑满 TAT 缓冲区，导致最后的 emit_finish_ok 被丢弃（Dropped）。
# 因此在 connection === 'open' 后立即 stubbed stdout，sync 日志不进入 TAT 缓冲区。
# 同时等待 creds.json 真正落盘后再 exit，避免时序竞态（exit 太快文件未写入）。
cat > login.js <<'LOGINEOF'
const baileys = require('baileys')
const makeWASocket = baileys.default || baileys.makeWASocket
const { useMultiFileAuthState, DisconnectReason, Browsers } = baileys
const fs = require('fs')

let reconnectCount = 0
const phone = process.argv[2]

async function connect() {
  const { state, saveCreds } = await useMultiFileAuthState('./auth')
  const sock = makeWASocket({
    auth: state,
    browser: Browsers.ubuntu('Chrome'),
    connectTimeoutMs: 60000,
    keepAliveIntervalMs: 15000,
  })
  sock.ev.on('creds.update', saveCreds)
  sock.ev.on('connection.update', (u) => {
    if (u.connection === 'open') {
      // 先输出最后一条 progress（在 stub 之前，确保 TAT 能收到）
      console.log(JSON.stringify({action:"progress",message:"WhatsApp 登录成功，正在写入通道配置..."}))

      // 立即 stub stdout：后续 baileys sync 日志不进入 TAT agent 输出缓冲
      process.stdout.write = () => true

      // 等待 creds.json 真正落盘后再 exit，避免时序竞态
      const maxWait = 30000   // 最多等 30s
      const start = Date.now()
      const checkAndExit = () => {
        try {
          if (fs.existsSync('./auth/creds.json') && fs.statSync('./auth/creds.json').size > 0) {
            process.exit(0)
          }
        } catch (_) {}
        if (Date.now() - start < maxWait) {
          setTimeout(checkAndExit, 200)
        } else {
          process.exit(1)
        }
      }
      setTimeout(checkAndExit, 500)
      return
    }
    if (u.connection === 'close') {
      const sc = u.lastDisconnect?.error?.output?.statusCode
      if (sc === DisconnectReason.restartRequired || sc === 515) {
        reconnectCount++
        if (reconnectCount <= 5) {
          console.log(JSON.stringify({action:"progress",message:"正在完成关联..."}))
          setTimeout(() => connect(), 1500)
        } else {
          console.log(JSON.stringify({action:"finish",success:false,code:"RECONNECT_FAILED",message:"重连次数过多"}))
          process.exit(1)
        }
        return
      }
      if (sc === 400) {
        console.log(JSON.stringify({action:"finish",success:false,code:"REJECTED",message:"WhatsApp 服务端拒绝(400)"}))
      } else {
        console.log(JSON.stringify({action:"finish",success:false,code:"LINK_FAILED",message:"关联失败,statusCode="+sc}))
      }
      process.exit(1)
    }
  })
  if (!state.creds.registered) {
    console.log(JSON.stringify({action:"progress",message:"正在连接 WhatsApp..."}))
    let code
    for (let i = 0; i < 3; i++) {
      try {
        await new Promise(r => setTimeout(r, 2000))
        code = await sock.requestPairingCode(phone)
        break
      } catch (e) {
        if (i === 2) {
          console.log(JSON.stringify({action:"finish",success:false,code:"NETWORK",message:"无法连接 WhatsApp: "+e.message}))
          process.exit(1)
        }
      }
    }
    console.log(JSON.stringify({action:"show_pairing_code",code,expires_in:60,message:"请在手机 WhatsApp 输入配对码"}))
  } else {
    console.log(JSON.stringify({action:"progress",message:"凭证有效，正在登录..."}))
  }
}
connect().catch(e => {
  console.log(JSON.stringify({action:"finish",success:false,code:"FATAL",message:e.message}))
  process.exit(1)
})
LOGINEOF

# ========== 5. 执行配对码登录（阻塞，等待用户手机操作） ==========
emit_progress "启动配对流程，请同时在手机输入配对码..."
rm -rf auth
node login.js "$PHONE"
EXIT_CODE=$?

# ========== 6. 判断结果 ==========
if [ $EXIT_CODE -eq 0 ] && [ -f auth/creds.json ]; then
  # 7. 拷贝完整凭证（成功后再替换旧凭证，避免配对失败破坏现有可用通道）
  emit_progress "拷贝 WhatsApp 凭证..."
  rm -rf "$AUTH_DIR"
  mkdir -p "$AUTH_DIR"
  cp -a auth/. "$AUTH_DIR/"
  audit "OK credentials copied to $AUTH_DIR ($(ls "$AUTH_DIR" | wc -l) files)"

  # 8. 注册通道（stdout 重定向，不污染 JSON Lines）
  emit_progress "注册 WhatsApp 通道..."
  timeout 30 openclaw channels add --channel whatsapp --account default >/dev/null 2>&1 || true

  # 9. 写入通道策略配置
  emit_progress "写入 channels.whatsapp 配置..."
  TMP_JSON="/tmp/openclaw_wa_$$.json"
  if ! jq --arg phone "$PHONE" --arg dm "$DM_POLICY" --argjson selfchat "$SELF_CHAT_MODE" \
    '.channels.whatsapp += {"dmPolicy":$dm,"allowFrom":[$phone],"selfChatMode":$selfchat,"accounts":{"default":{"enabled":true}}}' \
    "$OC_JSON" > "$TMP_JSON" 2>/dev/null; then
    emit_finish_err "CONFIG_WRITE_FAILED" "写入 WhatsApp 通道配置失败"
    audit "FAIL write channels.whatsapp config"
    exit 1
  fi
  mv "$TMP_JSON" "$OC_JSON"
  audit "OK channels config written"

  # 10. 确保 plugins 配置正确
  emit_progress "更新 plugins 配置..."
  if ! jq '.plugins.entries.whatsapp.enabled = true | .plugins.allow |= if . | index("whatsapp") == null then . + ["whatsapp"] else . end' \
    "$OC_JSON" > "$TMP_JSON" 2>/dev/null; then
    emit_finish_err "CONFIG_WRITE_FAILED" "写入 WhatsApp 插件配置失败"
    audit "FAIL write plugins config"
    exit 1
  fi
  mv "$TMP_JSON" "$OC_JSON"
  audit "OK plugins config ensured (entries+allow)"

  # 11. 重启 gateway
  emit_progress "重启 OpenClaw Gateway..."
  if ! timeout 30 systemctl --user restart openclaw-gateway.service >/dev/null 2>&1; then
    emit_finish_err "GATEWAY_RESTART_FAILED" "重启 OpenClaw Gateway 失败"
    audit "FAIL restart gateway"
    exit 1
  fi
  emit_finish_ok "WhatsApp 通道配置完成"
  audit "DONE successfully"
else
  emit_finish_err "INCOMPLETE" "配对码登录未完成"
  audit "FAIL pairing incomplete (exit=$EXIT_CODE, creds_exists=$([ -f auth/creds.json ] && echo yes || echo no))"
  exit 1
fi

# 12. 清理临时目录
rm -rf "$WA_LOGIN_DIR"