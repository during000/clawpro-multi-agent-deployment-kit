# Pagination

## 1. Purpose

- 统一列表底部分页器的尺寸、状态、总数文案和宿主仓落地方式。
- 这类组件在换皮时常被忽略，导致表格主体已经换了，分页还保留旧系统风格。

## 2. Scope

- 适用端：Admin 优先；Tenant 只有需要分页列表时复用
- 必用场景：管理端列表页、表格底部分页、需要页码跳转的长列表
- 不适用场景：无限滚动、纯卡片瀑布流、极短列表

## 3. Visual Standard

| Item | Default | Small | Notes |
|---|---|---|---|
| Font Size | 12px | 12px | 与表格口径一致 |
| Item Height | 28px | 24px | 仅按钮尺寸区分 |
| Radius | 8px / rounded-lg `allow-radius` | 8px / rounded-lg `allow-radius` | 页码按钮和前后页按钮统一 |
| Border | `--border` / `#EAEEF4` | 同体系 | 白底边框按钮使用蓝灰描边 token |
| Active | 白底 + 蓝边 + 蓝字 | 同逻辑 | 不做实心色块 |
| Hover | 弱灰 hover | 同逻辑 | 不跳变 |

## 4. Anatomy

```text
Pagination
  ShowTotal optional
  Prev
  PageItems
  Next
  SizeChanger optional
  QuickJumper optional
```

## 5. States

- default: 正常分页。
- active: 当前页高亮。
- disabled: 首尾页按钮禁用。
- simple: 轻量模式，适合空间紧凑区域。
- small: 尺寸更紧凑，但字号不变。

## 6. Demo Repo Usage

- 当前 demo 仓组件：`client/src/components/ui/pagination.tsx`
- 典型页面：`client/src/pages/admin/AuditLog.tsx`、`client/src/pages/admin/FileManagement.tsx`
- 典型放置方式：表格容器内、Table 外、底部边线上方或下方。

```tsx
import { Pagination } from "@/components/ui/pagination";

<Pagination
  total={245}
  current={1}
  pageSize={10}
  showTotal={(total) => `共 ${total} 条记录`}
  showSizeChanger
/>
```

## 7. Portable Fallback

### 7.1 If host repo already has Pagination

- 保留宿主仓分页逻辑。
- 只要求对齐按钮尺寸、12px 字号、active/hover 状态和总数文案口径。

### 7.2 Minimal React fallback

```tsx
export function PortablePagination() {
  return (
    <nav className="flex items-center gap-2 text-xs">
      <span className="text-[var(--cp-text-muted)]">共 245 条记录</span>
      <div className="flex items-center gap-1">
        <button className="inline-flex h-7 min-w-[28px] items-center justify-center rounded-lg border border-[var(--cp-border)] bg-[var(--cp-surface)] text-[var(--cp-text-muted)]" /* allow-radius: pagination uses 8px buttons */>‹</button>
        <button className="inline-flex h-7 min-w-[28px] items-center justify-center rounded-lg border border-[var(--cp-brand-blue)] bg-[var(--cp-surface)] px-1.5 text-[var(--cp-text-brand)]" /* allow-radius: pagination uses 8px buttons */>1</button>
        <button className="inline-flex h-7 min-w-[28px] items-center justify-center rounded-lg border border-[var(--cp-border)] bg-[var(--cp-surface)] px-1.5 text-[var(--cp-text-title)]" /* allow-radius: pagination uses 8px buttons */>2</button>
        <button className="inline-flex h-7 min-w-[28px] items-center justify-center rounded-lg border border-[var(--cp-border)] bg-[var(--cp-surface)] text-[var(--cp-text-muted)]" /* allow-radius: pagination uses 8px buttons */>›</button>
      </div>
    </nav>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<nav class="cp-pagination">
  <span class="cp-pagination-total">共 245 条记录</span>
  <div class="cp-pagination-items">
    <button>‹</button>
    <button class="active">1</button>
    <button>2</button>
    <button>›</button>
  </div>
</nav>
```

```css
.cp-pagination { display: flex; align-items: center; gap: 8px; font-size: 12px; }
.cp-pagination-total { color: var(--cp-text-muted); }
.cp-pagination-items { display: flex; gap: 4px; }
.cp-pagination-items button { min-width: 28px; height: 28px; border: 1px solid var(--cp-border); border-radius: 8px; background: var(--cp-surface); }
.cp-pagination-items .active { border-color: var(--cp-text-brand); color: #355EF1; }
```

## 8. Migration Rules

