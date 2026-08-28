#!/bin/bash
#
# WhatsApp 扫码登录脚本
#   管控端下发到 VM 上直接执行，stdout 直连管控端
#   管控端逐行解析 stdout 即可
#
# 扫码成功后自动:
#   1. stdout 输出 type=connected
#   2. 将 WhatsApp 账号写入 OpenClaw 配置
#   3. 重启网关 gateway 使通道生效
#   之后即可通过 WhatsApp 收发消息
#
# 退出码: 0=成功(可收发), 1=失败/超时
#
# === stdout 输出格式（管控端逐行解析）===
# 【日志】     以 [LOG] 开头 -> 可直接显示
# 【QR 码】    {"type":"qr","qrRaw":"2@..."}           <- 管控端推给前端渲染
# 【状态更新】  {"type":"status","status":"connected"|"timeout"|"logged_out"}
# 【错误】     {"type":"error","message":"..."}
#
set -euo pipefail

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# 配置 WhatsApp 通道前，先将插件 ID 加入 plugins.allow，再 enable 插件
# 检查 whatsapp 插件是否已经启用
PLUGIN_ENABLED=false
echo "检查 whatsapp 插件状态..."
if command -v openclaw >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    _plugin_info=$(
        set +o pipefail
        openclaw plugins list --json 2>/dev/null \
            | jq -r '.. | objects | select((.id // .name // "") == "whatsapp") | "\(.enabled // "")|\(.status // .state // "")"' 2>/dev/null \
            | head -n 1
    ) || true
    _plugin_enabled_field="${_plugin_info%%|*}"
    _plugin_status="${_plugin_info##*|}"
    if [ "$_plugin_enabled_field" = "true" ] || [ "$_plugin_status" = "loaded" ] || [ "$_plugin_status" = "enabled" ]; then
        PLUGIN_ENABLED=true
        echo "✓ 检测到 whatsapp 插件已启用 (enabled=${_plugin_enabled_field}, status=${_plugin_status})，跳过插件配置步骤"
    elif [ -n "$_plugin_status" ] || [ -n "$_plugin_enabled_field" ]; then
        echo "  whatsapp 插件当前 enabled=${_plugin_enabled_field}, status=${_plugin_status}，需要启用"
    else
        echo "  未从 JSON 输出中匹配到 whatsapp 插件，需要启用"
    fi
fi

if [ "$PLUGIN_ENABLED" = "false" ]; then
    # 将插件 ID 加入 plugins.allow（幂等：已存在则不重复追加）
    _cfg="$HOME/.openclaw/openclaw.json"
    if [ -f "$_cfg" ]; then
        jq --arg id "whatsapp" \
            '.plugins.allow = ((.plugins.allow // []) | if index($id) then . else . + [$id] end)' \
            "$_cfg" > /tmp/openclaw_allow.json && mv /tmp/openclaw_allow.json "$_cfg"
        echo "✓ whatsapp 已加入 plugins.allow"
    fi
    echo "启用 whatsapp 插件..."
    openclaw plugins enable whatsapp || true
    echo "✓ whatsapp 插件已启用"
    echo "重启 gateway 使插件生效..."
    systemctl --user restart openclaw-gateway || true
    echo "✓ gateway 已重启"
fi

VERBOSE="--verbose"

# 认证目录（持久化，下次自动复用）
AUTH_DIR="$HOME/.openclaw/whatsapp-login/whatsapp-auth"
WORK_DIR=$(mktemp -d /tmp/whatsapp-login-XXXXXX)
SCRIPT_FILE="$WORK_DIR/login.mjs"

# ---- 写入 Node.js 脚本 ----
cat > "$SCRIPT_FILE" << 'NODESCRIPT'
#!/usr/bin/env node
import { createRequire } from 'module';
import { homedir } from 'os';
import { readdirSync, existsSync } from 'fs';
import { join } from 'path';

function findWhatsappPlugin() {
  const projectsDir = join(homedir(), '.openclaw', 'npm', 'projects');
  if (!existsSync(projectsDir)) {
    throw new Error('Projects directory not found: ' + projectsDir);
  }
  const dirs = readdirSync(projectsDir, { withFileTypes: true })
    .filter(d => d.isDirectory() && d.name.startsWith('openclaw-whatsapp'));
  if (dirs.length === 0) {
    throw new Error('No openclaw-whatsapp project found in ' + projectsDir);
  }
  return join(projectsDir, dirs[0].name);
}

const whatsappProject = findWhatsappPlugin();
const baileysPath = join(whatsappProject, 'node_modules', '@openclaw', 'whatsapp', 'node_modules', 'baileys');
const req = createRequire(import.meta.url);
const pino = req(join(whatsappProject, 'node_modules', '@openclaw', 'whatsapp', 'node_modules', 'pino'));

const baileysMod = await import(`file://${baileysPath}/lib/index.js`);
const { makeWASocket, DisconnectReason, useMultiFileAuthState, fetchLatestBaileysVersion, makeCacheableSignalKeyStore } = baileysMod;

const args = process.argv.slice(2);
const authDir = parseArg(args, '--auth-dir');
const verbose = args.includes('--verbose');

function emit(action, data) { console.log(JSON.stringify({ action, ...data })); }
function log(msg, level) { emit('log', { level: level, message: msg }); }

