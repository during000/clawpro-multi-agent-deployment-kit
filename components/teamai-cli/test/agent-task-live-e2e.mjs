import fs from 'node:fs/promises';
import http from 'node:http';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawn } from 'node:child_process';

const root = fileURLToPath(new URL('..', import.meta.url));
const tempRoot = await fs.mkdtemp(path.join(os.tmpdir(), 'teamai-agent-task-live-'));
const localAgentHome = path.join(tempRoot, 'local-agent');
const workspace = path.join(tempRoot, 'workspace');
const proofPath = path.join(workspace, 'clawpro-teamai-proof.txt');
const expected = 'ClawPro -> TeamAI -> CodeBuddy ACP works';
const acknowledgements = [];

await fs.mkdir(localAgentHome, { recursive: true });
await fs.mkdir(workspace, { recursive: true });

const server = http.createServer(async (request, response) => {
  let raw = '';
  for await (const chunk of request) raw += chunk;
  if (request.url === '/local-agent/commands/ack' && request.method === 'POST') {
    acknowledgements.push(JSON.parse(raw));
    response.writeHead(200, { 'Content-Type': 'application/json' });
    response.end(JSON.stringify({ ok: true }));
    return;
  }
  response.writeHead(404, { 'Content-Type': 'application/json' });
  response.end(JSON.stringify({ error: 'not found' }));
});

await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
const address = server.address();
if (!address || typeof address === 'string') throw new Error('Failed to start mock backend');

await fs.writeFile(path.join(localAgentHome, 'config.json'), JSON.stringify({
  endpoint: `http://127.0.0.1:${address.port}`,
  token: 'local-e2e-token',
  createdAt: new Date().toISOString(),
  routes: { ack: '/local-agent/commands/ack' },
  workspaceBindings: {
    [workspace]: {
      projectId: 42,
      projectName: 'agent-task-live-e2e',
      boundAt: new Date().toISOString(),
      ideType: 'codebuddy',
    },
  },
}, null, 2), { mode: 0o600 });

const payload = {
  command: {
    id: 9001,
    type: 'execute_agent_task',
    agent_type: 'codebuddy',
    project_id: 42,
    workspace_path: workspace,
    prompt: `Create a file named clawpro-teamai-proof.txt in the current workspace. Its complete contents must be exactly: ${expected}`,
  },
  context: { tool: 'codebuddy', cwd: workspace },
};

try {
  const child = spawn(process.execPath, [path.join(root, 'dist/index.js'), 'source', 'execute-agent-task'], {
    cwd: root,
    env: {
      ...process.env,
      TEAMAI_LOCAL_AGENT_HOME: localAgentHome,
      TEAMAI_AGENT_TASK_TIMEOUT_MS: '180000',
    },
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  child.stdin.end(JSON.stringify(payload));
  let stdout = '';
  let stderr = '';
  child.stdout.on('data', (chunk) => { stdout += chunk.toString(); });
  child.stderr.on('data', (chunk) => { stderr += chunk.toString(); });

  const exitCode = await new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      child.kill('SIGKILL');
      reject(new Error('Live Agent task timed out after 180 seconds'));
    }, 180000);
    child.once('error', reject);
    child.once('exit', (code) => {
      clearTimeout(timer);
      resolve(code);
    });
  });
  if (exitCode !== 0) {
    throw new Error(`TeamAI worker exited ${exitCode}: ${(stderr || stdout).trim().slice(-1000)}`);
  }

  const proof = (await fs.readFile(proofPath, 'utf8')).trim();
  if (proof !== expected) throw new Error(`Unexpected proof contents: ${proof}`);
  const statuses = acknowledgements.map((item) => item.status);
  if (statuses[0] !== 'running' || statuses.at(-1) !== 'success') {
    throw new Error(`Unexpected acknowledgement sequence: ${statuses.join(',')}`);
  }
  const success = acknowledgements.at(-1);
  if (!success.session_id || !String(success.result ?? '').trim()) {
    throw new Error('Success acknowledgement is missing session_id or result');
  }
  process.stdout.write(JSON.stringify({
    ok: true,
    protocol: 'ACP JSON-RPC stdio',
    statuses,
    sessionId: success.session_id,
    proof,
  }, null, 2) + '\n');
} finally {
  await new Promise((resolve) => server.close(resolve));
  await fs.rm(tempRoot, { recursive: true, force: true });
}
