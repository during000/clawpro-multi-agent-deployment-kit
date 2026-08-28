---
name: clawpro-knowledge-deposition
description: 审阅最新完整回合，将一条可复用、可验证且已脱敏的 ClawPro/OpenClaw 企业知识按主题发布到工蜂 harness-knowledge-store。由 Stop Hook 高价值预筛后触发，也可在用户明确要求沉淀知识时使用。
metadata:
  short-description: ClawPro 工蜂知识沉淀与主题化入库
  version: 2.1.0
allowed-tools:
  - read_file
  - search_content
  - execute_command
---

# ClawPro 工蜂知识沉淀 V2

## 目标

将 Agent 工作中产生的事实、决策、约束、根因和操作流程转化为后续可以被准确召回的企业知识。

Stop Hook 每轮都会执行低成本信号预筛；只有出现 ClawPro/Agent 领域信号，并形成结论、变更、证据或用户决策时，才拉回本 Skill。一个回合最多发布一条知识，同一主题的多个相关结论必须合并。

触发不等于强制写入。没有达到质量门槛时必须静默跳过，不能用“本轮无知识”之类的占位内容污染知识库。

## 一、读取范围

1. 只读取 Stop Hook 提供的 `transcript_path`。
2. 只分析最新一个完整用户输入与对应助手回复；必要时向前读取少量上下文理解指代，但不得复制完整会话。
3. 不读取或沉淀 `MEMORY.md`、`USER.md`、`IDENTITY.md`、个人记忆目录、用户主目录私有资料或与当前回合无关的文件。

## 二、写入门槛

同时满足以下条件才允许发布：

- 与 ClawPro/OpenClaw 的产品、实现、配置、治理、Hook、Skill、Rules、知识库或协作流程直接相关。
- 形成了后续 Agent 可复用的事实、约束、决策、根因、验收结论或标准操作。
- 结论有本轮证据支撑，明确区分事实、用户决策和推测。
- 不含 Token、Cookie、密码、私钥、访问凭证、个人身份信息、私人对话原文或其他敏感内容。
- 与正式知识不是语义重复。

必须跳过：

- 寒暄、确认、状态询问和一次性命令执行结果。
- 尚未验证的猜测、临时脑暴、未确认方案。
- 只对当前机器、当前会话或个人偏好有效的信息。
- 纯代码流水账、完整聊天记录、长日志或敏感数据。
- 已有知识没有新增边界、证据或决策的重复表达。

## 三、先查重，再决定是否写入

发布前先在当前消息注入的工蜂知识缓存中检索标题、别名、实体和主题。若没有注入路径，默认只读检查：

```text
~/.clawpro-harness/gongfeng-knowledge-cache/current/
```

查重范围：

- `clawpro/modules/`
- `clawpro/product/`
- `clawpro/implementation/`
- `clawpro/topics/`
- `clawpro/decisions/`
- `clawpro/runbooks/`

处理规则：

- 完全重复：跳过。
- 相同主题出现新证据或新边界：形成一个新版本，并通过 `--supersedes` 指向旧知识 ID。
- 无法确认结论或语义重复关系：跳过，不生成文件。

## 四、分类和正式知识门槛

### 4.1 知识类型

| 类型 | 适用内容 |
| --- | --- |
| `fact` | 已验证的产品或实现事实 |
| `decision` | 用户或团队明确确认的决策 |
| `constraint` | 必须遵守的规则、边界或限制 |
| `runbook` | 可重复执行的标准操作 |
| `root-cause` | 已验证的问题根因和修复结论 |
| `implementation` | 已落地并验证的技术机制 |

### 4.2 正式知识门槛

只有满足以下任一证据条件时才允许写入：

- TAPD、正式文档或明确需求单支持。
- 代码路径和对应测试、自测或构建结果同时存在。
- 工蜂 MR、Commit 或发布记录能够核验。
- 用户在当前回合明确确认了一项决策，且知识类型为 `decision`。

否则直接跳过，不生成文件。

### 4.3 置信度

- `high`：多项证据一致，或正式需求加验证结果。
- `medium`：有一项可靠证据，但覆盖范围有限。
- 证据不足以达到 `medium`：跳过，不写文件。

## 五、主题化路径

