# 多 Agent 类型镜像支持技术方案

## 一、背景与目标

当前 clawpro 只支持 openclaw 这一种 Agent 类型，所有实例共用同一个镜像。随着业务发展，需要同时支持以下三种 Agent 类型：

- **openclaw**：现有类型，继续沿用
- **hermers**：新增类型
- **lightclawACE**：新增类型

每种类型都有独立的官方镜像，同时也支持管理员自行导入自定义镜像。

---

## 二、核心概念

### 2.1 Agent 类型

Agent 类型是镜像和实例的分类维度，共三种：`openclaw`、`hermers`、`lightclawACE`。每种类型拥有独立的镜像池，互不干扰。

### 2.2 官方镜像 vs 自定义镜像

- **官方镜像**：由系统内置，每种 Agent 类型有预设的候选镜像列表，系统启动时自动检测可用性并导入。
- **自定义镜像**：管理员通过镜像管理页面手动导入，导入时必须指定所属的 Agent 类型。

### 2.3 默认镜像

每种 Agent 类型最多指定一个镜像作为"默认镜像"。默认镜像用于创建和重装实例时使用，同一 Agent 类型内只能有一个默认镜像（设置新的默认镜像时，旧的自动取消）。

### 2.4 实例与 Agent 类型的关系

- 实例在**创建时**选择 Agent 类型，之后不可修改。
- 实例创建和重装时，使用该 Agent 类型的默认镜像。
- 如果某个 Agent 类型没有设置默认镜像，则**不允许创建**该类型的实例。

---

## 三、业务规则

| 规则 | 说明 |
|------|------|
| 每种 Agent 类型最多一个默认镜像 | 启用新的默认镜像时，旧的自动禁用 |
| 无默认镜像则禁止创建机器 | 员工端 Agent 类型选项对无默认镜像的类型置灰禁用 |
| 自定义镜像导入必须指定 Agent 类型 | 镜像归属 Agent 类型后不可变更 |
| 实例的 Agent 类型不可修改 | 创建后固定，重装也使用原类型的默认镜像 |
| 存量实例默认归属 openclaw 类型 | 升级兼容，不影响现有数据 |

---

## 四、管控端功能变更

### 4.1 镜像管理页

**现有功能：**
- 展示所有镜像，切换启用/禁用（全局唯一 enabled）
- 导入自定义镜像（无类型概念）

**变更后：**
- 镜像列表增加"Agent 类型"列，支持按类型筛选
- 导入镜像时必须选择所属 Agent 类型
- 设置默认镜像的逻辑变为：**同 Agent 类型内唯一**，不影响其他类型
- 页面顶部新增各 Agent 类型的默认镜像状态总览（哪些类型已配置默认镜像、哪些未配置）

### 4.2 实例管理页

- 实例列表新增"Agent 类型"列
- 支持按 Agent 类型筛选实例

### 4.3 新增：Agent 类型状态概览 API

管控端提供一个接口，返回所有 Agent 类型的状态（是否已配置默认镜像），用于管理员快速了解当前各类型的可用情况。

---

## 五、员工端功能变更

### 5.1 创建机器

**现有流程：** 直接创建，使用唯一启用镜像。

**变更后流程：**
1. 员工选择 **Agent 类型**（openclaw / hermers / lightclawACE）
2. 系统检查该类型是否有默认镜像：
   - **有**：继续创建流程，使用该类型的默认镜像
   - **无**：该选项置灰，提示"管理员尚未配置该类型的默认镜像"
3. 其余创建流程不变

### 5.2 实例详情/列表

- 展示实例所属的 Agent 类型（仅展示，不可修改）

---

## 六、兼容性说明

- **存量镜像数据**：自动归属 `openclaw` 类型，`enabled` 状态保持不变，不影响现有逻辑
- **存量实例数据**：自动归属 `openclaw` 类型，员工和管理员的使用体验不变
- **API 兼容**：现有的 `/openclaw/current-image` 接口保留，返回 openclaw 类型的默认镜像；新接口 `/openclaw/agent-types` 提供完整的多类型视图

---

## 七、需要改动的模块汇总

| 模块 | 改动范围 | 说明 |
|------|---------|------|
| 数据库 | `ai_images` 表增加 `agent_type` 字段 | 存量数据默认 openclaw |
| 数据库 | `instances` 表增加 `agent_type` 字段 | 存量数据默认 openclaw |
| 官方镜像配置 | 新增 hermers、lightclawACE 的候选镜像 ID | 需确认具体镜像 ID |
| 镜像初始化逻辑 | 按 Agent 类型分别检查和导入官方镜像 | 独立管理各类型 |
| 默认镜像逻辑 | enabled 改为"同类型唯一"而非"全局唯一" | 核心逻辑变更 |
| 实例创建接口 | 新增 `agent_type` 参数，无默认镜像时拒绝创建 | 员工端调用 |
| 实例重装接口 | 按实例自身的 Agent 类型查找默认镜像 | 管控端和员工端都涉及 |
| 管控端镜像管理页 | 增加类型筛选、导入时选类型、状态总览 | 前端改动 |
| 员工端创建机器页 | 增加 Agent 类型选择步骤 | 前端改动 |

