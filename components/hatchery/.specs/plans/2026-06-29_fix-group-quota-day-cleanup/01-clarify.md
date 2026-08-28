# 01. Clarify — 需求澄清

> AI 以产品经理角色进行 Discovery + Challenge，确保需求清晰、边界明确。

---

## 背景

分组级 Token 配额存在新旧两套字段：

- 旧：`token_quota_day` / `global_token_quota_day`（int，单值）
- 新：`token_quota_rules` / `global_token_quota_rules`（JSON，多模式规则）

读路径（`HandleGroupConfigGroups`、resolve 层、overview）已做完整双向兼容：查 `*_rules` 时若该组仅有 `*_day` binding，会从 day 反推 rules 返回；查 `*_day` 时若仅有 `*_rules`，会从 rules 反推 day 返回。

写路径存在**不对称缺陷**，导致前端出现"旧字段删不掉、删完刷新又复活"的循环：

1. **`HandleSetGroupPolicy` 写 `*_rules`**（controller/admin_group_config.go:518-534）：只写 rules binding，**不删配对 `*_day` binding**。而写 `*_day` 路径（537-603）会事务内迁移到 rules 并删除 day binding。两者不对称。
2. **`HandleDeleteGroupPolicy`**（controller/admin_group_config.go:619-670）：只删指定 key 的 binding，**不级联删配对 key**；且查询阶段无 day→rules 兼容反推，导致"该组仅有 `*_day`、无 `*_rules` binding"时，前端删除 `*_rules` 会被后端以"binding 不存在"拒绝。

### 复现链路（已与用户确认）

组 G 仅有 `token_quota_day` binding、无 `token_quota_rules` binding：

1. 刷新页面 → 后端从 day 反推出 rules → 前端显示"规则"（旧字段被显示）。
2. 点删除 → 前端发 `delete config_key=token_quota_rules` → 后端查 rules binding 为空 → 报错"不存在" → 删不掉。
3. 先设置一次 rules → G 同时挂 day + rules 两个 binding（写 rules 未清理 day）。
4. 删除 rules → 只删了 rules，day 仍在。
5. 刷新 → 后端又从 day 反推出 rules → 旧字段复活。

## 目标

让"管理 `*_rules`"在分组策略层面完全等价于"管理 `*_day`"，消除上述循环：

- [ ] 设置某组 `*_rules` 时，清理配对 `*_day` binding（与写 `*_day` 路径对称）。
- [ ] 删除某组 `*_rules` 时，若该 key binding 不存在，尝试删除配对 `*_day` binding（用户确认的方案）。
- [ ] 删除某组 `*_day` 时，若该 key binding 不存在，尝试删除配对 `*_rules` binding（同理对称）。
- [ ] 上述两处改动对 `token_quota_*` 与 `global_token_quota_*` 两对 key 均生效。
- [ ] 前端无需改动（前端已统一只管理 `*_rules`）。

## 范围

| 包含 | 不包含 |
|------|--------|
| `controller/admin_group_config.go` 的 `HandleSetGroupPolicy` 写 rules 路径 | 用户级（`/admin/users`）配额写路径（已正确清理 day） |
| `controller/admin_group_config.go` 的 `HandleDeleteGroupPolicy` | 站点级（`/admin/config`）配额写路径（已正确清理 day） |
| 4 个配额 policy key：`token_quota_day`/`token_quota_rules`/`global_token_quota_day`/`global_token_quota_rules` | 读路径兼容（已完整，不动） |
| 单元测试覆盖新增的清理/级联分支 | `HandleGroupConfigGroups` 查询兼容反推（已存在，不动） |

## 待确认问题

| # | 问题 | 状态 | 结论 |
|---|------|------|------|
| 1 | 删除路径采用"目标 key 不存在时尝试删配对 key"（而非"总是同时删两个 key"），对吗？ | 已确认 | 是。用户明确选择此更保守方案：仅在目标 key 不存在时 fallback 到配对 key，避免改变"删 A 只删 A"的语义基准。 |
| 2 | 设置 `*_rules` 时清理配对 `*_day`，是否同样采用事务（与写 day 路径一致）？ | 已确认 | 是。事务内 SetPolicy(rules) + DeletePolicy(day)，保证原子性。 |
| 3 | 删除"目标 key 不存在 → 删配对 key"是否也要事务？ | 待 Plan 阶段定 | 倾向事务：先查目标 key，不存在则查配对 key，存在则事务删除。Plan 阶段细化。 |

## 约束与依赖

- 不能破坏现有读路径兼容（`HandleGroupConfigGroups` 的 4 个反推 block 不动）。
- 不能破坏运行时 resolve 优先级（`rules > day > fallback`）。
- 遵循项目红线：GORM 接口、事务、i18n、审计（写接口已用 `WithAudit` 包装，无需新增）。
- 前端 `openclaw-enterprise-fronted` 无需改动，仅依赖后端行为修正。
