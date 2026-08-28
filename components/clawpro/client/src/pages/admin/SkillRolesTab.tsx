/**
 * SkillRolesTab - 角色设定管理
 * 功能：拖拽排序、开关可见性、编辑角色、删除角色、新增自定义角色、应用范围
 */
import { useState, useRef, useEffect, useCallback, Fragment } from "react";
import { Pagination } from "@/components/ui/pagination";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Button } from "@/components/ui/button";
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
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { FilterChipGroup } from "@/components/ui/filter-chip";
import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  EmptyDescription,
} from "@/components/ui/empty";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableExpandedRow,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import { SurfaceCard } from "@/components/ui/Surface";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  InstantMultiSelect,
} from "@/components/ui/select";
import { toast } from "sonner";
import {
  Plus,
  GripVertical,
  Trash2,
  X,
  Search,
  Check,
  CheckCircle2,
  RefreshCw,
  Package,
  Star,
  ChevronDown,
  ChevronRight,
  AlertCircle,
  Loader2,
  History,
} from "lucide-react";
import { StatusTag, type StatusTagColor } from "@/components/ui/status-tag";
import { SkillSelectCard } from "@/components/ui/skill-select-card";
import { AllUsersTag } from "@/components/ui/all-users-tag";
import {
  MOCK_ROLES,
  PROGRAMMER_SOUL,
} from "@/lib/mockData";
import type { Role, RoleSkill, RoleUpdateRecord } from "@/lib/mockData";
import { loadClawList, saveClawList, notifyClawListChange, type AgentItem } from "@/lib/openclawStore";
import { PUBLIC_SKILLS, type PublicSkill } from "./SkillLibrary/publicSkillMockData";
import {
  PUBLIC_SKILL_PACKAGES,
  type PublicSkillPackage,
} from "./SkillLibrary/publicSkillPackageMockData";
import { MOCK_SKILLS, DEFAULT_CATEGORIES, MOCK_GROUPS } from "./SkillLibrary/mockData";
import type { SkillScope } from "./SkillLibrary/types";
import { MOCK_GROUPS as MOCK_ONEID_GROUPS, MOCK_MANUAL_GROUPS } from "./MemberManagement/mock";
import {
  BodyText,
  BodyMedium,
  MetaText,
  MetaMedium,
  PanelTitle,
  HelperText,
} from "@/components/ui/Typography";

/** 与 agent 类型页一致的应用范围数据源（OneID 部门 + 自建组织），保证下拉面板与 agent 类型页完全一致 */
const ALL_GROUPS = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];

// ── 编辑应用范围（使用通用 ScopeSelect）──────────────────────
import { ScopeSelect, type ScopeType } from "@/components/ScopeSelect";

function EditRoleScopePopover({
  role,
  onConfirm,
}: {
  role: Role;
  onConfirm: (scope: SkillScope, groupIds: string[]) => void;
}) {
  const isPublic = role.scope === 'public' || !role.groupIds || role.groupIds.length === 0;
  const groupNames = (role.groupIds || []).map(gid => MOCK_GROUPS.find(g => g.id === gid)?.name || gid);

  const mapScope = (s: SkillScope): ScopeType => (s === "public" ? "all" : "groups");
  const reverseScope = (s: ScopeType): SkillScope => (s === "all" ? "public" : "private");

  return (
    <ScopeSelect
      scope={mapScope(role.scope || "public")}
      selectedGroupIds={role.groupIds || []}
      groups={MOCK_GROUPS}
      scopeLabels={groupNames}
      showBadges={true}
      onConfirm={(scope, groupIds) => {
        onConfirm(reverseScope(scope), groupIds);
      }}
    />
  );
}

// ── Mock 最新版本查询 ──────────────────────────────────────
/** 根据技能名称和来源查询最新可用版本 */
function getLatestVersionInfo(skillName: string, source: "公共" | "企业"): { latestVersion: string; updateNote: string } | null {
  if (source === "公共") {
    const pubSkill = PUBLIC_SKILLS.find(s => s.name === skillName || s.slug === skillName);
    if (pubSkill) {
      return { latestVersion: `v${pubSkill.version}`, updateNote: `公共技能 ${pubSkill.nameZh || pubSkill.name} 的最新版本更新` };
    }
  } else {
    const entSkill = MOCK_SKILLS.find(s => s.name === skillName || s.slug === skillName);
    if (entSkill) {
      const note = entSkill.versionHistory?.[0]?.changeLog || `企业技能 ${entSkill.name} 的最新版本更新`;
      return { latestVersion: `v${entSkill.version}`, updateNote: note };
    }
  }
  return null;
}

/** 比较 vA > vB （去掉 v 前缀） */
function versionGt(vA: string, vB: string): boolean {
  const a = vA.replace(/^v/, '').split('.').map(Number);
  const b = vB.replace(/^v/, '').split('.').map(Number);
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    const ai = a[i] ?? 0;
    const bi = b[i] ?? 0;
    if (ai > bi) return true;
    if (ai < bi) return false;
  }
  return false;
}

/** 检查技能是否有新版本 */
function checkSkillUpdate(skill: RoleSkill): { hasUpdate: boolean; latestVersion?: string; updateNote?: string } {
  const info = getLatestVersionInfo(skill.name, skill.source);
  if (!info) return { hasUpdate: false };
  if (versionGt(info.latestVersion, skill.version)) {
    return { hasUpdate: true, latestVersion: info.latestVersion, updateNote: info.updateNote };
  }
  return { hasUpdate: false };
}

// ── 批量下发角色弹窗 ────────────────────────────────────────
/** 下发状态筛选选项 */
type DistributeFilterOption = 'pending_update' | 'updating' | 'success' | 'failed';

/** 多角色实例下命中目标角色的单个角色位（用于展开分别勾选） */
interface DistributeRoleSlot {
  /** 角色位唯一标识 */
  slotId: string;
  /** 角色位显示名（同名场景下自动追加序号，如「设计师 1 / 设计师 2」） */
  displayName: string;
  /** 该角色位当前的角色版本号 */
  distributedRoleVersion?: string;
  /** 显式更新状态 */
  roleUpdateStatus?: DistributeFilterOption;
}

interface DistributeAgentItem {
  id: string;
  name: string;
  instanceId: string;
  status: string;
  creator?: string;
  groupId?: string;
  groupName?: string;
  agentType?: string;
  /** Agent 最后一次被更新时的角色版本号（undefined=从未更新） */
  distributedRoleVersion?: string;
  /** 显式更新状态：若未设置则根据版本号推导 */
  roleUpdateStatus?: DistributeFilterOption;
  /**
   * 多角色实例下，本实例内命中目标角色的角色位列表。
   * 长度 > 1 时视为「多角色实例」，行前显示展开箭头，可展开后分别勾选各角色位。
   * 缺省 / 长度 ≤ 1 时退回单角色平铺语义，零回归。
   */
  matchedSlots?: DistributeRoleSlot[];
}

const DISTRIBUTE_PAGE_SIZE_OPTIONS = [20, 50, 100] as const;
const ALL_STATUS_KEYS: DistributeFilterOption[] = ['pending_update', 'updating', 'success', 'failed'];

const DISTRIBUTE_STATUS_MAP: Record<DistributeFilterOption, { label: string; variant: StatusTagColor }> = {
  pending_update: { label: '待更新',   variant: 'orange' },
  updating:       { label: '更新中',   variant: 'blue'   },
  success:        { label: '已更新',   variant: 'green'  },
  failed:         { label: '更新失败', variant: 'red'    },
};

