# 02. Plan — get-config 接口方案设计

> 基于 iWiki 定稿 §5.A.4 + Clarify 结论 + 代码现状探查。改动遵循 CLAUDE.md / CODEBUDDY.md 红线。

---

## 一、改动文件清单

| 文件 | 改动 | 说明 |
|------|------|------|
| `model/local_agent_cls_credential.go` | **新增** | `LocalAgentCLSCredential` model（gorm.Model + Identifier + ConfigType + SecretID + SecretKey），含 `LocalAgentCLSCredentialTable` 常量 |
| `model/db.go` | 改 | `allModels` 切片追加 `&LocalAgentCLSCredential{}`（AutoMigrate + 迁移校验共用） |
| `model/migrate.go` | 改 | `MigrateFromSQLite` 内新增 `migrateTable[LocalAgentCLSCredential](...)`（通过 `checkMigrationCoverage` 校验） |
| `sql/init.sql` | 改 | 新增 `local_agent_cls_credentials` 建表语句（MySQL 全新部署用） |
| `sql/0715-add-local-agent-cls-credential.sql` | **新增** | 增量迁移（现有 MySQL 升级用）；命名 MMDD 按目标 Release 日期调整 |
| `i18n/keys.go` | 改 | 新增 `MsgGetConfigCredentialNotReady` |
| `i18n/en.go` | 改 | 注册上述 key 的英文翻译 |
| `controller/local_agent.go` | 改 | 新增 `HandleLocalAgentGetConfig` handler |
| `main.go` | 改 | 注册 `GET /local-agent/get-config` → `WithOpenAPI`（只读，无 WithAudit） |
| `docs/API.md` | 改 | 新增接口文档（参数表 + 响应字段表，严格遵循 CLAUDE.md 参数表格式） |

---

## 二、关键设计

### 2.1 Model（model/local_agent_cls_credential.go）

```go
package model

import "gorm.io/gorm"

// LocalAgentCLSCredential 本地 agent 拉取 CLS 公网上报配置所需的永久 AK/SK。
// 按租户隔离（identifier），不复用 site_config.CVMSecret*（那是对云 API 的凭据）。
// 凭据由运维按租户 SQL 写入，明文落库（按需求不加密）。
type LocalAgentCLSCredential struct {
	gorm.Model
	Identifier string `gorm:"index;not null;default:''" json:"-"` // 多租户标识；回调自动注入/过滤
	ConfigType string `gorm:"type:varchar(32);not null;default:'cls';index:idx_lacc_ident_type,unique,priority:1"`
	SecretID   string `gorm:"type:varchar(256);not null"`
	SecretKey  string `gorm:"type:varchar(512);not null"` // 明文落库（按需求不加密）
}

const LocalAgentCLSCredentialTable = "local_agent_cls_credentials"
```

唯一索引 `(identifier, config_type)`：每租户每 config_type 一行（一期每租户一个 cls）。

### 2.2 Handler（controller/local_agent.go）

```go
// GET /local-agent/get-config?config_type=cls
func HandleLocalAgentGetConfig(w http.ResponseWriter, r *http.Request) {
	jsonAPI(w)
	if r.Method != http.MethodGet {
		writeError(w, r, http.StatusMethodNotAllowed, ErrMethodNotAllowed)
		return
	}
	user := requireLogin(w, r)
	if user == nil {
		return
	}
	if !ensureLocalAgentAllowed(w, r, user) { // 复用两层白名单，拒绝已写 403
		return
	}
	ctx := r.Context()

	configType := r.URL.Query().Get("config_type")
	if configType == "" || configType != "cls" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgInvalidConfigType, configType))
		return
	}

	// endpoint（公网）
	endpoint := fmt.Sprintf("%s.cls.tencentcs.com", CVMRegion)

	// topic_id（实时查，不落库）
	clsResult, err := CheckCLSClawServiceOpened(ctx)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgInternalError))
		return
	}
	if clsResult == nil || clsResult.TopicId == "" {
		writeError(w, r, http.StatusBadRequest, hcommon.I18nError(i18n.MsgCLSServiceNotEnabled))
		return
	}

	// secret（按租户查）
	var cred model.LocalAgentCLSCredential
	if err := model.DB(ctx).Where("config_type = ?", configType).First(&cred).Error; err != nil {
		// 查不到 / 读取失败 → 5xx（凭据未配置）
		writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgGetConfigCredentialNotReady))
		return
	}

	// 固定值字段（install_cmd/update_cmd 先留空串常量，值后续由用户填入）
	resp := map[string]any{
		"cls": map[string]any{
			"endpoint":    endpoint,
			"topic_id":    clsResult.TopicId,
			"secret_id":   cred.SecretID,
			"secret_key":  cred.SecretKey,
			"user_id":     user.ID,
			"user_name":   user.Username,
			"install_cmd": localAgentCLSInstallCmd,  // 空串常量预留
			"update_cmd":  localAgentCLSUpdateCmd,   // 空串常量预留
		},
	}
	jsonOK(w, resp)
}
```

