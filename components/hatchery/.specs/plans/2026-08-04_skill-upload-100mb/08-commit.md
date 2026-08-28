# 08. Commit — 提交记录

---

## 提交前检查

> **执行顺序（强制）**：
> ① 写完本文件 → ② 更新 `00-overview.md` 状态为已完成 → ③ 确认所有 `0N-*.md` 已就绪 → ④ `git add` → ⑤ `git commit` → ⑥ `git push`

- [x] 所有 `0N-*.md` 产物已完成
- [x] `00-overview.md` Progress 全部勾选
- [x] `gofmt` + `go vet` 通过
- [x] 单元测试通过（P0 相关用例）
- [x] `docs/API.md` 已更新
- [x] 新 UT 文件 `controller/skill_upload_test.go` 纳入提交

## Commit Message

```
feat(controller): raise skill upload size limit to 100MB

Skill zip 上传上限从 50MB 调整为 100MB（管理端 create 与员工 contribute 共用）。
上传超限文案与 Bundle 下载 50MB 文案拆分，避免误报；同步 API 文档与边界单测。
```

## 提交文件清单

| 文件 | 改动类型 |
|------|---------|
| `controller/skill_upload.go` | 修改 |
| `controller/skill_upload_test.go` | 新增 |
| `controller/admin_skills_test.go` | 修改 |
| `i18n/keys.go` | 修改 |
| `i18n/en.go` | 修改 |
| `docs/API.md` | 修改 |
| `.specs/plans/2026-08-04_skill-upload-100mb/` | 新增/完善（SOP 产物） |
