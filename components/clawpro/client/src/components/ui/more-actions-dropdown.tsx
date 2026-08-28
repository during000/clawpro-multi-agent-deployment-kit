/**
 * MoreActionsDropdown — 更多操作下拉面板
 *
 * 基于企业技能库「更多操作」下拉面板样式封装的通用组件。
 * 适用于表格操作列、卡片右上角三点菜单等场景。
 *
 * ⚠️ 强制约束：每个菜单项必须提供 icon（图标为必填项），不允许纯文字菜单项。
 *
 * ═══════════════════════════════════════════════════════════
 * Token 使用规范（基于 SKILL-GLOBAL-COMPONENTS.md §17）
 * ═══════════════════════════════════════════════════════════
 *
 * 【面板容器】（由 DropdownMenuContent 承担，无需额外 className）
 * - 背景: var(--popover) → bg-popover
 * - 圆角: rounded-[8px]（DropdownMenuContent 内置）
 * - 最小宽度: min-w-[8rem]（内置）
 * - padding: p-1（内置）
 * - 阴影: 内置三层阴影
 *
 * 【菜单项 (DropdownMenuItem)】（内置样式，无需额外 className）
 * - padding: py-1.5 px-2（内置）
 * - 字号: text-sm (14px)（内置）
 * - 文字色: var(--foreground) → text-foreground（继承自 popover-foreground）
 * - hover/focus 背景: var(--bg-hover)（普通操作与危险操作统一使用同一背景色 token）
 * - disabled 文字色: 内置 data-[disabled]:!text-gray-300
 * - disabled 图标色: 内置 [&[data-disabled]_svg]:!text-gray-300
 * - disabled 交互: pointer-events-none（内置）
 *
 * 【图标】（必填）
 * - 尺寸: size-4 (16×16px)（由 [&_svg:not([class*='size-'])]:size-4 内置）
 * - 间距: gap-2 (8px)（由 DropdownMenuItem flex gap-2 内置）
 * - 普通操作图标色: var(--muted-foreground) → text-muted-foreground
 *   （由内置 [&_svg:not([class*='text-'])]:text-gray-600 处理）
 * - 危险操作图标色: var(--destructive) → text-destructive（与文字同色）
 *   （由 data-[variant=destructive]:*:[svg]:!text-destructive 内置处理）
 * - disabled 图标色: text-gray-300（由内置 [&[data-disabled]_svg]:!text-gray-300 处理）
 *
 * 【操作性质分类】
 * - 普通操作 (variant="default"): 默认 foreground 文字 + muted-foreground 图标
 *   示例: 安全检测、下载、卸载
 * - 危险操作 (variant="destructive"): destructive 红色文字 + 红色图标
 *   示例: 删除
 *   → 直接通过 DropdownMenuItem variant="destructive" prop 启用
 *
 * 【禁用状态】
 * - 组件级禁用: disabled prop → 触发按钮整体禁用，不可打开面板
 * - 菜单项级禁用: item.disabled → 该项不可点击，文字+图标变为 text-gray-300
 *
 * 【触发按钮】
 * - 表格行内: variant="link" 文字按钮 → 文字 "更多"
 * - 卡片/独立: 无边框竖向图标按钮 → <MoreVertical />
 *
 * ═══════════════════════════════════════════════════════════
 *
 * 用法示例:
 * ```tsx
 * <MoreActionsDropdown
 *   items={[
 *     { label: "安全检测", icon: ScanSearch, onClick: handleScan },
 *     { label: "下载", icon: Download, onClick: handleDownload },
 *     { label: "卸载", icon: PackageX, onClick: handleUninstall, disabled: true },
 *     { label: "删除", icon: Trash2, onClick: handleDelete, variant: "destructive" },
 *   ]}
 * />
 *
 * // 整体禁用
 * <MoreActionsDropdown disabled items={[...]} />
 * ```
 */
import * as React from "react";
import { MoreVertical } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

// ─── 类型定义 ───

