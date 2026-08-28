# Selection Controls (Switch / Checkbox / Radio)

> **Showcase mapping**: `checkbox` · `radio-group` · `radio-card` · `switch` · `toggle` · `toggle-group`（`client/src/pages/DesignSystemComponents.tsx`）

## 1. Purpose

- 统一开关、勾选、单选三类表单基础选择控件的视觉规范。
- 确保选中色、描边 token、disabled 态全局一致。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：设置开关、表格行勾选、表单单选/多选、协议确认
- 不适用场景：page header 下方一级 Tab（用 `tabs.md`）、卡片内 / 工具栏切换（用 `segment.md`）、按钮组互斥选择（用 Button plain）

## 3. 通用规则

| 规则 | 说明 |
|---|---|
| 品牌选中色 | `#1447E6` 边框 + `#355EF1` 填充（三者必须一致） |
| 默认描边 | `var(--cp-border-control)` = `#C8CFDA`（不是通用 `--cp-border`） |
| Hover | 边框变品牌蓝 `#1447E6` |
| Focus-visible | `ring-2 ring-[#355EF1]/20` |
| Disabled（未选中） | `bg-[#f3f3f4]` + `cursor-not-allowed` |
| Disabled（已选中） | `bg-[#d3d6db]` + 保留勾/圆点但整体灰化 |
| 禁止 | 不用 `opacity-50` 表达 disabled；不用 `border-gray-200` |

## 4. Checkbox

### 4.1 视觉参数

| Item | Value |
|---|---|
| 尺寸 | `16px × 16px` |
| 圆角 | `4px` |
| 边框宽度 | `1px` |
| 勾图标 | `14px`，白色，居中 |

### 4.2 状态

| 状态 | 边框 | 背景 | 勾色 |
|---|---|---|---|
| 默认 | `#C8CFDA` | 白色 | — |
| Hover | `#1447E6` | 白色 | — |
| Checked | `#1447E6` | `#355EF1` | 白色 |
| Indeterminate（半选） | `#1447E6` | `#355EF1` | 白色（横线） |
| Disabled（默认） | `#C8CFDA` | `#f3f3f4` | — |
| Disabled（checked） | `#C8CFDA` | `#d3d6db` | 白色 |

### 4.3 用法

```tsx
// 与 Label 配合
<div className="flex items-center gap-2">
  <input type="checkbox" className="h-4 w-4 rounded-[4px] border border-[var(--cp-border-control)] accent-[#355EF1]" />
  <label className="text-sm">我已阅读并同意</label>
</div>
```

## 5. Radio

### 5.1 视觉参数

| Item | Value |
|---|---|
| 尺寸 | `16px × 16px` |
| 圆角 | `rounded-full` |
| 圆点尺寸 | `8px × 8px`，绝对居中 |
| 组容器 | `grid gap-3`（默认竖排，12px 间距） |

### 5.2 状态

| 状态 | 边框 | 背景 | 圆点 |
|---|---|---|---|
| 默认 | `#C8CFDA` | 白色 | 隐藏 |
| Hover | `#1447E6` | 白色 | 隐藏 |
| Checked | `#1447E6` | 白色 | `#355EF1` 填充 |
| Disabled | `#C8CFDA` | `#f3f3f4` | — |

### 5.3 卡片式 Radio

部分场景将 Radio 包裹在卡片中，让整张卡可点击：

- 选中态：`border-[#1447E6]` + 浅蓝背景 `#F5F8FF`
- 默认描边复用 `var(--cp-border-control)`
- Radio 本身不隐藏，作为状态指示器一起出现

## 6. Switch

### 6.1 视觉参数

| Item | Value |
|---|---|
| 尺寸 | `h-5 w-9`（20px × 36px） |
| 轨道关闭色 | `#d3d6db` |
| 轨道开启色 | `#355EF1` |
| 滑块 | 白色圆形，4px 内缩 |

### 6.2 状态

| 状态 | 轨道色 | 说明 |
|---|---|---|
| Unchecked | `#d3d6db` | 灰色 |
| Checked | `#355EF1` | 品牌蓝 |
| Disabled | 降低对比度 | `cursor-not-allowed` |

## 7. Portable Fallback

### 7.1 Checkbox fallback

```tsx
function PortableCheckbox({ checked, disabled, onChange }: any) {
  return (
    <button
      role="checkbox"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange?.(!checked)}
      className={[
        "h-4 w-4 rounded-[4px] border flex items-center justify-center transition-colors",
        disabled
          ? "border-[var(--cp-border-control)] bg-[#f3f3f4] cursor-not-allowed"
          : checked
            ? "border-[#1447E6] bg-[#355EF1]"
            : "border-[var(--cp-border-control)] bg-white hover:border-[#1447E6]",
      ].join(" ")}
    >
      {checked && <svg className="w-3.5 h-3.5 text-white" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth={2}><path d="M3 7l3 3 5-5" /></svg>}
    </button>
  );
}
```

### 7.2 Switch fallback

