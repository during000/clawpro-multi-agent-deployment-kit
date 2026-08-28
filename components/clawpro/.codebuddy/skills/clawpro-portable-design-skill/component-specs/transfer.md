# Transfer

## 1. Purpose

- 在弹窗/抽屉中做"从全集中挑选一批资产"的双面板穿梭交互。
- 替代各业务页面以前各自实现的 `CvmSelectComponent`、双 Table + 自写勾选 + 自写翻页方案。
- 统一左右两侧的描边、表头、行高、勾选态、空态、分页与搬运语义，避免每个业务再重复造轮子。

## 2. Scope

- 适用端：Admin 优先（Tenant 仅在批量配置类弹窗中复用）。
- 必用场景：
  - 批量加策略/批量绑定（一对多关联）
  - 生效范围编辑（已选 vs 候选）
  - 网络管控选 Agent / 资产组成员维护
- 不适用场景：
  - 单选下拉/单条选择 → 用 `Select` / `SearchableSelect`（旧名 Combobox，已并入 Select）
  - **树形层级多选穿梭 → `TreeTransfer`（缺口：当前无组件、无 spec，暂未沉淀）**。触达此需求时**不要**用 `TreeSelect`（仅单选，见 `tree-select.md`）或手拼凑，应标 `needs-design-confirmation` 交设计补绘、并在 `references/conflict-log.md` 记一条缺口，待沉淀后再补 `tree-transfer.md`。
  - 页面级常驻列表（不是浮层） → 直接用 `Table`

## 3. Visual Standard

### 3.1 整体布局

