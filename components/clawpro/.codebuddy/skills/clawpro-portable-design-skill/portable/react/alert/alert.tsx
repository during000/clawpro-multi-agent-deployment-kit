/**
 * Alert 组件 - 页面级别的信息提示
 * ─────────────────────────────────────────────────────────────────────────────
 * 统一的信息提示、警告、成功、错误等提示条组件
 *
 * 6 种变体：
 *   - info: 信息提示（浅蓝底）
 *   - operation-info: 操作说明（白底灰边）
 *   - warning: 警告提示（橙色）
 *   - product-news: 产品动态（浅蓝底）
 *   - success: 成功提示（浅绿底）
 *   - error: 错误提示（浅红底）
 *
 * 使用方式：
 *   import { Alert, AlertTitle, AlertDescription, AlertInfoIcon } from "@/components/ui/alert";
 *
 *   <Alert variant="info">
 *     <AlertInfoIcon />
 *     <AlertDescription>信息提示文案</AlertDescription>
 *   </Alert>
 */

import React from "react";

type AlertVariant =
  | "info"
  | "operation-info"
  | "warning"
  | "product-news"
  | "success"
  | "error";

interface AlertProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: AlertVariant;
}

interface AlertIconProps extends React.SVGAttributes<SVGSVGElement> {}

/**
 * Alert 容器组件
 */
const Alert = React.forwardRef<HTMLDivElement, AlertProps>(
  ({ className = "", variant = "info", ...props }, ref) => {
    const variantClass = `alert alert-${variant}`;
    return (
      <div
        ref={ref}
        role="alert"
        className={`${variantClass} ${className}`}
        {...props}
      />
    );
  }
);
Alert.displayName = "Alert";

/**
 * AlertTitle - 用于提示的标题
 */
const AlertTitle = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className = "", ...props }, ref) => (
  <h5 ref={ref as any} className={`alert-title ${className}`} {...props} />
));
AlertTitle.displayName = "AlertTitle";

/**
 * AlertDescription - 提示的描述文案
 */
const AlertDescription = React.forwardRef<
  HTMLParagraphElement,
  React.HTMLAttributes<HTMLParagraphElement>
>(({ className = "", ...props }, ref) => (
  <div
    ref={ref}
    className={`alert-description ${className}`}
    {...props}
  />
));
AlertDescription.displayName = "AlertDescription";

/**
 * Info 图标 - 蓝色信息图标 (16px)
 */
const AlertInfoIcon = React.forwardRef<SVGSVGElement, AlertIconProps>(
  (props, ref) => (
    <svg
      ref={ref}
      className="alert-icon alert-info-icon"
      viewBox="0 0 16 16"
      fill="currentColor"
      aria-hidden="true"
      {...props}
    >
      <circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" strokeWidth="1.5" />
      <circle cx="8" cy="5" r="0.5" fill="currentColor" />
      <path d="M8 7v3" stroke="currentColor" strokeWidth="1.5" />
    </svg>
  )
);
AlertInfoIcon.displayName = "AlertInfoIcon";

/**
 * Operation-Info 图标 - 灰色操作说明图标 (16px)
 */
const AlertOperationInfoIcon = React.forwardRef<SVGSVGElement, AlertIconProps>(
  (props, ref) => (
    <svg
      ref={ref}
      className="alert-icon alert-operation-info-icon"
      viewBox="0 0 16 16"
      fill="currentColor"
      aria-hidden="true"
      {...props}
    >
      <circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" strokeWidth="1.5" />
      <circle cx="8" cy="5" r="0.5" fill="currentColor" />
      <path d="M8 7v3" stroke="currentColor" strokeWidth="1.5" />
    </svg>
  )
);
AlertOperationInfoIcon.displayName = "AlertOperationInfoIcon";

/**
 * Success 图标 - 绿色成功图标 (16px)
 */
const AlertSuccessIcon = React.forwardRef<SVGSVGElement, AlertIconProps>(
  (props, ref) => (
    <svg
      ref={ref}
      className="alert-icon alert-success-icon"
      viewBox="0 0 16 16"
      fill="currentColor"
      aria-hidden="true"
      {...props}
    >
      <circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" strokeWidth="1.5" />
      <path
        d="M5 8l2 2 4-4"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
    </svg>
  )
);
AlertSuccessIcon.displayName = "AlertSuccessIcon";

/**
 * Error 图标 - 红色错误图标 (16px)
 */
const AlertErrorIcon = React.forwardRef<SVGSVGElement, AlertIconProps>(
  (props, ref) => (
    <svg
      ref={ref}
      className="alert-icon alert-error-icon"
      viewBox="0 0 16 16"
      fill="currentColor"
      aria-hidden="true"
      {...props}
    >
      <circle cx="8" cy="8" r="7" fill="none" stroke="currentColor" strokeWidth="1.5" />
      <path
        d="M5 5l6 6M11 5l-6 6"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
);
AlertErrorIcon.displayName = "AlertErrorIcon";

/**
 * Product News 图标 - 蓝色产品动态图标 (16px)
 */
const AlertProductNewsIcon = React.forwardRef<SVGSVGElement, AlertIconProps>(
  (props, ref) => (
    <svg
      ref={ref}
      className="alert-icon alert-product-news-icon"
      viewBox="0 0 16 16"
      fill="currentColor"
      aria-hidden="true"
      {...props}
    >
      {/* Sparkle icon */}
      <path d="M8 1v3M8 11v3M1 8h3M11 8h3M2.5 2.5l2.12 2.12M11.38 11.38l2.12 2.12M13.5 2.5l-2.12 2.12M4.62 11.38l-2.12 2.12"
            stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
);
AlertProductNewsIcon.displayName = "AlertProductNewsIcon";

export {
  Alert,
  AlertTitle,
  AlertDescription,
  AlertInfoIcon,
  AlertOperationInfoIcon,
  AlertSuccessIcon,
  AlertErrorIcon,
  AlertProductNewsIcon,
  type AlertVariant,
  type AlertProps,
};
