# Hatchery 多 Agent 兼容改造方案

> **范围**：hatchery 后端 + `scripts/` 脚本 + 前端文案清理。
> **AgentType**：`openclaw` / `hermes` / `lightclawace`。
> **核心原则**：Hermes / ACE 对齐 OpenClaw —— 业务层 **0 if-else 分支**，仅通过「能力矩阵」+「脚本分派表」+「通道白名单」三张表驱动差异化。

---

## 一、需求总表

| # | 端 | 需求 | AgentType | 实现路径 |
|---|---|---|---|---|
| R1 | 管控端 | 进入 Agent 终端按钮 | All | 复用 `status.Actions`，终端仅由站点开关 `config.TerminalEnabled` 控制，三端 agentType 均直接放行 |
| R2 | 管控端 | 默认模型下发 | All | 矩阵 `SupportsModel=true` + `ResolveScript("set_model", type)` |
| R3 | 管控端 | 初始技能包下发 | All | 矩阵 `SupportsSkill=true` + `installSkillsAsync` 透传 agentType |
| R4 | 管控端 | Skill + Soul 下发 | All | Skill 走 R3；Soul 统一在 `llm_proxy.go` 注入 system message（无 TAT 脚本） |
| R5 | 管控端 | 企业技能库批量下发 | All | 矩阵 `SupportsSkill=true` + `ResolveScript("install_skill_from_smh", type)` |
| R6 | 员工端 | 模型配置 | All | 同 R2 |
| R7 | 员工端 | 通道配置 + 类型白名单 | All | `AgentTypeChannelAllowed` + 响应下发白名单 + 分派脚本 |
| R8 | 员工端 | 安装 skill | All | 同 R3 |

**不在本次范围内的功能**（按矩阵默认值即可拦截，无需新写业务代码）：

- **Plugin**（Hermes ❌ / ACE ❌）：`SupportsPlugin=false`；`createPluginInstallTasks` 内部已对非支持类型直接 `return`（[openclaw_plugin.go:93](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_plugin.go)），源码已自洽。
- **Reinstall 重装**（Hermes ❌ / ACE ❌）：`SupportsReinstall=false`；在 `HandleResetInstance` / `HandleAdminResetInstance` 前置 agentType 拦截。
- **Chatbot**（Hermes ❌ / ACE ✅）：Hermes 由矩阵默认拦截；ACE 需确认 `HandleChatbot*` 系列接口脚本与 OpenClaw 契约对齐（见下文脚本清单待核查项）。
- **Memory**（Hermes ❌ / ACE ❌）：`SupportsMemory=false`；`MemoryTDAIPlugin` 相关逻辑需跳过非支持类型（现有创建链路 [openclaw.go:1306](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw.go) `EnsureMemoryTDAIPluginRow` 需按 `AgentTypeSupportsMemory` 前置判断）。
- **SMH 个人空间**（Hermes ✅ / ACE ✅）：**不再是拦截项**，三端矩阵 `SupportsSMH=true`；创建链路 [openclaw.go:1331](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw.go) `config.SMHAutoProvisionOnCreate && AgentTypeSupportsSMH(agentType)` 天然适配，**SMH 环境初始化脚本**需新增 Hermes/ACE 版本。

---

## 二、通道白名单

| Channel | OpenClaw | Hermes | LightclawACE |
|---|:-:|:-:|:-:|
| wecom (企业微信) | ✅ | ✅ | ✅ |
| dingtalk (钉钉) | ✅ | ✅ | ❌ |
| feishu (飞书) | ✅ | ✅ | ✅ |
| discord | ❌ | ❌ | ❌ |
| qq | ✅ | ✅ | ✅ |
| qqbot | ✅ | ✅ | ✅ |
| weixin / openclaw-weixin (个微) | ✅ | ✅ | ✅ |