function RoleDistributeDialog({
  open,
  roleName,
  roleVersion,
  agents,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  roleName: string;
  roleVersion: string;
  agents: DistributeAgentItem[];
  onConfirm: (selectedIds: string[]) => void;
  onCancel: () => void;
}) {
  // 状态
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilterKeys, setStatusFilterKeys] = useState<Set<DistributeFilterOption>>(
    new Set(['pending_update', 'failed'])
  );
  const [selectedInstances, setSelectedInstances] = useState<string[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState<number>(20);
  // 已展开的多角色实例 id 集合
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  // 刷新 key：递增触发实例状态重新拉取（mock 环境下仅触发重渲染）
  const [refreshKey, setRefreshKey] = useState(0);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const handleRefresh = () => {
    setIsRefreshing(true);
    setRefreshKey(k => k + 1);
    setTimeout(() => setIsRefreshing(false), 600);
  };

  // 初始化：打开时重置状态，并默认全选（选中默认筛选「待更新 / 更新失败」下的全部可勾选单元）
  useEffect(() => {
    if (open) {
      const defaultFilter = new Set<DistributeFilterOption>(['pending_update', 'failed']);
      setSearchQuery('');
      setStatusFilterKeys(defaultFilter);
      setExpandedIds(new Set());
      setCurrentPage(1);
      setPageSize(20);

      // 默认全选：与筛选/可勾选逻辑保持一致（内联计算，避免依赖后置定义的函数）。
      // 状态判定：显式 roleUpdateStatus 优先，否则按 distributedRoleVersion 与最新版本比较推导。
    const versionLt = (a: string, b: string): boolean => {
        const [am, an] = a.split('.').map(Number);
        const [bm, bn] = b.split('.').map(Number);
        return am !== bm ? am < bm : an < bn;
      };
      const statusOf = (agent: DistributeAgentItem): DistributeFilterOption => {
        if (agent.roleUpdateStatus) return agent.roleUpdateStatus;
        const dv = agent.distributedRoleVersion;
        if (!dv) return 'pending_update';
        return versionLt(dv, String(roleVersion)) ? 'pending_update' : 'success';
 };
      // 默认全选仅覆盖默认筛选命中的实例；单角色实例 key=实例 id，多角色实例 key=各角色位 slotId。
      const defaultSelected = agents
     .filter(agent => defaultFilter.has(statusOf(agent)))
        .flatMap(agent =>
  (agent.matchedSlots?.length ?? 0) >= 1
            ? agent.matchedSlots!.map(s => s.slotId)
      : [agent.id]
 );
      setSelectedInstances(Array.from(new Set(defaultSelected)));
  }
  }, [open, agents, roleVersion]);

  // x.y 版本号逐段比较
  const compareRoleVersion = (a: string, b: string): number => {
    const [am, an] = a.split('.').map(Number);
    const [bm, bn] = b.split('.').map(Number);
    if (am !== bm) return am - bm;
    return an - bn;
  };

  // 根据显式状态或 distributedRoleVersion 与 roleVersion 比较判定更新状态
  const getAgentDistributeKey = (agent: DistributeAgentItem): DistributeFilterOption => {
    if (agent.roleUpdateStatus) return agent.roleUpdateStatus;
    const dv = agent.distributedRoleVersion;
    if (!dv) return 'pending_update';
    if (compareRoleVersion(dv, String(roleVersion)) < 0) return 'pending_update';
    return 'success';
  };

  // 筛选逻辑：搜索 + 状态
  const filteredAgents = agents.filter(agent => {
    // 搜索匹配
    const matchSearch = !searchQuery ||
      agent.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      agent.instanceId.toLowerCase().includes(searchQuery.toLowerCase());
    // 状态匹配
    const matchStatus = statusFilterKeys.size === 0 || statusFilterKeys.has(
      getAgentDistributeKey(agent)
    );
    return matchSearch && matchStatus;
  });

  // 分页
  const totalCount = filteredAgents.length;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const safeCurrentPage = Math.min(currentPage, totalPages);
  const pagedAgents = filteredAgents.slice(
    (safeCurrentPage - 1) * pageSize, safeCurrentPage * pageSize
  );

  // 判定是否为「多角色实例」：只要生成了 matchedSlots（即多角色实例且命中目标角色，
  // 哪怕仅命中 1 个角色位），行前即展示展开箭头、可分别勾选。
  const isMultiRole = (agent: DistributeAgentItem): boolean =>
    (agent.matchedSlots?.length ?? 0) >= 1;

  // 命中目标角色的角色位数量（单角色实例记为 1），用于实例名旁的「（n）」标注
  const getMatchedCount = (agent: DistributeAgentItem): number =>
    isMultiRole(agent) ? agent.matchedSlots!.length : 1;

  // 判断角色位是否可勾选（更新中和已更新不可勾选）
  const isSlotSelectable = (slot: DistributeRoleSlot): boolean => {
    const status: DistributeFilterOption = slot.roleUpdateStatus
      ? slot.roleUpdateStatus
      : (!slot.distributedRoleVersion || compareRoleVersion(slot.distributedRoleVersion, String(roleVersion)) < 0)
        ? 'pending_update'
        : 'success';
    return status === 'pending_update' || status === 'failed';
  };

  // 该实例下所有「可勾选单元」的 key：只包含可更新状态（待更新/更新失败）的角色位
  const getSelectableKeys = (agent: DistributeAgentItem): string[] =>
    isMultiRole(agent)
      ? agent.matchedSlots!.filter(s => isSlotSelectable(s)).map(s => s.slotId)
      : ((() => {
          // 单角色实例：判断实例自身状态是否可勾选
          const status: DistributeFilterOption = agent.roleUpdateStatus
            ? agent.roleUpdateStatus
            : (!agent.distributedRoleVersion || compareRoleVersion(agent.distributedRoleVersion, String(roleVersion)) < 0)
              ? 'pending_update'
              : 'success';
          return (status === 'pending_update' || status === 'failed') ? [agent.id] : [];
        })());

  // 当前页所有可勾选单元 key（多角色实例展开为各角色位）
  const pageSelectableKeys = pagedAgents.flatMap(getSelectableKeys);
  const isPageAllSelected =
    pageSelectableKeys.length > 0 && pageSelectableKeys.every(k => selectedInstances.includes(k));
  const isPageIndeterminate =
    !isPageAllSelected && pageSelectableKeys.some(k => selectedInstances.includes(k));

  // 全部可勾选单元 key（用于确认计数，仅统计筛选后仍存在的单元）
  const allSelectableKeys = new Set(filteredAgents.flatMap(getSelectableKeys));
  const selectedSelectableCount = selectedInstances.filter(k => allSelectableKeys.has(k)).length;

  // 已展开的多角色实例：展开/收起切换
  const toggleExpand = (id: string) => {
    setExpandedIds(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  // 全选/取消当前页（多角色实例按其全部角色位一并处理）
  const handleSelectAll = () => {
    if (isPageAllSelected) {
      setSelectedInstances(prev => prev.filter(k => !pageSelectableKeys.includes(k)));
    } else {
      setSelectedInstances(prev => Array.from(new Set([...prev, ...pageSelectableKeys])));
    }
  };

  // 单个可勾选单元（实例 id 或角色位 slotId）勾选/取消
  const handleSelectKey = (key: string) => {
    setSelectedInstances(prev =>
      prev.includes(key) ? prev.filter(x => x !== key) : [...prev, key]
    );
  };

  // 多角色实例父行勾选：一键选中/取消其全部角色位
  const handleSelectAgentRoles = (agent: DistributeAgentItem) => {
    const keys = getSelectableKeys(agent);
    const allChecked = keys.every(k => selectedInstances.includes(k));
    setSelectedInstances(prev =>
      allChecked
        ? prev.filter(k => !keys.includes(k))
        : Array.from(new Set([...prev, ...keys]))
    );
  };

  // 多角色实例父行勾选态（全选 / 半选 / 未选）
  const getAgentCheckState = (agent: DistributeAgentItem): boolean | "indeterminate" => {
    const keys = getSelectableKeys(agent);
    const checkedCount = keys.filter(k => selectedInstances.includes(k)).length;
    if (checkedCount === 0) return false;
    if (checkedCount === keys.length) return true;
    return "indeterminate";
  };

  // 获取实例的状态标签
  const getStatusDisplay = (agent: DistributeAgentItem) => {
    const status = getAgentDistributeKey(agent);
    const mapEntry = DISTRIBUTE_STATUS_MAP[status];
    const label =
      status === 'pending_update' && agent.distributedRoleVersion
        ? `待更新 v${agent.distributedRoleVersion}`
        : mapEntry.label;
    return (
      <StatusTag variant={mapEntry.variant} mode="fill">
        {label}
      </StatusTag>
    );
  };

  // 单个角色位的状态标签
  const getSlotStatusDisplay = (slot: DistributeRoleSlot) => {
    const status: DistributeFilterOption = slot.roleUpdateStatus
      ? slot.roleUpdateStatus
      : (!slot.distributedRoleVersion || compareRoleVersion(slot.distributedRoleVersion, String(roleVersion)) < 0)
        ? 'pending_update'
        : 'success';
    const mapEntry = DISTRIBUTE_STATUS_MAP[status];
    const label =
      status === 'pending_update' && slot.distributedRoleVersion
        ? `待更新 v${slot.distributedRoleVersion}`
        : mapEntry.label;
    return (
      <StatusTag variant={mapEntry.variant} mode="fill" icon={status === 'updating' ? <Loader2 className="size-3 animate-spin" /> : undefined}>
        {label}
      </StatusTag>
    );
  };


  const handleDistribute = () => {
    // 归约为去重后的实例 id 列表（外层按实例 id 更新版本，语义兼容单/多角色）
    const selectedAgentIds = filteredAgents
      .filter(a => getSelectableKeys(a).some(k => selectedInstances.includes(k)))
      .map(a => a.id);
    onConfirm(Array.from(new Set(selectedAgentIds)));
    setSelectedInstances([]);
    setExpandedIds(new Set());
    setCurrentPage(1);
  };

  const handleCancel = () => {
    setSelectedInstances([]);
    setExpandedIds(new Set());
    setCurrentPage(1);
    onCancel();
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) handleCancel(); }}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>批量更新角色「{roleName}」</DialogTitle>
          <DialogDescription className="sr-only">
            向运行中的实例批量更新角色「{roleName}」的最新版本
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="px-6 pb-1">
        {/* 说明文字（从 Header 移入内容区） */}
        <BodyText tone="muted" className="mb-4">
          仅支持向 <BodyMedium as="span" tone="secondary">运行中</BodyMedium> 的实例更新角色，
          可批量自动更新「<BodyMedium as="span" tone="secondary">{roleName}</BodyMedium>」最新版本；
          多角色实例下命中多个角色时，可展开分别勾选。
        </BodyText>

        {/* 工具栏：搜索框 + 状态筛选 */}
        <div className="flex gap-2 mb-4">
          {/* 搜索框 */}
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-muted)]" />
            <Input
              placeholder="搜索实例名称/ID..."
              value={searchQuery}
              onChange={(e) => { setSearchQuery(e.target.value); setCurrentPage(1); }}
              className="pl-10"
            />
          </div>

          {/* 状态多选下拉（标准 InstantMultiSelect，Portal 实现） */}
          <InstantMultiSelect
            triggerClassName="w-44 flex-shrink-0"
            placeholder="全部更新状态"
            searchable={false}
            selectAllLabel="全部更新状态"
            value={statusFilterKeys as unknown as Set<string>}
            onChange={(next) => {
              setStatusFilterKeys(next as unknown as Set<DistributeFilterOption>);
              setCurrentPage(1);
            }}
            options={ALL_STATUS_KEYS.map((key) => ({
              value: key,
              label: DISTRIBUTE_STATUS_MAP[key].label,
            }))}
          />
          {/* 刷新按钮：刷新实例状态 */}
          <button
            type="button"
            className="flex-shrink-0 inline-flex items-center justify-center size-9 rounded-[4px] border border-[var(--border)] bg-white text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-grey-normal,#F5F7FA)]"
            onClick={handleRefresh}
            disabled={isRefreshing}
            aria-label="刷新实例状态"
          >
            <RefreshCw className={`size-4 ${isRefreshing ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {/* 实例列表（压缩表格 density="compact"，行高 40px；多角色实例可展开分别勾选） */}
        <div className="border border-[var(--border)] rounded-[8px] overflow-hidden">
          <div className="max-h-[340px] overflow-y-auto">
          <Table density="compact" autoFixedColumns={false}>
            <TableHeader className="sticky top-0 z-10">
              <TableRow>
                <TableHead className="w-10">
                  <Checkbox
                    checked={isPageAllSelected ? true : isPageIndeterminate ? "indeterminate" : false}
                    onCheckedChange={handleSelectAll}
                    aria-label="全选"
                  />
                </TableHead>
                <TableHead className="w-7 !px-0"></TableHead>
                <TableHead>实例 / 角色</TableHead>
                <TableHead className="text-right w-28">更新状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pagedAgents.length === 0 ? (
                <TableRow className="hover:!bg-transparent">
                  <TableCell colSpan={4} className="!h-auto">
                    <div className="text-center py-12">
                      <HelperText>暂无匹配的实例</HelperText>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                pagedAgents.map((agent) => {
                  const multi = isMultiRole(agent);
                  const expanded = expandedIds.has(agent.id);
                  const matchedCount = getMatchedCount(agent);
                  // 状态汇总：按状态分组统计角色位数量，取最高优先级的状态展示
                  const slotStatuses = multi
                    ? agent.matchedSlots!.map((s) =>
                        s.roleUpdateStatus
                          ? s.roleUpdateStatus
                          : (!s.distributedRoleVersion || compareRoleVersion(s.distributedRoleVersion, String(roleVersion)) < 0)
                            ? 'pending_update' as const
                            : 'success' as const
                      )
                    : [getAgentDistributeKey(agent)];
                  // 优先级：failed > updating > pending_update > success
                  const statusPriority: DistributeFilterOption[] = ['failed', 'updating', 'pending_update', 'success'];
                  return (
                    <Fragment key={agent.id}>
                      {/* ── 实例行（父行） ── */}
                      <TableRow
                        className={multi ? "cursor-pointer" : "cursor-default"}
                        onClick={multi ? () => toggleExpand(agent.id) : undefined}
                      >
                        <TableCell onClick={(e) => e.stopPropagation()}>
                          <Checkbox
                            checked={multi ? getAgentCheckState(agent) : selectedInstances.includes(agent.id)}
                            onCheckedChange={getSelectableKeys(agent).length > 0 ? () => (multi ? handleSelectAgentRoles(agent) : handleSelectKey(agent.id)) : undefined}
                            disabled={getSelectableKeys(agent).length === 0}
                            aria-label="选择该实例"
                          />
                        </TableCell>
                        {/* 展开箭头列 */}
                        <TableCell className="!px-0 text-center">
                          {multi ? (
                            <button
                              type="button"
                              className="inline-flex items-center justify-center size-5 text-[var(--text-muted)] transition-colors hover:text-[var(--text-emphasis)]"
                              onClick={(e) => { e.stopPropagation(); toggleExpand(agent.id); }}
                              aria-label={expanded ? '折叠角色位' : '展开角色位'}
                              aria-expanded={expanded}
                            >
                              <ChevronRight
                                className={`size-4 transition-transform ${expanded ? 'rotate-90' : ''}`}
                                aria-hidden
                              />
                            </button>
                          ) : null}
                        </TableCell>
                        <TableCell>
                          <div className="min-w-0 flex flex-col gap-0.5">
                            {/* 实例名(N) + 实例ID 同行 */}
                              <div className="flex items-baseline gap-2 min-w-0">
                                <BodyMedium className="truncate shrink-0">
                                  {agent.name}
                                  {multi && <span className="ml-1">（{matchedCount}）</span>}
                                </BodyMedium>
                                {agent.instanceId && (
                                  <MetaText tone="weak" className="truncate shrink">
                                    {agent.instanceId}
                                  </MetaText>
                                )}
                              </div>
                              {(() => {
                                const infoText = [
                                  agent.creator && `创建人: ${agent.creator}`,
                                  agent.groupName && `分组: ${agent.groupName}`,
                                ].filter(Boolean).join(' | ');
                                return infoText ? (
                                  <MetaText tone="weak" className="truncate cursor-default">
                                    {infoText}
                                  </MetaText>
                                ) : null;
                              })()}
                            </div>
                        </TableCell>
                        {/* 更新状态列 */}
                        <TableCell className="text-right align-middle">
                          {(() => {
                              // 待更新 + 更新失败 = 可更新项
                              const updatableCount = slotStatuses.filter(
                                (s) => s === 'pending_update' || s === 'failed'
                              ).length;
                              return updatableCount > 0
                                ? (
                                  <span className="inline-flex items-center gap-1 whitespace-nowrap">
                                    <AlertCircle className="size-3.5 text-[var(--text-warning)]" />
                                    <BodyMedium as="span" tone="secondary" className="tabular-nums">{updatableCount}项可更新</BodyMedium>
                                  </span>
                                )
                                : <MetaText tone="secondary" className="tabular-nums whitespace-nowrap">无可更新项</MetaText>;
                            })()}
                        </TableCell>
                      </TableRow>

                      {/* ── 展开后的角色位行（浅灰卡片） ── */}
                      {multi && expanded && (
                        <TableRow className="hover:!bg-transparent">
                          <TableCell colSpan={4} className="!py-0 !px-0 !border-b-0">
                            <div className="py-3 px-3 bg-[var(--bg-grey-normal)]">
                              {/* 白色卡片：左边距对齐箭头列起始（checkbox列宽40px + cell内padding约12px = ~52px，再减去卡片自身无 pl） */}
                              <div className="rounded-[8px] border border-[var(--border)] bg-white overflow-hidden" style={{ marginLeft: '40px' }}>
                              {agent.matchedSlots!.map((slot) => {
                                const selectable = isSlotSelectable(slot);
                                return (
                                <div
                                  key={slot.slotId}
                                  role="button"
                                  tabIndex={selectable ? 0 : undefined}
                                  className={`flex items-center pl-0 pr-3 py-2 border-b border-gray-100 last:border-b-0 transition-colors ${
                                    selectable
                                      ? 'cursor-pointer hover:bg-[var(--bg-grey-hover)]'
                                      : 'cursor-default'
                                  }`}
                                  onClick={selectable ? () => handleSelectKey(slot.slotId) : undefined}
                                >
                                  {/* checkbox 居中对齐箭头列（箭头列 w-7=28px, !px-0） */}
                                  <span
                                    className="w-7 shrink-0 flex items-center justify-center"
                                    onClick={(e) => e.stopPropagation()}
                                  >
                                    <Checkbox
                                      checked={selectable && selectedInstances.includes(slot.slotId)}
                                      onCheckedChange={selectable ? () => handleSelectKey(slot.slotId) : undefined}
                                      disabled={!selectable}
                                      aria-label="选择该角色位"
                                    />
                                  </span>
                                  {/* 角色名称与实例名称左对齐（紧跟 w-7 后无额外间距，对应 col3 起始） */}
                                  <BodyMedium className="flex-1 min-w-0 truncate ml-1">{slot.displayName}</BodyMedium>
                                  <span className="shrink-0 ml-2">
                                    {getSlotStatusDisplay(slot)}
                                  </span>
                                </div>
                                );
                              })}
                              </div>
                            </div>
                          </TableCell>
                        </TableRow>
                      )}
                    </Fragment>
                  );
                })
              )}
            </TableBody>
          </Table>
          </div>
        </div>

        {/* 分页控件 — 两段式布局：总数左 + 分页器右（规范 §12.5 标准表格底部布局） */}
        <div className="grid grid-cols-[1fr_auto] items-center gap-4 pt-2">
          <MetaText className="justify-self-start">
            共{allSelectableKeys.size}个可选项，已选{selectedSelectableCount}项
          </MetaText>
          <Pagination
            total={totalCount}
            current={safeCurrentPage}
            pageSize={pageSize}
            size="small"
            mode="simple"
            showTotal={false}
            showSizeChanger
            pageSizeOptions={DISTRIBUTE_PAGE_SIZE_OPTIONS}
            className="justify-self-end justify-end flex-nowrap"
            onChange={(page, size) => {
              if (size !== pageSize) {
                setPageSize(size);
                setCurrentPage(1);
                setSelectedInstances([]);
              } else {
                setCurrentPage(page);
              }
            }}
          />
        </div>
        </DialogBody>

        <DialogFooter className="gap-2 sm:gap-2">
          <Button variant="outline" onClick={handleCancel}>取消</Button>
          <Button
            onClick={handleDistribute}
            disabled={selectedSelectableCount === 0}
          >
            确认更新（{selectedSelectableCount}）
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── 角色更新记录弹窗 ────────────────────────────────────────

const HISTORY_PAGE_SIZE_OPTIONS = [20, 50, 100] as const;

const HISTORY_STATUS_MAP: Record<string, { label: string; color: string }> = {
  success: { label: '全部成功', color: 'text-green-700 bg-green-50' },
  partial: { label: '部分成功', color: 'text-yellow-700 bg-yellow-50' },
  failed:  { label: '失败',     color: 'text-red-700 bg-red-50' },
};

function RoleUpdateHistoryDialog({
  open,
  onClose,
  role,
}: {
  open: boolean;
  onClose: () => void;
  role: Role | null;
}) {
  const [historyPage, setHistoryPage] = useState(1);
  const [historyPageSize, setHistoryPageSize] = useState<number>(20);

  useEffect(() => {
    if (open) {
      setHistoryPage(1);
      setHistoryPageSize(20);
    }
  }, [open]);

  if (!role) return null;
  const records = role.updateHistory || [];
  const totalHistory = records.length;
  const totalPages = Math.max(1, Math.ceil(totalHistory / historyPageSize));
  const safePage = Math.min(historyPage, totalPages);
  const paged = records.slice((safePage - 1) * historyPageSize, safePage * historyPageSize);

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent className="max-w-3xl" data-billing-exempt>
        <DialogHeader>
          <DialogTitle>
            「{role.name}」更新记录
          </DialogTitle>
          <DialogDescription className="sr-only">
            角色「{role.name}」的历史下发更新记录
          </DialogDescription>
        </DialogHeader>

        {records.length === 0 ? (
          <Empty className="border-0 py-8">
            <EmptyHeader>
              <EmptyMedia />
              <EmptyTitle>暂无更新记录</EmptyTitle>
              <EmptyDescription>该角色尚未进行过版本下发更新</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <>
            <div className="overflow-auto border border-[#EAEEF4] rounded-[4px] max-h-[360px]">
              <Table variant="white">
                <TableHeader>
                  <TableRow>
                    <TableHead>更新时间</TableHead>
                    <TableHead>版本号</TableHead>
                    <TableHead>操作人</TableHead>
                    <TableHead className="text-right">推送总数</TableHead>
                    <TableHead className="text-right">成功</TableHead>
                    <TableHead className="text-right">失败</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="min-w-[200px]">原因</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paged.map((r) => {
                    const st = HISTORY_STATUS_MAP[r.status] || { label: r.status, color: 'text-gray-500 bg-gray-50' };
                    return (
                      <TableRow key={r.id}>
                        <TableCell>
                          <MetaText>{r.operatedAt}</MetaText>
                        </TableCell>
                        <TableCell>
                          <BodyMedium>v{r.version}</BodyMedium>
                        </TableCell>
                        <TableCell>
                          <MetaText tone="muted">{r.operator}</MetaText>
                        </TableCell>
                        <TableCell className="text-right">
                          <MetaText>{r.totalCount}</MetaText>
                        </TableCell>
                        <TableCell className="text-right">
                          <MetaText tone="success">{r.successCount}</MetaText>
                        </TableCell>
                        <TableCell className="text-right">
                          <MetaText tone={r.failedCount > 0 ? 'danger' : 'muted'}>{r.failedCount}</MetaText>
                        </TableCell>
                        <TableCell>
                          <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${st.color}`}>
                            {st.label}
                          </span>
                        </TableCell>
                        <TableCell>
                          <MetaText>{r.reason}</MetaText>
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>

            {/* 分页 */}
            <div className="flex items-center justify-between mt-3 h-8">
              <div className="flex items-center gap-1.5">
                <span className="text-xs text-gray-500">共 {totalHistory} 条，每页</span>
                <select
                  className="h-7 px-1 text-xs border border-gray-200 rounded bg-white"
                  value={String(historyPageSize)}
                  onChange={(e) => { setHistoryPageSize(Number(e.target.value)); setHistoryPage(1); }}
                >
                  {HISTORY_PAGE_SIZE_OPTIONS.map(size => (
                    <option key={size} value={String(size)}>{size}</option>
                  ))}
                </select>
                <span className="text-xs text-gray-500">条</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  disabled={safePage <= 1}
                  onClick={() => setHistoryPage(p => Math.max(1, p - 1))}
                >上一页</Button>
                <span className="px-2 text-xs text-gray-600">{safePage} / {totalPages}</span>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  disabled={safePage >= totalPages}
                  onClick={() => setHistoryPage(p => Math.min(totalPages, p + 1))}
                >下一页</Button>
              </div>
            </div>
          </>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>关闭</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── 批量更新弹窗 ──────────────────────────────────────────
