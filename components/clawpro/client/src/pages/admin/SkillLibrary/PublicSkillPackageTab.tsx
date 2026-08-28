/**
 * 公共技能包 Tab
 *
 * 二级 Tab「公共技能包」的内容：浏览来自 SkillHub 的技能包（一组 Skill 的场景化组合）
 *
 * 数据来源：SkillHub 公共 API 数据快照（publicSkillPackageDataSnapshot.ts）
 *
 * 设计原则（与 PublicSkillTab 完全一致 / 走查后规范）：
 * 1. 公共技能包只是前端展示层概念，本质是多个公共技能的组合模板
 * 2. 不直接安装、不直接执行，只用于浏览、收藏，以及在角色设定中被展开为多个公共技能
 * 3. 视觉规范：rounded-[4px] 卡片、Typography 体系、FilterChipGroup 分类、Pagination 分页
 * 4. 卡片极简：标题 + 描述 + Skill chip 列表（前 2 个 + N） + 右下角收藏按钮
 * 5. 详情页：信息头卡 + 技能模块 + 工作流（MDXRenderer 渲染 SkillHub 真实 markdown）
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import { Search, RefreshCw, Puzzle, PackagePlus, Send, PackageX } from 'lucide-react';
import { FavoriteButton } from '@/components/ui/favorite-button';
import { toast } from 'sonner';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Pagination } from '@/components/ui/pagination';
import { FilterChipGroup } from '@/components/ui/filter-chip';
import { BackButton } from '@/components/ui/back-button';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { SegmentedTabs } from '@/components/ui/segmented-tabs';
import MDXRenderer from '@/components/MDXRenderer';
import { Empty, EmptyHeader, EmptyDescription, EmptyMedia } from '@/components/ui/empty';
import AddToPackageDialog from './AddToPackageDialog';
import { MoreActionsDropdown } from '@/components/ui/more-actions-dropdown';
import PublicBatchDistributeDialog from './PublicBatchDistributeDialog';
import PublicBatchDeleteDialog from './PublicBatchDeleteDialog';
import {
  TenantDocTitle,
  SectionTitle,
  CardTitle,
  BodyText,
  BodyMedium,
  MetaText,
  MetaMedium,
  HelperText,
} from '@/components/ui/Typography';
import { PUBLIC_SKILLS, type PublicSkill } from './publicSkillMockData';
import {
  PUBLIC_SKILL_PACKAGES,
  PUBLIC_SKILL_PACKAGE_CATEGORIES,
  type PublicSkillPackage,
  type PackageSkillRef,
} from './publicSkillPackageMockData';
import { MOCK_OPENCLAW_INSTANCES } from './mockData';
import {
  addDistributionRecord,
  createDistributionRecordId,
  getAllDistributionRecords,
  getCurrentSkillInstalledInstances,
  getSkillDistributionSummary,
  type CachedDistributionRecord,
  type SkillDistributionSummary,
  updateDistributionRecord,
} from './distributionCache';
import { type DistributionStatus, DISTRIBUTION_STATUS_MAP } from './types';

// ─── 子组件：Skill chip（identify 区块用） ───────────────────────────────────
// 规范：
// - 卡片场景（size='sm'）chip 固定最大宽度，超出用 ellipsis 截断，避免破坏卡片高度
// - 详情页场景（size='md'）chip 宽度自适应，完整展示 slug

function SkillChip({ skill, size = 'sm' }: { skill: PackageSkillRef; size?: 'sm' | 'md' }) {
  const padding = size === 'md' ? 'px-2.5 py-1' : 'px-2 py-0.5';
  const maxWidth = size === 'md' ? '' : 'max-w-[120px]';
  return (
    <span
      className={`inline-flex items-center gap-1 ${padding} ${maxWidth} rounded-[4px] bg-gray-50 border border-gray-200 overflow-hidden`}
      title={skill.slug}
    >
      <Puzzle className="w-3 h-3 text-[var(--text-weak)] flex-shrink-0" />
      {size === 'md' ? (
        <MetaMedium as="span" tone="secondary" className="truncate min-w-0">
          {skill.name}
        </MetaMedium>
      ) : (
        <span className="truncate min-w-0 text-[11px] font-medium text-[var(--text-secondary)]">
          {skill.name}
        </span>
      )}
    </span>
  );
}

// ─── 子组件：技能包卡片 ─────────────────────────────────────────────────────

interface PackageCardProps {
  pkg: PublicSkillPackage;
  isFavorited: boolean;
  onFavorite: (e: React.MouseEvent) => void;
  onAddToPackage: (e: React.MouseEvent) => void;
  onDistribute: () => void;
  onUninstall: () => void;
  disableActions?: boolean;
  uninstallDisabled?: boolean;
  onClick: () => void;
}

function PackageCard({
  pkg,
  isFavorited,
  onFavorite,
  onAddToPackage,
  onDistribute,
  onUninstall,
  disableActions = false,
  uninstallDisabled = false,
  onClick,
}: PackageCardProps) {
  // 卡片只展示前 2 个 Skill chip，超出折叠为 +N
  const VISIBLE_SKILLS = 2;
  const visibleSkills = pkg.skills.slice(0, VISIBLE_SKILLS);
  const overflowCount = pkg.skills.length - visibleSkills.length;

  return (
    <div
      className="relative flex flex-col overflow-hidden rounded-[4px] border border-gray-200 bg-white cursor-pointer transition-all hover:border-[#355EF1] group"
      onClick={onClick}
    >
      <div className="p-4 flex flex-col flex-1">
        {/* 标题 —— 单行省略，固定高度 */}
        <CardTitle
          as="h3"
          tone="primary"
          className="font-semibold group-hover:text-[var(--text-brand)] transition-colors leading-tight truncate mb-1"
          title={pkg.name}
        >
          {pkg.name}
        </CardTitle>

        {/* 描述 —— 固定两行，line-clamp 截断 */}
        <MetaText
          as="p"
          className="line-clamp-2 leading-relaxed"
          style={{ minHeight: '2.5rem' }}
        >
          {pkg.description}
        </MetaText>

        {/* Skill chip 列表（最多 2 个 + N）—— 单行不换行，chip 自身限宽截断保证整体定高 */}
        <div className="flex items-center gap-1.5 mt-3 flex-nowrap overflow-hidden">
          {visibleSkills.map((s) => (
            <SkillChip key={s.slug} skill={s} />
          ))}
          {overflowCount > 0 && (
            <MetaMedium as="span" tone="weak" className="flex-shrink-0 text-[11px]">
              +{overflowCount}
            </MetaMedium>
          )}
        </div>

        {/* 底部：右下角操作区（加入初始技能包 + 收藏） */}
        <div className="flex items-center justify-end gap-1 mt-3">
          <button
            onClick={onAddToPackage}
            className="w-7 h-7 rounded-[4px] flex items-center justify-center text-[var(--text-weak)] transition-colors hover:text-[var(--text-brand)] hover:bg-[var(--bg-brand-selected)]"
            title="加入初始技能包"
            aria-label="加入初始技能包"
          >
            <PackagePlus className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={onFavorite}
            className={`w-7 h-7 rounded-[4px] flex items-center justify-center transition-colors ${
              isFavorited
                ? 'text-[var(--text-danger)] bg-red-50 hover:bg-red-100'
                : 'text-[var(--text-weak)] hover:text-[var(--text-danger)] hover:bg-red-50'
            }`}
            title={isFavorited ? '取消收藏' : '添加到我的收藏'}
            aria-label={isFavorited ? '取消收藏' : '添加到我的收藏'}
          >
            <svg className={`w-3.5 h-3.5 ${isFavorited ? 'fill-current' : ''}`} viewBox="0 0 24 24" fill={isFavorited ? 'currentColor' : 'none'} stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z" />
            </svg>
          </button>
          <MoreActionsDropdown
            triggerType="icon"
            align="end"
            stopPropagation
            disabled={disableActions && uninstallDisabled}
            items={[
              {
                label: '下发',
                icon: Send,
                onClick: onDistribute,
                disabled: disableActions,
              },
              {
                label: '卸载',
                icon: PackageX,
                onClick: onUninstall,
                disabled: disableActions || uninstallDisabled,
              },
            ]}
          />
        </div>
      </div>
    </div>
  );
}

