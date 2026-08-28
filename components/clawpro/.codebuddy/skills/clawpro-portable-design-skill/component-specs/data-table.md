# DataTable

> 数据驱动壳子。`Table`（组合式底座）之上的"标准列表页"封装，对齐 Ant Design 风格的 `columns + dataSource` API，把 7 个状态机一次性内聚，杜绝"每新建一页都漏接 1～2 个状态"的 bug 模式。

## 1. Purpose

- 让"管理端列表页"成为**一行声明式调用**，业务无需自己维护选择 / 排序 / 筛选 / 分页 / 空态 / 加载 / rowKey 这 7 个状态。
- 不替代 `Table`：底层仍是 `Table / TableRow / TableCell`，`DataTable` 内部组合它们，并继承 clawpro 的所有视觉资产（12px / `autoFixedColumns` / 容器级阴影 / 固定列阴影 / `--bg-grey-normal` 选中态）。
- 与 `Table` 的关系是**双层**：业务 90% 列表页用 `DataTable`，特殊布局（嵌套表 / 自定义合并 / 设计稿强定制）保留 `Table` 逃生口。
- `DataTable` 只封装数据表壳，不承载区块标题、描述和页面级操作按钮；这些内容必须由外层页面骨架负责。

## 2. Scope

- 适用端：Admin 强制（新建列表页只允许用 `DataTable`），Tenant 复用列表能力时同样适用。
- 必用场景：
  - 管理端 CRUD 列表（资产 / 策略 / 用户 / 日志）
  - 带分页 / 选择 / 排序 / 筛选的标准数据列表
  - 弹窗内的对象选择列表（与 `Transfer` 互斥：穿梭走 `Transfer`，单面板选走 `DataTable`）
- 不适用场景（必须回落到底层 `Table`）：
  - 嵌套子表（行内展开后是另一张表，且内外列结构不一致）
  - 自定义 `colSpan` 跨列合并 / 表头分组（`ColumnGroup`）
  - 设计稿要求高度定制的视觉单元格（行内编辑、单元格内嵌 mini 图表等）
  - 任何**不是"行 × 列 × 数据"形态**的展示

## 3. Visual Standard

继承 `table.md` §3 全部口径（容器 / 表头底色 / 12px / 行高 54px / 边框 token / hover / selected）。**`DataTable` 不允许引入任何新视觉变量**，所有视觉差异通过透传到底层 `Table` 的 `variant` 实现。

| Item | Value | Notes |
|---|---|---|
| 行高 | 同 `table.md`：`default` 54px / `compact` 40px | 通过 `size` prop |
| 表头底色 | `gray-header` 默认 | 通过 `variant` prop 透传 |
| 选中态 | 整行 `--bg-grey-normal` | `rowSelection` 启用时自动；hover 时不再抬成蓝底 |
| Loading mask | 整表半透明 + 居中 spinner，**不锁页** | 行数 ≥ 1 时叠在 tbody 上，并覆盖左右固定列；空态时合并到空态视图 |
| Fixed Shadow | 左固定列向右投影；右固定列向左投影 | 仅在对应方向存在横向滚动时出现，避免方向反转 |
| 空态 | 表体内 `colSpan` 渲染 `EmptyState size="sm"` | `dataSource.length===0` 时自动 |
| 分页区 | `border-t` 内嵌于同一容器 | `<TableShell>` 内部处理，业务无需手挂 `border-t` |
| 标题区 | 不属于 `DataTable` | 标题 / 描述 / 操作按钮放在外层 section |

## 4. Anatomy

```text
SurfaceCard
  TableShell                      ← DataTable 内部用，业务感知不到
    Table (variant)
      TableHeader
        TableRow
          [SelectAllHead]         ← rowSelection 启用时自动
          [SortHead | FilterHead] ← column.sorter / column.filters 启用时自动
          TableHead * n
      TableBody
        TableRow * n              ← rowKey 内部取值
          [SelectCell]
          TableCell * n
        [EmptyRow]                ← dataSource 为空时
        [LoadingMask]             ← loading=true 时
    [PaginationFooter]            ← pagination 启用时
```

