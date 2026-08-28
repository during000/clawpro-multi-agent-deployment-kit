# Plan：资产管理同步模式与版本记录

> 需求:TAPD #1020422209135817508【clawpro】新增项目与资产管理页
> 负责范围:§5.3(资产编辑/保存生成版本 + 同步模式)+ §6(版本管理:触发、展示、下发)
> 协作分支:`feature/local-agent2-final`(现有实现负责资产绑定/项目/分组 CRUD 与本地 agent diff)
> 分工模式:**本模块提供函数(seam),由 调用方在已有接口里调用**,不修改其接口主体
> 关联文档:`docs/project-asset-api.md`(前端对接契约已定稿)

---

## 0. 代码现状(基于 `feature/local-agent2-final` @ ea009d4f)

| 项 | 现状 | 归属 |
|---|---|---|
| `POST /admin/assets/save` | 已实现;存绑定 + 生成版本记录(`manual_save`) + 按 `sync_mode` 触发下发;请求新增必填 `sync_mode`(项目强制 continuous,分组可取 initial_only/continuous 并写入 `user_groups.sync_mode`) | 现有实现 |
| `GET /admin/assets/detail` | 返回实时 `current_version`(查 MAX(version)，未保存为 0) + `target.sync_mode`(项目固定 continuous,分组取 user_groups.sync_mode) | 现有实现 |
| `model.Project` / `model.UserGroup` | **均无 `SyncMode` 字段** | 待本模块加 |
| `AssetVersionRecord` 表 / `RecordAssetVersion` / `PublishAssetVersion` | **全局不存在** | 本模块新建 |
| `HandleDistributeSkill` / `HandleDistributeRule` | 已存在(admin_skills.go:2096 / admin_rules.go:1235),基于 `createSkillTaskAndRecords` + `runSkillDistributeTask` 异步任务 + reporter/sync 拉取 ack | 调用方(本模块复用其下发能力) |
| sync_mode 常量 | 未定义 | 本模块定义 |

**结论**:现有代码已搭好资产绑定 + 前端契约骨架,本模块补齐「同步模式字段 + 版本记录生成 + 按同步模式下发 + 版本列表/详情查询」。

---

## 1. 同步模式字段(需求点 1)

### 1.1 数据模型

`model/project.go` 的 `Project` struct 新增:
```go
// SyncMode 资产同步模式;项目固定为 continuous(需求 §5.3.1 + 文档 §10.8:项目只允许 continuous)
SyncMode string `gorm:"size:32;not null;default:'continuous'" json:"sync_mode"`
```

`model/user_group.go` 的 `UserGroup` struct 新增:
```go
// SyncMode 资产同步模式;分组可为 initial_only(仅新增实例初始配置)或 continuous(始终同步)
// 默认值 initial_only:UserGroup 已有存量数据,存量分组统一按初始配置处理
SyncMode string `gorm:"size:32;not null;default:'initial_only'" json:"sync_mode"`
```

> 分组的 `sync_mode` 由 `POST /admin/assets/save` 的 `sync_mode` 字段指定(必填,见文档 §10.8);建表默认 `initial_only`,存量分组自动为初始配置语义。两种值(initial_only / continuous)均合法。

### 1.2 同步模式常量(建议放 `model/constant.go` 或 `model/project.go`)

```go
const (
    // AssetSyncModeInitialOnly 仅作为新增实例初始配置,存量实例不受影响
    AssetSyncModeInitialOnly = "initial_only"
    // AssetSyncModeContinuous 所有实例始终同步更新(仅追加,不卸载)
    AssetSyncModeContinuous = "continuous"
)
```

- 项目:`POST /admin/assets/save` 的 `sync_mode` 校验为 `continuous`(文档 §10.8:项目只允许 continuous);非 continuous 返回参数错误。
- 分组:两者均允许;新建分组默认 `initial_only`(存量数据无需迁移),保存资产时可由 `sync_mode` 字段改为 `continuous`。

### 1.3 红线

- 改 GORM model → 同步 `sql/init.sql` + 增量 migration SQL + `MigrateFromSQLite`
- 字段默认值:项目 `continuous`、分组 `initial_only`(UserGroup 已有存量数据,默认值保证存量分组为初始配置语义,无需补迁移)

---

## 2. 同步模式修改（合并进 `/admin/assets/save`，无独立接口）

> **决策**：不再单独提供同步模式修改接口。修改同步模式统一通过 `/admin/assets/save` 完成——save 请求参数已含 `sync_mode`（见 §1.3），前端带新的 `sync_mode` 调一次 save 即可改模式，也可同时改资产列表。

