/**
 * AgentTypesTable - 所有 Agent 类型在一张大表里
 *
 * 列定义（8 列）：
 *   Agent 类型 | Agent 版本 | 镜像来源 | 镜像 | 镜像状态 | 应用范围 | 用户可见 | 操作
 *
 *   - 「Agent 版本」列：版本号 + 自动更新/指定版本模式切换 + 版本下拉框（指定模式）+ 更新记录入口
 *   - 「镜像来源」列：来源 tag（公共/自定义）独立成列
 *   - 「镜像」列：第一行 = 镜像名 + 切换图标；第二行 = 镜像 ID
 *   - 「应用范围」列：每类型一个 Popover，决定该类型的镜像对哪些用户可见
 *   - 「操作」列：「设为首选 / 删除」两枚文字按钮（link 蓝色文字）；不可执行时禁用并 Tooltip 提示原因
 *
 * 行模型：每个 Agent 类型一行（不再有手风琴展开行）；
 *   未选镜像时镜像列展示「选择镜像」link 按钮；点击或镜像列内的「切换」图标都会弹出 SwitchImageDialog（标准弹窗组件 size=lg），
 *   在弹窗内单选目标镜像 → 点击「确认切换」完成切换。
 *
 * 样式规范：使用 @/components/ui/table 标准 Table 组件（禁止裸 <table>）
 *  - 表格内所有元素字号由全局 CSS 强制为 12px（text-xs），无需手动指定字号
 *  - TableCell 默认字色 #0A0A0A（纯黑），辅助信息可覆盖为 tone="weak"
 *  - 操作列使用 Button variant="link"（统一品牌蓝文字按钮）
 *  - 所有独立文字使用 Typography 组件（BodyMedium / BodyText / MetaText）
 */
