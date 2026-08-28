# Batch Actions Bar

## 1. Purpose

- 统一列表 / 表格多选后的选择反馈、跨页选择提示和批量动作区域。
- 这类区域很容易被页面临时写成“已选 N 项 + 几个文字按钮”，导致位置、颜色、危险操作和跨页语义不一致。

## 2. Scope

- 适用端：Admin 优先；Tenant 只有明确的列表管理 / 批量处理场景时复用。
- 必用场景：表格多选、卡片列表多选、Dialog / Drawer 内批量分发或批量绑定、跨页选择。
- 高风险场景：批量删除、批量关闭、批量升级、批量启用 / 停用、批量隔离 / 恢复。
- 不适用场景：单行操作列、PageHeader 主操作、普通搜索筛选工具栏、纯表单页。

## 3. Visual Standard

| Item | Default | Notes |
|---|---|---|
| Visibility | 无选中项时隐藏 | 常驻批量按钮则必须 disabled 并说明原因 |
| Container | 默认白底 + `--cp-border` 描边 + 4px | 贴近 Data Surface，不漂离列表 |
| Height | 32px-36px | 与列表工具条节奏一致 |
| Padding | `px-3 py-2` | Dialog 内可更紧凑 |
| Gap | `gap-2` / `gap-3` | 数量、分隔、动作关系清楚 |
| Summary Text | `已选择 N 项` | 必须显示数量，不只靠 checkbox 状态 |
| Cross Page Prompt | Gmail 风格提示 | 明确“当前页”和“全部筛选结果”的差异 |
| Action Button | `claw-outline` / 宿主等效次级按钮 | 同一条内避免多个强主按钮 |
| Danger Action | destructive / danger 语义 | 必须二次确认 |
| Clear Action | link / ghost / text | 文案用“清除选择”或“取消选择” |

## 4. Anatomy

```text
BatchActionsBar
  SelectionSummary
  CrossPagePrompt optional
  ActionGroup
    BatchActionButton * n
  ClearSelection
```

推荐放置：

```text
SearchFilterBar
Data Surface
  BatchActionsBar optional
  Table
  Pagination
```

跨页提示可放在表头下方、表体上方，或同一个批量操作条内，但必须让用户看懂选择范围。

## 5. States

- hidden: 没有选中项时隐藏批量操作条。
- selected: 显示 `已选择 N 项`、可执行动作和清除入口。
- partial-page-selected: 表头 checkbox 为 indeterminate，行选中态同步。
- page-all-selected: 当前页可选项全部选中；如果还有其他筛选结果，显示“选择全部 N 项”。
- all-filtered-selected: 已选择当前筛选条件下全部可选项，显示 `已选择全部 N 项` 和清除入口。
- all-filtered-selected visual: 跨页全选态容器背景使用 `var(--bg-grey-normal)`，不是 `--cp-brand-tint`。
- mixed-eligibility: 部分选中项不可执行某个动作时，按钮 disabled + helper / tooltip，或在确认弹窗内说明不可执行数量。
- processing: 批量操作执行中，按钮 loading 或局部提示，不锁死整页。
- success: 操作完成后清空选择并反馈结果。
- error: 显示失败原因和可重试入口；不要静默清空选择。
- danger-confirming: 危险动作必须进入 AlertDialog / 确认弹窗。

## 6. Demo Repo Usage

- 预览批量操作浮条：`client/src/pages/admin/MemoryManagementRedesign/MemoryManagementRedesign.tsx`
- 复杂表格选择：`client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx`
- 技能批量操作：`client/src/pages/admin/Security/AIAgent/Skills/index.tsx`
- 批量分发弹窗：`client/src/pages/admin/SkillLibrary/BatchDistributeDialog.tsx`
- 文件 / 网盘列表：`client/src/pages/admin/FileManagement.tsx`
- 表格选中态底层：`client/src/components/ui/table.tsx`