`RecordAssetSave`（在 save 事务内调用）对同步模式的处理：
- 读目标**旧** `SyncMode`（事务内 `tx` 查 `Project` / `UserGroup`）
- 若本次 save 的 `sync_mode` 相对旧值发生 **`initial_only → continuous` 跳变**：对该目标**所有存量实例**全量安装当前资产（调 `InstallAssetToTargets`，full=true）；其余情况（本就是 continuous、或变为 initial_only）仅记录不下发
- 同步模式变更与资产列表变更在同一 save 请求里，统一记一条版本记录（trigger_reason=manual_save）；跳变触发的全量下发是这条记录附带的动作

---

## 3. `POST /admin/assets/save` 事务内调用本模块函数（需求点 3）

> **边界约束**：`/admin/assets/save` 的接口形状**基本保持 现有契约（docs/project-asset-api.md §10.8）**，本模块在 save 上**仅新增 1 个请求参数 `sync_mode`（必填，见 §1.3）**——分组用它指定同步模式、项目传 `continuous`；其余现有请求参数与响应参数均不新增、不修改、不删除。
> `RecordAssetSave` 只是 调用方在 save 内部调用的一个内部函数，消费他已解析好的 `req` 字段（含新增的 `req.SyncMode`），不改变 save 对其他参数的处理方式。
### 3.1 改动位置
`HandleAdminAssetSave` 已有 `db.Transaction(func(tx *gorm.DB) error { ... })`,当前只做了 `ReplaceProjectConfigBindings` / `replaceGroupAssetBindings`。

**本模块函数应放进同一个事务 lambda 内调用**(与绑定写入原子提交:绑定成功才记版本,任一步失败整体回滚,不留半截记录):

```go
err = db.Transaction(func(tx *gorm.DB) error {
    // 1. 写绑定(现有逻辑)
    switch target.typeName {
    case assetTargetProject:
        if err := model.ReplaceProjectConfigBindings(tx, target.id, model.AssetBindingTypeSkill, skills); err != nil { return err }
        if err := model.ReplaceProjectConfigBindings(tx, target.id, model.AssetBindingTypeRule, rules); err != nil { return err }
    case assetTargetGroup:
        // ... replaceGroupAssetBindings ...
    }
    // 2. 本模块:在同一事务内生成版本记录 + 按需下发(tx 传入)
    if err := assetversion.RecordAssetSave(tx, assetversion.SaveInput{
        TargetType:   req.TargetType,
        TargetID:     req.TargetID,
        SyncMode:     req.SyncMode,
        OldSyncMode:  oldSyncMode,   // 调用方查出传入(本项目/分组的旧 SyncMode)
        Assets:       req.Assets,    // 新绑定
        OldAssets:    oldAssets,     // 调用方查出传入(旧绑定,用于 diff)
        Operator:     currentUser,
    }); err != nil {
        return err
    }
    return nil
})
```

> 函数签名为 `RecordAssetSave(tx *gorm.DB, in SaveInput) error`--**接收事务句柄**,不自己开事务。

### 3.1 自动触发链路同样收 `tx`（与手动链路一致）

资产库变更触发的版本记录（`publishAssetVersionForChange` / `publishScopeRemoval` / `PublishAssetVersion`）原各自 `model.DB(ctx)` 重新取连接、自管事务,与上层 handler 主事务脱节。现统一改为**接收 `tx *gorm.DB` 参数**,由上层 handler 在已有事务内调用并传入同一个 `tx`:

- `PublishAssetVersion(tx *gorm.DB, in PublishInput)`:**去掉内部 `tx := model.DB(ctx)` 与 `.Transaction` 包裹**,直接用传入 tx(内部 `bumpTargetVersion` / `Create` / `maybeInstall` 均在同一 tx 内)。
- `publishAssetVersionForChange(tx *gorm.DB, ...)` / `publishScopeRemoval(tx *gorm.DB, ...)`:内部 `ListAssetBindingTargetsWithDB(tx, ...)` + 查 `UserGroup.sync_mode` 均用传入 tx;再传 tx 给 `PublishAssetVersion`。
- 调用方(8 处,见 `admin_skills.go` / `admin_rules.go` 的 Create/Update/Delete handler)把 publish 调用**从主事务 `Commit()` 之后移到 `Commit()` 之前(或 `Transaction` 闭包 `return nil` 之前)**,传已有的 `tx`。

> 收益:「资产库主数据变更(创建/更新/删除 skill 或 rule)→ 受影响目标的版本记录 → 按 sync_mode 下发」整条链路同一事务原子提交;任一环节失败整体回滚,不留半截记录。
> 边界:`maybeInstall` 内部 `dispatchInstallFn` 是 DB 操作(非网络 RPC),同事务内执行为预期行为。

