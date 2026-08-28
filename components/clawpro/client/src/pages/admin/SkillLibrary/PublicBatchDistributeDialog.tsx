import { useState, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
} from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';
import { DialogPagination } from '@/components/ui/pagination';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Search, ChevronDown, Check, AlertTriangle } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { type SkillScope, type AgentInstance, type Group } from './types';

/** 筛选选项类型 —— 多选 */
type FilterOption = 'not_distributed' | 'failed' | 'pending_update' | 'success';

/** 版本筛选选项 */
type VersionFilterOption = 'all' | 'gte_0328' | 'lt_0328';

/** 版本筛选选项配置 */
const VERSION_FILTER_OPTIONS: { key: VersionFilterOption; label: string }[] = [
  { key: 'all', label: '全部' },
  { key: 'gte_0328', label: '26.3.28版本后（含28）' },
  { key: 'lt_0328', label: '26.3.28版本前' },
];

/** 版本号比较基准：2026.3.28 */
const VERSION_THRESHOLD = '2026.3.28';

/** 解析版本号为可比较的数值数组 */
function parseVersion(v: string): number[] {
  return v.split('.').map(Number);
}

/** 比较两个版本号，返回 -1/0/1 */
function compareVersion(a: string, b: string): number {
  const pa = parseVersion(a);
  const pb = parseVersion(b);
  for (let i = 0; i < Math.max(pa.length, pb.length); i++) {
    const na = pa[i] || 0;
    const nb = pb[i] || 0;
    if (na < nb) return -1;
    if (na > nb) return 1;
  }
  return 0;
}

interface PublicBatchDistributeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  skillId?: string;
  skillName?: string;
  /** 当前 Skill 最新版本号，用于判定"待更新" */
  skillVersion?: string;
  /** 当前 Skill 的应用范围 */
  skillScope?: SkillScope;
  /** 当前 Skill 关联的组织 ID 列表 */
  skillGroupIds?: string[];
  onDistributionStart?: (selectedInstanceIds: string[], selectedInstancesData: any[]) => void;
  /** 弹窗标题，默认 "批量下发 Skill" */
  title?: string;
  /** 是否显示应用范围筛选，默认 true */
  showScopeFilter?: boolean;
  /** Agent 实例列表（外部传入） */
  instances: AgentInstance[];
  /** 组织列表（外部传入，showScopeFilter=true 时必传） */
  groups?: Group[];
  /** MCP 场景：隐藏实例列表中的创建人、组织信息 */
  hideCreatorAndGroup?: boolean;
  /** MCP 场景：下发状态筛选改为单选下拉（只有 未下发 / 下发失败），去掉待更新和多选逻辑 */
  singleStatusFilter?: boolean;
  /** MCP 场景：自定义描述 ReactNode，覆盖默认描述 */
  descriptionNode?: React.ReactNode;
  /** MCP 场景：显示版本筛选下拉，默认 false */
  showVersionFilter?: boolean;
  /** MCP 场景：下发前需要二次确认弹窗，默认 false */
  showConfirmDialog?: boolean;
}

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100, 200, 500];
const DEFAULT_DISTRIBUTE_STATUS_FILTERS: FilterOption[] = ['not_distributed', 'failed', 'pending_update'];
const DEFAULT_MCP_STATUS_FILTERS: FilterOption[] = ['not_distributed', 'failed'];

const FILTER_OPTIONS: { key: FilterOption; label: string }[] = [
  { key: 'not_distributed', label: '未下发' },
  { key: 'failed', label: '下发失败' },
  { key: 'pending_update', label: '待更新' },
  { key: 'success', label: '下发成功' },
];

const MCP_FILTER_OPTIONS: { key: FilterOption; label: string }[] = [
  { key: 'not_distributed', label: '未下发' },
  { key: 'failed', label: '下发失败' },
  { key: 'success', label: '下发成功' },
];

const STATUS_DISPLAY_MAP: Record<FilterOption | 'distributing', { label: string; className: string }> = {
  not_distributed: { label: '未下发', className: 'text-[var(--text-muted)]' },
  failed: { label: '下发失败', className: 'text-[var(--text-danger)]' },
  pending_update: { label: '待更新', className: 'text-[var(--text-warning)]' },
  success: { label: '下发成功', className: 'text-[var(--text-success)]' },
  distributing: { label: '下发中', className: 'text-[var(--text-brand)]' },
};

const dropdownTriggerClassName =
  'flex h-9 items-center justify-between gap-1 rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-sm text-[var(--text-emphasis)] outline-none transition-colors hover:border-[var(--cp-brand-blue)] data-[state=open]:border-[var(--cp-brand-blue)]';

const dropdownPanelClassName =
  'absolute top-full z-50 mt-1 rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--cp-surface)] py-1 shadow-[var(--cp-shadow-overlay)]';

const dropdownOptionClassName =
  'flex w-full items-center gap-2 px-3 py-2 text-sm text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-grey-hover-subtle)]';

