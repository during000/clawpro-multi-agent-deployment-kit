/**
 * SwitchImageDialog - 切换镜像弹窗
 *
 * 交互：
 *   1. 点击主表「切换镜像」按钮 → 打开本弹窗
 *   2. 弹窗内通过单选选择目标镜像（仅更新本地待提交状态，不立即生效）
 *   3. 点「确认切换」→ 调用 onConfirm 完成切换并关闭弹窗
 *   4. 点「取消」/ 关闭按钮 / 点击遮罩 → 不做任何修改
 *
 * 使用项目最新规范：
 *   - DialogContent size="xl"（920px，对应「含多列数据表格/Tabs+列表管理」档位）
 *   - DialogHeader / DialogBody / DialogFooter 标准结构
 *   - Footer 按钮：取消（outline，左）+ 确认（default，右）
 */
import { useEffect, useMemo, useRef, useState } from "react";
import {
  CircleAlert,
  History,
  Plus,
  RefreshCw,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { StatusTag } from "@/components/ui/status-tag";
import { Badge } from "@/components/ui/badge";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableActionCell,
} from "@/components/ui/table";
import { SurfaceCard } from "@/components/ui/Surface";
import {
  BodyMedium,
  BodyText,
  CodeText,
  MetaMedium,
  MetaText,
} from "@/components/ui/Typography";
import { ImageStatusBadge } from "./ImageStatusBadge";
import { getCurrentVersion } from "./publicImageRecords";
import type { AgentTypeView, ViewImage } from "./deriveAgentTypeView";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 当前 Agent 类型展示名（如「OpenClaw」），副标题展示 */
  agentTypeLabel: string;
  view: AgentTypeView;
  isCustomAgentType: boolean;

  /**
   * 外部指定初始选中镜像 ID（如「克隆 Agent 为镜像」跳转 ?enableImage= 的启用引导），
   * 弹窗打开时预选该镜像；缺省为当前生效镜像。
   */
  initialPendingId?: string;

  /** 用户点击「确认切换」时调用，传出最终选定的 imageId */
  onConfirm: (imageId: string) => void;

  /** 弹窗内仍可触发的辅助操作（不会关闭弹窗） */
  onEditImage: (imageId: string) => void;
  onDeleteImage: (imageId: string) => void;
  onViewPublicHistory: (imageId: string) => void;
  onImportCustom: () => void;
}

