/**
 * GuideHighlightBubble - 步骤指引气泡（带呼吸灯 / 矩形标注的步骤性指引）
 * 对应场景：结构层 1.2 / 元素层 2.1~2.3 / 功能位置变动
 *
 * 定位方式（核心）：
 *   通过 `region.selector` 实时识别「当前真实页面」里的目标 DOM 元素，
 *   用 getBoundingClientRect 测量其真实位置与尺寸来绘制热点（呼吸灯 / 矩形标注），
 *   并把气泡贴到目标元素旁。热点会精确贴合元素而非凭空浮在页面上层。
 *   - 矩形标注：圆角描边框直接「框住」目标元素（按 padding 外扩）。
 *   - 呼吸灯：圆点贴到目标元素对应边缘的中点。
 *   未提供 selector 或命中不到元素时，回退到 region 的兜底坐标；
 *   两者都缺失则不渲染（避免出现随机大小/位置的浮层）。
 *
 * ══ 分类体系（对齐设计稿 node 4113-10615） ══
 *   1/ 呼吸灯（hotspotShape="circle"）—— 圆形脉冲热点
 *   2/ 矩形标注（hotspotShape="rect"）—— 蓝色圆角矩形标注
 *   每一类均可选择是否附加有序列表（listItems，≤3 条）
 */
