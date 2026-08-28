# 企业 MCP 库 — 产品需求文档

## 需求背景

当前平台的 AI 智能体（龙虾/OpenClaw）需要连接外部工具和服务来扩展能力。MCP（Model Context Protocol）是标准的外部能力接入协议，需要在管理端提供统一的 MCP 服务管理入口，支持对 MCP 服务进行新增、查看、删除和下发操作。

---

## 功能总览

企业 MCP 库作为「Agent 工具库」的第四个 Tab 页（与公共技能库、企业技能库、企业插件库并列），提供 MCP 服务的全生命周期管理能力。

---

## 一、管理端

### 1.1 Tab 入口

**页面入口**：管理端侧边栏 → Agent 工具库 → 企业MCP库（第四个 Tab）

**Tab 描述**：统一管理 MCP 服务配置，支持远程服务和本地命令两种连接方式，按需下发到智能体实例。

---

### 1.2 查看 MCP 列表

**视图模式**：支持列表视图和卡片视图两种模式切换（右上角切换按钮）

**列表视图字段**：

| 字段 | 说明 | 列宽 |
|------|------|------|
| 名称 / 服务ID | 名称为显示用途，服务ID 为唯一标识（对应 JSON 中 `mcp.servers.{服务ID}`） | 20% |
| 状态/下发动态 | 运行状态 + 最近下发结果（未下发/下发中/已下发成功率） | 15% |
| 版本号 | 灰色圆角标签，显示 `v{version}`，样式参考企业插件库 | 7% |
| 描述 | MCP 服务的简要说明（支持 Tooltip 展示完整内容） | 33% |
| 创建时间 | 格式：YYYY/MM/DD | 10% |
| 操作 | 下发、删除 | 15% |

**卡片视图**：3 列网格，每张卡片包含名称、版本号标签、描述（最多 2 行截断）、操作按钮（下发/删除）

**交互说明**：
- 列表支持按名称、描述、服务ID 搜索
- 点击行/卡片可进入详情页
- 交互风格参考现有「企业插件库」

---

### 1.3 操作能力

#### 1.3.1 删除

- 点击列表中的「删除」按钮，弹出二次确认弹窗（AlertDialog 危险操作样式）
- 确认文案：**确定要删除 MCP「xxx」吗？此操作不可撤销。**
- 删除按钮为红色，使用 AlertDialog 组件

#### 1.3.2 下发

- 功能说明：将选中的 MCP 服务配置下发到指定的智能体实例
- 支持从列表页和详情页两个入口发起下发
- 下发MCP到 ~/.openclaw/openclaw.json 文件中

**批量下发弹窗**（复用 BatchDistributeDialog 组件，MCP 专属配置）：

| 功能 | 说明 |
|------|------|
| 弹窗标题 | 批量下发 MCP 配置 |
| 实例搜索 | 支持按名称和服务ID 搜索 |
| 状态筛选 | 单选下拉：全部下发状态 / 未下发 / 下发失败（区别于技能库的多选筛选） |
| 隐藏字段 | 隐藏创建人和组织信息（MCP 场景不需要） |
| 全选 | 跨页全选/取消全选 |
| 分页 | 支持 10/20/50/100/200/500 条/页切换 |

**筛选限制**（仅展示满足以下全部条件的实例）：
- 智能体类型为 **OpenClaw**
- 实例状态为 **运行中**
- 下发状态为 **未下发** 或 **下发失败**

用户勾选目标实例后，点击「确认下发」执行部署。下发过程中显示进度条动画，完成后显示 toast 提示。

#### 1.3.3 下发流程

MCP 配置下发到实例时，按以下流程操作实例上的 `openclaw.json` 文件：

```
开始下发
  │
  ├─ 检查实例是否存在 openclaw.json 文件
  │    │
  │    ├─ 不存在 → 新建 openclaw.json 文件（写入完整 JSON 配置）→ 完成 ✅
  │    │
  │    └─ 存在 → 继续
  │         │
  │         ├─ 检查 JSON 格式是否正确
  │         │    │
  │         │    ├─ 不正确 → 报错 ❌（提示：openclaw.json 格式异常，请手动检查）
  │         │    │
  │         │    └─ 正确 → 继续
  │         │         │
  │         │         ├─ 检查 JSON 中是否存在 mcp.servers.{服务ID}
  │         │         │    │
  │         │         │    ├─ 不存在 → 新建 mcp.servers.{服务ID} 节点，写入配置 → 完成 ✅
  │         │         │    │
  │         │         │    └─ 存在 → 覆盖 mcp.servers.{服务ID} 内容 → 完成 ✅
```

> 注意：写入 openclaw.json 时统一使用 **2 空格**缩进。

---

### 1.4 MCP 详情页

**入口**：列表页点击某条 MCP 记录进入

