# 02. Plan — username 精确查询与显式模糊（v4）

## 结论

`GET /admin/audit` 的 username 参数改为默认精确匹配，并新增与 `/admin/users` 一致的 `fuzzy=1` 显式模糊开关：

```text
username=alice           → identifier=? AND username=?
username=ali&fuzzy=1     → identifier=? AND username LIKE '%ali%'
```

这样，使用完整 username 的 API 调用方无需修改调用结构，即可从默认的租户范围扫描切换为索引等值查询。确实需要部分字符串搜索的调用方必须显式增加 `fuzzy=1`。

前端操作记录页面仍执行「`/admin/users?fuzzy=1` 获取候选 → `/admin/audit?user_id=<ID>` 精确查询」，不依赖 audit username 模糊查询。

## 目标与边界

### 目标

1. `username` 默认使用 `=`，降低现有完整用户名 API 查询压力。
2. 新增可选参数 `fuzzy`；仅值为 `1` 且 username 非空时使用原 `LIKE '%keyword%'`。
3. 新增 `(identifier,username)` 联合索引，使默认 username Count 和列表查询可按租户、用户名定位。
4. 保留现有 `(identifier,user_id)`、`(identifier,resource_id)` 和同步 total/pagination 契约。
5. Count 和列表继续共享完全一致的筛选条件。

### 明确接受的兼容边界

- 使用完整 username 的 API 调用方不需要修改参数或请求路径。
- 此前依赖部分 username 且未传新参数的调用方，结果将从包含匹配变为精确匹配；用户已明确接受该行为变化。
- 模糊能力没有删除，调用方可显式传 `fuzzy=1` 恢复原行为。
- `fuzzy` 未传、为空或不是 `1` 时均按精确查询，和已有 `/admin/users` 约定一致。
- 只传 `fuzzy=1` 而未传 username 时不增加任何 username 条件。

## API 契约

### `GET /admin/audit`

新增参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| fuzzy | string | 否 | 传 `1` 时对 username 启用包含式模糊匹配；其他值或不传均为精确匹配 |

既有 username 参数改为：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| username | string | 否 | 默认按完整用户名精确匹配；需要部分字符串查询时同时传 `fuzzy=1` |

组合规则：

- `user_id`、username、action、resource_id、start_time、end_time 使用 AND 语义。
- fuzzy 只控制 username 的比较方式，不影响其他筛选。
- 响应继续包含 `logs`、`page`、`page_size`、`total`、`total_pages`。
- 不新增 count 路由、快速列表、缓存或其他响应模式。

## 数据库变更

相较 master 最终只新增三个联合索引，不修改字段、表或数据：

| 索引 | 操作 | 目的 |
|------|------|------|
| `idx_audit_logs_identifier_user_id(identifier,user_id)` | 新增 | 前端选定用户后的等值查询 |
| `idx_audit_logs_identifier_username(identifier,username)` | 新增 | API 完整 username 默认精确查询 |
| `idx_audit_logs_identifier_resource_id(identifier,resource_id)` | 新增 | resource_id 等值查询 |

同步维护：

- `model/audit.go`：SQLite AutoMigrate 索引标签。
- `model/audit_test.go`：三个联合索引存在性断言。
- `sql/init.sql`：全新 MySQL `CREATE TABLE audit_logs`。
- `sql/0804-add-audit-query-indexes.sql`：存量 MySQL 幂等新增三个索引，不删除任何既有索引。

## 改动文件

| 文件 | 操作 | 内容 |
|------|------|------|
| `controller/admin_audit.go` | 修改 | 解析 fuzzy；username 默认等值，fuzzy=1 时 LIKE |
| `controller/admin_audit_test.go` | 修改 | 精确默认、显式模糊、组合筛选回归 |
| `model/audit.go` / `model/audit_test.go` | 修改 | username 联合索引及三索引断言 |
| `sql/init.sql` | 修改 | 增加 username 联合索引 |
| `sql/0804-add-audit-query-indexes.sql` | 修改 | 幂等新增 username 联合索引 |
| `docs/API.md` | 修改 | fuzzy 参数、username 默认精确语义和调用示例 |
| `test/scripts/agent_bridge/test_agent_bridge_audit.py` | 修改 | 覆盖 username 精确与 fuzzy=1 |
| `.specs/plans/...` | 修改 | 记录 v4 决策、验证和交付状态 |

## 测试设计

### UT

| # | 场景 | 输入 | 预期 | 级别 |
|---|------|------|------|------|
| 1 | username 默认精确 | `username=alice` | 只返回 username 完全为 alice 的记录 | P0 |
| 2 | 显式模糊 | `username=ali&fuzzy=1` | 命中 alice、malice、alice-old | P0 |
| 3 | fuzzy 非 1 | `username=ali&fuzzy=true` | 按精确 ali 查询 | P1 |
| 4 | fuzzy 无 username | `fuzzy=1` | 不添加 username 条件 | P1 |
| 5 | user_id 及其他筛选 | user_id + username/fuzzy/action/time | 按 AND 语义且 Count/list 一致 | P0 |
| 6 | SQLite 索引 | AutoMigrate | 三个租户联合索引均存在 | P0 |
| 7 | 原有错误边界 | 非法 user_id、Count/Find 错误 | 保持 400/500 | P0 |

### IT

| # | 场景 | 预期 | 级别 |
|---|------|------|------|
| 1 | 完整 username API 调用 | 无需改调用结构，精确命中 | P0 |
| 2 | fuzzy=1 调用 | 保留包含式模糊能力 | P0 |
| 3 | user_id 前端链路 | 候选 ID 精确查询不回退 username LIKE | P0 |
| 4 | MySQL 新建与增量迁移 | 三个联合索引均创建，重复迁移安全 | P0 |
| 5 | EXPLAIN username 精确 | 使用 `(identifier,username)` 并显著缩小 rows | P0 |
| 6 | EXPLAIN username fuzzy | 记录仍扫描租户范围的显式性能边界 | P0 |
| 7 | OpenAPI 增量覆盖 | fuzzy 新参数被 Python IT 覆盖 | P0 |

## 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| 隐式依赖旧模糊语义的调用方 | 不再部分匹配 | 发布说明明确增加 `fuzzy=1`；用户已接受语义变化 |
| fuzzy=1 仍扫描租户范围 | 高数据量下仍有压力 | 显式低频能力；前端主路径继续使用 user_id |
| 新 username 索引增加写放大 | 审计写入多维护一个索引 | 该索引直接服务现有 API 精确查询，收益明确 |
| 首页无筛选 count | 仍随租户记录数增长 | 维持已接受边界，预发布观察延迟与 rows examined |

## 回滚

1. 逻辑回滚可恢复 username LIKE 默认行为并移除 fuzzy 分支。
2. `(identifier,username)` 不修改数据，可在确认无精确调用后低峰删除。
3. user_id/resource_id 索引及接口能力彼此独立，不需要随 username 调整回滚。

## 实施顺序

1. 更新筛选模型和 username 分支。
2. 更新三个 schema 维护点及索引测试。
3. 更新 controller/Python 测试和 API 文档。
4. 执行 UT、Docs/OpenAPI、MySQL/HTTP IT 与 Review。
5. 获得 Commit 授权后 amend 单提交并以 `--force-with-lease` 更新远端。
