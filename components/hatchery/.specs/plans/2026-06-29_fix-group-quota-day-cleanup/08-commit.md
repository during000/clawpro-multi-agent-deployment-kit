# 08. Commit — 提交

> 按 SOP 执行顺序：① 写 commit message → ② 更新 00-overview Meta → ③ 确认产物就绪 → ④ jj 提交 → ⑤ 同步 git

---

## Commit Message（中文，除前缀）

```
fix(controller): 修复分组配额 *_day 与 *_rules 绑定不对称导致的删不掉/刷新复活问题

HandleSetGroupPolicy 写 token_quota_rules / global_token_quota_rules 时，
事务内同步清理配对的 legacy *_day 绑定，与既有写 *_day 路径对称。

HandleDeleteGroupPolicy 当目标 key 绑定不存在时，fallback 尝试删除配对
key 绑定，避免组仅有 legacy *_day 绑定时前端删 *_rules 报"未配置"错误。

新增 QuotaPolicyPairedKey 辅助函数；9 个单元测试覆盖新增分支。
```

**type**：fix（修复分组配额兼容缺陷）
**scope**：controller
**红线关联**：无新增 API/参数，无需 IT；未改 model schema；复用既有 i18n Key

---

## 提交前检查清单

| 项 | 状态 |
|----|------|
| 08-commit.md 已写 | ✅ |
| 00-overview.md Meta 状态 → 已完成 | ⏳ 待更新 |
| 00-overview.md Progress 08 勾选 | ⏳ 待更新 |
| 00-overview.md 时间记录 08 填入 | ⏳ 待更新 |
| 所有 0N-*.md 就绪 | ✅ 01-07 均已产出 |
| gofmt / go vet / go test | ✅ 通过（见 04-ut.md / 07-review.md） |
| 改动文件清单 | controller/usergroup/types.go、controller/admin_group_config.go、controller/admin_group_config_coverage_test.go、docs/API.md |

---

## jj 提交计划

当前 jj 状态：
- `nppptwlw`（已 desc）：impl + UT 改动
- `otkzozzr`（当前工作副本，空 desc）：Docs + Review 改动 + Review 一致性修复

**方案**：`jj squash` 将当前工作副本改动折叠进 `nppptwlw`，形成单一干净提交，再用 `jj desc -r` 更新为最终中文 commit message。

---

## 执行结果

**jj 提交**：
- `jj squash` 将 Docs/Review 改动折叠进 impl+UT 提交，形成单一干净提交
- `jj desc -r nppptwlw` 更新为最终中文 commit message
- `jj bookmark set feature/fix-group-quota-day-cleanup -r nppptwlw` + `jj git export` 同步 git 分支

**最终提交**：
```
b924f7ab (HEAD, feature/fix-group-quota-day-cleanup)
  fix(controller): 修复分组配额 *_day 与 *_rules 绑定不对称导致的删不掉/刷新复活问题
  parent: 03568559 (origin/master)
```

**改动统计**：13 文件，+1034/-5
- `controller/usergroup/types.go`（+17，新增 QuotaPolicyPairedKey）
- `controller/admin_group_config.go`（+40/-3，Set rules 事务清理 + Delete fallback）
- `controller/admin_group_config_coverage_test.go`（+266，9 用例 + helper）
- `docs/API.md`（+4/-1，双向对称 + fallback 语义）
- `.specs/plans/2026-06-29_fix-group-quota-day-cleanup/`（9 个产物文件）

**未 push**：本地提交完成，未推送到 origin（push 影响共享状态，需用户确认）。
