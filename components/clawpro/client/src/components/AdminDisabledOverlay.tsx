/**
 * AdminDisabledOverlay - 管控端停服全局禁用层
 *
 * 当管控台到期停服时，在管控端所有页面上注入全局 CSS，
 * 禁用所有操作按钮使其置灰，但保留 hover 事件（用于提示原因）。
 * 点击时拦截并弹出 toast 提示"管控台已到期，请续费后操作"。
 *
 * 豁免规则（保持可用，不受停服影响）：
 *   1. 元素自身或祖先带 [data-billing-exempt]（业务侧显式标记，一劳永逸）
 *   2. 搜索框（type="search" 或 placeholder 以"搜索"开头的 input）—— 只读态下搜索属于查看类操作
 */
import { useEffect, useRef } from "react";
import { toast } from "sonner";
import { useServiceStatus } from "@/contexts/ServiceStatusContext";

const DISABLED_STYLE_ID = "admin-disabled-overlay-style";

/**
 * 搜索框识别：
 *   - type="search"（<input type="search">）
 *   - 或 placeholder 以"搜索"开头（如："搜索技能名称..."、"搜索用户 ID..."）
 * 只读态下搜索属查看操作，保留可用。
 */
const SEARCH_INPUT_SELECTOR = 'input[type="search"], input[placeholder^="搜索"]';

