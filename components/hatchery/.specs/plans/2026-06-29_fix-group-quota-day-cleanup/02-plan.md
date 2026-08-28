# 02. Plan — 方案设计

> 基于 01-clarify.md 确认的范围与方案，设计具体改动、调用链、测试用例与风险。

---

## 改动文件

| 文件 | 改动 |
|------|------|
| `controller/usergroup/types.go` | 新增 `QuotaPolicyPairedKey(key) string` 辅助函数，返回配额 key 的配对 key |
| `controller/admin_group_config.go` | ① `HandleSetGroupPolicy` 写 `*_rules` 路径改为事务：SetPolicy(rules) + DeletePolicy(配对 day)；② `HandleDeleteGroupPolicy` 目标 key 不存在时 fallback 删配对 key |
| `controller/admin_group_config_coverage_test.go` | 新增 6 个测试用例覆盖新增分支（见下方测试用例表） |

**不改动**：`HandleGroupConfigGroups` 查询兼容反推（4 个 block，已正确）、resolve 层、overview 兼容层、前端。

---

## 核心设计

### 辅助函数 `QuotaPolicyPairedKey`

放在 `controller/usergroup/types.go`，紧挨 `policyKeyOrder` 定义之后：

```go
// QuotaPolicyPairedKey 返回配额策略 key 的配对 key（legacy day ↔ new rules）。
// 非配额 key 返回空串。
func QuotaPolicyPairedKey(key string) string {
    switch key {
    case PolicyKeyTokenQuotaDay:
        return PolicyKeyTokenQuotaRules
    case PolicyKeyTokenQuotaRules:
        return PolicyKeyTokenQuotaDay
    case PolicyKeyGlobalTokenQuotaDay:
        return PolicyKeyGlobalTokenQuotaRules
    case PolicyKeyGlobalTokenQuotaRules:
        return PolicyKeyGlobalTokenQuotaDay
    }
    return ""
}
```

### 改动 1：`HandleSetGroupPolicy` 写 rules 路径（admin_group_config.go:517-534）

**现状**：normalize 后直接 `SetPolicy` 非事务写入，不清理配对 day binding。

**改后**：normalize 后用事务写入 rules + 幂等删除配对 day binding。镜像现有写 day 路径（537-603）的事务结构。

```go
if req.ConfigKey == usergroup.PolicyKeyTokenQuotaRules || req.ConfigKey == usergroup.PolicyKeyGlobalTokenQuotaRules {
    // ... 既有 normalize 逻辑不变，产出 valueJSON ...

    // 事务：写 rules + 清理配对 day binding（与写 day 路径对称）
    tx := model.DB(r.Context()).Begin()
    if err := usergroup.SetPolicy(tx, req.GroupID, req.ConfigKey, valueJSON); err != nil {
        tx.Rollback()
        writeError(w, r, http.StatusInternalServerError, hcommon.I18nError(i18n.MsgDatabaseOperationFailed))
        return
    }
    if pairedKey := usergroup.QuotaPolicyPairedKey(req.ConfigKey); pairedKey != "" {
        if err := usergroup.DeletePolicy(tx, req.GroupID, pairedKey); err != nil {
            tx.Rollback()
            writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
            return
        }
    }
    tx.Commit()
    jsonOK(w, map[string]interface{}{"ok": true})
    return
}
```

**幂等性**：`DeletePolicyBinding` 用 GORM `Where().Delete()`（model/group_config_binding.go:111-114），0 行受影响不报错，安全。

### 改动 2：`HandleDeleteGroupPolicy` fallback 删配对 key（admin_group_config.go:653-662）

**现状**：目标 key binding 不存在直接返回 422。

**改后**：目标 key 不存在时，若该 key 是配额类 key（有配对 key），尝试查配对 key binding；配对存在则删配对，配对也不存在才返回 422。

```go
// 校验绑定是否存在
configKeyToDelete := req.ConfigKey
bindings, err := model.GetPolicyBindingsByGroups(r.Context(), []uint{req.GroupID}, req.ConfigKey)
if err != nil {
    writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(err, i18n.MsgOperationFailed))
    return
}
if len(bindings) == 0 {
    // 兼容：目标 key 不存在时，尝试删除配对 key（仅对配额类 key 生效）
    if pairedKey := usergroup.QuotaPolicyPairedKey(req.ConfigKey); pairedKey != "" {
        pairedBindings, perr := model.GetPolicyBindingsByGroups(r.Context(), []uint{req.GroupID}, pairedKey)
        if perr != nil {
            writeError(w, r, http.StatusInternalServerError, hcommon.I18nRichError(perr, i18n.MsgOperationFailed))
            return
        }
        if len(pairedBindings) > 0 {
            configKeyToDelete = pairedKey
        } else {
            writeError(w, r, http.StatusUnprocessableEntity, hcommon.I18nError(i18n.MsgPolicyNotConfigured))
            return
        }
    } else {
        writeError(w, r, http.StatusUnprocessableEntity, hcommon.I18nError(i18n.MsgPolicyNotConfigured))
        return
    }
}

if err := usergroup.DeletePolicy(model.DB(r.Context()), req.GroupID, configKeyToDelete); err != nil {
    writeError(w, r, http.StatusInternalServerError, hcommon.EnsureRichErrorOrPanic(err))
    return
}
jsonOK(w, map[string]interface{}{"ok": true})
```

