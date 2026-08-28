# 02. Plan — 方案设计

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `controller/auth_passwordless.go` | 新增 | 两个 JSON API、随机凭证生成/摘要、可信域名链接构造、功能白名单与 Session 处理 |
| `controller/auth_passwordless_test.go` | 新增 | Handler 鉴权、校验、并发单次消费、Session 和安全属性测试 |
| `controller/auth.go` | 修改 | 抽取账号密码登录与免登录共用的本地 Session 建立逻辑，保持现有登录语义 |
| `controller/audit.go` | 修改 | 为申请和消费两个 POST 路由注册审计规则 |
| `model/passwordless_login_token.go` | 新增 | 租户级凭证摘要模型、创建、原子消费及过期清理 |
| `model/passwordless_login_token_test.go` | 新增 | 模型的有效期、并发消费、租户隔离和清理测试 |
| `model/feature_allowlist.go` | 修改 | 新增 `passwordless-login` 类型并集中注册各 feature type 的空表策略；调用方只传 feature type |
| `model/db.go` | 修改 | 将新模型加入 `allModels`，供 SQLite AutoMigrate 和 schema 校验使用 |
| `model/migrate.go` | 修改 | 不迁移两分钟有效的一次性凭证，仅登记 migration coverage |
| `sql/init.sql` | 修改 | 新部署 MySQL 的 `passwordless_login_tokens` 建表语句 |
| `sql/0721-passwordless-login.sql` | 新增 | 目标 2026-07-21 Release 的存量 MySQL 增量建表脚本 |
| `main.go` | 修改 | 注册两个路由，写接口同时接入 `WithAudit`，管理接口接入 `WithOpenAPI` |
| `i18n/keys.go` | 修改 | 新增白名单未开放、凭证无效等用户可见错误 Key |
| `i18n/en.go` | 修改 | 为新增 Key 注册英文翻译 |
| `docs/API.md` | 修改 | 记录申请/消费接口、Fragment 链接契约、状态码与安全约束 |
| `test/scripts/passwordless_login/test_passwordless_login.py` | 新增 | 真实部署下申请、消费、登录态和重放拒绝的端到端验证 |
| `../openclaw-enterprise-fronted/client/src/pages/PasswordlessLogin.tsx` | 新增 | 公开免登录落地页：提取并立即移除 Fragment、消费凭证、展示处理中/错误态并跳转 |
| `../openclaw-enterprise-fronted/client/src/App.tsx` | 修改 | 注册公开路由 `/passwordless-login` |
| `../openclaw-enterprise-fronted/client/src/services/auth.ts` | 修改 | 新增 `authApi.passwordlessLogin(token)`，使用 `/api` Axios 实例和 Cookie |
| `../openclaw-enterprise-fronted/client/src/types/api.ts` | 修改 | 新增免登录成功响应类型 |
| `../openclaw-enterprise-fronted/i18n` | 不修改 | 前端源码使用 `t()` 标记新增文案；生成与翻译由前端 i18n 流程负责 |
| `../openclaw-enterprise-fronted/docs/login-logic.md` | 修改 | 增补第四种登录模式、路由和前后端时序 |

Hatchery 仓库自身 UI 已永久停用，不修改其 `templates/` 或 `static/`；实际落地页在相邻 `openclaw-enterprise-fronted` SPA 中实现。
## 调用链 / 数据流

### 申请链接

```text
POST /admin/passwordless-login-link
  → requireAdmin
  → IsFeatureAllowed(type=passwordless-login, tenant identifier)
  → model.DB(r.Context()) 查询未软删目标 User
  → crypto/rand 生成 32 字节随机凭证
  → SHA-256 生成 64 字符十六进制摘要
  → 清理当前租户已过期记录
  → 保存摘要、UserID、ExpiresAt（不保存明文）
  → 使用 TenantSnapshot.Domain 构造绝对 URL
  → 返回 link / expires_in / expires_at
```

### 消费链接

