/**
 * GuideModuleFloat - 用户端非阻断性浮窗
 * 对应场景：结构层 1.2 页面重新排布 / 元素层 2.6 细节优化叠加
 * 
 * 固定位置：右下角
 * 
 * 内容变体：
 * - single: 单条内容 — 底部右对齐 CTA 黑色胶囊按钮（"立即体验"）
 * - multi: 多条内容 — 底部左侧"n/N  跳过引导"，右侧 ←→ 翻页，末页右箭头变"我知道了"
 * 
 * 设计稿参考：Figma node 4088:7837「用户端非阻断性浮窗」
 * 
 * 结构层次（对齐 Figma）：
 * ┌─ 外层容器（column, padding 12px, gap 12px, width 360px, radius 4px）
 * │  ├─ 内容区（column, gap 8px）
 * │  │  ├─ 头部行（row, justify-between）
 * │  │  │  ├─ 左侧（column, gap 4px）：副标题 + 标题
 * │  │  │  └─ 关闭按钮（20×20, radius 4px）
 * │  │  ├─ 描述文字 + 行动链接（12px, lh 20px）
 * │  │  └─ 配图区域（16:9 = 672×376, radius 4px）
 * │  └─ 底部操作区（row, justify-between / justify-end）
 * │     ├─ multi: 左侧"n/N  跳过引导" + 右侧翻页按钮
 * │     └─ single: 右对齐 CTA 黑色胶囊按钮
 * └─
 * 
 * 所有样式值均通过 --module-float-* CSS Token 引用，禁止硬编码。
 */
