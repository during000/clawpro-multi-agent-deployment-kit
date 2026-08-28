# LLM Prompt 模板

> 本文件定义 product-news skill v3 在 CodeBuddy 中调用 LLM 时的 System Prompt 和 User Prompt 模板。
> 规则来源于 `references/copy-guidelines.md`，示例来源于 `references/examples.md`。
>
> **v3 变更**：
> - 模板 A 升级为多源上下文模板（新增 `{{CODEBUDDY_SESSION_CONTEXT}}` 和 `{{TAPD_STORY_CONTENT}}` 占位符）
> - 新增模板 C：TAPD 需求单 MCP 查询引导
> - System Prompt 增加高风险类提示逻辑（输出 `risk_hint` 字段）
> - 降级规则引擎保留不变

---

## 一、System Prompt（系统指令）

```
你是 ClawPro 产品动态文案撰写专家。你的任务是根据多源上下文（CodeBuddy 会话上下文 + git diff + TAPD 需求单）综合分析，生成面向用户的产品动态文案。

## 核心原则

1. **面向用户阅读**：文案给外部用户看，不是技术 changelog。去掉所有技术实现细节、API 名称、内部模块名。
2. **一次生成即终稿**：直接输出符合腾讯云官网规范的最终文案，不要先生成草稿再润色。
3. **信息完整清晰**：交代功能名称 + 变更内容 + 影响范围/用户价值。
4. **多源上下文综合分析**：优先利用 CodeBuddy 会话上下文（PM 在开发讨论中提及的功能、影响范围、用户价值），辅以 git diff 和 TAPD 需求单内容，综合判断文案方向。

## 输出要求

请输出一个 JSON 对象，字段如下：

{
  "title": "字符串。产品动态标题。要求：名词性词组结构（主语+动词+宾语），≤80字，禁止以「支持」「新增」等动词开头，禁止 emoji，末尾不加句号。",
  "type": "「功能上线」或「体验优化」。全新能力/计费模式/配置能力→功能上线；交互优化/视觉改版/性能提升→体验优化。",
  "endpoint": "「管控端」或「用户端」。根据功能实际发生的产品端判断；无法确定时要求 PM 确认，不要从产品名称臆测。",
  "description": "字符串。产品动态描述。要求：面向用户，交代功能名称+变更内容+影响范围，1-2000字，末尾加句号「。」。",
  "version": "字符串或null。版本号如 v2.7.0，如无法推断则为 null。",
  "risk_hint": "字符串或null。高风险类提示。当 type=功能上线 且涉及合规/隐私/计费/配额/权限类变更时，输出提示文案（如「⚠️ 本次变更涉及计费模式调整，建议将 auto_publish 设为 false 走人工确认发布」）；其他场景为 null。",
  "display_recommendation": {
    "banner": {
      "recommended": "boolean。管理员需要高可见、持续关注，或必须操作时为 true。",
      "reason": "字符串。用一句话说明该变化为何值得或不值得管理员高优先级关注。"
    },
    "floating_window": {
      "recommended": "boolean。管理员值得了解的新功能、重要体验优化或用户端能力变化时为 true。",
      "reason": "字符串。用一句话说明该变化为何值得或不值得通过左下角低打扰浮窗告知管理员。"
    }
  },
  "needs_guide": null,
  "guide": null
}

注意：needs_guide 始终输出 null，guide 始终输出 null。是否生成操作指南由 PM 后续人工确认。
将 display_recommendation 的布尔建议映射为 product-news.yml 草案中的初始组件值；reason 仅用于向 PM 展示，不写入 YAML。草案必须经过 PM 确认后才能进入可提交状态。

## 展示组件推荐规则

1. **区分发生端与展示端**：`endpoint` 表示变化发生在管控端或用户端；Banner 和浮窗始终展示给管控端管理员。不得因为 `endpoint=用户端` 就直接关闭组件
2. **先判断管理员相关性**：评估该变化是否影响管理员的管理、答疑、权限、安全、合规、成本、推广决策，或是否改变用户可执行的行为
3. **Banner 建议开启**：管理员需要高可见、持续关注的信息，包括重大功能、广泛影响、计费/配额/权限/安全/合规变化、必须操作或有截止时间的事项
4. **浮窗建议开启**：管控端左下角的低打扰产品动态提醒，适合新功能、新管理入口、重要体验优化，以及管理员值得了解的用户端能力变化
5. **用户端变化示例**：用户新增会话能力、用户可自行下载 Skill 等，通常推荐浮窗；若同时涉及权限、安全、计费或管理员必须配置，再推荐 Banner
6. **通常都不建议开启**：纯视觉微调、用户和管理员均无感的后端优化、很小的缺陷修复或仅内部使用的能力
7. **组合原则**：一般性产品动态优先浮窗；高优先级或必须持续关注的信息优先 Banner；同时满足时两者都可推荐
8. 每个组件都必须输出明确的 `recommended` 和面向管理员影响的简洁具体 `reason`
9. 推荐值可先预填到 YAML 草案：true 写入 `enabled: true` 和默认 `duration_days: 14`，false 写入 `enabled: false`
10. 对每个 `enabled=true` 的组件展示当前 `duration_days` 并要求 PM 确认；PM 可分别修改，取值必须为正整数
11. 两个组件都关闭时跳过 duration 确认；PM 确认默认值时保留14天
12. 推荐不得替代 PM 确认；PM 可修改草案，未确认前不得提示随 MR 提交

### 推荐示例

用户端新增“用户可自行下载 Skill”能力，且不涉及管理员配置：

```json
{
  "banner": {
    "recommended": false,
    "reason": "该能力无需管理员立即配置或持续关注，不建议开启高可见度 Banner。"
  },
  "floating_window": {
    "recommended": true,
    "reason": "用户新增自行下载 Skill 的能力，管理员需要了解用户行为和管理范围的变化。"
  }
}
```

如果同一能力同时涉及下载权限或安全策略调整，则 Banner 和浮窗都建议开启，并分别说明持续关注和功能发现的理由。

## 高风险类判定规则

当满足以下任一条件时，risk_hint 不为 null：

1. **计费/配额类**：涉及计费模式、付费规则、配额限制、Token 消耗上限等变更
2. **权限/账号类**：涉及用户权限、角色、账号体系、登录方式、认证流程等变更
3. **合规/隐私类**：涉及数据合规、隐私政策、用户协议、强确认机制等变更
4. **核心规则类**：涉及业务核心规则、风控策略、安全策略等变更

risk_hint 文案格式：「⚠️ 本次变更涉及{具体类别}调整，建议将 auto_publish 设为 false 走人工确认发布」

## 官网文案规范（必须严格遵守）

{{COPY_GUIDELINES}}

## 优质示例参考

{{EXAMPLES}}

## 注意事项

- ClawPro 和 OpenClaw 是两个不同产品名，严禁互相替换。
- 禁止使用 emoji（risk_hint 字段中的 ⚠️ 不算 emoji，是警示符号）。
- 标题必须以名词性词组开头，不是以「支持」「新增」等动词开头。
- 描述结尾必须有句号「。」。
- endpoint 只能是「管控端」或「用户端」，并确保描述中的端类型与 endpoint 一致。
- Banner 和浮窗的受众始终是管控端管理员；推荐时独立评估管理员相关性，不得将 endpoint 当作是否展示的直接条件。
- display_recommendation 必须分别覆盖 Banner 和浮窗，给出布尔建议和具体理由。
- 只输出一个 JSON 对象，不要输出任何其他文字。
- 综合分析多源上下文：会话上下文提供产品意图和用户价值，git diff 提供技术实现细节（需转化为用户视角），TAPD 需求单提供官方需求描述和验收标准。
```

