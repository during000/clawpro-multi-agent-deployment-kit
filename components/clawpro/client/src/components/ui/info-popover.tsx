/**
 * InfoPopover - 气泡组件（基于 shadcn Popover）
 *
 * 行为：
 *   - 鼠标 hover 到触发元素时显示气泡
 *   - 鼠标移出触发元素或气泡时隐藏（带 120ms 延迟，方便用户从触发元素移到气泡内继续阅读）
 *   - 触发元素鼠标样式为 cursor-pointer（不使用 cursor-help/问号）
 *
 * 样式：
 *   - 直接复用 shadcn `<PopoverContent>`：白底黑字、浅灰描边、8px 圆角、标准阴影
 *   - 仅在外部对 `<PopoverContent>` 做 hover 行为绑定与宽度收窄，不重写颜色/边框/阴影
 *
 * 使用示例：
 *   <InfoPopover content="用户 ID 为唯一标识，不可修改" />
 *
 *   <InfoPopover
 *     title="说明"
 *     content={<div>多行说明文字...</div>}
 *     placement="top"
 *   />
 *
 *   <InfoPopover content="说明">
 *     <span>查看详情</span>
 *   </InfoPopover>
 */
import * as React from "react";
import { Info } from "lucide-react";

import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

interface InfoPopoverProps {
  /** 气泡内容，可为字符串或 ReactNode */
  content: React.ReactNode;
  /** 可选标题，渲染在内容上方，带底部分割线 */
  title?: React.ReactNode;
  /** 弹出方向，默认 top */
  placement?: "top" | "right" | "bottom" | "left";
  /** 弹出对齐方式，默认 center */
  align?: "start" | "center" | "end";
  /** 与触发元素的距离（px），默认 6 */
  sideOffset?: number;
  /** 关闭延迟（ms），默认 120ms。设为 0 可获得 Tooltip 风格的即时关闭 */
  closeDelay?: number;
  /** 自定义触发元素；不传时默认渲染 Info 图标 */
  children?: React.ReactNode;
  /** 触发元素自定义类名（仅对默认 Info 图标 wrapper 生效） */
  triggerClassName?: string;
  /** 气泡内容容器自定义类名（叠加到 shadcn PopoverContent 默认样式之后） */
  contentClassName?: string;
  /** 气泡最大宽度，默认 280px。传入 string 直接作为 CSS max-width */
  maxWidth?: number | string;
}

function InfoPopover({
  content,
  title,
  placement = "top",
  align = "center",
  sideOffset = 6,
  closeDelay = 120,
  children,
  triggerClassName,
  contentClassName,
  maxWidth = 280,
}: InfoPopoverProps) {
  const [open, setOpen] = React.useState(false);
  const closeTimer = React.useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancelClose = React.useCallback(() => {
    if (closeTimer.current) {
      clearTimeout(closeTimer.current);
      closeTimer.current = null;
    }
  }, []);

  const scheduleClose = React.useCallback(() => {
    cancelClose();
    closeTimer.current = setTimeout(() => setOpen(false), closeDelay);
  }, [cancelClose, closeDelay]);

  const handleEnter = React.useCallback(() => {
    cancelClose();
    setOpen(true);
  }, [cancelClose]);

  React.useEffect(() => {
    return () => cancelClose();
  }, [cancelClose]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <span
          data-slot="info-popover-trigger"
          className={cn(
            "inline-flex items-center justify-center cursor-pointer outline-none",
            triggerClassName
          )}
          onPointerEnter={handleEnter}
          onPointerLeave={scheduleClose}
          onFocus={handleEnter}
          onBlur={scheduleClose}
          // 阻止默认 click 触发：仅 hover 控制开合
          onClick={(e) => e.preventDefault()}
        >
          {children ?? <Info className="w-3.5 h-3.5 text-gray-400" />}
        </span>
      </PopoverTrigger>
      <PopoverContent
        side={placement}
        align={align}
        sideOffset={sideOffset}
        onPointerEnter={handleEnter}
        onPointerLeave={scheduleClose}
        // hover 模式下不抢焦点
        onOpenAutoFocus={(e) => e.preventDefault()}
        onCloseAutoFocus={(e) => e.preventDefault()}
        className={cn(
          // 复用 shadcn 默认（白底 #020617 文字 / #EAEEF4 边框 / 8px 圆角 / p-4 / 阴影）
          // 仅微调：放开默认 w-72 让宽度由 maxWidth 控制；缩小 padding 以贴合短提示语
          "w-auto p-3 text-sm",
          contentClassName
        )}
        style={{ maxWidth: typeof maxWidth === "number" ? `${maxWidth}px` : maxWidth }}
      >
        {title && (
          <div className="text-sm font-medium text-gray-950 pb-2 mb-2 border-b border-gray-200">
            {title}
          </div>
        )}
        <div className="text-sm leading-relaxed text-gray-950">
          {content}
        </div>
      </PopoverContent>
    </Popover>
  );
}

export { InfoPopover };
export type { InfoPopoverProps };
