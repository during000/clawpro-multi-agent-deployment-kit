package usergroup

import (
	"context"
	"strings"
	"testing"

	"hatchery/model"
)

// TestLandOneIDDepartmentsToGroups_SiblingsSameNameDifferentParents 覆盖本次 bug 的场景：
// OneID 有两组"同名但父不同"的部门（如 "运营组/一组" 与 "后台组/一组"），
// landing 必须把它们**全部**落地为两条独立的 user_groups 行，不能因 name 全局唯一索引
// 或者其他 silent continue 漏掉。
//
// 本测试用内存 SQLite（通过 AutoMigrate 建表，不会带上 0414 的 idx_ug_identifier_name
// 残留索引），所以创建应该全部成功；LandingFailures 必须为空。这个测试也是
// "现网上 DROP 残留索引后，landing 行为是否如预期"的正向验证。
func TestLandOneIDDepartmentsToGroups_SiblingsSameNameDifferentParents(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	// 还需要 OneIDDepartmentRecord 表
	if err := model.DB(context.Background()).AutoMigrate(&model.OneIDDepartmentRecord{}); err != nil {
		t.Fatalf("auto migrate oneid_departments: %v", err)
	}

	// 种部门：根(D0) → 运营组(D1)/后台组(D2) → 各自子部门"一组"/"二组"
	depts := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "D1", DepartmentName: "运营组", DepartmentParentID: "D0"},
		{DepartmentID: "D2", DepartmentName: "后台组", DepartmentParentID: "D0"},
		{DepartmentID: "D11", DepartmentName: "一组", DepartmentParentID: "D1"}, // 运营组/一组
		{DepartmentID: "D12", DepartmentName: "二组", DepartmentParentID: "D1"}, // 运营组/二组
		{DepartmentID: "D21", DepartmentName: "一组", DepartmentParentID: "D2"}, // 后台组/一组
		{DepartmentID: "D22", DepartmentName: "二组", DepartmentParentID: "D2"}, // 后台组/二组
	}
	for _, d := range depts {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}

	res, err := LandOneIDDepartmentsToGroups(context.Background())
	if err != nil {
		t.Fatalf("LandOneIDDepartmentsToGroups: %v", err)
	}

	if len(res.LandingFailures) != 0 {
		t.Fatalf("期望无 landing failure，实际 %d 条: %+v", len(res.LandingFailures), res.LandingFailures)
	}

	// 7 个部门 → 7 个本地组
	var oneidGroups []model.UserGroup
	if err := model.DB(context.Background()).Where("source = ?", model.GroupSourceOneIDDept).Find(&oneidGroups).Error; err != nil {
		t.Fatalf("list oneid_dept groups: %v", err)
	}
	if len(oneidGroups) != 7 {
		t.Fatalf("期望 7 个 oneid_dept 组，实际 %d: %+v", len(oneidGroups), oneidGroups)
	}

	// 校验两个"一组"确实落地到了不同父下
	byDeptID := map[string]model.UserGroup{}
	for _, g := range oneidGroups {
		byDeptID[g.SourceRef] = g
	}
	if byDeptID["D11"].ParentID == 0 || byDeptID["D21"].ParentID == 0 {
		t.Fatalf("同名【一组】落地父组丢失: D11=%+v D21=%+v", byDeptID["D11"], byDeptID["D21"])
	}
	if byDeptID["D11"].ParentID == byDeptID["D21"].ParentID {
		t.Fatalf("两个【一组】不应落在同一父下，实际都挂到 parent_id=%d",
			byDeptID["D11"].ParentID)
	}
	if byDeptID["D12"].ParentID == byDeptID["D22"].ParentID {
		t.Fatalf("两个【二组】不应落在同一父下")
	}
}

