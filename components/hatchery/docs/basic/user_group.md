# 用户组功能技术方案

## 一、背景与目标

为 ClawPro 平台新增用户组管理能力，支持管理员对用户进行分组管理，便于后续按组进行权限控制、资源分配等扩展。

**核心约束：**
- 只有管理员（`role = admin`）可以创建/删除用户组、修改组内成员、查看所有用户组
- 普通用户只能查询**自己所在的用户组**，无法查看整个平台的用户组列表
- 管理员概念与 `identifier` 绑定，`role = admin` 的作用域是当前租户实例，每个租户有独立的管理员体系
- 当前使用 SQLite，`Identifier` 字段作为多租户预留扩展字段，现阶段透明无感，迁移 MySQL 多租户时自动生效
- **单平台最多 1000 个用户组**，超出后创建接口返回 400
- **单用户组最多 10000 名成员**，超出后添加/设置成员接口返回 400

---

## 二、数据模型设计

### 2.1 `user_groups` — 用户组主表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | uint | PK, AUTO_INCREMENT | 主键 |
| `identifier` | string | NOT NULL, DEFAULT '' | 多租户预留字段，SQLite 下为空字符串 |
| `name` | string | NOT NULL | 用户组名称 |
| `description` | string | DEFAULT '' | 描述信息 |
| `created_at` | datetime | | 创建时间（GORM 自动维护） |
| `updated_at` | datetime | | 更新时间（GORM 自动维护） |

**唯一索引：** `(identifier, name)`，同一租户下组名不重复

### 2.2 `user_group_members` — 成员关联表

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | uint | PK, AUTO_INCREMENT | 主键 |
| `identifier` | string | NOT NULL, DEFAULT '' | 多租户预留字段 |
| `user_group_id` | uint | NOT NULL, INDEX | 所属用户组 ID |
| `user_id` | uint | NOT NULL, INDEX | 用户 ID |
| `created_at` | datetime | | 加入时间（GORM 自动维护） |

**唯一索引：** `(identifier, user_group_id, user_id)`，防止重复加入

> `user_group_members` 不使用软删除，移除成员直接物理删除记录。

---

## 三、API 接口设计

接口分为两类：
- **管理员接口**：路由前缀 `/admin/user-groups`，走 `requireAdmin` 中间件 + `WithOpenAPI` + `WithAudit` 装饰器
- **普通用户接口**：路由前缀 `/user-groups`，走普通登录鉴权中间件，只能查询自己所在的用户组

### 3.1 接口列表

#### 管理员接口（`/admin/user-groups`）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/admin/user-groups` | 分页列出所有用户组（含成员数） |
| `POST` | `/admin/user-groups/create` | 创建用户组 |
| `POST` | `/admin/user-groups/update` | 修改用户组信息（如名称、描述） |
| `POST` | `/admin/user-groups/delete` | 删除用户组（级联清除成员） |
| `GET` | `/admin/user-groups/members` | 查询组内成员列表（支持分页） |
| `POST` | `/admin/user-groups/members/set` | 全量替换组内成员 |
| `POST` | `/admin/user-groups/members/add` | 批量添加成员（幂等） |
| `POST` | `/admin/user-groups/members/remove` | 批量移除成员 |
| `GET` | `/admin/user-groups/groups-by-users` | 批量查询多个用户所在的所有用户组 |

> **注：** `GET /admin/users` 接口新增两个过滤参数：
> - `group_ids`：英文逗号分隔的用户组 ID，返回属于这些组中任意一个的用户（OR 语义），如 `group_ids=1,2,3`
> - `ungrouped`：为 `1` 或 `true` 时只返回未加入任何用户组的用户（优先于 `group_ids`）

