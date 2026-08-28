# 模型可见范围管理 — 后端设计方案

> 分支：`feature/model-visibility`
> 日期：2026-04-09（v2 重写）
> 状态：设计评审中

---

## 一、需求背景

当前 Hatchery 的模型权限为"全局开关"模式：管理员通过 `AIModel.Enabled` 字段控制模型是否对**所有用户**可见，无法按用户维度做差异化控制。

实际业务场景需要更细粒度的模型可见性管理：

| 人群 | 可用模型 | 典型场景 |
|------|----------|----------|
| 高层/VIP | 全模型（含 Claude Opus、GPT-5 等高成本模型） | 不限制，全面体验 |
| 普通员工 | 性价比模型（DeepSeek V3、Qwen Plus 等） | 日常办公，控制成本 |
| 设计/营销团队 | banana、seedance 等多模态模型 | 图片生成、视频创作 |

**核心诉求**：

1. 在管控端模型配置页，每个模型新增"**可见范围**"字段（全部用户 / 按分组）
2. 选择"按分组"时，可筛选并关联 1 个或多个用户分组
3. 可见范围决定用户端的模型下拉列表内容
4. 已应用到实例的模型不受后续可见范围变更影响（已绑定保留，但下拉列表中不再展示）

---

## 二、前置依赖：用户分组功能

> ⚠️ 本方案依赖另一位同事完成的"用户分组"能力。以下列出对分组功能的**接口和数据依赖**。

### 2.1 需要分组提供的数据模型

| 需求 | 说明 |
|------|------|
| **分组表** `user_groups` | 需包含 `id`（uint PK）、`identifier`（多租户）、`name`（分组名称）等基本字段 |
| **用户-分组关联表** `user_group_members` | 需包含 `user_id`、`group_id`，支持一个用户属于多个分组 |
| **Identifier 多租户支持** | 分组表需包含 `Identifier` 字段，接入现有 GORM 回调自动隔离 |

### 2.2 需要分组提供的查询函数（后端直接调用）

| 函数签名 | 用途 | 我方调用位置 |
|----------|------|-------------|
| `GetUserGroupIDs(userID uint) ([]uint, error)` | 查询用户所属的所有分组 ID | `controller/openclaw_model.go` — 用户模型列表过滤 + 绑定模型校验 |
| `GetGroupsByIDs(ids []uint) ([]UserGroup, error)` | 批量查询分组信息（ID → 名称） | `controller/admin_models.go` — 管理端模型列表展示分组名称 + 设置可见范围时校验分组存在 |

### 2.3 需要分组提供的管理端 API（前端直接调用，我方后端不经手）

| 接口 | 用途 | 说明 |
|------|------|------|
| `GET /admin/user-groups` | 返回所有分组列表 | 前端模型配置页"可见范围"编辑弹窗中的分组多选下拉框需要拉取全量分组列表。此接口由分组模块提供，我方后端代码不直接调用 |

> 注：`GetAllGroups()` 和 `GetGroupUserIDs()` 是分组模块内部函数，我方后端**不直接调用**，不列为前置依赖。

### 2.4 需要分组在删除分组时执行的操作

> ⚠️ **分组采用硬删除**，分组成员关联也采用**硬删除**。

#### 2.4.1 删除前检查

删除用户组前，必须检查该用户组是否被模型可见性配置引用。如果有模型使用该用户组作为可见范围，则**禁止删除**，返回错误提示。

我方提供检查函数，分组侧需在删除前调用：

```go
// 我方提供（定义在 model/model_visibility.go）
// IsGroupUsedByModelVisibility 检查用户组是否被模型可见性配置引用。
// 返回 true 表示该用户组被至少一个模型的可见性配置使用，不应被删除。
func IsGroupUsedByModelVisibility(groupID uint) (bool, error)
```

分组侧在 `CanDeleteUserGroup` 中调用：

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

#### 2.4.2 删除时级联清理（备用）

如果业务需要支持"强制删除"（即使被模型引用也能删除），则需在删除事务中清理 `model_visibility_groups` 关联记录：

```go
// 我方提供
func CleanupVisibilityByGroupID(tx *gorm.DB, groupID uint) error
```

分组侧调用示例（硬删除场景）：

```go
err := model.DB.Transaction(func(tx *gorm.DB) error {
    // 1. 清理分组成员关联（硬删除）
    if err := tx.Where("group_id = ?", groupID).Delete(&UserGroupMember{}).Error; err != nil {
        return err
    }
    // 2. 清理模型可见性关联（硬删除，因为是关联数据无需保留历史）
    if err := model.CleanupVisibilityByGroupID(tx, groupID); err != nil {
        return err
    }
    // 3. 硬删除分组本身
    if err := tx.Delete(&UserGroup{}, groupID).Error; err != nil {
        return err
    }
    return nil
})
```

> **当前策略**：优先使用 2.4.1 的检查方式，存在关联时返回错误，要求管理员先修改模型的可见范围配置。这样可以避免误删导致模型可见性配置丢失。

### 2.5 如果分组未完成的降级策略

- 可见范围字段先落库，`visibility_type` 默认为 `all`
- 前端分组选择器暂时置灰，tooltip 提示"分组功能开发中"
- 后端 `GetUserGroupIDs` 函数如果不存在，fallback 为返回空切片（等同于"不属于任何分组"，此时按分组配置的模型对该用户不可见——这是安全的降级方向）

---

## 三、整体架构

### 3.1 设计原则

1. **向后兼容**：新增字段 `VisibilityType` 默认 `all`，存量模型自动对所有用户可见，无需数据迁移
2. **已应用不受影响**：模型已绑定到实例的不受可见范围变更影响，仅影响下拉列表展示
3. **多分组并集**：一个模型设置了多个分组，取并集——只要用户在其中任一分组，即可见
4. **复用已有页面**：不新建管理页面，在现有模型配置页增加"可见范围"列
5. **多租户兼容**：所有新表均包含 `Identifier` 字段

---

## 四、数据库设计

### 4.1 修改表：`ai_models`

新增列：

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `visibility_type` | `varchar(20)` | NOT NULL, DEFAULT 'all' | 可见范围类型：`all`=全部用户, `group`=按分组 |

**GORM 迁移方式**：AutoMigrate 自动新增列，DEFAULT 'all'，存量数据自动为"全部用户"。