const DISABLED_CSS = `
/* 管控端停服全局禁用样式 - 保留 hover（不加 pointer-events:none） */
.admin-service-suspended button,
.admin-service-suspended a[href],
.admin-service-suspended input,
.admin-service-suspended select,
.admin-service-suspended textarea,
.admin-service-suspended [role="switch"],
.admin-service-suspended [role="checkbox"],
.admin-service-suspended [role="combobox"],
/* Radix Portal 浮层（挂在 body 下、不在 <main> 内）也要视觉禁用
   包括：Popover/Dropdown/Tooltip（popper-content-wrapper）、Dialog/AlertDialog/Drawer/Sheet */
body.admin-billing-suspended [data-radix-popper-content-wrapper] button,
body.admin-billing-suspended [data-radix-popper-content-wrapper] a[href],
body.admin-billing-suspended [data-radix-popper-content-wrapper] input,
body.admin-billing-suspended [data-radix-popper-content-wrapper] [role="switch"],
body.admin-billing-suspended [data-radix-popper-content-wrapper] [role="checkbox"],
body.admin-billing-suspended [data-radix-popper-content-wrapper] [role="combobox"],
body.admin-billing-suspended [data-radix-popper-content-wrapper] [role="menuitem"],
body.admin-billing-suspended [data-radix-popper-content-wrapper] [role="option"],
body.admin-billing-suspended [data-slot="dialog-content"] button,
body.admin-billing-suspended [data-slot="dialog-content"] a[href],
body.admin-billing-suspended [data-slot="dialog-content"] input,
body.admin-billing-suspended [data-slot="dialog-content"] textarea,
body.admin-billing-suspended [data-slot="dialog-content"] [role="switch"],
body.admin-billing-suspended [data-slot="dialog-content"] [role="checkbox"],
body.admin-billing-suspended [data-slot="dialog-content"] [role="combobox"],
body.admin-billing-suspended [data-slot="alert-dialog-content"] button,
body.admin-billing-suspended [data-slot="alert-dialog-content"] a[href],
body.admin-billing-suspended [data-slot="alert-dialog-content"] input,
body.admin-billing-suspended [data-slot="alert-dialog-content"] textarea,
body.admin-billing-suspended [data-slot="alert-dialog-content"] [role="switch"],
body.admin-billing-suspended [data-slot="alert-dialog-content"] [role="checkbox"],
body.admin-billing-suspended [data-slot="alert-dialog-content"] [role="combobox"],
body.admin-billing-suspended [data-slot="drawer-content"] button,
body.admin-billing-suspended [data-slot="drawer-content"] a[href],
body.admin-billing-suspended [data-slot="drawer-content"] input,
body.admin-billing-suspended [data-slot="drawer-content"] textarea,
body.admin-billing-suspended [data-slot="drawer-content"] [role="switch"],
body.admin-billing-suspended [data-slot="drawer-content"] [role="checkbox"],
body.admin-billing-suspended [data-slot="drawer-content"] [role="combobox"],
body.admin-billing-suspended [data-slot="sheet-content"] button,
body.admin-billing-suspended [data-slot="sheet-content"] a[href],
body.admin-billing-suspended [data-slot="sheet-content"] input,
body.admin-billing-suspended [data-slot="sheet-content"] textarea,
body.admin-billing-suspended [data-slot="sheet-content"] [role="switch"],
body.admin-billing-suspended [data-slot="sheet-content"] [role="checkbox"],
body.admin-billing-suspended [data-slot="sheet-content"] [role="combobox"] {
  opacity: 0.4 !important;
  cursor: not-allowed !important;
}

/* Portal 浮层内的豁免元素恢复正常态（自身或祖先带 data-billing-exempt） */
body.admin-billing-suspended [data-radix-popper-content-wrapper] [data-billing-exempt],
body.admin-billing-suspended [data-radix-popper-content-wrapper] [data-billing-exempt] button,
body.admin-billing-suspended [data-radix-popper-content-wrapper] [data-billing-exempt] a[href],
body.admin-billing-suspended [data-radix-popper-content-wrapper] [data-billing-exempt] *,
body.admin-billing-suspended [data-slot="dialog-content"][data-billing-exempt],
body.admin-billing-suspended [data-slot="dialog-content"][data-billing-exempt] *,
body.admin-billing-suspended [data-slot="dialog-content"] [data-billing-exempt],
body.admin-billing-suspended [data-slot="dialog-content"] [data-billing-exempt] *,
body.admin-billing-suspended [data-slot="alert-dialog-content"][data-billing-exempt],
body.admin-billing-suspended [data-slot="alert-dialog-content"][data-billing-exempt] *,
body.admin-billing-suspended [data-slot="alert-dialog-content"] [data-billing-exempt],
body.admin-billing-suspended [data-slot="alert-dialog-content"] [data-billing-exempt] *,
body.admin-billing-suspended [data-slot="drawer-content"] [data-billing-exempt],
body.admin-billing-suspended [data-slot="drawer-content"] [data-billing-exempt] *,
body.admin-billing-suspended [data-slot="sheet-content"] [data-billing-exempt],
body.admin-billing-suspended [data-slot="sheet-content"] [data-billing-exempt] * {
  opacity: 1 !important;
  cursor: pointer !important;
}

/*
 * 豁免规则（导航/查看/搜索类元素在停服只读模式下保持可用）：
 *   - 元素自身带 [data-billing-exempt]
 *   - 或祖先带 [data-billing-exempt]（容器整体豁免）
 *   - 或元素本身是搜索框（type="search" 或 placeholder 以"搜索"开头）
 */
.admin-service-suspended [data-billing-exempt],
.admin-service-suspended [data-billing-exempt] button,
.admin-service-suspended [data-billing-exempt] a[href],
.admin-service-suspended [data-billing-exempt] input,
.admin-service-suspended [data-billing-exempt] select,
.admin-service-suspended [data-billing-exempt] textarea,
.admin-service-suspended [data-billing-exempt] [role="switch"],
.admin-service-suspended [data-billing-exempt] [role="checkbox"],
.admin-service-suspended [data-billing-exempt] [role="combobox"],
.admin-service-suspended [data-billing-exempt] *,
/* 元素自身即豁免的交互元素（例如 <a href data-billing-exempt>）
   需要比灰化 ".admin-service-suspended a[href]" 更高的特异性，故这里显式列出 */
.admin-service-suspended a[href][data-billing-exempt],
.admin-service-suspended button[data-billing-exempt],
.admin-service-suspended input[data-billing-exempt],
.admin-service-suspended select[data-billing-exempt],
.admin-service-suspended textarea[data-billing-exempt],
.admin-service-suspended [role="switch"][data-billing-exempt],
.admin-service-suspended [role="checkbox"][data-billing-exempt],
.admin-service-suspended [role="combobox"][data-billing-exempt],
.admin-service-suspended input[type="search"],
.admin-service-suspended input[placeholder^="搜索"] {
  opacity: 1 !important;
  cursor: pointer !important;
}

/* 光标 pointer 只适用于交互元素，其他子元素恢复默认光标（避免文本区域出现手型） */
.admin-service-suspended [data-billing-exempt]:not(button):not(a):not(input):not(select):not(textarea):not([role="switch"]):not([role="checkbox"]):not([role="combobox"]),
.admin-service-suspended [data-billing-exempt] *:not(button):not(a):not(input):not(select):not(textarea):not([role="switch"]):not([role="checkbox"]):not([role="combobox"]) {
  cursor: auto !important;
}

/* 搜索框恢复文字输入光标 */
.admin-service-suspended input[type="search"],
.admin-service-suspended input[placeholder^="搜索"] {
  cursor: text !important;
}
`;

