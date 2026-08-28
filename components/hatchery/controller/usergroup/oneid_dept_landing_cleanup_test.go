package usergroup

import (
	"context"
	"testing"

	"hatchery/model"
)

// TestLandOneIDDepartmentsToGroups_CleanupPerGroupDecision 覆盖 v6 语义变更：
// OneID 子树消失时，必须**按组逐个判断**删除/保留，而不是整子树 all-or-nothing。
//
// 还原用户描述的场景：
//
//	OneID 侧删除了"后台组"整棵子树：后台组(4) / 1组(9) / 2组(10) / 二组(12)
//	本地状态：
//	  - id=9  绑定 model_visibility_groups → CanDeleteUserGroup=false → 保留
//	  - id=10 绑定 group_config_bindings   → CanDeleteUserGroup=false → 保留
//	  - id=12 无绑定 + 无子孙              → CanDeleteUserGroup=true  → 物理删
//	  - id=4  本身无绑定，但子孙 9/10 保留 → 父级级联保留（否则 9/10 变孤儿）
//
// 期望：landing 后 4 / 9 / 10 仍在但 to_be_deleted=1；12 被物理删除（DB 无行）。
func TestLandOneIDDepartmentsToGroups_CleanupPerGroupDecision(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(
		&model.OneIDDepartmentRecord{},
		&model.ModelVisibilityGroup{},
		&model.SkillVisibilityGroup{},
		&model.SkillBundleVisibilityGroup{},
		&model.RoleVisibilityGroup{},
		&model.AIModel{},
		&model.AIChannel{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 第一轮：同步出完整组织结构（总部/后台组/{1组,2组,二组}）
	initial := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "D4", DepartmentName: "后台组", DepartmentParentID: "D0"},
		{DepartmentID: "D9", DepartmentName: "1组", DepartmentParentID: "D4"},
		{DepartmentID: "D10", DepartmentName: "2组", DepartmentParentID: "D4"},
		{DepartmentID: "D12", DepartmentName: "二组", DepartmentParentID: "D4"},
	}
	for _, d := range initial {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("first landing: %v", err)
	}

	// 拿到本地 user_groups id
	idBy := map[string]uint{}
	for _, sr := range []string{"D0", "D4", "D9", "D10", "D12"} {
		var g model.UserGroup
		if err := model.DB(context.Background()).Where("source_ref = ?", sr).First(&g).Error; err != nil {
			t.Fatalf("find %s: %v", sr, err)
		}
		idBy[sr] = g.ID
	}

	// 给 D9 加模型绑定（legacy 旧表），给 D10 加 group_config_bindings（通道）
	if err := model.DB(context.Background()).Create(&model.AIModel{Provider: "tc", ModelID: "m1"}).Error; err != nil {
		t.Fatalf("seed model: %v", err)
	}
	var m model.AIModel
	model.DB(context.Background()).First(&m)
	if err := model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: m.ID, GroupID: idBy["D9"]}).Error; err != nil {
		t.Fatalf("seed model_visibility_groups: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.GroupConfigBinding{
		GroupID: idBy["D10"], ConfigType: model.ConfigTypeChannel, ConfigKey: "5",
	}).Error; err != nil {
		t.Fatalf("seed group_config_bindings: %v", err)
	}

	// 模拟 OneID 侧删除后台组整棵子树：删掉 D4/D9/D10/D12 这 4 条
	if err := model.DB(context.Background()).Where("department_id IN ?",
		[]string{"D4", "D9", "D10", "D12"}).Delete(&model.OneIDDepartmentRecord{}).Error; err != nil {
		t.Fatalf("delete oneid depts: %v", err)
	}

	// 第二轮 landing：应进入 B+C 清理流程
	res, err := LandOneIDDepartmentsToGroups(context.Background())
	if err != nil {
		t.Fatalf("second landing: %v", err)
	}
	if len(res.LandingFailures) != 0 {
		t.Fatalf("期望无 landing failure，实际 %+v", res.LandingFailures)
	}

	// 断言 1：id=12 走到了物理删除流，而非被标 to_be_deleted 保留。
	// 当前实现 DeleteUserGroupForOneIDDept 已改为物理删除（user_groups 不再有
	// gorm.DeletedAt 字段）。关键区别：
	//   - 清理流走 DeleteUserGroupForOneIDDept → 默认 First 查不到 → Unscoped 也查不到
	//   - 保留流走 markOneIDSubtreeToBeDeleted → 默认 First 能查到 + ToBeDeleted=true
	var g12 model.UserGroup
	err12 := model.DB(context.Background()).Where("source_ref = ?", "D12").First(&g12).Error
	if err12 == nil {
		t.Fatalf("id=12 应走删除流（查不到），但 First 命中: %+v", g12)
	}
	var g12u model.UserGroup
	if err := model.DB(context.Background()).Unscoped().Where("source_ref = ?", "D12").First(&g12u).Error; err == nil {
		t.Fatalf("D12 应被物理删除（Unscoped 也找不到），但仍命中: %+v", g12u)
	}

	// 断言 2：id=4 / 9 / 10 仍在且 to_be_deleted=true
	for _, sr := range []string{"D4", "D9", "D10"} {
		var g model.UserGroup
		if err := model.DB(context.Background()).Where("source_ref = ?", sr).First(&g).Error; err != nil {
			t.Fatalf("%s 应保留，但查不到: %v", sr, err)
		}
		if !g.ToBeDeleted {
			t.Fatalf("%s 应被标 to_be_deleted=true，实际 false", sr)
		}
	}

	// 断言 3：NewlyMarkedToBeDeleted 包含 D4/D9/D10 的 full_path + 对应 group_id
	expected := map[string]bool{"总部/后台组": false, "总部/后台组/1组": false, "总部/后台组/2组": false}
	for _, item := range res.NewlyMarkedToBeDeleted {
		if _, ok := expected[item.FullPath]; ok {
			expected[item.FullPath] = true
			if item.GroupID == 0 {
				t.Fatalf("NewlyMarkedToBeDeleted 项 %q 的 group_id 不应为 0", item.FullPath)
			}
		}
	}
	for p, seen := range expected {
		if !seen {
			t.Fatalf("期望 NewlyMarkedToBeDeleted 包含 %q，实际 %+v", p, res.NewlyMarkedToBeDeleted)
		}
	}
}