interface UpdatableSkill {
  index: number;
  skill: RoleSkill;
  latestVersion: string;
  updateNote: string;
}

const PAGE_SIZE_OPTIONS = [20, 50, 100, 200, 500] as const;

function BatchUpdateDialog({
  open,
  updatableSkills,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  updatableSkills: UpdatableSkill[];
  onConfirm: (selectedIndices: number[]) => void;
  onCancel: () => void;
}) {
  const [selectedIndices, setSelectedIndices] = useState<Set<number>>(new Set());
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState<number>(20);

  // 初始化：默认选中全部
  useEffect(() => {
    if (open) {
      setCurrentPage(1);
      setPageSize(20);
      setSelectedIndices(new Set(updatableSkills.map(s => s.index)));
    }
  }, [open, updatableSkills]);

  const totalPages = Math.max(1, Math.ceil(updatableSkills.length / pageSize));
  const pagedSkills = updatableSkills.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  // 当前页全选
  const currentPageIndices = pagedSkills.map(s => s.index);
  const allPageSelected = currentPageIndices.length > 0 && currentPageIndices.every(idx => selectedIndices.has(idx));

  const toggleAll = () => {
    setSelectedIndices(prev => {
      const next = new Set(prev);
      if (allPageSelected) {
        currentPageIndices.forEach(idx => next.delete(idx));
      } else {
        currentPageIndices.forEach(idx => next.add(idx));
      }
      return next;
    });
  };

  const toggleOne = (idx: number) => {
    setSelectedIndices(prev => {
      const next = new Set(prev);
      if (next.has(idx)) next.delete(idx);
      else next.add(idx);
      return next;
    });
  };

  const handleConfirm = () => {
    onConfirm(Array.from(selectedIndices));
    setSelectedIndices(new Set());
    setCurrentPage(1);
  };

  const handleCancel = () => {
    setSelectedIndices(new Set());
    setCurrentPage(1);
    onCancel();
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) handleCancel(); }}>
      <DialogContent
        size="lg"
        style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
      >
        <DialogHeader>
          <DialogTitle>批量刷新技能版本</DialogTitle>
          <DialogDescription className="sr-only">
            选择需要刷新到最新版本的技能
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="px-6 min-h-0">
          {updatableSkills.length === 0 ? (
            <Empty className="border-0">
              <EmptyHeader>
                <EmptyMedia />
                <EmptyTitle>所有技能均为最新版本</EmptyTitle>
                <EmptyDescription>当前没有可更新的技能</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="flex flex-col gap-3">
              {/* 弹窗内压缩表格：直接用 Table 自带 containerClassName 描边，禁止再套 div（参考 Table §1 / §8 + DispatchCommandDialog 规范案例） */}
              <Table
                density="compact"
                autoFixedColumns={false}
                containerClassName="border border-gray-200 rounded-[4px] overflow-hidden bg-white"
              >
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[44px]">
                      <Checkbox
                        checked={allPageSelected}
                        onCheckedChange={toggleAll}
                        aria-label={allPageSelected ? '取消全选' : '全选当前页'}
                      />
                    </TableHead>
                    <TableHead className="w-[26%]">技能名称</TableHead>
                    <TableHead className="w-[8%]">类型</TableHead>
                    <TableHead className="w-[12%]">新版本</TableHead>
                    <TableHead className="w-[12%]">原版本</TableHead>
                    <TableHead className="w-[34%]">更新说明</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pagedSkills.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={6}>
                        <div className="text-center py-10">
                          <HelperText>暂无可更新的技能</HelperText>
                        </div>
                      </TableCell>
                    </TableRow>
                  ) : (
                    pagedSkills.map((item) => {
                      const checked = selectedIndices.has(item.index);
                      return (
                        <TableRow
                          key={item.index}
                          data-state={checked ? 'selected' : undefined}
                          onClick={() => toggleOne(item.index)}
                          className="cursor-pointer"
                        >
                          <TableCell>
                            <Checkbox
                              checked={checked}
                              onCheckedChange={() => toggleOne(item.index)}
                              onClick={(e) => e.stopPropagation()}
                            />
                          </TableCell>
                          <TableCell className="font-medium">
                            <span className="block truncate">{item.skill.name}</span>
                          </TableCell>
                          <TableCell>
                            <StatusTag mode="fill" variant={item.skill.source === '公共' ? 'blue' : 'gray'}>
                              {item.skill.source}
                            </StatusTag>
                          </TableCell>
                          <TableCell>{item.latestVersion}</TableCell>
                          <TableCell className="text-[#737373]">{item.skill.version}</TableCell>
                          <TableCell className="whitespace-normal">{item.updateNote}</TableCell>
                        </TableRow>
                      );
                    })
                  )}
                </TableBody>
              </Table>

              {/* 分页控件 — 与设计分支主流页面统一使用默认尺寸（28px）*/}
              <Pagination
                total={updatableSkills.length}
                current={currentPage}
                pageSize={pageSize}
                size="default"
                showTotal={(total) =>
                  selectedIndices.size > 0
                    ? `共 ${total} 条，已选 ${selectedIndices.size} 条记录`
                    : `共 ${total} 条`
                }
                showSizeChanger
                pageSizeOptions={PAGE_SIZE_OPTIONS}
                className="w-full justify-between"
                onChange={(page, newPageSize) => {
                  if (newPageSize !== pageSize) {
                    setPageSize(newPageSize);
                    setCurrentPage(1);
                  } else {
                    setCurrentPage(page);
                  }
                }}
              />
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel}>取消</Button>
          <Button
            variant="dialog-confirm"
            onClick={handleConfirm}
            disabled={selectedIndices.size === 0}
          >
            确认刷新{selectedIndices.size > 0 ? `（${selectedIndices.size} 个）` : ''}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Sortable Row ────────────────────────────────────────