```text
OpenClaw Enterprise 前端打开
  /passwordless-login#passwordless_login_token=<token>
  → PasswordlessLogin 页面捕获 token
  → history.replaceState 立即清除地址栏 Fragment
  → POST /api/auth/passwordless-login {token}
  → Nginx /api 前缀改写到 Hatchery /auth/passwordless-login
  → Hatchery 检查 FeatureAllowlist(type=passwordless-login)
  → SHA-256 摘要
  → 查询未过期凭证记录
  → DELETE ... WHERE id=? AND token_hash=? AND expires_at>now
  → 仅 RowsAffected == 1 的并发请求获得消费权
  → 查询目标 User 当前状态
  → 覆盖建立 hatchery-session，清除旧 OneID 身份字段
  → 返回 {ok:true, redirect:\"/\", role}
  → 前端 window.location.replace(\"/\") 完整刷新
  → App 启动时 authApi.check() 从新 Session 同步真实用户信息
```

凭证经条件删除后即失效。并发请求可以同时读到记录，但只有一个请求能删除一行并建立 Session；其余请求统一返回无效凭证。

## 接口契约

### `POST /admin/passwordless-login-link`

- 鉴权：复用其它 `/admin` API 的 `requireAdmin`；支持管理员用户 API Token、管理员 Session 和进程 AdminToken，普通用户返回 `403`。
- Content-Type：`application/json`。
- 请求：

```json
{
  \"user_id\": 123
}
```

- 成功：`200 application/json`。

```json
{
  \"link\": \"https://tenant.example.com/passwordless-login#passwordless_login_token=<opaque-token>\",
  \"expires_in\": 120,
  \"expires_at\": \"2026-07-21T15:00:00Z\"
}
```

- `link` 只使用 TenantSnapshot 中可信的 `Domain`，不读取 `Host` / `X-Forwarded-Host`。
- 失败：缺失或非法参数 `400`；未认证 `401`；非管理员或白名单未开放 `403`；用户不存在/已封禁 `404`；可信域名未配置或数据库失败 `500`。

### `POST /auth/passwordless-login`

- 鉴权：请求前无需 Session/Bearer Token，一次性凭证本身是认证因子。
- Content-Type：`application/json`。
- 请求：

```json
{
  \"token\": \"<opaque-token>\"
}
```

- 成功：`200`，同时写入 `hatchery-session` Cookie。

```json
{
  \"ok\": true,
  \"redirect\": \"/\",
  \"role\": \"user\"
}
```

- 凭证缺失或长度非法返回 `400`；伪造、过期、已消费及跨租户凭证统一返回 `401`，不暴露具体失败原因；功能白名单关闭返回 `403`；数据库或 Session 保存失败返回 `500`。
- 若浏览器已有登录态，成功消费后覆盖为目标用户，并删除 `oneid_sid`、`oneid_sub`、`oneid_amr`、`user_id`、`login_failures` 等旧身份字段。

## 实现决策

- 功能白名单复用全局 `feature_allowlists`，新增 type 但不新增 CRUD API/UI；具体 identifier 由运维配置，migration 不写死客户数据。
- `IsFeatureAllowed` 根据 feature type 读取 model 中集中配置的空表策略：现有 `local-agent` 保持零记录默认开放，`passwordless-login` 零记录默认关闭；存在记录时均仅命中 identifier 放行。申请和消费都检查，保证运维移除白名单后尚未使用的链接也无法登录。
- 原始凭证使用 `crypto/rand` 生成 256 bit 熵，URL-safe 无 padding 编码；数据库和日志中只出现 SHA-256 摘要。
- 明文凭证放 URL Fragment 而非 Query，浏览器向服务端请求落地页时不会携带 Fragment，避免进入 access log 和 Referer。
- 不使用进程内 map：多 Pod 共享数据库，条件删除是单次消费的唯一真相源。
- 消费成功后复用账号密码登录的本地 Session 语义；不创建用户 API Token，不引入会话级功能权限。
- 创建新链接不主动吊销同一用户此前尚未过期的链接；每条链接独立有效、独立单次消费。创建时清理当前租户过期记录，避免表无限增长。
- 两个 POST 路由都注册审计；审计和访问日志不得记录明文凭证。