```tsx
{selectedIds.size > 0 && (
  <div className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-[4px] border border-[var(--cp-border)] bg-white px-3 py-2">
    <div className="flex items-center gap-2 text-xs text-[var(--cp-text-brand)]">
      <span className="font-medium">已选择 {selectedIds.size} 项</span>
      {isPageAllSelected && !isAllFilteredSelected ? (
        <button type="button" className="underline underline-offset-2">选择全部 {totalCount} 项</button>
      ) : null}
    </div>
    <div className="flex items-center gap-3">
      <Button variant="claw-outline" size="claw-sm">批量删除</Button>
      <Button variant="claw-outline" size="claw-sm">批量导出</Button>
      <Button variant="ghost" size="claw-sm">取消选择</Button>
    </div>
  </div>
)}

<div className="flex items-center justify-between rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-4 py-2.5">
  <MetaText>已选择全部 <span className="text-[var(--cp-text-brand)] font-medium">156</span> 项（跨页）</MetaText>
  <div className="flex items-center gap-3">
    <Button variant="claw-outline" size="claw-sm">批量删除</Button>
    <Button variant="ghost" size="claw-sm">取消选择</Button>
  </div>
</div>
```

## 7. Portable Fallback

### 7.1 If host repo already has batch toolbar / data table selection

- 优先复用宿主仓已有选择状态、checkbox 和批量操作逻辑。
- 必须补齐视觉与语义：选中数量、清除入口、跨页提示、危险操作确认、不可执行状态说明。
- 不要求迁成 demo 仓 API；表格实现可保留，但 selected row、indeterminate checkbox 和 toolbar 必须一致。

### 7.2 Minimal React fallback

```tsx
import * as React from "react";

export function PortableBatchActionsBar() {
  const [selectedCount, setSelectedCount] = React.useState(3);
  const totalCount = 36;
  const allFilteredSelected = selectedCount === totalCount;

  if (selectedCount === 0) return null;

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-brand-tint)] px-3 py-2">
      <div className="flex min-w-0 flex-wrap items-center gap-2 text-xs text-[var(--cp-text-brand)]">
        <span className="font-medium">{allFilteredSelected ? `已选择全部 ${totalCount} 项` : `已选择 ${selectedCount} 项`}</span>
        {!allFilteredSelected && (
          <button type="button" onClick={() => setSelectedCount(totalCount)} className="underline underline-offset-2">
            选择全部 {totalCount} 项
          </button>
        )}
      </div>
      <div className="flex shrink-0 flex-wrap items-center gap-2">
        <button type="button" className="h-8 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-xs text-[var(--cp-text-title)]">批量启用</button>
        <button type="button" className="h-8 rounded-[4px] border border-[var(--cp-border)] bg-[var(--cp-surface)] px-3 text-xs text-[var(--cp-text-danger)]">批量删除</button>
        <button type="button" onClick={() => setSelectedCount(0)} className="h-8 px-2 text-xs text-[var(--cp-text-muted)] hover:text-[var(--cp-text-title)]">清除选择</button>
      </div>
    </div>
  );
}
```

### 7.3 Minimal HTML/CSS fallback

```html
<div class="cp-batch-bar">
  <div class="cp-batch-summary">
    <strong>已选择 2 项</strong>
    <button type="button">选择全部 36 项</button>
  </div>
  <div class="cp-batch-actions">
    <button type="button" class="danger">批量删除</button>
    <button type="button">批量导出</button>
    <button type="button" class="clear">取消选择</button>
  </div>
</div>

<div class="cp-batch-bar cp-batch-bar-all">
  <div class="cp-batch-summary">
    <strong>已选择全部 156 项（跨页）</strong>
  </div>
  <div class="cp-batch-actions">
    <button type="button" class="danger">批量删除</button>
    <button type="button" class="clear">取消选择</button>
  </div>
</div>
```

```css
.cp-batch-bar { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 8px 12px; }
.cp-batch-bar-all { background: var(--bg-grey-normal); }
.cp-batch-summary { display: flex; min-width: 0; flex-wrap: wrap; align-items: center; gap: 8px; font-size: 12px; color: var(--cp-text-brand); }
.cp-batch-summary button { border: 0; background: transparent; padding: 0; color: var(--cp-text-brand); text-decoration: underline; text-underline-offset: 2px; }
.cp-batch-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
.cp-batch-actions button { height: 32px; border: 1px solid var(--cp-border); border-radius: 4px; background: var(--cp-surface); padding: 0 12px; font-size: 12px; color: var(--cp-text-title); }
.cp-batch-actions .danger { color: var(--cp-text-danger); }
.cp-batch-actions .clear { border-color: transparent; background: transparent; color: var(--cp-text-muted); }
```

## 8. Migration Rules

