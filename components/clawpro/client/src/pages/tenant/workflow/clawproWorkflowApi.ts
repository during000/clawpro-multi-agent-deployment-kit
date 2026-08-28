export type WorkflowAssignmentMode = "shared" | "mixed";
export type WorkflowDeliveryMode = "wss" | "hook";

export interface IMateOpenClawAgent {
  id: string;
  name: string;
  status: string;
  online_status: string;
}

export interface WorkflowRuntimeAgent {
  id: string;
  name: string;
  platform: "codebuddy" | "imate" | "cloudagent";
  location: "local" | "cloud";
  status: "online" | "offline";
  runtimeId: string;
  deviceId: string;
  targetAgentId?: string;
  detail: string;
  capabilities: string[];
  missingCapabilities: string[];
}

export interface WorkflowRuntimePhase {
  id: string;
  title: string;
  agent_id: string;
  agent_instance_id: string;
  project_agent_id?: string;
  device_id?: string;
  transport?: string;
  target_agent_id?: string | null;
  runtime_id: string;
  status: string;
  depends_on?: string[];
  on_pass?: string | null;
  on_fail?: string | null;
  artifacts: string[];
  approval_required?: boolean;
  session_id?: string | null;
}

export interface WorkflowRuntimeArtifact {
  artifact_id?: string;
  path: string;
  media_type?: string;
  size?: number;
  sha256?: string;
  lineage?: string[];
  version?: number;
  producer_node?: string;
}

export interface WorkflowPendingApproval {
  gate_id: string;
  title: string;
  description: string;
  action_label: string;
  status: string;
  after_phase_id: string;
  after_phase_title: string;
  next_phase_id: string;
  requested_at: string;
  summary?: string;
  artifacts?: WorkflowRuntimeArtifact[];
}

export interface WorkflowRuntimeTask {
  task_id: string;
  workflow_id?: string;
  runtime_id: string;
  status: string;
  execution_status: string;
  cancellable?: boolean;
  cancel_requested?: boolean;
  workflow_stage?: string;
  workflow_current_phase?: string | null;
  workflow_current_phases?: string[];
  workflow_phases?: WorkflowRuntimePhase[];
  agent_assignment_mode?: WorkflowAssignmentMode;
  agent_runtime_id?: string;
  target_agent_id?: string | null;
  imate_project_id?: string | null;
  handoff_contract?: string;
  pending_approval?: WorkflowPendingApproval | null;
  available_artifacts?: WorkflowRuntimeArtifact[];
  agent_instance_count?: number | null;
  agent_session_count?: number | null;
  handoff_count?: number | null;
  agent_output?: string;
  created_at: string;
  updated_at: string;
}

export interface WorkflowRuntimeEvent {
  event_id: string;
  seq: number;
  type: string;
  title: string;
  detail: string;
  timestamp: string;
  payload?: Record<string, unknown>;
}

export interface WorkflowNodeAssignment {
  phaseId: string;
  projectAgentId: string;
  platform: "codebuddy" | "imate" | "cloudagent";
  location: "local" | "cloud";
  targetAgentId?: string;
}

export interface WorkflowDefinitionNode {
  id: string;
  title: string;
  agentId: string;
  dependsOn: string[];
  prompt: string;
  inputs: Array<{
    key: string;
    label: string;
    type: string;
    required?: boolean;
    source?: { nodeId: string; outputKey: string };
  }>;
  artifacts: string[];
  optionalArtifacts?: string[];
  configAssets?: Array<{
    id: string;
    name: string;
    version: string;
    type: "rules" | "skill" | "contract";
    summary: string;
    source: string;
    content: string;
  }>;
  approvalRequired: boolean;
  requiredEvidence?: string[];
  rejectOutputMarkers?: string[];
  requiredCapabilities?: string[];
  onPass?: string | null;
  onFail?: string | null;
  decisionMode?: "review_verdict" | "size_class";
  maxRetries?: number;
}

export interface WorkflowTaskInputValue {
  type: string;
  value: string;
}

interface ApiErrorBody {
  error?: string;
  code?: string;
}

export class WorkflowApiError extends Error {
  readonly status: number;
  readonly code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = "WorkflowApiError";
    this.status = status;
    this.code = code;
  }
}

export function isWorkflowTaskNotFoundError(error: unknown): boolean {
  return (
    error instanceof WorkflowApiError &&
    error.status === 404 &&
    (!error.code || error.code === "TASK_NOT_FOUND")
  );
}

