# 02. Plan — 方案设计

> 改动文件清单、调用链、测试用例设计（自然语言）、风险评估。

---

## 一、改动文件清单

### 1. 核心代码（3 个文件）

| 文件 | 行号范围 | 改动 |
|------|---------|------|
| `controller/admin_models.go` | 806-828 | `handleModelConnectivity` 中移除 list-models 探活，直接调用 `CheckConnectivityWithChat`；更新注释 |
| `controller/provider/provider.go` | 118-123 | 删除 `Provider` 接口的 `CheckConnectivity` 方法声明 |
| `controller/provider/openai.go` | 300-344 | 删除 `OpenAIProvider.CheckConnectivity` 方法实现 |
| `controller/provider/anthropic.go` | 1043-1087 | 删除 `AnthropicProvider.CheckConnectivity` 方法实现 |

### 2. 测试文件（3 个文件）

| 文件 | 改动 |
|------|------|
| `controller/admin_models_test.go` | 适配/改写 8 个测试用例 |
| `controller/openclaw_model_test.go` | 适配 3 个测试用例 |
| `controller/provider/connectivity_chat_test.go` | 接口合规测试保留不变（仅验证 `CheckConnectivityWithChat`） |

### 3. 文档（1 个文件）

| 文件 | 改动 |
|------|------|
| `docs/API.md` | 两处 connectivity 文档补充探活方式说明 |

---

## 二、改动详情

### 2.1 `controller/admin_models.go`

**当前逻辑**（806-828 行）：
```go
// 用稍长的 ctx timeout 兜底，防止上游半死状态拖死调用方。
// provider 内部本身已有 10~15s 超时，这里再加 30s 上限即可
// （chat 探活 + list-models 回退最多两次 RTT，预算放宽一点）。
ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
defer cancel()

p := provider.GetProvider(modelType)

var (
    latency  time.Duration
    probeErr error
    probe    string
)

// 优先进行 list-models 探活
latency, probeErr = p.CheckConnectivity(ctx, apiKey, apiBase)
probe = "models"

// 如果 list-models 探活失败，则尝试 chat 探活
if probeErr != nil && modelID != "" {
    latency, probeErr = p.CheckConnectivityWithChat(ctx, apiKey, apiBase, modelID)
    probe = "chat"
}
```

**改为**：
```go
// 用稍长的 ctx timeout 兜底，防止上游半死状态拖死调用方。
// provider 内部本身已有 10~15s 超时，这里再加 30s 上限即可。
ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
defer cancel()

p := provider.GetProvider(modelType)

// 全部采用 chat 探活，确保 model id 正确性也能被验证
latency, probeErr := p.CheckConnectivityWithChat(ctx, apiKey, apiBase, modelID)
probe := "chat"
```

同时更新 `classifyConnectivityError` 函数注释（932 行）：将 "provider.CheckConnectivity" 改为 "provider.CheckConnectivityWithChat"。

### 2.2 `controller/provider/provider.go`

删除接口声明（118-123 行）：
```go
// CheckConnectivity probes the upstream provider to verify that the
// endpoint is reachable and the supplied apiKey is accepted.
// Implementations typically issue a lightweight GET /models request.
// This is the cheapest probe but does not work for providers that
// don't expose a list-models endpoint.
CheckConnectivity(ctx context.Context, apiKey, apiBase string) (latency time.Duration, err error)
```

### 2.3 `controller/provider/openai.go`

删除 `OpenAIProvider.CheckConnectivity` 方法（300-344 行）。

### 2.4 `controller/provider/anthropic.go`

删除 `AnthropicProvider.CheckConnectivity` 方法（1043-1087 行）。

---

## 三、测试用例设计（自然语言）

### 3.1 `controller/admin_models_test.go`