function SortableRoleRow({
  role,
  hasUpdatable,
  onToggle,
  onEdit,
  onDelete,
  onScopeChange,
  onDistribute,
  onShowHistory,
}: {
  role: Role;
  hasUpdatable: boolean;
  onToggle: (id: string) => void;
  onEdit: (role: Role) => void;
  onDelete: (role: Role) => void;
  onScopeChange: (id: string, scope: SkillScope, groupIds: string[]) => void;
  onDistribute: (role: Role) => void;
  onShowHistory: (role: Role) => void;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: role.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    zIndex: isDragging ? 10 : undefined,
    opacity: isDragging ? 0.9 : 1,
  };

  return (
    <TableRow
      ref={setNodeRef}
      style={style}
      className={isDragging ? "bg-blue-50/30 shadow-sm" : undefined}
    >
      {/* Drag Handle */}
      <TableCell className="w-10 px-3">
        <button
          {...attributes}
          {...listeners}
          className="p-1 rounded text-[var(--text-weak)] hover:text-[var(--text-muted)] cursor-grab active:cursor-grabbing"
        >
          <GripVertical className="w-4 h-4" />
        </button>
      </TableCell>
      {/* Name — clickable to open edit */}
      <TableCell>
        <button
          onClick={() => onEdit(role)}
          className="hover:underline transition-colors text-left"
        >
          <BodyMedium className="hover:text-[var(--text-brand)] transition-colors">
            {role.name}
          </BodyMedium>
        </button>
      </TableCell>
      {/* Description */}
      <TableCell className="max-w-[320px]">
        <BodyText tone="muted" className="truncate block">{role.description}</BodyText>
      </TableCell>
      {/* 应用范围 */}
      <TableCell>
        <EditRoleScopePopover
          role={role}
          onConfirm={(scope, groupIds) => onScopeChange(role.id, scope, groupIds)}
        />
      </TableCell>
      {/* Visible toggle */}
      <TableCell>
        <Switch
          checked={role.visible}
          onCheckedChange={() => onToggle(role.id)}
        />
      </TableCell>
      {/* Version — 参考企业技能库列表样式 BodyText tone="secondary" */}
      <TableCell>
        <BodyText as="span" tone="secondary">v{role.version}</BodyText>
      </TableCell>
      {/* Actions */}
      <TableActionCell className="w-[170px]">
        <Button variant="link" onClick={() => onEdit(role)}>编辑</Button>
        {hasUpdatable ? (
          <Button variant="link" onClick={() => onDistribute(role)}>更新</Button>
        ) : (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="inline-flex cursor-not-allowed">
                <Button variant="link" disabled className="pointer-events-none">更新</Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>当前无可更新项</TooltipContent>
          </Tooltip>
        )}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="link"
              className="text-[var(--brand,#1447E6)]"
              data-billing-exempt
            >
              更多
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-44">
            <DropdownMenuItem
              onClick={() => onShowHistory(role)}
              data-billing-exempt
            >
              <History className="w-4 h-4 mr-2 text-gray-500" />
              更新记录
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => onDelete(role)}
              className="text-red-600 focus:text-red-600"
            >
              <Trash2 className="w-4 h-4 mr-2 text-red-600" />
              删除
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </TableActionCell>
    </TableRow>
  );
}

// ── 公共技能库添加弹窗（与初始技能包交互一致）──────────────
// Tab1「公共技能」：单个 Skill 多选；Tab2「公共技能包」：多选包，提交时展开为多个 Skill
const MOCK_FAVORITES: PublicSkill[] = PUBLIC_SKILLS.slice(0, 5);

// ── 公共技能包弹窗 helpers（适配主干 PublicSkillPackage 结构） ──
// mock "我的收藏" —— 取前 4 个包模拟用户已收藏（后续接入全局收藏 store 时替换此处）
const MOCK_FAVORITE_PKG_IDS = new Set(
  PUBLIC_SKILL_PACKAGES.slice(0, 4).map(p => p.id)
);

/** 获取用户收藏的技能包列表（mock 实现，后续替换为全局 store 查询） */
function getFavoritePackages(): PublicSkillPackage[] {
  return PUBLIC_SKILL_PACKAGES.filter(p => MOCK_FAVORITE_PKG_IDS.has(p.id));
}

/** 将技能包展开为 RoleSkill[]（与单个 PublicSkill → RoleSkill 规则一致） */
function toRoleSkills(pkg: PublicSkillPackage): RoleSkill[] {
  return pkg.skills.map(ref => ({
    name: ref.name,
    version: "v1.0",
    source: "公共" as const,
  }));
}

/** 获取技能包内所有 Skill 的展示名（用于"N 个已存在"徽章计算） */
function getPackageSkillNames(pkg: PublicSkillPackage): string[] {
  return pkg.skills.map(ref => ref.name);
}

type PublicAddSubTab = "skill" | "package";

