/**
 * EnumPolicyCard - 策略编辑卡片（枚举多选一型）
 *
 * 适用场景：值域为有限枚举（多选一）的策略，例如「默认计费方式 = 包年包月 / 按量计费」。
 *
 * 视觉骨架完全对齐 PolicyEditCard（开关型）：
 *   - Card 容器 + 头部（icon / title / description / 编辑 or 取消+保存 按钮）
 *   - 视图态：灰底（#FAFBFD）表格行，三列 [策略类型 120] [当前值 auto] [用户/组织 auto]
 *   - 编辑态：表格行内 Select 切换枚举值；可选展示组织策略行 + 添加按钮
 *
 * 三种渲染状态：
 *   - disabledMessage 存在 → 禁用态（仅显示提示）
 *   - cardEditing = true   → 编辑态
 *   - cardEditing = false  → 视图态
 *
 * 与 PolicyEditCard 的差异：
 *   - 值域：`TValue extends string`（枚举），而非 boolean
 *   - 值的展示：通过 `options` 中的 `label` 决定（StatusTag 改为 BodyMedium 文本）
 *   - 新增 prop `supportGroups`：是否允许配置组织策略（默认 true，对齐 PolicyEditCard 行为）
 */
import React, { useState } from "react";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { Card, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableRow, TableCell, TableActionCell } from "@/components/ui/table";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { CardTitle, MetaText } from "@/components/ui/Typography";
import type { PolicyRule, GroupRenderProps } from "./types";

export interface EnumPolicyOption<TValue extends string> {
  value: TValue;
  label: string;
  /** 可选：选项右侧的辅助说明（仅在 Select 下拉项内展示，不影响 Trigger 文本） */
  description?: string;
}

export interface EnumPolicyCardProps<TValue extends string> extends Partial<GroupRenderProps> {
  icon: React.ReactNode;
  /** 兼容字段；当前实现未直接使用（保留以兼容旧调用方） */
  iconBg?: string;
  title: string;
  description: React.ReactNode;
  options: EnumPolicyOption<TValue>[];
  rules: PolicyRule<TValue>[];
  /** 返回 false 表示变更被拒绝（如前置校验失败），卡片不会弹成功 toast / 不会关闭编辑态 */
  onRulesChange: (rules: PolicyRule<TValue>[]) => boolean | void;
  /** 卡片底部额外内容（如说明 / 链接） */
  extraContent?: React.ReactNode;
  /** 标题右侧附加内容（如 Tag + 查看详情按钮） */
  titleExtra?: React.ReactNode;
  /**
   * 是否支持组织策略（默认 true，与 PolicyEditCard 行为一致）。
   * 传 false 时：隐藏组织规则区与"添加组织策略"按钮，仅维护 fallback 一行。
   */
  supportGroups?: boolean;
  /** 可选：禁用编辑按钮并显示提示信息 */
  disabledMessage?: React.ReactNode;
  /** 选择器尺寸 / 宽度（默认 144px，刚好容纳"按量计费"等四字标签） */
  selectWidthPx?: number;
  /**
   * 可选：自定义编辑态"值"列的渲染器。
   * 传入此 prop 时，编辑态下 fallback 行（以及组织策略行）的 Select 会被替换为调用方渲染。
   * 用于「计费模式」等需要并排选择卡片样式的场景。不传则使用默认的 Select。
   */
  renderValueEditor?: (args: {
    value: TValue;
    onChange: (v: TValue) => void;
    options: EnumPolicyOption<TValue>[];
  }) => React.ReactNode;
}

