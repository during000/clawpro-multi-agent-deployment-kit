# [YYYY-MM-DD] <任务标题>

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feature/xxx` |
| 摘要 | 一句话描述 |
| 状态 | 待开始 / 进行中 / 已完成 / 已取消 |
| 创建日期 | YYYY-MM-DD |
| 负责人 |  |
| 预期完成 | YYYY-MM-DD |

---

## Progress

<!--
Progress 更新规则（AI 必读）：
1. 步骤完成时，**原地替换** `- [ ] N. <step>` 为 `- [x] N. <step> (<结果摘要>)`
2. 禁止插入新行或重复编号，每个编号 01-08 有且仅有一行
3. 结果摘要示例：`(15/15 passed)`、`(覆盖率 92%)`、`(全量通过)`
4. 同步更新下方「当前步骤」章节
-->

- [ ] 01. Clarify    → [01-clarify.md](./01-clarify.md)
- [ ] 02. Plan       → [02-plan.md](./02-plan.md)
- [ ] 03. Implement  → [03-implement.md](./03-implement.md)
- [ ] 04. UT         → [04-ut.md](./04-ut.md)
- [ ] 05. Docs       → [05-docs.md](./05-docs.md)
- [ ] 06. IT         → [06-it.md](./06-it.md)
- [ ] 07. Review     → [07-review.md](./07-review.md)
- [ ] 08. Commit     → [08-commit.md](./08-commit.md)

---

## 当前步骤

> 恢复会话时，优先读取此处指向的阶段文件。

- **步骤**：⏳ 01. Clarify
- **文件**：[01-clarify.md](./01-clarify.md)
- **上次更新**：YYYY-MM-DD HH:MM

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | | | | |
| 02. Plan | | | | |
| 03. Implement | | | | |
| 04. UT | | | | |
| 05. Docs | | | | |
| 06. IT | | | | |
| 07. Review | | | | |
| 08. Commit | | | | |

---

## 关键决策备忘

> **跨阶段共享的关键上下文**。仅记录影响后续步骤的决策，避免恢复时还要翻阅历史阶段文件。

-

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
|  |  |  |  |

---

## 文件索引

| 文件 | 产物 |
|------|------|
| [00-overview.md](./00-overview.md) | 任务总览（本文件） |
| [01-clarify.md](./01-clarify.md) | 需求澄清：背景、目标、范围、待确认问题 |
| [02-plan.md](./02-plan.md) | 方案设计：改动文件、调用链、测试用例、风险 |
| [03-implement.md](./03-implement.md) | 实现：关键细节、与 Plan 差异 |
| [04-ut.md](./04-ut.md) | 单元测试：用例、覆盖率、未覆盖行 |
| [05-docs.md](./05-docs.md) | 文档更新清单 |
| [06-it.md](./06-it.md) | 集成测试：构建部署、端到端验证、增量覆盖率 |
| [07-review.md](./07-review.md) | Code Review：问题与修复 |
| [08-commit.md](./08-commit.md) | Commit message 与提交前检查 |