`topics` 是知识沉淀的唯一目录根。时间不得用作主目录；`topic` 和文件名中的
`slug` 必须为英文小写 kebab-case：

```text
harness-knowledge-store/clawpro/
└── topics/
    └── knowledge-hook/
        └── verified-only--1d7494ac.md
```

`domain` 只作为知识元数据和检索字段，不再形成额外目录层级。推荐领域：

- `local-agent`
- `asset-governance`
- `knowledge-governance`
- `agent-collaboration`
- `auth-and-users`
- `billing`
- `observability`

主题应表达稳定能力，例如 `knowledge-hook`、`skill-distribution`、`workspace-binding`，不得使用日期、用户名、会话 ID 或临时任务名。

`slug` 应简短表达该条知识的具体结论，例如 `verified-only`、`top-k-before-prompt`。
文件名末尾保留知识 ID 的前 8 位用于唯一定位和追踪；完整知识 ID 仍写在正文元数据中。

时间只写在 `创建时间`、`更新时间`、`生效时间` 元数据中。

## 六、每条知识字段

| 字段 | 要求 |
| --- | --- |
| `title` | 12–50 字，直接表达结论 |
| `summary` | 已确认的知识及其价值，最多 800 字 |
| `scope` | 适用模块、角色、Agent、事件或版本 |
| `evidence` | TAPD、代码路径、测试结果、MR 或用户决策 |
| `boundary` | 不适用范围、风险和待确认项；没有则写“无” |
| `knowledge-type` | 六种知识类型之一 |
| `domain` | 领域元数据，不形成目录 |
| `topic` | `topics` 下的稳定主题目录 |
| `slug` | 文件名中的可读英文语义 |
| `confidence` | `medium`、`high` |
| `aliases` | 用户可能使用的其他说法 |
| `entities` | ClawPro、TeamAI、Stop 等关键实体 |
| `source-refs` | 可追溯的需求、代码、MR、Commit 或文档 |
| `supersedes` | 被新知识替代的旧知识 ID |

## 七、发布

只能使用仓库发布脚本，不得切换当前业务工作树，不得直接修改 `main`：

```bash
python3 .codebuddy/skills/clawpro-knowledge-deposition/scripts/publish_knowledge.py \
  --title '<知识标题>' \
  --summary '<知识结论>' \
  --scope '<适用范围>' \
  --evidence '<核验依据>' \
  --boundary '<边界或无>' \
  --knowledge-type implementation \
  --domain local-agent \
  --topic knowledge-hook \
  --slug verified-only \
  --confidence high \
  --tag Hook \
  --tag 知识库 \
  --alias '知识沉淀 Hook' \
  --entity TeamAI \
  --entity Stop \
  --source-ref 'code:.codebuddy/hooks/clawpro_knowledge_deposit_stop.py' \
  --source-ref 'test:publish_knowledge self-test passed' \
  --session-id '<Stop Hook 提供的 session_id>'
```

脚本负责：

- 校验当前仓库与工蜂 `origin`。
- 拦截凭据和危险内容。
- 基于规范化内容生成稳定知识 ID。
- 在 `topics/<topic>/` 下生成“可读 slug + 短 ID”的正式知识路径。
- 在临时 Git 目录中拉取 `harness-knowledge-store`，不触碰业务工作树。
- 对相同知识 ID 去重。
- 以普通非强推方式提交；遇到并发非快进时重新拉取并最多重试 3 次。

返回状态：

- `published`：已创建正式知识。
- `duplicate`：相同知识已存在。
- `dry_run`：仅完成本地校验。
- 非零退出：发布失败。不得声称已沉淀。

## 八、召回约束

- `UserPromptSubmit` 只召回 `topics` 下的 `verified` 正式知识。
- 历史 `_inbox`、非 `verified` 条目及已被其他知识 `supersedes` 的旧条目不参与召回。
- 路径、标题、别名、实体和正文均参与检索；领域、主题和 slug 必须准确。
- 时间只用于有效期和新旧判断，不参与主要语义路由。

## 九、Stop 收口

- 发布或跳过后，不重复向用户输出刚才的答案。
- 不生成第二条知识。
- 不调用其他会改变业务代码或当前分支的流程。
- 正常结束；`stop_hook_active` 会让下一次 Stop 直接放行。