---

## 八、待确认事项

1. **hermers 和 lightclawACE 的官方镜像 ID** 是什么？（需提供后写入候选镜像配置）
2. **Agent 类型的显示名称**是否与代码中一致？（OpenClaw / Hermers / LightClaw ACE）
3. **不同 Agent 类型的实例是否有业务逻辑差异**？（如不同的初始化脚本、健康检查逻辑等，本期是否需要支持）
4. **员工端是否允许所有用户看到全部 Agent 类型**，还是按用户组权限控制？

---

## 九、多人协同开发拆分方案

### 任务依赖关系

```
T1（数据层 & 公共基础）
    ├── T2（管控端后端）── T4（管控端前端）
    └── T3（员工端后端）── T5（员工端前端）
```

**T1 是其他所有任务的前置依赖，需优先完成。T2/T3 完成后，T4/T5 才能联调。**

---

### T1：数据层 & 公共基础（优先级最高，约 1~2 天）

**负责人：1 人**

#### 改动范围

**`model/ai_image.go`**
- `AIImage` 结构体新增字段：
  ```go
  AgentType string `gorm:"type:varchar(32);not null;default:'openclaw';index" json:"agent_type"`
  ```
- 修改 `GetEnabledImage()` → 改为 `GetEnabledImageByAgentType(agentType string) *AIImage`，按 `agent_type` 查询 `enabled=true` 的镜像
- 新增 `GetAgentTypeStatus() map[string]bool`，返回各 Agent 类型是否已配置默认镜像

**`model/instance.go`**
- `Instance` 结构体新增字段：
  ```go
  AgentType string `gorm:"type:varchar(32);not null;default:'openclaw';index" json:"agent_type"`
  ```

**`common/image.go`**
- 将 `CandidateImages` 改为按 Agent 类型分组：
  ```go
  type CandidateImageEntry struct {
      ImageId   string
      ImageName string
      AgentType string // "openclaw" / "hermers" / "lightclawACE"
  }
  var CandidateImages = []CandidateImageEntry{...}
  ```
- 新增 `AgentTypes = []string{"openclaw", "hermers", "lightclawACE"}` 常量

**`sql/` 目录（MySQL 迁移脚本）**
- 新增迁移 SQL：
  ```sql
  ALTER TABLE ai_images ADD COLUMN agent_type VARCHAR(32) NOT NULL DEFAULT 'openclaw';
  ALTER TABLE instances ADD COLUMN agent_type VARCHAR(32) NOT NULL DEFAULT 'openclaw';
  CREATE INDEX idx_ai_images_agent_type ON ai_images(agent_type);
  CREATE INDEX idx_instances_agent_type ON instances(agent_type);
  ```

**验收标准：** 单元测试通过，数据库字段正常迁移，存量数据默认值为 `openclaw`

---

### T2：管控端后端（依赖 T1，约 2~3 天）

**负责人：1 人**

#### 改动范围

**`controller/admin_images.go`**

1. **`HandleAdminImages`（GET 镜像列表）**
   - JSON 响应新增 `agent_type` 字段
   - 支持 `?agent_type=xxx` 查询参数过滤

2. **`HandleImportImage`（导入自定义镜像）**
   - 新增必填参数 `agent_type`，校验合法性（只允许三种值）
   - 写入 `AIImage.AgentType`

3. **`HandleEnableImage`（启用/禁用镜像）**
   - 修改"全局唯一"逻辑 → **同 Agent 类型内唯一**：
     ```go
     // 禁用同类型其他镜像，再启用当前镜像
     model.DB.Model(&model.AIImage{}).
         Where("agent_type = ? AND enabled = ?", img.AgentType, true).
         Update("enabled", false)
     model.DB.Model(&img).Update("enabled", true)
     ```

4. **新增 `HandleAgentTypeStatus`（GET `/admin/agent-type-status`）**
   - 返回各 Agent 类型是否已配置默认镜像：
     ```json
     { "openclaw": true, "hermers": false, "lightclawACE": false }
     ```

**`controller/admin_instances.go`**
- 实例列表 JSON 响应新增 `agent_type` 字段，支持 `?agent_type=xxx` 过滤
- 重装实例接口：按实例自身的 `AgentType` 查找默认镜像

