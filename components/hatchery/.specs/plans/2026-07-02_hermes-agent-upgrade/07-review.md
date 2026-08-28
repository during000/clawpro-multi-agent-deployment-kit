---
task_id: hermes-agent-upgrade
stage: 07-review
author: AI（Code Review）
date: 2026-07-20
run_mode: manual
---

# Review

## 审查范围

对 `feature/hermes_agent_upgrade` 分支相对 `master` 的 38 个提交、全部改动文件做事后审查，覆盖：规范符合性、安全基线、风险点、测试覆盖。

## 结论摘要

| 类别 | 结果 |
|------|------|
| 编译 | 通过（`go build ./...`） |
| 单元测试 | 通过（`go test ./controller/... -run 'Hermes\|Upgrade\|Restore\|Backup'`） |
| 裸 SQL | 未发现，全部走 `model.DB(ctx)` |
| 审计规则 | `/openclaw/upgrade`、`/openclaw/upgrade/retry` 已注册 `WithAudit` |
| 异步 goroutine context | 全部使用 `hcommon.DetachContext(ctx)`，未直接传 `r.Context()` |
| i18n | 新增 key 均已在 `en.go` 补英文翻译 |
| GORM model 变更 | 无（`AgentType` 为编译期硬编码 map），红线 #3 不适用 |

## 已发现并修复的问题

### 问题 1：`config/agent_plugin_preserve_paths.json` 与脚本硬编码数组的配置漂移风险

**现象**：该 JSON 文件声明"重装后必须保留的插件路径"，但排查发现 Go 代码从未读取此文件（不同于 `plugin_version_map.json`/`clawpro_required_sg_rules.json`/`default_roles.json` 三者都有 `PluginVersionMapJSON` 等变量通过 `go:embed`/`os.ReadFile` 注入并被消费）。真正生效的是 `scripts/backup_pre_reinstall_hermes.sh` 内硬编码的 `PRESERVE_PATHS` 数组，脚本注释里也明确写"此列表为 JSON 的硬编码副本，脚本不动态读取 JSON，新增/修改需同时改两处"。

**风险**：两处手动同步，未来新增插件保留路径时容易漏改一处，导致：
- 只改了 JSON、没改脚本 → 备份时漏掉该路径，数据丢失；
- 只改了脚本、没改 JSON → 文档与实现脱节，后续维护者被误导。

**修复**：新增 `controller/agent_plugin_preserve_paths_consistency_test.go`，解析 JSON 与脚本内 `PRESERVE_PATHS` 数组做双向集合比较，任一方向缺失都会 fail，并给出明确的修复提示（缺声明 or 缺脚本同步）。已验证：
- 正常情况下测试通过；
- 人为引入漂移（临时改脚本值）后测试正确 fail，随后已完整还原文件（`git diff` 确认无残留改动）。

**未采用的替代方案**：改为 Go 侧动态读取 JSON 并作为脚本参数传入，从根本消除双写。原因：改动面更大（需要改 `RunAgentScript` 传参机制、Shell 脚本解析逻辑），超出本次"查漏补缺"的范畴，留作后续技术债务，已在 `00-overview.md` 风险速览中记录。

### 问题 2：Hermes 升级链路缺专属集成测试

**现象**：`test/scripts/openclaw_instance/test_instance_upgrade.py` 只做与 `agent_type` 无关的通用契约测试，未覆盖 Hermes 专属的能力位（`SupportsUpgrade`）契约。对比同项目 channel/model 等模块均有 `*_hermes.py` 专属测试文件，升级链路此前是空白。

**修复**：新增 `test/scripts/openclaw_instance/test_instance_upgrade_hermes.py`，详见 `06-it.md`。

## 排查后确认为非真实风险的点（已修正认知）

### 最初怀疑的"操作超时预算不足"

**最初判断**：`model.OperationTimeouts[OpUpgrade]=2700s`（45min）与内部步骤（`waitForInstanceRunning` 15min + `redetectAndPersistRuntimeUser` 5min + `waitForOpenclawReady` 20min + `restore` 单次 40min）总和相比明显不足，怀疑存在"锁超时先到期、恢复脚本仍在跑"的竞态。

**排查结论**：`isOperationTimedOut` 的唯一调用点（`controller/instance_state.go`）在判断逻辑中**显式排除了 `OpUpgrade`/`OpMigrate`** 两种操作类型 —— 这两类操作的生命周期完全由各自的异步 goroutine 自行管理（成功/失败/超时都由 goroutine 内部逻辑决定何时更新 `current_operation` 状态），进程重启导致 goroutine 丢失的情况由 `task.recoverInterruptedUpgradeAndMigrate` 在启动时兜底清理为失败态。`OperationTimeouts[OpUpgrade]` 常量目前只是一个"文档性/预留"的数字，并未被消费于任何实际超时判断路径。

**结论**：不存在真实的竞态风险，本次不做数值调整，仅在 `00-overview.md` 关键决策备忘中记录该事实，避免后续开发者被这个常量误导以为它在生效。

## 遗留建议（不在本次修复范围，记录供后续参考）

1. `config/agent_plugin_preserve_paths.json` 与脚本的双写问题，长期看应统一为 Go 侧动态读取并参数化传入脚本；当前已用单测兜底，短期风险可控。
2. Commit message 历史不完全符合 Conventional Commits 规范，历史已定型，不建议 rebase/rewrite（避免破坏协作分支历史）。
3. 建议在提 MR 前，在具备 Hermes 镜像 + CVM 环境的测试环境中实跑一次 `test_instance_upgrade_hermes.py`，并补跑一次 `BASE_BRANCH=origin/master bash .ci/ci-check-coverage.sh` 增量覆盖率检查。
