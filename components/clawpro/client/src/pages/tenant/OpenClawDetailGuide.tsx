/**
 * OpenClawDetailGuide - OpenClaw 详情页「基础配置」
 * Figma: node 247:5352（Clawpro 交互稿）
 *
 * 修改记录（2026-05-18）：
 *   1) 背景复用「我的 Agent」的线+点阵背景
 *   2) 左侧纵向 Tab 改为横向 Segmented Control（§8.6 规范）
 *   3) 技能区域新增「安装新技能」弹窗交互（含分类筛选 + 搜索 + 已安装标记）
 *   4) 整体视觉刷新至最新设计规范
 */
import { useState, useRef, useEffect, useCallback, useMemo, Fragment, type ReactNode } from "react";
import { useLocation } from "wouter";
import TenantLayout from "@/components/TenantLayout";
import { TenantCard, SurfaceInner } from "@/components/ui/Surface";
import { Separator } from "@/components/ui/separator";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { SelectPanel, SelectPanelItem } from "@/components/ui/select-panel";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import { BodyText, MetaText, HelperText, BodyMedium, MetaMedium, PanelTitle, CompactText } from "@/components/ui/Typography";
import { MOCK_ROLES, type Role, type AgentRoleSlot } from "@/lib/mockData";
import { RoleConfigProgressDialog, RoleConfigProgressContent, useRoleConfigProgress, type RoleConfigProgressItem } from "@/components/agent/RoleConfigProgress";
import { cn } from "@/lib/utils";
import { getDemoBackupStatus, type BackupPointStatus } from "@/lib/backupDemo";
import { TenantSection } from "@/components/ui/TenantSection";
import { TenantSegmentGroup, TenantSegmentOption } from "@/components/ui/segment";
import { Alert, AlertTitle, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { Empty, EmptyHeader, EmptyMedia, EmptyDescription } from "@/components/ui/empty";
import { Button, SmallIconStateButton } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
// 引导页与实例详情页共用同一份龙虾医生完整交互 UI，避免重复实现。
import { DoctorChatCard } from "./OpenClawDetail";
import {
  Trash2,
  Search,
  Plus,
  ExternalLink,
  RefreshCw,
  ArrowLeft,
  ArrowLeftRight,
  Send,
  Check,
  CheckCircle2,
  AlertCircle,
  CircleAlert,
  XCircle,
  Star,
  Download,
  Copy,
  Clock,
  Loader2,
  Megaphone,
  Wrench,
  ChevronRight,
  Info,
  HardDriveDownload,
  Users,
  Crown,
  Settings2,
  RotateCcw,
  Pencil,
  UserCog,
} from "lucide-react";

// ─── 自定义图标（设计稿：基础配置） ─────────────────────────────────────────

const Edit3 = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M13.6816 5.69979L11.2376 3.25635C11.1563 3.17508 11.0599 3.11061 10.9537 3.06662C10.8475 3.02264 10.7337 3 10.6188 3C10.5039 3 10.3901 3.02264 10.2839 3.06662C10.1777 3.11061 10.0813 3.17508 10 3.25635L3.25649 9.99987C3.17488 10.0808 3.11019 10.1772 3.06615 10.2834C3.02212 10.3896 2.99964 10.5034 3 10.6184V13.0624C3 13.2944 3.09219 13.517 3.25629 13.6811C3.42038 13.8452 3.64294 13.9374 3.875 13.9374H13.0625C13.1785 13.9374 13.2898 13.8913 13.3719 13.8092C13.4539 13.7272 13.5 13.6159 13.5 13.4999C13.5 13.3838 13.4539 13.2726 13.3719 13.1905C13.2898 13.1085 13.1785 13.0624 13.0625 13.0624H7.55657L13.6816 6.93737C13.7628 6.85611 13.8273 6.75965 13.8713 6.65347C13.9153 6.5473 13.9379 6.4335 13.9379 6.31858C13.9379 6.20366 13.9153 6.08986 13.8713 5.98368C13.8273 5.87751 13.7628 5.78104 13.6816 5.69979ZM6.31899 13.0624H3.875V10.6184L8.6875 5.80588L11.1315 8.24987L6.31899 13.0624ZM11.75 7.63135L9.30657 5.18737L10.6191 3.87487L13.0625 6.31885L11.75 7.63135Z" fill="currentColor"/>
  </svg>
);

const X = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M12 4L4 12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M4 4L12 12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
  </svg>
);

const ChevronDown = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M4 6L8 10L12 6" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
  </svg>
);

const Eye = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M14.5306 8.42161C14.0092 9.68679 13.0881 10.7466 11.9079 11.4391C10.7277 12.1316 9.35326 12.4189 7.99448 12.257C6.6357 12.095 5.37587 11.5203 4.38447 10.6147C3.39307 9.70909 2.72126 8.52099 2.45001 7.21411C2.39801 6.95779 2.39801 6.69222 2.45001 6.4359C2.72126 5.12902 3.39307 3.94092 4.38447 3.03529C5.37587 2.12967 6.6357 1.55497 7.99448 1.39304C9.35326 1.23111 10.7277 1.51835 11.9079 2.21089C13.0881 2.90342 14.0092 3.96319 14.5306 5.22837C14.5792 5.35934 14.5792 5.5034 14.5306 5.63437" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
    <circle cx="8" cy="8" r="2" stroke="currentColor"/>
  </svg>
);

const EyeOff = ({ className }: { className?: string }) => (
  <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" className={className}>
    <path d="M7.09448 4.58628C8.45326 4.42435 9.8277 4.71159 11.0079 5.40412C12.1881 6.09666 13.1092 7.15643 13.6306 8.42161C13.6792 8.55258 13.6792 8.69664 13.6306 8.8276C13.4162 9.34734 13.1329 9.83588 12.7883 10.2801" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M9.04893 9.88387C8.71889 10.2026 8.27684 10.379 7.81801 10.375C7.35917 10.3711 6.92026 10.187 6.5958 9.86255C6.27135 9.5381 6.08731 9.09918 6.08332 8.64035C6.07933 8.18151 6.25572 7.73947 6.57449 7.40942" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M11.0295 11.8328C10.2557 12.2912 9.39241 12.5777 8.49812 12.673C7.60383 12.7683 6.6995 12.6701 5.8465 12.3852C4.99349 12.1002 4.21178 11.635 3.55438 11.0213C2.89699 10.4075 2.37931 9.65957 2.03646 8.82813C1.98785 8.69716 1.98785 8.5531 2.03646 8.42214C2.55365 7.16791 3.46366 6.11525 4.62991 5.42212" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
    <path d="M3.896 3.81274L12.2083 13" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round"/>
  </svg>
);
import { toast } from "sonner";
import { AgentAvatar } from "@/components/agent/AgentAvatar";
import { RoleManageSheet } from "@/components/agent/RoleManageSheet";
import { BatchSwitchRoleDialog } from "@/components/agent/BatchSwitchRoleDialog";
import { AddRoleDialog } from "@/components/agent/AddRoleDialog";
import { SkillManagementUpdateNotice } from "@/components/agent/SkillManagementUpdateNotice";
import { AgentTypeSwitchDialog, type SwitchableAgentType } from "@/components/agent/AgentTypeSwitchDialog";
import { StatusBadge } from "@/components/agent/StatusBadge";
import { MODEL_PROVIDERS, getAdminModelProviders, getSelfProvidersByCategory, MODEL_PROVIDER_GROUP_LABELS, SELF_CONFIG_CATEGORY_LABELS, SELF_CONFIG_CATEGORY_ORDER, DEFAULT_CUSTOM_JSON, CHANNEL_OPTIONS, type SelfConfigCategory, type ChannelField, type ChannelConfig } from "@/lib/agentConfigConstants";
import {
  type CustomChannel as AdminCustomChannel,
  loadVisibleCustomChannels,
  onCustomChannelsChange,
  loadBuiltinChannelVisibility,
  onBuiltinChannelVisibilityChange,
} from "@/lib/customChannelStore";
import { getActivePush, compareVersion, type ActivePush } from "@/lib/upgradePushStore";
import ToolsMcpPanel from "./ToolsMcpPanel";
import FileSpace from "./FileSpace";
import MemoryPreview from "@/components/MemoryPreview";
import { loadClawList, saveClawList, notifyClawListChange } from "@/lib/openclawStore";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import { MOCK_OPENCLAW_LIST } from "@/lib/mockData";
import { groupStore } from "../admin/MemberManagement/groupStore";
import { projectAssetStore } from "../admin/project-assets/projectAssetStore";
import { getAssetItemDisplay } from "../admin/project-assets/assetSelectors";

// ─── 角色管理抽屉（复用「我的 Agent」方案1）配套定义 ───────────────────────
// 技能名 → 技能描述映射（RoleSkill 无 description 字段，此处按常见技能名给出说明，未命中回落到通用文案）
const SKILL_DESCRIPTION_MAP: Record<string, string> = {
  "github": "通过 GitHub CLI 管理 Issue、PR、CI 运行与 API 查询，覆盖代码协作全流程",
  "code-reviewer": "自动审查代码质量与安全漏洞，支持主流编程语言，提供修改建议与最佳实践",
  "docker-ops": "容器构建、镜像管理与编排部署，简化本地与云端的容器化工作流",
  "api-tester": "自动化接口测试与断言校验，支持批量用例与回归验证",
  "data-analyst": "数据清洗、统计分析与可视化，快速产出结论与图表",
  "sql-expert": "自然语言转 SQL、查询优化与库表结构分析",
  "web-search-pro": "高级联网检索与信息聚合，返回结构化、可溯源的搜索结果",
  "self-improving-agent": "根据反馈持续优化自身提示词与执行策略",
  "email-writer": "撰写与润色各类邮件，适配正式 / 商务 / 日常语气",
  "k8s-manager": "Kubernetes 集群管理、Pod 编排与资源诊断",
};
const getSkillDescription = (name: string) => SKILL_DESCRIPTION_MAP[name] ?? "该技能将随角色一并预装并激活";

// Pill 单选标签：未选中白底深字描边，选中变实心黑底白字
function PillRadioOption({
  value,
  id,
  children,
}: {
  value: string;
  id: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center">
      <RadioGroupItem value={value} id={id} className="peer sr-only" />
      <Label
        htmlFor={id}
        className="flex items-center justify-center whitespace-nowrap h-6 rounded-full border border-[var(--border)] bg-[var(--card)] px-3 text-xs font-medium text-[var(--text-emphasis)] hover:bg-[var(--accent)] hover:border-[var(--border-control)] cursor-pointer peer-data-[state=checked]:bg-[var(--text-emphasis)] peer-data-[state=checked]:text-white peer-data-[state=checked]:border-[var(--text-emphasis)] peer-data-[state=checked]:hover:bg-[var(--text-emphasis)] peer-data-[state=checked]:hover:text-white transition-colors"
      >
        {children}
      </Label>
    </div>
  );
}

interface BatchSwitchOption {
  value: string;
  label: string;
  intro?: { name: string; skills: string; soul: string } | null;
}
// Radix Select 的 Item value 不允许为空串，用哨兵值表示「不切换」
const BATCH_NO_SWITCH = "__no_switch__";

// ─── 当前实例 mock 数据（Guide 页演示用） ─────────────────────────────────
// 真实业务接入时替换为 findClawById(id)，目前 Guide 版写死演示数据
const MOCK_INSTANCE = {
  agentType: "OpenClaw" as const, // 与 upgradePushStore 的 key 对齐
  agentVersion: "2026.4.0",       // 当前实例版本（< 推送目标版本 2026.4.23 才会出现徽章）
};

const DEMO_OPENCLAW_PANEL_URL =
  "http://43.139.137.45:38341/knmnz8?token=8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb";
const DEMO_OPENCLAW_PANEL_TOKEN = "8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb";
const HERMES_018_DEMO_CREDENTIALS = {
  username: "ClawPro1134",
  password: "sm$WNzlXfhpRFJZ9",
} as const;

type DetailAgentMeta = {
  id: string;
  instanceId: string;
  name: string;
  status?: string;
  agentType?: "openclaw" | "hermes" | "lightclawace" | "localagent";
  model?: string;
  localProduct?: "Claude" | "CodeBuddy" | "WorkBuddy" | "Codex";
  localConnectionStatus?: "connected" | "disconnected";
  localResourceSyncStatus?: "syncing" | "synced";
  /** 外部 Agent 最近一次向 Hatchery 同步信息的时间 */
  lastReportedAt?: string;
  groupName?: string;
  roleName?: string;
  /** 多角色实例的角色位列表（AgentRoleSlot[]，与卡片页同源） */
  roles?: AgentRoleSlot[];
  /** 角色总数（与卡片页 roleCount 对齐） */
  roleCount?: number;
  /** 分组允许的角色名单（undefined=不限制） */
  allowedRoleNames?: string[];
  version?: string;
  modelVersion?: string;
  /** 关联项目 ID 列表（新增关联携带技能的项目会自动安装其技能） */
  projectIds?: string[];
  /** 关联项目名称列表（与 projectIds 一一对应，用于展示） */
  projectNames?: string[];
};

const LOCAL_AGENT_INACTIVE_DAYS = 7;
const LOCAL_AGENT_INACTIVE_MS = LOCAL_AGENT_INACTIVE_DAYS * 24 * 60 * 60 * 1000;

function parseLocalReportedAt(value?: string) {
  if (!value) return null;
  const normalized = value.includes("T") ? value : value.replace(" ", "T");
  const timestamp = new Date(normalized).getTime();
  return Number.isNaN(timestamp) ? null : timestamp;
}

function isLocalAgentInactive(agent?: DetailAgentMeta) {
  if (agent?.agentType !== "localagent") return false;
  const lastReportedAt = parseLocalReportedAt(agent.lastReportedAt);
  if (!lastReportedAt) return true;
  return Date.now() - lastReportedAt > LOCAL_AGENT_INACTIVE_MS;
}

const getRecentLocalAgentReportedAt = () =>
  new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString().slice(0, 19).replace("T", " ");

function normalizeExternalAgentType(value?: string) {
  if (!value) return "OpenClaw";
  if (value === "Hermes Agent") return "Hermes";
  if (value === "Claude") return "Claude Code";
  return value;
}

function formatLocalClientProduct(product?: DetailAgentMeta["localProduct"]) {
  if (product === "Claude") return "Claude Code";
  return product;
}

function formatAgentVersion(version?: string, localProduct?: DetailAgentMeta["localProduct"]) {
  const fallbackVersion = version || MOCK_INSTANCE.agentVersion;
  if (!localProduct) {
    return fallbackVersion.startsWith("v") ? fallbackVersion : `v${fallbackVersion}`;
  }

  const productPrefix = formatLocalClientProduct(localProduct)?.toLowerCase().replace(/\s+/g, "-") ?? localProduct.toLowerCase();
  const normalizedVersion = fallbackVersion
    .replace(new RegExp(`^v?${productPrefix}-`, "i"), "")
    .replace(new RegExp(`^v?${localProduct.toLowerCase()}-`, "i"), "")
    .replace(/^v/i, "");
  return `${formatLocalClientProduct(localProduct) ?? localProduct} ${normalizedVersion}`;
}

function formatExternalAgentVersion(version: string | undefined, agentTypeLabel: string) {
  const fallbackVersion = version || MOCK_INSTANCE.agentVersion;
  const typePrefix = agentTypeLabel.toLowerCase().replace(/\s+/g, "-");
  const normalizedVersion = fallbackVersion
    .replace(new RegExp(`^v?${typePrefix}-`, "i"), "")
    .replace(/^v/i, "");
  return `${agentTypeLabel} ${normalizedVersion}`;
}

const LOCAL_DETAIL_DEMOS: DetailAgentMeta[] = [
  {
    id: "oc-external-claude-demo",
    instanceId: "external-claude-01",
    name: "Claude Code 外部 Agent",
    status: "running",
    agentType: "localagent",
    localProduct: "Claude",
    localResourceSyncStatus: "synced",
    lastReportedAt: getRecentLocalAgentReportedAt(),
    groupName: "大计算 ClawPro / 计算产品中心",
    model: "Claude Code",
    modelVersion: "claude-code-1.0.90",
  },
  {
    id: "oc-external-codex-demo",
    instanceId: "external-codex-01",
    name: "Codex 外部 Agent",
    status: "running",
    agentType: "localagent",
    localProduct: "Codex",
    localResourceSyncStatus: "synced",
    lastReportedAt: getRecentLocalAgentReportedAt(),
    groupName: "大计算 ClawPro / 计算产品中心",
    model: "Codex",
    modelVersion: "codex-0.28.0",
  },
  {
    id: "oc-external-hermes-demo",
    instanceId: "external-hermes-01",
    name: "Hermes 外部 Agent",
    status: "running",
    agentType: "localagent",
    localResourceSyncStatus: "synced",
    lastReportedAt: getRecentLocalAgentReportedAt(),
    groupName: "大计算 ClawPro / 计算产品中心",
    model: "Hermes",
    modelVersion: "hermes-0.13.0",
  },
  {
    id: "oc-external-openclaw-demo",
    instanceId: "external-openclaw-01",
    name: "OpenClaw 外部 Agent",
    status: "running",
    agentType: "localagent",
    localResourceSyncStatus: "synced",
    lastReportedAt: "2026-07-20 20:50:44",
    groupName: "默认",
    model: "OpenClaw",
    modelVersion: "2026.4.23",
  },
  {
    id: "oc-local-workbuddy-demo",
    instanceId: "local-workbuddy-01",
    name: "WorkBuddy-运营笔记本",
    status: "running",
    agentType: "localagent",
    localProduct: "WorkBuddy",
    localConnectionStatus: "connected",
    localResourceSyncStatus: "syncing",
    lastReportedAt: getRecentLocalAgentReportedAt(),
    groupName: "运营组",
    modelVersion: "workbuddy-2.3.1",
  },
  {
    id: "oc-local-workbuddy-offline-demo",
    instanceId: "local-workbuddy-02",
    name: "WorkBuddy-离线笔记本",
    status: "running",
    agentType: "localagent",
    localProduct: "WorkBuddy",
    localConnectionStatus: "disconnected",
    localResourceSyncStatus: "synced",
    lastReportedAt: "2026-06-22 09:12:40",
    groupName: "运营组",
    modelVersion: "workbuddy-2.2.0",
  },
];

// ─── 顶部 Tab 数据（横向 Segmented Control） ─────────────────────────────
type DetailTab = "basic" | "tools" | "memory" | "files" | "doctor";
const DETAIL_TABS: { id: DetailTab; label: string }[] = [
  { id: "basic", label: "基础配置" },
  { id: "tools", label: "工具管理" },
  { id: "memory", label: "记忆管理" },
  { id: "files", label: "网盘管理" },
  { id: "doctor", label: "龙虾医院" },
];

// ─── 多角色（Multi-Role）─────────────────────────────────────────────────
// 多角色来源于「对话视图」左侧「角色」区用户创建的多个角色（如：通用助手 / 行业分析师）。
// 当 Agent 拥有 > 1 个角色时视为多角色，可为每个角色单独配置模型 / 通道 / 技能。
// 后端多角色字段尚未就绪，这里用 mock；接后端后改为读取该 Agent 在对话视图创建的角色列表。
type AgentRole = {
  id: string;
  name: string;
  /** 角色类型名（头像锚定 + 类型标注）；缺省回退到 name */
  baseRoleName?: string;
  /** 是否为主角色（isMain=true 的角色位） */
  isMain?: boolean;
  // 平台预设角色自带的技能清单：用户选择该角色创建时即带出，非空白。
  // 接后端后改为读取该预设角色模板携带的 skills。
  presetSkills?: { name: string; version: string }[];
};

// mock：对话视图创建的角色列表（与左侧「角色」区一致）
// 每个预设角色都携带各自的默认技能（模拟平台提供的角色模板自带 skill）。
const MOCK_AGENT_ROLES: AgentRole[] = [
  {
    id: "role-assistant",
    name: "通用助手",
    presetSkills: [
      { name: "code-interpreter", version: "1.2.0" },
      { name: "image-recognition", version: "0.9.1" },
      { name: "text-to-speech", version: "1.0.0" },
      { name: "pdf-parser", version: "1.1.0" },
      { name: "markdown-converter", version: "1.0.1" },
      { name: "summarizer", version: "2.2.0" },
      { name: "translation-engine", version: "3.0.1" },
      { name: "ocr-reader", version: "2.0.0" },
    ],
  },
  {
    id: "role-analyst",
    name: "行业分析师",
    presetSkills: [
      { name: "data-visualizer", version: "1.4.0" },
      { name: "chart-generator", version: "1.1.0" },
      { name: "csv-processor", version: "0.6.0" },
      { name: "excel-reader", version: "2.0.0" },
      { name: "sql-executor", version: "1.5.0" },
      { name: "sentiment-analyzer", version: "1.0.0" },
      { name: "keyword-extractor", version: "0.5.2" },
      { name: "summarizer", version: "2.2.0" },
    ],
  },
  {
    id: "role-pm",
    name: "项目经理",
    presetSkills: [
      { name: "task-planner", version: "2.1.0" },
      { name: "gantt-generator", version: "1.0.0" },
      { name: "meeting-summary", version: "1.3.0" },
      { name: "risk-analyzer", version: "1.2.0" },
      { name: "doc-summarizer", version: "1.3.0" },
      { name: "notification-bot", version: "0.8.0" },
    ],
  },
  {
    id: "role-designer",
    name: "设计师",
    presetSkills: [
      { name: "image-recognition", version: "0.9.1" },
      { name: "color-palette", version: "1.0.0" },
      { name: "figma-export", version: "2.0.1" },
      { name: "icon-search", version: "1.1.0" },
      { name: "layout-analyzer", version: "0.7.0" },
      { name: "screenshot-to-code", version: "1.4.0" },
    ],
  },
  {
    id: "role-writer",
    name: "内容创作者",
    presetSkills: [
      { name: "copywriting-assistant", version: "1.5.0" },
      { name: "seo-optimizer", version: "2.0.0" },
      { name: "grammar-checker", version: "1.2.0" },
      { name: "translation-engine", version: "3.0.1" },
      { name: "tone-adjuster", version: "1.0.0" },
      { name: "plagiarism-detector", version: "0.9.0" },
    ],
  },
];

// ─── 模拟数据：已接入通道 ─────────────────────────────────────────────────
const MOCK_CHANNELS: { id: string; name: string }[] = [
  { id: "feishu", name: "飞书" },
  { id: "qq", name: "QQ" },
];

// ─── 模拟数据：已安装技能 ─────────────────────────────────────────────────
const MOCK_INSTALLED_SKILLS: { name: string; version: string }[] = [
  { name: "code-interpreter", version: "1.2.0" },
  { name: "image-recognition", version: "0.9.1" },
  { name: "text-to-speech", version: "1.0.0" },
  { name: "pdf-parser", version: "1.1.0" },
  { name: "excel-reader", version: "2.0.0" },
  { name: "web-scraper", version: "1.3.2" },
  { name: "json-formatter", version: "0.8.0" },
  { name: "markdown-converter", version: "1.0.1" },
  { name: "api-tester", version: "2.1.0" },
  { name: "sql-executor", version: "1.5.0" },
  { name: "file-compressor", version: "0.9.0" },
  { name: "email-sender", version: "1.2.3" },
  { name: "calendar-sync", version: "1.0.0" },
  { name: "task-scheduler", version: "0.7.1" },
  { name: "log-analyzer", version: "2.0.0" },
  { name: "data-visualizer", version: "1.4.0" },
  { name: "chart-generator", version: "1.1.0" },
  { name: "csv-processor", version: "0.6.0" },
  { name: "translation-engine", version: "3.0.1" },
  { name: "sentiment-analyzer", version: "1.0.0" },
  { name: "keyword-extractor", version: "0.5.2" },
  { name: "summarizer", version: "2.2.0" },
  { name: "plagiarism-checker", version: "1.0.0" },
  { name: "grammar-fixer", version: "1.3.0" },
  { name: "tone-adjuster", version: "0.9.0" },
  { name: "image-resizer", version: "1.0.0" },
  { name: "screenshot-tool", version: "1.1.0" },
  { name: "video-transcriber", version: "0.8.1" },
  { name: "audio-converter", version: "1.0.0" },
  { name: "ocr-reader", version: "2.0.0" },
  { name: "qr-generator", version: "0.4.0" },
  { name: "barcode-scanner", version: "1.0.0" },
  { name: "color-picker", version: "0.3.0" },
  { name: "font-matcher", version: "0.2.1" },
  { name: "icon-finder", version: "1.0.0" },
  { name: "regex-builder", version: "1.1.0" },
  { name: "cron-parser", version: "0.5.0" },
  { name: "jwt-decoder", version: "1.0.0" },
  { name: "hash-generator", version: "0.8.0" },
  { name: "uuid-creator", version: "1.0.0" },
  { name: "dns-lookup", version: "0.6.0" },
  { name: "port-scanner", version: "1.2.0" },
  { name: "ssl-checker", version: "1.0.0" },
  { name: "ping-monitor", version: "0.9.0" },
  { name: "git-helper", version: "2.1.0" },
  { name: "docker-manager", version: "1.3.0" },
  { name: "k8s-inspector", version: "0.7.0" },
  { name: "ci-cd-trigger", version: "1.0.0" },
  { name: "env-validator", version: "0.4.0" },
  { name: "config-diff", version: "1.0.0" },
  { name: "dependency-checker", version: "1.5.0" },
  { name: "license-scanner", version: "0.3.0" },
  { name: "changelog-gen", version: "1.0.0" },
];

// ─── 模拟数据：安装队列（含 pending / installing / failed 三种状态） ─────
type PendingSkillStatus = "pending" | "installing" | "failed";
type PendingSkill = { id: string; name: string; status: PendingSkillStatus };
// ps-3 和 ps-7 模拟安装失败（与 main 分支保持一致）
const MOCK_FAIL_IDS = new Set(["ps-3", "ps-7"]);
const MOCK_PENDING_SKILLS: PendingSkill[] = [
  { id: "ps-1", name: "code-interpreter 1.2.0", status: "pending" },
  { id: "ps-2", name: "image-recognition 0.9.1", status: "pending" },
  { id: "ps-3", name: "data-analysis 2.0.0", status: "pending" },
  { id: "ps-4", name: "text-to-speech 1.0.0", status: "pending" },
  { id: "ps-5", name: "pdf-parser 1.1.0", status: "pending" },
  { id: "ps-6", name: "excel-reader 2.0.0", status: "pending" },
  { id: "ps-7", name: "video-transcribe 0.7.0", status: "pending" },
  { id: "ps-8", name: "sentiment-analysis 1.0.0", status: "pending" },
  { id: "ps-9", name: "ocr-scanner 1.3.0", status: "pending" },
  { id: "ps-10", name: "sql-query 2.1.0", status: "pending" },
  { id: "ps-11", name: "web-scraper 0.6.0", status: "pending" },
  { id: "ps-12", name: "chart-generator 1.0.0", status: "pending" },
];

// ─── 技能头像字母配色（浅底+深色字母） ────────────────────────────────────────────────
// 注：以下为 Skill / Agent 头像专用品牌色板，属插画级配色，不参与 token 化
const LETTER_COLORS: Record<string, { bg: string; text: string }> = {
  A: { bg: "#E8F4FD", text: "#1A73E8" },
  B: { bg: "#F3E8FD", text: "#8B5CF6" },
  C: { bg: "#E8FDF0", text: "#16A34A" },
  D: { bg: "#FDF2E8", text: "#EA580C" },
  E: { bg: "#FDE8F0", text: "#DC2626" },
  F: { bg: "#FDE8F0", text: "#DC2626" },
  G: { bg: "#E8FDF0", text: "#16A34A" },
  H: { bg: "#E8F4FD", text: "#1A73E8" },
  I: { bg: "#F3E8FD", text: "#8B5CF6" },
  J: { bg: "#FDF2E8", text: "#EA580C" },
  K: { bg: "#E8FDF0", text: "#16A34A" },
  L: { bg: "#E8F4FD", text: "#1A73E8" },
  M: { bg: "#F3E8FD", text: "#8B5CF6" },
  N: { bg: "#FDE8F0", text: "#DC2626" },
  O: { bg: "#FDF2E8", text: "#EA580C" },
  P: { bg: "#E8FDF0", text: "#16A34A" },
  Q: { bg: "#E8F4FD", text: "#1A73E8" },
  R: { bg: "#F3E8FD", text: "#8B5CF6" },
  S: { bg: "#E8F4FD", text: "#1A73E8" },
  T: { bg: "#F3E8FD", text: "#8B5CF6" },
  U: { bg: "#E8FDF0", text: "#16A34A" },
  V: { bg: "#FDF2E8", text: "#EA580C" },
  W: { bg: "#FDE8F0", text: "#DC2626" },
  X: { bg: "#E8F4FD", text: "#1A73E8" },
  Y: { bg: "#F3E8FD", text: "#8B5CF6" },
  Z: { bg: "#E8FDF0", text: "#16A34A" },
};
function getLetterColor(letter: string): { bg: string; text: string } {
  return LETTER_COLORS[letter.toUpperCase()] || { bg: "#E8F4FD", text: "#1A73E8" };
}

// ─── Skill 品牌色板（统一管理，避免散落 hex） ──────────────────────────────
// 这些色值用于 Skill / Agent 头像渲染，属插画级配色，不参与设计 token
const SKILL_BRAND_COLORS = {
  blue: "#4A6CF7",
  pink: "#E05A9C",
  purple: "#7C3AED",
  green: "var(--text-success)",
  orange: "#E67E22",
  cyan: "#0891B2",
  red: "var(--text-danger)",
  brightBlue: "#2563EB",
  violet: "#9333EA",
  rose: "#E11D48",
} as const;

// ─── 技能库数据（弹窗用，参考 skillhub.cn 分类体系） ────────────────────────
type SkillCategory = "all" | "ai" | "dev" | "tool" | "efficiency" | "data" | "content" | "security" | "collab";
const SKILL_CATEGORIES: { id: SkillCategory; label: string }[] = [
  { id: "all", label: "全部" },
  { id: "ai", label: "AI" },
  { id: "dev", label: "智能开发" },
  { id: "tool", label: "工具" },
  { id: "efficiency", label: "效率提升" },
  { id: "data", label: "数据分析" },
  { id: "content", label: "内容创作" },
  { id: "security", label: "安全合规" },
  { id: "collab", label: "通讯协作" },
];

function formatSkillCount(count: number) {
  if (count >= 10000) return `${Math.round(count / 10000)}万`;
  if (count >= 1000) return `${(count / 1000).toFixed(1)}k`;
  return `${count}`;
}

interface SkillItem {
  id: string;
  name: string;
  initial: string;
  color: string;
  description: string;
  category: SkillCategory;
  installed: boolean;
  favorites: number;
  downloads: number;
  source: "ClawHub" | "SkillHub";
}

const SKILL_LIBRARY: SkillItem[] = [
  // ── AI ──
  {
    id: "self-improving",
    name: "Self-Improving Agent",
    initial: "S",
    color: SKILL_BRAND_COLORS.blue,
    description: "捕获经验教训、错误和纠正，实现 Agent 持续自我改进。当命令执行失败或用户纠正输出时自动记录并优化后续策略。",
    category: "ai",
    installed: false,
    favorites: 785,
    downloads: 194000,
    source: "ClawHub",
  },
  {
    id: "find-skills",
    name: "Find Skills",
    initial: "F",
    color: SKILL_BRAND_COLORS.pink,
    description: "当用户询问\"如何做某事\"或希望扩展功能时，自动从 SkillHub 发现并推荐合适的技能插件。",
    category: "ai",
    installed: false,
    favorites: 1230,
    downloads: 312000,
    source: "SkillHub",
  },
  {
    id: "deepthink",
    name: "DeepThink Reasoner",
    initial: "D",
    color: SKILL_BRAND_COLORS.purple,
    description: "深度推理引擎，面对复杂多步骤问题时启用链式思考(CoT)，提升推理准确率和逻辑一致性。",
    category: "ai",
    installed: false,
    favorites: 2150,
    downloads: 458000,
    source: "ClawHub",
  },
  // ── 智能开发 ──
  {
    id: "github",
    name: "Github",
    initial: "G",
    color: "var(--text-success)",
    description: "通过 gh CLI 与 GitHub 深度集成，管理 Issue、PR、CI/CD 运行、代码审查及 GitHub API 高级查询。",
    category: "dev",
    installed: false,
    favorites: 3420,
    downloads: 890000,
    source: "SkillHub",
  },
  {
    id: "agent-browser",
    name: "Agent Browser",
    initial: "A",
    color: SKILL_BRAND_COLORS.orange,
    description: "基于 Rust 的高性能无头浏览器，支持页面导航、DOM 操作、截图和结构化数据提取，适合网页自动化任务。",
    category: "dev",
    installed: false,
    favorites: 567,
    downloads: 86000,
    source: "ClawHub",
  },
  {
    id: "code-review",
    name: "Code Review Assistant",
    initial: "C",
    color: SKILL_BRAND_COLORS.cyan,
    description: "智能代码审查助手，自动检测代码中的安全漏洞、性能瓶颈和规范违规，提供修复建议和最佳实践参考。",
    category: "dev",
    installed: true,
    favorites: 1890,
    downloads: 267000,
    source: "SkillHub",
  },
  // ── 工具 ──
  {
    id: "tavily-search",
    name: "Tavily Search",
    initial: "T",
    color: SKILL_BRAND_COLORS.brightBlue,
    description: "实时互联网搜索引擎，为 Agent 提供最新的网页搜索结果、摘要和事实核查能力，支持多语言查询。",
    category: "tool",
    installed: false,
    favorites: 4210,
    downloads: 1230000,
    source: "SkillHub",
  },
  {
    id: "file-converter",
    name: "File Converter",
    initial: "F",
    color: "var(--text-danger)",
    description: "万能文件格式转换工具，支持 PDF、Word、Excel、Markdown、HTML 等 50+ 格式间的互转。",
    category: "tool",
    installed: false,
    favorites: 923,
    downloads: 156000,
    source: "ClawHub",
  },
  // ── 效率提升 ──
  {
    id: "ai-ppt-generator",
    name: "AI PPT Generator",
    initial: "P",
    color: "var(--text-success)",
    description: "根据主题和大纲自动生成专业演示文稿，支持多种模板风格，包含配图建议和演讲者注记。",
    category: "efficiency",
    installed: false,
    favorites: 2670,
    downloads: 534000,
    source: "ClawHub",
  },
  {
    id: "meeting-summary",
    name: "Meeting Summary",
    initial: "M",
    color: SKILL_BRAND_COLORS.violet,
    description: "自动提取会议录音/文字记录的核心要点，生成结构化纪要，包含行动项、决策和待跟进事项。",
    category: "efficiency",
    installed: true,
    favorites: 1560,
    downloads: 345000,
    source: "SkillHub",
  },
  // ── 数据分析 ──
  {
    id: "data-viz",
    name: "Data Visualizer",
    initial: "D",
    color: SKILL_BRAND_COLORS.orange,
    description: "将结构化数据自动转化为图表（折线图、柱状图、饼图等），支持趋势分析和异常检测提示。",
    category: "data",
    installed: false,
    favorites: 1120,
    downloads: 278000,
    source: "SkillHub",
  },
  {
    id: "sql-assistant",
    name: "SQL Assistant",
    initial: "S",
    color: SKILL_BRAND_COLORS.purple,
    description: "自然语言转 SQL 查询，支持 MySQL/PostgreSQL/ClickHouse 等主流数据库，含查询优化建议。",
    category: "data",
    installed: false,
    favorites: 890,
    downloads: 198000,
    source: "ClawHub",
  },
  // ── 内容创作 ──
  {
    id: "xhs-skill",
    name: "小红书创作助手",
    initial: "小",
    color: SKILL_BRAND_COLORS.rose,
    description: "针对小红书平台优化的内容创作工具，自动生成爆款标题、正文排版、话题标签和发布时间建议。",
    category: "content",
    installed: false,
    favorites: 5680,
    downloads: 1450000,
    source: "SkillHub",
  },
  {
    id: "copywriting",
    name: "AI Copywriter",
    initial: "C",
    color: SKILL_BRAND_COLORS.violet,
    description: "多风格文案生成器，支持广告文案、品牌故事、产品描述和社交媒体帖子，可指定语气和目标受众。",
    category: "content",
    installed: false,
    favorites: 2340,
    downloads: 567000,
    source: "ClawHub",
  },
  // ── 安全合规 ──
  {
    id: "skill-vetter",
    name: "Skill Vetter",
    initial: "S",
    color: SKILL_BRAND_COLORS.blue,
    description: "技能安全预审工具，在安装第三方技能前自动检查风险信号、权限范围及可疑模式，保障 Agent 运行安全。",
    category: "security",
    installed: true,
    favorites: 3120,
    downloads: 720000,
    source: "ClawHub",
  },
  {
    id: "content-guard",
    name: "Content Guard",
    initial: "C",
    color: SKILL_BRAND_COLORS.orange,
    description: "内容合规审查引擎，实时检测 Agent 输出中的敏感信息、违规内容和隐私泄露风险，支持自定义规则。",
    category: "security",
    installed: false,
    favorites: 456,
    downloads: 67000,
    source: "SkillHub",
  },
  // ── 通讯协作 ──
  {
    id: "wecom-bot",
    name: "企业微信 Bot",
    initial: "企",
    color: SKILL_BRAND_COLORS.brightBlue,
    description: "企业微信深度集成，支持群消息推送、审批流转、日程同步和自动回复，适合企业内部协作场景。",
    category: "collab",
    installed: false,
    favorites: 1890,
    downloads: 423000,
    source: "ClawHub",
  },
  {
    id: "feishu-connector",
    name: "飞书 Connector",
    initial: "飞",
    color: SKILL_BRAND_COLORS.blue,
    description: "飞书平台连接器，支持文档协作、多维表格读写、机器人消息和审批流程自动化。",
    category: "collab",
    installed: false,
    favorites: 2100,
    downloads: 389000,
    source: "SkillHub",
  },
];

// ─── 已配置徽标 ──────────────────────────────────────────────────────────
function ConfiguredBadge() {
  return (
    <span
      className="inline-flex items-center gap-1.5 h-5 px-2 text-xs shrink-0"
      style={{ color: "var(--text-success)", letterSpacing: "0.015em" }}
    >
      <span className="w-2 h-2 rounded-full" style={{ background: "var(--text-success)" }} />
      已配置
    </span>
  );
}

// ─── 类型定义：workspace / 项目 / 技能 / 规范 ───────────────────────────────

type IdeType = "codebuddy" | "workbuddy";

type LocalAgentSkill = {
  id: string;
  name: string;
  version: string;
  description?: string;
  distributeStatus: "distributing" | "distributed" | "failed";
};

type SpecItem = {
  id: string;
  name: string;
  path: string;
  ideType: IdeType;
  type: "system-prompt" | "rule";
  status: "已生效" | "未生效";
  updatedAt: string;
  distributeStatus: "distributing" | "distributed" | "failed";
};

type LocalAgentMcp = {
  id: string;
  name: string;
  description?: string;
  /** 连接类型：stdio=本地命令，sse/streamable-http=远程服务 */
  transportType: "stdio" | "sse" | "streamable-http";
  /** 该 MCP 提供的工具数量 */
  toolCount: number;
  distributeStatus: "distributing" | "distributed" | "failed";
};

type WorkspaceProject = {
  id: string;
  name: string;
  isPrimary: boolean;
};

type WorkspaceProjectBindingStatus = "bound" | "removed" | "missing";

type Workspace = {
  id: string;
  name: string;
  path: string;
  projectId?: string;
  projectName?: string;
  projectBindingStatus: WorkspaceProjectBindingStatus;
  ideType: IdeType;
  skills: LocalAgentSkill[];
  specs: SpecItem[];
  mcps: LocalAgentMcp[];
};

type LocalAgentResourceSet = {
  userSkills: LocalAgentSkill[];
  userSpecs: SpecItem[];
  userMcps: LocalAgentMcp[];
  workspaces: Workspace[];
};

const IDE_LABELS: Record<IdeType, string> = {
  codebuddy: "CodeBuddy",
  workbuddy: "WorkBuddy",
};

// ─── Mock 数据：项目 ───────────────────────────────────────────────────────

const MOCK_PROJECTS: WorkspaceProject[] = [
  { id: "p1", name: "ClawPro 企业管理项目", isPrimary: true },
  { id: "p2", name: "异构计算项目", isPrimary: false },
  { id: "p3", name: "安全合规项目", isPrimary: false },
];

// ─── Mock 数据：用户级技能 ─────────────────────────────────────────────────

const USER_LEVEL_SKILLS: LocalAgentSkill[] = [
  { id: "us-1", name: "code-interpreter", version: "1.2.0", description: "代码执行、脚本验证和本地计算任务处理。", distributeStatus: "distributed" },
  { id: "us-2", name: "image-recognition", version: "0.9.1", description: "识别图片内容并提取可用于上下文的关键信息。", distributeStatus: "distributed" },
  { id: "us-3", name: "text-to-speech", version: "1.0.0", description: "将文本内容转为语音，适用于个人工作流播报。", distributeStatus: "distributed" },
  { id: "us-4", name: "pdf-parser", version: "1.1.0", description: "解析 PDF 文件内容，提取结构化文本。", distributeStatus: "distributed" },
  { id: "us-5", name: "excel-reader", version: "2.0.0", description: "读取 Excel 表格并辅助分析字段和数据。", distributeStatus: "distributing" },
  { id: "us-6", name: "web-scraper", version: "1.3.2", description: "抓取网页内容并转换为 Agent 可处理的上下文。", distributeStatus: "failed" },
];

// ─── Mock 数据：用户级规范 ─────────────────────────────────────────────────
// codebuddy: .codebuddy/CODEBUDDY.md + .codebuddy/rules/{slug}.md
// workbuddy: .workbuddy/WORKBUDDY.md + .workbuddy/rules/{slug}.md（假设与 codebuddy 对称）

const USER_LEVEL_SPECS: SpecItem[] = [
  { id: "us-spec-1", name: "CODEBUDDY.md", path: ".codebuddy/CODEBUDDY.md", ideType: "codebuddy", type: "system-prompt", status: "已生效", updatedAt: "2026-06-20", distributeStatus: "distributed" },
  { id: "us-spec-2", name: "分支规范", path: ".codebuddy/rules/branch.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-21", distributeStatus: "distributed" },
  { id: "us-spec-3", name: "代码评审", path: ".codebuddy/rules/review.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-19", distributeStatus: "distributed" },
  { id: "us-spec-4", name: "敏感信息限制", path: ".codebuddy/rules/sensitive.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-16", distributeStatus: "distributing" },
  { id: "us-spec-5", name: "WORKBUDDY.md", path: ".workbuddy/WORKBUDDY.md", ideType: "workbuddy", type: "system-prompt", status: "已生效", updatedAt: "2026-06-18", distributeStatus: "distributed" },
  { id: "us-spec-6", name: "提交规范", path: ".workbuddy/rules/commit.md", ideType: "workbuddy", type: "rule", status: "已生效", updatedAt: "2026-06-17", distributeStatus: "distributed" },
];

// ─── Mock 数据：用户级 MCP ─────────────────────────────────────────────────
// 企业为该用户下发的 MCP 服务，本地 Agent 自动同步到用户级配置

// 演示视角：mock 当前登录账号为管理员，包含系统自带 ClawPro 平台 MCP。
// 用户端角色的展示应过滤掉 id === "us-mcp-clawpro-platform" 的条目（当前 mock 不做角色切换）。
const USER_LEVEL_MCPS: LocalAgentMcp[] = [
  { id: "us-mcp-clawpro-platform", name: "ClawPro 平台 MCP", description: "管理平台自身：管控端 41 + 用户端 12（管理员是超集，全 53 个工具可用）。", transportType: "streamable-http", toolCount: 53, distributeStatus: "distributed" },
  { id: "us-mcp-1", name: "iWiki 文档服务", description: "连接 iWiki 知识库，支持文档搜索、内容获取。", transportType: "streamable-http", toolCount: 3, distributeStatus: "distributed" },
  { id: "us-mcp-2", name: "工蜂 MCP 服务", description: "连接工蜂代码仓库，支持代码搜索、文件浏览、PR 管理。", transportType: "sse", toolCount: 10, distributeStatus: "distributed" },
  { id: "us-mcp-3", name: "TAPD 项目管理", description: "连接 TAPD，支持需求查询、缺陷管理、迭代跟踪。", transportType: "sse", toolCount: 6, distributeStatus: "distributing" },
  { id: "us-mcp-4", name: "COS 对象存储", description: "访问腾讯云 COS 存储桶，支持文件上传、下载、列表。", transportType: "sse", toolCount: 5, distributeStatus: "failed" },
];

// ─── Mock 数据：项目级 MCP（按 workspace id 映射） ─────────────────────────

const WORKSPACE_MCPS: Record<string, LocalAgentMcp[]> = {
  "ws-1": [
    { id: "ws1-mcp-1", name: "工蜂 MCP 服务", description: "代码仓库集成", transportType: "sse", toolCount: 10, distributeStatus: "distributed" },
    { id: "ws1-mcp-2", name: "MySQL MCP", description: "数据库查询与结构分析", transportType: "stdio", toolCount: 4, distributeStatus: "distributed" },
  ],
  "ws-2": [
    { id: "ws2-mcp-1", name: "腾讯云 CVM MCP", description: "云服务器实例管理", transportType: "streamable-http", toolCount: 8, distributeStatus: "distributed" },
    { id: "ws2-mcp-2", name: "COS 对象存储", description: "对象存储读写", transportType: "sse", toolCount: 5, distributeStatus: "distributing" },
  ],
  "ws-3": [
    { id: "ws3-mcp-1", name: "iWiki 文档服务", description: "文档检索与内容获取", transportType: "streamable-http", toolCount: 3, distributeStatus: "distributed" },
  ],
  "ws-4": [
    { id: "ws4-mcp-1", name: "WeData 数据开发", description: "数据开发治理平台集成", transportType: "streamable-http", toolCount: 7, distributeStatus: "distributed" },
  ],
  "ws-5": [],
};

// ─── Mock 数据：项目级 Workspace ───────────────────────────────────────────

const MOCK_WORKSPACES: Omit<Workspace, "mcps">[] = [
  {
    id: "ws-1",
    name: "clawpro项目",
    path: "/Users/petzhou/CodeBuddy/clawpro",
    projectId: "p1",
    projectName: "ClawPro 企业管理项目",
    projectBindingStatus: "bound",
    ideType: "codebuddy",
    skills: [
      { id: "ws1-s1", name: "json-formatter", version: "0.8.0", description: "格式化接口响应、配置文件和调试日志。", distributeStatus: "distributed" },
      { id: "ws1-s2", name: "markdown-converter", version: "1.0.1", description: "把 Markdown 内容转换为页面预览或文档素材。", distributeStatus: "distributed" },
      { id: "ws1-s3", name: "api-tester", version: "2.1.0", description: "辅助调试接口请求、参数和响应结构。", distributeStatus: "distributed" },
      { id: "ws1-s4", name: "sql-executor", version: "1.5.0", description: "执行 SQL 查询并整理结果。", distributeStatus: "distributing" },
      { id: "ws1-s5", name: "file-compressor", version: "0.9.0", description: "压缩项目产物和临时交付文件。", distributeStatus: "distributed" },
      { id: "ws1-s6", name: "email-sender", version: "1.2.3", description: "发送项目通知、验收结论和待办提醒。", distributeStatus: "failed" },
    ],
    specs: [
      { id: "ws1-spec-1", name: "CODEBUDDY.md", path: ".codebuddy/CODEBUDDY.md", ideType: "codebuddy", type: "system-prompt", status: "已生效", updatedAt: "2026-06-21", distributeStatus: "distributed" },
      { id: "ws1-spec-2", name: "分支规范", path: ".codebuddy/rules/branch.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-21", distributeStatus: "distributed" },
      { id: "ws1-spec-3", name: "导航变更约束", path: ".codebuddy/rules/nav-change.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-21", distributeStatus: "distributing" },
      { id: "ws1-spec-4", name: "页面验收规则", path: ".codebuddy/rules/acceptance.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-19", distributeStatus: "distributed" },
    ],
  },
  {
    id: "ws-2",
    name: "cvm项目",
    path: "/Users/petzhou/CodeBuddy/cvm",
    projectId: "p1",
    projectName: "ClawPro 企业管理项目",
    projectBindingStatus: "bound",
    ideType: "codebuddy",
    skills: [
      { id: "ws2-s1", name: "dns-lookup", version: "0.6.0", description: "查询域名解析结果，辅助定位网络访问问题。", distributeStatus: "distributed" },
      { id: "ws2-s2", name: "port-scanner", version: "1.2.0", description: "检查端口连通性和服务监听状态。", distributeStatus: "distributed" },
      { id: "ws2-s3", name: "ssl-checker", version: "1.0.0", description: "校验证书有效期和 HTTPS 配置。", distributeStatus: "distributed" },
      { id: "ws2-s4", name: "ping-monitor", version: "0.9.0", description: "持续探测实例连通性并输出异常摘要。", distributeStatus: "distributed" },
      { id: "ws2-s5", name: "git-helper", version: "2.1.0", description: "辅助拉取仓库、分支和提交上下文。", distributeStatus: "distributed" },
      { id: "ws2-s6", name: "docker-manager", version: "1.3.0", description: "处理容器镜像、运行状态和基础操作。", distributeStatus: "distributed" },
    ],
    specs: [
      { id: "ws2-spec-1", name: "CODEBUDDY.md", path: ".codebuddy/CODEBUDDY.md", ideType: "codebuddy", type: "system-prompt", status: "已生效", updatedAt: "2026-06-13", distributeStatus: "distributed" },
      { id: "ws2-spec-2", name: "变更窗口规则", path: ".codebuddy/rules/change-window.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-13", distributeStatus: "distributed" },
      { id: "ws2-spec-3", name: "实例操作确认", path: ".codebuddy/rules/instance-op.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-11", distributeStatus: "distributed" },
      { id: "ws2-spec-4", name: "安全组变更", path: ".codebuddy/rules/security-group.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-08", distributeStatus: "distributed" },
    ],
  },
  {
    id: "ws-3",
    name: "接管本地虾-用户端",
    path: "/Users/petzhou/CodeBuddy/接管本地虾-用户端",
    projectId: "p1",
    projectName: "ClawPro 企业管理项目",
    projectBindingStatus: "bound",
    ideType: "codebuddy",
    skills: [
      { id: "ws3-s1", name: "code-explorer", version: "1.0.0", description: "跨文件代码搜索与符号定位。", distributeStatus: "distributed" },
      { id: "ws3-s2", name: "refactor-helper", version: "0.8.2", description: "辅助重构、提取函数和重命名符号。", distributeStatus: "distributed" },
      { id: "ws3-s3", name: "test-generator", version: "1.1.0", description: "根据函数签名自动生成测试用例骨架。", distributeStatus: "distributing" },
      { id: "ws3-s4", name: "doc-writer", version: "2.0.1", description: "根据代码变更生成或更新注释和文档。", distributeStatus: "distributed" },
    ],
    specs: [
      { id: "ws3-spec-1", name: "CODEBUDDY.md", path: ".codebuddy/CODEBUDDY.md", ideType: "codebuddy", type: "system-prompt", status: "已生效", updatedAt: "2026-07-01", distributeStatus: "distributed" },
      { id: "ws3-spec-2", name: "前端编码规范", path: ".codebuddy/rules/frontend.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-07-01", distributeStatus: "distributed" },
      { id: "ws3-spec-3", name: "组件使用约束", path: ".codebuddy/rules/component.md", ideType: "codebuddy", type: "rule", status: "已生效", updatedAt: "2026-06-28", distributeStatus: "distributed" },
    ],
  },
  {
    id: "ws-4",
    name: "异构项目",
    path: "/Users/petzhou/CodeBuddy/hetero",
    projectId: "p2",
    projectName: "异构计算项目",
    projectBindingStatus: "removed",
    ideType: "workbuddy",
    skills: [
      { id: "ws4-s1", name: "log-analyzer", version: "2.0.0", description: "分析日志中的资源申请、调度和失败原因。", distributeStatus: "distributed" },
      { id: "ws4-s2", name: "data-visualizer", version: "1.4.0", description: "生成资源使用趋势图和对比图。", distributeStatus: "distributed" },
      { id: "ws4-s3", name: "csv-processor", version: "0.6.0", description: "处理 CSV 格式的资源指标明细。", distributeStatus: "distributed" },
      { id: "ws4-s4", name: "translation-engine", version: "3.0.1", description: "翻译英文错误信息和平台文档片段。", distributeStatus: "distributed" },
    ],
    specs: [
      { id: "ws4-spec-1", name: "WORKBUDDY.md", path: ".workbuddy/WORKBUDDY.md", ideType: "workbuddy", type: "system-prompt", status: "已生效", updatedAt: "2026-06-15", distributeStatus: "distributed" },
      { id: "ws4-spec-2", name: "资源申请规则", path: ".workbuddy/rules/resource-apply.md", ideType: "workbuddy", type: "rule", status: "已生效", updatedAt: "2026-06-15", distributeStatus: "distributed" },
      { id: "ws4-spec-3", name: "GPU队列使用", path: ".workbuddy/rules/gpu-queue.md", ideType: "workbuddy", type: "rule", status: "已生效", updatedAt: "2026-06-12", distributeStatus: "distributed" },
    ],
  },
  {
    id: "ws-5",
    name: "未绑定演示工作区",
    path: "/Users/petzhou/WorkBuddy/unbound-demo",
    projectBindingStatus: "missing",
    ideType: "workbuddy",
    skills: [
      { id: "ws5-s1", name: "note-summarizer", version: "1.0.0", description: "整理会议纪要和项目待办。", distributeStatus: "distributed" },
      { id: "ws5-s2", name: "task-extractor", version: "0.7.2", description: "从文本中提取任务、负责人和截止时间。", distributeStatus: "distributed" },
    ],
    specs: [
      { id: "ws5-spec-1", name: "WORKBUDDY.md", path: ".workbuddy/WORKBUDDY.md", ideType: "workbuddy", type: "system-prompt", status: "已生效", updatedAt: "2026-07-08", distributeStatus: "distributed" },
    ],
  },
];

const markDistributingAsDistributed = <T extends { distributeStatus: "distributing" | "distributed" | "failed" }>(item: T): T =>
  item.distributeStatus === "distributing" ? { ...item, distributeStatus: "distributed" } : item;

function buildLocalAgentResources(hasDistributingMock: boolean): LocalAgentResourceSet {
  if (hasDistributingMock) {
    return {
      userSkills: USER_LEVEL_SKILLS,
      userSpecs: USER_LEVEL_SPECS,
      userMcps: USER_LEVEL_MCPS,
      workspaces: MOCK_WORKSPACES.map((workspace) => ({
        ...workspace,
        mcps: WORKSPACE_MCPS[workspace.id] ?? [],
      })),
    };
  }

  return {
    userSkills: USER_LEVEL_SKILLS.map(markDistributingAsDistributed),
    userSpecs: USER_LEVEL_SPECS.map(markDistributingAsDistributed),
    userMcps: USER_LEVEL_MCPS.map(markDistributingAsDistributed),
    workspaces: MOCK_WORKSPACES.map((workspace) => ({
      ...workspace,
      skills: workspace.skills.map(markDistributingAsDistributed),
      specs: workspace.specs.map(markDistributingAsDistributed),
      mcps: (WORKSPACE_MCPS[workspace.id] ?? []).map(markDistributingAsDistributed),
    })),
  };
}

function hasDistributingLocalResources(resources: LocalAgentResourceSet) {
  return (
    resources.userSkills.some((skill) => skill.distributeStatus === "distributing") ||
    resources.userSpecs.some((spec) => spec.distributeStatus === "distributing") ||
    resources.userMcps.some((mcp) => mcp.distributeStatus === "distributing") ||
    resources.workspaces.some((workspace) =>
      workspace.skills.some((skill) => skill.distributeStatus === "distributing") ||
      workspace.specs.some((spec) => spec.distributeStatus === "distributing") ||
      workspace.mcps.some((mcp) => mcp.distributeStatus === "distributing")
    )
  );
}

// ─── 技能 Chip 列表（高信息密度） ──────────────────────────────────────────

function DistributeStatusBadge({ status }: { status: "distributing" | "distributed" | "failed" }) {
  if (status === "distributed") return null;
  const isDistributing = status === "distributing";
  return (
    <TooltipProvider delayDuration={0}>
      <Tooltip>
        <TooltipTrigger asChild>
          <span
            className={`inline-flex items-center gap-0.5 ${isDistributing ? "text-[var(--text-brand)]" : "text-[var(--text-danger)]"}`}
          >
            {isDistributing ? <Loader2 className="w-3 h-3 animate-spin" /> : <CircleAlert className="w-3 h-3" />}
            {isDistributing ? "下发中" : "下发失败"}
          </span>
        </TooltipTrigger>
        <TooltipContent side="top" className="text-xs">
          {isDistributing
            ? "本地发起对话，资源自动同步"
            : "本地 Agent 安装失败，请检查插件状态后重试"}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function SkillChipList({ skills }: { skills: LocalAgentSkill[] }) {
  if (skills.length === 0) {
    return (
      <div className="flex min-h-[40px] items-center text-xs text-[var(--text-muted)]">
        暂无技能
      </div>
    );
  }

  return (
    <div className="flex flex-wrap gap-2">
      {skills.map((skill) => {
        const isDistributing = skill.distributeStatus === "distributing";
        const isFailed = skill.distributeStatus === "failed";
        return (
          <span
            key={skill.id}
            title={skill.description || skill.name}
            className={`inline-flex items-center gap-1.5 rounded-md border bg-[var(--bg-grey-normal)] px-2.5 py-1 text-xs transition-colors cursor-default ${
              isFailed
                ? "border-[var(--text-danger)] hover:border-[var(--text-danger)]"
                : isDistributing
                  ? "border-[var(--border)] hover:border-[var(--border)]"
                  : "border-[var(--border)] hover:border-[var(--text-brand)]"
            }`}
            style={{ color: "var(--foreground)" }}
          >
            <span className="font-medium">{skill.name}</span>
            <span className="text-[var(--text-weak)]">{skill.version}</span>
            <DistributeStatusBadge status={skill.distributeStatus} />
          </span>
        );
      })}
    </div>
  );
}

// ─── 规范 Chip 列表（高信息密度） ──────────────────────────────────────────

function SpecChipList({ specs }: { specs: SpecItem[] }) {
  if (specs.length === 0) {
    return (
      <div className="flex min-h-[40px] items-center text-xs text-[var(--text-muted)]">
        暂无规范
      </div>
    );
  }

  return (
    <div className="flex flex-wrap gap-2">
      {specs.map((spec) => {
        const isDistributing = spec.distributeStatus === "distributing";
        const isFailed = spec.distributeStatus === "failed";
        return (
          <span
            key={spec.id}
            title={`${spec.path}\n状态：${spec.status}\n更新：${spec.updatedAt}`}
            className={`inline-flex items-center gap-1.5 rounded-md border bg-[var(--bg-grey-normal)] px-2.5 py-1 text-xs transition-colors cursor-default ${
              isFailed
                ? "border-[var(--text-danger)] hover:border-[var(--text-danger)]"
                : isDistributing
                  ? "border-[var(--border)] hover:border-[var(--border)]"
                  : "border-[var(--border)] hover:border-[var(--text-brand)]"
            }`}
            style={{ color: "var(--foreground)" }}
          >
            <span className="font-medium">{spec.name}</span>
            <span className="text-[var(--text-weak)]">{IDE_LABELS[spec.ideType]}</span>
            <DistributeStatusBadge status={spec.distributeStatus} />
          </span>
        );
      })}
    </div>
  );
}

// ─── MCP Chip 列表（高信息密度） ───────────────────────────────────────────

const MCP_TRANSPORT_LABELS: Record<LocalAgentMcp["transportType"], string> = {
  stdio: "本地命令",
  sse: "远程服务",
  "streamable-http": "远程服务",
};

function McpChipList({ mcps }: { mcps: LocalAgentMcp[] }) {
  if (mcps.length === 0) {
    return (
      <div className="flex min-h-[40px] items-center text-xs text-[var(--text-muted)]">
        暂无 MCP
      </div>
    );
  }

  return (
    <div className="flex flex-wrap gap-2">
      {mcps.map((mcp) => {
        const isDistributing = mcp.distributeStatus === "distributing";
        const isFailed = mcp.distributeStatus === "failed";
        return (
          <span
            key={mcp.id}
            title={`${mcp.description || mcp.name}\n类型：${MCP_TRANSPORT_LABELS[mcp.transportType]}\n工具数：${mcp.toolCount}`}
            className={`inline-flex items-center gap-1.5 rounded-md border bg-[var(--bg-grey-normal)] px-2.5 py-1 text-xs transition-colors cursor-default ${
              isFailed
                ? "border-[var(--text-danger)] hover:border-[var(--text-danger)]"
                : isDistributing
                  ? "border-[var(--border)] hover:border-[var(--border)]"
                  : "border-[var(--border)] hover:border-[var(--text-brand)]"
            }`}
            style={{ color: "var(--foreground)" }}
          >
            <span className="font-medium">{mcp.name}</span>
            <span className="text-[var(--text-weak)]">{mcp.toolCount} 个工具</span>
            <DistributeStatusBadge status={mcp.distributeStatus} />
          </span>
        );
      })}
    </div>
  );
}

// ─── 项目绑定标签 ──────────────────────────────────────────────────────────

function ProjectBindingTag({
  projectName,
  status,
}: {
  projectName?: string;
  status: WorkspaceProjectBindingStatus;
}) {
  if (status === "missing" || !projectName) {
    return (
      <span
        className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[11px]"
        style={{ background: "var(--muted)", color: "var(--text-muted)" }}
      >
        <CircleAlert className="w-3 h-3" />
        未绑定/项目不存在
      </span>
    );
  }

  if (status === "removed") {
    return (
      <span
        className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[11px]"
        style={{ background: "rgba(237, 137, 54, 0.1)", color: "#ed8936" }}
      >
        <CircleAlert className="w-3 h-3" />
        项目：{projectName}（被移出）
      </span>
    );
  }

  return (
    <span
      className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[11px]"
      style={{ background: "var(--accent)", color: "var(--text-brand)" }}
    >
      项目：{projectName}
    </span>
  );
}

// ─── 资源区块标题 ──────────────────────────────────────────────────────────

const SPEC_TOOLTIP_CONTENT = (
  <div className="flex flex-col gap-2 text-xs">
    <div>
      <span className="font-medium">CodeBuddy</span>
      <div className="mt-1 flex flex-col gap-1 text-[var(--text-muted)]">
        <span>• System Prompt：.codebuddy/CODEBUDDY.md（或项目根目录 CODEBUDDY.md），初始化时自动创建，帮助 AI 快速了解项目上下文</span>
        <span>• Rules：.codebuddy/rules/&#123;slug&#125;.md，项目级编码规范，受版本控制管理，可团队共享</span>
      </div>
    </div>
    <div>
      <span className="font-medium">WorkBuddy</span>
      <div className="mt-1 text-[var(--text-muted)]">
        <span>• 在 workbuddy 目录下新建 rules 文件夹存放规范文件（WorkBuddy 本身不内置规则系统）</span>
      </div>
    </div>
  </div>
);

function ResourceSectionLabel({ label, count, tooltip }: { label: string; count: number; tooltip?: ReactNode }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-medium text-[var(--text-body)]">{label}</span>
      <span className="text-xs text-[var(--text-weak)]">（{count}）</span>
      {tooltip && (
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="inline-flex cursor-help" aria-label="规范说明">
                <Info className="w-3.5 h-3.5 text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors" />
              </span>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-[360px]">
              {tooltip}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      )}
    </div>
  );
}

// ─── Workspace 展开行 ─────────────────────────────────────────────────────

function WorkspaceRow({ workspace }: { workspace: Workspace }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border-b border-[var(--border)] last:border-b-0">
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-[var(--accent)]"
      >
        <ChevronDown
          className={`w-4 h-4 shrink-0 text-[var(--text-weak)] transition-transform ${expanded ? "" : "-rotate-90"}`}
        />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm font-medium text-[var(--text-emphasis)]">{workspace.name}</span>
            <ProjectBindingTag projectName={workspace.projectName} status={workspace.projectBindingStatus} />
          </div>
          <div className="mt-1 flex items-center gap-3 text-xs text-[var(--text-weak)]">
            <span className="truncate font-mono">{workspace.path}</span>
            <span className="shrink-0">{IDE_LABELS[workspace.ideType]}</span>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-3 text-xs text-[var(--text-weak)]">
          <span>技能 {workspace.skills.length}</span>
          <span>规范 {workspace.specs.length}</span>
          <span>MCP {workspace.mcps.length}</span>
        </div>
      </button>

      {expanded && (
        <div className="border-t border-[var(--border)] bg-[var(--bg-grey-normal)] px-4 py-4">
          <div className="flex flex-col gap-4">
            <div>
              <ResourceSectionLabel label="已安装 Skill" count={workspace.skills.length} />
              <div className="mt-2">
                <SkillChipList skills={workspace.skills} />
              </div>
            </div>
            <div>
              <ResourceSectionLabel label="已安装规范" count={workspace.specs.length} tooltip={SPEC_TOOLTIP_CONTENT} />
              <div className="mt-2">
                <SpecChipList specs={workspace.specs} />
              </div>
            </div>
            <div>
              <ResourceSectionLabel label="已安装 MCP" count={workspace.mcps.length} />
              <div className="mt-2">
                <McpChipList mcps={workspace.mcps} />
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ─── 本地 Agent 设置面板（用户级 + 项目级资源） ────────────────────────────

function LocalAgentSettingsPanel({
  resources,
  organizationName,
}: {
  resources: LocalAgentResourceSet;
  organizationName: string;
}) {
  const hasDistributingResources = hasDistributingLocalResources(resources);

  return (
    <div className="flex w-full flex-col gap-6">
      {hasDistributingResources && (
        <Alert variant="info">
          <AlertInfoIcon />
          <AlertDescription>
            本地发起对话，资源自动同步
          </AlertDescription>
        </Alert>
      )}

      {/* ── 用户级资源 ── */}
      <TenantCard padding="none" className="overflow-hidden">
        <section className="p-6">
          <div className="mb-5">
            <div className="flex items-center gap-2">
              <span className="truncate text-[18px] font-medium leading-6 text-[var(--text-emphasis)]">用户级资源</span>
            </div>
            <p className="mt-1 text-sm leading-5 text-[var(--text-muted)]">
              用户级资源跟随当前用户生效，不依赖具体工作空间；本地 Agent 会同步企业为该用户下发的 Skill、规范和 MCP。
            </p>
          </div>

          <div className="mb-5">
            <div className="flex items-center gap-2">
              <span className="text-sm text-[var(--text-weak)]">组织</span>
              <span className="text-sm text-[var(--text-body)]">{organizationName || "默认"}</span>
            </div>
          </div>

          <div className="flex flex-col gap-5">
            <div>
              <ResourceSectionLabel label="已安装 Skill" count={resources.userSkills.length} />
              <div className="mt-2">
                <SkillChipList skills={resources.userSkills} />
              </div>
            </div>
            <div>
              <ResourceSectionLabel label="已安装规范" count={resources.userSpecs.length} tooltip={SPEC_TOOLTIP_CONTENT} />
              <div className="mt-2">
                <SpecChipList specs={resources.userSpecs} />
              </div>
            </div>
            <div>
              <ResourceSectionLabel label="已安装 MCP" count={resources.userMcps.length} />
              <div className="mt-2">
                <McpChipList mcps={resources.userMcps} />
              </div>
            </div>
          </div>
        </section>
      </TenantCard>

      {/* ── 项目级资源 ── */}
      <TenantCard padding="none" className="overflow-hidden">
        <section className="p-6">
          <div className="mb-5">
            <div className="flex items-center gap-2">
              <span className="text-[18px] font-medium leading-6 text-[var(--text-emphasis)]">项目级资源</span>
            </div>
            <p className="mt-1 text-sm leading-5 text-[var(--text-muted)]">
              管理各工作空间（workspace）绑定项目下发的技能、规范和 MCP，点击展开查看详情。
            </p>
          </div>

          <div className="overflow-hidden rounded-[var(--radius-card)] border border-[var(--border)] bg-white">
            {resources.workspaces.map((ws) => (
              <WorkspaceRow key={ws.id} workspace={ws} />
            ))}
          </div>
        </section>
      </TenantCard>
    </div>
  );
}

function ExternalAgentResourcesPanel({
  isInactive,
  organizationName,
  resources,
}: {
  isInactive: boolean;
  organizationName: string;
  resources: LocalAgentResourceSet;
}) {
  const skills = [
    ...resources.userSkills,
    ...resources.workspaces.flatMap((workspace) => workspace.skills),
  ];
  const specs = [
    ...resources.userSpecs,
    ...resources.workspaces.flatMap((workspace) => workspace.specs),
  ];
  const mcps = [
    ...resources.userMcps,
    ...resources.workspaces.flatMap((workspace) => workspace.mcps),
  ];

  return (
    <TenantCard padding="none" className="overflow-hidden">
      <section className="p-6">
        <div className="mb-5 flex items-center gap-2">
          <span className="truncate text-[18px] font-medium leading-6 text-[var(--text-emphasis)]">外部 Agent 资源</span>
          {isInactive && (
            <span
              className="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
              style={{
                background: "var(--alert-warning-bg, #FEF3C7)",
                color: "var(--text-warning, #B45309)",
              }}
            >
              不活跃
            </span>
          )}
        </div>

        <div className="mb-5 flex items-center gap-2">
          <span className="text-sm text-[var(--text-weak)]">组织</span>
          <span className="text-sm text-[var(--text-body)]">{organizationName || "默认"}</span>
        </div>

        <div className="flex flex-col gap-5">
          <div>
            <ResourceSectionLabel label="已安装 Skill" count={skills.length} />
            <div className="mt-2">
              <SkillChipList skills={skills} />
            </div>
          </div>
          <div>
            <ResourceSectionLabel label="已安装规范" count={specs.length} tooltip={SPEC_TOOLTIP_CONTENT} />
            <div className="mt-2">
              <SpecChipList specs={specs} />
            </div>
          </div>
          <div>
            <ResourceSectionLabel label="已安装 MCP" count={mcps.length} />
            <div className="mt-2">
              <McpChipList mcps={mcps} />
            </div>
          </div>
        </div>
      </section>
    </TenantCard>
  );
}

// ─── 模型选择行 ──────────────────────────────────────────────────────────
function ModelRow({
  provider,
  model,
  showEdit,
  showDelete,
}: {
  provider: string;
  model: string;
  showEdit?: boolean;
  showDelete?: boolean;
}) {
  return (
    <div
      className="flex items-center justify-between rounded-[var(--radius-card)] px-4 py-3.5"
      style={{ background: "var(--bg-grey-normal)", border: "1px solid var(--border)" }}
    >
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium" style={{ color: "rgba(0,0,0,0.9)" }}>
          {provider}
        </span>
        <span className="text-xs" style={{ color: "rgba(0,0,0,0.3)" }}>
          {model}
        </span>
      </div>
      {showEdit && (
        <button
          className="text-[var(--muted-foreground)] hover:text-[var(--text-brand)] transition-colors"
          aria-label="编辑模型"
          onClick={() => toast.info("编辑模型（demo）")}
        >
          <Edit3 className="w-4 h-4" />
        </button>
      )}
      {showDelete && (
        <button
          className="text-[var(--muted-foreground)] hover:text-[var(--text-danger)] transition-colors"
          aria-label="删除模型"
          onClick={() => toast.info("删除模型（demo）")}
        >
          <Trash2 className="w-4 h-4" />
        </button>
      )}
    </div>
  );
}

// ─── 通道行 ──────────────────────────────────────────────────────────────
function ChannelRow({ name }: { name: string }) {
  return (
    <div
      className="flex items-center justify-between rounded-[var(--radius-card)] px-4 py-3.5"
      style={{ background: "var(--bg-grey-normal)", border: "1px solid var(--border)" }}
    >
      <div className="flex items-center gap-2">
        <ChevronDown className="w-4 h-4 text-[var(--muted-foreground)]" />
        <span className="text-xs font-medium" style={{ color: "rgba(0,0,0,0.9)" }}>
          {name}
        </span>
        <span
          className="inline-flex items-center gap-1.5 h-5 px-2 text-xs"
          style={{ color: "var(--text-success)" }}
        >
          <span className="w-2 h-2 rounded-full" style={{ background: "var(--text-success)" }} />
          运行中
        </span>
      </div>
      <button
        className="text-[var(--muted-foreground)] hover:text-[var(--text-danger)] transition-colors"
        aria-label="删除通道"
        onClick={() => toast.info(`删除通道 ${name}（demo）`)}
      >
        <Trash2 className="w-4 h-4" />
      </button>
    </div>
  );
}

// ─── 安装新技能弹窗 ──────────────────────────────────────────────────────
function SkillInstallModal({
  open,
  onOpenChange,
  onEnqueue,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onEnqueue?: (names: string[]) => void;
}) {
  const [category, setCategory] = useState<SkillCategory>("all");
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<"default" | "downloads" | "favorites">("default");
  const [addedSkills, setAddedSkills] = useState<string[]>([]);
  // ── 双场景开关：是否配置了 SkillHub（demo 默认 true，对齐截图） ──
  // A. true  → 显示 SkillHub 提示，搜索后列表才出卡片；不分类、不排序
  // B. false → 隐藏 SkillHub 提示，显示企业技能库全量列表 + 分类 + 排序 + 模糊搜索
  const [skillHubConfigured, setSkillHubConfigured] = useState(true);

  // B 模式：模糊匹配 + 分类 + 排序
  const filteredSkills = SKILL_LIBRARY.filter((s) => {
    if (category !== "all" && s.category !== category) return false;
    if (search && !s.name.toLowerCase().includes(search.toLowerCase())) return false;
    return true;
  }).sort((a, b) => {
    if (sortBy === "downloads") return b.downloads - a.downloads;
    if (sortBy === "favorites") return b.favorites - a.favorites;
    // 综合排序：收藏权重 + 下载权重
    return (b.favorites * 10 + b.downloads) - (a.favorites * 10 + a.downloads);
  });

  // A 模式：SkillHub「不支持模糊搜索」≈ 不允许任意子串，但允许前缀匹配作为输入辅助
  // 大小写不敏感、去首尾空格；最多返回 5 条，避免变成"全量浏览"
  const skillHubSearchResults = (() => {
    const q = search.trim().toLowerCase();
    if (!q) return [];
    return SKILL_LIBRARY
      .filter((s) => s.name.toLowerCase().startsWith(q))
      .slice(0, 5);
  })();

  const handleAdd = (skillId: string) => {
    setAddedSkills((prev) => [...prev, skillId]);
    toast.success("技能已添加到安装列表");
  };

  const handleInstall = () => {
    if (addedSkills.length === 0) {
      toast.warning("请先添加要安装的技能");
      return;
    }
    // 把勾选的技能加入待安装队列（含 name + 默认版本号 1.0.0）
    const names = addedSkills
      .map((id) => SKILL_LIBRARY.find((s) => s.id === id))
      .filter((s): s is (typeof SKILL_LIBRARY)[number] => !!s)
      .map((s) => `${s.name} 1.0.0`);
    onEnqueue?.(names);
    toast.success(`已加入安装队列：${addedSkills.length} 个技能`);
    onOpenChange(false);
    setAddedSkills([]);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[780px] p-0 overflow-hidden" showCloseButton={false}>
        {/* 弹窗头部 */}
        <div className="flex items-center justify-between px-6 pt-5 pb-4">
          <DialogHeader className="p-0 m-0 gap-0 space-y-0">
            <DialogTitle className="text-base font-semibold text-[var(--text-title)]">
              安装新技能
            </DialogTitle>
            <DialogDescription className="sr-only">
              从技能库中搜索、筛选并安装新的 Agent 技能。
            </DialogDescription>
          </DialogHeader>
          <div className="flex items-center gap-3">
            {/* Demo 开关：演示「配置了 SkillHub」与「未配置」两种场景（生产环境此开关来自管理员配置，非真实业务能力） */}
            <label
              className="flex items-center gap-2 cursor-pointer select-none"
              title="仅用于设计走查：切换演示「管理员是否配置 SkillHub」两种场景，非生产功能"
            >
              <span className="inline-flex items-center gap-1.5 rounded-full border border-dashed border-[var(--border)] bg-[var(--bg-grey-normal)] px-2 py-0.5 text-[10px] leading-4 text-[var(--text-muted)]">
                <span className="inline-block size-1.5 rounded-full bg-[var(--text-warning)]" aria-hidden />
                模拟数据
              </span>
              <span className="text-xs text-[var(--text-muted)]">SkillHub 模式</span>
              <Switch
                checked={skillHubConfigured}
                onCheckedChange={(checked) => {
                  setSkillHubConfigured(checked);
                  // 切换场景时重置搜索/分类，避免视觉错乱
                  setSearch("");
                  setCategory("all");
                }}
              />
            </label>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => onOpenChange(false)}
              className="h-7 w-7 text-[var(--text-weak)] hover:text-[var(--text-title)]"
              aria-label="关闭"
            >
              <X className="w-5 h-5" />
            </Button>
          </div>
        </div>

        {/* 管理员配置提示 */}
        <div className="px-6 pb-3 space-y-2">
          {skillHubConfigured && (
            <Alert variant="info">
              <AlertInfoIcon />
              <AlertDescription>
                管理员配置了
                <a
                  href="https://skillhub.tencent.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-[var(--text-brand)] hover:opacity-80 underline underline-offset-1 font-medium mx-1"
                >
                  SkillHub地址
                  <ExternalLink className="w-3 h-3 flex-shrink-0" />
                </a>
                ，不支持模糊搜索，请输入准确Skill名称
              </AlertDescription>
            </Alert>
          )}
          <Alert variant="warning">
            <AlertCircle className="w-4 h-4" />
            <AlertDescription>部分技能(Skills)可能存在安全风险，安装前请确认其安全性。</AlertDescription>
          </Alert>
        </div>

        {/* 当前添加 */}
        <div className="px-6 pb-4">
          {addedSkills.length === 0 ? (
            <div className="text-sm font-normal tracking-[0.07px] text-[var(--text-muted)]">
              暂未添加技能
            </div>
          ) : (
            <div className="w-full">
              <div className="text-sm font-normal tracking-[0.07px] text-[var(--text-title)]">
                <span>当前添加&nbsp;</span>
                <span className="font-medium">{addedSkills.length}</span>
                <span>&nbsp;个技能</span>
              </div>
              <div className="mt-1.5 flex h-6 flex-wrap gap-x-1 gap-y-2 overflow-hidden">
                {addedSkills.map((skillId) => {
                  const skill = SKILL_LIBRARY.find((s) => s.id === skillId);
                  if (!skill) return null;
                  return (
                    <Badge
                      key={skillId}
                      variant="outline"
                      className="h-6 justify-start gap-0 overflow-hidden rounded-full border-[var(--border)] bg-white px-0 py-0 text-[var(--text-title)]"
                    >
                      <span className="ml-2 max-w-[160px] truncate text-[12px] font-normal leading-5 tracking-[0.18px] text-[var(--text-title)]">
                        {skill.name}
                      </span>
                      <button
                        type="button"
                        onClick={() => setAddedSkills((prev) => prev.filter((id) => id !== skillId))}
                        className="ml-auto mr-2 inline-flex size-3 items-center justify-center text-[var(--text-muted)] transition-colors hover:text-[var(--text-title)]"
                        aria-label={`移除 ${skill.name}`}
                      >
                        <X className="size-3" />
                      </button>
                    </Badge>
                  );
                })}
              </div>
            </div>
          )}
        </div>

        {/* 搜索栏：A/B 都用同款输入框；B 模式额外加排序下拉 */}
        <div className="px-6 pb-2 flex items-center gap-3">
          <div className="relative flex-1">
            <Search
              className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 pointer-events-none text-[var(--text-weak)]"
            />
            <Input
              tenant
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={skillHubConfigured ? "请输入准确的 Skill 名称" : "输入 Skill 名称搜索并添加"}
              className="h-9 pl-9 text-sm"
            />
          </div>
          {!skillHubConfigured && (
            <Select value={sortBy} onValueChange={(v) => setSortBy(v as "default" | "downloads" | "favorites")}>
              <SelectTrigger tenant className="h-9 w-[120px] text-xs shrink-0">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="default">综合排序</SelectItem>
                <SelectItem value="downloads">下载量</SelectItem>
                <SelectItem value="favorites">收藏数</SelectItem>
              </SelectContent>
            </Select>
          )}
        </div>

        {/* 分类标签（仅 B 模式：未配 SkillHub 时显示企业技能库分类） */}
        {!skillHubConfigured && (
          <div className="px-6 pb-3">
            <div className="flex flex-wrap gap-2">
              {SKILL_CATEGORIES.map((cat) => (
                <Button key={cat.id} variant="tenant-plain" size="sm" data-state={category === cat.id ? "active" : undefined} onClick={() => setCategory(cat.id)}>
                  {cat.label}
                </Button>
              ))}
            </div>
          </div>
        )}

        {/* 列表区
             A 模式：未输入空态；有输入且命中前缀 → 显示同款卡片（最多 5 条）；无命中显示"未找到"
             B 模式：分类+排序后的本地技能库全量浏览 */}
        <div className="mx-6 mb-4 max-h-[340px] overflow-y-auto rounded-[var(--radius-card)] border border-[var(--border)] bg-white">
          {(() => {
            // ── A/B 共用 row 渲染 ──
            const renderSkillRow = (skill: (typeof SKILL_LIBRARY)[number]) => {
              const isAdded = addedSkills.includes(skill.id);
              return (
                <div
                  key={skill.id}
                  className="flex items-center gap-3 px-5 py-4 border-b border-gray-50 transition-colors last:border-b-0 hover:bg-[var(--accent)]"
                >
                  {/* 左：头像 */}
                  <div
                    className="w-9 h-9 shrink-0 self-start rounded-full flex items-center justify-center text-xs font-bold"
                    style={{ background: getLetterColor(skill.initial).bg, color: getLetterColor(skill.initial).text }}
                  >
                    {skill.initial}
                  </div>

                  {/* 中：标题 + 描述 + 指标 */}
                  <div className="flex-1 min-w-0">
                    <div className="truncate text-[14px] font-medium leading-[22px] text-[var(--text-title)]">
                      {skill.name}
                    </div>
                    <div className="mt-1 line-clamp-2 text-[12px] font-normal leading-5 tracking-[0.18px] text-[var(--text-muted)]">
                      {skill.description}
                    </div>
                    <div className="mt-2 flex items-center gap-3 text-xs font-normal leading-5 tracking-[0.18px] text-[var(--text-weak)]">
                      <span className="inline-flex items-center gap-1">
                        <Star className="size-3" />
                        {formatSkillCount(skill.favorites)}
                      </span>
                      <span className="inline-flex items-center gap-1">
                        <Download className="size-3" />
                        {formatSkillCount(skill.downloads)}
                      </span>
                      <span>源自{skill.source === "SkillHub" ? "Skillhub" : skill.source}</span>
                    </div>
                  </div>

                  {/* 右：操作 */}
                  <div className="shrink-0">
                    {skill.installed ? (
                      <SmallIconStateButton
                        state="disabled"
                        icon={Check}
                        label="已安装"
                        aria-label={`${skill.name} 已安装`}
                      />
                    ) : (
                      <SmallIconStateButton
                        icon={isAdded ? X : Plus}
                        label={isAdded ? "取消添加" : "添加"}
                        onClick={() => {
                          if (isAdded) {
                            setAddedSkills((prev) => prev.filter((id) => id !== skill.id));
                            return;
                          }
                          handleAdd(skill.id);
                        }}
                        aria-label={isAdded ? `从安装列表移除 ${skill.name}` : `添加 ${skill.name}`}
                      />
                    )}
                  </div>
                </div>
              );
            };

            // ── A 模式：搜索后才出现卡片，点卡片上的「+」加到上面 ──
            if (skillHubConfigured) {
              if (!search.trim()) {
                return (
                  <div className="px-4 py-10 text-center text-xs text-[var(--text-weak)]">
                    请在上方输入准确的 Skill 名称
                  </div>
                );
              }
              if (skillHubSearchResults.length === 0) {
                return (
                  <div className="px-4 py-10 text-center text-xs text-[var(--text-weak)]">
                    未找到名为「{search.trim()}」的技能，请确认名称是否准确
                  </div>
                );
              }
              return <div>{skillHubSearchResults.map(renderSkillRow)}</div>;
            }

            // ── B 模式：未配 SkillHub，企业技能库全量浏览 ──
            if (filteredSkills.length === 0) {
              return (
                <div className="px-4 py-10 text-center text-xs text-[var(--text-weak)]">
                  暂无符合条件的技能
                </div>
              );
            }
            return <div>{filteredSkills.map(renderSkillRow)}</div>;
          })()}
        </div>

        {/* 底部操作栏（§8.7 无分割线，按钮右对齐） */}
        <DialogFooter className="mx-0 mb-0 px-6 pb-6 pt-4 gap-3">
          <Button
            variant="tenant-outline"
            onClick={() => onOpenChange(false)}
          >
            取消
          </Button>
          <Button variant="tenant-primary" onClick={handleInstall}>
            开始安装
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 主页面 ──────────────────────────────────────────────────────────────
export default function OpenClawDetailGuide() {
  const [location, navigate] = useLocation();
  const routeAgentId = useMemo(() => location.split("?")[0].split("/").filter(Boolean).pop() || "", [location]);
  const detailAgent = useMemo<DetailAgentMeta | undefined>(() => {
    const fromStore = loadClawList().find((item) => item.id === routeAgentId);
    const seededMock = MOCK_OPENCLAW_LIST.find((item) => item.id === routeAgentId) as DetailAgentMeta | undefined;
    const fallback = LOCAL_DETAIL_DEMOS.find((item) => item.id === routeAgentId) ?? seededMock;
    if (!fromStore) return fallback;
    return {
      ...fallback,
      ...(fromStore as DetailAgentMeta),
      lastReportedAt: fallback?.lastReportedAt ?? (fromStore as DetailAgentMeta).lastReportedAt,
      localResourceSyncStatus: (fromStore as DetailAgentMeta).localResourceSyncStatus ?? fallback?.localResourceSyncStatus,
    };
  }, [routeAgentId]);
  const isLocalAgent = detailAgent?.agentType === "localagent";
  const localClientProduct = formatLocalClientProduct(detailAgent?.localProduct);
  const hasLocalResourceScopes = isLocalAgent && Boolean(detailAgent?.localProduct);
  const initialAgentType: SwitchableAgentType = detailAgent?.agentType === "hermes" ? "hermes" : "openclaw";
  const [displayAgentType, setDisplayAgentType] = useState<SwitchableAgentType>(initialAgentType);
  const supportsTypeSwitch = detailAgent?.agentType === "openclaw" || detailAgent?.agentType === "hermes" || !detailAgent?.agentType;
  const displayAgentTypeLabel = displayAgentType === "hermes" ? "Hermes" : "OpenClaw";
  const externalAgentTypeLabel = normalizeExternalAgentType(localClientProduct || detailAgent?.model || displayAgentTypeLabel);
  useEffect(() => {
    setDisplayAgentType(initialAgentType);
  }, [initialAgentType, routeAgentId]);
  const isLocalInactive = isLocalAgentInactive(detailAgent);
  const showLocalResourceSyncTip = detailAgent?.localResourceSyncStatus === "syncing";
  const localAgentResources = useMemo(
    () => buildLocalAgentResources(showLocalResourceSyncTip),
    [showLocalResourceSyncTip],
  );
  const agentVersion = detailAgent?.version || detailAgent?.modelVersion || MOCK_INSTANCE.agentVersion;
  const currentVersion = agentVersion.trim().replace(/^v/i, "");
  const isHermes0180 = displayAgentType === "hermes" && currentVersion === "0.18.0";
  const imageAgentType =
    detailAgent?.agentType === "hermes"
      ? "HermesAgent"
      : detailAgent?.agentType === "lightclawace"
        ? "LightClawACE"
        : "OpenClaw";
  const agentVersionLabel =
    isLocalAgent && !hasLocalResourceScopes
      ? formatExternalAgentVersion(agentVersion, externalAgentTypeLabel)
      : formatAgentVersion(agentVersion, isLocalAgent ? detailAgent?.localProduct : undefined);
  const visibleTabs = isLocalAgent
    ? DETAIL_TABS.filter((item) => item.id === "basic")
    : DETAIL_TABS;
  const [activeTab, setActiveTab] = useState<DetailTab>(() => {
    if (typeof window === "undefined") return "basic";
    const tab = new URLSearchParams(window.location.search).get("tab");
    return DETAIL_TABS.some((item) => item.id === tab) ? (tab as DetailTab) : "basic";
  });
  const [skillSearch, setSkillSearch] = useState("");
  const [skillModalOpen, setSkillModalOpen] = useState(false);

  // ── 多角色：判断是否多角色 Agent + 角色列表 + 当前激活角色 ──
  // 优先读取实例真实的 roles（AgentRoleSlot[]，与卡片页同源）；无 roles 时回退到 mock 演示角色。
  const agentRoles = useMemo<AgentRole[]>(() => {
    if (isLocalAgent) return [];
    const realSlots = detailAgent?.roles;
    if (realSlots && realSlots.length > 0) {
      // 每个角色位映射为一个 AgentRole：id=slotId（保证同名角色可精确区分），name=roleName，
      // presetSkills 按角色名从演示模板匹配（匹配不到则为空）。
      return realSlots.map((slot) => {
        const preset = MOCK_AGENT_ROLES.find((r) => r.name === (slot.baseRoleName ?? slot.roleName));
   return { id: slot.slotId, name: slot.roleName, baseRoleName: slot.baseRoleName ?? slot.roleName, isMain: slot.isMain, presetSkills: preset?.presetSkills } as AgentRole;
      });
    }
    // demo 兜底：无真实 roles 的非本地 Agent 用 mock 列表以便预览
    return MOCK_AGENT_ROLES;
  }, [isLocalAgent, detailAgent?.roles]);
  const isMultiRole = agentRoles.length > 1;
  // 角色总数（优先卡片页 roleCount，其次真实 slot 数）
  const roleCount = detailAgent?.roleCount ?? agentRoles.length;
  // 主角色（isMain=true 优先，否则取首个）
  const mainRole = agentRoles.find((r) => (r as any).isMain) ?? agentRoles[0];
  const mainRoleName = mainRole?.name ?? detailAgent?.roleName ?? "通用助手";
  // 主角色的角色类型名（用于头像锚定）；缺省回退到显示名
  const mainRoleType = mainRole?.baseRoleName ?? mainRoleName;

  // 展示型方案切换（B 下拉 / C 角色栏）——仅影响多角色信息的呈现形态
  const [rolePresentation, setRolePresentation] = useState<"B" | "C">("B");

  // 全部可选角色名（用于快捷切换候选）：演示模板 + 通用助手兜底，叠加分组白名单过滤
  const allSelectableRoleNames = useMemo(() => {
    const GENERAL = "通用助手";
    const base = Array.from(new Set([GENERAL, ...MOCK_AGENT_ROLES.map((r) => r.name), ...agentRoles.map((r) => r.name)]));
    const allowed = detailAgent?.allowedRoleNames;
    const scoped = allowed ? base.filter((n) => n === GENERAL || allowed.includes(n)) : base;
    return scoped;
  }, [agentRoles, detailAgent?.allowedRoleNames]);
  const getSwitchCandidates = useCallback(
    (slotRoleName: string) => allSelectableRoleNames.filter((n) => n !== slotRoleName),
    [allSelectableRoleNames],
  );

  // 同名角色折叠计数（保持「按类型名展示、不去重编号」口径，展开精确到 slotId）
  const groupedRoles = useMemo(() => {
    const map = new Map<string, { roleName: string; count: number; isMain: boolean; slotIds: string[] }>();
    for (const r of agentRoles) {
      const isMain = Boolean((r as any).isMain);
      const g = map.get(r.name);
      if (g) {
        g.count += 1;
        g.slotIds.push(r.id);
        if (isMain) g.isMain = true;
      } else {
        map.set(r.name, { roleName: r.name, count: 1, isMain, slotIds: [r.id] });
      }
    }
    return Array.from(map.values()).sort((a, b) => Number(b.isMain) - Number(a.isMain));
  }, [agentRoles]);

  // 快捷角色切换：对指定 slot 落库切换角色（写入 openclawStore，与卡片页同源），并给出反馈
  const handleQuickSwitchRole = useCallback(
    (slotId: string, targetRoleName: string) => {
      try {
        const latest = loadClawList();
        const next = latest.map((item: any) => {
          if (item.id !== routeAgentId) return item;
          const existing = (item.roles as { slotId: string; roleName: string; isMain: boolean }[] | undefined) ??
            (item.roleName ? [{ slotId: `slot-main-${item.id}`, roleName: item.roleName, isMain: true }] : []);
          const updatedSlots = existing.map((s) => (s.slotId === slotId ? { ...s, roleName: targetRoleName } : s));
          const nextMain = updatedSlots.find((s) => s.isMain)?.roleName ?? item.roleName;
          return { ...item, roles: updatedSlots, roleName: nextMain };
        });
        saveClawList(next);
        notifyClawListChange();
        toast.success(`已切换为「${targetRoleName}」，正在下发角色配置…`);
      } catch {
        toast.error("角色切换失败，请重试");
      }
    },
    [routeAgentId],
  );

  // ══════════════════════════════════════════════════════════════════════
  // 角色管理抽屉（完整复刻「我的 Agent」方案1）：状态 + 派生 + 落库桥接
  // ══════════════════════════════════════════════════════════════════════
  // 可选真实角色（来自角色库 MOCK_ROLES，含 soul / skills，用于介绍卡与技能编辑）
  const visibleRoles = useMemo(() => MOCK_ROLES.filter((r) => r.visible), []);
  // 当前实例的角色位（由 agentRoles 映射为 AgentRoleSlot：slotId=r.id）
  const switchRoleSlots = useMemo<AgentRoleSlot[]>(
    () => agentRoles.map((r) => ({ slotId: r.id, roleName: r.name, isMain: Boolean((r as any).isMain) })),
    [agentRoles],
  );

  // 抽屉主状态：null=关闭；打开时携带该 Agent 的 id/name/roles
  const [switchRoleDialog, setSwitchRoleDialog] = useState<{ id: string; name: string; roleName: string; allowedRoleNames?: string[]; roles?: AgentRoleSlot[] } | null>(null);
  const [switchRoleTarget, setSwitchRoleTarget] = useState<Role | "__general__" | null>(null);
  const [switchRoleTargets, setSwitchRoleTargets] = useState<Record<string, Role | "__general__" | null>>({});
  const [expandedSlotId, setExpandedSlotId] = useState<string | null>(null);
  const [confirmingSlotId, setConfirmingSlotId] = useState<string | null>(null);
  const [confirmedSlotIds, setConfirmedSlotIds] = useState<Set<string>>(new Set());
  const [deleteSlotConfirm, setDeleteSlotConfirm] = useState<{ slotId: string; roleName: string } | null>(null);
  const [renameSlotTarget, setRenameSlotTarget] = useState<{ slotId: string; roleName: string } | null>(null);
  const [switchRoleRenameValue, setSwitchRoleRenameValue] = useState("");
  const [switchRoleRenameError, setSwitchRoleRenameError] = useState("");
  const [editSlots, setEditSlots] = useState<AgentRoleSlot[]>([]);
  // per-slot 切换角色弹窗内的技能子集
  const [switchSlotSkillNames, setSwitchSlotSkillNames] = useState<Record<string, Set<string>>>({});
  const [switchSlotSkillPopoverSlot, setSwitchSlotSkillPopoverSlot] = useState<string | null>(null);
  const [switchSlotSkillsBackup, setSwitchSlotSkillsBackup] = useState<Set<string>>(new Set());
  const switchSlotSkillPanelRef = useRef<HTMLDivElement>(null);
  const [skillPanelWidth, setSkillPanelWidth] = useState<Record<string, number>>({});
  // 已单独提示过「切换成功」的 slotId
  const announcedSwitchSlotsRef = useRef<Set<string>>(new Set());
  // 「角色切换中」的 Agent：agentId → 正在切换的角色数量
  const [switchingRoleAgents, setSwitchingRoleAgents] = useState<Record<string, number>>({});
  const markRoleSwitching = useCallback((agentId: string, count: number) => {
    setSwitchingRoleAgents((prev) => ({ ...prev, [agentId]: count }));
  }, []);
  const clearRoleSwitching = useCallback((agentId: string) => {
    setSwitchingRoleAgents((prev) => {
      if (!(agentId in prev)) return prev;
      const next = { ...prev };
      delete next[agentId];
      return next;
    });
  }, []);
  const roleConfigProgress = useRoleConfigProgress();

  // 新增角色独立弹窗：弹窗内部交互态（选中角色 / 角色名称 / 校验）由共享组件 AddRoleDialog 自管理，
  // 本页仅保留「打开数据源」与落库回调。
  const [standaloneAddRole, setStandaloneAddRole] = useState<{ id: string; name: string; roles: AgentRoleSlot[]; allowedRoleNames?: string[] } | null>(null);
  // 批量切换独立弹窗：弹窗内部交互态与配置加载动画由共享组件 BatchSwitchRoleDialog 自管理，
  // 本页仅保留「打开数据源」与落库 / 后台标记回调。
  const [standaloneBatchSwitch, setStandaloneBatchSwitch] = useState<{ id: string; name: string; slots: AgentRoleSlot[]; allowedRoleNames?: string[] } | null>(null);

  // 正在运行的 Agent 名称集合：切换角色时校验角色名称不得与运行中 Agent 重名（排除当前 Agent 自身）。
  const runningAgentNames = useMemo(
    () =>
      loadClawList()
        .filter((c: any) => c.status === "running" && c.id !== routeAgentId)
        .map((c: any) => c.name),
    [routeAgentId, standaloneBatchSwitch],
  );

  const currentSlotRoleName = switchRoleDialog?.roleName ?? "通用助手";

  // 打开「切换角色」独立批量切换弹窗（与「我的 Agent」卡片切换角色 100% 一致的交互）：
  // 不打开角色管理抽屉，直接对该 Agent 的全部角色位做批量切换。
  const openStandaloneBatchSwitch = useCallback(
    ({ id, name, roles, allowedRoleNames }: { id: string; name: string; roles?: AgentRoleSlot[]; allowedRoleNames?: string[] }) => {
      const initSlots: AgentRoleSlot[] =
    roles && roles.length > 0
     ? roles.map((s) => ({ ...s }))
      : [{ slotId: `slot-main-${id}`, roleName: switchRoleDialog?.roleName ?? "通用助手", isMain: true }];
      setStandaloneBatchSwitch({ id, name, slots: initSlots, allowedRoleNames });
    },
    [switchRoleDialog?.roleName],
  );

  // 打开抽屉时初始化多角色映射表；关闭时清理全部相关状态
  useEffect(() => {
    if (!switchRoleDialog) {
      setSwitchRoleTargets({}); setExpandedSlotId(null); setEditSlots([]); setConfirmingSlotId(null);
      setConfirmedSlotIds(new Set()); setDeleteSlotConfirm(null); setRenameSlotTarget(null);
      return;
    }
    const slots = switchRoleDialog.roles;
    if (slots && slots.length >= 1) {
      const init: Record<string, Role | "__general__" | null> = {};
      slots.forEach((s) => { init[s.slotId] = null; });
      setSwitchRoleTargets(init);
      setExpandedSlotId(null);
      setConfirmingSlotId(null);
      setConfirmedSlotIds(new Set());
      setEditSlots(slots.map((s) => ({ ...s })));
    }
  }, [switchRoleDialog]);

  // 单角色实例：打开抽屉时给出默认目标，避免「确认切换」默认禁用
  useEffect(() => {
    if (!switchRoleDialog) return;
    const slots = switchRoleDialog.roles;
    if (slots && slots.length >= 1) return;
    const allowedRoleNames = switchRoleDialog.allowedRoleNames;
    const candidateRoles = allowedRoleNames
      ? visibleRoles.filter((r) => allowedRoleNames.includes(r.name))
      : visibleRoles;
    const switchableRoles = candidateRoles.filter((r) => r.name !== currentSlotRoleName);
    const canSwitchToGeneral = currentSlotRoleName !== "通用助手";
    setSwitchRoleTarget(canSwitchToGeneral ? "__general__" : switchableRoles[0] ?? null);
  }, [switchRoleDialog]);

  // 计算某角色位可切换的目标候选
  const computeRowOptions = useCallback((slotRoleName: string) => {
    const allowedRoleNames = switchRoleDialog?.allowedRoleNames;
    const candidateRoles = allowedRoleNames
      ? visibleRoles.filter((r) => allowedRoleNames.includes(r.name))
      : visibleRoles;
    const switchableRoles = candidateRoles.filter((r) => r.name !== slotRoleName);
    const canSwitchToGeneral = slotRoleName !== "通用助手";
    return { switchableRoles, canSwitchToGeneral };
  }, [switchRoleDialog, visibleRoles]);

  // 角色介绍（技能 + 风格）
  const getRoleIntro = useCallback((target: Role | "__general__" | null) => {
    const generalIntro = {
      name: "通用助手",
      skills: "web-search、file-reader、code-runner",
      soul: "无固定行业偏好的通用 AI 伙伴，擅长日常问答、信息检索与轻量创作，按需切换专业度",
    };
    if (!target || target === "__general__") return generalIntro;
    return {
      name: target.name,
      skills: target.skills.map((s) => s.name).join("、"),
      soul: target.soul,
    };
  }, []);

  // ── Inline rename state ──
  // 注：agentName 被下方 applyDetailRoleChanges 的依赖数组引用，声明必须前置于该 useCallback，
  //     否则会触发 "Cannot access 'agentName' before initialization"（TDZ）。
  const AGENT_NAME_MAX_BYTES = 128;
  const getAgentNameByteLength = (value: string) => new TextEncoder().encode(value).length;
  const [agentName, setAgentName] = useState(() => detailAgent?.name || "多组织示例–前端研发");
  const [isNameEditing, setIsNameEditing] = useState(false);
  const [nameDraft, setNameDraft] = useState("");
  const [nameError, setNameError] = useState<string>("");
  const [isNameOverflow, setIsNameOverflow] = useState(false);
  const nameEditWrapperRef = useRef<HTMLDivElement | null>(null);
  const nameInputRef = useRef<HTMLInputElement | null>(null);
  const nameTextRef = useRef<HTMLHeadingElement | null>(null);

  // 落库桥接：把角色位增删改统一写入 openclawStore（与卡片页同源），并同步刷新详情页 detailAgent
  const applyDetailRoleChanges = useCallback((
    nextSlots: AgentRoleSlot[],
    targets: Record<string, Role | "__general__" | null>,
    originalSlots: AgentRoleSlot[],
  ) => {
    const originalIds = new Set(originalSlots.map((s) => s.slotId));
    const nextIds = new Set(nextSlots.map((s) => s.slotId));
    const addedCount = nextSlots.filter((s) => !originalIds.has(s.slotId)).length;
    const removedCount = originalSlots.filter((s) => !nextIds.has(s.slotId)).length;
    let switchedCount = 0;
    const switchedPairs: { fromName: string; toName: string }[] = [];
    const resolvedSlots = nextSlots.map((slot) => {
      const target = targets[slot.slotId];
      if (!target) return slot;
      const targetName = target === "__general__" ? "通用助手" : target.name;
      if (targetName === slot.roleName) return slot;
      if (originalIds.has(slot.slotId)) {
        switchedCount += 1;
        if (!announcedSwitchSlotsRef.current.has(slot.slotId)) {
          switchedPairs.push({ fromName: slot.roleName || "通用助手", toName: targetName });
        }
      }
      return { ...slot, roleName: targetName };
    });

    try {
      const latest = loadClawList();
      const next = latest.map((item: any) => {
        if (item.id !== routeAgentId) return item;
        const mainSlot = resolvedSlots.find((s) => s.isMain);
        return {
          ...item,
          roles: resolvedSlots.map((s) => ({ ...s })),
          roleCount: resolvedSlots.length,
          roleName: mainSlot?.roleName ?? item.roleName,
        };
      });
      saveClawList(next);
      notifyClawListChange();
    } catch {
      toast.error("角色配置保存失败，请重试");
    }

    setSwitchRoleDialog(null);
    setSwitchRoleTargets({});
    setExpandedSlotId(null);
    setEditSlots([]);
    const hasAnyChange = addedCount > 0 || removedCount > 0 || switchedCount > 0;
    if (hasAnyChange) clearRoleSwitching(routeAgentId);

    const parts: string[] = [];
    if (addedCount > 0) parts.push(`新增 ${addedCount} 个角色`);
    if (removedCount > 0) parts.push(`删除 ${removedCount} 个角色`);
    if (switchedCount > 0) parts.push(`切换 ${switchedCount} 个角色`);
    const agentDisplayName = switchRoleDialog?.name ?? agentName;
    if (switchedCount > 0 && addedCount === 0 && removedCount === 0) {
      if (switchedPairs.length === 0) { announcedSwitchSlotsRef.current = new Set(); return; }
      const detail = switchedPairs.map((p) => `${p.fromName} → ${p.toName}`).join("、");
      toast.success(`「${agentDisplayName}」角色切换成功：${detail}`);
      announcedSwitchSlotsRef.current = new Set();
      return;
    }
    announcedSwitchSlotsRef.current = new Set();
    if (!hasAnyChange) return;
    toast.success(`「${agentDisplayName}」已${parts.join("、")}`);
  }, [routeAgentId, switchRoleDialog, agentName, clearRoleSwitching]);

  // 单角色实例「确认切换」落库
  const applySingleRoleSwitch = useCallback((targetName: string, previousRoleName: string) => {
    try {
      const latest = loadClawList();
      const next = latest.map((item: any) => {
        if (item.id !== routeAgentId) return item;
        const existing = (item.roles as AgentRoleSlot[] | undefined) ??
          (item.roleName ? [{ slotId: `slot-main-${item.id}`, roleName: item.roleName, isMain: true }] : []);
        const updatedSlots = existing.length > 0
          ? existing.map((s, i) => (i === 0 ? { ...s, roleName: targetName } : s))
          : [{ slotId: `slot-main-${item.id}`, roleName: targetName, isMain: true }];
        const nextMain = updatedSlots.find((s) => s.isMain)?.roleName ?? targetName;
        return { ...item, roles: updatedSlots, roleName: nextMain };
      });
      saveClawList(next);
      notifyClawListChange();
      toast.success(`角色切换成功：${previousRoleName} → ${targetName}`);
    } catch {
      toast.error("角色切换失败，请重试");
    }
    setSwitchRoleDialog(null);
    setSwitchRoleTarget(null);
    clearRoleSwitching(routeAgentId);
  }, [routeAgentId, clearRoleSwitching]);

  const [activeRoleId, setActiveRoleId] = useState<string>(() => agentRoles[0]?.id ?? "");
  const [roleDropdownOpen, setRoleDropdownOpen] = useState(false);
  const roleDropdownRef = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    // 切换 Agent 时重置到首个角色
    setActiveRoleId(agentRoles[0]?.id ?? "");
  }, [agentRoles]);
  // 点击下拉外部时关闭
  useEffect(() => {
    if (!roleDropdownOpen) return;
    const onDocClick = (e: MouseEvent) => {
      if (roleDropdownRef.current && !roleDropdownRef.current.contains(e.target as Node)) {
        setRoleDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, [roleDropdownOpen]);
  const activeRole = agentRoles.find((r) => r.id === activeRoleId) ?? agentRoles[0];

  useEffect(() => {
    setAgentName(detailAgent?.name || "多组织示例–前端研发");
  }, [detailAgent?.id, detailAgent?.name]);

  useEffect(() => {
    if (isLocalAgent && activeTab !== "basic") {
      setActiveTab("basic");
    }
  }, [activeTab, isLocalAgent]);

  const startNameEdit = useCallback(() => {
    setIsNameEditing(true);
    setNameDraft(agentName);
    setNameError("");
  }, [agentName]);

  const cancelNameEdit = useCallback(() => {
    setIsNameEditing(false);
    setNameDraft(agentName);
    setNameError("");
  }, [agentName]);

  const saveNameEdit = useCallback(() => {
    const trimmed = nameDraft.trim();
    if (!trimmed) {
      cancelNameEdit();
      return;
    }
    if (getAgentNameByteLength(trimmed) > AGENT_NAME_MAX_BYTES) {
      setNameError(`名称不能超过 ${AGENT_NAME_MAX_BYTES} 字节`);
      return;
    }
    setAgentName(trimmed);
    setIsNameEditing(false);
    setNameError("");
  }, [nameDraft, cancelNameEdit]);

  useEffect(() => {
    if (!isNameEditing) return;
    const frame = requestAnimationFrame(() => {
      nameInputRef.current?.focus();
      nameInputRef.current?.select();
    });
    return () => cancelAnimationFrame(frame);
  }, [isNameEditing]);

  useEffect(() => {
    if (!isNameEditing) return;
    const handlePointerDown = (e: MouseEvent) => {
      if (!nameEditWrapperRef.current) return;
      if (nameEditWrapperRef.current.contains(e.target as Node)) return;
      saveNameEdit();
    };
    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, [isNameEditing, saveNameEdit]);

  useEffect(() => {
    if (isNameEditing) return;
    const el = nameTextRef.current;
    if (!el) return;
    const updateOverflow = () => setIsNameOverflow(el.scrollWidth > el.clientWidth);
    updateOverflow();
    const ro = new ResizeObserver(updateOverflow);
    ro.observe(el);
    window.addEventListener("resize", updateOverflow);
    return () => { ro.disconnect(); window.removeEventListener("resize", updateOverflow); };
  }, [agentName, isNameEditing]);

  // ── 一键更新：确认弹窗 + 进度弹窗（与 OpenClawDetail 对齐） ──
  const [showUpdateConfirmDialog, setShowUpdateConfirmDialog] = useState(false);
  const [showUpdateProgressDialog, setShowUpdateProgressDialog] = useState(false);

  // 从列表页"可更新"胶囊跳来时（?action=update），自动打开更新确认弹窗
  // 触发后立即清掉 query，避免刷新或返回时再次弹出
  useEffect(() => {
    if (typeof window === "undefined") return;
    const params = new URLSearchParams(window.location.search);
    if (params.get("action") === "update") {
      setShowUpdateConfirmDialog(true);
      params.delete("action");
      const next = params.toString();
      const cleanUrl = `${window.location.pathname}${next ? `?${next}` : ""}`;
      window.history.replaceState(null, "", cleanUrl);
    }
    // 仅首次挂载触发；后续走 setShowUpdateConfirmDialog 主动管理
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const [isUpdating, setIsUpdating] = useState(false);
  const [updateStepsDone, setUpdateStepsDone] = useState<number>(0);
  const updateSteps = [
    "环境准备",
    "Agent 安装",
    "Doctor 修复",
    "Gateway 安装",
    "Clawhub 安装",
    "插件安装",
    "Skills 安装",
    "安装收尾",
  ];
  const handleStartUpdate = () => {
    setShowUpdateConfirmDialog(false);
    setIsUpdating(true);
    setUpdateStepsDone(0);
    setShowUpdateProgressDialog(true);
    // 随机间隔逐步完成 8 步
    let done = 0;
    const runNext = () => {
      if (done >= updateSteps.length) {
        setIsUpdating(false);
        setShowUpdateProgressDialog(false);
        if (targetVersion) {
          MOCK_INSTANCE.agentVersion = targetVersion;
        }
        toast.success("Agent 已更新至最新版本");
        return;
      }
      const delay = 800 + Math.random() * 2200; // 0.8s ~ 3s 随机
      setTimeout(() => {
        done += 1;
        setUpdateStepsDone(done);
        runNext();
      }, delay);
    };
    runNext();
  };

  // ── 平台策略：是否允许用户自助更新（与管控端 PlatformPolicy 共享 key） ──
  // 默认 true；管理员关闭后「一键更新」按钮置灰 + Tooltip 提示
  const [allowSelfUpgrade, setAllowSelfUpgrade] = useState<boolean>(() => {
    try {
      const raw = localStorage.getItem("admin_allow_self_upgrade");
      return raw === null ? true : raw === "true";
    } catch {
      return true;
    }
  });
  useEffect(() => {
    const sync = () => {
      try {
        const raw = localStorage.getItem("admin_allow_self_upgrade");
        setAllowSelfUpgrade(raw === null ? true : raw === "true");
      } catch { /* ignore */ }
    };
    window.addEventListener("storage", sync);
    return () => window.removeEventListener("storage", sync);
  }, []);

  // ── 管理员推送的「可更新」徽章（仅当前实例版本低于同类型推送版本时展示） ──
  const getRecommendPush = useCallback((): ActivePush | null => {
    if (isLocalAgent) return null;
    const push = getActivePush(imageAgentType);
    return push && compareVersion(currentVersion, push.version) < 0 ? push : null;
  }, [currentVersion, imageAgentType, isLocalAgent]);
  const [recommendPush, setRecommendPush] = useState<ActivePush | null>(() => getRecommendPush());
  useEffect(() => {
    const refresh = () => setRecommendPush(getRecommendPush());
    refresh();
    window.addEventListener("upgrade-push-changed", refresh);
    window.addEventListener("storage", refresh);
    return () => {
      window.removeEventListener("upgrade-push-changed", refresh);
      window.removeEventListener("storage", refresh);
    };
  }, [getRecommendPush]);

  // ── 一键更新：目标版本（读取管理员配置的同类型生效镜像版本） ──
  const getTargetVersion = useCallback((): string | null => {
    try {
      const raw = localStorage.getItem("admin_images_v3");
      if (!raw) return null;
      const imgs: Array<{ agentType: string; agentVersion: string; active: boolean }> = JSON.parse(raw);
      const activeImg = imgs.find(
        (img) => img.agentType === imageAgentType && img.active,
      );
      return activeImg?.agentVersion || null;
    } catch {
      return null;
    }
  }, [imageAgentType]);

  const [targetVersion, setTargetVersion] = useState<string | null>(() => getTargetVersion());

  useEffect(() => {
    const refresh = () => setTargetVersion(getTargetVersion());
    refresh();
    window.addEventListener("storage", refresh);
    return () => window.removeEventListener("storage", refresh);
  }, [getTargetVersion]);

  const isTargetNewer = targetVersion ? compareVersion(targetVersion, currentVersion) > 0 : false;

  // ─── 已安装 / 待安装技能（动态状态机，对齐 main 分支） ───
  const [installedSkills, setInstalledSkills] = useState<{ name: string; version: string }[]>(
    MOCK_INSTALLED_SKILLS,
  );
  const [pendingSkills, setPendingSkills] = useState<PendingSkill[]>(MOCK_PENDING_SKILLS);

  // 技能最新版本表（mock）：仅当已安装版本低于此处版本时，才展示"更新"入口
  const SKILL_LATEST_VERSIONS: Record<string, string> = {
    "code-interpreter": "1.3.0",
    "image-recognition": "1.0.0",
    "web-scraper": "1.4.0",
    "api-tester": "2.2.0",
    "sql-executor": "1.6.0",
    "translation-engine": "3.1.0",
    "log-analyzer": "2.1.0",
    "data-visualizer": "1.5.0",
  };

  // 计算某条已安装技能的更新信息
  const getSkillUpdateInfo = (skill: { name: string; version: string }) => {
    const latestVersion = SKILL_LATEST_VERSIONS[skill.name];
    const hasUpdate = !!latestVersion && !!skill.version && compareVersion(latestVersion, skill.version) > 0;
    return { latestVersion, hasUpdate };
  };

  // 卸载技能二次确认
  const [skillUninstallConfirm, setSkillUninstallConfirm] = useState<{ open: boolean; name: string; version: string }>({
    open: false,
    name: "",
    version: "",
  });

  // 更新技能到最新版本
  const handleUpdateSkill = (skill: { name: string; version: string }) => {
    const { latestVersion } = getSkillUpdateInfo(skill);
    if (!latestVersion) return;
    setInstalledSkills((prev) =>
      prev.map((s) => (s.name === skill.name && s.version === skill.version ? { ...s, version: latestVersion } : s)),
    );
    toast.success(`${skill.name} 已更新至 ${latestVersion}`, {
      description: skill.version ? `原版本 ${skill.version}` : undefined,
    });
  };

  // 确认卸载技能
  const handleConfirmUninstallSkill = () => {
    const { name, version } = skillUninstallConfirm;
    setInstalledSkills((prev) => prev.filter((s) => !(s.name === name && s.version === version)));
    setSkillUninstallConfirm({ open: false, name: "", version: "" });
    toast.success(`${name} 已卸载`);
  };

  // 进入页面后（及后续新加入的）所有 pending 同时变为 installing（模拟并行安装）
  const pendingWaitingKey = pendingSkills
    .filter((s) => s.status === "pending")
    .map((s) => s.id)
    .join();
  useEffect(() => {
    if (!pendingWaitingKey) return;
    setPendingSkills((prev) =>
      prev.map((s) => (s.status === "pending" ? { ...s, status: "installing" as PendingSkillStatus } : s)),
    );
  }, [pendingWaitingKey]);

  // 监听 installing 状态：3s 后一次性出结果（成功推入已安装；失败标记 failed）
  const installingKey = pendingSkills
    .filter((s) => s.status === "installing")
    .map((s) => s.id)
    .join();
  useEffect(() => {
    if (!installingKey) return;
    const timer = setTimeout(() => {
      setPendingSkills((prev) => {
        const installing = prev.filter((s) => s.status === "installing");
        if (installing.length === 0) return prev;
        const successList = installing.filter((s) => !MOCK_FAIL_IDS.has(s.id));
        const failedIds = new Set(
          installing.filter((s) => MOCK_FAIL_IDS.has(s.id)).map((s) => s.id),
        );
        // 删除成功项；失败项标记 failed
        const next = prev
          .filter((s) => !successList.some((ss) => ss.id === s.id))
          .map((s) => (failedIds.has(s.id) ? { ...s, status: "failed" as PendingSkillStatus } : s));
        if (successList.length > 0) {
          // 成功批量加入已安装列表（解析 "name version" 格式）
          setInstalledSkills((list) => [
            ...successList.map((s) => {
              const lastSpace = s.name.lastIndexOf(" ");
              return lastSpace > 0
                ? { name: s.name.slice(0, lastSpace), version: s.name.slice(lastSpace + 1) }
                : { name: s.name, version: "" };
            }),
            ...list,
          ]);
        }
        return next;
      });
    }, 3000);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [installingKey]);

  // 全部重试：所有 failed 重新进入 installing
  const handleRetryAllFailed = () => {
    const failedIds = pendingSkills.filter((s) => s.status === "failed").map((s) => s.id);
    failedIds.forEach((id) => MOCK_FAIL_IDS.delete(id));
    setPendingSkills((prev) =>
      prev.map((s) => (s.status === "failed" ? { ...s, status: "installing" as PendingSkillStatus } : s)),
    );
    if (failedIds.length > 0) toast.info(`已重试 ${failedIds.length} 个失败技能`);
  };

  // 全部删除：移除所有 failed
  const handleDeleteAllFailed = () => {
    const count = pendingSkills.filter((s) => s.status === "failed").length;
    setPendingSkills((prev) => prev.filter((s) => s.status !== "failed"));
    if (count > 0) toast.info(`已删除 ${count} 个失败项`);
  };

  // 把弹窗里勾选的技能加入待安装队列
  const handleEnqueueSkills = (names: string[]) => {
    if (names.length === 0) return;
    setPendingSkills((prev) => [
      ...prev,
      ...names.map((name, i) => ({
        id: `ps-${Date.now()}-${i}`,
        name,
        status: "pending" as PendingSkillStatus,
      })),
    ]);
  };

  // ── 头部「项目」编辑 ──────────────────────────────────────
  // 项目池来自共享 groupStore（管控端「资产管理 / 用户管理」创建的项目实时可见，单层级）
  const [projectPool, setProjectPool] = useState<{ id: string; name: string }[]>(
    () => groupStore.getAll().filter((g) => g.source === "project" && g.parentId === null).map((g) => ({ id: g.id, name: g.name })),
  );
  useEffect(
    () => groupStore.subscribe(() =>
      setProjectPool(
        groupStore.getAll().filter((g) => g.source === "project" && g.parentId === null).map((g) => ({ id: g.id, name: g.name })),
      ),
    ),
    [],
  );
  // detailAgent 为只读派生值，用本地 state 承载可编辑的项目关联
  const [projectIds, setProjectIds] = useState<string[]>([]);
  const [projectNames, setProjectNames] = useState<string[]>([]);
  useEffect(() => {
    setProjectIds(detailAgent?.projectIds ?? []);
    setProjectNames(detailAgent?.projectNames ?? []);
  }, [detailAgent]);

  const [projectEditOpen, setProjectEditOpen] = useState(false);
  const [projectDraftIds, setProjectDraftIds] = useState<string[]>([]);
  const toggleProjectDraft = (id: string) =>
    setProjectDraftIds((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));
  const openProjectEdit = useCallback(() => {
    setProjectDraftIds(projectIds);
    setProjectEditOpen(true);
  }, [projectIds]);

  /** 查询某项目在「资产管理」中关联的技能（企业技能 + 公共技能），返回 "名称 版本" 列表 */
  const getProjectSkillNames = useCallback((projectId: string): string[] => {
    const cfg = projectAssetStore.getConfig(projectId);
    const refs = [
      ...cfg.categories.enterpriseSkill.items.map((i) => ({ refId: i.refId, version: i.versionAtBind, cat: "enterpriseSkill" as const })),
      ...cfg.categories.publicSkill.items.map((i) => ({ refId: i.refId, version: i.versionAtBind, cat: "publicSkill" as const })),
    ];
    return refs.map((r) => {
      const d = getAssetItemDisplay(r.cat, r.refId);
      return `${d.name} ${r.version}`;
    });
  }, []);

  const saveProjectEdit = useCallback(() => {
    const prevIds = projectIds;
    const nextIds = projectDraftIds;
    const nextNames = projectPool.filter((p) => nextIds.includes(p.id)).map((p) => p.name);
    const addedIds = nextIds.filter((id) => !prevIds.includes(id));

    // 持久化项目关联到共享 store
    try {
      const latest = loadClawList();
      const updated = latest.map((item) =>
        item.id === routeAgentId ? { ...item, projectIds: nextIds, projectNames: nextNames } : item,
      );
      saveClawList(updated);
      notifyClawListChange();
    } catch {
      // ignore（演示环境）
    }
    setProjectIds(nextIds);
    setProjectNames(nextNames);

    // 收集新增项目携带的技能，注入待安装队列（复用现有 pending→installing→installed 流程）
    let newSkillCount = 0;
    if (addedIds.length > 0) {
      const skillNames = Array.from(new Set(addedIds.flatMap((pid) => getProjectSkillNames(pid))));
      const installedStr = installedSkills.map((s) => `${s.name} ${s.version}`);
      const toAdd = skillNames.filter(
        (name) => !installedStr.includes(name) && !pendingSkills.some((p) => p.name === name),
      );
      newSkillCount = toAdd.length;
      if (toAdd.length > 0) {
        setPendingSkills((cur) => [
          ...cur,
          ...toAdd
            .filter((name) => !cur.some((c) => c.name === name))
            .map((name) => ({ id: `proj-skill-${name}`, name, status: "pending" as PendingSkillStatus })),
        ]);
      }
    }

    setProjectEditOpen(false);
    toast.success(newSkillCount > 0 ? `已更新项目关联，正在安装 ${newSkillCount} 个新增技能` : "已更新项目关联");
  }, [routeAgentId, projectIds, projectDraftIds, projectPool, getProjectSkillNames, installedSkills, pendingSkills]);

  const [panelDialogOpen, setPanelDialogOpen] = useState(false);
  const [agentTypeSwitchOpen, setAgentTypeSwitchOpen] = useState(false);
  const [agentTypeSwitching, setAgentTypeSwitching] = useState(false);
  const agentTypeSwitchTimerRef = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (agentTypeSwitchTimerRef.current !== null) {
        window.clearTimeout(agentTypeSwitchTimerRef.current);
      }
    };
  }, []);

  const handleAgentTypeSwitchConfirm = (nextType: SwitchableAgentType) => {
    if (agentTypeSwitchTimerRef.current !== null) {
      window.clearTimeout(agentTypeSwitchTimerRef.current);
    }
    const shouldMockFailure = routeAgentId === "oc-015";
    const currentTypeLabel = displayAgentType === "hermes" ? "Hermes" : "OpenClaw";
    setAgentTypeSwitching(true);
    agentTypeSwitchTimerRef.current = window.setTimeout(() => {
      if (shouldMockFailure) {
        setAgentTypeSwitching(false);
        agentTypeSwitchTimerRef.current = null;
        toast.error(`切换失败，实例已恢复为 ${currentTypeLabel}，请重试`);
        return;
      }
      setDisplayAgentType(nextType);
      setAgentTypeSwitching(false);
      agentTypeSwitchTimerRef.current = null;
      toast.success(`已切换为 ${nextType === "hermes" ? "Hermes" : "OpenClaw"}`);
    }, 2400);
  };

  // ── 数据备份 mock 状态 ──
  // 备份点状态枚举：none（无备份点）/ creating（备份生成中）/ ready（备份完成，可回滚）
  // 通过 helper 读取 mock map：3 个演示用 Agent 分别绑定 none/creating/ready。
  // 其他普通 Agent 找不到记录时 fallback 到 "ready"，保持现有 demo 行为不变。
  // TODO(后端接入): 删除 getDemoBackupStatus 调用，改为从实例详情接口读取：
  //   - backup_point_status: "none"|"creating"|"ready"  备份点状态
  //   - backup_triggered_at: string (ISO)                最近一次备份点的生成时刻（弹窗「本次回滚到」展示用）
  //   - backup_trigger_type: "update"|"reinstall"        最近一次备份点的触发操作类型（展示用）
  // 后端当前仅暴露一个备份点（前端无法多选），且备份点只由「一键更新」/「重装」触发
  // 联调时：用真实字段替换 getDemoBackupStatus 调用即可，整段 mock 逻辑清除
  const [backupPointStatus] = useState<BackupPointStatus>(() => getDemoBackupStatus(routeAgentId));
  // TODO(后端接入): 替换为真实字段 backup_triggered_at / backup_trigger_type
  const [backupTriggeredAt] = useState("2026-08-10 14:32");
  const [backupTriggerType] = useState<"update" | "reinstall">("update");
  const [showBackupInfoDialog, setShowBackupInfoDialog] = useState(false);
  const [showBackupConfirmDialog, setShowBackupConfirmDialog] = useState(false);
  const [backupRollbackLoading, setBackupRollbackLoading] = useState(false);

  const openDataBackup = () => {
    setShowBackupInfoDialog(true);
  };

  const openBackupRollbackConfirm = () => {
    setShowBackupInfoDialog(false);
    setShowBackupConfirmDialog(true);
  };

  // TODO(后端接入): POST /tenant/instance/{id}/backup/rollback
  //   200 → 成功；409 → 冲突（任务进行中）；5xx → 其他错误
  //   联调时：发起请求 → 跳转列表页 → 后端 webhook/轮询推送最终结果
  const confirmBackupRollback = () => {
    setBackupRollbackLoading(true);
    // mock: 立即把 store 里该 Agent 状态改为 rollingBack，联调时改为真实请求
    const list = loadClawList();
    const updated = list.map((item) =>
    item.id === routeAgentId ? { ...item, status: "rollingBack" } : item
    );
    saveClawList(updated);
    notifyClawListChange();
    setBackupRollbackLoading(false);
    setShowBackupConfirmDialog(false);
    toast.info("回滚已发起，正在处理中…");
    navigate("/my-openclaw");
  };

  // ── WebUI 面板弹窗状态 ──
  const [showHermesPanelDialog, setShowHermesPanelDialog] = useState(false);
  const [showHermesPassword, setShowHermesPassword] = useState(false);
  const [showWebUIProgressDialog, setShowWebUIProgressDialog] = useState(false);
  const [webUIStep, setWebUIStep] = useState(0);
  const [webUIFailedStep, setWebUIFailedStep] = useState<"none" | "port" | "link">("none");
  const [showWebUIResultDialog, setShowWebUIResultDialog] = useState(false);
  const [webUIUrl, setWebUIUrl] = useState("");
  const [webUIToken, setWebUIToken] = useState("");

  const handleOpenAgentPanel = () => {
    if (isHermes0180) {
      handleOpenWebUI();
      return;
    }

    setWebUIUrl(DEMO_OPENCLAW_PANEL_URL);
    setWebUIToken(DEMO_OPENCLAW_PANEL_TOKEN);
    setShowWebUIResultDialog(true);
  };

  const copyHermesCredential = async (label: string, value: string) => {
    if (!value) {
      toast.error(`${label}暂不可用，请稍后重试`);
      return;
    }

    try {
      if (!navigator.clipboard?.writeText || !window.isSecureContext) {
        throw new Error("clipboard-api-unavailable");
      }
      await navigator.clipboard.writeText(value);
      toast.success(`${label}已复制`);
      return;
    } catch {
      const textArea = document.createElement("textarea");
      textArea.value = value;
      textArea.setAttribute("readonly", "");
      textArea.style.position = "fixed";
      textArea.style.top = "-9999px";
      document.body.appendChild(textArea);
      textArea.select();

      try {
        if (document.execCommand("copy")) {
          toast.success(`${label}已复制`);
          return;
        }
      } catch {
        // 继续展示统一的失败提示
      } finally {
        textArea.remove();
      }

      toast.error("复制失败，请手动选择文本");
    }
  };

  const openHermesPanel = () => {
    if (!DEMO_OPENCLAW_PANEL_URL) {
      toast.error("面板地址暂不可用，请稍后重试");
      return;
    }
    const panelWindow = window.open("about:blank", "_blank");
    if (!panelWindow) {
      toast.info("如面板未自动打开，请点击弹窗中的面板地址");
      return;
    }
    panelWindow.opener = null;
    panelWindow.location.replace(DEMO_OPENCLAW_PANEL_URL);
  };

  const handleOpenWebUI = () => {
    setShowWebUIProgressDialog(true);
    setWebUIStep(0);
    setWebUIFailedStep("none");
    setTimeout(() => setWebUIStep(1), 1500);
    setTimeout(() => setWebUIStep(2), 3500); // 1.5s + 2s = 3.5s
  };

  const handleWebUIRetry = () => {
    setWebUIFailedStep("none");
    setWebUIStep(0);
    setTimeout(() => setWebUIStep(1), 1500);
    setTimeout(() => setWebUIStep(2), 3500); // 1.5s + 2s = 3.5s
  };

  const handleWebUIProgressConfirm = () => {
    setShowWebUIProgressDialog(false);
    if (isHermes0180) {
      setShowHermesPassword(false);
      setShowHermesPanelDialog(true);
    } else {
      setWebUIUrl(DEMO_OPENCLAW_PANEL_URL);
      setWebUIToken(DEMO_OPENCLAW_PANEL_TOKEN);
      setShowWebUIResultDialog(true);
    }
  };

  // ── 飞书授权弹窗状态 ──
  const [showQrModal, setShowQrModal] = useState(false);
  const [feishuModalStage, setFeishuModalStage] = useState<"loading" | "qr" | "configuring" | "done" | "failed">("loading");
  const [feishuStepsDone, setFeishuStepsDone] = useState(0);
  const feishuSteps = ["创建应用", "配置权限", "配置事件回调", "申请高级权限", "发布应用"];
  const feishuHighPrivilegeStepIdx = 3;

  const handleOpenFeishu = () => {
    setShowQrModal(true);
    setFeishuModalStage("loading");
    setTimeout(() => setFeishuModalStage("qr"), 2000);
    // 模拟扫码后自动进入配置阶段
    setTimeout(() => {
      setFeishuModalStage("configuring");
      setFeishuStepsDone(0);
      let done = 0;
      const runFeishuStep = () => {
        if (done >= feishuSteps.length) {
          setFeishuModalStage("done");
          return;
        }
        setTimeout(() => {
          done += 1;
          setFeishuStepsDone(done);
          runFeishuStep();
        }, 1200 + Math.random() * 1000);
      };
      runFeishuStep();
    }, 6000);
  };

  // ── 微信扫码弹窗状态 ──
  const [showWechatQrModal, setShowWechatQrModal] = useState(false);
  const [wechatModalStage, setWechatModalStage] = useState<"checking" | "generating" | "qr">("checking");

  // ── WhatsApp Pairing Code 弹窗状态 ──
  const [showWhatsAppPairing, setShowWhatsAppPairing] = useState(false);
  // pairing: 展示配对码，等待用户在 WhatsApp 关联设备（无中间「配对中」态）；success: 后端确认配对完成，展示成功态
  const [whatsappPairingStage, setWhatsappPairingStage] = useState<"pairing" | "success">("pairing");
  const [whatsappPairingCode, setWhatsappPairingCode] = useState("");
  // 记录正在配对的通道信息，用于确定后入账
  const [whatsappPendingChannel, setWhatsappPendingChannel] = useState<ChannelConfig | null>(null);
  const [whatsappPendingPhone, setWhatsappPendingPhone] = useState("");

  // 生成一个 8 位配对码（模拟服务端返回，格式如 MWMV8RZK）
  const generatePairingCode = () => {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
    let s = "";
    for (let i = 0; i < 8; i++) s += chars[Math.floor(Math.random() * chars.length)];
    return s;
  };

  const handleOpenWechat = () => {
    setShowWechatQrModal(true);
    setWechatModalStage("checking");
    setTimeout(() => setWechatModalStage("generating"), 1200);
    setTimeout(() => setWechatModalStage("qr"), 2500);
  };

  // ── 模型操作确认弹窗 ──
  const [modelConfirmDialog, setModelConfirmDialog] = useState<{
    open: boolean;
    type: "delete" | "delete-backup" | "set-primary";
    modelId: number | null;
  }>({ open: false, type: "delete", modelId: null });

  // ── 模型连通性失败弹窗 ──
  const [connectFailResult, setConnectFailResult] = useState<string | null>(null);

  // ── 技能安装确认弹窗 ──
  const [skillInstallConfirm, setSkillInstallConfirm] = useState<{ open: boolean; skillName: string }>({ open: false, skillName: "" });

  // ── 龙虾医生弹窗 ──
  const [showStartModal, setShowStartModal] = useState(false);
  const [showEndModal, setShowEndModal] = useState(false);
  const [conflictInfo, setConflictInfo] = useState<{ instanceId: string; instanceName: string } | null>(null);
  const [diagAuthorize, setDiagAuthorize] = useState(false);
  const [diagSnapshot, setDiagSnapshot] = useState(true);
  const [snapshotCreated] = useState(false);
  const [rollbackChecked, setRollbackChecked] = useState(false);

  // ── 读取管控端「允许用户使用龙虾医生」开关状态（默认关闭）──
  // 与 OpenClawDetail.tsx 采用同一 localStorage key，保证两端一致。
  const [lobsterDoctorEnabled, setLobsterDoctorEnabled] = useState(
    () => localStorage.getItem("admin_allow_lobster_doctor") === "true"
  );
  // 监听：跨 tab 用原生 storage 事件；同 tab 用管控端派发的 CustomEvent。
  useEffect(() => {
    const handleStorage = (e: StorageEvent) => {
      if (e.key === "admin_allow_lobster_doctor") {
        setLobsterDoctorEnabled(e.newValue === "true");
      }
    };
    const handleCustom = (e: Event) => {
      const detail = (e as CustomEvent<{ enabled: boolean }>).detail;
      if (detail && typeof detail.enabled === "boolean") {
        setLobsterDoctorEnabled(detail.enabled);
      } else {
        setLobsterDoctorEnabled(localStorage.getItem("admin_allow_lobster_doctor") === "true");
      }
    };
    window.addEventListener("storage", handleStorage);
    window.addEventListener("lobster-doctor-policy-changed", handleCustom);
    return () => {
      window.removeEventListener("storage", handleStorage);
      window.removeEventListener("lobster-doctor-policy-changed", handleCustom);
    };
  }, []);

  const handleStartDiagnosisClick = () => {
    // 模拟冲突检测
    setShowStartModal(true);
  };

  const handleStartConfirm = () => {
    setShowStartModal(false);
    toast.success("龙虾医生正在初始化，请稍候…");
  };

  const handleEndClick = () => {
    setShowEndModal(true);
  };

  const handleEndConfirm = () => {
    setShowEndModal(false);
    toast.success("诊断已结束");
  };

  // ── Agent 迁移状态 ──
  const [migrationOpen, setMigrationOpen] = useState(false);
  const [migrationStep, setMigrationStep] = useState<"export" | "waitUpload" | "import" | "importing" | "success" | "failed">("export");
  const [migrationUploaded, setMigrationUploaded] = useState(false);
  const [migrationChecking, setMigrationChecking] = useState(false);
  const [migrationError, setMigrationError] = useState("");
  const [migrationCommandReady, setMigrationCommandReady] = useState(false);
  const [migrationCheckFailed, setMigrationCheckFailed] = useState(false);
  const [migrationCheckCount, setMigrationCheckCount] = useState(0);

  type ImportStepStatus = "pending" | "running" | "done" | "failed";
  type ImportStepItem = { label: string; status: ImportStepStatus; error?: string };
  const [importSteps, setImportSteps] = useState<ImportStepItem[]>([
    { label: "下载数据包", status: "pending" },
    { label: "备份当前配置", status: "pending" },
    { label: "解压并覆盖", status: "pending" },
    { label: "重启 Gateway", status: "pending" },
    { label: "验证生效", status: "pending" },
  ]);
  type VerifyItem = { label: string; cmd: string; passed: boolean; detail?: string };
  const [verifyResults, setVerifyResults] = useState<VerifyItem[]>([]);

  const migrationCosBucket = "clawpro-migrate-1302061491";
  const migrationCosKey = `single/guide-${Math.random().toString(36).substring(2, 8)}.tgz`;
  const migrationPresignedUrl = `https://${migrationCosBucket}.cos.ap-guangzhou.myqcloud.com/${migrationCosKey}?q-sign-algorithm=sha1&q-ak=AKID****&q-sign-time=****&q-signature=****`;
  const migrationExportCommand = `# 在源端 Agent 终端执行以下命令\nagent gateway stop\ntar -czf /tmp/openclaw-export.tgz -C $HOME .agent\ncurl -X PUT --upload-file /tmp/openclaw-export.tgz \\\n  "${migrationPresignedUrl}"\nrm -f /tmp/openclaw-export.tgz\nagent gateway start\necho "✅ 导出完成，数据已上传到 COS"`;

  const handleCheckUpload = () => {
    setMigrationChecking(true);
    setMigrationCheckFailed(false);
    setMigrationCheckCount((c) => c + 1);
    setTimeout(() => {
      const detected = migrationCheckCount >= 1 || Math.random() < 0.6;
      setMigrationChecking(false);
      if (detected) {
        setMigrationUploaded(true);
        setMigrationCheckFailed(false);
        setMigrationStep("import");
        toast.success("检测到已上传的数据包");
      } else {
        setMigrationCheckFailed(true);
        toast.error("未检测到数据包");
      }
    }, 1500);
  };

  const handleStartMigration = () => {
    setMigrationStep("importing");
    const steps: ImportStepItem[] = [
      { label: "下载数据包", status: "pending" },
      { label: "备份当前配置", status: "pending" },
      { label: "解压并覆盖", status: "pending" },
      { label: "重启 Gateway", status: "pending" },
      { label: "验证生效", status: "pending" },
    ];
    setImportSteps(steps);
    setVerifyResults([]);

    const delays = [1200, 1000, 1500, 2000, 1800];
    let current = 0;

    const runNext = () => {
      if (current >= steps.length) {
        setVerifyResults([
          { label: "配置文件", cmd: "ls ~/.agent/config.yaml", passed: true, detail: "存在" },
          { label: "Gateway 状态", cmd: "agent gateway status", passed: true, detail: "running" },
          { label: "通道连通性", cmd: "agent channel test", passed: true, detail: "3/3 通过" },
        ]);
        setMigrationStep("success");
        return;
      }
      setImportSteps((prev) => prev.map((s, i) => i === current ? { ...s, status: "running" } : s));
      setTimeout(() => {
        setImportSteps((prev) => prev.map((s, i) => i === current ? { ...s, status: "done" } : s));
        current++;
        runNext();
      }, delays[current]);
    };
    runNext();
  };

  const openInstanceMigration = () => {
    setMigrationOpen(true);
    setMigrationStep("export");
    setMigrationUploaded(false);
    setMigrationChecking(false);
    setMigrationCheckFailed(false);
    setMigrationCheckCount(0);
    setMigrationError("");
    setMigrationCommandReady(false);
    setVerifyResults([]);
    setImportSteps([
      { label: "下载数据包", status: "pending" },
      { label: "备份当前配置", status: "pending" },
      { label: "解压并覆盖", status: "pending" },
      { label: "重启 Gateway", status: "pending" },
      { label: "验证生效", status: "pending" },
    ]);
    setTimeout(() => setMigrationCommandReady(true), 1800);
  };

  const [showAddBackupModel, setShowAddBackupModel] = useState(false);
  // 注：「新增备用模型」厂商/版本下拉已统一替换为标准 Select 组件，
  // 此前的 backupCascadeOpen / backupHoveredProvider 状态已废弃移除。

  // ── Memory Tab state（mock）──
  const [memoryStatus, setMemoryStatus] = useState<"pro" | "free" | "none" | "upgrading">(() => {
    if (typeof window === "undefined") return "none";
    const status = new URLSearchParams(window.location.search).get("memoryStatus");
    return ["pro", "free", "none", "upgrading"].includes(status ?? "")
      ? (status as "pro" | "free" | "none" | "upgrading")
      : "none";
  });
  const [proQuotaAvailable] = useState(true);
  const [fileSpaceStatus] = useState<"enabled" | "closed">(() => {
    if (typeof window === "undefined") return "enabled";
    const status = new URLSearchParams(window.location.search).get("fileStatus");
    return status === "closed" ? "closed" : "enabled";
  });
  const [memoryLoading, setMemoryLoading] = useState(false);
  const [memoryDataLoaded, setMemoryDataLoaded] = useState(false);
  useEffect(() => {
    if (activeTab === "memory" && !memoryDataLoaded && (memoryStatus === "free" || memoryStatus === "pro")) {
      setMemoryLoading(true);
      const loadTime = memoryStatus === "free" ? 4000 : 1500;
      const timer = setTimeout(() => {
        setMemoryLoading(false);
        setMemoryDataLoaded(true);
      }, loadTime);
      return () => clearTimeout(timer);
    }
  }, [activeTab, memoryDataLoaded, memoryStatus]);

  // ── Doctor Tab state（一键修复 mock）──
  const [quickFixState, setQuickFixState] = useState<"idle" | "loading" | "success" | "failed">("idle");
  const [quickFixFailReason, setQuickFixFailReason] = useState("");
  const quickFixFailReasonsRef = useRef([
    "API KEY 校验未通过",
    "插件依赖加载超时",
    "通道配置文件解析异常",
  ]);
  const quickFixFailIdxRef = useRef(0);
  const runQuickFixMock = useCallback(() => {
    setQuickFixState("loading");
    setQuickFixFailReason("");
    setTimeout(() => {
      // 引导页演示版：默认走成功路径；如需演示失败可改为按计数轮换
      setQuickFixState("success");
      toast.success("一键修复执行完成");
    }, 3000);
  }, []);

  // ── Model state ──
  // 模型来源：admin = 管理员配置的模型；self = 自行配置
  const [modelSource, setModelSource] = useState<"admin" | "self">("admin");
  // 「自行配置」下的二级分类（Coding Plan / 模型 API / 自定义模型）
  const [selfCategory, setSelfCategory] = useState<SelfConfigCategory>("codingPlan");
  const [selectedProvider, setSelectedProvider] = useState(MODEL_PROVIDERS[0].value);
  const [selectedModel, setSelectedModel] = useState(MODEL_PROVIDERS[0].versions[0].value);
  const currentProvider = MODEL_PROVIDERS.find(p => p.value === selectedProvider) || MODEL_PROVIDERS[0];
  const currentVersions = currentProvider.versions;
  // 当前生效的厂商候选列表：admin 来源 = 管理员模型；self 来源 = 当前类别下的厂商
  const sourceProviders = modelSource === "admin"
    ? getAdminModelProviders()
    : getSelfProvidersByCategory(selfCategory);
  // 当前选中厂商是否为「自定义模型」/「自行配置的公开模型」
  const isCustomProvider = modelSource === "self" && selfCategory === "custom";
  const isPublicSelfProvider = modelSource === "self" && selfCategory !== "custom";
  // 管理员预置的「由用户端自行配置」模型：用户端选用时需自行填写 API Key 并支持连通性检测
  const isAdminUserKeyProvider = modelSource === "admin" && !!currentProvider.userProvidedKey;

  // 「模型来源」合并选择器的取值：admin 或 self:<category>（方案一：来源 + 配置类型合并为一层选择，减少一次决策）
  const modelSourceGroupValue = modelSource === "admin" ? "admin" : (`self:${selfCategory}` as const);
  // 切换合并后的「模型来源」：自动定位到该分组下第一个厂商
  const handleSourceGroupChange = (value: string) => {
    setFormErrors({}); // 切换来源后字段集合会变化，清空上一轮的校验错误态，避免残留红框
    if (value === "admin") {
      setModelSource("admin");
      const list = getAdminModelProviders();
      if (list.length > 0) {
        setSelectedProvider(list[0].value);
        setSelectedModel(list[0].versions[0]?.value ?? "");
      }
      return;
    }
    const category = value.replace(/^self:/, "") as SelfConfigCategory;
    setModelSource("self");
    setSelfCategory(category);
    const list = getSelfProvidersByCategory(category);
    if (list.length > 0) {
      setSelectedProvider(list[0].value);
      setSelectedModel(list[0].versions[0]?.value ?? "");
    }
  };

  // ── 自定义模型 state ──
  const [customInputMode, setCustomInputMode] = useState<"json" | "form">("json");
  const [customJson, setCustomJson] = useState(DEFAULT_CUSTOM_JSON);
  const [customForm, setCustomForm] = useState({ provider: "", base_url: "", api: "", api_key: "", model_id: "", model_name: "" });
  const [customMultimodal, setCustomMultimodal] = useState(false);

  // ── 自行配置·公开模型 state（厂商/版本已由 selectedProvider/selectedModel 表达）──
  // 公开模型（Coding Plan / 模型 API）无需填写模型 URL，仅需 API Key + 限额。
  const [publicForm, setPublicForm] = useState({ apiKey: "", dailyLimit: "100000" });
  // 高级配置（自定义模型）
  type HeaderEntry = { key: string; value: string };
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [advancedConfig, setAdvancedConfig] = useState<{ contextWindow: string; maxTokens: string; headers: HeaderEntry[] }>({
    contextWindow: "",
    maxTokens: "",
    headers: [{ key: "", value: "" }],
  });
  // 连通性检测
  const [connectTesting, setConnectTesting] = useState(false);
  // 表单校验错误态：key 对应字段名（apiKey / dailyLimit / base_url / api_key / model_id / json），
  // 修复"校验时表单缺少报错态"——之前校验失败只弹 toast，字段本身没有红框/错误提示，用户看不出具体哪个字段有问题
  const [formErrors, setFormErrors] = useState<Record<string, boolean>>({});
  const clearFieldError = (key: string) => {
    if (formErrors[key]) setFormErrors((prev) => { const next = { ...prev }; delete next[key]; return next; });
  };

  const handleConnectTest = async () => {
    if (isAdminUserKeyProvider) {
      // 管理员预置「由用户端自行配置」模型：仅需用户端填写 API Key
      if (!publicForm.apiKey) {
        setFormErrors({ apiKey: true });
        toast.error("请填写 API Key");
        return;
      }
    } else if (isPublicSelfProvider) {
      // 公开模型：需要版本 + API Key
      const errors: Record<string, boolean> = {};
      if (!selectedModel) errors.version = true;
      if (!publicForm.apiKey) errors.apiKey = true;
      if (Object.keys(errors).length > 0) {
        setFormErrors(errors);
        toast.error("请填写完整的模型配置信息");
        return;
      }
    } else if (customInputMode === "form") {
      const errors: Record<string, boolean> = {};
      if (!customForm.base_url) errors.base_url = true;
      if (!customForm.api_key) errors.api_key = true;
      if (!customForm.model_id) errors.model_id = true;
      if (Object.keys(errors).length > 0) {
        setFormErrors(errors);
        toast.error("请填写完整的模型配置信息");
        return;
      }
    } else {
      if (!customJson.trim()) {
        setFormErrors({ json: true });
        toast.error("请填写完整的模型配置信息");
        return;
      }
    }
    setFormErrors({});
    setConnectTesting(true);
    await new Promise((r) => setTimeout(r, 1500));
    setConnectTesting(false);
    setConnectFailResult(JSON.stringify({
      error: {
        message: "Invalid API Key",
        param: "Please provide valid API Key",
        code: "401",
        type: "invalid_key",
      }
    }, null, 2));
  };

  type AppliedModel = { id: number; providerLabel: string; versionLabel: string; primary: boolean; adminPreset?: boolean; isCustom?: boolean; customName?: string; multimodal?: boolean; selfPublic?: boolean; userProvidedKey?: boolean };
  const [appliedModels, setAppliedModels] = useState<AppliedModel[]>([
    { id: 1, providerLabel: "腾讯云 Token Plan 企业版专业套餐", versionLabel: "DeepSeek-V4-Pro", primary: true, adminPreset: true },
  ]);
  const [modelIdCounter, setModelIdCounter] = useState(2);

  const handleProviderChange = (providerValue: string) => {
    setSelectedProvider(providerValue);
    const provider = MODEL_PROVIDERS.find(p => p.value === providerValue);
    if (provider) setSelectedModel(provider.versions[0].value);
  };

  const handleApplyModel = (): boolean => {
    const provider = MODEL_PROVIDERS.find(p => p.value === selectedProvider);
    if (!provider) return false;
    let newEntry: AppliedModel;
    if (isCustomProvider) {
      // 校验
      if (customInputMode === "form") {
        const errors: Record<string, boolean> = {};
        if (!customForm.base_url) errors.base_url = true;
        if (!customForm.api_key) errors.api_key = true;
        if (!customForm.model_id) errors.model_id = true;
        if (Object.keys(errors).length > 0) {
          setFormErrors(errors);
          toast.error("请填写完整的模型配置信息");
          return false;
        }
      } else {
        if (!customJson.trim()) {
          setFormErrors({ json: true });
          toast.error("请填写完整的模型配置信息");
          return false;
        }
      }
      setFormErrors({});
      // 从 JSON / 表单解析 customName
      let customName = "";
      if (customInputMode === "form") {
        customName = customForm.model_name || customForm.model_id;
      } else {
        try {
          const obj = JSON.parse(customJson);
          customName = obj?.model?.name || obj?.model?.id || obj?.model_name || obj?.model_id || "";
        } catch {
          customName = "";
        }
      }
      newEntry = {
        id: modelIdCounter,
        providerLabel: "自定义模型",
        versionLabel: "",
        primary: !appliedModels.some(m => m.primary),
        isCustom: true,
        customName,
        multimodal: customMultimodal,
      };
    } else if (isPublicSelfProvider) {
      // 自行配置的公开模型：校验 版本 + URL + API Key + 限额
      const version = currentVersions.find(v => v.value === selectedModel);
      if (!version) { toast.error("请选择模型版本"); return false; }
      const errors: Record<string, boolean> = {};
      if (!publicForm.apiKey) errors.apiKey = true;
      if (!publicForm.dailyLimit || Number(publicForm.dailyLimit) <= 0) errors.dailyLimit = true;
      if (Object.keys(errors).length > 0) {
        setFormErrors(errors);
        toast.error(errors.apiKey ? "请填写 API Key" : "请填写有效的每日 Tokens 上限");
        return false;
      }
      setFormErrors({});
      newEntry = {
        id: modelIdCounter,
        providerLabel: provider.label,
        versionLabel: version.label,
        primary: !appliedModels.some(m => m.primary),
        selfPublic: true,
      };
      // 重置公开模型表单
      setPublicForm({ apiKey: "", dailyLimit: "100000" });
      setAdvancedOpen(false);
      setAdvancedConfig({ contextWindow: "", maxTokens: "", headers: [{ key: "", value: "" }] });
    } else if (isAdminUserKeyProvider) {
      // 管理员预置「由用户端自行配置」模型：用户端必须填写自己的 API Key
      const version = currentVersions.find(v => v.value === selectedModel);
      if (!version) { toast.error("请选择模型版本"); return false; }
      if (!publicForm.apiKey) {
        setFormErrors({ apiKey: true });
        toast.error("请填写 API Key");
        return false;
      }
      setFormErrors({});
      newEntry = {
        id: modelIdCounter,
        // 用户端展示名称：自定义模型/model_id(需自行填写 Key)
        providerLabel: `自定义模型/${version.value}（需自行填写 Key）`,
        versionLabel: "",
        primary: !appliedModels.some(m => m.primary),
        adminPreset: true,
        userProvidedKey: true,
      };
      setPublicForm({ apiKey: "", dailyLimit: "100000" });
    } else {
      const version = currentVersions.find(v => v.value === selectedModel);
      if (!version) return false;
      newEntry = {
        id: modelIdCounter,
        providerLabel: provider.label,
        versionLabel: version.label,
        primary: !appliedModels.some(m => m.primary),
      };
    }
    setAppliedModels(prev => [...prev, newEntry]);
    setModelIdCounter(c => c + 1);
    toast.success("模型已添加");
    return true;
  };

  // 删除模型：点击垃圾桶图标只是「发起请求」，打开下方已有的二次确认弹窗（modelConfirmDialog），
  // 不直接删除——修复此前"删除零确认"的问题（弹窗组件之前已经写好了，但两处删除按钮从未真正调用它）
  const handleRequestDeleteModel = (id: number, isPrimary: boolean) => {
    setModelConfirmDialog({ open: true, type: isPrimary ? "delete" : "delete-backup", modelId: id });
  };

  // 二次确认弹窗「确认删除」的真正执行逻辑：
  // 删除的是主模型时，自动把第一个备选模型升级为主模型（对齐弹窗文案"删除后将自动切换备选模型作为主模型"）——
  // 修复此前 handleDeleteModel 只是无脑 filter，导致删除主模型后没有模型是 primary，出现"无主模型但有备选模型"的非法状态
  const handleConfirmDeleteModel = () => {
    const { modelId } = modelConfirmDialog;
    if (modelId === null) return;
    setAppliedModels(prev => {
      const target = prev.find(m => m.id === modelId);
      const rest = prev.filter(m => m.id !== modelId);
      if (target?.primary) {
        const promoteIndex = rest.findIndex(m => !m.primary);
        if (promoteIndex !== -1) {
          rest[promoteIndex] = { ...rest[promoteIndex], primary: true };
        }
      }
      return rest;
    });
    setModelConfirmDialog(prev => ({ ...prev, open: false }));
    toast.success("模型已删除");
  };

  // ── 主模型编辑态：改为与"添加主模型"复用同一个 Dialog，而不是就地内嵌小卡片
  // （用户反馈：编辑的交互域应该和添加保持一致，都用弹窗）──
  const [editingModelId, setEditingModelId] = useState<number | null>(null);

  const handleEditPrimary = (id: number) => {
    const current = appliedModels.find((m) => m.id === id);
    if (!current) return;
    setFormErrors({});
    setEditingModelId(id);

    if (current.isCustom) {
      // 自定义模型：仅 customName / multimodal 有持久化数据，
      // base_url / api_key / model_id 等原始字段未存储，编辑时需要用户重新填写
      setModelSource("self");
      setSelfCategory("custom");
      setSelectedProvider("custom");
      setSelectedModel("custom");
      setCustomInputMode("form");
      setCustomForm({ provider: "", base_url: "", api: "", api_key: "", model_id: "", model_name: current.customName || "" });
      setCustomMultimodal(!!current.multimodal);
    } else if (current.selfPublic) {
      // 自行配置的公开模型（Coding Plan / 模型 API）：按 providerLabel 反查厂商 + 二级分类
      let found: { provider: (typeof MODEL_PROVIDERS)[number]; category: SelfConfigCategory } | undefined;
      for (const cat of SELF_CONFIG_CATEGORY_ORDER) {
        if (cat === "custom") continue;
        const p = getSelfProvidersByCategory(cat).find((pr) => pr.label === current.providerLabel);
        if (p) { found = { provider: p, category: cat }; break; }
      }
      setModelSource("self");
      setSelfCategory(found?.category ?? "codingPlan");
      setSelectedProvider(found?.provider.value ?? MODEL_PROVIDERS[0].value);
      const version = found?.provider.versions.find((v) => v.label === current.versionLabel);
      setSelectedModel(version?.value ?? found?.provider.versions[0]?.value ?? "");
      // API Key / 每日 Tokens 上限 / 高级配置未持久化存储，重置为空，需用户重新填写
      setPublicForm({ apiKey: "", dailyLimit: "100000" });
      setAdvancedOpen(false);
      setAdvancedConfig({ contextWindow: "", maxTokens: "", headers: [{ key: "", value: "" }] });
    } else {
      // 管理员预置模型
      const list = getAdminModelProviders();
      // 「由用户端自行配置」模型的 providerLabel 已被替换为展示名，无法按 label 反查，
      // 改用 userProvidedKey 标记定位对应厂商。
      const provider = current.userProvidedKey
        ? list.find((p) => p.userProvidedKey)
        : list.find((p) => p.label === current.providerLabel);
      setModelSource("admin");
      setSelectedProvider(provider?.value ?? list[0]?.value ?? MODEL_PROVIDERS[0].value);
      const version = current.userProvidedKey
        ? provider?.versions[0]
        : provider?.versions.find((v) => v.label === current.versionLabel);
      setSelectedModel(version?.value ?? provider?.versions[0]?.value ?? MODEL_PROVIDERS[0].versions[0].value);
      // 「由用户端自行配置」模型的 API Key 未持久化存储，编辑时重置为空，需用户重新填写
      if (provider?.userProvidedKey) {
        setPublicForm({ apiKey: "", dailyLimit: "100000" });
      }
    }
    setShowAddBackupModel(true);
  };

  const handleSaveEditModel = () => {
    if (editingModelId === null) return;
    const provider = MODEL_PROVIDERS.find(p => p.value === selectedProvider);
    if (!provider) return;
    let updates: Partial<AppliedModel> = {};
    if (isCustomProvider) {
      if (customInputMode === "form") {
        const errors: Record<string, boolean> = {};
        if (!customForm.base_url) errors.base_url = true;
        if (!customForm.api_key) errors.api_key = true;
        if (!customForm.model_id) errors.model_id = true;
        if (Object.keys(errors).length > 0) {
          setFormErrors(errors);
          toast.error("请填写完整的模型配置信息");
          return;
        }
      } else if (!customJson.trim()) {
        setFormErrors({ json: true });
        toast.error("请填写完整的模型配置信息");
        return;
      }
      setFormErrors({});
      let customName = "";
      if (customInputMode === "form") {
        customName = customForm.model_name || customForm.model_id;
      } else {
        try {
          const obj = JSON.parse(customJson);
          customName = obj?.model?.name || obj?.model?.id || obj?.model_name || obj?.model_id || "";
        } catch {
          customName = "";
        }
      }
      updates = { providerLabel: "自定义模型", versionLabel: "", isCustom: true, customName, multimodal: customMultimodal, adminPreset: false, selfPublic: false };
    } else if (isPublicSelfProvider) {
      const version = currentVersions.find(v => v.value === selectedModel);
      if (!version) { toast.error("请选择模型版本"); return; }
      const errors: Record<string, boolean> = {};
      if (!publicForm.apiKey) errors.apiKey = true;
      if (!publicForm.dailyLimit || Number(publicForm.dailyLimit) <= 0) errors.dailyLimit = true;
      if (Object.keys(errors).length > 0) {
        setFormErrors(errors);
        toast.error(errors.apiKey ? "请填写 API Key" : "请填写有效的每日 Tokens 上限");
        return;
      }
      setFormErrors({});
      updates = { providerLabel: provider.label, versionLabel: version.label, isCustom: false, selfPublic: true, adminPreset: false };
      setPublicForm({ apiKey: "", dailyLimit: "100000" });
      setAdvancedOpen(false);
      setAdvancedConfig({ contextWindow: "", maxTokens: "", headers: [{ key: "", value: "" }] });
    } else if (isAdminUserKeyProvider) {
      const version = currentVersions.find(v => v.value === selectedModel);
      if (!version) { toast.error("请选择模型版本"); return; }
      if (!publicForm.apiKey) {
        setFormErrors({ apiKey: true });
        toast.error("请填写 API Key");
        return;
      }
      setFormErrors({});
      updates = { providerLabel: `自定义模型/${version.value}（需自行填写 Key）`, versionLabel: "", isCustom: false, selfPublic: false, adminPreset: true, userProvidedKey: true };
      setPublicForm({ apiKey: "", dailyLimit: "100000" });
    } else {
      const version = currentVersions.find(v => v.value === selectedModel);
      if (!version) return;
      updates = { providerLabel: provider.label, versionLabel: version.label, isCustom: false, selfPublic: false, adminPreset: true, userProvidedKey: false };
    }
    setAppliedModels(prev => prev.map(m => (m.id === editingModelId ? { ...m, ...updates } : m)));
    toast.success("模型已更新");
    setEditingModelId(null);
    setShowAddBackupModel(false);
  };

  // ── Channel state ──
  const [selectedChannel, setSelectedChannel] = useState("wework");
  const [channelFields, setChannelFields] = useState<Record<string, string>>({});
  const [visibleSecrets, setVisibleSecrets] = useState<Set<string>>(new Set());

  // 自定义通道（管控端配置）
  const [visibleCustomChannels, setVisibleCustomChannels] = useState<AdminCustomChannel[]>(() => loadVisibleCustomChannels());
  useEffect(() => {
    const unsub = onCustomChannelsChange(() => setVisibleCustomChannels(loadVisibleCustomChannels()));
    return unsub;
  }, []);
  const [builtinChannelVisibility, setBuiltinChannelVisibility] = useState<Record<string, boolean>>(() => loadBuiltinChannelVisibility());
  useEffect(() => {
    const unsub = onBuiltinChannelVisibilityChange(() => setBuiltinChannelVisibility(loadBuiltinChannelVisibility()));
    return unsub;
  }, []);

  // 已接入通道
  type AppliedChannel = {
    type: string;
    channelValue: string;
    status: "running";
    fields: ChannelField[];
    fieldValues: Record<string, string>;
    /** 飞书：记录添加时所用的快捷/手动模式 */
    feishuConfigMode?: "quick" | "manual";
    /** 企业微信：记录添加时所用的快捷/手动模式 */
    weworkConfigMode?: "quick" | "manual";
  };
  const [appliedChannels, setAppliedChannels] = useState<AppliedChannel[]>([
    // 预置一条飞书手动审批配置，便于演示「pairing code 子框」
    {
      type: "飞书",
      channelValue: "feishu",
      status: "running",
      fields: CHANNEL_OPTIONS.find((c) => c.value === "feishu")!.fields!,
      fieldValues: { appId: "cli_a1b2c3d4e5f6", appSecret: "abc123456789" },
      feishuConfigMode: "manual",
    },
  ]);
  const [expandedChannelIdx, setExpandedChannelIdx] = useState<number | null>(null);
  const [visibleAppliedSecrets, setVisibleAppliedSecrets] = useState<Set<string>>(new Set());

  // ── 多角色：per-role 配置仓库 ──
  // 每个角色各自持有一份 模型 / 通道 / 已装技能 / 待装技能 配置。
  // 切换角色时把当前四个 state 快照存回旧角色分片，再从新角色分片恢复，
  // 从而让三张配置卡（模型 / 通道 / 技能）无需改动即可按角色联动。
  type RoleConfigSlice = {
    models: AppliedModel[];
    channels: AppliedChannel[];
    installed: { name: string; version: string }[];
    pending: PendingSkill[];
  };
  const roleConfigStoreRef = useRef<Record<string, RoleConfigSlice>>({});
  // 初始化各角色分片：首个角色沿用当前预置数据，其余角色给出差异化的初始配置以便演示联动
  const roleStoreInitedRef = useRef(false);
  if (!roleStoreInitedRef.current && agentRoles.length > 0) {
    roleStoreInitedRef.current = true;
    agentRoles.forEach((role, idx) => {
      const isPrimaryRole = idx === 0;
      roleConfigStoreRef.current[role.id] = {
        // 模型：用户未单独配置时，所有角色统一为管理员预设的主模型（保持一致）
        models: [{ id: 1, providerLabel: "腾讯云 Token Plan 企业版专业套餐", versionLabel: "DeepSeek-V4-Pro", primary: true, adminPreset: true }],
        // 通道：属于用户接入的运行态，仅首个角色带演示通道，其余等待用户接入
        channels: isPrimaryRole
          ? [
              {
                type: "飞书",
                channelValue: "feishu",
                status: "running",
                fields: CHANNEL_OPTIONS.find((c) => c.value === "feishu")!.fields!,
                fieldValues: { appId: "cli_a1b2c3d4e5f6", appSecret: "abc123456789" },
                feishuConfigMode: "manual",
              },
            ]
          : [],
        // 技能：平台预设角色自带技能，选择该角色即带出（非空白）；缺省才回退为空
        installed: role.presetSkills ?? [],
        // 安装队列：运行态，仅首个角色演示安装中/失败状态
        pending: isPrimaryRole ? MOCK_PENDING_SKILLS : [],
      };
    });
  }

  // 切换角色：保存当前角色分片 → 载入目标角色分片
  const handleSwitchRole = (nextRoleId: string) => {
    if (nextRoleId === activeRoleId) return;
    // 1) 快照当前四个 state 存回当前角色
    roleConfigStoreRef.current[activeRoleId] = {
      models: appliedModels,
      channels: appliedChannels,
      installed: installedSkills,
      pending: pendingSkills,
    };
    // 2) 从目标角色分片恢复（兜底：未初始化时也带出该角色自带的预设技能）
    const next = roleConfigStoreRef.current[nextRoleId] ?? {
      models: [{ id: 1, providerLabel: "腾讯云 Token Plan 企业版专业套餐", versionLabel: "DeepSeek-V4-Pro", primary: true, adminPreset: true }],
      channels: [],
      installed: agentRoles.find((r) => r.id === nextRoleId)?.presetSkills ?? [],
      pending: [],
    };
    setAppliedModels(next.models);
    setAppliedChannels(next.channels);
    setInstalledSkills(next.installed);
    setPendingSkills(next.pending);
    // 3) 重置卡片内的瞬时交互态，避免跨角色串场
    setShowChannelConfig(false);
    setExpandedChannelIdx(null);
    setSkillSearch("");
    setActiveRoleId(nextRoleId);
    const nextName = agentRoles.find((r) => r.id === nextRoleId)?.name ?? "角色";
    toast.success(`已切换到「${nextName}」，下方配置为该角色独立生效`);
  };

  // 飞书 pairing code（手动审批匹配输入）
  const [feishuPairingCode, setFeishuPairingCode] = useState("");
  const handleFeishuPairing = () => {
    if (!feishuPairingCode.trim()) {
      toast.error("请输入 pairing code");
      return;
    }
    toast.success("匹配成功");
    setFeishuPairingCode("");
  };

  // 合并通道选项（内置 + 管控端自定义）
  const allChannelOptions = [
    ...CHANNEL_OPTIONS.filter(c => builtinChannelVisibility[c.value] !== false),
    ...visibleCustomChannels.map(cc => {
      const isWhatsApp = /whatsapp/i.test((cc.name || "").trim());
      return {
        value: cc.id,
        label: cc.name,
        descText: "管控端自定义通道",
        fields: cc.credentialFields?.map((f: { key: string; label: string; secret?: boolean }) => ({ key: f.key, label: f.label, secret: f.secret ?? false })) || [],
        ...(isWhatsApp ? { whatsappMode: true as const } : {}),
      } as ChannelConfig;
    }),
  ];
  const currentChannelConfig = allChannelOptions.find(c => c.value === selectedChannel);

  /** 加密显示 */
  const maskSecret = (val: string) => {
    if (!val || val.length <= 3) return val || "";
    return val.slice(0, 3) + "••••••";
  };

  const toggleSecretVisibility = (key: string) => {
    setVisibleSecrets(prev => {
      const next = new Set(prev);
      next.has(key) ? next.delete(key) : next.add(key);
      return next;
    });
  };

  const toggleAppliedSecretVisibility = (channelIdx: number, fieldKey: string) => {
    const uniqueKey = `${channelIdx}-${fieldKey}`;
    setVisibleAppliedSecrets(prev => {
      const next = new Set(prev);
      next.has(uniqueKey) ? next.delete(uniqueKey) : next.add(uniqueKey);
      return next;
    });
  };

  // WhatsApp 配对：等待后端确认配对结果（无「配对中」中间态），确认后直接切成功态并以运行中入账
  const startWhatsAppPairingWatch = (ch: ChannelConfig, phone: string) => {
    // 模拟后端收到配对请求后回传「配对完成」，不展示配对中状态
    setTimeout(() => {
      const phoneField = ch.fields?.[0];
      const fieldKey = phoneField?.key || "phone_number";
      const fieldLabel = phoneField?.label || "手机号";
      const newEntry: AppliedChannel = {
        type: ch.label,
        channelValue: ch.value,
        status: "running",
        fields: [{ key: fieldKey, label: fieldLabel, secret: false }],
        fieldValues: { [fieldKey]: phone },
      };
      setAppliedChannels(prev => [...prev, newEntry]);
      setWhatsappPairingStage("success");
    }, 2500);
  };

  // WhatsApp 配对成功后点击「完成」→ 关闭弹窗并清理
  const handleFinishWhatsAppPairing = () => {
    const label = whatsappPendingChannel?.label || "WhatsApp";
    setShowWhatsAppPairing(false);
    setWhatsappPairingStage("pairing");
    setWhatsappPendingChannel(null);
    setWhatsappPendingPhone("");
    setWhatsappPairingCode("");
    setWhatsappPhone("");
    setShowChannelConfig(false);
    toast.success(`${label} 配对成功，已应用`);
  };

  // WhatsApp 配对：配对完成前关闭弹窗 → 不入账，需重新配置
  const handleCancelWhatsAppPairing = () => {
    setShowWhatsAppPairing(false);
    setWhatsappPairingStage("pairing");
    setWhatsappPendingChannel(null);
    setWhatsappPendingPhone("");
    setWhatsappPairingCode("");
    toast.info("已取消配对，请重新配置");
  };

  const handleApplyChannel = () => {
    const ch = allChannelOptions.find(c => c.value === selectedChannel);
    if (!ch) return;

    // WhatsApp 自定义通道：校验手机号 → 弹出 Pairing Code 弹窗，等待用户在 WhatsApp 关联设备后确认
    if (ch.whatsappMode) {
      const phone = whatsappPhone.trim();
      if (!phone) {
        toast.error("请输入手机号");
        return;
      }
      setWhatsappPendingChannel(ch);
      setWhatsappPendingPhone(phone);
      setWhatsappPairingCode(generatePairingCode());
      setWhatsappPairingStage("pairing");
      setShowWhatsAppPairing(true);
      // 后端收到配对请求后回传结果，无「配对中」态，完成后直接切成功态
      startWhatsAppPairingWatch(ch, phone);
      return;
    }

    // 微信：点击"前往授权"弹出二维码（checking → generating → qr → 自动写入并关闭）
    if (ch.wechatMode) {
      setWechatModalStage("checking");
      setShowWechatQrModal(true);
      setTimeout(() => setWechatModalStage("generating"), 1200);
      setTimeout(() => setWechatModalStage("qr"), 2500);
      // qr 出现后再 5s 自动关闭并写入「微信 ClawBot」
      setTimeout(() => {
        setShowWechatQrModal(false);
        setAppliedChannels(prev => {
          const existingIdx = prev.findIndex(c => c.channelValue === "wechat");
          const newEntry: AppliedChannel = {
            type: "微信 ClawBot",
            channelValue: "wechat",
            status: "running",
            fields: [],
            fieldValues: {},
          };
          if (existingIdx >= 0) {
            const next = [...prev];
            next[existingIdx] = newEntry;
            return next;
          }
          return [...prev, newEntry];
        });
        toast.success("微信 ClawBot 已添加");
      }, 7500);
      setShowChannelConfig(false);
      return;
    }

    // 飞书 + 快捷配置：触发 4 阶段授权弹窗
    if (ch.feishuMode && feishuConfigMode === "quick") {
      handleOpenFeishu();
      setShowChannelConfig(false);
      return;
    }

    // 企业微信 + 快捷配置：toast 提示 + 添加占位"企微机器人"
    if (ch.weworkMode && weworkConfigMode === "quick") {
      toast.info("即将跳转至企业微信授权页面，此功能即将开放");
      const newEntry: AppliedChannel = {
        type: "企微机器人",
        channelValue: "wework",
        status: "running",
        fields: ch.fields || [],
        fieldValues: { botId: "auto-authorized", secret: "auto-secret-key" },
        weworkConfigMode: "quick",
      };
      setAppliedChannels(prev => [...prev, newEntry]);
      setShowChannelConfig(false);
      toast.success("企微机器人已添加");
      return;
    }

    // 普通通道 / 手动配置：企业微信手动配置时类型显示为"企微机器人"
    const channelType = ch.weworkMode ? "企微机器人" : ch.label;
    const newEntry: AppliedChannel = {
      type: channelType,
      channelValue: ch.value,
      status: "running",
      fields: ch.fields || [],
      fieldValues: { ...channelFields },
      feishuConfigMode: ch.feishuMode ? feishuConfigMode : undefined,
      weworkConfigMode: ch.weworkMode ? weworkConfigMode : undefined,
    };
    setAppliedChannels(prev => [...prev, newEntry]);
    setChannelFields({});
    setShowChannelConfig(false);
    toast.success(`${channelType} 已添加并应用`);
  };

  const handleDeleteChannel = (idx: number) => {
    setAppliedChannels(prev => {
      const next = prev.filter((_, i) => i !== idx);
      if (next.length === 0) setShowChannelConfig(true);
      return next;
    });
    toast.info("通道已删除");
  };

  // 通道配置卡是否显示：默认收起；点击新增通道展开
  const [showChannelConfig, setShowChannelConfig] = useState(false);
  // WhatsApp 自定义通道专用：手机号输入
  const [whatsappPhone, setWhatsappPhone] = useState("");
  // 通道配置模式：快捷配置 / 手动配置（兜底；飞书 / 企业微信走独立 state）
  const [channelConfigMode, setChannelConfigMode] = useState<"quick" | "manual">("quick");
  // 飞书专用：快捷/手动 Tab（默认快捷配置）
  const [feishuConfigMode, setFeishuConfigMode] = useState<"quick" | "manual">("quick");
  // 企业微信专用：快捷/手动 Tab（默认快捷配置）
  const [weworkConfigMode, setWeworkConfigMode] = useState<"quick" | "manual">("quick");

  // ── 快捷角色切换 Popover 内容（复用于三方案）──
  // slots：该芯片聚合的角色位（同名可能多个），逐 slot 提供「切换为」候选；主角色标注皇冠。
  const renderRoleSwitchPanel = (slots: { id: string; name: string; isMain?: boolean }[]) => (
    <div className="w-[248px] max-h-[320px] overflow-y-auto p-1" style={{ scrollbarGutter: "stable" }}>
      {slots.map((s) => {
        const candidates = getSwitchCandidates(s.name);
        return (
          <div key={s.id} className="px-1 py-1">
            <div className="flex items-center gap-1.5 px-2 py-1">
              {s.isMain && <Crown className="w-3 h-3 text-[var(--text-brand)] flex-shrink-0" />}
              <span className="text-xs font-medium text-[var(--text-secondary)] truncate">{s.name}</span>
              {s.isMain && <span className="text-[10px] text-[var(--text-brand)] flex-shrink-0">主角色</span>}
            </div>
            {candidates.length === 0 ? (
              <p className="px-2 py-1 text-xs text-[var(--text-muted)]">当前分组无其他可切换角色</p>
            ) : (
              <div className="mt-0.5 space-y-0.5">
                {candidates.map((target) => (
                  <button
                    key={target}
                    type="button"
                    onClick={() => handleQuickSwitchRole(s.id, target)}
                    className="w-full flex items-center gap-2 px-2 py-1.5 rounded-[4px] text-left text-xs text-[var(--text-emphasis)] hover:bg-[var(--accent)] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)]"
                  >
                    <RotateCcw className="w-3 h-3 text-[var(--text-muted)] flex-shrink-0" />
                    切换为「{target}」
                  </button>
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );

  return (
    <TenantLayout>
      {/* 用户端单层 120px 骨架（SKILL-TENANT §6.1.1） */}
      <div className="min-w-[1200px]">
        <div className="max-w-[1920px] mx-auto page-enter">
          <div
            className="relative min-h-[calc(100vh-64px)] flex flex-col"
            style={{ paddingLeft: 120, paddingRight: 120, paddingBottom: 75 }}
          >
            {/* 内容主体 —— flex-1 + flex-col 保证不满一屏时底部吸底，超出一屏时跟随 */}
            <div className="relative flex flex-col flex-1">
              {/* ======== Header ======== */}
              <header className="relative flex flex-col gap-4 py-6">
                {/* 返回行：icon + 文字（左）；多角色实例的方案切换 Tab 紧跟返回按钮右侧 */}
                <div className="flex items-center gap-4 self-start">
                  <button
                    onClick={() => navigate("/my-openclaw")}
                    className="inline-flex items-center gap-1.5 text-[var(--text-secondary)] hover:text-[var(--text-brand)] transition-colors"
                    style={{ fontSize: 14, lineHeight: "20px" }}
                  >
                    <ArrowLeft className="w-4 h-4" />
                    <span>返回</span>
                  </button>
                </div>

                {/* 主信息行：头像顶对齐标题 + 右侧按钮 */}
                <div className="flex items-start justify-between gap-6">
                <div className="flex items-start gap-3 min-w-0 flex-1">
   {/* 头像：与主角色头像同源（按主角色的角色类型渲染） */}
    <AgentAvatar
   roleName={isLocalAgent ? undefined : mainRoleType}
  agentName={agentName}
      size={64}
      />

                  <div className="flex flex-col gap-2 min-w-0">
                    <div className="flex items-center gap-3">
                      {/* 可编辑名称 */}
                      <div ref={nameEditWrapperRef} className="group/name min-w-0">
                        {isNameEditing ? (
                          <div className="relative w-full">
                            <Input
                              ref={nameInputRef}
                              value={nameDraft}
                              onChange={(e) => {
                                const value = e.target.value.replace(/[\r\n]/g, "");
                                setNameDraft(value);
                                if (getAgentNameByteLength(value.trim()) > AGENT_NAME_MAX_BYTES) {
                                  setNameError(`名称不能超过 ${AGENT_NAME_MAX_BYTES} 字节`);
                                } else if (nameError) {
                                  setNameError("");
                                }
                              }}
                              onKeyDown={(e) => {
                                if (e.key === "Enter") { e.preventDefault(); saveNameEdit(); }
                                if (e.key === "Escape") { e.preventDefault(); cancelNameEdit(); }
                              }}
                              aria-label="编辑 Agent 名称"
                              aria-invalid={!!nameError}
                              className={`h-9 text-[24px] font-semibold bg-transparent rounded-[var(--radius-lg)] ${nameError ? "border-[var(--text-danger)] focus-visible:ring-[var(--text-danger)]" : "border-[var(--border)] focus-visible:ring-[var(--ring)]"}`}
                              style={{ color: "var(--foreground)", letterSpacing: "-1px", lineHeight: "32px" }}
                            />
                            {nameError && (
                              <p className="absolute left-0 top-full mt-1 text-xs text-[var(--text-danger)]">{nameError}</p>
                            )}
                          </div>
                        ) : (
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button
                                type="button"
                                onClick={startNameEdit}
                                className="inline-flex items-center gap-1.5 px-1 -ml-1 rounded-[var(--radius-lg)] transition-colors hover:bg-[var(--accent)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)]"
                                aria-label="重命名 Agent"
                              >
                                <h1
                                  ref={nameTextRef}
                                  className="text-[24px] font-semibold leading-8 max-w-[460px] truncate"
                                  style={{ color: "var(--foreground)", letterSpacing: "-1px" }}
                                >
                                  {agentName}
                                </h1>
                                <Edit3 className="w-4 h-4 text-[var(--text-weak)] flex-shrink-0" />
                              </button>
                            </TooltipTrigger>
                            {isNameOverflow && (
                              <TooltipContent side="top" className="text-xs max-w-[520px] break-all">
                                {agentName}
                              </TooltipContent>
                            )}
                          </Tooltip>
                        )}
                      </div>
                      {isLocalAgent ? (
                        <span
                          className="inline-flex items-center whitespace-nowrap text-xs"
                          style={{
                            gap: "4px",
                            padding: "2px 8px",
                            borderRadius: "999px",
                            color: isLocalInactive
                              ? "var(--text-warning, #B45309)"
                              : "var(--text-success)",
                            background: isLocalInactive
                              ? "var(--alert-warning-bg, #FEF3C7)"
                              : "var(--alert-success-bg, rgba(22,163,74,0.08))",
                          }}
                        >
                          <span
                            className="inline-block w-2 h-2 rounded-full"
                            style={{
                              background: isLocalInactive
                                ? "var(--text-warning, #D97706)"
                                : "var(--text-success)",
                            }}
                          />
                          {isLocalInactive ? "不活跃" : "运行中"}
                        </span>
                      ) : (
                        <StatusBadge status="running" />
                      )}
                      {/* 角色标签已移至下方「基础配置」Tab 栏后展示 */}
                    </div>
                    <div
                      className="flex flex-col"
                      style={{
                        gap: "4px",
                        fontFamily: "PingFang SC, -apple-system, BlinkMacSystemFont, sans-serif",
                        fontWeight: 400,
                        fontSize: "12px",
                        lineHeight: "20px",
                        color: "var(--text-secondary)",
                      }}
                    >
                      {/* 元信息第一行：角色 / 类型 / ID / 版本 + 可更新徽章（组织、项目固定拆到下一行，避免跟随换行跳跃） */}
                      <div className="flex items-center flex-wrap gap-x-1 gap-y-1 min-w-0">
                        {isLocalAgent ? (
                          <span>
                            {hasLocalResourceScopes
                              ? `本地客户端类型：${localClientProduct}`
                              : `外部 Agent 类型：${externalAgentTypeLabel}`}
                          </span>
                        ) : (
                          <>
                            {/* 角色：N 个（放在类型左侧，多角色实例显示真实数量） */}
          <span className="inline-flex items-center gap-1.5 shrink-0 whitespace-nowrap">
      角色：{roleCount} 个
   <button
        type="button"
         onClick={() => {
           // 打开与「我的 Agent」主页一致的角色管理抽屉（RoleManageSheet）。
           // 单角色实例无 roles 时合成单元素角色位，确保抽屉正常渲染。
           const slots: AgentRoleSlot[] =
             switchRoleSlots.length > 0
               ? switchRoleSlots.map((s) => ({ ...s }))
               : [{ slotId: `slot-main-${routeAgentId}`, roleName: mainRole?.name ?? "通用助手", isMain: true }];
           setSwitchRoleDialog({
             id: routeAgentId,
             name: agentName,
             roleName: mainRole?.name ?? "通用助手",
             allowedRoleNames: detailAgent?.allowedRoleNames,
             roles: slots,
           });
         }}
      className="inline-flex items-center gap-1 text-[var(--text-brand)] transition-colors hover:text-[var(--text-brand-deep)]"
    aria-label="管理角色"
      >
     <UserCog className="size-3" />
         管理
    </button>
             </span>
                            <span style={{ color: "var(--border)" }}>｜</span>
                            <span className="inline-flex items-center gap-1.5 whitespace-nowrap">
                              类型：{displayAgentTypeLabel}
                              {supportsTypeSwitch && (
                                <button
                                  type="button"
                                  onClick={() => setAgentTypeSwitchOpen(true)}
                                  disabled={agentTypeSwitching}
                                  className="inline-flex items-center gap-1 text-[var(--text-brand)] transition-colors hover:text-[var(--text-brand-deep)] disabled:cursor-not-allowed disabled:text-[var(--text-muted)]"
                                  aria-label={`将当前实例切换为${displayAgentType === "openclaw" ? "Hermes" : "OpenClaw"}`}
                                  data-testid="open-agent-type-switch"
                                >
                                  {agentTypeSwitching ? (
                                    <Loader2 className="size-3 animate-spin" />
                                  ) : (
                                    <RefreshCw className="size-3" />
                                  )}
                                  {agentTypeSwitching ? "切换中" : "切换"}
                                </button>
                              )}
                            </span>
                          </>
                        )}
                        <span style={{ color: "var(--border)" }}>｜</span>
                        <span className="shrink-0 whitespace-nowrap">
                          ID：{detailAgent?.instanceId || "ins-grpdemo02"}
                          <span className="mx-1" style={{ color: "var(--border)" }}>｜</span>
                          版本：
                          <span className="font-mono tabular-nums" style={{ color: "var(--text-muted)" }}>
                            {agentVersionLabel}
                          </span>
                        </span>
                        {/* 管理员推送的「可更新」徽章 */}
                        {!isLocalAgent && recommendPush && (
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span
                                className="inline-flex items-center gap-1 cursor-help whitespace-nowrap ml-1"
                                style={{
                                  padding: "2px 6px",
                                  borderRadius: "2px",
                                  fontSize: "11px",
                                  lineHeight: "16px",
                                  fontWeight: 500,
                                  color: "var(--text-brand)",
                                  backgroundColor: "var(--alert-info-bg)",
                                  border: "1px solid var(--alert-info-border)",
                                }}
                              >
                                <Megaphone className="w-3 h-3" />
                                可更新 v{recommendPush.version}
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-[320px] text-xs leading-relaxed">
                              <div className="space-y-1">
                                <div className="font-medium">
                                  {recommendPush.message ?? `推荐更新到 v${recommendPush.version}`}
                                </div>
                                <div className="text-[var(--text-weak)]">
                                  来自 {recommendPush.pushedBy} · {recommendPush.pushedAt}
                                </div>
                              </div>
                            </TooltipContent>
                          </Tooltip>
                        )}
                      </div>
                      {/* 元信息第二行：组织 / 项目（固定在类型行下一行，不随宽度变化跳跃） */}
                      {!isLocalAgent && (
                        <div className="flex items-center flex-wrap gap-x-1 gap-y-1 min-w-0">
                          <span className="whitespace-nowrap">组织：{detailAgent?.groupName || "默认"}</span>
                          <span style={{ color: "var(--border)" }}>｜</span>
                          <span className="inline-flex items-center gap-1 whitespace-nowrap">
                            项目：{projectNames.length > 0 ? projectNames.join("、") : "未关联"}
                            <button
                              type="button"
                              onClick={openProjectEdit}
                              className="inline-flex items-center justify-center w-4 h-4 rounded transition-colors hover:bg-[var(--bg-grey-hover)]"
                              style={{ color: "var(--text-muted)" }}
                              aria-label="编辑关联项目"
                              title="编辑关联项目"
                            >
                              <Edit3 className="w-3 h-3" />
                            </button>
                          </span>
                        </div>
                      )}
                    </div>
                  </div>
                </div>

                {/* 右：操作按钮 */}
                {!isLocalAgent && (
                <div className="flex items-center gap-2 shrink-0">
                  {isUpdating ? (
                    <Button
                      variant="tenant-outline"
                      size="claw"
                      title="查看更新进度"
                      onClick={() => setShowUpdateProgressDialog(true)}
                    >
                      <Loader2 className="w-3.5 h-3.5 animate-spin text-[var(--text-brand)]" />
                      更新中
                    </Button>
                  ) : allowSelfUpgrade ? (
                    <Button
                      variant="tenant-outline"
                      size="claw"
                      onClick={() => setShowUpdateConfirmDialog(true)}
                    >
                      一键更新
                    </Button>
                  ) : (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        {/* 用 span 包裹 disabled Button，确保 Tooltip 在禁用态仍可触发 */}
                        <span tabIndex={0}>
                          <Button
                            variant="tenant-outline"
                            size="claw"
                            disabled
                            className="cursor-not-allowed opacity-60"
                          >
                            一键更新
                          </Button>
                        </span>
                      </TooltipTrigger>
                      <TooltipContent side="bottom" className="text-xs max-w-[240px] leading-relaxed">
                        管理员已关闭"自助更新"，请联系管理员开启
                      </TooltipContent>
                    </Tooltip>
                  )}
                  {/* 数据备份按钮：none 不渲染；creating 置灰；ready 可点 */}
                  {backupPointStatus !== "none" && (
                    backupPointStatus === "creating" ? (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span tabIndex={0}>
                            <Button
                              variant="tenant-outline"
                              size="claw"
                              disabled
                              className="cursor-not-allowed opacity-60"
                              data-testid="open-data-backup"
                            >
                              <HardDriveDownload className="w-3.5 h-3.5" />
                              数据备份
                            </Button>
                          </span>
                        </TooltipTrigger>
                        <TooltipContent side="bottom" className="text-xs max-w-[240px] leading-relaxed">
                          备份生成中，请稍候
                        </TooltipContent>
                      </Tooltip>
                    ) : (
                      <Button
                        variant="tenant-outline"
                        size="claw"
                        onClick={openDataBackup}
                        data-testid="open-data-backup"
                      >
                        <HardDriveDownload className="w-3.5 h-3.5" />
                        数据备份
                      </Button>
                    )
                  )}
                  <Button
                    variant="tenant-outline"
                    size="claw"
                    onClick={handleOpenAgentPanel}
                  >
                    开启Agent面板
                  </Button>
                  <Button
                    variant="tenant-outline"
                    size="claw"
                    onClick={openInstanceMigration}
                    data-testid="start-instance-migration"
                  >
                    <ArrowLeftRight className="w-3.5 h-3.5" />
                    Agent 迁移
                  </Button>
                  <Button
                    variant="tenant-primary"
                    size="claw"
                    onClick={() => {
                      localStorage.setItem("openclaw_view_mode", "chat");
                      navigate("/my-openclaw");
                    }}
                  >
                    开始对话
                    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <path
                        fillRule="evenodd"
                        clipRule="evenodd"
                        d="M8 1.5C11.5898 1.5 14.5 4.41015 14.5 8C14.5 11.5898 11.5898 14.5 8 14.5H2.15039L3.2793 12.4678C2.17661 11.303 1.5 9.73056 1.5 8C1.5 4.41015 4.41015 1.5 8 1.5ZM8.41602 6.16699C8.0018 6.16699 7.66602 6.50278 7.66602 6.91699V7.41699C7.66615 7.83109 8.00188 8.16699 8.41602 8.16699C8.83015 8.16699 9.16588 7.83109 9.16602 7.41699V6.91699C9.16602 6.50278 8.83023 6.16699 8.41602 6.16699ZM11.416 6.16699C11.0018 6.16699 10.666 6.50278 10.666 6.91699V7.41699C10.6661 7.83109 11.0019 8.16699 11.416 8.16699C11.8301 8.16699 12.1659 7.83109 12.166 7.41699V6.91699C12.166 6.50278 11.8302 6.16699 11.416 6.16699Z"
                        fill="white"
                      />
                    </svg>
                  </Button>
                </div>
                )}
                </div>
              </header>

              {/* ===== 方案 C：header 下方独立「角色栏」（仅多角色实例 + 选中方案 C 时显示） ===== */}
              {!isLocalAgent && isMultiRole && rolePresentation === "C" && (
                <div
                  className="flex items-center gap-3 mt-3 px-4 py-3 rounded-[8px] flex-wrap"
                  style={{ background: "var(--alert-info-bg)", border: "1px solid var(--alert-info-border)" }}
                >
                  <span className="inline-flex items-center gap-1.5 text-sm font-semibold text-[var(--text-strong)] flex-shrink-0">
                    <Users className="w-4 h-4 text-[var(--text-brand)]" />
                    角色（{roleCount}）
                  </span>
                  <div className="w-px h-5 bg-[var(--border)] flex-shrink-0" />
                  <div className="flex items-center flex-wrap gap-2 flex-1 min-w-0">
                    {groupedRoles.map((g) => {
                      const slots = agentRoles
                        .filter((r) => g.slotIds.includes(r.id))
                        .map((r) => ({ id: r.id, name: r.name, isMain: (r as any).isMain }));
                      return (
                        <Popover key={g.roleName + g.slotIds[0]}>
                          <PopoverTrigger asChild>
                            <button
                              type="button"
                              className="inline-flex items-center gap-1.5 text-xs font-medium px-3 py-1.5 rounded-[6px] flex-shrink-0 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ring)]"
                              style={
                                g.isMain
                                  ? { border: "1px solid var(--alert-info-border)", background: "var(--card)", color: "var(--text-brand)" }
                                  : { border: "1px solid var(--border)", background: "var(--card)", color: "var(--text-secondary)" }
                              }
                              aria-label={`角色 ${g.roleName}${g.count > 1 ? ` ×${g.count}` : ""}，点击切换`}
                            >
                              {g.isMain && <Crown className="w-3.5 h-3.5 flex-shrink-0" />}
                              {g.isMain && <span style={{ opacity: 0.8 }}>主角色 ·</span>}
                              <span>{g.roleName}</span>
                              {g.count > 1 && <span style={{ opacity: 0.7 }}>×{g.count}</span>}
                              <ChevronDown className="w-3 h-3" />
                            </button>
                          </PopoverTrigger>
                          <PopoverContent align="start" className="p-0 w-auto">
                            {renderRoleSwitchPanel(slots)}
                          </PopoverContent>
                        </Popover>
                      );
                    })}
                  </div>
                  <Button
                    variant="tenant-outline"
                    size="claw-sm"
                    className="flex-shrink-0"
                    onClick={() => setActiveTab("basic")}
                  >
                    <Settings2 className="w-3.5 h-3.5" />
                    角色管理
                  </Button>
                </div>
              )}

              {/* ======== 横向 Segment Tab（灰底容器 + 白色选中滑块）======== */}
              {/* 仅 1 个 Tab 时直接隐藏整条容器：本地客户端 Agent（只剩"基础配置"）就不再展示孤立胶囊 */}
              {visibleTabs.length > 1 && (
                <div className="relative py-4 flex items-center justify-start gap-3">
                  <TenantSegmentGroup aria-label="详情页 Tab 切换">
                    {visibleTabs.map((t) => (
                      <TenantSegmentOption
                        key={t.id}
                        active={t.id === activeTab}
                        onClick={() => setActiveTab(t.id)}
                      >
                        {t.label}
                      </TenantSegmentOption>
                    ))}
                  </TenantSegmentGroup>
                  {/* 角色标签：基础配置tab=可选下拉；其它tab=头像堆叠（hover展示全部角色） */}
                  {!isLocalAgent && isMultiRole && (
                    activeTab === "basic" ? (
                      /* ===== 基础配置：可选角色下拉 ===== */
                      <Select
                        value={activeRoleId || mainRole?.id || agentRoles[0]?.id || ""}
                        onValueChange={(val) => {
                          const target = agentRoles.find((r) => r.id === val);
                          if (target) {
                            handleQuickSwitchRole(target.id, target.name);
                            handleSwitchRole(val);
                          }
                        }}
                      >
   <SelectTrigger tenant className="w-auto flex-shrink-0" data-testid="role-switch-dropdown">
  <span className="text-[var(--text-weak)] mr-1">当前配置角色：</span>
     {(() => {
const cur = agentRoles.find((r) => r.id === (activeRoleId || agentRoles[0]?.id));
   // 主角色名称与 Agent 名称（自定义、与卡片一致）保持一致；子角色显示各自的自定义角色名称
    const curLabel = cur ? (cur.isMain ? agentName : cur.name) : "请选择";
    return (
   <span className="inline-flex items-center gap-2">
       <AgentAvatar
        roleName={cur?.baseRoleName ?? cur?.name ?? ""}
  agentName={agentName}
         size={20}
     className="flex-shrink-0"
 />
      <span className="leading-5">{curLabel}</span>
        </span>
     );
      })()}
         </SelectTrigger>
     <SelectContent align="start" className="rounded-[12px]">
        {agentRoles.map((r) => {
  // 主角色名称与 Agent 名称（自定义、与卡片一致）保持一致；子角色显示各自的自定义角色名称
        const label = r.isMain ? agentName : r.name;
    // 名称旁统一展示「角色类型」（baseRoleName），而非主/子角色
        const roleType = r.baseRoleName ?? r.name;
       return (
  <SelectItem key={r.id} value={r.id} textValue={label}>
      <span className="inline-flex items-center gap-2 align-middle">
     <AgentAvatar
    roleName={roleType}
     agentName={agentName}
    size={20}
       className="flex-shrink-0"
     />
<span className="leading-5">{label}</span>
   <span className="text-[var(--text-weak)] text-xs ml-1">
      {roleType}
      </span>
   </span>
      </SelectItem>
 );
    })}
       </SelectContent>
                      </Select>
                    ) : (
                      /* ===== 非基础配置tab：纯文本展示，不再显示头像堆叠与 hover 气泡 ===== */
                      <div className="inline-flex items-center gap-1.5 flex-shrink-0 h-9 px-3 rounded-full border border-border bg-white cursor-default">
                        <span className="text-sm text-[var(--text-weak)]">当前配置角色：</span>
                        <span className="text-sm text-[var(--text-emphasis)]">全部角色</span>
                      </div>
                    )
                  )}
                </div>
              )}

              {/* ======== 三栏卡片 ======== */}
              {/* Tab 隐藏时（本地客户端）补一个上间距，避免与 header 贴太近 */}
              <div className={visibleTabs.length > 1 ? "py-0 flex-1" : "pt-4 flex-1"}>
              {activeTab === "basic" && (
                <>
                <div className={isLocalAgent ? "w-full" : "grid grid-cols-3 gap-6"}>
                  {!isLocalAgent && (
                    <>
                  {/* ===== 01/ 模型（Models） ===== */}
                  <TenantCard padding="none" className="flex flex-col p-6 gap-3">
                    {/* 标题区 */}
                    <div
                      className="flex flex-col gap-1 pb-5"
                      style={{ borderBottom: "1px solid var(--border)" }}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-end gap-2">
                          <span
                            className="text-[18px] leading-6"
                            style={{ fontFamily: "Menlo, Consolas, 'Courier New', monospace", color: "var(--text-brand)" }}
                          >
                            01/
                          </span>
                          <span className="text-[18px] font-medium leading-6" style={{ color: "var(--text-emphasis)" }}>
                            模型
                          </span>
                          <span className="text-[15px] leading-6" style={{ color: "var(--text-weak)" }}>
                            Models
                          </span>
                        </div>
                        {appliedModels.filter(m => m.primary).length > 0 ? (
                          <ConfiguredBadge />
                        ) : (
                          <span
                            className="inline-flex items-center gap-1.5 h-5 px-2 text-xs shrink-0"
                            style={{ color: "var(--text-weak)", letterSpacing: "0.015em" }}
                          >
                            <span className="w-2 h-2 rounded-full" style={{ background: "var(--text-weak)" }} />
                            未配置
                          </span>
                        )}
                      </div>
                      <p className="text-sm leading-[18px] min-h-[36px]" style={{ color: "var(--muted-foreground)" }}>
                        Agent 的"大脑"，决定 Agent 的智能水平和能力范围
                      </p>
                    </div>

                    {/* 新增模型按钮：方案一改为常驻按钮，表单不再内嵌在窄栏里，改用居中 Dialog 承载 */}
                    <Button
                      variant="tenant-outline-strong"
                      size="lg"
                      className="w-full"
                      onClick={() => { setFormErrors({}); setEditingModelId(null); setShowAddBackupModel(true); }}
                    >
                      <Plus className="w-3.5 h-3.5" />
                      新增模型
                    </Button>
                    {/* 新增模型 Dialog（方案一：脱离 grid-cols-3 窄栏限制，宽度升级到 720px，缓解字段拥挤）；
                        编辑主模型改为复用同一个 Dialog（editingModelId 非空时为编辑态），交互域与"新增"保持一致 */}
                    <Dialog
                      open={showAddBackupModel}
                      onOpenChange={(open) => { setShowAddBackupModel(open); if (!open) setEditingModelId(null); }}
                    >
                      <DialogContent size="lg" className="flex flex-col max-h-[90vh]">
                        <DialogHeader>
                          <DialogTitle>{editingModelId !== null ? "编辑模型" : "添加模型"}</DialogTitle>
                          <DialogDescription className="sr-only">
                            选择模型来源、厂商和版本，配置 API Key 与限额后添加主模型或备用模型。
                          </DialogDescription>
                        </DialogHeader>
                        <DialogBody className="px-6">
                        <div className="space-y-3 pb-1">
                        {/* 模型来源（方案一：管理员预置 / Coding Plan / 模型 API / 自定义模型 合并为一层分组选择，减少一次决策） */}
                        <div className="space-y-1.5">
                          <label className="text-xs text-[var(--text-secondary)]">模型来源</label>
                          <Select value={modelSourceGroupValue} onValueChange={handleSourceGroupChange}>
                            <SelectTrigger className="w-full bg-white border-[var(--border)]">
                              <SelectValue placeholder="选择模型来源" />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="admin">{MODEL_PROVIDER_GROUP_LABELS.admin}</SelectItem>
                              <SelectGroup>
                                <SelectLabel>{MODEL_PROVIDER_GROUP_LABELS.self}</SelectLabel>
                                {SELF_CONFIG_CATEGORY_ORDER.map((cat) => (
                                  <SelectItem key={cat} value={`self:${cat}`}>
                                    {SELF_CONFIG_CATEGORY_LABELS[cat]}
                                  </SelectItem>
                                ))}
                              </SelectGroup>
                            </SelectContent>
                          </Select>
                        </div>
                        {/* 厂商选择：admin 来源 或 self 来源的 Coding Plan / 模型 API 类别时展示 */}
                        {!isCustomProvider && (
                          <div className="space-y-1.5">
                            <label className="text-xs text-[var(--text-secondary)]">
                              模型厂商
                            </label>
                            <Select
                              value={selectedProvider}
                              onValueChange={(v) => {
                                setSelectedProvider(v);
                                const provider = MODEL_PROVIDERS.find((pr) => pr.value === v);
                                if (provider && provider.versions.length > 0) {
                                  setSelectedModel(provider.versions[0].value);
                                }
                              }}
                            >
                              <SelectTrigger className="w-full bg-white border-[var(--border)]">
                                <SelectValue placeholder="选择厂商" />
                              </SelectTrigger>
                              <SelectContent>
                                {sourceProviders.map((p) => (
                                  <SelectItem key={p.value} value={p.value}>
                                    {p.label}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                          </div>
                        )}
                        {/* 版本选择（方案一：仅当有多个版本可选时才展示下拉，否则直接展示只读文本，去掉"只有一个答案的选择题"） */}
                        {!isCustomProvider && (
                          <div className="space-y-1.5">
                            <label className="text-xs text-[var(--text-secondary)]">
                              模型版本
                            </label>
                            {currentVersions.length > 1 ? (
                              <Select value={selectedModel} onValueChange={setSelectedModel}>
                                <SelectTrigger className="w-full bg-white border-[var(--border)]">
                                  <SelectValue placeholder="选择版本" />
                                </SelectTrigger>
                                <SelectContent>
                                  {currentVersions.map((v) => (
                                    <SelectItem key={v.value} value={v.value}>
                                      {v.label}
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            ) : (
                              <div className="h-9 flex items-center px-3 text-sm rounded-[var(--radius-card)] bg-[var(--bg-grey-normal)] border border-[var(--border)] text-[var(--text-secondary)]">
                                {currentVersions[0]?.label ?? "—"}
                              </div>
                            )}
                          </div>
                        )}
                        {/* ===== 管理员预置「由用户端自行配置」模型：仅需用户端填写 API Key（支持连通性检测）===== */}
                        {isAdminUserKeyProvider && (
                          <div className="space-y-1.5 pt-1">
                            <label className="text-xs text-[var(--text-secondary)]">
                              API Key<span className="text-[var(--text-danger)] ml-0.5">*</span>
                            </label>
                            <Input
                              tenant
                              type="password"
                              placeholder="请输入 API Key"
                              value={publicForm.apiKey}
                              onChange={(e) => { setPublicForm({ ...publicForm, apiKey: e.target.value }); clearFieldError("apiKey"); }}
                              aria-invalid={!!formErrors.apiKey}
                              className={`bg-white text-sm rounded-[4px] ${formErrors.apiKey ? "border-[var(--text-danger)] focus:border-[var(--text-danger)]" : "border-[var(--border)]"}`}
                            />
                            {formErrors.apiKey
                              ? <p className="text-xs text-[var(--text-danger)]">请填写 API Key</p>
                              : <p className="text-xs text-[var(--text-tertiary)]">该模型由管理员预置，需填写您自己的 API Key 后方可使用。</p>}
                          </div>
                        )}
                        {/* ===== 自行配置·公开模型 表单（API Key / 限额 / 高级配置）===== */}
                        {isPublicSelfProvider && (
                          <div className="space-y-3 pt-1">
                            {/* API Key */}
                            <div className="space-y-1.5">
                              <label className="text-xs text-[var(--text-secondary)]">
                                API Key<span className="text-[var(--text-danger)] ml-0.5">*</span>
                              </label>
                              <Input
                                tenant
                                type="password"
                                placeholder="请输入 API Key"
                                value={publicForm.apiKey}
                                onChange={(e) => { setPublicForm({ ...publicForm, apiKey: e.target.value }); clearFieldError("apiKey"); }}
                                aria-invalid={!!formErrors.apiKey}
                                className={`bg-white text-sm rounded-[4px] ${formErrors.apiKey ? "border-[var(--text-danger)] focus:border-[var(--text-danger)]" : "border-[var(--border)]"}`}
                              />
                              {formErrors.apiKey && <p className="text-xs text-[var(--text-danger)]">请填写 API Key</p>}
                            </div>
                            {/* 每日 Tokens 上限 */}
                            <div className="space-y-1.5">
                              <label className="text-xs text-[var(--text-secondary)]">
                                每日 Tokens 数量上限<span className="text-[var(--text-danger)] ml-0.5">*</span>
                              </label>
                              <Input
                                tenant
                                type="number"
                                placeholder="请输入每日 Tokens 上限"
                                value={publicForm.dailyLimit}
                                onChange={(e) => { setPublicForm({ ...publicForm, dailyLimit: e.target.value }); clearFieldError("dailyLimit"); }}
                                aria-invalid={!!formErrors.dailyLimit}
                                className={`bg-white text-sm rounded-[4px] ${formErrors.dailyLimit ? "border-[var(--text-danger)] focus:border-[var(--text-danger)]" : "border-[var(--border)]"}`}
                              />
                              {formErrors.dailyLimit && <p className="text-xs text-[var(--text-danger)]">请填写有效的每日 Tokens 上限</p>}
                            </div>
                            {/* 高级配置：去掉外层卡片框（Dialog 内本来就是白底，多一层边框是不必要的套娃），
                                改为与其它字段同层级的轻量展开行 */}
                            <div>
                              <button
                                type="button"
                                onClick={() => setAdvancedOpen(v => !v)}
                                className="w-full flex items-center justify-between py-2"
                              >
                                <span className="flex items-center gap-1.5">
                                  <span className="text-sm font-medium text-[var(--text-emphasis)]">高级配置</span>
                                  <span className="text-xs text-[var(--text-weak)]">最大输出长度/请求头</span>
                                </span>
                                {advancedOpen ? <ChevronDown className="w-3.5 h-3.5 text-[var(--text-weak)]" /> : <ChevronRight className="w-3.5 h-3.5 text-[var(--text-weak)]" />}
                              </button>
                              {advancedOpen && (
                                <div className="space-y-3 pt-1">
                                  {/* 最大输出长度 */}
                                  <div className="space-y-1.5">
                                    <div className="flex items-center gap-1">
                                      <span className="text-xs text-[var(--text-secondary)]">最大输出长度</span>
                                      <Tooltip>
                                        <TooltipTrigger asChild>
                                          <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                                        </TooltipTrigger>
                                        <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                          maxTokens，即模型单次回复时最多输出的 Token 数。
                                        </TooltipContent>
                                      </Tooltip>
                                    </div>
                                    <Input
                                      tenant
                                      type="number"
                                      placeholder="请输入最大输出长度 maxTokens"
                                      value={advancedConfig.maxTokens}
                                      onChange={(e) => setAdvancedConfig({ ...advancedConfig, maxTokens: e.target.value })}
                                      className="bg-white border-[var(--border)] text-sm rounded-[4px]"
                                    />
                                  </div>
                                  {/* 请求头 */}
                                  <div className="space-y-1.5">
                                    <div className="flex items-center gap-1">
                                      <span className="text-xs text-[var(--text-secondary)]">请求头</span>
                                      <Tooltip>
                                        <TooltipTrigger asChild>
                                          <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                                        </TooltipTrigger>
                                        <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                          headers，在 HTTP 请求中用于传递认证信息或数据格式等元数据的参数。
                                        </TooltipContent>
                                      </Tooltip>
                                    </div>
                                    <div className="space-y-2">
                                      {advancedConfig.headers.map((entry, idx) => (
                                        <div key={idx} className="flex items-center gap-2">
                                          <Input
                                            tenant
                                            placeholder="key"
                                            value={entry.key}
                                            onChange={(e) => {
                                              const next = advancedConfig.headers.map((h, i) => i === idx ? { ...h, key: e.target.value } : h);
                                              setAdvancedConfig({ ...advancedConfig, headers: next });
                                            }}
                                            className="bg-white border-[var(--border)] text-sm w-[36%] shrink-0 rounded-[4px]"
                                          />
                                          <Input
                                            tenant
                                            placeholder="value"
                                            value={entry.value}
                                            onChange={(e) => {
                                              const next = advancedConfig.headers.map((h, i) => i === idx ? { ...h, value: e.target.value } : h);
                                              setAdvancedConfig({ ...advancedConfig, headers: next });
                                            }}
                                            className="bg-white border-[var(--border)] text-sm flex-1 rounded-[4px]"
                                          />
                                          <button
                                            type="button"
                                            onClick={() => setAdvancedConfig({ ...advancedConfig, headers: advancedConfig.headers.filter((_, i) => i !== idx) })}
                                            className="shrink-0 text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                                            aria-label="删除请求头"
                                          >
                                            <Plus className="w-3.5 h-3.5 rotate-45" />
                                          </button>
                                        </div>
                                      ))}
                                      <button
                                        type="button"
                                        onClick={() => setAdvancedConfig({ ...advancedConfig, headers: [...advancedConfig.headers, { key: "", value: "" }] })}
                                        className="flex items-center gap-1 text-xs text-[var(--text-brand)] hover:text-[var(--text-brand)] transition-colors mt-1"
                                      >
                                        <Plus className="w-3 h-3" />
                                        添加请求头
                                      </button>
                                    </div>
                                  </div>
                                </div>
                              )}
                            </div>
                          </div>
                        )}
                        {/* ===== 自定义模型 表单（厂商为 custom 时展示）===== */}
                        {isCustomProvider && (
                          <div className="space-y-3 pt-1">
                            {/* JSON / 表单 模式切换 */}
                            <TenantSegmentGroup className="w-full">
                              <TenantSegmentOption
                                className="flex-1"
                                active={customInputMode === "json"}
                                onClick={() => { setCustomInputMode("json"); setFormErrors({}); }}
                              >
                                JSON 输入
                              </TenantSegmentOption>
                              <TenantSegmentOption
                                className="flex-1"
                                active={customInputMode === "form"}
                                onClick={() => { setCustomInputMode("form"); setFormErrors({}); }}
                              >
                                表单输入
                              </TenantSegmentOption>
                            </TenantSegmentGroup>

                            {customInputMode === "json" ? (
                              <div className="space-y-1.5">
                                <Textarea
                                  value={customJson}
                                  onChange={(e) => { setCustomJson(e.target.value); clearFieldError("json"); }}
                                  aria-invalid={!!formErrors.json}
                                  className={`font-mono text-xs bg-white min-h-[180px] resize-none ${formErrors.json ? "border-[var(--text-danger)] focus:border-[var(--text-danger)]" : "border-[var(--border)]"}`}
                                  spellCheck={false}
                                />
                                {formErrors.json && <p className="text-xs text-[var(--text-danger)]">请填写完整的模型配置信息</p>}
                              </div>
                            ) : (
                              <div className="space-y-3">
                                {/* 修复：原来 6 个字段全靠 placeholder 顶替 label，填完内容后 placeholder 消失，
                                    字段身份完全丢失（截图里 3 个"111"根本分不出哪个是哪个）——补上持久展示的 <label>，
                                    placeholder 只保留简短提示 */}
                                {[
                                  { key: "provider", label: "Provider", placeholder: "如 openai / anthropic", required: false },
                                  { key: "base_url", label: "Base URL", placeholder: "如 https://api.example.com/v1", required: true },
                                  { key: "api", label: "API", placeholder: "如 chat/completions", required: false },
                                  { key: "api_key", label: "API Key", placeholder: "请输入 API Key", required: true },
                                  { key: "model_id", label: "Model ID", placeholder: "如 gpt-4o", required: true },
                                  { key: "model_name", label: "Model Name", placeholder: "模型展示名称", required: false },
                                ].map((field) => (
                                  <div key={field.key} className="space-y-1.5">
                                    <label className="text-xs text-[var(--text-secondary)]">
                                      {field.label}
                                      {field.required && <span className="text-[var(--text-danger)] ml-0.5">*</span>}
                                    </label>
                                    <Input
                                      tenant
                                      placeholder={field.placeholder}
                                      value={customForm[field.key as keyof typeof customForm]}
                                      onChange={(e) => { setCustomForm({ ...customForm, [field.key]: e.target.value }); clearFieldError(field.key); }}
                                      aria-invalid={!!formErrors[field.key]}
                                      className={`bg-white text-sm rounded-[4px] ${formErrors[field.key] ? "border-[var(--text-danger)] focus:border-[var(--text-danger)]" : "border-[var(--border)]"}`}
                                    />
                                    {formErrors[field.key] && <p className="text-xs text-[var(--text-danger)]">请填写{field.label}</p>}
                                  </div>
                                ))}
                              </div>
                            )}

                            {/* 高级配置：仅表单模式（去掉外层卡片框，与其它字段同层级） */}
                            {customInputMode === "form" && (
                              <div>
                                <button
                                  type="button"
                                  onClick={() => setAdvancedOpen(v => !v)}
                                  className="w-full flex items-center justify-between py-2"
                                >
                                  <span className="flex items-center gap-1.5">
                                    <span className="text-sm font-medium text-[var(--text-emphasis)]">高级配置</span>
                                    <span className="text-xs text-[var(--text-weak)]">上下文长度/最大输出长度/请求头</span>
                                  </span>
                                  {advancedOpen ? <ChevronDown className="w-3.5 h-3.5 text-[var(--text-weak)]" /> : <ChevronRight className="w-3.5 h-3.5 text-[var(--text-weak)]" />}
                                </button>
                                {advancedOpen && (
                                  <div className="space-y-3 pt-1">
                                    {/* 上下文长度 */}
                                    <div className="space-y-1.5">
                                      <div className="flex items-center gap-1">
                                        <span className="text-xs text-[var(--text-secondary)]">上下文长度</span>
                                        <Tooltip>
                                          <TooltipTrigger asChild>
                                            <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                                          </TooltipTrigger>
                                          <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                            contextWindow，指的是模型总上下文窗口大小。
                                          </TooltipContent>
                                        </Tooltip>
                                      </div>
                                      <Input
                                        tenant
                                        type="number"
                                        placeholder="请输入上下文长度 contextWindow"
                                        value={advancedConfig.contextWindow}
                                        onChange={(e) => setAdvancedConfig({ ...advancedConfig, contextWindow: e.target.value })}
                                        className="bg-white border-[var(--border)] text-sm rounded-[4px]"
                                      />
                                    </div>
                                    {/* 最大输出长度 */}
                                    <div className="space-y-1.5">
                                      <div className="flex items-center gap-1">
                                        <span className="text-xs text-[var(--text-secondary)]">最大输出长度</span>
                                        <Tooltip>
                                          <TooltipTrigger asChild>
                                            <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                                          </TooltipTrigger>
                                          <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                            maxTokens，即模型单次回复时最多输出的 Token 数。
                                          </TooltipContent>
                                        </Tooltip>
                                      </div>
                                      <Input
                                        tenant
                                        type="number"
                                        placeholder="请输入最大输出长度 maxTokens"
                                        value={advancedConfig.maxTokens}
                                        onChange={(e) => setAdvancedConfig({ ...advancedConfig, maxTokens: e.target.value })}
                                        className="bg-white border-[var(--border)] text-sm rounded-[4px]"
                                      />
                                    </div>
                                    {/* 请求头 */}
                                    <div className="space-y-1.5">
                                      <div className="flex items-center gap-1">
                                        <span className="text-xs text-[var(--text-secondary)]">请求头</span>
                                        <Tooltip>
                                          <TooltipTrigger asChild>
                                            <span className="cursor-default"><Info className="w-3 h-3 text-[var(--text-muted)]" /></span>
                                          </TooltipTrigger>
                                          <TooltipContent className="max-w-[240px] text-xs leading-relaxed">
                                            headers，在 HTTP 请求中用于传递认证信息或数据格式等元数据的参数。
                                          </TooltipContent>
                                        </Tooltip>
                                      </div>
                                      <div className="space-y-2">
                                        {advancedConfig.headers.map((entry, idx) => (
                                          <div key={idx} className="flex items-center gap-2">
                                            <Input
                                              tenant
                                              placeholder="key"
                                              value={entry.key}
                                              onChange={(e) => {
                                                const next = advancedConfig.headers.map((h, i) => i === idx ? { ...h, key: e.target.value } : h);
                                                setAdvancedConfig({ ...advancedConfig, headers: next });
                                              }}
                                              className="bg-white border-[var(--border)] text-sm w-[36%] shrink-0 rounded-[4px]"
                                            />
                                            <Input
                                              tenant
                                              placeholder="value"
                                              value={entry.value}
                                              onChange={(e) => {
                                                const next = advancedConfig.headers.map((h, i) => i === idx ? { ...h, value: e.target.value } : h);
                                                setAdvancedConfig({ ...advancedConfig, headers: next });
                                              }}
                                              className="bg-white border-[var(--border)] text-sm flex-1 rounded-[4px]"
                                            />
                                            <button
                                              type="button"
                                              onClick={() => setAdvancedConfig({ ...advancedConfig, headers: advancedConfig.headers.filter((_, i) => i !== idx) })}
                                              className="shrink-0 text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                                              aria-label="删除请求头"
                                            >
                                              <Plus className="w-3.5 h-3.5 rotate-45" />
                                            </button>
                                          </div>
                                        ))}
                                        <button
                                          type="button"
                                          onClick={() => setAdvancedConfig({ ...advancedConfig, headers: [...advancedConfig.headers, { key: "", value: "" }] })}
                                          className="flex items-center gap-1 text-xs text-[var(--text-brand)] hover:text-[var(--text-brand)] transition-colors mt-1"
                                        >
                                          <Plus className="w-3 h-3" />
                                          添加请求头
                                        </button>
                                      </div>
                                    </div>
                                  </div>
                                )}
                              </div>
                            )}

                            {/* 多模态模型 开关 */}
                            <div className="flex items-center justify-between rounded-[var(--radius-card)] bg-white border border-[var(--border)] px-3 py-2.5">
                              <div className="flex flex-col">
                                <span className="text-sm font-medium text-[var(--text-emphasis)]">多模态模型</span>
                                <span className="text-xs text-[var(--text-muted)] mt-0.5">支持图片、文字多模态输入</span>
                              </div>
                              <button
                                type="button"
                                role="switch"
                                aria-checked={customMultimodal}
                                onClick={() => setCustomMultimodal(v => !v)}
                                className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 ${
                                  customMultimodal ? "bg-[var(--text-brand)]" : "bg-[var(--muted)]"
                                }`}
                              >
                                <span
                                  className={`pointer-events-none block h-4 w-4 rounded-full bg-white shadow-sm ring-0 transition-transform ${
                                    customMultimodal ? "translate-x-4" : "translate-x-0"
                                  }`}
                                />
                              </button>
                            </div>

                            {/* 黄色提示卡 + 配置指引链接 */}
                            <Alert variant="warning">
                              <AlertCircle className="w-4 h-4" />
                              <AlertDescription>
                                使用自定义模型需自行承担 Tokens 费用，不计入公司提供的大模型 Tokens 范围。
                                <a href="#" className="inline-flex items-center gap-0.5 text-[var(--text-brand)] hover:opacity-80 underline underline-offset-2 ml-1 transition-colors">
                                  自定义模型配置指引 <ExternalLink className="w-3 h-3" />
                                </a>
                              </AlertDescription>
                            </Alert>
                          </div>
                        )}
                        </div>
                        </DialogBody>
                        <DialogFooter>
                        {/* 添加模型按钮 / 自定义模型 & 公开模型改为「连通性检测（次） + 添加（主）」双按钮：
                            按钮尺寸随内容自适应、Footer 默认右对齐（不再用 flex-1/w-full 拉满整行），
                            并按 SKILL-GLOBAL-COMPONENTS.md §Dialog 规范区分主/次按钮：主按钮 tenant-primary（黑底胶囊），次按钮 tenant-outline（白底描边胶囊） */}
                        {(isCustomProvider || isPublicSelfProvider || isAdminUserKeyProvider) ? (
                          <>
                            <Button
                              variant="tenant-outline"
                              size="lg"
                              onClick={handleConnectTest}
                              disabled={connectTesting}
                            >
                              {connectTesting && <Loader2 className="w-4 h-4 animate-spin mr-1" />}
                              {connectTesting ? "检测中…" : "连通性检测"}
                            </Button>
                            <Button
                              variant="tenant-primary"
                              size="lg"
                              onClick={() => {
                                if (editingModelId !== null) { handleSaveEditModel(); }
                                else if (handleApplyModel()) { setShowAddBackupModel(false); }
                              }}
                            >
                              {editingModelId !== null ? "保存修改" : (appliedModels.filter(m => m.primary).length > 0 ? "添加备用模型" : "添加主模型")}
                            </Button>
                          </>
                        ) : (
                          <Button
                            variant="tenant-primary"
                            size="lg"
                            onClick={() => {
                              if (editingModelId !== null) { handleSaveEditModel(); }
                              else if (handleApplyModel()) { setShowAddBackupModel(false); }
                            }}
                          >
                            {editingModelId !== null ? "保存修改" : (appliedModels.filter(m => m.primary).length > 0 ? "添加备用模型" : "添加主模型")}
                          </Button>
                        )}
                        </DialogFooter>
                      </DialogContent>
                    </Dialog>

                    {/* 已应用模型 */}
                    <div className="pt-2">
                      {/* 主模型 */}
                      <div className="mb-3">
                        <div className="text-sm font-medium mb-2" style={{ color: "var(--text-body)" }}>
                          主模型
                        </div>
                        {appliedModels.filter(m => m.primary).length > 0 ? (
                          appliedModels.filter(m => m.primary).map((model) => (
                            <div key={model.id} className="mb-2 last:mb-0">
                              {/* 编辑改为与"添加主模型"一致的交互：点击编辑打开同一个 Dialog（预填当前值），
                                  不再用"就地编辑"内嵌小卡片替换整块内容——两者交互域要保持统一 */}
                              <div
                                className="rounded-[var(--radius-card)] p-3 flex items-center justify-between gap-2"
                                style={{ border: "1px solid var(--border)" }}
                              >
                                <div className="flex flex-col gap-0.5 min-w-0 flex-1">
                                  <div className="flex items-center gap-2">
                                    <span className="text-sm font-medium truncate" style={{ color: "var(--foreground)" }}>
                                      {model.providerLabel}
                                    </span>
                                    {model.adminPreset && (
                                      <span
                                        className="inline-flex shrink-0 items-center px-2 py-0.5 text-xs rounded-[3px]"
                                        style={{ border: "1px solid var(--border)", color: "var(--muted-foreground)" }}
                                      >
                                        管理员预置
                                      </span>
                                    )}
                                    {model.isCustom && model.multimodal && (
                                      <span
                                        className="inline-flex shrink-0 items-center px-2 py-0.5 text-xs rounded-[3px]"
                                        style={{
                                          background: "var(--alert-warning-bg)",
                                          color: "var(--text-warning)",
                                          border: "1px solid var(--alert-warning-border)",
                                        }}
                                      >
                                        多模态
                                      </span>
                                    )}
                                  </div>
                                  <span className="text-xs" style={{ color: "var(--muted-foreground)" }}>
                                    {model.isCustom ? (model.customName || "—") : model.versionLabel}
                                  </span>
                                </div>
                                <div className="flex items-center gap-1 shrink-0">
                                  <button
                                    type="button"
                                    aria-label="编辑"
                                    className="text-[var(--muted-foreground)] hover:text-[var(--text-brand)] transition-colors"
                                    onClick={() => handleEditPrimary(model.id)}
                                  >
                                    <Edit3 className="w-4 h-4" />
                                  </button>
                                  <button
                                    type="button"
                                    aria-label="删除"
                                    className="p-1 rounded text-[var(--muted-foreground)] hover:text-[var(--text-danger)] transition-colors"
                                    onClick={() => handleRequestDeleteModel(model.id, true)}
                                  >
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </button>
                                </div>
                              </div>
                            </div>
                          ))
                        ) : (
                          <div className="py-3 text-center text-xs" style={{ color: "var(--text-weak)" }}>
                            暂无主模型
                          </div>
                        )}
                      </div>

                      {/* 备用模型 */}
                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <div className="text-sm font-medium" style={{ color: "var(--text-body)" }}>
                            备用模型（{appliedModels.filter(m => !m.primary).length}）
                          </div>
                        </div>
                        {/* 备选模型提示 */}
                        {appliedModels.filter(m => !m.primary).length > 0 && (
                        <div
                          className="flex items-start gap-2 rounded-[var(--radius-card)] px-3 py-2.5 mb-3"
                          style={{ background: "rgba(20,71,230,0.06)" }}
                        >
                          <AlertInfoIcon className="w-4 h-4 shrink-0 mt-0.5 text-[var(--text-brand)]" />
                          <span className="text-xs leading-[18px]" style={{ color: "var(--text-brand)" }}>
                            主模型不可用时会自动切换备选模型，此时备选模型消耗的token将统计到主模型下
                          </span>
                        </div>
                        )}
                        {/* 备用模型列表 */}
                        {appliedModels.filter(m => !m.primary).length > 0 ? (
                          <div className="space-y-2">
                            {appliedModels.filter(m => !m.primary).map((model) => (
                              <div
                                key={model.id}
                                className="rounded-[var(--radius-card)] p-3 flex items-center justify-between"
                                style={{ border: "1px solid var(--border)" }}
                              >
                                <div className="flex flex-col gap-0.5">
                                  <div className="flex items-center gap-2">
                                    <span className="text-sm font-medium" style={{ color: "var(--foreground)" }}>
                                      {model.providerLabel}
                                    </span>
                                    {model.isCustom && model.multimodal && (
                                      <span
                                        className="inline-flex shrink-0 items-center px-2 py-0.5 text-xs rounded-[3px]"
                                        style={{
                                          background: "var(--alert-warning-bg)",
                                          color: "var(--text-warning)",
                                          border: "1px solid var(--alert-warning-border)",
                                        }}
                                      >
                                        多模态
                                      </span>
                                    )}
                                  </div>
                                  <span className="text-xs" style={{ color: "var(--muted-foreground)" }}>
                                    {model.isCustom ? (model.customName || "—") : model.versionLabel}
                                  </span>
                                </div>
                                <button
                                  className="p-1 rounded text-[var(--muted-foreground)] hover:text-[var(--text-danger)] transition-colors"
                                  onClick={() => handleRequestDeleteModel(model.id, false)}
                                >
                                  <Trash2 className="w-3.5 h-3.5" />
                                </button>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <div className="py-4 text-center text-xs" style={{ color: "var(--text-weak)" }}>
                            暂无备用模型
                          </div>
                        )}
                      </div>
                    </div>
                  </TenantCard>

                  {/* ===== 02/ 通道（Channels） ===== */}
                  <TenantCard padding="none" className="flex flex-col p-6 gap-3">
                    {/* 标题区 */}
                    <div
                      className="flex flex-col gap-1 pb-5"
                      style={{ borderBottom: "1px solid var(--border)" }}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-end gap-2">
                          <span
                            className="text-[18px] leading-6"
                            style={{ fontFamily: "Menlo, Consolas, 'Courier New', monospace", color: "var(--text-brand)" }}
                          >
                            02/
                          </span>
                          <span className="text-[18px] font-medium leading-6" style={{ color: "var(--text-emphasis)" }}>
                            通道
                          </span>
                          <span className="text-[15px] leading-6" style={{ color: "var(--text-weak)" }}>
                            Channels
                          </span>
                        </div>
                        {appliedChannels.length > 0 ? (
                          <ConfiguredBadge />
                        ) : (
                          <span
                            className="inline-flex items-center gap-1.5 h-5 px-2 text-xs shrink-0"
                            style={{ color: "var(--text-weak)", letterSpacing: "0.015em" }}
                          >
                            <span className="w-2 h-2 rounded-full" style={{ background: "var(--text-weak)" }} />
                            未配置
                          </span>
                        )}
                      </div>
                      <p className="text-sm leading-[18px] min-h-[36px]" style={{ color: "var(--muted-foreground)" }}>
                        用户与 Agent 交互的入口，支持微信、QQ、飞书等
                      </p>
                    </div>

                    {/* 通道配置：新增通道按钮 / 配置卡切换 */}
                    <div>
                        {/* 新增通道按钮（收起态） */}
                        {!showChannelConfig && (
                          <Button
                            variant="tenant-outline-strong"
                            size="lg"
                            className="w-full mb-3"
                            onClick={() => setShowChannelConfig(true)}
                          >
                            <Plus className="w-3.5 h-3.5" />
                            新增通道
                          </Button>
                        )}
                        {/* 添加接入通道配置卡（展开态） */}
                        {showChannelConfig && (
                          <div className="relative rounded-[var(--radius-card)] bg-[var(--bg-grey-normal)] border border-[var(--border)] p-3 space-y-3 mb-3">
                            <button
                              type="button"
                              onClick={() => {
                                setChannelFields({});
                                setShowChannelConfig(false);
                              }}
                              className="absolute top-2 right-2 w-6 h-6 rounded-[var(--radius-lg)] flex items-center justify-center text-[var(--muted-foreground)] hover:text-[var(--foreground)] hover:bg-[var(--accent)] transition-colors z-10"
                              aria-label="关闭"
                            >
                              <X className="w-3.5 h-3.5" />
                            </button>
                            {/* 小标题 */}
                            <div className="text-xs font-medium" style={{ color: "var(--text-body)" }}>
                              添加接入通道
                            </div>
                            {/* 通道选择下拉 */}
                            <div className="flex items-center gap-2">
                              <Select
                                value={selectedChannel}
                                onValueChange={(v) => {
                                  setSelectedChannel(v);
                                  setChannelFields({});
                                  setWhatsappPhone("");
                                  setFeishuConfigMode("quick");
                                  setWeworkConfigMode("quick");
                                  setChannelConfigMode("quick");
                                }}
                              >
                                <SelectTrigger className="w-full border-[var(--border)] bg-white rounded-[var(--radius-lg)]">
                                  <SelectValue placeholder="选择通道类型" />
                                </SelectTrigger>
                                <SelectContent>
                                  {allChannelOptions.map((c) => (
                                    <SelectItem key={c.value} value={c.value}>
                                      {c.label}
                                    </SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                              {/* 企业微信通道：示意图 Tooltip（与 main 对齐） */}
                              {currentChannelConfig?.hasInfoIcon && (
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <button
                                      type="button"
                                      className="shrink-0 text-[var(--text-muted)] hover:text-[var(--text-brand)] transition-colors"
                                      aria-label="查看企业微信配置示意图"
                                    >
                                      <Info className="w-4 h-4" />
                                    </button>
                                  </TooltipTrigger>
                                  <TooltipContent
                                    side="right"
                                    className="p-0 border-0 shadow-xl bg-transparent"
                                    sideOffset={8}
                                  >
                                    <img
                                      src="https://d2xsxph8kpxj0f.cloudfront.net/310519663415970324/bygiZj33T3TUvGMBPvApKE/pasted_file_To1FVK_image_06b2d1cc.png"
                                      alt="企业微信通道示意图"
                                      className="rounded-[4px] max-w-xs"
                                      style={{ width: 320 }}
                                    />
                                  </TooltipContent>
                                </Tooltip>
                              )}
                            </div>

                            {/* WhatsApp 自定义通道：直接显示手机号输入框，跳过快捷/手动 Tab */}
                            {currentChannelConfig?.whatsappMode ? (
                              <div className="space-y-3">
                                <Input
                                  type="tel"
                                  placeholder="手机号（带国家码，不含+，如 85266803489）"
                                  value={whatsappPhone}
                                  onChange={(e) => setWhatsappPhone(e.target.value)}
                                  className="h-9 text-sm bg-white border-[var(--border)] rounded-[var(--radius-lg)]"
                                />
                                <Button
                                  variant="tenant-outline"
                                  size="claw"
                                  className="w-full"
                                  onClick={handleApplyChannel}
                                >
                                  添加并应用
                                </Button>
                              </div>
                            ) : null}

                            {/* 快捷配置 / 手动配置 切换：飞书 / 企业微信走独立 state；微信不显示 Tab；其他通道走全局 */}
                            {selectedChannel && !currentChannelConfig?.wechatMode && !currentChannelConfig?.whatsappMode && (
                              <Tabs
                                value={
                                  currentChannelConfig?.feishuMode
                                    ? feishuConfigMode
                                    : currentChannelConfig?.weworkMode
                                      ? weworkConfigMode
                                      : channelConfigMode
                                }
                                onValueChange={(v) => {
                                  const mode = v as "quick" | "manual";
                                  if (currentChannelConfig?.feishuMode) setFeishuConfigMode(mode);
                                  else if (currentChannelConfig?.weworkMode) setWeworkConfigMode(mode);
                                  else setChannelConfigMode(mode);
                                }}
                              >
                                <TabsList className="w-full rounded-full">
                                  <TabsTrigger value="quick" className="rounded-full">快捷配置</TabsTrigger>
                                  <TabsTrigger value="manual" className="rounded-full">手动配置</TabsTrigger>
                                </TabsList>
                              </Tabs>
                            )}

                            {/* 当前配置模式（按通道分流；微信视为快捷；WhatsApp 已单独渲染） */}
                            {!currentChannelConfig?.whatsappMode && (() => {
                              const effectiveMode: "quick" | "manual" = currentChannelConfig?.wechatMode
                                ? "quick"
                                : currentChannelConfig?.feishuMode
                                  ? feishuConfigMode
                                  : currentChannelConfig?.weworkMode
                                    ? weworkConfigMode
                                    : channelConfigMode;

                              // 快捷配置模式：单按钮"前往授权"或"添加并应用"
                              if (effectiveMode === "quick" && selectedChannel) {
                                const needAuth = currentChannelConfig?.wechatMode
                                  || (currentChannelConfig?.feishuMode && feishuConfigMode === "quick")
                                  || (currentChannelConfig?.weworkMode && weworkConfigMode === "quick");
                                return (
                                  <Button
                                    variant="tenant-outline"
                                    size="sm"
                                    className="w-full"
                                    onClick={handleApplyChannel}
                                  >
                                    {needAuth ? "前往授权" : "添加并应用"}
                                  </Button>
                                );
                              }

                              // 手动配置模式：表单字段
                              if (effectiveMode === "manual" && currentChannelConfig?.fields && currentChannelConfig.fields.length > 0) {
                                return (
                                  <div className="space-y-3">
                                    {currentChannelConfig.fields.map((field) => (
                                      <div key={field.key}>
                                        <div className="relative">
                                          <Input
                                            type={field.secret && !visibleSecrets.has(field.key) ? "password" : "text"}
                                            placeholder={field.label}
                                            value={channelFields[field.key] || ""}
                                            onChange={(e) => setChannelFields({ ...channelFields, [field.key]: e.target.value })}
                                            className="h-9 text-sm pr-9 bg-white border-[var(--border)] rounded-[var(--radius-lg)]"
                                          />
                                          {field.secret && (
                                            <button
                                              type="button"
                                              onClick={() => toggleSecretVisibility(field.key)}
                                              className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--muted-foreground)] hover:text-[var(--foreground)] transition-colors"
                                            >
                                              {visibleSecrets.has(field.key) ? <Eye className="w-4 h-4" /> : <EyeOff className="w-4 h-4" />}
                                            </button>
                                          )}
                                        </div>
                                      </div>
                                    ))}
                                    <Button
                                      variant="tenant-outline"
                                      size="sm"
                                      className="w-full"
                                      onClick={handleApplyChannel}
                                    >
                                      添加并应用
                                    </Button>
                                  </div>
                                );
                              }

                              return null;
                            })()}


                          </div>
                        )}

                      {appliedChannels.length > 0 ? (
                      <>
                      <div className="mb-3">
                        <div className="text-sm font-medium" style={{ color: "var(--text-body)" }}>
                          已接入通道（{appliedChannels.length}）
                        </div>
                      </div>

                        <div className="space-y-2">
                          {appliedChannels.map((ch, idx) => (
                            <div
                              key={`${ch.channelValue}-${idx}`}
                              className="rounded-[var(--radius-card)] border border-[var(--border)] overflow-hidden"
                            >
                              {/* 通道头部（微信通道不可展开） */}
                              <div
                                className={`flex items-center justify-between px-4 py-2.5 transition-colors ${
                                  ch.channelValue === "wechat"
                                    ? ""
                                    : "cursor-pointer hover:bg-[var(--accent)]"
                                }`}
                                onClick={() => {
                                  if (ch.channelValue === "wechat") return;
                                  setExpandedChannelIdx(prev => prev === idx ? null : idx);
                                }}
                              >
                                <div className="flex items-center gap-2">
                                  <span className="text-sm font-medium" style={{ color: "var(--foreground)" }}>{ch.type}</span>
                                  <Badge variant="secondary" className="bg-[rgba(22,163,74,0.08)] border-0 text-[var(--text-success)] text-[10px]">
                                    运行中
                                  </Badge>
                                </div>
                                <div className="flex items-center gap-1">
                                  <button
                                    onClick={(e) => { e.stopPropagation(); handleDeleteChannel(idx); }}
                                    className="p-1 rounded text-[var(--muted-foreground)] hover:text-[var(--text-danger)] transition-colors"
                                  >
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </button>
                                  {ch.channelValue !== "wechat" && (
                                    expandedChannelIdx === idx
                                      ? <ChevronDown className="w-4 h-4 text-[var(--muted-foreground)]" />
                                      : <ChevronDown className="w-4 h-4 text-[var(--muted-foreground)] -rotate-90" />
                                  )}
                                </div>
                              </div>
                              {/* 展开的配置详情（微信无展开内容） */}
                              {ch.channelValue !== "wechat" && expandedChannelIdx === idx && (ch.fields.length > 0 || ch.channelValue === "feishu") && (
                                <div className="border-t border-[var(--border)] px-4 py-3 bg-[var(--bg-grey-normal)] space-y-2">
                                  {ch.fields.map((field) => {
                                    const val = ch.fieldValues[field.key] || "";
                                    const uniqueKey = `${idx}-${field.key}`;
                                    const isVisible = visibleAppliedSecrets.has(uniqueKey);
                                    const displayVal = field.secret && !isVisible ? maskSecret(val) : val;
                                    return (
                                      <div key={field.key} className="flex items-center gap-1 text-sm">
                                        <span className="text-[var(--text-muted)] shrink-0">{field.label}：</span>
                                        <span className="text-[var(--text-emphasis)] font-mono break-all flex-1">{displayVal || "—"}</span>
                                        {field.secret && (
                                          <button
                                            onClick={() => toggleAppliedSecretVisibility(idx, field.key)}
                                            className="text-[var(--muted-foreground)] hover:text-[var(--foreground)] transition-colors p-0.5"
                                          >
                                            {isVisible ? <Eye className="w-3.5 h-3.5" /> : <EyeOff className="w-3.5 h-3.5" />}
                                          </button>
                                        )}
                                      </div>
                                    );
                                  })}
                                  {/* 飞书 pairing code：手动审批分支补输配对码（与 main 对齐） */}
                                  {ch.channelValue === "feishu" && (
                                    <div className="flex items-center gap-2 pt-1">
                                      <Input
                                        tenant
                                        placeholder="（如需）请输入 pairing code"
                                        value={feishuPairingCode}
                                        onChange={(e) => setFeishuPairingCode(e.target.value)}
                                        className="text-sm h-8 flex-1"
                                        onKeyDown={(e) => e.key === "Enter" && handleFeishuPairing()}
                                      />
                                      <Button
                                        variant="tenant-outline"
                                        size="sm"
                                        className="shrink-0 text-sm"
                                        onClick={handleFeishuPairing}
                                      >
                                        匹配
                                      </Button>
                                    </div>
                                  )}
                                </div>
                              )}
                            </div>
                          ))}
                        </div>
                      </>
                      ) : (
                      <>
                      <div className="mb-3">
                        <div className="text-sm font-medium" style={{ color: "var(--text-body)" }}>
                          已接入通道（0）
                        </div>
                      </div>
                      <div className="py-4 text-center text-xs" style={{ color: "var(--text-weak)" }}>
                        暂无接入通道
                      </div>
                      </>
                      )}
                      </div>
                  </TenantCard>
                    </>
                  )}

                  {hasLocalResourceScopes ? (
                    <LocalAgentSettingsPanel
                      resources={localAgentResources}
                      organizationName={detailAgent?.groupName || "默认"}
                    />
	                  ) : isLocalAgent ? (
	                    <ExternalAgentResourcesPanel
	                      isInactive={isLocalInactive}
	                      organizationName={detailAgent?.groupName || "默认"}
	                      resources={localAgentResources}
	                    />
                  ) : (
                  <>
                  {/* ===== 03/ 技能（Skills） ===== */}
                  <TenantCard
                    padding="none"
                    className={
                      isLocalAgent
                        // 本地 Agent：仅此一张卡，撑满视口
                        ? "flex flex-col p-6 gap-3 min-h-[calc(100vh-280px)]"
                        : "flex flex-col p-6 gap-3"
                    }
                  >
                    <div
                      className="flex flex-col gap-1 pb-5"
                      style={{ borderBottom: "1px solid var(--border)" }}
                    >
                      <div className="flex items-center justify-between">
                        <div className="flex items-end gap-2">
                          {!isLocalAgent && (
                            <span
                              className="text-[18px] leading-6"
                              style={{ fontFamily: "Menlo, Consolas, 'Courier New', monospace", color: "var(--text-brand)" }}
                            >
                              03/
                            </span>
                          )}
                          <div className="relative text-[18px] font-medium leading-6" style={{ color: "var(--text-emphasis)" }}>
                            技能
                            <SkillManagementUpdateNotice />
                          </div>
                          <span className="text-[15px] leading-6" style={{ color: "var(--text-weak)" }}>
                            Skills
                          </span>
                        </div>
                        <ConfiguredBadge />
                      </div>
                      <p className="text-sm leading-[18px] min-h-[36px]" style={{ color: "var(--muted-foreground)" }}>
                        为 Agent 添加搜索、绘图等扩展能力
                      </p>
                    </div>

                    {/* 安装技能（统一使用 tenant-outline-strong 变体，胶囊圆角） */}
                    <Button
                      variant="tenant-outline-strong"
                      size="lg"
                      className="w-full"
                      onClick={() => setSkillModalOpen(true)}
                    >
                      <Plus className="w-3.5 h-3.5" />
                      安装技能
                    </Button>

                    {/* 已安装技能列表 */}
                    <div className={isLocalAgent ? "flex flex-col gap-3 mt-1 pt-2 flex-1 min-h-0" : "flex flex-col gap-3 mt-1 pt-2"}>
                      <div className="flex items-center justify-between">
                        <div className="text-sm font-medium" style={{ color: "var(--text-body)" }}>
                          已安装技能（{installedSkills.filter((s) => skillSearch ? s.name.includes(skillSearch) : true).length}）
                        </div>
                      </div>
                      {/* Skill 搜索输入框 */}
                      <div className="relative">
                        <Search
                          className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 pointer-events-none"
                          style={{ color: "var(--text-weak)" }}
                        />
                        <Input
                          tenant
                          placeholder="输入skill名称搜索"
                          className="h-9 pl-9 text-sm"
                          value={skillSearch}
                          onChange={(e) => setSkillSearch(e.target.value)}
                        />
                      </div>
                      <div
                        className={
                          isLocalAgent
                            ? "rounded-[var(--radius-card)] flex flex-col flex-1 min-h-[280px] overflow-y-auto"
                            : "rounded-[var(--radius-card)] flex flex-col h-[280px] overflow-y-auto"
                        }
                        style={{ background: "var(--bg-grey-normal)", border: "1px solid var(--border)" }}
                      >
                        {(() => {
                          const filtered = installedSkills.filter((s) =>
                            skillSearch ? s.name.includes(skillSearch) : true,
                          );
                          if (filtered.length === 0 && skillSearch) {
                            return (
                              <div className="flex-1 flex items-center justify-center text-xs text-[var(--text-muted)]">
                                未找到相关技能
                              </div>
                            );
                          }
                          return filtered.map((s, idx) => {
                            const { latestVersion, hasUpdate } = getSkillUpdateInfo(s);
                            return (
                              <div
                                key={`${s.name}-${idx}`}
                                className="group flex items-center justify-between gap-2 px-4 text-sm h-9 shrink-0"
                                style={{ color: "var(--foreground)" }}
                              >
                                <span className="truncate">{s.name} {s.version}</span>
                                <div className="flex items-center gap-0.5 flex-shrink-0 -mr-1.5">
                                  {hasUpdate && (
                                    <Tooltip>
                                      <TooltipTrigger asChild>
                                        <button
                                          type="button"
                                          aria-label="更新技能"
                                          onClick={() => handleUpdateSkill(s)}
                                          className="w-7 h-7 inline-flex items-center justify-center rounded-[4px] text-[var(--text-brand)] hover:bg-[var(--accent)] transition-colors"
                                        >
                                          <RefreshCw className="w-3.5 h-3.5" />
                                        </button>
                                      </TooltipTrigger>
                                      <TooltipContent side="top" className="text-xs">
                                        更新至 {latestVersion}
                                      </TooltipContent>
                                    </Tooltip>
                                  )}
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <button
                                        type="button"
                                        aria-label="卸载技能"
                                        onClick={() => setSkillUninstallConfirm({ open: true, name: s.name, version: s.version })}
                                        className="w-7 h-7 inline-flex items-center justify-center rounded-[4px] text-[var(--text-muted)] hover:text-[var(--text-danger)] hover:bg-[var(--accent)] transition-colors"
                                      >
                                        <X className="w-3.5 h-3.5" />
                                      </button>
                                    </TooltipTrigger>
                                    <TooltipContent side="top" className="text-xs">
                                      卸载
                                    </TooltipContent>
                                  </Tooltip>
                                </div>
                              </div>
                            );
                          });
                        })()}
                      </div>
                    </div>

                    {/* 待安装技能（动态状态机：pending / installing / failed） */}
                    {pendingSkills.length > 0 && (
                      <div className="flex flex-col gap-3 mt-1">
                        <div className="flex items-center justify-between">
                          <div className="text-sm font-medium" style={{ color: "var(--text-body)" }}>
                            待安装技能（{pendingSkills.length}）
                          </div>
                          {pendingSkills.some((s) => s.status === "failed") && (
                            <div className="flex items-center gap-3">
                              <button
                                className="inline-flex items-center gap-1 text-xs hover:opacity-80"
                                style={{ color: "var(--text-brand)" }}
                                onClick={handleRetryAllFailed}
                              >
                                <RefreshCw className="w-3 h-3" />
                                重试
                              </button>
                              <button
                                className="inline-flex items-center gap-1 text-xs hover:opacity-80"
                                style={{ color: "var(--text-brand)" }}
                                onClick={handleDeleteAllFailed}
                              >
                                <Trash2 className="w-3 h-3" />
                                删除
                              </button>
                            </div>
                          )}
                        </div>
                        <div
                          className="rounded-[var(--radius-card)] flex flex-col max-h-[280px] overflow-y-auto"
                          style={{ background: "var(--bg-grey-normal)", border: "1px solid var(--border)" }}
                        >
                          {pendingSkills.map((s) => (
                            <div
                              key={s.id}
                              className="flex items-center justify-between px-4 h-9 shrink-0"
                            >
                              <span className="text-sm truncate" style={{ color: "var(--foreground)" }}>
                                {s.name}
                              </span>
                              {s.status === "installing" && (
                                <span
                                  className="inline-flex items-center gap-1 text-xs flex-shrink-0"
                                  style={{ color: "var(--text-brand)" }}
                                >
                                  <Loader2 className="w-3 h-3 animate-spin" />
                                  安装中
                                </span>
                              )}
                              {s.status === "pending" && (
                                <span className="text-xs flex-shrink-0" style={{ color: "var(--muted-foreground)" }}>
                                  待安装
                                </span>
                              )}
                              {s.status === "failed" && (
                                <span
                                  className="inline-flex items-center gap-1 text-xs flex-shrink-0"
                                  style={{ color: "var(--text-danger)" }}
                                >
                                  <svg
                                    width="12"
                                    height="12"
                                    viewBox="0 0 16 16"
                                    fill="none"
                                    xmlns="http://www.w3.org/2000/svg"
                                    aria-hidden="true"
                                  >
                                    <g clipPath="url(#clip0_70703_2337)">
                                      <path
                                        fillRule="evenodd"
                                        clipRule="evenodd"
                                        d="M8.00032 1.99984C11.3141 1.99984 14.0004 4.68613 14.0004 7.99984C14.0004 11.3135 11.3141 13.9998 8.00032 13.9998C4.68662 13.9998 2.00032 11.3135 2.00032 7.99984C2.00032 4.68613 4.68661 1.99984 8.00032 1.99984ZM15.3337 7.99984C15.3337 3.94975 12.0505 0.666503 8.00032 0.666504C3.95024 0.666504 0.666991 3.94975 0.666992 7.99984C0.666992 12.0499 3.95024 15.3332 8.00032 15.3332C12.0505 15.3332 15.3337 12.0499 15.3337 7.99984ZM7.33366 4.33317V9.33317H8.66699V4.33317H7.33366ZM8.66699 10.3332H7.33105V11.6691H8.66699V10.3332Z"
                                        fill="currentColor"
                                      />
                                    </g>
                                    <defs>
                                      <clipPath id="clip0_70703_2337">
                                        <rect width="16" height="16" fill="white" />
                                      </clipPath>
                                    </defs>
                                  </svg>
                                  安装失败，请重试
                                </span>
                              )}
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </TenantCard>
                  </>
                  )}
                </div>
                </>
              )}

              {/* 工具管理 tab */}
              {activeTab === "tools" && (
                <div className="w-full">
                  <ToolsMcpPanel />
                </div>
              )}

              {/* 记忆管理 tab */}
              {activeTab === "memory" && (
                <TenantSection
                  title={
                    <span className="flex items-center gap-2">
                      <span>Memory Pro 服务</span>
                      <Badge
                        variant="secondary"
                        className={memoryStatus === "none" ? "bg-[var(--muted)] border-0 text-[var(--text-muted)]" : "bg-[rgba(22,163,74,0.08)] border-0 text-[var(--text-success)]"}
                      >
                        {memoryStatus === "none" ? "已关闭" : "已开启"}
                      </Badge>
                    </span>
                  }
                  cardPadding="none"
                  className="p-6"
                >
                  <p className="text-sm -mt-1 mb-4" style={{ color: "var(--muted-foreground)", lineHeight: "17px" }}>
                    基于腾讯云向量数据库的企业级记忆服务，实现语义级记忆检索与数据管理。
                  </p>
                  <MemoryPreview
                    memoryStatus={memoryStatus}
                    proQuotaAvailable={proQuotaAvailable}
                    showConfidence={false}
                    showHeader={false}
                    isLoading={memoryLoading}
                    onStatusChange={async (newStatus) => {
                      await new Promise((resolve) => setTimeout(resolve, 1000));
                      setMemoryStatus(newStatus);
                    }}
                  />
                </TenantSection>
              )}

              {/* 网盘管理 tab */}
              {activeTab === "files" && (
                fileSpaceStatus === "closed" ? (
                  <TenantSection
                    title={
                      <span className="flex items-center gap-2">
                        <span>网盘管理</span>
                        <Badge variant="secondary" className="bg-[var(--muted)] border-0 text-[var(--text-muted)]">已关闭</Badge>
                      </span>
                    }
                    cardPadding="none"
                    className="p-6"
                  >
                    <p className="text-sm -mt-1 mb-4" style={{ color: "var(--text-muted)", lineHeight: "17px" }}>
                      为您提供专属、安全的云存储空间，由腾讯云存储 Agent Storage 服务提供支持
                    </p>
                    <FileSpace
                      clawName="OpenClaw 引导预览"
                      clawId="guide-preview"
                      status={fileSpaceStatus}
                      showHeader={false}
                      basePath="https://smh3jsttekkpsoqw.api.tencentsmh.cn"
                      libraryId="smh3jsttekkpsoqw"
                      spaceId="space232t1yug3w7up"
                      getAccessToken={async () => ({
                        accessToken:
                          "acctk021cf0f24emnem68z3dzwr734zcdpl74fd7783cgdesppskermqhhu7d9pnns4exa5gvc84n2yfhdq5unt754belzzvkwcd5psjuznzwt7jbcs2zsm5c3828ba4",
                        expiresAt: Date.now() + 3600 * 24 * 1000,
                      })}
                    />
                  </TenantSection>
                ) : (
                  <FileSpace
                    clawName="OpenClaw 引导预览"
                    clawId="guide-preview"
                    status={fileSpaceStatus}
                    basePath="https://smh3jsttekkpsoqw.api.tencentsmh.cn"
                    libraryId="smh3jsttekkpsoqw"
                    spaceId="space232t1yug3w7up"
                    getAccessToken={async () => ({
                      accessToken:
                        "acctk021cf0f24emnem68z3dzwr734zcdpl74fd7783cgdesppskermqhhu7d9pnns4exa5gvc84n2yfhdq5unt754belzzvkwcd5psjuznzwt7jbcs2zsm5c3828ba4",
                      expiresAt: Date.now() + 3600 * 24 * 1000,
                    })}
                  />
                )
              )}

              {/* 龙虾医院 tab（仅含「一键修复」卡片，引导页不嵌入龙虾医生对话） */}
              {activeTab === "doctor" && (
                <div className="flex flex-col gap-5">
                  <TenantSection title="一键修复" cardPadding="none" className="p-6">
                    <p className="text-sm -mt-1 mb-4" style={{ color: "var(--muted-foreground)" }}>
                      适合龙虾配置文件中 API KEY、插件、通道等配置异常导致无法启动等常见问题，系统自动检测并尝试修复。
                    </p>
                    <ul className="space-y-2 mb-6">
                      <li className="flex items-center gap-2 text-sm" style={{ color: "var(--text-secondary)" }}>
                        <span className="w-1.5 h-1.5 rounded-full flex-shrink-0" style={{ backgroundColor: "var(--text-weak)" }} />
                        自动执行
                        <code className="px-2 py-0.5 rounded-[2px] font-mono text-xs" style={{ backgroundColor: "var(--muted)", color: "var(--text-secondary)" }}>agent doctor --fix</code>
                      </li>
                      <li className="flex items-center gap-2 text-sm" style={{ color: "var(--text-secondary)" }}>
                        <span className="w-1.5 h-1.5 rounded-full flex-shrink-0" style={{ backgroundColor: "var(--text-weak)" }} />
                        自动恢复常见配置问题
                      </li>
                      <li className="flex items-center gap-2 text-sm" style={{ color: "var(--text-secondary)" }}>
                        <span className="w-1.5 h-1.5 rounded-full flex-shrink-0" style={{ backgroundColor: "var(--text-weak)" }} />
                        恢复前会将配置文件备份
                      </li>
                    </ul>
                    <div className="pt-4" style={{ borderTop: "1px solid var(--border)" }}>
                      {quickFixState === "idle" && (
                        <Button variant="tenant-outline-strong" size="claw-lg" onClick={runQuickFixMock}>
                          <Wrench className="w-4 h-4" />
                          一键修复
                        </Button>
                      )}
                      {quickFixState === "loading" && (
                        <div className="inline-flex items-center gap-2 px-3 h-8 rounded-[var(--radius-lg)] text-xs" style={{ backgroundColor: "var(--muted)", color: "var(--muted-foreground)" }}>
                          <span className="w-3 h-3 border-2 rounded-full animate-spin" style={{ borderColor: "var(--border)", borderTopColor: "var(--muted-foreground)" }} />
                          正在执行修复
                        </div>
                      )}
                      {quickFixState === "success" && (
                        <div className="flex items-center gap-2.5 flex-wrap">
                          <span className="badge-running inline-flex items-center gap-1.5">
                            <CheckCircle2 className="w-3.5 h-3.5 flex-shrink-0" />
                            修复成功
                          </span>
                          <span className="text-xs" style={{ color: "var(--muted-foreground)" }}>Gateway 已正常启动，请前往 Agent 对话确认问题是否已解决</span>
                        </div>
                      )}
                      {quickFixState === "failed" && (
                        <div className="flex items-center gap-2.5 flex-wrap">
                          <span className="badge-stopped inline-flex items-center gap-1.5">
                            <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
                            修复失败
                          </span>
                          <span className="text-xs" style={{ color: "var(--muted-foreground)" }}>{quickFixFailReason}，建议开启龙虾医生进行深度诊断</span>
                        </div>
                      )}
                    </div>
                  </TenantSection>

                  {/* ===== 龙虾医生对话卡片（受管控端「允许用户使用龙虾医生」开关控制） =====
                      与 OpenClawDetail.tsx line 3356 完全对齐：管控端未允许 → 整卡不渲染；
                      允许 → 复用同一份 <DoctorChatCard>，自带 Badge/开始诊断/待命中/对话区
                      /结束诊断/回滚等完整生命周期，禁止另起炉灶做静态占位。 */}
                  {lobsterDoctorEnabled && (
                    <div data-doctor-chat-card>
                      <DoctorChatCard
                        instanceId={detailAgent?.instanceId || "ins-grpdemo02"}
                        instanceName={agentName}
                      />
                    </div>
                  )}
                </div>
              )}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 安装新技能弹窗 */}
      <SkillInstallModal open={skillModalOpen} onOpenChange={setSkillModalOpen} onEnqueue={handleEnqueueSkills} />

      {/* 当前实例直接切换 Agent 类型，实例入口和 ID 保持不变。 */}
      <AgentTypeSwitchDialog
        open={agentTypeSwitchOpen}
        onOpenChange={setAgentTypeSwitchOpen}
        currentType={displayAgentType}
        onConfirm={handleAgentTypeSwitchConfirm}
        onViewBackups={openDataBackup}
      />

      {/* 开启 Agent 面板弹窗 */}
      <Dialog open={panelDialogOpen} onOpenChange={setPanelDialogOpen}>
        <DialogContent className="sm:max-w-fit">
          <DialogHeader>
            <DialogTitle className="text-base font-bold text-foreground">开启面板</DialogTitle>
          </DialogHeader>
          <div className="py-2 space-y-5">
            <Alert variant="info">
              <AlertInfoIcon />
              <AlertDescription>
                访问链接已生成，该链接含有您的 API Key 和加密配置，请勿分享给第三方，以防隐私泄露或资产损失。
              </AlertDescription>
            </Alert>
            <div className="rounded-[var(--radius-card)] border border-[var(--border)] overflow-hidden">
              <div className="flex items-center gap-3 px-4 py-3">
                <span className="text-sm text-muted-foreground w-24 shrink-0">WebSocket URL</span>
                <a
                  href="http://43.139.137.45:38341/knmnz8?token=8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex-1 text-sm text-foreground font-mono truncate hover:underline cursor-pointer"
                >
                  http://43.139.137.45:38341/knmnz8?token=8512b8ef...
                </a>
                <button
                  className="p-1.5 rounded-[var(--radius-lg)] hover:bg-[var(--alert-info-bg)] text-[var(--muted-foreground)] hover:text-[var(--text-brand)] transition-colors"
                  onClick={() => { navigator.clipboard.writeText("http://43.139.137.45:38341/knmnz8?token=8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb"); toast.success("已复制"); }}
                >
                  <Copy className="w-4 h-4" />
                </button>
              </div>
              <div className="h-px bg-[var(--border)]" />
              <div className="flex items-center gap-3 px-4 py-3">
                <span className="text-sm text-muted-foreground w-24 shrink-0">网关令牌</span>
                <span className="flex-1 text-sm text-foreground font-mono truncate">
                  8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb
                </span>
                <button
                  className="p-1.5 rounded-[var(--radius-lg)] hover:bg-[var(--alert-info-bg)] text-[var(--muted-foreground)] hover:text-[var(--text-brand)] transition-colors"
                  onClick={() => { navigator.clipboard.writeText("8512b8ef93cdfd393ad6af5efa42c1e54981f3cb69f381eb"); toast.success("已复制"); }}
                >
                  <Copy className="w-4 h-4" />
                </button>
              </div>
            </div>
            <p className="text-sm text-muted-foreground leading-relaxed">
              用浏览器打开 WebSocket URL，如面板需要填入网关令牌，则将网关令牌复制并粘贴过去，即可进入面板。
            </p>
          </div>
          <DialogFooter className="gap-3">
            <Button
              variant="tenant-outline"
              onClick={() => setPanelDialogOpen(false)}
            >
              关闭
            </Button>
            <Button
              variant="tenant-primary"
              onClick={() => window.open(DEMO_OPENCLAW_PANEL_URL, "_blank", "noopener,noreferrer")}
            >
              立即访问
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ==================== Agent 迁移弹窗 ==================== */}
      <Dialog open={migrationOpen} onOpenChange={setMigrationOpen}>
        <DialogContent size="lg">
          <DialogHeader>
            <DialogTitle>
              迁移 Agent 至当前实例
            </DialogTitle>
            <DialogDescription className="text-xs text-[var(--text-muted)]">
              将源端 Agent 的配置、通道状态、会话历史导入到当前实例
            </DialogDescription>
          </DialogHeader>

          <DialogBody className="px-6 space-y-5 pb-6">
            {/* 注意事项 */}
            <Alert variant="warning">
              <CircleAlert className="w-4 h-4" />
              <AlertTitle>注意事项</AlertTitle>
              <AlertDescription>
                <ul className="text-xs space-y-1 list-disc pl-4 leading-relaxed">
                  <li><strong>源端 Agent 类型必须与当前实例的 Agent 类型一致</strong>，否则配置文件将无法兼容，导致迁移失败</li>
                  <li>源端 Agent 的配置、通道登录状态、会话历史将完整导入到当前实例，源端仅做读取打包，不影响源端正常运行</li>
                  <li>导入将覆盖当前实例的 ~/.agent/ 目录，导入前自动备份，失败自动回滚</li>
                </ul>
              </AlertDescription>
            </Alert>

            {/* Step 1: 导出源端配置 */}
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className={`w-5 h-5 rounded-full flex items-center justify-center text-xs font-bold ${
                  migrationStep === "export" || migrationStep === "waitUpload"
                    ? "bg-[var(--text-brand)] text-white" : "bg-[var(--text-success)] text-white"
                }`}>
                  {migrationStep !== "export" && migrationStep !== "waitUpload" ? <Check className="w-3 h-3" strokeWidth={3} /> : "1"}
                </div>
                <h3 className="text-sm font-semibold text-[var(--text-title)]">导出源端 Agent 配置</h3>
              </div>
              <p className="text-xs text-[var(--text-muted)] ml-7">
                请复制下方命令，在源 Agent 终端或 IM 机器人对话框中执行。
              </p>
              {!migrationCommandReady ? (
                <div className="ml-7 bg-[var(--bg-grey-normal)] border border-[var(--border)] rounded-[var(--radius-lg)] p-6 flex flex-col items-center justify-center gap-2">
                  <Loader2 className="w-5 h-5 text-[var(--text-brand)] animate-spin" />
                  <p className="text-xs text-[var(--text-muted)]">正在生成迁移命令...</p>
                  <p className="text-xs text-[var(--text-weak)]">正在获取临时上传凭证和 COS 预签名链接</p>
                </div>
              ) : (
                <div className="ml-7 relative bg-[var(--bg-grey-normal)] border border-[var(--border)] rounded-[var(--radius-lg)] p-3">
                  <button
                    onClick={() => { navigator.clipboard.writeText(migrationExportCommand); toast.success("命令已复制"); }}
                    className="absolute top-2 right-2 p-1.5 rounded-[var(--radius-lg)] hover:bg-[var(--alert-info-bg)] text-[var(--text-muted)] hover:text-[var(--text-brand)] transition-colors"
                    title="复制命令"
                  >
                    <Copy className="w-3.5 h-3.5" />
                  </button>
                  <pre className="text-xs text-[var(--text-body)] font-mono whitespace-pre-wrap break-all leading-relaxed pr-8">{migrationExportCommand}</pre>
                </div>
              )}
              <div className="ml-7 text-xs text-[var(--text-weak)] space-y-0.5">
                <p className="flex items-center gap-1"><Clock className="w-3 h-3" /> 上传链接有效期 1 小时，超时请刷新页面重新获取</p>
              </div>
            </div>

            {/* Step 2: 检测上传 & 导入 */}
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <div className={`w-5 h-5 rounded-full flex items-center justify-center text-xs font-bold ${
                  migrationStep === "export" || migrationStep === "waitUpload" ? "border-2 border-[var(--border)] bg-white text-[var(--text-weak)]"
                  : migrationStep === "import" ? "bg-[var(--text-brand)] text-white"
                  : "bg-[var(--text-success)] text-white"
                }`}>
                  {migrationStep === "success" || migrationStep === "importing" ? <Check className="w-3 h-3" strokeWidth={3} /> : "2"}
                </div>
                <h3 className="text-sm font-semibold text-[var(--text-title)]">将源端配置导入当前实例</h3>
              </div>

              {!migrationUploaded && (migrationStep === "export" || migrationStep === "waitUpload") && (
                <div className="ml-7 space-y-2.5">
                  <p className="text-xs text-[var(--text-muted)]">执行完导出命令后，点击检测上传状态：</p>
                  <Button
                    variant="tenant-primary"
                    size="claw-sm"
                    onClick={handleCheckUpload}
                    disabled={migrationChecking || !migrationCommandReady}
                    className="text-xs"
                  >
                    {migrationChecking ? <Loader2 className="w-3 h-3 animate-spin" /> : <Search className="w-3 h-3" />}
                    {migrationChecking ? "检测中..." : migrationCheckFailed ? "重新检测" : "检测上传状态"}
                  </Button>
                  {migrationCheckFailed && (
                    <Alert variant="error">
                      <XCircle className="w-4 h-4" />
                      <AlertTitle>未检测到数据包</AlertTitle>
                      <AlertDescription>
                        <ul className="text-xs list-disc pl-4 space-y-0.5 leading-relaxed">
                          <li>请确认已在源端执行完导出命令，且命令输出包含 "✅ 导出完成"</li>
                          <li>检查源端网络是否正常，curl 上传是否报错</li>
                          <li>上传链接有效期 1 小时，超时请关闭弹窗重新打开获取新链接</li>
                        </ul>
                      </AlertDescription>
                    </Alert>
                  )}
                </div>
              )}

              {migrationStep === "import" && (
                <div className="ml-7 space-y-3">
                  <div className="flex items-center gap-2 px-3 py-2.5 rounded-[var(--radius-lg)] bg-[var(--alert-success-bg)] border border-[var(--text-success)]/20">
                    <CheckCircle2 className="w-4 h-4 text-[var(--text-success)] shrink-0" />
                    <span className="text-xs font-medium text-[var(--text-success)]">已检测到上传的数据包</span>
                  </div>
                  <Alert variant="warning">
                    <CircleAlert className="w-4 h-4" />
                    <AlertTitle>重要提醒</AlertTitle>
                    <AlertDescription>
                      执行导入将覆盖当前实例的全部 Agent 配置（~/.agent/ 目录）。
                      导入前会自动备份，失败时自动回滚。
                    </AlertDescription>
                  </Alert>
                  <Button
                    variant="tenant-primary"
                    size="claw-sm"
                    onClick={handleStartMigration}
                    className="text-xs"
                  >
                    <ArrowLeftRight className="w-3.5 h-3.5" />
                    导入并重启 Agent
                  </Button>
                </div>
              )}

              {migrationStep === "importing" && (
                <div className="ml-7 space-y-3">
                  <p className="text-xs text-[var(--text-brand)] flex items-center gap-1.5 font-medium">
                    <Loader2 className="w-3.5 h-3.5 animate-spin" /> 正在执行导入...
                  </p>
                  <div className="space-y-1.5">
                    {importSteps.map((step, i) => (
                      <div key={i} className={`flex items-center gap-2 px-3 py-1.5 rounded-[var(--radius-lg)] text-xs ${
                        step.status === "done" ? "bg-[var(--alert-success-bg)]" :
                        step.status === "running" ? "bg-[var(--alert-info-bg)]" :
                        step.status === "failed" ? "bg-[var(--alert-error-bg)]" : "bg-[var(--muted)]"
                      }`}>
                        {step.status === "done" && <CheckCircle2 className="w-3.5 h-3.5 text-[var(--text-success)] shrink-0" />}
                        {step.status === "running" && <Loader2 className="w-3.5 h-3.5 text-[var(--text-brand)] animate-spin shrink-0" />}
                        {step.status === "failed" && <XCircle className="w-3.5 h-3.5 text-[var(--text-danger)] shrink-0" />}
                        {step.status === "pending" && <div className="w-3.5 h-3.5 rounded-full border-2 border-[var(--border)] shrink-0" />}
                        <span className={
                          step.status === "done" ? "text-[var(--text-success)]" :
                          step.status === "running" ? "text-[var(--text-brand)] font-medium" :
                          step.status === "failed" ? "text-[var(--text-danger)]" : "text-[var(--text-weak)]"
                        }>
                          {step.label}
                        </span>
                        {step.status === "done" && <span className="text-[var(--text-success)] ml-auto">✓</span>}
                        {step.status === "running" && <span className="text-[var(--text-brand)]/60 ml-auto">进行中...</span>}
                        {step.status === "failed" && step.error && <span className="text-[var(--text-danger)] ml-auto">{step.error}</span>}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Step 3: Result */}
            {migrationStep === "success" && (
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <div className="w-5 h-5 rounded-full bg-[var(--text-success)] text-white flex items-center justify-center">
                    <Check className="w-3 h-3" strokeWidth={3} />
                  </div>
                  <h3 className="text-sm font-semibold text-[var(--text-success)]">
                    {verifyResults.every((r) => r.passed) ? "迁移成功，已验证生效" : "迁移完成，部分项需处理"}
                  </h3>
                </div>
                {verifyResults.length > 0 && (
                  <div className="ml-7 space-y-1.5">
                    <p className="text-xs text-[var(--text-muted)] font-medium">导入后验证：</p>
                    {verifyResults.map((v, i) => (
                      <div key={i} className={`flex items-center gap-2 px-3 py-1.5 rounded-[var(--radius-lg)] text-xs ${v.passed ? "bg-[var(--alert-success-bg)]" : "bg-[var(--alert-warning-bg)]"}`}>
                        {v.passed
                          ? <CheckCircle2 className="w-3.5 h-3.5 text-[var(--text-success)] shrink-0" />
                          : <CircleAlert className="w-3.5 h-3.5 text-[var(--text-warning)] shrink-0" />}
                        <span className={v.passed ? "text-[var(--text-success)]" : "text-[var(--text-warning)]"}>{v.label}</span>
                        <code className="text-[var(--text-weak)] font-mono ml-1">{v.cmd}</code>
                        <span className={`ml-auto ${v.passed ? "text-[var(--text-success)]" : "text-[var(--text-warning)]"}`}>{v.detail}</span>
                      </div>
                    ))}
                  </div>
                )}
                <DialogFooter className="mt-4">
                  <Button variant="tenant-outline" size="claw-sm" onClick={() => setMigrationOpen(false)}>
                    关闭
                  </Button>
                </DialogFooter>
              </div>
            )}

            {/* 失败状态 */}
            {migrationStep === "failed" && (
              <div className="space-y-3">
                <Alert variant="error">
                  <XCircle className="w-4 h-4" />
                  <AlertTitle>迁移失败</AlertTitle>
                  <AlertDescription>{migrationError}</AlertDescription>
                </Alert>
                <DialogFooter>
                  <Button
                    variant="tenant-outline"
                    size="claw-sm"
                    onClick={() => { setMigrationStep("export"); setMigrationCommandReady(false); setTimeout(() => setMigrationCommandReady(true), 1800); }}
                  >
                    重新开始
                  </Button>
                </DialogFooter>
              </div>
            )}
          </DialogBody>
        </DialogContent>
      </Dialog>

      {/* ===== 数据备份 信息弹窗（第一步） ===== */}
      <Dialog open={showBackupInfoDialog} onOpenChange={setShowBackupInfoDialog}>
        <DialogContent className="sm:max-w-fit">
          <DialogHeader>
            <DialogTitle>数据备份</DialogTitle>
            <DialogDescription className="sr-only">数据备份信息</DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="text-sm text-[var(--text-emphasis)] leading-relaxed space-y-4 min-w-[420px]">
              {/* 当前备份点 */}
              <div>
                <h4 className="text-sm font-medium text-[var(--text-emphasis)] mb-2">当前备份点</h4>
                <div className="rounded-lg border border-[var(--border)] bg-[var(--bg-subtle)] p-3 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-[var(--text-muted)]">备份时间</span>
                    <span className="font-mono text-[var(--text-emphasis)] font-medium">
                      {backupTriggeredAt}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-[var(--text-muted)]">触发操作</span>
                    <span className="text-[var(--text-emphasis)] font-medium">
                      {backupTriggerType === "update" ? "一键更新" : "重装"}
                    </span>
                  </div>
                </div>
              </div>

              {/* 备份说明 */}
              <div>
                <h4 className="text-sm font-medium text-[var(--text-emphasis)] mb-2">备份说明</h4>
                <p className="text-xs text-[var(--text-muted)] leading-relaxed">
                  系统将在「一键更新」或「重装」时自动生成备份点，仅保留最近一份。
                </p>
              </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setShowBackupInfoDialog(false)}
            >
              取消
            </Button>
            <Button
              variant="tenant-primary"
              size="claw-sm"
              onClick={openBackupRollbackConfirm}
            >
              回滚到此备份点
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 数据备份 回滚确认弹窗（第二步） ===== */}
      <Dialog
        open={showBackupConfirmDialog}
        onOpenChange={(open) => {
          if (!backupRollbackLoading) setShowBackupConfirmDialog(open);
        }}
      >
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>回滚确认</DialogTitle>
            <DialogDescription className="sr-only">回滚确认</DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="text-sm text-[var(--text-emphasis)] leading-relaxed space-y-3">
              <p>确定要回滚到以下备份点吗？</p>

              {/* 备份点信息卡片 */}
              <div className="rounded-lg border border-[var(--border)] bg-[var(--bg-subtle)] p-3 space-y-1.5">
                <div className="flex items-center justify-between">
                  <span className="text-[var(--text-muted)]">备份时间</span>
                  <span className="font-mono text-[var(--text-emphasis)] font-medium">
                    {backupTriggeredAt}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-[var(--text-muted)]">触发操作</span>
                  <span className="text-[var(--text-emphasis)] font-medium">
                    {backupTriggerType === "update" ? "一键更新" : "重装"}时自动备份
                  </span>
                </div>
              </div>

              {/* 回滚风险 */}
              <Alert variant="warning">
                <CircleAlert className="w-4 h-4" />
                <AlertTitle>回滚风险</AlertTitle>
                <AlertDescription>
                  <ul className="text-xs space-y-1.5 list-disc pl-4 leading-relaxed">
                    <li>回滚将 Agent 恢复至上述备份点状态，该备份点之后的数据、配置变更将不被保留，且不同步管控台最新配置（版本号、参数等）。</li>
                    <li>回滚期间 Agent 不可用，极端情况下回滚后可能无法恢复正常服务。</li>
                  </ul>
                </AlertDescription>
              </Alert>
              <p className="text-xs text-[var(--text-muted)]">建议回滚前自行保存 Agent 重要数据。</p>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setShowBackupConfirmDialog(false)}
              disabled={backupRollbackLoading}
            >
              取消
            </Button>
            <Button
              variant="tenant-destructive"
              size="claw-sm"
              onClick={confirmBackupRollback}
              disabled={backupRollbackLoading}
            >
              {backupRollbackLoading ? (
                <>
                  <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  正在回滚...
                </>
              ) : (
                "确认回滚"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 一键更新 确认弹窗 ===== */}
      <Dialog open={showUpdateConfirmDialog} onOpenChange={setShowUpdateConfirmDialog}>
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>更新确认</DialogTitle>
            <DialogDescription className="sr-only">更新确认</DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="text-sm text-[var(--text-emphasis)] leading-relaxed space-y-3">
              {/* 当前版本 vs 目标版本 */}
              <div className="rounded-lg border border-[var(--border)] p-3 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-[var(--text-muted)]">当前 Agent 版本</span>
                  <span className="font-mono text-[var(--text-emphasis)] font-medium">v{currentVersion}</span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-[var(--text-muted)]">目标 Agent 版本</span>
                  <span className="font-mono text-[var(--text-brand)] font-medium">
                    {targetVersion ? `v${targetVersion}` : "暂无可更新版本"}
                  </span>
                </div>
              </div>
              {/* 版本校验提示 */}
              {!targetVersion ? (
                <Alert variant="warning">
                  <AlertCircle className="w-4 h-4" />
                  <AlertDescription>
                    暂未找到该 Agent 类型的生效镜像版本，请联系管理员在镜像管理中配置。
                  </AlertDescription>
                </Alert>
              ) : !isTargetNewer ? (
                <Alert variant="warning">
                  <AlertCircle className="w-4 h-4" />
                  <AlertDescription>
                    目标版本（v{targetVersion}）不比当前版本（v{currentVersion}）更新，不支持向低版本更新。
                  </AlertDescription>
                </Alert>
              ) : null}
              <p>Agent 版本将会更新至管理员指定生效镜像所对应的版本，且不支持跨 Agent 类型更新。</p>
              <p>更新版本预计需要 5～10 分钟不等，请您耐心等待。更新期间 Agent 网关服务暂停，面板不可操作。</p>
              <p>更新版本后模型（Models）、通道（Channels）、技能（Skills）和记忆均不会丢失。</p>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setShowUpdateConfirmDialog(false)}
            >
              取消
            </Button>
            <Tooltip>
              <TooltipTrigger asChild>
                <span tabIndex={!isTargetNewer || !targetVersion ? 0 : -1}>
                  <Button
                    variant="tenant-primary"
                    size="claw-sm"
                    onClick={handleStartUpdate}
                    disabled={!isTargetNewer || !targetVersion}
                  >
                    确认更新
                  </Button>
                </span>
              </TooltipTrigger>
              {(!targetVersion || !isTargetNewer) && (
                <TooltipContent side="bottom" className="text-xs">
                  {!targetVersion
                    ? "暂无可更新的目标版本"
                    : `目标版本 v${targetVersion} 不比当前版本 v${currentVersion} 更新`}
                </TooltipContent>
              )}
            </Tooltip>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 一键更新 进度弹窗 ===== */}
      <Dialog
        open={showUpdateProgressDialog}
        onOpenChange={(open) => {
          if (!open) setShowUpdateProgressDialog(false);
        }}
      >
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>正在更新 Agent</DialogTitle>
            <DialogDescription className="sr-only">更新进度</DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6 pb-6">
            <div className="space-y-2.5">
              {updateSteps.map((step, idx) => {
                const stepNum = idx + 1;
                const isDone = updateStepsDone >= stepNum;
                const isActive = updateStepsDone === idx;
                return (
                  <div key={step} className="flex items-center gap-3">
                    {isDone ? (
                      <CheckCircle2 className="w-5 h-5 text-[var(--text-success)] shrink-0" />
                    ) : isActive ? (
                      <Loader2 className="w-5 h-5 text-[var(--text-brand)] animate-spin shrink-0" />
                    ) : (
                      <div className="w-5 h-5 rounded-full border-2 border-[var(--border-control)] shrink-0" />
                    )}
                    <span
                      className={`text-xs ${
                        isDone
                          ? "text-[var(--text-secondary)]"
                          : isActive
                          ? "text-[var(--text-brand)] font-medium"
                          : "text-[var(--text-muted)]"
                      }`}
                    >
                      [步骤{stepNum}] {step}
                    </span>
                  </div>
                );
              })}
            </div>
          </DialogBody>
        </DialogContent>
      </Dialog>

      {/* ===== Hermes 0.18.0 登录信息弹窗 ===== */}
      <Dialog open={showHermesPanelDialog} onOpenChange={setShowHermesPanelDialog}>
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>打开 Agent 面板</DialogTitle>
            <DialogDescription className="sr-only">
              查看 Hermes Agent 0.18.0 面板登录账号和密码
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            <Alert variant="info" className="mb-4">
              <AlertInfoIcon />
              <AlertDescription>
                Hermes 面板需要独立登录。若浏览器已保留登录状态，可直接打开；需要登录时再复制下方账号和密码。
              </AlertDescription>
            </Alert>
            <div className="rounded-[var(--radius-card)] border border-[var(--border)] overflow-hidden">
              <div className="flex items-center gap-3 px-4 py-3">
                <span className="w-16 shrink-0 text-xs text-[var(--text-secondary)]">面板链接</span>
                <a
                  href={DEMO_OPENCLAW_PANEL_URL}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex-1 min-w-0 truncate text-xs text-[var(--text-emphasis)] font-mono hover:underline"
                >
                  {DEMO_OPENCLAW_PANEL_URL}
                </a>
                <button
                  onClick={() => copyHermesCredential("面板链接", DEMO_OPENCLAW_PANEL_URL)}
                  aria-label="复制面板链接"
                  className="p-1.5 rounded-[var(--radius-lg)] hover:bg-[var(--cp-brand-blue-soft,#EFF6FF)] text-[var(--text-muted)] hover:text-[var(--text-brand)] transition-colors"
                >
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
              <div className="h-px bg-[var(--border)]" />
              <div className="flex items-center gap-3 px-4 py-3">
                <span className="w-16 shrink-0 text-xs text-[var(--text-secondary)]">登录账号</span>
                <code className="min-w-0 flex-1 select-all truncate text-xs text-[var(--text-emphasis)]">
                  {HERMES_018_DEMO_CREDENTIALS.username}
                </code>
                <button
                  onClick={() => copyHermesCredential("登录账号", HERMES_018_DEMO_CREDENTIALS.username)}
                  aria-label="复制登录账号"
                  className="p-1.5 rounded-[var(--radius-lg)] hover:bg-[var(--cp-brand-blue-soft,#EFF6FF)] text-[var(--text-muted)] hover:text-[var(--text-brand)] transition-colors"
                >
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
              <div className="h-px bg-[var(--border)]" />
              <div className="flex items-center gap-3 px-4 py-3">
                <span className="w-16 shrink-0 text-xs text-[var(--text-secondary)]">登录密码</span>
                <code className="min-w-0 flex-1 select-all truncate text-xs text-[var(--text-emphasis)]">
                  {showHermesPassword ? HERMES_018_DEMO_CREDENTIALS.password : "••••••••••"}
                </code>
                <button
                  onClick={() => setShowHermesPassword((visible) => !visible)}
                  aria-label={showHermesPassword ? "隐藏登录密码" : "显示登录密码"}
                  title={showHermesPassword ? "隐藏密码" : "显示密码"}
                  className="p-1.5 rounded-[var(--radius-lg)] hover:bg-[var(--cp-brand-blue-soft,#EFF6FF)] text-[var(--text-muted)] hover:text-[var(--text-brand)] transition-colors"
                >
                  {showHermesPassword ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                </button>
                <button
                  onClick={() => copyHermesCredential("登录密码", HERMES_018_DEMO_CREDENTIALS.password)}
                  aria-label="复制登录密码"
                  className="p-1.5 rounded-[var(--radius-lg)] hover:bg-[var(--cp-brand-blue-soft,#EFF6FF)] text-[var(--text-muted)] hover:text-[var(--text-brand)] transition-colors"
                >
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          </DialogBody>
          <DialogFooter className="gap-3">
            <Button variant="tenant-outline" onClick={() => setShowHermesPanelDialog(false)}>
              取消
            </Button>
            <Button variant="tenant-primary" onClick={openHermesPanel}>
              开启面板
              <ExternalLink className="h-3.5 w-3.5" />
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== WebUI 进度弹窗 ===== */}
      <Dialog open={showWebUIProgressDialog} onOpenChange={(open) => { if (!open) setShowWebUIProgressDialog(false); }}>
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>开启Agent面板</DialogTitle>
            <DialogDescription className="sr-only">开启Agent面板</DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            <Alert variant="info" className="mb-2">
              <AlertInfoIcon />
              <AlertDescription>
                Agent 面板（WebUI）是官方提供的浏览器操作界面，可直接在浏览器与 AI 对话，并且有查看会话记录、配置定时任务、监控系统日志等高级功能。
              </AlertDescription>
            </Alert>
            <p className="text-sm text-[var(--text-secondary)] mb-3">开启Agent面板将会依次执行以下操作，确定后将自动执行：</p>
            <div className="space-y-2.5">
              <div className="flex items-center gap-3">
                {webUIFailedStep === "port" ? (
                  <AlertCircle className="w-5 h-5 text-[var(--text-warning)] shrink-0" />
                ) : webUIStep >= 1 ? (
                  <CheckCircle2 className="w-5 h-5 text-[var(--text-success)] shrink-0" />
                ) : (
                  <Loader2 className="w-5 h-5 text-[var(--text-brand)] animate-spin shrink-0" />
                )}
                <span className={`text-xs ${
                  webUIFailedStep === "port" ? "text-[var(--text-warning)] font-medium" :
                  webUIStep >= 1 ? "text-[var(--text-secondary)]" : "text-[var(--text-brand)] font-medium"
                }`}>
                  {webUIFailedStep === "port"
                    ? "放通端口：放通端口失败，请重试"
                    : webUIStep >= 1
                    ? "放通端口：端口38341已放通"
                    : "放通端口：正在放通端口38341...预计1~2秒"}
                </span>
              </div>
              <div className="flex items-center gap-3">
                {webUIFailedStep === "link" ? (
                  <AlertCircle className="w-5 h-5 text-[var(--text-warning)] shrink-0" />
                ) : webUIStep >= 2 ? (
                  <CheckCircle2 className="w-5 h-5 text-[var(--text-success)] shrink-0" />
                ) : webUIStep === 1 ? (
                  <Loader2 className="w-5 h-5 text-[var(--text-brand)] animate-spin shrink-0" />
                ) : (
                  <div className="w-5 h-5 rounded-full border-2 border-[var(--border)] shrink-0" />
                )}
                <span className={`text-xs ${
                  webUIFailedStep === "link" ? "text-[var(--text-warning)] font-medium" :
                  webUIStep >= 2 ? "text-[var(--text-secondary)]" :
                  webUIStep === 1 ? "text-[var(--text-brand)] font-medium" : "text-[var(--text-muted)]"
                }`}>
                  {webUIFailedStep === "link"
                    ? "生成链接：生成链接失败，请重试"
                    : webUIStep >= 2
                    ? "生成链接：链接已生成"
                    : webUIStep === 1
                    ? "生成链接：正在为您生成Agent面板访问链接，预计5~10秒..."
                    : "生成链接：等待放通端口完成"}
                </span>
              </div>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="tenant-outline" size="claw-sm" onClick={() => setShowWebUIProgressDialog(false)}>
              取消
            </Button>
            <Button
              variant="tenant-primary"
              size="claw-sm"
              disabled={webUIStep < 2 && webUIFailedStep === "none"}
              onClick={webUIFailedStep !== "none" ? handleWebUIRetry : handleWebUIProgressConfirm}
            >
              {webUIFailedStep !== "none" ? "重试" : "确定"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== WebUI 结果弹窗 ===== */}
      <Dialog open={showWebUIResultDialog} onOpenChange={(open) => { if (!open) setShowWebUIResultDialog(false); }}>
        <DialogContent size="lg">
          <DialogHeader>
            <DialogTitle className="text-base font-bold text-foreground">开启面板</DialogTitle>
          </DialogHeader>
          <div className="min-w-0 py-2 space-y-5">
            <Alert variant="info">
              <AlertInfoIcon />
              <AlertDescription>
                访问链接已生成，该链接含有您的 API Key 和加密配置，请勿分享给第三方，以防隐私泄露或资产损失。
              </AlertDescription>
            </Alert>
            <div className="min-w-0 rounded-[var(--radius-card)] border border-[var(--border)] overflow-hidden">
              <div className="flex min-w-0 items-center gap-3 px-4 py-3">
                <span className="text-sm text-muted-foreground w-24 shrink-0">WebSocket URL</span>
                <a
                  href={webUIUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="min-w-0 flex-1 text-sm text-foreground font-mono truncate hover:underline cursor-pointer"
                >
                  {webUIUrl}
                </a>
                <button
                  aria-label="复制 WebSocket URL"
                  className="p-1.5 rounded-[var(--radius-lg)] hover:bg-[var(--alert-info-bg)] text-[var(--muted-foreground)] hover:text-[var(--text-brand)] transition-colors"
                  onClick={() => { navigator.clipboard.writeText(webUIUrl); toast.success("已复制链接"); }}
                >
                  <Copy className="w-4 h-4" />
                </button>
              </div>
              <div className="h-px bg-[var(--border)]" />
              <div className="flex min-w-0 items-center gap-3 px-4 py-3">
                <span className="text-sm text-muted-foreground w-24 shrink-0">网关令牌</span>
                <span className="min-w-0 flex-1 text-sm text-foreground font-mono truncate">{webUIToken}</span>
                <button
                  aria-label="复制网关令牌"
                  className="p-1.5 rounded-[var(--radius-lg)] hover:bg-[var(--alert-info-bg)] text-[var(--muted-foreground)] hover:text-[var(--text-brand)] transition-colors"
                  onClick={() => { navigator.clipboard.writeText(webUIToken); toast.success("已复制Token"); }}
                >
                  <Copy className="w-4 h-4" />
                </button>
              </div>
            </div>
            <p className="text-sm text-muted-foreground leading-relaxed">
              用浏览器打开 WebSocket URL，如面板需要填入网关令牌，则将网关令牌复制并粘贴过去，即可进入面板。
            </p>
          </div>
          <DialogFooter className="gap-3">
            <Button variant="tenant-outline" onClick={() => setShowWebUIResultDialog(false)}>
              关闭
            </Button>
            <Button variant="tenant-primary" onClick={() => window.open(webUIUrl, "_blank", "noopener,noreferrer")}>
              立即访问
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 飞书授权弹窗（4阶段） ===== */}
      <Dialog open={showQrModal} onOpenChange={(open) => {
        if (!open && (feishuModalStage === "done" || feishuModalStage === "loading" || feishuModalStage === "qr")) {
          setShowQrModal(false);
        }
      }}>
        <DialogContent size="md">
          {(feishuModalStage === "loading" || feishuModalStage === "qr") && (
            <>
              <DialogHeader>
                <DialogTitle>扫码配置飞书机器人</DialogTitle>
                <DialogDescription className="text-sm text-[var(--text-secondary)] mt-1">
                  请使用飞书账号扫码登录，完成授权后将自动为您创建机器人。
                </DialogDescription>
              </DialogHeader>
              <DialogBody className="px-6 pb-6">
                <div className="flex flex-col items-center justify-center bg-[var(--bg-grey-normal)] rounded-[var(--radius-card)] min-h-[240px]">
                  {feishuModalStage === "loading" ? (
                    <>
                      <Loader2 className="w-12 h-12 text-[var(--text-muted)] animate-spin mb-4" />
                      <p className="text-sm text-[var(--text-secondary)]">正在生成二维码...</p>
                    </>
                  ) : (
                    <div className="w-[180px] h-[180px] bg-white border border-[var(--border)] rounded flex items-center justify-center">
                      <span className="text-xs text-[var(--text-muted)]">[ 飞书二维码占位 ]</span>
                    </div>
                  )}
                </div>
              </DialogBody>
            </>
          )}

          {feishuModalStage === "configuring" && (
            <>
              <DialogHeader>
                <DialogTitle>正在配置飞书机器人</DialogTitle>
              </DialogHeader>
              <DialogBody className="px-6 pb-6">
                <div className="space-y-2.5">
                  {feishuSteps.map((step, idx) => {
                    const stepNum = idx + 1;
                    const isDone = feishuStepsDone >= stepNum;
                    const isActive = feishuStepsDone === idx;
                    const isHighPrivilege = idx === feishuHighPrivilegeStepIdx;
                    return (
                      <div key={step} className="flex items-center gap-3">
                        {isDone && isHighPrivilege ? (
                          <AlertCircle className="w-5 h-5 text-[var(--text-warning)] shrink-0" />
                        ) : isDone ? (
                          <CheckCircle2 className="w-5 h-5 text-[var(--text-success)] shrink-0" />
                        ) : isActive ? (
                          <Loader2 className="w-5 h-5 text-[var(--text-brand)] animate-spin shrink-0" />
                        ) : (
                          <div className="w-5 h-5 rounded-full border-2 border-[var(--border)] shrink-0" />
                        )}
                        <span className={`text-xs ${
                          isDone && isHighPrivilege ? "text-[var(--text-warning)] font-medium" :
                          isDone ? "text-[var(--text-secondary)]" : isActive ? "text-[var(--text-brand)] font-medium" : "text-[var(--text-muted)]"
                        }`}>
                          [步骤{stepNum}] {step}
                        </span>
                      </div>
                    );
                  })}
                </div>
              </DialogBody>
            </>
          )}

          {feishuModalStage === "failed" && (
            <>
              <DialogHeader>
                <div className="flex items-center gap-3 mb-1">
                  <CircleAlert className="w-5 h-5 text-[var(--text-danger)] shrink-0" />
                  <DialogTitle>飞书机器人发布失败</DialogTitle>
                </div>
                <DialogDescription className="text-sm text-[var(--text-danger)] mt-1 font-medium">
                  当前用户权限无法免审批发布飞书机器人，请联系管理员审批通过后再进行手动配置。
                </DialogDescription>
              </DialogHeader>
              <DialogBody className="px-6">
                <div className="space-y-1.5 text-sm bg-[var(--bg-grey-normal)] rounded-[var(--radius-card)] p-3 border border-[var(--border)]">
                  <div className="flex items-center gap-2">
                    <span className="text-[var(--text-secondary)] shrink-0">机器人名称：</span>
                    <span className="text-[var(--text-emphasis)] font-medium">Agent机器人-8791</span>
                  </div>
                  <div className="flex items-start gap-2">
                    <span className="text-[var(--text-secondary)] shrink-0">管理地址：</span>
                    <a href="https://open.feishu.cn/app/cli_a933983f95385cca" target="_blank" rel="noopener noreferrer" className="text-[var(--text-brand)] hover:underline break-all">
                      https://open.feishu.cn/app/cli_a933983f95385cca
                    </a>
                  </div>
                </div>
              </DialogBody>
              <DialogFooter>
                <Button variant="tenant-primary" size="claw-sm" onClick={() => setShowQrModal(false)}>完成</Button>
              </DialogFooter>
            </>
          )}

          {feishuModalStage === "done" && (
            <>
              <DialogHeader>
                <div className="flex items-center gap-3 mb-1">
                  <CheckCircle2 className="w-8 h-8 text-[var(--text-success)] shrink-0" />
                  <DialogTitle>飞书机器人授权配置成功</DialogTitle>
                </div>
              </DialogHeader>
              <DialogBody className="px-6">
                <div className="space-y-1.5 text-sm bg-[var(--bg-grey-normal)] rounded-[var(--radius-card)] p-3 border border-[var(--border)]">
                  <div className="flex items-center gap-2">
                    <span className="text-[var(--text-secondary)] shrink-0">机器人名称：</span>
                    <span className="text-[var(--text-emphasis)] font-medium">Agent机器人-4598</span>
                  </div>
                  <div className="flex items-start gap-2">
                    <span className="text-[var(--text-secondary)] shrink-0">管理地址：</span>
                    <a href="https://open.feishu.cn/app/cli_a9317ee80379dbc2" target="_blank" rel="noopener noreferrer" className="text-[var(--text-brand)] hover:underline break-all">
                      https://open.feishu.cn/app/cli_a9317ee80379dbc2
                    </a>
                  </div>
                </div>
                <Alert variant="warning" className="mt-4">
                  <CircleAlert className="w-4 h-4" />
                  <AlertDescription>
                    <p className="font-medium mb-2">以下高级权限无法免审批发布，已自动为您提交申请：</p>
                    <ol className="ml-4 space-y-1 list-decimal">
                      <li>查看、评论和下载云空间中所有文件</li>
                      <li>查看、评论、编辑和管理云空间中所有文件</li>
                    </ol>
                    <p className="mt-2">如需启用，请联系管理员前往审批：</p>
                    <a href="https://feishu.cn/admin/appCenter/audit" target="_blank" rel="noopener noreferrer" className="text-[var(--text-brand)] hover:underline block">
                      https://feishu.cn/admin/appCenter/audit
                    </a>
                  </AlertDescription>
                </Alert>
              </DialogBody>
              <DialogFooter>
                <Button variant="tenant-primary" size="claw-sm" onClick={() => setShowQrModal(false)}>完成</Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>

      {/* ===== 微信扫码弹窗 ===== */}
      <Dialog open={showWechatQrModal} onOpenChange={(open) => {
        if (!open && wechatModalStage === "qr") setShowWechatQrModal(false);
      }}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>微信扫码登录</DialogTitle>
            <DialogDescription className="text-sm text-[var(--text-secondary)] mt-1">
              使用微信（需要 iOS、Android系统 8.0.70 以上版本）"扫一扫"完成接入
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6 pb-6">
            <div className="flex flex-col items-center justify-center bg-[var(--bg-grey-normal)] rounded-[var(--radius-card)] min-h-[220px]">
              {wechatModalStage === "checking" && (
                <>
                  <Loader2 className="w-10 h-10 text-[var(--text-muted)] animate-spin mb-3" />
                  <p className="text-sm text-[var(--text-secondary)]">正在检查网关…</p>
                </>
              )}
              {wechatModalStage === "generating" && (
                <>
                  <Loader2 className="w-10 h-10 text-[var(--text-muted)] animate-spin mb-3" />
                  <p className="text-sm text-[var(--text-secondary)]">正在生成二维码…</p>
                </>
              )}
              {wechatModalStage === "qr" && (
                <div className="w-[180px] h-[180px] bg-white border border-[var(--border)] rounded flex items-center justify-center">
                  <span className="text-xs text-[var(--text-muted)]">[ 微信二维码占位 ]</span>
                </div>
              )}
            </div>
          </DialogBody>
        </DialogContent>
      </Dialog>

      {/* ===== WhatsApp Pairing Code 弹窗 ===== */}
      <Dialog
        open={showWhatsAppPairing}
        onOpenChange={(open) => {
          if (open) return;
          // 配对完成前关闭（点击遮罩/右上角 X/Esc）视为取消配对；配对成功后关闭视为完成
          if (whatsappPairingStage === "success") handleFinishWhatsAppPairing();
          else handleCancelWhatsAppPairing();
        }}
      >
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>WhatsApp 配对</DialogTitle>
            <DialogDescription className="text-sm text-[var(--text-secondary)] mt-1">
              {whatsappPairingStage === "success"
                ? "已成功关联，通道已开始运行"
                : "请在 WhatsApp 中输入下方配对码完成关联"}
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6 pb-2 space-y-4">
            {whatsappPairingStage === "pairing" ? (
              <>
                {/* 配对码 */}
                <div className="flex flex-col items-center justify-center bg-[var(--bg-grey-normal)] rounded-[var(--radius-card)] py-6">
                  <span className="text-xs text-[var(--text-muted)] mb-2">配对码 Pairing Code</span>
                  <span className="text-2xl font-mono font-semibold tracking-[0.3em] text-[var(--text-primary)] select-all">
                    {whatsappPairingCode}
                  </span>
                </div>
                {/* 操作步骤 */}
                <div className="text-sm text-[var(--text-secondary)] leading-relaxed">
                  <p className="mb-1 text-[var(--text-primary)] font-medium">操作步骤：</p>
                  <ol className="list-decimal pl-5 space-y-1">
                    <li>Navigate to “profile”</li>
                    <li>Select Linked devices</li>
                    <li>Click “link a device”</li>
                    <li>Select “Link with phone number instead”</li>
                    <li>Enter code as provided from ClawPro Console</li>
                  </ol>
                </div>
              </>
            ) : (
              // 配对成功态：绿色对勾 + 成功文案
              <div className="flex flex-col items-center justify-center bg-[var(--bg-grey-normal)] rounded-[var(--radius-card)] min-h-[220px]">
                <CheckCircle2 className="w-14 h-14 text-[#16A34A] mb-3" />
                <p className="text-base font-medium text-[#16A34A]">WhatsApp 接入配置成功！</p>
              </div>
            )}
          </DialogBody>
          {whatsappPairingStage === "success" && (
            <DialogFooter className="justify-center">
              <Button
                variant="tenant-primary"
                size="claw"
                onClick={handleFinishWhatsAppPairing}
              >
                完成
              </Button>
            </DialogFooter>
          )}
        </DialogContent>
      </Dialog>

      {/* ===== 模型操作二次确认弹窗 ===== */}
      <Dialog
        open={modelConfirmDialog.open}
        onOpenChange={(open) => !open && setModelConfirmDialog(prev => ({ ...prev, open: false }))}
      >
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-brand)]">
              {modelConfirmDialog.type === "delete" ? "确认删除主模型" : modelConfirmDialog.type === "delete-backup" ? "确认删除备选模型" : "切换主模型"}
            </DialogTitle>
            <DialogDescription className="text-[var(--text-secondary)] leading-relaxed pt-1">
              {modelConfirmDialog.type === "delete"
                ? "删除后将自动切换备选模型作为主模型，切换过程中将导致相关的 Gateway 服务重启"
                : modelConfirmDialog.type === "delete-backup"
                ? "删除后将导致相关的 Gateway 服务重启，确认删除么"
                : "将此模型设为主模型后，原主模型将降为备选模型。切换过程中会自动重启 Gateway 服务，是否继续？"}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="tenant-outline" size="claw-sm" onClick={() => setModelConfirmDialog(prev => ({ ...prev, open: false }))}>
              取消
            </Button>
            <Button
              variant={modelConfirmDialog.type === "delete" || modelConfirmDialog.type === "delete-backup" ? "tenant-destructive" : "tenant-primary"}
              size="claw-sm"
              onClick={() => {
                if (modelConfirmDialog.type === "delete" || modelConfirmDialog.type === "delete-backup") {
                  // 真正执行删除（此前这里只弹 toast + 关弹窗，从未修改 appliedModels，是个空壳按钮）
                  handleConfirmDeleteModel();
                } else {
                  setModelConfirmDialog(prev => ({ ...prev, open: false }));
                  toast.success("已设为主模型");
                }
              }}
            >
              {modelConfirmDialog.type === "delete" || modelConfirmDialog.type === "delete-backup" ? "确认删除" : "确认设置"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 连通性检测失败弹窗 ===== */}
      <Dialog open={!!connectFailResult} onOpenChange={() => setConnectFailResult(null)}>
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-[var(--text-danger)]">
              <CircleAlert className="w-5 h-5" />
              模型连接失败
            </DialogTitle>
            <DialogDescription className="sr-only">模型连接失败详情</DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            <pre className="rounded-[var(--radius-card)] bg-[var(--bg-grey-normal)] border border-[var(--border)] p-3 text-xs text-[var(--text-emphasis)] font-mono whitespace-pre-wrap break-all">
              {connectFailResult}
            </pre>
          </DialogBody>
          <DialogFooter>
            <Button variant="tenant-primary" size="claw-sm" onClick={() => setConnectFailResult(null)}>我知道了</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 技能安装确认弹窗 ===== */}
      <Dialog
        open={skillInstallConfirm.open}
        onOpenChange={(open) => !open && setSkillInstallConfirm(prev => ({ ...prev, open: false }))}
      >
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-brand)]">确认安装技能</DialogTitle>
            <DialogDescription className="text-[var(--text-secondary)] leading-relaxed pt-1">
              确认安装名称为
              <span className="font-semibold text-[var(--text-emphasis)] mx-1">{skillInstallConfirm.skillName}</span>
              的技能？
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            <Alert variant="info">
              <AlertInfoIcon />
              <AlertDescription>
                管理员配置了
                <a href="https://skillhub.tencent.com/" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 text-[var(--text-brand)] hover:text-[var(--text-brand)] underline underline-offset-1 font-medium">
                  SkillHub地址
                  <ExternalLink className="w-3 h-3 flex-shrink-0" />
                </a>
                ，不支持模糊搜索，请输入准确Skill名称
              </AlertDescription>
            </Alert>
            <Alert variant="warning" className="mt-2">
              <AlertDescription>部分技能(Skills)可能存在安全风险，安装前请确认其安全性。</AlertDescription>
            </Alert>
          </DialogBody>
          <DialogFooter>
            <Button variant="tenant-outline" size="claw-sm" onClick={() => setSkillInstallConfirm(prev => ({ ...prev, open: false }))}>
              取消
            </Button>
            <Button
              variant="tenant-primary"
              size="claw-sm"
              onClick={() => {
                const name = skillInstallConfirm.skillName;
                setSkillInstallConfirm({ open: false, skillName: "" });
                setPendingSkills(prev => [
                  ...prev,
                  { id: `ps-${Date.now()}`, name, status: "pending" as const },
                ]);
                toast.success(`技能「${name}」已加入安装队列`);
              }}
            >
              确认安装
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 技能卸载二次确认弹窗 ===== */}
      <AlertDialog
        open={skillUninstallConfirm.open}
        onOpenChange={(open) => !open && setSkillUninstallConfirm({ open: false, name: "", version: "" })}
      >
        <AlertDialogContent className="max-w-sm">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-base font-semibold text-[var(--text-strong)]">卸载技能</AlertDialogTitle>
            <AlertDialogDescription className="text-sm text-[var(--text-muted)] leading-relaxed pt-1">
              确认卸载
              <span className="font-semibold text-[var(--text-strong)] mx-1">{skillUninstallConfirm.name}</span>
              ？卸载后该 Agent 将无法继续使用此技能。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <Button
              variant="tenant-outline"
              size="claw-sm"
              onClick={() => setSkillUninstallConfirm({ open: false, name: "", version: "" })}
            >
              取消
            </Button>
            <Button
              variant="tenant-destructive"
              size="claw-sm"
              onClick={handleConfirmUninstallSkill}
            >
              确认卸载
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ===== 编辑关联项目弹窗 ===== */}
      <Dialog open={projectEditOpen} onOpenChange={setProjectEditOpen}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>编辑关联项目</DialogTitle>
            <DialogDescription className="text-[var(--text-secondary)] leading-relaxed pt-1">
              可选，关联项目并保存后，会立刻安装所选项目配置的特定技能、规范等，取消关联项目不会移除已安装的技能、规范等。
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="px-6">
            {projectPool.length === 0 ? (
              <p className="text-sm text-[var(--text-weak)] py-6 text-center">暂无可关联的项目</p>
            ) : (
              <div className="flex flex-wrap gap-2 max-h-[280px] overflow-y-auto pl-0 pr-2 py-2">
                {projectPool.map((p) => {
                  const active = projectDraftIds.includes(p.id);
                  return (
                    <button
                      key={p.id}
                      type="button"
                      onClick={() => toggleProjectDraft(p.id)}
                      className={`relative flex items-center justify-center whitespace-nowrap h-6 px-3 rounded-full border text-xs font-medium transition-colors cursor-pointer ${
                        active
                          ? "bg-[var(--text-emphasis)] text-white border-[var(--text-emphasis)]"
                          : "border-[var(--border)] bg-[var(--card)] text-[var(--text-emphasis)] hover:bg-[var(--accent)] hover:border-[var(--border-control)]"
                      }`}
                    >
                      {p.name}
                      {active && (
                        <CheckCircle2 className="absolute -top-1.5 -right-1.5 w-3.5 h-3.5 fill-white text-[var(--text-emphasis)]" />
                      )}
                    </button>
                  );
                })}
              </div>
            )}
          </DialogBody>
          <DialogFooter>
            <Button variant="tenant-outline" size="claw-sm" onClick={() => setProjectEditOpen(false)}>
              取消
            </Button>
            <Button variant="tenant-primary" size="claw-sm" onClick={saveProjectEdit}>
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {/* ===== 龙虾医生 - 开始诊断 ===== */}
      <Dialog open={showStartModal} onOpenChange={(open) => { if (!open) setShowStartModal(false); }}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>开始诊断</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
            <p className="text-sm text-[var(--text-emphasis)] leading-relaxed mb-3">
              即将创建龙虾医生 Agent，对当前 Agent 进行全面检测和修复。
            </p>
            <div className="rounded-[var(--radius-card)] border border-[var(--border)] bg-white">
              <label className="flex items-start gap-2.5 cursor-pointer select-none px-3 py-2.5">
                <input
                  type="checkbox"
                  checked={diagAuthorize}
                  onChange={(e) => setDiagAuthorize(e.target.checked)}
                  className="mt-0.5 accent-[var(--cp-brand-blue)]"
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-[var(--text-emphasis)] leading-snug">同意使用龙虾医生功能</p>
                  <p className="text-xs text-[var(--text-muted)] mt-0.5 leading-relaxed">龙虾医生将在当前 Agent 上创建临时诊断节点，诊断结束后自动销毁</p>
                </div>
              </label>
              <label className="flex items-start gap-2.5 cursor-pointer select-none px-3 py-2.5 border-t border-[var(--border)]">
                <input
                  type="checkbox"
                  checked={diagSnapshot}
                  onChange={(e) => setDiagSnapshot(e.target.checked)}
                  className="mt-0.5 accent-[var(--cp-brand-blue)]"
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-[var(--text-emphasis)] leading-snug">为本次诊断创建配置快照</p>
                  <p className="text-xs text-[var(--text-muted)] mt-0.5 leading-relaxed">勾选后，结束诊断时可一键回滚至 Agent 开始诊断前的状态</p>
                </div>
              </label>
            </div>
            <p className="text-xs text-[var(--text-muted)] mt-2">初始化约需 3-5 分钟，请稍作等待。</p>
          </DialogBody>
          <DialogFooter>
            <Button variant="tenant-outline" size="claw-sm" onClick={() => setShowStartModal(false)}>
              取消
            </Button>
            <Button
              variant="tenant-primary"
              size="claw-sm"
              onClick={handleStartConfirm}
              disabled={!diagAuthorize}
              title={!diagAuthorize ? "请先勾选「同意使用龙虾医生功能」" : undefined}
            >
              确认开始
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 龙虾医生 - 结束诊断 ===== */}
      <Dialog open={showEndModal} onOpenChange={(open) => { if (!open) setShowEndModal(false); }}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>结束诊断</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
            <p className="text-sm text-[var(--text-secondary)] leading-relaxed mb-3">
              结束后龙虾医生将下线，当前诊断将归档至历史诊断记录
            </p>
            {snapshotCreated && (
              <label className="flex items-start gap-2.5 cursor-pointer select-none rounded-[var(--radius-card)] px-3 py-2.5 border border-[var(--border)]">
                <input
                  type="checkbox"
                  checked={rollbackChecked}
                  onChange={(e) => setRollbackChecked(e.target.checked)}
                  className="mt-0.5 accent-[var(--cp-brand-blue)]"
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-[var(--text-emphasis)] leading-snug">回滚到诊断前快照</p>
                  <p className="text-xs text-[var(--text-muted)] mt-0.5 leading-relaxed">将撤销本次所有修复操作，恢复到 Agent 开始诊断前的配置</p>
                </div>
              </label>
            )}
          </DialogBody>
          <DialogFooter>
            <Button variant="tenant-outline" size="claw-sm" onClick={() => setShowEndModal(false)}>
              取消
            </Button>
            <Button variant="tenant-primary" size="claw-sm" onClick={handleEndConfirm}>
              确认结束
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 龙虾医生 - 诊断会话冲突 ===== */}
      <Dialog open={!!conflictInfo} onOpenChange={(open) => { if (!open) setConflictInfo(null); }}>
        <DialogContent size="sm">
          <DialogHeader>
            <DialogTitle>诊断会话冲突</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="text-sm text-[var(--text-secondary)] leading-relaxed">
              {conflictInfo ? (
                <>
                  {`同一时间仅支持一个 Agent 进行诊断。Agent（`}
                  <span className="font-mono text-xs text-[var(--text-emphasis)] whitespace-nowrap">{conflictInfo.instanceId}</span>
                  {` ｜ `}
                  <span className="text-[var(--text-emphasis)]">{conflictInfo.instanceName}</span>
                  {`）当前正在诊断中，请先结束其会话后再开始新的诊断。`}
                </>
              ) : (
                "同一时间仅支持一个 Agent 进行诊断。"
              )}
            </div>
          </DialogBody>
          <DialogFooter>
            <Button variant="tenant-primary" size="claw-sm" onClick={() => setConflictInfo(null)}>我知道了</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ===== 角色管理抽屉（共享组件 RoleManageSheet，与「我的 Agent」100% 一致） ===== */}
      <RoleManageSheet
        dialog={switchRoleDialog}
        onOpenChange={(open) => {
          if (!open) {
            const slots = switchRoleDialog?.roles;
            // 单角色 / 多角色统一走映射表抽屉：只要携带角色位（>=1）关闭即落库本次增删改。
            if (switchRoleDialog && slots && slots.length >= 1) {
              applyDetailRoleChanges(editSlots, switchRoleTargets, slots);
              return;
            }
            setSwitchRoleDialog(null);
            setSwitchRoleTarget(null);
            setSwitchRoleTargets({});
            setExpandedSlotId(null);
            setSwitchSlotSkillPopoverSlot(null);
            setSwitchSlotSkillNames({});
          }
        }}
        visibleRoles={visibleRoles}
        roleConfigProgress={roleConfigProgress}
        switchingRoleAgents={switchingRoleAgents}
        clearRoleSwitching={clearRoleSwitching}
        getSkillDescription={getSkillDescription}
        onApplyRoleChanges={(_id, _name, nextSlots, targets, originalSlots) => {
          applyDetailRoleChanges(nextSlots, targets, originalSlots);
        }}
        onSwitchRole={(_id, _name, targetRole, _targetSlotId, previousRoleName) => {
          applySingleRoleSwitch(targetRole, previousRoleName ?? currentSlotRoleName);
        }}
        onOpenAddRole={({ id, name, roles, allowedRoleNames }) => {
          setStandaloneAddRole({ id, name, roles, allowedRoleNames });
        }}
        onOpenBatchSwitch={({ id, name, roles, allowedRoleNames }) => {
          openStandaloneBatchSwitch({ id, name, roles, allowedRoleNames });
        }}
        onNavigateSettings={(id) => navigate(`/openclaw/${id}`)}
      />

      {/* 「新增角色」独立弹窗：抽取为共享组件 AddRoleDialog，与卡片页（MyOpenClaw）共用同一实现，UI 改动两页联动。 */}
      <AddRoleDialog
        source={standaloneAddRole}
        onClose={() => setStandaloneAddRole(null)}
        visibleRoles={visibleRoles}
        getRoleIntro={getRoleIntro}
        runningAgentNames={runningAgentNames}
        onConfirm={({ target, role, nextSlots, newSlotId, skillCount, roleName }) => {
          // 不直接落库：先进入配置进度弹窗，进度走完 100% 后由回调统一落库。
          // 进度弹窗头部副标题显示用户在新增弹窗里配置的「角色名称」（而非角色风格描述）。
          roleConfigProgress.start({
            roleName: role.name,
            roleSoul: roleName,
            skillCount,
            mode: "add",
            apply: () => {
              applyDetailRoleChanges(nextSlots, { [newSlotId]: role }, target.roles);
            },
          });
        }}
      />

      {/* 批量切换独立弹窗：与卡片页（MyOpenClaw）共用同一实现 BatchSwitchRoleDialog，UI 改动两页联动。
          必须显式传 scheme={1}，与卡片页 editRoleScheme(=1) 保持一致；否则组件默认落到 scheme=2
          （左右三列表格 + 目标角色详情），导致设置页弹窗与主页不一致。 */}
      <BatchSwitchRoleDialog
        source={standaloneBatchSwitch}
        onClose={() => setStandaloneBatchSwitch(null)}
        visibleRoles={visibleRoles}
        getRoleIntro={getRoleIntro}
        runningAgentNames={runningAgentNames}
        onBackgrounded={(agentId, count) => markRoleSwitching(agentId, count)}
        scheme={1}
        onAddRole={() => {
          if (!standaloneBatchSwitch) return;
          setStandaloneAddRole({
            id: standaloneBatchSwitch.id,
            name: standaloneBatchSwitch.name,
            roles: standaloneBatchSwitch.slots,
            allowedRoleNames: standaloneBatchSwitch.allowedRoleNames,
          });
        }}
        onCommit={({ id, slots, targets }) => {
          applyDetailRoleChanges(slots, targets, slots);
          clearRoleSwitching(id);
        }}
      />

      {/* 角色配置进度弹窗 */}
      <RoleConfigProgressDialog
        progress={roleConfigProgress.progress}
        onDismiss={() => {
          const p = roleConfigProgress.progress;
          if (p?.mode === "switch" && p.agentId) {
            markRoleSwitching(p.agentId, p.items?.length ?? 1);
          }
          roleConfigProgress.dismiss();
        }}
      />

    </TenantLayout>
  );
}
