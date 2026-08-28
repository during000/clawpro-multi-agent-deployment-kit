/**
 * Portable Alert — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 与 demo 仓 client/src/components/ui/alert.tsx + admin-notice-alert.tsx
 * 1:1 对齐的可移植兜底（spec/component-specs/alert.md）。
 *
 *  - 去除外部依赖：不依赖 cva / shadcn / @/lib/utils / @/components/ui/Typography，
 *    lucide 图标全部内联为 SVG（Info / CheckCircle2 / XCircle / CircleAlert / Sparkles）。
 *  - 仅需 React + 配套 portable/css/alert.css（承载 token 与排版）。
 *
 * 6 个 variant（spec §4.2）：info / operation-info / warning / product-news / success / error
 *
 * 用法：
 *   import "../css/alert.css";
 *   import {
 *     Alert, AlertTitle, AlertDescription,
 *     AlertInfoIcon, AlertOperationInfoIcon, AlertProductNewsIcon,
 *     AlertWarningIcon, AlertSuccessIcon, AlertErrorIcon,
 *     AdminNoticeAlert,
 *   } from "./alert";
 *
 *   // 普通信息
 *   <Alert variant="info">
 *     <AlertInfoIcon />
 *     <AlertDescription>数据每 5 分钟更新一次</AlertDescription>
 *   </Alert>
 *
 *   // 带标题
 *   <Alert variant="warning">
 *     <AlertWarningIcon />
 *     <AlertTitle>注意事项</AlertTitle>
 *     <AlertDescription>有 3 项基础配置未完成</AlertDescription>
 *   </Alert>
 *
 *   // 带右侧操作区
 *   <Alert variant="warning" withAction>
 *     <AlertWarningIcon />
 *     <AlertDescription>配额不足</AlertDescription>
 *     <span className="cp-alert__action"><button>去处理</button></span>
 *   </Alert>
 *
 *   // 管控端顶部公告条
 *   <AdminNoticeAlert type="pending-config" controls={<span>4/5</span>}>
 *     <span>有 3 项基础配置未完成，</span>
 *     <a href="#">前往基础信息配置处理</a>
 *   </AdminNoticeAlert>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(" ");
}

/* ── Alert 容器 ───────────────────────────────────────────────────────── */
export type AlertVariant =
  | "info"
  | "operation-info"
  | "warning"
  | "product-news"
  | "success"
  | "error";

export interface AlertProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: AlertVariant;
  /** 启用右侧操作区布局（spec §5 带右侧操作区），子节点用 .cp-alert__action 承载 */
  withAction?: boolean;
}

export function Alert({
  variant = "info",
  withAction = false,
  className,
  ...props
}: AlertProps) {
  return (
    <div
      role="alert"
      className={cx(
        "cp-alert",
        `cp-alert--${variant}`,
        withAction && "cp-alert--with-action",
        className
      )}
      {...props}
    />
  );
}

export function AlertTitle({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cx("cp-alert__title", className)} {...props} />;
}

export function AlertDescription({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cx("cp-alert__description", className)} {...props} />;
}

/* ── 图标（内联 SVG，外层包 .cp-alert__icon 以命中 has-[>svg] 列布局） ──── */
type IconProps = React.SVGProps<SVGSVGElement>;

function IconShell({ children }: { children: React.ReactNode }) {
  return (
    <span className="cp-alert__icon" aria-hidden="true">
      {children}
    </span>
  );
}

/** Info / Operation-Info 图标：lucide Info（spec §8 禁止业务自引 lucide Info） */
export function AlertInfoIcon(props: IconProps) {
  return (
    <IconShell>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
        <circle cx="12" cy="12" r="10" />
        <path d="M12 16v-4" />
        <path d="M12 8h.01" />
      </svg>
    </IconShell>
  );
}

export function AlertOperationInfoIcon(props: IconProps) {
  return <AlertInfoIcon {...props} />;
}

/** Warning 图标：lucide CircleAlert（spec §8 警告必须用 CircleAlert，禁 AlertTriangle） */
export function AlertWarningIcon(props: IconProps) {
  return (
    <IconShell>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
    </IconShell>
  );
}

