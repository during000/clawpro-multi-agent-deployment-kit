/**
 * Portable Selection Controls — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构选择控件时的可移植兜底实现。
 *  - 不依赖 shadcn / Radix / Tailwind；样式由 portable/css/selection-controls.css 提供。
 *  - 涵盖 Switch / Checkbox / Radio 三件套。
 *  - spec/component-specs/selection-controls.md §3 视觉标准全覆盖。
 *  - ⚠ 选择控件默认描边走 --cp-border-control (#C8CFDA)，不是 --cp-border。
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/selection-controls.css";
 *
 * 用法：
 *   <PortableSwitch checked={on} onChange={setOn} />
 *   <PortableCheckbox checked={selected} onChange={setSelected} />
 *   <PortableRadio name="mode" value="a" checked={val==="a"} onChange={()=>setVal("a")} />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

/* ─────────────── PortableSwitch ───────────────
 * 受控开关（toggle）组件。44×24 / full 圆角 / off 灰底，on 品牌蓝底。
 */

export interface PortableSwitchProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "type" | "onChange"> {
  onChange?: (checked: boolean) => void;
}

export const PortableSwitch = React.forwardRef<HTMLInputElement, PortableSwitchProps>(
  ({ checked, onChange, className, disabled, ...props }, ref) => {
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      onChange?.(e.target.checked);
    };

    return (
      <label
        className={["cp-switch", disabled && "cp-switch--disabled", className]
          .filter(Boolean)
          .join(" ")}
      >
        <input
          ref={ref}
          type="checkbox"
          checked={checked}
          onChange={handleChange}
          disabled={disabled}
          className="cp-switch__input"
          {...props}
        />
        <span
          className={["cp-switch__track", checked && "cp-switch__track--on"]
            .filter(Boolean)
            .join(" ")}
        >
          <span
            className={["cp-switch__thumb", checked && "cp-switch__thumb--on"]
              .filter(Boolean)
              .join(" ")}
          />
        </span>
      </label>
    );
  }
);
PortableSwitch.displayName = "PortableSwitch";

/* ─────────────── PortableCheckbox ───────────────
 * 勾选框。20×20 / 4px 圆角 / 未选白底描边，选中品牌蓝底白勾。
 */

export interface PortableCheckboxProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> {
  label?: React.ReactNode;
}

export const PortableCheckbox = React.forwardRef<HTMLInputElement, PortableCheckboxProps>(
  ({ checked, className, disabled, label, ...props }, ref) => {
    return (
      <label
        className={["cp-checkbox", disabled && "cp-checkbox--disabled", className]
          .filter(Boolean)
          .join(" ")}
      >
        <input
          ref={ref}
          type="checkbox"
          checked={checked}
          disabled={disabled}
          className="cp-checkbox__input"
          {...props}
        />
        <span
          className={["cp-checkbox__box", checked && "cp-checkbox__box--checked"]
            .filter(Boolean)
            .join(" ")}
        >
          {checked && (
            <svg
              width="14"
              height="14"
              viewBox="0 0 14 14"
              fill="none"
              stroke="white"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M2 7l4 4 6-8" />
            </svg>
          )}
        </span>
        {label && <span className="cp-checkbox__label">{label}</span>}
      </label>
    );
  }
);
PortableCheckbox.displayName = "PortableCheckbox";

/* ─────────────── PortableRadio ───────────────
 * 单选按钮（分组）。20×20 / 圆形 / 未选白底描边，选中品牌蓝环 + 蓝圆心。
 */

export interface PortableRadioProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> {
  label?: React.ReactNode;
}

export const PortableRadio = React.forwardRef<HTMLInputElement, PortableRadioProps>(
  ({ checked, className, disabled, label, ...props }, ref) => {
    return (
      <label
        className={["cp-radio", disabled && "cp-radio--disabled", className]
          .filter(Boolean)
          .join(" ")}
      >
        <input
          ref={ref}
          type="radio"
          checked={checked}
          disabled={disabled}
          className="cp-radio__input"
          {...props}
        />
        <span
          className={["cp-radio__circle", checked && "cp-radio__circle--checked"]
            .filter(Boolean)
            .join(" ")}
        >
          {checked && <span className="cp-radio__dot" />}
        </span>
        {label && <span className="cp-radio__label">{label}</span>}
      </label>
    );
  }
);
PortableRadio.displayName = "PortableRadio";

/* ─────────────── PortableRadioGroup ───────────────
 * 单选组便利包装。
 */

export interface PortableRadioGroupProps {
  name: string;
  value?: string;
  onChange?: (value: string) => void;
  options: Array<{ value: string; label: React.ReactNode }>;
  disabled?: boolean;
  className?: string;
  layout?: "vertical" | "horizontal";
}

export function PortableRadioGroup({
  name,
  value,
  onChange,
  options,
  disabled,
  className = "",
  layout = "vertical",
}: PortableRadioGroupProps) {
  const merged = [
    "cp-radio-group",
    layout === "horizontal" ? "cp-radio-group--horizontal" : "cp-radio-group--vertical",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={merged}>
      {options.map((opt) => (
        <PortableRadio
          key={opt.value}
          name={name}
          value={opt.value}
          checked={value === opt.value}
          onChange={(e) => onChange?.(e.currentTarget.value)}
          disabled={disabled}
          label={opt.label}
        />
      ))}
    </div>
  );
}
