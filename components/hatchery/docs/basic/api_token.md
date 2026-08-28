# API Token 管理 — 前端接入文档

## 概述

API Token 管理分为**用户端**和**管理端**两部分：

- **用户端**：用户自主管理个人 API Token（创建/查看/重置/销毁），通过头像下拉菜单弹窗交互
- **管理端**：管理员禁用/启用用户 Token，通过用户管理页「···」菜单操作

所有接口始终返回 JSON，认证方式为 Cookie Session（网页登录）。

---

## 一、用户端接口

### 1. 查看 Token 状态

- **请求：** `GET /api-token`
- **认证：** Cookie Session

- **响应：**

**未创建：**

```json
{
  "exists": false
}
```

**已创建（启用中）：**

```json
{
  "exists": true,
  "mask": "hk-6992****2dd7",
  "disabled": false,
  "created_at": "2026-03-28T10:00:00+08:00",
  "last_used_at": "2026-03-28T12:30:00+08:00"
}
```

**已创建（被管理员禁用）：**

```json
{
  "exists": true,
  "mask": "hk-6992****2dd7",
  "disabled": true,
  "created_at": "2026-03-28T10:00:00+08:00",
  "last_used_at": "2026-03-28T12:30:00+08:00"
}
```

- **字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| exists | bool | 是否已创建 Token |
| mask | string | 掩码展示，格式 `hk-xxxx****xxxx` |
| disabled | bool | 是否被管理员禁用 |
| created_at | string \| null | Token 创建/重置时间（ISO 8601） |
| last_used_at | string \| null | 最近一次 API 调用时间，未调用过为 `null` |

- **前端状态映射：**

| exists | disabled | 弹窗状态 |
|--------|----------|----------|
| false | — | 状态一：未创建 |
| true | false | 状态三：已创建（启用中） |
| true | true | 状态四：已创建（被管理员禁用） |

> 状态二（Token 已生成）仅在创建/重置成功后由前端本地展示，不对应后端状态。

---

### 2. 创建 Token

- **请求：** `POST /api-token/create`
- **认证：** Cookie Session

- **成功响应（200）：**

```json
{
  "token": "hk-6992a7b8ec41ac9bb5f5b2a591f8cd1856bf6a78750c2dd7",
  "mask": "hk-6992****2dd7"
}
```

> ⚠️ `token` 明文仅此一次返回，前端需立即展示给用户，关闭弹窗后不可再获取。

- **错误响应：**

| HTTP 状态码 | 场景 | error 内容 |
|-------------|------|------------|
| 409 | 已有 Token | `Token 已存在，如需更换请使用重置功能` |
| 403 | Token 被管理员禁用 | `您的 Token 已被管理员禁用，如需恢复请联系企业管理员` |

---

### 3. 重置 Token

- **请求：** `POST /api-token/reset`
- **认证：** Cookie Session

- **成功响应（200）：**

```json
{
  "token": "hk-b5a28f412686fff2be6832778c417ec3ca53bc3f1d28610a",
  "mask": "hk-b5a2****610a"
}
```

> 旧 Token 立即失效。返回格式与创建相同，前端展示新 Token 明文。

- **错误响应：**

| HTTP 状态码 | 场景 | error 内容 |
|-------------|------|------------|
| 403 | Token 被管理员禁用 | `您的 Token 已被管理员禁用，如需恢复请联系企业管理员` |

---

### 4. 销毁 Token

- **请求：** `POST /api-token/revoke`
- **认证：** Cookie Session

- **成功响应（200）：**

```json
{
  "ok": true
}
```

- **错误响应：**

| HTTP 状态码 | 场景 | error 内容 |
|-------------|------|------------|
| 403 | Token 被管理员禁用 | `您的 Token 已被管理员禁用，如需恢复请联系企业管理员` |

---

## 二、管理端接口

### 5. 禁用用户 Token

- **请求：** `POST /admin/token/disable?id={userID}`
- **认证：** Cookie Session（管理员）或 AdminToken

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 目标用户 ID（query 参数） |

- **成功响应（200）：**

```json
{
  "ok": true
}
```

- **错误响应：**

| HTTP 状态码 | 场景 | error 内容 |
|-------------|------|------------|
| 404 | 用户不存在 | `用户不存在` |
| 400 | 用户无 Token | `该用户没有 API Token` |
| 400 | 已是禁用状态 | `该用户 Token 已处于禁用状态` |

---

### 6. 启用用户 Token

- **请求：** `POST /admin/token/enable?id={userID}`
- **认证：** Cookie Session（管理员）或 AdminToken

- **参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 目标用户 ID（query 参数） |

- **成功响应（200）：**

```json
{
  "ok": true
}
```

- **错误响应：**

| HTTP 状态码 | 场景 | error 内容 |
|-------------|------|------------|
| 404 | 用户不存在 | `用户不存在` |
| 400 | 用户无 Token | `该用户没有 API Token` |
| 400 | Token 未被禁用 | `该用户 Token 未被禁用` |

---

## 三、前端交互要点

### 用户端弹窗状态流转

```
                        GET /api-token
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
         exists=false   exists=true    exists=true
                        disabled=false disabled=true
              │              │              │
              ▼              ▼              ▼
          【未创建】     【启用中】     【被禁用】
              │              │              │
      点击「创建」     点击「重置」     只读，提示
              │              │          联系管理员
              ▼              ▼
      POST /api-token  POST /api-token
          /create          /reset
              │              │
              ▼              ▼
        【Token 已生成】(前端本地状态)
         展示明文 + 复制按钮
         警告"仅显示一次"
              │
      点击「我已保存，关闭」
              │
              ▼
         GET /api-token → 刷新为【启用中】
```

### 管理端菜单动态规则

管理端需要先获取用户的 Token 状态来决定菜单项。可以在用户列表接口 `GET /admin/users` 的响应中读取每个用户的 `APIToken` 和 `APITokenDisabled` 字段：

| 用户字段 | 菜单显示 |
|----------|----------|
| `APIToken` 为空 | 不显示 Token 操作项 |
| `APIToken` 非空，`APITokenDisabled=false` | 显示「禁用 Token」（红色） |
| `APIToken` 非空，`APITokenDisabled=true` | 显示「启用 Token」（正常色） |

> 注意：`APIToken` 和 `APITokenDisabled` 字段在 User 模型中标记了 `json:"-"`，不会出现在 JSON 响应中。如果管理端需要这些字段，后端需要额外在用户列表接口中补充返回。**当前实现中用户列表不包含这些字段，前端如需判断可单独调用状态接口或后端配合调整。**

### 关键交互细节

1. **Token 明文仅展示一次**：创建和重置接口返回的 `token` 字段是唯一一次获取明文的机会，`GET /api-token` 不返回明文
2. **被禁用时不可操作**：创建/重置/销毁在 `disabled=true` 时均返回 403
3. **所有接口默认返回 JSON**：这些接口内部调用了 `jsonAPI()`，无需额外设置 `Accept` Header（设置也不影响）
4. **掩码格式**：`hk-` + 前4位 + `****` + 后4位，如 `hk-6992****2dd7`
5. **时间格式**：ISO 8601 带时区（服务器本地时区），如 `2026-03-28T10:00:00+08:00`
6. **`last_used_at` 为 null**：表示 Token 创建后从未被用于 API 调用

---

## 四、已下线接口

| 接口 | 说明 |
|------|------|
| `POST /admin/export-tokens` | 已下线，返回 404 |
