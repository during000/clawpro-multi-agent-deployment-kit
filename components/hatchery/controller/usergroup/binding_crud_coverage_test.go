package usergroup

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupBindingCrudCoverageDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.UserGroup{}, &model.GroupClosure{}, &model.UserGroupMember{},
		&model.GroupConfigBinding{}, &model.User{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	return db
}

// ── SetVisibility ────────────────────────────────────────

func TestCoverageSetVisibility_All(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	// 先绑定
	err := SetVisibility(db, ConfigTypeChannel, 10, "group", []uint{1, 2})
	if err != nil {
		t.Fatalf("SetVisibility group: %v", err)
	}

	// 设置为 all 清空绑定
	err = SetVisibility(db, ConfigTypeChannel, 10, "all", nil)
	if err != nil {
		t.Fatalf("SetVisibility all: %v", err)
	}

	var count int64
	db.Model(&model.GroupConfigBinding{}).Where("config_type = ? AND config_key = ?", model.ConfigTypeChannel, "10").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 bindings after all, got %d", count)
	}
}

func TestCoverageSetVisibility_Group(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	err := SetVisibility(db, ConfigTypeChannel, 5, "group", []uint{1, 2, 3})
	if err != nil {
		t.Fatalf("SetVisibility: %v", err)
	}

	var count int64
	db.Model(&model.GroupConfigBinding{}).Where("config_type = ? AND config_key = ?", model.ConfigTypeChannel, "5").Count(&count)
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestCoverageSetVisibility_InvalidType(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	err := SetVisibility(db, "invalid_type", 1, "group", []uint{1})
	if err == nil {
		t.Error("expected error for invalid config type")
	}
}

func TestCoverageSetVisibility_PolicyType(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	err := SetVisibility(db, ConfigTypePolicy, 1, "group", []uint{1})
	if err == nil {
		t.Error("expected error for non-additive type")
	}
}

// ── SetImageTypeVisibility ───────────────────────────────

func TestCoverageSetImageTypeVisibility_Group(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	err := SetImageTypeVisibility(db, "openclaw", "group", []uint{1, 2})
	if err != nil {
		t.Fatalf("SetImageTypeVisibility: %v", err)
	}

	var count int64
	db.Model(&model.GroupConfigBinding{}).Where("config_type = ? AND config_key = ?", model.ConfigTypeImageType, "openclaw").Count(&count)
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestCoverageSetImageTypeVisibility_All(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	SetImageTypeVisibility(db, "browser", "group", []uint{1})
	err := SetImageTypeVisibility(db, "browser", "all", nil)
	if err != nil {
		t.Fatalf("SetImageTypeVisibility all: %v", err)
	}

	var count int64
	db.Model(&model.GroupConfigBinding{}).Where("config_type = ? AND config_key = ?", model.ConfigTypeImageType, "browser").Count(&count)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

// ── SetPolicy ────────────────────────────────────────────

func TestCoverageSetPolicy_Valid(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	err := SetPolicy(db, 1, PolicyKeyTokenQuotaDay, `{"value":500}`)
	if err != nil {
		t.Fatalf("SetPolicy: %v", err)
	}

	var b model.GroupConfigBinding
	db.Where("config_type = ? AND config_key = ? AND group_id = ?", model.ConfigTypePolicy, PolicyKeyTokenQuotaDay, 1).First(&b)
	if b.ValueJSON != `{"value":500}` {
		t.Errorf("unexpected value: %s", b.ValueJSON)
	}
}

func TestCoverageSetPolicy_InvalidKey(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	err := SetPolicy(db, 1, "invalid_policy_key", `{}`)
	if err == nil {
		t.Error("expected error for invalid policy key")
	}
}

// ── DeletePolicy ─────────────────────────────────────────

func TestCoverageDeletePolicy_Valid(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	SetPolicy(db, 1, PolicyKeyInstanceQuota, `{"value":3}`)
	err := DeletePolicy(db, 1, PolicyKeyInstanceQuota)
	if err != nil {
		t.Fatalf("DeletePolicy: %v", err)
	}

	var count int64
	db.Model(&model.GroupConfigBinding{}).Where("config_type = ? AND config_key = ? AND group_id = ?", model.ConfigTypePolicy, PolicyKeyInstanceQuota, 1).Count(&count)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCoverageDeletePolicy_InvalidKey(t *testing.T) {
	db := setupBindingCrudCoverageDB(t)

	err := DeletePolicy(db, 1, "bad_key")
	if err == nil {
		t.Error("expected error for invalid policy key")
	}
}

// ── GetResourceBindingGroups ─────────────────────────────

func TestCoverageGetResourceBindingGroups_Found(t *testing.T) {
	setupBindingCrudCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "G1", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "G2", Source: "manual"})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "10", GroupID: 1, ValueJSON: "{}"})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "10", GroupID: 2, ValueJSON: "{}"})

	groups, err := GetResourceBindingGroups(context.Background(), model.ConfigTypeChannel, "10")
	if err != nil {
		t.Fatalf("GetResourceBindingGroups: %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("expected 2, got %d", len(groups))
	}
}

func TestCoverageGetResourceBindingGroups_Empty(t *testing.T) {
	setupBindingCrudCoverageDB(t)

	groups, err := GetResourceBindingGroups(context.Background(), model.ConfigTypeChannel, "999")
	if err != nil {
		t.Fatalf("GetResourceBindingGroups empty: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected 0, got %d", len(groups))
	}
}

// ── ValidateGroupIDs ─────────────────────────────────────

func TestCoverageValidateGroupIDs_Valid(t *testing.T) {
	setupBindingCrudCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "G1", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "G2", Source: "manual"})

	err := ValidateGroupIDs(context.Background(), []uint{1, 2})
	if err != nil {
		t.Errorf("ValidateGroupIDs valid: %v", err)
	}
}

func TestCoverageValidateGroupIDs_Invalid(t *testing.T) {
	setupBindingCrudCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "G1", Source: "manual"})

	err := ValidateGroupIDs(context.Background(), []uint{1, 999})
	if err == nil {
		t.Error("expected error for invalid group ID")
	}
}

func TestCoverageValidateGroupIDs_Empty(t *testing.T) {
	setupBindingCrudCoverageDB(t)

	err := ValidateGroupIDs(context.Background(), nil)
	if err != nil {
		t.Errorf("ValidateGroupIDs empty should not error: %v", err)
	}
}

// ── IsValidConfigType / IsValidPolicyKey ─────────────────

func TestCoverageIsValidConfigType(t *testing.T) {
	if !IsValidConfigType(ConfigTypeChannel) {
		t.Error("channel should be valid")
	}
	if !IsValidConfigType(ConfigTypePolicy) {
		t.Error("policy should be valid")
	}
	if IsValidConfigType("nonexistent") {
		t.Error("nonexistent should be invalid")
	}
}

func TestCoverageIsValidPolicyKey(t *testing.T) {
	if !IsValidPolicyKey(PolicyKeyTokenQuotaDay) {
		t.Error("token_quota_day should be valid")
	}
	if IsValidPolicyKey("bad_key") {
		t.Error("bad_key should be invalid")
	}
}

func TestCoverageGetPolicyDef(t *testing.T) {
	def, ok := GetPolicyDef(PolicyKeyInstanceQuota)
	if !ok {
		t.Fatal("expected to find instance_quota")
	}
	if def.ValueType != PolicyValueInt {
		t.Errorf("expected int, got %s", def.ValueType)
	}

	_, ok = GetPolicyDef("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}
