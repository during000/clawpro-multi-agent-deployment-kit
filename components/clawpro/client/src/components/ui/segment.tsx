import * as React from "react";
import * as TabsPrimitive from "@radix-ui/react-tabs";

import { cn } from "@/lib/utils";

/**
 * Segment 分段选择器（管理端 4px 方角原版）
 *
 * 对齐 Figma 设计稿：水平分段选择器（管理端规范，与 claw-* 按钮同档 4px 方角）。
 *
 * ⚠️ 端别提示（0523 修订）：
 *   - 管理端（Admin）：请用 `Segment / SegmentList / SegmentItem / SegmentContent`
 *     或 `SegmentGroup / SegmentOption`（本文件下半部分），保持 6px 容器 + 4px 滑块
 *   - 用户端（Tenant）：请用 `TenantSegment / TenantSegmentList / TenantSegmentItem`
 *     或 `TenantSegmentGroup / TenantSegmentOption`（本文件最下方），全圆角胶囊
 *   - 不要把管理端组件直接用 `className="rounded-full"` 临时改胶囊——会破坏单一真理源
 *
 * 设计令牌（管理端，恢复 0523 之前的原版）：
 *   ┌─────────────────────────────┬───────────────────────────────────────────────────┐
 *   │ Token                       │ Value                                              │
 *   ├─────────────────────────────┼───────────────────────────────────────────────────┤
 *   │ container / bg              │ var(--bg-segment-track)  = #DBDDE432（冷灰偏蓝 +20% alpha）│
 *   │ container / border          │ var(--bg-segment-track)  = #DBDDE432（与 bg 同色，融为一体）│
 *   │ container / radius          │ 6px                                                │
 *   │ container / padding         │ 3px                                                │
 *   │ container / height          │ 36px (h-9)                                         │
 *   │ item / active bg            │ #FFFFFF                                            │
 *   │ item / active text          │ #020617 (font-semibold)                            │
 *   │ item / active shadow        │ 0px 1px 2px rgba(0,0,0,0.05)                      │
 *   │ item / active radius        │ 4px                                                │
 *   │ item / inactive text        │ #7b818f (font-normal)                              │
 *   │ item / hover text           │ #4b5563                                            │
 *   │ item / padding              │ 4px 16px                                           │
 *   │ item / disabled text        │ #d3d6db                                            │
 *   └─────────────────────────────┴───────────────────────────────────────────────────┘
 *
 * 两种用法：
 *
 * 1. 受控模式（带 TabsContent 联动）：
 *   <Segment defaultValue="basic">
 *     <SegmentList>
 *       <SegmentItem value="basic">基础配置</SegmentItem>
 *       <SegmentItem value="tools">工具管理</SegmentItem>
 *     </SegmentList>
 *     <SegmentContent value="basic">...</SegmentContent>
 *   </Segment>
 *
 * 2. 独立模式（纯样式，自行管理状态）：
 *   <SegmentGroup>
 *     <SegmentOption active={mode === "all"} onClick={() => setMode("all")}>全部</SegmentOption>
 *     <SegmentOption active={mode === "group"} onClick={() => setMode("group")}>组织</SegmentOption>
 *   </SegmentGroup>
 */

/* ============================================================== */
/*  管理端｜基于 Radix Tabs 的受控版本（需要 Segment 包裹）            */
/* ============================================================== */

