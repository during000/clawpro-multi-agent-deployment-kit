import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawn } from 'node:child_process';
import fse from 'fs-extra';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

let tempHome: string;
let workspacePath: string;
let originalHome: string | undefined;
let originalCodeBuddyPath: string | undefined;
let originalIMatePath: string | undefined;
let originalTaiPatToken: string | undefined;
let originalCapturePath: string | undefined;
let originalAgentTaskTools: string | undefined;

async function installFakeCodeBuddy(): Promise<string> {
  const executable = path.join(tempHome, 'fake-codebuddy');
const source = `#!${process.execPath}
const fs = require('node:fs');
const readline = require('node:readline');
const args = process.argv.slice(2);
const configIndex = args.indexOf('--mcp-config');
const capturePath = process.env.FAKE_CODEBUDDY_CAPTURE_PATH;
let capture = {
  args,
  config: configIndex >= 0 ? fs.readFileSync(args[configIndex + 1], 'utf8') : '',
  sessionNewParams: null,
};
const writeCapture = () => {
  if (capturePath) fs.writeFileSync(capturePath, JSON.stringify(capture));
};
if (process.env.FAKE_CODEBUDDY_CAPTURE_PATH) {
  writeCapture();
}
const lines = readline.createInterface({ input: process.stdin });
let promptRequestId = null;
const send = (value) => process.stdout.write(JSON.stringify(value) + '\\n');
lines.on('line', (line) => {
  const message = JSON.parse(line);
  if (message.method === 'initialize') {
    send({ jsonrpc: '2.0', id: message.id, result: { protocolVersion: 1 } });
  } else if (message.method === 'session/new') {
    capture.sessionNewParams = message.params;
    writeCapture();
    send({ jsonrpc: '2.0', id: message.id, result: { sessionId: 'fake-session-1' } });
  } else if (message.method === 'session/prompt') {
    promptRequestId = message.id;
    send({
      jsonrpc: '2.0',
      id: 99,
      method: 'session/request_permission',
      params: { options: [{ optionId: 'allow-once', kind: 'allow_once' }] },
    });
  } else if (message.id === 99 && promptRequestId !== null) {
    send({
      jsonrpc: '2.0',
      method: 'session/update',
      params: {
        update: {
          sessionUpdate: 'agent_message_chunk',
          content: { type: 'text', text: 'Created the requested file.' },
        },
      },
    });
    send({ jsonrpc: '2.0', id: promptRequestId, result: { stopReason: 'end_turn' } });
  }
});
`;
  await fs.promises.writeFile(executable, source, { mode: 0o755 });
  return executable;
}

async function installFakeIMate(): Promise<string> {
  const executable = path.join(tempHome, 'fake-imate');
const source = `#!${process.execPath}
const fs = require('node:fs');
const path = require('node:path');
let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (chunk) => { input += chunk; });
process.stdin.on('end', () => {
  const rawArgs = process.argv.slice(2);
  const args = rawArgs[0] === '--server-url' ? rawArgs.slice(2) : rawArgs;
  const rerunMarker = path.join(process.env.HOME, '.fake-imate-rerun');
  if (args[0] === 'issue' && args[1] === 'create') {
    process.stdout.write(JSON.stringify({ id: 'ISSUE-1' }));
  } else if (args[0] === 'issue' && args[1] === 'rerun') {
    fs.writeFileSync(rerunMarker, '1');
    process.stdout.write(JSON.stringify({ id: 'RUN-1', status: 'queued' }));
  } else if (args[0] === 'issue' && args[1] === 'runs') {
    process.stdout.write(JSON.stringify(fs.existsSync(rerunMarker)
      ? [{ task_id: 'RUN-1', status: 'completed' }]
      : []));
  } else if (args[0] === 'issue' && args[1] === 'run-messages') {
    process.stdout.write(JSON.stringify([{ seq: 1, role: 'assistant', content: 'OpenClaw completed from iMate.' }]));
  } else {
    process.stderr.write('unexpected arguments: ' + args.join(' '));
    process.exitCode = 2;
  }
});
`;
  await fs.promises.writeFile(executable, source, { mode: 0o755 });
  return executable;
}

