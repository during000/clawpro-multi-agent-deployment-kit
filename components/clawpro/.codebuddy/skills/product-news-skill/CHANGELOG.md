# CHANGELOG

## v3.5 (2026-07-21)

**修改人：Nicole**

### Duration 确认

- YAML 草案预填后，向 PM 明确展示所有已开启组件的 `duration_days`
- Banner 和浮窗开启时默认14天，PM 必须确认默认值或分别修改
- `duration_days` 只允许正整数；输入0、负数或非法值时要求重新输入
- 组件关闭时省略 `duration_days`；Banner 和浮窗都关闭时跳过 duration 确认

## v3.4 (2026-07-17)

**修改人：Nicole**

### 推荐值预填 YAML

- 将 LLM 判断的 Banner 和浮窗建议直接预填到 YAML 草案
- 推荐开启时写入 `enabled: true` 和默认 `duration_days: 14`；推荐关闭时写入 `enabled: false`
- 推荐理由只在对话中展示，不写入 YAML
- YAML 草案可以先落盘，但 PM 确认前不得标记为可随 MR 提交
- PM 修改组件开关或时长后，使用同一 id 更新原条目

### 合并修复

- `merge_yaml.py` 改为比较完整条目，确保只修改组件开关、时长或其他非文案字段时也能正确更新

## v3.3 (2026-07-17)

**修改人：Nicole**

### 推荐原则优化

- 明确 Banner 和左下角浮窗都面向管控端管理员，`endpoint` 只记录功能变化发生端
- 推荐逻辑改为判断管理员相关性，包括管理、答疑、权限、安全、合规、成本、推广和用户行为变化
- Banner 定位为高可见、持续性通知；浮窗定位为低打扰的功能发现和产品动态提醒
- 用户端变更也可推荐展示，例如用户新增会话能力、用户可自行下载 Skill 等通常推荐浮窗
- 用户端变更若涉及权限、安全、计费或要求管理员配置，可同时推荐 Banner

## v3.2 (2026-07-17)

**修改人：Nicole**

### 推荐流程

- 文案生成后，LLM 分别给出 Banner 和浮窗的开启建议及理由
- PM 审核文案时同步查看组件建议，并分别确认或修改最终开关
- 推荐结果仅用于对话展示，不写入 YAML；PM 未确认时两个组件均关闭

### 展示时长

- `display_components` 调整为 Banner 和浮窗两个独立配置块
- 组件开启时记录 `duration_days`，默认14天，允许 PM 修改为其他正整数
- 组件关闭时省略 `duration_days`

### 配套更新

- 同步更新 Skill 流程、Prompt、Schema、YAML 合并脚本、示例、使用说明和测试

## v3.1 (2026-07-17)

**修改人：Nicole**

### 字段变更

- 新增必填字段 `endpoint`，仅允许「管控端」或「用户端」
- LLM 根据上下文生成端类型，并由 PM 在预览阶段确认
- 新增必填 `display_components`，分别用布尔值记录 Banner 和浮窗是否开启
- Banner 和浮窗默认关闭，必须由 PM 明确确认，不由 LLM 自动开启

### 配套更新

- 同步更新 Schema、YAML 合并脚本、Prompt、示例、使用说明和测试

## v3 (2026-07-15)

### 重大变更

- **多源上下文聚合（步骤 0 新增）**：skill 综合聚合 CodeBuddy 会话上下文（LLM 自然利用）+ git diff + TAPD 需求单内容（PM 提供链接后 skill 调 TAPD MCP 现场拉取），一次生成更准确的文案
- **CodeBuddy 内一次性审核**：替代旧流程的企微审核环节，PM 在 CodeBuddy 内完成文案审核后随 MR 提交
- **Operation Guide 引导调用（步骤 3 改动）**：从 v2 的 skill 内置调用改为**引导** PM 调用已安装的 `@clawpro-operation-guide` skill（不打包对方代码）
- **高风险类提示（步骤 4 新增）**：当 type=功能上线 且涉及合规/隐私/计费/配额/权限类变更时，LLM 输出 `risk_hint` 字段提示 PM 考虑 `auto_publish=false`
- **auto_publish 默认 true**：从 v2 的 false 改为 true（新流程 CodeBuddy 已审核，默认自动发布更合理）
- **source 区分前后端 mr_id**：新增 `source.frontend_mr_id` 可选字段（前端 MR id），保留 `source.mr_id` 表示后端 MR id
- **v3 暂不集成 update-awareness**：related_campaign 字段在 schema 中保留（向后兼容），但工作流不主动引导 PM 调用小晨 skill

