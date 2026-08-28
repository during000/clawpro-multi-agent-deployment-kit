/**
 * UpdateRecordsDrawer - 「镜像版本记录」右侧抽屉
 *
 * 纯粹版本记录查看器：按 Agent 类型 / 镜像筛选，展示版本时间线。
 */
import { useMemo, useState, useEffect } from "react";
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerBody,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { Button } from "@/components/ui/button";
import { StatusTag, type StatusTagColor } from "@/components/ui/status-tag";
import {
  CompactText,
  CodeText,
  HelperText,
  MetaMedium,
  MetaText,
  MiniBodyText,
  PanelTitle,
} from "@/components/ui/Typography";
import { X } from "lucide-react";
import {
  buildUpdateRecords,
  formatVersion,
  type UpdateRecord,
} from "./publicImageRecords";

/** 外部传入的初始筛选条件：进入侧边栏时自动定位 */
export type DrawerInitialFilter =
  | { kind: "all"; value?: string }
  | { kind: "type"; value: string }
  | { kind: "image"; value: string };

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 进入侧边栏时的初始筛选（如：从某行的"版本更新记录"按钮触发，自动筛选到该镜像） */
  initialFilter?: DrawerInitialFilter;
}

// ─── 镜像标签颜色 ────
type ImageColorToken = { variant: StatusTagColor; dot: string };

const AGENT_TYPE_COLOR_MAP: Record<string, ImageColorToken> = {
  OpenClaw: { variant: "blue", dot: "bg-[var(--brand-blue)]" },
  HermesAgent: { variant: "teal", dot: "bg-[var(--text-success)]" },
  LightClawACE: { variant: "violet", dot: "bg-[var(--text-brand)]" },
};

function getAgentTypeColor(agentType: string): ImageColorToken {
  return AGENT_TYPE_COLOR_MAP[agentType] ?? { variant: "gray", dot: "bg-[var(--text-title)]" };
}

