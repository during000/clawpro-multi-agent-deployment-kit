import React, { useState, useRef, useLayoutEffect } from "react";
import { useFileManagementPortalBillingExempt } from "./FileManagement/useFileManagementPortalBillingExempt";
import { useAdminDisabled } from "@/hooks/useAdminDisabled";
import { Pagination } from "@/components/ui/pagination";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogBody,
  DialogDescription,
} from "@/components/ui/dialog";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell } from "@/components/ui/table";
import { GroupSelect } from "@/components/GroupSelect";
import { Alert, AlertTitle, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { HelperText } from "@/components/ui/Typography";
import { Separator } from "@/components/ui/separator";
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  Popover, PopoverContent, PopoverTrigger,
} from "@/components/ui/popover";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { StatusTag } from "@/components/ui/status-tag";
import { toast } from "sonner";
import { 
  Search, 
  Bot,
  Building,
  ChevronDown,
  ChevronRight,
  AlertTriangle,
  CircleAlert,
  Info,
  ChevronLeft,
  ShoppingCart,
  Trash2,
  RotateCcw,
  RotateCw,
  RefreshCw,
  Plus,
  Pencil,
  X,
  ArrowDownToLine,
} from "lucide-react";
import { MOCK_GROUPS as MOCK_ONEID_GROUPS, MOCK_MANUAL_GROUPS } from "./MemberManagement/mock";
import { buildGroupTree, type GroupTreeNode } from "./MemberManagement/health";
import type { UserGroup, GroupSource } from "./MemberManagement/types";
import { MOCK_GROUP_TREE_MANUAL, type GroupNode } from "@/lib/mockData";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { TreeSelect } from "@/components/ui/tree-select";
import {
  GuidePointBubble,
  buildPersistenceKey,
  isDismissed,
  markDismissed,
  resolveBehavior,
  trackOnboarding,
  useBubbleQueue,
  useExposure,
} from "@/components/onboarding";

// ─── creator → 组织 ID 映射（普通模式下，与 MemberManagement mock 对齐） ──────
const CREATOR_GROUP_MAP: Record<string, string> = {
  "noah@acompany.com":  "mgrp-rd",
  "mia@acompany.com":   "mgrp-design",
  "leo@acompany.com":   "mgrp-product",
  "emma@acompany.com":  "mgrp-rd-fe",
  "alice@acompany.com": "mgrp-product",
  "bob@acompany.com":   "mgrp-rd-be",
  "carol@acompany.com": "mgrp-design",
  "david@acompany.com": "mgrp-ops",
  "frank@acompany.com": "mgrp-rd-fe",
  "grace@acompany.com": "mgrp-rd-be",
  "helen@acompany.com": "mgrp-rd-fe",
  "ivan@acompany.com":  "mgrp-rd-fe",
  "jason@acompany.com": "mgrp-rd-be",
  "kelly@acompany.com": "mgrp-rd-be",
  "lisa@acompany.com":  "mgrp-design",
  "tom@acompany.com":   "mgrp-ops",
  "amy@acompany.com":   "mgrp-product",
  "mike@acompany.com":  "mgrp-rd-be",
  "kate@acompany.com":  "mgrp-rd-fe",
  "ryan@acompany.com":  "mgrp-rd",
};

// ─── 组织筛选器（使用项目标准 TreeSelect 组件） ────────────────────────

// ─── PolicyRule 类型 ─────────────────────────────────────────────────────────

interface PolicyRule<T> {
  id: string;
  groupIds: string[];
  value: T;
}

// ─── 行容器样式常量 ──────────────────────────────────────────────────────────
const FM_ROW_CLASS = "flex items-center gap-3 px-3 h-10";
const FM_EDIT_ROW_CLASS = "flex items-start gap-3 px-3 min-h-10 py-1.5";

// ─── 组织选择器（委托给公共 GroupSelect 组件） ────────────────────────────────
function FMGroupTagSelector({
  selectedIds,
  disabledIds = [],
  onChange,
}: {
  selectedIds: string[];
  disabledIds?: string[];
  onChange: (ids: string[]) => void;
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
    />
  );
}

// ─── 组织名称展示 ────────────────────────────────────────────────────────────
function FMGroupBadges({ groupIds, maxVisible = 5 }: { groupIds: string[]; maxVisible?: number }) {
  const allGroups: UserGroup[] = [...MOCK_ONEID_GROUPS, ...MOCK_MANUAL_GROUPS];
  const paths = groupIds.map((id) => allGroups.find((g) => g.id === id)?.name ?? id);

  if (groupIds.length === 0) return <span className="text-xs text-[var(--text-muted)] font-medium">预设策略</span>;

  const visiblePaths = paths.slice(0, maxVisible);
  const rest = paths.length - visiblePaths.length;
  const tooltipText = paths.join("\n");

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
      <TooltipContent className="max-w-[320px] text-xs leading-relaxed whitespace-pre-line">
        {tooltipText}
      </TooltipContent>
    </Tooltip>
  );
}

// ─── 策略概览卡片 ─────────────────────────────────────────────────────────────
interface PolicyOverviewCardProps {
  icon: React.ReactNode;
  iconBg?: string;
  title: string;
  description: string;
  fallbackValue: boolean;
  groupCount: number;
  onClick?: () => void;
}

function PolicyOverviewCard({ icon, iconBg, title, description, fallbackValue, groupCount, onClick }: PolicyOverviewCardProps) {
  return (
    <Card className="overflow-hidden py-0 gap-0 flex flex-col cursor-pointer hover:border-[#1447E6] transition-colors" onClick={onClick}>
      <div className="px-5 pt-5 pb-4 flex-1 min-h-0 flex flex-col">
        <div className="flex items-start gap-3">
          <div className={`shrink-0 ${iconBg ? `w-8 h-8 rounded-[4px] flex items-center justify-center ${iconBg}` : ''}`}>{icon}</div>
          <div className="min-w-0 flex-1">
            <h3 className="text-[14px] font-semibold text-[var(--text-emphasis)] truncate">{title}</h3>
            <p className="text-[12px] text-[var(--text-muted)] leading-relaxed mt-1 line-clamp-2">{description}</p>
          </div>
        </div>

        {/* 底部灰色摘要条 */}
        <div className="mt-4 rounded-[4px] bg-[#FAFBFD] px-3 py-2 flex items-center justify-between">
          <div className="flex items-center gap-4 text-[12px]">
            <span className="text-[var(--text-muted)] inline-flex items-center gap-1">预设策略：<StatusTag mode="fill" variant={fallbackValue ? "green" : "gray"}>{fallbackValue ? "开启" : "关闭"}</StatusTag></span>
            <span className="text-[var(--text-muted)]">组织策略：<span className="text-[var(--text-emphasis)] font-medium">{groupCount} 条</span></span>
          </div>
          <span className="text-[12px] text-[#1447E6] inline-flex items-center gap-0.5">
            编辑策略<ChevronRight className="w-3.5 h-3.5" />
          </span>
        </div>
      </div>
    </Card>
  );
}

// ─── TogglePolicyCard ────────────────────────────────────────────────────────
interface TogglePolicyCardProps {
  icon: React.ReactNode;
  iconBg: string;
  title: string;
  description: string;
  rules: PolicyRule<boolean>[];
  onRulesChange: (rules: PolicyRule<boolean>[]) => boolean | void;
}

/**
 * 停服态「局部禁用」压制样式（同 MemoryManagement 方案）。
 * 背景：本页 useFileManagementPortalBillingExempt 给 dialog-content 打
 * [data-billing-exempt] 后，AdminDisabledOverlay 的恢复规则
 *...[data-slot="dialog-content"][data-billing-exempt] * { opacity:1 !important }
 * 会用「后代通配 + !important」把弹窗内所有元素（含我们灰化的禁用按钮）
 * 强制恢复正常态。此处注入特异性更高 (0,4,1) 的规则把 .suspend-blocked-el
 * 重新压回 opacity:0.4 + pointer-events:none，外层 .suspend-blocked-wrap 保持可点承接 toast。
 */
