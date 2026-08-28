# Search / Filter Bar

## 1. Purpose

- 统一列表页顶部的搜索、筛选、刷新和主操作工具条。
- 这块区域在宿主仓换皮时最容易被“顺手拼一下”，最后虽然功能能用，但 spacing、控件高度、分组关系和操作优先级都会散。

## 2. Scope

- 适用端：Admin 优先；Tenant 只有列表检索型页面时复用结构
- 必用场景：列表页顶部工具条、表格上方筛选区、抽屉内轻量检索区
- 不适用场景：完整表单页、导航栏搜索、Landing Hero 搜索

## 3. Visual Standard

| Item | Default | Notes |
|---|---|---|
| Layout | 左侧搜索 / 筛选，右侧操作 | 允许换行，不硬挤一行 |
| Gap | `gap-3` | 控件之间统一节奏 |
| Search Field | 左 icon + 输入框，常用宽度 220px-320px | 不省略搜索 icon |
| Control Height | 36px / `h-9` | Input / Select / DatePicker / Button 对齐 |
| Action Group | 刷新、导出、主操作成组 | 主操作放最右或最显著位置 |
| Placement | 通常位于 PageHeader 下、数据区上 | 不塞进表格 body 或卡片内容中部 |

### 3.1 子元素字号 / 色 token（改容器必贯彻，P3）

> ⚠️ 改工具条布局 / 间距时**必须同时核对控件子元素走语义 token**，不得只对齐外层 flex、内部文字 / 占位仍散写旧色：
>
> | 子元素 | 字号 / 高度 | 颜色 token |
> |---|---|---|
> | 控件高度 | `h-9`（36px），Input / Select / DatePicker / Button 对齐 | — |
> | 输入 / 控件文字 | `text-sm`（14px） | `var(--cp-text-title)` |
> | Placeholder / 弱态按钮文字 | `text-sm`（14px） | `var(--cp-text-muted)` |
> | 左侧 Search icon | 16px | `var(--cp-text-weak)` |
> | 控件描边 | — | `var(--cp-border)` |
> | 已选条件 chips「清除全部」 | `text-xs`（12px） | `var(--cp-text-muted)`，hover `var(--cp-text-title)` |
>
> 一律走 token，详见本页 §7 fallback 实现；不散写 `text-[#xxx]` / `text-gray-*` / 边框 hex（如 `#355EF1`）。

## 4. Anatomy

```text
SearchFilterBar
  Query optional
  Filters optional
  Chips optional
  SecondaryActions optional
  Refresh optional
  PrimaryAction optional
```

## 5. States

- default: 搜索、筛选、操作都存在，左右分组清楚。
- wrapped: 空间不足时自动换行，但搜索和筛选仍保持同组。
- with-active-filters: 有已选条件时，控件顺序和间距不跳变。
- loading-refresh: 刷新按钮可转 loading，但不锁死整条工具条。
- compact: 抽屉或窄容器里可减少控件数量，但结构不改。

## 6. Demo Repo Usage

- 典型管理端页：`client/src/pages/admin/AuditLog.tsx`
- 典型复杂列表：`client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx`
- 典型筛选组：`client/src/pages/admin/Security/AIAgent/Groups/BashPolicy/BashPolicyList.tsx`
- 典型组合筛选：`client/src/pages/admin/OpenClawMonitor.tsx`

```tsx
<div className="mb-4 flex flex-wrap items-center justify-between gap-3">
  <div className="flex min-w-0 flex-1 flex-wrap items-center gap-3">
    <div className="relative min-w-48 max-w-xs flex-1">
      <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--cp-text-weak)]" />
      <Input className="pl-9 bg-[var(--cp-surface)]" placeholder="搜索关键字" />
    </div>
    <Select />
    <DatePicker placeholder="开始日期" />
  </div>
  <div className="flex shrink-0 items-center gap-2">
    <Button variant="claw-outline" size="icon-sm">刷新</Button>
    <Button variant="claw-primary" size="claw">创建</Button>
  </div>
</div>
```

## 7. Portable Fallback

### 7.1 If host repo already has list toolbar / filters container

- 允许继续复用宿主仓的 Toolbar、FilterBar、TableHeaderActions 之类的现有容器。
- 但必须保留“搜索和筛选一组、动作一组”的结构，不要把刷新、导出、主操作散落到不同角落。
- 宿主仓已有 Select / DatePicker 时，优先复用逻辑，只对齐高度、边框、间距和排列方式。

### 7.2 Minimal React fallback

