# 05. Docs — 文档更新（username 精确查询 v4）

## 更新清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `docs/API.md` | 参数契约 | `username` 改为默认等值查询；新增可选字符串参数 `fuzzy`，仅值为 `1` 时执行包含匹配 |
| `docs/API.md` | 调用方迁移 | 完整 username 调用形式不变；依赖部分 username 匹配的调用方需显式增加 `fuzzy=1` |
| `docs/API.md` | 前端契约 | 保持 `/admin/users?fuzzy=1` 候选选择、选中后按 `user_id` 查询/导出的主流程 |
| `docs/API.md` | 性能边界 | 记录 username 等值联合索引及显式前导 `%` 模糊查询仍可能扫描租户范围 |
| `/tmp/audit-v4-openapi.json` | 临时生成验证 | 成功生成 416 paths、425 operations；仅用于核对，验证后移入废纸篓，不是仓库产物 |

## 最终 API 契约

- 只保留 `GET /admin/audit`，不提供 `/admin/audit/count`。
- 请求参数共 9 个：`page`、`page_size`、`user_id`、`username`、`fuzzy`、`action`、`resource_id`、`start_time`、`end_time`。
- `username=<完整用户名>` 默认使用等值查询；既有精确调用方的参数形式不变。
- 仅 `username=<keyword>&fuzzy=1` 使用 `LIKE '%keyword%'`；`fuzzy` 不为 `1` 时保持精确，单独传 `fuzzy=1` 不增加筛选条件。
- 此前依赖部分 username 匹配的调用方需要显式增加 `fuzzy=1`。
- `user_id` 为无符号整数等值筛选；显式 0 合法，负数、文本和溢出返回 400。
- 所有筛选同时提供时使用 AND 语义；响应继续同步返回 `logs`、`page`、`page_size`、`total`、`total_pages`。

## 前端交互契约

1. 首次进入调用 `/admin/audit`，使用同一响应中的列表和总数。
2. 用户输入操作人名称时，通过 `/admin/users?username=<输入>&fuzzy=1&page=1&page_size=20` 获取候选，并建议约 300 ms 防抖。
3. 候选对象实际字段为 `ID` / `Username`；展示名称但保存 ID。
4. 选定后调用 `/admin/audit?user_id=<ID>`，覆盖该用户改名前后的全部审计记录。
5. 导出复用同一 `user_id`；清空候选时移除该参数。
6. audit 的 `fuzzy=1` 只作为显式兼容能力，不驱动前端页面主查询。

## 性能边界

- 首页无筛选的精确 Count 使用当前租户 `identifier` 条件，仍随租户审计记录数增长。
- 选定用户后使用 `(identifier, user_id)` 联合索引；完整 username 使用 `(identifier, username)`；资源筛选使用 `(identifier, resource_id)`。
- `username=<keyword>&fuzzy=1` 带前导 `%`，仍可能扫描当前租户的大量索引项；是否允许该代价由调用方显式表达。

## OpenAPI 验证

```text
python3 test/api_md_to_openapi.py --output /tmp/audit-v4-openapi.json
Paths: 416
Operations: 425

/admin/audit GET 参数：
page integer
page_size integer
user_id integer
username string
fuzzy string
action string
resource_id string
start_time integer
end_time integer
```

断言结果：

- `/admin/audit` 参数名称和顺序与上述 9 项完全一致；
- `fuzzy` 恰好出现一次且 OpenAPI 类型为 `string`；
- 不存在 `/admin/audit/count` path；
- `docs/API.md` 不包含 `/admin/audit/count`、`include_total`、`counted_at` 或 `username_exact`；
- `git diff --check` 通过。

## 未在本文档阶段声称的事项

- OpenAPI 生成成功不等于真实 HTTP 集成测试通过；真实接口、MySQL 索引/迁移和 Python 场景验证留在 IT。
- 当前仓库不包含操作记录前端页面，因此文档只交付接入契约，不能替代前端 MR/联调证据。