> **已移除**：`lightclawbot`（三端均不支持）、`dashboard`（三端均不支持）、`yuanbao`（三端均不支持）—— 白名单中不再出现。

**自动扫码支持度**（`HandleAutoChannel` feature 分派）：

| Feature | openclaw | hermes | lightclawace |
|---|:-:|:-:|:-:|
| `qq_bot_creator` | ✅ | ❌ | ❌ |
| `feishu_bot_creator` | ✅ | ✅ | ✅ |
| `weixin_bot_creator` | ✅ | ✅ | ✅ |

> **说明**：
> - Hermes/ACE 允许**配置 QQ 通道**（白名单 ✅），但**不支持自动扫码创建**（自动扫码表 ❌）——用户走手动填写 QQ 号/Token 的表单。
> - **ACE 飞书扫码**：已在 `lightclaw-dashboard/lightclaw-ace/channel/feishu/feishu_bot_creator.py` 实现完整 Python 流程（JSON Lines 输出、qrcode 事件），hatchery 侧仅需新增薄 wrapper `scripts/feishu_bot_creator_ace.sh` 调用该 Python 脚本。

---

## 三、能力矩阵（`model/agent_type.go`）

### 3.1 字段定义

```go
type AgentType struct {
    Code              string
    Name              string
    Description       string
    SupportsRole      bool
    SupportsModel     bool
    SupportsChannel   bool
    SupportsSkill     bool
    SupportsPlugin    bool
    SupportsChatbot   bool
    SupportsSMH       bool
    SupportsMemory    bool
    SupportsReinstall bool   // 新增：是否允许重装（ResetInstance），仅 OpenClaw 为 true
    SortOrder         int
}
```

> **为什么没有 `SupportsTerminal`？** 终端是三端均支持的基础能力，且实际开关由站点级 `config.TerminalEnabled` 控制（同一开关对所有 agentType 生效），新增维度纯冢余，因此不引入该字段也不引入对应 guard。

### 3.2 矩阵取值

| 能力 | openclaw | hermes | lightclawace |
|---|:-:|:-:|:-:|
| Role | ✅ | ✅ | ✅ |
| Model | ✅ | ✅ | ✅ |
| Channel | ✅ | ✅ | ✅ |
| Skill | ✅ | ✅ | ✅ |
| Plugin | ✅ | ❌ | ❌ |
| Chatbot | ✅ | ❌ | ✅ |
| SMH | ✅ | ✅ | ✅ |
| Memory | ✅ | ❌ | ❌ |
| Reinstall | ✅ | ❌ | ❌ |

### 3.3 辅助函数

在 [agent_type.go](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/model/agent_type.go) 追加：

```go
func AgentTypeSupportsReinstall(code string) bool {
    t := GetAgentTypeByCode(code)
    if t == nil { return false }
    return t.SupportsReinstall
}

// 单 channel 类型白名单：检查某 channel 是否允许用于某 agent_type
// 规则：
//   1. agent_type 不在 whitelist map 中 → false（fail-closed）
//   2. whitelist[agent_type] 为 nil → true（表示该 agent 允许所有 channel，如 openclaw）
//   3. 否则 whitelist[agent_type] 是允许的 channel 列表，走严格匹配
// 注意：openclaw 历史上全放行，但按最新决策 discord/lightclawbot/dashboard 三端都不支持，
// 因此 openclaw 也改为显式白名单（移除全放行 nil 语义）。
var agentTypeChannelWhitelist = map[string]map[string]bool{
    AgentTypeOpenClaw:     {"wecom": true, "dingtalk": true, "feishu": true, "qq": true, "qqbot": true, "weixin": true, "openclaw-weixin": true},
    AgentTypeHermes:       {"wecom": true, "dingtalk": true, "feishu": true, "qq": true, "qqbot": true, "weixin": true},
    AgentTypeLightclawACE: {"wecom": true, "feishu": true, "qq": true, "qqbot": true, "weixin": true},
}

func AgentTypeChannelAllowed(agentType, channelID string) bool {
    wl, ok := agentTypeChannelWhitelist[agentType]
    if !ok { return false }
    return wl[channelID]
}

func SupportedChannelsByAgentType(agentType string) []string {
    wl, ok := agentTypeChannelWhitelist[agentType]
    if !ok { return nil }
    result := make([]string, 0, len(wl))
    for k := range wl { result = append(result, k) }
    return result
}
```