- 旧写法：表格主体换了，分页仍沿用宿主仓旧分页皮肤。
- 新口径：分页视为表格整体的一部分，一起换皮。
- 可以暂时兼容：宿主仓已有分页逻辑和交互。
- 不允许新增：大字号分页、实心蓝色页码块、和表格脱节的分页布局。

## 9. Do / Don't

Do:

- 保持分页字号与表格一致。
- 把总数文案、页码、切页按钮当作一个整体。
- 在空间小的浮层里用 small / simple 模式。

Don't:

- 不要让分页看起来像另一个系统。
- 不要只换表格不换分页。
- 不要把页码 active 做成重色块抢视觉。

## 10. QA Checklist

- [ ] 12px 字号口径一致
- [ ] active / disabled / hover 状态完整
- [ ] 与表格容器关系清楚
- [ ] 宿主仓 fallback 可执行

## 11. References

- Demo code: `client/src/components/ui/pagination.tsx`
- Demo page: `client/src/pages/admin/AuditLog.tsx`
- Demo page: `client/src/pages/admin/FileManagement.tsx`


## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 同口径。Pagination 5 项高频误用 → ClawPro 正确写法。

### 12.1 不要把表格主体换皮但保留旧分页

```tsx
// ❌ 表格已切到 DataTable，分页仍用 antd 旧皮
<DataTable rowKey="id" columns={columns} dataSource={list} />
<AntdPagination total={total} current={page} onChange={setPage} className="mt-4" />

// ✅ 用 DataTable pagination prop，分页与表格视觉一致
<DataTable
  rowKey="id"
  columns={columns}
  dataSource={list}
  pagination={{
    current: page,
    pageSize,
    total,
    onChange: setPage,
    showTotal: (t) => `共 ${t} 条记录`,
  }}
/>

// ✅ 独立列表场景：用统一 Pagination 组件
<Pagination total={total} current={page} pageSize={pageSize} onChange={setPage} showTotal={(t) => `共 ${t} 条记录`} />
```

### 12.2 字号 12px 不能放大成 14px / 16px

```tsx
// ❌ 用 text-base 让分页大一截，与表格 12px 节奏脱节
<nav className="flex items-center gap-2 text-base">
  <span>共 245 条</span>
  <button className="h-9 min-w-9 border rounded">‹</button>
  <button className="h-9 min-w-9 border rounded">1</button>
</nav>

// ✅ 12px / h-7 / 28px
<Pagination total={245} current={1} pageSize={10} showTotal={(t) => `共 ${t} 条记录`} />
```

### 12.3 Active 用白底 + 蓝边 + 蓝字，不用实心蓝块

```tsx
// ❌ Active 实心蓝色块抢视觉
<button className="h-7 min-w-[28px] rounded-lg bg-[#355EF1] text-white">1</button>

// ❌ Active 用红色 / 紫色等业务色
<button className="h-7 min-w-[28px] rounded-lg bg-purple-500 text-white">1</button>

// ✅ 白底 + 蓝边 + 蓝字
<button className="
  inline-flex h-7 min-w-[28px] items-center justify-center
  rounded-lg border border-[var(--cp-brand-blue)]
  bg-[var(--cp-surface)] px-1.5 text-[var(--cp-text-brand)]
"  /* allow-radius: pagination uses 8px buttons */>
  1
</button>
```

### 12.4 总数文案口径：showTotal，不要硬编码

```tsx
// ❌ 自己拼总数字符串，多语言 / 单位漂移
<span>{`第 ${page} 页 / 共 ${Math.ceil(total / pageSize)} 页 / ${total} 条`}</span>

// ❌ 把"共 245 条"翻译成"245 results"中英混排
<span>245 results</span>

// ✅ 走 showTotal API，统一文案
<Pagination
  total={total}
  current={page}
  pageSize={pageSize}
  onChange={setPage}
  showTotal={(t) => `共 ${t} 条记录`}
/>
```

### 12.5 紧凑场景用 simple / small，不要塞完整分页

```tsx
// ❌ 弹窗 / 浮层内仍放完整分页（页码 + 跳转 + 切换页大小）
<DialogContent>
  <DataList />
  <Pagination
    total={total}
    current={page}
    showSizeChanger
    showQuickJumper
    onChange={setPage}
  />
</DialogContent>

// ✅ 弹窗 / 浮层用 simple 模式，仅前后页 + 总数
<DialogContent>
  <DataList />
  <Pagination
    mode="simple"
    size="small"
    total={total}
    current={page}
    pageSize={pageSize}
    onChange={setPage}
    showTotal={(t) => `共 ${t} 条`}
  />
</DialogContent>
```
