# 07. Review — 代码审查

> 对照 [02-plan.md](./02-plan.md)、Clarify 验收项与实现/测试/文档产物。本步骤不改业务逻辑。

**开始时间**：2026-08-04 19:48:56  
**结论**：**PASS**（可进入 Commit）

---

## 一、Plan 覆盖核对

| Plan 项 | 实现 | 判定 |
|---------|------|------|
| A. `HandleTakedownSkill`：本人 published 归属 + 403/404 区分 | `contribution_skill.go` 178–193 | ✅ |
| A. `resource_id` = 最新 published（version DESC） | 201–214 | ✅ |
| B. `pending_review` 按 slug IN 查询 + map[string] | `admin_skills.go` 509–561, 717 | ✅ |
| B. 同 slug 多条 pending 取 id 最大 | `Order("id DESC")` + exists skip | ✅ |
| C. approve takedown 按 `req.Slug` 批量 published→offline | `contribution_skill.go` 445–459 | ✅ |
| C. slug 空时 fallback resource_id 取 slug | 448–454 | ✅ |
| 不改表 / 不做历史修数 | 无 schema/SQL 变更 | ✅ |
| Docs：takedown / approve / pending_review 语义 | `docs/API.md` | ✅ |
| UT T1–T5,T8 + 既有回归 | `contribution_skill_test.go` 多版本 5 例 + 既有 | ✅ |
| IT-M1 脚本 | `test_skill_contribute.py` | ✅（远程待部署） |

与 Plan **无实质差异**；`approve` slug 空 fallback 为 Plan 已写明的增强。

---

## 二、规范与安全

| 检查项 | 结果 |
|--------|------|
| HTTP 内 `model.DB(r.Context())` | ✅ 三处改动均 ctx-aware |
| 禁止裸 SQL（Exec/Raw/Table…） | ✅ 仅 GORM Model API |
| 腾讯云 Client 自建 | N/A |
| 新写接口 audit | N/A（无新路由；既有 takedown/approve 已 `WithAudit`） |
| 公开 API 破坏性变更 | ✅ 仅澄清语义；响应字段未删改含义（pending 关联规则修复为与文档一致） |
| MySQL 双维护 | N/A（无模型变更） |

---

## 三、回归风险点

| # | 点 | 评估 |
|---|----|------|
| 1 | 存量 pending `resource_id` 指向旧版 | 列表按 slug 可见；approve 按 slug 整批 offline → **已覆盖线上 bug** |
| 2 | 同 slug 仅部分版本 uploader=本人 | Plan 确认：存在本人 published 即可；最新版可能非本人上传 → 仍可下架整 slug（与 offline 对齐） |
| 3 | publish 审核路径 | 未改，仍按 resource_id 单行 |
| 4 | 远程 IT-M1 | 未部署本分支前会失败；UT 已覆盖根因，不阻塞合入 |

---

## 四、测试证据

```
go test ./controller/ -run 'TestHandleTakedownSkill_MultiVersion|TestHandleAdminSkills_PendingReview_BySlug|TestHandleApprove_Takedown_MultiVersion|TestHandleReject_Takedown_MultiVersion|TestHandleTakedownSkill_Success|TestHandleTakedownSkill_AdminUploaded|TestHandleTakedownSkill_Mutex' -count=1
→ ok hatchery/controller
```

---

## 五、改动文件清单（待 Commit）

| 文件 | 说明 |
|------|------|
| `controller/contribution_skill.go` | takedown 归属/最新 id；approve 整 slug offline |
| `controller/admin_skills.go` | pending_review 按 slug |
| `controller/contribution_skill_test.go` | 多版本 UT |
| `docs/API.md` | 语义澄清 |
| `test/scripts/helpers/contribution.py` | skillstore detail 探测 |
| `test/scripts/skill/test_skill_contribute.py` | IT-M1 |
| `.specs/plans/2026-08-04_fix-skill-takedown-multiversion/` | SOP 产物 |

---

## 六、Nits（不阻塞）

1. UT `TestHandleApprove_Takedown_MultiVersion_AllOffline` 请求体写死 `"id":1`，依赖空库自增 — 现有套件惯例，可接受。  
2. 详情页若仍按 `resource_id` 展示元数据，旧 pending 可能显示旧版字段（Plan 风险#1，影响低）。

---

## 七、结论

**PASS** — 三点根因均已按 Clarify/Plan 落地，UT 绿，文档与 IT 脚本齐备。可进入 **08. Commit**。
