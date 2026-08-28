# Hatchery 用户管理接口 — 统一账号模式 OneID 映射

> **适用范围**：统一账号模式（`OneIDAppID != ""`）下，Hatchery 用户管理操作与 OneID 接口的对应关系。
>
> **模式判断**：`site_configs.one_id_app_id` 非空 = 统一账号模式

---

## 调用架构

```
┌──────────────────────────────────────────────────────────────────────────┐
│                                                                          │
│  OpenAPI 接口（自建应用 Token）— Hatchery 直调 OneID                      │
│  ─────────────────────────────────────────────────                       │
│  Hatchery ──── Token 获取 ────► {ONEID_TOKEN_ENDPOINT}                   │
│           ──── 用户 CRUD ─────► {ONEID_API_BASE_URL}/openapi/v1/...     │
│                                                                          │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  Internal 接口（套件 tenant Token）— 必须走 Gateway 代理                  │
│  ────────────────────────────────────────────────────                    │
│  Hatchery ──── HMAC 认证 ────► Gateway /api/reset-password              │
│           ──── HMAC 认证 ────► Gateway /api/add-role-users              │
│                                                                          │
│  原因：套件应用的私钥（JWK）仅存储在 Gateway 侧，                         │
│        Hatchery 无法自行获取套件 tenant token。                           │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 接口映射总览

| # | Hatchery 接口 | 路由 | HTTP | 统一模式 OneID 操作 | OneID 接口 | Token 类型 | 调用方式 |
|---|---|---|---|---|---|---|---|
| 1 | 创建用户 | `/admin/create` | POST | 创建用户 + 添加角色 | `POST /openapi/v1/contacts/users` | 自建应用 | Hatchery 直调 |
| 2 | 批量创建用户 | `/admin/batch-create` | POST | 循环创建用户 + 添加角色 | `POST /openapi/v1/contacts/users`（逐个） | 自建应用 | Hatchery 直调 |
| 3 | 删除用户(软删除) | `/admin/delete` | DELETE | 停用用户 | `POST /openapi/v1/contacts/users/batch_disable` | 自建应用 | Hatchery 直调 |
| 4 | 硬删除用户 | `/admin/hard-delete` | DELETE | 删除用户 | `DELETE /openapi/v1/contacts/users/:union_id` | 自建应用 | Hatchery 直调 |
| 5 | 恢复用户 | `/admin/restore` | POST | 启用用户 | `POST /openapi/v1/contacts/users/:union_id/enable` | 自建应用 | Hatchery 直调 |
| 6 | 重置密码(管理员) | `/admin/reset-password` | POST | 重置密码 | Gateway `POST /api/reset-password` | 套件 (t-) | 通过 Gateway |
| 7 | 更新用户 | `/admin/update-user` | PUT | 更新用户信息 | `PATCH /openapi/v3/contacts/users/:union_id` | 自建应用 | Hatchery 直调 |
| 8 | 角色添加 | (创建用户/角色变更触发) | — | 角色绑定 | Gateway `POST /api/add-role-users` | 套件 (t-) | 通过 Gateway |
| 9 | 角色移除 | (角色变更触发) | — | 角色解绑 | Gateway `POST /api/remove-role-users` | 套件 (t-) | 通过 Gateway |
| 10 | 用户登录 | `/v1/authn/enterprise` | POST | OneID 登录验证 | `POST {API_BASE}/v1/authn/enterprise` | 无 | 透明代理 |
| 11 | 获取加密配置 | `/v1/authn/encrypt_setting` | GET | 加密公钥获取 | `GET {API_BASE}/v1/authn/encrypt_setting` | 无 | 透明代理 |
| 12 | 用户自助改密码 | `/v1/authn/enterprise/password:reset` | POST | 登录流中重置密码 | `POST {API_BASE}/v1/authn/enterprise/password:reset` | 无 | 透明代理 |
| 13 | 密码验证(获取state_token) | `/user/v1/user/enterprise/password:verify` | POST | 验证密码获取state_token | `POST {API_BASE}/user/v1/user/enterprise/password:verify` | 无 | 透明代理 |
| 14 | 导出 Token | `/admin/export-tokens` | GET | — | 无（本地操作） | — | — |
| 15 | 禁用 Token | `/admin/token/disable` | POST | — | 无（本地操作） | — | — |
| 16 | 启用 Token | `/admin/token/enable` | POST | — | 无（本地操作） | — | — |
| 17 | 用户 Token 管理 | `/admin/user-token` | POST | — | 无（本地操作） | — | — |
| 18 | 用户限额 | `/admin/user-limit` | POST | — | 无（本地操作） | — | — |
| 19 | 用户 VPC | `/admin/user-vpc` | POST | — | 无（本地操作） | — | — |
| 20 | 部门列表 | `/admin/departments` | GET | — | 无（本地操作） | — | — |

---

## 调用方式分类

| 类别 | 接口 | Token | 调用方 | 原因 |
|------|------|-------|--------|------|
| **OpenAPI** | 创建/删除/更新/停用/启用用户 | 自建应用 Token | **Hatchery 直调** OneID | 自建应用凭证由 Hatchery 持有，可直接调 |
| **Internal** | 重置密码、角色添加/移除用户 | 套件 tenant Token (t-) | **通过 Gateway** | 套件私钥(JWK)仅存 Gateway，Hatchery 无法获取 tenant token |
| **透明代理** | 登录、加密配置、自助改密码、密码验证 | 无需 Token | **Hatchery 代理** OneID base API | 用户侧接口，直接转发请求/响应，Cookie 做域名转化 |

---

## 详细说明

### 1. 创建用户 (`POST /admin/create`)

**双写顺序**：先调 OneID 创建 → 添加角色 → 写本地

```
Hatchery                                OneID
   │                                      │
   ├─ 获取自建应用 Token (本地缓存)         │
   │  POST {ONEID_TOKEN_ENDPOINT}          │
   │  {client_id, client_secret,           │
   │   grant_type: "client_credentials"}   │
   │                                       │
   ├───────────────────────────────────────►│
   │  POST {API_BASE}/openapi/v1/contacts/users
   │  {name, username, email, password}    │
   │                                       │
   │◄──────────────────────────────────────┤
   │  {union_id, user_id, username}        │
   │                                       │
   ├─ (若 role=admin)                       │
   │  POST Gateway /api/add-role-users     │
   │  {account_id, role_id:"1400000",      │
   │   union_ids:[union_id]}               │
   │                                       │
   ├─ 写本地 DB (one_id_sub = union_id)     │
   │                                       │
   └─ 返回成功                              │