## 前端实现

### 公开落地页

- 在 `App.tsx` 增加无需 `ProtectedRoute` 的 `/passwordless-login`，由 SPA fallback 正常承载；现有 Nginx `try_files ... /index.html` 无需修改。
- 新页面首次 effect 用 `URLSearchParams(window.location.hash.slice(1))` 读取 `passwordless_login_token`，把 token 保存到函数局部变量后立即用 `history.replaceState` 删除 Fragment。
- 使用 `useRef` 保证组件生命周期内只发起一次消费请求，防止重复 render/effect 导致单次凭证被第二次提交。
- 页面不把 token 写入 React state、Zustand、localStorage、sessionStorage、日志或 DOM。
- 调用 `authApi.passwordlessLogin(token)` 时设置 `silentError: true`，由页面展示稳定错误态，不触发全局 toast 叠加。
- 成功后不手工构造 `userInfo`，直接 `window.location.replace(response.redirect || \"/\")`；新页面加载时复用 `App.tsx` 现有 `authApi.check()` 同步真实 `user_id`、username 和 role。
- 缺少 token 或后端返回 4xx/5xx 时停留在落地页，展示“登录失败”和后端安全错误文案，提供“返回首页”按钮；不得自动重试一次性 token。

### API 与类型

```ts
export interface PasswordlessLoginResponse extends HatcheryOkResponse {
  role: 'admin' | 'user';
}

authApi.passwordlessLogin(token)
// POST /api/auth/passwordless-login
// JSON { token }, withCredentials=true, silentError=true
```

### 页面状态

| 状态 | 展示 | 行为 |
|------|------|------|
| `processing` | Logo、加载图标、`登录中...`，`aria-live=polite` | 只发送一次消费请求 |
| `failed` | `登录失败`、安全错误文案、`返回首页` | 不重试；按钮使用 `location.replace('/')` |
| `success` | 不驻留 | 立即按后端 `redirect` 完整刷新 |

复用 `DefaultLoginLogo`、`useSiteLogo`、现有按钮/Card 设计和已有 i18n 文案，国内站与国际站保持相同布局。
## 数据库变更

新增租户表 `passwordless_login_tokens`：

| 字段 | MySQL 类型 | GORM 约束 | 说明 |
|------|-----------|-----------|------|
| `id` | `bigint unsigned` | primary key, auto increment | 主键 |
| `identifier` | `varchar(191)` | index, not null, default `''` | 多租户标识，由 GORM callback 自动写入和过滤 |
| `token_hash` | `char(64)` | unique index, not null | SHA-256 十六进制摘要，不保存明文 |
| `user_id` | `bigint unsigned` | index, not null | 目标本地用户 ID |
| `expires_at` | `datetime(3)` | index, not null | 两分钟绝对过期时间 |
| `created_at` | `datetime(3)` | not null | 签发时间 |

- 不设置数据库外键，保持现有迁移和软删除约定；消费时重新查询 User，确保签发后被封禁/删除的用户不能登录。
- `token_hash` 使用全局唯一索引；随机碰撞时创建失败并返回内部错误，不降级复用。
- SQLite→MySQL 切换时不迁移一次性凭证，所有未消费链接直接失效。
- 原子消费使用 GORM `Where(...).Delete(&PasswordlessLoginToken{})`，禁止 `Exec` / `Raw` / `Table`。
- `sql/0721-passwordless-login.sql` 的 `0721` 与目标分支 `Release/2026_07_21` 日期一致。
## 测试用例设计（自然语言描述）

