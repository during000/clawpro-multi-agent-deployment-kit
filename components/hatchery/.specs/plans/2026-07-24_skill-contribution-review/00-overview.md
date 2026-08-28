# [2026-07-24] 技能共建审核（Skill Contribution Review）

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feature/skill-contribution-review` |
| 摘要 | 允许员工提交技能和申请下架，管理员审核通过后自动上架/下架，支持企业技能库共建 |
| 状态 | 已完成 |
| 创建日期 | 2026-07-24 |
| 负责人 | AI + 用户 |
| 预期完成 | 2026-07-31 |

---

## Progress

<!--
Progress 更新规则（AI 必读）：
1. 步骤完成时，**原地替换** `- [ ] N. <step>` 为 `- [x] N. <step> (<结果摘要>)`
2. 禁止插入新行或重复编号，每个编号 01-08 有且仅有一行
3. 结果摘要示例：`(15/15 passed)`、`(覆盖率 92%)`、`(全量通过)`
4. 同步更新下方「当前步骤」章节
-->

- [x] 01. Clarify    → [01-clarify.md](./01-clarify.md) (通用审批表 review_requests + 双状态机 + 2 种 action_type，9 个问题全部确认)
- [x] 02. Plan       → [02-plan.md](./02-plan.md) (17 个文件改动、8 个新接口、35 个测试用例、8 个风险)
- [x] 03. Implement  → [03-implement.md](./03-implement.md) (2 新建 model/controller + 15 文件改动，go build + go vet 通过)
- [x] 04. UT         → [04-ut.md](./04-ut.md) (21/21 PASS)
- [x] 05. Docs       → [05-docs.md](./05-docs.md) (API.md 新增 8 个接口文档)
- [x] 06. IT         → [06-it.md](./06-it.md) (本地验证全通过：build/vet/UT/schema/openapi，K8s 端到端待 CI)
- [x] 07. Review     → [07-review.md](./07-review.md) (12/13 红线通过，安全基线全通过，无阻塞问题)
- [x] 08. Commit     → [08-commit.md](./08-commit.md) (feat commit, 4 新建 + 11 修改)

---

## 当前步骤

> 恢复会话时，优先读取此处指向的阶段文件。

- **步骤**：✅ 已完成
- **文件**：—
- **上次更新**：2026-07-27 16:00

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-24 15:50:00 | 2026-07-24 16:47:57 | ~58min | 含多轮方案讨论 |
| 02. Plan | 2026-07-27 11:41:30 | 2026-07-27 11:46:13 | ~5min | |
| 03. Implement | 2026-07-27 14:59:42 | 2026-07-27 15:17:33 | ~18min | |
| 04. UT | 2026-07-27 15:30:50 | 2026-07-27 15:35:53 | ~5min | 21/21 PASS |
| 05. Docs | 2026-07-27 15:43:06 | 2026-07-27 15:45:18 | ~2min | |
| 06. IT | 2026-07-27 15:48:00 | 2026-07-27 15:51:36 | ~4min | 本地验证全通过，K8s 端到端待 CI |
| 07. Review | 2026-07-27 15:53:05 | 2026-07-27 15:55:32 | ~2min | 12/13 红线通过，安全基线全通过 |
| 08. Commit | 2026-07-27 15:57:58 | 2026-07-27 16:00:00 | ~2min | feat commit |

---

## 关键决策备忘

> **跨阶段共享的关键上下文**。仅记录影响后续步骤的决策，避免恢复时还要翻阅历史阶段文件。

- **通用审批表**：`review_requests`，`resource_type` 区分 skill/mcp/rule，未来扩展低成本
- **双状态机**：Skill.Status（published/pending_review/offline）+ ContributionRequest.Status（pending/approved/rejected）
- **COS 正式路径**：员工提交即上传到 `{slug}/{slug}-{version}.zip`，与管理员一致
- **Skill 提交时即创建**：status=pending_review，审核通过翻转为 published，拒绝则软删除
- **互斥维度**：slug 级别，同一 slug 只允许一个 pending 申请
- **下架权限**：Skill.UploaderID 字段校验，只有上传者可申请下架自己的技能
- **审核不支持修改元数据**（一期）
- **管理员直接上传/删除不受审核机制约束**

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