---

## 四、脚本分派表（`controller/script_registry.go`）

### 4.1 Feature → Script 映射

```go
// agentType 对 feature 无支持时 map 值为 ""。
// ResolveScript 返回 "" 时调用方必须报错（403/400），禁止继续执行。
var scriptResolveTable = map[string]map[string]string{
    "set_model": {
        AgentTypeOpenClaw:     "set_model.sh",
        AgentTypeHermes:       "set_model_hermes.sh",
        AgentTypeLightclawACE: "set_model_ace.sh",
    },
    "set_channel": {
        AgentTypeOpenClaw:     "set_channel.sh",
        AgentTypeHermes:       "set_channel_hermes.sh",
        AgentTypeLightclawACE: "set_channel_ace.sh",
    },
    "del_channel": {
        AgentTypeOpenClaw:     "del_channel.sh",
        AgentTypeHermes:       "del_channel_hermes.sh",
        AgentTypeLightclawACE: "del_channel_ace.sh",
    },
    "list_channels": {
        AgentTypeOpenClaw:     "list_channels.sh",
        AgentTypeHermes:       "list_channels_hermes.sh",
        AgentTypeLightclawACE: "list_channels_ace.sh",
    },
    "add_skill": {
        AgentTypeOpenClaw:     "add_skill.sh",
        AgentTypeHermes:       "add_skill_hermes.sh",
        AgentTypeLightclawACE: "add_skill_ace.sh",
    },
    "batch_install_skills_from_smh": {
        AgentTypeOpenClaw:     "batch_install_skills_from_smh.sh",
        AgentTypeHermes:       "batch_install_skills_from_smh_hermes.sh",
        AgentTypeLightclawACE: "batch_install_skills_from_smh_ace.sh",
    },
    "install_skill_from_smh": {
        AgentTypeOpenClaw:     "install_skill_from_smh.sh",
        AgentTypeHermes:       "install_skill_from_smh_hermes.sh",
        AgentTypeLightclawACE: "install_skill_from_smh_ace.sh",
    },
    "qq_bot_creator": {
        AgentTypeOpenClaw: "qq_bot_creator.sh",
        // hermes/ace 不支持 QQ 自动扫码
    },
    "feishu_bot_creator": {
        AgentTypeOpenClaw:     "feishu_bot_creator.sh",
        AgentTypeHermes:       "feishu_bot_creator_hermes.sh",
        AgentTypeLightclawACE: "feishu_bot_creator_ace.sh",
    },
    "weixin_bot_creator": {
        AgentTypeOpenClaw:     "weixin_bot_creator.sh",
        AgentTypeHermes:       "weixin_bot_creator_hermes.sh",
        AgentTypeLightclawACE: "weixin_bot_creator_ace.sh",
    },
}

// ResolveScript 将 (feature, agentType) 映射为真实脚本名。
// 空字符串视为 openclaw（兼容存量数据）。
func ResolveScript(feature, agentType string) string {
    if agentType == "" { agentType = model.AgentTypeOpenClaw }
    m, ok := scriptResolveTable[feature]
    if !ok { return "" }
    return m[agentType]
}
```

### 4.2 Loader 复用

`main.go:185 controller.LoadScript = loadScript` 已存在；`LoadScript` 先查 `InlineScriptRegistry`（内联脚本优先，供飞书 JSON Lines 等场景动态注入），未命中再读 `scripts/<name>` 磁盘文件。此机制复用 —— **0 新增 loader 代码**。

