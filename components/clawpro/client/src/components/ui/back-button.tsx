/**
 * BackButton - 标准返回按钮（公共组件）
 *
 * 设计语言：
 *   ┌──────────────────────────────────────────────────────────────────────┐
 *   │ 形态：[← 文案] 横向排列；icon 16x16，文字 14px。                      │
 *   │ 颜色：深灰（text #525252 → hover #0A0A0A）                           │
 *   │ 间距：gap 1.5（6px）                                                  │
 *   └──────────────────────────────────────────────────────────────────────┘
 *
 * 使用：
 *   <BackButton onClick={onBack} />                          // 默认 "返回"
 *   <BackButton onClick={onBack}>返回公共技能库</BackButton>     // 自定义文案
 *
 * 替换前的写法：
 *   <button onClick={onBack} className="flex items-center gap-1.5 text-sm text-[#525252] hover:text-[#0A0A0A] transition-colors">
 *     <ArrowLeft className="w-4 h-4" />
 *     返回 XXX
 *   </button>
 */
import * as React from "react";
import { ArrowLeft } from "lucide-react";
import { cn } from "@/lib/utils";

export interface BackButtonProps
  extends Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, "children"> {
  /** 自定义文案；不传时显示「返回」 */
  children?: React.ReactNode;
}

export function BackButton({
  children = "返回",
  className,
  type = "button",
  ...props
}: BackButtonProps) {
  return (
    <button
      type={type}
      data-slot="back-button"
      className={cn(
        "inline-flex items-center gap-1.5 text-sm text-[#525252] hover:text-[#0A0A0A] transition-colors",
        className
      )}
      {...props}
    >
      <ArrowLeft className="w-4 h-4" />
      <span>{children}</span>
    </button>
  );
}

export default BackButton;
