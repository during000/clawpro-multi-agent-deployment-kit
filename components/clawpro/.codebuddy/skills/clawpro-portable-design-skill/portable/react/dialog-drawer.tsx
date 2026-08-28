/**
 * Portable Dialog / AlertDialog / Drawer — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Modal / Drawer 时的可移植兜底实现。
 *  - 不依赖 @radix-ui / @floating-ui / shadcn；只用 React Portal + 一个 Esc 监听。
 *  - 视觉规范（spec/component-specs/dialog-drawer.md §3）：
 *      容器：白底 + 蓝灰描边 + 4px 圆角（管控端）+ overlay 阴影
 *      Header：title 16px / semibold / var(--cp-text-title) + 可选 description
 *      Footer：右对齐，主次按钮分明（次按钮 claw-outline，主按钮 dialog-confirm）
 *  - 不依赖 Tailwind；样式由 portable/css/dialog-drawer.css 提供。
 *    ⚠️ 必须同时引入：import "../css/tokens.css"; import "../css/dialog-drawer.css";
 *  - 已包含 3 件套：
 *      PortableDialog       一般信息 / 表单弹窗
 *      PortableAlertDialog  危险确认（视觉同 Dialog，语义不同：Esc 不关闭、强制 footer）
 *      PortableDrawer       右侧侧滑面板
 *
 * 基础约束：
 *  - 受控：调用方维护 `open` + `onOpenChange`。
 *  - 焦点 / 滚动锁定：Body overflow:hidden + Esc 监听；不实现高级 focus trap，
 *    宿主仓如对 a11y 有强诉求请改用 @radix-ui/react-dialog。
 *  - 圆角硬铁律：管控端只准 4px，不要在 className 上覆盖。
 *
 * 用法：
 *   <PortableDialog open={open} onOpenChange={setOpen} title="添加文档">
 *     <p>表单内容...</p>
 *     <PortableDialogFooter>
 *       <PortableButton variant="claw-outline" onClick={() => setOpen(false)}>取消</PortableButton>
 *       <PortableButton variant="dialog-confirm" onClick={save}>确认</PortableButton>
 *     </PortableDialogFooter>
 *   </PortableDialog>
 *
 *   <PortableAlertDialog open={open} onOpenChange={setOpen} title="确认删除？" tone="danger">
 *     此操作不可撤销。
 *     <PortableDialogFooter>
 *       <PortableButton variant="claw-outline" onClick={() => setOpen(false)}>取消</PortableButton>
 *       <PortableButton variant="destructive" onClick={remove}>确认删除</PortableButton>
 *     </PortableDialogFooter>
 *   </PortableAlertDialog>
 *
 *   <PortableDrawer open={open} onOpenChange={setOpen} title="详情" width={520}>
 *     ...
 *   </PortableDrawer>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

/* ───────────── Portal helper ─────────────
 * SSR 安全：服务端不渲染 portal；ssr 完成 hydrate 后再 mount。
 */
function PortablePortal({ children }: { children: React.ReactNode }) {
  const [mounted, setMounted] = React.useState(false);
  React.useEffect(() => setMounted(true), []);
  if (!mounted || typeof document === "undefined") return null;
  // 动态 import 避免 SSR：直接用 ReactDOM.createPortal
  // 这里手动写 require 走纯客户端，避免依赖额外库
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const { createPortal } = require("react-dom") as typeof import("react-dom");
  return createPortal(children, document.body);
}

/* ───────────── 共享 hook：滚动锁 + Esc 关闭 ───────────── */

function useDialogA11y(open: boolean, onClose: () => void, options?: { closeOnEsc?: boolean }) {
  const closeOnEsc = options?.closeOnEsc ?? true;
  React.useEffect(() => {
    if (!open || typeof document === "undefined") return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKey = (e: KeyboardEvent) => {
      if (closeOnEsc && e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = prev;
      document.removeEventListener("keydown", onKey);
    };
  }, [open, onClose, closeOnEsc]);
}

/* ───────────── Overlay ───────────── */

function PortableOverlay({ onClick }: { onClick?: () => void }) {
  return (
    <div
      aria-hidden="true"
      onClick={onClick}
      className="cp-dialog-overlay"
    />
  );
}

/* ───────────── PortableDialog ───────────── */

export interface PortableDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: React.ReactNode;
  description?: React.ReactNode;
  children?: React.ReactNode;
  /** 容器 max-width，默认 560 */
  maxWidth?: number | string;
  /** 点击遮罩是否关闭，默认 true */
  closeOnOverlay?: boolean;
  /** 按 Esc 是否关闭，默认 true（AlertDialog 默认 false） */
  closeOnEsc?: boolean;
  className?: string;
}

const DIALOG_CONTAINER = "cp-dialog";

