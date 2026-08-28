/**
 * TruncatedTooltip —— 溢出省略文本的统一 Tooltip 包裹器
 *
 * 用途：任何带 `truncate` / `line-clamp` 的文本（列表名称、卡片副标题、进度明细等），
 * hover 时用 Tooltip 展示完整内容；文本没被截断时不展示浮层，避免无意义 tooltip。
 *
 * 用法：包裹「真正带截断 class 的那个元素」（可以是原生标签或 Typography 组件，
 * 需能转发 ref），文本原文通过 text 传入：
 *
 *   <TruncatedTooltip text={name}>
 *     <span className="block truncate">{name}</span>
 *   </TruncatedTooltip>
 *
 * 实现要点：溢出判定在「每次 hover / focus 打开时」实时计算（scrollWidth > clientWidth），
 * 不依赖挂载时的一次性测量，因此容器宽度变化（弹窗尺寸切换、滚动条出现、文案变化）后依然准确。
 */
import { cloneElement, useRef, useState, type ReactElement, type Ref } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface TruncatedTooltipProps {
  /** Tooltip 中展示的完整文本 */
  text: string;
  /** 带截断样式的子元素（需可转发 ref 到真实 DOM） */
  children: ReactElement;
  /** Tooltip 方向，默认 top */
  side?: "top" | "right" | "bottom" | "left";
  /** TooltipContent 额外样式（如放宽最大宽度） */
  contentClassName?: string;
}

export function TruncatedTooltip({
  text,
  children,
  side = "top",
  contentClassName,
}: TruncatedTooltipProps) {
  const ref = useRef<HTMLElement>(null);
  const [open, setOpen] = useState(false);

  const node = cloneElement(children as ReactElement<{ ref?: Ref<HTMLElement> }>, { ref });

  return (
    <Tooltip
      open={open}
      delayDuration={150}
      onOpenChange={(next) => {
        if (!next) {
          setOpen(false);
          return;
        }
        const el = ref.current;
        // 纵向截断（line-clamp）与横向截断（truncate）都纳入判定
        const truncated =
          !!el && (el.scrollWidth > el.clientWidth + 1 || el.scrollHeight > el.clientHeight + 1);
        setOpen(truncated);
      }}
    >
      <TooltipTrigger asChild>{node}</TooltipTrigger>
      <TooltipContent side={side} className={cn("max-w-[320px] break-all", contentClassName)}>
        {text}
      </TooltipContent>
    </Tooltip>
  );
}

export default TruncatedTooltip;
