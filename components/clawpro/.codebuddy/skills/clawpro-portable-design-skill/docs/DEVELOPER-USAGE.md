# Developer Usage

> 面向产品前端的 ClawPro Portable Design Pack 使用说明。目标是在宿主仓低摩擦完成 Admin / Tenant 页面换皮，而不是直接拷贝 demo 仓代码或立即接入完整组件库。

## 1. 这包东西是什么

交付包位置：

```text
.codebuddy/skills/clawpro-portable-design-skill/
```

它回答四个问题：

1. 页面最终应该长什么样。
2. 高风险组件应该遵循什么视觉和交互标准。
3. demo 仓里已有实现可以如何参考。
4. 宿主仓没有同名组件时，如何用 fallback 方式还原。

当前目标是支持周一换皮落地，不是完整 npm 组件库化。

## 2. 拿到包后怎么读

### 2.1 总入口

1. `README.md`：了解目标、使用顺序、交付边界。
2. `HANDOFF.md`：了解如何移交、哪些可直接使用、哪些仍需设计侧后续确认。
3. `INDEX.md`：快速定位每个文件的用途。
4. `STATUS.md`：了解已完成内容、当前共识、已知问题。

### 2.2 全局和分端规则

1. `references/foundation.md`：全局 token、颜色、字体、圆角、阴影、动效。
2. Admin 页面读 `references/admin.md`。
3. Tenant 页面读 `references/tenant.md`。
4. 页面开工前读 `references/page-recipes.md` 和 `references/migration-map.md`。

### 2.3 组件规范

遇到组件时优先读 `component-specs/`，不要自由发挥。

| 页面里看到的东西 | 必读 spec |
|---|---|
| 表格 | `component-specs/table.md` |
| 卡片 / Surface | `component-specs/card-surface.md` |
| 按钮 | `component-specs/button.md` |
| 空状态 | `component-specs/empty-state.md` |
| 页头 | `component-specs/page-header.md` |
| 弹窗 / 抽屉 | `component-specs/dialog-drawer.md` |
| Popover / Dropdown / Tooltip | `component-specs/popover-dropdown-menu.md` |
| LineTabs（page header 下方一级 Tab） | `component-specs/tabs.md` |
| Segment（卡片内 / 工具栏切换） | `component-specs/segment.md` |
| 表单控件 | `component-specs/form-controls.md` |
| 状态标签（运行 / 角色 / 信息） | `component-specs/status-tag.md` |
| 分类标签 / Badge（版本 / 范围 / new 角标） | `component-specs/badge.md` |
| 分页 | `component-specs/pagination.md` |
| 搜索筛选工具条 | `component-specs/search-filter-bar.md` |
| 日期选择器 | `component-specs/date-picker.md` |
| Combobox / 搜索型选择器 | `component-specs/combobox.md` |
| 批量操作条 | `component-specs/batch-actions-bar.md` |
| Admin 侧栏 | `component-specs/admin-sidebar.md` |
| Tenant 顶栏 | `component-specs/tenant-topnav.md` |
| 面包屑 | `component-specs/breadcrumb.md` |
| 加载 / 进度 / 操作反馈 | `component-specs/loading-progress.md` |
| 图表 / 统计数字 | `component-specs/chart-stat.md` |
| 上传 / 文件浏览 | `component-specs/upload-file-browser.md` |
| 分步向导 / 步骤条 | `component-specs/stepper.md` |
| 树形单选下拉（部门 / 分组 / 目录） | `component-specs/tree-select.md` |

### 2.4 QA 自查

- Admin 页面：`qa/admin-checklist.md`
- Tenant 页面：`qa/tenant-checklist.md`
- 组件规范：`qa/component-review-checklist.md`

## 3. 怎么选试点页面

先选 1 个 Admin 列表页 + 1 个 Tenant 业务页，不要一开始全量铺开。

Admin 试点建议：用户列表、实例列表、技能列表、审计日志、模型 / 通道列表。页面最好包含 PageHeader、搜索筛选、表格、分页、空态、加载态、错误态和操作列。

