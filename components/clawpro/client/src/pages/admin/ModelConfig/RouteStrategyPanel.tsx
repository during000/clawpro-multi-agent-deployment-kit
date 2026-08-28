/**
* RouteStrategyPanel
* 「路由策略」Tab 主内容：列表 + 创建/编辑/删除 入口
*
* 职责：
*   - 列表 Table（策略名 / 模式 / 备选模型 / 应用范围 / 启用 / 操作）
*   - 创建 / 编辑 / 删除 / 启用切换
*   - 空态：暂未创建策略
*
* 不修改：App.tsx / AdminLayout.tsx / 路由表。完全在 ModelConfig 内部通过 Tabs 切换。
*/
import { useMemo, useState } from "react";
import { Plus, Sparkles, Info } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell, TableActionCell,
} from "@/components/ui/table";
import { SurfaceCard } from "@/components/ui/Surface";
import {
  PanelTitle, BodyText, MetaText, CardTitle, InlineNumber,
} from "@/components/ui/Typography";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import type { ModelRow } from "@/lib/modelConfigStore";
import {
  ROUTE_MODE_OPTIONS,
  useRoutingStrategiesState,
  isEligibleCandidate,
  type RoutingStrategy,
} from "@/lib/modelRoutingStore";
import { RouteStrategyDialog } from "./RouteStrategyDialog";
import { getGroupPath } from "./ScopePopover";
import type { UserGroup } from "@/pages/admin/MemberManagement/types";

export interface RouteStrategyPanelProps {
  /** 全部管控端模型（用于 RouteStrategyDialog 选择备选） */
  models: ModelRow[];
  /** 全部用户组（用于 ScopeSelect） */
  groups: UserGroup[];
}

