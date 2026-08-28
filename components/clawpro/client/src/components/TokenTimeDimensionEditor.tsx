/**
 * TokenTimeDimensionEditor — 单用户每日 Tokens 上限 + 时间维度配置
 *
 * 视觉基线：与「基础信息配置 / OneID 模式」(StandardBasicInfo.tsx) 对齐
 *  - SurfaceInner 卡内子卡 + 4px 圆角
 *  - SegmentGroup / SegmentOption 替代旧的 PillGroup
 *  - claw-outline + dialog-confirm 按钮
 *  - Typography 语义层 + StatusTag/CircleAlert 等规范组件
 *  - 业务逻辑（state / 周期切换二次确认 / onSave 回调签名 / 类型导出）一概不动
 */
import { forwardRef, useImperativeHandle, useState } from "react";
import { ChevronDown, Info, ExternalLink } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
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
import { SegmentGroup, SegmentOption } from "@/components/ui/segment";
import { SurfaceInner } from "@/components/ui/Surface";
import {
  BodyMedium,
  HelperText,
  InlineNumber,
  MetaText,
} from "@/components/ui/Typography";
import { toast } from "sonner";

// ─── 类型 ─────────────────────────────────────────────────────────────────────

type NaturalPeriod = "daily" | "monthly" | "yearly";
type CustomRefresh = "none" | "daily" | "monthly" | "yearly";

type TimeDimensionConfig =
  | { type: "natural"; period: NaturalPeriod }
  | {
      type: "custom";
      refresh: CustomRefresh;
    };

type TokenLimit = number | "unlimited";

interface TokenTimeDimensionEditorProps {
  timeDimension: TimeDimensionConfig;
  tokenLimit: TokenLimit;
  onSave: (config: TimeDimensionConfig, limit: TokenLimit) => void;
  /** 外部控制编辑态（受卡片级"编辑"按钮驱动时使用）。
   *  - 未传 → 组件自管编辑态（点击触发器展开）
   *  - 传入 → 由外部决定是否显示编辑面板，内部不再自行切换 */
  editing?: boolean;
  /** 当外部控制编辑态时，编辑面板不显示自己的取消/保存按钮（由外部卡片按钮负责） */
  hideActions?: boolean;
}

// ─── 工具 ─────────────────────────────────────────────────────────────────────

const periodLabel: Record<NaturalPeriod, string> = {
  daily: "每日",
  monthly: "每月",
  yearly: "每年",
};

const refreshLabel: Record<CustomRefresh, string> = {
  daily: "每日刷新",
  monthly: "每月刷新",
  yearly: "每年刷新",
  none: "不刷新",
};

function getDisplayParts(
  dim: TimeDimensionConfig,
  limit: TokenLimit,
): { prefix: string; main: string; suffix: string } {
  const isUnlimited = limit === "unlimited";
  const main = isUnlimited ? "无限制" : Number(limit).toLocaleString();
  const suffix = isUnlimited ? "" : "Tokens";

  if (dim.type === "natural") {
    return {
      prefix: `自然周期 · ${periodLabel[dim.period]}`,
      main,
      suffix,
    };
  }
  const refreshStr = refreshLabel[dim.refresh] ?? refreshLabel.daily;
  return {
    prefix: `自定义周期 · ${refreshStr}`,
    main,
    suffix,
  };
}

// ─── 周期说明 Tooltip（编辑/展示态共用） ────────────────────────────────────

