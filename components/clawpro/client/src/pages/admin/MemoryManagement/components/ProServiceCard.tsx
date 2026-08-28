import React, { useState, useCallback, useEffect, useRef } from 'react';
import {
  Gem,
  AlertCircle,
  Loader2,
  CheckCircle2,
  RotateCcw,
  X,
  Sparkles,
  Search,
  Shield,
  LayoutGrid,
  TrendingUp,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { toast } from 'sonner';
import { motion, AnimatePresence, useInView } from 'framer-motion';
import { ProActivationDialog } from './ProActivationDialog';
import { ProCloseDialog } from './ProCloseDialog';
import { RadarWidget } from './RadarWidget';

// ====== 评测数据（从 FreeVersionCard 迁移过来） ======
const BENCHMARK_DATA = [
  { label: '记住变化原因', tip: '知道你为什么改了主意', native: 70.97, free: 88.89, improvement: 25.25 },
  { label: '记住你说过的事', tip: '你提过的信息不会忘', native: 29.63, free: 79.07, improvement: 166.86 },
  { label: '记住关键信息', tip: '准确回忆对话中的事实', native: 25.00, free: 76.47, improvement: 205.88 },
  { label: '个性化推荐', tip: '基于你的习惯给出建议', native: 46.67, free: 76.36, improvement: 63.62 },
  { label: '跨场景理解', tip: '工作聊的事，生活场景也能用', native: 31.58, free: 78.95, improvement: 150.00 },
  { label: '跟踪偏好变化', tip: '你的喜好变了，它跟着变', native: 66.67, free: 83.45, improvement: 25.17 },
  { label: '创意启发', tip: '基于了解你给出新点子', native: 24.00, free: 45.16, improvement: 88.17 },
];

const TOTAL = { native: 47.85, free: 76.10, improvement: 59.04 };

// 动画计数器
function AnimatedCounter({
  value,
  duration = 1500,
  decimals = 2,
  suffix = '%',
  delay = 0,
}: {
  value: number;
  duration?: number;
  decimals?: number;
  suffix?: string;
  delay?: number;
}) {
  const [display, setDisplay] = useState(0);
  const ref = useRef<HTMLSpanElement>(null);
  const hasAnimated = useRef(false);
  const isInView = useInView(ref as React.RefObject<HTMLElement>, { once: true, amount: 0.5 });

  useEffect(() => {
    if (!isInView || hasAnimated.current) return;
    hasAnimated.current = true;
    const start = performance.now() + delay;
    let raf: number;
    const animate = (now: number) => {
      const elapsed = now - start;
      if (elapsed < 0) {
        raf = requestAnimationFrame(animate);
        return;
      }
      const p = Math.min(elapsed / duration, 1);
      const eased = 1 - Math.pow(1 - p, 3);
      setDisplay(eased * value);
      if (p < 1) raf = requestAnimationFrame(animate);
    };
    raf = requestAnimationFrame(animate);
    return () => cancelAnimationFrame(raf);
  }, [isInView, value, duration, delay]);

  return (
    <span ref={ref}>
      {display.toFixed(decimals)}
      {suffix}
    </span>
  );
}

// 紧凑维度行 — 单行双条形
function DimensionRow({
  data,
  index,
  radarHovered,
  waitingForExpand,
}: {
  data: (typeof BENCHMARK_DATA)[0];
  index: number;
  radarHovered: boolean;
  waitingForExpand: boolean;
}) {
  return (
    <div className="group">
      {/* Label row */}
      <div className="flex items-center justify-between mb-0.5">
        <span
          className="text-sm truncate transition-colors duration-300"
          style={{ color: radarHovered ? '#374151' : '#6b7280' }}
        >
          {data.label}
        </span>
        <AnimatePresence>
          {radarHovered && (
            <motion.span
              initial={{ opacity: 0, x: -6 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -6 }}
              transition={{ delay: index * 0.03 }}
              className="text-[11px] font-semibold text-green-600 ml-2 flex-shrink-0"
            >
              +{data.improvement.toFixed(0)}%
            </motion.span>
          )}
        </AnimatePresence>
      </div>

      {/* Dual bars stacked */}
      <div className="space-y-[2px]">
        {/* Agent bar */}
        <div className="flex items-center gap-1.5">
          <div className="flex-1 h-[4px] rounded-full overflow-hidden bg-gray-100">
            <div
              className="h-full rounded-full"
              style={{ background: '#d0d0e0', width: `${data.native}%` }}
            />
          </div>
          <span className="text-[11px] font-mono text-gray-400 w-[34px] text-right flex-shrink-0">
            {data.native.toFixed(0)}%
          </span>
        </div>

        {/* Memory bar — progressive reveal */}
        <div className="flex items-center gap-1.5">
          <div className="flex-1 h-[4px] rounded-full overflow-hidden bg-gray-100 relative">
            {/* Ghost dashed (idle) */}
            {!radarHovered && !waitingForExpand && (
              <div
                className="absolute inset-0 rounded-full"
                style={{
                  background:
                    'repeating-linear-gradient(90deg, rgba(59,130,246,0.12) 0px, rgba(59,130,246,0.12) 3px, transparent 3px, transparent 6px)',
                  width: `${data.free}%`,
                }}
              />
            )}
            {/* Solid bar (hovered/expanded) */}
            <motion.div
              className="h-full rounded-full relative z-10"
              style={{
                background: 'linear-gradient(90deg, #3B82F6, #2563EB)',
                boxShadow: radarHovered ? '0 0 6px rgba(59,130,246,0.25)' : 'none',
              }}
              initial={false}
              animate={{ width: radarHovered ? `${data.free}%` : '0%' }}
              transition={{
                delay: radarHovered ? 0.06 + index * 0.03 : 0,
                duration: radarHovered ? 0.4 : 0,
                ease: 'easeOut',
              }}
            />
          </div>
          <span
            className="text-[11px] font-mono font-semibold w-[34px] text-right flex-shrink-0 transition-colors duration-300"
            style={{ color: radarHovered ? '#2563EB' : 'rgba(59,130,246,0.25)' }}
          >
            {radarHovered ? `${data.free.toFixed(0)}%` : '??%'}
          </span>
        </div>
      </div>
    </div>
  );
}

// 骨架屏组件
const Skeleton: React.FC<{ className?: string }> = ({ className = '' }) => (
  <div className={`animate-pulse bg-gray-200 rounded ${className}`} />
);

type ServiceStatus = 'inactive' | 'activating' | 'active' | 'error';

interface ProServiceCardProps {
  serviceStatus: ServiceStatus;
  errorMessage: string;
  purchasedSpaces: number;
  proUsedCount: number;
  onActivated: () => void;
  onClosed: () => void;
  onRetry: () => void;
  /** 是否禁用（互斥时禁用） */
  disabled?: boolean;
  /** 禁用时 hover 提示文案 */
  disabledTooltip?: string;
  /** 点击"前往实例列表"按钮时的回调 */
  onGoToInstanceList?: () => void;
}

export const ProServiceCard: React.FC<ProServiceCardProps> = ({
  serviceStatus,
  errorMessage,
  purchasedSpaces,
  proUsedCount,
  onActivated,
  onClosed,
  onRetry,
  disabled = false,
  disabledTooltip = '',
  onGoToInstanceList,
}) => {
  const [activationDialogOpen, setActivationDialogOpen] = useState(false);
  const [closeDialogOpen, setCloseDialogOpen] = useState(false);
  const [showSuccessBanner, setShowSuccessBanner] = useState(false);
  const [radarHovered, setRadarHovered] = useState(true);
  const [autoExpanded, setAutoExpanded] = useState(false);

  const isInactive = serviceStatus === 'inactive';
  const isInitializing = serviceStatus === 'activating';
  const isError = serviceStatus === 'error';
  const isActive = serviceStatus === 'active';

  const memoryAllocationPercent = purchasedSpaces > 0
    ? Math.round((proUsedCount / purchasedSpaces) * 100)
    : 0;

  const formatNumber = (num: number) => num.toLocaleString('zh-CN');

  // 初始化完成后显示成功提示条
  React.useEffect(() => {
    if (isActive) {
      setShowSuccessBanner(true);
      const timer = setTimeout(() => setShowSuccessBanner(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [isActive]);

  // 页面挂载后延迟触发雷达图展开动画
  useEffect(() => {
    if (isInactive) {
      const timer = setTimeout(() => setAutoExpanded(true), 600);
      return () => clearTimeout(timer);
    } else {
      setAutoExpanded(false);
    }
  }, [isInactive]);

  const handleRadarHover = useCallback((h: boolean) => {
    setRadarHovered(h);
  }, []);

  const showExpanded = autoExpanded || radarHovered;

  // Pro 版核心优势数据
  const proFeatures: { icon: React.ElementType; color: string; bgColor: string; title: string; desc: string; tag?: string }[] = [
    {
      icon: Search,
      color: 'text-blue-600',
      bgColor: 'bg-blue-50',
      title: '混合双路检索',
      desc: '融合"关键字 + 向量语义"双路召回，精准捕获深层关联，让 Agent 的回答更精准、更具洞察力',
    },
    {
      icon: Shield,
      color: 'text-teal-600',
      bgColor: 'bg-teal-50',
      title: '企业级安全保障',
      desc: '提供完善的数据备份与强加密机制，匹配企业级数据隐私与合规要求，为核心资产保驾护航',
    },
    {
      icon: LayoutGrid,
      color: 'text-violet-600',
      bgColor: 'bg-violet-50',
      title: '全局资源管控',
      desc: '一站式可视化看板，统一管控所有实例的记忆资源，运维更省心',
    },
  ];

  // ==================== 未开通 / 开通中 / 开通失败 ====================
  if (isInactive || isInitializing || isError) {
    return (
      <TooltipProvider>
        <div
          className={`rounded-2xl overflow-hidden flex flex-col mb-5 ${disabled ? 'opacity-60' : ''}`}
          style={{
            background: '#ffffff',
            border: '1.5px solid rgba(59,130,246,0.15)',
            boxShadow: '0 1px 3px rgba(59,130,246,0.06), 0 4px 16px rgba(59,130,246,0.08)',
          }}
        >
          {/* ====== 头部：蓝色渐变背景 + 左侧文案 + 右侧即刻开通按钮 ====== */}
          <div
            className="px-8 py-7"
            style={{ background: 'linear-gradient(135deg, #4F8EF7 0%, #3B7BF2 30%, #2563EB 70%, #1D4ED8 100%)' }}
          >
            <div className="flex items-start justify-between">
              {/* 左侧信息 */}
              <div className="flex-1">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded-xl bg-white/20 backdrop-blur-sm flex items-center justify-center flex-shrink-0">
                    <Gem className="w-5 h-5 text-white" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="font-bold text-white text-lg">Memory Pro 版</h3>
                      <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-amber-400 text-amber-900 tracking-wide">
                        🎁 限时免费
                      </span>
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-white/20 text-white backdrop-blur-sm">
                        免费体验中
                      </span>
                    </div>
                    <p className="text-blue-100 text-sm mt-0.5">
                      基于腾讯云向量数据库的企业级记忆服务，接入腾讯云向量数据库与内置 Embedding 能力，实现语义级记忆检索与企业级数据管理。
                    </p>
                  </div>
                </div>
              </div>

              {/* 右侧：即刻开通按钮（红框位置） */}
              <div className="flex-shrink-0 ml-6 mt-1">
                {isInactive && (
                  disabled ? (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="cursor-not-allowed">
                          <button
                            disabled
                            className="inline-flex items-center gap-2 px-6 py-2.5 rounded-lg text-sm font-semibold text-blue-600 bg-white opacity-50 cursor-not-allowed flex-shrink-0"
                          >
                            <Sparkles className="w-4 h-4" />
                            立即开通
                          </button>
                        </span>
                      </TooltipTrigger>
                      <TooltipContent side="top" className="max-w-xs">
                        <p className="text-sm">{disabledTooltip}</p>
                      </TooltipContent>
                    </Tooltip>
                  ) : (
                    <button
                      onClick={() => setActivationDialogOpen(true)}
                      className="inline-flex items-center gap-2 px-6 py-2.5 rounded-lg text-sm font-semibold text-blue-600 bg-white transition-all hover:shadow-lg hover:shadow-blue-400/30 hover:scale-[1.02] active:scale-[0.98] flex-shrink-0"
                    >
                      <Sparkles className="w-4 h-4" />
                      立即开通
                    </button>
                  )
                )}
                {isInitializing && (
                  <button
                    disabled
                    className="inline-flex items-center gap-2 px-6 py-2.5 rounded-lg text-sm font-semibold text-blue-600 bg-white opacity-80 cursor-not-allowed flex-shrink-0"
                  >
                    <Loader2 className="w-4 h-4 animate-spin" />
                    开通中...
                  </button>
                )}
                {isError && (
                  <button
                    onClick={onRetry}
                    className="inline-flex items-center gap-2 px-6 py-2.5 rounded-lg text-sm font-semibold text-white transition-all hover:shadow-lg hover:shadow-red-400/20 hover:scale-[1.02] active:scale-[0.98] flex-shrink-0 bg-red-500 hover:bg-red-600"
                  >
                    <RotateCcw className="w-4 h-4" />
                    重试开通
                  </button>
                )}
              </div>
            </div>
          </div>

          {/* ====== 主体区域 ====== */}
          <div className="px-6 py-5 flex-1 flex flex-col bg-white">
            {/* 记忆效果对比标题 */}
            <div className="flex items-center gap-2 mb-4">
              <span className="text-sm font-semibold text-gray-900">记忆效果对比</span>
              <span className="text-[11px] text-gray-400">基于 PersonaMem 评测集</span>
            </div>

            {/* 上半部分：雷达图区 + 维度对比 并排 */}
            <div className="flex gap-6">
              {/* 左侧：雷达图 + 图例 */}
              <div className="flex-[45] flex flex-col items-center justify-center px-3">
                <RadarWidget hovered={showExpanded} onHoverChange={handleRadarHover} />

                {/* 图例 */}
                <div className="flex items-center gap-5 mt-3">
                  <div className="flex items-center gap-1.5">
                    <div className="w-2.5 h-2.5 rounded-full bg-gray-300" />
                    <span className="text-xs text-gray-400">Agent 原生</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <div
                      className="w-2.5 h-2.5 rounded-full"
                      style={{ background: 'linear-gradient(135deg, #3B82F6, #2563EB)' }}
                    />
                    <span className="text-xs text-gray-400">Memory Pro 版</span>
                  </div>
                </div>

                {/* Idle hint */}
                <AnimatePresence>
                  {!showExpanded && (
                    <motion.div
                      initial={{ opacity: 0 }}
                      animate={{ opacity: 1 }}
                      exit={{ opacity: 0 }}
                      className="mt-2"
                    >
                      <motion.p
                        animate={{ opacity: [0.4, 0.8, 0.4] }}
                        transition={{ duration: 3, repeat: Infinity, ease: 'easeInOut' }}
                        className="text-xs text-blue-400"
                      >
                        悬停雷达图查看对比详情
                      </motion.p>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>

              {/* 右侧：总分 + 维度对比 + 能力卡片 */}
              <div className="flex-[55] min-w-0">
                {/* 总分对比 Hero */}
                <div className="flex items-center gap-4 mb-4">
                  {/* 原生分数 */}
                  <div className="flex-1 text-center px-4 py-3 rounded-xl bg-gray-50 border border-gray-100">
                    <p className="text-xs text-gray-400 uppercase tracking-wide mb-1">Agent 原生</p>
                    <p className="text-2xl font-bold text-gray-400 font-mono">
                      <AnimatedCounter value={TOTAL.native} delay={200} duration={1800} />
                    </p>
                  </div>

                  {/* VS */}
                  <div className="w-9 h-9 rounded-full bg-gray-100 flex items-center justify-center flex-shrink-0">
                    <span className="text-xs font-bold text-gray-400">VS</span>
                  </div>

                  {/* Pro 版分数 */}
                  <div
                    className="flex-1 text-center px-4 py-3 rounded-xl border transition-all duration-500"
                    style={{
                      background: showExpanded ? 'rgba(59,130,246,0.04)' : 'rgba(59,130,246,0.02)',
                      borderColor: showExpanded ? 'rgba(59,130,246,0.2)' : 'rgba(59,130,246,0.08)',
                    }}
                  >
                    <p className="text-xs uppercase tracking-wide mb-1" style={{ color: '#2563EB' }}>
                      Memory Pro 版
                    </p>
                    <p className="text-2xl font-bold font-mono transition-colors duration-300" style={{ color: showExpanded ? '#2563EB' : '#bfdbfe' }}>
                      {showExpanded ? (
                        <AnimatedCounter value={TOTAL.free} delay={0} duration={800} />
                      ) : (
                        '??%'
                      )}
                    </p>
                  </div>

                  {/* 提升徽章 */}
                  <AnimatePresence>
                    {showExpanded && (
                      <motion.div
                        initial={{ opacity: 0, scale: 0.8 }}
                        animate={{ opacity: 1, scale: 1 }}
                        exit={{ opacity: 0, scale: 0.8 }}
                        className="flex-shrink-0"
                      >
                        <div className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-green-50 border border-green-100">
                          <TrendingUp className="w-3.5 h-3.5 text-green-600" />
                          <span className="text-sm font-bold text-green-600">
                            +<AnimatedCounter value={TOTAL.improvement} delay={200} duration={1000} decimals={1} />
                          </span>
                        </div>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>

                {/* 7 个维度对比 — 紧凑行内条形 */}
                <div className="rounded-lg border border-gray-100 bg-gray-50/30 px-4 py-3">
                  <div className="flex items-center justify-between mb-2">
                    <p className="text-xs font-semibold text-gray-400 uppercase tracking-wide">各维度记忆能力对比</p>
                    <div className="flex items-center gap-3">
                      <div className="flex items-center gap-1">
                        <div className="w-2 h-2 rounded-full bg-gray-300" />
                        <span className="text-[11px] text-gray-400">原生</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <div className="w-2 h-2 rounded-full" style={{ background: '#2563EB' }} />
                        <span className="text-[11px] text-gray-400">Pro 版</span>
                      </div>
                    </div>
                  </div>
                  <div className="space-y-2">
                    {BENCHMARK_DATA.map((d, i) => (
                      <DimensionRow
                        key={d.label}
                        data={d}
                        index={i}
                        radarHovered={showExpanded}
                        waitingForExpand={!autoExpanded}
                      />
                    ))}
                  </div>
                </div>
              </div>
            </div>

            {/* 核心能力标题 + 卡片网格 — 雷达图下方 */}
            <div className="flex items-center gap-2 mb-3 mt-5">
              <span className="text-sm font-semibold text-gray-900">Pro 版核心能力</span>
              <span className="text-[10px] px-1.5 py-0.5 rounded bg-blue-50 text-blue-600 font-semibold">全面升级</span>
            </div>
            <div className="grid grid-cols-4 gap-3">
              {proFeatures.map((feature) => (
                <div
                  key={feature.title}
                  className="rounded-lg p-4 transition-all hover:shadow-sm"
                  style={{
                    border: '1px solid rgba(59,130,246,0.1)',
                    background: 'rgba(59,130,246,0.02)',
                  }}
                >
                  <div className="flex items-center gap-2 mb-2">
                    <div className={`w-8 h-8 rounded-lg ${feature.bgColor} flex items-center justify-center flex-shrink-0`}>
                      <feature.icon className={`w-4 h-4 ${feature.color}`} />
                    </div>
                    <div className="font-medium text-sm text-gray-900">{feature.title}</div>
                    {feature.tag && (
                      <span className="text-[9px] px-1.5 py-0.5 rounded-full bg-amber-50 text-amber-600 font-medium border border-amber-100 whitespace-nowrap">
                        {feature.tag}
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-gray-500 leading-relaxed">{feature.desc}</div>
                </div>
              ))}
            </div>

            {/* 底部：状态文案（仅初始化中/错误时显示） */}
            {(isInitializing || isError) && (
              <div className="mt-4 pt-4 border-t border-blue-100/40">
                <div className="flex items-center">
                  {isInitializing && (
                    <span className="text-sm text-gray-500">正在初始化服务，请稍候...</span>
                  )}
                  {isError && (
                    <span className="text-sm text-red-600">{errorMessage || '初始化失败，请重试'}</span>
                  )}
                </div>
              </div>
            )}
          </div>

          <ProActivationDialog
            open={activationDialogOpen}
            onOpenChange={setActivationDialogOpen}
            onConfirm={onActivated}
          />
        </div>
      </TooltipProvider>
    );
  }

  // ==================== 已开通状态 ====================
  return (
    <TooltipProvider>
      <div
        className="bg-white rounded-xl border border-gray-100 px-6 py-6 mb-5"
        style={{ boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}
      >
        {/* 状态提示条 */}
        {isInitializing && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg px-4 py-2.5 flex items-center gap-3 mb-4">
            <Loader2 className="w-4 h-4 text-blue-500 animate-spin flex-shrink-0" />
            <span className="text-sm text-blue-700">Memory Pro 正在初始化中，预计需要几分钟...</span>
          </div>
        )}

        {isError && (
          <div className="bg-red-50 border border-red-200 rounded-lg px-4 py-2.5 flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <AlertCircle className="w-4 h-4 text-red-500 flex-shrink-0" />
              <span className="text-sm text-red-700">{errorMessage || 'Memory Pro 初始化失败，请重试'}</span>
            </div>
            <Button variant="outline" size="sm" className="text-red-600 border-red-300 hover:bg-red-100" onClick={onRetry}>
              <RotateCcw className="w-3.5 h-3.5 mr-1" />
              重试
            </Button>
          </div>
        )}

        {showSuccessBanner && isActive && (
          <div className="bg-green-50 border border-green-200 rounded-lg px-4 py-2.5 flex items-center justify-between mb-4 animate-in fade-in duration-300">
            <div className="flex items-center gap-3">
              <CheckCircle2 className="w-4 h-4 text-green-500 flex-shrink-0" />
              <span className="text-sm text-green-700">Memory Pro 已就绪</span>
            </div>
            <button onClick={() => setShowSuccessBanner(false)} className="text-green-500 hover:text-green-700">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* 头部 */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-blue-600 to-blue-500 flex items-center justify-center">
              <Gem className="w-5 h-5 text-white" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-gray-900">Memory Pro 服务</h3>
                <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-amber-400 text-amber-900 tracking-wide">
                  🎁 限时免费（至 8.15）
                </span>
                <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-blue-100 text-blue-600">
                  免费体验中
                </span>
                {isInitializing && (
                  <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-100 text-blue-600 rounded text-xs">
                    <Loader2 className="w-3 h-3 animate-spin" />
                    初始化中
                  </span>
                )}
              </div>
              <p className="text-sm text-gray-500 mt-0.5">
                基于腾讯云向量数据库的企业级记忆服务，统一管理所有 Agent 的记忆资源。
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Tooltip>
              <TooltipTrigger asChild>
                <span>
                  <Button
                    variant="outline"
                    size="sm"
                    className={`text-red-600 border-red-200 hover:bg-red-50 ${isInitializing ? 'opacity-50 cursor-not-allowed' : ''}`}
                    onClick={() => !isInitializing && setCloseDialogOpen(true)}
                    disabled={isInitializing}
                  >
                    关闭服务
                  </Button>
                </span>
              </TooltipTrigger>
              {isInitializing && (
                <TooltipContent><p>服务初始化中，暂不可操作</p></TooltipContent>
              )}
            </Tooltip>
          </div>
        </div>

        {/* 分割线 */}
        <div className="border-t border-gray-100 my-5" />

        {/* 额度信息 */}
        <div className="relative">
          {isInitializing && (
            <div className="absolute inset-0 bg-white/60 backdrop-blur-[1px] rounded-lg z-10 flex items-center justify-center">
              <div className="flex items-center gap-2 text-sm text-gray-500">
                <Loader2 className="w-4 h-4 animate-spin text-blue-500" />
                <span>数据加载中...</span>
              </div>
            </div>
          )}

          {isInitializing ? (
            <div className="space-y-3">
              <Skeleton className="h-8 w-24" />
              <Skeleton className="h-4 w-48" />
              <Skeleton className="h-2.5 w-full rounded-full" />
            </div>
          ) : (
            <>
              <div className="flex items-baseline gap-3 mb-3">
                <span className="text-3xl font-bold text-blue-600 tracking-tight">
                  {formatNumber(proUsedCount)}/{formatNumber(purchasedSpaces)}
                </span>
                <span className="text-sm text-gray-500">
                  已分配 <strong className="text-gray-700">{formatNumber(proUsedCount)}</strong> 个，
                  剩余 <strong className="text-gray-700">{formatNumber(purchasedSpaces - proUsedCount)}</strong> 个可分配
                </span>
              </div>

              <div className="mb-3">
                <div className="flex justify-between text-xs mb-1.5">
                  <span className="text-gray-500">Pro 额度使用率</span>
                  <span className="font-semibold text-blue-600">{memoryAllocationPercent}%</span>
                </div>
                <div className="h-2.5 bg-gray-100 rounded-full overflow-hidden">
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

              {memoryAllocationPercent >= 80 && (
                <div className={`px-3 py-2.5 rounded-lg text-xs flex items-center gap-2 mt-3 ${
                  memoryAllocationPercent >= 100
                    ? 'bg-red-50 border border-red-100 text-red-700'
                    : 'bg-amber-50 border border-amber-100 text-amber-700'
                }`}>
                  <AlertCircle className="w-4 h-4 flex-shrink-0" />
                  {memoryAllocationPercent >= 100
                    ? 'Pro 额度已用完，用户将无法新开启 Memory Pro 功能。如需更多空间请联系商务。'
                    : 'Pro 额度即将用完。'
                  }
                </div>
              )}
            </>
          )}
        </div>

        {/* 弹窗 */}
        <ProActivationDialog
          open={activationDialogOpen}
          onOpenChange={setActivationDialogOpen}
          onConfirm={onActivated}
        />

        <ProCloseDialog
          open={closeDialogOpen}
          onOpenChange={setCloseDialogOpen}
          onConfirm={onClosed}
          ocCount={proUsedCount}
          onGoToInstanceList={onGoToInstanceList}
        />
      </div>
    </TooltipProvider>
  );
};
