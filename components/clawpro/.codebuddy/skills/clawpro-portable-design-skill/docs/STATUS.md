# STATUS

> 这是一张续接状态板。新开对话时，先读这份文件，再继续推进，不依赖旧对话上下文。
> 最后更新时间：2026-06-06

## 1. 当前任务

目标：在下周一前交付一套 `clawpro-portable-design-skill`，用于支持产品前端在宿主仓中进行管理端、客户端、可能的落地页换皮。

当前目标不是做完整可安装组件库，而是完成一套可移交、可执行、可被人类和 AI 同时消费的 portable design pack。

## 2. 已完成

### 2.1 交付骨架

- 已创建 `.codebuddy/skills/clawpro-portable-design-skill/`
- 已有入口文件：`README.md`、`SKILL.md`、`STRUCTURE.md`
- 已有规则目录：`references/`
- 已有首批高风险组件 spec：`component-specs/`
- 已有 fallback 示例：`portable/`
- 已有 token、QA、assets、script 基础文件

### 2.2 首批 portable spec

- `component-specs/table.md`
- `component-specs/card-surface.md`
- `component-specs/button.md`
- `component-specs/empty-state.md`
- `component-specs/page-header.md`

### 2.3 第二批 portable spec

- `component-specs/dialog-drawer.md`
- `component-specs/tabs.md`
- `component-specs/segment.md`
- `component-specs/form-controls.md`

### 2.4 第三批 portable spec

- `component-specs/status-tag.md`
- `component-specs/badge.md`
- `component-specs/pagination.md`
- `component-specs/search-filter-bar.md`
- `component-specs/date-picker.md`

### 2.5 第四批 portable spec

- `component-specs/combobox.md`
- `component-specs/batch-actions-bar.md`

### 2.6 第五批 portable spec

- `component-specs/popover-dropdown-menu.md`
- `component-specs/tenant-topnav.md`
- `component-specs/loading-progress.md`
- `component-specs/chart-stat.md`
- `component-specs/upload-file-browser.md`

### 2.7 开发与审查说明

- `DEVELOPER-USAGE.md`
- `DESIGN-AUDIT-PLAYBOOK.md`
- 根目录 `PRODUCT.md` 已用于 impeccable product register 上下文

### 2.10 第六批 portable spec（2026-06-08）

- `component-specs/input-select.md`
- `component-specs/toast.md`
- `component-specs/selection-controls.md`
- `component-specs/tag-label.md`
- `component-specs/breadcrumb.md`
- `component-specs/tooltip.md`
- `component-specs/avatar.md`
- `component-specs/tree.md`

### 2.11 已有 spec 完整翻译更新（2026-06-08）

- `component-specs/empty-state.md`（基于 SKILL-GLOBAL §24 全面翻译）
- `component-specs/table.md`（基于 SKILL-GLOBAL §11.6 + §15 全面翻译）
- `component-specs/transfer.md`（基于 SKILL-GLOBAL §31 增强翻译）

### 2.12 页面级口径补充（2026-06-10）

- 已补充 Admin 页面与组件级局部预览的判定边界。
- 已明确：用户要求“管控端页面 / Admin 页面 / 列表页页面”时，默认必须包含 `AdminSidebar` / `AdminLayout`，不能只交付中间内容区。
- 已把该规则同步回 `SKILL.md`、`references/page-recipes.md`、`references/admin.md`、`qa/admin-checklist.md` 与 `references/conflict-log.md`。
- 已新增表格边界约束：表格只展示表格本身，标题 / 描述 / 操作按钮放表格外，并同步到 `table.md`、`data-table.md` 和 QA。

### 2.9 设计确认回写

- 已按确认表回写蓝灰文字 token、蓝灰描边 token、AdminSidebar 240px / 64px、Admin 默认无阴影与 TenantCard 默认轻投影。
- 已按确认表回写表格整体 12px、Tenant 业务卡 12px 边界、Tenant 表单控件双轨、Landing 周一交付边界。
- 已拆分并回写硬编码色治理：token 定义文件可有 hex，component spec / portable fallback 优先 token / CSS variable，历史页面 hex 作为存量债务。

### 2.13 设计治理补 spec（2026-06-26，§B.2 D5）

- 新增 `component-specs/stepper.md`（Stepper 分步向导步骤条，取值自 `client/src/components/ui/stepper.tsx`）。
- 新增 `component-specs/tree-select.md`（TreeSelect 树形单选下拉，button / filter-icon 两变体，取值自 `_internal/TreeSelectFilter.tsx` + `TableHeaderTreeFilter.tsx`）。
- 两份 spec 内的蓝/灰硬编码、6px 行圆角偏差已如实记录为「对齐缺口」，组件代码未改。
- `upload-file-browser.md` 收敛为纯上传职责、去与 `file-browser.md` 重叠并双向交叉引用（改名仍留单独 PR，见 `conflict-log.md`）。
- `transfer.md §2` 的 `TreeTransfer` 缺口标注强化为可执行出口（标 `needs-design-confirmation` + 记 conflict-log）。
- 同步更新 `MANIFEST.json` / `INDEX.md` / `SKILL.md` / `DEVELOPER-USAGE.md` / `STRUCTURE.md` / `conflict-log.md`；`verify-portable-skill.mjs` 通过。

