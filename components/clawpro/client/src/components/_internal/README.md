# _internal — 内部实现，禁止业务直接 import

此目录存放被新组件 wrapper 内部封装的"原始实现"，对业务层不可见。

## 谁能 import？

| 文件 | 仅由谁 import |
|------|--------------|
| `ScopeFilterDropdown.tsx` | `@/components/ScopeSelect` (instant 模式) |
| `ScopeEditPopover.tsx` | `@/components/ScopeSelect` (confirm 模式) |
| `TableHeaderFilter.tsx` | `@/components/ui/select` (FilterMultiSelect 导出) |
| `TableHeaderTreeFilter.tsx` | `@/components/ui/tree-select` (filter-icon 变体) |
| `TreeSelectFilter.tsx` | `@/components/ui/tree-select` (button 变体) |

## 业务侧应当 import 什么？

| 旧组件名 | 新使用方式 |
|---------|-----------|
| `TableHeaderFilter` | `import { FilterMultiSelect } from "@/components/ui/select"` |
| `TableHeaderTreeFilter` | `import { TreeSelect } from "@/components/ui/tree-select"`，传 `triggerVariant="filter-icon" + title` |
| `TreeSelectFilter` | `import { TreeSelect } from "@/components/ui/tree-select"`（默认 `triggerVariant="button"`） |
| `ScopeFilterDropdown` | `import { ScopeSelect } from "@/components/ScopeSelect"`（不传 `mode` 即 instant） |
| `ScopeEditPopover` | `import { ScopeSelect } from "@/components/ScopeSelect"`，传 `scope` 自动走 confirm 模式 |

## 为什么不直接物理删除？

1. **新 wrapper 通过 props 转发委托给这些实现**——直接删除会导致新组件失效。
2. **保留实现独立**便于未来重构：每个文件 200-400 行职责单一，比内联到一个 800+ 行 wrapper 更易读。
3. **物理位置标记**：移到 `_internal/` 表达"逻辑删除"——业务侧已经无任何文件直接引用它们。

如要彻底物理删除，需要先把实现内联到对应 wrapper（约 1500 行迁移工作），然后删除本目录。