常量预留（在 local_agent.go 顶部或 model 内）：
```go
// 固定值，install_cmd / update_cmd 由用户后续填入具体字符串
const (
	localAgentCLSInstallCmd = ""
	localAgentCLSUpdateCmd  = ""
)
```

> ⚠️ `model.DB(ctx)` 已自动走 identifier 回调按当前租户过滤，无需手动加 WHERE identifier。

### 2.3 路由（main.go）

```go
http.HandleFunc("/local-agent/get-config", controller.WithOpenAPI(controller.HandleLocalAgentGetConfig))
```

只读 GET，不加 `WithAudit`（与 `/local-agent/availability` 同风格）。

### 2.4 i18n

- 复用：`MsgInvalidConfigType`（已存在，`config_type 无效: %s`）、`MsgCLSServiceNotEnabled`（已存在，`CLS 服务未开启，请先开启 CLS 服务`）、`MsgInternalError`
- 新增：`MsgGetConfigCredentialNotReady = Key{string: "CLS 凭据未配置，请联系管理员"}`，并在 `en.go` 注册英文翻译

### 2.5 SQL

init.sql（建表，置于 local_instance 表附近）：
```sql
CREATE TABLE IF NOT EXISTS `local_agent_cls_credentials` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `identifier` varchar(191) COLLATE utf8mb4_unicode_ci DEFAULT '',
  `config_type` varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'cls',
  `secret_id` varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL,
  `secret_key` varchar(512) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_lacc_ident_type` (`identifier`,`config_type`),
  KEY `idx_lacc_identifier` (`identifier`),
  KEY `idx_lacc_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

增量迁移 `sql/0715-add-local-agent-cls-credential.sql`：同上 `CREATE TABLE`（MMDD 按目标 Release 分支日期调整，当前按 master 最新提交 2026-07-08 暂定 `0708`）。

---

## 三、调用链

```
GET /local-agent/get-config?config_type=cls
  → main.go: WithOpenAPI(HandleLocalAgentGetConfig)
  → requireLogin(w, r)                 // Bearer user token 反查 user
  → ensureLocalAgentAllowed(w, r, user) // ① feature_allowlist ② SiteConfig.LocalAgentEnabled → 拒绝 403
  → 校验 config_type == "cls"          // 否则 400
  → endpoint = <CVMRegion>.cls.tencentcs.com
  → CheckCLSClawServiceOpened(ctx)      // 实时查 topic；nil/空 → 400 MsgCLSServiceNotEnabled
  → model.DB(ctx).Where("config_type=?", "cls").First(&cred) // 按租户过滤；err → 500 MsgGetConfigCredentialNotReady
  → 组装 {cls:{...install_cmd,update_cmd...}} → jsonOK
