# Component Reference Mapping

ClawPro Portable Design Skill 的组件三层映射表。

**最后更新**: 2026-06-26  
**总组件数**: 42

---

## Quick Reference Table

| 组件 | Spec 文件 | React 实现 | CSS 文件 | 类型 |
|------|----------|----------|---------|------|
| admin-sidebar | [`admin-sidebar.md`](../component-specs/admin-sidebar.md) | [`admin-sidebar.tsx`](../portable/react/admin-sidebar.tsx) | - | 📦 UI |
| admin-sidebar-with-tooltip | - | [`admin-sidebar-with-tooltip.tsx`](../portable/react/admin-sidebar-with-tooltip.tsx) | - | 🔧 Framework |
| alert | [`alert.md`](../component-specs/alert.md) | [`alert.tsx`](../portable/react/alert.tsx) | [`alert.css`](../portable/css/alert.css) | 📦 UI |
| avatar | [`avatar.md`](../component-specs/avatar.md) | [`avatar.tsx`](../portable/react/avatar.tsx) | [`avatar.css`](../portable/css/avatar.css) | 📦 UI |
| badge | [`badge.md`](../component-specs/badge.md) | - | [`badge.css`](../portable/css/badge.css) | 📦 UI |
| badges | - | [`badges.tsx`](../portable/react/badges.tsx) | - | 🔧 Framework |
| batch-actions-bar | [`batch-actions-bar.md`](../component-specs/batch-actions-bar.md) | [`batch-actions-bar.tsx`](../portable/react/batch-actions-bar.tsx) | [`batch-actions-bar.css`](../portable/css/batch-actions-bar.css) | 📦 UI |
| breadcrumb | [`breadcrumb.md`](../component-specs/breadcrumb.md) | [`breadcrumb.tsx`](../portable/react/breadcrumb.tsx) | [`breadcrumb.css`](../portable/css/breadcrumb.css) | 📦 UI |
| button | [`button.md`](../component-specs/button.md) | [`button.tsx`](../portable/react/button.tsx) | [`button.css`](../portable/css/button.css) | 📦 UI |
| card | - | [`card.tsx`](../portable/react/card.tsx) | - | 🔧 Framework |
| card-surface | [`card-surface.md`](../component-specs/card-surface.md) | - | - | 📦 UI |
| chart-stat | [`chart-stat.md`](../component-specs/chart-stat.md) | [`chart-stat.tsx`](../portable/react/chart-stat.tsx) | [`chart-stat.css`](../portable/css/chart-stat.css) | 📦 UI |
| combobox | [`combobox.md`](../component-specs/combobox.md) | - | - | 📦 UI |
| data-table | [`data-table.md`](../component-specs/data-table.md) | - | - | 📦 UI |
| date-picker | [`date-picker.md`](../component-specs/date-picker.md) | [`date-picker.tsx`](../portable/react/date-picker.tsx) | [`date-picker.css`](../portable/css/date-picker.css) | 📦 UI |
| dialog-drawer | [`dialog-drawer.md`](../component-specs/dialog-drawer.md) | [`dialog-drawer.tsx`](../portable/react/dialog-drawer.tsx) | [`dialog-drawer.css`](../portable/css/dialog-drawer.css) | 📦 UI |
| empty-state | [`empty-state.md`](../component-specs/empty-state.md) | [`empty-state.tsx`](../portable/react/empty-state.tsx) | [`empty-state.css`](../portable/css/empty-state.css) | 📦 UI |
| file-browser | [`file-browser.md`](../component-specs/file-browser.md) | [`file-browser.tsx`](../portable/react/file-browser.tsx) | [`file-browser.css`](../portable/css/file-browser.css) | 📦 UI |
| form-controls | [`form-controls.md`](../component-specs/form-controls.md) | [`form-controls.tsx`](../portable/react/form-controls.tsx) | [`form-controls.css`](../portable/css/form-controls.css) | 📦 UI |
| input-select | [`input-select.md`](../component-specs/input-select.md) | [`input-select.tsx`](../portable/react/input-select.tsx) | - | 📦 UI |
| loading-progress | [`loading-progress.md`](../component-specs/loading-progress.md) | [`loading-progress.tsx`](../portable/react/loading-progress.tsx) | [`loading-progress.css`](../portable/css/loading-progress.css) | 📦 UI |
| number-card | [`number-card.md`](../component-specs/number-card.md) | [`number-card.tsx`](../portable/react/number-card.tsx) | [`number-card.css`](../portable/css/number-card.css) | 📦 UI |
| page-header | [`page-header.md`](../component-specs/page-header.md) | [`page-header.tsx`](../portable/react/page-header.tsx) | [`page-header.css`](../portable/css/page-header.css) | 📦 UI |
| pagination | [`pagination.md`](../component-specs/pagination.md) | [`pagination.tsx`](../portable/react/pagination.tsx) | [`pagination.css`](../portable/css/pagination.css) | 📦 UI |
| popover-dropdown-menu | [`popover-dropdown-menu.md`](../component-specs/popover-dropdown-menu.md) | - | - | 📦 UI |
| popover-menu | - | [`popover-menu.tsx`](../portable/react/popover-menu.tsx) | - | 🔧 Framework |
| search-filter-bar | [`search-filter-bar.md`](../component-specs/search-filter-bar.md) | [`search-filter-bar.tsx`](../portable/react/search-filter-bar.tsx) | [`search-filter-bar.css`](../portable/css/search-filter-bar.css) | 📦 UI |
| searchable-select | - | [`searchable-select.tsx`](../portable/react/searchable-select.tsx) | - | 🔧 Framework |
| segment | [`segment.md`](../component-specs/segment.md) | [`segment.tsx`](../portable/react/segment.tsx) | [`segment.css`](../portable/css/segment.css) | 📦 UI |
| segment-showcase | - | [`segment-showcase.tsx`](../portable/react/segment-showcase.tsx) | - | 🔧 Framework |
| selection-controls | [`selection-controls.md`](../component-specs/selection-controls.md) | [`selection-controls.tsx`](../portable/react/selection-controls.tsx) | [`selection-controls.css`](../portable/css/selection-controls.css) | 📦 UI |
| status-tag | [`status-tag.md`](../component-specs/status-tag.md) | [`status-tag.tsx`](../portable/react/status-tag.tsx) | [`status-tag.css`](../portable/css/status-tag.css) | 📦 UI |
| table | [`table.md`](../component-specs/table.md) | [`table.tsx`](../portable/react/table.tsx) | [`table.css`](../portable/css/table.css) | 📦 UI |
| tabs | [`tabs.md`](../component-specs/tabs.md) | [`tabs.tsx`](../portable/react/tabs.tsx) | [`tabs.css`](../portable/css/tabs.css) | 📦 UI |
| tag-label | [`tag-label.md`](../component-specs/tag-label.md) | - | - | 📦 UI |
| tenant-topnav | [`tenant-topnav.md`](../component-specs/tenant-topnav.md) | - | - | 📦 UI |
| toast | [`toast.md`](../component-specs/toast.md) | [`toast/toast.tsx`](../portable/react/toast/toast.tsx) | [`toast/toast.css`](../portable/css/toast/toast.css) | 📦 UI |
| tooltip | [`tooltip.md`](../component-specs/tooltip.md) | [`tooltip.tsx`](../portable/react/tooltip.tsx) | [`tooltip.css`](../portable/css/tooltip.css) | 📦 UI |
| transfer | [`transfer.md`](../component-specs/transfer.md) | [`transfer.tsx`](../portable/react/transfer.tsx) | [`transfer.css`](../portable/css/transfer.css) | 📦 UI |
| tree | [`tree.md`](../component-specs/tree.md) | [`tree.tsx`](../portable/react/tree.tsx) | [`tree.css`](../portable/css/tree.css) | 📦 UI |
| typography | [`typography.md`](../component-specs/typography.md) | - | - | 📦 UI |
| upload-file-browser | [`upload-file-browser.md`](../component-specs/upload-file-browser.md) | - | - | 📦 UI |

