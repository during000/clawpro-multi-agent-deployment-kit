# 03. Implement — 实现细节

> 基于 [02-plan.md](./02-plan.md) 实施，记录关键改动与差异。

---

## 一、已完成改动

| 文件 | 改动 |
|------|------|
| `controller/contribution_skill.go` | `HandleTakedownSkill`：按 slug 校验本人 published；`resource_id` 绑最新 published；`approveSkillContribution` takedown：按 `req.Slug` 批量 published→offline |
| `controller/admin_skills.go` | `pending_review` 改为按 slug 查询/挂载（`map[string]`），兼容旧 resource_id |

---

## 二、关键实现细节

### 2.1 HandleTakedownSkill

1. 归属：`Where(slug, published, uploader_id=user.ID).First`  
   - 失败时再 Count 是否存在任意 published → 有则 **403**，无则 **404**
2. 互斥：沿用 `HasPendingRequest`（按 slug）
3. 最新版：`Order(version_major/minor/patch DESC).First` → `resource_id = latest.ID`

### 2.2 approveSkillContribution (takedown)

- publish 分支仍按 `resource_id` 单行上架（不变）
- takedown：优先用 `req.Slug`；若为空则 fallback 查 `resource_id` 取 slug
- `UPDATE skills SET status=offline WHERE slug=? AND status=published`

### 2.3 HandleAdminSkills pending_review

- 收集本页 `slugs`（去重）
- `WHERE resource_type=skill AND slug IN (?) AND status=pending ORDER BY id DESC`
- `pendingReviewMap[slug]`，同 slug 只保留 id 最大的一条
- 赋值：`PendingReview: pendingReviewMap[s.Slug]`

---

## 三、与 Plan 差异

| # | Plan | 实际 | 说明 |
|---|------|------|------|
| 1 | 无实质差异 | — | 按 Plan 三点全部落地 |

---

## 四、编译验证

```
go build ./controller/  → 通过
```

---

## 五、待后续步骤

- [ ] UT：多版本 T1–T3 等用例
- [ ] Docs：API.md 明确 approve 整 slug
- [ ] IT：test_skill_contribute.py 多版本下架闭环
