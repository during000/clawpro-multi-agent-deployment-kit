/**
 * Portable Breadcrumb — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Breadcrumb 时的可移植兜底实现。
 *  - 不依赖 @radix-ui / shadcn / Tailwind；样式由 portable/css/breadcrumb.css 提供。
 *  - 视觉规范（component-specs/breadcrumb.md §3）：
 *      字号 14px；分隔符左右 gap 6px；仅文字 + 间距，无背景 / 边框 / 卡片。
 *      当前页：<span> + font-medium + text-title + aria-current="page"（不可点击）。
 *      祖先页：<a> + text-muted，hover 加深到 text-title（一定可点击）。
 *      分隔符：text-weak 弱灰（不要纯黑 / 品牌蓝）。
 *  - 单级页面不显示面包屑。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/breadcrumb.css";
 *
 * 用法：
 *   <PortableBreadcrumb
 *     items={[
 *       { label: "实例管理", href: "/admin/agents" },
 *       { label: "Agent 详情" },   // 末项无 href → 当前页
 *     ]}
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export interface PortableBreadcrumbItem {
  label: React.ReactNode;
  /** 有 href（或 onClick）= 祖先页可点击；末项不传 = 当前页 */
  href?: string;
  onClick?: (e: React.MouseEvent) => void;
}

export interface PortableBreadcrumbProps
  extends Omit<React.HTMLAttributes<HTMLElement>, "children"> {
  items: PortableBreadcrumbItem[];
  /** 分隔符，默认 "/"，可传 ">" 或自定义节点 */
  separator?: React.ReactNode;
}

export function PortableBreadcrumb({
  items,
  separator = "/",
  className = "",
  ...props
}: PortableBreadcrumbProps) {
  const merged = ["cp-breadcrumb", className].filter(Boolean).join(" ");
  return (
    <nav aria-label="breadcrumb" className={merged} {...props}>
      {items.map((item, i) => {
        const isLast = i === items.length - 1;
        const clickable = !isLast && (item.href != null || item.onClick != null);
        return (
          <React.Fragment key={i}>
            {i > 0 && (
              <span className="cp-breadcrumb__sep" aria-hidden="true">
                {separator}
              </span>
            )}
            {clickable ? (
              <a
                href={item.href}
                onClick={item.onClick}
                className="cp-breadcrumb__link"
              >
                {item.label}
              </a>
            ) : (
              <span
                className="cp-breadcrumb__page"
                aria-current={isLast ? "page" : undefined}
              >
                {item.label}
              </span>
            )}
          </React.Fragment>
        );
      })}
    </nav>
  );
}