```

**参数映射**：

| Hatchery 参数 | OneID 参数 | 说明 |
|---|---|---|
| `username` | `username` + `name` | name 用 username 填充 |
| `password` | `password` | 明文传入 |
| `email` | `email` | 可选 |
| — | `department_ids` | 统一模式暂不传部门 |

**角色处理**：
- 只有 `role=admin` 时才调 `add-role-users`（role_id 固定 `1400000`）
- 普通 user 不需要绑定 OneID 角色

---

### 2. 批量创建用户 (`POST /admin/batch-create`)

与单个创建相同逻辑，循环调用。单个失败不影响其他。

---

### 3. 删除用户 - 软删除 (`DELETE /admin/delete`)

**双写顺序**：先调 OneID 停用 → 成功后本地软删除

```
Hatchery 直调 OneID：
POST {API_BASE}/openapi/v1/contacts/users/batch_disable
Authorization: Bearer {自建应用 token}
Body: { "union_ids": ["{user.one_id_sub}"] }
```

---

### 4. 硬删除用户 (`DELETE /admin/hard-delete`)

**双写顺序**：先调 OneID 删除 → 成功后本地硬删除

```
Hatchery 直调 OneID：
DELETE {API_BASE}/openapi/v1/contacts/users/{union_id}
Authorization: Bearer {自建应用 token}
Body: {
  "legacies": [{
    "resolve_method": "reserve",
    "transfer_to_entity_type": "user",
    "app_id": "{ONEID_APP_ID}"
  }]
}
```

---

### 5. 恢复用户 (`POST /admin/restore`)

**双写顺序**：先调 OneID 启用 → 成功后本地恢复

```
Hatchery 直调 OneID：
POST {API_BASE}/openapi/v1/contacts/users/{union_id}/enable
Authorization: Bearer {自建应用 token}
```

---

### 6. 重置密码 (`POST /admin/reset-password`)

**双写顺序**：先调 Gateway 重置 → 成功后本地重置

```
通过 Gateway（Internal 接口需套件 Token）：
POST {GATEWAY_URL}/api/reset-password
X-Internal-Token: {HMAC 签名}
Body: {
  "account_id": "{ONEID_ACCOUNT_ID}",
  "union_id": "{user.one_id_sub}",
  "password": "{new_password}"
}
→ Gateway 用套件 tenant token 调：
  POST /internal/clawpro/contacts/users/{union_id}/password/reset
