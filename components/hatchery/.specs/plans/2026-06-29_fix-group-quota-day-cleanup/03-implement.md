# 03. Implement — 实现细节

> 与 Plan 一致，记录实际实现的关键细节与验证结果。

---

## 实际改动

### 1. `controller/usergroup/types.go` — 新增 `QuotaPolicyPairedKey`

在 `IsValidPolicyKey` 之后插入纯函数，返回配额 key 的配对 key，非配额 key 返回空串。

> 注意：原文件存在历史 gofmt 对齐不一致（`const` 块与 `PolicyDefs` map），`gofmt -w` 会顺带重排这些块产生大量噪声 diff。为保持最小改动，**未对 types.go 整体 gofmt**，仅手工保证新增函数本身 gofmt-clean。CI 若强制 `gofmt -l` 零输出，需单独处理该历史问题（不在本任务范围）。

### 2. `controller/admin_group_config.go`

**`HandleSetGroupPolicy` 写 `*_rules` 路径**（517-534 行附近）：normalize 后改为事务内 `SetPolicy(rules)` + `DeletePolicy(配对 day)`，与既有写 day 路径对称。`DeletePolicy` 走 GORM `Where().Delete()`，0 行不报错，幂等安全。

**`HandleDeleteGroupPolicy`**（653-662 行附近）：目标 key binding 不存在时，若 `QuotaPolicyPairedKey(req.ConfigKey) != ""`，查配对 key binding；存在则改删配对 key，配对也无则返回 422。非配额 key 走原逻辑直接 422。

### 3. `controller/admin_group_config_coverage_test.go`

新增 `countPolicyBindings` 辅助函数 + 9 个测试用例（与 Plan 测试用例表一一对应）。

---

## 与 Plan 差异

无实质差异。唯一调整：types.go 不整体 gofmt（避免历史对齐噪声进入本 PR），仅保证新增函数 gofmt-clean。

---

## 验证结果

| 项 | 命令 | 结果 |
|----|------|------|
| 构建 | `go build ./controller/...` | ✅ 通过 |
| 静态检查 | `go vet ./controller/` | ✅ 无错误 |
| 目标测试 | `go test ./controller/ -run "TestCoverage(Set\|Delete)GroupPolicy" -race` | ✅ 全部通过（含 9 个新用例 + 既有 13 个用例） |
| usergroup 包 | `go test ./controller/usergroup/ -race` | ✅ 通过 |
| gofmt | `gofmt -w` 作用于 admin_group_config.go、admin_group_config_coverage_test.go | ✅ 干净；types.go 因历史对齐问题未整体格式化（见上） |

---

## 预存在失败（与本改动无关）

`go test ./controller/ -race` 全量有 23 个测试失败（`TestWithAudit_PostPath`、`TestCommon_Admin_Success_JSON`、`TestHandleRetryFailedSkills` 等），均为 audit/channel/instance/role 模块的 **race condition 或环境问题**。

**验证方式**：`git stash` 本改动后在 clean origin/master 上复现相同失败，确认是预存在问题。失败堆栈指向 `controller/audit.go:270` 的 `WithAudit` goroutine 与 `model.LogAudit` → `model.DB()` 全局状态竞争，与本任务的 group-config quota 逻辑无任何代码路径交集。

UT 阶段将以目标测试 + 受影响包为准，不把这 23 个预存在失败计入本任务回归。
