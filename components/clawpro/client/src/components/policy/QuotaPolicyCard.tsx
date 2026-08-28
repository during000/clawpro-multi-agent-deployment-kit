/**
 * QuotaPolicyCard - 配额策略卡片（数额型）
 *
 * 列顺序（v2.0）：[策略类型 120] [配额值 120] [用户/组织 auto] [时间维度 120?] [操作 80?]
 *
 * 类型：
 *   - integer：如 Agent 数量上限（0-999 整数）
 *   - token：如 Tokens 上限（数字 / 无限制 二选一，使用 TokenValueEditor）
 *
 * 组织渲染（GroupTagSelector / GroupBadges）通过 render prop 注入。
 */
import React, { useState } from "react";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { Card, CardFooter } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableRow, TableCell, TableActionCell } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { CardTitle, MetaText } from "@/components/ui/Typography";
import { TokenValueEditor } from "./TokenValueEditor";
import type {
  PolicyRule,
  TokenLimit,
  TimeDimensionConfig,
  GroupRenderProps,
} from "./types";

export interface QuotaPolicyCardProps extends GroupRenderProps {
  icon: React.ReactNode;
  /** 兼容字段；当前实现未直接使用 */
  iconBg?: string;
  title: string;
  description: string;
  type: "integer" | "token";
  rules: PolicyRule<TokenLimit>[];
  onRulesChange: (rules: PolicyRule<TokenLimit>[]) => void;
  extraContent?: React.ReactNode;
  /** 可选：在配额列右侧追加「时间维度」列（卡片级单值，所有行共用） */
  timeDimension?: TimeDimensionConfig;
}