Tenant 试点建议：Agent 卡片列表、技能卡片列表、实例详情、用户端资源列表。页面最好能体现 Tenant 背景、业务卡片、Tenant 按钮、空态和简单筛选。

暂不建议第一批选择：复杂权限配置、多步骤向导、大量图表 Dashboard、Landing Hero、仍有较多设计待确认的页面。

## 4. Admin 列表页换皮步骤

标准结构：

```text
AdminPageHeader
SearchFilterBar
SurfaceCard
  BatchActionsBar optional
  Table
  Pagination
Empty / Loading / Error
```

执行顺序：

1. 判断页面属于 Admin，读取 `references/admin.md`。
2. 按 `references/page-recipes.md` 确认页面 recipe。
3. 页头对齐 `component-specs/page-header.md`。
4. 搜索、筛选、刷新对齐 `component-specs/search-filter-bar.md`。
5. 数据容器对齐 `component-specs/card-surface.md`。
6. 表格对齐 `component-specs/table.md`。
7. 分页对齐 `component-specs/pagination.md`。
8. 多选和跨页选择对齐 `component-specs/batch-actions-bar.md`。
9. 空 / 加载 / 错误态分别对齐 `empty-state.md` 和 `loading-progress.md`。
10. 用 `qa/admin-checklist.md` 自查。

注意：Admin 不使用 Tenant 胶囊卡片和用户端背景；不要新增旧品牌色、旧阴影、旧圆角。

## 5. Tenant 页面换皮步骤

标准结构视页面类型而定，通常是：

```text
TenantLayout / TopNav
Tenant page header or hero section
TenantCard / business content
Tabs / filters optional
Empty / Loading / Error
```

执行顺序：

1. 判断页面属于 Tenant，读取 `references/tenant.md`。
2. 先统一背景、TopNav、内容宽度和页面节奏。
3. 业务对象卡使用 Tenant 口径，不机械套 Admin `SurfaceCard`。
4. Tenant 按钮使用 `tenant-*` 视觉。
5. Tabs / Segment 先按端别分流，不新增 Text Switch。
6. 空态用具体标题和下一步说明，不只写“暂无数据”。
7. 用 `qa/tenant-checklist.md` 自查。

## 6. 宿主仓组件怎么映射

原则：优先复用宿主仓已有组件，但视觉必须对齐本包 spec。

```text
宿主仓已有组件 -> 保留逻辑和 API -> 按 component spec 调整视觉
宿主仓没有组件 -> 使用 spec 里的 Portable Fallback
```

不要把 demo 仓组件名当成唯一答案。demo 代码只用于理解结构、状态和视觉目标。

## 7. Portable Fallback 怎么用

每个 `component-specs/*.md` 都包含 `Portable Fallback`：

1. 先看宿主仓已有组件能否映射。
2. 不能映射时，用最小 React 结构还原。
3. 再不行，用 HTML/CSS fallback 先保视觉。

fallback 示例优先使用 CSS variable / token，不直接散写业务色 hex。

## 8. 设计冲突怎么处理

如果遇到颜色、圆角、阴影、组件分流、Tenant / Landing 待确认项：

1. 查 `references/conflict-log.md`（设计冲突与已确认裁决唯一活账本）。
2. 标注“暂选 / 后续进一步确认”的内容，不要写成不可变更最终裁决。
3. 无记录时回到设计侧确认，不由开发自行裁决。

## 9. 建议推进节奏

1. 第 1 天：选 Admin 试点页，按 migration map 完成组件映射。
2. 第 2 天：完成 Admin 试点换皮和 QA 自查。
3. 第 3 天：选 Tenant 试点页，验证 Tenant 卡片、按钮、Tabs、空态。
4. 第 4 天：把试点沉淀成宿主仓组件映射表。
5. 第 5 天：批量推广到同类页面。

每个试点页建议产出：截图或预览地址、使用了哪些 spec、宿主仓旧组件到新视觉的映射、QA checklist 结果、设计冲突记录。
