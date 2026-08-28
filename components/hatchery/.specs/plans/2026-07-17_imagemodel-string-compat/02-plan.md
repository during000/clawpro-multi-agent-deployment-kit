# 方案设计：修复 imageModel 字符串格式不兼容

## 改动文件清单

| # | 文件 | 改动类型 | 说明 |
|---|------|---------|------|
| 1 | `scripts/set_model.sh` | 修改 | jq 中设置 imageModel.primary 前增加类型判断 |
| 2 | `scripts/switch_model.sh` | 修改 | 同上 |
| 3 | `scripts/remove_model_provider.sh` | 修改 | jq 中访问 imageModel 属性前增加类型判断 |

> 不修改任何 Go 代码。Go 侧 `buildImageModelRefs` 生成脚本参数的逻辑不变。

## 改动详情

### 1. set_model.sh（第 195-200 行）

**修改前**：
```jq
| if $imageprimary == "" then
    del(.agents.defaults.imageModel)
  else
    .agents.defaults.imageModel.primary = $imageprimary
    | .agents.defaults.imageModel.fallbacks = $imagefallbacks
  end
```

**修改后**：
```jq
| if $imageprimary == "" then
    del(.agents.defaults.imageModel)
  else
    # 兼容旧格式：imageModel 为字符串时先重置为空对象，避免 jq "Cannot index string" 报错
    (if (.agents.defaults.imageModel | type) == "string" then
       .agents.defaults.imageModel = {}
     else . end)
    | .agents.defaults.imageModel.primary = $imageprimary
    | .agents.defaults.imageModel.fallbacks = $imagefallbacks
  end
```

**原理**：当 `imageModel` 是字符串（旧格式）时，先将其重置为 `{}`，随后 `.primary = $imageprimary` 和 `.fallbacks = $imagefallbacks` 正常写入对象属性。当 `imageModel` 是对象或 null 时，`type` 返回 `"object"` 或 `"null"`，走 `else .`（不变），后续赋值正常执行。

### 2. switch_model.sh（第 142-147 行）

与 set_model.sh 完全一致的修改逻辑。

### 3. remove_model_provider.sh（第 63-74 行）

**修改前**：两段独立 if（步骤4剔fallbacks + 步骤5清悬空primary）

**修改后**：合并为 if/elif/else，按类型分派：字符串直接del、对象走原逻辑、null不变。

**原理**：
- 字符串格式 → 直接 `del`（旧格式 imageModel 是过时残留，删除后由后续 add-model 重建）
- 对象格式 → 走原有逻辑（剔 fallbacks + 清悬空 primary），行为不变
- null/不存在 → `else .`（不变），兼容无 imageModel 字段的实例

## 调用链分析

```
用户操作（添加/切换/删除模型）
  → Go handler (controller/openclaw_model.go)
    → buildImageModelRefs() 生成 imagePrimary + imageFallbacks 参数
    → RunScript() 下发 TAT 脚本
      → set_model.sh / switch_model.sh / remove_model_provider.sh
        → jq 操作 openclaw.json 中的 imageModel  ← 修复点
      → systemctl restart openclaw-gateway
```

Go 侧不直接操作 openclaw.json，仅通过脚本参数传递 imagePrimary 和 imageFallbacks。修复仅涉及脚本中的 jq 表达式，不影响 Go 调用链。

## 测试用例设计（自然语言描述）

### TC-01: set_model.sh — imageModel 为字符串格式
- **场景**：openclaw.json 中 `imageModel` 是字符串
- **输入**：`imageprimary="hatchery-dall-e-3/dall-e-3"`，`imagefallbacks='[]'`
- **预期**：脚本成功执行，`imageModel` 变为对象，原字符串值被丢弃

### TC-02: set_model.sh — imageModel 为对象格式（正常路径）
- **场景**：openclaw.json 中 `imageModel` 是对象
- **预期**：行为与修复前一致

### TC-03: set_model.sh — imageModel 不存在
- **预期**：正常创建对象，行为与修复前一致

### TC-04: set_model.sh — imageprimary 为空
- **预期**：del imageModel（走原有 del 分支，不受类型判断影响）

### TC-05~06: switch_model.sh — 同 TC-01/02

### TC-07: remove_model_provider.sh — imageModel 为字符串格式
- **预期**：imageModel 字段被直接删除

### TC-08: remove_model_provider.sh — imageModel 为对象格式（正常路径）
- **预期**：走原逻辑，行为与修复前一致

### TC-09: remove_model_provider.sh — imageModel 不存在
- **预期**：不变

### 测试方式
构造测试 JSON 文件，直接运行 jq 表达式验证输出。

## 风险评估

| # | 风险 | 严重度 | 缓解措施 |
|---|------|-------|---------|
| 1 | 类型判断影响对象格式（正常路径）性能 | 极低 | jq `type` 函数是 O(1) 操作，对性能无影响 |
| 2 | 其他字段也有类似的字符串/对象兼容问题 | 低 | 已搜索确认仅 imageModel 存在此问题 |
| 3 | 修复后历史龙虾首次操作时 imageModel 字符串值丢失 | 低 | 字符串值是旧版残留，Go 侧 buildImageModelRefs 会重新计算正确值并覆盖 |
| 4 | jq 版本差异导致 `type` 函数行为不同 | 极低 | `type` 是 jq 标准内置函数，所有版本均支持 |