export function QuotaPolicyCard({
  icon,
  title,
  description,
  type,
  rules,
  onRulesChange,
  extraContent,
  timeDimension,
  renderGroupSelector,
  renderGroupBadges,
}: QuotaPolicyCardProps) {
  // 卡片级编辑态：编辑期间所有规则都可改
  const [cardEditing, setCardEditing] = useState(false);
  const [editRules, setEditRules] = useState<PolicyRule<TokenLimit>[]>([]);
  const [editValueStrs, setEditValueStrs] = useState<Record<string, string>>({});
  const [editModes, setEditModes] = useState<Record<string, "custom" | "unlimited">>({});

  const fallbackRule = rules.find((r) => r.groupIds.length === 0)!;
  const groupRules = rules.filter((r) => r.groupIds.length > 0);

  const displayValue = (v: TokenLimit) => {
    if (v === "unlimited" || v === -1) return "无限制";
    const num = Number(v).toLocaleString();
    return type === "integer" ? `${num} 个` : num;
  };

  // 在编辑态下，组织冲突依据当前草稿
  const getDisabledIds = (excludeRuleId: string) =>
    editRules.filter((r) => r.groupIds.length > 0 && r.id !== excludeRuleId).flatMap((r) => r.groupIds);

  const buildBlankGroupRule = (): PolicyRule<TokenLimit> => ({
    id: `rule-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    groupIds: [],
    value: type === "integer" ? 3 : 100000,
  });

  const startCardEdit = () => {
    let initial = [...rules];
    // 没有组织规则时，自动展示一行空白，无需显式「添加」
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
    const finalRules: PolicyRule<TokenLimit>[] = [];
    for (const r of editRules) {
      const isFallback = r.id === fallbackRule.id;
      const mode = editModes[r.id] ?? "custom";
      const valStr = editValueStrs[r.id] ?? "";
      let finalValue: TokenLimit;
      if (type === "token" && mode === "unlimited") {
        finalValue = "unlimited";
      } else {
        const n = parseInt(valStr, 10);
        if (isNaN(n) || n < 0) {
          toast.error(`请输入有效数值（${isFallback ? "预设策略" : "组织策略"}）`);
          return;
        }
        if (type === "integer" && n > 999) {
          toast.error("请输入 0-999 之间的整数");
          return;
        }
        finalValue = n;
      }
      if (!isFallback && r.groupIds.length === 0) {
        // 编辑态下若新增空白行未被填写，跳过（视为不保存该行）
        continue;
      }
      finalRules.push({ ...r, value: finalValue });
    }
    const finalGroupRules = finalRules.filter((r) => r.id !== fallbackRule.id);
    const finalFallback = finalRules.find((r) => r.id === fallbackRule.id)!;
    onRulesChange([...finalGroupRules, finalFallback]);
    toast.success("策略已保存");
    cancelCardEdit();
  };

  const updateGroups = (id: string, groupIds: string[]) =>
    setEditRules((prev) => prev.map((r) => (r.id === id ? { ...r, groupIds } : r)));
  const updateValueStr = (id: string, valStr: string) =>
    setEditValueStrs((prev) => ({ ...prev, [id]: valStr }));
  const updateMode = (id: string, mode: "custom" | "unlimited") =>
    setEditModes((prev) => ({ ...prev, [id]: mode }));

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
    setEditValueStrs((prev) => ({ ...prev, [blank.id]: type === "integer" ? "3" : "100000" }));
    setEditModes((prev) => ({ ...prev, [blank.id]: "custom" }));
  };

  // 编辑态：值编辑控件
  const renderValueEditor = (ruleId: string) => {
    const mode = editModes[ruleId] ?? "custom";
    const valStr = editValueStrs[ruleId] ?? "";
    if (type === "integer") {
      return (
        <Input
          type="number"
          value={valStr}
          onChange={(e) => updateValueStr(ruleId, e.target.value)}
          className="h-7 text-xs bg-white w-32"
          placeholder="0-999"
        />
      );
    }
    return (
      <TokenValueEditor
        mode={mode}
        valStr={valStr}
        onCommit={(nextMode, nextValStr) => {
          updateMode(ruleId, nextMode);
          updateValueStr(ruleId, nextValStr);
        }}
      />
    );
  };

  const editFallback = editRules.find((r) => r.id === fallbackRule.id);
  const editGroupRules = editRules.filter((r) => r.id !== fallbackRule.id);

  return (
    <Card className="overflow-hidden h-full py-0 gap-0 [&_[data-slot=table-container]]:!overflow-x-hidden">
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

      {cardEditing ? (
        /* 编辑态：预设策略灰底卡片 + 组织策略灰底卡片 */
        <div className="px-5 pb-4 space-y-2">
          {/* 预设策略 —— 列宽与下方组织策略表格对齐：标题 120 / 配额 160 / [时间维度 100] / 应用范围 */}
          {editFallback && (
            <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-x-auto px-4 py-3">
              <div className="flex items-center min-w-[480px]">
                <CardTitle as="h4" className="w-[120px] shrink-0 text-xs text-[var(--text-title)]">预设策略</CardTitle>
                <div className="w-[160px] shrink-0 flex items-center gap-1">{renderValueEditor(editFallback.id)}</div>
                {timeDimension && (
                  <div className="w-[100px] shrink-0">
                    <Select value={timeDimension.value} onValueChange={(v) => timeDimension.onChange(v as "daily" | "monthly")}>
                      <SelectTrigger size="sm" className="!h-7 w-[100px] text-xs bg-white">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="daily">每日</SelectItem>
                        <SelectItem value="monthly">每月</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                )}
                <Badge variant="outline">全部用户{editGroupRules.length > 0 && <span className="text-[var(--text-muted)] ml-1.5">组织策略除外</span>}</Badge>
              </div>
            </div>
          )}

          {/* 组织策略 + 添加按钮 —— 列：组织策略（序号）/ 配额 / [时间维度] / 应用范围 / 操作 */}
          <div className="rounded-[4px] bg-white border border-[var(--border)] overflow-hidden">
            {editGroupRules.length > 0 && (
              <Table density="compact" variant="white" autoFixedColumns={false} className="table-fixed">
                <colgroup>
                  <col className="w-[120px]" />
                  <col className="w-[160px]" />
                  {timeDimension && <col className="w-[100px]" />}
                  <col />
                  <col className="w-[80px]" />
                </colgroup>
                <TableBody>
                  {editGroupRules.map((rule, idx) => (
                    <TableRow key={rule.id} className={idx < editGroupRules.length - 1 ? "border-b border-[#EFEFEF]" : "border-0"}>
                      <TableCell className="text-xs text-[var(--text-muted)] tabular-nums">组织策略 {idx + 1}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">{renderValueEditor(rule.id)}</div>
                      </TableCell>
                      {timeDimension && (
                        <TableCell className="text-xs text-[var(--text-emphasis)]">{timeDimension.value === "daily" ? "每日" : "每月"}</TableCell>
                      )}
                      <TableCell>
                        {renderGroupSelector({
                          selectedIds: rule.groupIds,
                          disabledIds: getDisabledIds(rule.id),
                          onChange: (ids) => updateGroups(rule.id, ids),
                        })}
                      </TableCell>
                      <TableActionCell>
                        <Button variant="link" size="sm" onClick={() => removeRule(rule.id)}>删除</Button>
                      </TableActionCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
            {/* 添加组织按钮 —— 拉通底部 */}
            <button
              type="button"
              onClick={addBlankGroupRow}
              className={`w-full flex items-center justify-center gap-1 px-3 py-2.5 text-xs text-[var(--text-muted)] ${editGroupRules.length > 0 ? "border-t border-dashed border-[var(--border)]" : ""} hover:text-[var(--text-emphasis)] hover:bg-[var(--bg-grey-hover-subtle)] transition-colors`}
            >
              <Plus className="w-3.5 h-3.5" />添加组织策略
            </button>
          </div>
        </div>
      ) : (
        /* 视图态 */
        <div className="px-5 pb-4 space-y-2">
          {/* 预设策略 —— 列宽与下方组织策略表格对齐：标题 120 / 配额 160 / 应用范围 */}
          <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-x-auto px-4 py-3">
            <div className="flex items-center min-w-[480px]">
              <CardTitle as="h4" className="w-[120px] shrink-0 text-xs text-[var(--text-title)]">预设策略</CardTitle>
              <span className="w-[160px] shrink-0 text-xs text-[var(--text-emphasis)] font-semibold tabular-nums">
                {displayValue(fallbackRule.value)}{timeDimension && ` / ${timeDimension.value === "daily" ? "每日" : "每月"}`}
              </span>
              <Badge variant="outline">全部用户{groupRules.length > 0 && <span className="text-[var(--text-muted)] ml-1.5">组织策略除外</span>}</Badge>
            </div>
          </div>

          {/* 组织策略 —— 3 列：组织策略（序号）/ 配额 / 应用范围 */}
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
                    <TableRow key={rule.id} className={`hover:bg-transparent [&:hover_td]:!bg-transparent ${idx < groupRules.length - 1 ? "border-b border-[#EFEFEF]" : "border-0"}`}>
                      <TableCell className="text-xs !text-[#4B5563] tabular-nums">组织策略 {idx + 1}</TableCell>
                      <TableCell className="text-xs !text-[#4B5563] tabular-nums">
                        {displayValue(rule.value)}{timeDimension && ` / ${timeDimension.value === "daily" ? "每日" : "每月"}`}
                      </TableCell>
                      <TableCell className="text-xs !text-[#4B5563]">{renderGroupBadges(rule.groupIds)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </div>
      )}

      {/* 卡片底部 footer */}
      {extraContent && (
        <CardFooter className="px-5 pt-0 pb-3 flex-col items-start gap-3">
          {extraContent}
        </CardFooter>
      )}
    </Card>
  );
}

export default QuotaPolicyCard;
