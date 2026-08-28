import React, { useState, useEffect, useRef } from 'react';
import { toast } from 'sonner';
import { ComparisonTable } from './components/ComparisonTable';
import { InstanceTable, OcInstance, MemoryStatus } from './components/InstanceTable';
import { ProActivationDialog } from './components/ProActivationDialog';
import { ProCloseDialog } from './components/ProCloseDialog';
import { OneClickUpgradeDialog } from './components/OneClickUpgradeDialog';
import { DefaultMemoryVersion, MemoryVersionRule } from './components/DefaultMemoryVersion';
import { Loader2, Info, ChevronDown, ChevronUp, ArrowUpCircle, CircleOff, Zap, Crown, AlertCircle, CheckCircle2, X, Bot } from 'lucide-react';
import { SurfaceCard, SurfaceInner } from '@/components/ui/Surface';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { AdminPageHeader } from '@/components/ui/admin-page-header';
import { AdminNoticeAlert } from '@/components/ui/admin-notice-alert';
import {
  PanelTitle,
  CardTitle,
  BodyMedium,
  MetaText,
  HelperText,
  StatNumber,
} from '@/components/ui/Typography';
import { useMemoryManagementPortalBillingExempt } from './useMemoryManagementPortalBillingExempt';

// 配置常量
const FIXED_MEMORY_SPACES = 500; // 固定配额：每个用户限额 500 个记忆空间

// Pro 服务状态类型
type ProServiceStatus = 'inactive' | 'activating' | 'active' | 'error';

