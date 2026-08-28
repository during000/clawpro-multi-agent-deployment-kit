# 02. Plan — 方案设计

---

## 设计结论

采用三层复用，不直接调用 Admin HTTP Handler：

1. **状态层**：从 `skill_distribution_tasks` + `skill_distribution_records` 还原同实例、同 slug 的当前有效安装状态。
2. **分发层**：复用 Admin 的版本解析、Public 制品准备、Agent 脚本、分布式锁、task/record 与失败终态。
3. **用户接口层**：只处理登录与实例归属、running/能力准入、幂等、Enterprise 可见性和同步响应。

同步执行能力下沉到现有技能任务执行器；Admin 继续异步调用同一个执行核心。Public 最新版本缓存是具体模块，不增加只有一个实现的 Go interface，也不引入新依赖。

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `model/skill.go` | 修改 | 增加 Admin 下发技能当前状态结构与批量查询；按成功 distribute/uninstall 事件序列还原跨 Public/Enterprise 的有效来源、版本、SkillID |
| `controller/public_skill_version.go` | 新增 | 封装 Public 最新版本 HTTP 查询、5 分钟缓存、过期降级及同 slug 并发请求合并 |
| `controller/openclaw_skill_operations.go` | 新增 | 列表版本字段补充、同步更新/卸载 Handler、公共前置校验与响应结构 |
| `controller/openclaw_skill.go` | 修改 | `HandleSkillsList` 从原始 JSON 透传改为解析后补充 Admin 下发版本字段；保留现有 TAT 分块/压缩读取 |
| `controller/skill_task_executor.go` | 修改 | 从现有异步执行器提取可同步调用的执行核心；异步入口仅负责 goroutine/WaitGroup 包装 |
| `controller/admin_skill_distribution.go` | 修改 | 抽取下发/卸载任务配置与单实例执行器构造，供 Admin 异步入口和用户同步入口共同调用 |
| `scripts/list_skills.sh` | 修改 | 保留 OpenClaw workspace 安装目录 basename 作为 `slug`；命中用户目录时返回 `can_uninstall=true`，仅 CLI 可见的内建技能返回 false |
| `scripts/list_skills_hermes.sh` | 修改 | 为直接安装的技能条目保留目录 basename `slug` 并返回 `can_uninstall=true`；不扩展到本期范围外的分类目录修复 |
| `scripts/list_skills_ace.sh` | 修改 | 以目录匹配后的 LightClaw 技能标识作为 `slug`；未命中用户技能目录的 builtin 条目返回 `can_uninstall=false` |
| `main.go` | 修改 | 注册 `/openclaw/update-skill`、`/openclaw/uninstall-skill`，均使用 `WithAudit(WithOpenAPI(...))` |
| `controller/audit.go` | 修改 | 增加用户技能更新、卸载审计规则 |
| `i18n/keys.go`、`i18n/en.go` | 修改 | 增加“未安装/最新版本查询失败/更新失败/卸载失败”等用户可见中英文文案；能复用现有 Key 的不新增 |
| `model/distributed_skill_state_test.go` | 新增 | 覆盖 Admin 下发状态事件折叠、跨来源覆盖、失败记录不改变当前版本及隔离条件 |
| `controller/public_skill_version_test.go` | 新增 | 覆盖仓库响应解析、TTL、过期降级、更新拒用过期值、并发合并与取消 |
| `controller/openclaw_skill_operations_test.go` | 新增 | 覆盖列表补充及两个用户 Handler 的权限、幂等、成功、失败、锁与来源分支 |
| `controller/skill_task_executor_test.go` | 新增 | 证明同步执行返回终态，异步入口仍更新 task/record 并释放锁 |
| `test/scripts/openclaw_instance/test_instance_skill_ops.py` | 修改 | 增加新路由的认证、必填参数和非本人实例契约覆盖 |
| `test/scripts/skill/test_user_skill_operations.py` | 新增 | 真实 Admin 下发后，以用户身份验证列表版本、更新、重复更新、卸载和重复卸载闭环 |
| `docs/API.md` | 修改 | 更新技能列表响应，新增两个同步写接口、状态码、幂等与失败语义 |

