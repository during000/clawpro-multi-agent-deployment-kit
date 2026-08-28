# Date Picker

## 1. Purpose

- 统一日期选择触发器、弹出日历面板和 Admin / Tenant 分端圆角口径。
- 这类组件在宿主仓里通常有现成逻辑，但视觉最容易留在旧系统，尤其是触发器外壳和日历选中态。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：时间筛选、有效期设置、配置生效日期、时间范围表单
- 需要日期 + 时分（秒）的精确时间点：用本规范 §13 的 DateTimePicker 变体
- 不适用场景：复杂排班日历、营销活动日历视图

## 3. Visual Standard

| Item | Admin | Tenant | Notes |
|---|---|---|---|
| Trigger Height | 36px / `h-9` | 36px / `h-9` | 与 Input / Select 对齐 |
| Trigger Radius | 4px | 筛选场景可胶囊；普通表单 / 弹窗表单 4px | Tenant 跟随场景，不一律 full |
| Trigger Border | `--border` / 当前 `#EAEEF4` | 同体系 | 默认使用蓝灰描边；hover / open 用品牌蓝 token |
| Placeholder | 弱灰文字 | 弱灰文字 | 不要和已选值同色 |
| Calendar Surface | 白底浮层 | 白底浮层 | 用 Popover / Portal 承载 |
| Selected Day | 品牌蓝底白字 | 品牌蓝底白字 | 今天可有弱提示点，不抢选中态 |

## 4. Anatomy

```text
DatePicker
  Trigger
    Value / Placeholder
    CalendarIcon
  Popover
    Calendar Header
    Calendar Grid
```

## 5. States

- default: 未选择日期，显示 placeholder。
- selected: 已选择日期，显示格式化日期。
- open: 触发器边框强调，浮层展开。
- disabled: 灰底灰字、不可点击。
- with-range-pair: 开始 / 结束日期成对出现，高度和间距一致。
- tenant: 按场景决定 trigger 圆角；搜索 / 筛选可胶囊，普通表单 / 弹窗表单 4px，不单独发明另一套日历视觉。

## 6. Demo Repo Usage

- 当前组件：`client/src/components/ui/date-picker.tsx`
- 当前日历底层：`client/src/components/ui/calendar.tsx`
- 管理端筛选页：`client/src/pages/admin/AuditLog.tsx`、`client/src/pages/admin/TokensMonitor.tsx`、`client/src/pages/admin/OpenClawMonitor.tsx`
- 用户端筛选页：`client/src/pages/tenant/ModelQuota.tsx`

```tsx
<DatePicker
  value={dateFrom}
  onChange={setDateFrom}
  placeholder="开始日期"
  className="w-[140px] bg-[var(--cp-surface)]"
/>

<DatePicker
  value={tenantDate}
  onChange={setTenantDate}
  placeholder="选择日期"
  tenant
/>
```

## 7. Portable Fallback

### 7.1 If host repo already has DatePicker

- 保留宿主仓现有日期选择逻辑。
- 只要求对齐触发器高度、边框、placeholder、选中态颜色和 Admin / Tenant 圆角分流。
- 如果宿主仓只有 Calendar 没有封装好的 DatePicker，优先用其 Popover + Calendar 组合，而不是重新手写一套日历交互。

### 7.2 Minimal React fallback

```tsx
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
    <div className="relative inline-flex flex-col">
      <button
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        className={[
          "inline-flex h-9 min-w-[140px] items-center justify-between gap-2 border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-sm",
          tenant ? "rounded-full" : "rounded-[4px]",
          open ? "border-[var(--cp-brand-blue)]" : "",
        ].join(" ")}
      >
        <span className={value ? "text-[var(--cp-text-title)]" : "text-[var(--cp-text-weak)]"}>{value || placeholder}</span>
        <span className="text-[var(--cp-text-weak)]" aria-hidden="true">Cal</span>
      </button>

      {open && (
        <div className="absolute left-0 top-11 z-10 w-[280px] rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] p-3 shadow-sm">
          <div className="mb-3 flex items-center justify-between text-sm text-[var(--cp-text-title)]">
            <button type="button">‹</button>
            <span>2026 年 6 月</span>
            <button type="button">›</button>
          </div>
          <div className="grid grid-cols-7 gap-1 text-center text-xs text-[var(--cp-text-muted)]">
            {weekdays.map((day) => <span key={day}>{day}</span>)}
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
                    "mt-1 h-8 rounded-[4px] text-sm",
                    active ? "bg-[var(--cp-brand-blue)] text-white" : "text-[var(--cp-text-title)] hover:bg-[var(--cp-brand-tint)]",
                  ].join(" ")}
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
```