### 3.2 `RecordAssetSave` 职责(本模块实现,事务内执行)
1. **生成版本**:版本号是 `asset_versions` 表的衍生值,**不冗余存储在目标表(projects/user_groups)上**。每次生成时查该目标当前 `MAX(version)`(无记录为 0) `+1` 作为本次 `AssetVersionRecord.Version` 写入(trigger_type=manual, trigger_reason=manual_save, operator=当前登录用户)。
   - **不维护目标表 current_version 列**:避免与版本记录表双写一致性问题;查编辑历史表最新记录的 version 即当前版本。读取与记录插入在同一事务内。
   - **不使用乐观锁**:不做 expected/current 比对(简化处理)。save 接口的 `expected_version` 入参与 `version_conflict` 返回由 现有实现 按 §10.8 契约处理,本函数不介入、不修改 save 参数。
   - **首建即 v1**:`/admin/assets/save` 是「全量替换」语义,**新建与修改走同一 `HandleAdminAssetSave` 路径、同一 `RecordAssetSave` 调用**,不区分场景。目标此前无任何资产绑定时(首次保存),`MAX(version)` 为 0 → 本次 `AssetVersionRecord.Version=1`(即 v1),`Changes.Added` 为首次填入的全部资产、`Removed`/`Updated` 为空。若首建 `sync_mode=continuous` 则按 §3.2 下发规则对 added 项下发,`initial_only` 只记录不下发。
2. **计算 diff**:对比「旧绑定」(由调用方通过 `SaveInput.OldAssets` 传入,**本函数不自查**)与「新绑定」(入参 `SaveInput.Assets`),产出 `added/removed/updated`(updated 比对版本号变化)。
3. **写 changes 明细**:`AssetVersionRecord.ChangesJSON` = `AssetChanges{Added, Removed, Updated, SyncMode}`。
4. **按同步模式决定是否下发**(见 §4):
   - 旧 `SyncMode` 由调用方通过 `SaveInput.OldSyncMode` 传入(本函数不自查);与入参 `SyncMode` 比对:
     - **`initial_only → continuous` 跳变**:调 `InstallAssetToTargets`(full=true)对该目标**所有存量实例**全量安装当前资产(实例列表由调用方传入,见 §4)
     - 入参 `sync_mode = continuous` 且未跳变(本就是 continuous):仅对 `changes.Added` + `changes.Updated` 的项下发(同 save 普通编辑)
     - 入参 `sync_mode = initial_only`:仅记录,不下发
   - 下发动作(`InstallAssetToTargets`)在 `RecordAssetSave` 所在事务内同步调用,版本记录与下发 task/record 同事务提交;下发任务的实际执行由 现有的 `runXxxDistributeTask` 内部异步完成(非本模块职责)

---

## 4. 安装资产(批量下发,需求点 4)

### 4.1 复用底层任务创建函数(非 HTTP handler)
`HandleDistributeSkill` / `HandleDistributeRule` 是**单技能/单规范**的 HTTP handler(每次处理一个 slug+version)。但一个项目/分组下会挂**多个 skill + 多个 rule**,需要批量。

**复用策略**:不调 HTTP handler,直接复用 现有的底层任务创建函数,循环批量:
- 技能:`createSkillTaskAndRecords(ctx, item, TaskTypeDistribute, operatorID, validIDs, infoMap, batchID, now)`(来自 `HandleDistributeSkill` 内部)
- 规范:`createRuleTaskAndRecords(...)`(对应 `HandleDistributeRule`)
- 任务异步执行:复用 `runSkillDistributeTask` / `runRuleDistributeTask`(后台跑,本地 agent 实例走 pending→reporter/sync 拉取 ack)

> 即:本模块新增 `InstallAssetToTargets`,内部 `for` 循环对每个 skill/rule 调上述底层函数,**复用其任务创建 + 异步执行能力**,避免重复造轮子,也不破坏他现有接口。

### 4.2 本模块函数:`InstallAssetToTargets`(通用,不按类型拆分)
```go
// 对目标(project/group)下的实例安装资产;full=true 时全量安装当前资产(sync_mode 跳变场景),
// full=false 时仅安装 changes 中的 added/updated 项(save 场景)
// 内部按 AssetType 分流到 createSkillTaskAndRecords / createRuleTaskAndRecords,不拆两个函数
// 实例列表 / 全量资产清单均由调用方查询后传入,本函数不自查任何数据
func InstallAssetToTargets(ctx context.Context, tx *gorm.DB, targetType string, targetID uint, full bool, changes AssetChanges, instanceIDs []uint, fullAssets AssetChanges) error
```
- `instanceIDs`:调用方用 `model.ListLocalAgentInstancesByScope(ctx, scope, targetID)` 查好传入(本函数不自查)
- `fullAssets`:`full=true` 时调用方用 `model.ListAssetCatalogByX` 查好的**全部**当前绑定资产(skill+rule)传入;full=false 时可为空(只用 changes)
逻辑:
- 遍历待下发项(每项含 `AssetType`),**按 `AssetType` 分流**:skill → `createSkillTaskAndRecords` + `runSkillDistributeTask`;rule → `createRuleTaskAndRecords` + `runRuleDistributeTask`
- `full=true`:对 `instanceIDs` 下发 `fullAssets` 全部 skill + rule
- `full=false`:仅对 `changes.Added` + `changes.Updated` 的项调下发(removed 项不动,仅追加不卸载)
- **同步插入 task/record 数据即可**(下发动作由 现有的 `runXxxDistributeTask` 内部异步执行),本函数不需要自己再套 `DetachContext` 异步层