## 接口契约

### `GET /openclaw/skills?id=<instance_db_id>`

- 原有 `name`、`description`、`eligible` 保持兼容，所有实时列表项返回稳定目录 `slug` 和布尔值 `can_uninstall`。
- 仅当 slug 命中当前有效 Admin 分发记录时额外返回：
  - `version`
  - `latest_version`
  - `update_available`
- 无有效 Admin 下发记录的技能不返回版本和更新字段；数据库中有记录但实时列表不存在的技能不补造。
- `update_available` 仅在两个版本均为合法 `x.y.z` 且 latest 严格大于 current 时为 true。
- `can_uninstall` 在技能命中当前运行时的用户可管理目录时为 true；仅由 CLI 暴露、没有用户目录的内建技能为 false。

### `POST /openclaw/update-skill`

- Content-Type：`application/x-www-form-urlencoded`
- 字段：`id`、`slug`
- 成功响应固定字段：`slug`、`updated`、`old_version`、`version`
- 当前未安装或没有有效 Admin 下发记录：404。
- 已是最新版：200，`updated=false`，不创建 task/record，不执行脚本。
- 同技能被 Admin 或用户操作锁定：409。
- Public 仓库缓存过期且刷新失败：返回可重试的上游错误，不创建 task/record。

### `POST /openclaw/uninstall-skill`

- Content-Type：`application/x-www-form-urlencoded`
- 字段：`id`、`slug`
- 成功响应固定返回 `slug`、`uninstalled`；Admin 下发技能另返回已知 `version`
- 无 Admin 下发记录：获取 runtime+slug 锁后直接同步执行一次物理卸载，不查询实时列表，不创建 task/record
- 脚本成功统一返回 `uninstalled=true`；重复卸载依赖脚本自身幂等
- Admin 下发技能：同步物理删除，脚本成功后写 uninstall success
- 同技能被其他用户操作锁定，或 Admin 下发状态在锁内变化：409

## 调用链 / 数据流

### 列表

```text
GET /openclaw/skills
  → requireLogin + getInstanceByID
  → listInstanceSkills（现有 Agent 类型脚本 + TAT 分块/压缩读取）
  → 解析 runtime items（name 与内部 slug 分离）
  → model.ListDistributedSkillStates(instanceID, runtime slugs)
      → 只读取 success 的 distribute/uninstall
      → 按 record ID 顺序折叠每个 slug 的物理状态
  → Enterprise：批量取 skills 表同 slug 最新版本
  → Public：publicSkillVersionCache.Get(slug, allowStale=true)
      → fresh cache / 同 slug inflight 等待 / HTTP 刷新 / stale fallback
  → 仅补充命中 Admin 下发状态的 item
  → JSON array
```

### 更新

```text
POST /openclaw/update-skill (id, slug)
  → 登录、本人实例、非 Local、Agent 支持技能、running
  → 查询当前 DistributedSkillState
  → Public：获取未过期 latest；Enterprise：取 latest 并校验用户可见性
  → 按 Admin 规则获取 skill_dist 分布式锁
  → 锁内重查当前状态与来源
  → latest <= current：直接 200 updated=false
  → prepareDistributeSkillItem（目标为确定版本）
  → failPreviousPendingSkillDistribute
  → createSkillTaskAndRecords（单实例）
  → buildSkillDistributeExecution + executeSkillTask
      → ResolveScript(install_skill_from_smh, agentType)
      → RunScript
      → record success/upgrade_failed
      → task completed
      → release lock
  → 返回最终版本
```

### 卸载

