# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

`/api/admin/skills/instances` 接口在按多分组筛选时返回重复实例。用户请求：

```
GET /api/admin/skills/instances?slug=bv_warehouse_query_skill&page=1&page_size=20
  &status=uninstalled,failed,upgrade_failed,outdated,uninstall_failed_old,installed
  &group_id=112,119,81
```

当同一用户属于 112、119、81 中多个分组时，该用户的所有实例在结果中重复出现。

## 根因分析

问题位于 `controller/admin_skills.go` `HandleAdminSkillInstances` 第 1873-1877 行（`group_id` 多分组筛选分支）：

```go
} else if len(groupIDs) > 0 {
    // 仅指定分组
    baseQuery = baseQuery.Joins(
        "JOIN user_group_members ugm ON ugm.user_id = instances.user_id AND ugm.user_group_id IN ?", groupIDs)
}
```

**原因**：`JOIN user_group_members` 在用户属于多个分组时，`instances` 的每一行会匹配到多条 `user_group_members` 记录，导致结果集中的实例重复（每个匹配的分组一行）。

**对比**：同函数中 `includeUngrouped && len(groupIDs) > 0` 分支（第 1864-1868 行）使用了 `WHERE ... IN (子查询)` 方式，不存在此问题。

## 目标

修复多 `group_id` 筛选时实例重复的问题，确保每个实例在结果中仅出现一次。

- [x] 修改 `controller/admin_skills.go` 中 `group_id` 多分组筛选逻辑，将 `JOIN` 改为子查询
- [x] 确保与 `includeUngrouped + groupIDs` 分支行为一致
- [x] 不影响已有单分组筛选行为

## 范围

| 包含 | 不包含 |
|------|--------|
| 修复 `group_id` 多分组筛选的 JOIN 导致的重复问题 | 修改 `includeUngrouped + groupIDs` 分支（该分支已正确） |
| `controller/admin_skills.go` 第 1873-1877 行 | 修改 `BuildSkillInstanceQuery` 基础查询 |
| 对应的单元测试补充 | 修改其他筛选逻辑 |

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 修复方式：JOIN 改为 WHERE IN 子查询 | 已确认 | 改为 `baseQuery.Where("instances.user_id IN (?)", subQuery)`，与同函数 `includeUngrouped` 分支对齐 |
| 2 | 是否需要修改 `admin_skills_instances_test.go` | 已确认 | 需要补充多分组重复场景的测试用例 |

## 约束与依赖

- 改动仅涉及 1 行代码（JOIN → WHERE IN 子查询）
- 不涉及数据库 schema 变更
- 不涉及 API 接口变更
- 不影响多租户隔离（user_group_members 通过 user_id 外键关联已隔离的 instances 表）
