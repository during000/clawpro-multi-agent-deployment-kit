# 02. Plan — 方案设计

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `model/instance.go` | 修改 | 提供实例用户组过滤 |
| `controller/admin_skills.go` | 修改 | 技能下发解析 `select_all/statuses/group_ids`，单技能模式接入全选目标解析，返回实际 `total` |
| `controller/admin_skill_distribution.go` | 修改 | 多技能逐技能独立筛选；技能目标在资源内 keyset 分批建 record；批内复用 semaphore 执行 |
| `controller/admin_plugins.go` | 修改 | 插件下发/卸载的目标状态筛选、资源内 keyset 分批建 record、批内 semaphore 执行及 `total` 响应 |
| `controller/admin_mcp_distribute.go` | 修改 | MCP 状态/分组全选、资源内 keyset 分批建 record 与 installation、批内 semaphore 执行；全选响应不展开 `per_instance` |
| `controller/admin_distribution_selection.go` | 修改 | selection 新增 `search`，复用 `/admin/instances` 的三字段模糊匹配和 50 字符上限 |
| `model/instance_test.go` | 新增 | 实例用户组筛选测试 |
| `controller/admin_skills_distribute_test.go` | 修改 | 技能 selection/状态、单项/多项全选、跨批次、目标版本、分组与兼容性测试 |
| `controller/admin_plugins_test.go` | 修改 | 插件 selection/状态、下发/卸载全选、跨批次、数量限制兼容和汇总响应测试 |
| `controller/admin_mcp_test.go` | 修改 | MCP selection/状态、全选、跨批次、installation 和汇总/明细响应兼容测试 |
| `i18n/keys.go`、`i18n/en.go` | 修改 | 新增目标模式冲突、非法/过渡安装状态等中英文错误文案 |
| `docs/API.md` | 修改 | 更新三个 distribute 与技能/插件 uninstall 请求参数、状态枚举、分组语义、数量与响应契约 |
| `test/scripts/helpers/admin_skill.py` | 修改 | 下发助手支持显式 ID 或全选参数 |
| `test/scripts/helpers/admin_plugin.py` | 修改 | 下发助手支持显式 ID 或全选参数 |
| `test/scripts/helpers/admin_mcp.py` | 修改 | 下发助手支持显式 ID 或全选参数 |
| `test/scripts/asset_distribution/test_admin_asset_select_all.py` | 新增 | 使用隔离用户组和实例验证技能、插件、MCP 的状态+分组全选端到端链路与新增参数覆盖 |

## 调用链 / 数据流

```text
POST /admin/{skills|plugins|mcp}/distribute
  → requireAdmin / requireSMHEnabled（保持现状）
  → 解析资源参数 + 各资源独立 selection
  → 校验目标模式：
      A. select_all=false：instance_ids 非空，走现有显式 ID 限制
      B. select_all=true：instance_ids 必须为空，规范化 statuses/group_ids/search
  → 解析实际目标版本（latest 或明确版本）
  → 获取现有资源级分布式锁
  → 目标解析：
      A. 显式 ID：保持现有查询和 Agent 能力过滤
      B. 全选：
         Build*InstanceQuery(目标版本)
         → 安装状态 WHERE
         → 用户组半连接过滤 + search 模糊匹配实例名称/实例 ID/创建人用户名
         → Agent 能力过滤（不增加 running 条件）
         → instances.id keyset 分页，每批 200，批内按 ID 去重
  → 同步准备任务：
      第一批为空 → 400，不创建任务
      创建 task
      每批目标 → Create(records) 多行写入
      MCP 同批 upsert installations=installing
      完成后写 task.total；至此目标快照固定
      任一准备批失败 → 不启动执行器，清理/终结已准备的任务数据，返回 500
  → 返回 task_id + total
      MCP select_all 不返回 per_instance
      显式 ID 响应保持原形状
  → 启动点构造 detached context，并显式 `go` 调用同步 `run*SelectAllTask`
      → 启动 goroutine 入口 defer task panic 收敛与分布式锁释放
      → runner 按资源拆分为 keyset 调度、`load*SelectAllBatch` 和 `execute*SelectAllBatch`
      → load 阶段按 task_id + record.id keyset 读取 pending records，并批量加载实例 runtime_user/agent_type/source
      → execute 阶段按 SkillDistributeConcurrency semaphore 限制活跃 record goroutine
      → record goroutine 执行现有 RunScript 与逐 record 状态落库；入口 defer panic 收敛
      → 等待当前批完成后读取下一批，所有批次完成后更新 task success/failed/status
```

