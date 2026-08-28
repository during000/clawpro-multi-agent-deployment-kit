/**
 * RouteStrategyDialog
 * 创建 / 编辑「模型自动路由策略」弹窗
 *
 * 字段：
 *   - 策略名称（Input）
 *   - 路由模式（Segment：平衡 / 成本优先 / 效果优先）
 *   - 备选模型（多选 Checkbox List，仅展示「管控端配置的非自定义模型」）
 *   - 应用范围（复用 ScopeSelect：全部用户 / 按用户组）
 *   - 备注（Textarea，可选）
 *   - 启用状态（Switch，默认开）
 *
 * 不修改：App.tsx / AdminLayout.tsx / 路由表。仅作为 ModelConfig 路由策略 Tab 内部组件使用。
 */
import { useEffect, useMemo, useState } from "react";
import { Sparkles, Info } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter,
} from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { PanelTitle, BodyText, MetaMedium, MetaText } from "@/components/ui/Typography";
import { ScopeSelect, type ScopeType } from "@/components/ScopeSelect";

import { CUSTOM_PROVIDER_VALUE, type ModelRow } from "@/lib/modelConfigStore";
import {
  ROUTE_MODE_OPTIONS,
  type RouteMode,
  type RouteScope,
  type RoutingStrategy,
} from "@/lib/modelRoutingStore";
import type { UserGroup } from "@/pages/admin/MemberManagement/types";

export interface RouteStrategyDialogProps {
  /** 编辑时传入；新建时传 null */
  strategy: RoutingStrategy | null;
  open: boolean;
  onClose: () => void;
  /** 提交后回调：新建返回新策略，编辑返回更新后的策略 */
  onSubmit: (next: RoutingStrategy) => void;
  /** 当前所有「管控端配置的非自定义模型」 */
  candidates: ModelRow[];
  /** 用户组（用于 ScopeSelect） */
  groups: UserGroup[];
}