## 5. API

### 5.1 顶层 props

| Prop | Type | Default | Notes |
|---|---|---|---|
| `columns` | `ColumnDef<T>[]` | — | 必填。见 §5.2 |
| `dataSource` | `T[]` | — | 必填 |
| `rowKey` | `string \| (record: T) => string` | — | **必填**。强约束，避免 React key 散在各处 |
| `loading` | `boolean` | `false` | 整表 mask；不影响分页与表头 |
| `pagination` | `false \| PaginationProps` | `false` | `false` 关闭分页；传对象时内部渲染 `<Pagination>` 在 `border-t` 下 |
| `rowSelection` | `RowSelectionProps<T>` | — | 见 §5.3 |
| `size` | `"default" \| "compact"` | `"default"` | 透传到 `Table` |
| `variant` | `"white" \| "gray-header"` | `"gray-header"` | 透传到 `Table` |
| `emptyText` | `ReactNode` | 默认 `EmptyState` | 仅替换文案/插图，不破壳 |
| `onRow` | `(record, index) => HTMLProps` | — | 行级事件 / 类名挂载点（与 Ant 一致） |
| `rowClassName` | `string \| ((record, index) => string)` | — | 行级类名 |
| `autoFixedColumns` | `boolean` | `true` | 透传，保留 clawpro 自动钉首列 / 操作列能力 |

### 5.2 `ColumnDef<T>`

| Field | Type | Notes |
|---|---|---|
| `key` | `string` | 必填，唯一 |
| `title` | `ReactNode` | 表头内容 |
| `dataIndex` | `keyof T` | 取值字段；与 `render` 二选一即可 |
| `render` | `(value, record, index) => ReactNode` | 自定义渲染 |
| `width` | `number \| string` | 列宽 |
| `align` | `"left" \| "center" \| "right"` | 默认 `left`，操作列建议 `right` |
| `fixed` | `"left" \| "right"` | 配合 `autoFixedColumns` 一般无需手设 |
| `sorter` | `boolean \| (a, b) => number` | `true` 走受控、函数走非受控 |
| `sortOrder` | `"asc" \| "desc" \| null` | 受控用 |
| `filters` | `{ label, value }[]` | 启用列头 Popover 筛选 |
| `filteredValue` | `unknown[] \| null` | 受控 |
| `filterDropdown` | `(props) => ReactNode` | 自定义筛选浮层（fallback 到 `FilterChip` 视觉） |
| `ellipsis` | `boolean` | 溢出截断 + Tooltip |
| `className` | `string` | 单元格类名 |

### 5.3 `RowSelectionProps<T>`

| Field | Type | Notes |
|---|---|---|
| `selectedKeys` | `string[]` | 受控 |
| `onChange` | `(keys: string[], rows: T[]) => void` | 必填 |
| `preserveSelectedKeys` | `boolean` | 默认 `true`，跨页保留 |
| `getCheckboxProps` | `(record) => { disabled?: boolean }` | 行级禁用控制 |
| `type` | `"checkbox" \| "radio"` | 默认 `checkbox` |

### 5.4 整合回调

```ts
onChange?: (params: {
  pagination: { current: number; pageSize: number };
  sorter: { columnKey: string; order: "asc" | "desc" } | null;
  filters: Record<string, unknown[]>;
}) => void;
```

> 与 Ant 的 `onChange(pagination, filters, sorter, extra)` 等价，参数改成对象，便于后端 API 直接 spread。

## 6. Demo Repo Usage

### 5.5 Scope Boundary

- `DataTable` 负责：列头、行、选择、排序、筛选、空态、loading、分页。
- `DataTable` 不负责：区块标题、辅助描述、刷新按钮、创建按钮、右上角操作区。
- 如果设计稿里标题和表格相邻，做法应是“外层 section + 标题区 + DataTable”，不是给 `DataTable` 额外包一层 header slot。

