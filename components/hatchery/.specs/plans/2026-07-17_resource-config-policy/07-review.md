# 07. Review — 代码审查

---

## 审查方式

- [x] AI 自动 Review
- [ ] 人工 Review

审查基线：`origin/Release/2026_07_28`

审查范围：Resource Config Policy 的生产代码、GORM/MySQL schema、路由/审计/i18n、Go UT、Python IT 脚本、integration runner、API/模块文档和 Go module 元数据。

AI 审查由主审和 6 个并行专项审查组成：

1. 创建 overlay、镜像容量和云端预检；
2. API、权限、审计、缓存和 i18n；
3. model、`sql/init.sql` 和增量 migration；
4. 用户组 resolver、继承和配置总览；
5. integration runner、真实 CVM 清理和覆盖证据；
6. API/模块文档一致性。

## 核心结论

- Review 共确认 16 项问题：高 2、中 10、低 4；全部修复。
- 无高严重度未修复问题。
- 有效请求的站点/最近用户组整份策略、用户局部覆盖、镜像容量和机型 SELL 预检语义保持不变。
- 新增 fail-closed：持久化配置异常返回 500；用户覆盖与继承字段组合后无效返回 400；两者均早于 SG/CVM 副作用。
- `site_configs.resource_config` 新租户默认保持空值，继续实时 fallback 到 `CVMTemplate`，避免旧配置接口更新模板后被过期副本遮蔽。
- options cache hit/inflight waiter 不再依赖腾讯云 client 构造；三个 options handler 严格 GET-only；system-disks 只对匹配且 `SELL` 的机型查询 CBS。
- 真实 CVM IT 脚本改为 runner 独占执行，并在拿到 `cvm_id` 后立即登记清理；DB id 反查失败时可直接终止 CVM。

## 发现的问题