### 7.3 Minimal HTML/CSS fallback

```html
<div class="cp-date-picker">
  <button class="cp-date-trigger" type="button">
    <span class="cp-date-placeholder">开始日期</span>
    <span class="cp-date-icon" aria-hidden="true">Cal</span>
  </button>
  <div class="cp-date-panel">
    <div class="cp-date-header">
      <button type="button">‹</button>
      <span>2026 年 6 月</span>
      <button type="button">›</button>
    </div>
    <div class="cp-date-grid cp-date-weekdays">
      <span>一</span><span>二</span><span>三</span><span>四</span><span>五</span><span>六</span><span>日</span>
    </div>
    <div class="cp-date-grid cp-date-days">
      <button type="button">1</button>
      <button type="button">2</button>
      <button type="button">3</button>
      <button type="button">4</button>
      <button type="button">5</button>
      <button type="button" class="active">6</button>
      <button type="button">7</button>
    </div>
  </div>
</div>
```

```css
.cp-date-picker { position: relative; display: inline-flex; flex-direction: column; }
.cp-date-trigger { display: inline-flex; align-items: center; justify-content: space-between; gap: 8px; min-width: 140px; height: 36px; padding: 0 12px; border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); font-size: 14px; }
.cp-date-placeholder { color: var(--cp-text-weak); }
.cp-date-icon { color: var(--cp-text-weak); }
.cp-date-panel { position: absolute; top: 44px; left: 0; z-index: 10; width: 280px; border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 12px; box-shadow: var(--cp-shadow-overlay); }
.cp-date-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; font-size: 14px; color: var(--cp-text-title); }
.cp-date-grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 4px; text-align: center; }
.cp-date-weekdays { font-size: 12px; color: var(--cp-text-muted); }
.cp-date-days button { margin-top: 4px; height: 32px; border: 0; border-radius: 4px; background: var(--cp-surface); font-size: 14px; color: var(--cp-text-title); }
.cp-date-days button:hover { background: var(--cp-brand-tint); }
.cp-date-days button.active { background: var(--cp-brand-blue); color: white; }
```

## 8. Migration Rules

- 旧写法：宿主仓直接沿用原有日期触发器，外壳高度、边框、选中态都和当前系统不一致。
- 新口径：先统一 DatePicker 触发器和日历面板视觉，再决定底层沿用哪个日期逻辑组件。
- 可以暂时兼容：宿主仓继续使用自己的日期库和格式化逻辑。
- 不允许新增：Admin / Tenant 混用错误圆角；日期触发器比 Input / Select 高一截；选中态继续用旧品牌色。

## 9. Do / Don't

Do:

- 让 DatePicker 与 Input / Select 高度对齐。
- 开始日期 / 结束日期成对时保持相同宽度和间距。
- 只在端别差异上调整圆角，不为 Tenant 单独发明另一套日历结构。

Don't:

- 不要把日历 icon 省掉，只留一个纯文字按钮。
- 不要把打开态做成厚重阴影或高饱和 hover。
- 不要让 placeholder、禁用态和已选值颜色混在一起。

## 10. QA Checklist

- [ ] Trigger 高度与 Input / Select 一致
- [ ] Admin / Tenant 圆角口径正确
- [ ] open / selected / disabled 状态完整
- [ ] 日历选中态使用品牌蓝
- [ ] 宿主仓 fallback 可执行

## 11. References

