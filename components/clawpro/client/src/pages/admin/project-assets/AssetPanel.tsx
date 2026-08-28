/**
 * AssetPanel - 「项目资产管理」右侧面板
 * - Tab：项目资产 / 项目实例
 * - 项目资产：草稿编辑态（进入编辑仅改本地临时状态，点保存才写入 store 并生成版本记录）
 * - 顶部两张卡片并排：左=当前版本 + 查看更新记录；右=同步模式（仅新增实例初始配置 / 所有实例始终同步更新）
 * - 一张统一资产卡片：空态不留骨架；按大类聚类平铺标签；编辑态可逐个删除、点击「添加」弹出多 Tab 勾选弹窗
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { Pencil, Save, X, History, Boxes, Server, Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import { Sheet, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { StatusTag } from '@/components/ui/status-tag';
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Empty, EmptyHeader, EmptyDescription } from '@/components/ui/empty';
import { BodyMedium, MetaText } from '@/components/ui/Typography';
import { getGroupPath, getProjectMembers, getOrgMembersDeep } from './projectRelations';
import type { UserGroup, UserOrg } from '../MemberManagement/types';
import { AddUsersToGroupDialog } from '../MemberManagement/AddUsersToGroupDialog';
import { RemoveUserFromGroupDialog } from '../MemberManagement/RemoveUserFromGroupDialog';
import {
  ASSET_CATEGORY_MAP,
  ASSET_CATEGORY_ORDER,
  ASSET_SYNC_MODE_MAP,
  type AssetCategory,
  type AssetSyncMode,
  type ProjectAssetCategoryConfig,
  type ProjectAssetItemRef,
} from './types';
import { projectAssetStore } from './projectAssetStore';
import { getAssetItemDisplay, getAssetTagMeta, getCategoryLibraryItems } from './assetSelectors';
import { skillStore } from '../SkillLibrary/skillStore';
import { pluginStore } from '../SkillLibrary/pluginStore';
import { mcpStore } from '../SkillLibrary/mcpStore';
import { standardsStore } from '../SkillLibrary/standardsStore';
import AddAssetsDialog from './AddAssetsDialog';
import UpdateRecordsTab from './UpdateRecordsTab';
import ProjectInstancesTab from './ProjectInstancesTab';

type CategoryMap = Record<AssetCategory, ProjectAssetCategoryConfig>;
type PanelTab = 'assets' | 'instances';

function cloneCategories(categories: CategoryMap): CategoryMap {
  return ASSET_CATEGORY_ORDER.reduce((acc, category) => {
    const cat = categories[category];
    acc[category] = { items: cat.items.map((i) => ({ ...i })) };
    return acc;
  }, {} as CategoryMap);
}

interface AssetPanelProps {
  groupId: string;
  groupName: string;
  groups: UserGroup[];
  users: UserOrg[];
  /** 当前 Tab（受控，由父层持有以便跨节点切换保持） */
  tab: PanelTab;
  /** 切换 Tab 回调 */
  onTabChange: (tab: PanelTab) => void;
  onAddUsers: (userIds: string[]) => void;
  onRemoveUser: (userId: string) => void;
  /** 项目节点右上角的项目基本信息编辑入口；组织节点不展示 */
  onEditProject?: () => void;
}

