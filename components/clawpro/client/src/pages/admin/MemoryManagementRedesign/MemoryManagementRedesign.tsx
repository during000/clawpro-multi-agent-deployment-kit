/**
 * 记忆管理 · 重设计预览（独立页面，不影响原版 MemoryManagement）
 * --------------------------------------------------------------
 * 设计要点（与原版差异）：
 *   1. 服务级 vs 实例级分两层：顶部一行「服务状态条」承载开通/关闭，
 *      取代原版中和 4 张统计卡并列的「立即开通」入口。
 *   2. 配额信息单卡聚焦：仅在 Pro 已开通时显示「记忆配额」卡，去掉
 *      实例总数 / 未开启 / Free 这种摘要型统计，避免淹没主操作。
 *   3. 「Memory Pro 新能力 · 短期记忆压缩」放进可折叠的「能力更新」
 *      时间线，从主流抽离，未来新增能力按同格式追加。
 *   4. 「新建 Agent 默认策略」收纳为可折叠卡，文案明确「只影响未来」。
 *   5. 行内三态 Switch 替换为「状态 Tag + 操作按钮组」：动作有方向、
 *      不可回退场景天然适合按钮，覆盖更多状态（异常 / 升级中 / 迁移中）。
 *
 * 访问路径：/admin/memory-management-redesign
 */
import React, { useMemo, useState } from 'react';
import { toast } from 'sonner';
import {
  Search,
  RefreshCw,
  Loader2,
  Info,
  ChevronDown,
  ChevronUp,
  Sparkles,
  AlertCircle,
  CheckCircle2,
  X,
} from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { StatusTag } from '@/components/ui/status-tag';
import { Checkbox } from '@/components/ui/checkbox';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell } from '@/components/ui/table';
import { HelperText } from '@/components/ui/Typography';
import { Pagination } from '@/components/ui/pagination';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';

// ============================================================================
// 类型定义
// ============================================================================
type MemoryVersion = 'none' | 'free' | 'pro';
type MemoryState =
  | 'idle'
  | 'free-running'
  | 'pro-running'
  | 'enabling-free'
  | 'enabling-pro'
  | 'closing'
  | 'plugin-upgrading'
  | 'error';

interface AgentInstance {
  id: string;
  name: string;
  creator: string;
  agentType: 'OpenClaw' | 'Hermes Agent';
  state: MemoryState;
  version: MemoryVersion;
  enabledAt?: string;
  errorMessage?: string;
}

type ProServiceStatus = 'inactive' | 'activating' | 'active';

const FIXED_QUOTA = 500;