const FM_SUSPEND_BLOCKED_STYLE_ID = "fm-suspend-blocked-style";
const FM_SUSPEND_BLOCKED_CSS = `
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

function ensureFmSuspendBlockedStyle(): void {
  if (typeof document === "undefined") return;
  if (document.getElementById(FM_SUSPEND_BLOCKED_STYLE_ID)) return;
  const style = document.createElement("style");
  style.id = FM_SUSPEND_BLOCKED_STYLE_ID;
  style.textContent = FM_SUSPEND_BLOCKED_CSS;
  document.head.appendChild(style);
}

function FMTogglePolicyCard({ icon, iconBg, title, description, rules, onRulesChange }: TogglePolicyCardProps) {
  // 停服态：整个弹窗保持可用（Portal 豁免hook 已在页面顶层挂载），但弹窗内红框区域
  // （"应用范围/权限/操作"表格 + "+ 添加组织策略"按钮）需禁止操作 —— 采用
  // "外层 <span> 承接 click 弹 toast + 内层 disabled 按钮"的分层结构，pointer-events:none
  // 仅加在内层，外层 span 保持可点。对齐仓库 SecurityGroupManagement.tsx 的做法。
  // 编辑态下的"取消/保存"分支不做拦截：停服态下"编辑"入口已被拦截，编辑态运行时不可达。
  const { isAdminDisabled } = useAdminDisabled();
  const showSuspendedToast = () => toast.info("管控台已到期，请续费后操作");
  // 注入压制样式，盖过 Portal 豁免恢复规则对禁用按钮的强制还原
  if (isAdminDisabled) ensureFmSuspendBlockedStyle();

  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftGroupIds, setDraftGroupIds] = useState<string[]>([]);
  const [draftValue, setDraftValue] = useState<boolean>(true);
  const [addingNew, setAddingNew] = useState(false);
  const [confirmFallbackDraft, setConfirmFallbackDraft] = useState<boolean | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);

  const getDisabledIds = (excludeRuleId?: string) =>
    rules.filter((r) => r.groupIds.length > 0 && r.id !== excludeRuleId).flatMap((r) => r.groupIds);

  const fallbackRule = rules.find((r) => r.groupIds.length === 0)!;
  const groupRules = rules.filter((r) => r.groupIds.length > 0);
  const groupRuleValue = !fallbackRule.value;

  const startEdit = (rule: PolicyRule<boolean>) => {
    setEditingId(rule.id);
    setDraftGroupIds([...rule.groupIds]);
    setDraftValue(rule.groupIds.length === 0 ? rule.value : groupRuleValue);
    setAddingNew(false);
  };
  const startAdd = () => { setAddingNew(true); setEditingId(null); setDraftGroupIds([]); setDraftValue(groupRuleValue); };
  const cancelEdit = () => { setEditingId(null); setAddingNew(false); };

  const saveEdit = (ruleId?: string) => {
    if (addingNew) {
      if (draftGroupIds.length === 0) { toast.error("请选择至少一个组织"); return; }
      const result = onRulesChange([...groupRules, { id: `rule-${Date.now()}`, groupIds: draftGroupIds, value: groupRuleValue }, fallbackRule]);
      if (result === false) return;
      toast.success("策略已保存"); cancelEdit(); return;
    }
    if (!ruleId) return;
    if (ruleId === fallbackRule.id) {
      if (draftValue !== fallbackRule.value && groupRules.length > 0) { setConfirmFallbackDraft(draftValue); return; }
      const result = onRulesChange(rules.map((r) => r.id === ruleId ? { ...r, value: draftValue } : r));
      if (result === false) return;
      toast.success("策略已保存"); cancelEdit(); return;
    }
    const result = onRulesChange(rules.map((r) => r.id === ruleId ? { ...r, groupIds: draftGroupIds, value: groupRuleValue } : r));
    if (result === false) return;
    toast.success("策略已保存"); cancelEdit();
  };

  const handleConfirmFallbackSwitch = () => {
    if (confirmFallbackDraft === null) return;
    const result = onRulesChange([{ ...fallbackRule, value: confirmFallbackDraft }]);
    if (result !== false) { toast.success("已更新预设策略，组织策略已清空"); cancelEdit(); }
    setConfirmFallbackDraft(null);
  };

  const deleteRule = (ruleId: string) => {
    const result = onRulesChange(rules.filter((r) => r.id !== ruleId));
    if (result === false) return;
    toast.success("策略已删除");
  };

  return (
    <>
      {/* ── 卡片 ── */}
      <PolicyOverviewCard
        icon={icon}
        iconBg={iconBg}
        title={title}
        description={description}
        fallbackValue={fallbackRule.value}
        groupCount={groupRules.length}
        onClick={() => setDialogOpen(true)}
      />

      {/* ── 弹窗：表格行内编辑 ── */}
      <Dialog open={dialogOpen} onOpenChange={(v) => { setDialogOpen(v); if (!v) cancelEdit(); }}>
        <DialogContent className="sm:max-w-[960px]">
          <DialogHeader>
            <DialogTitle>{title}</DialogTitle>
          </DialogHeader>
          <DialogBody className="px-6">
            <div className="space-y-3">
              {/* 汇总行 */}
              <div className="flex items-center gap-4 text-[13px]">
                <span className="text-[var(--text-muted)] inline-flex items-center gap-1">预设策略：<StatusTag mode="fill" variant={fallbackRule.value ? "green" : "gray"}>{fallbackRule.value ? "开启" : "关闭"}</StatusTag></span>
                <span className="text-[var(--text-muted)]">组织策略：<span className="text-[var(--text-emphasis)] font-medium">{groupRules.length} 个</span></span>
              </div>

              {/* 表格 */}
              <div className="rounded-[4px] border border-[var(--border)] overflow-hidden">
              <Table density="compact" variant="gray-header">
                <colgroup><col /><col style={{ width: 100 }} /><col style={{ width: 100 }} /></colgroup>
                <TableHeader>
                  <TableRow><TableHead>应用范围</TableHead><TableHead>权限</TableHead><TableHead>操作</TableHead></TableRow>
                </TableHeader>
                <TableBody>
                  {/* 预设策略行 */}
                  <TableRow>
                    <TableCell>
                      {groupRules.length > 0 ? "全部用户（组织策略用户除外）" : "全部用户"}
                    </TableCell>
                    <TableCell>
                      {editingId === fallbackRule.id ? (
                        <Select value={draftValue ? "on" : "off"} onValueChange={(v) => setDraftValue(v === "on")}>
                          <SelectTrigger className="h-7 w-[80px] text-xs">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="on">开启</SelectItem>
                            <SelectItem value="off">关闭</SelectItem>
                          </SelectContent>
                        </Select>
                      ) : (
                        <StatusTag mode="fill" variant={fallbackRule.value ? "green" : "gray"}>{fallbackRule.value ? "开启" : "关闭"}</StatusTag>
                      )}
                    </TableCell>
                    <TableActionCell>
                      {editingId === fallbackRule.id ? (
                        <>
                          <Button variant="link" onClick={cancelEdit}>取消</Button>
                          <Button variant="link" onClick={() => saveEdit(fallbackRule.id)}>保存</Button>
                        </>
                      ) : isAdminDisabled ? (
                        // 停服态：外层 span弹 toast，内层 Button disabled + 强制原色（#355EF1）+ opacity 0.4
                        <span
                          role="button"
                          onClick={showSuspendedToast}
                          className="suspend-blocked-wrap inline-flex cursor-not-allowed"
                          title="管控台已到期，请续费后操作"
                        >
                          <Button
                            variant="link"
                            disabled
                            style={{ pointerEvents: "none", opacity: 0.4 }}
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
                          <FMGroupBadges groupIds={rule.groupIds} />
                        )}
                      </TableCell>
                      <TableCell>
                        <StatusTag mode="fill" variant={groupRuleValue ? "green" : "gray"}>{groupRuleValue ? "开启" : "关闭"}</StatusTag>
                      </TableCell>
                      <TableActionCell>
                        {editingId === rule.id ? (
                          <>
                            <Button variant="link" onClick={cancelEdit}>取消</Button>
                            <Button variant="link" onClick={() => saveEdit(rule.id)}>保存</Button>
                          </>
                        ) : isAdminDisabled ? (
                          // 停服态：编辑/删除均拦截
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
                                style={{ pointerEvents: "none", opacity: 0.4 }}
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
                                style={{ pointerEvents: "none", opacity: 0.4 }}
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
                        <StatusTag mode="fill" variant={groupRuleValue ? "green" : "gray"}>{groupRuleValue ? "开启" : "关闭"}</StatusTag>
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
              {isAdminDisabled ? (
                <span
                  role="button"
                  onClick={showSuspendedToast}
                  className="suspend-blocked-wrap block w-full cursor-not-allowed"
                  title="管控台已到期，请续费后操作"
                >
                  <button
                    type="button"
                    disabled
                    style={{ pointerEvents: "none", opacity: 0.4 }}
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
          <AlertDialogDescription>组织策略是基于「预设策略」的例外设置。切换「预设策略」后，现有组织策略将全部清空，需重新添加。</AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmFallbackSwitch}>确认切换</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

// Updated Mock Data for Enterprise Spaces
const ENTERPRISE_SPACES = [
  { id: "ent-001", name: "Agent 工具库", type: "公共", used: "12GB", quota: "50GB", expiry: "永久有效" },
  { id: "ent-002", name: "初始技能包", type: "公共", used: "8GB", quota: "50GB", expiry: "永久有效" },
];

// Mock Data for Personal Spaces (Flat Structure) - 已去重
const EXTERNAL_DOWNLOAD_TRAFFIC = {
  usedGb: 0,
  quotaGb: 100,
};

const TENCENT_CLOUD_PAYMENT_URL = "https://buy.cloud.tencent.com/tced";
const CAMPAIGN_DAY_MS = 86400000;

function openTencentCloudPaymentPage() {
  const paymentWindow = window.open(
    TENCENT_CLOUD_PAYMENT_URL,
    "_blank",
    "noopener,noreferrer",
  );

  if (!paymentWindow) {
    toast.error("无法打开腾讯云付款页面，请检查浏览器弹窗设置后重试");
    return false;
  }

  paymentWindow.opener = null;
  return true;
}

function isCampaignActive(launchedOn: string, durationDays: number) {
  const launchedAt = new Date(launchedOn).getTime();
  const now = Date.now();
  return Number.isFinite(launchedAt)
    && now >= launchedAt
    && now < launchedAt + durationDays * CAMPAIGN_DAY_MS;
}

const TRAFFIC_GUIDE_UPDATE_ID = "admin-file-traffic-20260727";
const TRAFFIC_GUIDE_LAUNCHED_ON = "2026-07-27T00:00:00";
const TRAFFIC_GUIDE_DURATION_DAYS = 14;
const TRAFFIC_GUIDE_ANALYTICS = {
  updateId: TRAFFIC_GUIDE_UPDATE_ID,
  component: "point-bubble",
  layer: "element",
  scenario: "2.1",
  endpoint: "admin" as const,
};
const TRAFFIC_GUIDE_BEHAVIOR = resolveBehavior("point-bubble", { maxExposures: 2 });
const TRAFFIC_GUIDE_DISMISS_BEHAVIOR = resolveBehavior("point-bubble", {
  showOnce: true,
  maxExposures: 0,
});
const TRAFFIC_GUIDE_EXPOSURE_KEY = buildPersistenceKey("point-bubble", TRAFFIC_GUIDE_UPDATE_ID);
const TRAFFIC_GUIDE_DISMISS_KEY = buildPersistenceKey("point-bubble", `${TRAFFIC_GUIDE_UPDATE_ID}-closed`);

const FILE_MANAGEMENT_GUIDE_LAUNCHED_ON = "2026-07-28T00:00:00";
const FILE_MANAGEMENT_GUIDE_DURATION_DAYS = 14;
const RENEW_EXPAND_GUIDE_UPDATE_ID = "admin-file-renew-expand-20260728";
const RENEW_EXPAND_GUIDE_ANALYTICS = {
  updateId: RENEW_EXPAND_GUIDE_UPDATE_ID,
  component: "point-bubble",
  layer: "element",
  scenario: "2.1/2.2",
  endpoint: "admin" as const,
};
const RENEW_EXPAND_GUIDE_BEHAVIOR = resolveBehavior("point-bubble", { maxExposures: 2 });
const RENEW_EXPAND_GUIDE_DISMISS_BEHAVIOR = resolveBehavior("point-bubble", {
  showOnce: true,
  maxExposures: 0,
});
const RENEW_EXPAND_GUIDE_EXPOSURE_KEY = buildPersistenceKey("point-bubble", RENEW_EXPAND_GUIDE_UPDATE_ID);
const RENEW_EXPAND_GUIDE_DISMISS_KEY = buildPersistenceKey("point-bubble", `${RENEW_EXPAND_GUIDE_UPDATE_ID}-closed`);

const TRAFFIC_PACKAGES = [
  { id: "traffic-20", capacity: "20 GB", description: "适合少量下载与临时文件分发" },
  { id: "traffic-50", capacity: "50 GB", description: "适合轻量下载与日常文件分发" },
  { id: "traffic-100", capacity: "100 GB", description: "适合日常协作与文件分发" },
  { id: "traffic-200", capacity: "200 GB", description: "适合团队协作与中等规模文件分发" },
  { id: "traffic-500", capacity: "500 GB", description: "适合高频下载与大规模文件分发" },
  { id: "traffic-1000", capacity: "1000 GB", description: "适合大流量下载与集中式文件分发" },
];

const PERSONAL_SPACES_DATA = [
  { id: "user-ins-1", instanceId: "ins-u25p9jqg", instanceName: "Noah的分析助手", creator: "noah@acompany.com", avatar: "N", type: "智能体网盘", used: "5GB", quota: "50GB", expiry: "2026-06-30", enabled: false, wasEnabled: true, deletedDaysAgo: 20 }, // 永久删除状态（超过15天）
  { id: "user-ins-3", instanceId: "ins-v88x2kww", instanceName: "Noah的测试沙盒", creator: "noah@acompany.com", avatar: "N", type: "智能体网盘", used: "2GB", quota: "50GB", expiry: "2026-06-30", enabled: false, requiresPaidEnable: true, autoRenew: true },
  { id: "user-ins-4", instanceId: "ins-t14o8ipf", instanceName: "Mia的新助手", creator: "mia@acompany.com", avatar: "M", type: "智能体网盘", used: "5GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-6", instanceId: "ins-s03n7heo", instanceName: "Leo的项目助手", creator: "leo@acompany.com", avatar: "L", type: "智能体网盘", used: "5GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: true },
  { id: "user-ins-7", instanceId: "ins-x11m9zzz", instanceName: "Leo的文档库", creator: "leo@acompany.com", avatar: "L", type: "智能体网盘", used: "15GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-9", instanceId: "ins-p99k3mnn", instanceName: "Emma的数据分析", creator: "emma@acompany.com", avatar: "E", type: "智能体网盘", used: "7GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: true },
  { id: "user-ins-10", instanceId: "ins-q22l4roo", instanceName: "David的代码助手", creator: "david@acompany.com", avatar: "D", type: "智能体网盘", used: "9GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-11", instanceId: "ins-r33m5spp", instanceName: "Sarah的研究工具", creator: "sarah@acompany.com", avatar: "S", type: "智能体网盘", used: "4GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: true },
  { id: "user-ins-12", instanceId: "ins-t44n6tqq", instanceName: "Jack的文案助手", creator: "jack@acompany.com", avatar: "J", type: "智能体网盘", used: "6GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-13", instanceId: "ins-u55o7urr", instanceName: "Lisa的设计工具", creator: "lisa@acompany.com", avatar: "L", type: "智能体网盘", used: "11GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-14", instanceId: "ins-v66p8vss", instanceName: "Tom的营销助手", creator: "tom@acompany.com", avatar: "T", type: "智能体网盘", used: "8GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: true },
  { id: "user-ins-15", instanceId: "ins-w77q9wtt", instanceName: "Amy的翻译工具", creator: "amy@acompany.com", avatar: "A", type: "智能体网盘", used: "3GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-16", instanceId: "ins-x88r0xuu", instanceName: "Mike的产品分析", creator: "mike@acompany.com", avatar: "M", type: "智能体网盘", used: "13GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-17", instanceId: "ins-y99s1yvv", instanceName: "Kate的客服助手", creator: "kate@acompany.com", avatar: "K", type: "智能体网盘", used: "5GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-18", instanceId: "ins-z00t2zww", instanceName: "Ryan的技术文档", creator: "ryan@acompany.com", avatar: "R", type: "智能体网盘", used: "10GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-19", instanceId: "ins-a11u3axv", instanceName: "这是一个名称非常非常长的智能助手用来测试超长文本截断效果", creator: "longname-user@very-long-domain-example.com", avatar: "L", type: "智能体网盘", used: "8GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
  { id: "user-ins-20", instanceId: "ins-b22v4byw", instanceName: "GPULab产品线专属AI智能运营分析与决策支持系统", creator: "product-ops-admin@enterprise-acompany.com", avatar: "G", type: "智能体网盘", used: "22GB", quota: "50GB", expiry: "2026-06-30", enabled: false, autoRenew: false },
];

function EnterpriseSpaceIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M15 2.25C15.7824 2.25 16.417 2.88459 16.417 3.66699V12.333C16.417 14.22 14.887 15.75 13 15.75H1.58301V6.91699H6.25V3.66699C6.25 2.88459 6.88459 2.25 7.66699 2.25H15ZM3.08398 14.25H4.33398C5.39207 14.2497 6.2506 13.392 6.25098 12.334V9H6.25V8.41699H3.08398V14.25ZM7.75 6.91699H7.75098V12.334C7.75084 13.0442 7.53321 13.7036 7.16211 14.25H13C14.0585 14.25 14.917 13.3916 14.917 12.333V3.75H7.75V6.91699ZM13.667 9.08301H9V7.58301H13.667V9.08301ZM13.667 6.41699H9V4.91699H13.667V6.41699Z" fill="url(#paint0_linear_enterprise_space)"/>
      <defs>
        <linearGradient id="paint0_linear_enterprise_space" x1="15.1949" y1="15.9039" x2="7.92473" y2="6.96759" gradientUnits="userSpaceOnUse">
          <stop stopColor="#0080FF"/>
          <stop offset="0.240385" stopColor="#0869C9"/>
          <stop offset="1" stopColor="#202020"/>
        </linearGradient>
      </defs>
    </svg>
  );
}

function TrafficUsageIcon() {
  return (
    <ArrowDownToLine
      className="h-[18px] w-[18px] text-[var(--text-title)]"
      strokeWidth={1.8}
      aria-hidden="true"
    />
  );
}

function AgentDiskIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 18 18" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M4.50781 11.1562C4.97366 11.1563 5.35156 11.534 5.35156 11.9999C5.3515 12.4657 4.97362 12.8435 4.50781 12.8437H4.5C4.03405 12.8437 3.65631 12.4659 3.65625 11.9999C3.65625 11.5339 4.03401 11.1562 4.5 11.1562H4.50781Z" fill="url(#paint0_linear_agent_disk)"/>
      <path d="M7.50781 11.1562C7.97366 11.1563 8.35156 11.534 8.35156 11.9999C8.3515 12.4657 7.97362 12.8435 7.50781 12.8437H7.5C7.03405 12.8437 6.65631 12.4659 6.65625 11.9999C6.65625 11.5339 7.03401 11.1562 7.5 11.1562H7.50781Z" fill="url(#paint1_linear_agent_disk)"/>
      <path fillRule="evenodd" clipRule="evenodd" d="M9 2.15616C9.46599 2.15616 9.84375 2.53392 9.84375 2.99991C9.84369 3.46585 9.46595 3.84366 9 3.84366H5.43066L5.33984 3.8505C5.24975 3.86312 5.16284 3.894 5.08496 3.9423C4.98124 4.00664 4.89709 4.09862 4.84277 4.20792L2.86133 8.16593H15.1387L14.9795 7.84757C14.771 7.43094 14.9399 6.92434 15.3564 6.71573C15.7731 6.50728 16.2797 6.67608 16.4883 7.09269L17.0957 8.3046C17.2589 8.63039 17.3437 8.99001 17.3438 9.35441V13.4999C17.3437 14.1215 17.0967 14.7176 16.6572 15.1571C16.2177 15.5966 15.6216 15.8437 15 15.8437H3C2.37843 15.8437 1.78231 15.5966 1.34277 15.1571C0.903263 14.7176 0.656281 14.1215 0.65625 13.4999V9.35441C0.656306 8.99001 0.74106 8.63039 0.904297 8.3046L3.33301 3.45499C3.52705 3.06543 3.82544 2.73712 4.19531 2.50773C4.56588 2.27793 4.99365 2.15639 5.42969 2.15616H9ZM2.34375 13.4999C2.34378 13.6739 2.41309 13.8407 2.53613 13.9638C2.6592 14.0868 2.82598 14.1562 3 14.1562H15C15.174 14.1562 15.3408 14.0868 15.4639 13.9638C15.5869 13.8407 15.6562 13.6739 15.6562 13.4999V9.85343H2.34375V13.4999Z" fill="url(#paint2_linear_agent_disk)"/>
      <path d="M13.3721 1.04776C13.4631 0.801787 13.8103 0.801787 13.9014 1.04776L14.5908 2.90909L16.4521 3.59855C16.698 3.68964 16.698 4.03679 16.4521 4.12784L14.5908 4.8173L13.9014 6.67862C13.816 6.90922 13.5051 6.92344 13.3916 6.72159L13.3721 6.67862L12.6826 4.8173L10.8213 4.12784C10.5753 4.03682 10.5753 3.68957 10.8213 3.59855L12.6826 2.90909L13.3721 1.04776Z" fill="url(#paint3_linear_agent_disk)"/>
      <defs>
        <linearGradient id="paint0_linear_agent_disk" x1="17" y1="16.5" x2="6.44431" y2="6.59202" gradientUnits="userSpaceOnUse">
          <stop stopColor="#0080FF"/><stop offset="1" stopColor="#202020"/>
        </linearGradient>
        <linearGradient id="paint1_linear_agent_disk" x1="17" y1="16.5" x2="6.44431" y2="6.59202" gradientUnits="userSpaceOnUse">
          <stop stopColor="#0080FF"/><stop offset="1" stopColor="#202020"/>
        </linearGradient>
        <linearGradient id="paint2_linear_agent_disk" x1="17" y1="16.5" x2="6.44431" y2="6.59202" gradientUnits="userSpaceOnUse">
          <stop stopColor="#0080FF"/><stop offset="1" stopColor="#202020"/>
        </linearGradient>
        <linearGradient id="paint3_linear_agent_disk" x1="17" y1="16.5" x2="6.44431" y2="6.59202" gradientUnits="userSpaceOnUse">
          <stop stopColor="#0080FF"/><stop offset="1" stopColor="#202020"/>
        </linearGradient>
      </defs>
    </svg>
  );
}

const StatCard = ({
  title,
  value,
  icon: IconComponent,
  action,
}: {
  title: string;
  value: React.ReactNode;
  icon: React.FC;
  action?: React.ReactNode;
}) => (
  <div className="flex flex-col gap-4 rounded-[4px] border border-[#EAEEF4] bg-white px-6 py-5">
    <div className="flex items-center gap-1">
      <IconComponent />
      <span className="text-sm font-medium text-[var(--text-title)] leading-[22px] tracking-[0.07px]">{title}</span>
    </div>
    <div className="flex min-h-9 items-end justify-between gap-4">
      <p className="text-2xl font-bold text-[var(--text-title)] leading-normal tabular-nums" style={{ fontFamily: "'DIN Next LT Pro', 'DIN', sans-serif" }}>{value}</p>
      {action ? <div className="shrink-0 pb-1">{action}</div> : null}
    </div>
  </div>
);

export default function FileManagement() {
  // 停服态下，本页面各种弹窗（"新增实例是否自动绑定网盘" Dialog、
  // 批量开通 / 单个开通 / 续期 / 扩容 / 购买存储 / 购买流量包 / 回收站 Dialog、
  // 永久删除 AlertDialog 等）都通过 Radix Portal 挂到<body>，脱离主体页面容器；
  // 若不在 dialog-content 上打 data-billing-exempt，会被 AdminDisabledOverlay
  // 视觉灰化 + 文档级 capture 事件拦截 —— 用户连"关闭 X"都点不动，被卡在弹窗里。
  // 详见 ./FileManagement/useFileManagementPortalBillingExempt.ts 头部注释。
  useFileManagementPortalBillingExempt();
  const [searchQuery, setSearchQuery] = useState("");
  const [groupFilter, setGroupFilter] = useState("");
  const [autoBindRules, setAutoBindRules] = useState<PolicyRule<boolean>[]>([
    { id: "autobind-fallback", groupIds: [], value: true },
  ]);
  const [instancesEnabled, setInstancesEnabled] = useState<Record<string, boolean>>(
    PERSONAL_SPACES_DATA.reduce((acc, item) => {
      acc[item.id] = item.enabled;
      return acc;
    }, {} as Record<string, boolean>)
  );
  // 追踪曾经启用过的实例（用于显示"可恢复"状态）
  const [instancesEverEnabled, setInstancesEverEnabled] = useState<Record<string, boolean>>(
    PERSONAL_SPACES_DATA.reduce((acc, item) => {
      // @ts-ignore - 使用 wasEnabled 字段初始化
      acc[item.id] = item.wasEnabled !== undefined ? item.wasEnabled : item.enabled;
      return acc;
    }, {} as Record<string, boolean>)
  );
  // 追踪实例的关闭时间（用于计算剩余天数）
  const [instancesDisabledTime, setInstancesDisabledTime] = useState<Record<string, Date>>(
    PERSONAL_SPACES_DATA.reduce((acc, item) => {
      // @ts-ignore - 如果有 deletedDaysAgo 字段，计算关闭时间
      if (item.deletedDaysAgo !== undefined) {
        const disabledDate = new Date();
        // @ts-ignore
        disabledDate.setDate(disabledDate.getDate() - item.deletedDaysAgo);
        acc[item.id] = disabledDate;
      }
      return acc;
    }, {} as Record<string, Date>)
  );
  // 追踪每个实例的自动续费状态（默认全部关闭，开启网盘后可手动开启）
  const [instancesAutoRenew, setInstancesAutoRenew] = useState<Record<string, boolean>>(
    PERSONAL_SPACES_DATA.reduce((acc, item) => {
      acc[item.id] = false;
      return acc;
    }, {} as Record<string, boolean>)
  );
  const [batchAutoRenewDialogOpen, setBatchAutoRenewDialogOpen] = useState(false);
  const [batchRenewDialogOpen, setBatchRenewDialogOpen] = useState(false);
  const [selectedInstances, setSelectedInstances] = useState<Set<string>>(new Set());
  const [disableDialogOpen, setDisableDialogOpen] = useState(false);
  const [instanceToDisable, setInstanceToDisable] = useState<{ id: string; name: string } | null>(null);
  const [batchEnableDialogOpen, setBatchEnableDialogOpen] = useState(false);
  const [singleEnableDialogOpen, setSingleEnableDialogOpen] = useState(false);
  const [instanceToEnable, setInstanceToEnable] = useState<{ id: string; name: string } | null>(null);
  const [purchaseDialogOpen, setPurchaseDialogOpen] = useState(false);
  const [instanceToPurchase, setInstanceToPurchase] = useState<{ id: string; name: string } | null>(null);
  const [trafficPackageDialogOpen, setTrafficPackageDialogOpen] = useState(false);
  const [selectedTrafficPackage, setSelectedTrafficPackage] = useState(TRAFFIC_PACKAGES[0].id);
  const [trafficGuideOpen, setTrafficGuideOpen] = useState(() => (
    isCampaignActive(TRAFFIC_GUIDE_LAUNCHED_ON, TRAFFIC_GUIDE_DURATION_DAYS)
      && !isDismissed(TRAFFIC_GUIDE_EXPOSURE_KEY, TRAFFIC_GUIDE_BEHAVIOR)
      && !isDismissed(TRAFFIC_GUIDE_DISMISS_KEY, TRAFFIC_GUIDE_DISMISS_BEHAVIOR)
  ));
  const canShowTrafficGuide = useBubbleQueue(TRAFFIC_GUIDE_UPDATE_ID, trafficGuideOpen);
  useExposure(
    trafficGuideOpen && canShowTrafficGuide,
    TRAFFIC_GUIDE_ANALYTICS,
    TRAFFIC_GUIDE_EXPOSURE_KEY,
  );
  const [renewExpandGuideOpen, setRenewExpandGuideOpen] = useState(() => (
    isCampaignActive(FILE_MANAGEMENT_GUIDE_LAUNCHED_ON, FILE_MANAGEMENT_GUIDE_DURATION_DAYS)
      && !isDismissed(RENEW_EXPAND_GUIDE_EXPOSURE_KEY, RENEW_EXPAND_GUIDE_BEHAVIOR)
      && !isDismissed(RENEW_EXPAND_GUIDE_DISMISS_KEY, RENEW_EXPAND_GUIDE_DISMISS_BEHAVIOR)
  ));
  const canShowRenewExpandGuide = useBubbleQueue(RENEW_EXPAND_GUIDE_UPDATE_ID, renewExpandGuideOpen);
  useExposure(
    renewExpandGuideOpen && canShowRenewExpandGuide,
    RENEW_EXPAND_GUIDE_ANALYTICS,
    RENEW_EXPAND_GUIDE_EXPOSURE_KEY,
  );
  const renewExpandGuideAnchorRef = useRef<HTMLDivElement>(null);
  const [renewExpandGuidePosition, setRenewExpandGuidePosition] = useState({ bottom: 0, right: 0 });
  useLayoutEffect(() => {
    if (!renewExpandGuideOpen || !canShowRenewExpandGuide) return;

    const updatePosition = () => {
      const anchor = renewExpandGuideAnchorRef.current;
      if (!anchor) return;
      const rect = anchor.getBoundingClientRect();
      setRenewExpandGuidePosition({
        bottom: window.innerHeight - rect.top + 8,
        right: Math.max(8, window.innerWidth - rect.right),
      });
    };

    updatePosition();
    window.addEventListener("resize", updatePosition);
    window.addEventListener("scroll", updatePosition, true);
    return () => {
      window.removeEventListener("resize", updatePosition);
      window.removeEventListener("scroll", updatePosition, true);
    };
  }, [renewExpandGuideOpen, canShowRenewExpandGuide]);
  const [selectedCapacity, setSelectedCapacity] = useState<string>("50GB");
  const [selectedDuration, setSelectedDuration] = useState<string>("3");
  const [renewDialogOpen, setRenewDialogOpen] = useState(false);
  const [instanceToRenew, setInstanceToRenew] = useState<{ id: string; name: string } | null>(null);
  const [renewingFromRecyclebin, setRenewingFromRecyclebin] = useState(false);
  const [renewDuration, setRenewDuration] = useState<string>("3");
  const [expandDialogOpen, setExpandDialogOpen] = useState(false);
  const [instanceToExpand, setInstanceToExpand] = useState<{ id: string; name: string } | null>(null);
  const [expandCapacity, setExpandCapacity] = useState<string>("100GB");
  const [recyclebinOpen, setRecyclebinOpen] = useState(false);
  const [recyclebinDeleteDialogOpen, setRecyclebinDeleteDialogOpen] = useState(false);
  const [instanceToDeletePermanently, setInstanceToDeletePermanently] = useState<{ id: string; name: string } | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 10;

  const handleToggleInstance = (
    instanceId: string,
    instanceName: string,
    currentEnabled: boolean,
    wasEverEnabled: boolean,
    requiresPaidEnable = false,
  ) => {
    if (currentEnabled) {
      // 如果当前是开启状态，尝试关闭时弹出确认对话框
      setInstanceToDisable({ id: instanceId, name: instanceName });
      setDisableDialogOpen(true);
    } else {
      // 如果当前是关闭状态
      // 检查回收站中是否有该实例自己的网盘（15天内可恢复）
      const hasOwnRecyclableSpace = wasEverEnabled && !isPermanentlyDeleted(instanceId);
      
      if (hasOwnRecyclableSpace) {
        toast.warning("回收站内有相关网盘资源，需要进行处理");
      } else if (requiresPaidEnable) {
        setInstanceToPurchase({ id: instanceId, name: instanceName });
        setPurchaseDialogOpen(true);
      } else {
        // 回收站中没有该实例自己的网盘，直接弹出首次启用确认对话框（不再检查其他实例）
        setInstanceToEnable({ id: instanceId, name: instanceName });
        setSingleEnableDialogOpen(true);
      }
    }
  };

  const handleConfirmDisable = () => {
    if (instanceToDisable) {
      setInstancesEnabled(prev => ({
        ...prev,
        [instanceToDisable.id]: false
      }));
      // 记录关闭时间
      setInstancesDisabledTime(prev => ({
        ...prev,
        [instanceToDisable.id]: new Date()
      }));
    }
    setDisableDialogOpen(false);
    setInstanceToDisable(null);
  };

  const handleCancelDisable = () => {
    setDisableDialogOpen(false);
    setInstanceToDisable(null);
  };

  const handleBatchEnable = () => {
    if (selectedInstances.size > 0) {
      setBatchEnableDialogOpen(true);
    }
  };

  const handleConfirmBatchEnable = () => {
    // 启用所有选中的实例
    const newEnabled = { ...instancesEnabled };
    const newEverEnabled = { ...instancesEverEnabled };
    const newDisabledTimes = { ...instancesDisabledTime };
    let hasRecyclableSpace = false;
    let hasPaidEnableInstance = false;
    selectedInstances.forEach(instanceId => {
      // 已启用的实例跳过（联动选择时可能包含已启用实例）
      if (instancesEnabled[instanceId]) return;
      const instance = PERSONAL_SPACES_DATA.find((item) => item.id === instanceId);
      if (instance?.requiresPaidEnable) {
        hasPaidEnableInstance = true;
        return;
      }
      if (instancesEverEnabled[instanceId] && !isPermanentlyDeleted(instanceId)) {
        hasRecyclableSpace = true;
        return;
      }
      newEnabled[instanceId] = true;
      newEverEnabled[instanceId] = true; // 标记为曾经启用过
      delete newDisabledTimes[instanceId]; // 清除关闭时间
    });
    if (hasRecyclableSpace) {
      toast.warning("回收站内有相关网盘资源，需要进行处理");
    }
    if (hasPaidEnableInstance) {
      toast.warning("付费网盘请通过启用开关购买容量和使用时长");
    }
    setInstancesEnabled(newEnabled);
    setInstancesEverEnabled(newEverEnabled);
    setInstancesDisabledTime(newDisabledTimes);
    setSelectedInstances(new Set()); // 清空选中状态
    setBatchEnableDialogOpen(false);
  };

  const handleCancelBatchEnable = () => {
    setBatchEnableDialogOpen(false);
  };

  // 批量开启自动续费：仅对已启用实例生效
  const handleConfirmBatchAutoRenew = () => {
    setInstancesAutoRenew((prev) => {
      const next = { ...prev };
      selectedInstances.forEach((instanceId) => {
        if (instancesEnabled[instanceId]) {
          next[instanceId] = true;
        }
      });
      return next;
    });
    toast.success(`已为 ${selectedEnabledCount} 个智能体开启自动续费`);
    setSelectedInstances(new Set());
    setBatchAutoRenewDialogOpen(false);
  };

  // 批量续费：对选中的已启用实例跳转腾讯云官方付款页
  const handleConfirmBatchRenew = () => {
    if (selectedEnabledCount === 0 || !openTencentCloudPaymentPage()) return;
    setSelectedInstances(new Set());
    setBatchRenewDialogOpen(false);
  };

  const handleConfirmSingleEnable = () => {
    if (instanceToEnable) {
      setInstancesEnabled(prev => ({
        ...prev,
        [instanceToEnable.id]: true
      }));
      setInstancesEverEnabled(prev => ({
        ...prev,
        [instanceToEnable.id]: true // 标记为曾经启用过
      }));
    }
    setSingleEnableDialogOpen(false);
    setInstanceToEnable(null);
  };

  const handleCancelSingleEnable = () => {
    setSingleEnableDialogOpen(false);
    setInstanceToEnable(null);
  };

  const handleConfirmPurchase = () => {
    if (!instanceToPurchase || !openTencentCloudPaymentPage()) return;

    setPurchaseDialogOpen(false);
    setInstanceToPurchase(null);
    setSelectedCapacity("50GB");
    setSelectedDuration("3");
  };

  const handleCancelPurchase = () => {
    setPurchaseDialogOpen(false);
    setInstanceToPurchase(null);
    // 重置选择
    setSelectedCapacity("50GB");
    setSelectedDuration("3");
  };

  // 计算购买价格
  const calculatePrice = () => {
    const capacityPrices: Record<string, number> = {
      "50GB": 2,
      "100GB": 4,
      "500GB": 8
    };
    const basePrice = capacityPrices[selectedCapacity] || 0;
    const duration = parseInt(selectedDuration);
    return basePrice * duration;
  };

  // 处理续费
  const handleRenew = (instanceId: string, instanceName: string, fromRecyclebin = false) => {
    setInstanceToRenew({ id: instanceId, name: instanceName });
    setRenewingFromRecyclebin(fromRecyclebin);
    setRenewDialogOpen(true);
  };

  const handleConfirmRenew = () => {
    if (!instanceToRenew || !openTencentCloudPaymentPage()) return;

    setRenewDialogOpen(false);
    setInstanceToRenew(null);
    setRenewingFromRecyclebin(false);
    setRenewDuration("3");
  };

  const handleCancelRenew = () => {
    setRenewDialogOpen(false);
    setInstanceToRenew(null);
    setRenewingFromRecyclebin(false);
    setRenewDuration("3");
  };

  // 计算续费价格
  const calculateRenewPrice = () => {
    const basePrice = 2; // 假设当前容量为50GB，单价2元/月
    const duration = parseInt(renewDuration);
    return basePrice * duration;
  };

  // 处理扩容
  const handleExpand = (instanceId: string, instanceName: string) => {
    setInstanceToExpand({ id: instanceId, name: instanceName });
    setExpandDialogOpen(true);
  };

  const handleConfirmExpand = () => {
    if (!instanceToExpand || !openTencentCloudPaymentPage()) return;

    setExpandDialogOpen(false);
    setInstanceToExpand(null);
    setExpandCapacity("50GB");
  };

  const handleCancelExpand = () => {
    setExpandDialogOpen(false);
    setInstanceToExpand(null);
    setExpandCapacity("100GB");
  };

  const expandCapacityOptions = [
    { value: "50GB", label: "50GB", price: 2 },
    { value: "100GB", label: "100GB", price: 4 },
    { value: "500GB", label: "500GB", price: 8 },
  ];

  const calculateExpandPrice = () => (
    expandCapacityOptions.find((option) => option.value === expandCapacity)?.price ?? 0
  );

  // 计算可恢复的剩余天数
  const getRemainingDays = (instanceId: string): number => {
    const disabledTime = instancesDisabledTime[instanceId];
    if (!disabledTime) return 15; // 如果没有关闭时间记录，默认显示15天
    
    const now = new Date();
    const diffTime = now.getTime() - disabledTime.getTime();
    const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));
    const remainingDays = 15 - diffDays;
    
    return Math.max(0, remainingDays); // 确保不返回负数
  };

  // 获取回收站中的实例（已关闭的实例）
  const getRecyclebinInstances = () => {
    return PERSONAL_SPACES_DATA.filter(item => {
      const isDisabled = !instancesEnabled[item.id];
      const wasEnabled = instancesEverEnabled[item.id];
      return isDisabled && wasEnabled;
    }).map(item => ({
      ...item,
      remainingDays: getRemainingDays(item.id),
      isPermanentlyDeleted: isPermanentlyDeleted(item.id)
    }));
  };

  // 从回收站永久删除实例
  const handlePermanentDelete = (instanceId: string) => {
    // 这里可以调用后端API永久删除数据
    console.log('永久删除实例:', instanceId);
    // 从instancesEverEnabled中移除，表示彻底删除
    setInstancesEverEnabled(prev => {
      const newEnabled = { ...prev };
      delete newEnabled[instanceId];
      return newEnabled;
    });
  };

  // 确认永久删除
  const handleConfirmPermanentDelete = () => {
    if (instanceToDeletePermanently) {
      handlePermanentDelete(instanceToDeletePermanently.id);
    }
    setRecyclebinDeleteDialogOpen(false);
    setInstanceToDeletePermanently(null);
  };

  // 取消永久删除
  const handleCancelPermanentDelete = () => {
    setRecyclebinDeleteDialogOpen(false);
    setInstanceToDeletePermanently(null);
  };

  // 直接恢复实例（免费）
  const handleDirectRecover = (instanceId: string) => {
    console.log('直接恢复实例:', instanceId);
    setInstancesEnabled(prev => ({
      ...prev,
      [instanceId]: true
    }));
    // 清除关闭时间记录
    setInstancesDisabledTime(prev => {
      const newTimes = { ...prev };
      delete newTimes[instanceId];
      return newTimes;
    });
  };

  // 从回收站恢复实例时进入续费付款流程
  const handleRestoreFromRecyclebin = (instanceId: string, instanceName: string) => {
    setRecyclebinOpen(false);
    handleRenew(instanceId, instanceName, true);
  };

  // 检查实例是否已永久删除（超过15天）
  const isPermanentlyDeleted = (instanceId: string): boolean => {
    const disabledTime = instancesDisabledTime[instanceId];
    if (!disabledTime) return false;
    
    const now = new Date();
    const diffTime = now.getTime() - disabledTime.getTime();
    const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));
    
    return diffDays >= 15; // 超过或等于15天则永久删除
  };

  // 所有可勾选的实例（排除永久删除）
  const allSelectableInstanceIds = React.useMemo(() => (
    PERSONAL_SPACES_DATA.filter((item) => !isPermanentlyDeleted(item.id)).map((item) => item.id)
  ), [instancesDisabledTime]);

  // 选中集合中：未启用数量、已启用数量
  const selectedDisabledCount = React.useMemo(() => (
    Array.from(selectedInstances).filter((id) => !instancesEnabled[id]).length
  ), [selectedInstances, instancesEnabled]);
  const selectedEnabledCount = React.useMemo(() => (
    Array.from(selectedInstances).filter((id) => instancesEnabled[id]).length
  ), [selectedInstances, instancesEnabled]);

  // 处理全选/取消全选
  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedInstances(new Set(allSelectableInstanceIds));
    } else {
      setSelectedInstances(new Set());
    }
  };

  // 处理单个实例选中
  const handleSelectInstance = (instanceId: string, checked: boolean) => {
    const newSelected = new Set(selectedInstances);
    if (checked) {
      newSelected.add(instanceId);
    } else {
      newSelected.delete(instanceId);
    }
    setSelectedInstances(newSelected);
  };

  // 判断是否全选
  const isAllSelected = allSelectableInstanceIds.length > 0 &&
    allSelectableInstanceIds.every(id => selectedInstances.has(id));

  // 判断是否部分选中
  const isIndeterminate = selectedInstances.size > 0 &&
    selectedInstances.size < allSelectableInstanceIds.length;

  // 计算统计数据
  const stats = React.useMemo(() => {
    // 计算企业公共空间数量
    const enterpriseSpacesCount = ENTERPRISE_SPACES.length;

    // 计算个人空间实例总数（只计算enabled=true的记录）
    const totalPersonalInstances = PERSONAL_SPACES_DATA.filter(item => instancesEnabled[item.id]).length;

    return {
      enterpriseSpacesCount,
      totalPersonalInstances
    };
  }, [instancesEnabled]);

  // 搜索过滤
  const filteredPersonalSpaces = React.useMemo(() => {
    let result = PERSONAL_SPACES_DATA;
    // 组织筛选
    if (groupFilter) {
      // 收集所选组织及其子孙 ID
      const collectIds = (nodes: GroupNode[], targetId: string): string[] => {
        const ids: string[] = [];
        const collect = (node: GroupNode) => { ids.push(node.id); node.children?.forEach(collect); };
        const find = (list: GroupNode[]): boolean => {
          for (const n of list) {
            if (n.id === targetId) { collect(n); return true; }
            if (n.children && find(n.children)) return true;
          }
          return false;
        };
        find(nodes);
        return ids;
      };
      const allowedGroupIds = collectIds(MOCK_GROUP_TREE_MANUAL, groupFilter);
      result = result.filter(item => {
        const g = CREATOR_GROUP_MAP[item.creator];
        return g && allowedGroupIds.includes(g);
      });
    }
    // 关键词搜索
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase().trim();
      result = result.filter(item =>
        item.instanceName.toLowerCase().includes(query) ||
        item.instanceId.toLowerCase().includes(query) ||
        item.creator.toLowerCase().includes(query)
      );
    }
    return result;
  }, [searchQuery, groupFilter]);

  // 计算总页数
  const totalPages = Math.ceil(filteredPersonalSpaces.length / itemsPerPage);

  // 获取当前页数据
  const paginatedPersonalSpaces = React.useMemo(() => {
    const startIndex = (currentPage - 1) * itemsPerPage;
    const endIndex = startIndex + itemsPerPage;
    return filteredPersonalSpaces.slice(startIndex, endIndex);
  }, [filteredPersonalSpaces, currentPage]);

  // 当搜索或组织条件变化时重置到第一页
  React.useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, groupFilter]);

  const dismissTrafficGuide = (event: "onboarding_click" | "onboarding_dismiss") => {
    markDismissed(TRAFFIC_GUIDE_DISMISS_KEY);
    trackOnboarding(event, TRAFFIC_GUIDE_ANALYTICS);
    setTrafficGuideOpen(false);
  };

  const dismissRenewExpandGuide = (event: "onboarding_click" | "onboarding_dismiss") => {
    markDismissed(RENEW_EXPAND_GUIDE_DISMISS_KEY);
    trackOnboarding(event, RENEW_EXPAND_GUIDE_ANALYTICS);
    setRenewExpandGuideOpen(false);
  };

  return (
    <div className="page-enter space-y-8 w-full">
      <AdminPageHeader
        title="网盘管理"
        description="为您提供专属、安全的云存储空间，由腾讯云存储 Agent Storage 服务提供支持"
      />

      {/* Stats Cards */}
      <div className="grid grid-cols-1 gap-5 md:grid-cols-3">
        <StatCard
          title="企业公共空间"
          value={stats.enterpriseSpacesCount}
          icon={EnterpriseSpaceIcon}
        />
        <StatCard
          title="已开通智能体网盘"
          value={stats.totalPersonalInstances}
          icon={AgentDiskIcon}
        />
        <StatCard
          title="外网下行流量"
          value={`${EXTERNAL_DOWNLOAD_TRAFFIC.usedGb} GB / ${EXTERNAL_DOWNLOAD_TRAFFIC.quotaGb} GB`}
          icon={TrafficUsageIcon}
          action={
            <div className="relative" data-guide="traffic-package-purchase">
              <Button
                variant="link"
                onClick={() => {
                  dismissTrafficGuide("onboarding_click");
                  setSelectedTrafficPackage(TRAFFIC_PACKAGES[0].id);
                  setTrafficPackageDialogOpen(true);
                }}
              >
                购买流量资源包
              </Button>
              {canShowTrafficGuide ? (
                <div className="absolute right-0 top-full z-[9985] mt-2">
                  <GuidePointBubble
                    open={trafficGuideOpen}
                    onClose={() => dismissTrafficGuide("onboarding_dismiss")}
                    title="外网流量管理上线"
                    description="可查看外网下行流量用量，并按需购买 20 GB 至 1000 GB 资源包。"
                    contentVariant="text-only"
                    placement="bottom"
                    showHotspot={false}
                    endpoint="admin"
                  />
                </div>
              ) : null}
            </div>
          }
        />
      </div>

      {/* Enterprise Public Space Section */}
      <div className="space-y-4">
        <div>
          <h2 className="font-semibold text-[var(--text-title)]">企业公共空间</h2>
          <p className="text-sm text-[var(--text-muted)] mt-1">默认开启,为您赠送 50GB + 50GB 永久免费空间,用于存放 Agent 工具库和初始技能包</p>
        </div>

        <div
          className="bg-white rounded-[4px] border border-[#EAEEF4] overflow-hidden"
        >
          <Table variant="white">
            <TableHeader>
              <TableRow>
                <TableHead className="w-[35%]">空间名称</TableHead>
                <TableHead className="w-[18%]">类型</TableHead>
                <TableHead className="w-[28%]">已用/存储容量</TableHead>
                <TableHead className="w-[19%]">有效期</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {ENTERPRISE_SPACES.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-medium">{item.name}</TableCell>
                  <TableCell>{item.type}</TableCell>
                  <TableCell>
                    <span className="tabular-nums">
                      {item.used}/<span className="font-semibold">{item.quota}</span>
                    </span>
                    <StatusTag mode="fill" variant="green" className="ml-2">免费</StatusTag>
                  </TableCell>
                  <TableCell className="tabular-nums">{item.expiry}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>

      {/* AI Agent Private Space Section */}
      <div className="space-y-4">
        <div>
          <h2 className="font-semibold text-[var(--text-title)]">智能体网盘</h2>
          <p className="text-sm text-[var(--text-muted)] mt-1">开启后,为您赠送每个智能体3个月50GB</p>
        </div>

        {/* 网盘配置卡片 */}
        <div className="grid grid-cols-2 gap-4">
          <FMTogglePolicyCard
            icon={<img src="/assets/admin-disk-management/auto-bind-disk.svg" alt="" className="w-8 h-8" />}
            iconBg=""
            title="新增实例是否自动绑定网盘"
            description="开启后,新创建的 AI 智能体实例将自动分配网盘空间"
            rules={autoBindRules}
            onRulesChange={setAutoBindRules}
          />
        </div>

        {/* 工具栏（独立于表格） */}
        <div className="flex items-center justify-between mb-4 mt-4">
            <div className="flex items-center gap-3">
              <Button
                variant="dialog-confirm"
                onClick={handleBatchEnable}
                disabled={selectedInstances.size === 0}
              >
                批量启用网盘服务{selectedDisabledCount > 0 && `(${selectedDisabledCount})`}
              </Button>
              <Button
                variant="claw-outline"
                size="claw"
                className="gap-2"
                onClick={() => setBatchAutoRenewDialogOpen(true)}
                disabled={selectedEnabledCount === 0}
              >
                <RotateCw className="w-4 h-4" />
                批量开启自动续费{selectedEnabledCount > 0 && `(${selectedEnabledCount})`}
              </Button>
              <Button
                variant="claw-outline"
                size="claw"
                className="gap-2"
                onClick={() => setBatchRenewDialogOpen(true)}
                disabled={selectedEnabledCount === 0}
              >
                <RefreshCw className="w-4 h-4" />
                批量续费{selectedEnabledCount > 0 && `(${selectedEnabledCount})`}
              </Button>
              <Button
                variant="claw-outline"
                size="claw"
                className="gap-2"
                onClick={() => setRecyclebinOpen(true)}
              >
                <Trash2 className="w-4 h-4" />
                回收站{getRecyclebinInstances().length > 0 && `（${getRecyclebinInstances().length}）`}
              </Button>
              <span className="text-[14px] text-[var(--text-muted)]">共计 <span className="font-semibold text-[var(--text-title)] tabular-nums">{stats.totalPersonalInstances}</span> 个智能体开启了网盘服务</span>
            </div>
            <div className="flex items-center gap-3">
              <div className="relative w-64">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" />
                <Input
                  placeholder="搜索名称、ID或创建人"
                  className="pl-9 h-9"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
              <TreeSelect
                nodes={MOCK_GROUP_TREE_MANUAL}
                value={groupFilter}
                onChange={(v) => setGroupFilter(v)}
                allLabel="全部组织"
                searchPlaceholder="搜索组织"
                triggerWidth={140}
                panelWidth={280}
                align="end"
              />
            </div>
        </div>

        <div
          className="bg-white rounded-[4px] border border-[#EAEEF4] overflow-hidden"
        >

          {/* Flat Table
              autoFixedColumns={false}：本表无横向滚动需求（列总宽 ≈ 996px，容器宽度自适应）。
              全局 CSS 会给 [data-auto-fixed="true"] 的 td:first-child 注入 position: sticky + bg-white，
              在 border-collapse 模式下 sticky cell 的背景边界存在 1px 渲染瑕疵，会让相邻第二列内容
              （"OpenClaw 实例" / 实例名 / 实例 ID）从 checkbox 列下"穿透"出来。无横滚场景显式关闭即可。 */}
          <Table variant="white" autoFixedColumns={false}>
            <TableHeader>
              <TableRow>
                <TableHead style={{ width: '56px', minWidth: '56px' }}>
                  <div className="flex items-center">
                    <Checkbox
                      checked={isAllSelected}
                      onCheckedChange={handleSelectAll}
                      disabled={allSelectableInstanceIds.length === 0}
                      className={allSelectableInstanceIds.length === 0 ? "opacity-60 cursor-not-allowed pointer-events-none" : ""}
                      aria-label="全选"
                    />
                  </div>
                </TableHead>
                <TableHead style={{ width: '220px', minWidth: '220px' }}>OpenClaw 实例</TableHead>
                <TableHead style={{ width: '220px', minWidth: '220px' }}>创建人</TableHead>
                <TableHead style={{ minWidth: '80px' }}>类型</TableHead>
                <TableHead style={{ minWidth: '200px' }}>已用/存储容量</TableHead>
                <TableHead style={{ minWidth: '120px' }}>有效期</TableHead>
                <TableHead style={{ minWidth: '100px' }}>自动续费</TableHead>
                <TableHead style={{ minWidth: '100px' }}>启用网盘</TableHead>
                <TableHead style={{ minWidth: '140px' }}>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginatedPersonalSpaces.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={9}>
                    <div className="text-center py-12 space-y-1">
                      <HelperText>未找到匹配的记录</HelperText>
                      <HelperText>请尝试其他搜索关键词</HelperText>
                    </div>
                  </TableCell>
                </TableRow>
              ) : (
                paginatedPersonalSpaces.map((item, itemIndex) => {
                  const isEnabled = instancesEnabled[item.id];
                  const wasEverEnabled = instancesEverEnabled[item.id];
                  const isSelected = selectedInstances.has(item.id);
                  const isFirstEnabledRow = isEnabled && paginatedPersonalSpaces.findIndex(
                    (space) => instancesEnabled[space.id],
                  ) === itemIndex;
                  const isDeleted = wasEverEnabled && isPermanentlyDeleted(item.id);
                  return (
                    <TableRow
                      key={item.id}
                      className={!isDeleted ? 'cursor-pointer' : undefined}
                      onClick={() => {
                        if (!isDeleted) {
                          handleSelectInstance(item.id, !isSelected);
                        }
                      }}
                    >
                      <TableCell style={{ width: '56px', minWidth: '56px' }}>
                        <div className="flex items-center">
                          <Checkbox
                            checked={isSelected}
                            onCheckedChange={(checked) => handleSelectInstance(item.id, checked as boolean)}
                            disabled={isDeleted}
                            className={isDeleted ? "opacity-60 cursor-not-allowed pointer-events-none bg-gray-300 border-gray-500" : ""}
                            aria-label={`选择 ${item.instanceName}`}
                          />
                        </div>
                      </TableCell>
                      <TableCell style={{ width: '220px', minWidth: '220px' }}>
                        <div className="flex flex-col min-w-0">
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <span className="font-medium text-[var(--text-title)] truncate max-w-[180px]">{item.instanceName}</span>
                            </TooltipTrigger>
                            <TooltipContent side="top" className="max-w-xs break-all">{item.instanceName}</TooltipContent>
                          </Tooltip>
                          <span className="font-mono text-[#355EF1]">{item.instanceId}</span>
                        </div>
                      </TableCell>
                      <TableCell style={{ width: '220px', minWidth: '220px' }}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="text-[var(--text-title)] truncate block max-w-[200px]">{item.creator}</span>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-xs break-all">{item.creator}</TooltipContent>
                        </Tooltip>
                      </TableCell>
                      <TableCell>
                        {item.type}
                      </TableCell>
                      <TableCell>
                        {isEnabled ? (
                          <span className="tabular-nums">
                            {item.used}/{<span className="font-semibold">{item.quota}</span>}
                            <span className="ml-2 px-2 py-0.5 rounded-[4px] font-medium bg-emerald-50 text-emerald-600">
                              免费
                            </span>
                          </span>
                        ) : wasEverEnabled && !isPermanentlyDeleted(item.id) ? (
                          <span className="tabular-nums flex items-center gap-1">
                            <span>
                              {item.used}/{<span className="font-semibold">{item.quota}</span>}
                              <span className="ml-2 px-2 py-0.5 rounded-[4px] font-medium bg-[#eff4ff] text-[#355EF1]">
                                可恢复
                              </span>
                            </span>
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Info className="w-3.5 h-3.5 text-[#355EF1] cursor-help shrink-0" />
                                </TooltipTrigger>
                                <TooltipContent>
                                  <p>当前在回收站有资源，剩余 {getRemainingDays(item.id)} 天可恢复</p>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          </span>
                        ) : (
                          <span className="text-[var(--text-weak)]">未启用</span>
                        )}
                      </TableCell>
                      <TableCell className="tabular-nums">
                        {isEnabled ? item.expiry : <span className="text-[var(--text-weak)]">-</span>}
                      </TableCell>
                      <TableCell onClick={(e) => e.stopPropagation()}>
                        {isEnabled ? (
                          <Switch
                            checked={!!instancesAutoRenew[item.id]}
                            onCheckedChange={(checked) => {
                              setInstancesAutoRenew((prev) => ({
                                ...prev,
                                [item.id]: checked,
                              }));
                            }}
                            aria-label={`自动续费 ${item.instanceName}`}
                          />
                        ) : (
                          <HelperText>开启后展示</HelperText>
                        )}
                      </TableCell>
                      <TableCell onClick={(e) => e.stopPropagation()}>
                        <Switch
                          checked={isEnabled}
                          onCheckedChange={() => handleToggleInstance(
                            item.id,
                            item.instanceName,
                            isEnabled,
                            wasEverEnabled,
                            item.requiresPaidEnable,
                          )}
                        />
                      </TableCell>
                      <TableCell onClick={(e) => e.stopPropagation()}>
                        {isEnabled ? (
                          <div
                            ref={isFirstEnabledRow ? renewExpandGuideAnchorRef : undefined}
                            className="flex items-center gap-2"
                            data-guide={isFirstEnabledRow ? "file-renew-expand-actions" : undefined}
                          >
                            <Button
                              variant="claw-outline"
                              size="claw-sm"
                              onClick={() => {
                                dismissRenewExpandGuide("onboarding_click");
                                handleRenew(item.id, item.instanceName);
                              }}
                            >
                              续费
                            </Button>
                            <Button
                              variant="claw-outline"
                              size="claw-sm"
                              onClick={() => {
                                dismissRenewExpandGuide("onboarding_click");
                                handleExpand(item.id, item.instanceName);
                              }}
                            >
                              扩容
                            </Button>
                            {isFirstEnabledRow && canShowRenewExpandGuide ? (
                              <div
                                className="fixed z-[9985]"
                                style={renewExpandGuidePosition}
                              >
                                <GuidePointBubble
                                  open={renewExpandGuideOpen}
                                  onClose={() => dismissRenewExpandGuide("onboarding_dismiss")}
                                  title="网盘续费与扩容"
                                  description="可在个人空间操作列为实例续期或调整存储容量。"
                                  contentVariant="text-only"
                                  placement="top"
                                  showHotspot={false}
                                  endpoint="admin"
                                />
                              </div>
                            ) : null}
                          </div>
                        ) : (
                          <HelperText>开启后展示</HelperText>
                        )}
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>

          {/* Pagination */}
          {/*
            停服态豁免（页面级，不动组件库）：
            本页使用 shadcn 版<Pagination>（@/components/ui/pagination），
            为其外层已有的 <div> 补 data-billing-exempt，命中
            AdminDisabledOverlay 的视觉/事件两条恢复分支。
            "停服前已禁用则延续禁用"：<Pagination> 内部对首页的上一页 /
            末页的下一页原生 disabled 依旧生效（CSS 恢复分支带
            :not([disabled]):not([aria-disabled="true"]) 保护）。
          */}
          <div className="px-4 py-2 border-t border-[#EAEEF4]" data-billing-exempt>
            <Pagination
              total={filteredPersonalSpaces.length}
              current={currentPage}
              pageSize={itemsPerPage}
              showTotal={(total) => `共 ${total} 条记录`}
              className="w-full justify-between"
              hideOnSinglePage
              onChange={(page) => { setCurrentPage(page); }}
            />
          </div>
        </div>
      </div>

      {/* Disable Confirmation Dialog */}
      <Dialog open={disableDialogOpen} onOpenChange={setDisableDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-title)]">
              确认关闭网盘
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <p className="text-sm text-[var(--text-secondary)]">
              您确定要关闭 <span className="font-bold text-[var(--text-title)]">"{instanceToDisable?.name}"</span> 的网盘功能吗？
            </p>
            <div className="p-3 bg-[#fafafa] border border-[#EAEEF4] rounded-[4px]">
              <div className="text-xs text-[var(--text-secondary)] space-y-1">
                <p className="font-semibold">关闭网盘后：</p>
                <div className="space-y-0.5 ml-1">
                  <p>• 该实例将无法访问网盘中的文件</p>
                  <p>• 15天内网盘数据可恢复</p>
                </div>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={handleCancelDisable}>取消</Button>
            <Button onClick={handleConfirmDisable} variant="destructive">
              确认关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Batch Enable Confirmation Dialog */}
      <Dialog open={batchEnableDialogOpen} onOpenChange={setBatchEnableDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-title)]">
              批量启用网盘服务
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <p className="text-sm text-[var(--text-secondary)]">
              您确定要为选中的 <span className="font-semibold text-[var(--text-title)] tabular-nums">{selectedInstances.size}</span> 个实例启用网盘服务吗?
            </p>
            <div className="bg-[#fafafa] border border-[#EAEEF4] rounded-[4px] px-3 py-2.5">
              <div className="text-xs text-[var(--text-secondary)] space-y-1 leading-relaxed">
                <p className="font-semibold">启用后：</p>
                <ul className="list-disc list-inside space-y-0.5 ml-1">
                  <li>每个实例将获得 3个月50GB 免费额度</li>
                  <li>实例可以访问专属网盘空间</li>
                  <li>到期后可购买资源包续租</li>
                </ul>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={handleCancelBatchEnable}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleConfirmBatchEnable}>
              确认启用
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Batch Auto Renew Confirmation Dialog */}
      <Dialog open={batchAutoRenewDialogOpen} onOpenChange={setBatchAutoRenewDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-title)]">
              批量开启自动续费
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <p className="text-sm text-[var(--text-secondary)]">
              需要为以下 <span className="font-semibold text-[var(--text-title)] tabular-nums">{selectedEnabledCount}</span> 个网盘自动开启自动续费吗？
            </p>
            <div className="bg-[#fafafa] border border-[#EAEEF4] rounded-[4px] px-3 py-2.5">
              <div className="text-xs text-[var(--text-secondary)] space-y-1 leading-relaxed">
                <ul className="list-disc list-inside space-y-0.5 ml-1">
                  <li>仅作用于当前已开通网盘的实例</li>
                  <li>到期前自动从企业账户扣费续期</li>
                </ul>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBatchAutoRenewDialogOpen(false)}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleConfirmBatchAutoRenew}>
              确认开启
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Batch Renew Confirmation Dialog */}
      <Dialog open={batchRenewDialogOpen} onOpenChange={setBatchRenewDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-title)]">
              批量续费
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <p className="text-sm text-[var(--text-secondary)]">
              将为选中的 <span className="font-semibold text-[var(--text-title)] tabular-nums">{selectedEnabledCount}</span> 个已开通网盘的智能体发起续费，确认后将跳转到腾讯云官方付款页面。
            </p>
            <div className="bg-[#fafafa] border border-[#EAEEF4] rounded-[4px] px-3 py-2.5">
              <div className="text-xs text-[var(--text-secondary)] space-y-1 leading-relaxed">
                <ul className="list-disc list-inside space-y-0.5 ml-1">
                  <li>仅作用于当前已启用网盘的实例</li>
                  <li>付款完成前不会变更网盘状态</li>
                </ul>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setBatchRenewDialogOpen(false)}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleConfirmBatchRenew}>
              确认付款
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Single Enable Confirmation Dialog */}
      <Dialog open={singleEnableDialogOpen} onOpenChange={setSingleEnableDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-title)]">
              启用网盘服务
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <p className="text-sm text-[var(--text-secondary)]">
              您确定要为 <span className="font-bold text-[var(--text-title)]">"{instanceToEnable?.name}"</span> 启用网盘服务吗?
            </p>
            <div className="bg-[#fafafa] border border-[#EAEEF4] rounded-[4px] px-3 py-2.5">
              <div className="text-xs text-[var(--text-secondary)] space-y-1 leading-relaxed">
                <p className="font-semibold">启用后：</p>
                <ul className="list-disc list-inside space-y-0.5 ml-1">
                  <li>该实例将获得 3个月50GB 免费额度</li>
                  <li>实例可以访问专属网盘空间</li>
                  <li>到期后可以进行续租</li>
                </ul>
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={handleCancelSingleEnable}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleConfirmSingleEnable}>
              确认启用
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Purchase Storage Dialog */}
      <Dialog open={purchaseDialogOpen} onOpenChange={setPurchaseDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-title)] flex items-center gap-2">
              <ShoppingCart className="w-5 h-5 text-purple-600" />
              付费开启网盘
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="bg-[#eff4ff] border border-[#355EF1] rounded-[4px] px-3 py-2.5">
              <p className="text-xs text-[#355EF1] leading-relaxed">
                为 <span className="font-semibold">"{instanceToPurchase?.name}"</span> 购买网盘容量和使用时长
              </p>
            </div>

            {/* 选择存储容量 */}
            <div className="space-y-3">
              <Label className="text-sm font-semibold text-[var(--text-title)]">选择存储容量</Label>
              <RadioGroup value={selectedCapacity} onValueChange={setSelectedCapacity}>
                <div className="grid grid-cols-3 gap-3">
                  {[
                    { value: "50GB", label: "50GB", price: "¥2/月" },
                    { value: "100GB", label: "100GB", price: "¥4/月" },
                    { value: "500GB", label: "500GB", price: "¥8/月" }
                  ].map((item) => (
                    <div key={item.value} className="flex items-center">
                      <RadioGroupItem value={item.value} id={item.value} className="peer sr-only" />
                      <Label
                        htmlFor={item.value}
                        className="flex flex-1 flex-col items-center justify-center rounded-[4px] border-2 border-[#EAEEF4] bg-white p-3 hover:bg-[#fafafa] cursor-pointer peer-data-[state=checked]:border-purple-600 peer-data-[state=checked]:bg-purple-50 transition-all"
                      >
                        <span className="text-sm font-semibold text-[var(--text-title)]">{item.label}</span>
                        <span className="text-xs text-[var(--text-muted)] mt-1">{item.price}</span>
                      </Label>
                    </div>
                  ))}
                </div>
              </RadioGroup>
            </div>

            {/* 选择购买时长 */}
            <div className="space-y-3">
              <Label className="text-sm font-semibold text-[var(--text-title)]">选择购买时长</Label>
              <RadioGroup value={selectedDuration} onValueChange={setSelectedDuration}>
                <div className="space-y-2">
                  {[
                    { value: "1", label: "1个月" },
                    { value: "3", label: "3个月" },
                    { value: "6", label: "6个月" },
                    { value: "12", label: "12个月" }
                  ].map((item) => (
                    <div key={item.value} className="flex items-center">
                      <RadioGroupItem value={item.value} id={`duration-${item.value}`} className="peer sr-only" />
                      <Label
                        htmlFor={`duration-${item.value}`}
                        className="flex flex-1 items-center justify-between rounded-[4px] border-2 border-[#EAEEF4] bg-white p-3 hover:bg-[#fafafa] cursor-pointer peer-data-[state=checked]:border-purple-600 peer-data-[state=checked]:bg-purple-50 transition-all"
                      >
                        <span className="text-sm font-medium text-[var(--text-title)]">{item.label}</span>
                      </Label>
                    </div>
                  ))}
                </div>
              </RadioGroup>
            </div>

            {/* 价格汇总 */}
            <div className="bg-gradient-to-br from-purple-50 to-blue-50 border border-purple-100 rounded-[4px] p-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-[var(--text-secondary)]">合计金额：</span>
                <div className="flex items-baseline gap-1">
                  <span className="text-2xl font-bold text-purple-600 tabular-nums">¥{calculatePrice()}</span>
                </div>
              </div>
              <p className="text-xs text-[var(--text-muted)] mt-2">
                购买后立即生效，有效期 {selectedDuration} 个月
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={handleCancelPurchase}>取消</Button>
            <Button
              variant="dialog-confirm"
              onClick={handleConfirmPurchase}
              className="gap-2"
            >
              <ShoppingCart className="w-4 h-4" />
              确认付款并开启
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Traffic Package Purchase Dialog */}
      <Dialog open={trafficPackageDialogOpen} onOpenChange={setTrafficPackageDialogOpen}>
        <DialogContent size="md" className="flex max-h-[80vh] flex-col">
          <DialogHeader>
            <DialogTitle className="text-[var(--cp-text-title)]">购买流量资源包</DialogTitle>
            <DialogDescription className="text-[var(--cp-text-muted)]">
              选择外网下行流量规格，购买后流量将叠加至当前可用额度。
            </DialogDescription>
          </DialogHeader>

          <DialogBody className="px-6">
            <RadioGroup
              value={selectedTrafficPackage}
              onValueChange={setSelectedTrafficPackage}
              aria-label="流量资源包规格"
              className="py-1"
            >
              {TRAFFIC_PACKAGES.map((item) => {
                const isSelected = selectedTrafficPackage === item.id;
                return (
                  <Label
                    key={item.id}
                    htmlFor={item.id}
                    className={`flex cursor-pointer items-start gap-3 rounded-[4px] border p-4 transition-colors ${
                      isSelected
                        ? "border-[#1447E6] bg-[#F5F8FF]"
                        : "border-[var(--cp-border-control)] bg-[var(--cp-surface)] hover:border-[#1447E6]"
                    }`}
                  >
                    <RadioGroupItem id={item.id} value={item.id} className="mt-0.5" />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm font-medium text-[var(--cp-text-title)] tabular-nums">
                        {item.capacity}
                      </span>
                      <span className="mt-1 block text-xs leading-5 text-[var(--cp-text-weak)]">
                        {item.description}
                      </span>
                    </span>
                  </Label>
                );
              })}
            </RadioGroup>
            <p className="mt-3 text-xs text-[var(--cp-text-weak)]">实际价格以订单结算结果为准。</p>
          </DialogBody>

          <DialogFooter>
            <Button variant="outline" onClick={() => setTrafficPackageDialogOpen(false)}>
              取消
            </Button>
            <Button
              variant="dialog-confirm"
              onClick={() => {
                if (!openTencentCloudPaymentPage()) return;
                setTrafficPackageDialogOpen(false);
              }}
            >
              确认购买
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Renew Storage Dialog */}
      <Dialog open={renewDialogOpen} onOpenChange={setRenewDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-title)] flex items-center gap-2">
              <ShoppingCart className="w-5 h-5 text-[#355EF1]" />
              续费网盘
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="bg-[#eff4ff] border border-[#355EF1] rounded-[4px] px-3 py-2.5">
              <p className="text-xs text-[#355EF1] leading-relaxed">
                为 <span className="font-semibold">"{instanceToRenew?.name}"</span>
                {renewingFromRecyclebin ? " 续费并恢复网盘服务" : " 续费网盘服务"}
              </p>
            </div>

            <div className="bg-[#fafafa] border border-[#EAEEF4] rounded-[4px] px-3 py-2.5">
              <div className="text-xs text-[var(--text-secondary)] space-y-1">
                <p className="font-semibold">当前配置：</p>
                <p>• 存储容量：50GB</p>
                <p>• 到期时间：2026-06-30</p>
              </div>
            </div>

            {/* 选择续费时长 */}
            <div className="space-y-3">
              <Label className="text-sm font-semibold text-[var(--text-title)]">选择续费时长</Label>
              <RadioGroup value={renewDuration} onValueChange={setRenewDuration}>
                <div className="space-y-2">
                  {[
                    { value: "1", label: "1个月" },
                    { value: "3", label: "3个月" },
                    { value: "6", label: "6个月" },
                    { value: "12", label: "12个月" }
                  ].map((item) => (
                    <div key={item.value} className="flex items-center">
                      <RadioGroupItem value={item.value} id={`renew-duration-${item.value}`} className="peer sr-only" />
                      <Label
                        htmlFor={`renew-duration-${item.value}`}
                        className="flex flex-1 items-center justify-between rounded-[4px] border-2 border-[#EAEEF4] bg-white p-3 hover:bg-[#fafafa] cursor-pointer peer-data-[state=checked]:border-blue-600 peer-data-[state=checked]:bg-[#eff4ff] transition-all"
                      >
                        <span className="text-sm font-medium text-[var(--text-title)]">{item.label}</span>
                      </Label>
                    </div>
                  ))}
                </div>
              </RadioGroup>
            </div>

            {/* 价格汇总 */}
            <div className="bg-gradient-to-br from-blue-50 to-cyan-50 border border-[#355EF1] rounded-[4px] p-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-[var(--text-secondary)]">续费金额：</span>
                <div className="flex items-baseline gap-1">
                  <span className="text-2xl font-bold text-[#355EF1] tabular-nums">¥{calculateRenewPrice()}</span>
                </div>
              </div>
              <p className="text-xs text-[var(--text-muted)] mt-2">
                续费后有效期延长 {renewDuration} 个月
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={handleCancelRenew}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleConfirmRenew}>
              {renewingFromRecyclebin ? "确认付款并恢复" : "确认续费"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Expand Storage Dialog */}
      <Dialog open={expandDialogOpen} onOpenChange={setExpandDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="text-[var(--text-title)] flex items-center gap-2">
              <ShoppingCart className="w-5 h-5 text-purple-600" />
              扩容网盘
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="bg-purple-50 border border-purple-100 rounded-[4px] px-3 py-2.5">
              <p className="text-xs text-purple-600 leading-relaxed">
                为 <span className="font-semibold">"{instanceToExpand?.name}"</span> 扩容网盘空间
              </p>
            </div>

            <div className="bg-[#fafafa] border border-[#EAEEF4] rounded-[4px] px-3 py-2.5">
              <div className="text-xs text-[var(--text-secondary)] space-y-1">
                <p className="font-semibold">当前配置：</p>
                <p>• 存储容量：50GB</p>
                <p>• 到期时间：2026-06-30</p>
              </div>
            </div>

            {/* 选择扩容容量 */}
            <div className="space-y-3">
              <Label className="text-sm font-semibold text-[var(--text-title)]">选择扩容容量</Label>
              <RadioGroup value={expandCapacity} onValueChange={setExpandCapacity}>
                <div className="grid grid-cols-3 gap-3 max-h-[240px] overflow-y-auto pr-2">
                  {expandCapacityOptions.map((item) => (
                    <div key={item.value} className="flex items-center">
                      <RadioGroupItem value={item.value} id={`expand-${item.value}`} className="peer sr-only" />
                      <Label
                        htmlFor={`expand-${item.value}`}
                        className="flex flex-1 flex-col items-center justify-center rounded-[4px] border-2 border-[#EAEEF4] bg-white p-3 hover:bg-[#fafafa] cursor-pointer peer-data-[state=checked]:border-purple-600 peer-data-[state=checked]:bg-purple-50 transition-all"
                      >
                        <span className="text-sm font-semibold text-[var(--text-title)]">{item.label}</span>
                        <span className="text-xs text-[var(--text-muted)] mt-1">¥{item.price}</span>
                      </Label>
                    </div>
                  ))}
                </div>
              </RadioGroup>
            </div>

            {/* 价格汇总 */}
            <div className="bg-gradient-to-br from-purple-50 to-pink-50 border border-purple-100 rounded-[4px] p-4">
              <div className="flex items-center justify-between">
                <span className="text-sm text-[var(--text-secondary)]">扩容费用：</span>
                <div className="flex items-baseline gap-1">
                  <span className="text-2xl font-bold text-purple-600 tabular-nums">¥{calculateExpandPrice()}</span>
                </div>
              </div>
              <p className="text-xs text-[var(--text-muted)] mt-2">
                扩容 {expandCapacity}，立即生效，不延长有效期
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={handleCancelExpand}>取消</Button>
            <Button variant="dialog-confirm" onClick={handleConfirmExpand}>
              确认扩容
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Recycle Bin Dialog */}
      <Dialog open={recyclebinOpen} onOpenChange={setRecyclebinOpen}>
        <DialogContent size="xl" className="flex max-h-[80vh] flex-col">
          <DialogHeader>
            <DialogTitle className="text-[var(--cp-text-title)]">回收站</DialogTitle>
            <DialogDescription className="text-[var(--cp-text-muted)]">
              {getRecyclebinInstances().length > 0 ? (
                <>共 <span className="font-medium tabular-nums text-[var(--cp-text-title)]">{getRecyclebinInstances().length}</span> 个网盘空间待处理 · 关闭后保留 15 天，逾期自动永久删除</>
              ) : (
                <>关闭后的网盘空间将在此保留 15 天，逾期自动永久删除</>
              )}
            </DialogDescription>
          </DialogHeader>

          <DialogBody className="min-h-0 flex-1 overflow-y-auto px-6 py-2" style={{ scrollbarGutter: "stable" }}>
            {getRecyclebinInstances().length === 0 ? (
              <div className="space-y-1 py-12 text-center">
                <p className="text-sm text-[var(--cp-text-body)]">回收站为空</p>
                <p className="text-xs text-[var(--cp-text-weak)]">没有待恢复的网盘空间</p>
              </div>
            ) : (
              <div className="space-y-3 pb-4">
                {getRecyclebinInstances().map((instance) => {
                  const days = instance.remainingDays;
                  // 紧迫度色阶：≤3天 红色 / ≤7天 橙色 / >7天 中性灰
                  const urgencyVariant: "red" | "orange" | "gray" =
                    days <= 3 ? "red" : days <= 7 ? "orange" : "gray";

                  return (
                    <div
                      key={instance.id}
                      className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-5 transition-colors hover:border-[var(--cp-brand-blue)]/30"
                    >
                      <div className="flex flex-col gap-5">
                        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-start sm:gap-6">
                          <div className="flex min-w-0 items-start gap-3.5">
                            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-[#1447E6] to-[#3066F2] text-sm font-semibold text-white">
                              {instance.avatar}
                            </div>
                            <div className="min-w-0 pt-0.5">
                              <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
                                <h4 className="truncate text-sm font-medium leading-5 text-[var(--cp-text-title)]">
                                  {instance.instanceName}
                                </h4>
                                <span className="truncate font-mono text-[11px] text-[var(--cp-text-muted)]">
                                  实例 ID：{instance.instanceId}
                                </span>
                              </div>
                              <div className="mt-2 flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-[var(--cp-text-muted)]">
                                <span className="truncate">创建人：{instance.creator}</span>
                                <span className="whitespace-nowrap tabular-nums">容量：{instance.used} / {instance.quota}</span>
                              </div>
                            </div>
                          </div>

                          <StatusTag
                            mode="fill"
                            variant={urgencyVariant}
                            className="h-7 min-w-[136px] shrink-0 self-start px-3 font-medium tabular-nums"
                          >
                            {days === 0 ? "还剩 0 天永久删除" : `还剩 ${days} 天永久删除`}
                          </StatusTag>
                        </div>

                        <div className="flex items-center justify-end border-t border-[var(--cp-border)] pt-3 -mr-2">
                          <Button
                            variant="link"
                            className="h-8 gap-1.5 px-2"
                            onClick={() => handleRestoreFromRecyclebin(instance.id, instance.instanceName)}
                          >
                            <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
                            恢复
                          </Button>
                          <Separator orientation="vertical" className="mx-1 !h-4" />
                          <Button
                            variant="link"
                            className="h-8 gap-1.5 px-2 text-[var(--cp-text-muted)] hover:text-[var(--cp-text-danger)]"
                            onClick={() => {
                              setInstanceToDeletePermanently({ id: instance.id, name: instance.instanceName });
                              setRecyclebinDeleteDialogOpen(true);
                            }}
                          >
                            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                            永久删除
                          </Button>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </DialogBody>
        </DialogContent>
      </Dialog>

      {/* Recyclebin Permanent Delete Confirmation - 警示弹窗 */}
      <AlertDialog open={recyclebinDeleteDialogOpen} onOpenChange={setRecyclebinDeleteDialogOpen}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-[var(--text-title)]">永久删除网盘空间</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <div className="space-y-4">
              <Alert variant="warning">
                <CircleAlert />
                <AlertTitle>注意事项</AlertTitle>
                <AlertDescription>
                  <ul className="space-y-0.5">
                    <li>• 网盘中所有文件和数据将被永久删除</li>
                    <li>• 删除后无法恢复任何内容</li>
                    <li>• 请谨慎操作</li>
                  </ul>
                </AlertDescription>
              </Alert>
              <p className="text-sm text-[var(--text-title)]">
                确定要永久删除 <span className="font-medium text-[var(--text-title)]">"{instanceToDeletePermanently?.name}"</span> 的网盘空间吗？
              </p>
            </div>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={handleCancelPermanentDelete}>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmPermanentDelete}>永久删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

    </div>
  );
}
