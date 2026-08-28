---
name: product-news
description: 产品动态文案生成工具，专为 ClawPro 等产品设计。PM 本地开发完成、提 MR 前调用，综合聚合多源上下文（CodeBuddy 会话记忆 + git diff + TAPD 需求单），一次生成符合腾讯云官网规范的 product-news.yml（核心字段：标题/类型/日期/描述/版本/排序），人工确认后随 MR 提交。AutoSync Bot 监听仓库扶正 MR 自动录入云端草稿，auto_publish=true 直接发布到腾讯云 8 地域，auto_publish=false 6h 兜底自动发布。type 仅支持「功能上线/体验优化」两种。包含格式校验和增量追加脚本。可选引导 PM 调用已安装的 clawpro-operation-guide skill 生成官网操作指南（本 skill 不打包对方代码）。触发词：「生成本次变更的产品动态」「帮我写产品动态文案」「生成 product-news.yml」。
---

# product-news skill

当 PM 声明"生成本次变更的产品动态"、"帮我写产品动态文案"、"生成 product-news.yml"时，使用本 skill。

## 目标

把"功能开发完成"自动推进到"产品动态文案已就绪可提交"：

1. 聚合多源上下文（git diff + CodeBuddy 会话上下文 + TAPD 需求单）。
2. 基于官网文案规范**一次生成**符合标准的文案（标题 + 类型 + 描述 + 版本号）。
3. 展示文案给 PM 确认，可逐字段修改。
4. 高风险类变更提示 PM 考虑 `auto_publish=false`。
5. 确认后增量追加到 `product-news.yml`（按 `id` 去重）。
6. 输出 `product-news.yml` 的变更，由 PM 随 MR 一起提交。

## 何时使用

- Demo 或产品功能本地开发完成，准备提 MR 之前。
- 需要在合入 main 时附带产品动态文案（供 AutoSync Bot 自动抓取录入云端并发布）。
- 推荐时机：在多人协作分支开发手册的「步骤 6 rebase main 解决冲突」之后、「步骤 7 push 发起 MR」之前调用本 skill。

## 输入

skill 会自动从当前工作区获取以下信息。优先使用已有上下文，不要因为信息不完整就阻塞。

### 自动获取

- **Git diff**（当前分支 vs main 或上次 commit 的变更）
- **MR 标题和描述**（PM 已经在工蜂上写的，或本次开发会话中说明的）
- **变更文件列表**
- **CodeBuddy 会话上下文**：LLM 在生成文案时自然利用当前会话的开发讨论上下文，不需要 PM 手动提取
- **当前日期**
- **已有 `.clawpro/product-news.yml` 文件**（如果存在）

### 可选输入（PM 主动提供）

- **TAPD 需求单链接或 ID**：PM 提供 TAPD 单 ID/链接后，skill 通过 TAPD MCP 现场拉取需求单内容（标题、描述、验收标准）作为生成上下文
- **当前 PM 的工蜂 ID**：用于 `source.author_gongfeng_id`
- **前端 MR id**（如 `!1234`）：用于 `source.frontend_mr_id`
- **后端 MR id**（如 `!5678`，如有）：用于 `source.mr_id`

## 工作流程

### 步骤 0：聚合多源上下文

#### 0.1 读取 git diff

执行以下动作获取本次代码变更信息：

```bash
# 获取分支名
git branch --show-current

# 获取最近一次提交信息
git log -1 --format="%H %s"

# 获取变更文件列表（vs main）
git diff --name-only origin/main...HEAD

# 获取关键 diff（排除配置文件、依赖、lock 文件）
git diff origin/main...HEAD -- \
  ':(exclude)*.lock' \
  ':(exclude)*.json' \
  ':(exclude)go.sum' \
  ':(exclude)pnpm-lock.yaml' \
  ':(exclude)package-lock.json'
```

#### 0.2 利用 CodeBuddy 会话上下文

LLM 在生成文案时**自然利用**当前会话中的开发讨论上下文，包括：
- PM 在本次会话中描述的功能需求
- 开发过程中讨论的影响范围、用户价值
- 已有的功能说明、端类型（管控端/用户端）等

