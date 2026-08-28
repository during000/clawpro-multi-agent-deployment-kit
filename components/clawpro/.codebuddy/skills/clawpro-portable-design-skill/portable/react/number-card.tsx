/**
 * Portable NumberCard — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 NumberCard 时的可移植兜底实现。
 *  - 不依赖 demo 仓 SurfaceCard / Typography 语义层。
 *  - 视觉规范（spec/component-specs/number-card.md §3）：
 *      容器：rounded-[4px] + 蓝灰描边 + p-5（20px）+ 默认无阴影
 *      Header：mb-3 + flex items-center gap-2
 *      Icon：18×18，shrink-0；可传任意 SVG / `<img>` / lucide
 *      Label：text-sm font-medium var(--cp-text-title)
 *      Value：24px / semibold / tabular-nums / DIN 字体
 *      Extra 槽：与 value 同行，gap-8（32px）
 *      Footer 槽：mt-2（8px）
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/number-card.css";
 *
 * 用法：
 *   // 使用项目内真实 SVG icon（推荐）
 *   <PortableNumberCard
 *     iconSrc="/assets/icons/requests-total.svg"
 *     label="总请求数"
 *     value="1,841"
 *   />
 *
 *   // 自定义 React icon
 *   <PortableNumberCard
 *     icon={<MyIcon />}
 *     label="今日全局配额消耗"
 *     value="68%"
 *     extra={<ProgressBar value={68} />}
 *     footer={<span>较昨日 +12%</span>}
 *   />
 *
 *   // 使用内置渐变图标（向后兼容）
 *   <PortableNumberCard
 *     icon={<PortableTotalTokensIcon />}
 *     label="总 Tokens"
 *     value="223,158"
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

/* ───────────── StatNumber（内联） ─────────────
 * tabular-nums + DIN 字体；调用方只传字符串/ReactNode。
 */
export const PortableStatNumber = React.forwardRef<
  HTMLSpanElement,
  React.HTMLAttributes<HTMLSpanElement>
>(({ className = "", style, ...props }, ref) => {
  const merged = ["cp-stat-number", className].filter(Boolean).join(" ");
  return <span ref={ref} className={merged} style={style} {...props} />;
});
PortableStatNumber.displayName = "PortableStatNumber";

/* ───────────── NumberCard ───────────── */

export interface PortableNumberCardProps
  extends Omit<React.HTMLAttributes<HTMLDivElement>, "title"> {
  /** 自定义 React icon（与 iconSrc 二选一） */
  icon?: React.ReactNode;
  /** 项目内 SVG 图标路径（与 icon 二选一）。例如 "/assets/icons/requests-total.svg" */
  iconSrc?: string;
  /** Icon 尺寸（默认 18） */
  iconSize?: number;
  label: React.ReactNode;
  value: React.ReactNode;
  extra?: React.ReactNode;
  footer?: React.ReactNode;
}

export const PortableNumberCard = React.forwardRef<
  HTMLDivElement,
  PortableNumberCardProps
>(
  (
    {
      icon,
      iconSrc,
      iconSize = 18,
      label,
      value,
      extra,
      footer,
      className = "",
      ...props
    },
    ref
  ) => {
    const merged = ["cp-number-card", className].filter(Boolean).join(" ");

    const iconNode = iconSrc ? (
      <img
        src={iconSrc}
        alt=""
        aria-hidden="true"
        width={iconSize}
        height={iconSize}
        style={{ width: iconSize, height: iconSize }}
        className="cp-number-card__icon"
      />
    ) : icon ? (
      <span
        className="cp-number-card__icon"
        style={{ width: iconSize, height: iconSize }}
      >
        {icon}
      </span>
    ) : null;

    return (
      <div
        ref={ref}
        data-component="portable-number-card"
        className={merged}
        {...props}
      >
        <div className="cp-number-card__header">
          {iconNode}
          <span className="cp-number-card__label">{label}</span>
        </div>
        {extra ? (
          <div className="cp-number-card__value-row">
            <PortableStatNumber className="cp-stat-number--shrink">
              {value}
            </PortableStatNumber>
            {extra}
          </div>
        ) : (
          <PortableStatNumber>{value}</PortableStatNumber>
        )}
        {footer ? <div className="cp-number-card__footer">{footer}</div> : null}
      </div>
    );
  }
);
PortableNumberCard.displayName = "PortableNumberCard";