beforeEach(async () => {
  tempHome = await fse.mkdtemp(path.join(os.tmpdir(), 'teamai-agent-task-'));
  workspacePath = path.join(tempHome, 'workspace');
  await fse.ensureDir(workspacePath);
  workspacePath = await fs.promises.realpath(workspacePath);

  originalHome = process.env.HOME;
  originalCodeBuddyPath = process.env.TEAMAI_CODEBUDDY_PATH;
  originalIMatePath = process.env.TEAMAI_IMATE_PATH;
  originalTaiPatToken = process.env.TAI_PAT_TOKEN;
  originalCapturePath = process.env.FAKE_CODEBUDDY_CAPTURE_PATH;
  originalAgentTaskTools = process.env.TEAMAI_AGENT_TASK_TOOLS;
  process.env.HOME = tempHome;
  process.env.TEAMAI_CODEBUDDY_PATH = await installFakeCodeBuddy();
  process.env.TEAMAI_IMATE_PATH = await installFakeIMate();

  const configDir = path.join(tempHome, '.teamai', 'local-agent');
  await fse.ensureDir(configDir);
  await fse.writeJson(path.join(configDir, 'config.json'), {
    endpoint: 'https://clawpro.example.test',
    token: 'test-token',
    createdAt: new Date().toISOString(),
    workspaceBindings: {
      [workspacePath]: {
        projectId: 42,
        projectName: 'demo',
        boundAt: new Date().toISOString(),
        ideType: 'codebuddy',
      },
    },
  });
});

afterEach(async () => {
  process.env.HOME = originalHome;
  if (originalCodeBuddyPath === undefined) delete process.env.TEAMAI_CODEBUDDY_PATH;
  else process.env.TEAMAI_CODEBUDDY_PATH = originalCodeBuddyPath;
  if (originalIMatePath === undefined) delete process.env.TEAMAI_IMATE_PATH;
  else process.env.TEAMAI_IMATE_PATH = originalIMatePath;
  if (originalTaiPatToken === undefined) delete process.env.TAI_PAT_TOKEN;
  else process.env.TAI_PAT_TOKEN = originalTaiPatToken;
  if (originalCapturePath === undefined) delete process.env.FAKE_CODEBUDDY_CAPTURE_PATH;
  else process.env.FAKE_CODEBUDDY_CAPTURE_PATH = originalCapturePath;
  if (originalAgentTaskTools === undefined) delete process.env.TEAMAI_AGENT_TASK_TOOLS;
  else process.env.TEAMAI_AGENT_TASK_TOOLS = originalAgentTaskTools;
  vi.unstubAllGlobals();
  await fse.remove(tempHome);
});

