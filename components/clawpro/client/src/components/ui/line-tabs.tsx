/**
 * LineTabs - 页面标题下方的一级 Tab 切换器（下划线式）
 *
 * 规范来源：SKILL-GLOBAL-COMPONENTS.md §11.5
 *   - 容器：flex items-center gap-1 border-b border-[#dbe6ff]
 *   - 单项：px-4 py-3 / 14px Medium
 *   - 选中：text-[var(--text-title)] + border-b-2 border-[#0A0A0A] -mb-px
 *   - 默认：text-[var(--text-muted)] + hover:text-[var(--text-title)]
 *
 * 仅用于：页面标题（AdminPageHeader）下方的一级导航 Tab。
 * 不可用于卡片内部、弹窗内部、表格工具栏（那些场景用 §11 Tab 切换卡 / Plain 按钮）。
 *
 * 用法：
 *   <LineTabs
 *     tabs={[
 *       { id: "preset", label: "初始技能包" },
 *       { id: "roles",  label: "角色设定" },
 *       { id: "source", label: "技能安装来源", comingSoon: true },
 *     ]}
 *     active={tab}
 *     onChange={setTab}
 *     description="当前 Tab 的描述文案"  // 可选
 *   />
 */
import { Badge } from "@/components/ui/badge";
import { BodyMedium, CompactText } from "@/components/ui/Typography";
import type { ReactNode } from "react";

export interface LineTabDef<T extends string> {
  id: T;
  label: string;
  /** 稳定引导定位标识 */
  dataGuide?: string;
  /** 标签右侧附加节点，例如新功能标识 */
  suffix?: ReactNode;
  /** 是否在标签右侧展示「即将开放」Badge */
  comingSoon?: boolean;
}

interface LineTabsProps<T extends string> {
  tabs: ReadonlyArray<LineTabDef<T>>;
  active: T;
  onChange: (id: T) => void;
  /** 当前选中 Tab 的描述文案（可选，渲染在 Tab 下方一行） */
  description?: string;
  /** 容器额外 className（一般用于外层布局留白） */
  className?: string;
  /** 单个 Tab button 的额外 className，默认不传以保持规范尺寸 */
  tabClassName?: string;
  /** 标签文字额外 className，默认不传以保持 14px Medium */
  labelClassName?: string;
}

export function LineTabs<T extends string>({
  tabs,
  active,
  onChange,
  description,
  className,
  tabClassName,
  labelClassName,
}: LineTabsProps<T>) {
  return (
    <div className={className}>
      {/* Tab 切换器 */}
      <div className="mb-1">
        <div className="flex items-center gap-1 border-b border-[#dbe6ff]">
          {tabs.map((tab) => {
            const isActive = active === tab.id;
            return (
              <button
                key={tab.id}
                type="button"
                data-guide={tab.dataGuide}
                onClick={() => onChange(tab.id)}
                className={`relative px-4 py-3 transition-colors whitespace-nowrap inline-flex items-center gap-1.5 ${
                  isActive
                    ? "border-b-2 border-[#0A0A0A] -mb-px"
                    : ""
                } ${tabClassName || ""}`}
              >
                <BodyMedium
                  as="span"
                  tone={isActive ? "primary" : "muted"}
                  className={
                    isActive
                      ? labelClassName
                      : `hover:text-[var(--text-title)] transition-colors ${labelClassName || ""}`
                  }
                >
                  {tab.label}
                </BodyMedium>
                {tab.suffix}
                {tab.comingSoon && (
                  <Badge variant="outline" className="px-1.5 py-0.5">
                    即将开放
                  </Badge>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Tab 描述（仅一行） */}
      {description && (
        <div className="flex items-center gap-3 mt-3 mb-6">
          <CompactText as="p" tone="muted" className="leading-relaxed">
            {description}
          </CompactText>
        </div>
      )}
    </div>
  );
}

export default LineTabs;