const configuredBase = (
  import.meta.env.VITE_CLAWPRO_WORKFLOW_API_BASE ?? ""
).trim();
const API_BASE = configuredBase.replace(/\/$/, "");

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        ...init?.headers,
      },
    });
  } catch {
    throw new WorkflowApiError(
      "无法连接 ClawPro 编排服务，请检查网络后刷新重试",
      0,
      "NETWORK_ERROR"
    );
  }
  if (!response.ok) {
    let detail = `请求失败（${response.status}）`;
    let code: string | undefined;
    try {
      const body = (await response.json()) as ApiErrorBody;
      if (body.error) detail = body.error;
      code = body.code;
    } catch {
      // Keep the HTTP fallback when the server does not return JSON.
    }
    throw new WorkflowApiError(detail, response.status, code);
  }
  return response.json() as Promise<T>;
}

export async function listIMateOpenClawAgents() {
  const data = await request<{ agents: IMateOpenClawAgent[] }>(
    "/api/imate/agents"
  );
  return data.agents;
}

export async function listWorkflowRuntimeAgents(): Promise<WorkflowRuntimeAgent[]> {
  const [runtimeData, imateAgents, cloudAgentData] = await Promise.all([
    request<{
      runtimes: Array<{
        runtime_id: string;
        available: boolean;
        capabilities?: string[];
        missing_capabilities?: string[];
        detail?: string;
      }>;
      device?: {
        device_id?: string;
        device_name?: string;
        trusted?: boolean;
        resident_listener?: { online?: boolean };
      };
    }>("/api/runtimes"),
    listIMateOpenClawAgents(),
    request<{
      configured: boolean;
      agents: Array<{
        id: string;
        name: string;
        status: "online" | "offline";
        runtime_id: string;
        detail: string;
        capabilities?: string[];
      }>;
    }>("/api/cloudagents"),
  ]);
  const device = runtimeData.device;
  const codeBuddyRuntime = runtimeData.runtimes.find(
    runtime =>
      runtime.runtime_id === "hatchery-teamai-codebuddy" && runtime.available
  );
  const agents: WorkflowRuntimeAgent[] = [];
  if (device?.device_id && codeBuddyRuntime) {
    const codeBuddyBase = {
      id: `codebuddy:${device.device_id}`,
      name: `CodeBuddy · ${device.device_name || device.device_id}`,
      platform: "codebuddy" as const,
      location: "local" as const,
      status:
        device.trusted && device.resident_listener?.online
          ? ("online" as const)
          : ("offline" as const),
      runtimeId: "hatchery-teamai-codebuddy",
      deviceId: device.device_id,
      detail: codeBuddyRuntime.detail ?? `TeamAI 设备 ${device.device_id}`,
      capabilities: codeBuddyRuntime.capabilities ?? [],
      missingCapabilities: codeBuddyRuntime.missing_capabilities ?? [],
    };
    agents.push(codeBuddyBase);
    agents.push(
      ...["A", "B", "C"].map((suffix, index) => ({
        ...codeBuddyBase,
        id: `codebuddy:project-progress-${String.fromCharCode(97 + index)}`,
        name: `CodeBuddy · 项目进展 Agent ${suffix}`,
        detail: `${codeBuddyBase.detail} · 独立 ACP 会话 ${suffix}`,
      }))
    );
  }
  agents.push(
    ...imateAgents.map(agent => ({
      id: agent.id,
      name: `iMate OpenClaw · ${agent.name}`,
      platform: "imate" as const,
      location: "cloud" as const,
      status: agent.online_status === "online" ? ("online" as const) : ("offline" as const),
      runtimeId: "hatchery-teamai-imate-openclaw",
      deviceId: device?.device_id ?? "",
      targetAgentId: agent.id,
      detail: `iMate Agent ${agent.id}`,
      capabilities: ["iwiki.read", "platform.read"],
      missingCapabilities: [],
    }))
  );
  agents.push(
    ...cloudAgentData.agents.map(agent => ({
      id: agent.id,
      name: `CloudAgent · ${agent.name}`,
      platform: "cloudagent" as const,
      location: "cloud" as const,
      status: agent.status,
      runtimeId: agent.runtime_id,
      deviceId: "devresonance-cloud",
      targetAgentId: agent.id,
      detail: agent.detail,
      capabilities: agent.capabilities ?? [],
      missingCapabilities: cloudAgentData.configured
        ? []
        : ["cloudagent.invoke"],
    }))
  );
  return agents;
}