**基本信息卡片**：
- 名称（大标题）
- 服务ID
- 连接方式标签（蓝色圆角）
- 版本号标签（灰色圆角）
- 创建时间
- 描述
- 操作按钮：删除

**Tab 区域**：

##### Tab 1 — 文件列表

三栏布局：

| 栏 | 宽度 | 内容 |
|----|------|------|
| 左列：版本列表 | 14% | 按版本号从新到旧排列，最新版带「最新」标签，点击切换版本 |
| 中列：文件列表 | 22% | 固定三个文件：使用说明.md、工具说明.md、服务配置.json |
| 右列：内容展示 | 剩余 | 支持「预览」和「源码」两种模式切换 |

- Markdown 文件默认预览模式（使用 MDXRenderer 渲染）
- JSON 文件默认源码模式（带语法高亮和行号）
- 空内容显示「暂无内容」占位

##### Tab 2 — 下发记录

- 展示该 MCP 的所有下发历史记录
- 每条记录包含：时间、下发状态（下发中/下发完成/部分失败）、进度条
- 点击「查看详情」打开弹窗，显示每个实例的下发状态、失败原因
- 下发详情弹窗支持搜索和状态筛选
- 右上角「批量下发」按钮（下发进行中时禁用）

---

### 1.5 新增 MCP

**入口**：列表页右上角「+ 新增 MCP」按钮

**交互方式**：两步表单 Dialog

#### 第一步：基本信息 + 服务配置

| 字段 | 必填 | 说明 |
|------|------|------|
| 服务ID | ✅ | MCP 服务的唯一标识（对应 JSON 中 `mcp.servers.{服务ID}`），创建后不可修改 |
| 名称 | — | MCP 服务的显示名称，不填则默认与服务ID 一致 |
| 描述 | — | MCP 服务的简要说明 |
| 连接方式 | ✅ | 下拉选择：SSE / Streamable HTTP / STDIO，每项包含说明文字 |
| 服务配置 | ✅ | JSON 格式编辑器（详见 1.5.1） |

#### 1.5.1 服务配置编辑器

**创新交互**：固化外层 JSON 结构，用户仅编辑第三层内容。

- 外层 `{ "mcp": { "servers": { ... } } }` 由系统固定显示（灰色背景，不可编辑）
- 用户在中间的可编辑区域填写服务器配置内容
- 切换连接方式时自动填入对应模板
- 连接方式下方有可折叠的配置参考文档，展示字段说明和完整示例

**连接方式下拉选项**：

| 选项值 | 显示名称 | 描述文字 |
|--------|---------|---------|
| `sse` | SSE | 远程服务（兼容旧版 MCP 2024-11-05） |
| `streamable-http` | Streamable HTTP | 远程服务（推荐，MCP 2025-06-18） |
| `stdio` | STDIO | 本地命令，通过标准输入输出通信 |

**编辑器自动填充模板**（切换连接方式时自动填入，仅当编辑区为空或仍为其他模板时触发）：

##### SSE 自动填充模板

```json
"your-server-name": {
    "transport": "sse",
    "url": "MCP服务的URL",
    "headers": {
        "Authorization": "<your-token>"
    },
    "timeout": 60
}
```

##### Streamable HTTP 自动填充模板

```json
"your-server-name": {
    "transport": "streamable-http",
    "url": "MCP服务的URL",
    "headers": {
        "Authorization": "<your-token>"
    },
    "timeout": 60
}
```

##### STDIO 自动填充模板

```json
"your-server-name": {
    "transport": "stdio",
    "command": "python3",
    "args": ["/opt/mcp/your-server.py"],
    "env": {
        "PYTHONUNBUFFERED": "1"
    },
    "cwd": "/path/to/your/workdir",
    "timeout": 60
}
```

#### 1.5.2 配置参考文档（可折叠面板）

选择连接方式后，在编辑器上方会出现一个可折叠面板，标题为 **「查看「{连接方式名称}」配置参考」**，点击展开后显示该连接方式的完整字段说明和配置示例。

##### SSE 配置参考

| 字段 | 必填 | 说明 |
|------|------|------|
| `transport` | ✅ | 固定值 `"sse"` |
| `url` | ✅ | 必须以 http 或 https 开头（常见以 `/sse` 结尾） |
| `headers` | — | 如 MCP Server 要求 Token 认证，在此填写；否则可删除 |
| `security_zone` | — | 如 MCP 部署在 DevCloud，填写 `"devnet"` |
| `timeout` | — | 超时时间，单位秒，默认 60 |
| `username` | — | 用户标识 |

##### Streamable HTTP 配置参考