describe('ClawPro Agent task worker', () => {
  it('shows the CodeBuddy error detail when ACP ends with refusal', async () => {
    const { codeBuddyStopError } = await import('../agent-task-executor.js');
    expect(codeBuddyStopError('refusal', JSON.stringify({
      error: { message: '当前模型服务暂不可用' },
    })).message).toBe('CodeBuddy 未执行本次任务：当前模型服务暂不可用');
    expect(codeBuddyStopError('refusal').message).not.toContain('stopped unexpectedly');
  });

  it('adds the fields required by the current iMate issue API', async () => {
    const { addIMateIssueCompatibilityFields } = await import('../agent-task-executor.js');
    expect(addIMateIssueCompatibilityFields({ title: 'demo' })).toEqual({
      title: 'demo',
      auto_generate_workflow: false,
      assign_mode: 'manual',
    });
    expect(addIMateIssueCompatibilityFields({
      title: 'demo', auto_generate_workflow: true, assign_mode: 'pm_auto',
    })).toMatchObject({ auto_generate_workflow: true, assign_mode: 'pm_auto' });
  });

  it.skipIf(process.env.TEAMAI_IMATE_LIVE_COMPAT !== '1')(
    'patches a real iMate create request before it reaches the server',
    async () => {
      const { startIMateCompatibilityProxy } = await import('../agent-task-executor.js');
      const proxy = await startIMateCompatibilityProxy();
      try {
        const child = spawn(
          process.env.TEAMAI_IMATE_PATH ?? 'imate',
          [
            '--server-url', proxy.serverUrl,
            'issue', 'create',
            '--project-id', '00000000-0000-0000-0000-000000000001',
            '--title', 'ClawPro compatibility validation',
            '--description-stdin', '--output', 'json',
          ],
          {
            stdio: ['pipe', 'pipe', 'pipe'],
            env: {
              ...process.env,
              HOME: originalHome,
            },
          },
        );
        child.stdin.end('Validation uses a nonexistent project and cannot create an issue.');
        let output = '';
        child.stdout.on('data', (chunk) => { output += chunk.toString(); });
        child.stderr.on('data', (chunk) => { output += chunk.toString(); });
        const code = await new Promise<number | null>((resolve, reject) => {
          child.once('error', reject);
          child.once('exit', resolve);
        });
        expect(code).not.toBe(0);
        expect(proxy.patchedRequestCount()).toBe(1);
        expect(output).toContain('project not found');
        expect(output).not.toContain('authorization required');
      } finally {
        await proxy.close();
      }
    },
    30_000,
  );

  it('runs CodeBuddy through ACP and reports running snapshots and success', async () => {
    const acknowledgements: Array<Record<string, unknown>> = [];
    vi.stubGlobal('fetch', vi.fn(async (_url: string, init?: RequestInit) => {
      acknowledgements.push(JSON.parse(String(init?.body)) as Record<string, unknown>);
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    }));

    const { runAgentTaskWorker } = await import('../local-agent.js');
    await runAgentTaskWorker(
      {
        id: 501,
        type: 'execute_agent_task',
        agent_type: 'codebuddy',
        project_id: 42,
        workspace_path: workspacePath,
        prompt: 'Create hello.txt',
      },
      { tool: 'codebuddy', cwd: workspacePath },
    );

    expect(acknowledgements.map((ack) => ack.status)).toEqual(['running', 'running', 'success']);
    expect(acknowledgements.at(-1)).toMatchObject({
      id: 501,
      type: 'execute_agent_task',
      status: 'success',
      result: 'Created the requested file.',
      session_id: 'fake-session-1',
    });
  });

  it('uploads the full declared artifact instead of replacing it with the agent summary', async () => {
    const artifactPath = path.join(
      workspacePath,
      'server',
      'artifacts',
      'contest-school-registry',
      '01-requirement',
      'requirement-report.md',
    );
    const artifactContent = '# 完整需求分析\n\n' + '验收标准与边界。\n'.repeat(80);
    await fse.outputFile(artifactPath, artifactContent);
    await fse.outputJson(path.join(workspacePath, 'workflow-state.json'), {
      task_slug: 'contest-school-registry',
    });
    const acknowledgements: Array<Record<string, unknown>> = [];
    vi.stubGlobal('fetch', vi.fn(async (_url: string, init?: RequestInit) => {
      acknowledgements.push(JSON.parse(String(init?.body)) as Record<string, unknown>);
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    }));

    const { runAgentTaskWorker } = await import('../local-agent.js');
    await runAgentTaskWorker(
      {
        id: 508,
        type: 'execute_agent_task',
        agent_type: 'codebuddy',
        project_id: 42,
        workspace_path: workspacePath,
        prompt: '## 本节点必须产出\n- 01-requirement/requirement-report.md\n\n## 已授权能力\n无',
      },
      { tool: 'codebuddy', cwd: workspacePath },
    );

    const finalResult = String(acknowledgements.at(-1)?.result ?? '');
    const encoded = finalResult.match(
      /<clawpro_artifact_bundle_v1>\n(.+)\n<\/clawpro_artifact_bundle_v1>/,
    )?.[1];
    expect(encoded).toBeTruthy();
    const bundle = JSON.parse(encoded!) as {
      artifacts: Array<{ path: string; source_path: string; size: number; content_base64: string }>;
    };
    expect(bundle.artifacts).toHaveLength(1);
    expect(bundle.artifacts[0]).toMatchObject({
      path: '01-requirement/requirement-report.md',
      source_path: 'server/artifacts/contest-school-registry/01-requirement/requirement-report.md',
      size: Buffer.byteLength(artifactContent),
    });
    expect(Buffer.from(bundle.artifacts[0].content_base64, 'base64').toString()).toBe(artifactContent);
  });

  it('injects a controlled read-only MCP into the ACP session without persisting the PAT', async () => {
    const capturePath = path.join(tempHome, 'codebuddy-launch.json');
    process.env.TAI_PAT_TOKEN = 'user-secret-pat';
    process.env.TEAMAI_AGENT_TASK_TOOLS = 'Read,Write';
    process.env.FAKE_CODEBUDDY_CAPTURE_PATH = capturePath;
    vi.stubGlobal('fetch', vi.fn(async () => (
      new Response(JSON.stringify({ ok: true }), { status: 200 })
    )));

    const { runAgentTaskWorker } = await import('../local-agent.js');
    await runAgentTaskWorker(
      {
        id: 506,
        type: 'execute_agent_task',
        agent_type: 'codebuddy',
        project_id: 42,
        workspace_path: workspacePath,
        prompt: 'Read iWiki metadata',
      },
      { tool: 'codebuddy', cwd: workspacePath },
    );

    const capture = await fse.readJson(capturePath) as {
      args: string[];
      config: string;
      sessionNewParams: {
        cwd: string;
        mcpServers: Array<{
          name: string;
          command: string;
          args: string[];
          env: Array<{ name: string; value: string }>;
        }>;
      };
    };
    expect(capture.args).not.toContain('--mcp-config');
    expect(capture.args).not.toContain('--strict-mcp-config');
    const toolsIndex = capture.args.indexOf('--tools');
    expect(capture.args[toolsIndex + 1]).toBe('default');
    const allowedToolsIndex = capture.args.indexOf('--allowedTools');
    expect(capture.args[allowedToolsIndex + 1]).toContain('mcp__iwiki__metadata');
    expect(capture.args[allowedToolsIndex + 1]).toContain('mcp__iwiki__getDocument');
    const disallowedToolsIndex = capture.args.indexOf('--disallowedTools');
    expect(capture.args[disallowedToolsIndex + 1]).toContain('Bash');
    expect(capture.sessionNewParams.cwd).toBe(workspacePath);
    expect(capture.sessionNewParams.mcpServers).toEqual([
      expect.objectContaining({
        name: 'iwiki',
        command: process.execPath,
        args: expect.arrayContaining(['iwiki-mcp-proxy']),
        env: [],
      }),
    ]);
    expect(JSON.stringify(capture)).not.toContain('user-secret-pat');
  });

  it('allows controlled Bash for a source-development task while keeping public network tools disabled', async () => {
    const capturePath = path.join(tempHome, 'codebuddy-source-development-launch.json');
    process.env.TAI_PAT_TOKEN = 'user-secret-pat';
    process.env.FAKE_CODEBUDDY_CAPTURE_PATH = capturePath;
    await fs.promises.writeFile(path.join(workspacePath, '.git'), 'gitdir: /tmp/fake-worktree\n');
    vi.stubGlobal('fetch', vi.fn(async () => (
      new Response(JSON.stringify({ ok: true }), { status: 200 })
    )));

    const { runAgentTaskWorker } = await import('../local-agent.js');
    await runAgentTaskWorker(
      {
        id: 507,
        type: 'execute_agent_task',
        agent_type: 'codebuddy',
        project_id: 42,
        workspace_path: workspacePath,
        prompt: '## 受控真实源码工作区\n请检查 Git 并运行构建。',
      },
      { tool: 'codebuddy', cwd: workspacePath },
    );

    const capture = await fse.readJson(capturePath) as {
      args: string[];
      sessionNewParams: { mcpServers: unknown[] };
    };
    const allowedToolsIndex = capture.args.indexOf('--allowedTools');
    expect(capture.args[allowedToolsIndex + 1]).toContain('Bash');
    const disallowedToolsIndex = capture.args.indexOf('--disallowedTools');
    expect(capture.args[disallowedToolsIndex + 1]).not.toContain('Bash');
    expect(capture.args[disallowedToolsIndex + 1]).toContain('WebFetch');
    expect(capture.args[disallowedToolsIndex + 1]).toContain('WebSearch');
  });

  it('allows controlled Bash for source development when no MCP authorization is configured', async () => {
    const capturePath = path.join(tempHome, 'codebuddy-source-development-no-mcp.json');
    delete process.env.TAI_PAT_TOKEN;
    process.env.FAKE_CODEBUDDY_CAPTURE_PATH = capturePath;
    await fs.promises.writeFile(path.join(workspacePath, '.git'), 'gitdir: /tmp/fake-worktree\n');
    vi.stubGlobal('fetch', vi.fn(async () => (
      new Response(JSON.stringify({ ok: true }), { status: 200 })
    )));

    const { runAgentTaskWorker } = await import('../local-agent.js');
    await runAgentTaskWorker(
      {
        id: 508,
        type: 'execute_agent_task',
        agent_type: 'codebuddy',
        project_id: 42,
        workspace_path: workspacePath,
        prompt: '## 受控真实源码工作区\n请运行构建。',
      },
      { tool: 'codebuddy', cwd: workspacePath },
    );

    const capture = await fse.readJson(capturePath) as {
      args: string[];
      sessionNewParams: { mcpServers: unknown[] };
    };
    const toolsIndex = capture.args.indexOf('--tools');
    expect(capture.args[toolsIndex + 1]).toContain('Bash');
    const disallowedToolsIndex = capture.args.indexOf('--disallowedTools');
    expect(capture.args[disallowedToolsIndex + 1]).toContain('WebFetch');
    expect(capture.args[disallowedToolsIndex + 1]).toContain('WebSearch');
    expect(capture.args).not.toContain('--strict-mcp-config');
    expect(capture.sessionNewParams.mcpServers).toEqual([]);
  });

  it('rejects a task whose project does not match the local binding', async () => {
    const acknowledgements: Array<Record<string, unknown>> = [];
    vi.stubGlobal('fetch', vi.fn(async (_url: string, init?: RequestInit) => {
      acknowledgements.push(JSON.parse(String(init?.body)) as Record<string, unknown>);
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    }));

    const { runAgentTaskWorker } = await import('../local-agent.js');
    await expect(runAgentTaskWorker(
      {
        id: 502,
        type: 'execute_agent_task',
        agent_type: 'codebuddy',
        project_id: 99,
        workspace_path: workspacePath,
        prompt: 'Create hello.txt',
      },
      { tool: 'codebuddy', cwd: workspacePath },
    )).rejects.toThrow('project does not match');

    expect(acknowledgements).toHaveLength(1);
    expect(acknowledgements[0]).toMatchObject({ id: 502, status: 'failed' });
  });

  it('creates an iMate shadow issue and reports the OpenClaw result', async () => {
    const acknowledgements: Array<Record<string, unknown>> = [];
    vi.stubGlobal('fetch', vi.fn(async (_url: string, init?: RequestInit) => {
      acknowledgements.push(JSON.parse(String(init?.body)) as Record<string, unknown>);
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    }));

    const { runAgentTaskWorker } = await import('../local-agent.js');
    await runAgentTaskWorker(
      {
        id: 503,
        type: 'execute_agent_task',
        agent_type: 'codebuddy',
        project_id: 42,
        workspace_path: workspacePath,
        prompt: 'Summarize the project',
        executor: 'imate',
        target_agent_id: 'agent-openclaw-1',
        imate_project_id: 'project-imate-1',
      },
      { tool: 'codebuddy', cwd: workspacePath },
    );

    expect(acknowledgements.map((ack) => ack.status)).toEqual(['running', 'running', 'success']);
    expect(acknowledgements.at(-1)).toMatchObject({
      id: 503,
      type: 'execute_agent_task',
      status: 'success',
      result: 'OpenClaw completed from iMate.',
      session_id: 'imate:ISSUE-1:RUN-1',
    });
  });
});
