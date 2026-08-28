# Hermes/LightclawACE 智能体类型支持 - 技术设计方案 V3（硬编码版）

## 1. 需求概述

在 OpenClaw 企业版系统中支持多种智能体类型，初始包含：
- **OpenClaw**：现有智能体类型，功能最完整
- **Hermes**：新增智能体类型，功能受限
- **LightclawACE**：新增智能体类型，功能受限

**关键设计原则**：
1. **类型硬编码** - 智能体类型定义在代码中，通过常量和 Map 配置
2. **前后端一致** - 前端不支持的功能，后端必须同步做好防护
3. **RESTful 规范** - API 设计遵循项目现有风格
4. **日志可追溯** - 所有关键操作记录日志
5. **向后兼容** - 后端升级过程中，旧版前端必须能正常工作
6. **增量开发** - 优先复用现有代码，仅做必要的增量修改

### 1.1 向后兼容策略

| 组件 | 策略 |
|------|------|
| `GetEnabledImage()` | **保留原有函数**，返回任意一个启用的镜像 |
| `GetEnabledImageByType()` | **新增函数**，按类型获取启用镜像 |
| 创建实例 | `agent_type` 参数可选，默认 `openclaw` |
| 镜像启用逻辑 | 无类型镜像保持全局互斥；有类型镜像改为同类型互斥 |
| 实例列表 | 响应增加字段，但不影响旧前端解析 |

### 1.2 复用现有代码

| 现有代码 | 复用方式 |
|----------|----------|
| `model/ai_image.go` | 新增字段和函数，保留 `GetEnabledImage()` |
| `model/instance.go` | 新增字段，不修改现有逻辑 |
| `controller/admin_images.go` | 修改 `HandleEnableImage`，增量改为同类型互斥 |
| `controller/audit.go` | 在 `auditRules` map 中增加条目 |
| `common/image.go` | 保持不变，`CandidateImages` 逻辑复用 |
| 各 Validate 函数 | 参考现有风格（如 `ValidateInternetAccessible`）编写新校验 |

### 1.3 功能差异矩阵

| 功能 | OpenClaw | Hermes | LightclawACE |
|------|----------|--------|--------------|
| 角色配置 | ✅ | ❌ | ❌ |
| 模型配置 | ✅ | ❌ | ❌ |
| 通道配置 | ✅ | ❌ | ❌ |
| 技能安装 | ✅ | ❌ | ❌ |
| 插件安装 | ✅ | ❌ | ❌ |
| Chatbot 连接 | ✅ | ❌ | ❌ |
| 一键升级 | ✅ | ✅ | ✅ |
| 终端访问 | ✅ | ✅ | ✅ |
| WebUI 访问 | ✅ | ✅ | ✅ |
| 重启/重装 | ✅ | ✅ | ✅ |

## 2. 数据库设计

> **重要规范**（参考 `CLAUDE.md`）：
> - **双重维护要求**：修改 GORM 模型时，必须同步更新：
>   1. GORM model struct tags（SQLite 自动迁移）
>   2. `sql/init.sql` 中的 `CREATE TABLE` 语句（全新 MySQL 部署）
>   3. 增量迁移文件 `sql/XXXX-*.sql`（现有 MySQL 数据库升级）
> - **TEXT 列限制**：MySQL 不允许 `TEXT/BLOB` 列设置 `DEFAULT`，需要改用 `varchar(N)` 或在应用层处理默认值
> - 遗漏任何一步都视为 Bug

### 2.1 修改表：`ai_images`

```sql
-- 迁移脚本：sql/0415-add-image-agent-type.sql
-- 描述：为镜像表添加智能体类型和版本字段
ALTER TABLE `ai_images`
  ADD COLUMN `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体类型' AFTER `enabled`,
  ADD COLUMN `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体版本号' AFTER `agent_type`;

-- 添加索引：按类型+启用状态查询
CREATE INDEX idx_ai_images_agent_type_enabled ON ai_images(agent_type, enabled);
```

### 2.2 修改表：`instances`

```sql
-- 迁移脚本：sql/0415-add-instance-agent-type.sql
-- 描述：为实例表添加智能体类型和版本字段
ALTER TABLE `instances`
  ADD COLUMN `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'openclaw' COMMENT '智能体类型' AFTER `role_id`,
  ADD COLUMN `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体版本号' AFTER `agent_type`;

-- 添加索引
CREATE INDEX idx_instances_agent_type ON instances(agent_type);
CREATE INDEX idx_instances_user_agent_type ON instances(user_id, agent_type);
```

### 2.3 同步更新 init.sql

需要同步更新 `hatchery/sql/init.sql`，在相应表定义中添加新字段：

1. **修改 `ai_images` 表**：在 `enabled` 字段后添加：
   ```sql
   `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体类型',
   `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体版本号',
   ```
2. **修改 `instances` 表**：在 `role_id` 字段后添加：
   ```sql
   `agent_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'openclaw' COMMENT '智能体类型',
   `agent_version` varchar(64) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '智能体版本号',
   ```
3. **添加索引**：在各表的 KEY 定义部分添加相应索引

## 3. 模型层设计

### 3.1 新增模型：`agent_type.go`（硬编码版）

```go
// model/agent_type.go

package model

import (
    "regexp"
)

// ========== 智能体类型常量 ==========

const (
    AgentTypeOpenClaw     = "openclaw"
    AgentTypeHermes       = "hermes"
    AgentTypeLightclawACE = "lightclawace"
)

// AgentType 智能体类型配置（硬编码）
type AgentType struct {
    Code            string `json:"code"`
    Name            string `json:"name"`
    Description     string `json:"description"`
    SupportsRole    bool   `json:"supports_role"`
    SupportsModel   bool   `json:"supports_model"`
    SupportsChannel bool   `json:"supports_channel"`
    SupportsSkill   bool   `json:"supports_skill"`
    SupportsPlugin  bool   `json:"supports_plugin"`
    SupportsChatbot bool   `json:"supports_chatbot"`
    SortOrder       int    `json:"sort_order"`
}

// agentTypesMap 硬编码的智能体类型配置
var agentTypesMap = map[string]*AgentType{
    AgentTypeOpenClaw: {
        Code:            AgentTypeOpenClaw,
        Name:            "OpenClaw",
        Description:     "功能最完整的智能体类型，支持模型配置、通道配置、技能配置等",
        SupportsRole:    true,
        SupportsModel:   true,
        SupportsChannel: true,
        SupportsSkill:   true,
        SupportsPlugin:  true,
        SupportsChatbot: true,
        SortOrder:       1,
    },
    AgentTypeHermes: {
        Code:            AgentTypeHermes,
        Name:            "Hermes",
        Description:     "轻量级智能体，支持终端和 WebUI",
        SupportsRole:    false,
        SupportsModel:   false,
        SupportsChannel: false,
        SupportsSkill:   false,
        SupportsPlugin:  false,
        SupportsChatbot: false,
        SortOrder:       2,
    },
    AgentTypeLightclawACE: {
        Code:            AgentTypeLightclawACE,
        Name:            "LightclawACE",
        Description:     "轻量级智能体，支持终端和 WebUI",
        SupportsRole:    false,
        SupportsModel:   false,
        SupportsChannel: false,
        SupportsSkill:   false,
        SupportsPlugin:  false,
        SupportsChatbot: false,
        SortOrder:       3,
    },
}