```tsx
function PortableSwitch({ checked, disabled, onChange }: any) {
  return (
    <button
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange?.(!checked)}
      className={[
        "relative h-5 w-9 rounded-full transition-colors",
        disabled ? "opacity-50 cursor-not-allowed" : "cursor-pointer",
        checked ? "bg-[#355EF1]" : "bg-[#d3d6db]",
      ].join(" ")}
    >
      <span className={[
        "absolute top-[2px] h-4 w-4 rounded-full bg-white shadow transition-transform",
        checked ? "translate-x-[18px]" : "translate-x-[2px]",
      ].join(" ")} />
    </button>
  );
}
```

## 8. Migration Rules

| 旧写法 | 新写法 |
|---|---|
| `border-gray-200` Checkbox/Radio | `border-[var(--cp-border-control)]` (#C8CFDA) |
| `opacity-50` 表达 disabled | 具体灰底 `#f3f3f4` |
| Tailwind `blue-500` (#3B82F6) 选中色 | 品牌蓝 `#1447E6` / `#355EF1` |
| 自定义开关颜色 | 统一 Switch 轨道色 |

## 9. Do / Don't

**Do:**
- 三者选中色全局统一（`#1447E6` / `#355EF1`）。
- 默认描边用 `--cp-border-control` token。
- Disabled 用具体灰底色，不用 opacity。

**Don't:**
- 不要用 `border-gray-200`。
- 不要用 `opacity-50` 做 disabled。
- 不要自定义选中色。
- 卡片式 Radio 不要隐藏 Radio 本身。

## 10. QA Checklist

- [ ] Checkbox/Radio/Switch 选中色统一 `#355EF1`
- [ ] 默认描边使用 `--cp-border-control` (#C8CFDA)
- [ ] Disabled 无 hover 反馈
- [ ] Disabled 不用 `opacity-50`
- [ ] 卡片式 Radio 的 Radio 可见
- [ ] fallback 使用 `var(--cp-*)` CSS variable

## 11. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`
- Related tokens: `--cp-border-control` (#C8CFDA), Brand Blue `#1447E6` / `#355EF1`

## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 同口径。Switch / Checkbox / Radio 5 项高频误用 → ClawPro 正确写法。

### 12.1 不要用 Tailwind blue-500 当选中色

```tsx
// ❌ 用 accent-blue-500（#3B82F6），与品牌蓝 #355EF1 不同
<input type="checkbox" className="h-4 w-4 accent-blue-500" />

// ❌ Switch checked 态走 bg-blue-500
<Switch className="data-[state=checked]:bg-blue-500" />

// ✅ Checkbox 走品牌蓝
<input type="checkbox" className="h-4 w-4 accent-[#355EF1]" />

// ✅ Switch 走品牌蓝 token
<Switch />  {/* 内部 data-[state=checked]:bg-[#355EF1] */}
```

### 12.2 默认描边走 --cp-border-control（不是通用 --cp-border）

```tsx
// ❌ Checkbox 用通用 border-gray-200 / --cp-border
<button role="checkbox" className="h-4 w-4 border border-[var(--cp-border)]">…</button>

// ❌ Radio 用 #EAEEF4 通用描边，与 Input 描边混淆
<input type="radio" className="h-4 w-4 border-[#EAEEF4]" />

// ✅ 选择控件专用描边 token
<button role="checkbox" className="h-4 w-4 border border-[var(--cp-border-control)]">…</button>
```

### 12.3 Disabled 用具体灰底，禁 opacity-50

```tsx
// ❌ opacity-50 表达 disabled，对比度 / 颜色都漂
<Checkbox className="opacity-50" disabled />
<Switch className="opacity-50" disabled />

// ✅ Checkbox 具体灰底（未选）
<Checkbox disabled />
{/* 内部：disabled:bg-[#f3f3f4] disabled:cursor-not-allowed */}

// ✅ Checked + Disabled：保留勾，整体走更深一档灰
<Checkbox checked disabled />
{/* 内部：disabled:data-[state=checked]:bg-[#d3d6db] */}
```

### 12.4 卡片式 Radio 不要隐藏 Radio 本身

```tsx
// ❌ 卡片选中靠边框 + 浅蓝底，但把 Radio 圆点藏掉，无障碍丢失
<label className="rounded-[4px] border data-[checked]:border-[#1447E6] data-[checked]:bg-[#F5F8FF] p-4 block">
  <input type="radio" className="sr-only" />
  <h4>方案 A</h4>
</label>

// ✅ Radio 与卡片状态共存，state 双重表达
<label className="rounded-[4px] border data-[checked]:border-[#1447E6] data-[checked]:bg-[#F5F8FF] p-4 flex items-start gap-3">
  <input type="radio" className="mt-0.5 h-4 w-4 border-[var(--cp-border-control)] accent-[#355EF1]" />
  <div>
    <h4 className="text-sm font-medium">方案 A</h4>
    <p className="text-xs text-[var(--cp-text-weak)]">推荐配置</p>
  </div>
</label>
```

### 12.5 Switch 轨道色用规定灰，禁自定义

```tsx
// ❌ 关闭态用 bg-gray-300 / Tailwind slate
<Switch className="data-[state=unchecked]:bg-gray-300" />

// ❌ 开启态用 emerald / sky 表示\"启用\"语义
<Switch className="data-[state=checked]:bg-emerald-500" />

// ✅ 统一规范：unchecked #d3d6db / checked #355EF1
<Switch />
{/* 内部：
   data-[state=unchecked]:bg-[#d3d6db]
   data-[state=checked]:bg-[#355EF1]
*/}
```
