# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## TAPD 原始信息

| 字段 | 内容 |
|------|------|
| TAPD | https://tapd.woa.com/tapd_fe/20422209/story/detail/1020422209134740125 |
| Workspace | `20422209` |
| Story ID | `1020422209134740125` |
| 标题 | `【clawpro】管控端agent列表支持批量配置模型` |
| 状态 | `待启动` (`status_57`) |
| 优先级 | High |
| 创建人 | `kathrynxia` |
| 处理人 | `kathrynxia;` |
| 创建时间 | 2026-05-28 14:15:32 |
| 更新时间 | 2026-07-09 10:57:20 |
| 截止时间 | 2026-07-13 |
| 分类 | `1020422209002772062` |
| 描述 | `(无描述内容)` |
| TAPD Request ID | `f18fd39d1c804f75c5ea9aaf33e1d90d` |

---

## 初始讨论确认的需求口径

本任务按最初讨论与后续 review 口径确认需求：

- 管控端 agent 列表需要支持批量配置模型，后端提供批量接口给列表批量操作使用。
- 批量配置是“覆盖目标实例当前模型集合”，不是追加模型；请求中的顶层模型是 primary，`fallbacks` 是最终 fallback 列表。
- 支持两种目标选择方式：实例数据库 `ids` 和 CVM `instance_ids`；两者至少一个，若同时提供则以 `ids` 为准。
- 单个目标失败不能影响其他目标；缺失目标也应在 `results` 中给出失败项，便于前端逐项展示。
- 已解析目标如果混合不同标准化 `agent_type`，应请求级拒绝，避免一次批量操作跨运行时脚本语义。
- OpenClaw 支持 primary + fallback；Hermes / LightclawACE 仅支持单模型；兼容 OpenClaw 的自定义 Agent 类型按 OpenClaw 运行时处理。
- OpenClaw `3.28.x` 不支持 fallback；带 fallback 的请求应按单项失败返回。
- 不改变已有单实例 `/openclaw/set-model` 和 `/admin/instances/set-model` 的对外契约，尤其成功响应继续保留 `provider` / `model_id`。
- API 文档和 OpenAPI 生成必须符合 CODEBUDDY.md 规范；新增 operation 需要 `test/scripts/` 下的轻量集成测试覆盖。
- 代码 review 中确认：`context_len` 只补当前接口的整数解析错误检查；`<=0` 继续走默认值，不扩大范围、不提取常量。


## 背景

本任务面向管控端 agent 列表批量配置模型场景，核心是让管理员在列表中一次性为多个实例覆盖模型配置。

当前分支已经围绕管控端批量设置模型完成后端能力：

- 新增管控端批量接口 `POST /admin/instances/batch-set-model`。
- 支持通过实例数据库 ID 列表或 CVM `instance_id` 列表选择目标。
- 对目标实例覆盖模型集合：顶层模型作为 primary，`fallbacks` 作为 fallback 列表。
- 保持单实例 `/openclaw/set-model` 既有返回结构，不因批量能力破坏旧接口。
- 已补充 API 文档和 endpoint-level 集成测试脚本。

---

## 目标

- [x] 管控端提供批量配置模型能力，适配 agent 列表批量操作入口。
- [x] 批量接口支持 primary + fallback 覆盖语义，避免把覆盖操作误设计成 add-model。
- [x] 单实例设置模型接口保持兼容，成功响应继续包含 `provider` 和 `model_id`。
- [x] 批量请求按目标返回独立结果，缺失目标或单项失败不影响其他目标。
- [x] 不同 Agent 类型混选、fallback 支持范围、旧 OpenClaw 版本限制等边界有明确失败语义。
- [x] API 文档可被 `test/api_md_to_openapi.py` 解析，参数表符合 CODEBUDDY.md 规范。
- [x] `test/` 下补充新增 OpenAPI operation 的轻量集成测试覆盖。

---

## 范围

| 包含 | 不包含 |
|------|--------|
| `POST /admin/instances/batch-set-model` 后端接口 | 前端页面交互实现 |
| 批量目标选择：`ids` / `instance_ids` | 新增模型管理能力 |
| primary + fallback 覆盖语义 | 把批量接口实现为 add/append fallback |
| OpenClaw / Hermes / LightclawACE 及兼容自定义 Agent 类型的模型设置限制 | 改造 Agent 脚本能力矩阵 |
| 请求级和单项级错误语义 | 全部失败时改成请求级失败 |
| API.md / OpenAPI 生成 / 集成测试覆盖 | 真实 CVM/TAT 端到端压测 |
| 保持已有 `/set-model` 兼容行为 | 改动用户侧单实例 API 合约 |

---

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | TAPD 无正文，是否存在未同步到 TAPD 的前端交互细节或验收补充？ | 不阻塞 | 当前按分支已实现后端/API/test 范围推进；若后续补充 UI 要求，单独进入前端任务。 |
| 2 | 是否需要真实 K8s/CVM 集成测试执行记录？ | 不阻塞 | 本分支已补 endpoint-level IT 脚本；完整 `make test` 依赖镜像、集群和凭证，后续在 IT 阶段记录是否执行或豁免。 |
| 3 | `context_len` 非整数是否要扩大到所有入口统一校验？ | 已确认 | 不扩大；仅在当前审查指出的接口处检查 `strconv.Atoi` 错误，`<=0` 仍保持默认值语义，不提取常量。 |

---

## 约束与依赖

- Bookmark 名称必须保持 `feature/admin-batch-set-model`，不携带 story number。
- story number 仅保留在 commit description：`--story=134740125`。
- 合入前按 CODEBUDDY.md SOP 补齐 `01`-`08` 产物。
- 代码如果当前审查无问题，可以跳过新的开发阶段，只记录实现事实和验证证据。
- 工作区必须保持 clean；所有改动 squash 到单一 feature commit。
- API 文档参数表必须满足 CODEBUDDY.md 规范：4 列、参数名不加反引号、必填列只用 `是` / `否` / `条件`。
- 新增 OpenAPI operation 需要 `test/scripts/` 集成测试脚本覆盖，避免增量覆盖率红线。
