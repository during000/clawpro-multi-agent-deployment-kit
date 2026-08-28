/**
 * PlatformPolicy - 平台策略页面
 * 基础信息 → 平台策略
 * 包含：用户配额 / 模型配额 / 功能权限开关
 * 全宽卡片布局，每张卡片支持按组织设置多行规则 + 全部用户兜底行
 */
import React, { useState, useMemo, useEffect, useRef } from "react";
import { useLocation } from "wouter";
import { Alert, AlertDescription, AlertOperationInfoIcon, AlertInfoIcon } from "@/components/ui/alert";
import {
  X,
  HelpCircle as _HelpCircle, Info,
  Plus, Search,
  ChevronDown, ChevronRight,
  ExternalLink,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableHeader, TableHead, TableBody, TableRow, TableCell, TableActionCell } from "@/components/ui/table";
import { Card, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Switch } from "@/components/ui/switch";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
import { toast } from "sonner";
import { pushAdminNotification } from "@/lib/adminNotificationStore";
import { StatusTag } from "@/components/ui/status-tag";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useAdminModelsState } from "@/lib/modelConfigStore";
import { MOCK_GROUPS as MOCK_ONEID_GROUPS, MOCK_MANUAL_GROUPS } from "./MemberManagement/mock";
import { buildGroupTree, buildUnifiedGroupTree, type GroupTreeNode } from "./MemberManagement/health";
import type { UserGroup, GroupSource } from "./MemberManagement/types";
import { GroupSelect } from "@/components/GroupSelect";
import {
  PolicyEditCard as BasePolicyEditCard,
  QuotaPolicyCard as BaseQuotaPolicyCard,
  TokenValueEditor,
  type PolicyRule,
  type TokenLimit,
} from "@/components/policy";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { DateTimePicker } from "@/components/ui/datetime-picker";
import { BodyMedium, CardTitle, HelperText, MetaText, SectionTitle } from "@/components/ui/Typography";

// ─── Tokens 时间维度类型 / 常量（cur 自带功能，2026 基底无） ───────────────────
type NaturalPeriod = "daily" | "monthly" | "yearly";
type CustomRefresh = "none" | NaturalPeriod;

type TimeDimensionConfig =
  | { type: "natural"; period: NaturalPeriod }
  | { type: "custom" };

const NATURAL_PERIOD_LABEL: Record<NaturalPeriod, string> = {
  daily: "每日",
  monthly: "每月",
  yearly: "每年",
};
// 自然周期次级说明：用于「周期类型」下拉 Select 项的副文本
const NATURAL_PERIOD_DESCRIPTION: Record<NaturalPeriod, string> = {
  daily: "按自然日 0 点起算",
  monthly: "按自然月 1 号 0 点起算",
  yearly: "按 1 月 1 日 0 点起算",
};
const CUSTOM_REFRESH_LABEL: Record<CustomRefresh, string> = {
  none: "不刷新",
  daily: "每日刷新",
  monthly: "每月刷新",
  yearly: "每年刷新",
};

// 扩展 PolicyRule：自定义周期下行级配置（startAt/endAt/refresh 均 optional，向后兼容原 PolicyRule）
type ExtendedPolicyRule<T> = PolicyRule<T> & {
  startAt?: string;            // ISO 字符串，格式 YYYY-MM-DDTHH:mm:ss
  endAt?: string | null;       // null 表示永不终止；预设策略始终为 null
  refresh?: CustomRefresh;     // 自定义周期内的刷新方式
};

