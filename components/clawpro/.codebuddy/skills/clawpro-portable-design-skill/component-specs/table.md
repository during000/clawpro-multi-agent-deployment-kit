# Table

## 1. Purpose

- 管理端/用户端的结构化数据展示统一组件。
- 重点解决：列对齐、操作列写法、空态、分页、横向滚动固定列、行状态/标签样式一致性。

## 2. Scope

- 适用端：Admin 优先；Tenant 仅在明确需要表格时复用。
- 必用场景：管理端数据列表、操作日志、配置清单、弹窗内选择列表。
- 不适用场景：强视觉营销卡片、移动端优先紧凑信息流、纯说明型内容块。

## 3. Visual Standard

### 3.1 设计令牌

| Item | Value | Notes |
|---|---|---|
| Container | 白底圆角容器 (`rounded-xl border border-[var(--cp-border)]`) | 管理端默认做法 |
| Scope Boundary | 容器只承载表格本体与分页 | 标题 / 描述 / 页面级按钮放表格外 |
| Header Background | `var(--cp-bg-subtle)` = `#FAFBFD` | 灰色表头，sticky 时跟随 |
| Header Text | `12px` / Medium / `var(--cp-text-muted)` (#64748B) | 表头不抢正文 |
| Header Height | `40px` | — |
| Body Text | `12px` / 正文 `var(--cp-text-body)` (#1E293B) | 关键字段可用 `var(--cp-text-emphasis)` |
| Row Height | 最小 `54px`；复杂内容允许自然撑高 | — |
| Row Border | `border-b border-[var(--cp-border)]` | 行与行间的分割线 |
| Row Hover | 弱灰背景 `var(--bg-grey-normal)` = `#FAFBFD` | 不做强色块 |
| Row Selected | **无背景高亮**（选中状态由 Checkbox 勾选态表达） | v2026.06 起移除蓝底 |
| Padding (header & cell) | `px-4` (16px)，cell `py-3` | 左右统一 16px，不因密度变化 |
| Footer | `grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[var(--cp-border)]` | 数量左对齐，分页器右对齐 |

### 3.2 Variant（视觉变体）

| variant | 表头背景 | 适用场景 |
|---|---|---|
| `gray-header`（默认） | `var(--cp-bg-subtle)` 灰色 | 白色背景容器内的表格 |
| `white` | 纯白 | 非白色背景（蓝色渐变/灰色页面底）上的浮起白卡表格 |

> `white` 变体**禁止**在 Dialog/Drawer 等弹窗内使用（白上加白看不见）。

### 3.3 统一口径

- 表头、正文、状态列、操作列、页脚、空态统一使用 `12px`。
- 横向 padding 固定 `px-4`（16px），不要因为列表变密而缩到 `px-2`。
- 行高默认最小 `54px`；复杂内容允许自然撑高。

## 4. Anatomy

```text
Table Region Outside
  SectionTitle / Description / Actions optional
Container (rounded border)
  Table
    TableHeader (灰底 sticky)
      TableRow
        TableHead × n
    TableBody
      TableRow × n (hover / selected)
        TableCell × n
        TableActionCell (操作列)
  Footer (总数 + 分页器)
```

## 4.1 Scope Boundary

- `Table` 组件只负责表格本身：表头、表体、行分割线、空态、加载态、分页。
- 页面级标题、描述、统计文案、主按钮、刷新按钮、更多操作按钮不属于 `Table` 组件职责。
- 如果页面需要“模型列表 / 实例数据 / 审计日志”这类标题区，必须放在表格外层 section，不要做成表格卡头。
- 批量操作条属于表格上下文，可紧贴表格放置；但它也不是表格标题栏。

## 5. 操作列规则（强制）

| 规则 | 说明 |
|---|---|
| 结构 | 操作列用固定容器包裹，内置 `flex items-center gap-6`（间距 24px） |
| 按钮类型 | **文字按钮**（品牌蓝色 `var(--cp-text-brand)`），hover 加下划线 |
| 禁止 | 不用图标按钮、不用实心按钮、不用红色删除按钮 |
| 删除语义 | 删除按钮也统一蓝色文字，危险操作由二次确认弹窗承担 |
| Disabled | 浅蓝灰色，保留色相但大幅降弱，不用透明度 |
| 间距 | 操作项之间固定 24px，由容器统一管控，业务侧不手写 wrapper |

```tsx
// 操作列标准写法
<td className="px-4 py-3">
  <div className="flex items-center gap-6">
    <button className="text-[var(--cp-text-brand)] hover:underline text-xs">编辑</button>
    <button className="text-[var(--cp-text-brand)] hover:underline text-xs">删除</button>
  </div>
</td>
```

## 6. 固定列（Fixed Columns）

适用场景：列数较多需横向滚动，首列（名称）或末列（操作）需常驻可见。

### 核心规则

| 规则 | 说明 |
|---|---|
| 开启方式 | 给 Table 设置最小内容宽度（如 `min-width: 1500px`），开启横向滚动 |
| 固定列实现 | `position: sticky` + `left: 0` / `right: 0` + `z-index` |
| 阴影分隔 | 固定列边界用 `6px` 渐变阴影提示，**不画 1px 硬分隔线** |
| 多列同侧 | 只在最外侧列保留阴影，内部固定列关闭阴影 |
| 表头底色 | 固定列与普通列必须**同色**，不能出现灰白割裂 |
| 单元格底色 | 白底，行 hover 时跟随变色 |
| 滚动条 | 默认隐藏，hover 时出现 |

### 禁止事项

- 禁止手写 `sticky right-0 z-10 bg-white` 模拟固定列
- 禁止手写阴影分隔线
- 禁止在固定列上覆盖背景色
- 禁止在固定列单元格注入 inline `backgroundColor`（会阻断 hover 跟随）

## 7. 表格内状态/标签样式规则（强制）

| 场景 | 写法 | 禁止 |
|---|---|---|
| 状态列（运行状态、下发状态） | 纯文字变色（12px Medium），无底色无圆点 | 禁止在表格内用带底色胶囊标签 |
| 版本号 | 纯文字（如 `v2.1.0`） | 禁止用灰色胶囊包裹 |
| 镜像来源/类型标签 | 拆为独立列，纯文字显示 | 禁止用彩色填充标签 |
| 辅助信息 | 12px 灰文字 | 禁止用标签组件包裹 |

> 总原则：表格行内只允许「状态列」有颜色文字，其余列一律黑白灰纯文字层次。

## 8. 表格页脚（分页器）

```tsx
// 标准写法：数量左对齐，分页器右对齐；不要在这个 footer 上方再塞标题和按钮
<div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[var(--cp-border)]">
  <span className="justify-self-start text-xs leading-[1.5] text-[var(--cp-text-muted)]">
    共 {total} 条
  </span>
  <nav className="justify-self-end">
    {/* 分页器组件 */}
  </nav>
</div>
```

规则：
- 横向 padding 固定 `px-4`（16px），与表头/单元格对齐
- 数量统计固定左对齐，分页器固定右对齐
- 页面级表格用默认尺寸（32px）分页器；弹窗内可用小尺寸（24px）

## 9. 空态规则

- 表格空态一律放在 `<td colSpan>` 内。
- **不用插画**，只用纯文字（默认双行：标题 + 描述）。
- 文字使用 `12px` / `var(--cp-text-weak)`。
- 详见 `component-specs/empty-state.md §5 表格`。

## 10. States

- **default**: 灰色表头 + 白色表体 + 行分隔线。
- **hover**: 行底色轻微抬起。
- **selected**: 无背景高亮，选中状态由 Checkbox 勾选态表达。
- **loading**: 骨架行或局部 loading，不锁整页。
- **empty**: 表体内 `td colSpan` 纯文字居中。
- **error**: 表格外或表体内给出错误原因与重试入口。
- **overflow-x**: 横向滚动，必要时固定首列和操作列。

## 11. Demo Repo Usage

```tsx
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  TableActionCell,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Pagination } from "@/components/ui/pagination";

<div className="bg-white rounded-xl border border-[var(--cp-border)] overflow-hidden">
  <Table>
    <TableHeader>
      <TableRow>
        <TableHead>名称</TableHead>
        <TableHead>状态</TableHead>
        <TableHead className="text-right">数量</TableHead>
        <TableHead>操作</TableHead>
      </TableRow>
    </TableHeader>
    <TableBody>
      {data.map((item) => (
        <TableRow key={item.id}>
          <TableCell className="font-medium">{item.name}</TableCell>
          <TableCell>{item.status}</TableCell>
          <TableCell className="text-right tabular-nums">{item.count}</TableCell>
          <TableActionCell>
            <Button variant="link">编辑</Button>
            <Button variant="link">删除</Button>
          </TableActionCell>
        </TableRow>
      ))}
    </TableBody>
  </Table>
  <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[var(--cp-border)]">
    <span className="justify-self-start text-xs leading-[1.5] text-[#737373]">共 {total} 条</span>
    <Pagination total={total} current={page} pageSize={10} className="justify-self-end" onChange={setPage} />
  </div>
</div>
```

## 12. Portable Fallback

### 12.1 If host repo already has a Table

- 可复用宿主仓表格结构。
- 但必须覆盖：表头灰底 + 12px 文字密度 + 操作列品牌蓝文字按钮 24px 间距 + 表体内空态 + 分页器两端对齐。
- 如果宿主仓有成熟 data table，不要求迁成 demo API，只要求视觉和状态对齐。

### 12.2 Minimal React fallback

```tsx
function PortableTable({ columns, data, total, page, onPageChange }: any) {
  return (
    <section className="overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-white">
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-xs text-[var(--cp-text-body)]">
          <thead>
            <tr className="bg-[var(--cp-bg-subtle)] border-b border-[var(--cp-border)]">
              {columns.map((col: any) => (
                <th key={col.key} className="h-10 px-4 text-left text-xs font-medium text-[var(--cp-text-muted)]">
                  {col.title}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.length === 0 ? (
              <tr>
                <td colSpan={columns.length}>
                  <div className="text-center py-12 space-y-1">
                    <p className="text-xs text-[var(--cp-text-weak)]">暂无记录</p>
                    <p className="text-xs text-[var(--cp-text-weak)]">尝试调整筛选条件，或新建一条记录</p>
                  </div>
                </td>
              </tr>
            ) : (
              data.map((row: any, i: number) => (
                <tr key={i} className="border-b border-[var(--cp-border)] hover:bg-[var(--bg-grey-normal)]">
                  {columns.map((col: any) => (
                    <td key={col.key} className="h-[54px] px-4 py-3">{row[col.key]}</td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      {total > 0 && (
        <div className="grid grid-cols-[1fr_auto] items-center gap-4 px-4 py-2 border-t border-[var(--cp-border)]">
          <span className="text-xs text-[var(--cp-text-muted)]">共 {total} 条</span>
          {/* 宿主仓分页器 */}
        </div>
      )}
    </section>
  );
}
```

### 12.3 操作列 Portable Fallback

```tsx
// 操作列固定写法
<td className="h-[54px] px-4 py-3">
  <div className="flex items-center gap-6 whitespace-nowrap">
    <button className="text-xs text-[var(--cp-text-brand)] hover:underline">编辑</button>
    <button className="text-xs text-[var(--cp-text-brand)] hover:underline">删除</button>
    <button className="text-xs text-[#d0dafa] cursor-not-allowed" disabled>禁用态</button>
  </div>
</td>
```

### 12.4 HTML/CSS fallback

见 `portable/html-css/table.html`。

## 13. Migration Rules

### 旧写法 → 新写法

| 旧写法 | 新写法 |
|---|---|
| 原生 `<table>` + 手写 `bg-gray-50` 表头 + `text-xs uppercase tracking-wide` | 统一表头灰底 + 12px Medium + 无大写 |
| 操作列用红色/灰色/黑色文字区分危险级别 | 全部品牌蓝文字按钮，危险由二次确认承担 |
| 表格空态另包大卡片或放大插画 | 空态放入 `<td colSpan>` 内纯文字 |
| 选中行加蓝色背景高亮 | 无背景高亮，选中用 Checkbox 表达 |
| 表头用 `text-gray-500 uppercase` | `12px Medium #737373`，不大写 |
| 表格页脚只放分页器、总数居中 | 总数左对齐 + 分页器右对齐（grid 两端布局） |
| 为省空间把左右 padding 缩到 `px-2` | 横向 padding 仍保持 `px-4` (16px) |
| 手写 `sticky right-0 z-10 bg-white` 做固定列 | 使用组件级 fixed 属性，背景/阴影由组件管控 |

### 不允许新增

- 表格空态用大插画或另包卡片
- 操作列使用红色文字按钮
- 自定义表头背景色
- 自定义行 hover 色
- 缩减横向 padding

## 14. Do / Don't

**Do:**

- 表格整体使用 12px 口径（单元格正文和表头统一）。
- 操作列统一品牌蓝文字按钮 + 24px 间距。
- 空态放在表体内 `<td colSpan>` 纯文字。
- 选中状态用 Checkbox 表达，不给行加底色。
- 列多时用横向滚动 + 固定列。
- 表头行禁止 hover 变色。
- 页脚数量左对齐、分页器右对齐。

**Don't:**

- 不要用原生 `<table>` + 临时 class 拼装。
- 不要在操作列混入多种颜色表达危险级别。
- 不要给表格空态包大卡片或放插画。
- 不要手写 sticky + bg-white 模拟固定列。
- 不要给选中行加蓝底高亮。
- 不要缩减横向 padding。
- 不要用 `variant="white"` 在白色容器/弹窗内。
- 不要覆盖固定列的背景色。

## 15. QA Checklist

- [ ] 表头、表体、分页是同一个模块
- [ ] 表格字号统一 12px 口径
- [ ] 操作列使用品牌蓝文字按钮，间距 24px
- [ ] 操作列删除按钮不用红色
- [ ] 空态在表体内 `<td colSpan>` 纯文字，无插画
- [ ] 选中行无背景高亮
- [ ] 表头行不 hover 变色
- [ ] 页脚：总数左 + 分页器右
- [ ] 横向 padding 保持 `px-4`
- [ ] 固定列无手写 sticky，无灰白割裂
- [ ] 表格内状态列只用纯文字变色，不用带底色标签
- [ ] fallback 使用 `var(--cp-*)` CSS variable
- [ ] 跨仓 fallback 可独立落地

## 16. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`
- Related tokens: `tokens/colors.md` (`--cp-text-body`, `--cp-text-muted`, `--cp-text-weak`, `--cp-text-brand`, `--cp-border`, `--cp-bg-subtle`)
- Related specs: `component-specs/empty-state.md`, `component-specs/pagination.md`, `component-specs/batch-actions-bar.md`
- Related recipes: `references/page-recipes.md`

## 17. 代码对照（✅/❌）

> 与 SKILL.md §2 / §7 同口径。表格 5 项高频误用 → ClawPro 正确写法。

### 17.1 操作列：纯蓝色文字按钮 + 24px 间距，禁红色

```tsx
// ❌ 用红色文字暗示"删除"危险级别
<TableActionCell>
  <Button variant="link" className="text-red-600">删除</Button>
  <Button variant="link">编辑</Button>
</TableActionCell>

// ❌ 操作列改用图标按钮，丢失文字语义
<TableActionCell>
  <button><Edit className="w-4 h-4" /></button>
  <button><Trash className="w-4 h-4 text-red-500" /></button>
</TableActionCell>

// ✅ 全部品牌蓝文字按钮，危险由二次确认弹窗承担；间距 24px 由容器统一
<TableActionCell>
  <Button variant="link">编辑</Button>
  <Button variant="link">删除</Button>
</TableActionCell>
```

### 17.2 表格内状态/标签：纯文字变色，禁用胶囊

```tsx
// ❌ 表格行内状态用带底色胶囊
<TableCell>
  <span className="px-2 py-0.5 rounded-full bg-green-100 text-green-700 text-xs">运行中</span>
</TableCell>

// ❌ 版本号包灰色胶囊
<TableCell>
  <span className="px-2 py-0.5 rounded bg-gray-100 text-gray-600">v2.1.0</span>
</TableCell>

// ✅ 状态列纯文字变色（12px Medium）
<TableCell>
  <span className="text-xs font-medium text-[var(--text-success)]">运行中</span>
</TableCell>

// ✅ 版本号 / 镜像来源等其余列：黑白灰纯文字
<TableCell className="text-[var(--cp-text-body)]">v2.1.0</TableCell>
```

### 17.3 空态：colSpan 内纯文字，禁插画/外包大卡

```tsx
// ❌ 空态另包大卡片 + 大插画
{rows.length === 0 && (
  <SurfaceCard className="py-20 text-center">
    <img src="/empty.svg" className="mx-auto" />
    <h3 className="font-bold text-base">暂无数据</h3>
  </SurfaceCard>
)}

// ✅ 表体内 <td colSpan> 纯文字双行
<TableBody>
  {rows.length === 0 ? (
    <TableRow>
      <TableCell colSpan={columns.length}>
        <div className="text-center py-12 space-y-1">
          <p className="text-xs text-[var(--cp-text-weak)]">暂无策略</p>
          <p className="text-xs text-[var(--cp-text-weak)]">尝试调整筛选条件，或新建一条策略</p>
        </div>
      </TableCell>
    </TableRow>
  ) : rows.map(...)}
</TableBody>
```

### 17.4 选中行：无背景高亮（v2026.06）

```tsx
// ❌ 选中行加蓝色背景，与表格 hover 灰色冲突
<TableRow className={cn(selected && "bg-[var(--cp-brand-blue)]/10")}>
  <TableCell><Checkbox checked={selected} /></TableCell>
  ...
</TableRow>

// ✅ 选中态由 Checkbox 勾选态表达，行背景不变
<TableRow>
  <TableCell><Checkbox checked={selected} /></TableCell>
  ...
</TableRow>
```

### 17.5 固定列：用组件级 fixed，不要手写 sticky

```tsx
// ❌ 手写 sticky + bg-white + 1px 硬分隔线
<th className="sticky left-0 z-10 bg-white border-r border-gray-200">名称</th>
<td className="sticky right-0 z-10 bg-white border-l border-gray-200">
  <Button variant="link">编辑</Button>
</td>

// ✅ 用 columns API 声明 fixed，背景 / 阴影 / hover 跟随由组件管控
const columns: ColumnDef<Policy>[] = [
  { key: "name", title: "名称", fixed: "left" },
  { key: "actions", title: "操作", align: "right", fixed: "right",
    render: (_, row) => <PolicyActions row={row} /> },
];
<DataTable rowKey="id" columns={columns} dataSource={list} autoFixedColumns />
```
