import fs from 'node:fs';
import http from 'node:http';
import https from 'node:https';
import path from 'node:path';
import readline from 'node:readline';
import { spawn, type ChildProcessWithoutNullStreams } from 'node:child_process';
import { resolveTeamaiEntryScript } from './builtin-hooks.js';

const DEFAULT_TASK_TIMEOUT_MS = 20 * 60 * 1000;
const RPC_TIMEOUT_MS = 30_000;
const SNAPSHOT_INTERVAL_MS = 1_000;
const MAX_RESULT_CHARS = 1_900_000;
const CODEBUDDY_BASE_TOOLS = ['Read', 'Write', 'Edit', 'Glob', 'Grep'];
const CODEBUDDY_IWIKI_TOOLS = [
  'mcp__iwiki__metadata',
  'mcp__iwiki__getSpacePageTree',
  'mcp__iwiki__getDocument',
];
const CODEBUDDY_IWIKI_DISALLOWED_TOOLS = ['Bash', 'WebFetch', 'WebSearch'];
const CODEBUDDY_SOURCE_DEVELOPMENT_ALLOWED_TOOLS = [...CODEBUDDY_IWIKI_TOOLS, 'Bash'];
const CODEBUDDY_SOURCE_DEVELOPMENT_DISALLOWED_TOOLS = ['WebFetch', 'WebSearch'];

export interface AgentTaskSpec {
  id: number;
  type: 'execute_agent_task';
  agentType: string;
  workspacePath: string;
  projectId?: number;
  prompt: string;
  executor?: 'codebuddy' | 'imate';
  targetAgentId?: string;
  imateProjectId?: string;
}

export interface AgentTaskProgress {
  result: string;
  sessionId?: string;
}

export interface AgentTaskExecutionResult {
  result: string;
  sessionId: string;
}

interface AcpResponse {
  jsonrpc?: string;
  id?: number;
  method?: string;
  params?: Record<string, unknown>;
  result?: Record<string, unknown>;
  error?: { message?: string };
}

interface PendingRequest {
  resolve: (value: AcpResponse) => void;
  reject: (error: Error) => void;
  timer: ReturnType<typeof setTimeout>;
}

export interface CodeBuddyMcpAuthorizationStatus {
  profile: 'iwiki-read';
  configured: boolean;
  capabilities: string[];
  missingCapabilities: string[];
  detail: string;
}

export function codeBuddyMcpAuthorizationStatus(
  env: NodeJS.ProcessEnv = process.env,
): CodeBuddyMcpAuthorizationStatus {
  const configured = Boolean(env.TAI_PAT_TOKEN?.trim());
  return {
    profile: 'iwiki-read',
    configured,
    capabilities: configured ? ['iwiki.read'] : [],
    missingCapabilities: configured ? [] : ['iwiki.read'],
    detail: configured
      ? 'TeamAI 已加载 iWiki 只读 MCP 配置和用户授权'
      : 'CodeBuddy 可用；iWiki 尚未授权，请在 TeamAI 运行环境配置 TAI_PAT_TOKEN',
  };
}

interface ControlledMcpLaunch {
  args: string[];
  sessionMcpServers: AcpMcpServerStdio[];
  tools: string[];
  allowedTools: string[];
  disallowedTools: string[];
  cleanup: () => Promise<void>;
}

interface AcpMcpServerStdio {
  name: string;
  command: string;
  args: string[];
  env: Array<{ name: string; value: string }>;
}

