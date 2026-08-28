package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupGCBTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&GroupConfigBinding{}, &UserGroup{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	oldDB := gdb
	gdb = db
	t.Cleanup(func() { gdb = oldDB })
	return db
}

func TestSetAdditiveBindings_Empty(t *testing.T) {
	db := setupGCBTestDB(t)

	// 先插入一条旧绑定
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch1", GroupID: 1, ValueJSON: "{}"})

	// 用空 groupIDs 清空
	err := SetAdditiveBindings(db, ConfigTypeChannel, "ch1", nil)
	if err != nil {
		t.Fatalf("SetAdditiveBindings empty: %v", err)
	}
	var count int64
	db.Model(&GroupConfigBinding{}).Where("config_type = ? AND config_key = ?", ConfigTypeChannel, "ch1").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 bindings, got %d", count)
	}
}

func TestSetAdditiveBindings_Replace(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch1", GroupID: 1, ValueJSON: "{}"})

	err := SetAdditiveBindings(db, ConfigTypeChannel, "ch1", []uint{2, 3})
	if err != nil {
		t.Fatalf("SetAdditiveBindings replace: %v", err)
	}
	var bindings []GroupConfigBinding
	db.Where("config_type = ? AND config_key = ?", ConfigTypeChannel, "ch1").Find(&bindings)
	if len(bindings) != 2 {
		t.Errorf("expected 2 bindings, got %d", len(bindings))
	}
}

func TestUpsertPolicyBinding_CreateAndUpdate(t *testing.T) {
	db := setupGCBTestDB(t)

	// 创建
	err := UpsertPolicyBinding(db, 1, "token_quota_day", `{"value":100}`)
	if err != nil {
		t.Fatalf("UpsertPolicyBinding create: %v", err)
	}
	var b GroupConfigBinding
	db.Where("config_type = ? AND config_key = ? AND group_id = ?", ConfigTypePolicy, "token_quota_day", 1).First(&b)
	if b.ValueJSON != `{"value":100}` {
		t.Errorf("unexpected value: %s", b.ValueJSON)
	}

	// 更新
	err = UpsertPolicyBinding(db, 1, "token_quota_day", `{"value":200}`)
	if err != nil {
		t.Fatalf("UpsertPolicyBinding update: %v", err)
	}
	db.Where("config_type = ? AND config_key = ? AND group_id = ?", ConfigTypePolicy, "token_quota_day", 1).First(&b)
	if b.ValueJSON != `{"value":200}` {
		t.Errorf("expected updated value, got: %s", b.ValueJSON)
	}
}

func TestDeletePolicyBinding(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "instance_quota", GroupID: 5, ValueJSON: `{"value":10}`})

	err := DeletePolicyBinding(db, 5, "instance_quota")
	if err != nil {
		t.Fatalf("DeletePolicyBinding: %v", err)
	}
	var count int64
	db.Model(&GroupConfigBinding{}).Where("group_id = ?", 5).Count(&count)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestGetBindingsByGroup(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch1", GroupID: 1, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeMCP, ConfigKey: "mcp1", GroupID: 1, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch2", GroupID: 2, ValueJSON: "{}"})

	// 不过滤
	all, err := GetBindingsByGroup(context.Background(), 1, "")
	if err != nil {
		t.Fatalf("GetBindingsByGroup all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}

	// 按类型过滤
	channels, err := GetBindingsByGroup(context.Background(), 1, ConfigTypeChannel)
	if err != nil {
		t.Fatalf("GetBindingsByGroup channel: %v", err)
	}
	if len(channels) != 1 {
		t.Errorf("expected 1 channel, got %d", len(channels))
	}
}

func TestGetBindingsByResource(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch1", GroupID: 1, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch1", GroupID: 2, ValueJSON: "{}"})

	bindings, err := GetBindingsByResource(context.Background(), ConfigTypeChannel, "ch1")
	if err != nil {
		t.Fatalf("GetBindingsByResource: %v", err)
	}
	if len(bindings) != 2 {
		t.Errorf("expected 2, got %d", len(bindings))
	}
}

