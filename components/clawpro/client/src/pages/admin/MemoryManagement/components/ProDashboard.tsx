import React, { useState, useEffect, useMemo } from 'react';
import { 
  Search, 
  Gem,
  AlertCircle,
  RefreshCw,
  Loader2,
  CheckCircle2,
  X,
  RotateCcw,
  TrendingUp,
  ChevronLeft,
  ChevronRight,
  Filter,
  Brain,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { toast } from 'sonner';
import { ProCloseDialog } from './ProCloseDialog';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// 骨架屏组件
const Skeleton: React.FC<{ className?: string }> = ({ className = '' }) => (
  <div className={`animate-pulse bg-gray-200 rounded ${className}`} />
);

// 骨架屏行组件（用于表格）
const SkeletonRow: React.FC = () => (
  <tr>
    <td className="px-6 py-4">
      <div className="flex items-center gap-3">
        <Skeleton className="w-8 h-8 rounded-lg" />
        <div className="flex flex-col gap-1">
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-3 w-16" />
        </div>
      </div>
    </td>
    <td className="px-6 py-4">
      <Skeleton className="h-6 w-24 rounded-full" />
    </td>
    <td className="px-6 py-4">
      <Skeleton className="h-4 w-20" />
    </td>
    <td className="px-6 py-4">
      <Skeleton className="h-4 w-28" />
    </td>
  </tr>
);

// 服务状态类型
type ServiceStatus = 'inactive' | 'activating' | 'active' | 'error';

// Memory 状态类型：Pro版已开启 | Free版已开启 | 未开启
type MemoryStatus = 'pro' | 'free' | 'none';

// 状态筛选选项
type StatusFilter = 'all' | 'pro' | 'free' | 'none';

// 状态配置
const memoryStatusConfig = {
  pro: { 
    label: 'Pro版已开启', 
    color: 'bg-green-500', 
    bgColor: 'bg-green-50', 
    textColor: 'text-green-700',
    icon: '🟢'
  },
  free: { 
    label: 'Free版已开启', 
    color: 'bg-amber-500', 
    bgColor: 'bg-amber-50', 
    textColor: 'text-amber-700',
    icon: '🟡'
  },
  none: { 
    label: '未开启', 
    color: 'bg-gray-400', 
    bgColor: 'bg-gray-50', 
    textColor: 'text-gray-600',
    icon: '⚪'
  },
};

// Mock 数据 - 18 个 Agent（全部开通 Pro，模拟 90% 配额使用率）
const mockOcList: Array<{ 
  id: string; 
  name: string; 
  memoryStatus: MemoryStatus; 
  memoryId: string; 
  updatedAt: string;
}> = [
  { id: 'oc-001', name: '小助手', memoryStatus: 'pro', memoryId: 'mem-001', updatedAt: '2026-04-07 10:30' },
  { id: 'oc-002', name: '客服Bot', memoryStatus: 'pro', memoryId: 'mem-002', updatedAt: '2026-04-06 15:20' },
  { id: 'oc-003', name: '导购员', memoryStatus: 'pro', memoryId: 'mem-003', updatedAt: '2026-04-05 09:15' },
  { id: 'oc-004', name: '代码审查', memoryStatus: 'pro', memoryId: 'mem-004', updatedAt: '2026-04-04 14:00' },
  { id: 'oc-005', name: 'HR助手', memoryStatus: 'pro', memoryId: 'mem-005', updatedAt: '2026-04-04 11:30' },
  { id: 'oc-006', name: '文档助手', memoryStatus: 'pro', memoryId: 'mem-006', updatedAt: '2026-04-03 11:45' },
  { id: 'oc-007', name: '销售Bot', memoryStatus: 'pro', memoryId: 'mem-007', updatedAt: '2026-04-02 16:30' },
  { id: 'oc-008', name: '培训助理', memoryStatus: 'pro', memoryId: 'mem-008', updatedAt: '2026-04-02 09:00' },
  { id: 'oc-009', name: '运维Bot', memoryStatus: 'pro', memoryId: 'mem-009', updatedAt: '2026-04-01 13:20' },
  { id: 'oc-010', name: '法务助手', memoryStatus: 'pro', memoryId: 'mem-010', updatedAt: '2026-03-31 10:00' },
  { id: 'oc-011', name: '数据分析', memoryStatus: 'pro', memoryId: 'mem-011', updatedAt: '2026-03-30 16:20' },
  { id: 'oc-012', name: '翻译Bot', memoryStatus: 'pro', memoryId: 'mem-012', updatedAt: '2026-03-30 17:45' },
  { id: 'oc-013', name: '财务助手', memoryStatus: 'pro', memoryId: 'mem-013', updatedAt: '2026-03-29 09:30' },
  { id: 'oc-014', name: '招聘Bot', memoryStatus: 'pro', memoryId: 'mem-014', updatedAt: '2026-03-28 14:15' },
  { id: 'oc-015', name: '采购助手', memoryStatus: 'pro', memoryId: 'mem-015', updatedAt: '2026-03-27 11:00' },
  { id: 'oc-016', name: '库存管理', memoryStatus: 'pro', memoryId: 'mem-016', updatedAt: '2026-03-26 08:45' },
  { id: 'oc-017', name: '物流追踪', memoryStatus: 'pro', memoryId: 'mem-017', updatedAt: '2026-03-25 16:00' },
  { id: 'oc-018', name: '质检助手', memoryStatus: 'pro', memoryId: 'mem-018', updatedAt: '2026-03-24 13:30' },
];

interface ProDashboardProps {
  onClose?: () => void;
  serviceStatus?: ServiceStatus;
  errorMessage?: string;
  onRetry?: () => void;
  onStatusBannerDismiss?: () => void;
  /** 用户购买的记忆空间数量 */
  purchasedSpaces?: number;
  /** 扩容成功回调 */
  onExpand?: (addSpaces: number, newTotalSpaces: number) => void;
  /** 点击"前往实例列表"按钮时的回调 */
  onGoToInstanceList?: () => void;
}

export const ProDashboard: React.FC<ProDashboardProps> = ({ 
  onClose, 
  serviceStatus = 'active',
  errorMessage = '',
  onRetry,
  onStatusBannerDismiss,
  purchasedSpaces = 20, // 默认 20，实际从父组件传入
  onExpand,
  onGoToInstanceList
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [currentPage, setCurrentPage] = useState(1);
  const [closeDialogOpen, setCloseDialogOpen] = useState(false);
  const [expandDialogOpen, setExpandDialogOpen] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  
  // 状态提示条显示控制
  const [showSuccessBanner, setShowSuccessBanner] = useState(false);
  
  // 是否处于初始化中（开通中或异常状态）
  const isInitializing = serviceStatus === 'activating';
  const isError = serviceStatus === 'error';
  const isReady = serviceStatus === 'active';
  
  // 初始化完成后显示成功提示条，3秒后自动消失
  useEffect(() => {
    if (isReady) {
      setShowSuccessBanner(true);
      const timer = setTimeout(() => {
        setShowSuccessBanner(false);
        onStatusBannerDismiss?.();
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [isReady, onStatusBannerDismiss]);
  
  const PAGE_SIZE = 10;
  
  // 状态统计
  const statusStats = useMemo(() => {
    const stats = { pro: 0, free: 0, none: 0 };
    mockOcList.forEach(oc => {
      stats[oc.memoryStatus]++;
    });
    return stats;
  }, []);
  
  // Pro 已使用数量（用于额度计算）
  const proUsedCount = statusStats.pro;
  
  // Memory 分配情况计算 - 使用用户购买的记忆空间数量
  const memoryAllocationPercent = Math.round((proUsedCount / purchasedSpaces) * 100);
  
  // 过滤和分页
  const filteredList = mockOcList.filter(oc => {
    // 搜索过滤
    const matchSearch = oc.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
                        oc.id.includes(searchQuery);
    // 状态过滤
    const matchStatus = statusFilter === 'all' || oc.memoryStatus === statusFilter;
    return matchSearch && matchStatus;
  });
  
  const totalPages = Math.max(1, Math.ceil(filteredList.length / PAGE_SIZE));
  const paginatedList = filteredList.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE);

  const formatNumber = (num: number) => num.toLocaleString('zh-CN');

  const handleRefresh = () => {
    setRefreshing(true);
    setTimeout(() => {
      setRefreshing(false);
      toast.success("列表已刷新");
    }, 1000);
  };
  
  // 当筛选条件变化时，重置到第一页
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, statusFilter]);

  return (
    <TooltipProvider>
      <div className="space-y-6">
        {/* 状态提示条 */}
        {isInitializing && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg px-4 py-3 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />
              <span className="text-sm text-blue-700">
                Memory Pro 正在初始化中，预计需要几分钟...
              </span>
            </div>
          </div>
        )}
        
        {isError && (
          <div className="bg-red-50 border border-red-200 rounded-lg px-4 py-3 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <AlertCircle className="w-5 h-5 text-red-500" />
              <span className="text-sm text-red-700">
                {errorMessage || 'Memory Pro 初始化失败，请重试'}
              </span>
            </div>
            <Button 
              variant="outline" 
              size="sm" 
              className="text-red-600 border-red-300 hover:bg-red-100"
              onClick={onRetry}
            >
              <RotateCcw className="w-4 h-4 mr-1" />
              重试
            </Button>
          </div>
        )}
        
        {showSuccessBanner && isReady && (
          <div className="bg-green-50 border border-green-200 rounded-lg px-4 py-3 flex items-center justify-between animate-in fade-in duration-300">
            <div className="flex items-center gap-3">
              <CheckCircle2 className="w-5 h-5 text-green-500" />
              <span className="text-sm text-green-700">
                Memory Pro 已就绪
              </span>
            </div>
            <button 
              onClick={() => setShowSuccessBanner(false)}
              className="text-green-500 hover:text-green-700"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* 页面头部 */}
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-3 mb-1">
              <h1 className="text-2xl font-bold text-gray-900">记忆管理</h1>
              <span className="inline-flex items-center gap-1.5 px-3 py-1 bg-gradient-to-r from-blue-600 to-blue-500 text-white rounded-full text-xs font-semibold">
                <Brain className="w-3.5 h-3.5" />
                MEMORY PRO
              </span>
              {isInitializing && (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-100 text-blue-600 rounded text-xs">
                  <Loader2 className="w-3 h-3 animate-spin" />
                  初始化中
                </span>
              )}
            </div>
            <p className="text-sm text-gray-500">
              基于腾讯云向量数据库的企业级记忆服务，统一管理所有 Agent 的记忆资源。由腾讯云数据库 Agent Memory 服务提供支持。
            </p>
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button 
                  variant="outline" 
                  className={`text-red-600 border-red-200 hover:bg-red-50 ${isInitializing ? 'opacity-50 cursor-not-allowed' : ''}`}
                  onClick={() => !isInitializing && setCloseDialogOpen(true)}
                  disabled={isInitializing}
                >
                  关闭服务
                </Button>
              </span>
            </TooltipTrigger>
            {isInitializing && (
              <TooltipContent>
                <p>服务初始化中，暂不可操作</p>
              </TooltipContent>
            )}
          </Tooltip>
        </div>

      {/* 资源卡片 */}
      <div className="grid grid-cols-1 gap-5">
        {/* Memory 分配情况 */}
        <div className="bg-white rounded-xl border border-gray-100 p-6 relative" style={{ boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
          {/* 初始化遮罩 */}
          {isInitializing && (
            <div className="absolute inset-0 bg-white/60 backdrop-blur-[1px] rounded-xl z-10 flex items-center justify-center">
              <div className="flex items-center gap-2 text-sm text-gray-500">
                <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                <span>数据加载中...</span>
              </div>
            </div>
          )}
          
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <span className="text-xl">📦</span>
              <span className="font-semibold text-gray-900">Memory 分配情况</span>
            </div>
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button 
                    variant="outline" 
                    size="sm" 
                    className={isInitializing ? 'opacity-50 cursor-not-allowed' : ''}
                    onClick={() => !isInitializing && setExpandDialogOpen(true)}
                    disabled={isInitializing}
                  >
                    <TrendingUp className="w-3.5 h-3.5 mr-1" />
                    扩容
                  </Button>
                </span>
              </TooltipTrigger>
              {isInitializing && (
                <TooltipContent>
                  <p>服务初始化中，暂不可操作</p>
                </TooltipContent>
              )}
            </Tooltip>
          </div>
          
          {/* 初始化时显示骨架屏，否则显示真实数据 */}
          {isInitializing ? (
            <>
              <Skeleton className="h-9 w-32 mb-2" />
              <Skeleton className="h-4 w-48 mb-4" />
              <div className="mb-3">
                <div className="flex justify-between mb-1.5">
                  <Skeleton className="h-4 w-16" />
                  <Skeleton className="h-4 w-10" />
                </div>
                <Skeleton className="h-2 w-full rounded-full" />
              </div>
              <div className="pt-3 border-t border-gray-100">
                <Skeleton className="h-3 w-56" />
              </div>
            </>
          ) : (
            <>
              <div className="text-3xl font-bold text-blue-600 mb-1">
                {formatNumber(proUsedCount)}/{formatNumber(purchasedSpaces)}
              </div>
              <div className="text-sm text-gray-500 mb-4">
                已分配 <strong className="text-gray-700">{formatNumber(proUsedCount)}</strong> 个 Pro 记忆空间，剩余 <strong className="text-gray-700">{formatNumber(purchasedSpaces - proUsedCount)}</strong> 个可分配
              </div>
              
              <div className="mb-3">
                <div className="flex justify-between text-sm mb-1.5">
                  <span className="text-gray-500">Pro 额度使用率</span>
                  <span className="font-semibold text-blue-600">{memoryAllocationPercent}%</span>
                </div>
                <div className="h-2 bg-gray-100 rounded-full overflow-hidden">
                  <div 
                    className={`h-full rounded-full transition-all ${
                      memoryAllocationPercent >= 100 ? 'bg-red-500' : 
                      memoryAllocationPercent >= 80 ? 'bg-amber-500' : 
                      'bg-gradient-to-r from-blue-500 to-blue-400'
                    }`}
                    style={{ width: `${Math.min(memoryAllocationPercent, 100)}%` }}
                  />
                </div>
              </div>
              
              {/* 容量告警提示 */}
              {memoryAllocationPercent >= 80 && (
                <div className={`mb-3 px-3 py-2 rounded-lg text-xs flex items-center gap-2 ${
                  memoryAllocationPercent >= 100 
                    ? 'bg-red-50 border border-red-100 text-red-700'
                    : 'bg-amber-50 border border-amber-100 text-amber-700'
                }`}>
                  <AlertCircle className="w-4 h-4 flex-shrink-0" />
                  {memoryAllocationPercent >= 100 
                    ? 'Pro 额度已用完，用户将无法新开启 Memory Pro 功能。'
                    : 'Pro 额度即将用完，建议及时扩容。'
                  }
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* OC 列表 - 只读查看 */}
      <div className="bg-white rounded-xl border border-gray-100 relative" style={{ boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
        {/* 初始化遮罩 */}
        {isInitializing && (
          <div className="absolute inset-0 bg-white/60 backdrop-blur-[1px] rounded-xl z-10 flex items-center justify-center">
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
              <span>正在加载实例状态...</span>
            </div>
          </div>
        )}
        
        {/* 工具栏 */}
        <div className="p-5 border-b border-gray-100">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-blue-600 to-blue-500 flex items-center justify-center">
                <Gem className="w-5 h-5 text-white" />
              </div>
              <div>
                <span className="font-semibold text-gray-900">实例记忆状态</span>
                <span className="ml-2 text-xs text-gray-400">（只读）</span>
              </div>
            </div>
            <div className="flex items-center gap-3">
              {/* 状态筛选 */}
              <Select 
                value={statusFilter} 
                onValueChange={(value) => setStatusFilter(value as StatusFilter)}
                disabled={isInitializing}
              >
                <SelectTrigger className="w-[140px]">
                  <Filter className="w-4 h-4 mr-2 text-gray-400" />
                  <SelectValue placeholder="筛选版本" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部版本</SelectItem>
                  <SelectItem value="pro">
                    <span className="flex items-center gap-2">
                      <span>🟢</span> Pro版已开启
                    </span>
                  </SelectItem>
                  <SelectItem value="free">
                    <span className="flex items-center gap-2">
                      <span>🟡</span> Free版已开启
                    </span>
                  </SelectItem>
                  <SelectItem value="none">
                    <span className="flex items-center gap-2">
                      <span>⚪</span> 未开启
                    </span>
                  </SelectItem>
                </SelectContent>
              </Select>
              
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
                <Input
                  placeholder="搜索 Agent 名称或 ID"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9 w-64"
                  disabled={isInitializing}
                />
              </div>
              <button
                onClick={handleRefresh}
                disabled={refreshing || isInitializing}
                className="w-9 h-9 flex items-center justify-center rounded-lg border border-gray-200 bg-white text-gray-400 hover:text-blue-500 hover:border-blue-300 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                title="刷新列表"
              >
                <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>
        </div>
        
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-100 bg-gray-50/50">
                <th className="text-left px-6 py-4 text-xs font-medium text-gray-500 tracking-wide">实例名称/ID</th>
                <th className="text-left px-6 py-4 text-xs font-medium text-gray-500 uppercase tracking-wide">Memory 状态</th>
                <th className="text-left px-6 py-4 text-xs font-medium text-gray-500 uppercase tracking-wide">记忆空间 ID</th>
                <th className="text-left px-6 py-4 text-xs font-medium text-gray-500 uppercase tracking-wide">更新时间</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {/* 初始化时显示骨架屏行 */}
              {isInitializing ? (
                <>
                  <SkeletonRow />
                  <SkeletonRow />
                  <SkeletonRow />
                  <SkeletonRow />
                  <SkeletonRow />
                </>
              ) : paginatedList.length === 0 ? (
                <tr>
                  <td colSpan={4} className="px-6 py-12 text-center text-sm text-gray-400">
                    暂无符合条件的实例
                  </td>
                </tr>
              ) : (
                paginatedList.map((oc, index) => {
                  // 使用饱和度较低的柔和颜色
                  const avatarColors = ['#6B7FD7', '#8B7FC7', '#5B8EC7', '#5BA8B0', '#5B9B7A', '#C9A05B', '#C76B6B', '#B76B9B'];
                  const avatarColor = avatarColors[index % avatarColors.length];
                  const statusConfig = memoryStatusConfig[oc.memoryStatus];
                  
                  return (
                    <tr key={oc.id} className="hover:bg-gray-50/50 transition-colors">
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-3">
                          <div 
                            className="w-8 h-8 rounded-lg flex items-center justify-center text-white text-sm font-semibold"
                            style={{ backgroundColor: avatarColor }}
                          >
                            {oc.name.charAt(0)}
                          </div>
                          <div className="flex flex-col">
                            <span className="text-sm font-medium text-gray-900">{oc.name}</span>
                            <span className="font-mono text-xs text-gray-400">{oc.id}</span>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${statusConfig.bgColor} ${statusConfig.textColor}`}>
                          <span>{statusConfig.icon}</span>
                          {statusConfig.label}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <span className="font-mono text-sm text-gray-500">{oc.memoryId}</span>
                      </td>
                      <td className="px-6 py-4">
                        <span className="text-sm text-gray-500">{oc.updatedAt}</span>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
        
        {/* 底部：状态统计 + 翻页 */}
        <div className="px-6 py-3 border-t border-gray-50 flex items-center justify-between">
          {isInitializing ? (
            <>
              <Skeleton className="h-4 w-48" />
              <div className="flex gap-1">
                <Skeleton className="w-7 h-7 rounded-md" />
                <Skeleton className="w-7 h-7 rounded-md" />
              </div>
            </>
          ) : (
            <>
              <div className="flex items-center gap-4 text-xs text-gray-500">
                <span>状态统计：</span>
                <span className="flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-green-500"></span>
                  Pro版已开启 {statusStats.pro} 个
                </span>
                <span className="text-gray-300">|</span>
                <span className="flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-amber-500"></span>
                  Free版已开启 {statusStats.free} 个
                </span>
                <span className="text-gray-300">|</span>
                <span className="flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-gray-400"></span>
                  未开启 {statusStats.none} 个
                </span>
              </div>
              {totalPages > 1 && (
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 w-7 p-0 text-gray-500"
                    disabled={currentPage === 1}
                    onClick={() => setCurrentPage(currentPage - 1)}
                  >
                    <ChevronLeft className="w-4 h-4" />
                  </Button>
                  {Array.from({ length: totalPages }, (_, i) => i + 1).map(page => (
                    <Button
                      key={page}
                      variant="ghost"
                      size="sm"
                      className={`h-7 w-7 p-0 text-xs ${
                        page === currentPage
                          ? 'bg-blue-50 text-blue-600 font-semibold'
                          : 'text-gray-500 hover:text-gray-700'
                      }`}
                      onClick={() => setCurrentPage(page)}
                    >
                      {page}
                    </Button>
                  ))}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 w-7 p-0 text-gray-500"
                    disabled={currentPage === totalPages}
                    onClick={() => setCurrentPage(currentPage + 1)}
                  >
                    <ChevronRight className="w-4 h-4" />
                  </Button>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Pro 版关闭弹窗 */}
      <ProCloseDialog 
        open={closeDialogOpen} 
        onOpenChange={setCloseDialogOpen}
        onConfirm={onClose}
        ocCount={proUsedCount}
        onGoToInstanceList={onGoToInstanceList}
      />

      {/* 扩容弹窗（ExpandDialog 已移除，此处保留占位） */}
      </div>
    </TooltipProvider>
  );
};
