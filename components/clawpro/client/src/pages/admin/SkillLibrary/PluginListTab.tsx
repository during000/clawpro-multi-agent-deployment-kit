/**
 * PluginListTab - 企业插件库列表
 * 复用企业技能库的列表 UI，无分类筛选，操作只有下发和删除
 */
import { useState, useEffect, useCallback, useMemo } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import { Input } from '@/components/ui/input';
import { SurfaceCard } from '@/components/ui/Surface';
import { StatusTag } from '@/components/ui/status-tag';
import {
  Table,
  TableActionCell,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Search, Grid3x3, List, Send, Trash2, PackageX, RefreshCw } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { MoreActionsDropdown } from '@/components/ui/more-actions-dropdown';
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
import { BodyText, BodyMedium, MetaText, CardTitle } from '@/components/ui/Typography';
import { ScopeSelect } from '@/components/ScopeSelect';
import { MOCK_GROUPS, MOCK_OPENCLAW_INSTANCES, MOCK_PROJECT_GROUPS } from './mockData';
import PluginUploadDialog, { type Plugin } from './PluginUploadDialog';
import PluginUpdateDialog from './PluginUpdateDialog';
import PluginDetail from './PluginDetail';
import BatchDistributeDialog from './BatchDistributeDialog';
import BatchDeleteDialog from './BatchDeleteDialog';
import { pluginStore } from './pluginStore';
import { projectAssetStore } from '../project-assets/projectAssetStore';
import {
  getSkillDistributionSummary,
  getCurrentPluginInstalledInstances,
  addDistributionRecord,
  updateDistributionRecord,
  createDistributionRecordId,
  type CachedDistributionRecord,
  type SkillDistributionSummary,
} from './distributionCache';

// Mock 数据与缓存统一由 pluginStore 提供（localStorage + CustomEvent 共享）

