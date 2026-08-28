# 02. Plan — 方案设计

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `model/site_config.go` | 修改 | 新增 `SkillHubEnabled bool` + `SkillHubAPIURL string` 字段 |
| `sql/init.sql` | 修改 | 新增 `skill_hub_enabled` + `skill_hub_api_url` 列（含 COMMENT） |
| `sql/0717-skillhub-site-config.sql` | 新增 | 增量迁移：ALTER TABLE site_configs ADD 两列 |
| `controller/skillhub_proxy.go` | 新增 | 装饰器 `WithSkillHubProxy(localHandler, skillHubHandler)` |
| `controller/skillhub.go` | 新增 | 核心：灰度判断、token 获取、OrgID 获取、缓存、状态查询 handler |
| `controller/admin_skills_skillhub.go` | 新增 | SkillHub 版技能列表 handler + 统一错误处理 |
| `skillhubclient/client.go` | 新增 | SkillHub HTTP 客户端：`NewClient`、`FetchOrgInfo`、`ListSkills` |
| `skillhubclient/adapter.go` | 新增 | 格式转换：`ConvertSkillHubListToHatchery` |
| `controller/skillhub_proxy_test.go` | 新增 | 装饰器+灰度+token缓存 单测 |
| `skillhubclient/client_test.go` | 新增 | client 单测 |
| `skillhubclient/adapter_test.go` | 新增 | adapter 单测 |
| `main.go` | 修改 | 路由注册：`WithSkillHubProxy` 包装 + `/admin/skillhub-status` |
| `docs/API.md` | 修改 | 新增 `/admin/skillhub-status` 接口文档 |

---

## 调用链 / 数据流

### 灰度请求流程

```
用户请求 GET /admin/skills
  → WithSkillHubProxy 装饰器
    → isSkillHubEnabled(r)? 检查 site_configs.skill_hub_enabled
      → true:  HandleAdminSkillsViaSkillHub
        → getSkillHubClient(r)
          → 获取 loginUser.OneIDSub (用户 sub)
          → 获取 snap.OneIDAccountID (租户 tid)
          → getSkillHubAccessToken(ctx, snap, sub, tid)
            → 查缓存 (key: "identifier:sub:tid")
            → 未命中: POST Gateway /api/access-token {sub, tid}
              → Gateway 用自身私钥签 JWT assertion
              → POST OneID /v1/internal/token 换 access_token
              → 缓存 token (提前 60s 刷新)
          → getSkillHubOrgID(ctx, snap, baseURL, token)
            → 查缓存 (key: "identifier")
            → 未命中: GET SkillHub /api/v1/auth/me → 解析 orgId
            → 缓存 OrgInfo
          → skillhubclient.NewClient(baseURL, token, orgID)
        → client.ListSkills(ctx, page, pageSize, keyword)
          → GET SkillHub /api/v1/orgs/{orgId}/skills
        → ConvertSkillHubListToHatchery(resp)
        → 返回 JSON
      → false: HandleAdminSkills (原有逻辑，不修改)
```

### Gateway token 获取流程（openclaw-oneid-gateway 仓库）

```
POST /api/access-token {sub, tid}
  → InternalAuth 中间件验证 X-Internal-Token
  → resolveConfigByRequest(r, store, tid, ...) 查 DB 判断国内/海外
  → generateClientAssertion (ES256 JWT, client 身份)
  → generateUserAssertion (ES256 JWT, 用户身份, 含 nbf + sub_type:union_id)
  → POST OneID /v1/internal/token
    (client_id, client_assertion, assertion, grant_type, aud_app_type=skillhub)
  → 返回 {access_token, expires_in}
```

---

## 数据库变更

| 表 | 字段 | 类型 | 说明 |
|----|------|------|------|
| `site_configs` | `skill_hub_enabled` | `tinyint(1) NOT NULL DEFAULT 0` | 是否启用 SkillHub 迁移（灰度开关） |
| `site_configs` | `skill_hub_api_url` | `text` | SkillHub API 请求地址（迁移代理用） |

**增量迁移文件**：`sql/0717-skillhub-site-config.sql`

**注意**：`skill_hub_api_url` 使用 `TEXT` 而非 `varchar(512)`，因 MySQL row size 限制（已有列占满后 varchar(512) 会超限）。

---

## 测试用例设计（自然语言描述）

> 先于实现编写，Implement 阶段据此编码。
> UT 用例走 `go test`，IT 用例走 Python 集成测试（`test/scripts/`）。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 装饰器灰度未命中 | `skill_hub_enabled=false` | 调用 localHandler | P0 |
| 2 | 装饰器灰度命中 | `skill_hub_enabled=true` | 调用 skillHubHandler | P0 |
| 3 | token 缓存命中 | 缓存未过期 | 不调 Gateway，返回缓存 token | P0 |
| 4 | token 缓存过期 | 缓存已过期 | 调 Gateway 获取新 token | P0 |
| 5 | Gateway 返回错误 | Gateway HTTP 500 | 返回错误，不降级 | P0 |
| 6 | OrgID 缓存命中 | 缓存有值 | 不调 SkillHub /auth/me | P0 |
| 7 | OrgID 首次获取 | 缓存无值 | 调 SkillHub /auth/me，缓存结果 | P0 |
| 8 | SkillHub API 返回错误 | ListSkills HTTP 500 | handler 返回 502 | P0 |
| 9 | 格式转换正常 | SkillHub 标准响应 | 正确 Hatchery 格式 | P0 |
| 10 | 格式转换空响应 | nil 响应 | 返回空数组 `[]` | P1 |
| 11 | 格式转换时间解析失败 | 非法 RFC3339 时间 | 兜底为 time.Now() | P1 |
| 12 | skillhub-status 端点 | 管理员请求 | 返回 enabled + skillhub_url | P1 |
| 13 | skillhub-status URL 推导 | api.skillhub.cn | skillhub.cn | P1 |
| 14 | 用户无 OneIDSub | OneIDSub=nil | 返回错误，不降级 | P0 |
| 15 | 配置缺失 skill_hub_api_url | 空字符串 | 返回错误 | P0 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 灰度关闭走本地 | skill_hub_enabled=false | 返回本地技能列表 | P0 |
| 2 | 灰度开启走 SkillHub | skill_hub_enabled=true | 返回 SkillHub 技能列表 | P0 |

> IT 本次未执行，计划 Phase 3 统一覆盖。

---

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | MySQL row size 超限 | 中 | 高 | `skill_hub_api_url` 使用 TEXT 类型 |
| 2 | 多 pod token 缓存不共享 | 高 | 低 | 提前 60s 刷新，各 pod 独立获取 |
| 3 | Gateway 不可用 | 低 | 高 | 同集群部署，同生命周期 |
| 4 | SkillHub API 超时 | 中 | 中 | Client 30s 超时 + 502 错误返回 |
| 5 | OneID token 频繁过期 | 中 | 低 | 缓存 + 提前 60s 刷新 |