---

## 五、Soul 注入（`llm_proxy.go`）

### 5.1 机制

Soul 不生成 TAT 脚本，统一在 LLM 代理层注入为 `messages[0]` 的 `system` 消息：

```go
// 插入位置：HandleLLMProxy 中 max_tokens 块末 `}` 之后、`isStreaming := false` 之前。
// instance 为值类型 (model.Instance)，签名必须以值形参接收。
injectSystemPrompt(reqBody, instance)
```

### 5.2 实现要点

- Soul 文本来源：`instance.RoleID` → `OpenClawRole.Soul`，经 LRU 缓存（键 `RoleID`，TTL 60s）降低 DB 压力。
- 如果 `messages[0].role == "system"`，则 **前置插入** 一条新的 system message，而非合并/覆盖（保证用户自定义 system 优先级更高）。
- 三端 AgentType 完全同机制；**Hermes/ACE 不支持直连绕过 hatchery 代理的模式** —— 只要用户走 hatchery，Soul 必然生效。

### 5.3 缓存失效钩点

在 3 个写 Role 入口调用 `InvalidateSoulCache(roleID)`：

| 文件 | 函数 | 插入位置 |
|---|---|---|
| [admin_roles.go:126](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/admin_roles.go) | `HandleCreateRole` | `Create` 成功之后、`jsonOK` 之前 |
| [admin_roles.go:282](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/admin_roles.go) | `HandleUpdateRole` | `txErr` 分支处理之后、`go syncRoleSkillsCosZipKey` 之前 |
| [admin_roles.go:454](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/admin_roles.go) | `HandleDeleteRole` | `Delete` 成功之后、`jsonOK` 之前 |

---

## 六、Controller 改造点清单