```tsx
export function PortableSearchFilterBar() {
  return (
    <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-3">
        <div className="relative min-w-[220px] max-w-xs flex-1">
          <svg aria-hidden="true" viewBox="0 0 16 16" className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--cp-text-weak)]" fill="none">
            <path d="M11.5 11.5 14 14" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
            <circle cx="7" cy="7" r="4.5" stroke="currentColor" strokeWidth="1.5" />
          </svg>
          <input className="h-9 w-full rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] pl-9 pr-3 text-sm text-[var(--cp-text-title)]" placeholder="搜索关键字" />
        </div>
        <button className="inline-flex h-9 items-center rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-sm text-[var(--cp-text-title)]">全部状态</button>
        <button className="inline-flex h-9 items-center rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-sm text-[var(--cp-text-muted)]">开始日期</button>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <button className="inline-flex h-9 items-center rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-sm text-[var(--cp-text-title)]">刷新</button>
        <button className="inline-flex h-9 items-center rounded-[4px] bg-[var(--cp-brand-black)] px-6 text-sm text-white">创建</button>
      </div>
    </div>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<div class="cp-filter-bar">
  <div class="cp-filter-bar-main">
    <label class="cp-search-field">
      <span class="cp-search-icon" aria-hidden="true">
        <svg viewBox="0 0 16 16" fill="none">
          <path d="M11.5 11.5 14 14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"></path>
          <circle cx="7" cy="7" r="4.5" stroke="currentColor" stroke-width="1.5"></circle>
        </svg>
      </span>
      <input type="text" placeholder="搜索关键字" />
    </label>
    <button class="cp-filter-btn">全部状态</button>
    <button class="cp-filter-btn cp-filter-btn-muted">开始日期</button>
  </div>
  <div class="cp-filter-bar-actions">
    <button class="cp-filter-btn">刷新</button>
    <button class="cp-filter-btn cp-filter-btn-primary">创建</button>
  </div>
</div>
```

```css
.cp-filter-bar { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 16px; }
.cp-filter-bar-main { display: flex; flex: 1 1 320px; flex-wrap: wrap; align-items: center; gap: 12px; min-width: 0; }
.cp-filter-bar-actions { display: flex; align-items: center; gap: 8px; }
.cp-search-field { position: relative; flex: 1 1 220px; max-width: 320px; }
.cp-search-field input { width: 100%; height: 36px; border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 0 12px 0 36px; font-size: 14px; color: var(--cp-text-title); }
.cp-search-icon { position: absolute; left: 12px; top: 50%; width: 16px; height: 16px; transform: translateY(-50%); color: var(--cp-text-weak); }
.cp-search-icon svg { width: 100%; height: 100%; }
.cp-filter-btn { display: inline-flex; align-items: center; height: 36px; border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 0 12px; font-size: 14px; color: var(--cp-text-title); }
.cp-filter-btn-muted { color: var(--cp-text-muted); }
.cp-filter-btn-primary { border-color: var(--cp-text-title); background: var(--cp-brand-black); color: white; padding: 0 24px; }
```

## 8. Migration Rules

- 旧写法：每个搜索框、筛选器、刷新按钮各自占一块，页面靠 margin 硬凑。
- 新口径：把列表顶部控制区视为一个完整 toolbar，先对齐分组结构，再映射到宿主仓组件。
- 可以暂时兼容：宿主仓沿用旧 Select / DatePicker / SearchableSelect 逻辑。
- 不允许新增：搜索区塞进表格 body；主操作漂到卡片内部；每个控件继续手写不同高度和间距。

## 9. Do / Don't

Do:

- 把搜索、筛选条件放在同一组里。
- 把刷新、导出、主操作放在右侧动作组里。
- 用 `flex-wrap` 处理窄宽度，不要压坏输入框和按钮。

Don't:

- 不要每个控件单独写 margin 来凑布局。
- 不要把刷新按钮放到表格 footer 或标题正文里。
- 不要让主操作和搜索框争同一视觉优先级。

## 10. QA Checklist

- [ ] 搜索组和动作组关系清楚
- [ ] Input / Select / DatePicker / Button 高度对齐
- [ ] 换行后结构仍清楚，没有挤压错位
- [ ] 刷新和主操作没有漂到数据区内部
- [ ] 宿主仓 fallback 可执行

## 11. References

- Demo page: `client/src/pages/admin/AuditLog.tsx`
- Demo page: `client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx`
- Demo page: `client/src/pages/admin/Security/AIAgent/Groups/BashPolicy/BashPolicyList.tsx`
- Demo page: `client/src/pages/admin/OpenClawMonitor.tsx`

## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 / form-controls.md §12 同口径。SearchFilterBar 5 项高频误用 → ClawPro 正确写法。

### 12.1 搜索 + 操作分组：左搜索 / 右动作，不要散落

