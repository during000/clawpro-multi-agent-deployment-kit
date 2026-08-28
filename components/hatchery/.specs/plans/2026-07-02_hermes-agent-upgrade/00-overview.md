# 2026-07-02 Hermes Agent 一键升级支持

> **本文件是本任务的单一真相源（Single Source of Truth）**：任务元信息、进度、当前步骤、关键决策全部在这里。
>
> ⚠️ 本任务目录为**事后反向补建**（分支开发时未走 SOP，2026-07-20 由 AI 根据分支实际改动、提交历史、代码现状反推整理），非实时同步产物。目的是把既成事实固化为可追溯文档，为后续同类改动建立参照。

---

## Meta

| 项 | 值 |
|----|----|
| 分支 | `feature/hermes_agent_upgrade` |
| 摘要 | 把"一键升级"（备份→SMH上传→重装→恢复）能力从仅支持 OpenClaw 扩展到 Hermes 类型实例 |
| 状态 | 已完成（代码已实现，本文档为事后补建） |
| 创建日期 | 2026-06-23（首个提交） |
| 负责人 | （原开发者，AI 反向整理） |
| 预期完成 | 2026-07-17（末次提交） |

---

## Progress

- [x] 01. Clarify    → [01-clarify.md](./01-clarify.md) (反向补建)
- [x] 02. Plan       → [02-plan.md](./02-plan.md) (反向补建)
- [x] 03. Implement  → [03-implement.md](./03-implement.md) (反向补建，代码已存在)
- [x] 04. UT         → [04-ut.md](./04-ut.md) (全量通过，反向补建)
- [x] 05. Docs       → [05-docs.md](./05-docs.md) (docs/API.md 已同步，反向补建)
- [x] 06. IT         → [06-it.md](./06-it.md) (Hermes 专属 IT 缺口已补齐)
- [x] 07. Review     → [07-review.md](./07-review.md) (AI 事后审查，2 项已修复)
- [x] 08. Commit     → [08-commit.md](./08-commit.md) (已按 MCP 提交)

---

## 当前步骤

- **步骤**：✅ 已完成（08. Commit 已提交远端）
- **文件**：[08-commit.md](./08-commit.md)
- **上次更新**：2026-07-20（全部阶段补齐并提交）

---

## 时间记录

> 反向补建，"开始/结束时间"取自 git 提交时间戳，非实时记录。

| 步骤 | 开始时间 | 结束时间 | 耗时 | 备注 |
|------|---------|---------|------|------|
| 01. Clarify | 2026-06-23 17:55 | 2026-06-23 17:55 | 0 | 反向补建，无原始记录 |
| 02. Plan | 2026-06-23 17:55 | 2026-06-23 17:55 | 0 | 反向补建，无原始记录 |
| 03. Implement | 2026-06-25 15:10 | 2026-07-17 08:25 | ~24 天（含多次迭代） | 38 个提交，含 3 次 revert |
| 04. UT | 2026-07-17 07:26 | 2026-07-17 08:25 | 分散 | controller/openclaw_upgrade_hermes_test.go 443 行 |
| 05. Docs | 2026-07-15 16:47 | 2026-07-15 16:47 | 0 | docs/API.md 随 0446f100 同步更新 |
| 06. IT | 2026-07-20 | 2026-07-20 | 本次补建 | 新增 test_instance_upgrade_hermes.py |
| 07. Review | 2026-07-20 | 2026-07-20 | 本次补建 | 详见 07-review.md |
| 08. Commit | 2026-07-20 | 2026-07-20 | 本次补建 | 通过 gongfeng MCP 提交至远端 |

---

## 关键决策备忘

1. **能力位化而非硬编码判断**：`AgentType.SupportsUpgrade` / `NeedsRuntimeUserCorrection` 取代原来的 `runtimeType == AgentTypeOpenClaw` 硬编码判断，OpenClaw/Hermes 走同一套 Go 升级骨架，通过 `ResolveScript` 表驱动分派备份/恢复脚本。
2. **RuntimeUser 重探机制**：重装本身是镜像变更事件，镜像侧可能改变默认运行用户（Hermes v0.12.0 起从 `agentuser` 切到 `ubuntu`）。新增 `redetectAndPersistRuntimeUser`，重装后以 root 身份重新探测并回写 DB，取代"假设旧值仍成立"的错误假设。
3. **恢复脚本覆盖式设计**：`restore_post_reinstall_hermes.sh` 采用"mv 新镜像 `.hermes` → 解压覆盖 → cp -an 回补新镜像独有文件"策略，兼顾速度（mv 同分区瞬间完成）与数据完整性（失败自动回滚）。
4. **升级后置 hook 表驱动分派**：`upgradePostHookTable[runtimeType]` 替代原来 if-else 判断，OpenClaw 走 5 项补丁（sync_gateway_port/fixPluginNodeModules/runCompatScripts/cleanupUpgradeTemp/approveDeviceAfterUpgrade），Hermes 走精简的二次 ready 探测 + 通道兼容 + 清理。
5. **`OperationTimeouts[OpUpgrade]` 未真正生效**：`isOperationTimedOut` 的唯一调用点（`instance_state.go` 的 `handleStatusSideEffects`）显式排除了 upgrade/migrate 操作，生命周期由异步 goroutine 自行管理，进程重启由 `recoverInterruptedUpgradeAndMigrate` 兜底清理。2700s 常量目前只是文档性描述，AI Review 阶段已修正其注释使其准确反映实际生效范围（详见 07-review.md）。
6. **`config/agent_plugin_preserve_paths.json` 与脚本内 `PRESERVE_PATHS` 双处同步**：Go 侧未消费该 JSON，纯声明性配置，脚本硬编码副本才是真正生效的列表。AI Review 阶段补充了单测守卫防止未来漂移（详见 07-review.md）。

---

## 风险速览

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | `config/agent_plugin_preserve_paths.json` 与脚本内数组双处同步，容易漂移 | 中 | 已补充 Go 单测交叉校验（见 07-review.md） |
| 2 | Hermes 升级链路缺专属集成测试 | 中 | 已补充 `test_instance_upgrade_hermes.py` 契约级用例 |
| 3 | 38 个提交历史含 3 次 revert（误改后撤回），说明中途有越界改动 | 低（已收敛） | 无需处理，仅记录为经验教训 |
| 4 | Commit message 不完全符合 Conventional Commits | 低 | 历史已定型，不做 rebase/rewrite（避免破坏协作分支） |

---

## 文件索引

| 文件 | 产物 |
|------|------|
| [00-overview.md](./00-overview.md) | 任务总览（本文件） |
| [01-clarify.md](./01-clarify.md) | 需求澄清（反向推导） |
| [02-plan.md](./02-plan.md) | 方案设计（反向推导） |
| [03-implement.md](./03-implement.md) | 实现细节（对照现有代码整理） |
| [04-ut.md](./04-ut.md) | 单元测试结果 |
| [05-docs.md](./05-docs.md) | 文档更新清单 |
| [06-it.md](./06-it.md) | 集成测试补齐记录 |
| [07-review.md](./07-review.md) | Code Review：问题与修复 |
| [08-commit.md](./08-commit.md) | Commit message 与提交前检查 |
