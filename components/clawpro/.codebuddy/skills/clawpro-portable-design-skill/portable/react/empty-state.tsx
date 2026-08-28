/**
 * Portable Empty State — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Empty 时的可移植兜底实现。
 *  - 不依赖 shadcn / Radix / Tailwind；样式由 portable/css/empty-state.css 提供。
 *  - 视觉规范（spec/component-specs/empty-state.md §3）：
 *      容器：min-h-[320px] + 蓝灰虚线描边 + 4px 圆角 + 白底 + p-12
 *      插画：100×100，居中（可选；表格内空态禁止用插画，详见 §5）
 *      标题：text-lg / Medium / var(--cp-text-title)
 *      描述：text-xs / var(--cp-text-weak)
 *      Action 槽：gap-6（可选）
 *  - 表格内空态请用 PortableTableEmpty（colSpan 内纯文字）。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/empty-state.css";
 *
 * 用法：
 *   <PortableEmpty title="暂无数据" description="尝试调整筛选条件" />
 *   <PortableEmpty
 *     title="暂无策略"
 *     description="点击下方按钮新建一条策略"
 *     action={<PortableButton variant="claw-primary">新建策略</PortableButton>}
 *     icon={<MyIllustration />}
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export interface PortableEmptyProps
  extends React.HTMLAttributes<HTMLDivElement> {
  /** 标题 */
  title?: React.ReactNode;
  /** 描述 */
  description?: React.ReactNode;
  /** 自定义图标 / 插画（默认无图，纯文字） */
  icon?: React.ReactNode;
  /** 操作按钮槽 */
  action?: React.ReactNode;
  /** 容器是否带蓝灰虚线描边（默认 true，表格内空态请传 false） */
  bordered?: boolean;
}

export const PortableEmpty = React.forwardRef<HTMLDivElement, PortableEmptyProps>(
  (
    {
      title = "暂无数据",
      description,
      icon,
      action,
      bordered = true,
      className = "",
      ...props
    },
    ref
  ) => {
    const merged = ["cp-empty", bordered && "cp-empty--bordered", className]
      .filter(Boolean)
      .join(" ");
    return (
      <div ref={ref} className={merged} {...props}>
        {icon ? <div className="cp-empty__media">{icon}</div> : null}
        <div className="cp-empty__body">
          <div className="cp-empty__title">{title}</div>
          {description ? (
            <div className="cp-empty__desc">{description}</div>
          ) : null}
        </div>
        {action ? <div className="cp-empty__action">{action}</div> : null}
      </div>
    );
  }
);
PortableEmpty.displayName = "PortableEmpty";

/* ───────────── PortableTableEmpty ─────────────
 * 表格内空态：colSpan 内纯文字双行。
 * 必须放在 <tr><td colSpan={N}><PortableTableEmpty /></td></tr> 中。
 */

export interface PortableTableEmptyProps {
  title?: React.ReactNode;
  description?: React.ReactNode;
}

export function PortableTableEmpty({
  title = "暂无数据",
  description = "尝试调整筛选条件，或新建一条记录",
}: PortableTableEmptyProps) {
  return (
    <div className="cp-table-empty">
      <p className="cp-table-empty__text">{title}</p>
      {description ? (
        <p className="cp-table-empty__text">{description}</p>
      ) : null}
    </div>
  );
}
