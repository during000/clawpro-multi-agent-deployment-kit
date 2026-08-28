import React, { useState } from 'react';
import { ChevronDown, ChevronRight, Plus, X } from 'lucide-react';
import { toast } from 'sonner';
import { useAdminDisabled } from '@/hooks/useAdminDisabled';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { SurfaceCard } from '@/components/ui/Surface';
import { MetaText, MiniBodyText, BodyMedium, CardTitle, HelperText } from '@/components/ui/Typography';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from '@/components/ui/table';
import { GroupSelect } from '@/components/GroupSelect';
import { StatusTag } from '@/components/ui/status-tag';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
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
import { MOCK_GROUPS as MOCK_ONEID_GROUPS, MOCK_MANUAL_GROUPS } from '../../MemberManagement/mock';
import { buildGroupTree, findGroupNode, type GroupTreeNode } from '../../MemberManagement/health';
import type { UserGroup } from '../../MemberManagement/types';

/**
 * 停服态「局部禁用」压制样式。
 *
 * 背景：本页 useMemoryManagementPortalBillingExempt 会给 dialog-content 打
 * [data-billing-exempt]，随后 AdminDisabledOverlay 的恢复规则
 *   body.admin-billing-suspended [data-slot="dialog-content"][data-billing-exempt] *
 *   { opacity:1 !important; cursor:pointer !important; }
 * 会用「后代通配 + !important」把弹窗内所有元素（含我们特意灰化的禁用按钮）
 * 强制恢复成正常态 —— 连内联 style={{opacity:0.4}} 都被 !important 盖掉，
 * 导致「禁用视觉」看不出来、点击也走豁免不弹 toast。
 *
 * 解法（不动组件库 / 不动 overlay）：在页面内注入一段特异性更高的规则，
 * 把带 .suspend-blocked-el 标记的元素重新压回 opacity:0.4 + not-allowed，
 * 并显式 pointer-events:none（外层 .suspend-blocked-wrap 保持可点承接 toast）。
 * 选择器特异性 (0,4,1) > overlay 恢复规则 (0,3,1)，且同带 !important，故胜出。
 */
const SUSPEND_BLOCKED_STYLE_ID = 'memory-suspend-blocked-style';
const SUSPEND_BLOCKED_CSS = `
body.admin-billing-suspended [data-slot="dialog-content"][data-billing-exempt] .suspend-blocked-el,
body.admin-billing-suspended [data-slot="dialog-content"][data-billing-exempt] .suspend-blocked-el * {
  opacity: 0.4 !important;
  cursor: not-allowed !important;
  pointer-events: none !important;
}
body.admin-billing-suspended [data-slot="dialog-content"][data-billing-exempt] .suspend-blocked-wrap {
  cursor: not-allowed !important;
  pointer-events: auto !important;
}
`;

/** 一次性注入压制样式（幂等） */
function ensureSuspendBlockedStyle(): void {
  if (typeof document === 'undefined') return;
  if (document.getElementById(SUSPEND_BLOCKED_STYLE_ID)) return;
  const style = document.createElement('style');
  style.id = SUSPEND_BLOCKED_STYLE_ID;
  style.textContent = SUSPEND_BLOCKED_CSS;
  document.head.appendChild(style);
}

// 默认记忆版本类型
export type DefaultMemoryVersionType = 'none' | 'free' | 'pro';

// 策略规则类型 —— 与网盘管理模块的 PolicyRule<T> 保持一致
export interface MemoryVersionRule {
  id: string;
  groupIds: string[];
  value: DefaultMemoryVersionType;
}

interface DefaultMemoryVersionProps {
  /** 策略规则列表：第 1 条 groupIds 为空数组的为「预设策略」（fallback），其余为组织例外 */
  rules: MemoryVersionRule[];
  /** 规则变化回调；返回 false 可阻止保存 */
  onRulesChange: (rules: MemoryVersionRule[]) => boolean | void;
  /** Pro 服务是否已开通 */
  isProActive: boolean;
  /** Pro 配额是否充足 */
  isProQuotaAvailable: boolean;
}

// ─── 版本视觉映射 ───────────────────────────────────────────────────────────
const VERSION_META: Record<DefaultMemoryVersionType, {
  label: string;
  /** 用于 StatusTag 的 variant */
  tagVariant: 'gray' | 'blue' | 'purple' | 'green' | 'zinc';
}> = {
  none: { label: '关闭', tagVariant: 'zinc' },
  free: { label: 'Free 版', tagVariant: 'gray' },
  pro:  { label: 'Pro 版',  tagVariant: 'blue' },
};

