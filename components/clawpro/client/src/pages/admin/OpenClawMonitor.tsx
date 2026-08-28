/**
 * AgentList - 管控端 Agent 列表页
 * 4 个模块：状态统计卡片、状态列+列头筛选、操作列、监控抽屉面板
 */
import { useState, useEffect, useRef, useMemo, useCallback, type ReactElement } from "react";
import { useLocation, Link } from "wouter";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { InfoPopover } from "@/components/ui/info-popover";
import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerBody,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { Input } from "@/components/ui/input";
import { Pagination } from "@/components/ui/pagination";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell } from "@/components/ui/table";
import { SurfaceCard, SurfaceInner } from "@/components/ui/Surface";
import { NumberCard } from "@/components/ui/number-card";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import {
  Dialog, DialogBody, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SearchableSelect,
} from "@/components/ui/select";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { SelectPanel, SelectPanelItem } from "@/components/ui/select-panel";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { StatusTag } from "@/components/ui/status-tag";
import {
  BodyMedium,
  BodyText,
  CodeText,
  MetaMedium,
  MetaText,
  MiniBodyText,
  PanelTitle,
} from "@/components/ui/Typography";
import { Checkbox } from "@/components/ui/checkbox";
import { toast } from "sonner";
import { DatePicker } from "@/components/ui/date-picker";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { getAgentBillingMode, getDefaultLaunchBillingMode } from "@/lib/agentBillingStore";
import { useOpenClawMonitorCalendarBillingExempt } from "./OpenClawMonitor/useOpenClawMonitorCalendarBillingExempt";
import {
  Search, Trash2, ChevronLeft, ChevronRight, RefreshCw, AlertCircle,
  Terminal, Power, MoreHorizontal, RotateCcw, HardDriveDownload, Copy,
  Activity, Loader2, ExternalLink, ChevronDown, Filter, HelpCircle, X, Eye, EyeOff,
  Server, CheckCircle2, PowerOff, Layers, ArrowUp, ArrowDown, Zap, BarChart3,
  MessageCircle, RotateCw, Check, ArrowLeftRight, CircleArrowUp, Tag, Info, SlidersHorizontal,
  Pencil, Plus, Minus, CircleAlert, AlertTriangle, Users, Clock, Wallet,
  TerminalSquare, ListChecks, History as HistoryIcon, ArrowUpCircle,
  Bell, Laptop, PackageCheck,
} from "lucide-react";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
import { MOCK_DEPARTMENTS, MOCK_CLAWS_WITH_DEPT, type DepartmentNode } from "@/lib/mockData";
import { CHANNEL_OPTIONS, MODEL_PROVIDERS, type ChannelConfig } from "@/lib/agentConfigConstants";
import { useAdminModels, CUSTOM_PROVIDER_VALUE, type ModelRow } from "@/lib/modelConfigStore";
import {
  loadBuiltinChannelVisibility,
  loadVisibleCustomChannels,
  onBuiltinChannelVisibilityChange,
  onCustomChannelsChange,
  type CustomChannel as AdminCustomChannel,
} from "@/lib/customChannelStore";
import { useAdminMode } from "@/contexts/AdminModeContext";
import { DepartmentColumnFilter, GroupColumnFilter } from "@/components/admin/ColumnFilters";
import { ScopeSelect } from "@/components/ScopeSelect";
import { MOCK_GROUPS, MOCK_MANUAL_GROUPS, MOCK_USERS, MOCK_USERS_MANUAL } from "./MemberManagement/mock";
import { groupStore } from "./MemberManagement/groupStore";
import type { UserGroup, GroupSource, UserOrg } from "./MemberManagement/types";
import { buildGroupTree, type GroupTreeNode } from "./MemberManagement/health";
import DispatchCommandDialog from "./VersionManagement/components/DispatchCommandDialog";
import CloneAgentImageDialog from "./OpenClawMonitor/CloneAgentImageDialog";
import {
  GuidePointBubble,
  buildPersistenceKey,
  isDismissed,
  markDismissed,
  markExposure,
  resolveBehavior,
  trackOnboarding,
} from "@/components/onboarding";
import ConfigDiffDialog, { buildMockInstanceCompare } from "./MemberManagement/ConfigDiffDialog";
import { getMockInstanceCompareItems } from "./MemberManagement/ConfigCompareDrawer";
import UpdateRecordsDrawer from "./ImageManagement/UpdateRecordsDrawer";
import { useOutdatedTypes, type OutdatedTypeStat } from "./BatchUpdateNotice";
import { compareVersion } from "@/lib/upgradePushStore";
import CreateAgentDialog, { type CreateAgentResult } from "./CreateAgentDialog";
import { loadClawList, saveClawList, notifyClawListChange, type AgentItem } from "@/lib/openclawStore";
import {
  loadAdminCreatedAgents,
  appendAdminCreatedAgent,
  notifyAdminCreatedAgentsChange,
  onAdminCreatedAgentsChange,
  type AdminCreatedAgent,
} from "@/lib/adminAgentStore";
import { AdminDoctorChatPanel } from "@/components/admin/AdminDoctorChatDialog";
import { queryDiagnosisStatus, getCallerId } from "@/lib/doctorDiagnosisApi";
// updateReminderStore 在 ImageUpdateBellEntry 内直接通过 localStorage + state 管理

export type ClawStatus = "creating" | "createFail" | "running" | "loading" | "loadFail" | "shutdown" | "maintaining" | "pending" | "upgrading";
type LocalConnectionStatus = "connected" | "disconnected";
type LocalAgentProduct = "Claude" | "CodeBuddy" | "WorkBuddy" | "Codex";
const LATEST_VERSION = "2026.4.2";

const OTHER_STATUS_GROUPS = [
  {
    title: "需关注",
    items: [
      { label: "创建失败", variant: "red" as const },
      { label: "加载失败", variant: "red" as const },
      { label: "维护中", variant: "gray" as const },
      { label: "待处理", variant: "gray" as const },
    ],
  },
  {
    title: "处理中",
    items: [
      { label: "创建中", variant: "blue" as const },
      { label: "加载中", variant: "blue" as const },
      { label: "升级中", variant: "blue" as const },
    ],
  },
] as const;

/* ─── 统计卡 4 枚渐变图标（与 feature/design-refresh-2026 视觉对齐） ───
 * 资源库阶段：原内联 SVG 已抽为独立资源文件，落地 client/public/icon/monitor-*.svg，
 * 由资源库管线收录、登记「适合 NumberCard 槽位」（classification componentSlot=number-card）。
 * 页面层经 /icon webPath 引用（NumberCard.icon 槽位接受 ReactNode）；组件源码不变。
 * viewBox 18×18，与 NumberCard 内置图标尺寸一致。
 */
function MonitorTotalIcon() {
  // 复用资源库既有资源 instance-total（与本图标逐字一致），不重复引入
  return <img src="/assets/admin-memory-management/instance-total.svg" width={18} height={18} alt="" aria-hidden="true" />;
}

function MonitorRunningIcon() {
  return <img src="/icon/monitor-running.svg" width={18} height={18} alt="" aria-hidden="true" />;
}

function MonitorShutdownIcon() {
  return <img src="/icon/monitor-shutdown.svg" width={18} height={18} alt="" aria-hidden="true" />;
}

function MonitorOtherIcon() {
  return <img src="/icon/monitor-other.svg" width={18} height={18} alt="" aria-hidden="true" />;
}

interface PluginVersions {
  wechat: string;
  dingtalk: string;
  feishu: string;
  wecom: string;
  qq: string;
}

export interface Claw {
  id: string;
  instanceId: string;
  name: string;
  /**
   * 用户ID：该 Agent 的归属用户 ID。
   * - 用户端自建：= 该用户自己
   * - 管控端代建：= 管理员在创建 Agent 弹窗中为其分配 Agent 的目标用户 ID
   *
   * 历史字段名沿用 `creator`（避免改动 mock 数据 / 存储 key），但语义已从"实际下单人"
   * 迁移为"归属用户"。列头显示为「用户ID」。
   */
  creator: string;
  createTime: string;
  status: ClawStatus;
  version: string;
  agentType: 'OpenClaw' | 'Hermes' | 'LightclawACE' | 'LocalAgent';
  localProduct?: LocalAgentProduct;
  localConnectionStatus?: LocalConnectionStatus;
  standards?: Array<string | { name?: string; title?: string; kind?: string; type?: string; lastSync?: string; updatedAt?: string }>;
  pluginVersions: PluginVersions;
  /**
   * Agent 所属组织：管理员在创建 Agent 弹窗中为该 Agent 选定的分组。
   * 一旦创建即固定绑定，不随归属用户的组织变化而变。
   * 缺省（如存量 MOCK_CLAWS 数据）时回退按 creator 反查其所属组织（getCreatorGroupItem*）。
   */
  groupId?: string;
  groupName?: string;
  department?: string;
  departmentId?: string;
  tags?: { key: string; value: string }[];
  /** 计费模式：按量计费(payg)的 Agent 关机不收费；包年包月(subscription)关机不涉及计费 */
  billingMode?: "subscription" | "payg";
  /** 累计运行分钟数（按量计费用于展示） */
  runningMinutes?: number;
}

function getClawBillingMode(c: Claw): "subscription" | "payg" {
  return c.billingMode ?? getAgentBillingMode(c.id) ?? getDefaultLaunchBillingMode();
}

/**
 * Agent 实例当前资源配置（用于列表「资源配置」列与「调整配置」弹窗展示当前现状）。
 * 真实环境应由后端返回；此处依据 id 做确定性映射，保证展示稳定、各实例现状有差异。
 */
interface ClawResourceConfig {
  /** 实例规格名（如 Ai2.MEDIUM4） */
  specName: string;
  /** CPU 核数 */
  cpu: number;
  /** 内存 GiB */
  memory: number;
  /** 系统盘容量 GB */
  systemDiskSize: number;
  /** 系统盘类型（仅 hover 展示，本期不支持变更） */
  systemDiskType: string;
}

const RESOURCE_SPEC_OPTIONS: { specName: string; cpu: number; memory: number }[] = [
  { specName: "Ai2.MEDIUM4",   cpu: 2, memory: 4 },
  { specName: "Ai2.LARGE8",    cpu: 4, memory: 8 },
  { specName: "Ai2.2XLARGE16", cpu: 8, memory: 16 },
];
const RESOURCE_DISK_SIZES = [50, 80, 100, 200];
const RESOURCE_DISK_TYPES = ["高性能云硬盘", "SSD云硬盘"];

/**
 * Demo 固定资源配置（按 instanceId 指定），用于「调整配置」演示场景，保证每次演示结果一致。
 * 覆盖列表当前展示的前 10 台中的云实例 Agent；未登记的实例继续走下方 seed 派生的确定性 mock。
 * 真实环境应由后端返回。
 */
const MOCK_INSTANCE_RESOURCE: Record<string, { specName: string; systemDiskSize: number }> = {
  "ins-e25z9taq": { specName: "Ai2.MEDIUM4",   systemDiskSize: 50 },
  "ins-d14y8szp": { specName: "Ai2.2XLARGE16", systemDiskSize: 200 },
  "ins-c03x7ryo": { specName: "Ai2.LARGE8",    systemDiskSize: 100 },
  "ins-b92w6qxn": { specName: "Ai2.MEDIUM4",   systemDiskSize: 80 },
  "ins-a81v5pwm": { specName: "Ai2.MEDIUM4",   systemDiskSize: 50 },
  "ins-z70u4ovl": { specName: "Ai2.2XLARGE16", systemDiskSize: 200 },
  "ins-y69t3nuk": { specName: "Ai2.LARGE8",    systemDiskSize: 100 },
  "ins-x58s2mtj": { specName: "Ai2.MEDIUM4",   systemDiskSize: 80 },
  "ins-k88w3vcd": { specName: "Ai2.LARGE8",    systemDiskSize: 200 },
  "ins-stopped-upgrade01": { specName: "Ai2.MEDIUM4", systemDiskSize: 100 },
};

function getClawResourceConfig(c: Claw): ClawResourceConfig {
  const seed = Number(String(c.id).replace(/\D/g, "")) || 0;
  const systemDiskType = RESOURCE_DISK_TYPES[seed % RESOURCE_DISK_TYPES.length];
  // demo 固定配置优先：保证演示场景下规格 / 系统盘稳定一致
  const fixed = c.instanceId ? MOCK_INSTANCE_RESOURCE[c.instanceId] : undefined;
  if (fixed) {
    const fixedSpec = RESOURCE_SPEC_OPTIONS.find((s) => s.specName === fixed.specName) ?? RESOURCE_SPEC_OPTIONS[0];
    return { ...fixedSpec, systemDiskSize: fixed.systemDiskSize, systemDiskType };
  }
  const spec = RESOURCE_SPEC_OPTIONS[seed % RESOURCE_SPEC_OPTIONS.length];
  const systemDiskSize = RESOURCE_DISK_SIZES[seed % RESOURCE_DISK_SIZES.length];
  return { ...spec, systemDiskSize, systemDiskType };
}

/** Demo 固定公网信息（按 instanceId 指定），未登记的实例由 seed 派生 */
interface PublicNetworkInfo {
  /** 是否已开启公网 */
  enabled: boolean;
  /** 公网计费模式 */
  billingLabel: string;
  /** 带宽上限 Mbps */
  bandwidth: number;
}
const MOCK_PUBLIC_NETWORK: Record<string, PublicNetworkInfo> = {
  "ins-g71c6vud": { enabled: true,  billingLabel: "按带宽计费", bandwidth: 5 },
  "ins-h92d7xwe": { enabled: true,  billingLabel: "按流量计费", bandwidth: 10 },
  "ins-k25f9zwg": { enabled: false, billingLabel: "—",         bandwidth: 0 },
  "ins-l36g0axh": { enabled: false, billingLabel: "—",         bandwidth: 0 },
  "ins-x58s2mtj": { enabled: true,  billingLabel: "按带宽计费", bandwidth: 3 },
};

function getPublicNetworkInfo(c: Claw): PublicNetworkInfo {
  if (c.instanceId && MOCK_PUBLIC_NETWORK[c.instanceId]) {
    return MOCK_PUBLIC_NETWORK[c.instanceId];
  }
  const seed = Number(String(c.id).replace(/\D/g, "")) || 0;
  const enabled = seed % 3 !== 0;
  return {
    enabled,
    billingLabel: enabled ? (seed % 2 === 0 ? "按带宽计费" : "按流量计费") : "—",
    bandwidth: enabled ? [3, 5, 10, 20, 50][seed % 5] : 0,
  };
}

/** 规格在 RESOURCE_SPEC_OPTIONS 中的序号（越大规格越高），找不到返回 -1 */
function getSpecRankByName(specName: string): number {
  return RESOURCE_SPEC_OPTIONS.findIndex((s) => s.specName === specName);
}

/** 规格展示格式：Ai2.LARGE8（4核8GiB），数字/单位间无空格 */
function formatSpecLabel(s: { specName: string; cpu: number; memory: number }): string {
  return `${s.specName}（${s.cpu}核${s.memory}GiB）`;
}

/**
 * 调整云资源配置（升配 / 扩容）的前端第一层可用性校验。
 * 返回不可调整的原因文案；可调整返回 null。
 * 注意：提交前仍需以后端校验为准（CVM 实例状态 / LatestOperationState / 库存 / 配额 / 费用 / 订单）。
 */
const ADJUSTABLE_STATUSES: ClawStatus[] = ["running", "shutdown"];

function getAdjustDisabledReason(c: Claw): string | null {
  // 优先级1：Agent 类型 / 实例 ID / 资源配置缺失
  if (c.agentType === "LocalAgent" || !c.instanceId) {
    return "当前 Agent 不支持配置云资源";
  }
  const rc = getClawResourceConfig(c);
  if (!rc || getSpecRankByName(rc.specName) < 0 || !rc.systemDiskSize) {
    return "当前 Agent 不支持配置云资源";
  }
  // 优先级2：ClawPro 状态（仅运行中 / 已关机可调整）
  if (!ADJUSTABLE_STATUSES.includes(c.status)) {
    return "当前状态不支持调整配置";
  }
  return null;
}



/** 逐台判定结果类别：adjustable=可调整 blocked=不可调整 no_change=无需调整 */
type AdjustResultKind = "adjustable" | "blocked" | "no_change";
type AdjustResult = { kind: AdjustResultKind; reason?: string };

function evalSpecAdjust(c: Claw, rc: ClawResourceConfig | null, targetSpecName: string): AdjustResult {
  // 不可调整：类型 / 状态，原因使用与列表一致的 Agent 类型 / 状态文案
  if (c.agentType === "LocalAgent") return { kind: "blocked", reason: `Agent 类型为${AGENT_TYPE_DISPLAY[c.agentType] ?? c.agentType}，暂不支持调整` };
  if (!c.instanceId) return { kind: "blocked", reason: `Agent 类型为${AGENT_TYPE_DISPLAY[c.agentType] ?? c.agentType}，暂不支持调整` };
  if (!ADJUSTABLE_STATUSES.includes(c.status)) return { kind: "blocked", reason: `因实例状态为「${STATUS_CONFIG[c.status]?.label ?? c.status}」，暂不支持调整` };
  if (!rc || !rc.specName) return { kind: "blocked", reason: "缺少实例规格信息" };
  const curRank = getSpecRankByName(rc.specName);
  if (curRank < 0) return { kind: "blocked", reason: "当前规格暂不支持升配" };
  const targetRank = getSpecRankByName(targetSpecName);
  if (curRank > targetRank) return { kind: "blocked", reason: "不支持降配" };
  if (curRank === targetRank) return { kind: "no_change", reason: "已是目标规格" };
  return { kind: "adjustable" };
}

/**
 * 批量逐台判定：系统盘容量扩容是否支持（基于已选目标容量）。
 */
function evalDiskAdjust(c: Claw, rc: ClawResourceConfig | null, targetSize: number): AdjustResult {
  if (c.agentType === "LocalAgent") return { kind: "blocked", reason: `Agent 类型为${AGENT_TYPE_DISPLAY[c.agentType] ?? c.agentType}，暂不支持调整` };
  if (!c.instanceId) return { kind: "blocked", reason: `Agent 类型为${AGENT_TYPE_DISPLAY[c.agentType] ?? c.agentType}，暂不支持调整` };
  if (!ADJUSTABLE_STATUSES.includes(c.status)) return { kind: "blocked", reason: `因实例状态为「${STATUS_CONFIG[c.status]?.label ?? c.status}」，暂不支持调整` };
  if (!rc || !rc.systemDiskSize) return { kind: "blocked", reason: "缺少系统盘容量信息" };
  if (rc.systemDiskSize > targetSize) return { kind: "blocked", reason: "不支持缩容" };
  if (rc.systemDiskSize === targetSize) return { kind: "no_change", reason: "已是目标容量" };
  return { kind: "adjustable" };
}

/** Agent 当前已配置的模型（用于批量配置弹窗中展示各 Agent 现状） */
type ClawCurrentModel = { providerLabel: string; versionLabel: string; primary: boolean };

/**
 * 派生某个 Agent 当前已配置的模型列表（主模型 + 备选模型）。
 * 真实环境应由后端返回；此处依据 id 做确定性映射，保证展示稳定、各 Agent 现状不同。
 */
function getClawCurrentModels(c: Claw): ClawCurrentModel[] {
  const flat: { providerLabel: string; versionLabel: string }[] = [];
  MODEL_PROVIDERS.forEach((p) => {
    p.versions.forEach((v) => {
      if (p.value === "custom") return; // 自定义模型不作为默认现状展示
      flat.push({ providerLabel: p.label, versionLabel: v.label });
    });
  });
  if (flat.length === 0) return [];
  const seed = Number(String(c.id).replace(/\D/g, "")) || 0;
  const primaryIdx = seed % flat.length;
  const result: ClawCurrentModel[] = [{ ...flat[primaryIdx], primary: true }];
  // 约一半 Agent 额外带一个备选模型
  if (seed % 2 === 0) {
    const altIdx = (primaryIdx + 1) % flat.length;
    if (altIdx !== primaryIdx) result.push({ ...flat[altIdx], primary: false });
  }
  return result;
}

export const STATUS_CONFIG: Record<ClawStatus, {
  label: string;
  tagVariant: "green" | "blue" | "red" | "gray";  // StatusTag variant
}> = {
  creating:    { label: "创建中",   tagVariant: "blue" },
  createFail:  { label: "创建失败", tagVariant: "red" },
  running:     { label: "运行中",   tagVariant: "green" },
  loading:     { label: "加载中",   tagVariant: "blue" },
  loadFail:    { label: "加载失败", tagVariant: "red" },
  shutdown:    { label: "已关机",   tagVariant: "gray" },
  maintaining: { label: "维护中",   tagVariant: "blue" },
  pending:     { label: "待处理",   tagVariant: "gray" },
  upgrading:   { label: "升级中",   tagVariant: "blue" },
};

const DEFAULT_PLUGIN_VERSIONS: PluginVersions = { wechat: "3.2.1", dingtalk: "2.8.0", feishu: "1.5.3", wecom: "2.1.4", qq: "1.0.2" };

// Agent 类型显示名称映射
export const AGENT_TYPE_DISPLAY: Record<string, string> = {
  'OpenClaw':    'OpenClaw',
  'Hermes':      'Hermes Agent',
  'LightclawACE': 'LightClaw ACE',
  'LocalAgent':  '本地 Agent',
};

const LOCAL_AGENT_PRODUCT_DISPLAY: Record<LocalAgentProduct, string> = {
  Claude: "Claude",
  CodeBuddy: "CodeBuddy",
  WorkBuddy: "WorkBuddy",
  Codex: "Codex",
};

const AGENT_TYPE_FILTER_GROUPS = [
  {
    key: "cloud",
    label: "云端 Agent",
    items: [
      { key: "OpenClaw", label: "OpenClaw" },
      { key: "Hermes", label: "Hermes Agent" },
      { key: "LightclawACE", label: "LightClaw ACE" },
    ],
  },
  {
    key: "local",
    label: "本地 Agent",
    items: [
      { key: "Claude", label: "Claude" },
      { key: "CodeBuddy", label: "CodeBuddy" },
      { key: "WorkBuddy", label: "WorkBuddy" },
      { key: "Codex", label: "Codex" },
    ],
  },
] as const;

type AgentTypeFilterKey = typeof AGENT_TYPE_FILTER_GROUPS[number]["items"][number]["key"];
const ALL_AGENT_TYPE_FILTER_KEYS = AGENT_TYPE_FILTER_GROUPS.flatMap((group) => group.items.map((item) => item.key)) as AgentTypeFilterKey[];

function getLocalAgentProduct(claw?: Pick<Claw, "localProduct"> | null): LocalAgentProduct {
  return claw?.localProduct ?? "WorkBuddy";
}

function getClawTypeFilterKey(claw: Claw): AgentTypeFilterKey {
  return claw.agentType === "LocalAgent" ? getLocalAgentProduct(claw) : claw.agentType;
}

function getClawTypeLabel(claw: Claw): string {
  if (claw.agentType === "LocalAgent") {
    const product = getLocalAgentProduct(claw);
    return LOCAL_AGENT_PRODUCT_DISPLAY[product] ?? product;
  }
  return AGENT_TYPE_DISPLAY[claw.agentType] ?? claw.agentType;
}

export const MOCK_CLAWS: Claw[] = [
  { id: "1",  instanceId: "ins-g71c6vud", name: "Alice的技术助手", tags: [{ key: "所属产品", value: "gpulab" }, { key: "env", value: "production" }],    creator: "alice@acompany.com",  createTime: "2025-12-01 09:12:34", status: "shutdown",     version: "2026.3.28", agentType: "OpenClaw",    pluginVersions: { wechat: "3.2.1", dingtalk: "2.8.0", feishu: "1.5.3", wecom: "2.1.4", qq: "1.0.2" } },
  { id: "2",  instanceId: "ins-h92d7xwe", name: "Bob工作助手",       creator: "bob@acompany.com",    createTime: "2025-12-15 14:05:22", status: "shutdown",     version: "2026.4.2",  agentType: "Hermes",      pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "3",  instanceId: "ins-j14e8yvf", name: "Carol的研究助手",   creator: "carol@acompany.com",  createTime: "2026-01-05 10:33:47", status: "shutdown",   version: "2026.3.28", agentType: "LightclawACE", pluginVersions: { wechat: "3.2.1", dingtalk: "2.8.0", feishu: "1.5.3", wecom: "2.1.4", qq: "1.0.2" } },
  { id: "4",  instanceId: "ins-k25f9zwg", name: "Dave的代码助手", tags: [{ key: "test", value: "test2" }],    creator: "dave@acompany.com",   createTime: "2026-01-20 16:48:09", status: "running",     version: "2026.3.28", agentType: "OpenClaw",    pluginVersions: { wechat: "3.1.5", dingtalk: "2.7.2", feishu: "1.4.8", wecom: "2.0.9", qq: "1.0.1" } },
  { id: "5",  instanceId: "ins-l36g0axh", name: "Eve的写作助手",     creator: "eve@acompany.com",    createTime: "2026-02-10 08:21:55", status: "shutdown", version: "2026.3.28", agentType: "Hermes",      pluginVersions: DEFAULT_PLUGIN_VERSIONS },
  { id: "6",  instanceId: "ins-m47h1byi", name: "Frank的数据助手",   creator: "frank@acompany.com",  createTime: "2026-02-18 11:07:30", status: "running",     version: "2026.4.2",  agentType: "OpenClaw",    pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "7",  instanceId: "ins-n58i2czj", name: "Grace的翻译助手",   creator: "grace@acompany.com",  createTime: "2026-02-25 15:44:18", status: "creating",   version: "2026.3.28", agentType: "LightclawACE", pluginVersions: DEFAULT_PLUGIN_VERSIONS },
  { id: "8",  instanceId: "ins-o69j3dak", name: "Henry的销售助手", tags: [{ key: "所属产品", value: "gpulab" }, { key: "team", value: "sales" }, { key: "env", value: "staging" }],   creator: "henry@acompany.com",  createTime: "2026-03-01 09:58:03", status: "running",     version: "2026.3.28", agentType: "OpenClaw",    pluginVersions: { wechat: "3.2.1", dingtalk: "2.8.0", feishu: "1.5.3", wecom: "2.1.4", qq: "1.0.2" } },
  { id: "9",  instanceId: "ins-p70k4ebl", name: "Ivy的客服助手",     creator: "ivy@acompany.com",    createTime: "2026-03-05 13:26:41", status: "running",     version: "2026.4.2",  agentType: "Hermes",      pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "10", instanceId: "ins-q81l5fcm", name: "Jack的会议助手",    creator: "jack@acompany.com",   createTime: "2026-03-08 17:02:15", status: "running",     version: "2026.3.28", agentType: "OpenClaw",    pluginVersions: { wechat: "3.2.0", dingtalk: "2.8.0", feishu: "1.5.2", wecom: "2.1.3", qq: "1.0.2" } },
  { id: "11", instanceId: "ins-r92m6gdn", name: "Karen的报告助手",   creator: "karen@acompany.com",  createTime: "2026-03-09 10:15:50", status: "loadFail",   version: "2026.3.28", agentType: "LightclawACE", pluginVersions: DEFAULT_PLUGIN_VERSIONS },
  { id: "12", instanceId: "ins-s03n7heo", name: "Leo的项目助手", tags: [{ key: "tencentcloud:autoscaling", value: "asg-1f7z0pa9" }],     creator: "leo@acompany.com",    createTime: "2026-03-10 08:39:27", status: "running",     version: "2026.4.2",  agentType: "OpenClaw",    pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "13", instanceId: "ins-t14o8ipf", name: "Mia的新助手",        creator: "mia@acompany.com",    createTime: "2026-03-12 11:00:00", status: "maintaining", version: "2026.3.28", agentType: "Hermes",      pluginVersions: DEFAULT_PLUGIN_VERSIONS },
  { id: "14", instanceId: "ins-u25p9jqg", name: "Noah的分析助手",    creator: "noah@acompany.com",   createTime: "2026-03-13 14:30:00", status: "pending",    version: "2026.3.28", agentType: "OpenClaw",    pluginVersions: DEFAULT_PLUGIN_VERSIONS },
  { id: "15", instanceId: "ins-v36q0krh", name: "Olivia的运营助手",  creator: "olivia@acompany.com",  createTime: "2026-03-14 09:00:00", status: "running",     version: "2026.4.2",  agentType: "LightclawACE", pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "16", instanceId: "ins-w47r1lsi", name: "Peter的财务助手",  creator: "peter@acompany.com",   createTime: "2026-03-15 10:20:00", status: "running",     version: "2026.3.28", agentType: "OpenClaw",    pluginVersions: { wechat: "3.2.1", dingtalk: "2.8.0", feishu: "1.5.3", wecom: "2.1.4", qq: "1.0.2" } },
  { id: "17", instanceId: "ins-x58s2mtj", name: "Quinn的法务助手",  creator: "quinn@acompany.com",   createTime: "2026-03-16 11:45:00", status: "running",     version: "2026.4.2",  agentType: "Hermes",      pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "18", instanceId: "ins-y69t3nuk", name: "Rachel的HR助手",      creator: "rachel@acompany.com",  createTime: "2026-03-17 13:10:00", status: "running",     version: "2026.3.28", agentType: "OpenClaw",    pluginVersions: { wechat: "3.2.1", dingtalk: "2.8.0", feishu: "1.5.3", wecom: "2.1.4", qq: "1.0.2" } },
  { id: "19", instanceId: "ins-z70u4ovl", name: "Sam的产品助手",    creator: "sam@acompany.com",     createTime: "2026-03-18 14:30:00", status: "running",     version: "2026.4.2",  agentType: "LightclawACE", pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "20", instanceId: "ins-a81v5pwm", name: "Tina的客服助手",  creator: "tina@acompany.com",    createTime: "2026-03-19 15:00:00", status: "running",     version: "2026.3.28", agentType: "OpenClaw",    pluginVersions: { wechat: "3.2.1", dingtalk: "2.8.0", feishu: "1.5.3", wecom: "2.1.4", qq: "1.0.2" } },
  { id: "21", instanceId: "ins-b92w6qxn", name: "Uma的设计助手",   creator: "uma@acompany.com",     createTime: "2026-03-20 09:30:00", status: "running",     version: "2026.4.2",  agentType: "Hermes",      pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "22", instanceId: "ins-c03x7ryo", name: "Victor的技术助手", creator: "victor@acompany.com",  createTime: "2026-03-21 10:00:00", status: "running",     version: "2026.3.28", agentType: "OpenClaw",    pluginVersions: { wechat: "3.2.1", dingtalk: "2.8.0", feishu: "1.5.3", wecom: "2.1.4", qq: "1.0.2" } },
  { id: "23", instanceId: "ins-d14y8szp", name: "这是一个名称非常非常长的智能助手用来测试超长文本截断效果", creator: "longname-user@very-long-domain-example.com", createTime: "2026-05-01 09:00:00", status: "running",     version: "2026.4.2",  agentType: "OpenClaw",    pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "24", instanceId: "ins-e25z9taq", name: "GPULab产品线专属AI智能运营分析与决策支持系统", creator: "product-ops-admin@enterprise-acompany.com", createTime: "2026-05-02 10:30:00", status: "running",     version: "2026.4.2",  agentType: "Hermes",      pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
  { id: "25", instanceId: "local-codebuddy-01", name: "CodeBuddy-研发工作站", creator: "dave@acompany.com", createTime: "2026-05-06 09:20:00", status: "running", version: "codebuddy-1.8.4", agentType: "LocalAgent", localProduct: "CodeBuddy", localConnectionStatus: "connected", standards: ["代码安全基线", "研发交付规范"], pluginVersions: DEFAULT_PLUGIN_VERSIONS, tags: [{ key: "部署形态", value: "local" }, { key: "客户端", value: "codebuddy" }, { key: "env", value: "dev" }] },
  { id: "26", instanceId: "local-workbuddy-01", name: "WorkBuddy-运营笔记本", creator: "olivia@acompany.com", createTime: "2026-05-07 10:10:00", status: "running", version: "workbuddy-2.3.1", agentType: "LocalAgent", localProduct: "WorkBuddy", localConnectionStatus: "connected", standards: ["Agent 安全合规基线", "交付协作规范"], pluginVersions: DEFAULT_PLUGIN_VERSIONS, tags: [{ key: "部署形态", value: "local" }, { key: "客户端", value: "workbuddy" }, { key: "env", value: "office" }] },
  { id: "27", instanceId: "local-workbuddy-02", name: "WorkBuddy-离线笔记本", creator: "developer@acompany.com", createTime: "2026-05-08 16:35:00", status: "running", version: "workbuddy-2.2.0", agentType: "LocalAgent", localProduct: "WorkBuddy", localConnectionStatus: "disconnected", pluginVersions: DEFAULT_PLUGIN_VERSIONS, tags: [{ key: "部署形态", value: "local" }, { key: "客户端", value: "workbuddy" }, { key: "env", value: "offline" }] },
  { id: "28", instanceId: "local-claude-01", name: "Claude-产品方案机", creator: "product-ops-admin@enterprise-acompany.com", createTime: "2026-05-09 14:05:00", status: "running", version: "claude-0.9.2", agentType: "LocalAgent", localProduct: "Claude", localConnectionStatus: "connected", standards: [{ name: "产品方案写作规范", type: "CLAUDE.md", lastSync: "2026-06-16 18:52" }], pluginVersions: DEFAULT_PLUGIN_VERSIONS, tags: [{ key: "部署形态", value: "local" }, { key: "客户端", value: "claude" }, { key: "env", value: "planning" }] },
  { id: "30", instanceId: "local-codex-01", name: "Codex-研发协作机", creator: "frontend-dev@acompany.com", createTime: "2026-05-10 11:25:00", status: "running", version: "codex-1.0.3", agentType: "LocalAgent", localProduct: "Codex", localConnectionStatus: "connected", standards: [{ name: "前端研发协作规范", type: "rules", lastSync: "2026-06-16 19:05" }], pluginVersions: DEFAULT_PLUGIN_VERSIONS, tags: [{ key: "部署形态", value: "local" }, { key: "客户端", value: "codex" }, { key: "env", value: "dev" }] },
  // 测试用：已关机云端 Agent，规格 Ai2.MEDIUM4（2核4GiB，非最高），支持升配/扩容
  { id: "29", instanceId: "ins-stopped-upgrade01", name: "已关机升配测试助手", creator: "test@acompany.com", createTime: "2026-03-20 00:00:00", status: "shutdown", version: "2026.4.2", agentType: "OpenClaw", pluginVersions: { wechat: "3.3.0", dingtalk: "2.9.1", feishu: "1.6.0", wecom: "2.2.0", qq: "1.1.0" } },
];

// ─── 调整配置（mock）：实例规格 / 系统盘映射 ───────────────────────────────
// 仅用于「调整配置」弹窗的前端 mock 校验；不写入 MOCK_CLAWS 主数据结构。
// 规格元数据：核数 / 内存 GiB（统一维护，供简洁/完整两种格式复用）
const SPEC_META: Record<string, { cpu: number; mem: number }> = {
  "Ai2.MEDIUM4":   { cpu: 2, mem: 4 },
  "Ai2.LARGE8":    { cpu: 4, mem: 8 },
  "Ai2.2XLARGE16": { cpu: 8, mem: 16 },
};
// 简洁格式：8核16GiB（弹窗实例列表 / Agent 列表使用）
function formatSpecShort(spec: string): string {
  const m = SPEC_META[spec];
  return m ? `${m.cpu}核${m.mem}GiB` : spec;
}
// 完整格式：8核16GiB(Ai2.2XLARGE16)（目标规格下拉 / Agent 详情抽屉使用）
function formatSpecFull(spec: string): string {
  const m = SPEC_META[spec];
  return m ? `${m.cpu}核${m.mem}GiB(${spec})` : spec;
}
const ADJUST_SPEC_OPTIONS = [
  { value: "Ai2.MEDIUM4",   label: formatSpecFull("Ai2.MEDIUM4") },
  { value: "Ai2.LARGE8",    label: formatSpecFull("Ai2.LARGE8") },
  { value: "Ai2.2XLARGE16", label: formatSpecFull("Ai2.2XLARGE16") },
];
const SPEC_RANK: Record<string, number> = {
  "Ai2.MEDIUM4": 1, "Ai2.LARGE8": 2, "Ai2.2XLARGE16": 3,
};
const SPEC_DISK_CYCLE = [50, 100, 200, 100];
// 最小 mock 映射：将每个云端 Agent 映射到其当前实例规格 / 系统盘 / 计费 / 公网（不含本地 Agent，不写入 MOCK_CLAWS 主结构）
type InstanceSpecInfo = {
  spec: string;
  systemDiskType: string;
  systemDiskCapacity: number;
  chargeType: string;       // 计费模式（mock）
  publicAssigned: boolean;  // 是否分配公网 IP
  publicChargeMode: string; // 公网计费模式（mock）
  publicBandwidth: string;  // 公网带宽上限（mock）
};
const MOCK_INSTANCE_SPEC_MAP: Record<string, InstanceSpecInfo> = (() => {
  const map: Record<string, InstanceSpecInfo> = {};
  MOCK_CLAWS.filter((c) => c.agentType !== "LocalAgent").forEach((c, idx) => {
    map[c.id] = {
      spec: ADJUST_SPEC_OPTIONS[idx % ADJUST_SPEC_OPTIONS.length].value,
      systemDiskType: "高性能云硬盘",
      systemDiskCapacity: SPEC_DISK_CYCLE[idx % SPEC_DISK_CYCLE.length],
      chargeType: idx % 2 === 0 ? "包年包月" : "按量计费",
      publicAssigned: idx % 3 !== 0,
      publicChargeMode: idx % 2 === 0 ? "按带宽计费" : "按流量计费",
      publicBandwidth: "5Mbps",
    };
  });
  return map;
})();

function getInstanceSpecInfo(c: Claw): InstanceSpecInfo {
  return MOCK_INSTANCE_SPEC_MAP[c.id] ?? {
    spec: "Ai2.MEDIUM4", systemDiskType: "高性能云硬盘", systemDiskCapacity: 50,
    chargeType: "按量计费", publicAssigned: true, publicChargeMode: "按带宽计费", publicBandwidth: "5 Mbps",
  };
}

type AdjustValidationResult = { label: string; note: string };

// 前端 mock 校验：未选目标配置时不执行任何校验（含实例状态），统一展示「—」；
// 选/填合法目标后：先校验实例状态，再比较目标配置；结果仅 可调整 / 不可调整 / —
function validateAdjust(
  claw: Claw,
  type: "spec" | "capacity",
  targetSpec?: string,
  targetCapacity?: number,
): AdjustValidationResult {
  // 0) 未选目标配置：不校验，结果/说明统一「—」
  if (type === "spec") {
    if (!targetSpec) return { label: "—", note: "—" };
  } else {
    if (targetCapacity == null || Number.isNaN(targetCapacity)) return { label: "—", note: "—" };
  }
  // 1) 校验实例状态：仅「运行中」「已关机」支持调整
  if (claw.status !== "running" && claw.status !== "shutdown") {
    return { label: "不可调整", note: `因实例状态为「${STATUS_CONFIG[claw.status].label}」，暂不支持调整` };
  }
  const info = getInstanceSpecInfo(claw);
  if (type === "spec") {
    const cur = SPEC_RANK[info.spec] ?? 0;
    const tgt = SPEC_RANK[targetSpec as string] ?? 0;
    if (tgt > cur) return { label: "可调整", note: "—" };
    if (tgt === cur) return { label: "不可调整", note: "已是目标规格" };
    return { label: "不可调整", note: "不支持降配" };
  }
  // 系统盘容量扩容：容量单位统一 GiB
  const cap = targetCapacity as number;
  if (cap > info.systemDiskCapacity) return { label: "可调整", note: "—" };
  if (cap < info.systemDiskCapacity) return { label: "不可调整", note: "不支持缩容" };
  return { label: "不可调整", note: "目标容量需大于当前系统盘容量" };
}

const PAGE_SIZE = 10;

// processing 实例操作列：保留原按钮文案，置灰禁用，hover 提示锁定原因（不展示「-」）
function LockedOpLabel({ label, tip }: { label: string; tip: string }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="text-[14px] text-[var(--text-brand)] opacity-40 cursor-not-allowed whitespace-nowrap">{label}</span>
      </TooltipTrigger>
      <TooltipContent side="top" className="text-xs">{tip}</TooltipContent>
    </Tooltip>
  );
}

// ─── 存量实例组织异常标记（mock） ───────────────────────────────────────────
/** 用户已不在该组织的实例 */
const MOCK_GROUP_MISMATCH_INSTANCES = new Set(["1", "5"]);
/** 待用户处理的实例 */
const MOCK_GROUP_PENDING_USER_INSTANCES = new Set(["2"]);
/** 待用户处理的允许操作 */
const MOCK_PENDING_USER_ACTIONS: Record<string, { allowTransfer: boolean; allowMigrate: boolean }> = {
  "2": { allowTransfer: true, allowMigrate: true },
};
/** 用户当前所在组织（用户变更后实际所在的组织） */
const MOCK_INSTANCE_CURRENT_GROUP: Record<string, string> = {
  "1": "AI 组",
  "5": "运营一组",
  "2": "前端研发同学",
};

// ─── 共享范围（与用户端 agent 共享范围联动） ───────────────────────────────
/**
 * 与用户端实例的共享范围联动的展示数据。
 * 字段口径与用户端 openclawStore.AgentItem 完全一致：
 *   - shareScope: "private"=仅自己（即未被共享），"shared"=共享给分组/个人
 *   - shareGroupNames: 共享到的「分组/组织」名称列表（整组覆盖时归并为组名）
 *   - shareUserNames : 共享到的「个人」名称列表（未被整组覆盖的散选成员）
 * 未在此表登记的实例 → 视为「仅自己」，即未被共享，展示与组织/部门一致的横线「—」。
 */
interface ClawShareScope {
  shareScope: "private" | "shared";
  shareGroupNames?: string[];
  shareUserNames?: string[];
}
const MOCK_INSTANCE_SHARE_SCOPE: Record<string, ClawShareScope> = {
  // 共享给单个分组 → 直接展示组名
  "1": { shareScope: "shared", shareGroupNames: ["前端组"] },
  // 共享给多个对象（分组 + 多个个人）→ 展示首项 + "+N"，hover 展开全部
  "4": { shareScope: "shared", shareGroupNames: ["产品组"], shareUserNames: ["Bob", "Carol", "Henry"] },
  // 共享给多个对象（分组 + 多个个人）→ 展示首项 + "+N"，hover 展开全部
  "6": { shareScope: "shared", shareGroupNames: ["运营组"], shareUserNames: ["Dave", "Grace"] },
  // 共享给多个分组 → 展示首项 + "+N"
  "8": { shareScope: "shared", shareGroupNames: ["AI 组", "数据组", "客服组"] },
  // 显式「仅自己」→ 未被共享，展示横线
  "9": { shareScope: "private" },
  // Victor的技术助手 → 共享给多个对象（分组 + 多个个人），展示首项 + "+N"，hover 展开全部
  "22": { shareScope: "shared", shareGroupNames: ["技术中台组"], shareUserNames: ["Alice", "Quinn", "Sam"] },
};

/** 取某实例的共享对象展示列表（组在前、个人在后），未共享返回空数组 */
function getClawShareTargets(clawId: string): string[] {
  const s = MOCK_INSTANCE_SHARE_SCOPE[clawId];
  if (!s || s.shareScope !== "shared") return [];
  return [...(s.shareGroupNames ?? []), ...(s.shareUserNames ?? [])];
}

// ─── 组织相关工具函数 ─────────────────────────────────────────────────────
/** 配置不符合所属组织的实例（mock）：用于「组织配置对比」演示 */
const MOCK_CONFIG_MISMATCH_INSTANCES = new Set(["1", "3", "5", "8"]);

/** 取某实例与所属组织的逐项配置对比（mock，严格对齐原型视图⑧配置项）。
 * 符合的实例：所有项均一致；不符合的实例：取原型演示用差异数据。 */
function getInstanceConfigCompareItems(clawId: string) {
  return getMockInstanceCompareItems(!MOCK_CONFIG_MISMATCH_INSTANCES.has(clawId));
}
/** 实例是否整体符合所属组织配置 */
function isInstanceConfigMatch(clawId: string): boolean {
  return !MOCK_CONFIG_MISMATCH_INSTANCES.has(clawId);
}
/** 实例是否为「异常组织」：用户不在该组织（含待用户处理 → 红点）或配置不符合组织（→ 橙点） */
function isOrgAnomalyInstance(clawId: string): boolean {
  return (
    MOCK_GROUP_MISMATCH_INSTANCES.has(clawId) ||
    MOCK_GROUP_PENDING_USER_INSTANCES.has(clawId) ||
    MOCK_CONFIG_MISMATCH_INSTANCES.has(clawId)
  );
}
/** 配置对比中「不检查」的类目：技能本体、技能安装来源、企业插件、企业MCP 不纳入符合/不符合判定。 */
function isUncheckedCompareCategory(category: string): boolean {
  return (
    category === "技能" ||
    category.includes("技能安装来源") ||
    category.includes("企业插件") ||
    category.includes("企业MCP")
  );
}

// 规则：被标记「用户不在该组织」（含待用户处理）的实例，其当前状态恒为「已关机」。
// 统一在此规范化，避免后续调整 MOCK_CLAWS 时遗漏某个被标记实例。
for (const claw of MOCK_CLAWS) {
  if (MOCK_GROUP_MISMATCH_INSTANCES.has(claw.id) || MOCK_GROUP_PENDING_USER_INSTANCES.has(claw.id)) {
    claw.status = "shutdown";
  }
}

/** 获取组织的完整路径（如 "产品组" 或 "研发组 / 前端"） */
function getGroupPath(groupId: string, groups: UserGroup[]): string {
  const map = new Map(groups.map((g) => [g.id, g]));
  const chain: string[] = [];
  let cur = map.get(groupId);
  while (cur) {
    chain.unshift(cur.name);
    cur = cur.parentId ? map.get(cur.parentId) : undefined;
  }
  return chain.join(" / ");
}

/** 按 source 分桶标题 */
const GROUP_SOURCE_LABELS: Record<GroupSource, string> = {
  "oneid-dept": "部门",
  "oneid-group": "自定义组织",
  manual: "自定义组织",
  project: "项目",
};

/** 获取某 agent creator 的所有部门路径（OneID 模式，主部门排首位） */
function getCreatorDeptPaths(creator: string): Array<{ path: string; isPrimary: boolean }> {
  const user = MOCK_USERS.find((u) => u.userId === creator);
  if (!user) return [];
  const deptGroupIds = user.groupIds.filter((gid) => {
    const g = MOCK_GROUPS.find((g) => g.id === gid);
    return g?.source === "oneid-dept";
  });
  if (deptGroupIds.length === 0) return [];
  return deptGroupIds
    .map((gid) => ({
      path: getGroupPath(gid, MOCK_GROUPS),
      isPrimary: gid === user.primaryGroupId,
    }))
    .sort((a, b) => (a.isPrimary ? -1 : b.isPrimary ? 1 : 0));
}

/** 获取某 agent creator 对应的组织信息（OneID 模式，只返回一个） */
function getCreatorGroupItemOneid(creator: string): { id: string; path: string; kind: "oneid-dept" | "oneid-group" } | null {
  const user = MOCK_USERS.find((u) => u.userId === creator);
  if (!user) return null;
  // 优先取 oneid-group（自定义组织），其次取 oneid-dept（部门）
  let deptItem: { id: string; path: string; kind: "oneid-dept" | "oneid-group" } | null = null;
  for (const gid of user.groupIds) {
    const g = MOCK_GROUPS.find((g) => g.id === gid);
    if (!g) continue;
    if (g.source === "oneid-group") {
      return { id: gid, path: getGroupPath(gid, MOCK_GROUPS), kind: "oneid-group" };
    }
    if (g.source === "oneid-dept" && !deptItem) {
      deptItem = { id: gid, path: getGroupPath(gid, MOCK_GROUPS), kind: "oneid-dept" };
    }
  }
  return deptItem;
}

/** 获取某 agent creator 对应的组织信息（普通模式，只返回一个） */
function getCreatorGroupItemManual(creator: string): { id: string; path: string } | null {
  const user = MOCK_USERS_MANUAL.find((u) => u.userId === creator);
  if (!user || user.groupIds.length === 0) return null;
  const gid = user.groupIds[0];
  return { id: gid, path: getGroupPath(gid, MOCK_MANUAL_GROUPS) };
}

/**
 * 取 Agent 所属组织（OneID 模式）：
 * - 优先使用 Agent 自身固定绑定的 groupId（管理员创建时选的组，不随用户组织变化）
 * - 回退：按 creator 反查用户所属组织（兼容存量 MOCK_CLAWS）
 * kind 字段仅当走 fallback 时可确定（"oneid-dept" / "oneid-group"）；Agent 自带 groupId 时按其在 MOCK_GROUPS 中的 source 推断。
 */
function getClawGroupItemOneid(c: Claw): { id: string; path: string; kind: "oneid-dept" | "oneid-group" } | null {
  if (c.groupId) {
    const g = MOCK_GROUPS.find((x) => x.id === c.groupId);
    const path = c.groupName || (g ? getGroupPath(c.groupId, MOCK_GROUPS) : c.groupId);
    const kind: "oneid-dept" | "oneid-group" = g?.source === "oneid-dept" ? "oneid-dept" : "oneid-group";
    return { id: c.groupId, path, kind };
  }
  return getCreatorGroupItemOneid(c.creator);
}

/** 取 Agent 所属组织（普通模式）：优先 Agent 自身 groupId，回退按 creator 反查 */
function getClawGroupItemManual(c: Claw): { id: string; path: string } | null {
  if (c.groupId) {
    const path = c.groupName || getGroupPath(c.groupId, MOCK_MANUAL_GROUPS) || c.groupId;
    return { id: c.groupId, path };
  }
  return getCreatorGroupItemManual(c.creator);
}

/** 取 Agent 所属的所有组织 id（用于组织树筛选：命中 Agent 绑定的组或其祖先/自身即可） */
function getClawAllGroupIds(c: Claw, hasOneid: boolean): string[] {
  // Agent 自身绑定了 groupId → 严格用它（不再按 creator 展开）
  if (c.groupId) return [c.groupId];
  // 存量数据回退到按 creator 反查
  return getCreatorAllGroupIds(c.creator, hasOneid);
}

/** 移交弹窗左侧组织树「未分配组织」常驻节点的固定 id（放没有任何组织归属的用户） */
const TRANSFER_UNASSIGNED_GROUP_ID = "__unassigned__";

/** 获取某 agent creator 所属的所有组织 id（含子孙逻辑：选中某组织时，其用户应该被命中） */
function getCreatorAllGroupIds(creator: string, hasOneid: boolean): string[] {
  if (hasOneid) {
    const user = MOCK_USERS.find((u) => u.userId === creator);
    return user ? user.groupIds : [];
  } else {
    const user = MOCK_USERS_MANUAL.find((u) => u.userId === creator);
    return user ? user.groupIds : [];
  }
}

/** 获取节点及其所有子孙 ID */
function getGroupDescendantIds(node: GroupTreeNode): string[] {
  const ids: string[] = [node.id];
  node.children.forEach((c) => ids.push(...getGroupDescendantIds(c)));
  return ids;
}

// 注：原 InstanceGroupFilter / InstanceDepartmentFilter 及其树节点、GroupTreeSelectField / MigrateGroupTreeNode 均为死代码（无 JSX 引用），
// 实际组织/部门筛选由 GroupColumnFilter / DepartmentColumnFilter / ScopeSelect 承载，移交弹窗组织树见下方 TransferGroupTreeNode，已移除其余死代码。

// ─── 移交弹窗：组织树节点（品牌蓝选中态，单选） ──────────────────────────
function TransferGroupTreeNode({
  node, level, selected, expanded, onToggle, onSelect,
}: {
  node: GroupTreeNode; level: number; selected: string;
  expanded: Set<string>; onToggle: (id: string) => void; onSelect: (id: string) => void;
}) {
  const hasChildren = node.children && node.children.length > 0;
  const isExpanded = expanded.has(node.id);
  const isSelected = selected === node.id;
  return (
    <div>
      <div
        className={`mx-1 mb-0.5 flex items-center gap-1 h-8 pr-2 rounded-[4px] cursor-pointer transition-colors ${
          isSelected
            ? "bg-[var(--cp-brand-tint)] text-[var(--text-brand)]"
            : "text-[var(--text-body)] hover:bg-[var(--bg-grey-hover)]"
        }`}
        style={{ paddingLeft: `${level * 16 + 8}px` }}
        onClick={() => onSelect(node.id)}
      >
        {hasChildren ? (
          <button type="button" className="w-4 h-4 flex items-center justify-center shrink-0"
            onClick={(e) => { e.stopPropagation(); onToggle(node.id); }}>
            {isExpanded
              ? <ChevronDown className="w-3.5 h-3.5 text-[var(--text-weak)]" />
              : <ChevronRight className="w-3.5 h-3.5 text-[var(--text-weak)]" />}
          </button>
        ) : (
          <span className="w-4 h-4 shrink-0" />
        )}
        <span className="text-sm truncate flex-1 min-w-0">{node.name}</span>
        {isSelected && <Check className="w-4 h-4 shrink-0" />}
      </div>
      {hasChildren && isExpanded && node.children.map((child) => (
        <TransferGroupTreeNode key={child.id} node={child} level={level + 1}
          selected={selected} expanded={expanded} onToggle={onToggle} onSelect={onSelect} />
      ))}
    </div>
  );
}

export default function AgentMonitor() {
  const [, setLocation] = useLocation();
  const { hasOneid } = useAdminMode();

  // 停服态下，本页两个"选择日期" DatePicker 弹出的日历面板需保持 100% 可用。
  // 触发器已通过外层 <div data-billing-exempt> 打标；但面板经 Radix Portal 挂到<body>，
  // 不在触发器的祖先链上，因此需要单独的 MutationObserver 在面板挂载时把 exempt 打到
  // popper-wrapper 内部的 Calendar 根节点上（详见该 hook 头部注释：作用域/ 幂等 /
  // 延续原生 disabled 的保证/ 与全局 useDatePickerBillingExempt 差异说明）。
  useOpenClawMonitorCalendarBillingExempt();

  // ─── 项目列：实例↔项目 mock 确定性映射（项目池来自共享 groupStore，跟随项目增删刷新） ──
  const [projectPool, setProjectPool] = useState<UserGroup[]>(
    () => groupStore.getAll().filter((g) => g.source === "project" && g.parentId === null),
  );
  useEffect(
    () => groupStore.subscribe(() =>
      setProjectPool(groupStore.getAll().filter((g) => g.source === "project" && g.parentId === null)),
    ),
    [],
  );

  const getClawProjects = useCallback(
    (clawId: string): UserGroup[] => {
      if (projectPool.length === 0) return [];
      let h = 0;
      for (let i = 0; i < clawId.length; i++) h = (h * 31 + clawId.charCodeAt(i)) >>> 0;
      // 约 1/3 的 Agent 不挂任何项目
      if (h % 3 === 0) return [];
      const count = (h % 2) + 1; // 1~2 个项目
      const picked = new Map<string, UserGroup>();
      for (let i = 0; i < count; i++) {
        const p = projectPool[(h + i * 7) % projectPool.length];
        picked.set(p.id, p);
      }
      return Array.from(picked.values());
    },
    [projectPool],
  );
  const [claws, setClaws] = useState<Claw[]>(() => {
    // 管控端通过「+ 创建 Agent」弹窗新建的 Agent，会持久化在 localStorage(admin_created_agents)；
    // 进入页面时合并到列表顶部，使本次会话内创建的记录跨刷新仍可见。
    const adminCreated = loadAdminCreatedAgents() as unknown as Claw[];
    if (hasOneid) {
      // MOCK_CLAWS_WITH_DEPT 缺少 agentType/version/pluginVersions/tags，从 MOCK_CLAWS 补充
      const clawMap = new Map(MOCK_CLAWS.map((c) => [c.id, c]));
      const base = (MOCK_CLAWS_WITH_DEPT as any[]).map((d) => {
        const ref = clawMap.get(d.id);
        return {
          ...d,
          agentType: ref?.agentType ?? "OpenClaw",
          version: ref?.version ?? "2026.3.28",
          pluginVersions: ref?.pluginVersions ?? DEFAULT_PLUGIN_VERSIONS,
          standards: ref?.standards,
          tags: ref?.tags,
        } as Claw;
      });
      return [...adminCreated, ...base].sort((a, b) => b.createTime.localeCompare(a.createTime));
    }
    return [...adminCreated, ...MOCK_CLAWS].sort((a, b) => b.createTime.localeCompare(a.createTime));
  });

  // 订阅 adminAgentStore 变更：当其他视图（或自身 onCreated）写入新记录时，自动合并到 claws
  // 避免「写入了但页面看不到」的情况；同时兼容未来从其他入口创建 Agent 的场景。
  useEffect(() => {
    const unsubscribe = onAdminCreatedAgentsChange(() => {
      const adminCreated = loadAdminCreatedAgents() as unknown as Claw[];
      setClaws((prev) => {
        // 用 id 去重：保留 prev 中所有非"管控端创建"的（即原 MOCK 数据），再合并最新缓存
        const cachedIds = new Set(adminCreated.map((c) => c.id));
        const mockOnly = prev.filter((c) => !cachedIds.has(c.id));
        return [...adminCreated, ...mockOnly].sort((a, b) => b.createTime.localeCompare(a.createTime));
      });
    });
    return unsubscribe;
  }, []);
  const [search, setSearch] = useState("");
  // 搜索类型切换：名称 / ID / 用户ID / 标签键 / 标签值
  const [searchType, setSearchType] = useState<'name' | 'id' | 'creator' | 'tagKey' | 'tag'>('name');
  const [searchTagKeys, setSearchTagKeys] = useState<string[]>([]);
  const [searchTagKeyForValue, setSearchTagKeyForValue] = useState<string>('');
  const [searchTagValues, setSearchTagValues] = useState<string[]>([]);
  const [tagSearchPopoverOpen, setTagSearchPopoverOpen] = useState(false);
  const [tagKeySearchText, setTagKeySearchText] = useState('');
  const [tagValueSearchText, setTagValueSearchText] = useState('');
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [page, setPage] = useState(1);
  const [refreshing, setRefreshing] = useState(false);
  // 克隆 Agent 为镜像：当前操作的 Agent（更多菜单触发，null = 弹窗关闭）
  const [cloneImageTarget, setCloneImageTarget] = useState<Claw | null>(null);

  // ─── 更新提示气泡：克隆 Agent 为镜像（plan-20260818-clone-image v1，duration 14 天）───
  // 首次进入页面展示，指向首行操作列「更多」；关闭或曝光 2 次后不再展示，2026-09-01 到期下线
  const CLONE_BUBBLE_UPDATE_ID = "clone-agent-image-20260818";
  const CLONE_BUBBLE_KEY = buildPersistenceKey("point-bubble", CLONE_BUBBLE_UPDATE_ID);
  const CLONE_BUBBLE_BEHAVIOR = resolveBehavior("point-bubble", { expiresAt: "2026-09-01T00:00:00+08:00" });
  const [cloneBubbleOpen, setCloneBubbleOpen] = useState(false);
  useEffect(() => {
    const expiresAt = CLONE_BUBBLE_BEHAVIOR.expiresAt;
    if (expiresAt && new Date(expiresAt).getTime() < Date.now()) return; // 超过 14 天窗口，不再展示
    if (isDismissed(CLONE_BUBBLE_KEY, CLONE_BUBBLE_BEHAVIOR)) return;   // 已关闭或曝光满 2 次
    setCloneBubbleOpen(true);
    markExposure(CLONE_BUBBLE_KEY);
    trackOnboarding("onboarding_impression", {
      updateId: CLONE_BUBBLE_UPDATE_ID,
      component: "point-bubble",
      layer: "element",
      scenario: "2.1",
      endpoint: "admin",
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const closeCloneBubble = () => {
    setCloneBubbleOpen(false);
    markDismissed(CLONE_BUBBLE_KEY);
    trackOnboarding("onboarding_dismiss", {
      updateId: CLONE_BUBBLE_UPDATE_ID,
      component: "point-bubble",
      endpoint: "admin",
    });
  };
  // URL 参数筛选：从存量处理弹窗跳转过来时携带 ?filter=pending-delete&ids=xxx,yyy
  const [pendingDeleteIds, setPendingDeleteIds] = useState<Set<string>>(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get("filter") === "pending-delete") {
      const ids = params.get("ids")?.split(",").filter(Boolean) ?? [];
      return new Set(ids);
    }
    return new Set();
  });
  const [departmentFilter, setDepartmentFilter] = useState("");
  const [groupFilter, setGroupFilter] = useState("");
  const ALL_AGENT_TYPES = ALL_AGENT_TYPE_FILTER_KEYS;
  const [agentTypeFilter, setAgentTypeFilter] = useState<Set<AgentTypeFilterKey>>(new Set(ALL_AGENT_TYPES));

  // 列头筛选弹窗状态
  const [deptColFilterOpen, setDeptColFilterOpen] = useState(false);
  const [groupColFilterOpen, setGroupColFilterOpen] = useState(false);
  const [typeColFilterOpen, setTypeColFilterOpen] = useState(false);
  const [tempTypeFilter, setTempTypeFilter] = useState<Set<AgentTypeFilterKey>>(new Set());

  // 状态卡片筛选
  const [activeCardFilter, setActiveCardFilter] = useState<"all" | "running" | "shutdown" | "other">("all");

  // 状态列筛选
  const ALL_STATUSES: ClawStatus[] = ["creating", "createFail", "running", "loading", "loadFail", "shutdown", "maintaining", "pending", "upgrading"];
  const [showStatusFilter, setShowStatusFilter] = useState(false);
  const [selectedStatuses, setSelectedStatuses] = useState<Set<ClawStatus>>(new Set(ALL_STATUSES));
  // 实例规格列筛选
  const [showSpecFilter, setShowSpecFilter] = useState(false);
  const [selectedSpecs, setSelectedSpecs] = useState<Set<string>>(new Set(ADJUST_SPEC_OPTIONS.map(o => o.value)));
  const [pendingSpecs, setPendingSpecs] = useState<Set<string>>(new Set(ADJUST_SPEC_OPTIONS.map(o => o.value)));
  // 系统盘容量列筛选
  const [showDiskFilter, setShowDiskFilter] = useState(false);
  const [diskCapacityFilter, setDiskCapacityFilter] = useState<{ cond: "<" | "=" | ">"; value: string } | null>(null);
  const [tempDiskCond, setTempDiskCond] = useState<"<" | "=" | ">">("<");
  const [tempDiskValue, setTempDiskValue] = useState("");

  // 表格横向滚动 — 仅保留祖先 flex 容器 min-width:0 兜底，固定列/阴影由全局 Table 组件提供
  const tableScrollRef = useRef<HTMLDivElement>(null);
  const [isTableScrolled, setIsTableScrolled] = useState(false);
  useEffect(() => {
    const el = tableScrollRef.current;
    if (!el) return;
    const onScroll = () => setIsTableScrolled(el.scrollLeft > 0);
    el.addEventListener('scroll', onScroll, { passive: true });
    onScroll();
    // 给 flex 父容器加 min-width:0 防止 table 撑开页面
    let parent = el.parentElement;
    while (parent) {
      const style = getComputedStyle(parent);
      if (style.display === 'flex' || style.display === 'inline-flex') {
        // 给 flex 容器的子元素加 min-width:0，避免横向大表撑开页面。
        // 不写 overflow，AdminLayout 的纵向滚动容器需要保留 overflow-y:auto。
        const children = parent.children;
        for (let i = 0; i < children.length; i++) {
          const child = children[i] as HTMLElement;
          if (child.tagName === 'MAIN' || getComputedStyle(child).flex !== '0 1 auto') {
            child.style.minWidth = '0';
          }
        }
        break;
      }
      parent = parent.parentElement;
    }
    return () => el.removeEventListener('scroll', onScroll);
  }, []);

  // 操作对话框
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [shutdownTarget, setShutdownTarget] = useState<string | null>(null);
  // 批量关机/开机弹窗开关
  const [showBatchShutdownDialog, setShowBatchShutdownDialog] = useState(false);
  const [showBatchPowerOnDialog, setShowBatchPowerOnDialog] = useState(false);
  const [reinstallTarget, setReinstallTarget] = useState<string | null>(null);
  const [reinstallInput, setReinstallInput] = useState("");
  const [deleteInput, setDeleteInput] = useState("");
  const [showBatchDeleteDialog, setShowBatchDeleteDialog] = useState(false);
  const [batchDeleteInput, setBatchDeleteInput] = useState("");
  const [restartConfirm, setRestartConfirm] = useState<{ id: string; name: string } | null>(null);
  const [restartFullServer, setRestartFullServer] = useState(false);
  const [adjustConfigContext, setAdjustConfigContext] = useState<
    | { mode: "single"; claw: Claw }
    | { mode: "batch"; claws: Claw[] }
    | null
  >(null);

  // 组织处理弹窗
  const [showGroupMigrateDialog, setShowGroupMigrateDialog] = useState(false);
  const [showMigrateConfigDiff, setShowMigrateConfigDiff] = useState(false);
  const [showGroupTransferDialog, setShowGroupTransferDialog] = useState(false);
  const [showTransferConfigDiff, setShowTransferConfigDiff] = useState(false);
  const [groupMigrateTarget, setGroupMigrateTarget] = useState("");
  const [groupTransferTarget, setGroupTransferTarget] = useState("");
  const [groupTransferTargetGroup, setGroupTransferTargetGroup] = useState("");
  const [transferPickerGroupId, setTransferPickerGroupId] = useState("");
  const [transferTreeExpanded, setTransferTreeExpanded] = useState<Set<string>>(new Set());
  // 移交弹窗：选择接手用户的统一搜索（按 用户ID / 组织名称，默认 用户ID）
  const [transferSearchField, setTransferSearchField] = useState<"userId" | "groupName">("userId");
  const [transferSearchKeyword, setTransferSearchKeyword] = useState("");

  // 批量更新
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [showBatchUpgradeDialog, setShowBatchUpgradeDialog] = useState(false);

  // 批量配置模型
  // 目标模型对所有所选 Agent 统一生效；支持添加多个模型（首个为主模型，其余为备选模型）
  type BatchConfigModel = { id: number; providerLabel: string; versionLabel: string; primary: boolean };
  const [showBatchConfigModelDialog, setShowBatchConfigModelDialog] = useState(false);
  // 厂商/版本不在 openBatchConfigModelDialog 里重置，保留用户上次批量配置时的选择，减少重复操作
  const [batchModelProvider, setBatchModelProvider] = useState(MODEL_PROVIDERS[0].value);
  const [batchModelVersion, setBatchModelVersion] = useState(MODEL_PROVIDERS[0].versions[0].value);
  const [batchAppliedModels, setBatchAppliedModels] = useState<BatchConfigModel[]>([]);
  const [batchModelIdCounter, setBatchModelIdCounter] = useState(1);
  // 「创建 Agent」两步弹窗
  const [showCreateAgent, setShowCreateAgent] = useState(false);
  // 「版本更新记录」侧边栏（点击新版本推送提醒打开）
  const [showUpdateRecordsDrawer, setShowUpdateRecordsDrawer] = useState(false);
  // 批量升级失败结果弹窗
  const [showUpgradeResultDialog, setShowUpgradeResultDialog] = useState(false);
  const [upgradeFailedAgents, setUpgradeFailedAgents] = useState<{ name: string; instanceId: string; agentType: string; reason: string }[]>([]);

  // 命令下发弹窗（取代旧抽屉）
  // dispatchPresetIds = null 表示 Dialog 关闭；非 null（即使是空数组）表示打开。
  // 通过工具栏「命令下发」按钮触发：勾选了实例则预填，否则为空，进入「先选命令再选实例」流程。
  const [dispatchPresetIds, setDispatchPresetIds] = useState<string[] | null>(null);

  // 调整配置弹窗内部状态：调整类型（默认「实例规格升配」）+ 目标规格 / 目标容量（GB）；仅前端校验，无真实提交
  const [adjustConfigType, setAdjustConfigType] = useState<"spec" | "capacity">("spec");
  const [adjustTargetSpec, setAdjustTargetSpec] = useState<string | undefined>(undefined);
  const [adjustTargetCapacity, setAdjustTargetCapacity] = useState<string>("");
  // 弹窗实例列表右上角筛选：仅影响列表展示，不影响可调整统计与提交范围；默认「全部」
  const [adjustListFilter, setAdjustListFilter] = useState<"all" | "adjustable" | "unadjustable">("all");
  // 弹窗内部步骤：config=调整配置，confirm=影响确认；打开/关闭均重置为 config
  const [adjustConfigStep, setAdjustConfigStep] = useState<"config" | "confirm">("config");
  // 系统盘容量扩容方式：online=在线扩容，offline=关机后扩容；默认在线扩容
  const [adjustExpandMode, setAdjustExpandMode] = useState<"online" | "offline">("online");
  // 前端 mock 操作状态：仅记录各实例「调整配置」的运行结果；不写 MOCK_CLAWS 主数据、不真变配/不调接口
  // 为演示，初始化时让首个云端实例 processing、第二个云端实例 failed（仅写入此 map，不动 main mock）
  type AdjustOpState = { status: "processing" | "failed"; errorCode?: string; adjustType?: "spec" | "capacity"; targetSpec?: string; targetDiskSize?: number };
  const [adjustOperationMap, setAdjustOperationMap] = useState<Record<string, AdjustOpState>>(() => {
    const cloud = MOCK_CLAWS.filter((c) => c.agentType !== "LocalAgent");
    const init: Record<string, AdjustOpState> = {};
    if (cloud[0]) init[cloud[0].id] = { status: "processing", adjustType: "spec", targetSpec: "Ai2.LARGE8" };
    if (cloud[1]) init[cloud[1].id] = { status: "failed", errorCode: "InternalError", adjustType: "capacity", targetDiskSize: 200 };
    return init;
  });
  const [adjustSubmitting, setAdjustSubmitting] = useState(false);
  // 校验阶段：idle（未校验）→ validating（校验中）→ validated（校验完成）
  type ValidationPhase = "idle" | "validating" | "validated";
  const [validationPhase, setValidationPhase] = useState<ValidationPhase>("idle");
  // 点击 ✕ 清空目标规格后，递增该 key 强制 Select 卸载重挂，确保 UI 回到 placeholder
  const [specResetKey, setSpecResetKey] = useState(0);
  // 调整任务成功后的资源回写：clawId → 更新后的 spec/diskSize
  const [resourceUpdates, setResourceUpdates] = useState<Record<string, { spec?: string; diskSize?: number }>>({});
  // 校验中状态 + 缓存校验结果；未触发校验时所有结果为 "—"
  const [adjustValidating, setAdjustValidating] = useState(false);
  const [adjustResults, setAdjustResults] = useState<Record<string, AdjustValidationResult>>({});
  const validationTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const clearAdjustTimers = () => {
    if (validationTimerRef.current) { clearTimeout(validationTimerRef.current); validationTimerRef.current = null; }
    if (debounceTimerRef.current) { clearTimeout(debounceTimerRef.current); debounceTimerRef.current = null; }
    if (autoCompleteTimerRef.current) { clearTimeout(autoCompleteTimerRef.current); autoCompleteTimerRef.current = null; }
  };
  const autoCompleteTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // 步骤 2 强制关机确认 checkbox（仅实例规格升配 + 存在运行中实例时展示）
  const [forceShutdownConfirmed, setForceShutdownConfirmed] = useState(false);
  // 打开弹窗时重置内部选择（默认为「实例规格升配」；若单实例且当前已是最高规格则默认切到容量扩容）
  useEffect(() => {
    setAdjustConfigStep("config");
    if (adjustConfigContext) {
      const isSingle = adjustConfigContext.mode === "single";
      const curSpecName = isSingle ? getInstanceSpecInfo(adjustConfigContext.claw).spec : "";
      const atMaxSpec = isSingle && SPEC_RANK[curSpecName] === 4;
      setAdjustConfigType(atMaxSpec ? "capacity" : "spec");
      setAdjustTargetSpec(undefined);
      setAdjustTargetCapacity("");
      setAdjustExpandMode("online");
      setAdjustListFilter("all");
      setAdjustValidating(false);
      setAdjustResults({});
      clearAdjustTimers();
      setForceShutdownConfirmed(false);
    }
    setValidationPhase("idle");
  }, [adjustConfigContext]);
  // 进入步骤 2 时重置强制关机确认 checkbox
  useEffect(() => {
    if (adjustConfigStep === "confirm")     setForceShutdownConfirmed(false);
    setAdjustSubmitting(false);
  }, [adjustConfigStep]);
  // 单实例当前规格等级（仅 single 模式有效；batch/null 返回 null，不限制）
  const singleInstanceSpecRank: number | null = (() => {
    if (!adjustConfigContext || adjustConfigContext.mode !== "single") return null;
    return SPEC_RANK[getInstanceSpecInfo(adjustConfigContext.claw).spec] ?? 0;
  })();
  const singleInstanceAtMaxSpec = singleInstanceSpecRank === 3;
  // 目标配置变化时：清空校验结果，回到 idle 状态
  const resetValidationState = () => {
    clearAdjustTimers();
    setAdjustResults({});
    setAdjustValidating(false);
    setValidationPhase("idle");
    setAdjustListFilter("all");
  };
  // 点击「校验」后：debounce + simulate → 缓存校验结果
  useEffect(() => {
    if (validationPhase === "idle" || validationPhase === "validated") {
      return;
    }
    // validating：清空旧结果，开始新校验
    clearAdjustTimers();
    setAdjustResults({});
    setAdjustValidating(true);
    if (adjustConfigType === "spec") {
      validationTimerRef.current = setTimeout(() => {
        const mapping: Record<string, AdjustValidationResult> = {};
        for (const c of adjustTargetClaws) {
          mapping[c.id] = validateAdjust(c, adjustConfigType, adjustTargetSpec, adjustTargetCapacityNum);
        }
        setAdjustResults(mapping);
        setAdjustValidating(false);
        setValidationPhase("validated");
      }, 2500);
    } else {
      debounceTimerRef.current = setTimeout(() => {
        validationTimerRef.current = setTimeout(() => {
          const mapping: Record<string, AdjustValidationResult> = {};
          for (const c of adjustTargetClaws) {
            mapping[c.id] = validateAdjust(c, adjustConfigType, adjustTargetSpec, adjustTargetCapacityNum);
          }
          setAdjustResults(mapping);
          setAdjustValidating(false);
          setValidationPhase("validated");
        }, 2500);
      }, 500);
    }
    return clearAdjustTimers;
  }, [validationPhase]);
  useEffect(() => {
    resetValidationState();
  }, [adjustConfigType, adjustTargetSpec, adjustTargetCapacity]);

  // 配置默认标签
  interface TencentTag { key: string; value: string; scope?: 'all' | 'groups'; groupIds?: string[]; }
  // 标签键 -> 可选值列表（模拟腾讯云标签库）
  const tagKeyValues: Record<string, string[]> = {
    'qcs:tag:thpc:node:creator':      ['alice', 'bob', 'charlie'],
    'qcs:tag:thpc:node:clusterId':    ['cluster-001', 'cluster-002'],
    'qcs:tag:thpc:node:nodeId':       ['node-a1', 'node-b2', 'node-c3'],
    'qcs:tag:thpc:workspace:creator': ['alice', 'dave'],
    'kaijian':                        ['kaijian', 'test'],
    'acs:tag:createdby':              ['system', 'user'],
    'tke_managed_by':                 ['tke', 'manual'],
    'niumengtao':                     ['体验', '正式', '测试'],
    '所属产品':                        ['gpulab', 'openclaw', 'tke'],
    'env':                            ['production', 'staging', 'dev'],
    '负责人':                          ['alice', 'bob', 'charlie'],
    '业务线':                          ['AI', 'Platform', 'Infra'],
  };
  const tagKeys = Object.keys(tagKeyValues);
  const [showTagConfigDialog, setShowTagConfigDialog] = useState(false);
  // 已确认的标签列表（key-value 对）
  const [selectedTags, setSelectedTags] = useState<TencentTag[]>([]);
  // 弹窗内临时状态：已加入队列的待保存标签
  const [pendingTags, setPendingTags] = useState<TencentTag[]>([]);
  const [keySearchText, setKeySearchText] = useState('');
  const [openKeyRow, setOpenKeyRow] = useState<number | null>(null);
  const [openValueRow, setOpenValueRow] = useState<number | null>(null);


  // 版本列筛选

  const isLocalConnected = (claw: Claw) => claw.agentType !== "LocalAgent" || claw.localConnectionStatus !== "disconnected";

  const getLocalConnectionMeta = (claw: Claw): { label: string; variant: "green" | "gray" } => (
    isLocalConnected(claw)
      ? { label: "已接入", variant: "green" }
      : { label: "未接入", variant: "gray" }
  );

  // 判断某实例是否可更新（仅云端运行中 Agent）
  const isUpgradable = (claw: Claw) => claw.agentType !== "LocalAgent" && claw.status === "running";

  const handleSelectAll = (checked: boolean) => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      // 全选勾选当前筛选结果的所有页所有实例，不限状态
      if (checked) { allFilteredIds.forEach(id => next.add(id)); }
      else { allFilteredIds.forEach(id => next.delete(id)); }
      return next;
    });
  };

  const handleSelectOne = (id: string, checked: boolean) => {
    const target = claws.find((c) => c.id === id);
    if (target?.agentType === "LocalAgent") {
      toast.info("本地 Agent 暂不支持批量处理、命令下发等云端能力");
      return;
    }
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (checked) next.add(id); else next.delete(id);
      return next;
    });
  };

  // agentType 映射：Claw 中的 agentType → ImageManagement 中的 agentType
  const CLAW_TO_IMAGE_AGENT_TYPE: Record<string, string> = {
    'OpenClaw':    'OpenClaw',
    'Hermes':      'HermesAgent',
    'LightclawACE': 'LightClawACE',
    'LocalAgent':  'LocalAgent',
  };
  const confirmBatchUpgrade = () => {
    const ids = Array.from(selectedIds);
    const selectedClaws = claws.filter(c => ids.includes(c.id));
    // 读取镜像管理中的生效镜像及版本（admin_images_v3）
    const targetMap = new Map<string, { version: string; imageName: string }>();
    try {
      const raw = localStorage.getItem("admin_images_v3");
      const imgs: { agentType: string; agentVersion: string; active: boolean; name: string }[] =
        raw ? JSON.parse(raw) : [];
      for (const img of imgs) {
        if (img.active && img.agentType && img.agentVersion) {
          targetMap.set(img.agentType, { version: img.agentVersion, imageName: img.name });
        }
      }
    } catch { /* ignore */ }
    // 组织：可升级 vs 无法升级（无生效镜像 或 目标版本 ≤ 当前版本）
    const failed: { name: string; instanceId: string; agentType: string; reason: string }[] = [];
    const upgradableIds: string[] = [];
    for (const c of selectedClaws) {
      const imageAgentType = CLAW_TO_IMAGE_AGENT_TYPE[c.agentType] ?? c.agentType;
      const target = targetMap.get(imageAgentType);
      if (!target) {
        failed.push({
          name: c.name, instanceId: c.instanceId, agentType: c.agentType,
          reason: `当前没有生效的 ${AGENT_TYPE_DISPLAY[c.agentType] ?? c.agentType} 镜像`,
        });
      } else if (compareVersion(c.version || "", target.version) >= 0) {
        const reason = compareVersion(c.version || "", target.version) === 0
          ? `已是最新版本 (${target.version})`
          : `不支持降级：当前 ${c.version} → 目标 ${target.version}`;
        failed.push({
          name: c.name, instanceId: c.instanceId, agentType: c.agentType, reason,
        });
      } else {
        upgradableIds.push(c.id);
      }
    }
    setShowBatchUpgradeDialog(false);
    if (failed.length > 0) {
      setUpgradeFailedAgents(failed);
      setShowUpgradeResultDialog(true);
    }
    if (upgradableIds.length > 0) {
      setClaws(prev => prev.map(c => upgradableIds.includes(c.id) ? { ...c, status: 'upgrading' as ClawStatus } : c));
      setSelectedIds(new Set());
      toast.success(`已开始升级 ${upgradableIds.length} 个实例`);
    } else if (failed.length === 0) {
      setSelectedIds(new Set());
    }
  };

  // 详情抽屉
  const [showDetailDrawer, setShowDetailDrawer] = useState(false);
  const [drawerLoading, setDrawerLoading] = useState(false);

  // 点击组织列「橙点」以「对比模式」打开 Agent 详情抽屉
  const [compareMode, setCompareMode] = useState(false);
  // 「筛选异常组织」：仅展示带红点（用户不在该组织）或橙点（配置不符合）的实例
  const [anomalyFilter, setAnomalyFilter] = useState(false);

  // 监控抽屉
  const [showMonitorDrawer, setShowMonitorDrawer] = useState(false);
  const [selectedClaw, setSelectedClaw] = useState<Claw | null>(null);

  // ─── 龙虾医生状态 ──────────────────────────────────────────────────────────
  const [doctorActive, setDoctorActive] = useState(false);
  const [doctorOccupied, setDoctorOccupied] = useState(false); // 是否被他人诊断中

  // 查询当前选中 Agent 的诊断状态（从服务端）
  //
  // 只依赖 selectedClaw：切换 Agent 时才查询一次，用于"自动恢复"场景——即
  // 页面刷新/切回该 Agent 时，如果服务端仍有自己发起的诊断则自动激活面板。
  //
  // 禁止依赖 doctorActive：否则会出现如下 bug ——
  //   ① 用户点结束诊断 → doctorActive 变 false
  //   ② useEffect rerun → 立刻向服务端查询
  //   ③ 若 endDiagnosis 未完成同步（是fire-and-forget，见AdminDoctorChatPanel
  //      的 doEndSession）→ 服务端仍返回 active && isMine
  //   ④ setDoctorActive(true) 再次激活 → AdminDoctorChatPanel 挂载 effect
  //      检测到无进行中诊断（此时服务端已同步完成）→ handleStartDiagnosisClick
  //      → 结束诊断后又弹出"开始诊断"弹窗，视觉上没有回到最初静态视图。
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!selectedClaw) return;
    let cancelled = false;
    const checkDiagStatus = async () => {
      const callerId = getCallerId("admin");
      const result = await queryDiagnosisStatus(selectedClaw.instanceId, callerId);
      if (cancelled) return;
      if (result.active && result.isMine) {
        // 自己发起的诊断仍在进行中 → 自动恢复面板，让内部组件恢复会话
        setDoctorOccupied(false);
        setDoctorActive(true);
      } else {
        setDoctorOccupied(result.active && !result.isMine);
      }
    };
    checkDiagStatus();
    return () => { cancelled = true; };
  }, [selectedClaw]);

  const [clsEnabled, setClsEnabled] = useState(() => {
    const stored = localStorage.getItem("globalClsEnabled");
    return stored === "true";
  });

  // 监听 localStorage 变化，实现跨页面同步
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "globalClsEnabled") {
        setClsEnabled(e.newValue === "true");
      }
    };
    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, []);

  // 计算统计数据
  const countByStatus = (status: ClawStatus | ClawStatus[]) => {
    const statuses = Array.isArray(status) ? status : [status];
    return claws.filter(c => statuses.includes(c.status)).length;
  };

  const totalCount = claws.length;
  const runningCount = countByStatus("running");
  const shutdownCount = countByStatus("shutdown");
  const otherCount = countByStatus(["creating", "loading", "createFail", "loadFail", "maintaining", "pending", "upgrading"]);

  // 根据卡片筛选限制状态列筛选的可选项
  const getAvailableStatuses = (): ClawStatus[] => {
    switch (activeCardFilter) {
      case "running": return ["running"];
      case "shutdown": return ["shutdown"];
      case "other": return ["creating", "loading", "createFail", "loadFail", "maintaining", "pending", "upgrading"];
      case "all": return ALL_STATUSES;
    }
  };

  const handleCardFilterChange = (filter: "all" | "running" | "shutdown" | "other") => {
    setActiveCardFilter(filter);
    setPage(1);
    // 重置状态列筛选为当前卡片允许的全选状态
    const availableStatuses: ClawStatus[] = (() => {
      switch (filter) {
        case "running": return ["running"];
        case "shutdown": return ["shutdown"];
        case "other": return ["creating", "loading", "createFail", "loadFail", "maintaining", "pending", "upgrading"];
        case "all": return ALL_STATUSES;
      }
    })();
    setSelectedStatuses(new Set(availableStatuses));
  };

  const handleStatusFilterChange = (status: ClawStatus, checked: boolean) => {
    const newStatuses = new Set(selectedStatuses);
    if (checked) {
      newStatuses.add(status);
    } else {
      newStatuses.delete(status);
    }
    setSelectedStatuses(newStatuses);
  };

  const handleStatusFilterReset = () => {
    const available = getAvailableStatuses();
    setSelectedStatuses(new Set(available));
  };

  const handleStatusFilterConfirm = () => {
    setShowStatusFilter(false);
    setPage(1);
  };

  // 实例规格筛选 handlers（confirm 模式）
  const handleSpecFilterToggle = (spec: string) => {
    setPendingSpecs((prev) => {
      const next = new Set(prev);
      if (next.has(spec)) next.delete(spec); else next.add(spec);
      return next;
    });
  };
  // 点击「全部」：全选态 → 清空；非全选态 → 选中全部
  const handleSpecFilterToggleAll = () => {
    setPendingSpecs((prev) => {
      if (prev.size === ADJUST_SPEC_OPTIONS.length) return new Set();
      return new Set(ADJUST_SPEC_OPTIONS.map(o => o.value));
    });
  };
  const handleSpecFilterReset = () => {
    setPendingSpecs(new Set(ADJUST_SPEC_OPTIONS.map(o => o.value)));
  };
  const handleSpecFilterConfirm = () => {
    setSelectedSpecs(pendingSpecs);
    setShowSpecFilter(false);
    setPage(1);
  };
  const handleSpecFilterCancel = () => {
    setPendingSpecs(new Set(selectedSpecs));
    setShowSpecFilter(false);
  };

  // 系统盘容量筛选 handlers（浮层内用 temp 状态，确认后才应用）
  const handleDiskFilterReset = () => {
    setTempDiskCond("<");
    setTempDiskValue("");
    setDiskCapacityFilter(null);
    setShowDiskFilter(false);
    setPage(1);
  };
  const handleDiskFilterConfirm = () => {
    const v = tempDiskValue.trim();
    if (v === "") {
      setDiskCapacityFilter(null);
    } else {
      const num = Number(v);
      if (Number.isInteger(num) && num > 0) {
        setDiskCapacityFilter({ cond: tempDiskCond, value: v });
      }
    }
    setShowDiskFilter(false);
    setPage(1);
  };

  // 筛选逻辑
  // 如果有 pending-delete 筛选（从存量处理弹窗跳转），优先只显示对应实例
  const pendingFiltered = pendingDeleteIds.size > 0
    ? claws.filter((c) => pendingDeleteIds.has(c.id))
    : claws;

  const timeFiltered = pendingFiltered.filter((c) => {
    const matchFrom = !dateFrom || c.createTime >= dateFrom;
    const matchTo = !dateTo || c.createTime <= dateTo;
    return matchFrom && matchTo;
  });

  const searchFiltered = timeFiltered.filter((c) => {
    // 按选定的搜索类型过滤
    if (searchType === 'name') {
      return !search || c.name.includes(search);
    }
    if (searchType === 'id') {
      return !search || c.instanceId.includes(search);
    }
    if (searchType === 'creator') {
      return !search || c.creator.includes(search);
    }
    if (searchType === 'tagKey') {
      if (searchTagKeys.length === 0) return true;
      return searchTagKeys.some(k => (c.tags || []).some(t => t.key === k));
    }
    if (searchType === 'tag') {
      if (!searchTagKeyForValue) return true;
      if (searchTagValues.length === 0) {
        return (c.tags || []).some(t => t.key === searchTagKeyForValue);
      }
      return (c.tags || []).some(t => t.key === searchTagKeyForValue && searchTagValues.includes(t.value));
    }
    return true;
  });

  // 部门筛选（OneID 模式）
  const findDeptAndChildren = (nodes: DepartmentNode[], targetId: string): string[] => {
    const ids: string[] = [];
    const collect = (node: DepartmentNode) => {
      ids.push(node.id);
      node.children?.forEach(collect);
    };
    const find = (list: DepartmentNode[]): boolean => {
      for (const n of list) {
        if (n.id === targetId) { collect(n); return true; }
        if (n.children && find(n.children)) return true;
      }
      return false;
    };
    find(nodes);
    return ids;
  };

  const deptFiltered = hasOneid ? searchFiltered.filter((c) => {
    if (!departmentFilter) return true;
    const allowedIds = findDeptAndChildren(MOCK_DEPARTMENTS, departmentFilter);
    return c.departmentId ? allowedIds.includes(c.departmentId) : false;
  }) : searchFiltered;

  // 组织筛选
  const groupFiltered = deptFiltered.filter((c) => {
    if (!groupFilter) return true;
    const currentGroups = hasOneid ? MOCK_GROUPS : MOCK_MANUAL_GROUPS;
    const trees = buildGroupTree(currentGroups);
    // 找到选中组织节点及其所有子孙 id
    const findNode = (nodes: GroupTreeNode[], id: string): GroupTreeNode | null => {
      for (const n of nodes) {
        if (n.id === id) return n;
        const hit = findNode(n.children, id);
        if (hit) return hit;
      }
      return null;
    };
    const targetNode = findNode(trees, groupFilter);
    if (!targetNode) return true;
    const allowedGroupIds = new Set(getGroupDescendantIds(targetNode));
    // 组织归属：优先取 Agent 自身固定绑定的分组，回退按 creator 反查（存量 mock 兼容）
    const clawGroupIds = getClawAllGroupIds(c, hasOneid);
    return clawGroupIds.some((gid) => allowedGroupIds.has(gid));
  });

  // Agent 类型筛选
  const typeFiltered = groupFiltered.filter((c) => {
    if (agentTypeFilter.size === 0 || agentTypeFilter.size === ALL_AGENT_TYPES.length) return true;
    return agentTypeFilter.has(getClawTypeFilterKey(c));
  });

  const cardFiltered = typeFiltered.filter((c) => {
    switch (activeCardFilter) {
      case "running": return c.status === "running";
      case "shutdown": return c.status === "shutdown";
      case "other": return ["creating", "loading", "createFail", "loadFail", "maintaining", "pending", "upgrading"].includes(c.status);
      case "all": return true;
    }
  });

  const statusFiltered = cardFiltered.filter((c) => {
    if (selectedStatuses.size === 0 || selectedStatuses.size === ALL_STATUSES.length) return true;
    return selectedStatuses.has(c.status);
  });


  // 实例规格筛选：本地 Agent 不参与（始终通过）
  const specFiltered = statusFiltered.filter((c) => {
    if (c.agentType === "LocalAgent") return true;
    if (selectedSpecs.size === 0 || selectedSpecs.size === ADJUST_SPEC_OPTIONS.length) return true;
    const info = getInstanceSpecInfo(c);
    return selectedSpecs.has(info.spec);
  });

  // 系统盘容量筛选：本地 Agent 不参与
  const diskFiltered = specFiltered.filter((c) => {
    if (!diskCapacityFilter || !diskCapacityFilter.value) return true;
    if (c.agentType === "LocalAgent") return true;
    const num = Number(diskCapacityFilter.value);
    if (isNaN(num) || num <= 0 || !Number.isInteger(num)) return true;
    const info = getInstanceSpecInfo(c);
    switch (diskCapacityFilter.cond) {
      case "<": return info.systemDiskCapacity < num;
      case "=": return info.systemDiskCapacity === num;
      case ">": return info.systemDiskCapacity > num;
    }
    return true;
  });

  // 「筛选异常组织」：仅保留带红点（用户不在该组织）或橙点（配置不符合）的实例
  const anomalyFiltered = anomalyFilter
    ? diskFiltered.filter((c) => isOrgAnomalyInstance(c.id))
    : diskFiltered;

  const versionFiltered = anomalyFiltered;

  const totalPages = Math.max(1, Math.ceil(versionFiltered.length / PAGE_SIZE));
  const safePage = Math.min(page, totalPages);
  const paginated = versionFiltered.slice((safePage - 1) * PAGE_SIZE, safePage * PAGE_SIZE);

  // 当前页是否存在任何带标签的实例；若全无标签则整列（表头 + 单元格）隐藏
  const hasAnyTagColumn = paginated.some((c) => c.tags && c.tags.length > 0);

  // 当前页所有实例 id
  const pageIds = paginated.filter((c) => c.agentType !== "LocalAgent").map(c => c.id);
  // 全筛选结果的所有 id（全选范围）
  const allFilteredIds = versionFiltered.filter((c) => c.agentType !== "LocalAgent").map(c => c.id);
  // 全选状态：当前筛选结果所有实例全部被勾选
  const isAllSelected = allFilteredIds.length > 0 && allFilteredIds.every(id => selectedIds.has(id));
  // 部分勾选：有且仅有部分实例被勾选（不显示 indeterminate，直接显示未勾选）
  const isIndeterminate = false;

  // 批量更新按钮禁用逻辑
  const selectedCount = selectedIds.size;
  const selectedClaws = claws.filter(c => selectedIds.has(c.id));
  const selectedCloudClaws = selectedClaws.filter(c => c.agentType !== "LocalAgent");
  const selectedCloudCount = selectedCloudClaws.length;
  // 调整配置弹窗：目标容量（GB）解析 + 待校验实例列表（仅云端 Agent，本地 Agent 永不进入）
  const adjustTargetCapacityNum = adjustTargetCapacity.trim() === "" ? undefined : Number(adjustTargetCapacity);
  const adjustTargetClaws = useMemo(() => {
    if (!adjustConfigContext) return [];
    if (adjustConfigContext.mode === "single") return [adjustConfigContext.claw];
    return adjustConfigContext.claws;
  }, [adjustConfigContext]);
  // 实际可调整实例（校验结果为「可调整」）；仅这些实例进入影响确认页
  const adjustAdjustableClaws = useMemo(() => {
    return adjustTargetClaws.filter(
      (c) => adjustResults[c.id]?.label === "可调整",
    );
  }, [adjustTargetClaws, adjustResults]);
  // 可调整实例中是否存在运行中实例（影响 warning / info 文案分支）
  const adjustHasRunningAdjustable = adjustAdjustableClaws.some((c) => c.status === "running");
  // 选中实例最小系统盘容量（本地校验阈值；批量取最小，单实例 = 当前容量）
  const minDiskCapacity = useMemo(() => {
    let min = Infinity;
    for (const c of adjustTargetClaws) {
      const info = getInstanceSpecInfo(c);
      if (info.systemDiskCapacity < min) min = info.systemDiskCapacity;
    }
    return min === Infinity ? 0 : min;
  }, [adjustTargetClaws]);
  // 输入框下方说明：最小可输入容量 = 最小盘容量 + 1
  const minInputCapacity = minDiskCapacity + 1;
  const diskHelperText = adjustConfigType === "capacity"
    ? `请输入 ${minInputCapacity}～2048 GiB 的整数容量。`
    : "";
  // 系统盘容量输入级本地实时校验：空不报错；非法值 / 超限 / 未超过最小盘 → 阻止后台校验
  const capacityInputError = useMemo(() => {
    if (adjustConfigType !== "capacity") return undefined;
    const raw = adjustTargetCapacity.trim();
    if (raw === "") return undefined;
    const num = Number(raw);
    // 非数字 / 小数 / 负数 / 0
    if (isNaN(num) || !Number.isInteger(num) || num <= 0) return "请输入有效的目标容量";
    if (num > 2048) return "目标容量最大不超过 2048GiB";
    if (num <= minDiskCapacity) return "目标容量需大于当前系统盘容量";
    return undefined;
  }, [adjustConfigType, adjustTargetCapacity, minDiskCapacity]);
  // 非法容量输入视为未选目标配置，不触发校验（统一展示「—」）
  const adjustEffectiveCapacity = capacityInputError !== undefined ? undefined : adjustTargetCapacityNum;
  // 目标配置是否有效（「校验」按钮可点条件）
  const canValidate = adjustConfigType === "spec"
    ? !!adjustTargetSpec
    : (capacityInputError === undefined && adjustTargetCapacityNum !== undefined && !Number.isNaN(adjustTargetCapacityNum));
  // 切换调整类型时：完整清空目标配置和校验状态
  const handleAdjustConfigTypeChange = (newType: "spec" | "capacity") => {
    if (newType === "spec") {
      setAdjustTargetCapacity("");
    } else {
      setAdjustTargetSpec(undefined);
      setSpecResetKey((k) => k + 1);
    }
    setAdjustConfigType(newType);
    resetValidationState();
  };
  // 实例统计：全部 / 可调整 / 不可调整（不含「无需调整」；结果来自缓存）
  const adjustStats = useMemo(() => {
    let adjustable = 0, unadjustable = 0;
    for (const c of adjustTargetClaws) {
      const r = adjustResults[c.id];
      if (!r) continue;
      if (r.label === "可调整") adjustable++;
      else if (r.label === "不可调整") unadjustable++;
    }
    return { total: adjustTargetClaws.length, adjustable, unadjustable };
  }, [adjustTargetClaws, adjustResults]);
  // ─── 随用户迁移到新组织：组织数据 & 配置对比的原/目标组织名 ───
  const migrateGroups = hasOneid ? MOCK_GROUPS : MOCK_MANUAL_GROUPS;
  /** 取某实例所属组织路径（优先 Agent 自身固定绑定的分组，回退按 creator 反查） */
  const getInstanceGroupName = (c: Claw): string =>
    (hasOneid ? getClawGroupItemOneid(c)?.path : getClawGroupItemManual(c)?.path)
    ?? "未分配组织配置";
  // 仅支持同一用户的实例批量迁移
  const migrateCreators = Array.from(new Set(selectedClaws.map((c) => c.creator)));
  const isSameMigrateUser = migrateCreators.length <= 1;
  const migrateBlocked = selectedCount > 0 && !isSameMigrateUser;
  const migrateUserId = selectedClaws[0]?.creator ?? "";
  // 目标组织候选：仅该用户所属的组织（一个或多个）；无组织则回退「未分配组织配置」
  const migrateUserGroupIds = isSameMigrateUser && migrateUserId
    ? getCreatorAllGroupIds(migrateUserId, hasOneid)
    : [];
  const migrateSelectableIds = new Set(migrateUserGroupIds);
  const hasMigrateUserGroups = migrateUserGroupIds.length > 0;
  // 目标组织下拉选项（普通单选，仅该用户所属组织，去重）：用户名下所有组织均可选，不校验是否为实例当前组织
  const migrateUserGroupOptions = Array.from(new Set(migrateUserGroupIds))
    .map((gid) => ({ id: gid, label: getGroupPath(gid, migrateGroups) }));
  // 默认选中第一个组织
  const firstSelectableMigrateGroup = migrateUserGroupOptions[0]?.id ?? "";
  // 配置对比：原组织（实例组织，多实例同源时取真实名）→ 目标组织
  const migrateCurrentOrgs = Array.from(new Set(selectedClaws.map(getInstanceGroupName)));
  const migrateFromGroupName = migrateCurrentOrgs.length === 1 ? migrateCurrentOrgs[0] : "实例组织";
  const migrateToGroupName = !hasMigrateUserGroups
    ? "未分配组织配置"
    : groupMigrateTarget ? getGroupPath(groupMigrateTarget, migrateGroups) : "目标组织";

  // ─── 移交给其他用户：双栏选择器数据 & 配置对比 ───
  const transferUserPool = hasOneid ? MOCK_USERS : MOCK_USERS_MANUAL;
  const transferGroups = migrateGroups;
  // 搜索关键字（按 用户ID / 组织名称）—— 与左侧组织树、右侧用户列表联动
  const transferKw = transferSearchKeyword.trim().toLowerCase();
  // 「未分配组织」常驻底部节点：放没有任何组织归属的用户（不参与上方搜索裁剪，始终展示）
  const transferUnassignedNode: GroupTreeNode = useMemo(() => ({
    id: TRANSFER_UNASSIGNED_GROUP_ID,
    name: "未分配组织",
    parentId: null,
    source: "manual",
    readonly: true,
    createdAt: "1970-01-01",
    children: [],
    path: "未分配组织",
    pathIds: [TRANSFER_UNASSIGNED_GROUP_ID],
    depth: 0,
  }), []);
  // 「全部组织」分区（树状结构，单选），按搜索条件裁剪分支；「未分配组织」始终常驻在最底部
  const transferAllGroupTree = useMemo(() => {
    const full = buildGroupTree(transferGroups);
    if (!transferKw) return [...full, transferUnassignedNode];
    const hitGroupIds =
      transferSearchField === "userId"
        ? new Set(
            transferUserPool
              .filter((u) => u.userId.toLowerCase().includes(transferKw))
              .flatMap((u) => u.groupIds),
          )
        : null;
    const keep = (node: GroupTreeNode): GroupTreeNode | null => {
      const children = node.children
        .map(keep)
        .filter((n): n is GroupTreeNode => n !== null);
      const selfMatch =
        transferSearchField === "groupName"
          ? node.name.toLowerCase().includes(transferKw)
          : hitGroupIds!.has(node.id);
      if (selfMatch || children.length > 0) return { ...node, children };
      return null;
    };
    const filtered = full.map(keep).filter((n): n is GroupTreeNode => n !== null);
    return [...filtered, transferUnassignedNode];
  }, [transferGroups, transferKw, transferSearchField, transferUserPool, transferUnassignedNode]);
  // 「用户ID」搜索时，右侧列表可不依赖左侧选中组织直接展示命中用户
  const transferSearchByUser = !!transferKw && transferSearchField === "userId";
  // 右侧成员列表：所有用户均可选（不校验是否为实例当前所属用户）；与搜索 / 左侧组织联动
  // 选中「未分配组织」时，展示 groupIds 为空（不隶属任何组织）的用户
  const transferGroupMembers = useMemo(() => {
    const isUnassignedPicked = transferPickerGroupId === TRANSFER_UNASSIGNED_GROUP_ID;
    if (transferSearchByUser) {
      let list = transferUserPool.filter((u) => u.userId.toLowerCase().includes(transferKw));
      if (isUnassignedPicked) list = list.filter((u) => u.groupIds.length === 0);
      else if (transferPickerGroupId) list = list.filter((u) => u.groupIds.includes(transferPickerGroupId));
      return list;
    }
    if (isUnassignedPicked) return transferUserPool.filter((u) => u.groupIds.length === 0);
    return transferPickerGroupId
      ? transferUserPool.filter((u) => u.groupIds.includes(transferPickerGroupId))
      : [];
  }, [transferSearchByUser, transferKw, transferPickerGroupId, transferUserPool]);
  // 接手用户对象
  const transferTargetUserObj = transferUserPool.find((u) => u.userId === groupTransferTarget) ?? null;
  // 接手用户所属组织（目标组织候选，排除部门）
  const transferTargetGroupOptions = transferTargetUserObj
    ? transferTargetUserObj.groupIds
        .filter((gid) => transferGroups.find((g) => g.id === gid)?.source !== "oneid-dept")
        .map((gid) => ({ id: gid, label: getGroupPath(gid, transferGroups) }))
    : [];
  const hasTransferTargetGroups = transferTargetGroupOptions.length > 0;
  // 配置对比的原 / 目标组织名
  const transferFromGroupName = migrateFromGroupName;
  const transferToGroupName = !hasTransferTargetGroups
    ? "未分配组织配置"
    : groupTransferTargetGroup ? getGroupPath(groupTransferTargetGroup, transferGroups) : "目标组织";
  const localSelectedCount = selectedClaws.filter((c) => c.agentType === "LocalAgent").length;
  const hasNonRunning = selectedClaws.some(c => !isUpgradable(c));
  const localUnsupportedTooltip = "本地 Agent 暂不支持批量处理、命令下发等云端能力";
  const batchDisabled = selectedCount === 0 || selectedCount > 20 || hasNonRunning || localSelectedCount > 0;
  const batchDeleteDisabled = selectedCount === 0 || localSelectedCount > 0;
  // 批量关机按钮逻辑：关机操作只对运行中的 Agent 生效（已关机/创建中等会被自动跳过）
  const runningSelectedClaws = selectedClaws.filter(c => c.agentType !== "LocalAgent" && c.status === "running");
  const batchShutdownDisabled = selectedCount === 0 || localSelectedCount > 0 || runningSelectedClaws.length === 0;
  const batchShutdownTooltip = selectedCount === 0
    ? "请先选择 Agent"
    : localSelectedCount > 0
    ? localUnsupportedTooltip
    : runningSelectedClaws.length === 0
    ? "所选 Agent 均不在运行中，无需关机"
    : "";
  // 批量配置模型按钮逻辑：仅运行中的 Agent 可配置，最多 20 台，统一目标模型
  // 范围口径与"关机"不同：本入口允许 openclaw（OpenClaw）与非 openclaw（Hermes/LightclawACE/LocalAgent）一起配置，
  // 但 openclaw 支持主备模式，非 openclaw 只支持单模型；二者能力不同，故禁止混合批量。
  const runningConfigurableClaws = selectedClaws.filter(c => c.agentType !== "LocalAgent" && c.status === "running");
  const openclawConfigurableClaws = runningConfigurableClaws.filter(c => c.agentType === "OpenClaw");
  const nonOpenclawConfigurableClaws = runningConfigurableClaws.filter(c => c.agentType !== "OpenClaw");
  const batchConfigModelDisabled =
    selectedCount === 0
    || localSelectedCount > 0
    || runningConfigurableClaws.length === 0
    || runningConfigurableClaws.length > 20
    || (openclawConfigurableClaws.length > 0 && nonOpenclawConfigurableClaws.length > 0);
  const batchConfigModelTooltip = selectedCount === 0
    ? "请先选择 Agent"
    : localSelectedCount > 0
    ? localUnsupportedTooltip
    : runningConfigurableClaws.length === 0
    ? "仅运行中的 Agent 支持批量配置模型"
    : runningConfigurableClaws.length > 20
    ? "批量配置模型数量不可大于 20"
    : (openclawConfigurableClaws.length > 0 && nonOpenclawConfigurableClaws.length > 0)
    ? "所选 Agent 包含 openclaw 与非 openclaw 两种类型，不能一起批量配置模型"
    : "";
  // 批量开机按钮逻辑：开机操作只对已关机的 Agent 生效
  const shutdownSelectedClaws = selectedClaws.filter(c => c.agentType !== "LocalAgent" && c.status === "shutdown");
  const batchPowerOnDisabled = selectedCount === 0 || localSelectedCount > 0 || shutdownSelectedClaws.length === 0;
  const batchPowerOnTooltip = selectedCount === 0
    ? "请先选择 Agent"
    : localSelectedCount > 0
    ? localUnsupportedTooltip
    : shutdownSelectedClaws.length === 0
    ? "所选 Agent 均不在关机状态，无需开机"
    : "";
  const batchTooltip = selectedCount === 0
    ? '请先选择实例'
    : localSelectedCount > 0
    ? localUnsupportedTooltip
    : selectedCount > 20
    ? '批量更新数量不可大于 20'
    : hasNonRunning
    ? '仅运行中的实例支持更新'
    : '';

  const handleRefresh = () => {
    setRefreshing(true);
    setTimeout(() => {
      setRefreshing(false);
      toast.success("列表已刷新");
    }, 1000);
  };

  // ── 批量配置模型 ──
  // 当前弹窗目标 Agent 中是否全为非 openclaw 类型（决定是否走「单模型」模式）
  const batchSingleModelMode = nonOpenclawConfigurableClaws.length > 0 && openclawConfigurableClaws.length === 0;

  const openBatchConfigModelDialog = () => {
    setBatchAppliedModels([]);
    setBatchModelIdCounter(1);
    // 厂商/版本有意不重置：保留用户上一次批量配置常用的选择，减少高频操作时的重复挑选
    setShowBatchConfigModelDialog(true);
  };

  // 切换厂商时同步重置为该厂商首个版本
  const handleBatchModelProviderChange = (providerValue: string) => {
    setBatchModelProvider(providerValue);
    const provider = MODEL_PROVIDERS.find(p => p.value === providerValue);
    if (provider && provider.versions.length > 0) {
      setBatchModelVersion(provider.versions[0].value);
    }
  };

  // 添加一个模型到目标模型列表
  // - openclaw：首个自动为主模型，其余为备选
  // - 非 openclaw：仅保留 1 个模型；再次添加时直接覆盖上一个
  const handleAddBatchModel = () => {
    const provider = MODEL_PROVIDERS.find(p => p.value === batchModelProvider);
    const version = provider?.versions.find(v => v.value === batchModelVersion);
    if (!provider || !version) return;
    // 不允许重复添加同一厂商+版本
    if (batchAppliedModels.some(m => m.providerLabel === provider.label && m.versionLabel === version.label)) {
      toast.error("该模型已添加");
      return;
    }
    setBatchAppliedModels(prev => {
      // 非 openclaw 模式：仅保留 1 个模型；新添加的覆盖旧的（替换效果）
      if (batchSingleModelMode) {
        const entry: BatchConfigModel = {
          id: batchModelIdCounter,
          providerLabel: provider.label,
          versionLabel: version.label,
          primary: false,
        };
        toast.success(prev.length > 0 ? "已替换为新的目标模型" : "模型已添加成功");
        return [entry];
      }
      // openclaw 模式：首个自动为主模型，其余为备选
      const hasPrimary = prev.some(m => m.primary);
      const entry: BatchConfigModel = {
        id: batchModelIdCounter,
        providerLabel: provider.label,
        versionLabel: version.label,
        primary: !hasPrimary,
      };
      toast.success(hasPrimary ? "备选模型已添加" : "已设为主模型");
      return [...prev, entry];
    });
    setBatchModelIdCounter(c => c + 1);
  };

  // 将某个备选模型设为主模型（原主模型降为备选），逻辑与单实例一致
  const handleSetBatchPrimary = (id: number) => {
    setBatchAppliedModels(prev => prev.map(m => ({ ...m, primary: m.id === id })));
  };

  // 删除目标模型；若删除的是主模型，则自动将列表中第一个升为主模型
  const handleRemoveBatchModel = (id: number) => {
    setBatchAppliedModels(prev => {
      const removed = prev.find(m => m.id === id);
      const next = prev.filter(m => m.id !== id);
      if (removed?.primary && next.length > 0 && !next.some(m => m.primary)) {
        next[0] = { ...next[0], primary: true };
      }
      return next;
    });
  };

  const confirmBatchConfigModel = () => {
    if (batchAppliedModels.length === 0) {
      toast.error("请至少添加一个目标模型");
      return;
    }
    const targets = runningConfigurableClaws;
    if (targets.length === 0) {
      toast.error("仅运行中的 Agent 支持批量配置模型");
      return;
    }
    // 安全闸：非 openclaw 模式只允许 1 个模型
    if (batchSingleModelMode && batchAppliedModels.length > 1) {
      toast.error("非 openclaw 类型仅支持配置单模型");
      return;
    }
    // 安全闸：openclaw 与非 openclaw 不应混合（菜单已禁用，但若绕过进入弹窗则在此再校验一次）
    if (openclawConfigurableClaws.length > 0 && nonOpenclawConfigurableClaws.length > 0) {
      toast.error("openclaw 与非 openclaw 类型不能一起批量配置模型");
      return;
    }
    // 仅对运行中的所选 Agent 统一应用相同的目标模型配置
    setShowBatchConfigModelDialog(false);
    setSelectedIds(new Set());
    toast.success(`已为 ${targets.length} 个 Agent 配置模型`);
  };

  const handleOpenTerminal = (claw: Claw) => {
    window.open(`/terminal/${claw.id}`, "_blank");
  };

  const handleRestart = (claw: Claw) => {
    setRestartConfirm({ id: claw.id, name: claw.name });
  };

  const handleReinstallClick = (claw: Claw) => {
    setReinstallTarget(claw.id);
    setReinstallInput("");
  };

  const confirmReinstall = () => {
    if (!reinstallTarget) return;
    const claw = claws.find(c => c.id === reinstallTarget);
    setClaws(claws.map(c => c.id === reinstallTarget ? { ...c, status: "running" as ClawStatus } : c));
    setReinstallTarget(null);
    setReinstallInput("");
    toast.success(`正在重新安装 ${claw?.name}...`);
  };

  const confirmRestart = () => {
    if (!restartConfirm) return;
    if (restartFullServer) {
      toast.success(`「${restartConfirm.name}」正在重启整台服务器，预计需要约 2 分钟...`);
    } else {
      toast.success(`「${restartConfirm.name}」正在重启 Agent 服务...`);
    }
    setRestartConfirm(null);
    setRestartFullServer(false);
  };

  const confirmShutdown = () => {
    if (!shutdownTarget) return;
    const claw = claws.find(c => c.id === shutdownTarget);
    setClaws(claws.map(c => c.id === shutdownTarget ? { ...c, status: "shutdown" as ClawStatus } : c));
    setShutdownTarget(null);
    toast.success(`已关机 ${claw?.name}`);
  };

  const confirmPowerOn = () => {
    if (!shutdownTarget) return;
    const claw = claws.find(c => c.id === shutdownTarget);
    setClaws(claws.map(c => c.id === shutdownTarget ? { ...c, status: "running" as ClawStatus } : c));
    setShutdownTarget(null);
    toast.success(`已开机 ${claw?.name}`);
  };

  const handleDeleteClick = (claw: Claw) => {
    setDeleteTarget(claw.id);
    setDeleteInput("");
  };

  const handleManageLocalSkills = (claw: Claw) => {
    setLocation(`/admin/agent-tool-library?tab=enterprise&targetAgent=${encodeURIComponent(claw.instanceId)}`);
  };

  const confirmDelete = () => {
    if (!deleteTarget) return;
    const claw = claws.find(c => c.id === deleteTarget);
    setClaws(claws.filter(c => c.id !== deleteTarget));
    setDeleteTarget(null);
    setDeleteInput("");
    toast.success(`${claw?.agentType === "LocalAgent" ? "已移除" : "已删除"} ${claw?.name}`);
  };

  // ── Agent 详情抽屉数据（可编辑） ─────────────────────────────────────────
  // 每个 claw 一份独立详情，编辑后保留在内存（管控端 demo 不持久化）。

  /** 已接入通道：除了基本展示字段，还保留一份凭证录入值 */
  interface ConnectedChannel {
    /** 通道展示名，与 CHANNEL_OPTIONS.label 对应，作为唯一标识 */
    name: string;
    /** 通道 value（CHANNEL_OPTIONS.value），便于反查 fields 定义 */
    value: string;
    /** 凭证字段值：按 ChannelField.key 存储 */
    fieldValues: Record<string, string>;
    bots: string[];
  }

  /**
   * 单条已应用模型：对应"模型配置"页中的一条记录。
   * - modelConfigId：关联管控端模型表 id；被删除/隐藏时按 Q3(c) 完全无感处理，仍用冗余字段展示
   * - providerLabel / versionLabel：展示态冗余，避免管控端模型变更后失去展示文案
   * - isCustom：是否自定义模型（展示"自定义模型"一级文案 + 小字为模型名）
   * - primary：是否主模型；整个列表至多一条 primary=true
   */
  interface AppliedModelItem {
    id: number;
    modelConfigId: string;
    providerLabel: string;
    versionLabel: string;
    isCustom: boolean;
    primary: boolean;
    addedAt: number;
  }

  interface ClawDetail {
    /** 已应用模型列表：可能为空（无模型）、只主、主+备 */
    appliedModels: AppliedModelItem[];
    /** 已接入通道列表 */
    connectedChannels: ConnectedChannel[];
    installedSkills: string[];
  }

  interface LocalSkillStatus {
    name: string;
    version: string;
    source: string;
    lastSync: string;
  }

  interface LocalStandardStatus {
    name: string;
    type: "CLAUDE.md" | "rules";
    lastSync: string;
  }

  interface LocalAgentDetail {
    product: LocalAgentProduct;
    runtimeStatus: "已接入" | "未接入";
    hostName: string;
    os: string;
    pluginVersion: string;
    reportTime: string;
    skills: LocalSkillStatus[];
    standards: LocalStandardStatus[];
  }

  /** 基于 clawId 稳定分布，模拟三种场景：hash%3 → 0=空 / 1=只主 / 2=主+备 */
  const hashClawId = (s: string): number => {
    let h = 0;
    for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0;
    return Math.abs(h);
  };

  const buildDefaultClawDetail = (clawId: string): ClawDetail => {
    const scenario = hashClawId(clawId) % 3;
    const baseTs = Date.now();
    const appliedModels: AppliedModelItem[] =
      scenario === 0
        ? []
        : scenario === 1
        ? [
            {
              id: 1,
              modelConfigId: "1",
              providerLabel: "腾讯云 DeepSeek",
              versionLabel: "DeepSeek V3 0324",
              isCustom: false,
              primary: true,
              addedAt: baseTs,
            },
          ]
        : [
            {
              id: 1,
              modelConfigId: "1",
              providerLabel: "腾讯云 DeepSeek",
              versionLabel: "DeepSeek V3 0324",
              isCustom: false,
              primary: true,
              addedAt: baseTs,
            },
            {
              id: 2,
              modelConfigId: "2",
              providerLabel: "腾讯混元",
              versionLabel: "Hunyuan Turbo",
              isCustom: false,
              primary: false,
              addedAt: baseTs - 60_000,
            },
          ];
    return {
      appliedModels,
      connectedChannels: [
        { name: "飞书", value: "feishu", fieldValues: { appId: "cli_a1b2c3", appSecret: "fsk_xxxxxx" }, bots: [] },
      ],
      installedSkills: [
        "feishu-doc", "feishu-drive", "feishu-perm", "feishu-wiki",
        "feishu-calendar", "feishu-message", "feishu-task",
      ],
    };
  };

  const buildLocalAgentDetail = (claw: Claw): LocalAgentDetail => {
    const product = getLocalAgentProduct(claw);
    const isConnected = isLocalConnected(claw);
    const standards = Array.isArray(claw.standards)
      ? claw.standards.map((standard, index) => {
          if (typeof standard === "string") {
            return {
              name: standard,
              type: "rules" as const,
              lastSync: index === 0 ? "2026-06-16 18:49" : "2026-06-16 18:50",
            };
          }
          return {
            name: standard.name || standard.title || "未命名规范",
            type: standard.kind === "entry" || standard.type === "CLAUDE.md" ? "CLAUDE.md" as const : "rules" as const,
            lastSync: standard.lastSync || standard.updatedAt || "2026-06-16 18:49",
          };
        })
      : [];
    return {
      product,
      runtimeStatus: isConnected ? "已接入" : "未接入",
      hostName: claw.instanceId === "local-workbuddy-02" ? "mbp-dev-offline" : product === "Claude" ? "pm-claude-studio" : product === "CodeBuddy" ? "dev-codebuddy-mbp" : product === "Codex" ? "codex-devbook" : "olivia-mbp",
      os: claw.instanceId === "local-workbuddy-02" ? "macOS 14.5" : product === "CodeBuddy" ? "Windows 11 Pro" : product === "Codex" ? "macOS 15.4" : "macOS 15.1",
      pluginVersion: claw.version,
      reportTime: isConnected ? "2026-06-16 18:45:12" : "2026-06-15 22:18:40",
      skills: [
        { name: "doc-summarizer", version: "1.3.0", source: "企业技能库", lastSync: "2026-06-16 18:45" },
        { name: "meeting-summary", version: "1.3.0", source: "公共技能库", lastSync: "2026-06-16 18:46" },
        { name: "local-shell-helper", version: "0.9.2", source: "本地客户端", lastSync: "2026-06-16 18:48" },
      ],
      standards,
    };
  };

  const [clawDetailMap, setClawDetailMap] = useState<Record<string, ClawDetail>>({});

  /** 读取某个 claw 的详情（不存在则按 clawId hash 生成默认快照，不写入 map 以避免 render 期间 setState） */
  const getClawDetail = (clawId: string): ClawDetail => {
    return clawDetailMap[clawId] ?? buildDefaultClawDetail(clawId);
  };

  /** 用 updater 形式更新某个 claw 的详情，缺失时基于默认值初始化 */
  const updateClawDetail = (
    clawId: string,
    updater: (prev: ClawDetail) => ClawDetail,
  ) => {
    setClawDetailMap(prev => {
      const current = prev[clawId] ?? buildDefaultClawDetail(clawId);
      return { ...prev, [clawId]: updater(current) };
    });
  };

  /** 生成下一个模型 entry id：取当前列表最大 id + 1 */
  const nextModelEntryId = (list: AppliedModelItem[]): number => {
    return list.reduce((max, m) => (m.id > max ? m.id : max), 0) + 1;
  };

  // ── 订阅"模型配置"页的数据（仅 visible=true 的对外可见） ───────────────────
  const adminModels = useAdminModels();
  const visibleAdminModels = useMemo(() => adminModels.filter(m => m.visible), [adminModels]);

  /**
   * 把可见模型按"厂商"组织：
   *   - 普通厂商：按 provider 组织，组 key = provider，组 label = 同 provider 第一条 name
   *   - 自定义模型（provider === __custom__）：聚合到单一"自定义模型"组下，每条作为一个版本
   * 厂商一级显示顺序：先按出现顺序，自定义模型组始终放最后。
   */
  interface ProviderGroup {
    key: string;           // provider 值；自定义模型组固定为 __custom__
    label: string;         // 一级 Select 显示文本
    models: ModelRow[];    // 该厂商下所有可见模型记录
    isCustom: boolean;
  }

  const providerGroups = useMemo<ProviderGroup[]>(() => {
    const orderedKeys: string[] = [];
    const buckets = new Map<string, ModelRow[]>();
    for (const m of visibleAdminModels) {
      const key = m.provider;
      if (!buckets.has(key)) {
        buckets.set(key, []);
        orderedKeys.push(key);
      }
      buckets.get(key)!.push(m);
    }
    const groups: ProviderGroup[] = [];
    let customGroup: ProviderGroup | null = null;
    for (const key of orderedKeys) {
      const models = buckets.get(key)!;
      if (key === CUSTOM_PROVIDER_VALUE) {
        customGroup = {
          key,
          label: "自定义模型",
          models,
          isCustom: true,
        };
      } else {
        groups.push({
          key,
          // 同 provider 的多条记录 name 理论上一致，取第一条
          label: models[0].name,
          models,
          isCustom: false,
        });
      }
    }
    if (customGroup) groups.push(customGroup);
    return groups;
  }, [visibleAdminModels]);

  // ── 模型编辑态 ───────────────────────────────────────────────────────────
  /**
   * 模型编辑上下文：
   * - idle：未进入编辑态
   * - add：点击右上角"添加备选/设为主模型"按钮 → 底部 inline 新增卡
   * - replace：点击某条模型行的 ✏️ → 底部 inline 卡用于替换该条
   */
  type ModelActionContext =
    | { kind: "idle" }
    | { kind: "add" }
    | { kind: "replace"; modelEntryId: number };
  const [modelAction, setModelAction] = useState<ModelActionContext>({ kind: "idle" });
  const modelEditing = modelAction.kind !== "idle";

  /** 一级草稿：厂商 key（即 provider 值） */
  const [modelDraftProvider, setModelDraftProvider] = useState<string>("");
  /** 二级草稿：具体模型记录 id */
  const [modelDraftModelId, setModelDraftModelId] = useState<string>("");

  /** 模型操作二次确认弹窗（复用用户端三种类型） */
  const [modelConfirmDialog, setModelConfirmDialog] = useState<{
    open: boolean;
    type: "set-primary" | "delete" | "delete-backup";
    modelEntryId: number | null;
  }>({ open: false, type: "set-primary", modelEntryId: null });

  /** 把一个管控端 ModelRow + 其所在组转换成 AppliedModelItem 的展示字段 */
  const toAppliedModelFields = (
    group: ProviderGroup,
    model: ModelRow,
  ): Pick<AppliedModelItem, "modelConfigId" | "providerLabel" | "versionLabel" | "isCustom"> => ({
    modelConfigId: model.id,
    providerLabel: group.label,
    // 自定义模型：一级展示"自定义模型"，二级用模型 name；普通模型二级用 version
    versionLabel: group.isCustom ? model.name : model.version,
    isCustom: group.isCustom,
  });

  /** 进入"添加"模式：默认草稿回填首组首项 */
  const startAddModel = () => {
    if (providerGroups.length === 0) {
      setModelDraftProvider("");
      setModelDraftModelId("");
      setModelAction({ kind: "add" });
      return;
    }
    const g0 = providerGroups[0];
    setModelDraftProvider(g0.key);
    setModelDraftModelId(g0.models[0]?.id ?? "");
    setModelAction({ kind: "add" });
  };

  /** 进入"替换"模式：按被替换条目当前的 modelConfigId 回填；找不到则回退首组首项 */
  const startReplaceModel = (entry: AppliedModelItem) => {
    if (providerGroups.length === 0) {
      setModelDraftProvider("");
      setModelDraftModelId("");
      setModelAction({ kind: "replace", modelEntryId: entry.id });
      return;
    }
    let targetGroup: ProviderGroup | undefined;
    let targetModel: ModelRow | undefined;
    for (const g of providerGroups) {
      const m = g.models.find(x => x.id === entry.modelConfigId);
      if (m) { targetGroup = g; targetModel = m; break; }
    }
    if (!targetGroup || !targetModel) {
      targetGroup = providerGroups[0];
      targetModel = targetGroup.models[0];
    }
    setModelDraftProvider(targetGroup.key);
    setModelDraftModelId(targetModel.id);
    setModelAction({ kind: "replace", modelEntryId: entry.id });
  };

  const cancelEditModel = () => setModelAction({ kind: "idle" });

  const saveEditModel = () => {
    if (!selectedClaw) return;
    const group = providerGroups.find(g => g.key === modelDraftProvider);
    const model = group?.models.find(m => m.id === modelDraftModelId);
    if (!group || !model) {
      toast.error("请选择有效的模型厂商和版本");
      return;
    }
    const fields = toAppliedModelFields(group, model);
    const action = modelAction;
    const current = getClawDetail(selectedClaw.id);
    const list = current.appliedModels;
    // 重复校验：同一条模型配置不可重复添加（替换时允许命中自己）
    const dupe = list.find(m => m.modelConfigId === fields.modelConfigId
      && !(action.kind === "replace" && m.id === action.modelEntryId));
    if (dupe) {
      toast.error("该模型已在列表中，请勿重复添加");
      return;
    }
    const hadPrimaryBefore = list.some(m => m.primary);
    updateClawDetail(selectedClaw.id, prev => {
      if (action.kind === "add") {
        const hasPrimary = prev.appliedModels.some(m => m.primary);
        const newEntry: AppliedModelItem = {
          id: nextModelEntryId(prev.appliedModels),
          ...fields,
          // 无主模型时新加的直接成为主模型；否则作为备选
          primary: !hasPrimary,
          addedAt: Date.now(),
        };
        return { ...prev, appliedModels: [...prev.appliedModels, newEntry] };
      }
      if (action.kind === "replace") {
        return {
          ...prev,
          appliedModels: prev.appliedModels.map(m => m.id === action.modelEntryId
            ? { ...m, ...fields }
            : m),
        };
      }
      return prev;
    });
    const _isOpenClawSave = selectedClaw?.agentType === 'OpenClaw';
    if (action.kind === "add") {
      toast.success(hadPrimaryBefore ? "备选模型已添加" : (_isOpenClawSave ? "已设为主模型" : "模型已添加成功"));
    } else {
      toast.success("模型已更新");
    }
    setModelAction({ kind: "idle" });
  };

  /** 切换一级厂商时，把二级草稿重置为该厂商的第一项 */
  const handleDraftProviderChange = (value: string) => {
    setModelDraftProvider(value);
    const group = providerGroups.find(g => g.key === value);
    if (group && group.models.length > 0) {
      setModelDraftModelId(group.models[0].id);
    } else {
      setModelDraftModelId("");
    }
  };

  /** 确认二次确认 Dialog 的操作 */
  const runModelConfirm = () => {
    if (!selectedClaw) return;
    const { type, modelEntryId } = modelConfirmDialog;
    if (modelEntryId === null) {
      setModelConfirmDialog(prev => ({ ...prev, open: false }));
      return;
    }
    updateClawDetail(selectedClaw.id, prev => {
      const list = prev.appliedModels;
      if (type === "set-primary") {
        return {
          ...prev,
          appliedModels: list.map(m => ({ ...m, primary: m.id === modelEntryId })),
        };
      }
      if (type === "delete-backup") {
        return { ...prev, appliedModels: list.filter(m => m.id !== modelEntryId) };
      }
      // type === "delete" (主模型)：删除后首条备选自动升主
      const next = list.filter(m => m.id !== modelEntryId);
      const wasPrimary = list.find(m => m.id === modelEntryId)?.primary ?? false;
      if (wasPrimary && next.length > 0 && !next.some(m => m.primary)) {
        next[0] = { ...next[0], primary: true };
      }
      return { ...prev, appliedModels: next };
    });
    setModelConfirmDialog(prev => ({ ...prev, open: false }));
    // 如果当前正在替换的正是被删除的这条，取消编辑态
    if (modelAction.kind === "replace" && modelAction.modelEntryId === modelEntryId) {
      setModelAction({ kind: "idle" });
    }
    const _isOpenClaw = selectedClaw?.agentType === 'OpenClaw';
    if (type === "set-primary") toast.success("已设为主模型");
    else if (type === "delete-backup") toast.success("备选模型已删除");
    else toast.success(_isOpenClaw ? "主模型已删除，已自动升级备选模型" : "模型删除成功");
  };

  // ── 通道编辑态 ───────────────────────────────────────────────────────────
  /** 是否处于"新增通道"模式（展示底部 inline 选择条） */
  const [channelAdding, setChannelAdding] = useState(false);
  const [channelDraft, setChannelDraft] = useState<string>("");
  /** 新增通道时正在录入的凭证字段值 */
  const [channelDraftFields, setChannelDraftFields] = useState<Record<string, string>>({});
  /** 待移除的通道 name（触发 AlertDialog 二次确认） */
  const [channelRemoveTarget, setChannelRemoveTarget] = useState<string | null>(null);
  /** 当前展开查看/编辑凭证的通道 name（null 表示全部收起） */
  const [expandedChannel, setExpandedChannel] = useState<string | null>(null);
  /** 当前展开通道的编辑草稿值；null 表示未进入编辑态（只读查看） */
  const [channelEditDraft, setChannelEditDraft] = useState<Record<string, string> | null>(null);
  /** 密码字段可见性：用 "channelName:fieldKey" 作为 key */
  const [visibleSecrets, setVisibleSecrets] = useState<Set<string>>(new Set());

  const toggleSecretVisibility = (channelName: string, fieldKey: string) => {
    const key = `${channelName}:${fieldKey}`;
    setVisibleSecrets(prev => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const isSecretVisible = (channelName: string, fieldKey: string): boolean => {
    return visibleSecrets.has(`${channelName}:${fieldKey}`);
  };

  /** 加密显示：保留前 3 字符，后面用 •••••• 替代 */
  const maskSecret = (val: string): string => {
    if (!val) return "—";
    if (val.length <= 3) return val;
    return val.slice(0, 3) + "••••••";
  };

  // ── 订阅"通道配置"页的可见性数据 ─────────────────────────────────────────
  const [builtinChannelVisibility, setBuiltinChannelVisibility] = useState<Record<string, boolean>>(
    () => loadBuiltinChannelVisibility(),
  );
  useEffect(() => {
    return onBuiltinChannelVisibilityChange(() => {
      setBuiltinChannelVisibility(loadBuiltinChannelVisibility());
    });
  }, []);

  const [visibleCustomChannels, setVisibleCustomChannels] = useState<AdminCustomChannel[]>(
    () => loadVisibleCustomChannels(),
  );
  useEffect(() => {
    return onCustomChannelsChange(() => {
      setVisibleCustomChannels(loadVisibleCustomChannels());
    });
  }, []);

  /**
   * 用户/Agent 可见的通道列表（内置 + 自定义）。
   *   - 内置通道：用 builtinId 或 value 查 builtinChannelVisibility，缺省按 true 处理
   *   - 自定义通道：来自管控端"通道配置"页，且 visible=true（loadVisibleCustomChannels 已过滤）
   */
  const availableChannelOptions = useMemo<ChannelConfig[]>(() => {
    const builtins = CHANNEL_OPTIONS.filter((ch) => {
      const key = ch.builtinId ?? ch.value;
      return builtinChannelVisibility[key] !== false;
    });
    const customs: ChannelConfig[] = visibleCustomChannels.map((cc) => ({
      value: `admin_custom_${cc.id}`,
      label: cc.name,
      descText: `企业自定义通道（Channel ID: ${cc.channelId}）`,
      detailUrl: "#",
      adminCustomMode: true as const,
      adminCustomId: cc.id,
      fields: cc.credentialFields.map((f) => ({
        key: f.key || f.id,
        label: f.label,
        secret: true,
      })),
    }));
    return [...builtins, ...customs];
  }, [builtinChannelVisibility, visibleCustomChannels]);

  /**
   * 通道反查表：用 channel.value 查 ChannelConfig（含 fields 定义）
   * - 内置通道：6 个全集（包括当前不可见的，避免已添加通道行失去字段定义）
   * - 自定义通道：所有"可见"的（loadVisibleCustomChannels 已过滤；不可见的暂不反查）
   * 注：现实场景中，自定义通道一旦被删除，已添加到 Agent 的同名通道将无 fields 元数据。
   */
  const channelLookup = useMemo<Map<string, ChannelConfig>>(() => {
    const map = new Map<string, ChannelConfig>();
    for (const ch of CHANNEL_OPTIONS) map.set(ch.value, ch);
    for (const ch of availableChannelOptions) {
      if (ch.adminCustomMode) map.set(ch.value, ch);
    }
    return map;
  }, [availableChannelOptions]);

  const startAddChannel = (detail: ClawDetail) => {
    // 默认选中第一个尚未被添加的通道；全部已添加时留空
    const existing = new Set(detail.connectedChannels.map(c => c.name));
    const firstAvailable = availableChannelOptions.find(c => !existing.has(c.label));
    setChannelDraft(firstAvailable?.value ?? "");
    setChannelDraftFields({});
    setChannelAdding(true);
    setExpandedChannel(null); // 收起已展开的通道，避免视觉混乱
    setChannelEditDraft(null);
  };

  const cancelAddChannel = () => {
    setChannelAdding(false);
    setChannelDraft("");
    setChannelDraftFields({});
  };

  /** 切换新增草稿中选择的通道 */
  const handleChannelDraftChange = (value: string) => {
    setChannelDraft(value);
    setChannelDraftFields({});
  };

  const confirmAddChannel = () => {
    if (!selectedClaw) return;
    const ch = availableChannelOptions.find(c => c.value === channelDraft);
    if (!ch) {
      toast.error("请选择要添加的通道");
      return;
    }
    const detail = getClawDetail(selectedClaw.id);
    if (detail.connectedChannels.some(c => c.name === ch.label)) {
      toast.error(`「${ch.label}」已添加，请勿重复`);
      return;
    }
    // 校验 fields（如有）必须填齐
    const requiredFields = ch.fields ?? [];
    const missing = requiredFields.find(f => !(channelDraftFields[f.key] ?? "").trim());
    if (missing) {
      toast.error(`请填写「${missing.label}」`);
      return;
    }
    updateClawDetail(selectedClaw.id, prev => ({
      ...prev,
      connectedChannels: [
        ...prev.connectedChannels,
        {
          name: ch.label,
          value: ch.value,
          fieldValues: { ...channelDraftFields },
          bots: [],
        },
      ],
    }));
    setChannelAdding(false);
    setChannelDraft("");
    setChannelDraftFields({});
    toast.success(`已添加通道「${ch.label}」`);
  };

  const confirmRemoveChannel = () => {
    if (!selectedClaw || !channelRemoveTarget) return;
    const targetName = channelRemoveTarget;
    updateClawDetail(selectedClaw.id, prev => ({
      ...prev,
      connectedChannels: prev.connectedChannels.filter(c => c.name !== targetName),
    }));
    // 如果被删除的通道正展开，顺手收起
    if (expandedChannel === targetName) {
      setExpandedChannel(null);
      setChannelEditDraft(null);
    }
    setChannelRemoveTarget(null);
    toast.success(`已移除通道「${targetName}」`);
  };

  /** 展开/收起某个通道的凭证展示区（同一时刻只展开一个） */
  const toggleExpandChannel = (channel: ConnectedChannel) => {
    if (expandedChannel === channel.name) {
      setExpandedChannel(null);
      setChannelEditDraft(null);
    } else {
      setExpandedChannel(channel.name);
      setChannelEditDraft(null); // 默认进入只读查看态
    }
  };

  /** 进入某个已接入通道的编辑态（只读 → 编辑） */
  const startEditChannel = (channel: ConnectedChannel) => {
    setExpandedChannel(channel.name);
    setChannelEditDraft({ ...channel.fieldValues });
  };

  const cancelEditChannel = () => {
    setChannelEditDraft(null);
  };

  const saveEditChannel = (channel: ConnectedChannel) => {
    if (!selectedClaw || !channelEditDraft) return;
    const chConfig = channelLookup.get(channel.value);
    const requiredFields = chConfig?.fields ?? [];
    const missing = requiredFields.find(f => !(channelEditDraft[f.key] ?? "").trim());
    if (missing) {
      toast.error(`请填写「${missing.label}」`);
      return;
    }
    updateClawDetail(selectedClaw.id, prev => ({
      ...prev,
      connectedChannels: prev.connectedChannels.map(c =>
        c.name === channel.name ? { ...c, fieldValues: { ...channelEditDraft } } : c,
      ),
    }));
    setChannelEditDraft(null);
    toast.success(`「${channel.name}」凭证已更新`);
  };

  const handleOpenDrawer = (claw: Claw) => {
    setSelectedClaw(claw);
    setShowDetailDrawer(true);
    // 切换实例时重置所有编辑态，避免上一个实例残留
    setModelAction({ kind: "idle" });
    setModelConfirmDialog({ open: false, type: "set-primary", modelEntryId: null });
    setChannelAdding(false);
    setChannelDraft("");
    setChannelDraftFields({});
    setExpandedChannel(null);
    setChannelEditDraft(null);
    setVisibleSecrets(new Set());
  };

  const handleRefreshDrawer = () => {
    if (!selectedClaw) return;
    setDrawerLoading(true);
    setTimeout(() => {
      setDrawerLoading(false);
      toast.success("信息已刷新");
    }, 1500);
  };

  const renderLocalAgentDetail = (claw: Claw) => {
    const detail = buildLocalAgentDetail(claw);
    const connectionMeta = getLocalConnectionMeta(claw);
    return (
      <div className="p-4 space-y-4">
        <div className="min-w-0 space-y-2">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <PanelTitle as="div" className="truncate leading-tight">{claw.name}</PanelTitle>
              <div className="mt-1 flex items-center gap-2 flex-wrap">
                <CodeText>{claw.instanceId}</CodeText>
                <StatusTag mode="soft" variant="blue">{detail.product}</StatusTag>
                <StatusTag mode="soft" variant={connectionMeta.variant}>{detail.runtimeStatus}</StatusTag>
              </div>
            </div>
            <div className="w-10 h-10 rounded-[6px] bg-[var(--accent)] text-[var(--text-brand)] flex items-center justify-center shrink-0">
              <Laptop className="w-5 h-5" />
            </div>
          </div>
        </div>

        <div>
          <MetaText as="div" className="mb-2">本地客户端信息</MetaText>
          <div className="overflow-hidden rounded-[4px] border border-[var(--border)] bg-background">
            {[
              ["插件载体", detail.product],
              ["运行状态", detail.runtimeStatus],
              ["主机名", detail.hostName],
              ["系统版本", detail.os],
              ["插件版本", detail.pluginVersion],
              ["信息更新时间", detail.reportTime],
            ].map(([label, value]) => (
              <div key={label} className="grid grid-cols-[96px_minmax(0,1fr)] gap-3 border-b border-[var(--border)] px-3 py-2 last:border-b-0">
                <MetaText tone="weak">{label}</MetaText>
                <MiniBodyText className="truncate">{value}</MiniBodyText>
              </div>
            ))}
          </div>
        </div>

        <div>
          <div className="flex items-center justify-between mb-2">
            <MetaText as="div" className="flex items-center gap-1.5">
              <PackageCheck className="w-3.5 h-3.5" />
              已安装 Skill（{detail.skills.length}）
            </MetaText>
          </div>
          <div className="overflow-hidden rounded-[4px] border border-[var(--border)] bg-background">
            <Table density="compact" autoFixedColumns={false}>
              <TableHeader>
                <TableRow>
                  <TableHead>Skill</TableHead>
                  <TableHead>来源</TableHead>
                  <TableHead>安装时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {detail.skills.map((skill) => (
                  <TableRow key={skill.name}>
                    <TableCell>
                      <MiniBodyText as="div">{skill.name}</MiniBodyText>
                      <MetaText tone="weak">v{skill.version}</MetaText>
                    </TableCell>
                    <TableCell>
                      <MetaText tone="secondary">{skill.source}</MetaText>
                    </TableCell>
                    <TableCell>
                      <MetaText tone="secondary">{skill.lastSync}</MetaText>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>

        <div>
          <div className="flex items-center justify-between mb-2">
            <MetaText as="div" className="flex items-center gap-1.5">
              <ListChecks className="w-3.5 h-3.5" />
              已安装企业规范（{detail.standards.length}）
            </MetaText>
          </div>
          {detail.standards.length === 0 ? (
            <MetaText as="div" tone="weak" className="px-4 py-6 bg-background rounded-[4px] border border-dashed border-[var(--border)] text-center">
              暂无已安装企业规范
            </MetaText>
          ) : (
            <div className="overflow-hidden rounded-[4px] border border-[var(--border)] bg-background">
              <Table density="compact" autoFixedColumns={false}>
                <TableHeader>
                  <TableRow>
                    <TableHead>企业规范</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>下发时间</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {detail.standards.map((standard) => (
                    <TableRow key={`${standard.type}-${standard.name}`}>
                      <TableCell>
                        <MiniBodyText as="div">{standard.name}</MiniBodyText>
                        <MetaText tone="weak">企业规范库</MetaText>
                      </TableCell>
                      <TableCell>
                        <MetaText tone="secondary">{standard.type}</MetaText>
                      </TableCell>
                      <TableCell>
                        <MetaText tone="secondary">{standard.lastSync}</MetaText>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      </div>
    );
  };

  /** 配置对比 · 左栏补充项块数组：详情抽屉未覆盖、但需纳入对比的配置项（实例当前值）。
   *  「模型 / 通道 / 技能」已由上方真实详情区块展示，这里仅补充原型⑧多出的其余项，
   *  每项作为独立块返回，供对比网格与右栏逐块对齐。 */
  const renderInstanceExtraBlocks = (claw: Claw): ReactElement[] => {
    const items = getInstanceConfigCompareItems(claw.id);
    const extras = items.filter((it) => !["模型", "通道", "技能"].includes(it.category));
    return extras.map((it) => {
      // 不检查类目（技能安装来源 / 企业插件 / 企业MCP）不纳入符合判定，配置值统一用「-」展示
      const unchecked = isUncheckedCompareCategory(it.category);
      return (
      <div key={it.category}>
        <MetaText as="div" className="mb-2">{it.category}</MetaText>
        <SurfaceInner className="px-4 py-3">
          <div className="space-y-0.5">
            {unchecked ? (
              <div className="text-sm leading-relaxed text-[var(--text-secondary)]">-</div>
            ) : (
              it.instanceLines.map((line, i) => (
                <div
                  key={i}
                  className={`text-sm leading-relaxed break-words ${
                    line.diff ? "text-[var(--text-warning)] font-medium" : "text-[var(--text-secondary)]"
                  }`}
                >
                  {line.text}
                </div>
              ))
            )}
          </div>
        </SurfaceInner>
      </div>
      );
    });
  };

  /** 配置对比 · 右栏块数组：所属组织的标准配置镜像（只读），按 [身份, 计费, 模型, 通道, 技能, ...补充项] 顺序返回，与左栏逐块对齐。 */
  const buildOrgBlocks = (claw: Claw): ReactElement[] => {
    const items = getInstanceConfigCompareItems(claw.id);
    const groupName = getInstanceGroupName(claw);
    // 模型以「实例真实已应用模型」为准：未配置任何模型（空）即视为符合，避免与左栏真实区块（已应用模型 0）脱节
    const realModelEmpty = getClawDetail(claw.id).appliedModels.length === 0;
    // 某配置项是否「符合」：不检查项恒符合；实例未配置（空）也视为符合；模型为空也视为符合；其余取数据判定
    const effSame = (i: typeof items[number]) =>
      isUncheckedCompareCategory(i.category) ||
      i.instanceLines.length === 0 ||
      (i.category === "模型" && realModelEmpty) ||
      i.isSame;
    // 不检查项与空配置项均不计入「不符合」数量；整体按受检项是否全部符合判定
    const mismatchCount = items.filter((i) => !effSame(i)).length;
    const match = mismatchCount === 0;
    const isPayg = getClawBillingMode(claw) === "payg";

    const renderOrgSection = (
      title: string,
      lines: { text: string; diff?: boolean }[],
      isSame: boolean,
    ) => {
      const unchecked = isUncheckedCompareCategory(title);
      return (
      <div key={title}>
        <div className="flex items-center justify-between mb-2 gap-3">
          <MetaText as="div">{title}</MetaText>
          {unchecked ? (
            <StatusTag mode="soft" variant="gray" icon={<Minus />}>不检查</StatusTag>
          ) : isSame ? (
            <StatusTag mode="soft" variant="green" icon={<Check />}>符合</StatusTag>
          ) : (
            <StatusTag mode="soft" variant="orange" icon={<AlertTriangle />}>不符合</StatusTag>
          )}
        </div>
        <SurfaceInner className="px-4 py-3">
          <div className="space-y-0.5">
            {unchecked ? (
              <div className="text-sm leading-relaxed text-[var(--text-secondary)]">-</div>
            ) : (
              lines.map((line, i) => (
                <div
                  key={i}
                  className={`text-sm leading-relaxed break-words ${
                    line.diff ? "text-[var(--text-warning)] font-medium" : "text-[var(--text-secondary)]"
                  }`}
                >
                  {line.text}
                </div>
              ))
            )}
          </div>
        </SurfaceInner>
      </div>
      );
    };

    const findItem = (cat: string) => items.find((i) => i.category === cat);
    const extras = items.filter((it) => !["模型", "通道", "技能"].includes(it.category));
    const sectionFor = (cat: string) => {
      const it = findItem(cat);
      return it
        ? renderOrgSection(it.category, it.orgLines, effSame(it))
        : <div key={`org-${cat}`} />;
    };
    return [
      // 组织身份 + 整体结论（与左栏实例身份镜像）
      <div key="org-identity" className="min-w-0 space-y-1.5">
        <PanelTitle as="div" className="truncate leading-tight">组织配置</PanelTitle>
        <div className="flex items-center justify-between gap-3">
          <MetaText tone="weak" className="truncate">{groupName}</MetaText>
          {match ? (
            <StatusTag mode="soft" variant="green" icon={<Check />}>符合组织配置</StatusTag>
          ) : (
            <StatusTag mode="soft" variant="orange" icon={<AlertTriangle />}>{mismatchCount} 项不符合</StatusTag>
          )}
        </div>
      </div>,
      // 计费模式（组织标准，与左栏计费卡对齐）
      renderOrgSection("计费模式", [{ text: isPayg ? "按量计费" : "包年包月" }], true),
      // 模型 / 通道 / 技能（与左栏真实区块对齐）
      sectionFor("模型"),
      sectionFor("通道"),
      sectionFor("技能"),
      // 其余补充项（与左栏 renderInstanceExtraBlocks 同序镜像）
      ...extras.map((it) => renderOrgSection(it.category, it.orgLines, effSame(it))),
    ];
  };

  /** 配置对比网格：左右块按索引成对放入同一 grid 行，使每个区块顶部强制对齐（行高取左右较大者）。 */
  const renderCompareGrid = (claw: Claw, leftBlocks: ReactElement[]) => {
    const rightBlocks = buildOrgBlocks(claw);
    const rowCount = Math.max(leftBlocks.length, rightBlocks.length);
    const cells: ReactElement[] = [];
    for (let i = 0; i < rowCount; i++) {
      const borderCls = i === rowCount - 1 ? "" : "border-b border-[var(--cp-border)]";
      cells.push(
        <div key={`l-${i}`} className={`min-w-0 px-4 py-4 ${borderCls}`}>{leftBlocks[i] ?? null}</div>,
        <div key={`r-${i}`} className={`min-w-0 px-4 py-4 bg-[var(--bg-grey-normal)] ${borderCls}`}>{rightBlocks[i] ?? null}</div>,
      );
    }
    return (
      <div className="grid grid-cols-2 divide-x divide-[var(--cp-border)]">
        {cells}
      </div>
    );
  };

  const handleOpenMonitor = (claw: Claw) => {
    setSelectedClaw(claw);
    setShowMonitorDrawer(true);
  };

  const isStatusDisabled = (status: ClawStatus): boolean => {
    const available = getAvailableStatuses();
    return !available.includes(status);
  };

  return (
    <TooltipProvider delayDuration={200}>
      <div className="page-enter min-w-0">
        <AdminPageHeader
          title="Agent 列表"
          description="查看和管理所有企业用户创建的 Agent 云服务器。"
          actions={
            /* 页头右上：日期范围筛选 + 刷新
             * 停服态豁免：日期筛选、清除筛选、刷新列表都属查看类操作
             * （仅重新拉取展示数据，不产生业务变更），需保持 100% 不透明与正常交互。
             * "停服前已禁用则延续禁用"：刷新按钮自身的 disabled={refreshing} 加载态
             * 由页面控制，与全局停服态无关，data-billing-exempt 不影响其呈现与拦截。 */
            <div className="flex items-center gap-2" data-billing-exempt>
              <DatePicker
                value={dateFrom}
                onChange={(v) => { setDateFrom(v); setPage(1); }}
              />
              <span className="text-[var(--text-weak)] text-sm">—</span>
              <DatePicker
                value={dateTo}
                onChange={(v) => { setDateTo(v); setPage(1); }}
              />
              {(dateFrom || dateTo) && (
                <Button
                  variant="claw-outline"
                  size="claw"
                  onClick={() => { setDateFrom(""); setDateTo(""); setPage(1); }}
                  className="px-3 whitespace-nowrap"
                >
                  清除筛选
                </Button>
              )}
              <Button
                variant="claw-outline"
                size="icon"
                onClick={handleRefresh}
                disabled={refreshing}
                title="刷新列表"
                className="w-9 h-9"
              >
                <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
              </Button>
            </div>
          }
        />

        {/* 从存量处理弹窗跳转过来的筛选提示条 */}
        {pendingDeleteIds.size > 0 && (
          <div className="mb-4 flex items-center justify-between gap-3 px-4 py-2.5 rounded-[4px] bg-[var(--accent)] border border-[var(--border)]">
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-[var(--text-brand)]" />
              <span className="text-sm text-[var(--text-brand)]">
                已筛选 <span className="font-semibold">{pendingDeleteIds.size}</span> 个待删除的 Agent 实例
              </span>
            </div>
            <button
              className="text-xs text-[var(--text-brand)] hover:underline"
              onClick={() => {
                setPendingDeleteIds(new Set());
                window.history.replaceState({}, "", window.location.pathname);
              }}
            >
              清除筛选
            </button>
          </div>
        )}

        {/* 状态统计卡片 */}
        <div className="grid grid-cols-4 gap-5 mb-6">
          {[
            {
              key: "all" as const,
              label: "总数",
              value: totalCount,
              icon: <MonitorTotalIcon />,
            },
            {
              key: "running" as const,
              label: "运行中",
              value: runningCount,
              icon: <MonitorRunningIcon />,
            },
            {
              key: "shutdown" as const,
              label: "已关机",
              value: shutdownCount,
              icon: <MonitorShutdownIcon />,
            },
          ].map((card) => (
            <NumberCard
              key={card.key}
              role="button"
              tabIndex={0}
              aria-pressed={activeCardFilter === card.key}
              data-state={activeCardFilter === card.key ? "selected" : undefined}
              label={card.label}
              value={card.value}
              icon={card.icon}
              onClick={() => handleCardFilterChange(card.key)}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  handleCardFilterChange(card.key);
                }
              }}
              className="cursor-pointer select-none transition-all duration-200 hover:border-[var(--brand-blue)] hover:shadow-[var(--shadow-sm)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-blue)]"
            />
          ))}

          <HoverCard openDelay={120} closeDelay={120}>
            <HoverCardTrigger asChild>
              <NumberCard
                role="button"
                tabIndex={0}
                aria-pressed={activeCardFilter === "other"}
                data-state={activeCardFilter === "other" ? "selected" : undefined}
                label="其他"
                value={otherCount}
                icon={<MonitorOtherIcon />}
                onClick={() => handleCardFilterChange("other")}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    handleCardFilterChange("other");
                  }
                }}
                className="cursor-pointer select-none transition-all duration-200 hover:border-[var(--brand-blue)] hover:shadow-[var(--shadow-sm)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--brand-blue)]"
              />
            </HoverCardTrigger>
            <HoverCardContent
              side="bottom"
              align="end"
              sideOffset={12}
              className="w-[320px] rounded-[4px] border border-[var(--border)] bg-[var(--popover)] p-5 shadow-[var(--shadow-overlay)]"
            >
              <div className="space-y-4">
                {OTHER_STATUS_GROUPS.map((group, index) => (
                  <div
                    key={group.title}
                    className={index === 0 ? "space-y-2.5" : "space-y-2.5 border-t border-[var(--border)] pt-4"}
                  >
                    <MetaText as="div">{group.title}：</MetaText>
                    <div className="flex flex-wrap gap-x-4 gap-y-2">
                      {group.items.map((item) => (
                        <StatusTag
                          key={item.label}
                          mode="text"
                          variant={item.variant}
                        >
                          {item.label}
                        </StatusTag>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </HoverCardContent>
          </HoverCard>
        </div>

        {/* 工具栏（独立于表格卡片） */}
        <div className="flex items-center justify-between gap-2 flex-wrap mb-4">
            <div className="flex items-center gap-2 flex-1 min-w-0">
              {/* 搜索类型切换
                * 停服态豁免：切换搜索维度（名称/ID/用户ID/标签键/标签）属于查看类操作
                * （仅改变筛选口径，不产生业务变更），与关键词搜索同档，
                * 需保持 100% 不透明与正常交互。
                * 页面未给 Select 传 disabled，"停服前已禁用则延续禁用"
                * 约束通过 Radix 的 disabled 属性依然生效（此处无）。 */}
              <Select
                value={searchType}
                onValueChange={(value) => {
                  setSearchType(value as "name" | "id" | "creator" | "tagKey" | "tag");
                  setSearch("");
                  setSearchTagKeys([]);
                  setSearchTagKeyForValue("");
                  setSearchTagValues([]);
                  setTagKeySearchText("");
                  setTagValueSearchText("");
                  setPage(1);
                }}
              >
                <SelectTrigger className="w-[88px] shrink-0" data-billing-exempt>
                  <SelectValue />
                </SelectTrigger>
                {/* 下拉面板 Portal 到 body 下，不在 Trigger 祖先链内，
                  * 需在Content 自身上再加一次 data-billing-exempt，
                  * 才能让 5 个搜索维度项（名称/ID/用户ID/标签键/标签）在停服态下正常可选。
                  * SelectItem 未设置 disabled，"停服前已禁用则延续禁用"约束
                  * 通过 Radix 原生 disabled 属性依然生效（此处无）。 */}
                <SelectContent align="start" className="w-[120px]" data-billing-exempt>
                  <SelectItem value="name">名称</SelectItem>
                  <SelectItem value="id">ID</SelectItem>
                  <SelectItem value="creator">用户ID</SelectItem>
                  <SelectItem value="tagKey">标签键</SelectItem>
                  <SelectItem value="tag">标签</SelectItem>
                </SelectContent>
              </Select>

              {/* 文本搜索框（名称 / ID / 用户ID） */}
              {(searchType === 'name' || searchType === 'id' || searchType === 'creator') && (
                <div className="relative flex-1 max-w-sm">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
                  <Input
                    placeholder={
                      searchType === 'name' ? '搜索名称'
                        : searchType === 'id' ? '搜索实例 ID'
                        : '搜索用户ID'
                    }
                    value={search}
                    onChange={(e) => { setSearch(e.target.value); setPage(1); }}
                    className="pl-9 h-9"
                  />
                </div>
              )}

              {/* 按标签键搜索 */}
              {searchType === 'tagKey' && (
                <Popover open={tagSearchPopoverOpen} onOpenChange={setTagSearchPopoverOpen}>
                  <PopoverTrigger asChild>
                    <div className="flex-1 max-w-sm min-h-9 flex items-center flex-wrap gap-1 px-2 py-1 bg-[var(--popover)] border border-[var(--border)] rounded-[4px] cursor-pointer hover:border-[var(--text-brand)] transition-colors">
                      {searchTagKeys.length === 0 ? (
                        <span className="text-sm text-[var(--text-weak)]">多个标签键，用逗号分隔</span>
                      ) : (
                        searchTagKeys.map(k => (
                          <span key={k} className="inline-flex items-center gap-1 px-1.5 py-0.5 bg-[var(--accent)] text-[var(--text-brand)] text-xs rounded-[4px] border border-[var(--border)]">
                            {k}
                            <button onClick={(e) => { e.stopPropagation(); setSearchTagKeys(prev => prev.filter(x => x !== k)); setPage(1); }} className="hover:text-[var(--text-danger)]">×</button>
                          </span>
                        ))
                      )}
                    </div>
                  </PopoverTrigger>
                  <PopoverContent className="w-[320px] p-0" align="start">
                    <div className="p-2 border-b border-[var(--border)]">
                      <div className="relative">
                        <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
                        <Input
                          value={tagKeySearchText}
                          onChange={(e) => setTagKeySearchText(e.target.value)}
                          placeholder="搜索标签键..."
                          className="h-7 pl-7 pr-2 text-xs"
                          autoFocus
                        />
                      </div>
                    </div>
                    <div className="max-h-[220px] overflow-y-auto py-1">
                      {tagKeys.filter(k => !tagKeySearchText || k.toLowerCase().includes(tagKeySearchText.toLowerCase())).length === 0 ? (
                        <div className="flex flex-col items-center py-6 text-[var(--text-weak)] text-xs gap-1">
                          <span>找不到该标签</span>
                        </div>
                      ) : (
                        tagKeys.filter(k => !tagKeySearchText || k.toLowerCase().includes(tagKeySearchText.toLowerCase())).map(k => (
                          <div
                            key={k}
                            onClick={() => {
                              setSearchTagKeys(prev => prev.includes(k) ? prev.filter(x => x !== k) : [...prev, k]);
                              setPage(1);
                            }}
                            className="flex items-center gap-2 px-3 py-1.5 hover:bg-[var(--accent)] cursor-pointer"
                          >
                            <div className={`w-4 h-4 rounded-[4px] border flex items-center justify-center shrink-0 ${
                              searchTagKeys.includes(k) ? 'bg-[var(--text-brand)] border-[var(--text-brand)]' : 'border-[var(--border)]'
                            }`}>
                              {searchTagKeys.includes(k) && <Check className="w-2.5 h-2.5 text-white" />}
                            </div>
                            <span className="text-sm text-[var(--text-body)] truncate">{k}</span>
                          </div>
                        ))
                      )}
                    </div>
                    {searchTagKeys.length > 0 && (
                      <div className="border-t border-[var(--border)] px-3 py-2 flex justify-end">
                        <button onClick={() => { setSearchTagKeys([]); setPage(1); }} className="text-xs text-[var(--text-weak)] hover:text-[var(--text-danger)]">清空</button>
                      </div>
                    )}
                  </PopoverContent>
                </Popover>
              )}

              {/* 按标签搜索（键+值） */}
              {searchType === 'tag' && (
                <Popover open={tagSearchPopoverOpen} onOpenChange={setTagSearchPopoverOpen}>
                  <PopoverTrigger asChild>
                    <div className="flex-1 max-w-sm min-h-9 flex items-center flex-wrap gap-1 px-2 py-1 bg-[var(--popover)] border border-[var(--border)] rounded-[4px] cursor-pointer hover:border-[var(--text-brand)] transition-colors">
                      {!searchTagKeyForValue && searchTagValues.length === 0 ? (
                        <span className="text-sm text-[var(--text-weak)]">选择标签键和标签值</span>
                      ) : (
                        <span className="text-sm text-[var(--text-body)]">
                          {searchTagKeyForValue}
                          {searchTagValues.length > 0 && (
                            <span className="text-[var(--text-weak)]"> : {searchTagValues.join(', ')}</span>
                          )}
                        </span>
                      )}
                    </div>
                  </PopoverTrigger>
                  <PopoverContent className="w-[480px] p-0" align="start">
                    <div className="flex">
                      {/* 左侧：标签键列表 */}
                      <div className="w-[220px] border-r border-[var(--border)] flex flex-col">
                        <div className="px-2 py-2 border-b border-[var(--border)]">
                          <div className="relative">
                            <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
                            <Input
                              value={tagKeySearchText}
                              onChange={(e) => setTagKeySearchText(e.target.value)}
                              placeholder="搜索标签键..."
                              className="h-7 pl-7 pr-2 text-xs"
                              autoFocus
                            />
                          </div>
                        </div>
                        <div className="text-xs text-[var(--text-weak)] px-3 py-1.5 font-medium">标签键</div>
                        <div className="max-h-[220px] overflow-y-auto">
                          {tagKeys.filter(k => !tagKeySearchText || k.toLowerCase().includes(tagKeySearchText.toLowerCase())).length === 0 ? (
                            <div className="flex flex-col items-center py-6 text-[var(--text-weak)] text-xs gap-1">
                              <span>找不到该标签</span>
                            </div>
                          ) : (
                            tagKeys.filter(k => !tagKeySearchText || k.toLowerCase().includes(tagKeySearchText.toLowerCase())).map(k => (
                              <div
                                key={k}
                                onClick={() => {
                                  setSearchTagKeyForValue(k);
                                  setSearchTagValues([]);
                                  setTagValueSearchText('');
                                  setPage(1);
                                }}
                                className={`flex items-center gap-2 px-3 py-1.5 cursor-pointer text-sm ${
                                  searchTagKeyForValue === k ? 'bg-[var(--accent)] text-[var(--text-brand)] font-medium' : 'hover:bg-[var(--accent)] text-[var(--text-body)]'
                                }`}
                              >
                                {searchTagKeyForValue === k && <Check className="w-3 h-3 shrink-0" />}
                                <span className="truncate">{k}</span>
                              </div>
                            ))
                          )}
                        </div>
                      </div>
                      {/* 右侧：标签值列表 */}
                      <div className="flex-1 flex flex-col">
                        <div className="px-2 py-2 border-b border-[var(--border)]">
                          <div className="relative">
                            <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
                            <Input
                              value={tagValueSearchText}
                              onChange={(e) => setTagValueSearchText(e.target.value)}
                              placeholder="搜索标签值..."
                              className="h-7 pl-7 pr-2 text-xs"
                            />
                          </div>
                        </div>
                        <div className="text-xs text-[var(--text-weak)] px-3 py-1.5 font-medium">标签值</div>
                        <div className="max-h-[220px] overflow-y-auto">
                          {!searchTagKeyForValue ? (
                            <div className="flex flex-col items-center py-6 text-[var(--text-weak)] text-xs gap-1">
                              <span>请先选择标签键</span>
                            </div>
                          ) : (
                            (() => {
                              const vals = (tagKeyValues[searchTagKeyForValue] || []).filter(v => !tagValueSearchText || v.toLowerCase().includes(tagValueSearchText.toLowerCase()));
                              return vals.length === 0 ? (
                                <div className="flex flex-col items-center py-6 text-[var(--text-weak)] text-xs gap-1">
                                  <span>找不到该标签</span>
                                </div>
                              ) : (
                                vals.map(v => (
                                  <div
                                    key={v}
                                    onClick={() => {
                                      setSearchTagValues(prev => prev.includes(v) ? prev.filter(x => x !== v) : [...prev, v]);
                                      setPage(1);
                                    }}
                                    className="flex items-center gap-2 px-3 py-1.5 hover:bg-[var(--accent)] cursor-pointer"
                                  >
                                    <div className={`w-4 h-4 rounded-[4px] border flex items-center justify-center shrink-0 ${
                                      searchTagValues.includes(v) ? 'bg-[var(--text-brand)] border-[var(--text-brand)]' : 'border-[var(--border)]'
                                    }`}>
                                      {searchTagValues.includes(v) && <Check className="w-2.5 h-2.5 text-white" />}
                                    </div>
                                    <span className="text-sm text-[var(--text-body)] truncate">{v}</span>
                                  </div>
                                ))
                              );
                            })()
                          )}
                        </div>
                        {(searchTagKeyForValue || searchTagValues.length > 0) && (
                          <div className="border-t border-[var(--border)] px-3 py-2 flex justify-end">
                            <button onClick={() => { setSearchTagKeyForValue(''); setSearchTagValues([]); setPage(1); }} className="text-xs text-[var(--text-weak)] hover:text-[var(--text-danger)]">清空</button>
                          </div>
                        )}
                      </div>
                    </div>
                  </PopoverContent>
                </Popover>
              )}
            </div>
            <div className="flex items-center gap-2 shrink-0">
              {/* 新版本推送提醒（点击打开版本更新记录侧边栏，紧贴二级按钮） */}
              <ImageUpdateBellEntry onClick={() => setShowUpdateRecordsDrawer(true)} />
              <Button
                variant="claw-outline"
                size="claw"
                onClick={() => {
                  // 已有标签 → 加载为编辑行；无标签 → 一行空白
                  setPendingTags(selectedTags.length > 0 ? [...selectedTags] : [{ key: '', value: '', scope: 'all', groupIds: [] }]);
                  setKeySearchText('');
                  setOpenKeyRow(null);
                  setOpenValueRow(null);
                  setShowTagConfigDialog(true);
                }}
                className="px-3 gap-1.5"
              >
                <Tag className="w-3.5 h-3.5" />
                配置标签
              </Button>
              <Link href="/admin/agent-migration">
                <Button variant="claw-outline" size="claw" className="px-3 gap-1.5">
                  <ArrowLeftRight className="w-3.5 h-3.5" />
                  智能体迁移
                </Button>
              </Link>
            </div>
            <div className="ml-auto flex items-center gap-2 shrink-0">
            {/* 批量梳理：收拢批量更新 / 开关机 / 删除 / 组织处理 */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="claw-outline" size="claw" className="px-3 gap-1.5">
                  <ListChecks className="w-3.5 h-3.5" />
                  <span>{`批量处理${selectedCount > 0 ? `（${selectedCount}）` : ""}`}</span>
                  <ChevronDown className="w-3.5 h-3.5 ml-0.5 text-[var(--text-weak)]" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <DropdownMenuItem
                  disabled={batchDisabled}
                  className="cursor-pointer gap-2"
                  onSelect={(e) => {
                    if (batchDisabled) {
                      e.preventDefault();
                      if (batchTooltip) toast.error(batchTooltip);
                      return;
                    }
                    setShowBatchUpgradeDialog(true);
                  }}
                >
                  <CircleArrowUp className="w-3.5 h-3.5" />
                  <span>批量更新{selectedCount > 0 ? `（${selectedCount}）` : ""}</span>
                </DropdownMenuItem>
                <Tooltip delayDuration={200}>
                  <TooltipTrigger asChild>
                    <div>
                      <DropdownMenuItem
                        disabled={batchConfigModelDisabled}
                        className="cursor-pointer gap-2"
                        onSelect={(e) => {
                          if (batchConfigModelDisabled) {
                            e.preventDefault();
                            if (batchConfigModelTooltip) toast.error(batchConfigModelTooltip);
                            return;
                          }
                          openBatchConfigModelDialog();
                        }}
                      >
                        <Layers className="w-3.5 h-3.5" />
                        <span>
                          批量配置模型
                          {!batchConfigModelDisabled && runningConfigurableClaws.length > 0
                            ? `（${runningConfigurableClaws.length}${selectedCount > runningConfigurableClaws.length ? " · 仅运行中" : ""}）`
                            : ""}
                        </span>
                      </DropdownMenuItem>
                    </div>
                  </TooltipTrigger>
                  {batchConfigModelDisabled && batchConfigModelTooltip && (
                    <TooltipContent side="left" className="text-xs max-w-[260px]">
                      {batchConfigModelTooltip}
                    </TooltipContent>
                  )}
                </Tooltip>
                <DropdownMenuItem
                  disabled={batchShutdownDisabled}
                  className="cursor-pointer gap-2"
                  onSelect={(e) => {
                    if (batchShutdownDisabled) {
                      e.preventDefault();
                      if (batchShutdownTooltip) toast.error(batchShutdownTooltip);
                      return;
                    }
                    setShowBatchShutdownDialog(true);
                  }}
                >
                  <PowerOff className="w-3.5 h-3.5" />
                  <span>批量关机{!batchShutdownDisabled && runningSelectedClaws.length > 0 ? `（${runningSelectedClaws.length}）` : ""}</span>
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={batchPowerOnDisabled}
                  className="cursor-pointer gap-2"
                  onSelect={(e) => {
                    if (batchPowerOnDisabled) {
                      e.preventDefault();
                      if (batchPowerOnTooltip) toast.error(batchPowerOnTooltip);
                      return;
                    }
                    setShowBatchPowerOnDialog(true);
                  }}
                >
                  <Power className="w-3.5 h-3.5" />
                  <span>批量开机{!batchPowerOnDisabled && shutdownSelectedClaws.length > 0 ? `（${shutdownSelectedClaws.length}）` : ""}</span>
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={selectedCloudCount === 0}
                  className="cursor-pointer gap-2"
                  onSelect={() => {
                    if (selectedCloudCount === 0) return;
                    setAdjustConfigContext({ mode: "batch", claws: selectedCloudClaws });
                  }}
                >
                  <SlidersHorizontal className="w-3.5 h-3.5" />
                  <span>批量调整配置{selectedCloudCount > 0 ? `（${selectedCloudCount}）` : ""}</span>
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={batchDeleteDisabled}
                  className="cursor-pointer gap-2 text-[var(--text-danger)] focus:text-[var(--text-danger)]"
                  onSelect={(e) => {
                    if (batchDeleteDisabled) {
                      e.preventDefault();
                      toast.error("请先选择实例");
                      return;
                    }
                    setBatchDeleteInput("");
                    setShowBatchDeleteDialog(true);
                  }}
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  <span>批量删除{selectedCount > 0 ? `（${selectedCount}）` : ""}</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            {/* 组织处理：迁移/移交单独入口 */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="claw-outline" size="claw" className="px-3 gap-1.5">
                  <Users className="w-3.5 h-3.5" />
                  组织迁移/移交用户{selectedCount > 0 ? `（${selectedCount}）` : ""}
                  <ChevronDown className="w-3.5 h-3.5 ml-0.5 text-[var(--text-weak)]" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                <Tooltip delayDuration={200}>
                  <TooltipTrigger asChild>
                    <div>
                      <DropdownMenuItem
                        disabled={selectedCount === 0}
                        className={`cursor-pointer gap-2 ${migrateBlocked ? "opacity-50 !cursor-not-allowed" : ""}`}
                        onSelect={(e) => {
                          if (selectedCount === 0) {
                            e.preventDefault();
                            toast.error("请先选择需要处理的实例");
                            return;
                          }
                          if (migrateBlocked) {
                            e.preventDefault();
                            toast.error("仅支持选择同一个用户的实例执行批量迁移");
                            return;
                          }
                          setGroupMigrateTarget(firstSelectableMigrateGroup);
                          setShowGroupMigrateDialog(true);
                        }}
                      >
                        <ArrowLeftRight className="w-3.5 h-3.5" />
                        <span>随用户迁移到新组织</span>
                      </DropdownMenuItem>
                    </div>
                  </TooltipTrigger>
                  {(selectedCount === 0 || migrateBlocked) && (
                    <TooltipContent side="left" className="text-xs">
                      {selectedCount === 0 ? "请先选择 Agent 实例" : "仅支持选择同一个用户的实例执行批量迁移"}
                    </TooltipContent>
                  )}
                </Tooltip>
                <Tooltip delayDuration={200}>
                  <TooltipTrigger asChild>
                    <div>
                      <DropdownMenuItem
                        disabled={selectedCount === 0}
                        className="cursor-pointer gap-2"
                        onSelect={(e) => {
                          if (selectedCount === 0) {
                            e.preventDefault();
                            toast.error("请先选择需要处理的实例");
                            return;
                          }
                          setGroupTransferTarget("");
                          setGroupTransferTargetGroup("");
                          setTransferPickerGroupId("");
                          setTransferSearchField("userId");
                          setTransferSearchKeyword("");
                          setTransferTreeExpanded(new Set(transferGroups.map((g) => g.id)));
                          setShowGroupTransferDialog(true);
                        }}
                      >
                        <Users className="w-3.5 h-3.5" />
                        <span>移交给其他用户</span>
                      </DropdownMenuItem>
                    </div>
                  </TooltipTrigger>
                  {selectedCount === 0 && (
                    <TooltipContent side="left" className="text-xs">
                      请先选择 Agent 实例
                    </TooltipContent>
                  )}
                </Tooltip>
                <DropdownMenuSeparator />
                {/* 筛选组织异常的 Agent 实例：仅展示带红点（用户不在该组织）或橙点（配置不符合）的实例 */}
                <DropdownMenuItem
                  className="cursor-pointer gap-2"
                  onSelect={() => { setAnomalyFilter((v) => !v); setPage(1); }}
                >
                  <Filter className={`w-3.5 h-3.5 ${anomalyFilter ? "text-[var(--text-brand)]" : ""}`} />
                  <span className={anomalyFilter ? "text-[var(--text-brand)]" : ""}>筛选组织异常的Agent</span>
                  {anomalyFilter && <Check className="w-3.5 h-3.5 ml-auto text-[var(--text-brand)]" />}
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  disabled={selectedCount === 0}
                  className="cursor-pointer gap-2"
                  onSelect={(e) => {
                    if (selectedCount === 0) {
                      e.preventDefault();
                      toast.error("请先选择需要处理的实例");
                      return;
                    }
                    setGroupTransferTarget("");
                    setShowGroupMigrateDialog(true);
                  }}
                >
                  <ArrowLeftRight className="w-3.5 h-3.5" />
                  <span>迁移到新组织</span>
                </DropdownMenuItem>
                <DropdownMenuItem
                  disabled={selectedCount === 0}
                  className="cursor-pointer gap-2"
                  onSelect={(e) => {
                    if (selectedCount === 0) {
                      e.preventDefault();
                      toast.error("请先选择需要处理的实例");
                      return;
                    }
                    setGroupTransferTarget("");
                    setShowGroupTransferDialog(true);
                  }}
                >
                  <Users className="w-3.5 h-3.5" />
                  <span>移交给同组织其他用户</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
            {/* 命令下发：
              * - 勾选实例时：变为主按钮，点击直接打开下发弹窗（预填实例 → 让用户挑命令）
              * - 未勾选时：保持二级菜单，命令列表/执行记录跳转到独立页 /admin/agent-commands
              */}
            {selectedCount > 0 ? (
              <Tooltip delayDuration={200}>
                <TooltipTrigger asChild>
                  <Button
                    onClick={() => {
                      // 仅取运行中的实例，过滤掉异常状态
                      const runningIds = selectedClaws
                        .filter((c) => c.status === "running")
                        .map((c) => c.instanceId);
                      if (runningIds.length === 0) {
                        toast.error("所选实例中没有运行中的 Agent，无法下发命令");
                        return;
                      }
                      if (runningIds.length < selectedCount) {
                        toast.info(`已自动跳过 ${selectedCount - runningIds.length} 台非运行中实例`);
                      }
                      setDispatchPresetIds(runningIds);
                    }}
                    variant="claw-primary"
                    size="claw"
                    className="px-3 gap-1.5"
                  >
                    <TerminalSquare className="w-3.5 h-3.5" />
                    命令下发（{selectedCount}）
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="text-xs">
                  对已选 {selectedCount} 台实例下发命令
                </TooltipContent>
              </Tooltip>
            ) : (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="claw-outline" size="claw" className="px-3 gap-1.5">
                    <TerminalSquare className="w-3.5 h-3.5" />
                    命令下发
                    <ChevronDown className="w-3.5 h-3.5 ml-0.5 text-[var(--text-weak)]" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  <DropdownMenuItem
                    className="cursor-pointer py-2.5"
                    onClick={() => setDispatchPresetIds([])}
                  >
                    <div>
                      <BodyMedium className="block">下发命令</BodyMedium>
                      <MetaText className="mt-0.5 block">挑选命令模板并选择目标实例</MetaText>
                    </div>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    className="cursor-pointer py-2.5"
                    onClick={() => setLocation("/admin/agent-commands?tab=list")}
                  >
                    <div>
                      <BodyMedium className="block">命令列表</BodyMedium>
                      <MetaText className="mt-0.5 block">管理命令模板（沉淀团队 SOP）</MetaText>
                    </div>
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    className="cursor-pointer py-2.5"
                    onClick={() => setLocation("/admin/agent-commands?tab=history")}
                  >
                    <div>
                      <BodyMedium className="block">执行记录</BodyMedium>
                      <MetaText className="mt-0.5 block">查看历史下发任务与单机输出</MetaText>
                    </div>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
            {/* 创建 Agent：最右主按钮，与 agent 列表右边缘对齐；其余三个按钮（批量处理 / 组织处理 / 命令下发）通过 ml-auto 整体左移 */}
            <Button
              variant="claw-primary"
              size="claw"
              className="px-3 gap-1.5"
              onClick={() => setShowCreateAgent(true)}
            >
              <Plus className="w-3.5 h-3.5" />
              创建 Agent
            </Button>
            </div>
          </div>

        {/* 表格卡片 */}
        <SurfaceCard className="overflow-hidden">
          <Table
            variant="white"
            containerRef={tableScrollRef}
            className="text-sm"
            scrollX="max-content"
          >
            <TableHeader>
              <TableRow>
                {/* 复选框列 - 固定左侧（多列同侧固定的第一列，不显示阴影） */}
                <TableHead fixed="left" fixedShadow={false} className="whitespace-nowrap px-4" style={{ width: '56px', minWidth: '56px' }}>
                  <div className="flex items-center">
                    <Checkbox
                      checked={isAllSelected ? true : isIndeterminate ? "indeterminate" : false}
                      onCheckedChange={(v) => handleSelectAll(!!v)}
                      aria-label="全选"
                    />
                  </div>
                </TableHead>
                {/* 名称 / ID 列 - 固定左侧（边界列，显示阴影），偏移 56px 错开复选框列 */}
                <TableHead fixed="left" className="whitespace-nowrap px-4" style={{ left: 56, width: '200px', minWidth: '200px', maxWidth: '200px' }}>名称 / ID</TableHead>
                <TableHead className="whitespace-nowrap" style={{ minWidth: '120px' }}>
                  <Popover open={showStatusFilter} onOpenChange={setShowStatusFilter}>
                    <PopoverTrigger asChild>
                      <button className="flex items-center gap-1 group/status">
                        <span>当前状态</span>
                        <Filter className={`w-3.5 h-3.5 transition-colors ${selectedStatuses.size > 0 && selectedStatuses.size < ALL_STATUSES.length ? 'text-[var(--text-brand)]' : 'text-[var(--text-weak)] group-hover/status:text-[var(--text-muted)]'}`} />
                      </button>
                    </PopoverTrigger>
                    <PopoverContent className="w-56 p-0" align="start" side="bottom">
                      <div className="max-h-64 space-y-2 overflow-y-auto p-3">
                        {ALL_STATUSES.map((status) => {
                          const disabled = isStatusDisabled(status);
                          return (
                            <label key={status} className={`flex items-center gap-2 ${disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}>
                              <Checkbox
                                checked={selectedStatuses.has(status)}
                                onCheckedChange={(checked) => handleStatusFilterChange(status, !!checked)}
                                disabled={disabled}
                              />
                              <span className={`text-sm ${disabled ? "text-[var(--text-weak)]" : "text-[var(--text-secondary)]"}`}>
                                {STATUS_CONFIG[status].label}
                              </span>
                            </label>
                          );
                        })}
                      </div>
                      <div className="flex gap-2 border-t border-[var(--border)] p-2">
                        <Button variant="claw-outline" size="claw-sm" onClick={handleStatusFilterReset} className="flex-1">
                          重置
                        </Button>
                        <Button variant="dialog-confirm" size="claw-sm" onClick={handleStatusFilterConfirm} className="flex-1">
                          确认
                        </Button>
                      </div>
                    </PopoverContent>
                  </Popover>
                </TableHead>
                <TableHead className="whitespace-nowrap" style={{ minWidth: '120px' }}>
                  <Popover open={showSpecFilter} onOpenChange={(v) => { setShowSpecFilter(v); if (v) setPendingSpecs(new Set(selectedSpecs)); }}>
                    <PopoverTrigger asChild>
                      <button className="flex items-center gap-1 group/spec">
                        <span>实例规格</span>
                        <Filter className={`w-3.5 h-3.5 transition-colors ${selectedSpecs.size > 0 && selectedSpecs.size < ADJUST_SPEC_OPTIONS.length ? 'text-[var(--text-brand)]' : 'text-[var(--text-weak)] group-hover/spec:text-[var(--text-muted)]'}`} />
                      </button>
                    </PopoverTrigger>
                    <PopoverContent className="w-[280px] p-0" align="start" side="bottom">
                      <SelectPanel
                        commitMode="confirm"
                        showSearch={false}
                        showFooter={true}
                        onConfirm={handleSpecFilterConfirm}
                        onCancel={handleSpecFilterCancel}
                        footerLeft={
                          <MetaText>{pendingSpecs.size === ADJUST_SPEC_OPTIONS.length ? "已选全部" : `已选 ${pendingSpecs.size} 项`}</MetaText>
                        }
                        footerRight={
                          <div className="flex items-center gap-1.5">
                            <Button variant="claw-outline" size="sm" className="text-xs h-7 px-2" onClick={handleSpecFilterReset}>重置</Button>
                            <Button variant="dialog-confirm" size="sm" className="text-xs h-7 px-3" onClick={handleSpecFilterConfirm}>确认</Button>
                          </div>
                        }
                      >
                        <SelectPanelItem selected={pendingSpecs.size === ADJUST_SPEC_OPTIONS.length} onClick={handleSpecFilterToggleAll}>
                          <Checkbox checked={pendingSpecs.size === ADJUST_SPEC_OPTIONS.length} className="pointer-events-none" />
                          <span className="text-sm whitespace-nowrap">全部</span>
                        </SelectPanelItem>
                        <div className="mx-1 my-1 border-t border-[#EAEEF4]" />
                        {ADJUST_SPEC_OPTIONS.map((o) => (
                          <SelectPanelItem
                            key={o.value}
                            selected={pendingSpecs.has(o.value)}
                            onClick={() => handleSpecFilterToggle(o.value)}
                          >
                            <Checkbox checked={pendingSpecs.has(o.value)} className="pointer-events-none" />
                            <span className="text-sm whitespace-nowrap">{o.label}</span>
                          </SelectPanelItem>
                        ))}
                      </SelectPanel>
                    </PopoverContent>
                  </Popover>
                </TableHead>
                <TableHead className="whitespace-nowrap" style={{ minWidth: '120px' }}>系统盘类型</TableHead>
                <TableHead className="whitespace-nowrap" style={{ minWidth: '120px' }}>
                  <Popover open={showDiskFilter} onOpenChange={setShowDiskFilter}>
                    <PopoverTrigger asChild>
                      <button className="flex items-center gap-1 group/disk">
                        <span>系统盘容量</span>
                        <Filter className={`w-3.5 h-3.5 transition-colors ${diskCapacityFilter ? 'text-[var(--text-brand)]' : 'text-[var(--text-weak)] group-hover/disk:text-[var(--text-muted)]'}`} />
                      </button>
                    </PopoverTrigger>
                    <PopoverContent className="w-64 p-0" align="start" side="bottom">
                      <div className="p-3 space-y-3">
                        <MetaText>系统盘容量</MetaText>
                        <div className="flex items-center gap-2">
                          <Select value={tempDiskCond} onValueChange={(v) => setTempDiskCond(v as "<" | "=" | ">")}>
                            <SelectTrigger className="w-20 h-8 text-xs"><SelectValue /></SelectTrigger>
                            <SelectContent>
                              <SelectItem value="<">小于</SelectItem>
                              <SelectItem value="=">等于</SelectItem>
                              <SelectItem value=">">大于</SelectItem>
                            </SelectContent>
                          </Select>
                          <Input type="number" min={1} step={1} value={tempDiskValue} onChange={(e) => setTempDiskValue(e.target.value.replace(/\D/g, ""))} placeholder="容量" className="flex-1 h-8 text-xs" />
                          <span className="text-xs text-[var(--text-muted)] shrink-0">GiB</span>
                        </div>
                      </div>
                      <div className="flex gap-2 border-t border-[var(--border)] p-2">
                        <Button variant="claw-outline" size="claw-sm" onClick={handleDiskFilterReset} className="flex-1">重置</Button>
                        <Button variant="dialog-confirm" size="claw-sm" onClick={handleDiskFilterConfirm} className="flex-1">确认</Button>
                      </div>
                    </PopoverContent>
                  </Popover>
                </TableHead>
                <TableHead className="whitespace-nowrap" style={{ width: '208px', minWidth: '160px', maxWidth: '208px' }}>用户ID</TableHead>
                {hasOneid && (
                  <TableHead className="whitespace-nowrap" style={{ width: 200, maxWidth: 200 }}>
                    <Popover open={deptColFilterOpen} onOpenChange={setDeptColFilterOpen}>
                      <PopoverTrigger asChild>
                        <button className="flex items-center gap-1 group/dept">
                          <span>部门</span>
                          <Filter className={`w-3.5 h-3.5 transition-colors ${departmentFilter ? 'text-[var(--text-brand)]' : 'text-[var(--text-weak)] group-hover/dept:text-[var(--text-muted)]'}`} />
                        </button>
                      </PopoverTrigger>
                      <PopoverContent className="w-[280px] p-0" align="start" side="bottom">
                        <DepartmentColumnFilter
                          departments={MOCK_DEPARTMENTS}
                          value={departmentFilter}
                          onConfirm={(v) => { setDepartmentFilter(v); setPage(1); setDeptColFilterOpen(false); }}
                          onCancel={() => setDeptColFilterOpen(false)}
                        />
                      </PopoverContent>
                    </Popover>
                  </TableHead>
                )}
                <TableHead className="whitespace-nowrap" style={{ width: 200, maxWidth: 200 }}>
                  <Popover open={groupColFilterOpen} onOpenChange={setGroupColFilterOpen}>
                    <PopoverTrigger asChild>
                      <button className="flex items-center gap-1 group/grp">
                        <span>组织</span>
                        <Filter className={`w-3.5 h-3.5 transition-colors ${groupFilter ? 'text-[var(--text-brand)]' : 'text-[var(--text-weak)] group-hover/grp:text-[var(--text-muted)]'}`} />
                      </button>
                    </PopoverTrigger>
                    <PopoverContent className="w-[280px] p-0" align="start" side="bottom">
                      <GroupColumnFilter
                        groups={hasOneid ? MOCK_GROUPS : MOCK_MANUAL_GROUPS}
                        value={groupFilter}
                        hasOneid={hasOneid}
                        specialOptions={[
                          { value: "__mismatch__", label: "用户已不在该组织", description: "筛选组织关系异常的 Agent" },
                          { value: "__pending_user__", label: "待用户处理", description: "筛选需要用户确认迁移或移交的 Agent" },
                        ]}
                        onConfirm={(v) => { setGroupFilter(v); setPage(1); setGroupColFilterOpen(false); }}
                        onCancel={() => setGroupColFilterOpen(false)}
                      />
                    </PopoverContent>
                  </Popover>
                </TableHead>
                <TableHead className="whitespace-nowrap" style={{ width: 200, maxWidth: 200 }}>项目</TableHead>
                <TableHead className="whitespace-nowrap" style={{ width: 200, maxWidth: 200 }}>共享范围</TableHead>
                <TableHead className="whitespace-nowrap" style={{ minWidth: '140px' }}>创建时间</TableHead>
                <TableHead className="whitespace-nowrap" style={{ minWidth: '130px' }}>
                  <Popover open={typeColFilterOpen} onOpenChange={(open) => {
                    setTypeColFilterOpen(open);
                    if (open) setTempTypeFilter(new Set(agentTypeFilter));
                  }}>
                    <PopoverTrigger asChild>
                      <button className="flex items-center gap-1 group/type">
                        <span>Agent类型</span>
                        <Filter className={`w-3.5 h-3.5 transition-colors ${agentTypeFilter.size > 0 && agentTypeFilter.size < ALL_AGENT_TYPES.length ? 'text-[var(--text-brand)]' : 'text-[var(--text-weak)] group-hover/type:text-[var(--text-muted)]'}`} />
                      </button>
                    </PopoverTrigger>
                    <PopoverContent className="w-[280px] p-0" align="start" side="bottom">
                      <div className="p-2 space-y-2">
                        {AGENT_TYPE_FILTER_GROUPS.map((group) => {
                          const groupKeys = group.items.map((item) => item.key);
                          const groupChecked = groupKeys.every((key) => tempTypeFilter.has(key));
                          return (
                            <div key={group.key} className="space-y-0.5">
                              <label className="flex items-center gap-2 px-2 py-1.5 rounded-[4px] cursor-pointer transition-colors hover:bg-[var(--accent)]">
                                <Checkbox
                                  checked={groupChecked}
                                  onCheckedChange={(checked) => {
                                    setTempTypeFilter(prev => {
                                      const next = new Set(prev);
                                      groupKeys.forEach((key) => {
                                        if (checked) next.add(key);
                                        else next.delete(key);
                                      });
                                      return next;
                                    });
                                  }}
                                />
                                <span className="text-sm font-medium text-[var(--text-title)]">{group.label}</span>
                              </label>
                              <div className="pl-6 space-y-0.5">
                                {group.items.map((item) => (
                                  <label
                                    key={item.key}
                                    className="flex items-center gap-2 px-2 py-1.5 rounded-[4px] cursor-pointer transition-colors hover:bg-[var(--accent)]"
                                  >
                                    <Checkbox
                                      checked={tempTypeFilter.has(item.key)}
                                      onCheckedChange={(checked) => {
                                        setTempTypeFilter(prev => {
                                          const next = new Set(prev);
                                          if (checked) next.add(item.key); else next.delete(item.key);
                                          return next;
                                        });
                                      }}
                                    />
                                    <span className="text-sm text-[var(--text-secondary)]">{item.label}</span>
                                  </label>
                                ))}
                              </div>
                            </div>
                          );
                        })}
                      </div>
                      <div className="border-t border-[var(--border)] p-2 flex gap-2">
                        <Button variant="claw-outline" size="claw-sm" className="flex-1" onClick={() => {
                          setTempTypeFilter(new Set(ALL_AGENT_TYPES));
                          setAgentTypeFilter(new Set(ALL_AGENT_TYPES));
                          setPage(1);
                          setTypeColFilterOpen(false);
                        }}>重置</Button>
                        <Button variant="dialog-confirm" size="claw-sm" className="flex-1" onClick={() => {
                          setAgentTypeFilter(new Set(tempTypeFilter));
                          setPage(1);
                          setTypeColFilterOpen(false);
                        }}>确认</Button>
                      </div>
                    </PopoverContent>
                  </Popover>
                </TableHead>
                <TableHead className="whitespace-nowrap" style={{ minWidth: '100px' }}>Agent 版本</TableHead>
                {hasAnyTagColumn && (
                  <TableHead className="whitespace-nowrap" style={{ minWidth: '60px' }}>标签</TableHead>
                )}
                <TableHead fixed="right" className="whitespace-nowrap" style={{ width: '240px', minWidth: '240px' }}>
                  操作
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginated.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={(hasOneid ? 16 : 15) - (hasAnyTagColumn ? 0 : 1)} className="px-6 py-12 text-center text-sm text-[var(--text-weak)]">
                    暂无符合条件的 Agent
                  </TableCell>
                </TableRow>
              ) : (
                paginated.map((claw) => {
                  const isLocalAgent = claw.agentType === "LocalAgent";
                  const info = getInstanceSpecInfo(claw);
                  const rUpdate = resourceUpdates[claw.id];
                  const displaySpec = rUpdate?.spec ?? info.spec;
                  const displayDiskCapacity = rUpdate?.diskSize ?? info.systemDiskCapacity;
                  const localConnectionMeta = getLocalConnectionMeta(claw);
                  const isRunning = claw.status === "running";
                  const statusConfig = STATUS_CONFIG[claw.status];

                  const upgradable = isUpgradable(claw);
                  const checkboxDisabled = isLocalAgent;
                  const checkboxTooltip = isLocalAgent ? "本地 Agent 暂不支持批量处理、命令下发等云端能力" : "";
                  // 调整配置 mock 操作状态：processing 整行置灰 + 操作禁用；failed 不锁定、可再次调整
                  const adjustOp = adjustOperationMap[claw.id];
                  const opLocked = adjustOp?.status === "processing";
                  const lockTip = "当前实例正在调整配置，暂不支持其他操作";

                  return (
                    <TableRow
                      key={claw.id}
                      className={cn("cursor-pointer", opLocked && "opacity-60 cursor-default")}
                      onClick={(e) => {
                        // 点击行内交互元素（按钮 / 链接 / 输入框 / 复选框等）时跳过，避免误触
                        if ((e.target as HTMLElement).closest('button, a, input, label, [role="checkbox"], [data-no-row-select]')) {
                          return;
                        }
                        if (!checkboxDisabled && !opLocked) handleSelectOne(claw.id, !selectedIds.has(claw.id));
                      }}
                    >
                      {/* 复选框 - 固定左侧（非边界列） */}
                      <TableCell fixed="left" fixedShadow={false} className="py-4 px-4 whitespace-nowrap" style={{ width: '56px', minWidth: '56px' }}>
                        {checkboxDisabled || opLocked ? (
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="inline-flex cursor-not-allowed">
                                <Checkbox checked={false} disabled aria-label={checkboxDisabled ? checkboxTooltip : lockTip} />
                              </span>
                            </TooltipTrigger>
                            <TooltipContent side="right" className="text-xs max-w-[240px]">
                              {checkboxDisabled ? checkboxTooltip : lockTip}
                            </TooltipContent>
                          </Tooltip>
                        ) : (
                          <Checkbox
                            checked={selectedIds.has(claw.id)}
                            onCheckedChange={(v) => handleSelectOne(claw.id, !!v)}
                          />
                        )}
                      </TableCell>
                      {/* 名称/ID - 固定左侧（边界列），偏移 56px */}
                      <TableCell fixed="left" className="px-4 py-4" style={{ left: 56, width: '200px', minWidth: '200px', maxWidth: '200px' }}>
                        <div className="flex items-center gap-2.5 min-w-0">
                          <div className="min-w-0 flex-1">
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <MiniBodyText as="div" className="truncate max-w-[130px]">{claw.name}</MiniBodyText>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="text-xs max-w-xs break-all">{claw.name}</TooltipContent>
                            </Tooltip>
                            <button
                              onClick={() => handleOpenDrawer(claw)}
                              className="text-[12px] font-mono cursor-pointer text-[var(--text-brand)] hover:underline"
                            >
                              {claw.instanceId}
                            </button>
                          </div>
                        </div>
                      </TableCell>
                      {/* 状态列 */}
                      <TableCell className="px-4 py-4">
                        <StatusTag mode="text" variant={isLocalAgent ? localConnectionMeta.variant : statusConfig.tagVariant}>
                          {isLocalAgent ? localConnectionMeta.label : statusConfig.label}
                        </StatusTag>
                        {adjustOp?.status === "processing" && (
                          <div className="mt-1 flex items-center gap-1 text-xs text-[var(--text-muted)]">
                            <Loader2 className="w-3 h-3 animate-spin" />
                            调整配置中
                          </div>
                        )}
                        {adjustOp?.status === "failed" && (
                          <div className="mt-1">
                            <InfoPopover
                              content={adjustOp.errorCode ? `您的实例调整配置失败，建议稍后重试。错误码：${adjustOp.errorCode}` : "您的实例调整配置失败，建议稍后重试。"}
                              placement="top"
                            >
                              <span className="flex items-center gap-1 text-xs text-[var(--destructive)] cursor-pointer">
                                <CircleAlert className="w-3 h-3" />
                                调整配置失败
                              </span>
                            </InfoPopover>
                          </div>
                        )}
                      </TableCell>
                      {/* 实例规格（云端展示简洁格式；本地 Agent 展示「—」） */}
                      <TableCell className="px-4 py-4 text-sm text-[var(--text-secondary)] whitespace-nowrap">
                        {isLocalAgent ? "—" : formatSpecShort(displaySpec)}
                      </TableCell>
                      {/* 系统盘类型 */}
                      <TableCell className="px-4 py-4 text-sm text-[var(--text-secondary)] whitespace-nowrap">
                        {isLocalAgent ? "—" : info.systemDiskType}
                      </TableCell>
                      {/* 系统盘容量（单位统一 GiB） */}
                      <TableCell className="px-4 py-4 text-sm text-[var(--text-secondary)] tabular-nums whitespace-nowrap">
                        {isLocalAgent ? "—" : `${displayDiskCapacity}GiB`}
                      </TableCell>
                      {/* 用户ID：该 Agent 的归属用户 ID（管控端代建时 = 管理员在弹窗中分配的目标用户） */}
                      <TableCell className="px-4 py-4" style={{ maxWidth: '208px' }}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="block truncate cursor-default">{claw.creator}</span>
                          </TooltipTrigger>
                          <TooltipContent side="bottom" align="start">
                            <span className="text-xs">{claw.creator}</span>
                          </TooltipContent>
                        </Tooltip>
                      </TableCell>
                      {/* 部门 - 仅 OneID 模式显示。部门归属跟随 creator（= 归属用户 ID） */}
                      {hasOneid && (
                        <TableCell className="px-4 py-4">
                          {(() => {
                            const deptPaths = getCreatorDeptPaths(claw.creator);
                            if (deptPaths.length === 0) return <span className="text-sm text-[var(--text-weak)]">—</span>;
                            if (deptPaths.length === 1) {
                              return (
                                <span className="text-sm text-[var(--text-secondary)] truncate block max-w-[200px]" title={deptPaths[0].path}>
                                  {deptPaths[0].path}
                                </span>
                              );
                            }
                            return (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="inline-flex items-center gap-1 max-w-[200px] cursor-default">
                                    <span className="text-sm text-[var(--text-secondary)] truncate">{deptPaths[0].path}</span>
                                    <span className="text-xs text-[var(--text-muted)] tabular-nums shrink-0">+{deptPaths.length - 1}</span>
                                  </span>
                                </TooltipTrigger>
                                <TooltipContent side="bottom" align="start" className="max-w-[360px] p-0">
                                  <div className="py-2">
                                    {deptPaths.map((dp, idx) => (
                                      <div key={idx} className="px-3 py-1.5 text-sm">
                                        <span className="text-[var(--text-weak)] mr-1">{idx + 1}.</span>
                                        <span className="text-white">{dp.path}</span>
                                        {dp.isPrimary && (
                                          <span className="ml-2 inline-flex items-center text-[10px] font-medium text-[var(--text-brand)] bg-[var(--accent)] rounded-[4px] px-1.5 py-0.5">
                                            主部门
                                          </span>
                                        )}
                                      </div>
                                    ))}
                                  </div>
                                </TooltipContent>
                              </Tooltip>
                            );
                          })()}
                        </TableCell>
                      )}
                      {/* 组织 - 优先取 Agent 自身固定绑定的分组（管理员创建时选的组），
                          存量 mock 无 groupId 时回退按归属用户反查 */}
                      <TableCell className="px-4 py-4 whitespace-nowrap">
                        {(() => {
                          const pendingUser = MOCK_GROUP_PENDING_USER_INSTANCES.has(claw.id);
                          // 待用户处理的实例必然也是「用户已不在该组织」，故 pending 隐含 mismatch
                          const groupMismatch = MOCK_GROUP_MISMATCH_INSTANCES.has(claw.id) || pendingUser;
                          // 配置不符合组织 → 橙点
                          const configMismatch = !isInstanceConfigMatch(claw.id);
                          const currentGroup = MOCK_INSTANCE_CURRENT_GROUP[claw.id] ?? "—";
                          const pendingActions = [
                            MOCK_PENDING_USER_ACTIONS[claw.id]?.allowMigrate ? "随用户迁移到新组织" : null,
                            MOCK_PENDING_USER_ACTIONS[claw.id]?.allowTransfer ? "移交给同组织其他用户" : null,
                          ].filter(Boolean).join("、");
                          // 红点（用户不在该组织，tooltip 合并「允许用户自行处理」） + 橙点（配置不符合，点击打开对比抽屉）
                          const dots = (
                            <>
                              {groupMismatch && (
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <span
                                      data-no-row-select
                                      className="ml-1 inline-flex items-center cursor-default"
                                    >
                                      <span className="w-2 h-2 rounded-full bg-[#DC2626]" />
                                    </span>
                                  </TooltipTrigger>
                                  <TooltipContent side="bottom" align="start" className="text-xs max-w-[300px]">
                                    <div className="space-y-0.5">
                                      <div>用户曾用该组织创建了 Agent 实例，但用户目前已不在该组织，请尽快将实例迁移至新组织或移交给原组织的其他用户。</div>
                                      <div className="mt-1.5 flex items-start gap-1.5">
                                        <span className="mt-[5px] w-1 h-1 rounded-full bg-white/60 shrink-0" />
                                        <span>用户当前组织：{currentGroup}</span>
                                      </div>
                                      {pendingUser && pendingActions && (
                                        <div className="flex items-start gap-1.5">
                                          <span className="mt-[5px] w-1 h-1 rounded-full bg-white/60 shrink-0" />
                                          <span>允许用户自行处理：{pendingActions}</span>
                                        </div>
                                      )}
                                    </div>
                                  </TooltipContent>
                                </Tooltip>
                              )}
                              {/* 同时存在红点与橙点时，仅展示红点；红点消除后再展示橙点 */}
                              {configMismatch && !groupMismatch && (
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <span
                                      data-no-row-select
                                      className="ml-1 inline-flex items-center cursor-default"
                                    >
                                      <span className="w-2 h-2 rounded-full bg-[#F59E0B]" />
                                    </span>
                                  </TooltipTrigger>
                                  <TooltipContent side="bottom" align="start" className="text-xs max-w-[300px]">
                                    <div className="space-y-1">
                                      <div>该Agent实例的配置与其所属组织的配置不一致</div>
                                      <button
                                        type="button"
                                        onClick={() => { setSelectedClaw(claw); setCompareMode(true); setShowDetailDrawer(true); }}
                                        className="inline-flex items-center gap-1 text-blue-300 hover:text-blue-200 cursor-pointer"
                                      >
                                        <Eye className="w-3.5 h-3.5 shrink-0" />
                                        <span>点击查看实例配置与组织配置对比</span>
                                      </button>
                                    </div>
                                  </TooltipContent>
                                </Tooltip>
                              )}
                            </>
                          );
                          if (hasOneid) {
                            const item = getClawGroupItemOneid(claw);
                            if (!item) return (
                              <div className="flex items-center flex-wrap gap-0.5">
                                <span className="text-sm text-[var(--text-weak)]">—</span>
                                {dots}
                              </div>
                            );
                            const segments = item.path.split(" / ");
                            const shortName = segments[segments.length - 1] || item.path;
                            return (
                              <div className="flex items-center flex-wrap gap-0.5">
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <span className="text-sm text-[var(--text-secondary)] truncate max-w-[120px] block cursor-default">{shortName}</span>
                                  </TooltipTrigger>
                                  <TooltipContent side="bottom" align="start">
                                    <span className="text-xs">{item.path}</span>
                                  </TooltipContent>
                                </Tooltip>
                                {dots}
                              </div>
                            );
                          } else {
                            const item = getClawGroupItemManual(claw);
                            if (!item) return (
                              <div className="flex items-center flex-wrap gap-0.5">
                                <span className="text-sm text-[var(--text-weak)]">—</span>
                                {dots}
                              </div>
                            );
                            return (
                              <div className="flex items-center flex-wrap gap-0.5">
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <span className="text-sm text-[var(--text-secondary)] truncate max-w-[120px] block cursor-default">{item.path}</span>
                                  </TooltipTrigger>
                                  <TooltipContent side="bottom" align="start">
                                    <span className="text-xs">{item.path}</span>
                                  </TooltipContent>
                                </Tooltip>
                                {dots}
                              </div>
                            );
                          }
                        })()}
                      </TableCell>
                      {/* 项目：一个 Agent 可挂 0~多个项目（mock 确定性映射）。展示口径与「部门」一致：纯文字 + N，hover 按序展开全部 */}
                      <TableCell className="px-4 py-4">
                        {(() => {
                          const projs = getClawProjects(claw.id);
                          if (projs.length === 0) return <span className="text-sm text-[var(--text-weak)]">—</span>;
                          if (projs.length === 1) {
                            return (
                              <span className="text-sm text-[var(--text-secondary)] truncate block max-w-[200px]" title={projs[0].name}>
                                {projs[0].name}
                              </span>
                            );
                          }
                          return (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="inline-flex items-center gap-1 max-w-[200px] cursor-default">
                                  <span className="text-sm text-[var(--text-secondary)] truncate">{projs[0].name}</span>
                                  <span className="text-xs text-[var(--text-muted)] tabular-nums shrink-0">+{projs.length - 1}</span>
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="bottom" align="start" className="max-w-[360px] p-0">
                                <div className="py-2">
                                  {projs.map((p, idx) => (
                                    <div key={idx} className="px-3 py-1.5 text-sm">
                                      <span className="text-[var(--text-weak)] mr-1">{idx + 1}.</span>
                                      <span className="text-white">{p.name}</span>
                                    </div>
                                  ))}
                                </div>
                              </TooltipContent>
                            </Tooltip>
                          );
                        })()}
                      </TableCell>
                      {/* 共享范围 - 与用户端 agent 共享范围联动 */}
                      <TableCell className="px-4 py-4">
                        {(() => {
                          const targets = getClawShareTargets(claw.id);
                          // 未被共享（仅自己 / 无共享范围）→ 横线，与组织、部门两列保持一致
                          if (targets.length === 0) return <span className="text-sm text-[var(--text-weak)]">—</span>;
                          // 共享给单个对象 → 直接展示
                          if (targets.length === 1) {
                            return (
                              <span className="text-sm text-[var(--text-secondary)] truncate block max-w-[200px]" title={targets[0]}>
                                {targets[0]}
                              </span>
                            );
                          }
                          // 共享给多个对象 → 首项 + “+N”，hover 展开全部
                          return (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="inline-flex items-center gap-1 max-w-[200px] cursor-default">
                                  <span className="text-sm text-[var(--text-secondary)] truncate">{targets[0]}</span>
                                  <span className="text-xs text-[var(--text-muted)] tabular-nums shrink-0">+{targets.length - 1}</span>
                                </span>
                              </TooltipTrigger>
                              <TooltipContent side="bottom" align="start" className="max-w-[240px]">
                                <span className="text-xs">{targets.join("、")}</span>
                              </TooltipContent>
                            </Tooltip>
                          );
                        })()}
                      </TableCell>
                      {/* 创建时间 */}
                      <TableCell className="px-4 py-4 whitespace-nowrap">{claw.createTime}</TableCell>
                      {/* 智能体 */}
                      <TableCell className="px-4 py-4">
                        {getClawTypeLabel(claw)}
                      </TableCell>
                      {/* Agent 版本 */}
                      <TableCell className="px-4 py-4 text-sm text-[var(--text-title)] tabular-nums whitespace-nowrap">
                        {claw.version}
                      </TableCell>
                      {/* 标签（当前页无任何带标签的实例时整列隐藏） */}
                      {hasAnyTagColumn && (
                        <TableCell className="px-4 py-4">
                          {claw.tags && claw.tags.length > 0 ? (
                            <HoverCard openDelay={100} closeDelay={150}>
                              <HoverCardTrigger asChild>
                                <button className="inline-flex items-center text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors">
                                  <Tag className="w-4 h-4" />
                                </button>
                              </HoverCardTrigger>
                              <HoverCardContent side="top" align="center" className="p-0 w-56 overflow-hidden border border-[var(--border)] bg-[var(--popover)]">
                                <div className="grid grid-cols-2 bg-[var(--accent)] border-b border-[var(--border)] px-3 py-2">
                                  <span className="text-xs font-semibold text-[var(--text-secondary)]">标签键</span>
                                  <span className="text-xs font-semibold text-[var(--text-secondary)]">标签值</span>
                                </div>
                                <div className="divide-y divide-[var(--border)]">
                                  {claw.tags.map((tag, i) => (
                                    <div key={i} className="grid grid-cols-2 px-3 py-2 gap-1">
                                      <Tooltip>
                                        <TooltipTrigger asChild>
                                          <span className="text-xs text-[var(--text-secondary)] truncate block max-w-full cursor-default">{tag.key}</span>
                                        </TooltipTrigger>
                                        <TooltipContent side="left"><span>{tag.key}</span></TooltipContent>
                                      </Tooltip>
                                      <Tooltip>
                                        <TooltipTrigger asChild>
                                          <span className="text-xs text-[var(--text-muted)] truncate block max-w-full cursor-default">{tag.value}</span>
                                        </TooltipTrigger>
                                        <TooltipContent side="right"><span>{tag.value}</span></TooltipContent>
                                      </Tooltip>
                                    </div>
                                  ))}
                                </div>
                              </HoverCardContent>
                            </HoverCard>
                          ) : (
                            <Tag className="w-4 h-4 text-[var(--text-weak)] opacity-40" />
                          )}
                        </TableCell>
                      )}
                      {/* 操作 - 全局 TableActionCell 内部按钮强制 link 蓝色样式（详见 SKILL §15 操作列规则） */}
                      <TableActionCell fixed="right" style={{ minWidth: '240px' }} actionsClassName="h-5">
                        {opLocked ? (
                          <>
                            <LockedOpLabel label="终端" tip={lockTip} />
                            <LockedOpLabel label={claw.status === "running" ? "关机" : "开机"} tip={lockTip} />
                            <LockedOpLabel label="删除" tip={lockTip} />
                            <LockedOpLabel label="更多" tip={lockTip} />
                          </>
                        ) : isLocalAgent ? (
                          <>
                            {isLocalConnected(claw) ? (
                              <Button variant="link" onClick={() => handleManageLocalSkills(claw)}>
                                Skill 管理
                              </Button>
                            ) : (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <span className="text-[14px] text-[var(--text-brand)] opacity-40 cursor-not-allowed whitespace-nowrap">Skill 管理</span>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="text-xs">
                                  未接入本地客户端暂不可管理 Skill
                                </TooltipContent>
                              </Tooltip>
                            )}
                            <Button variant="link" onClick={() => handleDeleteClick(claw)}>
                              移除
                            </Button>
                          </>
                        ) : (
                          <>
                          {/* 终端 */}
                          {!isRunning ? (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="text-[14px] text-[var(--text-brand)] opacity-40 cursor-not-allowed whitespace-nowrap">终端</span>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="text-xs">
                                仅运行中的实例可进入终端
                              </TooltipContent>
                            </Tooltip>
                          ) : (
                            <Button variant="link" onClick={() => handleOpenTerminal(claw)}>
                              终端
                            </Button>
                          )}

                          {/* 关机/开机 */}
                          {claw.status === "running" ? (
                            <Button variant="link" onClick={() => setShutdownTarget(claw.id)}>
                              关机
                            </Button>
                          ) : claw.status === "shutdown" ? (
                            <Button variant="link" onClick={() => setShutdownTarget(claw.id)}>
                              开机
                            </Button>
                          ) : (
                            <span className="text-[14px] text-[var(--text-brand)] opacity-40 whitespace-nowrap">开机</span>
                          )}

                          {/* 删除 */}
                          {["creating", "loading", "pending"].includes(claw.status) ? (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="text-[14px] text-[var(--text-brand)] opacity-40 cursor-not-allowed whitespace-nowrap">删除</span>
                              </TooltipTrigger>
                              <TooltipContent side="top" className="text-xs">
                                当前状态不可删除
                              </TooltipContent>
                            </Tooltip>
                          ) : (
                            <Button variant="link" onClick={() => handleDeleteClick(claw)}>
                              删除
                            </Button>
                          )}

                          {/* 更多操作（外包定位容器：承载更新提示气泡的绝对定位） */}
                          <span className="relative inline-flex">
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="link">
                                更多
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end" className="w-44">
                              <DropdownMenuItem
                                className={`cursor-pointer focus:bg-[var(--accent)] ${isRunning ? "text-[var(--text-title)] focus:text-[var(--text-title)] [&_svg:not([class*='text-'])]:text-[var(--text-title)]" : "text-[var(--text-weak)] opacity-40 cursor-not-allowed [&_svg:not([class*='text-'])]:text-[var(--text-weak)]"}`}
                                disabled={!isRunning}
                                onClick={() => handleRestart(claw)}
                              >
                                <RotateCcw className="w-3.5 h-3.5" />
                                <BodyText as="span" tone="inherit">重启 Agent</BodyText>
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                className={`cursor-pointer focus:bg-[var(--accent)] ${["running", "shutdown"].includes(claw.status) ? "text-[var(--text-title)] focus:text-[var(--text-title)] [&_svg:not([class*='text-'])]:text-[var(--text-title)]" : "text-[var(--text-weak)] opacity-40 cursor-not-allowed [&_svg:not([class*='text-'])]:text-[var(--text-weak)]"}`}
                                disabled={!["running", "shutdown"].includes(claw.status)}
                                onClick={() => handleReinstallClick(claw)}
                              >
                                <HardDriveDownload className="w-3.5 h-3.5" />
                                <BodyText as="span" tone="inherit">重新安装 Agent</BodyText>
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                className="cursor-pointer text-[var(--text-title)] focus:bg-[var(--accent)] focus:text-[var(--text-title)] [&_svg:not([class*='text-'])]:text-[var(--text-title)]"
                                onClick={() => handleOpenMonitor(claw)}
                              >
                                <Activity className="w-3.5 h-3.5" />
                                <BodyText as="span" tone="inherit">监控</BodyText>
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                className="cursor-pointer text-[var(--text-title)] focus:bg-[var(--accent)] focus:text-[var(--text-title)] [&_svg:not([class*='text-'])]:text-[var(--text-title)]"
                                onClick={() => setAdjustConfigContext({ mode: "single", claw })}
                              >
                                <SlidersHorizontal className="w-3.5 h-3.5" />
                                <BodyText as="span" tone="inherit">调整配置</BodyText>
                              </DropdownMenuItem>
                              {/* 克隆 Agent 为镜像：外部接入的 Agent（无 CVM 实例）不可用，悬浮提示原因 */}
                              <DropdownMenuItem
                                className={`focus:bg-[var(--accent)] ${claw.instanceId ? "cursor-pointer text-[var(--text-title)] focus:text-[var(--text-title)] [&_svg:not([class*='text-'])]:text-[var(--text-title)]" : "text-[var(--text-weak)] opacity-40 cursor-not-allowed [&_svg:not([class*='text-'])]:text-[var(--text-weak)]"}`}
                                onClick={() => { if (claw.instanceId) setCloneImageTarget(claw); }}
                              >
                                {claw.instanceId ? (
                                  <>
                                    <Copy className="w-3.5 h-3.5" />
                                    <BodyText as="span" tone="inherit">克隆为自定义镜像</BodyText>
                                  </>
                                ) : (
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <span className="flex items-center gap-2 w-full">
                                        <Copy className="w-3.5 h-3.5" />
                                        <BodyText as="span" tone="inherit">克隆为自定义镜像</BodyText>
                                      </span>
                                    </TooltipTrigger>
                                    <TooltipContent side="top" className="text-xs">外部接入的 Agent 不支持该功能</TooltipContent>
                                  </Tooltip>
                                )}
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                          {/* 更新提示气泡：克隆 Agent 为镜像（plan-20260818-clone-image v1）
                              仅首个云端 Agent 行展示，指向「更多」入口；关闭或曝光 2 次后不再展示，2026-09-01 到期 */}
                          {paginated.find((c) => c.agentType !== "LocalAgent")?.id === claw.id && (
                            <GuidePointBubble
                              open={cloneBubbleOpen}
                              onClose={closeCloneBubble}
                              title="克隆 Agent 为镜像"
                              description="在「更多」中将编辑好的 Agent 克隆为自定义镜像，沉淀企业模板，便于批量复制与分发。"
                              contentVariant="text-only"
                              placement="left"
                              endpoint="admin"
                              style={{ position: "absolute", right: "calc(100% + 4px)", top: "50%", transform: "translateY(-50%)" }}
                            />
                          )}
                          </span>
                          </>
                        )}
                      </TableActionCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>

          {/*
            Pagination（v0.9 迁移到新版声明式 API：total/current/pageSize/onChange）

            停服态豁免（页面级，非组件库改动）：
            本页使用的是 shadcn 版<Pagination>（@/components/ui/pagination），
            按"不动组件库维度代码"约束，改动落在页面上：为其外层包裹 <div>
            加 data-billing-exempt，触达 AdminDisabledOverlay 的两条恢复分支：
              1) 视觉：.admin-service-suspended [data-billing-exempt] *恢复
                 opacity/cursor/pointer-events 到正常态；
              2) 事件：文档级 click/mousedown 拦截通过
                 target.closest('[data-billing-exempt]') 命中后放行。
            "停服前已禁用则延续禁用"：<Pagination> 内部对首/末页按钮自身
            标注的 disabled/aria-disabled 依旧生效（CSS 恢复分支包含
            :not([disabled]):not([aria-disabled="true"]) 保护），因此
            "首页时上一页 / 末页时下一页" 依然禁用。
          */}
          <div
            className="px-4 py-2 border-t border-[var(--border)]"
            data-billing-exempt
          >
            <Pagination
              total={versionFiltered.length}
              current={safePage}
              pageSize={PAGE_SIZE}
              showTotal={(total) => `共 ${total} 条记录`}
              className="w-full justify-between"
              hideOnSinglePage
              onChange={(p) => setPage(p)}
            />
          </div>
        </SurfaceCard>

      </div>

      {/* 克隆 Agent 为镜像弹窗 */}
      <CloneAgentImageDialog
        open={!!cloneImageTarget}
        onOpenChange={(o) => { if (!o) setCloneImageTarget(null); }}
        claw={cloneImageTarget}
      />

      {/* 关机/开机确认弹窗 */}
      {(() => {
        const target = claws.find(c => c.id === shutdownTarget);
        const isRunning = target?.status === "running";

        // 关机 → 警示弹窗（AlertDialog）
        if (isRunning) {
          return (
            <AlertDialog open={!!shutdownTarget} onOpenChange={() => setShutdownTarget(null)}>
              <AlertDialogContent className="sm:max-w-[420px]">
                <AlertDialogHeader>
                  <AlertDialogTitle className="text-[var(--text-title)]">确认关机</AlertDialogTitle>
                </AlertDialogHeader>
                <AlertDialogDescription asChild>
                  <p className="text-sm text-[var(--text-secondary)]">
                    关机后该 Agent「{target?.name}」
                    <span className="text-[var(--text-danger)] font-medium">将无法使用，直到重新开机</span>
                    。确认关机吗？
                  </p>
                </AlertDialogDescription>
                <AlertDialogFooter>
                  <Button variant="claw-outline" onClick={() => setShutdownTarget(null)}>取消</Button>
                  <AlertDialogAction variant="dialog-confirm" onClick={confirmShutdown}>确认关机</AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          );
        }

        // 开机 → 普通弹窗
        return (
          <Dialog open={!!shutdownTarget} onOpenChange={() => setShutdownTarget(null)}>
            <DialogContent className="sm:max-w-[360px]">
              <DialogHeader>
                <DialogTitle className="text-base font-bold text-[var(--text-title)]">
                  确认开机
                </DialogTitle>
              </DialogHeader>
              <p className="text-sm text-[var(--text-muted)] leading-relaxed">
                开机后该 Agent「{target?.name}」将重新运行。确认开机吗？
              </p>
              <DialogFooter className="gap-2 pt-2">
                <Button variant="claw-outline" onClick={() => setShutdownTarget(null)}>取消</Button>
                <Button variant="dialog-confirm" onClick={confirmPowerOn}>确认开机</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        );
      })()}

      {/* 重新安装确认弹窗（警示弹窗） */}
      <AlertDialog open={!!reinstallTarget} onOpenChange={() => setReinstallTarget(null)}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[var(--text-title)]">确认重新安装</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <p className="text-sm text-[var(--text-secondary)]">
              将使用最新镜像重新安装「{claws.find(c => c.id === reinstallTarget)?.name}」，清除当前所有配置且无法恢复，
              <span className="text-[var(--text-danger)] font-medium">
                安装完成后需重新配置模型和通道。
              </span>
            </p>
          </AlertDialogDescription>
          <div>
            <BodyText as="label" className="mb-2 block">请输入「重装」以确认</BodyText>
            <Input
              value={reinstallInput}
              onChange={(e) => setReinstallInput(e.target.value)}
              placeholder="输入「重装」"
            />
          </div>
          <AlertDialogFooter>
            <Button variant="claw-outline" onClick={() => setReinstallTarget(null)}>取消</Button>
            <AlertDialogAction variant="dialog-confirm" onClick={confirmReinstall} disabled={reinstallInput !== "重装"}>确认重新安装</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 重启 Agent 确认弹窗 */}
      <AlertDialog open={!!restartConfirm} onOpenChange={(open) => { if (!open) { setRestartConfirm(null); setRestartFullServer(false); } }}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[var(--text-title)]">重启 Agent</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <p className="text-sm text-[var(--text-secondary)]">
              将会重启 Agent「{restartConfirm?.name}」服务，重启期间该 Agent 将短暂不可用。
            </p>
          </AlertDialogDescription>
          <div className="flex items-center gap-2 pt-1">
            <Checkbox
              id="admin-restart-full-server"
              checked={restartFullServer}
              onCheckedChange={(checked) => setRestartFullServer(checked === true)}
            />
            <label htmlFor="admin-restart-full-server" className="text-sm font-normal cursor-pointer text-[var(--text-secondary)]">
              重启云服务器
            </label>
            <Tooltip>
              <TooltipTrigger asChild>
                <HelpCircle className="w-4 h-4 text-[var(--text-muted)] cursor-help shrink-0" />
              </TooltipTrigger>
              <TooltipContent side="bottom" className="text-xs max-w-[240px]">
                勾选后，将重启整台云服务器，建议仅在重启 Agent 后服务仍异常时使用
              </TooltipContent>
            </Tooltip>
          </div>
          <AlertDialogFooter>
            <Button variant="claw-outline" onClick={() => { setRestartConfirm(null); setRestartFullServer(false); }}>取消</Button>
            <AlertDialogAction variant="dialog-confirm" onClick={confirmRestart}>确认重启</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 删除确认弹窗（警示弹窗） */}
      {(() => {
        const deleteTargetClaw = claws.find(c => c.id === deleteTarget);
        const isLocalAgent = deleteTargetClaw?.agentType === "LocalAgent";
        const isCreateFail = deleteTargetClaw?.status === "createFail";
        const isRunning = !isLocalAgent && deleteTargetClaw?.status === "running";
        return (
          <AlertDialog open={!!deleteTarget} onOpenChange={() => setDeleteTarget(null)}>
            <AlertDialogContent className="sm:max-w-[420px]">
              <AlertDialogHeader>
                <AlertDialogTitle className="text-[var(--text-title)]">{isLocalAgent ? "确认移除" : "确认删除"}</AlertDialogTitle>
              </AlertDialogHeader>
              <AlertDialogDescription asChild>
                {isLocalAgent ? (
                  <p className="text-sm text-[var(--text-secondary)]">
                    移除后「{deleteTargetClaw?.name}」将不再出现在管控端 Agent 列表中，不会关闭或卸载用户本地的 {deleteTargetClaw ? getClawTypeLabel(deleteTargetClaw) : "本地 Agent"}。
                  </p>
                ) : isCreateFail ? (
                  <p className="text-sm text-[var(--text-secondary)]">
                    此操作将移除「{deleteTargetClaw?.name}」该创建失败的记录，底层资源将由系统自动回收。
                  </p>
                ) : (
                  <p className="text-sm text-[var(--text-secondary)]">
                    此操作不可撤销。「{deleteTargetClaw?.name}」
                    <span className="text-[var(--text-danger)] font-medium">
                      实例及相关数据将被永久删除，已配置的模型、通道和插件将全部清除且无法恢复。
                    </span>
                  </p>
                )}
              </AlertDialogDescription>
              {isRunning && (
                <div>
                  <label className="block text-[14px] font-medium text-[var(--text-title)] mb-2">请输入「删除」以确认</label>
                  <Input
                    value={deleteInput}
                    onChange={(e) => setDeleteInput(e.target.value)}
                    placeholder="输入「删除」"
                  />
                </div>
              )}
              <AlertDialogFooter>
                <Button variant="claw-outline" onClick={() => setDeleteTarget(null)}>取消</Button>
                <AlertDialogAction variant="destructive" onClick={confirmDelete} disabled={isRunning && deleteInput !== "删除"}>
                  {isLocalAgent ? "确认移除" : "确认删除"}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        );
      })()}

      {/* 批量删除确认弹窗（警示弹窗） */}
      <AlertDialog
        open={showBatchDeleteDialog}
        onOpenChange={(open) => {
          if (!open) {
            setShowBatchDeleteDialog(false);
            setBatchDeleteInput("");
          }
        }}
      >
        <AlertDialogContent className="sm:max-w-[560px]">
          <AlertDialogHeader>
            <AlertDialogTitle asChild>
              <PanelTitle>批量删除</PanelTitle>
            </AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <BodyText tone="secondary">此操作不可撤销，请再次确认删除范围与影响。</BodyText>
          </AlertDialogDescription>
          <div className="space-y-5 py-2">
            <Alert variant="warning" className="items-start [&>svg]:translate-y-[1px]">
              <CircleAlert />
              <AlertDescription>
                <p>将永久删除 <MetaMedium as="span" tone="inherit" className="tabular-nums">{selectedCount}</MetaMedium> 个实例，以及相关模型、通道和插件配置，且无法恢复。</p>
              </AlertDescription>
            </Alert>
            {selectedCount > 0 && (
              <div className="space-y-2">
                <BodyText as="div">
                  待删除实例（<span className="tabular-nums">{selectedCount}</span> 个）
                </BodyText>
                <div className="overflow-hidden rounded-[4px] border border-[var(--border)] bg-[var(--popover)]">
                  <div className="max-h-[260px] overflow-y-auto scrollbar-on-hover">
                    <Table density="compact">
                      <TableHeader>
                        <TableRow>
                          <TableHead className="w-[60%]">名称 / ID</TableHead>
                          <TableHead className="w-[40%]">Agent 类型</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {selectedClaws.map((c) => (
                          <TableRow key={c.id}>
                            <TableCell className="whitespace-normal">
                              <div className="min-w-0 space-y-0.5 py-0.5">
                                <MiniBodyText as="div" className="break-words">{c.name}</MiniBodyText>
                                <CodeText tone="weak" className="break-all">{c.instanceId}</CodeText>
                              </div>
                            </TableCell>
                            <TableCell className="whitespace-normal">
                              <MetaText as="span" tone="secondary" className="break-words">{AGENT_TYPE_DISPLAY[c.agentType] ?? c.agentType}</MetaText>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </div>
              </div>
            )}
            <div className="space-y-2">
              <BodyText as="label" htmlFor="batch-delete-confirm-input" className="block">
                请输入「删除」以确认
              </BodyText>
              <Input
                id="batch-delete-confirm-input"
                value={batchDeleteInput}
                onChange={(e) => setBatchDeleteInput(e.target.value)}
                placeholder="输入「删除」"
                autoFocus
              />
              <MetaText as="p">输入正确后才可执行删除。</MetaText>
            </div>
          </div>
          <AlertDialogFooter>
            <Button variant="claw-outline" onClick={() => { setShowBatchDeleteDialog(false); setBatchDeleteInput(""); }}>取消</Button>
            <AlertDialogAction variant="destructive" disabled={batchDeleteInput !== "删除" || selectedCount === 0} onClick={() => {
              setClaws((prev) => prev.filter((c) => !selectedIds.has(c.id)));
              const removed = selectedCount;
              setSelectedIds(new Set());
              setShowBatchDeleteDialog(false);
              setBatchDeleteInput("");
              toast.success(`已删除 ${removed} 个实例`);
            }}>确认删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 批量关机确认弹窗 */}
      <Dialog open={showBatchShutdownDialog} onOpenChange={(o) => { if (!o) setShowBatchShutdownDialog(false); }}>
        <DialogContent className="sm:max-w-2xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>批量关机</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-[var(--text-secondary)] leading-relaxed">
            您已选择 <span className="font-medium text-[var(--text-title)]">{runningSelectedClaws.length}</span> 个运行中的 Agent，进行关机操作前，请确认：
          </p>
          {runningSelectedClaws.length > 0 && (
            <div className="rounded-[4px] bg-[var(--accent)] border border-[var(--border)] px-3 py-2 max-h-24 overflow-y-auto">
              <p className="text-xs text-[var(--text-muted)] leading-relaxed break-all">
                {runningSelectedClaws.map(c => c.name).join("、")}
              </p>
            </div>
          )}
          <DialogFooter className="gap-2 pt-2">
            <Button variant="claw-outline" size="claw-sm" onClick={() => setShowBatchShutdownDialog(false)}>取消</Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={() => {
                const ids = new Set(runningSelectedClaws.map(c => c.id));
                if (ids.size === 0) { setShowBatchShutdownDialog(false); return; }
                setClaws(prev => prev.map(c => ids.has(c.id) ? { ...c, status: "shutdown" as ClawStatus } : c));
                setShowBatchShutdownDialog(false);
                setSelectedIds(new Set());
                toast.success(`已关机 ${ids.size} 个 Agent`);
              }}
            >
              确认关机
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 批量开机确认弹窗 */}
      <Dialog open={showBatchPowerOnDialog} onOpenChange={(o) => { if (!o) setShowBatchPowerOnDialog(false); }}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>批量开机</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-[var(--text-secondary)] leading-relaxed">
            将对所选 <span className="font-medium text-[var(--text-title)]">{shutdownSelectedClaws.length}</span> 个已关机的 Agent 执行开机操作，开机后将重新运行。
          </p>
          {shutdownSelectedClaws.length > 0 && (
            <div className="rounded-[4px] bg-[var(--accent)] border border-[var(--border)] px-3 py-2 max-h-24 overflow-y-auto">
              <p className="text-xs text-[var(--text-muted)] leading-relaxed break-all">
                {shutdownSelectedClaws.map(c => c.name).join("、")}
              </p>
            </div>
          )}
          <DialogFooter className="gap-2 pt-2">
            <Button variant="claw-outline" size="claw-sm" onClick={() => setShowBatchPowerOnDialog(false)}>取消</Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={() => {
                const ids = new Set(shutdownSelectedClaws.map(c => c.id));
                if (ids.size === 0) { setShowBatchPowerOnDialog(false); return; }
                setClaws(prev => prev.map(c => ids.has(c.id) ? { ...c, status: "running" as ClawStatus } : c));
                setShowBatchPowerOnDialog(false);
                setSelectedIds(new Set());
                toast.success(`已开机 ${ids.size} 个 Agent`);
              }}
            >
              确认开机
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 批量配置模型弹窗 */}
      <Dialog open={showBatchConfigModelDialog} onOpenChange={(o) => { if (!o) setShowBatchConfigModelDialog(false); }}>
        <DialogContent className="rounded-[4px] sm:max-w-[640px] max-h-[85vh] min-w-0 flex flex-col">
          <DialogHeader className="pb-2">
            <DialogTitle className="text-[16px] font-semibold text-[var(--text-title)]">批量配置模型</DialogTitle>
            <DialogDescription>
              将对所选 <span className="font-din font-bold tabular-nums text-[var(--text-brand)]">{runningConfigurableClaws.length}</span> 个运行中的 Agent 统一配置相同的目标模型，这些 Agent 的当前模型将被覆盖。
            </DialogDescription>
          </DialogHeader>

          <DialogBody className="px-6">
          <div className="space-y-4">
            {/* 所选 Agent 当前模型现状 */}
            {runningConfigurableClaws.length > 0 && (
              <div>
                <div className="text-sm font-medium text-[var(--text-title)] mb-2">当前模型</div>
                <div className="rounded-[4px] border border-[var(--border)] divide-y divide-[var(--border)] max-h-48 overflow-y-auto">
                  {runningConfigurableClaws.map((c) => {
                    const current = getClawCurrentModels(c);
                    return (
                      <div key={c.id} className="grid grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)] gap-3 px-3 py-2 min-w-0">
                        <div className="min-w-0">
                          <p className="text-sm text-[var(--text-strong)] truncate" title={c.name}>{c.name}</p>
                          <p className="text-xs text-[var(--text-weak)] truncate">{c.instanceId}</p>
                        </div>
                        <div className="min-w-0 flex flex-col gap-1">
                          {current.length === 0 ? (
                            <span className="text-xs text-[var(--text-weak)]">未配置模型</span>
                          ) : (
                            current.map((m, i) => (
                              <div key={i} className="flex items-center gap-1.5 min-w-0">
                                <span className="shrink-0 w-9 text-xs text-right">
                                  <StatusTag mode="text" variant={m.primary ? "green" : "gray"}>
                                    {m.primary ? "主" : "备选"}
                                  </StatusTag>
                                </span>
                                <span className="text-xs text-[var(--text-muted)] truncate min-w-0 flex-1" title={`${m.providerLabel} · ${m.versionLabel}`}>
                                  {m.providerLabel}
                                  {m.versionLabel ? ` · ${m.versionLabel}` : ""}
                                </span>
                              </div>
                            ))
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}

            {/* 选择并添加目标模型 */}
            <div>
              <div className="text-sm font-medium text-[var(--text-title)] mb-2">目标模型</div>
              <div className="flex items-end gap-2">
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-[var(--text-muted)] mb-1">厂商</p>
                  <Select value={batchModelProvider} onValueChange={handleBatchModelProviderChange}>
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="选择厂商" />
                    </SelectTrigger>
                    <SelectContent>
                      {MODEL_PROVIDERS.map(p => (
                        <SelectItem key={p.value} value={p.value}>{p.label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-[var(--text-muted)] mb-1">版本</p>
                  <Select value={batchModelVersion} onValueChange={setBatchModelVersion}>
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="选择版本" />
                    </SelectTrigger>
                    <SelectContent>
                      {(MODEL_PROVIDERS.find(p => p.value === batchModelProvider)?.versions ?? []).map(v => (
                        <SelectItem key={v.value} value={v.value}>{v.label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <Button
                  variant="claw-outline"
                  size="claw"
                  className="px-3 gap-1.5 shrink-0 whitespace-nowrap"
                  onClick={handleAddBatchModel}
                  disabled={false}
                >
                  <Plus className="w-3.5 h-3.5" />
                  {batchSingleModelMode
                    ? (batchAppliedModels.length > 0 ? "替换模型" : "添加模型")
                    : (batchAppliedModels.some(m => m.primary) ? "添加备选模型" : "添加主模型")}
                </Button>
              </div>
              {batchAppliedModels.length === 0 && (
                <p className="text-xs text-[var(--text-weak)] mt-1.5">点击右侧按钮加入列表，"确认配置"后才应用到 Agent</p>
              )}
            </div>

            {/* 已添加的目标模型列表（openclaw：主/备选；非 openclaw：单模型区） */}
            {batchAppliedModels.length === 0 ? (
              <div className="rounded-[4px] border border-dashed border-[var(--border)] py-6 text-center text-xs text-[var(--text-weak)]">
                暂未添加目标模型，请先选择厂商与版本并添加
              </div>
            ) : batchSingleModelMode ? (
              // 非 openclaw 模式：单一「目标模型」区，无主备标签、无主备操作
              <div className="space-y-1.5">
                <p className="text-xs text-[var(--text-muted)]">目标模型</p>
                {batchAppliedModels.map(model => (
                  <div key={model.id} className="rounded-[4px] border bg-[var(--card)] hover:bg-[var(--accent)] border-[var(--border)] p-2.5 min-w-0 overflow-hidden">
                    <div className="flex items-center justify-between">
                      <div className="flex flex-col min-w-0 overflow-hidden">
                        <span className="text-sm font-medium text-[var(--text-strong)] leading-tight truncate block">{model.providerLabel}</span>
                        {model.versionLabel && (
                          <span className="text-xs text-[var(--text-muted)] leading-tight mt-0.5 truncate block">{model.versionLabel}</span>
                        )}
                      </div>
                      <div className="flex items-center gap-2 shrink-0 ml-2">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button
                              type="button"
                              onClick={() => handleRemoveBatchModel(model.id)}
                              className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="bg-[var(--text-emphasis)] text-white text-xs">删除模型</TooltipContent>
                        </Tooltip>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              // openclaw 模式：主模型 / 备选模型分区
              <div className="space-y-3">
                {/* 主模型 */}
                {batchAppliedModels.some(m => m.primary) && (
                  <div>
                    <p className="text-xs text-[var(--text-muted)] mb-1.5">主模型</p>
                    <div className="space-y-1.5">
                      {batchAppliedModels.filter(m => m.primary).map(model => (
                        <div key={model.id} className="rounded-[4px] border bg-[var(--card)] hover:bg-[var(--accent)] border-[var(--border)] p-2.5 min-w-0 overflow-hidden">
                          <div className="flex items-center justify-between">
                            <div className="flex flex-col min-w-0 overflow-hidden">
                              <span className="text-sm font-medium text-[var(--text-strong)] leading-tight truncate block">{model.providerLabel}</span>
                              {model.versionLabel && (
                                <span className="text-xs text-[var(--text-muted)] leading-tight mt-0.5 truncate block">{model.versionLabel}</span>
                              )}
                            </div>
                            <div className="flex items-center gap-2 shrink-0 ml-2">
                              <StatusTag mode="fill" variant="green">主模型</StatusTag>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <button
                                    type="button"
                                    onClick={() => handleRemoveBatchModel(model.id)}
                                    className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                                  >
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="bg-[var(--text-emphasis)] text-white text-xs">删除模型</TooltipContent>
                              </Tooltip>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
                {/* 备选模型 */}
                {batchAppliedModels.some(m => !m.primary) && (
                  <div>
                    <p className="text-xs text-[var(--text-muted)] mb-1">备选模型（{batchAppliedModels.filter(m => !m.primary).length}）</p>
                    <Alert variant="info" className="mb-2 text-xs">
                      <Info />
                      <AlertDescription>主模型不可用时会自动切换备选模型，此时备选模型消耗的 token 将统计到主模型下。</AlertDescription>
                    </Alert>
                    <div className="space-y-1.5">
                      {batchAppliedModels.filter(m => !m.primary).map(model => (
                        <div key={model.id} className="rounded-[4px] border bg-[var(--card)] border-[var(--border)] hover:bg-[var(--accent)] p-2.5 min-w-0 overflow-hidden">
                          <div className="flex items-center justify-between">
                            <div className="flex flex-col min-w-0 overflow-hidden">
                              <span className="text-sm font-medium text-[var(--text-strong)] leading-tight truncate block">{model.providerLabel}</span>
                              {model.versionLabel && (
                                <span className="text-xs text-[var(--text-muted)] leading-tight mt-0.5 truncate block">{model.versionLabel}</span>
                              )}
                            </div>
                            <div className="flex items-center gap-2 shrink-0 ml-2">
                              <StatusTag mode="fill" variant="gray">备选</StatusTag>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <button
                                    type="button"
                                    onClick={() => handleSetBatchPrimary(model.id)}
                                    className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors"
                                    aria-label="设为主模型"
                                  >
                                    <ArrowLeftRight className="w-3.5 h-3.5" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="bg-[var(--text-emphasis)] text-white text-xs">设为主模型</TooltipContent>
                              </Tooltip>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <button
                                    type="button"
                                    onClick={() => handleRemoveBatchModel(model.id)}
                                    className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                                  >
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="bg-[var(--text-emphasis)] text-white text-xs">删除模型</TooltipContent>
                              </Tooltip>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
          </DialogBody>

          <DialogFooter className="gap-2 pt-2">
            <Button variant="claw-outline" size="claw-sm" onClick={() => setShowBatchConfigModelDialog(false)}>取消</Button>
            <Tooltip delayDuration={200}>
              <TooltipTrigger asChild>
                <span className="inline-flex">
                  <Button
                    variant="dialog-confirm"
                    size="claw-sm"
                    disabled={batchAppliedModels.length === 0}
                    onClick={confirmBatchConfigModel}
                  >
                    确认配置
                  </Button>
                </span>
              </TooltipTrigger>
              {batchAppliedModels.length === 0 && (
                <TooltipContent side="top" className="text-xs">
                  请先添加目标模型
                </TooltipContent>
              )}
            </Tooltip>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 调整配置弹窗：实例规格升配 / 系统盘容量扩容，前端 mock 校验 + 影响确认页；均为前端 UI，不提交/不改列表数据 */}
      <Dialog
        open={!!adjustConfigContext}
        onOpenChange={(o) => { if (!o) setAdjustConfigContext(null); }}
      >
        <DialogContent
          size="lg"
          className="rounded-[var(--radius-lg)] max-h-[85vh] flex flex-col"
          onInteractOutside={(e) => e.preventDefault()}
          onEscapeKeyDown={(e) => e.preventDefault()}
        >
          <DialogHeader className="pb-4">
            <DialogTitle className="text-base font-semibold text-[var(--text-title)]">调整配置</DialogTitle>
          </DialogHeader>

          {adjustConfigStep === "config" ? (
            <DialogBody className="px-6">
              {/* 固定区：调整类型 + 目标配置（不随表格滚动） */}
              <div className="shrink-0 space-y-5">
              {/* 调整类型 */}
              <div>
                <div className="text-sm font-medium text-[var(--cp-text-title)] mb-2">调整类型</div>
                <RadioGroup
                  value={adjustConfigType}
                  onValueChange={(v) => handleAdjustConfigTypeChange(v as "spec" | "capacity")}
                  className="flex items-center gap-6"
                >
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <div className="flex items-center gap-2">
                        <RadioGroupItem value="spec" id="adjust-type-spec" disabled={singleInstanceAtMaxSpec} />
                        <Label htmlFor="adjust-type-spec" className={cn("text-sm", singleInstanceAtMaxSpec ? "cursor-not-allowed opacity-50" : "cursor-pointer")}>实例规格升配</Label>
                      </div>
                    </TooltipTrigger>
                    {singleInstanceAtMaxSpec && (
                      <TooltipContent side="top" className="text-xs">当前实例规格已是最高配置，无法继续升配</TooltipContent>
                    )}
                  </Tooltip>
                  <div className="flex items-center gap-2">
                    <RadioGroupItem value="capacity" id="adjust-type-capacity" />
                    <Label htmlFor="adjust-type-capacity" className="cursor-pointer text-sm">系统盘容量扩容</Label>
                  </div>
                </RadioGroup>
              </div>

              {/* 目标配置 */}
              <div>
                {adjustConfigType === "spec" ? (
                  <div>
                    <div className="text-sm font-medium text-[var(--cp-text-title)] mb-2">目标配置</div>
                    <div className="flex items-center gap-2">
                      <Select key={`spec-sel-${specResetKey}`} value={adjustTargetSpec} onValueChange={setAdjustTargetSpec}>
                        <SelectTrigger className="w-80">
                          <SelectValue placeholder="请选择目标规格" />
                        </SelectTrigger>
                        <SelectContent>
                          {ADJUST_SPEC_OPTIONS.map((o) => {
                            const targetRank = SPEC_RANK[o.value] ?? 0;
                            if (singleInstanceSpecRank === null || targetRank > singleInstanceSpecRank) {
                              return <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>;
                            }
                            const tip = targetRank < singleInstanceSpecRank ? "不支持降配" : "目标规格需高于当前实例规格";
                            return (
                              <Tooltip key={o.value}>
                                <TooltipTrigger asChild>
                                  <div>
                                    <SelectItem value={o.value} disabled>{o.label}</SelectItem>
                                  </div>
                                </TooltipTrigger>
                                <TooltipContent side="left" className="text-xs">{tip}</TooltipContent>
                              </Tooltip>
                            );
                          })}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                ) : (
                  <div>
                    <div className="text-sm font-medium text-[var(--cp-text-title)] mb-2">目标配置</div>
                    <div className="flex items-center gap-2">
                      <Input
                        type="number"
                        min={1}
                        value={adjustTargetCapacity}
                        onChange={(e) => setAdjustTargetCapacity(e.target.value)}
                        placeholder="请输入目标容量"
                        className="w-40"
                      />
                      <span className="text-sm text-[var(--cp-text-body)]">GiB</span>
                    </div>
                    {capacityInputError ? (
                      <p className="mt-1 text-xs text-[var(--cp-text-danger)]">{capacityInputError}</p>
                    ) : diskHelperText ? (
                      <p className="mt-1 text-xs text-[var(--cp-text-muted)]">{diskHelperText}</p>
                    ) : null}
                  </div>
                )}
              </div>
              </div>

              {/* 实例列表 + 校验结果：仅表格内部滚动 */}
              <div className="mt-4 flex flex-col">
                <div className="shrink-0 flex items-center justify-between gap-3 mb-2">
                  <div className="text-sm font-medium text-[var(--cp-text-title)]">实例列表</div>
                  {/* 右上角筛选 Segment：仅影响列表展示，默认「全部」 */}
                  <SegmentGroup className="text-xs">
                    {([
                      { key: "all", label: "全部", count: adjustStats.total },
                      { key: "adjustable", label: "可调整", count: adjustStats.adjustable },
                      { key: "unadjustable", label: "不可调整", count: adjustStats.unadjustable },
                    ] as const).map((it) => (
                      <SegmentOption
                        key={it.key}
                        active={adjustListFilter === it.key}
                        onClick={() => { if (validationPhase !== "validating") setAdjustListFilter(it.key); }}
                      >
                        {it.label} {it.count}
                      </SegmentOption>
                    ))}
                  </SegmentGroup>
                </div>
                <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] overflow-hidden relative">
                  <Table
                    density="compact"
                    className="w-full table-fixed"
                    autoFixedColumns={false}
                    containerClassName="max-h-[260px] overflow-y-auto"
                  >
                    <colgroup>
                      <col style={{ width: 210 }} />
                      <col style={{ width: 65 }} />
                      <col style={{ width: 75 }} />
                      <col style={{ width: 65 }} />
                      <col style={{ width: 75 }} />
                      <col style={{ width: 130 }} />
                    </colgroup>
                    <TableHeader className="sticky top-0 z-10">
                      <TableRow>
                        <TableHead>名称 / ID</TableHead>
                        <TableHead>当前状态</TableHead>
                        <TableHead>实例规格</TableHead>
                        <TableHead>系统盘</TableHead>
                        <TableHead>校验结果</TableHead>
                        <TableHead>说明</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {(() => {
                        const visible = adjustTargetClaws.filter((c) => {
                          const v = adjustResults[c.id] ?? { label: "—", note: "—" };
                          if (adjustListFilter === "adjustable" && v.label !== "可调整") return false;
                          if (adjustListFilter === "unadjustable" && v.label !== "不可调整") return false;
                          return true;
                        });
                        if (visible.length === 0) {
                          return (
                            <TableRow>
                              <TableCell colSpan={6} className="text-center py-12">
                                <p className="text-xs text-[var(--cp-text-weak)]">请选择目标配置后查看可调整实例</p>
                              </TableCell>
                            </TableRow>
                          );
                        }
                        return visible.map((c) => {
                          const info = getInstanceSpecInfo(c);
                          const v = adjustResults[c.id] ?? { label: "—", note: "—" };
                          return (
                            <TableRow key={c.id}>
                              <TableCell className="align-top">
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <MiniBodyText as="div" className="truncate max-w-[186px]">{c.name}</MiniBodyText>
                                  </TooltipTrigger>
                                  <TooltipContent side="top" className="text-xs max-w-xs break-all">{c.name}</TooltipContent>
                                </Tooltip>
                                <div className="text-xs text-[var(--text-muted)] truncate">{c.instanceId}</div>
                              </TableCell>
                              <TableCell className="align-top truncate"><StatusTag mode="text" variant={STATUS_CONFIG[c.status].tagVariant}>{STATUS_CONFIG[c.status].label}</StatusTag></TableCell>
                              <TableCell className="align-top text-[var(--text-secondary)] truncate">{formatSpecShort(info.spec)}</TableCell>
                              <TableCell className="align-top text-[var(--text-secondary)] truncate">{info.systemDiskCapacity}GiB</TableCell>
                              <TableCell className="align-top truncate">
                                {v.label === "可调整" ? (
                                  <StatusTag mode="text" variant="green">可调整</StatusTag>
                                ) : v.label === "不可调整" ? (
                                  <StatusTag mode="text" variant="red">不可调整</StatusTag>
                                ) : (
                                  <span className="text-[var(--text-muted)]">{v.label}</span>
                                )}
                              </TableCell>
                              <TableCell className="align-top text-[var(--text-secondary)]">
                                {v.note ? (
                                  <Tooltip>
                                    <TooltipTrigger asChild>
                                      <span className="block truncate">{v.note}</span>
                                    </TooltipTrigger>
                                    <TooltipContent side="top" className="text-xs max-w-xs break-all">{v.note}</TooltipContent>
                                  </Tooltip>
                                ) : v.note}
                              </TableCell>
                            </TableRow>
                          );
                        });
                      })()}
                    </TableBody>
                  </Table>
                  {validationPhase === "validating" && (
                    <div
                      data-slot="data-table-loading"
                      aria-busy="true"
                      className="absolute inset-0 z-40 flex items-center justify-center bg-white/60 backdrop-blur-[1px]"
                    >
                      <div
                        role="status"
                        aria-label="加载中"
                        className="size-6 animate-spin rounded-full border-2 border-[var(--border-control)] border-t-[var(--cp-brand)]"
                      />
                    </div>
                  )}
                </div>
              </div>
            </DialogBody>
          ) : (
            /* 影响确认页：根据调整类型 + 可调整实例状态展示 warning / info */
            <DialogBody className="px-6 space-y-4">
              {(() => {
                const hasRunning = adjustHasRunningAdjustable;
                if (adjustConfigType === "spec") {
                  return (
                    <>
                      {hasRunning && (
                        <Alert variant="warning">
                          <CircleAlert />
                          <AlertDescription>运行中的 Agent 实例将在升配过程中关机；如正常关机失败，系统可能执行强制关机。请确认实例内任务已完成、数据已保存后再继续。</AlertDescription>
                        </Alert>
                      )}
                      <ul className="list-disc pl-4 space-y-1 text-sm text-[var(--text-secondary)]">
                        <li>{hasRunning ? "升配完成后，原运行中的 Agent 实例会自动开机；原已关机 Agent 实例仍保持关机。" : "升配完成后，Agent 实例仍保持关机。"}</li>
                        <li>升配可能产生费用变化，具体费用以最终订单为准。</li>
                        <li>调整开始后，可在 Agent 列表查看各实例的执行状态和结果。</li>
                      </ul>
                      {hasRunning && (
                        <label className="flex items-start gap-2 cursor-pointer">
                          <Checkbox checked={forceShutdownConfirmed} onCheckedChange={(v) => setForceShutdownConfirmed(!!v)} className="mt-0.5" />
                          <span className="text-sm text-[var(--text-secondary)]">我已知晓并同意对运行中的 Agent 实例执行强制关机</span>
                        </label>
                      )}
                    </>
                  );
                }
                return (
                  <>
                    {hasRunning && (
                      <div>
                        <div className="text-sm font-medium text-[var(--cp-text-title)] mb-2">扩容方式</div>
                        <RadioGroup value={adjustExpandMode} onValueChange={(v) => setAdjustExpandMode(v as "online" | "offline")} className="flex items-center gap-6">
                          <div className="flex items-center gap-2">
                            <RadioGroupItem value="online" id="expand-online" />
                            <Label htmlFor="expand-online" className="cursor-pointer text-sm">在线扩容</Label>
                          </div>
                          <div className="flex items-center gap-2">
                            <RadioGroupItem value="offline" id="expand-offline" />
                            <Label htmlFor="expand-offline" className="cursor-pointer text-sm">关机后扩容</Label>
                          </div>
                        </RadioGroup>
                      </div>
                    )}
                    {hasRunning ? (
                      adjustExpandMode === "online" ? (
                        <ul className="list-disc pl-4 space-y-1 text-sm text-[var(--text-secondary)]">
                              <li>扩容完成后，原运行中的 Agent 实例仍保持运行中；原已关机 Agent 实例仍保持关机。</li>
                              <li>扩容完成后，请登录 Agent 终端确认分区和文件系统是否已完成扩展，未完成时需手动处理，详情请参考<a href="https://cloud.tencent.com/document/product/213/118523" target="_blank" rel="noreferrer" className="text-[var(--text-brand)] underline underline-offset-2">扩容云硬盘<ExternalLink className="w-3 h-3 ml-0.5 inline" /></a></li>
                              <li>扩容可能产生费用变化，具体费用以最终订单为准。</li>
                              <li>调整开始后，可在 Agent 列表查看各实例的执行状态和结果。</li>
                            </ul>
                      ) : (
                        <>
                          <Alert variant="warning">
                            <CircleAlert />
                            <AlertDescription>运行中的 Agent 实例将先关机后扩容；如正常关机失败，系统可能执行<strong>强制关机</strong>。请确认实例内任务已完成、<strong>数据已保存</strong>后再继续。</AlertDescription>
                          </Alert>
                            <ul className="list-disc pl-4 space-y-1 text-sm text-[var(--text-secondary)]">
                              <li>扩容完成后，原运行中的 Agent 实例会自动开机；原已关机 Agent 实例仍保持关机。</li>
                              <li>使用公共镜像创建的 Agent 实例，重启或后续手动开机后通常可自动扩展分区和文件系统；使用自定义镜像创建的 Agent 实例，请登录 Agent 终端确认，未完成时需手动处理，详情请参考<a href="https://cloud.tencent.com/document/product/213/118523" target="_blank" rel="noreferrer" className="text-[var(--text-brand)] underline underline-offset-2">扩容云硬盘<ExternalLink className="w-3 h-3 ml-0.5 inline" /></a></li>
                              <li>扩容可能产生费用变化，具体费用以最终订单为准。</li>
                              <li>调整开始后，可在 Agent 列表查看各实例的执行状态和结果。</li>
                            </ul>
                        </>
                      )
                    ) : (
                      <ul className="list-disc pl-4 space-y-1 text-sm text-[var(--text-secondary)]">
                            <li>扩容完成后，Agent 实例仍保持关机。</li>
                            <li>使用公共镜像创建的 Agent 实例，后续手动开机后通常可自动扩展分区和文件系统；使用自定义镜像创建的 Agent 实例，请登录 Agent 终端确认，未完成时需手动处理，详情请参考<a href="https://cloud.tencent.com/document/product/213/118523" target="_blank" rel="noreferrer" className="text-[var(--text-brand)] underline underline-offset-2">扩容云硬盘<ExternalLink className="w-3 h-3 ml-0.5 inline" /></a></li>
                            <li>扩容可能产生费用变化，具体费用以最终订单为准。</li>
                            <li>调整开始后，可在 Agent 列表查看各实例的执行状态和结果。</li>
                          </ul>
                    )}
                  </>
                );
              })()}
            </DialogBody>
          )}

          {adjustConfigStep === "config" ? (
            <DialogFooter className="gap-2 pt-3">
              <Button variant="claw-outline" onClick={() => setAdjustConfigContext(null)}>取消</Button>
              <Button
                variant="dialog-confirm"
                onClick={() => {
                  if (validationPhase === "idle") {
                    setValidationPhase("validating");
                  } else {
                    setAdjustConfigStep("confirm");
                  }
                }}
                disabled={
                  validationPhase === "validating" ||
                  (validationPhase === "idle" && !canValidate) ||
                  (validationPhase === "validated" && adjustAdjustableClaws.length === 0)
                }
              >
                {validationPhase === "validating" ? "校验中..." : validationPhase === "validated" ? "下一步" : "校验"}
              </Button>
            </DialogFooter>
          ) : (
            <DialogFooter className="gap-2 pt-3">
              <Button variant="claw-outline" onClick={() => setAdjustConfigStep("config")}>上一步</Button>
              <Button
                variant="dialog-confirm"
                disabled={
                  adjustSubmitting ||
                  (adjustConfigType === "spec" && adjustHasRunningAdjustable && !forceShutdownConfirmed)
                }
                onClick={() => {
                  setAdjustSubmitting(true);
                  // 模拟提交：仅对本次「可调整」实例写入 processing，覆盖旧 failed
                  setTimeout(() => {
                    const submittedIds = new Set(adjustAdjustableClaws.map((c) => c.id));
                    const next: Record<string, AdjustOpState> = {};
                    adjustAdjustableClaws.forEach((c) => {
                      next[c.id] = {
                        status: "processing",
                        adjustType: adjustConfigType,
                        targetSpec: adjustConfigType === "spec" ? adjustTargetSpec : undefined,
                        targetDiskSize: adjustConfigType === "capacity" ? adjustTargetCapacityNum : undefined,
                      };
                    });
                    setAdjustOperationMap((prev) => {
                      // 清理旧 autoComplete timer（避免叠加）
                      if (autoCompleteTimerRef.current) clearTimeout(autoCompleteTimerRef.current);
                      // 3000ms 后自动结束 mock 任务
                      autoCompleteTimerRef.current = setTimeout(() => {
                        const FAILED_INSTANCE_ID = "ins-a81v5pwm";
                        setAdjustOperationMap((prevMap) => {
                          const updated = { ...prevMap };
                          const successUpdates: Record<string, { spec?: string; diskSize?: number }> = {};
                          for (const id of submittedIds) {
                            const op = updated[id];
                            if (!op) continue;
                            const isTina = adjustTargetClaws.find((c) => c.id === id)?.instanceId === FAILED_INSTANCE_ID;
                            if (isTina) {
                              updated[id] = { ...op, status: "failed", errorCode: "InternalError" };
                            } else {
                              // 成功：记录资源变更 + 移除状态
                              if (op.adjustType === "spec" && op.targetSpec) {
                                successUpdates[id] = { spec: op.targetSpec };
                              } else if (op.adjustType === "capacity" && op.targetDiskSize) {
                                successUpdates[id] = { diskSize: op.targetDiskSize };
                              }
                              delete updated[id];
                            }
                          }
                          // 回写资源到 state，触发列表/抽屉同步更新
                          if (Object.keys(successUpdates).length > 0) {
                            setResourceUpdates((prev) => ({ ...prev, ...successUpdates }));
                          }
                          return updated;
                        });
                      }, 3000);
                      return { ...prev, ...next };
                    });
                    setAdjustConfigContext(null);
                    setAdjustSubmitting(false);
                    setSelectedIds(new Set()); // 提交成功后清空 Agent 列表选中态
                  }, 800);
                }}
              >
                {adjustSubmitting ? (<><Loader2 className="w-3.5 h-3.5 mr-1 animate-spin" />提交中...</>) : "开始调整"}
              </Button>
            </DialogFooter>
          )}
        </DialogContent>
      </Dialog>

      {/* 批量更新确认弹窗 */}
      <Dialog open={showBatchUpgradeDialog} onOpenChange={setShowBatchUpgradeDialog}>
        <DialogContent className="rounded-[4px] sm:max-w-[780px]">
          <DialogHeader className="pb-4">
            <DialogTitle className="text-[16px] font-semibold text-[var(--text-title)]">批量更新</DialogTitle>
            {(() => {
              // ── 读取目标版本映射 ──
              const targetMap = new Map<string, { version: string; imageName: string }>();
              try {
                const raw = localStorage.getItem("admin_images_v3");
                const imgs: { agentType: string; agentVersion: string; active: boolean; name: string }[] =
                  raw ? JSON.parse(raw) : [];
                for (const img of imgs) {
                  if (img.active && img.agentType && img.agentVersion) {
                    targetMap.set(img.agentType, { version: img.agentVersion, imageName: img.name });
                  }
                }
              } catch { /* ignore */ }

              const rows = claws.filter(c => selectedIds.has(c.id));
              const list = rows.map(c => {
                const imageAgentType = CLAW_TO_IMAGE_AGENT_TYPE[c.agentType] ?? c.agentType;
                const target = targetMap.get(imageAgentType);
                const currentVer = c.version;
                const targetVer = target?.version ?? null;
                let canUpgrade = true;
                let blockReason = "";
                if (!targetVer) {
                  canUpgrade = false;
                  blockReason = "无生效镜像";
                } else if (compareVersion(currentVer, targetVer) >= 0) {
                  canUpgrade = false;
                  blockReason = compareVersion(currentVer, targetVer) === 0
                    ? "已是最新版本"
                    : "不支持降级更新";
                }
                return { ...c, targetVer, targetImageName: target?.imageName ?? "", canUpgrade, blockReason, imageAgentType };
              });
              const upgradableCount = list.filter(l => l.canUpgrade).length;
              const blockedCount = list.length - upgradableCount;

              // 存到闭包外供 confirmBatchUpgrade 使用
              (window as any).__batchUpgradeList = list;

              return (
                <>
                  <DialogDescription>
                    共 <span className="font-din font-bold tabular-nums text-[var(--text-title)]">{list.length}</span> 个实例
                    {upgradableCount > 0 && (
                      <span>，其中 <span className="font-din font-bold tabular-nums text-[var(--text-brand)]">{upgradableCount}</span> 个可更新</span>
                    )}
                    {blockedCount > 0 && (
                      <span className="text-[var(--text-danger)]">，<span className="font-din font-bold tabular-nums">{blockedCount}</span> 个无法更新</span>
                    )}
                  </DialogDescription>

                  <div className="space-y-4 mt-3">
                    <Alert variant="warning" className="items-start px-3 py-2.5 [&>svg]:translate-y-[1px]">
                      <CircleAlert />
                      <AlertDescription>
                        <p>更新预计需要 5～10 分钟，期间 Agent 实例不可使用。</p>
                        <p>仅支持升级到更高版本，不支持降级更新。</p>
                      </AlertDescription>
                    </Alert>

                    <div>
                      <div className="text-sm font-medium text-[var(--text-title)] mb-2">待更新实例</div>

                      <Table
                        density="compact"
                        containerClassName="max-h-[300px] overflow-y-auto rounded-[4px] border border-[var(--border)] scrollbar-on-hover"
                      >
                        <TableHeader>
                          <TableRow>
                            <TableHead className="sticky top-0 z-10">实例</TableHead>
                            <TableHead className="sticky top-0 z-10">Agent 类型</TableHead>
                            <TableHead className="sticky top-0 z-10">当前版本</TableHead>
                            <TableHead className="sticky top-0 z-10">目标版本</TableHead>
                            <TableHead className="sticky top-0 z-10">状态</TableHead>
                            <TableHead className="sticky top-0 z-10">操作</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {list.length === 0 ? (
                            <TableRow className="hover:bg-transparent">
                              <TableCell colSpan={6} className="h-[60px] text-center text-sm text-[var(--text-weak)]">
                                暂无待更新实例
                              </TableCell>
                            </TableRow>
                          ) : list.map(item => {
                            const sc = STATUS_CONFIG[item.status];
                            const isBlocked = !item.canUpgrade;
                            return (
                              <TableRow key={item.id} className={isBlocked ? "opacity-50" : ""}>
                                <TableCell className="h-[52px] min-w-[160px] max-w-[220px] py-2.5">
                                  <div className="min-w-0">
                                    <div className="truncate text-xs font-medium text-[var(--text-title)]">{item.name}</div>
                                    <div className="font-mono text-xs text-[var(--text-weak)]">{item.instanceId}</div>
                                  </div>
                                </TableCell>
                                <TableCell className="h-[52px] py-2.5">
                                  <span className="text-xs text-[var(--text-secondary)]">{AGENT_TYPE_DISPLAY[item.agentType] ?? item.agentType}</span>
                                </TableCell>
                                <TableCell className="h-[52px] py-2.5">
                                  <span className="font-mono text-xs text-[var(--text-secondary)]">{item.version}</span>
                                </TableCell>
                                <TableCell className="h-[52px] py-2.5">
                                  {item.targetVer ? (
                                    <div className="flex flex-col gap-0.5">
                                      <span className={cn("font-mono text-xs", isBlocked ? "text-[var(--text-muted)] line-through" : "text-[var(--text-brand)] font-medium")}>
                                        {item.targetVer}
                                      </span>
                                      {isBlocked && (
                                        <span className="text-[10px] leading-none text-[var(--text-danger)]">{item.blockReason}</span>
                                      )}
                                    </div>
                                  ) : (
                                    <span className="text-[10px] text-[var(--text-danger)]">{item.blockReason}</span>
                                  )}
                                </TableCell>
                                <TableCell className="h-[52px] py-2.5">
                                  <StatusTag mode="text" variant={sc.tagVariant}>
                                    {sc.label}
                                  </StatusTag>
                                </TableCell>
                                <TableActionCell rawChildren className="h-[52px] py-2.5">
                                  <Button
                                    variant="link"
                                    onClick={() => setSelectedIds(prev => { const n = new Set(prev); n.delete(item.id); return n; })}
                                  >
                                    移除
                                  </Button>
                                </TableActionCell>
                              </TableRow>
                            );
                          })}
                        </TableBody>
                      </Table>
                    </div>
                  </div>
                </>
              );
            })()}
          </DialogHeader>

          <DialogFooter className="gap-2">
            <Button variant="claw-outline" size="claw-sm" onClick={() => setShowBatchUpgradeDialog(false)}>取消</Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={confirmBatchUpgrade}
              disabled={selectedIds.size === 0}
            >
              确认更新
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>


      {/* 批量升级失败结果弹窗 */}
      <Dialog open={showUpgradeResultDialog} onOpenChange={setShowUpgradeResultDialog}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle className="text-base font-bold text-[var(--text-title)]">下发失败提醒</DialogTitle>
          </DialogHeader>
          <Alert variant="warning" className="items-start [&>svg]:translate-y-[1px]">
            <CircleAlert />
            <AlertDescription>
              <p>部分 Agent 实例无法更新，原因如下：</p>
            </AlertDescription>
          </Alert>
          <BodyText tone="secondary">
            任务已提交，以下 <MetaMedium as="span" tone="danger" className="tabular-nums">{upgradeFailedAgents.length}</MetaMedium> 个实例无法执行
          </BodyText>
          <SurfaceInner className="max-h-64 overflow-y-auto p-0 scrollbar-on-hover">
            <Table density="compact">
              <TableHeader>
                <TableRow>
                  <TableHead>实例</TableHead>
                  <TableHead>Agent 类型</TableHead>
                  <TableHead>下发失败原因</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {upgradeFailedAgents.map((a, idx) => (
                  <TableRow key={idx}>
                    <TableCell>
                      <div className="min-w-0">
                        <MiniBodyText as="div" className="truncate">{a.name}</MiniBodyText>
                        <CodeText tone="weak" className="truncate">{a.instanceId}</CodeText>
                      </div>
                    </TableCell>
                    <TableCell>
                      <MetaText as="span" tone="secondary">{AGENT_TYPE_DISPLAY[a.agentType] ?? a.agentType}</MetaText>
                    </TableCell>
                    <TableCell>
                      <MetaText as="span" tone="danger">{a.reason}</MetaText>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </SurfaceInner>
          <DialogFooter className="gap-2 pt-2">
            <Button onClick={() => setShowUpgradeResultDialog(false)}>
              我知道了
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 配置默认标签弹窗 */}
      <Dialog open={showTagConfigDialog} onOpenChange={(open) => { if (!open) { setShowTagConfigDialog(false); setKeySearchText(''); setOpenKeyRow(null); setOpenValueRow(null); } }}>
        <DialogContent size="lg" className="max-h-[90vh] grid-rows-[auto_minmax(0,1fr)_auto]">
          <DialogHeader>
            <DialogTitle>配置默认标签</DialogTitle>
          </DialogHeader>

          <DialogBody className="px-6 space-y-4">
            <Alert variant="info" className="py-3">
              <AlertInfoIcon />
              <AlertDescription>
                <ol className="list-decimal space-y-1.5 pl-4 leading-relaxed">
                  <li>当前仅支持使用<a href="https://console.cloud.tencent.com/tag/taglist" target="_blank" rel="noopener noreferrer" className="mx-0.5 text-[var(--text-brand)] hover:underline" onClick={(e) => e.stopPropagation()}>腾讯云控制台</a>已创建的标签。</li>
                  <li>将在用户端新建实例时自动配置标签（仅限新建实例，已创建实例暂不支持绑定标签）。</li>
                  <li>可为每个标签设置应用范围：当用户使用所选组织新建实例时，系统会自动为该实例打上对应标签；选择「全部用户」则所有用户新建实例都会打上该标签。</li>
                </ol>
              </AlertDescription>
            </Alert>

            <SurfaceInner className="overflow-hidden">
              <div className="max-h-[420px] overflow-y-auto">
                <Table density="compact" autoFixedColumns={false}>
                  <TableHeader className="sticky top-0 z-10">
                    <TableRow>
                      <TableHead className="w-[58%]">标签（键：值）</TableHead>
                      <TableHead className="w-[27%]">应用范围</TableHead>
                      <TableHead className="w-[15%] text-center">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {pendingTags.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={3} className="py-10">
                          <div className="space-y-1 text-center">
                            <MetaText as="p" tone="weak">暂无默认标签</MetaText>
                            <MetaText as="p" tone="weak">点击下方「添加标签」开始配置</MetaText>
                          </div>
                        </TableCell>
                      </TableRow>
                    )}
                    {pendingTags.map((tag, idx) => {
                      const tagValueOptions = (tag.key ? tagKeyValues[tag.key] ?? [] : []).map((value) => ({
                        value,
                        label: value,
                      }));

                      return (
                        <TableRow key={idx}>
                          <TableCell className="whitespace-normal">
                            <div className="grid min-w-0 grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-1.5">
                              <SearchableSelect
                                options={tagKeys.map((key) => ({ value: key, label: key }))}
                                value={tag.key}
                                onChange={(key) => {
                                  setPendingTags((prev) => prev.map((item, i) => (
                                    i === idx ? { ...item, key, value: "" } : item
                                  )));
                                }}
                                placeholder="选择标签键"
                                searchPlaceholder="搜索标签键..."
                                triggerClassName="h-8 text-xs"
                                panelWidth="288px"
                              />
                              <MetaText as="span" tone="weak">:</MetaText>
                              <SearchableSelect
                                options={tagValueOptions}
                                value={tag.value}
                                onChange={(value) => {
                                  setPendingTags((prev) => prev.map((item, i) => (
                                    i === idx ? { ...item, value } : item
                                  )));
                                }}
                                placeholder={tag.key ? "选择值" : "请先选择键"}
                                searchPlaceholder="搜索标签值..."
                                disabled={!tag.key}
                                triggerClassName="h-8 text-xs"
                                panelWidth="220px"
                              />
                            </div>
                          </TableCell>
                          <TableCell className="whitespace-normal">
                            <ScopeSelect
                              scope={tag.scope ?? "all"}
                              selectedGroupIds={tag.groupIds ?? []}
                              groups={hasOneid ? MOCK_GROUPS : MOCK_MANUAL_GROUPS}
                              onConfirm={(scope, ids) => setPendingTags((prev) => prev.map((item, i) => (
                                i === idx ? { ...item, scope, groupIds: ids } : item
                              )))}
                            />
                          </TableCell>
                          <TableActionCell className="justify-center">
                            <Button
                              variant="link"
                              onClick={() => setPendingTags((prev) => prev.filter((_, i) => i !== idx))}
                            >
                              删除
                            </Button>
                          </TableActionCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>
              <div className="border-t border-[var(--border)] p-2">
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  className="w-full justify-center"
                  onClick={() => setPendingTags((prev) => [...prev, { key: "", value: "", scope: "all", groupIds: [] }])}
                >
                  <Plus className="size-4" />
                  添加标签
                </Button>
              </div>
            </SurfaceInner>
          </DialogBody>

          <DialogFooter>
            <Button size="claw-sm" variant="claw-outline" onClick={() => { setShowTagConfigDialog(false); setKeySearchText(''); setOpenKeyRow(null); setOpenValueRow(null); }}>取消</Button>
            <Button
              size="claw-sm"
              onClick={() => {
                // 存在填写不完整的行（只选了键或只选了值）
                if (pendingTags.some(t => (t.key && !t.value) || (!t.key && t.value))) {
                  toast.error('请完善标签的键和值');
                  return;
                }
                // 仅保留键值均已填写的有效标签
                const validTags = pendingTags.filter(t => t.key && t.value);
                // 重复键校验
                const keys = validTags.map(t => t.key);
                if (new Set(keys).size !== keys.length) {
                  toast.error('存在重复的标签键，请检查');
                  return;
                }
                setSelectedTags(validTags);
                setShowTagConfigDialog(false);
                setKeySearchText(''); setOpenKeyRow(null); setOpenValueRow(null);
                toast.success(
                  validTags.length > 0
                    ? `已配置 ${validTags.length} 个默认标签，新建实例将自动打 tag`
                    : '已清空默认标签配置'
                );
              }}
              variant="dialog-confirm"
            >
              确认
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 组织处理 — 随用户迁移到新组织 */}
      <Dialog open={showGroupMigrateDialog} onOpenChange={setShowGroupMigrateDialog}>
        <DialogContent className="sm:max-w-2xl max-h-[85vh] flex flex-col" onOpenAutoFocus={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle>随用户迁移到新组织</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
          <div className="py-2 space-y-4">
            <p className="text-sm text-[var(--text-secondary)]">
              将以下 <span className="font-semibold text-[var(--text-title)]">{selectedCount}</span> 个 Agent 实例随用户一并迁移至用户所在的组织：
            </p>

            {/* 待迁移实例列表 */}
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1.5">待迁移实例</div>
              <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] overflow-hidden">
                <Table className="table-fixed" autoFixedColumns={false} containerClassName="max-h-[220px] overflow-y-auto">
                  <TableHeader className="sticky top-0 z-10">
                    <TableRow>
                      <TableHead className="w-[44%]">Agent 实例名称 / ID</TableHead>
                      <TableHead className="w-[26%]">用户 ID</TableHead>
                      <TableHead className="w-[30%]">实例组织</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {selectedClaws.map((c) => (
                      <TableRow key={c.id}>
                        <TableCell>
                          <div className="text-sm text-[var(--text-title)] truncate">{c.name}</div>
                          <div className="text-xs text-[var(--text-muted)] truncate">{c.instanceId}</div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm text-[var(--text-body)] truncate">{c.creator}</div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm text-[var(--text-body)] truncate">{getInstanceGroupName(c)}</div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>

            {/* 目标组织（仅该用户所属组织可选；用户名下所有组织均可选，不校验是否为实例当前组织） */}
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1.5">目标组织</div>
              {hasMigrateUserGroups ? (
                <Select value={groupMigrateTarget} onValueChange={setGroupMigrateTarget}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="请选择目标组织" />
                  </SelectTrigger>
                  <SelectContent>
                    {migrateUserGroupOptions.map((opt) => (
                      <SelectItem key={opt.id} value={opt.id}>{opt.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <div className="flex items-center justify-between w-full h-9 px-3 rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] text-sm text-[var(--text-weak)] cursor-not-allowed">
                  <span className="truncate">回退为未分配组织配置</span>
                </div>
              )}
              {(hasMigrateUserGroups ? !!groupMigrateTarget : true) && (
                <button
                  type="button"
                  onClick={() => setShowMigrateConfigDiff(true)}
                  className="mt-2 inline-flex items-center gap-1 text-xs text-[var(--text-brand)] hover:underline"
                >
                  <Eye className="w-3.5 h-3.5" />
                  查看配置对比
                </button>
              )}
            </div>

            {/* 统一说明 */}
            <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-3 py-2.5">
              <div className="text-xs font-medium text-[var(--text-secondary)] mb-1">迁移说明：</div>
              <ul className="space-y-1 text-xs leading-relaxed text-[var(--text-secondary)] list-disc pl-4">
                <li>Agent 实例迁移后，实例的平台策略会立即应用新组织配置，其他已配置项保留不变。</li>
                <li>用户后续修改配置项时只能改为新组织的配置。</li>
                <li>管理员可后续到 Agent 列表查看实例与新组织的配置对比并调整配置项。</li>
                <li>Agent 实例迁移后，实例将自动开机。</li>
              </ul>
            </div>
          </div>
          </DialogBody>
          <DialogFooter className="shrink-0">
            <Button variant="claw-outline" size="claw-sm" onClick={() => setShowGroupMigrateDialog(false)}>取消</Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              disabled={hasMigrateUserGroups && !groupMigrateTarget}
              onClick={() => {
                setShowGroupMigrateDialog(false);
                toast.success(`已将 ${selectedCount} 个实例随用户迁移到${hasMigrateUserGroups ? "新组织" : "未分配组织配置"}`);
                setSelectedIds(new Set());
              }}
            >
              确认迁移
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 随用户迁移 — 配置对比 */}
      <ConfigDiffDialog
        open={showMigrateConfigDiff}
        onOpenChange={setShowMigrateConfigDiff}
        newGroupName={migrateToGroupName}
        instances={buildMockInstanceCompare(
          selectedClaws.map((c) => ({ instanceName: c.name, instanceId: c.instanceId }))
        )}
      />

      {/* 组织处理 — 移交给其他用户 */}
      <Dialog open={showGroupTransferDialog} onOpenChange={setShowGroupTransferDialog}>
        <DialogContent className="sm:max-w-2xl max-h-[85vh] flex flex-col" onOpenAutoFocus={(e) => e.preventDefault()}>
          <DialogHeader>
            <DialogTitle>移交给其他用户</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
          <div className="py-2 space-y-4">
            <p className="text-sm text-[var(--text-secondary)]">
              将以下 <span className="font-semibold text-[var(--text-title)]">{selectedCount}</span> 个 Agent 实例移交给其他用户。移交后实例归属变更为接手用户，并随其迁移到对应组织。
            </p>

            {/* 待移交实例列表 */}
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1.5">待移交实例</div>
              <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] overflow-hidden">
                <Table className="table-fixed" autoFixedColumns={false} containerClassName="max-h-[200px] overflow-y-auto">
                  <TableHeader className="sticky top-0 z-10">
                    <TableRow>
                      <TableHead className="w-[44%]">Agent 实例名称 / ID</TableHead>
                      <TableHead className="w-[26%]">用户 ID</TableHead>
                      <TableHead className="w-[30%]">实例组织</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {selectedClaws.map((c) => (
                      <TableRow key={c.id}>
                        <TableCell>
                          <div className="text-sm text-[var(--text-title)] truncate">{c.name}</div>
                          <div className="text-xs text-[var(--text-muted)] truncate">{c.instanceId}</div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm text-[var(--text-body)] truncate">{c.creator}</div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm text-[var(--text-body)] truncate">{getInstanceGroupName(c)}</div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>

            {/* 目标用户双栏选择器：左组织树 + 右成员单选，顶部统一分类搜索框联动 */}
            <div>
              <div className="text-xs text-[var(--text-muted)] mb-1.5">选择接手用户</div>
              {/* 统一搜索框：按 用户ID / 组织名称 搜索，联动左右列表 */}
              <div className="mb-2 flex items-center gap-2">
                <Select
                  value={transferSearchField}
                  onValueChange={(v) => {
                    setTransferSearchField(v as "userId" | "groupName");
                    setTransferSearchKeyword("");
                  }}
                >
                  <SelectTrigger className="w-[116px] shrink-0">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="userId">用户 ID</SelectItem>
                    <SelectItem value="groupName">组织名称</SelectItem>
                  </SelectContent>
                </Select>
                <div className="relative flex-1">
                  <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)]" />
                  <Input
                    value={transferSearchKeyword}
                    onChange={(e) => setTransferSearchKeyword(e.target.value)}
                    placeholder={transferSearchField === "userId" ? "搜索用户 ID" : "搜索组织名称"}
                    className="pl-8"
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 rounded-[var(--radius-lg)] border border-[var(--cp-border)] overflow-hidden" style={{ height: 240 }}>
                {/* 左：组织列表 */}
                <div className="border-r border-[var(--cp-border)] overflow-y-auto py-1">
                  {transferAllGroupTree.length === 0 ? (
                    <div className="h-full flex items-center justify-center text-xs text-[var(--text-weak)]">无匹配组织</div>
                  ) : (
                    transferAllGroupTree.map((root) => (
                      <TransferGroupTreeNode
                        key={`all-${root.id}`}
                        node={root}
                        level={0}
                        selected={transferPickerGroupId}
                        expanded={transferTreeExpanded}
                        onToggle={(id) =>
                          setTransferTreeExpanded((prev) => {
                            const next = new Set(prev);
                            if (next.has(id)) next.delete(id); else next.add(id);
                            return next;
                          })
                        }
                        onSelect={setTransferPickerGroupId}
                      />
                    ))
                  )}
                </div>
                {/* 右：成员列表（单选） */}
                <div className="overflow-y-auto py-1">
                  {!transferPickerGroupId && !transferSearchByUser ? (
                    <div className="h-full flex items-center justify-center text-xs text-[var(--text-weak)]">请选择左侧组织</div>
                  ) : transferGroupMembers.length === 0 ? (
                    <div className="h-full flex items-center justify-center text-xs text-[var(--text-weak)]">{transferSearchByUser ? "未找到匹配用户" : "该组织暂无可选用户"}</div>
                  ) : (
                    transferGroupMembers.map((u) => {
                      const selected = groupTransferTarget === u.userId;
                      return (
                        <button
                          type="button"
                          key={u.userId}
                          onClick={() => {
                            setGroupTransferTarget(u.userId);
                            const opts = u.groupIds.filter((gid) => transferGroups.find((g) => g.id === gid)?.source !== "oneid-dept");
                            setGroupTransferTargetGroup(opts[0] ?? "");
                          }}
                          className={`mx-1 mb-0.5 flex w-[calc(100%-8px)] items-center gap-2 rounded-[4px] h-9 px-2.5 text-sm text-left transition-colors ${
                            selected
                              ? "bg-[var(--cp-brand-tint)] text-[var(--text-brand)]"
                              : "text-[var(--text-body)] hover:bg-[var(--bg-grey-hover)]"
                          }`}
                        >
                          <span className="flex-1 min-w-0 truncate">{u.userId}</span>
                          {selected && <Check className="w-4 h-4 shrink-0 text-[var(--text-brand)]" />}
                        </button>
                      );
                    })
                  )}
                </div>
              </div>
            </div>

            {/* 目标组织（仅选完接手用户后出现） */}
            {groupTransferTarget && (
              <div>
                <div className="text-xs text-[var(--text-muted)] mb-1.5">目标组织</div>
                {hasTransferTargetGroups ? (
                  <Select value={groupTransferTargetGroup} onValueChange={setGroupTransferTargetGroup}>
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="请选择目标组织" />
                    </SelectTrigger>
                    <SelectContent>
                      {transferTargetGroupOptions.map((opt) => (
                        <SelectItem key={opt.id} value={opt.id}>{opt.label}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <div className="flex items-center justify-between w-full h-9 px-3 rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] text-sm text-[var(--text-weak)] cursor-not-allowed">
                    <span className="truncate">回退为未分配组织配置</span>
                  </div>
                )}
                {(hasTransferTargetGroups ? !!groupTransferTargetGroup : true) && (
                  <button
                    type="button"
                    onClick={() => setShowTransferConfigDiff(true)}
                    className="mt-2 inline-flex items-center gap-1 text-xs text-[var(--text-brand)] hover:underline"
                  >
                    <Eye className="w-3.5 h-3.5" />
                    查看配置对比
                  </button>
                )}
              </div>
            )}

            {/* 移交说明 */}
            <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-3 py-2.5">
              <div className="text-xs font-medium text-[var(--text-secondary)] mb-1">移交说明：</div>
              <ul className="space-y-1 text-xs leading-relaxed text-[var(--text-secondary)] list-disc pl-4">
                <li>Agent 实例移交后，实例的平台策略会立即应用新组织配置，其他已配置项保留不变。</li>
                <li>用户后续修改配置项时只能改为新组织的配置。</li>
                <li>管理员可后续到 Agent 列表查看实例与新组织的配置对比并调整配置项。</li>
                <li>Agent 实例移交后，实例将自动开机。</li>
              </ul>
            </div>
          </div>
          </DialogBody>
          <DialogFooter className="shrink-0">
            <Button variant="claw-outline" size="claw-sm" onClick={() => setShowGroupTransferDialog(false)}>取消</Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              disabled={!groupTransferTarget || (hasTransferTargetGroups && !groupTransferTargetGroup)}
              onClick={() => {
                setShowGroupTransferDialog(false);
                toast.success(`已将 ${selectedCount} 个实例移交给 ${groupTransferTarget}`);
                setSelectedIds(new Set());
              }}
            >
              确认移交
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 移交给其他用户 — 配置对比 */}
      <ConfigDiffDialog
        open={showTransferConfigDiff}
        onOpenChange={setShowTransferConfigDiff}
        newGroupName={transferToGroupName}
        instances={buildMockInstanceCompare(
          selectedClaws.map((c) => ({ instanceName: c.name, instanceId: c.instanceId }))
        )}
      />

      {/* 组织列「眼睛」入口现以「对比模式」复用下方 Agent 详情抽屉，不再单独渲染对比抽屉 */}

      {/* Agent 详情抽屉 */}
      <Drawer
        open={showDetailDrawer && !!selectedClaw}
        onOpenChange={(open) => { setShowDetailDrawer(open); if (!open) setCompareMode(false); }}
        direction="right"
      >
        {selectedClaw && (
          <DrawerContent className={`data-[vaul-drawer-direction=right]:${compareMode ? "w-[1040px]" : "w-[480px]"} data-[vaul-drawer-direction=right]:sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-background p-0`}>
            {/* 抽屉头 */}
            <DrawerHeader className="flex flex-row items-center justify-between gap-4 p-4 bg-background text-left">
              <DrawerTitle asChild>
                <PanelTitle as="h2">{compareMode ? "实例配置与组织配置对比" : "Agent 详情"}</PanelTitle>
              </DrawerTitle>
              <div className="flex items-center gap-1">
                {!compareMode && (
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-7 w-7 p-0 text-[var(--text-title)] hover:text-[var(--cp-brand-black)]"
                  onClick={handleRefreshDrawer}
                  disabled={drawerLoading}
                  aria-label="刷新"
                >
                  <RefreshCw className={`w-4 h-4 ${drawerLoading ? "animate-spin" : ""}`} />
                </Button>
                )}
                <DrawerClose asChild>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-7 w-7 p-0 text-[var(--text-title)] hover:text-[var(--cp-brand-black)]"
                    aria-label="关闭"
                  >
                    <X className="w-4 h-4" />
                  </Button>
                </DrawerClose>
              </div>
            </DrawerHeader>
            {/* 抽屉内容 */}
            <DrawerBody>
              {(() => {
              if (selectedClaw.agentType === "LocalAgent") return renderLocalAgentDetail(selectedClaw);
              const blkIdentity = (
                <div className="min-w-0 space-y-1.5">
                    <PanelTitle as="div" className="truncate leading-tight">{selectedClaw.name}</PanelTitle>
                    <div className="flex items-center gap-2">
                      <CodeText>{selectedClaw.instanceId}</CodeText>
                      <MetaText
                        as="button"
                        tone="brand"
                        className="inline-flex items-center gap-0.5 whitespace-nowrap hover:text-[var(--text-brand)]"
                        onClick={() => window.open(`https://console.cloud.tencent.com/cvm/instance/detail?rid=1&id=${selectedClaw.instanceId}`, "_blank")}
                      >
                        去腾讯云控制台管理
                        <ExternalLink className="w-3 h-3" />
                      </MetaText>
                    </div>
                  </div>
              );
                // 已应用模型
              const blkModels = (() => {
                  const detail = getClawDetail(selectedClaw.id);
                  const models = detail.appliedModels;
                  const hasPrimary = models.some(m => m.primary);
                  const primaryList = models.filter(m => m.primary);
                  const backupList = [...models.filter(m => !m.primary)].sort((a, b) => b.addedAt - a.addedAt);
                  // 是否为 OpenClaw 类型：OpenClaw 支持主模型 + 备选模型；其他类型只能配置一个模型
                  const isOpenClaw = selectedClaw.agentType === 'OpenClaw';
                  // 非 OpenClaw 且已有模型时不展示添加按鈕；OpenClaw 按现有逻辑
                  const canAddMore = isOpenClaw || models.length === 0;
                  const addButtonLabel = isOpenClaw
                    ? (hasPrimary ? "添加备选模型" : "添加主模型")
                    : "添加模型";
                  const isAdding = modelAction.kind === "add";

                  /** 卡片内两级 Select + 保存/取消（替换态 / 新增态共用） */
                  const renderInlineEditForm = () => (
                    <div className="bg-muted/30 p-3">
                      {providerGroups.length === 0 ? (
                        <div className="flex items-start gap-2.5 bg-[var(--alert-warning-bg)] border border-[var(--border)] rounded-[4px] px-3 py-2.5">
                          <AlertCircle className="w-4 h-4 text-[var(--text-warning)] mt-0.5 shrink-0" />
                          <MetaText as="p" className="leading-relaxed">
                            当前「模型配置」页中没有对用户可见的模型，请前往该页面添加或开启模型可见性。
                          </MetaText>
                        </div>
                      ) : (
                        <div className="overflow-hidden rounded-[4px] border border-[var(--border)] bg-background">
                          <div className="border-b border-[var(--border)] px-3 py-2">
                            <MetaMedium>模型配置</MetaMedium>
                          </div>
                          <div className="divide-y divide-[var(--border)]">
                            <div className="px-3 py-2 space-y-1.5">
                              <MetaMedium as="label" tone="secondary">模型厂商</MetaMedium>
                              <Select value={modelDraftProvider} onValueChange={handleDraftProviderChange}>
                                <SelectTrigger className="w-full bg-background border-[var(--border)] h-8 text-xs">
                                  <SelectValue placeholder="选择模型厂商" />
                                </SelectTrigger>
                                <SelectContent>
                                  {providerGroups.map((g) => (
                                    <SelectItem key={g.key} value={g.key}>{g.label}</SelectItem>
                                  ))}
                                </SelectContent>
                              </Select>
                            </div>
                            <div className="px-3 py-2 space-y-1.5">
                              <MetaMedium as="label" tone="secondary">模型名称</MetaMedium>
                              <Select value={modelDraftModelId} onValueChange={setModelDraftModelId}>
                                <SelectTrigger className="w-full bg-background border-[var(--border)] h-8 text-xs">
                                  <SelectValue placeholder="选择模型名称" />
                                </SelectTrigger>
                                <SelectContent>
                                  {(providerGroups.find(g => g.key === modelDraftProvider)?.models ?? []).map((m) => {
                                    const isCustom = m.provider === CUSTOM_PROVIDER_VALUE;
                                    return (
                                      <SelectItem key={m.id} value={m.id}>
                                        {isCustom ? m.name : m.version}
                                      </SelectItem>
                                    );
                                  })}
                                </SelectContent>
                              </Select>
                            </div>
                          </div>
                          <div className="flex justify-end gap-2 border-t border-[var(--border)] px-3 py-2">
                            <Button size="sm" variant="ghost" className="h-7 px-2 text-xs" onClick={cancelEditModel}>
                              取消
                            </Button>
                            <Button
                              size="sm"
                              variant="dialog-confirm"
                              className="h-7 px-2 text-xs"
                              onClick={saveEditModel}
                              disabled={!modelDraftProvider || !modelDraftModelId}
                            >
                              保存
                            </Button>
                          </div>
                        </div>
                      )}
                    </div>
                  );

                  /** 渲染一行模型卡：替换态下卡片内直接变为编辑表单 */
                  const renderModelRow = (model: AppliedModelItem, isPrimary: boolean) => {
                    const isReplacingThis = modelAction.kind === "replace" && modelAction.modelEntryId === model.id;
                    return (
                      <div
                        key={model.id}
                        className={isReplacingThis
                          ? "bg-[var(--popover)] rounded-[4px] border border-[var(--border)] overflow-hidden"
                          : "px-4 py-3 bg-[var(--popover)] rounded-[4px] border border-[var(--border)] transition-colors"}
                      >
                        {isReplacingThis ? (
                          renderInlineEditForm()
                        ) : (
                          <div className="flex items-center gap-3">
                            <div className="flex flex-col min-w-0 flex-1 overflow-hidden">
                              <BodyMedium className="truncate leading-tight">
                                {model.providerLabel}
                              </BodyMedium>
                              {model.versionLabel && (
                                <MetaText tone="weak" className="leading-tight mt-0.5 truncate">
                                  {model.versionLabel}
                                </MetaText>
                              )}
                            </div>
                            {isOpenClaw && (isPrimary ? (
                              <StatusTag mode="fill" variant="green">主模型</StatusTag>
                            ) : (
                              <StatusTag mode="fill" variant="gray">备选模型</StatusTag>
                            ))}
                            <div className="flex items-center gap-1 shrink-0">
                              {isOpenClaw && !isPrimary && (
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <button
                                      type="button"
                                      onClick={() => setModelConfirmDialog({ open: true, type: "set-primary", modelEntryId: model.id })}
                                      className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors"
                                    >
                                      <ArrowLeftRight className="w-3.5 h-3.5" />
                                    </button>
                                  </TooltipTrigger>
                                  <TooltipContent side="top" className="text-xs">
                                    设为主模型
                                  </TooltipContent>
                                </Tooltip>
                              )}
                              {!isOpenClaw && (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <button
                                    type="button"
                                    onClick={() => startReplaceModel(model)}
                                    className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors"
                                  >
                                    <Pencil className="w-3.5 h-3.5" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="text-xs">
                                  替换
                                </TooltipContent>
                              </Tooltip>
                              )}
                              {isOpenClaw && (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <button
                                    type="button"
                                    onClick={() => setModelConfirmDialog({
                                      open: true,
                                      type: isPrimary ? "delete" : "delete-backup",
                                      modelEntryId: model.id,
                                    })}
                                    className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors"
                                  >
                                    <Trash2 className="w-3.5 h-3.5" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent side="top" className="text-xs">
                                  删除模型
                                </TooltipContent>
                              </Tooltip>
                              )}
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  };

                  return (
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <MetaText as="div">已应用模型（{models.length}）</MetaText>
                        {!isAdding && canAddMore && (
                          <MetaText
                            as="button"
                            tone="brand"
                            onClick={startAddModel}
                            className="flex items-center gap-1 hover:text-[var(--text-brand)] transition-colors"
                          >
                            <Plus className="w-3 h-3" />
                            {addButtonLabel}
                          </MetaText>
                        )}
                      </div>

                      {/* 空态（无模型且不在新增态） */}
                      {models.length === 0 && !isAdding && (
                        <MetaText as="div" tone="weak" className="px-4 py-6 bg-background rounded-[4px] border border-dashed border-[var(--border)] text-center">
                          暂未配置模型
                        </MetaText>
                      )}

                      {/* 主模型组织 */}
                      {primaryList.length > 0 && (
                        <div className="space-y-1.5 mb-3">
                          {primaryList.map((m) => renderModelRow(m, true))}
                        </div>
                      )}

                      {/* 备选模型组织 */}
                      {backupList.length > 0 && (
                        <div>
                          <div className="space-y-1.5">
                            {backupList.map((m) => renderModelRow(m, false))}
                          </div>
                        </div>
                      )}

                      {/* 新增态：底部 inline 卡（替换态已在行内展示，不再重复渲染） */}
                      {isAdding && (
                        <div className="mt-2 bg-[var(--popover)] rounded-[4px] border border-[var(--border)] overflow-hidden">
                          {renderInlineEditForm()}
                        </div>
                      )}
                    </div>
                  );
                })();
                // 已接入通道
              const blkChannels = (
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <MetaText as="div">已接入通道（{getClawDetail(selectedClaw.id).connectedChannels.length}）</MetaText>
                    {!channelAdding && (
                      <MetaText
                        as="button"
                        tone="brand"
                        className="flex items-center gap-1 hover:text-[var(--text-brand)] transition-colors"
                        onClick={() => startAddChannel(getClawDetail(selectedClaw.id))}
                      >
                        <Plus className="w-3 h-3" />
                        添加通道
                      </MetaText>
                    )}
                  </div>
                  <div className="space-y-2">
                    {getClawDetail(selectedClaw.id).connectedChannels.map((channel) => {
                      const chConfig = channelLookup.get(channel.value);
                      const fields = chConfig?.fields ?? [];
                      const isExpanded = expandedChannel === channel.name;
                      const isEditingThis = isExpanded && channelEditDraft !== null;
                      return (
                        <div key={channel.name} className="bg-[var(--popover)] rounded-[4px] border border-[var(--border)] overflow-hidden">
                          {/* 行头：通道名 + 展开/折叠按钮 */}
                          <div className="group px-4 py-3 flex items-center gap-3">
                            <button
                              onClick={() => toggleExpandChannel(channel)}
                              className="text-[var(--text-weak)] hover:text-[var(--text-muted)] transition-colors flex-shrink-0"
                              title={isExpanded ? "收起" : "展开查看凭证"}
                            >
                              <ChevronRight className={`w-4 h-4 transition-transform ${isExpanded ? "rotate-90" : ""}`} />
                            </button>
                            <BodyMedium className="flex-1">{channel.name}</BodyMedium>
                            <button
                              onClick={() => setChannelRemoveTarget(channel.name)}
                              className="text-[var(--text-weak)] hover:text-[var(--text-danger)] transition-colors opacity-0 group-hover:opacity-100"
                              title="移除"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          </div>

                          {/* 展开区域：凭证查看 / 编辑 */}
                          {isExpanded && (
                            <div className="border-t border-[var(--border)] bg-muted/30 p-3">
                              {fields.length === 0 ? (
                                <div className="flex items-start gap-2.5 bg-[var(--accent)] border border-[var(--border)] rounded-[4px] px-3 py-2.5">
                                  <Info className="w-4 h-4 text-[var(--text-brand)] mt-0.5 shrink-0" />
                                  <MetaText as="p" tone="brand" className="leading-relaxed">
                                    该通道无需凭证配置（由租户在用户端完成扫码授权）。
                                  </MetaText>
                                </div>
                              ) : (
                                <div className="overflow-hidden rounded-[4px] border border-[var(--border)] bg-background">
                                  <div className="flex items-center justify-between gap-3 border-b border-[var(--border)] px-3 py-2">
                                    <MetaMedium>凭证信息</MetaMedium>
                                    {!isEditingThis && (
                                      <MetaText
                                        as="button"
                                        tone="brand"
                                        className="inline-flex items-center gap-1 hover:text-[var(--text-brand)]"
                                        onClick={() => startEditChannel(channel)}
                                      >
                                        <Pencil className="w-3 h-3" />
                                        编辑凭证
                                      </MetaText>
                                    )}
                                  </div>

                                  <div className="divide-y divide-[var(--border)]">
                                    {fields.map((field) => {
                                      const visible = isSecretVisible(channel.name, field.key);
                                      if (isEditingThis) {
                                        // 编辑态：Input + 密码可见切换
                                        return (
                                          <div key={field.key} className="px-3 py-2 space-y-1.5">
                                            <MetaMedium as="label" tone="secondary">{field.label}</MetaMedium>
                                            <div className="relative">
                                              <Input
                                                type={field.secret && !visible ? "password" : "text"}
                                                value={channelEditDraft![field.key] ?? ""}
                                                onChange={(e) => setChannelEditDraft(prev => ({ ...(prev ?? {}), [field.key]: e.target.value }))}
                                                className="bg-background border-[var(--border)] text-xs h-8 pr-9"
                                              />
                                              {field.secret && (
                                                <button
                                                  type="button"
                                                  onClick={() => toggleSecretVisibility(channel.name, field.key)}
                                                  className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-title)] transition-colors"
                                                >
                                                  {visible ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                                                </button>
                                              )}
                                            </div>
                                          </div>
                                        );
                                      }
                                      // 只读态：key - value（secret 自动 mask）
                                      const rawValue = channel.fieldValues[field.key] ?? "";
                                      const displayValue = field.secret && !visible ? maskSecret(rawValue) : (rawValue || "—");
                                      return (
                                        <div key={field.key} className="grid grid-cols-[112px_minmax(0,1fr)] items-center gap-3 px-3 py-2">
                                          <MetaText className="truncate" title={field.label}>{field.label}</MetaText>
                                          <div className="min-w-0 flex items-center gap-1.5">
                                            <CodeText tone="emphasis" className="min-w-0 break-all">{displayValue}</CodeText>
                                            {field.secret && rawValue && (
                                              <button
                                                type="button"
                                                onClick={() => toggleSecretVisibility(channel.name, field.key)}
                                                className="text-[var(--text-muted)] hover:text-[var(--text-title)] transition-colors shrink-0"
                                                title={visible ? "隐藏" : "查看"}
                                              >
                                                {visible ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                                              </button>
                                            )}
                                          </div>
                                        </div>
                                      );
                                    })}
                                  </div>

                                  {isEditingThis && (
                                    <div className="flex justify-end gap-2 border-t border-[var(--border)] px-3 py-2">
                                      <Button size="sm" variant="ghost" className="h-7 px-2 text-xs" onClick={cancelEditChannel}>
                                        取消
                                      </Button>
                                      <Button
                                        size="sm"
                                        variant="dialog-confirm"
                                        className="h-7 px-2 text-xs"
                                        onClick={() => saveEditChannel(channel)}
                                      >
                                        保存
                                      </Button>
                                    </div>
                                  )}
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      );
                    })}
                    {getClawDetail(selectedClaw.id).connectedChannels.length === 0 && !channelAdding && (
                      <MetaText as="div" tone="weak" className="px-4 py-6 bg-background rounded-[4px] border border-dashed border-[var(--border)] text-center">
                        暂未接入通道
                      </MetaText>
                    )}
                    {/* 新增通道面板 */}
                    {channelAdding && (() => {
                      const existing = new Set(getClawDetail(selectedClaw.id).connectedChannels.map(c => c.name));
                      const available = availableChannelOptions.filter(c => !existing.has(c.label));
                      const currentCh = availableChannelOptions.find(c => c.value === channelDraft);
                      const isWechatLike = currentCh?.wechatMode;
                      return (
                        <div className="bg-[var(--popover)] rounded-[4px] border border-[var(--border)] overflow-hidden">
                          <div className="bg-muted/30 p-3">
                            <div className="overflow-hidden rounded-[4px] border border-[var(--border)] bg-background">
                              <div className="border-b border-[var(--border)] px-3 py-2">
                                <MetaMedium>通道配置</MetaMedium>
                              </div>
                              <div className="divide-y divide-[var(--border)]">
                                {/* 通道选择 */}
                                <div className="px-3 py-2 space-y-1.5">
                                  <MetaMedium as="label" tone="secondary">通道类型</MetaMedium>
                                  <Select value={channelDraft} onValueChange={handleChannelDraftChange}>
                                    <SelectTrigger className="w-full bg-background border-[var(--border)] h-8 text-xs">
                                      <SelectValue placeholder="选择要添加的通道" />
                                    </SelectTrigger>
                                    <SelectContent>
                                      {available.length === 0 ? (
                                        <MetaText as="div" tone="weak" className="px-3 py-6 text-center">
                                          所有通道均已添加
                                        </MetaText>
                                      ) : (
                                        available.map((c) => (
                                          <SelectItem key={c.value} value={c.value}>{c.label}</SelectItem>
                                        ))
                                      )}
                                    </SelectContent>
                                  </Select>
                                </div>

                                {/* 无凭证字段的通道（微信）：提示框 */}
                                {currentCh && isWechatLike && (
                                  <div className="px-3 py-2">
                                    <div className="flex items-start gap-2.5 bg-[var(--accent)] border border-[var(--border)] rounded-[4px] px-3 py-2.5">
                                      <Info className="w-4 h-4 text-[var(--text-brand)] mt-0.5 shrink-0" />
                                      <MetaText as="p" tone="brand" className="leading-relaxed">
                                        微信通道通过扫码授权接入，管控端仅创建占位记录，实际扫码绑定由租户在用户端完成。
                                      </MetaText>
                                    </div>
                                  </div>
                                )}

                                {/* 凭证字段录入 */}
                                {currentCh && !isWechatLike && (currentCh.fields ?? []).length > 0 && (
                                  (currentCh.fields ?? []).map((field) => {
                                    const visible = isSecretVisible("__draft__", field.key);
                                    return (
                                      <div key={field.key} className="px-3 py-2 space-y-1.5">
                                        <MetaMedium as="label" tone="secondary">{field.label}</MetaMedium>
                                        <div className="relative">
                                          <Input
                                            type={field.secret && !visible ? "password" : "text"}
                                            value={channelDraftFields[field.key] ?? ""}
                                            onChange={(e) => setChannelDraftFields(prev => ({ ...prev, [field.key]: e.target.value }))}
                                            placeholder={field.label}
                                            className="bg-background border-[var(--border)] text-xs h-8 pr-9"
                                          />
                                          {field.secret && (
                                            <button
                                              type="button"
                                              onClick={() => toggleSecretVisibility("__draft__", field.key)}
                                              className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[var(--text-muted)] hover:text-[var(--text-title)] transition-colors"
                                            >
                                              {visible ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
                                            </button>
                                          )}
                                        </div>
                                      </div>
                                    );
                                  })
                                )}
                              </div>

                              <div className="flex justify-end gap-2 border-t border-[var(--border)] px-3 py-2">
                                <Button size="sm" variant="ghost" className="h-7 px-2 text-xs" onClick={cancelAddChannel}>
                                  取消
                                </Button>
                                <Button
                                  size="sm"
                                  variant="dialog-confirm"
                                  className="h-7 px-2 text-xs"
                                  onClick={confirmAddChannel}
                                  disabled={!channelDraft}
                                >
                                  确认添加
                                </Button>
                              </div>
                            </div>
                          </div>
                        </div>
                      );
                    })()}
                  </div>
                </div>
              );
              const blkSkills = (
                <div>
                  <MetaText as="div" className="mb-2">已安装技能（{getClawDetail(selectedClaw.id).installedSkills.length}）</MetaText>
                  {getClawDetail(selectedClaw.id).installedSkills.length === 0 ? (
                    <MetaText as="div" tone="weak" className="px-4 py-6 bg-background rounded-[4px] border border-dashed border-[var(--border)] text-center">
                      暂未安装技能
                    </MetaText>
                  ) : (
                    <div className="overflow-hidden rounded-[4px] border border-[var(--border)] bg-background">
                      <Table density="compact">
                        <TableHeader>
                          <TableRow>
                            <TableHead>技能名称</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {getClawDetail(selectedClaw.id).installedSkills.map((skill) => (
                            <TableRow key={skill}>
                              <TableCell>
                                <MiniBodyText>{skill}</MiniBodyText>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  )}
                </div>
              );
              const blkSpec = (() => {
                const specInfo = getInstanceSpecInfo(selectedClaw);
                const rUpdate = resourceUpdates[selectedClaw.id];
                const displaySpec = rUpdate?.spec ?? specInfo.spec;
                const displayDiskCapacity = rUpdate?.diskSize ?? specInfo.systemDiskCapacity;
                return (
                  <div>
                    <MetaText className="mb-2">云资源配置</MetaText>
                    <SurfaceInner className="px-4 py-3 divide-y divide-[var(--border)]">
                      <div className="flex items-center justify-between py-1.5">
                        <span className="text-xs text-[var(--text-muted)]">实例规格</span>
                        <span className="text-sm text-[var(--text-title)]">{formatSpecFull(displaySpec)}</span>
                      </div>
                      <div className="flex items-center justify-between py-1.5">
                        <span className="text-xs text-[var(--text-muted)]">系统盘</span>
                        <span className="text-sm text-[var(--text-title)]">{specInfo.systemDiskType} {displayDiskCapacity}GiB</span>
                      </div>
                      <div className="flex items-center justify-between py-1.5">
                        <span className="text-xs text-[var(--text-muted)]">计费模式</span>
                        <span className="text-sm text-[var(--text-title)]">{specInfo.chargeType}</span>
                      </div>
                      <div className="flex items-center justify-between py-1.5">
                        <span className="text-xs text-[var(--text-muted)]">公网 IP</span>
                        <span className="text-sm text-[var(--text-title)]">{specInfo.publicAssigned ? "分配" : "未分配"}</span>
                      </div>
                      <div className="flex items-center justify-between py-1.5">
                        <span className="text-xs text-[var(--text-muted)]">公网计费模式</span>
                        <span className="text-sm text-[var(--text-title)]">{specInfo.publicAssigned ? specInfo.publicChargeMode : "—"}</span>
                      </div>
                      <div className="flex items-center justify-between py-1.5">
                        <span className="text-xs text-[var(--text-muted)]">公网带宽上限</span>
                        <span className="text-sm text-[var(--text-title)] tabular-nums">{specInfo.publicAssigned ? specInfo.publicBandwidth : "—"}</span>
                      </div>
                    </SurfaceInner>
                  </div>
                );
              })();
              // ─── 龙虾医生区块 ───────────────────────────────────────────────
              // data-doctor-block: 供 AdminDoctorChatPanel 在 active 变 true 时
              // 通过 closest() 找到本区块根节点并 scrollIntoView。抽屉页面较长，
              // 龙虾医生位置偏下，点「开始诊断」后新展开的对话/交互面板容易被
              // 抽屉视口底部截断，需要用户手动下拉才能看全—— 通过该锚点自动
              // 把整个区块（含"龙虾医生"标题 + SurfaceInner + 新展开的Panel）
              // 滚到抽屉滚动容器顶部，无需人工下拉。
              const blkDoctor = (
                <div data-doctor-block="true">
                  <MetaText as="div" className="mb-2 font-medium">龙虾医生</MetaText>
                  <SurfaceInner className="relative px-4 py-3">
                    {/* 放大/展开按钮 —— SurfaceInner 内右上角，与提示文字首行顶对齐 */}
                    {doctorActive && (
                      <button
                        onClick={() => {
                          window.dispatchEvent(new CustomEvent("admin-doctor-toggle-expand"));
                        }}
                        className="absolute top-3 right-3 z-10 w-6 h-6 rounded-[var(--radius-lg)] flex items-center justify-center hover:bg-gray-100 transition-colors"
                        title="展开/放大"
                      >
                        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-gray-500">
                          <polyline points="15 3 21 3 21 9" /><polyline points="9 21 3 21 3 15" /><line x1="21" y1="3" x2="14" y2="10" /><line x1="3" y1="21" x2="10" y2="14" />
                        </svg>
                      </button>
                    )}

                    <MetaText as="div" tone="weak" className="leading-relaxed text-xs pr-8">
                      AI 智能诊断，帮助您发现并修复 Agent 运行问题。若诊断开始前已勾选「创建配置快照」，结束诊断后可选择配置回滚。
                    </MetaText>

                    {/* 开始诊断 / 结束诊断 —— 同级同样式 */}
                    {!doctorActive && (
                      <div className="mt-3">
                        <TooltipProvider>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="inline-block">
                                <Button
                                  variant="claw-primary"
                                  size="sm"
                                  className="scale-[0.7] origin-left"
                                  disabled={selectedClaw.status !== "running" || doctorOccupied}
                                  onClick={() => setDoctorActive(true)}
                                >
                                  开始诊断
                                </Button>
                              </span>
                            </TooltipTrigger>
                            {selectedClaw.status !== "running" && (
                              <TooltipContent>仅运行中的 Agent 可以诊断</TooltipContent>
                            )}
                            {selectedClaw.status === "running" && doctorOccupied && (
                              <TooltipContent>该 Agent 当前正在诊断中</TooltipContent>
                            )}
                          </Tooltip>
                        </TooltipProvider>
                      </div>
                    )}

                    {/* 内嵌对话面板 */}
                    <AdminDoctorChatPanel
                      active={doctorActive}
                      agentInstanceId={selectedClaw.instanceId}
                      onEnd={() => setDoctorActive(false)}
                    />
                  </SurfaceInner>
                </div>
              );

              const cloudBody = (
                <div className="p-4 space-y-6">
                  {blkIdentity}
                  {blkSpec}
                  {blkModels}
                  {blkChannels}
                  {blkSkills}
                  {blkDoctor}
                </div>
              );
              if (!compareMode) return cloudBody;
              const leftBlocks: ReactElement[] = [
                blkIdentity,
                blkSpec,
                blkModels,
                blkChannels,
                blkSkills,
                blkDoctor,
                ...renderInstanceExtraBlocks(selectedClaw),
              ];
              return renderCompareGrid(selectedClaw, leftBlocks);
              })()}
            </DrawerBody>
          </DrawerContent>
        )}
      </Drawer>

      {/* 移除通道二次确认 */}
      <AlertDialog open={!!channelRemoveTarget} onOpenChange={(open) => { if (!open) setChannelRemoveTarget(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认移除通道</AlertDialogTitle>
            <AlertDialogDescription>
              移除「{channelRemoveTarget}」后，该 Agent 将无法通过此通道收发消息。该操作不会删除通道下已有的凭证配置，可在用户端重新接入。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <Button variant="claw-outline" onClick={() => setChannelRemoveTarget(null)}>取消</Button>
            <AlertDialogAction variant="destructive" onClick={confirmRemoveChannel}>确认移除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 模型操作二次确认（设为主/删主/删备）—— 与用户端 OpenClawDetail 保持一致 */}
      <Dialog
        open={modelConfirmDialog.open}
        onOpenChange={(open) => !open && setModelConfirmDialog(prev => ({ ...prev, open: false }))}
      >
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-brand)]">
              {modelConfirmDialog.type === "delete"
                ? "确认删除主模型"
                : modelConfirmDialog.type === "delete-backup"
                ? "确认删除备选模型"
                : "切换主模型"}
            </DialogTitle>
            <DialogDescription className="text-[var(--text-muted)] leading-relaxed pt-1">
              {modelConfirmDialog.type === "delete"
                ? "删除后将自动切换备选模型作为主模型，切换过程中将导致相关的 Gateway 服务重启"
                : modelConfirmDialog.type === "delete-backup"
                ? "删除后将导致相关的 Gateway 服务重启，确认删除么"
                : "将此模型设为主模型后，原主模型将降为备选模型。切换过程中会自动重启 Gateway 服务，是否继续？"}
            </DialogDescription>
          </DialogHeader>
          <div className="flex justify-end gap-2 pt-2">
            <Button
              variant="claw-outline"
              size="sm"
              onClick={() => setModelConfirmDialog(prev => ({ ...prev, open: false }))}
            >
              取消
            </Button>
            <Button
              size="sm"
              variant="dialog-confirm"
              onClick={runModelConfirm}
            >
              {modelConfirmDialog.type === "delete" || modelConfirmDialog.type === "delete-backup"
                ? "确认删除"
                : "确认设置"}
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* 监控抽屉 */}
      <Drawer
        open={showMonitorDrawer && !!selectedClaw}
        onOpenChange={(open) => setShowMonitorDrawer(open)}
        direction="right"
      >
        {selectedClaw && (
          <DrawerContent className="data-[vaul-drawer-direction=right]:w-[640px] data-[vaul-drawer-direction=right]:sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-[var(--popover)] p-0">
            <DrawerHeader className="flex flex-row items-center justify-between gap-4 border-b border-[var(--border)] bg-[var(--popover)] p-4 text-left">
              <DrawerTitle asChild>
                <PanelTitle as="h2" className="truncate">{selectedClaw.name} - 监控</PanelTitle>
              </DrawerTitle>
              <DrawerClose asChild>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-7 w-7 p-0 text-[var(--text-title)] hover:text-[var(--cp-brand-black)]"
                  aria-label="关闭"
                >
                  <X className="w-4 h-4" />
                </Button>
              </DrawerClose>
            </DrawerHeader>

            <DrawerBody>
              <div className="space-y-6 p-6">
                <section className="space-y-4">
                  <PanelTitle as="h3">Tokens 分析</PanelTitle>
                  <div className="grid grid-cols-3 gap-3">
                    <NumberCard
                      className="p-4"
                      label="输入 Tokens"
                      value="1,234"
                      icon={<ArrowUp className="h-[18px] w-[18px] text-[var(--text-brand)]" />}
                    />
                    <NumberCard
                      className="p-4"
                      label="输出 Tokens"
                      value="5,678"
                      icon={<ArrowDown className="h-[18px] w-[18px] text-[var(--text-brand)]" />}
                    />
                    <NumberCard
                      className="p-4"
                      label="总 Tokens"
                      value="6,912"
                      icon={<Zap className="h-[18px] w-[18px] text-[var(--text-brand)]" />}
                    />
                  </div>
                  <Button
                    variant="link"
                    className="h-auto p-0"
                    onClick={() => setLocation('/admin/tokens-monitor')}
                  >
                    查看完整 Tokens 监控 <ExternalLink className="w-3.5 h-3.5" />
                  </Button>
                </section>

                {clsEnabled && <div className="border-t border-[var(--border)]" />}

                {clsEnabled && (
                  <section className="space-y-4">
                    <PanelTitle as="h3">会话记录</PanelTitle>
                    <div className="grid grid-cols-2 gap-3">
                      <NumberCard
                        className="p-4"
                        label="总会话数"
                        value="42"
                        icon={<MessageCircle className="h-[18px] w-[18px] text-[var(--text-brand)]" />}
                      />
                      <NumberCard
                        className="p-4"
                        label="平均轮次"
                        value="8.5"
                        icon={<RotateCw className="h-[18px] w-[18px] text-[var(--text-brand)]" />}
                      />
                    </div>

                    <SurfaceInner className="overflow-hidden p-0">
                      <Table density="compact" autoFixedColumns={false}>
                        <TableHeader>
                          <TableRow>
                            <TableHead className="w-[30%]">会话</TableHead>
                            <TableHead className="w-[13%]">类型</TableHead>
                            <TableHead className="w-[28%]">模型</TableHead>
                            <TableHead className="w-[24%]">最新时间</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {[
                            { id: "c3b2ac3c", type: "Feishu Dm", model: "hunyuan-turbos-latest", time: "2026-03-09 17:49" },
                            { id: "81c87c7b", type: "QQ Dm", model: "hunyuan-turbos-latest", time: "2026-03-09 10:07" },
                            { id: "267e462d", type: "CLI", model: "deepseek-v3.2", time: "2026-03-08 12:54" },
                          ].map((session) => (
                            <TableRow key={session.id}>
                              <TableCell className="truncate"><CodeText>{session.id}</CodeText></TableCell>
                              <TableCell className="truncate"><MetaText as="span" tone="secondary">{session.type}</MetaText></TableCell>
                              <TableCell className="truncate"><MetaText as="span" tone="secondary">{session.model}</MetaText></TableCell>
                              <TableCell className="truncate"><MetaText as="span" tone="secondary">{session.time}</MetaText></TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </SurfaceInner>

                    <Button
                      variant="link"
                      className="h-auto p-0"
                      onClick={() => setLocation('/admin/session-management')}
                    >
                      查看完整会话管理 <ExternalLink className="w-3.5 h-3.5" />
                    </Button>
                  </section>
                )}
              </div>
            </DrawerBody>
          </DrawerContent>
        )}
      </Drawer>

      <style>{`
        @keyframes breathing {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.3; }
        }
        .animate-breathing {
          animation: breathing 2s ease-in-out infinite;
        }
      `}</style>

      {/* 命令下发弹窗（取代旧抽屉）：
        * - 从工具栏「命令下发」主按钮触发，预填已选实例
        * - 用户在弹窗内选择命令模板 → 选执行策略 → 提交
        */}
      <DispatchCommandDialog
        open={dispatchPresetIds !== null}
        onOpenChange={(v) => !v && setDispatchPresetIds(null)}
        command={null}
        presetInstanceIds={dispatchPresetIds ?? undefined}
        onDispatched={() => {
          // 下发成功后清空选中状态，便于用户继续操作
          setSelectedIds(new Set());
        }}
      />

      {/* 创建 Agent 两步弹窗（步骤 1 必填、步骤 2 可选；同步用户平台策略决定是否展示模型/通道/技能） */}
      <CreateAgentDialog
        open={showCreateAgent}
        onOpenChange={setShowCreateAgent}
        onCreated={(result: CreateAgentResult) => {
          // 1) 把管控端创建结果落地为用户端 AgentItem，写入共享 store（localStorage key=openclaw_list）
          //    关键字段：creator = 选中的 userId（邮箱），保证该 Agent 在该用户的「我的 Agent」页面可见
          //    字段口径与用户端 MyOpenClaw.handleCreate 完全对齐（agentType / version / status:"creating" / memoryStatus:'none' / shareScope:"private" 等）
          const ts = Date.now();
          const DEFAULT_VERSION: Record<string, string> = {
            openclaw: "2026.4.23",
            hermes: "0.12.0",
            lightclawace: "0.1.8",
            localagent: "0.1.0",
          };
          // 取第二步第一个模型作为主模型（未配置则留空，与用户端创建一致）
          // 新 DraftModel schema：{ modelConfigId, providerLabel, versionLabel }
          //   - AgentItem.model        ← 厂商/「自定义模型」      (providerLabel)
          //   - AgentItem.modelVersion ← 版本/自定义模型名         (versionLabel)
          const primaryModel = result.models[0];
          const channelLabels: string[] = result.channels.map((c) => c.label);
          const newItem: AgentItem = {
            id: `oc-${ts}`,
            instanceId: `ins-${ts.toString(36).slice(-8)}`,
            name: result.name,
            status: "creating",
            agentType: result.agentType,
            createdAt: new Date().toLocaleString("zh-CN"),
            model: primaryModel?.providerLabel ?? "",
            modelVersion: primaryModel?.versionLabel ?? "",
            channels: channelLabels,
            skills: result.skills,
            roleName: result.roleName,
            memoryStatus: "none",
            groupId: result.groupId,
            groupName: result.groupName,
            // 管控端代建的 Agent 默认归属给目标用户本人，使用 private 共享范围
            shareScope: "private",
            shareGroupIds: [],
            shareGroupNames: [],
            shareUserIds: [],
            shareUserNames: [],
          };
          // AgentItem schema 没有 creator / version 字段（用户端 OpenClawItem 才有），但 localStorage 实际不做 schema 校验：
          //   - creator：Agent 的归属用户 ID = 弹窗里选中的目标用户
          //             （新口径下，管控端代建也直接把 creator 写成目标用户，不再区分 owner/creator）
          //   - version：用户端版本对比需要
          const persisted = {
            ...newItem,
            creator: result.userId,
            version: DEFAULT_VERSION[result.agentType] ?? "2026.4.23",
            // Agent 所属组织 = 管理员在弹窗中为其选定的分组（固定绑定，不随用户组织变化）
            groupId: result.groupId,
            groupName: result.groupName,
          };
          const list = loadClawList();
          saveClawList([persisted as AgentItem, ...list]);
          notifyClawListChange();

          // 2) 同步落地到管控端缓存（localStorage key=admin_created_agents），让本页表格立刻看到这条
          //    映射要点：用户端 agentType 是 lowercase（openclaw/hermes/…），管控端 Claw.agentType 是 PascalCase
          const AGENT_TYPE_TO_PASCAL: Record<string, AdminCreatedAgent["agentType"]> = {
            openclaw: "OpenClaw",
            hermes: "Hermes",
            lightclawace: "LightclawACE",
            localagent: "LocalAgent",
          };
          const adminAgent: AdminCreatedAgent = {
            id: persisted.id,
            instanceId: persisted.instanceId,
            name: persisted.name,
            // creator = 该 Agent 的归属用户 ID = 弹窗里选中的目标用户
            creator: result.userId,
            // Claw.createTime 期望 "YYYY-MM-DD HH:mm:ss" 字符串（参考 MOCK_CLAWS）；
            // 用 ISO 改写一下，避免本地 toLocaleString 受时区/中文符号影响导致排序异常
            createTime: new Date(ts).toISOString().replace("T", " ").slice(0, 19),
            status: "creating",
            version: DEFAULT_VERSION[result.agentType] ?? "2026.4.23",
            agentType: AGENT_TYPE_TO_PASCAL[result.agentType] ?? "OpenClaw",
            pluginVersions: {
              wechat: "3.2.1",
              dingtalk: "2.8.0",
              feishu: "1.5.3",
              wecom: "2.1.4",
              qq: "1.0.2",
            },
            // 组织：Agent 自身固定绑定的分组（管理员创建时选的组）
            groupId: result.groupId,
            groupName: result.groupName,
            // department* 沿用同一分组做部门展示回退（如需区分再拆）
            department: result.groupName,
            departmentId: result.groupId,
          };
          appendAdminCreatedAgent(adminAgent);
          notifyAdminCreatedAgentsChange();

          // 3) 清空当前选中并回到第一页，便于刷新观察
          setSelectedIds(new Set());
          setPage(1);
        }}
      />

      {/* 版本更新记录侧边栏（点击新版本推送提醒打开，默认开启「仅看可推送新版本」） */}
      <UpdateRecordsDrawerForAgentList
        open={showUpdateRecordsDrawer}
        onOpenChange={setShowUpdateRecordsDrawer}
      />


    </TooltipProvider>
  );
}

// ─── 工具栏更新提醒入口（↑ 图标 + Popover 开关 + 版本记录入口） ─────
function ImageUpdateBellEntry({ onClick }: { onClick: () => void }) {
  const outdated = useOutdatedTypes();
  const [open, setOpen] = useState(false);
  const [reminders, setReminders] = useState<Record<string, boolean>>(() => {
    try {
      const raw = localStorage.getItem("admin_update_reminder_v1");
      return raw ? JSON.parse(raw) : {};
    } catch { return {}; }
  });

  // 订阅其他页面（如 Agent 类型页）对提醒开关的修改
  useEffect(() => {
    const handler = () => {
      try {
        const raw = localStorage.getItem("admin_update_reminder_v1");
        setReminders(raw ? JSON.parse(raw) : {});
      } catch { setReminders({}); }
    };
    window.addEventListener("update-reminder-changed", handler);
    window.addEventListener("storage", handler);
    return () => {
      window.removeEventListener("update-reminder-changed", handler);
      window.removeEventListener("storage", handler);
    };
  }, []);

  const isEnabled = (agentType: string) => reminders[agentType] === true;
  const handleToggle = (agentType: string, checked: boolean) => {
    const next = { ...reminders };
    if (checked) next[agentType] = true;
    else delete next[agentType];
    setReminders(next);
    try {
      localStorage.setItem("admin_update_reminder_v1", JSON.stringify(next));
      window.dispatchEvent(new Event("update-reminder-changed"));
    } catch { /* ignore */ }
  };

  const hasDot = outdated.some((o) => isEnabled(o.agentType));

  const handleViewRecords = () => {
    setOpen(false);
    setTimeout(() => onClick(), 100);
  };

  return (
    <Popover open={open} onOpenChange={setOpen} modal={false}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="relative inline-flex items-center justify-center w-8 h-8 rounded-[4px] hover:bg-[var(--accent)] transition-colors cursor-pointer"
          aria-label="更新提醒状态"
        >
          <ArrowUpCircle className={cn("w-4 h-4", hasDot ? "text-[var(--text-brand)]" : "text-[var(--text-weak)]")} />
          {hasDot && (
            <span className="absolute top-1 right-1 w-2 h-2 bg-[var(--text-danger)] rounded-full ring-2 ring-white" />
          )}
        </button>
      </PopoverTrigger>
      <PopoverContent align="end" sideOffset={8} className="w-[360px] p-0">
        <div className="px-5 py-4 border-b border-gray-200">
          <div className="flex items-center gap-2 text-[14px] font-semibold text-[#020617]">
            <ArrowUpCircle className="w-4 h-4 text-[#1447E6]" />
            更新提醒
          </div>
          <p className="text-[12px] text-[#737373] mt-1 leading-relaxed">
            开启后，该 Agent 类型下运行旧版本的实例将在用户端收到升级提示，建议更新至当前生效镜像版本。
          </p>
        </div>

        {outdated.length === 0 ? (
          <div className="py-10 text-center text-[13px] text-[#a3a3a3]">
            所有实例已是最新版本
          </div>
        ) : (
          <ul className="max-h-[320px] overflow-y-auto divide-y divide-gray-200">
            {outdated.map((item) => {
              const enabled = isEnabled(item.agentType);
              return (
                <li key={item.agentType} className="px-5 py-3 flex items-center justify-between gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-[14px] font-medium text-[#020617]">
                        {item.agentTypeLabel}
                      </span>
                      <span className="text-[12px] text-[var(--text-brand)] tabular-nums">
                        升级至 v{item.enabledVersion}
                      </span>
                    </div>
                    <p className="text-[12px] text-[#737373] mt-0.5">
                      {item.outdatedCount} 个实例运行旧版本
                    </p>
                  </div>
                  <Switch
                    checked={enabled}
                    onCheckedChange={(checked) => handleToggle(item.agentType, checked)}
                  />
                </li>
              );
            })}
          </ul>
        )}

        <div className="px-5 py-3 border-t border-gray-200 space-y-2">
          <p className="text-[11px] text-[var(--text-muted)] text-center">
            提醒开关可在 Agent 类型页面统一管理
          </p>
          <Button
            variant="claw-outline"
            size="claw-sm"
            className="w-full justify-center gap-1.5 text-[12px]"
            onClick={handleViewRecords}
          >
            查看版本更新记录
            <ChevronRight className="w-3.5 h-3.5" />
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

// ─── 版本更新记录侧边栏 ─────
function UpdateRecordsDrawerForAgentList({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <UpdateRecordsDrawer
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}
