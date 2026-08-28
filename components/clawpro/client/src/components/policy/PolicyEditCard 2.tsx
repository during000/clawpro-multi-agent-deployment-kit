/**
 * PolicyEditCard - 策略编辑卡片（开关型）
 *
 * 卡片级编辑态：编辑期间所有规则（预设 + 组织）都可改，统一保存。
 *
 * 列顺序（v2.0）：[策略类型 120] [是否生效 160] [用户/组织 auto] [访问方式 140?] [操作 80?]
 *
 * 三种渲染状态：
 *   - disabledMessage 存在 → 禁用态（仅显示提示）
 *   - cardEditing = true   → 编辑态
 *   - cardEditing = false  → 视图态
 *
 * 组织渲染（GroupTagSelector / GroupBadges）通过 render prop 注入，与具体组织数据解耦。
 */
import React, { useState } from "react";
import { Loader2, Plus, Info } from "lucide-react";
import { toast } from "sonner";
import { Card, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableRow, TableCell, TableActionCell } from "@/components/ui/table";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";
import { StatusTag } from "@/components/ui/status-tag";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
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

  const renderFallbackEditor = () => (
    <Select value={editFallbackValue ? "on" : "off"} onValueChange={(v) => setEditFallbackValue(v === "on")}>
      <SelectTrigger className="h-7 w-[80px] text-xs">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="on">开启</SelectItem>
        <SelectItem value="off">关闭</SelectItem>
      </SelectContent>
    </Select>
  );

  // 行内 loading 文字
  const renderLoading = () => (
    <span className="inline-flex items-center gap-1.5 text-xs text-blue-500 font-medium whitespace-nowrap">
      <Loader2 className="w-3.5 h-3.5 animate-spin" />配置中，请勿关闭
    </span>
  );

  return (
    <Card className="overflow-hidden h-full py-0 gap-0">
      <div className="px-5 pt-5 pb-4">
        <div className="flex items-start gap-3">
          <div className={`shrink-0${disabledMessage ? " grayscale opacity-100" : ""}`}>{icon}</div>
          <div className="flex-1">
            <div className="flex items-center gap-2 overflow-visible">
              <h3 className="text-[14px] font-semibold text-[#020617] whitespace-nowrap">{title}</h3>
              {titleExtra}
            </div>
            <p className="text-[12px] text-[#737373] leading-relaxed mt-1">{description}</p>
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
        /* 禁用态：显示提示信息 */
        <div className="px-5 pb-4 space-y-2">
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
                  <TableCell>
                    <StatusTag mode="fill" variant="gray">关闭</StatusTag>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2 flex-wrap">
                      <Badge variant="outline">全部用户</Badge>
                      <span className="text-[13px] text-[#A3A3A3]">{disabledMessage}</span>
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
          <div className="rounded-[4px] bg-[#FAFAFA] overflow-hidden">
            <Table density="compact" autoFixedColumns={false}>
              <colgroup>
                <col style={{ width: 120 }} />
                <col style={{ width: 160 }} />
                <col />
                {accessModeRow && <col style={{ width: 140 }} />}
                <col style={{ width: 80 }} />
              </colgroup>
              <TableBody>
                <TableRow className="border-0">
                  <TableCell className="text-[13px] text-[#737373]">预设策略</TableCell>
                  <TableCell>
                    <div className="flex items-center">{renderFallbackEditor()}</div>
                  </TableCell>
                  <TableCell><Badge variant="outline">{editGroupRules.some(r => r.groupIds.length > 0) ? <><span>全部用户</span><span className="ml-1 text-[#A3A3A3] font-normal">组织策略用户除外</span></> : "全部用户"}</Badge></TableCell>
                  {accessModeRow && (
                    <TableCell>
                      <Select value={editAccessMode} onValueChange={(v: "public" | "private") => setEditAccessMode(v)}>
                        <SelectTrigger className="h-7 w-[120px] text-xs">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <div><SelectItem value="public">公网访问</SelectItem></div>
                            </TooltipTrigger>
                            <TooltipContent side="right" className="max-w-[280px] text-xs leading-relaxed">
                              <p><span className="font-semibold">公网访问：</span>用户通过公网直接访问 Agent 面板（WebUI），连接云服务器公网 IP。适用于大多数场景，<span className="text-white font-semibold">推荐选择</span>。</p>
                            </TooltipContent>
                          </Tooltip>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <div><SelectItem value="private">私网访问</SelectItem></div>
                            </TooltipTrigger>
                            <TooltipContent side="right" className="max-w-[280px] text-xs leading-relaxed">
                              <p><span className="font-semibold">私网访问：</span>用户通过同一私有网络访问 Agent 面板（WebUI），连接云服务器内网 IP。使用前需先自行将企业内网与腾讯云私有网络（VPC）打通，并在「网络管理」中将云服务器绑定至该 VPC。配置完成后，企业用户可通过企业内网访问面板，但无法通过公网访问。</p>
                            </TooltipContent>
                          </Tooltip>
                        </SelectContent>
                      </Select>
                    </TableCell>
                  )}
                  <TableActionCell />
                </TableRow>
              </TableBody>
            </Table>
          </div>

          {/* 组织策略 + 添加按钮 */}
          <div className="rounded-[4px] bg-[#FAFAFA] overflow-hidden">
            <Table density="compact" autoFixedColumns={false}>
              <colgroup>
                <col style={{ width: 120 }} />
                <col style={{ width: 160 }} />
                <col />
                {accessModeRow && <col style={{ width: 140 }} />}
                <col style={{ width: 80 }} />
              </colgroup>
              <TableBody>
                {editGroupRules.map((rule, idx) => (
                  <TableRow key={rule.id} className={idx < editGroupRules.length - 1 ? "border-b border-[#EFEFEF]" : "border-0"}>
                    <TableCell className="text-[13px] text-[#737373]">组织策略{idx + 1}</TableCell>
                    <TableCell>
                      <div className="flex items-center">
                        <StatusTag mode="fill" variant={editGroupRuleValue ? "green" : "gray"}>{editGroupRuleValue ? "开启" : "关闭"}</StatusTag>
                      </div>
                    </TableCell>
                    <TableCell>
                      {renderGroupSelector({
                        selectedIds: rule.groupIds,
                        disabledIds: getDisabledIds(rule.id),
                        onChange: (ids) => updateGroups(rule.id, ids),
                      })}
                    </TableCell>
                    {accessModeRow && (
                      <TableCell className="text-[13px] text-[#020617]">
                        {editAccessMode === "public" ? "公网访问" : "私网访问"}
                      </TableCell>
                    )}
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
              className="w-full flex items-center justify-center gap-1 px-3 py-2.5 text-[13px] text-[#737373] border-t border-dashed border-[#D4D4D4] hover:text-[#020617] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
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
                {accessModeRow && <col style={{ width: 140 }} />}
              </colgroup>
              <TableBody>
                <TableRow className="hover:bg-transparent border-0">
                  <TableCell className="text-[13px] text-[#737373]">预设策略</TableCell>
                  <TableCell>
                    {loadingRuleId === fallbackRule.id
                      ? renderLoading()
                      : <StatusTag mode="fill" variant={fallbackRule.value ? "green" : "gray"}>{fallbackRule.value ? "开启" : "关闭"}</StatusTag>}
                  </TableCell>
                  <TableCell><Badge variant="outline">{groupRules.length > 0 ? <><span>全部用户</span><span className="ml-1 text-[#A3A3A3] font-normal">组织策略用户除外</span></> : "全部用户"}</Badge></TableCell>
                  {accessModeRow && (
                    <TableCell>
                      <span className="inline-flex items-center gap-1.5 text-[13px] text-[#020617]">
                        {accessModeRow.mode === "public" ? "公网访问" : "私网访问"}
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="cursor-default"><Info className="w-3.5 h-3.5 text-[#A3A3A3]" /></span>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-[280px] text-xs leading-relaxed">
                            <p className="mb-1.5"><span className="font-semibold">公网访问：</span>用户通过公网直接访问 Agent 面板（WebUI），连接云服务器公网 IP。适用于大多数场景，<span className="text-white font-semibold">推荐选择</span>。</p>
                            <p><span className="font-semibold">私网访问：</span>用户通过同一私有网络访问 Agent 面板（WebUI），连接云服务器内网 IP。使用前需先自行将企业内网与腾讯云私有网络（VPC）打通，并在「网络管理」中将云服务器绑定至该 VPC。配置完成后，企业用户可通过企业内网访问面板，但无法通过公网访问。</p>
                          </TooltipContent>
                        </Tooltip>
                      </span>
                    </TableCell>
                  )}
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
                  {accessModeRow && <col style={{ width: 140 }} />}
                </colgroup>
                <TableBody>
                  {groupRules.map((rule, idx) => (
                    <TableRow key={rule.id} className={`hover:bg-transparent ${idx < groupRules.length - 1 ? "border-b border-[#EFEFEF]" : "border-0"}`}>
                      <TableCell className="text-[13px] text-[#737373]">组织策略{idx + 1}</TableCell>
                      <TableCell>
                        {loadingRuleId === rule.id
                          ? renderLoading()
                          : <StatusTag mode="fill" variant={rule.value ? "green" : "gray"}>{rule.value ? "开启" : "关闭"}</StatusTag>}
                      </TableCell>
                      <TableCell>{renderGroupBadges(rule.groupIds)}</TableCell>
                      {accessModeRow && (
                        <TableCell className="text-[13px] text-[#020617]">
                          {accessModeRow.mode === "public" ? "公网访问" : "私网访问"}
                        </TableCell>
                      )}
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

export default PolicyEditCard;