---

## Lookup Index

### By Component Name

- **admin-sidebar** (⚠ Partial)
  - Spec: ✓
  - React: ✓
  - CSS: ✗
- **admin-sidebar-with-tooltip** (⚠ Partial)
  - Spec: ✗
  - React: ✓
  - CSS: ✗
- **alert** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **avatar** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **badge** (⚠ Partial)
  - Spec: ✓
  - React: ✗
  - CSS: ✓
- **badges** (⚠ Partial)
  - Spec: ✗
  - React: ✓
  - CSS: ✗
- **batch-actions-bar** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **breadcrumb** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **button** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **card** (⚠ Partial)
  - Spec: ✗
  - React: ✓
  - CSS: ✗
- **card-surface** (⚠ Partial)
  - Spec: ✓
  - React: ✗
  - CSS: ✗
- **chart-stat** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **combobox** (⚠ Partial)
  - Spec: ✓
  - React: ✗
  - CSS: ✗
- **data-table** (⚠ Partial)
  - Spec: ✓
  - React: ✗
  - CSS: ✗
- **date-picker** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **dialog-drawer** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **empty-state** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **file-browser** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **form-controls** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **input-select** (⚠ Partial)
  - Spec: ✓
  - React: ✓
  - CSS: ✗
- **loading-progress** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **number-card** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **page-header** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **pagination** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **popover-dropdown-menu** (⚠ Partial)
  - Spec: ✓
  - React: ✗
  - CSS: ✗