/* ─────────────────────────────────────────────────────────────
 * 项目内真实 SVG 图标路径常量（推荐使用）
 *
 * 这些路径指向 portable/assets/icons/ 下的真实 SVG，
 * 由 /Users/addietang/Documents/cvm/openclaw-enterprise/icon/ 复制而来。
 *
 * 用法：
 *   <PortableNumberCard iconSrc={ICON_REQUESTS_TOTAL} ... />
 *
 * 注意：外部仓使用时，请把 portable/assets/icons/ 目录复制到自己的
 * public 静态资源目录，并按需调整路径前缀。
 * ───────────────────────────────────────────────────────────── */

export const PORTABLE_ICONS = {
  REQUESTS_TOTAL: "./assets/icons/requests-total.svg",
  TOKENS_INPUT: "./assets/icons/tokens-input.svg",
  TOKENS_OUTPUT: "./assets/icons/tokens-output.svg",
  TOKENS_TOTAL: "./assets/icons/tokens-total.svg",
  QUOTA_TODAY: "./assets/icons/quota-today.svg",
  TOTAL: "./assets/icons/total.svg",
  AGENT_ASSETS: "./assets/icons/agent-assets.svg",
  THREAT_ALERT: "./assets/icons/threat-alert.svg",
  RISK_EXIST: "./assets/icons/risk-exist.svg",
  RUNNING: "./assets/icons/running.svg",
  SHUTDOWN: "./assets/icons/shutdown.svg",
  OTHER: "./assets/icons/other.svg",
} as const;

/* ─────────────────────────────────────────────────────────────
 * GradientIcon · 向后兼容
 * 把任意 SVG path 包装成 NumberCard 默认渐变风格的 18×18 图标。
 * 用 React.useId 隔离 gradient id，避免多卡共享 id 渲染塌陷。
 * ⚠️ 推荐优先使用 iconSrc 引项目真实 SVG，仅在没有图源时降级用本套。
 * ───────────────────────────────────────────────────────────── */

export interface PortableGradientIconProps
  extends Omit<React.SVGAttributes<SVGSVGElement>, "fill"> {
  size?: number;
  /** 渐变起始色（左上） */
  from?: string;
  /** 渐变结束色（右下） */
  to?: string;
}

export const PortableGradientIcon = React.forwardRef<
  SVGSVGElement,
  PortableGradientIconProps
>(
  (
    {
      size = 18,
      from = "#202020",
      to = "#0080FF",
      children,
      className,
      viewBox = "0 0 18 18",
      ...props
    },
    ref
  ) => {
    const reactId = React.useId().replace(/:/g, "");
    const gradId = `portable-numbercard-grad-${reactId}`;
    return (
      <svg
        ref={ref}
        width={size}
        height={size}
        viewBox={viewBox}
        fill={`url(#${gradId})`}
        xmlns="http://www.w3.org/2000/svg"
        className={className}
        {...props}
      >
        <defs>
          <radialGradient
            id={gradId}
            cx="0"
            cy="0"
            r="1"
            gradientUnits="userSpaceOnUse"
            gradientTransform="translate(2.81 9) scale(13.5 720)"
          >
            <stop stopColor={from} />
            <stop offset="1" stopColor={to} />
          </radialGradient>
        </defs>
        {children}
      </svg>
    );
  }
);
PortableGradientIcon.displayName = "PortableGradientIcon";

/* ───────────── 内置 4 枚渐变图标（向后兼容） ───────────── */

export const PortableRequestsIcon = React.forwardRef<
  SVGSVGElement,
  Omit<PortableGradientIconProps, "children">