```

---

### 7. 更新用户 (`PUT /admin/update-user`)

**双写顺序**：先调 OneID 更新 → 成功后本地更新

```
Hatchery 直调 OneID：
PUT {API_BASE}/openapi/v1/contacts/users/{union_id}
Authorization: Bearer {自建应用 token}
Body: { "name": "...", "email": "..." }
```

**同步到 OneID 的字段**：

| 本地字段 | OneID 字段 |
|---|---|
| `username` | `name` |
| `email` | `email` |

**仅本地操作（不同步到 OneID）**：
- `role`（角色变更走 Gateway add-role-users）
- `instance_quota`、`token_quota_day`、`group_ids`

---

### 8. 角色添加用户

**触发时机**：创建用户时 role=admin，或更新用户时 user→admin

```
通过 Gateway（Internal 接口需套件 Token）：
POST {GATEWAY_URL}/api/add-role-users
X-Internal-Token: {HMAC 签名}
Body: {
  "account_id": "{ONEID_ACCOUNT_ID}",
  "role_id": "1400000",
  "union_ids": ["{union_id}"]
}
→ Gateway 用套件 tenant token 调：
  POST /internal/openapi/v2/permissions/roles/1400000/users
```

---

### 9. 角色移除用户

**触发时机**：更新用户时 admin→user

```
通过 Gateway（Internal 接口需套件 Token）：
POST {GATEWAY_URL}/api/remove-role-users
X-Internal-Token: {HMAC 签名}
Body: {
  "account_id": "{ONEID_ACCOUNT_ID}",
  "role_id": "1400000",
  "union_ids": ["{union_id}"]
}
→ Gateway 用套件 tenant token 调：
  DELETE /internal/openapi/v2/permissions/roles/1400000/users
```

---

### 10-13. 用户侧登录代理（透明转发）

统一账号模式下，用户登录/改密码相关接口透明代理到 OneID base API（`ONEID_API_BASE_URL`），无需任何 Token。

**架构**：

```
前端 ──── POST /v1/authn/enterprise ──────► Hatchery ──────► OneID base API
     ◄── Set-Cookie (domain 转化) + session ◄──────────────◄────────────────