import { useEffect, useLayoutEffect, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { GuidePointBubble } from "./GuidePointBubble";

type Placement = "top" | "bottom" | "left" | "right";

/** 演示用有序列表（region 未提供 listItems 时回退） */
const DEMO_LIST_ITEMS = [
  "管控端新增「Agent 类型管理」页面支持镜像更新推送",
  "管控端 Agent 列表工具栏新增镜像更新提醒铃铛",
  "管控端平台策略新增「允许员工自动更新 Agent 版本」开关",
];

export interface HighlightRegion {
  /** 区域标识 */
  id: string;
  /**
   * 目标元素 CSS 选择器（推荐）。命中后实时测量真实位置/尺寸进行标注，
   * 使呼吸灯/矩形精确贴合当前页面元素，而非凭空浮层。
   */
  selector?: string;
  /** 兜底位置（视口坐标系），仅在 selector 缺失/未命中时使用 */
  top?: number;
  left?: number;
  width?: number;
  height?: number;
  /** 矩形标注相对目标元素外扩的内边距（px，默认 6） */
  padding?: number;
  /** 气泡标题 */
  title: string;
  /** 气泡描述 */
  description: string;
  /** 气泡位置 */
  bubblePlacement?: "top" | "bottom" | "left" | "right";
  /** 可选有序列表（showList 时展示，缺省回退到演示列表） */
  listItems?: string[];
}

interface Rect {
  top: number;
  left: number;
  width: number;
  height: number;
}

interface GuideHighlightBubbleProps {
  open: boolean;
  onClose: () => void;
  /** 高亮区域列表（支持多步串联） */
  regions: HighlightRegion[];
  /** 当前高亮步骤 */
  currentIndex?: number;
  /** 切换步骤回调 */
  onIndexChange?: (index: number) => void;
  /** 端类型 — 控制按钮圆角（admin=4px 方角，tenant=full 胶囊） */
  endpoint?: "admin" | "tenant";
  /** 热点形状：circle=呼吸灯 / rect=矩形标注 */
  hotspotShape?: "circle" | "rect";
  /** 是否附加有序列表 */
  showList?: boolean;
}

export function GuideHighlightBubble({
  open,
  onClose,
  regions,
  currentIndex = 0,
  onIndexChange,
  endpoint = "tenant",
  hotspotShape = "circle",
  showList = false,
}: GuideHighlightBubbleProps) {
  const region = regions[currentIndex];
  const placement: Placement = region?.bubblePlacement || "right";

  // 实时测量目标元素位置/尺寸（selector 优先，未命中回退兜底坐标）
  const [rect, setRect] = useState<Rect | null>(null);

  // 气泡真实尺寸 —— 用于「自动匹配方向 + 视口边界 clamp」，
  // 确保任何 anchor 位置（含屏幕边缘元素）气泡都不会被截断。
  const bubbleRef = useRef<HTMLDivElement>(null);
  const [bubbleSize, setBubbleSize] = useState<{ w: number; h: number }>({ w: 280, h: 0 });

  useLayoutEffect(() => {
    const el = bubbleRef.current;
    if (!el) return;
    const measure = () => {
      const r = el.getBoundingClientRect();
      setBubbleSize((prev) =>
        prev.w === r.width && prev.h === r.height ? prev : { w: r.width, h: r.height }
      );
    };
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
    // rect 进入依赖：气泡 DOM 在 rect 测得后才渲染，需在其挂载后重新 observe
  }, [open, currentIndex, hotspotShape, showList, region, rect]);

  useEffect(() => {
    if (!open || !region) {
      setRect(null);
      return;
    }

    const same = (a: Rect | null, b: Rect | null) =>
      !!a && !!b && a.top === b.top && a.left === b.left && a.width === b.width && a.height === b.height;

    let raf = 0;
    const measure = () => {
      let next: Rect | null = null;
      if (region.selector) {
        const el = document.querySelector(region.selector) as HTMLElement | null;
        if (el) {
          const r = el.getBoundingClientRect();
          if (r.width > 0 || r.height > 0) {
            next = { top: r.top, left: r.left, width: r.width, height: r.height };
          }
        }
      }
      // selector 未命中 → 兜底坐标（视口坐标系）
      if (!next && region.top != null && region.left != null) {
        next = {
          top: region.top,
          left: region.left,
          width: region.width ?? 0,
          height: region.height ?? 0,
        };
      }
      setRect((prev) => (same(prev, next) ? prev : next));
      raf = requestAnimationFrame(measure); // 持续跟随滚动 / 动态布局
    };

    measure();
    return () => cancelAnimationFrame(raf);
  }, [open, region]);

  // 切换步骤时，把目标元素滚动到可视区域中央
  useEffect(() => {
    if (!open || !region?.selector) return;
    const el = document.querySelector(region.selector);
    el?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, [open, currentIndex, region?.selector]);

  if (!open || regions.length === 0 || !region || !rect) return null;

  const isMulti = regions.length > 1;
  const isLast = currentIndex === regions.length - 1;

  const handleNext = () => {
    if (isLast) onClose();
    else onIndexChange?.(currentIndex + 1);
  };
  const handlePrev = () => {
    if (currentIndex > 0) onIndexChange?.(currentIndex - 1);
  };

  // 矩形标注框（按 padding 外扩，覆盖目标元素）
  const pad = region.padding ?? 6;
  const box: Rect = {
    top: rect.top - pad,
    left: rect.left - pad,
    width: rect.width + pad * 2,
    height: rect.height + pad * 2,
  };

  // 气泡贴靠的参照框：矩形模式用外扩框，呼吸灯模式用元素本身
  const anchor = hotspotShape === "rect" ? box : rect;
  const GAP = 8; // 气泡与目标间距
  const HOT = hotspotShape === "circle" ? 16 : 0; // 为呼吸灯预留空间
  const MARGIN = 12; // 气泡与视口边缘的最小安全距离

  // ── 自动匹配方向 + 视口 clamp ──────────────────────────────
  // 根据 anchor 四周可用空间与气泡真实尺寸，翻转 / 兜底选择气泡方向，
  // 并把最终坐标夹紧进视口，确保气泡永不被屏幕截断或卡在边缘。
  const vw = typeof window !== "undefined" ? window.innerWidth : 0;
  const vh = typeof window !== "undefined" ? window.innerHeight : 0;
  const bw = bubbleSize.w;
  const bh = bubbleSize.h;
  const cx = anchor.left + anchor.width / 2;
  const cy = anchor.top + anchor.height / 2;

  const space: Record<Placement, number> = {
    top: anchor.top,
    bottom: vh - (anchor.top + anchor.height),
    left: anchor.left,
    right: vw - (anchor.left + anchor.width),
  };
  const opposite: Record<Placement, Placement> = {
    top: "bottom",
    bottom: "top",
    left: "right",
    right: "left",
  };
  // 该方向放下气泡所需的总空间（含间距 / 呼吸灯预留 / 安全边距）
  const need = (p: Placement) =>
    p === "top" || p === "bottom" ? bh + GAP + HOT + MARGIN : bw + GAP + HOT + MARGIN;

  let activePlacement: Placement = placement;
  if (space[activePlacement] < need(activePlacement)) {
    if (space[opposite[activePlacement]] >= need(activePlacement)) {
      // 首选方向放不下、但反方向够 → 翻转
      activePlacement = opposite[activePlacement];
    } else {
      // 相对两侧都放不下 → 选剩余空间最大的方向兜底
      activePlacement = (["bottom", "top", "right", "left"] as Placement[]).reduce(
        (best, cur) => (space[cur] > space[best] ? cur : best),
        activePlacement
      );
    }
  }

  // 依据最终方向计算气泡左上角坐标
  let left: number;
  let top: number;
  switch (activePlacement) {
    case "top":
      left = cx - bw / 2;
      top = anchor.top - GAP - HOT - bh;
      break;
    case "bottom":
      left = cx - bw / 2;
      top = anchor.top + anchor.height + GAP + HOT;
      break;
    case "left":
      left = anchor.left - GAP - HOT - bw;
      top = cy - bh / 2;
      break;
    case "right":
    default:
      left = anchor.left + anchor.width + GAP + HOT;
      top = cy - bh / 2;
      break;
  }
  // clamp 进视口（保留 MARGIN），气泡尺寸尚未测得时不强行 clamp，避免首帧跳动
  if (bw > 0) left = Math.max(MARGIN, Math.min(left, vw - bw - MARGIN));
  if (bh > 0) top = Math.max(MARGIN, Math.min(top, vh - bh - MARGIN));

  // 呼吸灯圆点：贴在目标元素「朝向气泡」那条边的中点（跟随翻转后的方向）
  const hotspotStyle = (): CSSProperties => {
    const hcx = rect.left + rect.width / 2;
    const hcy = rect.top + rect.height / 2;
    switch (activePlacement) {
      case "top":
        return { top: rect.top, left: hcx, transform: "translate(-50%, -50%)" };
      case "bottom":
        return { top: rect.top + rect.height, left: hcx, transform: "translate(-50%, -50%)" };
      case "left":
        return { top: hcy, left: rect.left, transform: "translate(-50%, -50%)" };
      case "right":
      default:
        return { top: hcy, left: rect.left + rect.width, transform: "translate(-50%, -50%)" };
    }
  };

  return (
    <div className="fixed inset-0 z-[9990] pointer-events-none">
      {/* 矩形标注：圆角描边框直接框住目标元素 */}
      {hotspotShape === "rect" && (
        <div
          className="absolute pointer-events-none"
          style={{ top: box.top, left: box.left, width: box.width, height: box.height }}
        >
          <div
            className="absolute inset-0 animate-pulse opacity-60"
            style={{
              borderRadius: "6px",
              border: "1px solid #1447E6",
              boxShadow: "0 4px 4px 0 rgba(20, 71, 230, 0.12)",
            }}
          />
          <div
            className="absolute inset-0"
            style={{
              borderRadius: "6px",
              border: "1px solid #1447E6",
              boxShadow: "0 4px 4px 0 rgba(20, 71, 230, 0.12)",
            }}
          />
        </div>
      )}

      {/* 呼吸灯：圆点贴在目标元素边缘中点 */}
      {hotspotShape === "circle" && (
        <div className="absolute w-3.5 h-3.5 pointer-events-none" style={hotspotStyle()}>
          <div className="absolute inset-0 rounded-full bg-[var(--text-brand)] animate-ping opacity-40" />
          <div className="absolute inset-0.5 rounded-full bg-[var(--text-brand)]" />
        </div>
      )}

      {/* 步骤指引气泡（dark）：定位到目标旁，方向已自动匹配并 clamp 进视口。
          用左上角绝对坐标定位（不再用 translate 居中），便于精确做边界夹紧。 */}
      <div
        ref={bubbleRef}
        className="absolute animate-in fade-in duration-200 pointer-events-auto"
        style={{ left, top, opacity: bh > 0 ? 1 : 0 }}
      >
        <GuidePointBubble
          open
          onClose={onClose}
          title={region.title}
          description={region.description}
          variant="dark"
          contentVariant="text-only"
          currentStep={currentIndex + 1}
          totalSteps={regions.length}
          showSteps={isMulti}
          actionText="我知道了"
          showHotspot={false}
          listItems={showList ? (region.listItems ?? DEMO_LIST_ITEMS) : undefined}
          placement={activePlacement}
          endpoint={endpoint}
          onNext={handleNext}
          onPrev={handlePrev}
        />
      </div>
    </div>
  );
}
