/**
 * SettingsLibraryTab - 企业设定库
 *
 * 「企业设定库」是 ClawPro 的一项标准平台能力，管理各项目的「团队全局设定」Markdown 文档
 * （即 CLAUDE.md 这类 Agent 每次会话自动加载的团队基础上下文）。下发时会落成各工具
 * 各自的设定文件（CLAUDE.md / CODEBUDDY.md），仅为落盘细节，不影响页面形态。
 *
 * 形态对齐企业资产库其它库（企业插件库 / MCP 库等）：
 *  - 顶部定位文案 + 与其它库区别的小卡片 + 「下发说明」浮层
 *  - 列表页：搜索 + 卡片/列表视图切换 + 新建；字段含 名称 / 适用范围 / 版本 / 更新时间 / 状态
 *  - 行内：编辑应用范围（EditScopePopover）、编辑、启用/下发
 *  - 新建/编辑：Markdown 编辑器 + 实时预览 + frontmatter 表单 + 规范校验（SettingEditDialog）
 */
import { useMemo, useState } from 'react';
import { toast } from 'sonner';
import {
  Search,
  Grid3x3,
  List,
  Send,
  Trash2,
  Info,
  ScrollText,
  Wrench,
  Plug,
  ShieldCheck,
  FolderGit2,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { SegmentGroup, SegmentOption } from '@/components/ui/segment';
import { SurfaceCard } from '@/components/ui/Surface';
import { StatusTag } from '@/components/ui/status-tag';
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty';
import {
  Table,
  TableActionCell,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card';
import { MoreActionsDropdown } from '@/components/ui/more-actions-dropdown';
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
import { MOCK_GROUPS } from './mockData';
import EditScopePopover from './EditScopeDialog';
import SettingEditDialog from './SettingEditDialog';
import SettingDistributeDialog from './SettingDistributeDialog';
import {
  type TeamSetting,
  TEAM_SETTING_STATUS_MAP,
  DISTRIBUTE_TARGETS,
  getProjectName,
  MOCK_TEAM_SETTINGS,
} from './settingsLibraryData';

// 与其它库的区别说明（顶部小卡片）
const LIBRARY_COMPARISONS: Array<{ icon: typeof Wrench; title: string; desc: string }> = [
  { icon: Wrench, title: '技能库', desc: '可按需调用的能力' },
  { icon: ScrollText, title: '规范库', desc: '需严格遵守的硬约束' },
  { icon: Plug, title: 'MCP / 插件库', desc: '可接入的外部工具与数据' },
  { icon: ShieldCheck, title: '设定库', desc: '会话级自动加载的团队基础设定', highlight: true } as never,
];

export default function SettingsLibraryTab() {
  const [searchQuery, setSearchQuery] = useState('');
  const [settings, setSettings] = useState<TeamSetting[]>(MOCK_TEAM_SETTINGS);
  const [viewMode, setViewMode] = useState<'card' | 'list'>('list');
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editingSetting, setEditingSetting] = useState<TeamSetting | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [distributing, setDistributing] = useState<Record<string, boolean>>({});
  const [distributeTarget, setDistributeTarget] = useState<TeamSetting | null>(null);

  const getGroupName = (groupId: string) => MOCK_GROUPS.find((g) => g.id === groupId)?.name || groupId;

  const getScopeLabels = (s: TeamSetting): string[] => {
    if (s.scope === 'public' || !s.groupIds || s.groupIds.length === 0) return ['全部用户'];
    return s.groupIds.map(getGroupName);
  };

  const filtered = useMemo(
    () =>
      settings
        .filter(
          (s) =>
            s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
            s.description.toLowerCase().includes(searchQuery.toLowerCase()),
        )
        .sort((a, b) => b.updatedAt.getTime() - a.updatedAt.getTime()),
    [settings, searchQuery],
  );

  const handleNew = () => {
    setEditingSetting(null);
    setEditDialogOpen(true);
  };

  const handleEdit = (s: TeamSetting) => {
    setEditingSetting(s);
    setEditDialogOpen(true);
  };

  const handleSave = (data: { id?: string; name: string; description: string; projectId: string; version: string; content: string }) => {
    if (data.id) {
      setSettings((prev) =>
        prev.map((s) =>
          s.id === data.id
            ? { ...s, name: data.name, description: data.description, projectId: data.projectId, version: data.version, content: data.content, updatedAt: new Date() }
            : s,
        ),
      );
      toast.success('设定已保存');
    } else {
      const newSetting: TeamSetting = {
        id: `setting-${Date.now()}`,
        name: data.name,
        description: data.description,
        projectId: data.projectId,
        scope: 'public',
        groupIds: [],
        version: data.version,
        status: 'draft',
        updatedAt: new Date(),
        content: data.content,
      };
      setSettings((prev) => [newSetting, ...prev]);
      toast.success('设定已创建（草稿）');
    }
  };

  const openDistribute = (s: TeamSetting) => {
    if (distributing[s.id]) return;
    setDistributeTarget(s);
  };

  const handleConfirmDistribute = (targetIds: string[]) => {
    const s = distributeTarget;
    if (!s) return;
    const targetNames = DISTRIBUTE_TARGETS.filter((t) => targetIds.includes(t.id)).map((t) => t.name).join('、');
    setDistributing((prev) => ({ ...prev, [s.id]: true }));
    toast.success(`已开始向 ${targetNames} 下发`);
    // mock 下发：1.2s 后置为已下发
    setTimeout(() => {
      setSettings((prev) => prev.map((it) => (it.id === s.id ? { ...it, status: 'distributed' as const } : it)));
      setDistributing((prev) => ({ ...prev, [s.id]: false }));
      toast.success(`下发完成，已写入 ${targetNames} 的设定文件`);
    }, 1200);
  };

  const handleConfirmDelete = () => {
    if (!deleteId) return;
    const name = settings.find((s) => s.id === deleteId)?.name || '';
    setSettings((prev) => prev.filter((s) => s.id !== deleteId));
    toast.success(`设定「${name}」已删除`);
    setDeleteId(null);
  };

  const deleteSetting = settings.find((s) => s.id === deleteId);

  return (
    <div className="space-y-4">
      {/* 与其它库区别的小卡片 + 下发说明 */}
      <div className="flex items-start justify-between gap-4">
        <div className="grid grid-cols-4 gap-3 flex-1">
          {LIBRARY_COMPARISONS.map((item) => {
            const Icon = item.icon;
            const highlight = (item as { highlight?: boolean }).highlight;
            return (
              <div
                key={item.title}
                className={`rounded-[4px] border p-3 ${
                  highlight
                    ? 'border-[#C7D7FE] bg-[#E8ECFE]'
                    : 'border-[var(--cp-border)] bg-[var(--cp-surface)]'
                }`}
              >
                <div className="flex items-center gap-1.5 mb-1">
                  <Icon className={`size-3.5 ${highlight ? 'text-[#1447E6]' : 'text-[var(--text-muted)]'}`} />
                  <BodyMedium as="span" tone={highlight ? 'brand' : 'primary'}>{item.title}</BodyMedium>
                </div>
                <MetaText as="p" tone="secondary" className="leading-relaxed">{item.desc}</MetaText>
              </div>
            );
          })}
        </div>
        <HoverCard openDelay={120} closeDelay={150}>
          <HoverCardTrigger asChild>
            <button className="inline-flex shrink-0 items-center gap-1 text-[var(--text-brand)] hover:opacity-80 transition-opacity pt-1">
              <Info className="size-4" />
              <MetaText as="span" tone="brand">下发说明</MetaText>
            </button>
          </HoverCardTrigger>
          <HoverCardContent side="bottom" align="end" className="w-[360px] p-4">
            <div className="space-y-2">
              <BodyMedium as="p" tone="primary">下发后会发生什么？</BodyMedium>
              <MetaText as="p" tone="secondary" className="leading-relaxed">
                下发时，平台会把这份标准设定写入各工具各自的设定文件（如 Claude 的{' '}
                <span className="font-mono text-[var(--text-emphasis)]">CLAUDE.md</span>、CodeBuddy 的{' '}
                <span className="font-mono text-[var(--text-emphasis)]">CODEBUDDY.md</span>）。
              </MetaText>
              <MetaText as="p" tone="secondary" className="leading-relaxed">
                平台仅通过「托管标记块」圈定自己管理的区域，<BodyMedium as="span" tone="primary">不会影响</BodyMedium>用户在文件中的本地自定义内容。
              </MetaText>
            </div>
          </HoverCardContent>
        </HoverCard>
      </div>

      {/* 工具栏 */}
      <div className="flex items-center gap-3">
        <div className="relative min-w-0 flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-weak)] w-4 h-4" />
          <Input
            placeholder="搜索设定名称或描述..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-10"
          />
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {/* 视图切换
            * 停服态豁免：卡片/列表视图切换属于查看类操作（不产生变更），
            * 需保持 100% 不透明与正常交互。data-billing-exempt 透传到
            * SegmentGroup 根<div> 上，命中 AdminDisabledOverlay 的
            * CSS 恢复分支 + 文档级 capture 事件放行；与本页面其他 Tab
            * （SkillListTab / MCPListTab / PluginListTab / StandardsLibraryTab）
            * 的视图切换器行为对齐。
            * "停服前已禁用则延续禁用"：SegmentOption 自身未设置 disabled
            * （此处也未传），若未来传入 disabled，其 `disabled:pointer-events-none`
            * class 依旧生效；overlay 的恢复规则不覆盖 [disabled] / [aria-disabled]，
            * 所以延续禁用的语义天然保留。 */}
          <SegmentGroup data-billing-exempt>
            <SegmentOption active={viewMode === 'card'} onClick={() => setViewMode('card')} title="卡片视图">
              <Grid3x3 className="w-4 h-4" />
            </SegmentOption>
            <SegmentOption active={viewMode === 'list'} onClick={() => setViewMode('list')} title="列表视图">
              <List className="w-4 h-4" />
            </SegmentOption>
          </SegmentGroup>
          <Button variant="claw-primary" size="claw-sm" onClick={handleNew}>
            + 新建设定
          </Button>
        </div>
      </div>

      {/* 空状态 */}
      {filtered.length === 0 && (
        <Empty className="border border-[var(--cp-border)] bg-[var(--cp-surface)]">
          <EmptyHeader>
            <EmptyTitle>暂无团队设定</EmptyTitle>
            <EmptyDescription>新建一份团队设定，为项目 Agent 确立团队身份与全局工作准则。</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button variant="claw-primary" size="claw-sm" onClick={handleNew}>
              + 新建设定
            </Button>
          </EmptyContent>
        </Empty>
      )}

      {/* 卡片视图 */}
      {viewMode === 'card' && filtered.length > 0 && (
        <div className="grid grid-cols-3 gap-4">
          {filtered.map((s) => {
            const dist = distributing[s.id];
            const statusCfg = TEAM_SETTING_STATUS_MAP[s.status];
            return (
              <SurfaceCard
                key={s.id}
                hover
                onClick={() => handleEdit(s)}
                className="relative flex flex-col overflow-hidden p-4 cursor-pointer group"
              >
                <div className="flex items-center gap-2 mb-2">
                  <CardTitle as="h3" tone="primary" className="flex-1 truncate font-semibold group-hover:text-[var(--text-brand)] transition-colors">{s.name}</CardTitle>
                  <StatusTag mode="fill" variant="gray" className="shrink-0">v{s.version}</StatusTag>
                </div>
                <Tooltip delayDuration={1000}>
                  <TooltipTrigger asChild>
                    <BodyText as="p" tone="muted" className="line-clamp-2 mb-4 cursor-default" style={{ minHeight: '2.5rem' }}>{s.description || '-'}</BodyText>
                  </TooltipTrigger>
                  {s.description && s.description.length > 40 && (
                    <TooltipContent side="bottom" className="max-w-[320px]">
                      <MetaText as="p" tone="inherit" className="whitespace-pre-wrap">{s.description}</MetaText>
                    </TooltipContent>
                  )}
                </Tooltip>
                <div className="flex items-center gap-1.5 mb-2">
                  <FolderGit2 className="size-3.5 shrink-0 text-[var(--text-weak)]" />
                  <MetaText as="span" tone="secondary" className="truncate">{getProjectName(s.projectId)}</MetaText>
                </div>
                <div className="flex items-center gap-2 mb-3">
                  <StatusTag mode="text" variant={statusCfg.variant}>{statusCfg.label}</StatusTag>
                  <MetaText as="span" tone="weak">·</MetaText>
                  <MetaText as="span" tone="weak" className="truncate">{getScopeLabels(s).join('、')}</MetaText>
                </div>
                <div className="flex items-center gap-2 pt-3 mt-auto border-t border-[var(--cp-border)]" onClick={(e) => e.stopPropagation()}>
                  <Button variant="claw-outline" size="sm" onClick={() => openDistribute(s)} disabled={dist} className="h-8">
                    <Send className="size-3.5" />
                    {dist ? '下发中' : s.status === 'distributed' ? '重新下发' : '下发'}
                  </Button>
                  <Button variant="claw-outline" size="sm" onClick={() => handleEdit(s)} className="h-8">
                    编辑
                  </Button>
                  <div className="ml-auto">
                    <MoreActionsDropdown
                      triggerType="icon"
                      align="end"
                      items={[
                        {
                          label: '删除',
                          icon: Trash2,
                          onClick: () => setDeleteId(s.id),
                          variant: 'destructive' as const,
                        },
                      ]}
                    />
                  </div>
                </div>
              </SurfaceCard>
            );
          })}
        </div>
      )}

      {/* 列表视图 */}
      {viewMode === 'list' && filtered.length > 0 && (
        <SurfaceCard className="overflow-hidden">
          <Table variant="white" scrollX={1200}>
            <TableHeader>
              <TableRow>
                <TableHead style={{ width: 220, minWidth: 220 }}>名称 / 描述</TableHead>
                <TableHead style={{ width: 120, minWidth: 120 }}>关联项目</TableHead>
                <TableHead style={{ width: 80, minWidth: 80 }}>状态</TableHead>
                <TableHead style={{ width: 160, minWidth: 160 }}>适用范围</TableHead>
                <TableHead style={{ width: 80, minWidth: 80 }}>版本</TableHead>
                <TableHead style={{ width: 120, minWidth: 120 }}>更新时间</TableHead>
                <TableHead fixed="right" style={{ width: 130, minWidth: 130, maxWidth: 130 }}>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((s) => {
                const dist = distributing[s.id];
                const statusCfg = dist
                  ? { label: '下发中', variant: 'blue' as const }
                  : TEAM_SETTING_STATUS_MAP[s.status];
                return (
                  <TableRow key={s.id} onClick={() => handleEdit(s)} className="cursor-pointer">
                    <TableCell>
                      <div className="min-w-0 space-y-1">
                        <BodyMedium as="p" tone="primary" className="truncate">{s.name}</BodyMedium>
                        <Tooltip delayDuration={1000}>
                          <TooltipTrigger asChild>
                            <MetaText as="p" tone="weak" className="line-clamp-1 cursor-default">{s.description || '-'}</MetaText>
                          </TooltipTrigger>
                          {s.description && s.description.length > 30 && (
                            <TooltipContent side="bottom" className="max-w-[400px]">
                              <MetaText as="p" tone="inherit" className="whitespace-pre-wrap">{s.description}</MetaText>
                            </TooltipContent>
                          )}
                        </Tooltip>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1.5 min-w-0">
                        <FolderGit2 className="size-3.5 shrink-0 text-[var(--text-weak)]" />
                        <BodyText as="span" tone="secondary" className="truncate">{getProjectName(s.projectId)}</BodyText>
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusTag mode="text" variant={statusCfg.variant}>{statusCfg.label}</StatusTag>
                    </TableCell>
                    <TableCell onClick={(e) => e.stopPropagation()}>
                      <EditScopePopover
                        groups={MOCK_GROUPS}
                        currentScope={s.scope}
                        currentGroupIds={s.groupIds}
                        scopeLabels={getScopeLabels(s)}
                        isPublic={s.scope === 'public' || s.groupIds.length === 0}
                        onConfirm={(scope, groupIds) => {
                          setSettings((prev) => prev.map((it) => (it.id === s.id ? { ...it, scope, groupIds } : it)));
                          toast.success('适用范围修改成功');
                        }}
                      />
                    </TableCell>
                    <TableCell>
                      <BodyText as="span" tone="secondary">v{s.version}</BodyText>
                    </TableCell>
                    <TableCell>
                      <BodyText as="span" tone="secondary" className="tabular-nums">
                        {s.updatedAt.toLocaleDateString('zh-CN')}
                      </BodyText>
                    </TableCell>
                    <TableActionCell fixed="right" onClick={(e) => e.stopPropagation()}>
                      <Button variant="link" size="sm" onClick={() => openDistribute(s)} disabled={dist}>
                        {dist ? '下发中' : s.status === 'distributed' ? '重新下发' : '下发'}
                      </Button>
                      <Button variant="link" size="sm" onClick={() => handleEdit(s)}>
                        编辑
                      </Button>
                      <MoreActionsDropdown
                        triggerType="text"
                        align="end"
                        items={[
                          {
                            label: '删除',
                            icon: Trash2,
                            onClick: () => setDeleteId(s.id),
                            variant: 'destructive' as const,
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

      {/* 新建 / 编辑弹窗 */}
      <SettingEditDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        setting={editingSetting}
        existingSettings={settings}
        onSave={handleSave}
      />

      {/* 下发弹窗 */}
      <SettingDistributeDialog
        open={!!distributeTarget}
        onOpenChange={(open) => !open && setDistributeTarget(null)}
        setting={distributeTarget}
        onConfirm={handleConfirmDistribute}
      />

      {/* 删除确认 */}
      <AlertDialog open={!!deleteId} onOpenChange={(open) => !open && setDeleteId(null)}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <AlertDialogTitle>删除团队设定</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <div className="space-y-1">
              <BodyText as="p" tone="primary">
                确定要删除设定「<BodyMedium as="span" tone="primary">{deleteSetting?.name}</BodyMedium>」吗？
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