const getCustomCheckboxClassName = (selected: boolean) =>
  `flex h-4 w-4 shrink-0 items-center justify-center rounded-[3px] border ${
    selected
      ? 'border-[var(--cp-brand-blue)] bg-[var(--cp-brand-blue)]'
      : 'border-[var(--cp-border-control)] bg-[var(--cp-surface)]'
  }`;

export default function PublicBatchDistributeDialog({
  open,
  onOpenChange,
  skillName,
  skillVersion,
  skillScope,
  skillGroupIds,
  onDistributionStart,
  title = '批量下发 Skill',
  showScopeFilter = true,
  instances,
  groups = [],
  hideCreatorAndGroup = false,
  singleStatusFilter = false,
  descriptionNode,
  showVersionFilter = false,
  showConfirmDialog = false,
}: PublicBatchDistributeDialogProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedInstances, setSelectedInstances] = useState<string[]>([]);
  /** 是否处于"选择全部"模式（跨页全选） */
  const [selectAllMode, setSelectAllMode] = useState(false);
  /** 状态多选筛选 */
  const [statusFilters, setStatusFilters] = useState<FilterOption[]>([]);
  /** 应用范围筛选：空数组=全部, 否则为选中的组织 ID 列表（多选） */
  const [scopeFilters, setScopeFilters] = useState<string[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  /** 多选下拉的展开状态 */
  const [filterDropdownOpen, setFilterDropdownOpen] = useState(false);
  const filterDropdownRef = useRef<HTMLDivElement>(null);
  /** 组织筛选下拉 */
  const [scopeDropdownOpen, setScopeDropdownOpen] = useState(false);
  const [scopeSearchQuery, setScopeSearchQuery] = useState('');
  const scopeDropdownRef = useRef<HTMLDivElement>(null);
  /** 版本筛选 */
  const [versionFilter, setVersionFilter] = useState<VersionFilterOption>('all');
  /** 二次确认弹窗 */
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false);
  const [confirmInput, setConfirmInput] = useState('');
  /** 已下发成功实例再次下发确认 */
  const [redistributeConfirmOpen, setRedistributeConfirmOpen] = useState(false);

  // 点击外部关闭下拉
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (filterDropdownRef.current && !filterDropdownRef.current.contains(e.target as Node)) {
        setFilterDropdownOpen(false);
      }
      if (scopeDropdownRef.current && !scopeDropdownRef.current.contains(e.target as Node)) {
        setScopeDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // 当打开弹窗时，重置筛选状态；默认不选中任何实例
  useEffect(() => {
    if (open) {
      // MCP 场景默认选中「未下发」+「下发失败」；普通下发默认选中「未下发」+「下发失败」+「待更新」
      setStatusFilters(singleStatusFilter ? [...DEFAULT_MCP_STATUS_FILTERS] : [...DEFAULT_DISTRIBUTE_STATUS_FILTERS]);
      setSearchQuery('');
      setCurrentPage(1);
      setPageSize(20);
      setSelectAllMode(false);
      setFilterDropdownOpen(false);
      setScopeDropdownOpen(false);
      setScopeSearchQuery('');
      setVersionFilter('all');
      setConfirmDialogOpen(false);
      setConfirmInput('');
      setRedistributeConfirmOpen(false);
      // 根据 Skill 应用范围设置默认筛选
      if (showScopeFilter) {
        if (skillScope === 'private' && skillGroupIds && skillGroupIds.length > 0) {
          setScopeFilters([...skillGroupIds]);
        } else {
          setScopeFilters([]);
        }
      } else {
        setScopeFilters([]);
      }
      // 默认不选中任何实例
      setSelectedInstances([]);
    }
  }, [open, skillScope, skillGroupIds, singleStatusFilter]);

  const activeStatusOptions = singleStatusFilter ? MCP_FILTER_OPTIONS : FILTER_OPTIONS;

  /** 获取筛选下拉的显示文本 */
  const getFilterDisplayText = () => {
    if (statusFilters.length === 0) return '全部下发状态';
    if ((statusFilters as any)[0] === '__none__') return '下发状态';
    if (statusFilters.length === activeStatusOptions.length) return '全部下发状态';
    return statusFilters.map(k => activeStatusOptions.find(o => o.key === k)?.label).filter(Boolean).join('、');
  };

  /** 判断当前是否为"全部状态" */
  const isAllStatusSelected = statusFilters.length === 0 || statusFilters.length === activeStatusOptions.length;

  /** 获取组织筛选显示文本 */
  const getScopeDisplayText = () => {
    if (scopeFilters.length === 0) return '全部组织';
    if (scopeFilters[0] === '__none__') return '未选择组织';
    const names: string[] = [];
    const groupFilterIds = scopeFilters.filter(id => id !== '__public__' && id !== '__none__' && id !== '__ungrouped__');
    const hasUngrouped = scopeFilters.includes('__ungrouped__');
    // 全部组织 = 所有组织 + 未分配组织
    if (groupFilterIds.length === groups.length && hasUngrouped) return '全部组织';
    groupFilterIds.forEach(id => {
      const g = groups.find(g => g.id === id);
      if (g) names.push(g.name);
    });
    if (hasUngrouped) names.push('未分配组织');
    return names.join('、') || '组织';
  };

  /** 获取实例的显示状态（运行时计算，pending_update 不是持久化状态） */
  const getInstanceFilterKey = (instance: AgentInstance): FilterOption | null => {
    if (!instance.distributionStatus || instance.distributionStatus === 'not_distributed') return 'not_distributed';
    if (instance.distributionStatus === 'failed') return 'failed';
    if (instance.distributionStatus === 'success') {
      if (!singleStatusFilter && skillVersion && instance.distributedVersion && instance.distributedVersion !== skillVersion) {
        return 'pending_update';
      }
      return 'success';
    }
    return null;
  };

  /** MCP 场景：26.3.28 前的 OpenClaw 实例不支持 MCP 服务 */
  const isUnsupportedMCPVersion = (instance: AgentInstance) => {
    if (!singleStatusFilter && !showVersionFilter) return false;
    if (!instance.agentVersion) return true;
    return compareVersion(instance.agentVersion, VERSION_THRESHOLD) < 0;
  };

  /** MCP 场景：已下发成功的实例不可再选 */
  const isAlreadyDistributed = (instance: AgentInstance) => {
    if (!singleStatusFilter) return false;
    return instance.distributionStatus === 'success';
  };

  /** 获取组织名称 */
  const getGroupName = (groupId: string) => {
    return groups.find(g => g.id === groupId)?.name || groupId;
  };

  const allFilteredInstances = instances
    .filter(instance => {
      // 仅显示运行中的实例
      if (instance.status !== 'running') return false;
      // 仅显示未下发、下发失败、待更新的实例
      const filterKey = getInstanceFilterKey(instance);
      if (!filterKey) return false;

      const matchesSearch = instance.name.toLowerCase().includes(searchQuery.toLowerCase()) || instance.id.toLowerCase().includes(searchQuery.toLowerCase());
      
      // 多选筛选逻辑：空数组 = 全部；['__none__'] = 全不选
      let matchesStatus = true;
      if (statusFilters.length > 0) {
        if ((statusFilters as any)[0] === '__none__') {
          matchesStatus = false;
        } else {
          matchesStatus = statusFilters.includes(filterKey);
        }
      }

      // 组织筛选（多选）：空数组 = 全部；['__none__'] = 全不选
      let matchesScope = true;
      if (scopeFilters.length > 0) {
        if (scopeFilters[0] === '__none__') {
          matchesScope = false;
        } else {
          const groupFilterIds = scopeFilters.filter(id => id !== '__public__' && id !== '__none__' && id !== '__ungrouped__');
          const hasUngrouped = scopeFilters.includes('__ungrouped__');
          const matchesGroup = groupFilterIds.length > 0 && instance.groupIds?.some(gId => groupFilterIds.includes(gId));
          const matchesUngrouped = hasUngrouped && (!instance.groupIds || instance.groupIds.length === 0 || instance.groupIds.every(gId => gId === '__public__'));
          matchesScope = matchesGroup || matchesUngrouped || false;
        }
      }

      return matchesSearch && matchesStatus && matchesScope;
    })
    // 版本筛选（仅 MCP 场景启用）
    .filter(instance => {
      if (!showVersionFilter || versionFilter === 'all') return true;
      const ver = instance.agentVersion;
      if (!ver) return versionFilter === 'lt_0328'; // 无版本信息视为旧版
      const cmp = compareVersion(ver, VERSION_THRESHOLD);
      if (versionFilter === 'gte_0328') return cmp >= 0;
      if (versionFilter === 'lt_0328') return cmp < 0;
      return true;
    })
    .sort((a, b) => {
      const aUnsupported = isUnsupportedMCPVersion(a);
      const bUnsupported = isUnsupportedMCPVersion(b);
      if (aUnsupported !== bUnsupported) return aUnsupported ? 1 : -1;
      return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
    });

  // 分页计算
  const totalCount = allFilteredInstances.length;
  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));
  const safeCurrentPage = Math.min(currentPage, totalPages);
  const startIndex = (safeCurrentPage - 1) * pageSize;
  const pagedInstances = allFilteredInstances.slice(startIndex, startIndex + pageSize);
  const selectableFilteredInstances = allFilteredInstances.filter(instance => !isUnsupportedMCPVersion(instance) && !isAlreadyDistributed(instance));
  const selectableFilteredInstanceIds = selectableFilteredInstances.map(i => i.id);
  const selectableFilteredInstanceIdsKey = selectableFilteredInstanceIds.join('\u0000');

  // 全选模式下，筛选条件变化时自动同步 selectedInstances
  useEffect(() => {
    if (!selectAllMode) return;
    setSelectedInstances(prev => {
      const unchanged = prev.length === selectableFilteredInstanceIds.length && prev.every((id, index) => id === selectableFilteredInstanceIds[index]);
      return unchanged ? prev : selectableFilteredInstanceIds;
    });
  }, [selectAllMode, selectableFilteredInstanceIdsKey]);

  /** 全选 / 取消全选 —— 跨页选中/取消所有符合筛选条件的可选实例 */
  const handleSelectAll = () => {
    if (selectAllMode) {
      setSelectedInstances([]);
      setSelectAllMode(false);
      return;
    }
    setSelectedInstances(selectableFilteredInstanceIds);
    setSelectAllMode(true);
  };

  const handleSelectInstance = (instance: AgentInstance) => {
    if (selectAllMode || isUnsupportedMCPVersion(instance) || isAlreadyDistributed(instance)) return;
    const id = instance.id;

    setSelectedInstances(prev => {
      if (prev.includes(id)) {
        return prev.filter(i => i !== id);
      }
      return [...prev, id];
    });
  };

  const hasRedistributeSelected = () => selectedInstances.some((id) => {
    const instance = instances.find(i => i.id === id);
    return instance?.distributionStatus === 'success' && getInstanceFilterKey(instance) === 'success';
  });

  const proceedDistributionFlow = () => {
    if (hasRedistributeSelected()) {
      setRedistributeConfirmOpen(true);
      return;
    }

    if (showConfirmDialog) {
      // 打开二次确认弹窗
      setConfirmInput('');
      setConfirmDialogOpen(true);
      return;
    }
    doDistribute();
  };

  const handleDistribute = () => {
    proceedDistributionFlow();
  };

  const doDistribute = () => {
    const selectedInstancesData = selectAllMode
      ? selectableFilteredInstances
      : instances.filter(i => selectedInstances.includes(i.id) && !isUnsupportedMCPVersion(i));
    const selectedInstanceIds = selectedInstancesData.map(i => i.id);

    if (selectedInstanceIds.length === 0) {
      setSelectedInstances([]);
      setSelectAllMode(false);
      setConfirmDialogOpen(false);
      setConfirmInput('');
      return;
    }
    
    if (onDistributionStart) {
      onDistributionStart(selectedInstanceIds, selectedInstancesData);
    }
    
    setSelectedInstances([]);
    setSelectAllMode(false);
    setSearchQuery('');
    setStatusFilters(singleStatusFilter ? [...DEFAULT_MCP_STATUS_FILTERS] : [...DEFAULT_DISTRIBUTE_STATUS_FILTERS]);
    setScopeFilters([]);
    setCurrentPage(1);
    setPageSize(20);
    setConfirmDialogOpen(false);
    setConfirmInput('');
    setRedistributeConfirmOpen(false);
    onOpenChange(false);
  };

  const handleConfirmDistribute = () => {
    if (confirmInput !== '确认下发') return;
    doDistribute();
  };

  const handleRedistributeConfirm = () => {
    setRedistributeConfirmOpen(false);
    if (showConfirmDialog) {
      setConfirmInput('');
      setConfirmDialogOpen(true);
      return;
    }
    doDistribute();
  };

  const getStatusDisplay = (instance: AgentInstance) => {
    const filterKey = getInstanceFilterKey(instance);
    const status = filterKey || instance.distributionStatus || 'not_distributed';
    const display = STATUS_DISPLAY_MAP[status as keyof typeof STATUS_DISPLAY_MAP] || STATUS_DISPLAY_MAP.not_distributed;

    if (filterKey === 'pending_update') {
      return (
        <div className="text-right">
          <span className={`text-xs font-medium ${display.className}`}>{display.label}</span>
          <div className="mt-0.5 text-center text-[11px] text-[var(--text-weak)]">v{instance.distributedVersion}</div>
        </div>
      );
    }

    return <span className={`text-xs font-medium ${display.className}`}>{display.label}</span>;
  };

  // 全选判断（跨页）：只统计符合筛选条件的可选实例
  const allIds = selectableFilteredInstanceIds;
  const selectedInFilterCount = allIds.filter(id => selectedInstances.includes(id)).length;
  const selectedSelectableCount = selectAllMode ? allIds.length : selectedInFilterCount;
  const isAllFilteredSelected = selectAllMode || (allIds.length > 0 && allIds.every(id => selectedInstances.includes(id)));
  const isIndeterminate = !selectAllMode && selectedInFilterCount > 0 && !isAllFilteredSelected;

  return (
    <>
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[86vh] max-w-[920px] flex-col">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription asChild>
            <div className="text-sm leading-6 text-[var(--text-muted)]">
              {descriptionNode || (
                <>
                  <p>将 <span className="font-semibold text-[var(--text-emphasis)]">{skillName}{skillVersion ? ` (${skillVersion})` : ''}</span> 部署至所选实例。</p>
                  <p className="mt-1">筛选限制：仅限状态为 <span className="font-medium text-[var(--text-secondary)]">运行中</span> 的实例；同时，该实例的下发状态须为 <span className="font-medium text-[var(--text-secondary)]">未下发</span>{showScopeFilter ? <>{' '}、 <span className="font-medium text-[var(--text-secondary)]">下发失败</span> 或 <span className="font-medium text-[var(--text-secondary)]">待更新</span></> : <>{' '}、 <span className="font-medium text-[var(--text-secondary)]">下发失败</span> 或 <span className="font-medium text-[var(--text-secondary)]">待更新</span></>}。</p>
                </>
              )}
            </div>
          </DialogDescription>
        </DialogHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
        {/* 搜索框 + 应用范围筛选 + 版本筛选 + 状态下拉 */}
        <div className="mb-4 flex gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--text-weak)]" />
            <Input
              placeholder="搜索实例名称/ID..."
              value={searchQuery}
              onChange={(e) => { setSearchQuery(e.target.value); setCurrentPage(1); }}
              className="pl-10"
            />
          </div>
          {/* 版本筛选 — MCP 场景 */}
          {showVersionFilter && (
            <Select
              value={versionFilter}
              onValueChange={(value) => {
                setVersionFilter(value as VersionFilterOption);
                setCurrentPage(1);
              }}
            >
              <SelectTrigger className="w-24 h-9">
                <span>版本</span>
              </SelectTrigger>
              <SelectContent>
                {VERSION_FILTER_OPTIONS.map(opt => (
                  <SelectItem key={opt.key} value={opt.key}>{opt.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
          {/* 组织筛选 — 扁平多选列表 */}
          {showScopeFilter && (
          <div className="relative" ref={scopeDropdownRef}>
            <Tooltip delayDuration={1000} open={scopeDropdownOpen ? false : undefined}>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    onClick={() => setScopeDropdownOpen(prev => !prev)}
                    className={`${dropdownTriggerClassName} w-32`}
                  >
                    <span className="truncate text-left">
                      {getScopeDisplayText()}
                    </span>
                    <ChevronDown className={`h-4 w-4 flex-shrink-0 text-[var(--text-weak)] transition-transform ${scopeDropdownOpen ? 'rotate-180' : ''}`} />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-[280px]">
                  <p className="break-words">{getScopeDisplayText()}</p>
                </TooltipContent>
              </Tooltip>
            {scopeDropdownOpen && (() => {
              const groupOnlyFilters = scopeFilters.filter(id => id !== '__public__' && id !== '__none__' && id !== '__ungrouped__');
              const hasUngrouped = scopeFilters.includes('__ungrouped__');
              const allGroupIds = groups.map(g => g.id);
              // 全部组织 = 所有组织 + 未分配组织
              const isAllGroupSelected = scopeFilters.length === 0 || (allGroupIds.every(id => groupOnlyFilters.includes(id)) && hasUngrouped);
              const selectedCount = groupOnlyFilters.length + (hasUngrouped ? 1 : 0);
              const isSomeGroupSelected = selectedCount > 0 && !isAllGroupSelected;
              const filteredGroups = groups.filter(g => g.name.toLowerCase().includes(scopeSearchQuery.toLowerCase()));
              const showUngrouped = !scopeSearchQuery || '未分配组织'.includes(scopeSearchQuery);

              return (
              <div className={`${dropdownPanelClassName} left-0 w-[220px]`}>
                {/* 搜索框 */}
                <div className="px-2 pb-1.5 pt-1.5">
                  <div className="relative">
                    <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[var(--text-weak)]" />
                    <input
                      placeholder="搜索组织..."
                      value={scopeSearchQuery}
                      onChange={(e) => setScopeSearchQuery(e.target.value)}
                      className="h-8 w-full rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--cp-surface)] pl-7 pr-2 text-sm text-[var(--text-emphasis)] outline-none transition-colors placeholder:text-[var(--text-weak)] hover:border-[var(--cp-brand-blue)] focus:border-[var(--cp-brand-blue)]"
                      onClick={(e) => e.stopPropagation()}
                      autoFocus
                    />
                  </div>
                </div>
                {/* 全部组织 */}
                {!scopeSearchQuery && (
                  <button
                    type="button"
                    onClick={() => {
                      if (isAllGroupSelected) {
                        // 全选 → 取消全部
                        setScopeFilters(['__none__']);
                      } else {
                        // 非全选 → 选中所有（全部组织 + 未分配组织）
                        setScopeFilters([...allGroupIds, '__ungrouped__']);
                      }
                      setCurrentPage(1);
                    }}
                    className={`${dropdownOptionClassName} border-b border-[var(--cp-border)]`}
                  >
                    <div className={getCustomCheckboxClassName(isAllGroupSelected || isSomeGroupSelected)}>
                      {isAllGroupSelected && <Check className="w-3 h-3 text-white" />}
                      {isSomeGroupSelected && <div className="w-2 h-0.5 bg-white rounded-sm" />}
                    </div>
                    <span>全部组织</span>
                  </button>
                )}
                {/* 组织列表 */}
                <div className="max-h-[200px] overflow-y-auto">
                  {filteredGroups.map(group => {
                    const isSelected = isAllGroupSelected || groupOnlyFilters.includes(group.id);
                    return (
                      <button
                        key={group.id}
                        type="button"
                        onClick={() => {
                          setScopeFilters(prev => {
                            const cleaned = prev.filter(id => id !== '__public__' && id !== '__none__');
                            const hasUng = cleaned.includes('__ungrouped__');
                            const grpOnly = cleaned.filter(id => id !== '__ungrouped__');
                            // 如果当前是"全部"(空数组)，点击某项 = 取消该项
                            if (prev.length === 0) {
                              const remaining = allGroupIds.filter(id => id !== group.id);
                              return [...remaining, '__ungrouped__'];
                            }
                            const next = grpOnly.includes(group.id)
                              ? grpOnly.filter(id => id !== group.id)
                              : [...grpOnly, group.id];
                            const combined = hasUng ? [...next, '__ungrouped__'] : next;
                            if (combined.length === 0) return ['__none__'];
                            // 全部选中 → 重置为空（全部组织）
                            if (next.length === allGroupIds.length && hasUng) return [];
                            return combined;
                          });
                          setCurrentPage(1);
                        }}
                        className={dropdownOptionClassName}
                      >
                        <div className={getCustomCheckboxClassName(isSelected)}>
                          {isSelected && <Check className="w-3 h-3 text-white" />}
                        </div>
                        <span className="truncate text-left" title={group.name}>{group.name}</span>
                      </button>
                    );
                  })}
                  {/* 未分配组织 */}
                  {showUngrouped && (
                    <button
                      type="button"
                      onClick={() => {
                        setScopeFilters(prev => {
                          const cleaned = prev.filter(id => id !== '__public__' && id !== '__none__');
                          const grpOnly = cleaned.filter(id => id !== '__ungrouped__');
                          const hadUng = cleaned.includes('__ungrouped__');
                          // 如果当前是"全部"(空数组)，点击未分配组织 = 取消它
                          if (prev.length === 0) {
                            return [...allGroupIds]; // 保留所有组织，移除未分配组织
                          }
                          if (hadUng) {
                            // 取消未分配组织
                            const result = grpOnly.length > 0 ? grpOnly : ['__none__'];
                            return result;
                          } else {
                            // 选中未分配组织
                            const combined = [...grpOnly, '__ungrouped__'];
                            // 全选判断
                            if (grpOnly.length === allGroupIds.length) return [];
                            return combined;
                          }
                        });
                        setCurrentPage(1);
                      }}
                      className={dropdownOptionClassName}
                    >
                      <div className={getCustomCheckboxClassName(isAllGroupSelected || hasUngrouped)}>
                        {(isAllGroupSelected || hasUngrouped) && <Check className="w-3 h-3 text-white" />}
                      </div>
                      <span className="text-[var(--text-muted)]">未分配组织</span>
                    </button>
                  )}
                  {filteredGroups.length === 0 && !showUngrouped && scopeSearchQuery && (
                    <p className="py-3 text-center text-xs text-[var(--text-weak)]">没有匹配的组织</p>
                  )}
                </div>
                {/* 底部统计 + 清除筛选 */}
                {selectedCount > 0 && !isAllGroupSelected && (
                  <div className="flex items-center justify-between border-t border-[var(--cp-border)] px-3 py-2 text-xs">
                    <span className="text-[var(--text-muted)]">已选 {selectedCount} 个组织</span>
                    <button
                      type="button"
                      onClick={() => { setScopeFilters([]); setCurrentPage(1); }}
                      className="font-medium text-[var(--text-brand)] hover:underline"
                    >
                      清除筛选
                    </button>
                  </div>
                )}
              </div>
              );
            })()}
          </div>
          )}
          <div className="relative" ref={filterDropdownRef}>
            <Tooltip delayDuration={1000} open={filterDropdownOpen ? false : undefined}>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  onClick={() => setFilterDropdownOpen(prev => !prev)}
                  className={`${dropdownTriggerClassName} w-36`}
                >
                  <span className="truncate text-left">{getFilterDisplayText()}</span>
                  <ChevronDown className={`h-4 w-4 flex-shrink-0 text-[var(--text-weak)] transition-transform ${filterDropdownOpen ? 'rotate-180' : ''}`} />
                </button>
              </TooltipTrigger>
              <TooltipContent side="bottom" className="max-w-[280px]">
                <p className="break-words">{getFilterDisplayText()}</p>
              </TooltipContent>
            </Tooltip>
            {filterDropdownOpen && (
              <div className={`${dropdownPanelClassName} right-0 w-36`}>
                <button
                  type="button"
                  onClick={() => {
                    setStatusFilters(isAllStatusSelected ? (['__none__'] as any) : []);
                    setCurrentPage(1);
                  }}
                  className={dropdownOptionClassName}
                >
                  <div className={getCustomCheckboxClassName(isAllStatusSelected)}>
                    {isAllStatusSelected && <Check className="w-3 h-3 text-white" />}
                  </div>
                  <span>全部下发状态</span>
                </button>
                {activeStatusOptions.map(opt => {
                  const isOptSelected = isAllStatusSelected || (!(statusFilters as any).includes('__none__') && statusFilters.includes(opt.key));
                  return (
                    <button
                      key={opt.key}
                      type="button"
                      onClick={() => {
                        setStatusFilters(prev => {
                          const cleaned = (prev as any).filter((k: string) => k !== '__none__') as FilterOption[];
                          if (prev.length === 0 || prev.length === activeStatusOptions.length) {
                            return activeStatusOptions.filter(o => o.key !== opt.key).map(o => o.key);
                          }
                          const next = cleaned.includes(opt.key)
                            ? cleaned.filter(k => k !== opt.key)
                            : [...cleaned, opt.key];
                          if (next.length === 0) return ['__none__'] as any;
                          if (next.length === activeStatusOptions.length) return [];
                          return next;
                        });
                        setCurrentPage(1);
                      }}
                      className={dropdownOptionClassName}
                    >
                      <div className={getCustomCheckboxClassName(isOptSelected)}>
                        {isOptSelected && <Check className="w-3 h-3 text-white" />}
                      </div>
                      <span>{opt.label}</span>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        </div>

        {/* 实例列表 */}
        <div className="max-h-[340px] overflow-y-auto rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--cp-surface)]">
          {/* 全选复选框 — 跨页全选当前筛选结果 */}
          <div className="sticky top-0 z-10 flex items-center justify-between border-b border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-3 py-2.5">
            <div className="flex items-center gap-3">
              <Checkbox
                checked={isAllFilteredSelected ? true : isIndeterminate ? 'indeterminate' : false}
                onCheckedChange={handleSelectAll}
                disabled={allIds.length === 0}
                aria-label={isAllFilteredSelected ? '取消全选' : '全选'}
              />
              <span className="text-sm font-medium text-[var(--text-emphasis)]">
                全选
              </span>
            </div>
            {selectedSelectableCount > 0 && (
              <span className="text-sm text-[var(--text-muted)]">
                已选 {selectedSelectableCount} 条
              </span>
            )}
          </div>

          {/* 实例项 */}
          {pagedInstances.length === 0 ? (
            <div className="flex items-center justify-center py-8 text-sm text-[var(--text-weak)]">
              暂无匹配的实例
            </div>
          ) : (
            pagedInstances.map(instance => {
              const unsupportedMCPVersion = isUnsupportedMCPVersion(instance);
              const alreadyDistributed = isAlreadyDistributed(instance);
              const isDisabled = unsupportedMCPVersion || alreadyDistributed;
              const isRowDisabled = isDisabled || selectAllMode;
              const isSelected = selectAllMode ? !isDisabled : selectedInstances.includes(instance.id);
              const isSuccessStatus = getInstanceFilterKey(instance) === 'success';
              const rowContent = (
                <div
                  key={instance.id}
                  onClick={() => !isRowDisabled && handleSelectInstance(instance)}
                  className={`flex items-center gap-3 border-b border-[var(--cp-border)] px-3 py-3 transition-colors last:border-b-0 ${
                    isRowDisabled ? 'cursor-not-allowed' : 'cursor-pointer hover:bg-[var(--bg-grey-hover-subtle)]'
                  } ${isDisabled ? 'opacity-60' : ''} ${isSelected ? 'bg-[var(--bg-brand-selected)]' : ''}`}
                >
                  <div className="flex-shrink-0">
                    <Checkbox
                      checked={isSelected}
                      disabled={isRowDisabled}
                      onCheckedChange={() => handleSelectInstance(instance)}
                      onClick={(e) => e.stopPropagation()}
                      aria-label={`选择 ${instance.name}`}
                    />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-3">
                      <span className="truncate text-sm font-medium text-[var(--text-emphasis)]">{instance.name}</span>
                      <span className="flex-shrink-0 font-mono text-xs text-[var(--text-weak)]">{instance.id}</span>
                    </div>
                    {/* Agent 类型和版本信息 — MCP 场景显示 */}
                    {(singleStatusFilter || showVersionFilter) && instance.agentType && (
                      <div className="mt-0.5 text-xs text-[var(--text-weak)]">
                        {instance.agentType}{instance.agentVersion ? `(${instance.agentVersion})` : ''}
                      </div>
                    )}
                    {!hideCreatorAndGroup && (
                    <div className="flex items-center gap-3 mt-0.5">
                      <span className="text-xs text-[var(--text-muted)]">创建人：{instance.createdBy}</span>
                      {(() => {
                        const groupText = instance.groupIds && instance.groupIds.length > 0
                          ? instance.groupIds.filter(gId => gId !== '__public__').map(gId => getGroupName(gId)).join('、')
                          : '';
                        const displayText = groupText || '-';
                        return (
                          <Tooltip delayDuration={300}>
                            <TooltipTrigger asChild>
                              <span className="inline-block max-w-[180px] cursor-default truncate align-bottom text-xs text-[var(--text-muted)]">
                                组织：{displayText}
                              </span>
                            </TooltipTrigger>
                            {displayText !== '-' && displayText.length > 10 && (
                              <TooltipContent side="top" className="max-w-[320px]">
                                <p className="break-words">组织：{displayText}</p>
                              </TooltipContent>
                            )}
                          </Tooltip>
                        );
                      })()}
                    </div>
                    )}
                  </div>
                  <div className="flex-shrink-0 self-center">
                    {getStatusDisplay(instance)}
                  </div>
                </div>
              );

              if (!isDisabled && !isSuccessStatus) return rowContent;

              return (
                <Tooltip key={instance.id} delayDuration={300}>
                  <TooltipTrigger asChild>{rowContent}</TooltipTrigger>
                  <TooltipContent side="top">
                    <span className="text-xs">{unsupportedMCPVersion ? '版本过低，不支持MCP服务。' : '已下发当前版本，可再次下发'}</span>
                  </TooltipContent>
                </Tooltip>
              );
            })
          )}
        </div>

        <div className="pt-3">
          {selectAllMode && selectedSelectableCount > 0 && (
            <p className="mb-1 text-xs text-[var(--cp-text-muted)]">已选择全部符合条件的实例</p>
          )}
          <DialogPagination
            total={totalCount}
            currentPage={safeCurrentPage}
            totalPages={totalPages}
            onPrevPage={() => {
              setCurrentPage((p) => Math.max(1, p - 1));
              if (!selectAllMode) setSelectedInstances([]);
            }}
            onNextPage={() => {
              setCurrentPage((p) => Math.min(totalPages, p + 1));
              if (!selectAllMode) setSelectedInstances([]);
            }}
          />
        </div>
        </div>

        <DialogFooter className="gap-2 sm:gap-2">
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            variant="dialog-confirm"
            onClick={handleDistribute}
            disabled={selectedSelectableCount === 0}
          >
            确认下发（{selectedSelectableCount}）
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    {/* 二次确认弹窗 */}
    <AlertDialog open={confirmDialogOpen} onOpenChange={setConfirmDialogOpen}>
      <AlertDialogContent className="sm:max-w-md">
        <AlertDialogHeader>
          <AlertDialogTitle className="flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-[var(--alert-warning-icon)]" />
            风险提示
          </AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-4">
              <div className="flex items-start gap-2.5 rounded-[var(--radius-lg)] border border-[var(--alert-warning-border)] bg-[var(--alert-warning-bg)] px-4 py-3">
                <p className="text-sm leading-relaxed text-[var(--alert-warning-foreground)]">
                  配置 MCP 会修改 <code className="rounded-[3px] bg-[var(--cp-surface)] px-1 py-0.5 font-mono text-xs text-[var(--text-emphasis)]">~/.openclaw/openclaw.json</code> 文件中的 <code className="rounded-[3px] bg-[var(--cp-surface)] px-1 py-0.5 font-mono text-xs text-[var(--text-emphasis)]">mcp.servers</code> 相关配置，修改后需重启 gateway 生效，将会导致实例短暂不可用，可能影响正在运行的任务。
                </p>
              </div>
              <div>
                <p className="mb-2 text-sm text-[var(--text-secondary)]">请输入<span className="font-semibold text-[var(--text-emphasis)]">「确认下发」</span>后开始执行。</p>
                <Input
                  value={confirmInput}
                  onChange={(e) => setConfirmInput(e.target.value)}
                  placeholder="确认下发"
                  className="mt-1"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && confirmInput === '确认下发') {
                      handleConfirmDistribute();
                    }
                  }}
                />
              </div>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => { setConfirmDialogOpen(false); setConfirmInput(''); }}>
            取消
          </AlertDialogCancel>
          <Button
            variant="dialog-confirm"
            onClick={handleConfirmDistribute}
            disabled={confirmInput !== '确认下发'}
          >
            确认下发
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>


    {/* 已下发成功实例再次下发确认 */}
    <AlertDialog open={redistributeConfirmOpen} onOpenChange={setRedistributeConfirmOpen}>
      <AlertDialogContent className="sm:max-w-md">
        <AlertDialogHeader>
          <AlertDialogTitle>确认再次下发</AlertDialogTitle>
          <AlertDialogDescription>
            此次操作将覆盖原有配置，请确认是否继续。
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel onClick={() => setRedistributeConfirmOpen(false)}>
            取消
          </AlertDialogCancel>
          <Button
            variant="dialog-confirm"
            onClick={handleRedistributeConfirm}
          >
            确认
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
    </>
  );
}
