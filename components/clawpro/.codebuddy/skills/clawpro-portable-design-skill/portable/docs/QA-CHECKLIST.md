# ClawPro Portable Components · QA / 交付清单

> Owner: addietang
> 用途：本清单是 35+ 全局组件「可移植化」的总账。
> 每个组件标注当前状态（✅/🟡/❌）+ 关键依赖 + 配套 CSS。
> 同事拿到本包应能 1:1 还原 `http://localhost:3002/design-system/components` 的视觉。

---

## 状态图例

| 标记 | 含义 |
|---|---|
| ✅ Ready | React 源码 + 独立 CSS 全部就绪，可直接 import 使用 |
| 🟡 Partial | React 源码已就绪，但样式还嵌在 className 内（依赖宿主 Tailwind）；目标是抽出独立 CSS |
| ❌ Missing | 还未生成 portable 文件，使用方需自行降级或等待补齐 |

---

## Wave 1 ─ 现状盘点

### Foundation 基础视觉（11）

| id | 状态 | React 文件 | CSS 文件 | 备注 |
|---|---|---|---|---|
| color | ✅ | — | `css/tokens.css` | 已通过 token.css 提供完整色板 |
| typography | ❌ | — | — | 待 Wave 2 补齐 |
| surface-card | ❌ | — | — | 待 Wave 2 补齐（含 inner / config / overlay / tenant 5 种 variant） |
| surface-inner | ❌ | — | — | 同上 |
| surface-config | ❌ | — | — | 同上 |
| tenant-card | ❌ | — | — | 同上 |
| separator | ❌ | — | — | 待 Wave 2 补齐 |
| aspect-ratio | ❌ | — | — | 待 Wave 2 补齐 |
| card | ✅ | `react/card.tsx` | — | Wave 1 完善（含 Surface / Inner / Config / Tenant 4 variant） |
| avatar | ❌ | — | — | 待 Wave 2 补齐 |
| dark-veil | 🟡 | `react/dark-veil/dark-veil-static.tsx` | `css/dark-veil.css` | 管控端 hero 动态背景。已就绪 L1 静态兜底（CSS + React 包装）+ `html-css/dark-veil.html`；L0 完整版需宿主仓复制 `client/src/components/ui/DarkVeil.tsx` + `npm i ogl`。分档口径见 `component-specs/dark-veil.md §9` |

### Action 操作（6）

| id | 状态 | React 文件 | CSS 文件 | 备注 |
|---|---|---|---|---|
| button | ✅ | `react/button.tsx` | `css/button.css` | Wave 1 完整抽 CSS（6 variant + 3 size） |
| dropdown-menu | ❌ | — | — | 待 Wave 4 补齐（依赖 Radix） |
| toggle-group | ❌ | — | — | 待 Wave 3 补齐（依赖 Radix） |
| context-menu | ❌ | — | — | 待 Wave 4 补齐（依赖 Radix） |
| favorite-button | ❌ | — | — | 待 Wave 2 补齐（纯按钮组合） |
| more-actions-dropdown | ❌ | — | — | 待 Wave 4 补齐（基于 dropdown-menu） |

### Form 表单（15）

| id | 状态 | React 文件 | CSS 文件 | 备注 |
|---|---|---|---|---|
| input | ✅ | `react/input-select.tsx` | (Tailwind) | Wave 1 |
| input-group | ❌ | — | — | 待 Wave 4 补齐 |
| textarea | ❌ | — | — | 待 Wave 4 补齐 |
| select | ✅ | `react/input-select.tsx` | (Tailwind) | Wave 1 |
| date-picker | ✅ | `react/date-picker.tsx` | (Tailwind) | Wave 1 |
| checkbox | ✅ | `react/selection-controls.tsx` | (Tailwind) | Wave 1 |
| radio-group | ✅ | `react/selection-controls.tsx` | (Tailwind) | Wave 1 |
| radio-card | ❌ | — | — | 待 Wave 3 补齐 |
| switch | ✅ | `react/selection-controls.tsx` | (Tailwind) | Wave 1 |
| transfer | ❌ | — | — | 待 Wave 4 补齐 |
| upload | ❌ | — | — | 待 Wave 4 补齐 |
| slider | ❌ | — | — | 待 Wave 3 补齐（依赖 Radix） |
| tree-select | ❌ | — | — | 待 Wave 4 补齐（复合依赖） |
| form | ❌ | — | — | 待 Wave 4 补齐（依赖 react-hook-form） |
| field | ❌ | — | — | 待 Wave 4 补齐 |
| calendar | ✅ | `react/date-picker.tsx` | (Tailwind) | 与 date-picker 共用 |