/** Success 图标：lucide CheckCircle2 */
export function AlertSuccessIcon(props: IconProps) {
  return (
    <IconShell>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
        <path d="M21.801 10A10 10 0 1 1 17 3.335" />
        <path d="m9 11 3 3L22 4" />
      </svg>
    </IconShell>
  );
}

/** Error 图标：lucide XCircle */
export function AlertErrorIcon(props: IconProps) {
  return (
    <IconShell>
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" {...props}>
        <circle cx="12" cy="12" r="10" />
        <path d="m15 9-6 6" />
        <path d="m9 9 6 6" />
      </svg>
    </IconShell>
  );
}

/** Product-News 图标：固定 sparkle（与 demo 仓 alert.tsx 内联 SVG 完全一致） */
export function AlertProductNewsIcon(props: IconProps) {
  return (
    <IconShell>
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" {...props}>
        <path
          d="M12.4375 7.83187L9.31996 6.6825L8.16809 3.5625C8.08 3.32361 7.9208 3.11747 7.71193 2.97187C7.50306 2.82627 7.25457 2.74821 6.99996 2.74821C6.74535 2.74821 6.49686 2.82627 6.28799 2.97187C6.07913 3.11747 5.91992 3.32361 5.83184 3.5625L4.68246 6.6825L1.56246 7.83187C1.32357 7.91996 1.11743 8.07916 0.971833 8.28803C0.826231 8.4969 0.748169 8.74539 0.748169 9C0.748169 9.25461 0.826231 9.5031 0.971833 9.71197C1.11743 9.92084 1.32357 10.08 1.56246 10.1681L4.67996 11.3175L5.83184 14.4375C5.91992 14.6764 6.07913 14.8825 6.28799 15.0281C6.49686 15.1737 6.74535 15.2518 6.99996 15.2518C7.25457 15.2518 7.50306 15.1737 7.71193 15.0281C7.9208 14.8825 8.08 14.6764 8.16809 14.4375L9.31746 11.32L12.4375 10.1681C12.6763 10.08 12.8825 9.92084 13.0281 9.71197C13.1737 9.5031 13.2518 9.25461 13.2518 9C13.2518 8.74539 13.1737 8.4969 13.0281 8.28803C12.8825 8.07916 12.6763 7.91996 12.4375 7.83187ZM8.47621 10.0294C8.37441 10.0669 8.28196 10.1261 8.20524 10.2028C8.12852 10.2795 8.06936 10.3719 8.03184 10.4738L6.99996 13.2675L5.97059 10.4738C5.93307 10.3719 5.87391 10.2795 5.79719 10.2028C5.72047 10.1261 5.62802 10.0669 5.52621 10.0294L2.73246 9L5.52621 7.97062C5.62802 7.93311 5.72047 7.87395 5.79719 7.79723C5.87391 7.72051 5.93307 7.62805 5.97059 7.52625L6.99996 4.7325L8.02934 7.52625C8.06686 7.62805 8.12602 7.72051 8.20274 7.79723C8.27946 7.87395 8.37191 7.93311 8.47371 7.97062L11.2675 9L8.47621 10.0294ZM8.74996 2.5C8.74996 2.30109 8.82898 2.11032 8.96963 1.96967C9.11028 1.82902 9.30105 1.75 9.49996 1.75H10.25V1C10.25 0.801088 10.329 0.610322 10.4696 0.46967C10.6103 0.329018 10.801 0.25 11 0.25C11.1989 0.25 11.3896 0.329018 11.5303 0.46967C11.6709 0.610322 11.75 0.801088 11.75 1V1.75H12.5C12.6989 1.75 12.8896 1.82902 13.0303 1.96967C13.1709 2.11032 13.25 2.30109 13.25 2.5C13.25 2.69891 13.1709 2.88968 13.0303 3.03033C12.8896 3.17098 12.6989 3.25 12.5 3.25H11.75V4C11.75 4.19891 11.6709 4.38968 11.5303 4.53033C11.3896 4.67098 11.1989 4.75 11 4.75C10.801 4.75 10.6103 4.67098 10.4696 4.53033C10.329 4.38968 10.25 4.19891 10.25 4V3.25H9.49996C9.30105 3.25 9.11028 3.17098 8.96963 3.03033C8.82898 2.88968 8.74996 2.69891 8.74996 2.5ZM15.75 5.5C15.75 5.69891 15.6709 5.88968 15.5303 6.03033C15.3896 6.17098 15.1989 6.25 15 6.25H14.75V6.5C14.75 6.69891 14.6709 6.88968 14.5303 7.03033C14.3896 7.17098 14.1989 7.25 14 7.25C13.801 7.25 13.6103 7.17098 13.4696 7.03033C13.329 6.88968 13.25 6.69891 13.25 6.5V6.25H13C12.801 6.25 12.6103 6.17098 12.4696 6.03033C12.329 5.88968 12.25 5.69891 12.25 5.5C12.25 5.30109 12.329 5.11032 12.4696 4.96967C12.6103 4.82902 12.801 4.75 13 4.75H13.25V4.5C13.25 4.30109 13.329 4.11032 13.4696 3.96967C13.6103 3.82902 13.801 3.75 14 3.75C14.1989 3.75 14.3896 3.82902 14.5303 3.96967C14.6709 4.11032 14.75 4.30109 14.75 4.5V4.75H15C15.1989 4.75 15.3896 4.82902 15.5303 4.96967C15.6709 5.11032 15.75 5.30109 15.75 5.5Z"
          fill="currentColor"
        />
      </svg>
    </IconShell>
  );
}