export function RouteStrategyDialog({
  strategy,
  open,
  onClose,
  onSubmit,
  candidates,
  groups,
}: RouteStrategyDialogProps) {
  const isEdit = strategy !== null;

  // 表单 state
  const [name, setName] = useState("");
  const [mode, setMode] = useState<RouteMode>("balance");
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [scope, setScope] = useState<RouteScope>("all");
  const [selectedGroupIds, setSelectedGroupIds] = useState<string[]>([]);
  const [description, setDescription] = useState("");
  const [enabled, setEnabled] = useState(true);

  // 每次打开时同步
  useEffect(() => {
    if (!open) return;
    if (strategy) {
      setName(strategy.name);
      setMode(strategy.mode);
      setSelectedIds(new Set(strategy.candidateModelIds));
      setScope(strategy.scope);
      setSelectedGroupIds(strategy.selectedGroupIds);
      setDescription(strategy.description ?? "");
      setEnabled(strategy.enabled);
    } else {
      setName("");
      setMode("balance");
      setSelectedIds(new Set());
      setScope("all");
      setSelectedGroupIds([]);
      setDescription("");
      setEnabled(true);
    }
  }, [strategy, open]);

  const toggleCandidate = (id: string, checked: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  };

  // 候选模型只保留管控端非自定义模型（与 store.isEligibleCandidate 同步过滤以防外部传入脏数据）
  const eligibleCandidates = useMemo(
    () => candidates.filter((m) => m.provider !== CUSTOM_PROVIDER_VALUE),
    [candidates]
  );

  const handleSubmit = () => {
    if (!name.trim()) {
      toast.error("请填写策略名称");
      return;
    }
    if (selectedIds.size === 0) {
      toast.error("请至少选择 1 个备选模型");
      return;
    }
    if (scope === "groups" && selectedGroupIds.length === 0) {
      toast.error("选择「按用户组」时，请至少选择 1 个用户组");
      return;
    }

    const next: RoutingStrategy = {
      id: strategy?.id ?? `rs-${Date.now()}`,
      name: name.trim(),
      mode,
      candidateModelIds: Array.from(selectedIds),
      scope,
      selectedGroupIds: scope === "groups" ? selectedGroupIds : [],
      description: description.trim() || undefined,
      enabled,
      createdAt: strategy?.createdAt ?? Date.now(),
    };
    onSubmit(next);
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent size="lg" className="flex flex-col max-h-[90vh]">
        <DialogHeader className="shrink-0">
          <DialogTitle asChild>
            <PanelTitle>
              {isEdit ? "编辑路由策略" : "创建路由策略"}
            </PanelTitle>
          </DialogTitle>
          <MetaText as="p" tone="weak" className="mt-1">
            用户端将看到「智能路由（Auto）策略」选项，由系统按策略自动选择最优模型，用户无需感知细节。
          </MetaText>
        </DialogHeader>

        <DialogBody className="px-6 space-y-5">
          {/* 策略名称 */}
          <div className="space-y-1.5">
            <MetaMedium as="label" tone="secondary" className="block">
              策略名称<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
            </MetaMedium>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="例如：全公司默认智能路由"
              maxLength={32}
            />
          </div>

          {/* 路由模式 */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1">
              <MetaMedium as="label" tone="secondary" className="block">
                路由策略<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
              </MetaMedium>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="cursor-default">
                    <Info className="w-3 h-3 text-[var(--text-weak)]" />
                  </span>
                </TooltipTrigger>
                <TooltipContent className="max-w-[280px]">
                  <MetaText tone="inherit">路由策略决定系统如何在备选模型集合中自动选择；不同策略对应不同的成本/效果偏好。</MetaText>
                </TooltipContent>
              </Tooltip>
            </div>
            <div className="grid grid-cols-3 gap-2">
              {ROUTE_MODE_OPTIONS.map((opt) => {
                const active = mode === opt.value;
                return (
                  <button
                    type="button"
                    key={opt.value}
                    onClick={() => setMode(opt.value)}
                    className={[
                      "text-left px-3 py-2.5 rounded-[var(--radius)] border transition-colors",
                      active
                        ? "border-[#1447E6] bg-[#F0F4FF]"
                        : "border-[var(--cp-border)] bg-white hover:border-[#1447E6]/50",
                    ].join(" ")}
                  >
                    <div className="flex items-center gap-1.5">
                      <Sparkles className={["w-3.5 h-3.5", active ? "text-[#1447E6]" : "text-[var(--text-weak)]"].join(" ")} />
                      <BodyText as="span" className={active ? "font-semibold text-[#020617]" : "text-[var(--text-secondary)]"}>
                        {opt.label}
                      </BodyText>
                    </div>
                    <MetaText as="p" className="mt-0.5 text-xs leading-relaxed">
                      {opt.description}
                    </MetaText>
                  </button>
                );
              })}
            </div>
          </div>

          {/* 备选模型多选 */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-1">
                <MetaMedium as="label" tone="secondary" className="block">
                  备选模型<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                </MetaMedium>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="cursor-default">
                      <Info className="w-3 h-3 text-[var(--text-weak)]" />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent className="max-w-[280px]">
                    <MetaText tone="inherit">系统将在选中的备选模型中按路由策略自动选择。仅展示管控端配置的非自定义模型。</MetaText>
                  </TooltipContent>
                </Tooltip>
              </div>
              <MetaText as="span" tone="weak">
                已选 {selectedIds.size} / {eligibleCandidates.length}
              </MetaText>
            </div>
            {eligibleCandidates.length === 0 ? (
              <div className="rounded-[var(--radius)] border border-dashed border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-4 py-6 text-center">
                <MetaText tone="muted">暂无可选模型，请先在「模型列表」添加管控端模型</MetaText>
              </div>
            ) : (
              <div className="rounded-[var(--radius)] border border-[var(--cp-border)] overflow-hidden">
                <div className="max-h-56 overflow-y-auto divide-y divide-[var(--cp-border)]">
                  {eligibleCandidates.map((m) => {
                    const checked = selectedIds.has(m.id);
                    return (
                      <label
                        key={m.id}
                        className="flex items-center gap-3 px-3 py-2.5 hover:bg-[var(--bg-grey-normal)] cursor-pointer transition-colors"
                      >
                        <Checkbox
                          checked={checked}
                          onCheckedChange={(v) => toggleCandidate(m.id, !!v)}
                        />
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <BodyText as="span" className="truncate font-medium">
                              {m.name}
                            </BodyText>
                            <MetaText as="span" tone="weak" className="truncate">
                              · {m.version}
                            </MetaText>
                          </div>
                          <MetaText as="p" tone="weak" className="truncate text-xs">
                            {m.modelUrl}
                          </MetaText>
                        </div>
                      </label>
                    );
                  })}
                </div>
              </div>
            )}
          </div>

          {/* 应用范围（复用 ScopeSelect） */}
          <div className="space-y-1.5">
            <div className="flex items-center gap-1">
              <MetaMedium as="label" tone="secondary" className="block">
                应用范围<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
              </MetaMedium>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="cursor-default">
                    <Info className="w-3 h-3 text-[var(--text-weak)]" />
                  </span>
                </TooltipTrigger>
                <TooltipContent className="max-w-[280px]">
                  <MetaText tone="inherit">不同用户组可以应用不同路由策略；不选择用户组时，对全部用户生效。</MetaText>
                </TooltipContent>
              </Tooltip>
            </div>
            <ScopeSelect
              scope={scope as ScopeType}
              selectedGroupIds={selectedGroupIds}
              groups={groups}
              align="start"
              onConfirm={(s, gids) => {
                setScope(s as RouteScope);
                setSelectedGroupIds(gids);
              }}
            />
          </div>

          {/* 备注 */}
          <div className="space-y-1.5">
            <MetaMedium as="label" tone="secondary" className="block">备注</MetaMedium>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="可选：补充策略的使用场景或注意事项"
              maxLength={200}
              className="min-h-20"
            />
          </div>

          {/* 启用 */}
          <div className="rounded-[var(--radius)] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-4 py-3 flex items-center justify-between">
            <div>
              <BodyText as="p" className="font-medium">启用策略</BodyText>
              <MetaText as="p" className="mt-0.5">关闭后，用户端将不再看到该路由策略</MetaText>
            </div>
            <Switch
              checked={enabled}
              onCheckedChange={setEnabled}
              aria-label={enabled ? "关闭策略" : "启用策略"}
            />
          </div>
        </DialogBody>

        <DialogFooter className="shrink-0">
          <Button variant="claw-outline" size="claw-sm" onClick={onClose}>取消</Button>
          <Button variant="dialog-confirm" size="claw-sm" onClick={handleSubmit}>
            {isEdit ? "保存" : "创建"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