>((props, ref) => (
  <PortableGradientIcon ref={ref} aria-label="请求数" {...props}>
    <path d="M11.1557 0.568474C11.2759 0.547602 11.3997 0.565694 11.5083 0.621208C11.6168 0.676751 11.7039 0.766463 11.7573 0.876091C11.8107 0.985788 11.8275 1.10986 11.8042 1.22961L10.77 6.39172L14.8227 7.91125C14.9089 7.94398 14.9857 7.99716 15.0464 8.06652C15.1071 8.13609 15.1505 8.2197 15.1714 8.30968C15.1922 8.39969 15.1905 8.4939 15.1665 8.58312C15.1425 8.67222 15.0968 8.75406 15.0337 8.8214H15.0366L7.1616 17.2589L7.09421 17.3204C7.0224 17.3757 6.9373 17.4131 6.84714 17.4288L6.7573 17.4366C6.69672 17.4373 6.63627 17.4288 6.57859 17.4103L6.49461 17.3751C6.386 17.3195 6.29798 17.2299 6.24461 17.1202C6.20472 17.0381 6.18625 16.9479 6.18894 16.8575L6.19871 16.7667L7.22996 11.6105L3.17722 10.089C3.11208 10.0646 3.05213 10.0285 3.00046 9.98254L2.95164 9.93273C2.9057 9.8803 2.86992 9.82011 2.84617 9.755L2.82664 9.68859C2.80577 9.59809 2.80709 9.50378 2.83152 9.41418C2.85597 9.32456 2.90234 9.2423 2.96629 9.17492L10.8413 0.737419C10.9247 0.648358 11.0355 0.589437 11.1557 0.568474ZM5.34324 9.09972L9.1655 10.5353L8.63035 13.2111L11.1528 10.5089H11.1401L12.6479 8.89758L8.83445 7.46789L9.37058 4.78527L5.34324 9.09972Z" />
  </PortableGradientIcon>
));
PortableRequestsIcon.displayName = "PortableRequestsIcon";

export const PortableInputTokensIcon = React.forwardRef<
  SVGSVGElement,
  Omit<PortableGradientIconProps, "children">
>((props, ref) => (
  <PortableGradientIcon ref={ref} aria-label="输入 Tokens" {...props}>
    <path d="M5.02805 6.22195C4.86954 6.06344 4.78049 5.84846 4.78049 5.6243C4.78049 5.40013 4.86954 5.18515 5.02805 5.02664C5.18656 4.86813 5.40154 4.77908 5.6257 4.77908C5.84987 4.77908 6.06485 4.86813 6.22336 5.02664L8.15625 6.96094V1.6875C8.15625 1.46372 8.24514 1.24911 8.40338 1.09088C8.56161 0.932645 8.77622 0.84375 9 0.84375C9.22378 0.84375 9.43839 0.932645 9.59662 1.09088C9.75485 1.24911 9.84375 1.46372 9.84375 1.6875V6.96094L11.778 5.02594C11.9366 4.86743 12.1515 4.77838 12.3757 4.77838C12.5999 4.77838 12.8149 4.86743 12.9734 5.02594C13.1319 5.18445 13.2209 5.39943 13.2209 5.62359C13.2209 5.84776 13.1319 6.06274 12.9734 6.22125L9.59836 9.59625C9.51997 9.67491 9.42683 9.73732 9.32427 9.77991C9.22171 9.82249 9.11175 9.84442 9.0007 9.84442C8.88965 9.84442 8.7797 9.82249 8.67714 9.77991C8.57458 9.73732 8.48143 9.67491 8.40305 9.59625L5.02805 6.22195ZM15.75 8.15625H13.2188C12.995 8.15625 12.7804 8.24514 12.6221 8.40338C12.4639 8.56161 12.375 8.77622 12.375 9C12.375 9.22378 12.4639 9.43839 12.6221 9.59662C12.7804 9.75485 12.995 9.84375 13.2188 9.84375H15.4688V13.7812H2.53125V9.84375H4.78125C5.00503 9.84375 5.21964 9.75485 5.37787 9.59662C5.53611 9.43839 5.625 9.22378 5.625 9C5.625 8.77622 5.53611 8.56161 5.37787 8.40338C5.21964 8.24514 5.00503 8.15625 4.78125 8.15625H2.25C1.87704 8.15625 1.51935 8.30441 1.25563 8.56813C0.991908 8.83185 0.84375 9.18954 0.84375 9.5625V14.0625C0.84375 14.4355 0.991908 14.7931 1.25563 15.0569C1.51935 15.3206 1.87704 15.4688 2.25 15.4688H15.75C16.123 15.4688 16.4806 15.3206 16.7444 15.0569C17.0081 14.7931 17.1562 14.4355 17.1562 14.0625V9.5625C17.1563 9.18954 17.0081 8.83185 16.7444 8.56813C16.4806 8.30441 16.123 8.15625 15.75 8.15625ZM14.3438 11.8125C14.3438 11.59 14.2778 11.3725 14.1542 11.1875C14.0305 11.0025 13.8548 10.8583 13.6493 10.7731C13.4437 10.688 13.2175 10.6657 12.9993 10.7091C12.781 10.7525 12.5806 10.8597 12.4233 11.017C12.2659 11.1743 12.1588 11.3748 12.1154 11.593C12.072 11.8113 12.0942 12.0375 12.1794 12.243C12.2645 12.4486 12.4087 12.6243 12.5937 12.7479C12.7787 12.8715 12.9962 12.9375 13.2188 12.9375C13.5171 12.9375 13.8033 12.819 14.0142 12.608C14.2252 12.397 14.3438 12.1109 14.3438 11.8125Z" />
  </PortableGradientIcon>
));
PortableInputTokensIcon.displayName = "PortableInputTokensIcon";