export async function createStructuredWorkflowTask(input: {
  prompt: string;
  workflowId: string;
  workflowName: string;
  assignmentMode: WorkflowAssignmentMode;
  deliveryMode: WorkflowDeliveryMode;
  targetAgentId?: string;
  imateProjectId?: string;
  nodeAssignments: WorkflowNodeAssignment[];
  workflowNodes: WorkflowDefinitionNode[];
  workflowInputs: Record<string, WorkflowTaskInputValue>;
}) {
  const assignedPlatforms = new Set(
    input.nodeAssignments.map(assignment => assignment.platform)
  );
  const agentRuntimeId = assignedPlatforms.has("cloudagent")
    ? "node-routed-multi-agent"
    : input.assignmentMode === "shared"
      ? "codebuddy-acp"
      : "codebuddy-imate-mixed";
  const data = await request<{ task: WorkflowRuntimeTask }>("/api/tasks", {
    method: "POST",
    body: JSON.stringify({
      prompt: input.prompt,
      // 项目协作页上的工作流已将节点契约完整提交给编排服务，
      // 无需再依赖服务器本机 Git/Ruby 重新拉取并校验原始包。
      runtime_id: "structured-project-workflow",
      model: "default",
      target_agent_id: input.targetAgentId ?? "",
      imate_project_id: input.imateProjectId ?? "",
      delivery_mode: input.deliveryMode,
      agent_assignment_mode: input.assignmentMode,
      agent_runtime_id: agentRuntimeId,
      node_assignments: input.nodeAssignments.map(assignment => ({
        phase_id: assignment.phaseId,
        project_agent_id: assignment.projectAgentId,
        platform: assignment.platform,
        location: assignment.location,
        target_agent_id: assignment.targetAgentId ?? "",
      })),
      // Do not only send the field declarations in workflow_definition.phases[].inputs.
      // The orchestrator needs the concrete task values to build the first-node handoff.
      workflow_inputs: input.workflowInputs,
      workflow_definition: {
        workflow_id: input.workflowId,
        name: input.workflowName,
        execution_mode: input.workflowNodes.some(
          node => node.decisionMode || node.onFail
        )
          ? "state_machine"
          : "dag",
        phases: input.workflowNodes.map((node, index) => ({
          id: node.id,
          title: node.title,
          agent_id: node.agentId,
          depends_on: node.dependsOn,
          prompt: node.prompt,
          inputs: node.inputs,
          artifacts: node.artifacts,
          optional_artifacts: node.optionalArtifacts ?? [],
          config_assets: node.configAssets ?? [],
          approval_required: node.approvalRequired,
          required_evidence: node.requiredEvidence ?? [],
          reject_output_markers: node.rejectOutputMarkers ?? [],
          required_capabilities: node.requiredCapabilities ?? [],
          on_pass:
            node.onPass === undefined
              ? input.workflowNodes[index + 1]?.id ?? null
              : node.onPass,
          on_fail: node.onFail ?? node.id,
          decision_mode: node.decisionMode ?? null,
          max_retries: node.maxRetries ?? 0,
        })),
      },
    }),
  });
  return data.task;
}

export async function getWorkflowTask(taskId: string, after = 0) {
  return request<{ task: WorkflowRuntimeTask; events: WorkflowRuntimeEvent[] }>(
    `/api/tasks/${encodeURIComponent(taskId)}/events?after=${after}`
  );
}

export async function approveWorkflowGate(taskId: string, gateId: string) {
  const data = await request<{ task: WorkflowRuntimeTask }>(
    `/api/tasks/${encodeURIComponent(taskId)}/approve`,
    {
      method: "POST",
      body: JSON.stringify({ gate_id: gateId }),
    }
  );
  return data.task;
}

export async function cancelWorkflowTask(taskId: string) {
  const data = await request<{ task: WorkflowRuntimeTask }>(
    `/api/tasks/${encodeURIComponent(taskId)}/cancel`,
    {
      method: "POST",
      body: JSON.stringify({}),
    }
  );
  return data.task;
}

export function workflowArtifactUrl(taskId: string, path: string) {
  const encodedPath = path
    .split("/")
    .filter(Boolean)
    .map(part => encodeURIComponent(part))
    .join("/");
  return `${API_BASE}/artifacts/${encodeURIComponent(taskId)}/${encodedPath}`;
}