export const MemoryManagement: React.FC = () => {
  // 停服态下，本页面各种弹窗（"新建 Agent 默认记忆版本"Dialog、
  // "Pro 激活/关闭"Dialog、"一键升级"Dialog、"切换后将清空组织策略"
  // AlertDialog 等）都通过 Radix Portal 挂到<body>，脱离主体页面容器；
  // 若不在 dialog-content 上打data-billing-exempt，会被 AdminDisabledOverlay
  // 视觉灰化 + 文档级capture 事件拦截 —— 用户连"关闭 X"都点不动，被卡在弹窗里。
  // 详见 ./useMemoryManagementPortalBillingExempt.ts 头部注释。
  useMemoryManagementPortalBillingExempt();
  // ========== Pro 服务状态 ==========
  const [proServiceStatus, setProServiceStatus] = useState<ProServiceStatus>('inactive');
  const [purchasedSpaces, setPurchasedSpaces] = useState<number>(0);

  const [showSuccessBanner, setShowSuccessBanner] = useState(false);
  // 新实例默认记忆版本：支持「预设策略 + 组织例外」
  // - 第 1 条 groupIds 为空数组 = 预设策略（fallback），唯一且不可删除
  // - 其余为组织例外，组织之间互斥
  const [memoryVersionRules, setMemoryVersionRules] = useState<MemoryVersionRule[]>([
    { id: 'mem-default-fallback', groupIds: [], value: 'none' },
  ]);

  // ========== 弹窗状态 ==========
  const [activationDialogOpen, setActivationDialogOpen] = useState(false);
  const [closeDialogOpen, setCloseDialogOpen] = useState(false);
  // 一键升级弹窗：触发后由弹窗内部异步检测可升级实例并展示对应态
  const [oneClickUpgradeDialogOpen, setOneClickUpgradeDialogOpen] = useState(false);
  // 版本对比折叠状态，默认收起
  const [versionCompareExpanded, setVersionCompareExpanded] = useState(false);
  
  // 实例列表 ref，用于滚动定位
  const instanceTableRef = useRef<HTMLDivElement>(null);

  // ========== Mock 实例数据 ==========
  // 给 19 个 mock 实例按固定模式分配 agentType，保证 OC 占多数、Hermes 适中、LightClaw ACE 少量，
  // 便于在记忆空间列表中验证「Agent 类型」列的展示效果。
  // 真实接入后，agentType 由后端返回，这段映射可移除。
  const [instances, setInstances] = useState<OcInstance[]>(() => {
    const baseList: OcInstance[] = [
      { id: 'oc-000', name: '智能问答助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'zhangsan@tencent.com' },
      { id: 'oc-001', name: '客服助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'lisi@tencent.com' },
      { id: 'oc-002', name: '营销策划师', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'wangwu@tencent.com' },
      { id: 'oc-003', name: '数据分析师', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'zhaoliu@tencent.com' },
      { id: 'oc-004', name: '代码助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'zhangsan@tencent.com' },
      { id: 'oc-005', name: '文档编写助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'lisi@tencent.com' },
      { id: 'oc-006', name: '培训教练', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'wangwu@tencent.com' },
      { id: 'oc-007', name: '产品经理助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'zhaoliu@tencent.com' },
      { id: 'oc-008', name: '人力资源顾问', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'sunqi@tencent.com' },
      { id: 'oc-009', name: '财务分析助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'zhouba@tencent.com' },
      { id: 'oc-010', name: '运维监控助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'wujiu@tencent.com' },
      { id: 'oc-011', name: '法务合规助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'zhengshi@tencent.com' },
      { id: 'oc-012', name: '设计灵感助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'zhangsan@tencent.com' },
      { id: 'oc-013', name: '项目管理助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'lisi@tencent.com' },
      { id: 'oc-014', name: '内容审核助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'wangwu@tencent.com' },
      { id: 'oc-015', name: '翻译助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'zhaoliu@tencent.com' },
      { id: 'oc-016', name: '测试工程助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'sunqi@tencent.com' },
      { id: 'oc-017', name: '安全审计助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'zhouba@tencent.com' },
      { id: 'oc-018', name: '知识库管理助手', memoryStatus: 'none', version: 'none', state: 'idle', memoryId: '-', enabledAt: '-', creator: 'wujiu@tencent.com' },
      { id: 'oc-019', name: '这是一个名称非常非常长的智能助手用来测试超长文本截断效果', memoryStatus: 'pro', version: 'none', state: 'running', memoryId: 'mem-long-001', enabledAt: '2026-05-01', creator: 'longname-user@very-long-domain-example.com' },
      { id: 'oc-020', name: 'GPULab产品线专属AI智能运营分析与决策支持系统', memoryStatus: 'free', version: 'none', state: 'running', memoryId: 'mem-long-002', enabledAt: '2026-05-02', creator: 'product-ops-admin@enterprise-acompany.com' },
    ];
    // 给 21 个 mock 实例按固定模式分配 agentType：每 3 个里第 2 个为 Hermes，其余为 OpenClaw，
    // 便于在记忆空间列表中验证「Agent 类型」列的展示效果。
    // 真实接入后，agentType 由后端返回，这段映射可移除。
    return baseList.map((inst, idx) => {
      const agentType: 'openclaw' | 'hermes' = idx % 3 === 1 ? 'hermes' : 'openclaw';
      return { ...inst, agentType };
    });
  });

  const [loading, setLoading] = useState(false);

  // 统计数据
  const stats = {
    total: instances.length,
    proCount: instances.filter(i => i.memoryStatus === 'pro').length,
    freeCount: instances.filter(i => i.memoryStatus === 'free').length,
    noneCount: instances.filter(i => i.memoryStatus === 'none').length,
  };

  // 一键启用短期记忆压缩 · 候选集：本期能力「短期记忆压缩」仅对 OpenClaw 类型的 Pro 版 Agent 生效，
  // 因此候选集只包含 agentType === 'openclaw' && memoryStatus === 'pro' 且未在插件升级中的 Agent。
  // 控制台不预判版本新旧，点击入口后由弹窗发起异步检测、由后端返回真正需要升级的清单。
  // 关闭、开启中、关闭中、异常、插件升级中等中间态 Agent 不在候选范围内。
  const upgradeCandidates = instances.filter(
    i => (i.agentType ?? 'openclaw') === 'openclaw'
      && i.memoryStatus === 'pro'
      && !i.isPluginUpgrading
  );
  // 是否有任一 Agent 正在异步升级中：升级期间禁用"一键升级"入口，避免重复下发。
  const hasUpgradingInstance = instances.some(i => i.isPluginUpgrading);

  // Pro 额度使用率
  const memoryAllocationPercent = purchasedSpaces > 0
    ? Math.round((stats.proCount / purchasedSpaces) * 100)
    : 0;

  // 状态判断
  const isProInactive = proServiceStatus === 'inactive';
  const isProActivating = proServiceStatus === 'activating';
  const isProActive = proServiceStatus === 'active';

  // 开通 Pro 服务（固定 500 配额）
  const handleActivatePro = (config?: { autoEnableForNewInstances: boolean }) => {
    setProServiceStatus('activating');
    setPurchasedSpaces(FIXED_MEMORY_SPACES);
    
    setTimeout(() => {
      if (Math.random() > 0.1) {
        setProServiceStatus('active');
        setShowSuccessBanner(true);
        // 联动逻辑：如果勾选了「默认开通」，将「预设策略」切换为 Pro
        // 不影响已有的组织例外（用户既有的组织级覆盖应保留）
        if (config?.autoEnableForNewInstances) {
          setMemoryVersionRules(prev => prev.map(r =>
            r.groupIds.length === 0 ? { ...r, value: 'pro' as const } : r
          ));
          toast.success('Memory Pro 服务开通成功，新实例将默认开启 Pro 版');
        } else {
          toast.success('Memory Pro 服务开通成功！');
        }
      } else {
        setProServiceStatus('error');
        toast.error('服务初始化失败，请重试');
      }
    }, 2000);
  };

  // 关闭 Pro 服务
  const handleClosePro = () => {
    setProServiceStatus('inactive');
    setPurchasedSpaces(0);
    // 方案丙：关闭 Pro 服务时一切归零
    //   1. 预设策略：统一切到「关闭」（无论原值是 Pro / Free / 关闭）
    //   2. 组织策略：全部清空
    let presetChanged = false;
    let clearedGroupCount = 0;
    setMemoryVersionRules(prev => {
      const next: typeof prev = [];
      prev.forEach(r => {
        const isPreset = r.groupIds.length === 0;
        if (isPreset) {
          if (r.value !== 'none') {
            presetChanged = true;
            next.push({ ...r, value: 'none' as const });
          } else {
            next.push(r);
          }
        } else {
          // 组织策略：全部丢弃
          clearedGroupCount++;
        }
      });
      return next;
    });
    if (presetChanged && clearedGroupCount > 0) {
      toast.success(`Memory Pro 服务已关闭，预设策略已切换为关闭，${clearedGroupCount} 条组织策略已清空`);
    } else if (presetChanged) {
      toast.success('Memory Pro 服务已关闭，预设策略已切换为关闭');
    } else if (clearedGroupCount > 0) {
      toast.success(`Memory Pro 服务已关闭，${clearedGroupCount} 条组织策略已清空`);
    } else {
      toast.success('Memory Pro 服务已关闭');
    }
  };

  // 成功提示条自动消失
  useEffect(() => {
    if (showSuccessBanner) {
      const timer = setTimeout(() => setShowSuccessBanner(false), 3000);
      return () => clearTimeout(timer);
    }
  }, [showSuccessBanner]);

  return (
    <div className="page-enter">
      {/* 页面头部 */}
      <AdminPageHeader
        title="记忆管理"
        description="让 AI 智能体真正理解你、记住你，长期保持一致的工作习惯与决策偏好。由腾讯云数据库 Agent Memory 服务提供支持（已支持 OpenClaw、Hermes，其他 Agent 类型敬请期待）。"
      />

      {/* 状态提示条 - 初始化中 */}
      {isProActivating && (
        <div className="mb-6">
          <AdminNoticeAlert type="pending-config" tagLabel="初始化">
            <span>Memory Pro 正在初始化中，预计需要几分钟...</span>
          </AdminNoticeAlert>
        </div>
      )}

      {/* 状态提示条 - 成功 */}
      {showSuccessBanner && isProActive && (
        <div className="mb-6 animate-in fade-in duration-300">
          <AdminNoticeAlert type="product-news">
            <span>Memory Pro 已就绪</span>
          </AdminNoticeAlert>
        </div>
      )}

      {/* 顶部：版本对比说明（可折叠） */}
      <SurfaceCard className="mb-6 overflow-hidden">
        {/* 折叠触发器 */}
        <button
          onClick={() => setVersionCompareExpanded(!versionCompareExpanded)}
          className={`w-full px-6 py-4 flex items-center justify-between transition-colors ${
            versionCompareExpanded ? '' : 'hover:bg-[#FAFAFA]'
          }`}
        >
          <div className="flex items-center gap-2">
            <Info className="w-4 h-4 text-[var(--text-title)]" />
            <BodyMedium tone="primary">了解 Memory Free 版与 Pro 版的区别</BodyMedium>
          </div>
          <ChevronDown
            className={`w-4 h-4 text-[var(--text-muted)] transition-transform duration-200 ${versionCompareExpanded ? 'rotate-180' : ''}`}
          />
        </button>
        
        {/* 可折叠内容 */}
        {versionCompareExpanded && (
          <div className="px-6 pt-4 pb-5">
            <ComparisonTable 
              isProActive={isProActive}
            />
          </div>
        )}
      </SurfaceCard>

      {/* 服务概览 - 统计卡片 */}
      <SurfaceCard className="mb-6">
        <div className="px-6 py-5 border-b border-[#EAEEF4]">
          <PanelTitle>服务概览</PanelTitle>
        </div>
        <div className="p-5">
          <div className="grid grid-cols-5 gap-4">
            {/* 实例总数 */}
            <SurfaceInner className="px-6 py-5 flex flex-col gap-4">
              <div className="flex items-center gap-1">
                <img src="/assets/admin-memory-management/instance-total.svg" className="shrink-0" />
                <CardTitle as="span">实例总数</CardTitle>
              </div>
              <StatNumber>{stats.total}</StatNumber>
            </SurfaceInner>

            {/* 未开启 */}
            <SurfaceInner className="px-6 py-5 flex flex-col gap-4">
              <div className="flex items-center gap-1">
                <img src="/assets/admin-memory-management/instance-disabled.svg" className="shrink-0" />
                <CardTitle as="span">未开启</CardTitle>
              </div>
              <StatNumber>{stats.noneCount}</StatNumber>
            </SurfaceInner>

            {/* Free 版 */}
            <SurfaceInner className="px-6 py-5 flex flex-col gap-4">
              <div className="flex items-center gap-1">
                <img src="/assets/admin-memory-management/instance-free.svg" className="shrink-0" />
                <CardTitle as="span">Free 版</CardTitle>
              </div>
              <StatNumber>{stats.freeCount}</StatNumber>
            </SurfaceInner>

            {/* Pro 版 - 融合配额管理 */}
            <SurfaceInner
              className={`col-span-2 px-6 py-5 flex flex-col gap-4 ${
                isProActive && memoryAllocationPercent >= 100
                  ? 'border-red-200'
                  : isProActive && memoryAllocationPercent >= 80
                    ? 'border-yellow-200'
                    : ''
              }`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1">
                  <img src="/assets/admin-memory-management/instance-pro.svg" className="shrink-0" />
                  <div className="flex flex-col gap-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <CardTitle as="span">Pro 版</CardTitle>
                      <Badge color="blue">免费体验中</Badge>
                    </div>

                  </div>
                </div>
                <div className="flex items-center gap-3 shrink-0">
                  {isProInactive && (
                    <Button
                      variant="claw-primary"
                      size="claw-sm"
                      onClick={() => setActivationDialogOpen(true)}
                    >
                      立即开通
                    </Button>
                  )}
                  {isProActive && (
                    // Pro 卡片右上角仅保留「关闭服务」入口；
                    // 升级/开通能力的入口统一收口到 Banner 上的「一键开通」，
                    // 避免在 Pro 卡片再放一个语义重复的「一键升级」造成认知割裂。
                    <Button
                      variant="link"
                      size="sm"
                      onClick={(e) => { e.stopPropagation(); setCloseDialogOpen(true); }}
                      className="!text-[var(--text-muted)] hover:!text-[var(--text-danger)]"
                    >
                      关闭服务
                    </Button>
                  )}
                  {isProActivating && (
                    <span className="inline-flex items-center gap-1 whitespace-nowrap">
                      <Loader2 className="w-3 h-3 animate-spin text-[var(--text-brand)]" />
                      <MetaText tone="brand">初始化中</MetaText>
                    </span>
                  )}
                </div>
              </div>

              {/* 未开通状态 */}
              {isProInactive && (
                <div className="flex items-baseline gap-2">
                  <StatNumber tone="weak">0/0</StatNumber>
                </div>
              )}

              {/* 开通中状态 */}
              {isProActivating && (
                <div>
                  <div className="h-8 w-24 bg-gray-200 rounded animate-pulse mb-2" />
                  <div className="h-2 w-full bg-gray-200 rounded-full animate-pulse" />
                </div>
              )}

              {/* 已开通状态 */}
              {isProActive && (
                <div>
                  <div className="flex items-baseline gap-2 mb-1">
                    <StatNumber tone="brand">{stats.proCount}/{purchasedSpaces}</StatNumber>
                    <MetaText>
                      已分配 <span className="text-[var(--text-brand)]">{stats.proCount}</span> 个，剩余 <span className="text-[var(--text-brand)]">{purchasedSpaces - stats.proCount}</span> 个可分配
                    </MetaText>
                  </div>
                  {/* 进度条 */}
                  <div className="mt-2">
                    <Progress
                      value={Math.min(memoryAllocationPercent, 100)}
                      className="h-1.5"
                    />
                    <div className="flex justify-between items-center mt-1">
                      <HelperText as="span">{memoryAllocationPercent}% 已用</HelperText>
                      {memoryAllocationPercent >= 80 && (
                        <HelperText
                          as="span"
                          tone={memoryAllocationPercent >= 100 ? 'danger' : 'muted'}
                        >
                          {memoryAllocationPercent >= 100 ? '空间已满' : '即将用完'}
                        </HelperText>
                      )}
                    </div>
                  </div>
                </div>
              )}

            </SurfaceInner>
          </div>

          {/* 记忆空间告警提示 */}
          {isProActive && memoryAllocationPercent >= 80 && (
            <div className="mt-4">
              <AdminNoticeAlert type={memoryAllocationPercent >= 100 ? 'resource-alert' : 'pending-config'} tagLabel={memoryAllocationPercent >= 100 ? '资源告警' : '容量预警'}>
                <span>
                  {memoryAllocationPercent >= 100
                    ? 'Pro 记忆空间已用完，用户将无法新开启 Memory Pro 功能。如需更多空间请联系商务。'
                    : `Pro 记忆空间即将用完（${stats.proCount}/${purchasedSpaces}）`
                  }
                </span>
              </AdminNoticeAlert>
            </div>
          )}

          {/* Memory Pro 新能力 · 介绍条 */}
          <SurfaceInner className="mt-4 px-5 py-4 flex items-center gap-4">
            <div className="flex-1 min-w-0">
              <BodyMedium as="div" className="mb-1">Memory Pro 新能力：短期记忆压缩</BodyMedium>
              <MetaText as="p" className="leading-relaxed">
                基于 WideSearch 等数据集测试，长任务可节省 <span className="font-semibold text-[var(--text-title)]">45%</span> 的 Token 消耗、提高 <span className="font-semibold text-[var(--text-title)]">20%</span> 完成率（需开通 Pro 并升级记忆服务至最新版本，暂仅对 OpenClaw 类型 Agent 生效）
              </MetaText>
            </div>
            {(() => {
              const hasCandidate = upgradeCandidates.length > 0;
              const disabled = !isProActive || hasUpgradingInstance || !hasCandidate;
              // 仅在禁用态提供 Tooltip 说明原因；可点击态不展示 Tooltip，避免对用户造成干扰。
              const title = !isProActive
                ? '请先开通 Memory Pro'
                : hasUpgradingInstance
                  ? '有 Agent 正在升级中，请等待当前任务完成后再发起'
                  : !hasCandidate
                    ? '暂无可启用的 Pro 版 Agent'
                    : undefined;
              return (
                <Button
                  variant="claw-primary"
                  size="claw-sm"
                  onClick={() => {
                    if (disabled) return;
                    setOneClickUpgradeDialogOpen(true);
                  }}
                  disabled={disabled}
                  title={title}
                  className="shrink-0"
                >
                  一键启用
                </Button>
              );
            })()}
          </SurfaceInner>

          {/* 新实例默认记忆版本 - 支持「预设策略 + 组织例外」 */}
          <div className="mt-4">
            <DefaultMemoryVersion
              rules={memoryVersionRules}
              onRulesChange={setMemoryVersionRules}
              isProActive={isProActive}
              isProQuotaAvailable={purchasedSpaces - stats.proCount > 0}
            />
          </div>
        </div>
      </SurfaceCard>

      {/* 实例列表 */}
      <div ref={instanceTableRef}>
      <InstanceTable 
        instances={instances} 
        loading={loading}
        isProActive={isProActive}
        proSpacesAvailable={purchasedSpaces - stats.proCount}
        onEnableFree={async (instance) => {
          // 第一步：立即将状态变为 free-enabling
          setInstances(prev => prev.map(i => 
            i.id === instance.id 
              ? { ...i, memoryStatus: 'free-enabling' as MemoryStatus, version: 'free' as const, state: 'enabling' as const }
              : i
          ));
          toast.info(`正在为「${instance.name}」开启 Free 版记忆...`);
          
          // 第二步：模拟 API 调用延迟
          await new Promise(resolve => setTimeout(resolve, 2000));
          
          // 第三步：开启完成，更新为 free 状态
          setInstances(prev => prev.map(i => 
            i.id === instance.id 
              ? { ...i, memoryStatus: 'free' as MemoryStatus, version: 'free' as const, state: 'running' as const, memoryId: `mem-local-${instance.id.split('-')[1]}`, enabledAt: new Date().toLocaleString('sv-SE', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).replace('T', ' ') }
              : i
          ));
          toast.success(`已为「${instance.name}」开启 Free 版记忆`);
        }}
        onEnablePro={async (instance) => {
          if (!isProActive) {
            toast.error('请先开通 Memory Pro 服务');
            return;
          }
          if (purchasedSpaces - stats.proCount <= 0) {
            toast.error('Pro 记忆空间已满，如需更多空间请联系商务');
            return;
          }
          
          // 判断是否从 Free 版开启（需要数据迁移）
          const isFromFree = instance.memoryStatus === 'free';
          
          // 第一步：立即将状态变为 pro-enabling
          setInstances(prev => prev.map(i => 
            i.id === instance.id 
              ? { ...i, memoryStatus: 'pro-enabling' as MemoryStatus, version: 'pro' as const, state: 'enabling' as const }
              : i
          ));
          
          if (isFromFree) {
            toast.info(`正在为「${instance.name}」开启 Pro 版记忆，数据迁移中...`);
          } else {
            toast.info(`正在为「${instance.name}」开启 Pro 版记忆...`);
          }
          
          // 第二步：模拟 API 调用延迟（如果是从 Free 迁移，时间更长）
          await new Promise(resolve => setTimeout(resolve, isFromFree ? 4000 : 2000));
          
          // 第三步：开启完成，更新为 pro 状态
          setInstances(prev => prev.map(i => 
            i.id === instance.id 
              ? { ...i, memoryStatus: 'pro' as MemoryStatus, version: 'pro' as const, state: 'running' as const, memoryId: `mem-${instance.id.split('-')[1]}`, enabledAt: new Date().toLocaleString('sv-SE', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).replace('T', ' ') }
              : i
          ));
          toast.success(`已为「${instance.name}」开启 Pro 版记忆`);
        }}
        onDisableMemory={async (instance) => {
          const wasProVersion = instance.memoryStatus === 'pro';
          // 模拟 API 调用延迟
          await new Promise(resolve => setTimeout(resolve, 1500));
          setInstances(prev => prev.map(i => 
            i.id === instance.id 
              ? { ...i, memoryStatus: 'none' as MemoryStatus, version: 'none' as const, state: 'idle' as const, memoryId: '-', enabledAt: '-' }
              : i
          ));
          toast.success(`已关闭「${instance.name}」的${wasProVersion ? ' Pro 版' : ' Free 版'}记忆`);
        }}
        onBatchEnableFree={async (selectedInstances) => {
          toast.info(`正在为 ${selectedInstances.length} 个实例批量开通 Free 版记忆...`);
          
          // 第一步：将所有选中实例状态变为 free-enabling
          setInstances(prev => prev.map(i => 
            selectedInstances.some(s => s.id === i.id)
              ? { ...i, memoryStatus: 'free-enabling' as MemoryStatus, version: 'free' as const, state: 'enabling' as const }
              : i
          ));
          
          // 第二步：模拟 API 调用延迟
          await new Promise(resolve => setTimeout(resolve, 2000));
          
          // 第三步：开启完成
          setInstances(prev => prev.map(i => {
            if (selectedInstances.some(s => s.id === i.id)) {
              return { 
                ...i, 
                memoryStatus: 'free' as MemoryStatus, 
                version: 'free' as const, 
                state: 'running' as const,
                memoryId: `mem-local-${i.id.split('-')[1]}`,
                enabledAt: new Date().toLocaleString('sv-SE', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).replace('T', ' ')
              };
            }
            return i;
          }));
          
          toast.success(`已为 ${selectedInstances.length} 个实例开通 Free 版记忆`);
        }}
        onBatchEnablePro={async (selectedInstances) => {
          toast.info(`正在为 ${selectedInstances.length} 个实例批量升级 Pro 版记忆...`);
          
          // 第一步：将所有选中实例状态变为 pro-enabling
          setInstances(prev => prev.map(i => 
            selectedInstances.some(s => s.id === i.id)
              ? { ...i, memoryStatus: 'pro-enabling' as MemoryStatus, version: 'pro' as const, state: 'enabling' as const }
              : i
          ));
          
          // 第二步：模拟 API 调用延迟
          await new Promise(resolve => setTimeout(resolve, 3000));
          
          // 第三步：开启完成
          setInstances(prev => prev.map(i => {
            if (selectedInstances.some(s => s.id === i.id)) {
              return { 
                ...i, 
                memoryStatus: 'pro' as MemoryStatus, 
                version: 'pro' as const, 
                state: 'running' as const,
                memoryId: `mem-${i.id.split('-')[1]}`,
                enabledAt: new Date().toLocaleString('sv-SE', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).replace('T', ' ')
              };
            }
            return i;
          }));
          
          toast.success(`已为 ${selectedInstances.length} 个实例升级 Pro 版记忆`);
        }}
        onBatchDisable={async (selectedInstances) => {
          const proCount = selectedInstances.filter(i => i.memoryStatus === 'pro').length;
          const freeCount = selectedInstances.filter(i => i.memoryStatus === 'free').length;
          
          toast.info(`正在为 ${selectedInstances.length} 个实例批量关闭记忆服务...`);
          
          // 第一步：将所有选中实例状态变为 closing
          setInstances(prev => prev.map(i => 
            selectedInstances.some(s => s.id === i.id)
              ? { ...i, memoryStatus: 'closing' as MemoryStatus, state: 'closing' as const }
              : i
          ));
          
          // 第二步：模拟 API 调用延迟
          await new Promise(resolve => setTimeout(resolve, 2000));
          
          // 第三步：关闭完成
          setInstances(prev => prev.map(i => {
            if (selectedInstances.some(s => s.id === i.id)) {
              return { 
                ...i, 
                memoryStatus: 'none' as MemoryStatus, 
                version: 'none' as const, 
                state: 'idle' as const,
                memoryId: '-',
                enabledAt: '-'
              };
            }
            return i;
          }));
          
          toast.success(`已关闭 ${selectedInstances.length} 个实例的记忆服务（${proCount > 0 ? `${proCount} 个 Pro 版` : ''}${proCount > 0 && freeCount > 0 ? '、' : ''}${freeCount > 0 ? `${freeCount} 个 Free 版` : ''}）`);
        }}
      />
      </div>

      {/* 弹窗 */}
      <ProActivationDialog
        open={activationDialogOpen}
        onOpenChange={setActivationDialogOpen}
        onConfirm={handleActivatePro}
      />

      <ProCloseDialog
        open={closeDialogOpen}
        onOpenChange={setCloseDialogOpen}
        onConfirm={handleClosePro}
        ocCount={stats.proCount}
        groupPolicyCount={memoryVersionRules.filter(r => r.groupIds.length > 0).length}
        presetVersion={memoryVersionRules.find(r => r.groupIds.length === 0)?.value}
        onGoToInstanceList={() => {
          instanceTableRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }}
      />
      {/* 一键升级弹窗 —— 打开后由弹窗内部异步检测需要升级的 Agent：
            - 检测中：loading 态
            - 有可升级：展示清单 + 影响说明，确认后异步下发任务、弹窗立即关闭
            - 全部最新：友好提示，客户关闭即可
          升级过程由列表行内「插件升级中」loading 体现，任务完成后 Agent 自动回到稳态。*/}
      <OneClickUpgradeDialog
        open={oneClickUpgradeDialogOpen}
        onOpenChange={setOneClickUpgradeDialogOpen}
        candidateInstances={upgradeCandidates}
        onConfirm={(targets) => {
          // 异步下发升级任务：
          // 1) 立即把所有 target Agent 标记为「插件升级中」
          // 2) 给出 toast 提示，用户可以离开页面
          // 3) 每个 Agent 独立随机延迟（3~6 秒）后回到稳态，模拟后端异步回调
          if (targets.length === 0) return;
          const targetIds = new Set(targets.map(t => t.id));
          setInstances(prev => prev.map(i =>
            targetIds.has(i.id) ? { ...i, isPluginUpgrading: true } : i
          ));
          toast.success(`已下发 ${targets.length} 个 Agent 的记忆服务升级任务`);

          targets.forEach(t => {
            const delay = 3000 + Math.floor(Math.random() * 3000);
            setTimeout(() => {
              setInstances(prev => prev.map(i =>
                i.id === t.id ? { ...i, isPluginUpgrading: false } : i
              ));
            }, delay);
          });
        }}
      />
    </div>
  );
};

export default MemoryManagement;
