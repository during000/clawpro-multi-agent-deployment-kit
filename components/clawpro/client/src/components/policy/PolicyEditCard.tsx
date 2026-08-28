/**
 * PolicyEditCard - 策略编辑卡片（开关型）
 *
 * 卡片级编辑态：编辑期间所有规则（预设 + 组织）都可改，统一保存。
 *
 * 列顺序（v2.0）：[策略类型 120] [是否生效 120] [用户/组织 auto] [访问方式 140?] [操作 80?]
 *
 * 三种渲染状态：
 *   - disabledMessage 存在 → 禁用态（仅显示提示）
 *   - cardEditing = true   → 编辑态
 *   - cardEditing = false  → 视图态
 *
 * 组织渲染（GroupTagSelector / GroupBadges）通过 render prop 注入，与具体组织数据解耦。
 */
import React, { useState } from "react";
import { Loader2, Plus, Info, CheckCircle2, XCircle } from "lucide-react";
import { toast } from "sonner";
import { Card, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableRow, TableCell, TableActionCell } from "@/components/ui/table";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { CardTitle, MetaText } from "@/components/ui/Typography";
import type { PolicyRule, AccessModeRowConfig, GroupRenderProps } from "./types";

export interface PolicyEditCardProps extends GroupRenderProps {
  icon: React.ReactNode;
  /** 兼容字段；当前实现未直接使用（保留以兼容旧调用方） */
  iconBg?: string;
  title: string;
  description: React.ReactNode;
  rules: PolicyRule<boolean>[];
  /** 返回 false 表示变更被拒绝（如前置校验失败），卡片不会弹成功 toast / 不会关闭编辑态 */
  onRulesChange: (rules: PolicyRule<boolean>[]) => boolean | void;
  extraContent?: React.ReactNode;
  /** 标题右侧附加内容（如 Tag + 查看详情按钮） */
  titleExtra?: React.ReactNode;
  /** 指定哪一行（rule.id）正在 loading：该行权限列显示「配置中，请勿关闭」 */
  loadingRuleId?: string | null;
  /** 可选：在预设策略行上方插入一行「访问方式」 */
  accessModeRow?: AccessModeRowConfig;
  /** 可选：禁用编辑按钮并在预设策略区域显示提示信息（ReactNode 支持 link） */
  disabledMessage?: React.ReactNode;
  /** 可选：点击编辑按钮前的校验回调；返回 false 则阻止进入编辑态 */
  onBeforeEdit?: () => boolean | void;
}