- Demo code: `client/src/components/ui/date-picker.tsx`
- Demo code: `client/src/components/ui/calendar.tsx`
- Demo page: `client/src/pages/admin/AuditLog.tsx`
- Demo page: `client/src/pages/admin/TokensMonitor.tsx`
- Demo page: `client/src/pages/tenant/ModelQuota.tsx`

## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 / §3 同口径。DatePicker 5 项高频误用 → ClawPro 正确写法。

### 12.1 Trigger 与同行 Input / Select 等高（h-9）

```tsx
// ❌ 日期选择硬塞 h-10，比同行 Input 高一截
<button className="h-10 rounded-[4px] border w-36">2026-06-09</button>

// ❌ 直接用 antd / 旧库 DatePicker，触发器 32px
<OldDatePicker style={{ height: 32 }} />

// ✅ 统一 36px / h-9
<DatePicker value={date} onChange={setDate} placeholder="开始日期" className="w-[140px]" />
```

### 12.2 端别圆角分流（Admin 4px / Tenant 筛选胶囊 / Tenant 表单 4px）

```tsx
// ❌ Admin 配置页用 rounded-full（tenant prop 误传）
<DatePicker tenant value={date} onChange={setDate} className="w-36" />

// ❌ Tenant 弹窗表单用 rounded-full（应跟随场景：表单 4px）
<DatePicker tenant value={date} onChange={setDate} placeholder="生效日期" />

// ✅ Admin 全场景 4px
<DatePicker value={date} onChange={setDate} placeholder="开始日期" />

// ✅ Tenant 搜索 / 筛选：胶囊
<TenantFilterDatePicker value={date} onChange={setDate} placeholder="选择日期" />

// ✅ Tenant 表单 / 弹窗表单：4px
<TenantFormDatePicker value={date} onChange={setDate} placeholder="生效日期" />
```

### 12.3 不省略 CalendarIcon（视觉锚点）

```tsx
// ❌ 触发器只剩文字，看起来像普通按钮
<button className="h-9 rounded-[4px] border px-3 text-left w-36">
  {date || "开始日期"}
</button>

// ✅ 右侧固定 CalendarIcon 弱灰色
<button className="h-9 rounded-[4px] border px-3 w-36 flex items-center justify-between">
  <span className={date ? "text-[var(--cp-text-title)]" : "text-[var(--cp-text-weak)]"}>
    {date || "开始日期"}
  </span>
  <CalendarIcon className="h-4 w-4 text-[var(--cp-text-weak)]" />
</button>
```

### 12.4 选中日：品牌蓝实底，今日只是弱提示

```tsx
// ❌ 把"今天"做成主蓝实心，与已选日抢视觉
<button className="h-8 rounded-[4px] bg-[var(--cp-brand-blue)] text-white">9</button>  {/* 今天 */}
<button className="h-8 rounded-[4px] border border-[var(--cp-brand-blue)]">12</button> {/* 已选 */}

// ✅ 已选日：实心品牌蓝；今日：底部 1.5px 蓝色小点
<button className="h-8 rounded-[4px] hover:bg-[var(--cp-brand-tint)]">
  <span className="relative">
    9
    <span className="absolute left-1/2 -translate-x-1/2 -bottom-1 h-1 w-1 rounded-full bg-[var(--cp-brand-blue)]" />
  </span>
</button>

<button className="h-8 rounded-[4px] bg-[var(--cp-brand-blue)] text-white">12</button>
```

### 12.5 范围选择：起止两个 Trigger 等宽对齐

```tsx
// ❌ 起止宽度参差，间距用空格 / 单独 ml
<div>
  <DatePicker value={from} onChange={setFrom} className="w-32" placeholder="开始" />
  <span className="mx-2">至</span>
  <DatePicker value={to} onChange={setTo} className="w-40" placeholder="结束" />
</div>

// ✅ 等宽 + flex gap，连接符语义清晰
<div className="flex items-center gap-2">
  <DatePicker value={from} onChange={setFrom} placeholder="开始日期" className="w-[140px]" />
  <span className="text-[var(--cp-text-weak)]">至</span>
  <DatePicker value={to} onChange={setTo} placeholder="结束日期" className="w-[140px]" />
</div>
```

