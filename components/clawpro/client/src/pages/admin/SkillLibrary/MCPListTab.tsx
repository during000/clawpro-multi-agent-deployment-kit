/**
 * MCPListTab - 企业 MCP 库列表
 * 复用企业插件库的列表 UI，展示 MCP 服务列表，操作包括下发和删除
 */
import { useState, useEffect, useCallback } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Search, Grid3x3, List, Send, Trash2, Loader, ShieldCheck, Settings } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { MOCK_CLOUD_OPENCLAW_INSTANCES, MOCK_GROUPS } from './mockData';
import MCPAddDialog from './MCPAddDialog';
import MCPDetail from './MCPDetail';
import BatchDistributeDialog from './BatchDistributeDialog';
import { type MCPService } from './types';
import ClawProAdminMCPDetail from './clawProMCP/ClawProAdminMCPDetail';
import {
  CLAWPRO_PLATFORM_MCP_ID,
  CLAWPRO_PLATFORM_MCP_NAME,
  CLAWPRO_PLATFORM_MCP_TAGLINE,
  CLAWPRO_PLATFORM_MCP_VERSION,
  CLAWPRO_PLATFORM_MCP_SERVICE_URL,
  CLAWPRO_PLATFORM_MCP_TRANSPORT,
} from './clawProMCP/constants';
import {
  loadDistributedAgents,
  loadUserTokens,
  saveDistributedAgents,
} from './clawProMCP/store';
import type { DistributedAgent } from './clawProMCP/types';
import { mcpStore } from './mcpStore';
import { projectAssetStore } from '../project-assets/projectAssetStore';
import {
  getSkillDistributionSummary,
  addDistributionRecord,
  updateDistributionRecord,
  createDistributionRecordId,
  type CachedDistributionRecord,
  type SkillDistributionSummary,
} from './distributionCache';