export default function AssetPanel({
  groupId,
  groupName,
  groups,
  users,
  tab,
  onTabChange,
  onAddUsers,
  onRemoveUser,
  onEditProject,
}: AssetPanelProps) {
  const setTab = onTabChange;
  const [editing, setEditing] = useState(false);
  const [categories, setCategories] = useState<CategoryMap>(() => projectAssetStore.getConfig(groupId).categories);
  const [mode, setMode] = useState<AssetSyncMode>(() => projectAssetStore.getConfig(groupId).mode);
  const [version, setVersion] = useState<number>(() => projectAssetStore.getConfig(groupId).version);
  const [draft, setDraft] = useState<CategoryMap>(() => cloneCategories(projectAssetStore.getConfig(groupId).categories));
  const [draftMode, setDraftMode] = useState<AssetSyncMode>(() => projectAssetStore.getConfig(groupId).mode);
  const [records, setRecords] = useState(() => projectAssetStore.getUpdateRecords(groupId));
  const [recordsDrawerOpen, setRecordsDrawerOpen] = useState(false);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [, setLibTick] = useState(0);

  // ─── 轻量成员管理（表头 hover 标签 + 添加/移除弹窗） ──────────
  const isProject = useMemo(
    () => groups.find((g) => g.id === groupId)?.source === 'project',
    [groups, groupId],
  );
  const term = isProject ? '项目' : '组织';
  // 成员口径：组织聚合本级 + 所有下级组织成员（去重）；项目单层取直接成员
  const members = useMemo(
    () => (isProject ? getProjectMembers(groupId, users) : getOrgMembersDeep(groupId, groups, users)),
    [isProject, groupId, groups, users],
  );

  const [addMemberOpen, setAddMemberOpen] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<UserOrg | null>(null);

  const reload = useCallback(() => {
    const cfg = projectAssetStore.getConfig(groupId);
    setCategories(cfg.categories);
    setMode(cfg.mode);
    setVersion(cfg.version);
    setRecords(projectAssetStore.getUpdateRecords(groupId));
  }, [groupId]);

  // 切换组织：重置编辑态并加载该组织配置
  // 注意：不重置 tab —— 用户停留在「实例」tab 时切换组织/项目应保持在实例 tab，
  // 不再强制跳回「资产」tab。
  useEffect(() => {
    const cfg = projectAssetStore.getConfig(groupId);
    setEditing(false);
    setCategories(cfg.categories);
    setMode(cfg.mode);
    setVersion(cfg.version);
    setRecords(projectAssetStore.getUpdateRecords(groupId));
  }, [groupId]);

  // 订阅 projectAssetStore + 各工具库 store：联动/外部变更时刷新
  useEffect(() => {
    const bump = () => setLibTick((v) => v + 1);
    const unsubs = [
      projectAssetStore.subscribe(() => {
        if (!editing) reload();
        else setRecords(projectAssetStore.getUpdateRecords(groupId));
      }),
      skillStore.subscribe(bump),
      pluginStore.subscribe(bump),
      mcpStore.subscribe(bump),
      standardsStore.subscribe(bump),
    ];
    return () => unsubs.forEach((fn) => fn());
  }, [editing, groupId, reload]);

  const activeCategories = editing ? draft : categories;
  // 项目节点：同步模式固定为「所有实例始终同步更新」，不可切换
  const activeMode: AssetSyncMode = isProject ? 'autoSync' : editing ? draftMode : mode;
  const totalCount = useMemo(
    () => ASSET_CATEGORY_ORDER.reduce((sum, c) => sum + activeCategories[c].items.length, 0),
    [activeCategories],
  );
  const hasAnyItems = totalCount > 0;

  const handleEnterEdit = () => {
    setDraft(cloneCategories(categories));
    setDraftMode(mode);
    setEditing(true);
  };

  const handleCancel = () => {
    setDraft(cloneCategories(categories));
    setDraftMode(mode);
    setEditing(false);
  };

  const handleSave = () => {
    const saved = projectAssetStore.saveConfig(groupId, isProject ? 'autoSync' : draftMode, draft, '平台管理员');
    setCategories(saved.categories);
    setMode(saved.mode);
    setVersion(saved.version);
    setRecords(projectAssetStore.getUpdateRecords(groupId));
    setEditing(false);
    toast.success(`${term}资产已保存`);
  };

  // 确认添加：将弹窗返回的各大类勾选集合合并进草稿（保留已存在项的版本快照）
  const handleConfirmAdd = (result: Record<AssetCategory, string[]>) => {
    setDraft((prev) => {
      const next = { ...prev };
      ASSET_CATEGORY_ORDER.forEach((category) => {
        const refIds = result[category] || [];
        const prevMap = new Map(prev[category].items.map((i) => [i.refId, i]));
        const libItems = getCategoryLibraryItems(category);
        next[category] = {
          items: refIds.map((refId) => {
            const existing = prevMap.get(refId);
            if (existing) return existing;
            const lib = libItems.find((i) => i.refId === refId);
            return {
              refId,
              versionAtBind: lib?.version || '1.0.0',
              addedAt: new Date().toISOString(),
            };
          }),
        };
      });
      return next;
    });
  };

  const removeItem = (category: AssetCategory, refId: string) => {
    setDraft((prev) => ({
      ...prev,
      [category]: { items: prev[category].items.filter((i) => i.refId !== refId) },
    }));
  };

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* 头部：名称 · N人（虚线，hover 出成员标签，可移除 / 添加） */}
      <div className="flex items-center justify-between gap-3 px-6 py-4 border-b border-[var(--cp-border)]">
        <div className="min-w-0">
          <div className="flex items-center gap-2.5 flex-wrap">
            <h2 className="text-lg font-semibold text-[var(--text-title)] truncate m-0">{groupName}</h2>
            <div className="flex items-center gap-1 text-sm text-[var(--text-muted)] tabular-nums">
              <span aria-hidden="true">·</span>
              <HoverCard openDelay={80} closeDelay={120}>
                <HoverCardTrigger asChild>
                  <button
                    type="button"
                    className="cursor-default border-b border-dashed border-[var(--text-weak)] leading-tight pb-px focus-visible:outline-none"
                  >
                    {members.length} 人
                  </button>
                </HoverCardTrigger>
                <HoverCardContent align="start" sideOffset={6} className="w-[480px] rounded-[4px] p-3">
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <MetaText tone="secondary">
                      {isProject ? '项目成员' : '组织成员'}（{members.length}）
                    </MetaText>
                  </div>
                  <div className="flex flex-wrap gap-1.5 max-h-[300px] overflow-y-auto -mr-1 pr-1">
                    {members.map((u) => {
                      const direct = isProject || u.groupIds.includes(groupId);
                      const hasAlias = !!u.displayName && u.displayName !== u.userId;
                      return (
                        <span
                          key={u.userId}
                          className="inline-flex items-center gap-1 h-6 pl-2 pr-1 rounded-[4px] bg-[var(--color-gray-100)] text-xs text-[var(--text-body)] max-w-full"
                        >
                          {hasAlias ? (
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <span className="truncate max-w-[120px] cursor-default">{u.userId}</span>
                              </TooltipTrigger>
                              <TooltipContent side="top">
                                {u.userId}（{u.displayName}）
                              </TooltipContent>
                            </Tooltip>
                          ) : (
                            <span className="truncate max-w-[120px]">{u.userId}</span>
                          )}
                          {direct ? (
                            <button
                              type="button"
                              onClick={() => setRemoveTarget(u)}
                              className="shrink-0 w-4 h-4 flex items-center justify-center rounded text-[var(--text-weak)] hover:text-[var(--text-danger)] hover:bg-white transition-colors"
                              aria-label={`移除 ${u.userId}`}
                            >
                              <X className="w-3 h-3" />
                            </button>
                          ) : (
                            <span className="shrink-0 pr-1 text-[10px] leading-none text-[var(--text-weak)]">
                              子组织
                            </span>
                          )}
                        </span>
                      );
                    })}
                    {/* 末尾添加按钮 */}
                    <button
                      type="button"
                      onClick={() => setAddMemberOpen(true)}
                      className="inline-flex items-center gap-1 h-6 px-2 rounded-[4px] border border-dashed border-[var(--cp-border-control)] text-xs text-[var(--text-secondary)] hover:border-[var(--cp-brand-blue)] hover:text-[var(--text-brand)] transition-colors"
                    >
                      <Plus className="w-3 h-3" />
                      添加
                    </button>
                  </div>
                </HoverCardContent>
              </HoverCard>
            </div>
          </div>
          <MetaText tone="secondary" className="mt-0.5 block">
            {isProject ? `项目名称：${groupName}` : `组织名称：${getGroupPath(groupId, groups)}`}
          </MetaText>
        </div>
        {isProject && onEditProject && (
          <Button variant="claw-outline" onClick={onEditProject}>
            <Pencil className="h-4 w-4" />
            编辑项目
          </Button>
        )}
      </div>

      {/* Tab 切换；编辑态下「实例」tab 置灰不可点，hover 提示
        * 停服态豁免：切换「资产 / 实例」属于查看类导航（不产生变更），
        * 需保持 100% 不透明与正常交互。
        * "停服前已禁用则延续禁用"：编辑态下「实例」tab 的业务级禁用
        * （opacity-50 + cursor-not-allowed + aria-disabled + preventDefault）
        * 由 SegmentOption 自身控制，data-billing-exempt 不影响其呈现与拦截。*/}
      <div className="px-6 pt-4 flex items-center gap-3">
        <SegmentGroup data-billing-exempt>
          <SegmentOption className="gap-1.5" active={tab === 'assets'} onClick={() => !editing && setTab('assets')}>
            <Boxes className="w-4 h-4" />
            {term}资产
          </SegmentOption>
          {editing ? (
            <HoverCard openDelay={80} closeDelay={80}>
              <HoverCardTrigger asChild>
                <SegmentOption
                  className="gap-1.5 opacity-50 cursor-not-allowed"
                  active={tab === 'instances'}
                  aria-disabled
                  onClick={(e) => e.preventDefault()}
                >
                  <Server className="w-4 h-4" />
                  {term}实例
                </SegmentOption>
              </HoverCardTrigger>
              <HoverCardContent align="start" sideOffset={6} className="w-auto rounded-[4px] px-3 py-2">
                <MetaText tone="secondary" className="whitespace-nowrap">
                  编辑中，保存或取消后可切换标签
                </MetaText>
              </HoverCardContent>
            </HoverCard>
          ) : (
            <SegmentOption className="gap-1.5" active={tab === 'instances'} onClick={() => setTab('instances')}>
              <Server className="w-4 h-4" />
              {term}实例
            </SegmentOption>
          )}
        </SegmentGroup>
      </div>

      {/* 内容 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-6 py-4">
        {tab === 'assets' && (
          <div className="space-y-4">
            {/* 版本信息卡：带框线只读元信息（当前版本 + 查看更新记录同排） */}
            <div className="flex items-center justify-between gap-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-4 py-3">
              <div className="flex items-center gap-2">
                <MetaText tone="secondary">当前版本</MetaText>
                <StatusTag variant="gray" mode="soft">v{version}</StatusTag>
              </div>
              <Button variant="claw-outline" size="sm" onClick={() => setRecordsDrawerOpen(true)}>
                <History className="w-4 h-4" />
                查看更新记录
              </Button>
            </div>

            {/* 资产配置合并卡：卡头承载编辑操作（编辑范围＝同步模式＋资产清单，整体一起编辑）；
                body 顶部为同步模式，分隔线下方为资产清单 */}
            <div className="rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)]">
              <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-[var(--cp-border)]">
                <BodyMedium>资产配置</BodyMedium>
                <div className="flex items-center gap-2 shrink-0">
                  {editing ? (
                    <>
                      <Button variant="claw-outline" size="sm" onClick={handleCancel}>
                        <X className="w-4 h-4" />
                        取消
                      </Button>
                      <Button variant="claw-primary" size="sm" onClick={handleSave}>
                        <Save className="w-4 h-4" />
                        保存
                      </Button>
                    </>
                  ) : (
                    <Button variant="claw-primary" size="sm" onClick={handleEnterEdit}>
                      <Pencil className="w-4 h-4" />
                      编辑
                    </Button>
                  )}
                </div>
              </div>

              <div className="p-4 space-y-4">
                {/* 同步模式：查看态与用户端一致为紧凑标签 + hover 说明；编辑态展开完整配置 */}
                {editing ? (
                  <>
                    <div>
                      <BodyMedium tone="body" className="block mb-2">同步模式</BodyMedium>
                      {!isProject ? (
                        <div className="space-y-2">
                          {(['autoSync', 'initial'] as AssetSyncMode[]).map((m) => {
                            const selected = draftMode === m;
                            return (
                              <button
                                key={m}
                                type="button"
                                onClick={() => setDraftMode(m)}
                                className={`w-full flex items-start gap-2.5 px-3 py-2.5 rounded-[4px] border text-left transition-colors cursor-pointer ${
                                  selected
                                    ? 'border-[var(--cp-brand-blue)] bg-[var(--bg-brand-selected)]'
                                    : 'border-[var(--cp-border-control)] bg-white hover:border-[var(--cp-brand-blue)]'
                                }`}
                              >
                                <span
                                  className={`mt-0.5 shrink-0 w-4 h-4 rounded-full border flex items-center justify-center ${
                                    selected ? 'border-[var(--cp-brand-blue)]' : 'border-[var(--cp-border-control)]'
                                  }`}
                                >
                                  {selected && <span className="w-2 h-2 rounded-full bg-[var(--cp-brand-blue)]" />}
                                </span>
                                <span className="min-w-0 flex flex-col gap-0.5">
                                  <span
                                    className={`text-sm ${
                                      selected ? 'text-[var(--text-brand)] font-medium' : 'text-[var(--text-body)]'
                                    }`}
                                  >
                                    {ASSET_SYNC_MODE_MAP[m].label}
                                  </span>
                                  <span className="text-xs text-[var(--text-muted)] leading-relaxed">
                                    {ASSET_SYNC_MODE_MAP[m].description.replace(/组织/g, term)}
                                  </span>
                                </span>
                              </button>
                            );
                          })}
                        </div>
                      ) : (
                        <div className="flex items-center gap-2">
                          <span className="inline-flex items-center h-7 px-2.5 rounded-[4px] border border-[var(--cp-border)] bg-[var(--color-gray-100)] text-sm text-[var(--text-body)]">
                            {ASSET_SYNC_MODE_MAP[activeMode].label}
                          </span>
                          <MetaText tone="weak">（项目固定，不可更改）</MetaText>
                        </div>
                      )}
                      <MetaText className="block mt-1">
                        {ASSET_SYNC_MODE_MAP[activeMode].description.replace(/组织/g, term)}
                      </MetaText>
                    </div>
                    <div className="border-t border-[var(--cp-border)]" />
                  </>
                ) : (
                  <div className="flex items-center gap-2">
                    <MetaText tone="weak">同步模式：</MetaText>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="inline-flex items-center h-6 px-2 rounded-[4px] border border-[var(--cp-border)] bg-[var(--color-gray-100)] text-xs text-[var(--text-body)] cursor-help">
                          {ASSET_SYNC_MODE_MAP[activeMode].label}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent className="max-w-xs">
                        {ASSET_SYNC_MODE_MAP[activeMode].description.replace(/组织/g, term)}
                      </TooltipContent>
                    </Tooltip>
                  </div>
                )}

                {/* 资产清单：次级小标题；编辑态右侧「添加」，大类为次层级分组小标题 */}
                <div>
                  <div className="flex items-center justify-between gap-2 mb-2">
                    <BodyMedium tone="body">资产清单</BodyMedium>
                    {editing && (
                      <Button variant="claw-outline" size="sm" onClick={() => setAddDialogOpen(true)}>
                        <Plus className="w-4 h-4" />
                        添加
                      </Button>
                    )}
                  </div>
                  {!hasAnyItems ? (
                    <Empty className="py-10">
                      <EmptyHeader>
                        <EmptyDescription>
                          暂无{term}资产，{editing ? '点击「添加」选择配置' : '进入编辑后可添加'}
                        </EmptyDescription>
                      </EmptyHeader>
                    </Empty>
                  ) : (
                    <div className="space-y-4">
                      {ASSET_CATEGORY_ORDER.filter((c) => activeCategories[c].items.length > 0).map(
                        (category) => (
                          <div key={category} className="pl-3">
                            <MetaText tone="secondary" className="block mb-2">
                              {ASSET_CATEGORY_MAP[category].label}
                              <span className="ml-1 text-[var(--text-weak)] tabular-nums">
                                （{activeCategories[category].items.length}）
                              </span>
                            </MetaText>
                            <div className="flex flex-wrap gap-2">
                              {activeCategories[category].items.map((item) => {
                                const display = getAssetItemDisplay(category, item.refId);
                                const meta = display.exists
                                  ? getAssetTagMeta(category, item.refId)
                                  : undefined;
                                return (
                                  <span
                                    key={item.refId}
                                    className="inline-flex items-center gap-1.5 h-7 pl-2.5 pr-1 rounded-[4px] border border-[var(--cp-border)] bg-[var(--color-gray-100)] max-w-full"
                                    title={display.name}
                                  >
                                    <span className="text-sm text-[var(--text-body)] truncate max-w-[220px]">
                                      {display.name}
                                    </span>
                                    {!display.exists ? (
                                      <span className="text-xs text-[var(--text-danger)] shrink-0">
                                        工具库已删除
                                      </span>
                                    ) : (
                                      meta && (
                                        <span className="text-xs text-[var(--text-muted)] tabular-nums shrink-0">
                                          {meta}
                                        </span>
                                      )
                                    )}
                                    {editing ? (
                                      <button
                                        type="button"
                                        onClick={() => removeItem(category, item.refId)}
                                        className="shrink-0 w-5 h-5 flex items-center justify-center rounded-[4px] text-[var(--text-weak)] hover:text-[var(--text-danger)] hover:bg-white transition-colors"
                                        aria-label={`移除 ${display.name}`}
                                      >
                                        <X className="w-3.5 h-3.5" />
                                      </button>
                                    ) : (
                                      <span className="w-1 shrink-0" />
                                    )}
                                  </span>
                                );
                              })}
                            </div>
                          </div>
                        ),
                      )}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}

        {tab === 'instances' && (
          <ProjectInstancesTab groupId={groupId} groups={groups} isProject={isProject} />
        )}
      </div>

      {/* 更新记录抽屉 */}
      <Sheet open={recordsDrawerOpen} onOpenChange={setRecordsDrawerOpen}>
        <SheetContent side="right" className="w-[480px] sm:max-w-[480px] flex flex-col p-0">
          <SheetHeader className="px-6 py-4 border-b border-[var(--cp-border)]">
            <SheetTitle>更新记录</SheetTitle>
          </SheetHeader>
          <div className="flex-1 overflow-y-auto px-6 py-4">
            <UpdateRecordsTab records={records} />
          </div>
        </SheetContent>
      </Sheet>

      {/* 统一添加资产弹窗（多 Tab 勾选 + 企业类上传） */}
      <AddAssetsDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        groupId={groupId}
        groupName={groupName}
        groups={groups}
        selectedRefIds={ASSET_CATEGORY_ORDER.reduce((acc, c) => {
          acc[c] = draft[c].items.map((i) => i.refId);
          return acc;
        }, {} as Record<AssetCategory, string[]>)}
        onConfirm={handleConfirmAdd}
      />

      {/* 添加用户到组织 / 项目弹窗（复用用户管理视图弹窗） */}
      <AddUsersToGroupDialog
        open={addMemberOpen}
        onOpenChange={setAddMemberOpen}
        nodeName={groupName}
        nodeId={groupId}
        allUsers={users}
        groups={groups}
        showDept={false}
        hasOneid={false}
        term={term}
        onConfirm={(userIds) => onAddUsers(userIds)}
      />

      {/* 从组织 / 项目中移除成员弹窗（复用用户管理视图弹窗） */}
      <RemoveUserFromGroupDialog
        userId={removeTarget?.userId ?? null}
        nodeId={groupId}
        nodeName={groupName}
        groups={groups}
        isProject={isProject}
        onClose={() => setRemoveTarget(null)}
        onConfirm={(userId) => onRemoveUser(userId)}
      />
    </div>
  );
}
