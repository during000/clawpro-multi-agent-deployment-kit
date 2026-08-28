# 07. Review — 代码审查

---

## 审查方式

- [x] 重新并行执行 Standards / Spec 双轴 Review
- [x] 逐项对照最近生产代码与仓库主流约定复核 reviewer 结论
- [x] Go 专项复核：并发、Context、错误终态、GORM、安全、性能与测试生命周期
- [ ] 人工 Review

审查基线：

- 固定点：`origin/master`
- 固定点提交：`9f2b2120c713301738866398e17b2e0b3562c684`
- 审查对象：`git diff origin/master...HEAD`
- 分支在 master 之后保持单一提交

## 重新审查结论

### 已确认并修复

| # | 文件 | 问题 | 严重度 | 修复与回归保护 |
|---|------|------|-------|---------------|
| R1 | `controller/openclaw_skill_operations.go` | Enterprise 用户操作锁已安装版本 ID，无法与 Admin 对目标/最新版本 ID 的锁互斥 | 高 | 更新直接锁目标 SkillID；卸载按 Admin 规则解析最新 SkillID 后加锁；锁内同时复核最初安装来源与 SkillID；更新、卸载各有锁键测试 |
| R2 | `controller/skill_task_executor.go` | record 成功终态写入失败后仍累计 success 并执行 `OnSuccess`，造成 task/record 与副作用矛盾 | 高 | 改计 failed、禁止 `OnSuccess`，并再次写入失败终态；注入 GORM 更新失败验证 task/record/回调 |
| R3 | `model/skill.go`、`controller/public_skill_version_test.go` | `NormalizeSkillVersion` 接受 `-1.0.0` / `1.+2.3`，非法 Public 版本会进入五分钟缓存 | 中 | 规范化前拒绝符号组件；模型与缓存测试覆盖 signed version 不缓存 |
| R4 | `scripts/list_skills.sh` | 为假设中的重复 display name 构建多值映射和保序去重，复杂度无实际收益 | 低 | OpenClaw 源码确认本地技能按 name 合并、远程重名会改名，CLI 输出 name 唯一；改为单值 name→slug 映射 |
| R5 | `controller/openclaw_skill_operations.go` | Public 冷缓存逐项串行访问仓库，故障延迟可放大为约 `30s × N` | 中 | 预收集 Public slug，使用最多 8 个 worker 并发查询；测试同时证明并行与并发上限 |
| R6 | `test/scripts/skill/test_user_skill_operations.py` | 审计断言可命中旧测试残留记录 | 中 | 查询并断言本次实例唯一 `resource_id` |
| R7 | `controller/skill_task_executor_test.go` | 异步测试在释放 channel 前 `t.Fatal` 会令清理永久等待 | 中 | 三处阻塞测试均注册 `sync.Once` 保护的 cleanup 释放与 WaitGroup 等待 |
| R8 | `test/scripts/skill/test_user_skill_operations.py` | 卸载后 DB 会移除响应 slug，原 absent 检查可能在物理目录仍存在时误通过 | 中 | 卸载前保存 display name，absent 同时按原 slug/name 检查实时列表 |
| R9 | `test/scripts/skill/test_user_skill_operations.py` | Hermes 实例创建重复实现既有 helper，遗漏并发配置覆盖重试 | 中 | 复用 `setup_hermes_instance`，保留既有镜像、Agent type、重试与 RuntimeUser 稳定等待 |
| R10 | `controller/openclaw_skill_operations.go` | 用户 slug 在脚本模板赋值中可能先触发 shell 展开再进入脚本校验 | 高 | Handler 在任何 TAT 调用前限制为安全的 ASCII 目录 slug；非法输入返回 400，测试覆盖 `$(id)` |
| R11 | `controller/openclaw_skill_operations.go` | 已是最新版时在获取技能锁前返回，无法与并发卸载互斥 | 中 | 所有更新路径先获取 Admin 技能锁再做锁内状态与幂等复查；最新版锁冲突测试返回 409 |
| R12 | `controller/openclaw_skill_operations.go` | 锁内发现并发卸载已完成时返回 `uninstalled=false` | 中 | 按固定幂等契约返回 `uninstalled=true` 和已知版本 |
| R13 | `controller/openclaw_skill_operations.go` | 直接卸载使用固定 `runtime` 锁前缀，跨运行时同 slug 会误冲突 | 中 | 锁键使用实际 `AgentType+slug`，三运行时测试断言具体锁键 |
| R14 | `controller/openclaw_skill_operations.go`、`controller/admin_skill_distribution.go` | 直接卸载构造空 task/nil records 的伪执行配置，两个同步 wrapper 只有单一调用点 | 低 | 直接卸载改为调用 ResolveScript/RunScript；同步调用点直接 build+execute，删除两个透传 wrapper |