export function RouteStrategyPanel({ models, groups }: RouteStrategyPanelProps) {
  const [strategies, setStrategies] = useRoutingStrategiesState();

  // 弹窗 state
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<RoutingStrategy | null>(null);
  // 删除确认
  const [deleting, setDeleting] = useState<RoutingStrategy | null>(null);

  // 派生：仅管控端非自定义模型可作为备选
  const eligibleCandidates = useMemo(
    () => models.filter((m) => isEligibleCandidate(m.provider)),
    [models]
  );

  // 模型 id -> ModelRow 映射（用于列里显示备选模型名）
  const modelMap = useMemo(() => {
    const m = new Map<string, ModelRow>();
    for (const row of models) m.set(row.id, row);
    return m;
  }, [models]);

  const openCreate = () => {
    setEditing(null);
    setDialogOpen(true);
  };

  const openEdit = (s: RoutingStrategy) => {
    setEditing(s);
    setDialogOpen(true);
  };

  const handleSubmit = (next: RoutingStrategy) => {
    setStrategies((prev) => {
      const exists = prev.some((p) => p.id === next.id);
      return exists
        ? prev.map((p) => (p.id === next.id ? next : p))
        : [next, ...prev];
    });
    toast.success(editing ? "路由策略已更新" : "路由策略已创建");
    setDialogOpen(false);
    setEditing(null);
  };

  const handleToggleEnabled = (id: string, value: boolean) => {
    setStrategies((prev) => prev.map((s) => (s.id === id ? { ...s, enabled: value } : s)));
    toast.success(value ? "策略已启用" : "策略已禁用");
  };

  const handleDelete = () => {
    if (!deleting) return;
    setStrategies((prev) => prev.filter((s) => s.id !== deleting.id));
    toast.success("路由策略已删除");
    setDeleting(null);
  };

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <PanelTitle>路由策略列表</PanelTitle>
          <BodyText as="p" tone="muted" className="mt-1">
            启用路由策略后，平台依据每次请求的 Prompt 自动为每次任务动态匹配最合适的模型。
          </BodyText>
        </div>
        <Button variant="claw-primary" size="claw-sm" onClick={openCreate}>
          <Plus className="w-3.5 h-3.5" />
          创建策略
        </Button>
      </div>

      {/* 列表 */}
      <SurfaceCard className="overflow-hidden">
        <Table variant="white" scrollX={1100}>
          <TableHeader>
            <TableRow>
              <TableHead fixed="left" style={{ width: 220, minWidth: 220, maxWidth: 220 }}>
                策略名称
              </TableHead>
              <TableHead className="w-[140px]">路由模式</TableHead>
              <TableHead className="w-[260px]">备选模型</TableHead>
              <TableHead className="w-[220px]">
                <div className="flex items-center gap-1">
                  应用范围
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <span className="cursor-default">
                        <Info className="w-3 h-3 text-[var(--text-weak)]" />
                      </span>
                    </TooltipTrigger>
                    <TooltipContent className="max-w-[260px]">
                      <MetaText tone="inherit">应用范围决定哪些用户组将看到该路由策略；不选则对全部用户生效。</MetaText>
                    </TooltipContent>
                  </Tooltip>
                </div>
              </TableHead>
              <TableHead className="w-[100px]">启用</TableHead>
              <TableHead fixed="right" style={{ width: 120, minWidth: 120, maxWidth: 120 }}>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {strategies.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-12">
                  <div className="flex flex-col items-center gap-2">
                    <Sparkles className="w-8 h-8 text-[var(--text-weak)]" />
                    <BodyText as="p" tone="muted">暂无路由策略</BodyText>
                    <MetaText as="p" tone="weak">点击右上角「创建策略」开始配置</MetaText>
                  </div>
                </TableCell>
              </TableRow>
            ) : (
              strategies.map((s) => {
                const modeOption = ROUTE_MODE_OPTIONS.find((o) => o.value === s.mode);
                return (
                  <TableRow key={s.id}>
                    <TableCell fixed="left" style={{ width: 220, minWidth: 220, maxWidth: 220 }}>
                      <div className="min-w-0 space-y-1">
                        <div className="flex items-center gap-1.5 min-w-0">
                          <Sparkles className="w-3.5 h-3.5 text-[#1447E6] shrink-0" />
                          <CardTitle as="p" className="truncate">{s.name}</CardTitle>
                        </div>
                        {s.description && (
                          <MetaText as="p" tone="weak" className="truncate">
                            {s.description}
                          </MetaText>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="w-[140px]">
                      <Badge variant="secondary">{modeOption?.label ?? s.mode}</Badge>
                    </TableCell>
                    <TableCell className="w-[260px]">
                      <div className="flex flex-col gap-0.5 max-w-[240px]">
                        {s.candidateModelIds.length === 0 ? (
                          <MetaText as="span" tone="muted">未选择</MetaText>
                        ) : (
                          <>
                            <BodyText as="span" className="truncate">
                              {s.candidateModelIds
                                .slice(0, 2)
                                .map((id) => {
                                  const m = modelMap.get(id);
                                  return m ? `${m.name}·${m.version}` : "已删除模型";
                                })
                                .join("、")}
                              {s.candidateModelIds.length > 2 && (
                                <span className="text-[var(--text-weak)]">
                                  {" "}+{s.candidateModelIds.length - 2}
                                </span>
                              )}
                            </BodyText>
                            <MetaText as="span" tone="weak">
                              共 <InlineNumber tone="emphasis">{s.candidateModelIds.length}</InlineNumber> 个备选
                            </MetaText>
                          </>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="w-[220px]">
                      {s.scope === "all" || s.selectedGroupIds.length === 0 ? (
                        <Badge variant="outline">全部用户</Badge>
                      ) : (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="inline-flex items-center gap-1 cursor-default">
                              <Badge variant="secondary" className="max-w-[160px]">
                                <span className="block truncate max-w-[144px]">
                                  {getGroupPath(s.selectedGroupIds[0], groups)}
                                </span>
                              </Badge>
                              {s.selectedGroupIds.length > 1 && (
                                <Badge variant="secondary">+{s.selectedGroupIds.length - 1}</Badge>
                              )}
                            </span>
                          </TooltipTrigger>
                          <TooltipContent className="max-w-[320px] whitespace-pre-line">
                            <MetaText tone="inherit">
                              {s.selectedGroupIds
                                .map((gid) => getGroupPath(gid, groups))
                                .join("\n")}
                            </MetaText>
                          </TooltipContent>
                        </Tooltip>
                      )}
                    </TableCell>
                    <TableCell className="w-[100px]">
                      <div className="flex min-h-9 items-center">
                        <Switch
                          checked={s.enabled}
                          onCheckedChange={(v) => handleToggleEnabled(s.id, v)}
                          aria-label={s.enabled ? "关闭策略" : "启用策略"}
                        />
                      </div>
                    </TableCell>
                    <TableActionCell fixed="right" style={{ width: 120, minWidth: 120, maxWidth: 120 }} actionsClassName="justify-start gap-1">
                      <Button variant="link" size="sm" onClick={() => openEdit(s)}>
                        编辑
                      </Button>
                      <Button variant="link" size="sm" onClick={() => setDeleting(s)}>
                        删除
                      </Button>
                    </TableActionCell>
                  </TableRow>
                );
              })
            )}
          </TableBody>
        </Table>
      </SurfaceCard>

      {/* 创建/编辑弹窗 */}
      <RouteStrategyDialog
        strategy={editing}
        open={dialogOpen}
        onClose={() => {
          setDialogOpen(false);
          setEditing(null);
        }}
        onSubmit={handleSubmit}
        candidates={eligibleCandidates}
        groups={groups}
      />

      {/* 删除二次确认 */}
      <AlertDialog open={!!deleting} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除路由策略「{deleting?.name}」？</AlertDialogTitle>
          </AlertDialogHeader>
          <AlertDialogDescription>
            删除后，使用该策略的用户将回退到默认路由行为。此操作不可恢复。
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>确认删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