```

---

## 四、测试用例设计（自然语言，先于实现）

### 4.1 单元测试（controller/local_agent_get_config_test.go）

| # | 场景 | 输入 | 预期 |
|---|------|------|------|
| UT1 | 未登录 | 无 token | 401 |
| UT2 | 白名单未放行（feature_allowlist 有记录但 identifier 不在） | 已登录，allowlist 拒绝 | 403 `MsgLocalAgentNotAllowed` |
| UT3 | SiteConfig.LocalAgentEnabled=false | 已登录且 allowlist 过 | 403 `MsgLocalAgentNotAllowed` |
| UT4 | config_type 缺失 | `?` | 400 `MsgInvalidConfigType` |
| UT5 | config_type 非法（如 `smh`） | `?config_type=smh` | 400 `MsgInvalidConfigType` |
| UT6 | CLS 服务未开通（CheckCLSClawServiceOpened 返回 nil） | 合法 cls + CLS 关 | 400 `MsgCLSServiceNotEnabled` |
| UT7 | CLS 返回 err | mock 返回 error | 500 `MsgInternalError` |
| UT8 | 凭据表无数据 | 合法 cls + CLS 开 + DB 空 | 500 `MsgGetConfigCredentialNotReady` |
| UT9 | **正常路径** | 合法 cls + CLS 开 + DB 有凭据 | 200，`cls.endpoint/topic_id/secret_id/secret_key/user_id/user_name/install_cmd/update_cmd` 全返回；install_cmd/update_cmd 为 `""` |
| UT10 | 租户隔离 | 租户 A 的 token 查，DB 仅有租户 B 的凭据 | 500 `MsgGetConfigCredentialNotReady`（identifier 回调过滤后查不到） |

> mock 方式：gomonkey patch `CheckCLSClawServiceOpened` 与 `model.DB`（或构造测试 DB 种子数据）。参考现有 `controller/local_agent_test.go` / `feature_allowlist_test.go` 写法。

### 4.2 集成测试（test/，覆盖新增 API —— 红线 13）

- 新增 `test/local_agent_get_config_test.go`：起服务 + 真实/模拟 CLS，验证 GET 接口 + 参数 + 响应字段。覆盖 config_type 参数传入。

---

## 五、风险与缓解

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| R1 | `CheckCLSClawServiceOpened` 每次调用都打 CLS API，get-config 高频拉取有成本 | 低 | 一期复用现有函数（与 CVM 安装同源），后续可加缓存；不在本期范围 |
| R2 | secret_key 明文落库 + 接口返回，泄露风险 | 中 | 仅本人 token 可拉（user 反查），禁止写入 access_log；运维按租户最小权限写入 |
| R3 | 增量迁移文件名 MMDD 与目标 Release 不符 | 低 | Plan 标注按目标 Release 日期调整 |
| R4 | install_cmd/update_cmd 值为空串时前端行为 | 低 | 用户后续填值；一期空串不影响接口契约 |

---

## 六、待办门禁

- `install_cmd` / `update_cmd` 的具体字符串值由用户后续提供，Implement 阶段填入常量后即完成（已确认先留空串）。
- Implement 前需确认：目标 Release 分支日期 → 决定增量迁移文件 MMDD 前缀。

---

## 七、需求变更记录（2026-07-14 16:27，用户补充）

**变更**：`config_type` 从**必传**改为**非必传**。

- 不传 → 返回全量配置（一期仅 `cls`，即返回 `{cls: {...}}`）
- 传 `cls` → 等价于筛选 `cls`（现有逻辑）
- 传其他类型 → 400 `MsgInvalidConfigType`

**改动点**：
- handler 校验：`configType == "" || configType != "cls"` → `configType != "" && configType != "cls"`
- DB 查询：`Where("config_type = ?", configType)` → `Where("config_type = ?", "cls")`（`config_type` 为空=全量时也返回 cls 块，一期只有 cls）
- 单测 UT4（MissingConfigType）：从「预期 400」改为「预期 200 全量，含 cls 对象」
- 集成测试用例 3：从「缺 config_type → 400」改为「缺 config_type → 200 或 500（环境无凭据）」
- API.md：`config_type` 必填→可选，错误响应描述「缺失或不支持」→「传了但不支持」

**原因**：用户明确 config_type 默认全量、传了当筛选，更贴合「配置中心」语义。

---

## 八、需求变更记录（2026-07-14 16:40，用户补充）

**变更**：新增 `POST /admin/local-agent/cls-credential` 写接口（运维/测试 seed 凭据），使集成测试自洽，不依赖人工/运维 SQL 写入。

**背景**：用户指出集成测试环境没有「人工写入」通道，测试应自洽 seed 数据。参考已有内部/测试专用写接口惯例（不审计），新增 admin 写接口：

- 路由：`POST /admin/local-agent/cls-credential`，`WithOpenAPI` 包装（**不审计**，与用户要求的「IsInternalAccount 类接口不审计」一致）
- 鉴权：`requireAdmin`（admin-token 或 role=admin）
- body：`{ identifier(可选,默认空串), config_type(可选,默认cls), secret_id(必填), secret_key(必填) }`
- 写入：`model.DB(ctx).Where(...).Assign(...).FirstOrCreate(...)` 幂等 upsert（唯一索引 `(identifier, config_type)`）
- 明文存储（设计如此，与 get-config 读取一致）
- 单测：UT11(非admin→403) / UT12(缺secret→400) / UT13(upsert幂等+唯一索引)
- 集成测试：用例 5 前置 `admin_set_local_agent_cls_credential(admin.token, "", ...)` seed，断言 200 全字段（secret_id/secret_key 回显）

**说明**：集成测试环境为单租户，用户 identifier 默认空串，seed 与查询走同一空标识匹配。


---

## 九、需求变更记录（2026-07-14 16:52，用户三点反馈）

1. **还原 .gitignore**：撤回「将 .specs/ 加入 .gitignore」的改动（不该动该文件）。`.specs/` 仍不纳入代码仓库（纯本地 SOP 过程文件）。
2. **cls-credential 写接口加 IsInternalAccount 守卫**：`HandleAdminLocalAgentCLSCredential` 增加 `IsInternalAccount` 判定（抽 `var IsInternalAccountFn` seam），非内部测试账号部署返回 403，避免生产环境明文写接口被滥用。与 `image_id` 隐藏参数的内部账号守卫模式一致。
3. **新增跨租户白名单写接口**：`POST /admin/feature-allowlist`（增，幂等 upsert）+ `DELETE /admin/feature-allowlist`（删），使 `ensureLocalAgentAllowed` 第①层可显式配置，不依赖「空表=全开」默认。写路径走 `model.DBGlobal` 绕过 identifier 回调（全局表）。不审计（WithOpenAPI）。

**改动文件**：
- `controller/internal_account.go`：抽 `var IsInternalAccountFn = IsInternalAccount`（seam）
- `controller/local_agent.go`：`HandleAdminLocalAgentCLSCredential` 加 `IsInternalAccountFn` 守卫 + 新增 i18n key `MsgInternalAccountRequired`
- `controller/admin_feature_allowlist.go`：新增 `HandleAdminFeatureAllowlist`（POST/DELETE 写接口）
- `main.go`：注册 `/admin/feature-allowlist`
- `i18n/keys.go` + `en.go`：`MsgInternalAccountRequired`
- 单测：UT12/13 用 `IsInternalAccountFn` seam 模拟内部账号；UT14（非内部→403）；UT15/16（feature_allowlist 增/删）
- 集成测试 helper：`admin_set_feature_allowlist` / `admin_delete_feature_allowlist`；用例 5 前置显式加白名单
- API.md：get-config 写接口补 IsInternalAccount 守卫说明 + feature_allowlist 写接口文档 + 目录表

---

## 十、补充：cls-credential 写接口加 DELETE（与 feature-allowlist 对称，2026-07-14 17:20）

用户反馈两个写接口不对称（feature-allowlist 有 POST+DELETE，cls-credential 只有 POST）。补 `DELETE /admin/local-agent/cls-credential`：

- `HandleAdminLocalAgentCLSCredential` 重构为 switch(method)：POST → handleAdminSetCLSCredential（原逻辑），DELETE → handleAdminDeleteCLSCredential（按 `(identifier, config_type)` 删行）
- DELETE 同样过 requireAdmin + IsInternalAccountFn 守卫
- 单测 UT17（DELETE 清凭据→行移除，seam 注入内部账号）、UT18（非内部账号 DELETE→403）
- 集成测试 helper `admin_delete_local_agent_cls_credential`；用例 5 末尾演示 DELETE 清理 seed 状态
- API.md：cls-credential 加 DELETE 文档块 + 目录表行

**改动文件**：`controller/local_agent.go`（重构+DELETE 分支）、`controller/local_agent_get_config_test.go`（UT17/18）、`test/scripts/helpers/local_agent.py`（helper）、`test/scripts/local_agent/test_local_agent_get_config.py`（用例演示）、`docs/API.md`（文档+目录）

---

## 十一、install_cmd 等固定值字段定稿（2026-07-14 20:47）

用户确认 4 个全局固定命令字段的具体值（新增 run_cmd / uninstall_cmd）：

| 字段 | 值 |
|------|-----|
| `install_cmd` | `npm install -g tencentcloud-cls-sdk-codebuddy` |
| `run_cmd` | `cls-codebuddy start` |
| `update_cmd` | `npm install -g tencentcloud-cls-sdk-codebuddy` |
| `uninstall_cmd` | `ls-codebuddy uninstall` |

**改动**：
- `controller/local_agent.go`：`localAgentCLSInstallCmd` / `localAgentCLSUpdateCmd` 由空串回填为上述值，新增 `localAgentCLSRunCmd` / `localAgentCLSUninstallCmd` 常量；响应 `resp.cls` 由 2 字段扩为 4 字段（`install_cmd` / `run_cmd` / `update_cmd` / `uninstall_cmd`）
- 集成测试 `test_local_agent_get_config.py` 用例 5 补 4 字段断言
- `docs/API.md` 响应示例 + 字段表更新（4 字段全列，值明确）
- 00-overview.md / 06-it.md / 07-review.md 中「install_cmd/update_cmd 待定值/空串」描述已过时，本提交闭合

**改动文件**：`controller/local_agent.go`、`test/scripts/local_agent/test_local_agent_get_config.py`、`docs/API.md`、`.specs/`

---

## 十二、topic_id 落表、get-config 直接读取（2026-07-14 21:17）

用户确认：topic_id 不再实时调 CLS OpenClawService，改为直接落到 local_agent_cls_credentials 表，get-config 读取返回。

**改动**：
- `model/local_agent_cls_credential.go`：加 `TopicID` 字段（`varchar(64) not null default ''`）
- `sql/init.sql`：建表加 `topic_id` 列
- `sql/0715-add-local-agent-cls-credential.sql`：测试阶段建表文件，直接在建表语句里加 `topic_id` 列（不单独新增 migration 文件）
- `controller/local_agent.go`：
  - `HandleLocalAgentGetConfig` 的 `topic_id` 从 `cred.TopicID` 读（替换 `clsResult.TopicId`）
  - 保留 `localAgentCheckCLSClawServiceOpened` 作为「CLS 服务开通」校验（返回 nil→4xx），但不再用其 TopicId
  - `adminSetCLSCredentialRequest` 加 `TopicID`；upsert `Assign` 含 `TopicID`
- 单测 UT4/UT9：seed 凭据带 topic_id，断言从表读（mock CLS 仅用于通过开通校验）
- IT helper `admin_set_local_agent_cls_credential` 加 `topic_id` 参数；用例 5 seed 传 UUID 并断言返回
- API.md：响应示例 + 字段表 + 写接口 body 更新

**红线合规**：改 GORM model 同步了 init.sql + 测试阶段建表文件（0708）+ migrate.go（LocalAgentCLSCredential 已注册 migrateTable，无外键 remap，无需改）。测试阶段不单独新增增量 migration 文件。

---

## 十三、4 个 cmd 常量值更新（2026-07-15 12:04）

用户更新 install/run/update/uninstall 四个命令字符串：
- install_cmd / update_cmd: `npm install -g tencentcloud-cls-sdk-codebuddy-test --registry https://mirrors.tencentyun.com/npm/`
- run_cmd（模板，含占位符）: `cls-codebuddy setup --endpoint ${endpoint} --topic-id ${topic_id} --secret-id ${secret_id} --secret-key ${secret_key} --service-name ${local_agent_id} --user-name ${user_name} --user-id ${user_id}`
- uninstall_cmd: `cls-codebuddy uninstall-all`