import { useMemo, useState, useEffect, type ReactNode } from "react";
import {
  History,
  ArrowLeftRight,
  AlertTriangle,
  Bell,
  BellOff,
  Info,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { StatusTag } from "@/components/ui/status-tag";
import { SurfaceCard } from "@/components/ui/Surface";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import {
  BodyMedium,
  BodyText,
  MetaText,
} from "@/components/ui/Typography";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import SwitchImageDialog from "./SwitchImageDialog";
import { ImageStatusBadge } from "./ImageStatusBadge";
import type { AgentTypeView, ViewImage } from "./deriveAgentTypeView";
import { IMG_TO_VERSION_KEY } from "./deriveAgentTypeView";
import type { CustomAgentType } from "./types";
import { AGENT_VERSIONS } from "../VersionManagement/mockData";
import {
  getVersionMode,
  setVersionMode,
  getPinnedVersion,
  setPinnedVersion,
  getEffectiveVersion,
  getLatestPlatformVersion,
  hasNewerVersion,
} from "@/lib/versionModeStore";
import { pruneOnVersionChange } from "@/lib/upgradePushStore";
import { isReminderEnabled, setReminderEnabled } from "@/lib/updateReminderStore";
import { toast } from "sonner";

// ─── 主组件 ───────────────────────────────────────────────────────────
export interface AgentTypeRowData {
  agentType: string;
  view: AgentTypeView;
  label: string;
  isDefault: boolean;
  customType?: CustomAgentType;
  /** 兼容内核 / 自研内核展示标签（如"OpenClaw" 或 undefined） */
  kernelBaseLabel?: string;
}

interface Props {
  rows: AgentTypeRowData[];

  // 类型级操作
  onSetDefaultType: (agentType: string) => void;
  onRemoveCustomType: (agentType: string) => void;
  onPushUpgrade: (agentType: string) => void;
  /** 撤回某 Agent 类型当前正在生效的推送 */
  onRevokePush?: (agentType: string) => void;
  /** 版本模式/版本号变更回调（用于父组件联动推送状态） */
  onVersionModeChange?: (agentType: string, mode: "auto" | "pinned", version: string | null) => void;

  // 镜像级操作
  onEnableImage: (imageId: string, agentType: string) => void;
  onDisableImage: (imageId: string) => void;
  onSelectImage: (imageId: string, agentType: string) => void;
  onEditImage: (imageId: string) => void;
  onDeleteImage: (imageId: string) => void;
  onViewPublicHistory: (publicImageId: string) => void;
  onImportCustom: (agentType: string) => void;

  /** 各 Agent 类型的可推送状态（agentType -> 过期实例信息 + 是否在推送中） */
  pushableByType?: Map<
    string,
    { outdatedInstanceCount: number; allUpToDate: boolean; isActivePushing?: boolean }
  >;

  /**
   * 启用引导：由「克隆 Agent 为镜像」弹窗跳转携带 ?enableImage=<id> 触发。
   * 定位包含该镜像的 Agent 类型行 → 自动打开「切换镜像」弹窗并预选该镜像，同时高亮该行。
   */
  autoEnableImageId?: string;

  // 应用范围渲染槽：由父组件按 agentType 渲染各自的 Popover
  renderScope: (agentType: string) => ReactNode;
}

export default function AgentTypesTable({
  rows,
  onSetDefaultType,
  onRemoveCustomType,
  onPushUpgrade,
  onRevokePush,
  onVersionModeChange,
  onEnableImage,
  onDisableImage,
  onSelectImage,
  onEditImage,
  onDeleteImage,
  onViewPublicHistory,
  onImportCustom,
  pushableByType,
  autoEnableImageId,
  renderScope,
}: Props) {
  // 当前打开切换弹窗的 agentType（null = 未打开）
  const [dialogAgentType, setDialogAgentType] = useState<string | null>(null);
  const dialogRow = useMemo(
    () =>
      dialogAgentType
        ? rows.find((r) => r.agentType === dialogAgentType) ?? null
        : null,
    [rows, dialogAgentType]
  );

  // 启用引导：高亮目标类型行（切换镜像弹窗关闭后清除）
  const [highlightAgentType, setHighlightAgentType] = useState<string | null>(null);
  useEffect(() => {
    if (!autoEnableImageId) return;
    const hit = rows.find(
      (r) =>
        r.view.customRow?.allImages?.some((i) => i.id === autoEnableImageId) ||
        r.view.publicRow?.allImages?.some((i) => i.id === autoEnableImageId)
    );
    if (hit) {
      setHighlightAgentType(hit.agentType);
      setDialogAgentType(hit.agentType);
    }
    // 仅在携带参数进入时执行一次
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [autoEnableImageId]);

  return (
    <SurfaceCard className="overflow-hidden">
      <Table variant="white" className="table-auto" scrollX={1160}>
        <TableHeader>
          <TableRow>
            <TableHead fixed="left" style={{ width: 170, minWidth: 170, maxWidth: 170 }}>
              Agent 类型
            </TableHead>
            <TableHead style={{ minWidth: 200 }}>Agent 版本</TableHead>
            <TableHead style={{ minWidth: 90 }}>镜像来源</TableHead>
            <TableHead style={{ minWidth: 210 }}>镜像</TableHead>
            <TableHead style={{ minWidth: 90 }}>镜像状态</TableHead>
            <TableHead style={{ minWidth: 140 }}>应用范围</TableHead>
            <TableHead style={{ minWidth: 90 }}>用户可见</TableHead>
            <TableHead fixed="right" style={{ minWidth: 150, width: "1%" }}>操作</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <AgentTypeRow
              key={row.agentType}
              row={row}
              highlighted={highlightAgentType === row.agentType}
              onOpenSwitchDialog={() => setDialogAgentType(row.agentType)}
              onSetDefaultType={() => onSetDefaultType(row.agentType)}
              onRemoveCustomType={() => onRemoveCustomType(row.agentType)}
              onPushUpgrade={() => onPushUpgrade(row.agentType)}
              onRevokePush={onRevokePush ? () => onRevokePush(row.agentType) : undefined}
              onVersionModeChange={onVersionModeChange}
              onEnableImage={(imgId) => onEnableImage(imgId, row.agentType)}
              onDisableImage={onDisableImage}
              onViewPublicHistory={onViewPublicHistory}
              pushableInfo={pushableByType?.get(row.agentType)}
              scopeSlot={renderScope(row.agentType)}
            />
          ))}
        </TableBody>
      </Table>

      {dialogRow && (
        <SwitchImageDialog
          open={!!dialogAgentType}
          onOpenChange={(open) => {
            if (!open) {
              setDialogAgentType(null);
              setHighlightAgentType(null);
            }
          }}
          agentTypeLabel={dialogRow.label}
          view={dialogRow.view}
          isCustomAgentType={!!dialogRow.customType}
          initialPendingId={autoEnableImageId || undefined}
          onConfirm={(imageId) =>
            onSelectImage(imageId, dialogRow.agentType)
          }
          onEditImage={onEditImage}
          onDeleteImage={onDeleteImage}
          onViewPublicHistory={onViewPublicHistory}
          onImportCustom={() => onImportCustom(dialogRow.agentType)}
        />
      )}
    </SurfaceCard>
  );
}