// TestLandOneIDDepartmentsToGroups_CleanupAllDeletable 辅助回归：
// 子树里所有组都无绑定时，仍然按"叶子先"顺序全部物理删除。
func TestLandOneIDDepartmentsToGroups_CleanupAllDeletable(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(
		&model.OneIDDepartmentRecord{},
		&model.ModelVisibilityGroup{},
		&model.SkillVisibilityGroup{},
		&model.SkillBundleVisibilityGroup{},
		&model.RoleVisibilityGroup{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	initial := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "D1", DepartmentName: "A组", DepartmentParentID: "D0"},
		{DepartmentID: "D11", DepartmentName: "A1", DepartmentParentID: "D1"},
		{DepartmentID: "D12", DepartmentName: "A2", DepartmentParentID: "D1"},
	}
	for _, d := range initial {
		model.DB(context.Background()).Create(&d)
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("first landing: %v", err)
	}

	// OneID 删 A 组子树（D1/D11/D12 一起）
	model.DB(context.Background()).Where("department_id IN ?", []string{"D1", "D11", "D12"}).
		Delete(&model.OneIDDepartmentRecord{})

	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("second landing: %v", err)
	}

	// 三个都应走物理删除流（默认 First 查不到 + Unscoped 也查不到）
	for _, sr := range []string{"D1", "D11", "D12"} {
		var g model.UserGroup
		if err := model.DB(context.Background()).Where("source_ref = ?", sr).First(&g).Error; err == nil {
			t.Fatalf("%s 应走删除流，但默认 First 命中: %+v", sr, g)
		}
		var gu model.UserGroup
		if err := model.DB(context.Background()).Unscoped().Where("source_ref = ?", sr).First(&gu).Error; err == nil {
			t.Fatalf("%s 应被物理删除（Unscoped 也找不到），但仍命中: %+v", sr, gu)
		}
	}
	// D0 仍在（不在消失集）
	var d0 model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "D0").First(&d0).Error; err != nil {
		t.Fatalf("D0 不应被动: %v", err)
	}
	if d0.ToBeDeleted {
		t.Fatalf("D0 不应被标 to_be_deleted")
	}
}

