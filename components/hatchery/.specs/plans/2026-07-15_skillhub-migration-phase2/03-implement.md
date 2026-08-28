# 03. Implement — 实现记录

---

## 关键实现细节

### 1. 装饰器模式（`controller/skillhub_proxy.go`）

`WithSkillHubProxy(localHandler, skillHubHandler)` 返回一个 `http.HandlerFunc`，内部根据 `isSkillHubEnabled(r)` 决定走哪个 handler。扩展新接口只需：写 `XxxViaSkillHub` handler + 路由注册改一行。

### 2. Token 获取（`controller/skillhub.go` — `getSkillHubAccessToken`）

- 缓存结构：`sync.Mutex + map[string]*skillHubTokenEntry`，key = `"identifier:sub:tid"`
- 与 `getOneIDAppToken`（`oneid_unified.go`）同模式
- 调 Gateway `POST /api/access-token`，传 `{sub, tid}`，Header 带 `X-Internal-Tenant` + `X-Internal-Token`（HMAC 签名）
- 缓存提前 60s 刷新（`expiresAt.Add(-60*time.Second)`）

### 3. OrgID 获取（`controller/skillhub.go` — `getSkillHubOrgID`）

- 缓存结构：`sync.Mutex + map[string]*skillhubclient.OrgInfo`，key = `"identifier"`
- 首次调用通过 SkillHub `GET /api/v1/auth/me` 获取
- 一个 OneID 租户在 SkillHub 的 orgId 唯一，缓存后不再重复查询

### 4. SkillHub Client（`skillhubclient/client.go`）

- 无状态设计：`NewClient(baseURL, token, orgID)` 每次请求由 controller 创建
- `ListSkills` 使用 `url.QueryEscape` 对 keyword 编码
- `FetchOrgInfo` 解析嵌套 JSON：`user.enterprise.orgId`

### 5. 格式适配（`skillhubclient/adapter.go`）

- `Categories` / `VisibilityGroups` 初始化为 `[]`（非 nil），前端兼容
- `LastTask` / `SecurityScan` 保持 `nil`（指针类型）
- 时间解析失败兜底 `time.Now()`
- `VisibilityType` 默认 `"all"`

### 6. 错误处理（`controller/admin_skills_skillhub.go`）

- `skillHubClientOrFail`：获取客户端失败返回 502，不降级
- `skillHubAPIFail`：API 调用失败返回 502
- 使用 `hcommon.I18nRichError` 包装错误

### 7. 前端 URL 推导（`controller/skillhub.go` — `HandleSkillHubStatus`）

- 从 `skill_hub_api_url` 去掉 `://api.` 前缀得到前端 URL
- 例：`https://api.skillhub.cn` → `https://skillhub.cn`

### 8. 数据库字段

- `SkillHubEnabled`：`gorm:"type:tinyint(1)"`，默认 `false`
- `SkillHubAPIURL`：`gorm:"type:text"`，默认 `""`
- `ApplySiteConfigDefaults` 中设默认值 `SkillHubAPIURL: "https://api.skillhub.cn"`

---

## 与 Plan 差异

| 差异 | 原因 |
|------|------|
| 方案演进：从 if-branch → 装饰器模式 | 用户反馈"侵入性增加 if 代码块不优雅"，改用装饰器 |
| 方案演进：从直接签 JWT → Gateway 代理 | Hatchery 无 OneID 私钥，改为通过 Gateway 转发 |
| 方案演进：从 `OneIDTokenProvider` 对象 → `sync.Mutex+map` | 用户要求匹配现有 `getOneIDAppToken` 缓存模式 |
| `skill_hub_api_url` 从 `varchar(512)` → `TEXT` | CI/CD MySQL row size 超限 |
| 不删除 `InvalidateSkillHubCache` | 用户决定不需要缓存失效逻辑 |
| 端点名从 `/admin/skillhub-state` → `/admin/skillhub-status` | 用户确认命名 |

---

## 检查项

- [x] `gofmt` 格式化通过
- [x] `go vet ./...` 无错误
- [x] 写接口已添加审计日志（本任务无新增写接口）
- [x] 数据库变更已同步 `sql/init.sql` + migration SQL
- [x] 使用 `model.DB(r.Context())` 而非 `model.DB`（本任务无直接 DB 操作）
- [x] 无硬编码密钥/配置
- [x] 用户可见文案使用 `i18n.T()`，新增 Key 已同步 `en.go` 英文翻译
