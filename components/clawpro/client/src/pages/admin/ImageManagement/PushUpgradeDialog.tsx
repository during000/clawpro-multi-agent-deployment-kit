/**
 * PushUpgradeDialog - 推送更新提醒弹窗（支持 Agent 类型多选）
 *
 * 心智：
 *   推送是声音放大 = "管理员让用户知道：你的实例和当前生效镜像版本不一致"
 *   - 不需要选目标版本：版本由"当前启用镜像"决定
 *   - 不需要选实例范围：默认就是该类型下所有"实例版本 ≠ 启用版本"的用户
 *   - 弹窗里只让用户选「Agent 类型」（支持多选）
 *
 * 可推送条件：
 *   - 该类型已启用某镜像（启用版本存在）
 *   - 实际有 ≥ 1 个实例版本 < 启用版本（mock：演示时全部类型都可选）
 */
import { useEffect, useMemo, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import { Alert, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import {
  BodyMedium,
  HelperText,
  MetaMedium,
  MetaText,
  MiniBodyText,
} from "@/components/ui/Typography";
import { Megaphone, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { setActivePush, listActivePushes, type ActivePush } from "@/lib/upgradePushStore";
import { cn } from "@/lib/utils";
import type { ImageRow } from "./deriveAgentTypeView";

// ─── 输入信息 ─────────────────────────────────────────────
/** 一个可被推送的 Agent 类型条目 */
export interface PushableAgentType {
  /** Agent 类型 ID（OpenClaw / HermesAgent / ... / custom-xxx） */
  agentType: string;
  /** 展示名 */
  agentTypeLabel: string;
  /** 当前启用版本（必有，否则不会出现在列表） */
  enabledVersion: string;
  /** 当前启用镜像 */
  enabledImage: ImageRow;
  /** 启用镜像名（人类可读） */
  imageName: string;
  /** 镜像来源 */
  imageSource: "public" | "custom";
  /** 旧版本实例数（mock 演示用） */
  outdatedInstanceCount: number;
  /** 是否所有实例都已是最新版（true 时不可推送） */
  allUpToDate: boolean;
}

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 当前所有可推送的 Agent 类型 */
  pushable: PushableAgentType[];
  /** 默认选中的 agent 类型（从某条更新记录入口触发时传） */
  defaultAgentType?: string;
  /** 推送人（mock） */
  pushedBy?: string;
  /** 点击「查看全部更新记录」link 时触发 */
  onViewAllRecords?: () => void;
}

export default function PushUpgradeDialog({
  open,
  onOpenChange,
  pushable,
  defaultAgentType,
  pushedBy = "alice@acompany.com",
  onViewAllRecords,
}: Props) {
  // 活跃推送列表（用于在每张卡片上显示「正在提醒」状态）
  const [activePushes, setActivePushes] = useState<ActivePush[]>(() => listActivePushes());
  useEffect(() => {
    if (!open) return;
    setActivePushes(listActivePushes());
    const onChange = () => setActivePushes(listActivePushes());
    window.addEventListener("upgrade-push-changed", onChange);
    return () => window.removeEventListener("upgrade-push-changed", onChange);
  }, [open]);

  /** agentType -> 当前活跃推送（仅当推送版本 = 当前启用版本时生效，旧推送由 store 自动清理） */
  const activePushByType = useMemo(() => {
    const map = new Map<string, ActivePush>();
    pushable.forEach((p) => {
      const push = activePushes.find(
        (ap) => ap.agentType === p.agentType && ap.version === p.enabledVersion,
      );
      if (push) map.set(p.agentType, push);
    });
    return map;
  }, [pushable, activePushes]);

  // 仅"已启用 + 有旧版本实例 + 未在推送当前版本"才可选
  const selectable = useMemo(
    () => pushable.filter((p) => !p.allUpToDate && !activePushByType.has(p.agentType)),
    [pushable, activePushByType],
  );

  // 多选：保存被选中的 agentType 列表
  const [selectedTypes, setSelectedTypes] = useState<string[]>([]);

  useEffect(() => {
    if (!open) return;
    // 优先用默认值；否则默认全选
    if (defaultAgentType && selectable.some((p) => p.agentType === defaultAgentType)) {
      setSelectedTypes([defaultAgentType]);
    } else if (selectable.length > 0) {
      setSelectedTypes(selectable.map((p) => p.agentType));
    } else {
      setSelectedTypes([]);
    }
  }, [open, defaultAgentType, selectable]);

  const selectedItems = useMemo(
    () =>
      pushable.filter(
        (p) =>
          selectedTypes.includes(p.agentType) &&
          !p.allUpToDate &&
          !activePushByType.has(p.agentType),
      ),
    [pushable, selectedTypes, activePushByType],
  );

  const totalOutdated = useMemo(
    () => selectedItems.reduce((sum, p) => sum + p.outdatedInstanceCount, 0),
    [selectedItems],
  );

  const allSelectableChecked =
    selectable.length > 0 && selectable.every((p) => selectedTypes.includes(p.agentType));
  const someSelectableChecked = selectable.some((p) => selectedTypes.includes(p.agentType));

  const toggleOne = (agentType: string, checked: boolean) => {
    setSelectedTypes((prev) =>
      checked ? Array.from(new Set([...prev, agentType])) : prev.filter((t) => t !== agentType),
    );
  };

  const toggleAll = (checked: boolean) => {
    if (checked) {
      setSelectedTypes(selectable.map((p) => p.agentType));
    } else {
      setSelectedTypes([]);
    }
  };

  const canPush = selectedItems.length > 0;

  const handleConfirm = () => {
    if (selectedItems.length === 0) {
      toast.error("请至少选择一个 Agent 类型");
      return;
    }
    const ts = nowStr();
    selectedItems.forEach((item) => {
      const push: ActivePush = {
        agentType: item.agentType,
        agentTypeLabel: item.agentTypeLabel,
        version: item.enabledVersion,
        imageName: item.imageName,
        imageSource: item.imageSource,
        pushedAt: ts,
        pushedBy,
        message: `管理员推荐更新到 v${item.enabledVersion}`,
      };
      setActivePush(push);
    });
    const labels = selectedItems.map((p) => p.agentTypeLabel).join("、");
    toast.success(
      `已向「${labels}」共 ${totalOutdated} 个旧版本 Agent 推送更新提醒`,
    );
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader>
          <DialogTitle>
            推送更新提醒
          </DialogTitle>
          <DialogDescription>
            向使用某 Agent 类型的用户推送更新提醒，建议更新到当前启用的镜像版本
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-1">
          {/* 1. Agent 类型多选 */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <MetaMedium as="label" tone="secondary">
                Agent 类型<MetaMedium as="span" tone="danger" className="ml-1">*</MetaMedium>
                <MetaText tone="weak" className="ml-1">（可多选）</MetaText>
              </MetaMedium>
              {selectable.length > 0 && (
                <button
                  type="button"
                  onClick={() => toggleAll(!allSelectableChecked)}
                  className="hover:opacity-80"
                >
                  <MetaText tone="brand" className="text-[11px]">
                    {allSelectableChecked ? "取消全选" : "全选"}
                  </MetaText>
                </button>
              )}
            </div>
            {pushable.length === 0 ? (
              <HelperText className="bg-[var(--bg-subtle)] rounded-[4px] px-3 py-3 text-center">
                暂无已启用的 Agent 类型，请先到表格中启用一个镜像
              </HelperText>
            ) : (
              <div className="space-y-2 max-h-[320px] overflow-y-auto">
                {pushable.map((p) => {
                  const checked = selectedTypes.includes(p.agentType);
                  const activePush = activePushByType.get(p.agentType);
                  const isActivePushing = !!activePush;
                  // 已是最新版 或 已经在推送当前版本 → 不可选
                  const disabled = p.allUpToDate || isActivePushing;
                  const id = `push-type-${p.agentType}`;
                  return (
                    <label
                      key={p.agentType}
                      htmlFor={id}
                      className={cn(
                        "flex items-start gap-2.5 rounded-[4px] border px-3 py-3 transition-colors",
                        "border-[var(--border)] bg-[var(--popover)]",
                        !checked && !disabled && "hover:border-[var(--text-brand)] cursor-pointer",
                        checked && "border-[var(--text-brand)] bg-[var(--accent)]",
                        p.allUpToDate && "cursor-not-allowed opacity-60",
                        isActivePushing && "cursor-not-allowed",
                      )}
                    >
                      <Checkbox
                        id={id}
                        checked={checked}
                        disabled={disabled}
                        onCheckedChange={(c) => toggleOne(p.agentType, c === true)}
                        className="mt-0.5 shrink-0"
                      />
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 min-w-0">
                          {p.imageSource === "custom" && (
                            <Sparkles className="w-3 h-3 text-[var(--decoration-purple)] shrink-0" />
                          )}
                          <BodyMedium tone="primary" className="truncate leading-snug">
                            {p.agentTypeLabel}
                          </BodyMedium>
                          <MetaText tone="weak" className="text-[11px] font-mono tabular-nums shrink-0">
                            v{p.enabledVersion}
                          </MetaText>
                          {isActivePushing ? (
                            <MetaText tone="brand" className="text-[11px] inline-flex items-center gap-1 shrink-0 ml-auto">
                              <Megaphone className="w-2.5 h-2.5" />
                              正在提醒用户更新
                            </MetaText>
                          ) : p.allUpToDate ? (
                            <MetaText tone="weak" className="text-[11px] shrink-0 ml-auto">
                              已是最新版
                            </MetaText>
                          ) : (
                            <MetaText tone="muted" className="text-[11px] shrink-0 ml-auto">
                              {p.outdatedInstanceCount} 个旧版本
                            </MetaText>
                          )}
                        </div>
                        <HelperText className="leading-relaxed mt-0.5 truncate">
                          {p.imageName}
                        </HelperText>
                      </div>
                    </label>
                  );
                })}
              </div>
            )}
          </div>

          {/* 2. 信息提示（多选汇总） */}
          {selectedItems.length > 0 && (
            <Alert variant="info" className="items-start">
              <AlertInfoIcon />
              <AlertDescription>
                <MiniBodyText as="div" tone="inherit" className="leading-relaxed">
                  推送后，已选
                  <MetaMedium as="span" tone="inherit" className="mx-1 font-semibold tabular-nums">
                    {selectedItems.length}
                  </MetaMedium>
                  个 Agent 类型下共
                  <MetaMedium as="span" tone="inherit" className="mx-1 font-semibold tabular-nums">
                    {totalOutdated}
                  </MetaMedium>
                  个旧版本 Agent，将在用户端收到更新提醒。
                </MiniBodyText>
                <ul className="mt-1.5 space-y-0.5">
                  {selectedItems.map((item) => (
                    <li key={item.agentType} className="flex items-center gap-1">
                      <MetaMedium tone="inherit" className="text-[11px]">{item.agentTypeLabel}</MetaMedium>
                      <MetaText tone="inherit" className="text-[11px] font-mono tabular-nums">→ v{item.enabledVersion}</MetaText>
                      <MetaText tone="inherit" className="text-[11px] opacity-80">· {item.outdatedInstanceCount} 个</MetaText>
                    </li>
                  ))}
                </ul>
              </AlertDescription>
            </Alert>
          )}

          {/* 3. 全部已是最新版提示 */}
          {selectable.length === 0 && pushable.length > 0 && (
            <Alert>
              <AlertInfoIcon />
              <AlertDescription>
                所有 Agent 类型下的实例都已是最新版，无需推送
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button variant="claw-outline" size="claw-sm" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          {onViewAllRecords && (
            <Button variant="claw-outline" size="claw-sm" onClick={onViewAllRecords}>
              查看全部更新记录
            </Button>
          )}
          <Button
            variant="dialog-confirm"
            size="claw-sm"
            onClick={handleConfirm}
            disabled={!canPush}
          >
            确认推送
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function nowStr(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