### 4.3 实例列表由调用方传入(本函数不自查)

`InstallAssetToTargets` 不再自查实例,`instanceIDs` 由调用方在调用前查好传入。调用方查询方式(供 现有实现 参考,非本模块职责):

- **project**:`model.ListLocalAgentInstancesByScope(ctx, model.LocalAgentScopeWorkspace, targetID)`(scope=`workspace`)
- **group**:`model.ListLocalAgentInstancesByScope(ctx, model.LocalAgentScopeUser, targetID)`(scope=`user`,`LocalAgentScopeUser` 在代码里即表示分组资源,见 `model/project.go`)

> 本模块函数**只消费传入的 `instanceIDs`**,不调用任何 现有实现 查询函数,不查 `LocalAgentScopeBinding`。这避免调用方上下文已查过的数据被我们重复查询一次。

---

## 5. 版本记录列表接口(需求点 5)

文档 §4.4 `GET /admin/assets/versions` 已定稿。**不单独做 `version-detail`**--`changes` 明细直接包含在 `versions` 列表的每一项里返回。

### 5.1 本模块实现
- `HandleAdminAssetVersions` → `GET /admin/assets/versions`（**分页接口**）

**请求字段（query string）**:

| 参数 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `target_type` | string | 是 | `project` / `group` |
| `target_id` | uint | 是 | 项目 ID 或分组 ID |
| `page` | int | 否 | 页码，从 1 开始，默认 1 |
| `page_size` | int | 否 | 每页条数，默认 20，最大 100 |

**响应字段（JSON body）**:

| 字段 | 类型 | 说明 |
|---|---|---|
| `total` | int | 符合条件的总记录数 |
| `page` | int | 当前页码 |
| `page_size` | int | 每页条数 |
| `data` | array | 版本记录列表，倒序（最新版本在前） |
| `data[].record_id` | uint | 记录 ID |
| `data[].version` | int | 版本号（自增序列，v1…vN） |
| `data[].trigger_type` | string | `manual` / `system` |
| `data[].trigger_reason` | string | `manual_save` / `asset_version_published` / `asset_deleted` / `asset_scope_changed` |
| `data[].operator` | object | 操作人：`type`(`admin`手动/`system`自动) + `id`(管理员ID, system为0) + `name`(管理员姓名, system为空) |
| `data[].segments` | array | 变更分段，见下（手动/自动统一结构） |
| `data[].segments[].type` | string | 分段类型（见下各场景） |
| `data[].segments[].items` | array | 该段变更项列表；成员字段按场景不同（见下） |
| `data[].segments[].items[].name` | string | `sync_mode` 段的 `items` 为单元素数组，`name` 承载新模式值（如 `continuous`） |
| `data[].created_at` | string | ISO8601 时间 |

> **不返回 `summary` / `changes` 字段**：后端只返回结构化关键信息，文案拼接由前端完成。`ChangesJSON` 仍落库（source of truth、审计、计算 segments 用），但不进响应。
>
> **`segments` 实现要点**：
> - `items[].name`：**响应时按 `asset_type`+`slug` 实时查 `skills`/`enterprise_rules` 表返回真实名称**（`ChangesJSON` 只存 slug，不存名称，避免资产改名导致历史失真）。
> - `sync_mode` 段：**仅当本次 `sync_mode` 相对旧值发生变化时出现**（单独改模式、或改资产的同时改模式都会带此段）；`changes.SyncMode` 由 `RecordAssetSave` 在 `in.SyncMode != in.OldSyncMode` 时填充。`diffAssets` 本身只算资产 diff，不处理 sync_mode。

**`segments[].items` 成员字段（按场景）**:

| 场景 | segment.type | items 成员字段 |
|---|---|---|
| 手动-新增资产(a) | `added` | `asset_type`, `name` |
| 手动-删除资产(b) | `removed` | `asset_type`, `name` |
| 手动-改同步模式(c) | `sync_mode` | `items` 为单元素数组，`name` 承载新模式（如 `continuous`） |
| 自动-版本更新(d1) | `version_published` | `asset_type`, `name`, `old_version`, `new_version` |
| 自动-工具库删除(d2) | `deleted` | `asset_type`, `name` |
| 自动-应用范围变更(d3) | `scope_changed` | `asset_type`, `name` |