// agentTypesList 按排序顺序排列的智能体类型列表
var agentTypesList = []*AgentType{
    agentTypesMap[AgentTypeOpenClaw],
    agentTypesMap[AgentTypeHermes],
    agentTypesMap[AgentTypeLightclawACE],
}

// 版本号校验正则
var agentVersionRegex = regexp.MustCompile(`^[a-zA-Z0-9][\w.\-]{0,62}[a-zA-Z0-9]$|^[a-zA-Z0-9]$|^$`)

// ========== 查询函数 ==========

// GetAgentTypeByCode 根据代码获取智能体类型
func GetAgentTypeByCode(code string) *AgentType {
    return agentTypesMap[code]
}

// GetAllAgentTypes 获取所有智能体类型（按排序顺序）
func GetAllAgentTypes() []*AgentType {
    return agentTypesList
}

// GetAllAgentTypesMap 获取所有智能体类型（Map 形式）
func GetAllAgentTypesMap() map[string]*AgentType {
    return agentTypesMap
}

// IsValidAgentType 校验智能体类型是否有效
func IsValidAgentType(agentType string) bool {
    if agentType == "" {
        return false
    }
    _, exists := agentTypesMap[agentType]
    return exists
}

// IsValidAgentVersion 校验版本号格式
func IsValidAgentVersion(v string) bool {
    return agentVersionRegex.MatchString(v)
}

// ========== 功能支持检查函数 ==========

// AgentTypeSupportsRole 检查类型是否支持角色配置
func AgentTypeSupportsRole(code string) bool {
    t := GetAgentTypeByCode(code)
    if t == nil {
        return false
    }
    return t.SupportsRole
}

// AgentTypeDetailConfigFlags 详细配置支持情况
type AgentTypeDetailConfigFlags struct {
    SupportsModel   bool `json:"supports_model"`
    SupportsChannel bool `json:"supports_channel"`
    SupportsSkill   bool `json:"supports_skill"`
    SupportsPlugin  bool `json:"supports_plugin"`
}

// GetAgentTypeDetailConfigFlags 获取类型的详细配置支持情况
func GetAgentTypeDetailConfigFlags(code string) *AgentTypeDetailConfigFlags {
    t := GetAgentTypeByCode(code)
    if t == nil {
        return nil
    }
    return &AgentTypeDetailConfigFlags{
        SupportsModel:   t.SupportsModel,
        SupportsChannel: t.SupportsChannel,
        SupportsSkill:   t.SupportsSkill,
        SupportsPlugin:  t.SupportsPlugin,
    }
}

// AgentTypeSupportsDetailConfig 检查类型是否支持详细配置（模型/通道/技能/插件中任意一项）
func AgentTypeSupportsDetailConfig(code string) bool {
    flags := GetAgentTypeDetailConfigFlags(code)
    if flags == nil {
        return false
    }
    return flags.SupportsModel || flags.SupportsChannel || flags.SupportsSkill || flags.SupportsPlugin
}

// AgentTypeSupportsChatbot 检查类型是否支持 Chatbot
func AgentTypeSupportsChatbot(code string) bool {
    t := GetAgentTypeByCode(code)
    if t == nil {
        return false
    }
    return t.SupportsChatbot
}

// AgentTypeSupportsSkill 检查类型是否支持技能安装
func AgentTypeSupportsSkill(code string) bool {
    t := GetAgentTypeByCode(code)
    if t == nil {
        return false
    }
    return t.SupportsSkill
}

// AgentTypeSupportsPlugin 检查类型是否支持插件安装
func AgentTypeSupportsPlugin(code string) bool {
    t := GetAgentTypeByCode(code)
    if t == nil {
        return false
    }
    return t.SupportsPlugin
}

// GetAgentTypeDisplayName 获取类型显示名称
func GetAgentTypeDisplayName(code string) string {
    t := GetAgentTypeByCode(code)
    if t == nil {
        return code // 类型不存在时返回原始 code
    }
    return t.Name
}
```

### 3.2 修改模型：`ai_image.go`

> **重要**：保留原有 `GetEnabledImage()` 函数，确保旧前端向后兼容
> 
> **新增依赖**：需要在 import 中添加 `"errors"`, `"fmt"`, `"gorm.io/gorm"`

```go
// model/ai_image.go 新增字段

type AIImage struct {
    // ... 原有字段 ...
    AgentType    string `gorm:"type:varchar(32);default:''" json:"agent_type"`
    AgentVersion string `gorm:"type:varchar(64);default:''" json:"agent_version"`
}

// ========== 向后兼容函数（保留原有） ==========

// GetEnabledImage 返回当前启用的镜像（全局，向后兼容）
// 旧前端调用时返回任意一个启用的镜像
// 注意：此函数保持原有行为，不修改
func GetEnabledImage() *AIImage {
    var img AIImage
    if DB.Where("enabled = ?", true).First(&img).Error != nil {
        return nil
    }
    return &img
}

// ========== 新增函数 ==========

// GetEnabledImageByType 获取指定类型的启用镜像
// 返回值：(*AIImage, error)
//   - 找到镜像：返回 (&img, nil)
//   - 镜像不存在：返回 (nil, nil)
//   - 数据库错误：返回 (nil, err)
func GetEnabledImageByType(agentType string) (*AIImage, error) {
    var img AIImage
    // 优先精确匹配类型
    err := DB.Where("agent_type = ? AND enabled = ?", agentType, true).First(&img).Error
    if err == nil {
        return &img, nil
    }
    if !errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, fmt.Errorf("query enabled image by type failed: %w", err)
    }
    
    // 兼容：回退到空类型的镜像（旧数据）
    err = DB.Where("(agent_type = '' OR agent_type IS NULL) AND enabled = ?", true).First(&img).Error
    if err == nil {
        return &img, nil
    }
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil // 没有可用镜像
    }
    return nil, fmt.Errorf("query fallback enabled image failed: %w", err)
}

// GetEnabledImagesMap 批量获取所有类型的启用镜像
func GetEnabledImagesMap() (map[string]*AIImage, error) {
    var images []AIImage
    if err := DB.Where("enabled = ?", true).Find(&images).Error; err != nil {
        return nil, fmt.Errorf("query enabled images failed: %w", err)
    }
    
    result := make(map[string]*AIImage)
    for i := range images {
        img := &images[i]
        if img.AgentType != "" {
            result[img.AgentType] = img
        }
    }
    return result, nil
}
```

### 3.3 修改模型：`instance.go`

```go
// model/instance.go 新增字段

type Instance struct {
    // ... 原有字段 ...
    AgentType    string `gorm:"type:varchar(32);default:'openclaw'" json:"agent_type"`
    AgentVersion string `gorm:"type:varchar(64);default:''" json:"agent_version"`
}
```

## 4. 后端 API 设计（RESTful 规范）

### 4.1 智能体类型 API（只读，硬编码版）

由于类型是硬编码的，只提供查询接口，不提供 CRUD 操作：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/admin/agent-types` | 获取智能体类型列表（只读） |

#### 4.1.1 GET /admin/agent-types

```json
// 响应
{
  "agent_types": [
    {
      "code": "openclaw",
      "name": "OpenClaw",
      "description": "功能最完整的智能体类型",
      "supports_role": true,
      "supports_model": true,
      "supports_channel": true,
      "supports_skill": true,
      "supports_plugin": true,
      "supports_chatbot": true,
      "sort_order": 1,
      "has_enabled_image": true,
      "enabled_image": {
        "id": 1,
        "image_name": "OpenClaw v2026.4.2",
        "agent_version": "2026.4.2"
      }
    },
    {
      "code": "hermes",
      "name": "Hermes",
      "description": "轻量级智能体，支持终端和 WebUI",
      "supports_role": false,
      "supports_model": false,
      "supports_channel": false,
      "supports_skill": false,
      "supports_plugin": false,
      "supports_chatbot": false,
      "sort_order": 2,
      "has_enabled_image": false,
      "enabled_image": null
    }
  ]
}
```

