# 03. Implement — 实现

---

## 改动文件

### 1. `controller/admin_skills.go`（第 2001-2005 行）✅ 已提交

**修改后**：`Joins` → `Where + 子查询`

### 2. `controller/admin_skills_instances_test.go` ✅ 已提交

新增 `TestHandleAdminSkillInstances_GroupIDFilter_MultiGroupNoDuplicate`

### 3. `controller/admin_mcp_instances.go`（第 108-112 行）🔧 已提交

Joins → Where + 子查询。影响：`GET /admin/mcp/instances?group_id=...`

### 4. `controller/admin_skill_distribution.go`（第 1304-1306 行）🔧 已提交

Joins → Where + 子查询。影响：`POST /admin/skills/instances`（public skillset）

### 5. `controller/admin_mcp_test.go`（新增测试）🔧 已提交

新增 `TestHandleAdminMcpInstances_GroupIDFilter_MultiGroupNoDuplicate`

### 6. `controller/admin_skills_uninstall_test.go`（新增测试）🔧 已提交

新增 `TestHandleAdminSkillInstances_PublicSkillset_GroupIDFilter_MultiGroupNoDuplicate`

## 与 Plan 差异

补充了 Plan 中未覆盖的 2 处修复 + 2 个测试。
