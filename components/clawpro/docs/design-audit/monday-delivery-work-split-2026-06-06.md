# ClawPro 周一交付分工建议

> 目标：在下周一前，交付一套能直接支持产品前端换皮的 portable design pack。
> 当前人力假设：你 + 1 位设计同事 + AI 协作。
> 日期：2026-06-06

## 1. 分工原则

- 不按“文档类型”平均分，而按“决策密度”和“依赖关系”分。
- 你负责高判断密度和收口型工作。
- 同事负责规则整理、模板落文和资料补齐。
- AI 负责提纲、初稿、格式统一、已有代码 / 规范对照、重复性整理。
- 周一之前不追求全量覆盖，优先把最影响换皮的内容打透。

## 2. 推荐分工

### 角色 A：你

建议你负责：

- 最终交付包结构拍板。
- `README.md` / `SKILL.md` 总入口口径。
- Admin / Tenant 端级规范的最终裁决。
- 高风险组件里最重要的 2 个：`table.md`、`card-surface.md`。
- migration map 的最终判断。
- 对外给前端 / 领导的口径统一。

原因：

- 这些内容最需要判断“什么是必须写死的，什么先不扩”。
- 一旦口径不稳，同事和 AI 会越写越散。

### 角色 B：设计同事

建议同事负责：

- 把你定下来的模板扩成规范正文。
- 补 `button.md`、`empty-state.md`、`page-header.md` 等组件 portable spec。
- 补 `qa/admin-checklist.md`、`qa/tenant-checklist.md`。
- 补 `assets-icons.md`、`icon-registry`、页面 recipe 中的例子和引用。
- 把 Figma 节点、设计截图、现有 demo 对应关系补完整。

原因：

- 这些内容适合并行推进。
- 规则边界清楚后，更多是整理和结构化表达。

### 角色 C：AI

建议 AI 负责：

- 根据现有代码和规范生成各组件 spec 初稿。
- 补 portable fallback 的 React / HTML/CSS 示例。
- 帮你们统一 markdown 结构。
- 扫描项目中现有引用，回填“Demo Repo Usage”与“Migration Rules”。
- 帮忙做 checklist 和 conflict log 初稿。

注意：

- AI 只能起草，不替代你们做高风险裁决。
- 所有涉及“最终视觉标准”的地方，必须由你或设计同事定。

## 3. 两人版最现实工作包

如果只有你和一个同事，建议按下面切：

### 你负责的 P0

1. 交付包骨架
2. 总入口文档
3. `foundation.md` 最终口径
4. `admin.md`、`tenant.md` 最终口径
5. `table.md`
6. `card-surface.md`
7. `migration-map.md`

### 同事负责的 P0

1. `components.md` 收口整理
2. `page-recipes.md` 收口整理
3. `button.md`
4. `empty-state.md`
5. `qa/admin-checklist.md`
6. `qa/tenant-checklist.md`
7. `assets-icons.md` / `icon-registry`

### AI 并行支持

1. 先按统一模板生成组件 spec 初稿
2. 从现有仓库抽引用和文件路径
3. 生成 portable fallback 示例
4. 帮忙统一术语和排版

## 4. 如果临时还能再拉 1 个碳基同事

那就改成三线并行：

### 你

- 决策收口
- 高风险组件 `table`、`card`
- 总入口

### 同事 1

- Admin / Tenant / page recipes / QA

### 同事 2

- Button / Empty / Header / Form controls 的 portable spec
- assets / icon registry
- portable fallback 示例整理

这样周一前能更稳地把交付包做成一套可发版本，而不只是草稿。

## 5. 时间安排建议

### 今天

- 先把交付骨架和统一模板定掉。
- 同步分工。
- 所有人按模板开写，不再各写各的。

### 明天

- 集中补 P0 组件 spec。
- 收口端级规范和 migration map。
- 第一轮互审，删掉重复和冲突。

### 周日前

- 补 QA checklist。
- 补 portable fallback 示例。
- 统一命名、引用、目录。
- 形成可发版本。

### 周一

- 面向前端和产品做交付说明。
- 明确“先看哪里、遇到冲突看哪里、不会用 demo 组件时怎么办”。

## 6. 输出标准

所有文档都尽量满足下面四点：

- 能被人直接读懂。
- 能被 AI 直接读取并执行。
- 不依赖你口头补充才能落地。
- 不强依赖当前 demo 仓实现细节。

## 7. 当前最值得你亲自抓的两件事

如果你精力有限，只亲自抓两件事，优先抓：

1. `table.md` 和 `card-surface.md` 的 portable spec
2. 整个交付包的总入口和迁移映射

因为这两件事最直接决定产品前端会不会觉得“这包东西真能用”。

