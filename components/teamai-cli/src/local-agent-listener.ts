import { log } from './utils/logger.js';
import {
  buildSyncPayload,
  loadLocalAgentConfig,
  reportAndSyncLocalAgent,
  resolveRoute,
  type LocalAgentConfig,
  type LocalAgentContext,
} from './local-agent.js';

const HEARTBEAT_INTERVAL_MS = 20_000;
const RECONNECT_MAX_MS = 15_000;
const OPEN_TIMEOUT_MS = 10_000;

interface WakeTicketResponse {
  ticket?: string;
  expires_in_seconds?: number;
  error?: string;
}

interface WakeMessage {
  type?: string;
  task_id?: number;
  reason?: string;
}

export interface LocalAgentWakeListenerOptions {
  tool?: string;
  cwd?: string;
  once?: boolean;
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function endpointURL(config: LocalAgentConfig, routeName: 'wakeTicket' | 'wake'): URL {
  const endpoint = config.endpoint.replace(/\/+$/, '');
  return new URL(`${endpoint}${resolveRoute(config, routeName)}`);
}

async function requestWakeTicket(
  config: LocalAgentConfig,
  context: LocalAgentContext,
): Promise<string> {
  const syncPayload = await buildSyncPayload(config, context);
  const response = await fetch(endpointURL(config, 'wakeTicket'), {
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      ...(config.token ? { Authorization: `Bearer ${config.token}` } : {}),
    },
    body: JSON.stringify({
      agent_type: syncPayload.agent_type,
      local_agent_id: syncPayload.local_agent_id,
    }),
    signal: AbortSignal.timeout(15_000),
  });
  const body = await response.json() as WakeTicketResponse;
  if (!response.ok || !body.ticket) {
    throw new Error(body.error || `wake ticket request failed: HTTP ${response.status}`);
  }
  return body.ticket;
}

function wakeURL(config: LocalAgentConfig, ticket: string): string {
  const url = endpointURL(config, 'wake');
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  url.searchParams.set('ticket', ticket);
  return url.toString();
}

function messageText(data: unknown): string {
  if (typeof data === 'string') return data;
  if (data instanceof ArrayBuffer) return Buffer.from(data).toString('utf8');
  if (ArrayBuffer.isView(data)) {
    return Buffer.from(data.buffer, data.byteOffset, data.byteLength).toString('utf8');
  }
  return String(data ?? '');
}

async function openWakeSession(
  config: LocalAgentConfig,
  context: LocalAgentContext,
  ticket: string,
  stopped: () => boolean,
): Promise<void> {
  if (typeof WebSocket === 'undefined') {
    throw new Error('This Node.js runtime does not provide a WebSocket client; Node.js 22+ is required');
  }

  const socket = new WebSocket(wakeURL(config, ticket));
  let heartbeat: ReturnType<typeof setInterval> | undefined;
  let syncing = false;
  let syncAgain = false;

  const sync = async (message: WakeMessage): Promise<void> => {
    syncAgain = true;
    if (syncing) return;
    syncing = true;
    try {
      while (syncAgain && !stopped()) {
        syncAgain = false;
        log.info(
          `[local-agent-listen] ${message.type ?? 'wake'}${message.task_id ? ` task=${message.task_id}` : ''}; syncing`,
        );
        await reportAndSyncLocalAgent(context);
      }
    } finally {
      syncing = false;
    }
  };

  await new Promise<void>((resolve, reject) => {
    const openTimer = setTimeout(() => {
      socket.close();
      reject(new Error('WebSocket wake connection timed out'));
    }, OPEN_TIMEOUT_MS);
    let opened = false;

    socket.addEventListener('open', () => {
      opened = true;
      clearTimeout(openTimer);
      log.info('[local-agent-listen] WSS wake channel connected');
      heartbeat = setInterval(() => {
        if (socket.readyState === WebSocket.OPEN) {
          socket.send(JSON.stringify({ type: 'heartbeat', at: new Date().toISOString() }));
        }
      }, HEARTBEAT_INTERVAL_MS);
      void sync({ type: 'connected' });
    });
    socket.addEventListener('message', (event) => {
      try {
        const message = JSON.parse(messageText(event.data)) as WakeMessage;
        if (message.type === 'task_available' || message.type === 'sync_required') {
          void sync(message);
        }
      } catch (error) {
        log.debug(`[local-agent-listen] ignored invalid wake message: ${(error as Error).message}`);
      }
    });
    socket.addEventListener('close', () => {
      clearTimeout(openTimer);
      if (heartbeat) clearInterval(heartbeat);
      log.info('[local-agent-listen] WSS wake channel disconnected');
      resolve();
    });
    socket.addEventListener('error', () => {
      if (!opened) {
        clearTimeout(openTimer);
        reject(new Error('WebSocket wake connection failed'));
      } else {
        socket.close();
      }
    });
  });
}

/**
 * Keep an outbound TeamAI connection to ClawPro. WebSocket messages contain
 * only a task id; the actual command is always claimed through authenticated
 * HTTPS sync, so disconnects are recovered by the sync run after reconnect.
 */
export async function runLocalAgentWakeListener(
  options: LocalAgentWakeListenerOptions = {},
): Promise<void> {
  const config = await loadLocalAgentConfig();
  if (!config) throw new Error('Local Agent configuration is missing; connect TeamAI to ClawPro first');
  const context: LocalAgentContext = {
    cwd: options.cwd ?? process.cwd(),
    tool: options.tool ?? 'codebuddy',
    status: 'running',
  };
  let shouldStop = false;
  const stop = () => { shouldStop = true; };
  process.once('SIGINT', stop);
  process.once('SIGTERM', stop);

  let backoff = 500;
  try {
    while (!shouldStop) {
      try {
        const ticket = await requestWakeTicket(config, context);
        await openWakeSession(config, context, ticket, () => shouldStop);
        backoff = 500;
        if (options.once) return;
      } catch (error) {
        if (options.once) throw error;
        log.warn(`[local-agent-listen] ${(error as Error).message}; reconnecting in ${backoff}ms`);
      }
      if (!shouldStop) {
        await delay(backoff);
        backoff = Math.min(backoff * 2, RECONNECT_MAX_MS);
      }
    }
  } finally {
    process.off('SIGINT', stop);
    process.off('SIGTERM', stop);
  }
}
