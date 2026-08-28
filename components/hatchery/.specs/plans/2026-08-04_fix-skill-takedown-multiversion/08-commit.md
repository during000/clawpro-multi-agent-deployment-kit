# 08. Commit — 提交记录

> 基于 Review PASS，将修复合入 feature 分支。

**开始时间**：2026-08-04 19:51:09  
**结束时间**：2026-08-04 19:51:22

---

## 提交结果

| 项 | 值 |
|----|-----|
| 分支 | `feature/fix-skill-takedown-multiversion` |
| Commit | 分支 tip（`git log -1`，message 见下） |
| 相对 | ahead of `origin/Release/2026_08_04` by 1 |
| 工作区 | clean |

### Message

```
fix(controller): 多版本技能下架按 slug 展示 pending 并整技能 offline

修复管理端列表只展示最新版时，下架申请挂在旧 version id 导致
pending_review 为空、审核只下架旧版的问题。
```

### 含文件（15）

- `controller/contribution_skill.go` / `admin_skills.go` / `contribution_skill_test.go`
- `docs/API.md`
- `test/scripts/helpers/contribution.py` / `skill/test_skill_contribute.py`
- `.specs/plans/2026-08-04_fix-skill-takedown-multiversion/*`

---

## 后续（本 SOP 不自动执行）

- Push：`git push -u origin HEAD`
- MR：目标分支 `Release/2026_08_04`
- 部署后跑 `test/scripts/skill/test_skill_contribute.py`（IT-M1）