// TestLandOneIDDepartmentsToGroups_RetryAfterUnbinding 覆盖 bug：
// 用户解绑配置后，再次同步应把此前标了 to_be_deleted 的组真正清理掉。
//
// 原 bug：missingRoots 过滤条件 "!g.ToBeDeleted" 把已标记的组排除在清理集合外，
// 一旦第一次被标 to_be_deleted 就永远不会被再评估 → 解绑后再同步也不会删。
//
// 期望流程：
//  1. 第一轮：OneID 侧删除组 X（X 有绑定）→ X.ToBeDeleted=true
//  2. 用户解绑 X 的所有绑定
//  3. 第二轮 landing → 重新评估 X → CanDelete=true → 软删
func TestLandOneIDDepartmentsToGroups_RetryAfterUnbinding(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(
		&model.OneIDDepartmentRecord{},
		&model.ModelVisibilityGroup{},
		&model.SkillVisibilityGroup{},
		&model.SkillBundleVisibilityGroup{},
		&model.RoleVisibilityGroup{},
		&model.AIModel{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 初始：总部 / X组
	initial := []model.OneIDDepartmentRecord{
		{DepartmentID: "D0", DepartmentName: "总部", DepartmentParentID: ""},
		{DepartmentID: "DX", DepartmentName: "X组", DepartmentParentID: "D0"},
	}
	for _, d := range initial {
		model.DB(context.Background()).Create(&d)
	}
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("first landing: %v", err)
	}
	var gx model.UserGroup
	if err := model.DB(context.Background()).Where("source_ref = ?", "DX").First(&gx).Error; err != nil {
		t.Fatalf("find DX: %v", err)
	}

	// 给 X 加模型绑定
	model.DB(context.Background()).Create(&model.AIModel{Provider: "tc", ModelID: "m1"})
	var m model.AIModel
	model.DB(context.Background()).First(&m)
	if err := model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: m.ID, GroupID: gx.ID}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	// OneID 删除 X
	model.DB(context.Background()).Where("department_id = ?", "DX").Delete(&model.OneIDDepartmentRecord{})

	// 第二轮 landing → X 不可删 → to_be_deleted=1
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("second landing: %v", err)
	}
	model.DB(context.Background()).Where("source_ref = ?", "DX").First(&gx)
	if !gx.ToBeDeleted {
		t.Fatalf("期望 X 被标 to_be_deleted，实际 %+v", gx)
	}

	// 用户解绑
	if err := model.DB(context.Background()).Where("ai_model_id = ? AND group_id = ?", m.ID, gx.ID).
		Delete(&model.ModelVisibilityGroup{}).Error; err != nil {
		t.Fatalf("unbind: %v", err)
	}

	// 第三轮 landing → X 应被真正清理
	if _, err := LandOneIDDepartmentsToGroups(context.Background()); err != nil {
		t.Fatalf("third landing: %v", err)
	}
	if err := model.DB(context.Background()).Where("source_ref = ?", "DX").First(&gx).Error; err == nil {
		t.Fatalf("解绑后第三次同步应清理 X，但 First 仍命中: %+v", gx)
	}
	var gxU model.UserGroup
	if err := model.DB(context.Background()).Unscoped().Where("source_ref = ?", "DX").First(&gxU).Error; err == nil {
		t.Fatalf("X 应被物理删除（Unscoped 也找不到），但仍命中: %+v", gxU)
	}
}