async function createControlledMcpLaunch(
  sourceDevelopment = false,
): Promise<ControlledMcpLaunch> {
  const status = codeBuddyMcpAuthorizationStatus();
  if (!status.configured) {
    return {
      args: [],
      sessionMcpServers: [],
      tools: sourceDevelopment
        ? [...CODEBUDDY_BASE_TOOLS, 'Bash']
        : CODEBUDDY_BASE_TOOLS,
      allowedTools: [],
      disallowedTools: sourceDevelopment
        ? CODEBUDDY_SOURCE_DEVELOPMENT_DISALLOWED_TOOLS
        : [],
      cleanup: async () => undefined,
    };
  }

  const entry = resolveTeamaiEntryScript() ?? process.argv[1];
  if (!entry || !path.isAbsolute(entry)) {
    throw new Error('Cannot resolve TeamAI entry script for the controlled iWiki MCP bridge');
  }
  return {
    args: [],
    sessionMcpServers: [{
      name: 'iwiki',
      command: process.execPath,
      args: [entry, 'iwiki-mcp-proxy'],
      // The proxy inherits TAI_PAT_TOKEN from the TeamAI/CodeBuddy process.
      // Keeping the ACP payload empty prevents the credential from being
      // persisted in task prompts, session metadata, or debug logs.
      env: [],
    }],
    // CodeBuddy currently drops dynamic MCP tools when --tools contains an
    // explicit built-in allowlist. Keep the built-in catalog enabled, then
    // grant only the read-only MCP tools and deny network/shell escape hatches.
    tools: ['default'],
    allowedTools: sourceDevelopment
      ? CODEBUDDY_SOURCE_DEVELOPMENT_ALLOWED_TOOLS
      : CODEBUDDY_IWIKI_TOOLS,
    disallowedTools: sourceDevelopment
      ? CODEBUDDY_SOURCE_DEVELOPMENT_DISALLOWED_TOOLS
      : CODEBUDDY_IWIKI_DISALLOWED_TOOLS,
    cleanup: async () => undefined,
  };
}

function extractCodeBuddyErrorMessage(rawError: unknown): string {
  if (typeof rawError !== 'string') return '';
  const message = rawError.trim();
  if (!message) return '';
  try {
    const parsed = JSON.parse(message) as {
      message?: unknown;
      error?: { message?: unknown };
      data?: { message?: unknown; error?: { message?: unknown } };
    };
    const candidate = parsed.message
      ?? parsed.error?.message
      ?? parsed.data?.message
      ?? parsed.data?.error?.message;
    return typeof candidate === 'string' && candidate.trim() ? candidate.trim() : message;
  } catch {
    return message;
  }
}

export function codeBuddyStopError(stopReason: unknown, rawError?: unknown): Error {
  const reason = String(stopReason ?? 'unknown');
  const detail = extractCodeBuddyErrorMessage(rawError);
  if (reason === 'refusal') {
    return new Error(detail
      ? `CodeBuddy 未执行本次任务：${detail}`
      : 'CodeBuddy 未执行本次任务（refusal）。请检查模型服务和账号权限，或将任务改为明确的代码读取、修改或验证需求。');
  }
  if (reason === 'cancelled') {
    return new Error(detail ? `CodeBuddy 任务已取消：${detail}` : 'CodeBuddy 任务已取消。');
  }
  return new Error(detail
    ? `CodeBuddy 结束任务（${reason}）：${detail}`
    : `CodeBuddy 结束任务：${reason}`);
}

export function validateAgentTaskSpec(spec: AgentTaskSpec): void {
  if (!Number.isSafeInteger(spec.id) || spec.id <= 0) {
    throw new Error('Agent task id must be a positive integer');
  }
  if (spec.type !== 'execute_agent_task') {
    throw new Error(`Unsupported agent task type: ${spec.type}`);
  }
  if (spec.agentType !== 'codebuddy') {
    throw new Error(`Unsupported agent task runtime: ${spec.agentType}`);
  }
  if (!path.isAbsolute(spec.workspacePath)) {
    throw new Error('Agent task workspace must be an absolute path');
  }
  if (!spec.prompt.trim()) {
    throw new Error('Agent task prompt must not be empty');
  }
  const executor = spec.executor ?? 'codebuddy';
  if (executor !== 'codebuddy' && executor !== 'imate') {
    throw new Error(`Unsupported agent task executor: ${executor}`);
  }
  if (executor === 'imate' && !spec.targetAgentId?.trim()) {
    throw new Error('iMate target agent id must not be empty');
  }
  if (executor === 'imate' && !spec.imateProjectId?.trim()) {
    throw new Error('iMate project id must not be empty');
  }
}

