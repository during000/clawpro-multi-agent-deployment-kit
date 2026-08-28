# DataTable 双层 API RFC

> Date: 2026-06-08
> Author: addietang
> Audience: 媛媛 / 前端 Owner
> Status: 待对齐
> Related spec: `.codebuddy/skills/clawpro-portable-design-skill/component-specs/data-table.md`

---

## 1. 背景

clawpro 表格目前是 **shadcn 风格组合式 API**：业务自己写 `<Table><TableRow><TableCell>`，自己维护 7 个状态：

1. `selectedKeys` + 全选 + 跨页保留
2. `sortBy / sortOrder` + 三角箭头 + cycle
3. `filters` + Popover + 受控值
4. `pagination` + `border-t` + 字号穿透
5. `loading` mask / `empty` 兜底
6. `rowKey` 取值 + `key={}` 散在各处
7. 固定列阴影 + 选中态 `group-hover` 同步

**症状**：每新建一个列表页都漏接 1～2 个状态，导致选中态 / 空态 / 分页 bug 反复出现（已观察到 `BashPolicyList` / `NetGroupList` / `PRDList` 各自重复造）。

## 2. 决策

引入 **`DataTable`（数据驱动壳子）**，与现有 `Table`（组合式底座）形成**双层 API**。

```
┌─────────────────────────────────────────────────┐
│  <DataTable columns dataSource>     ← 新（推荐） │  90% 列表页
│    内部组合 ↓                                    │
│  <Table><TableRow><TableCell>       ← 旧（保留） │  特殊布局逃生口
└─────────────────────────────────────────────────┘
```

- 新建列表页**强制用 `DataTable`**。
- `Table` 不删除，作为底层逃生口，用于嵌套表 / `colSpan` 合并 / 行内编辑 / 单元格 mini 图表等场景。
- `DataTable` 内部全部使用 `Table`，**视觉资产 100% 继承**：12px 强制 / `autoFixedColumns` / 容器级阴影 / `--bg-brand-selected` 换皮。

## 3. 不做的事

| 备选方案 | 不做的原因 |
|---|---|
| 全量切 Ant Design `<Table>` | 失去 clawpro 已积累的视觉优势（容器级阴影 / 强制字号 / 换皮 token）；迁移成本陡 |
| 仅沉淀 `useTableSelection` 等小 hook | 业务仍要手动拼 7 个 hook，bug 面没缩小，只是从"忘了 useState"挪到"忘了用 hook" |
| 在 `Table` 上加可选 props 实现数据驱动 | 单组件双范式，team 阅读成本高；shadcn 组合式风格被污染 |

## 4. API（要点，全文见 spec）

```tsx
<DataTable<Policy>
  rowKey="id"
  columns={[
    { key: "name", title: "名称", dataIndex: "name" },
    { key: "status", title: "状态", filters: STATUS_FILTERS },
    { key: "updatedAt", title: "更新时间", sorter: true },
    { key: "actions", title: "操作", align: "right", render: (_, row) => <Actions row={row} /> },
  ]}
  dataSource={list}
  loading={loading}
  pagination={{ current, pageSize, total, onChange: setPage }}
  rowSelection={{ selectedKeys, onChange: setSelected }}
  onChange={({ sorter, filters }) => fetch({ sorter, filters })}
/>
```

对照 Ant Design `<Table>`：API 形状一致，回调聚合为对象（`onChange({ pagination, sorter, filters })`），便于 spread 给后端。

## 5. 视觉与组件清单（请媛媛重点 review）

| 项 | 取自 | 是否新增 |
|---|---|---|
| 行高 / 表头底色 / 12px / 边框 | `table.md` 现行口径 | 否，沿用 |
| 选中态背景 | `--bg-brand-selected` | 否，沿用 |
| 空态 | `EmptyState size="sm"` 表体内 | 否，沿用 |
| Loading mask | 整表半透明 + 居中 spinner | **新增**：需要媛媛确认半透明值 |
| 列头排序三角 | 与 Ant 同位置（标题右侧） | **新增**：需要媛媛确认三角图标 / cycle 顺序（`null → asc → desc → null`） |
| 列头筛选漏斗 | 复用现有 `FilterChip` 视觉 | 沿用 |
| 分页区 | `border-t` 内嵌同卡片 | 沿用 |

> **唯一需要新出图的**：列头排序三角图标 + 整表 loading mask 颜色值。

## 6. 落地节奏

| 周 | 事 | 产出 | Owner |
|---|---|---|---|
| W1（本周） | spec 走查 | 媛媛 review `data-table.md` + 出排序三角 / loading mask 视觉 | 媛媛 + addietang |
| W2 | MVP | `DataTable` 实现 selection + pagination + empty + loading + rowKey（覆盖 80% 列表页） | addietang |
| W3 | pilot 验证 | 选 `BashPolicyList` 或 `NetGroupList` 迁移，跑通后再补 sorter / filter | addietang |
| W4+ | sorter / filter / 整合回调补齐，老页按需迁 | — | addietang |

**不阻塞当前 design refresh 主线**，全部在独立分支推进。

## 7. 风险

| 风险 | 缓解 |
|---|---|
| 双层 API 让团队混乱"什么时候用哪个" | spec §6.2 给了明确决策表；新建列表页 PR 若用底层 `Table` 必须在 description 写明属于哪类特殊场景 |
| `DataTable` 把所有口子封死，遇到边角需求只能"加 prop 雪球" | API 表克制，明确不做的清单（无 `bordered` 竖线、无 `<Table.ColumnGroup>`、无行内编辑），边角需求一律走底层逃生口 |
| 老页迁移引入回归 | 不强迁，pilot 跑通后再渐进式迁，每页迁完做一次设计走查 |

## 8. 待确认

请媛媛在以下三点反馈：

1. **是否同意双层 API 方向**（`DataTable` + `Table` 共存，新页强制 `DataTable`）。
2. **排序三角图标**：复用现有 `IconArrowUp/Down` 还是出一套新的？cycle 顺序按 Ant 默认（`null → asc → desc → null`）OK 吗？
3. **Loading mask 半透明值**：建议 `bg-white/60` + `backdrop-blur-sm`，是否可？

确认后即进入 W2 MVP。

## 9. 参考

- spec：`.codebuddy/skills/clawpro-portable-design-skill/component-specs/data-table.md`
- 底层 spec：`.codebuddy/skills/clawpro-portable-design-skill/component-specs/table.md`
- Ant Design 对照：`https://ant.design/components/table-cn`
