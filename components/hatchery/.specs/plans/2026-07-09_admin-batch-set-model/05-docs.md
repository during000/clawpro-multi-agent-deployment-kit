# 05. Docs — 文档更新

---

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 修改 | 新增管控端 `POST /admin/instances/batch-set-model` 接口说明，并在接口总览中登记 |
| `docs/API.md` | 修改 | 补充批量覆盖语义：顶层模型为 primary，`fallbacks` 为最终 fallback 集合，本接口不是 add-model |
| `docs/API.md` | 修改 | 补充限制：单次最多 20 个目标、并发 5、running 实例、本地实例、混合 Agent 类型、fallback runtime 与 OpenClaw 3.28.x 限制 |
| `docs/API.md` | 修改 | 补充目标选择语义：`ids` / `instance_ids` 二选一，`ids` 优先 |
| `docs/API.md` | 修改 | 补充请求参数、请求示例、响应 `results[]` 字段和请求级错误响应 |
| `docs/API.md` | 修改 | 修正相关参数表格式：`uint[]` / `string[]` / `array`，必填列使用 `是` / `否` / `条件` |
| `docs/openapi.json` | 生成 | 由 `python3 test/api_md_to_openapi.py` 从 `docs/API.md` 生成；该文件按仓库规则为生成产物，不作为本次手写文档维护对象 |

---

## 检查项

- [x] `docs/API.md` 已更新（新增/修改 API 已记录）
- [x] `.specs/docs/` 相关文档已同步（本仓库当前无本需求对应 `.specs/docs/` 专项文档；使用 CODEBUDDY SOP 产物承载）
- [x] 参数表格式符合 CODEBUDDY.md / CLAUDE.md 要求（4 列、无反引号包裹参数名、必填列合法）
- [x] OpenAPI 生成通过
- [x] 新增 endpoint 的 IT 覆盖将在 `06-it.md` 记录

---

## 验证命令

### OpenAPI 生成

```bash
python3 test/api_md_to_openapi.py
```

结果：PASS。

```text
Found 381 endpoint definitions
Unique paths: 363
Endpoints with parameters: 294
Total operations: 373
Validation:
  Paths: 363
  Operations: 373
Writing docs/openapi.json ... Done
```

### 参数表规范检查

检查接口：

- `POST /admin/instances/set-model`
- `POST /admin/instances/batch-set-model`

检查规则：

- 参数名不使用反引号包裹
- 类型属于允许集合：`string` / `int` / `uint` / `bool` / `object` / `string[]` / `int[]` / `uint[]` / `array`
- 必填列只使用：`是` / `否` / `条件`

结果：PASS。

```text
POST /admin/instances/set-model: params=13 bad=[]
POST /admin/instances/batch-set-model: params=14 bad=[]
```

---

## 文档语义核对

| 项 | 结论 |
|----|------|
| 批量接口是否明确是覆盖语义 | 是，文档写明最终模型集合等于 primary + `fallbacks` |
| 是否避免误导为 add-model | 是，文档明确“本接口不是 add-model” |
| `ids` / `instance_ids` 条件必填是否说明清楚 | 是，表格中写明二选一与 `ids` 优先 |
| 自定义 Agent 类型限制是否与实现一致 | 是，文档写明按兼容内置运行时处理，未配置兼容运行时时单项失败 |
| fallback runtime 限制是否与实现一致 | 是，OpenClaw 支持 fallback，Hermes / LightclawACE 仅支持一个模型 |
| OpenClaw 3.28.x 限制是否记录 | 是 |
| 请求级错误与单项失败是否区分 | 是 |
| `/set-model` 兼容响应是否保留 | 是，相关文档保留 `provider` / `model_id` |