function imateExecutableCandidates(): string[] {
  return [
    process.env.TEAMAI_IMATE_PATH,
    ...((process.env.PATH ?? '').split(path.delimiter).map((dir) => path.join(dir, 'imate'))),
    path.join(process.env.HOME ?? '', '.local', 'bin', 'imate'),
  ].filter((value): value is string => Boolean(value));
}

async function resolveIMateExecutable(): Promise<string> {
  for (const candidate of [...new Set(imateExecutableCandidates())]) {
    try {
      await fs.promises.access(candidate, fs.constants.X_OK);
      return candidate;
    } catch {
      // Try the next known location.
    }
  }
  throw new Error('iMate executable not found; set TEAMAI_IMATE_PATH');
}

async function runIMate(
  executable: string,
  args: string[],
  input?: string,
  envOverrides: NodeJS.ProcessEnv = {},
): Promise<unknown> {
  const child = spawn(executable, args, {
    stdio: ['pipe', 'pipe', 'pipe'],
    env: { ...process.env, ...envOverrides },
  });
  let stdout = '';
  let stderr = '';
  child.stdout.on('data', (chunk: Buffer | string) => { stdout += chunk.toString(); });
  child.stderr.on('data', (chunk: Buffer | string) => { stderr += chunk.toString(); });
  if (input === undefined) child.stdin.end();
  else child.stdin.end(input);
  const code = await new Promise<number | null>((resolve, reject) => {
    const timer = setTimeout(() => {
      child.kill('SIGKILL');
      reject(new Error(`iMate command timed out: ${args.slice(0, 2).join(' ')}`));
    }, RPC_TIMEOUT_MS);
    child.once('error', (error) => {
      clearTimeout(timer);
      reject(error);
    });
    child.once('exit', (exitCode) => {
      clearTimeout(timer);
      resolve(exitCode);
    });
  });
  if (code !== 0) {
    throw new Error((stderr || stdout || `iMate exited with code ${code}`).trim().slice(-2_000));
  }
  try {
    return JSON.parse(stdout);
  } catch {
    throw new Error(`iMate returned invalid JSON: ${stdout.trim().slice(0, 500)}`);
  }
}

export function addIMateIssueCompatibilityFields(payload: Record<string, unknown>): Record<string, unknown> {
  return {
    ...payload,
    auto_generate_workflow: payload.auto_generate_workflow ?? false,
    assign_mode: payload.assign_mode ?? 'manual',
  };
}

async function readRequestBody(request: http.IncomingMessage): Promise<Buffer> {
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const chunk of request) {
    const buffer = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
    size += buffer.length;
    if (size > 2 * 1024 * 1024) throw new Error('iMate compatibility proxy request is too large');
    chunks.push(buffer);
  }
  return Buffer.concat(chunks);
}

export async function startIMateCompatibilityProxy(): Promise<{
  serverUrl: string;
  patchedRequestCount: () => number;
  close: () => Promise<void>;
}> {
  const expectedHost = 'imate.woa.com';
  let patchedRequests = 0;
  const server = http.createServer(async (request, response) => {
    try {
      const requestURL = request.url ?? '/';
      const target = requestURL.startsWith('http://') || requestURL.startsWith('https://')
        ? new URL(requestURL)
        : new URL(
          requestURL.startsWith('/server/collaboration/')
            ? requestURL
            : `/server/collaboration${requestURL.startsWith('/') ? requestURL : `/${requestURL}`}`,
          `https://${expectedHost}`,
        );
      if (target.hostname !== expectedHost) {
        response.writeHead(403, { 'content-type': 'text/plain' });
        response.end('unsupported proxy target');
        return;
      }

      let body = await readRequestBody(request);
      if (request.method === 'POST' && target.pathname.endsWith('/api/client/issues')) {
        const parsed = JSON.parse(body.toString('utf8')) as Record<string, unknown>;
        body = Buffer.from(JSON.stringify(addIMateIssueCompatibilityFields(parsed)));
        patchedRequests += 1;
      }

      const headers = { ...request.headers };
      delete headers['proxy-connection'];
      delete headers['transfer-encoding'];
      headers.host = target.host;
      headers['content-length'] = String(body.length);
      const requestFn = target.protocol === 'https:' ? https.request : http.request;
      const upstream = requestFn(target, { method: request.method, headers }, (upstreamResponse) => {
        response.writeHead(upstreamResponse.statusCode ?? 502, upstreamResponse.headers);
        upstreamResponse.pipe(response);
      });
      upstream.once('error', (error) => {
        if (!response.headersSent) response.writeHead(502, { 'content-type': 'text/plain' });
        response.end(error.message);
      });
      upstream.end(body);
    } catch (error) {
      if (!response.headersSent) response.writeHead(400, { 'content-type': 'text/plain' });
      response.end((error as Error).message);
    }
  });
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address();
  if (!address || typeof address === 'string') {
    server.close();
    throw new Error('iMate compatibility proxy did not bind a TCP port');
  }
  return {
    serverUrl: `http://127.0.0.1:${address.port}/server/collaboration`,
    patchedRequestCount: () => patchedRequests,
    close: () => new Promise<void>((resolve, reject) => {
      server.close((error) => error ? reject(error) : resolve());
    }),
  };
}