| # | 文件 / 符号 | 问题 | 严重度 | 状态 |
|---|-------------|------|--------|------|
| 1 | `controller/openclaw.go`、`validateAppliedResourceConfig` | 用户局部带宽/预付费覆盖只按局部字段校验，可绕过继承的 charge type 组合规则 | 中 | 已修：持久化层和用户层 apply 后分别校验有效请求；补 handler 回归 |
| 2 | `controller/resource_config.go`、`ApplyResourceConfigToRequest` | 局部 `instance_charge_prepaid` 会替换整个 SDK 子结构并把未提供字段清空 | 中 | 已修：只写正数/非空字段并保留已有 renew/period；补回归 |
| 3 | `controller/resource_config.go`、`ValidateResourceConfig` | 未显式带 charge type 时，非法 prepaid period/renew flag 不校验；负 disk/bandwidth 被当作省略 | 中 | 已修：provided prepaid 始终校验；0 仍表示省略，负数拒绝 |
| 4 | `controller/usergroup/resolve.go`、`ResolveResourceConfig` | 持久化 `{"value":null}`/数组/标量会被当作空组策略，静默压掉站点 fallback | 中 | 已修：wrapper `value` 必须为 JSON object；null/数组/string/number 回归通过 |
| 5 | `model/site_config.go`、`ApplySiteConfigDefaults` | 新租户把 `CVMTemplate` 复制进 `ResourceConfig`，后续旧 CVM 配置接口更新会被过期副本遮蔽 | 中 | 已修：默认保持 `ResourceConfig` 空值，fallback 始终读取当前模板 |
| 6 | `controller/resource_options.go` 三个公开 handler | POST/PUT 等非 GET 方法仍可进入 options 查询并返回 200 | 中 | 已修：方法校验先于鉴权，统一返回 405；三路回归通过 |
| 7 | `controller/resource_options.go` options handler | cache hit 前先构造 CVM/CBS client，临时凭证故障会让有效缓存不可用 | 中 | 已修：client 构造移动到 cache/inflight winner 内部；缓存命中 0 client call 回归通过 |
| 8 | `controller/resource_options.go` system-disks | 直接取 CVM quota 第一项，不要求请求机型匹配且 `Status=SELL` | 中 | 已修：遍历匹配 requested type + SELL；不可售时不构造/调用 CBS |
| 9 | `controller/admin_config.go`、`HandleUpdateResourceConfig` | 新 JSON 写接口使用无界 `io.ReadAll`，可被大 body 占用内存 | 中 | 已修：增加 1 MiB `http.MaxBytesReader`；超限 400 回归通过 |
| 10 | `controller/admin_user_group_tree.go`、`policyKeyOrder` | `resource_config` 声明为 user quota，却显示在 feature toggle 子分组且排序落在功能项末尾 | 低 | 已修：sub-label 以 `PolicyDef.Category` 为真相源；resource policy 移入 user-quota 顺序；总览回归通过 |
| 11 | `test/integration/runner.go` | 真实实例脚本修改站点配置和启用镜像行，但 runner 默认并发随机执行，可能交叉覆盖 fixture | 高 | 已修：该脚本在普通脚本全部结束后独占执行；新增真实 Python 进程调度回归 |
| 12 | `test_exclusive_instance_create_resource_config.py` | 创建成功后先反查 DB id、后登记 cleanup；反查失败会遗漏已分配 CVM | 高 | 已修：拿到 `cvm_id` 立即登记；cleanup 重试 DB id，失败则通过 cloud mutate 终止并等待终态 |
| 13 | `test_exclusive_instance_create_resource_config.py` | ImageSize=0 的可接受云失败只证明到达 RunInstances，未证明最终磁盘未被修改 | 中 | 已修：RunInstances 失败日志记录 `system_disk_size`；脚本断言期望容量 |
| 14 | `docs/API.md` | basic options 示例遗漏有效预付费周期 4、5、7–11 | 低 | 已修：示例与实际 1–12、24、36、48、60 完全一致 |
| 15 | `go.mod` | 生产代码直接 import CBS，但 module 被标记为 indirect | 低 | 已修：CBS v1.3.115 移入 direct require，版本不变 |
| 16 | `controller/openclaw.go` 注释、`docs/API.md`、`test/scripts/README.md` | 注释误称创建预检同时校验磁盘；文档未记录有效请求复验和独占 IT 调度 | 低 | 已修：明确创建预检只校验机型 SELL；补有效请求复验和 runner 独占契约 |

## 未采纳 / 误报项

| 项 | 结论 | 依据 |
|----|------|------|
| 创建前必须再次查询 CBS 校验最终系统盘 availability | 不采纳，不扩展本任务契约 | `docs/API.md` 明确创建前预检只校验最终 zone/charge type/instance type 的 SELL；system-disks 是管理员选项接口。代码误导注释已修正。 |
| migration 文件名与目标 Release 不一致 | 已修复 | 当前分支增量统一为 `sql/0728-resource-management.sql`，代码基线为 `Release/2026_07_28`。 |
| 三个 Resource Config IT 脚本不存在 | 误报 | 三个脚本均已在当前 worktree 落地并完成 IT；它们尚未进入 HEAD 是因为 SOP Commit 尚未开始，下一步统一 add/commit。 |
| broad grep 命中的 `db.Table` / `cvm.NewClient` | 非本分支新增 redline | `git diff origin/Release/2026_07_28` 证明命中分别位于未改动的既有逻辑和既有统一 client factory 内；本次新增差异未引入裸 SQL或绕过 factory。 |

## Schema / 安全 / 兼容性审查

### Schema

- GORM：`SiteConfig.ResourceConfig string gorm:"type:text"`。
- 新部署 MySQL：`sql/init.sql` 为 nullable `TEXT`。
- 现有 MySQL：统一执行 `sql/0728-resource-management.sql`，资源策略与实例调整 DDL 不重复。
- 非新模型/新表：无需加入 `allModels` 或 `MigrateFromSQLite`；SQLite 继续由现有 `SiteConfig` AutoMigrate 覆盖。

### 权限、审计和多租户

