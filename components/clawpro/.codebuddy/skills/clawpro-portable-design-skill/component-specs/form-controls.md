# Form Controls

## 1. Purpose

- 统一 Input、Select、筛选区和基础表单控件的宿主仓还原口径。
- 防止换皮时只改按钮和卡片，表单控件仍留在旧系统里。

## 2. Scope

- 适用端：Admin / Tenant / Shared
- 必用场景：搜索框、普通输入框、选择器、筛选条、弹窗表单
- 不适用场景：富文本编辑器、代码编辑器、复杂图形化控件

## 3. Visual Standard

| Item | Admin | Tenant | Notes |
|---|---|---|---|
| Input Radius | 4px | 搜索 / 筛选控件胶囊；普通表单和弹窗表单 4px | Tenant 双轨，不再一律 full |
| Border | `--border` / 当前 `#EAEEF4` | 同体系 | 默认使用蓝灰描边；hover / focus 用品牌蓝 token |
| Search Icon | 左侧弱灰图标 | 同逻辑 | 不省略 |
| Filter Gap | `gap-3` | `gap-3` | 不逐个写 margin |
| Select Panel | 白底浮层 | 白底浮层 | 用 portal 逃逸裁剪 |

### 3.1 子元素字号 / 色 token（改容器必贯彻，P3）

> ⚠️ 改控件容器（圆角 / 边框 / 高度）时**必须同时核对子元素走语义 token**，不得只对齐外框、内部文字 / 占位仍是旧字号 / 旧色：
>
> | 子元素 | 字号 / 高度 | 颜色 token |
> |---|---|---|
> | 控件高度 | `h-9`（36px） | — |
> | 输入文字 | `text-sm`（14px） | `var(--cp-text-title)` / 已输入正文 `var(--cp-text-body)` |
> | Placeholder | `text-sm`（14px） | `var(--cp-text-muted)` |
> | 左侧 Search icon | 16px | `var(--cp-text-weak)` |
> | Label | `text-sm`（14px） | `var(--cp-text-title)`（Label→Control 间距 `space-y-2`） |
> | Helper / Error | `text-xs`（12px） | Helper `var(--cp-text-weak)` / Error `var(--cp-text-danger)` |
> | 控件描边 | — | `var(--cp-border)`；hover / focus 用 `var(--cp-brand-blue)` |
>
> 一律走 token，不散写 `text-[#xxx]` / `text-gray-*` / 边框 hex（如 `#355EF1`）。

## 4. Anatomy

```text
Field
  Label optional
  Control
    Input / Select / DatePicker
  Helper / Error optional
```

## 5. States

- default: 边框清晰、背景白。
- hover: 可轻微强调边框。
- focus: 品牌蓝弱边 / ring。
- disabled: 灰底、灰字、不可操作。
- error: 靠近字段给错误提示。
- loading: 异步选项或提交 loading 要明确。

## 6. Demo Repo Usage

- 当前 demo 仓输入组件：`client/src/components/ui/input.tsx`
- 典型管理端筛选：`client/src/pages/admin/AuditLog.tsx`
- 核心不是强制同 API，而是让宿主仓也对齐控件视觉、圆角和筛选区节奏。

```tsx
<div className="flex flex-wrap items-center gap-3">
  <div className="relative min-w-48 max-w-xs flex-1">
    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--cp-text-weak)]" />
    <Input className="pl-9" placeholder="搜索操作人或操作事件" />
  </div>
  <Select />
</div>
```

## 7. Portable Fallback

### 7.1 If host repo already has Input / Select

- 保留宿主仓控件逻辑。
- 只要求按 Admin / Tenant 场景分流圆角、边框、筛选区排列和焦点态；Tenant 搜索 / 筛选可胶囊，普通表单和弹窗表单保持 4px。

### 7.2 Minimal React fallback

```tsx
export function PortableAdminInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className="h-9 w-full rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-sm text-[var(--cp-text-body)]" />;
}

export function PortableTenantSearchInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className="h-9 w-full rounded-full border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-sm text-[var(--cp-text-body)]" />;
}

export function PortableTenantFormInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input {...props} className="h-9 w-full rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-sm text-[var(--cp-text-body)]" />;
}
```

## 8. Migration Rules