### Data 数据展示（10）

| id | 状态 | React 文件 | CSS 文件 | 备注 |
|---|---|---|---|---|
| table | ✅ | `react/table.tsx` | `css/globals.css`（12px !important） | Wave 1 完善（含 DataTable 声明式 API） |
| pagination | ✅ | `react/pagination.tsx` | (Tailwind) | Wave 1 |
| empty | ✅ | `react/empty-state.tsx` | (Tailwind) | Wave 1 完善（含 Empty + TableEmpty 两件套） |
| number-card | ✅ | `react/number-card.tsx` | (Tailwind) | Wave 1（含 4 枚渐变图标） |
| chart-stat | ❌ | — | — | 待 Wave 4 补齐（依赖 recharts，建议提供 SVG mock） |
| skeleton | ❌ | — | — | 待 Wave 2 补齐 |
| progress | ❌ | — | — | 待 Wave 3 补齐（依赖 Radix） |
| spinner | ❌ | — | — | 待 Wave 2 补齐 |
| stepper | ❌ | — | — | 待 Wave 4 补齐 |
| file-browser | ❌ | — | — | 待 Wave 4 补齐 |
| batch-actions-bar | ❌ | — | — | 待 Wave 3 补齐 |
| search-filter-bar | ✅ | `react/search-filter-bar.tsx` | (Tailwind) | Wave 1 完善（受控搜索框 + filters/actions 双槽） |
| filter-trigger | ❌ | — | — | 待 Wave 3 补齐 |
| select-panel | ❌ | — | — | 待 Wave 3 补齐 |

### Feedback 反馈（8）

| id | 状态 | React 文件 | CSS 文件 | 备注 |
|---|---|---|---|---|
| alert | ❌ | — | — | 待 Wave 3 补齐（已在 tokens.css 准备 token） |
| alert-dialog | ❌ | — | — | 待 Wave 4 补齐（依赖 Radix） |
| dialog | ✅ | `react/dialog-drawer.tsx` | (Tailwind) | Wave 1 |
| drawer | ✅ | `react/dialog-drawer.tsx` | (Tailwind) | Wave 1 |
| sheet | ❌ | — | — | 待 Wave 4 补齐（基于 dialog） |
| toast | ❌ | — | — | 待 Wave 4 补齐（基于 sonner） |
| tooltip | ❌ | — | — | 待 Wave 3 补齐（依赖 Radix） |
| hover-card | ❌ | — | — | 待 Wave 3 补齐（依赖 Radix） |

### Navigation 导航（10）

| id | 状态 | React 文件 | CSS 文件 | 备注 |
|---|---|---|---|---|
| admin-sidebar | ✅ | `react/admin-sidebar.tsx` | `css/admin-sidebar.css` | Wave 0 已就绪 |
| admin-page-header | ❌ | — | — | 待 Wave 3 补齐 |
| breadcrumb | ❌ | — | — | 待 Wave 3 补齐 |
| tabs | ✅ | `react/tabs.tsx` | (Tailwind) | Wave 1 |
| line-tabs | ✅ | `react/tabs.tsx` | (Tailwind) | 同 tabs |
| segment | ❌ | — | — | 待 Wave 3 补齐 |
| segmented-tabs | ❌ | — | — | 待 Wave 3 补齐 |
| pagination | ✅ | `react/pagination.tsx` | (Tailwind) | 见 data |
| accordion | ❌ | — | — | 待 Wave 3 补齐（依赖 Radix） |
| collapsible | ❌ | — | — | 待 Wave 3 补齐（依赖 Radix） |
| back-button | ❌ | — | — | 待 Wave 2 补齐（按钮组合） |

