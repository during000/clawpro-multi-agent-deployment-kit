# TreeSelect（树形单选下拉）

## 1. Purpose

- 统一「树形层级数据的单选」下拉体验：搜索、展开/折叠、单选 + 确认、底部已选回显。
- 收敛两类入口的视觉：toolbar 按钮触发的树形筛选、表头漏斗图标触发的列筛选，避免各页面手拼 `Popover + Input + 自绘树`。

> 本 spec 应 §B.2 D5 裁决「真漏补 spec」新建（此前 `tree-select` 在 `conflict-log.md` 列为「业务封装 id，触达时按需补」）。

## 2. Scope

- 适用端：Admin 优先（部门 / 分组 / 资产树的筛选与单选）。Tenant 在确有树形单选时复用。
- 必用场景：候选项是**层级树**（部门、分组、目录、网络拓扑）且**单选**；放在 toolbar 或表格列头作筛选器。
- 不适用场景：扁平候选 → `SearchableSelect`（旧名 Combobox，见 `combobox.md` / `input-select.md`）；多选穿梭 → `transfer.md`；纯展示/导航树 → `tree.md`（FileTree）；树形**多选** → `transfer.md` 的 `TreeTransfer`（暂未沉淀，见 `transfer.md §2`）。

## 3. 入口与变体

> 真实实现：`client/src/components/ui/tree-select.tsx`（统一封装，内部委托 `_internal/TreeSelectFilter` 与 `_internal/TableHeaderTreeFilter`）。

| 变体 | `triggerVariant` | 触发器 | 承载旧实现 | 典型场景 |
|---|---|---|---|---|
| 按钮 | `"button"`（默认） | 带边框的下拉按钮（默认宽 160） | `TreeSelectFilter` | toolbar 筛选器 |
| 表头漏斗 | `"filter-icon"` | 列标题 + `Filter` 漏斗图标 | `TableHeaderTreeFilter` | 表格列头筛选 |

## 4. Visual Standard

### 4.1 触发器

| 变体 | 触发器口径 |
|---|---|
| button | `h-9`（36px）/ `rounded-[4px]` / `border border-border bg-white` / `px-3 text-sm`；hover & open 切 `border-blue-500`；右侧 `ChevronDown` 16px（open 旋转 180°）；文字 `var(--text-title)`，默认宽 160（`triggerWidth`） |
| filter-icon | 列标题文字 + `Filter` 图标 `w-3.5 h-3.5`（14px）；**未筛选**：`var(--text-weak)`，hover `var(--text-muted)`；**已筛选（激活）**：蓝色，提示该列有筛选生效 |

### 4.2 浮层面板（两变体共用口径）

| Item | Value | Notes |
|---|---|---|
| 面板 | `rounded-[4px]` / `border-none` / `shadow-[var(--shadow-popover)]` | 默认宽 280（`panelWidth`），`sideOffset 4` |
| 搜索区 | 顶部 `p-2 pb-0`；`Search` 图标 14px `var(--text-weak)`；`Input` `h-8 pl-8 text-sm` | `filter-icon` 可 `showSearch={false}` 隐藏 |
| 列表区 | `overflow-y-auto p-2 space-y-0.5`；button 变体 `max-h-[280px]`，filter-icon 变体 `max-h-[260px]` | filter-icon 滚动条仅 hover 显示 |
| 「全部」选项 | 与节点行同款，`value=""` 表示全部 | 搜索态隐藏 |
| 节点行 | `h-8`（32px）/ `px-3` / `rounded-[6px]` / `text-sm`；缩进 `level × 16 + 12`px | 展开箭头 `ChevronRight/Down` 14px |
| 选中态 | `bg-[var(--bg-brand-selected)]` + 蓝色文字 `font-medium` + 右侧 `Check` 16px | — |
| hover 态 | `bg-[var(--bg-grey-hover)]` | — |
| 底部 Footer | `mx-2 border-t` + `py-2`：左侧已选回显（`MetaText` truncate）+ 右侧「取消 / 确认」 | `filter-icon` `instant` 模式建议 `showFooter={false}` |
| Footer 按钮 | 取消 `Button variant="claw-outline" size="sm"`；确认 `variant="dialog-confirm" size="sm"`（`h-7`，`text-xs`） | — |

### 4.3 提交模式（仅 filter-icon 变体）

| `commitMode` | 行为 |
|---|---|
| `"confirm"`（默认） | 点选项只更新临时态，点「确认」才回调生效、关闭面板 |
| `"instant"` | 点选项立即回调并关闭面板（建议同时 `showFooter={false}`，可配 `showSearch={false}`） |

### 4.4 ⚠️ 实现现状 / 对齐缺口（如实记录，勿据此改组件）

> 两个内部实现当前存在硬编码偏差，**列为已知对齐项**（与 M4-4「实现侧去硬编码」同批），本 spec 不改组件代码：
>
> | 元素 | 组件现状 | 本 skill 目标 token |
> |---|---|---|
> | 触发器 hover/open 描边 | `border-blue-500`（#3B82F6） | 品牌蓝 `var(--cp-brand-blue)`（#1447E6） |
> | 选中文字 / Check 图标 | `text-blue-500`（#3B82F6） | 品牌蓝语义 `var(--cp-text-brand)` |
> | filter 激活图标 | `text-[#355EF1]`（裸 hex，第三种蓝） | 品牌蓝 `var(--cp-brand-blue)` |
> | 展开箭头 / ChevronDown | `text-gray-400` / `text-gray-500` | `var(--cp-text-weak)` / `var(--cp-text-muted)` |
> | 节点行圆角 | `rounded-[6px]` | ClawPro 控件标准 `4px`（`rounded-[4px]`） |
> | Footer 分隔线 | `border-[#EAEEF4]`（裸 hex） | `var(--cp-border)` |
>
> **三种蓝并存（`blue-500` / `#355EF1` / 品牌蓝 `#1447E6`）是当前最突出的不一致**，统一方向为品牌蓝。新建页面**直接引用 `TreeSelect` 组件**即可，不要在调用处覆盖这些颜色。