### 关键设计

1. **独立选择契约**：技能、插件、MCP 分别定义选择结构和校验，当前 JSON 字段一致，但允许后续按资源独立演进。
2. **状态白名单**：
   - 技能下发：`uninstalled/installed/outdated/failed/upgrade_failed/uninstall_failed/uninstall_failed_old`。
   - 插件下发：`uninstalled/installed/outdated/failed/upgrade_failed/uninstall_failed/uninstall_failed_old`。
   - MCP 下发：`uninstalled/installed/outdated/failed`。
   - 技能/插件卸载：`installed/outdated/upgrade_failed/uninstall_failed/uninstall_failed_old`。
   - 技能/插件拒绝 `installing/uninstalling`；MCP 拒绝 `installing`。空 `statuses` 展开为对应操作的稳定状态全集，确保 SQL 明确排除过渡态和不可卸载状态。
3. **目标版本基准**：资源准备完成后才构造状态查询；企业/公共技能、插件和 MCP 均把解析出的目标版本传给现有状态 CASE。
4. **用户组过滤**：使用 `UserGroupMember` 子查询做半连接，避免一个用户属于多个命中组时 JOIN 扇出；`group_ids` 先去重。
5. **有界资源使用**：目标和 records 不形成无界总切片。查询、写入、异步读取均采用 200 条批次；批内 semaphore 继续读取 `SkillDistributeConcurrency`，默认 100。
6. **固定目标快照**：异步执行只读取当前 task 已持久化的 records，不重新按 status/group 查询。
7. **兼容切割**：显式 ID 模式的校验、插件/MCP 500 上限和 MCP `per_instance/warnings` 保持；脚本参数和终态判定不变。
8. **技能多项模式**：`skills[]` 共用选择条件，但每个技能在拿到自己的目标版本后独立生成 task 和目标 records；现有 batch_id 与部分提交成功语义不变。
9. **卸载复用**：技能与插件卸载复用各自 select-all task/record 快照和 runner，通过 task type 切换卸载脚本与失败状态；MCP 无管理员批量卸载接口，不扩展。
10. **搜索过滤**：`search` 仅允许全选模式，按 rune 截断为 50 字符并使用 `escapeSQLLike`；三类资源 query 已有 `users u` JOIN，统一匹配 `instances.name/instances.instance_id/u.username`。

## 数据库变更

无。继续使用现有：

- `skill_distribution_tasks` / `skill_distribution_records`
- `plugin_distribution_tasks` / `plugin_distribution_records`
- `mcp_distribution_tasks` / `mcp_distribution_records` / `mcp_installations`
- `user_group_members`

不新增字段、索引或迁移 SQL。批处理通过 GORM slice `Create`、主键 keyset 条件和现有索引实现。

## 测试用例设计（自然语言描述）