> Implement 阶段按下表编码。UT 使用内存/临时 SQLite 和 `model.UseDBForTest`；IT 使用 K8s 中真实 Hatchery、AdminToken 和独立 Cookie Session。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| UT-01 | 创建凭证记录 | tenant A、用户 1、已计算摘要、未来两分钟过期时间 | 记录带 tenant identifier 落库，字段正确，库中不存在明文 token | P0 |
| UT-02 | 有效凭证消费 | 正确摘要、`expires_at > now` | 返回目标记录并删除一行，之后查询不到该摘要 | P0 |
| UT-03 | 并发消费同一凭证 | 两个 goroutine 同时消费同一摘要 | 恰好一个成功，另一个得到统一无效凭证结果 | P0 |
| UT-04 | 过期凭证消费 | `expires_at <= now` | 不删除记录，不返回用户身份，判定无效 | P0 |
| UT-05 | 跨租户消费 | tenant B 上下文消费 tenant A 的摘要 | 查询不到记录且 tenant A 数据保持不变 | P0 |
| UT-06 | 过期清理 | 同租户一条过期、一条有效；另一个租户一条过期 | 仅删除当前租户过期记录，保留有效记录和其它租户记录 | P1 |
| UT-07 | 申请成功 | 有效管理员认证、白名单放行、现存用户、可信 Domain | 200；返回绝对 HTTPS 链接、`expires_in=120`、RFC3339 过期时间；数据库只保存链接 token 的摘要 | P0 |
| UT-08 | 申请鉴权矩阵 | 无 Token、错误 Token、普通用户 Token、管理员用户 Token、管理员 Session、进程 AdminToken | 前两类 401；普通用户 403；三种管理员认证均成功 | P0 |
| UT-09 | 申请参数校验 | 非 POST、非法 JSON、缺失/零值 `user_id` | 分别返回 405/400，不创建记录 | P1 |
| UT-10 | 目标用户不可用 | 不存在 ID、软删除用户 | 404，不签发链接 | P0 |
| UT-11 | 申请时白名单未开放 | `passwordless-login` 空表，或仅存在其它 identifier | 均返回 403，不签发链接；Local Agent 空表开放的原有行为不变 | P0 |
| UT-12 | 可信域名缺失 | TenantSnapshot.Domain 为空 | 500，不签发可用链接 | P1 |
| UT-13 | 消费成功 | 未登录请求携带有效 token | 200；返回 `redirect=/` 与目标用户最新 role；响应设置 `hatchery-session`，凭证被删除 | P0 |
| UT-14 | 消费参数校验 | 非 POST、非法 JSON、空 token、超长 token | 分别返回 405/400，数据库状态不变 | P1 |
| UT-15 | 无效凭证统一错误 | 随机伪造、已过期、其它租户 token | 均返回相同 401 错误，不泄露原因 | P0 |
| UT-16 | 重放已消费凭证 | 首次成功后再次提交同一 token | 首次 200，第二次 401，第二次不改变 Session | P0 |
| UT-17 | 消费前白名单被移除 | 签发后让当前 identifier 不再命中 allowlist | 403，凭证不被消费；运维恢复白名单且仍未过期时可继续使用 | P1 |
| UT-18 | 用户签发后被删除 | 有效 token 指向已软删除用户 | 不建立 Session，返回统一 401，凭证不可用于登录 | P0 |
| UT-19 | 覆盖既有登录态 | 请求携带另一用户 Session 和 OneID 字段后消费有效 token | Session 改为目标用户，旧 OneID/失败计数/user_id 字段清除，identifier 刷新 | P0 |
| UT-20 | 现有账号密码登录回归 | 正确用户名密码 | 保持 200、`redirect=/`、role 和原 Session 字段语义 | P0 |
| UT-21 | 审计规则完整 | 检查两个新路径的 `auditRules` | action/resource 均已注册且路由使用 `WithAudit` | P1 |
| UT-22 | 中英文错误 | 白名单拒绝或凭证无效，分别使用 zh/en | 返回对应语言的稳定错误文案 | P1 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| IT-01 | 申请并完成免登录 | AdminToken + 测试用户 `user_id`；从 link Fragment 提取 token；独立 `requests.Session` 调用消费接口 | 申请 200、TTL=120；消费 200 并收到 Cookie；随后 `GET /` 返回目标用户身份 | P0 |
| IT-02 | 单次使用 | 重复提交 IT-01 的 token | 第二次返回 401，原已建立 Session 仍属于目标用户 | P0 |
| IT-03 | 严格 AdminToken | 使用测试管理员的用户 API Token 申请 | 返回 403；证明普通管理员凭证不能签发链接 | P0 |
| IT-04 | 新参数校验 | 申请缺 `user_id`；消费缺 `token` | 均返回 400；覆盖两个新增请求参数 | P1 |
| IT-05 | 伪造凭证 | 提交随机 URL-safe token | 返回 401，不设置 Session Cookie | P0 |

