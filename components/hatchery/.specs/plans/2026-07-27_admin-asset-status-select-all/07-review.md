# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review
- [ ] 人工 Review

## 审查范围与标准

- Go：错误处理、context 传播、goroutine 生命周期、批内 semaphore、数据结构与命名。
- 项目：权限/审计、ORM 与租户隔离、i18n、API 兼容、文档和测试 DB/异步规范。
- Karpathy：仅审查本需求改动，不增加 speculative abstraction；保留三类资源各自不同的终态处理。

## 发现的问题

| # | 文件 | 行 | 问题 | 严重度 | 状态 |
|---|------|---|------|-------|------|
| 1 | `controller/admin_skills.go`、`admin_plugins.go`、`admin_skill_distribution.go` | select-all 创建失败分支 | `MsgSkillStoreCreateRecordFail` 含 `%v`，调用未传格式参数，会向用户返回 `%!v(MISSING)` | 中 | 已修 |
| 2 | `06-it.md` | 完整全选 IT | 基本显式下发脚本已通过，但真实 select-all 命中、分组隔离和 installed 重复下发尚未执行 | 中 | 按用户要求后续执行 |
| 3 | `controller/admin_distribution_select_all.go` | 共享 selection/状态/批大小 | 三类资源当前字段相同但后续契约可能分化，共享定义会形成无必要耦合 | 中 | 已拆回各资源并删除文件 |
| 4 | `controller/admin_distribution_select_all.go` | 查询/并发/失败收口 helper | 实例 GORM 逻辑、并发设施和具体 record 表被混放，泛型失败收口隐藏表结构约束 | 中 | 实例逻辑迁 `model`，失败收口改为具体函数，共享文件删除 |
| 5 | `controller/admin_distribution_select_all_test.go` | 跨资源测试归属 | 独立资源契约和 model 查询测试集中在共享 controller 测试文件，降低局部可维护性 | 低 | 测试迁回资源文件及 `model/instance_test.go`，原文件删除 |
| 6 | `common/concurrency.go` | `RunWorkers` | 执行已经按 200 条分批，再新增 worker 抽象与显式 ID 路径的 semaphore 模式不一致 | 中 | 删除，三类全选均复用现有 semaphore + WaitGroup |
| 7 | 三类 select-all task 创建循环 | `CreateInBatches(..., batchSize)` | 资源内 keyset 查询已保证当前批不超过 200，二次分批无实际作用 | 低 | 改为 slice `Create`，仍生成单条多行 INSERT |
| 8 | `model.WalkInstanceTargets(*gorm.DB, ...)` | 公共 API 边界 | 签名接受任意 query，却隐含 `instances` 表、alias、主键顺序和 SELECT 字段前提，易被误用 | 中 | 删除；target 和 keyset 循环内联至三类资源创建函数 |
| 9 | `controller/admin_mcp_distribute.go` | `mcpDistributeWG` | 仅供测试等待后台 goroutine，测试同步机制泄漏到生产代码 | 中 | 删除；测试按自身 task ID 带超时轮询 completed，并通过 `t.Cleanup` 保证异常断言后的等待 |
| 10 | 三类 select-all 异步执行 | goroutine panic | detached goroutine 未 recover 会终止进程，并可能留下 running task/pending record | 高 | task/record goroutine 入口均注册 recover，记录 stack 并收敛数据库状态后释放资源 |
| 11 | 三类 `run*SelectAllTask` | 内部启动 goroutine | 函数名无法表达隐式异步，调用方不能选择同步执行或直接测试 | 中 | runner 改为同步；detached context 与 `go` 外置到启动点 |
| 12 | 三类 `run*SelectAllTask` | 单函数混合分页、实例加载和执行 | 三类流程结构应一致，但加载字段、失败终态和执行参数不同，不适合共享抽象 | 中 | 各资源独立拆为 keyset 调度、load batch、execute batch 三阶段；不新增跨资源 helper |
| 13 | `POST /admin/skills/uninstall` | 仅支持显式 `instance_ids` | 管理员无法按状态/用户组批量选择企业或公共技能卸载目标 | 中 | 复用技能 select-all 创建和执行链路；独立限制卸载状态白名单 |
| 14 | `POST /admin/plugins/uninstall` | 仅支持显式 `instance_ids` | 管理员无法按状态/用户组批量选择插件卸载目标 | 中 | 复用插件 select-all 创建和执行链路；独立限制卸载状态白名单 |
| 15 | 三类 selection 与五套状态 normalizer | JSON 字段、目标模式校验及白名单校验/去重算法机械重复，修正规则容易遗漏资源 | 低 | 仅抽取公共 DTO 和纯校验算法；资源状态策略与执行链路保持独立 |
| 16 | 五个批量接口 `search` | 若各资源复制关键词截断与三字段 SQL，后续语义容易漂移 | 低 | 只共享 `applyDistributionSearch` 纯查询条件；资源 task/query 策略保持独立 |

