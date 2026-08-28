/**
 * ImageManagement - 管控端 Agent 类型 / 镜像管理页（扁平表格 + 行内展开版）
 *
 * 心智路径：
 *   1. 进入页面 → 看到所有 Agent 类型表格
 *   2. 每行 = 一个 Agent 类型（含当前生效镜像信息 + 应用范围）
 *   3. 系统预设类型（OpenClaw / Hermes / LightClaw ACE）固有，自定义类型可加可删
 *   4. 「用户可见 Switch」决定该类型对用户开放
 *   5. 「应用范围」决定该类型对哪些用户/组织可见（沿用 OneID 群组三态）
 *   6. 「查看镜像」展开行内二级 Tab：公共镜像候选 / 已导入自定义镜像
 *
 * 关键约束：
 *   - 同 Agent 类型下只能有一个镜像启用（active=true）
 *   - 删除自定义类型前必须先清空类型下的镜像 + 不能是用户端首选类型
 *   - 公共镜像不可删除 / 不可编辑
 *   - 创建 native（自研）类型必须勾选确认"允许用户进入终端"
 */
import { useEffect, useMemo, useState } from "react";
import { Alert, AlertDescription, AlertTitle, AlertInfoIcon } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Card } from "@/components/ui/card";
import { RadioCard } from "@/components/ui/radio-card";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { cn } from "@/lib/utils";
import { StatusTag } from "@/components/ui/status-tag";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { toast } from "sonner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  RefreshCw,
  ExternalLink,
  Plus,
  Sparkles,
  Star,
  Wrench,
  Check,
  X as XIcon,
  CircleAlert,
  ChevronDown,
  ChevronRight,
  Pencil,
  Users,
  History,
  Minus,
} from "lucide-react";

import { buildGroupTree, type GroupTreeNode } from "./MemberManagement/health";

import AgentTypesTable, {
  type AgentTypeRowData,
} from "./ImageManagement/AgentTypesTable";
import UpdateRecordsDrawer, {
  type DrawerInitialFilter,
} from "./ImageManagement/UpdateRecordsDrawer";
import PushUpgradeDialog, {
  type PushableAgentType,
} from "./ImageManagement/PushUpgradeDialog";
import { loadCustomTypes, saveCustomTypes, nowStr } from "./ImageManagement/customAgentStore";
import type { CustomAgentType, KernelBase } from "./ImageManagement/types";
import {
  AGENT_VERSIONS,
  getImageMeta,
} from "./VersionManagement/mockData";
import {
  deriveAgentTypeView,
  IMG_TO_VERSION_KEY,
  getHardenedImageId,
  type ImageRow,
} from "./ImageManagement/deriveAgentTypeView";
import {
  pruneOnVersionChange,
  listActivePushes,
  setActivePush,
  clearActivePush,
  type ActivePush,
} from "@/lib/upgradePushStore";
import { isReminderEnabled } from "@/lib/updateReminderStore";
import {
  getVersionMode,
  getEffectiveVersion,
} from "@/lib/versionModeStore";
import type { UserGroup } from "./MemberManagement/types";
import { MOCK_GROUPS as MOCK_ONEID_GROUPS, MOCK_MANUAL_GROUPS, MOCK_USERS, MOCK_USERS_MANUAL } from "./MemberManagement/mock";
import { ScopeSelect } from "@/components/ScopeSelect";
import { AdminPageHeader } from "@/components/ui/admin-page-header";

// ─── 系统预设 Agent 类型 ────────────────────────────────────────────────
interface AgentTypeConfig {
  value: string;
  label: string;
  isSystem: boolean;
  versionPlaceholder: string;
  versionRegex: RegExp | null;
}

const SYSTEM_AGENT_TYPES: AgentTypeConfig[] = [
  { value: "OpenClaw",     label: "OpenClaw",      isSystem: true, versionPlaceholder: "如 2026.4.2", versionRegex: /^\d{4}\.\d{1,2}\.\d{1,2}$/ },
  { value: "HermesAgent",  label: "Hermes Agent",  isSystem: true, versionPlaceholder: "如 0.8.0",    versionRegex: /^\d+\.\d+\.\d+$/ },
  { value: "LightClawACE", label: "LightClaw ACE", isSystem: true, versionPlaceholder: "如 1.0.2",    versionRegex: /^\d+\.\d+\.\d+$/ },
];

const DEFAULT_AGENT_TYPE = "OpenClaw";

// 自定义内核（native）创建提醒文案
const NATIVE_KERNEL_NOTICE_TITLE = "管控台部分功能在该类型上不可用";
const NATIVE_KERNEL_NOTICE_LINES = [
  "1. 用户端需要登录\"终端\"配置模型/通道/技能，不支持管控台快捷配置；",
  "2. 管控端部分功能不可用：如 Agent 工具库、记忆管理、网盘管理、运维观测、AI Agent 安全、会话管理 等功能将不可用",
];

function validateVersion(config: AgentTypeConfig, version: string): boolean {
  if (!config.versionRegex) return version.trim().length > 0;
  if (!config.versionRegex.test(version)) return false;
  if (config.value === "OpenClaw") {
    const [y, m, d] = version.split(".").map(Number);
    const date = new Date(y, m - 1, d);
    return date.getFullYear() === y && date.getMonth() === m - 1 && date.getDate() === d;
  }
  return true;
}

// 兼容内核选项（创建自定义类型时用）
interface KernelOption {
  value: KernelBase;
  title: string;
  description: string;
}
const KERNEL_OPTIONS: KernelOption[] = [
  { value: "OpenClaw",        title: "兼容 OpenClaw",        description: "该类型与 OpenClaw 完全兼容，管控台功能与 OpenClaw 保持一致" },
  { value: "HermesAgent",     title: "兼容 Hermes Agent",    description: "该类型与 Hermes Agent 完全兼容，管控台功能与 Hermes Agent 保持一致" },
  { value: "LightClawACE",    title: "兼容 LightClaw ACE",   description: "该类型与 LightClaw ACE 完全兼容，管控台功能与 LightClaw ACE 保持一致" },
  { value: "DeepSeekHarness", title: "兼容 DeepSeek Harness", description: "该类型与 DeepSeek Harness 完全兼容，管控台功能与 DeepSeek Harness 保持一致" },
  { value: "native",          title: "自定义内核 / 不兼容上述已知类型", description: "完全自研，与上述已知 Agent 内核均不兼容；部分管控台功能将不可用，详见下方说明" },
];

const kernelBaseLabel = (kb: KernelBase): string | undefined => {
  if (kb === "OpenClaw") return "OpenClaw";
  if (kb === "HermesAgent") return "Hermes Agent";
  if (kb === "LightClawACE") return "LightClaw ACE";
  if (kb === "DeepSeekHarness") return "DeepSeek Harness";
  return undefined;
};

/**
 * 仅作为自定义类型"兼容内核"回退配置的内核基线（不进 SYSTEM_AGENT_TYPES，
 * 不会在类型表中产生独立的系统类型行）。DeepSeek Harness 即此类：
 * 可被自定义类型选中作为兼容基线，提供版本格式校验与提示。
 */
const KERNEL_BASE_FALLBACK_CONFIGS: Record<string, AgentTypeConfig> = {
  DeepSeekHarness: {
    value: "DeepSeekHarness",
    label: "DeepSeek Harness",
    isSystem: false,
    versionPlaceholder: "如 1.0.0",
    versionRegex: /^\d+\.\d+\.\d+$/,
  },
};

// ─── Mock 镜像数据 ─────────────────────────────────────────────────────
// ─── 公共镜像 mock：使用真实 ID（参考产研发版记录）──────────────────────
// 每条 agentVersion 取该镜像最新更新记录中的版本号
const PUBLIC_IMAGES: { id: string; name: string; agentType: string; agentVersion: string; os: string; activeByDefault: boolean }[] = [
  // OpenClaw 系列（4 个）
  { id: "img-idzg74s9", name: "OpenClaw on Ubuntu 24.04",                          agentType: "OpenClaw",     agentVersion: "2026.4.23", os: "Ubuntu 24.04 x86_64",          activeByDefault: true  },
  { id: "img-nmg7pw1r", name: "OpenClaw on TencentOS Server 4",                    agentType: "OpenClaw",     agentVersion: "2026.4.23", os: "TencentOS Server 4 x86_64",    activeByDefault: false },
  { id: "img-pf18atu9", name: "OpenClaw on TencentOS Server 4 For Tencent",        agentType: "OpenClaw",     agentVersion: "2026.4.23", os: "TencentOS Server 4 (TKernel5)", activeByDefault: false },
  // Hermes Agent 系列（2 个）
  { id: "img-al484uhr", name: "Hermes Agent on Ubuntu 24.04",                      agentType: "HermesAgent",  agentVersion: "0.12.0",    os: "Ubuntu 24.04 x86_64",          activeByDefault: true  },
  { id: "img-ppz9gfjn", name: "Hermes Agent on TencentOS Server 4",                agentType: "HermesAgent",  agentVersion: "0.12.0",    os: "TencentOS Server 4 x86_64",    activeByDefault: false },
  // LightClaw ACE（1 个）
  { id: "img-0dvlda3b", name: "LightClaw ACE on TencentOS Server 4",               agentType: "LightClawACE", agentVersion: "0.1.8",     os: "TencentOS Server 4 x86_64",    activeByDefault: true  },
];