### 6.1 最小用法

```tsx
import { DataTable } from "@/components/ui/table";
import { SurfaceCard } from "@/components/ui/Surface";

<SurfaceCard className="overflow-hidden">
  <DataTable<Policy>
    rowKey="id"
    columns={[
      { key: "name", title: "策略名称", dataIndex: "name" },
      { key: "status", title: "状态", dataIndex: "status", filters: STATUS_FILTERS },
      { key: "updatedAt", title: "更新时间", dataIndex: "updatedAt", sorter: true },
      {
        key: "actions",
        title: "操作",
        align: "right",
        render: (_, row) => <PolicyActions row={row} />,
      },
    ]}
    dataSource={list}
    loading={loading}
    pagination={{ current, pageSize, total, onChange: setPage }}
    rowSelection={{ selectedKeys, onChange: setSelected }}
    onChange={({ sorter, filters }) => fetch({ sorter, filters })}
  />
</SurfaceCard>
```

### 6.2 何时回落到底层 `Table`

| 场景 | 选择 |
|---|---|
| 标准 CRUD 列表 + 分页 + 选择 | `DataTable` |
| 弹窗内对象单选/多选 | `DataTable` |
| 行展开后是结构相同的子表 | `DataTable` + `expandable`（v2 增强） |
| 行展开后是**完全不同的视图**（详情卡 / 表单 / 子表头不一致） | 底层 `Table` + `TableExpandedRow` |
| 表头分组 / `colSpan` 合并 | 底层 `Table` |
| 单元格内行内编辑 | 底层 `Table` + 业务自治 |
| 设计稿要求 mini 图表 / 复合卡片单元格 | 底层 `Table`，但视觉仍按 `table.md` |

## 7. Portable Fallback

### 7.1 If host repo already has a data table

- 宿主仓如已有 Ant `<Table>` / 自研数据驱动表：**不要求迁成 demo 仓 `DataTable` API**。
- 必须对齐：12px 文字、灰头白身、行高 54、空态在表体内、选中态用 `--bg-grey-normal`、分页与表格在同一卡片内（`border-t`）。
- 严禁：把 Ant `<Table>` 的默认 `bordered` 竖线打开、把表头改成深色块、把操作列染红字。

### 7.2 Minimal React fallback（无 `DataTable` 时）

直接退回到 `table.md` §7.2 的 `PortableAdminTable`，业务侧手动维护 7 个状态。**这是承认劣化**，仅在宿主仓没有任何数据驱动表时使用。

```tsx
// 见 component-specs/table.md §7.2
```

### 7.3 兼容 Ant Design 宿主仓（推荐路径）

如果宿主仓用 Ant，**直接用 `<antd.Table>`** + 以下配置即可视觉对齐：

```tsx
<ConfigProvider
  theme={{
    components: {
      Table: {
        headerBg: "var(--cp-bg-subtle)",
        headerColor: "var(--cp-text-muted)",
        rowSelectedBg: "var(--bg-grey-normal)",
        rowSelectedHoverBg: "var(--bg-grey-normal)",
        borderColor: "var(--cp-border)",
        fontSize: 12,
      },
    },
  }}
>
  <Table bordered={false} size="middle" pagination={{ ... }} />
</ConfigProvider>
```

## 8. Migration Rules

### 8.1 旧写法 → 新写法

旧（业务自己组合）：

```tsx
<Table>
  <TableHeader>
    <TableRow>
      <TableHead><Checkbox checked={allSelected} onChange={...} /></TableHead>
      <TableHead onClick={() => toggleSort("name")}>名称 {sortIcon}</TableHead>
      ...
    </TableRow>
  </TableHeader>
  <TableBody>
    {list.map(row => (
      <TableRow key={row.id}>
        <TableCell><Checkbox checked={selected.includes(row.id)} ... /></TableCell>
        ...
      </TableRow>
    ))}
    {list.length === 0 && <EmptyRow colSpan={5} />}
  </TableBody>
</Table>
<div className="border-t px-4 py-2"><Pagination ... /></div>
```