不需要 PM 手动总结或提供会话上下文，LLM 会自动参考。

#### 0.3 拉取 TAPD 需求单内容（可选）

**触发条件**：PM 在调用 skill 时或会话中提供了 TAPD 单链接或 ID（如 `https://www.tapd.cn/20422209/prong/stories/view/1020422209135360254` 或直接 ID `1020422209135360254`）。

**拉取方式**：通过 TAPD MCP 现场拉取需求单内容：

- **MCP 网关**：`https://mcpgw.knot.woa.com/tapd/`
- **调用方式**：JSON-RPC，`method="tools/call"`，`params.name="proxy_execute_tool"`
- **workspace**：`20422209`（ClawPro）
- **拉取字段**：需求单标题、描述、验收标准、自定义字段（前后端开发人员）
- **配置注入**：Access Token 由环境变量 `TAPD_MCP_TOKEN` 或 CodeBuddy MCP 配置注入，skill 文档中不写死 token

**降级策略**：若 TAPD MCP 不可用（网络问题、token 缺失等），提示 PM 手动粘贴 TAPD 需求单的关键内容（标题、描述、验收标准）作为补充上下文，不阻塞流程。

### 步骤 1：生成文案

将聚合的多源上下文填入 User Prompt 模板（`references/prompt-templates.md` 模板 A），结合 System Prompt 中的官网文案规范（`references/copy-guidelines.md`）和示例（`references/examples.md`），**一次 LLM 调用**生成最终文案。

输出格式：
```json
{
  "title": "产品动态标题（名词性词组）",
  "type": "功能上线",
  "endpoint": "管控端",
  "description": "产品动态描述。结尾加句号。",
  "version": "v2.7.0",
  "risk_hint": null,
  "display_recommendation": {
    "banner": {
      "recommended": false,
      "reason": "本次变更无需管理员高优先级持续关注，不建议开启 Banner。"
    },
    "floating_window": {
      "recommended": true,
      "reason": "本次上线新增管理员需要了解的产品能力，适合通过左下角浮窗帮助发现。"
    }
  },
  "needs_guide": null,
  "guide": null
}
```

**字段说明**：
- `risk_hint`：高风险类提示。当 `type=功能上线` 且涉及合规/隐私/计费/配额/权限类变更时，LLM 输出提示文案（如「⚠️ 本次变更涉及计费模式调整，建议将 auto_publish 设为 false 走人工确认发布」）；其他场景为 `null`
- `endpoint`：本次产品动态所属端，只能是「管控端」或「用户端」。LLM 根据上下文判断，PM 在预览阶段确认
- `display_recommendation`：LLM 对 Banner 和浮窗分别给出是否开启的建议及理由；将布尔建议映射为 YAML 草案的初始组件值，推荐理由不写入 YAML
- `needs_guide` / `guide`：始终输出 `null`，是否生成操作指南由 PM 后续人工确认

**如果 LLM 不可用**：使用 `references/prompt-templates.md` 中的降级规则引擎生成占位文案，并明文告知 PM 需要人工重写。

### 步骤 2：展示并确认

先按步骤 5 的规则生成 id，并将推荐结果预填到 YAML 草案：

- `recommended=true` → `enabled: true`，同时写入 `duration_days: 14`
- `recommended=false` → `enabled: false`，省略 `duration_days`
- `reason` 只在对话中展示，不写入 YAML

使用 `merge_yaml.py` 将草案先写入 `.clawpro/product-news.yml`，然后向 PM 展示文案、推荐理由和 YAML 草案。此时必须明确标注“待 PM 确认”，不得提示可随 MR 提交。

向 PM 展示生成的文案（表格形式），逐字段可修改：

