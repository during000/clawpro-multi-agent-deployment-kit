/**
 * SegmentedTabs - 分段胶囊 Tab（公共组件）
 *
 * 设计语言：
 *   ┌───────────────────────────────────────────────────────────────────────┐
 *   │ 容器：灰色胶囊背景（#F5F5F5），内边距 4px，圆角 4px。                  │
 *   │ Tab：胶囊型按钮，激活时白色背景 + segment 阴影；                       │
 *   │       非激活时透明背景 + muted 文字。                                  │
 *   │ 字体：BodyMedium（sm + medium）；激活态 primary，非激活态 muted。       │
 *   └───────────────────────────────────────────────────────────────────────┘
 *
 * 使用：
 *   <SegmentedTabs
 *     tabs={[{ id: 'overview', label: '概述' }, { id: 'files', label: '文件列表' }]}
 *     active={activeTab}
 *     onChange={setActiveTab}
 *     ariaLabel="技能详情 Tab 切换"
 *   />
 */
import * as React from "react";
import { cn } from "@/lib/utils";
import { BodyMedium } from "./Typography";

export interface SegmentedTabItem {
  id: string;
  label: React.ReactNode;
}

export interface SegmentedTabsProps {
  tabs: SegmentedTabItem[];
  active: string;
  onChange: (id: string) => void;
  ariaLabel?: string;
  className?: string;
  /**
   * 是否占满父容器宽度。
   * - true：容器变 flex（覆盖默认 inline-flex），每个 option 变 flex-1 等分。
   *   适用于弹窗/卡片内的全宽切换（与 DialogBody 边距对齐）。
   * - false（默认）：保留 w-fit，按钮贴合文字宽度。
   */
  fullWidth?: boolean;
}

export function SegmentedTabs({
  tabs,
  active,
  onChange,
  ariaLabel,
  className,
  fullWidth = false,
}: SegmentedTabsProps) {
  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      className={cn(
        "self-start items-center gap-1 p-1 rounded-[4px]",
        fullWidth ? "flex w-full" : "inline-flex w-fit",
        className
      )}
      style={{ background: "#F5F5F5" }}
    >
      {tabs.map((t) => {
        const isActive = t.id === active;
        return (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onChange(t.id)}
            className={cn(
              "px-3 py-1.5 rounded-[3px] transition-all duration-150",
              fullWidth && "flex-1"
            )}
            style={{
              background: isActive ? "#FFFFFF" : "transparent",
              boxShadow: isActive ? "var(--shadow-segment)" : undefined,
            }}
          >
            <BodyMedium tone={isActive ? "primary" : "muted"}>
              {t.label}
            </BodyMedium>
          </button>
        );
      })}
    </div>
  );
}

export default SegmentedTabs;