```go
// AIModel 新增字段
VisibilityType string `gorm:"not null;default:'all'"` // all / group
```

### 4.2 新增表：`model_visibility_groups`

模型-分组关联表。仅当 `AIModel.VisibilityType = 'group'` 时，此表中的记录生效。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | `uint` | PK, AUTO_INCREMENT | 主键 |
| `identifier` | `varchar(255)` | INDEX | 多租户标识 |
| `created_at` | `datetime` | NOT NULL | 创建时间 |
| `ai_model_id` | `uint` | NOT NULL, INDEX | 关联 AI 模型 ID |
| `group_id` | `uint` | NOT NULL, INDEX | 关联用户分组 ID |

**索引**：
- `idx_mvg_model_id` (`ai_model_id`) — 按模型查关联分组
- `idx_mvg_group_id` (`group_id`) — 按分组查关联模型（分组删除时清理）
- `idx_mvg_unique` UNIQUE (`identifier`, `ai_model_id`, `group_id`) — 防止重复关联

**GORM 结构体**：

```go
// ModelVisibilityGroup 模型-分组可见性关联
type ModelVisibilityGroup struct {
    ID         uint      `gorm:"primarykey" json:"id"`
    Identifier string    `gorm:"uniqueIndex:idx_mvg_unique;index;default:''" json:"-"`
    CreatedAt  time.Time `json:"created_at"`
    AIModelID  uint      `gorm:"uniqueIndex:idx_mvg_unique;not null;index" json:"ai_model_id"`
    GroupID    uint      `gorm:"uniqueIndex:idx_mvg_unique;not null;index" json:"group_id"`
}
```

### 4.3 无需修改的表

| 表 | 说明 |
|------|------|
| `users` | **无改动** |
| `site_configs` | **无改动** |

---

## 五、接口设计

### 5.1 模型可见范围管理 API（管理端）

#### 5.1.1 更新模型可见范围 — `POST /admin/models/visibility?id=N`

```
Content-Type: application/json

{
    "visibility_type": "group",
    "group_ids": [1, 3, 5]
}
```

**逻辑**：
1. 校验模型存在
2. 校验 `visibility_type` 为 `all` 或 `group`
3. 如果 `visibility_type = group`：校验 `group_ids` 非空且所有 ID 在分组表中存在
4. **在同一个事务中**执行以下操作：
   - 删除该模型在 `model_visibility_groups` 中的所有旧关联
   - 如果 `visibility_type = group`：批量创建新关联
   - 更新 `AIModel.VisibilityType`
   - 任一步骤失败则整体回滚

**校验规则**：
- `visibility_type`：必填，枚举 `all` / `group`
- `group_ids`：当 `visibility_type=group` 时必填且非空；`visibility_type=all` 时忽略

**Response 成功**:
```json
{ "ok": true }
```

**错误码**（复用项目现有 `writeError` 格式）：

| HTTP 状态码 | 场景 | 错误信息 |
|:---:|------|----------|
| 400 | `visibility_type` 不是 `all` / `group` | `"visibility_type 必须为 all 或 group"` |
| 400 | `visibility_type=group` 但 `group_ids` 为空 | `"按分组可见时必须选择至少一个分组"` |
| 400 | `group_ids` 中包含不存在的分组 ID | `"分组不存在: [5, 9]"` |
| 400 | 请求体 JSON 解析失败 | `"请求参数格式错误"` |
| 404 | 模型 ID 不存在 | `"模型不存在"` |
| 500 | DB 事务失败 | `"更新可见范围失败"` |

#### 5.1.2 获取模型列表（已有接口增强） — `GET /admin/models`

现有响应中每个模型对象新增字段：

```json
{
    "id": 1,
    "provider": "腾讯云 DeepSeek",
    "model_id": "DeepSeek V3 0324",
    "enabled": true,
    "visibility_type": "group",
    "visibility_groups": [
        { "group_id": 1, "group_name": "高层管理" },
        { "group_id": 3, "group_name": "设计团队" }
    ]
}
```

**实现要点**（固定 3 次 DB 查询，无 N+1 问题）：

```
第1步（现有）：查 ai_models 表 → 得到模型列表
        SELECT * FROM ai_models ORDER BY id DESC

第2步（+1次 DB）：筛出 visibility_type="group" 的模型 ID，一次性查出所有关联
        SELECT * FROM model_visibility_groups WHERE ai_model_id IN (2, 3, ...)
        在内存中按 ai_model_id 分组 → map[modelID][]groupID

第3步（+1次 DB）：收集所有去重的 group_id，一次性查分组名称
        GetGroupsByIDs([]uint{1, 5, 7})
        在内存中建 map → map[groupID]groupName

第4步（纯内存）：遍历模型列表组装响应
        visibility_type=all  → visibility_groups: []
        visibility_type=group → 从 map 中取 group_id + group_name
        防御性处理：如果某 group_id 在 map 中查不到名称（分组已被软删除但关联未清理），
        跳过该条，不返回给前端（避免显示"未知分组"），并记录 Warn 日志
```

| 步骤 | DB 查询 | 数据量 |
|------|:---:|---|
| 查模型列表 | 1 次（现有） | < 50 条 |
| 查模型-分组关联 | +1 次 | < 1000 条 |
| 查分组名称 | +1 次 | < 20 条 |
| 内存组装 | 0 次 | — |
| **总计** | **3 次** | — |

#### 5.1.3 查询用户组关联的模型 — `GET /admin/user-groups/associated-models?group_id=N`

用于删除用户组前查询该组关联了哪些模型，方便前端提示用户。

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|:---:|------|
| `group_id` | uint | ✅ | 用户组 ID |

**响应示例**：

```json
{
    "count": 3,
    "models": [
        { "id": 1, "provider": "腾讯云 DeepSeek", "model_id": "DeepSeek V3 0324" },
        { "id": 2, "provider": "OpenAI", "model_id": "gpt-4o" },
        { "id": 5, "provider": "Claude", "model_id": "claude-3-opus" }
    ]
}
```

**响应字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `count` | int | 关联的模型数量 |
| `models` | array | 关联的模型列表 |
| `models[].id` | uint | 模型 ID |
| `models[].provider` | string | 模型提供商 |
| `models[].model_id` | string | 模型标识 |