- **前端渲染**：遍历 `segments`，按 `type` 套对应文案模板，从 `items` 取字段拼；`sync_mode` 段的 `items` 为单元素数组、`name` 承载模式值。不展示空段（`items` 为空且无 name 的段可跳过）

### 5.1.1 响应示例

**示例 A — 手动编辑（a/b/c 三段，items 为列表）**:

```json
{
  "total": 5,
  "page": 1,
  "page_size": 20,
  "data": [
    {
      "record_id": 102,
      "version": 5,
      "trigger_type": "manual",
      "trigger_reason": "manual_save",
      "operator": {"type": "admin", "id": 8812, "name": "张三"},
      "segments": [
        {"type": "added", "items": [
          {"asset_type": "skill", "name": "API 文档生成器"},
          {"asset_type": "rule", "name": "代码审查工具"},
          {"asset_type": "skill", "name": "会议纪要生成器"}
        ]},
        {"type": "removed", "items": []},
        {"type": "sync_mode", "items": [{"name": "continuous"}]}
      ],
      "created_at": "2026-07-17 19:20:00"
    }
  ]
}
```

**示例 B — 自动同步（d1 版本更新 / d3 范围变更，结构化字段，无 summary）**:

```json
{
  "total": 5,
  "page": 1,
  "page_size": 20,
  "data": [
    {
      "record_id": 101,
      "version": 4,
      "trigger_type": "system",
      "trigger_reason": "asset_version_published",
      "operator": {"type": "system", "id": 0, "name": ""},
      "segments": [
        {"type": "version_published", "items": [
          {"asset_type": "skill", "name": "代码审查工具", "old_version": "v2.0.0", "new_version": "v2.1.0"}
        ]}
      ],
      "created_at": "2026-07-16 19:20:00"
    },
    {
      "record_id": 100,
      "version": 3,
      "trigger_type": "system",
      "trigger_reason": "asset_scope_changed",
      "operator": {"type": "system", "id": 0, "name": ""},
      "segments": [
        {"type": "scope_changed", "items": [
          {"asset_type": "skill", "name": "文档总结助手"}
        ]}
      ],
      "created_at": "2026-07-15 17:30:00"
    },
    {
      "record_id": 99,
      "version": 2,
      "trigger_type": "system",
      "trigger_reason": "asset_deleted",
      "operator": {"type": "system", "id": 0, "name": ""},
      "segments": [
        {"type": "deleted", "items": [
          {"asset_type": "skill", "name": "日志分析器"}
        ]}
      ],
      "created_at": "2026-07-14 23:10:00"
    }
  ]
}
```

### 5.2 数据来源
读 `AssetVersionRecord` 表(按 target_type+target_id 过滤,版本倒序),`ChangesJSON` 反序列化为 `AssetChanges` 一并返回。

### 5.3 路由 + 审计
```go
mux.HandleFunc("/admin/assets/versions", controller.HandleAdminAssetVersions)
```
- 查询接口,requireAdmin;无写操作,不审计

---

## 6. 给 `HandleCreateSkill`/`HandleCreateRule` 的函数(需求点 6,对应 §6.1 自动类型)

### 6.1 触发场景(文档 §5.1 规则 6 + §6.1)
- 工具库技能/规范**发布新版本** → 自动 `asset_version_published`
- 工具库技能/规范**被删除** → 自动 `asset_deleted`
- 工具库技能/规范**修改应用范围**导致移出某项目/分组 → 自动 `asset_scope_changed`

> 注意:上述 4 种自动触发**都会记录版本历史**;但是否触发下发,只有一种情况会——见 §6.2 行为 4。

### 6.2 本模块提供函数(通用,按 AssetType 参数化)
```go
// 工具库技能/规范版本更新(含发布新版本、删除、范围变更)时由 现有实现 调用
// AssetType 区分 skill / rule,内部按类型分流,不拆两个函数
func PublishAssetVersion(ctx context.Context, in PublishInput) (uint, error)
```
`PublishInput`:
```go
type PublishInput struct {
    AssetType   string   // skill | rule
    Slug        string
    FromVersion string   // 空=新增;删除/范围变更也可能无 from
    ToVersion   string   // 空=删除
    TriggerReason string // asset_version_published | asset_deleted | asset_scope_changed
    // 受影响的目标(必填):由调用方用 model.ListAssetBindingTargets 查好传入;
    // 每项 AssetTarget 自带该目标的 SyncMode(调用方查出),本函数不自查
    AffectedTargets []AssetTarget
}
```