- `POST /admin/config/resource`：`requireAdmin`；`auditRules` 已注册；路由使用 `WithAudit(WithOpenAPI(...))`。
- 用户组 policy set/delete：既有 admin 权限、审计包装保持完整。
- options：admin-only、GET-only；cache key 含 tenant identifier、region 和全部 scope。
- 新 DB 访问继续使用请求 context；无新增 `model.DB` 全局越权路径。
- 新云调用继续通过 `GetCVMClient(ctx)` / `GetCBSClient(ctx)` 和 `CallSDKAPITyped`；未新增硬编码密钥、Token 或 Secret 日志。
- IT 的 bootstrap token 只经子进程环境传递；脚本不打印 token；直接 SQLite 操作仅用于隔离测试部署 fixture/异常清理，不进入生产代码。

### API 兼容性

- 未删除/重命名公共字段、operation 或已有参数。
- 未知 wrapper/ResourceConfig 字段继续兼容忽略。
- `disk_size=0` 继续表示不覆盖；历史 CVMTemplate 异常小盘继续保持原值，由镜像检查或 CVM 判断。
- 新增行为均为契约收紧：错误方法 405、非 object 组策略 fail closed、已知负值/无效组合拒绝。

## 验证证据

| 检查 | 结果 |
|------|------|
| `go test ./controller -run 'ResourceConfig\|ResourceOptions\|BuildCategories_PolicyEntries' -count=1` | PASS |
| `go test ./controller/usergroup -run 'ResolveResourceConfig\|ResourceConfig' -count=1` | PASS |
| `go test ./model -run 'ApplySiteConfigDefaults\|ExtractResourceConfig' -count=1` | PASS |
| 上述 controller / usergroup / model 定向命令追加 `-race` | PASS；无 DATA RACE |
| `go test ./... -count=1`（root module） | PASS：8 packages ok，1 package no tests |
| `go test ./... -count=1`（`test/integration` module） | PASS |
| `go vet ./...`（root + nested integration module） | PASS；无输出 |
| gopls diagnostics（本次生产文件） | PASS：`OK` |
| `python3 -m py_compile`（3 个 Resource Config IT 脚本） | PASS；无输出、无残留 pyc |
| 红线 grep：裸 SQL、直接 SDK New、硬编码 secret | 新增差异无命中；宽范围命中均经 baseline diff 确认为既有代码 |
| `go mod tidy -diff` | CBS direct 问题已消失；仍报告 baseline 已存在的 `robfig/cron/v3` indirect 分类和旧 common v1.3.73 checksum，本任务不扩大范围修改 |

Review 阶段没有重新部署 EKS。有效 I01–I12 路径已在前一 IT 阶段由隔离 EKS + 真实腾讯云通过；Review 新增的是错误分支、fail-closed、缓存/方法约束、独占调度和清理兜底，已由定向/竞态/全量 Go 测试及 Python 语法检查覆盖。

## 审查通过确认

- [x] 无高严重度未修复问题
- [x] 代码风格一致
- [x] 安全基线检查通过
- [x] API 兼容性检查通过
- [x] Schema 三处同步检查通过
- [x] Review 修复已由可观察行为测试覆盖

---

## 2026-07-22 资源策略实体化重设计 Review

> 本节审查最终独立 `ResourcePolicy` 实现，并覆盖本文前述 SiteConfig/分组内嵌资源配置版本的 Review 结论；冲突时以本节为准。

### 核心结论

- 确认 5 项问题：高 2、中 1、低 2；全部修复。
- 无高或中严重度未解决问题。
- 默认策略保留名、事务并发、租户隔离、双向索引、解析优先级、API 权限/审计、创建链路和干净切换契约一致。
- Review 修复未改变腾讯云创建、options 过滤或镜像容量行为，因此不重复部署；新增行为由模型/controller 定向竞态测试、nested runner 测试和 OpenAPI 断言验证。

### 发现与修复

