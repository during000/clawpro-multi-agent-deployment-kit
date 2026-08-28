# 需求澄清：修复 imageModel 字符串格式不兼容

## 背景

### 问题描述

TAPD 缺陷单：[1020422209160822747](https://tapd.woa.com/tapd_fe/20422209/bug/detail/1020422209160822747)

**优先级**：高优需要紧急修复

客户历史创建的龙虾（OpenClaw 实例）添加模型报错。

**根因**：历史创建的龙虾 `~/.openclaw/openclaw.json` 中 `agents.defaults.imageModel` 字段是**字符串**格式（旧版）：

```json
"imageModel": "hatchery-glm-4-plus/glm-4-plus"
```

而最新代码将其作为**对象**格式处理（新版）：

```json
"imageModel": { "primary": "hatchery-glm-4-plus/glm-4-plus", "fallbacks": [] }
```

三个脚本（`set_model.sh`、`switch_model.sh`、`remove_model_provider.sh`）中的 jq 操作假设 `imageModel` 是对象，直接访问 `.primary` / `.fallbacks`。当实际是字符串时，jq 报类型错误 `Cannot index string with "primary"`，脚本中止执行，导致添加模型失败。

### 受影响脚本与具体位置

| 脚本 | 位置 | 问题代码 |
|------|------|---------|
| `set_model.sh` | 第 195-200 行 jq | `.agents.defaults.imageModel.primary = $imageprimary` — 字符串上报错 |
| `switch_model.sh` | 第 142-147 行 jq | 同上 |
| `remove_model_provider.sh` | 第 63-74 行 jq | `.agents.defaults.imageModel.fallbacks` / `.primary` — 字符串上访问属性报错 |

### 影响范围

- **触发条件**：历史创建的龙虾（imageModel 为字符串格式）执行添加/切换/删除模型操作
- **影响**：所有模型操作失败，用户无法管理模型
- **处理人**：royleiyang；**责任人**：xiankeli

## 目标

修复三个脚本，使其兼容 `imageModel` 字段的字符串格式（旧格式）和对象格式（新格式），确保历史龙虾能正常执行模型操作。

## 范围

### 做

- 修改 `scripts/set_model.sh`：jq 中设置 `imageModel.primary` 前检查类型，字符串格式先重置为空对象 `{}`
- 修改 `scripts/switch_model.sh`：同上
- 修改 `scripts/remove_model_provider.sh`：jq 中访问 `imageModel.fallbacks` / `.primary` 前检查类型，字符串格式直接 `del`

### 不做

- 不修改 Go 代码（Go 代码只生成脚本参数，不直接操作 openclaw.json 中的 imageModel）
- 不做数据迁移（不在 Go 侧主动将旧格式 imageModel 转为新格式）
- 不修改 openclaw.json 的数据结构定义

## 待确认问题及结论

### Q1: 是否需要在 Go 侧做数据迁移，主动将历史龙虾的字符串格式 imageModel 转为对象格式？

**结论**：不做。原因：
1. openclaw.json 存储在 CVM 实例本地，Go 侧无法直接修改（只能通过 TAT 脚本下发）
2. 脚本兼容两种格式更安全，且改动最小
3. 字符串格式的 imageModel 在下次模型操作时会被脚本自动转换为对象格式

### Q2: 字符串格式的 imageModel 是否需要保留其值？

**结论**：不保留。原因：
1. 字符串格式的 imageModel 是旧版残留，其值可能已过时
2. 脚本每次执行时会由 Go 侧 `buildImageModelRefs` 重新计算 primary 和 fallbacks
3. set_model.sh / switch_model.sh 会用新值覆盖 imageModel
4. remove_model_provider.sh 中字符串格式直接 del 即可（后续 add-model 会重建）

### Q3: 是否有其他脚本也访问 imageModel？

**结论**：已搜索确认，仅这三个脚本访问 `agents.defaults.imageModel`。其他脚本（如 backup/restore 相关）不直接操作该字段。

## 约束与依赖

- **兼容性**：必须同时支持字符串格式（旧）和对象格式（新），不能只支持新格式
- **幂等性**：修复后的脚本对已经是对象格式的 imageModel 行为不变
- **安全性**：jq 类型判断不影响正常路径（对象格式）的执行
- **TAPD 关联**：bug ID 1020422209160822747
