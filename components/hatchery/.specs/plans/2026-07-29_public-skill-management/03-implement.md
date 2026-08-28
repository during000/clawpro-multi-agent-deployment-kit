# 03. Implement — 实现记录

---

## 关键实现细节

- `model.ListDistributedSkillStates` 按实例、slug 折叠成功的 distribute/uninstall 事件，跨 Public/Enterprise 还原最后一次物理状态；pending/failed 不改变当前版本。
- `GET /openclaw/skills` 保留 Agent 实际列表，所有条目返回稳定 `slug` 和 `can_uninstall`；仅命中有效 Admin 下发记录的技能补充 `version`、`latest_version`、`update_available`。Public 最新版本使用 5 分钟同 slug 合并缓存；列表刷新失败可回退过期缓存，无缓存可回退的失败按请求聚合为一条 Warn 日志。
- OpenClaw、Hermes、LightClaw ACE 列表脚本从安装目录名取得 slug。OpenClaw/ACE 再用 `SKILL.md` 的展示名关联 CLI 项；命中用户技能目录时 `can_uninstall=true`，仅 CLI 可见或 `source=builtin` 的内建技能为 false，Hermes 直接目录扫描结果恒为 true。
- 把 Admin 下发/卸载执行器拆为共享执行配置和同步/异步入口。Admin 保持原异步行为；用户接口同步复用制品准备、SMH 下载、Agent 脚本、task/record 终态及技能锁。
- 执行脚本、下载 URL、技能准备、分布式锁和 Public 版本缓存均通过局部依赖显式传入；`NewUserSkillHandlers` 返回三个包级 Handler 函数并在闭包内共享线程安全缓存，测试只修改各用例自己的依赖值，不再覆盖包级全局变量。
- 新增 `POST /openclaw/update-skill`、`POST /openclaw/uninstall-skill`：校验登录、实例归属、CVM、Agent 技能能力、running 状态与安全 slug；更新仅支持 Admin 下发技能；Admin 下发技能卸载维护 task/record，其他技能获取 runtime+slug 锁后直接执行一次物理 executor。
- 两个写接口均注册 `WithAudit`，错误响应复用现有 i18n Key；没有新增用户可见硬编码文案。

## 与 Plan 差异

- Plan 将 OpenClaw/ACE 的 slug 描述为脚本内部字段；实现进一步明确以安装目录名为唯一稳定来源，CLI `name` 仅用于展示。
- 未修改数据库 schema、`sql/init.sql` 或 migration；状态由现有任务和记录表推导。
- 三个运行时列表脚本的真实输出契约由集成测试覆盖；Go 单元测试只验证后端对 `slug`、`can_uninstall` 和 Admin 下发版本字段的合并。
- Commit 后按用户确认扩展其他运行时技能卸载：没有 Admin 下发状态时按 runtime+slug 互斥，不补造版本或分发记录；性能收敛后删除卸载前后的三次远程列表查询，脚本成功统一返回 true。
- Commit 后按用户确认增加卸载能力字段：列表脚本负责识别用户目录与内建技能，Go 响应结构只透传布尔值，前端据此阻止内建技能卸载。

## 烟雾验证

- `bash -n scripts/list_skills.sh scripts/list_skills_hermes.sh scripts/list_skills_ace.sh`：通过。
- 三运行时实际 CVM IT 验证列表脚本均返回稳定 slug 和正确 `can_uninstall`，且不提供技能版本字段。
- `go test ./controller -run 'TestHandleSkillsList|TestListInstanceSkills|TestReadSkillsViaChunks'`：通过。
- `go test ./controller -run 'Test(EnrichDistributedSkillVersions|HandleUserUninstallSkill_DirectRuntimeSkill)' -count=1 -race`：通过。
- `go test ./controller -run 'Test(HandleUserUninstallSkill|SkillTaskExecution_RoutesScriptsByAgentType)' -count=1 -race`：直接卸载收敛后通过。
- `go test ./... -run '^$'`：9 个 package 通过，1 个 package 无测试。
- `go vet ./...`：通过。

## 检查项

- [x] `gofmt` 格式化通过
- [x] 受影响 package 的 `go vet` 无错误
- [x] 写接口已添加审计日志
- [x] 无数据库变更，无需更新 `sql/init.sql` 或 migration
- [x] 数据访问使用请求/执行上下文中的 `model.DB(ctx)`
- [x] 无硬编码密钥/配置
- [x] 用户可见文案复用 i18n Key，无新增 Key
