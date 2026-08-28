# Design Review Playbook

> 用途：把 `clawpro-portable-design-skill` 的 ClawPro 专项规范，与 `impeccable` 的通用产品 UI 审查能力结合，用于产品 demo、开发换皮、宿主仓实现、设计仓库页面和 PR / MR 的统一设计审查与验收。

## 1. 审查定位

联合审查有两个层次：

1. ClawPro 专项一致性：由本交付包负责，判断页面是否符合 ClawPro token、端别、组件规范和 portable fallback 要求。
2. 通用产品 UI 质量：由 `impeccable` 的 product register 和 critique / audit / polish 思路负责，判断页面是否清晰、可用、可访问、可信。

优先级：

```text
clawpro-portable-design-skill
> references/conflict-log.md
> impeccable 通用 UI 建议
```

如果 `impeccable` 的建议与 ClawPro 已确认 token、圆角、阴影、组件分流冲突，以 ClawPro 规范为准。

## 2. 适用场景

| 场景 | 目的 | 输出 |
|---|---|---|
| 产品 demo 页面审查 | 判断 demo 页面是否真正符合 ClawPro 新规范 | 页面审计报告 + 修复建议 |
| 开发换皮过程评审 | 在开发实现中途判断方向是否偏离，避免后期返工 | 阶段性 P0 / P1 / P2 问题 |
| 开发换皮验收 | 判断宿主仓页面是否达到交付标准 | 通过 / 带条件通过 / 不通过结论 |
| 设计仓库反向审查 | 用现有页面验证 portable spec 是否完整 | 页面问题 + spec 回写建议 |
| 产品前端宿主仓审查 | 检查宿主仓组件映射和 fallback 是否符合 ClawPro | 实现偏差清单 |
| PR / MR 设计走查 | 防止新增页面继续偏离规范 | 轻量 checklist |

## 3. 审查模式

| 模式 | 使用时机 | 规则 |
|---|---|---|
| Audit only | 评审会前、设计走查、反向审查 | 只输出报告，不改代码 |
| Acceptance | 开发换皮完成后验收 | P0 不通过，P1 可带条件通过，P2 记录 |
| Fix guided | 设计侧确认后修复 | 只修 P0 和高价值 P1，不自行裁决待确认项 |

默认使用 `Audit only`；除非明确要求修复，否则不要改代码。

## 4. 审查前必读

每次审查页面前，先读：

1. `PRODUCT.md`
2. `.codebuddy/skills/clawpro-portable-design-skill/README.md`
3. `.codebuddy/skills/clawpro-portable-design-skill/SKILL.md`
4. `.codebuddy/skills/clawpro-portable-design-skill/STATUS.md`
5. `.codebuddy/skills/clawpro-portable-design-skill/references/foundation.md`
6. `.codebuddy/skills/clawpro-portable-design-skill/references/components.md`
7. `.codebuddy/skills/clawpro-portable-design-skill/references/conflict-log.md`

再按页面端别读取：

- Admin：`references/admin.md`
- Tenant：`references/tenant.md`
- Landing：`references/landing.md`
- 页面结构：`references/page-recipes.md`
- 迁移映射：`references/migration-map.md`

涉及具体组件时，读取对应 `component-specs/*.md`。

## 5. 适合先审或验收的页面

第一批不要全仓扫描，先抽样：

1. 一个 Admin 列表页。
2. 一个 Admin 表单 / 配置页。
3. 一个 Tenant 卡片列表页。
4. 一个 Tenant 详情页。
5. 一个 Dashboard / 监控页。

每类页面先做报告，不直接改代码。等设计侧确认 P0 / P1 后，再进入修复。

## 4. 审查步骤

### 4.1 判断端别和页面 recipe

先回答：

- 页面是 Admin / Tenant / Landing / Mixed？
- 属于列表页、表单页、详情页、Dashboard、设置页，还是特殊页面？
- 页面是否误用了另一个端的规则？

### 4.2 识别高风险组件

列出页面实际使用的组件，并映射到 spec：

- PageHeader
- SearchFilterBar
- SurfaceCard / TenantCard
- Table
- Pagination
- BatchActionsBar
- Button
- Dialog / Drawer
- Popover / Dropdown
- Tabs / Segment
- Form Controls
- StatusTag / Badge
- EmptyState
- Loading / Progress
- Chart / Stat
- Upload / File Browser
- Navigation / Sidebar

### 4.3 ClawPro 专项检查

重点检查：

- 是否新增旧色值、旧渐变、旧品牌蓝紫体系。
- 是否手写卡片、重阴影、错误圆角。
- 是否 Admin / Tenant 组件分流错误。
- 是否表格、筛选、分页、批量操作不成体系。
- 是否按钮变体错误，或用 `outline` 伪装业务按钮。
- 是否缺空态、加载态、错误态、无权限态。
- 是否使用未登记图标或 emoji 图标。
- 是否需要 portable fallback，但文档或代码没有说明。

### 4.4 impeccable 通用产品 UI 检查

重点检查：