export const PortableOutputTokensIcon = React.forwardRef<
  SVGSVGElement,
  Omit<PortableGradientIconProps, "children">
>((props, ref) => (
  <PortableGradientIcon ref={ref} aria-label="输出 Tokens" {...props}>
    <path d="M13.8157 10.653C13.9742 10.8116 14.0633 11.0265 14.0633 11.2507C14.0633 11.4749 13.9742 11.6899 13.8157 11.8484C13.6572 12.0069 13.4422 12.0959 13.2181 12.0959C12.9939 12.0959 12.7789 12.0069 12.6204 11.8484L11.8125 11.0391V14.625C11.8125 14.8488 11.7236 15.0634 11.5654 15.2216C11.4071 15.3799 11.1925 15.4688 10.9688 15.4688C10.745 15.4688 10.5304 15.3799 10.3721 15.2216C10.2139 15.0634 10.125 14.8488 10.125 14.625V11.0391L9.31572 11.8491C9.15721 12.0076 8.94222 12.0966 8.71806 12.0966C8.4939 12.0966 8.27891 12.0076 8.1204 11.8491C7.9619 11.6906 7.87285 11.4756 7.87285 11.2514C7.87285 11.0272 7.9619 10.8123 8.1204 10.6538L10.3704 8.40375C10.4488 8.32509 10.5419 8.26268 10.6445 8.22009C10.7471 8.17751 10.857 8.15558 10.9681 8.15558C11.0791 8.15558 11.1891 8.17751 11.2916 8.22009C11.3942 8.26268 11.4873 8.32509 11.5657 8.40375L13.8157 10.653ZM11.25 2.53125C10.0822 2.53181 8.93632 2.84821 7.9337 3.44694C6.93107 4.04567 6.10905 4.90443 5.5547 5.93227C4.9091 5.86465 4.2565 5.9292 3.63666 6.12198C3.01682 6.31477 2.44272 6.63175 1.94937 7.05361C1.45601 7.47547 1.05372 7.99337 0.767018 8.57575C0.480315 9.15814 0.315204 9.7928 0.281746 10.4411C0.248288 11.0893 0.347185 11.7376 0.57241 12.3464C0.797634 12.9552 1.14447 13.5118 1.59178 13.9822C2.03908 14.4526 2.57749 14.827 3.17419 15.0826C3.77089 15.3382 4.41338 15.4695 5.06251 15.4688H7.03126C7.25504 15.4688 7.46965 15.3799 7.62788 15.2216C7.78612 15.0634 7.87501 14.8488 7.87501 14.625C7.87501 14.4012 7.78612 14.1866 7.62788 14.0284C7.46965 13.8701 7.25504 13.7812 7.03126 13.7812H5.06251C4.25632 13.7763 3.4839 13.4569 2.90972 12.8909C2.33554 12.325 2.00496 11.5573 1.98838 10.7512C1.97179 9.94518 2.2705 9.1645 2.82091 8.57542C3.37132 7.98633 4.12994 7.63537 4.93525 7.59727C4.83275 8.0578 4.78111 8.5282 4.78126 9C4.78126 9.22378 4.87016 9.43839 5.02839 9.59662C5.18663 9.75485 5.40124 9.84375 5.62501 9.84375C5.84879 9.84375 6.0634 9.75485 6.22163 9.59662C6.37987 9.43839 6.46876 9.22378 6.46876 9C6.46934 8.30834 6.61998 7.62505 6.91028 6.99726C7.20057 6.36948 7.62362 5.81215 8.15022 5.36373C8.67682 4.91532 9.29445 4.58649 9.96047 4.39995C10.6265 4.2134 11.3251 4.17358 12.008 4.28322C12.6909 4.39287 13.3419 4.64938 13.916 5.03504C14.4902 5.42071 14.9738 5.92635 15.3336 6.51708C15.6933 7.10781 15.9206 7.76956 15.9998 8.45666C16.079 9.14377 16.0082 9.83988 15.7922 10.497C15.7575 10.6022 15.7439 10.7133 15.7522 10.8238C15.7604 10.9343 15.7904 11.0421 15.8403 11.1411C15.8902 11.24 15.9591 11.3282 16.0431 11.4005C16.1271 11.4728 16.2245 11.5279 16.3297 11.5625C16.435 11.5972 16.5461 11.6108 16.6566 11.6026C16.7671 11.5943 16.8749 11.5644 16.9739 11.5145C17.0728 11.4645 17.161 11.3956 17.2333 11.3116C17.3056 11.2277 17.3607 11.1303 17.3953 11.025C17.7147 10.0532 17.7992 9.01945 17.6418 8.00864C17.4845 6.99784 17.0898 6.03872 16.4902 5.20992C15.8905 4.38112 15.103 3.70624 14.1921 3.24063C13.2812 2.77501 12.273 2.5319 11.25 2.53125Z" />
  </PortableGradientIcon>
));
PortableOutputTokensIcon.displayName = "PortableOutputTokensIcon";

