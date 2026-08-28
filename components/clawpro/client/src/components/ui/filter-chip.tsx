/**
 * FilterChip 可选择标签 / 分类筛选组件
 * ─────────────────────────────────────────────────────────────────
 * 用于分类筛选场景：一组可点击的标签/胶囊，单选激活。
 *
 * 使用场景：
 *   - 公共技能库 / 企业技能库分类筛选
 *   - 技能初始包弹窗分类筛选
 *   - 任何需要单选 Tag/Chip 筛选的场景
 *
 * 视觉规范：
 *   - Active：黑底白字 `#0A0A0A` + 同色边框
 *   - Inactive：白底 + `#EAEEF4` 边框 + `#525252` 文字
 *   - Hover（inactive）：边框变深 `#0A0A0A` + 文字变深
 *
 * 用法：
 *   <FilterChipGroup
 *     items={[{ id: "all", label: "全部" }, { id: "dev", label: "开发工具" }]}
 *     value="all"
 *     onChange={(id) => setCategory(id)}
 *   />
 */
import { cn } from "@/lib/utils";

export interface FilterChipItem {
  id: string;
  label: string;
}

/** 尺寸变体：default 常规；sm 最小（紧凑场景，如引导体系体验面板） */
export type FilterChipSize = "default" | "sm";

const FILTER_CHIP_SIZE_CLASSES: Record<FilterChipSize, string> = {
  default: "px-4 py-1.5 rounded-[4px] text-sm",
  sm: "px-2 py-0.5 rounded-[4px] text-[10px]",
};

export interface FilterChipGroupProps {
  /** 标签列表 */
  items: FilterChipItem[];
  /** 当前选中的 id */
  value: string;
  /** 选中变化回调 */
  onChange: (id: string) => void;
  /** 尺寸：default（默认）/ sm（最小） */
  size?: FilterChipSize;
  /** 额外 className（作用于外层容器） */
  className?: string;
}

export function FilterChipGroup({
  items,
  value,
  onChange,
  size = "default",
  className,
}: FilterChipGroupProps) {
  return (
    <div className={cn("flex items-center gap-1.5 flex-wrap", className)}>
      {items.map((item) => {
        const isActive = value === item.id;
        return (
          <button
            key={item.id}
            type="button"
            onClick={() => onChange(item.id)}
            className={cn(
              "font-normal transition-colors border",
              FILTER_CHIP_SIZE_CLASSES[size],
              isActive
                ? "bg-[#020617] text-white border-[#020617]"
                : "bg-white text-gray-950 border-[#EAEEF4] hover:border-gray-950"
            )}
          >
            {item.label}
          </button>
        );
      })}
    </div>
  );
}

/**
 * FilterChip 单个标签（独立使用时）
 */
export interface FilterChipProps {
  /** 是否激活 */
  active?: boolean;
  /** 点击回调 */
  onClick?: () => void;
  /** 标签内容 */
  children: React.ReactNode;
  /** 尺寸：default（默认）/ sm（最小） */
  size?: FilterChipSize;
  /** 额外 className */
  className?: string;
}

export function FilterChip({
  active = false,
  onClick,
  children,
  size = "default",
  className,
}: FilterChipProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "font-normal transition-colors border",
        FILTER_CHIP_SIZE_CLASSES[size],
        active
          ? "bg-[#020617] text-white border-[#020617]"
          : "bg-white text-gray-950 border-[#EAEEF4] hover:border-gray-950",
        className
      )}
    >
      {children}
    </button>
  );
}
