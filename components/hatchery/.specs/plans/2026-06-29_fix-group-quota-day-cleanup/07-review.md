# 07. Review — Code Review

> 按 `.specs/review.md` 维度审查，记录问题与修复。

---

## 红线检查（全部通过）

| # | 红线 | 结果 | 说明 |
|---|------|------|------|
| 1 | 裸 SQL 接口 | ✅ | 全程 GORM（`SetPolicy`/`DeletePolicy`/`GetPolicyBindingsByGroups` 均走 ORM） |
| 2 | 写接口无审计 | ✅ | 路由 `WithAudit` 包装未变（main.go:607-608） |
| 3 | model 未同步 sql | ✅ | 未改 model schema |
| 4 | 破坏 API 兼容性 | ✅ | 端点/参数/状态码不变，仅补充兼容语义 |
| 5 | 硬编码密钥 | ✅ | 无 |
| 6 | `model.DB` 而非 `model.DB(r.Context())` | ✅ | handler 内均用 `model.DB(r.Context())`；tx 由其 `Begin()` 派生 |
| 7 | 硬编码中文文案 | ✅ | 错误响应复用既有 i18n Key |
| 8 | 新增 i18n Key 无英文翻译 | ✅ | 未新增 Key |
| 9 | 新增 API/参数 IT 未覆盖 | N/A | 未新增接口/参数 |

---

## 维度审查

### 1. 正确性

- **事务一致性**：Set rules 路径 `Begin → SetPolicy → DeletePolicy(paired) → Commit`，任一失败 `Rollback`。与既有写 day 路径同构。✅
- **幂等性**：`DeletePolicyBinding` 走 `Where().Delete()`，0 行不报错；Set rules 清理不存在的 day binding 安全。✅
- **fallback 正确性**：delete 目标不存在 → 查配对 → 配对存在则删配对，配对也无则 422。非配额 key 不触发 fallback。✅
- **边界**：both-present（legacy 脏数据）只删目标（符合"目标存在则只删目标"语义，Plan 已确认）。✅
- **TOCTOU**：delete 路径 query→delete 非事务，存在极窄竞态窗口（并发 set/delete），但 DeletePolicy 幂等，最坏情况是"200 但无行删除"或"422 但目标刚被并发创建"，对 admin 操作可接受。✅
- **早返回**：Set rules 路径改为事务后 `return`，跳过下方 day/global-day 块（均被 `if ConfigKey==...Day` 守卫，不会误入）。✅

### 2. 安全性

- **SQL 注入**：无（全 GORM 参数化）。✅
- **权限**：`requireAdmin` 顶部校验未变。✅
- **敏感信息**：响应仅 `{"ok": true}` 或 i18n 错误。✅

### 3. 规范性

- **DB 访问模式**：handler 用 `model.DB(r.Context())`，tx 派生自同一 ctx。✅
- **并发**：无新增 goroutine。✅
- **日志**：无新增日志；错误经 `writeError` 自动记录。✅
- **i18n**：复用 `MsgPolicyNotConfigured`/`MsgOperationFailed`/`MsgInvalidJSON`/`MsgInvalidTokenQuotaRules`。✅
- **测试规范**：`countPolicyBindings` 用 `context.Background()`（测试 helper，与既有测试一致）；`setupTreeTestDB` 经 `UseDBForTest` 注入 + `t.Cleanup` 归还。✅

### 4. 可维护性

- **函数长度**：`HandleSetGroupPolicy` / `HandleDeleteGroupPolicy` 增量小，未超 80 行红线。✅
- **命名**：`QuotaPolicyPairedKey`、`configKeyToDelete` 清晰。✅
- **复杂度**：与既有 day 路径对称，无额外抽象（避免过度工程）。✅

### 5. 性能

- **查询数**：Set rules 路径 +1 次幂等 DeletePolicy；Delete fallback 最坏 +1 次查询。非 N+1。✅
- **goroutine 泄漏**：无。✅

---

## 发现的问题与处理

### 问题 1（低，已修复）：Set rules 事务内错误处理不一致

**现象**：SetPolicy 失败用 `I18nError(MsgDatabaseOperationFailed)`，DeletePolicy 失败用 `EnsureRichErrorOrPanic(err)`，同一事务块内不一致。

**处理**：统一为 `EnsureRichErrorOrPanic(err)`（保留错误链，便于诊断，与 global_token_quota_day 路径一致）。已修复，build/vet/test 通过。

### 观察 1（低，未修复）：types.go 历史对齐问题

**现象**：`controller/usergroup/types.go` 原文件非 gofmt-clean（`const` 块与 `PolicyDefs` map 对齐不一致）。`gofmt -w` 会重排这些块产生噪声 diff。

**处理**：本任务仅新增 `QuotaPolicyPairedKey` 函数（手工保证 gofmt-clean），未对全文件 gofmt，避免噪声。若 CI 强制 `gofmt -l` 零输出，需单独处理该历史问题（不在本任务范围）。

---

## 结论

✅ **Review 通过**。9 条红线全部通过；1 个低级问题已修复；1 个历史观察项记录但不阻塞。build/vet/22 用例全绿。
