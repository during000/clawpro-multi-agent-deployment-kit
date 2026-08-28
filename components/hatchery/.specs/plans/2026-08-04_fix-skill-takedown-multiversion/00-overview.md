# [2026-08-04] 修复技能多版本下架申请挂错版本

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feature/fix-skill-takedown-multiversion` |
| 摘要 | 修复多版本技能下架申请挂到旧版本 id，导致管理端技能列表看不到 pending，且审核通过只下架单版本的问题 |
| 状态 | 进行中 |
| 创建日期 | 2026-08-04 |
| 负责人 | AI + 用户 |
| 预期完成 | 2026-08-04 |
| Base | `origin/Release/2026_08_04` |

---

## Progress

<!--
Progress 更新规则（AI 必读）：
1. 步骤完成时，**原地替换** `- [ ] N. <step>` 为 `- [x] N. <step> (<结果摘要>)`
2. 禁止插入新行或重复编号，每个编号 01-08 有且仅有一行
3. 结果摘要示例：`(15/15 passed)`、`(覆盖率 92%)`、`(全量通过)`
4. 同步更新下方「当前步骤」章节
-->

- [x] 01. Clarify    → [01-clarify.md](./01-clarify.md) (6 点决策全部确认：整 slug 下架 + pending 按 slug + 代码兼容旧 pending)
- [x] 02. Plan       → [02-plan.md](./02-plan.md) (3 处代码修复 + UT/IT 用例；无 schema 变更)
- [x] 03. Implement  → [03-implement.md](./03-implement.md) (takedown/approve/pending_review 三处已改，build 通过)
- [x] 04. UT         → [04-ut.md](./04-ut.md) (新增多版本 5/5 PASS + 既有下架/审核回归 PASS)
- [x] 05. Docs       → [05-docs.md](./05-docs.md) (API.md 明确整 slug 下架与 pending 按 slug)
- [x] 06. IT         → [06-it.md](./06-it.md) (IT-M1 脚本已补；远程 E2E 待部署后跑)
- [x] 07. Review     → [07-review.md](./07-review.md) (PASS：Plan 全覆盖，UT 绿，可 Commit)
- [x] 08. Commit     → [08-commit.md](./08-commit.md) (fix 多版本下架 pending/offline)

---

## 当前步骤

> 恢复会话时，优先读取此处指向的阶段文件。

- **步骤**：✅ SOP 完成（push / MR 按需另开）
- **文件**：[08-commit.md](./08-commit.md)
- **上次更新**：2026-08-04 19:51

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-08-04 17:51:47 | 2026-08-04 18:08:00 | ~16min | 含用户确认 6 点决策 |
| 02. Plan | 2026-08-04 19:29:45 | 2026-08-04 19:30:52 | ~1min | 方案对齐 offline 语义 |
| 03. Implement | 2026-08-04 19:32:12 | 2026-08-04 19:33:44 | ~1.5min | 三处核心逻辑 |
| 04. UT | 2026-08-04 19:36:48 | 2026-08-04 19:38:16 | ~1.5min | 5 个新用例 + 回归 |
| 05. Docs | 2026-08-04 19:39:11 | 2026-08-04 19:40:06 | ~1min | API.md 语义澄清 |
| 06. IT | 2026-08-04 19:43:45 | 2026-08-04 19:44:57 | ~1min | IT-M1 脚本；远程待部署 |
| 07. Review | 2026-08-04 19:48:56 | 2026-08-04 19:50:30 | ~1.5min | PASS |
| 08. Commit | 2026-08-04 19:51:09 | 2026-08-04 19:51:22 | ~13s | 本地 1 commit ahead |

---

## 关键决策备忘

> **跨阶段共享的关键上下文**。仅记录影响后续步骤的决策，避免恢复时还要翻阅历史阶段文件。

- Base 分支：`Release/2026_08_04`（已发布含共建审核功能）
- 线上复现（diagram-imagexxx）：takedown pending 的 `resource_id=4887`(v1.0.0)，列表展示最新版 `4888`(v1.0.1) → `pending_review` 为空
- 产品语义：下架针对整个 slug（与 `POST /admin/skills/offline` 一致）
- **已确认修复策略**：
  1. 整 slug 下架（approve 批量 offline）
  2. `pending_review` 按 slug 关联（兼容旧 resource_id）
  3. 新 takedown 的 `resource_id` 绑最新 published
  4. 不做历史 SQL 修数
  5. 归属：存在「本人 + published」版本即可申请

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | 已存在的 pending takedown 仍挂旧 resource_id | 中 | 展示侧按 slug 关联；通过时按 slug 整技能下架 |
| 2 | 改动 approve 语义可能影响已有单测 | 低 | 同步更新 UT |

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