**实现要点**：

```
第1步：根据 group_id 查 model_visibility_groups 表，获取所有关联的 ai_model_id
        SELECT ai_model_id FROM model_visibility_groups WHERE group_id = ?

第2步：根据 ai_model_id 列表查 ai_models 表，获取模型详情
        SELECT id, provider, model_id FROM ai_models WHERE id IN (...)
```

**错误码**：

| HTTP 状态码 | 场景 | 错误信息 |
|-------------|------|----------|
| 400 | 缺少 group_id 参数 | `"缺少 group_id 参数"` |
| 400 | group_id 格式错误 | `"group_id 格式错误"` |
| 500 | DB 查询失败 | `"查询关联模型失败"` |

**使用场景**：

前端在删除用户组前调用此接口，如果 `count > 0`，则弹窗提示：

> 该用户组已关联 {count} 个模型（{model_names}），删除后这些模型的可见范围配置将受到影响。是否继续？

### 5.2 用户端模型 API 修改

#### 5.2.1 获取可用模型列表 — `GET /openclaw/models`

**当前逻辑**：
```sql
SELECT * FROM ai_models WHERE enabled = true
```

**修改后逻辑**：
```
1. 查询所有 enabled=true 的模型
2. 获取当前用户所属的分组 IDs: userGroupIDs = GetUserGroupIDs(user.ID)
3. 过滤模型:
   - visibility_type = 'all' → 可见
   - visibility_type = 'group' → 检查该模型关联的 group_ids 与 userGroupIDs 是否有交集
4. 返回过滤后的模型列表
```

**举例说明**：

假设系统中有 3 个模型、3 个分组：

| 模型 | visibility_type | 关联分组 |
|------|:---:|---|
| DeepSeek V3 | `all` | —（全员可见） |
| Claude Opus | `group` | 分组1（高层管理） |
| banana | `group` | 分组2（设计团队）、分组3（营销团队） |

用户"张三"属于**分组2（设计团队）**，执行流程：

```
第1步：查出所有 enabled=true 的模型 → [DeepSeek V3, Claude Opus, banana]

第2步：查张三属于哪些分组 → [分组2]        ← 1 次 DB 查询

第3步：一次性查出所有 group 类型模型的分组关联 ← 1 次 DB 查询
       Claude Opus → [分组1]
       banana      → [分组2, 分组3]

第4步：逐个判断（纯内存，无 DB）：
  - DeepSeek V3: visibility_type=all           → ✅ 可见
  - Claude Opus: 关联[分组1], 张三在[分组2]    → 无交集 → ❌ 不可见
  - banana:      关联[分组2,分组3], 张三在[分组2] → 有交集 → ✅ 可见

第5步：返回 [DeepSeek V3, banana]
```

> **性能说明**：无论模型数量多少，整个过滤过程固定只有 **2 次 DB 查询**（1 次查用户分组 + 1 次批量查模型-分组关联），其余全部在内存中完成。

**伪代码**（实现在 `controller/openclaw_model.go` 中，作为 `HandleModelsList` 的内部辅助函数）：
```go
func filterModelsByVisibility(models []model.AIModel, userID uint) []model.AIModel {
    // 第1次 DB 查询：获取用户所属的所有分组 ID
    userGroupIDs, err := model.GetUserGroupIDs(userID)
    if err != nil {
        slog.Error("[ModelVisibility] 查询用户分组失败", "user_id", userID, "error", err)
        userGroupIDs = nil // 查询失败，安全降级：视为不属于任何分组
    }
    userGroupSet := toSet(userGroupIDs) // 转为 set，便于 O(1) 判断交集

    // 收集所有需要检查分组的模型 ID
    var modelIDs []uint
    for _, m := range models {
        if m.VisibilityType == "group" {
            modelIDs = append(modelIDs, m.ID)
        }
    }

    // 第2次 DB 查询：一次性批量查出所有 group 类型模型的分组关联
    // 返回 map[modelID][]groupID，例如 {2: [1], 3: [2, 3]}
    modelGroupMap, err := model.GetModelVisibilityGroupIDs(modelIDs)
    if err != nil {
        slog.Error("[ModelVisibility] 批量查询模型分组关联失败", "error", err)
        modelGroupMap = nil // 查询失败，安全降级：所有 group 类型模型视为不可见
    }

    // 纯内存过滤，无额外 DB 查询
    var result []model.AIModel
    for _, m := range models {
        if m.VisibilityType == "all" {
            result = append(result, m) // all 类型直接通过
        } else if m.VisibilityType == "group" {
            groupIDs := modelGroupMap[m.ID]
            if hasIntersection(groupIDs, userGroupSet) {
                result = append(result, m) // 用户分组与模型分组有交集 → 可见
            }
        }
    }
    return result
}
```

#### 5.2.2 绑定模型到实例 — `POST /openclaw/set-model`

在现有的 `enabled = true` 校验**之后**，新增可见性校验：

```go
visible, err := model.IsModelVisibleToUser(aiModel, user.ID)
if err != nil {
    slog.Error("[ModelVisibility] 可见性检查失败", "user_id", user.ID, "model_id", aiModel.ID, "error", err)
    writeError(w, r, http.StatusInternalServerError, errors.New("模型可见性检查失败"))
    return
}
if !visible {
    writeError(w, r, http.StatusForbidden, errors.New("您无权使用该模型"))
    return
}
```

**新增错误码**：

| HTTP 状态码 | 场景 | 错误信息 |
|:---:|------|----------|
| 403 | 用户尝试绑定不在可见范围内的模型 | `"您无权使用该模型"` |

**关键规则**：已应用到实例的模型不受影响。具体行为：
- 用户尝试**新绑定**模型时检查可见性 → 不在可见范围则 403
- 已绑定的模型继续使用 → **不做可见性检查**（已应用保留）
- **自定义模型（ai_model_id=0）**：前端传 `ai_model_id=0` 表示选择自定义模型，进入 `handleCustomModel` 分支。注意这里的 0 不是数据库 ID，而是前端约定的特殊值。`hatchery/custom` 内置记录在 `ai_models` 表中有自己的自增 ID（如 id=1）。可见性检查需用该记录的**真实 DB ID**：先查出 `customFlag`（现有代码已有此查询），再调 `IsModelVisibleToUser(&customFlag, user.ID)`。如果不可见则 403，错误信息同上

