/**
 * useOpsObservationCalendarBillingExempt
 *
 * OpsObservation（运维观测页）专用：停服态下让本页两个"选择日期" DatePicker
 * 弹出的日历面板 100% 正常可用（不透明 / 正常配色 / 正常点击 / 正常聚焦）。
 *
 * 背景与方案与 OpenClawMonitor 页同款（详见
 * pages/admin/OpenClawMonitor/useOpenClawMonitorCalendarBillingExempt.ts 头部注释）：
 *   · Radix Popover 的日历面板走 Portal 挂到 <body>，触发器外层 data-billing-exempt
 *     覆盖不到面板；
 *   · 采用"多层 data-billing-exempt 打标 + 页面级 CSS 补充规则 + 事件同步兜底"四重兜底恢复；
 *   · 严格排除本身 disabled / aria-disabled / .rdp-day_disabled 的日期按钮，
 *     保证"停服前已禁用则延续禁用"。
 *
 * 四重兜底
 * ─────────────────────────────────────────────
 *   (a) 打标 wrapper / popover-content / calendar 三层 data-billing-exempt——
 *       为 AdminDisabledOverlay 的 JS 事件拦截提供多层放行标记。
 *   (b) 打标 calendar 根节点 —— 命中 overlay CSS "[wrapper] [data-billing-exempt] *"
 *       恢复分支，恢复内部所有 button / span 后代到 opacity:1、cursor:pointer。
 *   (c) 注入 body.admin-billing-suspended 作用域下补充 style，显式恢复
 *       popper-wrapper 内含 calendar 时其所有 button 视觉状态。
 *   (d) 事件同步兜底：window capture 阶段（早于 AdminDisabledOverlay 的 document
 *       capture 监听点）监听 pointerdown/mousedown/click，就地同步打标 wrapper，
 *       覆盖 MutationObserver 微任务尚未回调的极端时序缝隙。
 *
 * 独立 STYLE_ID：与其它页面各自幂等注入 & 卸载时各自移除，避免相互干扰。
 *
 * 约束
 * ─────────────────────────────────────────────
 * · 只修改本页范围内逻辑：hook 文件位于 pages/admin/OpsObservation/ 子目录；
 *   在 OpsObservation.tsx 中调用。
 * · 不动组件库：date-picker.tsx / popover.tsx / calendar.tsx 全部保持不动。
 * · 不动 AdminDisabledOverlay.tsx 与全局 hooks/useDatePickerBillingExempt.ts。
 * · 幂等：已打标节点直接跳过；style 元素以 id 幂等注入。
 */
import { useEffect } from "react";

const WRAPPER_SELECTOR = "[data-radix-popper-content-wrapper]";
const CONTENT_SELECTOR = '[data-slot="popover-content"]';
const CALENDAR_MARK = '[data-slot="calendar"]';
const EXEMPT_ATTR = "data-billing-exempt";
const SUSPENDED_BODY_CLASS = "admin-billing-suspended";
const STYLE_ID = "ops-observation-calendar-billing-exempt-style";

/**
 * 一次性注入本页级补充 style，作为 CSS 优先级兜底。
 * 幂等：以 STYLE_ID 判定是否已注入。
 */
function ensureSupplementaryStyle(): void {
  if (typeof document === "undefined") return;
  if (document.getElementById(STYLE_ID)) return;
  const style = document.createElement("style");
  style.id = STYLE_ID;
  style.textContent = `
/* OpsObservation 页 DatePicker 日历面板停服态豁免（页面级补充规则）
 * 与 AdminDisabledOverlay 里"popper-wrapper button 灰化"配对；
 * :has([data-slot="calendar"]) 锁定"只是日历面板"，不影响其它 Popover。
 * 排除 disabled / aria-disabled / .rdp-day_disabled 保证"停服前已禁用则延续禁用"。 */
body.${SUSPENDED_BODY_CLASS} ${WRAPPER_SELECTOR}:has(${CALENDAR_MARK}) button:not([disabled]):not([aria-disabled="true"]):not(.rdp-day_disabled),
body.${SUSPENDED_BODY_CLASS} ${WRAPPER_SELECTOR}:has(${CALENDAR_MARK}) a[href],
body.${SUSPENDED_BODY_CLASS} ${WRAPPER_SELECTOR}:has(${CALENDAR_MARK}) input,
body.${SUSPENDED_BODY_CLASS} ${WRAPPER_SELECTOR}:has(${CALENDAR_MARK}) select,
body.${SUSPENDED_BODY_CLASS} ${WRAPPER_SELECTOR}:has(${CALENDAR_MARK}) [role="combobox"],
body.${SUSPENDED_BODY_CLASS} ${WRAPPER_SELECTOR}:has(${CALENDAR_MARK}) [role="option"],
body.${SUSPENDED_BODY_CLASS} ${WRAPPER_SELECTOR}:has(${CALENDAR_MARK}) [role="menuitem"],
body.${SUSPENDED_BODY_CLASS} ${WRAPPER_SELECTOR}:has(${CALENDAR_MARK}) * {
  opacity: 1 !important;
  cursor: pointer !important;
}
/* 非交互后代恢复默认光标 */
body.${SUSPENDED_BODY_CLASS} ${WRAPPER_SELECTOR}:has(${CALENDAR_MARK}) *:not(button):not(a):not(input):not(select):not([role="combobox"]):not([role="option"]):not([role="menuitem"]) {
  cursor: auto !important;
}
`;
  document.head.appendChild(style);
}

