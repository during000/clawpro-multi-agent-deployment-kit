# 05. Docs — 文档更新

> 评估并执行增量文档更新。本次为后端内部行为修正，API 契约（端点/参数/状态码）未变，但兼容语义需补充说明。

---

## 更新清单

### `docs/API.md`

**1. `POST /admin/group-config/policy`（line 11407 附近）**

补充写 `*_rules` 时事务内清除配对 `*_day` 绑定的双向对称语义。原文仅说明写 `*_day`→生成 rules，未说明写 `*_rules`→清 day。

改动后：
> 旧字段 `token_quota_day` / `global_token_quota_day` 仅用于兼容；写入 `*_day` 时会同步生成同组对应的 rules 配置并清除既有 `*_day` 绑定，写入 `*_rules` 时也会事务内清除同组的 `*_day` 绑定（双向对称，避免两个绑定共存导致删除后旧字段复活）。…

**2. `POST /admin/group-config/policy/delete`（line 11413-11417）**

新增「配额 key 兼容」说明 + 细化 422 错误条件。

改动后：
- 新增：删除配额 key 时若目标 key 绑定不存在，fallback 删配对 key 绑定。
- 422 条件：目标 key 与配对 key 均未配置。

---

## 未更新

| 文档 | 原因 |
|------|------|
| `docs/i18n.md` | 未新增 i18n Key（复用既有 `MsgPolicyNotConfigured` / `MsgDatabaseOperationFailed` / `MsgOperationFailed`） |
| `docs/testing.md` | 测试方法与既有模式一致，无新规范 |
| `sql/init.sql` / 增量 migration | 未改 schema（仅改应用层行为，`*_day` / `*_rules` 列均存在） |
| OpenAPI spec | API 端点/参数/响应结构未变，spec 无需重新生成 |

---

## 红线检查

- ✅ 红线 #10（修改 API 但未更新 `docs/API.md`）：已更新
- ✅ 未破坏 API 兼容性（红线 #4）：端点/参数/状态码不变，仅补充兼容语义