// ─── Tokens 时间维度工具函数 ─────────────────────────────────────────────────
function pad2(n: number): string { return n < 10 ? `0${n}` : String(n); }
function toIsoLocal(d: Date): string {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}`;
}
function parseIsoLocal(s: string | undefined | null): Date | null {
  if (!s) return null;
  const d = new Date(s);
  return isNaN(d.getTime()) ? null : d;
}
function nowPlus(minutes: number): string {
  const d = new Date();
  d.setMinutes(d.getMinutes() + minutes);
  return toIsoLocal(d);
}
function formatDisplay(s: string | null | undefined): string {
  if (!s) return "";
  const d = parseIsoLocal(s);
  if (!d) return s;
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

// ─── DateTimeField（自定义周期 startAt/endAt 编辑） ──────────────────────────
interface DateTimeFieldProps {
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  min?: string;
  disabled?: boolean;
  invalid?: boolean;
  /** 允许清空为空（终止时间「无终止」），开启后选择器面板内显示「设为无终止」蓝色按钮 */
  clearable?: boolean;
  /** 清空按钮文案 */
  clearLabel?: string;
}
function DateTimeField({ value, onChange, placeholder, disabled, clearable, clearLabel }: DateTimeFieldProps) {
  // DateTimePicker 内部用 moment，返回 ISO string；我们需要转为本地格式存储
  const handleChange = (isoVal: string | null) => {
    if (!isoVal) { onChange(""); return; }
    const d = new Date(isoVal);
    if (isNaN(d.getTime())) { onChange(""); return; }
    onChange(toIsoLocal(d));
  };
  // 将本地格式字符串转为 ISO 给 DateTimePicker
  const isoValue = value ? (() => {
    const d = parseIsoLocal(value);
    return d ? d.toISOString() : value;
  })() : null;

  return (
    <DateTimePicker
      value={isoValue}
      onChange={handleChange}
      placeholder={placeholder ?? "选择日期时间"}
      disabled={disabled}
      clearable={clearable}
      clearLabel={clearLabel}
      className="!h-7 text-xs w-full"
    />
  );
}

// ─── PeriodConfigBar（周期类型编辑/展示双态，由父组件 cardEditing 控制） ─────
interface PeriodConfigBarProps {
  config: TimeDimensionConfig;
  onChange: (next: TimeDimensionConfig) => void;
  /** 切换类型前可拦截（用于二次确认 / 数据迁移）。返回 false 表示由 beforeTypeChange 自行处理（弹 AlertDialog 等）。 */
  beforeTypeChange?: (nextType: "natural" | "custom", draftPeriod?: NaturalPeriod) => boolean | Promise<boolean>;
  /** 适用范围：user=单用户，global=全局；用于差异化 tooltip 文案 */
  scope?: "user" | "global";
  /** 父卡片是否处于编辑态：true 时展示 Select，false 时只读文字 */
  cardEditing?: boolean;
  /** 嵌入模式：true 时不渲染自身灰底外壳（背景/边框/圆角由外层统一容器提供），仅输出一行内容 */
  embedded?: boolean;
}
function PeriodConfigBar({ config, onChange, beforeTypeChange, scope = "user", cardEditing = false, embedded = false }: PeriodConfigBarProps) {
  const currentPeriod: NaturalPeriod = config.type === "natural" ? config.period : "daily";

  const handleTypeChange = async (nextType: "natural" | "custom") => {
    if (nextType === config.type) return;
    if (beforeTypeChange) {
      await beforeTypeChange(nextType, currentPeriod);
    } else {
      if (nextType === "natural") onChange({ type: "natural", period: currentPeriod });
      else onChange({ type: "custom" });
    }
  };
  const handlePeriodChange = (nextPeriod: NaturalPeriod) => {
    if (config.type === "natural" && nextPeriod !== config.period) {
      onChange({ type: "natural", period: nextPeriod });
    }
  };

  const displayText = config.type === "natural"
    ? `自然周期 · ${NATURAL_PERIOD_LABEL[config.period]}`
    : "自定义周期";

  return (
    <div className={embedded ? "flex items-center px-4 py-3" : "flex items-center px-4 py-3 rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)]"}>
      {/* label 列：固定 120px（含 16px gap 至值列），与下方"预设策略"行栅格一致 */}
      <div className="w-[120px] shrink-0 flex items-center gap-1">
        <span className="text-xs text-[var(--text-muted)]">周期类型</span>
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="cursor-default"><Info className="w-3.5 h-3.5 text-gray-400" /></span>
          </TooltipTrigger>
          <TooltipContent side="top" className="text-xs max-w-[340px] leading-relaxed bg-white text-[var(--text-secondary)] border border-[var(--border)] shadow-[0_4px_12px_rgba(0,0,0,0.08)] px-4 py-3">
            <p className="mb-1.5"><span className="font-medium text-[var(--text-title)]">自然周期：</span>所有策略起止时间相同，新周期开始时用量自动刷新，上限额度恢复，重新开始统计。</p>
            <p><span className="font-medium text-[var(--text-title)]">自定义周期：</span>每条策略独立配置开始时间、终止时间与刷新方式。</p>
            <p className="mt-1.5 pt-1.5 border-t border-[var(--border)]">
              详细的周期介绍和 Tokens 统计规则请查看
              <a href="https://docs.openclaw.com" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-0.5 text-[var(--cp-brand-blue)] hover:opacity-80 underline underline-offset-2 ml-1 transition-colors">
                说明文档<ExternalLink className="w-3 h-3" />
              </a>
            </p>
          </TooltipContent>
        </Tooltip>
      </div>
      {/* 值列：与下方"预设策略"行的值列起点严格对齐 */}
      <div className="flex items-center gap-3 flex-1 min-w-0">
      {cardEditing ? (
        <>
          <SegmentGroup className="h-7">
            <Tooltip>
              <TooltipTrigger asChild>
                <SegmentOption className="text-xs px-2.5" active={config.type === "natural"} onClick={() => handleTypeChange("natural")}>
                  自然周期
                </SegmentOption>
              </TooltipTrigger>
              <TooltipContent side="top" className="text-xs max-w-[280px] leading-relaxed">
                所有策略起止时间相同，新周期开始时用量自动刷新，上限额度恢复，重新开始统计。
              </TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <SegmentOption className="text-xs px-2.5" active={config.type === "custom"} onClick={() => handleTypeChange("custom")}>
                  自定义周期
                </SegmentOption>
              </TooltipTrigger>
              <TooltipContent side="top" className="text-xs max-w-[280px] leading-relaxed">
                每条策略独立配置开始时间、终止时间与刷新方式
              </TooltipContent>
            </Tooltip>
          </SegmentGroup>
          {config.type === "natural" && (
            <div className="flex items-center text-xs">
              <Select value={config.period} onValueChange={(v) => handlePeriodChange(v as NaturalPeriod)}>
                <SelectTrigger size="sm" className="!h-7 w-[260px] px-3 text-xs bg-background">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent className="min-w-[var(--radix-select-trigger-width)]">
                  {(["daily", "monthly", "yearly"] as const).map((p) => (
                    <SelectItem key={p} value={p} className="text-xs">
                      <span className="inline-flex items-center gap-2">
                        <span>{NATURAL_PERIOD_LABEL[p]}</span>
                        <HelperText as="span" className="leading-tight">{NATURAL_PERIOD_DESCRIPTION[p]}</HelperText>
                      </span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}
        </>
      ) : (
        <span className="text-xs text-[var(--text-title)]">{displayText}</span>
      )}

      </div>
    </div>
  );
}

// ─── 类型 ────────────────────────────────────────────────────────────────────

// 网络管理页默认安全组的本地快照（用于平台策略读/补规则）
type SnapshotInboundRule = {
  id: string;
  source: string;
  protocol: string;
  port: string;
  policy: string;
  remark?: string;
};
type DefaultSecurityGroupSnapshot = {
  id?: string;
  name?: string;
  inboundRules: SnapshotInboundRule[];
};

// 端口字段是否覆盖指定目标端口：支持 "ALL" / "80,443" / "6000-7000" / "6080"
function doesPortCoverTarget(port: string, target: number): boolean {
  const trimmed = (port || "").trim();
  if (!trimmed) return false;
  if (trimmed.toUpperCase() === "ALL") return true;
  if (trimmed.includes(",")) return trimmed.split(",").some((p) => doesPortCoverTarget(p, target));
  if (trimmed.includes("-")) {
    const [s, e] = trimmed.split("-").map((x) => Number(x.trim()));
    if (Number.isFinite(s) && Number.isFinite(e)) return s <= target && target <= e;
    return false;
  }
  return Number(trimmed) === target;
}
// 入站规则是否放通了目标端口（源 0.0.0.0/0、允许、TCP/ALL）
function isInboundRuleCoverPort(rule: SnapshotInboundRule, target: number): boolean {
  if (!rule) return false;
  if (rule.source !== "0.0.0.0/0") return false;
  if (rule.policy !== "允许") return false;
  const proto = (rule.protocol || "").toUpperCase();
  if (proto !== "TCP" && proto !== "ALL") return false;
  return doesPortCoverTarget(rule.port, target);
}

// ─── 组织数据 ─────────────────────────────────────────────────────────────────

const ALL_GROUPS: UserGroup[] = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];

// 预设策略「最新保存时间」——原型 mock，实际应由后端返回最近一次保存的时间戳
const PRESET_SAVED_AT = "2026-06-29T19:12:00";

function getGroupPath(groupId: string, groups: UserGroup[]): string {
  const g = groups.find((x) => x.id === groupId);
  if (!g) return groupId;
  const parts: string[] = [g.name];
  let current = g;
  while (current.parentId) {
    const parent = groups.find((x) => x.id === current.parentId);
    if (!parent) break;
    parts.unshift(parent.name);
    current = parent;
  }
  return parts.join(" / ");
}

// ─── 树工具函数 ──────────────────────────────────────────────────────────────

type CheckState = "checked" | "unchecked" | "indeterminate";

/** 节点本身或其任一祖先被选中 */
function isNodeOrAncestorSelected(
  node: GroupTreeNode,
  selectedIds: Set<string>,
  groupMap: Map<string, UserGroup>
): boolean {
  if (selectedIds.has(node.id)) return true;
  let cur: UserGroup | undefined = groupMap.get(node.id);
  while (cur && cur.parentId) {
    if (selectedIds.has(cur.parentId)) return true;
    cur = groupMap.get(cur.parentId);
  }
  return false;
}

/** 任一子孙被选中（不含自身） */
function hasSelectedDescendant(node: GroupTreeNode, selectedIds: Set<string>): boolean {
  for (const c of node.children) {
    if (selectedIds.has(c.id)) return true;
    if (hasSelectedDescendant(c, selectedIds)) return true;
  }
  return false;
}

/** 三态：本身被选=checked；祖先被选=checked；有子孙被选=indeterminate；其他=unchecked */
function getCheckState(
  node: GroupTreeNode,
  selectedIds: Set<string>,
  groupMap: Map<string, UserGroup>
): CheckState {
  if (selectedIds.has(node.id)) return "checked";
  // 祖先被选中 → 自动视为 checked
  let cur: UserGroup | undefined = groupMap.get(node.id);
  while (cur && cur.parentId) {
    if (selectedIds.has(cur.parentId)) return "checked";
    cur = groupMap.get(cur.parentId);
  }
  if (hasSelectedDescendant(node, selectedIds)) return "indeterminate";
  return "unchecked";
}

function getDescendantIds(node: GroupTreeNode): string[] {
  const ids: string[] = [node.id];
  node.children.forEach((c) => ids.push(...getDescendantIds(c)));
  return ids;
}

/**
 * 递归向上聚合：若某父节点的所有直接可用（非 disabled）子节点都已被选中，
 * 则将这些子节点 id 全部移除，换成该父节点 id。继续向上直到无法再聚合。
 */
function aggregateSelection(
  selected: Set<string>,
  roots: GroupTreeNode[],
  disabledIds: Set<string>
): Set<string> {
  const result = new Set(selected);
  let changed = true;
  while (changed) {
    changed = false;
    const walk = (node: GroupTreeNode) => {
      if (node.children.length === 0) return;
      // 先递归处理子节点（自底向上）
      node.children.forEach(walk);
      // 若本节点尚未被选中
      if (result.has(node.id)) return;
      // 所有直接子节点都必须可聚合：非 disabled 且都已选中
      const hasDisabled = node.children.some((c) => disabledIds.has(c.id));
      if (hasDisabled) return;
      const allSelected = node.children.every((c) => result.has(c.id));
      if (!allSelected) return;
      // 聚合：移除所有直接子节点，加入本节点
      node.children.forEach((c) => result.delete(c.id));
      result.add(node.id);
      changed = true;
    };
    roots.forEach(walk);
  }
  return result;
}

const SOURCE_LABELS: Record<GroupSource, string> = {
  "oneid-dept": "部门",
  "oneid-group": "用户组",
  "manual": "自定义组织",
  "project": "项目",
};
// 选择框内只展示部门和自定义组织（不含用户组）
const SOURCE_ORDER: GroupSource[] = ["oneid-dept", "manual"];

// ─── 组织选择器（委托给公共 GroupSelect 组件） ─────────────────────────────────

function GroupTagSelector({
  selectedIds,
  disabledIds,
  onChange,
}: {
  selectedIds: string[];
  disabledIds: string[];
  onChange: (ids: string[]) => void;
}) {
  return (
    <GroupSelect
      groups={ALL_GROUPS.filter((g) => SOURCE_ORDER.includes(g.source))}
      selectedIds={selectedIds}
      disabledIds={disabledIds}
      disabledTooltip="该组织已设置策略，每个组织只允许有一个平台策略"
      onChange={onChange}
      placeholder="请选择组织"
      sourceFilter={SOURCE_ORDER}
      sourceLabels={SOURCE_LABELS as Partial<Record<GroupSource, string>>}
      compactTrigger
    />
  );
}

// ─── 组织名称展示（保存后的只读态：独立 tag + 最多 5 个 + 溢出 +N） ──────────

function GroupBadges({ groupIds, maxVisible = 5 }: { groupIds: string[]; maxVisible?: number }) {
  const paths = useMemo(
    () => groupIds.map((id) => getGroupPath(id, ALL_GROUPS)),
    [groupIds]
  );

  if (groupIds.length === 0) return <span className="text-xs font-medium text-[var(--text-muted)]">预设策略</span>;
  const visible = paths.slice(0, maxVisible);
  const overflow = paths.length - maxVisible;
  const tooltipText = paths.join("\n");

  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      {visible.map((name, i) => (
        <Badge key={i} variant="secondary" className="max-w-[160px] cursor-default">
          <span className="truncate">{name}</span>
        </Badge>
      ))}
      {overflow > 0 && (
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge variant="secondary" className="cursor-default">+{overflow}</Badge>
          </TooltipTrigger>
          <TooltipContent side="right" align="start" className="max-w-[360px] text-xs leading-relaxed whitespace-pre-line">
            {tooltipText}
          </TooltipContent>
        </Tooltip>
      )}
    </div>
  );
}

// ─── 策略卡片：组织渲染 render prop 适配 ─────────────────────────────────────
// PolicyEditCard / QuotaPolicyCard 通过 render prop 接收组织渲染逻辑，
// 此处把本页面的 GroupTagSelector / GroupBadges 适配为标准接口，
// 并通过 LocalTogglePolicyCard / LocalQuotaPolicyCard 自动注入，避免每处调用都重复传 props。
const policyRenderProps = {
  renderGroupSelector: ({
    selectedIds,
    disabledIds,
    onChange,
  }: {
    selectedIds: string[];
    disabledIds: string[];
    onChange: (ids: string[]) => void;
  }) => (
    <GroupTagSelector
      selectedIds={selectedIds}
      disabledIds={disabledIds}
      onChange={onChange}
    />
  ),
  renderGroupBadges: (groupIds: string[]) => <GroupBadges groupIds={groupIds} />,
};

/** 本页内的策略编辑卡片：自动注入组织渲染逻辑。等价于 <BasePolicyEditCard {...policyRenderProps} ... /> */
function TogglePolicyCard(props: Omit<React.ComponentProps<typeof BasePolicyEditCard>, "renderGroupSelector" | "renderGroupBadges">) {
  return <BasePolicyEditCard {...policyRenderProps} {...props} />;
}

/** 本页内的配额策略卡片：自动注入组织渲染逻辑。 */
function QuotaPolicyCard(props: Omit<React.ComponentProps<typeof BaseQuotaPolicyCard>, "renderGroupSelector" | "renderGroupBadges">) {
  return <BaseQuotaPolicyCard {...policyRenderProps} {...props} />;
}

// ─── TokensQuotaCardWithPeriod ─────────────────────────────────────────────
//
// 单/全局 Tokens 上限专用卡片：在 2026 风格基础上叠加自然 / 自定义周期双模式 +
// AlertDialog 类型切换二次确认。仅 token 类型可用。
// - 视觉骨架：Card 容器 + 头部（icon/title/desc + 编辑按钮）+ PeriodConfigBar + 表格区
// - 阅读态：渲染只读表格（按 timeDimension.type 切换）
// - 编辑态：cardEditing=true，全部行可编辑，底部固定 取消/保存
// - 周期类型切换：仅在卡片编辑态下可改（PeriodConfigBar 接收 cardEditing=true）；
//   切换类型时 AlertDialog 二次确认 + 数据迁移
// ───────────────────────────────────────────────────────────────────────────
interface TokensQuotaCardWithPeriodProps {
  icon: React.ReactNode;
  title: string;
  description: string;
  rules: ExtendedPolicyRule<TokenLimit>[];
  onRulesChange: (rules: ExtendedPolicyRule<TokenLimit>[]) => void;
  timeDimension: TimeDimensionConfig;
  onTimeDimensionChange: (next: TimeDimensionConfig) => void;
  /** 适用范围：user=单用户，global=全局；用于差异化文案 */
  scope?: "user" | "global";
  extraContent?: React.ReactNode;
}

function TokensQuotaCardWithPeriod({
  icon, title, description, rules, onRulesChange,
  timeDimension, onTimeDimensionChange, scope = "user", extraContent,
}: TokensQuotaCardWithPeriodProps) {
  const isCustom = timeDimension.type === "custom";

  // 切换到自定义周期时，为缺失时间字段的策略填充默认值
  useEffect(() => {
    if (!isCustom) return;
    const needsFill = rules.some(
      (r) => r.startAt === undefined || r.refresh === undefined
    );
    if (!needsFill) return;
    const defaultStart = nowPlus(1);
    const next = rules.map((r) => {
      if (r.refresh !== undefined) return r;
      const isFb = r.groupIds.length === 0;
      return {
        ...r,
        startAt: isFb ? undefined : (r.startAt ?? defaultStart),
        endAt: isFb ? null : (r.endAt ?? null),
        refresh: (r.refresh ?? "daily") as CustomRefresh,
      };
    });
    onRulesChange(next);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isCustom]);

  // 卡片级编辑态
  const [cardEditing, setCardEditing] = useState(false);
  const [editRules, setEditRules] = useState<ExtendedPolicyRule<TokenLimit>[]>([]);
  const [editValueStrs, setEditValueStrs] = useState<Record<string, string>>({});
  const [editModes, setEditModes] = useState<Record<string, "custom" | "unlimited">>({});

  const fallbackRule = rules.find((r) => r.groupIds.length === 0)!;
  const groupRules = rules.filter((r) => r.groupIds.length > 0);

  const displayValue = (v: TokenLimit) => {
    if (v === "unlimited" || v === -1) return "无限制";
    return Number(v).toLocaleString();
  };

  const buildBlankGroupRule = (): ExtendedPolicyRule<TokenLimit> => {
    const base: ExtendedPolicyRule<TokenLimit> = {
      id: `rule-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
      groupIds: [],
      value: 100000,
    };
    if (isCustom) {
      base.startAt = nowPlus(1);
      base.endAt = null;
      base.refresh = "daily";
    }
    return base;
  };

  const startCardEdit = () => {
    let initial: ExtendedPolicyRule<TokenLimit>[] = [...rules];
    if (!initial.some((r) => r.groupIds.length > 0)) {
      const blank = buildBlankGroupRule();
      const fbIdx = initial.findIndex((r) => r.id === fallbackRule.id);
      initial = [...initial.slice(0, fbIdx), blank, ...initial.slice(fbIdx)];
    }
    const strs: Record<string, string> = {};
    const modes: Record<string, "custom" | "unlimited"> = {};
    initial.forEach((r) => {
      strs[r.id] = r.value === "unlimited" ? "" : String(r.value);
      modes[r.id] = r.value === "unlimited" ? "unlimited" : "custom";
    });
    setEditRules(initial);
    setEditValueStrs(strs);
    setEditModes(modes);
    setCardEditing(true);
  };

  const cancelCardEdit = () => {
    setCardEditing(false);
    setEditRules([]);
    setEditValueStrs({});
    setEditModes({});
  };

  const saveCardEdit = () => {
    const finalRules: ExtendedPolicyRule<TokenLimit>[] = [];
    for (const r of editRules) {
      const isFb = r.id === fallbackRule.id;
      const mode = editModes[r.id] ?? "custom";
      const valStr = editValueStrs[r.id] ?? "";
      let finalValue: TokenLimit;
      if (mode === "unlimited") {
        finalValue = "unlimited";
      } else {
        const n = parseInt(valStr, 10);
        if (isNaN(n) || n < 0) {
          toast.error(`请输入有效数值（${isFb ? "预设策略" : "组织策略"}）`);
          return;
        }
        finalValue = n;
      }
      if (!isFb && r.groupIds.length === 0) {
        // 空白行未填组织，跳过
        continue;
      }
      // 自定义周期：校验时间字段（仅非预设）
      if (isCustom && !isFb) {
        if (!r.startAt) { toast.error("请填写开始时间（组织策略）"); return; }
        const startD = parseIsoLocal(r.startAt);
        if (!startD) { toast.error("开始时间格式不正确"); return; }
        if (r.endAt) {
          const endD = parseIsoLocal(r.endAt);
          if (!endD) { toast.error("终止时间格式不正确"); return; }
          if (endD.getTime() <= startD.getTime()) { toast.error("终止时间必须晚于开始时间"); return; }
        }
      }
      finalRules.push({ ...r, value: finalValue });
    }
    const finalGroupRules = finalRules.filter((r) => r.id !== fallbackRule.id);
    const finalFb = finalRules.find((r) => r.id === fallbackRule.id)!;
    onRulesChange([...finalGroupRules, finalFb]);
    toast.success("策略已保存");
    cancelCardEdit();
  };

  const updateGroups = (id: string, groupIds: string[]) =>
    setEditRules((prev) => prev.map((r) => (r.id === id ? { ...r, groupIds } : r)));
  const updateValueStr = (id: string, valStr: string) =>
    setEditValueStrs((prev) => ({ ...prev, [id]: valStr }));
  const updateMode = (id: string, mode: "custom" | "unlimited") =>
    setEditModes((prev) => ({ ...prev, [id]: mode }));
  const updateStartAt = (id: string, v: string) =>
    setEditRules((prev) => prev.map((r) => (r.id === id ? { ...r, startAt: v } : r)));
  const updateEndAt = (id: string, v: string) =>
    setEditRules((prev) => prev.map((r) => (r.id === id ? { ...r, endAt: v || null } : r)));
  const updateRefresh = (id: string, v: CustomRefresh) =>
    setEditRules((prev) => prev.map((r) => (r.id === id ? { ...r, refresh: v } : r)));

  const removeRule = (id: string) => {
    setEditRules((prev) => prev.filter((r) => r.id !== id));
    setEditValueStrs((prev) => { const { [id]: _omit, ...rest } = prev; return rest; });
    setEditModes((prev) => { const { [id]: _omit, ...rest } = prev; return rest; });
  };

  const addBlankGroupRow = () => {
    const blank = buildBlankGroupRule();
    setEditRules((prev) => {
      const fbIdx = prev.findIndex((r) => r.id === fallbackRule.id);
      return [...prev.slice(0, fbIdx), blank, ...prev.slice(fbIdx)];
    });
    setEditValueStrs((prev) => ({ ...prev, [blank.id]: "100000" }));
    setEditModes((prev) => ({ ...prev, [blank.id]: "custom" }));
  };

  const getDisabledIds = (excludeRuleId: string) =>
    editRules.filter((r) => r.groupIds.length > 0 && r.id !== excludeRuleId).flatMap((r) => r.groupIds);

  const renderValueEditor = (ruleId: string, compact?: boolean) => {
    const mode = editModes[ruleId] ?? "custom";
    const valStr = editValueStrs[ruleId] ?? "";
    return (
      <TokenValueEditor
        mode={mode}
        valStr={valStr}
        onCommit={(nextMode, nextValStr) => {
          updateMode(ruleId, nextMode);
          updateValueStr(ruleId, nextValStr);
        }}
        {...(compact ? { triggerClassName: "group relative w-full h-7 px-3 pr-8 rounded-[4px] border border-[var(--border)] bg-background hover:border-[var(--cp-brand-blue)] data-[state=open]:border-[var(--cp-brand-blue)] transition-colors cursor-pointer flex items-center text-left text-xs" } : {})}
      />
    );
  };

  // ─── 周期类型切换二次确认 ─────────────────────────────────────────────
  const [pendingTypeSwitch, setPendingTypeSwitch] = useState<"natural" | "custom" | null>(null);
  const [pendingDraftPeriod, setPendingDraftPeriod] = useState<NaturalPeriod>("daily");

  const handleBeforeTypeChange = async (nextType: "natural" | "custom", draftPeriod?: NaturalPeriod): Promise<boolean> => {
    if (nextType === "natural" && draftPeriod) {
      setPendingDraftPeriod(draftPeriod);
    }
    setPendingTypeSwitch(nextType);
    return false;
  };

  const confirmTypeSwitch = () => {
    const nextType = pendingTypeSwitch!;
    if (nextType === "custom") {
      const currentPeriod: NaturalPeriod = timeDimension.type === "natural" ? timeDimension.period : "daily";
      const mappedRefresh: CustomRefresh = currentPeriod;
      onRulesChange([{ ...fallbackRule, startAt: undefined, endAt: null, refresh: mappedRefresh }]);
      onTimeDimensionChange({ type: "custom" });
    } else {
      const { startAt: _s, endAt: _e, refresh: _r, ...cleanFb } = fallbackRule;
      onRulesChange([cleanFb as ExtendedPolicyRule<TokenLimit>]);
      onTimeDimensionChange({ type: "natural", period: pendingDraftPeriod });
    }
    setPendingTypeSwitch(null);
  };

  const editFallback = editRules.find((r) => r.id === fallbackRule.id);
  const editGroupRules = editRules.filter((r) => r.id !== fallbackRule.id);

  // 周期类型行（嵌入模式，无自身外壳）——独立浅蓝灰底卡片
  const periodConfigBarEmbedded = (
    <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-hidden">
      <PeriodConfigBar
        config={timeDimension}
        onChange={onTimeDimensionChange}
        beforeTypeChange={handleBeforeTypeChange}
        scope={scope}
        cardEditing={cardEditing}
        embedded
      />
    </div>
  );

  // ─── 渲染：自然态阅读 ──────────────────────────────────────────────
  const renderNaturalReadOnly = () => {
    const periodLabel = timeDimension.type === "natural" ? NATURAL_PERIOD_LABEL[timeDimension.period] : "";
    return (
      <div className="space-y-2">
        {/* 预设策略 —— 列宽与下方组织策略表格对齐：标题 120 / 配额 160 / 应用范围 */}
        <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-x-auto px-4 py-3">
          <div className="flex items-center min-w-[480px]">
            <div className="w-[120px] shrink-0 text-xs text-[var(--text-title)] font-medium">预设策略</div>
            <span className="w-[160px] shrink-0 text-xs text-[var(--text-emphasis)] font-semibold tabular-nums">
              {displayValue(fallbackRule.value)}{periodLabel && ` / ${periodLabel}`}
            </span>
            <Badge variant="outline" className="cursor-default">
              全部用户{groupRules.length > 0 && <span className="text-[var(--text-muted)] ml-1.5">组织策略除外</span>}
            </Badge>
          </div>
        </div>
        {/* 组织策略 —— 3 列：组织策略（序号）/ 配额 / 组织（隐藏表头） */}
        {groupRules.length > 0 && (
          <div className="rounded-[4px] bg-white border border-[var(--border)] overflow-hidden">
            <Table density="compact" variant="white" autoFixedColumns={false} className="table-fixed">
              <colgroup>
                <col className="w-[120px]" />
                <col className="w-[160px]" />
                <col />
              </colgroup>
              <TableBody>
                {groupRules.map((rule, idx) => (
                  <TableRow key={rule.id} className={`hover:bg-transparent [&:hover_td]:!bg-transparent ${idx < groupRules.length - 1 ? "border-b border-[var(--border)]" : "border-0"}`}>
                    <TableCell className="text-xs !text-[#4B5563] tabular-nums">组织策略 {idx + 1}</TableCell>
                    <TableCell className="text-xs !text-[#4B5563] tabular-nums">
                      {displayValue(rule.value)}{periodLabel && ` / ${periodLabel}`}
                    </TableCell>
                    <TableCell className="text-xs !text-[#4B5563]"><GroupBadges groupIds={rule.groupIds} maxVisible={2} /></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    );
  };

  // ─── 渲染：自定义态阅读 ──────────────────────────────────────────────
  const renderCustomReadOnly = () => {
    return (
      <div className="space-y-2">
        {/* 预设策略 —— 使用 Table 布局与组织策略表格列宽完全对齐 */}
        <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-x-auto">
          <Table density="compact" autoFixedColumns={false} className="table-fixed min-w-[920px] bg-transparent">
            <colgroup>
              <col className="w-[120px]" />
              <col className="w-[160px]" />
              <col />
              <col className="w-[180px]" />
              <col className="w-[140px]" />
              <col className="w-[120px]" />
            </colgroup>
            <TableBody>
              <TableRow className="hover:bg-transparent [&:hover_td]:!bg-transparent border-0">
                <TableCell className="text-[var(--text-title)] font-medium">预设策略</TableCell>
                <TableCell className="text-[var(--text-emphasis)] font-semibold tabular-nums">{displayValue(fallbackRule.value)}</TableCell>
                <TableCell>
                  <Badge variant="outline" className="cursor-default whitespace-nowrap">
                    全部用户{groupRules.length > 0 && <span className="text-[var(--text-muted)] ml-1.5">组织策略除外</span>}
                  </Badge>
                </TableCell>
                <TableCell className="text-xs text-[var(--text-title)]">
                  {scope === "global" ? (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="border-b border-dashed border-[var(--text-muted)] cursor-pointer tabular-nums">{formatDisplay(PRESET_SAVED_AT)}</span>
                      </TooltipTrigger>
                      <TooltipContent side="top" className="text-xs max-w-[280px] leading-relaxed bg-white text-[var(--text-body)] border border-[var(--border)] shadow-[0_4px_12px_rgba(0,0,0,0.08)] px-3 py-2">
                        以"预设策略最新保存时间"为开始时间
                      </TooltipContent>
                    </Tooltip>
                  ) : (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="border-b border-dashed border-[var(--text-muted)] cursor-pointer">无统一开始时间</span>
                      </TooltipTrigger>
                      <TooltipContent side="top" className="text-xs max-w-[280px] leading-relaxed bg-white text-[var(--text-body)] border border-[var(--border)] shadow-[0_4px_12px_rgba(0,0,0,0.08)] px-3 py-2">
                        <div className="space-y-2.5">
                          <div className="space-y-0.5">
                            <p className="font-semibold text-[var(--text-emphasis)]">无组织的新用户</p>
                            <p className="text-[var(--text-secondary)]">以"添加用户时间"为开始时间，可在用户管理页对单个用户进行再次调整</p>
                          </div>
                          <div className="space-y-0.5">
                            <p className="font-semibold text-[var(--text-emphasis)]">未匹配任何组织策略的组织</p>
                            <p className="text-[var(--text-secondary)]">以"预设策略最新保存时间"为开始时间，当前为{formatDisplay(PRESET_SAVED_AT)}</p>
                          </div>
                        </div>
                      </TooltipContent>
                    </Tooltip>
                  )}
                </TableCell>
                <TableCell className="text-xs text-[var(--text-title)]">无终止时间</TableCell>
                <TableCell className="text-xs text-[var(--text-title)]">{CUSTOM_REFRESH_LABEL[fallbackRule.refresh ?? "daily"]}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </div>
        {/* 组织策略：表格 —— 列：组织策略（序号）/ 配额 / 组织 / 开始时间 / 终止时间 / 刷新方式 */}
        {groupRules.length > 0 && (
          <div className="rounded-[4px] bg-white border border-[var(--border)] overflow-x-auto">
            <Table density="compact" variant="white" autoFixedColumns={false} className="table-fixed min-w-[920px]">
              <colgroup>
                <col className="w-[120px]" />
                <col className="w-[160px]" />
                <col />
                <col className="w-[180px]" />
                <col className="w-[140px]" />
                <col className="w-[120px]" />
              </colgroup>
              <TableBody>
                {groupRules.map((rule, idx) => (
                  <TableRow key={rule.id} className={`hover:bg-transparent [&:hover_td]:!bg-transparent ${idx < groupRules.length - 1 ? "border-b border-[var(--border)]" : "border-0"}`}>
                    <TableCell className="text-xs !text-[#4B5563] tabular-nums">组织策略 {idx + 1}</TableCell>
                    <TableCell className="text-xs !text-[#4B5563] tabular-nums">{displayValue(rule.value)}</TableCell>
                    <TableCell className="text-xs !text-[#4B5563]"><GroupBadges groupIds={rule.groupIds} maxVisible={2} /></TableCell>
                    <TableCell className="text-xs !text-[#4B5563] tabular-nums">{formatDisplay(rule.startAt)}{rule.startAt ? " 开始" : ""}</TableCell>
                    <TableCell className="text-xs !text-[#4B5563] tabular-nums">{rule.endAt ? `${formatDisplay(rule.endAt)} 终止` : "无终止时间"}</TableCell>
                    <TableCell className="text-xs !text-[#4B5563]">{CUSTOM_REFRESH_LABEL[rule.refresh ?? "none"]}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    );
  };

  // ─── 渲染：自然态编辑 ──────────────────────────────────────────────
  const renderNaturalEditing = () => (
    <div className="space-y-2">
      {/* 周期类型 —— 独立灰底卡片 */}
      {periodConfigBarEmbedded}
      {/* 预设策略 —— 列宽与下方组织策略表格对齐：标题 120 / 配额 160 / 应用范围 */}
      {editFallback && (
        <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-hidden px-4 py-3">
          <div className="flex items-center">
            <div className="w-[120px] shrink-0 text-xs text-[var(--text-title)] font-medium">预设策略</div>
            <div className="w-[160px] shrink-0 flex items-center">
              <TokenValueEditor
                mode={editModes[editFallback.id] ?? "custom"}
                valStr={editValueStrs[editFallback.id] ?? ""}
                onCommit={(nextMode, nextValStr) => {
                  updateMode(editFallback.id, nextMode);
                  updateValueStr(editFallback.id, nextValStr);
                }}
                triggerClassName="group relative h-7 w-[128px] px-3 pr-8 rounded-[4px] border border-[var(--border)] bg-background hover:border-[var(--cp-brand-blue)] data-[state=open]:border-[var(--cp-brand-blue)] transition-colors cursor-pointer flex items-center text-left text-xs"
              />
            </div>
            <Badge variant="outline" className="cursor-default">
              全部用户{editGroupRules.length > 0 && <span className="text-[var(--text-muted)] ml-1.5">组织策略除外</span>}
            </Badge>
          </div>
        </div>
      )}
      {/* 组织策略 + 添加按钮 —— 4 列：组织策略（序号）/ 配额 / 组织 / 操作（隐藏表头） */}
      <div className="rounded-[4px] bg-white border border-[var(--border)] overflow-hidden">
        {editGroupRules.length > 0 && (
          <Table density="compact" variant="white" autoFixedColumns={false} className="table-fixed">
            <colgroup>
              <col className="w-[120px]" />
              <col className="w-[160px]" />
              <col />
              <col className="w-[80px]" />
            </colgroup>
            <TableBody>
              {editGroupRules.map((rule, idx) => (
                <TableRow key={rule.id} className={idx < editGroupRules.length - 1 ? "border-b border-[var(--border)]" : "border-0"}>
                  <TableCell className="text-xs text-[var(--text-body)] tabular-nums">组织策略 {idx + 1}</TableCell>
                  <TableCell>{renderValueEditor(rule.id, true)}</TableCell>
                  <TableCell>
                    <GroupTagSelector
                      selectedIds={rule.groupIds}
                      disabledIds={getDisabledIds(rule.id)}
                      onChange={(ids) => updateGroups(rule.id, ids)}
                    />
                  </TableCell>
                  <TableActionCell>
                    <Button variant="link" size="sm" onClick={() => removeRule(rule.id)}>删除</Button>
                  </TableActionCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
        <button
          type="button"
          onClick={addBlankGroupRow}
          className={`w-full flex items-center justify-center gap-1 px-3 py-2.5 text-xs text-[var(--text-muted)] ${editGroupRules.length > 0 ? "border-t border-dashed border-[var(--border)]" : ""} hover:text-[var(--text-emphasis)] hover:bg-[var(--bg-grey-hover-subtle)] transition-colors`}
        >
          <Plus className="w-3.5 h-3.5" />添加组织策略
        </button>
      </div>
    </div>
  );

  // ─── 渲染：自定义态编辑 ──────────────────────────────────────────────
  const renderCustomEditing = () => {
    return (
      <div className="space-y-2">
        {/* 周期类型 —— 独立灰底卡片 */}
        {periodConfigBarEmbedded}
        {/* 预设策略 —— 单行，列宽与下方组织策略表格对齐：标题 120 / 配额 160 / 应用范围（填充）/ 开始时间 200 / 终止时间 200 / 刷新方式 160 / [操作 80 空位]。每列内 px-4 模拟 td padding */}
        {editFallback && (
        <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-x-auto py-3">
          <div className="flex items-center min-w-[1100px]">
            <div className="w-[100px] shrink-0 px-4 text-xs text-[var(--text-title)] font-medium">预设策略</div>
            <div className="w-[160px] shrink-0 px-4 flex items-center">
              <TokenValueEditor
                mode={editModes[editFallback.id] ?? "custom"}
                valStr={editValueStrs[editFallback.id] ?? ""}
                onCommit={(nextMode, nextValStr) => {
                  updateMode(editFallback.id, nextMode);
                  updateValueStr(editFallback.id, nextValStr);
                }}
                triggerClassName="group relative !h-7 w-full px-3 pr-8 rounded-[4px] border border-[var(--border)] bg-background hover:border-[var(--cp-brand-blue)] data-[state=open]:border-[var(--cp-brand-blue)] transition-colors cursor-pointer flex items-center text-left text-xs"
              />
            </div>
            <div className="w-[200px] shrink-0 px-4">
              <Badge variant="outline" className="cursor-default whitespace-nowrap">
                全部用户{editGroupRules.length > 0 && <span className="text-[var(--text-muted)] ml-1.5">组织策略除外</span>}
              </Badge>
            </div>
            <div className="w-[180px] shrink-0 px-4 text-xs text-[var(--text-title)]">
              {scope === "global" ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="border-b border-dashed border-[var(--text-muted)] cursor-pointer tabular-nums">{formatDisplay(PRESET_SAVED_AT)}</span>
                  </TooltipTrigger>
                  <TooltipContent side="top" className="text-xs max-w-[280px] leading-relaxed bg-white text-[var(--text-body)] border border-[var(--border)] shadow-[0_4px_12px_rgba(0,0,0,0.08)] px-3 py-2">
                    以"预设策略最新保存时间"为开始时间
                  </TooltipContent>
                </Tooltip>
              ) : (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="border-b border-dashed border-[var(--text-muted)] cursor-pointer">无统一开始时间</span>
                  </TooltipTrigger>
                  <TooltipContent side="top" className="text-xs max-w-[280px] leading-relaxed bg-white text-[var(--text-body)] border border-[var(--border)] shadow-[0_4px_12px_rgba(0,0,0,0.08)] px-3 py-2">
                    <div className="space-y-2.5">
                      <div className="space-y-0.5">
                        <p className="font-semibold text-[var(--text-emphasis)]">无组织的新用户</p>
                        <p className="text-[var(--text-secondary)]">以"添加用户时间"为开始时间，可在用户管理页对单个用户进行再次调整</p>
                      </div>
                      <div className="space-y-0.5">
                        <p className="font-semibold text-[var(--text-emphasis)]">未匹配任何组织策略的组织</p>
                        <p className="text-[var(--text-secondary)]">以"预设策略最新保存时间"为开始时间，当前为{formatDisplay(PRESET_SAVED_AT)}</p>
                      </div>
                    </div>
                  </TooltipContent>
                </Tooltip>
              )}
            </div>
            <div className="w-[200px] shrink-0 px-4 text-xs text-[var(--text-title)]">无终止时间</div>
            <div className="w-[160px] shrink-0 px-4">
              <Select value={editFallback.refresh ?? "daily"} onValueChange={(v) => updateRefresh(editFallback.id, v as CustomRefresh)}>
                <SelectTrigger size="sm" className="!h-7 w-full px-3 text-xs bg-background">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {(["daily", "monthly", "yearly", "none"] as const).map((r) => (
                    <SelectItem key={r} value={r} className="text-xs">{CUSTOM_REFRESH_LABEL[r]}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="w-[80px] shrink-0" />
          </div>
        </div>
        )}
        {/* 组织策略：表格 + 添加按钮 —— 列：组织策略（序号）/ 配额 / 组织 / 开始时间 / 终止时间 / 刷新方式 / 操作 */}
        <div className="rounded-[4px] bg-white border border-[var(--border)] overflow-x-auto">
          {editGroupRules.length > 0 && (
            <Table density="compact" variant="white" autoFixedColumns={false} className="table-fixed min-w-[1100px]">
              <colgroup>
                <col className="w-[100px]" />
                <col className="w-[160px]" />
                <col className="w-[200px]" />
                <col className="w-[200px]" />
                <col className="w-[200px]" />
                <col className="w-[160px]" />
                <col className="w-[80px]" />
              </colgroup>
              <TableHeader>
                <TableRow className="border-0">
                  <TableHead className="text-xs text-[var(--text-weak)]">组织策略</TableHead>
                  <TableHead className="text-xs text-[var(--text-weak)]">配额</TableHead>
                  <TableHead className="text-xs text-[var(--text-weak)]">组织</TableHead>
                  <TableHead className="text-xs text-[var(--text-weak)]">开始时间</TableHead>
                  <TableHead className="text-xs text-[var(--text-weak)]">终止时间</TableHead>
                  <TableHead className="text-xs text-[var(--text-weak)]">刷新方式</TableHead>
                  <TableHead className="text-xs text-[var(--text-weak)]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {editGroupRules.map((rule, idx) => (
                  <TableRow key={rule.id} className={idx < editGroupRules.length - 1 ? "border-b border-[var(--border)]" : "border-0"}>
                    <TableCell className="text-xs text-[var(--text-body)] tabular-nums">组织策略 {idx + 1}</TableCell>
                    <TableCell>{renderValueEditor(rule.id)}</TableCell>
                    <TableCell>
                      <GroupTagSelector
                        selectedIds={rule.groupIds}
                        disabledIds={getDisabledIds(rule.id)}
                        onChange={(ids) => updateGroups(rule.id, ids)}
                      />
                    </TableCell>
                    <TableCell>
                      <DateTimeField value={rule.startAt ?? ""} onChange={(v) => updateStartAt(rule.id, v)} />
                    </TableCell>
                    <TableCell>
                      <DateTimeField value={rule.endAt ?? ""} onChange={(v) => updateEndAt(rule.id, v)} placeholder="无终止时间" clearable clearLabel="设为无终止时间" />
                    </TableCell>
                    <TableCell>
                      <Select value={rule.refresh ?? "daily"} onValueChange={(v) => updateRefresh(rule.id, v as CustomRefresh)}>
                        <SelectTrigger size="sm" className="!h-7 w-full px-3 text-xs bg-background">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {(["daily", "monthly", "yearly", "none"] as const).map((r) => (
                            <SelectItem key={r} value={r} className="text-xs">{CUSTOM_REFRESH_LABEL[r]}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableActionCell>
                      <Button variant="link" size="sm" onClick={() => removeRule(rule.id)}>删除</Button>
                    </TableActionCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <button
            type="button"
            onClick={addBlankGroupRow}
            className={`w-full flex items-center justify-center gap-1 px-3 py-2.5 text-xs text-[var(--text-muted)] ${editGroupRules.length > 0 ? "border-t border-dashed border-[var(--border)]" : ""} hover:text-[var(--text-emphasis)] hover:bg-[var(--bg-grey-hover-subtle)] transition-colors`}
          >
          <Plus className="w-3.5 h-3.5" />添加组织策略
        </button>
        </div>
      </div>
    );
  };

  return (
    <Card className="overflow-hidden h-full py-0 gap-0 [&_[data-slot=table-container]]:!overflow-x-auto">
      <div className="px-5 pt-5 pb-4">
        <div className="flex items-start gap-3">
          <div className="shrink-0">{icon}</div>
          <div className="min-w-0 flex-1">
            <CardTitle as="h3">{title}</CardTitle>
            <MetaText as="p" className="mt-1 leading-relaxed">{description}</MetaText>
          </div>
          {cardEditing ? (
            <div className="flex items-center gap-2 shrink-0">
              <Button variant="claw-outline" size="claw-sm" onClick={cancelCardEdit}>取消</Button>
              <Button variant="dialog-confirm" size="claw-sm" onClick={saveCardEdit}>保存</Button>
            </div>
          ) : (
            <Button variant="claw-outline" size="claw-sm" className="shrink-0" onClick={startCardEdit}>
              编辑
            </Button>
          )}
        </div>
      </div>

      <div className="px-5 pb-4">
        {cardEditing
          ? (isCustom ? renderCustomEditing() : renderNaturalEditing())
          : (isCustom ? renderCustomReadOnly() : renderNaturalReadOnly())
        }
      </div>

      {extraContent && (
        <CardFooter className="px-5 pt-0 pb-3 flex-col items-start gap-3">
          {extraContent}
        </CardFooter>
      )}

      {/* 周期类型切换二次确认 */}
      <AlertDialog open={!!pendingTypeSwitch} onOpenChange={(open) => { if (!open) setPendingTypeSwitch(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>切换周期类型</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription className="space-y-3 !text-[var(--text-body)]">
            <span className="block">切换到「{pendingTypeSwitch === "natural" ? "自然周期" : "自定义周期"}」后：</span>
            <ul className="list-disc pl-4 space-y-1.5">
              {pendingTypeSwitch === "natural" ? (
                <>
                  <li>已配置的自定义周期组织策略将被清空，需要重新配置</li>
                  <li>预设策略将作用于全部用户，按所选自然周期（{NATURAL_PERIOD_LABEL[pendingDraftPeriod]}）刷新</li>
                </>
              ) : (
                <>
                  <li>已配置的自然周期组织策略将被清空，需要重新配置</li>
                  <li>预设策略将作用于全部用户，刷新方式将基于当前自然周期长度自动映射</li>
                </>
              )}
            </ul>
            <span className="block">是否确认切换？</span>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={confirmTypeSwitch}>确认切换</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  );
}

// ─── 通用选项指示器（label + 当前值 badge + 编辑 Popover + 信息 Tooltip） ───────

interface LabeledOption<T extends string> {
  value: T;
  label: string;
}

function LabeledOptionIndicator<T extends string>({
  label,
  value,
  options,
  onSave,
  tooltipContent,
  saveToastFormatter,
}: {
  label: string;
  value: T;
  options: LabeledOption<T>[];
  onSave: (v: T) => void;
  tooltipContent: React.ReactNode;
  /** 保存后的 toast 文案生成器，参数为新选中项的 label */
  saveToastFormatter?: (nextLabel: string) => string;
}) {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<T>(value);

  const handleOpenChange = (v: boolean) => {
    if (v) setDraft(value);
    setOpen(v);
  };
  const handleConfirm = () => {
    onSave(draft);
    setOpen(false);
    const nextLabel = options.find((o) => o.value === draft)?.label ?? "";
    toast.success(saveToastFormatter ? saveToastFormatter(nextLabel) : `已切换为${nextLabel}`);
  };

  const currentLabel = options.find((o) => o.value === value)?.label ?? "";

  return (
    <div className="flex items-center gap-2 text-xs">
      <span className="text-gray-400">{label}</span>
      <StatusTag mode="fill" variant="gray">{currentLabel}</StatusTag>
      <Popover open={open} onOpenChange={handleOpenChange}>
        <PopoverTrigger asChild>
          <Button variant="link-dark" size="sm" className="h-auto px-0" title={`编辑${label}`}>编辑</Button>
        </PopoverTrigger>
        <PopoverContent className="w-56 p-0" align="start" sideOffset={6}>
          <div className="px-3.5 pt-3.5 pb-2.5 space-y-2.5">
            <SegmentGroup className="w-full">
              {options.map((opt) => (
                <SegmentOption
                  key={opt.value}
                  active={draft === opt.value}
                  onClick={() => setDraft(opt.value)}
                  className="flex-1"
                >
                  {opt.label}
                </SegmentOption>
              ))}
            </SegmentGroup>
          </div>
          <div className="flex items-center justify-end gap-2 px-3.5 py-2.5 border-t border-[var(--border)]">
            <Button size="sm" variant="outline" className="h-7 text-xs px-3" onClick={() => setOpen(false)}>取消</Button>
            <Button size="sm" className="h-7 text-xs px-3" onClick={handleConfirm}>确认</Button>
          </div>
        </PopoverContent>
      </Popover>
      <Tooltip>
        <TooltipTrigger asChild><span className="cursor-default"><Info className="w-3 h-3 text-gray-400" /></span></TooltipTrigger>
        <TooltipContent side="top" className="text-xs max-w-[320px] leading-relaxed">
          {tooltipContent}
        </TooltipContent>
      </Tooltip>
    </div>
  );
}

// ─── 访问方式指示器（基于 LabeledOptionIndicator） ────────────────────────────

function AccessModeIndicator({ mode, onSave }: { mode: "public" | "private"; onSave: (m: "public" | "private") => void }) {
  return (
    <LabeledOptionIndicator<"public" | "private">
      label="访问方式"
      value={mode}
      options={[
        { value: "public", label: "公网访问" },
        { value: "private", label: "私网访问" },
      ]}
      onSave={onSave}
      tooltipContent={
        <>
          <p className="mb-1.5 text-justify"><span className="font-medium">公网访问：</span>用户通过公网直接访问 Agent 面板（WebUI），连接云服务器公网 IP。适用于大多数场景，推荐选择。</p>
          <p className="text-justify"><span className="font-medium">私网访问：</span>用户通过同一私有网络访问 Agent 面板（WebUI），连接云服务器内网 IP。使用前需先自行将企业内网与腾讯云私有网络（VPC）打通，并在「网络管理」中将云服务器绑定至该 VPC。配置完成后，企业用户可通过企业内网访问面板，但无法通过公网访问。</p>
        </>
      }
    />
  );
}

// ─── 时间维度指示器（基于 LabeledOptionIndicator） ────────────────────────────

function TimeDimensionIndicator({ mode, onSave }: { mode: "daily" | "monthly"; onSave: (m: "daily" | "monthly") => void }) {
  return (
    <LabeledOptionIndicator<"daily" | "monthly">
      label="时间维度"
      value={mode}
      options={[
        { value: "daily", label: "每日" },
        { value: "monthly", label: "每月" },
      ]}
      onSave={onSave}
      tooltipContent={
        <>
          <p className="mb-1.5"><span className="font-medium">每日：</span>每日全局 Tokens 到达上限即暂停服务，按自然日统计，每天 0 点重置。</p>
          <p><span className="font-medium">每月：</span>每月全局 Tokens 到达上限即暂停服务，按自然月统计，每月 1 号 0 点重置。</p>
        </>
      }
    />
  );
}

// ─── 统一的行容器 ─────────────────────────────────────────────────────────────
const ROW_CLASS = "flex items-center gap-3 h-10";
// 编辑行：允许组织标签撑开高度（多标签时换行）
const EDIT_ROW_CLASS = "flex items-center gap-3 min-h-10 py-1.5";

// ─── 子组件：配额策略卡片 / 策略编辑卡片 已抽离到 @/components/policy ───────
// PolicyEditCard、QuotaPolicyCard、TokenValueEditor 现统一从 @/components/policy 导入


// ─── Hover 气泡组件（白底黑字，hover 触发） ─────────────────────────────────

function HoverPopover({ trigger, children, width = 280 }: { trigger: React.ReactNode; children: React.ReactNode; width?: number }) {
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <span onMouseEnter={() => setOpen(true)} onMouseLeave={() => setOpen(false)}>
          {trigger}
        </span>
      </PopoverTrigger>
      <PopoverContent
        className="p-3 bg-white text-[#020617] shadow-[var(--shadow-popover)]"
        style={{ width }}
        align="start"
        sideOffset={6}
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
      >
        {children}
      </PopoverContent>
    </Popover>
  );
}

// ─── 主页面 ───────────────────────────────────────────────────────────────────

export default function PlatformPolicy() {
  const [, navigate] = useLocation();

  // 平台策略页：让 inset main 显式作为滚动容器，并解除 wrapper 的 overflow 限制
  // 这样 sticky 锚点导航可相对 main 视口顶部吸顶
  useEffect(() => {
    const inset = document.querySelector('[data-slot="admin-sidebar-inset"]') as HTMLElement | null;
    if (!inset) return;
    const wrapper = inset.querySelector(':scope > div') as HTMLElement | null;

    // 保存原值
    const prevInset = {
      height: inset.style.height,
      maxHeight: inset.style.maxHeight,
      overflowY: inset.style.overflowY,
    };
    const prevWrapper = {
      overflow: wrapper?.style.overflow ?? "",
      overflowX: wrapper?.style.overflowX ?? "",
      overflowY: wrapper?.style.overflowY ?? "",
    };

    // 1) inset 显式 100vh + overflow-y:auto，使其成为唯一稳定滚动容器
    inset.style.height = "100vh";
    inset.style.maxHeight = "100vh";
    inset.style.overflowY = "auto";
    // 2) wrapper 解除所有 overflow 限制
    if (wrapper) {
      wrapper.style.overflow = "visible";
      wrapper.style.overflowX = "visible";
      wrapper.style.overflowY = "visible";
    }

    return () => {
      inset.style.height = prevInset.height;
      inset.style.maxHeight = prevInset.maxHeight;
      inset.style.overflowY = prevInset.overflowY;
      if (wrapper) {
        wrapper.style.overflow = prevWrapper.overflow;
        wrapper.style.overflowX = prevWrapper.overflowX;
        wrapper.style.overflowY = prevWrapper.overflowY;
      }
    };
  }, []);

  // ── 用户配额规则 ──
  const [clawRules, setClawRules] = useState<PolicyRule<TokenLimit>[]>([
    { id: "claw-fallback", groupIds: [], value: 3 },
  ]);
  const [tokenRules, setTokenRules] = useState<ExtendedPolicyRule<TokenLimit>[]>([
    { id: "token-fallback", groupIds: [], value: 500000 },
  ]);

  // ── 模型配额规则 ──
  const [globalTokenRules, setGlobalTokenRules] = useState<ExtendedPolicyRule<TokenLimit>[]>([
    { id: "global-fallback", groupIds: [], value: 1000000 },
  ]);
  // 全局 Tokens 时间维度（升级为 TimeDimensionConfig 支持 natural / custom 双模式）
  const [globalTokenTimeDim, setGlobalTokenTimeDim] = useState<TimeDimensionConfig>(() => {
    const raw = localStorage.getItem("admin_global_token_time_dim");
    if (!raw) return { type: "natural", period: "monthly" };
    // 旧版兼容："daily" / "monthly" 字符串 → 包装为 natural
    if (raw === "daily" || raw === "monthly" || raw === "yearly") {
      return { type: "natural", period: raw as NaturalPeriod };
    }
    try {
      const parsed = JSON.parse(raw);
      if (parsed && (parsed.type === "natural" || parsed.type === "custom")) {
        return parsed as TimeDimensionConfig;
      }
    } catch { /* ignore */ }
    return { type: "natural", period: "monthly" };
  });
  // 单用户 Tokens 时间维度（cur 上独有：跨页 localStorage 同步）
  const [userTokenTimeDim, setUserTokenTimeDim] = useState<TimeDimensionConfig>(() => {
    const raw = localStorage.getItem("admin_user_token_time_dim");
    if (!raw) return { type: "natural", period: "daily" };
    if (raw === "daily" || raw === "monthly" || raw === "yearly") {
      return { type: "natural", period: raw as NaturalPeriod };
    }
    try {
      const parsed = JSON.parse(raw);
      if (parsed && (parsed.type === "natural" || parsed.type === "custom")) {
        return parsed as TimeDimensionConfig;
      }
    } catch { /* ignore */ }
    return { type: "natural", period: "daily" };
  });
  // 持久化：写 localStorage + 派发 storage 事件，便于 TokensMonitor 等页面跨页同步
  const persistGlobalTokenTimeDim = (next: TimeDimensionConfig) => {
    setGlobalTokenTimeDim(next);
    const serialized = JSON.stringify(next);
    localStorage.setItem("admin_global_token_time_dim", serialized);
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_global_token_time_dim", newValue: serialized, storageArea: localStorage }));
  };
  const persistUserTokenTimeDim = (next: TimeDimensionConfig) => {
    setUserTokenTimeDim(next);
    const serialized = JSON.stringify(next);
    localStorage.setItem("admin_user_token_time_dim", serialized);
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_user_token_time_dim", newValue: serialized, storageArea: localStorage }));
  };
  // 监听 storage 事件：其他页面（如用户管理）修改时间维度时同步本页
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === "admin_global_token_time_dim" && e.newValue) {
        try {
          const parsed = JSON.parse(e.newValue);
          if (parsed && (parsed.type === "natural" || parsed.type === "custom")) {
            setGlobalTokenTimeDim(parsed);
          }
        } catch { /* ignore */ }
      }
      if (e.key === "admin_user_token_time_dim" && e.newValue) {
        try {
          const parsed = JSON.parse(e.newValue);
          if (parsed && (parsed.type === "natural" || parsed.type === "custom")) {
            setUserTokenTimeDim(parsed);
          }
        } catch { /* ignore */ }
      }
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);
  // 兼容旧 daily/monthly 文案：提取当前自然周期 label（非 natural 时回退为 monthly）
  const globalTokenPeriodLabel: string = globalTokenTimeDim.type === "natural"
    ? NATURAL_PERIOD_LABEL[globalTokenTimeDim.period]
    : "自定义";
  // 初次挂载时把当前组织策略同步到 localStorage，确保 TokensMonitor 能读到
  useEffect(() => {
    const groupRules = globalTokenRules
      .filter((r) => r.groupIds.length > 0)
      .map((r) => ({ id: r.id, groupIds: r.groupIds, value: r.value }));
    const serialized = JSON.stringify(groupRules);
    localStorage.setItem("admin_global_token_group_rules", serialized);
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_global_token_group_rules", newValue: serialized, storageArea: localStorage }));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  const handleGlobalTokenRulesChange = (next: ExtendedPolicyRule<TokenLimit>[]) => {
    setGlobalTokenRules(next);
    const fallback = next.find((r) => r.groupIds.length === 0);
    if (fallback) {
      if (fallback.value === "unlimited") {
        localStorage.setItem("globalLimitMode", "unlimited");
        window.dispatchEvent(new StorageEvent("storage", { key: "globalLimitMode", newValue: "unlimited", storageArea: localStorage }));
      } else {
        localStorage.setItem("globalLimitMode", "custom");
        localStorage.setItem("globalLimit", String(fallback.value));
        window.dispatchEvent(new StorageEvent("storage", { key: "globalLimitMode", newValue: "custom", storageArea: localStorage }));
      }
    }
    // 同步组织策略（除兜底行之外）到 localStorage，供 TokensMonitor 读取
    const groupRules = next
      .filter((r) => r.groupIds.length > 0)
      .map((r) => ({ id: r.id, groupIds: r.groupIds, value: r.value }));
    const serialized = JSON.stringify(groupRules);
    localStorage.setItem("admin_global_token_group_rules", serialized);
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_global_token_group_rules", newValue: serialized, storageArea: localStorage }));
  };

  // ── 功能权限开关规则 ──
  const [configModelRules, setConfigModelRules] = useState<PolicyRule<boolean>[]>([{ id: "cm-fallback", groupIds: [], value: true }]);
  const [configChannelRules, setConfigChannelRules] = useState<PolicyRule<boolean>[]>([{ id: "cc-fallback", groupIds: [], value: true }]);
  const [customModelRules, setCustomModelRules] = useState<PolicyRule<boolean>[]>([{ id: "cust-fallback", groupIds: [], value: false }]);
  const [terminalRules, setTerminalRules] = useState<PolicyRule<boolean>[]>([{ id: "term-fallback", groupIds: [], value: false }]);
  // 允许员工自助更新 Agent 版本（用户端 OpenClawDetail 读取同一 key）
  // 默认值 true（新企业符合既有产品行为：员工有"一键更新"按钮）
  const [selfUpgradeRules, setSelfUpgradeRules] = useState<PolicyRule<boolean>[]>(() => {
    const raw = localStorage.getItem("admin_allow_self_upgrade");
    const value = raw === null ? true : raw === "true";
    return [{ id: "selfup-fallback", groupIds: [], value }];
  });
  const handleSelfUpgradeRulesChange = (next: PolicyRule<boolean>[]): boolean | void => {
    setSelfUpgradeRules(next);
    const enabled = next.some((r) => r.value);
    localStorage.setItem("admin_allow_self_upgrade", enabled ? "true" : "false");
  };
  // 一次性清掉 panel / cloudBrowser 两个开关的 localStorage 残留，确保默认显示为"不允许"
  // 仅在首次执行时清理一次（用版本号 key 标记），后续管理员自行开启的持久化值不再被清掉
  if (typeof window !== "undefined" && localStorage.getItem("admin_policy_default_reset_v1") !== "done") {
    localStorage.removeItem("admin_allow_panel_access");
    localStorage.removeItem("admin_allow_cloud_browser");
    // 同步清理与之关联的派生状态，避免残留出现"已分配端口/规则"等不一致的 UI 提示
    localStorage.removeItem("admin_panel_port");
    localStorage.removeItem("admin_panel_sg_rule_id");
    localStorage.removeItem("admin_cloud_browser_sg_rule_id");
    localStorage.setItem("admin_policy_default_reset_v1", "done");
  }

  const [panelRules, setPanelRules] = useState<PolicyRule<boolean>[]>(() => [
    { id: "panel-fallback", groupIds: [], value: localStorage.getItem("admin_allow_panel_access") === "true" },
  ]);
  const [chatViewRules, setChatViewRules] = useState<PolicyRule<boolean>[]>([{ id: "chat-fallback", groupIds: [], value: true }]);
  const [cloudBrowserRules, setCloudBrowserRules] = useState<PolicyRule<boolean>[]>(() => [
    { id: "cb-fallback", groupIds: [], value: localStorage.getItem("admin_allow_cloud_browser") === "true" },
  ]);
  const [lobsterDoctorRules, setLobsterDoctorRules] = useState<PolicyRule<boolean>[]>(() => [
    // 仅显式"允许"才开启；与用户端采用同一判定，避免两个页面首次进入时状态不一致。
    { id: "ld-fallback", groupIds: [], value: localStorage.getItem("admin_allow_lobster_doctor") === "true" },
  ]);
  const [modelQuotaRules, setModelQuotaRules] = useState<PolicyRule<boolean>[]>([{ id: "mq-fallback", groupIds: [], value: true }]);

  // 允许用户「接入外部 Agent」
  const [localClientAccessRules, setLocalClientAccessRules] = useState<PolicyRule<boolean>[]>(() => {
    const rawFallback = localStorage.getItem("admin_allow_local_client_access");
    const fallbackEnabled = rawFallback === null ? true : rawFallback === "true";
    const rules: PolicyRule<boolean>[] = [{ id: "localclient-fallback", groupIds: [], value: fallbackEnabled }];
    try {
      const raw = localStorage.getItem("admin_local_client_access_group_rules");
      if (raw) {
        const parsed = JSON.parse(raw) as PolicyRule<boolean>[];
        if (Array.isArray(parsed)) {
          parsed.forEach((r) => {
            if (r && Array.isArray(r.groupIds) && r.groupIds.length > 0) {
              rules.push({ id: r.id, groupIds: r.groupIds, value: !!r.value });
            }
          });
        }
      }
    } catch { /* 忽略损坏的本地数据 */ }
    return rules;
  });
  const handleLocalClientAccessRulesChange = (next: PolicyRule<boolean>[]): boolean | void => {
    setLocalClientAccessRules(next);
    const fallback = next.find((r) => r.groupIds.length === 0);
    const fallbackEnabled = fallback ? fallback.value : true;
    localStorage.setItem("admin_allow_local_client_access", fallbackEnabled ? "true" : "false");
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_allow_local_client_access", newValue: fallbackEnabled ? "true" : "false", storageArea: localStorage }));
    const groupRules = next.filter((r) => r.groupIds.length > 0).map((r) => ({ id: r.id, groupIds: r.groupIds, value: r.value }));
    const serialized = JSON.stringify(groupRules);
    localStorage.setItem("admin_local_client_access_group_rules", serialized);
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_local_client_access_group_rules", newValue: serialized, storageArea: localStorage }));
  };

  // 允许用户「共享 agent」
  const [shareAgentRules, setShareAgentRules] = useState<PolicyRule<boolean>[]>(() => {
    // 默认「允许」：localStorage 无存值（首次进入）时兜底为 true，管理员显式关闭后才写入 "false"。
    const stored = localStorage.getItem("admin_allow_share_agent");
    const fallbackEnabled = stored === null ? true : stored === "true";
    const rules: PolicyRule<boolean>[] = [{ id: "shareagent-fallback", groupIds: [], value: fallbackEnabled }];
    try {
      const raw = localStorage.getItem("admin_share_agent_group_rules");
      if (raw) {
        const parsed = JSON.parse(raw) as PolicyRule<boolean>[];
        if (Array.isArray(parsed)) {
          parsed.forEach((r) => {
            if (r && Array.isArray(r.groupIds) && r.groupIds.length > 0) {
              rules.push({ id: r.id, groupIds: r.groupIds, value: !!r.value });
            }
          });
        }
      }
    } catch { /* 忽略损坏的本地数据 */ }
    return rules;
  });
  const handleShareAgentRulesChange = (next: PolicyRule<boolean>[]): boolean | void => {
    setShareAgentRules(next);
    const fallback = next.find((r) => r.groupIds.length === 0);
    const fallbackEnabled = fallback ? fallback.value : false;
    localStorage.setItem("admin_allow_share_agent", fallbackEnabled ? "true" : "false");
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_allow_share_agent", newValue: fallbackEnabled ? "true" : "false", storageArea: localStorage }));
    const groupRules = next.filter((r) => r.groupIds.length > 0).map((r) => ({ id: r.id, groupIds: r.groupIds, value: r.value }));
    const serialized = JSON.stringify(groupRules);
    localStorage.setItem("admin_share_agent_group_rules", serialized);
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_share_agent_group_rules", newValue: serialized, storageArea: localStorage }));
  };

  // 允许用户「编辑项目」（用户端 ProjectCollaboration 读取同一 key）
  const [projectEditRules, setProjectEditRules] = useState<PolicyRule<boolean>[]>(() => {
    const stored = localStorage.getItem("admin_allow_project_edit");
    const fallbackEnabled = stored === null ? true : stored === "true";
    const rules: PolicyRule<boolean>[] = [{ id: "projectedit-fallback", groupIds: [], value: fallbackEnabled }];
    try {
      const raw = localStorage.getItem("admin_project_edit_group_rules");
      if (raw) {
        const parsed = JSON.parse(raw) as PolicyRule<boolean>[];
        if (Array.isArray(parsed)) {
          parsed.forEach((r) => {
            if (r && Array.isArray(r.groupIds) && r.groupIds.length > 0) {
              rules.push({ id: r.id, groupIds: r.groupIds, value: !!r.value });
            }
          });
        }
      }
    } catch { /* 忽略损坏的本地数据 */ }
    return rules;
  });
  const handleProjectEditRulesChange = (next: PolicyRule<boolean>[]): boolean | void => {
    setProjectEditRules(next);
    const fallback = next.find((r) => r.groupIds.length === 0);
    const fallbackEnabled = fallback ? fallback.value : true;
    localStorage.setItem("admin_allow_project_edit", fallbackEnabled ? "true" : "false");
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_allow_project_edit", newValue: fallbackEnabled ? "true" : "false", storageArea: localStorage }));
    const groupRules = next.filter((r) => r.groupIds.length > 0).map((r) => ({ id: r.id, groupIds: r.groupIds, value: r.value }));
    const serialized = JSON.stringify(groupRules);
    localStorage.setItem("admin_project_edit_group_rules", serialized);
    window.dispatchEvent(new StorageEvent("storage", { key: "admin_project_edit_group_rules", newValue: serialized, storageArea: localStorage }));
  };

  // ── Agent 面板属性 ──
  const [panelAccessMode, setPanelAccessMode] = useState<"public" | "private">(() =>
    (localStorage.getItem("admin_panel_access_mode") as "public" | "private") || "public"
  );
  const [panelPort, setPanelPort] = useState<string | null>(() => localStorage.getItem("admin_panel_port"));
  const [panelLoadingRuleId, setPanelLoadingRuleId] = useState<string | null>(null);
  // 本次自动追加的 面板端口 放通规则 id（用于在卡片内展示"已自动添加"提示）
  const [panelSgRuleId, setPanelSgRuleId] = useState<string | null>(() => localStorage.getItem("admin_panel_sg_rule_id"));

  // 计算面板规则是否已开启（任一规则值为 true）
  const isPanelEnabled = (rs: PolicyRule<boolean>[]) => rs.some((r) => r.value);

  // 检查是否已配置安全组
  const hasSecurityGroup = useMemo(() => {
    const snapshotRaw = localStorage.getItem("admin_default_security_group_snapshot");
    if (!snapshotRaw) return false;
    try {
      const snapshot = JSON.parse(snapshotRaw);
      return snapshot && Array.isArray(snapshot.inboundRules);
    } catch { return false; }
  }, []);

  // 面板 / 云桌面：点击编辑后安全组校验失败标记（默认可用，点编辑后才校验）
  const [panelSgCheckFailed, setPanelSgCheckFailed] = useState(false);
  const [cloudBrowserSgCheckFailed, setCloudBrowserSgCheckFailed] = useState(false);

  // 找到触发开启的那一行（next 中 value=true 但 prev 中 false 的第一行；找不到则返回兜底行）
  const findTriggeredEnableRule = (prev: PolicyRule<boolean>[], next: PolicyRule<boolean>[]) => {
    const prevMap = new Map(prev.map((r) => [r.id, r.value]));
    const enabledRow = next.find((r) => r.value && !prevMap.get(r.id));
    return enabledRow?.id ?? next.find((r) => r.value)?.id ?? null;
  };

  const handlePanelRulesChange = (next: PolicyRule<boolean>[]): boolean | void => {
    const prev = panelRules;
    const wasEnabled = isPanelEnabled(prev);
    const willEnable = isPanelEnabled(next);

    // 关闭 → 开启：执行开启流程（校验安全组 + loading + 分配端口 + 补规则）
    if (!wasEnabled && willEnable) {
      const snapshotRaw = localStorage.getItem("admin_default_security_group_snapshot");
      let snapshot: DefaultSecurityGroupSnapshot | null = null;
      if (snapshotRaw) {
        try { snapshot = JSON.parse(snapshotRaw) as DefaultSecurityGroupSnapshot; } catch { snapshot = null; }
      }
      if (!snapshot || !Array.isArray(snapshot.inboundRules)) {
        toast.error("请先前往网络管理配置 ClawPro 的安全组，再开启该功能");
        return false; // 回滚：不应用本次规则变更，卡片也不弹成功 toast
      }
      // 应用规则变更并进入 loading
      setPanelRules(next);
      localStorage.setItem("admin_allow_panel_access", "true");
      const triggeredId = findTriggeredEnableRule(prev, next);
      setPanelLoadingRuleId(triggeredId);
      setTimeout(() => {
        const randomPort = String(Math.floor(Math.random() * 1000) + 9000);
        const portNum = Number(randomPort);
        const hasCovered = snapshot!.inboundRules.some((r) => isInboundRuleCoverPort(r, portNum));
        if (!hasCovered) {
          const newRule: SnapshotInboundRule = {
            id: `panel-${Date.now()}`,
            source: "0.0.0.0/0",
            protocol: "TCP",
            port: randomPort,
            policy: "允许",
            remark: "Agent 面板访问",
          };
          const nextSnapshot: DefaultSecurityGroupSnapshot = {
            ...snapshot!,
            inboundRules: [...snapshot!.inboundRules, newRule],
          };
          localStorage.setItem("admin_default_security_group_snapshot", JSON.stringify(nextSnapshot));
          localStorage.setItem("admin_panel_sg_rule_id", newRule.id);
          setPanelSgRuleId(newRule.id);
        } else {
          localStorage.removeItem("admin_panel_sg_rule_id");
          setPanelSgRuleId(null);
        }
        setPanelPort(randomPort);
        localStorage.setItem("admin_panel_port", randomPort);
        setPanelLoadingRuleId(null);
        toast.success("已开启用户端访问 Agent 面板");
      }, 3000);
      return;
    }

    // 开启 → 关闭：清理端口和自动补规则标记
    if (wasEnabled && !willEnable) {
      setPanelRules(next);
      localStorage.setItem("admin_allow_panel_access", "false");
      setPanelPort(null);
      localStorage.removeItem("admin_panel_port");
      localStorage.removeItem("admin_panel_sg_rule_id");
      setPanelSgRuleId(null);
      toast.success("已禁止用户端访问 Agent 面板");
      return;
    }

    // 其他情况（已开启状态下规则增删/值变更，或都是关闭态）：直接应用
    setPanelRules(next);
  };

  // ── Agent 云桌面属性 ──
  const [cloudBrowserSgRuleId, setCloudBrowserSgRuleId] = useState<string | null>(() =>
    localStorage.getItem("admin_cloud_browser_sg_rule_id"),
  );
  const isCloudBrowserEnabled = (rs: PolicyRule<boolean>[]) => rs.some((r) => r.value);

  const handleCloudBrowserRulesChange = (next: PolicyRule<boolean>[]): boolean | void => {
    const wasEnabled = isCloudBrowserEnabled(cloudBrowserRules);
    const willEnable = isCloudBrowserEnabled(next);

    // 关闭 → 开启：校验安全组并尝试补 6080 放通规则
    if (!wasEnabled && willEnable) {
      const snapshotRaw = localStorage.getItem("admin_default_security_group_snapshot");
      let snapshot: DefaultSecurityGroupSnapshot | null = null;
      if (snapshotRaw) {
        try { snapshot = JSON.parse(snapshotRaw) as DefaultSecurityGroupSnapshot; } catch { snapshot = null; }
      }
      if (!snapshot || !Array.isArray(snapshot.inboundRules)) {
        toast.error("请先前往网络管理配置 ClawPro 的安全组，再开启该功能");
        return false;
      }

      // 判断是否已有 6080 放通规则
      const hasCovered = snapshot.inboundRules.some((r) => isInboundRuleCoverPort(r, 6080));
      if (!hasCovered) {
        const newRule: SnapshotInboundRule = {
          id: `cb-${Date.now()}`,
          source: "0.0.0.0/0",
          protocol: "TCP",
          port: "6080",
          policy: "允许",
          remark: "云桌面访问",
        };
        const nextSnapshot: DefaultSecurityGroupSnapshot = {
          ...snapshot,
          inboundRules: [...snapshot.inboundRules, newRule],
        };
        localStorage.setItem("admin_default_security_group_snapshot", JSON.stringify(nextSnapshot));
        localStorage.setItem("admin_cloud_browser_sg_rule_id", newRule.id);
        setCloudBrowserSgRuleId(newRule.id);
      } else {
        localStorage.removeItem("admin_cloud_browser_sg_rule_id");
        setCloudBrowserSgRuleId(null);
      }

      setCloudBrowserRules(next);
      localStorage.setItem("admin_allow_cloud_browser", "true");
      toast.success("已开启 Agent 云桌面");
      return;
    }

    // 开启 → 关闭：清掉"自动添加"标记（规则保留在安全组中）
    if (wasEnabled && !willEnable) {
      setCloudBrowserRules(next);
      localStorage.setItem("admin_allow_cloud_browser", "false");
      localStorage.removeItem("admin_cloud_browser_sg_rule_id");
      setCloudBrowserSgRuleId(null);
      toast.success("已关闭 Agent 云桌面");
      return;
    }

    // 其他情况：直接应用
    setCloudBrowserRules(next);
  };

  // ── 龙虾医生开关持久化 ──────────────────────────────────────────────
  // 该策略在同域同浏览器内由 localStorage 共享；自定义事件补齐同一标签页内
  // localStorage 写入不触发 storage 事件的浏览器限制。
  const isLobsterDoctorEnabled = (rs: PolicyRule<boolean>[]) => rs.some((r) => r.value);

  useEffect(() => {
    const enabled = isLobsterDoctorEnabled(lobsterDoctorRules);
    const expected = enabled ? "true" : "false";
    if (localStorage.getItem("admin_allow_lobster_doctor") !== expected) {
      localStorage.setItem("admin_allow_lobster_doctor", expected);
      window.dispatchEvent(
        new CustomEvent("lobster-doctor-policy-changed", { detail: { enabled } })
      );
    }
  }, [lobsterDoctorRules]);

  const handleLobsterDoctorRulesChange = (next: PolicyRule<boolean>[]): boolean | void => {
    const wasEnabled = isLobsterDoctorEnabled(lobsterDoctorRules);
    const willEnable = isLobsterDoctorEnabled(next);
    setLobsterDoctorRules(next);
    if (!wasEnabled && willEnable) {
      toast.success("已开启龙虾医生");
    } else if (wasEnabled && !willEnable) {
      toast.success("已关闭龙虾医生");
    }
  };

  // ── 锚点导航聚焦 ──
  const [activeAnchor, setActiveAnchor] = useState<string>("claw");
  const [highlightKey, setHighlightKey] = useState<string | null>(null);
  const highlightTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const triggerHighlight = (key: string) => {
    if (highlightTimerRef.current) clearTimeout(highlightTimerRef.current);
    setHighlightKey(key);
    highlightTimerRef.current = setTimeout(() => setHighlightKey(null), 1500);
  };
  // ── 龙虾医生详情弹窗 ──
  const [showLobsterDoctorDialog, setShowLobsterDoctorDialog] = useState(false);
  //龙虾医生诊断模型配置（"__default__" 表示跟随默认配置模型）
  const [adminModels] = useAdminModelsState();
  const [lobsterPrimaryModel, setLobsterPrimaryModel] = useState<string>("__default__");
  const [lobsterBackupModel, setLobsterBackupModel] = useState<string>("__none__");

  // 已启用模型列表（供诊断模型 Select 使用）
  const enabledModels = useMemo(() => adminModels.filter((m) => m.visible), [adminModels]);
  const defaultModel = useMemo(() => adminModels.find((m) => m.isDefault), [adminModels]);

  // ── 方案2/方案3 共用的卡片数据定义 ──
  const quotaCards = [
    { key: "claw", icon: "/assets/admin-platform-policy/user-agent-limit.svg", iconClass: "w-[42px]", title: "单用户 Agent 数量上限", description: "单用户最多可以创建的 Agent 数量，新用户创建时自动应用此默认值，可在用户管理中对单个用户单独调整", type: "integer" as const, rules: clawRules, onRulesChange: setClawRules },
    { key: "token", icon: "/assets/admin-platform-policy/user-daily-token-limit.svg", iconClass: "w-10", title: "单用户每日 Tokens 上限", description: "单用户每日最多可消耗的 Tokens 数量，新用户创建时自动应用此默认值，可在用户管理中对单个用户单独调整", type: "token" as const, rules: tokenRules, onRulesChange: setTokenRules },
    { key: "global", icon: "/assets/admin-platform-policy/global-token-limit.svg", iconClass: "w-10", title: "全局 Tokens 上限", description: "全局 Tokens 指所有企业用户使用所有模型所消耗的总 Tokens 数量，达到上限后将暂停服务", type: "token" as const, rules: globalTokenRules, onRulesChange: handleGlobalTokenRulesChange },
  ];

  const toggleCards = [
    { key: "configModel", icon: "/assets/admin-platform-policy/allow-config-model.svg", title: "允许用户「配置模型」", navTitle: "配置模型", description: "开启后，用户可在 Agent 详细配置中自行选择和切换模型。关闭后，模型配置区域将锁定，用户无法调整", rules: configModelRules, onRulesChange: setConfigModelRules },
    { key: "configChannel", icon: "/assets/admin-platform-policy/allow-config-channel.svg", title: "允许用户「配置通道」", navTitle: "配置通道", description: "开启后，用户可在 Agent 详细配置中自行添加和管理通道。关闭后，通道配置区域将锁定，用户无法调整", rules: configChannelRules, onRulesChange: setConfigChannelRules },
    { key: "customModel", icon: "/assets/admin-platform-policy/allow-custom-model.svg", title: "允许用户「添加自定义模型」", navTitle: "添加自定义模型", description: "开启后，用户可在 Agent 中自行添加自定义模型，不在企业管控和 Tokens 覆盖范围内", rules: customModelRules, onRulesChange: setCustomModelRules },
    { key: "terminal", icon: "/assets/admin-platform-policy/allow-agent-terminal.svg", title: "允许用户「进入 Agent 终端」", navTitle: "进入 Agent 终端", description: "开启后，所有用户在用户端可看到「进入终端」选项，进入对应 Agent 云服务器的终端", rules: terminalRules, onRulesChange: setTerminalRules },
    { key: "selfUpgrade", icon: "/assets/admin-platform-policy/allow-agent-self-upgrade.svg", title: "允许用户「自助更新版本」", navTitle: "自助更新版本", description: "开启后，员工可在 Agent 详细配置中点击「一键更新」自助更新到管理员设置的版本", rules: selfUpgradeRules, onRulesChange: handleSelfUpgradeRulesChange },
    { key: "panel", icon: "/assets/admin-platform-policy/allow-agent-panel.svg", title: "允许用户「访问 Agent 面板」", navTitle: "访问 Agent 面板", description: "开启后，系统会为企业分配一个随机端口并自动添加一条安全组规则放通该端口", rules: panelRules, onRulesChange: handlePanelRulesChange },
    { key: "chatView", icon: "/assets/admin-platform-policy/allow-chat-view.svg", title: "允许用户「使用对话视图」", navTitle: "使用对话视图", description: "开启后，用户可在「我的 Agent」中使用对话视图，通过浏览器与 AI 对话", rules: chatViewRules, onRulesChange: setChatViewRules },
    { key: "cloudBrowser", icon: "/assets/admin-platform-policy/allow-cloud-browser.svg", title: "允许用户「访问云桌面」", navTitle: "访问云桌面", description: "开启后，用户可在对话视图里访问云桌面，查看 AI 浏览器执行过程并进入操作", rules: cloudBrowserRules, onRulesChange: handleCloudBrowserRulesChange },
    { key: "lobsterDoctor", icon: "/assets/admin-platform-policy/allow-lobster-doctor.svg", title: "允许用户「使用龙虾医生」", navTitle: "使用龙虾医生", description: "开启后，所有用户在用户端可免费使用「龙虾医生」AI 诊断功能", rules: lobsterDoctorRules, onRulesChange: handleLobsterDoctorRulesChange },
    { key: "modelQuota", icon: "/assets/admin-platform-policy/allow-model-quota.svg", title: "允许用户「查看模型额度」", navTitle: "查看模型额度", description: "开启后，用户可在顶部导航栏看到「模型额度」入口，查看个人的 Token 使用情况", rules: modelQuotaRules, onRulesChange: setModelQuotaRules },
    { key: "localClientAccess", icon: "/assets/admin-platform-policy/allow-local-client.svg", title: "允许用户「接入外部 Agent」", navTitle: "接入外部 Agent", description: "开启后，用户可接入外部 Agent，并通过 Reporter 上报至平台统一管理", rules: localClientAccessRules, onRulesChange: handleLocalClientAccessRulesChange },
    { key: "shareAgent", icon: "/assets/admin-platform-policy/allow-share-agent.svg", title: "允许用户「共享 agent」", navTitle: "共享 agent", description: "开启后，用户可在用户端创建 agent 时选择共享自己的 agent，也可对存量云端 agent 修改共享范围", rules: shareAgentRules, onRulesChange: handleShareAgentRulesChange },
    { key: "projectEdit", icon: "/assets/admin-platform-policy/allow-share-agent.svg", title: "允许用户「编辑项目」", navTitle: "编辑项目", description: "开启后，项目管理员可在用户端修改项目名称、描述、目标和成员资产编辑权限。关闭后，用户端项目保持只读", rules: projectEditRules, onRulesChange: handleProjectEditRulesChange },
  ];

  // ── 方案4：解析当前页的滚动容器（优先 admin-sidebar-inset，回退到沿 DOM 向上找可滚动祖先） ──
  const getScrollContainer = (fromEl?: HTMLElement | null): HTMLElement | null => {
    const inset = document.querySelector('[data-slot="admin-sidebar-inset"]') as HTMLElement | null;
    if (inset) {
      const cs = window.getComputedStyle(inset);
      const oy = cs.overflowY;
      if ((oy === "auto" || oy === "scroll" || oy === "overlay") && inset.scrollHeight > inset.clientHeight + 1) {
        return inset;
      }
    }
    // 沿 DOM 向上找最近的可滚动祖先
    let el: HTMLElement | null = fromEl ?? document.getElementById("plan4-scroll-container");
    while (el && el !== document.body) {
      const cs = window.getComputedStyle(el);
      const oy = cs.overflowY;
      if ((oy === "auto" || oy === "scroll" || oy === "overlay") && el.scrollHeight > el.clientHeight + 1) {
        return el;
      }
      el = el.parentElement;
    }
    return inset; // 最终 fallback
  };

  // 平滑滚动到锚点 — 自动适配滚动容器或 window
  const scrollToAnchor = (targetId: string) => {
    const el = document.getElementById(targetId);
    if (!el) return;
    const container = getScrollContainer(el);
    const offset = Math.round(window.innerHeight * 0.1);
    programmaticScrollUntilRef.current = Date.now() + 800;
    if (container) {
      const targetTop = container.scrollTop + el.getBoundingClientRect().top - container.getBoundingClientRect().top - offset;
      container.scrollTo({ top: targetTop, behavior: "smooth" });
    } else {
      const targetTop = window.scrollY + el.getBoundingClientRect().top - offset;
      window.scrollTo({ top: targetTop, behavior: "smooth" });
    }
  };

  // ── 滚动监听 — 内容区滚动时自动同步右侧锚点导航高亮 ──
  // 程序化滚动锁：点击锚点触发 smooth scroll 期间，暂停 observer 自动更新（避免闪烁）
  const programmaticScrollUntilRef = useRef<number>(0);
  useEffect(() => {
    const container = getScrollContainer();
    if (!container) return;

    const anchorKeys: string[] = [...quotaCards.map(c => c.key), ...toggleCards.map(c => c.key)];
    const observed: HTMLElement[] = [];
    anchorKeys.forEach(key => {
      const el = document.getElementById(`plan4-${key}`);
      if (el) observed.push(el);
    });
    if (observed.length === 0) return;

    // 激活线：距 root 顶部 10vh，底部 60vh — 形成顶部细带，命中其中的卡片即为当前锚点
    const topOffset = Math.round(window.innerHeight * 0.1);
    const bottomOffset = Math.round(window.innerHeight * 0.6);
    const observer = new IntersectionObserver(
      (entries) => {
        if (Date.now() < programmaticScrollUntilRef.current) return;
        const visible = entries
          .filter(e => e.isIntersecting)
          .map(e => ({ key: (e.target as HTMLElement).id.replace(/^plan4-/, ""), top: e.boundingClientRect.top }))
          .sort((a, b) => a.top - b.top);
        if (visible.length > 0) {
          setActiveAnchor(visible[0].key);
        }
      },
      {
        root: container,
        rootMargin: `-${topOffset}px 0px -${bottomOffset}px 0px`,
        threshold: 0,
      }
    );
    observed.forEach(el => observer.observe(el));

    // 滚动到底部 / 接近底部时强制锚定最后一项
    // 因为 observer 激活带在视口顶部 10vh 处，靠近底部时最后一张卡片可能永远进不到激活带
    const lastKey = toggleCards.length > 0 ? toggleCards[toggleCards.length - 1].key : (quotaCards.length > 0 ? quotaCards[quotaCards.length - 1].key : null);
    const onScroll = () => {
      if (Date.now() < programmaticScrollUntilRef.current) return;
      if (!lastKey) return;

      // 判定"已抵达底部区域"：满足以下任一条件
      // 1) 容器自身已滚到底（容忍 8px）
      const containerReachedBottom = container.scrollTop + container.clientHeight >= container.scrollHeight - 8;
      // 2) 最后一张卡片的底部已进入视口下半部分（兜底，应对容器没有真正滚到底的情况）
      const lastEl = document.getElementById(`plan4-${lastKey}`);
      let lastVisibleEnough = false;
      if (lastEl) {
        const rect = lastEl.getBoundingClientRect();
        // 卡片底部在视口内（rect.bottom <= viewport height）即视为最后一项已完整可见
        lastVisibleEnough = rect.bottom <= window.innerHeight + 8;
      }
      if (containerReachedBottom || lastVisibleEnough) {
        setActiveAnchor(lastKey);
      }
    };
    container.addEventListener("scroll", onScroll, { passive: true });
    // 初始触发一次（页面打开时若已在底部）
    onScroll();

    return () => {
      observer.disconnect();
      container.removeEventListener("scroll", onScroll);
    };
  }, [quotaCards.length, toggleCards.length]);


  return (
    <div className="page-enter space-y-6">
      {/* 页面标题 */}
      <div>
        <AdminPageHeader title="平台策略" description="管理平台默认配额、全局限制和功能权限开关，支持按组织设置不同策略" />
      </div>

      {/* 优先级说明信息条 */}
      <Alert variant="operation-info">
        <AlertOperationInfoIcon />
        <AlertDescription>
          <ul className="space-y-1 list-disc pl-4">
            <li>无需按组织设置策略时，直接使用<span className="font-medium">「预设策略」</span>，全部用户应用该策略。</li>
            <li>需要按组织设置策略时，添加<span className="font-medium">「组织策略」</span>，优先采用本组织策略；本组织无则采用最近的上级组织策略；均无则使用<span className="font-medium">「预设策略」</span>。</li>
            <li>若用户属于多个组织，用户将在用户端创建 Agent 时自行选择组织，该 Agent 即拥有所选组织对应的策略权限。</li>
          </ul>
        </AlertDescription>
      </Alert>

      {/* 锚点导航 + 瀑布流单列卡片 */}
      <div className="flex gap-[36px]">
          {/* 左侧瀑布流卡片（页面整体滚动，左侧不再独立滚动） */}
          <div id="plan4-scroll-container" className="flex-1 min-w-0 space-y-8">
            <section id="plan4-section-quota">
              <SectionTitle className="mb-4">配额设置</SectionTitle>
              <div className="space-y-4">
                <div id="plan4-claw" className={`rounded-[4px] transition-shadow ${highlightKey === "claw" ? "anchor-highlight" : ""}`}>
                  <QuotaPolicyCard
                    icon={<img src="/assets/admin-platform-policy/user-agent-limit.svg" className="shrink-0 w-[42px]" />}
                    iconBg=""
                    title="单用户 Agent 数量上限"
                    description="单用户最多可以创建的 Agent 数量，新用户创建时自动应用此默认值，可在用户管理中对单个用户单独调整"
                    type="integer"
                    rules={clawRules}
                    onRulesChange={setClawRules}
                  />
                </div>
                <div id="plan4-token" className={`rounded-[4px] transition-shadow ${highlightKey === "token" ? "anchor-highlight" : ""}`}>
                  <TokensQuotaCardWithPeriod
                    icon={<img src="/assets/admin-platform-policy/user-daily-token-limit.svg" className="shrink-0 w-10" />}
                    title="单用户每日 Tokens 上限"
                    description="单用户每日最多可消耗的 Tokens 数量，新用户创建时自动应用此默认值，可在用户管理中对单个用户单独调整"
                    rules={tokenRules}
                    onRulesChange={setTokenRules}
                    timeDimension={userTokenTimeDim}
                    onTimeDimensionChange={persistUserTokenTimeDim}
                    scope="user"
                  />
                </div>
                <div id="plan4-global" className={`rounded-[4px] transition-shadow ${highlightKey === "global" ? "anchor-highlight" : ""}`}>
                  <TokensQuotaCardWithPeriod
                    icon={<img src="/assets/admin-platform-policy/global-token-limit.svg" className="shrink-0 w-10" />}
                    title="全局 Tokens 上限"
                    description="全局 Tokens 指所有企业用户使用所有模型所消耗的总 Tokens 数量，达到上限后将暂停服务"
                    rules={globalTokenRules}
                    onRulesChange={handleGlobalTokenRulesChange}
                    timeDimension={globalTokenTimeDim}
                    onTimeDimensionChange={persistGlobalTokenTimeDim}
                    scope="global"
                  />
                </div>
              </div>
            </section>

            <section>
              <SectionTitle className="mb-4">功能权限开关</SectionTitle>
              <div className="space-y-4">
                <div id="plan4-configModel" className={`rounded-[4px] transition-shadow ${highlightKey === "configModel" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-config-model.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「配置模型」" description="开启后，用户可在 Agent 详细配置中自行选择和切换模型。关闭后，模型配置区域将锁定，用户无法调整（适用于管理员已统一预配置模型的场景）" rules={configModelRules} onRulesChange={setConfigModelRules} />
                </div>
                <div id="plan4-configChannel" className={`rounded-[4px] transition-shadow ${highlightKey === "configChannel" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-config-channel.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「配置通道」" description="开启后，用户可在 Agent 详细配置中自行添加和管理通道。关闭后，通道配置区域将锁定，用户无法调整（适用于管理员已统一预配置通道的场景）" rules={configChannelRules} onRulesChange={setConfigChannelRules} />
                </div>
                <div id="plan4-customModel" className={`rounded-[4px] transition-shadow ${highlightKey === "customModel" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-custom-model.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「添加自定义模型」" description="开启后，用户可在 Agent 中自行添加自定义模型，不在企业管控和 Tokens 覆盖范围内（注意需要先开启「配置模型」）" rules={customModelRules} onRulesChange={setCustomModelRules} />
                </div>
                <div id="plan4-terminal" className={`rounded-[4px] transition-shadow ${highlightKey === "terminal" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-agent-terminal.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「进入 Agent 终端」" description="开启后，所有用户在用户端可看到「进入终端」选项，进入对应 Agent 云服务器的终端" rules={terminalRules} onRulesChange={setTerminalRules} />
                </div>
                <div id="plan4-selfUpgrade" className={`rounded-[4px] transition-shadow ${highlightKey === "selfUpgrade" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-agent-self-upgrade.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「自助更新版本」" description="开启后，员工可在 Agent 详细配置中点击「一键更新」自助更新到管理员设置的版本。关闭后，所有更新动作只能由管理员推送或批量发起" rules={selfUpgradeRules} onRulesChange={handleSelfUpgradeRulesChange} />
                </div>
                <div id="plan4-panel" className={`rounded-[4px] transition-shadow ${highlightKey === "panel" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-agent-panel.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「访问 Agent 面板」" description="开启后，系统会为企业分配一个随机端口并自动添加一条安全组规则放通该端口，用户可通过该端口访问 Agent 面板" rules={panelRules} onRulesChange={handlePanelRulesChange} loadingRuleId={panelLoadingRuleId} accessModeRow={hasSecurityGroup ? { mode: panelAccessMode, onModeChange: (m) => { setPanelAccessMode(m); localStorage.setItem("admin_panel_access_mode", m); }, tooltipContent: "选择用户访问 Agent 面板的网络方式" } : undefined} onBeforeEdit={() => { if (!hasSecurityGroup) { setPanelSgCheckFailed(true); pushAdminNotification({ message: "开启「访问 Agent 面板」失败：尚未配置安全组，请先前往网络管理/安全组配置至少一个安全组后再开启", category: "failure", actionHref: "/admin/security-group?tab=security", actionLabel: "前往配置", dedupeKey: "platform-policy:panel:no-security-group" }); return false; } setPanelSgCheckFailed(false); }} disabledMessage={panelSgCheckFailed ? <>请先前往 <button onClick={() => navigate("/admin/security-group?tab=security")} className="text-[var(--text-brand)] hover:underline">网络管理/安全组</button> 配置至少一个安全组，再开启该功能</> : undefined} extraContent={hasSecurityGroup && panelPort ? (<Alert variant="info" className="w-full"><AlertInfoIcon /><AlertDescription>{panelSgRuleId ? `已为您分配随机端口 ${panelPort} 并自动为默认安全组添加该端口放通规则，` : `已为您分配随机端口 ${panelPort}，`}如用户端仍无法访问面板，请在网络管理的<button onClick={() => navigate("/admin/security-group")} className="underline underline-offset-2 font-medium hover:text-blue-900 transition-colors mx-0.5">安全组规则</button>处检查是否生效</AlertDescription></Alert>) : undefined} />
                </div>
                <div id="plan4-chatView" className={`rounded-[4px] transition-shadow ${highlightKey === "chatView" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-chat-view.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「使用对话视图」" description="开启后，用户可在「我的 Agent」中使用对话视图，通过浏览器与 AI 对话（建议提前配置默认模型，用户创建 Agent 后 AI 即可正常回复）" rules={chatViewRules} onRulesChange={setChatViewRules} />
                </div>
                <div id="plan4-cloudBrowser" className={`rounded-[4px] transition-shadow ${highlightKey === "cloudBrowser" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-cloud-browser.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「访问云桌面」" description="开启后，用户可在对话视图里访问云桌面，查看 AI 浏览器执行过程并进入操作（注意需要先开启「对话视图」）" rules={cloudBrowserRules} onRulesChange={handleCloudBrowserRulesChange} onBeforeEdit={() => { if (!hasSecurityGroup) { setCloudBrowserSgCheckFailed(true); pushAdminNotification({ message: "开启「访问云桌面」失败：尚未配置安全组，请先前往网络管理/安全组配置至少一个安全组后再开启", category: "failure", actionHref: "/admin/security-group?tab=security", actionLabel: "前往配置", dedupeKey: "platform-policy:cloud-browser:no-security-group" }); return false; } setCloudBrowserSgCheckFailed(false); }} disabledMessage={cloudBrowserSgCheckFailed ? <>请先前往 <button onClick={() => navigate("/admin/security-group?tab=security")} className="text-[var(--text-brand)] hover:underline">网络管理/安全组</button> 配置至少一个安全组，再开启该功能</> : undefined} extraContent={hasSecurityGroup && isCloudBrowserEnabled(cloudBrowserRules) && cloudBrowserSgRuleId ? (<Alert variant="info" className="w-full"><AlertInfoIcon /><AlertDescription>已为您当前的安全组添加该功能所需的 6080 端口放通规则，如用户端仍无法访问，请在网络管理的<button onClick={() => navigate("/admin/security-group")} className="underline underline-offset-2 font-medium hover:text-blue-900 transition-colors mx-0.5">安全组规则</button>处检查是否生效</AlertDescription></Alert>) : undefined} />
                </div>
                <div id="plan4-lobsterDoctor" className={`rounded-[4px] transition-shadow ${highlightKey === "lobsterDoctor" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-lobster-doctor.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「使用龙虾医生」" description={<>开启后，所有用户在用户端可免费使用「龙虾医生」AI 诊断功能，自动检测并对话式修复 Agent 运行问题。<span className="text-[#020617] font-medium">龙虾医生每次诊断会产生费用消耗</span>，详见 <button onClick={(e) => { e.stopPropagation(); setShowLobsterDoctorDialog(true); }} className="text-[var(--text-brand)] hover:underline">使用说明</button></>} rules={lobsterDoctorRules} onRulesChange={handleLobsterDoctorRulesChange} />
                </div>
                <div id="plan4-modelQuota" className={`rounded-[4px] transition-shadow ${highlightKey === "modelQuota" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-model-quota.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「查看模型额度」" description="开启后，用户可在顶部导航栏看到「模型额度」入口，查看个人的 Token 使用情况" rules={modelQuotaRules} onRulesChange={setModelQuotaRules} />
                </div>
                <div id="plan4-localClientAccess" className={`rounded-[4px] transition-shadow ${highlightKey === "localClientAccess" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-local-client.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「接入外部 Agent」" description="开启后，用户可接入外部 Agent，并通过 Reporter 上报至平台统一管理" rules={localClientAccessRules} onRulesChange={handleLocalClientAccessRulesChange} />
                </div>
                <div id="plan4-shareAgent" className={`rounded-[4px] transition-shadow ${highlightKey === "shareAgent" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-share-agent.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「共享 agent」" description="开启后，用户可在用户端创建 agent 时选择共享自己的 agent，也可对存量云端 agent 修改共享范围" rules={shareAgentRules} onRulesChange={handleShareAgentRulesChange} />
                </div>
                <div id="plan4-projectEdit" className={`rounded-[4px] transition-shadow ${highlightKey === "projectEdit" ? "anchor-highlight" : ""}`}>
                  <TogglePolicyCard icon={<img src="/assets/admin-platform-policy/allow-share-agent.svg" className="shrink-0 w-10" />} iconBg="" title="允许用户「编辑项目」" description="开启后，项目管理员可在用户端修改项目名称、描述、目标和成员资产编辑权限。关闭后，用户端项目保持只读" rules={projectEditRules} onRulesChange={handleProjectEditRulesChange} />
                </div>
              </div>
            </section>
            {/* 底部占位，确保最后的卡片也能滚动到顶部 */}
            <div className="h-[10vh] shrink-0" />
          </div>

          {/* 右侧锚点导航 - 滚动到顶部后吸顶
              停服态下属于查看/导航类操作，保持可用（data-billing-exempt 豁免全局禁用层） */}
          <div className="w-[16vw] shrink-0 self-start sticky top-0 z-10" data-billing-exempt>
            <div className="max-h-[calc(100vh-32px)] overflow-y-auto py-2">
              {/* 导航列表 */}
              <div>
                <p className="text-[11px] text-[var(--text-weak)] pl-3 py-1.5 uppercase tracking-wide">配额设置</p>
                <div className="ml-3 relative before:absolute before:left-0 before:top-[8px] before:bottom-[8px] before:w-px before:bg-[var(--border)]">
                  {/* 滑动高亮指示条 */}
                  {(() => {
                    const idx = quotaCards.findIndex(c => c.key === activeAnchor);
                    if (idx < 0) return null;
                    const ITEM_H = 32; // py-2 (8+8) + text-xs line-height 16 = 32
                    return (
                      <span
                        aria-hidden
                        className="absolute left-0 top-0 w-[2px] h-4 bg-[#020617] rounded-full transition-transform duration-300 ease-out"
                        style={{ transform: `translateY(${idx * ITEM_H + (ITEM_H - 16) / 2}px)` }}
                      />
                    );
                  })()}
                  {quotaCards.map((card, idx) => (
                    <button
                      key={card.key}
                      type="button"
                      onClick={() => {
                        setActiveAnchor(card.key);
                        triggerHighlight(card.key);
                        const targetId = idx === 0 ? 'plan4-section-quota' : `plan4-${card.key}`;
                        scrollToAnchor(targetId);
                      }}
                      className={`group block w-full text-left pl-4 pr-4 py-2 text-xs whitespace-nowrap transition-colors relative ${activeAnchor === card.key ? "text-[var(--text-emphasis)] font-medium" : "text-[var(--text-emphasis)]"}`}
                    >
                      <span aria-hidden className="pointer-events-none absolute left-[5px] right-[3px] top-0 bottom-0 rounded-[4px] bg-transparent group-hover:bg-white/50 transition-colors" />
                      <span className="relative">{card.title}</span>
                    </button>
                  ))}
                </div>
                <p className="text-[11px] text-[var(--text-weak)] pl-3 py-1.5 mt-3 uppercase tracking-wide">功能权限开关</p>
                <div className="ml-3 relative before:absolute before:left-0 before:top-[8px] before:bottom-[8px] before:w-px before:bg-[var(--border)]">
                  {/* 滑动高亮指示条 */}
                  {(() => {
                    const idx = toggleCards.findIndex(c => c.key === activeAnchor);
                    if (idx < 0) return null;
                    const ITEM_H = 32; // py-2 (8+8) + text-xs line-height 16 = 32
                    return (
                      <span
                        aria-hidden
                        className="absolute left-0 top-0 w-[2px] h-4 bg-[#020617] rounded-full transition-transform duration-300 ease-out"
                        style={{ transform: `translateY(${idx * ITEM_H + (ITEM_H - 16) / 2}px)` }}
                      />
                    );
                  })()}
                  {toggleCards.map((card) => (
                    <button
                      key={card.key}
                      type="button"
                      onClick={() => {
                        setActiveAnchor(card.key);
                        triggerHighlight(card.key);
                        scrollToAnchor(`plan4-${card.key}`);
                      }}
                      className={`group block w-full text-left pl-4 pr-4 py-2 text-xs whitespace-nowrap transition-colors relative ${activeAnchor === card.key ? "text-[var(--text-emphasis)] font-medium" : "text-[var(--text-emphasis)]"}`}
                    >
                      <span aria-hidden className="pointer-events-none absolute left-[5px] right-[3px] top-0 bottom-0 rounded-[4px] bg-transparent group-hover:bg-white/50 transition-colors" />
                      <span className="relative">{card.navTitle}</span>
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>

      {/* 龙虾医生详情弹窗 */}
      <Dialog open={showLobsterDoctorDialog} onOpenChange={setShowLobsterDoctorDialog}>
        <DialogContent className="sm:max-w-[560px]">
          <DialogHeader>
            <DialogTitle>龙虾医生使用说明</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6 space-y-5 text-[14px] text-[#334155] leading-relaxed">
            <Alert variant="info">
              <AlertInfoIcon />
              <AlertDescription>龙虾医生每次诊断会产生部分底层资源费用和 Token 消耗，请注意费用消耗</AlertDescription>
            </Alert>
            {/* 费用消耗说明 */}
            <div className="space-y-3">
              <p className="text-[14px] font-semibold text-[#020617]">费用消耗说明</p>
              <ol className="space-y-2 pl-5 list-decimal text-[14px] text-[#334155]">
                <li><span className="font-medium text-[#020617]">资源费用：</span>底层云资源费用可在 <a href="https://console.cloud.tencent.com/expense" target="_blank" rel="noopener noreferrer" className="text-[var(--text-brand)] hover:underline">腾讯云费用中心</a> 查看</li>
                <li><span className="font-medium text-[#020617]">Token 消耗：</span>诊断消耗的 Token 计入对应用户的 Token 消耗，可在 <button onClick={() => { setShowLobsterDoctorDialog(false); navigate("/admin/tokens-monitor"); }} className="text-[var(--text-brand)] hover:underline">Tokens 监控</button> 查看</li>
              </ol>
            </div>
            {/* 诊断模型配置 */}
            <div className="space-y-3">
              <p className="text-[14px] font-semibold text-[#020617]">诊断模型配置</p>
              <div className="space-y-3">
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-[#020617]">主模型</label>
                  <Select value={lobsterPrimaryModel} onValueChange={setLobsterPrimaryModel}>
                    <SelectTrigger className="w-full h-9 text-sm">
                      <SelectValue placeholder="选择主模型" />
                    </SelectTrigger>
                    <SelectContent>
                <SelectItem value="__default__">
                        {defaultModel ? (
                          <span><span className="font-semibold">跟随默认配置模型</span>（{defaultModel.name} {defaultModel.version}）</span>
                        ) : (
                          <span className="font-semibold">跟随默认配置模型</span>
                        )}
                      </SelectItem>
                      {enabledModels
                        .filter((m) => !m.isDefault)
                        .map((m) => (
                          <SelectItem key={m.id} value={m.id}>
                            {m.name} {m.version}
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-1.5">
                  <label className="text-xs font-medium text-[#020617]">备用模型（可选）</label>
                  <Select value={lobsterBackupModel} onValueChange={setLobsterBackupModel}>
                    <SelectTrigger className="w-full h-9 text-sm">
                      <SelectValue placeholder="选择备用模型" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__none__">未设置</SelectItem>
                      {enabledModels
                        .filter((m) => {
                          if (lobsterPrimaryModel === "__default__") return !m.isDefault;
                          return m.id !== lobsterPrimaryModel;
                        })
                        .map((m) => (
                          <SelectItem key={m.id} value={m.id}>
                            {m.name} {m.version}
                          </SelectItem>
                        ))}
                    </SelectContent>
                  </Select>
                </div>
                <p className="text-xs text-gray-400">主模型不可用时将自动切换至备用模型</p>
              </div>
            </div>
            {/* 工作原理 */}
            <div className="space-y-3">
              <p className="text-[14px] font-semibold text-[#020617]">工作原理</p>
              <p className="text-[14px] text-[#334155]">当用户点击「开始诊断」后，ClawPro 平台将完成以下步骤：</p>
              <ol className="space-y-2 pl-5 list-decimal text-[14px] text-[#334155]">
                <li>创建一个临时按量计费的龙虾医生 Agent 节点</li>
                <li>通过该节点对用户的目标 Agent 进行检测和修复</li>
                <li>诊断结束后，临时节点自动销毁，不留存任何数据</li>
              </ol>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="claw-outline"
              size="claw-sm"
              onClick={() => setShowLobsterDoctorDialog(false)}
            >
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              size="claw-sm"
              onClick={() => {
                toast.success("诊断模型配置已保存");
                setShowLobsterDoctorDialog(false);
              }}
            >
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
