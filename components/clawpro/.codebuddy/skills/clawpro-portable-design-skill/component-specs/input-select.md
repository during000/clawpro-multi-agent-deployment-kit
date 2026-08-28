# Input & Select

> **Showcase mapping**: `input` · `input-group` · `textarea` · `select`（`client/src/pages/DesignSystemComponents.tsx`）。日期类走 `date-picker.md`，搜索型对象选择走 `combobox.md`（alias 文档：实际组件为 `select.tsx` 内的 `SearchableSelect`）。

## 1. Purpose

- 统一所有文本输入框和下拉选择器的视觉规范。
- 确保 Input / Select 在页面、弹窗、筛选区、表单中描边、焦点、报错、禁用态完全一致。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：表单字段、筛选区、搜索框、弹窗内表单
- 不适用场景：带搜索的对象选择器（请用 `select.tsx` 的 `SearchableSelect`，旧名 Combobox；spec 见 `combobox.md`）

## 3. Visual Standard — Input

### 3.1 尺寸与圆角

| Item | Value |
|---|---|
| 高度 | `36px` (h-9) |
| 圆角 | `4px` |
| 横向 padding | `px-3` (12px) |
| 字号 | `14px` (text-sm) |
| 文字色 | `var(--cp-text-emphasis)` (#020617) |

### 3.2 状态

| 状态 | 边框 | 背景 | 其他 |
|---|---|---|---|
| 默认 | `var(--cp-border)` (#EAEEF4) | 白色 | — |
| Hover | `#355EF1`（品牌蓝） | 白色 | — |
| Focus | `#355EF1` | 白色 | **无 ring、无 shadow** |
| 报错 | `#d42a1e`（危险红） | 白色 | — |
| Disabled | `var(--cp-border)` | `#f3f3f4` | `text-[var(--cp-text-weak)]` + `cursor-not-allowed` |
| Placeholder | — | — | `var(--cp-text-weak)` (#94A3B8) |

### 3.3 禁止事项

- 禁止默认态加底色（如 `bg-gray-50`）
- 禁止 focus 加 ring / box-shadow（只变边框色）
- 禁止 disabled 态有 hover 样式（边框变蓝、底色加深等）
- 禁止在弹窗内重新编造 Input 样式

## 4. Visual Standard — Select

### 4.1 Trigger（与 Input 完全一致）

- `h-9 rounded-[4px] border-[var(--cp-border)]`
- **宽度**：`w-full`（必须填满父容器，确保 ChevronDown 图标始终靠右；禁止用 `w-fit`）
- Hover / Open：`border-[#355EF1]`
- Disabled：同 Input

### 4.2 Content（下拉面板）

| Item | Value |
|---|---|
| 背景 | 白色 |
| 边框 | 无 |
| 圆角 | `4px` |
| 阴影 | `0px 0px 2px rgba(0,0,0,0.1), 0px 4px 16px rgba(0,0,0,0.12)` |
| 内边距 | `p-2` |

### 4.3 Item（选项）

| Item | Value |
|---|---|
| 高度 | `32px` (h-8) |
| 圆角 | `6px` |
| 横向 padding | `px-3` |
| Hover | `bg-[#f3f3f4]` |
| 选中态 | `text-[#355EF1] font-medium` + 蓝色勾号 |

## 5. Portable Fallback

### 5.1 Input React fallback

```tsx
function PortableInput({ error, disabled, placeholder, ...props }: any) {
  return (
    <input
      className={[
        "h-9 w-full rounded-[4px] border px-3 text-sm text-[var(--cp-text-emphasis)] outline-none transition-colors",
        "placeholder:text-[var(--cp-text-weak)]",
        error
          ? "border-[#d42a1e]"
          : disabled
            ? "border-[var(--cp-border)] bg-[#f3f3f4] text-[var(--cp-text-weak)] cursor-not-allowed"
            : "border-[var(--cp-border)] hover:border-[#355EF1] focus:border-[#355EF1]",
      ].join(" ")}
      disabled={disabled}
      placeholder={placeholder}
      {...props}
    />
  );
}
```

### 5.2 Select React fallback

```tsx
function PortableSelect({ options, value, onChange, disabled }: any) {
  return (
    <select
      className={[
        "h-9 w-full appearance-none rounded-[4px] border px-3 text-sm text-[var(--cp-text-emphasis)] outline-none transition-colors bg-white",
        disabled
          ? "border-[var(--cp-border)] bg-[#f3f3f4] text-[var(--cp-text-weak)] cursor-not-allowed"
          : "border-[var(--cp-border)] hover:border-[#355EF1] focus:border-[#355EF1]",
      ].join(" ")}
      value={value}
      onChange={(e) => onChange?.(e.target.value)}
      disabled={disabled}
    >
      {options.map((opt: any) => (
        <option key={opt.value} value={opt.value}>{opt.label}</option>
      ))}
    </select>
  );
}
```

## 6. Migration Rules

| 旧写法 | 新写法 |
|---|---|
| Input 加灰底 `bg-gray-50` | 白底 + `border-[var(--cp-border)]` |
| Focus 加 ring / shadow | 只变边框色 `#355EF1` |
| Disabled 态有 hover 样式 | Disabled 锁死，无任何 hover 反馈 |
| 弹窗内重新编造 Input 样式 | 统一复用全局 Input |
| Select 加自定义阴影 | 使用标准三层阴影 |

## 7. Do / Don't

**Do:**
- Input 和 Select 视觉完全对齐（高度、圆角、描边、焦点色）。
- 同行控件高度必须一致（如 Input h-9 + Button h-9）。
- 弹窗内直接复用全局 Input/Select，不二次编造。

**Don't:**
- 不要给默认态加底色。
- 不要给 focus 加 ring 或 shadow。
- 不要给 disabled 态加 hover 效果。
- 不要在弹窗/抽屉里重新写一套控件样式。

## 8. QA Checklist

- [ ] Input 默认态白底 + 蓝灰描边
- [ ] Focus 只变边框色，无 ring/shadow
- [ ] Disabled 无 hover 反馈
- [ ] Select trigger 与 Input 视觉一致
- [ ] Select 面板无边框、有阴影、`4px` 圆角
- [ ] 同行控件等高
- [ ] 弹窗内未重新编造样式
- [ ] fallback 使用 `var(--cp-*)` CSS variable

## 9. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`
- Related tokens: `--cp-border`, `--cp-text-emphasis`, `--cp-text-weak`
- Related specs: `component-specs/form-controls.md`, `component-specs/combobox.md`

## 10. 代码对照（✅/❌）

> 与 SKILL.md §2 / §3 同口径。Input / Select 5 项高频误用 → ClawPro 正确写法。

### 10.1 默认态不要加底色

```tsx
// ❌ 默认态加灰底，与表格行 / 卡片背景冲突
<input className="h-9 w-full rounded-[4px] border border-[var(--cp-border)] bg-gray-50 px-3" />

// ✅ 默认白底
<input className="h-9 w-full rounded-[4px] border border-[var(--cp-border)] bg-white px-3" />
```

### 10.2 Focus 只变边框色，禁 ring / box-shadow

```tsx
// ❌ Focus 加 ring 蓝光圈
<Input className="focus:ring-2 focus:ring-blue-500 focus:ring-offset-2" />

// ❌ Focus 加 box-shadow 涟漪
<input className="h-9 border focus:shadow-[0_0_0_3px_rgba(53,94,241,0.2)]" />

// ✅ 只把边框色切到 #355EF1
<Input />  {/* 内部：focus:border-[#355EF1] */}
```

### 10.3 Disabled 锁死，禁 hover 变蓝 / opacity-50

```tsx
// ❌ disabled 还有 hover 边框变蓝，给用户错觉可点
<input className="h-9 border-[var(--cp-border)] hover:border-[#355EF1] disabled:bg-[#f3f3f4]" disabled />

// ❌ 用 opacity-50 表达禁用，对比度不达标
<Input className="opacity-50" disabled />

// ✅ 具体灰底 + cursor-not-allowed，无 hover 反馈
<Input disabled />
{/* 内部：disabled:bg-[#f3f3f4] disabled:text-[var(--cp-text-weak)] disabled:cursor-not-allowed
        且 hover/focus 样式被 disabled: 前缀豁免 */}
```

### 10.4 Select 面板使用标准三层阴影，不自定义

```tsx
// ❌ Select 面板硬塞 Tailwind shadow-lg，阴影方向 / 模糊都和全局不一致
<SelectContent className="shadow-lg border border-gray-200 rounded-[8px]">
  ...
</SelectContent>

// ✅ 4px 圆角 + 全局 overlay 阴影（无边框，靠阴影区隔层级）
<SelectContent className="rounded-[4px] border-0 shadow-[var(--cp-shadow-overlay)] p-2">
  ...
</SelectContent>
```

### 10.5 Select 选中态：品牌蓝文字 + Check，禁底色高亮

```tsx
// ❌ 选中项加蓝底块，与 hover 灰底冲突
<SelectItem className="data-[state=checked]:bg-[#355EF1]/10 data-[state=checked]:text-[#355EF1]">
  策略 A
</SelectItem>

// ✅ 选中态文字变蓝 + Medium + 蓝色 Check，背景仍跟随 hover 灰
<SelectItem className="data-[state=checked]:text-[#355EF1] data-[state=checked]:font-medium">
  策略 A
</SelectItem>
```