export default function UpdateRecordsDrawer({ open, onOpenChange, initialFilter }: Props) {
  /** 筛选粒度：按 Agent 类型 或 按具体镜像 */
  const [filter, setFilter] = useState<{ kind: "all" | "type" | "image"; value: string }>({
    kind: "all",
    value: "",
  });

  useEffect(() => {
    if (!open) return;
    if (initialFilter) {
      setFilter({ kind: initialFilter.kind, value: initialFilter.value ?? "" });
    } else {
      setFilter({ kind: "all", value: "" });
    }
  }, [open, initialFilter]);

  const records = useMemo(() => buildUpdateRecords(), []);

  const types = useMemo(() => {
    const map = new Map<string, string>();
    records.forEach((r) => map.set(r.agentType, r.agentTypeLabel));
    return Array.from(map.entries()).map(([agentType, label]) => ({ agentType, label }));
  }, [records]);

  const imagesByType = useMemo(() => {
    const grouped = new Map<string, { imageId: string; imageName: string }[]>();
    records.forEach((r) => {
      const list = grouped.get(r.agentType) ?? [];
      if (!list.some((x) => x.imageId === r.imageId)) {
        list.push({ imageId: r.imageId, imageName: r.imageName });
      }
      grouped.set(r.agentType, list);
    });
    return grouped;
  }, [records]);

  const filtered = useMemo(() => {
    let list = records;
    if (filter.kind === "type") list = list.filter((r) => r.agentType === filter.value);
    else if (filter.kind === "image") list = list.filter((r) => r.imageId === filter.value);
    return list;
  }, [records, filter]);

  const latestIdxPerImage = useMemo(() => {
    const map = new Map<string, number>();
    filtered.forEach((r, idx) => {
      if (!map.has(r.imageId)) map.set(r.imageId, idx);
    });
    return map;
  }, [filtered]);

  const activeTypeForImageFilter =
    filter.kind === "type"
      ? filter.value
      : filter.kind === "image"
        ? records.find((r) => r.imageId === filter.value)?.agentType ?? ""
        : "";

  const imageOptionsForActiveType = activeTypeForImageFilter
    ? imagesByType.get(activeTypeForImageFilter) ?? []
    : [];

  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction="right">
      <DrawerContent className="h-full max-w-[calc(100vw-24px)] rounded-none bg-background p-0 data-[vaul-drawer-direction=right]:w-[480px] data-[vaul-drawer-direction=right]:sm:max-w-none">
        <DrawerHeader className="shrink-0 gap-7 bg-background p-4 text-left">
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 flex-1 space-y-2">
              <DrawerTitle asChild>
                <PanelTitle as="h2">Agent 版本记录</PanelTitle>
              </DrawerTitle>
              <DrawerDescription asChild>
                <CompactText as="p" tone="muted" className="w-full">
                  查看各 Agent 镜像的版本演进与更新历史。
                </CompactText>
              </DrawerDescription>
            </div>
            <DrawerClose asChild>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                aria-label="关闭"
                className="h-7 w-7 shrink-0 p-0 text-[var(--text-title)] hover:text-[var(--text-emphasis)]"
              >
                <X className="size-4" />
              </Button>
            </DrawerClose>
          </div>
        </DrawerHeader>

        <DrawerBody>
          <div className="space-y-6 p-4">
            {/* 筛选标签 */}
            <div className="space-y-3">
              <div className="space-y-1.5">
                <MetaMedium as="div" tone="muted">Agent 类型</MetaMedium>
                <div className="flex flex-wrap items-center gap-1.5">
                  <button
                    type="button"
                    onClick={() => setFilter({ kind: "all", value: "" })}
                    className={`inline-flex h-7 items-center rounded-[4px] border px-2.5 transition-colors ${
                      filter.kind === "all"
                        ? "border-[var(--text-title)] bg-background text-[var(--text-title)]"
                        : "border-[var(--border)] bg-background text-[var(--text-muted)] hover:border-[var(--text-title)] hover:text-[var(--text-title)]"
                    }`}
                  >
                    <MetaMedium as="span" className="text-inherit">全部</MetaMedium>
                  </button>
                  {types.map((t) => {
                    const isActive =
                      (filter.kind === "type" && filter.value === t.agentType)
                      || (filter.kind === "image"
                        && records.find((r) => r.imageId === filter.value)?.agentType === t.agentType);
                    return (
                      <button
                        key={t.agentType}
                        type="button"
                        onClick={() => setFilter({ kind: "type", value: t.agentType })}
                        className={`inline-flex h-7 items-center rounded-[4px] border px-2.5 transition-colors ${
                          isActive
                            ? "border-[var(--text-title)] bg-background text-[var(--text-title)]"
                            : "border-[var(--border)] bg-background text-[var(--text-muted)] hover:border-[var(--text-title)] hover:text-[var(--text-title)]"
                        }`}
                      >
                        <MetaMedium as="span" className="text-inherit">{t.label}</MetaMedium>
                        <MetaText as="span" className="ml-1 tabular-nums text-inherit opacity-75">
                          {imagesByType.get(t.agentType)?.length ?? 0} 镜像
                        </MetaText>
                      </button>
                    );
                  })}
                </div>
              </div>

              {imageOptionsForActiveType.length > 0 && (
                <div className="space-y-1.5">
                  <MetaMedium as="div" tone="muted">镜像</MetaMedium>
                  <div className="flex flex-wrap items-center gap-1.5">
                    <button
                      type="button"
                      onClick={() => setFilter({ kind: "type", value: activeTypeForImageFilter })}
                      className={`inline-flex h-7 items-center rounded-[4px] border px-2.5 transition-colors ${
                        filter.kind === "type"
                          ? "border-[var(--text-title)] bg-background text-[var(--text-title)]"
                          : "border-[var(--border)] bg-background text-[var(--text-muted)] hover:border-[var(--text-title)] hover:text-[var(--text-title)]"
                      }`}
                    >
                      <MetaMedium as="span" className="text-inherit">全部镜像</MetaMedium>
                    </button>
                    {imageOptionsForActiveType.map((img) => {
                      const c = getAgentTypeColor(activeTypeForImageFilter);
                      const isActive = filter.kind === "image" && filter.value === img.imageId;
                      return (
                        <button
                          key={img.imageId}
                          type="button"
                          onClick={() => setFilter({ kind: "image", value: img.imageId })}
                          className={`inline-flex h-7 items-center gap-1.5 rounded-[4px] border px-2.5 transition-colors ${
                            isActive
                              ? "border-[var(--text-title)] bg-background text-[var(--text-title)]"
                              : "border-[var(--border)] bg-background text-[var(--text-muted)] hover:border-[var(--text-title)] hover:text-[var(--text-title)]"
                          }`}
                        >
                          <span className={`size-1.5 rounded-full ${c.dot}`} />
                          <MetaText as="span" className="max-w-[180px] truncate text-inherit">{img.imageName}</MetaText>
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>

            {/* 记录列表 */}
            <div className="space-y-3">
              <MetaText as="div" tone="muted">共 {filtered.length} 条记录</MetaText>

              {filtered.length === 0 ? (
                <div className="text-center py-12 space-y-1">
                  <HelperText>暂无更新记录</HelperText>
                  <HelperText>当前筛选条件下没有匹配到镜像更新版本。</HelperText>
                </div>
              ) : (
                <div className="relative">
                  <div aria-hidden className="absolute bottom-3 left-[11px] top-3 w-px bg-[var(--border)]" />
                  <ol className="space-y-3">
                    {filtered.map((r, idx) => {
                      const isFirstRelease = r.type === "firstRelease";
                      const isLatestOfImage = latestIdxPerImage.get(r.imageId) === idx;
                      const c = getAgentTypeColor(r.agentType);

                      return (
                        <li key={`${r.imageId}-${r.version}-${r.releaseDate}-${idx}`} className="relative pl-8">
                          <span
                            aria-hidden
                            className="absolute left-[3px] top-5 flex size-4 items-center justify-center rounded-full border border-[var(--border)] bg-background"
                          >
                            <span
                              className={`size-1.5 rounded-full ${
                                isFirstRelease
                                  ? "bg-[var(--text-brand)]"
                                  : "bg-[var(--text-weak)]"
                              }`}
                            />
                          </span>

                          <div className="rounded-[4px] border border-[var(--border)] bg-background p-5">
                            <div className="flex items-start justify-between gap-3">
                              <div className="min-w-0 space-y-3">
                                <div className="flex flex-wrap items-center gap-2">
                                  <StatusTag
                                    mode="soft"
                                    variant={c.variant}
                                    className="max-w-full justify-start rounded-full"
                                  >
                                    <span className="max-w-[180px] truncate">{r.agentTypeLabel}</span>
                                  </StatusTag>

                                  {isLatestOfImage && !isFirstRelease && (
                                    <StatusTag variant="role">最新版本</StatusTag>
                                  )}
                                </div>

                                <div className="space-y-1">
                                  <PanelTitle as="div" className="leading-6">
                                    {formatVersion(r.version)}
                                  </PanelTitle>
                                  <div className="flex flex-wrap items-center gap-2">
                                    <MetaText as="span" tone="muted">{r.imageName}</MetaText>
                                    <MetaText as="span" tone="weak">|</MetaText>
                                    <CodeText tone="muted">{r.imageId}</CodeText>
                                    <MetaText as="span" tone="weak">|</MetaText>
                                    <MetaText as="span" tone="muted">{r.releaseDate} 更新</MetaText>
                                  </div>
                                </div>
                              </div>
                            </div>

                            {r.description ? (
                              <MiniBodyText as="p" tone="secondary" className="mt-3 leading-relaxed">
                                {r.description}
                              </MiniBodyText>
                            ) : (
                              <MetaText as="div" tone="weak" className="mt-3 italic leading-relaxed">
                                {isFirstRelease ? "镜像首次上线" : `更新到 ${formatVersion(r.version)} 版本`}
                              </MetaText>
                            )}
                          </div>
                        </li>
                      );
                    })}
                  </ol>
                </div>
              )}
            </div>
          </div>
        </DrawerBody>
      </DrawerContent>
    </Drawer>
  );
}
