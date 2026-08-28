/**
 * CloudDevManagement - 管控端云开发管理页
 * Design: 「流动蓝图」Fluid Blueprint - Admin Side
 */
import React, { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogBody } from "@/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell } from "@/components/ui/table";
import { StatusTag } from "@/components/ui/status-tag";
import { GroupSelect } from "@/components/GroupSelect";
import { AdminPageHeader } from "@/components/ui/admin-page-header";
import { NumberCard } from "@/components/ui/number-card";
import type { PolicyRule } from "@/components/policy/types";
import { MOCK_GROUPS } from "./MemberManagement/mock";
import { type GroupNode } from "@/lib/mockData";
import { TreeSelect } from "@/components/ui/tree-select";
import { buildGroupTree, type GroupTreeNode } from "./MemberManagement/health";
import { toast } from "sonner";
import { Search, RefreshCw, ChevronLeft, ChevronRight, Copy, Plus, UserCircle, X, Sparkles } from "lucide-react";
import { ScopeUserOrGroupPopover } from "@/components/ScopeUserOrGroupPopover";
import CloudDevEnvDetailDialog from "@/components/admin/CloudDevEnvDetailDialog";
import CloudDevCreateEnvDialog, { type CreateEnvForm } from "@/components/admin/CloudDevCreateEnvDialog";
import CloudDevClaimTrialDialog, { MAX_TRIAL_CLAIM } from "@/components/admin/CloudDevClaimTrialDialog";

type EnvStatus = "running" | "stopped" | "creating" | "error";

interface Env {
  id: string;
  envId: string;
  name: string;
  status: EnvStatus;
  region: string;
  packageName: string;
  storageUsed: string;
  dbUsed: string;
  functionCount: number;
  staticHosting: boolean;
  createdAt: string;
  expireAt: string;
  lastDeployAt: string;
  appCount: number;
  appNames: string[];
  createdBy: string; // 创建用户
  allowedUsers: string[]; // 可使用的用户 ID 列表
  allowedGroups: string[]; // 可使用的组织 ID 列表
  dbType: "cloud" | "postgresql"; // 数据库类型
  overflowBilling: boolean; // 超限按量计费
  autoRenewal: boolean; // 自动续费
  claimedFromTrial?: boolean; // 是否来自「免费体验领取」入口（用于统计领取上限）
}

/* ═══════════════════════════════════════════════════════════════════════════
 * 本地策略卡组件（参考 client/src/pages/admin/FileManagement.tsx
 * 的 PolicyOverviewCard + FMTogglePolicyCard 实现，扩展 quota 数值型版本）
 *
 * 提供两种用法：
 *   - <CDTogglePolicyCard>  布尔开关型（同网盘的 FMTogglePolicyCard）
 *   - <CDQuotaPolicyCard>   数值配额型（integer，0-999）
 *
 * 视觉骨架完全对齐网盘管理 §智能体网盘 的配置卡：
 *   Card 概览（icon + title + desc + 灰底摘要条）→ 点击进 Dialog 行内编辑表格
 * ═══════════════════════════════════════════════════════════════════════════ */

/* eslint-disable @typescript-eslint/no-unused-vars */

interface CDPolicyOverviewCardProps {
  icon: React.ReactNode;
  iconBg?: string;
  title: string;
  description: string;
  /** 摘要条上的预设策略展示节点（toggle: StatusTag 开/关；quota: "N 个"） */
  fallbackSummary: React.ReactNode;
  groupCount: number;
  onClick?: () => void;
}

function CDPolicyOverviewCard({ icon, iconBg, title, description, fallbackSummary, groupCount, onClick }: CDPolicyOverviewCardProps) {
  return (
    <Card
      className={`overflow-hidden py-0 gap-0 flex flex-col transition-colors ${onClick ? "cursor-pointer hover:border-[var(--cp-brand-blue)]" : ""}`}
      onClick={onClick}
    >
      <div className="px-5 pt-5 pb-4 flex-1 min-h-0 flex flex-col">
        <div className="flex items-start gap-3">
          <div className={`shrink-0 ${iconBg ? `w-8 h-8 rounded-[var(--radius-lg)] flex items-center justify-center ${iconBg}` : ""}`}>{icon}</div>
          <div className="min-w-0 flex-1">
            <h3 className="text-[14px] font-medium text-[var(--text-emphasis)] truncate">{title}</h3>
            <p className="text-[12px] text-[var(--text-muted)] leading-relaxed mt-1 line-clamp-2">{description}</p>
          </div>
        </div>

        {/* 底部灰色摘要条 */}
        <div className="mt-4 rounded-[var(--radius-lg)] bg-[var(--bg-grey-normal)] px-3 py-2 flex items-center justify-between">
          <div className="flex items-center gap-4 text-[12px]">
            <span className="text-[var(--text-muted)] inline-flex items-center gap-1">预设策略：{fallbackSummary}</span>
            <span className="text-[var(--text-muted)]">分组策略：<span className="text-[var(--text-emphasis)] font-medium">{groupCount} 条</span></span>
          </div>
          <span className="text-[12px] text-[var(--cp-brand-blue)] inline-flex items-center gap-0.5">
            编辑策略<ChevronRight className="w-3.5 h-3.5" />
          </span>
        </div>
      </div>
    </Card>
  );
}

