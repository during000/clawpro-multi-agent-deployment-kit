---
task_id: hermes-agent-upgrade
stage: 03-implement
author: AI（反向推导）
date: 2026-07-20
run_mode: manual
---

# Implement（反向补建，对照现有代码整理）

代码已在分支中实现完成，本文档仅做关键实现点的事后说明，不重复贴代码全文。

## 1. 能力位化（`model/agent_type.go`）

`AgentType` struct 新增两个 bool 字段：
- `SupportsUpgrade`：该类型是否允许走一键升级流程
- `NeedsRuntimeUserCorrection`：重装后是否需要重新探测运行用户（Hermes=true，OpenClaw=false）

取代此前 `runtimeType == "openclaw"` 的硬编码判断，为后续新增 agent 类型留出扩展点。

## 2. RuntimeUser 重探测（`controller/openclaw_upgrade.go::redetectAndPersistRuntimeUser`）

重装完成、CVM RUNNING 后，以 root 身份通过 TAT 探测脚本执行用户识别命令，取得实际运行用户后写回 `instances.runtime_user` 字段（`model.DB(ctx)`，符合多租户隔离规范）。仅当 `AgentType.NeedsRuntimeUserCorrection=true` 时触发，OpenClaw 分支保持原有行为不变。

## 3. 脚本表驱动分派（`controller/script_registry.go::ResolveScript`）

以 `(runtimeType, scriptKind)` 为 key 查表返回脚本相对路径，替代原来 `if runtimeType == "openclaw" { ... } else if ... `的链式判断。新增两个 Hermes 专属脚本：
- `scripts/backup_pre_reinstall_hermes.sh`
- `scripts/restore_post_reinstall_hermes.sh`

## 4. Hermes 恢复脚本策略（`scripts/restore_post_reinstall_hermes.sh`）

采用"mv 新镜像目录为 `.hermes.bak` → 解压备份覆盖 → `cp -an` 补齐新镜像独有的新文件"策略：
- `mv` 同分区操作，耗时接近 O(1)，不占用磁盘 IO 峰值；
- 解压覆盖保证用户历史数据优先；
- `cp -an`（`-n` = no-clobber）确保新镜像引入的新文件不会被旧备份覆盖丢失；
- 任一步骤失败自动从 `.hermes.bak` 回滚，保证不产生"半升级"状态。

## 5. 升级后置动作表驱动（`upgradePostHookTable`）

```go
var upgradePostHookTable = map[string]func(ctx context.Context, instance *model.Instance) error{
    "openclaw": runOpenClawUpgradePostHooks, // sync_gateway_port / fixPluginNodeModules / runCompatScripts / cleanupUpgradeTemp / approveDeviceAfterUpgrade
    "hermes":   runHermesUpgradePostHooks,   // 二次 ready 探测 + 通道兼容 + 清理临时文件
}
```

## 6. preserve_paths 配置（`config/agent_plugin_preserve_paths.json`）

原意图是作为"重装后必须保留路径"的单一声明源，但排查发现 **Go 代码未读取该 JSON**，脚本内 `PRESERVE_PATHS` 数组是硬编码副本，两者靠人工保持一致。属于本次 Review 发现并修复的问题（见 07-review.md）。

## 7. i18n

新增 `MsgAgentTypeDoNotSupportUpgradeWithDetail` 等 key，均已在 `i18n/en.go` 补充英文翻译，符合红线 #12 要求。

## 8. 审计与安全基线核查结果

- 审计规则：`/openclaw/upgrade`、`/openclaw/upgrade/retry` 已在 `controller/audit.go::auditRules` 注册并用 `WithAudit()` 包装。
- 无裸 SQL：`openclaw_upgrade.go` 全部通过 `model.DB(ctx)` 访问数据库。
- 异步 goroutine 全部使用 `hcommon.DetachContext(ctx)`，未直接传递 `r.Context()`，符合红线 #8。
- 无 GORM model 结构变更（`AgentType` 为编译期硬编码 map，非持久化表），不涉及 `sql/init.sql` 或迁移脚本，红线 #3 不适用。
