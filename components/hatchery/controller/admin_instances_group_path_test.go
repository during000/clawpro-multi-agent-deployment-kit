package controller

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initGroupPathTestDB 仅迁移 UserGroup 表，供 fetchGroupFullPathMap 单测使用。
func initGroupPathTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.UserGroup{}); err != nil {
		t.Fatalf("migrate UserGroup: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// TestFetchGroupFullPathMap_Basic 验证批量查询、去重、过滤 0。
func TestFetchGroupFullPathMap_Basic(t *testing.T) {
	initGroupPathTestDB(t)

	g1 := model.UserGroup{Name: "研发组", FullPath: "研发中心/研发组", Source: model.GroupSourceManual}
	g2 := model.UserGroup{Name: "后端组", FullPath: "研发中心/研发组/后端组", Source: model.GroupSourceManual}
	model.DB(context.Background()).Create(&g1)
	model.DB(context.Background()).Create(&g2)

	got := fetchGroupFullPathMap(context.Background(), []uint{g1.ID, 0, g2.ID, g1.ID, 0, 99999})

	if got[g1.ID] != "研发中心/研发组" {
		t.Errorf("g1 full_path 错: %q", got[g1.ID])
	}
	if got[g2.ID] != "研发中心/研发组/后端组" {
		t.Errorf("g2 full_path 错: %q", got[g2.ID])
	}
	if _, ok := got[0]; ok {
		t.Errorf("0 键不应出现在结果里")
	}
	if _, ok := got[99999]; ok {
		t.Errorf("不存在的 ID 不应出现在结果里")
	}
}

// TestFetchGroupFullPathMap_EmptyAndAllZero 空列表 / 全零 → 空 map。
func TestFetchGroupFullPathMap_EmptyAndAllZero(t *testing.T) {
	initGroupPathTestDB(t)

	if got := fetchGroupFullPathMap(context.Background(), nil); len(got) != 0 {
		t.Errorf("nil 输入应返回空 map，实际 %+v", got)
	}
	if got := fetchGroupFullPathMap(context.Background(), []uint{}); len(got) != 0 {
		t.Errorf("空切片应返回空 map，实际 %+v", got)
	}
	if got := fetchGroupFullPathMap(context.Background(), []uint{0, 0, 0}); len(got) != 0 {
		t.Errorf("全零输入应返回空 map，实际 %+v", got)
	}
}

// TestBuildAdminInstanceWithStatus_GroupFields buildAdminInstanceWithStatus
// 把 inst.GroupID 填到输出的 GroupID 字段；GroupFullPath 由调用方回填，这里应默认空串。
func TestBuildAdminInstanceWithStatus_GroupFields(t *testing.T) {
	inst := model.Instance{
		Name:    "x",
		UserID:  1,
		GroupID: 42,
	}
	// 用 adminInstanceItem 包装
	item := adminInstanceItem{Instance: inst, Username: "alice"}
	out := buildAdminInstanceWithStatus(context.Background(), item, nil)

	if out.GroupID != 42 {
		t.Errorf("GroupID 应为 42，实际 %d", out.GroupID)
	}
	if out.GroupFullPath != "" {
		t.Errorf("GroupFullPath 应为空串（由调用方回填），实际 %q", out.GroupFullPath)
	}

	// GroupID=0 的场景
	inst2 := model.Instance{Name: "y", UserID: 1, GroupID: 0}
	out2 := buildAdminInstanceWithStatus(context.Background(), adminInstanceItem{Instance: inst2}, nil)
	if out2.GroupID != 0 {
		t.Errorf("GroupID 应为 0，实际 %d", out2.GroupID)
	}
	if out2.GroupFullPath != "" {
		t.Errorf("GroupFullPath 应为空串，实际 %q", out2.GroupFullPath)
	}
}