### Schema 变更

- `auto_publish` 默认值 `false` → `true`
- `Source` 新增 `frontend_mr_id` 可选字段
- `RelatedCampaign` 字段保留（v3 暂不主动填写）
- 其他字段全部保留兼容（不破坏 v2 已有 yaml）

### 脚本变更

- `merge_yaml.py`：适配 `auto_publish` 默认值 true；新增 `frontend_mr_id` 序列化逻辑
- `product-news-validator.py`：新增语义校验「`source.frontend_mr_id` 和 `source.mr_id` 至少有一个非空」
- `test_validator.py`：新增 3 个 v3 测试用例（共 15 个，全部通过）

### Prompt 模板变更

- 模板 A 升级为多源上下文模板：新增 `{{CODEBUDDY_SESSION_CONTEXT}}` 和 `{{TAPD_STORY_CONTENT}}` 占位符
- 新增模板 C：TAPD 需求单 MCP 查询引导（PM 提供链接 → skill 调 MCP → 填入 Prompt；MCP 不可用时降级为 PM 手动粘贴）
- System Prompt 增加高风险类提示逻辑（输出 `risk_hint` 字段）
- 降级规则引擎：`risk_hint=null` + 提示 PM 自行评估高风险类

### 文档变更

- `SKILL.md` 升级工作流为 9 步（步骤 0 多源上下文聚合 + 步骤 3 Operation Guide 引导 + 步骤 4 高风险类提示）
- `使用说明.md` 更新调用示例（多源上下文 / TAPD 链接 / Operation Guide 引导）
- `多人协作手册-步骤6.5-skill调用段落.md` 新建（多人协作分支开发手册步骤 6.5 集成段落，不改原 docx）
- `ClawPro-产品动态本地生成方案.md` 更新流程图
- `hawke-开发任务.md` 新增 v3 任务块
- `开发会话提示词.md` 新增 v3 会话提示词

### 关键决策

| 决策 | 理由 |
|---|---|
| skill 内不硬编码仓库地址 | skill 只生成 yaml，不管提交到哪里；仓库地址在 Bot 配置中维护 |
| Operation Guide 引导调用而非打包 | `clawpro-operation-guide` 已在 CodeBuddy 用户 Skill tab 全局可用，引导 PM 触发即可 |
| v3 不集成 update-awareness | 由小晨 skill 独立负责触发；related_campaign 字段保留以保持向后兼容 |
| LLM 自然利用会话上下文 | 不主动读取 .codebuddy/memory/ 文件，依赖 LLM 自然利用当前会话 |
| TAPD 通过 MCP 现场拉取 | PM 提供链接/ID → skill 调 TAPD MCP → 填入 Prompt；MCP 不可用时降级为 PM 手动粘贴 |
| auto_publish 默认 true + 高风险类提示 | 新流程 CodeBuddy 已审核，默认自动发布更合理；高风险类在 skill 中显式提示 |

---

## v2 (2026-07-06 19:50)

### 优化

- **自动触发 Operation Guide Skill**：当 PM 回答"是"需要操作指南时，skill 自动调用 `clawpro-operation-guide` 的 `scripts/orchestrator.py run` 生成官网操作文档
- 操作指南文档存到 `~/.clawpro/product-news-guides/{change_id}/`，**不上传 MR**
- `product-news.yml` 中 `guide` 字段只记录元信息（doc_type/feature_name/feature_url/endpoint），不记录本地路径
- 工作流从 7 步扩展为 8 步（新增"4. 触发 Operation Guide Skill"）

### 打包

- `product-news-skill-v2.zip` (28KB)

---

## v1.0.0 (2026-07-06 17:00)

### 新增

- 初版 product-news skill，支持从 git diff 生成产品动态文案
- JSON Schema 定义（`product-news-schema.json`）
- 官网文案规范（`copy-guidelines.md`，从 AutoSync 项目提炼）
- 优质文案示例（`examples.md`，从 156 条历史文案提炼）
- LLM Prompt 模板 + 降级规则引擎（`prompt-templates.md`）
- 格式校验脚本（`product-news-validator.py`，12 个测试用例）
- YAML 增量追加/去重脚本（`merge_yaml.py`）

### 设计决策

- type 仅 2 种：`功能上线` / `体验优化`（与 AutoSync 录入表单一致）
- needs_guide 由 PM 人工确认，skill 不自动判断
- 文案一次生成即符合规范，不二次调用 LLM 润色
- id 格式：`{type-prefix}-{slug}-{date}`
- product-news.yml 使用按 id 去重 + 增量追加模式
