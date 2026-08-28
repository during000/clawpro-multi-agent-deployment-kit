# 08. Commit — 提交记录

---

## 提交前检查

> **执行顺序（强制）**：
> ① 写完本文件 → ② 更新 `00-overview.md` 状态为已完成 → ③ 确认所有 `0N-*.md` 已就绪 → ④ `git add` → ⑤ `git commit` → ⑥ `git push`

- [x] `01`–`08` 阶段产物均已就绪
- [x] `00-overview.md` 已准备更新为完成状态
- [x] 变更 Go 文件 `gofmt` 无差异；LSP 跨 package 缓存误报由目标编译、`go vet` 与测试排除
- [x] Model、Controller 目标单测与 race 检查通过
- [x] 三个列表脚本 `bash -n`、IT Python `py_compile` 通过
- [x] `go test . -run '^$' -count=1` 与 `git diff --check` 通过
- [x] 全仓 `go vet ./...` 通过
- [x] 提交范围明确排除无关的 untracked `AGENTS.md`

## Commit Message

```text
--story=135599722 【clawpro】用户端-支持对公共技能的更新和卸载

https://tapd.woa.com/tapd_fe/20422209/story/detail/1020422209135599722
```

## 提交范围

- 从 Admin 分发 task/record 还原当前技能来源与版本
- 用户同步更新、卸载接口及 `skill_update` / `skill_uninstall` 审计动作
- Admin 同步/异步共用任务执行核心
- 三种 Agent 的稳定 slug 与卸载能力识别
- 单元测试、集成测试、API 文档与 SOP 阶段产物

不包含：仓库根目录下无关的 `AGENTS.md`。

## 历史整理

- 分支已 rebase 到 `origin/master`
- master 之后的实现、修复和性能收敛提交已 squash 为上述单一 story 提交
- 不保留中间提交、兼容别名或旧术语