## 13. DateTimePicker 变体（日期 + 时分秒）

> 当业务需要精确到「分」或「秒」的时间点（定时任务、生效时间、日志时间点）时，用 DateTimePicker，而不是 DatePicker 旁边再塞一个时间输入框。它复用同一个 react-day-picker 日历与品牌蓝选中态，只在右侧扩展时间列。

### 13.1 Scope

- 必用场景：定时任务执行时间、精确生效 / 失效时间点、到秒级的配置与日志筛选。
- 不适用场景：仅需选日期（用 DatePicker）、时间范围段（用两个 DateTimePicker 成对）、复杂排班。

### 13.2 Anatomy

```text
DateTimePicker
  Trigger（同 DatePicker：Value / Placeholder + CalendarIcon）
  Popover
    Calendar（左，复用 DatePicker 全部能力）
    TimeColumns（右：时 / 分 /（秒），选中态品牌蓝）
    Footer（预览文本 + 「确定」黑底按钮）
```

### 13.3 Visual Standard（增量）

| Item | 规范 | Notes |
|---|---|---|
| Trigger | 与 DatePicker 完全一致（`h-9`、Admin 4px / Tenant 圆角分流、品牌蓝 hover/open） | 唯一差异是显示值带时间 |
| Time Column 选中态 | 品牌蓝实底白字 `#1447E6` | 与日历选中日同色，不另发明色 |
| Time Column hover | 弱蓝 `#eff4ff` | 不抢选中态 |
| Footer 预览 | 弱灰文字，显示草稿态完整值 | 与已选触发器值同格式 |
| Footer 确定按钮 | 黑底白字（Button 默认变体） | 草稿态：点确定才提交 |

### 13.4 Props（在 DatePicker 基础上新增）

- `showSeconds?: boolean`：默认 `false`。开启后显示「秒」列，值格式扩展为 `YYYY-MM-DD HH:mm:ss`。
- `minuteStep?: number`：分钟步长，默认 1。
- `secondStep?: number`：秒步长，默认 1，仅 `showSeconds` 时生效。
- `value` / `onChange`：默认格式 `YYYY-MM-DD HH:mm`；`showSeconds` 时为 `YYYY-MM-DD HH:mm:ss`。
- 其余 `placeholder` / `disabled` / `min` / `max` / `tenant` / `className` 与 DatePicker 对齐。

### 13.5 代码对照（✅/❌）

```tsx
// ❌ DatePicker 旁边硬塞一个时间输入框，两套外壳、两种交互
<DatePicker value={date} onChange={setDate} />
<input type="time" className="h-9 border" />

// ✅ 日期 + 时分一体，值格式 YYYY-MM-DD HH:mm
<DateTimePicker value={dt} onChange={setDt} placeholder="选择日期时间" />

// ✅ 需要到秒：开启 showSeconds，值格式 YYYY-MM-DD HH:mm:ss
<DateTimePicker showSeconds value={dt} onChange={setDt} />

// ✅ 秒按 5 步长，Tenant 胶囊
<DateTimePicker showSeconds secondStep={5} tenant value={dt} onChange={setDt} />
```

### 13.6 QA Checklist（增量）

- [ ] 触发器与 DatePicker / Input / Select 同高 `h-9`
- [ ] 时 / 分 /（秒）列选中态使用品牌蓝，与日历选中日同色
- [ ] `showSeconds` 关闭时值格式不含秒，开启后为 `HH:mm:ss`
- [ ] 草稿态：选日期 / 时分秒只更新预览，点「确定」才 `onChange`
- [ ] 展示台已接入：`/design-system/components` → DateTimePicker

### 13.7 References

- Demo code: `client/src/components/ui/date-time-picker.tsx`
- Showcase: `client/src/pages/DesignSystemComponents.tsx`（DateTimePicker 可交互示例）
