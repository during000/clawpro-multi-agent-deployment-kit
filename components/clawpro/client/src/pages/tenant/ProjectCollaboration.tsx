/**
 * ProjectCollaboration —— 员工端「我的项目 / 项目工作台」
 *
 * 独立路由 /project-collaboration，顶部一级导航入口。
 * 排布参考管控端「项目资产管理」：左侧项目列表 + 右侧工作台面板。
 *
 * 范式：真实项目成员负责，成员授权自己的个人 Agent 参与项目。
 *  - 概览：项目目标 + 健康度指标 + 最近动态；
 *  - 经验：agent 自动上报的经验流 + 被学习(recall)次数（飞轮，无人工新增入口）；
 *  - 资产：6 类资产（项目编辑权限控制成员是否可编辑）。
 *
 * 项目的新建、删除及成员编辑权限仅管理员可管理。
 */
import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { useLocation } from "wouter";
import {
  Users,
  Plus,
  X,
  Pencil,
  Save,
  Check,
  History,
  ShieldCheck,
  Lock,
  Boxes,
  LayoutGrid,
  Lightbulb,
  Repeat,
  Bot,
  BarChart3,
  ClipboardList,
  Search,
  Send,
  CircleCheck,
  Sparkles,
  CalendarDays,
  GitBranch,
  Play,
  Eye,
  FileText,
  Download,
  ChevronRight,
  ChevronDown,
  Paperclip,
  ArrowLeft,
  Trash2,
  Square,
  RotateCcw,
  Upload,
  Workflow,
  Layers,
  TrendingUp,
  AlertTriangle,
  ExternalLink,
  RefreshCw,
  Target,
  LifeBuoy,
  Settings2,
  ClipboardCheck,
  UserRound,
  MoreHorizontal,
  Copy,
  Info,
  Apple,
  Monitor,
} from "lucide-react";
import TenantLayout from "@/components/TenantLayout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { StatusTag } from "@/components/ui/status-tag";
import { Spinner } from "@/components/ui/spinner";
import { TenantCard } from "@/components/ui/Surface";
import { BodyMedium, MetaText } from "@/components/ui/Typography";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  DialogBody,
} from "@/components/ui/dialog";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Empty,
  EmptyContent,
  EmptyHeader,
  EmptyDescription,
} from "@/components/ui/empty";
import { useUserRole } from "@/contexts/UserRoleContext";
import { useProjectCollaborationAccessAllowed } from "@/lib/tenantExternalAgentAccess";
import {
  ASSET_CATEGORY_MAP,
  ASSET_CATEGORY_ORDER,
  ASSET_SYNC_MODE_MAP,
  type AssetCategory,
} from "../admin/project-assets/types";
import {
  getCategoryLibraryItems,
  getAssetItemDisplay,
  getAssetVersionLabel,
} from "../admin/project-assets/assetSelectors";
import AdminAddAssetsDialog from "../admin/project-assets/AddAssetsDialog";
import UpdateRecordsTab from "../admin/project-assets/UpdateRecordsTab";
import {
  tenantProjectStore,
  useTenantProjects,
  useTenantProject,
  type TenantProject,
  type TenantProjectMember,
  type TenantProjectAssets,
  type TenantAgent,
  type TenantLearning,
  type TenantActivity,
  type TenantTask,
  type TenantTaskStatus,
  type TenantTaskPriority,
  type TenantWorkflowTemplateId,
  type TenantWorkflowNode,
  type TenantTaskArtifact,
  type TenantPipelineTemplate,
  type TenantPipelineTemplateNode,
  type TenantProgressSnapshot,
  type TenantReportingSpec,
  type TenantRuntimeExecutionArtifact,
  isoWeekKey,
  TENANT_WORKFLOW_TEMPLATES,
  isTenantExecutableWorkflowTask,
  getProjectIntroSkill,
  type TenantAutomation,
  type TenantAutomationSchedule,
  type TenantAutomationScheduleType,
  type TenantAutomationOutputMode,
} from "./tenantProjectStore";
import AddUsersToGroupDialog from "../admin/MemberManagement/AddUsersToGroupDialog";
import { groupStore } from "../admin/MemberManagement/groupStore";
// P3：工作流可视化画布 + agent 经 MCP 回传
import WorkflowCanvas from "./workflow/WorkflowCanvas";
import McpImportDialog from "./workflow/McpImportDialog";
// 工作流独立 Tab（重构版）
import WorkflowTab from "./workflow/WorkflowTab";
import {
  LiveWorkflowRun,
  LiveWorkflowStartDialog,
} from "./workflow/LiveWorkflowRun";
import { NodeConfigAssets } from "./workflow/NodeConfigAssets";
import {
  getWorkflowTask,
  listWorkflowRuntimeAgents,
  workflowArtifactUrl,
} from "./workflow/clawproWorkflowApi";
// P4：项目级可观测分析 Tab —— 已随重构删除
// P5：PM 多项目汇报统一入口 —— 已随重构删除
import {
  addUsersToGroup,
  removeUserFromGroup,
  userStore,
} from "../admin/MemberManagement/userStore";
import type { UserGroup, UserOrg } from "../admin/MemberManagement/types";

function assetTotal(assets: TenantProjectAssets): number {
  return ASSET_CATEGORY_ORDER.reduce(
    (sum, c) => sum + (assets[c]?.length ?? 0),
    0
  );
}

// ─── 成员标签（与管控端 AssetPanel 一致：灰色标签 + 别名 tooltip） ──
function MemberChips({
  members,
  onRemove,
}: {
  members: TenantProjectMember[];
  onRemove?: (m: TenantProjectMember) => void;
}) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {members.map(u => {
        const hasAlias = !!u.displayName && u.displayName !== u.userId;
        return (
          <span
            key={u.userId}
            className="inline-flex items-center gap-1 h-6 pl-2 pr-1 rounded-[4px] bg-[var(--color-gray-100)] text-xs text-[var(--text-body)] max-w-full"
          >
            {hasAlias ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="truncate max-w-[120px] cursor-default">
                    {u.userId}
                  </span>
                </TooltipTrigger>
                <TooltipContent side="top">
                  {u.userId}（{u.displayName}）
                </TooltipContent>
              </Tooltip>
            ) : (
              <span className="truncate max-w-[120px]">{u.userId}</span>
            )}
            {onRemove && (
              <button
                type="button"
                onClick={e => {
                  e.stopPropagation();
                  onRemove(u);
                }}
                className="shrink-0 w-4 h-4 flex items-center justify-center rounded text-[var(--text-weak)] hover:text-[var(--text-danger)] hover:bg-white transition-colors"
                aria-label={`移除 ${u.userId}`}
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </span>
        );
      })}
    </div>
  );
}

// ─── 添加资产弹窗 ────────────────────────────────────────
function AddAssetDialog({
  open,
  onOpenChange,
  selected,
  onConfirm,
  projectId,
  projectName,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  selected: TenantProjectAssets;
  onConfirm: (next: TenantProjectAssets) => void;
  projectId: string;
  projectName: string;
}) {
  const [directoryGroups, setDirectoryGroups] = useState<UserGroup[]>(() =>
    groupStore.getAll()
  );

  useEffect(
    () => groupStore.subscribe(() => setDirectoryGroups(groupStore.getAll())),
    []
  );

  const groups = useMemo(() => {
    const directoryGroup = directoryGroups.find(
      group => group.id === projectId
    );
    return [
      ...directoryGroups.filter(group => group.id !== projectId),
      {
        id: projectId,
        name: projectName,
        parentId: directoryGroup?.parentId ?? null,
        source: "project" as const,
        readonly: false,
        createdAt: directoryGroup?.createdAt ?? new Date().toISOString(),
      },
    ];
  }, [directoryGroups, projectId, projectName]);

  return (
    <AdminAddAssetsDialog
      open={open}
      onOpenChange={onOpenChange}
      groupId={projectId}
      groupName={projectName}
      groups={groups}
      selectedRefIds={selected}
      onConfirm={onConfirm}
    />
  );
}