#### 5.2.2.1 创建实例时的默认模型 — `POST /openclaw/create`

**问题场景**：管理员配置了系统默认模型（`DefaultModelID`），同时将该模型设置为"部分用户可见"（`visibility_type=group`）。对于不在可见范围内的用户，创建实例时仍会自动应用该默认模型，这不符合预期。

**解决方案**：创建实例时，在应用默认模型前检查用户对该模型的可见性。

**修改后的逻辑**：

```go
// 若配置了默认模型，检查用户可见性后再应用
if config.DefaultModelID > 0 {
    var defaultModel model.AIModel
    if model.DB.Where("id = ? AND enabled = ?", config.DefaultModelID, true).First(&defaultModel).Error == nil {
        // 检查用户对默认模型的可见性
        visible, visErr := model.IsModelVisibleToUser(&defaultModel, user.ID)
        if visErr != nil {
            slog.Warn("[DefaultModel] 可见性检查失败，跳过默认模型", "user_id", user.ID, "model_id", defaultModel.ID, "error", visErr)
        } else if visible {
            model.DB.Model(&placeholderInstance).Update("ai_model_id", defaultModel.ID)
            go injectDefaultModel(placeholderInstance.ID, defaultModel.ID)
        } else {
            slog.Info("[DefaultModel] 默认模型对用户不可见，跳过", "user_id", user.ID, "model_id", defaultModel.ID)
        }
    }
}
```

**行为说明**：

| 场景 | 行为 |
|------|------|
| 默认模型对用户可见 | ✅ 正常应用默认模型 |
| 默认模型对用户不可见 | ⏭️ 跳过，不应用默认模型（记录日志） |
| 可见性检查失败 | ⏭️ 跳过，不应用默认模型（记录警告日志） |

**注意**：用户创建的实例不会自动绑定任何模型，需要用户手动选择一个可见的模型。

#### 5.2.3 获取实例列表 — `GET /openclaw/list`

**问题场景**：当管理员修改模型可见范围后，用户已应用的模型可能不在其可见列表中。此时前端无法从 `/openclaw/models` 返回的列表中匹配到已应用模型的信息，导致"已应用模型"显示为空。

**解决方案**：在 `/openclaw/list` 接口返回实例信息时，**额外返回已应用模型的基本信息**（`model_provider`、`model_name`），不受可见性过滤影响。

**修改后的响应结构**：

```json
{
  "instances": [
    {
      "ID": 1,
      "AIModelID": 3,
      "CustomModelConfig": "",
      "role_name": "通用助手",
      "model_provider": "腾讯云 DeepSeek",
      "model_name": "DeepSeek V3 0324"
    }
  ]
}
```

**新增字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `model_provider` | `string` | 模型提供商，仅当 `AIModelID > 1` 时返回 |
| `model_name` | `string` | 模型名称，仅当 `AIModelID > 1` 时返回 |

**实现逻辑**：

```go
// 收集需要查询模型信息的 AIModelID（> 1 的才是预置模型）
var modelIDs []uint
for _, inst := range freshInstances {
    if inst.AIModelID > 1 {
        modelIDs = append(modelIDs, inst.AIModelID)
    }
}

// 批量查询模型信息
modelInfoMap := make(map[uint]model.AIModel)
if len(modelIDs) > 0 {
    var models []model.AIModel
    model.DB.Where("id IN ?", modelIDs).Find(&models)
    for _, m := range models {
        modelInfoMap[m.ID] = m
    }
}

// 填充响应
for i, inst := range freshInstances {
    item := instanceWithRole{Instance: inst, RoleName: roleNameMap[inst.RoleID]}
    if inst.AIModelID > 1 {
        if m, ok := modelInfoMap[inst.AIModelID]; ok {
            item.ModelProvider = m.Provider
            item.ModelName = m.ModelName
        }
    }
    result[i] = item
}
```

**为什么 `AIModelID <= 1` 不返回模型信息**：

| `AIModelID` | 说明 | 模型信息来源 |
|-------------|------|-------------|
| `0` | 自定义模型（旧逻辑） | 前端从 `CustomModelConfig` 解析 |
| `1` | `hatchery/custom` 内置记录 | 前端从 `CustomModelConfig` 解析 |
| `> 1` | 预置模型 | 后端返回 `model_provider`、`model_name` |

**前端改动**：

1. `InstanceInfo` 类型新增 `model_provider`、`model_name` 可选字段
2. 显示已应用模型时，优先使用实例返回的模型信息（即使模型不在可见列表中）

**效果**：

| 场景 | 之前 | 现在 |
|------|------|------|
| 预置模型在可见范围内 | ✅ 正常显示 | ✅ 正常显示 |
| 预置模型不在可见范围内 | ❌ 不显示 | ✅ 使用后端返回的信息显示 |
| 自定义模型 | ✅ 用 `CustomModelConfig` | ✅ 逻辑不变 |

### 5.3 LLM 代理 API 修改

#### `POST /v1/chat/completions`

**不做可见性检查**。原因：

1. LLM 代理使用的是实例已绑定的模型（`instance.AIModelID`），不是用户实时选择的
2. 需求明确"已应用的模型不受影响"
3. 已有的三层配额检查（全局/模型级/用户级）+ `enabled` 检查已足够

### 5.4 模型删除级联处理

#### 删除模型 — `POST /admin/models/delete?id=N`

在现有逻辑基础上新增（在同一事务中）：
- 调用 `CleanupVisibilityByModelID(tx, modelID)` 删除 `model_visibility_groups` 中所有 `ai_model_id = N` 的关联记录

### 5.5 分组软删除级联处理（需分组功能配合）

分组采用**软删除**（GORM `DeletedAt`），软删除后标准查询将查不到该分组。为避免产生"幽灵关联"（关联记录指向已软删除的分组），分组软删除时必须同步**硬删除** `model_visibility_groups` 中对应的关联记录。

详细说明和调用示例见 **2.4 节**。

**额外处理**：如果某模型的所有关联分组都被软删除（导致关联全部被清理），`visibility_type` 仍为 `group` 但无关联分组 → 该模型对所有用户不可见（安全方向）。管理员需手动调整为 `all` 或重新关联分组。