export const PortableTotalTokensIcon = React.forwardRef<
  SVGSVGElement,
  Omit<PortableGradientIconProps, "children">
>((props, ref) => (
  <PortableGradientIcon ref={ref} aria-label="总 Tokens" {...props}>
    <path d="M16.6322 7.68155L12.2953 6.10444L10.7182 1.76757C10.6198 1.49691 10.4405 1.26309 10.2047 1.09786C9.9688 0.932629 9.68779 0.843994 9.39982 0.843994C9.11184 0.843994 8.83084 0.932629 8.59498 1.09786C8.35912 1.26309 8.17983 1.49691 8.08146 1.76757L6.50435 6.10444L2.16747 7.68155C1.89682 7.77992 1.66299 7.95921 1.49776 8.19507C1.33253 8.43093 1.2439 8.71193 1.2439 8.99991C1.2439 9.28789 1.33253 9.56889 1.49776 9.80475C1.66299 10.0406 1.89682 10.2199 2.16747 10.3183L6.50435 11.8954L8.08146 16.2323C8.17983 16.5029 8.35912 16.7367 8.59498 16.902C8.83084 17.0672 9.11184 17.1558 9.39982 17.1558C9.68779 17.1558 9.9688 17.0672 10.2047 16.902C10.4405 16.7367 10.6198 16.5029 10.7182 16.2323L12.2953 11.8954L16.6322 10.3183C16.9028 10.2199 17.1366 10.0406 17.3019 9.80475C17.4671 9.56889 17.5557 9.28789 17.5557 8.99991C17.5557 8.71193 17.4671 8.43093 17.3019 8.19507C17.1366 7.95921 16.9028 7.77992 16.6322 7.68155ZM11.3489 10.4441C11.2329 10.4863 11.1277 10.5533 11.0404 10.6405C10.9532 10.7278 10.8862 10.833 10.844 10.949L9.39982 14.9209L7.9556 10.949C7.91347 10.833 7.84643 10.7278 7.7592 10.6405C7.67198 10.5533 7.56669 10.4863 7.45075 10.4441L3.4788 8.99991L7.45075 7.55569C7.56669 7.51356 7.67198 7.44653 7.7592 7.3593C7.84643 7.27208 7.91347 7.16679 7.9556 7.05085L9.39982 3.0789L10.844 7.05085C10.8862 7.16679 10.9532 7.27208 11.0404 7.3593C11.1277 7.44653 11.2329 7.51356 11.3489 7.55569L15.3208 8.99991L11.3489 10.4441Z" />
  </PortableGradientIcon>
));
PortableTotalTokensIcon.displayName = "PortableTotalTokensIcon";