// 防抖：避免短时间内重复弹 toast
const TOAST_THROTTLE_MS = 1500;

/** 判断元素是否为豁免的搜索框 */
function isExemptSearchInput(el: HTMLElement): boolean {
  return el.matches(SEARCH_INPUT_SELECTOR);
}

export default function AdminDisabledOverlay() {
  const { isAdminDisabled } = useServiceStatus();
  const lastToastRef = useRef(0);

  useEffect(() => {
    if (isAdminDisabled) {
      // 注入禁用样式
      if (!document.getElementById(DISABLED_STYLE_ID)) {
        const style = document.createElement("style");
        style.id = DISABLED_STYLE_ID;
        style.textContent = DISABLED_CSS;
        document.head.appendChild(style);
      }

      // 给 main 内容区加 class（覆盖主内容区所有交互元素）
      const mainEl = document.querySelector("main");
      if (mainEl) {
        mainEl.classList.add("admin-service-suspended");
      }
      // 同时给 body 加 class，供 Radix Portal 挂到 body 下的浮层 CSS 命中
      document.body.classList.add("admin-billing-suspended");

      // 判定元素是否属于「管控端管辖范围」：
      //   - 在 <main> 内：主要业务区
      //   - 或渲染到 Radix Portal（Popover/Dropdown/Dialog/AlertDialog/Drawer/Sheet 等浮层）
      const isInAdminScope = (el: Element): boolean => {
        if (mainEl?.contains(el)) return true;
        // 常见浮层容器（Radix 会给这些属性/角色/data-slot）
        return !!el.closest(
          '[data-radix-popper-content-wrapper], ' +
          '[data-slot="popover-content"], ' +
          '[data-slot="dropdown-menu-content"], ' +
          '[data-slot="dialog-content"], ' +
          '[data-slot="alert-dialog-content"], ' +
          '[data-slot="drawer-content"], ' +
          '[data-slot="sheet-content"], ' +
          '[role="dialog"], [role="alertdialog"], [role="menu"], [role="listbox"], [role="tooltip"]'
        );
      };

      // 拦截点击事件，弹 toast 提示
      const handleClick = (e: MouseEvent) => {
        const target = e.target as HTMLElement;
        if (target.closest("[data-billing-exempt]")) return;
        // 搜索框：豁免，不拦截
        if (target instanceof HTMLElement && isExemptSearchInput(target)) return;

        const interactiveEl = target.closest(
          "button, a[href], input, select, textarea, [role='switch'], [role='checkbox'], [role='combobox'], [role='menuitem'], [role='option']"
        );
        if (interactiveEl && isInAdminScope(interactiveEl)) {
          e.preventDefault();
          e.stopPropagation();
          e.stopImmediatePropagation();

          // 防抖弹 toast
          const now = Date.now();
          if (now - lastToastRef.current > TOAST_THROTTLE_MS) {
            lastToastRef.current = now;
            toast.error("管控台已到期，请续费后操作");
          }
        }
      };

      // 阻止 input/textarea 获得焦点
      const handleFocus = (e: FocusEvent) => {
        const target = e.target as HTMLElement;
        if (target.closest("[data-billing-exempt]")) return;
        // 搜索框：豁免，允许聚焦
        if (target instanceof HTMLElement && isExemptSearchInput(target)) return;

        const isInput = target.matches("input, select, textarea, [role='combobox']");
        if (isInput && isInAdminScope(target)) {
          e.preventDefault();
          (target as HTMLElement).blur();
        }
      };

      document.addEventListener("click", handleClick, true);
      document.addEventListener("mousedown", handleClick, true);
      document.addEventListener("focusin", handleFocus, true);

      return () => {
        document.removeEventListener("click", handleClick, true);
        document.removeEventListener("mousedown", handleClick, true);
        document.removeEventListener("focusin", handleFocus, true);
        const style = document.getElementById(DISABLED_STYLE_ID);
        if (style) style.remove();
        if (mainEl) mainEl.classList.remove("admin-service-suspended");
        document.body.classList.remove("admin-billing-suspended");
      };
    } else {
      const style = document.getElementById(DISABLED_STYLE_ID);
      if (style) style.remove();
      const mainEl = document.querySelector("main");
      if (mainEl) mainEl.classList.remove("admin-service-suspended");
      document.body.classList.remove("admin-billing-suspended");
    }
  }, [isAdminDisabled]);

  return null;
}