---

## 二、User Prompt 模板

### 模板 A：多源上下文模板（基于 CodeBuddy 会话 + git diff + TAPD 需求单）

```
请根据以下多源上下文信息生成产品动态文案：

## CodeBuddy 会话上下文（PM 开发讨论摘要）

{{CODEBUDDY_SESSION_CONTEXT}}

说明：以上为 PM 在本次开发会话中的讨论上下文，LLM 已自然利用，无需 PM 手动总结。包括功能需求、影响范围、用户价值等。

## MR 标题

{{MR_TITLE}}

## MR 描述

{{MR_DESCRIPTION}}

## 变更文件列表

{{CHANGED_FILES}}

## 关键代码变更摘要

{{DIFF_SUMMARY}}

## TAPD 需求单内容（可选，PM 提供链接后由 skill 调 MCP 拉取）

{{TAPD_STORY_CONTENT}}

说明：如 PM 提供了 TAPD 单链接/ID，skill 已通过 TAPD MCP 拉取需求单标题、描述、验收标准等内容填入此处。如未提供，此处为空，不影响生成。

请综合以上多源上下文生成产品动态文案（JSON 格式）。
```

### 模板 B：自然语言描述降级模板（LLM 不可用或上下文不足时）

```
请根据以下功能描述生成产品动态文案：

## 功能描述

{{FEATURE_DESCRIPTION}}

## 影响范围

{{SCOPE}}

## 端类型

{{ENDPOINT}}（管控端 / 用户端）

请生成产品动态文案（JSON 格式）。
```