## 3. 当前共识

- 交付重点是 portable design pack，不是周一前先做 npm 组件库。
- 规范不能只写“当前 demo 仓怎么用”，必须补“宿主仓里怎么还原”。
- 高风险组件优先级高于继续补厚总规范。
- 你希望规范最终收口到“推荐 token / 推荐口径”，而不是在文档中大量保留旧值做禁止对照。
- 如果遇到颜色、token、设计裁决不确定，必须先向你确认。
- 全局组件、管理端规范、以及任何需要设计人员拍板的问题，默认都以你为设计决策入口，不由我或 agent 私自拍板。

## 4. 待办优先级

### P0

- 周一交付包主体已完成，交付前保持 `verify-portable-skill.mjs` 通过。
- 对 `C-016` / `C-019` 这类暂选项保留后续确认入口，不包装成不可变更最终裁决。
- 若新增文件，必须同步 `MANIFEST.json`、`INDEX.md`、`STATUS.md`。

### P1

- 按你和同事的最终分工继续补全 landing / QA / assets 收口。
- 视时间为新增中风险 spec 补更多 standalone HTML/React 示例。

### P2

- 周一交付后，再讨论是否抽成通用 skill 包或组件库化。

## 5. 当前已知问题

- `check-design-usage.mjs` 可运行；当前会输出存量页面 warning，但不阻断 portable pack 校验。后续新增 spec 时继续以"只保留推荐口径"为准。
- `check-spec-symbols.mjs`（2026-06-11 新增）：扫 spec 内 `import { X } from "@/..."` 是否在 `client/src` 真实导出，揪 ghost identifier。**首次跑通**：在治理掉 Combobox / OpenClawCombobox 等幽灵引用后，目前对 SKILL.md / component-specs/* / references/* 报告 0 ghost。建议接到 PR CI / pre-commit；新增 spec 时跑一遍。
- `sync-tokens.mjs`（2026-06-11 新增）：从 client/src/index.css 同步 token 定义到 tokens/design-tokens.json；对 spec markdown 里的 `var(--xxx)` 引用做检查。首次跑发现 **19 个未定义 token**（--cp-surface-muted / --cp-text-secondary / --text-disabled 等旧名或改名的），已列出。建议逐个修正 spec 引用或核实是否真的删除了。
- `PRODUCT-USAGE.md`（2026-06-11 新增）：产品同事零门槛入口。包含"做列表页 / 适配 Tenant / FAQ"等实操指南，比 DEVELOPER-USAGE 更贴近产品端需求。
- 已治理：原 `Combobox` / `OpenClawCombobox` 在仓内不存在，spec 已统一替换为 `SearchableSelect`（见 `component-specs/combobox.md` alias 文档）；原 `ModelConfig.tsx` 内的 `Label` 未 import / `getCheckState` / `getDescendantIds` / `PROVIDER_VERSIONS` 死代码已清理。
- `C-016`、`C-019` 带"暂选 / 后续进一步确认"备注，规范中只能写当前口径，不能写成不可变更最终裁决。
- `D-03` 已由同事确认：删除 Tenant Text Switch，周一交付包不再定义该弱切换样式。
- 部分 component spec 还没有逐个配套 standalone HTML/React 示例页；目前 table / card / empty-state / search-filter-bar / date-picker / button / input-select / dialog-drawer / pagination / number-card / status-tag 已有示例（共 11 个），其他 spec 先以内联 Portable Fallback 为准。新增 standalone 示例时必须同步 `MANIFEST.json`。
- 近期发现一类执行偏差：AI 容易把 page recipe 当成"完整页面成品"而忽略 Admin 外层壳。此问题已通过 `C-012` 和相关规范补丁收口，后续按新口径执行。
- 近期又补了一类边界约束：AI 容易把表格标题区、描述和操作按钮塞进表格容器。此问题已通过 `C-014` 和相关规范补丁收口。

## 6. 如果新开对话，建议这样续接

把下面这段直接发给新对话：

```text
继续推进 ClawPro 周一交付包。

先读这些文件：
1. .codebuddy/skills/clawpro-portable-design-skill/STATUS.md
2. .codebuddy/skills/clawpro-portable-design-skill/README.md
3. .codebuddy/skills/clawpro-portable-design-skill/SKILL.md
4. .codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md

目标：继续把 `.codebuddy/skills/clawpro-portable-design-skill` 做成周一可交付版本。

重要约束：
- 当前优先做 portable design pack，不做完整组件库。
- 如果你不确定 token、颜色或设计裁决，必须先问我。
- 规范收口方向是“只保留当前推荐口径”，不要继续把旧值对照越写越多。
- 全局组件、管理端规范、以及任何需要设计拍板的问题，默认都先和我对齐，不要自行裁决。
```

## 7. 续接时的工作方式

- 新对话先基于 `STATUS.md` 更新计划。
- 先读当前 `git status`，确认是否有同事新改动。
- 再决定是继续本地做，还是分给多个 agent 并行。