func TestGetBindingGroupIDs(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeMCP, ConfigKey: "mcp1", GroupID: 3, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeMCP, ConfigKey: "mcp1", GroupID: 7, ValueJSON: "{}"})

	ids, err := GetBindingGroupIDs(context.Background(), ConfigTypeMCP, "mcp1")
	if err != nil {
		t.Fatalf("GetBindingGroupIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2, got %d", len(ids))
	}
}

func TestGetBindingsByGroups(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch1", GroupID: 1, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch2", GroupID: 2, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeMCP, ConfigKey: "m1", GroupID: 1, ValueJSON: "{}"})

	// 空 group
	empty, err := GetBindingsByGroups(context.Background(), nil, ConfigTypeChannel)
	if err != nil {
		t.Fatalf("GetBindingsByGroups nil: %v", err)
	}
	if empty != nil {
		t.Errorf("expected nil, got %v", empty)
	}

	results, err := GetBindingsByGroups(context.Background(), []uint{1, 2}, ConfigTypeChannel)
	if err != nil {
		t.Fatalf("GetBindingsByGroups: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2, got %d", len(results))
	}
}

func TestGetPolicyBindingsByGroups(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "token_quota_day", GroupID: 1, ValueJSON: `{"value":50}`})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "token_quota_day", GroupID: 2, ValueJSON: `{"value":100}`})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "instance_quota", GroupID: 1, ValueJSON: `{"value":5}`})

	// 空
	empty, err := GetPolicyBindingsByGroups(context.Background(), nil, "token_quota_day")
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if empty != nil {
		t.Errorf("expected nil, got %v", empty)
	}

	results, err := GetPolicyBindingsByGroups(context.Background(), []uint{1, 2}, "token_quota_day")
	if err != nil {
		t.Fatalf("GetPolicyBindingsByGroups: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2, got %d", len(results))
	}
}

func TestGetAllPolicyBindingsByGroups(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "token_quota_day", GroupID: 1, ValueJSON: `{"value":50}`})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "instance_quota", GroupID: 1, ValueJSON: `{"value":5}`})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch1", GroupID: 1, ValueJSON: "{}"})

	// 空
	empty, err := GetAllPolicyBindingsByGroups(context.Background(), nil)
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if empty != nil {
		t.Errorf("expected nil, got %v", empty)
	}

	results, err := GetAllPolicyBindingsByGroups(context.Background(), []uint{1})
	if err != nil {
		t.Fatalf("GetAllPolicyBindingsByGroups: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 policy bindings, got %d", len(results))
	}
}

func TestGetResourceVisibilityGroupIDsByUint(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "10", GroupID: 1, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "10", GroupID: 2, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "20", GroupID: 3, ValueJSON: "{}"})

	// 空
	empty, err := GetResourceVisibilityGroupIDsByUint(context.Background(), ConfigTypeChannel, nil)
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected empty map, got %v", empty)
	}

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

func TestIsGroupUsedByConfigBindings(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch1", GroupID: 5, ValueJSON: "{}"})

	used, err := IsGroupUsedByConfigBindings(context.Background(), 5)
	if err != nil {
		t.Fatalf("IsGroupUsedByConfigBindings: %v", err)
	}
	if !used {
		t.Error("expected true")
	}

	unused, err := IsGroupUsedByConfigBindings(context.Background(), 999)
	if err != nil {
		t.Fatalf("IsGroupUsedByConfigBindings unused: %v", err)
	}
	if unused {
		t.Error("expected false")
	}
}

func TestCleanupConfigBindingsByGroupID(t *testing.T) {
	db := setupGCBTestDB(t)

	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch1", GroupID: 5, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypePolicy, ConfigKey: "p1", GroupID: 5, ValueJSON: "{}"})
	db.Create(&GroupConfigBinding{ConfigType: ConfigTypeChannel, ConfigKey: "ch2", GroupID: 6, ValueJSON: "{}"})

	err := CleanupConfigBindingsByGroupID(db, 5)
	if err != nil {
		t.Fatalf("CleanupConfigBindingsByGroupID: %v", err)
	}

	var count int64
	db.Model(&GroupConfigBinding{}).Where("group_id = ?", 5).Count(&count)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
	db.Model(&GroupConfigBinding{}).Where("group_id = ?", 6).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 for group 6, got %d", count)
	}
}