const PUBLIC_IMAGE_ROWS: ImageRow[] = PUBLIC_IMAGES.map((p) => ({
  id: p.id, name: p.name, status: "available", type: "public",
  agentType: p.agentType, agentVersion: p.agentVersion, os: p.os,
  createTime: "2026-04-30 17:11:52", active: p.activeByDefault,
}));

const CUSTOM_IMAGES = [
  { id: "img-cust00001", name: "openclaw-custom-v1.0" },
  { id: "img-cm9xkd24",  name: "hermes-custom-v0.7" },
  { id: "img-q8r3s7t0",  name: "lightclaw-prod-2025Q4" },
  { id: "img-m3n4o5p6",  name: "openclaw-dev-latest" },
  { id: "img-a7v2zk0w",  name: "agent-test-v2.0" },
];

const MOCK_IMAGES: ImageRow[] = [
  ...PUBLIC_IMAGE_ROWS,
  // 演示用自定义镜像
  { id: "img-cust00001", name: "openclaw-custom-v1.0",   status: "available", type: "custom", agentType: "OpenClaw", agentVersion: "2025.9.1", os: "CentOS 7.9 64位", createTime: "2025-09-15 14:22:35", active: false },
  { id: "img-cust00002", name: "legacy-image-v1",        status: "available", type: "custom", agentType: "OpenClaw", agentVersion: "",         os: "CentOS 7.9 64位", createTime: "2025-06-01 12:00:00", active: false },
];

function normalizeImages(imgs: ImageRow[]): ImageRow[] {
  return imgs.map((img) => (img.agentType ? img : { ...img, agentType: "OpenClaw" }));
}

// ─── 应用范围（按 Agent 类型维度） ───────────────────────────────────────
interface ImageScopeData {
  visibilityScope: "all" | "groups";
  visibilityGroupIds: string[];
}

const ALL_GROUPS: UserGroup[] = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];
/** 「全部用户」对应的用户总数（来自用户管理列表的 mock 数据） */
const TOTAL_USER_COUNT = MOCK_USERS.length + MOCK_USERS_MANUAL.length;

function getGroupPath(groupId: string, groups: UserGroup[]): string {
  const map = new Map(groups.map((g) => [g.id, g]));
  const chain: string[] = [];
  let cur = map.get(groupId);
  while (cur) {
    chain.unshift(cur.name);
    cur = cur.parentId ? map.get(cur.parentId) : undefined;
  }
  return chain.join("/");
}

type CheckState = "checked" | "unchecked" | "indeterminate";

function getCheckState(node: GroupTreeNode, selectedIds: Set<string>): CheckState {
  if (selectedIds.has(node.id)) return "checked";
  if (node.children.length === 0) return "unchecked";
  let hasChecked = false;
  let hasUnchecked = false;
  for (const c of node.children) {
    const s = getCheckState(c, selectedIds);
    if (s === "checked") hasChecked = true;
    else if (s === "unchecked") hasUnchecked = true;
    else { hasChecked = true; hasUnchecked = true; }
    if (hasChecked && hasUnchecked) return "indeterminate";
  }
  if (hasChecked && !hasUnchecked) return "checked";
  if (!hasChecked && hasUnchecked) return "unchecked";
  return "indeterminate";
}

function getDescendantIds(node: GroupTreeNode): string[] {
  const ids: string[] = [node.id];
  node.children.forEach((c) => ids.push(...getDescendantIds(c)));
  return ids;
}