```
📌 产品动态文案预览

| 字段 | 值 |
|------|-----|
| 标题 | Skill 管理新增下发和更新能力 |
| 类型 | 功能上线 |
| 端类型 | 管控端 |
| 日期 | 2026-07-15 |
| 版本号 | v2.7.0 |
| 描述 | 支持管理员按租户下发 Skill 到指定 Agent 实例... |

请确认以上文案，或指定要修改的字段。

📣 展示组件建议

| 组件 | 建议 | 理由 |
|------|------|------|
| Banner | 建议开启 | 本次上线新增核心能力，影响范围较广，建议提高曝光。 |
| 浮窗 | 建议开启 | 本次上线新增管理员需要了解的产品能力，适合通过左下角浮窗帮助发现。 |

⏱ 展示时长确认

| 组件 | 当前设置 |
|------|----------|
| Banner | 14天 |
| 浮窗 | 14天 |
```

PM 回复"确认"后，**额外询问**：

```
另外，是否需要生成官网操作指南？（是 / 否）
YAML 草案已按以上建议预填。请确认或修改 Banner、浮窗开关。
开启组件的存在时长默认14天，是否确认？也可以分别指定其他天数。
```

PM 修改后，使用同一 id 再次调用 `merge_yaml.py` 更新条目。将 PM 最终选择分别写入
`display_components.banner.enabled` 和 `display_components.floating_window.enabled`。
组件开启时写入 `duration_days`，默认14天；PM 可指定其他正整数天数。组件关闭时省略 `duration_days`。
仅对 `enabled=true` 的组件执行 duration 确认；两个组件都关闭时跳过 duration 确认。
PM 未明确修改时长但确认默认值时，保留14天；输入非法值、0或负数时要求重新输入。
只有 PM 确认且后续高风险、操作指南等字段处理完成并通过校验后，才通知“可随 MR 提交”。

推荐规则：
- Banner 和浮窗都展示在管控端，目标受众都是管理员；`endpoint` 只表示功能变化发生在「管控端」还是「用户端」，不得直接决定组件开关
- 先判断管理员是否需要感知该变化，包括管理、答疑、权限、安全、合规、成本、推广和用户行为变化等影响
- Banner 是高可见、持续性通知。重大功能、广泛影响、计费/配额/权限/安全/合规变化、管理员必须操作或存在截止时间时建议开启
- 浮窗位于管控端左下角，是低打扰的功能发现和产品动态提醒。新功能、新管理入口、重要体验优化，以及管理员需要了解的用户端能力变化时建议开启
- 用户端变更同样可能需要通知管理员。例如用户新增会话能力、用户可自行下载 Skill 等，通常建议开启浮窗；若同时涉及权限、安全、计费或管理员必须配置，再建议开启 Banner
- 纯视觉微调、用户和管理员均无感的后端优化、很小的缺陷修复或内部能力通常两个组件都不建议开启
- 一般性产品动态优先推荐浮窗；高优先级或必须持续关注的信息推荐 Banner；两种条件同时满足时可同时推荐
- 推荐值可以预填并先写入 YAML 草案，但不得绕过 PM 确认进入可提交状态

### 步骤 3：引导调用 Operation Guide Skill（条件执行）

> **仅当 PM 回答"是"时执行此步骤。**

如果 PM 回答"是"，**引导** PM 调用 CodeBuddy 中已安装的 `clawpro-operation-guide` skill 生成操作文档。

> **重要**：本 skill **不打包** `clawpro-operation-guide` 的代码，仅引导 PM 触发已安装的 skill。

**引导文案示例**：

```
如需生成《{功能名称}》的操作指南文档，请在 CodeBuddy 对话中调用 @clawpro-operation-guide：

  使用 @clawpro-operation-guide 生成操作文档。
  功能名称：{功能名称}
  功能页面 URL：{feature_url}
  端类型：{endpoint}

Operation Guide Skill 完成产出后，将文档存到本地 `~/.clawpro/product-news-guides/{change_id}/`（**不上传 MR**）。
```

**Operation Guide Skill 完成产出**（参考其 SKILL.md）：
- `doc_draft.md` / `doc_polished.md`（草稿+润色版本）
- `screenshots/`（截图目录）
- `export` 后的单文件自包含 Markdown（含 base64 内嵌截图）