首次审查中已完成的修复也重新核对：

- Enterprise 最新版本与可见性使用固定次数批量 DB 查询，无逐技能 DB N+1。
- Enterprise 更新脚本失败按 slug 回放当前安装状态，旧版本存在时记录 `upgrade_failed`。
- `update_available` 仅在 current/latest 均可规范化且 latest 更高时为 true。
- 卸载脚本失败使用卸载专用 i18n 文案。

### Reviewer 建议处理

| 建议 | 结论 | 依据 |
|------|------|------|
| 把新路由改成 Go 1.22 `POST /path` pattern | 不采纳并已撤回 | `main.go` 的主流生产约定是 path-only pattern，Handler 内执行 method guard；当前保留 `/openclaw/update-skill`、`/openclaw/uninstall-skill`，未引入第二套路由风格 |
| 移除 package-global Public 版本缓存 | 采纳 | `NewUserSkillHandlers` 构造三个包级 Handler 函数，版本缓存只由闭包持有并在三个函数间共享；不再暴露包级可变状态 |
| 为执行构造参数新增输入 struct/interface | 部分采纳 | 使用具体的局部依赖 struct 显式传入脚本执行和下载 URL 函数；不增加单实现 interface |
| 删除 TAT/锁测试函数 hook | 采纳 | 生产依赖由构造函数填充，测试仅覆盖各用例自己的依赖副本，不再修改包级函数变量或依赖 cleanup 恢复 |

仓库既有实现约定优先于孤立的文字规则。上一版 Review 错误接受 method-pattern 建议，本版结论已纠正，且不再把该改动列为修复。

## 安全、兼容与复杂度复核

- 所有外部 `slug` 在进入脚本模板前完成安全字符校验；未引入裸 SQL、跨租户 `model.DB`、直接 SDK `New`、硬编码密钥或未校验外部写请求。
- 两个写接口继续使用 `WithAudit(WithOpenAPI(...))`；面向用户错误继续走 i18n。
- 无 GORM schema 变更，因此不需要 `sql/init.sql` 或 migration。
- Public 缓存、任务执行核心与 Admin 现有流程复用；未增加单实现 interface、透传 wrapper 或兼容 shim。
- 未修改 `/openclaw/add-skill`、`/openclaw/skillstore/*` 语义。

## 验证证据

- [x] `go test ./model -run 'Test(NormalizeSkillVersion|ListDistributedSkillStates)' -count=1 -race`
- [x] `go test ./controller -run 'Test(PublicSkillVersionCache|FetchPublicSkillLatestVersion|ListPublicSkillLatest|EnrichDistributedSkillVersions|HandleUserUpdateSkill|HandleUserUninstallSkill|SkillOperationHandlers|SkillTaskExecution|ExecuteSkillTask|RunSkill)' -count=1 -race`
- [x] 修复收敛后再次运行锁、并发、终态与异步清理回归子集：通过
- [x] `go test . -run '^$' -count=1`
- [x] 变更 Go 文件 `gofmt -d`：无输出
- [x] `git diff --check`：通过
- [x] `bash -n` 三个列表脚本；`py_compile` 技能操作脚本、接口脚本与 helper
- [x] 三运行时脚本契约由实际 CVM IT 覆盖；未把外部 Shell 依赖包装成可跳过的 Go 单元测试
- [ ] 全仓 `pre-review.sh ./...` 被既有 gofmt 基线、controller typecheck 与 Go 1.25/1.26 工具链依赖问题阻断；本次 `go vet ./...` 与目标检查通过

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 无已确认的中严重度未修复问题
- [x] 路由、DB、i18n、审计与测试约定符合仓库主流实现
- [x] Review 修复均有目标测试或可重复冒烟证据

双轴审查累计确认 14 项问题（高 3、中 10、低 1），均已修复；3 项不符合仓库约定或属于推测性抽象的建议有依据地不采纳。Review 结论：通过。