function PeriodInfoTooltip({
  side = "left",
  align = "start",
  className = "",
}: {
  side?: "top" | "right" | "bottom" | "left";
  align?: "start" | "center" | "end";
  className?: string;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          onClick={(e) => e.stopPropagation()}
          className={`text-[var(--text-weak)] hover:text-[var(--text-muted)] transition-colors ${className}`}
          aria-label="周期说明"
        >
          <Info className="w-4 h-4" />
        </button>
      </TooltipTrigger>
      <TooltipContent
        side={side}
        align={align}
        className="text-xs max-w-[340px] leading-relaxed px-4 py-3"
      >
        <p className="mb-1.5">
          <span className="font-medium">自然周期：</span>
          所有策略起止时间相同，新周期开始时用量自动刷新，上限额度恢复，重新开始统计。
        </p>
        <p>
          <span className="font-medium">自定义周期：</span>
          每条策略独立配置开始时间、终止时间与刷新方式。预设策略生效规则：没有组织的新增用户默认以添加用户时间为开始时间，也可在用户管理页对单个用户进行再次调整；未匹配任何组织策略的组织以预设策略最新保存时间为开始时间；均无终止时间。
        </p>
        <p className="mt-1.5 pt-1.5 border-t border-white/15">
          详细的周期介绍和 Tokens 统计规则请查看{" "}
          <a
            href="#"
            className="text-[var(--text-brand)] hover:underline inline-flex items-center gap-0.5"
            onClick={(e) => e.preventDefault()}
          >
            说明文档
            <ExternalLink className="w-3 h-3" />
          </a>
        </p>
      </TooltipContent>
    </Tooltip>
  );
}

// ─── Ref 接口 ─────────────────────────────────────────────────────────────────

export interface TokenTimeDimensionEditorRef {
  /** 外部调用：提交当前 draft，等同于用户按"保存" */
  save: () => boolean;
}

// ─── 主组件 ───────────────────────────────────────────────────────────────────

