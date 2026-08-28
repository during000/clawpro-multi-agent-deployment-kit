# 02. Plan — 方案设计

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `main.go` | 修改 | 注册 `POST /admin/instances/batch-set-model` OpenAPI + Audit 管控端路由 |
| `controller/admin_instances.go` | 修改 | 新增批量设置模型请求结构、参数解析、目标解析、混合 Agent 类型校验、并发执行与结果聚合 |
| `controller/openclaw_model.go` | 修改 | 复用单实例模型校验与下发能力；新增批量覆盖 primary/fallback 绑定、DB 快照回滚、TAT 失败补偿；保留 `/set-model` 返回兼容；补充当前接口 `context_len` 整数解析错误检查 |
| `controller/audit.go` | 修改 | 新增批量设置模型审计动作 `instance_admin_batch_set_model` |
| `i18n/keys.go` / `i18n/en.go` | 修改 | 新增批量接口错误文案：重复模型、混合 Agent 类型、非 OpenClaw fallback 限制等 |
| `controller/admin_instances_test.go` | 修改 | 增加批量设置模型 handler 单元测试，覆盖请求校验、目标解析、失败隔离、回滚、fallback 覆盖和 Agent 类型限制 |
| `docs/API.md` | 修改 | 增加 `POST /admin/instances/batch-set-model` 文档，修正参数表格式、限制说明、响应与错误语义 |
| `test/scripts/model/test_batch_set_model.py` | 新增 | 新增 OpenAPI endpoint-level 集成测试脚本，覆盖鉴权、参数校验和全部目标不存在结果形态 |
| `.specs/plans/2026-07-09_admin-batch-set-model/` | 新增 | 按 CODEBUDDY.md SOP 补齐 01-08 阶段产物 |

---

## 调用链 / 数据流

### 路由入口

```text
POST /admin/instances/batch-set-model
  → controller.WithOpenAPI
  → controller.WithAudit
  → controller.HandleAdminBatchSetModel
  → handleAdminBatchSetModel
```

### 请求级处理

```text
handleAdminBatchSetModel
  → requireAdmin
  → method == POST
  → json.Decoder.Decode(adminBatchSetModelRequest)
  → parseAdminBatchSetModelSelectors
      - ids 优先
      - ids / instance_ids 至少一个
      - 单次最多 20
      - 去重、过滤非法空值
  → parseAdminBatchSetModelInputs
      - 顶层模型作为 primary
      - fallbacks 作为 fallback 列表
      - primary/fallback 不允许重复
  → 查询目标实例
  → 所有已解析目标必须是同一个标准化 agent_type
```

### 单目标执行

```text
每个目标实例
  → batchSetModelForInstance
      → validateSetModelInstance
      → model.GetAgentRuntimeType
      → fallback 能力校验
      → resolveSetModelBindingForInstance(primary)
      → resolveSetModelBindingForInstance(fallback...)
      → replaceInstanceModelBindings
          - 快照旧 instances.ai_model_id/custom_model_config
          - 快照旧 instance_models 绑定集合
          - 删除旧绑定
          - 写入目标 primary + fallback
          - 同步 instances 上的 primary 字段
      → injectModelConfigToCVM(primary/fallback...)
      → 任一下发失败：restoreInstanceModelBindings 回滚 DB
      → 返回单项 ok/failed 结果
```

### 并发与结果

```text
目标列表按请求去重后的顺序生成 results
  → 每个目标 goroutine 执行
  → semaphore 限制并发为 5
  → 缺失目标直接写 failed result
  → 单项失败不阻塞其他目标
  → 请求级校验通过时 HTTP 200 返回 { ok: true, results: [...] }
```

---

## 数据库变更

无 schema 变更。

| 表 | 字段 | 类型 | 说明 |
|----|------|------|------|
| `instances` | `ai_model_id` | 既有字段 | 批量覆盖 primary 后同步为 primary 内置模型 ID；自定义 primary 时写 0 |
| `instances` | `custom_model_config` | 既有字段 | 自定义 primary 时保存配置；内置 primary 时清空 |
| `instance_models` | 整行集合 | 既有表 | 批量接口按目标集合覆盖：删除旧绑定，写入请求中的 primary + fallbacks |

---

## 测试用例设计（自然语言描述）

