# 02. Plan — 方案设计

> 基于 [01-clarify.md](./01-clarify.md) 已确认的 6 点决策。本阶段不改生产代码。

---

## 改动文件清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `controller/contribution_skill.go` | 修改 | `HandleTakedownSkill`：按 slug 校验归属 + 绑最新 published 为 `resource_id`；`approveSkillContribution` takedown 分支按 slug 批量 offline |
| `controller/admin_skills.go` | 修改 | `HandleAdminSkills` 的 `pending_review` 改为按 **slug** 关联 pending（兼容旧 `resource_id`） |
| `controller/contribution_skill_test.go` | 修改 | 补充多版本下架 / 展示 / 审批用例；必要时微调既有断言 |
| `docs/API.md` | 修改 | 明确 approve takedown = 整 slug 所有 published→offline；`pending_review` 按 slug 挂到最新列表行 |
| `test/scripts/skill/test_skill_contribute.py` | 修改 | 增补 IT：多版本 published 后下架 → 列表可见 pending → 通过后两版均 offline |
| `.specs/plans/.../03-implement.md` 等 | 后续步骤 | 本 Plan 不改代码 |

**不改**：`review_requests` 表结构、`HasPendingRequest`（已按 slug）、`/admin/skills/offline`、publish 提交流程。

---

## 调用链 / 数据流

### A. 员工申请下架（修复后）

```
POST /openclaw/skills/takedown {slug, reason}
  → requireLogin
  → 校验：存在 skills WHERE slug=? AND status=published AND uploader_id=user.ID
       （无则 403；若该 slug 无任何 published → 404）
  → HasPendingRequest(skill, slug) 互斥（已有，按 slug）
  → 取最新 published：
       ORDER BY version_major DESC, version_minor DESC, version_patch DESC LIMIT 1
  → Create ReviewRequest{ resource_id=最新.id, slug, action=takedown, status=pending }
  → 通知管理员
```

### B. 管理端技能列表挂 pending（修复后）

```
GET /admin/skills
  → 列表仍只返回 LatestVersionSkillIDs（每 slug 一行最新版）
  → 收集本页 slugs
  → SELECT * FROM review_requests
       WHERE resource_type='skill' AND status='pending' AND slug IN (本页 slugs)
  → pendingReviewMap[slug] = info
  → 每行 PendingReview = pendingReviewMap[skill.Slug]
       （旧 pending 的 resource_id 即使指向旧版本，只要 slug 对得上也能挂上）
```

### C. 审核通过下架（修复后）

```
POST /admin/contributions/approve {id}
  → load ReviewRequest
  → approveSkillContribution
       takedown 分支：
         UPDATE skills SET status='offline'
         WHERE slug=? AND status='published'
         （与 HandleAdminSkillOffline 一致，整 slug）
       再更新 ReviewRequest → approved
```

### 对照：现状问题（diagram-imagexxx）

```
v1.0.0 id=4887, v1.0.1 id=4888
申请：resource_id=4887（First 无序）
列表：只展示 4888，按 resource_id 关联 → pending_review=null
通过：只 offline 4887 → 4888 仍 published
```

---

## 关键实现要点

### 1. `HandleTakedownSkill`

| 步骤 | 现逻辑 | 新逻辑 |
|------|--------|--------|
| 查技能 | `Where(slug, published).First`（无序） | 先确认存在本人 published；再单独查最新 published 填 `resource_id` |
| 归属 | 看 First 行的 `UploaderID` | `Count/First`：`slug + published + uploader_id=user.ID` |
| resource_id | First 行 id | 最新版本 id |

建议伪代码：

```go
var owned model.Skill
if db.Where("slug=? AND status=? AND uploader_id=?", slug, published, user.ID).First(&owned).Error != nil {
  // 区分：有 published 但非本人 → 403；完全无 published → 404
}
var latest model.Skill
db.Where("slug=? AND status=?", slug, published).
  Order("version_major DESC, version_minor DESC, version_patch DESC").
  First(&latest)
// resource_id = latest.ID
```

404 vs 403：若 slug 下存在 published 但 `uploader_id` 均不等于本人（含管理员 `0`）→ **403**；若无任何 published → **404**。与现网错误语义对齐。

### 2. `HandleAdminSkills` pending 关联

- 查询条件由 `resource_id IN (skillIDs)` 改为 `slug IN (slugs)`
- map key 由 `uint(resource_id)` 改为 `string(slug)`
- 赋值：`PendingReview: pendingReviewMap[s.Slug]`

同一 slug 理论上只有一个 pending（`HasPendingRequest` 互斥）；若异常多条，取 id 最大的一条即可。

### 3. `approveSkillContribution` takedown

```go
tx.Model(&model.Skill{}).
  Where("slug = ? AND status = ?", skill.Slug, model.SkillStatusPublished).
  Update("status", model.SkillStatusOffline)
```