### 4.2 镜像管理 API（修改）

#### 4.2.1 GET /admin/images

响应新增字段：`agent_type`, `agent_version`

#### 4.2.2 POST /admin/images/import

请求新增可选字段：
```json
{
  "image_id": "img-xxx",
  "agent_type": "openclaw",
  "agent_version": "2026.4.2"
}
```

#### 4.2.3 POST /admin/images/update（新增）

```json
// 请求
{
  "id": 1,
  "agent_type": "openclaw",
  "agent_version": "2026.4.2"
}
```

#### 4.2.4 POST /admin/images/enable

修改逻辑：启用镜像时，仅禁用同类型的其他镜像。

### 4.3 实例管理 API（修改）

#### 4.3.1 GET /openclaw/list

响应新增字段：`agent_type`, `agent_version`

#### 4.3.2 POST /openclaw/create

请求新增可选字段：
```json
{
  "name": "我的智能体",
  "role_id": 1,
  "agent_type": "openclaw"  // 默认为 "openclaw"
}
```

#### 4.3.3 GET /admin/instances

响应新增统计：
```json
{
  "instances": [...],
  "stats": {
    "total": 100,
    "by_agent_type": {
      "openclaw": 70,
      "hermes": 20,
      "lightclawace": 10
    }
  }
}
```

## 5. 后端防护设计

### 5.1 防护检查函数

```go
// controller/agent_type_guard.go

package controller

import (
    "context"
    "fmt"
    "log/slog"
    
    "hatchery/model"
)

// checkAgentTypeValid 校验智能体类型有效性
func checkAgentTypeValid(agentType string) error {
    if agentType == "" {
        return nil // 允许空值（兼容）
    }
    if !model.IsValidAgentType(agentType) {
        return fmt.Errorf("无效的智能体类型: %s", agentType)
    }
    return nil
}

// checkAgentVersionValid 校验版本号格式
func checkAgentVersionValid(version string) error {
    if !model.IsValidAgentVersion(version) {
        return fmt.Errorf("无效的版本号格式: %s", version)
    }
    return nil
}

// checkInstanceSupportsDetailConfig 校验实例是否支持详细配置
func checkInstanceSupportsDetailConfig(ctx context.Context, instance *model.Instance) error {
    if !model.AgentTypeSupportsDetailConfig(instance.AgentType) {
        typeName := model.GetAgentTypeDisplayName(instance.AgentType)
        slog.WarnContext(ctx, "[Guard] 实例类型不支持详细配置",
            "instance_id", instance.ID,
            "agent_type", instance.AgentType)
        return fmt.Errorf("%s 类型实例不支持此配置", typeName)
    }
    return nil
}

// checkInstanceSupportsChatbot 校验实例是否支持 Chatbot
func checkInstanceSupportsChatbot(ctx context.Context, instance *model.Instance) error {
    if !model.AgentTypeSupportsChatbot(instance.AgentType) {
        typeName := model.GetAgentTypeDisplayName(instance.AgentType)
        slog.WarnContext(ctx, "[Guard] 实例类型不支持 Chatbot",
            "instance_id", instance.ID,
            "agent_type", instance.AgentType)
        return fmt.Errorf("%s 类型实例不支持 Chatbot 功能", typeName)
    }
    return nil
}
```

### 5.2 实例创建防护

```go
// controller/openclaw.go - HandleCreateInstance 修改

func HandleCreateInstance(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    // ... 原有用户获取逻辑 ...
    
    // 解析 agent_type 参数
    agentType := strings.TrimSpace(r.FormValue("agent_type"))
    if agentType == "" {
        agentType = model.AgentTypeOpenClaw
    }
    
    // 校验 agent_type（硬编码白名单校验）
    if err := checkAgentTypeValid(agentType); err != nil {
        slog.WarnContext(ctx, "[CreateInstance] 无效的智能体类型", 
            "agent_type", agentType, "user", user.Username)
        writeError(w, r, http.StatusBadRequest, err)
        return
    }
    
    // 【关键防护】非支持角色的类型，强制忽略 role_id
    roleID := parseUint(r.FormValue("role_id"))
    if !model.AgentTypeSupportsRole(agentType) && roleID > 0 {
        slog.InfoContext(ctx, "[CreateInstance] 非角色支持类型，忽略 role_id",
            "agent_type", agentType, "role_id", roleID)
        roleID = 0
    }
    
    // 根据 agent_type 获取对应的启用镜像
    enabledImage, err := model.GetEnabledImageByType(agentType)
    if err != nil {
        slog.ErrorContext(ctx, "[CreateInstance] 查询启用镜像失败",
            "agent_type", agentType, "error", err)
        writeError(w, r, http.StatusInternalServerError, fmt.Errorf("查询镜像失败"))
        return
    }
    if enabledImage == nil {
        typeName := model.GetAgentTypeDisplayName(agentType)
        slog.WarnContext(ctx, "[CreateInstance] 未找到该类型的启用镜像",
            "agent_type", agentType)
        writeError(w, r, http.StatusBadRequest,
            fmt.Errorf("管理员尚未为 %s 类型配置生效镜像，请联系管理员", typeName))
        return
    }
    
    // 从镜像获取 agent_version
    agentVersion := enabledImage.AgentVersion
    
    // ... 后续创建逻辑，使用 agentType 和 agentVersion ...
}
```

### 5.3 配置接口防护（模型/通道/技能/插件）

```go
// controller/openclaw.go - HandleSetModel 修改

func HandleSetModel(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    // ... 获取实例 ...
    
    // 【关键防护】校验实例是否支持模型配置
    if err := checkInstanceSupportsDetailConfig(ctx, &instance); err != nil {
        writeError(w, r, http.StatusForbidden, err)
        return
    }
    
    // ... 原有逻辑 ...
}

// HandleAddSkill, HandleAddPlugin, HandleSetChannel 同理
```

### 5.4 升级/重装防护

```go
// controller/openclaw_upgrade.go - HandleUpgrade 修改

func HandleUpgrade(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    // ... 获取实例 ...
    
    // 【关键修改】根据实例的 agent_type 获取对应镜像
    defaultImage, err := model.GetEnabledImageByType(instance.AgentType)
    if err != nil {
        slog.ErrorContext(ctx, "[Upgrade] 查询启用镜像失败",
            "instance_id", instance.ID, "agent_type", instance.AgentType, "error", err)
        writeError(w, r, http.StatusInternalServerError, fmt.Errorf("查询镜像失败"))
        return
    }
    if defaultImage == nil {
        typeName := model.GetAgentTypeDisplayName(instance.AgentType)
        slog.ErrorContext(ctx, "[Upgrade] 未找到该类型已启用的镜像",
            "instance_id", instance.ID, "agent_type", instance.AgentType)
        writeError(w, r, http.StatusInternalServerError,
            fmt.Errorf("管理员尚未为 %s 类型配置生效镜像", typeName))
        return
    }
    
    // ... 使用 defaultImage 进行升级 ...
}
```

### 5.5 初始技能/插件安装防护

