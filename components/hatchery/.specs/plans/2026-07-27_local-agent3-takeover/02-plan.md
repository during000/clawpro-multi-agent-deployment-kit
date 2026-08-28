# 02. Plan — 实施计划

> 方案定稿见 iWiki：https://iwiki.woa.com/p/4027911568
> 按以下 6 个功能点顺序实现，每个功能点走「功能代码 → go build → UT → 全绿」小循环。

---

## 功能点拆分

### F1. local_agent_tasks 表 + model + migration

- `model/local_agent_task.go`：LocalAgentTask struct（见 iWiki §二），TableName `local_agent_tasks`
- 常量：`LocalAgentTaskTypeUninstallTeamai = "uninstall_teamai"`；status 复用 `RuleRecordStatusPending/Success/Failed/Cancelled` 语义（新建 `LocalAgentTaskStatusXxx` 别名或直接引用，实现时定）
- `sql/init.sql` 建表 + 增量 migration `sql/0727-local-agent-phase3.sql` + `MigrateFromSQLite` 注册
- 索引：identifier、instance_id、deleted_at

### F2. 移除本地 Agent 接口 ×2

- `POST /local-agent/remove`（用户端）：requireLogin + ensureLocalAgentAllowed；入参 `{"instance_id": uint}`；校验实例存在 + source=local + owner=当前用户
- `POST /admin/local-agent/remove`（管控端）：requireAdmin；入参同；校验 source=local
- 共用内部函数 `createUninstallTeamaiTask(ctx, inst, operatorID)`：
  - 幂等：已有 pending 的 uninstall_teamai 任务 → 直接返回已有 task
  - cmd 拼装落表：`fmt.Sprintf("teamai uninstall --force --agent %s", inst.AgentType)`
- 路由注册 main.go + auditRules（admin 接口）+ API 响应 `{ok, task_id, status}`
- i18n：新增错误文案 key（实例不存在/非本地实例）复用现有 `MsgInstanceNotFoundOrNoPerm` 优先

### F3. sync 双列表（cmds）+ 合并 local_agent_tasks

- 新增统一结构 `syncCmdItem`：id/type/slug/version/handle_type/event/cmd/download_url/scope/workspace_path/project_id（全部按 iWiki §7.1 omitempty 规则）
- 组装顺序：skill records → rule records → local_agent_tasks pending
- 单一数据源：先组装 `[]syncCmdItem`（cmds），`commands` 由 cmds 映射回老字段名（skill_slug/rule_slug/rule_type），保证两列表数据一致
- uninstall_teamai item：仅 id/type/cmd
- 老 reporter 兼容：commands 结构与现有完全一致（含 rule_type 字段名不变）

### F4. ack 路由 uninstall_teamai + report 激活

- ack type 枚举加 `uninstall_teamai`：路由到 local_agent_tasks
  - success → 事务内：任务置 success + 软删 instances 行（tx.Delete）
  - failed → 任务置 failed + 记 error，保留可重试
  - 幂等：pending→success/failed、failed→success；终态不覆盖
- report 激活：`HandleLocalAgentReport` 的 instance 查询改 `Unscoped()`，命中 `deleted_at IS NOT NULL` 行时 `Update("deleted_at", nil)` 恢复
- 注意：软删实例的 sync/ack 请求处理（软删后 reporter 可能还有一次 ack 到达——ack 查 instance 需 Unscoped 容忍）

### F5. 7 天阈值 + codex 纳管 + 列表筛选

- `LocalInstanceOfflineThreshold`：24h → 7*24h（找到常量定义处直接改）
- codex：`model/agent_type.go` 枚举/映射加 codex；AgentTypeSupportsSkill/Rule 兼容表纳入
- Agent 列表接口：加 `source` 筛选参数（cloud/local），透传到 query Where

### F6. Hook 资源（复用 EnterpriseRule）

- model：`EnterpriseRuleTypeHook = "hook"`；EnterpriseRule 加 `Event`（varchar(32)）/`Cmd`（text）字段；init.sql + 增量 migration + MigrateFromSQLite
- event 枚举常量：SessionStart/UserPromptSubmit/PreToolUse/PostToolUse/Stop + 校验函数
- 创建：`HandleCreateRule` 加 hook 分支——type=hook 时要求 event+cmd 非空、跳过文件上传（无 COSKey）；slug 系统生成
- 下发/卸载：复用 distribute/uninstall handler；下发对象校验加 hook 兼容性（本地实例 + agent_type 支持）
- sync：rule commands 组装时 hook 类型输出 `install_hook_rule`/`uninstall_hook_rule` + event/cmd 字段、无 download_url
- ack：type 枚举加 `install_hook_rule`/`uninstall_hook_rule`，走现有 rule ack 管道（handleLocalAgentRuleAck 的 type 映射扩展）
- 版本记录/资产绑定管道自动复用（handle_type=hook 的 rule 走同一 slug 体系）

---

## 依赖关系

```
F1 → F2 → F3 → F4（严格顺序，同一条链路）
F5 独立，可穿插
F6 依赖 F3 的 cmds 结构，排最后
```

## UT 策略（04 阶段汇总跑）

- F1/F2：内存 sqlite + httptest；幂等重复调用断言
- F3：sync 响应双列表一致性断言（cmds vs commands 字段映射）；uninstall_teamai item 字段裁剪断言
- F4：ack 成功软删断言（Unscoped 可查）+ report 激活断言（deleted_at 置空）+ 幂等
- F6：hook 创建校验（event 枚举/cmd 非空/无文件）、sync 输出 install_hook_rule 字段断言、ack 路由

## 风险

- report 激活改 Unscoped：注意不能影响正常未删实例的路径（先 Unscoped 查，未软删走原逻辑）
- commands 老结构回归：现有 reporter UT 必须全绿（不能动老字段名/语义）
- EnterpriseRule 加字段：确认 `MigrateFromSQLite` 与单测 AutoMigrate 均覆盖
