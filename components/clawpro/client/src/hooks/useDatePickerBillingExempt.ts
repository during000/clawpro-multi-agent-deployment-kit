/**
 * useDatePickerBillingExempt
 *
 * 停服态下，让所有 DatePicker（含 DateTimePicker 等基于 Calendar 的浮层）
 * 的日历面板保持正常可用（100% 不透明、正常配色、正常交互），实现全站一次生效。
 *
 * 背景 / 为什么需要这个 hook
 * ─────────────────────────────────────────────
 * · DatePicker 内部使用 Radix Popover，PopoverContent 被 Portal 到<body> 下，
 *   与 Trigger 在真实 DOM 里是兄弟树，不在 Trigger 的祖先链上。
 * · AdminDisabledOverlay 使用原生 document.addEventListener(..., capture=true)
 *   来拦截 click / focus，判定依赖 event.target.closest('[data-billing-exempt]')
 *   —— 沿的是真实 DOM 链，不会沿 React 合成事件树冒泡。
 * · 因此 Trigger（或它的祖先）上添加的 data-billing-exempt 无法传导给日历面板；
 *   同时 overlay 的 CSS 规则 body.admin-billing-suspended
 *   [data-radix-popper-content-wrapper] button 会把面板整体灰化。
 *
 * 方案
 * ─────────────────────────────────────────────
 * · 在应用根节点挂一次 MutationObserver，监听 <body> 子树变化；
 * · 一旦发现新挂载的 [data-radix-popper-content-wrapper] 内含
 *   [data-slot="calendar"]（Calendar 组件根节点稳定标记），就给该 wrapper
 *   打上 data-billing-exempt，使 overlay 事件拦截命中豁免祖先直接放行，
 *   同时命中 overlay CSS 的豁免分支恢复 100% opacity。
 *
 * "停服前已禁用则延续禁用" 约束
 * ─────────────────────────────────────────────
 * · 本 hook 仅让日历面板整体豁免容器可用，不改Calendar 内部日期禁用逻辑；
 *   若 DatePicker 传入 min/max 或 disabledMatcher，Calendar 内部会把
 *   对应日期按钮打上原生 disabled，那些日期依然不可点，行为不变；
 * · 若 DatePicker 本身处于 disabled 状态，Popover 根本不会打开，
 *   面板不会出现，与本 hook 互不干扰。
 *
 * 约束
 * ─────────────────────────────────────────────
 * · 不改组件库代码（date-picker.tsx / popover.tsx / calendar.tsx /
 *   AdminDisabledOverlay.tsx 全部保持不动）。
 * · 该 hook 只应在应用根节点挂载一次（例如 App.tsx 中），避免多次实例化。
 */
import { useEffect } from "react";

const WRAPPER_SELECTOR = "[data-radix-popper-content-wrapper]";
const CALENDAR_MARK = '[data-slot="calendar"]';
const EXEMPT_ATTR = "data-billing-exempt";

/**
 * 遍历 root 范围内所有 popper wrapper，若其内含Calendar，则打上豁免标记。
 *幂等：已有 data-billing-exempt 的 wrapper 直接跳过。
 */
function markCalendarPanels(root: ParentNode): void {
  const wrappers = root.querySelectorAll?.(WRAPPER_SELECTOR);
  wrappers?.forEach((wrapper) => {
    const el = wrapper as HTMLElement;
    if (el.hasAttribute(EXEMPT_ATTR)) return;
    if (el.querySelector(CALENDAR_MARK)) {
      el.setAttribute(EXEMPT_ATTR, "");
    }
  });
}

export function useDatePickerBillingExempt(): void {
  useEffect(() => {
    if (typeof document === "undefined") return;

    // 初次扫描：进入应用时若已有日历面板挂载则立即处理（一般不会有）
    markCalendarPanels(document.body);

    // 长期驻留的 MutationObserver：捕获后续所有 Popover Portal 挂载
    const observer = new MutationObserver((mutations) => {
      for (const m of mutations) {
        m.addedNodes.forEach((node) => {
          if (!(node instanceof HTMLElement)) return;
          // node 本身就是 wrapper，或其子树里含 wrapper
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

    return () => observer.disconnect();
  }, []);
}

export default useDatePickerBillingExempt;