function RoleAddPublicSkillDialog({
  open,
  existingSkillNames,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  existingSkillNames: string[];
  /**
   * 确认提交：
   *  - skills：本次新增到角色技能列表的 RoleSkill 集合（已合并、已按 name 去重，不与现有技能重复）
   *  - packageIds：本次勾选的公共技能包 id（用于来源追溯，可选）
   */
  onConfirm: (skills: RoleSkill[], packageIds?: string[]) => void;
  onCancel: () => void;
}) {
  const [activeSubTab, setActiveSubTab] = useState<PublicAddSubTab>("skill");
  const [selectedSkillIds, setSelectedSkillIds] = useState<string[]>([]);
  const [selectedPackageIds, setSelectedPackageIds] = useState<string[]>([]);

  const toggleSkill = (skillId: string) => {
    setSelectedSkillIds(prev =>
      prev.includes(skillId) ? prev.filter(id => id !== skillId) : [...prev, skillId]
    );
  };

  const togglePackage = (pkgId: string) => {
    setSelectedPackageIds(prev =>
      prev.includes(pkgId) ? prev.filter(id => id !== pkgId) : [...prev, pkgId]
    );
  };

  const resetSelections = () => {
    setSelectedSkillIds([]);
    setSelectedPackageIds([]);
    setActiveSubTab("skill");
  };

  const handleConfirm = () => {
    // 1) 单个公共技能 → RoleSkill[]
    const skillRoleSkills: RoleSkill[] = selectedSkillIds.map(id => {
      const skill = MOCK_FAVORITES.find(s => s.id === id)!;
      return { name: skill.name, version: `v${skill.version}`, source: "公共" as const };
    });

    // 2) 公共技能包 → 展开为 RoleSkill[]（每个包内部各自展开）
    const selectedPackages = selectedPackageIds
      .map(id => PUBLIC_SKILL_PACKAGES.find(p => p.id === id))
      .filter((p): p is PublicSkillPackage => Boolean(p));
    const packageRoleSkills: RoleSkill[] = selectedPackages.flatMap(toRoleSkills);

    // 3) 合并 + 按 name 去重（同 name 优先保留单技能侧的选择，再按出现顺序保留首项）
    //    同时排除掉角色技能列表里已经存在的同名 Skill
    const existingSet = new Set(existingSkillNames);
    const merged: RoleSkill[] = [];
    const seen = new Set<string>();
    for (const s of [...skillRoleSkills, ...packageRoleSkills]) {
      if (existingSet.has(s.name) || seen.has(s.name)) continue;
      seen.add(s.name);
      merged.push(s);
    }

    onConfirm(merged, selectedPackageIds.length > 0 ? [...selectedPackageIds] : undefined);
    resetSelections();
  };

  const handleCancel = () => {
    resetSelections();
    onCancel();
  };

  // 底部按钮文案与可用性
  const isSkillTab = activeSubTab === "skill";
  const currentCount = isSkillTab ? selectedSkillIds.length : selectedPackageIds.length;
  const confirmText = isSkillTab
    ? `确认添加${currentCount > 0 ? `（${currentCount} 个）` : ""}`
    : `确认添加技能包${currentCount > 0 ? `（${currentCount} 个）` : ""}`;

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) handleCancel(); }}>
      <DialogContent size="xl" style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}>
        <DialogHeader>
          <DialogTitle>从公共技能库添加</DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
          <div className="pb-3">
            <BodyMedium className="flex items-center gap-1.5">我的收藏</BodyMedium>
          </div>

          {MOCK_FAVORITES.length === 0 ? (
            <Empty className="border-0">
              <EmptyHeader>
                <EmptyMedia />
                <EmptyTitle>还没有收藏任何技能</EmptyTitle>
                <EmptyDescription>可先前往公共技能库收藏技能</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <div className="grid grid-cols-2 gap-3">
              {MOCK_FAVORITES.map(skill => {
                const isAlreadyAdded = existingSkillNames.includes(skill.name);
                const isSelected = selectedSkillIds.includes(skill.id);
                return (
                  <SkillSelectCard
                    key={skill.id}
                    name={skill.name}
                    version={skill.version}
                    description={skill.descriptionZh}
                    state={isAlreadyAdded ? "disabled" : isSelected ? "selected" : "default"}
                    onClick={() => toggleSkill(skill.id)}
                  />
                );
              })}
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel}>取消</Button>
          <Button
            variant="dialog-confirm"
            onClick={handleConfirm}
            disabled={currentCount === 0}
          >
            {confirmText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── 企业技能库添加弹窗（支持应用范围筛选）──────────────

function RoleAddEnterpriseSkillDialog({
  open,
  existingSkillNames,
  onConfirm,
  onCancel,
  /** 当前角色的应用范围，用于预设筛选 */
  roleScope,
  roleGroupIds,
}: {
  open: boolean;
  existingSkillNames: string[];
  onConfirm: (skills: RoleSkill[]) => void;
  onCancel: () => void;
  roleScope?: SkillScope;
  roleGroupIds?: string[];
}) {
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [activeCategory, setActiveCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  // 应用范围多选筛选
  const [scopeFilters, setScopeFilters] = useState<string[]>([]);
  const [scopeDropdownOpen, setScopeDropdownOpen] = useState(false);
  const [scopeSearchQuery, setScopeSearchQuery] = useState('');

  // 打开时根据角色应用范围预设筛选
  // 规则：【全部用户】默认必勾选
  // 如果角色是【全部用户】的，只勾选【全部用户】，不再多勾其他
  // 如果角色不是【全部用户】的，勾选【全部用户】+ 该角色关联的组织
  useEffect(() => {
    if (open) {
      if (roleScope === 'public' || !roleGroupIds || roleGroupIds.length === 0) {
        // 全部用户的角色：只勾选【全部用户】
        setScopeFilters(['__public__']);
      } else {
        // 非全部用户的角色：勾选【全部用户】+ 关联组织
        setScopeFilters(['__public__', ...roleGroupIds]);
      }
      setScopeDropdownOpen(false);
      setScopeSearchQuery('');
    }
  }, [open, roleScope, roleGroupIds]);

  const toggleSkill = (skillId: string) => {
    setSelectedIds(prev =>
      prev.includes(skillId) ? prev.filter(id => id !== skillId) : [...prev, skillId]
    );
  };

  const handleConfirm = () => {
    const newSkills: RoleSkill[] = selectedIds.map(id => {
      const skill = MOCK_SKILLS.find(s => s.id === id)!;
      return { name: skill.name, version: `v${skill.version}`, source: "企业" as const };
    });
    onConfirm(newSkills);
    setSelectedIds([]);
    setActiveCategory('all');
    setSearchQuery('');
    setScopeFilters([]);
  };

  const handleCancel = () => {
    setSelectedIds([]);
    setActiveCategory('all');
    setSearchQuery('');
    setScopeFilters([]);
    setScopeDropdownOpen(false);
    setScopeSearchQuery('');
    onCancel();
  };

  const handleRefresh = () => {
    setSearchQuery('');
    setActiveCategory('all');
    setSelectedIds([]);
    setScopeFilters([]);
    setScopeDropdownOpen(false);
    setScopeSearchQuery('');
  };

  const filteredSkills = MOCK_SKILLS.filter(s => {
    const matchCategory = activeCategory === 'all' || s.categories.includes(activeCategory);
    const q = searchQuery.trim().toLowerCase();
    const matchSearch = q === '' || s.name.toLowerCase().includes(q) || (s.description ?? '').toLowerCase().includes(q);
    // 应用范围筛选
    let matchScope = true;
    if (scopeFilters.length > 0) {
      const allIds = ['__public__', ...MOCK_GROUPS.map(g => g.id)];
      const allSelected = allIds.every(id => scopeFilters.includes(id));
      if (!allSelected) {
        matchScope = false;
        if (scopeFilters.includes('__public__') && s.scope === 'public') {
          matchScope = true;
        }
        const selectedGroupIds = scopeFilters.filter(f => f !== '__public__');
        if (selectedGroupIds.length > 0 && s.groupIds) {
          if (selectedGroupIds.some(gid => s.groupIds.includes(gid))) {
            matchScope = true;
          }
        }
      }
    }
    return matchCategory && matchSearch && matchScope;
  });

  // 获取筛选显示文本
  const getScopeFilterLabel = () => {
    const allIds = ['__public__', ...MOCK_GROUPS.map(g => g.id)];
    const allSelected = allIds.every(id => scopeFilters.includes(id));
    if (scopeFilters.length === 0 || allSelected) return '全部应用范围';
    if (scopeFilters.includes('__public__') && scopeFilters.length === 1) return '全部用户';
    return `已选 ${scopeFilters.filter(f => f !== '__public__').length + (scopeFilters.includes('__public__') ? 1 : 0)} 项`;
  };

  const renderSkillCard = (skill: typeof MOCK_SKILLS[0]) => {
    const isAlreadyAdded = existingSkillNames.includes(skill.name);
    const isSelected = selectedIds.includes(skill.id);

    // 应用范围标签
    const scopeLabelsArr: string[] = (skill.scope === 'public' || !skill.groupIds || skill.groupIds.length === 0)
      ? ['全部用户']
      : skill.groupIds.map(id => MOCK_GROUPS.find(g => g.id === id)?.name || id);
    const isPublicScope = skill.scope === 'public' || !skill.groupIds || skill.groupIds.length === 0;

    const scopeExtra = !isAlreadyAdded ? (
      <div className="flex items-center gap-1 shrink-0">
        {isPublicScope ? (
          <AllUsersTag />
        ) : (
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              <span className="inline-flex items-center gap-1 cursor-default">
                <StatusTag mode="fill" variant="gray" className="whitespace-nowrap">
                  {scopeLabelsArr[0]}
                </StatusTag>
                {scopeLabelsArr.length > 1 && (
                  <StatusTag mode="fill" variant="gray">
                    +{scopeLabelsArr.length - 1}
                  </StatusTag>
                )}
              </span>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-[280px] leading-relaxed">
              <MetaText tone="inherit">{scopeLabelsArr.join('，')}</MetaText>
            </TooltipContent>
          </Tooltip>
        )}
      </div>
    ) : undefined;

    return (
      <SkillSelectCard
        key={skill.id}
        name={skill.name}
        version={skill.version}
        description={skill.description}
        state={isAlreadyAdded ? "disabled" : isSelected ? "selected" : "default"}
        extra={scopeExtra}
        onClick={() => toggleSkill(skill.id)}
      />
    );
  };

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) handleCancel(); }}>
      <DialogContent size="xl" style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }} onOpenAutoFocus={e => e.preventDefault()}>
        <DialogHeader>
          <DialogTitle>从企业技能库添加</DialogTitle>
        </DialogHeader>
        <DialogBody className="px-6 flex-1">
          {/* 搜索框 + 应用范围筛选 + 刷新 */}
          <div className="pb-3 flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)] pointer-events-none" />
              <Input
                type="text"
                placeholder="搜索技能名称或描述..."
                value={searchQuery}
                onChange={e => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
            {/* 应用范围多选下拉 */}
            <Popover open={scopeDropdownOpen} onOpenChange={setScopeDropdownOpen}>
              <Tooltip delayDuration={1000} open={scopeDropdownOpen ? false : undefined}>
                <TooltipTrigger asChild>
                  <PopoverTrigger asChild>
                    <button
                      type="button"
                      className="flex items-center justify-between gap-2 border border-gray-200 bg-white px-3 py-[5px] text-sm font-normal whitespace-nowrap transition-colors outline-none rounded-[4px] h-9 min-w-[10rem] max-w-[16rem] hover:border-blue-500 data-[state=open]:border-blue-500"
                    >
                      <span className="truncate text-left text-sm text-[#020617]">{getScopeFilterLabel()}</span>
                      <ChevronDown className="size-4 text-gray-500 shrink-0 transition-transform duration-200 [[data-state=open]>&]:rotate-180" />
                    </button>
                  </PopoverTrigger>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-[280px]">
                  <MetaText as="p" tone="inherit" className="break-words">{getScopeFilterLabel()}</MetaText>
                </TooltipContent>
              </Tooltip>
              <PopoverContent align="end" sideOffset={6} className="w-56 p-0">
                {(() => {
                  const allIds = ['__public__', ...MOCK_GROUPS.map(g => g.id)];
                  const allSelected = allIds.every(id => scopeFilters.includes(id));
                  const filteredGroups = MOCK_GROUPS.filter(g => g.name.toLowerCase().includes(scopeSearchQuery.toLowerCase()));
                  const showPublic = !scopeSearchQuery || '全部用户'.includes(scopeSearchQuery);
                  const showGroupSection = !scopeSearchQuery || '按组织'.includes(scopeSearchQuery) || filteredGroups.length > 0;

                  const toggleScopeItem = (key: string) => {
                    setScopeFilters(prev => {
                      if (prev.includes(key)) return prev.filter(f => f !== key);
                      return [...prev, key];
                    });
                  };

                  return (
                    <div className="py-1">
                      {/* 搜索框 */}
                      <div className="px-2 pb-1.5 pt-1">
                        <div className="relative">
                          <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)] pointer-events-none" />
                          <Input
                            placeholder="搜索..."
                            value={scopeSearchQuery}
                            onChange={(e) => setScopeSearchQuery(e.target.value)}
                            className="h-8 pl-7 pr-2 rounded-[4px]"
                          />
                        </div>
                      </div>
                      {/* 全部应用范围 — 全选/全不选切换 */}
                      {(!scopeSearchQuery || '全部应用范围'.includes(scopeSearchQuery)) && (
                        <button
                          type="button"
                          onClick={() => {
                            if (allSelected) {
                              setScopeFilters([]);
                            } else {
                              setScopeFilters(allIds);
                            }
                            setScopeSearchQuery('');
                          }}
                          className="flex items-center gap-2 w-full px-3 py-1.5 hover:bg-[#FAFAFA] transition-colors"
                        >
                          <Checkbox
                            checked={allSelected}
                            className="h-4 w-4 pointer-events-none"
                          />
                          <BodyText tone="primary" className="truncate text-left">全部应用范围</BodyText>
                        </button>
                      )}
                      {/* 全部用户 区域 */}
                      {showPublic && (
                        <>
                          <div className="px-3 pt-2 pb-1 select-none">
                            <MetaMedium tone="weak">全部用户</MetaMedium>
                          </div>
                          <button
                            type="button"
                            onClick={() => toggleScopeItem('__public__')}
                            className="flex items-center gap-2 w-full px-3 py-1.5 hover:bg-[#FAFAFA] transition-colors"
                          >
                            <Checkbox
                              checked={scopeFilters.includes('__public__')}
                              className="h-4 w-4 pointer-events-none"
                            />
                            <BodyText tone="primary" className="truncate text-left">全部用户</BodyText>
                          </button>
                        </>
                      )}
                      {/* 按组织 区域 */}
                      {showGroupSection && (
                        <>
                          <div className="px-3 pt-2.5 pb-1 select-none">
                            <MetaMedium tone="weak">按组织</MetaMedium>
                          </div>
                          <div className="max-h-44 overflow-y-auto">
                            {filteredGroups.map(group => {
                              const checked = scopeFilters.includes(group.id);
                              return (
                                <button
                                  key={group.id}
                                  type="button"
                                  onClick={() => toggleScopeItem(group.id)}
                                  className="flex items-center gap-2 w-full px-3 py-1.5 hover:bg-[#FAFAFA] transition-colors"
                                >
                                  <Checkbox
                                    checked={checked}
                                    className="h-4 w-4 pointer-events-none"
                                  />
                                  <BodyText tone="primary" className="truncate text-left" title={group.name}>{group.name}</BodyText>
                                </button>
                              );
                            })}
                            {filteredGroups.length === 0 && !showPublic && scopeSearchQuery && (
                              <MetaText as="p" tone="weak" className="py-2 text-center">没有匹配的结果</MetaText>
                            )}
                          </div>
                        </>
                      )}
                      {/* 底部已选信息 + 清除 */}
                      {scopeFilters.length > 0 && (
                        <div className="flex items-center justify-between px-3 py-2 border-t border-[#E5E5E5] mt-1">
                          <MetaText>
                            已选 {scopeFilters.filter(f => f !== '__public__').length + (scopeFilters.includes('__public__') ? 1 : 0)} 项
                          </MetaText>
                          <button
                            type="button"
                            onClick={() => { setScopeFilters([]); setScopeSearchQuery(''); }}
                            className="hover:opacity-80 transition-opacity"
                          >
                            <MetaText tone="brand">清除</MetaText>
                          </button>
                        </div>
                      )}
                    </div>
                  );
                })()}
              </PopoverContent>
            </Popover>
            <Tooltip delayDuration={300}>
              <TooltipTrigger asChild>
                <Button
                  variant="claw-outline"
                  size="claw-square"
                  onClick={handleRefresh}
                  aria-label="刷新"
                >
                  <RefreshCw />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top">
                <MetaText tone="inherit">刷新</MetaText>
              </TooltipContent>
            </Tooltip>
          </div>

          <FilterChipGroup
            items={[{ id: 'all', label: '全部' }, ...DEFAULT_CATEGORIES.map(cat => ({ id: cat.id, label: cat.name }))]}
            value={activeCategory}
            onChange={setActiveCategory}
            className="pb-3"
          />

          {filteredSkills.length > 0 ? (
            <div className="grid grid-cols-2 gap-3">
              {filteredSkills.map(skill => renderSkillCard(skill))}
            </div>
          ) : (
            <Empty className="border-0">
              <EmptyHeader>
                <EmptyMedia />
                <EmptyTitle>暂无匹配的技能</EmptyTitle>
                <EmptyDescription>请尝试调整搜索关键词或筛选条件</EmptyDescription>
              </EmptyHeader>
            </Empty>
          )}
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={handleCancel}>取消</Button>
          <Button
            variant="dialog-confirm"
            onClick={handleConfirm}
            disabled={selectedIds.length === 0}
          >
            确认添加{selectedIds.length > 0 ? `（${selectedIds.length} 个）` : ''}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── Edit/Create Role Modal ──────────────────────────────
const NAME_MAX_LEN = 8;

