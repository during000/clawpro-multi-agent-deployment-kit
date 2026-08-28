# Code Review

## 审查范围

| 文件 | 改动 |
|------|------|
| `scripts/set_model.sh` | +4 行（类型判断 + 重置为空对象） |
| `scripts/switch_model.sh` | +4 行（同上） |
| `scripts/remove_model_provider.sh` | +6/-8 行（重构为 if/elif/else） |
| `test/scripts/test_imagemodel_compat.sh` | 新增测试脚本（10 个用例） |

## 审查维度

### 1. 规范性 ✅
- 脚本缩进与原有代码一致（set_model 8 空格、switch_model 3 空格、remove 4 空格）
- 注释清晰，说明兼容旧格式的原因
- jq 表达式风格与原有一致（if/then/else/end）

### 2. 安全性 ✅
- 无 shell 注入风险（jq 表达式内不涉及用户输入拼接）
- 不改变文件锁（flock）机制
- 不改变备份（cp .bak）机制
- 不改变原子替换（mv tmpfile）机制

### 3. 性能 ✅
- jq `type` 函数是 O(1) 操作，无性能影响
- 仅在 imageModel 为字符串时执行额外赋值（`{}`），对象格式走原路径

### 4. 兼容性 ✅
- 向后兼容：对象格式（新）行为完全不变
- 向前兼容：字符串格式（旧）被正确处理
- null/不存在：走 `else .`（不变），与原有行为一致

### 5. 逻辑正确性 ✅
- set_model/switch_model：字符串 → 重置 `{}` → 写入 primary/fallbacks，逻辑正确
- remove_model_provider：字符串 → 直接 del（合理，旧格式残留无需保留），逻辑正确
- remove TC-08b 验证了 primary 不匹配时仅过滤 fallbacks，保留 imageModel，逻辑正确

## 问题与修复

无问题。改动通过全部审查维度。

## 审查结论

**通过**。改动范围小、逻辑清晰、测试覆盖完整，可提交。
