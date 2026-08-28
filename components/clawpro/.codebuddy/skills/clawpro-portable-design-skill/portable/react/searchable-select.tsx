import React, { useEffect, useRef, useState } from "react";
import ReactDOM from "react-dom";

/**
 * SearchableSelect Option 类型定义
 */
export interface PortableSearchableSelectOption {
  value: string | number;
  label: React.ReactNode;
  searchText?: string;
  disabled?: boolean;
}

/**
 * SearchableSelect Props 类型定义
 */
export interface PortableSearchableSelectProps {
  options: PortableSearchableSelectOption[];
  value?: string | number | null;
  onChange?: (value: string | number | null) => void;
  placeholder?: string;
  searchable?: boolean;
  searchPlaceholder?: string;
  onSearch?: (query: string) => void;
  showCount?: boolean;
  countTemplate?: (total: number, filtered: number) => string;
  clearable?: boolean;
  onClear?: () => void;
  align?: "start" | "center" | "end";
  panelWidth?: number | string;
  disabled?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  triggerClassName?: string;
  panelClassName?: string;
  className?: string;
}

/**
 * PortableSearchableSelect 组件
 *
 * 支持搜索功能的下拉选择器，面板通过 Portal 渲染到 document.body
 */
export const PortableSearchableSelect = React.forwardRef<
  HTMLDivElement,
  PortableSearchableSelectProps
