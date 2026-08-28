import { useState, useEffect, useRef } from 'react';
import {
  Dialog,
  DialogContent,
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
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  InstantMultiSelect,
} from '@/components/ui/select';
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table';
import { Pagination } from '@/components/ui/pagination';
import { Search, ChevronDown, Check, AlertTriangle } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { DISTRIBUTION_STATUS_MAP, type SkillScope, type AgentInstance, type Group } from './types';

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

interface BatchDistributeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  skillId?: string;
  skillName?: string;
  /** 当前 Skill 最新版本号，用于判定"待更新" */
  skillVersion?: string;
  /** 当前 Skill 的应用范围 */
  skillScope?: SkillScope;
  /** 当前 Skill 关联的分组 ID 列表 */
  skillGroupIds?: string[];
  onDistributionStart?: (selectedInstanceIds: string[], selectedInstancesData: any[]) => void;
  /** 弹窗标题，默认 "批量下发 Skill" */
  title?: string;
  /** 是否显示应用范围筛选，默认 true */
  showScopeFilter?: boolean;
  /** Agent 实例列表（外部传入） */
  instances: AgentInstance[];
  /** 分组列表（外部传入，showScopeFilter=true 时必传） */
  groups?: Group[];
  /** MCP 场景：隐藏实例列表中的创建人、分组信息 */
  hideCreatorAndGroup?: boolean;
  /** MCP 场景：下发状态筛选改为单选下拉（只有 未下发 / 下发失败），去掉待更新和多选逻辑 */
  singleStatusFilter?: boolean;
  /** MCP 场景：自定义描述 ReactNode，覆盖默认描述 */
  descriptionNode?: React.ReactNode;
  /** MCP 场景：显示版本筛选下拉，默认 false */
  showVersionFilter?: boolean;
  /** MCP 场景：下发前需要二次确认弹窗，默认 false */
  showConfirmDialog?: boolean;
  /** 在实例行展示 Agent 类型/版本信息 */
  showAgentType?: boolean;
}

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