| Item | Value | Notes |
|---|---|---|
| Container | 两面板水平排列，圆角 `4px`，描边 `var(--cp-border)` (#EAEEF4) | 放在 Dialog/Drawer body 内 |
| Panel Width | 左右等宽 `flex-1`；`min-w-0` | `oneWay` 时仍保持等宽 |
| Body Height | 默认 `330px`；可配置 | 弹窗内不再外包滚动条 |
| 中间操作列（batch 模式） | `flex flex-col gap-2`，含 `>` `<` 两个按钮 | `instant` 模式不渲染 |

### 3.2 颜色/字号 token 映射

| 槽位 | Token | 说明 |
|---|---|---|
| 外壳描边 | `var(--cp-border)` (#EAEEF4) | 与全局浅边框一致 |
| 面板内分割线 | `#f0f0f0` | Header↔Search↔Body↔Footer 分割 |
| 面板头部底色 | `var(--cp-bg-subtle)` (#FAFBFD) | 与 Table gray-header 同源 |
| 标题文字 | 14px / Medium / `var(--cp-text-emphasis)` (#0A0A0A) | — |
| 计数/已选 N 项/共 N 项 | 12px / `var(--cp-text-muted)` (#737373) | — |
| 空态文案 | 12px / `var(--cp-text-weak)` (#94A3B8) | — |
| 「清空选择」link | 品牌蓝文字按钮 `var(--cp-text-brand)` | — |
| Search 图标 | `var(--cp-text-weak)` | — |
| Search 输入框 | `border: var(--cp-border)`，白底 | — |
| 行末 X 按钮 | `var(--cp-text-muted)` + hover 变深 | — |
| 表格行/字号/行分割线/hover | 复用 Table `density="compact"` 全局态 | 12px 字号，40px 行高 |
| 分页 | simple + small（24px） | 弹窗内紧凑分页 |
| 中间穿梭按钮（batch） | outline 变体 | 白底 + 边框 |

## 4. Anatomy

```text
Transfer (root)
  ├─ Section (left)
  │    ├─ Header  (Checkbox + 标题 + 计数 + 全选/反选)
  │    ├─ Search  (搜索图标 + Input)
  │    ├─ TableBody (Table density="compact")
  │    │    ├─ TableHeader / TableHead
  │    │    └─ TableBody / TableRow / TableCell
  │    └─ Footer  (共 N 条 + Pagination simple small)
  ├─ Actions [仅 batch 模式]
  │    ├─ Button >  (move to right)
  │    └─ Button <  (move to left, oneWay 时隐藏)
  └─ Section (right)
       ├─ Header  (标题 + 计数 + 清空选择)
       ├─ Search
       ├─ TableBody (instant 模式：每行末尾 X 按钮)
       └─ Footer
```

## 5. 移动模式

| 模式 | 行为 | 适用场景 |
|---|---|---|
| **`instant`（默认推荐）** | 左侧 row checkbox 勾上 → 立刻搬到右侧；右侧 header 显示「清空选择」；右侧每行末尾 X 移除 | 弹窗内最省空间，交互最直接 |
| **`batch`（Ant 经典）** | 左右各维护内部勾选；中间渲染 `>` / `<` 按钮做批量穿梭 | "先选再确认"场景 |
| **`oneWay`** | 右侧不可移回。instant 模式下隐藏行末 X；batch 模式下隐藏 `<` 按钮 | 只允许添加不允许反向移除 |

## 6. States

- **default**：两侧白底，灰头，分页静止。
- **hover**：行底色由表格全局态接管。
- **selected**：行内 Checkbox 勾选状态表达，行本身由表格全局态接管。
- **disabled row**：`isItemDisabled` 命中时 60% 透明度 + `cursor-not-allowed`；通过 `renderDisabledTrigger` 包 Tooltip 给出禁用原因。
- **search empty**：表体内空态，文案区分 4 种：
  - 左侧无数据：`暂无可选数据`
  - 左侧搜不到：`没有匹配项，换个关键词试试`
  - 右侧空：`从左侧选择条目添加`
  - 调用方可通过 `notFoundContent` 覆盖
- **loading**：组件不内置 loading，调用方在外侧用骨架占位。

## 7. Props 速查

| Prop | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `dataSource` | `T[]` | — | 全集（包含已选项） |
| `targetKeys` | `string[]` | — | 受控：当前已选 keys |
| `onChange` | `(nextKeys, dir, moveKeys) => void` | — | keys 变更回调 |
| `rowKey` | `string \| (item) => string` | `'key'` | 行唯一标识 |
| `titles` | `[ReactNode, ReactNode]` | `['全部', '已选']` | 面板标题 |
| `height` | `number` | `330` | 单侧 body 高度 |
| `showSearch` | `boolean` | `false` | 是否显示搜索 |
| `searchPlaceholder` | `string \| [string, string]` | — | 搜索框占位文字 |
| `filterOption` | `(input, item) => boolean` | 全字段不区分大小写 | 搜索过滤函数 |
| `pagination` | `{ pageSize?: number } \| boolean` | `{ pageSize: 10 }` | 分页配置 |
| `isItemDisabled` | `(item) => boolean` | — | 禁用判断 |
| `renderDisabledTrigger` | `(item, defaultCheckbox) => ReactNode` | — | 在禁用行外侧挂 Tooltip |
| `columns` | `TransferColumn<T>[]` | — | 同时作用左右的列定义 |
| `leftColumns` | `TransferColumn<T>[]` | — | 仅左侧列定义 |
| `rightColumns` | `TransferColumn<T>[]` | — | 仅右侧列定义 |
| `mode` | `'instant' \| 'batch'` | `'instant'` | 移动模式 |
| `oneWay` | `boolean` | `false` | 右侧不可移回 |
| `selectedKeys` | `string[]` | — | 仅 batch 模式：受控内部勾选 |
| `onSelectChange` | `(sourceKeys, targetKeys) => void` | — | 仅 batch 模式 |

## 8. Demo Repo Usage

```tsx
import { Transfer } from "@/components/ui/transfer";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MetaText } from "@/components/ui/Typography";

<Transfer<HostItem>
  dataSource={(aiAgentHostList ?? []).map((h) => ({ ...h, key: h.Quuid }))}
  rowKey="key"
  targetKeys={selectMachine}
  onChange={(nextKeys) => setSelectMachine(nextKeys)}
  showSearch
  searchPlaceholder={['搜索资产名称 / ID / IP', '搜索已选资产']}
  pagination={{ pageSize: 8 }}
  height={300}
  titles={['全部 AI Agent 资产', '已选 AI Agent 资产']}
  isItemDisabled={(h) => h.ProtectType !== 'Flagship'}
  renderDisabledTrigger={(_h, defaultCheckbox) => (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex">{defaultCheckbox}</span>
      </TooltipTrigger>
      <TooltipContent>基础版资产请升级到旗舰版以使用该能力</TooltipContent>
    </Tooltip>
  )}
  filterOption={(input, h) => {
    const needle = input.toLowerCase();
    return [h.OpenClawName, h.MachineName, h.InstanceID, h.MachineIp]
      .filter((v): v is string => typeof v === 'string')
      .some((v) => v.toLowerCase().includes(needle));
  }}
  columns={[
    {
      key: 'name',
      header: 'Agent 名称 / ID',
      render: (h) => (
        <div className="min-w-0">
          <div className="truncate text-[var(--cp-text-emphasis)]">
            {h.OpenClawName || h.MachineName || '-'}
          </div>
          <MetaText className="block truncate">{h.InstanceID || '-'}</MetaText>
        </div>
      ),
    },
    { key: 'version', header: '防护版本', width: 100, render: (h) => hostVersionMap[h.ProtectType] ?? '-' },
    { key: 'ip', header: '内网IP', width: 140, render: (h) => h.MachineIp || '-' },
  ]}
/>
```

## 9. Portable Fallback

### 9.1 If host repo already has a Transfer

- 可复用宿主仓的 Transfer 结构（如 Ant Design `<Transfer>`）。
- 但必须覆盖到以下标准：
  - 等宽双面板
  - 表格紧凑密度（12px 字号，40px 行高）
  - 表头浅灰
  - 分割线 `var(--cp-border)`
  - 表体内空态（纯文字，不用插画）
  - 弹窗内 simple 分页
  - 行 hover/selected 走全局 token
- 不要求迁成 demo 仓 API，只要求视觉与状态对齐。

### 9.2 Minimal React fallback

```tsx
function PortableTransfer({ left, right, onMoveRight, onMoveLeft }: any) {
  return (
    <div className="flex items-stretch gap-3">
      <TransferPanel title="全部" count={left.length} items={left} side="left" />
      <div className="flex flex-col items-center justify-center gap-2 py-10">
        <button className="h-9 w-9 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm" onClick={onMoveRight}>›</button>
        <button className="h-9 w-9 rounded-[4px] border border-[var(--cp-border)] bg-white text-sm" onClick={onMoveLeft}>‹</button>
      </div>
      <TransferPanel title="已选" count={right.length} items={right} side="right" />
    </div>
  );
}

function TransferPanel({ title, count, items, side }: any) {
  return (
    <div className="flex min-w-0 flex-1 flex-col overflow-hidden rounded-[4px] border border-[var(--cp-border)] bg-white">
      {/* Header */}
      <header className="flex h-9 items-center justify-between border-b border-[#f0f0f0] bg-[var(--cp-bg-subtle)] px-3">
        <span className="text-sm font-medium text-[var(--cp-text-emphasis)]">{title}</span>
        <span className="text-xs text-[var(--cp-text-muted)]">共 {count} 条</span>
      </header>
      {/* Search */}
      <div className="border-b border-[#f0f0f0] p-2">
        <input
          className="h-7 w-full rounded-[4px] border border-[var(--cp-border)] px-2 text-xs placeholder:text-[var(--cp-text-weak)]"
          placeholder="搜索"
        />
      </div>
      {/* Body — 复用 Table compact fallback */}
      <div className="min-h-[160px] flex-1 overflow-auto">
        {items.length === 0 ? (
          <div className="text-center py-10">
            <p className="text-xs text-[var(--cp-text-weak)]">
              {side === 'left' ? '暂无可选数据' : '从左侧选择条目添加'}
            </p>
          </div>
        ) : (
          <table className="w-full border-collapse text-xs text-[var(--cp-text-body)]">
            <tbody>
              {items.map((item: any, i: number) => (
                <tr key={i} className="border-b border-[#f0f0f0] hover:bg-[#fafafa] h-10">
                  <td className="px-4 py-2">{item.name}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      {/* Footer */}
      <footer className="flex h-9 items-center justify-between border-t border-[#f0f0f0] px-3 text-xs text-[var(--cp-text-muted)]">
        <span>共 {count} 条</span>
        <span>1 / 1</span>
      </footer>
    </div>
  );
}
```

### 9.3 HTML/CSS fallback

见 `portable/html-css/transfer.html`（待补，结构为双份 `table.html` + 中间操作列）。

## 10. Migration Rules

### 旧写法 → 新写法

| 旧写法 | 新写法 |
|---|---|
| `CvmSelectComponent`（自写双面板 + 勾选 + 分页） | 统一 `Transfer` 组件 |
| 双 `<Table>` + 手管 `selectedKeys` + 手写分页 | `Transfer dataSource + targetKeys + pagination` |
| 禁用行无说明、直接隐藏 | `isItemDisabled` + `renderDisabledTrigger` 包 Tooltip |
| 面板分割线写死 `#e4e4e4` / `#f0f0f0` | 统一 `var(--cp-border)` |
| 行末移除用裸 `<button>` + 自定义 hover | 组件内置 ghost 图标按钮 |

### 不允许新增

- 弹窗里再起一份"双 Table + 自定义勾选"结构
- 行末 X 按钮用裸 `<button>` + 自定义 hover 样式
- 面板内分割线硬编码 hex
- 禁用项默默隐藏不给原因

## 11. 强制规则

1. **禁止用 `<Table>` + `<Checkbox>` 手搓"双列穿梭框"** —— 一律使用 Transfer。
2. 列定义中**不得**在 `render` 内自行拼装 `text-xs` / `text-[var(--text-*)]`；正文用 Typography 语义组件，辅助行用 MetaText。
3. 禁用项**必须**通过 `renderDisabledTrigger` 包 Tooltip 给出原因，不允许隐藏禁用行。
4. 弹窗内首选 `mode="instant"` + `pagination={{ pageSize: 8 }}` + `height={300}`。
5. 不得通过 `className` 覆盖颜色/边框/圆角/字号——统一走 token。
6. 如确需新增 props，先更新本节规范并经设计 owner 审核。

## 12. Do / Don't

**Do:**

- 弹窗内首选 `instant` 模式，最直接。
- 所有按钮（移除、移动、清空）都走 Button 变体。
- 表内 hover/selected 继承 Table 全局态。
- 弹窗内用 `simple size="small"` 分页器。
- 禁用行给出 Tooltip 说明原因。

**Don't:**

- 不要在面板里放大插画空态。
- 不要左右两侧字号/行高不一致。
- 不要用列宽硬编码堆出第二种"列对齐"方案。
- 不要在 instant 模式下保留左侧 header checkbox 的"全部搬到右侧"语义。
- 不要在弹窗里再手搓"双 Table + Checkbox"。

## 13. QA Checklist

- [ ] 左右面板等宽，描边统一 `var(--cp-border)`
- [ ] 表头浅灰与 Table gray-header 一致
- [ ] 行高 40px、字号 12px（Table compact 强制）
- [ ] 分页为 simple size="small"
- [ ] `instant` 模式：左侧勾选立即搬右；右侧行末有 X；右侧 header 有"清空选择"
- [ ] `batch` 模式：中间渲染 `<` / `>`；左右各自勾选
- [ ] `oneWay`：instant 下隐藏行末 X；batch 下隐藏 `<`
- [ ] 空态文案区分 4 种场景
- [ ] disabled 行有 60% 透明度 + cursor-not-allowed + Tooltip
- [ ] 所有 token 使用 `var(--cp-*)`，不散写 hex
- [ ] 跨仓 fallback 可独立落地

## 14. References

- 数据来源: `.codebuddy/skills/clawpro-portable-design-skill/`
- Related specs: `component-specs/table.md`, `component-specs/dialog-drawer.md`
- Related tokens: `tokens/colors.md` (`--cp-border`, `--cp-bg-subtle`, `--cp-text-emphasis`, `--cp-text-muted`, `--cp-text-weak`, `--cp-text-brand`)
- Related recipes: `references/components.md`
- Migration source: 旧 `CvmSelectComponent`（已下线）

## 代码对照（✅/❌）

### ❌ 错误：两面板宽度不等
```tsx
<div className="flex gap-4">
  <div className="w-80">{/* 左源面板 */}</div>
  <div className="w-96">{/* 右目标面板 */}</div>
</div>
```
**为什么错**：左右不等宽视觉不平衡；resize 时左右抖动不一致。

### ✅ 正确：flex-1 等宽
```tsx
<Transfer
  source={available}
  target={selected}
  mode="instant"
/>
{/* 内部两面板均 flex-1 / min-w-[280px] */}
```

---

### ❌ 错误：中间放大箭头按钮
```tsx
<div className="flex items-center gap-3">
  <Panel ... />
  <div className="flex flex-col gap-2">
    <Button size="lg" onClick={moveRight}>→</Button>
    <Button size="lg" onClick={moveLeft}>←</Button>
  </div>
  <Panel ... />
</div>
```
**为什么错**：instant 模式不需要中间按钮；按钮+面板重复操作；箭头不符合 Lucide 体系。

### ✅ 正确：instant 模式直接 click 移动
```tsx
<Transfer mode="instant" />
{/* 点击源面板项 → 自动移到目标；
    点击目标面板 X → 自动移回源 */}
```

```tsx
{/* 仅 batch 模式才有中间按钮，且用 Lucide */}
<Transfer mode="batch">
  {/* 内部 ChevronRight / ChevronLeft 中等大小，不放大 */}
</Transfer>
```

---

### ❌ 错误：行高 48px / 字号 14px
```tsx
<TransferItem className="h-12 text-sm">{name}</TransferItem>
```
**为什么错**：Transfer 是密集列表，48px 浪费纵向空间；与紧凑 DataTable 不一致。

### ✅ 正确：紧凑 32px / 12px
```tsx
<Transfer rowHeight={32} fontSize={12} />
{/* 默认即紧凑模式；保留更多可见项 */}
```

---

### ❌ 错误：用业务表格做 Transfer
```tsx
<DataTable
  columns={[checkbox, name, action]}
  data={items}
  rowSelection={{ ... }}
/>
{/* 然后旁边再放一个一样的目标表格 */}
```
**为什么错**：DataTable 行高 40px、列宽自由，Transfer 节奏完全不同；选择态语义混乱。

### ✅ 正确：Transfer 内部紧凑列表
```tsx
<Transfer
  columns={[
    { key: 'name', title: '名称' },
    { key: 'type', title: '类型', width: 80 },
  ]}
  data={items}
  defaultSelected={selectedIds}
  onChange={setSelectedIds}
/>
```

---

### ❌ 错误：自由分页 + 大页签
```tsx
<Pagination total={total} pageSize={20} showSizeChanger showQuickJumper />
```
**为什么错**：Transfer 面板宽度有限，完整 Pagination 会挤占空间；showSizeChanger/Jumper 无意义。

### ✅ 正确：simple 分页
```tsx
<Transfer
  pagination={{ mode: 'simple', pageSize: 50 }}
/>
{/* simple = ←  3/12 → 紧凑两按钮 */}
```
