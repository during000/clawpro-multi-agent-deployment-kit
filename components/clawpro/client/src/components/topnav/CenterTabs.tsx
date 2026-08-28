/**
 * CenterTabs - 中央 segmented 切换控件（v5 / [Figma 1116-6184] 胶囊版）
 *
 * 设计来源：Figma 「公共组件/导航」中央切换（节点 1116:6184）
 * 视觉规范：
 *   - 容器：高 36px、bg rgba(219,221,228,0.32)、圆角 80px、relative 定位
 *   - 滑块（独立绝对定位 Rectangle）：与容器等高 36px、bg #FFFFFF、
 *     border 1px solid #CDD4DC、shadow 0px 1px 4px rgba(0,0,0,0.05)、圆角 40px
 *     → 根据 active tab 位置动态平移
 *   - Tab 文字：padding 4px 12px、字号 14、行高 22px、z-10 浮于滑块之上
 *     - Active：color #020617、font-weight 500
 *     - Normal：color #334155、font-weight 400
 *
 * 用法：
 *   <CenterTabs
 *     items={[{ label: "我的 Agent", value: "/my-openclaw" }, ...]}
 *     activeValue={location}
 *     onChange={(value) => navigate(value)}
 *   />
 */
import React, { useRef, useState, useEffect, useCallback } from "react";

export interface CenterTabItem<V extends string = string> {
  label: string;
  value: V;
  /** 稳定引导定位标识 */
  dataGuide?: string;
  /** 可选：自定义 Active 判断函数，默认严格等于；常用于路由前缀匹配 */
  matches?: (current: string) => boolean;
}

export interface CenterTabsProps<V extends string = string> {
  items: CenterTabItem<V>[];
  activeValue: string;
  onChange?: (value: V, index: number) => void;
  /** 自定义 Active 判断（全局），优先级低于 item.matches */
  isActive?: (item: CenterTabItem<V>, current: string) => boolean;
  className?: string;
}

export default function CenterTabs<V extends string = string>({
  items,
  activeValue,
  onChange,
  isActive,
  className = "",
}: CenterTabsProps<V>) {
  const navRef = useRef<HTMLElement>(null);
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const [sliderStyle, setSliderStyle] = useState<React.CSSProperties>({
    opacity: 0,
  });

  const checkActive = (item: CenterTabItem<V>) => {
    if (item.matches) return item.matches(activeValue);
    if (isActive) return isActive(item, activeValue);
    return item.value === activeValue;
  };

  const activeIndex = items.findIndex((item) => checkActive(item));

  const updateSlider = useCallback(() => {
    const nav = navRef.current;
    const activeTab = tabRefs.current[activeIndex];
    if (!nav || !activeTab) {
      setSliderStyle({ opacity: 0 });
      return;
    }
    const navRect = nav.getBoundingClientRect();
    const tabRect = activeTab.getBoundingClientRect();
    setSliderStyle({
      left: tabRect.left - navRect.left,
      width: tabRect.width,
      opacity: 1,
    });
  }, [activeIndex]);

  useEffect(() => {
    updateSlider();
  }, [updateSlider]);

  // 首次渲染后跳过动画
  const [mounted, setMounted] = useState(false);
  useEffect(() => {
    const timer = setTimeout(() => setMounted(true), 50);
    return () => clearTimeout(timer);
  }, []);

  return (
    <nav
      ref={navRef}
      className={`relative inline-flex items-center h-8 gap-2 rounded-[80px] ${className}`} /* allow-radius: Tenant 胶囊导航 */
      style={{ background: "rgba(219, 221, 228, 0.32)" }}
      role="tablist"
    >
      {/* 滑块：绝对定位，与容器等高，描边用 outline 不内缩 */}
      <div
        className="absolute top-0 h-full rounded-[40px] bg-white outline outline-1 outline-[#DFE5ED] shadow-[0px_1px_4px_0px_rgba(0,0,0,0.05)]" /* allow-radius: Tenant 胶囊滑块 */
        style={{
          ...sliderStyle,
          transition: mounted ? "left 200ms ease, width 200ms ease" : "none",
        }}
      />
      {/* Tab 文字层 */}
      {items.map((item, idx) => {
        const active = checkActive(item);
        return (
          <button
            key={item.value}
            ref={(el) => { tabRefs.current[idx] = el; }}
            type="button"
            role="tab"
            data-guide={item.dataGuide}
            aria-selected={active}
            onClick={() => onChange?.(item.value, idx)}
            className={[
              "relative z-10 px-3 py-1 text-[14px] leading-[22px] tracking-[0.005em] whitespace-nowrap transition-colors duration-150",
              active
                ? "text-[#020617] font-medium"
                : "text-[#334155] hover:text-[#020617] font-normal",
            ].join(" ")}
          >
            {item.label}
          </button>
        );
      })}
    </nav>
  );
}
