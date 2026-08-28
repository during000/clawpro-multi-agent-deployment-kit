# [2026-07-31] 废弃 list-models 探活，全部采用 chat 探活

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `bugfix/connectivity` |
| 摘要 | 废弃 list-models 探活方式，全部采用 chat 探活，确保 model id 错误时能被检测出来 |
| 状态 | 已完成 |
| 创建日期 | 2026-07-31 |
| 负责人 |  |
| 预期完成 | 2026-07-31 |

---

## Progress

- [x] 01. Clarify (废弃 list-models 探活，全部采用 chat 探活)    → [01-clarify.md](./01-clarify.md)
- [x] 02. Plan (改动 4 个核心文件 + 3 个测试文件 + 1 个文档)       → [02-plan.md](./02-plan.md)
- [x] 03. Implement (删除 CheckConnectivity 接口及实现，handleModelConnectivity 改用 chat 探活)  → [03-implement.md](./03-implement.md)
- [x] 04. UT (相关测试全部 PASS)         → [04-ut.md](./04-ut.md)
- [x] 05. Docs (API.md 两处补充 chat 探活说明)       → [05-docs.md](./05-docs.md)
- [x] 06. IT (无 API 契约变更，无需新增 IT 用例)         → [06-it.md](./06-it.md)
- [x] 07. Review (审查通过，无阻塞性问题)     → [07-review.md](./07-review.md)
- [x] 08. Commit (fix(controller): deprecate list-models probe, use chat probe only)     → [08-commit.md](./08-commit.md)

---

## 当前步骤

- **步骤**：✅ 已完成
- **文件**：[08-commit.md](./08-commit.md)
- **上次更新**：2026-07-31 16:30

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-31 16:08:45 | 2026-07-31 16:09:57 | 1m12s | |
| 02. Plan | 2026-07-31 16:09:57 | 2026-07-31 16:11:25 | 1m28s | |
| 03. Implement | 2026-07-31 16:11:25 | 2026-07-31 16:17:15 | 5m50s | |
| 04. UT | 2026-07-31 16:17:15 | 2026-07-31 16:27:58 | 10m43s | |
| 05. Docs | 2026-07-31 16:27:58 | 2026-07-31 16:28:37 | 39s | |
| 06. IT | 2026-07-31 16:28:37 | 2026-07-31 16:28:54 | 17s | |
| 07. Review | 2026-07-31 16:28:54 | 2026-07-31 16:29:41 | 47s | |
| 08. Commit | 2026-07-31 16:29:41 | 2026-07-31 16:30:09 | 28s | |

---

## 关键决策备忘

1. 完全删除 `CheckConnectivity` 方法及其所有实现（openai/anthropic），不保留、不标记 deprecated
2. `resolveConnectivityArgs` 中 modelID 为空的校验已存在，返回参数错误，无需修改
3. `handleModelConnectivity` 中 `probe` 变量仍保留用于日志，但值固定为 "chat"
4. `handleModelConnectivity` 函数注释需更新（不再有两次 RTT）

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | 删除接口方法可能影响其它调用点 | 中 | 已确认仅 handleModelConnectivity 调用 |
| 2 | 测试用例改动范围较大 | 中 | 逐个适配所有 list-models 相关测试 |

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