行为:
1. 遍历调用方传入的 `AffectedTargets`(已是完整受影响目标列表,本函数不反查)
2. 对每个目标:版本号查该目标 `asset_versions` 表 `MAX(version)+1`(无记录为 0 → 本次从 1 起),取该值作为 `AssetVersionRecord.Version` 写入(trigger_type=system, trigger_reason=入参, operator=系统);目标的 `SyncMode` 直接用 `AssetTarget.SyncMode`(调用方已查出传入)
3. 计算该目标的 changes(updated: from→to / removed / added)
4. **是否下发(关键判定:按"本次变更是否有新增/更新项" + 目标 SyncMode)**:
   - **下发前提**:仅当该目标 `SyncMode=continuous` **且** 本次变更存在 `added`/`updated` 项(即需要给存量实例**新增或更新** skill/rule)时才触发下发;调用 §4 的 `InstallAssetToTargets`(实例列表由调用方传入)
   - **不下发**:`initial_only` 目标一律只记录不下发;或本次变更仅有 `removed` 项(删除 / 应用范围缩小移出,无新增可装)——符合“仅追加不卸载”原则,无需下发任务
   - **所有自动同步场景(版本更新 / 删除 / 范围变更)都生成版本历史**(record 必须有);下发任务按需触发
   - 各 trigger_reason 说明:
     - `asset_version_published`:技能/规范发新版。若该目标 continuous 且本次有 added/updated → 下发更新
     - `asset_deleted` / `asset_scope_changed`:删除 / 范围缩小。通常仅 removed,无新增 → 只记录不下发
   - 所有判定所需数据(受影响目标、各目标 SyncMode、实例列表)均由调用方在调用前查询并传入,**本函数不调用任何 现有实现 查询函数**
5. 返回 record_id

### 6.3 调用胶水代码(本模块写,负责查数据并传入)

> **调用方代码由本模块编写**:即在 现有的 handler(`HandleCreateSkill` 等)里插入调用我们函数的胶水代码,这段胶水代码也归本模块负责。胶水代码里需要调用 现有提供的查询函数拿到数据,再塞进我们的函数参数。

本模块函数本身**不调用任何 现有实现 查询函数**(保持纯消费、易测);但**胶水代码(本模块写)需要调用**以下 现有 函数:

- 受影响目标:胶水代码调 `model.ListAssetBindingTargets(ctx, assetType, slug)` 查 → 填入 `PublishInput.AffectedTargets`(每项自带 `SyncMode`)
- 目标实例列表:胶水代码调 `model.ListLocalAgentInstancesByScope(ctx, scope, targetID)` 查 → 传入 `InstallAssetToTargets` 的 `instanceIDs`
- 全量资产(full=true):胶水代码调 `model.ListAssetCatalogByGroup/Project` 查 → 传入 `InstallAssetToTargets` 的 `fullAssets`
- 旧绑定 / 旧 SyncMode(save 场景):胶水代码查出 → 传入 `SaveInput.OldAssets` / `OldSyncMode`

> 查询动作发生在"我们写的胶水代码"里,被调函数内部仍是纯消费。这样职责清晰、易测(测试直接构造入参即可,无需 mock 查询函数)。

### 6.4 调用点(本模块在 现有的 handler 内插入胶水代码)

> 这些调用位置由**本模块**负责在 现有的 handler 里插入胶水代码(见 §6.3)。**自动触发在事务内同步调用 `PublishAssetVersion`,不另起 `go func`**(本需求不需要异步,见 §10 第3条)。

**生成版本记录的 5 类触发场景(全部落 `AssetVersionRecord`,全部事务内同步)**:

| # | 触发类型 | trigger_type | trigger_reason | 调用函数 | 调用点(事务内) |
|---|---|---|---|---|---|
| 1 | 手动保存资产 | manual | `manual_save` | `RecordAssetSave` | 现有 `HandleAdminAssetSave` 事务 lambda 内 |
| 2 | 工具库发布新版本 | system | `asset_version_published` | `PublishAssetVersion` | 本模块胶水代码 → `HandleCreateSkill`(新版本成功) |
| 3 | 工具库资产被删除 | system | `asset_deleted` | `PublishAssetVersion` | 本模块胶水代码 → `HandleDeleteSkill` |
| 4 | 应用范围变更·移出目标 | system | `asset_scope_changed` | `PublishAssetVersion` | 本模块胶水代码 → `HandleUpdateSkill`(范围变更,对比前后,**移出**的目标) |

> skill 与 rule 完全对称:`HandleCreateRule` / `HandleDeleteRule` / `HandleUpdateRule` 与 skill 三处调用点一一对应(同一套胶水代码,按 `AssetType` 参数化)。

**`HandleUpdateSkill`/`HandleUpdateRule` 应用范围变更的分流逻辑**(胶水代码实现):
- 调 `model.ListAssetBindingTargets`(旧 slug) + 新应用范围计算出 **变更前后目标集合**
- 移出的目标 → `AffectedTargets[]` 中 `TriggerReason=asset_scope_changed`
- 两者可同时存在(同一资产范围调整,部分项目移出、部分新增),分别作为独立 record 落库