新：

```tsx
<DataTable
  rowKey="id"
  columns={columns}
  dataSource={list}
  rowSelection={{ selectedKeys, onChange: setSelected }}
  pagination={paginationProps}
  onChange={({ sorter }) => refetch({ sorter })}
/>
```

### 8.2 强约束

- **新建列表页只允许用 `DataTable`**：写到 `references/admin.md` 与 `qa/admin-checklist.md`。
- **老页不强迁**：按业务节奏，bug 出现时再迁。
- **逃生口需在 PR description 里声明理由**：使用底层 `Table` 而非 `DataTable` 时，PR 必须写明属于 §6.2 哪一种特殊场景。

### 8.3 不允许新增

- 在 `DataTable` 之外再造一个数据驱动表壳。
- 在业务侧自己 `useState<string[]>` 维护 selectedKeys（应交给 `rowSelection`）。
- 在业务侧自己拼 `<Pagination>` + `border-t`（应交给 `pagination` prop）。
- 在业务侧自己写"`list.length===0` 渲染 `<td colSpan>`"（应交给内部空态）。

## 9. Do / Don't

Do:

- 永远传 `rowKey`，且优先传 string 字面量字段名。
- 把所有列状态收敛到 `columns` 声明里，不要在外面散写排序按钮。
- 整表 loading 用 `loading` prop，不要在外层包 spinner。
- 跨页保留默认开（`preserveSelectedKeys: true`），除非业务明确不需要。

Don't:

- 不要把 `DataTable` 当作"万能表格"塞进表头分组 / 行内编辑 / mini 图表场景，那是底层 `Table` 的事。
- 不要在 `render` 里再起一个完整的子表（嵌套表用 `expandable`，结构差异大就回落到底层）。
- 不要绕过 `rowSelection` 自己接 checkbox（视觉与状态会双轨漂移）。
- 不要在 `DataTable` 外层再包 `border-t` 拼分页（pagination 已内置）。

## 10. QA Checklist

- [ ] `rowKey` 已传，且类型正确
- [ ] `columns` 中所有 `key` 唯一
- [ ] 分页通过 `pagination` prop，业务未在外层手挂 `border-t`
- [ ] 选择通过 `rowSelection`，业务未自己维护 `selectedKeys` 数组
- [ ] 排序 / 筛选通过 `column.sorter` / `column.filters`，业务未在表头手画三角箭头
- [ ] `dataSource.length === 0` 时表体内自动渲染 `EmptyState`，无外层包大卡片
- [ ] `loading` 切换不引起表头闪动 / 不锁住分页交互，且遮罩覆盖固定列
- [ ] 选中行与 hover 行背景使用 `--bg-grey-normal`，不再漂成品牌蓝
- [ ] 横向滚动时首列与操作列均钉死，且左固定列向右投影、右固定列向左投影
- [ ] 12px 字号、PingFang SC 与 `table.md` 一致
- [ ] PR 若使用底层 `Table` 而非 `DataTable`，已在 PR description 写明 §6.2 理由

## 11. References

- 上层依赖：`component-specs/table.md`（视觉与底层组合规范）
- 上层依赖：`component-specs/pagination.md`
- 上层依赖：`component-specs/empty-state.md`
- 上层依赖：`component-specs/batch-actions-bar.md`（多选时常配合）
- Demo code：`client/src/components/ui/table.tsx` §9（与 `Table` 同文件出口；MVP 已落地：rowKey / columns / dataSource / rowSelection / pagination / loading / empty）
- 待补能力：`column.sorter` / `column.filters` / `column.ellipsis` Tooltip / `expandable` / 整合 `onChange`
- Demo page（pilot 候选）：`client/src/pages/admin/BashPolicyList.tsx` / `NetGroupList.tsx`
- Ant Design 对照：`https://ant.design/components/table-cn`
- 决策背景：本仓 `feature/design-addietang` 分支 2026-06-08 对话记录（双层 API 决策）

## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 同口径。`DataTable` 7 个状态机的常见误用 → 声明式正确写法。

### 12.1 不要在业务侧 useState 维护 selectedKeys

```tsx
// ❌ 业务自己 useState + 自己接 checkbox，状态视觉双轨漂移
const [selected, setSelected] = useState<string[]>([]);
<Table>
  <TableBody>
    {list.map(row => (
      <TableRow key={row.id}>
        <TableCell><Checkbox checked={selected.includes(row.id)} onChange={...} /></TableCell>
        ...
      </TableRow>
    ))}
  </TableBody>
</Table>

// ✅ 交给 rowSelection，跨页保留默认开
<DataTable<Policy>
  rowKey="id"
  columns={columns}
  dataSource={list}
  rowSelection={{ selectedKeys, onChange: setSelected }}
/>
```

### 12.2 不要在外层手挂 border-t 拼分页

```tsx
// ❌ DataTable 外面再包一层分页
<>
  <DataTable rowKey="id" columns={columns} dataSource={list} />
  <div className="border-t px-4 py-2">
    <Pagination total={total} current={page} onChange={setPage} />
  </div>
</>

// ✅ 用 pagination prop，内部统一处理 border-t / 两端布局 / 高度
<DataTable
  rowKey="id"
  columns={columns}
  dataSource={list}
  pagination={{ current, pageSize, total, onChange: setPage }}
/>
```

### 12.3 不要手写 list.length === 0 渲染空态

```tsx
// ❌ 业务侧自己判空 + 包大卡片
{list.length === 0 ? (
  <SurfaceCard className="py-12 text-center">
    <p>暂无数据</p>
  </SurfaceCard>
) : (
  <DataTable rowKey="id" columns={columns} dataSource={list} />
)}

// ✅ 交给 DataTable，空态自动渲染在 <td colSpan> 内（参考 empty-state.md §14.1）
<DataTable
  rowKey="id"
  columns={columns}
  dataSource={list}
  emptyText="暂无策略，先去新建一条"  // 仅替换文案，不破壳
/>
```

### 12.4 rowKey 必填，不靠 index 兜底

```tsx
// ❌ 不传 rowKey，React 把 index 当 key，删除 / 重排时行状态错乱
<DataTable columns={columns} dataSource={list} />

// ❌ 传函数但其实有现成字段，徒增心智负担
<DataTable
  columns={columns}
  dataSource={list}
  rowKey={(record) => record.id}
/>

// ✅ 优先传 string 字面量字段名
<DataTable rowKey="id" columns={columns} dataSource={list} />

// ✅ 仅当无单字段唯一键时才传函数
<DataTable
  rowKey={(r) => `${r.tenantId}:${r.policyId}`}
  columns={columns}
  dataSource={list}
/>
```

### 12.5 标准列表用 DataTable，特殊场景才回落底层 Table

```tsx
// ❌ 标准 CRUD 列表选择回落底层 Table，重新发明 7 个状态
<Table>
  <TableHeader>...</TableHeader>
  <TableBody>{list.map(row => <TableRow key={row.id}>...</TableRow>)}</TableBody>
</Table>

// ✅ 标准列表强制用 DataTable
<DataTable<Policy>
  rowKey="id"
  columns={columns}
  dataSource={list}
  pagination={paginationProps}
  rowSelection={{ selectedKeys, onChange: setSelected }}
/>

// ✅ 表头分组 / colSpan 合并 / 行内编辑 → 回落底层 Table，PR description 写明 §6.2 理由
<Table>
  <TableHeader>
    <TableRow>
      <TableHead colSpan={2}>基础信息</TableHead>
      <TableHead colSpan={3}>运行指标</TableHead>
    </TableRow>
    {/* … */}
  </TableHeader>
</Table>
```