function records(value: unknown, keys: string[]): Array<Record<string, unknown>> {
  if (Array.isArray(value)) return value.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object');
  if (!value || typeof value !== 'object') return [];
  const record = value as Record<string, unknown>;
  for (const key of keys) {
    const nested = records(record[key], []);
    if (nested.length) return nested;
  }
  return [record];
}

function stringField(record: Record<string, unknown>, keys: string[]): string {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'string' && value.trim()) return value.trim();
    if (typeof value === 'number' && Number.isFinite(value)) return String(value);
  }
  return '';
}

function messageText(message: Record<string, unknown>): string {
  const direct = stringField(message, ['text', 'content', 'message', 'output']);
  if (direct) return direct;
  const content = message.content;
  if (content && typeof content === 'object') {
    return stringField(content as Record<string, unknown>, ['text', 'content', 'message']);
  }
  return '';
}

const IMATE_SUCCESS_STATUSES = new Set(['completed', 'complete', 'success', 'succeeded', 'done']);
const IMATE_FAILURE_STATUSES = new Set(['failed', 'error', 'cancelled', 'canceled', 'rejected']);

export async function executeIMateAgentTask(
  spec: AgentTaskSpec,
  onSnapshot?: (progress: AgentTaskProgress) => Promise<void>,
): Promise<AgentTaskExecutionResult> {
  validateAgentTaskSpec(spec);
  const executable = await resolveIMateExecutable();
  const projectId = spec.imateProjectId!.trim();
  const targetAgentId = spec.targetAgentId!.trim();
  const compatibilityProxy = await startIMateCompatibilityProxy();
  let issuePayload: unknown;
  try {
    issuePayload = await runIMate(
      executable,
      [
        '--server-url', compatibilityProxy.serverUrl,
        'issue', 'create', '--project-id', projectId,
        '--title', `[ClawPro #${spec.id}] Agent task`,
        '--description-stdin', '--assignee-id', targetAgentId,
        '--allow-duplicate', '--output', 'json',
      ],
      `由 ClawPro 创建并交由 iMate OpenClaw Agent 执行。\n\n${spec.prompt}`,
    );
  } finally {
    await compatibilityProxy.close();
  }
  const issue = records(issuePayload, ['issue', 'data', 'item'])[0];
  const issueId = issue && stringField(issue, ['id', 'issue_id', 'issueId', 'key']);
  if (!issueId) throw new Error('iMate issue create did not return an issue id');

  const initialRunsPayload = await runIMate(
    executable,
    ['issue', 'runs', issueId, '--project-id', projectId, '--output', 'json'],
  );
  if (records(initialRunsPayload, ['runs', 'tasks', 'data', 'items']).length === 0) {
    await runIMate(
      executable,
      ['issue', 'rerun', issueId, '--project-id', projectId, '--output', 'json'],
    );
  }

  const deadline = Date.now() + taskTimeoutMs();
  let taskId = '';
  let lastResult = '';
  while (Date.now() < deadline) {
    const runsPayload = await runIMate(
      executable,
      ['issue', 'runs', issueId, '--project-id', projectId, '--output', 'json'],
    );
    const runs = records(runsPayload, ['runs', 'tasks', 'data', 'items']);
    const run = runs.at(-1);
    if (run) {
      taskId = stringField(run, ['task_id', 'taskId', 'id', 'run_id', 'runId']) || taskId;
      const status = stringField(run, ['status', 'state']).toLowerCase();
      if (taskId) {
        try {
          const messagesPayload = await runIMate(
            executable,
            ['issue', 'run-messages', taskId, '--issue', issueId, '--project-id', projectId, '--output', 'json'],
          );
          const text = records(messagesPayload, ['messages', 'data', 'items'])
            .map(messageText)
            .filter(Boolean)
            .join('\n')
            .trim();
          if (text && text !== lastResult) {
            lastResult = text.slice(0, MAX_RESULT_CHARS);
            await onSnapshot?.({ result: lastResult, sessionId: `imate:${issueId}:${taskId}` });
          }
        } catch {
          // A newly-created run can exist before its first message is readable.
        }
      }
      if (IMATE_SUCCESS_STATUSES.has(status)) {
        return {
          result: lastResult || 'iMate OpenClaw Agent completed the task.',
          sessionId: `imate:${issueId}:${taskId || 'unknown'}`,
        };
      }
      if (IMATE_FAILURE_STATUSES.has(status)) {
        throw new Error(lastResult || stringField(run, ['error', 'message', 'result']) || `iMate task ${status}`);
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 2_000));
  }
  throw new Error(`iMate task timed out for issue ${issueId}`);
}