| # | 测试函数 | 当前行为 | 改动方案 |
|---|---------|---------|---------|
| 1 | `TestHandleAdminModelConnectivity_TemporaryCredentialsSuccess` | mock 只处理 `/models`，期望命中 `/models` | 改为 mock 处理 `/chat/completions`，返回 chat completions 格式响应 |
| 2 | `TestHandleAdminModelConnectivity_SavedModelSuccess` | mock 兼容 `/chat/completions` 和 `/models` | 移除 `/models`，只处理 `/chat/completions`，返回 chat completions 格式响应 |
| 3 | `TestHandleAdminModelConnectivity_SavedModelFailureReturnsDetails` | mock 兼容 `/chat/completions` 和 `/models` | 移除 `/models`，只处理 `/chat/completions` |
| 4 | `TestHandleAdminModelConnectivity_AnthropicTemporaryCredentialsSuccess` | mock 只处理 `/v1/models` | 改为 mock 处理 `/v1/messages`，返回 Anthropic messages 格式响应 |
| 5 | `TestHandleAdminModelConnectivity_ListModelsHitWhenChatModelGiven` | 验证 list-models 优先命中，chat 不命中 | **删除**此测试（list-models 探活已废弃），改写为 `TestHandleAdminModelConnectivity_ChatProbeDirectHit`：验证 chat 探活直接命中，不触发 `/models` |
| 6 | `TestHandleAdminModelConnectivity_ChatProbeFallbackOnListModelsFailure` | 验证 list-models 失败回退 chat | **删除**此测试（不再有回退逻辑） |
| 7 | `TestHandleAdminModelConnectivity_NoFallbackWithoutModel` | 验证无 model 时 list-models 失败不回退 | **改写**为 `TestHandleAdminModelConnectivity_NoModelReturns400`：验证无 model 字段时直接返回 400（已有 `resolveConnectivityArgs` 校验），mock server 不应被命中 |
| 8 | `TestHandleAdminModelConnectivity_BothProbesFail` | 验证两路都失败 | **改写**为 `TestHandleAdminModelConnectivity_ChatProbeFail`：验证 chat 探活失败时返回诊断信息，mock 只处理 `/chat/completions` 返回 400 |
| 9 | `TestHandleAdminModelConnectivity_AnthropicChatProbeFallback` | 验证 Anthropic list-models 失败回退 | **删除**此测试（不再有回退逻辑），改写为 `TestHandleAdminModelConnectivity_AnthropicChatProbeDirectHit`：验证 Anthropic chat 探活直接命中 `/v1/messages` |

### 3.2 `controller/openclaw_model_test.go`

| # | 测试函数 | 当前行为 | 改动方案 |
|---|---------|---------|---------|
| 1 | `TestHandleModelConnectivity_TemporaryCredentialsSuccess` | mock 只处理 `/models` | 改为 mock 处理 `/chat/completions` |
| 2 | `TestHandleModelConnectivity_AnthropicTemporaryCredentialsSuccess` | mock 只处理 `/v1/models` | 改为 mock 处理 `/v1/messages` |
| 3 | `TestHandleModelConnectivity_SavedModelVisibilityAllowed` | mock 兼容 `/chat/completions` 和 `/models` | 移除 `/models`，只处理 `/chat/completions` |

### 3.3 `controller/provider/connectivity_chat_test.go`

- `TestProvider_CheckConnectivityWithChat_InterfaceCompliance`：保留不变，删除 `CheckConnectivity` 后仍验证 `OpenAIProvider` 和 `AnthropicProvider` 满足 `Provider` 接口

---

## 四、调用链

```
HandleAdminModelConnectivity (admin)
HandleModelConnectivity (user)
  └─ handleModelConnectivity
       ├─ resolveConnectivityArgs (校验参数，modelID 为空返回 400)
       ├─ provider.GetProvider(modelType)
       └─ p.CheckConnectivityWithChat(ctx, apiKey, apiBase, modelID)  ← 唯一探活方式
            ├─ OpenAIProvider.CheckConnectivityWithChat → POST /chat/completions
            └─ AnthropicProvider.CheckConnectivityWithChat → POST /v1/messages
```

---

## 五、风险评估

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | 删除接口方法可能影响其它调用点 | 中 | 已确认仅 `handleModelConnectivity` 调用 `CheckConnectivity` |
| 2 | 测试用例改动范围较大（11 个用例） | 中 | 逐个适配，删除回退类测试，改写路径期望 |
| 3 | chat 探活比 list-models 探活消耗 token | 低 | 探活请求 `max_tokens=1`，消耗极小 |
| 4 | 某些 provider 不支持 `/chat/completions` 或 `/v1/messages` | 低 | 这是已有方法，原回退逻辑已覆盖此场景，现改为唯一方式后行为一致 |