/** 幂等打标：已有属性则跳过。 */
function mark(el: HTMLElement): void {
  if (!el.hasAttribute(EXEMPT_ATTR)) {
    el.setAttribute(EXEMPT_ATTR, "");
  }
}

/**
 * 处理单个 popper wrapper：若其内含 Calendar，则给
 * wrapper / popover-content / calendar 三层都打 exempt。
 */
function markWrapper(wrapperEl: HTMLElement): void {
  const calendar = wrapperEl.querySelector(CALENDAR_MARK) as HTMLElement | null;
  if (!calendar) return;
  mark(wrapperEl);
  const content = wrapperEl.querySelector(CONTENT_SELECTOR) as HTMLElement | null;
  if (content) mark(content);
  mark(calendar);
}

/**
 * 遍历 root 范围内所有 popper wrapper，若其内含 Calendar，则依 markWrapper 打标。
 *
 * 注意：Radix Popover 通过 Portal 将 wrapper **直接**append 到 body 下，此时
 * MutationObserver 收到的 addedNode 就是 wrapper 本身；而 querySelectorAll
 * 不包含 node 自身。因此需要显式检查 root 自身是否即为 wrapper —— 否则会漏打标。
 */
function markCalendarPanels(root: ParentNode): void {
  if (
    root instanceof HTMLElement &&
    root.matches?.(WRAPPER_SELECTOR)
  ) {
    markWrapper(root);
  }
  const wrappers = root.querySelectorAll?.(WRAPPER_SELECTOR);
  wrappers?.forEach((wrapper) => markWrapper(wrapper as HTMLElement));
}

export function useOpsObservationCalendarBillingExempt(): void {
  useEffect(() => {
    if (typeof document === "undefined") return;

    // (c) 注入 CSS 补充规则
    ensureSupplementaryStyle();

    // (a)(b) 初次扫描 + MutationObserver
    markCalendarPanels(document.body);

    const observer = new MutationObserver((mutations) => {
      for (const m of mutations) {
        m.addedNodes.forEach((node) => {
          if (!(node instanceof HTMLElement)) return;
          if (
            node.matches?.(WRAPPER_SELECTOR) ||
            node.querySelector?.(WRAPPER_SELECTOR)
          ) {
            markCalendarPanels(node);
          }
        });
      }
    });
    observer.observe(document.body, { childList: true, subtree: true });

    // (d) 事件同步兜底：window capture 阶段
    const syncMarkFromEvent = (e: Event) => {
      const target = e.target;
      if (!(target instanceof Element)) return;
      const wrapper = target.closest(WRAPPER_SELECTOR) as HTMLElement | null;
      if (!wrapper) return;
      if (!wrapper.querySelector(CALENDAR_MARK)) return;
      markWrapper(wrapper);
    };
    window.addEventListener("pointerdown", syncMarkFromEvent, true);
    window.addEventListener("mousedown", syncMarkFromEvent, true);
    window.addEventListener("click", syncMarkFromEvent, true);

    return () => {
      observer.disconnect();
      window.removeEventListener("pointerdown", syncMarkFromEvent, true);
      window.removeEventListener("mousedown", syncMarkFromEvent, true);
      window.removeEventListener("click", syncMarkFromEvent, true);
      const style = document.getElementById(STYLE_ID);
      if (style) style.remove();
    };
  }, []);
}

export default useOpsObservationCalendarBillingExempt;