---

## 六、辅助查询函数设计

新建文件 `model/model_visibility.go`：

### 6.1 `IsModelVisibleToUser`

> **适用场景**：单个模型的可见性判断（绑定模型 `HandleSetModel` / `handleCustomModel`）。
> 内部会查 1~2 次 DB（查用户分组 + 查关联），适合单模型判断，**不适合循环调用**。
> 模型列表批量过滤请使用 `filterModelsByVisibility`（见 5.2.1 节），已做批量优化。

```go
// IsModelVisibleToUser 检查某模型对指定用户是否可见。
// visibility_type=all 时返回 (true, nil)。
// visibility_type=group 时检查用户所属分组与模型关联分组是否有交集。
// 查询失败时返回 (false, err)，调用方应按不可见处理并记录日志。
func IsModelVisibleToUser(aiModel *AIModel, userID uint) (bool, error) {
    if aiModel.VisibilityType != "group" {
        return true, nil // all 或空值均视为全部可见
    }
    userGroupIDs, err := GetUserGroupIDs(userID)
    if err != nil {
        return false, fmt.Errorf("查询用户分组失败: %w", err)
    }
    if len(userGroupIDs) == 0 {
        return false, nil // 用户不属于任何分组
    }
    var count int64
    if err := DB.Model(&ModelVisibilityGroup{}).
        Where("ai_model_id = ? AND group_id IN ?", aiModel.ID, userGroupIDs).
        Count(&count).Error; err != nil {
        return false, fmt.Errorf("查询模型可见性关联失败: %w", err)
    }
    return count > 0, nil
}
```

### 6.2 `GetModelVisibilityGroupIDs`

```go
// GetModelVisibilityGroupIDs 批量查询多个模型的可见分组 ID。
// 返回 map[modelID][]groupID。
func GetModelVisibilityGroupIDs(modelIDs []uint) (map[uint][]uint, error) {
    if len(modelIDs) == 0 {
        return nil, nil
    }
    var rows []ModelVisibilityGroup
    if err := DB.Where("ai_model_id IN ?", modelIDs).Find(&rows).Error; err != nil {
        return nil, fmt.Errorf("批量查询模型可见性关联失败: %w", err)
    }
    result := make(map[uint][]uint)
    for _, r := range rows {
        result[r.AIModelID] = append(result[r.AIModelID], r.GroupID)
    }
    return result, nil
}
```

### 6.3 `GetUserGroupIDs`（依赖分组功能）

```go
// GetUserGroupIDs 查询用户所属的所有分组 ID。
// 【依赖分组功能】此函数由分组模块实现，此处为占位签名。
// 如果分组功能未完成，fallback 返回空切片 + nil error。
var GetUserGroupIDs = func(userID uint) ([]uint, error) {
    // TODO: 由分组模块实现
    // var groupIDs []uint
    // err := DB.Model(&UserGroupMember{}).Where("user_id = ?", userID).Pluck("group_id", &groupIDs).Error
    // return groupIDs, err
    return nil, nil // fallback: 用户不属于任何分组
}
```

### 6.4 `CleanupVisibilityByGroupID`

```go
// CleanupVisibilityByGroupID 清理某分组被删除后的模型可见性关联。
// 必须传入事务 tx，确保与分组删除在同一事务中执行。
func CleanupVisibilityByGroupID(tx *gorm.DB, groupID uint) error {
    return tx.Where("group_id = ?", groupID).Delete(&ModelVisibilityGroup{}).Error
}
```

### 6.5 `IsGroupUsedByModelVisibility`

```go
// IsGroupUsedByModelVisibility 检查用户组是否被模型可见性配置引用。
// 返回 true 表示该用户组被至少一个模型的可见性配置使用，不应被删除。
// 由 CanDeleteUserGroup 调用，用于删除用户组前的前置检查。
func IsGroupUsedByModelVisibility(groupID uint) (bool, error) {
    var count int64
    if err := DB.Model(&ModelVisibilityGroup{}).Where("group_id = ?", groupID).Count(&count).Error; err != nil {
        return false, err
    }
    return count > 0, nil
}
```

### 6.6 `GetModelsAssociatedWithGroup`

```go
// GetModelsAssociatedWithGroup 查询与指定用户组关联的模型列表。
// 用于删除用户组前提示用户该组关联了哪些模型。
// 返回关联的模型 ID 列表，调用方可根据需要进一步查询模型详情。
func GetModelsAssociatedWithGroup(groupID uint) ([]uint, error) {
    var modelIDs []uint
    if err := DB.Model(&ModelVisibilityGroup{}).
        Where("group_id = ?", groupID).
        Pluck("ai_model_id", &modelIDs).Error; err != nil {
        return nil, err
    }
    return modelIDs, nil
}
```

### 6.7 `CleanupVisibilityByModelID`

```go
// CleanupVisibilityByModelID 清理某模型被删除后的可见性关联。
// 由模型删除逻辑调用。同样建议在事务中调用。
func CleanupVisibilityByModelID(tx *gorm.DB, modelID uint) error {
    return tx.Where("ai_model_id = ?", modelID).Delete(&ModelVisibilityGroup{}).Error
}
```

---

## 七、路由注册与审计日志

### 7.1 路由注册

在 `main.go` 中新增路由：

```go
// Model visibility management
http.HandleFunc("/admin/models/visibility", controller.WithAudit(controller.WithOpenAPI(controller.HandleUpdateModelVisibility)))

// User group associated models query
http.HandleFunc("/admin/user-groups/associated-models", controller.WithOpenAPI(controller.HandleGetGroupAssociatedModels))
```

新增 **2 个路由**，其余在现有接口上修改。

### 7.2 审计日志（操作记录）

项目使用 `WithAudit` 中间件自动记录操作日志，机制是通过 `auditRules` map 匹配路径 → 自动写入 `audit_logs` 表，前端"操作记录"页面会自动展示。

需要在 `controller/audit.go` 的 `auditRules` 中新增：

```go
"/admin/models/visibility": {"model_visibility_update", "ai_model"},
```

**已有路由自动覆盖的操作记录**（无需额外配置）：