// ─── 资产清单（分类标签，样式对齐管控端 AssetPanel） ──────
function AssetList({
  assets,
  editing,
  onRemove,
  project,
  onOpenIntro,
}: {
  assets: TenantProjectAssets;
  editing: boolean;
  onRemove?: (category: AssetCategory, refId: string) => void;
  /** 用于在「企业技能」分类里注入置顶的「项目上手 Skill」特殊项 */
  project: TenantProject;
  onOpenIntro: () => void;
}) {
  const introSkill = getProjectIntroSkill(project);
  // 企业技能分类始终展示（因为项目上手 Skill 是系统必带项，挂在此分类下）
  const nonEmpty = ASSET_CATEGORY_ORDER.filter(
    c => c === "enterpriseSkill" || (assets[c]?.length ?? 0) > 0
  );
  return (
    <div className="space-y-4">
      {nonEmpty.map(category => {
        const isSkillCat = category === "enterpriseSkill";
        const count = (assets[category]?.length ?? 0) + (isSkillCat ? 1 : 0);
        return (
          <div key={category} className="pl-3">
            <MetaText tone="secondary" className="block mb-2">
              {ASSET_CATEGORY_MAP[category].label}
              <span className="ml-1 text-[var(--text-weak)] tabular-nums">
                （{count}）
              </span>
            </MetaText>
            <div className="flex flex-wrap gap-2">
              {/* 项目上手 Skill：置顶、蓝色、可点击弹窗，系统内置不可移除 */}
              {isSkillCat && (
                <button
                  type="button"
                  onClick={onOpenIntro}
                  title="点击查看项目上手 Skill 详情"
                  className="inline-flex items-center gap-1.5 h-7 pl-2.5 pr-2.5 rounded-[4px] border border-[var(--cp-brand-blue)] bg-[var(--bg-brand-selected)] max-w-full hover:bg-[#EAF1FF] transition-colors"
                >
                  <Sparkles className="w-3.5 h-3.5 text-[var(--cp-brand-blue)] shrink-0" />
                  <span className="text-sm text-[var(--text-brand)] truncate max-w-[220px]">
                    项目上手 Skill
                  </span>
                  <span className="text-xs text-[var(--cp-brand-blue)] tabular-nums shrink-0">
                    v{introSkill.version}
                  </span>
                  <span className="text-[10px] px-1 rounded-[3px] bg-white/70 text-[var(--cp-brand-blue)] shrink-0">
                    系统内置
                  </span>
                </button>
              )}
              {(assets[category] ?? []).map(refId => {
                const display = getAssetItemDisplay(category, refId);
                return (
                  <span
                    key={refId}
                    className="inline-flex items-center gap-1.5 h-7 pl-2.5 pr-1 rounded-[4px] border border-[var(--cp-border)] bg-[var(--color-gray-100)] max-w-full"
                    title={display.name}
                  >
                    <span className="text-sm text-[var(--text-body)] truncate max-w-[220px]">
                      {display.name}
                    </span>
                    {display.exists ? (
                      <span className="text-xs text-[var(--text-muted)] tabular-nums shrink-0">
                        {getAssetVersionLabel(category, display.version)}
                      </span>
                    ) : (
                      <span className="text-xs text-[var(--text-danger)] shrink-0">
                        工具库已删除
                      </span>
                    )}
                    {editing ? (
                      <button
                        type="button"
                        onClick={() => onRemove?.(category, refId)}
                        className="shrink-0 w-5 h-5 flex items-center justify-center rounded-[4px] text-[var(--text-weak)] hover:text-[var(--text-danger)] hover:bg-white transition-colors"
                        aria-label={`移除 ${display.name}`}
                      >
                        <X className="w-3.5 h-3.5" />
                      </button>
                    ) : (
                      <span className="w-1 shrink-0" />
                    )}
                  </span>
                );
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ─── 概览 tab ───────────────────────────────────────────
const ACTIVITY_ICON: Record<TenantActivity["kind"], typeof Bot> = {
  learning_report: Lightbulb,
  learning_recall: Repeat,
  asset_add: Boxes,
  task_dispatch: Send,
  task_done: CircleCheck,
  skill_convert: Sparkles,
  report_dispatch: Send,
  report_submit: CircleCheck,
  automation_run: Play,
};

// 旧 OverviewTab（P1–P5 时期版本）已不再使用，重命名保留为死代码以防误引用
// 新版 OverviewTab 见文件下方（TaskTab 之前）
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function _LegacyOverviewTab({ project }: { project: TenantProject }) {
  const m = project.metrics;
  const cards = [
    { icon: Bot, label: "托管 Agent", value: m.agentCount, unit: "" },
    {
      icon: ClipboardList,
      label: "进行中任务",
      value: m.inProgressTasks,
      unit: "",
    },
    { icon: Lightbulb, label: "沉淀经验", value: m.learningCount, unit: "" },
    { icon: Repeat, label: "本周被学习", value: m.weekRecalled, unit: "次" },
  ];
  return (
    <div className="space-y-4">
      {/* 目标 */}
      <div className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-4 py-3">
        <MetaText tone="secondary" className="block mb-1">
          项目目标
        </MetaText>
        <div className="text-sm text-[var(--text-body)]">{project.goal}</div>
      </div>

      {/* 指标卡 ×4 */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        {cards.map(c => (
          <div
            key={c.label}
            className="rounded-[4px] border border-[var(--cp-border)] bg-white px-4 py-3"
          >
            <div className="flex items-center gap-1.5 text-xs text-[var(--text-muted)] mb-1.5">
              <c.icon className="w-3.5 h-3.5" />
              {c.label}
            </div>
            <div className="text-2xl font-semibold text-[var(--text-title)] tabular-nums leading-none">
              {c.value}
              {c.unit && (
                <span className="text-sm font-normal text-[var(--text-muted)] ml-1">
                  {c.unit}
                </span>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* 最近动态 */}
      <div className="rounded-[4px] border border-[var(--cp-border)] bg-white">
        <div className="px-4 py-3 border-b border-[var(--cp-border)]">
          <BodyMedium>最近动态</BodyMedium>
        </div>
        <div className="p-2">
          {project.activities.length === 0 ? (
            <Empty className="py-8">
              <EmptyHeader>
                <EmptyDescription>暂无动态</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <ul className="space-y-0.5">
              {project.activities.map(a => {
                const Icon = ACTIVITY_ICON[a.kind];
                return (
                  <li
                    key={a.id}
                    className="flex items-center gap-2.5 px-2 py-2 rounded-[4px] hover:bg-[var(--bg-grey-hover)]"
                  >
                    <span className="shrink-0 w-6 h-6 rounded-full bg-[var(--color-gray-100)] flex items-center justify-center text-[var(--text-secondary)]">
                      <Icon className="w-3.5 h-3.5" />
                    </span>
                    <span className="flex-1 text-sm text-[var(--text-body)] truncate">
                      {a.text}
                    </span>
                    <span className="shrink-0 text-xs text-[var(--text-weak)]">
                      {a.time}
                    </span>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}

// ─── 进展 tab（场景1：管理员定规格 → 派汇报 → agent 产出快照 → 聚合下钻） ──
/** 演示态：当前登录用户（对齐 store 里的 CURRENT_USER） */
const CURRENT_USER = "alice@acompany.com";

const REPORT_FIELD_ICON: Record<string, typeof Bot> = {
  summary: TrendingUp,
  risk: AlertTriangle,
  next: Target,
  support: LifeBuoy,
};

/** 进度底座条：完成度 + 任务数 + 卡点，系统机械算，不让人重打 */
function ProgressBaseView({
  base,
}: {
  base: TenantProgressSnapshot["progressBase"];
}) {
  return (
    <div className="rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-[var(--color-gray-50)] px-3 py-2.5 space-y-2">
      <div className="flex items-center gap-1.5">
        <BarChart3 className="w-3.5 h-3.5 text-[var(--text-secondary)]" />
        <MetaText tone="secondary">进度底座（系统自动，来自任务）</MetaText>
      </div>
      <div className="flex items-center gap-2">
        <div className="h-1.5 flex-1 overflow-hidden rounded-full bg-[var(--color-gray-200)]">
          <div
            className="h-full rounded-full bg-[var(--cp-brand-blue)]"
            style={{ width: `${base.completion}%` }}
          />
        </div>
        <span className="shrink-0 text-xs font-medium tabular-nums text-[var(--text-body)]">
          {base.completion}%
        </span>
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-[var(--text-muted)]">
        <span>
          任务 {base.taskDone}/{base.taskTotal} 完成
        </span>
        <span>进行中 {base.taskActive}</span>
      </div>
      {base.blockers.length > 0 && (
        <ul className="space-y-0.5">
          {base.blockers.map((b, i) => (
            <li
              key={i}
              className="flex items-start gap-1 text-[11px] text-[var(--text-warning,#B45309)]"
            >
              <AlertTriangle className="mt-0.5 w-3 h-3 shrink-0" />
              <span>{b}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/** 汇报字段填写弹窗（成员/其 agent 按规格产出解读层；底座只读展示） */
function FillSnapshotDialog({
  open,
  onOpenChange,
  spec,
  snapshot,
  onSubmit,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  spec: TenantReportingSpec;
  snapshot: TenantProgressSnapshot | null;
  onSubmit: (interpretation: Record<string, string>) => void;
}) {
  const [values, setValues] = useState<Record<string, string>>({});

  useEffect(() => {
    if (open && snapshot) setValues({ ...snapshot.interpretation });
  }, [open, snapshot]);

  /** 模拟 agent 按规格草拟解读（演示态：真实环境由 agent 依 skill 产出） */
  const autofillByAgent = () => {
    if (!snapshot) return;
    const base = snapshot.progressBase;
    setValues({
      summary: `本周期完成度 ${base.completion}%，已完成 ${base.taskDone}/${base.taskTotal} 项任务，进行中 ${base.taskActive} 项。`,
      risk:
        base.blockers.length > 0
          ? base.blockers.join("；")
          : "暂无明显风险，节点按计划推进。",
      next: "推进剩余进行中任务，优先清理待确认节点。",
      support: "暂无需要额外协调的资源。",
    });
    toast.info("已由 agent 按规格草拟解读，可修改后提交");
  };

  if (!snapshot) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>填写进展快照 · {snapshot.periodLabel}</DialogTitle>
          <DialogDescription>
            客观进度由系统从任务自动带出；你（或你的
            agent）只需补充机器给不出的解读判断。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="space-y-3">
          <ProgressBaseView base={snapshot.progressBase} />
          <div className="flex items-center justify-between">
            <MetaText tone="secondary">解读层（按项目汇报规格）</MetaText>
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={autofillByAgent}
            >
              <Bot className="w-4 h-4" />让 agent 按规格草拟
            </Button>
          </div>
          {spec.fields.map(f => {
            const Icon = REPORT_FIELD_ICON[f.key] ?? FileText;
            return (
              <div key={f.key} className="space-y-1.5">
                <Label className="flex items-center gap-1.5">
                  <Icon className="w-3.5 h-3.5 text-[var(--text-secondary)]" />
                  {f.label}
                </Label>
                <Textarea
                  rows={2}
                  value={values[f.key] ?? ""}
                  onChange={e =>
                    setValues(prev => ({ ...prev, [f.key]: e.target.value }))
                  }
                  placeholder={f.placeholder}
                />
              </div>
            );
          })}
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="tenant-primary"
            onClick={() => {
              onSubmit(values);
              onOpenChange(false);
            }}
          >
            提交快照
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** 汇报规格配置弹窗（管理员）：周期 + 汇报人范围（起步字段固定，只读展示） */
function ReportingSpecDialog({
  open,
  onOpenChange,
  project,
  onSave,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  project: TenantProject;
  onSave: (patch: Partial<TenantReportingSpec>) => void;
}) {
  const spec = project.reportingSpec;
  const [enabled, setEnabled] = useState(spec.enabled);
  const [cycle, setCycle] = useState(spec.cycle);
  const [scope, setScope] = useState<string[]>(spec.reporterScope);

  useEffect(() => {
    if (open) {
      setEnabled(spec.enabled);
      setCycle(spec.cycle);
      setScope(spec.reporterScope);
    }
  }, [open, spec]);

  const toggleScope = (userId: string) =>
    setScope(prev =>
      prev.includes(userId)
        ? prev.filter(id => id !== userId)
        : [...prev, userId]
    );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>汇报规格 · 项目配置</DialogTitle>
          <DialogDescription>
            管理员对项目的整体规划：大家怎么汇报。这套规格会随汇报任务作为 skill
            下发给成员的 agent。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="space-y-4">
          <label className="flex items-center justify-between gap-3 rounded-[var(--radius-md)] border border-[var(--cp-border)] px-3 py-2.5 cursor-pointer">
            <span className="text-sm text-[var(--text-body)]">
              启用周期汇报
            </span>
            <input
              type="checkbox"
              checked={enabled}
              onChange={e => setEnabled(e.target.checked)}
              className="h-4 w-4 accent-[var(--cp-brand-blue)]"
            />
          </label>

          <div className="space-y-1.5">
            <Label>汇报周期</Label>
            <Select
              value={cycle}
              onValueChange={v => setCycle(v as TenantReportingSpec["cycle"])}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="weekly">每周</SelectItem>
                <SelectItem value="biweekly">每两周</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label>汇报字段（起步固定四项）</Label>
            <div className="flex flex-wrap gap-1.5">
              {spec.fields.map(f => (
                <StatusTag key={f.key} variant="gray" mode="soft">
                  {f.label}
                </StatusTag>
              ))}
            </div>
            <MetaText tone="weak">后续可扩展为自定义表单。</MetaText>
          </div>

          <div className="space-y-1.5">
            <Label>汇报人范围（不选=全体成员）</Label>
            <div className="space-y-1">
              {project.members.map(m => (
                <label
                  key={m.userId}
                  className="flex items-center gap-2 rounded-[4px] px-2 py-1.5 hover:bg-[var(--bg-grey-hover)] cursor-pointer"
                >
                  <input
                    type="checkbox"
                    checked={scope.includes(m.userId)}
                    onChange={() => toggleScope(m.userId)}
                    className="h-4 w-4 accent-[var(--cp-brand-blue)]"
                  />
                  <span className="text-sm text-[var(--text-body)]">
                    {m.displayName}
                  </span>
                </label>
              ))}
            </div>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="tenant-primary"
            onClick={() => {
              onSave({ enabled, cycle, reporterScope: scope });
              onOpenChange(false);
            }}
          >
            保存规格
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** 单张成员快照卡（聚合视图里可展开下钻） */
function SnapshotCard({
  snapshot,
  spec,
  isMine,
  onFill,
  onOpenTask,
}: {
  snapshot: TenantProgressSnapshot;
  spec: TenantReportingSpec;
  isMine: boolean;
  onFill: () => void;
  onOpenTask: (taskId: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const submitted = snapshot.status === "submitted";
  return (
    <div className="rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white">
      <div className="flex items-center gap-3 px-4 py-3">
        <span className="shrink-0 flex h-8 w-8 items-center justify-center rounded-full bg-[var(--color-gray-100)] text-sm font-medium text-[var(--text-secondary)]">
          {snapshot.reporterName.slice(0, 1).toUpperCase()}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-[var(--text-title)]">
              {snapshot.reporterName}
            </span>
            {submitted ? (
              <StatusTag variant="green" mode="soft">
                已提交
              </StatusTag>
            ) : (
              <StatusTag variant="orange" mode="soft">
                待填写
              </StatusTag>
            )}
          </div>
          <div className="mt-1 flex items-center gap-2">
            <div className="h-1.5 w-24 overflow-hidden rounded-full bg-[var(--color-gray-200)]">
              <div
                className="h-full rounded-full bg-[var(--cp-brand-blue)]"
                style={{ width: `${snapshot.progressBase.completion}%` }}
              />
            </div>
            <span className="text-[11px] tabular-nums text-[var(--text-muted)]">
              {snapshot.progressBase.completion}% · 任务{" "}
              {snapshot.progressBase.taskDone}/{snapshot.progressBase.taskTotal}
            </span>
          </div>
        </div>
        {!submitted && isMine && (
          <Button variant="tenant-primary" size="sm" onClick={onFill}>
            <Bot className="w-4 h-4" />
            生成快照
          </Button>
        )}
        <button
          type="button"
          onClick={() => setOpen(v => !v)}
          className="shrink-0 flex h-8 w-8 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-muted)] hover:bg-[var(--color-gray-100)]"
          aria-label="展开快照"
        >
          {open ? (
            <ChevronDown className="w-4 h-4" />
          ) : (
            <ChevronRight className="w-4 h-4" />
          )}
        </button>
      </div>
      {open && (
        <div className="border-t border-[var(--cp-border)] px-4 py-3 space-y-3">
          <ProgressBaseView base={snapshot.progressBase} />
          {submitted ? (
            <div className="space-y-2">
              <MetaText tone="secondary">解读层</MetaText>
              {spec.fields.map(f => {
                const Icon = REPORT_FIELD_ICON[f.key] ?? FileText;
                const val = snapshot.interpretation[f.key];
                if (!val) return null;
                return (
                  <div key={f.key} className="space-y-0.5">
                    <div className="flex items-center gap-1.5 text-xs text-[var(--text-muted)]">
                      <Icon className="w-3.5 h-3.5" />
                      {f.label}
                    </div>
                    <p className="text-sm text-[var(--text-body)] whitespace-pre-wrap m-0">
                      {val}
                    </p>
                  </div>
                );
              })}
            </div>
          ) : (
            <MetaText tone="weak">
              解读层尚未产出。
              {isMine
                ? "点击「生成快照」由 agent 按规格草拟。"
                : "等待该成员的 agent 提交。"}
            </MetaText>
          )}
          {/* 下钻：引用的任务 */}
          {snapshot.progressBase.taskIds.length > 0 && (
            <div className="space-y-1">
              <MetaText tone="secondary">下钻到任务</MetaText>
              <div className="flex flex-wrap gap-1.5">
                {snapshot.progressBase.taskIds.map(tid => (
                  <button
                    key={tid}
                    type="button"
                    onClick={() => onOpenTask(tid)}
                    className="inline-flex items-center gap-1 h-6 px-2 rounded-[4px] border border-[var(--cp-border)] bg-white text-[11px] text-[var(--text-body)] hover:border-[var(--cp-brand-blue)]"
                  >
                    <ClipboardList className="w-3 h-3" />
                    查看任务
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function ProgressTab({
  project,
  isAdmin,
  onOpenTask,
}: {
  project: TenantProject;
  isAdmin: boolean;
  onOpenTask: (taskId: string) => void;
}) {
  const spec = project.reportingSpec;
  const { key: periodKey, label: periodLabel } = isoWeekKey();
  const [specOpen, setSpecOpen] = useState(false);
  const [fillId, setFillId] = useState<string | null>(null);

  const periodSnaps = project.progressSnapshots.filter(
    s => s.period === periodKey
  );
  const fillSnap = project.progressSnapshots.find(s => s.id === fillId) ?? null;

  const submittedCount = periodSnaps.filter(
    s => s.status === "submitted"
  ).length;
  const aggCompletion =
    periodSnaps.length === 0
      ? 0
      : Math.round(
          periodSnaps.reduce((s, x) => s + x.progressBase.completion, 0) /
            periodSnaps.length
        );

  const dispatch = () => {
    const n = tenantProjectStore.dispatchReports(project.id);
    if (n > 0) toast.success(`已派发本周期汇报任务给 ${n} 名成员`);
    else toast.info("本周期汇报任务已全部派发");
  };

  return (
    <div className="space-y-4">
      {/* 规格状态条 + 管理员操作 */}
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-[var(--radius-card)] border border-[#93C5FD] bg-[#F8FBFF] px-4 py-3">
        <div className="flex items-start gap-2.5 min-w-0">
          <ClipboardCheck className="mt-0.5 w-4 h-4 shrink-0 text-[#1447E6]" />
          <div className="min-w-0">
            <div className="text-sm font-medium text-[var(--text-body)]">
              {spec.enabled ? (
                <>
                  周期汇报已启用 · {spec.cycle === "weekly" ? "每周" : "每两周"}{" "}
                  ·{" "}
                  {spec.reporterScope.length > 0
                    ? `${spec.reporterScope.length} 名汇报人`
                    : "全体成员"}
                </>
              ) : (
                "周期汇报未启用"
              )}
            </div>
            <MetaText tone="secondary">
              进展 = 系统自动进度底座 + agent
              按统一规格回传的解读；成员保持原工作习惯，无需手工重复录入。
            </MetaText>
          </div>
        </div>
        {isAdmin && (
          <div className="flex items-center gap-2">
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={() => setSpecOpen(true)}
            >
              <Settings2 className="w-4 h-4" />
              配置规格
            </Button>
            <Button
              variant="tenant-primary"
              size="sm"
              disabled={!spec.enabled}
              onClick={dispatch}
            >
              <Send className="w-4 h-4" />
              派发本周期汇报
            </Button>
          </div>
        )}
      </div>

      {/* 聚合视图（领导/管理员机械 roll-up） */}
      {periodSnaps.length > 0 && (
        <div className="grid grid-cols-3 gap-3">
          <div className="rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white px-4 py-3">
            <div className="flex items-center gap-1.5 text-xs text-[var(--text-muted)] mb-1.5">
              <BarChart3 className="w-3.5 h-3.5" />
              项目整体完成度
            </div>
            <div className="text-2xl font-semibold text-[var(--text-title)] tabular-nums leading-none">
              {aggCompletion}
              <span className="text-sm font-normal text-[var(--text-muted)] ml-0.5">
                %
              </span>
            </div>
          </div>
          <div className="rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white px-4 py-3">
            <div className="flex items-center gap-1.5 text-xs text-[var(--text-muted)] mb-1.5">
              <ClipboardCheck className="w-3.5 h-3.5" />
              已交快照
            </div>
            <div className="text-2xl font-semibold text-[var(--text-title)] tabular-nums leading-none">
              {submittedCount}
              <span className="text-sm font-normal text-[var(--text-muted)] ml-0.5">
                /{periodSnaps.length}
              </span>
            </div>
          </div>
          <div className="rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white px-4 py-3">
            <div className="flex items-center gap-1.5 text-xs text-[var(--text-muted)] mb-1.5">
              <CalendarDays className="w-3.5 h-3.5" />
              当前周期
            </div>
            <div className="text-sm font-medium text-[var(--text-title)] leading-tight pt-1">
              {periodLabel}
            </div>
          </div>
        </div>
      )}

      {/* 成员快照列表 */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <BodyMedium>{periodLabel} · 成员进展快照</BodyMedium>
        </div>
        {periodSnaps.length === 0 ? (
          <Empty className="py-10">
            <EmptyHeader>暂无本周期进展</EmptyHeader>
            <EmptyDescription>
              {spec.enabled
                ? isAdmin
                  ? "点击「派发本周期汇报」，给成员各建一条汇报任务与待填快照。"
                  : "等待管理员派发本周期汇报任务。"
                : "管理员启用周期汇报后，即可派发汇报任务。"}
            </EmptyDescription>
          </Empty>
        ) : (
          <div className="space-y-2">
            {periodSnaps.map(s => (
              <SnapshotCard
                key={s.id}
                snapshot={s}
                spec={spec}
                isMine={s.reporterId === CURRENT_USER}
                onFill={() => setFillId(s.id)}
                onOpenTask={onOpenTask}
              />
            ))}
          </div>
        )}
      </div>

      <ReportingSpecDialog
        open={specOpen}
        onOpenChange={setSpecOpen}
        project={project}
        onSave={patch => {
          tenantProjectStore.saveReportingSpec(project.id, patch);
          toast.success("汇报规格已保存");
        }}
      />
      <FillSnapshotDialog
        open={fillId !== null}
        onOpenChange={v => !v && setFillId(null)}
        spec={spec}
        snapshot={fillSnap}
        onSubmit={interpretation => {
          if (fillSnap)
            tenantProjectStore.submitSnapshot(
              project.id,
              fillSnap.id,
              interpretation
            );
          toast.success("进展快照已提交");
        }}
      />
    </div>
  );
}

// ─── 经验 tab（飞轮） ────────────────────────────────────
function ExperienceAutomationControls({
  project,
}: {
  project: TenantProject;
}) {
  const legacyFlywheelEnabled = project.experienceFlywheelEnabled !== false;
  const tokenSavingEnabled =
    project.experienceRecallEnabled ?? legacyFlywheelEnabled;
  const experienceDepositionEnabled =
    project.experienceDepositionEnabled ?? legacyFlywheelEnabled;
  const estimatedTokensSaved = project.learnings.reduce(
    (total, learning) =>
      total +
      learning.recalledCount *
        Math.max(120, Math.ceil(learning.summary.length / 1.5)),
    0
  );
  const tokenSavedLabel =
    estimatedTokensSaved >= 1000
      ? `${(estimatedTokensSaved / 1000).toFixed(1)}K`
      : estimatedTokensSaved.toLocaleString();

  const toggleExperienceAutomation = (
    capability: "recall" | "deposition",
    enabled: boolean
  ) => {
    tenantProjectStore.setExperienceAutomationEnabled(
      project.id,
      capability,
      enabled
    );
    toast.success(
      `${capability === "recall" ? "Token 节省" : "经验沉淀"}已${enabled ? "开启" : "关闭"}`
    );
  };

  return (
    <div className="rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white">
      <div className="flex items-center justify-between border-b border-[var(--cp-border)] px-4 py-3">
        <div>
          <BodyMedium>Agent 自动化</BodyMedium>
          <MetaText tone="weak" className="ml-2">按项目独立控制</MetaText>
        </div>
        <span className="text-xs text-[var(--text-weak)]">历史数据不会因关闭而清除</span>
      </div>
      <div className="grid grid-cols-2 divide-x divide-[var(--cp-border)]">
        <div className="flex items-center gap-3 px-4 py-3">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[6px] bg-[var(--cp-brand-tint)] text-[var(--cp-brand-blue)]">
            <BarChart3 className="h-4 w-4" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline gap-2">
              <BodyMedium>Token 节省</BodyMedium>
              <span className="text-sm font-medium tabular-nums text-[var(--text-title)]">{tokenSavedLabel} Token</span>
            </div>
            <MetaText tone="weak" className="block truncate">任务前自动检索项目经验</MetaText>
          </div>
          <Button
            variant={tokenSavingEnabled ? "tenant-primary" : "tenant-outline"}
            size="sm"
            className="shrink-0"
            aria-pressed={tokenSavingEnabled}
            onClick={() =>
              toggleExperienceAutomation("recall", !tokenSavingEnabled)
            }
          >
            {tokenSavingEnabled ? "已开启" : "已关闭"}
          </Button>
        </div>
        <div className="flex items-center gap-3 px-4 py-3">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[6px] bg-[var(--cp-brand-tint)] text-[var(--cp-brand-blue)]">
            <Repeat className="h-4 w-4" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline gap-2">
              <BodyMedium>经验沉淀</BodyMedium>
              <span className="text-sm font-medium tabular-nums text-[var(--text-title)]">{project.metrics.learningCount} 条</span>
            </div>
            <MetaText tone="weak" className="block truncate">Agent 自动上报踩坑与新解法</MetaText>
          </div>
          <Button
            variant={
              experienceDepositionEnabled
                ? "tenant-primary"
                : "tenant-outline"
            }
            size="sm"
            className="shrink-0"
            aria-pressed={experienceDepositionEnabled}
            onClick={() =>
              toggleExperienceAutomation(
                "deposition",
                !experienceDepositionEnabled
              )
            }
          >
            {experienceDepositionEnabled ? "已开启" : "已关闭"}
          </Button>
        </div>
      </div>
    </div>
  );
}

function ExperienceTab({ project }: { project: TenantProject }) {
  const [selectedLearning, setSelectedLearning] =
    useState<TenantLearning | null>(null);
  const [assetLearning, setAssetLearning] = useState<TenantLearning | null>(
    null
  );
  const [assetType, setAssetType] = useState<"skill" | "rules">("skill");
  const [assetName, setAssetName] = useState("");
  const [configuredLearningIds, setConfiguredLearningIds] = useState<
    Set<string>
  >(() => new Set());

  const openAssetConfig = (learning: TenantLearning) => {
    setAssetLearning(learning);
    setAssetType("skill");
    setAssetName(learning.title);
  };

  const confirmAssetConfig = () => {
    if (!assetLearning || !assetName.trim()) return;
    setConfiguredLearningIds(current => {
      const next = new Set(current);
      next.add(assetLearning.id);
      return next;
    });
    toast.success(
      `《${assetLearning.title}》已配置为${assetType === "skill" ? " Skill" : " Rules"} 资产`
    );
    setAssetLearning(null);
  };

  return (
    <div className="space-y-4">
      {/* 经验流 */}
      {project.learnings.length === 0 ? (
        <Empty className="py-12">
          <EmptyHeader>
            <EmptyDescription>项目 agent 暂未上报经验</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-white">
          {project.learnings.map(l => (
            <div
              key={l.id}
              className="flex items-center gap-3 border-b border-[var(--cp-border)] px-4 py-3 last:border-b-0"
            >
              <button
                type="button"
                className="flex min-w-0 flex-1 items-center justify-between gap-3 text-left"
                onClick={() => setSelectedLearning(l)}
              >
                <div className="truncate text-sm font-medium text-[var(--text-title)]">
                  {l.title}
                </div>
                <ChevronRight className="h-4 w-4 shrink-0 text-[var(--text-weak)]" />
              </button>
              <Button
                variant="tenant-outline"
                size="sm"
                className="shrink-0"
                disabled={configuredLearningIds.has(l.id)}
                onClick={() => openAssetConfig(l)}
              >
                {configuredLearningIds.has(l.id) ? "已配置" : "配置为资产"}
              </Button>
            </div>
          ))}
        </div>
      )}

      <Sheet
        open={Boolean(selectedLearning)}
        onOpenChange={open => {
          if (!open) setSelectedLearning(null);
        }}
      >
        <SheetContent
          side="right"
          className="flex w-[520px] flex-col p-0 sm:max-w-[520px]"
        >
          <SheetHeader className="border-b border-[var(--cp-border)] px-6 py-4">
            <SheetTitle>经验详情</SheetTitle>
          </SheetHeader>
          {selectedLearning && (
            <div className="flex-1 space-y-5 overflow-y-auto px-6 py-5">
              <div>
                <div className="text-base font-medium leading-6 text-[var(--text-title)]">
                  {selectedLearning.title}
                </div>
                <div className="mt-3 flex flex-wrap gap-2">
                  {selectedLearning.tags.map(tag => (
                    <StatusTag key={tag} variant="gray" mode="soft">
                      {tag}
                    </StatusTag>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3 rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-4">
                <div>
                  <MetaText tone="weak">来源 Agent</MetaText>
                  <BodyMedium className="mt-1 block">
                    {selectedLearning.sourceAgent}
                  </BodyMedium>
                </div>
                <div>
                  <MetaText tone="weak">上报时间</MetaText>
                  <BodyMedium className="mt-1 block">
                    {selectedLearning.time}
                  </BodyMedium>
                </div>
                <div>
                  <MetaText tone="weak">触发场景</MetaText>
                  <BodyMedium className="mt-1 block">
                    {selectedLearning.scene}
                  </BodyMedium>
                </div>
                <div>
                  <MetaText tone="weak">被学习次数</MetaText>
                  <BodyMedium className="mt-1 block">
                    {selectedLearning.recalledCount} 次
                  </BodyMedium>
                </div>
              </div>

              <div>
                <BodyMedium>经验内容</BodyMedium>
                <p className="mt-2 whitespace-pre-wrap text-sm leading-6 text-[var(--text-secondary)]">
                  {selectedLearning.summary}
                </p>
              </div>

              <Button
                variant="tenant-primary"
                className="w-full"
                disabled={configuredLearningIds.has(selectedLearning.id)}
                onClick={() => {
                  setSelectedLearning(null);
                  openAssetConfig(selectedLearning);
                }}
              >
                {configuredLearningIds.has(selectedLearning.id)
                  ? "已配置为资产"
                  : "配置为资产"}
              </Button>
            </div>
          )}
        </SheetContent>
      </Sheet>

      <Dialog
        open={Boolean(assetLearning)}
        onOpenChange={open => {
          if (!open) setAssetLearning(null);
        }}
      >
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>配置为资产</DialogTitle>
            <DialogDescription>
              将这条经验转换为可下发给项目 Agent 的资产。
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-4">
            <div className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 py-2.5">
              <MetaText tone="weak">经验</MetaText>
              <BodyMedium className="mt-1 block">
                {assetLearning?.title}
              </BodyMedium>
            </div>
            <div className="space-y-2">
              <Label htmlFor="experience-asset-name">资产名称</Label>
              <Input
                id="experience-asset-name"
                value={assetName}
                onChange={event => setAssetName(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>资产类型</Label>
              <Select
                value={assetType}
                onValueChange={value =>
                  setAssetType(value as "skill" | "rules")
                }
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="skill">Skill</SelectItem>
                  <SelectItem value="rules">Rules</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="tenant-outline"
              onClick={() => setAssetLearning(null)}
            >
              取消
            </Button>
            <Button
              variant="tenant-primary"
              disabled={!assetName.trim()}
              onClick={confirmAssetConfig}
            >
              确认配置
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ─── 任务 tab：成员负责 + 个人 Agent 执行 + 工作流确认 ─────
const TASK_COLUMNS: {
  key: string;
  label: string;
  statuses: TenantTaskStatus[];
  dotClass: string;
}[] = [
  {
    key: "todo",
    label: "待启动",
    statuses: ["todo"],
    dotClass: "bg-[var(--text-weak)]",
  },
  {
    key: "in_progress",
    label: "进行中",
    statuses: ["in_progress", "review"],
    dotClass: "bg-[var(--cp-brand-blue)]",
  },
  {
    key: "done",
    label: "已完成",
    statuses: ["done"],
    dotClass: "bg-[var(--text-success)]",
  },
  {
    key: "hold",
    label: "挂起",
    statuses: ["hold"],
    dotClass: "bg-[var(--text-warning)]",
  },
];

/** 单个状态 → 显示 label（用于任务卡/详情） */
const TASK_STATUS_LABEL: Record<TenantTaskStatus, string> = {
  todo: "待启动",
  in_progress: "进行中",
  review: "进行中",
  done: "已完成",
  hold: "挂起",
};

const TASK_STATUS_VARIANT: Record<
  TenantTaskStatus,
  "gray" | "blue" | "green" | "orange"
> = {
  todo: "gray",
  in_progress: "blue",
  review: "blue",
  done: "green",
  hold: "orange",
};

function formatDueDate(date: string) {
  const [, month, day] = date.split("-");
  return `${month}-${day}`;
}

/** 任务节点进度：已确认节点数 / 总节点数（供看板卡片展示 agent 执行进度） */
function taskNodeProgress(task: TenantTask): {
  confirmed: number;
  total: number;
  percent: number;
} {
  const total = task.workflow.length;
  const confirmed = task.workflow.filter(n => n.status === "confirmed").length;
  const percent = total === 0 ? 0 : Math.round((confirmed / total) * 100);
  return { confirmed, total, percent };
}

function downloadArtifact(artifact: TenantTaskArtifact) {
  const blob = new Blob([artifact.content], {
    type:
      artifact.type === "JSON"
        ? "application/json;charset=utf-8"
        : "text/markdown;charset=utf-8",
  });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = artifact.name;
  anchor.click();
  URL.revokeObjectURL(url);
}

function CreateTaskDialog({
  open,
  onOpenChange,
  pipelineTemplates,
  onCreate,
  onGotoWorkflow,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  pipelineTemplates: TenantPipelineTemplate[];
  onCreate: (input: {
    title: string;
    description: string;
    priority: TenantTaskPriority;
    dueDate: string;
    workflowTemplateId: TenantWorkflowTemplateId;
    pipelineTemplateId: string;
  }) => void;
  /** 点"+ 新建工作流"时关闭本弹窗、切到工作流 Tab */
  onGotoWorkflow: () => void;
}) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [dueDate, setDueDate] = useState("2026-08-15");
  // workflowTemplateId 保留字段做 store 兼容；实际链路统一走 pipelineTemplateId
  const [workflowTemplateId, setWorkflowTemplateId] =
    useState<TenantWorkflowTemplateId>("auto");
  const [pipelineTemplateId, setPipelineTemplateId] = useState<string>("");
  const selectedPipeline = pipelineTemplates.find(
    t => t.id === pipelineTemplateId
  );

  const reset = () => {
    setTitle("");
    setDescription("");
    setDueDate("2026-08-15");
    setWorkflowTemplateId("auto");
    setPipelineTemplateId("");
  };

  useEffect(() => {
    if (open) reset();
  }, [open]);

  return (
    <Dialog
      open={open}
      onOpenChange={next => {
        onOpenChange(next);
      }}
    >
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>新建任务</DialogTitle>
          <DialogDescription>
            创建后再进入任务流程图，为每个节点选择成员授权的 Agent。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="px-6">
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="project-task-title">任务名称</Label>
              <Input
                id="project-task-title"
                value={title}
                onChange={event => setTitle(event.target.value)}
                placeholder="输入任务名称"
                maxLength={60}
              />
            </div>
            {pipelineTemplates.length > 0 && (
              <div className="space-y-2">
                <Label>工作流</Label>
                <Select
                  value={pipelineTemplateId}
                  onValueChange={v => {
                    if (v === "__create__") {
                      onOpenChange(false);
                      onGotoWorkflow();
                      return;
                    }
                    setPipelineTemplateId(v);
                  }}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择一条工作流" />
                  </SelectTrigger>
                  <SelectContent>
                    {pipelineTemplates.map(t => (
                      <SelectItem key={t.id} value={t.id}>
                        <span className="inline-flex items-center gap-1.5">
                          <Workflow className="w-3 h-3" />
                          {t.name}
                          <span className="text-[var(--text-weak)]">
                            · {t.nodes.length} 节点
                          </span>
                        </span>
                      </SelectItem>
                    ))}
                    <SelectItem value="__create__">
                      <span className="inline-flex items-center gap-1.5 text-[var(--cp-brand-blue)]">
                        <Plus className="w-3 h-3" />
                        新建工作流…
                      </span>
                    </SelectItem>
                  </SelectContent>
                </Select>
                {selectedPipeline && (
                  <MetaText tone="weak" className="block">
                    将按「{selectedPipeline.name}」实例化 workflow：{" "}
                    {selectedPipeline.nodes.map(n => n.title).join(" → ")}
                  </MetaText>
                )}
              </div>
            )}
            {pipelineTemplates.length === 0 && (
              <div className="space-y-2">
                <Label>工作流</Label>
                <button
                  type="button"
                  onClick={() => {
                    onOpenChange(false);
                    onGotoWorkflow();
                  }}
                  className="flex h-9 w-full items-center justify-between gap-3 rounded-[var(--radius-card)] border border-dashed border-[var(--cp-border)] bg-white px-3 text-left transition-colors hover:border-[var(--cp-brand-blue)]"
                >
                  <span className="inline-flex items-center gap-2 text-sm text-[var(--text-secondary)]">
                    <Plus className="h-4 w-4" />
                    去工作流 Tab 新建
                  </span>
                  <ChevronRight className="h-4 w-4 text-[var(--text-weak)]" />
                </button>
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="project-task-due-date">截止日期</Label>
              <Input
                id="project-task-due-date"
                type="date"
                value={dueDate}
                onChange={event => setDueDate(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="project-task-description">
                执行提示词（Prompt）
              </Label>
              <Textarea
                id="project-task-description"
                value={description}
                onChange={event => setDescription(event.target.value)}
                placeholder="作为输入下发给工作流的第一个节点"
                className="min-h-20"
              />
              <MetaText tone="weak" className="block">
                该 Prompt
                将作为输入下发给工作流的第一个节点。自动化触发的任务会自动带入自动化规则中配置的提示词。
              </MetaText>
            </div>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="tenant-primary"
            disabled={!title.trim() || !pipelineTemplateId || !dueDate}
            onClick={() => {
              onCreate({
                title: title.trim(),
                description: description.trim(),
                priority: "medium",
                dueDate,
                workflowTemplateId,
                pipelineTemplateId: pipelineTemplateId!,
              });
              onOpenChange(false);
            }}
          >
            创建
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function TaskCard({
  task,
  onOpen,
  onDelete,
}: {
  task: TenantTask;
  onOpen: () => void;
  onDelete: () => void;
}) {
  const progress = taskNodeProgress(task);
  const hasReviewNode = task.workflow.some(node => node.status === "review");
  return (
    <TenantCard
      padding="none"
      className="p-2.5"
      interactive
      role="button"
      tabIndex={0}
      aria-label={`打开任务：${task.title}`}
      onClick={onOpen}
      onKeyDown={event => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onOpen();
        }
      }}
    >
      <div className="flex items-start gap-2">
        <div className="min-w-0 flex-1 truncate text-sm font-medium leading-5 text-[var(--text-title)]">
          {task.title}
        </div>
        {hasReviewNode && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span
                role="status"
                aria-label="有节点待确认"
                className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-[var(--cp-brand-blue)]"
              />
            </TooltipTrigger>
            <TooltipContent>有节点待确认</TooltipContent>
          </Tooltip>
        )}
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              aria-label={`删除任务：${task.title}`}
              className="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-weak)] hover:bg-[var(--color-gray-100)] hover:text-[var(--text-danger)]"
              onClick={event => {
                event.stopPropagation();
                onDelete();
              }}
              onKeyDown={event => event.stopPropagation()}
            >
              <Trash2 className="h-3 w-3" />
            </button>
          </TooltipTrigger>
          <TooltipContent>删除任务</TooltipContent>
        </Tooltip>
      </div>
      <div className="mt-1 flex items-center justify-between gap-2">
        <MetaText
          className="inline-flex items-center gap-1 shrink-0"
          tone="weak"
        >
          <CalendarDays className="w-3.5 h-3.5" />
          {formatDueDate(task.dueDate)}
        </MetaText>
        {progress.total > 0 && (
          <span className="inline-flex shrink-0 items-center gap-1 text-[11px] text-[var(--text-weak)]">
            <GitBranch className="h-3 w-3 text-[var(--cp-brand-blue)]" />
            {progress.confirmed}/{progress.total}
          </span>
        )}
      </div>
      {progress.total > 0 && (
        <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-[var(--color-gray-100)]">
          <div
            className="h-full rounded-full bg-[var(--cp-brand-blue)] transition-all"
            style={{ width: `${progress.percent}%` }}
          />
        </div>
      )}
    </TenantCard>
  );
}

function workflowIoTypeLabel(type: string) {
  return (
    {
      text: "文本",
      markdown: "Markdown",
      json: "对象",
      file: "文件",
      url: "链接",
    }[type] ?? type
  );
}

function ResultDialog({
  task,
  nodeId,
  open,
  onOpenChange,
  onConfirm,
}: {
  task: TenantTask;
  nodeId: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: (nodeId: string) => void;
}) {
  const node = task.workflow.find(item => item.id === nodeId);
  const [preview, setPreview] = useState<TenantTaskArtifact | null>(null);
  if (!node) return null;

  const upstreamInputs = Object.entries(node.inputValues ?? {});
  const nodeOutputs = Object.entries(node.outputValues ?? {});
  const nodeTitleById = Object.fromEntries(
    task.workflow.map(workflowNode => [workflowNode.id, workflowNode.title]),
  );
  const displayInputs =
    node.runtimeInputs && node.runtimeInputs.length > 0
      ? node.runtimeInputs
      : node.runtimePrompt
        ? [
            {
              key: "task_goal",
              label: "任务目标",
              type: "text" as const,
              description: node.runtimePrompt,
            },
          ]
        : [];
  const advancedInputKeys = new Set([
    "repository_url",
    "base_dir",
    "constraints",
    "task_slug",
    "run_mode",
    "runtime_mode",
    "reference_artifacts",
    "workflow_state",
  ]);
  const primaryDisplayInputs = displayInputs.filter(
    input => !advancedInputKeys.has(input.key),
  );
  const advancedDisplayInputs = displayInputs.filter(input =>
    advancedInputKeys.has(input.key),
  );

  const renderInputItem = (input: (typeof displayInputs)[number]) => {
    const value = node.inputValues?.[input.key] ??
      (!input.source ? task.taskInputs?.[input.key] : undefined) ??
      (input.key === "task_goal" && node.runtimePrompt
        ? { type: "text" as const, value: node.runtimePrompt }
        : undefined);
    const sourceNode = input.source
      ? nodeTitleById[input.source.nodeId] ?? input.source.nodeId
      : null;
    const isUserInput = input.key === "requirement" || input.key === "input";

    return (
      <div key={input.key} className="px-3 py-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-medium text-[var(--text-body)]">
                {input.label}
              </span>
              <span className="rounded-[4px] bg-[var(--bg-brand-subtle)] px-1.5 py-0.5 text-[11px] text-[var(--cp-brand-blue)]">
                {sourceNode
                  ? "上一步结果"
                  : isUserInput
                    ? "你填写的"
                    : "工作流已准备"}
              </span>
            </div>
            <p className="mb-0 mt-1.5 whitespace-pre-wrap break-words text-sm leading-6 text-[var(--text-secondary)]">
              {value?.value || input.description || "暂未提供"}
            </p>
            <MetaText className="mt-1 block" tone="weak">
              {sourceNode
                ? `来自「${sourceNode}」`
                : isUserInput
                  ? "启动任务时填写"
                  : "由工作流自动提供"}
            </MetaText>
          </div>
          <span
            className={`shrink-0 text-xs ${
              value?.value
                ? "text-[var(--text-success)]"
                : "text-[var(--text-warning)]"
            }`}
          >
            {value?.value ? "已准备" : "缺少内容"}
          </span>
        </div>
      </div>
    );
  };

  return (
    <Dialog
      open={open}
      onOpenChange={next => {
        if (!next) setPreview(null);
        onOpenChange(next);
      }}
    >
      <DialogContent
        size="lg"
        className="max-h-[min(90vh,720px)] grid-rows-[auto_minmax(0,1fr)_auto]"
      >
        <DialogHeader>
          <DialogTitle>
            {preview?.name ?? `${node.title} · 执行结果`}
          </DialogTitle>
          <DialogDescription>
            {preview
              ? `${preview.type} 文件预览`
              : "按“收到什么 → 完成什么 → 交付什么”查看节点执行结果。"}
          </DialogDescription>
        </DialogHeader>
        <DialogBody
          aria-label="节点执行结果滚动区"
          data-scroll-region="task-node-result"
          className="touch-pan-y overscroll-contain px-6 [&::-webkit-scrollbar-thumb]:!bg-gray-300"
        >
          {preview ? (
            <pre className="min-h-[280px] whitespace-pre-wrap break-words rounded-[var(--radius-card)] bg-[var(--color-gray-100)] p-5 text-sm leading-7 text-[var(--text-body)]">
              {preview.content}
            </pre>
          ) : (
            <div className="space-y-5">
              <section>
                <div className="mb-2 flex items-center gap-2">
                  <span className="flex h-5 w-5 items-center justify-center rounded-full bg-[#e8f3ff] text-[11px] font-semibold text-[var(--cp-brand-blue)]">
                    1
                  </span>
                  <BodyMedium>本节点会用到</BodyMedium>
                </div>
                <div className="space-y-2 rounded-[var(--radius-card)] border border-[var(--cp-border)] p-4">
                  {displayInputs.length > 0 && (
                    <div className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--cp-border)]">
                      <div className="flex items-center justify-between gap-3 border-b border-[var(--cp-border)] bg-[var(--bg-subtle)] px-3 py-2">
                        <MetaText>任务要求和资料</MetaText>
                        <MetaText tone="weak">共 {displayInputs.length} 项</MetaText>
                      </div>
                      <div className="divide-y divide-[var(--cp-border)]">
                        {primaryDisplayInputs.map(renderInputItem)}
                        {advancedDisplayInputs.length > 0 && (
                          <details className="group">
                            <summary className="flex cursor-pointer list-none items-center justify-between gap-3 bg-[var(--bg-subtle)] px-3 py-2.5 text-sm text-[var(--text-secondary)] [&::-webkit-details-marker]:hidden">
                              <span>查看高级信息</span>
                              <span className="text-xs text-[var(--text-muted)]">
                                {advancedDisplayInputs.length} 项
                              </span>
                            </summary>
                            <div className="divide-y divide-[var(--cp-border)] border-t border-[var(--cp-border)]">
                              {advancedDisplayInputs.map(renderInputItem)}
                            </div>
                          </details>
                        )}
                      </div>
                    </div>
                  )}
                  {node.runtimePrompt &&
                    (node.configAssets?.length ?? 0) === 0 && (
                    <div>
                      <MetaText tone="weak">任务要求</MetaText>
                      <p className="mb-0 mt-1 whitespace-pre-wrap text-sm leading-6 text-[var(--text-body)]">
                        {node.runtimePrompt}
                      </p>
                    </div>
                    )}
                  {upstreamInputs.length > 0 ? (
                    upstreamInputs.map(([key, input]) => (
                      <div
                        key={key}
                        className="border-t border-[var(--cp-border)] pt-3 first:border-t-0 first:pt-0"
                      >
                        <MetaText tone="weak">
                          来自「{nodeTitleById[input.producedBy ?? key] ?? "上一节点"}」
                        </MetaText>
                        <p className="mb-0 mt-1 whitespace-pre-wrap text-sm leading-6 text-[var(--text-body)]">
                          {input.value}
                        </p>
                      </div>
                    ))
                  ) : !node.runtimePrompt ? (
                    <p className="m-0 text-sm text-[var(--text-weak)]">
                      此节点直接承接任务目标，无额外上游输入。
                    </p>
                  ) : null}
                </div>
              </section>

              <section>
                <div className="mb-2 flex items-center gap-2">
                  <span className="flex h-5 w-5 items-center justify-center rounded-full bg-[#e8f8f0] text-[11px] font-semibold text-[var(--text-success)]">
                    2
                  </span>
                  <BodyMedium>本节点完成的结果</BodyMedium>
                </div>
                <div className="rounded-[var(--radius-card)] bg-[var(--color-gray-100)] p-4">
                  <MetaText tone="weak">执行结论</MetaText>
                  <p className="mb-0 mt-1 whitespace-pre-wrap text-sm leading-6 text-[var(--text-body)]">
                    {node.result || "节点尚未返回执行结论。"}
                  </p>
                </div>
              </section>

              {((node.runtimeOutputs?.length ?? 0) > 0 || nodeOutputs.length > 0) && (
                <section>
                  <div className="mb-2 flex items-center gap-2">
                    <span className="flex h-5 w-5 items-center justify-center rounded-full bg-[#f2edff] text-[11px] font-semibold text-[#6d44cc]">
                      3
                    </span>
                    <BodyMedium>交给下一节点的内容</BodyMedium>
                  </div>
                  <div className="space-y-2 rounded-[var(--radius-card)] border border-[var(--cp-border)] p-4">
                    {(node.runtimeOutputs?.length
                      ? node.runtimeOutputs.map(output => [output.key, node.outputValues?.[output.key]] as const)
                      : nodeOutputs
                    ).map(([key, output]) => {
                      const definition = node.runtimeOutputs?.find(item => item.key === key);
                      return (
                        <div key={key} className="flex items-start gap-3 border-b border-[var(--cp-border)] pb-2 last:border-b-0 last:pb-0">
                          <div className="min-w-0 flex-1">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="text-sm font-medium text-[var(--text-body)]">
                                {definition?.label ?? "节点结论"}
                              </span>
                              {definition && (
                                <MetaText tone="weak">{workflowIoTypeLabel(definition.type)}</MetaText>
                              )}
                              {definition?.required && (
                                <span className="text-[11px] text-[var(--text-danger)]">必填</span>
                              )}
                            </div>
                            <p className="mb-0 mt-1 whitespace-pre-wrap text-sm leading-6 text-[var(--text-body)]">
                              {output?.value || definition?.description || "等待 Agent 返回该字段"}
                            </p>
                          </div>
                          <span className={`shrink-0 text-xs ${output?.value ? "text-[var(--text-success)]" : "text-[var(--text-warning)]"}`}>
                            {output?.value ? "已返回" : "待返回"}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </section>
              )}

              <section>
                <div className="mb-2 flex items-center gap-2">
                  <span className="flex h-5 w-5 items-center justify-center rounded-full bg-[#fff1e8] text-[11px] font-semibold text-[#b55b13]">
                    {nodeOutputs.length > 0 ? 4 : 3}
                  </span>
                  <BodyMedium>交付产物</BodyMedium>
                </div>
                {node.artifacts.length > 0 ? (
                  <div className="space-y-2">
                    {node.artifacts.map(artifact => (
                      <div
                        key={artifact.id}
                        className="flex items-center gap-3 rounded-[var(--radius-card)] border border-[var(--cp-border)] px-3 py-3"
                      >
                        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[4px] bg-[var(--bg-brand-subtle)] text-[var(--cp-brand-blue)]">
                          <FileText className="h-4 w-4" />
                        </span>
                        <button
                          type="button"
                          className="min-w-0 flex-1 text-left text-sm text-[var(--text-body)] hover:text-[var(--cp-brand-blue)]"
                          onClick={() => setPreview(artifact)}
                        >
                          <span className="block truncate font-medium">{artifact.name}</span>
                          <MetaText tone="weak">来源：{node.title} · {artifact.type} 文件</MetaText>
                        </button>
                        <span className="shrink-0 text-xs text-[var(--text-success)]">已回传</span>
                        <Button
                          variant="tenant-outline"
                          size="sm"
                          onClick={() => setPreview(artifact)}
                        >
                          <Eye className="h-3.5 w-3.5" />
                          查看
                        </Button>
                        <Button
                          variant="tenant-outline"
                          size="sm"
                          onClick={() => downloadArtifact(artifact)}
                        >
                          <Download className="h-3.5 w-3.5" />
                          下载
                        </Button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="rounded-[var(--radius-card)] border border-dashed border-[var(--cp-border)] px-4 py-5 text-center text-sm text-[var(--text-weak)]">
                    本节点没有生成独立文件，执行结论会直接交给下一节点。
                  </div>
                )}
              </section>
            </div>
          )}
        </DialogBody>
        <DialogFooter>
          {preview && (
            <Button variant="tenant-outline" onClick={() => setPreview(null)}>
              返回结果
            </Button>
          )}
          {!preview && node.status === "review" && (
            <Button
              variant="tenant-primary"
              onClick={() => {
                onConfirm(node.id);
                onOpenChange(false);
              }}
            >
              <Check className="h-4 w-4" />
              确认该节点结果
            </Button>
          )}
          <Button
            variant={
              !preview && node.status === "review"
                ? "tenant-outline"
                : "tenant-primary"
            }
            onClick={() => onOpenChange(false)}
          >
            关闭
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

const WORKFLOW_NODE_WIDTH = 288;
const WORKFLOW_NODE_GAP = 72;
const WORKFLOW_CANVAS_PADDING = 20;
const WORKFLOW_NODE_TOP = 42;
const WORKFLOW_NODE_HEIGHT = 226;
const WORKFLOW_ROW_GAP = 36;

const WORKFLOW_NODE_STATUS = {
  pending: { label: "待执行", variant: "gray" as const },
  running: { label: "执行中", variant: "blue" as const },
  stopped: { label: "已停止", variant: "orange" as const },
  review: { label: "待确认", variant: "orange" as const },
  confirmed: { label: "已确认", variant: "green" as const },
};

const TASK_WORKBENCH_STATUS: Record<
  TenantTaskStatus,
  { label: string; variant: "gray" | "blue" | "orange" | "green" }
> = {
  todo: { label: "待启动", variant: "gray" },
  in_progress: { label: "进行中", variant: "blue" },
  review: { label: "待确认", variant: "orange" },
  done: { label: "已完成", variant: "green" },
  hold: { label: "挂起", variant: "orange" },
};

const AGENT_RUNTIME_LABEL = {
  clawpro: "Hy3",
  imate: "iMate Cloud",
  codex: "GPT-5.6-Sol",
  codebuddy: "CodeBuddy",
  workbuddy: "WorkBuddy",
  cloudagent: "DevResonance CloudAgent",
} as const;

function buildWorkflowLayout(workflow: TenantWorkflowNode[]) {
  const stageByNodeId = new Map<string, number>();
  const stages: TenantWorkflowNode[][] = [];

  workflow.forEach(node => {
    const stage =
      node.dependsOn.length === 0
        ? 0
        : Math.max(
            ...node.dependsOn.map(
              dependency => stageByNodeId.get(dependency) ?? 0
            )
          ) + 1;
    stageByNodeId.set(node.id, stage);
    (stages[stage] ??= []).push(node);
  });
  stages.forEach(stage => {
    // 完整研发链路是主路径，SOLO 小需求快捷分支固定放在下方。
    const isSolo = (node: TenantWorkflowNode) =>
      node.runtimePhaseId === "SOLO" ||
      node.id === "SOLO" ||
      node.title.startsWith("SOLO");
    stage.sort((a, b) => Number(isSolo(a)) - Number(isSolo(b)));
  });

  const rowByNodeId = new Map<string, number>();
  const nodes = stages.flatMap((stageNodes, stageIndex) => {
    const occupiedRows = new Set<number>();
    return stageNodes.map((node, rowIndex) => {
      const dependencyRows = node.dependsOn
        .map(dependency => rowByNodeId.get(dependency))
        .filter((row): row is number => row !== undefined);
      // 同层多节点形成不同泳道；后续节点继承上游泳道，避免分支中途折回首行。
      let lane = stageNodes.length > 1
        ? rowIndex
        : dependencyRows.length > 0
          ? dependencyRows[dependencyRows.length - 1]
          : 0;
      while (occupiedRows.has(lane)) lane += 1;
      occupiedRows.add(lane);
      rowByNodeId.set(node.id, lane);
      return {
        node,
        stageIndex,
        rowIndex: lane,
        x:
          WORKFLOW_CANVAS_PADDING +
          stageIndex * (WORKFLOW_NODE_WIDTH + WORKFLOW_NODE_GAP),
        y:
          WORKFLOW_NODE_TOP +
          lane * (WORKFLOW_NODE_HEIGHT + WORKFLOW_ROW_GAP),
      };
    });
  });
  const nodeById = new Map(nodes.map(item => [item.node.id, item]));
  const edges = nodes.flatMap(target =>
    target.node.dependsOn.flatMap(dependency => {
      const source = nodeById.get(dependency);
      return source ? [{ source, target }] : [];
    })
  );
  const maxRows = Math.max(1, ...nodes.map(item => item.rowIndex + 1));

  return {
    stages,
    nodes,
    edges,
    canvasWidth:
      WORKFLOW_CANVAS_PADDING * 2 +
      stages.length * WORKFLOW_NODE_WIDTH +
      Math.max(0, stages.length - 1) * WORKFLOW_NODE_GAP,
    canvasHeight:
      WORKFLOW_NODE_TOP +
      maxRows * WORKFLOW_NODE_HEIGHT +
      Math.max(0, maxRows - 1) * WORKFLOW_ROW_GAP +
      12,
  };
}

function supportsWorkflowNode(
  agent: TenantAgent,
  node: TenantWorkflowNode
): boolean {
  if (agent.status !== "online") return false;
  const required = node.runtimeRequiredCapabilities ?? [];
  const available = new Set(agent.runtimeCapabilities ?? []);
  return required.every(capability => available.has(capability));
}

function TaskWorkflowPage({
  project,
  task,
  onBack,
  onRuntimeStarted,
}: {
  project: TenantProject;
  task: TenantTask;
  onBack: () => void;
  onRuntimeStarted?: (backendTaskId: string) => void;
}) {
  const [resultNodeId, setResultNodeId] = useState<string | null>(null);
  const [configNodeId, setConfigNodeId] = useState<string | null>(null);
  const [solidifyOpen, setSolidifyOpen] = useState(false);
  const [solidifyName, setSolidifyName] = useState("");
  const [startOpen, setStartOpen] = useState(false);
  const [stopRequestNonce, setStopRequestNonce] = useState(0);
  const [liveRuntimeIsActive, setLiveRuntimeIsActive] = useState<
    boolean | null
  >(null);
  const [detailTab, setDetailTab] = useState<"workflow" | "artifacts" | "info">(
    "workflow"
  );
  const [liveWorkflowAgents, setLiveWorkflowAgents] = useState<TenantAgent[]>(
    []
  );
  const [liveAgentsLoading, setLiveAgentsLoading] = useState(false);
  const [liveAgentsError, setLiveAgentsError] = useState("");

  const owner = project.members.find(member => member.userId === task.ownerId);
  const isExecutableWorkflow = isTenantExecutableWorkflowTask(task);
  const runtimeIsActive =
    liveRuntimeIsActive ??
    Boolean(
      task.runtimeExecution &&
        (task.runtimeExecution.canStop ??
          (!task.runtimeExecution.cancelRequested &&
            (!["completed", "failed", "canceled"].includes(
              task.runtimeExecution.status
            ) ||
              task.runtimeExecution.phases.some(phase =>
                ["running", "awaiting_approval"].includes(phase.status)
              ))))
    );
  const availableWorkflowAgents: TenantAgent[] = isExecutableWorkflow
    ? liveWorkflowAgents
    : project.agents;

  useEffect(() => {
    if (!isExecutableWorkflow) return;
    let active = true;
    setLiveAgentsLoading(true);
    setLiveAgentsError("");
    listWorkflowRuntimeAgents()
      .then(items => {
        if (!active) return;
        const fallbackOwner = project.members[0];
        const resolved: TenantAgent[] = items.map(agent => ({
          id: agent.id,
          name: agent.name,
          platform: agent.platform,
          location: agent.location,
          ownerId: fallbackOwner?.userId ?? "runtime",
          owner: fallbackOwner?.displayName ?? "真实 Runtime",
          role:
            agent.platform === "codebuddy"
              ? "开发"
              : agent.platform === "cloudagent"
                ? "业务自动化"
                : "评审 / 测试",
          status: agent.status,
          authorization:
            agent.platform === "codebuddy"
              ? "TeamAI 已信任设备"
              : agent.platform === "cloudagent"
                ? "DevResonance 服务端受控授权"
                : "iMate 项目已授权 OpenClaw Agent",
          kind: "project",
          runtimeId: agent.runtimeId,
          deviceId: agent.deviceId,
          targetAgentId: agent.targetAgentId,
          runtimeDetail: agent.detail,
          runtimeCapabilities: agent.capabilities,
          runtimeMissingCapabilities: agent.missingCapabilities,
        }));
        setLiveWorkflowAgents(resolved);

        task.workflow.forEach(node => {
          const currentRuntimeAgent = resolved.find(
            agent => agent.id === node.agentId
          );
          // 已明确绑定的 Runtime 即使暂时离线也应保留，供用户看到真实
          // 授权/连通状态；不能静默改派给另一个平台的 Agent。
          if (currentRuntimeAgent) return;
          const eligibleAgents = resolved.filter(agent =>
            supportsWorkflowNode(agent, node)
          );
          const previous = project.agents.find(
            agent => agent.id === node.agentId
          );
          const platform = previous?.platform;
          const replacement =
            eligibleAgents.find(agent => agent.platform === platform) ??
            eligibleAgents[0];
          if (replacement) {
            tenantProjectStore.assignWorkflowAgent(
              project.id,
              task.id,
              node.id,
              replacement.id
            );
          }
        });
      })
      .catch(error => {
        if (active) setLiveAgentsError((error as Error).message);
      })
      .finally(() => {
        if (active) setLiveAgentsLoading(false);
      });
    return () => {
      active = false;
    };
  }, [
    isExecutableWorkflow,
    project.agents,
    project.id,
    project.members,
    task.id,
    task.workflow,
  ]);
  const workflowLayout = useMemo(
    () => buildWorkflowLayout(task.workflow),
    [task.workflow]
  );
  const arrowId = `workflow-arrow-${task.id}`;
  const taskStatus = TASK_WORKBENCH_STATUS[task.status];
  const configNode = task.workflow.find(node => node.id === configNodeId);

  useEffect(() => {
    setLiveRuntimeIsActive(null);
  }, [task.runtimeExecution?.backendTaskId]);

  return (
    <>
      <div className="flex min-h-full flex-col bg-[var(--bg-brand-subtle)]">
        <div className="border-b border-[var(--cp-border)] bg-white px-7 pb-4 pt-5">
          <div className="flex items-center justify-between gap-5">
            <div className="flex min-w-0 items-center gap-3">
              <button
                type="button"
                onClick={onBack}
                className="inline-flex shrink-0 items-center gap-1.5 text-sm font-medium text-[var(--text-title)] hover:text-[var(--cp-brand-blue)]"
              >
                <ArrowLeft className="h-4 w-4" />
                返回看板
              </button>
              <span className="text-[var(--text-weak)]">/</span>{" "}
              <h2 className="m-0 max-w-[720px] truncate text-lg font-semibold text-[var(--text-title)]">
                {task.title}
              </h2>
              {task.pipelineTemplateId && (
                <StatusTag mode="soft" variant="blue">
                  <Workflow className="w-3 h-3 mr-1 inline" />
                  来自工作流：
                  {project.pipelineTemplates.find(
                    t => t.id === task.pipelineTemplateId
                  )?.name ?? "已删除的工作流"}
                </StatusTag>
              )}
            </div>
            <div className="flex items-center gap-2 pr-8">
              {/* 状态下拉：任何时候都可切换 */}
              <Select
                value={task.status}
                onValueChange={v => {
                  tenantProjectStore.updateTaskStatus(
                    project.id,
                    task.id,
                    v as TenantTaskStatus
                  );
                  toast.success(
                    `任务状态已更新为「${TASK_STATUS_LABEL[v as TenantTaskStatus]}」`
                  );
                }}
              >
                <SelectTrigger className="h-8 w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="todo">待启动</SelectItem>
                  <SelectItem value="in_progress">进行中</SelectItem>
                  <SelectItem value="done">已完成</SelectItem>
                  <SelectItem value="hold">挂起</SelectItem>
                </SelectContent>
              </Select>
              {isExecutableWorkflow &&
                ((task.status === "todo" && !task.runtimeExecution) ||
                  task.runtimeExecution) && (
                <Button
                  variant={runtimeIsActive ? "tenant-outline" : "tenant-primary"}
                  size="sm"
                  className={`h-8 px-3 ${
                    runtimeIsActive
                      ? "border-[var(--text-danger)] text-[var(--text-danger)] hover:bg-[var(--bg-danger-subtle)]"
                      : ""
                  }`}
                  onClick={() => {
                    if (runtimeIsActive) {
                      setStopRequestNonce(value => value + 1);
                      return;
                    }
                    if (task.runtimeExecution) {
                      tenantProjectStore.prepareRuntimeRerun(
                        project.id,
                        task.id
                      );
                      toast.success(
                        "已返回节点配置，请选择真实 Agent 后再启动"
                      );
                      return;
                    }
                    setStartOpen(true);
                  }}
                >
                  {runtimeIsActive ? (
                    <Square className="h-3.5 w-3.5" />
                  ) : (
                    <Play className="h-4 w-4" />
                  )}
                  {runtimeIsActive
                    ? "停止"
                    : task.runtimeExecution
                      ? "重新运行"
                      : "启动"}
                </Button>
              )}
              {task.status === "done" && task.workflow.length > 0 && (
                <Button
                  variant="tenant-outline"
                  size="sm"
                  className="h-8 px-3"
                  onClick={() => {
                    setSolidifyName(`${task.title} 流水线`);
                    setSolidifyOpen(true);
                  }}
                >
                  <Workflow className="h-4 w-4" />
                  固化为模板
                </Button>
              )}
            </div>
          </div>
        </div>

        {/* 任务详情 Tab：工作流 / 产物 / 信息 */}
        <div className="border-b border-[var(--cp-border)] bg-white px-7">
          <div className="flex items-center gap-1">
            {(
              [
                { id: "workflow", label: "工作流" },
                { id: "artifacts", label: "产物" },
                { id: "info", label: "信息" },
              ] as const
            ).map(t => (
              <button
                key={t.id}
                type="button"
                onClick={() => setDetailTab(t.id)}
                className={`relative px-3 py-2.5 text-sm transition-colors ${
                  detailTab === t.id
                    ? "font-medium text-[var(--cp-brand-blue)] after:absolute after:inset-x-2 after:-bottom-px after:h-0.5 after:bg-[var(--cp-brand-blue)]"
                    : "text-[var(--text-secondary)] hover:text-[var(--text-body)]"
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>

        {detailTab === "workflow" && (
          <div className="flex-1 overflow-auto">
            {task.runtimeExecution ? (
              <LiveWorkflowRun
                projectId={project.id}
                task={task}
                agents={availableWorkflowAgents}
                members={project.members}
                stopRequestNonce={stopRequestNonce}
                onRuntimeExpired={onBack}
                onRuntimeActiveChange={setLiveRuntimeIsActive}
              />
            ) : (
              <div
                className="relative mx-auto"
                style={{
                  width: Math.max(workflowLayout.canvasWidth, 920),
                  height: workflowLayout.canvasHeight,
                }}
              >
                <svg
                  aria-hidden
                  className="pointer-events-none absolute inset-0 h-full w-full"
                  viewBox={`0 0 ${Math.max(workflowLayout.canvasWidth, 920)} ${
                    workflowLayout.canvasHeight
                  }`}
                >
                  <defs>
                    <marker
                      id={arrowId}
                      markerWidth="8"
                      markerHeight="8"
                      refX="7"
                      refY="4"
                      orient="auto"
                    >
                      <path d="M 0 0 L 8 4 L 0 8 z" fill="var(--text-weak)" />
                    </marker>
                  </defs>
                  {workflowLayout.edges.map(({ source, target }) => {
                    const sourceX = source.x + WORKFLOW_NODE_WIDTH;
                    const sourceY = source.y + WORKFLOW_NODE_HEIGHT / 2;
                    const targetX = target.x;
                    const targetY = target.y + WORKFLOW_NODE_HEIGHT / 2;
                    const curveX = (sourceX + targetX) / 2;
                    return (
                      <path
                        key={`${source.node.id}-${target.node.id}`}
                        d={`M ${sourceX} ${sourceY} C ${curveX} ${sourceY}, ${curveX} ${targetY}, ${
                          targetX - 6
                        } ${targetY}`}
                        fill="none"
                        stroke="var(--text-weak)"
                        strokeWidth="1.5"
                        markerEnd={`url(#${arrowId})`}
                      />
                    );
                  })}
                </svg>

                {workflowLayout.stages.map((stage, stageIndex) => {
                  const stageDone = stage.every(
                    node => node.status === "confirmed"
                  );
                  const stageActive = stage.some(
                    node =>
                      node.status === "running" ||
                      node.status === "stopped" ||
                      node.status === "review"
                  );
                  return (
                    <div
                      key={`stage-${stageIndex}`}
                      className="absolute top-3 flex items-center gap-2"
                      style={{
                        left:
                          WORKFLOW_CANVAS_PADDING +
                          stageIndex *
                            (WORKFLOW_NODE_WIDTH + WORKFLOW_NODE_GAP),
                        width: WORKFLOW_NODE_WIDTH,
                      }}
                    >
                      <span
                        className={`inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-full border ${
                          stageDone || stageActive
                            ? "border-[var(--cp-brand-blue)] text-[var(--cp-brand-blue)]"
                            : "border-[var(--border-strong)] text-[var(--text-weak)]"
                        }`}
                      >
                        {stageDone ? (
                          <Check className="h-3 w-3" />
                        ) : (
                          <span className="h-1.5 w-1.5 rounded-full bg-current" />
                        )}
                      </span>
                      <BodyMedium className="truncate">
                        {stage.length > 1
                          ? stage.map(node => node.title).join(" / ")
                          : stage[0]?.title}
                      </BodyMedium>
                    </div>
                  );
                })}

                {workflowLayout.nodes.map(({ node, x, y }) => {
                  const selectedAgent = availableWorkflowAgents.find(
                    agent => agent.id === node.agentId
                  );
                  const selectedAgentReady = selectedAgent
                    ? supportsWorkflowNode(selectedAgent, node)
                    : false;
                  const member = selectedAgent
                    ? project.members.find(
                        item => item.userId === selectedAgent.ownerId
                      )
                    : undefined;
                  const locked = node.status !== "pending";
                  const status = WORKFLOW_NODE_STATUS[node.status];

                  return (
                    <TenantCard
                      key={node.id}
                      padding="none"
                      className={`absolute overflow-hidden bg-white transition-[border-color,box-shadow] ${
                        node.status === "running"
                          ? "border-[var(--cp-brand-blue)] [box-shadow:0_0_0_3px_var(--bg-brand-selected-solid),var(--shadow-card)]"
                          : "border-white shadow-[var(--shadow-card)]"
                      }`}
                      style={{
                        left: x,
                        top: y,
                        width: WORKFLOW_NODE_WIDTH,
                        height: WORKFLOW_NODE_HEIGHT,
                      }}
                    >
                      <div className="flex h-full flex-col p-4">
                        <div className="flex items-center gap-2.5">
                          <CircleCheck
                            className={`h-4 w-4 shrink-0 ${
                              node.status === "confirmed"
                                ? "text-[var(--cp-brand-blue)]"
                                : "text-[var(--text-muted)]"
                            }`}
                          />
                          <h3 className="m-0 min-w-0 flex-1 truncate text-sm font-semibold text-[var(--text-title)]">
                            {node.title}
                          </h3>
                          {(node.configAssets?.length ?? 0) > 0 && (
                            <button
                              type="button"
                              className="inline-flex h-6 shrink-0 items-center gap-1 rounded-[4px] bg-[var(--bg-brand-subtle)] px-1.5 text-[11px] font-medium text-[var(--cp-brand-blue)] hover:bg-[var(--bg-brand-selected-solid)]"
                              aria-label={`查看「${node.title}」的配置资产`}
                              onClick={() => setConfigNodeId(node.id)}
                            >
                              <Settings2 className="h-3 w-3" />
                              资产 {node.configAssets?.length}
                            </button>
                          )}
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[var(--bg-brand-selected-solid)] text-xs font-semibold text-[var(--cp-brand-blue)]">
                                {(
                                  member?.displayName ??
                                  owner?.displayName ??
                                  "负"
                                ).slice(0, 1)}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent>
                              负责人：
                              {member?.displayName ??
                                owner?.displayName ??
                                task.ownerId}
                            </TooltipContent>
                          </Tooltip>
                        </div>

                        <div className="mt-3 rounded-[4px] border border-[var(--cp-border)] px-2.5 py-2">
                          <div className="flex items-center justify-between gap-2">
                            <MetaText tone="weak">执行 Agent</MetaText>
                            <StatusTag mode="soft" variant={status.variant}>
                              {status.label}
                            </StatusTag>
                          </div>
                          <Select
                            value={node.agentId ?? undefined}
                            disabled={
                              locked ||
                              liveAgentsLoading ||
                              availableWorkflowAgents.length === 0
                            }
                            onValueChange={agentId =>
                              tenantProjectStore.assignWorkflowAgent(
                                project.id,
                                task.id,
                                node.id,
                                agentId
                              )
                            }
                          >
                            <SelectTrigger className="mt-1.5 h-8 w-full bg-white text-xs">
                              <SelectValue placeholder="选择云端或本地 Agent" />
                            </SelectTrigger>
                            <SelectContent>
                              {availableWorkflowAgents.map(agent => {
                                const agentMember = project.members.find(
                                  item => item.userId === agent.ownerId
                                );
                                const agentReady = supportsWorkflowNode(
                                  agent,
                                  node
                                );
                                const missingCapabilities = (
                                  node.runtimeRequiredCapabilities ?? []
                                ).filter(
                                  capability =>
                                    !(agent.runtimeCapabilities ?? []).includes(
                                      capability
                                    )
                                );
                                return (
                                  <SelectItem
                                    key={agent.id}
                                    value={agent.id}
                                    disabled={!agentReady}
                                  >
                                    {agent.location === "local"
                                      ? "本地"
                                      : "云端"}{" "}
                                    · {agent.name} ·{" "}
                                    {agentReady
                                      ? (agent.runtimeDetail ??
                                        agentMember?.role ??
                                        agent.owner)
                                      : `待授权：${missingCapabilities.join(", ")}`}
                                  </SelectItem>
                                );
                              })}
                            </SelectContent>
                          </Select>
                          {selectedAgent && selectedAgentReady ? (
                            <div className="mt-1.5 flex items-center gap-1.5">
                              <span className="inline-flex items-center gap-1 text-xs text-[var(--cp-brand-blue)]">
                                <GitBranch className="h-3 w-3" />
                                运行信息
                              </span>
                              <span className="truncate text-xs text-[var(--text-muted)]">
                                {AGENT_RUNTIME_LABEL[selectedAgent.platform]} ·{" "}
                                {selectedAgent.location === "local"
                                  ? "本地"
                                  : "云端"}
                                {selectedAgent.runtimeDetail
                                  ? ` · ${selectedAgent.runtimeDetail}`
                                  : ""}
                              </span>
                            </div>
                          ) : selectedAgent ? (
                            <MetaText className="mt-1 block" tone="weak">
                              {selectedAgent.runtimeDetail ??
                                "当前 Agent 缺少节点所需授权"}
                            </MetaText>
                          ) : (
                            <MetaText className="mt-1 block" tone="weak">
                              {liveAgentsLoading
                                ? "正在读取真实 Agent"
                                : liveAgentsError || "未发现可用的真实 Agent"}
                            </MetaText>
                          )}
                        </div>

                        <div className="mt-auto border-t border-[var(--cp-border)] pt-2">
                          {node.status === "running" && (
                            <div className="flex items-center justify-end">
                              <button
                                type="button"
                                className="inline-flex h-6 items-center gap-1 rounded-[4px] px-1.5 text-xs text-[var(--text-muted)] hover:bg-[var(--color-gray-100)] hover:text-[var(--text-title)]"
                                onClick={() => {
                                  tenantProjectStore.stopWorkflowNode(
                                    project.id,
                                    task.id,
                                    node.id
                                  );
                                  toast.success("节点已停止，可随时重试");
                                }}
                              >
                                <Square className="h-3 w-3" />
                                停止
                              </button>
                            </div>
                          )}

                          {node.status === "stopped" && (
                            <div className="flex justify-end">
                              <button
                                type="button"
                                className="inline-flex h-6 items-center gap-1 rounded-[4px] px-1.5 text-xs font-medium text-[var(--cp-brand-blue)] hover:bg-[var(--bg-brand-selected-solid)]"
                                onClick={() => {
                                  tenantProjectStore.retryWorkflowNode(
                                    project.id,
                                    task.id,
                                    node.id
                                  );
                                  toast.success("节点已重新开始执行");
                                }}
                              >
                                <RotateCcw className="h-3 w-3" />
                                重新执行
                              </button>
                            </div>
                          )}

                          {(node.status === "review" ||
                            node.status === "confirmed") && (
                            <div>
                              <div className="flex items-center gap-2">
                                <FileText className="h-3.5 w-3.5 shrink-0 text-[var(--text-secondary)]" />
                                <button
                                  type="button"
                                  className="min-w-0 flex-1 truncate text-left text-xs font-medium text-[var(--text-body)] hover:text-[var(--cp-brand-blue)]"
                                  onClick={() => setResultNodeId(node.id)}
                                >
                                  {node.title} · 执行结果
                                </button>
                                <button
                                  type="button"
                                  aria-label={`查看「${node.title}」执行结果`}
                                  className="text-[var(--text-secondary)] hover:text-[var(--cp-brand-blue)]"
                                  onClick={() => setResultNodeId(node.id)}
                                >
                                  <Eye className="h-3.5 w-3.5" />
                                </button>
                                <button
                                  type="button"
                                  className="inline-flex items-center gap-1 text-xs text-[var(--text-muted)] hover:text-[var(--cp-brand-blue)]"
                                  onClick={() => setResultNodeId(node.id)}
                                >
                                  <Paperclip className="h-3 w-3" />
                                  {node.artifacts.length}
                                </button>
                                {node.status === "review" && (
                                  <button
                                    type="button"
                                    className="inline-flex h-6 shrink-0 items-center gap-1 rounded-[4px] px-1.5 text-xs font-medium text-[var(--cp-brand-blue)] hover:bg-[var(--bg-brand-selected-solid)]"
                                    onClick={() => {
                                      tenantProjectStore.confirmWorkflowNode(
                                        project.id,
                                        task.id,
                                        node.id
                                      );
                                      toast.success("该节点结果已确认");
                                    }}
                                  >
                                    <Check className="h-3 w-3" />
                                    确认
                                  </button>
                                )}
                              </div>
                            </div>
                          )}
                          {/* 数据流：本节点跑完后，产出会传给哪些下游节点 */}
                          {node.status === "confirmed" &&
                            (() => {
                              const downstream = task.workflow.filter(n =>
                                n.dependsOn.includes(node.id)
                              );
                              if (downstream.length === 0) return null;
                              return (
                                <div className="mt-1.5 flex items-center gap-1 text-[11px] text-[#22c55e]">
                                  <span>→</span>
                                  <span className="truncate">
                                    传给：
                                    {downstream.map(n => n.title).join("、")}
                                  </span>
                                </div>
                              );
                            })()}
                        </div>
                      </div>
                    </TenantCard>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {detailTab === "artifacts" && (
          <div className="flex-1 overflow-auto p-6">
            <TaskArtifactsView task={task} />
          </div>
        )}

        {detailTab === "info" && (
          <div className="flex-1 overflow-auto p-6">
            <TaskInfoView
              task={task}
              project={project}
              owner={owner}
              taskStatus={taskStatus}
            />
          </div>
        )}
      </div>
      <ResultDialog
        task={task}
        nodeId={resultNodeId}
        open={!!resultNodeId}
        onOpenChange={open => !open && setResultNodeId(null)}
        onConfirm={nodeId => {
          tenantProjectStore.confirmWorkflowNode(project.id, task.id, nodeId);
          toast.success("该节点结果已确认");
        }}
      />
      <Sheet
        open={Boolean(configNode)}
        onOpenChange={open => {
          if (!open) setConfigNodeId(null);
        }}
      >
        <SheetContent
          side="right"
          className="flex h-[100dvh] max-h-[100dvh] w-[min(620px,94vw)] flex-col gap-0 overflow-hidden p-0 sm:max-w-[620px]"
        >
          {configNode && (
            <>
              <SheetHeader className="border-b border-[var(--cp-border)] px-6 py-4 pr-12">
                <SheetTitle>{configNode.title} · 配置资产</SheetTitle>
                <MetaText className="mt-1 block" tone="weak">
                  展开查看节点规则；启动任务时，同一份内容会注入 Agent Prompt。
                </MetaText>
              </SheetHeader>
              <div className="min-h-0 flex-1 overflow-y-auto bg-[var(--bg-subtle)] px-6 py-5">
                <NodeConfigAssets assets={configNode.configAssets ?? []} />
              </div>
            </>
          )}
        </SheetContent>
      </Sheet>
      <LiveWorkflowStartDialog
        open={startOpen}
        onOpenChange={setStartOpen}
        onStarted={onRuntimeStarted}
        projectId={project.id}
        task={task}
        agents={availableWorkflowAgents}
      />
      <Dialog open={solidifyOpen} onOpenChange={setSolidifyOpen}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>固化为流水线模板</DialogTitle>
            <DialogDescription>
              把本 issue
              跑顺的执行流程沉淀为可复用的流水线模板，供后续同类工作一键实例化。
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="space-y-1.5">
              <Label>模板名称</Label>
              <Input
                value={solidifyName}
                onChange={e => setSolidifyName(e.target.value)}
                placeholder="例如：工单处理流水线"
              />
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="tenant-outline"
              onClick={() => setSolidifyOpen(false)}
            >
              取消
            </Button>
            <Button
              variant="tenant-primary"
              disabled={!solidifyName.trim()}
              onClick={() => {
                const id = tenantProjectStore.solidifyIssueAsTemplate(
                  project.id,
                  task.id,
                  solidifyName.trim()
                );
                setSolidifyOpen(false);
                if (id) toast.success("已固化为流水线模板");
                else toast.error("固化失败，该任务无可用工作流");
              }}
            >
              固化
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

// ─── 任务详情：产物 / 信息 Tab 的内容视图 ─────────────
/** 生成一组 mock 产物（当任务尚无真实产物时用于演示，让产物 Tab 不为空） */
function buildMockArtifacts(
  task: TenantTask
): {
  id: string;
  name: string;
  type: "Markdown" | "JSON";
  nodeTitle: string;
  nodeId: string;
}[] {
  const nodeTitles =
    task.workflow.length > 0
      ? task.workflow.map(n => n.title)
      : ["需求分析", "方案设计", "开发实现", "联调测试"];
  const presets: { suffix: string; type: "Markdown" | "JSON" }[] = [
    { suffix: "产出说明.md", type: "Markdown" },
    { suffix: "结构化结果.json", type: "JSON" },
    { suffix: "交接纪要.md", type: "Markdown" },
  ];
  const items: {
    id: string;
    name: string;
    type: "Markdown" | "JSON";
    nodeTitle: string;
    nodeId: string;
  }[] = [];
  nodeTitles.forEach((title, i) => {
    // 每个节点挑1 个产物，控制总量在 4~6 条之间
    const preset = presets[i % presets.length];
    items.push({
      id: `mock-${task.id}-${i}`,
      name: `${title}·${preset.suffix}`,
      type: preset.type,
      nodeTitle: title,
      nodeId: `mock-node-${i}`,
    });
  });
  return items.slice(0, 6);
}

function TaskArtifactsView({ task }: { task: TenantTask }) {
  const runtimeBackendTaskId = task.runtimeExecution?.backendTaskId ?? "";
  const [runtimeItems, setRuntimeItems] = useState<
    TenantRuntimeExecutionArtifact[]
  >(() => task.runtimeExecution?.artifacts ?? []);
  const [runtimeItemsLoading, setRuntimeItemsLoading] = useState(false);
  const [runtimeItemsError, setRuntimeItemsError] = useState("");
  const [runtimePreview, setRuntimePreview] =
    useState<TenantRuntimeExecutionArtifact | null>(null);
  const [runtimeContent, setRuntimeContent] = useState("");
  const [runtimeContentError, setRuntimeContentError] = useState("");
  const [runtimeContentLoading, setRuntimeContentLoading] = useState(false);
  const realItems = task.workflow.flatMap(n =>
    n.artifacts.map(a => ({ ...a, nodeTitle: n.title, nodeId: n.id }))
  );
  const isRuntime = Boolean(task.runtimeExecution);
  const isMock = !isRuntime && realItems.length === 0;
  const items = isMock ? buildMockArtifacts(task) : realItems;

  useEffect(() => {
    if (!runtimeBackendTaskId) {
      setRuntimeItems([]);
      setRuntimeItemsError("");
      setRuntimeItemsLoading(false);
      return;
    }
    let active = true;
    setRuntimeItems(task.runtimeExecution?.artifacts ?? []);
    setRuntimeItemsError("");
    setRuntimeItemsLoading(true);
    getWorkflowTask(runtimeBackendTaskId)
      .then(({ task: runtimeTask }) => {
        if (!active) return;
        setRuntimeItems(
          (runtimeTask.available_artifacts ?? []).map((artifact, index) => ({
            id:
              artifact.artifact_id ??
              `${runtimeBackendTaskId}-artifact-${index}`,
            path: artifact.path,
            mediaType: artifact.media_type,
            size: artifact.size,
            sha256: artifact.sha256,
          })),
        );
      })
      .catch(error => {
        if (!active) return;
        setRuntimeItemsError((error as Error).message);
      })
      .finally(() => {
        if (active) setRuntimeItemsLoading(false);
      });
    return () => {
      active = false;
    };
  }, [runtimeBackendTaskId]);

  useEffect(() => {
    if (!runtimePreview || !task.runtimeExecution) {
      setRuntimeContent("");
      setRuntimeContentError("");
      setRuntimeContentLoading(false);
      return;
    }
    const controller = new AbortController();
    const url = workflowArtifactUrl(
      task.runtimeExecution.backendTaskId,
      runtimePreview.path,
    );
    setRuntimeContent("");
    setRuntimeContentError("");
    setRuntimeContentLoading(true);
    fetch(url, { signal: controller.signal, credentials: "same-origin" })
      .then(response => {
        if (!response.ok) {
          throw new Error(`读取失败（HTTP ${response.status}）`);
        }
        return response.text();
      })
      .then(text => {
        if (runtimePreview.path.toLowerCase().endsWith(".json")) {
          try {
            setRuntimeContent(JSON.stringify(JSON.parse(text), null, 2));
            return;
          } catch {
            // JSON 不完整时仍展示原文，避免把可读产物隐藏掉。
          }
        }
        setRuntimeContent(text);
      })
      .catch(error => {
        if (error instanceof DOMException && error.name === "AbortError") return;
        setRuntimeContentError((error as Error).message);
      })
      .finally(() => setRuntimeContentLoading(false));
    return () => controller.abort();
  }, [runtimePreview, task.runtimeExecution]);

  if (isRuntime && runtimeItemsLoading && runtimeItems.length === 0) {
    return (
      <TenantCard
        state="static"
        className="mx-auto max-w-[860px] items-center bg-white py-8 text-center"
      >
        <Spinner className="mx-auto mb-2" />
        <MetaText tone="weak">正在读取真实 Agent 产物...</MetaText>
      </TenantCard>
    );
  }
  if (isRuntime && runtimeItems.length === 0) {
    return (
      <TenantCard
        state="static"
        className="mx-auto max-w-[860px] items-center bg-white py-8 text-center"
      >
        <MetaText tone="weak">
          {runtimeItemsError
            ? `产物读取失败：${runtimeItemsError}`
            : "真实工作流尚未回传产物。节点执行完成后，文件会自动汇总到这里。"}
        </MetaText>
      </TenantCard>
    );
  }
  return (
    <div className="space-y-2 max-w-[860px] mx-auto">
      {isMock && (
        <div className="rounded-[4px] border border-[#D6E4FF] bg-[#F8FBFF] px-3 py-2 mb-1">
          <MetaText tone="secondary">
            以下为示例产物（该任务节点尚未产出真实交付物）。节点执行完成后，真实产物会自动在此汇总。
          </MetaText>
        </div>
      )}
      {isRuntime &&
        runtimeItems.map(a => (
          <button
            type="button"
            key={a.id}
            onClick={() => setRuntimePreview(a)}
            className="flex w-full items-center gap-3 rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white px-4 py-3 text-left transition-colors hover:bg-[var(--bg-brand-subtle)]"
          >
            <Paperclip className="w-4 h-4 text-[var(--cp-brand-blue)] shrink-0" />
            <div className="min-w-0 flex-1">
              <div className="text-sm text-[var(--text-body)] truncate">
                {a.path}
              </div>
              <div className="text-xs text-[var(--text-weak)] mt-0.5">
                真实 Agent 产物 · {a.size ? `${a.size} B` : "文件"}
              </div>
            </div>
            <StatusTag variant="green" mode="soft">
              已回传
            </StatusTag>
            <Eye className="h-4 w-4 text-[var(--cp-brand-blue)]" />
          </button>
        ))}
      {items.map(a => (
        <div
          key={a.id}
          className="rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white px-4 py-3 flex items-center gap-3"
        >
          <Paperclip className="w-4 h-4 text-[var(--cp-brand-blue)] shrink-0" />
          <div className="min-w-0 flex-1">
            <div className="text-sm text-[var(--text-body)] truncate">
              {a.name}
            </div>
            <div className="text-xs text-[var(--text-weak)] mt-0.5">
              {a.type} · 来自节点「{a.nodeTitle}」
            </div>
          </div>
          <StatusTag variant="gray" mode="soft">
            {a.type}
          </StatusTag>
        </div>
      ))}

      <Dialog
        open={Boolean(runtimePreview)}
        onOpenChange={open => {
          if (!open) setRuntimePreview(null);
        }}
      >
        <DialogContent
          size="lg"
          className="max-h-[min(90vh,760px)] grid-rows-[auto_minmax(0,1fr)_auto]"
        >
          <DialogHeader>
            <DialogTitle>
              {runtimePreview?.path.split("/").at(-1) ?? "产物内容"}
            </DialogTitle>
            <DialogDescription>
              {runtimePreview?.path} · {runtimePreview?.size ?? 0} B
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            {runtimeContentLoading ? (
              <div className="flex min-h-48 items-center justify-center gap-2 text-sm text-[var(--text-muted)]">
                <Spinner />
                正在读取真实产物内容
              </div>
            ) : runtimeContentError ? (
              <div className="rounded-[var(--radius-card)] border border-[var(--text-danger)] bg-[var(--bg-danger-subtle)] p-4 text-sm text-[var(--text-danger)]">
                {runtimeContentError}
              </div>
            ) : (
              <pre className="m-0 max-h-[62vh] overflow-auto whitespace-pre-wrap break-words rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-[var(--bg-subtle)] p-4 text-sm leading-6 text-[var(--text-body)]">
                {runtimeContent || "该产物暂无文本内容。"}
              </pre>
            )}
          </DialogBody>
          <DialogFooter>
            {runtimePreview && task.runtimeExecution && (
              <a
                href={workflowArtifactUrl(
                  task.runtimeExecution.backendTaskId,
                  runtimePreview.path,
                )}
                target="_blank"
                rel="noreferrer"
                className="inline-flex h-8 items-center gap-1.5 rounded-[var(--radius-md)] border border-[var(--cp-border)] px-3 text-sm text-[var(--text-body)] hover:bg-[var(--bg-subtle)]"
              >
                <ExternalLink className="h-3.5 w-3.5" />
                打开原文件
              </a>
            )}
            <Button variant="tenant-primary" onClick={() => setRuntimePreview(null)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function TaskInfoView({
  task,
  project,
  owner,
  taskStatus,
}: {
  task: TenantTask;
  project: TenantProject;
  owner?: TenantProjectMember;
  taskStatus: { label: string };
}) {
  const pipeline = task.pipelineTemplateId
    ? project.pipelineTemplates.find(t => t.id === task.pipelineTemplateId)
    : undefined;
  const triggerLabel: Record<NonNullable<TenantTask["triggerType"]>, string> = {
    manual: "人手动创建",
    periodic: "周期触发",
    external: "外部 agent 触发（MCP）",
    system: "系统自动创建（如自动化）",
  };
  const rows: Array<{ label: string; value: React.ReactNode }> = [
    { label: "任务名称", value: task.title },
    { label: "描述", value: task.description || "—" },
    { label: "状态", value: taskStatus.label },
    { label: "负责人", value: owner?.displayName ?? task.ownerId },
    { label: "截止时间", value: task.dueDate },
    {
      label: "触发来源",
      value: triggerLabel[task.triggerType ?? "manual"],
    },
    {
      label: "关联工作流",
      value: pipeline ? pipeline.name : "未关联",
    },
    ...(task.runtimeExecution
      ? [
          {
            label: "真实执行任务",
            value: task.runtimeExecution.backendTaskId,
          },
          {
            label: "交接契约",
            value: task.runtimeExecution.handoffContract,
          },
        ]
      : []),
    {
      label: "标签",
      value:
        task.tags && task.tags.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {task.tags.map(t => (
              <StatusTag key={t} variant="gray" mode="soft">
                {t}
              </StatusTag>
            ))}
          </div>
        ) : (
          "—"
        ),
    },
    { label: "创建时间", value: task.createdAt.slice(0, 19).replace("T", " ") },
    { label: "更新时间", value: task.updatedAt.slice(0, 19).replace("T", " ") },
  ];
  return (
    <div className="max-w-[860px] mx-auto rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white divide-y divide-[var(--cp-border)]">
      {rows.map(r => (
        <div key={r.label} className="px-4 py-3 flex items-start gap-4">
          <div className="w-28 shrink-0 text-sm text-[var(--text-weak)]">
            {r.label}
          </div>
          <div className="flex-1 min-w-0 text-sm text-[var(--text-body)]">
            {r.value}
          </div>
        </div>
      ))}
    </div>
  );
}

// ─── 流水线视图（场景2：标准化任务流水线） ─────────────────
const PIPELINE_SOURCE_LABEL: Record<
  NonNullable<TenantTask["source"]>,
  string
> = {
  manual: "手动",
  import: "导入",
  mcp: "Agent 上报",
};

/** 把模板节点按依赖拓扑分层：同层节点无相互依赖，可并行执行 */
function templateStages(
  template: TenantPipelineTemplate
): TenantPipelineTemplateNode[][] {
  const stageOf: Record<string, number> = {};
  const compute = (id: string): number => {
    if (stageOf[id] !== undefined) return stageOf[id];
    const node = template.nodes.find(n => n.id === id);
    if (!node || node.dependsOn.length === 0) {
      stageOf[id] = 0;
      return 0;
    }
    const s = Math.max(...node.dependsOn.map(compute)) + 1;
    stageOf[id] = s;
    return s;
  };
  template.nodes.forEach(n => compute(n.id));
  const maxStage = Math.max(0, ...Object.values(stageOf));
  const stages: TenantPipelineTemplateNode[][] = Array.from(
    { length: maxStage + 1 },
    () => []
  );
  template.nodes.forEach(n => stages[stageOf[n.id]].push(n));
  return stages;
}

/** 计算 issue 当前所处阶段：第一个未确认节点的序号；全部确认落在“已完成”列 */
function pipelineStageOf(task: TenantTask, template: TenantPipelineTemplate) {
  if (task.status === "done") return template.nodes.length;
  for (let i = 0; i < template.nodes.length; i++) {
    const node = task.workflow.find(n => n.id === `${task.id}-node-${i + 1}`);
    if (!node || node.status !== "confirmed") return i;
  }
  return template.nodes.length;
}

function PipelineIssueCard({
  task,
  agents,
  onOpen,
}: {
  task: TenantTask;
  agents: TenantProject["agents"];
  onOpen: () => void;
}) {
  const activeNode =
    task.workflow.find(n => n.status === "running" || n.status === "review") ??
    task.workflow.find(n => n.status !== "confirmed");
  const agent = agents.find(a => a.id === activeNode?.agentId);
  return (
    <button
      type="button"
      onClick={onOpen}
      className="w-full text-left rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-white px-3 py-2.5 transition-colors hover:border-[var(--cp-brand-blue)]"
    >
      <div className="flex items-start gap-2">
        <span className="text-sm font-medium text-[var(--text-title)] line-clamp-2">
          {task.title}
        </span>
      </div>
      {task.status === "done" ? (
        <div className="mt-2 inline-flex items-center gap-1 text-xs text-[var(--text-success)]">
          <CircleCheck className="w-3.5 h-3.5" />
          全部节点已确认
        </div>
      ) : (
        <div className="mt-2 flex items-center gap-1.5 text-xs text-[var(--text-muted)]">
          <GitBranch className="w-3.5 h-3.5 text-[var(--cp-brand-blue)]" />
          <span className="text-[var(--text-body)]">{activeNode?.title}</span>
          {agent && (
            <span className="inline-flex items-center gap-1 text-[var(--text-weak)]">
              <Bot className="w-3 h-3" />
              {agent.name}
            </span>
          )}
        </div>
      )}
      <div className="mt-2 flex items-center justify-between text-[11px] text-[var(--text-weak)]">
        <span className="inline-flex items-center gap-1">
          <Download className="w-3 h-3" />
          {PIPELINE_SOURCE_LABEL[task.source ?? "manual"]}
        </span>
        <span className="inline-flex items-center gap-1">
          <CalendarDays className="w-3 h-3" />
          {formatDueDate(task.dueDate)}
        </span>
      </div>
    </button>
  );
}

function CreateIssueDialog({
  open,
  onOpenChange,
  template,
  onCreate,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  template: TenantPipelineTemplate | undefined;
  onCreate: (input: {
    title: string;
    priority: TenantTaskPriority;
    dueDate: string;
  }) => void;
}) {
  const [title, setTitle] = useState("");
  const [dueDate, setDueDate] = useState("2026-08-15");

  useEffect(() => {
    if (open) {
      setTitle("");
      setDueDate("2026-08-15");
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>新建 issue</DialogTitle>
          <DialogDescription>
            按「{template?.name}」实例化工作流，创建后自动为各节点匹配并指派
            Agent。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="space-y-3">
          <div className="space-y-1.5">
            <Label>标题</Label>
            <Input
              value={title}
              onChange={e => setTitle(e.target.value)}
              placeholder="例如：登录接口偶发 502"
            />
          </div>
          <div className="space-y-1.5">
            <Label>截止日期</Label>
            <Input
              type="date"
              value={dueDate}
              onChange={e => setDueDate(e.target.value)}
            />
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="tenant-primary"
            disabled={!title.trim()}
            onClick={() => {
              onCreate({ title: title.trim(), priority: "medium", dueDate });
              onOpenChange(false);
            }}
          >
            创建
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function BatchImportDialog({
  open,
  onOpenChange,
  template,
  onImport,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  template: TenantPipelineTemplate | undefined;
  onImport: (titles: string[]) => void;
}) {
  const [text, setText] = useState("");
  useEffect(() => {
    if (open) setText("");
  }, [open]);
  const lines = text
    .split("\n")
    .map(l => l.trim())
    .filter(Boolean);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>批量导入 issue</DialogTitle>
          <DialogDescription>
            每行一条，统一走「{template?.name}」流水线，导入后自动指派 Agent。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="space-y-2">
          <Textarea
            rows={7}
            value={text}
            onChange={e => setText(e.target.value)}
            placeholder={"登录接口偶发 502\nCVM 列表加载缓慢\n导出报表中文乱码"}
          />
          <MetaText tone="weak">已识别 {lines.length} 条</MetaText>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="tenant-primary"
            disabled={lines.length === 0}
            onClick={() => {
              onImport(lines);
              onOpenChange(false);
            }}
          >
            导入 {lines.length} 条
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface EditorNode {
  title: string;
  agentRole: string;
  /** 依赖的前序节点索引集合（只能依赖排在自己前面的节点，天然无环 → 支持并行 DAG） */
  dependsOn: number[];
}

function PipelineTemplateEditor({
  open,
  onOpenChange,
  template,
  onSave,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  /** 传入表示编辑，null 表示新建 */
  template: TenantPipelineTemplate | null;
  onSave: (
    template: TenantPipelineTemplate | null,
    data: { name: string; description: string; nodes: EditorNode[] }
  ) => void;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nodes, setNodes] = useState<EditorNode[]>([]);

  useEffect(() => {
    if (!open) return;
    if (template) {
      setName(template.name);
      setDescription(template.description);
      const idToIndex: Record<string, number> = {};
      template.nodes.forEach((n, i) => {
        idToIndex[n.id] = i;
      });
      setNodes(
        template.nodes.map(n => ({
          title: n.title,
          agentRole: n.agentRole,
          dependsOn: n.dependsOn
            .map(d => idToIndex[d])
            .filter(idx => idx !== undefined),
        }))
      );
    } else {
      setName("");
      setDescription("");
      setNodes([{ title: "", agentRole: "", dependsOn: [] }]);
    }
  }, [open, template]);

  const valid =
    name.trim() && nodes.length > 0 && nodes.every(n => n.title.trim());

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>
            {template ? "编辑流水线模板" : "新建流水线模板"}
          </DialogTitle>
          <DialogDescription>
            定义这类工作的
            SOP：为节点勾选依赖的前序节点，可让多个节点并行、汇合后再进下一步（DAG）。实例化时按角色自动匹配
            Agent。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="space-y-3">
          <div className="space-y-1.5">
            <Label>模板名称</Label>
            <Input
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="例如：工单处理流水线"
            />
          </div>
          <div className="space-y-1.5">
            <Label>说明</Label>
            <Input
              value={description}
              onChange={e => setDescription(e.target.value)}
              placeholder="一句话描述这条流水线处理什么"
            />
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>节点（勾选依赖即可并行）</Label>
              <Button
                variant="tenant-outline"
                size="sm"
                onClick={() =>
                  setNodes(prev => [
                    ...prev,
                    {
                      title: "",
                      agentRole: "",
                      // 默认串在上一个节点后面，用户可自行改成并行
                      dependsOn: prev.length > 0 ? [prev.length - 1] : [],
                    },
                  ])
                }
              >
                <Plus className="w-4 h-4" />
                添加节点
              </Button>
            </div>
            <div className="space-y-2">
              {nodes.map((node, i) => (
                <div
                  key={i}
                  className="rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-2.5 space-y-2"
                >
                  <div className="flex items-center gap-2">
                    <span className="shrink-0 flex h-6 w-6 items-center justify-center rounded-full bg-[var(--color-gray-100)] text-xs tabular-nums text-[var(--text-muted)]">
                      {i + 1}
                    </span>
                    <Input
                      className="flex-1"
                      value={node.title}
                      onChange={e =>
                        setNodes(prev =>
                          prev.map((n, idx) =>
                            idx === i ? { ...n, title: e.target.value } : n
                          )
                        )
                      }
                      placeholder="节点名称，如 根因定位"
                    />
                    <Input
                      className="w-32"
                      value={node.agentRole}
                      onChange={e =>
                        setNodes(prev =>
                          prev.map((n, idx) =>
                            idx === i ? { ...n, agentRole: e.target.value } : n
                          )
                        )
                      }
                      placeholder="角色，如 后端"
                    />
                    <button
                      type="button"
                      aria-label="删除节点"
                      disabled={nodes.length <= 1}
                      onClick={() =>
                        setNodes(prev => removeEditorNode(prev, i))
                      }
                      className="shrink-0 flex h-8 w-8 items-center justify-center rounded-[var(--radius-md)] text-[var(--text-muted)] transition-colors hover:bg-[var(--color-gray-100)] disabled:opacity-40"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                  {i > 0 && (
                    <div className="flex flex-wrap items-center gap-1.5 pl-8">
                      <span className="text-[11px] text-[var(--text-weak)]">
                        依赖：
                      </span>
                      {nodes.slice(0, i).map((dep, di) => {
                        const on = node.dependsOn.includes(di);
                        return (
                          <button
                            key={di}
                            type="button"
                            onClick={() =>
                              setNodes(prev =>
                                prev.map((n, idx) => {
                                  if (idx !== i) return n;
                                  const set = new Set(n.dependsOn);
                                  if (set.has(di)) set.delete(di);
                                  else set.add(di);
                                  return {
                                    ...n,
                                    dependsOn: Array.from(set).sort(
                                      (a, b) => a - b
                                    ),
                                  };
                                })
                              )
                            }
                            className={`inline-flex items-center gap-1 h-6 px-2 rounded-[4px] border text-[11px] transition-colors ${
                              on
                                ? "border-[var(--cp-brand-blue)] bg-[#F0F5FF] text-[var(--cp-brand-blue)]"
                                : "border-[var(--cp-border)] bg-white text-[var(--text-muted)] hover:border-[var(--text-weak)]"
                            }`}
                          >
                            {on && <CircleCheck className="w-3 h-3" />}
                            {di + 1} {dep.title.trim() || "未命名"}
                          </button>
                        );
                      })}
                      {node.dependsOn.length === 0 && (
                        <span className="text-[11px] text-[var(--text-weak)]">
                          无依赖（与其他起始节点并行）
                        </span>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="tenant-primary"
            disabled={!valid}
            onClick={() => {
              onSave(template, {
                name: name.trim(),
                description: description.trim(),
                nodes: nodes.map(n => ({
                  title: n.title.trim(),
                  agentRole: n.agentRole.trim() || "研发",
                  dependsOn: n.dependsOn,
                })),
              });
              onOpenChange(false);
            }}
          >
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** EditorNode 列表 → 模板节点（保留并行 DAG 依赖，索引依赖映射成节点 id） */
function editorNodesToTemplateNodes(
  nodes: EditorNode[]
): TenantPipelineTemplateNode[] {
  return nodes.map((n, i) => ({
    id: `n${i + 1}`,
    title: n.title,
    dependsOn: Array.from(new Set(n.dependsOn))
      .filter(idx => idx >= 0 && idx < i)
      .map(idx => `n${idx + 1}`),
    agentRole: n.agentRole,
  }));
}

/** 删除第 removeIdx 个节点，并把所有引用它/其后节点的依赖索引重新校正 */
function removeEditorNode(
  nodes: EditorNode[],
  removeIdx: number
): EditorNode[] {
  return nodes
    .filter((_, idx) => idx !== removeIdx)
    .map(n => ({
      ...n,
      dependsOn: n.dependsOn
        .filter(d => d !== removeIdx)
        .map(d => (d > removeIdx ? d - 1 : d)),
    }));
}

function PipelineView({
  project,
  onOpenTask,
}: {
  project: TenantProject;
  onOpenTask: (taskId: string) => void;
}) {
  const templates = project.pipelineTemplates;
  const [templateId, setTemplateId] = useState(templates[0]?.id ?? "");
  const [createIssueOpen, setCreateIssueOpen] = useState(false);
  const [batchOpen, setBatchOpen] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [canvasOpen, setCanvasOpen] = useState(false);
  const [mcpOpen, setMcpOpen] = useState(false);
  const [canvasTemplate, setCanvasTemplate] =
    useState<TenantPipelineTemplate | null>(null);
  const [editingTemplate, setEditingTemplate] =
    useState<TenantPipelineTemplate | null>(null);

  useEffect(() => {
    if (templates.length && !templates.find(t => t.id === templateId)) {
      setTemplateId(templates[0].id);
    }
  }, [templates, templateId]);

  const template = templates.find(t => t.id === templateId);

  const openEditTemplate = () => {
    if (!template) return;
    setEditingTemplate(template);
    setEditorOpen(true);
  };
  const handleSaveTemplate = (
    editing: TenantPipelineTemplate | null,
    data: { name: string; description: string; nodes: EditorNode[] }
  ) => {
    const nodes = editorNodesToTemplateNodes(data.nodes);
    if (editing) {
      tenantProjectStore.updatePipelineTemplate(project.id, editing.id, {
        name: data.name,
        description: data.description,
        nodes,
      });
      toast.success("模板已更新");
    } else {
      const newId = tenantProjectStore.createPipelineTemplate(project.id, {
        name: data.name,
        description: data.description,
        nodes,
      });
      setTemplateId(newId);
      toast.success("模板已创建");
    }
  };

  // 画布编辑/新建：WorkflowCanvas 直接给出标准节点，保存走 create/update
  const openCanvas = (tpl: TenantPipelineTemplate | null) => {
    setCanvasTemplate(tpl);
    setCanvasOpen(true);
  };
  const handleCanvasSave = (
    editing: TenantPipelineTemplate | null,
    data: {
      name: string;
      description: string;
      nodes: TenantPipelineTemplateNode[];
    }
  ) => {
    if (editing) {
      tenantProjectStore.updatePipelineTemplate(project.id, editing.id, {
        name: data.name,
        description: data.description,
        nodes: data.nodes,
      });
      toast.success("模板已更新");
    } else {
      const newId = tenantProjectStore.createPipelineTemplate(project.id, {
        name: data.name,
        description: data.description,
        nodes: data.nodes,
      });
      setTemplateId(newId);
      toast.success("模板已创建");
    }
  };
  // agent 经 MCP 回传工作流 → 落库为模板
  const handleMcpReceive = (
    nodes: TenantPipelineTemplateNode[],
    meta: { name: string; description: string }
  ) => {
    const newId = tenantProjectStore.receiveWorkflowFromMCP(
      project.id,
      nodes,
      meta
    );
    if (newId) setTemplateId(newId);
  };

  if (!template) {
    return (
      <>
        <Empty>
          <EmptyHeader>暂无流水线模板</EmptyHeader>
          <EmptyDescription>
            用画布新建一条模板，或让 agent 经 MCP 回传，固化某类工作的标准流程。
          </EmptyDescription>
          <div className="mt-3 flex items-center justify-center gap-2">
            <Button
              variant="tenant-primary"
              size="sm"
              onClick={() => openCanvas(null)}
            >
              <Workflow className="w-4 h-4" />
              画布新建模板
            </Button>
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={() => setMcpOpen(true)}
            >
              <Bot className="w-4 h-4" />
              MCP 回传
            </Button>
          </div>
        </Empty>
        <PipelineTemplateEditor
          open={editorOpen}
          onOpenChange={setEditorOpen}
          template={editingTemplate}
          onSave={handleSaveTemplate}
        />
        <WorkflowCanvas
          open={canvasOpen}
          onOpenChange={setCanvasOpen}
          template={canvasTemplate}
          onSave={handleCanvasSave}
          members={project.members}
          agents={project.agents}
        />
        <McpImportDialog
          open={mcpOpen}
          onOpenChange={setMcpOpen}
          onReceive={handleMcpReceive}
        />
      </>
    );
  }

  const issues = project.tasks.filter(
    t => t.pipelineTemplateId === template.id
  );
  const columns = [
    ...template.nodes.map((n, i) => ({
      key: n.id,
      label: n.title,
      role: n.agentRole,
      index: i,
      done: false,
    })),
    {
      key: "__done__",
      label: "已完成",
      role: "",
      index: template.nodes.length,
      done: true,
    },
  ];

  return (
    <div className="space-y-4">
      {/* 模板栏 */}
      <div className="rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Layers className="w-4 h-4 text-[var(--cp-brand-blue)]" />
            <Select value={templateId} onValueChange={setTemplateId}>
              <SelectTrigger className="w-56">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {templates.map(t => (
                  <SelectItem key={t.id} value={t.id}>
                    {t.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <StatusTag variant="gray" mode="soft">
              {template.nodes.length} 个节点
            </StatusTag>
            <StatusTag variant="blue" mode="soft">
              复用{" "}
              {tenantProjectStore.countTemplateUsage(project.id, template.id)}{" "}
              次
            </StatusTag>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={() => openCanvas(template)}
            >
              <Workflow className="w-4 h-4" />
              画布编辑
            </Button>
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={openEditTemplate}
            >
              <Pencil className="w-4 h-4" />
              表单编辑
            </Button>
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={() => setMcpOpen(true)}
            >
              <Bot className="w-4 h-4" />
              MCP 回传
            </Button>
            <Button
              variant="tenant-outline"
              size="sm"
              onClick={() => openCanvas(null)}
            >
              <Plus className="w-4 h-4" />
              新建模板
            </Button>
          </div>
        </div>
        <MetaText tone="secondary" className="mt-2 block">
          {template.description}
        </MetaText>
        {/* 节点链：按拓扑层级分组，同层并排展示并行分支 */}
        <div className="mt-3 flex flex-wrap items-center gap-1.5">
          {templateStages(template).map((stage, si, arr) => (
            <div key={si} className="flex items-center gap-1.5">
              <div className="flex flex-col gap-1">
                {stage.map(n => (
                  <span
                    key={n.id}
                    className="inline-flex items-center gap-1.5 rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-white px-2.5 py-1 text-xs text-[var(--text-body)]"
                  >
                    {n.title}
                    <span className="inline-flex items-center gap-0.5 text-[var(--text-weak)]">
                      <UserRound className="w-3 h-3" />
                      {n.agentRole}
                    </span>
                  </span>
                ))}
              </div>
              {si < arr.length - 1 && (
                <ChevronRight className="w-3.5 h-3.5 text-[var(--text-weak)]" />
              )}
            </div>
          ))}
        </div>
      </div>

      {/* 操作条 */}
      <div className="flex items-center justify-between gap-3">
        <MetaText tone="weak">共 {issues.length} 条 issue</MetaText>
        <div className="flex items-center gap-2">
          <Button
            variant="tenant-outline"
            size="sm"
            onClick={() => setBatchOpen(true)}
          >
            <Upload className="w-4 h-4" />
            批量导入
          </Button>
          <Button
            variant="tenant-primary"
            size="sm"
            onClick={() => setCreateIssueOpen(true)}
          >
            <Plus className="w-4 h-4" />
            新建 issue
          </Button>
        </div>
      </div>

      {/* 按阶段分列的 issue 看板 */}
      <div className="flex gap-3 overflow-x-auto pb-2">
        {columns.map(col => {
          const colIssues = issues.filter(
            it => pipelineStageOf(it, template) === col.index
          );
          return (
            <div
              key={col.key}
              className="flex min-h-[280px] w-[248px] shrink-0 flex-col rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-[var(--cp-surface)]"
            >
              <div className="flex items-center gap-2 border-b border-[var(--cp-border)] px-3 py-2.5">
                <span
                  className={`h-1.5 w-1.5 rounded-full ${
                    col.done
                      ? "bg-[var(--text-success)]"
                      : "bg-[var(--cp-brand-blue)]"
                  }`}
                />
                <span className="text-sm font-medium text-[var(--text-body)]">
                  {col.label}
                </span>
                <span className="ml-auto text-xs tabular-nums text-[var(--text-weak)]">
                  {colIssues.length}
                </span>
              </div>
              <div className="flex-1 space-y-2 p-2">
                {colIssues.length === 0 ? (
                  <div className="flex min-h-[160px] items-center justify-center text-xs text-[var(--text-weak)]">
                    —
                  </div>
                ) : (
                  colIssues.map(task => (
                    <PipelineIssueCard
                      key={task.id}
                      task={task}
                      agents={project.agents}
                      onOpen={() => onOpenTask(task.id)}
                    />
                  ))
                )}
              </div>
            </div>
          );
        })}
      </div>

      <CreateIssueDialog
        open={createIssueOpen}
        onOpenChange={setCreateIssueOpen}
        template={template}
        onCreate={input => {
          tenantProjectStore.createIssue(project.id, {
            ...input,
            pipelineTemplateId: template.id,
            source: "manual",
          });
          toast.success("issue 已创建，已按角色自动指派 Agent");
        }}
      />
      <BatchImportDialog
        open={batchOpen}
        onOpenChange={setBatchOpen}
        template={template}
        onImport={titles => {
          const n = tenantProjectStore.createIssuesBatch(
            project.id,
            titles,
            template.id
          );
          toast.success(`已导入 ${n} 条 issue`);
        }}
      />
      <PipelineTemplateEditor
        open={editorOpen}
        onOpenChange={setEditorOpen}
        template={editingTemplate}
        onSave={handleSaveTemplate}
      />
      <WorkflowCanvas
        open={canvasOpen}
        onOpenChange={setCanvasOpen}
        template={canvasTemplate}
        onSave={handleCanvasSave}
        members={project.members}
        agents={project.agents}
      />
      <McpImportDialog
        open={mcpOpen}
        onOpenChange={setMcpOpen}
        onReceive={handleMcpReceive}
      />
    </div>
  );
}

// ─── 总览 Tab（自动化、协作运行、项目上手 Skill） ─────────
function OverviewTab({
  project,
  onGotoTasks,
  onGotoWorkflow,
  onGotoAssets,
}: {
  project: TenantProject;
  onGotoTasks: () => void;
  onGotoWorkflow: () => void;
  onGotoAssets: () => void;
}) {
  const [introSkillOpen, setIntroSkillOpen] = useState(false);
  const introSkill = useMemo(() => getProjectIntroSkill(project), [project]);
  const assetCategoryStats = ASSET_CATEGORY_ORDER.map(cat => ({
    cat,
    label: ASSET_CATEGORY_MAP[cat].label,
    count: project.assets[cat]?.length ?? 0,
  }));
  const totalAssets = assetCategoryStats.reduce((s, c) => s + c.count, 0);
  const totalTasks = project.tasks.length;
  const doneTasks = project.tasks.filter(t => t.status === "done").length;
  const inProgressTasks = project.tasks.filter(
    t => t.status === "in_progress" || t.status === "review"
  ).length;
  const todoTasks = project.tasks.filter(t => t.status === "todo").length;
  const holdTasks = project.tasks.filter(t => t.status === "hold").length;
  const taskCompletion = totalTasks
    ? Math.round((doneTasks / totalTasks) * 100)
    : 0;
  const onlineAgents = project.agents.filter(
    agent => agent.status === "online"
  ).length;
  const localAgents = project.agents.filter(
    agent => agent.location === "local"
  ).length;
  const cloudAgents = project.agents.length - localAgents;
  const assetScaleMax = Math.max(
    1,
    ...assetCategoryStats.map(item => item.count)
  );

  return (
    <div className="space-y-3 pb-2">
      <ExperienceAutomationControls project={project} />

      <div>
        <section className="overflow-hidden rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white">
          <div className="flex items-center justify-between border-b border-[var(--cp-border)] px-4 py-3">
            <div>
              <BodyMedium>协作运行</BodyMedium>
              <MetaText tone="weak" className="ml-2">任务进展与资源协作关系</MetaText>
            </div>
            <div className="flex items-center gap-4">
              <button type="button" onClick={onGotoTasks} className="text-xs text-[var(--cp-brand-blue)] hover:underline">查看任务</button>
              <button type="button" onClick={onGotoAssets} className="text-xs text-[var(--cp-brand-blue)] hover:underline">管理资产</button>
            </div>
          </div>

          <div className="grid grid-cols-[230px_minmax(0,1fr)] divide-x divide-[var(--cp-border)]">
            <div className="flex flex-col items-center justify-center px-5 py-5">
              <div
                className="relative flex h-32 w-32 items-center justify-center rounded-full"
                style={{
                  background: `conic-gradient(var(--cp-brand-blue) ${taskCompletion}%, var(--color-gray-100) 0)`,
                }}
              >
                <div className="absolute inset-[9px] rounded-full bg-white" />
                <div className="relative text-center">
                  <div className="text-3xl font-semibold tabular-nums text-[var(--text-title)]">{taskCompletion}%</div>
                  <MetaText tone="weak">任务完成</MetaText>
                </div>
              </div>
              <div className="mt-4 flex items-center gap-3 text-xs text-[var(--text-muted)]">
                <span><b className="font-medium text-[var(--text-title)]">{doneTasks}</b> 完成</span>
                <span className="h-3 w-px bg-[var(--cp-border)]" />
                <span><b className="font-medium text-[var(--text-title)]">{inProgressTasks}</b> 进行中</span>
                <span className="h-3 w-px bg-[var(--cp-border)]" />
                <span><b className="font-medium text-[var(--text-title)]">{todoTasks + holdTasks}</b> 待处理</span>
              </div>
            </div>

            <div className="min-w-0 p-5">
              <div className="flex items-center justify-center gap-3">
                {[
                  { label: "项目成员", value: project.members.length, icon: Users },
                  { label: "在线 Agent", value: onlineAgents, icon: Bot },
                  { label: "项目资产", value: totalAssets, icon: Boxes },
                ].map((item, index) => {
                  const ItemIcon = item.icon;
                  return (
                    <div key={item.label} className="contents">
                      {index > 0 && (
                        <div className="flex min-w-8 flex-1 items-center">
                          <span className="h-px flex-1 bg-[var(--cp-border)]" />
                          <ChevronRight className="h-4 w-4 text-[var(--text-weak)]" />
                        </div>
                      )}
                      <div className="flex min-w-[130px] items-center gap-3 rounded-[8px] border border-[var(--cp-border)] bg-[var(--color-gray-50)] px-3 py-3">
                        <span className="flex h-8 w-8 items-center justify-center rounded-[6px] bg-white text-[var(--cp-brand-blue)] shadow-sm"><ItemIcon className="h-4 w-4" /></span>
                        <div><div className="text-lg font-semibold tabular-nums text-[var(--text-title)]">{item.value}</div><MetaText tone="weak">{item.label}</MetaText></div>
                      </div>
                    </div>
                  );
                })}
              </div>

              <div className="mt-5 grid grid-cols-[180px_minmax(0,1fr)] gap-5 border-t border-[var(--cp-border)] pt-4">
                <div>
                  <div className="mb-2 flex items-center justify-between text-xs text-[var(--text-muted)]"><span>Agent 形态</span><span>{project.agents.length} 个</span></div>
                  <div className="flex h-2 overflow-hidden rounded-full bg-[var(--color-gray-100)]">
                    {localAgents > 0 && <span className="bg-[var(--cp-brand-blue)]" style={{ width: `${(localAgents / Math.max(1, project.agents.length)) * 100}%` }} />}
                    {cloudAgents > 0 && <span className="bg-[var(--color-gray-300)]" style={{ width: `${(cloudAgents / Math.max(1, project.agents.length)) * 100}%` }} />}
                  </div>
                  <div className="mt-2 flex items-center justify-between text-xs text-[var(--text-muted)]"><span>本地 {localAgents}</span><span>云端 {cloudAgents}</span></div>
                  <button type="button" onClick={onGotoWorkflow} className="mt-4 flex items-center gap-1 text-xs text-[var(--cp-brand-blue)] hover:underline">{project.pipelineTemplates.length} 条工作流 <ChevronRight className="h-3.5 w-3.5" /></button>
                </div>

                <div>
                  <div className="mb-2 flex items-center justify-between text-xs text-[var(--text-muted)]"><span>资产构成</span><span>共 {totalAssets}</span></div>
                  <div className="flex h-[62px] items-end gap-2">
                    {assetCategoryStats.map(item => (
                      <div key={item.cat} className="group flex min-w-0 flex-1 flex-col items-center justify-end gap-1">
                        <span className="text-[10px] tabular-nums text-[var(--text-weak)] opacity-0 transition-opacity group-hover:opacity-100">{item.count}</span>
                        <div className="w-full max-w-8 rounded-t-[3px] bg-[var(--cp-brand-blue)] transition-opacity group-hover:opacity-80" style={{ height: `${8 + (item.count / assetScaleMax) * 34}px`, opacity: item.count > 0 ? 1 : 0.15 }} />
                        <span className="w-full truncate text-center text-[10px] text-[var(--text-muted)]">{item.label.replace("企业", "")}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </section>

      </div>

      <button
        type="button"
        onClick={() => setIntroSkillOpen(true)}
        className="group flex w-full items-center gap-3 rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white px-4 py-3 text-left transition-colors hover:border-[var(--cp-brand-blue)] hover:bg-[var(--cp-brand-tint)]"
      >
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-[6px] bg-[var(--cp-brand-tint)] text-[var(--cp-brand-blue)]"><Sparkles className="h-4 w-4" /></span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2"><BodyMedium>项目上手 Skill</BodyMedium><StatusTag variant="blue" mode="soft">v{introSkill.version}</StatusTag></div>
          <MetaText tone="weak" className="mt-0.5 block truncate">Agent 关联项目后自动获得的第一份上下文 · {project.metrics.agentCount} 个 Agent 已读取</MetaText>
        </div>
        <span className="text-xs text-[var(--cp-brand-blue)]">查看内容</span>
        <ChevronRight className="h-4 w-4 shrink-0 text-[var(--text-weak)] transition-transform group-hover:translate-x-0.5" />
      </button>

      <ProjectIntroSkillDialog
        open={introSkillOpen}
        onOpenChange={setIntroSkillOpen}
        project={project}
      />
    </div>
  );
}

/** 项目上手 Skill 详情弹窗：点击资产清单里的「项目上手 Skill」特殊项后打开。
 *  展示由getProjectIntroSkill 派生的完整内容——外部 agent 关联本项目后自动获得的第一份上下文，
 *  随每次资产下发强制携带，用户不可删除。 */
function ProjectIntroSkillDialog({
  open,
  onOpenChange,
  project,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  project: TenantProject;
}) {
  const skill = useMemo(() => getProjectIntroSkill(project), [project]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-[680px] flex flex-col"
        style={{ maxHeight: "min(90vh, 760px)" }}
      >
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-[var(--cp-brand-blue)]" />
            项目上手 Skill
            <StatusTag variant="blue" mode="soft">
              v{skill.version}
            </StatusTag>
            <StatusTag variant="gray" mode="soft">
              系统内置 · 必带
            </StatusTag>
          </DialogTitle>
          <DialogDescription>
            系统自动生成、随资产下发的第一份项目上下文。成员的 agent
            关联本项目后自动获得，即可知道项目内容、任务、工作流与使用方式，无需打开
            ClawPro 页面。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="px-6 space-y-3">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="inline-flex items-center gap-1 h-7 px-2 rounded-[4px] bg-[var(--color-gray-100)] border border-[var(--cp-border)] text-xs text-[var(--text-weak)]">
              <Bot className="w-3.5 h-3.5 text-[var(--cp-brand-blue)]" />
              已被 {project.metrics.agentCount} 个 agent 关联读取
            </span>
            <MetaText tone="weak">
              由项目数据实时派生，任何字段变化后自动更新。
            </MetaText>
          </div>
          <pre className="whitespace-pre-wrap text-xs text-[var(--text-body)] leading-relaxed bg-[var(--color-gray-100)] rounded-[4px] border border-[var(--cp-border)] p-3 max-h-[52vh] overflow-auto">
            {skill.content}
          </pre>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            关闭
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function TaskTab({
  project,
  onOpenTask,
  createSignal,
  onGotoWorkflow,
}: {
  project: TenantProject;
  onOpenTask: (taskId: string) => void;
  /** 外部触发新建（Tab 栏的"新建任务"按钮通过它触发，避免此处再放一次按钮） */
  createSignal: number;
  onGotoWorkflow: () => void;
}) {
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTaskId, setDeleteTaskId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const deleteTask = project.tasks.find(task => task.id === deleteTaskId);
  const lastCreateSignalRef = useRef(createSignal);

  // 外部信号驱动打开创建弹窗：只在信号"递增"时开，避免 Tab 切回时重挂 useEffect 误触发
  useEffect(() => {
    if (createSignal > lastCreateSignalRef.current) {
      setCreateOpen(true);
    }
    lastCreateSignalRef.current = createSignal;
  }, [createSignal]);

  // 应用搜索后的任务列表
  const filteredTasks = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return project.tasks;
    return project.tasks.filter(t => {
      const wf = project.pipelineTemplates.find(
        p => p.id === t.pipelineTemplateId
      );
      const haystack = [t.title, wf?.name ?? ""].join(" ").toLowerCase();
      return haystack.includes(q);
    });
  }, [project.tasks, project.pipelineTemplates, search]);

  return (
    <div className="space-y-4">
      {/* 搜索 */}
      <div className="flex items-center gap-2 flex-wrap">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-muted)]" />
          <Input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="搜索任务名称、工作流"
            className="pl-8 h-8 w-64"
          />
        </div>
      </div>

      <div className="grid grid-cols-4 gap-2">
        {TASK_COLUMNS.map(column => {
          const tasks = filteredTasks.filter(task =>
            column.statuses.includes(task.status)
          );
          return (
            <div
              key={column.key}
              className="flex min-h-[260px] flex-col rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-[var(--cp-surface)]"
            >
              <div className="flex items-center gap-2 border-b border-[var(--cp-border)] px-3 py-2">
                <span
                  className={`h-1.5 w-1.5 rounded-full ${column.dotClass}`}
                />
                <span className="text-sm font-medium text-[var(--text-body)]">
                  {column.label}
                </span>
                <span className="text-xs tabular-nums text-[var(--text-weak)]">
                  {tasks.length}
                </span>
              </div>
              <div className="flex-1 space-y-1.5 p-1.5">
                {tasks.length === 0 ? (
                  <div className="flex min-h-[160px] items-center justify-center text-xs text-[var(--text-weak)]">
                    暂无任务
                  </div>
                ) : (
                  tasks.map(task => (
                    <TaskCard
                      key={task.id}
                      task={task}
                      onOpen={() => onOpenTask(task.id)}
                      onDelete={() => setDeleteTaskId(task.id)}
                    />
                  ))
                )}
              </div>
            </div>
          );
        })}
      </div>

      <CreateTaskDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        pipelineTemplates={project.pipelineTemplates}
        onGotoWorkflow={onGotoWorkflow}
        onCreate={input => {
          tenantProjectStore.createTask(project.id, input);
          toast.success("任务已创建，工作流节点等待选择 Agent");
        }}
      />
      <Dialog
        open={!!deleteTaskId}
        onOpenChange={open => !open && setDeleteTaskId(null)}
      >
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>删除任务</DialogTitle>
            <DialogDescription>
              删除“{deleteTask?.title}
              ”后，其工作流、执行结果和附件记录也会一并移除。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="tenant-outline"
              onClick={() => setDeleteTaskId(null)}
            >
              取消
            </Button>
            <Button
              variant="tenant-primary"
              onClick={() => {
                if (!deleteTaskId) return;
                tenantProjectStore.deleteTask(project.id, deleteTaskId);
                setDeleteTaskId(null);
                toast.success("任务已删除");
              }}
            >
              删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ─── 项目自动化面板（弹窗）：列表 + 新建/编辑 ────────────────
function AutomationsTab({
  project,
  onGotoWorkflow,
}: {
  project: TenantProject;
  onGotoWorkflow: () => void;
}) {
  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<TenantAutomation | null>(null);

  const openCreate = () => {
    setEditing(null);
    setFormOpen(true);
  };
  const openEdit = (a: TenantAutomation) => {
    setEditing(a);
    setFormOpen(true);
  };

  const runNow = (id: string) => {
    tenantProjectStore.runAutomationNow(project.id, id);
    toast.success("已运行，任务已创建到看板");
  };

  // 无工作流时的引导：自动化必须绑一条工作流才能建
  if (project.pipelineTemplates.length === 0) {
    return (
      <div className="max-w-[720px] mx-auto py-16">
        <Empty>
          <EmptyHeader>
            <EmptyDescription>
              自动化需要绑定一条工作流来定义"要做什么"。请先去「工作流」Tab
              建一条。
            </EmptyDescription>
          </EmptyHeader>
          <Button
            variant="tenant-primary"
            size="sm"
            className="mt-3"
            onClick={onGotoWorkflow}
          >
            <Plus className="w-4 h-4" />
            去建工作流
          </Button>
        </Empty>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <AutomationList
        project={project}
        onCreate={openCreate}
        onEdit={openEdit}
        onRun={runNow}
      />
      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>{editing ? "编辑自动化" : "新建自动化"}</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6 max-h-[60vh] overflow-y-auto">
            <AutomationForm
              project={project}
              editing={editing}
              onCancel={() => setFormOpen(false)}
              onSaved={() => setFormOpen(false)}
            />
          </DialogBody>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function AutomationList({
  project,
  onCreate,
  onEdit,
  onRun,
}: {
  project: TenantProject;
  onCreate: () => void;
  onEdit: (a: TenantAutomation) => void;
  onRun: (id: string) => void;
}) {
  const list = project.automations;
  const cycleLabel = (a: TenantAutomation) => {
    if (a.schedule.type === "once")
      return `单次 · ${a.schedule.onceAt ?? "未设"}`;
    if (a.schedule.type === "interval")
      return `每隔 ${a.schedule.intervalHours ?? 1} 小时`;
    const t = a.schedule.time ?? "09:00";
    if (a.schedule.cycle === "weekly") {
      const w = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"][
        a.schedule.weekday ?? 1
      ];
      return `每${w} ${t}`;
    }
    return `每天 ${t}`;
  };
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <MetaText tone="secondary">
          共 {list.length} 条自动化，{list.filter(a => a.enabled).length}{" "}
          条启用中
        </MetaText>
        <Button variant="tenant-primary" size="sm" onClick={onCreate}>
          <Plus className="w-4 h-4" />
          新建自动化
        </Button>
      </div>
      {list.length === 0 ? (
        <Empty className="py-10">
          <EmptyHeader>
            <EmptyDescription>
              还没有自动化。新建一条，让 AI 定时为项目产出。
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="space-y-2">
          {list.map(a => {
            const wf = a.pipelineTemplateId
              ? project.pipelineTemplates.find(
                  t => t.id === a.pipelineTemplateId
                )
              : null;
            return (
              <div
                key={a.id}
                className="rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white p-3 space-y-2"
              >
                <div className="flex items-center gap-2">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <BodyMedium className="truncate">{a.name}</BodyMedium>
                      <StatusTag
                        variant={a.enabled ? "blue" : "gray"}
                        mode="soft"
                      >
                        {a.enabled ? "启用中" : "已暂停"}
                      </StatusTag>
                    </div>
                    <div className="text-xs text-[var(--text-weak)] mt-0.5 truncate">
                      {cycleLabel(a)}
                      {wf ? ` · 走「${wf.name}」` : " · 未关联工作流"}
                    </div>
                  </div>
                </div>
                {a.prompt && (
                  <MetaText tone="secondary" className="line-clamp-2">
                    {a.prompt}
                  </MetaText>
                )}
                <div className="flex items-center gap-2 pt-1">
                  <Button
                    variant="tenant-outline"
                    size="sm"
                    onClick={() => onRun(a.id)}
                  >
                    <Play className="w-3.5 h-3.5" />
                    立即运行
                  </Button>
                  <Button
                    variant="tenant-outline"
                    size="sm"
                    onClick={() =>
                      tenantProjectStore.toggleAutomation(
                        project.id,
                        a.id,
                        !a.enabled
                      )
                    }
                  >
                    {a.enabled ? "暂停" : "启用"}
                  </Button>
                  <Button
                    variant="tenant-outline"
                    size="sm"
                    onClick={() => onEdit(a)}
                  >
                    <Pencil className="w-3.5 h-3.5" />
                    编辑
                  </Button>
                  <Button
                    variant="tenant-outline"
                    size="sm"
                    className="text-[var(--text-danger)]"
                    onClick={() => {
                      tenantProjectStore.deleteAutomation(project.id, a.id);
                      toast.success("已删除");
                    }}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                    删除
                  </Button>
                  {a.nextRunAt && a.enabled && (
                    <span className="ml-auto text-xs text-[var(--text-weak)]">
                      下次：{new Date(a.nextRunAt).toLocaleString("zh-CN")}
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function AutomationForm({
  project,
  editing,
  onCancel,
  onSaved,
}: {
  project: TenantProject;
  editing: TenantAutomation | null;
  onCancel: () => void;
  onSaved: () => void;
}) {
  const [name, setName] = useState(editing?.name ?? "");
  const [prompt, setPrompt] = useState(editing?.prompt ?? "");
  const [scheduleType, setScheduleType] =
    useState<TenantAutomationScheduleType>(
      editing?.schedule.type ?? "periodic"
    );
  const [cycle, setCycle] = useState<"daily" | "weekly">(
    editing?.schedule.cycle ?? "daily"
  );
  const [weekday, setWeekday] = useState<number>(
    editing?.schedule.weekday ?? 1
  );
  const [time, setTime] = useState<string>(editing?.schedule.time ?? "09:00");
  const [intervalHours, setIntervalHours] = useState<number>(
    editing?.schedule.intervalHours ?? 6
  );
  const [onceAt, setOnceAt] = useState<string>(
    editing?.schedule.onceAt ??
      new Date(Date.now() + 3600_000).toISOString().slice(0, 16)
  );
  const [pipelineTemplateId, setPipelineTemplateId] = useState<string>(
    editing?.pipelineTemplateId ?? project.pipelineTemplates[0]?.id ?? ""
  );
  const [enabled, setEnabled] = useState<boolean>(editing?.enabled ?? true);

  const submit = () => {
    if (!name.trim()) {
      toast.error("请填写名称");
      return;
    }
    if (!pipelineTemplateId) {
      toast.error("请选择要触发的工作流");
      return;
    }
    const schedule: TenantAutomationSchedule =
      scheduleType === "periodic"
        ? {
            type: "periodic",
            cycle,
            weekday: cycle === "weekly" ? weekday : undefined,
            time,
          }
        : scheduleType === "interval"
          ? { type: "interval", intervalHours }
          : { type: "once", onceAt };
    const payload = {
      name: name.trim(),
      prompt: prompt.trim(),
      triggerKind: "schedule" as const,
      schedule,
      outputMode: "createTask" as const,
      pipelineTemplateId,
      enabled,
    };
    if (editing) {
      tenantProjectStore.updateAutomation(project.id, editing.id, payload);
      toast.success("已保存");
    } else {
      tenantProjectStore.createAutomation(project.id, payload);
      toast.success("已创建");
    }
    onSaved();
  };

  return (
    <div className="space-y-3">
      <div className="space-y-1.5">
        <Label>
          名称 <span className="text-[var(--text-danger)]">*</span>
        </Label>
        <Input
          value={name}
          onChange={e => setName(e.target.value)}
          placeholder="如：每日项目动态汇总"
        />
      </div>
      <div className="space-y-1.5">
        <Label>提示词（可选，作为额外指令补充到工作流）</Label>
        <textarea
          value={prompt}
          onChange={e => setPrompt(e.target.value)}
          placeholder="例如：汇总过去 24 小时项目动态并整理跟进事项"
          className="w-full min-h-20 rounded-[4px] border border-[var(--cp-border)] bg-white px-2 py-1.5 text-sm text-[var(--text-body)]"
        />
      </div>
      <div className="space-y-1.5">
        <Label>
          关联工作流 <span className="text-[var(--text-danger)]">*</span>
        </Label>
        <Select
          value={pipelineTemplateId}
          onValueChange={setPipelineTemplateId}
        >
          <SelectTrigger>
            <SelectValue placeholder="选择要触发的工作流" />
          </SelectTrigger>
          <SelectContent>
            {project.pipelineTemplates.map(t => (
              <SelectItem key={t.id} value={t.id}>
                {t.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <MetaText tone="weak" className="block">
          工作流每个节点的执行者由工作流自身配置（agent 或成员）。
        </MetaText>
      </div>
      <div className="space-y-1.5">
        <Label>调度类型</Label>
        <div className="flex items-center gap-1">
          {(
            [
              { id: "periodic", label: "周期" },
              { id: "interval", label: "按间隔" },
              { id: "once", label: "单次" },
            ] as const
          ).map(t => (
            <button
              key={t.id}
              type="button"
              onClick={() => setScheduleType(t.id)}
              className={`h-8 px-3 rounded-[4px] border text-xs ${
                scheduleType === t.id
                  ? "border-[var(--cp-brand-blue)] bg-[var(--bg-brand-selected)] text-[var(--text-brand)]"
                  : "border-[var(--cp-border)] bg-white text-[var(--text-secondary)]"
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
      </div>
      {scheduleType === "periodic" && (
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1.5">
            <Label>频率</Label>
            <Select
              value={cycle}
              onValueChange={v => setCycle(v as "daily" | "weekly")}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="daily">每天</SelectItem>
                <SelectItem value="weekly">每周</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <Label>时间</Label>
            <Input
              type="time"
              value={time}
              onChange={e => setTime(e.target.value)}
            />
          </div>
          {cycle === "weekly" && (
            <div className="space-y-1.5 col-span-2">
              <Label>星期</Label>
              <Select
                value={String(weekday)}
                onValueChange={v => setWeekday(Number(v))}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {["周日", "周一", "周二", "周三", "周四", "周五", "周六"].map(
                    (w, i) => (
                      <SelectItem key={i} value={String(i)}>
                        {w}
                      </SelectItem>
                    )
                  )}
                </SelectContent>
              </Select>
            </div>
          )}
        </div>
      )}
      {scheduleType === "interval" && (
        <div className="space-y-1.5">
          <Label>每隔多少小时执行一次</Label>
          <Input
            type="number"
            min={1}
            value={intervalHours}
            onChange={e => setIntervalHours(Number(e.target.value) || 1)}
          />
        </div>
      )}
      {scheduleType === "once" && (
        <div className="space-y-1.5">
          <Label>执行时间</Label>
          <Input
            type="datetime-local"
            value={onceAt.slice(0, 16)}
            onChange={e => setOnceAt(e.target.value)}
          />
        </div>
      )}
      <div className="flex items-center gap-2 pt-2">
        <label className="flex items-center gap-2 text-sm text-[var(--text-body)]">
          <input
            type="checkbox"
            checked={enabled}
            onChange={e => setEnabled(e.target.checked)}
          />
          立即启用
        </label>
        <div className="ml-auto flex items-center gap-2">
          <Button variant="tenant-outline" onClick={onCancel}>
            取消
          </Button>
          <Button variant="tenant-primary" onClick={submit}>
            {editing ? "保存" : "创建"}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ─── 项目设置弹窗（改名/描述/目标 + 管理员配置成员编辑权限） ─────
function ProjectSettingsDialog({
  open,
  onOpenChange,
  project,
  canManagePermission = false,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  project: TenantProject;
  canManagePermission?: boolean;
}) {
  const [name, setName] = useState(project.name);
  const [description, setDescription] = useState(project.description);
  const [goal, setGoal] = useState(project.goal);

  useEffect(() => {
    if (!open) return;
    setName(project.name);
    setDescription(project.description);
    setGoal(project.goal);
  }, [open, project]);

  const submit = () => {
    const nextName = name.trim() || project.name;
    tenantProjectStore.updateProjectInfo(project.id, {
      name: nextName,
      description: description.trim(),
      goal: goal.trim(),
    });
    const sharedProject = groupStore.getById(project.id);
    if (sharedProject && sharedProject.name !== nextName) {
      groupStore.update(project.id, current => ({
        ...current,
        name: nextName,
      }));
    }
    toast.success("项目已更新");
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>编辑项目</DialogTitle>
          <DialogDescription>
            修改项目基本信息{canManagePermission ? "与成员编辑项目的权限" : ""}
            。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="px-6 space-y-3 max-h-[60vh] overflow-y-auto">
          <div className="space-y-1.5">
            <Label>项目名称</Label>
            <Input
              value={name}
              onChange={e => setName(e.target.value)}
              maxLength={40}
            />
          </div>
          <div className="space-y-1.5">
            <Label>项目描述</Label>
            <Textarea
              value={description}
              onChange={e => setDescription(e.target.value)}
              className="min-h-16"
            />
          </div>
          <div className="space-y-1.5">
            <Label>项目目标</Label>
            <Textarea
              value={goal}
              onChange={e => setGoal(e.target.value)}
              className="min-h-16"
            />
          </div>
          {canManagePermission && (
            <div className="flex items-start justify-between gap-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 py-2.5">
              <div className="min-w-0">
                <div className="flex items-center gap-1.5 text-sm font-medium text-[var(--text-body)]">
                  <ShieldCheck className="w-4 h-4 text-[var(--cp-brand-blue)]" />
                  允许编辑项目
                </div>
                <p className="text-xs text-[var(--text-muted)] mt-0.5 mb-0">
                  开启后项目成员可修改项目信息及资产，关闭后项目保持只读。
                </p>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={project.allowMemberEdit}
                onClick={() =>
                  tenantProjectStore.toggleAllowMemberEdit(project.id)
                }
                className={`shrink-0 relative w-11 h-6 rounded-full transition-colors ${
                  project.allowMemberEdit
                    ? "bg-[var(--cp-brand-blue)]"
                    : "bg-[var(--color-gray-300,#cbd5e1)]"
                }`}
              >
                <span
                  className={`absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow transition-transform ${
                    project.allowMemberEdit ? "translate-x-5" : ""
                  }`}
                />
              </button>
            </div>
          )}
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button variant="tenant-primary" onClick={submit}>
            保存
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 邀请成员弹窗（复用管控端统一选人交互） ──────────────
function InviteMemberDialog({
  open,
  onOpenChange,
  project,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  project: TenantProject;
}) {
  const [directoryUsers, setDirectoryUsers] = useState<UserOrg[]>(() =>
    userStore.getAll()
  );
  const [directoryGroups, setDirectoryGroups] = useState<UserGroup[]>(() =>
    groupStore.getAll()
  );

  useEffect(
    () => userStore.subscribe(() => setDirectoryUsers(userStore.getAll())),
    []
  );
  useEffect(
    () => groupStore.subscribe(() => setDirectoryGroups(groupStore.getAll())),
    []
  );

  const usersWithProjectMembership = useMemo(() => {
    const memberIds = new Set(project.members.map(member => member.userId));
    return directoryUsers.map(user =>
      memberIds.has(user.userId) && !user.groupIds.includes(project.id)
        ? { ...user, groupIds: [...user.groupIds, project.id] }
        : user
    );
  }, [directoryUsers, project.id, project.members]);

  const groups = useMemo(() => {
    if (directoryGroups.some(group => group.id === project.id)) {
      return directoryGroups;
    }
    return [
      ...directoryGroups,
      {
        id: project.id,
        name: project.name,
        parentId: null,
        source: "project" as const,
        readonly: false,
        createdAt: new Date().toISOString(),
      },
    ];
  }, [directoryGroups, project.id, project.name]);

  return (
    <AddUsersToGroupDialog
      open={open}
      onOpenChange={onOpenChange}
      nodeName={project.name}
      nodeId={project.id}
      allUsers={usersWithProjectMembership}
      groups={groups}
      showDept
      hasOneid
      term="项目"
      onConfirm={userIds => {
        if (!groupStore.getById(project.id)) {
          groupStore.add({
            id: project.id,
            name: project.name,
            parentId: null,
            source: "project",
            readonly: false,
            createdAt: new Date().toISOString(),
          });
        }
        addUsersToGroup(userIds, project.id);
        userIds.forEach(userId => {
          const user = directoryUsers.find(item => item.userId === userId);
          if (!user) return;
          tenantProjectStore.addProjectMember(project.id, {
            userId: user.userId,
            displayName: user.displayName,
            role: user.role === "admin" ? "管理员" : "成员",
            admin: false,
          });
        });
        toast.success(`已添加 ${userIds.length} 名项目成员`);
      }}
    />
  );
}

function CreateProjectDialog({
  open,
  onOpenChange,
  onCreate,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (input: {
    name: string;
    description: string;
    goal: string;
    allowMemberEdit: boolean;
  }) => void;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [goal, setGoal] = useState("");
  const [allowMemberEdit, setAllowMemberEdit] = useState(true);

  useEffect(() => {
    if (!open) return;
    setName("");
    setDescription("");
    setGoal("");
    setAllowMemberEdit(true);
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>新建项目</DialogTitle>
          <DialogDescription>
            填写项目基本信息并配置成员编辑项目的权限。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="px-6 space-y-3 max-h-[60vh] overflow-y-auto">
          <div className="space-y-1.5">
            <Label htmlFor="new-project-name">项目名称</Label>
            <Input
              id="new-project-name"
              value={name}
              onChange={event => setName(event.target.value)}
              placeholder="输入项目名称"
              maxLength={40}
              autoFocus
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="new-project-description">项目描述</Label>
            <Textarea
              id="new-project-description"
              value={description}
              onChange={event => setDescription(event.target.value)}
              placeholder="说明项目背景与协作范围"
              className="min-h-16"
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="new-project-goal">项目目标</Label>
            <Textarea
              id="new-project-goal"
              value={goal}
              onChange={event => setGoal(event.target.value)}
              placeholder="填写项目希望达成的目标"
              className="min-h-16"
            />
          </div>
          <div className="flex items-start justify-between gap-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 py-2.5">
            <div className="min-w-0">
              <div className="flex items-center gap-1.5 text-sm font-medium text-[var(--text-body)]">
                <ShieldCheck className="w-4 h-4 text-[var(--cp-brand-blue)]" />
                允许编辑项目
              </div>
              <p className="text-xs text-[var(--text-muted)] mt-0.5 mb-0">
                开启后项目成员可修改项目信息及资产，关闭后项目保持只读。
              </p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={allowMemberEdit}
              onClick={() => setAllowMemberEdit(value => !value)}
              className={`shrink-0 relative w-11 h-6 rounded-full transition-colors ${
                allowMemberEdit
                  ? "bg-[var(--cp-brand-blue)]"
                  : "bg-[var(--color-gray-300,#cbd5e1)]"
              }`}
            >
              <span
                className={`absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow transition-transform ${
                  allowMemberEdit ? "translate-x-5" : ""
                }`}
              />
            </button>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="tenant-primary"
            disabled={!name.trim()}
            onClick={() => {
              onCreate({
                name: name.trim(),
                description: description.trim(),
                goal: goal.trim(),
                allowMemberEdit,
              });
              onOpenChange(false);
            }}
          >
            创建项目
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

const EXTERNAL_AGENT_NAMES = [
  "CodeBuddy",
  "WorkBuddy",
  "Claude Code",
  "Codex",
  "iMate",
  "KnotBot",
] as const;

function buildExternalAgentAccessPrompt() {
  return `请帮我配置 ClawPro 外部 Agent 接入，将外部 Agent（如 CodeBuddy / WorkBuddy / Claude Code / Codex / iMate / KnotBot）接入企业管理。

接入要求：
1. 先识别当前可用的接入方式，以及底层 Agent 类型（OpenClaw / Hermes 等）。
2. 检查系统环境、企业授权、接入凭证，以及插件目录、扩展管理入口或云端授权入口是否可用。
3. 按 Agent 提示完成企业登录和授权绑定；如缺少企业内部安装包、下载地址或接入凭证，请明确提示我提供，不要使用不可信来源。
4. 完成授权绑定后，检查外部 Agent 是否能向 ClawPro / Hatchery 同步基本信息、接入状态和已安装企业 Skill。

完成后请返回：
- 接入配置状态
- 插件版本、安装路径或外部 Agent 接入信息
- 当前接入状态
- 仍需我手动处理的事项`;
}

function ExternalAgentAccessDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const copyPrompt = async () => {
    try {
      await navigator.clipboard.writeText(buildExternalAgentAccessPrompt());
      toast.success("已复制接入 Prompt");
    } catch {
      toast.error("复制失败，请手动复制 Prompt");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>接入外部 Agent</DialogTitle>
          <DialogDescription>
            支持将 CodeBuddy、WorkBuddy、Claude Code、Codex、iMate、KnotBot
            等外部 Agent 接入企业管理，统一同步企业 Skill 和规范。
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="px-6 space-y-4">
          <div className="flex items-start gap-2 rounded-[var(--radius-md)] border border-[var(--cp-brand-blue)] bg-[var(--alert-info-bg)] px-3 py-2.5">
            <Info className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[var(--cp-brand-blue)]" />
            <MetaText tone="secondary" className="leading-5">
              复制接入 Prompt，在外部 Agent 的对话或授权入口中执行，即可完成
              ClawPro 授权绑定和资源同步。
            </MetaText>
          </div>
          <div>
            <BodyMedium className="mb-2 block">支持的接入环境</BodyMedium>
            <div className="flex flex-wrap gap-2">
              {[
                { label: "macOS", Icon: Apple },
                { label: "Windows", Icon: Monitor },
              ].map(({ label, Icon }) => (
                <span
                  key={label}
                  className="inline-flex items-center gap-1.5 rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-white px-3 py-1.5 text-xs text-[var(--text-body)]"
                >
                  <Icon className="h-3.5 w-3.5 text-[var(--text-weak)]" />
                  {label}
                </span>
              ))}
            </div>
          </div>
          <div>
            <BodyMedium className="mb-2 block">支持的外部 Agent</BodyMedium>
            <div className="flex flex-wrap gap-2">
              {EXTERNAL_AGENT_NAMES.map(name => (
                <span
                  key={name}
                  className="inline-flex items-center gap-1.5 rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-white px-3 py-1.5 text-xs text-[var(--text-body)]"
                >
                  <Monitor className="h-3.5 w-3.5 text-[var(--text-weak)]" />
                  {name}
                </span>
              ))}
            </div>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="tenant-outline" onClick={() => onOpenChange(false)}>
            关闭
          </Button>
          <Button variant="tenant-primary" onClick={copyPrompt}>
            <Copy className="h-4 w-4" />
            复制接入 Prompt
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ProjectInstancesTab({ project }: { project: TenantProject }) {
  return (
    <div className="space-y-3">
      <MetaText tone="secondary">
        共 {project.agents.length} 个项目实例
      </MetaText>
      {project.agents.length === 0 ? (
        <Empty className="py-16">
          <EmptyHeader>
            <EmptyDescription>
              该项目暂无 Agent 实例，点击“接入外部 Agent”完成接入。
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="overflow-hidden rounded-[var(--radius-md)] border border-[var(--cp-border)] bg-[var(--cp-surface)]">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>实例名称 / ID</TableHead>
                <TableHead>当前状态</TableHead>
                <TableHead>用户 ID</TableHead>
                <TableHead>项目</TableHead>
                <TableHead>Agent 类型</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {project.agents.map(agent => (
                <TableRow key={agent.id}>
                  <TableCell>
                    <div className="max-w-[220px]">
                      <BodyMedium className="block truncate">
                        {agent.name}
                      </BodyMedium>
                      <MetaText tone="secondary" className="block truncate">
                        {agent.id}
                      </MetaText>
                    </div>
                  </TableCell>
                  <TableCell>
                    <StatusTag
                      mode="text"
                      variant={agent.status === "online" ? "green" : "gray"}
                    >
                      {agent.status === "online" ? "已接入" : "离线"}
                    </StatusTag>
                  </TableCell>
                  <TableCell>
                    <MetaText tone="secondary">{agent.ownerId}</MetaText>
                  </TableCell>
                  <TableCell>
                    <MetaText tone="secondary">{project.name}</MetaText>
                  </TableCell>
                  <TableCell>
                    <MetaText tone="secondary">
                      {agent.location === "local" ? "本地 Agent" : "云端 Agent"}{" "}
                      · {agent.platform}
                    </MetaText>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}

function ProjectMembersTab({ project }: { project: TenantProject }) {
  const [inviteOpen, setInviteOpen] = useState(false);
  const [expandedMemberIds, setExpandedMemberIds] = useState<Set<string>>(
    () => new Set(project.members.slice(0, 1).map(member => member.userId))
  );
  const [selectedAgent, setSelectedAgent] = useState<TenantAgent | null>(null);
  const [liveRuntimeAgents, setLiveRuntimeAgents] = useState<TenantAgent[]>([]);
  const [directoryUsers, setDirectoryUsers] = useState<UserOrg[]>(() =>
    userStore.getAll()
  );
  const [directoryGroups, setDirectoryGroups] = useState<UserGroup[]>(() =>
    groupStore.getAll()
  );

  useEffect(
    () => userStore.subscribe(() => setDirectoryUsers(userStore.getAll())),
    []
  );
  useEffect(
    () => groupStore.subscribe(() => setDirectoryGroups(groupStore.getAll())),
    []
  );

  const usersWithProjectMembership = useMemo(() => {
    const memberIds = new Set(project.members.map(member => member.userId));
    return directoryUsers.map(user =>
      memberIds.has(user.userId) && !user.groupIds.includes(project.id)
        ? { ...user, groupIds: [...user.groupIds, project.id] }
        : user
    );
  }, [directoryUsers, project.id, project.members]);

  useEffect(() => {
    setExpandedMemberIds(
      new Set(project.members.slice(0, 1).map(member => member.userId))
    );
    setSelectedAgent(null);
  }, [project.id]);

  useEffect(() => {
    let active = true;
    const owner =
      project.members.find(member => member.userId === "alice@acompany.com") ??
      project.members[0];
    listWorkflowRuntimeAgents()
      .then(agents => {
        if (!active || !owner) return;
        setLiveRuntimeAgents(
          agents.map(agent => ({
            id: `live-${agent.id}`,
            name: agent.name,
            platform: agent.platform,
            location: agent.location,
            ownerId: owner.userId,
            owner: owner.displayName,
            role: "真实 Agent Runtime",
            status: agent.status,
            authorization: agent.detail,
            kind: "personal",
            runtimeId: agent.runtimeId,
            deviceId: agent.deviceId,
            targetAgentId: agent.targetAgentId,
            runtimeDetail: agent.detail,
            runtimeCapabilities: agent.capabilities,
            runtimeMissingCapabilities: agent.missingCapabilities,
          }))
        );
      })
      .catch(() => {
        if (active) setLiveRuntimeAgents([]);
      });
    return () => {
      active = false;
    };
  }, [project.id, project.members]);

  const agentsByMember = useMemo(() => {
    const livePlatforms = new Set(
      liveRuntimeAgents.map(agent => `${agent.ownerId}:${agent.platform}`)
    );
    const projectAgents = project.agents.filter(agent => {
      const hasLiveReplacement = livePlatforms.has(
        `${agent.ownerId}:${agent.platform}`
      );
      return !(
        hasLiveReplacement &&
        (agent.runtimeId || agent.name.includes("真实 Runtime"))
      );
    });
    return [...projectAgents, ...liveRuntimeAgents];
  }, [liveRuntimeAgents, project.agents]);

  const agentAssetGroups = useMemo(() => {
    const introSkill = getProjectIntroSkill(project);
    const toItems = (category: AssetCategory) =>
      (project.assets[category] ?? []).map(refId => {
        const display = getAssetItemDisplay(category, refId);
        return {
          id: `${category}-${refId}`,
          name: display.name,
          meta: display.exists
            ? getAssetVersionLabel(category, display.version)
            : "资产已移除",
        };
      });

    return [
      {
        key: "skill",
        label: "Skill",
        icon: Sparkles,
        items: [
          {
            id: introSkill.id,
            name: introSkill.name,
            meta: `v${introSkill.version} · 项目自动生成`,
          },
          ...toItems("publicSkill"),
          ...toItems("enterpriseSkill"),
        ],
      },
      {
        key: "rules",
        label: "Rules",
        icon: FileText,
        items: toItems("enterpriseStandard"),
      },
      {
        key: "mcp",
        label: "MCP",
        icon: Boxes,
        items: toItems("enterpriseMcp"),
      },
    ];
  }, [project]);

  const assetCounts = useMemo(
    () =>
      Object.fromEntries(
        agentAssetGroups.map(group => [group.key, group.items.length])
      ) as Record<"skill" | "rules" | "mcp", number>,
    [agentAssetGroups]
  );

  const toggleMember = (userId: string) => {
    setExpandedMemberIds(current => {
      const next = new Set(current);
      if (next.has(userId)) next.delete(userId);
      else next.add(userId);
      return next;
    });
  };

  return (
    <>
      <div className="flex items-center justify-between gap-3">
        <div>
          <BodyMedium>项目成员</BodyMedium>
          <MetaText className="mt-0.5 block" tone="weak">
            共 {project.members.length} 人
          </MetaText>
        </div>
        <Button
          variant="tenant-primary"
          size="sm"
          onClick={() => {
            if (!groupStore.getById(project.id)) {
              groupStore.add({
                id: project.id,
                name: project.name,
                parentId: null,
                source: "project",
                readonly: false,
                createdAt: new Date().toISOString(),
              });
            }
            setInviteOpen(true);
          }}
        >
          <Plus className="h-4 w-4" />
          邀请成员
        </Button>
      </div>

      <div className="mt-4 space-y-3">
        {project.members.map(member => {
          const isCreator = member.userId === "alice@acompany.com";
          const memberAgents = agentsByMember.filter(
            agent => agent.ownerId === member.userId
          );
          const expanded = expandedMemberIds.has(member.userId);
          return (
            <div
              key={member.userId}
              className="overflow-hidden rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-white"
            >
              <div className="flex items-center gap-3 px-4 py-3">
                <button
                  type="button"
                  className="flex min-w-0 flex-1 items-center gap-3 text-left"
                  onClick={() => toggleMember(member.userId)}
                  aria-expanded={expanded}
                >
                  {expanded ? (
                    <ChevronDown className="h-4 w-4 shrink-0 text-[var(--text-weak)]" />
                  ) : (
                    <ChevronRight className="h-4 w-4 shrink-0 text-[var(--text-weak)]" />
                  )}
                  <span className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[var(--color-gray-100)] text-sm font-medium text-[var(--text-secondary)]">
                    {member.displayName.slice(0, 1).toUpperCase()}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-1.5">
                      <BodyMedium>{member.displayName}</BodyMedium>
                    </div>
                    <MetaText className="mt-0.5 block" tone="weak">
                      {member.userId} · {member.role} · {memberAgents.length} 个
                      Agent
                    </MetaText>
                  </div>
                </button>
                {isCreator ? (
                  <MetaText tone="weak">创建人</MetaText>
                ) : (
                  <Button
                    variant="tenant-outline"
                    size="sm"
                    onClick={() => {
                      removeUserFromGroup(member.userId, project.id);
                      tenantProjectStore.removeProjectMember(
                        project.id,
                        member.userId
                      );
                      toast.success(`${member.displayName} 已移出项目`);
                    }}
                  >
                    移除
                  </Button>
                )}
              </div>

              {expanded && (
                <div className="border-t border-[var(--cp-border)] bg-[var(--cp-surface)] px-4 py-3 pl-12">
                  {memberAgents.length === 0 ? (
                    <div className="rounded-[4px] border border-dashed border-[var(--cp-border)] bg-white px-4 py-6 text-center">
                      <MetaText tone="weak">
                        该成员暂无已授权到项目的 Agent
                      </MetaText>
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {memberAgents.map(agent => (
                        <button
                          key={agent.id}
                          type="button"
                          onClick={() => setSelectedAgent(agent)}
                          className="flex w-full items-center gap-3 rounded-[4px] border border-[var(--cp-border)] bg-white px-3 py-3 text-left transition-colors hover:bg-[var(--cp-surface)]"
                        >
                          <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-[var(--color-gray-100)] text-[var(--text-secondary)]">
                            <Bot className="h-4 w-4" />
                          </span>
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <BodyMedium className="truncate">
                                {agent.name}
                              </BodyMedium>
                              <StatusTag
                                mode="text"
                                variant={
                                  agent.status === "online" ? "green" : "gray"
                                }
                              >
                                {agent.status === "online" ? "已接入" : "离线"}
                              </StatusTag>
                            </div>
                            <MetaText className="mt-0.5 block" tone="weak">
                              {agent.location === "local"
                                ? "本地 Agent"
                                : "云端 Agent"}{" "}
                              · {agent.platform}
                            </MetaText>
                          </div>
                          <div className="flex shrink-0 items-center gap-1.5">
                            <StatusTag variant="gray" mode="soft">
                              {assetCounts.skill} 个 Skill
                            </StatusTag>
                            <StatusTag variant="gray" mode="soft">
                              {assetCounts.rules} 个 Rules
                            </StatusTag>
                            <StatusTag variant="gray" mode="soft">
                              {assetCounts.mcp} 个 MCP
                            </StatusTag>
                          </div>
                          <ChevronRight className="h-4 w-4 shrink-0 text-[var(--text-weak)]" />
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      <AddUsersToGroupDialog
        open={inviteOpen}
        onOpenChange={setInviteOpen}
        nodeName={project.name}
        nodeId={project.id}
        allUsers={usersWithProjectMembership}
        groups={directoryGroups}
        showDept
        hasOneid
        term="项目"
        onConfirm={userIds => {
          addUsersToGroup(userIds, project.id);
          userIds.forEach(userId => {
            const user = directoryUsers.find(item => item.userId === userId);
            if (!user) return;
            tenantProjectStore.addProjectMember(project.id, {
              userId: user.userId,
              displayName: user.displayName,
              admin: false,
            });
          });
          toast.success(`已邀请 ${userIds.length} 名成员`);
        }}
      />

      <Sheet
        open={Boolean(selectedAgent)}
        onOpenChange={open => {
          if (!open) setSelectedAgent(null);
        }}
      >
        <SheetContent
          side="right"
          className="w-[520px] sm:max-w-[520px] flex flex-col p-0"
        >
          <SheetHeader className="border-b border-[var(--cp-border)] px-6 py-4">
            <SheetTitle>Agent 资产详情</SheetTitle>
          </SheetHeader>
          {selectedAgent && (
            <div className="flex-1 space-y-5 overflow-y-auto px-6 py-5">
              <div className="rounded-[var(--radius-card)] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-4">
                <div className="flex items-start gap-3">
                  <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-white text-[var(--text-secondary)]">
                    <Bot className="h-5 w-5" />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <BodyMedium>{selectedAgent.name}</BodyMedium>
                      <StatusTag
                        mode="text"
                        variant={
                          selectedAgent.status === "online" ? "green" : "gray"
                        }
                      >
                        {selectedAgent.status === "online" ? "已接入" : "离线"}
                      </StatusTag>
                    </div>
                    <MetaText className="mt-1 block" tone="secondary">
                      {selectedAgent.owner} ·{" "}
                      {selectedAgent.location === "local"
                        ? "本地 Agent"
                        : "云端 Agent"}{" "}
                      · {selectedAgent.platform}
                    </MetaText>
                    <MetaText className="mt-1 block" tone="weak">
                      {selectedAgent.authorization}
                    </MetaText>
                    {selectedAgent.runtimeId && (
                      <MetaText className="mt-1 block" tone="weak">
                        Runtime：{selectedAgent.runtimeId}
                        {selectedAgent.deviceId
                          ? ` · 设备：${selectedAgent.deviceId}`
                          : ""}
                      </MetaText>
                    )}
                  </div>
                </div>
              </div>

              <div>
                <BodyMedium>已同步项目资产</BodyMedium>
                <MetaText className="mt-1 block" tone="weak">
                  项目资产更新后，会按项目的同步策略下发到该 Agent。
                </MetaText>
                <div className="mt-3 grid grid-cols-3 gap-2">
                  {agentAssetGroups.map(group => {
                    const Icon = group.icon;
                    return (
                      <div
                        key={group.key}
                        className="rounded-[4px] border border-[var(--cp-border)] bg-white px-3 py-3"
                      >
                        <div className="flex items-center gap-1.5 text-[var(--text-secondary)]">
                          <Icon className="h-3.5 w-3.5" />
                          <MetaText tone="secondary">{group.label}</MetaText>
                        </div>
                        <div className="mt-1 text-xl font-semibold tabular-nums text-[var(--text-title)]">
                          {group.items.length}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>

              <div className="space-y-4">
                {agentAssetGroups.map(group => {
                  const Icon = group.icon;
                  return (
                    <section key={group.key}>
                      <div className="mb-2 flex items-center gap-2">
                        <Icon className="h-4 w-4 text-[var(--text-secondary)]" />
                        <BodyMedium>{group.label}</BodyMedium>
                        <MetaText tone="weak">{group.items.length} 项</MetaText>
                      </div>
                      {group.items.length === 0 ? (
                        <div className="rounded-[4px] border border-dashed border-[var(--cp-border)] px-3 py-4 text-center">
                          <MetaText tone="weak">暂无该类资产</MetaText>
                        </div>
                      ) : (
                        <div className="overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-white">
                          {group.items.map((item, itemIndex) => (
                            <div
                              key={item.id}
                              className={`flex items-center justify-between gap-3 px-3 py-2.5 ${
                                itemIndex > 0
                                  ? "border-t border-[var(--cp-border)]"
                                  : ""
                              }`}
                            >
                              <BodyMedium className="min-w-0 truncate">
                                {item.name}
                              </BodyMedium>
                              <MetaText className="shrink-0" tone="weak">
                                {item.meta}
                              </MetaText>
                            </div>
                          ))}
                        </div>
                      )}
                    </section>
                  );
                })}
              </div>
            </div>
          )}
        </SheetContent>
      </Sheet>
    </>
  );
}

// ─── 右侧工作台面板 ─────────────────────────────────────
type PanelTab = "overview" | "tasks" | "workflow" | "members" | "assets";

function ProjectPanel({
  project,
  isAdmin,
  onOpenTask,
}: {
  project: TenantProject;
  isAdmin: boolean;
  onOpenTask: (taskId: string) => void;
}) {
  const [tab, setTab] = useState<PanelTab>("tasks");
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<TenantProjectAssets>(project.assets);
  const [addOpen, setAddOpen] = useState(false);
  const [taskCreateSignal, setTaskCreateSignal] = useState(0);
  const [workflowCreateSignal, setWorkflowCreateSignal] = useState(0);
  const [automationOpen, setAutomationOpen] = useState(false);
  const [projectEditOpen, setProjectEditOpen] = useState(false);
  const [inviteMemberOpen, setInviteMemberOpen] = useState(false);
  const [introSkillOpen, setIntroSkillOpen] = useState(false);
  const [assetRecordsOpen, setAssetRecordsOpen] = useState(false);

  const canEditProject = isAdmin || project.allowMemberEdit;
  const canEdit = canEditProject;
  const activeAssets = editing ? draft : project.assets;
  const assetUpdateRecords = project.assetUpdateRecords ?? [
    {
      id: `tenant-asset-record-${project.id}-current`,
      groupId: project.id,
      version: project.assetVersion,
      tagKind: "manual" as const,
      sections: [
        {
          title: `当前版本包含 ${assetTotal(project.assets)} 项资产`,
          items: ASSET_CATEGORY_ORDER.flatMap(category =>
            (project.assets[category] ?? []).map(
              refId =>
                `${ASSET_CATEGORY_MAP[category].label}：${getAssetItemDisplay(category, refId).name}`
            )
          ),
        },
      ],
      operator: "项目管理员",
      createdAt: project.updatedAt,
    },
  ];

  const enterEdit = () => {
    setDraft(project.assets);
    setEditing(true);
  };
  const cancelEdit = () => {
    setDraft(project.assets);
    setEditing(false);
  };
  const save = () => {
    tenantProjectStore.saveAssets(project.id, draft);
    setEditing(false);
    toast.success("项目资产已保存");
  };
  const removeItem = (category: AssetCategory, refId: string) => {
    setDraft(prev => ({
      ...prev,
      [category]: (prev[category] ?? []).filter(id => id !== refId),
    }));
  };

  const TABS: { id: PanelTab; label: string; icon: typeof Boxes }[] = [
    { id: "overview", label: "总览", icon: LayoutGrid },
    { id: "tasks", label: "任务看板", icon: ClipboardList },
    { id: "workflow", label: "工作流", icon: Workflow },
    { id: "members", label: "项目成员", icon: Users },
    { id: "assets", label: "资产与沉淀", icon: Boxes },
  ];

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* 头部：项目名 · N人（HoverCard 成员，对齐管控端） */}
      <div className="flex items-center justify-between gap-3 px-6 py-4 border-b border-[var(--cp-border)]">
        <div className="min-w-0">
          <div className="flex items-center gap-2.5 flex-wrap">
            <h2 className="text-lg font-semibold text-[var(--text-title)] truncate m-0">
              {project.name}
            </h2>
            <div className="flex items-center gap-1 text-xs text-[var(--text-muted)] tabular-nums">
              <span aria-hidden="true">·</span>
              <HoverCard openDelay={80} closeDelay={120}>
                <HoverCardTrigger asChild>
                  <button
                    type="button"
                    className="cursor-default border-b border-dashed border-[var(--text-weak)] leading-tight pb-px focus-visible:outline-none"
                  >
                    {project.members.length} 人
                  </button>
                </HoverCardTrigger>
                <HoverCardContent
                  align="start"
                  sideOffset={6}
                  className="w-[480px] rounded-[4px] p-3"
                >
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <MetaText tone="secondary">
                      项目成员（{project.members.length}）
                    </MetaText>
                  </div>
                  <div className="flex flex-wrap gap-1.5 max-h-[300px] overflow-y-auto -mr-1 pr-1">
                    <MemberChips
                      members={project.members}
                      onRemove={
                        canEditProject
                          ? m => {
                              tenantProjectStore.removeProjectMember(
                                project.id,
                                m.userId
                              );
                              toast.success(`${m.displayName} 已移出项目`);
                            }
                          : undefined
                      }
                    />
                    {canEditProject && (
                      <button
                        type="button"
                        onClick={() => setInviteMemberOpen(true)}
                        className="inline-flex items-center gap-1 h-6 px-2 rounded-[4px] border border-dashed border-[var(--cp-border-control)] text-xs text-[var(--text-secondary)] hover:border-[var(--cp-brand-blue)] hover:text-[var(--text-brand)] transition-colors"
                      >
                        <Plus className="w-3 h-3" />
                        添加
                      </button>
                    )}
                  </div>
                </HoverCardContent>
              </HoverCard>
            </div>
            {canEdit ? (
              <span className="inline-flex items-center gap-1 h-5 px-2 rounded-[4px] bg-[var(--alert-success-bg)] text-xs text-[var(--cp-text-success)]">
                <Pencil className="w-3 h-3" />
                可编辑项目
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 h-5 px-2 rounded-[4px] bg-[var(--color-gray-100)] text-xs text-[var(--text-muted)]">
                <Lock className="w-3 h-3" />
                项目只读
              </span>
            )}
          </div>
          <MetaText
            tone="secondary"
            className="mt-0.5 block truncate"
            title={project.description}
          >
            {project.description}
          </MetaText>
        </div>
        <div className="shrink-0">
          {canEditProject && (
            <button
              type="button"
              onClick={() => setProjectEditOpen(true)}
              className="inline-flex items-center gap-1.5 h-8 px-3 rounded-[var(--radius-md)] text-sm text-[var(--text-secondary)] hover:bg-[var(--cp-surface)] hover:text-[var(--text-title)] transition-colors border border-[var(--cp-border)]"
            >
              <Pencil className="w-3.5 h-3.5" />
              编辑项目
            </button>
          )}
        </div>
      </div>

      {/* Tab 栏 + 右侧按当前 Tab 上下文动作 */}
      <div className="px-6 pt-4 flex items-center justify-between gap-3">
        <SegmentGroup>
          {TABS.map(t => (
            <SegmentOption
              key={t.id}
              className={`gap-1.5 ${editing && t.id !== "assets" ? "opacity-50 cursor-not-allowed" : ""}`}
              active={tab === t.id}
              onClick={() => {
                if (editing && t.id !== "assets") return;
                setTab(t.id);
              }}
            >
              <t.icon className="w-4 h-4" />
              {t.label}
            </SegmentOption>
          ))}
        </SegmentGroup>
        <div className="flex items-center gap-2">
          {tab === "tasks" && (
            <>
              <Button
                variant="tenant-primary"
                size="sm"
                onClick={() => setTaskCreateSignal(s => s + 1)}
              >
                <Plus className="w-4 h-4" />
                新建任务
              </Button>
              {project.pipelineTemplates.length > 0 && (
                <Button
                  variant="tenant-outline"
                  size="sm"
                  onClick={() => setAutomationOpen(true)}
                >
                  <Play className="w-4 h-4" />
                  自动化
                  {project.automations.length > 0 && (
                    <span className="inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full bg-[var(--cp-brand-blue)] text-white text-xs tabular-nums">
                      {project.automations.length}
                    </span>
                  )}
                </Button>
              )}
            </>
          )}
          {tab === "workflow" && (
            <Button
              variant="tenant-primary"
              size="sm"
              onClick={() => setWorkflowCreateSignal(s => s + 1)}
            >
              <Plus className="w-4 h-4" />
              新建工作流
            </Button>
          )}
        </div>
      </div>

      {/* Tab 副标题 */}
      <div className="px-6 pt-1">
        <MetaText tone="weak" className="text-xs">
          {tab === "overview" && "Agent 自动化、任务进展与资源协作关系"}
          {tab === "tasks" && "管理项目任务，支持搜索、新建和自动化触发"}
          {tab === "workflow" &&
            "定义项目标准工作流，支持可视化编辑和 JSON 导入"}
          {tab === "members" && "管理项目成员、成员角色和 Agent 参与权限"}
          {tab === "assets" && "统一管理项目资产和 Agent 沉淀的项目经验"}
        </MetaText>
      </div>

      {/* 内容 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-6 py-4">
        {tab === "overview" && (
          <OverviewTab
            project={project}
            onGotoTasks={() => setTab("tasks")}
            onGotoWorkflow={() => setTab("workflow")}
            onGotoAssets={() => setTab("assets")}
          />
        )}
        {tab === "tasks" && (
          <TaskTab
            project={project}
            onOpenTask={onOpenTask}
            createSignal={taskCreateSignal}
            onGotoWorkflow={() => setTab("workflow")}
          />
        )}
        {tab === "workflow" && (
          <WorkflowTab
            project={project}
            currentUser={CURRENT_USER}
            createSignal={workflowCreateSignal}
          />
        )}

        {tab === "members" && <ProjectMembersTab project={project} />}

        {tab === "assets" && (
          <div className="space-y-4">
            {/* 版本卡 */}
            <div className="flex items-center justify-between gap-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-4 py-3">
              <div className="flex items-center gap-2">
                <MetaText tone="secondary">当前版本</MetaText>
                <StatusTag variant="gray" mode="soft">
                  v{project.assetVersion}
                </StatusTag>
              </div>
              <Button
                variant="tenant-outline"
                size="sm"
                onClick={() => setAssetRecordsOpen(true)}
              >
                <History className="w-4 h-4" />
                查看更新记录
              </Button>
            </div>

            {/* 「允许编辑项目」由管理员在右上角“编辑项目”弹窗中配置 */}

            {/* 资产配置卡 */}
            <div className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)]">
              <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-[var(--cp-border)]">
                <BodyMedium>资产配置</BodyMedium>
                <div className="flex items-center gap-2 shrink-0">
                  {!canEdit ? (
                    <span className="inline-flex items-center gap-1 text-xs text-[var(--text-muted)]">
                      <Lock className="w-3.5 h-3.5" />
                      未开启项目编辑权限
                    </span>
                  ) : editing ? (
                    <>
                      <Button
                        variant="claw-outline"
                        size="sm"
                        onClick={cancelEdit}
                      >
                        <X className="w-4 h-4" />
                        取消
                      </Button>
                      <Button variant="claw-primary" size="sm" onClick={save}>
                        <Save className="w-4 h-4" />
                        保存
                      </Button>
                    </>
                  ) : (
                    <Button
                      variant="claw-primary"
                      size="sm"
                      onClick={enterEdit}
                    >
                      <Pencil className="w-4 h-4" />
                      编辑
                    </Button>
                  )}
                </div>
              </div>

              <div className="p-4 space-y-4">
                {/* 同步模式：非编辑态精简展示（hover 查看说明），编辑态展开详情 */}
                {editing ? (
                  <>
                    <div>
                      <BodyMedium tone="body" className="block mb-2">
                        同步模式
                      </BodyMedium>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="inline-flex items-center h-7 px-2.5 rounded-[4px] border border-[var(--cp-border)] bg-[var(--color-gray-100)] text-sm text-[var(--text-body)]">
                            {ASSET_SYNC_MODE_MAP.autoSync.label}
                          </span>
                          <MetaText tone="weak">
                            （项目固定，不可更改）
                          </MetaText>
                        </div>
                        <MetaText className="block mt-1">
                          {ASSET_SYNC_MODE_MAP.autoSync.description.replace(
                            /组织/g,
                            "项目"
                          )}
                        </MetaText>
                      </div>
                    </div>
                    <div className="border-t border-[var(--cp-border)]" />
                  </>
                ) : (
                  <div className="flex items-center gap-2">
                    <MetaText tone="weak">同步模式：</MetaText>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="inline-flex items-center h-6 px-2 rounded-[4px] border border-[var(--cp-border)] bg-[var(--color-gray-100)] text-xs text-[var(--text-body)] cursor-help">
                          {ASSET_SYNC_MODE_MAP.autoSync.label}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent className="max-w-xs">
                        {ASSET_SYNC_MODE_MAP.autoSync.description.replace(
                          /组织/g,
                          "项目"
                        )}
                      </TooltipContent>
                    </Tooltip>
                  </div>
                )}

                <div>
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <BodyMedium tone="body">资产清单</BodyMedium>
                    {editing && (
                      <Button
                        variant="claw-outline"
                        size="sm"
                        onClick={() => setAddOpen(true)}
                      >
                        <Plus className="w-4 h-4" />
                        添加资产
                      </Button>
                    )}
                  </div>
                  {editing && (
                    <div className="flex items-start gap-2 rounded-[4px] bg-[#F0F5FF] px-3 py-2 mb-2">
                      <MetaText tone="weak" className="leading-relaxed text-xs">
                        Agent 绑定方式：归属人在项目中手动接入，或在创建云端
                        Agent 时选择绑定本项目。
                      </MetaText>
                    </div>
                  )}
                  <AssetList
                    assets={activeAssets}
                    editing={editing}
                    onRemove={removeItem}
                    project={project}
                    onOpenIntro={() => setIntroSkillOpen(true)}
                  />
                </div>
              </div>
            </div>

            <ProjectIntroSkillDialog
              open={introSkillOpen}
              onOpenChange={setIntroSkillOpen}
              project={project}
            />

            <AddAssetDialog
              open={addOpen}
              onOpenChange={setAddOpen}
              selected={draft}
              onConfirm={next => setDraft(next)}
              projectId={project.id}
              projectName={project.name}
            />

            <div className="border-t border-[var(--cp-border)] pt-4">
              <div className="mb-3">
                <BodyMedium>经验沉淀</BodyMedium>
                <MetaText className="mt-0.5 block" tone="weak">
                  Agent 执行过程中的经验会自动沉淀并回流到项目。
                </MetaText>
              </div>
              <ExperienceTab project={project} />
            </div>
          </div>
        )}
      </div>
      {/* 自动化管理弹窗 */}
      <Dialog open={automationOpen} onOpenChange={setAutomationOpen}>
        <DialogContent size="lg" className="max-h-[85vh]">
          <DialogHeader>
            <DialogTitle>自动化</DialogTitle>
            <DialogDescription>
              自动化 = 任务的自动生成器。到点触发 → 按输出模式决定是否建任务 →
              走关联工作流 → 结果沉淀回任务看板。
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6 max-h-[70vh] overflow-y-auto">
            <AutomationsTab
              project={project}
              onGotoWorkflow={() => {
                setAutomationOpen(false);
                setTab("workflow");
              }}
            />
          </DialogBody>
        </DialogContent>
      </Dialog>

      {/* 项目级弹窗（编辑项目 / 邀请成员 / 接入 agent） */}
      <ProjectSettingsDialog
        open={projectEditOpen}
        onOpenChange={setProjectEditOpen}
        project={project}
        canManagePermission={isAdmin}
      />
      <InviteMemberDialog
        open={inviteMemberOpen}
        onOpenChange={setInviteMemberOpen}
        project={project}
      />
      <Sheet open={assetRecordsOpen} onOpenChange={setAssetRecordsOpen}>
        <SheetContent
          side="right"
          className="w-[480px] sm:max-w-[480px] flex flex-col p-0"
        >
          <SheetHeader className="px-6 py-4 border-b border-[var(--cp-border)]">
            <SheetTitle>更新记录</SheetTitle>
          </SheetHeader>
          <div className="flex-1 overflow-y-auto px-6 py-4">
            <UpdateRecordsTab records={assetUpdateRecords} />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}

// ─── 左侧项目列表（样式对齐管控端左树行） ─────────────────
function ProjectListPanel({
  projects,
  selectedId,
  onSelect,
  onCreate,
  onDelete,
  canManageProjects,
}: {
  projects: TenantProject[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onCreate: () => void;
  onDelete: (project: TenantProject) => void;
  canManageProjects: boolean;
}) {
  const [query, setQuery] = useState("");
  const visibleProjects = useMemo(() => {
    const keyword = query.trim().toLowerCase();
    if (!keyword) return projects;
    return projects.filter(project =>
      project.name.toLowerCase().includes(keyword)
    );
  }, [projects, query]);

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="space-y-3 border-b border-[var(--cp-border)] px-3 py-3">
        <div className="flex items-center gap-2">
          <div className="text-sm font-medium text-[var(--text-title)]">
            我的项目
            <span className="ml-1 text-[var(--text-weak)] tabular-nums">
              （{projects.length}）
            </span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {canManageProjects && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="tenant-outline"
                  size="icon"
                  onClick={onCreate}
                  aria-label="新建项目"
                >
                  <Plus className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top" className="text-xs">
                新建项目
              </TooltipContent>
            </Tooltip>
          )}
          <div className="relative min-w-0 flex-1">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-[var(--text-weak)]" />
            <Input
              value={query}
              onChange={event => setQuery(event.target.value)}
              placeholder="搜索项目..."
              className="pl-10"
            />
          </div>
        </div>
      </div>
      <div className="flex-1 min-h-0 overflow-y-auto pt-2 pb-3">
        {visibleProjects.length > 0 ? (
          <div className="space-y-0.5 px-3">
            {visibleProjects.map(p => {
              const active = p.id === selectedId;
              const actionClass = `flex h-5 w-5 shrink-0 items-center justify-center rounded transition-colors ${
                active
                  ? "text-[var(--text-muted)] hover:bg-white hover:text-[var(--text-title)]"
                  : "text-[var(--text-weak)] hover:bg-white hover:text-[var(--text-secondary)]"
              }`;
              return (
                <div
                  key={p.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => onSelect(p.id)}
                  onKeyDown={event => {
                    if (event.key === "Enter" || event.key === " ")
                      onSelect(p.id);
                  }}
                  className={`group flex h-8 cursor-pointer items-center gap-1.5 rounded-[4px] pl-2 pr-2 text-sm text-[var(--text-title)] transition-colors ${
                    active
                      ? "bg-[var(--bg-grey-hover)] font-medium"
                      : "hover:bg-[var(--bg-grey-hover)]"
                  }`}
                >
                  <span className="min-w-0 flex-1 truncate" title={p.name}>
                    {p.name}
                  </span>
                  {p.allowMemberEdit ? (
                    <Pencil
                      className="h-3.5 w-3.5 shrink-0 text-[var(--cp-text-success)]"
                      aria-label="普通成员可编辑"
                    />
                  ) : (
                    <Lock
                      className="h-3.5 w-3.5 shrink-0 text-[var(--text-weak)]"
                      aria-label="普通成员不可编辑"
                    />
                  )}
                  <span className="shrink-0 text-[11px] tabular-nums text-[var(--text-weak)]">
                    ({p.members.length})
                  </span>
                  {canManageProjects && (
                    <span
                      className="shrink-0"
                      onClick={event => event.stopPropagation()}
                    >
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <button
                            type="button"
                            aria-label={`管理项目：${p.name}`}
                            className={actionClass}
                          >
                            <MoreHorizontal className="h-3 w-3" />
                          </button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            className="text-[var(--text-danger)]"
                            onClick={() => onDelete(p)}
                          >
                            <Trash2 className="h-4 w-4" />
                            删除项目
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        ) : (
          <div className="py-8 text-center">
            <MetaText tone="secondary">没有匹配的项目</MetaText>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── 主组件 ──────────────────────────────────────────────
export default function ProjectCollaboration() {
  const { isAdmin } = useUserRole();
  const [, navigate] = useLocation();
  const projectCollaborationEnabled = useProjectCollaborationAccessAllowed();
  const projects = useTenantProjects();
  const [selectedId, setSelectedId] = useState<string | null>(
    () => projects[0]?.id ?? null
  );
  const [createProjectOpen, setCreateProjectOpen] = useState(false);
  const [deleteProject, setDeleteProject] = useState<TenantProject | null>(
    null
  );
  const [activeTaskRef, setActiveTaskRef] = useState<{
    projectId: string;
    taskId: string;
  } | null>(null);
  const [directRuntimeTaskId, setDirectRuntimeTaskId] = useState<string | null>(
    () => new URLSearchParams(window.location.search).get("task")
  );
  const selected = useTenantProject(selectedId) ?? projects[0];
  const activeTaskProject = activeTaskRef
    ? projects.find(project => project.id === activeTaskRef.projectId)
    : undefined;
  const activeTask = activeTaskProject
    ? activeTaskProject.tasks.find(task => task.id === activeTaskRef?.taskId)
    : undefined;
  const directRuntimeTask = useMemo<TenantTask | undefined>(() => {
    if (!directRuntimeTaskId || !selected) return undefined;
    const existing = projects
      .flatMap(project => project.tasks)
      .find(
        task =>
          task.id === directRuntimeTaskId ||
          task.runtimeExecution?.backendTaskId === directRuntimeTaskId
      );
    if (existing) return undefined;
    const now = new Date().toISOString();
    return {
      id: `runtime-view-${directRuntimeTaskId}`,
      title: `真实工作流执行详情 · ${directRuntimeTaskId}`,
      description:
        "该任务由 ClawPro 编排后端创建，当前以运行态直达方式展示。",
      status: "in_progress",
      ownerId: selected.members[0]?.userId ?? "runtime",
      priority: "high",
      dueDate: now.slice(0, 10),
      workflowTemplateId: "auto",
      workflow: [],
      runtimeExecutable: true,
      runtimeExecution: {
        backendTaskId: directRuntimeTaskId,
        runtimeId: "structured-project-workflow",
        assignmentMode: "loading",
        agentRuntimeId: "loading",
        status: "loading",
        canStop: true,
        cancelRequested: false,
        currentPhase: null,
        handoffContract: "ClawPro Handoff v2",
        phases: [],
        artifacts: [],
        updatedAt: now,
      },
      triggerType: "system",
      tags: ["真实 Agent", "工作流评测"],
      createdAt: now,
      updatedAt: now,
    };
  }, [directRuntimeTaskId, projects, selected]);
  const displayedTaskProject = activeTaskProject ??
    (directRuntimeTask ? selected : undefined);
  const displayedTask = activeTask ?? directRuntimeTask;

  useEffect(() => {
    if (!directRuntimeTaskId) return;
    for (const project of projects) {
      const task = project.tasks.find(
        item =>
          item.id === directRuntimeTaskId ||
          item.runtimeExecution?.backendTaskId === directRuntimeTaskId
      );
      if (!task) continue;
      setSelectedId(project.id);
      setActiveTaskRef({ projectId: project.id, taskId: task.id });
      setDirectRuntimeTaskId(null);
      return;
    }
  }, [directRuntimeTaskId, projects]);

  // 重新运行会生成新的后端执行 ID。若当前详情由直达链接打开，
  // 必须同步替换 URL 中的旧 ID，否则刷新后会再次请求已失效的执行实例。
  useEffect(() => {
    const latestBackendTaskId = activeTask?.runtimeExecution?.backendTaskId;
    if (!latestBackendTaskId) return;
    const params = new URLSearchParams(window.location.search);
    const linkedTaskId = params.get("task");
    if (!linkedTaskId || linkedTaskId === latestBackendTaskId) return;
    params.set("task", latestBackendTaskId);
    navigate(`/project-collaboration?${params.toString()}`, { replace: true });
  }, [activeTask?.runtimeExecution?.backendTaskId, navigate]);

  const handleRuntimeStarted = (backendTaskId: string) => {
    setDirectRuntimeTaskId(null);
    const params = new URLSearchParams(window.location.search);
    params.set("task", backendTaskId);
    navigate(`/project-collaboration?${params.toString()}`, { replace: true });
  };

  const closeTaskDetail = () => {
    setActiveTaskRef(null);
    setDirectRuntimeTaskId(null);
    if (new URLSearchParams(window.location.search).has("task")) {
      navigate("/project-collaboration", { replace: true });
    }
  };

  useEffect(() => {
    if (!projectCollaborationEnabled)
      navigate("/my-openclaw", { replace: true });
  }, [navigate, projectCollaborationEnabled]);

  const confirmDeleteProject = () => {
    if (!isAdmin || !deleteProject) return;
    const nextProject = projects.find(
      project => project.id !== deleteProject.id
    );
    deleteProject.members.forEach(member =>
      removeUserFromGroup(member.userId, deleteProject.id)
    );
    groupStore.remove(deleteProject.id);
    tenantProjectStore.deleteProject(deleteProject.id);
    setSelectedId(nextProject?.id ?? null);
    toast.success(`项目“${deleteProject.name}”已删除`);
    setDeleteProject(null);
  };

  if (!projectCollaborationEnabled) return null;

  return (
    <TenantLayout>
      <div className="min-w-[1200px]">
        <div className="max-w-[1920px] mx-auto page-enter">
          <div className="relative flex flex-col min-h-[calc(100vh-64px)] pl-[120px] pr-[120px] pt-5 pb-6">
            <div className="flex items-center justify-between gap-3 mb-4">
              <h1 className="text-xl font-semibold text-[var(--text-title)] m-0">
                我的项目
              </h1>
            </div>

            {projects.length === 0 ? (
              <Empty className="py-20">
                <EmptyHeader>
                  <EmptyDescription>你还没有归属任何项目</EmptyDescription>
                </EmptyHeader>
                {isAdmin && (
                  <EmptyContent>
                    <Button
                      variant="tenant-primary"
                      onClick={() => setCreateProjectOpen(true)}
                    >
                      <Plus className="h-4 w-4" />
                      新建项目
                    </Button>
                  </EmptyContent>
                )}
              </Empty>
            ) : (
              <div className="flex flex-1 min-h-[560px] overflow-hidden rounded-[8px] border border-[var(--cp-border)] bg-white">
                <div className="w-[300px] shrink-0 border-r border-[var(--cp-border)] bg-[var(--cp-surface)]">
                  <ProjectListPanel
                    projects={projects}
                    selectedId={selected?.id ?? null}
                    onSelect={setSelectedId}
                    onCreate={() => setCreateProjectOpen(true)}
                    onDelete={setDeleteProject}
                    canManageProjects={isAdmin}
                  />
                </div>
                <div className="flex-1 min-w-0">
                  {selected && (
                    <ProjectPanel
                      key={selected.id}
                      project={selected}
                      isAdmin={isAdmin}
                      onOpenTask={taskId =>
                        setActiveTaskRef({ projectId: selected.id, taskId })
                      }
                    />
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 任务详情：弹窗形式覆盖在任务看板之上，不占全屏、关闭即回看板 */}
      <Dialog
        open={!!(displayedTaskProject && displayedTask)}
        onOpenChange={o => !o && closeTaskDetail()}
      >
        <DialogContent
          className="p-0 gap-0 overflow-hidden sm:max-w-[1080px]"
          style={{ maxHeight: "min(88vh, 900px)", height: "88vh" }}
        >
          <DialogTitle className="sr-only">
            {displayedTask?.title ?? "任务详情"}
          </DialogTitle>
          {displayedTaskProject && displayedTask && (
            <div className="flex h-full min-h-0 flex-col overflow-hidden">
              <TaskWorkflowPage
                project={displayedTaskProject}
                task={displayedTask}
                onBack={closeTaskDetail}
                onRuntimeStarted={handleRuntimeStarted}
              />
            </div>
          )}
        </DialogContent>
      </Dialog>
      <CreateProjectDialog
        open={isAdmin && createProjectOpen}
        onOpenChange={setCreateProjectOpen}
        onCreate={input => {
          if (!isAdmin) return;
          const projectId = tenantProjectStore.createProject(input);
          groupStore.upsert({
            id: projectId,
            name: input.name,
            parentId: null,
            source: "project",
            readonly: false,
            createdAt: new Date().toISOString(),
          });
          addUsersToGroup([CURRENT_USER], projectId);
          setSelectedId(projectId);
          toast.success("项目已创建，可继续邀请成员");
        }}
      />
      <AlertDialog
        open={isAdmin && !!deleteProject}
        onOpenChange={open => !open && setDeleteProject(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除项目</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription>
            删除“{deleteProject?.name}
            ”后，项目任务、成员关系、资产配置和协作记录将不再展示。此操作不可撤销。
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="tenant-outline">取消</Button>
            </AlertDialogCancel>
            <AlertDialogAction
              variant="tenant-destructive"
              onClick={confirmDeleteProject}
            >
              删除项目
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </TenantLayout>
  );
}