export function PortableDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  maxWidth = 560,
  closeOnOverlay = true,
  closeOnEsc = true,
  className = "",
}: PortableDialogProps) {
  const close = React.useCallback(() => onOpenChange(false), [onOpenChange]);
  useDialogA11y(open, close, { closeOnEsc });
  if (!open) return null;
  return (
    <PortablePortal>
      <PortableOverlay onClick={closeOnOverlay ? close : undefined} />
      <div role="dialog" aria-modal="true" className="cp-dialog-positioner">
        <section
          className={[DIALOG_CONTAINER, className].filter(Boolean).join(" ")}
          style={{ maxWidth }}
          onClick={(e) => e.stopPropagation()}
        >
          <header className="cp-dialog__header">
            <h2 className="cp-dialog__title">{title}</h2>
            {description ? (
              <p className="cp-dialog__desc">{description}</p>
            ) : null}
          </header>
          <div className="cp-dialog__body">{children}</div>
        </section>
      </div>
    </PortablePortal>
  );
}

/* ───────────── PortableAlertDialog（危险 / 强确认） ─────────────
 * 与 PortableDialog 视觉一致；语义上：
 *   - 默认不允许 Esc / 点遮罩关闭，强制走 Footer 按钮
 *   - title 可选 tone="danger"，文字 var(--cp-text-danger)
 */

export interface PortableAlertDialogProps extends Omit<PortableDialogProps, "closeOnOverlay" | "closeOnEsc"> {
  tone?: "default" | "danger";
  /** 危险确认默认禁止遮罩关闭 / Esc 关闭，需要可放开 */
  closeOnOverlay?: boolean;
  closeOnEsc?: boolean;
}

export function PortableAlertDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  maxWidth = 420,
  tone = "default",
  closeOnOverlay = false,
  closeOnEsc = false,
  className = "",
}: PortableAlertDialogProps) {
  const close = React.useCallback(() => onOpenChange(false), [onOpenChange]);
  useDialogA11y(open, close, { closeOnEsc });
  if (!open) return null;
  const titleClass =
    tone === "danger"
      ? "cp-dialog__title cp-dialog__title--danger"
      : "cp-dialog__title";
  return (
    <PortablePortal>
      <PortableOverlay onClick={closeOnOverlay ? close : undefined} />
      <div role="alertdialog" aria-modal="true" className="cp-dialog-positioner">
        <section
          className={[DIALOG_CONTAINER, className].filter(Boolean).join(" ")}
          style={{ maxWidth }}
          onClick={(e) => e.stopPropagation()}
        >
          <header className="cp-dialog__header">
            <h2 className={titleClass}>{title}</h2>
            {description ? (
              <p className="cp-dialog__desc">{description}</p>
            ) : null}
          </header>
          <div className="cp-dialog__body">{children}</div>
        </section>
      </div>
    </PortablePortal>
  );
}

/* ───────────── PortableDrawer ─────────────
 * 右侧侧滑面板，全高度，适合详情 / 长表单。
 * 视觉：白底 + 蓝灰左描边 + overlay 阴影；圆角 0（贴右）。
 */

export interface PortableDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: React.ReactNode;
  description?: React.ReactNode;
  children?: React.ReactNode;
  /** 抽屉宽度，默认 480 */
  width?: number | string;
  closeOnOverlay?: boolean;
  closeOnEsc?: boolean;
  className?: string;
}

export function PortableDrawer({
  open,
  onOpenChange,
  title,
  description,
  children,
  width = 480,
  closeOnOverlay = true,
  closeOnEsc = true,
  className = "",
}: PortableDrawerProps) {
  const close = React.useCallback(() => onOpenChange(false), [onOpenChange]);
  useDialogA11y(open, close, { closeOnEsc });
  if (!open) return null;
  const merged = ["cp-drawer", className].filter(Boolean).join(" ");
  return (
    <PortablePortal>
      <PortableOverlay onClick={closeOnOverlay ? close : undefined} />
      <aside
        role="dialog"
        aria-modal="true"
        className={merged}
        style={{ width }}
        onClick={(e) => e.stopPropagation()}
      >
        <header className="cp-drawer__header">
          <h2 className="cp-dialog__title">{title}</h2>
          {description ? (
            <p className="cp-dialog__desc">{description}</p>
          ) : null}
        </header>
        <div className="cp-drawer__body">{children}</div>
      </aside>
    </PortablePortal>
  );
}

/* ───────────── PortableDialogFooter ─────────────
 * 右对齐，gap-2；调用方放 PortableButton。
 */
export function PortableDialogFooter({
  children,
  className = "",
}: {
  children: React.ReactNode;
  className?: string;
}) {
  const merged = ["cp-dialog-footer", className].filter(Boolean).join(" ");
  return <footer className={merged}>{children}</footer>;
}
