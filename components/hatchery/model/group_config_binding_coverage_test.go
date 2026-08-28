package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupBindingCoverageDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&GroupConfigBinding{}, &UserGroup{}, &GroupClosure{}, &UserGroupMember{}, &User{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	gdb = db
	t.Cleanup(func() { gdb = nil })
	return db
}

// ── SetAdditiveBindings ──────────────────────────────────

func TestCoverageSetAdditiveBindings_CreateNew(t *testing.T) {
	db := setupBindingCoverageDB(t)

	err := SetAdditiveBindings(db, ConfigTypeChannel, "101", []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("SetAdditiveBindings: %v", err)
	}

	var bindings []GroupConfigBinding
	db.Where("config_type = ? AND config_key = ?", ConfigTypeChannel, "101").Find(&bindings)
	if len(bindings) != 3 {
		t.Errorf("expected 3 bindings, got %d", len(bindings))
	}
}

func TestCoverageSetAdditiveBindings_ReplaceExisting(t *testing.T) {
	db := setupBindingCoverageDB(t)

	// 第一次写入
	SetAdditiveBindings(db, ConfigTypeChannel, "101", []uint{1, 2})
	// 第二次覆盖
	err := SetAdditiveBindings(db, ConfigTypeChannel, "101", []uint{3, 4, 5})
	if err != nil {
		t.Fatalf("SetAdditiveBindings replace: %v", err)
	}

	var bindings []GroupConfigBinding
	db.Where("config_type = ? AND config_key = ?", ConfigTypeChannel, "101").Find(&bindings)
	if len(bindings) != 3 {
		t.Errorf("expected 3 bindings after replace, got %d", len(bindings))
	}
}

func TestCoverageSetAdditiveBindings_EmptyGroupIDs(t *testing.T) {
	db := setupBindingCoverageDB(t)

	// 先写入
	SetAdditiveBindings(db, ConfigTypeChannel, "102", []uint{1})
	// 清空
	err := SetAdditiveBindings(db, ConfigTypeChannel, "102", nil)
	if err != nil {
		t.Fatalf("SetAdditiveBindings empty: %v", err)
	}

	var count int64
	db.Model(&GroupConfigBinding{}).Where("config_type = ? AND config_key = ?", ConfigTypeChannel, "102").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 after clear, got %d", count)
	}
}

// ── UpsertPolicyBinding ──────────────────────────────────

func TestCoverageUpsertPolicyBinding_Create(t *testing.T) {
	db := setupBindingCoverageDB(t)

	err := UpsertPolicyBinding(db, 1, "token_quota_day", `{"value":100}`)
	if err != nil {
		t.Fatalf("UpsertPolicyBinding create: %v", err)
	}

	var b GroupConfigBinding
	db.Where("config_type = ? AND config_key = ? AND group_id = ?", ConfigTypePolicy, "token_quota_day", 1).First(&b)
	if b.ValueJSON != `{"value":100}` {
		t.Errorf("unexpected value: %s", b.ValueJSON)
	}
}

func TestCoverageUpsertPolicyBinding_Update(t *testing.T) {
	db := setupBindingCoverageDB(t)

	UpsertPolicyBinding(db, 1, "token_quota_day", `{"value":100}`)
	err := UpsertPolicyBinding(db, 1, "token_quota_day", `{"value":200}`)
	if err != nil {
		t.Fatalf("UpsertPolicyBinding update: %v", err)
	}

	var b GroupConfigBinding
	db.Where("config_type = ? AND config_key = ? AND group_id = ?", ConfigTypePolicy, "token_quota_day", 1).First(&b)
	if b.ValueJSON != `{"value":200}` {
		t.Errorf("unexpected value after update: %s", b.ValueJSON)
	}
}

// ── DeletePolicyBinding ──────────────────────────────────

func TestCoverageDeletePolicyBinding(t *testing.T) {
	db := setupBindingCoverageDB(t)

	UpsertPolicyBinding(db, 5, "instance_quota", `{"value":10}`)
	err := DeletePolicyBinding(db, 5, "instance_quota")
	if err != nil {
		t.Fatalf("DeletePolicyBinding: %v", err)
	}

	var count int64
	db.Model(&GroupConfigBinding{}).Where("group_id = ? AND config_key = ?", 5, "instance_quota").Count(&count)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCoverageDeletePolicyBinding_Idempotent(t *testing.T) {
	db := setupBindingCoverageDB(t)

	// 删除不存在的绑定不应报错
	err := DeletePolicyBinding(db, 999, "nonexistent")
	if err != nil {
		t.Errorf("delete nonexistent should not error: %v", err)
	}
}

// ── GetBindingsByGroup ───────────────────────────────────

func TestCoverageGetBindingsByGroup_All(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "1", GroupID: 10, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "quota", GroupID: 10, ValueJSON: `{"value":5}`})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "2", GroupID: 20, ValueJSON: "{}"})

	bindings, err := GetBindingsByGroup(context.Background(), 10, "")
	if err != nil {
		t.Fatalf("GetBindingsByGroup: %v", err)
	}
	if len(bindings) != 2 {
		t.Errorf("expected 2, got %d", len(bindings))
	}
}