先 `First` 出 skill 仍可用于取 `Slug`（兼容旧 pending 的旧 resource_id：只要该行未删，slug 正确即可；若旧版本行仍在且 published，slug 批量更新仍会覆盖最新版）。

边界：若 `resource_id` 指向的行已被软删，`First` 失败 → 现有返回「技能不存在」。可选增强：`Unscoped` 或按 `req.Slug` 直接更新。**本期采用：优先用 `req.Slug` 做批量 offline**（申请单已冗余 slug），避免依赖旧 resource_id 行是否还在。

```go
case model.ActionTypeTakedown:
  if req.Slug == "" { return err }
  tx.Model(&Skill{}).Where("slug=? AND status=?", req.Slug, published).
    Update("status", offline)
```

### 4. 文档

`docs/API.md` 中：

- `POST .../takedown`：已写「整个 slug」——保留，并注明申请单 `resource_id` 为最新 published id  
- `POST .../approve`：明确 takedown 时将该 slug 下所有 `published` → `offline`  
- `GET /admin/skills` 的 `pending_review`：注明按 slug 关联，与列表最新行一致  

---

## 数据库变更

无。不改表、不做历史 SQL 修数；靠展示按 slug + approve 按 slug 兼容存量 pending。

---

## 测试用例设计（自然语言描述）

> 先于实现编写，Implement 阶段据此编码。

### 单元测试（UT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| T1 | 多版本下架申请绑最新 id | slug 有 v1.0.0(id小) + v1.0.1(id大) 均为 published、uploader=员工；调用 takedown | 200；ReviewRequest.resource_id = 大 id（最新版） | P0 |
| T2 | 多版本列表可见 pending（含旧 resource_id） | 列表最新为 v1.0.1；手动插入 pending takedown 且 resource_id=旧版 id | `GET /admin/skills?slug=` 最新行 `pending_review.action_type=takedown` 非空 | P0 |
| T3 | 审核通过整 slug offline | 两版 published；approve takedown（resource_id 可指向旧版） | 两版 status 均为 offline；广场不可见 | P0 |
| T4 | 归属：仅存在本人 published 即可 | 两版 uploader=本人 | takedown 200 | P0 |
| T5 | 归属：管理员上传（uploader_id=0） | published 且 uploader=0 | 403 | P0（已有，保持） |
| T6 | 单版本回归 | 仅一版 published | takedown → approve → offline，行为与现网一致 | P0（已有，保持） |
| T7 | 互斥 | 已有 pending | 再次 takedown 400 | P0（已有） |
| T8 | reject takedown | 多版本 pending takedown 后 reject | 两版仍 published | P0 |

### 集成测试（IT）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| IT-M1 | 多版本下架闭环 | 员工 contribute 1.0.0→approve；再 contribute 1.0.1→approve；takedown | admin skills 最新行 pending_review 非空；approve 后两版均 offline；skillstore 不可见 | P0 |
| IT-M2 | 旧 pending 兼容（可选） | 若 IT 不便造旧 resource_id，可由 UT-T2 覆盖 | — | P1 |

脚本落点：`test/scripts/skill/test_skill_contribute.py` 新增阶段即可（不强制新文件）。

---

## 风险评估

| # | 风险 | 概率 | 影响 | 缓解 |
|---|------|------|------|------|
| 1 | 存量 pending 的 resource_id 仍指向旧版，详情页展示旧版元数据 | 中 | 低（列表已可见；详情仍可读 slug） | 详情以 req.Slug 为准；可选后续再优化 |
| 2 | approve 按 slug 批量 offline 误伤同 slug 其他状态 | 低 | 中 | WHERE 限定 `status=published`，不动 pending_review/offline |
| 3 | 列表按 slug 查 pending 性能 | 低 | 低 | 仅本页 slug 列表 IN 查询，与现 resource_id IN 同量级 |
| 4 | 文档已写「整 slug」但实现不符，修复后行为与文档对齐 | — | — | Docs 步骤再写清 approve 语义，避免歧义 |

---

## 实施顺序建议

1. 改 `HandleTakedownSkill`（归属 + 最新 resource_id）  
2. 改 `approveSkillContribution` takedown（按 `req.Slug` 批量 offline）  
3. 改 `HandleAdminSkills` pending 按 slug  
4. UT（T1–T8）  
5. Docs  
6. IT-M1  

---

## 验收对照（Clarify）

| Clarify 验收项 | Plan 覆盖 |
|----------------|-----------|
| 多版本下架后列表 `pending_review` 非空 | B + T2 + IT-M1 |
| contributions 列表仍可查 | 不改 list API |
| approve 后全部 published→offline | C + T3 + IT-M1 |
| 单版本兼容 | T6 |
| 旧 pending 无需修库 | B 按 slug |
| 单测覆盖多版本 | T1–T3 |