**为什么不事务**：删除只涉及单条 binding，无需事务。fallback 查询与删除之间窗口极小且幂等，不影响正确性。

---

## 调用链影响

```
HandleSetGroupPolicy(config_key=token_quota_rules)
  └─ tx.Begin
     ├─ SetPolicy(tx, rules)        # 写 rules binding
     └─ DeletePolicy(tx, day)       # 幂等删 day binding（新增）
  └─ tx.Commit

HandleDeleteGroupPolicy(config_key=token_quota_rules)
  ├─ GetPolicyBindingsByGroups(rules)
  ├─ 若空 → GetPolicyBindingsByGroups(day)   # 新增 fallback
  │         ├─ 有 → configKeyToDelete = day
  │         └─ 无 → 422
  └─ DeletePolicy(configKeyToDelete)
```

`global_token_quota_rules` / `global_token_quota_day` 同理对称。

---

## 测试用例（自然语言，先于实现）

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 设置 token_quota_rules，组原有 token_quota_day binding | POST set rules，组预置 day binding | 200；rules binding 写入；day binding 被删除 | P0 |
| 2 | 设置 global_token_quota_rules，组原有 global_token_quota_day binding | POST set global rules，组预置 global day binding | 200；global rules 写入；global day 删除 | P0 |
| 3 | 设置 token_quota_rules，组无任何配额 binding | POST set rules | 200；rules 写入；无 day binding（DeletePolicy 幂等不报错） | P0 |
| 4 | 删除 token_quota_rules，组仅有 token_quota_day binding | POST delete rules，组预置 day | 200；day binding 被删除（fallback 生效） | P0 |
| 5 | 删除 global_token_quota_rules，组仅有 global_token_quota_day binding | POST delete global rules，组预置 global day | 200；global day 被删除 | P0 |
| 6 | 删除 token_quota_day，组仅有 token_quota_rules binding | POST delete day，组预置 rules | 200；rules 被删除（反向 fallback） | P0 |
| 7 | 删除 token_quota_rules，组既无 rules 也无 day | POST delete rules | 422 MsgPolicyNotConfigured | P0 |
| 8 | 删除非配额 key（instance_quota）不存在时 | POST delete instance_quota，无 binding | 422，不触发 fallback | P0 |
| 9 | 删除 token_quota_rules，组同时有 rules 和 day（legacy 脏数据） | POST delete rules | 200；仅删 rules，day 保留（符合"目标存在则只删目标"语义） | P1 |

**用例 9 说明**：legacy 脏数据（两个 binding 共存）场景下，单次 delete rules 只删 rules，day 保留。这是用户确认的"目标 key 不存在时才 fallback"语义的直接结果。该场景在本修复后不会新增（Set rules 已清理 day），仅影响历史脏数据，用户再次 delete rules（此时目标不存在）会 fallback 清掉 day。若用户希望一次删除同时清两个，需调整方案——在 Plan 评审时确认。

---

## 风险评估

| # | 风险 | 严重度 | 缓解 |
|---|------|-------|------|
| 1 | 既有测试 `TestCoverageDeleteGroupPolicy_BindingNotExist` 期望 422 | 低 | 该测试组无任何 binding，fallback 查配对也为空 → 仍 422，不受影响 |
| 2 | 既有测试 `TestCoverageSetGroupPolicy_Success` 用 day key | 低 | 走 day 迁移路径（未改），不受影响 |
| 3 | legacy 脏数据（rules+day 共存）一次 delete 不清 day | 中 | 符合用户确认的语义；Set rules 已防新增脏数据；历史脏数据二次 delete 可清。评审时确认是否接受 |
| 4 | 事务失败回滚一致性 | 低 | 沿用既有 day 路径的事务模式（Begin/SetPolicy/DeletePolicy/Commit + Rollback） |
| 5 | 非配额 key 误触发 fallback | 低 | `QuotaPolicyPairedKey` 对非配额 key 返回空串，分支不进入 |

---

## 验收标准

- [ ] 9 个测试用例全部通过
- [ ] `go test ./controller/ -run "TestCoverage(Set|Delete)GroupPolicy" -v -race` 全绿
- [ ] `gofmt -w . && go vet ./...` 无错误
- [ ] 新增/修改函数覆盖率 >= 80%
- [ ] 不破坏既有 `TestCoverageDeleteGroupPolicy_*` 与 `TestCoverageSetGroupPolicy_*` 用例