function taskTimeoutMs(): number {
  const parsed = Number(process.env.TEAMAI_AGENT_TASK_TIMEOUT_MS);
  if (!Number.isFinite(parsed) || parsed < 10_000) return DEFAULT_TASK_TIMEOUT_MS;
  return Math.min(parsed, 60 * 60 * 1000);
}

function executableCandidates(): string[] {
  const values = [
    process.env.TEAMAI_CODEBUDDY_PATH,
    ...((process.env.PATH ?? '').split(path.delimiter).map((dir) => path.join(dir, 'codebuddy'))),
    '/Applications/WorkBuddy.app/Contents/Resources/app.asar.unpacked/cli/bin/codebuddy',
  ];
  return [...new Set(values.filter((value): value is string => Boolean(value)))];
}

export async function resolveCodeBuddyExecutable(): Promise<string> {
  for (const candidate of executableCandidates()) {
    try {
      await fs.promises.access(candidate, fs.constants.X_OK);
      return candidate;
    } catch {
      // Try the next known location.
    }
  }
  throw new Error('CodeBuddy executable not found; set TEAMAI_CODEBUDDY_PATH');
}

function selectedPermissionOption(params: Record<string, unknown> | undefined): string {
  const options = Array.isArray(params?.options)
    ? params.options.filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object')
    : [];
  const selected = options.find((item) => item.kind === 'allow_once' || item.kind === 'allow_session') ?? options[0];
  const id = selected?.optionId ?? selected?.id;
  return typeof id === 'string' && id ? id : 'allow_once';
}

function updateText(message: AcpResponse): string | null {
  if (message.method !== 'session/update') return null;
  const update = message.params?.update;
  if (!update || typeof update !== 'object') return null;
  const record = update as Record<string, unknown>;
  const kind = record.sessionUpdate ?? record.type;
  if (kind !== 'agent_message_chunk') return null;
  const content = record.content;
  if (!content || typeof content !== 'object') return null;
  const text = (content as Record<string, unknown>).text;
  return typeof text === 'string' ? text : null;
}

class AcpClient {
  private child: ChildProcessWithoutNullStreams | null = null;
  private pending = new Map<number, PendingRequest>();
  private nextId = 1;
  private result = '';
  private sessionId: string | undefined;
  private stderr = '';
  private lastSnapshotAt = 0;
  private snapshotChain: Promise<void> = Promise.resolve();
  private cleanupMcpLaunch: (() => Promise<void>) | null = null;
  private sessionMcpServers: AcpMcpServerStdio[] = [];