#### 普通用户接口（`/user-groups`）

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/user-groups/mine` | 查询当前用户所在的所有用户组 |

### 3.2 接口详情

#### GET `/admin/user-groups`
**Query 参数：**
```
page      int    页码，默认 1
page_size int    每页数量，默认 20
```
**响应：**
```json
{
  "ok": true,
  "total": 5,
  "groups": [
    {
      "id": 1,
      "name": "研发组",
      "description": "研发部门成员",
      "member_count": 8,
      "created_at": "2026-04-11T10:00:00Z"
    }
  ]
}
```

#### POST `/admin/user-groups/create`
**请求体：**
```json
{
  "name": "研发组",
  "description": "研发部门成员"
}
```
**响应：**
```json
{
  "ok": true,
  "group": { "id": 1, "name": "研发组", "description": "研发部门成员" }
}
```
**错误：** 组名已存在返回 400；平台用户组数量已达上限（1000 个）返回 400

#### POST `/admin/user-groups/delete`
**请求体：**
```json
{ "id": 1 }
```
**响应：**
```json
{ "ok": true }
```
**说明：** 执行前先调用 `CanDeleteUserGroup` 检查是否允许删除（存在关联资源时返回 400）；通过后在同一事务内先物理删除 `user_group_members`，再物理删除 `user_groups`

#### POST `/admin/user-groups/update`
修改用户组的名称或描述，名称修改时仍需满足同一租户下唯一的约束。

**请求体：**
```json
{
  "id": 1,
  "name": "研发一组",
  "description": "研发部门核心成员"
}
```
**响应：**
```json
{
  "ok": true,
  "group": { "id": 1, "name": "研发一组", "description": "研发部门核心成员" }
}
```
**错误：** 新名称与其他用户组重名返回 400；用户组不存在返回 400

#### GET `/admin/user-groups/members`
**Query 参数：**
```
id        int    用户组 ID（必填）
page      int    页码，默认 1
page_size int    每页数量，默认 20
```
**响应：**
```json
{
  "ok": true,
  "total": 50,
  "members": [
    { "user_id": 2, "username": "alice", "joined_at": "2026-04-11T10:00:00Z" }
  ]
}
```

#### POST `/admin/user-groups/members/set`
全量替换，适合前端"勾选成员列表"场景，事务内先清空再批量插入。

**请求体：**
```json
{
  "id": 1,
  "user_ids": [2, 3, 5]
}
```
**响应：**
```json
{ "ok": true }
```
**说明：** `user_ids` 为空数组时，清空该组所有成员；`user_ids` 长度超过 10000 返回 400

#### POST `/admin/user-groups/members/add`
批量添加，幂等，已存在的成员忽略，不报错。

**请求体：**
```json
{ "id": 1, "user_ids": [2, 3, 4] }
```
**响应：**
```json
{ "ok": true }
```
**错误：** 任一 `user_id` 不存在则整体返回 400；添加后成员总数超过 10000 返回 400

#### POST `/admin/user-groups/members/remove`
批量移除，`user_ids` 中不存在于组内的成员静默忽略。

**请求体：**
```json
{ "id": 1, "user_ids": [3, 4] }
```
**响应：**
```json
{ "ok": true }
```

#### GET `/admin/user-groups/groups-by-users`
批量查询多个用户所在的所有用户组，返回组的基本信息（不含成员列表）。

**Query 参数：**
```
user_ids  string  多个用户 ID，英文逗号分隔，最多 100 个，如 1,2,3（必填）
```

**响应：**
```json
{
  "ok": true,
  "data": {
    "1": [{ "id": 1, "name": "研发组", "description": "研发部门成员" }],
    "2": [],
    "3": [{ "id": 2, "name": "运营组", "description": "" }]
  }
}
```
**说明：** `data` 的 key 为用户 ID 字符串，value 为该用户所属的用户组列表（无所属组时为空数组）；`user_ids` 最多支持 100 个，超出返回 400

---

### 3.3 普通用户接口详情

#### GET `/user-groups/mine`
返回当前登录用户所在的所有用户组，不分页（用户所在组数量通常较少）。

**响应：**
```json
{
  "ok": true,
  "groups": [
    {
      "id": 1,
      "name": "研发组",
      "description": "研发部门成员",
      "member_count": 8,
      "members": [
        { "user_id": 2, "username": "alice" },
        { "user_id": 3, "username": "bob" }
      ]
    },
    {
      "id": 3,
      "name": "项目A组",
      "description": "",
      "member_count": 3,
      "members": [
        { "user_id": 2, "username": "alice" }
      ]
    }
  ]
}
```
**说明：** 返回组的基本信息及完整成员列表（含 `member_count` 和 `members`），成员信息只包含 `user_id`、`username`，不含敏感字段（如密码、token 等）

---

## 四、文件改动清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `model/user_group.go` | **新增** | 定义 `UserGroup`、`UserGroupMember` 模型及 CRUD 辅助函数 |
| `controller/admin_user_groups.go` | **新增** | 管理员 9 个 HTTP Handler |
| `controller/user_groups.go` | **新增** | 普通用户 1 个 HTTP Handler（查询自己所在的组） |
| `controller/admin_users.go` | **修改** | 创建/修改用户接口支持传入 `group_ids`；删除用户时自动解绑用户组成员关系；查询用户列表（`GET /admin/users`）响应中每个用户对象新增 `groups` 字段（含 `id` 和 `name`），批量查询避免 N+1 |
| `model/db.go` | **修改** | SQLite `AutoMigrate` 列表追加 `&UserGroup{}`、`&UserGroupMember{}` |
| `main.go` | **修改** | 注册用户组相关路由 |
| `sql/user-groups.sql` | **新增** | MySQL 模式下的建表迁移脚本（预留） |

---

## 五、内部函数（供其他模块直接调用）

以下两个函数定义在 `model/user_group.go`，供后端其他模块直接调用，**不对外暴露 HTTP 接口**。

### 5.1 `GetUserGroupIDs`

```go
// GetUserGroupIDs 查询用户所属的所有用户组 ID
// 调用方：controller/openclaw_model.go（用户模型列表过滤 + 绑定模型时校验用户所在组）
func GetUserGroupIDs(userID uint) ([]uint, error)
```

- 查询 `user_group_members` 表，返回指定用户所属的全部 `user_group_id` 列表
- 若用户不属于任何组，返回空切片（`[]uint{}`）和 `nil` error
- 若数据库查询失败，返回 `nil` 和具体 error，调用方需处理

### 5.2 `GetGroupsByIDs`

```go
// GetGroupsByIDs 批量查询用户组信息（ID → 名称等基本信息）
// 调用方：controller/admin_models.go（管理端模型列表展示分组名称 + 设置可见范围时校验分组是否存在）
func GetGroupsByIDs(ids []uint) ([]UserGroup, error)
```

- 根据传入的 ID 列表批量查询 `user_groups` 表，返回对应的 `UserGroup` 切片
- 若 `ids` 为空，直接返回空切片和 `nil` error，不发起数据库查询
- 若数据库查询失败，返回 `nil` 和具体 error，调用方需处理
- 返回结果数量可能少于传入 ID 数量（已软删除或不存在的组不会出现在结果中），调用方按需自行校验

### 5.3 `GetGroupsCVMInstanceIDs`

```go
// GetGroupsCVMInstanceIDs 查询多个用户组内所有用户关联的 CVM 实例 ID 列表
// 调用方：controller/admin_users.go（对用户组批量执行 CVM 操作，如停机、销毁等）
func GetGroupsCVMInstanceIDs(groupIDs []uint) ([]string, error)
```

- 若 `groupIDs` 为空，直接返回空切片和 `nil` error，不发起数据库查询
- 先查询 `user_group_members` 表（`WHERE user_group_id IN ?`）获取所有组内的 `user_id`，再查询 `instances` 表获取这些用户关联的所有 CVM 实例 ID
- 只返回 `instance_id != ''` 的记录（已绑定 CVM 的实例）
- 若所有用户组均为空或组内用户均无关联实例，返回空切片和 `nil` error
- 若数据库查询失败，返回 `nil` 和具体 error，调用方需处理
- 返回的实例 ID 列表已去重，多个用户组存在共同成员时不会重复计入
- 参考 `controller/admin_users.go` 中的 `getUserInstanceIDs(userID uint)` 函数，本函数是其用户组维度的批量扩展版本

---

## 六、关键设计决策

### 5.1 删除级联
删除用户组时，先调用 `CanDeleteUserGroup` 进行前置检查，再在同一事务内：
1. 物理删除 `user_group_members` 中该组的所有成员记录
2. 物理删除 `user_groups` 中的该组记录

**删除用户时解绑用户组：**
删除用户（`DELETE /admin/users`）时，在同一事务内额外执行：
- 物理删除 `user_group_members` 中所有 `user_id = 被删用户 ID` 的记录

无需前置检查，直接级联清除，保证用户组成员数据不残留孤儿记录。

`user_groups` 不使用软删除，改为硬删除，避免软删除记录与新建同名用户组产生唯一索引冲突。

**`CanDeleteUserGroup` 预检函数：**
```go
// CanDeleteUserGroup 检查用户组是否允许被删除。
// 当用户组存在关联资源（如绑定了模型可见范围等）时，应返回 false 阻止删除。
func CanDeleteUserGroup(groupID uint) (bool, error) {
    // 检查是否有模型可见性配置引用该用户组
    used, err := IsGroupUsedByModelVisibility(groupID)
    if err != nil {
        return false, err
    }
    if used {
        return false, nil // 存在关联的模型可见性配置，不允许删除
    }
    return true, nil
}
```

**依赖函数（定义在 `model/model_visibility.go`）：**
```go
// IsGroupUsedByModelVisibility 检查用户组是否被模型可见性配置引用。
// 返回 true 表示该用户组被至少一个模型的可见性配置使用，不应被删除。
func IsGroupUsedByModelVisibility(groupID uint) (bool, error) {
    var count int64
    if err := DB.Model(&ModelVisibilityGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
        return false, err
    }
    return count > 0, nil
}
```

**检查项说明：**

| 关联资源 | 检查逻辑 | 删除受阻时的错误信息 |
|---------|---------|---------------------|
| 模型可见性配置 | `ModelVisibilityGroup` 表中是否有 `group_id` 等于该用户组 ID 的记录 | "用户组存在关联资源，无法删除" |

保证数据一致性，避免孤儿成员记录。

### 5.2 创建/修改用户时指定用户组

**创建用户（`POST /admin/create`）** 和 **修改用户（`POST /admin/update-user`）** 接口均新增可选字段 `group_ids`：

```json
{
  "username": "alice",
  "...": "...",
  "group_ids": [1, 3]
}
```

- `group_ids` 为可选字段，不传或传 `null` 时不修改用户组归属
- 传空数组 `[]` 时，将该用户从所有用户组中移除
- 传非空数组时，先校验所有 `group_id` 是否存在，任一不存在则整体返回 400；校验通过后在同一事务内全量替换该用户的用户组归属（先删除 `user_group_members` 中该用户的所有记录，再批量插入新的关联记录）
- 修改用户组归属时同样受 `MaxMembersPerUserGroup`（10000）限制，超出返回 400

### 5.3 幂等批量添加
`AddGroupMembers` 先校验所有 `user_id` 合法性，再批量 `INSERT OR IGNORE`（SQLite）/ `INSERT IGNORE`（MySQL），重复添加同一用户不报错，接口可安全重试。批量移除同理，不在组内的成员静默忽略。

### 5.4 全量替换
`SetGroupMembers` 在事务内先 `DELETE` 再批量 `INSERT`，适合前端"勾选成员列表"的交互场景，避免逐条 diff 的复杂度。

### 5.5 未分组用户查询
`GetUngroupedUsers` 通过子查询实现：
```sql
SELECT * FROM users
WHERE id NOT IN (
    SELECT DISTINCT user_id FROM user_group_members WHERE identifier = ?
)
ORDER BY created_at ASC
LIMIT ? OFFSET ?
```
同时执行 `COUNT(*)` 查询返回总数，供前端分页展示。

### 5.6 容量限制

| 维度 | 上限 | 校验位置 | 超出行为 |
|------|------|----------|----------|
| 单平台用户组数量 | 1000 个 | `model.CreateUserGroup` | 返回 400，前端弹窗行内提示 |
| 单用户组成员数量 | 10000 人 | `model.AddGroupMembers` / `model.SetGroupMembers` | 返回 400，前端 toast 提示 |

常量定义在 `model/user_group.go`：
```go
const (
    MaxUserGroupsPerPlatform = 1000
    MaxMembersPerUserGroup   = 10000
)
```

前端常量定义在 `UserGroupManagement.tsx`：
```ts
const MAX_GROUPS = 1000;
const MAX_MEMBERS = 10000;
```
前端在提交前做客户端预校验，减少无效请求；后端作为最终防线，保证数据一致性。
两张表均含 `Identifier` 字段，与现有 `registerIdentifierCallbacks` 回调机制完全兼容。当前 SQLite 模式下 `Identifier` 为空字符串，透明无感；迁移 MySQL 多租户时，回调自动注入 `WHERE identifier = ?`，业务代码无需任何改动。

管理员概念本身也与 `identifier` 绑定（`role = admin` 的作用域是当前租户），GORM 回调自动保证管理员只能操作本租户的用户组，无需额外鉴权逻辑。

### 5.5 成员合法性校验
添加/全量设置成员时，先校验所有 `user_id` 是否存在于 `users` 表，任一不存在则整体返回 400，避免脏数据写入。
