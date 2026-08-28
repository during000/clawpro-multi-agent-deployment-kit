package usergroup

import (
	"context"
	"testing"

	"hatchery/model"
)

// TestLandOneIDDepartmentsToGroups_CLSScopeSetInit 覆盖 clsScopeSet 初始化路径（291-292行）：
// 当 GetCLSCollectScopeGroupIDs 成功返回非空列表时，clsScopeSet 被正确初始化。
func TestLandOneIDDepartmentsToGroups_CLSScopeSetInit(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(
		&model.OneIDDepartmentRecord{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 第一轮：同步出组织结构
	initial := []model.OneIDDepartmentRecord{
		{DepartmentID: "D1", DepartmentName: "研发部", DepartmentParentID: ""},
		{DepartmentID: "D2", DepartmentName: "后端组", DepartmentParentID: "D1"},
	}
	for _, d := range initial {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("first landing: %v", err)
	}

	// 查询 D2 的本地分组 ID
	var g2 model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D2").First(&g2).Error; err != nil {
		t.Fatalf("find D2: %v", err)
	}

	// 在 CLS scope 中添加 D2 的分组（触发 clsScopeSet 初始化路径）
	if err := model.SetCLSCollectScope(context.Background(), []uint{g2.ID}); err != nil {
		t.Fatalf("set cls scope: %v", err)
	}

	// 第二轮：D1/D2 仍在 OneID，不触发清理，但 clsScopeSet 会被初始化
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("second landing: %v", err)
	}

	// 验证 CLS scope 仍然存在
	scopeIDs, err := model.GetCLSCollectScopeGroupIDs(context.Background())
	if err != nil {
		t.Fatalf("get cls scope: %v", err)
	}
	if len(scopeIDs) != 1 || scopeIDs[0] != g2.ID {
		t.Errorf("期望 CLS scope 包含 D2 的分组 ID %d，实际 %v", g2.ID, scopeIDs)
	}
}

// TestLandOneIDDepartmentsToGroups_CLSScopeOverlapWarning 覆盖 CLS scope 告警路径（369-379行）：
// 当标记待删的 OneID 组仍在 CLS 采集范围中时，触发告警日志。
//
// 场景：
//   - D1（研发部）有绑定（CanDeleteUserGroup=false），被标记为 to_be_deleted
//   - D1 的分组在 CLS scope 中
//   - 运行 landing 后，触发 CLS scope 告警
func TestLandOneIDDepartmentsToGroups_CLSScopeOverlapWarning(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(
		&model.OneIDDepartmentRecord{},
		&model.ModelVisibilityGroup{},
		&model.AIModel{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 第一轮：同步出组织结构（只有 D1 一个根组）
	initial := []model.OneIDDepartmentRecord{
		{DepartmentID: "D1", DepartmentName: "研发部", DepartmentParentID: ""},
	}
	for _, d := range initial {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("first landing: %v", err)
	}

	// 查询 D1 的本地分组 ID
	var g1 model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D1").First(&g1).Error; err != nil {
		t.Fatalf("find D1: %v", err)
	}

	// 给 D1 加模型绑定（使 CanDeleteUserGroup=false，D1 会被标记为 to_be_deleted）
	if err := model.DB(context.Background()).Create(&model.AIModel{Provider: "tc", ModelID: "m1"}).Error; err != nil {
		t.Fatalf("seed model: %v", err)
	}
	var m model.AIModel
	model.DB(context.Background()).First(&m)
	if err := model.DB(context.Background()).Create(&model.ModelVisibilityGroup{
		AIModelID: m.ID, GroupID: g1.ID,
	}).Error; err != nil {
		t.Fatalf("seed model_visibility_groups: %v", err)
	}

	// 在 CLS scope 中添加 D1 的分组（触发 CLS scope 告警路径）
	if err := model.SetCLSCollectScope(context.Background(), []uint{g1.ID}); err != nil {
		t.Fatalf("set cls scope: %v", err)
	}

	// 删除 D1 的 OneID 部门记录（模拟 OneID 侧删除）
	if err := model.DB(context.Background()).Where("department_id = ?", "D1").
		Delete(&model.OneIDDepartmentRecord{}).Error; err != nil {
		t.Fatalf("delete oneid dept: %v", err)
	}

	// 第二轮 landing：D1 因有绑定无法物理删除，被标记为 to_be_deleted
	// 同时 D1 在 CLS scope 中，触发告警日志
	res, err := LandOneIDDepartmentsToGroups(context.Background())
	if err != nil {
		t.Fatalf("second landing: %v", err)
	}

	// 验证 D1 被标记为 to_be_deleted
	var g1After model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D1").First(&g1After).Error; err != nil {
		t.Fatalf("find D1 after: %v", err)
	}
	if !g1After.ToBeDeleted {
		t.Errorf("D1 应被标记为 to_be_deleted=true，实际 false")
	}

	// 验证 NewlyMarkedToBeDeleted 包含 D1
	found := false
	for _, item := range res.NewlyMarkedToBeDeleted {
		if item.GroupID == g1.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("NewlyMarkedToBeDeleted 应包含 D1 的分组 ID %d，实际 %v", g1.ID, res.NewlyMarkedToBeDeleted)
	}
}
