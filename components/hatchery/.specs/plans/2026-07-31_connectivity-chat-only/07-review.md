# 07. Review — 代码审查

> Code Review：问题与修复。

---

## 一、审查清单

| # | 检查项 | 结果 |
|---|--------|------|
| 1 | 是否有裸 SQL 调用 | ✅ 无 |
| 2 | 写接口是否注册审计日志 | ✅ 不涉及（本次仅改连通性检测逻辑） |
| 3 | 是否有 GORM model 变更 | ✅ 无 |
| 4 | 是否破坏公共 API 兼容性 | ✅ 无（接口契约不变） |
| 5 | handler 是否使用 `model.DB(r.Context())` | ✅ 不涉及 DB 操作变更 |
| 6 | 是否使用 `controller.GetXxxClient(ctx)` | ✅ 不涉及云 SDK |
| 7 | 是否硬编码配置信息 | ✅ 无 |
| 8 | 异步 goroutine 是否使用 DetachContext | ✅ 不涉及异步场景 |
| 9 | 是否在 master/main 分支开发 | ✅ 在 `bugfix/connectivity` 分支 |
| 10 | API 文档是否更新 | ✅ 已更新 `docs/API.md` |
| 11 | 面向用户文案是否使用 i18n.T() | ✅ 不涉及文案变更 |
| 12 | 新增 i18n.Key 是否有英文翻译 | ✅ 不涉及 |
| 13 | 新增 API 接口/参数是否集成测试覆盖 | ✅ 不涉及新增 |

---

## 二、代码审查结论

### 核心逻辑

- `handleModelConnectivity` 改动简洁正确：直接调用 `CheckConnectivityWithChat`，移除了 list-models 探活和回退逻辑
- `probe` 变量保留用于日志，值固定为 `"chat"`，与日志输出一致
- `resolveConnectivityArgs` 中 `modelID` 为空校验已存在，确保 model 字段必填

### 接口删除

- `Provider` 接口删除 `CheckConnectivity` 方法声明
- `OpenAIProvider` 和 `AnthropicProvider` 删除对应实现
- 删除后编译通过，无其他调用点引用 `CheckConnectivity`

### 测试覆盖

- 9 个 admin 测试用例已适配/改写，覆盖成功、失败、无 model、SSRF 阻断等场景
- 3 个 openclaw 测试用例已适配
- 2 个 SSRF 测试删除（方法已删除，由 `CheckConnectivityWithChat` 版本覆盖）
- 接口合规测试保留

### 文档

- `docs/API.md` 两处连通性检测文档已补充探活方式说明

### Lint 检查

- 所有 lint 问题均为预存的 HINT/INFO 级别（`interface{}` → `any` 风格提示），本次改动未引入新问题

---

## 三、审查结论

**通过**，无阻塞性问题。