PM 完成 Operation Guide 后回到本 skill，将 `needs_guide=true` 写入 product-news.yml 的对应条目，`guide` 字段填入：
```yaml
guide:
  doc_type: "operation_guide"   # 5 选 1，与 Operation Guide Skill 的文档类型对应
  feature_name: "Skill 下发管理"
  feature_url: "https://clawpro.woa.com/admin/skills/delivery"
  endpoint: "管控端"
```

**注意**：`product-news.yml` 中 `guide` 字段**只**记录元信息（doc_type、feature_name、feature_url、endpoint），**不**记录本地文档路径。因为 yaml 会上传 MR，不应包含本地绝对路径。

如果 PM 回答"否"，则 `needs_guide=false`，整个 `guide` 字段省略。

### 步骤 4：高风险类提示（条件执行）

> **仅当步骤 1 LLM 输出的 `risk_hint` 不为 null 时执行此步骤。**

如果 LLM 判断本次变更为高风险类（`type=功能上线` 且涉及合规/隐私/计费/配额/权限类变更），向 PM 展示提示：

```
⚠️ 高风险类变更提示

本次变更涉及 {具体类别，如计费模式/权限规则/合规要求} 调整，建议将 auto_publish 设为 false 走人工确认发布。

是否将 auto_publish 设为 false？（是 / 否，默认否即 auto_publish=true）
```

- PM 回答"是"：`auto_publish=false`（合入后 6h 兜底自动发布）
- PM 回答"否"或不响应：`auto_publish=true`（合入后直接发布）

如果 `risk_hint` 为 null（非高风险类），跳过此步骤，`auto_publish` 默认 `true`。

### 步骤 5：生成 id

步骤 2 写入 YAML 草案前即按本规则生成 id；后续更新必须复用同一 id。

id 格式：`{type-prefix}-{slug}-{date}`

- `type-prefix`：`feat`（功能上线）或 `impr`（体验优化）
- `slug`：从标题提取的英文 kebab-case 短名
- `date`：`YYYYMMDD`

示例：`feat-skill-delivery-20260715`

### 步骤 6：构造完整条目

```yaml
- id: "feat-skill-delivery-20260715"
  title: "Skill 管理新增下发和更新能力"
  type: "功能上线"
  date: "2026-07-15"
  endpoint: "管控端"
  description: "支持管理员按租户..."
  version: "v2.7.0"
  source:
    type: "repo_skill"
    frontend_mr_id: "!1234"        # 前端工蜂 MR id（新流程产品改前端时填此字段）
    mr_id: "!5678"                 # 后端工蜂 MR id（可选，产品只改前端时为空）
    commit: "abc1234"
    author_gongfeng_id: "hawkechen"
  needs_guide: true
  guide:
    doc_type: "operation_guide"
    feature_name: "Skill 下发管理"
    feature_url: "https://clawpro.woa.com/admin/skills/delivery"
    endpoint: "管控端"
  auto_publish: true               # 默认 true；高风险类时由 PM 决定是否设 false
  display_components:
    banner:
      enabled: true                # PM 最终确认值
      duration_days: 14            # 开启时必填，默认14天
    floating_window:
      enabled: false               # 关闭时省略 duration_days
```

**字段约束**：
- `source.frontend_mr_id` 和 `source.mr_id` 至少有一个非空（避免两者都为空导致 Bot 无法追溯）
- `endpoint` 必填，只能是「管控端」或「用户端」
- `display_components.banner.enabled` 和 `display_components.floating_window.enabled` 必填，均为布尔值，未确认时默认 `false`
- 组件 `enabled=true` 时 `duration_days` 必填且必须为正整数，PM 未指定时默认14天；`enabled=false` 时省略 `duration_days`
- `auto_publish` 默认 `true`
- `related_campaign.update_id` 字段**保留**（v2 已有字段，v3 暂不主动填写，等小晨 update-awareness skill 集成后再用）

### 步骤 7：写入 product-news.yml

使用 `scripts/merge_yaml.py` 增量追加（按 id 去重）：

```bash
python scripts/merge_yaml.py \
  --input .clawpro/product-news.yml \
  --new-json '{"id":"feat-...","title":"...","type":"...","date":"...","description":"...",...}' \
  --output .clawpro/product-news.yml
```

