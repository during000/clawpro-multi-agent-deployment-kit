# 03. Implement — username 精确查询与显式模糊（v4）

## 实现结果

### Controller

- `adminAuditFilters` 新增 `Fuzzy bool`。
- `parseAdminAuditFilters` 与 `/admin/users` 保持一致：仅 `fuzzy=1` 为 true。
- username 非空且未传 fuzzy=1 时使用参数化 `username = ?`。
- username 非空且 fuzzy=1 时使用原参数化 `username LIKE ?`，值为 `%keyword%`。
- 只传 fuzzy 而无 username 时不增加查询条件。
- user_id、username、action、resource_id、时间范围继续使用 AND 语义，Count 和列表共享同一过滤器。

### 数据库

最终相较 master 新增三个租户联合索引：

```text
idx_audit_logs_identifier_user_id(identifier,user_id)
idx_audit_logs_identifier_username(identifier,username)
idx_audit_logs_identifier_resource_id(identifier,resource_id)
```

已同步：

- `model/audit.go`：SQLite AutoMigrate 标签。
- `model/audit_test.go`：三个联合索引存在性测试。
- `sql/init.sql`：全新 MySQL schema。
- `sql/0804-add-audit-query-indexes.sql`：存量 MySQL 幂等新增。

迁移不包含 DROP INDEX，不修改字段、表或数据。

### 测试代码

- Controller UT 覆盖 username 默认精确、fuzzy=1 显式模糊、fuzzy 非 1、fuzzy 无 username，以及 user_id 与两种 username 模式组合。
- Python IT helper 新增 fuzzy 参数，测试完整 username 精确命中、部分 username 默认不命中、fuzzy=1 部分命中。
- 原 user_id、非法参数、查询错误和同步总数能力保持不变。

## 改动文件

| 文件 | 变更 |
|------|------|
| `controller/admin_audit.go` | username 默认等值，fuzzy=1 时 LIKE |
| `controller/admin_audit_test.go` | v4 查询语义用例 |
| `model/audit.go` | username 租户联合索引 |
| `model/audit_test.go` | 三联合索引断言 |
| `sql/init.sql` | 新建库 username 联合索引 |
| `sql/0804-add-audit-query-indexes.sql` | 存量库幂等新增 username 联合索引 |
| `test/scripts/agent_bridge/test_agent_bridge_audit.py` | 精确/fuzzy=1 IT 覆盖 |
| `.specs/plans/...` | v4 决策和执行记录 |

`docs/API.md` 尚未在 Implement 阶段修改，留在获得授权后的 Docs 阶段统一更新并生成 OpenAPI。

## 实现期验证

```text
gofmt
PASS

Python AST parse
PASS

git diff --check
PASS

go test ./controller -run '^(TestHandleAdminAudit|TestAdminAuditHandlers)' -count=1 -race
PASS (2.914s)

go test ./model -run '^(TestLogAudit|TestAuditLog)' -count=1 -race
PASS (1.402s)
```

## 与 Plan 差异

无。

## 待后续阶段验证

- UT：全量非 race、全量 race、覆盖率、vet 和双模式 build。
- Docs：API 参数表、精确/模糊示例、OpenAPI 新参数。
- IT：MySQL 新建/增量/重复迁移、EXPLAIN 精确与模糊、真实 HTTP 和 Python 增量覆盖。
- Review：公共 API 语义变化已由用户接受，但仍需作为显式兼容边界审查。