func TestCoverageGetBindingsByGroup_Filtered(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "1", GroupID: 10, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "quota", GroupID: 10, ValueJSON: `{"value":5}`})

	bindings, err := GetBindingsByGroup(context.Background(), 10, ConfigTypePolicy)
	if err != nil {
		t.Fatalf("GetBindingsByGroup filtered: %v", err)
	}
	if len(bindings) != 1 {
		t.Errorf("expected 1, got %d", len(bindings))
	}
}

// ── GetBindingsByResource ────────────────────────────────

func TestCoverageGetBindingsByResource(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "55", GroupID: 1, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "55", GroupID: 2, ValueJSON: "{}"})

	bindings, err := GetBindingsByResource(context.Background(), ConfigTypeChannel, "55")
	if err != nil {
		t.Fatalf("GetBindingsByResource: %v", err)
	}
	if len(bindings) != 2 {
		t.Errorf("expected 2, got %d", len(bindings))
	}
}

// ── GetBindingGroupIDs ───────────────────────────────────

func TestCoverageGetBindingGroupIDs(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeMCP, ConfigKey: "mcp1", GroupID: 10, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeMCP, ConfigKey: "mcp1", GroupID: 20, ValueJSON: "{}"})

	ids, err := GetBindingGroupIDs(context.Background(), ConfigTypeMCP, "mcp1")
	if err != nil {
		t.Fatalf("GetBindingGroupIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2, got %d", len(ids))
	}
}

// ── GetBindingsByGroups ──────────────────────────────────

func TestCoverageGetBindingsByGroups(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "1", GroupID: 10, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "2", GroupID: 20, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "x", GroupID: 10, ValueJSON: "{}"})

	bindings, err := GetBindingsByGroups(context.Background(), []uint{10, 20}, ConfigTypeChannel)
	if err != nil {
		t.Fatalf("GetBindingsByGroups: %v", err)
	}
	if len(bindings) != 2 {
		t.Errorf("expected 2, got %d", len(bindings))
	}
}

func TestCoverageGetBindingsByGroups_Empty(t *testing.T) {
	setupBindingCoverageDB(t)

	bindings, err := GetBindingsByGroups(context.Background(), nil, ConfigTypeChannel)
	if err != nil {
		t.Fatalf("GetBindingsByGroups nil: %v", err)
	}
	if bindings != nil {
		t.Errorf("expected nil, got %v", bindings)
	}
}

// ── GetPolicyBindingsByGroups ────────────────────────────

func TestCoverageGetPolicyBindingsByGroups(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "token_quota_day", GroupID: 1, ValueJSON: `{"value":100}`})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "token_quota_day", GroupID: 2, ValueJSON: `{"value":200}`})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "instance_quota", GroupID: 1, ValueJSON: `{"value":5}`})

	bindings, err := GetPolicyBindingsByGroups(context.Background(), []uint{1, 2}, "token_quota_day")
	if err != nil {
		t.Fatalf("GetPolicyBindingsByGroups: %v", err)
	}
	if len(bindings) != 2 {
		t.Errorf("expected 2, got %d", len(bindings))
	}
}

func TestCoverageGetPolicyBindingsByGroups_Empty(t *testing.T) {
	setupBindingCoverageDB(t)

	bindings, err := GetPolicyBindingsByGroups(context.Background(), nil, "key")
	if err != nil {
		t.Fatalf("GetPolicyBindingsByGroups nil: %v", err)
	}
	if bindings != nil {
		t.Errorf("expected nil, got %v", bindings)
	}
}

// ── GetAllPolicyBindingsByGroups ─────────────────────────

func TestCoverageGetAllPolicyBindingsByGroups(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "a", GroupID: 1, ValueJSON: `{}`})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "b", GroupID: 1, ValueJSON: `{}`})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "c", GroupID: 1, ValueJSON: `{}`})

	bindings, err := GetAllPolicyBindingsByGroups(context.Background(), []uint{1})
	if err != nil {
		t.Fatalf("GetAllPolicyBindingsByGroups: %v", err)
	}
	if len(bindings) != 2 {
		t.Errorf("expected 2 policy bindings, got %d", len(bindings))
	}
}