// ─── 子组件：详情页 ────────────────────────────────────────────────────────

interface PackageDetailProps {
  pkg: PublicSkillPackage;
  groups: Array<{ id: string; name: string; parentId?: string | null }>;
  isFavorited: boolean;
  onFavorite: () => void;
  onBack: () => void;
}

function PackageDetailView({
  pkg,
  groups,
  isFavorited,
  onFavorite,
  onBack,
}: PackageDetailProps) {
  const [activeTab, setActiveTab] = useState<'files' | 'distribution' | 'uninstall'>('files');
  const [distributionRecords, setDistributionRecords] = useState<CachedDistributionRecord[]>([]);
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
  const [activeDistributionId, setActiveDistributionId] = useState<string | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [statusFilter, setStatusFilter] = useState<'all' | DistributionStatus>('all');
  const [detailSearchQuery, setDetailSearchQuery] = useState('');
  const categoryName = useMemo(
    () => PUBLIC_SKILL_PACKAGE_CATEGORIES.find((c) => c.id === pkg.category)?.name ?? '',
    [pkg.category],
  );
  const skillResourceId = useMemo(() => getPublicPackageResourceId(pkg.id), [pkg.id]);

  const refreshRecords = useCallback(() => {
    const cachedRecords = getAllDistributionRecords().filter((record) => record.skillId === skillResourceId);
    setDistributionRecords(mergeFixedPackageRecords(pkg.id, cachedRecords));
  }, [pkg.id, skillResourceId]);

  useEffect(() => {
    refreshRecords();
    const handleCacheUpdated = () => refreshRecords();
    window.addEventListener('distribution-cache-updated', handleCacheUpdated);
    return () => window.removeEventListener('distribution-cache-updated', handleCacheUpdated);
  }, [refreshRecords]);

  const distributionOnlyRecords = useMemo(
    () => distributionRecords.filter((record) => (record.type || 'distribute') === 'distribute'),
    [distributionRecords],
  );
  const deleteOnlyRecords = useMemo(
    () => distributionRecords.filter((record) => record.type === 'delete'),
    [distributionRecords],
  );
  const hasInProgress = distributionRecords.some((record) => record.status === 'distributing' || record.status === 'deleting');
  const hasDeleting = distributionRecords.some((record) => record.status === 'deleting');
  const hasDistributing = distributionRecords.some((record) => record.status === 'distributing');
  const distributeInstances = useMemo(
    () => getInstancesWithPackageDistributionStatus(pkg, MOCK_OPENCLAW_INSTANCES),
    [distributionRecords, pkg],
  );
  const distributedInstancesForUninstall = useMemo(() => {
    return getPackageInstancesForUninstall(pkg, groups);
  }, [distributionRecords, groups, pkg]);
  const activeDistribution = useMemo(
    () => distributionRecords.find((record) => record.id === activeDistributionId) ?? null,
    [activeDistributionId, distributionRecords],
  );
  const filteredInstances = useMemo(() => {
    if (!activeDistribution) return [];

    return activeDistribution.instances.filter((instance) => {
      const matchesStatus = statusFilter === 'all' || instance.distributionStatus === statusFilter;
      const searchLower = detailSearchQuery.trim().toLowerCase();
      const matchesSearch = !searchLower
        || instance.name.toLowerCase().includes(searchLower)
        || instance.id.toLowerCase().includes(searchLower);
      return matchesStatus && matchesSearch;
    });
  }, [activeDistribution, detailSearchQuery, statusFilter]);

  const handleDistributeStart = useCallback((_selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    startPackageOperation({
      pkg,
      selectedInstancesData,
      type: 'distribute',
    });
    setDistributeDialogOpen(false);
    toast.success('已开始下发流程');
  }, [pkg]);

  const handleUninstallStart = useCallback((_selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    startPackageOperation({
      pkg,
      selectedInstancesData,
      type: 'delete',
    });
    setUninstallDialogOpen(false);
    toast.success('已开始卸载流程');
  }, [pkg]);

  return (
    <div className="space-y-4">
      {/* 顶部返回
          停服态下仍允许返回公共技能包：纯导航类操作。 */}
      <div data-billing-exempt>
        <BackButton onClick={onBack}>返回公共技能包</BackButton>
      </div>

      {/* 技能包信息头部 */}
      <div className="bg-white rounded-[4px] border border-gray-200 p-5">
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1 min-w-0">
            <TenantDocTitle as="h1" tone="primary" className="truncate mb-0.5">
              {pkg.name}
            </TenantDocTitle>
            {categoryName && (
              <MetaText as="p" tone="weak" className="mb-2">
                分类：{categoryName}
              </MetaText>
            )}
            <BodyText as="p" tone="secondary" className="leading-relaxed">
              {pkg.descriptionLong}
            </BodyText>
          </div>
          <div className="flex-shrink-0">
            <FavoriteButton isFavorited={isFavorited} onToggle={onFavorite} />
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as typeof activeTab)} className="w-full">
        <div className="flex items-center justify-between">
          {/* 文件列表 / 下发记录 / 卸载记录 二级 Tab
              停服态下仍允许切换 Tab 查看内容：纯导航/查看类操作。
              通过 data-billing-exempt 豁免；批量下发/批量卸载按钮不受此豁免影响，继续受停服约束。 */}
          <div data-billing-exempt>
            <SegmentedTabs
              tabs={[
                { id: 'files', label: '文件列表' },
                { id: 'distribution', label: '下发记录' },
                { id: 'uninstall', label: '卸载记录' },
              ]}
              active={activeTab}
              onChange={(value) => setActiveTab(value as typeof activeTab)}
              ariaLabel="公共技能包详情 Tab 切换"
            />
          </div>
          {activeTab === 'distribution' && (
            <Button
              variant="claw-primary"
              size="claw"
              onClick={() => setDistributeDialogOpen(true)}
              disabled={hasInProgress}
            >
              {hasDistributing ? '下发中...' : '批量下发'}
            </Button>
          )}
          {activeTab === 'uninstall' && (
            <Button
              variant="claw-primary"
              size="claw"
              onClick={() => setUninstallDialogOpen(true)}
              disabled={hasInProgress || distributedInstancesForUninstall.length === 0}
              className={hasInProgress || distributedInstancesForUninstall.length === 0 ? 'opacity-50 cursor-not-allowed' : ''}
            >
              {hasDeleting ? '卸载中...' : '批量卸载'}
            </Button>
          )}
        </div>
        <TabsList className="hidden">
          <TabsTrigger value="files" />
          <TabsTrigger value="distribution" />
          <TabsTrigger value="uninstall" />
        </TabsList>

        <TabsContent value="files" className="mt-4 space-y-4 p-0">
          <div className="bg-white rounded-[4px] border border-gray-200 p-5">
            <SectionTitle as="h3" tone="primary" className="mb-0.5">
              技能模块
            </SectionTitle>
            <MetaText as="p" tone="weak" className="mb-3">
              共 {pkg.skills.length} 个 Skill
            </MetaText>
            <BodyText as="p" tone="muted" className="leading-relaxed mb-4">
              本技能包是一个场景化组合模板，包含以下 Skill。在「角色设定」中应用此技能包时，将自动展开为下列公共技能。
            </BodyText>
            <div className="flex flex-wrap items-center gap-2">
              {pkg.skills.map((s) => (
                <SkillChip key={s.slug} skill={s} size="md" />
              ))}
            </div>
          </div>

          {pkg.workflowMarkdown ? (
            <div className="bg-white rounded-[4px] border border-gray-200 p-5">
              <MDXRenderer content={pkg.workflowMarkdown} />
            </div>
          ) : (
            <div className="bg-white rounded-[4px] border border-gray-200 p-5 text-center">
              <HelperText>该技能包暂无工作流说明</HelperText>
            </div>
          )}
        </TabsContent>

        <TabsContent value="distribution" className="mt-4 p-0">
          <div className="space-y-3">
            {distributionOnlyRecords.length === 0 ? (
              <div className="flex flex-col items-center justify-center rounded-[4px] border border-gray-200 bg-white py-12 text-center">
                <HelperText as="p">还没有下发记录</HelperText>
              </div>
            ) : (
              distributionOnlyRecords.map((record, idx) => {
                const summary = getPackageRecordSummary(record);
                return (
                  <div key={record.id} className="rounded-[4px] border border-gray-200 bg-white p-4">
                    <div className="mb-3 flex items-start justify-between">
                      <BodyText as="p" tone="primary" className="font-semibold">
                        #{idx + 1} · {new Date(record.timestamp).toLocaleString('zh-CN')}
                      </BodyText>
                      <div className="flex items-center gap-2">
                        <span className={`inline-block rounded px-3 py-1 text-xs font-medium ${summary.tone}`}>
                          {summary.text}
                        </span>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setActiveDistributionId(record.id);
                            setStatusFilter('all');
                            setDetailSearchQuery('');
                            setDetailsOpen(true);
                          }}
                          className="h-auto px-2 py-1 text-[var(--text-brand)] hover:text-[var(--text-brand)]"
                        >
                          查看详情
                        </Button>
                      </div>
                    </div>
                    {record.status === 'distributing' && (
                      <div className="h-2 w-full rounded-full bg-gray-200">
                        <div className="h-2 rounded-full bg-blue-600 transition-all duration-300" style={{ width: `${summary.progress}%` }} />
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </TabsContent>

        <TabsContent value="uninstall" className="mt-4 p-0">
          <div className="space-y-3">
            {deleteOnlyRecords.length === 0 ? (
              <div className="flex flex-col items-center justify-center rounded-[4px] border border-gray-200 bg-white py-12 text-center">
                <HelperText as="p">还没有卸载记录</HelperText>
              </div>
            ) : (
              deleteOnlyRecords.map((record, idx) => {
                const summary = getPackageRecordSummary(record);
                const isInProgress = record.status === 'deleting' || record.status === 'distributing';
                return (
                  <div key={record.id} className="rounded-[4px] border border-gray-200 bg-white p-4">
                    <div className="mb-3 flex items-start justify-between">
                      <BodyText as="p" tone="primary" className="font-semibold">
                        #{idx + 1} · {new Date(record.timestamp).toLocaleString('zh-CN')}
                      </BodyText>
                      <div className="flex items-center gap-2">
                        <span className={`inline-block rounded px-3 py-1 text-xs font-medium ${summary.tone}`}>
                          {summary.text}
                        </span>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setActiveDistributionId(record.id);
                            setStatusFilter('all');
                            setDetailSearchQuery('');
                            setDetailsOpen(true);
                          }}
                          className="h-auto px-2 py-1 text-[var(--text-brand)] hover:text-[var(--text-brand)]"
                        >
                          查看详情
                        </Button>
                      </div>
                    </div>
                    {isInProgress && (
                      <div className="h-2 w-full rounded-full bg-gray-200">
                        <div className="h-2 rounded-full bg-red-500 transition-all duration-300" style={{ width: `${summary.progress}%` }} />
                      </div>
                    )}
                  </div>
                );
              })
            )}
          </div>
        </TabsContent>
      </Tabs>
      <PublicBatchDistributeDialog
        open={distributeDialogOpen}
        onOpenChange={setDistributeDialogOpen}
        skillId={skillResourceId}
        skillName={pkg.name}
        skillVersion=""
        instances={distributeInstances as any}
        groups={groups}
        onDistributionStart={handleDistributeStart}
        title="批量下发技能包"
        descriptionNode={(
          <div className="space-y-1">
            <BodyText as="p" tone="muted">
              将 <BodyMedium as="span" tone="primary" className="font-semibold">
                {pkg.name}
              </BodyMedium> 中的 <BodyMedium as="span" tone="primary" className="font-semibold">
                {pkg.skills.length}
              </BodyMedium> 个技能部署至所选实例。
            </BodyText>
            <BodyText as="p" tone="muted">
              仅支持向 <BodyText as="span" tone="secondary">运行中</BodyText> 的实例下发技能包；下发后会将该技能包内包含的公共技能统一安装到目标实例。已下发当前技能包的实例默认不展示，可选择后再次下发。
            </BodyText>
          </div>
        )}
      />
      <PublicBatchDeleteDialog
        open={uninstallDialogOpen}
        onOpenChange={setUninstallDialogOpen}
        skillName={pkg.name}
        skillVersion=""
        resourceLabel="技能包"
        introNode={(
          <>
            从已下发实例中卸载来自技能包 <span className="font-medium text-[var(--text-title)]">{pkg.name}</span> 的 <span className="font-medium text-[var(--text-title)]">{pkg.skills.length}</span> 个技能
          </>
        )}
        distributedInstances={distributedInstancesForUninstall}
        groups={groups}
        onDeleteStart={handleUninstallStart}
        warningText="卸载成功后，该技能包在对应实例上恢复为“未下发”状态。"
        emptyText="暂无已下发的实例"
        showDistributedVersion={false}
      />
      <Dialog open={detailsOpen} onOpenChange={setDetailsOpen}>
        <DialogContent className="flex w-[700px] max-h-[min(90vh,780px)] !max-w-[700px] flex-col overflow-hidden">
          <DialogHeader className="-mx-6 shrink-0 px-6 pt-6 pb-3">
            <DialogTitle>{activeDistribution?.type === 'delete' ? '卸载详情' : '下发详情'}</DialogTitle>
          </DialogHeader>
          {activeDistribution && (
            <div className="-mx-6 flex-1 min-h-0 overflow-y-auto px-6 py-0 pb-6">
              <div className="flex min-h-0 flex-col space-y-4 overflow-hidden">
                <div className="flex items-center gap-2">
                  <div className="relative flex-1">
                    <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
                    <Input
                      placeholder="搜索实例名称/ID..."
                      value={detailSearchQuery}
                      onChange={(e) => setDetailSearchQuery(e.target.value)}
                      className="h-9 pl-10"
                    />
                  </div>
                  <Select value={statusFilter} onValueChange={(value: any) => setStatusFilter(value)}>
                    <SelectTrigger className="w-28">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部</SelectItem>
                      <SelectItem value="success">成功</SelectItem>
                      <SelectItem value="failed">失败</SelectItem>
                      <SelectItem value="distributing">{activeDistribution.type === 'delete' ? '卸载中' : '下发中'}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex-1 min-h-0 overflow-y-auto rounded-[4px] border border-gray-200">
                  <Table density="compact" containerClassName="max-h-full overflow-y-auto">
                    <TableHeader>
                      <TableRow>
                        <TableHead>实例名称</TableHead>
                        <TableHead className="min-w-[140px]">实例ID</TableHead>
                        <TableHead>状态</TableHead>
                        <TableHead>失败原因</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {filteredInstances.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={4} className="text-center">
                            <BodyText as="span" tone="muted">没有符合条件的记录</BodyText>
                          </TableCell>
                        </TableRow>
                      ) : (
                        filteredInstances.map((instance) => (
                          <TableRow key={instance.id}>
                            <TableCell>{instance.name}</TableCell>
                            <TableCell className="font-mono text-gray-500">{instance.id}</TableCell>
                            <TableCell>
                              <span className={`inline-block rounded px-2 py-1 text-xs font-medium ${DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.color || 'bg-gray-50 text-gray-500'}`}>
                                {DISTRIBUTION_STATUS_MAP[instance.distributionStatus]?.label || '未下发'}
                              </span>
                            </TableCell>
                            <TableCell className="text-gray-500">{instance.failReason || '-'}</TableCell>
                          </TableRow>
                        ))
                      )}
                    </TableBody>
                  </Table>
                </div>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ─── 主组件 ───────────────────────────────────────────────────────────────────

interface PublicSkillPackageTabProps {
  packages: Array<{ id: string; name: string; isActive: boolean; scopeType?: 'all-users' | 'groups'; scopeLabel?: string; groupIds?: string[] }>;
  groups: Array<{ id: string; name: string; parentId?: string | null }>;
  onAddSkillToPackage: (skillId: string, packageId: string) => void;
}

const PAGE_SIZE = 24;
const PUBLIC_SKILL_RESOURCE_PREFIX = 'public-skill';
const PUBLIC_PACKAGE_RESOURCE_PREFIX = 'public-package';
const ACADEMIC_WRITING_PACKAGE_ID = 'pkg-1';

const getPublicSkillResourceId = (skillId: string) => `${PUBLIC_SKILL_RESOURCE_PREFIX}:${skillId}`;
const getPublicPackageResourceId = (packageId: string) => `${PUBLIC_PACKAGE_RESOURCE_PREFIX}:${packageId}`;

type PackageOperationType = 'distribute' | 'delete';

const ACADEMIC_WRITING_MOCK_DISTRIBUTION_RECORDS: CachedDistributionRecord[] = [
  {
    id: 'mock-academic-writing-dist-success',
    skillId: getPublicPackageResourceId(ACADEMIC_WRITING_PACKAGE_ID),
    timestamp: '2026-06-18T03:20:00.000Z',
    totalCount: 2,
    successCount: 2,
    failedCount: 0,
    inProgressCount: 0,
    status: 'success',
    type: 'distribute',
    operator: 'admin',
    instances: [
      {
        id: 'oc-7',
        name: 'OpenClaw-预发布环境',
        createdBy: 'admin',
        distributionStatus: 'success',
        packageOperationStatus: 'success',
      } as any,
      {
        id: 'oc-6',
        name: 'OpenClaw-回归测试',
        createdBy: 'dev-team',
        distributionStatus: 'success',
        packageOperationStatus: 'success',
      } as any,
    ],
  },
  {
    id: 'mock-academic-writing-dist-mixed',
    skillId: getPublicPackageResourceId(ACADEMIC_WRITING_PACKAGE_ID),
    timestamp: '2026-06-17T09:40:00.000Z',
    totalCount: 2,
    successCount: 1,
    failedCount: 1,
    inProgressCount: 0,
    status: 'failed',
    type: 'distribute',
    operator: 'admin',
    instances: [
      {
        id: 'oc-1',
        name: 'OpenClaw-生产环境',
        createdBy: 'admin',
        distributionStatus: 'success',
        packageOperationStatus: 'success',
      } as any,
      {
        id: 'oc-4',
        name: 'OpenClaw-备用实例',
        createdBy: 'ops',
        distributionStatus: 'failed',
        packageOperationStatus: 'failed',
        failReason: '实例权限不足',
      } as any,
    ],
  },
  {
    id: 'mock-academic-writing-dist-existing-failed',
    skillId: getPublicPackageResourceId(ACADEMIC_WRITING_PACKAGE_ID),
    timestamp: '2026-06-17T07:35:00.000Z',
    totalCount: 2,
    successCount: 1,
    failedCount: 1,
    inProgressCount: 0,
    status: 'failed',
    type: 'distribute',
    operator: 'admin',
    instances: [
      {
        id: 'oc-2',
        name: 'OpenClaw-测试环境',
        createdBy: 'dev-team',
        distributionStatus: 'success',
        packageOperationStatus: 'success',
      } as any,
      {
        id: 'oc-5',
        name: 'OpenClaw-灾备中心',
        createdBy: 'admin',
        distributionStatus: 'failed',
        packageOperationStatus: 'failed',
        failReason: '部分skill下发失败',
      } as any,
    ],
  },
  {
    id: 'mock-academic-writing-dist-failed',
    skillId: getPublicPackageResourceId(ACADEMIC_WRITING_PACKAGE_ID),
    timestamp: '2026-06-16T10:15:00.000Z',
    totalCount: 2,
    successCount: 0,
    failedCount: 2,
    inProgressCount: 0,
    status: 'failed',
    type: 'distribute',
    operator: 'admin',
    instances: [
      {
        id: 'oc-12',
        name: 'OpenClaw-华北节点A',
        createdBy: 'dev-team',
        distributionStatus: 'failed',
        packageOperationStatus: 'failed',
        failReason: '实例未处于运行中',
      } as any,
      {
        id: 'oc-15',
        name: 'OpenClaw-西北节点',
        createdBy: 'ops',
        distributionStatus: 'failed',
        packageOperationStatus: 'failed',
        failReason: '网络连接超时',
      } as any,
    ],
  },
];

function normalizePackageRecord(record: CachedDistributionRecord): CachedDistributionRecord {
  return {
    ...record,
    instances: record.instances.map((instance) => {
      const packageOperationStatus = (instance as any).packageOperationStatus;
      if (packageOperationStatus !== 'partial_success') return instance;

      return {
        ...instance,
        distributionStatus: 'failed',
        packageOperationStatus: 'failed',
        failReason: record.type === 'delete' ? '部分skill卸载失败' : '部分skill下发失败',
      } as typeof instance;
    }),
  };
}

const mergeFixedPackageRecords = (
  packageId: string,
  records: CachedDistributionRecord[],
) => {
  const normalizedRecords = records.map(normalizePackageRecord);
  if (packageId !== ACADEMIC_WRITING_PACKAGE_ID) return normalizedRecords;

  const recordIds = new Set(normalizedRecords.map((record) => record.id));
  const fixedRecords = ACADEMIC_WRITING_MOCK_DISTRIBUTION_RECORDS.filter(
    (record) => !recordIds.has(record.id),
  );

  return [...fixedRecords, ...normalizedRecords].sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
  ).map(normalizePackageRecord);
};

const getPackageIdFromResourceId = (packageResourceId: string) =>
  packageResourceId.startsWith(`${PUBLIC_PACKAGE_RESOURCE_PREFIX}:`)
    ? packageResourceId.slice(PUBLIC_PACKAGE_RESOURCE_PREFIX.length + 1)
    : packageResourceId;

const getPackageRecords = (packageResourceId: string) => {
  const packageId = getPackageIdFromResourceId(packageResourceId);
  const cachedRecords = getAllDistributionRecords().filter(
    (record) => record.skillId === packageResourceId,
  );

  return mergeFixedPackageRecords(packageId, cachedRecords);
};

const getResolvedPackageSkills = (pkg: PublicSkillPackage): PublicSkill[] =>
  pkg.skills
    .map((skillRef) => PUBLIC_SKILLS.find((skill) => skill.slug === skillRef.slug))
    .filter((skill): skill is PublicSkill => Boolean(skill));

const getLatestPackageRecord = (
  packageResourceId: string,
  type: PackageOperationType = 'distribute',
) =>
  getPackageRecords(packageResourceId).find(
    (record) =>
      (record.type || 'distribute') === type,
  );

const getPackageRecordInstalledInstanceIds = (
  pkg: PublicSkillPackage,
) => {
  const packageResourceId = getPublicPackageResourceId(pkg.id);
  const records = getPackageRecords(packageResourceId);
  const installedIds = new Set<string>();

  [...records].reverse().forEach((record) => {
    record.instances.forEach((instance) => {
      if (instance.distributionStatus !== 'success') return;

      if ((record.type || 'distribute') === 'delete') {
        installedIds.delete(instance.id);
        return;
      }

      installedIds.add(instance.id);
    });
  });

  return installedIds;
};

const getPackageInstalledInstanceIds = (
  pkg: PublicSkillPackage,
  mode: 'all' | 'any',
  allInstances = MOCK_OPENCLAW_INSTANCES,
) => {
  const skills = getResolvedPackageSkills(pkg);
  const fallbackInstalledIds = getPackageRecordInstalledInstanceIds(pkg);
  if (skills.length === 0) return fallbackInstalledIds;

  const allRecords = getAllDistributionRecords();
  const hasSkillLevelRecords = skills.some((skill) =>
    allRecords.some((record) => record.skillId === getPublicSkillResourceId(skill.id)),
  );
  if (!hasSkillLevelRecords) return fallbackInstalledIds;

  const installedSets = skills.map(
    (skill) =>
      new Set(
        getCurrentSkillInstalledInstances(
          getPublicSkillResourceId(skill.id),
          skill.version,
          allInstances,
        ).map((instance) => instance.id),
      ),
  );

  const skillLevelInstalledIds = new Set(
    allInstances
      .filter((instance) =>
        mode === 'all'
          ? installedSets.every((ids) => ids.has(instance.id))
          : installedSets.some((ids) => ids.has(instance.id)),
      )
      .map((instance) => instance.id),
  );

  fallbackInstalledIds.forEach((id) => skillLevelInstalledIds.add(id));
  return skillLevelInstalledIds;
};

const getInstancesWithPackageDistributionStatus = (
  pkg: PublicSkillPackage,
  allInstances = MOCK_OPENCLAW_INSTANCES,
) => {
  const packageResourceId = getPublicPackageResourceId(pkg.id);
  const latestRecord = getLatestPackageRecord(packageResourceId, 'distribute');
  const latestStatusById = new Map(
    latestRecord?.instances.map((instance) => [instance.id, instance]) ?? [],
  );
  const fullyInstalledIds = getPackageInstalledInstanceIds(pkg, 'all', allInstances);

  return allInstances.map((instance) => {
    const latestStatus = latestStatusById.get(instance.id);
    const fullyInstalled = fullyInstalledIds.has(instance.id);

    if (fullyInstalled) {
      return {
        ...instance,
        ...latestStatus,
        distributionStatus: 'success' as const,
      };
    }

    if (
      latestStatus &&
      (latestStatus.distributionStatus === 'failed' ||
        latestStatus.distributionStatus === 'distributing')
    ) {
      return {
        ...instance,
        ...latestStatus,
      };
    }

    return {
      ...instance,
      distributionStatus: 'not_distributed' as const,
    };
  });
};

const getPackageInstancesForUninstall = (
  pkg: PublicSkillPackage,
  groups: Array<{ id: string; name: string; parentId?: string | null }>,
) => {
  const anyInstalledIds = getPackageInstalledInstanceIds(pkg, 'any');
  const latestDeleteRecord = getLatestPackageRecord(
    getPublicPackageResourceId(pkg.id),
    'delete',
  );
  const latestDeleteStatusById = new Map(
    latestDeleteRecord?.instances.map((instance) => [instance.id, instance]) ?? [],
  );

  return MOCK_OPENCLAW_INSTANCES.filter((instance) => anyInstalledIds.has(instance.id)).map(
    (instance) => {
      const primaryGroupId = instance.groupIds?.[0];
      const groupName = primaryGroupId
        ? groups.find((group) => group.id === primaryGroupId)?.name
        : undefined;
      const latestDeleteStatus = latestDeleteStatusById.get(instance.id);

      return {
        id: instance.id,
        name: instance.name,
        createdBy: instance.createdBy || 'admin',
        groupName: groupName || '全部用户',
        distributedTime: instance.createdAt,
        agentType: instance.agentType,
        agentVersion: instance.agentVersion,
        localProduct: instance.localProduct,
        deleteStatus:
          latestDeleteStatus?.distributionStatus === 'failed'
            ? ('delete_failed' as const)
            : ('not_deleted' as const),
        deleteFailReason: latestDeleteStatus?.failReason,
      };
    },
  );
};

const getPackageRecordSummary = (record: CachedDistributionRecord) => {
  if (record.status === 'distributing' || record.status === 'deleting') {
    const progress =
      record.totalCount > 0 ? Math.round((record.successCount / record.totalCount) * 100) : 0;
    return {
      progress,
      tone: record.type === 'delete' ? 'bg-red-100 text-red-700' : 'bg-blue-50 text-blue-700',
      text: `${record.type === 'delete' ? '卸载中' : '下发中'} ${progress}%`,
    };
  }

  const successCount = record.instances.filter((instance) => {
    const packageOperationStatus = (instance as any).packageOperationStatus;
    return packageOperationStatus === 'success'
      || (!packageOperationStatus && instance.distributionStatus === 'success');
  }).length;
  const failedCount = record.instances.filter((instance) => {
    const packageOperationStatus = (instance as any).packageOperationStatus;
    return packageOperationStatus === 'failed'
      || (!packageOperationStatus && instance.distributionStatus === 'failed');
  }).length;

  if (failedCount === 0) {
    return {
      progress: 100,
      tone: 'bg-green-50 text-green-700',
      text: `${record.type === 'delete' ? '卸载完成' : '下发完成'}，${successCount}个成功，${failedCount}个失败`,
    };
  }

  if (successCount === 0) {
    return {
      progress: 100,
      tone: 'bg-red-50 text-red-700',
      text: `${record.type === 'delete' ? '卸载完成' : '下发完成'}，${successCount}个成功，${failedCount}个失败`,
    };
  }

  return {
    progress: 100,
    tone: 'bg-yellow-50 text-yellow-700',
    text: `${record.type === 'delete' ? '卸载完成' : '下发完成'}，${successCount}个成功，${failedCount}个失败`,
  };
};

const startPackageOperation = ({
  pkg,
  selectedInstancesData,
  type,
  onPackageRecordCreated,
}: {
  pkg: PublicSkillPackage;
  selectedInstancesData: any[];
  type: PackageOperationType;
  onPackageRecordCreated?: (recordId: string) => void;
}) => {
  const packageResourceId = getPublicPackageResourceId(pkg.id);
  const resolvedSkills = getResolvedPackageSkills(pkg);
  const packageRecordId = createDistributionRecordId();
  const totalCount = selectedInstancesData.length;

  const baseInstances = selectedInstancesData.map((instance) => ({
    id: instance.id,
    name: instance.name,
    createdBy: instance.createdBy || 'admin',
    agentType: instance.agentType,
    agentVersion: instance.agentVersion,
    localProduct: instance.localProduct,
    distributionStatus: 'distributing' as const,
  }));

  addDistributionRecord({
    id: packageRecordId,
    skillId: packageResourceId,
    timestamp: new Date().toISOString(),
    totalCount,
    successCount: 0,
    failedCount: 0,
    inProgressCount: totalCount,
    status: type === 'delete' ? 'deleting' : 'distributing',
    type,
    operator: 'admin',
    instances: baseInstances,
  });

  const skillRecordIds = resolvedSkills.map((skill) => {
    const recordId = createDistributionRecordId();
    addDistributionRecord({
      id: recordId,
      skillId: getPublicSkillResourceId(skill.id),
      timestamp: new Date().toISOString(),
      totalCount,
      successCount: 0,
      failedCount: 0,
      inProgressCount: totalCount,
      status: type === 'delete' ? 'deleting' : 'distributing',
      type,
      operator: 'admin',
      instances: baseInstances.map((instance) => ({
        ...instance,
        distributedVersion: skill.version,
      })),
    });
    return { skill, recordId };
  });

  onPackageRecordCreated?.(packageRecordId);

  let completed = 0;
  const interval = setInterval(() => {
    completed += Math.floor(Math.random() * 3) + 1;
    if (completed < totalCount) {
      updateDistributionRecord(packageRecordId, (record) => ({
        ...record,
        successCount: completed,
        inProgressCount: totalCount - completed,
      }));
      skillRecordIds.forEach(({ recordId }) => {
        updateDistributionRecord(recordId, (record) => ({
          ...record,
          successCount: completed,
          inProgressCount: totalCount - completed,
        }));
      });
      return;
    }

    clearInterval(interval);

    const packageInstanceResults = selectedInstancesData.map((instance) => {
      const skillResults = skillRecordIds.map(({ skill }) => ({
        skill,
        success: Math.random() < (type === 'delete' ? 0.9 : 0.85),
      }));
      const failedSkills = skillResults.filter((result) => !result.success);
      const operationStatus = failedSkills.length === 0 ? 'success' : 'failed';

      return {
        instance,
        skillResults,
        success: failedSkills.length === 0,
        operationStatus,
        failReason:
          failedSkills.length > 0
            ? (type === 'delete' ? '部分skill卸载失败' : '部分skill下发失败')
            : undefined,
      };
    });

    const packageSuccessCount = packageInstanceResults.filter((result) => result.success).length;
    const packageFailedCount = packageInstanceResults.filter(
      (result) => result.operationStatus === 'failed',
    ).length;

    updateDistributionRecord(packageRecordId, (record) => ({
      ...record,
      successCount: packageSuccessCount,
      failedCount: packageFailedCount,
      inProgressCount: 0,
      status: packageFailedCount === 0 ? 'success' : 'failed',
      instances: packageInstanceResults.map((result) => ({
        id: result.instance.id,
        name: result.instance.name,
        createdBy: result.instance.createdBy || 'admin',
        agentType: result.instance.agentType,
        agentVersion: result.instance.agentVersion,
        localProduct: result.instance.localProduct,
        distributionStatus: result.operationStatus === 'success' ? ('success' as const) : ('failed' as const),
        packageOperationStatus: result.operationStatus,
        failReason: result.failReason,
      })),
    }));

    skillRecordIds.forEach(({ skill, recordId }) => {
      const skillSuccessCount = packageInstanceResults.filter((result) =>
        result.skillResults.find((skillResult) => skillResult.skill.id === skill.id)?.success,
      ).length;
      const skillFailedCount = totalCount - skillSuccessCount;

      updateDistributionRecord(recordId, (record) => ({
        ...record,
        successCount: skillSuccessCount,
        failedCount: skillFailedCount,
        inProgressCount: 0,
        status: skillFailedCount === 0 ? 'success' : 'failed',
        instances: packageInstanceResults.map((result) => {
          const skillResult = result.skillResults.find(
            (item) => item.skill.id === skill.id,
          );
          return {
            id: result.instance.id,
            name: result.instance.name,
            createdBy: result.instance.createdBy || 'admin',
            agentType: result.instance.agentType,
            agentVersion: result.instance.agentVersion,
            localProduct: result.instance.localProduct,
            distributedVersion: skill.version,
            distributionStatus: skillResult?.success ? ('success' as const) : ('failed' as const),
            failReason: skillResult?.success
              ? undefined
              : type === 'delete'
                ? '实例离线'
                : '命令下发失败',
          };
        }),
      }));
    });
  }, 800);
};

export default function PublicSkillPackageTab({
  packages,
  groups,
  onAddSkillToPackage,
}: PublicSkillPackageTabProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [activeCategory, setActiveCategory] = useState<string>('all');
  const [favoriteIds, setFavoriteIds] = useState<Set<string>>(new Set());
  const [addedPackageIdsByPackage, setAddedPackageIdsByPackage] = useState<Record<string, string[]>>({});
  const [selectedPackageId, setSelectedPackageId] = useState<string | null>(null);
  const [addToPackagePackageId, setAddToPackagePackageId] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [distributeDialogOpen, setDistributeDialogOpen] = useState(false);
  const [distributePackageId, setDistributePackageId] = useState<string | null>(null);
  const [uninstallDialogOpen, setUninstallDialogOpen] = useState(false);
  const [uninstallPackageId, setUninstallPackageId] = useState<string | null>(null);
  const [distributionSummaries, setDistributionSummaries] = useState<Record<string, SkillDistributionSummary>>({});

  const refreshDistributionSummaries = useCallback(() => {
    const next: Record<string, SkillDistributionSummary> = {};
    PUBLIC_SKILL_PACKAGES.forEach((pkg) => {
      const resourceId = getPublicPackageResourceId(pkg.id);
      const summary = getSkillDistributionSummary(resourceId);
      if (summary) next[resourceId] = summary;
    });
    setDistributionSummaries(next);
  }, []);

  useEffect(() => {
    refreshDistributionSummaries();
    const handleCacheUpdated = () => refreshDistributionSummaries();
    window.addEventListener('distribution-cache-updated', handleCacheUpdated);
    return () => window.removeEventListener('distribution-cache-updated', handleCacheUpdated);
  }, [refreshDistributionSummaries]);

  const isFavorited = (id: string) => favoriteIds.has(id);

  // 收藏切换 + Toast 反馈
  const handleFavorite = (pkg: PublicSkillPackage) => {
    setFavoriteIds((prev) => {
      const next = new Set(prev);
      const willFavorite = !next.has(pkg.id);
      if (willFavorite) {
        next.add(pkg.id);
        toast.success(`已收藏「${pkg.name}」`);
      } else {
        next.delete(pkg.id);
        toast.success(`已取消收藏「${pkg.name}」`);
      }
      return next;
    });
  };

  const handleAddToInitialPackage = (pkgId: string) => {
    setAddToPackagePackageId(pkgId);
  };

  const handleDistribute = (pkgId: string) => {
    setDistributePackageId(pkgId);
    setDistributeDialogOpen(true);
  };

  const handleUninstall = (pkgId: string) => {
    setUninstallPackageId(pkgId);
    setUninstallDialogOpen(true);
  };

  const handlePackageSelected = (packageIds: string[]) => {
    if (addToPackagePackageId && packageIds.length > 0) {
      const selectedPackage = PUBLIC_SKILL_PACKAGES.find((pkg) => pkg.id === addToPackagePackageId);
      selectedPackage?.skills.forEach((skill) => {
        packageIds.forEach((packageId) => onAddSkillToPackage(skill.slug, packageId));
      });
      setAddedPackageIdsByPackage((prev) => {
        const existed = prev[addToPackagePackageId] ?? [];
        return {
          ...prev,
          [addToPackagePackageId]: Array.from(new Set([...existed, ...packageIds])),
        };
      });
    }
    setAddToPackagePackageId(null);
  };

  // 过滤
  const filteredPackages = useMemo(() => {
    let list: PublicSkillPackage[];
    if (activeCategory === 'all') {
      list = [...PUBLIC_SKILL_PACKAGES];
    } else if (activeCategory === 'favorites') {
      list = PUBLIC_SKILL_PACKAGES.filter((p) => favoriteIds.has(p.id));
    } else {
      list = PUBLIC_SKILL_PACKAGES.filter((p) => p.category === activeCategory);
    }

    const q = searchQuery.trim().toLowerCase();
    if (q) {
      list = list.filter(
        (p) =>
          p.name.toLowerCase().includes(q) ||
          p.descriptionLong.toLowerCase().includes(q) ||
          p.skills.some(
            (s) => s.slug.toLowerCase().includes(q) || s.name.toLowerCase().includes(q),
          ),
      );
    }
    return list;
  }, [activeCategory, searchQuery, favoriteIds]);

  // 分页
  const pagedPackages = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredPackages.slice(start, start + PAGE_SIZE);
  }, [filteredPackages, currentPage]);

  // 切分类、改搜索都要重置到第一页
  const handleCategoryChange = (catId: string) => {
    setActiveCategory(catId);
    setCurrentPage(1);
  };
  const handleSearchChange = (q: string) => {
    setSearchQuery(q);
    setCurrentPage(1);
  };

  const pendingAddPackage = PUBLIC_SKILL_PACKAGES.find((pkg) => pkg.id === addToPackagePackageId);
  const selectedDistributePackage = distributePackageId
    ? PUBLIC_SKILL_PACKAGES.find((pkg) => pkg.id === distributePackageId) ?? null
    : null;
  const selectedUninstallPackage = uninstallPackageId
    ? PUBLIC_SKILL_PACKAGES.find((pkg) => pkg.id === uninstallPackageId) ?? null
    : null;

  const distributeInstances = useMemo(() => {
    if (!selectedDistributePackage) return MOCK_OPENCLAW_INSTANCES;
    return getInstancesWithPackageDistributionStatus(
      selectedDistributePackage,
      MOCK_OPENCLAW_INSTANCES,
    );
  }, [distributionSummaries, selectedDistributePackage]);

  const distributedInstancesForUninstall = useMemo(() => {
    if (!selectedUninstallPackage) return [];
    return getPackageInstancesForUninstall(selectedUninstallPackage, groups);
  }, [distributionSummaries, groups, selectedUninstallPackage]);

  const isDistributionInProgress = useCallback(
    (pkgId: string) => distributionSummaries[getPublicPackageResourceId(pkgId)]?.hasInProgress ?? false,
    [distributionSummaries],
  );

  const getUninstallableCount = useCallback((pkgId: string) => {
    const pkg = PUBLIC_SKILL_PACKAGES.find((item) => item.id === pkgId);
    if (!pkg) return 0;
    return getPackageInstalledInstanceIds(pkg, 'any', MOCK_OPENCLAW_INSTANCES).size;
  }, []);

  const handleDistributeStart = useCallback((_selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    if (!selectedDistributePackage) return;
    startPackageOperation({
      pkg: selectedDistributePackage,
      selectedInstancesData,
      type: 'distribute',
    });
    setDistributeDialogOpen(false);
    toast.success('已开始下发流程');
  }, [selectedDistributePackage]);

  const handleUninstallStart = useCallback((_selectedInstanceIds: string[], selectedInstancesData: any[]) => {
    if (!uninstallPackageId) return;
    if (!selectedUninstallPackage) return;
    startPackageOperation({
      pkg: selectedUninstallPackage,
      selectedInstancesData,
      type: 'delete',
    });
    setUninstallDialogOpen(false);
    toast.success('已开始卸载流程');
  }, [selectedUninstallPackage, uninstallPackageId]);

  // 详情页
  if (selectedPackageId) {
    const pkg = PUBLIC_SKILL_PACKAGES.find((p) => p.id === selectedPackageId);
    if (pkg) {
      return (
        <PackageDetailView
          pkg={pkg}
          groups={groups}
          isFavorited={isFavorited(pkg.id)}
          onFavorite={() => handleFavorite(pkg)}
          onBack={() => setSelectedPackageId(null)}
        />
      );
    }
  }

  return (
    <>
      <div className="space-y-4">
        {/* 搜索框 + 刷新
            停服态下仍允许搜索与刷新：纯导航/查看类操作。
            通过 data-billing-exempt 豁免 AdminDisabledOverlay 的全局灰化与点击拦截。 */}
        <div className="flex items-center gap-2" data-billing-exempt>
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
            <Input
              placeholder="搜索技能包名称或关键词..."
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              className="pl-9 bg-white"
            />
          </div>
          <Button
            variant="claw-outline"
            size="icon"
            onClick={() => {
              setIsRefreshing(true);
              setTimeout(() => {
                handleSearchChange('');
                setCurrentPage(1);
                setTimeout(() => setIsRefreshing(false), 50);
              }, 250);
            }}
            title="刷新"
            className="w-9 h-9"
          >
            <RefreshCw className="w-4 h-4" />
          </Button>
        </div>

        {/* 分类标签栏（统一 FilterChipGroup）
            停服态下仍允许分类切换：纯导航/查看类操作。 */}
        <div data-billing-exempt>
          <FilterChipGroup
            items={PUBLIC_SKILL_PACKAGE_CATEGORIES.map((cat) => ({ id: cat.id, label: cat.name }))}
            value={activeCategory}
            onChange={handleCategoryChange}
          />
        </div>

        {/* 卡片网格 + 分页 */}
        {filteredPackages.length > 0 ? (
          <>
            <div
              className="grid grid-cols-3 gap-4"
              style={{ opacity: isRefreshing ? 0 : 1, transition: 'opacity 0.25s ease' }}
            >
              {pagedPackages.map((pkg) => (
                <PackageCard
                  key={pkg.id}
                  pkg={pkg}
                  isFavorited={isFavorited(pkg.id)}
                  onFavorite={(e) => {
                    e.stopPropagation();
                    handleFavorite(pkg);
                  }}
                  onAddToPackage={(e) => {
                    e.stopPropagation();
                    handleAddToInitialPackage(pkg.id);
                  }}
                  onDistribute={() => handleDistribute(pkg.id)}
                  onUninstall={() => handleUninstall(pkg.id)}
                  disableActions={isDistributionInProgress(pkg.id)}
                  uninstallDisabled={getUninstallableCount(pkg.id) === 0}
                  onClick={() => setSelectedPackageId(pkg.id)}
                />
              ))}
            </div>
            <div className="pt-3 border-t border-gray-200 mt-2">
              <Pagination
                total={filteredPackages.length}
                current={currentPage}
                pageSize={PAGE_SIZE}
                showTotal={(total) => `共 ${total} 个技能包`}
                className="w-full justify-between"
                onChange={(p) => {
                  setCurrentPage(p);
                  window.scrollTo({ top: 0, behavior: 'smooth' });
                }}
              />
            </div>
          </>
        ) : (
          <Empty className="border-0 py-16">
            <EmptyHeader>
              <EmptyMedia />
              <EmptyDescription>
                {activeCategory === 'favorites' && favoriteIds.size === 0
                  ? '还没有收藏任何技能包'
                  : '没有找到匹配的技能包'}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
      </div>
      <AddToPackageDialog
        open={!!addToPackagePackageId}
        itemName={pendingAddPackage?.name || ''}
        itemType="package"
        packages={packages}
        groups={groups}
        addedPackageIds={addToPackagePackageId ? addedPackageIdsByPackage[addToPackagePackageId] ?? [] : []}
        successMessage={(packageCount) => `已将「${pendingAddPackage?.name ?? ''}」中的 ${pendingAddPackage?.skills.length ?? 0} 个技能添加到 ${packageCount} 个初始技能包`}
        onConfirm={handlePackageSelected}
        onCancel={() => setAddToPackagePackageId(null)}
      />
      <PublicBatchDistributeDialog
        open={distributeDialogOpen}
        onOpenChange={setDistributeDialogOpen}
        skillId={selectedDistributePackage ? getPublicPackageResourceId(selectedDistributePackage.id) : undefined}
        skillName={selectedDistributePackage?.name}
        instances={distributeInstances as any}
        groups={groups}
        onDistributionStart={handleDistributeStart}
        title="批量下发技能包"
        descriptionNode={(
          <div className="space-y-1">
            <BodyText as="p" tone="muted">
              将 <BodyMedium as="span" tone="primary" className="font-semibold">
                {selectedDistributePackage?.name}
              </BodyMedium> 中的 <BodyMedium as="span" tone="primary" className="font-semibold">
                {selectedDistributePackage?.skills.length ?? 0}
              </BodyMedium> 个技能部署至所选实例。
            </BodyText>
            <BodyText as="p" tone="muted">
              仅支持向 <BodyText as="span" tone="secondary">运行中</BodyText> 的实例下发技能包；下发后会将该技能包内包含的公共技能统一安装到目标实例。已下发当前技能包的实例默认不展示，可选择后再次下发。
            </BodyText>
          </div>
        )}
      />
      {selectedUninstallPackage && (
        <PublicBatchDeleteDialog
          open={uninstallDialogOpen}
          onOpenChange={setUninstallDialogOpen}
          skillName={selectedUninstallPackage.name}
          skillVersion=""
          resourceLabel="技能包"
          introNode={(
            <>
              从已下发实例中卸载来自技能包 <span className="font-medium text-[var(--text-title)]">{selectedUninstallPackage.name}</span> 的 <span className="font-medium text-[var(--text-title)]">{selectedUninstallPackage.skills.length}</span> 个技能
            </>
          )}
          distributedInstances={distributedInstancesForUninstall}
          groups={groups}
          onDeleteStart={handleUninstallStart}
          warningText="卸载成功后，该技能包在对应实例上恢复为“未下发”状态。"
          emptyText="暂无已下发的实例"
          showDistributedVersion={false}
        />
      )}
    </>
  );
}