```go
// controller/openclaw.go - createSkillInstallTasks 修改

func createSkillInstallTasks(instanceID uint, roleID uint) {
    ctx := context.Background()
    
    // 【关键防护】获取实例的 agent_type
    var instance model.Instance
    if model.DB.First(&instance, instanceID).Error != nil {
        return
    }
    
    // 非技能支持类型，跳过技能安装
    if !model.AgentTypeSupportsSkill(instance.AgentType) {
        slog.InfoContext(ctx, "[Skill] 跳过技能安装，该类型不支持",
            "instance_id", instanceID, "agent_type", instance.AgentType)
        return
    }
    
    // ... 原有逻辑 ...
}
```

### 5.6 镜像启用逻辑

```go
// controller/admin_images.go - HandleEnableImage 修改

func HandleEnableImage(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    // ... 获取镜像 ...
    
    if img.Enabled {
        // 禁用该镜像
        model.DB.Model(&img).Update("enabled", false)
        slog.InfoContext(ctx, "[Image] 禁用镜像", "image_id", img.ImageId)
    } else {
        // 【关键修改】启用镜像时，仅禁用同类型的其他镜像
        if img.AgentType != "" {
            // 禁用同类型的其他已启用镜像
            result := model.DB.Model(&model.AIImage{}).
                Where("agent_type = ? AND enabled = ? AND id != ?", 
                    img.AgentType, true, img.ID).
                Update("enabled", false)
            if result.RowsAffected > 0 {
                slog.InfoContext(ctx, "[Image] 禁用同类型其他镜像",
                    "agent_type", img.AgentType, "count", result.RowsAffected)
            }
        } else {
            // 兼容：未设置类型的镜像，禁用所有已启用镜像
            model.DB.Model(&model.AIImage{}).
                Where("enabled = ?", true).
                Update("enabled", false)
        }
        model.DB.Model(&img).Update("enabled", true)
        slog.InfoContext(ctx, "[Image] 启用镜像",
            "image_id", img.ImageId, "agent_type", img.AgentType)
    }
    // ...
}

// HandleUpdateImage 更新镜像的智能体类型和版本
func HandleUpdateImage(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    if r.Method != http.MethodPost {
        writeError(w, r, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
        return
    }
    
    id := parseUint(r.FormValue("id"))
    if id == 0 {
        writeError(w, r, http.StatusBadRequest, fmt.Errorf("id 不能为空"))
        return
    }
    
    var img model.AIImage
    if err := model.DB.First(&img, id).Error; err != nil {
        writeError(w, r, http.StatusNotFound, fmt.Errorf("镜像不存在"))
        return
    }
    
    updates := map[string]interface{}{}
    
    // 更新智能体类型
    if agentType := strings.TrimSpace(r.FormValue("agent_type")); agentType != "" {
        if err := checkAgentTypeValid(agentType); err != nil {
            writeError(w, r, http.StatusBadRequest, err)
            return
        }
        updates["agent_type"] = agentType
    }
    
    // 更新版本号
    if agentVersion := strings.TrimSpace(r.FormValue("agent_version")); agentVersion != "" {
        if err := checkAgentVersionValid(agentVersion); err != nil {
            writeError(w, r, http.StatusBadRequest, err)
            return
        }
        updates["agent_version"] = agentVersion
    }
    
    if len(updates) == 0 {
        writeError(w, r, http.StatusBadRequest, fmt.Errorf("没有需要更新的字段"))
        return
    }
    
    if err := model.DB.Model(&img).Updates(updates).Error; err != nil {
        slog.ErrorContext(ctx, "[Image] 更新镜像失败", "id", id, "error", err)
        writeError(w, r, http.StatusInternalServerError, fmt.Errorf("更新失败"))
        return
    }
    
    slog.InfoContext(ctx, "[Image] 更新镜像成功",
        "id", id, "image_id", img.ImageId, "updates", updates)
    
    writeJSON(w, map[string]interface{}{
        "success": true,
    })
}
```

### 5.7 智能体类型查询控制器（只读）

```go
// controller/admin_agent_types.go

package controller

import (
    "fmt"
    "log/slog"
    "net/http"
    
    "hatchery/model"
)

// AgentTypeResponse 智能体类型响应（包含启用镜像信息）
type AgentTypeResponse struct {
    Code            string            `json:"code"`
    Name            string            `json:"name"`
    Description     string            `json:"description"`
    SupportsRole    bool              `json:"supports_role"`
    SupportsModel   bool              `json:"supports_model"`
    SupportsChannel bool              `json:"supports_channel"`
    SupportsSkill   bool              `json:"supports_skill"`
    SupportsPlugin  bool              `json:"supports_plugin"`
    SupportsChatbot bool              `json:"supports_chatbot"`
    SortOrder       int               `json:"sort_order"`
    HasEnabledImage bool              `json:"has_enabled_image"`
    EnabledImage    *EnabledImageInfo `json:"enabled_image,omitempty"`
}

// EnabledImageInfo 启用镜像简要信息
type EnabledImageInfo struct {
    ID           uint   `json:"id"`
    ImageName    string `json:"image_name"`
    AgentVersion string `json:"agent_version"`
}

// HandleAdminAgentTypes 获取智能体类型列表（只读）
func HandleAdminAgentTypes(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    if r.Method != http.MethodGet {
        writeError(w, r, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
        return
    }
    
    // 从硬编码配置获取类型列表
    types := model.GetAllAgentTypes()
    
    // 批量获取各类型的启用镜像
    imagesMap, err := model.GetEnabledImagesMap()
    if err != nil {
        slog.ErrorContext(ctx, "[AgentTypes] 查询启用镜像失败", "error", err)
        writeError(w, r, http.StatusInternalServerError, fmt.Errorf("查询镜像失败"))
        return
    }
    
    // 构建响应
    var responses []AgentTypeResponse
    for _, t := range types {
        resp := AgentTypeResponse{
            Code:            t.Code,
            Name:            t.Name,
            Description:     t.Description,
            SupportsRole:    t.SupportsRole,
            SupportsModel:   t.SupportsModel,
            SupportsChannel: t.SupportsChannel,
            SupportsSkill:   t.SupportsSkill,
            SupportsPlugin:  t.SupportsPlugin,
            SupportsChatbot: t.SupportsChatbot,
            SortOrder:       t.SortOrder,
        }
        if img, ok := imagesMap[t.Code]; ok && img != nil {
            resp.HasEnabledImage = true
            resp.EnabledImage = &EnabledImageInfo{
                ID:           img.ID,
                ImageName:    img.ImageName,
                AgentVersion: img.AgentVersion,
            }
        }
        responses = append(responses, resp)
    }
    
    writeJSON(w, map[string]interface{}{
        "agent_types": responses,
    })
}
```

## 6. 日志规范

### 6.1 日志级别使用规范

| 级别 | 使用场景 |
|------|---------|
| `slog.ErrorContext` | 系统错误、数据库错误、外部服务调用失败 |
| `slog.WarnContext` | 业务校验失败（如类型不支持）、潜在问题 |
| `slog.InfoContext` | 关键业务操作（创建/删除/更新）、状态变更 |
| `slog.DebugContext` | 调试信息、详细流程追踪 |

### 6.2 日志字段规范

```go
// 统一使用以下字段格式
slog.InfoContext(ctx, "[模块] 操作描述",
    "instance_id", instance.ID,        // 实例 ID
    "agent_type", instance.AgentType,  // 智能体类型
    "user", user.Username,             // 操作用户
    "image_id", img.ImageId,           // 镜像 ID
    "error", err,                      // 错误信息（仅 Error/Warn 级别）
)
```

