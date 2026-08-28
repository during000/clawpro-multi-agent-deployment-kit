/**
 * Portable Popover / DropdownMenu — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Popover / DropdownMenu 时的可移植兜底实现。
 *  - 不依赖 @radix-ui / shadcn / Tailwind；样式由 portable/css/popover-menu.css 提供。
 *  - 自带点击外部 + Esc 关闭；通过 Portal 浮层逃逸滚动容器裁剪，避免被截断。
 *  - 视觉规范（component-specs/popover-dropdown-menu.md §3）：
 *      白底、4px 圆角、蓝灰描边、overlay 轻阴影、z-50；Trigger 36px；
 *      Item text-sm 紧凑、hover bg-subtle；danger item 走 text-danger；
 *      1px 弱分割线；空态单行弱提示；列表过长面板内部滚动。
 *  - 长说明 / 字段说明用 Popover（不要塞 Tooltip）；
 *    危险菜单项只作入口，真正执行前进入 AlertDialog 二次确认。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/popover-menu.css";
 *
 * 用法：
 *   <PortableDropdownMenu
 *     trigger="更多"
 *     items={[
 *       { key: "edit", label: "编辑", onSelect: onEdit },
 *       { type: "separator" },
 *       { key: "del", label: "删除", danger: true, onSelect: () => setConfirm(true) },
 *     ]}
 *     usePortal={true}  // 可选，默认 true，设置 false 使用 absolute 定位
 *   />
 *
 *   <PortablePopover trigger={<button>打开</button>} usePortal={true}>
 *     <p>这是一段说明……</p>
 *   </PortablePopover>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

/* ───────────── 共享：点击外部 + Esc 关闭 ───────────── */

function useDismiss(
  open: boolean,
  setOpen: (v: boolean) => void,
  rootRef: React.RefObject<HTMLElement>
) {
  React.useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open, setOpen, rootRef]);
}

/* ───────────── Portal 渲染工具 ───────────── */

interface PortalPanelProps {
  open: boolean;
  triggerRef: React.RefObject<HTMLElement>;
  panel: React.ReactNode;
  className: string;
  usePortal?: boolean;
}

/**
 * 通过 Portal 或原位置渲染面板
 * 当 usePortal=true 时，通过 createPortal 挂载到 document.body
 * 当 usePortal=false 时，在原位置用 absolute 定位（可能被 overflow 裁剪）
 */
function PanelRenderer({
  open,
  triggerRef,
  panel,
  className,
  usePortal = true,
}: PortalPanelProps) {
  const [position, setPosition] = React.useState<{ top: number; left: number } | null>(null);

  React.useEffect(() => {
    if (!open || !triggerRef.current) return;

    const updatePosition = () => {
      const trigger = triggerRef.current;
      if (!trigger) return;

      const rect = trigger.getBoundingClientRect();
      // 面板距 trigger 下方 4px
      setPosition({
        top: rect.bottom + 4 + window.scrollY,
        left: rect.left + window.scrollX,
      });
    };

    updatePosition();
    window.addEventListener("scroll", updatePosition);
    window.addEventListener("resize", updatePosition);

    return () => {
      window.removeEventListener("scroll", updatePosition);
      window.removeEventListener("resize", updatePosition);
    };
  }, [open, triggerRef]);

  if (!open) return null;

  const panelContent = <div className={className}>{panel}</div>;

  if (!usePortal) {
    // 直接渲染在原位置（absolute 定位由 CSS 处理）
    return panelContent;
  }

  // 通过 Portal 到 body，使用计算的绝对位置
  if (position) {
    return React.createPortal(
      <div
        style={{
          position: "fixed",
          top: `${position.top}px`,
          left: `${position.left}px`,
        }}
      >
        {panelContent}
      </div>,
      document.body
    );
  }

  return null;
}

/* ───────────── PortableDropdownMenu ───────────── */

export type PortableMenuItem =
  | {
      type?: "item";
      key: string;
      label: React.ReactNode;
      icon?: React.ReactNode;
      danger?: boolean;
      disabled?: boolean;
      onSelect?: () => void;
    }
  | { type: "separator"; key?: string };