- 视觉层级是否清楚。
- 信息密度是否服务任务，而不是制造噪音。
- 文字对比度和可读性是否达标。
- 交互状态是否完整：default、hover、focus、active、disabled、loading、error。
- 覆盖层是否被 overflow 裁剪。
- 空态是否教用户下一步，而不是只写“暂无数据”。
- 动效是否服务状态变化，并支持 reduced motion。
- 是否有泛 SaaS、AI 味、过度装饰、重阴影、玻璃拟态、无意义渐变等反模式。

## 5. 问题分级

| 级别 | 含义 | 示例 |
|---|---|---|
| P0 | 阻塞交付或会误导开发 | 端别混用、旧品牌色扩散、危险操作无确认、关键状态缺失 |
| P1 | 明显影响一致性或使用体验 | 表格密度不统一、筛选工具条散乱、按钮变体错误、空态文案过弱 |
| P2 | 可后续优化 | 局部间距节奏、微文案、轻量动效、个别图标替换 |

验收口径：

- 存在 P0：不通过。
- 无 P0，但存在关键 P1：带条件通过，需约定修复时间。
- 仅 P2：通过，记录后续优化。

## 8. 审查报告模板

```md
# Design Audit: 页面名称

## 1. 基本信息

- 路由 / 文件：
- 端别：Admin / Tenant / Landing
- 页面类型：列表页 / 表单页 / 详情页 / Dashboard / 设置页
- 读取的 specs：

## 2. 总体结论

- 是否符合 ClawPro portable design pack：是 / 部分 / 否
- 是否存在 P0：有 / 无
- 审查场景：产品 demo / 开发换皮过程 / 开发换皮验收 / 设计仓库反向审查 / 宿主仓审查 / PR 走查
- 审查模式：Audit only / Acceptance / Fix guided
- 验收结论：通过 / 带条件通过 / 不通过 / 不适用
- 是否建议进入修复：是 / 否

## 3. ClawPro spec 对齐

| 区域 | 对应 spec | 结论 | 问题 |
|---|---|---|---|
| PageHeader | `component-specs/page-header.md` |  |  |
| Table | `component-specs/table.md` |  |  |

## 4. impeccable 通用 UI 审查

- 视觉层级：
- 可读性 / 对比度：
- 交互状态：
- 空 / 加载 / 错误态：
- 可访问性：
- 反模式：

## 5. P0 / P1 / P2 问题

### P0

1. 问题：
   - 依据：
   - 影响：
   - 建议：

### P1

### P2

## 6. 修复建议

- 建议改动文件：
- 建议优先级：
- 是否需要设计确认：
```

## 9. 可直接使用的 prompt

### 9.1 单页只审不改

```text
请结合 clawpro-portable-design-skill 和 impeccable，对这个页面做设计审查：<页面路径或路由>。

要求：
1. 以 clawpro-portable-design-skill 为主规范。
2. 用 impeccable product register 做通用 UI / UX / a11y / 反模式审查。
3. 先判断端别和页面 recipe。
4. 列出涉及的 component-specs。
5. 输出 P0 / P1 / P2 问题。
6. 每个问题说明依据和修复建议。
7. 先不要改代码。
```

### 9.2 多页抽样审查

```text
请从当前仓库中选 3 个典型页面，结合 clawpro-portable-design-skill 和 impeccable 做设计审查。

优先覆盖：Admin 列表页、Tenant 卡片列表页、Dashboard / 监控页。
请说明每个页面属于产品 demo、开发换皮验收、设计仓库反向审查还是宿主仓实现审查。
先输出审计报告，不改代码。
```

### 9.3 开发换皮验收

```text
请结合 clawpro-portable-design-skill 和 impeccable，对这个开发换皮页面做验收：<页面路径或路由>。

要求：
1. 输出通过 / 带条件通过 / 不通过结论。
2. P0 判定为不通过。
3. P1 判定为带条件通过，并列出必须修复项。
4. P2 只记录后续优化。
5. 先不要改代码。
```

### 9.4 审查后修复

```text
请根据上一轮审计报告，只修复 P0 和高优先级 P1。
要求：遵守 clawpro-portable-design-skill，不自行裁决 C-016 / C-019 这类暂选项。
```

## 10. 不要做的事

- 不要用 `impeccable` 的通用建议覆盖 ClawPro 已确认规范。
- 不要一次性全仓改样式。
- 不要边审边大规模重构组件库。
- 不要把历史页面 hardcoded style 当作新增例外。
- 不要让开发自行裁决 token、圆角、阴影、端别分流。

## 11. 建议落地节奏

1. 先完成 3 个页面审查报告，覆盖产品 demo、开发换皮验收和设计仓库反向审查中的至少两类场景。
2. 设计侧确认 P0 / P1 是否成立。
3. 开发换皮页面按验收口径给出通过 / 带条件通过 / 不通过。
4. 只修 P0 和高价值 P1。
5. 把修复经验回写到 `component-specs/` 或 `references/conflict-log.md`。
6. 再扩大到同类页面批量审查。