// ─── 版本只读 Tag ───────────────────────────────────────────────────────────
function VersionTag({ value }: { value: DefaultMemoryVersionType }) {
  const meta = VERSION_META[value];
  return (
    <StatusTag mode="fill" variant={meta.tagVariant}>{meta.label}</StatusTag>
  );
}

// ─── 组织选择器（委托给公共 GroupSelect 组件） ─────────────────────────────────
function GroupTagSelector({
  selectedIds,
  disabledIds = [],
  onChange,
  onSave,
}: {
  selectedIds: string[];
  disabledIds?: string[];
  onChange: (ids: string[]) => void;
  onSave?: () => void;
}) {
  const allGroups: UserGroup[] = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];
  return (
    <GroupSelect
      groups={allGroups}
      selectedIds={selectedIds}
      disabledIds={disabledIds}
      onChange={onChange}
      placeholder="选择组织…"
      enableAggregation={false}
      variant={onSave ? "confirm" : "default"}
      onSave={onSave}
    />
  );
}

// ─── 组织名称展示（与 FileManagement.FMGroupBadges 完全对齐） ───────────────
function GroupBadges({ groupIds, maxVisible = 5 }: { groupIds: string[]; maxVisible?: number }) {
  const allGroups: UserGroup[] = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];
  const tree = buildGroupTree(allGroups);
  const paths = groupIds.map((id) => findGroupNode(tree, id)?.path ?? id);

  if (groupIds.length === 0) return <BodyMedium as="span" tone="primary">全部用户</BodyMedium>;

  const visiblePaths = paths.slice(0, maxVisible);
  const rest = paths.length - visiblePaths.length;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex items-center gap-1 flex-wrap cursor-default">
          {visiblePaths.map((name, idx) => (
            <Badge key={`${name}-${idx}`} variant="secondary" className="max-w-[140px]">
              <span className="block truncate max-w-[124px]">{name}</span>
            </Badge>
          ))}
          {rest > 0 && (
            <Badge variant="secondary">+{rest}</Badge>
          )}
        </span>
      </TooltipTrigger>
      <TooltipContent><HelperText as="p" tone="inherit">{paths.join('、')}</HelperText></TooltipContent>
    </Tooltip>
  );
}