function RoleEditModal({
  open,
  role,
  roles,
  onClose,
  onSave,
}: {
  open: boolean;
  role: Role | null;
  roles: Role[];
  onClose: () => void;
  onSave: (role: Role) => void;
}) {
  const isNew = role === null;
  const [name, setName] = useState("");
  const [nameError, setNameError] = useState("");
  const [description, setDescription] = useState("");
  const [soul, setSoul] = useState("");
  const [skills, setSkills] = useState<RoleSkill[]>([]);
  const [visible, setVisible] = useState(true);
  const [scope, setScope] = useState<SkillScope>('public');
  const [groupIds, setGroupIds] = useState<string[]>([]);
  const [version, setVersion] = useState("1.0");
  const [versionError, setVersionError] = useState("");
  const [showAddPublicDialog, setShowAddPublicDialog] = useState(false);
  const [showAddEnterpriseDialog, setShowAddEnterpriseDialog] = useState(false);
  const [showBatchUpdateDialog, setShowBatchUpdateDialog] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [initialized, setInitialized] = useState(false);

  // Reset form when dialog opens
  if (open && !initialized) {
    setName(role?.name ?? "");
    setNameError("");
    setDescription(role?.description ?? "");
    setSoul(role?.soul ?? "");
    setSkills(role?.skills ? [...role.skills] : []);
    setVisible(role?.visible ?? true);
    setScope(role?.scope ?? 'public');
    setGroupIds(role?.groupIds ? [...role.groupIds] : []);
    setVersion(role?.version ? String(role.version) : "1.0");
    setInitialized(true);
  }
  if (!open && initialized) {
    setInitialized(false);
  }

  // 版本号格式校验：固定 x.y 两段式
  const ROLE_VERSION_REGEX = /^\d+\.\d+$/;
  // x.y 逐段比较：返回 >0 / ==0 / <0
  const cmpVer = (a: string, b: string): number => {
    const [am, an] = a.split('.').map(Number);
    const [bm, bn] = b.split('.').map(Number);
    if (am !== bm) return am - bm;
    return an - bn;
  };
  const handleVersionChange = (val: string) => {
    setVersion(val);
    if (!val) {
      setVersionError("");
    } else if (!ROLE_VERSION_REGEX.test(val)) {
      setVersionError("版本号格式必须为 x.y");
    } else if (!isNew && role && cmpVer(val, String(role.version)) <= 0) {
      setVersionError(`新版本号需高于上个版本号 ${role.version}`);
    } else {
      setVersionError("");
    }
  };

  const handleNameChange = (val: string) => {
    if (val.length > NAME_MAX_LEN) {
      setNameError(`角色名称不超过 ${NAME_MAX_LEN} 个字`);
      return;
    }
    setName(val);
    setNameError("");
  };

  const handleSave = () => {
    if (!name.trim()) {
      toast.error("请输入角色名称");
      return;
    }
    if (name.trim().length > NAME_MAX_LEN) {
      toast.error(`角色名称不超过 ${NAME_MAX_LEN} 个字`);
      return;
    }
    // 名称唯一性校验：不允许存在同名角色（编辑时排除自身）
    const trimmedName = name.trim();
    const duplicate = roles.find(r => r.name === trimmedName && r.id !== role?.id);
    if (duplicate) {
      toast.error(`同名角色「${trimmedName}」已存在，请使用其他名称`);
      return;
    }
    // 版本号校验：x.y 格式 + 更新时须高于上一版本（优先使用 inline error）
    if (!ROLE_VERSION_REGEX.test(version)) {
      setVersionError("版本号格式必须为 x.y");
      toast.error("版本号格式必须为 x.y");
      return;
    }
    if (!isNew && role && cmpVer(version, String(role.version)) <= 0) {
      setVersionError(`新版本号需高于上个版本号 ${role.version}`);
      toast.error(`新版本号需高于上个版本号 ${role.version}`);
      return;
    }
    // 保存时清除 previousVersion，使得再次编辑时不再显示"(原vX.X.X)"
    const cleanedSkills = skills.map(s => {
      const { previousVersion, ...rest } = s;
      return rest;
    });
    onSave({
      id: role?.id ?? `role-${Date.now()}`,
      name: name.trim(),
      description: description.trim(),
      soul: soul.trim(),
      skills: cleanedSkills,
      visible,
      scope,
      groupIds: scope === 'public' ? [] : groupIds,
      version,
    });
    onClose();
  };

  const removeSkill = (idx: number) => {
    setSkills(skills.filter((_, i) => i !== idx));
    setIsDirty(true);
  };

  const handleAddSkills = (newSkills: RoleSkill[], packageIds?: string[]) => {
    setSkills([...skills, ...newSkills]);
    setShowAddPublicDialog(false);
    setShowAddEnterpriseDialog(false);
    setIsDirty(true);

    // 来自「公共技能包」Tab 的添加：单独给出成功反馈，便于来源追溯
    if (packageIds && packageIds.length > 0) {
      const pickedPkgs = packageIds
        .map(id => PUBLIC_SKILL_PACKAGES.find(p => p.id === id))
        .filter((p): p is PublicSkillPackage => Boolean(p));
      if (pickedPkgs.length > 0) {
        const names = pickedPkgs.map(p => p.name).join("、");
        toast.success(`已添加公共技能包：${names}`);
      }
    }
  };

  // 单技能刷新
  const handleRefreshSingleSkill = (idx: number) => {
    const skill = skills[idx];
    const result = checkSkillUpdate(skill);
    if (result.hasUpdate && result.latestVersion) {
      setSkills(prev => prev.map((s, i) => i === idx ? { ...s, previousVersion: s.previousVersion || s.version, version: result.latestVersion!, latestVersion: result.latestVersion, updateNote: result.updateNote } : s));
      setIsDirty(true);
      toast.success(`${skill.name} 已更新至 ${result.latestVersion}`);
    }
  };

  // 获取可更新技能列表
  const getUpdatableSkills = (): UpdatableSkill[] => {
    const list: UpdatableSkill[] = [];
    skills.forEach((skill, idx) => {
      const result = checkSkillUpdate(skill);
      if (result.hasUpdate && result.latestVersion) {
        list.push({ index: idx, skill, latestVersion: result.latestVersion, updateNote: result.updateNote || '' });
      }
    });
    return list;
  };

  // 批量更新确认
  const handleBatchUpdateConfirm = (selectedIndices: number[]) => {
    setSkills(prev => prev.map((s, i) => {
      if (selectedIndices.includes(i)) {
        const result = checkSkillUpdate(s);
        if (result.hasUpdate && result.latestVersion) {
          return { ...s, previousVersion: s.previousVersion || s.version, version: result.latestVersion, latestVersion: result.latestVersion, updateNote: result.updateNote };
        }
      }
      return s;
    }));
    setIsDirty(true);
    setShowBatchUpdateDialog(false);
    toast.success(`已更新 ${selectedIndices.length} 个技能`);
  };

  return (
    <>
      <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
        <DialogContent
          className="sm:max-w-2xl"
          style={{ maxHeight: 'min(90vh, 780px)', display: 'flex', flexDirection: 'column' }}
        >
          <DialogHeader>
            <DialogTitle>
              {isNew ? "自定义角色" : `编辑角色 — ${role?.name}`}
            </DialogTitle>
            <DialogDescription className="sr-only">
              {isNew ? "创建一个新的自定义角色，配置角色的名称、描述、灵魂、应用范围与角色技能" : "编辑该角色的名称、描述、灵魂、应用范围与角色技能"}
            </DialogDescription>
          </DialogHeader>

          <DialogBody className="px-6 flex-1">
            <div className="space-y-4">
            {/* Name */}
            <div>
              <MetaMedium as="label" tone="secondary">
                角色名称
                <HelperText as="span" className="ml-1.5">{name.length}/{NAME_MAX_LEN}</HelperText>
              </MetaMedium>
              <Input
                value={name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="例如：营养师、法律顾问..."
                className={`mt-1.5 rounded-[4px] ${nameError ? "border-red-400" : ""}`}
                autoFocus={isNew}
                maxLength={NAME_MAX_LEN}
              />
              {nameError && (
                <MetaText as="p" tone="danger" className="mt-1">{nameError}</MetaText>
              )}
            </div>

            {/* Description — use Textarea for auto wrap */}
            <div>
              <MetaMedium as="label" tone="secondary">角色描述</MetaMedium>
              <Textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="一句话描述角色的核心能力"
                className="mt-1.5 min-h-[60px] resize-none rounded-[4px]"
                rows={2}
              />
            </div>

            {/* Soul */}
            <div>
              <MetaMedium as="label" tone="secondary">
                角色灵魂
                <HelperText as="span" className="ml-1.5">— 定义智能体的人格、价值观与行为准则</HelperText>
              </MetaMedium>
              <Textarea
                value={soul}
                onChange={(e) => setSoul(e.target.value)}
                placeholder="描述角色的人格特质、专业领域和行为准则..."
                className="mt-1.5 min-h-[80px] resize-none rounded-[4px]"
                rows={3}
              />
            </div>

            {/* 应用范围 — 使用与表格行一致的 MOCK_GROUPS 数据源 */}
            <div>
              <MetaMedium as="label" tone="secondary">应用范围</MetaMedium>
              <div className="mt-2">
                <ScopeSelect
                  scope={scope === 'public' ? 'all' : 'groups'}
                  selectedGroupIds={groupIds}
                  groups={MOCK_GROUPS}
                  onConfirm={(s, ids) => {
                    if (s === 'all') {
                      setScope('public');
                      setGroupIds([]);
                    } else {
                      setScope('private');
                      setGroupIds(ids);
                    }
                  }}
                />
              </div>
            </div>

            {/* 版本号 — 固定 x.y 两段式，编辑时需高于上一版本 */}
            <div>
              <MetaMedium as="label" tone="secondary">版本号</MetaMedium>
              <Input
                type="text"
                value={version}
                onChange={(e) => handleVersionChange(e.target.value)}
                placeholder={isNew ? "例如 1.0、2.0" : `新版本号需高于上个版本号 ${role?.version ?? '1.0'}`}
                className={`mt-1.5 rounded-[4px] ${versionError ? "border-red-400" : ""}`}
              />
              {versionError && (
                <MetaText as="p" tone="danger" className="mt-1">{versionError}</MetaText>
              )}
            </div>

            {/* Skills */}
            <div>
              <MetaMedium as="label" tone="secondary">
                角色技能
                <HelperText as="span" className="ml-1.5">— 赋予智能体专业执行能力的技能工具</HelperText>
              </MetaMedium>
              <div className="mt-1.5 border border-[var(--border)] rounded-[4px] overflow-hidden">
                <div className="px-4 border-b border-[var(--border)] flex items-center justify-between min-h-12">
                  <BodyMedium>
                    技能列表（共 {skills.length} 个）
                  </BodyMedium>
                  <div className="flex items-center gap-2">
                    {/* 批量刷新按钮：无可刷新技能时禁用 */}
                    {(() => {
                      const updatableCount = getUpdatableSkills().length;
                      const disabled = updatableCount === 0;
                      return (
                        <Tooltip delayDuration={300}>
                          <TooltipTrigger asChild>
                            <Button
                              variant="claw-outline"
                              size="claw-sm"
                              disabled={disabled}
                              onClick={() => setShowBatchUpdateDialog(true)}
                              className="h-7 px-3 text-xs gap-1.5"
                            >
                              <RefreshCw className="w-3.5 h-3.5" />
                              批量刷新
                              {updatableCount > 0 ? <span>（{updatableCount}）</span> : null}
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent side="top">
                            {disabled
                              ? skills.length === 0
                                ? '暂无技能可刷新'
                                : '所有技能已是最新版本'
                              : '检查并批量刷新技能到最新版本'}
                          </TooltipContent>
                        </Tooltip>
                      );
                    })()}
                  </div>
                </div>
                {skills.length === 0 ? (
                  <div className="text-center py-12 space-y-1">
                    <HelperText>该角色还没有技能</HelperText>
                    <HelperText>可从公共技能库或企业技能库添加</HelperText>
                  </div>
                ) : (
                  <div className="divide-y divide-[#F5F5F5]">
                    {skills.map((skill, idx) => {
                      const updateResult = checkSkillUpdate(skill);
                      const wasRefreshed = !!skill.previousVersion;
                      return (
                        <div key={`${skill.name}-${idx}`} className="flex items-center gap-3 px-4 py-3 hover:bg-[#F5F5F5] transition-colors">
                          <div className="w-8 h-8 rounded-[4px] bg-[#F5F5F5] flex items-center justify-center shrink-0">
                            <Package className="w-4 h-4 text-[var(--text-muted)]" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <BodyMedium className="font-mono">
                                {skill.name}
                              </BodyMedium>
                            </div>
                            <div className="flex items-center gap-2 mt-0.5">
                              <StatusTag mode="fill" variant={skill.source === '公共' ? 'blue' : 'gray'}>
                                {skill.source}
                              </StatusTag>
                              {wasRefreshed ? (
                                <span className="font-mono">
                                  <MetaMedium className="text-[var(--text-success)]">{skill.version}</MetaMedium>
                                  <MetaText tone="weak" className="ml-0.5">(原{skill.previousVersion})</MetaText>
                                </span>
                              ) : (
                                <MetaText tone="weak" className="font-mono">{skill.version}</MetaText>
                              )}
                            </div>
                          </div>
                          {/* 刷新按钮 */}
                          <Tooltip delayDuration={300}>
                            <TooltipTrigger asChild>
                              <Button
                                type="button"
                                variant="claw-outline"
                                onClick={() => updateResult.hasUpdate ? handleRefreshSingleSkill(idx) : undefined}
                                disabled={!updateResult.hasUpdate}
                                className="h-7 w-7 p-0 rounded-[4px] border-0 bg-transparent hover:bg-transparent hover:text-[var(--text-brand)] [&_svg]:size-3.5 text-[var(--text-title)]"
                                title={updateResult.hasUpdate ? '有新版本，点击刷新' : '已是最新'}
                              >
                                <RefreshCw />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              {updateResult.hasUpdate
                                ? `有新版本 ${updateResult.latestVersion}，点击刷新`
                                : '已是最新版本'}
                            </TooltipContent>
                          </Tooltip>
                          {/* 删除按钮 */}
                          <Button
                            type="button"
                            variant="claw-outline"
                            onClick={() => removeSkill(idx)}
                            className="h-7 w-7 p-0 rounded-[4px] border-0 bg-transparent hover:bg-transparent hover:text-[var(--text-danger)] [&_svg]:size-3.5 text-[var(--text-title)]"
                            title="从角色中移除"
                          >
                            <Trash2 />
                          </Button>
                        </div>
                      );
                    })}
                  </div>
                )}
                <div className="px-4 py-2 border-t border-[var(--border)] flex items-center gap-2">
                  <Button variant="claw-outline" size="claw-sm" className="gap-1.5 text-xs h-8" onClick={() => setShowAddPublicDialog(true)}>
                    <Plus className="w-3.5 h-3.5" />
                    从公共技能库添加
                  </Button>
                  <Button variant="claw-outline" size="claw-sm" className="gap-1.5 text-xs h-8" onClick={() => setShowAddEnterpriseDialog(true)}>
                    <Plus className="w-3.5 h-3.5" />
                    从企业技能库添加
                  </Button>
                </div>
              </div>
            </div>

            {/* Visible */}
            <div className="flex items-center justify-between">
              <div>
                <MetaMedium as="label" tone="secondary">用户可见</MetaMedium>
                <HelperText className="mt-0.5">启用后，用户创建 Agent 时可选择此角色</HelperText>
              </div>
              <Switch checked={visible} onCheckedChange={setVisible} />
            </div>
            </div>
          </DialogBody>

          <DialogFooter>
            <Button variant="outline" onClick={onClose}>取消</Button>
            <Button
              variant="dialog-confirm"
              onClick={handleSave}
            >
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <RoleAddPublicSkillDialog
        open={showAddPublicDialog}
        existingSkillNames={skills.map(s => s.name)}
        onConfirm={handleAddSkills}
        onCancel={() => setShowAddPublicDialog(false)}
      />

      <RoleAddEnterpriseSkillDialog
        open={showAddEnterpriseDialog}
        existingSkillNames={skills.map(s => s.name)}
        onConfirm={handleAddSkills}
        onCancel={() => setShowAddEnterpriseDialog(false)}
        roleScope={scope}
        roleGroupIds={groupIds}
      />

      <BatchUpdateDialog
        open={showBatchUpdateDialog}
        updatableSkills={getUpdatableSkills()}
        onConfirm={handleBatchUpdateConfirm}
        onCancel={() => setShowBatchUpdateDialog(false)}
      />
    </>
  );
}

// ── 定期同步：从专家包拉取最新 Skill / Soul ──────────────

/** 专家包同步配置 */
const EXPERT_PACKAGE_SYNC_CONFIG = {
  /** 专家包业务 ID */
  packageId: 'tcb-programmer-agent',
  /** 同步间隔（毫秒）：默认 5 分钟 */
  intervalMs: 5 * 60 * 1000,
  /** 本地存储 key：上次同步时间戳 */
  lastSyncKey: 'expert_package_last_sync_tcb_programmer',
};

/** 模拟从专家包 API 拉取最新版本（生产环境替换为真实 HTTP 请求）
 *
 * 实际调用：
 * GET /api/v1/expert-packages/{packageId}/versions/latest?agent=openclaw&agentVersion=3.6.0&client=clawpro
 * GET /api/v1/expert-packages/{packageId}/skill-list
 */
async function fetchExpertPackageLatest(): Promise<{
  soul: string;
  skills: RoleSkill[];
  version: string;
} | null> {
  // TODO: 生产环境接入真实 API
  // const manifest = await fetch(`/api/v1/expert-packages/${EXPERT_PACKAGE_SYNC_CONFIG.packageId}/versions/latest?client=clawpro`).then(r => r.json());
  // const skillList = await fetch(`/api/v1/expert-packages/${EXPERT_PACKAGE_SYNC_CONFIG.packageId}/skill-list`).then(r => r.json());

  // Mock：返回当前已知的最新数据（模拟专家包已更新）
  return {
    soul: PROGRAMMER_SOUL,
    skills: [
      { name: 'github', version: 'v2.1.0', source: '公共' },
      { name: 'code-reviewer', version: 'v1.4.0', source: '公共' },
      { name: 'docker-ops', version: 'v1.2.0', source: '公共' },
      { name: 'api-tester', version: 'v1.6.0', source: '公共' },
    ],
    version: '1.1',
  };
}

// ── Main Tab ────────────────────────────────────────────
export default function SkillRolesTab() {
  const [roles, setRoles] = useState<Role[]>([...MOCK_ROLES]);
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const [showEdit, setShowEdit] = useState(false);
  const [isNewRole, setIsNewRole] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Role | null>(null);
  // 应用范围筛选（多选 checkbox）
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set());
  const [scopeDropdownOpen, setScopeDropdownOpen] = useState(false);
  const [scopeSearchQuery, setScopeSearchQuery] = useState('');
  const scopeDropdownRef = useRef<HTMLDivElement>(null);
  const allScopeKeys = ['public', ...MOCK_GROUPS.map(g => g.id)];
  // Agent 列表（从 global store 加载，用于角色同步检测）
  const [agents, setAgents] = useState<AgentItem[]>(() => loadClawList());
  // 角色同步弹窗
  const [distributeTarget, setDistributeTarget] = useState<Role | null>(null);
  const [showDistribute, setShowDistribute] = useState(false);
  const [historyTarget, setHistoryTarget] = useState<Role | null>(null);

  // ── 定期同步专家包数据 ──────────────────────────────────
  /** 执行一次同步：从专家包拉取最新数据并更新「程序员」角色 */
  const syncProgrammerRole = useCallback(async () => {
    const latest = await fetchExpertPackageLatest();
    if (!latest) return;

    setRoles(prevRoles => {
      const idx = prevRoles.findIndex(r => r.id === 'role-tcb-programmer');
      if (idx === -1) {
        // 角色不存在时自动创建（兜底）
        const newRole: Role = {
          id: 'role-tcb-programmer',
          name: '程序员',
          description: '经验丰富的全栈开发工程师，精通网站、小程序和全栈应用的开发部署场景',
          soul: latest.soul,
          skills: latest.skills,
          visible: true,
          scope: 'public',
          groupIds: [],
          version: latest.version,
        };
        return [newRole, ...prevRoles];
      }

      const existing = prevRoles[idx];
      // 仅当数据发生变化时才更新，避免无意义重渲染
      const soulChanged = existing.soul !== latest.soul;
      const skillsChanged = JSON.stringify(existing.skills) !== JSON.stringify(latest.skills);
      const versionChanged = existing.version !== latest.version;

      if (!soulChanged && !skillsChanged && !versionChanged) {
        return prevRoles;
      }

      const updated = { ...existing };
      if (soulChanged) updated.soul = latest.soul;
      if (skillsChanged) updated.skills = latest.skills;
      if (versionChanged) updated.version = latest.version;

      const next = [...prevRoles];
      next[idx] = updated;
      return next;
    });

    localStorage.setItem(EXPERT_PACKAGE_SYNC_CONFIG.lastSyncKey, String(Date.now()));
  }, []);

  // 组件挂载时立即执行一次同步，并启动定时器
  useEffect(() => {
    syncProgrammerRole();
    const timer = setInterval(syncProgrammerRole, EXPERT_PACKAGE_SYNC_CONFIG.intervalMs);
    return () => clearInterval(timer);
  }, [syncProgrammerRole]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  // 点击外部关闭应用范围下拉
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (scopeDropdownRef.current && !scopeDropdownRef.current.contains(e.target as Node)) {
        setScopeDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      setRoles((items) => {
        const oldIndex = items.findIndex((i) => i.id === active.id);
        const newIndex = items.findIndex((i) => i.id === over.id);
        return arrayMove(items, oldIndex, newIndex);
      });
    }
  };

  const handleToggle = (id: string) => {
    setRoles(roles.map((r) => (r.id === id ? { ...r, visible: !r.visible } : r)));
  };

  const handleEdit = (role: Role) => {
    setEditingRole(role);
    setIsNewRole(false);
    setShowEdit(true);
  };

  const handleNew = () => {
    setEditingRole(null);
    setIsNewRole(true);
    setShowEdit(true);
  };

  const handleSave = (saved: Role) => {
    if (isNewRole) {
      setRoles([saved, ...roles]);
      toast.success(`角色「${saved.name}」已创建`);
    } else {
      setRoles(roles.map((r) => (r.id === saved.id ? saved : r)));
      toast.success(`角色「${saved.name}」已更新至 v${saved.version}`);
    }
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    setRoles(roles.filter((r) => r.id !== deleteTarget.id));
    toast.success(`角色「${deleteTarget.name}」已删除`);
    setDeleteTarget(null);
  };

  const handleScopeChange = (id: string, scope: SkillScope, groupIds: string[]) => {
    setRoles(roles.map(r => r.id === id ? { ...r, scope, groupIds } : r));
    toast.success('应用范围修改成功');
  };

  // ── 角色同步到 Agent ──────────────────────────────────
  /** 刷新 agent 列表（从 global store 重新加载） */
  const refreshAgents = () => {
    setAgents(loadClawList());
  };

  /** 计算使用指定角色的 Agent 总数 */
  /** 判定指定角色是否存在「可更新」的 Agent（待更新 / 更新失败） */
  const hasUpdatableAgents = useCallback((role: Role): boolean => {
    const cmp = (a: string, b: string): number => {
      const [am, an] = a.split('.').map(Number);
      const [bm, bn] = b.split('.').map(Number);
      if (am !== bm) return (am || 0) - (bm || 0);
      return (an || 0) - (bn || 0);
    };
    return agents.some(a => {
      // 命中条件与更新弹窗保持一致：主角色名 = 目标角色，或多角色实例内任一角色位 = 目标角色
      const matchedByMain = a.roleName === role.name;
      const matchedBySlot = ((a as any).roles as { roleName: string }[] | undefined)?.some(s => s.roleName === role.name) ?? false;
      if (!matchedByMain && !matchedBySlot) return false;
      const explicit = (a as any).roleUpdateStatus as string | undefined;
      if (explicit) return explicit === 'pending_update' || explicit === 'failed';
      const dv = (a as any).distributedRoleVersion as string | undefined;
      if (!dv) return true;
      return cmp(dv, String(role.version)) < 0;
    });
  }, [agents]);

  /** 打开下发弹窗 */
  const handleOpenDistribute = (role: Role) => {
    refreshAgents();
    setDistributeTarget(role);
    setShowDistribute(true);
  };

  const handleShowHistory = (role: Role) => {
    setHistoryTarget(role);
  };

  /** 确认更新：更新选中 Agent 的 distributedRoleVersion，并记录更新历史 */
  const handleDistributeConfirm = (selectedIds: string[]) => {
    if (!distributeTarget) return;
    const successCount = selectedIds.length;
    const updated = agents.map(a => {
      // 命中实例即更新版本（含主角色实例与多角色实例下命中目标角色的实例）
      if (selectedIds.includes(a.id)) {
        return { ...a, distributedRoleVersion: distributeTarget.version };
      }
      return a;
    });
    saveClawList(updated);
    notifyClawListChange();
    setAgents(updated);
    // 追加更新记录到角色
    const record: RoleUpdateRecord = {
      id: `hist-${Date.now()}`,
      version: distributeTarget.version,
      totalCount: successCount,
      successCount,
      failedCount: 0,
      operator: 'admin@acompany.com',
      operatedAt: new Date().toISOString().replace('T', ' ').slice(0, 19),
      status: 'success',
      reason: '-',
    };
    setRoles(prev => prev.map(r => {
      if (r.id === distributeTarget.id) {
        return { ...r, updateHistory: [...(r.updateHistory || []), record] };
      }
      return r;
    }));
    // 同步更新 distributeTarget 引用给后续对话框用
    setDistributeTarget(prev => prev ? { ...prev, updateHistory: [...(prev.updateHistory || []), record] } : prev);
    toast.success(`已更新 ${selectedIds.length} 个实例至「${distributeTarget.name}」v${distributeTarget.version}`);
    setShowDistribute(false);
    setDistributeTarget(null);
  };

  // 筛选后的角色列表（多选）
  const filteredRoles = roles.filter(role => {
    if (selectedScopes.size === 0) return true; // 无筛选 = 全部
    const isAllSelected = allScopeKeys.length > 0 && allScopeKeys.every(k => selectedScopes.has(k));
    if (isAllSelected) return true;
    // 检查是否匹配勾选的任意范围
    const isPublicRole = role.scope === 'public' || !role.groupIds || role.groupIds.length === 0;
    if (selectedScopes.has('public') && isPublicRole) return true;
    if (!isPublicRole && role.groupIds) {
      for (const gid of role.groupIds) {
        if (selectedScopes.has(gid)) return true;
      }
    }
    return false;
  });

  return (
    <>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <PanelTitle as="div">角色列表</PanelTitle>
        <div className="flex items-center gap-3">
          {/* 应用范围筛选 */}
          {/*
            停服时仍允许「选择应用范围」下拉筛选正常可用：纯导航/筛选类操作，不消耗管控台写权限。
            包装层（含触发按钮 + 内联下拉浮层）整体豁免停服禁用。
          */}
          <div className="relative" ref={scopeDropdownRef} data-billing-exempt>
            <Tooltip delayDuration={1000} open={scopeDropdownOpen ? false : undefined}>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  onClick={() => setScopeDropdownOpen(prev => !prev)}
                  className="flex items-center justify-between gap-1 min-w-[10rem] max-w-[20rem] h-9 px-3 border border-gray-200 rounded-xl bg-white hover:bg-gray-50 transition-colors"
                >
                  <BodyText tone="secondary" className="truncate text-left">
                    {selectedScopes.size === 0
                      ? '选择应用范围'
                      : allScopeKeys.every(k => selectedScopes.has(k))
                        ? '全部应用范围'
                        : Array.from(selectedScopes).map(s => s === 'public' ? '全部用户' : MOCK_GROUPS.find(g => g.id === s)?.name || s).join('、')}
                  </BodyText>
                  <ChevronDown className={`w-4 h-4 text-[var(--text-weak)] flex-shrink-0 transition-transform ${scopeDropdownOpen ? 'rotate-180' : ''}`} />
                </button>
              </TooltipTrigger>
              <TooltipContent side="bottom" className="max-w-[280px]">
                <MetaText as="p" tone="inherit" className="break-words">
                  {selectedScopes.size === 0
                    ? '选择应用范围'
                    : allScopeKeys.every(k => selectedScopes.has(k))
                      ? '全部应用范围'
                      : Array.from(selectedScopes).map(s => s === 'public' ? '全部用户' : MOCK_GROUPS.find(g => g.id === s)?.name || s).join('、')}
                </MetaText>
              </TooltipContent>
            </Tooltip>
            {scopeDropdownOpen && (() => {
              const filteredGroups = MOCK_GROUPS.filter(g => g.name.toLowerCase().includes(scopeSearchQuery.toLowerCase()));
              const showPublic = !scopeSearchQuery || '全部用户'.includes(scopeSearchQuery);
              const showGroupSection = !scopeSearchQuery || '按组织'.includes(scopeSearchQuery) || filteredGroups.length > 0;
              const isAllSelected = allScopeKeys.length > 0 && allScopeKeys.every(k => selectedScopes.has(k));

              const toggleScope = (key: string) => {
                setSelectedScopes(prev => {
                  const next = new Set(prev);
                  if (next.has(key)) next.delete(key); else next.add(key);
                  return next;
                });
              };

              return (
                <div className="absolute right-0 top-full mt-1 w-56 bg-white rounded-[4px] shadow-[var(--shadow-popover)] z-50 pt-2 px-2 pb-0">
                  {/* 搜索框 */}
                  <div className="mb-1">
                    <div className="relative">
                      <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400 pointer-events-none" />
                      <Input
                        type="text"
                        placeholder="搜索..."
                        value={scopeSearchQuery}
                        onChange={(e) => setScopeSearchQuery(e.target.value)}
                        className="h-8 pl-8 pr-2 text-sm"
                        onClick={(e) => e.stopPropagation()}
                      />
                    </div>
                  </div>
                  {/* 全部应用范围 — 全选/全不选切换 */}
                  {(!scopeSearchQuery || '全部应用范围'.includes(scopeSearchQuery)) && (
                    <button
                      type="button"
                      onClick={() => {
                        if (isAllSelected) {
                          setSelectedScopes(new Set());
                        } else {
                          setSelectedScopes(new Set(allScopeKeys));
                        }
                        setScopeSearchQuery('');
                      }}
                      className={`flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors ${isAllSelected ? 'bg-[var(--bg-brand-selected)]' : 'hover:bg-[#FAFAFA]'}`}
                    >
                      <Checkbox
                        checked={isAllSelected}
                        className="pointer-events-none"
                      />
                      <BodyText tone="secondary" className="truncate text-left">全部应用范围</BodyText>
                    </button>
                  )}
                  {/* 全部用户 区域 */}
                  {showPublic && (
                    <>
                      <div className="px-3 pt-2 pb-1 select-none">
                        <MetaMedium tone="weak">全部用户</MetaMedium>
                      </div>
                      <button
                        type="button"
                        onClick={() => toggleScope('public')}
                        className={`flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors ${selectedScopes.has('public') ? 'bg-[var(--bg-brand-selected)]' : 'hover:bg-[#FAFAFA]'}`}
                      >
                        <Checkbox
                          checked={selectedScopes.has('public')}
                          className="pointer-events-none"
                        />
                        <BodyText tone="secondary" className="truncate text-left">全部用户</BodyText>
                      </button>
                    </>
                  )}
                  {/* 按组织 区域 */}
                  {showGroupSection && (
                    <>
                      <div className="px-3 pt-2.5 pb-1 select-none">
                        <MetaMedium tone="weak">按组织</MetaMedium>
                      </div>
                      <div className="max-h-44 overflow-y-auto space-y-0.5">
                        {filteredGroups.map(group => {
                          const checked = selectedScopes.has(group.id);
                          return (
                            <button
                              key={group.id}
                              type="button"
                              onClick={() => toggleScope(group.id)}
                              className={`flex items-center gap-2 w-full h-8 px-3 rounded-[6px] transition-colors ${checked ? 'bg-[var(--bg-brand-selected)]' : 'hover:bg-[#FAFAFA]'}`}
                            >
                              <Checkbox
                                checked={checked}
                                className="pointer-events-none"
                              />
                              <BodyText tone="secondary" className="truncate text-left" title={group.name}>{group.name}</BodyText>
                            </button>
                          );
                        })}
                        {filteredGroups.length === 0 && !showPublic && scopeSearchQuery && (
                          <MetaText as="p" tone="weak" className="py-2 text-center">没有匹配的结果</MetaText>
                        )}
                      </div>
                    </>
                  )}
                  {/* 底部：已选数量 + 清除 */}
                  {selectedScopes.size > 0 && (
                    <div className="border-t border-[#EAEEF4] mt-1 px-1 h-9 flex items-center justify-between">
                      <MetaText>已选 {selectedScopes.size} 个应用范围</MetaText>
                      <Button
                        variant="link"
                        className="text-xs"
                        onClick={() => {
                          setSelectedScopes(new Set());
                          setScopeSearchQuery('');
                        }}
                      >
                        清除
                      </Button>
                    </div>
                  )}
                </div>
              );
            })()}
          </div>
          <Button
            onClick={handleNew}
          >
            <Plus className="w-4 h-4 mr-1.5" />
            自定义角色
          </Button>
        </div>
      </div>

      {/* Table */}
      <SurfaceCard className="overflow-hidden">
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead className="w-10 px-3" />
                <TableHead>角色名称</TableHead>
                <TableHead>角色描述</TableHead>
                <TableHead>应用范围</TableHead>
                <TableHead>用户可见</TableHead>
                <TableHead>版本</TableHead>
                <TableHead className="w-[170px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <SortableContext
              items={filteredRoles.map((r) => r.id)}
              strategy={verticalListSortingStrategy}
            >
              <TableBody>
                {filteredRoles.map((role) => (
                  <SortableRoleRow
                    key={role.id}
                    role={role}
                    hasUpdatable={hasUpdatableAgents(role)}
                    onToggle={handleToggle}
                    onEdit={handleEdit}
                    onDelete={setDeleteTarget}
                    onScopeChange={handleScopeChange}
                    onDistribute={handleOpenDistribute}
                    onShowHistory={handleShowHistory}
                  />
                ))}
              </TableBody>
            </SortableContext>
          </Table>
        </DndContext>
        <div className="px-4 h-10 flex items-center border-t border-[#EAEEF4]">
          <MetaText>
            共 {filteredRoles.length} 个角色{selectedScopes.size > 0 ? `（筛选中，全部 ${roles.length} 个）` : ''}
          </MetaText>
        </div>
      </SurfaceCard>

      {/* Edit/Create Modal */}
      <RoleEditModal
        open={showEdit}
        role={isNewRole ? null : editingRole}
        roles={roles}
        onClose={() => setShowEdit(false)}
        onSave={handleSave}
      />

      {/* 角色下发弹窗 */}
      {distributeTarget && (
        <RoleDistributeDialog
          open={showDistribute}
          roleName={distributeTarget.name}
          roleVersion={distributeTarget.version}
          agents={agents
            // 命中条件：主角色名 = 目标角色，或多角色实例内任一角色位 = 目标角色
            .filter(a =>
              a.roleName === distributeTarget.name ||
              (a.roles || []).some(s => s.roleName === distributeTarget.name)
            )
              .map(a => {
    const instanceId = (a as any).instanceId || a.id;
            // 该实例内命中目标角色的角色位。
              // 多角色实例：取其 roles 中命中目标角色的角色位；
           // 单角色实例（无 roles 数组，仅 roleName 命中）：合成一个虚拟角色位（slotId 沿用 instanceId），
              // 使其也能像多角色实例一样「按实例分组 + 可展开 + 展开后是其下角色位」统一展示。
   const rawHitSlots = (a.roles || []).filter(s => s.roleName === distributeTarget.name);
     const hitSlots = rawHitSlots.length > 0
      ? rawHitSlots
 : [{ slotId: instanceId, roleName: distributeTarget.name }];
    // 所有命中实例统一生成 matchedSlots，行前一律展示计数与展开箭头，
   // 单/多角色实例结构完全一致（与「设计师」弹窗按实例分组展示保持一致）。
   const matchedSlots = hitSlots.map((s, i) => ({
                slotId: s.slotId,
        // 同名角色位自动追加序号：设计师 1 / 设计师 2 …
         // 仅命中 1 个时不加序号，直接用角色名。
   displayName: hitSlots.length > 1
           ? `${distributeTarget.name} ${i + 1}`
      : distributeTarget.name,
            distributedRoleVersion: (s as any).distributedRoleVersion || (a as any).distributedRoleVersion,
            roleUpdateStatus: (s as any).roleUpdateStatus,
   }));
              return {
   id: a.id,
                name: a.name,
   instanceId,
       status: a.status,
                creator: (a as any).creator,
                groupId: (a as any).groupId,
                groupName: (a as any).groupName,
                agentType: a.agentType,
                distributedRoleVersion: (a as any).distributedRoleVersion,
                matchedSlots,
              };
            })}
          onConfirm={handleDistributeConfirm}
          onCancel={() => { setShowDistribute(false); setDistributeTarget(null); }}
        />
      )}

      {/* 角色更新记录弹窗 */}
      {historyTarget && (
        <RoleUpdateHistoryDialog
          open={!!historyTarget}
          onClose={() => setHistoryTarget(null)}
          role={historyTarget}
        />
      )}

      {/* Delete Confirm —— 遵循项目标准警示弹窗规范：
            标题/正文黑色，强调字段告警色，destructive 主按钮，右上角带 X 关闭按钮 */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(o) => { if (!o) setDeleteTarget(null); }}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除角色</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <div className="space-y-2">
              <BodyText as="p" tone="primary">
                确定要删除角色「<BodyMedium tone="danger">{deleteTarget?.name}</BodyMedium>」吗？
              </BodyText>
              <BodyText as="p" tone="primary">
                删除后，已使用该角色的 Agent 仍可正常使用，但后续新建 Agent 将无法选择此角色，
                <BodyMedium tone="danger">此操作不可撤销</BodyMedium>。
              </BodyText>
            </div>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDeleteTarget(null)}>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