| 字段 | 必填 | 说明 |
|------|------|------|
| `transport` | ✅ | 固定值 `"streamable-http"` |
| `url` | ✅ | 必须以 http 或 https 开头（常见以 `/mcp` 结尾） |
| `headers` | — | 如 MCP Server 要求 Token 认证，在此填写；否则可删除 |
| `security_zone` | — | 如 MCP 部署在 DevCloud，填写 `"devnet"` |
| `timeout` | — | 超时时间，单位秒，默认 60 |
| `username` | — | 用户标识 |

##### STDIO 配置参考

| 字段 | 必填 | 说明 |
|------|------|------|
| `transport` | ✅ | 固定值 `"stdio"` |
| `command` | ✅ | 可执行文件路径（支持绝对/相对路径） |
| `args` | — | 传给命令的参数数组，没有可留空 `[]` |
| `env` | — | 启动时的环境变量，没有可整段删除 |
| `cwd` | — | 子进程工作目录，默认继承 Agent 目录 |
| `timeout` | — | 超时时间，单位秒，默认 60 |

> 💡 完整字段说明和类型定义见文末「[附录：服务配置完整字段参考](#附录服务配置完整字段参考)」。

---

#### 第二步：文档说明

| 字段 | 说明 |
|------|------|
| 使用说明 | Markdown 格式，支持编辑/预览切换。打开时自动填入默认模板 |
| 工具说明 | Markdown 格式，支持编辑/预览切换。打开时自动填入默认模板 |

Markdown 编辑/预览切换使用圆角分段控件（编辑 / 预览），预览模式使用 MDXRenderer 实时渲染。

**使用说明默认模板**：

```markdown
# 功能特点
此MCP具备的功能,比如：天气的MCP服务，支持天气的按小时查询、按天查询等功能

# 在 Openclaw 中使用
在 Openclaw 中添加mcp.json：

## 远程服务（Streamable HTTP / SSE）
{
  "mcp": {
    "servers": {
      "your-server-name": {
        "transport": "streamable-http",
        "url": "MCP服务的URL",
        "headers": {
          "Authorization": "<your-token>"
        },
        "timeout": 60
      }
    }
  }
}

## 本地命令（STDIO）
{
  "mcp": {
    "servers": {
      "your-server-name": {
        "transport": "stdio",
        "command": "python3",
        "args": ["/opt/mcp/your-server.py"],
        "env": {
          "PYTHONUNBUFFERED": "1"
        },
        "cwd": "/path/to/your/workdir",
        "timeout": 60
      }
    }
  }
}
```

**工具说明默认模板**：

```markdown
# 工具1：工具1的名称
功能：工具1具备的功能

---

参数：
* 参数1（必填）：参数1的详细内容
* 参数2（必填）：参数2的详细内容

| 参数 | 是否必填 | 内容 |
|------|-----|-----------|
| 参数1 | 必填 | 参数1的详细内容 |
| 参数2 | 必填 | 参数2的详细内容 |
```

---

### 1.6 校验规则

新增时，前端需进行以下校验：

| 校验项 | 规则 | 提示文案 |
|--------|------|---------|
| 服务ID | 不能为空 | 请输入服务ID |
| 服务ID 唯一性 | 不可与已有 MCP 的服务ID 重复 | 该服务ID 已存在，请使用其他标识 |
| 服务ID 格式 | 仅支持英文字母、数字、中划线、下划线 | 服务ID 仅支持英文、数字、中划线、下划线 |
| 连接方式 | 必须选择一项 | 请选择连接方式 |
| JSON 格式 | 必须是合法 JSON | JSON 格式错误，请检查 |
| 服务器配置 | 至少配置一个服务器 | 请至少配置一个服务器 |
| `transport` | 必须与所选连接方式匹配 | transport 与连接方式不一致（期望 "{选择的方式}"，实际 "{配置中的方式}"） |
| `url`（SSE / Streamable HTTP） | 必须以 http 或 https 开头 | URL 必须以 http 或 https 开头 |
| `command`（STDIO） | 不能为空 | 请输入可执行命令 |

---

### 1.7 数据持久化

- 使用 localStorage 缓存 MCP 列表数据和下发记录
- 缓存带版本管理，版本号变更时自动清除旧缓存并重新加载 mock 数据
- 下发记录通过 distributionCache 模块统一管理，支持跨组件事件通知

---

## 二、技术实现

### 2.1 文件结构

```
client/src/pages/admin/
├── AgentToolLibrary.tsx          # 新增「企业MCP库」Tab 入口
└── SkillLibrary/
    ├── MCPListTab.tsx            # MCP 列表页（新增）
    ├── MCPAddDialog.tsx          # 新增 MCP 弹窗（新增）
    ├── MCPDetail.tsx             # MCP 详情页（新增）
    ├── BatchDistributeDialog.tsx # 批量下发弹窗（复用，新增 MCP 专属参数）
    ├── DeleteSkillDialog.tsx     # 删除确认弹窗（复用）
    └── types.ts                  # 新增 MCP 类型定义
```

### 2.2 新增 / 修改的文件

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `AgentToolLibrary.tsx` | 修改 | TABS 数组新增 `mcp` Tab，渲染 `MCPListTab` |
| `MCPListTab.tsx` | 新增 | MCP 列表页，含 mock 数据、缓存管理、列表/卡片视图 |
| `MCPAddDialog.tsx` | 新增 | 两步表单、固化 JSON 编辑器、Markdown 编辑预览 |
| `MCPDetail.tsx` | 新增 | 详情页，三栏文件浏览 + 下发记录 Tab |
| `BatchDistributeDialog.tsx` | 修改 | 新增 `hideCreatorAndGroup`、`singleStatusFilter`、`descriptionNode` 属性以支持 MCP 场景 |
| `DeleteSkillDialog.tsx` | 修改 | 文案适配 MCP（「MCP」替换「Skill」） |
| `types.ts` | 修改 | 新增 `MCPTransportType`、`MCP_TRANSPORT_MAP`、`MCPService` 接口（含 `name` 服务ID 和 `displayName` 名称） |

### 2.3 组件复用策略

企业 MCP 库最大程度复用了企业技能库/插件库的已有组件和交互模式：

| 复用组件 | 原组件 | 差异化配置 |
|---------|--------|-----------|
| BatchDistributeDialog | 技能库下发弹窗 | `hideCreatorAndGroup=true`（隐藏创建人和组织）、`singleStatusFilter=true`（单选下拉替代多选）、自定义 `descriptionNode` |
| DeleteSkillDialog | 技能库删除弹窗 | 文案适配 MCP |
| distributionCache | 技能库下发缓存 | 直接复用，共享事件通知机制 |
| MDXRenderer | Markdown 渲染器 | 直接复用 |

---

## 附录：三种连接方式对比

| 维度 | SSE | Streamable HTTP | STDIO |
|------|-----|-----------------|-------|
| 协议版本 | MCP 2024-11-05（旧版兼容） | MCP 2025-06-18（推荐） | - |
| 适用场景 | 远程服务（兼容旧版，新接入建议用 Streamable HTTP） | 远程服务（新版标准） | 本地命令 |
| 连接方式 | 通过 SSE URL 连接 | 通过 HTTP URL 连接 | 启动本地进程，通过标准输入输出通信 |
| `transport` 值 | `"sse"` | `"streamable-http"` | `"stdio"` |
| 必填字段 | `url` · `transport` | `url` · `transport` | `command` · `transport` |

---

## 附录：服务配置完整字段参考

### SSE 配置

| 字段 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `transport` | ✅ | `string` | 固定值 `"sse"` |
| `url` | ✅ | `string` | MCP 服务端点 URL，必须以 http 或 https 开头（常见以 `/sse` 结尾） |
| `headers` | — | `object` | 请求头键值对。如 MCP Server 要求 Token 认证，在此填写（如 `"Authorization": "<your-token>"`）；否则可删除整段 |
| `security_zone` | — | `string` | 安全区域标识。如 MCP 部署在 DevCloud 环境，填写 `"devnet"` |
| `timeout` | — | `number` | 超时时间，单位秒，默认 60 |
| `username` | — | `string` | 用户标识，用于服务端区分调用方 |

### Streamable HTTP 配置

| 字段 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `transport` | ✅ | `string` | 固定值 `"streamable-http"` |
| `url` | ✅ | `string` | MCP 服务端点 URL，必须以 http 或 https 开头（常见以 `/mcp` 结尾） |
| `headers` | — | `object` | 请求头键值对。如 MCP Server 要求 Token 认证，在此填写（如 `"Authorization": "<your-token>"`）；否则可删除整段 |
| `security_zone` | — | `string` | 安全区域标识。如 MCP 部署在 DevCloud 环境，填写 `"devnet"` |
| `timeout` | — | `number` | 超时时间，单位秒，默认 60 |
| `username` | — | `string` | 用户标识，用于服务端区分调用方 |

### STDIO 配置

| 字段 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `transport` | ✅ | `string` | 固定值 `"stdio"` |
| `command` | ✅ | `string` | 可执行文件路径，支持绝对路径（如 `/usr/bin/python3`）或相对路径（如 `uvx`，需在 PATH 中） |
| `args` | — | `string[]` | 传给命令的参数数组，没有可留空 `[]` |
| `env` | — | `object` | 启动时的环境变量键值对（如 `"PYTHONUNBUFFERED": "1"`），没有可整段删除 |
| `cwd` | — | `string` | 子进程工作目录，默认继承 Agent 目录 |
| `timeout` | — | `number` | 超时时间，单位秒，默认 60 |