// TestLandOneIDDepartmentsToGroups_ReportsFailureOnStaleUniqueIndex 直接模拟
// 本次现网问题：在 user_groups 上保留老的 idx_ug_identifier_name 唯一索引，
// 触发同名子部门 landing 时冲突。修复前 silent continue 吞掉错误；修复后
// LandingFailures 必须至少有 1 条，且包含冲突 dept 的 ID。
//
// 这个测试是"失败可观测性"的回归保护：让 CI 能在未来有人再加类似静默 continue
// 时立刻发现。
func TestLandOneIDDepartmentsToGroups_ReportsFailureOnStaleUniqueIndex(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(&model.OneIDDepartmentRecord{}); err != nil {
		t.Fatalf("auto migrate oneid_departments: %v", err)
	}

	// 模拟老库：补建一条全局 (identifier, name) 唯一索引
	if err := model.DB(context.Background()).Exec(
		`CREATE UNIQUE INDEX idx_ug_identifier_name ON user_groups(identifier, name)`,
	).Error; err != nil {
		t.Fatalf("create stale unique index: %v", err)
	}

	// 种 4 个同名子部门（分别挂两个父）
	depts := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "D1", DepartmentName: "运营组", DepartmentParentID: "D0"},
		{DepartmentID: "D2", DepartmentName: "后台组", DepartmentParentID: "D0"},
		{DepartmentID: "D11", DepartmentName: "一组", DepartmentParentID: "D1"},
		{DepartmentID: "D21", DepartmentName: "一组", DepartmentParentID: "D2"}, // 与 D11 同名
	}
	for _, d := range depts {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}

	res, err := LandOneIDDepartmentsToGroups(context.Background())
	if err != nil {
		t.Fatalf("LandOneIDDepartmentsToGroups: %v", err)
	}

	// 修复前：len(LandingFailures)==0（silent continue）
	// 修复后：至少有 1 条冲突记录
	if len(res.LandingFailures) == 0 {
		t.Fatalf("期望 LandingFailures 至少有 1 条冲突，实际为空")
	}

	// 冲突必定是 D11/D21 中的一个（先处理的那个成功，后处理的那个冲突）
	hasExpected := false
	for _, f := range res.LandingFailures {
		if f.DepartmentID == "D11" || f.DepartmentID == "D21" {
			if f.Stage != "create" {
				t.Fatalf("期望 stage=create，实际 %s", f.Stage)
			}
			if !strings.Contains(strings.ToLower(f.Err), "unique") {
				t.Fatalf("期望 err 包含 unique 冲突信息，实际 %q", f.Err)
			}
			hasExpected = true
		}
	}
	if !hasExpected {
		t.Fatalf("LandingFailures 未包含 D11/D21 的冲突记录: %+v", res.LandingFailures)
	}
}

// TestLandOneIDDepartmentsToGroups_RenameRecomputesFullPath 覆盖 bug：
// OneID 侧改部门名后，landing 必须把本地 user_groups 的 name **和 full_path**
// 同时更新；子孙的 full_path 也要级联重算。
//
// 现网现象：运营组/二组 改名为 运营组/2组 后，name=2组 但 full_path 还是
// "OpenClaw企业版体验/运营组/二组"（停留在老值）。
// 根因：landing 在仅改名分支走了裸 DB.Updates({"name": ...})，绕过了
// recomputeSubtreeFullPathTx，没重算 full_path。
func TestLandOneIDDepartmentsToGroups_RenameRecomputesFullPath(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(&model.OneIDDepartmentRecord{}); err != nil {
		t.Fatalf("auto migrate oneid_departments: %v", err)
	}

	// 初始结构：总部/运营组/二组/小组
	initial := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "D1", DepartmentName: "运营组", DepartmentParentID: "D0"},
		{DepartmentID: "D12", DepartmentName: "二组", DepartmentParentID: "D1"},
		{DepartmentID: "D121", DepartmentName: "小组", DepartmentParentID: "D12"},
	}
	for _, d := range initial {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}

	// 第一轮 landing：建 4 个组
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("first landing: %v", err)
	}
	var g12 model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D12").First(&g12).Error; err != nil {
		t.Fatalf("find D12: %v", err)
	}
	if g12.FullPath != "总部/运营组/二组" {
		t.Fatalf("初始 full_path 错误: %q", g12.FullPath)
	}

	// 模拟 OneID 改名：D12 "二组" → "2组"
	if err := model.DB(context.Background()).Model(&model.OneIDDepartmentRecord{}).
		Where("department_id = ?", "D12").
		Update("department_name", "2组").Error; err != nil {
		t.Fatalf("rename dept: %v", err)
	}

	// 第二轮 landing：应触发改名 + full_path 重算
	res, err := LandOneIDDepartmentsToGroups(context.Background())
	if err != nil {
		t.Fatalf("second landing: %v", err)
	}
	if len(res.LandingFailures) != 0 {
		t.Fatalf("期望无 landing failure，实际 %+v", res.LandingFailures)
	}

	// 校验自身 name + full_path
	if err := model.DB(context.Background()).Where("source_ref = ?", "D12").First(&g12).Error; err != nil {
		t.Fatalf("re-find D12: %v", err)
	}
	if g12.Name != "2组" {
		t.Fatalf("name 未更新: %q", g12.Name)
	}
	if g12.FullPath != "总部/运营组/2组" {
		t.Fatalf("full_path 未重算: 期望 %q 实际 %q", "总部/运营组/2组", g12.FullPath)
	}

	// 校验子孙 full_path 级联重算
	var g121 model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D121").First(&g121).Error; err != nil {
		t.Fatalf("find D121: %v", err)
	}
	if g121.FullPath != "总部/运营组/2组/小组" {
		t.Fatalf("子孙 full_path 未级联重算: 期望 %q 实际 %q",
			"总部/运营组/2组/小组", g121.FullPath)
	}
}
