# 08. Commit — 提交记录（TAPD 160256132）

## 提交前检查

- [x] 所有 `0N-*.md` 产物已完成
- [x] `00-overview.md` Progress 全部勾选
- [x] `gofmt` 通过
- [x] `go vet ./...` 通过
- [x] 定向 race 单元测试通过
- [x] `controller` 全包测试通过
- [x] `skillhubclient` 测试通过
- [x] `go build ./...` 通过
- [x] `git diff --check` 通过
- [x] 原 `hatchery/master` 工作区保持干净

## Commit Message

```text
--bug=160256132 【clawpro bug 单】企业技能创建时需保证skill文件名称为uft-8编码

https://tapd.woa.com/tapd_fe/20422209/bug/detail/1020422209160256132
```

## 推送目标

```text
origin/bugfix/160256132-skill-filename-utf8
```
