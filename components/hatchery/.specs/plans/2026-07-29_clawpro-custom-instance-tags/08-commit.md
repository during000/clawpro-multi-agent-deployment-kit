# 08. Commit — 提交记录

## 提交前检查

- [x] 所有 `0N-*.md` 产物已完成
- [x] `00-overview.md` Progress 全部勾选
- [x] `gofmt`、聚焦测试、全量测试和 `go vet ./controller/...` 通过
- [x] 全仓 vet 的既有告警、未安装 staticcheck、真实云 IT 结果已分别记录在 `04-ut.md` / `06-it.md`
- [x] 单元测试通过

## Commit Message

```text
--story=136340282 【clawpro】创建agent支持可以自由指定标签
```

## 提交结果

- 分支：`feature/story-136340282-custom-instance-tags`
- 提交：本文件随上述 story commit 一并入库，最终哈希以该分支 `git log -1` 为准
- 推送目标：`origin/feature/story-136340282-custom-instance-tags`

## 真实云 IT 后续修复

真实云验证发现并修复创建期标签缓存只记录默认标签的问题，后续提交信息：

```text
--story=136340282 修复自定义标签创建期缓存一致性
```