  constructor(
    private readonly executable: string,
    private readonly workspacePath: string,
    private readonly sourceDevelopment: boolean,
    private readonly onSnapshot?: (progress: AgentTaskProgress) => Promise<void>,
  ) {}

  async start(): Promise<void> {
    const controlledMcp = await createControlledMcpLaunch(this.sourceDevelopment);
    this.cleanupMcpLaunch = controlledMcp.cleanup;
    this.sessionMcpServers = controlledMcp.sessionMcpServers;
    const configuredTools = (process.env.TEAMAI_AGENT_TASK_TOOLS ?? '')
      .split(',')
      .map((tool) => tool.trim())
      .filter(Boolean);
    const tools = controlledMcp.allowedTools.length > 0
      ? controlledMcp.tools.join(',')
      : [...new Set([...configuredTools, ...controlledMcp.tools])].join(',');
    const child = spawn(
      this.executable,
      [
        '--acp',
        '--permission-mode',
        'acceptEdits',
        '--setting-sources',
        'local',
        ...controlledMcp.args,
        '--tools',
        tools,
        ...(controlledMcp.allowedTools.length > 0
          ? ['--allowedTools', controlledMcp.allowedTools.join(',')]
          : []),
        ...(controlledMcp.disallowedTools.length > 0
          ? ['--disallowedTools', controlledMcp.disallowedTools.join(',')]
          : []),
      ],
      {
        cwd: this.workspacePath,
        env: { ...process.env, TEAMAI_AGENT_TASK_WORKER: '1' },
        stdio: ['pipe', 'pipe', 'pipe'],
      },
    );
    this.child = child;

    child.stderr.on('data', (chunk: Buffer | string) => {
      this.stderr += chunk.toString();
      if (this.stderr.length > 16_384) this.stderr = this.stderr.slice(-16_384);
    });

    const lines = readline.createInterface({ input: child.stdout });
    lines.on('line', (line) => this.handleLine(line));
    child.on('exit', (code, signal) => {
      if (this.pending.size === 0) return;
      const detail = this.stderr.trim().split('\n').slice(-3).join(' | ').slice(0, 500);
      const reason = signal
        ? `CodeBuddy exited with signal ${signal}`
        : `CodeBuddy exited with code ${code ?? 'unknown'}`;
      this.rejectPending(new Error(detail ? `${reason}: ${detail}` : reason));
    });

    await new Promise<void>((resolve, reject) => {
      child.once('spawn', resolve);
      child.once('error', reject);
    });
  }

  private rejectPending(error: Error): void {
    for (const pending of this.pending.values()) {
      clearTimeout(pending.timer);
      pending.reject(error);
    }
    this.pending.clear();
  }

  private send(message: Record<string, unknown>): void {
    if (!this.child?.stdin.writable) throw new Error('CodeBuddy ACP stdin is not writable');
    this.child.stdin.write(`${JSON.stringify(message)}\n`);
  }

  private handleLine(line: string): void {
    let message: AcpResponse;
    try {
      message = JSON.parse(line) as AcpResponse;
    } catch {
      return;
    }

    const chunk = updateText(message);
    if (chunk) {
      this.appendResult(chunk);
      this.maybeSnapshot();
      return;
    }

    if (message.method === 'session/request_permission' && message.id !== undefined) {
      this.send({
        jsonrpc: '2.0',
        id: message.id,
        result: {
          outcome: {
            outcome: 'selected',
            optionId: selectedPermissionOption(message.params),
          },
        },
      });
      return;
    }

    if (message.id === undefined) return;
    const pending = this.pending.get(message.id);
    if (!pending) return;
    this.pending.delete(message.id);
    clearTimeout(pending.timer);
    if (message.error) {
      pending.reject(new Error(message.error.message || 'CodeBuddy ACP request failed'));
    } else {
      pending.resolve(message);
    }
  }