## 5. Anatomy

```text
TreeSelect
  Trigger
    ├ button      → 下拉按钮（标签 + ChevronDown）
    └ filter-icon → 列标题 + Filter 漏斗（激活变蓝）
  Panel (Popover, 280px)
    Search (可选)
    OptionList (max-h 260~280)
      "全部" option
      TreeNode (可嵌套, 缩进 level*16+12)
        ├ Chevron (有子节点时)
        ├ Label (truncate)
        └ Check (选中时)
    Footer (可选)
      SelectedLabel + [取消] [确认]
```

## 6. API

```ts
// 入口：import { TreeSelect } from "@/components/ui/tree-select";
interface TreeSelectNode { id: string; name: string; children?: TreeSelectNode[]; path?: string; }

// 公共 props
nodes: TreeSelectNode[];
value: string;                 // ""=全部
onChange?: (value: string) => void;
allLabel?: string;             // 默认 "全部"
searchPlaceholder?: string;
panelWidth?: number;           // 默认 280
align?: "start" | "center" | "end";

// button 变体专有
triggerVariant?: "button";
triggerWidth?: number;         // 默认 160

// filter-icon 变体专有
triggerVariant: "filter-icon";
title: string;                 // 列标题（必填）
commitMode?: "confirm" | "instant";  // 默认 confirm
showSearch?: boolean;          // 默认 true
showFooter?: boolean;          // 默认 true
```

```tsx
// button 变体（toolbar 筛选）
<TreeSelect nodes={deptTree} value={dept} onChange={setDept} allLabel="全部部门" />

// filter-icon 变体（表头列筛选，点即生效）
<TreeSelect
  triggerVariant="filter-icon"
  title="部门"
  nodes={deptTree}
  value={dept}
  onChange={setDept}
  commitMode="instant"
  showFooter={false}
/>
```

- `path?` 仅 button 变体用于底部面包屑回显（格式如 `"A公司/技术部/前端组"`）。
- filter-icon 变体在所有顶层节点均无 `children` 时**自动进入扁平模式**（不渲染缩进与箭头占位）。

## 7. Portable Fallback

- 宿主仓需要「树形单选下拉」兜底时，建议直接抄 `_internal/TreeSelectFilter.tsx`（依赖 Radix `Popover` + `Input` + lucide），并按 §4.4 把蓝色 / 灰色 / 圆角对齐到 token。
- 若宿主仓不能引 Radix：用 `tree.md` 的 PortableTree 渲染树体，外层套一个受控浮层（触发器 + 浮层定位自理），单选态、搜索、确认按本 spec 口径补齐。
- 不可信数据（节点 name / path）一律按纯文本渲染，不拼 HTML。

## 8. Migration Rules

| 旧写法 | 新写法 |
|---|---|
| 手拼 `Popover + Input + 自绘树` 做筛选 | 统一 `TreeSelect`（按场景选 button / filter-icon） |
| 表头筛选各列样式不一 | 统一 filter-icon 变体；激活态图标变蓝 |
| 直接引用 `TreeSelectFilter` / `TableHeaderTreeFilter` | 迁移到统一入口 `TreeSelect`（旧实现待全部迁移后内联删除） |
| 节点行用品牌蓝实心块选中 | 选中态走 `bg-[var(--bg-brand-selected)]` 浅底 + 蓝字 + Check |

## 9. Do / Don't

**Do:**
- 单选树场景统一用 `TreeSelect`，按入口选 button / filter-icon 变体。
- 表头筛选用 `instant` 模式时关掉 footer，避免多余确认步骤。
- 选中态靠浅底 + 蓝字 + Check 三件套传达，不靠整行重色块。

**Don't:**
- 不在调用处覆盖蓝色 / 灰色 / 圆角（对齐缺口由组件侧统一，见 §4.4）。
- 不用 `TreeSelect` 承载**多选**（多选走 `transfer.md`）。
- 不直接新引 `TreeSelectFilter` / `TableHeaderTreeFilter`（走统一入口 `TreeSelect`）。
- 扁平候选不要套树形选择，用 `SearchableSelect`。

## 10. QA Checklist

- [ ] 候选确为层级树 + 单选，才用 TreeSelect
- [ ] 入口选对：toolbar → button；表头列 → filter-icon
- [ ] 触发器 `h-9` / `rounded-[4px]`；面板 `rounded-[4px]` + `shadow-[var(--shadow-popover)]`
- [ ] 节点行 `h-8`、缩进 `level*16+12`、选中浅底 + 蓝字 + Check
- [ ] confirm / instant 模式与 footer 显隐匹配
- [ ] 调用处未私自覆盖颜色 / 圆角
- [ ] 节点文本按纯文本渲染

## 11. References

- 真实实现：`client/src/components/ui/tree-select.tsx`、`client/src/components/_internal/TreeSelectFilter.tsx`、`client/src/components/_internal/TableHeaderTreeFilter.tsx`
- Related specs：`tree.md`（展示/导航树）、`transfer.md`（多选穿梭 / TreeTransfer 缺口）、`input-select.md` & `combobox.md`（扁平单选）、`popover-dropdown-menu.md`（浮层基线）
- 色 / 圆角 token：`tokens/colors.md`、`tokens/radius-shadow.md`