| # | 文件 | 函数 | 改动 |
|---|---|---|---|
| C1 | [openclaw.go:1479](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw.go) | `HandleInstanceStatus` | 删除原有「`agentType==openclaw` 才保留 terminal action」分支；改为仅根据 `config.TerminalEnabled` 决定是否下发 terminal action，三端 agentType 同一处理 |
| C2 | [openclaw.go:1947](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw.go) | `HandleInstanceTerminal` | 保留原有 `config.TerminalEnabled` 判断；删除原有对 agentType 的硬编码判断（若有） |
| C3 | [openclaw.go:1629](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw.go) | `HandleResetInstance` | `getInstanceByID` 之后加 agentType 拦截：`!AgentTypeSupportsReinstall` 返回 403 |
| C4 | [admin_instances.go:1405](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/admin_instances.go) | `HandleAdminResetInstance` | 查询 instance 后加 `!AgentTypeSupportsReinstall` 拦截返回 403 |
| C5 | [openclaw_model.go](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_model.go) | `HandleSetModel` / `handleCustomModel` / `injectDefaultModel` | 3 处 `RunScript(..., "set_model.sh", ...)` 替换为 `RunScript(..., ResolveScript("set_model", instance.AgentType), ...)` |
| C6 | [openclaw_skill.go](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_skill.go) | `HandleAddSkill` | `RunScript(..., "add_skill.sh", ...)` 替换为 `ResolveScript("add_skill", instance.AgentType)` |
| C7 | [openclaw_skill.go:352](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_skill.go) | `installSkillsAsync` | 签名追加 `agentType string` 形参；内部 `batch_install_skills_from_smh.sh` 替换为 `ResolveScript(...)` |
| C8 | [openclaw_skill.go:185](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_skill.go) | `HandleRetryFailedSkills` | `installSkillsAsync` 调用补 `instance.AgentType` |
| C9 | [openclaw.go:1323](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw.go) | 创建流程 | `installSkillsAsync` 调用补 `agentType` 参数（同一作用域已有变量） |
| C10 | [openclaw.go:1740](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw.go) | 重装流程 | `installSkillsAsync` 调用补 `instance.AgentType` |
| C11 | [openclaw_channel.go:14](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_channel.go) | `HandleSetChannel` | guard 之后、`RunScript` 之前追加 `AgentTypeChannelAllowed` 检查；`RunScript` 脚本名走 `ResolveScript("set_channel", ...)` |
| C12 | [openclaw_channel.go:111](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_channel.go) | `HandleDelChannel` | **已补 `checkInstanceSupportsChannel` guard**；脚本名走 `ResolveScript("del_channel", ...)` |
| C13 | [openclaw_channel.go:155](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_channel.go) | `listInstanceChannels` | 签名追加 `agentType string`；调用端 [openclaw_channel.go:195](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_channel.go) 透传 `instance.AgentType`；内部走 `ResolveScript("list_channels", ...)` |
| C14 | [openclaw_channel.go:204](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw_channel.go) | `HandleAutoChannel` | **重写 switch** 为 feature 分派表 + `ResolveScript`；`ResolveScript` 返回空字符串时（例如 ACE+qqbot）走 `writeError 400`（此时 SSE header 尚未设置，允许普通 JSON 错误响应） |
| C15 | [admin_skills.go](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/admin_skills.go) | `HandleDistributeSkill` | `instInfos` 同时建立 `atMap[ID]→AgentType`；闭包 goroutine 内按 atMap 查出 agentType，走 `ResolveScript("install_skill_from_smh", at)` |
| C16 | [openclaw.go:1888](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/openclaw.go) | `HandleCheckOpenclawPort` | 改名 `HandleCheckAgentReady`；按 agentType 路由到 `check_openclaw_ready.sh` / `check_hermes_ready.sh` / `check_ace_ready.sh` |
| C17 | [llm_proxy.go](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/llm_proxy.go) | `HandleLLMProxy` | max_tokens 块之后插入 `injectSystemPrompt(reqBody, instance)` |
| C18 | [admin_roles.go](/Users/yingningchen/Bo5hengProjects/clawpro/hatchery/controller/admin_roles.go) | `HandleCreateRole` / `HandleUpdateRole` / `HandleDeleteRole` | 3 处在写成功后调用 `InvalidateSoulCache` |

---

## 七、Guard 清单（`controller/agent_type_guard.go`）

| Guard 函数 | 对应能力 | 使用位置 |
|---|---|---|
| `checkInstanceSupportsModel` | Model | `HandleSetModel`（已挂） |
| `checkInstanceSupportsChannel` | Channel | `HandleSetChannel`（已挂）、`HandleDelChannel`（**本次补挂**） |
| `checkInstanceSupportsSkill` | Skill | `HandleAddSkill`（已挂） |
| `checkInstanceSupportsPlugin` | Plugin | `HandleAddPlugin`（已挂） |
| `checkInstanceSupportsChatbot` | Chatbot | `HandleChatbotXxx`（已挂） |
| `checkInstanceSupportsDetailConfig` | 综合 | 用于详情页配置入口 |
| `checkInstanceSupportsRole` | Role | （若需要在绑定角色时拦截，按需补挂） |
| `checkInstanceSupportsReinstall` | **新增** | `HandleResetInstance` / `HandleAdminResetInstance` |

> 所有 guard 统一签名 `func(ctx context.Context, instance *model.Instance) error`，错误信息使用 `GetAgentTypeDisplayName` 本地化。

---

## 八、脚本清单（`scripts/`）

### 8.1 现有已落盘（✅）