// ============================================================================
// Mock 数据
// ============================================================================
const INITIAL_INSTANCES: AgentInstance[] = [
  { id: 'oc-000', name: '智能问答助手', creator: 'zhangsan@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-001', name: '客服助手', creator: 'lisi@tencent.com', agentType: 'Hermes Agent', state: 'free-running', version: 'free', enabledAt: '2026-04-12 14:30' },
  { id: 'oc-002', name: '营销策划师', creator: 'wangwu@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-003', name: '数据分析师', creator: 'zhaoliu@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-004', name: '代码助手', creator: 'zhangsan@tencent.com', agentType: 'Hermes Agent', state: 'idle', version: 'none' },
  { id: 'oc-005', name: '文档编写助手', creator: 'lisi@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-006', name: '培训教练', creator: 'wangwu@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-007', name: '产品经理助手', creator: 'zhaoliu@tencent.com', agentType: 'Hermes Agent', state: 'idle', version: 'none' },
  { id: 'oc-008', name: '人力资源顾问', creator: 'sunqi@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-009', name: '财务分析助手', creator: 'zhouba@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-010', name: '运维监控助手', creator: 'wujiu@tencent.com', agentType: 'Hermes Agent', state: 'idle', version: 'none' },
  { id: 'oc-011', name: '法务合规助手', creator: 'zhengshi@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-012', name: '设计灵感助手', creator: 'zhangsan@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-013', name: '项目管理助手', creator: 'lisi@tencent.com', agentType: 'Hermes Agent', state: 'idle', version: 'none' },
  { id: 'oc-014', name: '内容审核助手', creator: 'wangwu@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-015', name: '翻译助手', creator: 'zhaoliu@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-016', name: '测试工程助手', creator: 'sunqi@tencent.com', agentType: 'Hermes Agent', state: 'idle', version: 'none' },
  { id: 'oc-017', name: '安全审计助手', creator: 'zhouba@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-018', name: '知识库管理助手', creator: 'wujiu@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-019', name: 'GPULab 智能运营分析助手', creator: 'product-ops@tencent.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
  { id: 'oc-020', name: '超长名称用以测试截断效果的智能助手', creator: 'longname-user@enterprise-acompany.com', agentType: 'OpenClaw', state: 'idle', version: 'none' },
];

// ============================================================================
// 状态 Tag：表达「记忆服务当前是什么状态」
// ============================================================================
const StateTag: React.FC<{ instance: AgentInstance }> = ({ instance }) => {
  const { state, enabledAt } = instance;

  switch (state) {
    case 'idle':
      return <StatusTag variant="gray" mode="dot">未开启</StatusTag>;
    case 'free-running':
      return (
        <div className="flex flex-col gap-0.5">
          <StatusTag variant="blue" mode="dot">Free 版</StatusTag>
          {enabledAt && <span className="text-[11px] text-[#A3A3A3] pl-3">{enabledAt} 开启</span>}
        </div>
      );
    case 'pro-running':
      return (
        <div className="flex flex-col gap-0.5">
          <span className="inline-flex items-center gap-1">
            <span className="h-1.5 w-1.5 rounded-full bg-purple-500" />
            <span className="text-sm font-medium text-purple-600">Pro 版</span>
          </span>
          {enabledAt && <span className="text-[11px] text-[#A3A3A3] pl-3">{enabledAt} 开启</span>}
        </div>
      );
    case 'enabling-free':
      return (
        <span className="inline-flex items-center gap-1.5 text-sm text-[#1447E6]">
          <Loader2 className="w-3.5 h-3.5 animate-spin" />
          正在开通 Free 版...
        </span>
      );
    case 'enabling-pro':
      return (
        <span className="inline-flex items-center gap-1.5 text-sm text-purple-600">
          <Loader2 className="w-3.5 h-3.5 animate-spin" />
          正在开通 Pro 版...
        </span>
      );
    case 'closing':
      return (
        <span className="inline-flex items-center gap-1.5 text-sm text-[#737373]">
          <Loader2 className="w-3.5 h-3.5 animate-spin" />
          正在关闭服务...
        </span>
      );
    case 'plugin-upgrading':
      return (
        <div className="flex flex-col gap-0.5">
          <span className="inline-flex items-center gap-1">
            <span className="h-1.5 w-1.5 rounded-full bg-purple-500" />
            <span className="text-sm font-medium text-purple-600">Pro 版</span>
            <Loader2 className="w-3 h-3 animate-spin text-purple-500 ml-1" />
          </span>
          <span className="text-[11px] text-[#A3A3A3] pl-3">能力升级中</span>
        </div>
      );
    case 'error':
      return <StatusTag variant="red" mode="dot">服务异常</StatusTag>;
    default:
      return null;
  }
};

// ============================================================================
// 行内操作按钮组：根据状态决定有哪些按钮可用
// ============================================================================
interface RowActionsProps {
  instance: AgentInstance;
  isProActive: boolean;
  proSpaceRemaining: number;
  onEnableFree: (i: AgentInstance) => void;
  onEnablePro: (i: AgentInstance) => void;
  onClose: (i: AgentInstance) => void;
}

const RowActions: React.FC<RowActionsProps> = ({
  instance,
  isProActive,
  proSpaceRemaining,
  onEnableFree,
  onEnablePro,
  onClose,
}) => {
  const { state } = instance;

  // 过渡态：不显示任何操作
  if (state === 'enabling-free' || state === 'enabling-pro' || state === 'closing') {
    return <span className="text-xs text-[#A3A3A3]">—</span>;
  }

  // 异常态
  if (state === 'error') {
    return (
      <div className="flex items-center gap-2">
        <Button variant="link" size="sm">重试</Button>
        <Button variant="link-dark" size="sm">查看日志</Button>
      </div>
    );
  }

  // 升级中
  if (state === 'plugin-upgrading') {
    return <Button variant="link" size="sm">查看进度</Button>;
  }

  const proDisabledReason = !isProActive
    ? '请先开通 Memory Pro 服务'
    : proSpaceRemaining <= 0
      ? 'Pro 配额已满，请联系商务扩容'
      : null;

  if (state === 'idle') {
    return (
      <div className="flex items-center gap-2">
        <Button
          variant="claw-outline"
          size="claw-sm"
          onClick={() => onEnableFree(instance)}
        >
          开通 Free
        </Button>
        {proDisabledReason ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button variant="claw-outline" size="claw-sm" disabled>
                  开通 Pro
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>{proDisabledReason}</TooltipContent>
          </Tooltip>
        ) : (
          <Button
            variant="claw-primary"
            size="claw-sm"
            onClick={() => onEnablePro(instance)}
          >
            开通 Pro
          </Button>
        )}
      </div>
    );
  }

  if (state === 'free-running') {
    return (
      <div className="flex items-center gap-2">
        {proDisabledReason ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span>
                <Button variant="claw-primary" size="claw-sm" disabled>
                  升级到 Pro
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>{proDisabledReason}</TooltipContent>
          </Tooltip>
        ) : (
          <Button
            variant="claw-primary"
            size="claw-sm"
            onClick={() => onEnablePro(instance)}
          >
            升级到 Pro
          </Button>
        )}
        <Button variant="link-dark" size="sm" onClick={() => onClose(instance)}>
          关闭
        </Button>
      </div>
    );
  }

  if (state === 'pro-running') {
    // Pro 不允许降级到 Free（业务规则），仅保留「关闭」入口
    return (
      <Button variant="link-dark" size="sm" onClick={() => onClose(instance)}>
        关闭
      </Button>
    );
  }

  return null;
};