func TestCoverageGetAllPolicyBindingsByGroups_Empty(t *testing.T) {
	setupBindingCoverageDB(t)

	bindings, err := GetAllPolicyBindingsByGroups(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetAllPolicyBindingsByGroups nil: %v", err)
	}
	if bindings != nil {
		t.Errorf("expected nil, got %v", bindings)
	}
}

// ── GetRestrictedImageTypes ──────────────────────────────

func TestCoverageGetRestrictedImageTypes(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeImageType, ConfigKey: "openclaw", GroupID: 1, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeImageType, ConfigKey: "browser", GroupID: 2, ValueJSON: "{}"})

	keys, err := GetRestrictedImageTypes(context.Background())
	if err != nil {
		t.Fatalf("GetRestrictedImageTypes: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2, got %d", len(keys))
	}
}

func TestCoverageGetRestrictedImageTypes_Empty(t *testing.T) {
	setupBindingCoverageDB(t)

	keys, err := GetRestrictedImageTypes(context.Background())
	if err != nil {
		t.Fatalf("GetRestrictedImageTypes empty: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0, got %d", len(keys))
	}
}

// ── GetVisibleImageTypesByGroups ─────────────────────────

func TestCoverageGetVisibleImageTypesByGroups(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeImageType, ConfigKey: "openclaw", GroupID: 1, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeImageType, ConfigKey: "browser", GroupID: 2, ValueJSON: "{}"})

	keys, err := GetVisibleImageTypesByGroups(context.Background(), []uint{1})
	if err != nil {
		t.Fatalf("GetVisibleImageTypesByGroups: %v", err)
	}
	if len(keys) != 1 || keys[0] != "openclaw" {
		t.Errorf("expected [openclaw], got %v", keys)
	}
}

func TestCoverageGetVisibleImageTypesByGroups_Empty(t *testing.T) {
	setupBindingCoverageDB(t)

	keys, err := GetVisibleImageTypesByGroups(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetVisibleImageTypesByGroups nil: %v", err)
	}
	if keys != nil {
		t.Errorf("expected nil, got %v", keys)
	}
}

// ── GetResourceVisibilityGroupIDsByUint ──────────────────

func TestCoverageGetResourceVisibilityGroupIDsByUint(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "10", GroupID: 1, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "10", GroupID: 2, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "20", GroupID: 3, ValueJSON: "{}"})

	result, err := GetResourceVisibilityGroupIDsByUint(context.Background(), ConfigTypeChannel, []uint{10, 20})
	if err != nil {
		t.Fatalf("GetResourceVisibilityGroupIDsByUint: %v", err)
	}
	if len(result[10]) != 2 {
		t.Errorf("expected 2 groups for resource 10, got %d", len(result[10]))
	}
	if len(result[20]) != 1 {
		t.Errorf("expected 1 group for resource 20, got %d", len(result[20]))
	}
}

func TestCoverageGetResourceVisibilityGroupIDsByUint_Empty(t *testing.T) {
	setupBindingCoverageDB(t)

	result, err := GetResourceVisibilityGroupIDsByUint(context.Background(), ConfigTypeChannel, nil)
	if err != nil {
		t.Fatalf("GetResourceVisibilityGroupIDsByUint nil: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

// ── IsGroupUsedByConfigBindings ──────────────────────────

func TestCoverageIsGroupUsedByConfigBindings_True(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "1", GroupID: 5, ValueJSON: "{}"})

	used, err := IsGroupUsedByConfigBindings(context.Background(), 5)
	if err != nil {
		t.Fatalf("IsGroupUsedByConfigBindings: %v", err)
	}
	if !used {
		t.Error("expected true")
	}
}

func TestCoverageIsGroupUsedByConfigBindings_False(t *testing.T) {
	setupBindingCoverageDB(t)

	used, err := IsGroupUsedByConfigBindings(context.Background(), 999)
	if err != nil {
		t.Fatalf("IsGroupUsedByConfigBindings: %v", err)
	}
	if used {
		t.Error("expected false")
	}
}

// ── CleanupConfigBindingsByGroupID ───────────────────────

func TestCoverageCleanupConfigBindingsByGroupID(t *testing.T) {
	db := setupBindingCoverageDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "1", GroupID: 7, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "k", GroupID: 7, ValueJSON: "{}"})

	err := CleanupConfigBindingsByGroupID(db, 7)
	if err != nil {
		t.Fatalf("CleanupConfigBindingsByGroupID: %v", err)
	}

	var count int64
	db.Model(&GroupConfigBinding{}).Where("group_id = ?", 7).Count(&count)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}
