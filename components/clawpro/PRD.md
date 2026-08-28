# OpenClaw Enterprise 管控端核心页面 PRD

## 目录
1. [Tokens 监控](#tokens-监控)
2. [运维观测](#运维观测)
3. [会话管理](#会话管理)
4. [镜像管理（本期迭代）](#镜像管理本期迭代)

---

## Tokens 监控

### 页面概述
**Tokens 监控** 是管控端的核心数据分析页面，用于实时监控企业全局和用户级别的 Token 消耗情况，帮助管理员了解资源使用状况、成本分布和趋势。

### 核心功能

#### 1. 全局配额监控卡片
**位置**：页面顶部，五个关键指标卡

| 指标 | 说明 | 数据来源 |
|------|------|--------|
| 总请求数 | 选定时间范围内的总请求数 | `GET /api/tokens/summary` |
| 输入 Tokens | 选定时间范围内的输入 Token 总数 | `GET /api/tokens/summary` |
| 输出 Tokens | 选定时间范围内的输出 Token 总数 | `GET /api/tokens/summary` |
| 总 Tokens | 输入 + 输出 Token 总数 | 计算字段 |
| 今日全局配额消耗 | 当日 Token 消耗占全局上限的百分比 | `GET /api/tokens/global-quota` |

**接口规范**：
- **GET /api/tokens/summary**
  - 参数：`dateFrom`, `dateTo`
  - 返回：`{ totalRequests, inputTokens, outputTokens }`
  - 功能说明：[待补充]

- **GET /api/tokens/global-quota**
  - 参数：`date`
  - 返回：`{ consumed, limit, percentage, isUnlimited }`
  - 功能说明：[待补充]

#### 2. CLS 日志服务管理
**位置**：全局配额消耗卡片右侧

**功能**：
- 显示 CLS 日志服务的当前状态（开启/关闭）
- 支持开启 CLS 日志服务
- 支持关闭 CLS 日志服务（需要确认）

**状态显示**：
- 开启时：显示绿色对勾 + "CLS 日志服务已开启" + 功能说明
- 关闭时：显示"开启 CLS 日志服务"按钮

**接口规范**：
- **POST /api/cls/enable**
  - 参数：无
  - 返回：`{ success, message }`
  - 功能说明：[待补充]

- **POST /api/cls/disable**
  - 参数：无
  - 返回：`{ success, message }`
  - 功能说明：[待补充]

- **GET /api/cls/status**
  - 参数：无
  - 返回：`{ enabled, features: [] }`
  - 功能说明：[待补充]

#### 3. Token 消耗趋势图
**位置**：顶部指标卡下方

**功能**：
- 展示最近 7 天或选定时间范围内的 Token 消耗趋势
- 支持两条线：输入 Token 和输出 Token
- 支持日期范围选择

**接口规范**：
- **GET /api/tokens/trend**
  - 参数：`dateFrom`, `dateTo`
  - 返回：`[{ date, inputTokens, outputTokens }, ...]`
  - 功能说明：[待补充]

#### 4. 按用户分类统计
**位置**：趋势图下方，标签页 "按用户"

**功能**：
- 展示每个用户的 Token 消耗情况
- 支持按总请求数、输入 Token、输出 Token 排序
- 支持分页显示（每页 10 条）

**表格列**：
| 列名 | 说明 | 数据来源 |
|------|------|--------|
| 用户 ID | 用户邮箱或 ID | `GET /api/tokens/by-user` |
| 总请求数 | 该用户的请求总数 | 计算字段 |
| 输入 Tokens | 该用户的输入 Token 总数 | 计算字段 |
| 输出 Tokens | 该用户的输出 Token 总数 | 计算字段 |
| 总 Tokens | 输入 + 输出 | 计算字段 |

**接口规范**：
- **GET /api/tokens/by-user**
  - 参数：`dateFrom`, `dateTo`, `page`, `pageSize`
  - 返回：`{ data: [{ userId, requests, inputTokens, outputTokens }], total }`
  - 功能说明：[待补充]

#### 5. 按模型分类统计
**位置**：标签页 "按模型"

**功能**：
- 展示每个模型的 Token 消耗情况
- 支持按总请求数、输入 Token、输出 Token 排序
- 支持分页显示

**表格列**：同按用户分类

**接口规范**：
- **GET /api/tokens/by-model**
  - 参数：`dateFrom`, `dateTo`, `page`, `pageSize`
  - 返回：`{ data: [{ modelName, requests, inputTokens, outputTokens }], total }`
  - 功能说明：[待补充]

#### 6. 按会话分类统计
**位置**：标签页 "按会话"

**功能**：
- 展示每个会话的 Token 消耗情况
- 支持按总请求数、输入 Token、输出 Token 排序
- 支持分页显示

**接口规范**：
- **GET /api/tokens/by-session**
  - 参数：`dateFrom`, `dateTo`, `page`, `pageSize`
  - 返回：`{ data: [{ sessionId, requests, inputTokens, outputTokens }], total }`
  - 功能说明：[待补充]

### 交互流程

1. **初始加载**：
   - 加载今天的数据
   - 显示全局配额消耗
   - 显示最近 7 天的趋势图

2. **日期范围选择**：
   - 用户选择日期范围
   - 刷新所有数据和图表

3. **开启/关闭 CLS 日志服务**：
   - 用户点击"开启 CLS 日志服务"按钮
   - 调用 `POST /api/cls/enable` 接口
   - 显示加载状态
   - 成功后显示成功提示（Toast）
   - 更新 CLS 状态显示

4. **关闭 CLS 日志服务**：
   - 用户点击"关闭 CLS 日志服务"按钮
   - 显示确认对话框
   - 用户确认后调用 `POST /api/cls/disable` 接口
   - 显示加载状态
   - 成功后显示成功提示

### 数据刷新
- 支持手动刷新按钮
- 刷新时显示加载状态
- 刷新完成后显示成功提示

---

## 运维观测

### 页面概述
**运维观测** 是管控端的系统监控页面，用于实时监控系统的日志、消息处理、队列状态等运维指标，帮助运维人员快速定位和解决系统问题。

### 核心功能

#### 1. 日志级别分布
**位置**：页面左上方

**功能**：
- 展示不同日志级别（ERROR、WARNING、INFO、DEBUG）的日志数量
- 支持柱状图展示

**接口规范**：
- **GET /api/logs/level-distribution**
  - 参数：`dateFrom`, `dateTo`
  - 返回：`[{ level, count }, ...]`
  - 功能说明：[待补充]

#### 2. 日志模块分布
**位置**：页面右上方

**功能**：
- 展示不同模块的日志数量
- 支持水平柱状图展示
- 显示模块名称和日志数量

**接口规范**：
- **GET /api/logs/module-distribution**
  - 参数：`dateFrom`, `dateTo`
  - 返回：`[{ moduleName, count }, ...]`
  - 功能说明：[待补充]

#### 3. 消息处理统计
**位置**：页面中间

**功能**：
- 展示消息处理数量和队列中的消息数量
- 支持折线图展示
- 显示处理速度和队列深度

**接口规范**：
- **GET /api/messages/processing-stats**
  - 参数：`dateFrom`, `dateTo`
  - 返回：`[{ time, processed, queued }, ...]`
  - 功能说明：[待补充]

#### 4. 队列状态监控
**位置**：页面下方

**功能**：
- 展示队列的平均深度和平均等待时间
- 支持折线图展示
- 实时监控队列健康状态

**接口规范**：
- **GET /api/queue/status**
  - 参数：`dateFrom`, `dateTo`
  - 返回：`[{ time, depthAvg, waitMsAvg }, ...]`
  - 功能说明：[待补充]

#### 5. 日志搜索和筛选
**位置**：页面底部

**功能**：
- 支持按日志级别筛选
- 支持按模块筛选
- 支持按关键词搜索
- 支持日期范围筛选

**接口规范**：
- **GET /api/logs/search**
  - 参数：`level`, `module`, `keyword`, `dateFrom`, `dateTo`, `page`, `pageSize`
  - 返回：`{ data: [{ timestamp, level, module, message }], total }`
  - 功能说明：[待补充]

### 交互流程

1. **初始加载**：
   - 加载最近 24 小时的日志统计数据
   - 显示各个图表

2. **日期范围选择**：
   - 用户选择日期范围
   - 刷新所有数据和图表

3. **日志搜索**：
   - 用户输入搜索条件
   - 调用搜索接口
   - 显示搜索结果

4. **数据刷新**：
   - 支持手动刷新按钮
   - 刷新时显示加载状态

---

## 会话管理

### 页面概述
**会话管理** 是管控端的会话监控页面，用于查看和管理企业内所有的对话会话，包括会话统计、渠道分布、模型分布等信息。

### 核心功能

#### 1. 会话统计卡片
**位置**：页面顶部，四个关键指标卡

| 指标 | 说明 | 数据来源 |
|------|------|--------|
| 总会话数 | 企业内的会话总数 | `GET /api/sessions/stats` |
| 平均轮次 | 每个会话的平均轮次 | `GET /api/sessions/stats` |
| 工具调用 | 所有会话中的工具调用总数 | `GET /api/sessions/stats` |
| 活跃渠道 | 当前活跃的渠道数量 | `GET /api/sessions/stats` |

**接口规范**：
- **GET /api/sessions/stats**
  - 参数：`dateFrom`, `dateTo`
  - 返回：`{ totalSessions, avgRounds, toolCalls, activeChannels: [] }`
  - 功能说明：[待补充]

#### 2. 按渠道分布
**位置**：页面左下方

**功能**：
- 展示不同渠道（Feishu、QQ、Webchat 等）的会话分布
- 支持柱状图展示

**接口规范**：
- **GET /api/sessions/by-channel**
  - 参数：`dateFrom`, `dateTo`
  - 返回：`[{ channelName, count }, ...]`
  - 功能说明：[待补充]

#### 3. 按模型分布
**位置**：页面右下方

**功能**：
- 展示不同模型的会话分布
- 支持饼图展示

**接口规范**：
- **GET /api/sessions/by-model**
  - 参数：`dateFrom`, `dateTo`
  - 返回：`[{ modelName, count }, ...]`
  - 功能说明：[待补充]

#### 4. 会话列表
**位置**：页面中间

**功能**：
- 展示所有会话的详细信息
- 支持按会话名称、渠道、模型、状态筛选
- 支持按 Token 消耗、创建时间排序
- 支持分页显示（每页 10 条）
- 支持点击进入会话详情页

**表格列**：
| 列名 | 说明 | 数据来源 |
|------|------|--------|
| 会话名称 | 会话的显示名称 | `GET /api/sessions/list` |
| 渠道 | 会话所属的渠道 | `GET /api/sessions/list` |
| 模型 | 会话使用的模型 | `GET /api/sessions/list` |
| Tokens | 会话消耗的 Token 数量 | `GET /api/sessions/list` |
| 成本 | 会话的成本（美元） | 计算字段 |
| 最后消息 | 会话的最后一条消息摘要 | `GET /api/sessions/list` |
| 更新时间 | 会话的最后更新时间 | `GET /api/sessions/list` |
| 状态 | 会话的当前状态（活跃/已关闭） | `GET /api/sessions/list` |

**接口规范**：
- **GET /api/sessions/list**
  - 参数：`channel`, `model`, `status`, `sortBy`, `sortOrder`, `page`, `pageSize`
  - 返回：`{ data: [{ id, name, channel, model, tokens, cost, lastMessage, updatedAt, status }], total }`
  - 功能说明：[待补充]

#### 5. 会话详情页
**位置**：点击会话列表中的会话后跳转

**功能**：
- 展示会话的详细信息
- 展示会话的对话历史
- 支持返回会话列表

**接口规范**：
- **GET /api/sessions/{sessionId}**
  - 参数：`sessionId`
  - 返回：`{ id, name, channel, model, tokens, cost, createdAt, updatedAt, status, messages: [] }`
  - 功能说明：[待补充]

- **GET /api/sessions/{sessionId}/messages**
  - 参数：`sessionId`, `page`, `pageSize`
  - 返回：`{ data: [{ timestamp, role, content }], total }`
  - 功能说明：[待补充]

### 交互流程

1. **初始加载**：
   - 加载会话统计数据
   - 加载会话列表（第一页）
   - 加载渠道和模型分布数据

2. **筛选和排序**：
   - 用户选择筛选条件（渠道、模型、状态）
   - 用户选择排序方式
   - 刷新会话列表

3. **分页**：
   - 用户点击分页按钮
   - 加载对应页的数据

4. **查看会话详情**：
   - 用户点击会话列表中的会话
   - 跳转到会话详情页
   - 加载会话的详细信息和对话历史

5. **返回列表**：
   - 用户点击返回按钮
   - 返回会话列表页

---

## 通用接口规范

### 错误处理
所有接口返回格式：
```json
{
  "success": true/false,
  "data": {},
  "error": "错误信息（仅在失败时返回）"
}
```

### 分页
- `page`：页码（从 1 开始）
- `pageSize`：每页条数（默认 10）
- 返回：`{ data: [], total, page, pageSize }`

### 日期格式
- 所有日期参数使用 `YYYY-MM-DD` 格式
- 所有返回的时间戳使用 ISO 8601 格式

### 认证
- 所有接口需要在请求头中包含 `Authorization: Bearer <token>`

---

## 镜像管理（本期迭代）

> **分支**：feature/admin-image-management  
> **日期**：2026-04-15  
> **状态**：前端开发完成，待后端对齐

### 页面概述
**镜像管理** 是管控端的运行环境管理页面，管理不同 Agent 类型的运行环境镜像。本期核心改造：将镜像启用模式从「全局单镜像」升级为「按 Agent 类型多镜像」，支持平台同时运行多种 Agent 启动模板，并为每个镜像建立可持久化的版本信息。

### 核心改造

#### 1. 镜像启用逻辑改造：单启用 → 按类型多启用

##### 1.1 变更说明

| 维度 | 改造前 | 改造后 |
|------|--------|--------|
| 启用粒度 | 全局唯一一个 `activeImageId` | 每个 `agentType` 各自维护一个启用镜像 |
| 启动模板 | 平台 1 个启动模板 | 平台 1~N 个启动模板（每个已启用类型对应一个） |
| 用户端选择 | 无类型选择 | 用户创建 Agent 时可选择已启用的类型 |

##### 1.2 启用规则

| 规则 | 说明 |
|------|------|
| 类型内唯一 | 同一 `agentType` 下最多一个镜像 `active=true`，启用新镜像时自动取消该类型下旧的启用镜像 |
| 无版本禁止启用 | `agentVersion` 为空的镜像不允许启用（前端 Switch 禁用 + hover 提示；后端接口拒绝） |
| 存量兼容（无版本已启用） | 已 `active=true` 但无版本的存量镜像，保持当前运行不中断，但给**强警告提示**：版本列显示黄色 `⚠ 未填写版本` 标签，Switch hover 提示"该镜像缺少版本信息，建议尽快删除后重新导入并填写版本" |
| 存量兼容（无版本未启用） | Switch 禁用（灰色不可点击），hover 提示"缺少 Agent 版本信息，无法启用。请删除后重新导入并填写版本" |

##### 1.3 用户端首选逻辑

| 规则 | 说明 |
|------|------|
| 全局唯一 | 仅一个 `agentType` 可设为"用户端首选"，存储为 `defaultAgentType` |
| 设置前提 | 该类型下必须有已启用的镜像，否则拒绝设置 |
| 首选类型约束 | 首选类型的启用镜像不可取消启用、不可删除 |
| 非首选类型 | 可取消启用（取消后该类型在用户端不可选） |
| 用户端表现 | 用户创建 Agent 时默认选中首选类型，也可手动切换其他已启用类型 |
| 一键升级 | 以该类型启用镜像的版本作为升级目标版本 |

##### 1.4 前端改造
- 镜像列表按 `agentType` 组织展示，每组为一个可折叠的卡片模块
- 每个模块标题栏显示：类型名称、镜像数量、启用状态标签（用户端可选/不可选）、首选标签
- 每行镜像独立的 Switch 启用开关
- 无版本镜像：未启用的 Switch 禁用 + Tooltip 提示
- 存量已启用无版本镜像：版本列显示黄色 `⚠ 未填写版本` 标签 + Tooltip 提示建议操作

##### 1.5 后端改造
- **数据模型**：原全局 `activeImageId` 改为按 `agentType` 维护，新增 `default_agent_type` 全局配置字段
- **启用接口**：新增 `agentVersion` 非空校验；启用时事务内先取消同类型旧启用，再设置新启用
- **启动模板**：从 1 个扩展为 N 个，每个已启用类型对应一个启动模板

**接口规范**：

- **PUT /api/images/{imageId}/activate**
  - 参数：`imageId`
  - 校验：`agentVersion` 非空，否则返回 400
  - 逻辑：事务内取消同 `agentType` 下旧启用镜像，设置目标镜像 `active=true`
  - 返回：`{ success, data: { imageId, agentType, active } }`

- **PUT /api/images/{imageId}/deactivate**
  - 参数：`imageId`
  - 校验：若该镜像所属 `agentType` 为 `defaultAgentType`，拒绝取消
  - 返回：`{ success }`

- **PUT /api/config/default-agent-type**
  - 参数：`{ agentType }`
  - 校验：该类型下必须有 `active=true` 的镜像
  - 返回：`{ success, data: { defaultAgentType } }`

- **GET /api/config/default-agent-type**
  - 返回：`{ defaultAgentType }`

---

#### 2. 导入镜像支持选择 Agent 类型 + 版本

##### 2.1 变更说明
导入镜像时必须指定 `agentType` 和 `agentVersion`，不再允许无类型/无版本的裸导入。

##### 2.2 导入弹窗交互

| 步骤 | 公共镜像 | 自定义镜像（首次导入） | 自定义镜像（删除后重新导入） |
|------|----------|----------------------|--------------------------|
| 选择镜像 | 从腾讯云镜像列表选择 | 从腾讯云镜像列表选择 | 从腾讯云镜像列表选择 |
| Agent 类型 | 自动匹配，只读不可改 | 手动从下拉选择 | **自动回填上次配置，可编辑** |
| Agent 版本 | 自动填充，只读不可改 | 手动输入，需校验格式 | **自动回填上次版本，可编辑** |

> 删除镜像时后端记录该镜像的 `agentType` + `agentVersion`；再次导入同一镜像时自动回填，并提示"已自动填入上次配置，可修改"。

##### 2.3 支持的 Agent 类型（本期）

| 类型 | 版本格式 | 示例 | 校验规则 |
|------|----------|------|----------|
| OpenClaw | `YYYY.M.D`（日期格式） | `2026.4.2` | 正则 `^\d{4}\.\d{1,2}\.\d{1,2}$` + 日期合法性 |
| Hermes Agent | semver | `0.8.0` | 正则 `^\d+\.\d+\.\d+$` |
| LightClaw ACE | semver | `1.0.2` | 正则 `^\d+\.\d+\.\d+$` |

> **本期仅支持以上三个系统预设类型，不支持自定义 Agent 类型。** 自定义类型后续迭代考虑。

##### 2.4 其他规则
- 同一镜像 ID 不允许重复导入
- 镜像大小不超过 50GiB（后端校验）
- 版本信息导入后不可在列表页直接编辑（需删除后重新导入，此时会自动回填上次配置）
- 前后端双重校验版本格式

##### 2.5 前端改造
- 导入弹窗新增 Agent 类型下拉（仅三个系统预设类型）
- 导入弹窗新增 Agent 版本输入框，根据类型动态显示 placeholder 和格式校验
- 镜像列表分"公共镜像（腾讯云维护）"和"自定义镜像（企业维护）"两组展示
- 支持搜索镜像 ID/名称、刷新列表
- **删除后重新导入**：自动回填上次的 type/version，蓝色提示"已自动填入上次配置，可修改"

##### 2.6 后端改造
- 镜像导入接口新增 `agentType`（必填）、`agentVersion`（必填）参数
- 后端双重校验版本格式（同前端校验规则）
- 写入 images 表对应字段
- **删除镜像时记录历史**：后端需在 `image_delete_history` 表（或同表软删除）保存 `imageId` + `agentType` + `agentVersion`，供重新导入时查询回填

**接口规范**：

- **POST /api/images/import**
  - 参数：`{ imageId, agentType, agentVersion }`
  - 校验：imageId 不重复、agentVersion 格式合法、镜像大小 ≤ 50GiB
  - 返回：`{ success, data: { id, name, agentType, agentVersion, type, status, os, createTime } }`

- **GET /api/images/importable**
  - 功能：代理查询腾讯云 CVM 可导入的镜像列表
  - 参数：`{ keyword? }`（可选搜索关键词）
  - 返回：`{ publicImages: [{ id, name, agentType, agentVersion }], customImages: [{ id, name }] }`

- **GET /api/images/delete-history/{imageId}**
  - 功能：查询该镜像上次删除前的 type/version 配置
  - 返回：`{ agentType?, agentVersion? }`（无历史则返回空）

---

#### 3. 镜像版本信息持久化

##### 3.1 变更说明
版本信息从前端硬编码改为后端持久化存储，前端从接口读取。

##### 3.2 数据库改造

**images 表新增字段**：

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `agent_type` | `VARCHAR(64)` | `NULL` | Agent 类型标识，如 `OpenClaw`、`HermesAgent`、`LightClawACE` |
| `agent_version` | `VARCHAR(32)` | `NULL` | Agent 版本号 |

**新增 image_delete_history 表**（或复用软删除字段）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `image_id` | `VARCHAR(64)` PK | 镜像 ID |
| `agent_type` | `VARCHAR(64)` | 删除前的 Agent 类型 |
| `agent_version` | `VARCHAR(32)` | 删除前的 Agent 版本 |
| `deleted_at` | `TIMESTAMP` | 删除时间 |

##### 3.3 存量数据迁移

| 场景 | 迁移策略 |
|------|----------|
| 无 `agent_type` 的存量镜像 | 默认填充 `OpenClaw` |
| `agent_version` 留空 | 不强制填充，保持为空 |
| 已 `active=true` 且无版本 | 保持运行不中断，前端给强警告提示 |
| 原全局 `activeImageId` | 迁移为该镜像所属 `agentType` 下的启用镜像 |
| `defaultAgentType` | 初始设为 `OpenClaw` |

##### 3.4 接口改造

| 接口 | 改造内容 |
|------|----------|
| 镜像列表查询 | 返回 `agentType`、`agentVersion` 字段，支持按 `agentType` 组织 |
| 镜像导入 | 新增 `agentType`、`agentVersion` 必填参数 |
| 镜像启用 | 校验 `agentVersion` 非空 |
| 镜像删除 | 校验：首选类型的启用镜像不可删除；删除时写入 delete_history |

**接口规范**：

- **GET /api/images**
  - 参数：`{ agentType? }`（可选过滤）
  - 返回：`{ data: [{ id, name, status, type, agentType, agentVersion, os, createTime, active }] }`

- **DELETE /api/images/{imageId}**
  - 校验：启用中且所属类型为首选类型的镜像不可删除
  - 副作用：写入 `image_delete_history` 表
  - 返回：`{ success }`

---

### 用户端联动（后续跟进）

| 联动点 | 说明 |
|--------|------|
| 创建 Agent | 展示已启用类型列表，默认选中首选类型 |
| 一键升级 | 以对应类型启用镜像版本为目标版本 |
| Agent 详情 | 展示 `agentType` + `agentVersion` 信息 |

### 后续迭代预留

| 功能 | 说明 |
|------|------|
| 自定义 Agent 类型 | 支持管理员自行创建新的 Agent 类型（本期仅支持三个系统预设） |

---

## 后续补充清单

- [ ] 所有接口的详细功能说明
- [ ] 所有接口的错误处理规范
- [ ] 所有接口的速率限制
- [ ] 所有接口的缓存策略
- [ ] 实时数据更新的推送机制（WebSocket/Server-Sent Events）
- [ ] 数据导出功能（CSV/Excel）
- [ ] 告警和通知机制
- [ ] 权限控制规范
