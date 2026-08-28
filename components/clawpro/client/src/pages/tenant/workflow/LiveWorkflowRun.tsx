import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import {
  Check,
  ChevronDown,
  CircleStop,
  Download,
  ExternalLink,
  FileText,
  GitBranch,
  Play,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Spinner } from "@/components/ui/spinner";
import { StatusTag, type StatusTagColor } from "@/components/ui/status-tag";
import { TenantCard } from "@/components/ui/Surface";
import { Textarea } from "@/components/ui/textarea";
import { BodyMedium, MetaText } from "@/components/ui/Typography";
import { NodeConfigAssets } from "./NodeConfigAssets";
import { cloneMultiAgentNodeAssets } from "./multiAgentDevelopmentAssets";
import type {
  IOValue,
  TenantAgent,
  TenantProjectMember,
  TenantTask,
} from "../tenantProjectStore";
import {
  TENANT_MULTI_AGENT_DEV_TEMPLATE_ID,
  TENANT_PRE_VISIT_BRIEF_TEMPLATE_ID,
  tenantProjectStore,
} from "../tenantProjectStore";
import {
  approveWorkflowGate,
  cancelWorkflowTask,
  createStructuredWorkflowTask,
  getWorkflowTask,
  isWorkflowTaskNotFoundError,
  workflowArtifactUrl,
  type WorkflowAssignmentMode,
  type WorkflowDeliveryMode,
  type WorkflowRuntimeEvent,
  type WorkflowRuntimeArtifact,
  type WorkflowRuntimePhase,
  type WorkflowRuntimeTask,
  type WorkflowNodeAssignment,
} from "./clawproWorkflowApi";

const TERMINAL_STATUSES = new Set(["completed", "failed", "canceled"]);
const ACTIVE_PHASE_STATUSES = new Set(["running", "awaiting_approval"]);

function isRuntimeTaskActive(task: WorkflowRuntimeTask | null | undefined) {
  if (!task || task.cancel_requested || task.cancellable === false) return false;
  if (!TERMINAL_STATUSES.has(task.status)) return true;
  return (task.workflow_phases ?? []).some(phase =>
    ACTIVE_PHASE_STATUSES.has(phase.status)
  );
}
const ISSUEFIX_PHASE_IDS = [
  "analyze",
  "fix",
  "review",
  "test",
  "mr",
  "checkers",
  "verify",
  "close",
] as const;

function runtimeWorkflowId(task: TenantTask) {
  return task.pipelineTemplateId === "pl-skillhub-issuefix-demo"
    ? "skillhub-issuefix"
    : task.pipelineTemplateId ?? task.id;
}

function runtimePhaseId(task: TenantTask, index: number) {
  if (runtimeWorkflowId(task) === "skillhub-issuefix") {
    return ISSUEFIX_PHASE_IDS[index];
  }
  return task.workflow[index]?.runtimePhaseId ?? `node-${index + 1}`;
}

function runtimeArtifactPaths(task: TenantTask, index: number, optional = false) {
  const outputs = (task.workflow[index]?.runtimeOutputs ?? []).filter(output =>
    optional ? output.required === false : output.required !== false
  );
  const paths = outputs.map(output => {
    if (/\.[a-z0-9]+$/i.test(output.label)) return output.label;
    const extension = output.type === "json" ? "json" : "md";
    return `${output.key}.${extension}`;
  });
  return paths.length > 0 ? Array.from(new Set(paths)) : [`node-${index + 1}-result.md`];
}

function isVisibleWorkflowEvent(event: WorkflowRuntimeEvent) {
  return (
    event.type.startsWith("workflow.") ||
    event.type === "task.completed" ||
    event.type === "task.failed"
  );
}

const TASK_STATUS: Record<string, { label: string; variant: StatusTagColor }> =
  {
    queued: { label: "排队中", variant: "gray" },
    running: { label: "执行中", variant: "blue" },
    waiting_approval: { label: "等待确认", variant: "orange" },
    completed: { label: "已完成", variant: "green" },
    skipped: { label: "已跳过", variant: "gray" },
    failed: { label: "执行失败", variant: "red" },
    canceled: { label: "已取消", variant: "gray" },
  };

const PHASE_STATUS: Record<string, { label: string; variant: StatusTagColor }> =
  {
    ready: { label: "等待执行", variant: "gray" },
    pending: { label: "等待执行", variant: "gray" },
    running: { label: "执行中", variant: "blue" },
    awaiting_approval: { label: "等待确认", variant: "orange" },
    completed: { label: "已完成", variant: "green" },
    failed: { label: "执行失败", variant: "red" },
    canceled: { label: "已停止", variant: "gray" },
  };

function normalizeSnapshot(task: WorkflowRuntimeTask) {
  return {
    backendTaskId: task.task_id,
    runtimeId: task.runtime_id,
    assignmentMode: task.agent_assignment_mode ?? "shared",
    agentRuntimeId: task.agent_runtime_id ?? "codebuddy-acp",
    status: task.status,
    canStop: isRuntimeTaskActive(task),
    cancelRequested: Boolean(task.cancel_requested),
    currentPhase: task.workflow_current_phase ?? null,
    handoffContract: task.handoff_contract ?? "ClawPro Handoff v2",
    phases: (task.workflow_phases ?? []).map(phase => ({
      id: phase.id,
      title: phase.title,
      runtimeId: phase.runtime_id,
      agentInstanceId: phase.agent_instance_id,
      status: phase.status,
      outputs: phase.artifacts ?? [],
      approvalRequired: Boolean(phase.approval_required),
    })),
    artifacts: (task.available_artifacts ?? []).map((artifact, index) => ({
      id: artifact.artifact_id ?? `${task.task_id}-artifact-${index}`,
      path: artifact.path,
      mediaType: artifact.media_type,
      size: artifact.size,
      sha256: artifact.sha256,
    })),
    updatedAt: task.updated_at,
  };
}

