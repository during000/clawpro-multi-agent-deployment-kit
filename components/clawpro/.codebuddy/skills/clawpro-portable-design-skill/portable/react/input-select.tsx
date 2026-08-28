/**
 * Portable Input + Select + Field — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 Input / Select 时的可移植兜底实现。
 *  - 不依赖 @radix-ui / shadcn / 任何下拉浮层库（Select 用自定义浮层）。
 *  - 颜色 / 尺寸 / 状态全部由 portable/css/input.css 提供（cp-input / cp-select / cp-field）。
 *  - 视觉规范（spec/component-specs/input-select.md §3 / §4）：
 *      Input：h-9（36px）/ 4px 圆角 / 蓝灰描边 var(--cp-border) /
 *             hover & focus 品牌蓝 #355EF1 / **无 ring 无 shadow** /
 *             placeholder var(--cp-text-weak) / disabled bg #f3f3f4 + 灰字 /
 *             aria-invalid 红描边
 *      Select Trigger：与 Input 完全一致 + 右侧 ChevronDown 图标
 *      Select Content：白底 / 无边框 / 4px 圆角 / 标准三层阴影 / p-2
 *      Select Item：h-8 / 6px 圆角 / px-3 / hover bg-#f3f3f4 /
 *                   选中 text-#355EF1 + Medium + Check
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/input.css";
 *
 * 用法：
 *   <PortableInput placeholder="请输入..." />
 *   <PortableInput aria-invalid disabled />
 *
 *   <PortableSelect
 *     value={v}
 *     onChange={setV}
 *     options={[
 *       { value: "openai", label: "OpenAI" },
 *       { value: "anthropic", label: "Anthropic" },
 *     ]}
 *     placeholder="选择厂商"
 *   />
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

/* ───────────── PortableInput ───────────── */

export type PortableInputProps = React.InputHTMLAttributes<HTMLInputElement>;

export const PortableInput = React.forwardRef<HTMLInputElement, PortableInputProps>(
  ({ className = "", type = "text", ...props }, ref) => {
    const merged = ["cp-input", className].filter(Boolean).join(" ");
    return <input ref={ref} type={type} className={merged} {...props} />;
  }
);
PortableInput.displayName = "PortableInput";

/* ───────────── PortableSelect ─────────────
 * 自定义浮层 Select（不依赖 Radix），1:1 对齐 spec §4。
 *
 * Props：
 *   options       选项数组 [{ value, label, disabled? }]
 *   value         当前值（受控）
 *   onChange      切换回调（拿到 value）
 *   placeholder   未选中时占位文字
 *   disabled      整体禁用
 *   ariaInvalid   报错态（红描边）
 *   className     trigger className
 *   panelClassName 面板 className（如自定义宽度）
 */

export interface PortableSelectOption {
  value: string;
  label: React.ReactNode;
  disabled?: boolean;
}

export interface PortableSelectProps {
  options: PortableSelectOption[];
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: React.ReactNode;
  disabled?: boolean;
  ariaInvalid?: boolean;
  className?: string;
  panelClassName?: string;
  id?: string;
  name?: string;
  /** 触发器自定义渲染（不传则显示 selected option 的 label） */
  renderTrigger?: (selected?: PortableSelectOption) => React.ReactNode;
}