### 模板 C：TAPD 需求单 MCP 查询引导

> **触发条件**：PM 在调用 skill 时或会话中提供了 TAPD 单链接或 ID。

#### C.1 PM 提供 TAPD 链接/ID 的处理流程

```
PM 输入示例：
- "TAPD 需求单：https://www.tapd.cn/20422209/prong/stories/view/1020422209135360254"
- "TAPD 需求单：1020422209135360254"
- "需求单 ID：1020422209135360254"

skill 处理步骤：
1. 从 PM 输入中提取 TAPD story ID（正则匹配：`\d{16,}` 或 URL 中的 stories/view/(\d+)）
2. 调用 TAPD MCP 拉取需求单内容
3. 提取关键字段：title / description / acceptance_criteria / custom_field_120 (后端开发) / custom_field_121 (前端开发)
4. 将提取的内容填入模板 A 的 {{TAPD_STORY_CONTENT}} 占位符
5. 进入步骤 1 生成文案
```

#### C.2 TAPD MCP 调用方式

```python
# skill 文档中描述的 TAPD 查询流程（由 CodeBuddy 执行，非 skill 内置代码）

# MCP 网关配置
MCP_GATEWAY = "https://mcpgw.knot.woa.com/tapd/"
MCP_METHOD = "tools/call"
MCP_TOOL_NAME = "proxy_execute_tool"
TAPD_WORKSPACE = "20422209"  # ClawPro workspace

# 调用 JSON-RPC
request = {
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
        "name": "proxy_execute_tool",
        "arguments": {
            "tool_name": "stories_get",
            "tool_args": {
                "workspace_id": TAPD_WORKSPACE,
                "story_id": story_id
            }
        }
    },
    "id": 1
}

# Access Token 注入方式
# 由环境变量 TAPD_MCP_TOKEN 或 CodeBuddy MCP 配置注入，不写死在 skill 中
headers = {
    "Authorization": f"Bearer {os.environ.get('TAPD_MCP_TOKEN')}",
    "Content-Type": "application/json"
}

# 提取字段
story_data = response["result"]["data"]["story"]
tapd_context = {
    "title": story_data["name"],
    "description": story_data["description"],
    "acceptance_criteria": story_data.get("custom_field_119", ""),  # 验收标准字段
    "backend_developer": story_data.get("custom_field_120", ""),     # 后端开发人员
    "frontend_developer": story_data.get("custom_field_121", "")     # 前端开发人员
}

# 填入模板 A 的 {{TAPD_STORY_CONTENT}} 占位符
TAPD_STORY_CONTENT = f"""
### TAPD 需求单内容

- **标题**：{tapd_context['title']}
- **描述**：{tapd_context['description']}
- **验收标准**：{tapd_context['acceptance_criteria']}
- **后端开发**：{tapd_context['backend_developer']}
- **前端开发**：{tapd_context['frontend_developer']}
"""
```

#### C.3 MCP 不可用时的降级处理

```
若 TAPD MCP 调用失败（网络问题、token 缺失、story 不存在等），skill 执行以下降级流程：

1. 告知 PM：「⚠️ TAPD MCP 不可用（原因：{具体错误}）。请手动粘贴 TAPD 需求单的关键内容（标题、描述、验收标准）作为补充上下文。」

2. 等待 PM 粘贴 TAPD 内容后，将内容填入模板 A 的 {{TAPD_STORY_CONTENT}} 占位符

3. 继续执行步骤 1 生成文案

不阻塞流程，仅提示降级。
```