export default function SwitchImageDialog({
  open,
  onOpenChange,
  agentTypeLabel,
  view,
  isCustomAgentType,
  initialPendingId,
  onConfirm,
  onEditImage,
  onDeleteImage,
  onViewPublicHistory,
  onImportCustom,
}: Props) {
  // ─── 当前生效镜像 ID（短路查找，无须合并数组） ─────────────────────
  const effectiveId = useMemo(
    () =>
      view.publicRow?.allImages.find((i) => i.isEffective)?.id ??
      view.customRow.allImages.find((i) => i.isEffective)?.id ??
      "",
    [view.publicRow, view.customRow]
  );

  // 当前生效所在 Tab（仅系统预设类型用到）
  const enabledTab: "public" | "custom" =
    view.enabled.source === "custom" ? "custom" : "public";

  const [tab, setTab] = useState<"public" | "custom">(enabledTab);

  // ─── 待提交选中（不立即生效，确认后才切换） ─────────────────────
  const [pendingId, setPendingId] = useState<string>(effectiveId);

  // 仅在弹窗"由关到开"那一刻重置选中态与 Tab：
  // 防止外部 view 在弹窗开启期间变化（例如其它行启停镜像）时，把用户的待选改动覆盖掉。
  // initialPendingId 存在时（启用引导场景）优先预选该镜像，并定位到其所在 Tab。
  const prevOpen = useRef(open);
  useEffect(() => {
    if (open && !prevOpen.current) {
      setPendingId(initialPendingId || effectiveId);
      const isCustomPick =
        !!initialPendingId &&
        !!view.customRow?.allImages?.some((i) => i.id === initialPendingId);
      setTab(isCustomPick ? "custom" : enabledTab);
    }
    prevOpen.current = open;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const canConfirm = !!pendingId && pendingId !== effectiveId;

  const handleConfirm = () => {
    if (!canConfirm) return;
    onConfirm(pendingId);
    onOpenChange(false);
  };

  // 当前生效镜像对象（用于信息卡片）
  const effectiveImage = useMemo(() => {
    if (!effectiveId) return null;
    return (
      view.publicRow?.allImages.find((i) => i.id === effectiveId) ??
      view.customRow.allImages.find((i) => i.id === effectiveId) ??
      null
    );
  }, [effectiveId, view.publicRow, view.customRow]);

  // 所有镜像（含当前生效镜像，切换弹窗内全部展示）
  const publicImages = useMemo(
    () => (view.publicRow?.allImages ?? []),
    [view.publicRow]
  );
  const customImages = useMemo(
    () => view.customRow.allImages,
    [view.customRow]
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="xl"
        style={{
          maxHeight: "min(90vh, 780px)",
          display: "flex",
          flexDirection: "column",
        }}
      >
        <DialogHeader>
          <DialogTitle>
            切换镜像
            {agentTypeLabel && (
              <MetaText tone="weak" className="ml-2">
                · {agentTypeLabel}
              </MetaText>
            )}
          </DialogTitle>
        </DialogHeader>

        <DialogBody className="px-6 flex-1">
          <div className="space-y-3">
            {/* 当前用户可见镜像 —— 始终展示（无论是否已选择 / 是否自定义类型） */}
            <div className="space-y-2">
              <MetaMedium as="div">当前用户可见镜像</MetaMedium>
              <SurfaceCard className="overflow-hidden">
                <Table density="compact" autoFixedColumns={false} className="table-fixed">
                  <TableHeader>
                    <TableRow>
                      <TableHead style={{ width: 40 }} />
                      <TableHead style={{ width: 140 }}>Agent 版本</TableHead>
                      <TableHead style={{ width: 100 }}>镜像来源</TableHead>
                      <TableHead style={{ width: 240 }}>镜像</TableHead>
                      <TableHead style={{ width: 100 }}>镜像状态</TableHead>
                      <TableHead style={{ width: 110 }}>创建时间</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {effectiveImage ? (
                      <TableRow>
                        <TableCell style={{ width: 40 }} />
                        {/* Agent 版本：版本号 + History */}
                        <TableCell>
                          <div className="flex items-center gap-1.5">
                            {effectiveImage.source === "public" ? (
                              <span className="inline-flex items-center gap-1">
                                <span className="text-[10px] text-[var(--text-muted)]">最新</span>
                                <BodyMedium tone="primary" className="tabular-nums">
                                  v{getCurrentVersion(effectiveImage.id) ?? effectiveImage.agentVersion}
                                </BodyMedium>
                              </span>
                            ) : (
                              <BodyMedium tone="primary" className="tabular-nums whitespace-nowrap">
                                v{effectiveImage.agentVersion}
                              </BodyMedium>
                            )}
                            {effectiveImage.source === "public" && (() => {
                              const hasHistory = getCurrentVersion(effectiveImage.id) !== null;
                              return (
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <button
                                      type="button"
                                      disabled={!hasHistory}
                                      onClick={() => hasHistory && onViewPublicHistory(effectiveImage.id)}
                                      className="cursor-pointer inline-flex items-center justify-center w-5 h-5 rounded-[4px] text-[var(--text-weak)] hover:text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)] transition-colors disabled:cursor-not-allowed disabled:text-[var(--text-weak)] disabled:hover:bg-transparent"
                                      aria-label="版本更新记录"
                                    >
                                      <History className="w-3.5 h-3.5" />
                                    </button>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    {hasHistory ? "版本更新记录" : "暂无版本更新记录"}
                                  </TooltipContent>
                                </Tooltip>
                              );
                            })()}
                          </div>
                        </TableCell>

                        {/* 镜像来源 */}
                        <TableCell>
                          <StatusTag mode="fill" variant={effectiveImage.source === "public" ? "blue" : "gray"}>
                            {effectiveImage.source === "public" ? "公共" : "自定义"}
                          </StatusTag>
                        </TableCell>

                        {/* 镜像：名称 / ID */}
                        <TableCell>
                          <div className="flex items-center gap-1.5 min-w-0">
                            <BodyMedium tone="primary" className="truncate max-w-[240px]">
                              {effectiveImage.name}
                            </BodyMedium>
                          </div>
                          <CodeText tone="weak" as="div" className="mt-1 truncate">
                            {effectiveImage.id}
                          </CodeText>
                        </TableCell>

                        {/* 镜像状态 */}
                        <TableCell>
                          <ImageStatusBadge status={effectiveImage.status || "available"} />
                        </TableCell>

                        {/* 创建时间 */}
                        <TableCell>
                          <BodyText as="span" tone="secondary" className="tabular-nums">
                            {effectiveImage.createTime ? effectiveImage.createTime.split(" ")[0] : "—"}
                          </BodyText>
                        </TableCell>
                      </TableRow>
                    ) : (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center py-6">
                          <BodyText tone="weak" as="span">尚未选择镜像</BodyText>
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </SurfaceCard>
            </div>

            {isCustomAgentType ? (
              // 自定义类型：无 Tab，顶部直接放"导入自定义镜像"按钮
              <>
                <div className="flex items-center justify-between">
                  <MetaMedium as="div">可选自定义镜像</MetaMedium>
                  <Button variant="outline" size="sm" onClick={onImportCustom}>
                    <Plus className="w-3.5 h-3.5 mr-1" />
                    导入自定义镜像
                  </Button>
                </div>
                <CustomList
                  row={view.customRow}
                  pendingId={pendingId}
                  onSelect={setPendingId}
                  onEditImage={onEditImage}
                  onDeleteImage={onDeleteImage}
                  onViewHistory={onViewPublicHistory}
                />
              </>
            ) : (
              <>
              <div className="flex items-center gap-3">
                  <SegmentGroup>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <SegmentOption
                        active={tab === "public"}
                        onClick={() => setTab("public")}
                      >
                        公共镜像
                        <MetaText tone="weak" className="ml-1.5">
                          ({publicImages.length})
                        </MetaText>
                      </SegmentOption>
                    </TooltipTrigger>
                    <TooltipContent>
                      <MetaText tone="inherit">由腾讯云持续维护更新，自动跟随官方版本</MetaText>
                    </TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <SegmentOption
                        active={tab === "custom"}
                        onClick={() => setTab("custom")}
                      >
                        自定义镜像
                        <MetaText tone="weak" className="ml-1.5">
                          ({customImages.length})
                        </MetaText>
                      </SegmentOption>
                    </TooltipTrigger>
                    <TooltipContent>
                      <MetaText tone="inherit">由企业自行制作和维护</MetaText>
                    </TooltipContent>
                  </Tooltip>
                </SegmentGroup>
                <div className="ml-auto flex items-center gap-2">
                  {tab === "custom" && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={onImportCustom}
                    >
                      <Plus className="w-3.5 h-3.5 mr-1" />
                      导入自定义镜像
                    </Button>
                  )}
                  <PublicRefreshButton />
                </div>
              </div>

              {tab === "public" ? (
                publicImages.length > 0 ? (
                  <ImageList
                    images={publicImages}
                    pendingId={pendingId}
                    onSelect={setPendingId}
                    onViewHistory={onViewPublicHistory}
                  />
                ) : (
                  <EmptyHint text="暂无公共镜像" />
                )
              ) : (
                <CustomList
                  row={view.customRow}
                  pendingId={pendingId}
                  onSelect={setPendingId}
                  onEditImage={onEditImage}
                  onDeleteImage={onDeleteImage}
                  onViewHistory={onViewPublicHistory}
                />
              )}
              </>
            )}
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button variant="dialog-confirm" disabled={!canConfirm} onClick={handleConfirm}>
            确认切换
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── 自定义镜像列表 ─────────────────────────────────────────────────────
function CustomList({
  row,
  pendingId,
  onSelect,
  onEditImage,
  onDeleteImage,
  onViewHistory,
}: {
  row: AgentTypeView["customRow"];
  pendingId: string;
  onSelect: (imageId: string) => void;
  onEditImage: (imageId: string) => void;
  onDeleteImage: (imageId: string) => void;
  onViewHistory?: (imageId: string) => void;
}) {
  const all = row.allImages
    .filter((i) => i.source === "custom")
    .sort((a, b) => b.createTime.localeCompare(a.createTime));

  return (
    <div className="space-y-2">
      {all.length > 0 ? (
        <ImageList
          images={all}
          allowSelectIfMissingVersion={false}
          pendingId={pendingId}
          onSelect={onSelect}
          onViewHistory={onViewHistory}
          renderActions={(img) => {
            return (
              <>
                <Button
                  variant="link"
                  size="sm"
                  onClick={() => onEditImage(img.id)}
                >
                  编辑
                </Button>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex">
                      <Button
                        variant="link"
                        size="sm"
                        disabled={img.isEffective}
                        onClick={() => onDeleteImage(img.id)}
                      >
                        删除
                      </Button>
                    </span>
                  </TooltipTrigger>
                  {img.isEffective && (
                    <TooltipContent side="left" className="max-w-[220px]">
                      用户可见的镜像不可删除
                    </TooltipContent>
                  )}
                </Tooltip>
              </>
            );
          }}
        />
      ) : (
        <EmptyHint text="尚未导入任何自定义镜像" />
      )}
    </div>
  );
}

// ─── 通用镜像列表（弹窗内单选，使用规范 Table + RadioGroupItem） ─────
function ImageList({
  images,
  allowSelectIfMissingVersion = true,
  pendingId,
  onSelect,
  onViewHistory,
  renderActions,
}: {
  images: ViewImage[];
  allowSelectIfMissingVersion?: boolean;
  pendingId: string;
  onSelect: (imageId: string) => void;
  onViewHistory?: (imageId: string) => void;
  renderActions?: (img: ViewImage) => React.ReactNode;
}) {
  return (
    <RadioGroup value={pendingId} onValueChange={onSelect}>
      <SurfaceCard className="overflow-hidden">
      <Table density="compact" autoFixedColumns={false} className="table-fixed">
        <TableHeader>
          <TableRow>
            <TableHead style={{ width: 40 }} />
            <TableHead style={{ width: 140 }}>Agent 版本</TableHead>
            <TableHead style={{ width: 100 }}>镜像来源</TableHead>
            <TableHead style={{ width: 240 }}>镜像</TableHead>
            <TableHead style={{ width: 100 }}>镜像状态</TableHead>
            <TableHead style={{ width: 110 }}>创建时间</TableHead>
            {renderActions && <TableHead style={{ width: 120 }}>操作</TableHead>}
          </TableRow>
        </TableHeader>
        <TableBody>
          {images.map((img) => {
            const checked = pendingId === img.id;
            const missingVersion = !img.agentVersion?.trim();
            const selectable = allowSelectIfMissingVersion || !missingVersion;
            const imgType = img.source === "public" ? "公共" : "自定义";
            return (
              <TableRow
                key={img.id}
                data-state={checked ? "selected" : undefined}
                className={selectable ? "cursor-pointer" : ""}
                onClick={() => selectable && onSelect(img.id)}
              >
                {/* 规范 Radio */}
                <TableCell className="w-[40px]">
                  <RadioGroupItem
                    value={img.id}
                    disabled={!selectable}
                    aria-label={`选择镜像 ${img.name}`}
                  />
                </TableCell>

                {/* Agent 版本 */}
                <TableCell>
                  <div className="flex items-center gap-1.5">
                    {missingVersion ? (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span
                            className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded whitespace-nowrap cursor-help bg-[var(--alert-warning-bg)] text-[var(--text-warning)] border border-[var(--alert-warning-border)]"
                          >
                            <CircleAlert className="w-3 h-3" />
                            <MetaMedium tone="inherit">缺版本号</MetaMedium>
                          </span>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-[240px]">
                          缺少版本号，无法对用户可见，请编辑后补齐
                        </TooltipContent>
                      </Tooltip>
                    ) : img.source === "public" ? (
                      <span className="inline-flex items-center gap-1">
                        <span className="text-[10px] text-[var(--text-muted)]">最新</span>
                        <BodyMedium tone="primary" className="tabular-nums">
                          v{getCurrentVersion(img.id) ?? img.agentVersion}
                        </BodyMedium>
                      </span>
                    ) : (
                      <BodyMedium tone="primary" className="tabular-nums whitespace-nowrap">
                        v{img.agentVersion}
                      </BodyMedium>
                    )}
                    {onViewHistory && (() => {
                      const hasHistory = img.source === "public" && getCurrentVersion(img.id) !== null;
                      return (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button
                              type="button"
                              disabled={!hasHistory}
                              onClick={(e) => {
                                e.stopPropagation();
                                if (hasHistory) onViewHistory(img.id);
                              }}
                              className="cursor-pointer inline-flex items-center justify-center w-5 h-5 rounded-[4px] text-[var(--text-weak)] hover:text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)] transition-colors disabled:cursor-not-allowed disabled:text-[var(--text-weak)] disabled:hover:bg-transparent"
                              aria-label="版本更新记录"
                            >
                              <History className="w-3.5 h-3.5" />
                            </button>
                          </TooltipTrigger>
                          <TooltipContent>
                            {hasHistory ? "版本更新记录" : "暂无版本更新记录"}
                          </TooltipContent>
                        </Tooltip>
                      );
                    })()}
                  </div>
                </TableCell>

                {/* 镜像来源（公共/自定义） */}
                <TableCell>
                  <StatusTag mode="fill" variant={img.source === "public" ? "blue" : "gray"}>
                    {imgType}
                  </StatusTag>
                </TableCell>

                {/* 镜像：名称 + ID */}
                <TableCell>
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <BodyMedium tone="primary" className="truncate max-w-[240px]">
                      {img.name}
                    </BodyMedium>
                    {img.isEffective && (
                      <StatusTag variant="green">
                        当前生效
                      </StatusTag>
                    )}
                  </div>
                  <div className="flex items-center gap-2 mt-1 flex-wrap">
                    <CodeText tone="weak" className="truncate">{img.id}</CodeText>
                  </div>
                </TableCell>

                {/* 镜像状态 */}
                <TableCell>
                  <ImageStatusBadge status={img.status} />
                </TableCell>

                {/* 创建时间 */}
                <TableCell className="tabular-nums">
                  <BodyText tone="secondary" as="span" className="tabular-nums">
                    {img.createTime ? img.createTime.split(" ")[0] : "—"}
                  </BodyText>
                </TableCell>

                {/* 操作列 */}
                {renderActions && (
                  <TableActionCell
                    onClick={(e) => e.stopPropagation()}
                  >
                    <div className="flex items-center gap-2">
                      {renderActions(img)}
                    </div>
                  </TableActionCell>
                )}
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      </SurfaceCard>
    </RadioGroup>
  );
}

// ─── 空提示 ─────────────────────────────────────────────────────────
function EmptyHint({ text }: { text: string }) {
  return (
    <SurfaceCard className="px-3 py-8 text-center">
      <BodyText tone="weak" as="span">{text}</BodyText>
    </SurfaceCard>
  );
}

// ─── 公共镜像刷新按钮 ─────────────────────────────────────────────────
function PublicRefreshButton() {
  const [refreshing, setRefreshing] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const handleRefresh = () => {
    if (refreshing) return;
    setRefreshing(true);
    timerRef.current = setTimeout(() => {
      setRefreshing(false);
      toast.success("已刷新公共镜像列表");
      timerRef.current = null;
    }, 1200);
  };

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          onClick={handleRefresh}
          disabled={refreshing}
        >
          <RefreshCw className={`w-3.5 h-3.5 mr-1.5 ${refreshing ? "animate-spin" : ""}`} />
          刷新
        </Button>
      </TooltipTrigger>
      <TooltipContent className="max-w-[260px] leading-relaxed">
        重新拉取腾讯云提供的公共镜像列表（如新加白账号生效后可用）
      </TooltipContent>
    </Tooltip>
  );
}
