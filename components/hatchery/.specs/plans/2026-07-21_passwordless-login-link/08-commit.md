# 08. Commit — 提交记录

---

## 提交前检查

- [x] 所有 `0N-*.md` 产物已完成
- [x] `00-overview.md` Progress 将在提交前全部勾选
- [x] Go 变更已执行 `gofmt`
- [x] 聚焦单元测试与 race 测试通过
- [x] 前端目标文件 ESLint 通过
- [x] K8s API、浏览器 E2E 与新增接口覆盖率通过
- [x] `AGENTS.md`、`docs/openapi_base.json` 不提交且不修改 `.gitignore`
- [x] 前端既有 `Dockerfile` 用户修改不提交
- [x] 前端 `i18n/ClawPro/*.json` 生成翻译产物不提交

## Commit Messages

后端仓库：

```
feat(auth): add passwordless login links
docs(auth): finalize passwordless login task
refactor(model): centralize feature allowlist policies
```

前端仓库：

```
feat(auth): add passwordless login landing page
```

## 推送目标

- 后端：`origin/feature/passwordless-login-link`
- 前端：`origin/feature/passwordless-login-link`
- 两个仓库均已 rebase 到 `origin/Release/2026_07_21`，使用 `git push --force-with-lease origin feature/passwordless-login-link` 更新远端同名分支。