```text
POST /openclaw/uninstall-skill (id, slug)
  → 与更新相同的实例准入
  → 查询当前 DistributedSkillState
  → 有有效 Admin 下发状态：
      → 按 Admin 规则获取 skill_dist 锁并锁内重查
      → createSkillTaskAndRecords（记录实际来源/版本）
      → buildSkillUninstallExecution + executeSkillTask
      → record/task 终态并释放锁
  → 无有效 Admin 下发状态：
      → 获取 runtime+slug 锁并锁内重查 Admin 下发状态
      → ResolveScript(uninstall_skill, agentType) → RunScript（通过局部依赖注入）
      → 不创建 task/record，不查询实时技能列表
  → 返回卸载结果；仅 Admin 下发技能返回已知版本
```

## 关键实现约束

1. 所有 DB 访问使用 `model.DB(ctx)` 和 GORM；不使用 `Raw`、`Exec`、`Table`、`Row(s)`。
2. 当前状态只由成功操作改变；pending、failed、upgrade_failed、uninstall_failed_old 不改变物理状态推导。
3. Enterprise 历史 task.slug 为空时，允许通过 record.skill_id 关联 `skills.slug` 兼容旧记录。
4. 列表批量查询状态与 Enterprise 最新版本，禁止按技能逐条查 DB。
5. Public cache 在进程内按 slug 全局共享；受 mutex 保护。inflight 等待支持 `ctx.Done()`，不启动无生命周期 goroutine。
6. Public HTTP 响应限制读取大小；优先解析 `latestVersion.version`，兼容 `skill.tags.latest`，非法版本不进入成功缓存。
7. 列表可使用过期 Public 缓存；更新只能使用 TTL 内缓存或本次成功刷新值。
8. 用户同步执行使用请求 context；Admin 异步入口继续使用 detached context。执行核心负责所有终态写入和锁释放。
9. 不修改 `/openclaw/add-skill`、`/openclaw/skillstore/*` 语义，不支持 Local Agent、agent_id 或后台轮询；无 Admin 下发记录的技能只支持按实时 slug 卸载，不支持版本管理和更新。
10. 不修复 Hermes 分类目录中的技能发现问题；仅卸载列表脚本实际返回的直接目录 slug。

## 数据库变更

无。复用现有：

- `skill_distribution_tasks`
- `skill_distribution_records`
- `skills`

不修改 GORM model 字段、`sql/init.sql` 或 migration SQL。

## 测试用例设计（自然语言描述）

