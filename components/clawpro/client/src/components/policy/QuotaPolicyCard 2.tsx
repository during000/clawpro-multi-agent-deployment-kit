/**
 * QuotaPolicyCard - 配额策略卡片（数额型）
 *
 * 列顺序（v2.0）：[策略类型 120] [配额值 160] [用户/组织 auto] [时间维度 120?] [操作 80?]
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
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableRow, TableCell, TableActionCell } from "@/components/ui/table";
import { Input } from "@/components/ui/input";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
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
          className="h-9 text-xs bg-white w-32"
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
    <Card className="overflow-hidden h-full py-0 gap-0">
      <div className="px-5 pt-5 pb-4">
        <div className="flex items-start gap-3">
          <div className="shrink-0">{icon}</div>
          <div className="min-w-0 flex-1">
            <h3 className="text-[14px] font-semibold text-[#020617]">{title}</h3>
            <p className="text-[12px] text-[#737373] leading-relaxed mt-1">{description}</p>
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
          {/* 预设策略 */}
          {editFallback && (
            <div className="rounded-[4px] bg-[#FAFAFA] overflow-hidden">
              <Table density="compact" autoFixedColumns={false}>
                <colgroup>
                  <col style={{ width: 120 }} />
                  <col style={{ width: 160 }} />
                  <col />
                  {timeDimension && <col style={{ width: 120 }} />}
                  <col style={{ width: 80 }} />
                </colgroup>
                <TableBody>
                  <TableRow className="border-0">
                    <TableCell className="text-[13px] text-[#737373]">预设策略</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1">{renderValueEditor(editFallback.id)}</div>
                    </TableCell>
                    <TableCell><Badge variant="outline">{editGroupRules.some(r => r.groupIds.length > 0) ? <><span>全部用户</span><span className="ml-1 text-[#A3A3A3] font-normal">组织策略用户除外</span></> : "全部用户"}</Badge></TableCell>
                    {timeDimension && (
                      <TableCell>
                        <Select value={timeDimension.value} onValueChange={(v) => timeDimension.onChange(v as "daily" | "monthly")}>
                          <SelectTrigger className="h-9 w-full text-sm bg-white">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="daily">每日</SelectItem>
                            <SelectItem value="monthly">每月</SelectItem>
                          </SelectContent>
                        </Select>
                      </TableCell>
                    )}
                    <TableActionCell />
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          )}

          {/* 组织策略 + 添加按钮 */}
          <div className="rounded-[4px] bg-[#FAFAFA] overflow-hidden">
            {editGroupRules.length > 0 && (
              <Table density="compact" autoFixedColumns={false}>
                <colgroup>
                  <col style={{ width: 120 }} />
                  <col style={{ width: 160 }} />
                  <col />
                  {timeDimension && <col style={{ width: 120 }} />}
                  <col style={{ width: 80 }} />
                </colgroup>
                <TableBody>
                  {editGroupRules.map((rule, idx) => (
                    <TableRow key={rule.id} className={idx < editGroupRules.length - 1 ? "border-b border-[#EFEFEF]" : "border-0"}>
                      <TableCell className="text-[13px] text-[#737373]">组织策略{idx + 1}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">{renderValueEditor(rule.id)}</div>
                      </TableCell>
                      <TableCell>
                        {renderGroupSelector({
                          selectedIds: rule.groupIds,
                          disabledIds: getDisabledIds(rule.id),
                          onChange: (ids) => updateGroups(rule.id, ids),
                        })}
                      </TableCell>
                      {timeDimension && (
                        <TableCell className="text-[13px] text-[#020617]">{timeDimension.value === "daily" ? "每日" : "每月"}</TableCell>
                      )}
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
              className={`w-full flex items-center justify-center gap-1 px-3 py-2.5 text-[13px] text-[#737373] ${editGroupRules.length > 0 ? "border-t border-dashed border-[#D4D4D4]" : ""} hover:text-[#020617] transition-colors`}
            >
              <Plus className="w-3.5 h-3.5" />添加组织策略
            </button>
          </div>
        </div>
      ) : (
        /* 视图态 */
        <div className="px-5 pb-4 space-y-2">
          {/* 预设策略 */}
          <div className="rounded-[4px] bg-[#FAFAFA] overflow-hidden">
            <Table density="compact" autoFixedColumns={false}>
              <colgroup>
                <col style={{ width: 120 }} />
                <col style={{ width: 160 }} />
                <col />
              </colgroup>
              <TableBody>
                <TableRow className="hover:bg-transparent border-0">
                  <TableCell className="text-[13px] text-[#737373]">预设策略</TableCell>
                  <TableCell className="text-[13px] text-[#020617] font-medium tabular-nums">
                    {displayValue(fallbackRule.value)}{timeDimension && `/${timeDimension.value === "daily" ? "每日" : "每月"}`}
                  </TableCell>
                  <TableCell><Badge variant="outline">{groupRules.length > 0 ? <><span>全部用户</span><span className="ml-1 text-[#A3A3A3] font-normal">组织策略用户除外</span></> : "全部用户"}</Badge></TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          {/* 组织策略 */}
          {groupRules.length > 0 && (
            <div className="rounded-[4px] bg-[#FAFAFA] overflow-hidden">
              <Table density="compact" autoFixedColumns={false}>
                <colgroup>
                  <col style={{ width: 120 }} />
                  <col style={{ width: 160 }} />
                  <col />
                </colgroup>
                <TableBody>
                  {groupRules.map((rule, idx) => (
                    <TableRow key={rule.id} className={`hover:bg-transparent ${idx < groupRules.length - 1 ? "border-b border-[#EFEFEF]" : "border-0"}`}>
                      <TableCell className="text-[13px] text-[#737373]">组织策略{idx + 1}</TableCell>
                      <TableCell className="text-[13px] text-[#020617] font-medium tabular-nums">
                        {displayValue(rule.value)}{timeDimension && `/${timeDimension.value === "daily" ? "每日" : "每月"}`}
                      </TableCell>
                      <TableCell>{renderGroupBadges(rule.groupIds)}</TableCell>
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