const QR_TIMEOUT_MS   = 60_000;
const LOGIN_TIMEOUT_MS = 120_000;

function parseArg(args, name) {
  const i = args.indexOf(name);
  return i !== -1 && i + 1 < args.length ? args[i + 1] : null;
}
const sleep = ms => new Promise(r => setTimeout(r, ms));

async function main() {
  log('Connecting to WhatsApp...', 'info');
  const logger = pino({ level: verbose ? 'info' : 'silent' });
  const { version } = await fetchLatestBaileysVersion();
  const { state, saveCreds, clearAuth } = await useMultiFileAuthState(authDir);

  const sock = makeWASocket({
    auth: {
      creds: state.creds,
      keys: makeCacheableSignalKeyStore(state.keys, logger),
    },
    version, logger,
    printQRInTerminal: false,
    browser: ['openclaw', 'cli', '2026.5.28'],
    syncFullHistory: false,
    markOnlineOnConnect: false,
    keepAliveIntervalMs: 25_000,
    connectTimeoutMs: 60_000,
    defaultQueryTimeoutMs: 60_000,
  });

  sock.ev.on('creds.update', () => saveCreds());

  let qrSent = false;
  let connected = false;
  let loginError = null;
  let currentQr = null;

  sock.ev.on('connection.update', async update => {
    const { connection, lastDisconnect, qr } = update;

    if (qr) {
      currentQr = qr;
      if (!qrSent) {
        qrSent = true;
        emit('show_qrcode', { content: qr, render_qr: false });
      }
    }

    if (connection === 'open') {
      connected = true;
      qrSent = true;
    }

    if (connection === 'close') {
      const code = lastDisconnect?.error?.output?.statusCode
                ?? lastDisconnect?.error?.data?.statusCode;
      const reason = lastDisconnect?.error?.message || 'unknown';

      if (code === DisconnectReason.loggedOut) {
        loginError = 'logged_out';
        log('Logged out on other device', 'error');
      } else if (code === DisconnectReason.badSession) {
        loginError = 'bad_session';
        log('Session crush, clearing auth...', 'error');
        clearAuth();
      } else if (code === 515) {
        // Baileys v7: restartRequired — 扫码后自动断开通知重启
        // creds 已写入磁盘，视为登录成功
        if (qrSent) {
          connected = true;
        }
      } else {
        log(`断开(code=${code}): ${reason.slice(0,80)}`, 'info');
      }
    }
  });

  // 等待二维码（60s）
  log('waiting for QR...', 'info');
  const qrDeadline = Date.now() + QR_TIMEOUT_MS;
  while (!qrSent && Date.now() < qrDeadline) await sleep(500);

  if (!qrSent) {
    log('QR generate timeout', 'error');
    emit('finish', { level: 'error', step: 'finish', message: 'QR generate timeout' });
    sock.end();
    process.exit(1);
  }

  // 等待扫码（120s），同时检测 QR 刷新
  const loginDeadline = Date.now() + LOGIN_TIMEOUT_MS;
  while (!connected && !loginError && Date.now() < loginDeadline) {
    await sleep(3000);
    if (currentQr) {
      const prev = currentQr;
      await sleep(500);
      if (currentQr !== prev && currentQr) {
        log('QR refresh, resending...', 'info');
        emit('show_qrcode', { content: qr, render_qr: false });
      }
    }
  }

  sock.ev.removeAllListeners();
  sock.end();

  if (connected) {
    process.exit(0);
  } else if (loginError) {
    emit('finish', { level: 'error', step: 'finish', message: 'Login failed: ${loginError}' });
    process.exit(1);
  } else {
    emit('finish', { level: 'error', step: 'finish', message: 'QR timeout' });
    process.exit(1);
  }
}

main().catch(err => {
  emit('finish', { level: 'error', step: 'finish', message: err.message });
  process.exit(1);
});
NODESCRIPT

# ---- 执行 Node.js 登录 ----
node "$SCRIPT_FILE" --auth-dir "$AUTH_DIR" $VERBOSE
EXIT_CODE=$?
# ---- 扫码成功: 注册到 OpenClaw 配置 ----
if [[ $EXIT_CODE -eq 0 ]]; then
  CONFIG_FILE="$HOME/.openclaw/openclaw.json"
  CONFIG_BAK="${CONFIG_FILE}.bak.$(date +%y-%m-%dT%H:%M:%S)"
  cp "$CONFIG_FILE" "$CONFIG_BAK"
  python3 -c "
import json

with open('$CONFIG_FILE', 'r') as f:
    config = json.load(f)

config.setdefault('channels', {})
wa_cfg = config['channels'].setdefault('whatsapp', {})
accounts = wa_cfg.setdefault('accounts', {})
accounts['main'] = {'authDir': '$AUTH_DIR'}
wa_cfg.setdefault('enabled', True)

with open('$CONFIG_FILE', 'w') as f:
    json.dump(config, f, indent=4)
"
  systemctl --user restart openclaw-gateway.service 2>/dev/null
  echo '{"action":"finish","level":"success","step":"finish","message":"Whatsapp channel configured"}'
else
  echo '{"action":"finish","level":"error","step":"finish","message":"Whatsapp channel configuration failed"}'
fi

# ---- 清理 ----
rm -rf "$WORK_DIR"
exit $EXIT_CODE