// oneid_dept_landing_depth_cap_test.go
//
// 回归：OneID 上游组织结构超过 MaxGroupDepth（10 层）时：
//  - 第 1..10 层（depth 0..9）正常落地
//  - 第 11 层（depth=10）create 失败 → 记 LandingFailures(Stage="create")
//  - 第 12 层（parent=D10，但 D10 自己 create 失败）→ **必须** 连锁跳过，
//    记 LandingFailures(Stage="skipped_due_to_parent")，不能错挂到根

package usergroup

import (
	"context"
	"testing"

	"hatchery/model"
)

// TestLandOneIDDepartmentsToGroups_DepthOverLimit_CascadesSkip
// 构造 12 层 OneID 部门链 D0..D11（父指针连起来），观察落地结果。
func TestLandOneIDDepartmentsToGroups_DepthOverLimit_CascadesSkip(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(&model.OneIDDepartmentRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// D0 (root, depth=0) → D1 (depth=1) → ... → D11 (depth=11)
	depts := make([]model.OneIDDepartmentRecord, 0, 12)
	for i := 0; i < 12; i++ {
		d := model.OneIDDepartmentRecord{
			DepartmentID:   idOf(i),
			DepartmentName: "L" + idOf(i),
		}
		if i > 0 {
			d.DepartmentParentID = idOf(i - 1)
		}
		depts = append(depts, d)
	}
	for _, d := range depts {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}

	res, err := LandOneIDDepartmentsToGroups(context.Background())
	if err != nil {
		t.Fatalf("landing: %v", err)
	}

	// 1) 前 10 层（D0..D9，depth 0..9）应该都落地
	for i := 0; i < 10; i++ {
		var g model.UserGroup
		if err := model.DB(context.Background()).Where("source = ? AND source_ref = ?",
			model.GroupSourceOneIDDept, idOf(i)).First(&g).Error; err != nil {
			t.Fatalf("expected D%d 落地成功，但未找到: %v", i, err)
		}
		if g.Depth != i {
			t.Errorf("D%d 期望 depth=%d，实际 %d", i, i, g.Depth)
		}
	}

	// 2) 第 11 层（D10，depth 应为 10）应 create 失败（超 MaxGroupDepth）
	var g10 model.UserGroup
	if err := model.DB(context.Background()).Where("source = ? AND source_ref = ?",
		model.GroupSourceOneIDDept, idOf(10)).First(&g10).Error; err == nil {
		t.Fatalf("D10 不应落地（depth 超限），但找到了: %+v", g10)
	}

	// 3) 第 12 层（D11）必须连锁跳过，绝对不能以 parentGroupID=0 被错挂到根
	var g11 model.UserGroup
	if err := model.DB(context.Background()).Where("source = ? AND source_ref = ?",
		model.GroupSourceOneIDDept, idOf(11)).First(&g11).Error; err == nil {
		t.Fatalf("D11 父(D10)落地失败，本身必须跳过不得错挂根，但找到了: %+v", g11)
	}

	// 4) LandingFailures 应包含 D10(create) + D11(skipped_due_to_parent)
	failMap := make(map[string]string, len(res.LandingFailures))
	for _, f := range res.LandingFailures {
		failMap[f.DepartmentID] = f.Stage
	}
	if failMap[idOf(10)] != "create" {
		t.Errorf("D10 期望 LandingFailures Stage=create，实际 %q, 全量=%+v",
			failMap[idOf(10)], res.LandingFailures)
	}
	if failMap[idOf(11)] != "skipped_due_to_parent" {
		t.Errorf("D11 期望 LandingFailures Stage=skipped_due_to_parent，实际 %q, 全量=%+v",
			failMap[idOf(11)], res.LandingFailures)
	}

	// 5) 总 user_groups 数量应严格等于 10（D0..D9）
	var total int64
	model.DB(context.Background()).Model(&model.UserGroup{}).
		Where("source = ?", model.GroupSourceOneIDDept).
		Count(&total)
	if total != 10 {
		t.Errorf("期望落地 10 个 oneid_dept 组（D0..D9），实际 %d（可能 D11 被错挂根）", total)
	}
}

// TestLandOneIDDepartmentsToGroups_MissingParentStillMountsToRoot
// 保证"父不在本次数据集"（授权范围裁剪）的老语义不回归：
// 这种情况仍应挂根，不应被新增的 skipped_due_to_parent 分支误伤。
func TestLandOneIDDepartmentsToGroups_MissingParentStillMountsToRoot(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	if err := model.DB(context.Background()).AutoMigrate(&model.OneIDDepartmentRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 只含 D1，但声明 parent=D_ghost（D_ghost 不在本次数据里）
	depts := []model.OneIDDepartmentRecord{
		{DepartmentID: "D1", DepartmentName: "L1", DepartmentParentID: "D_ghost"},
	}
	for _, d := range depts {
		if err := model.DB(context.Background()).Create(&d).Error; err != nil {
			t.Fatalf("seed dept %s: %v", d.DepartmentID, err)
		}
	}

	res, err := LandOneIDDepartmentsToGroups(context.Background())
	if err != nil {
		t.Fatalf("landing: %v", err)
	}

	// D1 应挂根成功（parent 在 OneID 存在但不在本次拉取范围内 = 合法授权裁剪）
	var g1 model.UserGroup
	if err := model.DB(context.Background()).Where("source = ? AND source_ref = ?",
		model.GroupSourceOneIDDept, "D1").First(&g1).Error; err != nil {
		t.Fatalf("D1 应挂根成功: %v", err)
	}
	if g1.ParentID != 0 {
		t.Errorf("D1 应挂根（ParentID=0），实际 %d", g1.ParentID)
	}
	// 也不应有任何 LandingFailures
	if len(res.LandingFailures) != 0 {
		t.Errorf("不应产生 LandingFailures，实际 %+v", res.LandingFailures)
	}
}

// idOf 把 0..11 转成 "D0" .. "D11"。
func idOf(i int) string {
	if i < 10 {
		return "D" + string(rune('0'+i))
	}
	return "D1" + string(rune('0'+(i-10)))
}