// ─── 单行 ────────────────────────────────────────────────────────────
function AgentTypeRow({
  row,
  highlighted,
  onOpenSwitchDialog,
  onSetDefaultType,
  onRemoveCustomType,
  onPushUpgrade,
  onRevokePush,
  onVersionModeChange,
  onEnableImage,
  onDisableImage,
  onViewPublicHistory,
  pushableInfo,
  scopeSlot,
}: {
  row: AgentTypeRowData;
  /** 启用引导高亮（?enableImage= 跳转定位到本行时为 true） */
  highlighted?: boolean;
  onOpenSwitchDialog: () => void;
  onSetDefaultType: () => void;
  onRemoveCustomType: () => void;
  onPushUpgrade: () => void;
  onRevokePush?: () => void;
  onVersionModeChange?: (agentType: string, mode: "auto" | "pinned", version: string | null) => void;
  onEnableImage: (imgId: string) => void;
  onDisableImage: (imgId: string) => void;
  onViewPublicHistory: (imgId: string) => void;
  pushableInfo?: { outdatedInstanceCount: number; allUpToDate: boolean; isActivePushing?: boolean };
  scopeSlot: ReactNode;
}) {
  const { view, label, isDefault, customType, kernelBaseLabel } = row;
  const isNative = customType?.kernelBase === "native";
  const isCustom = !!customType;

  const selected = view.selectedImage;
  const isEnabled = view.enabled.isEnabled;

  const handleSwitchToggle = (next: boolean) => {
    if (!selected) return;
    if (next) onEnableImage(selected.id);
    else onDisableImage(selected.id);
  };

  return (
    <TableRow
      id={`section-${row.agentType}`}
      data-anchor={row.agentType}
      className={`group [&>td]:py-4 [&>td]:align-middle ${highlighted ? "ring-1 ring-inset ring-[#1447E6] [&>td]:bg-[#F0F5FF]" : ""}`}
    >
      {/* 1. Agent 类型 */}
      <TableCell
        fixed="left"
        className="py-4 whitespace-normal"
        style={{
          width: 170,
          minWidth: 170,
          maxWidth: 170,
        }}
      >
        <div className="min-w-0">
          <BodyMedium tone="primary" className="font-semibold truncate max-w-[130px] block">
            {label}
          </BodyMedium>
          {(isDefault || isNative || (customType && !isNative && kernelBaseLabel)) && (
            <div className="mt-1">
              {isDefault && <StatusTag mode="fill" variant="blue">用户端首选</StatusTag>}
              {isNative && <StatusTag mode="fill" variant="gray">自定义内核</StatusTag>}
              {customType && !isNative && kernelBaseLabel && (
                <StatusTag variant="role">{`兼容 ${kernelBaseLabel}`}</StatusTag>
              )}
            </div>
          )}
        </div>
      </TableCell>


      {/* 2. Agent 版本 */}
      <TableCell className="py-4">
        {selected ? (
          <AgentVersionCell
            image={selected}
            agentType={view.agentType}
            agentLabel={label}
            onViewHistory={view.publicRow ? () => onViewPublicHistory(selected.id) : undefined}
            onVersionModeChange={onVersionModeChange}
          />
        ) : (
          <BodyText tone="weak" as="span">—</BodyText>
        )}
      </TableCell>

      {/* 3. 镜像来源（独立列：公共/自定义） */}
      <TableCell>
        {selected ? (
          <StatusTag mode="fill" variant={selected.source === "public" ? "blue" : "gray"}>
            {selected.source === "public" ? "公共" : "自定义"}
          </StatusTag>
        ) : (
          <BodyText tone="weak" as="span">—</BodyText>
        )}
      </TableCell>

      {/* 4. 镜像（名称 + 切换按钮 + ID） */}
      <TableCell className="whitespace-normal">
        {selected ? (
          <div className="min-w-0">
            <div className="flex items-center gap-2 min-w-0">
              <BodyMedium tone="primary" className="truncate min-w-0">
                {selected.name}
              </BodyMedium>
              <Button variant="claw-outline" size="claw-sm" onClick={onOpenSwitchDialog} className="shrink-0 !h-6 !px-2 !py-0 !text-xs">
                <ArrowLeftRight className="w-2 h-2 mr-0.5" />
                切换
              </Button>
            </div>
            <div className="flex items-center gap-2 mt-1">
              <MetaText tone="weak" className="truncate">{selected.id}</MetaText>
            </div>
          </div>
        ) : (
          <div className="flex items-center gap-2">
            <BodyText tone="weak" as="span">尚未选择镜像</BodyText>
            <Button variant="link" size="sm" onClick={onOpenSwitchDialog}>
              去选择
            </Button>
          </div>
        )}
      </TableCell>

      {/* 5. 镜像状态（独立列） */}
      <TableCell>
        {selected ? (
          <ImageStatusBadge status={selected.status} />
        ) : (
          <BodyText tone="weak" as="span">—</BodyText>
        )}
      </TableCell>

      {/* 5. 应用范围（外部注入） */}
      <TableCell className="whitespace-normal">
        {scopeSlot}
      </TableCell>

      {/* 6. 用户可见 */}
      <TableCell>
        {isDefault ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="inline-flex items-center gap-1.5 cursor-not-allowed">
                <Switch checked disabled />
              </span>
            </TooltipTrigger>
            <TooltipContent side="top">
              <MetaText tone="inherit">用户端首选 Agent 必须保持用户可见</MetaText>
            </TooltipContent>
          </Tooltip>
        ) : selected ? (
          isEnabled ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="inline-flex items-center gap-1.5">
                  <Switch
                    checked={isEnabled}
                    onCheckedChange={handleSwitchToggle}
                  />
                </span>
              </TooltipTrigger>
              <TooltipContent side="top">
                <MetaText tone="inherit">取消用户可见后，系统将不再推送 Agent 版本更新信息</MetaText>
              </TooltipContent>
            </Tooltip>
          ) : (
            <span className="inline-flex items-center gap-1.5">
              <Switch
                checked={isEnabled}
                onCheckedChange={handleSwitchToggle}
              />
            </span>
          )
        ) : (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="inline-flex items-center gap-1.5 cursor-not-allowed">
                <Switch checked={false} disabled />
              </span>
            </TooltipTrigger>
            <TooltipContent side="top">
              <MetaText tone="inherit">请先选择一个镜像</MetaText>
            </TooltipContent>
          </Tooltip>
        )}
      </TableCell>

      {/* 6. 操作：「设为首选 / 删除」两枚文字按钮（条件不满足时禁用） */}
      <TableActionCell fixed="right">
        {/* 设为首选 — 已首选时保留按钮但禁用；未选镜像/用户不可见时禁用并 Tooltip 提示原因 */}
        {isDefault ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span tabIndex={0} className="inline-block">
                <Button variant="link" size="sm" disabled>
                  设为首选
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent side="top">
              <MetaText tone="inherit">当前已是用户端首选类型</MetaText>
            </TooltipContent>
          </Tooltip>
        ) : !selected || !isEnabled ? (
          <Tooltip>
            <TooltipTrigger asChild>
              <span tabIndex={0} className="inline-block">
                <Button variant="link" size="sm" disabled>
                  设为首选
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent side="top">
              <MetaText tone="inherit">{!selected ? "请先选择该类型的镜像" : "请先开启用户可见"}</MetaText>
            </TooltipContent>
          </Tooltip>
        ) : (
          <Button variant="link" size="sm" onClick={onSetDefaultType}>
            设为首选
          </Button>
        )}

        {/* 删除 — 仅自定义类型可删除；系统预设类型隐藏 */}
        {isCustom && (
          <Button variant="link" size="sm" onClick={onRemoveCustomType}>
            删除
          </Button>
        )}
      </TableActionCell>
    </TableRow>
  );
}

