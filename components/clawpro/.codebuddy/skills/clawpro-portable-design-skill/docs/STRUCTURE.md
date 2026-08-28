# Structure

## 1. 为什么这份交付包要单独成一个文件夹

- 周一交付物需要可整体打包、转交、补充和归档。
- 人类、CodeBuddy、Codex、Cursor 等协作者都需要一套统一的入口。
- 把内容散落在现有项目多个路径下，会让产品前端和协作者不知道该看哪里。

因此这份 portable design pack 默认作为一个独立文件夹维护。

## 2. 目录说明

```text
clawpro-portable-design-skill/
├── README.md
├── HANDOFF.md
├── DEVELOPER-USAGE.md
├── INDEX.md
├── SKILL.md
├── STATUS.md
├── STRUCTURE.md
├── references/
├── component-specs/
├── portable/
├── tokens/
├── qa/
├── assets/
└── scripts/
```

## 3. 各目录职责

`README.md`

- 面向人类的总入口。

`SKILL.md`

- 面向 AI 的总入口。

`STATUS.md`

- 面向新对话续接的状态板。

`HANDOFF.md`

- 面向“怎么整体交付给别人”的交付说明。

`DEVELOPER-USAGE.md`

- 面向产品前端的换皮执行说明。

`DESIGN-AUDIT-PLAYBOOK.md`

- 面向设计审查的联合审计流程，连接本包规范与 impeccable。

`INDEX.md`

- 面向“这包里具体有哪些文件”的快速索引。

`references/`

- 存放稳定规则和端级规范。

`component-specs/`

- 存放高风险组件的 portable spec。
- 每个 spec 都必须包含 demo 仓用法与 portable fallback。

`portable/`

- 存放最小 HTML/CSS/React 示例。
- 用于“宿主仓没有当前组件体系”的情况。

`tokens/`

- 存放可被抽取和复用的设计常量。
- 当前已拆出 `colors.md`、`typography.md`、`radius-shadow.md`、`spacing.md`。

`qa/`

- 存放验收 checklist。

`assets/`

- 存放资产登记和元数据示例。

`scripts/`

- 存放检查脚本。

## 4. 当前周一版最小可交付范围

当前版本优先包含：

- `references/foundation.md`
- `references/admin.md`
- `references/tenant.md`
- `references/components.md`
- `references/page-recipes.md`
- `references/migration-map.md`
- `component-specs/table.md`
- `component-specs/card-surface.md`
- `component-specs/button.md`
- `component-specs/empty-state.md`
- `component-specs/page-header.md`
- `component-specs/dialog-drawer.md`
- `component-specs/popover-dropdown-menu.md`
- `component-specs/tabs.md`
- `component-specs/segment.md`
- `component-specs/form-controls.md`
- `component-specs/status-tag.md`
- `component-specs/badge.md`
- `component-specs/pagination.md`
- `component-specs/search-filter-bar.md`
- `component-specs/date-picker.md`
- `component-specs/combobox.md`
- `component-specs/batch-actions-bar.md`
- `component-specs/tenant-topnav.md`
- `component-specs/loading-progress.md`
- `component-specs/chart-stat.md`
- `component-specs/upload-file-browser.md`
- `component-specs/stepper.md`
- `component-specs/tree-select.md`
- `DEVELOPER-USAGE.md`
- `DESIGN-AUDIT-PLAYBOOK.md`
- `qa/admin-checklist.md`
- `qa/tenant-checklist.md`

## 5. 文件命名规则

- 组件 spec 用 kebab-case，例如 `card-surface.md`
- 端级规则保持简短直接，例如 `admin.md`
- checklist 用 `{scope}-checklist.md`
- token 文件按维度拆分，例如 `colors.md`、`radius-shadow.md`