// ─── 版本下拉编辑器（用于表格内选择默认版本） ──────────────────────────────
// excludeValues：需要从下拉中隐藏的候选项（用于互斥校验，例如组织规则编辑时排除预设值与其他组织规则已占用值）
function VersionSelect({
  value,
  onChange,
  isProActive,
  isProQuotaAvailable,
  excludeValues = [],
}: {
  value: DefaultMemoryVersionType;
  onChange: (v: DefaultMemoryVersionType) => void;
  isProActive: boolean;
  isProQuotaAvailable: boolean;
  excludeValues?: DefaultMemoryVersionType[];
}) {
  const proDisabled = !isProActive || !isProQuotaAvailable;
  const proDisabledReason = !isProActive
    ? '请先开通 Memory Pro 服务'
    : !isProQuotaAvailable
      ? 'Pro 记忆空间已用完'
      : undefined;

  const allItems: Array<{ key: DefaultMemoryVersionType; disabled?: boolean; reason?: string }> = [
    { key: 'none' },
    { key: 'free' },
    { key: 'pro', disabled: proDisabled, reason: proDisabledReason },
  ];
  const items = allItems.filter((it) => !excludeValues.includes(it.key));

  return (
    <Select value={value} onValueChange={(v) => onChange(v as DefaultMemoryVersionType)}>
      <SelectTrigger size="sm" className="w-[100px] text-xs">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {items.map((it) => {
          const disabled = it.disabled && it.key !== value;
          return (
            <SelectItem key={it.key} value={it.key} disabled={disabled}>
              <span className="inline-flex items-center gap-1">
                {VERSION_META[it.key].label}
                {disabled && it.reason ? <MetaText as="span" tone="weak">（{it.reason}）</MetaText> : null}
              </span>
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
}

// ─── 策略概览卡片（对齐 FileManagement.PolicyOverviewCard 视觉） ─────────────
function PolicyOverviewCard({
  fallbackValue,
  groupCount,
  onClick,
}: {
  fallbackValue: DefaultMemoryVersionType;
  groupCount: number;
  onClick?: () => void;
}) {
  return (
    <SurfaceCard className="overflow-hidden cursor-pointer transition-colors hover:border-[#1447E6]" onClick={onClick}>
      <div className="px-5 pt-5 pb-4 flex-1 min-h-0 flex flex-col">
        <div className="flex items-start gap-3">
          <img
            src="/assets/admin-memory-management/default-memory-version-icon.svg"
            alt=""
            className="shrink-0 w-9 h-9"
          />
          <div className="min-w-0 flex-1">
            <CardTitle as="h3" className="truncate">新建 Agent 默认记忆版本</CardTitle>
            <MetaText as="p" tone="muted" className="leading-relaxed mt-1 line-clamp-2">
              可设置全局「预设策略」，并对指定组织配置不同的默认版本（例如：全局关闭，但对研发组织默认开启 Free 版）。
            </MetaText>
          </div>
        </div>

        {/* 底部灰色摘要条 */}
        <div className="mt-4 rounded-[4px] bg-[var(--bg-grey-hover-subtle)] px-3 py-2 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <MetaText as="span" tone="muted" className="inline-flex items-center gap-1">预设策略：<VersionTag value={fallbackValue} /></MetaText>
            <MetaText as="span" tone="muted">组织策略：<MetaText as="span" tone="primary" className="font-medium">{groupCount} 条</MetaText></MetaText>
          </div>
          <MetaText as="span" tone="brand" className="inline-flex items-center gap-0.5">
            编辑策略<ChevronRight className="w-3.5 h-3.5" />
          </MetaText>
        </div>
      </div>
    </SurfaceCard>
  );
}

/**
 * 新实例默认记忆版本 - 支持「预设策略 + 组织例外」
 *
 * 视觉与交互对齐网盘管理模块的「新增实例是否自动绑定网盘」策略卡片：
 * - 概览卡片展示标题/描述 + 底部摘要条；点击「编辑策略」打开弹窗
 * - 弹窗内用 4 列表格行内编辑：策略类型 / 应用范围 / 默认版本 / 操作
 * - 预设策略（fallback）：唯一 groupIds 为空的规则；切换时若已有组织例外，需二次确认并清空
 * - 组织例外：组织之间互斥；版本值与预设/其他组织互斥（避免冗余）
 */
export const DefaultMemoryVersion: React.FC<DefaultMemoryVersionProps> = ({
  rules,
  onRulesChange,
  isProActive,
  isProQuotaAvailable,
}) => {
  // 停服态：整个弹窗（含关闭 X）保持可用（Portal 豁免 hook 已在页面顶层挂载），
  // 但弹窗内红框区域（"应用范围/默认版本/操作"表格 + "+ 添加组织策略"按钮）需
  // 禁止操作 —— 采用"外层 <span> 承接 click弹 toast + 内层 disabled 按钮"
  // 的分层结构（对齐仓库现有 SecurityGroupManagement.tsx 的做法）。
  // 说明：编辑态下的"取消/保存"按钮此处不做拦截，因为停服态下用户无法进入
  // 编辑态（"编辑"入口已被拦截），编辑态代码分支运行时不可达；即便未来通过
  // 其他路径进入，也应保留"取消"作为退路。
  const { isAdminDisabled } = useAdminDisabled();
  const showSuspendedToast = () => toast.info('管控台已到期，请续费后操作');
  // 注入压制样式，盖过 Portal 豁免恢复规则对禁用按钮的强制还原（详见文件顶部注释）
  if (isAdminDisabled) ensureSuspendBlockedStyle();

  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftGroupIds, setDraftGroupIds] = useState<string[]>([]);
  const [draftValue, setDraftValue] = useState<DefaultMemoryVersionType>('none');
  const [addingNew, setAddingNew] = useState(false);
  const [confirmFallbackDraft, setConfirmFallbackDraft] = useState<DefaultMemoryVersionType | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  const fallbackRule = rules.find((r) => r.groupIds.length === 0)!;
  const groupRules = rules.filter((r) => r.groupIds.length > 0);

  const getDisabledIds = (excludeRuleId?: string) =>
    groupRules.filter((r) => r.id !== excludeRuleId).flatMap((r) => r.groupIds);

  // 版本值上限：Pro 可用时 3 种（关闭/Free/Pro），不可用时 2 种（关闭/Free）
  const maxGroupRules = isProActive && isProQuotaAvailable ? 2 : 1;

  const startEdit = (rule: MemoryVersionRule) => {
    setEditingId(rule.id);
    setDraftGroupIds([...rule.groupIds]);
    setDraftValue(rule.value);
    setAddingNew(false);
  };
  const startAdd = () => {
    setAddingNew(true);
    setEditingId(null);
    setDraftGroupIds([]);
    // 默认草稿值：排除预设值与已被其他组织规则占用的值
    const usedValues = new Set<DefaultMemoryVersionType>([fallbackRule.value, ...groupRules.map((r) => r.value)]);
    const candidates: DefaultMemoryVersionType[] = ['free', 'pro', 'none'];
    const defaultDraft = candidates.find((v) => {
      if (usedValues.has(v)) return false;
      if (v === 'pro' && (!isProActive || !isProQuotaAvailable)) return false;
      return true;
    }) ?? 'none';
    setDraftValue(defaultDraft);
  };
  const cancelEdit = () => { setEditingId(null); setAddingNew(false); };

  const saveEdit = (ruleId?: string) => {
    if (addingNew) {
      if (draftGroupIds.length === 0) { toast.error('请选择至少一个组织'); return; }
      if (draftValue === fallbackRule.value) { toast.error('组织策略需与预设策略不同'); return; }
      if (groupRules.some((r) => r.value === draftValue)) { toast.error('该版本已存在组织策略，请直接编辑现有规则'); return; }
      const result = onRulesChange([
        ...groupRules,
        { id: `mem-rule-${Date.now()}`, groupIds: draftGroupIds, value: draftValue },
        fallbackRule,
      ]);
      if (result === false) return;
      toast.success('策略已保存'); cancelEdit(); return;
    }
    if (!ruleId) return;

    if (ruleId === fallbackRule.id) {
      // 切换预设：若已存在组织例外且预设值实际变更，先弹窗二次确认
      if (draftValue !== fallbackRule.value && groupRules.length > 0) {
        setConfirmFallbackDraft(draftValue);
        return;
      }
      const result = onRulesChange(rules.map((r) => (r.id === ruleId ? { ...r, value: draftValue } : r)));
      if (result === false) return;
      toast.success('策略已保存'); cancelEdit(); return;
    }

    // 编辑组织规则
    if (draftGroupIds.length === 0) { toast.error('请选择至少一个组织'); return; }
    if (draftValue === fallbackRule.value) { toast.error('组织策略需与预设策略不同'); return; }
    if (groupRules.some((r) => r.id !== ruleId && r.value === draftValue)) { toast.error('该版本已存在组织策略，请直接编辑现有规则'); return; }
    const result = onRulesChange(rules.map((r) => (r.id === ruleId ? { ...r, groupIds: draftGroupIds, value: draftValue } : r)));
    if (result === false) return;
    toast.success('策略已保存'); cancelEdit();
  };

  const handleConfirmFallbackSwitch = () => {
    if (confirmFallbackDraft === null) return;
    // 确认切换：清空所有组织例外，仅保留更新后的预设策略
    const result = onRulesChange([{ ...fallbackRule, value: confirmFallbackDraft }]);
    if (result !== false) { toast.success('已更新预设策略，组织策略已清空'); cancelEdit(); }
    setConfirmFallbackDraft(null);
  };

  const deleteRule = (ruleId: string) => {
    const result = onRulesChange(rules.filter((r) => r.id !== ruleId));
    if (result === false) return;
    toast.success('策略已删除');
  };

  return (
    <TooltipProvider>
      {/* ── 概览卡片 ── */}
      <PolicyOverviewCard
        fallbackValue={fallbackRule.value}
        groupCount={groupRules.length}
        onClick={() => setDialogOpen(true)}
      />

      {/* ── 弹窗：表格行内编辑 ── */}
      <Dialog open={dialogOpen} onOpenChange={(v) => { setDialogOpen(v); if (!v) cancelEdit(); }}>
        <DialogContent size="xl">
          <DialogHeader>
            <DialogTitle>新建 Agent 默认记忆版本</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="space-y-3">
              {/* 汇总行 */}
              <div className="flex items-center gap-4 text-[13px]">
                <span className="text-[var(--text-muted)] inline-flex items-center gap-1">预设策略：<VersionTag value={fallbackRule.value} /></span>
                <span className="text-[var(--text-muted)]">组织策略：<span className="text-[var(--text-emphasis)] font-medium">{groupRules.length} 个</span></span>
              </div>

              {/* 表格 */}
              <div className="rounded-[4px] border border-[var(--border)] overflow-hidden">
              <Table density="compact" variant="gray-header">
                <colgroup><col /><col style={{ width: 120 }} /><col style={{ width: 100 }} /></colgroup>
                <TableHeader>
                  <TableRow><TableHead>应用范围</TableHead><TableHead>默认版本</TableHead><TableHead>操作</TableHead></TableRow>
                </TableHeader>
                <TableBody>
                  {/* 预设策略行 */}
                  <TableRow>
                    <TableCell>
                      {groupRules.length > 0 ? '全部用户（组织策略用户除外）' : '全部用户'}
                    </TableCell>
                    <TableCell>
                      {editingId === fallbackRule.id ? (
                        <VersionSelect
                          value={draftValue}
                          onChange={setDraftValue}
                          isProActive={isProActive}
                          isProQuotaAvailable={isProQuotaAvailable}
                        />
                      ) : (
                        <VersionTag value={fallbackRule.value} />
                      )}
                    </TableCell>
                    <TableActionCell>
                      {editingId === fallbackRule.id ? (
                        <>
                          <Button variant="link" onClick={cancelEdit}>取消</Button>
                          <Button variant="link" onClick={() => saveEdit(fallbackRule.id)}>保存</Button>
                        </>
                      ) : isAdminDisabled ? (
                        // 停服态：外层 span 承接点击弹 toast；内层 Button 加 disabled + 强制原色（#355EF1）覆盖 link 变体的 disabled 灰色，
                        // 再统一乘 opacity-40；pointer-events:none 限制在内层，外层 span 保持可点击。
                        <span
                          role="button"
                          onClick={showSuspendedToast}
                          className="suspend-blocked-wrap inline-flex cursor-not-allowed"
                title="管控台已到期，请续费后操作"
                        >
                          <Button
                            variant="link"
                            disabled
                            style={{ pointerEvents: 'none', opacity: 0.4 }}
                            className="suspend-blocked-el !text-[#355EF1] disabled:!text-[#355EF1] disabled:!opacity-100"
                          >
                            编辑
                          </Button>
                        </span>
                      ) : (
                        <Button variant="link" onClick={() => startEdit(fallbackRule)}>编辑</Button>
                      )}
                    </TableActionCell>
                  </TableRow>

                  {/* 组织策略行 */}
                  {groupRules.map((rule, idx) => (
                    <TableRow key={rule.id}>
                      <TableCell>
                        {editingId === rule.id ? (
                          <GroupSelect
                            groups={[...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS]}
                            selectedIds={draftGroupIds}
                            disabledIds={getDisabledIds(rule.id)}
                            onChange={setDraftGroupIds}
                            disabledTooltip="该组织已被其他策略使用"
                            placeholder="选择组织"
                            variant="confirm"
                            onSave={() => saveEdit(rule.id)}
                          />
                        ) : (
                          <GroupBadges groupIds={rule.groupIds} />
                        )}
                      </TableCell>
                      <TableCell>
                        {editingId === rule.id ? (
                          <VersionSelect
                            value={draftValue}
                            onChange={setDraftValue}
                            isProActive={isProActive}
                            isProQuotaAvailable={isProQuotaAvailable}
                            excludeValues={[fallbackRule.value, ...groupRules.filter((r) => r.id !== rule.id).map((r) => r.value)]}
                          />
                        ) : (
                          <VersionTag value={rule.value} />
                        )}
                      </TableCell>
                      <TableActionCell>
                        {editingId === rule.id ? (
                          <>
                            <Button variant="link" onClick={cancelEdit}>取消</Button>
                            <Button variant="link" onClick={() => saveEdit(rule.id)}>保存</Button>
                          </>
                        ) : isAdminDisabled ? (
                          // 停服态：编辑/删除均需拦截；外层 span 弹 toast，内层 Button 视觉禁用（覆盖 link disabled 灰色回品牌蓝再乘 0.4）
                          <>
                            <span
                              role="button"
                              onClick={showSuspendedToast}
                              className="suspend-blocked-wrap inline-flex cursor-not-allowed"
                              title="管控台已到期，请续费后操作"
                            >
                              <Button
                                variant="link"
                                disabled
                                style={{ pointerEvents: 'none', opacity: 0.4 }}
                                className="suspend-blocked-el !text-[#355EF1] disabled:!text-[#355EF1] disabled:!opacity-100"
                              >
                                编辑
                              </Button>
                            </span>
                            <span
                              role="button"
                              onClick={showSuspendedToast}
                              className="suspend-blocked-wrap inline-flex cursor-not-allowed"
                              title="管控台已到期，请续费后操作"
                            >
                              <Button
                                variant="link"
                                disabled
                                style={{ pointerEvents: 'none', opacity: 0.4 }}
                                className="suspend-blocked-el !text-[#355EF1] disabled:!text-[#355EF1] disabled:!opacity-100"
                              >
                                删除
                              </Button>
                            </span>
                          </>
                        ) : (
                          <>
                            <Button variant="link" onClick={() => startEdit(rule)}>编辑</Button>
                            <Button variant="link" onClick={() => deleteRule(rule.id)}>删除</Button>
                          </>
                        )}
                      </TableActionCell>
                    </TableRow>
                  ))}

                  {/* 新增行 */}
                  {addingNew && (
                    <TableRow>
                      <TableCell className="text-[var(--text-muted)]">组织策略{groupRules.length + 1}</TableCell>
                      <TableCell>
                        <GroupSelect
                          groups={[...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS]}
                          selectedIds={draftGroupIds}
                          disabledIds={getDisabledIds()}
                          onChange={setDraftGroupIds}
                          disabledTooltip="该组织已被其他策略使用"
                          placeholder="选择组织"
                          variant="confirm"
                          onSave={() => saveEdit()}
                        />
                      </TableCell>
                      <TableCell>
                        <VersionSelect
                          value={draftValue}
                          onChange={setDraftValue}
                          isProActive={isProActive}
                          isProQuotaAvailable={isProQuotaAvailable}
                          excludeValues={[fallbackRule.value, ...groupRules.map((r) => r.value)]}
                        />
                      </TableCell>
                      <TableActionCell>
                        <Button variant="link" onClick={cancelEdit}>取消</Button>
                        <Button variant="link" disabled={draftGroupIds.length === 0} onClick={() => saveEdit()}>保存</Button>
                      </TableActionCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>

              {/* 添加组织策略按钮 */}
              {groupRules.length < maxGroupRules && (
                isAdminDisabled ? (
                  // 停服态：外层 span 弹 toast + 内层原生 button 视觉禁用（原按钮 hover/border-dashed 视觉保留）
                  <span
                    role="button"
                    onClick={showSuspendedToast}
                    className="suspend-blocked-wrap block w-full cursor-not-allowed"
                    title="管控台已到期，请续费后操作"
                  >
                    <button
                      type="button"
                      disabled
                      style={{ pointerEvents: 'none', opacity: 0.4 }}
                      className="suspend-blocked-el w-full flex items-center justify-center gap-1 px-3 py-2 text-[13px] text-[var(--text-emphasis)] bg-white border-t border-dashed border-[var(--border)]"
                    >
                      <Plus className="w-3.5 h-3.5" />添加组织策略
                    </button>
                  </span>
                ) : (
                  <button
                    type="button"
                    onClick={startAdd}
                    disabled={editingId !== null || addingNew}
                    className="w-full flex items-center justify-center gap-1 px-3 py-2 text-[13px] text-[var(--text-emphasis)] bg-white border-t border-dashed border-[var(--border)] hover:bg-[var(--bg-grey-hover-subtle)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Plus className="w-3.5 h-3.5" />添加组织策略
                  </button>
                )
              )}
              </div>
            </div>
          </DialogBody>
          <DialogFooter />
        </DialogContent>
      </Dialog>

      <AlertDialog open={confirmFallbackDraft !== null} onOpenChange={(o) => { if (!o) setConfirmFallbackDraft(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>切换后将清空组织策略</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription>
            组织策略是基于「预设策略」的例外设置。切换「预设策略」后，现有组织策略将全部清空，需重新添加。
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmFallbackSwitch}>确认切换</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </TooltipProvider>
  );
};
