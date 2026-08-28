import React, { useState, useEffect, useMemo, useRef } from 'react';
import {
  Search,
  RefreshCw,
  Loader2,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { SurfaceCard } from '@/components/ui/Surface';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from '@/components/ui/dialog';
import { Alert, AlertDescription, AlertInfoIcon } from '@/components/ui/alert';
import {
  PanelTitle,
  BodyText,
  BodyMedium,
  MetaText,
  HelperText,
} from '@/components/ui/Typography';
import { toast } from 'sonner';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';

import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import { cn } from '@/lib/utils';
import { useAdminMode } from '@/contexts/AdminModeContext';
import { TreeSelect, type TreeFilterNode } from '@/components/ui/tree-select';
import { FilterMultiSelect } from '@/components/ui/select';
import {
  GroupCell,
  matchesGroupFilter,
  getGroupsForMode,
} from './GroupColumn';
import { buildGroupTree, type GroupTreeNode } from '../../MemberManagement/health';
// 将 GroupTreeNode 转为 TreeSelect 需要的 TreeFilterNode
function groupTreeToFilterNodes(nodes: GroupTreeNode[]): TreeFilterNode[] {
  return nodes.map((n) => ({
    id: n.id,
    name: n.name,
    children: n.children.length > 0 ? groupTreeToFilterNodes(n.children) : undefined,
  }));
}

// 骨架屏组件
const Skeleton: React.FC<{ className?: string }> = ({ className = '' }) => (
  <div className={`animate-pulse bg-gray-200 rounded ${className}`} />
);

// 骨架屏行组件 - 6列（复选框、名称/ID、创建人、Agent 类型、组织、记忆管理）
const SkeletonRow: React.FC = () => (
  <TableRow>
    <TableCell className="w-12 px-4 py-4"><Skeleton className="w-4 h-4 rounded" /></TableCell>
    <TableCell className="px-6 py-4">
      <div className="flex flex-col gap-1">
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-3 w-32" />
      </div>
    </TableCell>
    <TableCell className="px-6 py-4"><Skeleton className="h-4 w-32" /></TableCell>
    <TableCell className="px-4 py-4"><Skeleton className="h-4 w-20" /></TableCell>
    <TableCell className="px-4 py-4"><Skeleton className="h-4 w-24" /></TableCell>
    <TableCell className="px-6 py-4"><Skeleton className="h-8 w-[200px] rounded-full" /></TableCell>
  </TableRow>
);

// Memory 版本类型
export type MemoryVersion = 'none' | 'free' | 'pro';

// Agent 类型显示名称映射
// 与管控端 Agent 列表（OpenClawMonitor）保持一致：
//   - OpenClaw / openclaw → "OpenClaw"
//   - Hermes / hermes     → "Hermes Agent"
// 兼容大小写两种写法（OcInstance 上的 agentType 历史上用小写，OpenClawMonitor 用大写）。
// 当前仅 OpenClaw / Hermes 支持记忆服务，其它类型不纳入映射，列表数据源也不会下发。
const AGENT_TYPE_DISPLAY: Record<string, string> = {
  openclaw: 'OpenClaw',
  OpenClaw: 'OpenClaw',
  hermes: 'Hermes Agent',
  Hermes: 'Hermes Agent',
};

// Memory 状态类型
// idle: 空闲（未开启）
// enabling: 开启中
// running: 已开启
// closing: 关闭中
// error: 异常
export type MemoryState = 'idle' | 'enabling' | 'running' | 'closing' | 'error';

// 组合状态（用于过滤和显示）
export type MemoryStatus = 'none' | 'free-enabling' | 'free' | 'pro-enabling' | 'pro' | 'closing' | 'error';

// 弹窗类型（去掉 upgrade-to-pro，合并到 enable-pro）
// 增加批量操作弹窗类型
type DialogType = 'none' | 'enable-free' | 'enable-pro' | 'disable' | 'batch-enable-free' | 'batch-enable-pro' | 'batch-disable';

// 三态开关组件（关闭 / Free / Pro）
// 合并了原先的「开启状态」Switch 和「记忆管理」版本切换，统一为一列
interface TriStateSwitchProps {
  value: 'none' | 'free' | 'pro';
  isTransitioning?: boolean;
  isError?: boolean;
  isProDisabled?: boolean;
  proDisabledReason?: string;
  // 锁定态：不可交互。视觉上复用与「开通中」一致的滑块内 loading，
  // 在当前档位（Pro）的 button 内叠加 spinner，但保留档位文字 —— 用户依然
  // 知道当前档位没变，只是有一个异步任务（插件升级）在后台跑。
  isLocked?: boolean;
  onChange: (newValue: 'none' | 'free' | 'pro') => void;
}

const TriStateSwitch: React.FC<TriStateSwitchProps> = ({
  value,
  isTransitioning = false,
  isError = false,
  isProDisabled = false,
  proDisabledReason,
  isLocked = false,
  onChange,
}) => {
  const options: { key: 'none' | 'free' | 'pro'; label: string }[] = [
    { key: 'none', label: '关闭' },
    { key: 'free', label: 'Free版' },
    { key: 'pro', label: 'Pro版' },
  ];

  const handleClick = (key: 'none' | 'free' | 'pro') => {
    if (isTransitioning) return;
    if (isLocked) return;
    if (key === value) return;
    // Pro 禁用时不可切换到 Pro
    if (key === 'pro' && isProDisabled) return;
    // 不支持从 Pro 切换到 Free
    if (value === 'pro' && key === 'free') return;
    onChange(key);
  };

  // 计算滑块位置
  const getSliderPosition = () => {
    switch (value) {
      case 'none': return 'left-0.5';
      case 'free': return 'left-[calc(33.33%+1px)]';
      case 'pro': return 'left-[calc(66.66%+2px)]';
      default: return 'left-0.5';
    }
  };

  // 计算滑块背景色
  const getSliderBg = () => {
    if (isTransitioning) return 'bg-blue-500';
    if (isError) return 'bg-red-500';
    switch (value) {
      case 'none': return 'bg-gray-500';
      case 'free': return 'bg-blue-500';
      case 'pro': return 'bg-purple-500';
      default: return 'bg-gray-500';
    }
  };

  return (
    <div className={`relative inline-flex items-center h-8 bg-gray-100 rounded-full p-0.5 w-[200px] ${isLocked ? 'opacity-60' : ''}`}>
      {/* 滑块 */}
      <div
        className={`absolute h-7 w-[calc(33.33%-2px)] rounded-full transition-all duration-200 ${getSliderPosition()} ${getSliderBg()}`}
      >
        {(isTransitioning || isLocked) && (
          <div className="absolute inset-0 flex items-center justify-center">
            <Loader2 className="w-3.5 h-3.5 text-white animate-spin" />
          </div>
        )}
      </div>

      {/* 选项 */}
      {options.map((opt) => {
        const isActive = value === opt.key;
        const isDisabled = isTransitioning || isLocked ||
          (opt.key === 'pro' && isProDisabled) ||
          (value === 'pro' && opt.key === 'free'); // Pro 不能降级到 Free

        // Pro 选项且被禁用时显示 tooltip
        if (opt.key === 'pro' && isProDisabled && !isTransitioning) {
          return (
            <Tooltip key={opt.key}>
              <TooltipTrigger asChild>
                <button
                  className={`relative z-10 flex-1 h-7 text-xs font-medium rounded-full transition-colors
                    ${isActive ? 'text-white' : 'text-gray-400 cursor-not-allowed'}`}
                >
                  {opt.label}
                </button>
              </TooltipTrigger>
              <TooltipContent side="top" className="text-xs">
                {proDisabledReason || '不可用'}
              </TooltipContent>
            </Tooltip>
          );
        }

        return (
          <button
            key={opt.key}
            onClick={() => handleClick(opt.key)}
            disabled={isDisabled}
            className={`relative z-10 flex-1 h-7 text-xs font-medium rounded-full transition-colors
              ${isActive
                ? ((isTransitioning || isLocked) ? 'text-white/80' : 'text-white')
                : isDisabled
                  ? 'text-gray-400 cursor-not-allowed'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
          >
            {isActive && (isTransitioning || isLocked) ? '' : opt.label}
          </button>
        );
      })}
    </div>
  );
};

export interface OcInstance {
  id: string;
  name: string;
  // 兼容旧的 memoryStatus 字段
  memoryStatus: MemoryStatus;
  // 新增：版本和状态分离
  version: MemoryVersion;
  state: MemoryState;
  memoryId: string;
  enabledAt: string;
  creator?: string;
  errorMessage?: string; // 异常状态时的错误信息
  // Agent 类型：用于在列表中显示对应展示文案（见 AGENT_TYPE_DISPLAY）。
  // 当前仅 OpenClaw / Hermes 支持记忆服务，其它类型不应进入本列表。
  // 未设置时默认视作 openclaw。
  agentType?: 'openclaw' | 'hermes' | string;
  // 记忆插件是否正在异步升级中。与 memoryStatus 解耦：
  // 升级过程不改变 Free / Pro 版本档位，仅在"记忆管理"列叠加一个 loading，
  // 并在升级完成前禁用该实例的一切操作（切换版本、勾选批量等）。
  isPluginUpgrading?: boolean;
}

// 辅助函数：从 memoryStatus 解析出 version 和 state
export function parseMemoryStatus(status: MemoryStatus): { version: MemoryVersion; state: MemoryState } {
  switch (status) {
    case 'none':
      return { version: 'none', state: 'idle' };
    case 'free-enabling':
      return { version: 'free', state: 'enabling' };
    case 'free':
      return { version: 'free', state: 'running' };
    case 'pro-enabling':
      return { version: 'pro', state: 'enabling' };
    case 'pro':
      return { version: 'pro', state: 'running' };
    case 'closing':
      return { version: 'none', state: 'closing' };
    case 'error':
      return { version: 'none', state: 'error' };
    default:
      return { version: 'none', state: 'idle' };
  }
}

interface InstanceTableProps {
  instances: OcInstance[];
  loading?: boolean;
  isProActive?: boolean; // Pro 服务是否已开通
  proSpacesAvailable?: number; // Pro 剩余可用空间数
  onOpenDetail?: (instance: OcInstance) => void;
  onEnableFree?: (instance: OcInstance) => void | Promise<void>;
  // 开启 Pro（若当前是 Free，自动处理数据迁移）
  onEnablePro?: (instance: OcInstance) => void | Promise<void>;
  onDisableMemory?: (instance: OcInstance) => void | Promise<void>;
  // 批量操作回调
  onBatchEnableFree?: (instances: OcInstance[]) => void | Promise<void>;
  onBatchEnablePro?: (instances: OcInstance[]) => void | Promise<void>;
  onBatchDisable?: (instances: OcInstance[]) => void | Promise<void>;
  // 列表工具栏右侧（搜索框左侧）可插入的自定义操作区，用于承载"一键升级记忆插件"等全局入口
  toolbarRight?: React.ReactNode;
}

export const InstanceTable: React.FC<InstanceTableProps> = ({
  instances,
  loading = false,
  isProActive = false,
  proSpacesAvailable = 0,
  onOpenDetail,
  onEnableFree,
  onEnablePro,
  onDisableMemory,
  onBatchEnableFree,
  onBatchEnablePro,
  onBatchDisable,
  toolbarRight,
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [refreshing, setRefreshing] = useState(false);  
  // 批量选择状态
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  // Gmail 风格：是否选择了全部（跨页）
  const [isSelectAll, setIsSelectAll] = useState(false);
  
  // 记忆状态筛选（合并版本和状态）
  const [selectedMemoryStates, setSelectedMemoryStates] = useState<Set<string>>(new Set(['none', 'free', 'pro']));
  
  // 弹窗相关状态
  const [dialogType, setDialogType] = useState<DialogType>('none');
  const [targetInstance, setTargetInstance] = useState<OcInstance | null>(null);
  const [isProcessing, setIsProcessing] = useState(false);

  // 组织列筛选（使用 TreeSelect 组件）
  const { hasOneid } = useAdminMode();
  const [groupFilter, setGroupFilter] = useState('');

  const PAGE_SIZE = 10;

  // 将 Set 转为可序列化字符串用于依赖检测
  const selectedMemoryStatesKey = Array.from(selectedMemoryStates).sort().join(',');
  
  // 过滤和分页
  const filteredList = useMemo(() => {
    return instances.filter((oc) => {
      const matchSearch =
        oc.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        oc.id.includes(searchQuery);
      
      const { version, state } = parseMemoryStatus(oc.memoryStatus);
      
      // 记忆状态筛选（简化为：未开启、Free 版、Pro 版）
      // 过渡态的实例按其目标版本归类
      let matchMemoryState = false;
      if (state === 'idle' || state === 'error') {
        matchMemoryState = selectedMemoryStates.has('none');
      } else if (version === 'free') {
        // Free 版（包括 running 和过渡态）
        matchMemoryState = selectedMemoryStates.has('free');
      } else if (version === 'pro') {
        // Pro 版（包括 running 和过渡态）
        matchMemoryState = selectedMemoryStates.has('pro');
      } else {
        matchMemoryState = selectedMemoryStates.has('none');
      }

      // 组织筛选：creator 是否属于选中组织（含子孙组织）
      const matchGroup = matchesGroupFilter(oc.creator, groupFilter, hasOneid);

      return matchSearch && matchMemoryState && matchGroup;
    });
  }, [instances, searchQuery, selectedMemoryStatesKey, selectedMemoryStates, groupFilter, hasOneid]);

  const totalPages = Math.max(1, Math.ceil(filteredList.length / PAGE_SIZE));
  const paginatedList = filteredList.slice(
    (currentPage - 1) * PAGE_SIZE,
    currentPage * PAGE_SIZE
  );

  const handleRefresh = () => {
    setRefreshing(true);
    setTimeout(() => {
      setRefreshing(false);
      toast.success('列表已刷新');
    }, 1000);
  };

  // 批量选择逻辑
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      // 选中当前页所有可操作的实例（排除过渡态 / 插件升级中）
      const selectableIds = paginatedList
        .filter(oc => {
          const { state } = parseMemoryStatus(oc.memoryStatus);
          return state !== 'enabling' && state !== 'closing' && !oc.isPluginUpgrading;
        })
        .map(oc => oc.id);
      setSelectedIds(new Set(selectableIds));
      setIsSelectAll(false); // 重置全选状态
    } else {
      setSelectedIds(new Set());
      setIsSelectAll(false);
    }
  };

  // Gmail 风格：选择全部（跨页）
  const handleSelectAllPages = () => {
    // 选中所有可操作的实例
    const allSelectableIds = filteredList
      .filter(oc => {
        const { state } = parseMemoryStatus(oc.memoryStatus);
        return state !== 'enabling' && state !== 'closing' && !oc.isPluginUpgrading;
      })
      .map(oc => oc.id);
    setSelectedIds(new Set(allSelectableIds));
    setIsSelectAll(true);
  };

  // Gmail 风格：清除选择
  const handleClearSelection = () => {
    setSelectedIds(new Set());
    setIsSelectAll(false);
  };

  const handleSelectOne = (id: string, checked: boolean) => {
    const newSet = new Set(selectedIds);
    if (checked) {
      newSet.add(id);
    } else {
      newSet.delete(id);
      setIsSelectAll(false); // 取消单个选择时，清除全选状态
    }
    setSelectedIds(newSet);
  };

  // 获取当前选中的实例
  const selectedInstances = instances.filter(i => selectedIds.has(i.id));
  
  // 判断当前页是否全选
  const selectableInPage = paginatedList.filter(oc => {
    const { state } = parseMemoryStatus(oc.memoryStatus);
    return state !== 'enabling' && state !== 'closing' && !oc.isPluginUpgrading;
  });
  const isAllSelected = selectableInPage.length > 0 && selectableInPage.every(oc => selectedIds.has(oc.id));
  const isPartialSelected = selectableInPage.some(oc => selectedIds.has(oc.id)) && !isAllSelected;

  // Gmail 风格：计算全部可选择的实例数（跨页）
  const allSelectableInstances = filteredList.filter(oc => {
    const { state } = parseMemoryStatus(oc.memoryStatus);
    return state !== 'enabling' && state !== 'closing' && !oc.isPluginUpgrading;
  });
  const totalSelectableCount = allSelectableInstances.length;
  
  // 是否显示「选择全部」提示条：当前页全选了，且还有更多可选
  const showSelectAllBanner = isAllSelected && selectedIds.size < totalSelectableCount && totalSelectableCount > selectableInPage.length;

  // 批量操作 - 打开确认弹窗
  const handleBatchEnableFree = () => {
    // 筛选出可以开启 Free 的实例（未开启的）
    const eligibleInstances = selectedInstances.filter(i => i.memoryStatus === 'none');
    if (eligibleInstances.length === 0) {
      toast.warning('请选择未开启记忆的实例');
      return;
    }
    // 打开批量开启 Free 确认弹窗
    setDialogType('batch-enable-free');
  };

  const handleBatchEnablePro = () => {
    // 筛选出可以升级 Pro 的实例（未开启或 Free）
    const eligibleInstances = selectedInstances.filter(i => i.memoryStatus === 'none' || i.memoryStatus === 'free');
    if (eligibleInstances.length === 0) {
      toast.warning('请选择未开启或 Free 版的实例');
      return;
    }
    if (!isProActive) {
      toast.error('请先开通 Memory Pro 服务');
      return;
    }
    if (eligibleInstances.length > proSpacesAvailable) {
      toast.error(`记忆空间不足，当前剩余 ${proSpacesAvailable} 个，需要 ${eligibleInstances.length} 个`);
      return;
    }
    // 打开批量升级 Pro 确认弹窗
    setDialogType('batch-enable-pro');
  };

  const handleBatchDisable = () => {
    // 筛选出可以关闭的实例（已开启 Free 或 Pro）
    const eligibleInstances = selectedInstances.filter(i => i.memoryStatus === 'free' || i.memoryStatus === 'pro');
    if (eligibleInstances.length === 0) {
      toast.warning('请选择已开启记忆的实例');
      return;
    }
    // 打开批量关闭确认弹窗
    setDialogType('batch-disable');
  };

  // 批量升级记忆插件：已迁移到顶部"一键升级"全局入口，此处不再保留行内/工具栏入口。

  // 打开确认弹窗
  const openDialog = (type: DialogType, instance: OcInstance) => {
    setTargetInstance(instance);
    setDialogType(type);
  };

  // 关闭弹窗
  const closeDialog = () => {
    if (isProcessing) return;
    setDialogType('none');
    setTargetInstance(null);
  };

  // 执行状态变更
  const executeStatusChange = async () => {
    setIsProcessing(true);
    try {
      // 批量操作
      if (dialogType === 'batch-enable-free') {
        const eligibleInstances = selectedInstances.filter(i => i.memoryStatus === 'none');
        closeDialog();
        setIsProcessing(false);
        setSelectedIds(new Set());
        onBatchEnableFree?.(eligibleInstances);
        return;
      }
      if (dialogType === 'batch-enable-pro') {
        const eligibleInstances = selectedInstances.filter(i => i.memoryStatus === 'none' || i.memoryStatus === 'free');
        closeDialog();
        setIsProcessing(false);
        setSelectedIds(new Set());
        onBatchEnablePro?.(eligibleInstances);
        return;
      }
      if (dialogType === 'batch-disable') {
        const eligibleInstances = selectedInstances.filter(i => i.memoryStatus === 'free' || i.memoryStatus === 'pro');
        closeDialog();
        setIsProcessing(false);
        setSelectedIds(new Set());
        onBatchDisable?.(eligibleInstances);
        return;
      }

      // 单实例操作
      if (!targetInstance) return;
      
      // 根据弹窗类型调用对应的回调
      switch (dialogType) {
        case 'enable-free':
          // 开启 Free 是异步操作，弹窗立即关闭
          closeDialog();
          setIsProcessing(false);
          onEnableFree?.(targetInstance);
          return;
        case 'enable-pro':
          // 开启 Pro 是异步操作（若当前是 Free，自动迁移数据），弹窗立即关闭
          closeDialog();
          setIsProcessing(false);
          onEnablePro?.(targetInstance);
          return;
        case 'disable':
          await onDisableMemory?.(targetInstance);
          break;
      }
      closeDialog();
    } catch (error) {
      console.error('状态变更失败:', error);
    } finally {
      setIsProcessing(false);
    }
  };

  // 确认弹窗组件
  const ConfirmDialog = () => {
    if (dialogType === 'none') return null;
    
    // 关闭弹窗的确认文字输入
    const [confirmText, setConfirmText] = useState('');
    
    // 批量操作时的实例统计
    const batchStats = {
      enableFree: selectedInstances.filter(i => i.memoryStatus === 'none'),
      enablePro: selectedInstances.filter(i => i.memoryStatus === 'none' || i.memoryStatus === 'free'),
      disable: selectedInstances.filter(i => i.memoryStatus === 'free' || i.memoryStatus === 'pro'),
    };
    const hasProInDisable = selectedInstances.some(i => i.memoryStatus === 'pro');

    type DialogConfig = {
      title: string;
      content: React.ReactNode;
      confirmLabel: string;
      confirmVariant: 'claw-primary' | 'destructive';
      confirmDisabled: boolean;
    };

    const getDialogConfig = (): DialogConfig | null => {
      // 批量开启 Free
      if (dialogType === 'batch-enable-free') {
        const count = batchStats.enableFree.length;
        return {
          title: '批量开启 Memory Free',
          content: (
            <div className="space-y-3">
              <Alert variant="warning">
                <AlertInfoIcon />
                <AlertDescription>开启后将重启相关 Gateway 服务，届时会有短暂的服务中断。</AlertDescription>
              </Alert>
              <BodyText as="p" tone="secondary" className="leading-relaxed">
                即将为 <BodyMedium as="span" tone="primary">{count}</BodyMedium> 个 Agent 开启 Memory Free 服务。
              </BodyText>
              {selectedInstances.length > count && (
                <HelperText>注：已选中 {selectedInstances.length} 个 Agent，其中 {selectedInstances.length - count} 个已开启记忆，将被跳过。</HelperText>
              )}
            </div>
          ),
          confirmLabel: '确认开启',
          confirmVariant: 'claw-primary',
          confirmDisabled: false,
        };
      }

      // 批量升级 Pro
      if (dialogType === 'batch-enable-pro') {
        const count = batchStats.enablePro.length;
        const fromFreeCount = batchStats.enablePro.filter(i => i.memoryStatus === 'free').length;
        return {
          title: '开启 Memory Pro',
          content: (
            <div className="space-y-3">
              <Alert variant="warning">
                <AlertInfoIcon />
                <AlertDescription>
                  <ul className="space-y-1 list-disc pl-4">
                    <li>开启 Pro 版后不支持回退到 Free 版</li>
                    <li>开启后将重启 Gateway 服务，届时会有短暂的服务中断。</li>
                  </ul>
                </AlertDescription>
              </Alert>
              <BodyText as="p" tone="secondary" className="leading-relaxed">
                确认为 <BodyMedium as="span" tone="primary">{count}</BodyMedium> 个 Agent 开启 Memory Pro 服务？{fromFreeCount > 0 && `其中 ${fromFreeCount} 个 Agent 将从 Free 版升级，数据将自动迁移。`}
              </BodyText>
              {selectedInstances.length > count && (
                <HelperText>注：已选中 {selectedInstances.length} 个 Agent，其中 {selectedInstances.length - count} 个已是 Pro 版，将被跳过。</HelperText>
              )}
            </div>
          ),
          confirmLabel: '确认开启',
          confirmVariant: 'claw-primary',
          confirmDisabled: false,
        };
      }

      // 批量关闭
      if (dialogType === 'batch-disable') {
        const count = batchStats.disable.length;
        const proCount = batchStats.disable.filter(i => i.memoryStatus === 'pro').length;
        const freeCount = count - proCount;
        return {
          title: '批量关闭 Memory 服务',
          content: (
            <div className="space-y-3">
              <Alert variant="warning">
                <AlertInfoIcon />
                {hasProInDisable ? (
                  <AlertDescription>
                    <ul className="space-y-1 list-disc pl-4">
                      <li>关闭后将重启相关 Gateway 服务，届时会有短暂的服务中断。</li>
                      <li>{`${proCount > 0 ? `${proCount} 个 Pro 版实例的` : ''}所有记忆数据将被清除，此操作不可恢复。`}</li>
                    </ul>
                  </AlertDescription>
                ) : (
                  <AlertDescription>关闭后将重启相关 Gateway 服务，届时会有短暂的服务中断。</AlertDescription>
                )}
              </Alert>
              <BodyText as="p" tone="secondary" className="leading-relaxed">
                即将关闭 <BodyMedium as="span" tone="primary">{count}</BodyMedium> 个 Agent 的 Memory 服务。
              </BodyText>
              {proCount > 0 && freeCount > 0 && (
                <BodyText as="p" tone="muted">
                  包含 {proCount} 个 Pro 版、{freeCount} 个 Free 版实例。
                </BodyText>
              )}
              {!hasProInDisable && (
                <BodyText as="p" tone="muted" className="leading-relaxed">
                  Free 版实例的记忆数据将保留在本地，重新开启后可继续使用。
                </BodyText>
              )}
              {/* 二次确认输入框 */}
              <div className="pt-2">
                <label className="block mb-2">
                  <BodyText as="span" tone="secondary">
                    请输入「<BodyMedium as="span" tone="danger">关闭</BodyMedium>」以确认：
                  </BodyText>
                </label>
                <Input
                  value={confirmText}
                  onChange={(e) => setConfirmText(e.target.value)}
                  placeholder="请输入「关闭」"
                />
              </div>
              {selectedInstances.length > count && (
                <HelperText>注：已选中 {selectedInstances.length} 个 Agent，其中 {selectedInstances.length - count} 个未开启记忆，将被跳过。</HelperText>
              )}
            </div>
          ),
          confirmLabel: '确认关闭',
          confirmVariant: 'destructive',
          confirmDisabled: confirmText !== '关闭',
        };
      }

      // 单实例操作
      if (!targetInstance) return null;

      const { version } = parseMemoryStatus(targetInstance.memoryStatus);
      const isFromFree = version === 'free';
      const isProVersion = targetInstance.memoryStatus === 'pro';

      switch (dialogType) {
        case 'enable-free':
          return {
            title: '开启 Memory Free',
            content: (
              <div className="space-y-3">
                <Alert variant="warning">
                  <AlertInfoIcon />
                  <AlertDescription>开启后将重启相关 Gateway 服务，届时会有短暂的服务中断。</AlertDescription>
                </Alert>
                <BodyText as="p" tone="secondary" className="leading-relaxed">
                  确认为 Agent「<BodyMedium as="span" tone="primary">{targetInstance.name}</BodyMedium>」开启 Memory Free 服务？
                </BodyText>
              </div>
            ),
            confirmLabel: '确认开启',
            confirmVariant: 'claw-primary',
            confirmDisabled: false,
          };
        case 'enable-pro': {
          return {
            title: '开启 Memory Pro',
            content: (
              <div className="space-y-3">
                <Alert variant="warning">
                  <AlertInfoIcon />
                  <AlertDescription>
                    <ul className="space-y-1 list-disc pl-4">
                      <li>开启 Pro 版后不支持回退到 Free 版</li>
                      <li>开启后将重启 Gateway 服务，届时会有短暂的服务中断。</li>
                    </ul>
                  </AlertDescription>
                </Alert>
                <BodyText as="p" tone="secondary" className="leading-relaxed">
                  确认为 Agent「<BodyMedium as="span" tone="primary">{targetInstance.name}</BodyMedium>」开启 Memory Pro 服务？
                </BodyText>
                {isFromFree && (
                  <BodyText as="p" tone="muted" className="leading-relaxed">
                    Free 版的记忆数据将自动迁移到 Pro 版。
                  </BodyText>
                )}
              </div>
            ),
            confirmLabel: '确认开启',
            confirmVariant: 'claw-primary',
            confirmDisabled: false,
          };
        }
        case 'disable': {
          return {
            title: '关闭 Memory 服务',
            content: (
              <div className="space-y-3">
                <Alert variant="warning">
                  <AlertInfoIcon />
                  {isProVersion ? (
                    <AlertDescription>
                      <ul className="space-y-1 list-disc pl-4">
                        <li>关闭后将重启 Gateway 服务，届时会有短暂的服务中断。</li>
                        <li>所有记忆数据将被清除，此操作不可恢复。</li>
                      </ul>
                    </AlertDescription>
                  ) : (
                    <AlertDescription>关闭后将重启 Gateway 服务，届时会有短暂的服务中断。</AlertDescription>
                  )}
                </Alert>
                <BodyText as="p" tone="secondary" className="leading-relaxed">
                  确认关闭 Agent「<BodyMedium as="span" tone="primary">{targetInstance.name}</BodyMedium>」的 Memory 服务？{!isProVersion && '记忆数据将保留在本地，重新开启后可继续使用。'}
                </BodyText>
                {/* 二次确认输入框 */}
                <div className="pt-2">
                  <label className="block mb-2">
                    <BodyText as="span" tone="secondary">
                      请输入「<BodyMedium as="span" tone="danger">关闭</BodyMedium>」以确认：
                    </BodyText>
                  </label>
                  <Input
                    value={confirmText}
                    onChange={(e) => setConfirmText(e.target.value)}
                    placeholder="请输入「关闭」"
                  />
                </div>
              </div>
            ),
            confirmLabel: '确认关闭',
            confirmVariant: 'destructive',
            confirmDisabled: confirmText !== '关闭',
          };
        }
        default:
          return null;
      }
    };

    const config = getDialogConfig();
    if (!config) return null;

    return (
      <Dialog open onOpenChange={(o) => { if (!o && !isProcessing) closeDialog(); }}>
        <DialogContent className="sm:max-w-[480px] rounded-[8px]">
          <DialogHeader>
            <DialogTitle>{config.title}</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">{config.content}</DialogBody>
          <DialogFooter>
            <Button variant="claw-outline" size="claw-sm" onClick={closeDialog} disabled={isProcessing}>
              取消
            </Button>
            <Button
              variant={config.confirmVariant}
              size="claw-sm"
              onClick={executeStatusChange}
              disabled={isProcessing || config.confirmDisabled}
            >
              {isProcessing && <Loader2 className="w-4 h-4 animate-spin" />}
              {config.confirmLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    );
  };

  // 筛选变化重置页码
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, selectedMemoryStates]);

  // 记忆状态筛选选项配置
  const memoryFilterOptions = [
    { value: 'none', label: '关闭' },
    { value: 'free', label: 'Free版' },
    { value: 'pro', label: 'Pro版' },
  ];

  // 组织树形筛选节点数据
  const groupTreeFilterNodes = useMemo((): TreeFilterNode[] => {
    const groups = getGroupsForMode(hasOneid);
    const trees = buildGroupTree(groups);
    return groupTreeToFilterNodes(trees);
  }, [hasOneid]);

  return (
    <div>
      {/* 工具栏 - 放在表格卡片外部 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <PanelTitle as="h2">记忆空间列表</PanelTitle>
          {/* 搜索框 */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
            <Input
              placeholder="搜索 Agent 名称或 ID"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9 w-64"
              disabled={loading}
            />
          </div>
        </div>
        <div className="flex flex-wrap gap-3 items-center">
          {/* 自定义工具栏右侧：承载一键升级记忆插件等全局入口 */}
          {toolbarRight}
          {/* 批量操作按钮 */}
          {(() => {
            // 统计选中实例的状态分布
            const noneCount = selectedInstances.filter(i => i.memoryStatus === 'none').length;
            const freeCount = selectedInstances.filter(i => i.memoryStatus === 'free').length;
            const proCount = selectedInstances.filter(i => i.memoryStatus === 'pro').length;
            // 是否有选中且可操作的实例
            const canEnableFree = noneCount > 0;
            const canEnablePro = (noneCount > 0 || freeCount > 0);
            const canDisable = (freeCount > 0 || proCount > 0);
            const hasSelection = selectedIds.size > 0;
            
            return (
              <div className="flex items-center gap-2">
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  onClick={handleBatchEnableFree}
                  disabled={!hasSelection || !canEnableFree}
                >
                  批量开通 Free 版{canEnableFree ? `（${noneCount}）` : ''}
                </Button>
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  onClick={handleBatchEnablePro}
                  disabled={!hasSelection || !canEnablePro || !isProActive}
                >
                  批量开通 Pro 版{canEnablePro ? `（${noneCount + freeCount}）` : ''}
                </Button>
                <Button
                  variant="claw-outline"
                  size="claw-sm"
                  onClick={handleBatchDisable}
                  disabled={!hasSelection || !canDisable}
                >
                  批量关闭{canDisable ? `（${freeCount + proCount}）` : ''}
                </Button>
              </div>
            );
          })()}

          {/* 刷新按钮 */}
          <Button
            variant="claw-outline"
            size="icon-sm"
            onClick={handleRefresh}
            disabled={refreshing || loading}
            title="刷新列表"
            aria-label="刷新列表"
          >
            <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
          </Button>
        </div>
      </div>

    <SurfaceCard className="relative overflow-hidden">
      {/* 加载遮罩 */}
      {loading && (
        <div className="absolute inset-0 bg-[var(--background)]/60 backdrop-blur-[1px] z-10 flex items-center justify-center">
          <div className="flex items-center gap-2">
            <Loader2 className="w-4 h-4 animate-spin text-[var(--text-brand)]" />
            <BodyText as="span" tone="secondary">正在加载实例状态...</BodyText>
          </div>
        </div>
      )}

      {/* 表格 - 白色表头列表 */}
      <Table variant="white" density="compact" containerClassName="overflow-x-auto overflow-y-visible">
          <TableHeader>
          <TableRow>
            {/* 全选复选框 */}
            <TableHead className="w-12 px-4 py-3">
              <Checkbox
                checked={isAllSelected}
                onCheckedChange={handleSelectAll}
                className={isPartialSelected ? 'data-[state=checked]:bg-[var(--text-brand)]' : ''}
                  ref={(el) => {
                    if (el) {
                      // 设置 indeterminate 状态
                      const input = el.querySelector('button');
                      if (input) {
                        (input as any).dataset.state = isPartialSelected ? 'indeterminate' : (isAllSelected ? 'checked' : 'unchecked');
                      }
                    }
                  }}
                />
              </TableHead>
              <TableHead className="text-left px-6 py-3 text-xs font-medium text-[var(--text-muted)] tracking-wide" style={{ width: '30%' }}>
                <span>Agent 名称/ID</span>
              </TableHead>
              <TableHead className="text-left px-6 py-3 text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide" style={{ width: '20%' }}>
                创建人
              </TableHead>
              {/* Agent 类型 —— 与管控端 Agent 列表保持一致：纯灰色文本，不使用 Badge / 颜色，避免视觉权重抢戏 */}
              <TableHead className="text-left px-4 py-3 text-xs font-medium text-[var(--text-muted)] normal-case" style={{ width: '16%' }}>
                Agent 类型
              </TableHead>
              {/* 组织 - 使用 TreeSelect 组件 */}
              <TableHead className="text-left px-4 py-3 text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide whitespace-nowrap" style={{ width: '12%', minWidth: '120px' }}>
                <TreeSelect
                  triggerVariant="filter-icon"
                  title="组织"
                  nodes={groupTreeFilterNodes}
                  value={groupFilter}
                  onChange={(v) => { setGroupFilter(v); setCurrentPage(1); }}
                  allLabel="全部组织"
                  searchPlaceholder="搜索组织"
                />
              </TableHead>

              {/* 记忆管理 - 使用 FilterMultiSelect 组件 */}
              <TableHead className="text-left px-6 py-3 text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide" style={{ width: '14%' }}>
                <FilterMultiSelect
                  title="记忆管理"
                  options={memoryFilterOptions}
                  selectedValues={selectedMemoryStates}
                  onConfirm={(values) => { setSelectedMemoryStates(values); setCurrentPage(1); }}
                />
              </TableHead>
            </TableRow>
          </TableHeader>
          {/* Gmail 风格：选择全部提示条 */}
          {(showSelectAllBanner || isSelectAll) && (
            <TableBody>
              <TableRow>
                <TableCell colSpan={6} className="px-0 py-0">
                  <Alert variant="info" className="rounded-none border-x-0 border-t-0 justify-center">
                    <AlertDescription className="text-center">
                      {isSelectAll ? (
                        <BodyText as="span" tone="brand">
                          已选择全部 <strong>{selectedIds.size}</strong> 个 Agent。
                          <Button
                            variant="link"
                            size="sm"
                            onClick={handleClearSelection}
                            className="ml-2 h-auto p-0 underline underline-offset-2"
                          >
                            清除选择
                          </Button>
                        </BodyText>
                      ) : (
                        <BodyText as="span" tone="brand">
                          已选择此页 <strong>{selectableInPage.length}</strong> 个实例。
                          <Button
                            variant="link"
                            size="sm"
                            onClick={handleSelectAllPages}
                            className="ml-2 h-auto p-0 underline underline-offset-2"
                          >
                            选择全部 {totalSelectableCount} 个 Agent
                          </Button>
                        </BodyText>
                      )}
                    </AlertDescription>
                  </Alert>
                </TableCell>
              </TableRow>
            </TableBody>
          )}
          <TableBody>
            {loading ? (
              <>
                <SkeletonRow />
                <SkeletonRow />
                <SkeletonRow />
                <SkeletonRow />
                <SkeletonRow />
              </>
            ) : paginatedList.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="px-6 py-12 text-center">
                  <BodyText as="span" tone="weak">暂无符合条件的实例</BodyText>
                </TableCell>
              </TableRow>
            ) : (
              paginatedList.map((oc) => {
                const { version, state } = parseMemoryStatus(oc.memoryStatus);
                
                // 判断是否处于过渡态（开启中/关闭中）
                const isMemoryTransitioning = state === 'enabling' || state === 'closing';
                // 插件升级中（异步任务）——不改变记忆版本，但需要禁用所有行内操作
                const isPluginUpgrading = !!oc.isPluginUpgrading;
                // 对外统一视为"过渡态"，复用现有 UI 禁用逻辑
                const isTransitioning = isMemoryTransitioning || isPluginUpgrading;
                // 判断是否异常
                const isError = state === 'error';

                // 计算三态 Switch 当前显示的值
                const getSwitchValue = (): 'none' | 'free' | 'pro' => {
                  if (state === 'closing') return 'none'; // 关闭中显示在未开启位置
                  if (version === 'free' && (state === 'running' || state === 'enabling')) return 'free';
                  if (version === 'pro' && (state === 'running' || state === 'enabling')) return 'pro';
                  return 'none';
                };
                
                // Pro 按钮禁用原因
                const getProDisabledReason = () => {
                  if (version === 'pro' && state === 'running') return null; // Pro 已开启
                  if (!isProActive) return '请先开通 Memory Pro 服务';
                  if (proSpacesAvailable <= 0) return '记忆空间不足';
                  return null;
                };
                const proDisabledReason = getProDisabledReason();
                const isProDisabled = !!proDisabledReason;
                
                // 处理三态 Switch 切换
                const handleSwitchChange = (newValue: 'none' | 'free' | 'pro') => {
                  const currentValue = getSwitchValue();
                  if (newValue === currentValue) return;
                  
                  if (newValue === 'none') {
                    // 切换到关闭
                    openDialog('disable', oc);
                  } else if (newValue === 'free') {
                    // 切换到 Free
                    openDialog('enable-free', oc);
                  } else if (newValue === 'pro') {
                    // 切换到 Pro（可能是从 none 或 free）
                    openDialog('enable-pro', oc);
                  }
                };

                return (
                  <TableRow
                    key={oc.id}
                    className="hover:bg-gray-50/50 transition-colors cursor-pointer"
                    onClick={(e) => {
                      // 点击行内交互元素（按钮 / 链接 / 输入框 / 开关 / 复选框等）时跳过
                      if ((e.target as HTMLElement).closest('button, a, input, label, [role="checkbox"], [role="switch"], [data-no-row-select]')) {
                        return;
                      }
                      if (isTransitioning) return;
                      handleSelectOne(oc.id, !selectedIds.has(oc.id));
                    }}
                  >
                    {/* 复选框 */}
                    <TableCell className="w-12 px-4 py-4">
                      <Checkbox
                        checked={selectedIds.has(oc.id)}
                        onCheckedChange={(checked) => handleSelectOne(oc.id, !!checked)}
                        disabled={isTransitioning}
                      />
                    </TableCell>
                    {/* 名称/ID */}
                    <TableCell className="px-6 py-4" style={{ width: '220px', minWidth: '220px', maxWidth: '220px' }}>
                      <div className="min-w-0">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <div className="text-sm font-medium text-[var(--text-emphasis)] truncate max-w-[180px]">{oc.name}</div>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="text-xs max-w-xs break-all">{oc.name}</TooltipContent>
                        </Tooltip>
                        {onOpenDetail ? (
                          <button
                            onClick={() => onOpenDetail(oc)}
                            className="text-xs font-mono cursor-pointer text-[var(--text-brand)] hover:text-[var(--text-brand-strong,#0F3BBE)] hover:underline"
                          >
                            {oc.id}
                          </button>
                        ) : (
                          <span className="text-xs font-mono text-[var(--text-brand)]">{oc.id}</span>
                        )}
                      </div>
                    </TableCell>
                    {/* 创建人 */}
                    <TableCell className="px-6 py-4 text-sm text-[var(--text-muted)]">
                      {oc.creator || '—'}
                    </TableCell>
                    {/* Agent 类型 —— 复用 AGENT_TYPE_DISPLAY 映射，未配置或未知值时回退展示原值 */}
                    <TableCell className="px-4 py-4">
                      <span className="text-xs font-medium text-[var(--text-muted)]">
                        {AGENT_TYPE_DISPLAY[oc.agentType ?? 'openclaw'] ?? (oc.agentType ?? 'OpenClaw')}
                      </span>
                    </TableCell>
                    {/* 组织 - 复用 Agent 列表同款单元格组件 */}
                    <TableCell className="px-4 py-4">
                      <GroupCell creator={oc.creator} hasOneid={hasOneid} />
                    </TableCell>
                    {/* 记忆管理 - 三态 Switch（关闭/Free/Pro）
                        插件升级中（isLocked）：与「开通中」视觉一致，spinner 直接叠在当前档位；
                        档位文字保留，提示用户当前版本未变、仅是后台异步任务在跑。 */}
                    <TableCell className="px-6 py-4">
                      <TriStateSwitch
                        value={getSwitchValue()}
                        isTransitioning={isMemoryTransitioning}
                        isError={isError}
                        isProDisabled={isProDisabled}
                        proDisabledReason={proDisabledReason || undefined}
                        isLocked={isPluginUpgrading}
                        onChange={handleSwitchChange}
                      />
                    </TableCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>

      {/* 底部：翻页 - 与 Agent 列表保持一致 */}
      {!loading && (
        // 停服态豁免（页面级，不动组件库）：
        //   分页器外层容器打 data-billing-exempt 标记，命中
        //   AdminDisabledOverlay 的两条恢复分支：
        //     1) CSS：.admin-service-suspended [data-billing-exempt] *
        //        恢复 opacity / cursor / pointer-events 到正常态；
        //     2) 事件：文档级click / mousedown 捕获通过
        //        target.closest('[data-billing-exempt]') 命中后放行。
        //   "停服前已禁用则延续禁用"由内部两个 Button 自身的
        //   disabled={currentPage === 1} / disabled={currentPage === totalPages}
        //   保证 —— CSS 恢复分支带 :not([disabled]) 保护，
        //   首页时上一页 / 末页时下一页仍保持灰化禁用，符合原有交互。
        <div className="px-6 py-3 border-t border-[#EAEEF4] flex items-center justify-between" data-billing-exempt>
          <MetaText as="span" tone="weak">
            共 {filteredList.length} 条记录
            {filteredList.length > 0 && `，第 ${currentPage} / ${totalPages} 页`}
          </MetaText>
          <div className="flex items-center gap-2">
            <Button
              variant="claw-outline"
              size="icon-sm"
              onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
              disabled={currentPage === 1}
              aria-label="上一页"
            >
              <ChevronLeft className="w-4 h-4" />
            </Button>
            <MetaText as="span" tone="weak" className="px-2">第 {currentPage} 页</MetaText>
            <Button
              variant="claw-outline"
              size="icon-sm"
              onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
              disabled={currentPage === totalPages}
              aria-label="下一页"
            >
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}

      {/* 确认弹窗 */}
      <ConfirmDialog />
    </SurfaceCard>
    </div>
  );
};