| # | 文件 / 符号 | 问题 | 严重度 | 修复 |
|---|-------------|------|--------|------|
| 1 | `model/resource_policy.go`、Create/Update | 普通策略在默认策略尚未懒创建时可占用固定名“企业默认资源策略”，导致后续默认查询冲突并使列表不可用；普通策略也可在默认尚不存在时改成保留名 | 高 | Create/Update 均显式拒绝保留名；模型和 handler 回归验证 409、策略/范围不变，真实默认仍可创建 |
| 2 | `test/integration/main.go`、`runner.go` | 隔离 runner 将 bootstrap admin token、管理员 API token 和普通用户 token 明文写入控制台日志 | 高 | bootstrap token 只记录“generated”；init.py 输出在日志侧统一脱敏，原始值仅保留在进程内用于子进程环境；新增 `TestRedactInitOutput` |
| 3 | `model/group_config_binding.go` | GORM tags 只声明单列 `idx_gcb_group`，且缺少 `idx_gcb_resource`，与 `sql/init.sql` 的两个租户复合索引不一致；AutoMigrate schema 与 MySQL 初始化 schema 会漂移 | 中 | tags 对齐 `identifier,group_id,config_type` 和 `identifier,config_type,config_key`；新增索引列顺序回归测试 |
| 4 | `i18n/keys.go`、`en.go` | 默认策略保护错误仍写成“不可删除”，用于改名、绑定和保留名冲突时误导 | 低 | 中英文统一改为“企业默认资源策略受保护” |
| 5 | `docs/API.md`、旧测试注释 | API 未说明固定默认名称也是普通策略的保留名；两个创建错误测试注释仍引用已删除的旧 resolver | 低 | create/update 参数文档补保留名 409；注释改为独立 ResourcePolicy resolver；重新生成 OpenAPI |

### 模型与并发审查

- `ResourcePolicy.ConfigJSON` 是配置唯一真相源；`GroupConfigBinding.ValueJSON={}` 仅为空占位。
- 默认策略由 `(identifier,name)` 唯一索引和 `ON CONFLICT DO NOTHING` 并发安全懒创建；100 并发测试返回同一 ID。
- 普通策略更新先锁策略行，再按 ID 排序锁定所有目标 `UserGroup` 行；同组并发争抢只有一个成功。
- 创建/更新的分组占用校验、策略写入和全部绑定替换位于同一事务；冲突完整回滚。
- 一策略多组使用 `(identifier,config_type,config_key,group_id)` 前缀；按组查询使用 `(identifier,group_id,config_type)`。
- tenant callback 自动填充/过滤 `ResourcePolicy`、`UserGroup`、`GroupClosure` 和 `GroupConfigBinding`；跨租户策略查询和绑定测试通过。
- 删除普通策略在同一事务清理资源绑定；默认策略不可删除。

### API、安全和调用链审查

- 四个管理 API admin-only；三个写接口均经 `WithAudit` 且审计规则存在。
- 两个 options admin-only、GET-only，使用 tenant-aware cache 和统一 Tencent client/SDK wrapper。
- body 受 1 MiB 限制；ResourceConfig 只接受单个 JSON object，未知字段兼容忽略，已知字段规范化/校验。
- tree 使用批量直接策略查询，不把继承后代标成直接占用。
- 创建链路为 CVMTemplate → self/nearest/default 单一策略 → 用户覆盖 → `disk_type`；持久化坏配置返回 500，用户错误返回 400，镜像容量检查早于 VPC/SG/CVM。
- 旧 `/admin/config/resource*`、static basic、`SiteConfig.ResourceConfig`、`PolicyKeyResourceConfig`、`resource_policy_id` 和专用关联表均无生产/文档/脚本残留。
- 用户提供的 TCR 临时凭证未进入代码、文档或报告；远端已 logout。

### Schema 审查

