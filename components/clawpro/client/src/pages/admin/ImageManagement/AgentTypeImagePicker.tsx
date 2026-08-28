/**
 * AgentTypeImagePicker - 行内手风琴展开的二级列表
 *
 * 列结构（与主表对齐）：
 *   [启用圆点] [用户可见徽章] | 版本 | 镜像（含状态/导入时间副位） | 操作
 *
 * - 系统预设类型：Tabs[公共镜像 / 自定义镜像]，默认 Tab = 启用所在的；都没启用 → 公共
 * - 自定义类型：无 Tab，直接展示自定义镜像列表
 * - 单选圆点点选 → 立即切换 active
 * - 自定义镜像缺版本号的"未关联版本"镜像也直接混排在自定义列表里，靠版本列 ⚠ 标识
 * - 「+ 导入自定义镜像」按钮放在自定义 Tab 右上角，使用蓝紫渐变实心样式
 */
import { useEffect, useMemo, useState } from "react";
import {
  CheckCircle2,
  History,
  Pencil,
  Plus,
  Trash2,
  AlertTriangle,
  RefreshCw,
} from "lucide-react";
import { toast } from "sonner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import type { AgentTypeView, ViewImage } from "./deriveAgentTypeView";

// ─── 状态徽章（紧凑） ─────────────────────────────────────────────────
function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { text: string; dot: string; text2: string }> = {
    available: { text: "可用", dot: "bg-green-500", text2: "text-gray-600" },
    creating: { text: "创建中", dot: "bg-amber-500", text2: "text-amber-600" },
    failed: { text: "异常", dot: "bg-red-500", text2: "text-red-600" },
    error: { text: "异常", dot: "bg-red-500", text2: "text-red-600" },
  };
  const c = map[status] ?? map.available;
  return (
    <span className={`inline-flex items-center gap-1 text-[11px] ${c.text2} whitespace-nowrap`}>
      <span className={`w-1.5 h-1.5 rounded-full ${c.dot}`} />
      {c.text}
    </span>
  );
}

interface Props {
  view: AgentTypeView;
  isCustomAgentType: boolean;

  onSelectImage: (imageId: string) => void;
  onEditImage: (imageId: string) => void;
  onDeleteImage: (imageId: string) => void;
  onViewPublicHistory: (publicImageId: string) => void;
  onImportCustom: () => void;
}