export function PolicyEditCard({
  icon,
  title,
  description,
  rules,
  onRulesChange,
  extraContent,
  titleExtra,
  loadingRuleId,
  accessModeRow,
  disabledMessage,
  onBeforeEdit,
  renderGroupSelector,
  renderGroupBadges,
}: PolicyEditCardProps) {
  // 卡片级编辑态
  const [cardEditing, setCardEditing] = useState(false);
  const [editFallbackValue, setEditFallbackValue] = useState<boolean>(true);
  const [editGroupRules, setEditGroupRules] = useState<PolicyRule<boolean>[]>([]);
  // 访问方式草稿（仅在 accessModeRow 存在时使用）
  const [editAccessMode, setEditAccessMode] = useState<"public" | "private">("public");

  const fallbackRule = rules.find((r) => r.groupIds.length === 0)!;
  const groupRules = rules.filter((r) => r.groupIds.length > 0);
  // 视图态：组织规则的值 = 兜底值的相反（「例外」语义）
  // 编辑态：以草稿兜底为基准
  const editGroupRuleValue = !editFallbackValue;

  const buildBlankGroupRule = (): PolicyRule<boolean> => ({
    id: `rule-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
    groupIds: [],
    value: !editFallbackValue,
  });

  const startCardEdit = () => {
    if (onBeforeEdit && onBeforeEdit() === false) return;
    let initial = [...groupRules];
    if (initial.length === 0) {
      // 没有组织规则时，自动放一行空白可填
      initial = [{ id: `rule-${Date.now()}`, groupIds: [], value: !fallbackRule.value }];
    }
    setEditFallbackValue(fallbackRule.value);
    setEditGroupRules(initial);
    if (accessModeRow) setEditAccessMode(accessModeRow.mode);
    setCardEditing(true);
  };

  const cancelCardEdit = () => {
    setCardEditing(false);
    setEditGroupRules([]);
  };

  const saveCardEdit = () => {
    // 仅保留填了组织的规则；统一以草稿兜底的相反值作为组织值（例外语义）
    const finalGroupRules = editGroupRules
      .filter((r) => r.groupIds.length > 0)
      .map((r) => ({ ...r, value: !editFallbackValue }));
    const finalFallback: PolicyRule<boolean> = { ...fallbackRule, value: editFallbackValue };
    const result = onRulesChange([...finalGroupRules, finalFallback]);
    if (result === false) return;
    // 同步保存访问方式
    if (accessModeRow) accessModeRow.onModeChange(editAccessMode);
    toast.success("策略已保存");
    cancelCardEdit();
  };

  const updateGroups = (id: string, groupIds: string[]) =>
    setEditGroupRules((prev) => prev.map((r) => (r.id === id ? { ...r, groupIds } : r)));

  const removeRule = (id: string) =>
    setEditGroupRules((prev) => prev.filter((r) => r.id !== id));

  const addBlankGroupRow = () =>
    setEditGroupRules((prev) => [...prev, buildBlankGroupRule()]);

  const getDisabledIds = (excludeRuleId: string) =>
    editGroupRules.filter((r) => r.groupIds.length > 0 && r.id !== excludeRuleId).flatMap((r) => r.groupIds);

  // 下拉项 / Trigger 文案前的小图标块（与 renderPermissionIndicator 视觉一致）
  // - 用作 SelectItem 的 children，因此与文字共用 ItemText，Trigger 中 SelectValue 也会复用同一结构
  // - 注意：SelectItem 父级带 [&_svg:not([class*='size-'])]:size-4 强制规则，
  //   所以这里 svg 必须用 size-* 类（不能用 w-* h-*），否则会被撑成 16px 导致图标错位
  const renderPermissionOptionLabel = (allowed: boolean) => (
    <span className="inline-flex items-center gap-1.5">
      {allowed ? (
        <CheckCircle2 className="size-3.5 shrink-0 text-[var(--text-success)]" strokeWidth={2.5} aria-hidden />
      ) : (
        <XCircle className="size-3.5 shrink-0 text-[var(--text-warning)]" strokeWidth={2.5} aria-hidden />
      )}
      {allowed ? "允许" : "不允许"}
    </span>
  );

  const renderFallbackEditor = () => (
    <Select value={editFallbackValue ? "on" : "off"} onValueChange={(v) => setEditFallbackValue(v === "on")}>
      <SelectTrigger size="sm" className="!h-7 w-[120px] text-xs">
        <SelectValue />
      </SelectTrigger>
      <SelectContent align="start">
        <SelectItem value="on">{renderPermissionOptionLabel(true)}</SelectItem>
        <SelectItem value="off">{renderPermissionOptionLabel(false)}</SelectItem>
      </SelectContent>
    </Select>
  );

  // 行内 loading 文字
  const renderLoading = () => (
    <span className="inline-flex items-center gap-1.5 text-xs text-[var(--text-brand)] font-medium whitespace-nowrap">
      <Loader2 className="w-3.5 h-3.5 animate-spin" />配置中，请勿关闭
    </span>
  );

  // 权限指示器：允许 = 绿色空心环 + 绿色对勾；不允许 = 橘黄色空心环 + 橘黄色叉
  // 替代原 StatusTag，仅用于"功能权限开关"场景下的允许/不允许语义展示
  // 描边色：允许 var(--text-success)（#16A34A）/ 不允许 var(--text-warning)（#D97706），strokeWidth=2.5 加粗
  // 文字色：统一使用 var(--text-body) 黑色正文 token
  // emphasis=true：用于预设策略行，文字字号 13px 加粗
  const renderPermissionIndicator = (allowed: boolean, emphasis = false) => (
    <span
      className={`inline-flex items-center gap-1.5 whitespace-nowrap text-[var(--text-body)] ${
        emphasis ? "text-xs font-medium" : "text-xs"
      }`}
    >
      {allowed ? (
        <CheckCircle2 className="size-3.5 shrink-0 text-[var(--text-success)]" strokeWidth={2.5} aria-hidden />
      ) : (
        <XCircle className="size-3.5 shrink-0 text-[var(--text-warning)]" strokeWidth={2.5} aria-hidden />
      )}
      {allowed ? "允许" : "不允许"}
    </span>
  );

  // 访问方式独立卡片（白底，放在组织策略下面）
  // - 列布局 120 / 160：与组织策略表格列宽对齐
  // - 小标题样式与"组织策略 n"一致：text-xs !text-[#4B5563] tabular-nums
  // - 正文样式与表格正文一致：text-xs !text-[#4B5563]
  const accessModeBlock = accessModeRow && !disabledMessage ? (
    <div className="rounded-[4px] bg-white border border-[var(--border)] overflow-hidden">
      <div className="flex items-center px-4 py-2.5">
        <div className="w-[120px] shrink-0 text-xs !text-[#4B5563] tabular-nums">访问方式</div>
        <div className="flex items-center gap-2 flex-1 min-w-0">
          {cardEditing ? (
            <Select value={editAccessMode} onValueChange={(v: "public" | "private") => setEditAccessMode(v)}>
              <SelectTrigger size="sm" className="!h-7 w-[160px] px-3 text-xs bg-background">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="public" className="text-xs">公网访问</SelectItem>
                <SelectItem value="private" className="text-xs">私网访问</SelectItem>
              </SelectContent>
            </Select>
          ) : (
            <span className="text-xs !text-[#4B5563]">{accessModeRow.mode === "public" ? "公网访问" : "私网访问"}</span>
          )}
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="cursor-default"><Info className="w-3.5 h-3.5 text-gray-400" /></span>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-[280px] text-xs leading-relaxed bg-white text-[var(--text-title)] border border-[var(--border)] shadow-[0_4px_12px_rgba(0,0,0,0.08)] px-3 py-2">
              <p className="mb-1.5"><span className="font-semibold">公网访问：</span>用户通过公网直接访问 Agent 面板（WebUI），连接云服务器公网 IP。适用于大多数场景，<span className="font-semibold">推荐选择</span>。</p>
              <p><span className="font-semibold">私网访问：</span>用户通过同一私有网络访问 Agent 面板（WebUI），连接云服务器内网 IP。使用前需先自行将企业内网与腾讯云私有网络（VPC）打通，并在「网络管理」中将云服务器绑定至该 VPC。配置完成后，企业用户可通过企业内网访问面板，但无法通过公网访问。</p>
            </TooltipContent>
          </Tooltip>
        </div>
      </div>
    </div>
  ) : null;

  return (
    <Card className="overflow-hidden h-full py-0 gap-0 [&_[data-slot=table-container]]:!overflow-x-hidden">
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
            <Button variant="claw-outline" size="claw-sm" className="shrink-0" onClick={startCardEdit} disabled={!!loadingRuleId || !!disabledMessage}>
              编辑
            </Button>
          )}
        </div>
      </div>

      {disabledMessage ? (
        /* 禁用态：显示提示信息 —— 列宽与编辑态/视图态对齐：标题 120 / 权限 160 / 应用范围 */
        <div className="px-5 pb-4 space-y-2">
          <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-hidden">
            <div className="px-4 py-3">
              <div className="flex items-center">
                <div className="w-[120px] shrink-0 flex items-center text-xs text-[var(--text-title)] font-medium leading-none">预设策略</div>
                <div className="w-[160px] shrink-0 flex items-center">{renderPermissionIndicator(false, true)}</div>
                <div className="flex items-center text-xs text-[var(--text-weak)]">{disabledMessage}</div>
              </div>
            </div>
          </div>
        </div>
      ) : cardEditing ? (
        /* 编辑态 */
        <div className="px-5 pb-4 space-y-2">
          {/* 预设策略 —— 独立浅蓝灰底卡片；列宽与下方组织策略表格对齐：标题 120 / 权限 160 / 应用范围 */}
          <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-hidden">
            <div className="px-4 py-3">
              <div className="flex items-center">
                <div className="w-[120px] shrink-0 flex items-center text-xs text-[var(--text-title)] font-medium leading-none">预设策略</div>
                <div className="w-[160px] shrink-0 flex items-center">{renderFallbackEditor()}</div>
                <div className="flex items-center">
                  <Badge variant="outline">全部用户{editGroupRules.length > 0 && <span className="text-[var(--text-muted)] ml-1.5">组织策略除外</span>}</Badge>
                </div>
              </div>
            </div>
          </div>

          {/* 组织策略 + 添加按钮 —— 列：组织策略（序号）/ 组织 / 权限 / 操作 */}
          <div className="rounded-[4px] bg-white border border-[var(--border)] overflow-hidden">
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
                    <TableCell className="text-xs text-[var(--text-muted)] tabular-nums">组织策略 {idx + 1}</TableCell>
                    <TableCell>
                      {renderPermissionIndicator(editGroupRuleValue)}
                    </TableCell>
                    <TableCell>
                      <div className="min-w-0">
                        {renderGroupSelector({
                          selectedIds: rule.groupIds,
                          disabledIds: getDisabledIds(rule.id),
                          onChange: (ids) => updateGroups(rule.id, ids),
                        })}
                      </div>
                    </TableCell>
                    <TableActionCell>
                      <Button variant="link" size="sm" onClick={() => removeRule(rule.id)}>删除</Button>
                    </TableActionCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            {/* 添加组织按钮 —— 拉通底部 */}
            <button
              type="button"
              onClick={addBlankGroupRow}
              disabled={!!loadingRuleId}
              className="w-full flex items-center justify-center gap-1 px-3 py-2.5 text-xs text-[var(--text-muted)] border-t border-dashed border-[var(--border)] hover:text-[var(--text-emphasis)] hover:bg-[var(--bg-grey-hover-subtle)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed disabled:bg-transparent disabled:text-[var(--text-muted)]"
            >
              <Plus className="w-3.5 h-3.5" />添加组织策略
            </button>
          </div>

          {/* 访问方式 —— 独立白底卡片，置于组织策略下方 */}
          {accessModeBlock}
        </div>
      ) : (
        /* 视图态 */
        <div className="px-5 pb-4 space-y-2">
          {/* 预设策略 —— 独立浅蓝灰底卡片；列宽与下方组织策略表格对齐：标题 120 / 权限 160 / 应用范围 */}
          <div className="rounded-[4px] bg-[var(--bg-grey-normal)] border border-[var(--border)] overflow-hidden">
            <div className="px-4 py-3">
              <div className="flex items-center">
                <div className="w-[120px] shrink-0 flex items-center text-xs text-[var(--text-title)] font-medium leading-none">预设策略</div>
                <div className="w-[160px] shrink-0 flex items-center">
                  {loadingRuleId === fallbackRule.id
                    ? renderLoading()
                    : renderPermissionIndicator(fallbackRule.value, true)}
                </div>
                <div className="flex items-center">
                  <Badge variant="outline">全部用户{groupRules.length > 0 && <span className="text-[var(--text-muted)] ml-1.5">组织策略除外</span>}</Badge>
                </div>
              </div>
            </div>
          </div>

          {/* 组织策略 —— 3 列：组织策略（序号）/ 权限 / 组织 */}
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
                      <TableCell>
                        {loadingRuleId === rule.id
                          ? renderLoading()
                          : renderPermissionIndicator(rule.value)}
                      </TableCell>
                      <TableCell className="text-xs !text-[#4B5563]">
                        <div className="min-w-0">{renderGroupBadges(rule.groupIds)}</div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          {/* 访问方式 —— 独立白底卡片，置于组织策略下方 */}
          {accessModeBlock}
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

export default PolicyEditCard;
