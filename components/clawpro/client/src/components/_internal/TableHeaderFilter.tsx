/**
 * TableHeaderFilter - 表头筛选组件（普通多选）
 *
 * 封装筛选图标 + 下拉面板，适用于表格列头多选筛选场景。
 * 功能：
 *   - 全选/全不选切换（带分割线）
 *   - Checkbox 多选，选中态 bg-[var(--bg-brand-selected)]
 *   - 底部已选计数 + 重置/确认按钮
 *   - 滚动条仅交互时显示
 *   - 筛选激活时图标变蓝
 *
 * 视觉规范：
 *   - 选项行：h-8 px-3 rounded-[6px]
 *   - Footer：mx-2 border-t border-[#EAEEF4]
 *   - 阴影：shadow-[var(--shadow-popover)]
 */
import { useState } from "react";
import { Filter } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { MetaText } from "@/components/ui/Typography";

// ─── 类型定义 ────────────────────────────────────────────────────────────────

export interface FilterOption {
  /** 唯一标识 */
  value: string;
  /** 展示文字 */
  label: string;
  /** 是否禁用 */
  disabled?: boolean;
}

export interface TableHeaderFilterProps {
  /** 列标题 */
  title: string;
  /** 筛选选项列表 */
  options: FilterOption[];
  /** 当前选中的值集合 */
  selectedValues: Set<string>;
  /** 确认回调 */
  onConfirm: (values: Set<string>) => void;
  /** 面板宽度（默认 220px） */
  panelWidth?: number;
  /** 面板对齐方式 */
  align?: "start" | "center" | "end";
  /** "全部"选项文字（默认 "全部"） */
  allLabel?: string;
  /** 已选全部时的计数文案（默认 "已选全部"） */
  allSelectedText?: string;
  /** 已选 N 项模板（默认 "已选 {count} 项"） */
  selectedCountTemplate?: string;
}

// ─── 组件实现 ────────────────────────────────────────────────────────────────

export function TableHeaderFilter({
  title,
  options,
  selectedValues,
  onConfirm,
  panelWidth = 220,
  align = "start",
  allLabel = "全部",
  allSelectedText = "已选全部",
  selectedCountTemplate = "已选 {count} 项",
}: TableHeaderFilterProps) {
  const [open, setOpen] = useState(false);
  const [tempSelected, setTempSelected] = useState<Set<string>>(new Set(selectedValues));

  // 打开时同步外部状态
  const handleOpenChange = (v: boolean) => {
    if (v) {
      setTempSelected(new Set(selectedValues));
    }
    setOpen(v);
  };

  const allValues = options.map((o) => o.value);
  const isAllSelected = allValues.length > 0 && allValues.every((v) => tempSelected.has(v));
  const isFiltered = selectedValues.size > 0 && selectedValues.size < options.length;

  const handleToggle = (value: string, checked: boolean) => {
    const next = new Set(tempSelected);
    if (checked) {
      next.add(value);
    } else {
      next.delete(value);
    }
    setTempSelected(next);
  };

  const handleToggleAll = (checked: boolean) => {
    if (checked) {
      setTempSelected(new Set(allValues));
    } else {
      setTempSelected(new Set());
    }
  };

  const handleReset = () => {
    setTempSelected(new Set(allValues));
  };

  const handleConfirm = () => {
    onConfirm(tempSelected);
    setOpen(false);
  };

  const countText = tempSelected.size === options.length
    ? allSelectedText
    : selectedCountTemplate.replace("{count}", String(tempSelected.size));

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <button className="flex items-center gap-1 group/filter">
          <span>{title}</span>
          <Filter className={`w-3.5 h-3.5 transition-colors ${isFiltered ? "text-[#355EF1]" : "text-[var(--text-weak)] group-hover/filter:text-[var(--text-muted)]"}`} />
        </button>
      </PopoverTrigger>
      <PopoverContent
        className="p-0 rounded-[4px] border-none shadow-[var(--shadow-popover)]"
        style={{ width: panelWidth }}
        align={align}
        side="bottom"
      >
        {/* 选项列表 */}
        <div
          className="p-2 space-y-0.5 max-h-64 overflow-y-auto"
          style={{ scrollbarWidth: "thin", scrollbarColor: "transparent transparent" }}
          onMouseEnter={(e) => { (e.currentTarget as HTMLElement).style.scrollbarColor = "rgba(0,0,0,0.2) transparent"; }}
          onMouseLeave={(e) => { (e.currentTarget as HTMLElement).style.scrollbarColor = "transparent transparent"; }}
          onWheel={(e) => e.stopPropagation()}
        >
          {/* 全部选项 */}
          <label
            className={`flex items-center gap-2 h-8 px-3 rounded-[6px] cursor-pointer transition-colors ${isAllSelected ? "bg-[var(--bg-brand-selected)]" : "hover:bg-[var(--bg-grey-hover)]"}`}
          >
            <Checkbox
              checked={isAllSelected}
              onCheckedChange={(v) => handleToggleAll(!!v)}
            />
            <span className="text-sm text-[var(--text-title)]">{allLabel}</span>
          </label>
          <div className="mx-2 my-1 border-t border-[#EAEEF4]" />
          {/* 子选项 */}
          {options.map((opt) => {
            const checked = tempSelected.has(opt.value);
            return (
              <label
                key={opt.value}
                className={`flex items-center gap-2 h-8 px-3 rounded-[6px] transition-colors ${
                  opt.disabled
                    ? "cursor-not-allowed opacity-40"
                    : checked
                      ? "cursor-pointer bg-[var(--bg-brand-selected)]"
                      : "cursor-pointer hover:bg-[var(--bg-grey-hover)]"
                }`}
              >
                <Checkbox
                  checked={checked}
                  onCheckedChange={(v) => handleToggle(opt.value, !!v)}
                  disabled={opt.disabled}
                />
                <span className={`text-sm ${opt.disabled ? "text-[var(--text-weak)]" : "text-[var(--text-title)]"}`}>
                  {opt.label}
                </span>
              </label>
            );
          })}
        </div>
        {/* Footer */}
        <div className="mx-2 border-t border-[#EAEEF4] py-2 flex items-center justify-between">
          <MetaText>{countText}</MetaText>
          <div className="flex items-center gap-1.5">
            <Button variant="claw-outline" size="sm" className="text-xs h-7 px-2" onClick={handleReset}>
              重置
            </Button>
            <Button variant="dialog-confirm" size="sm" className="text-xs h-7 px-3" onClick={handleConfirm}>
              确认
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export default TableHeaderFilter;
