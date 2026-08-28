import https from 'node:https';
import readline from 'node:readline';

const IWIKI_MCP_URL = new URL('https://prod.mcp.it.woa.com/app_iwiki_mcp/mcp3');
const ALLOWED_TOOLS = new Set(['metadata', 'getSpacePageTree', 'getDocument']);
const REQUEST_TIMEOUT_MS = 30_000;

interface JsonRpcMessage {
  jsonrpc?: string;
  id?: string | number | null;
  method?: string;
  params?: Record<string, unknown>;
  result?: Record<string, unknown>;
  error?: { code?: number; message?: string };
}

function jsonRpcError(request: JsonRpcMessage, code: number, message: string): JsonRpcMessage {
  return {
    jsonrpc: '2.0',
    id: request.id ?? null,
    error: { code, message },
  };
}

function parseResponseBody(raw: string): JsonRpcMessage | null {
  const body = raw.trim();
  if (!body) return null;
  if (!body.startsWith('data:')) return JSON.parse(body) as JsonRpcMessage;

  const payload = body
    .split(/\r?\n/)
    .filter((line) => line.startsWith('data:'))
    .map((line) => line.slice(5).trim())
    .filter(Boolean)
    .at(-1);
  return payload ? JSON.parse(payload) as JsonRpcMessage : null;
}

async function postToIWiki(request: JsonRpcMessage, token: string): Promise<JsonRpcMessage | null> {
  const payload = JSON.stringify(request);
  return new Promise<JsonRpcMessage | null>((resolve, reject) => {
    const upstream = https.request(
      IWIKI_MCP_URL,
      {
        method: 'POST',
        headers: {
          accept: 'application/json, text/event-stream',
          authorization: `Bearer ${token}`,
          'content-length': Buffer.byteLength(payload),
          'content-type': 'application/json',
        },
      },
      (response) => {
        const chunks: Buffer[] = [];
        response.on('data', (chunk: Buffer | string) => chunks.push(Buffer.from(chunk)));
        response.on('end', () => {
          const raw = Buffer.concat(chunks).toString('utf8');
          if ((response.statusCode ?? 500) >= 400) {
            reject(new Error(`iWiki MCP authorization/request failed (${response.statusCode}): ${raw.slice(0, 300)}`));
            return;
          }
          try {
            resolve(parseResponseBody(raw));
          } catch (error) {
            reject(new Error(`iWiki MCP returned invalid JSON: ${String(error)}`));
          }
        });
      },
    );
    const timer = setTimeout(() => {
      upstream.destroy(new Error(`iWiki MCP request timed out after ${REQUEST_TIMEOUT_MS / 1000}s`));
    }, REQUEST_TIMEOUT_MS);
    upstream.once('close', () => clearTimeout(timer));
    upstream.once('error', reject);
    upstream.end(payload);
  });
}

function restrictToolList(response: JsonRpcMessage | null): JsonRpcMessage | null {
  const tools = response?.result?.tools;
  if (!Array.isArray(tools)) return response;
  response!.result!.tools = tools.filter((tool) => {
    if (!tool || typeof tool !== 'object') return false;
    return ALLOWED_TOOLS.has(String((tool as Record<string, unknown>).name ?? ''));
  });
  return response;
}

/**
 * Read-only stdio MCP bridge used by TeamAI-launched CodeBuddy sessions.
 * The user PAT is inherited from TeamAI's environment and is never written to
 * the CodeBuddy MCP configuration, task workspace, stdout, or error messages.
 */
export async function runIWikiMcpProxy(): Promise<void> {
  const lines = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
  for await (const line of lines) {
    if (!line.trim()) continue;
    let request: JsonRpcMessage;
    try {
      request = JSON.parse(line) as JsonRpcMessage;
    } catch {
      process.stdout.write(`${JSON.stringify(jsonRpcError({}, -32700, 'Invalid JSON-RPC payload'))}\n`);
      continue;
    }

    const token = process.env.TAI_PAT_TOKEN?.trim() ?? '';
    if (!token) {
      process.stdout.write(`${JSON.stringify(jsonRpcError(
        request,
        -32001,
        'iWiki is not authorized. Configure TAI_PAT_TOKEN in TeamAI and retry.',
      ))}\n`);
      continue;
    }

    if (request.method === 'tools/call') {
      const toolName = String(request.params?.name ?? '');
      if (!ALLOWED_TOOLS.has(toolName)) {
        process.stdout.write(`${JSON.stringify(jsonRpcError(
          request,
          -32601,
          `Tool is not allowed by the TeamAI iWiki read-only profile: ${toolName}`,
        ))}\n`);
        continue;
      }
    }

    try {
      let response = await postToIWiki(request, token);
      if (request.method === 'tools/list') response = restrictToolList(response);
      if (response && request.id !== undefined) {
        process.stdout.write(`${JSON.stringify(response)}\n`);
      }
    } catch (error) {
      if (request.id !== undefined) {
        process.stdout.write(`${JSON.stringify(jsonRpcError(
          request,
          -32002,
          (error as Error).message,
        ))}\n`);
      }
    }
  }
}