export default function PluginListTab() {
  const [searchQuery, setSearchQuery] = useState('');
  const [plugins, setPlugins] = useState<Plugin[]>(() => pluginStore.getAll());
  const [selectedPluginId, setSelectedPluginId] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'card' | 'list'>('list');
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false);
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [distributePluginId, setDistributePluginId] = useState<string | null>(null);
  const [updateDialogOpen, setUpdateDialogOpen] = useState(false);
  const [updatePluginId, setUpdatePluginId] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletePluginId, setDeletePluginId] = useState<string | null>(null);
  const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
  const [uninstallPluginId, setUninstallPluginId] = useState<string | null>(null);
  const [distributionSummaries, setDistributionSummaries] = useState<Record<string, SkillDistributionSummary>>({});
  const [distributing, setDistributing] = useState<Record<string, boolean>>({});

  // 订阅 pluginStore：跨模块（含「项目资产管理」联动）变更时刷新列表
  useEffect(() => pluginStore.subscribe(() => setPlugins(pluginStore.getAll())), []);

  const refreshDistributionSummaries = useCallback(() => {
    const summaries: Record<string, SkillDistributionSummary> = {};
    plugins.forEach(p => {
      const summary = getSkillDistributionSummary(p.id);
      if (summary) summaries[p.id] = summary;
    });
    setDistributionSummaries(summaries);
  }, [plugins]);

  useEffect(() => {
    refreshDistributionSummaries();
    const handler = () => refreshDistributionSummaries();
    window.addEventListener('distribution-cache-updated', handler);
    return () => window.removeEventListener('distribution-cache-updated', handler);
  }, [refreshDistributionSummaries]);

  const isDistributing = (pluginId: string) => distributing[pluginId] || distributionSummaries[pluginId]?.hasInProgress || false;
  const isPluginIdentifierMissing = (plugin?: Plugin) => !plugin?.id || !plugin.slug?.trim();
  const isUpdateDisabled = (plugin: Plugin) => isDistributing(plugin.id) || isPluginIdentifierMissing(plugin);
  const getUninstallableCount = (plugin: Plugin) =>
    getCurrentPluginInstalledInstances(plugin.id, plugin.version, MOCK_OPENCLAW_INSTANCES).length;
  const getUninstallDisabledReason = (plugin: Plugin) => {
    if (isDistributing(plugin.id)) return '有任务进行中';
    if (getUninstallableCount(plugin) === 0) return '暂无可卸载实例';
    return '';
  };

  const filteredPlugins = plugins.filter(p =>
    p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    p.description.toLowerCase().includes(searchQuery.toLowerCase())
  );
  const sortedPlugins = [...filteredPlugins].sort((a, b) => b.uploadTime.getTime() - a.uploadTime.getTime());

  const handleUploadPlugin = (plugin: Plugin) => {
    pluginStore.add(plugin);
    setPlugins(pluginStore.getAll());
  };

  const handleDistribute = (pluginId: string) => {
    setDistributePluginId(pluginId);
    setDistributeDialogOpen(true);
  };

  const handleDistributeStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    if (!distributePluginId) return;
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: distributePluginId,
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
    setDistributing(prev => ({ ...prev, [distributePluginId]: true }));
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
        setDistributing(prev => ({ ...prev, [distributePluginId!]: false }));
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

  const handleUpdate = (pluginId: string) => {
    const targetPlugin = plugins.find(p => p.id === pluginId);
    if (!targetPlugin || isUpdateDisabled(targetPlugin)) return;
    setUpdatePluginId(pluginId);
    setUpdateDialogOpen(true);
  };

  const handlePluginUpdated = (updatedPlugin: Plugin) => {
    pluginStore.update(updatedPlugin.id, () => updatedPlugin);
    setPlugins(pluginStore.getAll());
    setUpdateDialogOpen(false);
    setUpdatePluginId(null);
  };

  const handleDelete = (pluginId: string) => {
    setDeletePluginId(pluginId);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    if (!deletePluginId) return;
    const pluginName = plugins.find(p => p.id === deletePluginId)?.name || '';
    pluginStore.remove(deletePluginId);
    projectAssetStore.onLibraryItemDeleted('enterprisePlugin', deletePluginId);
    setPlugins(pluginStore.getAll());
    toast.success(`插件「${pluginName}」已删除`);
    setDeleteDialogOpen(false);
    setDeletePluginId(null);
  };

  const handleUninstall = (pluginId: string) => {
    const plugin = plugins.find(p => p.id === pluginId);
    if (!plugin || getUninstallDisabledReason(plugin)) return;
    setUninstallPluginId(pluginId);
    setUninstallDialogOpen(true);
  };

  const handleUninstallStart = (selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    if (!uninstallPluginId) return;
    const recordId = createDistributionRecordId();
    const newRecord: CachedDistributionRecord = {
      id: recordId,
      skillId: uninstallPluginId,
      timestamp: new Date().toISOString(),
      totalCount: selectedInstanceIds.length,
      successCount: 0,
      failedCount: 0,
      inProgressCount: selectedInstanceIds.length,
      status: 'deleting',
      type: 'delete',
      operator: 'admin',
      instances: selectedInstancesData.map(inst => ({
        id: inst.id,
        name: inst.name,
        createdBy: inst.createdBy || 'admin',
        distributionStatus: 'distributing' as const,
      })),
    };
    addDistributionRecord(newRecord);
    setUninstallDialogOpen(false);
    setDistributing(prev => ({ ...prev, [uninstallPluginId]: true }));
    toast.success('已开始卸载流程');

    const totalCount = selectedInstanceIds.length;
    let completed = 0;
    const failReasons = ['实例离线', '权限不足', '插件被占用', '网络超时', '实例已停止'];
    const interval = setInterval(() => {
      completed += Math.floor(Math.random() * 3) + 1;
      if (completed >= totalCount) {
        completed = totalCount;
        clearInterval(interval);
        const results = Array.from({ length: totalCount }, () => Math.random() < 0.9);
        const successCount = results.filter(Boolean).length;
        const failedCount = totalCount - successCount;
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount,
          failedCount,
          inProgressCount: 0,
          status: 'success',
          instances: record.instances.map((inst, idx) => ({
            ...inst,
            distributionStatus: (results[idx] ? 'success' : 'failed') as 'success' | 'failed',
            failReason: results[idx] ? undefined : failReasons[Math.floor(Math.random() * failReasons.length)],
          })),
        }));
        setDistributing(prev => ({ ...prev, [uninstallPluginId!]: false }));
        toast.success('卸载完成');
      } else {
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      }
    }, 800);
  };

  const distributePlugin = plugins.find(p => p.id === distributePluginId);
  const updatePlugin = plugins.find(p => p.id === updatePluginId);
  const deletePlugin = plugins.find(p => p.id === deletePluginId);
  const uninstallPlugin = plugins.find(p => p.id === uninstallPluginId);
  const uninstallPluginVersion = uninstallPlugin?.version;
  const distributedInstancesForUninstall = useMemo(() => {
    if (!uninstallPluginId || !uninstallPluginVersion) return [];

    return getCurrentPluginInstalledInstances(uninstallPluginId, uninstallPluginVersion, MOCK_OPENCLAW_INSTANCES)
      .map(inst => {
        const groupName = inst.groupIds?.[0]
          ? MOCK_GROUPS.find(g => g.id === inst.groupIds[0])?.name
          : undefined;

        return {
          id: inst.id,
          name: inst.name,
          createdBy: inst.createdBy || 'admin',
          groupName: groupName || '全部用户',
          distributedVersion: inst.distributedVersion || uninstallPluginVersion,
          deleteStatus: inst.failReason ? 'delete_failed' as const : 'not_deleted' as const,
          deleteFailReason: inst.failReason,
        };
      });
  }, [uninstallPluginId, uninstallPluginVersion, distributionSummaries]);

  // 如果选中了插件，显示详情页
  if (selectedPluginId) {
    const selectedPlugin = plugins.find(p => p.id === selectedPluginId);
    if (selectedPlugin) {
      return (
        <PluginDetail
          plugin={selectedPlugin}
          onBack={() => setSelectedPluginId(null)}
          onPluginDelete={(pluginId) => {
            pluginStore.remove(pluginId);
            projectAssetStore.onLibraryItemDeleted('enterprisePlugin', pluginId);
            setPlugins(pluginStore.getAll());
            setSelectedPluginId(null);
          }}
        />
      );
    }
  }

  return (
    <div className="space-y-4">
      {/* 工具栏 */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-[#A3A3A3] w-4 h-4" />
          <Input
            placeholder="搜索插件名称、标识或描述..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10 bg-white border border-gray-200"
          />
        </div>

        <div className="flex items-center gap-2">
          {/* 视图切换
            * 停服态豁免：卡片/列表视图切换属于查看类操作（不产生变更），
            * 需保持 100% 不透明与正常交互。
            * SegmentOption 自身未设置 disabled，"停服前已禁用则延续禁用"
            * 约束通过组件 disabled 属性依然生效（此处无）。 */}
          <SegmentGroup data-billing-exempt>
            <SegmentOption active={viewMode === 'card'} onClick={() => setViewMode('card')} title="卡片视图">
              <Grid3x3 className="w-4 h-4" />
            </SegmentOption>
            <SegmentOption active={viewMode === 'list'} onClick={() => setViewMode('list')} title="列表视图">
              <List className="w-4 h-4" />
            </SegmentOption>
          </SegmentGroup>

          <Button variant="claw-primary" size="claw-sm" onClick={() => setUploadDialogOpen(true)}>
            + 发布插件
          </Button>
        </div>
      </div>

      {/* 空状态 */}
      {sortedPlugins.length === 0 && (
        <div className="text-center py-12">
          <BodyText as="p" tone="muted">还没有发布任何插件</BodyText>
          <Button onClick={() => setUploadDialogOpen(true)} className="mt-4">+ 发布插件</Button>
        </div>
      )}

      {/* 卡片视图 */}
      {viewMode === 'card' && sortedPlugins.length > 0 && (
        <div className="grid grid-cols-3 gap-4">
          {sortedPlugins.map(plugin => {
            const dist = isDistributing(plugin.id);
            return (
              <div
                key={plugin.id}
                onClick={() => setSelectedPluginId(plugin.id)}
                className="relative flex flex-col overflow-hidden rounded-[4px] border border-gray-200 bg-white p-4 cursor-pointer transition-all hover:border-[#355EF1] group"
              >
                <div className="flex items-center gap-2 mb-2">
                  <CardTitle as="h3" tone="primary" className="flex-1 truncate font-semibold group-hover:text-[var(--text-brand)] transition-colors">{plugin.name}</CardTitle>
                  <StatusTag mode="fill" variant="gray" className="shrink-0">v{plugin.version}</StatusTag>
                </div>
                <Tooltip delayDuration={1000}>
                  <TooltipTrigger asChild>
                    <BodyText as="p" tone="muted" className="line-clamp-2 mb-4 cursor-default" style={{ minHeight: '2.5rem' }}>{plugin.description || '-'}</BodyText>
                  </TooltipTrigger>
                  {plugin.description && plugin.description.length > 40 && (
                    <TooltipContent side="bottom" className="max-w-[320px]">
                      <MetaText as="p" tone="inherit" className="whitespace-pre-wrap">{plugin.description}</MetaText>
                    </TooltipContent>
                  )}
                </Tooltip>
                <div className="flex items-center gap-2 pt-3 mt-auto border-t border-[#F5F5F5]" onClick={(e) => e.stopPropagation()}>
                  <Button variant="claw-outline" size="sm" onClick={() => handleDistribute(plugin.id)} disabled={dist} className="h-8">
                    <Send className="size-3.5" />
                    {dist ? '下发中' : '下发'}
                  </Button>
                  <Button
                    variant="claw-outline"
                    size="sm"
                    onClick={() => handleUpdate(plugin.id)}
                    disabled={isUpdateDisabled(plugin)}
                    className="h-8"
                  >
                    <RefreshCw className="size-3.5" />
                    更新
                  </Button>
                  <div className="ml-auto">
                    <MoreActionsDropdown
                      triggerType="icon"
                      align="end"
                      items={[
                        {
                          label: "卸载",
                          icon: PackageX,
                          onClick: () => handleUninstall(plugin.id),
                          disabled: !!getUninstallDisabledReason(plugin),
                        },
                        {
                          label: "删除",
                          icon: Trash2,
                          onClick: () => handleDelete(plugin.id),
                          variant: "destructive" as const,
                        },
                      ]}
                    />
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* 列表视图 */}
      {viewMode === 'list' && sortedPlugins.length > 0 && (
        <SurfaceCard className="overflow-hidden">
          <Table variant="white" scrollX={1390}>
            <TableHeader>
              <TableRow>
                <TableHead style={{ width: 240, minWidth: 240 }}>名称/SLUG</TableHead>
                <TableHead style={{ width: 100, minWidth: 100 }}>状态</TableHead>
                <TableHead style={{ width: 120, minWidth: 120 }}>下发</TableHead>
                <TableHead style={{ width: 104, minWidth: 104 }}>版本号</TableHead>
                <TableHead style={{ width: 360, minWidth: 360 }}>描述</TableHead>
                <TableHead style={{ width: 190, minWidth: 190 }}>应用范围</TableHead>
                <TableHead style={{ width: 140, minWidth: 140 }}>发布时间</TableHead>
                <TableHead fixed="right" style={{ width: 128, minWidth: 128, maxWidth: 128 }}>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sortedPlugins.map((plugin) => {
                const dist = isDistributing(plugin.id);
                const summary = distributionSummaries[plugin.id];
                const hasDistribution = summary && summary.lastDistributionStatus !== 'not_distributed';
                const isDistributionRunning = summary?.lastDistributionStatus === 'distributing';
                const total = summary?.lastDistributionInstanceCount || 0;
                const success = summary?.lastDistributionSuccessCount ?? total;
                const statusLabel = isDistributionRunning ? '下发中' : '正常';
                const statusVariant = isDistributionRunning ? 'blue' : 'green';
                const distributionLabel = isDistributionRunning
                  ? `${summary?.lastDistributionProgress || 0}%`
                  : hasDistribution
                    ? `已下发 ${success}/${total}`
                    : '未下发';
                const distributionVariant = isDistributionRunning ? 'blue' : success === total ? 'green' : 'red';

                return (
                  <TableRow key={plugin.id} onClick={() => setSelectedPluginId(plugin.id)} className="cursor-pointer">
                    <TableCell>
                      <div className="min-w-0 space-y-1">
                        <BodyMedium as="p" tone="primary" className="truncate">{plugin.name}</BodyMedium>
                        <MetaText as="p" tone="weak" className="truncate font-mono">{plugin.slug}</MetaText>
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusTag mode="text" variant={statusVariant}>{statusLabel}</StatusTag>
                    </TableCell>
                    <TableCell>
                      {hasDistribution ? (
                        <BodyText as="span" tone="secondary">{distributionLabel}</BodyText>
                      ) : (
                        <BodyText as="span" tone="weak">{distributionLabel}</BodyText>
                      )}
                    </TableCell>
                    <TableCell>
                      <BodyText as="span" tone="secondary">v{plugin.version}</BodyText>
                    </TableCell>
                    <TableCell className="whitespace-normal" style={{ overflow: 'hidden' }}>
                      <Tooltip delayDuration={1000}>
                        <TooltipTrigger asChild>
                          <BodyText as="span" tone="secondary" className="line-clamp-2 cursor-default break-all">
                            {plugin.description || '-'}
                          </BodyText>
                        </TooltipTrigger>
                        {plugin.description && plugin.description.length > 40 && (
                          <TooltipContent side="bottom" className="max-w-[400px]">
                            <MetaText as="p" tone="inherit" className="whitespace-pre-wrap">{plugin.description}</MetaText>
                          </TooltipContent>
                        )}
                      </Tooltip>
                    </TableCell>
                    <TableCell className="" onClick={(e) => e.stopPropagation()}>
                      <ScopeSelect
                        scope={(!plugin.scope || plugin.scope === 'public' || !plugin.groupIds || plugin.groupIds.length === 0) ? 'all' : 'groups'}
                        selectedGroupIds={plugin.groupIds || []}
                        groups={MOCK_GROUPS}
                        projects={MOCK_PROJECT_GROUPS}
                        maxVisibleBadges={3}
                        onConfirm={(scope, groupIds) => {
                          pluginStore.update(plugin.id, prev => ({
                            ...prev,
                            scope: scope === 'all' ? 'public' : 'private',
                            groupIds,
                          }));
                          setPlugins(pluginStore.getAll());
                          toast.success('应用范围修改成功');
                        }}
                      />
                    </TableCell>
                    <TableCell>
                      <BodyText as="span" tone="secondary" className="tabular-nums">
                        {plugin.uploadTime.toLocaleDateString('zh-CN')}
                      </BodyText>
                    </TableCell>
                    <TableActionCell fixed="right" onClick={(e) => e.stopPropagation()}>
                      <Button
                        variant="link"
                        size="sm"
                        onClick={() => handleDistribute(plugin.id)}
                        disabled={dist}
                      >
                        {dist ? '下发中' : '下发'}
                      </Button>
                      <Button
                        variant="link"
                        size="sm"
                        onClick={() => handleUpdate(plugin.id)}
                        disabled={isUpdateDisabled(plugin)}
                      >
                        更新
                      </Button>
                      <MoreActionsDropdown
                        triggerType="text"
                        align="end"
                        items={[
                          {
                            label: "卸载",
                            icon: PackageX,
                            onClick: () => handleUninstall(plugin.id),
                            disabled: !!getUninstallDisabledReason(plugin),
                          },
                          {
                            label: "删除",
                            icon: Trash2,
                            onClick: () => handleDelete(plugin.id),
                            variant: "destructive" as const,
                          },
                        ]}
                      />
                    </TableActionCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </SurfaceCard>
      )}

      {/* 发布插件弹窗 */}
      <PluginUploadDialog
        open={uploadDialogOpen}
        onOpenChange={setUploadDialogOpen}
        onConfirm={handleUploadPlugin}
        existingSlugs={plugins.map(p => p.slug)}
      />

      {/* 下发弹窗 */}
      <BatchDistributeDialog
        open={distributeDialogOpen}
        onOpenChange={setDistributeDialogOpen}
        skillId={distributePluginId || undefined}
        skillName={distributePlugin?.name}
        skillVersion={distributePlugin?.version}
        skillScope={distributePlugin?.scope}
        skillGroupIds={distributePlugin?.groupIds}
        onDistributionStart={handleDistributeStart}
        title="批量下发插件"
        showScopeFilter
        instances={MOCK_OPENCLAW_INSTANCES}
        groups={MOCK_GROUPS}
        showAgentType
      />

      {/* 更新插件弹窗 */}
      {updatePlugin && (
        <PluginUpdateDialog
          open={updateDialogOpen}
          onOpenChange={(open) => {
            setUpdateDialogOpen(open);
            if (!open) setUpdatePluginId(null);
          }}
          plugin={updatePlugin}
          onConfirm={handlePluginUpdated}
          defaultSecurityScan={false}
          onDefaultSecurityScanChange={() => {}}
          securityServiceActive={true}
        />
      )}

      {/* 批量卸载弹窗 */}
      {uninstallPlugin && (
        <BatchDeleteDialog
          open={uninstallDialogOpen}
          onOpenChange={(open) => {
            setUninstallDialogOpen(open);
            if (!open) setUninstallPluginId(null);
          }}
          skillName={uninstallPlugin.name}
          skillVersion={uninstallPlugin.version}
          distributedInstances={distributedInstancesForUninstall}
          groups={MOCK_GROUPS}
          onDeleteStart={handleUninstallStart}
          resourceLabel="插件"
          warningText="卸载成功后，该插件在对应实例上恢复为“未下发”状态。"
          emptyText="暂无可卸载实例"
        />
      )}

      {/* 删除确认弹窗 - 警示弹窗 */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[#0A0A0A]">删除插件</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <div className="space-y-1">
              <BodyText as="p" tone="primary">
                确定要删除插件「<BodyMedium as="span" tone="primary">{deletePlugin?.name}</BodyMedium>」吗？
              </BodyText>
              <BodyText as="p" tone="danger">此操作不可撤销。</BodyText>
            </div>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmDelete}>确认删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