**改动**：
- `controller/local_agent.go`：常量更新；`run_cmd` 改为模板常量 `localAgentCLSRunCmdTmpl` + 新增 `renderCLSRunCmd()` 实时替换已知 6 字段（endpoint/topic_id/secret_id/secret_key/user_name/user_id），`${local_agent_id}` 服务端未知、保留占位符下发给 agent 端自填
- 单测 UT9：run_cmd 断言改为渲染后期望值（含 `${local_agent_id}` 保留）
- IT 用例 5：install/update/uninstall 精确断言新值；run_cmd 断言前缀 + 含 topic_id/secret_id/`${local_agent_id}` 占位符
- API.md：响应示例 + 字段表更新（run_cmd 标注模板渲染语义）

**说明**：`${local_agent_id}` 不在服务端响应里（handler 只有 user 对象，无 agent 自身 id），故保留原样下发，由本地 agent 拉取后自行替换自己的 id。

### 13.1 修正：run_cmd 不渲染，原样返回（2026-07-15 12:14）

用户明确 run_cmd 不需要服务端替换占位符，直接返回原始字符串即可。

- 删除 `renderCLSRunCmd()` 函数与 `localAgentCLSRunCmdTmpl` 模板常量
- `localAgentCLSRunCmd` 直接等于原始模板串（含 `${endpoint}` 等占位符）
- 响应 `run_cmd` 直接返回 `localAgentCLSRunCmd`，不做任何替换
- 单测 UT9 改回精确断言原始串；IT 用例 5 run_cmd 精确断言；API.md 字段表去掉"模板渲染"描述

