/**
 * Portable DarkVeil — L1 静态兜底的 React 等价包装（无 ogl 时使用）
 * ───────────────────────────────────────────────────────────────────────────
 * 对应 component-specs/dark-veil.md §9.2 的「可选 React 包装」。
 * 当宿主仓不便引 ogl / WebGL（走 L1）但仍用 React 时，用本组件替代
 * demo 仓的 client/src/components/ui/DarkVeil.tsx（L0 完整移植）。
 *
 * 视觉来自配套样式 portable/css/dark-veil.css（cp-darkveil-hero 三层结构），
 * 本组件只负责挂同一套 className 语义，保持与 L0 一致的 hero 结构：
 *   基底 + 径向光晕飘带（+顶部 mask 淡出）+ 底部收束叠层 + 内容层 z-10。
 *
 * 用法：
 *   import "../../css/dark-veil.css";
 *   import { DarkVeilStaticFallback } from "./dark-veil-static";
 *
 *   <DarkVeilStaticFallback>
 *     {/* 图标 + 标题 + 描述 + 按钮 + 权益卡 *\/}
 *   </DarkVeilStaticFallback>
 *
 * 三档兜底详见 dark-veil.md §9（禁止改写 L0 / L1 / L2 含义）；
 * 能装 ogl 则优先 L0（复制 DarkVeil.tsx + npm i ogl）。
 * ───────────────────────────────────────────────────────────────────────────
 */

import React from "react";

export interface DarkVeilStaticFallbackProps {
  /** hero 内容（图标 / 标题 / 描述 / 按钮 / 权益卡等），始终落在 z-10 内容层 */
  children?: React.ReactNode;
  /** 额外类名（仅用于布局微调，禁止覆盖背景三层的 pointer-events / z 层级） */
  className?: string;
}

export function DarkVeilStaticFallback({ children, className }: DarkVeilStaticFallbackProps) {
  return (
    <div className={["cp-darkveil-hero", className].filter(Boolean).join(" ")}>
      {/* 第 0+1+2 层合一：基底 + 光晕飘带 + 顶部淡出；底部收束由 ::after 提供。纯背景，aria-hidden */}
      <div className="cp-darkveil-hero__bg" aria-hidden="true" />
      {/* 内容层：永远压在背景之上 */}
      <div className="cp-darkveil-hero__content">{children}</div>
    </div>
  );
}

export default DarkVeilStaticFallback;
