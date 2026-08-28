# 03. Implement — 实现记录

---

## 关键实现细节

1. 技能、插件、MCP 共用 `distributionSelection` JSON 字段和严格一致的目标模式校验；各资源的下发/卸载状态白名单仍留在资源文件独立定义，避免业务策略耦合。
2. 三类资源分别复用现有安装状态 CASE，并以请求解析后的目标版本计算状态。空状态展开为稳定状态全集；显式过渡状态和非法状态在资源查询、加锁前返回 400。
3. 用户组筛选由 `model.FilterInstancesByUserGroups` 统一实现：多个组取并集，`0` 表示没有任何用户组成员关系的用户；子查询避免多组成员 JOIN 扇出。
4. 三类资源各自在 task 创建函数内以 `instances.id` 做 keyset 分页，并定义独立 target 与 200 条批大小；同步创建全部 task records 后才返回响应，形成固定目标集合。MCP 同批事务 upsert installation 为 `installing`。
5. 异步执行只读取已固化 records，每批批量加载实例信息；启动 goroutine 前通过现有 `SkillDistributeConcurrency` semaphore 获取槽位，等待当前批完成后再读取下一批。
6. 多技能模式逐技能独立解析目标集合，顶层 `total` 保留技能项数量，各结果通过 `instance_count` 返回实例数。
7. MCP 全选响应只含 `task_id/total`；显式 ID 响应继续保留 `per_instance/warnings`。插件显式模式补充实际目标数 `total`。
8. 准备阶段失败时使用 detached context 清理或终结已创建数据，避免请求取消导致永久 running task；MCP 测试通过带超时轮询自身 task 终态等待后台执行，不向生产代码注入测试 WaitGroup。
9. 新增三类独立模式/状态校验、分组、能力过滤、零目标、跨批次、不限量、三类 handler 和多技能独立筛选测试。
10. 三类 select-all 的 task panic recovery 均直接 defer 在生产启动 goroutine；批内 record goroutine 在各自 execute 阶段注册 recovery。两层均记录 stack、收敛数据库终态，并由 defer 保证已启动 record、semaphore、锁和测试 WaitGroup 正常释放。
11. `runSkillSelectAllTask`、`runPluginSelectAllTask`、`runMCPSelectAllTask` 均为同步函数；goroutine 和 detached context 在 handler/批量提交启动点显式创建，测试可直接同步调用 runner。
12. 三类 runner 各自拆为资源专属的 keyset 调度、`load*SelectAllBatch` 与 `execute*SelectAllBatch`；不引入跨资源 interface、泛型或阶段 helper，保留技能、插件、MCP 不同的加载与终态语义。
13. 技能和插件卸载接口分别复用资源内 selection、状态查询、用户组过滤、keyset task/record 创建和 select-all runner；卸载全选仅允许 `installed/outdated/upgrade_failed/uninstall_failed/uninstall_failed_old`，执行阶段按 task type 调用现有卸载脚本，卸载成功不增加 `distribute_count`。MCP 无管理员批量卸载接口，不扩展。
14. Review 调整仅抽取 selection DTO、目标模式校验和状态白名单校验/去重算法；不共享 target、批大小、查询、keyset、task/record、runner 或失败终态，不引入 interface、泛型和回调框架。
15. `distributionSelection` 新增 `search`；公共 `applyDistributionSearch` 仅封装三资源完全一致的 SQL 条件、50-rune 截断和 LIKE 转义。Skill/Plugin/MCP 的状态、分组、能力过滤及 task 创建仍保持资源独立。三类 IT helper 可构造全选请求，现有重装后二次下发继续覆盖显式 ID 兼容路径。

## 与 Plan 差异

功能契约无差异。人工 Review 后删除跨资源的 controller/model 分页抽象：资源选择、target、keyset 循环和批大小归各自 controller，`model/instance.go` 仅保留用户组过滤；异步批次执行复用各资源既有 semaphore，泛型 pending-record 收口改为技能/插件具体函数。`go vet ./controller ./i18n` 通过；`go vet ./...` 被未改动的 `skillhubclient/client_test.go:278`（先使用 `resp` 后检查 error）阻断，本任务未修改该文件，留待 UT 阶段记录。

## 检查项

- [x] `gofmt` 格式化通过
- [ ] `go vet ./...` 无错误（本次改动包通过；全量被 `skillhubclient/client_test.go:278` 阻断）
- [x] 写接口审计沿用现有路由规则（未新增端点）
- [x] 无数据库 schema 变更，无需更新 `sql/init.sql` 或 migration SQL
- [x] 数据库访问均传递请求或 detached context
- [x] 无硬编码密钥/配置
- [x] 用户可见文案使用 i18n Key，已同步 `en.go` 英文翻译