```

**Cookie 处理**：
- **响应方向**：OneID 返回的 `Set-Cookie` 中 `Domain` 属性替换为租户配置的 `domain` 字段
- **请求方向**：仅转发 `state_token` 和 `session_token` cookie，过滤掉 `hatchery-session` 等无关 cookie
- **登录接口**：不转发浏览器旧 cookie 给 OneID，强制每次创建新 session

**各接口特殊逻辑**：

| 路由 | Handler | 特殊行为 |
|------|---------|----------|
| `GET /v1/authn/encrypt_setting` | `HandleOneIDAuthnProxy` | 纯透传 |
| `POST /v1/authn/enterprise` | `HandleOneIDAuthnLogin` | 成功后建 hatchery session + 响应追加 `ok/redirect/role` |
| `POST /v1/authn/enterprise/password:reset` | `HandleOneIDPasswordReset` | 成功后清 hatchery session + 清 `state_token` cookie |
| `POST /user/v1/user/enterprise/password:verify` | `HandleOneIDAuthnProxy` | 纯透传（需 session_token） |

**登录成功判断**（`POST /v1/authn/enterprise`）：
- HTTP 200 且 response 中 `next` 不存在，或 `next.type` 不是 `CAPTCHA_OPTIONS` / `ACCOUNT_RESET_PASSWORD`
- 成功时从 query 参数 `?username=xxx` 获取用户名，查本地 DB 建立 session
- 响应在 OneID 原始 JSON 基础上追加：`{"ok": true, "redirect": "/", "role": "admin|user"}`

**前端调用示例**：

```bash
# 获取加密公钥
curl -X GET "https://your-domain/v1/authn/encrypt_setting"

# 登录（username 通过 query 传入）
curl -X POST "https://your-domain/v1/authn/enterprise?username=zhangsan" \
  -H "Content-Type: application/json" \
  -d '{"credential":"...encrypted...", "accountId":"1448840952541086165", "captchaVerification":"..."}'

# 首次登录强制改密码（使用登录返回的 state_token）
curl -X POST "https://your-domain/v1/authn/enterprise/password:reset" \
  -H "Content-Type: application/json" \
  -b "state_token=xxx" \
  -d '{"credential":"...encrypted new password..."}'

# 密码验证获取 state_token（已登录状态）
curl -X POST "https://your-domain/user/v1/user/enterprise/password:verify" \
  -H "Content-Type: application/json" \
  -b "session_token=xxx" \
  -d '{"credential":"...encrypted current password..."}'
```

---

### 14-20. 本地操作（不涉及 OneID）

以下接口不涉及 OneID 双写，仅操作本地数据：
- 导出 Token、禁用/启用 Token、用户 Token 管理
- 用户限额、用户 VPC、部门列表

---

## 认证方式

统一账号模式涉及两种不同的认证机制，分别用于不同的调用链路：

### 一、调用 OneID OpenAPI — 自建应用 Token

**适用接口**：创建/删除/更新/停用/启用用户（Hatchery 直调 OneID）

**前置动作**：每次调用 OneID OpenAPI 前，必须先获取自建应用的 access_token。Token 有有效期，需缓存复用。

```
┌─────────────────────────────────────────────────────────────────┐
│  Step 1: 获取 Token（首次或缓存过期时）                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  POST {ONEID_TOKEN_ENDPOINT}                                    │
│  Content-Type: application/json                                 │
│                                                                 │
│  {                                                              │
│    "client_id": "{ONEID_CLIENT_ID}",                            │
│    "client_secret": "{ONEID_CLIENT_SECRET}",                    │
│    "grant_type": "client_credentials"                           │
│  }                                                              │
│                                                                 │
│  Response:                                                      │
│  {                                                              │
│    "access_token": "xxx",    ← 不带 t- 前缀                     │
│    "token_type": "Bearer",                                      │
│    "expires_in": 7200        ← 有效期(秒)                        │
│  }                                                              │
│                                                                 │
│  缓存策略：内存缓存，提前 60s 刷新                                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  Step 2: 调用 OneID OpenAPI                                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  POST/PUT/DELETE {ONEID_API_BASE_URL}/openapi/v1/contacts/users/...│
│  Authorization: Bearer {access_token}                           │
│  Content-Type: application/json                                 │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 二、调用 Gateway Internal API — HMAC 签名

**适用接口**：重置密码、角色添加用户（通过 Gateway 代理 OneID Internal 接口）

**前置动作**：无需获取 token。使用 HMAC-SHA256 对当前时间戳签名，生成 `X-Internal-Token` 请求头。