// ─── Agent 版本单元（版本号 + 自动更新/指定版本模式 + 更新记录入口） ──
function AgentVersionCell({
  image,
  agentType,
  agentLabel,
  onViewHistory,
  onVersionModeChange,
}: {
  image: ViewImage;
  agentType: string;
  agentLabel: string;
  onViewHistory?: () => void;
  onVersionModeChange?: (agentType: string, mode: "auto" | "pinned", version: string | null) => void;
}) {
  const isPublic = image.source === "public";
  const versionKey = IMG_TO_VERSION_KEY[agentType];
  const versionList = versionKey
    ? AGENT_VERSIONS.filter((v) => v.agentType === versionKey)
    : [];
  const hasHistory = isPublic && !!onViewHistory && versionList.length > 0;

  // 版本模式状态
  const [mode, setMode] = useState<"auto" | "pinned">(() => {
    if (isPublic) return getVersionMode(agentType);
    return "auto";
  });
  const [pinnedVersion, setPinnedVersionState] = useState<string | null>(() => {
    if (isPublic) return getPinnedVersion(agentType);
    return null;
  });

  // 更新提醒开关
  const [reminderOn, setReminderOn] = useState(() => isReminderEnabled(agentType));

  // 有效版本
  const effectiveVersion = isPublic
    ? (mode === "auto"
      ? getLatestPlatformVersion(agentType) || image.agentVersion
      : pinnedVersion || getLatestPlatformVersion(agentType) || image.agentVersion)
    : image.agentVersion;

  // 所有版本（按发布时间倒序）
  const allVersionsSorted = useMemo(() => {
    if (!versionKey) return [];
    return AGENT_VERSIONS
      .filter((v) => v.agentType === versionKey)
      .sort((a, b) => b.releaseTime.localeCompare(a.releaseTime));
  }, [versionKey]);

  // 最近 3 个版本
  const top3 = useMemo(() => allVersionsSorted.slice(0, 3), [allVersionsSorted]);

  // 可选版本列表：最近 3 个 + 当前版本（如果当前版本不在 top3 中）
  const versionInfos = useMemo(() => {
    if (!isPublic || !versionKey) return [];
    const list = [...top3];
    if (effectiveVersion && !list.some((v) => v.version === effectiveVersion)) {
      const extra = allVersionsSorted.find((v) => v.version === effectiveVersion);
      if (extra) list.push(extra);
    }
    return list;
  }, [isPublic, versionKey, effectiveVersion, top3, allVersionsSorted]);

  // 是否有新版本
  const newerAvailable = isPublic && hasNewerVersion(agentType, effectiveVersion);

  // 切换到旧版本（离开后不可回选）的二次确认
  const [showOldVersionConfirm, setShowOldVersionConfirm] = useState(false);
  const [pendingVersion, setPendingVersion] = useState<string | null>(null);

  // 当前版本是否不在 top3 中（即"旧版本"）
  const isCurrentOld = effectiveVersion && !top3.some((v) => v.version === effectiveVersion);

  const handleModeChange = (newMode: "auto" | "pinned") => {
    setMode(newMode);
    setVersionMode(agentType, newMode);

    if (newMode === "pinned") {
      // 切换到指定版本时，默认选中当前最新版本
      const latest = getLatestPlatformVersion(agentType) || effectiveVersion;
      setPinnedVersionState(latest);
      setPinnedVersion(agentType, latest);
      onVersionModeChange?.(agentType, "pinned", latest);
    } else {
      // 切换到自动更新
      setPinnedVersionState(null);
      setPinnedVersion(agentType, "");
      onVersionModeChange?.(agentType, "auto", null);
    }
  };

  const handleVersionChange = (v: string) => {
    // 当前版本是旧版本（不在 top3 中）且用户要切换到新版本 → 二次确认
    if (isCurrentOld && v !== effectiveVersion) {
      setPendingVersion(v);
      setShowOldVersionConfirm(true);
    } else {
      applyVersionChange(v);
    }
  };

  const applyVersionChange = (v: string) => {
    setPinnedVersionState(v);
    setPinnedVersion(agentType, v);
    onVersionModeChange?.(agentType, "pinned", v);
    toast.success(`「${agentLabel}」已指定版本 v${v}`);
  };

  const confirmOldVersion = () => {
    if (pendingVersion) {
      applyVersionChange(pendingVersion);
      setPendingVersion(null);
      setShowOldVersionConfirm(false);
    }
  };

  return (
    <div className="min-w-0 py-0.5">
      {/* Row 1: 版本号 + 历史按钮 + 提醒开关 */}
      <div className="flex items-center gap-1 whitespace-nowrap">
        <BodyMedium tone="primary" className="tabular-nums text-xs">
          v{effectiveVersion}
        </BodyMedium>
        {onViewHistory && (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                disabled={!hasHistory}
                onClick={onViewHistory}
                className="cursor-pointer inline-flex items-center justify-center w-4 h-4 rounded-[3px] text-[var(--text-weak)] hover:text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)] transition-colors disabled:cursor-not-allowed disabled:text-[var(--text-weak)] disabled:hover:bg-transparent"
                aria-label="版本更新记录"
              >
                <History className="w-3 h-3" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="top">
              <MetaText tone="inherit">{hasHistory ? "版本更新记录" : "暂无版本更新记录"}</MetaText>
            </TooltipContent>
          </Tooltip>
        )}
        {isPublic && (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={() => {
                  const next = !reminderOn;
                  setReminderOn(next);
                  setReminderEnabled(agentType, next);
                  if (next) {
                    toast.success(`「${agentLabel}」有新版本时将提醒用户更新`);
                  }
                }}
                className="cursor-pointer inline-flex items-center justify-center w-4 h-4 rounded-[3px] text-[var(--text-weak)] hover:text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)] transition-colors"
              >
                {reminderOn ? (
                  <Bell className="w-3 h-3 scale-110 transition-transform" color="var(--brand-blue)" />
                ) : (
                  <BellOff className="w-3 h-3" color="var(--text-weak)" />
                )}
              </button>
            </TooltipTrigger>
            <TooltipContent side="top">
              <MetaText tone="inherit">
                {reminderOn ? "有新版本时提醒用户端更新" : "开启后，有新版本时提醒用户端更新"}
              </MetaText>
            </TooltipContent>
          </Tooltip>
        )}
      </div>

      {/* Row 2: 模式下拉 + 版本下拉 + 帮助图标 */}
      {isPublic ? (
        <div className="flex items-center gap-1.5 mt-1">
          {/* 模式选择下拉框 */}
          <Select value={mode} onValueChange={(v) => handleModeChange(v as "auto" | "pinned")}>
            <SelectTrigger size="sm" className="!h-7 w-[100px] px-2 text-xs bg-background">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="auto" className="text-xs">自动更新</SelectItem>
              <SelectItem value="pinned" className="text-xs">指定版本</SelectItem>
            </SelectContent>
          </Select>

          {/* 指定版本模式：版本选择下拉框 */}
          {mode === "pinned" && (
            <span className="relative inline-flex">
              <Select value={pinnedVersion || effectiveVersion} onValueChange={handleVersionChange}>
              <SelectTrigger size="sm" className="!h-7 text-xs bg-white w-[108px] px-2">
                <span className="tabular-nums">v{pinnedVersion || effectiveVersion}</span>
              </SelectTrigger>
              <SelectContent align="start" className="min-w-[200px]">
                {versionInfos.map((vi) => (
                  <SelectItem key={vi.version} value={vi.version} className="text-xs">
                    <span className="font-medium tabular-nums">v{vi.version}</span>
                    <span className="ml-2 text-[var(--text-muted)]">{vi.releaseTime}</span>
                    {vi.isLatest && (
                      <span className="ml-1 text-[10px] text-[var(--text-brand)]">最新</span>
                    )}
                  </SelectItem>
                ))}
                {/* 提示：仅显示最近 3 个版本 */}
                <div className="px-2 py-1.5 text-[10px] text-[var(--text-weak)] leading-relaxed border-t border-[var(--border)] mt-1">
                  仅显示最近 3 个版本。切换后，超出最近 3 个的较旧版本将不可再选择。
                </div>
              </SelectContent>
            </Select>
              {newerAvailable && (
                <span className="absolute -top-0.5 -right-0.5 size-2 rounded-full bg-orange-500 ring-1 ring-white" />
              )}
            </span>
          )}

          {/* 帮助图标 */}
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="cursor-default shrink-0"><Info className="w-3.5 h-3.5 text-gray-400" /></span>
            </TooltipTrigger>
            <TooltipContent side="top" className="max-w-[260px] bg-white border border-[var(--border)] text-[var(--text-title)]">
              <div className="space-y-2 text-xs">
                <div>
                  <span className="font-medium">自动更新</span>
                  <span className="text-[var(--text-secondary)]">：Agent 版本随平台发布自动升级，享受最新功能与安全补丁。</span>
                </div>
                <div>
                  <span className="font-medium">指定版本</span>
                  <span className="text-[var(--text-secondary)]">：管理员手动选择并锁定版本，不自动升级，适合对稳定性要求高的场景。</span>
                </div>
              </div>
            </TooltipContent>
          </Tooltip>
        </div>
      ) : (
        /* 自定义镜像 */
        <div className="mt-0.5">
          <StatusTag variant="role">指定版本，不自动更新</StatusTag>
        </div>
      )}

      {/* 旧版本切换二次确认弹窗 */}
      <AlertDialog open={showOldVersionConfirm} onOpenChange={setShowOldVersionConfirm}>
        <AlertDialogContent className="sm:max-w-[420px]">
          <AlertDialogHeader>
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-orange-500" />
              <AlertDialogTitle>切换版本确认</AlertDialogTitle>
            </div>
          </AlertDialogHeader>
          <AlertDialogDescription asChild>
            <div className="space-y-3">
              <p className="text-sm text-[var(--text-secondary)]">
                你即将将「{agentLabel}」从 <span className="font-medium">v{effectiveVersion}</span> 切换到 <span className="font-medium">v{pendingVersion}</span>。
              </p>
              <p className="text-sm text-[var(--text-secondary)]">
                切换后，旧版本 <span className="font-medium">v{effectiveVersion}</span> 将从可选列表中移除，不可再次选择。是否确认切换？
              </p>
            </div>
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => { setShowOldVersionConfirm(false); setPendingVersion(null); }}>
              取消
            </AlertDialogCancel>
            <AlertDialogAction onClick={confirmOldVersion}>
              确认切换
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}