export function PortableSelect({
  options,
  value,
  onChange,
  placeholder = "请选择",
  disabled,
  ariaInvalid,
  className = "",
  panelClassName = "",
  id,
  name,
  renderTrigger,
}: PortableSelectProps) {
  const [open, setOpen] = React.useState(false);
  const triggerRef = React.useRef<HTMLButtonElement>(null);
  const panelRef = React.useRef<HTMLDivElement>(null);
  const [highlight, setHighlight] = React.useState(-1);

  const selected = options.find((o) => o.value === value);

  /* 关闭逻辑：点击外部 + Esc */
  React.useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (
        triggerRef.current?.contains(t) ||
        panelRef.current?.contains(t)
      )
        return;
      setOpen(false);
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
  }, [open]);

  /* 键盘上下选择 + Enter */
  const onTriggerKey = (e: React.KeyboardEvent) => {
    if (disabled) return;
    if (!open && (e.key === "Enter" || e.key === " " || e.key === "ArrowDown")) {
      e.preventDefault();
      setOpen(true);
      const idx = options.findIndex((o) => o.value === value);
      setHighlight(idx >= 0 ? idx : 0);
      return;
    }
    if (open) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setHighlight((h) => Math.min(options.length - 1, h + 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setHighlight((h) => Math.max(0, h - 1));
      } else if (e.key === "Enter") {
        e.preventDefault();
        const opt = options[highlight];
        if (opt && !opt.disabled) {
          onChange?.(opt.value);
          setOpen(false);
        }
      }
    }
  };

  return (
    <span className="cp-select">
      <button
        ref={triggerRef}
        type="button"
        id={id}
        name={name}
        disabled={disabled}
        aria-invalid={ariaInvalid || undefined}
        aria-haspopup="listbox"
        aria-expanded={open}
        data-state={open ? "open" : "closed"}
        data-placeholder={selected ? undefined : "true"}
        onClick={() => !disabled && setOpen((p) => !p)}
        onKeyDown={onTriggerKey}
        className={["cp-select__trigger", className].filter(Boolean).join(" ")}
      >
        <span className="cp-select__value">
          {renderTrigger
            ? renderTrigger(selected)
            : selected?.label ?? placeholder}
        </span>
        <svg
          aria-hidden="true"
          className="cp-select__chevron"
          viewBox="0 0 16 16"
          width="14"
          height="14"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M4 6l4 4 4-4" />
        </svg>
      </button>

      {open && (
        <div
          ref={panelRef}
          role="listbox"
          className={["cp-select__panel", panelClassName].filter(Boolean).join(" ")}
        >
          {options.length === 0 ? (
            <div className="cp-select__empty">暂无选项</div>
          ) : (
            options.map((opt, idx) => {
              const isSelected = opt.value === value;
              const isHighlight = idx === highlight;
              return (
                <button
                  key={opt.value}
                  type="button"
                  role="option"
                  aria-selected={isSelected}
                  disabled={opt.disabled}
                  data-highlight={isHighlight ? "true" : undefined}
                  onMouseEnter={() => setHighlight(idx)}
                  onClick={() => {
                    if (opt.disabled) return;
                    onChange?.(opt.value);
                    setOpen(false);
                  }}
                  className={[
                    "cp-select__item",
                    isSelected && "cp-select__item--selected",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                >
                  <span className="cp-select__value">{opt.label}</span>
                  {isSelected && (
                    <svg
                      aria-hidden="true"
                      className="cp-select__check"
                      viewBox="0 0 16 16"
                      width="14"
                      height="14"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    >
                      <path d="M3 8l3 3 7-7" />
                    </svg>
                  )}
                </button>
              );
            })
          )}
        </div>
      )}
    </span>
  );
}

/* ───────────── PortableField ─────────────
 * Label + control + helper 成组（与 spec form-controls.md §Field 对齐）。
 */

export interface PortableFieldProps {
  label?: React.ReactNode;
  required?: boolean;
  helper?: React.ReactNode;
  helperTone?: "weak" | "danger";
  children: React.ReactNode;
  className?: string;
  htmlFor?: string;
}

export function PortableField({
  label,
  required,
  helper,
  helperTone = "weak",
  children,
  className = "",
  htmlFor,
}: PortableFieldProps) {
  const merged = ["cp-field", className].filter(Boolean).join(" ");
  return (
    <div className={merged}>
      {label !== undefined && (
        <label htmlFor={htmlFor} className="cp-field__label">
          {label}
          {required && (
            <span className="cp-field__required" aria-hidden="true">
              *
            </span>
          )}
        </label>
      )}
      {children}
      {helper !== undefined && (
        <div
          className={[
            "cp-field__helper",
            helperTone === "danger" && "cp-field__helper--danger",
          ]
            .filter(Boolean)
            .join(" ")}
        >
          {helper}
        </div>
      )}
    </div>
  );
}