// ============================================================================
// 主页面
// ============================================================================
export const MemoryManagementRedesign: React.FC = () => {
  const [proStatus, setProStatus] = useState<ProServiceStatus>('inactive');
  const [instances, setInstances] = useState<AgentInstance[]>(INITIAL_INSTANCES);

  // UI 状态
  const [defaultPolicyExpanded, setDefaultPolicyExpanded] = useState(false);
  const [updatesExpanded, setUpdatesExpanded] = useState(false);
  const [versionDiffOpen, setVersionDiffOpen] = useState(false);
  const [defaultVersion, setDefaultVersion] = useState<MemoryVersion>('none');

  // 列表筛选
  const [search, setSearch] = useState('');
  const [agentTypeFilter, setAgentTypeFilter] = useState<'all' | 'OpenClaw' | 'Hermes Agent'>('all');
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 8;

  // 批量选择
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  // 统计
  const stats = useMemo(() => {
    const total = instances.length;
    const proCount = instances.filter(i => i.version === 'pro' || i.state === 'plugin-upgrading').length;
    const freeCount = instances.filter(i => i.version === 'free').length;
    const idleCount = instances.filter(i => i.state === 'idle' || i.state === 'error').length;
    return { total, proCount, freeCount, idleCount };
  }, [instances]);

  const proRemaining = proStatus === 'active' ? FIXED_QUOTA - stats.proCount : 0;
  const proUsedPercent = proStatus === 'active' && FIXED_QUOTA > 0
    ? Math.round((stats.proCount / FIXED_QUOTA) * 100)
    : 0;

  // 列表过滤
  const filtered = useMemo(() => {
    return instances.filter(i => {
      const matchSearch =
        !search ||
        i.name.toLowerCase().includes(search.toLowerCase()) ||
        i.id.toLowerCase().includes(search.toLowerCase());
      const matchType = agentTypeFilter === 'all' || i.agentType === agentTypeFilter;
      return matchSearch && matchType;
    });
  }, [instances, search, agentTypeFilter]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  // ========== 服务级动作 ==========
  const handleActivatePro = () => {
    setProStatus('activating');
    toast.info('Memory Pro 正在初始化中...');
    setTimeout(() => {
      setProStatus('active');
      toast.success('Memory Pro 服务已开通，500 个 Pro 名额已就绪');
    }, 1500);
  };

  const handleClosePro = () => {
    if (stats.proCount > 0) {
      toast.error(`当前还有 ${stats.proCount} 个 Pro 实例，请先关闭后再关闭服务`);
      return;
    }
    setProStatus('inactive');
    toast.success('Memory Pro 服务已关闭');
  };

  // ========== 实例级动作 ==========
  const updateInstance = (id: string, patch: Partial<AgentInstance>) => {
    setInstances(prev => prev.map(i => (i.id === id ? { ...i, ...patch } : i)));
  };

  const handleEnableFree = async (instance: AgentInstance) => {
    updateInstance(instance.id, { state: 'enabling-free' });
    toast.info(`正在为「${instance.name}」开通 Free 版...`);
    await new Promise(r => setTimeout(r, 1500));
    updateInstance(instance.id, {
      state: 'free-running',
      version: 'free',
      enabledAt: new Date().toLocaleString('sv-SE').slice(0, 16).replace('T', ' '),
    });
    toast.success(`已为「${instance.name}」开通 Free 版`);
  };

  const handleEnablePro = async (instance: AgentInstance) => {
    if (proStatus !== 'active') {
      toast.error('请先开通 Memory Pro 服务');
      return;
    }
    if (proRemaining <= 0) {
      toast.error('Pro 配额已满');
      return;
    }
    const fromFree = instance.version === 'free';
    updateInstance(instance.id, { state: 'enabling-pro' });
    toast.info(fromFree
      ? `正在将「${instance.name}」从 Free 升级到 Pro 版（数据迁移中）...`
      : `正在为「${instance.name}」开通 Pro 版...`);
    await new Promise(r => setTimeout(r, fromFree ? 2500 : 1500));
    updateInstance(instance.id, {
      state: 'pro-running',
      version: 'pro',
      enabledAt: new Date().toLocaleString('sv-SE').slice(0, 16).replace('T', ' '),
    });
    toast.success(`已为「${instance.name}」开通 Pro 版`);
  };

  const handleClose = async (instance: AgentInstance) => {
    updateInstance(instance.id, { state: 'closing' });
    toast.info(`正在关闭「${instance.name}」的记忆服务...`);
    await new Promise(r => setTimeout(r, 1200));
    updateInstance(instance.id, {
      state: 'idle',
      version: 'none',
      enabledAt: undefined,
    });
    toast.success(`已关闭「${instance.name}」的记忆服务`);
  };

  // ========== 批量动作 ==========
  const selectedInstances = instances.filter(i => selectedIds.has(i.id));
  const selectableInPage = paged.filter(
    i => i.state !== 'enabling-free' && i.state !== 'enabling-pro' && i.state !== 'closing' && i.state !== 'plugin-upgrading'
  );
  const isAllInPageSelected =
    selectableInPage.length > 0 && selectableInPage.every(i => selectedIds.has(i.id));

  const togglePageSelection = (checked: boolean) => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (checked) {
        selectableInPage.forEach(i => next.add(i.id));
      } else {
        selectableInPage.forEach(i => next.delete(i.id));
      }
      return next;
    });
  };

  const handleBatchEnableFree = async () => {
    const targets = selectedInstances.filter(i => i.state === 'idle');
    if (!targets.length) return toast.warning('所选实例中没有可开通 Free 的');
    setSelectedIds(new Set());
    toast.info(`正在为 ${targets.length} 个实例开通 Free 版...`);
    targets.forEach(t => updateInstance(t.id, { state: 'enabling-free' }));
    await new Promise(r => setTimeout(r, 1800));
    targets.forEach(t =>
      updateInstance(t.id, {
        state: 'free-running',
        version: 'free',
        enabledAt: new Date().toLocaleString('sv-SE').slice(0, 16).replace('T', ' '),
      })
    );
    toast.success(`已为 ${targets.length} 个实例开通 Free 版`);
  };

  const handleBatchEnablePro = async () => {
    if (proStatus !== 'active') return toast.error('请先开通 Memory Pro 服务');
    const targets = selectedInstances.filter(i => i.state === 'idle' || i.state === 'free-running');
    if (!targets.length) return toast.warning('所选实例均已是 Pro 版');
    if (targets.length > proRemaining) {
      return toast.error(`Pro 配额不足（剩余 ${proRemaining}，需要 ${targets.length}）`);
    }
    setSelectedIds(new Set());
    toast.info(`正在为 ${targets.length} 个实例开通 Pro 版...`);
    targets.forEach(t => updateInstance(t.id, { state: 'enabling-pro' }));
    await new Promise(r => setTimeout(r, 2200));
    targets.forEach(t =>
      updateInstance(t.id, {
        state: 'pro-running',
        version: 'pro',
        enabledAt: new Date().toLocaleString('sv-SE').slice(0, 16).replace('T', ' '),
      })
    );
    toast.success(`已为 ${targets.length} 个实例开通 Pro 版`);
  };

  const handleBatchClose = async () => {
    const targets = selectedInstances.filter(
      i => i.state === 'free-running' || i.state === 'pro-running'
    );
    if (!targets.length) return toast.warning('所选实例没有已开启的记忆服务');
    setSelectedIds(new Set());
    toast.info(`正在关闭 ${targets.length} 个实例的记忆服务...`);
    targets.forEach(t => updateInstance(t.id, { state: 'closing' }));
    await new Promise(r => setTimeout(r, 1500));
    targets.forEach(t =>
      updateInstance(t.id, { state: 'idle', version: 'none', enabledAt: undefined })
    );
    toast.success(`已关闭 ${targets.length} 个实例的记忆服务`);
  };

  // ========== 渲染 ==========
  const isProActive = proStatus === 'active';
  const isProActivating = proStatus === 'activating';

  return (
    <div className="page-enter pb-12">
      {/* 顶部 banner：标识这是预览页面 */}
      <div className="mb-6 flex items-start gap-2 bg-amber-50 border border-amber-100 rounded-[4px] px-4 py-2.5">
        <Sparkles className="w-4 h-4 text-amber-600 mt-0.5 shrink-0" />
        <p className="text-xs text-amber-700 leading-relaxed">
          这是「记忆管理」的<strong>重设计预览页</strong>，与原版 <code className="px-1 py-0.5 bg-white rounded">/admin/memory-management</code> 并存，便于对比评估。
        </p>
      </div>

      {/* ========== ① 页头 + 服务总开关条 ========== */}
      <div className="mb-6">
        <div className="flex items-end justify-between mb-2">
          <div>
            <h1 className="text-2xl font-bold text-[#0A0A0A]">记忆管理</h1>
            <p className="text-sm text-[#737373] mt-1">
              让 AI 智能体记住你的偏好和工作习惯。当前已为 <strong>{stats.total}</strong> 个 Agent 提供服务（OpenClaw / Hermes）。
            </p>
          </div>
          <button
            onClick={() => setVersionDiffOpen(!versionDiffOpen)}
            className="text-sm text-[#1447E6] hover:underline inline-flex items-center gap-1"
          >
            <Info className="w-4 h-4" />
            了解 Free 与 Pro 的区别
          </button>
        </div>

        {/* 服务状态条 */}
        {!isProActive && !isProActivating && (
          <div className="bg-gradient-to-r from-purple-50 via-blue-50 to-white border border-purple-100 rounded-[4px] px-5 py-4 flex items-center gap-4">
            <div className="shrink-0 w-10 h-10 rounded-[4px] bg-gradient-to-br from-purple-500 to-blue-500 flex items-center justify-center">
              <Sparkles className="w-5 h-5 text-white" />
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h3 className="text-sm font-semibold text-[#0A0A0A]">Memory Pro 服务</h3>
                <StatusTag variant="blue">免费体验中</StatusTag>
              </div>
              <p className="text-xs text-[#737373] mt-1">
                解锁更长的记忆周期、跨会话联想、短期记忆压缩等高级能力（500 名额起）
              </p>
            </div>
            <Button variant="claw-primary" size="claw" onClick={handleActivatePro}>
              立即开通 Pro
            </Button>
          </div>
        )}

        {isProActivating && (
          <div className="bg-blue-50 border border-blue-100 rounded-[4px] px-5 py-4 flex items-center gap-3">
            <Loader2 className="w-5 h-5 text-[#1447E6] animate-spin shrink-0" />
            <p className="text-sm text-[#1447E6]">Memory Pro 正在初始化中，预计需要几分钟...</p>
          </div>
        )}

        {isProActive && (
          <div className="bg-white border border-gray-200 rounded-[4px] px-5 py-3.5 flex items-center gap-4">
            <CheckCircle2 className="w-5 h-5 text-emerald-500 shrink-0" />
            <div className="flex-1 flex items-center gap-3 text-sm">
              <span className="font-semibold text-[#0A0A0A]">Memory Pro 服务已开通</span>
              <span className="text-[#A3A3A3]">·</span>
              <span className="text-[#737373]">{FIXED_QUOTA} 个名额</span>
              <span className="text-[#A3A3A3]">·</span>
              <span className="text-[#737373]">免费体验中</span>
            </div>
            <Button variant="link-dark" size="sm" onClick={handleClosePro}>
              关闭服务
            </Button>
          </div>
        )}

        {/* 版本对比抽屉式说明（点击展开） */}
        {versionDiffOpen && (
          <div className="mt-3 bg-white border border-gray-200 rounded-[4px] p-5 animate-in fade-in slide-in-from-top-1 duration-200">
            <div className="flex items-start justify-between mb-3">
              <h3 className="text-sm font-semibold text-[#0A0A0A]">Free 版 vs Pro 版</h3>
              <button onClick={() => setVersionDiffOpen(false)} className="text-[#A3A3A3] hover:text-[#0A0A0A]">
                <X className="w-4 h-4" />
              </button>
            </div>
            <div className="grid grid-cols-3 gap-x-6 text-sm">
              <div className="text-xs font-medium text-[#737373]">能力</div>
              <div className="text-xs font-medium text-[#1447E6]">Free 版</div>
              <div className="text-xs font-medium text-purple-600">Pro 版</div>
              {[
                ['记忆周期', '7 天', '永久'],
                ['跨会话记忆', '不支持', '支持'],
                ['短期记忆压缩', '不支持', '节省 45% Token'],
                ['配额', '不限', '按池子分配'],
                ['Agent 类型', 'OpenClaw / Hermes', 'OpenClaw / Hermes'],
              ].map(([k, f, p]) => (
                <React.Fragment key={k}>
                  <div className="text-sm text-[#0A0A0A] py-2 border-t border-[#F5F5F5]">{k}</div>
                  <div className="text-sm text-[#737373] py-2 border-t border-[#F5F5F5]">{f}</div>
                  <div className="text-sm text-[#0A0A0A] py-2 border-t border-[#F5F5F5]">{p}</div>
                </React.Fragment>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* ========== ② 配额使用情况（仅 Pro 已开通显示） ========== */}
      {isProActive && (
        <div className="mb-6 bg-white border border-gray-200 rounded-[4px] px-6 py-5">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-sm font-semibold text-[#0A0A0A]">记忆配额</h2>
            <button className="text-xs text-[#1447E6] hover:underline">申请扩容 →</button>
          </div>

          <div className="flex items-end gap-8 mb-3">
            <div>
              <div className="text-xs text-[#737373] mb-1">Pro 已分配</div>
              <div className="flex items-baseline gap-1">
                <span className="text-2xl font-bold text-purple-600" style={{ fontFamily: "'DIN Next LT Pro', 'DIN', sans-serif" }}>
                  {stats.proCount}
                </span>
                <span className="text-sm text-[#A3A3A3]">/ {FIXED_QUOTA}</span>
              </div>
            </div>
            <div className="h-10 w-px bg-[#E5E5E5]" />
            <div>
              <div className="text-xs text-[#737373] mb-1">Free 已开启</div>
              <div className="text-2xl font-bold text-[#1447E6]" style={{ fontFamily: "'DIN Next LT Pro', 'DIN', sans-serif" }}>
                {stats.freeCount}
              </div>
            </div>
            <div className="h-10 w-px bg-[#E5E5E5]" />
            <div>
              <div className="text-xs text-[#737373] mb-1">未开启</div>
              <div className="text-2xl font-bold text-[#737373]" style={{ fontFamily: "'DIN Next LT Pro', 'DIN', sans-serif" }}>
                {stats.idleCount}
              </div>
            </div>
          </div>

          {/* 进度条 */}
          <div className="h-1.5 bg-[#F5F5F5] rounded-full overflow-hidden">
            <div
              className={`h-full transition-all ${
                proUsedPercent >= 100 ? 'bg-red-500' :
                proUsedPercent >= 80 ? 'bg-amber-500' :
                'bg-purple-500'
              }`}
              style={{ width: `${Math.min(proUsedPercent, 100)}%` }}
            />
          </div>
          <div className="flex justify-between mt-1.5 text-[11px]">
            <span className="text-[#A3A3A3]">{proUsedPercent}% 已用 · 剩余 {proRemaining} 个</span>
            {proUsedPercent >= 80 && (
              <span className={proUsedPercent >= 100 ? 'text-red-500' : 'text-amber-600'}>
                {proUsedPercent >= 100 ? '配额已满' : '即将用完'}
              </span>
            )}
          </div>

          {proUsedPercent >= 80 && (
            <div className={`mt-3 flex items-start gap-2 rounded-[4px] px-3 py-2 ${
              proUsedPercent >= 100 ? 'bg-red-50 border border-red-100' : 'bg-amber-50 border border-amber-100'
            }`}>
              <AlertCircle className={`w-3.5 h-3.5 mt-0.5 shrink-0 ${
                proUsedPercent >= 100 ? 'text-red-500' : 'text-amber-500'
              }`} />
              <p className={`text-xs ${proUsedPercent >= 100 ? 'text-red-700' : 'text-amber-700'}`}>
                {proUsedPercent >= 100
                  ? 'Pro 配额已用完，新实例无法升级到 Pro，请联系商务扩容。'
                  : `Pro 配额即将用完，建议提前申请扩容。`}
              </p>
            </div>
          )}
        </div>
      )}

      {/* ========== ③ 默认策略（折叠卡） ========== */}
      <div className="mb-6 bg-white border border-gray-200 rounded-[4px] overflow-hidden">
        <button
          onClick={() => setDefaultPolicyExpanded(!defaultPolicyExpanded)}
          className="w-full px-6 py-3.5 flex items-center justify-between hover:bg-[#FAFAFA] transition-colors"
        >
          <div className="flex items-center gap-2">
            <Info className="w-4 h-4 text-[#737373]" />
            <span className="text-sm font-medium text-[#0A0A0A]">新建 Agent 默认策略</span>
            <span className="text-xs text-[#A3A3A3]">·</span>
            <span className="text-xs text-[#A3A3A3]">
              当前：{defaultVersion === 'none' ? '不开启记忆' : defaultVersion === 'free' ? 'Free 版' : 'Pro 版'}
            </span>
          </div>
          <span className="text-sm text-[#737373] flex items-center gap-0.5">
            {defaultPolicyExpanded ? '收起' : '展开'}
            {defaultPolicyExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
          </span>
        </button>

        {defaultPolicyExpanded && (
          <div className="px-6 pb-5 pt-1 border-t border-[#F5F5F5]">
            <p className="text-xs text-[#737373] mb-3">
              这里的设置只影响 <strong>以后新建</strong> 的 Agent，不会改动下方已有实例的记忆配置。
            </p>
            <div className="flex items-center gap-2">
              {([
                { key: 'none', label: '不开启', desc: '新 Agent 默认无记忆' },
                { key: 'free', label: 'Free 版', desc: '默认开启 Free 版' },
                { key: 'pro', label: 'Pro 版', desc: '默认开启 Pro 版（消耗配额）' },
              ] as const).map(opt => {
                const active = defaultVersion === opt.key;
                const proUnavailable = opt.key === 'pro' && (!isProActive || proRemaining <= 0);
                return (
                  <Tooltip key={opt.key}>
                    <TooltipTrigger asChild>
                      <button
                        disabled={proUnavailable}
                        onClick={() => setDefaultVersion(opt.key)}
                        className={`px-4 py-2 text-sm rounded-[4px] border transition-all ${
                          active
                            ? 'bg-[#0A0A0A] border-[#0A0A0A] text-white'
                            : proUnavailable
                              ? 'bg-[#FAFAFA] border-gray-200 text-[#A3A3A3] cursor-not-allowed'
                              : 'bg-white border-gray-200 text-[#0A0A0A] hover:border-[#0A0A0A]'
                        }`}
                      >
                        {opt.label}
                      </button>
                    </TooltipTrigger>
                    <TooltipContent>
                      {proUnavailable
                        ? '请先开通 Pro 服务且有可用配额'
                        : opt.desc}
                    </TooltipContent>
                  </Tooltip>
                );
              })}
            </div>
          </div>
        )}
      </div>

      {/* ========== ④ 实例列表（主操作区） ========== */}
      <div className="bg-white border border-gray-200 rounded-[4px] overflow-hidden">
        {/* 工具栏 */}
        <div className="px-6 py-4 border-b border-[#F5F5F5] flex items-center justify-between gap-4 flex-wrap">
          <div className="flex items-center gap-3">
            <h2 className="text-sm font-semibold text-[#0A0A0A]">记忆空间</h2>
            <span className="text-xs text-[#A3A3A3]">共 {filtered.length} 个实例</span>
          </div>

          {/* 批量操作浮条 - 仅在选中后显示 */}
          {selectedIds.size > 0 ? (
            <div className="flex items-center gap-2 bg-blue-50 border border-blue-100 rounded-[4px] px-3 py-1.5">
              <span className="text-xs font-medium text-[#1447E6]">已选 {selectedIds.size} 项</span>
              <span className="h-3 w-px bg-blue-200" />
              <button
                onClick={handleBatchEnableFree}
                className="text-xs text-[#1447E6] hover:bg-blue-100 px-2 py-1 rounded transition-colors"
              >
                开通 Free
              </button>
              <button
                onClick={handleBatchEnablePro}
                disabled={!isProActive}
                className="text-xs text-purple-600 hover:bg-purple-50 px-2 py-1 rounded transition-colors disabled:text-[#A3A3A3] disabled:cursor-not-allowed disabled:hover:bg-transparent"
              >
                升级到 Pro
              </button>
              <button
                onClick={handleBatchClose}
                className="text-xs text-red-600 hover:bg-red-50 px-2 py-1 rounded transition-colors"
              >
                关闭服务
              </button>
              <button
                onClick={() => setSelectedIds(new Set())}
                className="text-xs text-[#737373] hover:text-[#0A0A0A] ml-1"
              >
                清除
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-3">
              {/* Agent 类型筛选 */}
              <Select value={agentTypeFilter} onValueChange={(v) => setAgentTypeFilter(v as any)}>
                <SelectTrigger className="w-[140px]">
                  <SelectValue placeholder="全部类型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部类型</SelectItem>
                  <SelectItem value="OpenClaw">OpenClaw</SelectItem>
                  <SelectItem value="Hermes Agent">Hermes Agent</SelectItem>
                </SelectContent>
              </Select>

              {/* 搜索 */}
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[#A3A3A3]" />
                <Input
                  placeholder="搜索 Agent 名称或 ID"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-9 w-60"
                />
              </div>

              <button
                className="w-9 h-9 flex items-center justify-center rounded-[4px] border border-gray-200 bg-white text-[#737373] hover:text-[#1447E6] hover:border-[#1447E6] transition-colors"
                title="刷新"
                onClick={() => toast.success('已刷新')}
              >
                <RefreshCw className="w-4 h-4" />
              </button>
            </div>
          )}
        </div>

        {/* 表格 */}
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-12">
                <Checkbox
                  checked={isAllInPageSelected}
                  onCheckedChange={(c) => togglePageSelection(!!c)}
                />
              </TableHead>
              <TableHead style={{ width: '26%' }}>Agent 名称 / ID</TableHead>
              <TableHead style={{ width: '20%' }}>创建人</TableHead>
              <TableHead style={{ width: '12%' }}>类型</TableHead>
              <TableHead style={{ width: '18%' }}>记忆服务</TableHead>
              <TableHead style={{ width: '24%' }}>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {paged.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6}>
                  <div className="text-center py-12">
                    <HelperText>暂无符合条件的实例</HelperText>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              paged.map(inst => {
                const isLocked =
                  inst.state === 'enabling-free' ||
                  inst.state === 'enabling-pro' ||
                  inst.state === 'closing' ||
                  inst.state === 'plugin-upgrading';
                return (
                  <TableRow key={inst.id}>
                    <TableCell className="w-12">
                      <Checkbox
                        checked={selectedIds.has(inst.id)}
                        disabled={isLocked}
                        onCheckedChange={(c) => {
                          setSelectedIds(prev => {
                            const next = new Set(prev);
                            if (c) next.add(inst.id);
                            else next.delete(inst.id);
                            return next;
                          });
                        }}
                      />
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-col gap-0.5 min-w-0">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="font-medium truncate max-w-[260px]">
                              {inst.name}
                            </span>
                          </TooltipTrigger>
                          <TooltipContent>{inst.name}</TooltipContent>
                        </Tooltip>
                        <span className="font-mono text-[#1447E6]">{inst.id}</span>
                      </div>
                    </TableCell>
                    <TableCell className="text-gray-500">{inst.creator}</TableCell>
                    <TableCell className="text-gray-500">{inst.agentType}</TableCell>
                    <TableCell>
                      <StateTag instance={inst} />
                    </TableCell>
                    <TableActionCell rawChildren>
                      <RowActions
                        instance={inst}
                        isProActive={isProActive}
                        proSpaceRemaining={proRemaining}
                        onEnableFree={handleEnableFree}
                        onEnablePro={handleEnablePro}
                        onClose={handleClose}
                      />
                    </TableActionCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>

        {/* 分页 */}
        {filtered.length > PAGE_SIZE && (
          <div className="px-6 py-3 border-t border-[#F5F5F5]">
            <Pagination
              total={filtered.length}
              current={page}
              pageSize={PAGE_SIZE}
              showTotal={(t) => `共 ${t} 条记录`}
              size="default"
              className="w-full justify-between"
              onChange={setPage}
            />
          </div>
        )}
      </div>

      {/* ========== ⑤ 能力更新（折叠时间线） ========== */}
      <div className="mt-6 bg-white border border-gray-200 rounded-[4px] overflow-hidden">
        <button
          onClick={() => setUpdatesExpanded(!updatesExpanded)}
          className="w-full px-6 py-3.5 flex items-center justify-between hover:bg-[#FAFAFA] transition-colors"
        >
          <div className="flex items-center gap-2">
            <Sparkles className="w-4 h-4 text-purple-500" />
            <span className="text-sm font-medium text-[#0A0A0A]">Memory Pro 能力更新</span>
            <StatusTag variant="blue">2 项新能力</StatusTag>
          </div>
          <span className="text-sm text-[#737373] flex items-center gap-0.5">
            {updatesExpanded ? '收起' : '展开'}
            {updatesExpanded ? <ChevronUp className="w-4 h-4" /> : <ChevronDown className="w-4 h-4" />}
          </span>
        </button>

        {updatesExpanded && (
          <div className="px-6 pb-5 pt-3 border-t border-[#F5F5F5]">
            <div className="space-y-4">
              {/* 时间线节点 */}
              <div className="relative pl-6 border-l-2 border-purple-200">
                <span className="absolute -left-[7px] top-1 w-3 h-3 rounded-full bg-purple-500 ring-4 ring-purple-50" />
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="text-xs text-[#A3A3A3]">2026-05</span>
                      <h4 className="text-sm font-semibold text-[#0A0A0A]">短期记忆压缩</h4>
                      <StatusTag variant="blue">新</StatusTag>
                    </div>
                    <p className="text-xs text-[#737373] leading-relaxed">
                      基于 WideSearch 等数据集测试，长任务可节省 <strong className="text-purple-600">45%</strong> 的 Token 消耗、提高 <strong className="text-purple-600">20%</strong> 完成率。
                      <span className="text-[#A3A3A3]"> · 适用：OpenClaw 类型 Pro 版 Agent</span>
                    </p>
                  </div>
                  <Button
                    variant="claw-outline"
                    size="claw-sm"
                    disabled={!isProActive || stats.proCount === 0}
                    onClick={() => toast.success('已下发升级任务')}
                  >
                    一键启用
                  </Button>
                </div>
              </div>

              <div className="relative pl-6 border-l-2 border-gray-200">
                <span className="absolute -left-[7px] top-1 w-3 h-3 rounded-full bg-[#A3A3A3] ring-4 ring-[#FAFAFA]" />
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs text-[#A3A3A3]">2026-03</span>
                    <h4 className="text-sm font-semibold text-[#0A0A0A]">跨会话长期记忆</h4>
                  </div>
                  <p className="text-xs text-[#737373]">Pro 版默认启用，无需操作。</p>
                </div>
              </div>

              <div className="relative pl-6">
                <span className="absolute -left-[7px] top-1 w-3 h-3 rounded-full bg-[#D4D4D4] ring-4 ring-[#FAFAFA]" />
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <span className="text-xs text-[#A3A3A3]">2026-01</span>
                    <h4 className="text-sm font-semibold text-[#0A0A0A]">Memory Pro 正式上线</h4>
                  </div>
                  <p className="text-xs text-[#737373]">企业级记忆服务，支持 OpenClaw / Hermes Agent。</p>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default MemoryManagementRedesign;
