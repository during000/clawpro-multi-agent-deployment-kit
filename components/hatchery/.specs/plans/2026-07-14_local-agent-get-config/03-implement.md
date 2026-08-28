# 03. Implement — get-config 接口实现

> 按 Plan 落地 10 个文件改动。`go build ./...` 通过，`go vet ./model/ ./controller/` 无告警。

---

## 一、改动清单与落点

| 文件 | 改动 | 落点 |
|------|------|------|
| `model/local_agent_cls_credential.go` | 新增 | `LocalAgentCLSCredential` struct（gorm.Model + Identifier + ConfigType + SecretID + SecretKey）+ `LocalAgentCLSCredentialTable` 常量 |
| `model/db.go` | 改 | `allModels` 追加 `&LocalAgentCLSCredential{}`（AutoMigrate + 迁移校验共用） |
| `model/migrate.go` | 改 | 新增 `migrateTable[LocalAgentCLSCredential](ctx, srcDB, tx, ids, migrated, nil)`（无外键 remap） |
| `sql/init.sql` | 改 | 新增 `local_agent_cls_credentials` 建表（唯一索引 `idx_lacc_ident_type (identifier, config_type)`） |
| `sql/0715-add-local-agent-cls-credential.sql` | 新增 | 增量迁移（MMDD 按目标 Release 调整，当前暂定 0708） |
| `i18n/keys.go` | 改 | 新增 `MsgLocalGetConfigCredentialNotReady` |
| `i18n/en.go` | 改 | 注册上述 key 英文翻译 |
| `controller/local_agent.go` | 改 | 新增 `HandleLocalAgentGetConfig` + `localAgentCLSInstallCmd` / `localAgentCLSUpdateCmd` 空串常量 |
| `main.go` | 改 | 注册 `GET /local-agent/get-config` → `WithOpenAPI`（只读，无 WithAudit） |
| `docs/API.md` | 改 | 目录表 + 新增接口文档（参数表 + 响应字段表，遵循 CLAUDE.md 格式） |

---

## 二、关键实现细节

### 2.1 Model
- 复用 `gorm.Model`（自带 ID/CreatedAt/UpdatedAt/DeletedAt，软删除）
- `Identifier` 用 `json:"-"` + index，由 identifier 回调自动注入/过滤（与 `LocalInstanceInfo` 同风格）
- 唯一索引 `(identifier, config_type)`：每租户每 config_type 一行
- SecretKey 明文（`varchar(512)`），按需求不加密

### 2.2 Handler 逻辑顺序
```
requireLogin → ensureLocalAgentAllowed（两层白名单，拒绝已写 403）
  → config_type 校验（非空且 == "cls"，否则 400 MsgInvalidConfigType）
  → endpoint = <CVMRegion>.cls.tencentcs.com
  → CheckCLSClawServiceOpened(ctx)（nil/空 → 400 MsgCLSServiceNotEnabled；err → 500）
  → model.DB(ctx).Where("config_type = ?").First(&cred)（err → 500 MsgLocalGetConfigCredentialNotReady）
  → 组装 {cls:{endpoint,topic_id,secret_id,secret_key,user_id,user_name,install_cmd,update_cmd}} → jsonOK
```
- `model.DB(ctx)` 自动走 identifier 回调，无需手动加 WHERE identifier（满足红线 5）
- 复用了现有 `ensureLocalAgentAllowed` / `CheckCLSClawServiceOpened` / `MsgInvalidConfigType` / `MsgCLSServiceNotEnabled`，仅新增 1 个 i18n key

### 2.3 install_cmd / update_cmd
- 常量 `localAgentCLSInstallCmd = ""` / `localAgentCLSUpdateCmd = ""`，当前空串
- 用户后续给具体值后，直接替换这两个常量即可，接口契约不变

### 2.4 路由
- `GET /local-agent/get-config` 用 `WithOpenAPI`（只读，与 `/local-agent/availability` 同风格，无 `WithAudit`）

---

## 三、与 Plan 的差异
- 无实质差异。i18n key 命名定为 `MsgLocalGetConfigCredentialNotReady`（Plan 草稿写 `MsgGetConfigCredentialNotReady`，实现时归入 local_agent 命名组，语义一致）。
- 增量迁移文件名 `0708`（按 master 最新提交 2026-07-08），合入具体 Release 前按需改 MMDD 前缀。

---

## 四、验证
- `go build ./...` → exit 0
- `go vet ./model/ ./controller/` → 无告警
- 红线自查：裸 SQL 无（全 GORM）；只读接口无 WithAudit（符合只读语义）；handler 用 `model.DB(ctx)`；新增 i18n key 已加 en.go 翻译；API.md 已更新；model 改动同步 init.sql + 增量迁移 + MigrateFromSQLite。

---

## 五、下步
- 进 04. UT：实现 Plan 设计的 10 个单测 + 集成测试（红线 13 要求新增 API 必须有集成测试覆盖）
- install_cmd/update_cmd 值由用户后续提供，填值动作不阻塞流程
