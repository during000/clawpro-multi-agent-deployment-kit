# 08. Commit — 提交记录

---

## 提交前检查

> **执行顺序（强制）**：
> ① 写完本文件 → ② 更新 `00-overview.md` 状态为已完成 → ③ 确认所有 `0N-*.md` 已就绪 → ④ `git add` → ⑤ `git commit` → ⑥ `git push`

- [x] 所有 `0N-*.md` 产物已完成
- [x] `00-overview.md` Progress 全部勾选
- [x] `gofmt` + `go vet` 通过
- [x] 单元测试通过

## Commit Message

```
--story=136018709 【clawpro】企业技能库和mcp库批量下发的时候均需支持全选，不能只单选当前页内容

https://tapd.woa.com/tapd_fe/20422209/story/detail/1020422209136018709
```

## 发布方式

- 将 Review 窄化校验抽象 amend 到现有提交，并使用 `git push --force-with-lease` 覆盖远端。
- 不提交工作区外的 `AGENTS.md`。
- `search` 增量已获用户确认，将 amend/push；继续排除未跟踪的 `AGENTS.md`。