**下发判定(与触发类型无关,只看目标 SyncMode + 是否有 added/updated 项,见 §6.2 行为4)**:
- `manual_save`:`initial_only→continuous` 跳变→全量下发;continuous 普通编辑→对 added/updated 下发;initial_only→只记录
- `asset_version_published`:目标 continuous 且有 added/updated→下发(added 全量, published 增量)
- `asset_deleted` / `asset_scope_changed`:通常仅 removed→只记录不下发

---

## 7. 数据模型汇总(本模块新建/修改)

### 7.1 新增 `model/asset_version.go`
```go
type AssetVersionRecord struct {
    ID           uint      `gorm:"primaryKey;autoIncrement" json:"record_id"`
    TargetType   string    `gorm:"size:32;not null;index:idx_av_target" json:"target_type"` // group | project
    TargetID     uint      `gorm:"not null;index:idx_av_target" json:"target_id"`
    Version      int       `gorm:"not null" json:"version"` // 目标内自增
    TriggerType  string    `gorm:"size:32;not null" json:"trigger_type"` // manual | system
    TriggerReason string   `gorm:"size:32;not null" json:"trigger_reason"` // manual_save|asset_version_published|asset_deleted|asset_scope_changed|reconcile
    OperatorID   uint      `gorm:"not null;default:0" json:"operator_id"`
    OperatorName string    `gorm:"size:64; not null;default:''" json:"operator_name"`
    OperatorType string    `gorm:"size:16; not null;default:'user'" json:"operator_type"`
    ChangesJSON  string    `gorm:"type:text; not null" json:"-"` // AssetChanges{Added,Removed,Updated,SyncMode}，落库不进响应
    CreatedAt    time.Time `json:"created_at"`
}
```
- 注册 `AutoMigrate` + `MigrateFromSQLite`(`model/db.go` / `model/migrate.go`)
- **不含 `sync_status` / `batch_id` / `total/success/failed/skipped` / `completed_at` 字段**:下发状态由 现有的 task/record 表承载,本模块只管"版本记录",不重复记录下发进度
- `ChangesJSON` 反序列化为 `AssetChanges{Added, Removed, Updated, SyncMode}` 随 `versions` 返回
### 7.2 修改
- `model/project.go`:+`SyncMode`
- `model/user_group.go`:+`SyncMode`
- `sql/init.sql`:+`asset_versions` 表 + `projects.sync_mode` + `user_groups.sync_mode` 列
- 增量 migration SQL(按项目规范命名,如 `sql/0716-asset-version.sql`)

---

## 8. 文件清单(本模块交付)

| 文件 | 动作 | 说明 |
|---|---|---|
| `model/asset_version.go` | 新建 | `AssetVersionRecord` + `AssetChanges` + 常量 |
| `model/project.go` | 改 | +`SyncMode` |
| `model/user_group.go` | 改 | +`SyncMode` |
| `model/db.go` / `model/migrate.go` | 改 | 注册新表 |
| `controller/asset_version.go` | 新建 | `RecordAssetSave` / `PublishAssetVersion` / `InstallAssetToTargets` + 查询 handler（`HandleAdminAssetVersions`） |
| `sql/0716-asset-version.sql` | 新建 | 增量建表 + 加列 |
| `sql/init.sql` | 改 | 同步结构 |
| `main.go` | 改 | 注册 1 个路由（`assets/versions`）；`/admin/assets/save` 由 现有代码已注册，本模块只在其内部调 `RecordAssetSave` |
| `i18n/keys.go` / `i18n/en.go` | 改 | 新增 i18n key(响应文案不硬编码) |
| `docs/API.md` | 改 | 补 1 个接口文档(`assets/versions`) |

> 注意:现有实现 需在他侧 `HandleAdminAssetSave` 调用 `RecordAssetSave`、在 `HandleCreateSkill/Rule/Delete` 调用 `PublishAssetVersion`。本模块不修改这些文件。

---

## 9. 红线自查(项目规范)

1. 改 GORM model → 同步 `sql/init.sql` + 增量 migration + `MigrateFromSQLite` ✅
2. 写接口(`/admin/assets/save` 由 现有实现 注册并审计;本模块不新增写接口) ✅
3. handler 用 `model.DB(r.Context())` 而非 `model.DB` ✅
4. 响应文案用 `i18n.T` / i18n key,不硬编码中文 ✅
5. 异步 goroutine 用 `hcommon.DetachContext(r.Context())` ✅
6. 新增 i18n.Key 在 `en.go` 加英文翻译 ✅
7. 改 API → 更新 `docs/API.md` + 集成测试覆盖 ✅
8. 多租户:所有查询带租户隔离(复用 现有的 target 解析,资产绑定已隔离)✅