- `ResourcePolicy` 已加入 `allModels` 和 SQLite→MySQL migration。
- `sql/init.sql` 与 `sql/0728-resource-management.sql` 的 `resource_policies` 表结构一致。
- SQLite→MySQL 先迁移策略 ID，再重映射 `GroupConfigBinding.config_key` 中的数字策略 ID。
- GORM binding index tags 与 MySQL SQL 的 `uk_gcb`、`idx_gcb_group`、`idx_gcb_resource` 列及顺序一致，回归测试通过。
- 未新增 `user_groups.resource_policy_id` 或 `resource_policy_groups`。

### 验证证据

| 检查 | 结果 |
|------|------|
| `go test -race ./model -run '^TestResourcePolicy' -count=1` | PASS；含保留名、100 并发、并发争抢、租户隔离和 schema index |
| `go test -race ./controller -run 'TestResourcePolicy\|TestGroupTreeReturnsDirectResourcePolicyOnly\|TestHandleCreateInstance_U3[1-6]' -count=1` | PASS |
| `go test ./... -count=1`（`test/integration`） | PASS；含 token redaction 回归 |
| `go vet ./...`（`test/integration`） | PASS |
| `go vet ./model ./controller ./controller/usergroup ./i18n .` | PASS |
| gopls diagnostics：resource model、binding model、controller、测试和 integration runner | PASS；均为 OK |
| `make openapi BASE_BRANCH=origin/Release/2026_07_15` | PASS；current 383 paths/393 ops，base 375/385 |
| OpenAPI 保留名、新旧路径和 `with_resource_policy` 断言 | PASS |
| 三个 Python IT 脚本内存编译 | PASS；无 pyc 残留 |

全仓 `go test ./... -count=1` 在非本功能测试基础设施上不稳定：本次 `controller` 的 `TestCreateSkillInstallTasks_RolePlusAllBundle` 在全包并发中触发 10 分钟超时，但精确单测复验 PASS；`task` 的 doctor cleanup 后台 goroutine 在测试 DB 清理后访问 nil/已关闭 DB 并 panic，单包复跑仍可复现。资源策略定向 `-race`、此前 UT 全仓 PASS 和隔离 EKS IT 均通过；没有把本次全仓失败虚报为通过。

全仓 `go vet ./...` 仍只报告既有 `skillhubclient/client_test.go:278` 在检查 error 前使用 response；受影响包和 nested integration vet 均通过。

### 审查通过确认

- [x] 无高/中严重度未修复问题
- [x] 默认策略保留名和保护语义完整
- [x] GORM / init SQL / migration 索引契约一致
- [x] 权限、审计、多租户和凭证日志安全检查通过
- [x] 解析、创建和干净切换契约一致
- [x] Review 修复均有可观察行为测试

### 2026-07-22 默认策略名称 i18n 调整

用户指出默认策略名称必须在写入或所有读取点完成 i18n，且不能影响普通策略。本次采用**读取时本地化**：

- 数据库继续保存稳定 canonical 名称“企业默认资源策略”，避免首次请求语言改变租户数据和唯一索引语义；
- `ResourcePolicy.DisplayName(ctx)` 只在 `IsDefault=true` 时调用 i18n；普通策略始终返回管理员原始名称；
- 管理列表、配置总览和用户组树直接策略元数据三个展示读取点统一调用 `DisplayName`；
- 中文返回“企业默认资源策略”，英文返回 `Enterprise Default Resource Policy`；
- 英文客户端把读取到的本地化默认名称原样回传 update 时视为未改名；真正的其他名称仍返回 409；
- 新增 `MsgDefaultResourcePolicyName` 中英文翻译，API、README 和集成脚本同步。

验证：

- handler 测试同时断言英文默认名称、普通中文策略名称不变、英文名称 round-trip update 成功、数据库 canonical 名称不变；
- 资源策略模型/controller 定向 `-race` PASS；
- 受影响包 vet 和 gopls diagnostics PASS；
- OpenAPI 383 paths/393 operations 重新生成，默认展示名称契约断言 PASS；
- 更新后的 Python IT 脚本内存编译 PASS。

该调整只改变展示读取和默认策略 update 的本地化 round-trip，不改变持久化、索引、解析或腾讯云创建链路，因此未重复执行 EKS/CVM IT。