// ── 组件 ────────────────────────────────────────────────
export default function MCPListTab() {
  const [searchQuery, setSearchQuery] = useState('');
  const [mcps, setMCPs] = useState<MCPService[]>(() => mcpStore.getAll());
  const [selectedMCPId, setSelectedMCPId] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'card' | 'list'>('list');
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [distributeMCPId, setDistributeMCPId] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleteMCPId, setDeleteMCPId] = useState<string | null>(null);
  const [distributionSummaries, setDistributionSummaries] = useState<Record<string, SkillDistributionSummary>>({});
  const [distributing, setDistributing] = useState<Record<string, boolean>>({});

  // 订阅 mcpStore：跨模块（含「项目资产管理」联动）变更时刷新列表
  useEffect(() => mcpStore.subscribe(() => setMCPs(mcpStore.getAll())), []);

  const refreshDistributionSummaries = useCallback(() => {
    const summaries: Record<string, SkillDistributionSummary> = {};
    mcps.forEach(m => {
      const summary = getSkillDistributionSummary(m.name);
      if (summary) summaries[m.name] = summary;
    });
    setDistributionSummaries(summaries);
  }, [mcps]);

  useEffect(() => {
    refreshDistributionSummaries();
    const handler = () => refreshDistributionSummaries();
    window.addEventListener('distribution-cache-updated', handler);
    return () => window.removeEventListener('distribution-cache-updated', handler);
  }, [refreshDistributionSummaries]);

  const isDistributing = (mcpName: string) => distributing[mcpName] || distributionSummaries[mcpName]?.hasInProgress || false;

  const filteredMCPs = mcps.filter(m =>
    m.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    (m.displayName || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
    m.description.toLowerCase().includes(searchQuery.toLowerCase())
  );
  const sortedMCPs = [...filteredMCPs].sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime());

  const handleAddMCP = (mcp: MCPService) => {
    mcpStore.add(mcp);
    setMCPs(mcpStore.getAll());
  };

  const handleDistribute = (mcpId: string) => {
    setDistributeMCPId(mcpId);
    setDistributeDialogOpen(true);
  };

  const handleDistributeStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    if (!distributeMCPId) return;

    // ── 特殊路径：ClawPro 平台 MCP 走 clawProMCP store ──
    if (distributeMCPId === CLAWPRO_PLATFORM_MCP_ID) {
      const tokens = loadUserTokens();
      const existing = loadDistributedAgents();
      const existingIds = new Set(existing.map(a => a.agentId));
      const now = new Date().toLocaleString('zh-CN', { hour12: false });

      const newRecords: DistributedAgent[] = selectedInstancesData
        .filter(inst => !existingIds.has(inst.id))
        .map(inst => {
          const ownerId: string = inst.createdBy || inst.creator || 'admin';
          const matchedToken = tokens.find(t => t.userId === ownerId);
          // 找不到 token 时按 createdBy 推断角色：admin 邮箱(暂用名)给 admin，否则 member
          const role = matchedToken?.role ?? (ownerId === 'admin' ? 'admin' : 'member');
          return {
            agentId: inst.id,
            agentName: inst.name,
            ownerUserId: ownerId,
            ownerUserName: matchedToken?.userName ?? ownerId,
            ownerRole: role,
            injectedTokenMask: matchedToken?.tokenMask ?? '（待生成）',
            source: 'manual' as const,
            distributedAt: now,
          };
        });

      saveDistributedAgents([...existing, ...newRecords]);
      setDistributeDialogOpen(false);
      toast.success(
        newRecords.length === selectedInstanceIds.length
          ? `已下发到 ${newRecords.length} 个 Agent`
          : `已下发到 ${newRecords.length} 个新 Agent（${selectedInstanceIds.length - newRecords.length} 个已存在，已跳过）`,
      );
      return;
    }

    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: distributeMCPId,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'distributing',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as const,
      })),
    };
    addDistributionRecord(newRecord);
    setDistributeDialogOpen(false);
    setDistributing(prev => ({ ...prev, [distributeMCPId]: true }));
    toast.success('已开始下发流程');

    const totalCount = selectedInstanceIds.length;
    let completed = 0;
    const interval = setInterval(() => {
      completed += Math.floor(Math.random() * 3) + 1;
      if (completed >= totalCount) {
        completed = totalCount;
        clearInterval(interval);
        const failedCount = Math.floor(Math.random() * 2);
        const successCount = totalCount - failedCount;
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount,
          failedCount,
          inProgressCount: 0,
          status: failedCount === 0 ? 'success' : 'failed',
          instances: record.instances.map((inst, idx) => ({
            ...inst,
            distributionStatus: idx < successCount ? 'success' as const : 'failed' as const,
          })),
        }));
        setDistributing(prev => ({ ...prev, [distributeMCPId!]: false }));
        toast.success('下发完成');
      } else {
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  };

  const handleDelete = (mcpId: string) => {
    setDeleteMCPId(mcpId);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    if (!deleteMCPId) return;
    const mcpName = mcps.find(m => m.name === deleteMCPId)?.displayName || mcps.find(m => m.name === deleteMCPId)?.name || '';
    mcpStore.remove(deleteMCPId);
    projectAssetStore.onLibraryItemDeleted('enterpriseMcp', deleteMCPId);
    setMCPs(mcpStore.getAll());
    toast.success(`MCP「${mcpName}」已删除`);
    setDeleteDialogOpen(false);
    setDeleteMCPId(null);
  };

  // 为下发对话框找到对应 MCP；对 ClawPro 平台 MCP 这条特殊条目，构造虚拟 MCPService
  const clawProVirtualMCP: MCPService = {
    name: CLAWPRO_PLATFORM_MCP_ID,
    displayName: CLAWPRO_PLATFORM_MCP_NAME,
    description: CLAWPRO_PLATFORM_MCP_TAGLINE,
    version: CLAWPRO_PLATFORM_MCP_VERSION,
    transport: CLAWPRO_PLATFORM_MCP_TRANSPORT,
    configJson: JSON.stringify(
      {
        mcp: {
          servers: {
            [CLAWPRO_PLATFORM_MCP_ID]: {
              url: CLAWPRO_PLATFORM_MCP_SERVICE_URL,
              transport: CLAWPRO_PLATFORM_MCP_TRANSPORT,
              headers: { Authorization: '<auto-injected-per-user>' },
            },
          },
        },
      },
      null,
      2,
    ),
    createdAt: new Date(),
    updatedAt: new Date(),
    scope: 'public',
    groupIds: [],
  };

  const distributeMCP =
    distributeMCPId === CLAWPRO_PLATFORM_MCP_ID
      ? clawProVirtualMCP
      : mcps.find(m => m.name === distributeMCPId);
  const deleteMCP = mcps.find(m => m.name === deleteMCPId);

  // ClawPro 平台 MCP（特殊条目）详情页
  if (selectedMCPId === CLAWPRO_PLATFORM_MCP_ID) {
    return (
      <>
        <ClawProAdminMCPDetail
          onBack={() => setSelectedMCPId(null)}
          onRequestDistribute={() => handleDistribute(CLAWPRO_PLATFORM_MCP_ID)}
        />
        {/* 下发弹窗（不能因 early return 丢失） */}
        {distributeMCPId && (
          <BatchDistributeDialog
            open={!!distributeMCPId}
            onOpenChange={(v) => !v && setDistributeMCPId(null)}
            mcp={distributeMCP}
            instances={MOCK_CLOUD_OPENCLAW_INSTANCES}
            groups={MOCK_GROUPS}
            skillId={distributeMCPId || undefined}
            onDistributionStart={handleDistributeStart}
            distributing={!!distributing[distributeMCPId]}
          />
        )}
      </>
    );
  }

  // 如果选中了 MCP，显示详情页
  if (selectedMCPId) {
    const selectedMCP = mcps.find(m => m.name === selectedMCPId);
    if (selectedMCP) {
      return (
        <MCPDetail
          mcp={selectedMCP}
          onBack={() => setSelectedMCPId(null)}
          onMCPDelete={(mcpId) => {
            mcpStore.remove(mcpId);
            projectAssetStore.onLibraryItemDeleted('enterpriseMcp', mcpId);
            setMCPs(mcpStore.getAll());
            setSelectedMCPId(null);
          }}
        />
      );
    }
  }

  return (
    <div className="space-y-4">
      {/* 工具栏 */}
      <div className="flex items-center justify-between gap-6">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
          <Input
            placeholder="搜索 MCP 名称、标识或描述..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10 bg-white border border-gray-200"
          />
        </div>

        <div className="flex items-center gap-4">
          {/* 视图切换
            * 停服态豁免：卡片/列表视图切换属于查看类操作（不产生变更），
            * 需保持 100% 不透明与正常交互。
            * 原生 <button> 未设置 disabled，"停服前已禁用则延续禁用"
            * 约束通过原生 disabled 属性依然生效（此处无）。 */}
          <div
            className="flex items-center gap-1 border border-gray-200 rounded p-1 bg-white"
            data-billing-exempt
          >
            <button
              onClick={() => setViewMode('card')}
              className={`p-2 rounded transition-colors ${viewMode === 'card' ? 'bg-gray-100 text-gray-900' : 'text-gray-600 hover:bg-gray-50'}`}
              title="卡片视图"
            >
              <Grid3x3 className="w-4 h-4" />
            </button>
            <button
              onClick={() => setViewMode('list')}
              className={`p-2 rounded transition-colors ${viewMode === 'list' ? 'bg-gray-100 text-gray-900' : 'text-gray-600 hover:bg-gray-50'}`}
              title="列表视图"
            >
              <List className="w-4 h-4" />
            </button>
          </div>

          <Button onClick={() => setAddDialogOpen(true)} className="bg-blue-600 hover:bg-blue-700">
            + 新增 MCP
          </Button>
        </div>
      </div>

      {/* 空状态 */}
      {sortedMCPs.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-500">还没有创建任何 MCP 服务</p>
          <Button onClick={() => setAddDialogOpen(true)} className="mt-4">+ 新增 MCP</Button>
        </div>
      )}

      {/* 卡片视图 */}
      {viewMode === 'card' && (
        <div className="grid grid-cols-3 gap-4">
          {/* ── 特殊卡片：ClawPro 平台 MCP（左蓝条+实心盾标+蓝底，无文字标签） ── */}
          <div
            key={CLAWPRO_PLATFORM_MCP_ID}
            onClick={() => setSelectedMCPId(CLAWPRO_PLATFORM_MCP_ID)}
            className="relative rounded-lg border-2 border-blue-200 bg-blue-50/60 p-4 transition-all hover:shadow-md hover:border-blue-300 cursor-pointer flex flex-col overflow-hidden"
          >
            <div className="absolute left-0 top-0 bottom-0 w-[4px] bg-blue-600" />
            <div className="flex items-center gap-2.5 mb-2">
              <div className="w-7 h-7 rounded bg-blue-600 flex items-center justify-center flex-shrink-0">
                <ShieldCheck className="w-4 h-4 text-white" />
              </div>
              <h3 className="font-semibold text-gray-900 flex-1 truncate">
                {CLAWPRO_PLATFORM_MCP_NAME}
              </h3>
              <span className="inline-block px-2.5 py-0.5 bg-gray-100 text-gray-600 text-xs font-medium rounded-full shrink-0">
                v{CLAWPRO_PLATFORM_MCP_VERSION}
              </span>
            </div>
            <p className="text-xs text-gray-400 font-mono mb-1 truncate">{CLAWPRO_PLATFORM_MCP_ID}</p>
            <p
              className="text-sm text-gray-600 line-clamp-2 mb-4 cursor-default"
              style={{ minHeight: '2.5rem' }}
            >
              {CLAWPRO_PLATFORM_MCP_TAGLINE}
            </p>
            <div
              className="flex items-center gap-1 mt-auto"
              onClick={e => e.stopPropagation()}
            >
              <Button
                variant="outline"
                size="sm"
                onClick={() => handleDistribute(CLAWPRO_PLATFORM_MCP_ID)}
                className="h-7 text-xs"
              >
                <Send className="w-3 h-3 mr-1" />
                下发
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setSelectedMCPId(CLAWPRO_PLATFORM_MCP_ID)}
                className="h-7 text-xs"
              >
                <Settings className="w-3 h-3 mr-1" />
                管理
              </Button>
            </div>
          </div>
          {sortedMCPs.map(mcp => {
            const dist = isDistributing(mcp.name);
            return (
              <div key={mcp.name} onClick={() => setSelectedMCPId(mcp.name)} className="rounded-lg border border-gray-200 bg-white p-4 transition-all hover:shadow-md cursor-pointer flex flex-col">
                <div className="flex items-center gap-2 mb-2">
                  <h3 className="font-semibold text-gray-900 flex-1 truncate">{mcp.displayName || mcp.name}</h3>
                  <span className="inline-block px-2.5 py-0.5 bg-gray-100 text-gray-600 text-xs font-medium rounded-full shrink-0">
                    v{mcp.version}
                  </span>
                </div>
                {mcp.displayName && (
                  <p className="text-xs text-gray-400 font-mono mb-1 truncate">{mcp.name}</p>
                )}
                <Tooltip delayDuration={1000}>
                  <TooltipTrigger asChild>
                    <p className="text-sm text-gray-600 line-clamp-2 mb-4 cursor-default" style={{ minHeight: '2.5rem' }}>{mcp.description || '-'}</p>
                  </TooltipTrigger>
                  {mcp.description && mcp.description.length > 40 && (
                    <TooltipContent side="bottom" className="max-w-[320px]">
                      <p className="text-xs whitespace-pre-wrap">{mcp.description}</p>
                    </TooltipContent>
                  )}
                </Tooltip>
                <div className="flex items-center gap-1 mt-auto" onClick={(e) => e.stopPropagation()}>
                  <Button variant="outline" size="sm" onClick={() => handleDistribute(mcp.name)} disabled={dist} className={`h-7 text-xs ${dist ? 'opacity-50 cursor-not-allowed' : ''}`}>
                    <Send className="w-3 h-3 mr-1" />
                    {dist ? '下发中' : '下发'}
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => handleDelete(mcp.name)} className="h-7 text-xs text-red-600 hover:text-red-700 hover:bg-red-50 border-red-200">
                    <Trash2 className="w-3 h-3 mr-1" />删除
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* 列表视图 */}
      {viewMode === 'list' && (
        <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wide" style={{ width: '20%' }}>
                  名称/标识
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 tracking-wide" style={{ width: '15%' }}>状态/下发动态</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wide" style={{ width: '10%' }}>版本号/连接方式</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wide" style={{ width: '30%' }}>描述</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wide" style={{ width: '10%' }}>创建时间</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wide" style={{ width: '15%' }}>操作</th>
              </tr>
            </thead>
            <tbody>
              {/* ── 特殊行：ClawPro 平台 MCP（置顶，蓝条+蓝底，无文字标签） ── */}
              <tr
                key={CLAWPRO_PLATFORM_MCP_ID}
                onClick={() => setSelectedMCPId(CLAWPRO_PLATFORM_MCP_ID)}
                className="border-b border-blue-100 cursor-pointer transition-colors bg-blue-50/60 hover:bg-blue-50/80 relative"
                style={{ boxShadow: 'inset 4px 0 0 0 #2563eb' }}
              >
                <td className="px-4 py-3.5">
                  <div className="flex items-center gap-2.5">
                    <div className="w-7 h-7 rounded bg-blue-600 flex items-center justify-center flex-shrink-0">
                      <ShieldCheck className="w-4 h-4 text-white" />
                    </div>
                    <div>
                      <div className="font-semibold text-gray-900">
                        {CLAWPRO_PLATFORM_MCP_NAME}
                      </div>
                      <div className="text-xs text-gray-400 font-mono mt-0.5">
                        {CLAWPRO_PLATFORM_MCP_ID}
                      </div>
                    </div>
                  </div>
                </td>
                <td className="pl-4 pr-2 py-3.5">
                  <div className="text-sm font-medium text-gray-700">正常</div>
                  <div className="text-xs mt-0.5 text-gray-400">
                    已下发 <strong className="text-gray-700">{loadDistributedAgents().length}</strong> 个 Agent
                  </div>
                </td>
                <td className="px-4 py-3.5">
                  <span className="inline-block px-2.5 py-0.5 bg-gray-100 text-gray-600 text-xs font-medium rounded-full">
                    v{CLAWPRO_PLATFORM_MCP_VERSION}
                  </span>
                  <div className="text-xs text-gray-400 mt-0.5">远程服务</div>
                </td>
                <td className="px-4 py-3.5">
                  <span className="text-sm text-gray-600 block truncate">
                    {CLAWPRO_PLATFORM_MCP_TAGLINE}
                  </span>
                </td>
                <td className="px-4 py-3.5">
                  <span className="text-sm text-gray-400">系统内置</span>
                </td>
                <td className="px-4 py-3.5" onClick={e => e.stopPropagation()}>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDistribute(CLAWPRO_PLATFORM_MCP_ID)}
                      className="h-7 text-xs"
                    >
                      <Send className="w-3 h-3 mr-1" />
                      下发
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setSelectedMCPId(CLAWPRO_PLATFORM_MCP_ID)}
                      className="h-7 text-xs"
                    >
                      <Settings className="w-3 h-3 mr-1" />
                      管理
                    </Button>
                  </div>
                </td>
              </tr>
              {sortedMCPs.map((mcp) => {
                const dist = isDistributing(mcp.name);
                const summary = distributionSummaries[mcp.name];

                const hasDistribution = summary && summary.lastDistributionStatus !== 'not_distributed';
                let statusLine1 = '正常';
                let statusLine2 = '未下发';
                let statusLine1Color = 'text-gray-700';
                let statusLine2Color = 'text-gray-400';
                let statusLine2Bg = '';
                let statusLine2HoverBg = '';
                if (summary) {
                  if (summary.lastDistributionStatus === 'distributing') {
                    statusLine1 = '下发中';
                    statusLine1Color = 'text-blue-600';
                    statusLine2 = `${summary.lastDistributionProgress || 0}%`;
                    statusLine2Color = 'text-blue-600';
                    statusLine2Bg = 'bg-blue-50';
                    statusLine2HoverBg = 'hover:bg-blue-100';
                  } else if (hasDistribution) {
                    statusLine1 = '正常';
                    statusLine1Color = 'text-gray-700';
                    const total = summary.lastDistributionInstanceCount || 0;
                    const success = summary.lastDistributionSuccessCount ?? total;
                    statusLine2 = `已下发(${success}/${total}成功)`;
                    if (success === total) {
                      statusLine2Color = 'text-green-600';
                      statusLine2Bg = 'bg-green-50';
                      statusLine2HoverBg = 'hover:bg-green-100';
                    } else {
                      statusLine2Color = 'text-yellow-600';
                      statusLine2Bg = 'bg-yellow-50';
                      statusLine2HoverBg = 'hover:bg-yellow-100';
                    }
                  }
                }

                return (
                  <tr key={mcp.name} onClick={() => setSelectedMCPId(mcp.name)} className="border-b border-gray-100 hover:bg-gray-50 cursor-pointer transition-colors group">
                    {/* 名称 / 标识 */}
                    <td className="px-4 py-3">
                      <div className="font-medium text-gray-900 truncate">{mcp.displayName || mcp.name}</div>
                      {mcp.displayName && <div className="text-xs text-gray-400 font-mono mt-0.5 truncate">{mcp.name}</div>}
                    </td>
                    {/* 状态/下发动态 */}
                    <td className="pl-4 pr-2 py-3">
                      <div className={`text-sm font-medium ${statusLine1Color}`}>{statusLine1}</div>
                      <div
                        className={hasDistribution
                          ? `inline-flex items-center px-1.5 py-0.5 mt-0.5 rounded-full text-xs font-medium cursor-default transition-colors ${statusLine2Color} ${statusLine2Bg} ${statusLine2HoverBg}`
                          : `text-xs mt-0.5 ${statusLine2Color}`
                        }
                      >
                        {statusLine2}
                      </div>
                    </td>
                    {/* 版本号/连接方式 */}
                    <td className="px-4 py-3">
                      <span className="inline-block px-2.5 py-0.5 bg-gray-100 text-gray-600 text-xs font-medium rounded-full">
                        v{mcp.version}
                      </span>
                      <div className="text-xs text-gray-400 mt-0.5">
                        {mcp.transport === 'stdio' ? '本地命令' : '远程服务'}
                      </div>
                    </td>
                    {/* 描述 */}
                    <td className="px-4 py-3" style={{ overflow: 'hidden' }}>
                      <Tooltip delayDuration={1000}>
                        <TooltipTrigger asChild>
                          <span
                            className="text-sm text-gray-600 cursor-default block"
                            style={{
                              display: '-webkit-box',
                              WebkitLineClamp: 2,
                              WebkitBoxOrient: 'vertical',
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              wordBreak: 'break-all',
                            }}
                          >{mcp.description || '-'}</span>
                        </TooltipTrigger>
                        {mcp.description && mcp.description.length > 40 && (
                          <TooltipContent side="bottom" className="max-w-[400px]">
                            <p className="text-xs whitespace-pre-wrap">{mcp.description}</p>
                          </TooltipContent>
                        )}
                      </Tooltip>
                    </td>
                    {/* 创建时间 */}
                    <td className="px-4 py-3">
                      <span className="text-sm text-gray-500">
                        {mcp.createdAt.toLocaleDateString('zh-CN')}
                      </span>
                    </td>
                    {/* 操作 */}
                    <td className="px-4 py-3" onClick={(e) => e.stopPropagation()}>
                      <div className="flex items-center gap-1">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleDistribute(mcp.name)}
                          disabled={dist}
                          className={`h-7 text-xs min-w-[62px] ${dist ? 'opacity-50 cursor-not-allowed' : ''}`}
                        >
                          {dist ? <Loader className="w-3 h-3 mr-1 animate-spin" /> : <Send className="w-3 h-3 mr-1" />}
                          {dist ? '下发中' : '下发'}
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleDelete(mcp.name)}
                          className="h-7 text-xs text-red-600 hover:text-red-700 hover:bg-red-50 border-red-200"
                        >
                          <Trash2 className="w-3 h-3 mr-1" />
                          删除
                        </Button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* 新增 MCP 弹窗 */}
      <MCPAddDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        onConfirm={handleAddMCP}
        existingNames={mcps.map(m => m.name)}
      />

      {/* 下发弹窗 */}
      <BatchDistributeDialog
        open={distributeDialogOpen}
        onOpenChange={setDistributeDialogOpen}
        skillId={distributeMCPId || undefined}
        skillName={distributeMCP?.displayName || distributeMCP?.name}
        onDistributionStart={handleDistributeStart}
        title="批量下发 MCP 配置"
        showScopeFilter
        instances={MOCK_CLOUD_OPENCLAW_INSTANCES}
        groups={MOCK_GROUPS}
        singleStatusFilter
        showConfirmDialog
        descriptionNode={
          <>
            将 <span className="font-semibold">「{distributeMCP?.displayName || distributeMCP?.name || ''}」</span> 部署至所选实例。
            <br />
            仅支持 <span className="font-medium">运行中</span> 的 <span className="font-medium">OpenClaw 及本地 Agent</span> 实例。默认展示{' '}
            <span className="font-medium">未下发</span> 和 <span className="font-medium">下发失败</span> 的实例，已下发成功实例可通过状态筛选查看。
            <br />
            <span className="font-medium">26.3.28 以下版本</span>暂不支持 MCP 服务。
          </>
        }
      />

      {/* 删除确认弹窗 */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-[400px]">
          <DialogHeader>
            <DialogTitle>删除 MCP</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-gray-600">
            确定要删除 MCP「<span className="font-medium text-gray-900">{deleteMCP?.displayName || deleteMCP?.name}</span>」吗？此操作无法撤销。
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>取消</Button>
            <Button variant="destructive" onClick={handleConfirmDelete}>确认删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
