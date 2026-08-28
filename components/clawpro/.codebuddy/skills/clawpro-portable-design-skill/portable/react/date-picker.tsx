/**
 * Portable DatePicker — ClawPro Portable Design Skill
 * ───────────────────────────────────────────────────────────────────────────
 * 用途：宿主仓没有同构 DatePicker 时的可移植兜底实现（演示骨架）。
 *  - 不依赖 shadcn / Radix / Tailwind；样式由 portable/css/date-picker.css 提供。
 *  - 视觉规范（spec/component-specs/date-picker.md §3）：
 *      触发器：36px 高 / 4px 圆角（Tenant 胶囊全圆角）/ open 时蓝描边
 *      面板：280px + 蓝灰描边 + overlay 阴影 + 7 列日历网格
 *      当前选中日：品牌蓝底 + 白字；hover：浅蓝 tint
 *
 * ⚠️ 必须同时引入：
 *    import "../css/tokens.css";
 *    import "../css/date-picker.css";
 *
 * 用法：
 *   <PortableDatePicker placeholder="选择日期" />
 *   <PortableDatePicker tenant />   // Tenant 胶囊形态
 * ───────────────────────────────────────────────────────────────────────────
 */
import * as React from "react";

export function PortableDatePicker({
  placeholder = "选择日期",
  tenant = false,
}: {
  placeholder?: string;
  tenant?: boolean;
}) {
  const [open, setOpen] = React.useState(false);
  const [value, setValue] = React.useState("");
  const weekdays = ["一", "二", "三", "四", "五", "六", "日"];

  return (
    <div className="cp-date-picker">
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        className={[
          "cp-date-picker__trigger",
          tenant && "cp-date-picker__trigger--tenant",
          open && "cp-date-picker__trigger--open",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <span
          className={
            value
              ? "cp-date-picker__value"
              : "cp-date-picker__value cp-date-picker__value--placeholder"
          }
        >
          {value || placeholder}
        </span>
        <span className="cp-date-picker__suffix" aria-hidden="true">
          Cal
        </span>
      </button>

      {open && (
        <div className="cp-date-picker__panel">
          <div className="cp-date-picker__header">
            <button type="button" className="cp-date-picker__nav">
              ‹
            </button>
            <span>2026 年 6 月</span>
            <button type="button" className="cp-date-picker__nav">
              ›
            </button>
          </div>
          <div className="cp-date-picker__grid">
            {weekdays.map((day) => (
              <span key={day}>{day}</span>
            ))}
            {Array.from({ length: 30 }).map((_, index) => {
              const label = index + 1;
              const active = label === 6;
              return (
                <button
                  key={label}
                  type="button"
                  onClick={() => {
                    setValue(`2026-06-${String(label).padStart(2, "0")}`);
                    setOpen(false);
                  }}
                  className={[
                    "cp-date-picker__day",
                    active && "cp-date-picker__day--active",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                >
                  {label}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