export interface PortableDropdownMenuProps {
  trigger: React.ReactNode;
  items: PortableMenuItem[];
  /** 面板对齐，默认 start（左对齐） */
  align?: "start" | "end";
  /** 无可用项时的空态文案 */
  emptyText?: React.ReactNode;
  className?: string;
  panelClassName?: string;
  /** 是否通过 Portal 逃逸滚动容器裁剪，默认 true */
  usePortal?: boolean;
}

export function PortableDropdownMenu({
  trigger,
  items,
  align = "start",
  emptyText = "暂无操作",
  className = "",
  panelClassName = "",
  usePortal = true,
}: PortableDropdownMenuProps) {
  const [open, setOpen] = React.useState(false);
  const rootRef = React.useRef<HTMLDivElement>(null);
  const triggerRef = React.useRef<HTMLButtonElement>(null);
  useDismiss(open, setOpen, rootRef);

  const root = ["cp-menu", className].filter(Boolean).join(" ");
  const panel = [
    "cp-menu__panel",
    `cp-menu__panel--${align}`,
    panelClassName,
  ]
    .filter(Boolean)
    .join(" ");

  const actionable = items.filter((i) => i.type !== "separator");

  const panelContent = (
    <>
      {actionable.length === 0 ? (
        <div className="cp-menu__empty">{emptyText}</div>
      ) : (
        items.map((item, i) => {
          if (item.type === "separator") {
            return (
              <div
                key={item.key ?? `sep-${i}`}
                className="cp-menu__separator"
                role="separator"
              />
            );
          }
          const cls = [
            "cp-menu__item",
            item.danger && "cp-menu__item--danger",
          ]
            .filter(Boolean)
            .join(" ");
          return (
            <button
              key={item.key}
              type="button"
              role="menuitem"
              disabled={item.disabled}
              className={cls}
              onClick={() => {
                if (item.disabled) return;
                item.onSelect?.();
                setOpen(false);
              }}
            >
              {item.icon != null && (
                <span className="cp-menu__item-icon">{item.icon}</span>
              )}
              <span>{item.label}</span>
            </button>
          );
        })
      )}
    </>
  );

  return (
    <div ref={rootRef} className={root}>
      <button
        ref={triggerRef}
        type="button"
        className="cp-menu__trigger"
        data-state={open ? "open" : "closed"}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        {trigger}
      </button>
      <PanelRenderer
        open={open}
        triggerRef={triggerRef}
        panel={<div role="menu">{panelContent}</div>}
        className={panel}
        usePortal={usePortal}
      />
    </div>
  );
}

/* ───────────── PortablePopover ───────────── */

export interface PortablePopoverProps {
  trigger: React.ReactNode;
  children: React.ReactNode;
  align?: "start" | "end";
  className?: string;
  panelClassName?: string;
  /** 受控开关（不传则内部维护） */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** 是否通过 Portal 逃逸滚动容器裁剪，默认 true */
  usePortal?: boolean;
}

export function PortablePopover({
  trigger,
  children,
  align = "start",
  className = "",
  panelClassName = "",
  open: openProp,
  onOpenChange,
  usePortal = true,
}: PortablePopoverProps) {
  const [openState, setOpenState] = React.useState(false);
  const open = openProp ?? openState;
  const setOpen = React.useCallback(
    (v: boolean) => {
      setOpenState(v);
      onOpenChange?.(v);
    },
    [onOpenChange]
  );
  const rootRef = React.useRef<HTMLDivElement>(null);
  const triggerRef = React.useRef<HTMLSpanElement>(null);
  useDismiss(open, setOpen, rootRef);

  const root = ["cp-popover", className].filter(Boolean).join(" ");
  const panel = [
    "cp-popover__panel",
    align === "end" && "cp-popover__panel--end",
    panelClassName,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div ref={rootRef} className={root}>
      <span
        ref={triggerRef}
        onClick={() => setOpen(!open)}
        style={{ cursor: "pointer" }}
      >
        {trigger}
      </span>
      <PanelRenderer
        open={open}
        triggerRef={triggerRef}
        panel={<div role="dialog">{children}</div>}
        className={panel}
        usePortal={usePortal}
      />
    </div>
  );
}
