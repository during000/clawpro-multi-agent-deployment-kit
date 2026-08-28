---
task_id: hermes-agent-upgrade
stage: 02-plan
author: AI（反向推导）
date: 2026-07-20
run_mode: manual
---

# Plan（反向补建）

## 改动文件清单

| 文件 | 改动类型 | 说明 |
|------|---------|------|
| `model/agent_type.go` | 修改 | `AgentType` 新增 `SupportsUpgrade` / `NeedsRuntimeUserCorrection` 能力位 |
| `controller/agent_type_guard.go` | 修改 | 升级前置校验改为读能力位而非硬编码类型判断 |
| `controller/openclaw_upgrade.go` | 修改（核心） | 升级主流程：新增 `redetectAndPersistRuntimeUser`、`runHermesUpgradePostHooks`、`upgradePostHookTable` 表驱动分派 |
| `controller/upgrade_prereq.go` | 修改 | 升级前置检查支持 Hermes 分支 |
| `controller/script_registry.go` | 修改 | `ResolveScript` 按 `runtimeType + scriptKind` 表驱动解析脚本路径 |
| `scripts/backup_pre_reinstall_hermes.sh` | 新增 | Hermes 专属备份脚本 |
| `scripts/restore_post_reinstall_hermes.sh` | 新增 | Hermes 专属恢复脚本（mv+解压覆盖+cp -an 补新镜像独有文件） |
| `config/agent_plugin_preserve_paths.json` | 新增 | 声明重装后需保留的路径（**发现未被 Go 消费，纯文档性**） |
| `i18n/keys.go` / `i18n/en.go` | 修改 | 新增升级相关错误提示的中英文案 |
| `docs/API.md` | 修改 | 同步升级接口对 Hermes 的类型支持说明 |
| `controller/openclaw_upgrade_hermes_test.go` | 新增 | Hermes 升级单测（12 个用例） |

## 调用链

```
POST /openclaw/upgrade
  → controller.HandleUpgrade
    → checkInstanceSupportsUpgrade(instance)      # 读 AgentType.SupportsUpgrade
    → upgrade_prereq.go: 前置校验（余额/状态/并发锁）
    → go performUpgrade(ctx, instance)             # 异步 goroutine，detach context
        → ResolveScript(runtimeType, "backup_pre_reinstall")
        → RunScript(backup)                        # 备份并上传 SMH
        → 触发 CVM 重装
        → waitForInstanceRunning()                 # 等待 CVM RUNNING
        → if instance.AgentType.NeedsRuntimeUserCorrection:
              redetectAndPersistRuntimeUser(ctx, instance)   # 重新探测运行用户并回写 DB
        → waitForOpenclawReady()                   # 等待 Agent 就绪
        → ResolveScript(runtimeType, "restore_post_reinstall")
        → RunScript(restore)                       # 恢复备份数据
        → upgradePostHookTable[runtimeType](ctx, instance)   # 表驱动后置动作
        → 标记 OpStateSuccess / OpStateFailed
```

## 测试用例设计（自然语言描述）

### 单元测试
1. Hermes 类型实例调用升级接口 → 应通过 `SupportsUpgrade` 校验，进入正常升级流程。
2. 不支持升级的 AgentType（如未设置 `SupportsUpgrade`）→ 应返回 `MsgAgentTypeDoNotSupportUpgradeWithDetail` 错误。
3. `NeedsRuntimeUserCorrection=true` 的类型（Hermes）→ 重装完成后必须调用 `redetectAndPersistRuntimeUser`，且新用户写回 DB。
4. `NeedsRuntimeUserCorrection=false` 的类型（OpenClaw）→ 不触发重探测，沿用原逻辑。
5. `ResolveScript` 按 `runtimeType="hermes", kind="backup_pre_reinstall"` → 应解析到 `backup_pre_reinstall_hermes.sh`。
6. `upgradePostHookTable["hermes"]` → 应执行 Hermes 专属后置动作，不执行 OpenClaw 的 5 项 post-hook。

### 集成测试（原缺失，本次补齐，见 06-it.md）
7. Hermes 实例 + 未认证/越权用户调用升级接口 → 401/403。
8. Hermes 实例升级接口契约校验（400 参数缺失、404 实例不存在）。
9. 不支持升级的 AgentType 尝试升级 Hermes 实例路径 → 应返回业务错误码而非 500。

## 风险评估

| 风险 | 说明 | 结论 |
|------|------|------|
| 升级操作超时预算 | 最初怀疑 `OperationTimeouts[OpUpgrade]=2700s` 与内部步骤耗时（备份/等待/恢复）总和不匹配 | **排查后确认非真实风险**：`isOperationTimedOut` 显式排除 upgrade/migrate，生命周期由异步 goroutine 自行管理；只是代码注释描述不够精确，已在 Review 阶段修正注释 |
| preserve_paths 双写漂移 | JSON 声明与脚本硬编码数组分别维护，无自动校验 | 已在 Review 阶段补充 Go 单测交叉校验兜底 |
| Hermes IT 覆盖缺口 | 无专属集成测试文件 | 已在 06-it.md 补齐 |