export function EnumPolicyCard<TValue extends string>({
  icon,
  title,
  description,
  options,
  rules,
  onRulesChange,
  extraContent,
  titleExtra,
  supportGroups = true,
  disabledMessage,
  selectWidthPx = 144,
  renderGroupSelector,
  renderGroupBadges,
  renderValueEditor,
}: EnumPolicyCardProps<TValue>) {
  // 卡片级编辑态
  const [cardEditing, setCardEditing] = useState(false);
  const [editFallbackValue, setEditFallbackValue] = useState<TValue>(options[0].value);
  const [editGroupRules, setEditGroupRules] = useState<PolicyRule<TValue>[]>([]);

  const fallbackRule = rules.find((r) => r.groupIds.length === 0)!;
  const groupRules = rules.filter((r) => r.groupIds.length > 0);

  const optionMap = React.useMemo(
    () => new Map(options.map((o) => [o.value, o])),
    [options],
  );
  const labelOf = (v: TValue) => optionMap.get(v)?.label ?? v;

  const buildBlankGroupRule = (): PolicyRule<TValue> => ({
    id: `rule-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    groupIds: [],
    value: editFallbackValue,
  });

  const startCardEdit = () => {
    let initial = [...groupRules];
    if (supportGroups && initial.length === 0) {
      initial = [{ id: `rule-${Date.now()}`, groupIds: [], value: fallbackRule.value }];
    }
    setEditFallbackValue(fallbackRule.value);
    setEditGroupRules(initial);
    setCardEditing(true);
  };

  const cancelCardEdit = () => {
    setCardEditing(false);
    setEditGroupRules([]);
  };

  const saveCardEdit = () => {
    const finalGroupRules = supportGroups
      ? editGroupRules.filter((r) => r.groupIds.length > 0)
      : [];
    const finalFallback: PolicyRule<TValue> = { ...fallbackRule, value: editFallbackValue };
    const result = onRulesChange([...finalGroupRules, finalFallback]);
    if (result === false) return;
    toast.success("策略已保存");
    cancelCardEdit();
  };

  const updateGroups = (id: string, groupIds: string[]) =>
    setEditGroupRules((prev) => prev.map((r) => (r.id === id ? { ...r, groupIds } : r)));

  const updateGroupValue = (id: string, value: TValue) =>
    setEditGroupRules((prev) => prev.map((r) => (r.id === id ? { ...r, value } : r)));

  const removeRule = (id: string) =>
    setEditGroupRules((prev) => prev.filter((r) => r.id !== id));

  const addBlankGroupRow = () =>
    setEditGroupRules((prev) => [...prev, buildBlankGroupRule()]);

  const getDisabledIds = (excludeRuleId: string) =>
    editGroupRules.filter((r) => r.groupIds.length > 0 && r.id !== excludeRuleId).flatMap((r) => r.groupIds);

  const renderValueSelect = (value: TValue, onChange: (v: TValue) => void) => (
    <Select value={value} onValueChange={(v) => onChange(v as TValue)}>
      <SelectTrigger className="h-7 text-xs" style={{ width: selectWidthPx }}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((opt) => (
          <SelectItem key={opt.value} value={opt.value}>
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );

  return (
    <Card className="overflow-hidden h-full py-0 gap-0">
      <div className="px-5 pt-5 pb-4">
        <div className="flex items-start gap-3">
          <div className={`shrink-0${disabledMessage ? " grayscale opacity-100" : ""}`}>{icon}</div>
          <div className="flex-1">
            <div className="flex items-center gap-2 overflow-visible">
              <CardTitle as="h3" className="whitespace-nowrap">{title}</CardTitle>
              {titleExtra}
            </div>
            <MetaText as="p" className="mt-1 leading-relaxed">{description}</MetaText>
          </div>
          {cardEditing ? (
            <div className="flex items-center gap-2 shrink-0">
              <Button variant="claw-outline" size="claw-sm" onClick={cancelCardEdit}>取消</Button>
              <Button variant="dialog-confirm" size="claw-sm" onClick={saveCardEdit}>保存</Button>
            </div>
          ) : (
            <Button
              variant="claw-outline"
              size="claw-sm"
              className="shrink-0"
              onClick={startCardEdit}
              disabled={!!disabledMessage}
            >
              编辑
            </Button>
          )}
        </div>
      </div>

      {disabledMessage ? (
        /* 禁用态 */
        <div className="px-5 pb-4 space-y-2">
          <div className="rounded-[4px] bg-[#FAFBFD] overflow-hidden">
            <Table density="compact" autoFixedColumns={false}>
              <colgroup>
                <col style={{ width: 120 }} />
                <col style={{ width: 120 }} />
                <col />
              </colgroup>
              <TableBody>
                <TableRow className="hover:bg-transparent border-0">
                  <TableCell className="text-[13px] text-[var(--text-muted)]">预设策略</TableCell>
                  <TableCell className="text-[13px] text-[var(--text-emphasis)] font-medium">
                    {labelOf(fallbackRule.value)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2 flex-wrap">
                      {supportGroups && <Badge variant="outline">全部用户</Badge>}
                      <span className="text-[13px] text-[var(--text-weak)]">{disabledMessage}</span>
                    </div>
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>
        </div>
      ) : cardEditing ? (
        /* 编辑态 */
        <div className="px-5 pb-4 space-y-2">
          {/* 预设策略 */}
          {renderValueEditor && !supportGroups ? (
            // 自定义编辑器（如计费模式的并排卡片）：脱离 Table，直接整宽渲染
            <div>
              {renderValueEditor({
                value: editFallbackValue,
                onChange: setEditFallbackValue,
                options,
              })}
            </div>
          ) : (
            <div className="rounded-[4px] bg-[#FAFBFD] overflow-hidden">
              <Table density="compact" autoFixedColumns={false}>
                <colgroup>
                  <col style={{ width: 120 }} />
                  <col style={{ width: 120 }} />
                  <col />
                  <col style={{ width: 80 }} />
                </colgroup>
                <TableBody>
                  <TableRow className="border-0">
                    <TableCell className="text-[13px] text-[var(--text-muted)]">预设策略</TableCell>
                    <TableCell>
                      <div className="flex items-center">
                        {renderValueSelect(editFallbackValue, setEditFallbackValue)}
                      </div>
                    </TableCell>
                    <TableCell>
                      {supportGroups && (
                        <Badge variant="outline">
                          {editGroupRules.some((r) => r.groupIds.length > 0) ? (
                            <>
                              <span>全部用户</span>
                              <span className="ml-1 text-[var(--text-weak)] font-normal">组织策略用户除外</span>
                            </>
                          ) : (
                            "全部用户"
                          )}
                        </Badge>
                      )}
                    </TableCell>
                    <TableActionCell />
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          )}

          {/* 组织策略 + 添加按钮 */}
          {supportGroups && renderGroupSelector && (
            <div className="rounded-[4px] bg-[#FAFBFD] overflow-hidden">
              <Table density="compact" autoFixedColumns={false}>
                <colgroup>
                  <col style={{ width: 120 }} />
                  <col style={{ width: 120 }} />
                  <col />
                  <col style={{ width: 80 }} />
                </colgroup>
                <TableBody>
                  {editGroupRules.map((rule, idx) => (
                    <TableRow key={rule.id} className={idx < editGroupRules.length - 1 ? "border-b border-[#EFEFEF]" : "border-0"}>
                      <TableCell className="text-[13px] text-[var(--text-muted)]">组织策略{idx + 1}</TableCell>
                      <TableCell>
                        <div className="flex items-center">
                          {renderValueSelect(rule.value, (v) => updateGroupValue(rule.id, v))}
                        </div>
                      </TableCell>
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
              <button
                type="button"
                onClick={addBlankGroupRow}
                className="w-full flex items-center justify-center gap-1 px-3 py-2.5 text-[13px] text-[var(--text-muted)] border-t border-dashed border-[var(--border)] hover:text-[var(--text-emphasis)] transition-colors"
              >
                <Plus className="w-3.5 h-3.5" />添加组织策略
              </button>
            </div>
          )}
        </div>
      ) : (
        /* 视图态 */
        <div className="px-5 pb-4 space-y-2">
          <div className="rounded-[4px] bg-[#FAFBFD] overflow-hidden">
            <Table density="compact" autoFixedColumns={false}>
              <colgroup>
                <col style={{ width: 120 }} />
                <col style={{ width: 120 }} />
                <col />
              </colgroup>
              <TableBody>
                <TableRow className="hover:bg-transparent border-0">
                  <TableCell className="text-[13px] text-[var(--text-muted)]">预设策略</TableCell>
                  <TableCell className="text-[13px] text-[var(--text-emphasis)] font-medium">
                    {labelOf(fallbackRule.value)}
                  </TableCell>
                  <TableCell>
                    {supportGroups && (
                      <Badge variant="outline">
                        {groupRules.length > 0 ? (
                          <>
                            <span>全部用户</span>
                            <span className="ml-1 text-[var(--text-weak)] font-normal">组织策略用户除外</span>
                          </>
                        ) : (
                          "全部用户"
                        )}
                      </Badge>
                    )}
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          {/* 组织策略 */}
          {supportGroups && groupRules.length > 0 && renderGroupBadges && (
            <div className="rounded-[4px] bg-[#FAFBFD] overflow-hidden">
              <Table density="compact" autoFixedColumns={false}>
                <colgroup>
                  <col style={{ width: 120 }} />
                  <col style={{ width: 120 }} />
                  <col />
                </colgroup>
                <TableBody>
                  {groupRules.map((rule, idx) => (
                    <TableRow key={rule.id} className={`hover:bg-transparent ${idx < groupRules.length - 1 ? "border-b border-[#EFEFEF]" : "border-0"}`}>
                      <TableCell className="text-[13px] text-[var(--text-muted)]">组织策略{idx + 1}</TableCell>
                      <TableCell className="text-[13px] text-[var(--text-emphasis)] font-medium">
                        {labelOf(rule.value)}
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

      {extraContent && (
        <CardFooter className="px-5 pt-0 pb-3 flex-col items-start gap-3">
          {extraContent}
        </CardFooter>
      )}
    </Card>
  );
}

export default EnumPolicyCard;
