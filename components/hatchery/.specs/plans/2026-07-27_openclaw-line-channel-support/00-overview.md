# [2026-07-27] 新增 LINE IM 通道支持（仅 Hermes Agent）

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feat/openclaw-support-line` |
| 摘要 | 为 Hermes Agent 新增 LINE Messaging API 通道支持，包含反向代理路由、安全组规则、设置/删除脚本、API 文档全链路 |
| 状态 | 已完成 |
| 创建日期 | 2026-07-27 |
| 负责人 | kaypton |
| 预期完成 | 2026-07-27 |

---

## Progress

- [x] 01. Clarify    → [01-clarify.md](./01-clarify.md) (需求澄清：LINE IM 通道接入方案)
- [x] 02. Plan       → [02-plan.md](./02-plan.md) (方案设计：14 个文件、测试用例设计)
- [x] 03. Implement  → [03-implement.md](./03-implement.md) (实现：commit e47c6849 + 87427ffb；增量：config.yaml gateway.platforms.line 支持)
- [x] 04. UT         → [04-ut.md](./04-ut.md) (单元测试：8 个新增用例全部通过，目标行已覆盖)
- [x] 05. Docs       → [05-docs.md](./05-docs.md) (文档更新：API.md 增量更新)
- [x] 06. IT         → [06-it.md](./06-it.md) (集成测试：新增 test_line_channel_hermes.py + helper 变更)
- [ ] 07. Review     → [07-review.md](./07-review.md)
- [x] 08. Commit     → [08-commit.md](./08-commit.md) (commit e47c6849 + b887f78d)

---

## 当前步骤

> 恢复会话时，优先读取此处指向的阶段文件。

- **步骤**：⏳ 07. Review
- **文件**：[07-review.md](./07-review.md)
- **上次更新**：2026-07-31 11:35

> 注：本任务功能已在分支 `feat/openclaw-support-line` 上开发完成（commit e47c6849 + b887f78d），
> SOP 文档为事后补写。Clarify / Plan / Implement / UT / Docs / Commit 阶段均标记为已完成（补写），
> IT 和 Review 阶段待后续执行。

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-27 19:27:00 | 2026-07-27 19:30:00 | 3min | 补写：功能已开发完成 |
| 02. Plan | 2026-07-27 19:27:00 | 2026-07-27 19:31:00 | 4min | 补写：功能已开发完成 |
| 03. Implement | 2026-07-27 19:27:00 | 2026-07-27 19:32:00 | 5min | 补写：对应 commit e47c6849 + 87427ffb；2026-07-30 增量更新 config.yaml 支持 |
| 04. UT | 2026-07-27 19:27:00 | 2026-07-27 19:33:00 | 6min | 补写：commit b887f78d |
| 05. Docs | 2026-07-27 19:27:00 | 2026-07-27 19:34:00 | 7min | 补写：API.md 已更新 |
| 06. IT | 2026-07-31 11:20:00 | 2026-07-31 11:35:00 | 15min | 新增 test_line_channel_hermes.py + helper 变更 |
| 07. Review | | | | 待执行 |
| 08. Commit | 2026-07-27 19:27:00 | 2026-07-27 19:35:00 | 8min | 补写：commit e47c6849 + b887f78d |

---

## 关键决策备忘

- LINE 通道仅对 Hermes Agent 开放（`agentTypeChannelWhitelist[AgentTypeHermes]`），OpenClaw/ACE 不支持
- LINE 通道标记为海外范围（`ChannelScopeOverseas`），仅海外站点可见
- LINE 代理路由端口 8646，路径 `/line/webhook`，支持 GET+POST（Webhook 验证需要 GET）
- 安全组规则 `allow_agent_proxy_line` 通过 `agent_proxy_line_enable` 条件动态控制
- 设置脚本写入 4 个环境变量：`LINE_CHANNEL_ACCESS_TOKEN`、`LINE_CHANNEL_SECRET`、`LINE_PORT`、`LINE_ALLOW_ALL_USERS`
- 设置/删除脚本额外维护 `~/.hermes/config.yaml` 的 `gateway.platforms.line.enabled` 配置项（设置时写入 true，删除时移除 line 块），使用 yq / Python+PyYAML / Python 纯文本三层 fallback 保证可靠性

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | LINE Webhook 需要 GET 请求验证，Teams 仅 POST，需确保代理路由方法检查正确 | 低 | 已在 `HandleAgentProxy` 中对 LINE 单独处理 GET+POST |
| 2 | 安全组规则 8646 端口暴露 | 低 | 通过 `agent_proxy_line_enable` 条件控制，仅配置 LINE 的实例才放通 |

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
