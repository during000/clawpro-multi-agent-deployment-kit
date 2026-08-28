/**
 * Portable Form Controls（granular 基元）— ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：与 input-select.tsx 的复合 PortableField 互补，提供可单独组合的
 *       字段基元，便于宿主仓手动拼装 Field > Label > Control > Helper/Error。
 *  - 不依赖 shadcn / Tailwind；样式由 portable/css/form-controls.css 提供。
 *  - 视觉规范（component-specs/form-controls.md §4 / §6）：
 *      Label 12px medium text-secondary；Helper 14px text-muted；Error 14px text-danger。
 *      筛选条用统一 gap-3（PortableFieldRow），控件本身不写 margin。
 *  - 圆角端别分流：Admin 4px / Tenant 搜索胶囊 / Tenant 表单 4px（见 input-select）。
 *  - 搜索框保留左侧 icon；弹窗内不要重新发明输入框；同行控件 h-9 等高。
 *
 *  说明：完整的「Label + Input + Helper」一体化组件请用 input-select.tsx 的
 *        PortableField；本文件提供更细的基元供自由组合。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/form-controls.css";
 *
 * 用法：
 *   <PortableFieldGroup>
 *     <PortableLabel htmlFor="name" required>名称</PortableLabel>
 *     <PortableInput id="name" placeholder="名称" />
 *     <PortableHelperText>用于在列表中识别该实例</PortableHelperText>
 *   </PortableFieldGroup>
 *
 *   <PortableFieldRow>
 *     <PortableInput placeholder="搜索" />
 *     <PortableSelect ... />
 *   </PortableFieldRow>
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

/* ───────────── PortableLabel ───────────── */

export interface PortableLabelProps
  extends React.LabelHTMLAttributes<HTMLLabelElement> {
  /** 末尾追加红色 * 必填标记 */
  required?: boolean;
}

export function PortableLabel({
  required,
  className = "",
  children,
  ...props
}: PortableLabelProps) {
  const merged = ["cp-label", className].filter(Boolean).join(" ");
  return (
    <label className={merged} {...props}>
      {children}
      {required && (
        <span className="cp-label__required" aria-hidden="true">
          *
        </span>
      )}
    </label>
  );
}

/* ───────────── PortableFieldGroup（label + control + helper 竖排） ───────────── */

export type PortableFieldGroupProps = React.HTMLAttributes<HTMLDivElement>;

export function PortableFieldGroup({ className = "", ...props }: PortableFieldGroupProps) {
  const merged = ["cp-field-group", className].filter(Boolean).join(" ");
  return <div data-slot="field-group" className={merged} {...props} />;
}

/* ───────────── PortableFieldRow（筛选条容器，统一 gap-3） ───────────── */

export type PortableFieldRowProps = React.HTMLAttributes<HTMLDivElement>;

export function PortableFieldRow({ className = "", ...props }: PortableFieldRowProps) {
  const merged = ["cp-field-row", className].filter(Boolean).join(" ");
  return <div data-slot="field-row" className={merged} {...props} />;
}

/* ───────────── PortableHelperText ───────────── */

export type PortableHelperTextProps = React.HTMLAttributes<HTMLParagraphElement>;

export function PortableHelperText({ className = "", ...props }: PortableHelperTextProps) {
  const merged = ["cp-helper-text", className].filter(Boolean).join(" ");
  return <p className={merged} {...props} />;
}

/* ───────────── PortableFieldError ───────────── */

export type PortableFieldErrorProps = React.HTMLAttributes<HTMLParagraphElement>;

export function PortableFieldError({
  className = "",
  children,
  ...props
}: PortableFieldErrorProps) {
  if (children == null || children === false) return null;
  const merged = ["cp-field-error", className].filter(Boolean).join(" ");
  return (
    <p role="alert" className={merged} {...props}>
      {children}
    </p>
  );
}