export default function AgentTypeImagePicker({
  view,
  isCustomAgentType,
  onSelectImage,
  onEditImage,
  onDeleteImage,
  onViewPublicHistory,
  onImportCustom,
}: Props) {
  const initialTab = useMemo<"public" | "custom">(() => {
    if (view.enabled.source === "custom") return "custom";
    return "public";
  }, [view.enabled.source]);

  const [tab, setTab] = useState<"public" | "custom">(initialTab);
  useEffect(() => {
    setTab(initialTab);
  }, [initialTab]);

  // 自定义类型：无 Tab，直接展示
  if (isCustomAgentType) {
    return (
      <div className="px-4 py-4">
        <CustomList
          row={view.customRow}
          onSelectImage={onSelectImage}
          onEditImage={onEditImage}
          onDeleteImage={onDeleteImage}
          onImportCustom={onImportCustom}
        />
      </div>
    );
  }

  const publicCount = view.publicRow?.allImages.length ?? 0;
  const customCount = view.customRow.allImages.length; // 含孤儿
  const enabledInPublic = view.enabled.source === "public";
  const enabledInCustom = view.enabled.source === "custom";

  return (
    <div className="px-4 py-4">
      <Tabs value={tab} onValueChange={(v) => setTab(v as "public" | "custom")}>
        <TabsList className="h-auto p-0.5 bg-gray-100/80">
          <TabsTrigger
            value="public"
            className="text-xs h-7 data-[state=active]:bg-white"
          >
            公共镜像
            <span className="ml-1.5 text-gray-400">({publicCount})</span>
            {enabledInPublic && (
              <CheckCircle2 className="w-3 h-3 ml-1 text-blue-500" />
            )}
          </TabsTrigger>
          <TabsTrigger
            value="custom"
            className="text-xs h-7 data-[state=active]:bg-white"
          >
            自定义镜像
            <span className="ml-1.5 text-gray-400">({customCount})</span>
            {enabledInCustom && (
              <CheckCircle2 className="w-3 h-3 ml-1 text-blue-500" />
            )}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="public" className="mt-3">
          <div className="flex items-center justify-between mb-2">
            <p className="text-[11px] text-gray-500">
              由腾讯云持续维护更新，自动跟随官方版本
            </p>
            <PublicRefreshButton />
          </div>
          {view.publicRow && view.publicRow.allImages.length > 0 ? (
            <ImageList
              images={view.publicRow.allImages}
              accent="blue"
              onSelectImage={onSelectImage}
              renderActions={(img) => (
                <>
                  <button
                    onClick={() => onViewPublicHistory(img.id)}
                    className="px-2 py-1 text-[11px] text-blue-600 hover:bg-blue-50 rounded inline-flex items-center gap-1 transition-colors whitespace-nowrap"
                  >
                    <History className="w-3 h-3" />
                    版本更新记录
                  </button>
                  <DeleteIconButton
                    disabled={!!(img.isEffective || img.isVirtual)}
                    disabledReason={
                      img.isEffective
                        ? "用户可见的镜像不可删除"
                        : img.isVirtual
                          ? "腾讯云提供的镜像，未启用过无需删除"
                          : ""
                    }
                    onClick={() => onDeleteImage(img.id)}
                  />
                </>
              )}
            />
          ) : (
            <EmptyHint text="暂无公共镜像" />
          )}
        </TabsContent>

        <TabsContent value="custom" className="mt-3">
          <CustomList
            row={view.customRow}
            onSelectImage={onSelectImage}
            onEditImage={onEditImage}
            onDeleteImage={onDeleteImage}
            onImportCustom={onImportCustom}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ─── 自定义镜像列表 ─────────────────────────────────────────────────────
function CustomList({
  row,
  onSelectImage,
  onEditImage,
  onDeleteImage,
  onImportCustom,
}: {
  row: AgentTypeView["customRow"];
  onSelectImage: (imageId: string) => void;
  onEditImage: (imageId: string) => void;
  onDeleteImage: (imageId: string) => void;
  onImportCustom: () => void;
}) {
  // 全部自定义镜像（含孤儿），按导入时间倒序混排
  const all = row.allImages
    .filter((i) => i.source === "custom")
    .sort((a, b) => b.createTime.localeCompare(a.createTime));

  return (
    <>
      <div className="flex items-center justify-between mb-2">
        <p className="text-[11px] text-[var(--text-tertiary)]">由企业自行制作和维护</p>
        <Button
          onClick={onImportCustom}
          variant="claw-primary"
          size="sm"
          className="h-7 px-2.5 text-[11px] gap-1"
        >
          <Plus className="w-3 h-3" />
          导入自定义镜像
        </Button>
      </div>

      {all.length > 0 ? (
        <ImageList
          images={all}
          accent="purple"
          allowSelectIfMissingVersion={false}
          onSelectImage={onSelectImage}
          renderActions={(img) => {
            const missingVersion = !img.agentVersion?.trim();
            return (
              <>
                {missingVersion && (
                  <button
                    onClick={() => onEditImage(img.id)}
                    className="px-2 py-1 text-[11px] text-blue-600 hover:bg-blue-50 rounded transition-colors whitespace-nowrap"
                  >
                    编辑补齐版本
                  </button>
                )}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      onClick={() => onEditImage(img.id)}
                      className="p-1 text-gray-400 hover:text-blue-500 transition-colors rounded"
                    >
                      <Pencil className="w-3.5 h-3.5" />
                    </button>
                  </TooltipTrigger>
                  <TooltipContent>编辑镜像信息</TooltipContent>
                </Tooltip>
                <DeleteIconButton
                  disabled={img.isEffective}
                  disabledReason="用户可见的镜像不可删除"
                  onClick={() => onDeleteImage(img.id)}
                />
              </>
            );
          }}
        />
      ) : (
        <EmptyHint text="尚未导入任何自定义镜像" />
      )}
    </>
  );
}

// ─── 通用镜像列表 ───────────────────────────────────────────────────
function ImageList({
  images,
  accent,
  allowSelectIfMissingVersion = true,
  onSelectImage,
  renderActions,
}: {
  images: ViewImage[];
  accent: "blue" | "purple";
  allowSelectIfMissingVersion?: boolean;
  onSelectImage: (imageId: string) => void;
  renderActions: (img: ViewImage) => React.ReactNode;
}) {
  const accentColors = {
    blue: {
      ringChecked: "border-blue-500 bg-blue-500",
      bgChecked: "bg-blue-50/60",
    },
    purple: {
      ringChecked: "border-purple-500 bg-purple-500",
      bgChecked: "bg-purple-50/60",
    },
  } as const;
  const colors = accentColors[accent];

  return (
    <div className="rounded-[4px] border border-border bg-background overflow-hidden">
      {images.map((img, idx) => {
        const checked = img.isEffective;
        const missingVersion = !img.agentVersion?.trim();
        const selectable = allowSelectIfMissingVersion || !missingVersion;
        return (
          <div
            key={img.id}
            onClick={() => selectable && onSelectImage(img.id)}
            className={`flex items-center gap-3 px-3 py-2.5 transition-colors ${
              idx > 0 ? "border-t border-gray-100" : ""
            } ${
              checked
                ? colors.bgChecked
                : selectable
                  ? "bg-white hover:bg-gray-50/60 cursor-pointer"
                  : "bg-amber-50/20"
            }`}
          >
            {/* 单选圆点 */}
            <span
              className={`inline-flex items-center justify-center w-4 h-4 rounded-full border-2 shrink-0 ${
                checked
                  ? colors.ringChecked
                  : selectable
                    ? "border-gray-300 bg-white"
                    : "border-gray-200 bg-gray-50"
              }`}
            >
              {checked && <span className="w-1.5 h-1.5 rounded-full bg-white" />}
            </span>

            {/* 用户可见徽章 */}
            <div className="basis-0 grow shrink-0 min-w-[70px] max-w-[90px]">
              {checked ? (
                <span className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-semibold bg-blue-50 text-blue-700 border border-blue-200 whitespace-nowrap">
                  <CheckCircle2 className="w-2.5 h-2.5" />
                  用户可见
                </span>
              ) : (
                <span className="text-[11px] text-gray-300">—</span>
              )}
            </div>

            {/* 版本（含缺版本号警告） */}
            <div className="basis-0 grow shrink-0 min-w-[100px] max-w-[130px]">
              {missingVersion ? (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-semibold bg-amber-50 text-amber-700 border border-amber-200 whitespace-nowrap cursor-help">
                      <AlertTriangle className="w-2.5 h-2.5" />
                      缺版本号
                    </span>
                  </TooltipTrigger>
                  <TooltipContent className="max-w-[240px] text-xs">
                    缺少版本号，无法对用户可见，请编辑后补齐
                  </TooltipContent>
                </Tooltip>
              ) : (
                <span className="font-mono font-semibold text-[13px] text-gray-900 tabular-nums whitespace-nowrap">
                  v{img.agentVersion}
                </span>
              )}
            </div>

            {/* 镜像（名称 + ID 副位含状态）—— 占主要空间 */}
            <div className="basis-0 grow-[3] min-w-0">
              <div className="flex items-center gap-1.5 flex-wrap">
                <span className="text-[13px] font-medium text-gray-900 truncate">
                  {img.name}
                </span>
              </div>
              <div className="flex items-center gap-2 text-[11px] text-gray-400 mt-0.5 flex-wrap">
                <span className="font-mono truncate">{img.id}</span>
                <span className="text-gray-300">·</span>
                <StatusBadge status={img.status} />
              </div>
            </div>

            {/* 导入时间 */}
            <div className="basis-0 grow shrink-0 min-w-[90px] max-w-[110px] text-[11px] text-gray-500 font-mono tabular-nums whitespace-nowrap">
              {img.createTime ? img.createTime.split(" ")[0] : "—"}
            </div>

            {/* 操作 */}
            <div
              className="basis-0 grow-[1.5] shrink-0 min-w-[120px] flex items-center justify-end gap-1"
              onClick={(e) => e.stopPropagation()}
            >
              {renderActions(img)}
            </div>
          </div>
        );
      })}
    </div>
  );
}

// ─── 删除按钮 ───────────────────────────────────────────────────────
function DeleteIconButton({
  disabled,
  disabledReason,
  onClick,
}: {
  disabled: boolean;
  disabledReason: string;
  onClick: () => void;
}) {
  if (disabled) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex p-1 cursor-not-allowed">
            <Trash2 className="w-3.5 h-3.5 text-gray-200" />
          </span>
        </TooltipTrigger>
        <TooltipContent side="left" className="max-w-[220px] text-xs">
          {disabledReason}
        </TooltipContent>
      </Tooltip>
    );
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          onClick={onClick}
          className="p-1 text-gray-400 hover:text-red-500 transition-colors rounded"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </TooltipTrigger>
      <TooltipContent side="left" className="text-xs">
        删除此镜像
      </TooltipContent>
    </Tooltip>
  );
}