- 旧写法：宿主仓旧表单控件继续沿用旧描边、旧圆角、旧筛选布局。
- 新口径：优先对齐输入框和筛选区视觉，再做更深层组件替换。
- 可以暂时兼容：逻辑组件不换，只改样式层。
- 不允许新增：筛选区继续每个控件单独手写 margin；焦点态缺失；Admin / Tenant 控件圆角混用。

## 9. Do / Don't

Do:

- 搜索框保留左侧 icon。
- 用统一 `gap-3` 组织筛选区。
- 在弹窗内为长表单提供滚动和 helper text。

Don't:

- 不要在页面里继续发明新输入框样式。
- 不要把旧系统 outline 直接当业务按钮用。
- 不要忽略焦点态和禁用态。

## 10. QA Checklist

- [ ] Admin / Tenant 圆角口径正确
- [ ] 搜索、筛选、选择控件间距统一
- [ ] focus / disabled / error 状态完整
- [ ] 宿主仓 fallback 可执行

## 11. References

- Demo code: `client/src/components/ui/input.tsx`
- Demo page: `client/src/pages/admin/AuditLog.tsx`
- Related rules: `references/components.md`


## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 / §6 同口径。表单容器 5 项高频误用 → ClawPro 正确写法。

### 12.1 筛选条用 gap-3，不要逐个写 margin

```tsx
// ❌ 每个控件单独 ml-2，跨页节奏漂移
<div className="flex flex-wrap items-center">
  <Input className="ml-0 w-48" placeholder="搜索" />
  <Select className="ml-2 w-32" />
  <DatePicker className="ml-2 w-36" />
  <Button className="ml-2">筛选</Button>
</div>

// ✅ 容器 gap-3，控件本身不写 margin
<div className="flex flex-wrap items-center gap-3">
  <Input className="w-48" placeholder="搜索" />
  <Select className="w-32" />
  <DatePicker className="w-36" />
  <Button>筛选</Button>
</div>
```

### 12.2 搜索框保留左侧 icon，不省略

```tsx
// ❌ 只剩纯输入框，搜索语义靠 placeholder 文案承载
<Input placeholder="搜索操作人或事件" className="w-64" />

// ✅ 左侧 Search icon + placeholder 双重提示
<div className="relative w-64">
  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--cp-text-weak)]" />
  <Input className="pl-9" placeholder="搜索操作人或事件" />
</div>
```

### 12.3 弹窗内不要重新发明输入框

```tsx
// ❌ Dialog 内手写一套独立样式
<DialogContent>
  <input className="border border-gray-200 rounded-lg h-10 px-4" placeholder="名称" />
</DialogContent>

// ✅ 直接复用全局 Input（高度、圆角、focus 跟全局走）
<DialogContent>
  <Field>
    <Label>名称</Label>
    <Input placeholder="名称" />
  </Field>
</DialogContent>
```

### 12.4 同行控件等高（h-9 同高，不混用 h-10 / h-8）

```tsx
// ❌ Button 用了 h-10、Input 是 h-9，行高错位
<div className="flex items-center gap-3">
  <Input className="w-48 h-9" placeholder="搜索" />
  <Button className="h-10">查询</Button>
</div>

// ✅ 全部 h-9（默认尺寸）
<div className="flex items-center gap-3">
  <Input className="w-48" placeholder="搜索" />
  <Button size="claw">查询</Button>
</div>
```

### 12.5 端别圆角分流（Admin 4px / Tenant 搜索胶囊 / Tenant 表单 4px）

```tsx
// ❌ Tenant 弹窗表单也用胶囊，输入字段读起来像按钮
<DialogContent data-end="tenant">
  <input className="rounded-full border h-9 px-3" placeholder="API Key" />
</DialogContent>

// ❌ Admin 筛选条用胶囊，与同行卡片 4px 视觉不一致
<div data-end="admin" className="flex gap-3">
  <Input className="rounded-full" placeholder="搜索" />
  <Select className="rounded-full" />
</div>

// ✅ Admin 筛选 + 普通表单：4px
<Input placeholder="搜索操作人" />          {/* h-9 rounded-[4px] */}

// ✅ Tenant 搜索 / 筛选：胶囊
<TenantSearchInput placeholder="搜索 Agent" />  {/* h-9 rounded-full */}

// ✅ Tenant 弹窗表单 / 普通表单字段：仍是 4px
<TenantFormInput placeholder="API Key" />       {/* h-9 rounded-[4px] */}
```