/* ───────── 分组名称展示（弹窗内分组规则行使用） ───────── */
function CDGroupBadges({ groupIds }: { groupIds: string[] }) {
  if (groupIds.length === 0) return <span className="text-xs text-[var(--text-muted)] font-medium">预设策略</span>;
  return (
    <div className="flex items-center gap-1 flex-wrap">
      {groupIds.map((gid) => {
        const name = MOCK_GROUPS.find((g) => g.id === gid)?.name ?? gid;
        return <Badge key={gid} variant="secondary" className="max-w-[140px]"><span className="block truncate max-w-[124px]">{name}</span></Badge>;
      })}
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════════════════════
 * CDTogglePolicyCard - 布尔开关型策略卡（同 FileManagement.FMTogglePolicyCard）
 * ═══════════════════════════════════════════════════════════════════════════ */
interface CDTogglePolicyCardProps {
  icon: React.ReactNode;
  iconBg?: string;
  title: string;
  description: string;
  rules: PolicyRule<boolean>[];
  onRulesChange: (rules: PolicyRule<boolean>[]) => void;
}

function CDTogglePolicyCard({ icon, iconBg, title, description, rules, onRulesChange }: CDTogglePolicyCardProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftGroupIds, setDraftGroupIds] = useState<string[]>([]);
  const [draftValue, setDraftValue] = useState<boolean>(true);
  const [addingNew, setAddingNew] = useState(false);
  const [confirmFallbackDraft, setConfirmFallbackDraft] = useState<boolean | null>(null);

  const fallbackRule = rules.find((r) => r.groupIds.length === 0)!;
  const groupRules = rules.filter((r) => r.groupIds.length > 0);
  const groupRuleValue = !fallbackRule.value;

  const getDisabledIds = (excludeRuleId?: string) =>
    rules.filter((r) => r.groupIds.length > 0 && r.id !== excludeRuleId).flatMap((r) => r.groupIds);

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
      if (draftGroupIds.length === 0) { toast.error("请选择至少一个分组"); return; }
      onRulesChange([...groupRules, { id: `rule-${Date.now()}`, groupIds: draftGroupIds, value: groupRuleValue }, fallbackRule]);
      toast.success("策略已保存"); cancelEdit(); return;
    }
    if (!ruleId) return;
    if (ruleId === fallbackRule.id) {
      if (draftValue !== fallbackRule.value && groupRules.length > 0) { setConfirmFallbackDraft(draftValue); return; }
      onRulesChange(rules.map((r) => r.id === ruleId ? { ...r, value: draftValue } : r));
      toast.success("策略已保存"); cancelEdit(); return;
    }
    onRulesChange(rules.map((r) => r.id === ruleId ? { ...r, groupIds: draftGroupIds, value: groupRuleValue } : r));
    toast.success("策略已保存"); cancelEdit();
  };

  const handleConfirmFallbackSwitch = () => {
    if (confirmFallbackDraft === null) return;
    onRulesChange([{ ...fallbackRule, value: confirmFallbackDraft }]);
    toast.success("已更新预设策略，分组策略已清空");
    cancelEdit();
    setConfirmFallbackDraft(null);
  };

  const deleteRule = (ruleId: string) => {
    onRulesChange(rules.filter((r) => r.id !== ruleId));
    toast.success("策略已删除");
  };

  return (
    <>
      <CDPolicyOverviewCard
        icon={icon}
        iconBg={iconBg}
        title={title}
        description={description}
        fallbackSummary={<StatusTag mode="fill" variant={fallbackRule.value ? "green" : "gray"}>{fallbackRule.value ? "开启" : "关闭"}</StatusTag>}
        groupCount={groupRules.length}
        onClick={() => setDialogOpen(true)}
      />

      <Dialog open={dialogOpen} onOpenChange={(v) => { setDialogOpen(v); if (!v) cancelEdit(); }}>
        <DialogContent className="sm:max-w-[960px]">
          <DialogHeader><DialogTitle>{title}</DialogTitle></DialogHeader>
          <DialogBody className="px-6">
            <div className="space-y-3">
              <div className="flex items-center gap-4 text-[13px]">
                <span className="text-[var(--text-muted)] inline-flex items-center gap-1">预设策略：<StatusTag mode="fill" variant={fallbackRule.value ? "green" : "gray"}>{fallbackRule.value ? "开启" : "关闭"}</StatusTag></span>
                <span className="text-[var(--text-muted)]">分组策略：<span className="text-[var(--text-emphasis)] font-medium">{groupRules.length} 个</span></span>
              </div>

              <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] overflow-hidden">
                <Table density="compact" variant="gray-header">
                  <colgroup><col style={{ width: 90 }} /><col /><col style={{ width: 100 }} /><col style={{ width: 100 }} /></colgroup>
                  <TableHeader>
                    <TableRow><TableHead>策略类型</TableHead><TableHead>应用范围</TableHead><TableHead>权限</TableHead><TableHead>操作</TableHead></TableRow>
                  </TableHeader>
                  <TableBody>
                    {/* 预设策略行 */}
                    <TableRow>
                      <TableCell className="text-[var(--text-muted)]">预设策略</TableCell>
                      <TableCell>{groupRules.length > 0 ? "全部用户（分组策略用户除外）" : "全部用户"}</TableCell>
                      <TableCell>
                        {editingId === fallbackRule.id ? (
                          <Select value={draftValue ? "on" : "off"} onValueChange={(v) => setDraftValue(v === "on")}>
                            <SelectTrigger className="h-7 w-[80px] text-xs"><SelectValue /></SelectTrigger>
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
                        ) : (
                          <Button variant="link" onClick={() => startEdit(fallbackRule)}>编辑</Button>
                        )}
                      </TableActionCell>
                    </TableRow>

                    {/* 分组策略行 */}
                    {groupRules.map((rule, idx) => (
                      <TableRow key={rule.id}>
                        <TableCell className="text-[var(--text-muted)]">分组策略{idx + 1}</TableCell>
                        <TableCell>
                          {editingId === rule.id ? (
                            <GroupSelect
                              groups={MOCK_GROUPS}
                              selectedIds={draftGroupIds}
                              disabledIds={getDisabledIds(rule.id)}
                              onChange={setDraftGroupIds}
                              disabledTooltip="该分组已被其他策略使用"
                              placeholder="选择分组"
                              variant="confirm"
                              onSave={() => saveEdit(rule.id)}
                            />
                          ) : (
                            <CDGroupBadges groupIds={rule.groupIds} />
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
                        <TableCell className="text-[var(--text-muted)]">分组策略{groupRules.length + 1}</TableCell>
                        <TableCell>
                          <GroupSelect
                            groups={MOCK_GROUPS}
                            selectedIds={draftGroupIds}
                            disabledIds={getDisabledIds()}
                            onChange={setDraftGroupIds}
                            disabledTooltip="该分组已被其他策略使用"
                            placeholder="选择分组"
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
                <button
                  type="button"
                  onClick={startAdd}
                  disabled={editingId !== null || addingNew}
                  className="w-full flex items-center justify-center gap-1 px-3 py-2 text-[13px] text-[var(--text-emphasis)] bg-white border-t border-dashed border-[var(--cp-border)] hover:bg-[var(--bg-grey-normal)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Plus className="w-3.5 h-3.5" />添加组织策略
                </button>
              </div>
            </div>
          </DialogBody>
          <DialogFooter />
        </DialogContent>
      </Dialog>

      <AlertDialog open={confirmFallbackDraft !== null} onOpenChange={(o) => { if (!o) setConfirmFallbackDraft(null); }}>
        <AlertDialogContent>
          <AlertDialogHeader><AlertDialogTitle>切换后将清空分组策略</AlertDialogTitle></AlertDialogHeader>
          <AlertDialogDescription>分组策略是基于「预设策略」的例外设置。切换「预设策略」后，现有分组策略将全部清空，需重新添加。</AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmFallbackSwitch}>确认切换</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

/* ═══════════════════════════════════════════════════════════════════════════
 * CDQuotaPolicyCard - 数值配额型策略卡（integer，0-999）
 * 同 toggle 卡片骨架，区别：值列用 Input number 编辑，展示用 "N 个"
 * ═══════════════════════════════════════════════════════════════════════════ */
interface CDQuotaPolicyCardProps {
  icon: React.ReactNode;
  iconBg?: string;
  title: string;
  description: string;
  /** 单位后缀，如 "个" / "次" */
  unit?: string;
  rules: PolicyRule<number>[];
  onRulesChange: (rules: PolicyRule<number>[]) => void;
}

function CDQuotaPolicyCard({ icon, iconBg, title, description, unit = "个", rules, onRulesChange }: CDQuotaPolicyCardProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [draftGroupIds, setDraftGroupIds] = useState<string[]>([]);
  const [draftValueStr, setDraftValueStr] = useState<string>("");
  const [addingNew, setAddingNew] = useState(false);

  const fallbackRule = rules.find((r) => r.groupIds.length === 0)!;
  const groupRules = rules.filter((r) => r.groupIds.length > 0);

  const formatValue = (v: number) => `${v.toLocaleString()} ${unit}`;

  const getDisabledIds = (excludeRuleId?: string) =>
    rules.filter((r) => r.groupIds.length > 0 && r.id !== excludeRuleId).flatMap((r) => r.groupIds);

  const startEdit = (rule: PolicyRule<number>) => {
    setEditingId(rule.id);
    setDraftGroupIds([...rule.groupIds]);
    setDraftValueStr(String(rule.value));
    setAddingNew(false);
  };
  const startAdd = () => { setAddingNew(true); setEditingId(null); setDraftGroupIds([]); setDraftValueStr("3"); };
  const cancelEdit = () => { setEditingId(null); setAddingNew(false); };

  const parseAndValidate = (): number | null => {
    const n = parseInt(draftValueStr, 10);
    if (isNaN(n) || n < 0) { toast.error("请输入有效的非负整数"); return null; }
    if (n > 999) { toast.error("请输入 0-999 之间的整数"); return null; }
    return n;
  };

  const saveEdit = (ruleId?: string) => {
    const value = parseAndValidate();
    if (value === null) return;
    if (addingNew) {
      if (draftGroupIds.length === 0) { toast.error("请选择至少一个分组"); return; }
      onRulesChange([...groupRules, { id: `rule-${Date.now()}`, groupIds: draftGroupIds, value }, fallbackRule]);
      toast.success("策略已保存"); cancelEdit(); return;
    }
    if (!ruleId) return;
    if (ruleId === fallbackRule.id) {
      onRulesChange(rules.map((r) => r.id === ruleId ? { ...r, value } : r));
      toast.success("策略已保存"); cancelEdit(); return;
    }
    onRulesChange(rules.map((r) => r.id === ruleId ? { ...r, groupIds: draftGroupIds, value } : r));
    toast.success("策略已保存"); cancelEdit();
  };

  const deleteRule = (ruleId: string) => {
    onRulesChange(rules.filter((r) => r.id !== ruleId));
    toast.success("策略已删除");
  };

  return (
    <>
      <CDPolicyOverviewCard
        icon={icon}
        iconBg={iconBg}
        title={title}
        description={description}
        fallbackSummary={<span className="text-[var(--text-emphasis)] font-medium tabular-nums">{formatValue(fallbackRule.value)}</span>}
        groupCount={groupRules.length}
        onClick={() => setDialogOpen(true)}
      />

      <Dialog open={dialogOpen} onOpenChange={(v) => { setDialogOpen(v); if (!v) cancelEdit(); }}>
        <DialogContent className="sm:max-w-[960px]">
          <DialogHeader><DialogTitle>{title}</DialogTitle></DialogHeader>
          <DialogBody className="px-6">
            <div className="space-y-3">
              <div className="flex items-center gap-4 text-[13px]">
                <span className="text-[var(--text-muted)] inline-flex items-center gap-1">预设策略：<span className="text-[var(--text-emphasis)] font-medium tabular-nums">{formatValue(fallbackRule.value)}</span></span>
                <span className="text-[var(--text-muted)]">分组策略：<span className="text-[var(--text-emphasis)] font-medium">{groupRules.length} 个</span></span>
              </div>

              <div className="rounded-[var(--radius-lg)] border border-[var(--cp-border)] overflow-hidden">
                <Table density="compact" variant="gray-header">
                  <colgroup><col style={{ width: 90 }} /><col /><col style={{ width: 140 }} /><col style={{ width: 100 }} /></colgroup>
                  <TableHeader>
                    <TableRow><TableHead>策略类型</TableHead><TableHead>应用范围</TableHead><TableHead>配额上限</TableHead><TableHead>操作</TableHead></TableRow>
                  </TableHeader>
                  <TableBody>
                    {/* 预设策略行 */}
                    <TableRow>
                      <TableCell className="text-[var(--text-muted)]">预设策略</TableCell>
                      <TableCell>{groupRules.length > 0 ? "全部用户（分组策略用户除外）" : "全部用户"}</TableCell>
                      <TableCell>
                        {editingId === fallbackRule.id ? (
                          <Input type="number" min={0} max={999} value={draftValueStr} onChange={(e) => setDraftValueStr(e.target.value)} className="h-8 w-[120px] text-xs" placeholder="0-999" />
                        ) : (
                          <span className="tabular-nums">{formatValue(fallbackRule.value)}</span>
                        )}
                      </TableCell>
                      <TableActionCell>
                        {editingId === fallbackRule.id ? (
                          <>
                            <Button variant="link" onClick={cancelEdit}>取消</Button>
                            <Button variant="link" onClick={() => saveEdit(fallbackRule.id)}>保存</Button>
                          </>
                        ) : (
                          <Button variant="link" onClick={() => startEdit(fallbackRule)}>编辑</Button>
                        )}
                      </TableActionCell>
                    </TableRow>

                    {/* 分组策略行 */}
                    {groupRules.map((rule, idx) => (
                      <TableRow key={rule.id}>
                        <TableCell className="text-[var(--text-muted)]">分组策略{idx + 1}</TableCell>
                        <TableCell>
                          {editingId === rule.id ? (
                            <GroupSelect
                              groups={MOCK_GROUPS}
                              selectedIds={draftGroupIds}
                              disabledIds={getDisabledIds(rule.id)}
                              onChange={setDraftGroupIds}
                              disabledTooltip="该分组已被其他策略使用"
                              placeholder="选择分组"
                              variant="confirm"
                              onSave={() => saveEdit(rule.id)}
                            />
                          ) : (
                            <CDGroupBadges groupIds={rule.groupIds} />
                          )}
                        </TableCell>
                        <TableCell>
                          {editingId === rule.id ? (
                            <Input type="number" min={0} max={999} value={draftValueStr} onChange={(e) => setDraftValueStr(e.target.value)} className="h-8 w-[120px] text-xs" placeholder="0-999" />
                          ) : (
                            <span className="tabular-nums">{formatValue(rule.value)}</span>
                          )}
                        </TableCell>
                        <TableActionCell>
                          {editingId === rule.id ? (
                            <>
                              <Button variant="link" onClick={cancelEdit}>取消</Button>
                              <Button variant="link" onClick={() => saveEdit(rule.id)}>保存</Button>
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
                        <TableCell className="text-[var(--text-muted)]">分组策略{groupRules.length + 1}</TableCell>
                        <TableCell>
                          <GroupSelect
                            groups={MOCK_GROUPS}
                            selectedIds={draftGroupIds}
                            disabledIds={getDisabledIds()}
                            onChange={setDraftGroupIds}
                            disabledTooltip="该分组已被其他策略使用"
                            placeholder="选择分组"
                            variant="confirm"
                            onSave={() => saveEdit()}
                          />
                        </TableCell>
                        <TableCell>
                          <Input type="number" min={0} max={999} value={draftValueStr} onChange={(e) => setDraftValueStr(e.target.value)} className="h-8 w-[120px] text-xs" placeholder="0-999" />
                        </TableCell>
                        <TableActionCell>
                          <Button variant="link" onClick={cancelEdit}>取消</Button>
                          <Button variant="link" disabled={draftGroupIds.length === 0} onClick={() => saveEdit()}>保存</Button>
                        </TableActionCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>

                <button
                  type="button"
                  onClick={startAdd}
                  disabled={editingId !== null || addingNew}
                  className="w-full flex items-center justify-center gap-1 px-3 py-2 text-[13px] text-[var(--text-emphasis)] bg-white border-t border-dashed border-[var(--cp-border)] hover:bg-[var(--bg-grey-normal)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <Plus className="w-3.5 h-3.5" />添加组织策略
                </button>
              </div>
            </div>
          </DialogBody>
          <DialogFooter />
        </DialogContent>
      </Dialog>
    </>
  );
}

/* eslint-enable @typescript-eslint/no-unused-vars */

// ── 用户账号（与用户管理 MemberManagement 一致） ──────────────────────
interface UserRef {
  id: string;
  name: string;
  email: string;
  department: string;
}

const USERS: UserRef[] = [
  { id: "alice@acompany.com", name: "Alice", email: "alice@acompany.com", department: "技术部/前端组" },
  { id: "bob@acompany.com", name: "Bob", email: "bob@acompany.com", department: "技术部/后端组" },
  { id: "carol@acompany.com", name: "Carol", email: "carol@acompany.com", department: "技术部/AI 组" },
  { id: "david@acompany.com", name: "David", email: "david@acompany.com", department: "产品部/产品策划" },
  { id: "eve@acompany.com", name: "Eve", email: "eve@acompany.com", department: "产品部/设计组" },
  { id: "frank@acompany.com", name: "Frank", email: "frank@acompany.com", department: "技术部/前端组" },
  { id: "grace@acompany.com", name: "Grace", email: "grace@acompany.com", department: "技术部/后端组" },
  { id: "henry@acompany.com", name: "Henry", email: "henry@acompany.com", department: "人力资源" },
  { id: "iris@acompany.com", name: "Iris", email: "iris@acompany.com", department: "技术部/AI 组" },
  { id: "jack@acompany.com", name: "Jack", email: "jack@acompany.com", department: "财务部" },
  { id: "kate@acompany.com", name: "Kate", email: "kate@acompany.com", department: "技术部/前端组" },
  { id: "leo@acompany.com", name: "Leo", email: "leo@acompany.com", department: "产品部/产品策划" },
  { id: "mike@acompany.com", name: "Mike", email: "mike@acompany.com", department: "技术部/后端组" },
  { id: "nina@acompany.com", name: "Nina", email: "nina@acompany.com", department: "产品部/设计组" },
  { id: "oscar@acompany.com", name: "Oscar", email: "oscar@acompany.com", department: "财务部" },
];

// ── Mock 环境数据 ────────────────────────────────────────────────
const MOCK: Env[] = [
  { id: "1", envId: "openclaw-prod-8g2k1a", name: "生产环境", status: "running", region: "ap-guangzhou", packageName: "标准版", storageUsed: "2.3 GB / 10 GB", dbUsed: "156 MB / 2 GB", functionCount: 12, staticHosting: true, createdAt: "2025-12-01", expireAt: "2026-12-01", lastDeployAt: "2026-03-28 14:32:00", appCount: 3, appNames: ["库存管理系统", "CRM 后台", "数据看板"], createdBy: "alice@acompany.com", allowedGroups: ["dept-tech"], allowedUsers: [], dbType: "postgresql", overflowBilling: false, autoRenewal: true },
  { id: "2", envId: "openclaw-staging-5f3m2b", name: "预发布环境", status: "running", region: "ap-guangzhou", packageName: "标准版", storageUsed: "1.1 GB / 10 GB", dbUsed: "89 MB / 2 GB", functionCount: 8, staticHosting: true, createdAt: "2026-01-15", expireAt: "2027-01-15", lastDeployAt: "2026-03-27 09:15:00", appCount: 2, appNames: ["API 网关", "日志服务"], createdBy: "", allowedGroups: ["dept-tech", "dept-be"], allowedUsers: [], dbType: "postgresql", overflowBilling: false, autoRenewal: true },
  { id: "3", envId: "openclaw-dev-9h4n3c", name: "开发环境", status: "running", region: "ap-beijing", packageName: "标准版", storageUsed: "0.8 GB / 10 GB", dbUsed: "42 MB / 2 GB", functionCount: 15, staticHosting: false, createdAt: "2026-02-01", expireAt: "2027-02-01", lastDeployAt: "2026-03-29 16:45:00", appCount: 5, appNames: ["模型训练平台", "数据标注工具", "推理服务", "监控面板", "文档站"], createdBy: "alice@acompany.com", allowedGroups: [], allowedUsers: ["alice@acompany.com", "eve@acompany.com", "iris@acompany.com"], dbType: "cloud", overflowBilling: false, autoRenewal: true },
  { id: "4", envId: "openclaw-test-2j5p4d", name: "测试环境", status: "stopped", region: "ap-guangzhou", packageName: "个人版", storageUsed: "0.2 GB / 10 GB", dbUsed: "15 MB / 2 GB", functionCount: 3, staticHosting: false, createdAt: "2026-02-20", expireAt: "2027-02-20", lastDeployAt: "2026-03-10 11:20:00", appCount: 1, appNames: ["需求管理工具"], createdBy: "carol@acompany.com", allowedGroups: [], allowedUsers: ["carol@acompany.com"], dbType: "cloud", overflowBilling: false, autoRenewal: true },
  { id: "5", envId: "openclaw-demo-7k6q5e", name: "演示环境", status: "creating", region: "ap-shanghai", packageName: "个人版", storageUsed: "0 GB / 10 GB", dbUsed: "0 MB / 2 GB", functionCount: 0, staticHosting: false, createdAt: "2026-03-30", expireAt: "2027-03-30", lastDeployAt: "-", appCount: 0, appNames: [], createdBy: "", allowedGroups: [], allowedUsers: [], dbType: "cloud", overflowBilling: false, autoRenewal: true },
  { id: "6", envId: "openclaw-crm-3l7r6f", name: "CRM 系统环境", status: "error", region: "ap-guangzhou", packageName: "标准版", storageUsed: "4.7 GB / 50 GB", dbUsed: "1.2 GB / 5 GB", functionCount: 28, staticHosting: true, createdAt: "2025-11-10", expireAt: "2026-05-10", lastDeployAt: "2026-03-25 08:30:00", appCount: 1, appNames: ["运营数据中心"], createdBy: "david@acompany.com", allowedGroups: ["dept-product", "dept-operation"], allowedUsers: [], dbType: "postgresql", overflowBilling: false, autoRenewal: true },
  { id: "7", envId: "openclaw-hr-8m8s7g", name: "HR 管理环境", status: "running", region: "ap-beijing", packageName: "标准版", storageUsed: "0.5 GB / 10 GB", dbUsed: "78 MB / 2 GB", functionCount: 6, staticHosting: true, createdAt: "2026-01-20", expireAt: "-", lastDeployAt: "2026-03-26 17:00:00", appCount: 2, appNames: ["招聘系统", "考勤小程序"], createdBy: "", allowedGroups: ["dept-hr"], allowedUsers: [], dbType: "cloud", overflowBilling: false, autoRenewal: true },
  { id: "8", envId: "openclaw-mini-4n9t8h", name: "小程序环境", status: "running", region: "ap-guangzhou", packageName: "标准版", storageUsed: "1.8 GB / 10 GB", dbUsed: "210 MB / 2 GB", functionCount: 9, staticHosting: false, createdAt: "2026-02-10", expireAt: "2027-02-10", lastDeployAt: "2026-03-29 10:10:00", appCount: 1, appNames: ["组件文档站"], createdBy: "eve@acompany.com", allowedGroups: [], allowedUsers: ["eve@acompany.com", "iris@acompany.com"], dbType: "cloud", overflowBilling: false, autoRenewal: true },
];

const RM: Record<string, string> = { "ap-guangzhou": "广州", "ap-beijing": "北京", "ap-shanghai": "上海" };
// 状态列：对齐 Agent 列表页（OpenClawMonitor）— 用 StatusTag mode="text"（纯文字 + 圆点，无背景胶囊）
const SC: Record<EnvStatus, { label: string; tagVariant: "green" | "blue" | "red" | "gray" }> = {
  running:  { label: "运行中", tagVariant: "green" },
  stopped:  { label: "已停用", tagVariant: "gray" },
  creating: { label: "创建中", tagVariant: "blue" },
  error:    { label: "异常",   tagVariant: "red" },
};
const PS = 10;

export default function CloudDevManagement() {
  const [envs, setEnvs] = useState<Env[]>(MOCK);
  const [search, setSearch] = useState("");
  const [groupFilter, setGroupFilter] = useState("");
  const [page, setPage] = useState(1);
  const [refreshing, setRefreshing] = useState(false);
  const [detailEnv, setDetailEnv] = useState<Env | null>(null);
  const [deleteEnv, setDeleteEnv] = useState<Env | null>(null);
  // 受控打开"应用范围"Popover：分别跟踪来自徽章列铅笔图标 vs 操作列按钮的触发
  const [scopeOpenFromBadge, setScopeOpenFromBadge] = useState<string | null>(null);
  const [scopeOpenFromAction, setScopeOpenFromAction] = useState<string | null>(null);
  const [showAdd, setShowAdd] = useState(false);
  const [showClaim, setShowClaim] = useState(false);
  const [showActivity, setShowActivity] = useState(true);
  /** 预设给"新建环境"弹窗的初始表单值（默认 undefined 即用组件 DEFAULT_CREATE_ENV_FORM） */
  const [createInitialForm, setCreateInitialForm] = useState<Partial<CreateEnvForm> | undefined>(undefined);

  // 注：旧的"调整可使用范围"Dialog 已被表格行内 <ScopeUserOrGroupPopover> 替换，
  // 不再需要 showUserBind / bindTarget / editAllowedUsers / editAllowedGroups / bindTab state。

  // ── 策略配置 state（PolicyRule[] 数组，兼容 CDQuotaPolicyCard / CDTogglePolicyCard）──
  const [envQuotaRules, setEnvQuotaRules] = useState<PolicyRule<number>[]>([
    { id: "fb-quota", groupIds: [], value: 0 },
  ]);
  const [createEnvRules, setCreateEnvRules] = useState<PolicyRule<boolean>[]>([
    { id: "fb-create", groupIds: [], value: true },
  ]);

  // 分组筛选：收集所选分组及其子孙 ID
  const collectGroupIds = (nodes: GroupNode[], targetId: string): string[] => {
    const ids: string[] = [];
    const collect = (node: GroupNode) => { ids.push(node.id); node.children?.forEach(collect); };
    const find = (list: GroupNode[]): boolean => {
      for (const n of list) { if (n.id === targetId) { collect(n); return true; } if (n.children && find(n.children)) return true; }
      return false;
    };
    find(nodes);
    return ids;
  };

  // ── 分组筛选树：将 MOCK_GROUPS 转为 GroupNode[] 格式供 TreeSelect 使用 ──
  const filterGroupTree = useMemo<GroupNode[]>(() => {
    const tree = buildGroupTree(MOCK_GROUPS);
    const toGroupNode = (nodes: GroupTreeNode[]): GroupNode[] =>
      nodes.map(n => ({
        id: n.id,
        name: n.name,
        path: n.path,
        ...(n.children.length > 0 ? { children: toGroupNode(n.children) } : {}),
      }));
    return toGroupNode(tree);
  }, []);

  const filtered = useMemo(() => {
    let result = envs;
    // 分组筛选
    if (groupFilter) {
      const allowedGroupIds = collectGroupIds(filterGroupTree, groupFilter);
      result = result.filter(e => e.allowedGroups.some(g => allowedGroupIds.includes(g)));
    }
    // 关键词搜索
    if (search) {
      const q = search.toLowerCase();
      result = result.filter(e => e.name.toLowerCase().includes(q) || e.envId.toLowerCase().includes(q));
    }
    return result;
  }, [envs, search, groupFilter, filterGroupTree]);
  const tp = Math.max(1, Math.ceil(filtered.length / PS));
  const cp = Math.min(page, tp);
  const pg = filtered.slice((cp - 1) * PS, cp * PS);
  const tot = envs.length, inUse = envs.filter(e => e.allowedUsers.length > 0 || e.allowedGroups.length > 0).length, boundUsers = new Set(envs.flatMap(e => e.allowedUsers)).size, boundGroups = new Set(envs.flatMap(e => e.allowedGroups)).size;
  const boundText = `${boundUsers}/${boundGroups}`;

  const pkgNames: Record<string, string> = { free: "免费体验版", personal: "个人版", standard: "标准版", enterprise: "企业版" };

  // 创建环境（由 <CloudDevCreateEnvDialog onConfirm> 调用，form 已通过组件内部校验）
  const handleAdd = (form: CreateEnvForm) => {
    const id = String(Date.now());
    const eid = `openclaw-${form.name.replace(/\s+/g, "").toLowerCase().slice(0, 6)}-${Math.random().toString(36).slice(2, 8)}`;
    const d = new Date();
    const t = d.toISOString().slice(0, 10);
    d.setFullYear(d.getFullYear() + 1);
    setEnvs([...envs, {
      id, envId: eid, name: form.name, status: "creating", region: form.region, packageName: pkgNames[form.pkg] || form.pkg,
      storageUsed: "0 GB / 10 GB", dbUsed: "0 MB / 2 GB", functionCount: 0, staticHosting: false,
      createdAt: t, expireAt: d.toISOString().slice(0, 10), lastDeployAt: "-", appCount: 0, appNames: [],
      createdBy: "alice@acompany.com", allowedGroups: [], allowedUsers: [], dbType: form.dbType, overflowBilling: form.overflowBilling, autoRenewal: form.autoRenewal,
    }]);
    setShowAdd(false);
    toast.success(`环境「${form.name}」已创建`);
  };

  // 已领取的免费体验环境数（基于 Env.claimedFromTrial 标记字段）
  const claimedTrialCount = envs.filter(e => e.claimedFromTrial).length;
  const claimedReachedLimit = claimedTrialCount >= MAX_TRIAL_CLAIM;

  // 打开「免费体验领取」弹窗（独立于新建环境弹窗）
  const handleOpenClaimDialog = () => {
    if (claimedReachedLimit) {
      toast.error("已达领取上限", {
        description: `当前云账号已领取 ${MAX_TRIAL_CLAIM} 个免费体验环境，无法继续领取。`,
      });
      return;
    }
    setShowClaim(true);
  };

  // 确认领取免费体验：创建一个固定为「标准版」、6 个月有效期的环境
  const handleClaimConfirm = (envName: string) => {
    const id = String(Date.now());
    const eid = `openclaw-${envName.replace(/\s+/g, "").toLowerCase().slice(0, 6)}-${Math.random().toString(36).slice(2, 8)}`;
    const today = new Date();
    const expire = new Date(today);
    expire.setMonth(expire.getMonth() + 6); // 固定免费 6 个月

    setEnvs([
      ...envs,
      {
        id,
        envId: eid,
        name: envName,
        status: "creating",
        region: "ap-guangzhou",
        packageName: "标准版",
        storageUsed: "0 GB / 10 GB",
        dbUsed: "0 MB / 2 GB",
        functionCount: 0,
        staticHosting: false,
        createdAt: today.toISOString().slice(0, 10),
        expireAt: expire.toISOString().slice(0, 10),
        lastDeployAt: "-",
        appCount: 0,
        appNames: [],
        createdBy: "alice@acompany.com",
        allowedGroups: [],
        allowedUsers: [],
        dbType: "cloud",
        overflowBilling: false,
        autoRenewal: false, // 免费体验环境默认不自动续费
        claimedFromTrial: true,
      },
    ]);
    setShowClaim(false);
    toast.success(`已领取免费体验环境「${envName}」`, {
      description: "标准版规格 · 免费使用 6 个月",
    });
  };

  const handleRefresh = () => { setRefreshing(true); setTimeout(() => { setRefreshing(false); toast.success("环境列表已刷新"); }, 1000); };  const handleDel = () => { if (!deleteEnv) return; setEnvs(envs.filter(e => e.id !== deleteEnv.id)); const n = deleteEnv.name; setDeleteEnv(null); toast.success(`环境「${n}」已删除`); };
  const handleCopy = (t: string) => { navigator.clipboard.writeText(t); toast.success("已复制到剪贴板"); };

  // 更新环境的可使用范围（来自 <ScopeUserOrGroupPopover> 确认回调）
  // 互斥：mode="groups" 时 userIds 强制为空；mode="users" 时 groupIds 强制为空
  const updateEnvScope = (envId: string, scopeMode: "groups" | "users", groupIds: string[], userIds: string[]) => {
    const target = envs.find(e => e.id === envId);
    if (!target) return;
    setEnvs(envs.map(e => e.id === envId ? { ...e, allowedGroups: scopeMode === "groups" ? groupIds : [], allowedUsers: scopeMode === "users" ? userIds : [] } : e));
    const suffix = scopeMode === "groups"
      ? `（${groupIds.length} 个分组）`
      : `（${userIds.length} 位用户）`;
    toast.success(`已更新「${target.name}」的可用范围${suffix}`);
  };

  // 根据用户 ID 获取用户名
  const userName = (uid: string) => USERS.find(u => u.id === uid)?.name || uid;

  // ── 策略卡片分组渲染逻辑 ──
  const allPolicyGroups = useMemo(() => MOCK_GROUPS.filter(g => g.source !== "oneid-group"), []);
  const groupMap = useMemo(() => new Map(allPolicyGroups.map(g => [g.id, g])), [allPolicyGroups]);

  const getGroupPath = (groupId: string): string => {
    const parts: string[] = [];
    let node = groupMap.get(groupId);
    while (node) {
      parts.unshift(node.name);
      node = node.parentId ? groupMap.get(node.parentId) : undefined;
    }
    return parts.join("/");
  };

  // 注：原 PolicyEditCard / QuotaPolicyCard 通过 renderGroupSelector / renderGroupBadges
  // 注入分组渲染；现策略卡换成本地 CDTogglePolicyCard / CDQuotaPolicyCard（内嵌 GroupSelect
  // + CDGroupBadges），不再需要 render prop 注入。getGroupPath / allPolicyGroups
  // 仍在表格列 / 用户绑定弹窗的分组路径展示中使用。

  return (
    <>
      <div className="page-enter space-y-6">
        <AdminPageHeader
          title="云开发管理"
          description="管理企业云开发环境的创建、分配与生命周期。管理员可为成员分配独立的云开发环境，统一配置运行环境与规格，为成员提供应用开发及部署能力。"
        />

        {/* ── 活动引导：云开发 × Clawpro 免费体验（沿用 design-refresh 营销样式） ── */}
        {showActivity && (
          <div className="relative flex items-center gap-4 p-5 rounded-[var(--radius-lg)] border border-[var(--alert-info-border)] bg-white/60 overflow-hidden">
            <img src="/assets/admin-platform-policy/user-daily-token-limit.svg" alt="" aria-hidden="true" className="w-11 h-11 flex-shrink-0" />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <h3 className="text-sm font-semibold text-[var(--text-title)]">云开发 × Clawpro 免费体验</h3>
                <span className="text-[11px] px-2 py-0.5 rounded-full bg-orange-100 text-orange-600 font-medium flex-shrink-0">限时活动</span>
              </div>
              <p className="text-xs text-[var(--text-muted)] mt-0.5 leading-relaxed">
                即刻为当前 Clawpro 平台领取 <span className="font-semibold text-[var(--text-title)]">标准版</span> 云开发环境，
                <span className="font-semibold text-[var(--text-title)]">免费使用 6 个月</span>，
                每个云账号最多领取 <span className="font-semibold text-[var(--text-title)]">{MAX_TRIAL_CLAIM} 个</span>
                （已领 {claimedTrialCount}/{MAX_TRIAL_CLAIM}）。
              </p>
            </div>
            <Button
              variant="claw-outline"
              size="sm"
              className="flex-shrink-0"
              onClick={handleOpenClaimDialog}
              disabled={claimedReachedLimit}
            >
              <Sparkles className="w-3.5 h-3.5 mr-1.5" />
              {claimedReachedLimit ? "已达领取上限" : "立即领取"}
            </Button>
            <button onClick={() => setShowActivity(false)} className="flex-shrink-0 w-7 h-7 rounded-[var(--radius-lg)] flex items-center justify-center text-[var(--text-weak)] hover:text-[var(--text-secondary)] hover:bg-white/70 transition-all" aria-label="关闭活动">
              <X className="w-4 h-4" />
            </button>
          </div>
        )}

        {/* ── KPI 概览（3 张 NumberCard，沿用 design-refresh 资源库 SVG 图标） ── */}
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { iconSrc: "/assets/admin-memory-management/instance-total.svg", label: "环境总数", value: tot },
            { iconSrc: "/icon/已开通智能体网盘.svg", label: "使用中环境", value: inUse },
            { iconSrc: "/icon/memory-personal.svg", label: "使用用户/分组数", value: boundText },
          ].map(s => (
            <NumberCard
              key={s.label}
              icon={<img src={s.iconSrc} width={18} height={18} alt="" aria-hidden="true" className="h-[18px] w-[18px] shrink-0" />}
              label={s.label}
              value={s.value}
            />
          ))}
        </div>

        {/* ── 环境策略配置（参考网盘管理 §智能体网盘 配置项布局：h2 + 描述 + 双列策略卡平铺） ── */}
        <div className="space-y-4">
          <div>
            <h2 className="font-semibold text-[var(--text-title)]">环境策略配置</h2>
            <p className="text-sm text-[var(--text-muted)] mt-1">为成员设置创建云开发环境的配额与权限。预设策略对全部用户生效，可按分组追加例外。</p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <CDQuotaPolicyCard
              icon={<img src="/assets/admin-network-features/enterprise-network-interconnect.svg" alt="" aria-hidden="true" className="shrink-0 w-[42px]" />}
              title="单用户创建环境数量上限"
              description="每个用户最多可创建的云开发环境数量，超出后需管理员分配"
              unit="个"
              rules={envQuotaRules}
              onRulesChange={setEnvQuotaRules}
            />
            <CDTogglePolicyCard
              icon={<img src="/assets/admin-platform-policy/allow-model-quota.svg" alt="" aria-hidden="true" className="shrink-0 w-[42px]" />}
              title="允许用户创建环境"
              description="开启后，用户可在 Agent 工作台中自行创建云开发环境"
              rules={createEnvRules}
              onRulesChange={setCreateEnvRules}
            />
          </div>
        </div>

        {/* ── 环境列表 ── */}
        <div className="space-y-4">
          <div>
            <h2 className="font-semibold text-[var(--text-title)]">环境列表</h2>
            <p className="text-sm text-[var(--text-muted)] mt-1">查看并管理所有已创建的云开发环境，支持搜索、按分组筛选、关联用户/分组与删除。</p>
          </div>

          <div className="flex flex-wrap gap-3 items-center">
            <div className="relative"><Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-weak)]" /><Input placeholder="搜索环境名称或 ID..." value={search} onChange={e => { setSearch(e.target.value); setPage(1); }} className="pl-9 bg-[var(--cp-surface)] w-72" /></div>
            <TreeSelect
              nodes={filterGroupTree}
              value={groupFilter}
              onChange={v => { setGroupFilter(v); setPage(1); }}
              allLabel="全部分组"
              searchPlaceholder="搜索分组"
              triggerWidth={140}
              panelWidth={280}
              align="start"
            />
            <Button variant="claw-outline" size="claw-square" onClick={handleRefresh} aria-label="刷新">
              <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
            </Button>
            <div className="ml-auto flex items-center gap-3">
              <span className="text-sm text-[var(--text-weak)]">共 {filtered.length} 个环境</span>
              <Button
                variant="claw-primary"
                onClick={() => {
                  setCreateInitialForm(undefined);
                  setShowAdd(true);
                }}
              >
                <Plus className="w-3.5 h-3.5 mr-1.5" />新建环境
              </Button>
            </div>
          </div>

          <div className="bg-white rounded-[var(--radius-lg)] border border-[var(--cp-border)]">
            <Table variant="white">
              <colgroup>
                <col />
                <col style={{ width: 100 }} />
                <col style={{ width: 120 }} />
                <col style={{ width: 240 }} />
                <col style={{ width: 160 }} />
                <col style={{ width: 160 }} />
                <col style={{ width: 140 }} />
                <col style={{ width: 200 }} />
              </colgroup>
              <TableHeader>
                <TableRow>
                  <TableHead>环境</TableHead>
                  <TableHead>状态</TableHead>
                  <TableHead>规格</TableHead>
                  <TableHead>使用用户/分组</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead>到期时间</TableHead>
                  <TableHead>创建用户</TableHead>
                  <TableHead>操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pg.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} className="px-6 py-10 text-center">
                      <p className="text-sm font-medium text-[var(--text-title)]">暂无云开发环境</p>
                      <p className="text-xs text-[var(--text-weak)] mt-1">尝试调整搜索关键词或分组筛选；或点击右上角新建一个</p>
                    </TableCell>
                  </TableRow>
                ) : pg.map(env => (
                  <TableRow key={env.id}>
                    <TableCell>
                      <p className="text-sm font-medium text-[var(--text-title)]">{env.name}</p>
                      <div className="flex items-center gap-1 mt-0.5">
                        <p className="text-xs text-[var(--text-weak)] font-mono">{env.envId}</p>
                        <button onClick={() => handleCopy(env.envId)} className="text-[var(--text-weak)] hover:text-[var(--text-brand)] transition-colors" aria-label="复制环境 ID"><Copy className="w-3 h-3" /></button>
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusTag mode="text" variant={SC[env.status].tagVariant}>
                        {SC[env.status].label}
                      </StatusTag>
                    </TableCell>
                    <TableCell className="text-sm text-[var(--cp-text-body)]">{env.packageName}</TableCell>
                    <TableCell>
                      <ScopeUserOrGroupPopover
                        mode={env.allowedUsers.length > 0 ? "users" : "groups"}
                        selectedGroupIds={env.allowedGroups}
                        selectedUserIds={env.allowedUsers}
                        groups={allPolicyGroups.map(g => ({ id: g.id, name: getGroupPath(g.id), parentId: g.parentId }))}
                        users={USERS.map(u => ({ id: u.id, name: u.name, email: u.email, department: u.department }))}
                        onConfirm={(scopeMode, groupIds, userIds) => updateEnvScope(env.id, scopeMode, groupIds, userIds)}
                        emptyLabel="未分配"
                        maxVisibleBadges={3}
                        open={scopeOpenFromBadge === env.id}
                        onOpenChange={o => setScopeOpenFromBadge(o ? env.id : null)}
                      />
                    </TableCell>
                    <TableCell className="text-sm text-[var(--cp-text-body)]">{env.createdAt}</TableCell>
                    <TableCell className="text-sm text-[var(--cp-text-body)]">{env.expireAt === "-" ? "永久" : env.expireAt}</TableCell>
                    <TableCell>
                      {env.createdBy ? (
                        <div className="flex items-center gap-1.5">
                          <UserCircle className="w-3.5 h-3.5 text-[var(--text-weak)]" />
                          <span className="text-sm text-[var(--cp-text-body)]">{env.createdBy}</span>
                        </div>
                      ) : (
                        <span className="text-sm text-[var(--text-weak)]">—</span>
                      )}
                    </TableCell>
                    {/* 操作列：对齐 Agent 列表页（OpenClawMonitor）— TableActionCell + Button variant="link"（品牌蓝），
                        连"删除"等危险操作也用 link 蓝色，红/黑语义差异由文案 + 二次确认 Dialog 承载（详见 table.tsx §5）*/}
                    <TableActionCell>
                      <Button variant="link" onClick={() => setDetailEnv(env)}>查看</Button>
                      <ScopeUserOrGroupPopover
                        mode={env.allowedUsers.length > 0 ? "users" : "groups"}
                        selectedGroupIds={env.allowedGroups}
                        selectedUserIds={env.allowedUsers}
                        groups={allPolicyGroups.map(g => ({ id: g.id, name: getGroupPath(g.id), parentId: g.parentId }))}
                        users={USERS.map(u => ({ id: u.id, name: u.name, email: u.email, department: u.department }))}
                        onConfirm={(scopeMode, groupIds, userIds) => updateEnvScope(env.id, scopeMode, groupIds, userIds)}
                        emptyLabel="未分配"
                        maxVisibleBadges={3}
                        showBadges={false}
                        trigger={<Button variant="link">关联用户/分组</Button>}
                        open={scopeOpenFromAction === env.id}
                        onOpenChange={o => setScopeOpenFromAction(o ? env.id : null)}
                      />
                      <Button variant="link" onClick={() => setDeleteEnv(env)}>删除</Button>
                    </TableActionCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        {tp > 1 && (<div className="flex items-center justify-center gap-1">
          <Button variant="ghost" size="sm" className="w-7 h-7 p-0" disabled={cp <= 1} onClick={() => setPage(cp - 1)} aria-label="上一页"><ChevronLeft className="w-4 h-4" /></Button>
          {Array.from({ length: tp }, (_, i) => i + 1).map(p => p === cp ? (<button key={p} className="w-7 h-7 rounded-[var(--radius-lg)] text-white text-xs font-medium bg-[var(--cp-brand-blue)]" aria-current="page">{p}</button>) : (<Button key={p} variant="ghost" size="sm" className="w-7 h-7 text-xs text-[var(--text-muted)]" onClick={() => setPage(p)}>{p}</Button>))}
          <Button variant="ghost" size="sm" className="w-7 h-7 p-0" disabled={cp >= tp} onClick={() => setPage(cp + 1)} aria-label="下一页"><ChevronRight className="w-4 h-4" /></Button>
        </div>)}
        </div>
      </div>

      {/* ═══ 环境详情弹窗（左侧导航 + 右侧内容） ═════════════════════════════
          mode="admin"：显示「前往控制台」「列表项查看详情」，envOptions 传全量环境 */}
      <CloudDevEnvDetailDialog
        open={!!detailEnv}
        onOpenChange={o => { if (!o) setDetailEnv(null); }}
        env={detailEnv}
        mode="admin"
        envOptions={envs}
        onSelectEnv={(e) => setDetailEnv(e)}
      />

      {/* ═══ 删除确认弹窗 ════════════════════════════════════════════════════════ */}
      <AlertDialog open={!!deleteEnv} onOpenChange={o => { if (!o) setDeleteEnv(null); }}>
        <AlertDialogContent><AlertDialogHeader><AlertDialogTitle>确认删除环境</AlertDialogTitle><AlertDialogDescription>确定要删除云开发环境「{deleteEnv?.name}」（{deleteEnv?.envId}）吗？删除后该环境下的所有资源将被清理，此操作不可撤销。</AlertDialogDescription></AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel asChild><Button variant="claw-outline">取消</Button></AlertDialogCancel>
            <AlertDialogAction asChild><Button variant="destructive" onClick={handleDel}>确认删除</Button></AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ═══ 新建环境弹窗（共享组件） ═══════════════════════════════════════════ */}
      <CloudDevCreateEnvDialog
        open={showAdd}
        onOpenChange={setShowAdd}
        initialForm={createInitialForm}
        onConfirm={handleAdd}
      />

      {/* ═══ 免费体验领取弹窗（独立于新建环境） ════════════════════════════════
          规则：固定标准版 + 6 个月 + 每企业最多 3 个，仅填环境名 */}
      <CloudDevClaimTrialDialog
        open={showClaim}
        onOpenChange={setShowClaim}
        claimedCount={claimedTrialCount}
        existingNames={envs.map(e => e.name)}
        onConfirm={handleClaimConfirm}
      />
    </>
  );
}
