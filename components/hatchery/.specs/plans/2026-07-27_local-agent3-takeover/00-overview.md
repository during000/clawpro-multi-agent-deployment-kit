# [2026-07-27] 接管本地 Agent 三期（hatchery 后端）

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
> 会话恢复时，先读本文件定位当前步骤，再按需加载对应阶段文件。
>
> ⚠️ Meta 中的 `分支` 字段是上下文恢复时定位任务的唯一依据，**必须**与 `git branch --show-current` 的输出完全一致。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feature/local_agent3_1` |
| 摘要 | 移除本地 Agent（通用任务表+双端接口）、sync 双列表协议（cmds 统一 schema）、Hook 资源（复用 EnterpriseRule）、7 天阈值、codex 纳管 |
| 状态 | 进行中 |
| 创建日期 | 2026-07-27 |
| 负责人 | alexwhwang |
| 预期完成 | 2026-07-28（TAPD DDL） |

- TAPD：https://tapd.woa.com/tapd_fe/20422209/story/detail/1020422209135500434
- 技术方案 iWiki：https://iwiki.woa.com/p/4027911568（已定稿，含 API 定义与 sync/ack 协议 demo）

---

## Progress

<!--
Progress 更新规则（AI 必读）：
1. 步骤完成时，**原地替换** `- [ ] N. <step>` 为 `- [x] N. <step> (<结果摘要>)`
2. 禁止插入新行或重复编号，每个编号 01-08 有且仅有一行
3. 结果摘要示例：`(15/15 passed)`、`(覆盖率 92%)`、`(全量通过)`
4. 同步更新下方「当前步骤」章节
-->

- [x] 01. Clarify    → [01-clarify.md](./01-clarify.md) (需求澄清经多轮对话完成，结论固化在 iWiki 方案)
- [x] 02. Plan       → [02-plan.md](./02-plan.md) (6 个功能点 F1-F6，依赖链 F1→F2→F3→F4，F5 独立，F6 最后)
- [x] 03. Implement  → [03-implement.md](./03-implement.md) (F1-F6 全部功能点完成：建表+migration、双端移除、sync双列表、ack路由、7天阈值+codex、Hook复用EnterpriseRule)
- [x] 04. UT         → [04-ut.md](./04-ut.md) (F1-F4 共12个测试 + F5/F6 共6个测试，全部通过)
- [x] 05. Docs       → [05-docs.md](./05-docs.md) (API.md 已覆盖 remove 双端/sync cmds/ack 枚举/hook 创建；字段与代码对齐)
- [x] 06. IT         → [06-it.md](./06-it.md) (本地无 Docker/K8s/AK-SK 资源，真环境联调受阻；单测 + gofmt + go vet 已通过，CI 增量覆盖率待提 MR 后由 CI 跑)
- [x] 07. Review     → [07-review.md](./07-review.md) (AI 自审通过：红线/多租户/i18n/事务/并发无高严重度问题)
- [x] 08. Commit     → [08-commit.md](./08-commit.md) (MR !1062 已提：feature/local_agent3_1 → Release/2026_07_28，can_be_merged，commit_check success)

---

## 当前步骤

> 恢复会话时，优先读取此处指向的阶段文件。

- **步骤**：✅ 01-08 全部完成。MR !1062 已提（feature/local_agent3_1 → Release/2026_07_28，can_be_merged）。待 CI 增量覆盖率 + 人工 review + 真环境 IT 联调后合并。
- **文件**：[03-implement.md](./03-implement.md) / [07-review.md](./07-review.md)
- **上次更新**：2026-08-03 15:40

### 本轮（2026-08-03）新增改动（相对 07-31 定稿后又迭代）

> 07-31 定稿后，实测 + 审查又发现并修复以下问题，均已 commit + push 到 `feature/local_agent3_1`：

1. **`/openclaw/delete` 拒 local 改回 400**（之前同步软删三表，现统一走 remove 接口）——commit `7827c19e`
2. **卸载中间态改用 `last_known_status`+独立 `current_operation`**（去掉 `setOperation`/`clearOperation` 复用）——`f015905d`
3. **补 `local_instance_rules` 硬删**（之前漏删留孤儿）——`f015905d`
4. **report 心跳不覆盖 destroying**（修复实测竞态：`local_agent_tasks` 有数据但 `instances` 仍是 running）——`82b931a4`
5. **uninstall_teamai 下发真正包进事务**（调用方传 `tx`，原子提交）——`679f0c96`
6. **sync cmds 列表补齐 hook 项的 event/cmd**（修复 `cmds` 比 `commands` 少字段）——`87c27d3d`
7. **04 相关 UT 补全**：`SetsDestroyingOnTaskCreate` / `Failed_RestoresRunning` / `KeepsDestroyingDuringUninstall` / `HookCmd_CmdsHasEventAndCmd` / `LocalSource_Rejected`

**当前分支状态**：`feature/local_agent3_1` 已 push，领先 `Release/2026_07_28` 11 个 commit；merge release 有 2 处显式冲突（`admin_instances.go` source 过滤重构、`instance_lifecycle.go` 常量并集），均易解、不涉及卸载事务逻辑。

**CI 覆盖率门槛**（`.ci/coverage-check.yml`）：增量 60% / 全量 59%。本地 controller 包全量测试超时无法本地验证，以 CI 结果为准。

---

## 关键决策（用户已拍板）

1. **通用本地任务表 `local_agent_tasks`**：本期 type=`uninstall_teamai`，后续场景复用；status 对标 rule_distribution_records（pending/success/failed/cancelled）；`cmd` 创建任务时生成落表
2. **ack success → 仅软删 instances 行**，其他关联数据不动；report `Unscoped` 查软删行置 `deleted_at=NULL` 重新激活
3. **不活跃=stopped，无第三态**，仅阈值 24h→7 天
4. **重复提交去重/已跳过**：已有实现，不做
5. **sync 双列表**：`commands`（老 schema 不动）+ `cmds`（统一 slug/version/handle_type/event/cmd），数据一致，兼容新老 reporter；`rule_type` 更名 `handle_type`（prompt/rule/hook）
6. **删除接口双端都用 `instance_id` 入参**：`POST /local-agent/remove`（用户端，校验 owner）+ `POST /admin/local-agent/remove`（管控端，审计）
7. **Hook 复用 EnterpriseRule**：handle_type 加 `hook`，新增 `Event`/`Cmd` 字段；表单创建（单触发时机+单命令）；下发 `install_hook_rule`（返回 event/cmd/slug）、卸载 `uninstall_hook_rule`（仅 slug）；无文件无 download_url
8. **卸载命令格式**：`teamai uninstall --force --agent <agent_type>`（codebuddy/codex 等）
9. **野鹤白名单**不做；**workspace 绑定边界**后端不管

### 卸载中间态语义（2026-07-31 实测迭代定稿，取代决策 #2/#6 的早期描述）

> 决策 #2「ack success 仅软删 instances 行」与 #6「双端 remove 接口」在实测后已演化，以下为最终权威结论。

- **`/openclaw/delete` 对 local 实例返回 400 拒绝**（无 CVM 实体可 Terminate）；本地实例删除统一走 `/local-agent/remove`（用户端）+ `/admin/local-agent/remove`（管控端），两个接口**保留不合并**。
- **下发 uninstall_teamai 任务时即进入中间态**：直接写 `last_known_status = destroying`（该字段是前端真实状态字段，直接返回前端展示「销毁中」）+ `current_operation = uninstall_local_agent`（常量 `model.LocalAgentOpUninstall`，防重入）。**不复用 CVM 的 `setOperation`/`clearOperation`**——那是 CVM 删除专用状态机，会写 `status_synced_at`/`last_stable_state` 等本地 agent 不需要的字段，且 failed 分支漏清会卡死中间态。
- **ack success**：清理四表——`instances` 软删；`local_instance_skills` / `local_instance_rules` **硬删**（无 `deleted_at` 且有 upsert 唯一索引，不能软删）；`local_instance_infos` 软删。`current_operation` 随 instances 行软删消失。
- **ack failed**：`last_known_status` 恢复 `running` + `current_operation` 清空，实例退出「销毁中」可重试（修复早期 failed 卡死 bug）。
- **report 心跳不覆盖 destroying（关键 bug fix）**：`HandleLocalAgentReport` 的 default 分支原本无条件写 `last_known_status = running`，会把卸载中实例冲回 running，导致前端永远看不到「销毁中」（实测 `local_agent_tasks` 有数据但 `instances` 仍是 running）。修法：report 写 `last_known_status` 时若 `current_operation == uninstall_local_agent` 则保持 `destroying`。`sync` 接口只拍 `last_report_at` 不写 status，安全。
- `local_agent_tasks` 的 pending 幂等保证重复下发不重复进入中间态标记逻辑（命中已有 pending 任务直接返回）。