/* ── AdminNoticeAlert 管控端顶部公告条（spec §6） ─────────────────────── */
export type AdminNoticeType = "product-news" | "pending-config" | "resource-alert";

const NOTICE_LABEL: Record<AdminNoticeType, string> = {
  "product-news": "产品动态",
  "pending-config": "待配置",
  "resource-alert": "资源告警",
};

function NoticeTagIcon({ type }: { type: AdminNoticeType }) {
  if (type === "product-news") {
    return (
      <span className="cp-notice__tag-icon" aria-hidden="true">
        <svg viewBox="0 0 11 10" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M6.0625 3.67969L6.14062 3.85938L6.32031 3.9375L8.75098 5L6.32031 6.0625L6.14062 6.14062L6.0625 6.32031L5 8.75098L3.9375 6.32031L3.85938 6.14062L3.67969 6.0625L1.24805 5L3.67969 3.9375L3.85938 3.85938L3.9375 3.67969L5 1.24805L6.0625 3.67969ZM8.30566 7.78711L8.37988 7.93652L8.53027 8.01172L8.76758 8.12988L8.53027 8.24805L8.37988 8.32324L8.30566 8.47266L8.18652 8.71094L8.06836 8.47266L7.99414 8.32324L7.84375 8.24805L7.60547 8.12988L7.84375 8.01172L7.99414 7.93652L8.06836 7.78711L8.18652 7.54785L8.30566 7.78711Z" stroke="currentColor" />
        </svg>
      </span>
    );
  }
  return (
    <span className="cp-notice__tag-icon" aria-hidden="true">
      <svg viewBox="0 0 11 11" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="5.30357" cy="5.30357" r="4.80357" stroke="currentColor" />
        <path d="M4.71484 2.35718H5.89342V6.28575H4.71484V2.35718Z" fill="currentColor" />
        <path d="M4.71484 7.07146H5.89342V8.25003H4.71484V7.07146Z" fill="currentColor" />
      </svg>
    </span>
  );
}

export interface AdminNoticeAlertProps extends React.HTMLAttributes<HTMLDivElement> {
  type: AdminNoticeType;
  /** 右侧控制区插槽（spec §6：组件本身不内置关闭按钮） */
  controls?: React.ReactNode;
}

export function AdminNoticeAlert({
  type,
  controls,
  className,
  children,
  ...props
}: AdminNoticeAlertProps) {
  return (
    <div role="alert" className={cx("cp-notice", `cp-notice--${type}`, className)} {...props}>
      <span className="cp-notice__tag">
        <NoticeTagIcon type={type} />
        <span>{NOTICE_LABEL[type]}</span>
      </span>
      <div className="cp-notice__content">{children}</div>
      {controls != null && <div className="cp-notice__controls">{controls}</div>}
    </div>
  );
}