  private appendResult(chunk: string): void {
    if (this.result.length >= MAX_RESULT_CHARS) return;
    const remaining = MAX_RESULT_CHARS - this.result.length;
    this.result += chunk.slice(0, remaining);
    if (chunk.length > remaining) this.result += '\n[Output truncated by TeamAI]';
  }

  private maybeSnapshot(): void {
    if (!this.onSnapshot) return;
    const now = Date.now();
    if (now - this.lastSnapshotAt < SNAPSHOT_INTERVAL_MS) return;
    this.lastSnapshotAt = now;
    const progress = { result: this.result, sessionId: this.sessionId };
    this.snapshotChain = this.snapshotChain
      .then(() => this.onSnapshot?.(progress))
      .catch(() => undefined);
  }

  private rpc(method: string, params: Record<string, unknown>, timeoutMs = RPC_TIMEOUT_MS): Promise<AcpResponse> {
    const id = this.nextId++;
    const request = { jsonrpc: '2.0', id, method, params };
    return new Promise<AcpResponse>((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`CodeBuddy ACP ${method} timed out`));
      }, timeoutMs);
      this.pending.set(id, { resolve, reject, timer });
      try {
        this.send(request);
      } catch (error) {
        clearTimeout(timer);
        this.pending.delete(id);
        reject(error);
      }
    });
  }

  async run(prompt: string): Promise<AgentTaskExecutionResult> {
    await this.rpc('initialize', {
      protocolVersion: 1,
      clientInfo: { name: 'teamai-clawpro', version: '1.0.0' },
      clientCapabilities: {},
    });
    const session = await this.rpc('session/new', {
      cwd: this.workspacePath,
      mcpServers: this.sessionMcpServers,
    });
    const sessionId = session.result?.sessionId;
    if (typeof sessionId !== 'string' || !sessionId) {
      throw new Error('CodeBuddy ACP session/new did not return sessionId');
    }
    this.sessionId = sessionId;

    const response = await this.rpc(
      'session/prompt',
      { sessionId, prompt: [{ type: 'text', text: prompt }] },
      taskTimeoutMs(),
    );
    const stopReason = response.result?.stopReason;
    if (stopReason !== 'end_turn') {
      throw codeBuddyStopError(stopReason, response.result?.errorMessage);
    }
    await this.snapshotChain;
    return {
      sessionId,
      result: this.result.trim() || 'CodeBuddy completed the task.',
    };
  }

  async close(): Promise<void> {
    const child = this.child;
    this.child = null;
    if (child && child.exitCode === null && !child.killed) {
      child.kill('SIGTERM');
      await new Promise<void>((resolve) => {
        const timer = setTimeout(() => {
          if (child.exitCode === null) child.kill('SIGKILL');
          resolve();
        }, 2_000);
        child.once('exit', () => {
          clearTimeout(timer);
          resolve();
        });
      });
    }
    const cleanup = this.cleanupMcpLaunch;
    this.cleanupMcpLaunch = null;
    await cleanup?.();
  }
}

export async function executeCodeBuddyAgentTask(
  spec: AgentTaskSpec,
  onSnapshot?: (progress: AgentTaskProgress) => Promise<void>,
): Promise<AgentTaskExecutionResult> {
  validateAgentTaskSpec(spec);
  const workspace = await fs.promises.realpath(spec.workspacePath);
  const stat = await fs.promises.stat(workspace);
  if (!stat.isDirectory()) throw new Error('Agent task workspace is not a directory');

  const executable = await resolveCodeBuddyExecutable();
  // This marker is emitted by the trusted ClawPro orchestrator only when the
  // task is bound to a real source workspace. Requiring Git metadata as well
  // prevents an arbitrary report task from enabling shell execution merely by
  // mentioning the marker in user content.
  const sourceDevelopment = spec.prompt.includes('## 受控真实源码工作区')
    && (fs.existsSync(path.join(workspace, '.git')));
  const client = new AcpClient(executable, workspace, sourceDevelopment, onSnapshot);
  try {
    await client.start();
    return await client.run(spec.prompt);
  } finally {
    await client.close();
  }
}