| 路由 | 审计 action | 已有 |
|------|------------|:---:|
| `/admin/models/create` | `model_create` | ✅ |
| `/admin/models/update` | `model_update` | ✅ |
| `/admin/models/delete` | `model_delete` | ✅ |
| `/admin/models/toggle` | `model_toggle` | ✅ |
| `/openclaw/set-model` | `instance_set_model` | ✅ |
| `/admin/models/visibility` | `model_visibility_update` | **新增** |

> 由于新路由使用了 `WithAudit` 装饰器，操作记录会**自动记录到 `audit_logs` 表**，前端操作记录页面无需改动即可展示。

---

## 八、性能分析

### 8.1 热路径影响

| 路径 | 当前 | 修改后 | 影响 |
|------|------|--------|------|
| `GET /openclaw/models` | 1 次 DB 查询 | +1 次查用户分组 + 1 次查模型-分组关联 | +2 次简单查询，<2ms |
| `POST /openclaw/set-model` | 1 次 DB 查询 | +1 次可见性检查（`IsModelVisibleToUser`） | +1~2 次索引查询，<1ms |
| `POST /v1/chat/completions` | 4 次 DB 查询 | **无改动** | 零影响 |

### 8.2 数据量估算

- `ai_models`：每租户 < 50 条
- `model_visibility_groups`：每模型 × 每分组 = 50 × 20 = 1000 条上限
- 模型列表查询：一次 IN 查询，数据量极小

### 8.3 优化空间（当前不实施）

- 可在内存中缓存 `userID → groupIDs` 映射，减少 LLM 代理热路径的 DB 查询
- 当前规模下无必要

---

## 九、安全设计

### 9.1 权限控制

| 接口 | 权限要求 |
|------|----------|
| 模型可见范围设置（`/admin/models/visibility`） | `requireAdmin` |
| 模型列表过滤（`/openclaw/models`） | `requireLogin`（已有） |
| 模型绑定校验（`/openclaw/set-model`） | `requireLogin`（已有） |
| LLM 代理（`/v1/chat/completions`） | Bearer Token（已有，**不加可见性检查**） |

### 9.2 数据安全

- 用户端接口仅返回用户可见的模型，**不泄露不可见模型的存在**
- 管理端接口返回所有模型（含可见范围配置），APIKey 脱敏

### 9.3 输入校验

| 字段 | 校验规则 |
|------|----------|
| `visibility_type` | 枚举：`all` / `group` |
| `group_ids` | 数组，每个元素为正整数，数组长度 1-100，所有 ID 必须存在于分组表 |

### 9.4 SQL 注入防护

所有查询使用 GORM 参数绑定（`Where("id = ?", id)`），不做字符串拼接。

---

## 十、多租户兼容

### 10.1 Identifier 字段

`ModelVisibilityGroup` 包含 `Identifier` 字段，遵循现有多租户隔离机制：

- **SQLite 模式**：`Identifier` 为空，单租户
- **MySQL 模式**：由 GORM 回调自动填充和过滤

### 10.2 唯一约束

`model_visibility_groups` 的唯一约束为 `(identifier, ai_model_id, group_id)` 联合索引，确保同租户内不重复关联。

---

## 十一、AutoMigrate 注册

在 `model/db.go` 的 `AutoMigrate` 调用中添加新模型：

```go
DB.AutoMigrate(
    // ... 现有模型 ...
    &ModelVisibilityGroup{},
)
```

`AIModel.VisibilityType` 新增字段会被 AutoMigrate 自动添加到已有 `ai_models` 表，DEFAULT 'all'。

---

## 十二、文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `model/ai_model.go` | 修改 | AIModel 增加 `VisibilityType` 字段 |
| `model/model_visibility.go` | **新增** | ModelVisibilityGroup 结构体 + 辅助查询函数（含 `IsGroupUsedByModelVisibility` 用于删除用户组前检查） |
| `model/db.go` | 修改 | AutoMigrate 添加 ModelVisibilityGroup |
| `controller/admin_models.go` | 修改 | 模型列表新增 visibility 字段返回；删除模型级联清理；新增 HandleUpdateModelVisibility |
| `controller/audit.go` | 修改 | `auditRules` 新增 `model_visibility_update` 条目 |
| `controller/openclaw_model.go` | 修改 | HandleModelsList 增加可见性过滤；HandleSetModel 增加可见性校验；handleCustomModel 增加可见性校验 |
| `controller/openclaw.go` | 修改 | HandleInstanceList 返回已应用模型的 `model_provider`、`model_name`（见 5.2.3 节）；HandleCreateInstance 创建实例时检查默认模型可见性（见 5.2.2.1 节） |
| `controller/auth_oneid.go` | **无改动** |
| `controller/admin_users.go` | **无改动** |
| `controller/admin_config.go` | **无改动** |
| `controller/llm_proxy.go` | **无改动**（已绑定模型不受影响） |
| `main.go` | 修改 | 注册 1 个新路由 |

### 前端文件变更（`openclaw-enterprise-fronted`）

| 文件 | 操作 | 说明 |
|------|------|------|
| `client/src/types/api.ts` | 修改 | `InstanceInfo` 新增 `model_provider`、`model_name` 可选字段 |
| `client/src/pages/tenant/ModelsColumn.tsx` | 修改 | 显示已应用模型时优先使用实例返回的模型信息 |

---

## 十三、边界场景处理

| 场景 | 处理方式 |
|------|----------|
| 模型 `visibility_type=group` 但未关联任何分组 | 该模型对所有普通用户不可见（安全方向）；管理端可正常看到并管理 |
| 模型关联的分组被软删除 | 分组软删除时调用 `CleanupVisibilityByGroupID` **硬删除**关联记录（见 2.4 节）；如果所有分组都被软删除，同上 |
| 已软删除的分组被恢复 | 恢复后管理员需重新配置模型的可见范围（关联已在软删除时被清理，这是设计预期） |
| 模型被删除 | 模型删除时调用 `CleanupVisibilityByModelID` 清理关联 |
| 用户不属于任何分组 | `GetUserGroupIDs` 返回空 → 所有 `visibility_type=group` 的模型对该用户不可见，`visibility_type=all` 的模型可见 |
| 可见范围从 `all` 改为 `group` | 已绑定到实例的模型继续工作（LLM 代理不检查可见性），但下拉列表中不再展示该模型 |
| 可见范围从 `group` 改为 `all` | 所有用户立即可在下拉列表中看到该模型 |
| 可见范围修改，按分组改为不同的分组集合 | 旧分组的用户下拉列表中不再看到该模型，新分组的用户可以看到 |
| 自定义模型（`hatchery/custom`） | 内置记录也支持 `visibility_type` 配置，与普通模型一致 |
| 管理员（admin 角色） | 管理员在用户端**也受可见性限制**，与普通用户一致（管理员需通过管理端查看/配置所有模型） |
| 分组功能未上线 | `GetUserGroupIDs` fallback 返回空 → 所有 `group` 类型模型不可见，`all` 类型正常 → 安全降级 |