import { useState } from "react";
import { ArrowLeft, ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";

export interface ModuleFloatItem {
  /** 该步骤的副标题（如"消息通知"），可选，覆盖组件级 subtitle */
  subtitle?: string;
  /** 该步骤的主标题 */
  title: string;
  /** 描述文字 */
  description: string;
  /** 配图 URL（16:9 比例，推荐 672×376） */
  image?: string;
  /** 行动链接文案（如"跳转链接 →"） */
  actionText?: string;
  /** 行动链接地址 */
  actionHref?: string;
}

/** 内容变体 */
export type ModuleFloatVariant = "single" | "multi";

/**
 * 单个浮窗来源 —— 每个 source 代表一个「本应独立弹出」的浮窗。
 * 用于多浮窗合并约束：界面中最多只能存在一个浮窗。
 */
export interface ModuleFloatSource {
  /** 合并先后序号（越小越靠前页）；缺省时按数组传入顺序 */
  order?: number;
  /** 该浮窗副标题 */
  subtitle?: string;
  /** 该浮窗主标题 */
  title?: string;
  /** 该浮窗内容（单条浮窗即一条；合并时取首条作为该浮窗在合并后的「一页」） */
  items: ModuleFloatItem[];
}

/**
 * 用户端非阻断浮窗合并逻辑（界面中最多只能存在一个浮窗）：
 * - 0 个来源 → null（不展示）
 * - 1 个来源 → 单条内容浮窗（variant=single）
 * - ≥2 个来源 → 合并为一个「多条内容」浮窗（variant=multi）：
 *   每个来源作为「一条内容」分到不同页，按合并先后（order 升序，缺省按传入顺序）排序，
 *   最先合并的排在前一页。
 */
export function mergeModuleFloats(
  sources: ModuleFloatSource[],
): { variant: ModuleFloatVariant; items: ModuleFloatItem[] } | null {
  if (!sources || sources.length === 0) return null;

  // 按合并先后排序：order 升序，缺省回退到原始传入下标，最先合并 → 前一页
  const ordered = sources
    .map((source, index) => ({ source, index }))
    .sort((a, b) => (a.source.order ?? a.index) - (b.source.order ?? b.index))
    .map(({ source }) => source);

  // 把单个来源规整成「一页」内容（来源级 subtitle/title 覆盖其首条内容）
  const toItem = (source: ModuleFloatSource): ModuleFloatItem => {
    const primary = source.items[0] ?? { title: source.title ?? "", description: "" };
    return {
      ...primary,
      subtitle: source.subtitle ?? primary.subtitle,
      title: source.title ?? primary.title,
    };
  };

  if (ordered.length === 1) {
    // 单浮窗：保留其完整内容项
    return { variant: "single", items: ordered[0].items };
  }

  // 多浮窗：每个来源合并为不同页的一条内容
  return { variant: "multi", items: ordered.map(toItem) };
}

interface GuideModuleFloatProps {
  open: boolean;
  onClose: () => void;
  /** 副标题（如"消息通知"），12px Regular rgba(0,0,0,0.5) */
  subtitle?: string;
  /** 主标题（如"Agent 新版本上线"），14px Medium #000 */
  title?: string;
  /** 内容项列表（与 sources 二选一；同时传入时优先 sources） */
  items?: ModuleFloatItem[];
  /**
   * 多浮窗来源：传入后组件内部按「最多一个浮窗」约束自动合并，
   * 覆盖 items / variant。两个及以上来源会合并为多条内容浮窗（最先合并在前一页）。
   */
  sources?: ModuleFloatSource[];
  /** 内容变体：single=单CTA / multi=翻页导航（未传 sources 时生效） */
  variant?: ModuleFloatVariant;
  /** 单条模式下 CTA 按钮文案（默认"立即体验"） */
  confirmText?: string;
  /** CTA 点击回调 */
  onConfirm?: () => void;
}

export function GuideModuleFloat({
  open,
  onClose,
  subtitle = "消息通知",
  title = "Agent 新版本上线",
  items,
  sources,
  variant = "single",
  confirmText = "立即体验",
  onConfirm,
}: GuideModuleFloatProps) {
  const [currentItem, setCurrentItem] = useState(0);

  // 「界面中最多只能存在一个浮窗」约束：传入 sources 时按合并逻辑生成唯一浮窗，
  // 覆盖外部 items / variant；未传 sources 时退化为原有受控用法。
  const merged = sources ? mergeModuleFloats(sources) : null;
  const effectiveItems: ModuleFloatItem[] = merged ? merged.items : items ?? [];
  const effectiveVariant: ModuleFloatVariant = merged ? merged.variant : variant;

  if (!open || effectiveItems.length === 0) return null;

  // 合并/数据变化后页码可能越界，做一次安全收敛
  const safeIndex = Math.min(currentItem, effectiveItems.length - 1);
  const item = effectiveItems[safeIndex];
  const isMulti = effectiveVariant === "multi" && effectiveItems.length > 1;
  const isLast = safeIndex === effectiveItems.length - 1;
  const isFirst = safeIndex === 0;

  const handlePrev = () => setCurrentItem((s) => Math.max(0, Math.min(s, effectiveItems.length - 1) - 1));
  const handleNext = () => setCurrentItem((s) => Math.min(effectiveItems.length - 1, s + 1));
  const handleConfirm = () => { onConfirm?.(); onClose(); };

  return (
    <div
      className="fixed bottom-6 right-6 z-[10000] animate-in slide-in-from-bottom-4 duration-300"
      style={{ width: "var(--module-float-width)" }}
    >
      {/* 外层容器：column, padding 12px, gap 12px */}
      <div
        className="flex flex-col"
        style={{
          padding: "var(--module-float-padding)",
          gap: "12px",
          background: "var(--module-float-bg)",
          borderRadius: "var(--module-float-radius)",
          border: "1px solid var(--module-float-border)",
          boxShadow: "var(--module-float-shadow)",
        }}
      >
        {/* 内容区：column, gap 8px */}
        <div className="flex flex-col" style={{ gap: "8px" }}>
          {/* 头部行：row, justify-between */}
          <div className="flex items-start justify-between">
            {/* 左侧：column, gap 4px — 副标题 + 标题 */}
            <div className="flex-1 min-w-0 flex flex-col" style={{ gap: "4px" }}>
              {(item?.subtitle || subtitle) && (
                <span
                  className="block text-xs font-normal"
                  style={{ color: "var(--module-float-subtitle)" }}
                >
                  {item?.subtitle || subtitle}
                </span>
              )}
              <h3
                className="text-sm font-medium leading-[22px]"
                style={{ color: "var(--module-float-title)" }}
              >
                {item?.title || title}
              </h3>
            </div>
            {/* 关闭按钮：20×20, radius 4px */}
            <button
              onClick={onClose}
              className="shrink-0 flex items-center justify-center rounded transition-colors text-[var(--guide-bubble-close)] hover:bg-black/5"
              style={{
                width: "var(--module-float-close-size)",
                height: "var(--module-float-close-size)",
                borderRadius: "var(--module-float-radius)",
              }}
              aria-label="关闭"
            >
              <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path
                  d="M13.9595 7.17139L11.1304 9.99951L13.9595 12.8286L12.8286 13.9595L9.99951 11.1304L7.17139 13.9595L6.04053 12.8286L8.86865 9.99951L6.04053 7.17139L7.17139 6.04053L9.99951 8.86865L12.8286 6.04053L13.9595 7.17139Z"
                  fill="black"
                />
              </svg>
            </button>
          </div>

          {/* 描述文字 + 行动链接：12px Regular lh-20px */}
          <p
            className="text-xs font-normal leading-[20px]"
            style={{ color: "var(--module-float-desc)" }}
          >
            {item?.description}
            {!isMulti && item?.actionText && (
              <a
                href={item.actionHref || "#"}
                className="inline-flex items-center gap-0.5 font-medium ml-1 hover:opacity-80 transition-opacity"
                style={{ color: "var(--module-float-link)" }}
              >
                {item.actionText}
                <span className="ml-0.5">→</span>
              </a>
            )}
            {isMulti && (
              <button
                onClick={handleConfirm}
                className="inline-flex items-center font-medium hover:opacity-80 transition-opacity"
                style={{ color: "var(--module-float-link)", padding: 0, margin: 0 }}
              >{confirmText}<span className="ml-0.5">→</span></button>
            )}
          </p>

          {/* 配图区域：16:9 比例（672×376），radius 4px */}
          <div
            className="w-full aspect-[16/9] overflow-hidden"
            style={{ borderRadius: "var(--module-float-image-radius)" }}
          >
            {item?.image ? (
              <img src={item.image} alt={item.title} className="w-full h-full object-cover object-top" />
            ) : (
              <div className="w-full h-full flex items-center justify-center bg-[#ECEEF6]">
                <div
                  className="w-[74%] h-[68%]"
                  style={{
                    borderRadius: "var(--module-float-image-radius)",
                    background: "rgba(255, 255, 255, 0.85)",
                    border: "1px solid #FFFFFF",
                  }}
                />
              </div>
            )}
          </div>
        </div>

        {/* 底部操作区 */}
        {isMulti ? (
          /* 多条内容：row, justify-between, align-center */
          <div className="flex items-center justify-between">
            {/* 左侧："n/N  跳过引导"（一体文本节点，中间两个空格） */}
            <button
              onClick={onClose}
              className="text-xs font-normal tabular-nums transition-opacity hover:opacity-70"
              style={{
                color: "var(--module-float-step-text)",
                letterSpacing: "0.015em",
                lineHeight: "22px",
              }}
            >
              {safeIndex + 1}/{effectiveItems.length}&nbsp;&nbsp;跳过引导
            </button>

            {/* 右侧：翻页按钮组 */}
            <div className="flex items-center gap-2">
              {/* 上一步箭头 */}
              <Button
                variant="tenant-outline"
                size="claw-sm"
                onClick={handlePrev}
                disabled={isFirst}
                className="w-[38px] h-[28px] !px-0"
                aria-label="上一步"
              >
                <ArrowLeft className="w-4 h-4" />
              </Button>

              {isLast ? (
                /* 最后一步：显示"我知道了"文字按钮 */
                <Button
                  variant="tenant-outline"
                  size="claw-sm"
                  onClick={onClose}
                  className="h-[28px] px-4 text-[12px]"
                >
                  我知道了
                </Button>
              ) : (
                /* 下一步箭头 */
                <Button
                  variant="tenant-outline"
                  size="claw-sm"
                  onClick={handleNext}
                  className="w-[38px] h-[28px] !px-0"
                  aria-label="下一步"
                >
                  <ArrowRight className="w-4 h-4" />
                </Button>
              )}
            </div>
          </div>
        ) : (
          /* 单条内容：row, justify-end */
          <div className="flex justify-end">
            {/* CTA 黑色胶囊按钮 */}
            <Button
              variant="tenant-primary"
              size="claw-sm"
              onClick={handleConfirm}
              className="w-[80px] h-[28px] text-xs"
            >
              {confirmText}
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}
