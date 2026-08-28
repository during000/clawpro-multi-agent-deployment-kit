# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

当前 `controller/admin_models.go` 中的 `handleModelConnectivity` 函数采用 **list-models 优先、chat 回退** 的双探活策略：

1. 优先调用 `p.CheckConnectivity(ctx, apiKey, apiBase)` 发起 `GET /models` 请求
2. 仅当 list-models 失败且 `modelID` 非空时，才回退到 `p.CheckConnectivityWithChat(...)`

**问题**：list-models 探活只验证了 endpoint 可达 + API key 有效，但**无法验证 model id 是否正确**。只要 API key 有效，即便用户填了一个不存在的 model id，list-models 探活也会成功返回，前端展示"可联通"，用户实际使用时才发现模型不可用。

---

## 目标

废弃 list-models 探活方式，全部采用 chat 探活，确保 model id 错误时能被检测出来。

验收标准：

- [x] `handleModelConnectivity` 中不再调用 `CheckConnectivity`（list-models 探活）
- [x] 直接使用 `CheckConnectivityWithChat` 进行探活
- [x] `Provider` 接口的 `CheckConnectivity` 方法声明被删除
- [x] OpenAI/Anthropic 两个实现中的 `CheckConnectivity` 方法被删除
- [x] `modelID` 为空时返回参数错误（已有校验，保持不变）
- [x] 所有相关单测通过 `go test ./... -v -race`
- [x] `docs/API.md` 连通性检测说明更新

---

## 范围

| 包含 | 不包含 |
|------|--------|
| `controller/admin_models.go` 的 `handleModelConnectivity` 探活逻辑改造 | 数据库 schema 变更 |
| `controller/provider/provider.go` 删除 `CheckConnectivity` 接口声明 | 新增 API 接口 |
| `controller/provider/openai.go` 删除 `CheckConnectivity` 实现 | `CheckConnectivityWithChat` 方法本身逻辑修改 |
| `controller/provider/anthropic.go` 删除 `CheckConnectivity` 实现 | `resolveConnectivityArgs` 逻辑修改 |
| `controller/admin_models_test.go` 适配 list-models 相关测试 | |
| `controller/openclaw_model_test.go` 适配 list-models 相关测试 | |
| `controller/provider/connectivity_chat_test.go` 适配接口合规测试（如需） | |
| `docs/API.md` 更新连通性检测说明 | |

---

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 当前分支为 `bugfix/connectivity`，没有对应的任务目录。按 SOP 流程，如何处理分支？ | 已确认 | 直接在当前 `bugfix/connectivity` 分支上继续开发并创建任务目录 |
| 2 | 废弃 list-models 探活后，provider 接口的 `CheckConnectivity` 方法（list-models 实现）如何处理？ | 已确认 | 完全删除 `CheckConnectivity` 方法及其所有实现（openai/anthropic） |
| 3 | 既然全部采用 chat 探活，当 `modelID` 为空时的行为如何处理？ | 已确认 | 强制要求 `modelID` 非空，为空时直接返回参数错误（已有校验） |

---

## 约束与依赖

- Go 1.21+，使用 `net/http` 标准库
- 必须遵循项目红线：无裸 SQL、写接口需审计日志、i18n 文案等
- 本次改动不涉及数据库变更，无需 migration SQL
- 本次改动不涉及新增 API 接口，无需新增集成测试用例
- `CheckConnectivityWithChat` 方法本身逻辑不变，仅废弃 `CheckConnectivity` 的调用
