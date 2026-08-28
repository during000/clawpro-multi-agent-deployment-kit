/**
 * SelectPanel - 下拉筛选面板通用骨架
 *
 * 所有筛选/选择面板的统一三段式布局：
 *   1. Header（搜索框）
 *   2. Body（选项列表）
 *   3. Footer（计数/操作按钮）
 *
 * 交互模型：
 *   - commitMode="instant"：选中即生效，无确认按钮（如 ScopeFilterDropdown）
 *   - commitMode="confirm"：需点击确认才生效（如 TableHeaderFilter）
 *
 * 视觉规范：
 *   - 面板：rounded-[4px] shadow-[var(--shadow-popover)] border-none
 *   - 选项行：h-8 px-3 rounded-[6px]
 *   - 选中态：bg-[var(--bg-brand-selected)] text-blue-500 font-medium
 *   - Hover：hover:bg-[var(--bg-grey-hover)]
 *   - Footer：mx-2 border-t border-[#EAEEF4] py-2（左右8px与搜索框对齐，高度自适应）
 */
import React, { useState, useMemo, useCallback } from "react";
import { Search, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { MetaText } from "@/components/ui/Typography";
import { cn } from "@/lib/utils";

// ─── 类型定义 ────────────────────────────────────────────────────────────────

export type SelectPanelCommitMode = "instant" | "confirm";

export interface SelectPanelProps {
  /** 交互模式：instant=选中即生效，confirm=需点确认 */
  commitMode?: SelectPanelCommitMode;
  /** 是否显示搜索框（默认 true） */
  showSearch?: boolean;
  /** 搜索框 placeholder */
  searchPlaceholder?: string;
  /** 搜索值（受控） */
  searchValue?: string;
  /** 搜索值变化回调 */
  onSearchChange?: (value: string) => void;
  /** 列表区内容（slot） */
  children: React.ReactNode;
  /** Footer 左侧自定义内容（如计数文字） */
  footerLeft?: React.ReactNode;
  /** Footer 右侧自定义内容（优先级高于 commitMode 内置按钮） */
  footerRight?: React.ReactNode;
  /** 是否显示 Footer（默认 true） */
  showFooter?: boolean;
  /** 确认回调（commitMode="confirm" 时使用） */
  onConfirm?: () => void;
  /** 取消回调（commitMode="confirm" 时使用） */
  onCancel?: () => void;
  /** 确认按钮禁用 */
  confirmDisabled?: boolean;
  /** 面板额外 className */
  className?: string;
  /** 面板最大高度（默认 320px） */
  maxHeight?: number | string;
  /** 面板宽度 */
  width?: number | string;
}

// ─── 组件实现 ────────────────────────────────────────────────────────────────

export function SelectPanel({
  commitMode = "instant",
  showSearch = true,
  searchPlaceholder = "搜索...",
  searchValue: controlledSearch,
  onSearchChange,
  children,
  footerLeft,
  footerRight,
  showFooter = true,
  onConfirm,
  onCancel,
  confirmDisabled = false,
  className,
  maxHeight = 320,
  width,
}: SelectPanelProps) {
  // 内部搜索状态（非受控模式）
  const [internalSearch, setInternalSearch] = useState("");
  const searchValue = controlledSearch ?? internalSearch;
  const handleSearchChange = useCallback(
    (v: string) => {
      if (onSearchChange) onSearchChange(v);
      else setInternalSearch(v);
    },
    [onSearchChange]
  );

  // 是否显示 Footer
  const shouldShowFooter = showFooter && (commitMode === "confirm" || footerLeft || footerRight);

  return (
    <div
      className={cn("flex flex-col bg-white", className)}
      style={{ maxHeight, width }}
    >
      {/* Header: 搜索框 */}
      {showSearch && (
        <div className="p-2 pb-0 shrink-0">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-weak)] pointer-events-none" />
            <Input
              tenant
              className="h-8 pl-8 pr-7 text-sm"
              placeholder={searchPlaceholder}
              value={searchValue}
              onChange={(e) => handleSearchChange(e.target.value)}
            />
            {searchValue && (
              <button
                type="button"
                onClick={() => handleSearchChange("")}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-[var(--text-weak)] hover:text-[var(--text-secondary)]"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>
        </div>
      )}

      {/* Body: 列表区 */}
      <div
        className="flex-1 min-h-0 overflow-y-auto overscroll-contain p-2 space-y-0.5"
        style={{ scrollbarWidth: "thin", scrollbarColor: "transparent transparent" }}
        onMouseEnter={(e) => {
          (e.currentTarget as HTMLElement).style.scrollbarColor = "rgba(0,0,0,0.2) transparent";
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLElement).style.scrollbarColor = "transparent transparent";
        }}
        onWheel={(e) => e.stopPropagation()}
      >
        {children}
      </div>

      {/* Footer */}
      {shouldShowFooter && (
        <div className="shrink-0 mx-2 border-t border-[#EAEEF4] py-2 flex items-center justify-between">
          <div className="flex-1 min-w-0 truncate">
            {footerLeft}
          </div>
          <div className="flex items-center gap-1.5 shrink-0">
            {footerRight || (
              commitMode === "confirm" && (
                <>
                  <Button
                    variant="claw-outline"
                    size="sm"
                    className="text-xs h-7 px-2"
                    onClick={onCancel}
                  >
                    取消
                  </Button>
                  <Button
                    variant="dialog-confirm"
                    size="sm"
                    className="text-xs h-7 px-3"
                    disabled={confirmDisabled}
                    onClick={onConfirm}
                  >
                    确认
                  </Button>
                </>
              )
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ─── 选项行辅助组件 ──────────────────────────────────────────────────────────

export interface SelectPanelItemProps {
  /** 是否选中 */
  selected?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 点击回调 */
  onClick?: () => void;
  /** 内容 */
  children: React.ReactNode;
  /** 额外 className */
  className?: string;
}

/** 标准选项行：h-8 px-3 rounded-[6px]，选中态+hover态已内置 */
export function SelectPanelItem({
  selected = false,
  disabled = false,
  onClick,
  children,
  className,
}: SelectPanelItemProps) {
  return (
    <div
      className={cn(
        "relative flex items-center gap-2 h-8 px-3 rounded-[6px] cursor-pointer transition-colors text-sm",
        selected
          ? "bg-[var(--bg-brand-selected)] text-blue-500 font-medium"
          : "text-[var(--text-title)] hover:bg-[var(--bg-grey-hover)]",
        disabled && "opacity-40 cursor-not-allowed",
        className
      )}
      onClick={disabled ? undefined : onClick}
    >
      {children}
    </div>
  );
}

// ─── 空状态辅助组件 ──────────────────────────────────────────────────────────

export function SelectPanelEmpty({ text = "未找到匹配项" }: { text?: string }) {
  return (
    <div className="py-3 text-center">
      <MetaText tone="weak">{text}</MetaText>
    </div>
  );
}

export default SelectPanel;