IT 环境不等待两分钟验证过期，过期边界由可控时钟的 UT 覆盖；IT 专注真实路由、AdminToken、Cookie 和接口覆盖率。

### 前端验证

| # | 场景 | 操作 | 预期输出 | 级别 |
|---|------|------|---------|------|
| FE-01 | 完整免登录 | 浏览器打开后端生成的 `/passwordless-login#...` 链接 | 仅一次消费请求；地址栏立即移除 Fragment；成功后跳首页并显示目标用户登录态 | P0 |
| FE-02 | 缺少凭证 | 直接访问 `/passwordless-login` | 不发送消费请求，展示登录失败和返回首页按钮 | P1 |
| FE-03 | 无效/过期/已消费 | 打开携带不可用 token 的链接 | 展示后端安全错误，不弹重复 toast、不自动重试、不设置登录态 | P0 |
| FE-04 | 已有其它用户 Session | 登录 A 后打开用户 B 的有效链接 | 成功后完整刷新，首页显示 B，旧用户前端缓存被 `authApi.check()` 覆盖 | P0 |
| FE-05 | 前端 i18n | 检查新增用户文案 | 所有新增文案均使用 `t()` 标记；不在功能提交中维护生成翻译 JSON | P1 |

- 静态验证：在前端仓库运行 `pnpm check` 和 `pnpm build`。
- UI 验证：启动前端并用真实浏览器执行 FE-01 至 FE-04；检查 Network 中 token 只出现在 POST JSON Body，不出现在请求 URL。
## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 链接被截获后在合法用户前消费 | 中 | 高 | 256 bit 随机凭证、两分钟 TTL、单次原子消费、Fragment 传递、日志不落明文 |
| 2 | 多 Pod 并发导致同一链接成功两次 | 中 | 高 | 共享数据库条件删除，以 `RowsAffected == 1` 作为唯一成功条件；并发 UT |
| 3 | 跨租户域名消费其它租户链接 | 低 | 高 | token 表含 Identifier，所有查询使用 `model.DB(r.Context())` 自动隔离；签发与消费双重白名单 |
| 4 | 未配置白名单导致免登录全量开放 | 低 | 高 | 空表策略与 feature type 在 model 中集中关联；调用方无法传错策略，UT 同时覆盖免登录空表拒绝与 Local Agent 空表开放兼容性 |
| 5 | 原始 token 进入日志、数据库或 Referer | 中 | 高 | 数据库仅存摘要；Fragment 不随 HTTP 请求发送；敏感 JSON key 使用 `token` 触发现有日志脱敏 |
| 6 | 前后端 Fragment/路由契约不一致 | 中 | 高 | 固定 `/passwordless-login#passwordless_login_token=...`；同一 Plan 同步修改前端路由、service、类型和登录文档，并做浏览器端到端验证 |
| 7 | 目标用户在签发后被封禁仍可登录 | 低 | 高 | 消费成功抢占后重新按当前租户查询未软删 User，失败不建立 Session |
| 8 | Pod 时钟偏差影响两分钟边界 | 低 | 中 | 使用 UTC 绝对时间并依赖集群 NTP；过期判断统一为 `expires_at > now` |
| 9 | 过期记录累积 | 低 | 中 | 每次申请前按当前租户、索引字段清理过期记录；模型 UT 验证隔离 |
| 10 | 前端仓库 detached HEAD 上已有用户修改 | 中 | 中 | Implement 前从 `c0f409e6` 创建对应任务分支；保留且不触碰现有 `Dockerfile` 修改 |