> 先于实现编写，Implement 阶段据此编码。每个用例验证可观察契约，不断言源码文本。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| U1 | 首次成功下发 | 同实例/slug 一条 distribute success | 状态为已安装，来源、版本、SkillID 取该记录 | P0 |
| U2 | 成功卸载清除状态 | distribute success 后 uninstall success | 状态为未安装 | P0 |
| U3 | 失败和 pending 不改变当前版本 | 旧版 success 后追加新版 pending、failed、upgrade_failed | 当前版本仍为旧版 | P0 |
| U4 | Public/Enterprise 同 slug 覆盖 | 两来源交错成功 distribute/uninstall | 最后一条成功物理操作决定状态与来源 | P0 |
| U5 | 历史 Enterprise task.slug 为空 | task 仅有 skill_id，skills 表存在同 slug | 仍能还原当前版本 | P0 |
| U6 | 状态查询隔离 | 不同实例、slug、tenant 混合记录 | 只返回当前 ctx、实例和候选 slug | P0 |
| U7 | Public 详情主字段解析 | 仓库返回 `latestVersion.version` | 返回规范化版本并写 5 分钟缓存 | P0 |
| U8 | Public 详情兼容字段解析 | 主字段为空、`skill.tags.latest` 有值 | 返回 fallback 版本 | P1 |
| U9 | Public fresh cache | 同 slug TTL 内调用两次 | 只发起一次 HTTP 请求 | P0 |
| U10 | Public 同 slug 并发刷新 | 多 goroutine 同时冷查询 | 仅一个请求访问仓库，所有调用得到同一结果 | P0 |
| U11 | Public 列表过期降级 | 有过期值且刷新失败，allowStale=true | 返回过期版本且不报错 | P0 |
| U12 | Public 更新拒用过期值 | 有过期值且刷新失败，allowStale=false | 返回错误，不返回旧值 | P0 |
| U13 | Public 无缓存失败 | 首次查询失败 | 列表补充为空 latest/false；更新返回错误 | P0 |
| U14 | Public 非法版本/超大响应 | 仓库返回非法版本或超过上限 | 不缓存并返回可诊断错误 | P1 |
| U15 | 列表仅补充实际 Admin 下发技能 | runtime 有 Admin 下发和其他技能各一项，DB 另有一项但 runtime 缺失 | 两项均保留 slug；仅命中下发记录的条目有版本三字段；不补造第三项 | P0 |
| U16 | 显示名与 slug 不同 | runtime item `name != slug`，DB 以 slug 记录 | 正确保留 name 和目录 slug，并按 slug 补充 Admin 下发字段 | P0 |
| U17 | 版本严格比较 | current 分别小于、等于、大于 latest | 仅小于时 `update_available=true` | P0 |
| U18 | 更新通用准入 | 未登录、非本人、Local、未知/不支持 Agent、非 running、空或非法 slug | 返回对应 4xx；无 task/record/脚本调用 | P0 |
| U19 | 更新未安装 | 无有效 Admin 下发状态 | 404；无副作用 | P0 |
| U20 | 已是最新版 | current == latest | 200、`updated=false`、`old_version`/`version` 均为当前版本；无 task/record/脚本调用 | P0 |
| U21 | Public 更新成功 | 旧 Public 版本、仓库返回更高确定版本、脚本成功 | 以确定版本准备包；同步返回新版本；task/record completed+success | P0 |
| U22 | Enterprise 更新成功与可见性 | 最新 Enterprise 对用户可见/不可见两种输入 | 可见时成功更新；不可见时 404 且无副作用 | P0 |
| U23 | 更新脚本失败 | 已安装旧版，安装脚本返回错误 | 接口失败；record 为 upgrade_failed；task completed+failed；旧版仍为当前 | P0 |
| U24 | 更新锁冲突 | Admin 同技能锁已持有 | 409；无新 task/record/脚本调用 | P0 |
| U25 | 无 Admin 下发状态直接卸载 | 无有效下发状态、脚本成功 | 执行一次卸载脚本并返回 `uninstalled=true`；无 task/record | P0 |
| U26 | 卸载成功 | 有效 Admin 下发状态、Agent 卸载脚本成功 | 同步返回原版本；uninstall task/record success；状态变未安装 | P0 |
| U27 | 卸载失败 | Agent 卸载脚本返回错误 | 接口失败；失败记录保留；原版本仍为当前 | P0 |
| U28 | 卸载锁冲突 | 同技能锁已持有 | 409；无新副作用 | P0 |
| U29 | 三运行时脚本路由 | OpenClaw、Hermes、LightClaw ACE 各执行更新/卸载 | 分别选择对应 install/uninstall 脚本并传 `skill_slug`、版本、下载地址 | P0 |
| U30 | 同步执行核心 | 单 record 执行成功/失败/DB 写失败 | 返回最终 error，task/record 终态正确，锁恰好释放一次 | P0 |
| U31 | Admin 异步行为回归 | 现有批量任务调用异步包装 | Handler 不等待；后台仍写终态、统计并释放锁 | P0 |
| U32 | 直接卸载其他运行时技能 | 三运行时无 Admin 分发状态 | 每次请求只选择并执行一次对应卸载脚本；首次和重复请求均为 true；不创建 task/record、不返回未知 version | P0 |
| U33 | 列表卸载能力 | OpenClaw/ACE 分别返回用户目录技能和仅 CLI 可见的内建技能，Hermes 返回直接目录技能 | 用户目录技能 `can_uninstall=true`；内建技能 false；管理字段补充不改变该值 | P0 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| I1 | 新接口认证与参数 | 无登录/无 id/无 slug/非本人实例请求两个 POST | 覆盖所有新增参数；均返回稳定 4xx 且无副作用 | P0 |
| I2 | Enterprise 用户更新闭环 | Admin 创建并下发 v1，再创建同 slug v2；用户查询并更新 | 列表返回 slug、v1、v2、true；更新同步返回 v1→v2；再次查询为 v2/false | P0 |
| I3 | 更新幂等 | I2 完成后再次更新 | 200、`updated=false`，`old_version`/`version` 均为 v2 | P0 |
| I4 | Enterprise 用户卸载闭环 | I2 的已更新技能执行卸载 | 200、`uninstalled=true`、version=v2；实时列表不再出现该技能 | P0 |
| I5 | 卸载幂等 | I4 完成后再次卸载 | 再次执行幂等卸载脚本，200、`uninstalled=true`，不新增分发记录 | P0 |
| I6 | Public 最新版本与卸载 | Admin 以确定版本下发稳定 Public fixture，用户查询并卸载 | 列表从公共仓库返回 latest/update_available；卸载同步成功且列表移除 | P0 |
| I7 | Public 实际更新 | Admin 下发稳定 Public fixture 的旧版本，用户调用更新 | 安装仓库当前最新版，响应和后续列表版本一致 | P0 |
| I8 | 三运行时覆盖 | 对可用的 OpenClaw、Hermes、LightClaw ACE fixture 执行同一生命周期 | 三种 Agent 均走正确脚本并满足同一外部契约 | P0 |
| I9 | 审计日志 | 用户成功更新、卸载 | 审计记录分别包含新 action 与 skill resource | P1 |
| I10 | 其他运行时技能卸载 | 用户直接安装 Enterprise 技能后调用卸载 | 列表返回 slug 但无版本字段；物理删除成功；无 Admin 分发记录；重复请求仍返回 true | P0 |
| I11 | 列表卸载能力 | 查询包含普通安装技能与运行时内建技能的实例 | 普通技能 `can_uninstall=true`；内建技能 false；前端无需根据 slug 猜测 | P0 |