function Segment({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return (
    <TabsPrimitive.Root
      data-slot="segment"
      className={cn("flex flex-col gap-4", className)}
      {...props}
    />
  );
}

function SegmentList({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.List>) {
  return (
    <TabsPrimitive.List
      data-slot="segment-list"
      className={cn(
        "bg-[var(--bg-segment-track)] text-[#7b818f] inline-flex h-9 w-fit items-center justify-center rounded-[6px] border border-[var(--bg-segment-track)] p-[2px]",
        className
      )}
      {...props}
    />
  );
}

function SegmentItem({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Trigger>) {
  return (
    <TabsPrimitive.Trigger
      data-slot="segment-item"
      className={cn(
        "text-[#7b818f] font-normal inline-flex h-[calc(100%-1px)] items-center justify-center rounded-[4px] border border-transparent px-4 py-1 text-sm whitespace-nowrap transition-all " +
          "data-[state=active]:bg-white data-[state=active]:text-gray-950 data-[state=active]:font-semibold data-[state=active]:shadow-[0px_1px_2px_rgba(0,0,0,0.05)] " +
          "hover:text-[#4b5563] " +
          "focus-visible:ring-[3px] focus-visible:ring-[#355EF1]/20 focus-visible:outline-none " +
          "disabled:pointer-events-none disabled:text-[#d3d6db]",
        className
      )}
      {...props}
    />
  );
}

function SegmentContent({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="segment-content"
      className={cn("flex-1 outline-none", className)}
      {...props}
    />
  );
}

/* ============================================================== */
/*  管理端｜独立版本（纯样式，不依赖 Radix Tabs Root）                 */
/* ============================================================== */

function SegmentGroup({
  className,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="segment-group"
      role="tablist"
      className={cn(
        "bg-[var(--bg-segment-track)] text-[#7b818f] inline-flex h-9 w-fit items-center justify-center rounded-[6px] border border-[var(--bg-segment-track)] p-[2px]",
        className
      )}
      {...props}
    />
  );
}

interface SegmentOptionProps extends React.ComponentProps<"button"> {
  active?: boolean;
}

// forwardRef：允许外层 Tooltip/HoverCard 等以 asChild 方式包裹并正确定位（不影响既有用法与样式）
const SegmentOption = React.forwardRef<HTMLButtonElement, SegmentOptionProps>(
  function SegmentOption({ className, active = false, ...props }, ref) {
    return (
      <button
        ref={ref}
        data-slot="segment-option"
        role="tab"
        aria-selected={active}
        data-state={active ? "active" : "inactive"}
        className={cn(
          "inline-flex h-[calc(100%-1px)] items-center justify-center rounded-[4px] border border-transparent px-4 py-1 text-sm whitespace-nowrap transition-all " +
            "focus-visible:ring-[3px] focus-visible:ring-[#355EF1]/20 focus-visible:outline-none " +
            "disabled:pointer-events-none disabled:text-[#d3d6db]",
          active
            ? "bg-white text-gray-950 font-semibold shadow-[0px_1px_2px_rgba(0,0,0,0.05)]"
            : "text-[#7b818f] font-normal hover:text-[#4b5563]",
          className
        )}
        {...props}
      />
    );
  }
);

/* ============================================================== */
/*  用户端｜TenantSegment 胶囊版（0525 对齐 Figma 1077-33424）        */
/*  与上方管理端组件接口完全一致，仅视觉差异：                          */
/*    - 容器：h-36px、圆角 80px、bg rgba(219,221,228,0.32)           */
/*    - Tab：px-12 py-4、圆角 80px、14/22/500                       */
/*    - Active：bg #FFF、border #CDD4DC、shadow 0 1px 4px 0.05     */
/*    - Normal：color #334155、font-weight 400                      */
/*  仅供 client/src/pages/tenant/** 使用，管理端禁用。               */
/* ============================================================== */

/**
 * TenantSegment 尺寸档位（仅用户端）。参照既有 segment 度量：
 *   - default：h-9（36px）容器 + px-3 / py-1 / 14px，对齐现有 TenantSegment
 *   - sm：h-8（32px）容器 + px-2.5 / py-0.5 / 13px，紧凑场景（如视图切换）
 * ⚠️ 该尺寸变体仅供用户端（client/src/pages/tenant/**）使用，管理端请用 Segment*。
 */
export type TenantSegmentSize = "default" | "sm";

const TENANT_SEGMENT_SIZE: Record<TenantSegmentSize, { track: string; trigger: string }> = {
  default: { track: "h-9", trigger: "px-3 py-1 text-[14px] leading-[22px]" },
  sm: { track: "h-8", trigger: "px-2.5 py-0.5 text-[13px] leading-[20px]" },
};

const TenantSegmentSizeContext = React.createContext<TenantSegmentSize>("default");

function TenantSegment({
  className,
  size = "default",
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Root> & { size?: TenantSegmentSize }) {
  return (
    <TenantSegmentSizeContext.Provider value={size}>
      <TabsPrimitive.Root
        data-slot="tenant-segment"
        className={cn("flex flex-col gap-4", className)}
        {...props}
      />
    </TenantSegmentSizeContext.Provider>
  );
}

function TenantSegmentList({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.List>) {
  const size = React.useContext(TenantSegmentSizeContext);
  return (
    <TabsPrimitive.List
      data-slot="tenant-segment-list"
      className={cn(
        "relative text-muted-foreground inline-flex w-fit items-center rounded-[80px] p-0",
        TENANT_SEGMENT_SIZE[size].track,
        className
      )}
      style={{ background: "rgba(219, 221, 228, 0.32)" }}
      {...props}
    />
  );
}

function TenantSegmentItem({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Trigger>) {
  const size = React.useContext(TenantSegmentSizeContext);
  return (
    <TabsPrimitive.Trigger
      data-slot="tenant-segment-item"
      className={cn(
        "relative z-10 text-slate-700 font-normal inline-flex h-full items-center justify-center gap-2 rounded-[40px] tracking-[0.005em] whitespace-nowrap transition-all border border-transparent " +
          TENANT_SEGMENT_SIZE[size].trigger + " " +
          "data-[state=active]:bg-white data-[state=active]:text-gray-950 data-[state=active]:font-medium data-[state=active]:border-[#CDD4DC] data-[state=active]:shadow-[var(--shadow-segment)] " +
          "hover:text-gray-950 " +
          "outline-none focus:outline-none focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-[var(--ring)]/20 " +
          "disabled:pointer-events-none disabled:text-gray-300",
        className
      )}
      {...props}
    />
  );
}

function TenantSegmentContent({
  className,
  ...props
}: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return (
    <TabsPrimitive.Content
      data-slot="tenant-segment-content"
      className={cn("flex-1 outline-none", className)}
      {...props}
    />
  );
}

function TenantSegmentGroup({
  className,
  size = "default",
  ...props
}: React.ComponentProps<"div"> & { size?: TenantSegmentSize }) {
  return (
    <TenantSegmentSizeContext.Provider value={size}>
      <div
        data-slot="tenant-segment-group"
        role="tablist"
        className={cn(
          "relative text-muted-foreground inline-flex w-fit items-center rounded-[80px] p-0",
          TENANT_SEGMENT_SIZE[size].track,
          className
        )}
        style={{ background: "rgba(219, 221, 228, 0.32)" }}
        {...props}
      />
    </TenantSegmentSizeContext.Provider>
  );
}

interface TenantSegmentOptionProps extends React.ComponentProps<"button"> {
  active?: boolean;
}

function TenantSegmentOption({
  className,
  active = false,
  ...props
}: TenantSegmentOptionProps) {
  const size = React.useContext(TenantSegmentSizeContext);
  return (
    <button
      data-slot="tenant-segment-option"
      role="tab"
      aria-selected={active}
      data-state={active ? "active" : "inactive"}
      className={cn(
        "relative z-10 inline-flex h-full items-center justify-center gap-2 rounded-[40px] tracking-[0.005em] whitespace-nowrap transition-all border border-transparent " +
          TENANT_SEGMENT_SIZE[size].trigger + " " +
          "outline-none focus:outline-none focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-[var(--ring)]/20 " +
          "disabled:pointer-events-none disabled:text-gray-300",
        active
          ? "bg-white text-gray-950 font-medium border-[#CDD4DC] shadow-[var(--shadow-segment)]"
          : "text-slate-700 font-normal hover:text-gray-950",
        className
      )}
      {...props}
    />
  );
}

/* ============================================================== */
/*  TextSwitch：纯文字切换器（用户端轻量场景，0523 新增）              */
/* ============================================================== */

/**
 * TextSwitch 文字切换器
 *
 * 严格对齐 Figma `1077:33980`（用户端「普通 / 多组织」切换）。
 *
 * 与 TenantSegmentGroup 的差异：
 *   ┌────────────┬──────────────────────────────┬────────────────────────────┐
 *   │            │ TenantSegmentGroup（胶囊版）   │ TextSwitch（文字版）          │
 *   ├────────────┼──────────────────────────────┼────────────────────────────┤
 *   │ 容器        │ var(--muted) 圆角胶囊 + h-9   │ 无背景，纯横排                  │
 *   │ active     │ 白底 + shadow-segment          │ #020617 深字 + 14/400          │
 *   │ inactive   │ #737373 灰字                  │ #A7A7A7 浅灰 14/400            │
 *   │ 分隔        │ 无                            │ 中间 `/` 字符 #E2E8F0           │
 *   │ 字号        │ 14 / inactive 400             │ 14 / 400（active/inactive 同字重）│
 *   │ 适用场景      │ 强切换（视图模式、分类筛选）       │ 弱切换（次要状态、配套主操作的辅助开关）│
 *   └────────────┴──────────────────────────────┴────────────────────────────┘
 *
 * 用法：
 *   <TextSwitch>
 *     <TextSwitchOption active={mode === "a"} onClick={() => setMode("a")}>普通</TextSwitchOption>
 *     <TextSwitchOption active={mode === "b"} onClick={() => setMode("b")}>多组织</TextSwitchOption>
 *   </TextSwitch>
 *
 * 实现细节：
 *   - 分隔符 `/` 由组件内部自动在相邻两个 `TextSwitchOption` 之间渲染（aria-hidden="true"）
 *   - gap 12px = Figma `layout_LE2IPO`
 *   - active/inactive 字重均为 400（Figma `style_3FUI4B` fontWeight=400），靠颜色拉差异
 */

function TextSwitch({
  className,
  children,
  ...props
}: React.ComponentProps<"div">) {
  // 在相邻 option 之间插入分隔符 `/`，保持 aria 语义干净（分隔符对屏幕阅读器隐藏）
  const items = React.Children.toArray(children).filter(Boolean);
  const interleaved: React.ReactNode[] = [];
  items.forEach((node, idx) => {
    interleaved.push(node);
    if (idx < items.length - 1) {
      interleaved.push(
        <span
          key={`sep-${idx}`}
          aria-hidden="true"
          className="text-gray-200 text-sm font-normal leading-none select-none"
        >
          /
        </span>
      );
    }
  });

  return (
    <div
      data-slot="text-switch"
      role="tablist"
      className={cn("inline-flex items-center gap-3", className)}
      {...props}
    >
      {interleaved}
    </div>
  );
}

interface TextSwitchOptionProps extends React.ComponentProps<"button"> {
  active?: boolean;
}

function TextSwitchOption({
  className,
  active = false,
  ...props
}: TextSwitchOptionProps) {
  return (
    <button
      type="button"
      data-slot="text-switch-option"
      role="tab"
      aria-selected={active}
      data-state={active ? "active" : "inactive"}
      className={cn(
        // 基础排版：14px / 400 / line-height 22px / letter-spacing 0.5%（Figma style_3FUI4B）
        "text-sm font-normal leading-[22px] tracking-[0.005em] transition-colors " +
          "focus-visible:outline-none focus-visible:ring-[2px] focus-visible:ring-[#355EF1]/20 focus-visible:rounded-sm " +
          "disabled:pointer-events-none disabled:text-gray-300",
        active
          ? "text-gray-950 cursor-default"
          : "text-gray-400 hover:text-gray-950 cursor-pointer",
        className
      )}
      {...props}
    />
  );
}

export {
  // 管理端 4px 方角
  Segment,
  SegmentList,
  SegmentItem,
  SegmentContent,
  SegmentGroup,
  SegmentOption,
  // 用户端胶囊
  TenantSegment,
  TenantSegmentList,
  TenantSegmentItem,
  TenantSegmentContent,
  TenantSegmentGroup,
  TenantSegmentOption,
  // 用户端文字切换
  TextSwitch,
  TextSwitchOption,
};