const TokenTimeDimensionEditor = forwardRef<TokenTimeDimensionEditorRef, TokenTimeDimensionEditorProps>(function TokenTimeDimensionEditor({
  timeDimension,
  tokenLimit,
  onSave,
  editing: externalEditing,
  hideActions = false,
}, ref) {
  const [internalEditing, setInternalEditing] = useState(false);

  // 外部控制模式：externalEditing 直接决定是否展示编辑面板
  // 非外部控制模式：由内部 state 决定
  const isControlled = externalEditing !== undefined;
  const editing = isControlled ? externalEditing : internalEditing;
  const setEditing = (v: boolean) => {
    if (!isControlled) setInternalEditing(v);
  };
  // 周期类型切换的二次确认
  const [pendingTypeSwitch, setPendingTypeSwitch] = useState<
    "natural" | "custom" | null
  >(null);

  // draft states
  const [draftDimType, setDraftDimType] = useState<"natural" | "custom">(
    timeDimension.type,
  );
  const [draftPeriod, setDraftPeriod] = useState<NaturalPeriod>(
    timeDimension.type === "natural" ? timeDimension.period : "daily",
  );
  const [draftRefresh, setDraftRefresh] = useState<CustomRefresh>(
    timeDimension.type === "custom" ? timeDimension.refresh : "daily",
  );
  const [draftLimitMode, setDraftLimitMode] = useState<"unlimited" | "custom">(
    tokenLimit === "unlimited" ? "unlimited" : "custom",
  );
  const [draftLimitValue, setDraftLimitValue] = useState<string>(
    tokenLimit === "unlimited" ? "500000" : String(tokenLimit),
  );

  const openEditor = () => {
    setDraftDimType(timeDimension.type);
    setDraftPeriod(
      timeDimension.type === "natural" ? timeDimension.period : "daily",
    );
    setDraftRefresh(
      timeDimension.type === "custom" ? timeDimension.refresh : "daily",
    );
    setDraftLimitMode(tokenLimit === "unlimited" ? "unlimited" : "custom");
    setDraftLimitValue(
      tokenLimit === "unlimited" ? "500000" : String(tokenLimit),
    );
    setEditing(true);
  };

  const handleSave = () => {
    if (draftLimitMode === "custom") {
      const n = parseInt(draftLimitValue, 10);
      if (isNaN(n) || n < 0) {
        toast.error("请输入大于等于 0 的整数");
        return;
      }
    }

    if (draftDimType !== timeDimension.type) {
      setPendingTypeSwitch(draftDimType);
      return;
    }

    commitSave();
  };

  const commitSave = () => {
    const finalLimit: TokenLimit =
      draftLimitMode === "unlimited"
        ? "unlimited"
        : parseInt(draftLimitValue, 10);

    const config: TimeDimensionConfig =
      draftDimType === "natural"
        ? { type: "natural", period: draftPeriod }
        : { type: "custom", refresh: draftRefresh };

    onSave(config, finalLimit);
    setEditing(false);
    setPendingTypeSwitch(null);
    toast.success("Token 配额已保存");
  };

  // 暴露 save 方法给外部（卡片级"保存"按钮调用）
  useImperativeHandle(ref, () => ({
    save: () => {
      if (draftLimitMode === "custom") {
        const n = parseInt(draftLimitValue, 10);
        if (isNaN(n) || n < 0) {
          toast.error("请输入大于等于 0 的整数");
          return false;
        }
      }
      if (draftDimType !== timeDimension.type) {
        setPendingTypeSwitch(draftDimType);
        return false; // 需要二次确认，暂不完成
      }
      const finalLimit: TokenLimit =
        draftLimitMode === "unlimited"
          ? "unlimited"
          : parseInt(draftLimitValue, 10);
      const config: TimeDimensionConfig =
        draftDimType === "natural"
          ? { type: "natural", period: draftPeriod }
          : { type: "custom", refresh: draftRefresh };
      onSave(config, finalLimit);
      return true;
    },
  }));

  const blockInvalidKeys = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (["-", "+", ".", "e", "E"].includes(e.key)) {
      e.preventDefault();
    }
  };

  const handleLimitInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value.replace(/[^0-9]/g, "");
    setDraftLimitValue(val);
  };

  const { prefix, main, suffix } = getDisplayParts(timeDimension, tokenLimit);

  return (
    <div className="space-y-2">
      <MetaText as="p">
        单用户 Tokens 上限
        <span className="text-[var(--text-weak)] ml-1">
          此处对应管控端平台策略页的预设策略
        </span>
      </MetaText>

      {!editing ? (
        /* ── 纯视图态：inline 文字展示 ── */
        <div className="flex items-baseline gap-0">
          <span className="text-xs text-[var(--text-title)] mr-1.5">{prefix}</span>
          <InlineNumber as="span" tone="emphasis" className="font-bold">
            {main}
          </InlineNumber>
          {suffix && (
            <BodyMedium as="span" tone="emphasis" className="ml-1 font-normal">
              {suffix}
            </BodyMedium>
          )}
        </div>
      ) : (
        /* ── 编辑状态：SurfaceInner 子卡（4px 圆角，无阴影） ── */
        <SurfaceInner className="relative px-4 py-3 space-y-2.5 -mr-36">
          <div className="absolute top-3 right-3">
            <PeriodInfoTooltip side="left" align="start" />
          </div>

          {/* 第一行：时间周期类型 */}
          <div className="flex items-center gap-3">
            <MetaText as="span" className="w-16 shrink-0">
              周期类型
            </MetaText>
            <Select
              value={draftDimType}
              defaultValue="natural"
              onValueChange={(v) => setDraftDimType(v as "natural" | "custom")}
            >
              <SelectTrigger className="w-28 !h-7 min-h-0 text-xs border-gray-200 rounded-[4px] px-2 py-0 [&>svg]:h-3 [&>svg]:w-3">
                <SelectValue placeholder="自然周期" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="natural" className="text-xs">自然周期</SelectItem>
                <SelectItem value="custom" className="text-xs">自定义周期</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* 第二行：周期详情 */}
          {draftDimType === "natural" ? (
            <div className="flex items-center gap-3">
              <MetaText as="span" className="w-16 shrink-0">
                周期长度
              </MetaText>
              <Select
                value={draftPeriod}
                defaultValue="daily"
                onValueChange={(v) => setDraftPeriod(v as NaturalPeriod)}
              >
                <SelectTrigger className="w-20 !h-7 min-h-0 text-xs border-gray-200 rounded-[4px] px-2 py-0 [&>svg]:h-3 [&>svg]:w-3">
                  <SelectValue placeholder="每日" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="daily" className="text-xs">每日</SelectItem>
                  <SelectItem value="monthly" className="text-xs">每月</SelectItem>
                  <SelectItem value="yearly" className="text-xs">每年</SelectItem>
                </SelectContent>
              </Select>
            </div>
          ) : (
            <div className="space-y-2.5">
              <div className="flex items-start gap-3">
                <MetaText as="span" className="w-16 shrink-0 mt-0.5">
                  起止时间
                </MetaText>
                <HelperText as="span" className="leading-relaxed">
                  没有组织的新增用户默认以添加用户时间为开始时间，也可在用户管理页对单个用户进行再次调整；未匹配任何组织策略的组织以预设策略最新保存时间为开始时间；均无终止时间
                </HelperText>
              </div>
              <div className="flex items-center gap-3">
                <MetaText as="span" className="w-16 shrink-0">
                  刷新方式
                </MetaText>
                <Select
                  value={draftRefresh}
                  defaultValue="daily"
                  onValueChange={(v) => setDraftRefresh(v as CustomRefresh)}
                >
                  <SelectTrigger className="w-24 !h-7 min-h-0 text-xs border-gray-200 rounded-[4px] px-2 py-0 [&>svg]:h-3 [&>svg]:w-3">
                    <SelectValue placeholder="每日刷新" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="daily" className="text-xs">每日刷新</SelectItem>
                    <SelectItem value="monthly" className="text-xs">每月刷新</SelectItem>
                    <SelectItem value="yearly" className="text-xs">每年刷新</SelectItem>
                    <SelectItem value="none" className="text-xs">不刷新</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          )}

          {/* 第三行：配额 */}
          <div className="flex items-center gap-3">
            <MetaText as="span" className="w-16 shrink-0">
              配额
            </MetaText>
            <SegmentGroup className="h-7">
              <SegmentOption
                className="text-xs px-2.5"
                active={draftLimitMode === "unlimited"}
                onClick={() => setDraftLimitMode("unlimited")}
              >
                无限制
              </SegmentOption>
              <SegmentOption
                className="text-xs px-2.5"
                active={draftLimitMode === "custom"}
                onClick={() => setDraftLimitMode("custom")}
              >
                自定义
              </SegmentOption>
            </SegmentGroup>
            {draftLimitMode === "custom" && (
              <Input
                type="number"
                value={draftLimitValue}
                min={0}
                onKeyDown={blockInvalidKeys}
                onChange={handleLimitInput}
                className="bg-white border-gray-200 text-xs h-7 w-32 px-2 py-0 rounded-[4px]"
                placeholder="请输入数量"
                autoFocus
              />
            )}
          </div>

          {/* 底部操作按钮（hideActions 时由外部卡片按钮负责保存） */}
          {!hideActions && (
            <div className="flex justify-end gap-2 pt-2">
              <Button
                variant="claw-outline"
                size="claw-sm"
                onClick={() => setEditing(false)}
              >
                取消
              </Button>
              <Button
                variant="dialog-confirm"
                size="claw-sm"
                onClick={handleSave}
              >
                保存
              </Button>
            </div>
          )}
        </SurfaceInner>
      )}

      {/* 周期类型切换确认弹窗 */}
      <AlertDialog
        open={!!pendingTypeSwitch}
        onOpenChange={(open) => {
          if (!open) setPendingTypeSwitch(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>切换周期类型</AlertDialogTitle>
            <AlertDialogDescription asChild>
              <div className="space-y-2 text-sm text-[var(--text-secondary)] text-justify">
                <p>
                  切换到「
                  {pendingTypeSwitch === "natural" ? "自然周期" : "自定义周期"}
                  」后：
                </p>
                <ul className="list-disc pl-4 space-y-1">
                  <li>
                    所有组织策略将被清空，组织将回退使用预设策略；组织用户已消耗 Tokens 中，落在新周期内的部分会保留并计入，落在新周期外的部分不计入新周期统计。没有组织的用户不受影响，仍使用原有时间周期
                  </li>
                  {pendingTypeSwitch === "natural" ? (
                    <li>切换周期后，预设策略的上限值保留</li>
                  ) : (
                    <li>
                      切换周期后，预设策略的上限值保留；当前自然周期的「周期长度」会自动同步为自定义周期的「刷新方式」（例如：自然周期为每月 → 自定义周期为每月刷新）
                    </li>
                  )}
                </ul>
                <p className="text-[var(--text-danger)] font-medium">此操作不可撤销。</p>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction asChild>
              <Button variant="dialog-confirm" onClick={commitSave}>
                确定切换
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
});

export default TokenTimeDimensionEditor;

export type { TimeDimensionConfig, TokenLimit, NaturalPeriod, CustomRefresh };