| Feature | OpenClaw | Hermes | ACE |
|---|---|---|---|
| set_model | `set_model.sh` ✅ | `set_model_hermes.sh` ✅ | `set_model_ace.sh` ✅ |
| set_channel | `set_channel.sh` ✅ | `set_channel_hermes.sh` ✅ | `set_channel_ace.sh` ✅ |
| del_channel | `del_channel.sh` ✅ | `del_channel_hermes.sh` ✅ | `del_channel_ace.sh` ✅ |
| list_channels | `list_channels.sh` ✅ | `list_channels_hermes.sh` ✅ | `list_channels_ace.sh` ✅ |
| add_skill | `add_skill.sh` ✅ | `add_skill_hermes.sh` ✅ | `add_skill_ace.sh` ✅ |
| batch_install_skills_from_smh | `batch_install_skills_from_smh.sh` ✅ | `batch_install_skills_from_smh_hermes.sh` ✅ | `batch_install_skills_from_smh_ace.sh` ✅ |
| qq_bot_creator | `qq_bot_creator.sh` ✅ | — | — |
| feishu_bot_creator | `feishu_bot_creator.sh` ✅ | `feishu_bot_creator_hermes.sh` ✅ | **待落盘** |
| weixin_bot_creator | `weixin_bot_creator.sh` ✅ | `weixin_bot_creator_hermes.sh` ✅ | `weixin_bot_creator_ace.sh` ✅ |
| check_ready | `check_openclaw_ready.sh` ✅ | `check_hermes_ready.sh` ✅ | `check_ace_ready.sh` ✅ |

### 8.2 需新增落盘

| 脚本 | 说明 |
|---|---|
| `install_skill_from_smh_hermes.sh` | 企业技能库单技能下发 —— Hermes 版本，从 SMH 下载 zip 后调用 `add_skill_hermes.sh` 逻辑 |
| `install_skill_from_smh_ace.sh` | 同上 —— ACE 版本 |
| `feishu_bot_creator_ace.sh` | Wrapper，调用 `lightclaw-dashboard/lightclaw-ace/channel/feishu/feishu_bot_creator.py`，保持 stdout JSON Lines 契约（`show_qrcode` / `log` / `progress` / `finish`） |

### 8.3 脚本契约

- **TAT 参数**：统一使用 `{{placeholder}}` 模板渲染；`add_skill` 系列统一接收 `{{skill_name}}`（skill slug）。
- **ACE 飞书脚本契约**：ACE 版本 Python 脚本输出行已对齐 hatchery SSE 解析（见 `openclaw_channel.go::newJSONLinesHandler`），无需额外适配。
- **日志**：所有脚本向 `/var/log/clawpro/<script_name>.log` 输出，失败时 exit 非 0。

---

## 九、前端改造点

### 9.1 文案清理

搜索并清除所有形如「仅限 OpenClaw / 暂不支持 / OpenClaw Only」的提示：

- **默认模型页**：去掉副标题「（仅限 OpenClaw，其他 Agent 暂不支持）」。
- **通道配置页**：不再根据 agent_type 硬隐藏整块；改用白名单响应字段置灰不支持的 channel 图标。
- **技能下发页**：批量下发时，若选中实例含不支持 skill 的类型（理论上此方案后 Hermes/ACE 都支持），后端也会自动过滤，UI 只需显示 skipped 数量。
- **重装按钮**：在实例列表行，根据 agent_type 控制「重装」按钮是否显示（Hermes/ACE 隐藏）。前端可用 `status.Actions` 响应判断。

### 9.2 通道白名单消费

新增实例详情接口 / 或实例列表接口响应字段：

```json
{
  "agent_type": "hermes",
  "supported_channels": ["wecom", "dingtalk", "feishu", "discord", "weixin", "lightclawbot"]
}
```

前端据此隐藏或置灰不支持的通道入口。

### 9.3 ACE 飞书自动扫码

前端对 ACE + feishu 组合不做降级：直接复用 openclaw 的二维码 SSE 流即可。

---

## 十、实施顺序（5 个 PR）

