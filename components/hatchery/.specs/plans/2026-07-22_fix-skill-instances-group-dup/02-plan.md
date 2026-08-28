# 02. Plan — 方案设计

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `controller/admin_skills.go` | 修改 | 将 `group_id` 多分组筛选的 `Joins` 改为 `Where + 子查询` |
| `controller/admin_skills_instances_test.go` | 修改 | 新增多分组重复场景测试用例 |

## 调用链 / 数据流

```
GET /admin/skills/instances?slug=xxx&group_id=112,119,81
  → HandleAdminSkillInstances()
    → BuildSkillInstanceQuery() — 基础查询 (instances + users + lr + lis)
    → group_id 过滤（本节修复点）
       修复前: JOIN user_group_members → 多分组时重复行
       修复后: WHERE instances.user_id IN (子查询) → 无重复
    → search / instance_type 过滤
    → 全量查询 + ResolveInstanceStatus 过滤 running
    → 内存分页
```

## 数据库变更

无。

## 修复方案

### 改动点（`controller/admin_skills.go` 第 2001-2005 行）

**修改前**：
```go
} else if len(groupIDs) > 0 {
    // 仅指定分组
    baseQuery = baseQuery.Joins(
        "JOIN user_group_members ugm ON ugm.user_id = instances.user_id AND ugm.user_group_id IN ?", groupIDs)
}
```

**修改后**（对齐 `admin_plugins.go:1650-1652` 的已有正确实现）：
```go
} else if len(groupIDs) > 0 {
    // 仅指定分组（使用子查询避免 JOIN 产生重复行）
    groupedSubQ := model.DB(r.Context()).Model(&model.UserGroupMember{}).Select("DISTINCT user_id").Where("user_group_id IN ?", groupIDs)
    baseQuery = baseQuery.Where("instances.user_id IN (?)", groupedSubQ)
}
```

### 为什么子查询不会重复

`SELECT DISTINCT user_id` 确保即使一个用户在 `user_group_members` 中有多行，子查询也只返回一行，`WHERE instances.user_id IN (...)` 语义正确。

## 测试用例设计（自然语言描述）

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 用户属于多个分组时查询无重复 | 用户 alice 属于 groupA 和 groupB，alice 有 2 个实例；查询 `group_id=groupA,groupB` | `total=2`，每个实例只出现一次 | P0 |
| 2 | 多分组内不同用户各自有实例 | 用户 alice(groupA) 有 1 实例，bob(groupB) 有 1 实例；查询 `group_id=groupA,groupB` | `total=2`，两个实例分别出现一次 | P0 |
| 3 | 单分组筛选行为不变（回归） | 同`TestHandleAdminSkillInstances_GroupIDFilter_NormalGroup` | 行为不变 | P1 |
| 4 | 未分组+多分组组合不变（回归） | 同`TestHandleAdminSkillInstances_GroupIDFilter_UngroupedPlusNormal` | 行为不变 | P1 |

### 集成测试（IT）

不涉及集成测试（此修改为 SQL 查询级别修复，无需额外集成测试）。

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 子查询性能不如 JOIN | 低 | 低 | user_group_members 数据量小，且已有 `admin_plugins.go` 线上验证 |
| 2 | 与其他筛选条件组合异常 | 极低 | 中 | 已有回归测试覆盖组合筛选 |