**`controller/admin_images.go` — `SeedAvailableImages()`**
- 改为按 Agent 类型分别初始化：
  ```go
  for _, agentType := range common.AgentTypes {
      seedImagesForAgentType(agentType)
  }
  ```

**`main.go`**
- 注册新路由 `GET /admin/agent-type-status`

**验收标准：** 镜像导入/启用/列表接口均携带 `agent_type`；同类型内只能有一个 enabled 镜像；`/admin/agent-type-status` 接口返回正确

---

### T3：员工端后端（依赖 T1，约 1~2 天）

**负责人：1 人**

#### 改动范围

**`controller/openclaw.go` — `HandleCreateInstance`**
- 新增参数 `agent_type`（默认 `openclaw`），校验合法性
- 检查该 Agent 类型是否有默认镜像，无则返回 400：
  ```go
  agentType := r.FormValue("agent_type")
  if agentType == "" {
      agentType = "openclaw"
  }
  defaultImage := model.GetEnabledImageByAgentType(agentType)
  if defaultImage == nil {
      writeError(w, r, http.StatusBadRequest,
          fmt.Errorf("Agent 类型 %s 尚未配置默认镜像，无法创建实例", agentType))
      return
  }
  ```
- 创建占位记录时写入 `AgentType`

**`controller/openclaw.go` — `HandleGetInstances`（实例列表）**
- 返回 JSON 新增 `agent_type` 字段

**`controller/openclaw_upgrade.go` — `HandleUpgrade`**
- 修改获取默认镜像逻辑：按实例自身 `AgentType` 查找
  ```go
  defaultImage := model.GetEnabledImageByAgentType(instance.AgentType)
  ```

**新增 `HandleGetAgentTypes`（GET `/openclaw/agent-types`）**
- 返回各 Agent 类型状态（是否有默认镜像），供员工端创建机器时判断哪些类型可用

**`main.go`**
- 注册新路由 `GET /openclaw/agent-types`

**验收标准：** 创建实例时 `agent_type` 参数生效；无默认镜像时拒绝创建；升级按实例类型查镜像

---

### T4：管控端前端（依赖 T2，约 2~3 天）

**负责人：1 人**

#### 改动范围

**`templates/admin_images.html`**
- 镜像列表表格新增「Agent 类型」列（加 badge 样式区分）
- 顶部新增各 Agent 类型默认镜像状态总览卡片（调用 `/admin/agent-type-status`）
- 导入镜像弹窗新增「Agent 类型」必选下拉框
- 筛选栏新增「按 Agent 类型筛选」下拉

**`templates/admin_instances.html`**
- 实例列表新增「Agent 类型」列
- 筛选栏新增「按 Agent 类型筛选」

**验收标准：** 镜像管理页展示 Agent 类型；导入时必须选类型；状态总览正确显示

---

### T5：员工端前端（依赖 T3，约 2~3 天）

**负责人：1 人**

#### 改动范围

**`templates/openclaw_create.html`（创建实例页）**
- 新增 Agent 类型选择步骤（默认选中 openclaw）
- 调用 `/openclaw/agent-types` 接口，对无默认镜像的类型置灰并提示「管理员尚未配置该类型的默认镜像」

**`templates/openclaw.html`（实例列表页）**
- 实例卡片按 Agent 类型分 tab 展示

**`templates/openclaw_detail.html`（实例详情页）**
- 展示实例所属 Agent 类型（只读）

**验收标准：** 创建流程有 Agent 类型选择；实例列表按类型分 tab；无默认镜像的类型正确置灰

---

### 任务汇总

| 任务 | 负责人数 | 预估工期 | 前置依赖 | 核心文件 |
|------|---------|---------|---------|---------|
| **T1 数据层 & 公共基础** | 1 人 | 1~2 天 | 无 | `model/ai_image.go`、`model/instance.go`、`common/image.go`、`sql/` |
| **T2 管控端后端** | 1 人 | 2~3 天 | T1 | `controller/admin_images.go`、`controller/admin_instances.go` |
| **T3 员工端后端** | 1 人 | 1~2 天 | T1 | `controller/openclaw.go`、`controller/openclaw_upgrade.go` |
| **T4 管控端前端** | 1 人 | 2~3 天 | T2 | `templates/admin_images.html`、`templates/admin_instances.html` |
| **T5 员工端前端** | 1 人 | 2~3 天 | T3 | `templates/openclaw*.html` |

> **注意：** T2 和 T3 可并行；T4 和 T5 可并行，但需等各自后端完成后联调。`SeedAvailableImages()` 的改造是核心风险点，建议 T1 完成后优先评审。hermers 和 lightclawACE 的官方镜像 ID 待确认后填入 `CandidateImages`。
