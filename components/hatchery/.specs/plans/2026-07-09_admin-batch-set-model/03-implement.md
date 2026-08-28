# 03. Implement — 实现记录

---

## 关键实现细节

### 1. 路由与审计

- `main.go` 注册 `POST /admin/instances/batch-set-model`。
- 路由包装保持与其他管控端实例操作一致：`WithOpenAPI` + `WithAudit`。
- `controller/audit.go` 增加审计动作 `instance_admin_batch_set_model`，保证写接口有审计记录。

### 2. 请求结构与参数解析

- `adminBatchSetModelRequest` 承载顶层 primary 模型字段和目标选择字段：
  - `ids []uint`
  - `instance_ids []string`
  - `ai_model_id *uint`
  - 自定义模型字段
  - `fallbacks []adminBatchSetModelModelPayload`
- `ai_model_id` 使用指针区分“未传”和“传 0”：
  - 未传：请求级 `400`
  - 传 `0`：自定义模型配置
- `parseAdminBatchSetModelSelectors` 负责目标选择：
  - `ids` 优先于 `instance_ids`
  - 至少提供一类目标
  - 单次最多 20
  - 过滤 0 / 空字符串并按首次出现顺序去重
- `parseAdminBatchSetModelInputs` 负责模型输入：
  - 顶层字段解析为 primary
  - `fallbacks` 逐项解析为 fallback
  - primary 与 fallback、fallback 之间不允许重复

### 3. 目标构造与混合类型校验

- `handleAdminBatchSetModel` 内部定义局部 `batchSetModelTarget`，只在当前 handler 内承载：
  - 请求原始顺序下标
  - 已解析实例指针
  - 预置 result
- 目标查询后保持请求去重后的顺序写回 `results`。
- 所有已解析目标必须拥有同一个标准化 `agent_type`：
  - 空 `agent_type` 视为 `openclaw`
  - 混合 OpenClaw / Hermes / LightclawACE / 自定义类型时请求级 `400`
  - 缺失目标不参与混合类型判断

### 4. 单目标批量设置流程

`batchSetModelForInstance` 是单个目标的执行入口：

1. 复用 `validateSetModelInstance` 校验实例状态、来源和模型能力。
2. 使用 `model.GetAgentRuntimeType` 确定脚本运行时。
3. 有 fallback 时额外校验：
   - 运行时必须是 OpenClaw
   - OpenClaw `3.28.x` 不支持 fallback
4. 解析 primary 和 fallback 为 `resolvedSetModelBinding`：
   - 内置模型：校验启用、可见、分组可见性、Domain
   - 自定义模型：校验自定义模型开关、必填字段、model_id、URL、model_type、headers
5. 一次性构造 desired `instance_models` 集合。
6. 调用 `replaceInstanceModelBindings` 覆盖 DB 绑定。
7. 对 primary/fallback 逐个执行 `injectModelConfigToCVM`，保证每个 provider 配置写入 CVM。
8. 任一 TAT 下发失败时调用 `restoreInstanceModelBindings` 恢复 DB。
9. 若 DB 回滚成功且存在旧绑定，继续调用 `restoreInstanceModelBindingsToCVM` 重新下发旧 primary/fallback；若不存在旧绑定，则通过 `cleanupDesiredModelBindingsFromCVM` 调用 `remove_model_provider.sh` 清理本次新写入 provider，避免前序成功的 TAT 调用使 CVM 与 DB 分叉。

### 5. DB 覆盖与回滚

- `replaceInstanceModelBindings` 在写入前保存 `instanceModelBindingsSnapshot`：
  - 旧 `instances.ai_model_id`
  - 旧 `instances.custom_model_config`
  - 旧 `instance_models` 绑定集合
- 覆盖策略：
  - 删除目标实例全部旧 `instance_models`
  - 按请求写入新的 primary + fallback
  - 同步 `instances` 上的 primary 字段
- `restoreInstanceModelBindings` 在 TAT 失败后恢复快照；`restoreInstanceModelBindingsToCVM` 在多模型下发部分成功后 best-effort 重放旧绑定到 CVM；`cleanupDesiredModelBindingsFromCVM` 覆盖无旧绑定的全新实例回滚路径，按 provider key 清理本次新模型配置。

### 6. 并发与单项失败隔离

- 批量执行使用 `sync.WaitGroup` + `chan struct{}` semaphore。
- 并发上限 `adminBatchSetModelConcurrency = 5`，参考现有管控端批量操作的保守并发值。
- 缺失目标直接返回 failed result。
- 单项失败只写该目标 result，不影响其他目标。
- 请求级校验通过后始终返回 HTTP 200 和 `results` 数组。

### 7. 单实例接口兼容

- `/openclaw/set-model` 成功响应保持：

```json
{"ok": true, "provider": "...", "model_id": "..."}
```

- 批量能力不改变用户侧单实例接口返回字段。
- 当前 review 要求的 `context_len` 解析错误检查仅在当前接口补 `strconv.Atoi` 错误返回；`context_len <= 0` 仍保持既有默认值语义。

---

## 与 Plan 差异

- 本 SOP 是在代码实现与多轮 review 后补齐，Implement 阶段未新增开发，只记录当前实现。
- 与 Plan 一致：批量接口采用覆盖语义、单项失败隔离、DB 快照回滚、OpenAPI 文档和集成测试覆盖。
- 与 Plan 差异：无新的实现差异。

---

## 检查项

- [x] `gofmt` 格式化通过（最近一次代码调整已执行 `gofmt -w controller/openclaw_model.go controller/admin_instances_test.go`）
- [x] `go vet ./...` 无错误（2026-07-09 14:51:55 执行通过）
- [x] 写接口已添加审计日志（`instance_admin_batch_set_model`）
- [x] 数据库变更已同步 `sql/init.sql` + migration SQL（无 schema 变更，不适用）
- [x] 使用 `model.DB(r.Context())` / 传入 `ctx` 的 DB 访问模式，批量异步执行使用 detach context 保留请求上下文
- [x] 无硬编码密钥/配置
- [x] 用户可见文案使用 i18n Key；新增 Key 已同步 `en.go` 英文翻译
