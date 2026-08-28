# 实现：修复 imageModel 字符串格式不兼容

## 改动概要

按 Plan 方案修改了 3 个 shell 脚本的 jq 表达式，在操作 `agents.defaults.imageModel` 前增加类型判断，兼容字符串格式（旧）和对象格式（新）。

## 关键实现细节

### 1. set_model.sh / switch_model.sh（相同逻辑）

在 `else` 分支（`$imageprimary` 非空）中，设置 `.imageModel.primary` 前插入类型判断：

```jq
(if (.agents.defaults.imageModel | type) == "string" then
   .agents.defaults.imageModel = {}
 else . end)
| .agents.defaults.imageModel.primary = $imageprimary
| .agents.defaults.imageModel.fallbacks = $imagefallbacks
```

- `type == "string"` → 重置为 `{}`，后续赋值正常写入对象属性
- `type == "object"` 或 `"null"` → `else .`（不变），后续赋值正常执行（null 时 jq 自动创建对象）

### 2. remove_model_provider.sh

将原来的两段独立 `if`（步骤 4 + 步骤 5）合并为一个 `if/elif/else`：

```jq
(if (.agents.defaults.imageModel | type) == "string" then
    del(.agents.defaults.imageModel)
  elif (.agents.defaults.imageModel | type) == "object" then
    (if .agents.defaults.imageModel.fallbacks then ... end)
    | (if (.agents.defaults.imageModel.primary // "" | startswith($prefix)) then ... end)
  else . end)
```

- `string` → 直接 `del`（旧格式残留，无需保留）
- `object` → 走原逻辑（剔 fallbacks + 清悬空 primary）
- `null`/不存在 → `else .`（不变）

## 与 Plan 差异

无差异，完全按 Plan 方案实现。

## 验证结果

用 jq 命令行对修改后的表达式进行了 6 个场景验证，全部通过：

| 用例 | 输入 imageModel | 操作 | 输出 |
|------|----------------|------|------|
| TC-01 | `"old-ref"`（字符串） | set_model primary="new-ref" | `{"primary":"new-ref","fallbacks":[]}` ✅ |
| TC-02 | `{"primary":"old","fallbacks":["old-fb"]}`（对象） | set_model primary="new-ref" | `{"primary":"new-ref","fallbacks":["new-fb"]}` ✅ |
| TC-03 | 不存在 | set_model primary="new-ref" | `{"primary":"new-ref","fallbacks":[]}` ✅ |
| TC-07 | `"hatchery-glm-4-plus/glm-4-plus"`（字符串） | remove provider | imageModel 被 del ✅ |
| TC-08 | `{"primary":"hatchery-glm-4-plus/...","fallbacks":[...]}`（对象） | remove provider | imageModel 被 del（primary 匹配） ✅ |
| TC-09 | 不存在 | remove provider | 不变 ✅ |

## 改动文件

| 文件 | 改动行数 |
|------|---------|
| `scripts/set_model.sh` | +4 |
| `scripts/switch_model.sh` | +4 |
| `scripts/remove_model_provider.sh` | +6 / -8（重构为 if/elif/else） |