### 6.3 日志示例

```go
// 实例创建成功
slog.InfoContext(ctx, "[CreateInstance] 创建实例成功",
    "instance_id", instance.ID,
    "name", instance.Name,
    "agent_type", instance.AgentType,
    "agent_version", instance.AgentVersion,
    "user", user.Username)

// 类型校验失败
slog.WarnContext(ctx, "[CreateInstance] 无效的智能体类型",
    "agent_type", agentType,
    "user", user.Username)

// 防护触发
slog.WarnContext(ctx, "[SetModel] 实例类型不支持模型配置",
    "instance_id", instance.ID,
    "agent_type", instance.AgentType,
    "user", user.Username)

// 数据库错误
slog.ErrorContext(ctx, "[GetEnabledImageByType] 查询镜像失败",
    "agent_type", agentType,
    "error", err)
```

## 7. 安全性设计

### 7.1 输入校验

```go
// 智能体类型校验（硬编码白名单）
if !model.IsValidAgentType(agentType) {
    return fmt.Errorf("无效的智能体类型: %s", agentType)
}

// 版本号格式校验（正则）
if !model.IsValidAgentVersion(version) {
    return fmt.Errorf("无效的版本号格式: %s", version)
}
```

### 7.2 SQL 注入防护

所有数据库操作使用 GORM 参数化查询：
```go
// 正确
DB.Where("agent_type = ? AND enabled = ?", agentType, true).First(&img)

// 禁止
DB.Raw("SELECT * FROM ai_images WHERE agent_type = '" + agentType + "'")
```

### 7.3 权限校验

- 智能体类型查询接口：仅管理员可访问（使用 `requireAdmin` 中间件）
- 镜像管理接口：仅管理员可访问
- 实例创建接口：普通用户可访问，但受配额限制

## 8. 性能优化

### 8.1 数据库索引

```sql
-- ai_images 表索引
CREATE INDEX idx_ai_images_agent_type_enabled ON ai_images(agent_type, enabled);

-- instances 表索引
CREATE INDEX idx_instances_agent_type ON instances(agent_type);
CREATE INDEX idx_instances_user_agent_type ON instances(user_id, agent_type);
```

### 8.2 硬编码优势

由于智能体类型配置是硬编码的：
- **无需数据库查询**：类型校验直接在内存中完成，O(1) 时间复杂度
- **无需缓存**：配置已在内存中，无需额外缓存机制
- **启动时无需初始化**：不需要在应用启动时从数据库加载配置

## 9. 审计日志设计

> **重要规范**（参考 `CLAUDE.md`）：
> - **每个新增的写操作端点（POST/PUT/DELETE）必须有审计日志**
> - 实现步骤：
>   1. 在 `controller/audit.go` 的 `auditRules` map 中添加规则条目
>   2. 在 `main.go` 路由注册时使用 `WithAudit()` 包装 handler
> - 遗漏审计日志视为 Bug

### 9.1 新增审计规则

在 `controller/audit.go` 的 `auditRules` map 中添加：

```go
// controller/audit.go - auditRules map 新增条目

// 镜像管理（补充）
"/admin/images/update": {Action: "update", Resource: "image"},
```

> **注意**：由于智能体类型是硬编码的，不需要 CRUD 接口，因此不需要添加智能体类型的审计规则。

### 9.2 路由注册（WithAudit 包装）

```go
// main.go - 路由注册

// 智能体类型查询（只读，不需要审计）
mux.HandleFunc("/admin/agent-types", controller.RequireAdmin(controller.HandleAdminAgentTypes))

// 镜像更新接口
mux.HandleFunc("/admin/images/update", controller.WithAudit(controller.RequireAdmin(controller.HandleUpdateImage)))
```

### 9.3 审计日志字段

审计日志应记录以下信息：

| 字段 | 说明 | 示例 |
|------|------|------|
| `action` | 操作类型 | update |
| `resource` | 资源类型 | image |
| `resource_id` | 资源 ID | 1 |
| `user` | 操作用户 | admin |
| `ip` | 客户端 IP | 192.168.1.1 |
| `timestamp` | 操作时间 | 2026-04-15T10:30:00Z |
| `details` | 变更详情（可选） | {"agent_type": "hermes"} |

## 10. 单元测试设计

### 10.1 模型层测试

```go
// model/agent_type_test.go

package model

import (
    "testing"
)

func TestIsValidAgentType(t *testing.T) {
    tests := []struct {
        name      string
        agentType string
        expected  bool
    }{
        {"valid openclaw", "openclaw", true},
        {"valid hermes", "hermes", true},
        {"valid lightclawace", "lightclawace", true},
        {"invalid type", "unknown", false},
        {"empty string", "", false},
        {"sql injection attempt", "openclaw'; DROP TABLE--", false},
        {"case sensitive", "OpenClaw", false}, // 区分大小写
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := IsValidAgentType(tt.agentType)
            if result != tt.expected {
                t.Errorf("IsValidAgentType(%q) = %v, want %v", tt.agentType, result, tt.expected)
            }
        })
    }
}

func TestIsValidAgentVersion(t *testing.T) {
    tests := []struct {
        name     string
        version  string
        expected bool
    }{
        {"valid semver", "1.0.0", true},
        {"valid date version", "2026.4.2", true},
        {"valid with suffix", "1.0.0-beta", true},
        {"empty allowed", "", true},
        {"single char", "1", true},
        {"too long", string(make([]byte, 100)), false},
        {"invalid start", "-1.0.0", false},
        {"script injection", "1.0<script>", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := IsValidAgentVersion(tt.version)
            if result != tt.expected {
                t.Errorf("IsValidAgentVersion(%q) = %v, want %v", tt.version, result, tt.expected)
            }
        })
    }
}

func TestAgentTypeSupportsRole(t *testing.T) {
    tests := []struct {
        code     string
        expected bool
    }{
        {"openclaw", true},
        {"hermes", false},
        {"lightclawace", false},
        {"unknown", false}, // 未知类型返回 false
    }
    
    for _, tt := range tests {
        t.Run(tt.code, func(t *testing.T) {
            result := AgentTypeSupportsRole(tt.code)
            if result != tt.expected {
                t.Errorf("AgentTypeSupportsRole(%s) = %v, want %v", tt.code, result, tt.expected)
            }
        })
    }
}

func TestAgentTypeSupportsDetailConfig(t *testing.T) {
    result := AgentTypeSupportsDetailConfig("openclaw")
    if !result {
        t.Error("OpenClaw should support detail config")
    }
    
    result = AgentTypeSupportsDetailConfig("hermes")
    if result {
        t.Error("Hermes should not support detail config")
    }
    
    result = AgentTypeSupportsDetailConfig("lightclawace")
    if result {
        t.Error("LightclawACE should not support detail config")
    }
}

func TestGetAgentTypeDetailConfigFlags(t *testing.T) {
    // 测试 OpenClaw - 支持所有配置
    flags := GetAgentTypeDetailConfigFlags("openclaw")
    if flags == nil {
        t.Fatal("flags should not be nil for openclaw")
    }
    if !flags.SupportsModel || !flags.SupportsChannel || !flags.SupportsSkill || !flags.SupportsPlugin {
        t.Error("OpenClaw should support all detail configs")
    }
    
    // 测试 Hermes - 不支持任何配置
    flags = GetAgentTypeDetailConfigFlags("hermes")
    if flags == nil {
        t.Fatal("flags should not be nil for hermes")
    }
    if flags.SupportsModel || flags.SupportsChannel || flags.SupportsSkill || flags.SupportsPlugin {
        t.Error("Hermes should not support any detail config")
    }
    
    // 测试不存在的类型
    flags = GetAgentTypeDetailConfigFlags("nonexistent")
    if flags != nil {
        t.Error("expected nil for nonexistent agent type")
    }
}

func TestAgentTypeSupportsChatbot(t *testing.T) {
    result := AgentTypeSupportsChatbot("openclaw")
    if !result {
        t.Error("OpenClaw should support chatbot")
    }
    
    result = AgentTypeSupportsChatbot("hermes")
    if result {
        t.Error("Hermes should not support chatbot")
    }
}

func TestGetAllAgentTypes(t *testing.T) {
    types := GetAllAgentTypes()
    if len(types) != 3 {
        t.Errorf("expected 3 agent types, got %d", len(types))
    }
    
    // 验证排序
    if types[0].Code != "openclaw" {
        t.Error("first type should be openclaw")
    }
    if types[1].Code != "hermes" {
        t.Error("second type should be hermes")
    }
    if types[2].Code != "lightclawace" {
        t.Error("third type should be lightclawace")
    }
}

func TestGetAgentTypeByCode(t *testing.T) {
    // 测试存在的类型
    t1 := GetAgentTypeByCode("openclaw")
    if t1 == nil || t1.Name != "OpenClaw" {
        t.Error("should return OpenClaw type")
    }
    
    // 测试不存在的类型
    t2 := GetAgentTypeByCode("nonexistent")
    if t2 != nil {
        t.Error("should return nil for nonexistent type")
    }
}

func TestGetAgentTypeDisplayName(t *testing.T) {
    name := GetAgentTypeDisplayName("openclaw")
    if name != "OpenClaw" {
        t.Errorf("expected 'OpenClaw', got '%s'", name)
    }
    
    name = GetAgentTypeDisplayName("hermes")
    if name != "Hermes" {
        t.Errorf("expected 'Hermes', got '%s'", name)
    }
    
    // 不存在的类型返回原始 code
    name = GetAgentTypeDisplayName("unknown")
    if name != "unknown" {
        t.Errorf("expected 'unknown', got '%s'", name)
    }
}
```