// ─── 空提示 ─────────────────────────────────────────────────────────
function EmptyHint({ text }: { text: string }) {
  return (
    <div className="rounded-[4px] border border-dashed border-border px-3 py-6 text-center">
      <span className="text-xs text-[var(--text-tertiary)]">{text}</span>
    </div>
  );
}

// ─── 公共镜像刷新按钮 ─────────────────────────────────────────────────
function PublicRefreshButton() {
  const [refreshing, setRefreshing] = useState(false);
  const handleRefresh = () => {
    if (refreshing) return;
    setRefreshing(true);
    // mock：1.2 秒后完成；真实场景下走刷新接口
    setTimeout(() => {
      setRefreshing(false);
      toast.success("已刷新公共镜像列表");
    }, 1200);
  };
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          onClick={handleRefresh}
          disabled={refreshing}
          className="px-2 py-1 text-[11px] text-gray-600 bg-white border border-gray-300 hover:border-blue-300 hover:text-blue-600 rounded inline-flex items-center gap-1 transition-colors whitespace-nowrap disabled:opacity-60"
        >
          <RefreshCw
            className={`w-3 h-3 ${refreshing ? "animate-spin" : ""}`}
          />
          刷新
        </button>
      </TooltipTrigger>
      <TooltipContent className="max-w-[260px] text-xs leading-relaxed">
        重新拉取腾讯云提供的公共镜像列表（如新加白账号生效后可用）
      </TooltipContent>
    </Tooltip>
  );
}