| PR | 内容 | 独立运行可行性 |
|---|---|---|
| **PR1** | 能力矩阵扩展（新增 `SupportsReinstall` 字段与辅助函数）+ 通道白名单 + 单测 | ✅ 无调用侧改动，0 行为变化 |
| **PR2** | `script_registry.go::ResolveScript` + 分派表 + 单测 | ✅ 未被调用，0 行为变化 |
| **PR3** | Soul 注入（`llm_proxy.go::injectSystemPrompt` + `admin_roles.go` 3 处缓存失效） | ✅ 独立功能，风险低 |
| **PR4** | 脚本落盘（3 个新脚本：`install_skill_from_smh_{hermes,ace}.sh`、`feishu_bot_creator_ace.sh`） | ✅ 脚本无引用前不生效 |
| **PR5** | Controller 改造 C1~C18（建议再拆 5 个子 PR：终端/模型/技能/通道/重装&下发） | 依赖 PR1~PR4 |

---

## 十一、风险与回滚

| 风险 | 缓解 | 回滚手段 |
|---|---|---|
| Hermes/ACE 线上实例首次被下发命令 | 先灰度 1~2 台，跑 10 项 E2E 场景 | 矩阵字段改 `false`，秒级收敛 |
| ACE 飞书 Python 依赖未就绪 | wrapper 脚本首步检查并走 `feishu_bot_creator.py init` | 矩阵 `feishu_bot_creator[ACE]` 置空即自动降级为 400 |
| Soul LRU 缓存不一致 | 3 个写 Role 入口必挂失效 | 调整缓存 TTL=0（等同透传） |
| Reinstall 拦截误伤管理员排障 | 管控端接口仍走 `AgentTypeSupportsReinstall` 拦截，错误信息建议「Hermes/ACE 不支持重装，请删除后重建」 | 矩阵 `SupportsReinstall=true` 可临时开放 |

---

## 十二、E2E 验证场景（灰度用）

对每个 agent_type（openclaw / hermes / lightclawace）各跑一遍：

1. 创建实例 → 默认模型成功注入
2. 绑定 DB 已有 model → `set_model` 成功
3. 绑定自定义 model → `handleCustomModel` 成功
4. 角色技能 + 全局技能包 → `installSkillsAsync` 成功（OpenClaw 含 plugin，Hermes/ACE 跳过 plugin）
5. 单技能手动安装 → `HandleAddSkill` 成功
6. 企业技能库批量下发 → `HandleDistributeSkill` 跳过不支持 skill 的实例（本方案后 3 端均应成功）
7. 配置 channel（白名单内的）→ `HandleSetChannel` 成功
8. 配置 channel（白名单外的）→ 400 返回正确错误
9. 删除 channel → `HandleDelChannel` 成功（含 guard 拦截测试）
10. 自动扫码 feishu → SSE 正常推送二维码，最终 finish
11. 进入 Agent 终端 → URL 正常返回（三端统一受 `config.TerminalEnabled` 控制）
12. 重装实例（OpenClaw 应成功，Hermes/ACE 应被拦截返回 403）
13. LLM 代理请求 → 注入 system=Soul 验证
14. 修改 Role → 下次请求 Soul 立即生效

---

## 十三、开发对齐清单（Front / Back / Ops）

**后端**：按第六章 C1~C18 + 第七章 Guard + 第五章 Soul 逐条落地。

**前端**：
- 清理所有「仅限 OpenClaw」硬编码文案。
- 消费 `supported_channels` 响应字段置灰不支持通道。
- 重装按钮按 `status.Actions` 响应渲染。
- ACE + feishu 不做降级，复用 OpenClaw 二维码 SSE 流程。

**Ops**：
- 灰度发布顺序：Hermes 1~2 台 → 10 台 → 全量；ACE 同步复用 Hermes 经验。
- 监控：`TAT RunCommand 失败率` 按 `script_name` 拆分维度；一旦 `*_hermes.sh` / `*_ace.sh` 成功率低于 95% 立即报警。
- 紧急回滚：`UPDATE site_config / 或代码常量`将 agent_type 矩阵所有 `Supports*` 置 false 即可阻断。

