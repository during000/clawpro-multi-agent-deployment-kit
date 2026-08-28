# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
make build          # Dev build with race detector
make release        # Production build (CGO_ENABLED=0, stripped, embedded assets)
make clean          # Remove build artifacts

# First run with SQLite (default, creates initial admin user, --region is required)
./hatchery --region ap-guangzhou --init-user admin --init-pass 123456

# Subsequent runs (SQLite)
./hatchery --region ap-guangzhou

# Common flags (SQLite)
./hatchery --region ap-guangzhou -addr :9090 -db app.db --admin-token xxx --secret my-key --domain https://hatchery.example.com

# MySQL mode  — first initialize schema, then start
mysql -u user -p hatchery < sql/init.sql

## MySQL mode (--identifier is required)
./hatchery --db-type mysql --db "user:pass@tcp(host:3306)/hatchery?charset=utf8mb4&parseTime=True&loc=Local" --identifier my-tenant --uin 100001 --region ap-guangzhou --init-user admin --init-pass 123456

# SQLite → MySQL one-time data migration (requires --db-type=mysql and --identifier)
./hatchery --db-type mysql --db "user:pass@tcp(host:3306)/hatchery?charset=utf8mb4&parseTime=True&loc=Local" --identifier my-tenant --region ap-guangzhou --db-migrate /path/to/old-hatchery.db
```

Key flags: `-addr` (listen address, default `:8080`), `-db` (SQLite path or MySQL DSN, default `hatchery.db`), `--db-type` (database type: `sqlite` or `mysql`, default `sqlite`), `--region` (required, e.g. `ap-guangzhou`), `--init-user`/`--init-pass` (required on first run), `--admin-token` (Bearer token for admin API), `--secret` (cookie encryption key), `--uin` (Tencent Cloud UIN, optional, used for STS credential refresh and role ARN), `--identifier` (multi-tenant identifier, **required for MySQL mode**, enables data isolation), `--domain` (base URL of this service), `--email-api-url` (email API endpoint), `--log-json` (JSON log format), `--disable-ui` (API-only mode, no HTML rendering), `--db-migrate` (source SQLite DB path for one-time data migration to MySQL, requires `--db-type=mysql` and `--identifier`).

## Architecture

Go web app using `net/http` (no framework).

**Build tag split (`!release` / `release`):**
- `assets_dev.go`: Reads scripts from `scripts/` dir
- `assets_release.go`: Embeds scripts into binary via `go:embed`

**Layers:**
- `main.go` — Route registration, flag parsing, server startup, HTTP request logging middleware, `forceJSONMiddleware` (for `--disable-ui` mode), graceful shutdown
- `controller/` — HTTP handlers, organized by domain:
  - **Auth**: `auth.go` — login/logout/session/password, shared state (`Store`, `LoadScript`, `AdminToken`, `CVMRegion`, `DisableUI`, etc.), Bearer token auth (`getUserFromToken`), `RegionZones` mapping
  - **Instance management**: `openclaw.go` — CVM instance CRUD, status queries; `openclaw_channel.go` — user-facing channel management (QQ/WeChat/Feishu bots); `openclaw_model.go` — user-facing model selection; `openclaw_skill.go` — user-facing skill management
  - **Admin panel**: `admin_common.go` — `requireAdmin` guard; `admin_users.go` — user CRUD (including batch create); `admin_config.go` — site config, CVM config, VPC/subnet/security-group management; `admin_models.go` — AI model CRUD; `admin_channels.go` — channel toggle management; `admin_images.go` — CVM image management; `admin_instances.go` — global instance monitoring; `admin_usage.go` — usage stats and quota overview; `admin_audit.go` — audit log viewer
  - **LLM Proxy**: `llm_proxy.go` — OpenAI-compatible `/v1/chat/completions` and `/v1/models` endpoints, streaming support, async usage logging; `llm_quota.go` — multi-level quota enforcement (global/model/user); `provider/` — Provider abstraction supporting OpenAI (`openai-completions`) and Anthropic (`anthropic-messages`) backends
  - **Infrastructure**: `tat.go` — Tencent Cloud TAT remote script execution; `sts.go` — STS credential refresh; `email.go` — email sending; `audit.go` — audit middleware (`WithAudit()`); `json_response.go` — content negotiation helpers (`jsonOK`, `jsonError`, `writeError`, `jsonRedirect`); `api_error.go` — API error handling
- `model/` — GORM models over SQLite or MySQL (dual backend). `db.go` (InitDB with dual backend support, auto-migration for SQLite via shared `allModels` list, identifier callback registration for multi-tenant isolation). `migrate.go` (SQLite → MySQL one-time data migration with FK remapping and batch insert). `distlock.go` (distributed lock via MySQL `GET_LOCK()`, no-op for SQLite). `user.go` (User with instance quota, daily token quota, VPC config, soft delete). `instance.go` (CVM instance with proxy token and model binding). `ai_model.go` (AI model config). `ai_channel.go` (AI channel config). `ai_image.go` (CVM image). `site_config.go` (site name, logo, CVM secrets, global config). `llm.go` (LLMUsageLog + DailyUsageSummary). `audit.go` (audit log model)
- `scripts/` — Shell scripts executed on remote instances via TAT. Covers instance init, service check, bot creation (QQ/Feishu), channel/skill/model management. Embedded into release binary.
- `static/` — (deprecated, no longer used)
- `templates/` — (deprecated, no longer used)

> **UI deprecation note:** The HTML/HTMX UI is permanently disabled. `controller.DisableUI` is hardcoded to `true`, and `--disable-ui` is deprecated and forced on. `forceJSONMiddleware` (applied to the entire `DefaultServeMux` in `main.go`) unconditionally sets `Accept: application/json` on every request, so all handlers take their JSON branch and never render HTML fragments. The `templates/` directory and `static/` assets are dead code retained only for historical reference. New work should assume JSON-only responses.

## Key Patterns

- Session-based auth with `gorilla/sessions` cookie store. Login failure counter triggers captcha. Admin API also supports Bearer token auth (`--admin-token`).
- Admin pages use `requireAdmin` guard. Admin user cannot be deleted.
- User-facing pages use `requireLogin` guard (returns JSON 401 when unauthenticated). Full-page handlers that need HTTP redirect use `getLoginUser` + manual `http.Redirect` instead.
- Most responses are JSON: `forceJSONMiddleware` forces `Accept: application/json` on every request. Error responses use `writeError` (returns JSON errors). Exceptions: SSE streams (`/openclaw/auto-channel`, `/v1/chat/completions` streaming), images (`/captcha`, `/logo`, `/favicon.ico`), static assets (`/static/*`), and WebSocket (`/openclaw/vnc-ws-proxy`).
- Default admin credentials seeded on first run via `--init-user` and `--init-pass` flags (required when DB has no users, fatal error if omitted).
- Database: SQLite (default) or MySQL backend (`--db-type`). SQLite uses single file (WAL mode, foreign keys enabled), auto-migrated on startup. MySQL mode requires `--identifier` (mandatory) and manual schema initialization via `sql/init.sql` before first run (`mysql -u user -p hatchery < sql/init.sql`); AutoMigrate is skipped in MySQL mode to avoid concurrent DDL conflicts in multi-instance deployments. `--identifier` enables automatic data isolation via GORM callbacks that auto-inject `WHERE identifier = ?` on queries and set `identifier` on creates. Seed data (default config, channels, models, etc.) is protected by a distributed lock (`GET_LOCK`) in MySQL mode to prevent race conditions during multi-instance startup. `--db-migrate` enables one-time SQLite → MySQL data migration: reads all data from a source SQLite file, remaps foreign keys, and writes into the current MySQL tenant within a single transaction; skips if tenant data already exists. Graceful shutdown flushes usage logs and closes DB.
- CVM instance status is fetched via Tencent Cloud CVM API and returned as JSON.
- Service status is fetched via TAT `RunScript()` which returns raw JSON.
- LLM Proxy uses Bearer token (instance's proxy token) for auth, independent of session. Usage logs are written asynchronously via channel; `InitUsageLogger()` on startup, `FlushUsageLogs()` on shutdown.
- STS credentials are refreshed in the background via `StartSTSRefresher()` (requires `--uin`).

## Audit Logging

- **Mandatory:** Every new write endpoint (POST/PUT/DELETE) MUST have audit logging. This requires two steps:
  1. Add an entry to the `auditRules` map in `controller/audit.go` with an appropriate action name and resource type.
  2. Wrap the handler with `WithAudit()` in `main.go` route registration.
- Omitting audit logging on a write endpoint is considered a bug.

## Database Schema Changes

- **Dual maintenance required:** SQLite uses GORM AutoMigrate (auto-applies on startup), but MySQL uses manually maintained SQL scripts. When modifying GORM models (adding/removing/renaming fields, changing types, adding indexes), you MUST update:
  1. The GORM model struct tags in `model/*.go` (takes effect for SQLite automatically).
  2. The corresponding `CREATE TABLE` statement in `sql/init.sql` (for fresh MySQL deployments).
  3. A new incremental migration file in `sql/` (e.g. `sql/002_add_foo_column.sql`) containing the `ALTER TABLE` / `CREATE TABLE` / `CREATE INDEX` statements needed to upgrade an existing MySQL database. Migration files should be numbered sequentially and include a comment header describing the change.
- **Adding new model to AutoMigrate:** If a new GORM model struct is created, add it to the `allModels` slice in `model/db.go` (shared by SQLite AutoMigrate and MySQL migration validation), add a corresponding `CREATE TABLE` in `sql/init.sql`, AND create a migration file with the same `CREATE TABLE` in `sql/`.
- **TEXT columns with DEFAULT:** MySQL does not allow `DEFAULT` on `TEXT`/`BLOB` columns. If a GORM tag declares `type:text` with a `default`, the `sql/init.sql` version must either use `varchar(N)` (to preserve the default) or drop the `DEFAULT` clause (and handle defaults in application code). Document any such divergence in the comment block at the top of `sql/init.sql`.
- Omitting `sql/init.sql` or the incremental migration file when changing a GORM model is considered a bug.

## Public API Compatibility

- **Breaking changes forbidden:** All public API endpoints (i.e., handlers wrapped with `controller.WithOpenAPI`) are considered stable contracts. The following breaking changes are **NOT allowed**:
  - Removing existing response fields.
  - Changing the semantics or meaning of existing fields (e.g., redefining what a field represents, changing its unit, or altering its value range).
- Adding new fields to requests or responses is allowed, as it is backward-compatible.
- Violating public API compatibility is considered a bug.

## API Documentation

- **Mandatory updates:** When adding or modifying API endpoints, always update `docs/API.md` accordingly.
- **Markdown table formatting:** Tables must be written at the top level (no indentation inside list items), with a blank line before and after each table. Otherwise GitLab will not render them correctly. Example:

```markdown
- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 名称 |

- **响应：** ...
```

- **Parameter table format (strict):** `docs/API.md` is parsed to generate an OpenAPI spec. To ensure correct parsing, request parameter tables MUST follow this exact format:
  1. **Table header** must be exactly: `| 参数 | 类型 | 必填 | 说明 |` (4 columns, no extra columns like "默认值" or "传参方式").
  2. **Parameter names** must NOT be wrapped in backticks. Write `| name |` not `` | `name` | ``.
  3. **"必填" column** values: `是` (required), `否` (optional), or `条件` (conditionally required).
  4. **Type values**: Use `string`, `int`, `uint`, `bool`, `object`, `string[]`, `int[]`, `uint[]`, `array`.
  5. **Default values**: If a parameter has a default, describe it within the "说明" column (e.g. "模式，默认 `day`"), not in a separate column.
  6. **Response field tables** use a different 3-column format (`| 字段 | 类型 | 说明 |`) — these are NOT parsed as request parameters.
  7. **Referencing another endpoint's params** (e.g. "参数同 `POST /other`") is NOT parseable — always write out the full parameter table explicitly.
  8. **Content-Type declaration** should be present for POST endpoints: `- **Content-Type：** \`application/json\`` or `- **Content-Type：** \`application/x-www-form-urlencoded\``.

  Violating these rules will cause the parameter to be missing from the generated OpenAPI spec, which is considered a documentation bug.

## Multi-Tenant Context (ctx-aware) 开发规范

背景：多租户阶段一(openspec/changes/multi-tenant-universe-mode)引入了请求级
`TenantSnapshot`，GORM 回调从 ctx 读取 identifier 做租户隔离。桥接期内现有
`model.DB` 句柄继续可用，但新代码必须遵循以下规范：

### 1. DB 访问

- **HTTP handler 内**：使用 `model.DB(r.Context())` 替代 `model.DB`
- **全局表(无 Identifier 字段)**：使用 `model.DBGlobal(ctx)`，会跳过 identifier 过滤
- **后台任务/一次性逻辑**：使用`context.DetachContext(ctx)`，在 main 中传入 task 启动函数

### 2. 异步 goroutine 必须用 DetachContext

HTTP handler 里启动 goroutine 时，`r.Context()` 会在 handler return 后被 cancel，
直接用 `context.Background()` 会丢失 TenantSnapshot 导致 DB 操作异常。正确写法：

```go
go func(ctx context.Context) {
    // DB / 腾讯云 SDK 调用
    model.DB(ctx).Update(...)
    cli, _ := controller.GetCVMClient(ctx)
}(common.DetachContext(r.Context()))
```

纯计算 / 纯日志类 goroutine 不涉及数据层，无需 DetachContext。

### 3. 腾讯云 SDK Client 不要自己 New

不要直接调用 `cvm.NewClient(credential, region, cpf)`，统一使用：

```go
cli, err := controller.GetCVMClient(ctx)     // CVM
cli, err := controller.GetVPCClient(ctx)     // VPC
cli, err := controller.GetSTSClient(ctx)     // STS
cli, err := controller.GetCLSClient(ctx)     // CLS
cli, err := controller.GetTagClient(ctx)     // Tag
```

### 4. 禁止使用裸 SQL 接口

**禁止**直接调用以下绕过 GORM Model 层的接口。这些调用不会触发 GORM 回调（callbacks），导致多租户 identifier 自动注入、软删除过滤等 hook 机制失效，产生数据越权或逻辑错误。违反此规则视为 bug。

禁止的接口列表：
- `db.Exec()` — 执行裸 SQL，不触发回调
- `db.Raw()` — 裸 SQL 查询，不触发回调
- `db.Table()` — 绕过 Model 检测，identifier 回调无法识别模型字段
- `db.Row()` — 返回 `*sql.Row`，绕过回调链
- `db.Rows()` — 返回 `*sql.Rows`，绕过回调链
- `db.ScanRows()` — 手动扫描行，不经过回调链
- `db.Connection()` — 直接获取底层连接，完全绕过 GORM

正确做法：始终通过 GORM 的 ORM 接口操作数据，例如 `db.Create()`、`db.Save()`、`db.Where().Find()`、`db.Model(&m).Update()` 等。

```go
// ✗ 错误 — hook 不生效
db.Exec("UPDATE users SET name = ? WHERE id = ?", name, id)
db.Raw("SELECT * FROM users WHERE id = ?", id).Scan(&user)
db.Table("users").Where("id = ?", id).Update("name", name)

// ✓ 正确 — 走 GORM 回调链
db.Model(&User{}).Where("id = ?", id).Update("name", name)
db.Where("id = ?", id).First(&user)
db.Save(&user)
```

### 5. 后台任务

后台任务保持各自 goroutine 自管理模式，但从main传入ctx（`task.StartXxx(ctx)` / `controller.StartXxx(ctx)`）。
内部 DB 操作使用传下来的ctx。