### Tag / Status 标签（6）

| id | 状态 | React 文件 | CSS 文件 | 备注 |
|---|---|---|---|---|
| badge | ✅ | `react/badges.tsx` | (Tailwind) | Wave 1 |
| status-tag | ✅ | `react/status-tag.tsx` | (内联 style) | Wave 1 |
| filter-chip | ❌ | — | — | 待 Wave 3 补齐 |
| all-users-tag | ❌ | — | — | 待 Wave 2 补齐 |
| tag-label | ❌ | — | — | 待 Wave 3 补齐 |
| kbd | ❌ | — | — | 待 Wave 2 补齐 |

---

## 总览（v0.1 完成后）

| 状态 | 数量 |
|---|---|
| ✅ Ready（React + 视觉验证 1:1） | 18 个组件 + tokens / globals / button / admin-sidebar 4 份 CSS |
| 🟡 Partial | 0 |
| ❌ Missing（待 Wave 2/3/4） | ~20（typography / tooltip / popover / accordion / dropdown-menu / form / transfer / tree-select 等） |

---

## Wave 推进路线

### Wave 1（已完成 ✅ 2026-06-11）：盘点 + 完善 + 统一入口
- [x] 编写 QA-CHECKLIST.md
- [x] 完善 4 个简陋组件：card / empty-state / table / search-filter-bar
- [x] 抽出 button.css 独立 CSS（其余 13 组件保留 Tailwind class，外部仓装 Tailwind 即用）
- [x] 产出 `css/tokens.css` `css/globals.css` `css/portable.css`（统一入口，按顺序 @import）
- [x] 产出 `react/index.ts`（统一导出 18 个组件）
- [x] 产出 `demo.html`（双击浏览器打开，预览所有现有组件 1:1 还原）
- [x] 产出 `README.md`（3 步集成指南）

### Wave 2：低依赖补齐
- [ ] typography / surface / separator / aspect-ratio / avatar
- [ ] skeleton / spinner / kbd / favorite-button / back-button / all-users-tag

### Wave 3：单 Radix 依赖补齐
- [ ] checkbox / switch / radio-group / tooltip / popover / hover-card
- [ ] collapsible / scroll-area / slider / toggle-group / accordion / progress
- [ ] breadcrumb / alert / segment / radio-card / batch-actions-bar
- [ ] filter-chip / filter-trigger / tag-label / admin-page-header

### Wave 4：复合依赖收尾
- [ ] dropdown-menu / context-menu / more-actions-dropdown
- [ ] dialog / alert-dialog / sheet / drawer / toast
- [ ] transfer / tree-select / form / field
- [ ] input / textarea / input-group / stepper
- [ ] chart-stat / file-browser / select-panel / segmented-tabs

### Wave 5：打包交付
- [ ] 校验所有组件可独立运行
- [ ] 产出最小 demo HTML
- [ ] 打 zip 给到产品同事

---

## 包结构（最终目标）

```
clawpro-portable/
├── README.md              # 使用指南
├── QA-CHECKLIST.md        # 本文件
├── css/
│   ├── tokens.css         # 设计 token 总入口
│   ├── globals.css        # 字体 / 全局规则（PingFang SC / box-sizing 等）
│   ├── portable.css       # 所有组件 CSS 总入口（@import 各组件）
│   ├── button.css
│   ├── card.css
│   ├── table.css
│   ├── ...
│   └── admin-sidebar.css  # 已就绪
├── react/
│   ├── index.ts           # 统一导出
│   ├── button.tsx
│   ├── card.tsx
│   ├── ...
│   └── admin-sidebar.tsx  # 已就绪
└── demo/
    └── index.html         # 最小演示页（可在浏览器直接打开预览所有组件）
```
