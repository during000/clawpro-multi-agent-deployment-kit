# 08. Commit — 提交

> Commit message 与提交前检查。

---

## 一、Commit Message

```
fix(controller): deprecate list-models probe, use chat probe only

list-models 探活无法检测 model id 是否正确，导致 model id 错误也显示
模型可以联通。废弃 list-models 探活方式，全部采用 chat 探活，确保
model id 正确性也能被验证。

- 删除 Provider 接口的 CheckConnectivity 方法声明
- 删除 OpenAIProvider 和 AnthropicProvider 的 CheckConnectivity 实现
- handleModelConnectivity 直接调用 CheckConnectivityWithChat
- 适配所有相关单元测试（删除回退类测试，改写路径期望）
- 更新 docs/API.md 补充探活方式说明
```

---

## 二、提交前检查

- [x] `go build ./...` 通过
- [x] `go vet ./controller/... ./controller/provider/...` 通过
- [x] `gofmt -l` 无格式问题
- [x] 相关单元测试全部通过
- [x] `docs/API.md` 已更新
- [x] 无 lint 新增错误
- [x] `00-overview.md` Meta 状态将更新为已完成
- [x] 所有 `0N-*.md` 产物文件已就绪

---

## 三、提交文件清单

| 文件 | 改动类型 |
|------|---------|
| `controller/admin_models.go` | 修改 |
| `controller/provider/provider.go` | 修改 |
| `controller/provider/openai.go` | 修改 |
| `controller/provider/anthropic.go` | 修改 |
| `controller/admin_models_test.go` | 修改 |
| `controller/openclaw_model_test.go` | 修改 |
| `controller/provider/ssrf_guard_test.go` | 修改 |
| `docs/API.md` | 修改 |
| `.specs/plans/2026-07-31_connectivity-chat-only/` | 新增（任务目录） |