```
┌─────────────────────────────────────────────────────────────────┐
│  签名生成（每次请求实时计算）                                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  timestamp = 当前 Unix 时间戳（秒）                              │
│  signature = HMAC-SHA256(timestamp, INTERNAL_SECRET) → hex      │
│  token     = "{timestamp}.{signature}"                          │
│                                                                 │
│  示例：1716710400.a3f8b2c1d4e5...                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│  调用 Gateway                                                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  POST {GATEWAY_URL}/api/reset-password                          │
│  X-Internal-Token: {timestamp}.{signature}                      │
│  Content-Type: application/json                                 │
│                                                                 │
│  Body: { "account_id": "...", "union_id": "...", ... }          │
│                                                                 │
│  Gateway 收到后：                                                │
│    1. 验证 HMAC 签名                                            │
│    2. 用套件私钥(JWK)获取 tenant token (t- 前缀)                │
│    3. 用 tenant token 调用 OneID Internal API                   │
│    4. 返回结果给 Hatchery                                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 对比

| | 调 OneID OpenAPI | 调 Gateway |
|---|---|---|
| **认证方式** | Bearer Token（自建应用 access_token） | HMAC 签名（X-Internal-Token） |
| **前置动作** | 先调 token endpoint 获取 access_token | 实时计算 HMAC，无网络开销 |
| **缓存** | access_token 缓存至过期前 60s | 无需缓存（每次计算） |
| **密钥来源** | `ONEID_CLIENT_ID` + `ONEID_CLIENT_SECRET` | `INTERNAL_SECRET` |
| **适用场景** | OpenAPI 用户 CRUD | Internal 接口（角色/密码） |

---

## 配置参数

| 环境变量 | Hatchery site_configs | Gateway oneid_tenants | 说明 | 默认值 |
|---|---|---|---|---|
| `ONEID_ACCOUNT_ID` | `one_id_account_id` | `account_id` | 租户 ID | — |
| `ONEID_APP_ID` | `one_id_app_id` | `app_id` | 自建应用 ID（统一模式判断条件） | — |
| `ONEID_CLIENT_ID` | `one_id_client_id` | `client_id` | 自建应用 client_id | — |
| `ONEID_CLIENT_SECRET` | `one_id_client_secret` | `client_secret` | 自建应用 client_secret | — |
| `ONEID_TOKEN_ENDPOINT` | `one_id_token_endpoint` | `token_endpoint` | Token 获取端点 | — |
| `ONEID_API_BASE_URL` | — (代码默认+环境变量覆盖) | — | OpenAPI 域名 | `https://api.account.tencent.com` |

---

## 向前兼容

| 模式 | 判断条件 | 用户创建 | 用户管理 |
|---|---|---|---|
| 独立账号 | `OneIDAccountID == ""` | 本地创建 | 本地操作 |
| 现有 OneID | `OneIDAccountID != ""` && `OneIDAppID == ""` | ❌ 禁止（OneID 同步落库） | ❌ 禁止创建/批量创建 |
| **统一账号** | `OneIDAppID != ""` | 双写（OneID + 本地） | 双写（OneID + 本地） |

---

## 实施步骤

### Phase 1: Hatchery 参数注入
- site_configs 加 5 个字段（app_id, client_id, client_secret, token_endpoint, api_base_url）
- tenant_init 环境变量回填

### Phase 2: Hatchery OneID Client
- 自建应用 Token 获取 + 内存缓存
- OpenAPI 调用封装（创建/删除/更新/停用/启用用户）

### Phase 3: Hatchery 用户管理双写
- 7 个 handler 改造（统一模式下先调 OneID → 成功后写本地）
- 创建用户后调 Gateway add-role-users（admin 角色时）

---

*文档版本：v1.3 | 更新日期：2026-06-01*
*更新历史：*
- *v1.3：新增用户侧登录代理接口（登录/加密配置/自助改密码/密码验证）、角色移除接口、更正 OneID API 版本为 v3*
- *v1.2：修正调用方式 — OpenAPI 接口 Hatchery 直调，仅 Internal 接口走 Gateway（因套件私钥仅存 Gateway）*
- *v1.1：补充创建用户后角色绑定流程*
- *v1.0：初版*