export default function BatchDistributeDialog({
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
  showAgentType = false,
}: BatchDistributeDialogProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedInstances, setSelectedInstances] = useState<string[]>([]);
  /** 是否处于"选择全部"模式（跨页全选） */
  const [selectAllMode, setSelectAllMode] = useState(false);
  /** 状态多选筛选（Set 存储选中的 key） */
  const [statusFilters, setStatusFilters] = useState<Set<string>>(new Set());
  /** 应用范围筛选：空数组=全部, 否则为选中的分组 ID 列表（多选） */
  const [scopeFilters, setScopeFilters] = useState<string[]>([]);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  /** 分组筛选下拉 */
  const [scopeDropdownOpen, setScopeDropdownOpen] = useState(false);
  const [scopeSearchQuery, setScopeSearchQuery] = useState('');
  const scopeDropdownRef = useRef<HTMLDivElement>(null);
  /** 版本筛选 */
  const [versionFilter, setVersionFilter] = useState<VersionFilterOption>('all');
  /** 二次确认弹窗（MCP 场景风险提示） */
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false);
  const [confirmInput, setConfirmInput] = useState('');

  // 点击外部关闭下拉
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
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
      // MCP 场景默认选中「未下发」+「下发失败」；Skill 场景默认选中「未下发」+「下发失败」
      setStatusFilters(new Set(singleStatusFilter ? DEFAULT_MCP_STATUS_FILTERS : ['not_distributed', 'failed']));
      setSearchQuery('');
      setCurrentPage(1);
      setPageSize(20);
      setSelectAllMode(false);
      setScopeDropdownOpen(false);
      setScopeSearchQuery('');
      setVersionFilter('all');
      setConfirmDialogOpen(false);
      setConfirmInput('');
      // 根据 Skill 应用范围设置默认筛选
      if (showScopeFilter) {
        if ((skillScope === 'private' || skillScope === 'groups') && skillGroupIds && skillGroupIds.length > 0) {
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

  /** 判断当前是否为"全部状态" */
  const isAllStatusSelected = statusFilters.size === 0 || statusFilters.size === activeStatusOptions.length;

  /** 获取分组筛选显示文本 */
  const getScopeDisplayText = () => {
    if (scopeFilters.length === 0) return '全部分组';
    if (scopeFilters[0] === '__none__') return '分组';
    const names: string[] = [];
    const groupFilterIds = scopeFilters.filter(id => id !== '__public__' && id !== '__none__' && id !== '__ungrouped__');
    const hasUngrouped = scopeFilters.includes('__ungrouped__');
    // 全部分组 = 所有分组 + 未分组
    if (groupFilterIds.length === scopeFilterGroups.length && hasUngrouped) return '全部分组';
    groupFilterIds.forEach(id => {
      const g = scopeFilterGroups.find(g => g.id === id) || groups.find(g => g.id === id);
      if (g) names.push(g.name);
    });
    if (hasUngrouped) names.push('未分组');
    return names.join('、') || '分组';
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

  /** 获取分组名称 */
  const getGroupName = (groupId: string) => {
    return groups.find(g => g.id === groupId)?.name || groupId;
  };

  const isScopedToGroups = showScopeFilter && (skillScope === 'private' || skillScope === 'groups');
  const allowedScopeGroupIds = isScopedToGroups ? (skillGroupIds || []).filter(Boolean) : [];
  const allowedScopeGroupNames = allowedScopeGroupIds.map(getGroupName).join('、');
  const scopeFilterGroups = allowedScopeGroupIds.length > 0
    ? groups.filter(group => allowedScopeGroupIds.includes(group.id))
    : groups;

  const allFilteredInstances = instances
    .filter(instance => {
      // 仅显示运行中的实例
      if (instance.status !== 'running') return false;
      // 资产本身配置了应用范围时，只允许范围内实例进入候选列表。
      if (isScopedToGroups) {
        if (allowedScopeGroupIds.length === 0) return false;
        const instanceGroupIds = instance.groupIds || [];
        if (!instanceGroupIds.some(gId => allowedScopeGroupIds.includes(gId))) return false;
      }
      // 仅显示可归类到下发状态筛选的实例
      const filterKey = getInstanceFilterKey(instance);
      if (!filterKey) return false;

      const matchesSearch = instance.name.toLowerCase().includes(searchQuery.toLowerCase()) || instance.id.toLowerCase().includes(searchQuery.toLowerCase());
      
      // 多选筛选逻辑：空集合 = 全部
      let matchesStatus = true;
      if (statusFilters.size > 0) {
        matchesStatus = statusFilters.has(filterKey);
      }

      // 分组筛选（多选）：空数组 = 全部；['__none__'] = 全不选
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
  const handleToggleSelectAll = () => {
    if (selectAllMode) {
      setSelectedInstances([]);
      setSelectAllMode(false);
    } else {
      setSelectedInstances(selectableFilteredInstanceIds);
      setSelectAllMode(true);
    }
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

  const handleDistribute = () => {
    if (showConfirmDialog) {
      // 打开二次确认弹窗（MCP 风险提示）
      setConfirmInput('');
      setConfirmDialogOpen(true);
      return;
    }
    doDistribute();
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
    setConfirmDialogOpen(false);
    setConfirmInput('');
    setSearchQuery('');
    setStatusFilters(new Set(singleStatusFilter ? DEFAULT_MCP_STATUS_FILTERS : DEFAULT_DISTRIBUTE_STATUS_FILTERS));
    setScopeFilters([]);
    setCurrentPage(1);
    setPageSize(20);
    setConfirmDialogOpen(false);
    setConfirmInput('');
    onOpenChange(false);
  };

  const handleConfirmDistribute = () => {
    if (confirmInput !== '确认下发') return;
    doDistribute();
  };

  const getStatusDisplay = (instance: AgentInstance) => {
    const filterKey = getInstanceFilterKey(instance);
    // 待更新：黄色样式 + 老版本号
    if (filterKey === 'pending_update') {
      return (
        <div className="text-right">
          <span className="inline-block px-2 py-0.5 rounded-full text-xs font-medium text-yellow-700 bg-yellow-50">待更新</span>
          <div className="text-[11px] text-gray-400 mt-0.5 text-center">v{instance.distributedVersion}</div>
        </div>
      );
    }
    if (filterKey === 'success') {
      return <span className="inline-block px-2 py-0.5 rounded-full text-xs font-medium text-green-700 bg-green-50">下发成功</span>;
    }
    const s = instance.distributionStatus || 'not_distributed';
    const { label, color } = DISTRIBUTION_STATUS_MAP[s];
    return <span className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${color}`}>{label}</span>;
  };

  // 全选判断（跨页）：selectAllMode 下视为所有可选实例已选中
  const allIds = selectableFilteredInstanceIds;
  const selectedSelectableCount = selectAllMode ? allIds.length : selectableFilteredInstances.filter(i => selectedInstances.includes(i.id)).length;
  const isAllFilteredSelected = selectAllMode || (allIds.length > 0 && allIds.every(id => selectedInstances.includes(id)));
  const isIndeterminate = !selectAllMode && selectedInstances.length > 0 && !isAllFilteredSelected;

  return (
    <>
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>

        <div className="text-sm text-muted-foreground mb-4">
          {descriptionNode || (
            <>
              <p>将 <span className="font-semibold text-gray-900">{skillName}{skillVersion ? ` (${skillVersion})` : ''}</span> 部署至所选实例。</p>
              {isScopedToGroups && (
                <p className="mt-1">
                  应用范围限制：仅展示属于 <span className="font-medium text-gray-700">{allowedScopeGroupNames || '已选分组'}</span> 的 Agent。
                </p>
              )}
              <p className="mt-1">筛选限制：仅限状态为 <span className="font-medium text-gray-700">运行中</span> 的实例；同时，该实例的下发状态须为 <span className="font-medium text-gray-700">未下发</span>{showScopeFilter ? <>{' '}、 <span className="font-medium text-gray-700">下发失败</span> 或 <span className="font-medium text-gray-700">待更新</span></> : <>{' '}、 <span className="font-medium text-gray-700">下发失败</span> 或 <span className="font-medium text-gray-700">待更新</span></>}。</p>
            </>
          )}
        </div>

        {/* 搜索框 + 应用范围筛选 + 版本筛选 + 状态下拉 */}
        <div className="flex gap-2 mb-4">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
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
          {/* 分组筛选 — 扁平多选列表 */}
          {showScopeFilter && (
          <div className="relative" ref={scopeDropdownRef}>
            <Tooltip delayDuration={1000} open={scopeDropdownOpen ? false : undefined}>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    onClick={() => setScopeDropdownOpen(prev => !prev)}
                    className="flex items-center justify-between gap-1 w-32 h-9 px-3 border border-gray-200 rounded-md bg-white text-sm text-gray-700 hover:bg-gray-50 transition-colors"
                  >
                    <span className="truncate text-left">
                      {getScopeDisplayText()}
                    </span>
                    <ChevronDown className={`w-4 h-4 text-gray-400 flex-shrink-0 transition-transform ${scopeDropdownOpen ? 'rotate-180' : ''}`} />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="max-w-[280px]">
                  <p className="break-words">{getScopeDisplayText()}</p>
                </TooltipContent>
              </Tooltip>
            {scopeDropdownOpen && (() => {
              const groupOnlyFilters = scopeFilters.filter(id => id !== '__public__' && id !== '__none__' && id !== '__ungrouped__');
              const hasUngrouped = scopeFilters.includes('__ungrouped__');
              const allGroupIds = scopeFilterGroups.map(g => g.id);
              // 全部分组 = 所有分组 + 未分组
              const isAllGroupSelected = scopeFilters.length === 0 || (allGroupIds.every(id => groupOnlyFilters.includes(id)) && hasUngrouped);
              const selectedCount = groupOnlyFilters.length + (hasUngrouped ? 1 : 0);
              const isSomeGroupSelected = selectedCount > 0 && !isAllGroupSelected;
              const filteredGroups = scopeFilterGroups.filter(g => g.name.toLowerCase().includes(scopeSearchQuery.toLowerCase()));
              const showUngrouped = !isScopedToGroups && (!scopeSearchQuery || '未分组'.includes(scopeSearchQuery));

              return (
              <div className="absolute left-0 top-full mt-1 w-[220px] bg-white border border-gray-200 rounded-lg shadow-lg z-50 py-1">
                {/* 搜索框 */}
                <div className="px-2 pb-1.5 pt-1.5">
                  <div className="relative">
                    <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400" />
                    <input
                      placeholder="搜索分组..."
                      value={scopeSearchQuery}
                      onChange={(e) => setScopeSearchQuery(e.target.value)}
                      className="w-full pl-7 pr-2 h-8 text-sm border border-gray-200 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                      onClick={(e) => e.stopPropagation()}
                      autoFocus
                    />
                  </div>
                </div>
                {/* 全部分组 */}
                {!scopeSearchQuery && (
                  <button
                    type="button"
                    onClick={() => {
                      if (isAllGroupSelected) {
                        // 全选 → 取消全部
                        setScopeFilters(['__none__']);
                      } else {
                        // 非全选 → 选中所有（全部分组 + 未分组）
                        setScopeFilters([...allGroupIds, '__ungrouped__']);
                      }
                      setCurrentPage(1);
                    }}
                    className="flex items-center gap-2 w-full px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 transition-colors border-b border-gray-100"
                  >
                    <div className={`w-4 h-4 rounded border flex items-center justify-center shrink-0 ${
                      isAllGroupSelected ? 'bg-blue-600 border-blue-600' : isSomeGroupSelected ? 'bg-blue-600 border-blue-600' : 'border-gray-300'
                    }`}>
                      {isAllGroupSelected && <Check className="w-3 h-3 text-white" />}
                      {isSomeGroupSelected && <div className="w-2 h-0.5 bg-white rounded-sm" />}
                    </div>
                    <span>全部分组</span>
                  </button>
                )}
                {/* 分组列表 */}
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
                            // 全部选中 → 重置为空（全部分组）
                            if (next.length === allGroupIds.length && hasUng) return [];
                            return combined;
                          });
                          setCurrentPage(1);
                        }}
                        className="flex items-center gap-2 w-full px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 transition-colors"
                      >
                        <div className={`w-4 h-4 rounded border flex items-center justify-center shrink-0 ${
                          isSelected ? 'bg-blue-600 border-blue-600' : 'border-gray-300'
                        }`}>
                          {isSelected && <Check className="w-3 h-3 text-white" />}
                        </div>
                        <span className="truncate text-left" title={group.name}>{group.name}</span>
                      </button>
                    );
                  })}
                  {/* 未分组 */}
                  {showUngrouped && (
                    <button
                      type="button"
                      onClick={() => {
                        setScopeFilters(prev => {
                          const cleaned = prev.filter(id => id !== '__public__' && id !== '__none__');
                          const grpOnly = cleaned.filter(id => id !== '__ungrouped__');
                          const hadUng = cleaned.includes('__ungrouped__');
                          // 如果当前是"全部"(空数组)，点击未分组 = 取消它
                          if (prev.length === 0) {
                            return [...allGroupIds]; // 保留所有分组，移除未分组
                          }
                          if (hadUng) {
                            // 取消未分组
                            const result = grpOnly.length > 0 ? grpOnly : ['__none__'];
                            return result;
                          } else {
                            // 选中未分组
                            const combined = [...grpOnly, '__ungrouped__'];
                            // 全选判断
                            if (grpOnly.length === allGroupIds.length) return [];
                            return combined;
                          }
                        });
                        setCurrentPage(1);
                      }}
                      className="flex items-center gap-2 w-full px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 transition-colors"
                    >
                      <div className={`w-4 h-4 rounded border flex items-center justify-center shrink-0 ${
                        (isAllGroupSelected || hasUngrouped) ? 'bg-blue-600 border-blue-600' : 'border-gray-300'
                      }`}>
                        {(isAllGroupSelected || hasUngrouped) && <Check className="w-3 h-3 text-white" />}
                      </div>
                      <span className="text-gray-500">未分组</span>
                    </button>
                  )}
                  {filteredGroups.length === 0 && !showUngrouped && scopeSearchQuery && (
                    <p className="text-xs text-gray-400 py-3 text-center">没有匹配的分组</p>
                  )}
                </div>
                {/* 底部统计 + 清除筛选 */}
                {selectedCount > 0 && !isAllGroupSelected && (
                  <div className="flex items-center justify-between px-3 py-2 border-t border-gray-100 text-xs">
                    <span className="text-gray-500">已选 {selectedCount} 个分组</span>
                    <button
                      type="button"
                      onClick={() => { setScopeFilters([]); setCurrentPage(1); }}
                      className="text-blue-600 hover:text-blue-700 font-medium"
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
          <InstantMultiSelect
            options={activeStatusOptions.map(opt => ({ value: opt.key, label: opt.label }))}
            value={statusFilters}
            onChange={(next) => { setStatusFilters(next); setCurrentPage(1); }}
            placeholder="全部下发状态"
            selectAllLabel="全部下发状态"
            searchable={false}
            showFooter={false}
            align="end"
            triggerClassName="w-36"
          />
        </div>

        {/* 实例列表 — 使用项目规范 Table 组件 compact 密度 */}
        <div className="border border-gray-200 rounded-[4px] overflow-hidden">
          {/* 表头区域（不参与滚动） */}
          <Table density="compact" variant="gray-header">
            <TableHeader>
              <TableRow>
                <TableHead style={{ width: 48 }}>
                  <Checkbox
                    checked={isAllFilteredSelected ? true : isIndeterminate ? 'indeterminate' : false}
                    onCheckedChange={handleToggleSelectAll}
                    disabled={allIds.length === 0}
                    aria-label={isAllFilteredSelected ? '取消全选' : '全选所有符合条件的实例'}
                  />
                </TableHead>
                <TableHead>实例信息</TableHead>
                <TableHead className="text-right" style={{ width: 100 }}>下发状态</TableHead>
              </TableRow>
            </TableHeader>
          </Table>

          {/* 表格内容区（独立滚动，scrollbar-on-hover 仅在滚动时显示滚动条） */}
          <div className="max-h-[280px] overflow-y-auto scrollbar-on-hover">
            <Table density="compact" variant="gray-header">
              <TableBody>
                {pagedInstances.length === 0 ? (
                  <TableRow className="hover:!bg-transparent">
                    <TableCell colSpan={3} className="!h-auto text-center py-8 text-gray-400">
                      暂无匹配的实例
                    </TableCell>
                  </TableRow>
                ) : (
                  pagedInstances.map(instance => {
                    const unsupportedMCPVersion = isUnsupportedMCPVersion(instance);
                    const alreadyDistributed = isAlreadyDistributed(instance);
                    const isInstanceDisabled = unsupportedMCPVersion || alreadyDistributed;
                    const isRowDisabled = isInstanceDisabled || selectAllMode;
                    const isSelected = selectAllMode ? !isInstanceDisabled : selectedInstances.includes(instance.id);
                    const rowClassName = isInstanceDisabled
                      ? 'cursor-not-allowed opacity-60'
                      : selectAllMode
                        ? 'cursor-not-allowed'
                        : 'cursor-pointer';

                    const row = (
                      <TableRow
                        key={instance.id}
                        data-state={isSelected ? 'selected' : undefined}
                        className={rowClassName}
                        onClick={() => !isRowDisabled && handleSelectInstance(instance)}
                      >
                        <TableCell style={{ width: 48 }} onClick={(e) => e.stopPropagation()}>
                          <Checkbox
                            checked={isSelected}
                            disabled={isRowDisabled}
                            onCheckedChange={() => handleSelectInstance(instance)}
                            aria-label={`选择 ${instance.name}`}
                          />
                        </TableCell>
                        <TableCell>
                          <div className="min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="font-medium truncate">{instance.name}</span>
                              <span className="text-gray-400 flex-shrink-0">{instance.id}</span>
                            </div>
                            {(showAgentType || singleStatusFilter || showVersionFilter) && instance.agentType && (
                              <div className="text-gray-400 mt-0.5">
                                {instance.agentType === 'LocalAgent' ? `本地 Agent${instance.localProduct ? ` · ${instance.localProduct}` : ''}` : instance.agentType}
                                {instance.agentVersion ? `(${instance.agentVersion})` : ''}
                              </div>
                            )}
                            {!hideCreatorAndGroup && (
                              <div className="flex items-center gap-3 mt-0.5">
                                <span className="text-gray-500">创建人：{instance.createdBy}</span>
                                {(() => {
                                  const groupText = instance.groupIds && instance.groupIds.length > 0
                                    ? instance.groupIds.filter(gId => gId !== '__public__').map(gId => getGroupName(gId)).join('、')
                                    : '';
                                  const displayText = groupText || '-';
                                  return (
                                    <Tooltip delayDuration={300}>
                                      <TooltipTrigger asChild>
                                        <span className="max-w-[180px] truncate inline-block align-bottom cursor-default text-gray-500">
                                          分组：{displayText}
                                        </span>
                                      </TooltipTrigger>
                                      {displayText !== '-' && displayText.length > 10 && (
                                        <TooltipContent side="top" className="max-w-[320px]">
                                          <p className="break-words">分组：{displayText}</p>
                                        </TooltipContent>
                                      )}
                                    </Tooltip>
                                  );
                                })()}
                              </div>
                            )}
                          </div>
                        </TableCell>
                        <TableCell className="text-right" style={{ width: 100 }}>
                          {getStatusDisplay(instance)}
                        </TableCell>
                      </TableRow>
                    );

                    if (!isInstanceDisabled) return row;

                    return (
                      <Tooltip key={instance.id} delayDuration={300}>
                        <TooltipTrigger asChild>{row}</TooltipTrigger>
                        <TooltipContent side="top">
                          <span>{unsupportedMCPVersion ? '版本过低，不支持MCP服务。' : '实例已下发该MCP'}</span>
                        </TooltipContent>
                      </Tooltip>
                    );
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </div>

        {/* 分页控件 — 弹窗场景用 simple 模式（规范 §12.5） */}
        <div className="pt-3">
          {selectAllMode && selectedSelectableCount > 0 && (
            <p className="mb-1 text-xs text-[var(--cp-text-muted)]">已选择全部符合条件的实例</p>
          )}
          <Pagination
            total={totalCount}
            current={safeCurrentPage}
            pageSize={pageSize}
            mode="simple"
            showTotal={() => `共 ${totalCount} 条，第 ${safeCurrentPage} / ${totalPages} 页`}
            className="w-full justify-between"
            onChange={(p) => { setCurrentPage(p); if (!selectAllMode) setSelectedInstances([]); }}
          />
        </div>

        <DialogFooter className="gap-2 sm:gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
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
            <AlertTriangle className="w-5 h-5 text-amber-500" />
            风险提示
          </AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-4">
              <div className="flex items-start gap-2.5 bg-amber-50 border border-amber-100 rounded-lg px-4 py-3">
                <p className="text-sm text-amber-700 leading-relaxed">
                  配置 MCP 会修改 <code className="px-1 py-0.5 bg-amber-100/60 rounded text-xs font-mono">~/.openclaw/openclaw.json</code> 文件中的 <code className="px-1 py-0.5 bg-amber-100/60 rounded text-xs font-mono">mcp.servers</code> 相关配置，修改后需重启 gateway 生效，将会导致实例短暂不可用，可能影响正在运行的任务。
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-600 mb-2">请输入<span className="font-semibold text-gray-900">「确认下发」</span>后开始执行。</p>
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

    </>
  );
}