---

## 十四、topic_id 改回实时查询，db 不再存（2026-07-15 15:04）

用户决定 topic_id 仍由 get-config 实时从 CLS OpenClawService 获取，不落 db。回退 §十二 的落表方案（选 A：彻底删除 db 字段）。

**改动**：
- `model/local_agent_cls_credential.go`：删除 `TopicID` 字段
- `sql/init.sql` + `sql/0715-add-local-agent-cls-credential.sql`：删除 `topic_id` 列
- `controller/local_agent.go`：
  - `topic_id` 改回 `clsResult.TopicId`（实时查）；CLS 校验恢复 `clsResult == nil || clsResult.TopicId == ""` → 4xx
  - `adminSetCLSCredentialRequest` 删除 `TopicID` 字段；upsert `Assign` 不再含 TopicID
- 单测 UT4/UT9：seed 去掉 topic_id，mock CLS 返回 `TopicId`，断言实时值
- IT helper `admin_set_local_agent_cls_credential` 去掉 topic_id 参数；用例 5 seed 不带 topic_id，断言 topic_id 非空（实时查）
- API.md topic_id 字段表改回「实时查询，不落库」

**说明**：写接口不再接受 topic_id（db 无该列）；CLS 服务未开通 → get-config 4xx 的原有语义恢复。

