# 05. Docs — 文档更新清单

> 增量更新 `docs/API.md`。

---

## 更新清单

### `docs/API.md`

#### 1. `POST /openclaw/models/connectivity`（约 3164 行）

在「请求说明」末尾新增探活方式说明：

> **探活方式：** 使用 chat 探活（发送极简对话请求，max_tokens=1），同时验证 API 地址可达性、API Key 有效性及模型 ID 正确性。

#### 2. `POST /admin/models/connectivity`（约 6061 行）

在「请求说明」末尾新增探活方式说明：

> **探活方式：** 使用 chat 探活（发送极简对话请求，max_tokens=1），同时验证 API 地址可达性、API Key 有效性及模型 ID 正确性。

---

## 无其他文档变更

- 无数据库 schema 变更，无需更新 `sql/init.sql`
- 无新增 API 接口，无需新增接口文档
- 无新增 i18n 文案，无需更新 `i18n/`