### 10.2 镜像管理测试

```go
// model/ai_image_test.go

package model

import (
    "testing"
    
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func setupTestDB(t *testing.T) {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to open test db: %v", err)
    }
    db.AutoMigrate(&AIImage{}, &Instance{})
    DB = db
}

func TestGetEnabledImageByType(t *testing.T) {
    setupTestDB(t)
    
    // 准备测试数据
    DB.Create(&AIImage{ImageId: "img-openclaw", AgentType: "openclaw", Enabled: true})
    DB.Create(&AIImage{ImageId: "img-hermes", AgentType: "hermes", Enabled: true})
    DB.Create(&AIImage{ImageId: "img-disabled", AgentType: "openclaw", Enabled: false})
    
    // 测试获取 OpenClaw 镜像
    img, err := GetEnabledImageByType("openclaw")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if img == nil || img.ImageId != "img-openclaw" {
        t.Error("should return openclaw image")
    }
    
    // 测试获取 Hermes 镜像
    img, err = GetEnabledImageByType("hermes")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if img == nil || img.ImageId != "img-hermes" {
        t.Error("should return hermes image")
    }
    
    // 测试获取不存在类型的镜像
    img, err = GetEnabledImageByType("lightclawace")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if img != nil {
        t.Error("should return nil for type without enabled image")
    }
}

func TestGetEnabledImageByType_Fallback(t *testing.T) {
    setupTestDB(t)
    
    // 只有一个没有设置类型的启用镜像（兼容旧数据）
    DB.Create(&AIImage{ImageId: "img-legacy", AgentType: "", Enabled: true})
    
    // 任何类型都应该回退到这个镜像
    img, err := GetEnabledImageByType("openclaw")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if img == nil || img.ImageId != "img-legacy" {
        t.Error("should fallback to legacy image")
    }
}

func TestGetEnabledImagesMap(t *testing.T) {
    setupTestDB(t)
    
    DB.Create(&AIImage{ImageId: "img-openclaw", AgentType: "openclaw", Enabled: true})
    DB.Create(&AIImage{ImageId: "img-hermes", AgentType: "hermes", Enabled: true})
    
    imagesMap, err := GetEnabledImagesMap()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if len(imagesMap) != 2 {
        t.Errorf("expected 2 images, got %d", len(imagesMap))
    }
    if imagesMap["openclaw"] == nil || imagesMap["openclaw"].ImageId != "img-openclaw" {
        t.Error("should contain openclaw image")
    }
    if imagesMap["hermes"] == nil || imagesMap["hermes"].ImageId != "img-hermes" {
        t.Error("should contain hermes image")
    }
}
```

### 10.3 控制器层测试

```go
// controller/agent_type_guard_test.go

package controller

import (
    "context"
    "testing"
    
    "hatchery/model"
    
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

func setupTestDB(t *testing.T) {
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("failed to open test db: %v", err)
    }
    db.AutoMigrate(&model.AIImage{}, &model.Instance{})
    model.DB = db
}

func TestCheckAgentTypeValid(t *testing.T) {
    // 有效类型
    if err := checkAgentTypeValid("openclaw"); err != nil {
        t.Errorf("openclaw should be valid: %v", err)
    }
    
    if err := checkAgentTypeValid("hermes"); err != nil {
        t.Errorf("hermes should be valid: %v", err)
    }
    
    // 空值允许
    if err := checkAgentTypeValid(""); err != nil {
        t.Errorf("empty should be allowed: %v", err)
    }
    
    // 无效类型
    if err := checkAgentTypeValid("invalid"); err == nil {
        t.Error("invalid type should return error")
    }
}

func TestCheckInstanceSupportsDetailConfig(t *testing.T) {
    setupTestDB(t)
    ctx := context.Background()
    
    // OpenClaw 实例应该支持
    openclawInstance := &model.Instance{AgentType: "openclaw"}
    if err := checkInstanceSupportsDetailConfig(ctx, openclawInstance); err != nil {
        t.Errorf("openclaw should support detail config: %v", err)
    }
    
    // Hermes 实例不应该支持
    hermesInstance := &model.Instance{AgentType: "hermes"}
    if err := checkInstanceSupportsDetailConfig(ctx, hermesInstance); err == nil {
        t.Error("hermes should not support detail config")
    }
}

func TestCheckInstanceSupportsChatbot(t *testing.T) {
    setupTestDB(t)
    ctx := context.Background()
    
    // OpenClaw 实例应该支持
    openclawInstance := &model.Instance{AgentType: "openclaw"}
    if err := checkInstanceSupportsChatbot(ctx, openclawInstance); err != nil {
        t.Errorf("openclaw should support chatbot: %v", err)
    }
    
    // Hermes 实例不应该支持
    hermesInstance := &model.Instance{AgentType: "hermes"}
    if err := checkInstanceSupportsChatbot(ctx, hermesInstance); err == nil {
        t.Error("hermes should not support chatbot")
    }
}
```

### 10.4 镜像启用逻辑测试