```tsx
// ❌ 主操作"创建"夹在搜索框中间
<div className="flex gap-3 mb-4">
  <Input className="w-48" placeholder="搜索" />
  <Button variant="claw-primary">创建</Button>
  <Select className="w-32" />
  <DatePicker />
  <Button variant="claw-outline">刷新</Button>
</div>

// ✅ 左搜索筛选组 + 右动作组（justify-between）
<div className="mb-4 flex flex-wrap items-center justify-between gap-3">
  <div className="flex min-w-0 flex-1 flex-wrap items-center gap-3">
    <SearchInput placeholder="搜索关键字" className="min-w-48 max-w-xs flex-1" />
    <Select className="w-32" />
    <DatePicker className="w-36" />
  </div>
  <div className="flex shrink-0 items-center gap-2">
    <Button variant="claw-outline" size="icon-sm"><RefreshCw /></Button>
    <Button variant="claw-primary" size="claw">创建</Button>
  </div>
</div>
```

### 12.2 搜索宽度 220-320px，不要硬撑全宽

```tsx
// ❌ 搜索框 w-full 占满整行，与右侧操作错位
<div className="flex justify-between gap-3 mb-4">
  <Input className="w-full" placeholder="搜索" />
  <Button>创建</Button>
</div>

// ❌ 搜索框 w-32 太窄，placeholder 都被截断
<Input className="w-32" placeholder="搜索操作人或操作事件" />

// ✅ flex-1 + min/max 兜底（小屏 220px / 大屏 320px）
<div className="relative min-w-48 max-w-xs flex-1">
  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--cp-text-weak)]" />
  <Input className="pl-9" placeholder="搜索操作人或操作事件" />
</div>
```

### 12.3 主操作不要塞进 PageHeader 又塞进 FilterBar

```tsx
// ❌ "创建"同时在 PageHeader 和 FilterBar 出现两次
<AdminPageHeader
  title="实例管理"
  actions={<Button variant="claw-primary" size="claw">创建实例</Button>}
/>
<div className="flex justify-between gap-3 mb-4">
  <Input placeholder="搜索" />
  <Button variant="claw-primary" size="claw">创建实例</Button>  {/* 重复 */}
</div>

// ✅ 主操作只放 PageHeader；FilterBar 只保留次级动作
<AdminPageHeader
  title="实例管理"
  actions={<Button variant="claw-primary" size="claw">创建实例</Button>}
/>
<div className="mb-4 flex flex-wrap items-center justify-between gap-3">
  <SearchInput placeholder="搜索 Agent 名称" />
  <div className="flex items-center gap-2">
    <Button variant="claw-outline" size="icon-sm"><RefreshCw /></Button>
    <Button variant="claw-outline" size="claw">导出</Button>
  </div>
</div>
```

### 12.4 不要把 FilterBar 塞进表格 body 内

```tsx
// ❌ 把搜索条嵌到 thead 里，与表头粘连
<table>
  <thead>
    <tr>
      <td colSpan={5}>
        <Input placeholder="搜索" />
      </td>
    </tr>
    <tr>
      <th>名称</th>
      <th>状态</th>
    </tr>
  </thead>
</table>

// ✅ FilterBar 在 PageHeader 下、表格上的同级位置
<AdminPageHeader title="审计日志" />
<div className="mb-4 flex flex-wrap items-center justify-between gap-3">
  <SearchInput placeholder="搜索操作人" />
  <div className="flex items-center gap-2">
    <Button variant="claw-outline" size="icon-sm"><RefreshCw /></Button>
  </div>
</div>
<DataTable rowKey="id" columns={columns} dataSource={list} />
```

### 12.5 已选筛选条件用 chips 表达，不靠 Select 文字提示

```tsx
// ❌ 把已选筛选条件硬塞 placeholder："已选 3 项"
<Select>
  <SelectTrigger>
    <SelectValue placeholder={`已选 ${selectedRoles.length} 项`} />
  </SelectTrigger>
</Select>

// ✅ 已选条件下方用 chips 区显示，可单独清除
<div className="mb-4 flex flex-wrap items-center justify-between gap-3">
  <SearchInput placeholder="搜索关键字" />
  <Select multiple value={selectedRoles} onChange={setSelectedRoles}>...</Select>
</div>
{selectedRoles.length > 0 && (
  <div className="mb-4 flex flex-wrap items-center gap-2">
    {selectedRoles.map(r => (
      <Tag key={r} closable onClose={() => removeRole(r)}>{r}</Tag>
    ))}
    <button
      className="text-xs text-[var(--cp-text-muted)] hover:text-[var(--cp-text-title)]"
      onClick={() => setSelectedRoles([])}
    >
      清除全部
    </button>
  </div>
)}
```
