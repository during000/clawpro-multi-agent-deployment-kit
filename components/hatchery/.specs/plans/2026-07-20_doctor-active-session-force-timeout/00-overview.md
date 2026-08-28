# 2026-07-20 龙虾医生诊断会话兜底销毁缺陷修复

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `bugfix/doctor-active-session-force-timeout` |
| 摘要 | 修复龙虾医生诊断会话在 mtime/CVM/TAT 探测持续失败时永久卡在 active 状态、触发用户维度互斥锁的兜底缺陷（TAPD bug 1020422209160782882） |
| 状态 | 已完成 |
| 创建日期 | 2026-07-20 |
| 负责人 | 杨磊 |
| 预期完成 | 2026-07-20 |

---

## Progress

- [x] 01. Clarify    → [01-clarify.md](./01-clarify.md) (用户已在对话中确认根因分析与方案边界，无异议)
- [x] 02. Plan       → [02-plan.md](./02-plan.md) (改动清单+5条UT用例设计完成)
- [x] 03. Implement  → [03-implement.md](./03-implement.md) (新增8行兜底分支，go build通过)
- [x] 04. UT         → [04-ut.md](./04-ut.md) (新增4个用例全通过，task+controller全量回归通过)
- [x] 05. Docs       → [05-docs.md](./05-docs.md) (无API/schema/i18n变更，无需更新)
- [x] 06. IT         → [06-it.md](./06-it.md) (无新增API，现有IT契约测试已覆盖，无需新增)
- [x] 07. Review     → [07-review.md](./07-review.md) (无问题，符合项目红线)
- [x] 08. Commit     → [08-commit.md](./08-commit.md) (已通过gongfeng MCP提交到远端bugfix分支)

---

## 当前步骤

> 恢复会话时，优先读取此处指向的阶段文件。

- **步骤**：✅ 全部完成
- **文件**：[08-commit.md](./08-commit.md)
- **上次更新**：2026-07-20

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-20 11:00:00 | 2026-07-20 11:05:00 | 5min | 分析已在对话前置阶段完成，用户确认无异议 |
| 02. Plan | 2026-07-20 11:05:00 | 2026-07-20 11:15:00 | 10min | |
| 03. Implement | 2026-07-20 11:15:00 | 2026-07-20 11:25:00 | 10min | |
| 04. UT | 2026-07-20 11:25:00 | 2026-07-20 11:35:00 | 10min | |
| 05. Docs | 2026-07-20 11:35:00 | 2026-07-20 11:36:00 | 1min | 无需变更 |
| 06. IT | 2026-07-20 11:36:00 | 2026-07-20 11:37:00 | 1min | 无需新增 |
| 07. Review | 2026-07-20 11:37:00 | 2026-07-20 11:40:00 | 3min | |
| 08. Commit | 2026-07-20 11:40:00 | 2026-07-20 11:45:00 | 5min | 已提交到远端 bugfix 分支 |

---

## 关键决策备忘

- TAPD bug: https://tapd.woa.com/tapd_fe/20422209/bug/detail/1020422209160782882
- 根因：`task/doctor_cleanup.go` 的 `cleanupDoctorSessions` 对 active 会话的 12h 超时判断完全依赖远端探测成功
  （`GetDoctorSessionMtimeFn` 通过 TAT 读取会话文件 mtime），一旦 `DoctorInstanceID`/`doctorInst`/`InstanceId`
  异常或远端探测持续失败，会话被 `continue` 跳过，永久卡在 `active`，触发 `HandleDoctorStart` 里按用户维度的
  互斥锁，导致用户无法创建新的诊断会话。
- `controller/doctor.go` 的 `CleanupDoctorSession`（ending 阶段）本身已经保证 CVM 销毁失败时也会把状态推进到
  `ended`，不会卡住；问题只在于 active 会话未必能进入 ending 流程。
- 方案（已与用户确认，改动最小/侵入最低）：在 `cleanupDoctorSessions` 的 active 会话循环最前面新增一个不依赖
  任何远端调用的绝对兜底判断——只要 `time.Since(s.CreatedAt) > timeout`，无论后续 CVM/TAT/mtime 探测是否正常，
  强制调用 `endDoctorSession` 推进到 `ending`。不改现有分支顺序和语义，只新增一个前置分支。
- 影响范围：仅 `task/doctor_cleanup.go`，不涉及 API / DB schema / i18n。

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | 新增兜底分支误伤"探测正常但业务确实还在使用"的会话 | 低 | timeout 仍为既有 12h 常量，与现有 mtime 判断超时阈值一致，不缩短用户可用时长 |
| 2 | 并发场景下 `endDoctorSession` 被重复调用 | 低 | `endDoctorSession` 仅做状态置为 `ending` 的幂等 Update，重复调用无副作用 |

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