```go
// controller/admin_images_test.go

package controller

import (
    "testing"
    
    "hatchery/model"
)

func TestEnableImage_OnlyOnePerType(t *testing.T) {
    setupTestDB(t)
    
    // 创建两个 OpenClaw 类型镜像
    img1 := model.AIImage{ImageId: "img-1", AgentType: "openclaw", Enabled: false}
    img2 := model.AIImage{ImageId: "img-2", AgentType: "openclaw", Enabled: false}
    model.DB.Create(&img1)
    model.DB.Create(&img2)
    
    // 启用 img1
    model.DB.Model(&img1).Update("enabled", true)
    
    // 启用 img2，模拟 HandleEnableImage 逻辑
    model.DB.Model(&model.AIImage{}).
        Where("agent_type = ? AND enabled = ? AND id != ?", "openclaw", true, img2.ID).
        Update("enabled", false)
    model.DB.Model(&img2).Update("enabled", true)
    
    // 验证
    var updated1, updated2 model.AIImage
    model.DB.First(&updated1, img1.ID)
    model.DB.First(&updated2, img2.ID)
    
    if updated1.Enabled {
        t.Error("img1 should be disabled after enabling img2")
    }
    if !updated2.Enabled {
        t.Error("img2 should be enabled")
    }
}

func TestEnableImage_DifferentTypesIndependent(t *testing.T) {
    setupTestDB(t)
    
    // 创建不同类型的镜像
    openclawImg := model.AIImage{ImageId: "img-openclaw", AgentType: "openclaw", Enabled: true}
    hermesImg := model.AIImage{ImageId: "img-hermes", AgentType: "hermes", Enabled: false}
    model.DB.Create(&openclawImg)
    model.DB.Create(&hermesImg)
    
    // 启用 Hermes 镜像，模拟 HandleEnableImage 逻辑
    model.DB.Model(&model.AIImage{}).
        Where("agent_type = ? AND enabled = ? AND id != ?", "hermes", true, hermesImg.ID).
        Update("enabled", false)
    model.DB.Model(&hermesImg).Update("enabled", true)
    
    // 验证 OpenClaw 镜像仍然启用
    var updated1, updated2 model.AIImage
    model.DB.First(&updated1, openclawImg.ID)
    model.DB.First(&updated2, hermesImg.ID)
    
    if !updated1.Enabled {
        t.Error("openclaw image should still be enabled")
    }
    if !updated2.Enabled {
        t.Error("hermes image should be enabled")
    }
}
```

## 11. 前端设计（概要）

### 11.1 类型定义

```typescript
// types/api.ts

export interface AgentType {
  code: string;
  name: string;
  description: string;
  supports_role: boolean;
  supports_model: boolean;
  supports_channel: boolean;
  supports_skill: boolean;
  supports_plugin: boolean;
  supports_chatbot: boolean;
  sort_order: number;
  has_enabled_image?: boolean;
  enabled_image?: {
    id: number;
    image_name: string;
    agent_version: string;
  } | null;
}

export interface ImageInfo {
  // ... 原有字段
  agent_type: string;
  agent_version: string;
}

export interface InstanceInfo {
  // ... 原有字段
  agent_type: string;
  agent_version: string;
}
```

### 11.2 页面修改清单

| 页面 | 修改内容 |
|------|---------|
| 镜像管理 | 新增"智能体类型"、"版本"列；支持编辑类型和版本 |
| 模型配置 | 顶部增加提示：仅对 OpenClaw 类型生效 |
| 通道配置 | 顶部增加提示：仅对 OpenClaw 类型生效 |
| 技能配置 | 初始技能包 Tab 增加提示 |
| 创建实例 | 新增智能体类型选择，非 OpenClaw 隐藏角色选择 |
| 实例列表 | 按类型 Tab 分组；非 OpenClaw 隐藏配置按钮 |
| 对话视图 | 非 OpenClaw 类型实例置灰不可选 |

## 12. 开发顺序（1天并发交付版）

> **核心原则**：
> 1. **向后兼容优先** - 后端升级过程中，旧版前端必须能正常工作
> 2. **先有镜像，才能创建实例** - 镜像管理必须先完成
> 3. **先有防护，才能开放创建** - 防护逻辑必须在实例创建之前完成
> 4. **增量开发** - 优先复用现有代码，仅做必要的增量修改

---

### 开发顺序（并发最大化）

```
阶段0：数据库迁移脚本（阻塞点，必须先完成）
  └── sql/0415-add-image-agent-type.sql  
  └── sql/0415-add-instance-agent-type.sql
  └── sql/init.sql 同步
      │
      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 阶段1：模型层（可并发）                                           │
│                                                                   │
│  [开发者A]                    [开发者B]                           │
│  model/agent_type.go          model/ai_image.go                  │
│  - AgentType 结构体（硬编码）   - 新增字段                         │
│  - 常量定义                    - GetEnabledImageByType()          │
│  - GetAgentTypeByCode()        - GetEnabledImagesMap()            │
│  - GetAllAgentTypes()          - 保留 GetEnabledImage()           │
│  - IsValidAgentType()                                              │
│  - GetAgentTypeDetailConfigFlags()    [开发者C]                   │
│                                       model/instance.go           │
│                                       - 新增字段                  │
└─────────────────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 阶段2：后端 Controller（可并发）                                  │
│                                                                   │
│  [开发者A]                    [开发者B]                           │
│  controller/admin_images.go   controller/agent_type_guard.go     │
│  - 列表增加字段                - checkAgentTypeValid()            │
│  - 导入支持类型参数            - checkInstanceSupportsDetailConfig()│
│  - HandleUpdateImage                                              │
│  - HandleEnableImage 修改      [开发者C]                          │
│                                controller/openclaw.go             │
│  [开发者D]                     - HandleCreateInstance 修改        │
│  controller/openclaw_upgrade.go - HandleInstanceList 修改        │
│  - HandleUpgrade 修改                                             │
│  - HandleReinstall 修改        [开发者E]                          │
│                                controller/admin_agent_types.go   │
│                                - HandleAdminAgentTypes (只读)     │
└─────────────────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 阶段3：防护集成（依赖 guard.go 完成）                             │
│                                                                   │
│  修改以下接口，增加防护调用：                                      │
│  - HandleSetModel                                                 │
│  - HandleSetChannel                                               │
│  - HandleAddSkill                                                 │
│  - HandleAddPlugin                                                │
│  - HandleSetRole（如支持）                                        │
└─────────────────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 阶段4：路由注册 + 审计规则（阻塞点）                               │
│                                                                   │
│  main.go                                                          │
│  - 注册新路由                                                     │
│                                                                   │
│  controller/audit.go                                              │
│  - 添加审计规则（仅 /admin/images/update）                        │
└─────────────────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 阶段5：前端（可并发）                                             │
│                                                                   │
│  [前端A - 管控端]              [前端B - 用户端]                    │
│  镜像管理页                    创建实例弹窗                       │
│  - 类型选择下拉                - 类型选择                         │
│  - 版本显示                    实例列表                           │
│  - 类型筛选                    - 类型标签显示                     │
│                                - 类型筛选                         │
└─────────────────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────────┐
│ 阶段6：测试 + 文档（可并发）                                      │
│                                                                   │
│  [测试]                        [文档]                             │
│  model/agent_type_test.go      docs/API.md 更新                  │
│  controller/*_test.go                                             │
└─────────────────────────────────────────────────────────────────┘
```

---

### 依赖关系图（关键路径）