---

## 十四、前后端兼容性保证

> **后端可先于前端独立发布**，不会 break 老版本前端。

### 兼容性设计要点

1. **存量数据零迁移**：`visibility_type` DEFAULT 'all'，所有存量模型自动对全部用户可见
2. **响应只增不改**：`GET /admin/models` 响应中新增 `visibility_type`、`visibility_groups` 字段，老前端自动忽略
3. **用户端模型列表格式不变**：`GET /openclaw/models` 响应格式不变，只是内容可能因可见性过滤而减少
4. **新增接口独立**：`/admin/models/visibility` 为新路由，老前端不调用
5. **LLM 代理零改动**：已绑定模型的实例完全不受影响

### 风险场景

| 场景 | 影响 | 风险等级 |
|------|------|:---:|
| 后端上线，前端未更新 | 所有模型 `visibility_type=all`，行为完全等同当前 | 无 |
| 管理员用新前端设置了按分组可见 | 部分用户下拉列表减少模型选项，已绑定实例不受影响 | 无 |

---

## 十五、关键日志规范

> 代码中必须保留关键日志，使用 `log/slog` 标准库，遵循现有项目的日志风格（如 `[DefaultModel]`、`[LLM Proxy]` 前缀标签），便于线上问题排查。

### 15.1 日志埋点清单

| 场景 | 级别 | 日志标签 | 必含字段 | 示例 |
|------|------|----------|----------|------|
| 管理员修改模型可见范围 | `Info` | `[ModelVisibility]` | model_id, visibility_type, group_ids, admin_user | `slog.Info("[ModelVisibility] 可见范围已更新", "model_id", 3, "visibility_type", "group", "group_ids", "[1,5]", "admin", "alice")` |
| 修改可见范围失败（DB 错误） | `Error` | `[ModelVisibility]` | model_id, error | `slog.Error("[ModelVisibility] 更新失败", "model_id", 3, "error", err)` |
| 用户模型列表被可见性过滤 | `Info` | `[ModelVisibility]` | user_id, total_models, visible_models, user_groups | `slog.Info("[ModelVisibility] 模型列表已过滤", "user_id", 42, "total", 10, "visible", 6, "user_groups", "[1,3]")` |
| 用户绑定模型被可见性拦截 | `Warn` | `[ModelVisibility]` | user_id, model_id, user_groups, model_groups | `slog.Warn("[ModelVisibility] 绑定被拒：模型不在可见范围", "user_id", 42, "model_id", 3, "user_groups", "[1]", "model_groups", "[5,7]")` |
| 删除模型级联清理可见性关联 | `Info` | `[ModelVisibility]` | model_id, deleted_count | `slog.Info("[ModelVisibility] 模型删除，级联清理可见性关联", "model_id", 3, "deleted", 2)` |
| 删除分组级联清理可见性关联 | `Info` | `[ModelVisibility]` | group_id, deleted_count | `slog.Info("[ModelVisibility] 分组删除，级联清理可见性关联", "group_id", 5, "deleted", 3)` |
| 查询用户分组失败（分组服务异常） | `Error` | `[ModelVisibility]` | user_id, error | `slog.Error("[ModelVisibility] 查询用户分组失败", "user_id", 42, "error", err)` |

### 15.2 日志规范要求

1. **统一标签前缀**：所有可见性相关日志使用 `[ModelVisibility]` 前缀，便于 `grep` 过滤
2. **结构化字段**：使用 `slog` 的 key-value 参数，不拼接字符串，便于日志系统解析
3. **敏感信息脱敏**：日志中不输出 APIKey、用户密码等敏感字段
4. **不记录正常热路径**：`GET /openclaw/models` 每次请求都会触发过滤，只在**过滤结果与全量不同时**记录 Info 日志（即有模型被过滤掉时），避免日志量过大
5. **拦截必记录**：所有 403 拦截（绑定被拒）必须记录 Warn 日志，含用户 ID、模型 ID、用户分组、模型分组，方便排查"为什么用户看不到某个模型"

---

## 十六、测试要求

> **所有新增/修改的核心逻辑必须有对应的单元测试**，测试文件与源文件同目录，命名 `*_test.go`。

### 16.1 单测方案说明

#### 测试环境搭建

使用 SQLite 内存数据库（`:memory:`）作为测试 DB，每个测试函数独立初始化，互不干扰：

```go
func setupTestDB(t *testing.T) {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    require.NoError(t, err)
    db.AutoMigrate(&AIModel{}, &ModelVisibilityGroup{}, &User{})
    model.DB = db
}
```

#### 分组依赖的 Mock 策略

`GetUserGroupIDs` 依赖分组模块，单测中通过**函数变量**实现 mock：

```go
// model/model_visibility.go
var GetUserGroupIDs = func(userID uint) ([]uint, error) {
    // 默认实现：查 DB（分组模块提供）
    // 分组未完成时 fallback 返回空切片
    return nil, nil
}

// 单测中替换：
func TestIsModelVisibleToUser_GroupType_InGroup(t *testing.T) {
    setupTestDB(t)
    // mock：用户 42 属于分组 1 和分组 3
    model.GetUserGroupIDs = func(userID uint) ([]uint, error) {
        if userID == 42 { return []uint{1, 3}, nil }
        return nil, nil
    }
    defer func() { model.GetUserGroupIDs = nil }() // 恢复

    // ... 测试逻辑 ...
}
```

这种方式**不需要引入第三方 mock 框架**，与项目现有测试风格一致。

#### 事务测试

级联清理函数接受 `tx *gorm.DB` 参数，单测中直接传 `model.DB`（内存 SQLite 不需要真正事务），验证记录是否被正确删除。