- 旧写法：页面只在右侧临时显示 `已选 N 项` 和几个文字按钮，跨页选择、清除入口、危险确认各不一致。
- 新口径：把批量操作条视为 Data Surface 的状态区，和 Table / Pagination 一起设计。
- 表头 checkbox 必须支持 checked / unchecked / indeterminate；行级 selected 状态和批量条数量同步。
- 当前页全选与全部筛选结果全选必须用文案区分，不允许 checkbox 静默选择所有页。
- 批量危险操作必须走确认弹窗；确认内容应包含数量和影响对象类型。
- 部分不可执行的选中项必须在按钮、helper 或确认弹窗中说明，不要点击后才 toast 一个模糊失败。
- 操作执行中不锁死整页；局部按钮 loading 或状态提示即可。
- 成功后清空选择；失败时保留选择并给出重试 / 修正路径。

## 9. Do / Don't

Do:

- 显示清楚的选中数量和清除入口。
- 对跨页选择使用明确文案，例如“已选择此页 10 项，选择全部 36 项”。
- 对危险批量操作使用 destructive 语义和二次确认。
- 让批量操作条和表格处于同一个数据区域。

Don't:

- 不要在没有选中项时展示可点击的批量动作。
- 不要只靠颜色表达“已选中”。
- 不要让批量操作条漂到 PageHeader 或页面底部，远离数据表。
- 不要静默忽略不可执行的选中项。

## 10. QA Checklist

- [ ] 无选中项时批量操作条隐藏或按钮 disabled 且说明原因
- [ ] 已选中时显示数量、操作和清除入口
- [ ] 表头 checkbox 支持 indeterminate
- [ ] 行选中态和批量条数量同步
- [ ] 当前页全选和全部筛选结果全选语义明确
- [ ] 跨页选择有“选择全部 / 清除选择”入口
- [ ] 危险批量操作有二次确认
- [ ] 部分不可执行项有提示或禁用说明
- [ ] 执行中 / 成功 / 失败状态完整
- [ ] 批量操作条与 Table / Pagination 属于同一个 Data Surface
- [ ] 宿主仓无法复用 demo 组件时，fallback 仍可独立落地

## 11. References

- Demo page: `client/src/pages/admin/MemoryManagementRedesign/MemoryManagementRedesign.tsx`
- Demo page: `client/src/pages/admin/MemoryManagement/components/InstanceTable.tsx`
- Demo page: `client/src/pages/admin/Security/AIAgent/Skills/index.tsx`
- Demo page: `client/src/pages/admin/SkillLibrary/BatchDistributeDialog.tsx`
- Related spec: `component-specs/table.md`
- Related spec: `component-specs/search-filter-bar.md`
- Related spec: `component-specs/button.md`
- Related recipe: `references/page-recipes.md`

## 12. 代码对照（✅/❌）

> 与 SKILL.md §2 / table.md §12 同口径。BatchActionsBar 5 项高频误用 → ClawPro 正确写法。

### 12.1 没选中项时必须隐藏，不要常驻 disabled

```tsx
// ❌ 批量操作条永远显示，按钮 disabled，挤占垂直空间
<div className="mb-3 flex justify-between rounded-[4px] border bg-white px-3 py-2">
  <span className="text-xs text-gray-400">已选择 0 项</span>
  <div className="flex gap-2">
    <Button disabled>批量删除</Button>
    <Button disabled>批量导出</Button>
  </div>
</div>
<DataTable ... />

// ✅ selectedIds = 0 时直接隐藏（特殊场景必须常驻才用 disabled，且配 helper 说明原因）
{selectedIds.size > 0 && (
  <div className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-[4px] border border-[var(--cp-border)] bg-white px-3 py-2">
    <span className="text-xs font-medium text-[var(--cp-text-brand)]">
      已选择 {selectedIds.size} 项
    </span>
    <div className="flex items-center gap-3">
      <Button variant="claw-outline" size="claw-sm">批量删除</Button>
      <Button variant="ghost" size="claw-sm" onClick={clearSelection}>取消选择</Button>
    </div>
  </div>
)}
```

### 12.2 当前页全选 vs 跨页全选语义必须明确