```
数据库脚本
    │
    ├──► model/agent_type.go ──┬──► controller/admin_agent_types.go (只读)
    │    (硬编码)               │
    ├──► model/ai_image.go ────┼──► controller/admin_images.go ──► 路由注册
    │                          │
    └──► model/instance.go ────┼──► controller/openclaw.go
                               │
                               ├──► controller/agent_type_guard.go ──► 防护集成
                               │
                               └──► controller/openclaw_upgrade.go
```

---

## 12.1 向后兼容保障矩阵

| 场景 | 旧前端 + 新后端 | 新前端 + 新后端 |
|------|----------------|----------------|
| 获取启用镜像 | `GetEnabledImage()` 返回任意一个 | `GetEnabledImageByType(agentType)` 返回对应类型 |
| 创建实例 | 不传 `agent_type`，默认 `openclaw` | 传 `agent_type` |
| 启用镜像 | 无类型镜像全局互斥 | 有类型镜像同类型互斥 |
| 实例列表 | 正常显示所有实例 | 增加类型字段 |
| 配置接口 | 只操作 OpenClaw 实例，正常 | 非 OpenClaw 类型被防护拒绝 |

## 13. 文件修改清单

### 13.1 后端文件

| 文件 | 类型 | 说明 |
|------|------|------|
| `sql/0415-add-image-agent-type.sql` | 新增 | 镜像表新增字段 |
| `sql/0415-add-instance-agent-type.sql` | 新增 | 实例表新增字段 |
| `sql/init.sql` | 修改 | 同步表结构 |
| `model/agent_type.go` | 新增 | 智能体类型配置（硬编码） |
| `model/ai_image.go` | 修改 | 新增字段+查询函数 |
| `model/instance.go` | 修改 | 新增字段 |
| `controller/admin_agent_types.go` | 新增 | 类型查询 API（只读） |
| `controller/agent_type_guard.go` | 新增 | 防护检查函数 |
| `controller/admin_images.go` | 修改 | 镜像管理接口 |
| `controller/openclaw.go` | 修改 | 实例创建/列表 |
| `controller/openclaw_upgrade.go` | 修改 | 升级接口 |
| `controller/admin_instances.go` | 修改 | 管控端实例列表 |
| `controller/audit.go` | 修改 | 新增审计规则 |
| `main.go` | 修改 | 注册新路由+WithAudit |
| `docs/API.md` | 修改 | API 文档更新 |

### 13.2 后端测试文件

| 文件 | 类型 | 说明 |
|------|------|------|
| `model/agent_type_test.go` | 新增 | 类型校验测试 |
| `model/ai_image_test.go` | 新增 | 镜像查询测试 |
| `controller/agent_type_guard_test.go` | 新增 | 防护逻辑测试 |
| `controller/admin_images_test.go` | 新增 | 镜像启用测试 |

### 13.3 前端文件

| 文件 | 类型 | 说明 |
|------|------|------|
| `src/types/api.ts` | 修改 | 类型定义 |
| `src/services/admin.ts` | 修改 | API 调用 |
| `src/pages/admin/ImageManagement.tsx` | 修改 | 镜像管理 |
| `src/pages/admin/ModelConfig.tsx` | 修改 | 增加提示 |
| `src/pages/admin/ChannelConfig.tsx` | 修改 | 增加提示 |
| `src/pages/admin/SkillConfig.tsx` | 修改 | 增加提示 |
| `src/pages/tenant/InstanceList.tsx` | 修改 | Tab 分组 |
| `src/pages/tenant/CreateInstance.tsx` | 修改 | 类型选择 |
| `src/components/ChatView.tsx` | 修改 | 类型筛选 |

## 14. 实施检查清单

### 14.1 后端防护检查点

- [ ] 实例创建：校验 agent_type 有效性（硬编码白名单）
- [ ] 实例创建：非角色支持类型忽略 role_id
- [ ] 实例创建：按类型获取启用镜像
- [ ] 模型配置：校验实例类型支持
- [ ] 通道配置：校验实例类型支持
- [ ] 技能安装：校验实例类型支持
- [ ] 插件安装：校验实例类型支持
- [ ] 初始技能安装：按类型判断跳过
- [ ] 初始插件安装：按类型判断跳过
- [ ] 一键升级：按类型获取启用镜像
- [ ] 重装实例：按类型获取启用镜像
- [ ] 镜像启用：同类型互斥

### 14.2 安全检查点

- [ ] AgentType 硬编码白名单校验
- [ ] AgentVersion 正则格式校验
- [ ] 所有 SQL 使用参数化查询
- [ ] 管理接口有权限校验

### 14.3 日志检查点

- [ ] 实例创建成功记录 Info
- [ ] 类型校验失败记录 Warn
- [ ] 防护触发记录 Warn
- [ ] 数据库错误记录 Error

### 14.4 测试检查点

- [ ] IsValidAgentType 测试通过
- [ ] IsValidAgentVersion 测试通过
- [ ] AgentTypeSupportsRole 测试通过
- [ ] AgentTypeSupportsDetailConfig 测试通过
- [ ] GetEnabledImageByType 测试通过
- [ ] GetEnabledImageByType_Fallback 测试通过
- [ ] EnableImage_OnlyOnePerType 测试通过
- [ ] EnableImage_DifferentTypesIndependent 测试通过

### 14.5 审计日志检查点

- [ ] `/admin/images/update` 审计规则已添加
- [ ] 所有新增写操作接口已使用 `WithAudit()` 包装

### 14.6 数据库脚本检查点

- [ ] `sql/0415-add-image-agent-type.sql` 已创建（含注释头）
- [ ] `sql/0415-add-instance-agent-type.sql` 已创建（含注释头）
- [ ] `sql/init.sql` 已同步更新
- [ ] GORM 模型与 SQL 脚本字段类型一致

### 14.7 向后兼容检查点

- [ ] `GetEnabledImage()` 函数保留，行为不变
- [ ] 创建实例不传 `agent_type` 时默认 `openclaw`
- [ ] 无类型镜像（`agent_type` 为空）保持全局互斥
- [ ] 实例列表响应增加字段但不破坏旧前端解析
- [ ] 旧前端可正常操作 OpenClaw 类型实例

## 15. 风险与注意事项

### 15.1 兼容性

- 现有实例 `agent_type` 默认为 `openclaw`
- 现有镜像 `agent_type` 默认为空，需管理员手动配置
- 未配置类型的镜像可被任意类型实例使用（向后兼容回退）

### 15.2 数据一致性

- 每个智能体类型只能有一个生效镜像（应用层保证）
- 创建实例时需检查对应类型是否有可用镜像

### 15.3 扩展性

- 新增智能体类型需要修改 `model/agent_type.go` 中的硬编码配置
- 修改后需要重新编译和部署
- **优点**：无需数据库迁移，类型校验更快（内存操作）
- **缺点**：无法在运行时动态添加类型

### 15.4 硬编码 vs 数据库存储对比

| 特性 | 硬编码 | 数据库存储 |
|------|--------|-----------|
| 性能 | O(1) 内存查找 | 需要数据库查询 |
| 扩展性 | 需修改代码+重新部署 | 可运行时动态添加 |
| 复杂度 | 简单，无需额外表 | 需要管理额外的表和 API |
| 一致性 | 代码即配置，版本控制 | 需要数据迁移脚本 |
| 适用场景 | 类型变化不频繁 | 需要频繁动态调整 |

**当前选择硬编码的原因**：
1. 智能体类型数量有限且变化不频繁
2. 简化系统复杂度，减少数据库表
3. 无需管理员界面维护类型配置
4. 类型配置与代码版本保持同步
