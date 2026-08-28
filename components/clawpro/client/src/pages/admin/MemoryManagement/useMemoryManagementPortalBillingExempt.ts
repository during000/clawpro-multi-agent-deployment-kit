/**
 * useMemoryManagementPortalBillingExempt
 *
 * 记忆管理页（pages/admin/MemoryManagement/MemoryManagement.tsx 及其子组件
 * DefaultMemoryVersion / ProActivationDialog / ProCloseDialog / OneClickUpgradeDialog
 * / InstanceTable）专用：停服态下，让本页面交互触发的所有 Radix Portal
 *浮层（Dialog / AlertDialog / Select 下拉面板 / Popover / Dropdown /
 * Tooltip / Drawer / Sheet 等）保持"正常可用"—— 100% 不透明、正常配色、
 * 可点击、可选中，与非停服态一致。
 *
 * 触发场景（本次修复的具体表现）
 * ─────────────────────────────────────────────
 * · 停服后打开"新建 Agent 默认记忆版本"弹窗：整块dialog-content 被
 *   AdminDisabledOverlay 灰化+ 事件拦截；右上角关闭 X、"编辑"按钮、
 *   "+ 添加组织策略"等均无法点击 —— 用户被卡在弹窗里出不来。
 *
 * 为什么打标要精细到 Content 节点
 * ─────────────────────────────────────────────
 * · AdminDisabledOverlay 的"恢复规则"对不同 Portal 类型覆盖不一致：
 *
 *   popper-wrapper 类（Select / Popover / Dropdown / Tooltip）：
 *灰化：  body.admin-billing-suspended [data-radix-popper-content-wrapper] button|input|[role=option]…
 *     恢复：  body.admin-billing-suspended [data-radix-popper-content-wrapper] [data-billing-exempt] *
 *     → 仅"后代分支"，没有"wrapper 同元素分支"，
 *       必须打在 wrapper 内部子节点（Radix Content 元素）上。
 *
 *   dialog-content / alert-dialog-content 类：
 *     恢复：  content[data-billing-exempt] *  和  content [data-billing-exempt] *
 *     → 同元素+后代都覆盖，打在 Content 自身即可（本弹窗为此类）。
 *
 *   drawer-content / sheet-content 类：
 *     恢复：  drawer|sheet-content [data-billing-exempt] *
 *     → 仅"后代分支"；额外给Content 的第一个 element 子节点补一个 exempt。
 *
 * · 组件库层面（dialog.tsx / select.tsx / popover.tsx / alert-dialog.tsx 等）
 *   不改（用户约束：仅改本页面）。页面侧用 MutationObserver 监听
 *   document.body，一旦新 Portal Content 挂载即补打 exempt，让 overlay
 *   现成的"豁免分支"CSS 与事件放行链路同时生效。
 *
 * 作用域限定
 * ─────────────────────────────────────────────
 * · Hook 只在记忆管理页挂载期间 attach observer，卸载即disconnect。
 * · 内部叠一层"仅当 body.admin-billing-suspended 存在时才处理"的判断；
 *   非停服态下不介入任何 Portal 节点。
 *
 * "停服前已禁用则延续禁用" 约束
 * ─────────────────────────────────────────────
 * · 本 hook 只在 Portal 内容节点上加一个属性标记，从不改动浮层内任何
 *   元素的 disabled 属性 / aria-disabled /原生表单 disabled / pointer-events inline 样式。
 * · 组件自身传入的 disabled（例如"当前策略不可再选中"的 <SelectItem disabled>、
 *   业务规则算出的 <Button disabled>）依旧生效，overlay 的 CSS 恢复规则
 *   带 :not([disabled]):not([aria-disabled="true"]) 保护（在 overlay 源头
 *   已内置），停服前已禁用的元素继续保持灰化 + 无法点击。
 */
import { useEffect } from "react";

const EXEMPT_ATTR = "data-billing-exempt";

// popper 类（Select/Popover/Dropdown/Tooltip 的 Content 节点，Radix 会把它们放在 wrapper 内）
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
  // popper wrapper 也顺带打上，虽然 CSS 本身对 wrapper 同元素无效，但事件拦截
  // 通过 target.closest('[data-billing-exempt]') 检查—— 打在 wrapper 上能让
  // 事件拦截通过；不能替代 CSS 视觉恢复，仅作补充。
  "[data-radix-popper-content-wrapper]",
];
const ALL_SELECTOR = ALL_SELECTORS.join(", ");
const NEEDS_INNER_EXEMPT_SELECTOR = NEEDS_INNER_EXEMPT_SELECTORS.join(", ");

/** 幂等地在节点上打标 */
function mark(el: Element): void {
  if (!el.hasAttribute(EXEMPT_ATTR)) {
    el.setAttribute(EXEMPT_ATTR, "");
  }
}

/** 对 drawer/sheet-content 类，需要在其内部第一个 element 子节点上再打一个 exempt */
function markInnerIfNeeded(el: Element): void {
  if (!el.matches(NEEDS_INNER_EXEMPT_SELECTOR)) return;
  const inner = el.firstElementChild;
  if (inner) mark(inner);
}

/** 处理一个可能是 Portal Content 的节点 */
function processCandidate(el: Element): void {
  if (!el.matches(ALL_SELECTOR)) return;
  mark(el);
  markInnerIfNeeded(el);
}

/** 扫描 root 下所有已存在的 Portal Content 节点 */
function scanAndMark(root: ParentNode): void {
  root.querySelectorAll?.(ALL_SELECTOR).forEach(processCandidate);
}

export function useMemoryManagementPortalBillingExempt(): void {
  useEffect(() => {
    if (typeof document === "undefined") return;

    const isSuspended = () => document.body.classList.contains("admin-billing-suspended");

    // 初次进入若已经有 Portal 打开着，一并处理
    if (isSuspended()) scanAndMark(document.body);

    const observer = new MutationObserver((mutations) => {
      if (!isSuspended()) return;

      for (const m of mutations) {
        m.addedNodes.forEach((node) => {
          if (!(node instanceof HTMLElement)) return;
          // 情况 1：新增节点本身就是 Portal Content
          processCandidate(node);
          // 情况 2：新增节点子树中含 Portal Content（Radix 有时先挂父层再填充）
          node.querySelectorAll?.(ALL_SELECTOR).forEach(processCandidate);
        });
      }
    });

    observer.observe(document.body, { childList: true, subtree: true });

    return () => observer.disconnect();
  }, []);
}

export default useMemoryManagementPortalBillingExempt;