function ImageScopePopover({
  scopeData,
  groups,
  onSave,
}: {
  scopeData: ImageScopeData;
  groups: UserGroup[];
  onSave: (scope: "all" | "groups", groupIds: string[]) => void;
}) {
  const [open, setOpen] = useState(false);
  const [draftScope, setDraftScope] = useState<"all" | "groups">(scopeData.visibilityScope);
  const [draftGroupIds, setDraftGroupIds] = useState<string[]>(scopeData.visibilityGroupIds);
  const [searchQuery, setSearchQuery] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  type DisplayBucket = "dept" | "custom";
  const groupsByBucket = useMemo(() => {
    const buckets: Record<DisplayBucket, UserGroup[]> = { dept: [], custom: [] };
    groups.forEach((g) => { if (g.source === "oneid-dept") buckets.dept.push(g); else buckets.custom.push(g); });
    return buckets;
  }, [groups]);

  const activeBuckets = useMemo(() => {
    const order: DisplayBucket[] = ["dept", "custom"];
    return order.filter((b) => groupsByBucket[b].length > 0);
  }, [groupsByBucket]);

  const BUCKET_LABELS: Record<DisplayBucket, string> = { dept: "部门", custom: "自定义组织" };

  const treesMap = useMemo(() => {
    const map: Record<string, GroupTreeNode[]> = {};
    activeBuckets.forEach((b) => { map[b] = buildGroupTree(groupsByBucket[b]); });
    return map;
  }, [activeBuckets, groupsByBucket]);

  const hasGroups = activeBuckets.length > 0;

  const handleOpenChange = (v: boolean) => {
    if (v) {
      setDraftScope(scopeData.visibilityScope);
      setDraftGroupIds([...scopeData.visibilityGroupIds]);
      setSearchQuery("");
      const expandSet = new Set<string>();
      const groupMap = new Map(groups.map((g) => [g.id, g]));
      scopeData.visibilityGroupIds.forEach((gid) => {
        let cur = groupMap.get(gid);
        while (cur && cur.parentId) { expandSet.add(cur.parentId); cur = groupMap.get(cur.parentId); }
      });
      activeBuckets.forEach((b) => { treesMap[b]?.forEach((root) => expandSet.add(root.id)); });
      setExpanded(expandSet);
    }
    setOpen(v);
  };

  const toggleExpand = (id: string) =>
    setExpanded((prev) => { const next = new Set(prev); if (next.has(id)) next.delete(id); else next.add(id); return next; });

  const toggleNode = (node: GroupTreeNode) => {
    const ids = new Set(draftGroupIds);
    const state = getCheckState(node, ids);
    const descendants = getDescendantIds(node);
    if (state === "checked") { descendants.forEach((d) => ids.delete(d)); }
    else { descendants.forEach((d) => ids.add(d)); }
    setDraftGroupIds(Array.from(ids));
  };

  const handleClearSelection = () => { setDraftGroupIds([]); setSearchQuery(""); };

  const isConfirmDisabled = draftScope === "groups" && draftGroupIds.length === 0;

  const handleConfirm = () => {
    if (isConfirmDisabled) return;
    onSave(draftScope, draftScope === "all" ? [] : draftGroupIds);
    setOpen(false);
    toast.success("应用范围已更新");
  };

  const matchedGroupIds = useMemo(() => {
    if (!searchQuery.trim()) return null;
    const q = searchQuery.toLowerCase();
    return new Set(
      groups
        .filter((g) => g.name.toLowerCase().includes(q) || getGroupPath(g.id, groups).toLowerCase().includes(q))
        .map((g) => g.id),
    );
  }, [searchQuery, groups]);

  const isNodeVisible = (node: GroupTreeNode): boolean => {
    if (!matchedGroupIds) return true;
    if (matchedGroupIds.has(node.id)) return true;
    return node.children.some(isNodeVisible);
  };

  const renderTreeNode = (node: GroupTreeNode, depth: number) => {
    if (!isNodeVisible(node)) return null;
    const selectedSet = new Set(draftGroupIds);
    const checkState = getCheckState(node, selectedSet);
    const isExpanded = expanded.has(node.id);
    const hasChildren = node.children.length > 0;
    return (
      <div key={node.id}>
        <button
          onClick={() => toggleNode(node)}
          className="w-full flex items-center gap-1.5 px-2 py-1.5 rounded-[3px] hover:bg-gray-50 transition-colors text-left"
          style={{ paddingLeft: 8 + depth * 16 }}
        >
          {hasChildren ? (
            <span
              onClick={(e) => { e.stopPropagation(); toggleExpand(node.id); }}
              className="w-4 h-4 flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-muted)] shrink-0 cursor-pointer"
            >
              {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            </span>
          ) : (
            <span className="w-4 h-4 shrink-0" />
          )}
          <span
            className={`w-3.5 h-3.5 rounded border shrink-0 flex items-center justify-center transition-colors ${
              checkState === "checked"
                ? "bg-blue-500 border-blue-500"
                : checkState === "indeterminate"
                  ? "bg-blue-500 border-blue-500"
                  : "border-gray-300 bg-white"
            }`}
          >
            {checkState === "checked" && <Check className="w-2.5 h-2.5 text-white" />}
            {checkState === "indeterminate" && <Minus className="w-2.5 h-2.5 text-white" />}
          </span>
          <span className="text-xs text-[var(--text-secondary)] truncate">{node.name}</span>
        </button>
        {hasChildren && isExpanded && node.children.map((c) => renderTreeNode(c, depth + 1))}
      </div>
    );
  };

  const selectedTags = useMemo(() => {
    const selectedSet = new Set(draftGroupIds);
    const collectEffective = (nodes: GroupTreeNode[]): string[] => {
      const result: string[] = [];
      for (const node of nodes) {
        const state = getCheckState(node, selectedSet);
        if (state === "checked") { result.push(node.id); }
        else if (state === "indeterminate") { result.push(...collectEffective(node.children)); }
      }
      return result;
    };
    const effectiveIds: string[] = [];
    activeBuckets.forEach((b) => { effectiveIds.push(...collectEffective(treesMap[b] || [])); });
    return effectiveIds.map((gid) => ({ id: gid, path: getGroupPath(gid, groups) }));
  }, [draftGroupIds, groups, activeBuckets, treesMap]);

  const selectedGroupPaths = useMemo(() => {
    const selectedSet = new Set(scopeData.visibilityGroupIds);
    const collectEffective = (nodes: GroupTreeNode[]): string[] => {
      const result: string[] = [];
      for (const node of nodes) {
        const state = getCheckState(node, selectedSet);
        if (state === "checked") { result.push(node.id); }
        else if (state === "indeterminate") { result.push(...collectEffective(node.children)); }
      }
      return result;
    };
    const effectiveIds: string[] = [];
    activeBuckets.forEach((b) => { effectiveIds.push(...collectEffective(treesMap[b] || [])); });
    return effectiveIds.map((gid) => getGroupPath(gid, groups));
  }, [groups, scopeData.visibilityGroupIds, activeBuckets, treesMap]);

  const renderScopeText = () => {
    if (scopeData.visibilityScope === "all" || selectedGroupPaths.length === 0) {
      return <Badge variant="outline">全部用户</Badge>;
    }
    const firstName = selectedGroupPaths[0];
    const rest = selectedGroupPaths.length - 1;
    const tooltipText = selectedGroupPaths.join("\n");
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex max-w-full items-center gap-1 cursor-default">
            <Badge variant="secondary" className="max-w-[140px]">
              <span className="block truncate max-w-[124px]">{firstName}</span>
            </Badge>
            {rest > 0 && (
              <Badge variant="secondary">+{rest}</Badge>
            )}
          </span>
        </TooltipTrigger>
        <TooltipContent className="max-w-[320px] text-xs leading-relaxed whitespace-pre-line">
          {tooltipText}
        </TooltipContent>
      </Tooltip>
    );
  };

  return (
    <div className="inline-flex items-center gap-1.5 min-h-[20px] max-w-[160px]">
      {renderScopeText()}
      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
          <button className="self-center text-[var(--text-weak)] hover:text-[#355EF1] transition-colors" title="编辑应用范围">
            <Pencil className="w-3 h-3" />
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-72 p-0 flex flex-col max-h-[420px]" align="start" sideOffset={6}>
          <div className="px-3.5 pt-3.5 pb-2.5 space-y-2.5 overflow-y-auto flex-1 min-h-0">
            <div className="flex gap-2">
              <button
                onClick={() => setDraftScope("all")}
                className={`flex-1 h-8 px-4 rounded-[4px] text-sm leading-[22px] tracking-[0.07px] border transition-colors ${
                  draftScope === "all"
                    ? "bg-[#020617] border-[#020617] text-white"
                    : "bg-white border-[#EAEEF4] text-[var(--text-emphasis)] hover:border-[#020617]"
                }`}
              >
                全部用户
              </button>
              <button
                onClick={() => setDraftScope("groups")}
                className={`flex-1 h-8 px-4 rounded-[4px] text-sm leading-[22px] tracking-[0.07px] border transition-colors ${
                  draftScope === "groups"
                    ? "bg-[#020617] border-[#020617] text-white"
                    : "bg-white border-[#EAEEF4] text-[var(--text-emphasis)] hover:border-[#020617]"
                }`}
              >
                按组织
              </button>
            </div>
            {draftScope === "groups" && (
              <div className="space-y-1.5">
                {!hasGroups ? (
                  <div className="text-center py-5 px-2">
                    <p className="text-xs text-[var(--text-weak)] leading-relaxed">
                      暂无组织，请前往
                      <a
                        href="/admin/members"
                        className="text-[#355EF1] hover:text-[#355EF1] hover:underline mx-0.5"
                        onClick={(e) => { e.preventDefault(); setOpen(false); window.location.href = "/admin/members"; }}
                      >
                        用户管理
                      </a>
                      建立组织
                    </p>
                  </div>
                ) : (
                  <>
                    <div className="group relative flex flex-wrap items-center gap-1 px-2 py-1.5 border border-gray-200 rounded-[4px] bg-gray-50 focus-within:border-blue-300 focus-within:ring-1 focus-within:ring-blue-100 transition-colors max-h-[80px] overflow-y-auto">
                      {selectedTags.map((tag) => (
                        <span
                          key={tag.id}
                          className="inline-flex items-center gap-0.5 px-1.5 py-0.5 bg-blue-50 text-[#1447E6] text-[10px] rounded-[3px] border border-blue-100 shrink-0 max-w-[200px]"
                        >
                          <span className="truncate">{tag.path}</span>
                          <button
                            onClick={() => {
                              const findNode = (nodes: GroupTreeNode[]): GroupTreeNode | undefined => {
                                for (const n of nodes) {
                                  if (n.id === tag.id) return n;
                                  const found = findNode(n.children);
                                  if (found) return found;
                                }
                                return undefined;
                              };
                              let targetNode: GroupTreeNode | undefined;
                              for (const b of activeBuckets) {
                                targetNode = findNode(treesMap[b] || []);
                                if (targetNode) break;
                              }
                              const idsToRemove = targetNode
                                ? new Set(getDescendantIds(targetNode))
                                : new Set([tag.id]);
                              setDraftGroupIds((prev) => prev.filter((id) => !idsToRemove.has(id)));
                            }}
                            className="text-[#355EF1] hover:text-[#355EF1] shrink-0"
                          >
                            <XIcon className="w-2.5 h-2.5" />
                          </button>
                        </span>
                      ))}
                      <input
                        type="text"
                        placeholder={selectedTags.length === 0 ? "请输入组织名称" : ""}
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="flex-1 min-w-[60px] text-xs bg-transparent outline-none placeholder:text-[var(--text-weak)]"
                      />
                      {(selectedTags.length > 0 || searchQuery) && (
                        <button
                          onClick={handleClearSelection}
                          className="absolute right-2 top-1/2 -translate-y-1/2 text-[var(--text-weak)] hover:text-[var(--text-muted)] opacity-0 group-hover:opacity-100 transition-opacity"
                          title="清除全部"
                        >
                          <XIcon className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                    <div className="max-h-[220px] overflow-y-auto">
                      {activeBuckets.map((bucket) => {
                        const trees = treesMap[bucket];
                        if (!trees || trees.length === 0) return null;
                        const anyVisible = trees.some(isNodeVisible);
                        if (!anyVisible) return null;
                        return (
                          <div key={bucket} className="mb-1">
                            <div className="px-2 py-1 text-[10px] font-medium text-[var(--text-weak)] uppercase tracking-wider">
                              {BUCKET_LABELS[bucket]}
                            </div>
                            {trees.map((root) => renderTreeNode(root, 0))}
                          </div>
                        );
                      })}
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
          <div className="flex items-center justify-end gap-2 px-3.5 py-2.5 border-t border-gray-200 shrink-0">
            <Button variant="claw-outline" size="claw-sm" onClick={() => setOpen(false)}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              disabled={isConfirmDisabled}
              onClick={handleConfirm}
            >
              确认
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  );
}

// ─── 顶部总览卡片 ───────────────────────────────────────────────────────
function OverviewStats({
  typeCount,
  enabledTypeCount,
}: {
  typeCount: number;
  enabledTypeCount: number;
}) {
  return (
    <div className="h-9 inline-flex items-center gap-4 px-4 rounded-[4px] bg-white border border-gray-200">
      <div className="flex items-center gap-2">
        <span className="text-[13px] text-[var(--text-muted)]">Agent 类型</span>
        <span className="text-[13px] text-[var(--text-title)] font-semibold tabular-nums">{typeCount}</span>
      </div>
      <span className="w-px h-3 bg-[#E5E5E5]" />
      <div className="flex items-center gap-2">
        <span className="text-[13px] text-[var(--text-muted)]">已对用户可见</span>
        <span className="text-[13px] text-[var(--text-title)] font-semibold tabular-nums">{enabledTypeCount}</span>
      </div>
    </div>
  );
}

// ─── 主组件 ────────────────────────────────────────────────────────────
export default function ImageManagement() {
  // 镜像列表（持久化）
  const [images, setImages] = useState<ImageRow[]>(() => {
    try {
      const raw = localStorage.getItem("admin_images_v3");
      if (raw) return JSON.parse(raw) as ImageRow[];
    } catch { /* ignore */ }
    // 首次访问：用默认 mock 数据并主动持久化，确保其他页面（如 OpenClawMonitor）能立即读到启用版本
    const initial = normalizeImages(MOCK_IMAGES);
    try {
      localStorage.setItem("admin_images_v3", JSON.stringify(initial));
    } catch { /* ignore */ }
    return initial;
  });
  const syncImages = (next: ImageRow[]) => {
    setImages(next);
    localStorage.setItem("admin_images_v3", JSON.stringify(next));
  };

  // 自定义类型（持久化）
  const [customTypes, setCustomTypes] = useState<CustomAgentType[]>(() => loadCustomTypes());

  // 已添加的 agentType（系统 + 自定义），用于控制展示顺序
  const [addedTypes, setAddedTypes] = useState<string[]>(() => {
    const systemIds = SYSTEM_AGENT_TYPES.map((t) => t.value);
    const customIds = loadCustomTypes().map((t) => t.id);
    return [...systemIds, ...customIds];
  });

  // 用户端首选类型
  const [defaultAgentType, setDefaultAgentType] = useState<string>(() => {
    try {
      const v = localStorage.getItem("admin_default_agent_type");
      if (v) return v;
    } catch { /* ignore */ }
    return DEFAULT_AGENT_TYPE;
  });
  const syncDefaultAgentType = (v: string) => {
    setDefaultAgentType(v);
    localStorage.setItem("admin_default_agent_type", v);
  };

  // 应用范围（按 agentType 维度）
  const [typeScopeMap, setTypeScopeMap] = useState<Record<string, ImageScopeData>>(() => {
    try {
      const raw = localStorage.getItem("admin_image_type_scope_v1");
      if (raw) return JSON.parse(raw) as Record<string, ImageScopeData>;
    } catch { /* ignore */ }
    return {};
  });
  const getTypeScope = (typeValue: string): ImageScopeData =>
    typeScopeMap[typeValue] ?? { visibilityScope: "all", visibilityGroupIds: [] };
  const handleScopeChange = (typeValue: string, scope: "all" | "groups", groupIds: string[]) => {
    const next = { ...typeScopeMap, [typeValue]: { visibilityScope: scope, visibilityGroupIds: groupIds } };
    setTypeScopeMap(next);
    localStorage.setItem("admin_image_type_scope_v1", JSON.stringify(next));
  };

  // 添加自定义类型弹窗
  const [showCreateCustomDialog, setShowCreateCustomDialog] = useState(false);
  const [newTypeName, setNewTypeName] = useState("");
  const [newKernelBase, setNewKernelBase] = useState<KernelBase | null>(null);

  // Agent 类型筛选
  const [agentTypeFilter, setAgentTypeFilter] = useState("all");
  const [nativeAck, setNativeAck] = useState(false);
  const newTypeNameError = useMemo(() => {
    if (!newTypeName.trim()) return "";
    const v = newTypeName.trim();
    const allDisplayNames = [
      ...SYSTEM_AGENT_TYPES.map((t) => t.label),
      ...customTypes.map((t) => t.displayName),
    ];
    if (allDisplayNames.includes(v)) return "该类型名称已存在";
    if (v.length < 2 || v.length > 40) return "长度需在 2-40 个字符之间";
    return "";
  }, [newTypeName, customTypes]);
  const resetCreateDialog = () => {
    setNewTypeName("");
    setNewKernelBase(null);
    setNativeAck(false);
  };

  // 删除自定义类型二次确认
  const [removeCustomConfirm, setRemoveCustomConfirm] = useState<CustomAgentType | null>(null);

  // 导入自定义镜像弹窗
  const [showImportDialog, setShowImportDialog] = useState(false);
  const [importTargetAgentType, setImportTargetAgentType] = useState("");
  const [selectedImageId, setSelectedImageId] = useState("");
  const [importAgentType, setImportAgentType] = useState("");
  const [importAgentVersion, setImportAgentVersion] = useState("");
  const [versionError, setVersionError] = useState("");
  const [refreshing, setRefreshing] = useState(false);

  // 编辑单个镜像弹窗
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [editingImageId, setEditingImageId] = useState("");
  const [editAgentType, setEditAgentType] = useState("");
  const [editAgentVersion, setEditAgentVersion] = useState("");
  const [editVersionError, setEditVersionError] = useState("");

  // 推送更新弹窗
  const [showPushDialog, setShowPushDialog] = useState(false);
  const [pushDefaultType, setPushDefaultType] = useState<string | undefined>(undefined);

  // 「查看全部更新记录」抽屉
  const [showAllRecordsDrawer, setShowAllRecordsDrawer] = useState(false);
  // 抽屉打开时的初始筛选（点击表格中"版本更新记录"按钮时，自动定位到该镜像）
  const [drawerInitialFilter, setDrawerInitialFilter] = useState<DrawerInitialFilter | undefined>(undefined);
  // 抽屉打开时是否默认开启「仅看可推送新版本」（点击黄色「新版本更新推送提醒」横幅触发时为 true）
  // (drawer initialPushableOnly 已废弃，推送逻辑改为开关方案)

  // 通过 URL ?openRecords=1 直接打开"全部更新记录"抽屉（供 Agent 列表页等其他页面跳转触发）
  useEffect(() => {
    const search = typeof window !== "undefined" ? window.location.search : "";
    const params = new URLSearchParams(search);
    if (params.get("openRecords") === "1") {
      setDrawerInitialFilter(undefined);
      setShowAllRecordsDrawer(true);
      // 清掉 query 参数，避免再次刷新页面时重复弹出
      params.delete("openRecords");
      const next = params.toString();
      const nextUrl = window.location.pathname + (next ? `?${next}` : "") + window.location.hash;
      window.history.replaceState(null, "", nextUrl);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 通过 URL ?enableImage=<imageId> 自动启用引导（由「克隆 Agent 为镜像」弹窗跳转触发）：
  // 定位包含该镜像的 Agent 类型行 → 自动打开「切换镜像」弹窗并预选该镜像，由用户确认切换启用
  const [autoEnableImageId, setAutoEnableImageId] = useState("");
  useEffect(() => {
    const search = typeof window !== "undefined" ? window.location.search : "";
    const params = new URLSearchParams(search);
    const enableImage = params.get("enableImage");
    if (enableImage) {
      setAutoEnableImageId(enableImage);
      // 清掉 query 参数，避免再次刷新页面时重复触发
      params.delete("enableImage");
      const next = params.toString();
      const nextUrl = window.location.pathname + (next ? `?${next}` : "") + window.location.hash;
      window.history.replaceState(null, "", nextUrl);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 活跃推送订阅（用于在表格 / 顶部黄色提示中显示「正在提醒」状态）
  const [activePushes, setActivePushes] = useState<ActivePush[]>(() => listActivePushes());
  useEffect(() => {
    const onChange = () => setActivePushes(listActivePushes());
    window.addEventListener("upgrade-push-changed", onChange);
    return () => window.removeEventListener("upgrade-push-changed", onChange);
  }, []);

  const openPushDialog = (defaultType?: string) => {
    setPushDefaultType(defaultType);
    setShowPushDialog(true);
  };

  // helpers
  const getTypeConfig = (value: string) => SYSTEM_AGENT_TYPES.find((t) => t.value === value);
  const isCustomType = (value: string) => customTypes.some((t) => t.id === value);
  const getCustomType = (value: string) => customTypes.find((t) => t.id === value);
  const getTypeLabel = (value: string) =>
    getTypeConfig(value)?.label ?? getCustomType(value)?.displayName ?? value;

  // 用于"导入/编辑"时按目标类型决定校验规则
  const getEffectiveTypeConfig = (value: string): AgentTypeConfig | null => {
    const sys = getTypeConfig(value);
    if (sys) return sys;
    const custom = getCustomType(value);
    if (!custom) return null;
    if (custom.kernelBase !== "native") {
      return (
        SYSTEM_AGENT_TYPES.find((t) => t.value === custom.kernelBase) ??
        KERNEL_BASE_FALLBACK_CONFIGS[custom.kernelBase] ??
        null
      );
    }
    return {
      value: custom.id,
      label: custom.displayName,
      isSystem: false,
      versionPlaceholder: "如 1.0.0",
      versionRegex: /^\d+\.\d+\.\d+$/,
    };
  };

  const displayTypes = addedTypes.filter(
    (v) => SYSTEM_AGENT_TYPES.some((c) => c.value === v) || isCustomType(v),
  );

  // 派生视图
  const views = useMemo(
    () =>
      displayTypes.map((t) => ({
        agentType: t,
        view: deriveAgentTypeView(images, t),
      })),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [images, addedTypes, customTypes],
  );

  // 计算"可推送的 Agent 类型"列表（已启用镜像 + 开启了提醒开关 + 旧版本实例数 mock）
  const pushable: PushableAgentType[] = useMemo(() => {
    return views
      .filter(({ view, agentType }) =>
        view.enabled.isEnabled && view.enabled.version && isReminderEnabled(agentType),
      )
      .map(({ agentType, view }) => {
        const enabledVersion = view.enabled.version!;
        const enabledImage = images.find(
          (i) => i.agentType === agentType && i.active,
        );
        if (!enabledImage) return null;
        // mock：旧版本实例数 = hash(agentType + version) % 30 + 5（演示用稳定值）
        const seed = `${agentType}:${enabledVersion}`;
        let h = 0;
        for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0;
        const outdatedInstanceCount = (h % 28) + 3;
        const allUpToDate = outdatedInstanceCount === 0;
        return {
          agentType,
          agentTypeLabel: getTypeLabel(agentType),
          enabledVersion,
          enabledImage,
          imageName: enabledImage.name,
          imageSource: enabledImage.type,
          outdatedInstanceCount,
          allUpToDate,
        } satisfies PushableAgentType;
      })
      .filter((x): x is PushableAgentType => x !== null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [views, images]);

  // 创建自定义类型
  const canCreateCustom = !!newTypeName.trim()
    && !newTypeNameError
    && !!newKernelBase
    && (newKernelBase !== "native" || nativeAck);

  const handleCreateCustomType = () => {
    if (!canCreateCustom || !newKernelBase) return;
    const slugBase = newTypeName
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, "-")
      .replace(/^-|-$/g, "")
      .slice(0, 40) || "custom";
    let slug = slugBase;
    let idx = 1;
    const existingIds = customTypes.map((t) => t.id);
    while (existingIds.includes(`custom-${slug}`)) {
      slug = `${slugBase}-${++idx}`;
    }
    const newType: CustomAgentType = {
      id: `custom-${slug}`,
      displayName: newTypeName.trim(),
      kernelBase: newKernelBase,
      createdAt: nowStr(),
      updatedAt: nowStr(),
      linkedInstanceCount: 0,
      createdBy: "alice@acompany.com",
      nativeTerminalAck: newKernelBase === "native" ? true : undefined,
    };
    const nextTypes = [...customTypes, newType];
    setCustomTypes(nextTypes);
    saveCustomTypes(nextTypes);
    if (!addedTypes.includes(newType.id)) {
      setAddedTypes([...addedTypes, newType.id]);
    }
    toast.success(`已添加自定义 Agent 类型「${newType.displayName}」`);
    resetCreateDialog();
    setShowCreateCustomDialog(false);
    setTimeout(() => {
      const el = document.getElementById(`section-${newType.id}`);
      if (el) el.scrollIntoView({ behavior: "smooth", block: "start" });
    }, 100);
  };

  // 删除类型
  const handleRemoveType = (value: string) => {
    if (SYSTEM_AGENT_TYPES.some((t) => t.value === value)) {
      toast.error("系统预设 Agent 类型不可移除");
      return;
    }
    if (defaultAgentType === value) {
      toast.error("该类型是用户端首选类型，请先切换首选类型");
      return;
    }
    if (images.some((i) => i.agentType === value && i.active)) {
      toast.error("该类型下有用户可见的镜像，请先关闭用户可见");
      return;
    }
    if (isCustomType(value)) {
      const customType = getCustomType(value)!;
      setRemoveCustomConfirm(customType);
    }
  };

  const confirmRemoveCustomType = () => {
    if (!removeCustomConfirm) return;
    const value = removeCustomConfirm.id;
    const nextCustomTypes = customTypes.filter((t) => t.id !== value);
    setCustomTypes(nextCustomTypes);
    saveCustomTypes(nextCustomTypes);
    syncImages(images.filter((i) => i.agentType !== value));
    setAddedTypes(addedTypes.filter((v) => v !== value));
    if (typeScopeMap[value]) {
      const next = { ...typeScopeMap };
      delete next[value];
      setTypeScopeMap(next);
      localStorage.setItem("admin_image_type_scope_v1", JSON.stringify(next));
    }
    setRemoveCustomConfirm(null);
    toast.success(`已删除自定义 Agent 类型「${removeCustomConfirm.displayName}」`);
  };

  // 启用/关闭/编辑/导入/默认 等 handlers
  const handleEnableImage = (imageId: string, hintAgentType?: string) => {
    const existing = images.find((i) => i.id === imageId);
    if (existing) {
      if (!existing.agentVersion?.trim()) {
        toast.error("此镜像缺少版本号，无法对用户可见，请先编辑补齐");
        return;
      }
      syncImages(images.map((i) =>
        i.agentType === existing.agentType ? { ...i, active: i.id === imageId } : i,
      ));
      // 启用版本变了 → 撤回该类型的旧推送（如有）
      pruneOnVersionChange(existing.agentType, existing.agentVersion);
      toast.success(`已对用户可见「${existing.name}」（v${existing.agentVersion}）`);
      return;
    }
    const virtual = resolveVirtualPublicImage(imageId, hintAgentType);
    if (!virtual) { toast.error("镜像信息不完整，无法对用户可见"); return; }
    const newImage: ImageRow = {
      id: virtual.imageId, name: virtual.imageName, status: "available", type: "public",
      agentType: virtual.agentType, agentVersion: virtual.latestVersion,
      os: "CentOS 7.9 64位", createTime: nowStr(), active: true,
    };
    syncImages([
      ...images.map((i) => i.agentType === virtual.agentType ? { ...i, active: false } : i),
      newImage,
    ]);
    if (!addedTypes.includes(virtual.agentType)) setAddedTypes([...addedTypes, virtual.agentType]);
    pruneOnVersionChange(virtual.agentType, virtual.latestVersion);
    toast.success(`已对用户可见「${virtual.imageName}」（v${virtual.latestVersion}）`);
  };

  const resolveVirtualPublicImage = (imageId: string, hintAgentType?: string) => {
    const candidates: string[] = hintAgentType ? [hintAgentType] : Object.keys(IMG_TO_VERSION_KEY);
    for (const agentType of candidates) {
      const versionKey = IMG_TO_VERSION_KEY[agentType];
      if (!versionKey) continue;
      const meta = getImageMeta(versionKey);
      const latest = AGENT_VERSIONS.filter((v) => v.agentType === versionKey)
        .sort((a, b) => b.releaseTime.localeCompare(a.releaseTime))[0];
      if (!meta || !latest) continue;
      if (imageId === meta.imageId) {
        return { imageId: meta.imageId, imageName: meta.imageName, agentType, latestVersion: latest.version };
      }
      const hardenedId = getHardenedImageId(agentType);
      if (hardenedId && imageId === hardenedId) {
        return { imageId: hardenedId, imageName: `${meta.imageName} · 安全加固版`, agentType, latestVersion: latest.version };
      }
    }
    return null;
  };

  const handleDisableImage = (imageId: string) => {
    const target = images.find((i) => i.id === imageId);
    if (!target || !target.active) return;
    if (target.agentType === defaultAgentType) {
      toast.error(`${getTypeLabel(target.agentType)} 是用户端首选类型，必须保持有用户可见的镜像。请先切换其它镜像或修改首选类型`);
      return;
    }
    syncImages(images.map((i) => (i.id === imageId ? { ...i, active: false } : i)));
    // 关闭用户可见 → 没有启用版本，撤回该类型的旧推送
    pruneOnVersionChange(target.agentType, null);
    toast.success(`已关闭「${target.name}」用户可见，该 Agent 类型当前对用户不可见`);
  };

  const openPublicHistoryByImage = (imageId: string) => {
    // 优先使用表格里的实际镜像 ID；虚拟（安全加固版等）则取解析后的真实 ID
    const existing = images.find((i) => i.id === imageId);
    let resolvedId = existing?.id;
    if (!resolvedId) {
      const virtual = resolveVirtualPublicImage(imageId);
      resolvedId = virtual?.imageId;
    }
    // 唤起全部更新记录侧边栏，并自动筛选到对应 agent 镜像
    if (resolvedId) {
      setDrawerInitialFilter({ kind: "image", value: resolvedId });
    } else {
      setDrawerInitialFilter(undefined);
    }
    setShowAllRecordsDrawer(true);
  };

  const openImportFor = (agentType: string) => {
    setImportTargetAgentType(agentType);
    setImportAgentType(agentType);
    setImportAgentVersion("");
    setVersionError("");
    setSelectedImageId("");
    setShowImportDialog(true);
  };

  const handleDialogOpenChange = (open: boolean) => {
    setShowImportDialog(open);
    if (!open) {
      setSelectedImageId(""); setImportAgentType(""); setImportAgentVersion("");
      setVersionError(""); setImportTargetAgentType("");
    }
  };

  const handleRefresh = () => {
    setRefreshing(true);
    setTimeout(() => { setRefreshing(false); toast.success("镜像列表已刷新"); }, 1000);
  };

  const handleVersionChange = (v: string) => {
    setImportAgentVersion(v);
    if (v && importAgentType) {
      const config = getEffectiveTypeConfig(importAgentType);
      if (config && config.versionRegex && !config.versionRegex.test(v)) {
        setVersionError(`格式不正确，请输入 ${config.versionPlaceholder.replace("如 ", "")} 格式`);
      } else if (importAgentType === "OpenClaw" && config && !validateVersion(config, v)) {
        setVersionError("日期不合法");
      } else { setVersionError(""); }
    } else { setVersionError(""); }
  };

  const canImport =
    selectedImageId && importAgentType && importAgentVersion.trim() && !versionError;

  const handleImport = () => {
    if (!selectedImageId) { toast.error("请选择要关联的镜像"); return; }
    if (!importAgentType) { toast.error("请选择 Agent 类型"); return; }
    if (!importAgentVersion.trim()) { toast.error("请填写 Agent 版本"); return; }
    const config = getEffectiveTypeConfig(importAgentType);
    if (config && config.versionRegex && !validateVersion(config, importAgentVersion)) {
      toast.error("版本格式不正确"); return;
    }
    if (images.some((img) => img.id === selectedImageId)) { toast.error("该镜像已在列表中"); return; }
    const img = CUSTOM_IMAGES.find((i) => i.id === selectedImageId)!;
    const noneActive = !images.some((i) => i.agentType === importAgentType && i.active);
    const newImage: ImageRow = {
      id: img.id, name: img.name, status: "available", type: "custom",
      agentType: importAgentType, agentVersion: importAgentVersion.trim(),
      os: "CentOS 7.9 64位", createTime: nowStr(), active: noneActive,
    };
    syncImages([...images, newImage]);
    if (!addedTypes.includes(importAgentType)) setAddedTypes([...addedTypes, importAgentType]);
    handleDialogOpenChange(false);
    toast.success(noneActive
      ? `镜像「${img.name}」已导入至 ${getTypeLabel(importAgentType)} 并自动对用户可见`
      : `镜像「${img.name}」已导入至 ${getTypeLabel(importAgentType)}`);
  };

  const openEditDialog = (imgId: string) => {
    const img = images.find((i) => i.id === imgId);
    if (!img) return;
    if (img.type === "public") { toast.error("公共镜像不支持编辑"); return; }
    setEditingImageId(img.id);
    setEditAgentType(img.agentType);
    setEditAgentVersion(img.agentVersion);
    setEditVersionError("");
    setShowEditDialog(true);
  };

  const handleEditVersionChange = (v: string) => {
    setEditAgentVersion(v);
    if (v && editAgentType) {
      const config = getEffectiveTypeConfig(editAgentType);
      if (config && config.versionRegex && !config.versionRegex.test(v)) {
        setEditVersionError(`格式不正确，请输入 ${config.versionPlaceholder.replace("如 ", "")} 格式`);
      } else if (editAgentType === "OpenClaw" && config && !validateVersion(config, v)) {
        setEditVersionError("日期不合法");
      } else { setEditVersionError(""); }
    } else { setEditVersionError(""); }
  };

  const handleEditSave = () => {
    if (!editAgentType) { toast.error("请选择 Agent 类型"); return; }
    if (!editAgentVersion.trim()) { toast.error("请填写 Agent 版本"); return; }
    const config = getEffectiveTypeConfig(editAgentType);
    if (config && config.versionRegex && !validateVersion(config, editAgentVersion)) {
      toast.error("版本格式不正确"); return;
    }
    syncImages(images.map((i) =>
      i.id === editingImageId
        ? { ...i, agentType: editAgentType, agentVersion: editAgentVersion.trim() }
        : i,
    ));
    if (!addedTypes.includes(editAgentType)) setAddedTypes([...addedTypes, editAgentType]);
    setShowEditDialog(false);
    toast.success("镜像信息已更新");
  };

  const handleDeleteImage = (imgId: string) => {
    const img = images.find((i) => i.id === imgId);
    if (!img) return;
    if (img.type === "public") { toast.error("公共镜像不支持删除"); return; }
    if (img.active) {
      if (img.agentType === defaultAgentType) {
        toast.error("该镜像为用户端首选类型的启用镜像，无法删除"); return;
      }
      toast.error("当前用户可见的镜像无法删除，请先切换至其它镜像"); return;
    }
    syncImages(images.filter((i) => i.id !== imgId));
    toast.success("镜像已删除");
  };

  const handleSetDefaultType = (agentType: string) => {
    if (!images.some((i) => i.agentType === agentType && i.active)) {
      toast.error(`请先为 ${getTypeLabel(agentType)} 启用一个用户可见的镜像`);
      return;
    }
    syncDefaultAgentType(agentType);
    toast.success(`已将「${getTypeLabel(agentType)}」设为用户端首选类型`);
  };

  /** 直接推送某 Agent 类型当前启用版本（不弹窗，立即生效） */
  const handleDirectPush = (agentType: string) => {
    const item = pushable.find((p) => p.agentType === agentType);
    if (!item) {
      toast.error("无法获取当前用户可见镜像信息");
      return;
    }
    if (item.allUpToDate) {
      toast.info(`「${item.agentTypeLabel}」所有 Agent 已是最新版，无需推送`);
      return;
    }
    setActivePush({
      agentType: item.agentType,
      agentTypeLabel: item.agentTypeLabel,
      version: item.enabledVersion,
      imageName: item.imageName,
      imageSource: item.imageSource,
      pushedAt: nowStr(),
      pushedBy: "alice@acompany.com",
      message: `管理员推荐更新到 v${item.enabledVersion}`,
    });
    toast.success(
      `已向「${item.agentTypeLabel}」共 ${item.outdatedInstanceCount} 个旧版本 Agent 推送更新提醒`,
    );
  };

  /** 撤回某 Agent 类型当前的推送 */
  const handleRevokePush = (agentType: string) => {
    const label = getTypeLabel(agentType);
    clearActivePush(agentType);
    toast.success(`已撤回「${label}」的更新推送提醒`);
  };

  /** 版本模式/版本号变更：联动推送状态 */
  const handleVersionModeChange = (agentType: string, mode: "auto" | "pinned", version: string | null) => {
    // 切换模式或版本时，旧推送自动失效
    if (version) {
      pruneOnVersionChange(agentType, version);
    } else {
      pruneOnVersionChange(agentType, null);
    }
  };

  // ── 渲染 ──────────────────────────────────────────────────────────
  return (
    <>
      <div className="page-enter">
        <div className="min-w-0">
          <AdminPageHeader
            title="Agent 类型"
            description="通过启用镜像决定用户端可以使用的 Agent 类型，支持自定义 Agent 类型。"
          />

          {/* 主体内容 */}
          <div className="min-w-0">
          {/* 顶部总览 + 添加自定义类型按钮 */}
          <div className="mb-4 flex items-center justify-between gap-3 flex-wrap">
            <div className="flex items-center gap-3 flex-wrap">
              <OverviewStats
                typeCount={views.length}
                enabledTypeCount={views.filter((v) => v.view.enabled.isEnabled).length}
              />
            </div>
            <div className="flex items-center gap-2">
            <Button
              variant="claw-outline"
              size="claw"
              onClick={() => {
                setDrawerInitialFilter(undefined);
                setShowAllRecordsDrawer(true);
              }}
              className="shrink-0"
            >
              <History className="w-4 h-4" />
              版本更新记录
            </Button>
            <Button
              variant="claw-primary"
              size="claw"
              onClick={() => setShowCreateCustomDialog(true)}
              className="shrink-0"
            >
              <Plus className="w-4 h-4" />
              添加自定义 Agent 类型
            </Button>
            </div>
          </div>

          {/* 大表格 */}
          <AgentTypesTable
            autoEnableImageId={autoEnableImageId}
            rows={views.filter(({ agentType }) => agentTypeFilter === "all" || agentType === agentTypeFilter).map(({ agentType, view }): AgentTypeRowData => {
              const customType = getCustomType(agentType);
              return {
                agentType,
                view,
                label: getTypeLabel(agentType),
                isDefault: defaultAgentType === agentType,
                customType,
                kernelBaseLabel:
                  customType && customType.kernelBase !== "native"
                    ? kernelBaseLabel(customType.kernelBase)
                    : undefined,
              };
            })}
            onSetDefaultType={handleSetDefaultType}
            onRemoveCustomType={handleRemoveType}
            onPushUpgrade={(agentType) => handleDirectPush(agentType)}
            onRevokePush={(agentType) => handleRevokePush(agentType)}
            onVersionModeChange={handleVersionModeChange}
            pushableByType={new Map(pushable.map(p => {
              const ap = activePushes.find(
                (x) => x.agentType === p.agentType && x.version === p.enabledVersion,
              );
              return [p.agentType, {
                outdatedInstanceCount: p.outdatedInstanceCount,
                allUpToDate: p.allUpToDate,
                isActivePushing: !!ap,
              }];
            }))}
            onEnableImage={(imageId, agentType) => handleEnableImage(imageId, agentType)}
            onDisableImage={handleDisableImage}
            onSelectImage={(imageId, agentType) => handleEnableImage(imageId, agentType)}
            onEditImage={openEditDialog}
            onDeleteImage={handleDeleteImage}
            onViewPublicHistory={openPublicHistoryByImage}
            onImportCustom={(agentType) => openImportFor(agentType)}
            renderScope={(agentType) => {
              const scopeData = getTypeScope(agentType);
              // 应用范围标签：全部用户使用 outline（白底描边），组织使用 secondary（灰底黑字）
              const isAll = scopeData.visibilityScope === "all" || scopeData.visibilityGroupIds.length === 0;
              const groupPaths = scopeData.visibilityGroupIds.map((gid) => getGroupPath(gid, ALL_GROUPS));
              const firstName = groupPaths[0];
              const rest = groupPaths.length - 1;
              const tooltipText = groupPaths.join("\n");
              return (
                <div className="inline-flex items-center gap-1.5 min-h-[20px] max-w-[220px]">
                  {isAll ? (
                    <Badge variant="outline">全部用户</Badge>
                  ) : (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="inline-flex max-w-full items-center gap-1 cursor-default">
                          <Badge variant="secondary" className="max-w-[140px]">
                            <span className="block truncate max-w-[124px]">{firstName}</span>
                          </Badge>
                          {rest > 0 && (
                            <Badge variant="secondary">+{rest}</Badge>
                          )}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent className="max-w-[320px] text-xs leading-relaxed whitespace-pre-line">
                        {tooltipText}
                      </TooltipContent>
                    </Tooltip>
                  )}
                  <ScopeSelect
                    scope={scopeData.visibilityScope}
                    selectedGroupIds={scopeData.visibilityGroupIds}
                    groups={ALL_GROUPS}
                    showBadges={false}
                    onConfirm={(scope, groupIds) => handleScopeChange(agentType, scope, groupIds)}
                  />
                </div>
              );
            }}
          />
          </div>
        </div>
      </div>

      {/* ─── 导入自定义镜像弹窗 ─── */}
      <Dialog open={showImportDialog} onOpenChange={handleDialogOpenChange}>
        <DialogContent
          className="sm:max-w-md"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
        >
          <DialogHeader>
            <DialogTitle>
              导入自定义镜像
              {importTargetAgentType && (
                <span className="text-xs text-[var(--text-weak)] font-normal ml-2">
                  · {getTypeLabel(importTargetAgentType)}
                </span>
              )}
            </DialogTitle>
          </DialogHeader>

          <Alert variant="info">
            <AlertInfoIcon />
            <AlertTitle>自定义镜像由企业自行制作和维护</AlertTitle>
            <AlertDescription>
              <ul className="space-y-1 text-[11px]">
                <li className="flex items-start gap-1.5">
                  <Check className="w-3 h-3 mt-0.5 text-green-600 shrink-0" />
                  <span>版本固定不变，便于合规审计与稳定性管理</span>
                </li>
                <li className="flex items-start gap-1.5">
                  <Check className="w-3 h-3 mt-0.5 text-green-600 shrink-0" />
                  <span>可基于业务需要预装技能、插件、企业内部依赖</span>
                </li>
                <li className="flex items-start gap-1.5">
                  <XIcon className="w-3 h-3 mt-0.5 text-[var(--text-weak)] shrink-0" />
                  <span>需要企业自行跟进版本更新与维护</span>
                </li>
              </ul>
              <p className="mt-2 pt-2 border-t border-[var(--alert-info-border)]/60 text-[11px] leading-relaxed">
                以下镜像均为已在腾讯云创建好的镜像。若需要创建新镜像，请前往{" "}
                <a
                  href="https://console.cloud.tencent.com/cvm/image"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-[#1447E6] hover:opacity-80 inline-flex items-center gap-0.5"
                >
                  腾讯云云服务器控制台 <ExternalLink className="w-3 h-3" />
                </a>{" "}
                操作后，再回此处刷新并关联。
              </p>
            </AlertDescription>
          </Alert>

          <div className="mt-4 space-y-4 py-1">
            <div className="space-y-2">
              <Label>选择镜像 <span className="text-red-400">*</span></Label>
              <div className="flex items-center gap-2">
                <Select
                  value={selectedImageId}
                  onValueChange={(v) => setSelectedImageId(v)}
                >
                  <SelectTrigger className="bg-white flex-1">
                    <SelectValue placeholder="请选择要关联的镜像" />
                  </SelectTrigger>
                  <SelectContent>
                    {CUSTOM_IMAGES.map((img) => (
                      <SelectItem key={img.id} value={img.id}>
                        <span className="inline-flex items-center gap-2">
                          <span>{img.name}</span>
                          <span className="text-xs text-[var(--text-weak)] font-mono">{img.id}</span>
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button
                  variant="claw-outline"
                  size="icon"
                  onClick={handleRefresh}
                  disabled={refreshing}
                  title="刷新镜像列表"
                  className="flex-shrink-0"
                >
                  <RefreshCw className={refreshing ? "animate-spin" : ""} />
                </Button>
              </div>
              <p className="text-xs text-[var(--text-weak)]">镜像大小不允许超过 50 GiB</p>
            </div>

            <div className="space-y-2">
              <Label>Agent 类型 <span className="text-red-400">*</span></Label>
              <Select
                value={importAgentType}
                onValueChange={(v) => {
                  setImportAgentType(v);
                  setImportAgentVersion("");
                  setVersionError("");
                }}
              >
                <SelectTrigger className="bg-white w-full">
                  <SelectValue placeholder="请选择 Agent 类型" />
                </SelectTrigger>
                <SelectContent>
                  {SYSTEM_AGENT_TYPES.map((t) => (
                    <SelectItem key={t.value} value={t.value}>
                      {t.label}
                    </SelectItem>
                  ))}
                  {customTypes.length > 0 && (
                    <>
                      <div className="px-2 py-1 text-[10px] text-[var(--text-weak)] uppercase tracking-wide border-t border-gray-100 mt-1">
                        自定义类型
                      </div>
                      {customTypes.map((t) => (
                        <SelectItem key={t.id} value={t.id}>
                          <span className="inline-flex items-center gap-1.5">
                            <span>{t.displayName}</span>
                            {t.kernelBase !== "native" && (
                              <StatusTag mode="fill" variant="gray">
                                兼容 {kernelBaseLabel(t.kernelBase)}
                              </StatusTag>
                            )}
                          </span>
                        </SelectItem>
                      ))}
                    </>
                  )}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Agent 版本 <span className="text-red-400">*</span></Label>
              <Input
                value={importAgentVersion}
                onChange={(e) => handleVersionChange(e.target.value)}
                placeholder={getEffectiveTypeConfig(importAgentType)?.versionPlaceholder ?? "请先选择 Agent 类型"}
                className={versionError ? "border-red-300 focus-visible:ring-red-500" : ""}
                disabled={!importAgentType}
              />
              {versionError && <p className="text-xs text-red-500">{versionError}</p>}
            </div>
          </div>

          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => handleDialogOpenChange(false)}>取消</Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={handleImport}
              disabled={!canImport}
            >
              导入并关联
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ─── 编辑镜像信息弹窗 ─── */}
      <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
        <DialogContent
          className="sm:max-w-md"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
        >
          <DialogHeader>
            <DialogTitle>编辑镜像信息</DialogTitle>
            <DialogDescription className="text-xs text-[var(--text-muted)]">
              修改该自定义镜像所属的 Agent 类型和版本号
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-1">
            <div className="space-y-2">
              <Label>Agent 类型 <span className="text-red-400">*</span></Label>
              <Select
                value={editAgentType}
                onValueChange={(v) => { setEditAgentType(v); setEditAgentVersion(""); setEditVersionError(""); }}
              >
                <SelectTrigger className="bg-white w-full">
                  <SelectValue placeholder="请选择 Agent 类型" />
                </SelectTrigger>
                <SelectContent>
                  {SYSTEM_AGENT_TYPES.map((t) => (
                    <SelectItem key={t.value} value={t.value}>{t.label}</SelectItem>
                  ))}
                  {customTypes.length > 0 && (
                    <>
                      <div className="px-2 py-1 text-[10px] text-[var(--text-weak)] uppercase tracking-wide border-t border-gray-100 mt-1">
                        自定义类型
                      </div>
                      {customTypes.map((t) => (
                        <SelectItem key={t.id} value={t.id}>{t.displayName}</SelectItem>
                      ))}
                    </>
                  )}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Agent 版本 <span className="text-red-400">*</span></Label>
              <Input
                value={editAgentVersion}
                onChange={(e) => handleEditVersionChange(e.target.value)}
                placeholder={getEffectiveTypeConfig(editAgentType)?.versionPlaceholder ?? ""}
                className={editVersionError ? "border-red-300 focus-visible:ring-red-500" : ""}
              />
              {editVersionError && <p className="text-xs text-red-500">{editVersionError}</p>}
            </div>
          </div>
          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={() => setShowEditDialog(false)}>取消</Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={handleEditSave}
            >
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ─── 添加自定义 Agent 类型弹窗 ─── */}
      <Dialog open={showCreateCustomDialog} onOpenChange={(o) => { if (!o) resetCreateDialog(); setShowCreateCustomDialog(o); }}>
        <DialogContent
          size="lg"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
        >
          <DialogHeader>
            <DialogTitle>添加自定义 Agent 类型</DialogTitle>
            <DialogDescription className="text-xs text-[var(--text-muted)]">
              支持基于现有 Agent 内核扩展，或添加完全自研的 Agent 类型
            </DialogDescription>
          </DialogHeader>

          <DialogBody className="px-6 flex-1">
          <div className="space-y-4">
            {/* 类型名称 */}
            <div>
              <Label className="text-xs">类型名称 <span className="text-red-400">*</span></Label>
              <Input
                value={newTypeName}
                onChange={(e) => setNewTypeName(e.target.value)}
                placeholder="例如：CustomClaw"
                maxLength={40}
                autoFocus
                className={`mt-1.5 ${newTypeNameError ? "border-red-300 focus-visible:ring-red-500" : ""}`}
              />
              {newTypeNameError ? (
                <p className="text-xs text-red-500 mt-1">{newTypeNameError}</p>
              ) : (
                <p className="text-xs text-[var(--text-weak)] mt-1">用户端展示的类型名称，可包含中英文，需保持唯一</p>
              )}
            </div>

            {/* 兼容内核 */}
            <div>
              <Label className="text-xs">兼容内核 <span className="text-red-400">*</span></Label>
              <RadioGroup
                value={newKernelBase ?? ""}
                onValueChange={(v) => {
                  setNewKernelBase(v as KernelBase);
                  if (v !== "native") setNativeAck(false);
                }}
                className="mt-2 gap-2"
              >
                {KERNEL_OPTIONS.map((opt) => {
                  const isSelected = newKernelBase === opt.value;
                  const isNative = opt.value === "native";
                  return (
                    <RadioCard
                      key={opt.value}
                      id={`kernel-opt-${opt.value}`}
                      value={opt.value}
                      checked={isSelected}
                      title={opt.title}
                      description={opt.description}
                      checkedClassName={isNative ? "border-orange-400 bg-orange-50/60" : undefined}
                      radioCheckedClassName={isNative ? "border-orange-500 data-[state=checked]:border-orange-500 [&_svg]:fill-orange-500 [&_svg]:text-orange-500" : undefined}
                    />
                  );
                })}
              </RadioGroup>
            </div>

            {/* 兼容内核：温馨提示 */}
            {newKernelBase && newKernelBase !== "native" && (
              <Alert variant="info">
                <AlertInfoIcon />
                <AlertDescription>
                  将直接复用 <strong>{kernelBaseLabel(newKernelBase)}</strong> 的全部管控能力，无需额外配置。创建后可直接导入镜像使用
                </AlertDescription>
              </Alert>
            )}

            {/* 自研（native）：风险提示 + 必勾选确认 */}
            {newKernelBase === "native" && (
              <div className="space-y-2">
                <Alert variant="warning">
                  <CircleAlert />
                  <AlertTitle>{NATIVE_KERNEL_NOTICE_TITLE}：</AlertTitle>
                  <AlertDescription>
                    {NATIVE_KERNEL_NOTICE_LINES.map((l, i) => (<p key={i}>{l}</p>))}
                  </AlertDescription>
                </Alert>
                <label className="flex items-start gap-2 px-3 py-2 rounded-[4px] border border-gray-200 cursor-pointer hover:bg-[#FAFAFA]">
                  <Checkbox
                    id="native-ack"
                    checked={nativeAck}
                    onCheckedChange={(c) => setNativeAck(!!c)}
                    className="mt-0.5"
                  />
                  <span className="text-xs text-[var(--text-secondary)] leading-relaxed">
                    我已知晓上述限制，且确认<span className="font-medium">允许用户进入该类型 Agent 的终端</span>
                  </span>
                </label>
              </div>
            )}
          </div>
          </DialogBody>

          <DialogFooter>
            <Button variant="outline" onClick={() => { resetCreateDialog(); setShowCreateCustomDialog(false); }}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={handleCreateCustomType}
              disabled={!canCreateCustom}
            >
              添加
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ─── 删除自定义类型 二次确认（警示弹窗） ─── */}
      <AlertDialog open={!!removeCustomConfirm} onOpenChange={(o) => { if (!o) setRemoveCustomConfirm(null); }}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[var(--text-title)]">删除自定义 Agent 类型</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <div className="space-y-4">
              <Alert variant="warning">
                <CircleAlert />
                <AlertTitle>注意事项</AlertTitle>
                <AlertDescription>
                  删除后该类型及其下镜像将不再展示，可重新添加。此操作不会影响已存在的腾讯云镜像数据。
                </AlertDescription>
              </Alert>
              <p className="text-sm text-[var(--text-title)]">
                确定要删除自定义 Agent 类型 <span className="font-medium text-[var(--text-title)]">「{removeCustomConfirm?.displayName}」</span> 吗？
              </p>
            </div>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setRemoveCustomConfirm(null)}>取消</AlertDialogCancel>
            <AlertDialogAction onClick={confirmRemoveCustomType}>确认删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ─── 推送更新弹窗 ─── */}
      <PushUpgradeDialog
        open={showPushDialog}
        onOpenChange={(o) => {
          setShowPushDialog(o);
          if (!o) setPushDefaultType(undefined);
        }}
        pushable={pushable}
        defaultAgentType={pushDefaultType}
        onViewAllRecords={() => {
          setShowPushDialog(false);
          setPushDefaultType(undefined);
          setDrawerInitialFilter(undefined);
          setShowAllRecordsDrawer(true);
        }}
      />

      {/* ─── 镜像更新记录抽屉 ─── */}
      <UpdateRecordsDrawer
        open={showAllRecordsDrawer}
        onOpenChange={(o) => {
          setShowAllRecordsDrawer(o);
          if (!o) {
            setDrawerInitialFilter(undefined);
          }
        }}
        initialFilter={drawerInitialFilter}
      />
    </>
  );
}