---

## 十五、写接口鉴权收紧 + 覆盖率 + IT 导入修复（2026-07-15 16:37）

用户三项要求：
1. **写接口去掉 WithOpenAPI，仅 admin token 可调用**：`/admin/local-agent/cls-credential` 与 `/admin/feature-allowlist` 路由去掉 `WithOpenAPI` 包装（main.go 234/237 行），只走 `requireAdmin` + `IsInternalAccountFn` 守卫。用户 API Token 在非 OpenAPI 路由下被 `getUserFromToken` 忽略，无 session → 401/403，无法访问。`/admin/feature-allowlist/check`（查询接口）保留 WithOpenAPI。
2. **单测覆盖率 ≥ 60%**：新增 UT19（用户 token 访问写接口→拒）、UT20（feature-allowlist 非内部账号→403）、UT21（feature_allowlist 不支持方法→405）。覆盖率：HandleLocalAgentGetConfig 92.9%、HandleAdminLocalAgentCLSCredential 88.9%、HandleAdminFeatureAllowlist 60.5%、HandleAdminFeatureAllowlistCheck 84.2%。
3. **修复 IT 导入错误**：`helpers/__init__.py` 补导出 `user_get_local_agent_config`/`admin_set_local_agent_cls_credential`/`admin_delete_local_agent_cls_credential`/`admin_set_feature_allowlist`/`admin_delete_feature_allowlist`，解决 `from helpers import user_get_local_agent_config` ImportError。

**改动文件**：main.go、controller/local_agent_get_config_test.go（UT19/20/21）、test/scripts/helpers/__init__.py、docs/API.md（写接口鉴权说明）