如果 `.clawpro/product-news.yml` 不存在，skill 会创建新文件。

> **yaml 文件路径说明**：默认路径为 `.clawpro/product-news.yml`（相对于当前工作区根目录）。PM 可根据实际仓库结构调整路径，skill 不硬编码仓库地址。

### 步骤 8：校验

使用 `scripts/product-news-validator.py` 校验生成的 YAML：

```bash
python scripts/product-news-validator.py .clawpro/product-news.yml
```

确保格式正确后通知 PM 可以随 MR 提交。

## 输出格式

回复用户时使用中文，包含：

- 生成的文案预览（表格）
- 高风险类提示（如有）
- 写入的文件路径
- 校验结果
- **如果引导了 Operation Guide Skill**，附加操作文档的本地路径提示（`~/.clawpro/product-news-guides/{change_id}/`，由 PM 自行管理）
- 下一步提示：

```
文案已就绪 → product-news.yml（请随 MR 提交到仓库）
操作指南文档 → ~/.clawpro/product-news-guides/{change_id}/（本地留存，不上传 MR，可手动上传到写写平台）
```

## 参考文件

- `references/copy-guidelines.md`：官网文案规范（腾讯云标准 + ClawPro 特殊规则）
- `references/examples.md`：优质文案示例（从 156 条历史文案提炼）
- `references/prompt-templates.md`：LLM Prompt 模板 + 降级规则引擎
- `scripts/product-news-schema.json`：YAML JSON Schema 定义
- `scripts/product-news-validator.py`：格式校验脚本
- `scripts/merge_yaml.py`：增量追加/去重脚本

## 关联 Skill

- **ClawPro Operation Guide Skill** (`clawpro-operation-guide`)：当 PM 回答"是"时由本 skill **引导** PM 调用（**不打包对方代码**），生成官网操作指南文档。六阶段流水线（CHECK → PLAN → BROWSER → DOCUMENT → POLISH → QUALITY），文档存到 `~/.clawpro/product-news-guides/{change_id}/`，**不上传 MR**。

## 与小晨 update-awareness skill 的关系

> **v3 暂不集成**：本 skill 工作流不主动引导 PM 调用小晨的 update-awareness skill，相关组件方案由小晨 skill 独立负责触发。

- `product-news.yml` 中的 `related_campaign.update_id` 字段**保留**（v2 已有字段），保持 schema 向后兼容
- 等小晨 update-awareness skill 最新版稳定后，再单独做集成（不在 v3 范围）
- 集成时通过 `related_campaign.update_id` 与小晨 `campaigns.yaml` 的 `updateId` 对齐

## 常用命令

### 生成本次变更的产品动态

```
@product-news 帮我生成本次变更的产品动态
```

### 提供 TAPD 链接作为补充上下文

```
@product-news 生成本次的产品动态。
TAPD 需求单：https://www.tapd.cn/20422209/prong/stories/view/1020422209135360254
```

### 提供完整上下文（推荐）

```
@product-news 生成本次的产品动态。
功能说明：新增了 Agent 计费模式配置页面，支持按量计费和包年包月切换。
TAPD 需求单：1020422209135360254
前端 MR：!1234
```

### 查看已有产品动态

```
@product-news 显示当前已有的产品动态列表
```

## 多人协作分支开发流程中的位置

本 skill 在多人协作分支开发手册中的位置：

```
步骤 4：每次开发前同步主干
   ↓
步骤 5：自然语言对话开发
   ↓
步骤 6：开发完成 rebase main 解决冲突
   ↓
步骤 6.5：⭐ 调用 @product-news 生成 product-news.yml + PM 在 CodeBuddy 内审核
   ↓
步骤 7：push 到 feature/* 分支 + 发起向 main 的 MR
   ↓
步骤 8：点击 MR 链接确认合入
   ↓
步骤 9：CI/CD 自动部署 demo 环境
```

详细步骤 6.5 集成段落见 `多人协作手册-步骤6.5-skill调用段落.md`。