## 实施顺序

1. 先实现并验证 Admin 下发状态事件折叠与 Public 版本缓存。
2. 再提取同步任务执行核心，跑现有 Admin 技能分发相关测试证明行为不变。
3. 接入列表补充，再实现更新与卸载 Handler。
4. 最后注册路由/审计/i18n，补齐 API 文档与 IT。

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| R1 | 将最新 pending/failed 目标版本误认为当前版本 | 高 | 高 | 只折叠成功物理操作；U1-U6 覆盖交错序列 |
| R2 | OpenClaw 显示名与目录 slug 不同导致操作错技能 | 高 | 高 | 脚本内部保留目录 slug；请求与响应统一使用 slug；U16 |
| R3 | Public 仓库故障拖慢列表或引发请求风暴 | 中 | 高 | TTL、同 slug inflight 合并、有限并发、响应上限、列表 stale fallback |
| R4 | 同步提取破坏既有 Admin 异步任务统计或锁释放 | 中 | 高 | 同一执行核心 + U30/U31 + 现有 Admin 分发测试回归 |
| R5 | Enterprise 版本更新绕过可见范围 | 中 | 高 | 更新前复用 `IsSkillVisibleToUser`；不可见按 404 处理 |
| R6 | 跨 Public/Enterprise 同 slug 历史导致来源误判 | 中 | 高 | 以成功事件的全局顺序折叠，不按来源分别取“最新” |
| R7 | Public E2E fixture 被仓库删除或版本变化 | 中 | 中 | 测试配置固定 slug/旧版本，并在用例开始先验证 fixture；仓库协议细节由可控 HTTP UT 兜底 |