> 先于实现编写，Implement 阶段据此编码。
> UT 用例走 `go test`，IT 用例走 Python 集成测试（`test/scripts/`）。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 旧显式 ID 模式保持有效 | `instance_ids=[1]`，不传 `select_all` | 通过模式校验，行为与现有接口一致 | P0 |
| 2 | 两种目标模式同时使用 | `select_all=true, instance_ids=[1]` | HTTP 400，不创建 task | P0 |
| 3 | 两种目标模式都未使用 | `select_all=false, instance_ids=[]` | HTTP 400，不创建 task | P0 |
| 4 | 全选空状态展开稳定全集 | `select_all=true, statuses=[]` | 包含 `installed`，不包含任何过渡状态 | P0 |
| 5 | 多状态去重 | `statuses=["failed","uninstalled","failed"]` | 规范化为两个状态，不重复选实例 | P1 |
| 6 | 非法状态 | `statuses=["typo"]` | HTTP 400，返回 i18n 参数错误 | P0 |
| 7 | 技能/插件显式过渡状态 | `statuses=["installing"]` 或 `["uninstalling"]` | HTTP 400 | P0 |
| 8 | MCP 显式过渡状态 | `statuses=["installing"]` | HTTP 400 | P0 |
| 9 | 分组并集且不重复 | 用户同时属于组 1/2，`group_ids=[1,2]` | 实例只产生一条 record | P0 |
| 10 | 未分组筛选 | 未分组用户实例，`group_ids=[0]` | 命中；已分组用户实例不命中 | P0 |
| 11 | 未分组与正常组混合 | `group_ids=[0,1]` | 命中未分组或组 1 用户实例，取并集且无重复 | P0 |
| 12 | 目标版本决定状态 | 实例已装 v2，资源最新 v3，请求目标 v2，筛选 `installed` | 命中；筛选 `outdated` 不命中 | P0 |
| 13 | 不增加运行状态过滤 | 匹配状态/分组但实例非 running | 仍创建 record，保持现有 distribute 行为 | P0 |
| 14 | Agent 能力过滤 | 状态/分组命中但 Agent 不支持该资源 | 不创建该实例 record | P0 |
| 15 | 零目标 | 所有实例均不匹配状态/分组/能力 | HTTP 400，不创建空 task | P0 |
| 16 | 技能单项全选 | 一个匹配、一个状态不匹配、一个组不匹配 | task.total=1，仅匹配实例有 record；响应 `total=1` | P0 |
| 17 | 技能本地 Agent | 本地实例状态/分组命中 | 创建 pending record，不交给 TAT worker | P0 |
| 18 | 多技能独立筛选 | 实例 A 仅 skill-1=failed，实例 B 仅 skill-2=failed | 两个 task 各自只有对应实例，`instance_count` 分别正确 | P0 |
| 19 | 插件全选 | 多状态、多分组目标 | 只为命中且支持插件的实例建 record，响应 `total` 正确 | P0 |
| 20 | 插件显式 ID 上限兼容 | 501 个显式 ID | 仍返回 HTTP 400 | P0 |
| 21 | MCP 全选与 installation | 两个命中实例 | task/records 分批创建，两个 installation 指向新 task 并进入 installing | P0 |
| 22 | MCP 全选汇总响应 | `select_all=true` | HTTP 202，含 `task_id/total`，不含完整 `per_instance` | P0 |
| 23 | MCP 显式 ID 响应兼容 | `instance_ids=[1]` | HTTP 202，继续返回 `per_instance/warnings` | P0 |
| 24 | MCP 显式 ID 上限兼容 | 501 个显式 ID | 仍返回 HTTP 400 | P0 |
| 25 | keyset 跨批次无遗漏/重复 | 目标数为 `2*batchSize+1`，含重复组命中 | 每个实例恰好一条 record，task.total 等于唯一实例数 | P0 |
| 26 | 批次准备失败 | 中间批写入注入错误 | 不启动异步脚本；task/records 被清理或终结，不留下永久 running task | P1 |
| 27 | 异步批次读取失败 | record 分页查询注入错误 | 任务收敛为 completed，未处理 pending 记录转 failed，锁被释放 | P1 |
| 28 | 旧技能/插件/MCP 显式 ID 核心用例 | 运行现有测试集合 | 原有 task、record、脚本参数和终态断言全部通过 | P0 |
| 29 | 异步 goroutine panic 收敛 | task/record 执行路径注入 panic | 记录 stack；record/task 收敛为 failed/completed；semaphore、WaitGroup 和锁正常释放 | P0 |
| 30 | 技能卸载全选状态白名单 | 空状态或 `installed/outdated/upgrade_failed/uninstall_failed/uninstall_failed_old` | 通过；`uninstalled/failed/installing/uninstalling` 返回 400 | P0 |
| 31 | 企业技能卸载全选 | `statuses=["installed"]` + 用户组 | 只为组内已安装实例创建 uninstall record | P0 |
| 32 | 公共技能批量卸载全选 | `skills[]` 含 public 技能，筛选 installed | 公共技能使用无目标版本状态查询，返回正确 instance_count 和 uninstall task | P0 |
| 33 | 插件卸载全选状态白名单 | 空状态或 `installed/outdated/upgrade_failed/uninstall_failed/uninstall_failed_old` | 通过；`uninstalled/failed/installing/uninstalling` 返回 400 | P0 |
| 34 | 插件按状态+分组卸载全选 | 组内含 installed、uninstall_failed、failed、uninstalled，组外含 installed | 只为组内可卸载状态创建 uninstall record，响应 total 正确且不增加 distribute_count | P0 |
| 35 | search 对齐实例列表 | `select_all=true,search="needle"`，分别由实例名、云实例 ID、创建人用户名命中 | 三种字段均产生 record，其他实例不命中 | P0 |
| 36 | 显式 ID 携带 search | `instance_ids=[1],search="needle"` | HTTP 400，避免筛选参数被静默忽略 | P0 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 技能按状态+分组全选 | 隔离组 G 内一个运行实例；`select_all=true,statuses=["uninstalled"],group_ids=[G]` | 返回 task_id/total=1，实例技能状态离开 uninstalled 并最终收敛 | P0 |
| 2 | 插件按状态+分组全选 | 同一隔离实例和组，插件目标状态 `uninstalled` | 返回 task_id/total=1，插件安装任务完成 | P0 |
| 3 | MCP 按状态+分组全选 | 同一隔离实例和组，MCP 目标状态 `uninstalled` | 返回 202、task_id/total=1，不展开 per_instance，MCP 安装任务完成 | P0 |
| 4 | 分组隔离 | 另一个不属于 G 的实例同样为 uninstalled | 三类全选任务均不包含该实例 | P0 |
| 5 | 已安装状态重复下发 | 首次任务收敛后以 `statuses=["installed"]` 再次全选 | 仍命中同组实例并创建新任务，验证“全部状态包含 installed” | P1 |
| 6 | 新参数覆盖 | 三个请求均实际传入 `select_all/statuses/group_ids` 并生成 OpenAPI 增量报告 | `new_params` 无未覆盖项 | P0 |
| 7 | 显式 ID 冒烟兼容 | 对其中一个资源继续使用旧 `instance_ids` 请求 | 旧调用成功且响应结构未退化 | P0 |
| 8 | search 参数覆盖 | 三类首次下发均传 `select_all=true,statuses=["uninstalled"],search=<CVM实例ID>` | 只命中测试实例并创建 task，OpenAPI 参数覆盖可识别 | P0 |

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 全选命中数很大，单次查询/INSERT/内存切片失控 | 高 | 高 | 统一 200 条 keyset 批次；禁止累计无界目标/record 切片 |
| 2 | 全选总目标无上限导致 goroutine-per-target | 高 | 高 | 每批最多 200 条，启动 goroutine 前获取 semaphore，等待当前批完成后再读取下一批 |
| 3 | 多组成员 JOIN 扇出造成重复下发 | 中 | 高 | 用户组半连接 + 批内 ID 去重 + 跨批次唯一性测试 |
| 4 | 状态按最新版本而非目标版本计算，造成误选 | 中 | 高 | 先解析目标资源，再构建状态 CASE；v2/v3 交叉测试 |
| 5 | 准备阶段中途失败留下 running task 或 pending records | 中 | 高 | 执行器仅在全部批次准备成功后启动；失败路径集中清理/终结并测试 |
| 6 | MCP 分批 upsert installation 后失败导致部分状态污染 | 中 | 高 | task/record/installation 每批保持一致，失败时按 last_task_id 收敛；重点覆盖回滚/终结 |
| 7 | 全选不限制 running，非运行实例产生失败 | 高 | 中 | 这是已确认的现有 handler 语义；逐实例记录错误，不把可预见失败隐藏 |
| 8 | 公共技能本地快照 JOIN 或多 scope 产生重复实例 | 中 | 中 | 目标遍历按实例 ID 去重，跨批次用单调主键游标 |
| 9 | 新路径改变显式 ID 兼容行为 | 低 | 高 | 模式早分流；保留 500 上限和 MCP 明细响应；运行现有回归用例 |
| 10 | 异步 context 丢失租户/语言信息 | 低 | 高 | 启动前使用 `DetachContext` + `i18n.WithPrinter`，后台 DB 始终使用传入 ctx |