## 人工 Review 调整

- 技能、插件、MCP 共用 `distributionSelection` 与目标模式校验；状态白名单和过渡状态由资源 wrapper 独立传入公共 normalizer。
- 三类资源分别保留数据库 `InstanceID` 和云侧 `InstanceCID` target：前者用于 record/FK，后者是 TAT 执行及任务创建时的云实例快照。
- `model` 仅保留用户组过滤；target、keyset 循环、批大小和写入全部归各资源 task 创建函数，批内并发复用现有 semaphore + WaitGroup。
- 共享 controller 文件仅包含 selection DTO 和纯校验算法；target、keyset、批大小、task/record、runner、失败收敛继续归各资源，不引入 interface、泛型、回调框架或 model walker。
- MCP 测试不再向生产代码注入 WaitGroup；按自身 task ID 轮询 completed。task recovery 直接 defer 在生产启动 goroutine，record recovery 保留在实际 record goroutine。
- 三类 `run*SelectAllTask` 为同步 runner；生产启动点显式 `go`，测试直接调用并立即断言，无需等待 runner 内部隐藏的 goroutine。
- 三类 runner 保持相同的 keyset 调度、load batch、execute batch 结构，但函数和资源终态逻辑相互独立，无 interface、泛型或单次复用 helper。
- 技能卸载全选复用同一 selection、企业/公共状态查询、用户组过滤、keyset 分批和 runner；只允许状态矩阵中“未卸载/卸载失败”的五种稳定状态。
- 插件卸载全选复用插件状态查询、用户组过滤、keyset 分批和 runner；失败状态沿用 `ResolvePluginUninstallFailedStatus`，卸载任务不增加 `distribute_count`。
- Review 要求去除批量全选参数和校验的机械重复后，采用窄抽象：三类 handler 共享五个 JSON 字段与一致模式校验，五套状态策略 wrapper 仅复用去重/白名单算法。
- 技能/插件下发与卸载定向 race 回归 `go test ./controller -race -count=2 -run '(DistributeSkill|UninstallSkill|SkillSelectAll|DistributePlugin|UninstallPlugin|PluginSelectAll|NormalizePlugin)'` 通过；`go vet ./model ./controller` 与 Go workspace diagnostics 通过。
- `search` 仅扩展公共 selection 和共同查询条件；未新增 interface、泛型、配置或资源策略抽象。LIKE 值参数化并复用 `escapeSQLLike`，用户名使用三类既有 `users u` LEFT JOIN。
- search 定向及相关批量接口 race 回归、`go vet ./controller`、Python IT 脚本语法检查和 Go workspace diagnostics 全部通过。

## 审查结论

- 无裸 SQL、无新增 schema、无硬编码密钥；所有新增输入均参数化。
- 三个 handler 均沿用 `requireAdmin`、`WithAudit` 和既有 audit rule。
- 请求 context 仅用于同步准备；异步执行和失败清理使用 detached context。
- 全选路径按 200 条 keyset 分页、分批落库；每批在启动 goroutine 前获取 semaphore 并等待整批完成，不按全部目标数创建 goroutine，也不存在 N+1 实例查询。
- 显式 ID API 保持兼容；插件仅新增 `total`，MCP 显式模式继续保留 `per_instance/warnings`。
- i18n 中英文、`docs/API.md` 和生成的 OpenAPI 三个新参数均已覆盖。

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致，未引入无必要抽象
- [x] 安全基线检查通过