---

## 10. 待确认项（统一汇总）

> 所有未决事项集中于此，各节不再散落待确认描述。

1. **函数最终签名（本模块决定，现有 确认调用便利性）**：`RecordAssetSave` / `PublishAssetVersion` / `InstallAssetToTargets` 的入参/返回值由本模块提案定稿（本模块是 seam 提供方，现有实现 是调用方）；需与 现有 对齐的仅是调用便利性细节（如 target_type 取值、AssetTarget 字段是否他查出来直接能填、错误返回形式），决策权在本模块
2. **`RecordAssetSave` 插入点**：在 `HandleAdminAssetSave` 的 `db.Transaction` lambda 内、绑定写入之后（已确认，见 §3.1）
3. **自动触发调用时机（已确认）**：`PublishAssetVersion` 在 `HandleCreateSkill` 等 handler 的**事务内同步调用**，不用 `go func`（本需求不需要异步）
4. **响应字段规则（已确认）**：后端**不拼 summary / 不返回 changes**，只返回 `segments[].items` 结构化关键信息（手动：added/removed 段 items 含 asset_type+name，sync_mode 段 items 为单元素数组、name 承载模式值；自动：按 trigger_reason 返回 asset_type/name/old_version/new_version 等）。文案由前端拼接。`operator` 返回 `type`+`id`+`name`（system 时 id=0/name=""）。`ChangesJSON` 落库（source of truth + 审计）但不进响应。
5. **save 接口参数**：`/admin/assets/save` 仅新增 `sync_mode` 一个请求参数，其余请求/响应参数保持 §10.8 原样（已确认）
6. **实例查询函数名（已确认）**：现有代码已提供 `model.ListLocalAgentInstancesByScope(ctx, scope, targetID)`（及 `...WithDB` 事务版），scope 用 `LocalAgentScopeWorkspace`(项目) / `LocalAgentScopeUser`(分组)。group 维度**只查本组，不含子孙组**（现有 返回即为最终结果，本模块不扩展）
7. **skill/rule → 目标反向关联函数名（已确认）**：`model.ListAssetBindingTargets(ctx, assetType, slug)`（及 `...WithDB` 版）；查目标资产用 `ListAssetCatalogByGroup/Project`。全部已实现，本模块只调用


## 11. 版本历史 UI 字段映射（基于 UI 设计稿）

UI 展示为一条带「类型徽章 + 操作描述 + 子项明细」的时间线（组织视角，本需求为 project/group 视角，结构通用）。**后端只返回结构化关键信息（`segments[].items`），文案由前端拼接。**

| UI 展示 | 数据来源 | 说明 |
|---|---|---|
| 版本号 v1…vN | `AssetVersionRecord.Version` | 倒序返回 |
| 时间 | `AssetVersionRecord.CreatedAt` | |
| 操作人（文本） | `operator.type` / `operator.name` | `type=admin`→平台管理员(姓名取 `operator.name`) / `type=system`→系统自动同步 |
| 徽章 | `trigger_type` + `trigger_reason` | 见下方徽章映射表 |
| 主描述 / 子项 | 前端按 `segments[].type` + `segments[].items` 字段拼接 | 后端不拼文案 |

**徽章映射**（trigger_type + trigger_reason → UI 文案）：

| trigger_type | trigger_reason | UI 徽章 |
|---|---|---|
| manual | manual_save | 手动编辑 |
| system | asset_version_published | 自动同步·更新 |
| system | asset_deleted | 自动同步·删除 |
| system | asset_scope_changed | 自动同步·超范围调整 |

**前端拼接规则（依据 `segments`）**:

| segment.type | 前端文案模板 | items 取字段 |
|---|---|---|
| `added` | 新增 x 项资产：<逐项列出> | `asset_type`, `name` |
| `removed` | 删除 x 项资产：<逐项列出> | `asset_type`, `name` |
| `sync_mode` | 同步模式修改为「<name>」 | `items[0].name` |
| `version_published` | 企业技能「<name>」版本更新 <old_version> → <new_version> | `asset_type`, `name`, `old_version`, `new_version` |
| `deleted` | 企业技能「<name>」工具库已删除，已自动同步移除 | `asset_type`, `name` |
| `scope_changed` | 企业技能「<name>」应用范围调整，不再命中本组织，已自动同步移除 | `asset_type`, `name` |

> 注：手动场景（a/b/c）对应 `added`/`removed`/`sync_mode` 三段；自动场景（d1/d2/d3）对应 `version_published`/`deleted`/`scope_changed` 单段。所有文案前端拼，后端只返回 `items` 结构化字段。`ChangesJSON` 落库但不进响应（source of truth + 审计）。
