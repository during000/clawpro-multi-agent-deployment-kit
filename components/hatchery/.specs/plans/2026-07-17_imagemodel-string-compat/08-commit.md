# Commit

## Commit Message

```
fix(scripts): compat legacy string-format imageModel in model scripts

历史龙虾 openclaw.json 中 agents.defaults.imageModel 为字符串格式（旧版），
而 set_model.sh / switch_model.sh / remove_model_provider.sh 的 jq 表达式
假设其为对象格式，直接访问 .primary/.fallbacks 导致 "Cannot index string"
报错，用户添加/切换/删除模型全部失败。

修复：在三个脚本的 jq 中增加 type == "string" 类型判断：
- set_model.sh / switch_model.sh：字符串格式先重置为 {} 再写入属性
- remove_model_provider.sh：字符串格式直接 del（旧格式残留）

新增 test/scripts/test_imagemodel_compat.sh 覆盖 10 个场景（字符串/对象/
不存在 × set/switch/remove），全部通过。

TAPD: bug 1020422209160822747
```

## 提交前检查

- [x] 三个脚本已修改且 jq 验证通过
- [x] 测试脚本已创建且 10/10 通过
- [x] 文档无需更新（API 契约不变）
- [x] Code Review 通过
- [x] commit message 符合 Conventional Commits 规范

## 提交文件

| 文件 | 操作 |
|------|------|
| `scripts/set_model.sh` | 修改 |
| `scripts/switch_model.sh` | 修改 |
| `scripts/remove_model_provider.sh` | 修改 |
| `test/scripts/test_imagemodel_compat.sh` | 新增 |
