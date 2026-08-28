# [2026-07-14] 本地 agent get-config 接口（CLS 公网配置拉取）

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。

> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feature/local-agent-get-config` |
| 摘要 | 新增 `GET /local-agent/get-config` 接口，本地 agent 拉取 CLS 公网上报配置（endpoint/topic_id 实时查 + secret 落 `local_agent_cls_credentials` 表按租户隔离） |
| 状态 | 进行中 |
| 创建日期 | 2026-07-14 |
| 负责人 | alexwhwang |
| 预期完成 | 2026-07-14 |

---

## Progress

<!--
Progress 更新规则（AI 必读）：
1. 步骤完成时，**原地替换** `- [ ] N. <step>` 为 `- [x] N. <step> (<结果摘要>)`
2. 禁止插入新行或重复编号，每个编号 01-08 有且仅有一行
3. 结果摘要示例：`(15/15 passed)`、`(覆盖率 92%)`、`(全量通过)`
4. 同步更新下方「当前步骤」章节
-->

- [x] 01. Clarify    → [01-clarify.md](./01-clarify.md) (规格清晰，6 项待确认，install_cmd/update_cmd 先留空串常量待填)
- [x] 02. Plan       → [02-plan.md](./02-plan.md) (10 文件改动清单+调用链+10单测设计)
- [x] 03. Implement  → [03-implement.md](./03-implement.md) (10 文件落地，build/vet 通过)
- [x] 04. UT         → [04-ut.md](./04-ut.md) (controller 9 + model 2 单测全绿，无回归)
- [x] 05. Docs       → [05-docs.md](./05-docs.md) (API.md 已更新，核对无遗漏)
- [x] 06. IT         → [06-it.md](./06-it.md) (集成测试 5 用例，红线 13 满足)
- [x] 07. Review     → [07-review.md](./07-review.md) (红线全核对，无阻断)
- [x] 08. Commit     → [08-commit.md](./08-commit.md) (commit ec39ab79，推送远端)

---

## 当前步骤

> 恢复会话时，优先读取此处指向的阶段文件。

- **步骤**：✅ Done
- **文件**：[08-commit.md](./08-commit.md)
- **上次更新**：2026-07-14 17:10

---

## 时间记录

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-07-14 15:05:00 | 2026-07-14 15:12:00 | 7m | 规格清晰，install_cmd/update_cmd 先留空串常量待用户填 |
| 02. Plan | 2026-07-14 15:12:00 | 2026-07-14 15:18:00 | 6m | 10 文件改动清单+调用链+测试设计 |
| 03. Implement | 2026-07-14 15:16:00 | 2026-07-14 15:35:00 | 19m | 10 文件落地，build/vet 通过 |
| 04. UT | 2026-07-14 15:38:00 | 2026-07-14 16:10:00 | 32m | controller 9 + model 2 单测全绿 |
| 05. Docs | 2026-07-14 16:10:00 | 2026-07-14 16:25:00 | 15m | API.md 已更新 + 红线核对 |
| 06. IT | 2026-07-14 16:25:00 | 2026-07-14 16:40:00 | 15m | 集成测试 5 用例 + helper 封装 |
| 07. Review | 2026-07-14 16:40:00 | 2026-07-14 16:55:00 | 15m | 红线 1-9 全核对 |
| 08. Commit | 2026-07-14 16:55:00 | 2026-07-14 17:10:00 | 15m | commit ec39ab79 + 推送 |

---

## 关键决策备忘

> **跨阶段共享的关键上下文**。仅记录影响后续步骤的决策，避免恢复时还要翻阅历史阶段文件。

- **规格来源**：iWiki 技术方案 `https://iwiki.woa.com/p/4022150701` §5.A.4（已定稿，定义阶段完成）。注意：本任务以**今天 iWiki 定稿版**为准，**不是** 2026-07-13 的 `output/cls_config_api_5a4.md` 草稿版（草稿版含 `agent_type` 必填 / `service_name` / `config_type` 包裹层 / 密钥加密 / `(config_type,agent_type)` 索引 —— 均已被定稿版废弃，勿复用）。
- **接口形态**：`GET /local-agent/get-config?config_type=cls`。响应扁平 `{cls:{endpoint,topic_id,secret_id,secret_key,user_id,user_name}}`，无 `config_type` 包裹层、无 `agent_type` 参数、无 `service_name`。
- **鉴权前置**：复用 `ensureLocalAgentAllowed`（两层白名单：① `feature_allowlist` type=local-agent ② `SiteConfig.LocalAgentEnabled`），拒绝返 403 `openclaw.local_agent.not_allowed`。
- **secret 存储**：独立表 `local_agent_cls_credentials`，按 `(identifier, config_type)` 唯一索引、按租户隔离、**明文落库**（不加密）。运维按租户 SQL 写入，不在调用链里写。
- **固定值字段（用户已确认，2026-07-14 20:47 定稿，2026-07-15 已更新为 -test 版 + setup 模板）**：`cls` 对象返回 4 个全局固定命令字段，详见上方 2026-07-15 记录。常量在 `controller/local_agent.go`，响应含全部 4 字段。
- **i18n 约束**：所有面向用户文案/错误用 i18n key，handler 禁硬编码字面量。
- **(2026-07-14 16:27 变更) config_type 非必传**：不传→返回全量配置（一期仅 cls）；传 cls→筛选 cls；传其他→400。handler 校验改 `configType != "" && configType != "cls"`，DB 查询固定 `Where("config_type = ?", "cls")`（空=全量时也返回 cls 块）。见 02-plan.md §七。
- **(2026-07-14 16:40 变更) 新增 admin 写接口 seed 凭据**：集成测试环境无人工写入通道，新增 `POST /admin/local-agent/cls-credential`（requireAdmin + WithOpenAPI 不审计 + 幂等 upsert），使 get-config IT 自洽跑通 200。见 02-plan.md §八、06-it.md §七。
- **(2026-07-14 17:20) cls-credential 补 DELETE**：与 feature-allowlist 对称，新增 `DELETE /admin/local-agent/cls-credential`（同守卫），单测 UT17/18 + IT helper + 文档。见 02-plan.md §十。
- **(2026-07-14 16:52 变更) 三点反馈**：①还原 .gitignore；②cls-credential 写接口加 `IsInternalAccount` 守卫（非内部账号→403，seam `IsInternalAccountFn`）；③新增 `POST/DELETE /admin/feature-allowlist` 写接口（显式配置跨租户白名单，不审计）。见 02-plan.md §九。
- **(2026-07-15 12:04 更新 / 12:14 修正) 4 cmd 常量定稿**：install/update 改为 `tencentcloud-cls-sdk-codebuddy-test` + 腾讯云 npm 镜像；run_cmd 改为 `cls-codebuddy setup` 模板串（含 ${endpoint}/${topic_id}/${secret_id}/${secret_key}/${local_agent_id}/${user_name}/${user_id} 占位符，**原样返回不渲染**，由本地 agent 自行替换）；uninstall 改为 `cls-codebuddy uninstall-all`。见 02-plan.md §十三 / 13.1。
- **(2026-07-14 20:47 初版，已被上文覆盖) 4 cmd 初版**：install/update=`npm install -g tencentcloud-cls-sdk-codebuddy`、run_cmd=`cls-codebuddy start`、uninstall=`ls-codebuddy uninstall`（已废弃）。

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

- **(2026-07-15 15:04 回退) topic_id 改回实时查询**：用户决定 topic_id 仍由 get-config 实时从 CLS OpenClawService 获取，**不落 db**。回退此前「topic_id 落表」方案（§十二），删除 model TopicID 字段 + init.sql/0708 的 topic_id 列 + 写接口 topic_id 参数。db 里不再有 topic_id。见 02-plan.md §十四。


- **(2026-07-15 16:37) 写接口鉴权收紧 + 覆盖率 + IT 修复**：写接口去 WithOpenAPI（仅 admin token）；补 UT19/20/21 覆盖率达 60%+；修复 helpers 导入错误。见 02-plan.md §十五。