### 16.2 数据层单测（`model/model_visibility_test.go`）

| 测试用例 | 覆盖场景 |
|----------|----------|
| `TestIsModelVisibleToUser_AllType` | `visibility_type=all` → 始终 true |
| `TestIsModelVisibleToUser_GroupType_InGroup` | 用户在可见分组中 → true |
| `TestIsModelVisibleToUser_GroupType_NotInGroup` | 用户不在可见分组中 → false |
| `TestIsModelVisibleToUser_GroupType_NoGroups` | 模型无关联分组 → false |
| `TestIsModelVisibleToUser_UserNoGroup` | 用户不属于任何分组 → false |
| `TestIsModelVisibleToUser_MultiGroup_Union` | 模型关联多个分组，用户在其中一个 → true（验证并集） |
| `TestIsModelVisibleToUser_GetUserGroupIDsError` | `GetUserGroupIDs` 返回 error → 返回 (false, err) |
| `TestIsModelVisibleToUser_DBError` | `model_visibility_groups` 查询失败 → 返回 (false, err) |
| `TestGetModelVisibilityGroupIDs` | 批量查询正确 |
| `TestGetModelVisibilityGroupIDs_Empty` | 空输入返回 nil |
| `TestGetModelVisibilityGroupIDs_DBError` | DB 查询失败 → 返回 (nil, err) |
| `TestCleanupVisibilityByGroupID` | 删除分组后关联记录清理，其他分组不受影响 |
| `TestCleanupVisibilityByModelID` | 删除模型后关联记录清理，其他模型不受影响 |

### 16.3 接口层单测（`controller/admin_models_test.go` 补充）

| 测试用例 | 覆盖场景 |
|----------|----------|
| `TestUpdateModelVisibility_All` | 设置为 all，清理关联 |
| `TestUpdateModelVisibility_Group` | 设置为 group，创建关联 |
| `TestUpdateModelVisibility_SwitchGroupToAll` | 从 group 切换为 all，关联被清理 |
| `TestUpdateModelVisibility_UpdateGroups` | 修改分组集合，旧关联删除新关联创建 |
| `TestUpdateModelVisibility_InvalidType` | 非法类型报错 |
| `TestUpdateModelVisibility_EmptyGroups` | group 类型但空分组数组报错 |
| `TestUpdateModelVisibility_InvalidGroupID` | 不存在的分组 ID 报错 |
| `TestUpdateModelVisibility_ModelNotFound` | 模型不存在报错 |
| `TestAdminModelsList_IncludesVisibility` | 模型列表响应包含 visibility_type 和 visibility_groups |

### 16.4 权限校验单测（`controller/openclaw_model_test.go` 补充）

| 测试用例 | 覆盖场景 |
|----------|----------|
| `TestModelsList_AllVisible` | 全部模型 visibility_type=all → 返回所有 |
| `TestModelsList_GroupFiltered` | 用户在分组中 → 返回分组内模型 |
| `TestModelsList_GroupFiltered_NotInGroup` | 用户不在分组中 → 不返回 |
| `TestModelsList_MixedVisibility` | 混合 all + group 类型，正确过滤 |
| `TestSetModel_VisibleAllowed` | 模型可见 → 绑定成功 |
| `TestSetModel_VisibleDenied` | 模型不可见 → 403 |
| `TestCustomModel_VisibleAllowed` | hatchery/custom 在可见范围 → 自定义模型成功 |
| `TestCustomModel_VisibleDenied` | hatchery/custom 不在可见范围 → 403 |

### 16.5 级联清理单测

| 测试用例 | 覆盖场景 |
|----------|----------|
| `TestDeleteModel_CascadeVisibility` | 删除模型 → 关联记录清理 |
| `TestDeleteGroup_CascadeVisibility` | 删除分组 → 关联记录清理 |
| `TestDeleteGroup_OtherModelsUnaffected` | 删除分组 → 其他分组的关联不受影响 |

### 16.6 集成测试

- [ ] 完整流程：创建模型 → 设置按分组可见 → 分组内用户可见 → 分组外用户不可见
- [ ] 向后兼容：存量模型 `visibility_type=all` 对所有用户可见
- [ ] 已应用不受影响：绑定模型后修改可见范围，LLM 代理正常工作
- [ ] 分组并集：模型关联多个分组，多分组用户均可见
- [ ] 多租户隔离：不同租户的模型可见性互不影响

---

## 十七、开发顺序建议

```
Phase 1: 数据层  ← 无依赖，可立即开始
  ├── model/ai_model.go（+VisibilityType 字段）
  ├── model/model_visibility.go（ModelVisibilityGroup 结构体 + 辅助函数）
  ├── model/model_visibility_test.go（数据层单测）
  └── model/db.go（AutoMigrate）

Phase 2: 管理 API  ← 依赖 Phase 1 + 分组功能（软依赖）
  ├── controller/admin_models.go（可见范围管理 + 模型列表增强 + 删除级联）
  ├── controller/admin_models_test.go（接口层单测）
  ├── controller/audit.go（新增 auditRules 条目）
  └── main.go（路由注册）
  │
  │  分组依赖点：
  │  ├── POST /admin/models/visibility 校验 group_ids 存在 → 需要 GetGroupsByIDs
  │  └── GET /admin/models 展示分组名称 → 需要 GetGroupsByIDs
  │  降级：分组未完成时，visibility_type 只能设为 all（group 选项前端置灰）

Phase 3: 用户端权限校验  ← 依赖 Phase 1 + 分组功能（软依赖）
  ├── controller/openclaw_model.go（模型列表过滤 + 绑定校验）
  └── 补充权限相关单测
  │
  │  分组依赖点：
  │  └── 模型列表过滤 → 需要 GetUserGroupIDs
  │  降级：分组未完成时，GetUserGroupIDs 返回空 → group 类型模型不可见，all 类型正常
```

Phase 1 可立即开始。Phase 2 和 Phase 3 均**软依赖**分组功能，可并行开发：
- 分组未完成时：Phase 2 的可见范围只能设为 `all`；Phase 3 的 `group` 类型模型安全降级为不可见
- 分组完成后：对接 `GetUserGroupIDs` / `GetGroupsByIDs`，全部功能生效
