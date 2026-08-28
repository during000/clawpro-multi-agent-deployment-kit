/**
 * useAIAgentPortalBillingExempt
 *
 * AI Agent 安全模块专用：停服态下，让本模块交互触发的所有 Radix Portal 浮层
 *（Select / Dropdown / Popover / Tooltip / Dialog / AlertDialog / Drawer / Sheet 等）
 * 保持"正常可用"—— 100% 不透明、正常配色、可点击、可选中，与非停服态一致。
 *
 * 为什么打标要精细到 Content 节点而不是 popper-wrapper
 * ─────────────────────────────────────────────
 * AdminDisabledOverlay 的"恢复规则"在不同 Portal 类型上覆盖不一致：
 *
 *   popper-wrapper 类（Select / Popover / Dropdown / Tooltip）：
 *     灰化：  body.admin-billing-suspended [data-radix-popper-content-wrapper] button|input|[role=option]…
 *     恢复：  body.admin-billing-suspended [data-radix-popper-content-wrapper] [data-billing-exempt] *
 *     → 仅"后代分支"，没有"wrapper 同元素分支"。
 *     → 把 exempt 打在 wrapper 自身完全无效——必须打在 wrapper 内部（Radix Content 元素）上。
 *
 *   dialog-content / alert-dialog-content 类：
 *     恢复：  content[data-billing-exempt] * 和 content [data-billing-exempt] *都有
 *     → 打在 Content 自身即可。
 *
 *   drawer-content / sheet-content 类：
 *     恢复：  drawer|sheet-content [data-billing-exempt] *
 *     → 仅"后代分支"；打在 Content 自身无效，必须打在 Content 内部一个节点上。
 *     本 hook 额外给Content 的第一个 element 子节点补一个 exempt。
 *
 * 组件库层面不改（用户约束：仅本模块）。在页面侧用 MutationObserver 给新
 * 出现的 Portal Content 补打 exempt，让 overlay 现成的"豁免分支"CSS 与事件拦截同时生效。
 *
 * 作用域限定
 * ─────────────────────────────────────────────
 * · Hook 只在 AI Agent 页面组件挂载期间 attach observer，卸载即disconnect。
 * · AI Agent 是路由级页面（/security/aiagent）。
 * · Hook 内部叠一层"仅当 body.admin-billing-suspended 存在时才处理"的判断。
 *
 * "停服前已禁用则延续禁用" 约束
 * ─────────────────────────────────────────────
 * · 本 hook 只在 Portal 内容节点上加一个属性标记，不改动浮层内任何元素的
 *   disabled 属性、aria-disabled、原生表单 disabled、pointer-events inline 样式。
 */
import { useEffect } from "react";

const EXEMPT_ATTR = "data-billing-exempt";

// popper类（Select/Popover/Dropdown/Tooltip 的 Content 节点）
// CSS 恢复分支只有后代分支： [popper-wrapper] [data-billing-exempt] * —— 需要在 wrapper 内
const POPPER_CONTENT_SELECTORS = [
  '[data-slot="select-content"]',
  '[data-slot="popover-content"]',
  '[data-slot="dropdown-menu-content"]',
  '[data-slot="tooltip-content"]',
  '[data-slot="menubar-content"]',
  '[data-slot="context-menu-content"]',
  '[data-slot="hover-card-content"]',
  '[role="menu"]',
  '[role="listbox"]',
  '[role="tooltip"]',
];

// 同元素恢复分支已覆盖（打在 Content 自身即可）
const DIALOG_CONTENT_SELECTORS = [
  '[data-slot="dialog-content"]',
  '[data-slot="alert-dialog-content"]',
  '[role="dialog"]',
  '[role="alertdialog"]',
];

// 仅有后代恢复分支（同元素无效，需要给内部子节点再补一个 exempt）
const NEEDS_INNER_EXEMPT_SELECTORS = [
  '[data-slot="drawer-content"]',
  '[data-slot="sheet-content"]',
];

const ALL_SELECTORS = [
  ...POPPER_CONTENT_SELECTORS,
  ...DIALOG_CONTENT_SELECTORS,
  ...NEEDS_INNER_EXEMPT_SELECTORS,
  // popper wrapper 顺带打标——CSS 视觉恢复对wrapper 同元素无效，但事件拦截
  // 用 target.closest('[data-billing-exempt]') 检查，打在 wrapper 上能兜住
  // 事件放行（视觉恢复仍由内层 Content 上的 exempt 完成）
  "[data-radix-popper-content-wrapper]",
];
const ALL_SELECTOR = ALL_SELECTORS.join(", ");
const NEEDS_INNER_EXEMPT_SELECTOR = NEEDS_INNER_EXEMPT_SELECTORS.join(", ");

function mark(el: Element): void {
  if (!el.hasAttribute(EXEMPT_ATTR)) {
    el.setAttribute(EXEMPT_ATTR, "");
  }
}

function markInnerIfNeeded(el: Element): void {
  if (!el.matches(NEEDS_INNER_EXEMPT_SELECTOR)) return;
  const inner = el.firstElementChild;
  if (inner) mark(inner);
}

function processCandidate(el: Element): void {
  if (!el.matches(ALL_SELECTOR)) return;
  mark(el);
  markInnerIfNeeded(el);
}

function scanAndMark(root: ParentNode): void {
  root.querySelectorAll?.(ALL_SELECTOR).forEach(processCandidate);
}

export function useAIAgentPortalBillingExempt(): void {
  useEffect(() => {
    if (typeof document === "undefined") return;

    const isSuspended = () => document.body.classList.contains("admin-billing-suspended");

    if (isSuspended()) scanAndMark(document.body);

    const observer = new MutationObserver((mutations) => {
      if (!isSuspended()) return;

      for (const m of mutations) {
        m.addedNodes.forEach((node) => {
          if (!(node instanceof HTMLElement)) return;
          processCandidate(node);
          node.querySelectorAll?.(ALL_SELECTOR).forEach(processCandidate);
        });
      }
    });

    observer.observe(document.body, { childList: true, subtree: true });

    return () => observer.disconnect();
  }, []);
}

export default useAIAgentPortalBillingExempt;