export interface MoreActionItem {
  /** 菜单项显示文字 */
  label: string;
  /** lucide-react 图标组件（必填，每项必须带图标） */
  icon: React.ComponentType<{ className?: string }>;
  /** 点击回调 */
  onClick: () => void;
  /** 操作性质：default=普通操作，destructive=危险操作(红色) */
  variant?: "default" | "destructive";
  /** 是否禁用该菜单项 */
  disabled?: boolean;
  /** 禁用时的提示文案（hover tooltip），仅在 disabled 为 true 时生效 */
  disabledReason?: string;
  /** 是否在此项之前插入分割线 */
  separatorBefore?: boolean;
  /** 是否隐藏此项 */
  hidden?: boolean;
}

export interface MoreActionsDropdownProps {
  /** 菜单项列表 */
  items: MoreActionItem[];
  /** 触发按钮类型：icon=图标按钮(默认), text=文字"更多"按钮 */
  triggerType?: "icon" | "text";
  /** 自定义触发按钮（完全自定义时使用） */
  trigger?: React.ReactNode;
  /** 面板对齐方式 */
  align?: "start" | "center" | "end";
  /** 面板宽度 className，如 "w-40" */
  contentClassName?: string;
  /** 触发按钮额外样式 */
  triggerClassName?: string;
  /** 触发按钮无障碍标签 */
  ariaLabel?: string;
  /** 阻止事件冒泡（卡片内使用时需要） */
  stopPropagation?: boolean;
  /** 整体禁用：触发按钮不可点击，不可打开面板 */
  disabled?: boolean;
}

// ─── 组件实现 ───

export function MoreActionsDropdown({
  items,
  triggerType = "icon",
  trigger,
  align = "end",
  contentClassName,
  triggerClassName,
  ariaLabel = "更多操作",
  stopPropagation = false,
  disabled = false,
}: MoreActionsDropdownProps) {
  // 过滤隐藏项
  const visibleItems = items.filter((item) => !item.hidden);

  if (visibleItems.length === 0) return null;

  const handleTriggerClick = (e: React.MouseEvent) => {
    if (stopPropagation) e.stopPropagation();
  };

  // 渲染触发按钮
  const renderTrigger = () => {
    if (trigger) return trigger;

    if (triggerType === "text") {
      return (
        <Button
          variant="link"
          className={cn(triggerClassName)}
          aria-label={ariaLabel}
          onClick={handleTriggerClick}
          disabled={disabled}
        >
          更多
        </Button>
      );
    }

    return (
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          "size-8 rounded-[var(--radius-lg)] border-0 p-0 text-[var(--text-weak)] shadow-none hover:bg-[var(--bg-grey-hover-subtle)] hover:text-[var(--text-emphasis)]",
          triggerClassName,
        )}
        aria-label={ariaLabel}
        onClick={handleTriggerClick}
        disabled={disabled}
      >
        <MoreVertical className="size-3.5" />
      </Button>
    );
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        {renderTrigger()}
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className={cn(contentClassName)}>
        {visibleItems.map((item, index) => (
          <React.Fragment key={`${item.label}-${index}`}>
            {item.separatorBefore && <DropdownMenuSeparator />}
            {item.disabled && item.disabledReason ? (
              <Tooltip delayDuration={200}>
                <TooltipTrigger asChild>
                  {/* 用一层 span 包裹，避开 DropdownMenuItem disabled 的 pointer-events: none
                   * 让 Radix Tooltip 仍能接收 mouse enter 事件触发提示。
                   * 视觉上 DropdownMenuItem 自身仍保留 disabled 灰色样式。 */}
                  <span
                    className="block"
                    onPointerDown={(e) => e.stopPropagation()}
                  >
                    <DropdownMenuItem
                      onSelect={(e) => e.preventDefault()}
                      disabled
                      variant={item.variant === "destructive" ? "destructive" : "default"}
                    >
                      <item.icon className="size-4" />
                      {item.label}
                    </DropdownMenuItem>
                  </span>
                </TooltipTrigger>
                <TooltipContent side="left">
                  {item.disabledReason}
                </TooltipContent>
              </Tooltip>
            ) : (
              <DropdownMenuItem
                onClick={(e) => {
                  if (stopPropagation) e.stopPropagation();
                  item.onClick();
                }}
                disabled={item.disabled}
                variant={item.variant === "destructive" ? "destructive" : "default"}
              >
                <item.icon className="size-4" />
                {item.label}
              </DropdownMenuItem>
            )}
          </React.Fragment>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