export function LiveWorkflowStartDialog({
  open,
  onOpenChange,
  onStarted,
  projectId,
  task,
  agents: projectAgents,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onStarted?: (backendTaskId: string) => void;
  projectId: string;
  task: TenantTask;
  agents: TenantAgent[];
}) {
  const assignmentMode = useMemo<WorkflowAssignmentMode | null>(() => {
    const platforms = task.workflow.map(node =>
      projectAgents.find(agent => agent.id === node.agentId)?.platform
    );
    const supported = platforms.every(
      platform =>
        platform === "codebuddy" ||
        platform === "imate" ||
        platform === "cloudagent"
    );
    if (!supported || platforms.some(platform => !platform)) return null;
    return new Set(platforms).size === 1 ? "shared" : "mixed";
  }, [projectAgents, task.workflow]);
  const [deliveryMode, setDeliveryMode] = useState<WorkflowDeliveryMode>("wss");
  const [imateProjectId, setIMateProjectId] = useState(
    () => localStorage.getItem("clawpro.imateProjectId") ?? ""
  );
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState("");
  const [workflowInputs, setWorkflowInputs] = useState<Record<string, IOValue>>(
    () => structuredClone(task.taskInputs ?? {})
  );

  useEffect(() => {
    if (!open) return;
    setWorkflowInputs(structuredClone(task.taskInputs ?? {}));
    setSubmitError("");
  }, [open, task.id]);

  const editableTaskInput =
    task.pipelineTemplateId === TENANT_MULTI_AGENT_DEV_TEMPLATE_ID
      ? { key: "requirement", value: workflowInputs.requirement }
      : task.pipelineTemplateId === TENANT_PRE_VISIT_BRIEF_TEMPLATE_ID
        ? { key: "input", value: workflowInputs.input }
        : undefined;
  const nodeAssignments = useMemo<WorkflowNodeAssignment[]>(
    () =>
      task.workflow.flatMap((node, index) => {
        const agent = projectAgents.find(item => item.id === node.agentId);
        const phaseId = runtimePhaseId(task, index);
        if (
          !agent ||
          !phaseId ||
          agent.platform !== "codebuddy" &&
          agent.platform !== "imate" &&
          agent.platform !== "cloudagent"
        ) {
          return [];
        }
        return [
          {
            phaseId,
            projectAgentId: agent.id,
            platform: agent.platform,
            location: agent.location,
            targetAgentId: agent.targetAgentId,
          },
        ];
      }),
    [projectAgents, task]
  );

  const workflowNodes = useMemo(
    () =>
      task.workflow.map((node, index) => {
        const phaseByNodeId = new Map(
          task.workflow.map((item, itemIndex) => [
            item.id,
            runtimePhaseId(task, itemIndex),
          ])
        );
        return {
          id: runtimePhaseId(task, index),
          title: node.title,
          agentId: node.role ?? `agent-${index + 1}`,
          dependsOn: node.dependsOn
            .map(dependency => phaseByNodeId.get(dependency))
            .filter((phaseId): phaseId is string => Boolean(phaseId)),
          prompt:
            node.runtimePrompt ??
            `执行“${node.title}”，承接上游 Handoff v2 的结论和产物，完成后输出本节点结论。`,
          inputs: (node.runtimeInputs ?? []).map(input => ({
            key: input.key,
            label: input.label,
            type: input.type,
            required: input.required,
            source: input.source
              ? {
                  nodeId: phaseByNodeId.get(input.source.nodeId) ?? input.source.nodeId,
                  outputKey: input.source.outputKey,
                }
              : undefined,
          })),
          artifacts: runtimeArtifactPaths(task, index),
          optionalArtifacts: runtimeArtifactPaths(task, index, true),
          configAssets: node.configAssets?.map(asset => ({ ...asset })) ?? [],
          approvalRequired: Boolean(node.runtimeApprovalRequired),
          requiredEvidence: node.runtimeRequiredEvidence,
          rejectOutputMarkers: node.runtimeRejectOutputMarkers,
          requiredCapabilities: node.runtimeRequiredCapabilities,
          onPass: node.runtimeOnPass,
          onFail: node.runtimeOnFail,
          decisionMode: node.runtimeDecisionMode,
          maxRetries: node.runtimeMaxRetries,
        };
      }),
    [task]
  );

  const selectedIMateAgents = useMemo(
    () =>
      Array.from(
        new Map(
          nodeAssignments
            .filter(assignment => assignment.platform === "imate")
            .map(assignment => [
              assignment.targetAgentId,
              projectAgents.find(agent => agent.id === assignment.projectAgentId),
            ])
        ).values()
      ).filter((agent): agent is TenantAgent => Boolean(agent)),
    [nodeAssignments, projectAgents]
  );

  const hasIMateNodes = nodeAssignments.some(
    assignment => assignment.platform === "imate"
  );
  const cloudAgentCount = nodeAssignments.filter(
    assignment => assignment.platform === "cloudagent"
  ).length;
  const unavailableAssignments = nodeAssignments.filter(assignment => {
    const agent = projectAgents.find(
      item => item.id === assignment.projectAgentId
    );
    return !agent || agent.status !== "online";
  });

  const canSubmit =
    Boolean(assignmentMode) &&
    nodeAssignments.length === task.workflow.length &&
    unavailableAssignments.length === 0 &&
    (!editableTaskInput?.value || Boolean(editableTaskInput.value.value.trim())) &&
    (!hasIMateNodes ||
      (nodeAssignments
        .filter(assignment => assignment.platform === "imate")
        .every(assignment => Boolean(assignment.targetAgentId)) &&
        Boolean(imateProjectId.trim())));

  async function handleStart() {
    if (!assignmentMode || !canSubmit || submitting) return;
    setSubmitting(true);
    setSubmitError("");
    try {
      if (hasIMateNodes) {
        localStorage.setItem("clawpro.imateProjectId", imateProjectId.trim());
      }
      const runtimeTask = await createStructuredWorkflowTask({
        prompt:
          `# 项目任务\n\n${task.title}\n\n${task.description || "请按工作流完成任务并回传节点产物。"}` +
          (editableTaskInput?.value
            ? `\n\n# 本次任务输入\n\n${editableTaskInput.value.value.trim()}`
            : ""),
        workflowId: runtimeWorkflowId(task),
        workflowName: task.title,
        assignmentMode,
        deliveryMode,
        targetAgentId:
          hasIMateNodes
            ? nodeAssignments.find(assignment => assignment.platform === "imate")
                ?.targetAgentId
            : undefined,
        imateProjectId:
          hasIMateNodes ? imateProjectId.trim() : undefined,
        nodeAssignments,
        workflowNodes,
        workflowInputs,
      });
      tenantProjectStore.bindRuntimeExecution(
        projectId,
        task.id,
        normalizeSnapshot(runtimeTask)
      );
      onStarted?.(runtimeTask.task_id);
      onOpenChange(false);
      toast.success("真实工作流已提交，节点状态将自动更新");
    } catch (error) {
      setSubmitError((error as Error).message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>启动真实 Agent 工作流</DialogTitle>
          <DialogDescription>
            当前接入真实 POC 后端，按“{task.title}”的 {task.workflow.length}
            个节点执行；节点状态、人工确认和产物会回传到当前任务。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="px-6">
          <div className="space-y-4">
            {editableTaskInput?.value && (
              <div className="space-y-1.5">
                <Label htmlFor="workflow-task-input">
                  本次任务输入
                </Label>
                <Textarea
                  id="workflow-task-input"
                  value={editableTaskInput.value.value}
                  onChange={event =>
                    setWorkflowInputs(current => ({
                      ...current,
                      [editableTaskInput.key]: {
                        ...editableTaskInput.value,
                        value: event.target.value,
                      },
                    }))
                  }
                  rows={
                    task.pipelineTemplateId === TENANT_PRE_VISIT_BRIEF_TEMPLATE_ID
                      ? 4
                      : 7
                  }
                  className="min-h-28 resize-y"
                  placeholder={
                    task.pipelineTemplateId === TENANT_PRE_VISIT_BRIEF_TEMPLATE_ID
                      ? "例如：生成 44813 号日程的会前简报"
                      : "请输入本次要完成的研发任务"
                  }
                  aria-describedby="workflow-task-input-help"
                />
                <MetaText
                  id="workflow-task-input-help"
                  className="block"
                  tone="secondary"
                >
                  已预置示例模板，可直接修改。本次内容只用于当前运行，不会覆盖
                  SOP 模板。
                </MetaText>
                {workflowInputs.reference_artifacts && (
                  <TenantCard
                    state="static"
                    className="bg-[var(--bg-subtle)] p-3"
                  >
                    <BodyMedium>产物对照基准</BodyMedium>
                    <MetaText className="mt-1 block" tone="secondary">
                      contest_school_registry_20260810_1018.zip · 10
                      份核心产物，最终汇总节点会逐项比较路径、完整性、规模和结果指标。
                    </MetaText>
                  </TenantCard>
                )}
                {!editableTaskInput.value.value.trim() && (
                  <p className="m-0 text-xs text-[var(--text-danger)]">
                    请输入本次任务内容后再开始执行。
                  </p>
                )}
              </div>
            )}

            {task.pipelineTemplateId === TENANT_MULTI_AGENT_DEV_TEMPLATE_ID &&
              workflowInputs.repository_url && (
                <div className="space-y-1.5">
                  <Label htmlFor="workflow-repository-url">
                    源码仓库（选填）
                  </Label>
                  <Input
                    id="workflow-repository-url"
                    value={workflowInputs.repository_url.value}
                    onChange={event =>
                      setWorkflowInputs(current => ({
                        ...current,
                        repository_url: {
                          ...current.repository_url,
                          value: event.target.value,
                        },
                      }))
                    }
                    placeholder="留空则使用当前项目已绑定的工作区"
                  />
                  <MetaText className="block" tone="secondary">
                    只在需要指定其他仓库时填写。留空不会阻止启动；执行时由
                    TeamAI 自动定位当前项目的仓库、分支和工作区。
                  </MetaText>
                </div>
              )}

            <div className="space-y-1.5">
              <Label>节点 Agent 方案</Label>
              <TenantCard state="static" className="bg-[var(--bg-brand-subtle)] p-3">
                <BodyMedium>
                  {assignmentMode === "shared"
                    ? cloudAgentCount === task.workflow.length
                      ? `DevResonance CloudAgent · 全部 ${task.workflow.length} 个节点`
                      : `CodeBuddy ACP · 全部 ${task.workflow.length} 个节点`
                    : assignmentMode === "mixed"
                      ? `CodeBuddy ${nodeAssignments.filter(item => item.platform === "codebuddy").length} · iMate ${nodeAssignments.filter(item => item.platform === "imate").length} · CloudAgent ${cloudAgentCount}`
                      : "当前节点组合暂不支持执行"}
                </BodyMedium>
                <MetaText className="mt-1 block" tone="secondary">
                  每个节点会连同 Agent 路由提交到 ClawPro；本地 Agent 经
                  TeamAI 执行，CloudAgent 由后端通过 HTTPS direct-prompt 调用。
                </MetaText>
              </TenantCard>
              {!assignmentMode && (
                <p className="m-0 text-xs text-[var(--text-danger)]">
                  当前 POC 支持 CodeBuddy、iMate 与 DevResonance CloudAgent 节点组合。
                </p>
              )}
              {assignmentMode && unavailableAssignments.length > 0 && (
                <p className="m-0 text-xs text-[var(--text-danger)]">
                  当前有 {unavailableAssignments.length} 个节点的 Agent
                  未授权或不在线，请先完成 Agent 共享与服务端调用凭证配置。
                </p>
              )}
            </div>

            {hasIMateNodes && (
              <>
                <div className="space-y-1.5">
                  <Label>iMate 项目 ID</Label>
                  <Input
                    value={imateProjectId}
                    onChange={event => setIMateProjectId(event.target.value)}
                    placeholder="输入 iMate 项目 ID"
                  />
                </div>
                <div className="space-y-1.5">
                  <Label>已选真实 iMate Agent</Label>
                  <TenantCard state="static" className="bg-[var(--bg-subtle)] p-3">
                    {selectedIMateAgents.map(agent => (
                      <MetaText key={agent.id} className="block" tone="secondary">
                        {agent.name} · {agent.targetAgentId}
                      </MetaText>
                    ))}
                  </TenantCard>
                  <MetaText className="block" tone="weak">
                    iMate Agent 由工作流节点下拉选定，启动时不再二次选择。
                  </MetaText>
                </div>
                <div className="space-y-1.5">
                  <Label>唤醒方式</Label>
                  <Select
                    value={deliveryMode}
                    onValueChange={value =>
                      setDeliveryMode(value as WorkflowDeliveryMode)
                    }
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="wss">WSS 常驻连接</SelectItem>
                      <SelectItem value="hook">本地 Hook 唤醒</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </>
            )}

            {submitError && (
              <p className="m-0 text-sm text-[var(--text-danger)]">
                {submitError}
              </p>
            )}
          </div>
        </DialogBody>
        <DialogFooter>
          <Button
            variant="tenant-outline"
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            取消
          </Button>
          <Button
            variant="tenant-primary"
            onClick={handleStart}
            disabled={!canSubmit || submitting}
          >
            {submitting ? <Spinner /> : <Play className="h-4 w-4" />}
            开始执行
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function phaseInputLabel(phases: WorkflowRuntimePhase[], index: number) {
  if (index === 0) return "启动任务时填写的需求，以及工作流已准备的资料";
  return `上一步「${phases[index - 1]?.title ?? "前序节点"}」交付的结论和文件`;
}

interface WorkflowNodeInputPayload {
  node?: { id?: string };
  task?: {
    goal?: string;
    inputs?: Record<string, { type?: string; value?: string }>;
  };
  inputs?: {
    mappings?: Record<string, string>;
    data?: Record<string, unknown> & { upstream?: unknown };
    artifacts?: WorkflowRuntimeArtifact[];
  };
  output_contract?: Record<string, unknown>;
}

interface WorkflowNodeResultPayload {
  node_id?: string;
  data?: Record<string, unknown> & {
    summary?: string;
    runtime_check?: unknown;
  };
  artifacts?: WorkflowRuntimeArtifact[];
  handoff?: Record<string, unknown>;
}

function phaseEventPayload<T>(
  events: WorkflowRuntimeEvent[],
  type: string,
  phaseId: string
): T | null {
  const event = [...events].reverse().find(item => {
    if (item.type !== type) return false;
    const payload = item.payload as
      | { node?: { id?: string }; node_id?: string }
      | undefined;
    return payload?.node?.id === phaseId || payload?.node_id === phaseId;
  });
  return (event?.payload as T | undefined) ?? null;
}

function isEmptyValue(value: unknown): boolean {
  if (value === undefined || value === null || value === "") return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value).length === 0;
  return false;
}

const FIELD_LABELS: Record<string, string> = {
  requirement: "原始需求",
  summary: "结论",
  status: "状态",
  task_goal: "任务目标",
  upstream_result: "上游结果",
  upstream: "上游内容",
  artifacts: "产物",
  files: "文件",
  path: "文件路径",
  type: "类型",
  required: "必填",
  source: "来源",
  node_id: "节点 ID",
  from_node: "来源节点",
  to_node: "目标节点",
  contract: "交接协议",
  version: "版本",
  runtime_check: "运行检查",
};

function readableFieldLabel(key: string) {
  return (
    FIELD_LABELS[key] ??
    key
      .replace(/[_-]+/g, " ")
      .replace(/\b\w/g, character => character.toUpperCase())
  );
}

function readablePrimitive(value: unknown) {
  if (typeof value === "boolean") return value ? "是" : "否";
  return String(value);
}

function StructuredValue({ value }: { value: unknown }) {
  if (isEmptyValue(value)) {
    return <MetaText tone="weak">暂无内容</MetaText>;
  }
  if (typeof value !== "object") {
    return (
      <p className="m-0 whitespace-pre-wrap break-words text-sm leading-6 text-[var(--text-body)]">
        {readablePrimitive(value)}
      </p>
    );
  }
  if (Array.isArray(value)) {
    return (
      <div className="space-y-2">
        {value.map((item, index) => (
          <div
            key={index}
            className="rounded-[4px] border border-[var(--cp-border)] bg-white px-3 py-2"
          >
            <StructuredValue value={item} />
          </div>
        ))}
      </div>
    );
  }

  const entries = Object.entries(value as Record<string, unknown>).filter(
    ([, entryValue]) => !isEmptyValue(entryValue)
  );
  return (
    <div className="overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-white">
      {entries.map(([key, entryValue], index) => (
        <div
          key={key}
          className={`grid grid-cols-[132px_minmax(0,1fr)] gap-3 px-3 py-2.5 ${
            index > 0 ? "border-t border-[var(--cp-border)]" : ""
          }`}
        >
          <MetaText tone="weak">{readableFieldLabel(key)}</MetaText>
          {typeof entryValue === "object" ? (
            <StructuredValue value={entryValue} />
          ) : (
            <span className="break-words text-sm leading-5 text-[var(--text-body)]">
              {readablePrimitive(entryValue)}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

function artifactPath(path: string) {
  return path.startsWith("agent-workspace/")
    ? path
    : `agent-workspace/${path}`;
}

function formatFileSize(size?: number) {
  if (!size) return "文件";
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(1)} MB`;
}

function artifactTypeLabel(mediaType?: string, path?: string) {
  const extension = path?.split(".").at(-1)?.toLowerCase();
  if (mediaType?.includes("markdown") || extension === "md") return "Markdown";
  if (mediaType?.includes("json") || extension === "json") return "JSON";
  if (mediaType?.includes("python") || extension === "py") return "Python";
  if (mediaType?.startsWith("text/")) return "文本";
  return extension?.toUpperCase() || "文件";
}

function artifactProducerId(artifact: WorkflowRuntimeArtifact) {
  if (artifact.producer_node) return artifact.producer_node;
  const parts = artifact.artifact_id?.split(":") ?? [];
  return parts.length >= 3 ? parts.at(-3) : undefined;
}

function isUserVisibleInputArtifact(artifact: WorkflowRuntimeArtifact) {
  const fileName = artifact.path.split("/").filter(Boolean).at(-1)?.toLowerCase();
  return ![
    "task.md",
    "input.json",
    "workflow-state.json",
    "context-package.json",
    ".handoff.json",
    ".handoff.md",
  ].includes(fileName ?? "");
}

function canPreviewArtifact(artifact: WorkflowRuntimeArtifact) {
  const extension = artifact.path.split(".").at(-1)?.toLowerCase();
  return (
    artifact.media_type?.startsWith("text/") ||
    artifact.media_type?.includes("json") ||
    ["md", "json", "txt", "py", "js", "ts", "tsx", "jsx", "yaml", "yml"].includes(
      extension ?? ""
    )
  );
}

function decodeArtifactText(buffer: ArrayBuffer) {
  const bytes = new Uint8Array(buffer);
  if (bytes[0] === 0xff && bytes[1] === 0xfe) {
    return new TextDecoder("utf-16le").decode(bytes.subarray(2));
  }
  if (bytes[0] === 0xfe && bytes[1] === 0xff) {
    return new TextDecoder("utf-16be").decode(bytes.subarray(2));
  }
  return new TextDecoder("utf-8").decode(bytes);
}

function formatArtifactText(
  text: string,
  artifact: WorkflowRuntimeArtifact
) {
  const extension = artifact.path.split(".").at(-1)?.toLowerCase();
  if (artifact.media_type?.includes("json") || extension === "json") {
    try {
      return JSON.stringify(JSON.parse(text), null, 2);
    } catch {
      return text;
    }
  }
  return text;
}

function WorkflowArtifactPreview({
  taskId,
  artifact,
  producerLabel,
}: {
  taskId: string;
  artifact: WorkflowRuntimeArtifact;
  producerLabel?: string;
}) {
  const [open, setOpen] = useState(false);
  const [fullOpen, setFullOpen] = useState(false);
  const [content, setContent] = useState<string | null>(null);
  const [contentLoading, setContentLoading] = useState(false);
  const path = artifactPath(artifact.path);
  const href = workflowArtifactUrl(taskId, path);
  const fileName = artifact.path.split("/").filter(Boolean).at(-1) ?? artifact.path;
  const previewable = canPreviewArtifact(artifact);
  const producer = producerLabel ?? artifactProducerId(artifact) ?? "当前节点";

  useEffect(() => {
    if (!open || content !== null) return;
    if (!previewable) {
      setContent("该文件不是文本格式，请打开完整文件查看。");
      return;
    }
    let active = true;
    setContentLoading(true);
    fetch(href, { credentials: "same-origin" })
      .then(response => {
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.arrayBuffer();
      })
      .then(buffer => {
        if (!active) return;
        setContent(formatArtifactText(decodeArtifactText(buffer), artifact));
      })
      .catch(() => {
        if (active) setContent("当前产物尚未生成或暂不可预览。");
      })
      .finally(() => {
        if (active) setContentLoading(false);
      });
    return () => {
      active = false;
    };
  }, [artifact, content, href, open, previewable]);

  const previewContent =
    content && content.length > 6000
      ? `${content.slice(0, 6000)}\n\n……内容较长，请点击“查看完整内容”继续阅读`
      : content;

  return (
    <>
      <div className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-white">
        <div className="flex items-start gap-3 px-3 py-3 text-[var(--text-body)]">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[4px] bg-[var(--bg-brand-subtle)] text-[var(--cp-brand-blue)]">
            <FileText className="h-4 w-4" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <span className="truncate text-sm font-medium">{fileName}</span>
              <span className="rounded-[4px] bg-[var(--bg-subtle)] px-1.5 py-0.5 text-[11px] text-[var(--text-muted)]">
                {artifactTypeLabel(artifact.media_type, artifact.path)}
              </span>
              {artifact.version && (
                <span className="text-[11px] text-[var(--text-muted)]">v{artifact.version}</span>
              )}
            </div>
            <MetaText className="mt-1 block truncate" tone="weak">
              来源：{producer} · {formatFileSize(artifact.size)}
            </MetaText>
          </div>
          <div className="flex shrink-0 items-center gap-1.5">
            <button
              type="button"
              onClick={() => setOpen(current => !current)}
              disabled={!previewable}
              className="inline-flex h-8 items-center gap-1 rounded-[var(--radius-md)] px-2.5 text-xs text-[var(--cp-brand-blue)] hover:bg-[var(--bg-brand-subtle)] disabled:cursor-not-allowed disabled:text-[var(--text-muted)]"
            >
              {open ? "收起" : "预览"}
              <ChevronDown
                className={`h-3.5 w-3.5 transition-transform ${open ? "rotate-180" : ""}`}
              />
            </button>
            <a
              href={href}
              download={fileName}
              className="inline-flex h-8 items-center gap-1 rounded-[var(--radius-md)] px-2.5 text-xs text-[var(--cp-brand-blue)] hover:bg-[var(--bg-brand-subtle)]"
            >
              下载
              <Download className="h-3.5 w-3.5" />
            </a>
          </div>
        </div>
        {open && (
          <div className="space-y-3 border-t border-[var(--cp-border)] bg-[var(--bg-subtle)] p-3">
          <div>
            <div className="mb-2 flex items-center justify-between gap-3">
              <MetaText>内容预览</MetaText>
              {previewable && (
                <button
                  type="button"
                  onClick={() => setFullOpen(true)}
                  className="inline-flex items-center gap-1 text-xs text-[var(--cp-brand-blue)]"
                >
                  查看完整内容
                  <ExternalLink className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
            <pre className="m-0 max-h-72 overflow-auto whitespace-pre-wrap break-words rounded-[var(--radius-md)] bg-white p-3 text-sm leading-6 text-[var(--text-secondary)]">
              {contentLoading
                ? "正在读取产物内容..."
                : previewContent ?? "展开后读取产物内容。"}
            </pre>
          </div>
          <div className="overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-white">
            <div className="grid grid-cols-[88px_minmax(0,1fr)] gap-3 px-3 py-2.5">
              <MetaText tone="weak">完整路径</MetaText>
              <span className="break-all text-xs text-[var(--text-body)]">{artifact.path}</span>
            </div>
            <div className="grid grid-cols-[88px_minmax(0,1fr)] gap-3 border-t border-[var(--cp-border)] px-3 py-2.5">
              <MetaText tone="weak">来源节点</MetaText>
              <span className="text-xs text-[var(--text-body)]">{producer}</span>
            </div>
            <div className="grid grid-cols-[88px_minmax(0,1fr)] gap-3 border-t border-[var(--cp-border)] px-3 py-2.5">
              <MetaText tone="weak">SHA-256</MetaText>
              <span className="break-all font-mono text-[11px] leading-5 text-[var(--text-body)]">
                {artifact.sha256 ?? "未提供校验值"}
              </span>
            </div>
            <div className="grid grid-cols-[88px_minmax(0,1fr)] gap-3 border-t border-[var(--cp-border)] px-3 py-2.5">
              <MetaText tone="weak">上游血缘</MetaText>
              {artifact.lineage && artifact.lineage.length > 0 ? (
                <div className="space-y-1">
                  {artifact.lineage.map(item => (
                    <div key={item} className="break-all font-mono text-[11px] leading-5 text-[var(--text-body)]">
                      {item}
                    </div>
                  ))}
                </div>
              ) : (
                <span className="text-xs text-[var(--text-muted)]">首节点产物，无上游血缘</span>
              )}
            </div>
          </div>
          </div>
        )}
      </div>

      <Dialog open={fullOpen} onOpenChange={setFullOpen}>
        <DialogContent size="lg">
          <DialogHeader>
            <DialogTitle>{fileName}</DialogTitle>
            <DialogDescription>
              已按 UTF 编码解析完整内容，JSON 文件会自动格式化。
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            <pre
              aria-label={`${fileName} 完整内容`}
              className="m-0 max-h-[68vh] overflow-auto whitespace-pre-wrap break-words rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-[var(--bg-subtle)] p-4 text-sm leading-6 text-[var(--text-body)]"
            >
              {contentLoading ? "正在读取产物内容..." : content ?? "暂无内容。"}
            </pre>
          </DialogBody>
          <DialogFooter>
            <a
              href={href}
              download={fileName}
              className="inline-flex h-8 items-center gap-1.5 rounded-[var(--radius-md)] border border-[var(--cp-border)] px-3 text-sm text-[var(--text-body)] hover:bg-[var(--bg-subtle)]"
            >
              <Download className="h-3.5 w-3.5" />
              下载文件
            </a>
            <Button variant="tenant-primary" onClick={() => setFullOpen(false)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function NodeDataBlock({
  label,
  value,
  emphasis = false,
}: {
  label: string;
  value: unknown;
  emphasis?: boolean;
}) {
  if (isEmptyValue(value)) return null;
  const rawExpandable = typeof value === "object";
  return (
    <div
      className={`rounded-[var(--radius-md)] border p-3 ${
        emphasis
          ? "border-[var(--cp-brand-blue)] bg-[var(--bg-brand-subtle)]"
          : "border-[var(--cp-border)] bg-[var(--bg-subtle)]"
      }`}
    >
      <BodyMedium className="mb-2 block">{label}</BodyMedium>
      <StructuredValue value={value} />
      {rawExpandable && (
        <details className="mt-2 border-t border-[var(--cp-border)] pt-2">
          <summary className="cursor-pointer text-xs text-[var(--text-muted)]">
            查看原始数据
          </summary>
          <pre className="mb-0 mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words rounded-[4px] bg-white p-2 text-xs leading-5 text-[var(--text-muted)]">
            {JSON.stringify(value, null, 2)}
          </pre>
        </details>
      )}
    </div>
  );
}

function ExecutionSummaryBlock({ value }: { value: unknown }) {
  if (isEmptyValue(value)) return null;
  const raw = typeof value === "string" ? value.trim() : JSON.stringify(value, null, 2);
  const readable = raw
    .replaceAll("**", "")
    .replaceAll("`", "")
    .replace(/\n{3,}/g, "\n\n");
  const isTruncated = readable.length > 320;
  const preview = isTruncated ? `${readable.slice(0, 320).trim()}…` : readable;
  return (
    <div className="rounded-[var(--radius-md)] border border-[var(--cp-brand-blue)] bg-[var(--bg-brand-subtle)] p-4">
      <p className="m-0 whitespace-pre-line text-sm leading-6 text-[var(--text-body)]">
        {preview}
      </p>
      {isTruncated && (
        <details className="group mt-3 border-t border-[var(--cp-border)] pt-3">
          <summary className="flex cursor-pointer list-none items-center gap-1.5 text-xs text-[var(--cp-brand-blue)] [&::-webkit-details-marker]:hidden">
            查看完整执行记录
            <ChevronDown className="h-3.5 w-3.5 transition-transform group-open:rotate-180" />
          </summary>
          <p className="mb-0 mt-3 whitespace-pre-line text-sm leading-6 text-[var(--text-body)]">
            {readable}
          </p>
        </details>
      )}
    </div>
  );
}

function DetailDisclosure({
  title,
  meta,
  defaultOpen = false,
  children,
}: {
  title: string;
  meta?: string;
  defaultOpen?: boolean;
  children: ReactNode;
}) {
  return (
    <details
      defaultOpen={defaultOpen}
      className="group border-t border-[var(--cp-border)] first:border-t-0"
    >
      <summary className="flex cursor-pointer list-none items-center gap-3 py-4 [&::-webkit-details-marker]:hidden">
        <BodyMedium className="min-w-0 flex-1">{title}</BodyMedium>
        {meta && <MetaText tone="weak">{meta}</MetaText>}
        <ChevronDown className="h-4 w-4 shrink-0 text-[var(--text-muted)] transition-transform group-open:rotate-180" />
      </summary>
      <div className="space-y-3 pb-5">{children}</div>
    </details>
  );
}

function inputSourceMeta(source: string) {
  if (source.startsWith("$task.inputs.")) {
    const key = source.slice("$task.inputs.".length).split(".")[0];
    const isUserInput = key === "requirement" || key === "input";
    return {
      kind: isUserInput ? "你填写的" : "工作流已准备",
      detail: isUserInput ? "启动任务时填写" : readableFieldLabel(key),
    };
  }
  if (source.startsWith("$nodes.")) {
    const [node, , field] = source.slice("$nodes.".length).split(".");
    return {
      kind: "上一步结果",
      detail: `${node}${field ? ` · ${field}` : ""}`,
    };
  }
  if (source.startsWith("$input.")) {
    return { kind: "本次任务", detail: readableFieldLabel(source.slice("$input.".length)) };
  }
  if (source.startsWith("$vars.")) {
    const [node, ...field] = source.slice("$vars.".length).split(".");
    return {
      kind: "上一步结果",
      detail: `${node}${field.length > 0 ? ` · ${field.join(".")}` : ""}`,
    };
  }
  if (source.startsWith("$artifacts.")) {
    const [node] = source.slice("$artifacts.".length).split(".");
    return { kind: "上一步文件", detail: node || "全部文件" };
  }
  if (source.startsWith("$item.")) {
    return { kind: "循环项", detail: source.slice("$item.".length) };
  }
  if (source.startsWith("$local.")) {
    return { kind: "本地引用", detail: source.slice("$local.".length) };
  }
  return { kind: "工作流资料", detail: source };
}

function compactValue(value: unknown) {
  if (isEmptyValue(value)) return "暂未提供";
  if (typeof value === "string") {
    const compact = value.replace(/\s+/g, " ").trim();
    return compact.length > 120 ? `${compact.slice(0, 120)}…` : compact;
  }
  if (Array.isArray(value)) return `${value.length} 项内容`;
  if (typeof value === "object") {
    const summary = (value as { summary?: unknown }).summary;
    if (typeof summary === "string") return compactValue(summary);
    return `${Object.keys(value as Record<string, unknown>).length} 个字段`;
  }
  return readablePrimitive(value);
}

function resolveMappedInput(
  key: string,
  source: string,
  payload: WorkflowNodeInputPayload | null,
) {
  if (!payload) return undefined;
  if (source === "$input.task_goal" || key === "task_goal") {
    return payload.task?.goal;
  }
  if (source.startsWith("$task.inputs.")) {
    const inputKey = source.slice("$task.inputs.".length).split(".")[0];
    return payload.task?.inputs?.[inputKey]?.value;
  }
  if (source.startsWith("$nodes.")) {
    const [producerNode] = source.slice("$nodes.".length).split(".");
    const candidates = (payload.inputs?.artifacts ?? []).filter(
      artifact => artifactProducerId(artifact) === producerNode
    );
    if (candidates.length === 0) return undefined;
    const normalizedKey = key.toLowerCase().replaceAll("-", "_");
    const exact = candidates.find(artifact => {
      const fileName = artifact.path.split("/").at(-1) ?? artifact.path;
      const stem = fileName.replace(/\.[^.]+$/, "").toLowerCase().replaceAll("-", "_");
      return stem === normalizedKey || stem.includes(normalizedKey);
    });
    const selected = exact ? [exact] : candidates;
    if (selected.length === 1) {
      const artifact = selected[0];
      return {
        path: artifact.path,
        size: artifact.size,
        sha256: artifact.sha256,
        producer_node: artifactProducerId(artifact),
      };
    }
    return selected.map(artifact => ({
      path: artifact.path,
      size: artifact.size,
      sha256: artifact.sha256,
    }));
  }
  if (source.startsWith("$artifacts.")) {
    return (payload.inputs?.artifacts ?? []).map(artifact => artifact.path);
  }
  const data = payload.inputs?.data;
  if (data && key in data) return data[key];
  if (source.startsWith("$vars.")) return data?.upstream;
  return undefined;
}

function InputMappingPanel({
  payload,
}: {
  payload: WorkflowNodeInputPayload | null;
}) {
  const mappings = Object.entries(payload?.inputs?.mappings ?? {});
  if (mappings.length === 0) return null;

  return (
    <div className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-white">
      <div className="divide-y divide-[var(--cp-border)]">
        {mappings.map(([key, source]) => {
          const sourceMeta = inputSourceMeta(source);
          const value = resolveMappedInput(key, source, payload);
          return (
            <details key={key} className="group">
              <summary className="flex cursor-pointer list-none items-start gap-3 px-3 py-3 [&::-webkit-details-marker]:hidden">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm font-medium text-[var(--text-body)]">
                      {readableFieldLabel(key)}
                    </span>
                    <span className="rounded-[4px] bg-[var(--bg-brand-subtle)] px-1.5 py-0.5 text-[11px] text-[var(--cp-brand-blue)]">
                      {sourceMeta.kind}
                    </span>
                  </div>
                  <p className="mb-0 mt-1 line-clamp-2 text-xs leading-5 text-[var(--text-muted)]">
                    {compactValue(value)}
                  </p>
                  <MetaText className="mt-1 block" tone="weak">
                    来源：{sourceMeta.detail}
                  </MetaText>
                </div>
                <span className={`mt-0.5 shrink-0 text-xs ${isEmptyValue(value) ? "text-[var(--text-warning)]" : "text-[var(--text-success)]"}`}>
                  {isEmptyValue(value) ? "缺少内容" : "已准备"}
                </span>
                <ChevronDown className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[var(--text-muted)] transition-transform group-open:rotate-180" />
              </summary>
              <div className="border-t border-[var(--cp-border)] bg-[var(--bg-subtle)] p-3">
                <StructuredValue value={value} />
                <MetaText className="mt-2 block" tone="weak">
                  内容来源：{sourceMeta.detail}
                </MetaText>
              </div>
            </details>
          );
        })}
      </div>
    </div>
  );
}

interface OutputContractShape {
  data_schema?: {
    properties?: Record<string, { type?: unknown; description?: string }>;
    required?: string[];
  };
  required_artifacts?: string[];
}

function schemaTypeLabel(type: unknown) {
  const normalized = Array.isArray(type) ? type.filter(item => item !== "null") : [type];
  return normalized
    .map(item => {
      if (item === "string") return "文本";
      if (item === "object") return "对象";
      if (item === "array") return "列表";
      if (item === "number" || item === "integer") return "数字";
      if (item === "boolean") return "是/否";
      return String(item ?? "任意");
    })
    .join(" / ");
}

function OutputContractPanel({
  contract,
  actualData,
  artifacts,
}: {
  contract: Record<string, unknown> | undefined;
  actualData: WorkflowNodeResultPayload["data"];
  artifacts: WorkflowRuntimeArtifact[];
}) {
  const normalized = (contract ?? {}) as OutputContractShape;
  const properties = Object.entries(normalized.data_schema?.properties ?? {});
  const required = new Set(normalized.data_schema?.required ?? []);
  const requiredArtifacts = normalized.required_artifacts ?? [];
  if (properties.length === 0 && requiredArtifacts.length === 0) return null;

  return (
    <div className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-white">
      <div className="flex items-center justify-between gap-3 border-b border-[var(--cp-border)] px-3 py-2.5">
        <BodyMedium>输出契约</BodyMedium>
        <MetaText tone="weak">iMate 结构化字段</MetaText>
      </div>
      <div className="divide-y divide-[var(--cp-border)]">
        {properties.map(([key, definition]) => {
          const value = actualData?.[key];
          const hasValue = !isEmptyValue(value);
          return (
            <details key={key} className="group">
              <summary className="flex cursor-pointer list-none items-start gap-3 px-3 py-3 [&::-webkit-details-marker]:hidden">
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm font-medium text-[var(--text-body)]">
                      {readableFieldLabel(key)}
                    </span>
                    <span className="rounded-[4px] bg-[var(--bg-subtle)] px-1.5 py-0.5 text-[11px] text-[var(--text-muted)]">
                      {schemaTypeLabel(definition.type)}
                    </span>
                    {required.has(key) && (
                      <span className="text-[11px] text-[var(--text-danger)]">必填</span>
                    )}
                  </div>
                  <p className="mb-0 mt-1 line-clamp-2 text-xs leading-5 text-[var(--text-muted)]">
                    {hasValue
                      ? compactValue(value)
                      : definition.description || "等待 Agent 返回该字段"}
                  </p>
                </div>
                <span className={`mt-0.5 shrink-0 text-xs ${hasValue ? "text-[var(--text-success)]" : "text-[var(--text-warning)]"}`}>
                  {hasValue ? "已返回" : "待返回"}
                </span>
                <ChevronDown className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[var(--text-muted)] transition-transform group-open:rotate-180" />
              </summary>
              <div className="border-t border-[var(--cp-border)] bg-[var(--bg-subtle)] p-3">
                <StructuredValue value={value} />
              </div>
            </details>
          );
        })}
        {requiredArtifacts.map(path => {
          const fileName = path.split("/").filter(Boolean).at(-1) ?? path;
          const produced = artifacts.some(artifact =>
            artifact.path.endsWith(fileName)
          );
          return (
            <div key={path} className="flex items-center gap-3 px-3 py-3">
              <FileText className="h-4 w-4 shrink-0 text-[var(--cp-brand-blue)]" />
              <div className="min-w-0 flex-1">
                <div className="text-sm font-medium text-[var(--text-body)]">{fileName}</div>
                <MetaText tone="weak">必需产物</MetaText>
              </div>
              <span className={`text-xs ${produced ? "text-[var(--text-success)]" : "text-[var(--text-warning)]"}`}>
                {produced ? "已生成" : "待生成"}
              </span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function LiveWorkflowRun({
  projectId,
  task,
  agents,
  members,
  stopRequestNonce = 0,
  onRuntimeExpired,
  onRuntimeActiveChange,
}: {
  projectId: string;
  task: TenantTask;
  agents: TenantAgent[];
  members: TenantProjectMember[];
  stopRequestNonce?: number;
  onRuntimeExpired?: () => void;
  onRuntimeActiveChange?: (active: boolean) => void;
}) {
  const backendTaskId = task.runtimeExecution?.backendTaskId ?? "";
  const [runtimeTask, setRuntimeTask] = useState<WorkflowRuntimeTask | null>(
    null
  );
  const [events, setEvents] = useState<WorkflowRuntimeEvent[]>([]);
  const [error, setError] = useState("");
  const [approving, setApproving] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [stopOpen, setStopOpen] = useState(false);
  const [expandedPhaseId, setExpandedPhaseId] = useState<string | null>(null);
  const lastSeqRef = useRef(0);
  const lastSnapshotRef = useRef("");
  const lastStopRequestRef = useRef(stopRequestNonce);
  const staleTaskHandledRef = useRef(false);
  const onRuntimeExpiredRef = useRef(onRuntimeExpired);

  useEffect(() => {
    onRuntimeExpiredRef.current = onRuntimeExpired;
  }, [onRuntimeExpired]);

  useEffect(() => {
    if (!runtimeTask) return;
    onRuntimeActiveChange?.(isRuntimeTaskActive(runtimeTask));
  }, [onRuntimeActiveChange, runtimeTask]);

  const refresh = useCallback(async () => {
    if (!backendTaskId) return;
    try {
      const data = await getWorkflowTask(backendTaskId, lastSeqRef.current);
      setRuntimeTask(data.task);
      if (data.events.length > 0) {
        lastSeqRef.current = data.events[data.events.length - 1].seq;
        setEvents(current =>
          [...current, ...data.events.filter(isVisibleWorkflowEvent)].slice(-200)
        );
      }
      setError("");
      const snapshot = normalizeSnapshot(data.task);
      const serialized = JSON.stringify(snapshot);
      if (serialized !== lastSnapshotRef.current) {
        lastSnapshotRef.current = serialized;
        tenantProjectStore.syncRuntimeExecution(projectId, task.id, snapshot);
      }
    } catch (requestError) {
      if (isWorkflowTaskNotFoundError(requestError)) {
        if (staleTaskHandledRef.current) return;
        staleTaskHandledRef.current = true;
        if (task.id.startsWith("runtime-view-")) {
          toast.error("这条执行链接已失效，已返回任务看板，请打开任务卡查看最新执行");
          onRuntimeExpiredRef.current?.();
        } else {
          tenantProjectStore.prepareRuntimeRerun(projectId, task.id);
          toast.info("旧执行记录已失效，任务已恢复为可重新运行状态");
        }
        return;
      }
      setError((requestError as Error).message);
    }
  }, [backendTaskId, projectId, task.id]);

  useEffect(() => {
    lastSeqRef.current = 0;
    lastSnapshotRef.current = "";
    setEvents([]);
    setRuntimeTask(null);
    setExpandedPhaseId(null);
    staleTaskHandledRef.current = false;
    void refresh();
  }, [backendTaskId, refresh]);

  useEffect(() => {
    if (stopRequestNonce === lastStopRequestRef.current) return;
    lastStopRequestRef.current = stopRequestNonce;
    setStopOpen(true);
  }, [stopRequestNonce]);

  useEffect(() => {
    if (!backendTaskId || !isRuntimeTaskActive(runtimeTask)) {
      return;
    }
    const timer = window.setInterval(() => void refresh(), 1200);
    return () => window.clearInterval(timer);
  }, [backendTaskId, refresh, runtimeTask?.status]);

  const phases = runtimeTask?.workflow_phases ?? [];
  const status =
    TASK_STATUS[runtimeTask?.status ?? "queued"] ?? TASK_STATUS.queued;
  const recentEvents = useMemo(() => events.slice(-6).reverse(), [events]);
  const detailIndex = expandedPhaseId
    ? phases.findIndex(phase => phase.id === expandedPhaseId)
    : -1;
  const detailPhase = detailIndex >= 0 ? phases[detailIndex] : null;
  const detailWorkflowNode = detailPhase
    ? task.workflow.find(
        (node, index) => runtimePhaseId(task, index) === detailPhase.id
      )
    : undefined;
  const detailInputPayload = detailPhase
    ? phaseEventPayload<WorkflowNodeInputPayload>(
        events,
        "workflow.node.started",
        detailPhase.id
      )
    : null;
  const detailResultPayload = detailPhase
    ? phaseEventPayload<WorkflowNodeResultPayload>(
        events,
        "workflow.node.completed",
        detailPhase.id
      )
    : null;
  const detailInputArtifacts = (detailInputPayload?.inputs?.artifacts ?? []).filter(
    isUserVisibleInputArtifact
  );
  const detailOutputArtifacts = detailResultPayload?.artifacts ?? [];
  const detailInputMappings = Object.keys(
    detailInputPayload?.inputs?.mappings ?? {}
  );
  const detailConfigAssets =
    (detailWorkflowNode?.configAssets?.length ?? 0) > 0
      ? detailWorkflowNode?.configAssets ?? []
      : runtimeTask?.workflow_id === TENANT_MULTI_AGENT_DEV_TEMPLATE_ID && detailPhase
        ? cloneMultiAgentNodeAssets(detailPhase.id)
        : [];
  const detailPending =
    detailPhase &&
    runtimeTask?.pending_approval?.status === "pending" &&
    runtimeTask.pending_approval.after_phase_id === detailPhase.id
      ? runtimeTask.pending_approval
      : null;

  async function handleApprove() {
    const pending = runtimeTask?.pending_approval;
    if (!pending || approving) return;
    setApproving(true);
    try {
      const updated = await approveWorkflowGate(backendTaskId, pending.gate_id);
      setRuntimeTask(updated);
      toast.success("已确认，工作流继续执行");
      await refresh();
    } catch (approvalError) {
      toast.error((approvalError as Error).message);
    } finally {
      setApproving(false);
    }
  }

  async function handleStop() {
    if (!isRuntimeTaskActive(runtimeTask) || stopping) {
      return;
    }
    setStopping(true);
    try {
      const updated = await cancelWorkflowTask(backendTaskId);
      setRuntimeTask(updated);
      setStopOpen(false);
      toast.success("工作流已停止，已生成产物继续保留");
      await refresh();
    } catch (stopError) {
      toast.error((stopError as Error).message);
    } finally {
      setStopping(false);
    }
  }

  if (!runtimeTask && !error) {
    return (
      <div className="flex items-center justify-center gap-2 py-16 text-sm text-[var(--text-muted)]">
        <Spinner className="text-[var(--cp-brand-blue)]" />
        正在连接真实工作流后端
      </div>
    );
  }

  return (
    <div className="w-full space-y-4 p-6">
      <TenantCard
        state="static"
        padding="none"
        className="overflow-hidden bg-white"
      >
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-[var(--cp-border)] px-5 py-4">
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <BodyMedium>真实执行 · {task.title}</BodyMedium>
              <StatusTag mode="soft" variant={status.variant}>
                {status.label}
              </StatusTag>
            </div>
            <MetaText className="mt-1 block" tone="weak">
              任务 ID：{backendTaskId} ·{" "}
              {runtimeTask?.handoff_contract ?? "ClawPro Handoff v2"}
            </MetaText>
          </div>
          <div className="flex items-center gap-2">
            <StatusTag mode="soft" variant="gray">
              {runtimeTask?.agent_assignment_mode === "mixed"
                ? "CodeBuddy + iMate"
                : "CodeBuddy 单 Agent"}
            </StatusTag>
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={() => void refresh()}
            >
              刷新状态
            </Button>
          </div>
        </div>
        {error && (
          <div className="flex items-center justify-between gap-4 px-5 py-3">
            <p className="m-0 text-sm text-[var(--text-danger)]">{error}</p>
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={() => void refresh()}
            >
              重新连接
            </Button>
          </div>
        )}
      </TenantCard>

      <div>
        <div className="mb-3 flex items-end justify-between gap-4">
          <div>
            <h3 className="m-0 text-base font-semibold text-[var(--text-title)]">
              节点流转
            </h3>
            <MetaText className="mt-1 block" tone="weak">
              点击节点后，按“收到什么 → 完成什么 → 交付什么”查看执行结果。
            </MetaText>
          </div>
          <MetaText tone="weak">共 {phases.length} 个真实执行节点</MetaText>
        </div>

        <div className="overflow-auto rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-[var(--bg-brand-subtle)]">
          <div
            className="relative"
            style={{
              width: Math.max(920, 40 + phases.length * 288 + Math.max(0, phases.length - 1) * 72),
              minWidth: Math.max(
                920,
                40 +
                  phases.length * 288 +
                  Math.max(0, phases.length - 1) * 72
              ),
              height: 420,
            }}
          >
            <svg
              aria-hidden
              className="pointer-events-none absolute inset-0 h-full w-full"
            >
              <defs>
                <marker
                  id={`runtime-arrow-${backendTaskId}`}
                  markerWidth="8"
                  markerHeight="8"
                  refX="7"
                  refY="4"
                  orient="auto"
                >
                  <path d="M 0 0 L 8 4 L 0 8 z" fill="var(--text-weak)" />
                </marker>
              </defs>
              {phases.slice(0, -1).map((phase, index) => {
                const sourceX = 20 + index * 360 + 288;
                const targetX = 20 + (index + 1) * 360;
                const y = 152;
                const curveX = (sourceX + targetX) / 2;
                return (
                  <path
                    key={`${phase.id}-${phases[index + 1]?.id}`}
                    d={`M ${sourceX} ${y} C ${curveX} ${y}, ${curveX} ${y}, ${targetX - 6} ${y}`}
                    fill="none"
                    stroke="var(--text-weak)"
                    strokeWidth="1.5"
                    markerEnd={`url(#runtime-arrow-${backendTaskId})`}
                  />
                );
              })}
            </svg>

            {phases.map((phase, index) => {
              const phaseStatus =
                PHASE_STATUS[phase.status] ?? PHASE_STATUS.ready;
              const inputPayload = phaseEventPayload<WorkflowNodeInputPayload>(
                events,
                "workflow.node.started",
                phase.id
              );
              const resultPayload = phaseEventPayload<WorkflowNodeResultPayload>(
                events,
                "workflow.node.completed",
                phase.id
              );
              const expanded = expandedPhaseId === phase.id;
              const pending =
                runtimeTask?.pending_approval?.status === "pending" &&
                runtimeTask.pending_approval.after_phase_id === phase.id
                  ? runtimeTask.pending_approval
                  : null;
              const outputArtifacts = resultPayload?.artifacts ?? [];
              const left = 20 + index * 360;
              const workflowNode = task.workflow[index];
              const selectedAgent = agents.find(
                agent => agent.id === workflowNode?.agentId
              );
              const selectedMember = selectedAgent
                ? members.find(member => member.userId === selectedAgent.ownerId)
                : undefined;
              const agentSelectionLocked = Boolean(
                runtimeTask && isRuntimeTaskActive(runtimeTask)
              );

              return (
                <div key={phase.id}>
                  <div
                    className="absolute top-3 flex items-center gap-2"
                    style={{ left, width: 288 }}
                  >
                    <span
                      className={`inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full border ${
                        phase.status === "completed" || phase.status === "awaiting_approval"
                          ? "border-[var(--cp-brand-blue)] text-[var(--cp-brand-blue)]"
                          : "border-[var(--border-strong)] text-[var(--text-weak)]"
                      }`}
                    >
                      {phase.status === "completed" ? (
                        <Check className="h-3 w-3" />
                      ) : (
                        <span className="h-1.5 w-1.5 rounded-full bg-current" />
                      )}
                    </span>
                    <BodyMedium className="truncate">{phase.title}</BodyMedium>
                  </div>

                  <TenantCard
                    state="static"
                    padding="none"
                    className={`absolute top-[54px] overflow-hidden bg-white transition-[border-color,box-shadow] ${
                      pending
                        ? "border-[var(--text-warning)] [box-shadow:0_0_0_3px_var(--bg-warning-subtle),var(--shadow-card)]"
                        : phase.status === "running"
                          ? "border-[var(--cp-brand-blue)] [box-shadow:0_0_0_3px_var(--bg-brand-selected-solid),var(--shadow-card)]"
                          : "border-white shadow-[var(--shadow-card)]"
                    }`}
                    style={{
                      left,
                      width: 288,
                      height: 334,
                      zIndex: expanded ? 2 : 1,
                    }}
                  >
                    <div className="flex h-full flex-col">
                      <button
                        type="button"
                        className="flex w-full items-start gap-2 border-b border-[var(--cp-border)] p-4 text-left hover:bg-[var(--bg-subtle)]"
                        aria-expanded={expanded}
                        onClick={() =>
                          setExpandedPhaseId(current =>
                            current === phase.id ? null : phase.id
                          )
                        }
                      >
                        <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[var(--bg-brand-subtle)] text-xs font-semibold text-[var(--cp-brand-blue)]">
                          {index + 1}
                        </span>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <h4 className="m-0 min-w-0 flex-1 truncate text-sm font-semibold text-[var(--text-title)]">
                              {phase.title}
                            </h4>
                            {(workflowNode?.configAssets?.length ?? 0) > 0 && (
                              <span className="shrink-0 rounded-[4px] bg-[var(--bg-brand-subtle)] px-1.5 py-0.5 text-[11px] font-medium text-[var(--cp-brand-blue)]">
                                资产 {workflowNode?.configAssets?.length}
                              </span>
                            )}
                            <StatusTag mode="soft" variant={phaseStatus.variant}>
                              {phaseStatus.label}
                            </StatusTag>
                          </div>
                          <MetaText className="mt-1 block truncate" tone="weak">
                            {phase.runtime_id} · {phase.agent_instance_id}
                          </MetaText>
                          {phase.device_id && (
                            <MetaText className="mt-0.5 block truncate" tone="weak">
                              设备 {phase.device_id} · {phase.transport ?? "wss+https"}
                            </MetaText>
                          )}
                        </div>
                        <ChevronDown
                          className={`mt-1 h-4 w-4 shrink-0 text-[var(--text-muted)] transition-transform ${
                            expanded ? "rotate-180" : ""
                          }`}
                        />
                      </button>

                      <div className="px-4 py-3">
                        <div className="mb-3 rounded-[4px] border border-[var(--cp-border)] px-2.5 py-2">
                          <div className="flex items-center justify-between gap-2">
                            <MetaText tone="weak">执行 Agent</MetaText>
                            {agentSelectionLocked && (
                              <MetaText tone="weak">运行中锁定</MetaText>
                            )}
                          </div>
                          <Select
                            value={workflowNode?.agentId ?? undefined}
                            disabled={!workflowNode || agentSelectionLocked}
                            onValueChange={agentId => {
                              if (!workflowNode) return;
                              tenantProjectStore.assignWorkflowAgent(
                                projectId,
                                task.id,
                                workflowNode.id,
                                agentId
                              );
                              toast.success("节点执行 Agent 已更新，重新运行时生效");
                            }}
                          >
                            <SelectTrigger className="mt-1.5 h-8 w-full bg-white text-xs">
                              <SelectValue placeholder="选择云端或本地 Agent" />
                            </SelectTrigger>
                            <SelectContent>
                              {agents.map(agent => {
                                const member = members.find(
                                  item => item.userId === agent.ownerId
                                );
                                return (
                                  <SelectItem key={agent.id} value={agent.id}>
                                    {agent.location === "local" ? "本地" : "云端"} · {agent.name}
                                    {member?.role ? ` · ${member.role}` : ""}
                                  </SelectItem>
                                );
                              })}
                            </SelectContent>
                          </Select>
                          {selectedAgent && (
                            <div className="mt-1.5 flex items-center gap-1.5">
                              <GitBranch className="h-3 w-3 shrink-0 text-[var(--cp-brand-blue)]" />
                              <span className="min-w-0 flex-1 truncate text-xs text-[var(--text-muted)]">
                                {selectedAgent.platform} · {selectedAgent.location === "local" ? "本地" : "云端"}
                                {selectedMember?.displayName
                                  ? ` · ${selectedMember.displayName}`
                                  : ""}
                              </span>
                            </div>
                          )}
                        </div>

                        <div className="grid grid-cols-3 gap-1.5 text-center">
                          <div className="rounded-[4px] bg-[var(--bg-subtle)] px-1 py-2">
                            <MetaText className="block" tone="weak">输入</MetaText>
                            <span className="text-xs font-medium text-[var(--text-body)]">
                              {inputPayload ? "已接收" : "待接收"}
                            </span>
                          </div>
                          <div className="rounded-[4px] bg-[var(--bg-subtle)] px-1 py-2">
                            <MetaText className="block" tone="weak">输出</MetaText>
                            <span className="text-xs font-medium text-[var(--text-body)]">
                              {resultPayload ? "已回传" : "待回传"}
                            </span>
                          </div>
                          <div className="rounded-[4px] bg-[var(--bg-subtle)] px-1 py-2">
                            <MetaText className="block" tone="weak">产物</MetaText>
                            <span className="text-xs font-medium text-[var(--text-body)]">
                              {outputArtifacts.length}
                            </span>
                          </div>
                        </div>

                        {pending ? (
                          <Button
                            variant="tenant-primary"
                            size="sm"
                            className="mt-3 w-full"
                            onClick={event => {
                              event.stopPropagation();
                              void handleApprove();
                            }}
                            disabled={approving}
                          >
                            {approving ? <Spinner /> : <Check className="h-4 w-4" />}
                            {pending.action_label}
                          </Button>
                        ) : (
                          <button
                            type="button"
                            className="mt-3 flex w-full items-center justify-center gap-1.5 text-xs font-medium text-[var(--cp-brand-blue)]"
                            onClick={() =>
                              setExpandedPhaseId(current =>
                                current === phase.id ? null : phase.id
                              )
                            }
                          >
                            {expanded ? "收起执行详情" : "查看本节点的输入和结果"}
                            <ChevronDown
                              className={`h-3.5 w-3.5 transition-transform ${expanded ? "rotate-180" : ""}`}
                            />
                          </button>
                        )}
                      </div>

                    </div>
                  </TenantCard>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {(runtimeTask?.available_artifacts?.length ?? 0) > 0 && (
        <div>
          <div className="mb-3 flex items-end justify-between gap-3">
            <div>
              <h3 className="m-0 text-base font-semibold text-[var(--text-title)]">
                工作流产物
              </h3>
              <MetaText className="mt-1 block" tone="weak">
                参考 iMate 产物模型，保留来源、版本、校验值和节点血缘。
              </MetaText>
            </div>
            <MetaText tone="weak">
              {runtimeTask?.available_artifacts?.length ?? 0} 个文件
            </MetaText>
          </div>
          <div className="space-y-2">
            {runtimeTask?.available_artifacts?.map((artifact, index) => (
              <WorkflowArtifactPreview
                key={artifact.artifact_id ?? `${artifact.path}-${index}`}
                taskId={backendTaskId}
                artifact={artifact}
                producerLabel={
                  phases.find(item => item.id === artifactProducerId(artifact))
                    ?.title
                }
              />
            ))}
          </div>
        </div>
      )}

      {recentEvents.length > 0 && (
        <div>
          <h3 className="mb-3 mt-0 text-base font-semibold text-[var(--text-title)]">
            最近执行动态
          </h3>
          <TenantCard
            state="static"
            padding="none"
            className="divide-y divide-[var(--cp-border)] bg-white"
          >
            {recentEvents.map(event => (
              <div
                key={event.event_id}
                className="flex items-start gap-3 px-5 py-3"
              >
                <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-[var(--cp-brand-blue)]" />
                <div className="min-w-0 flex-1">
                  <p className="m-0 text-sm text-[var(--text-body)]">
                    {event.title}
                  </p>
                  {event.detail && (
                    <MetaText className="mt-0.5 block" tone="weak">
                      {event.detail}
                    </MetaText>
                  )}
                </div>
              </div>
            ))}
          </TenantCard>
        </div>
      )}

      <Sheet
        modal={false}
        open={Boolean(detailPhase)}
        onOpenChange={open => {
          if (!open) setExpandedPhaseId(null);
        }}
      >
        <SheetContent
          side="right"
          showOverlay={false}
          className="flex h-[100dvh] max-h-[100dvh] w-[min(640px,94vw)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[640px]"
        >
          {detailPhase && (
            <>
              <SheetHeader className="border-b border-[var(--cp-border)] px-6 py-4 pr-12">
                <div className="flex items-center gap-2">
                  <SheetTitle>{detailPhase.title}</SheetTitle>
                  <StatusTag
                    mode="soft"
                    variant={
                      (PHASE_STATUS[detailPhase.status] ?? PHASE_STATUS.ready)
                        .variant
                    }
                  >
                    {(PHASE_STATUS[detailPhase.status] ?? PHASE_STATUS.ready).label}
                  </StatusTag>
                </div>
              </SheetHeader>

              <div
                aria-label="节点执行详情滚动区"
                data-scroll-region="live-workflow-node-detail"
                className="min-h-0 flex-1 touch-pan-y overflow-y-auto overscroll-contain bg-white px-6 [scrollbar-gutter:stable] [&::-webkit-scrollbar]:w-[6px] [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-gray-300 [&::-webkit-scrollbar-track]:bg-transparent"
              >
                <DetailDisclosure
                  title="输入"
                  meta={`${detailInputArtifacts.length} 个文件`}
                  defaultOpen
                >
                  <InputMappingPanel payload={detailInputPayload} />
                  {detailInputMappings.length === 0 && (
                    <NodeDataBlock label="原始需求" value={detailInputPayload?.task?.goal} />
                  )}
                  <NodeDataBlock
                    label="上游输入"
                    value={detailInputPayload?.inputs?.data?.upstream}
                  />
                  {detailInputArtifacts.map((artifact, artifactIndex) => (
                    <WorkflowArtifactPreview
                      key={artifact.artifact_id ?? `${detailPhase.id}-input-${artifactIndex}`}
                      taskId={backendTaskId}
                      artifact={artifact}
                      producerLabel={
                        phases.find(item => item.id === artifactProducerId(artifact))?.title
                      }
                    />
                  ))}
                </DetailDisclosure>

                <DetailDisclosure
                  title="输出"
                  meta={`${detailOutputArtifacts.length} 个文件`}
                  defaultOpen
                >
                  <ExecutionSummaryBlock value={detailResultPayload?.data?.summary} />
                  {!detailResultPayload && (
                    <MetaText tone="weak">
                      {detailPhase.status === "running"
                        ? "Agent 正在执行。"
                        : "节点尚未输出结果。"}
                    </MetaText>
                  )}
                  {detailOutputArtifacts.length > 0 ? (
                    detailOutputArtifacts.map((artifact, artifactIndex) => (
                      <WorkflowArtifactPreview
                        key={
                          artifact.artifact_id ??
                          `${detailPhase.id}-output-${artifactIndex}`
                        }
                        taskId={backendTaskId}
                        artifact={artifact}
                        producerLabel={detailPhase.title}
                      />
                    ))
                  ) : detailResultPayload ? (
                    <MetaText tone="weak">本节点没有文件输出。</MetaText>
                  ) : null}
                </DetailDisclosure>

                <DetailDisclosure
                  title="节点配置"
                  meta={`${detailConfigAssets.length} 项`}
                >
                  {detailConfigAssets.length > 0 ? (
                    <NodeConfigAssets assets={detailConfigAssets} />
                  ) : (
                    <MetaText tone="weak">当前节点没有额外配置。</MetaText>
                  )}
                </DetailDisclosure>

                {detailPending && (
                  <div className="border-t border-[var(--cp-border)] py-4 text-sm text-[var(--cp-brand-blue)]">
                    当前节点已完成，等待你确认后继续。
                  </div>
                )}
              </div>

              {detailPending && (
                <div className="border-t border-[var(--cp-border)] bg-white px-6 py-4">
                  <Button
                    variant="tenant-primary"
                    className="w-full"
                    onClick={() => void handleApprove()}
                    disabled={approving}
                  >
                    {approving ? <Spinner /> : <Check className="h-4 w-4" />}
                    {detailPending.action_label}
                  </Button>
                </div>
              )}
            </>
          )}
        </SheetContent>
      </Sheet>

      <Dialog open={stopOpen} onOpenChange={setStopOpen}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>停止当前工作流？</DialogTitle>
            <DialogDescription>
              停止后不会继续执行后续节点；已经完成的节点、输入输出和产物会继续保留。
            </DialogDescription>
          </DialogHeader>
          <DialogBody>
            <p className="m-0 text-sm text-[var(--text-secondary)]">
              如果 iMate 远端任务已经开始，当前 POC 会停止 ClawPro
              编排并忽略迟到结果；iMate 侧任务可能仍会执行到结束。
            </p>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="tenant-outline"
              onClick={() => setStopOpen(false)}
              disabled={stopping}
            >
              继续执行
            </Button>
            <Button
              variant="tenant-primary"
              className="bg-[var(--text-danger)] hover:bg-[var(--text-danger)]"
              onClick={handleStop}
              disabled={stopping}
            >
              {stopping ? <Spinner /> : <CircleStop className="h-4 w-4" />}
              确认停止
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