> 本阶段按已有实现反推计划；代码已经完成时，Implement 阶段记录实际实现与本计划一致性即可。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 非 POST 方法 | `GET /admin/instances/batch-set-model` | HTTP 405 | P0 |
| 2 | JSON 格式错误 | 非法 JSON body | HTTP 400 | P0 |
| 3 | 缺少目标选择器 | 仅传 `ai_model_id` | HTTP 400 | P0 |
| 4 | `ids` 超过上限 | `ids` 长度 21 | HTTP 400 | P0 |
| 5 | 全部 ID 不存在 | `ids=[999999]` | HTTP 200，`results[0].status=failed`，message 为实例不存在 | P0 |
| 6 | 部分存在、部分缺失 | `ids=[existing, missing]` | HTTP 200，存在目标执行，缺失目标 failed，顺序保持 | P0 |
| 7 | 可见性失败不阻塞其他项 | 两个实例，其中一个无模型可见权限 | HTTP 200；不可见项 failed，其他项继续 | P0 |
| 8 | 本地实例不支持 | `source=local` 实例 | 单项 failed，不触发 TAT | P0 |
| 9 | TAT 失败回滚单实例 | 一个目标 TAT runner 返回错误 | 该目标 failed，DB 恢复旧绑定，其他目标不受影响 | P0 |
| 10 | OpenClaw fallback 覆盖旧 fallback | 旧 primary + 多个旧 fallback，新请求 primary + 一个 fallback | DB 最终只保留新 primary + 新 fallback | P0 |
| 11 | 无 fallback 时清空旧 fallback | 请求仅 primary | DB 最终只保留新 primary，旧 fallback 被删除 | P0 |
| 12 | Hermes / ACE 带 fallback | 非 OpenClaw runtime 且请求有 `fallbacks` | 单项 failed，返回非 OpenClaw 仅支持一个模型 | P0 |
| 13 | 混合 Agent 类型 | 同一批次包含 OpenClaw 与 Hermes/ACE | 请求级 HTTP 400，不执行 DB/TAT | P0 |
| 14 | OpenClaw 3.28.x 带 fallback | `agent_version` 以 `3.28` 开头且有 fallback | 单项 failed，返回 Agent 不支持 fallback | P1 |
| 15 | 重复模型校验 | primary 与 fallback 相同，或 fallbacks 重复 | 请求级 HTTP 400 | P0 |
| 16 | 忽略 `instance_model_id` | 顶层或 fallback 内传 `instance_model_id` | 不精确更新单条绑定，仍按覆盖集合处理 | P1 |
| 17 | 单实例 `/set-model` 返回兼容 | 调用原用户侧 set-model | 成功响应仍包含 `provider` / `model_id` | P0 |
| 18 | `context_len` 整数解析错误 | 当前接口自定义模型 form 中 `context_len=abc` | HTTP 400；`context_len<=0` 仍走默认语义 | P1 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 管控端鉴权 | 无 token、错误 token、非管理员 token | HTTP 401/403 | P0 |
| 2 | 非 POST 方法 | GET 请求 | HTTP 405 | P0 |
| 3 | JSON 格式错误 | raw body `not json` | HTTP 400 | P0 |
| 4 | 缺少选择器 | `{"ai_model_id": 1}` | HTTP 400 | P0 |
| 5 | 缺少 `ai_model_id` | `{"ids": [1,2]}` | HTTP 400 | P0 |
| 6 | `ids` 超上限 | 21 个 ids | HTTP 400 | P0 |
| 7 | `instance_ids` 超上限 | 21 个 instance_ids | HTTP 400 | P0 |
| 8 | fallback 重复 | primary 与 fallback `ai_model_id` 相同 | HTTP 400 | P0 |
| 9 | 全部 ids 不存在 | 两个不存在的 ids | HTTP 200，两个 failed result | P0 |
| 10 | 全部 instance_ids 不存在 | 两个不存在的 instance_ids | HTTP 200，两个 failed result | P0 |

---

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 批量覆盖误解为追加 fallback，导致旧 fallback 未删除 | 中 | 高 | 文档、函数命名和测试统一强调覆盖语义；UT 验证旧 fallback 被删除 |
| 2 | DB 已写入但 TAT 下发失败，造成 DB/CVM 不一致 | 中 | 高 | 写入前保存完整 binding snapshot；TAT 失败后补偿恢复 `instances` 和 `instance_models` |
| 3 | 混合 Agent 类型导致同一批次走不同脚本契约 | 中 | 中 | 请求级校验所有已解析目标的标准化 `agent_type` 一致 |
| 4 | fallback 下发到不支持多模型的 runtime | 中 | 中 | 单目标执行前使用 runtime type 和版本限制校验 |
| 5 | 新接口未被 OpenAPI 增量覆盖率识别 | 中 | 中 | API.md 使用规范参数表；新增 `test/scripts/model/test_batch_set_model.py` 覆盖 operation 和主要参数 |
| 6 | 单实例 set-model 兼容性被批量能力牵连 | 低 | 高 | 不改变成功响应结构；保留 `provider` / `model_id`；UT 覆盖 SetModel 相关路径 |
| 7 | 并发过高压垮外部 TAT/DB | 低 | 中 | 并发上限固定为 5，参考现有管理端批量操作保守值 |
| 8 | TAPD 无正文，验收边界可能不完整 | 中 | 中 | 本 SOP 固化初始讨论口径；后续新增前端/交互验收单独补充 |
