/**
 * Portable SearchFilterBar — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 SearchFilterBar 时的可移植兜底实现。
 *  - 不依赖 shadcn / Radix / Tailwind；样式由 portable/css/search-filter-bar.css 提供。
 *  - 视觉规范（spec/component-specs/search-filter-bar.md §3）：
 *      左侧：搜索输入（带放大镜 icon）+ 任意筛选触发器
 *      右侧：刷新 / 主操作按钮
 *      所有控件 h-9（36px）+ 4px 圆角 + 蓝灰描边
 *      全部走 flex + gap-3 成组（禁单独 margin）
 *      下方可选：chips 区（已选筛选条件）+ 清除全部按钮
 *
 *  - 槽位：
 *      filters          左侧筛选（搜索框 + 状态选择 + 时间选择 等）
 *      secondaryActions 右侧次级操作（刷新、导出等）
 *      primaryAction    右侧主操作（创建 / 导入等）
 *      chips            已选筛选条件数组（可选）
 *      onClearAllChips  清除全部 chips 回调（可选）
 *      onRefresh        刷新按钮回调（可选）
 *      refreshLoading   刷新按钮加载态（可选）
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/search-filter-bar.css";
 *
 * 用法（基础）：
 *   <PortableSearchFilterBar
 *     searchValue={s} onSearchChange={setS}
 *     searchPlaceholder="搜索关键字"
 *     filters={
 *       <>
 *         <PortableSelect value={status} onChange={...}>...</PortableSelect>
 *         <PortableDatePicker />
 *       </>
 *     }
 *     secondaryActions={<button>刷新</button>}
 *     primaryAction={<button>创建</button>}
 *   />
 *
 * 用法（含 Chips）：
 *   <PortableSearchFilterBar
 *     searchValue={s}
 *     onSearchChange={setS}
 *     filters={...}
 *     chips={[
 *       { key: "status-active", label: "状态：活跃", onRemove: () => removeStatus() },
 *       { key: "role-admin", label: "角色：管理员", onRemove: () => removeRole() }
 *     ]}
 *     onClearAllChips={() => clearAllFilters()}
 *     secondaryActions={...}
 *     primaryAction={...}
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export interface ChipItem {
  /** 唯一标识（用于 key） */
  key: string;
  /** Chip 标签内容 */
  label: React.ReactNode;
  /** 删除此 chip 的回调 */
  onRemove: () => void;
}

export interface PortableSearchFilterBarProps {
  /** 搜索框受控 value */
  searchValue?: string;
  onSearchChange?: (value: string) => void;
  searchPlaceholder?: string;
  /** 是否显示搜索框（默认 true） */
  showSearch?: boolean;
  /** 左侧筛选槽（搜索框右侧） */
  filters?: React.ReactNode;
  /** 右侧次级操作槽（刷新、导出等） */
  secondaryActions?: React.ReactNode;
  /** 右侧主操作槽（创建、导入等），通常放在 secondaryActions 右侧 */
  primaryAction?: React.ReactNode;
  /** 已选筛选条件数组，若不为空则显示 chips 区 */
  chips?: ChipItem[];
  /** 清除全部 chips 的回调 */
  onClearAllChips?: () => void;
  /** 刷新按钮点击回调（若提供则显示内置刷新按钮） */
  onRefresh?: () => void;
  /** 刷新按钮是否处于加载状态 */
  refreshLoading?: boolean;
  className?: string;
}

/**
 * 刷新 icon（SVG）
 * 简单的循环箭头，可由业务代码替换为其他 icon
 */
function RefreshIcon({ isLoading }: { isLoading?: boolean }) {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 16 16"
      className={isLoading ? "animate-spin" : ""}
      fill="none"
      style={{
        width: 16,
        height: 16,
        animation: isLoading ? "spin 1s linear infinite" : "none",
      }}
    >
      <path
        d="M13.5 2.5a6 6 0 0 0-10.18.46"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M2.5 13.5a6 6 0 0 0 10.18-.46"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

/**
 * 关闭 icon（X）
 */
function CloseIcon() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 16 16"
      fill="none"
      style={{ width: 12, height: 12 }}
    >
      <path
        d="m12 4-8 8M4 4l8 8"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

export function PortableSearchFilterBar({
  searchValue = "",
  onSearchChange,
  searchPlaceholder = "搜索关键字",
  showSearch = true,
  filters,
  secondaryActions,
  primaryAction,
  chips,
  onClearAllChips,
  onRefresh,
  refreshLoading = false,
  className = "",
}: PortableSearchFilterBarProps) {
  const merged = ["cp-search-filter-bar", className].filter(Boolean).join(" ");
  const hasChips = chips && chips.length > 0;

  // CSS animation 样式：只在 refreshLoading 时需要
  const spinStyle = refreshLoading ? (
    <style>{`
      @keyframes spin {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
      }
      .animate-spin { animation: spin 1s linear infinite; }
    `}</style>
  ) : null;

  return (
    <>
      {spinStyle}
      <div className={merged}>
        {/* 左侧：搜索 + 筛选 */}
        <div className="cp-search-filter-bar__left">
          {showSearch && (
            <label className="cp-search-filter-bar__search">
              <svg
                aria-hidden="true"
                viewBox="0 0 16 16"
                className="cp-search-filter-bar__search-icon"
                fill="none"
              >
                <path
                  d="M11.5 11.5 14 14"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                />
                <circle cx="7" cy="7" r="4.5" stroke="currentColor" strokeWidth="1.5" />
              </svg>
              <input
                className="cp-search-filter-bar__input"
                value={searchValue}
                onChange={(e) => onSearchChange?.(e.target.value)}
                placeholder={searchPlaceholder}
              />
            </label>
          )}
          {filters && <div className="cp-search-filter-bar__filter">{filters}</div>}
        </div>

        {/* 右侧：刷新 + 次级 + 主操作 */}
        {(onRefresh || secondaryActions || primaryAction) && (
          <div className="cp-search-filter-bar__actions">
            {onRefresh && (
              <button
                className="cp-search-filter-bar__refresh"
                onClick={onRefresh}
                disabled={refreshLoading}
                title="刷新"
                aria-label="刷新"
              >
                <RefreshIcon isLoading={refreshLoading} />
              </button>
            )}
            {secondaryActions}
            {primaryAction}
          </div>
        )}
      </div>

      {/* 下方：Chips 区（若有已选条件） */}
      {hasChips && (
        <div className="cp-search-filter-bar__chips">
          {chips.map((chip) => (
            <div key={chip.key} className="cp-search-filter-bar__chip">
              <span>{chip.label}</span>
              <button
                className="cp-search-filter-bar__chip-remove"
                onClick={chip.onRemove}
                title={`删除 ${chip.label}`}
                aria-label={`删除 ${chip.label}`}
              >
                <CloseIcon />
              </button>
            </div>
          ))}
          {onClearAllChips && (
            <button className="cp-search-filter-bar__clear-chips" onClick={onClearAllChips}>
              清除全部
            </button>
          )}
        </div>
      )}
    </>
  );
}
