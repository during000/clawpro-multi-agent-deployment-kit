# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

已发布的技能共建审核能力中，员工可对**自己上传**的技能申请下架（`POST /openclaw/skills/takedown`）。线上反馈：技能 `diagram-imagexxx` 上架与版本更新均正常，提交下架申请后，管理员在技能列表上看不到待审批标记。

线上数据已核实：

| 对象 | 关键字段 |
|------|----------|
| skills 1.0.0 | id=`4887`, status=published |
| skills 1.0.1（最新） | id=`4888`, status=published |
| review_requests#4 | action=`takedown`, status=`pending`, **resource_id=`4887`** |

即：申请已成功入库，但挂在**旧版本** id 上；管理端技能列表只展示每个 slug 的**最新版本**，并用「当前行 skill.id == resource_id」挂 `pending_review`，因此最新行上 `pending_review = null`，看起来像「没收到申请」。

### 根因（代码）

三处语义不一致，叠加触发：

1. **创建下架申请**（`HandleTakedownSkill`）  
   `Where(slug, published).First(&skill)` **无 ORDER BY** → 默认取 id 最小的旧版本 → `resource_id` 写成旧 id。

2. **管理端技能列表**（`HandleAdminSkills`）  
   只返回 `LatestVersionSkillIDs` 最新版；`pending_review` 按 `resource_id IN (本页 skill ids)` 关联 → 旧 id 的 pending 挂不到最新行。

3. **审核通过下架**（`approveSkillContribution`）  
   只 `Update` `id = resource_id` 那一行 → 即使审过，也只下架旧版本；最新版仍 published。  
   对比：管理员直下架 `POST /admin/skills/offline` 是按 **slug** 把所有 published 改为 offline。

产品语义（一期共建澄清 + offline 接口行为）：**下架针对整个 slug，不是单个版本。**

---

## 目标

修复多版本场景下「下架申请看不见 / 审过只下架一版」的问题，使行为与「整技能下架」一致。

验收标准：

- [ ] 员工对已有多版本 published 的技能提交下架后，`GET /admin/skills?slug=...` 最新行的 `pending_review` 非空，且 `action_type=takedown`
- [ ] `GET /admin/contributions?status=pending&action_type=takedown` 仍可查到该申请（不回归）
- [ ] 管理员审核通过后，该 slug 下**所有** `published` 版本变为 `offline`（与 `/admin/skills/offline` 一致）
- [ ] 仅单版本时行为与现网一致（兼容）
- [ ] 已存在的「挂在旧 resource_id 上的 pending takedown」在修复后，技能列表也能显示 pending（无需手工改库）
- [ ] 相关单测覆盖多版本场景；无 schema 变更

---

## 范围

| 包含 | 不包含 |
|------|--------|
| 修复下架申请创建时选取/绑定 skill 的方式 | 改动 publish 提交流程（上架仍按单版本） |
| 修复管理端 `pending_review` 关联方式（按 slug） | 前端改动（后端字段语义对齐即可） |
| 修复审核通过 takedown：按 slug 整技能 offline | 历史数据 SQL 迁移脚本（优先代码兼容旧 pending） |
| 补充/修正单元测试与 API 文档中下架语义说明 | 管理员 offline/online 接口本身（已正确，仅对齐） |
| 可选：IT 脚本补多版本下架场景 | 通知文案/审计规则变更 |

---

## Challenge（挑战已有假设）

| # | 挑战点 | 分析 | 倾向 |
|---|--------|------|------|
| C1 | 是否只改展示、不改 approve？ | 只改展示能「看见申请」，但通过后最新版仍在架，用户仍会投诉「审了没下架」 | **必须同时改 approve** |
| C2 | `resource_id` 是否改为永远绑最新版？ | 新申请绑最新可改善；旧 pending 仍是旧 id，故展示层必须按 **slug** 查 pending，不能只靠改写入 | **展示按 slug；写入建议绑最新 published** |
| C3 | 权限校验用旧版还是最新版的 `uploader_id`？ | 共建路径各版本 `uploader_id` 一般相同；应以「该 slug 是否存在本人上传的 published 版本」为准，避免 First 到脏数据 | **按 slug 校验归属（存在 uploader_id=本人的 published）** |
| C4 | 是否要数据修复脚本？ | 线上已有挂旧 id 的 pending；若展示按 slug，无需改历史行 | **不做 migration，代码兼容** |

---

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 下架产品语义是否确认为「整 slug 所有 published 版本 → offline」（对齐 `/admin/skills/offline`）？ | 已确认 | ✅ 整 slug 下架 |
| 2 | 管理端 `pending_review` 是否改为按 **slug** 关联 pending（从而兼容已落库的旧 resource_id）？ | 已确认 | ✅ 按 slug 关联 |
| 3 | 新创建的 takedown 申请：`resource_id` 是否改为绑定**最新** published 版本 id（冗余字段仍保留，详情可展示）？ | 已确认 | ✅ 绑最新 published |
| 4 | 审核通过 takedown 时，是否按 slug 更新所有 `status=published` 的版本为 offline？ | 已确认 | ✅ 按 slug 批量 offline |
| 5 | 线上已存在的 pending（如 diagram-imagexxx#4）是否仅靠代码兼容展示+通过即可，不做一次性 SQL 修补？ | 已确认 | ✅ 不做 SQL 修数，代码兼容 |
| 6 | 归属校验：是否要求「存在 uploader_id=当前用户且 published 的版本」即可申请下架（不再依赖 First 到的那一行）？ | 已确认 | ✅ 按 slug 校验本人 published 版本 |

---

## 约束与依赖

- Base：`Release/2026_08_04` / 分支：`feature/fix-skill-takedown-multiversion`
- 不改表结构；`review_requests.resource_id` / `slug` 字段保留
- 多租户：继续走 `model.DB(r.Context())`
- 需回归：单版本下架、互斥 pending、非 owner 403、管理员 offline 行为不变
- 文档：`docs/API.md` 中 takedown / approve 对「整技能下架」语义写清楚

---

## 已确认决策摘要

用户于 2026-08-04 确认上述 6 点全部采纳，可作为 Plan / Implement 的约束输入。