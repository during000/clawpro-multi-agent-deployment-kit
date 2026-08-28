# 03. Implement — 实现细节

> 关键实现细节、特殊处理说明。

---

## 一、核心代码改动

### 1.1 `controller/admin_models.go` — `handleModelConnectivity`

**改动前**：先调 `CheckConnectivity`（list-models），失败且有 modelID 时回退 `CheckConnectivityWithChat`。

**改动后**：直接调 `CheckConnectivityWithChat`，`probe` 固定为 `"chat"`。

```go
// 全部采用 chat 探活，确保 model id 正确性也能被验证
latency, probeErr := p.CheckConnectivityWithChat(ctx, apiKey, apiBase, modelID)
probe := "chat"
```

同时更新了两处注释：
- ctx timeout 注释：移除"chat 探活 + list-models 回退最多两次 RTT"的描述
- `classifyConnectivityError` 函数注释：`provider.CheckConnectivity` → `provider.CheckConnectivityWithChat`

### 1.2 `controller/provider/provider.go` — `Provider` 接口

删除 `CheckConnectivity` 方法声明（6 行注释 + 1 行签名）。

### 1.3 `controller/provider/openai.go` — `OpenAIProvider`

删除 `CheckConnectivity` 方法实现（45 行），更新 `CheckConnectivityWithChat` 注释（移除与 `CheckConnectivity` 的对比描述）。

### 1.4 `controller/provider/anthropic.go` — `AnthropicProvider`

删除 `CheckConnectivity` 方法实现（45 行），更新 `CheckConnectivityWithChat` 注释（移除与 `CheckConnectivity` 的对比描述）。

---

## 二、测试改动

### 2.1 `controller/admin_models_test.go`（9 个测试用例）

| 原测试 | 改动 |
|--------|------|
| `TemporaryCredentialsSuccess` | mock 从 `/models` 改为 `/chat/completions`，返回 chat 格式 |
| `SavedModelSuccess` | 移除 `/models` 兼容，只处理 `/chat/completions`，返回 chat 格式 |
| `SavedModelFailureReturnsDetails` | 移除 `/models` 兼容 |
| `AnthropicTemporaryCredentialsSuccess` | mock 从 `/v1/models` 改为 `/v1/messages`，返回 messages 格式 |
| `ListModelsHitWhenChatModelGiven` | **删除**，改写为 `ChatProbeDirectHit`：验证 chat 直接命中，`/models` 不被命中 |
| `ChatProbeFallbackOnListModelsFailure` | **删除**（不再有回退逻辑） |
| `NoFallbackWithoutModel` | **改写**为 `NoModelReturns400`：验证无 model 字段直接 400，上游不被命中 |
| `BothProbesFail` | **改写**为 `ChatProbeFail`：验证 chat 探活失败返回诊断信息，`/models` 不被命中 |
| `AnthropicChatProbeFallback` | **改写**为 `AnthropicChatProbeDirectHit`：验证 Anthropic chat 直接命中 `/v1/messages` |

### 2.2 `controller/openclaw_model_test.go`（3 个测试用例）

| 测试 | 改动 |
|------|------|
| `TemporaryCredentialsSuccess` | mock 从 `/models` 改为 `/chat/completions` |
| `AnthropicTemporaryCredentialsSuccess` | mock 从 `/v1/models` 改为 `/v1/messages` |
| `SavedModelVisibilityAllowed` | 移除 `/models` 兼容，只处理 `/chat/completions` |

### 2.3 `controller/provider/ssrf_guard_test.go`（2 个测试删除）

| 测试 | 改动 |
|------|------|
| `TestOpenAICheckConnectivity_SSRFBlocked` | **删除**（方法已删除，SSRF 由 `CheckConnectivityWithChat` 版本覆盖） |
| `TestAnthropicCheckConnectivity_SSRFBlocked` | **删除**（同上） |

### 2.4 `controller/provider/connectivity_chat_test.go`

`TestProvider_CheckConnectivityWithChat_InterfaceCompliance` 保留不变，仍验证接口合规。

---

## 三、与 Plan 差异

无差异，完全按 Plan 执行。

---

## 四、验证结果

```bash
go build ./controller/... ./controller/provider/...   # ✅ 通过
go vet ./controller/... ./controller/provider/...      # ✅ 通过
gofmt -l <所有改动文件>                                 # ✅ 无格式问题
go test ./controller/ -run '<connectivity tests>' -v -race  # ✅ 全部通过
go test ./controller/provider/ -v -race                     # ✅ 全部通过
```