- **popover-menu** (⚠ Partial)
  - Spec: ✗
  - React: ✓
  - CSS: ✗
- **search-filter-bar** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **searchable-select** (⚠ Partial)
  - Spec: ✗
  - React: ✓
  - CSS: ✗
- **segment** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **segment-showcase** (⚠ Partial)
  - Spec: ✗
  - React: ✓
  - CSS: ✗
- **selection-controls** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **status-tag** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **table** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **tabs** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **tag-label** (⚠ Partial)
  - Spec: ✓
  - React: ✗
  - CSS: ✗
- **tenant-topnav** (⚠ Partial)
  - Spec: ✓
  - React: ✗
  - CSS: ✗
- **toast** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **tooltip** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **transfer** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **tree** (✓ Complete)
  - Spec: ✓
  - React: ✓
  - CSS: ✓
- **typography** (⚠ Partial)
  - Spec: ✓
  - React: ✗
  - CSS: ✗
- **upload-file-browser** (⚠ Partial)
  - Spec: ✓
  - React: ✗
  - CSS: ✗

---

## By Implementation Type

### UI Components (with Specs)
- admin-sidebar
- alert
- avatar
- badge
- batch-actions-bar
- breadcrumb
- button
- card-surface
- chart-stat
- combobox
- data-table
- date-picker
- dialog-drawer
- empty-state
- file-browser
- form-controls
- input-select
- loading-progress
- number-card
- page-header
- pagination
- popover-dropdown-menu
- search-filter-bar
- segment
- selection-controls
- status-tag
- table
- tabs
- tag-label
- tenant-topnav
- toast
- tooltip
- transfer
- tree
- typography
- upload-file-browser

### Framework Components (without Specs)
- admin-sidebar-with-tooltip
- badges
- card
- popover-menu
- searchable-select
- segment-showcase

---

## Coverage Analysis

### Complete Implementations (3/3 layers)
- alert
- avatar
- batch-actions-bar
- breadcrumb
- button
- chart-stat
- date-picker
- dialog-drawer
- empty-state
- file-browser
- form-controls
- loading-progress
- number-card
- page-header
- pagination
- search-filter-bar
- segment
- selection-controls
- status-tag
- table
- tabs
- toast
- tooltip
- transfer
- tree

### Partial Implementations
- admin-sidebar (spec, react)
- admin-sidebar-with-tooltip (react)
- badge (spec, css)
- badges (react)
- card (react)
- card-surface (spec)
- combobox (spec)
- data-table (spec)
- input-select (spec, react)
- popover-dropdown-menu (spec)
- popover-menu (react)
- searchable-select (react)
- segment-showcase (react)
- tag-label (spec)
- tenant-topnav (spec)
- typography (spec)
- upload-file-browser (spec)

---

## MANIFEST Validation

Run `npm run verify-manifest` to validate this mapping against actual files.

**Last validated**: 2026-06-26

---

**Generated by**: `scripts/generate-component-mapping.mjs`  
**Auto-update**: Yes (regenerate from actual files)
