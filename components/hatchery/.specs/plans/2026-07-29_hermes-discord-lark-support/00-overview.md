# [2026-07-29] Hermes 支持 Discord 和 Lark 通道

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feat/hermes-discord-support` |
| 摘要 | 为 Hermes agent 类型新增 Discord 和 Lark 通道支持 |
| 状态 | 进行中 |
| 创建日期 | 2026-07-29 |
| 负责人 | kaypton |
| 预期完成 | 2026-07-29 |

---

## Progress

- [x] 01. Clarify (事后补录，跳过)
- [x] 02. Plan (事后补录)
- [x] 03. Implement (discord + lark 通道，7 文件改动)
- [x] 04. UT (TestSupportedChannelsByAgentType 全部 PASS)
- [x] 05. Docs (docs/API.md 已更新 discord user_id 参数)
- [x] 06. IT (补齐 discord/lark hermes 通道集成测试：TC-H4.6 / TC-H4.7)
- [x] 07. Review (自查通过)
- [x] 08. Commit (commit 2ac76948，已 push)

---

## 当前步骤

- **步骤**：✅ 全部完成（IT 已补齐集成测试）
- **文件**：`test/scripts/channel/test_discord_channel_hermes.py`、`test_lark_channel_hermes.py`
- **上次更新**：2026-07-30 16:20

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-29 11:51:00 | 2026-07-29 11:51:00 | 0 | 跳过：事后补录 |
| 02. Plan | 2026-07-29 11:51:00 | 2026-07-29 11:51:00 | 0 | 跳过：事后补录 |
| 03. Implement | 2026-07-29 11:51:00 | 2026-07-29 12:03:00 | 12m | rebase + merge + squash |
| 04. UT | 2026-07-29 12:04:00 | 2026-07-29 12:05:00 | 1m | 修复测试断言 |
| 05. Docs | 2026-07-29 12:05:00 | 2026-07-29 12:05:00 | 0 | API.md 已在实现中更新 |
| 06. IT | 2026-07-29 12:05:00 | 2026-07-30 16:20:00 | — | 补齐：新增 TC-H4.6 discord / TC-H4.7 lark hermes 通道集成测试 |
| 07. Review | 2026-07-29 12:05:00 | 2026-07-29 12:05:00 | 0 | 自查通过 |
| 08. Commit | 2026-07-29 12:03:00 | 2026-07-29 12:05:00 | 2m | 已 commit + push |

---

## 关键决策备忘

> **跨阶段共享的关键上下文**。

- Discord 通道：写入 `DISCORD_BOT_TOKEN` + `DISCORD_ALLOWED_USERS`（指定用户 ID，不支持通配符）到 `~/.hermes/.env`
- Lark 通道：复用 feishu 配置字段，`FEISHU_DOMAIN` 设为 `lark`，其余与 feishu 一致
- 分支合并策略：两个 feature 分支（discord / lark）各自 rebase 到 `Release/2026_07_28`，再合并后 squash 为单个 commit
- del_channel_hermes.sh 中 lark 删除逻辑修复了原分支的 bug（误用 `update_env` 而非 `delete_env`，且缺少 `ENV_FILE` 定义）

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | Discord 不支持通配符匹配用户 | 低 | 必须指定 `user_id`，脚本校验非空 |
| 2 | Lark 复用 feishu 配置字段，删除时需清理所有 FEISHU_* 变量 | 低 | del_channel_hermes.sh 已覆盖 |

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
