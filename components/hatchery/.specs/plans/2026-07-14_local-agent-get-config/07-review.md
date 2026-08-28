# 07. Review — 代码审查

> 逐条核对 CLAUDE.md 红线 1-5 + MEMORY 项目红线 1-9 + 跨 run 记忆里的 hatchery 红线。结论：全部合规，无阻断性问题。

---

## 一、红线逐项核对

### CLAUDE.md 多租户红线（1-5）

| # | 要求 | 本次改动 | 结论 |
|---|------|----------|------|
| 1 | HTTP handler 用 `model.DB(r.Context())` | handler `ctx := r.Context()` → `model.DB(ctx).Where(...).First(&cred)` 走 identifier 回调 | ✅ |
| 2 | 异步 goroutine 用 `common.DetachContext` | 本接口纯同步 GET，无 goroutine | N/A |
| 3 | 腾讯云 Client 用工厂 | 复用既有 `CheckCLSClawServiceOpened`（内部已用 `GetCLSClient` 工厂），未自行 New | ✅ |
| 4 | 禁止裸 SQL（db.Exec/Raw/Table） | 运行时全走 GORM；`sql/init.sql` / `sql/0708-*.sql` 仅为 DDL 建表，非运行时裸 SQL | ✅ |
| 5 | 后台任务从 main 传 ctx | 本接口无后台任务 | N/A |

### MEMORY 项目红线（1-9）

| # | 要求 | 本次改动 | 结论 |
|---|------|----------|------|
| 1 | 禁止裸 SQL | 同 CLAUDE 红线 4 | ✅ |
| 2 | 写接口需 `WithAudit` 包装 | get-config 是 **GET 只读查询**，与 `availability` 同款仅 `WithOpenAPI`（无 audit）。红线 2 针对「写接口」，本接口不写 | ✅ |
| 3 | 改 model 同步 `init.sql` + 增量 migration + `MigrateFromSQLite` | `allModels` ✅；`init.sql` 建表 ✅；`sql/0715-add-local-agent-cls-credential.sql` 增量 ✅；`MigrateFromSQLite` 加 `migrateTable[LocalAgentCLSCredential]` ✅ | ✅ |
| 4 | handler 用 `model.DB(r.Context())` | 同 CLAUDE 红线 1 | ✅ |
| 5 | 腾讯云 Client 用工厂 | 同 CLAUDE 红线 3 | ✅ |
| 6 | 文案硬编码 → i18n | 所有 `writeError` 走 `hcommon.I18nError(i18n.MsgXxx)`；无硬编码中文 | ✅ |
| 7 | 新增 i18n.Key 加 en.go | `MsgLocalGetConfigCredentialNotReady` 在 `keys.go` + `en.go` 双落 | ✅ |
| 8 | 异步 goroutine 用 `DetachContext` | 本接口无异步 | N/A |
| 9 | 改 API 更新 `docs/API.md` + 集成测试覆盖 | `docs/API.md` 已更新（目录表 + 接口块，错误响应全覆盖）；集成测试 `test_local_agent_get_config.py` 已补（5 用例） | ✅ |

---

## 二、代码质量（MEMORY 风格规范）

| 项 | 标准 | 实际 | 结论 |
|----|------|------|------|
| 函数行数 | ≤ 80 行 | `HandleLocalAgentGetConfig` 72 行（含注释/空行） | ✅ |
| 行宽 | ≤ 120 字符 | 最长行 `fmt.Sprintf("%s.cls.tencentcs.com", CVMRegion)` 等均在 120 内 | ✅ |
| 语义冲突自检（2026-07-03 教训） | merge 后查新分支写法对齐 | 本 handler 为只读查询，无 `SaveSelectedFields`/`updateFields` 模式，无 merge 冲突风险 | ✅ |
| WithOpenAPI vs WithAudit | 只读查询与 availability 一致 | get-config 走 `WithOpenAPI`，与 availability 同模式（设计使然） | ✅ |

---

## 三、一致性核对

- **i18n key 命名**：`MsgLocalGetConfigCredentialNotReady`（与 `MsgLocalAgentNotAllowed` 等 local_agent 系列前缀一致）✅
- **错误响应文案**：`CLS 凭据未配置，请联系管理员` 与 API.md 文档 `500 {"error": "CLS 凭据未配置，请联系管理员"}` 一致 ✅
- **API.md ↔ handler 状态码**：401/403/400/400/500/405 双向一致 ✅
- **增量 SQL ↔ init.sql**：建表字段、联合唯一索引 `idx_lacc_ident_type (identifier, config_type)`、二级索引完全对齐 ✅
- **MigrateFromSQLite**：`migrateTable[LocalAgentCLSCredential]` 用 `nil` remap（无外键，正确）✅

---

## 四、待用户决策项（非违规，按 2026-07-14 约定）

- `install_cmd` / `update_cmd` 当前为**空串常量**（`localAgentCLSInstallCmd = ""`），等用户给具体字符串值后替换常量即可，不影响接口契约。
- 集成测试用例 5 在 200 分支未断言 `install_cmd != ""`（因当前为空），值填入后可补该断言（可选）。

---

## 五、Review 结论

✅ **无阻断性问题，可进入 08. Commit。**

建议提交前最后一步：跑一次完整 `go build ./...` + `go vet` 确认（已实现于 03/04 步骤，均通过）；集成测试需 CI 真实环境验证（本地无服务）。