```tsx
// ❌ 表头 checkbox 一勾就静默选中 36 项（跨页），用户以为只选了当前页 10 项
<input
  type="checkbox"
  onChange={(e) => setSelectedIds(e.target.checked ? new Set(allFilteredIds) : new Set())}
/>

// ✅ 表头 checkbox 只选当前页，跨页选择走显式 prompt
<Checkbox
  checked={isPageAllSelected}
  indeterminate={isPagePartialSelected}
  onCheckedChange={(v) => setSelectedIds(v ? new Set(currentPageIds) : new Set())}
/>

{isPageAllSelected && !isAllFilteredSelected && (
  <div className="text-xs text-[var(--cp-text-brand)]">
    已选择此页 {currentPageIds.length} 项，
    <button
      type="button"
      className="underline underline-offset-2"
      onClick={() => setSelectedIds(new Set(allFilteredIds))}
    >
      选择全部 {totalCount} 项
    </button>
  </div>
)}
```

### 12.3 危险批量操作必须二次确认 + 数量影响说明

```tsx
// ❌ 点"批量删除"直接调 API
<Button
  variant="claw-outline"
  size="claw-sm"
  className="text-[var(--cp-text-danger)]"
  onClick={() => batchDelete(Array.from(selectedIds))}
>
  批量删除
</Button>

// ✅ 进 AlertDialog，确认文案带数量 + 影响对象
<Button
  variant="claw-outline"
  size="claw-sm"
  className="text-[var(--cp-text-danger)]"
  onClick={() => setConfirmOpen(true)}
>
  批量删除
</Button>

<AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>批量删除 {selectedIds.size} 个 Agent？</AlertDialogTitle>
      <AlertDialogDescription>
        将释放对应的会话历史与配额，操作不可恢复。
        其中 {disabledIds.length} 个 Agent 仍有进行中的会话，将跳过删除。
      </AlertDialogDescription>
    </AlertDialogHeader>
    <AlertDialogFooter>
      <AlertDialogCancel>取消</AlertDialogCancel>
      <AlertDialogAction onClick={runBatchDelete}>确认删除</AlertDialogAction>
    </AlertDialogFooter>
  </AlertDialogContent>
</AlertDialog>
```

### 12.4 部分不可执行项要在按钮 / 弹窗里说清

```tsx
// ❌ 选了 5 个，其中 2 个不可批量启用，点击后 toast 一句"部分失败"
<Button onClick={() => batchEnable(Array.from(selectedIds))}>批量启用</Button>
// 点击后：toast.error("部分启用失败")

// ✅ 按钮 hover 提示哪些不能 + 确认弹窗里说明数量
<Tooltip>
  <TooltipTrigger asChild>
    <Button variant="claw-outline" size="claw-sm" onClick={() => setConfirmOpen(true)}>
      批量启用
    </Button>
  </TooltipTrigger>
  {disabledForEnable > 0 && (
    <TooltipContent>
      其中 {disabledForEnable} 个已停服 Agent 不会被启用
    </TooltipContent>
  )}
</Tooltip>

<AlertDialog open={confirmOpen}>
  <AlertDialogContent>
    <AlertDialogTitle>批量启用 {effectiveCount} 个 Agent？</AlertDialogTitle>
    <AlertDialogDescription>
      已选 {selectedIds.size} 项，其中 {disabledForEnable} 项已停服，
      实际生效 {effectiveCount} 项。
    </AlertDialogDescription>
  </AlertDialogContent>
</AlertDialog>
```

### 12.5 跨页全选态背景用 --bg-grey-normal，不用 --cp-brand-tint

```tsx
// ❌ 跨页全选态用品牌蓝 tint，与 hover / selected option 同色，视觉过激
{isAllFilteredSelected && (
  <div className="mb-3 rounded-[4px] border bg-[var(--cp-brand-tint)] px-4 py-2.5">
    已选择全部 156 项（跨页）
  </div>
)}

// ✅ 已选择全部跨页态用 --bg-grey-normal（最浅灰），与 brand-tint 区隔
{isAllFilteredSelected && (
  <div className="mb-3 flex items-center justify-between rounded-[4px] border border-[var(--cp-border)] bg-[var(--bg-grey-normal)] px-4 py-2.5">
    <MetaText>
      已选择全部 <span className="text-[var(--cp-text-brand)] font-medium">{totalCount}</span> 项（跨页）
    </MetaText>
    <div className="flex items-center gap-3">
      <Button variant="claw-outline" size="claw-sm">批量删除</Button>
      <Button variant="ghost" size="claw-sm" onClick={clearSelection}>取消选择</Button>
    </div>
  </div>
)}
```