>(
  (
    {
      options,
      value,
      onChange,
      placeholder = "请选择",
      searchable = true,
      searchPlaceholder = "搜索...",
      onSearch,
      showCount = true,
      countTemplate = (total, filtered) => `共 ${filtered} 条`,
      clearable = false,
      onClear,
      align = "start",
      panelWidth,
      disabled = false,
      open: controlledOpen,
      onOpenChange,
      triggerClassName,
      panelClassName,
      className,
    },
    ref
  ) => {
    const [isOpen, setIsOpen] = useState(false);
    const [searchQuery, setSearchQuery] = useState("");
    const triggerRef = useRef<HTMLDivElement>(null);
    const searchInputRef = useRef<HTMLInputElement>(null);
    const panelRef = useRef<HTMLDivElement>(null);
    const [panelPosition, setPanelPosition] = useState<{
      top: number;
      left: number;
      width: number;
    } | null>(null);

    // 受控/非受控开合状态
    const isControlled = controlledOpen !== undefined;
    const open = isControlled ? controlledOpen : isOpen;

    // 获取已选项目
    const selectedOption = options.find((opt) => opt.value === value);
    const selectedLabel = selectedOption?.label;

    // 过滤选项
    const filteredOptions = searchQuery
      ? options.filter((opt) => {
          const searchText = opt.searchText || String(opt.label || "");
          return searchText.toLowerCase().includes(searchQuery.toLowerCase());
        })
      : options;

    // 处理打开/关闭
    const handleToggle = () => {
      if (disabled) return;

      const newOpen = !open;
      if (isControlled) {
        onOpenChange?.(newOpen);
      } else {
        setIsOpen(newOpen);
      }
    };

    // 处理选项点击
    const handleSelectOption = (option: PortableSearchableSelectOption) => {
      if (option.disabled) return;

      onChange?.(option.value);
      setSearchQuery("");

      if (isControlled) {
        onOpenChange?.(false);
      } else {
        setIsOpen(false);
      }
    };

    // 处理清除
    const handleClear = (e: React.MouseEvent) => {
      e.stopPropagation();
      onChange?.(null);
      onClear?.();
    };

    // 处理搜索变化
    const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const query = e.target.value;
      setSearchQuery(query);
      onSearch?.(query);
    };

    // 点击外部关闭
    useEffect(() => {
      if (!open) return;

      const handleClickOutside = (e: MouseEvent) => {
        const target = e.target as Node;
        if (
          triggerRef.current &&
          !triggerRef.current.contains(target) &&
          panelRef.current &&
          !panelRef.current.contains(target)
        ) {
          if (isControlled) {
            onOpenChange?.(false);
          } else {
            setIsOpen(false);
          }
        }
      };

      document.addEventListener("mousedown", handleClickOutside);
      return () => {
        document.removeEventListener("mousedown", handleClickOutside);
      };
    }, [open, isControlled, onOpenChange]);

    // 计算 Panel 位置
    useEffect(() => {
      if (!open || !triggerRef.current) {
        setPanelPosition(null);
        return;
      }

      const calculatePosition = () => {
        const rect = triggerRef.current!.getBoundingClientRect();
        const width = panelWidth ?? rect.width;

        let left = rect.left;
        if (align === "center") {
          left = rect.left + rect.width / 2 - (typeof width === "number" ? width / 2 : 0);
        } else if (align === "end") {
          left = rect.right - (typeof width === "number" ? width : 0);
        }

        setPanelPosition({
          top: rect.bottom + 4,
          left,
          width: typeof width === "number" ? width : rect.width,
        });
      };

      calculatePosition();
      const resizeObserver = new ResizeObserver(calculatePosition);
      resizeObserver.observe(triggerRef.current);

      return () => {
        resizeObserver.disconnect();
      };
    }, [open, align, panelWidth]);

    // 打开时自动聚焦搜索框
    useEffect(() => {
      if (open && searchable && searchInputRef.current) {
        setTimeout(() => searchInputRef.current?.focus(), 0);
      }
    }, [open, searchable]);

    // 触发器类名
    const triggerClasses = [
      "cp-searchable-select__trigger",
      open && "cp-searchable-select__trigger--open",
      disabled && "cp-searchable-select__trigger--disabled",
      triggerClassName,
    ]
      .filter(Boolean)
      .join(" ");

    // 面板内容
    const panelContent = (
      <div
        ref={panelRef}
        className={[
          "cp-searchable-select__panel",
          panelClassName,
        ]
          .filter(Boolean)
          .join(" ")}
        style={{
          position: "fixed",
          top: `${panelPosition?.top}px`,
          left: `${panelPosition?.left}px`,
          width:
            typeof panelWidth === "number"
              ? `${panelWidth}px`
              : panelWidth || `${panelPosition?.width}px`,
          zIndex: 1000,
          minWidth: 0,
        }}
      >
        {/* 搜索框 */}
        {searchable && (
          <div className="cp-searchable-select__search">
            <div className="cp-input-search">
              <svg
                className="cp-input-search__icon"
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <circle cx="6.5" cy="6.5" r="5" stroke="currentColor" strokeWidth="1.5" />
                <path
                  d="M10 10L14 14"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                />
              </svg>
              <input
                ref={searchInputRef}
                type="text"
                className="cp-input cp-searchable-select__search-input"
                placeholder={searchPlaceholder}
                value={searchQuery}
                onChange={handleSearchChange}
              />
            </div>
          </div>
        )}

        {/* 选项列表 */}
        <div className="cp-searchable-select__list">
          {filteredOptions.length > 0 ? (
            filteredOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                className={[
                  "cp-searchable-select__option",
                  value === option.value && "cp-searchable-select__option--selected",
                ]
                  .filter(Boolean)
                  .join(" ")}
                onClick={() => handleSelectOption(option)}
                disabled={option.disabled}
              >
                <span>{option.label}</span>
                {value === option.value && (
                  <svg
                    className="cp-searchable-select__check"
                    width="16"
                    height="16"
                    viewBox="0 0 16 16"
                    fill="none"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      d="M13.5 4L6 11.5L2.5 8"
                      stroke="currentColor"
                      strokeWidth="1.5"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                )}
              </button>
            ))
          ) : (
            <div className="cp-searchable-select__empty">
              {searchQuery ? "未找到匹配项" : "暂无选项"}
            </div>
          )}
        </div>

        {/* 计数信息 */}
        {showCount && filteredOptions.length > 0 && (
          <div className="cp-searchable-select__count">
            {countTemplate(options.length, filteredOptions.length)}
          </div>
        )}
      </div>
    );

    return (
      <div
        ref={ref}
        className={["cp-searchable-select", className].filter(Boolean).join(" ")}
      >
        {/* 触发器 */}
        <div
          ref={triggerRef}
          className={triggerClasses}
          onClick={handleToggle}
          role="combobox"
          aria-expanded={open}
          aria-disabled={disabled}
          tabIndex={disabled ? -1 : 0}
        >
          <span className="cp-select__value">
            {selectedLabel || placeholder}
          </span>

          <div className="cp-searchable-select__actions">
            {clearable && value !== null && value !== undefined && (
              <button
                type="button"
                className="cp-searchable-select__clear"
                onClick={handleClear}
                aria-label="清除选择"
              >
                <svg
                  width="16"
                  height="16"
                  viewBox="0 0 16 16"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    d="M12 4L4 12M4 4L12 12"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                  />
                </svg>
              </button>
            )}
            <svg
              className="cp-searchable-select__chevron"
              width="16"
              height="16"
              viewBox="0 0 16 16"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M12 6L8 10L4 6"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </div>
        </div>

        {/* Portal 面板 */}
        {open && typeof document !== "undefined" && ReactDOM.createPortal(
          panelContent,
          document.body
        )}
      </div>
    );
  }
);

PortableSearchableSelect.displayName = "PortableSearchableSelect";