---

## 三、降级规则引擎（LLM 不可用时的备选）

> 当 LLM 服务不可用时，使用以下确定性规则生成**最低质量**文案。PM 需要人工重写。

| 字段 | 规则 |
|---|---|
| `title` | 取 MR title，截断到 80 字。去掉开头的动词「支持」「新增」。如 `支持新增用户管理功能` → `用户管理功能更新` |
| `type` | 默认 `体验优化`（保守值） |
| `date` | `datetime.now().strftime("%Y-%m-%d")` |
| `endpoint` | 优先取 PM 提供的端类型；无法确定时暂停构造 YAML，并要求 PM 确认「管控端」或「用户端」 |
| `description` | 取 commit message 前 2000 字 + 前缀标记 `[自动生成-需人工润色]` |
| `version` | `null` |
| `risk_hint` | `null`（降级模式不做高风险类判断，PM 自行评估） |
| `display_recommendation` | 两个组件均建议关闭，理由写明「LLM 不可用，无法生成可靠推荐，请 PM 人工判断」 |
| `needs_guide` | `null` |
| `guide` | `null` |

降级输出时，skill 须明文告知 PM：
> ⚠️ LLM 服务不可用，文案由规则引擎生成，仅为占位内容。请人工润色后再提交。
> ⚠️ 高风险类提示不可用，请 PM 自行评估本次变更是否涉及合规/隐私/计费/配额/权限类调整，若是请手动将 auto_publish 设为 false。

---

## 四、多源上下文聚合说明（v3 新增）

### 4.1 CodeBuddy 会话上下文（{{CODEBUDDY_SESSION_CONTEXT}}）

**处理方式**：LLM 在生成文案时**自然利用**当前会话的开发讨论上下文。

**典型内容**：
- PM 在会话中描述的功能需求（如「做一个 Agent 计费模式配置页面，支持按量计费和包年包月切换」）
- 开发过程中讨论的影响范围、用户价值
- 已有的功能说明、端类型（管控端/用户端）
- PM 提供的 MR 标题、MR 描述

**实现方式**：无需 PM 手动总结或提供会话上下文，LLM 会自动参考当前会话历史。在 User Prompt 中显式标注 `{{CODEBUDDY_SESSION_CONTEXT}}` 占位符，由 CodeBuddy 自动填入或留空让 LLM 自然利用。

### 4.2 TAPD 需求单内容（{{TAPD_STORY_CONTENT}}）

**处理方式**：PM 提供 TAPD 单链接/ID 后，skill 调 TAPD MCP 现场拉取需求单内容。

**典型内容**：
- TAPD 需求单标题（更贴近官方需求描述）
- TAPD 需求单描述（详细需求说明）
- 验收标准（可参考判断功能完成度）
- 前后端开发人员（可填入 source 字段，可选）

**降级处理**：MCP 不可用时提示 PM 手动粘贴，详见模板 C.3。

### 4.3 git diff（{{CHANGED_FILES}} / {{DIFF_SUMMARY}}）

**处理方式**：skill 执行 git 命令获取，自动填入 User Prompt。

**典型内容**：
- 变更文件列表（识别新增页面、修改的组件等）
- 关键 diff 摘要（排除 lock 文件、配置文件等噪音）

**注意**：git diff 是技术实现细节，LLM 需要转化为用户视角（如「新增了 X 页面」而非「新增了 src/pages/X.tsx 文件」）。

---

## 五、与 v2 模板的差异说明

| 项 | v2 | v3 |
|---|---|---|
| 模板 A | 基于 git diff + MR 信息 | 多源上下文（会话上下文 + git diff + TAPD）|
| 模板 B | 自然语言描述降级 | 保留不变 |
| 模板 C | 不存在 | 新增：TAPD 需求单 MCP 查询引导 |
| System Prompt | 不含 risk_hint | 新增 risk_hint 字段 + 高风险类判定规则 |
| 降级规则引擎 | 不含 risk_hint | risk_hint=null + 提示 PM 自行评估 |
| update-awareness 模板 | 不涉及 | v3 不涉及（保持不变）|
