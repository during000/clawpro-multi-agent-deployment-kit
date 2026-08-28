package usergroup

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupResolveCoverageDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.UserGroup{}, &model.GroupClosure{}, &model.UserGroupMember{},
		&model.GroupConfigBinding{}, &model.User{}, &model.AIModel{},
		&model.ModelVisibilityGroup{}, &model.AIChannel{}, &model.AIImage{},
		&model.VpcConfig{}, &model.PluginBundle{}, &model.OpenClawRole{},
		&model.RoleVisibilityGroup{}, &model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// 创建组 + closure 的辅助函数
func createGroupWithClosure(t *testing.T, id uint, parentID uint, name string) {
	t.Helper()
	model.DB(context.Background()).Create(&model.UserGroup{ID: id, Name: name, ParentID: parentID, Source: "manual", FullPath: name})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: id, DescendantID: id, Depth: 0})
	if parentID > 0 {
		// 为 parent 的祖先链每条都复制一份到 id
		var parentClosures []model.GroupClosure
		model.DB(context.Background()).Where("descendant_id = ?", parentID).Find(&parentClosures)
		for _, c := range parentClosures {
			model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: c.AncestorID, DescendantID: id, Depth: c.Depth + 1})
		}
	}
}

// ── ResolvePolicyBool ────────────────────────────────────

func TestCoverageResolvePolicyBool_Found(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	createGroupWithClosure(t, 2, 1, "Child")

	// 在 Root 上设置策略
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyAgentTerminal,
		GroupID:    1,
		ValueJSON:  `{"enabled":true}`,
	})

	// 从 Child 解析应继承 Root
	val, source, err := ResolvePolicyBool(context.Background(), PolicyKeyAgentTerminal, []uint{2, 1}, false)
	if err != nil {
		t.Fatalf("ResolvePolicyBool: %v", err)
	}
	if !val {
		t.Error("expected true")
	}
	if source.Type != SourceInherited {
		t.Errorf("expected inherited, got %s", source.Type)
	}
}

func TestCoverageResolvePolicyBool_LocalOverride(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	createGroupWithClosure(t, 2, 1, "Child")

	// Root 设 true，Child 设 false
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypePolicy, ConfigKey: PolicyKeyAgentTerminal, GroupID: 1, ValueJSON: `{"enabled":true}`})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypePolicy, ConfigKey: PolicyKeyAgentTerminal, GroupID: 2, ValueJSON: `{"enabled":false}`})

	val, source, _ := ResolvePolicyBool(context.Background(), PolicyKeyAgentTerminal, []uint{2, 1}, true)
	if val {
		t.Error("expected false (local override)")
	}
	if source.Type != SourceLocal {
		t.Errorf("expected local, got %s", source.Type)
	}
}

func TestCoverageResolvePolicyBool_Fallback(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")

	val, source, _ := ResolvePolicyBool(context.Background(), PolicyKeyAgentTerminal, []uint{1}, true)
	if !val {
		t.Error("expected fallback true")
	}
	if source.Type != SourceSiteDefault {
		t.Errorf("expected site_default, got %s", source.Type)
	}
}

func TestCoverageResolvePolicyBool_EmptyAncestors(t *testing.T) {
	setupResolveCoverageDB(t)

	val, source, _ := ResolvePolicyBool(context.Background(), PolicyKeyAgentTerminal, nil, true)
	if !val {
		t.Error("expected fallback for empty ancestors")
	}
	if source.Type != SourceSiteDefault {
		t.Errorf("expected site_default, got %s", source.Type)
	}
}

func TestCoverageResolvePolicyBool_InvalidJSON(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypePolicy, ConfigKey: "test_key", GroupID: 1, ValueJSON: `invalid json`})

	val, _, err := ResolvePolicyBool(context.Background(), "test_key", []uint{1}, false)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if val != false {
		t.Error("expected fallback on error")
	}
}

// ── ResolvePolicyInt ─────────────────────────────────────

func TestCoverageResolvePolicyInt_Found(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypePolicy, ConfigKey: PolicyKeyTokenQuotaDay, GroupID: 1, ValueJSON: `{"value":500}`})

	val, source, _ := ResolvePolicyInt(context.Background(), PolicyKeyTokenQuotaDay, []uint{1}, 100)
	if val != 500 {
		t.Errorf("expected 500, got %d", val)
	}
	if source.Type != SourceLocal {
		t.Errorf("expected local, got %s", source.Type)
	}
}

func TestCoverageResolvePolicyInt_Fallback(t *testing.T) {
	setupResolveCoverageDB(t)

	val, _, _ := ResolvePolicyInt(context.Background(), PolicyKeyTokenQuotaDay, []uint{1}, 999)
	if val != 999 {
		t.Errorf("expected fallback 999, got %d", val)
	}
}

func TestCoverageResolvePolicyInt_InvalidJSON(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypePolicy, ConfigKey: PolicyKeyTokenQuotaDay, GroupID: 1, ValueJSON: `not json`})

	val, _, err := ResolvePolicyInt(context.Background(), PolicyKeyTokenQuotaDay, []uint{1}, 100)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
	if val != 100 {
		t.Error("expected fallback on error")
	}
}

// ── ResolvePolicyBoolForGroup ────────────────────────────

func TestCoverageResolvePolicyBoolForGroup_ZeroGroupID(t *testing.T) {
	setupResolveCoverageDB(t)

	val := ResolvePolicyBoolForGroup(context.Background(), PolicyKeyAgentTerminal, 0, true)
	if !val {
		t.Error("expected fallback for groupID=0")
	}
}

func TestCoverageResolvePolicyBoolForGroup_WithGroup(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypePolicy, ConfigKey: PolicyKeyGatewayUI, GroupID: 1, ValueJSON: `{"enabled":false}`})

	val := ResolvePolicyBoolForGroup(context.Background(), PolicyKeyGatewayUI, 1, true)
	if val {
		t.Error("expected false from group config")
	}
}

// ── ResolvePolicyIntForGroup ─────────────────────────────

func TestCoverageResolvePolicyIntForGroup_ZeroGroupID(t *testing.T) {
	setupResolveCoverageDB(t)

	val := ResolvePolicyIntForGroup(context.Background(), PolicyKeyInstanceQuota, 0, 5)
	if val != 5 {
		t.Errorf("expected 5, got %d", val)
	}
}

func TestCoverageResolvePolicyIntForGroup_WithGroup(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypePolicy, ConfigKey: PolicyKeyInstanceQuota, GroupID: 1, ValueJSON: `{"value":10}`})

	val := ResolvePolicyIntForGroup(context.Background(), PolicyKeyInstanceQuota, 1, 5)
	if val != 10 {
		t.Errorf("expected 10, got %d", val)
	}
}

// ── ResolvePolicyString ────────────────────────────────────

func TestResolvePolicyString_Found(t *testing.T) {
	setupResolveCoverageDB(t)
	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyTokenQuotaRules,
		GroupID:    1,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":200}]"}`,
	})

	val, source, err := ResolvePolicyString(context.Background(), PolicyKeyTokenQuotaRules, []uint{1}, "fallback")
	if err != nil {
		t.Fatalf("ResolvePolicyString: %v", err)
	}
	if val != `[{"mode":"day","limit":200}]` {
		t.Errorf("expected rules json, got %q", val)
	}
	if source.Type != SourceLocal {
		t.Errorf("expected source local, got %v", source.Type)
	}
}

func TestResolvePolicyString_Fallback(t *testing.T) {
	setupResolveCoverageDB(t)
	createGroupWithClosure(t, 1, 0, "Root")

	val, source, _ := ResolvePolicyString(context.Background(), PolicyKeyTokenQuotaRules, []uint{1}, "fb")
	if val != "fb" {
		t.Errorf("expected fallback, got %q", val)
	}
	if source.Type != SourceSiteDefault {
		t.Errorf("expected site_default, got %v", source.Type)
	}
}

func TestResolvePolicyStringForGroup(t *testing.T) {
	setupResolveCoverageDB(t)
	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyTokenQuotaRules,
		GroupID:    1,
		ValueJSON:  `{"value":"[{\"mode\":\"month\",\"limit\":1000000}]"}`,
	})

	val := ResolvePolicyStringForGroup(context.Background(), PolicyKeyTokenQuotaRules, 1, "")
	if val != `[{"mode":"month","limit":1000000}]` {
		t.Errorf("expected rules json, got %q", val)
	}
}

func TestResolvePolicyStringForGroup_Zero(t *testing.T) {
	setupResolveCoverageDB(t)
	val := ResolvePolicyStringForGroup(context.Background(), PolicyKeyTokenQuotaRules, 0, "fb")
	if val != "fb" {
		t.Errorf("expected fallback for groupID=0, got %q", val)
	}
}

// ── IsResourceVisible ────────────────────────────────────

func TestCoverageIsResourceVisible_AllType(t *testing.T) {
	setupResolveCoverageDB(t)

	visible, err := IsResourceVisible(context.Background(), ConfigTypeChannel, 1, "all", []uint{1, 2})
	if err != nil {
		t.Fatalf("IsResourceVisible: %v", err)
	}
	if !visible {
		t.Error("expected visible for type=all")
	}
}

func TestCoverageIsResourceVisible_GroupType_Visible(t *testing.T) {
	setupResolveCoverageDB(t)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "5", GroupID: 1, ValueJSON: "{}"})

	visible, err := IsResourceVisible(context.Background(), ConfigTypeChannel, 5, "group", []uint{1, 2})
	if err != nil {
		t.Fatalf("IsResourceVisible: %v", err)
	}
	if !visible {
		t.Error("expected visible")
	}
}

func TestCoverageIsResourceVisible_GroupType_NotVisible(t *testing.T) {
	setupResolveCoverageDB(t)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "5", GroupID: 99, ValueJSON: "{}"})

	visible, err := IsResourceVisible(context.Background(), ConfigTypeChannel, 5, "group", []uint{1, 2})
	if err != nil {
		t.Fatalf("IsResourceVisible: %v", err)
	}
	if visible {
		t.Error("expected not visible")
	}
}

func TestCoverageIsResourceVisible_EmptyGroups(t *testing.T) {
	setupResolveCoverageDB(t)

	visible, err := IsResourceVisible(context.Background(), ConfigTypeChannel, 5, "group", nil)
	if err != nil {
		t.Fatalf("IsResourceVisible: %v", err)
	}
	if visible {
		t.Error("expected not visible for empty groups")
	}
}

// ── ResolveImageTypes ────────────────────────────────────

func TestCoverageResolveImageTypes_NoRestrictions(t *testing.T) {
	setupResolveCoverageDB(t)

	allTypes := []string{"openclaw", "browser", "custom"}
	result, err := ResolveImageTypes(context.Background(), []uint{1}, allTypes)
	if err != nil {
		t.Fatalf("ResolveImageTypes: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 (no restrictions), got %d", len(result))
	}
}

func TestCoverageResolveImageTypes_WithRestrictions(t *testing.T) {
	setupResolveCoverageDB(t)

	// 限制 "browser" 只对组 1 可见
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeImageType, ConfigKey: "browser", GroupID: 1, ValueJSON: "{}"})

	allTypes := []string{"openclaw", "browser", "custom"}

	// 组 1 可以看到所有
	result, _ := ResolveImageTypes(context.Background(), []uint{1}, allTypes)
	if len(result) != 3 {
		t.Errorf("group 1: expected 3, got %d: %v", len(result), result)
	}

	// 组 2 看不到 browser
	result, _ = ResolveImageTypes(context.Background(), []uint{2}, allTypes)
	if len(result) != 2 {
		t.Errorf("group 2: expected 2, got %d: %v", len(result), result)
	}
}

func TestCoverageResolveImageTypes_EmptyGroups(t *testing.T) {
	setupResolveCoverageDB(t)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeImageType, ConfigKey: "browser", GroupID: 1, ValueJSON: "{}"})

	allTypes := []string{"openclaw", "browser"}
	result, _ := ResolveImageTypes(context.Background(), nil, allTypes)
	// 无组时只能看到不受限制的
	if len(result) != 1 {
		t.Errorf("expected 1 (only unrestricted), got %d: %v", len(result), result)
	}
}

// ── ResolveAdditiveResources ─────────────────────────────

func TestCoverageResolveAdditiveResources(t *testing.T) {
	setupResolveCoverageDB(t)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "10", GroupID: 1, ValueJSON: "{}"})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "20", GroupID: 2, ValueJSON: "{}"})

	ids, err := ResolveAdditiveResources(context.Background(), ConfigTypeChannel, []uint{1, 2})
	if err != nil {
		t.Fatalf("ResolveAdditiveResources: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2, got %d", len(ids))
	}
}

func TestCoverageResolveAdditiveResources_ImageType(t *testing.T) {
	setupResolveCoverageDB(t)

	// image_type 应返回 nil
	ids, err := ResolveAdditiveResources(context.Background(), ConfigTypeImageType, []uint{1})
	if err != nil {
		t.Fatalf("ResolveAdditiveResources image_type: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil for image_type, got %v", ids)
	}
}

// ── FilterModelsByVisibility ─────────────────────────────

func TestCoverageFilterModelsByVisibility_AllType(t *testing.T) {
	setupResolveCoverageDB(t)

	models := []model.AIModel{
		{Model: gorm.Model{ID: 1}, VisibilityType: "all", ModelName: "M1"},
		{Model: gorm.Model{ID: 2}, VisibilityType: "all", ModelName: "M2"},
	}

	result := FilterModelsByVisibility(context.Background(), models, 0)
	if len(result) != 2 {
		t.Errorf("expected 2, got %d", len(result))
	}
}

func TestCoverageFilterModelsByVisibility_GroupType_Visible(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.AIModel{Model: gorm.Model{ID: 10}, VisibilityType: "group", ModelName: "GroupModel"})
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: 10, GroupID: 1})

	models := []model.AIModel{
		{Model: gorm.Model{ID: 10}, VisibilityType: "group", ModelName: "GroupModel"},
	}

	result := FilterModelsByVisibility(context.Background(), models, 1)
	if len(result) != 1 {
		t.Errorf("expected 1 visible, got %d", len(result))
	}
}

func TestCoverageFilterModelsByVisibility_GroupType_NotVisible(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.AIModel{Model: gorm.Model{ID: 10}, VisibilityType: "group", ModelName: "GroupModel"})
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: 10, GroupID: 99})

	models := []model.AIModel{
		{Model: gorm.Model{ID: 10}, VisibilityType: "group", ModelName: "GroupModel"},
	}

	result := FilterModelsByVisibility(context.Background(), models, 1)
	if len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

func TestCoverageFilterModelsByVisibility_ZeroGroup(t *testing.T) {
	setupResolveCoverageDB(t)

	models := []model.AIModel{
		{Model: gorm.Model{ID: 1}, VisibilityType: "all", ModelName: "M1"},
		{Model: gorm.Model{ID: 2}, VisibilityType: "group", ModelName: "M2"},
	}

	// agentGroupID=0 时 group 类型不可见
	result := FilterModelsByVisibility(context.Background(), models, 0)
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
}

// ── FilterChannelsByVisibility ───────────────────────────

func TestCoverageFilterChannelsByVisibility_AllType(t *testing.T) {
	setupResolveCoverageDB(t)

	enabled := true
	channels := []model.AIChannel{
		{Model: gorm.Model{ID: 1}, VisibilityType: "all", Name: "Ch1", Enabled: &enabled},
	}

	result := FilterChannelsByVisibility(context.Background(), channels, 0)
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
}

func TestCoverageFilterChannelsByVisibility_GroupType(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "5", GroupID: 1, ValueJSON: "{}"})

	enabled := true
	channels := []model.AIChannel{
		{Model: gorm.Model{ID: 5}, VisibilityType: "group", Name: "Ch5", Enabled: &enabled},
	}

	result := FilterChannelsByVisibility(context.Background(), channels, 1)
	if len(result) != 1 {
		t.Errorf("expected 1 visible, got %d", len(result))
	}
}

// ── FilterPluginBundlesByVisibility ──────────────────────

func TestCoverageFilterPluginBundlesByVisibility_AllType(t *testing.T) {
	setupResolveCoverageDB(t)

	bundles := []model.PluginBundle{
		{ID: 1, VisibilityType: "all", Name: "B1"},
	}

	result := FilterPluginBundlesByVisibility(context.Background(), bundles, 0)
	if len(result) != 1 {
		t.Errorf("expected 1, got %d", len(result))
	}
}

func TestCoverageFilterPluginBundlesByVisibility_GroupType(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypePluginBundle, ConfigKey: "7", GroupID: 1, ValueJSON: "{}"})

	bundles := []model.PluginBundle{
		{ID: 7, VisibilityType: "group", Name: "B7"},
	}

	result := FilterPluginBundlesByVisibility(context.Background(), bundles, 1)
	if len(result) != 1 {
		t.Errorf("expected 1 visible, got %d", len(result))
	}
}

// ── ResolveVpcConfig ─────────────────────────────────────

func TestCoverageResolveVpcConfig_ZeroGroupID(t *testing.T) {
	setupResolveCoverageDB(t)

	globalSubnets := map[string][]string{"zone1": {"subnet-1"}}
	vpcID, subnets := ResolveVpcConfig(context.Background(), 0, "vpc-global", globalSubnets)
	if vpcID != "vpc-global" {
		t.Errorf("expected vpc-global, got %s", vpcID)
	}
	if len(subnets["zone1"]) != 1 {
		t.Errorf("unexpected subnets: %v", subnets)
	}
}

func TestCoverageResolveVpcConfig_WithGroupBinding(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")

	// 创建 VpcConfig
	model.DB(context.Background()).Create(&model.VpcConfig{Model: gorm.Model{ID: 100}, VpcId: "vpc-group", SubnetIds: `{"zone2":["subnet-2"]}`})
	// 绑定 vpc 到组
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeVPC, ConfigKey: "100", GroupID: 1, ValueJSON: "{}"})

	globalSubnets := map[string][]string{"zone1": {"subnet-1"}}
	vpcID, subnets := ResolveVpcConfig(context.Background(), 1, "vpc-global", globalSubnets)
	if vpcID != "vpc-group" {
		t.Errorf("expected vpc-group, got %s", vpcID)
	}
	if len(subnets["zone2"]) != 1 || subnets["zone2"][0] != "subnet-2" {
		t.Errorf("unexpected subnets: %v", subnets)
	}
}

func TestCoverageResolveVpcConfig_InheritFromAncestor(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")
	createGroupWithClosure(t, 2, 1, "Child")

	model.DB(context.Background()).Create(&model.VpcConfig{Model: gorm.Model{ID: 200}, VpcId: "vpc-root", SubnetIds: `{"z":["s1"]}`})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeVPC, ConfigKey: "200", GroupID: 1, ValueJSON: "{}"})

	globalSubnets := map[string][]string{"default": {"d1"}}
	vpcID, _ := ResolveVpcConfig(context.Background(), 2, "vpc-global", globalSubnets)
	if vpcID != "vpc-root" {
		t.Errorf("expected vpc-root (inherited), got %s", vpcID)
	}
}

func TestCoverageResolveVpcConfig_NoBinding(t *testing.T) {
	setupResolveCoverageDB(t)

	createGroupWithClosure(t, 1, 0, "Root")

	globalSubnets := map[string][]string{"z": {"s"}}
	vpcID, _ := ResolveVpcConfig(context.Background(), 1, "vpc-global", globalSubnets)
	if vpcID != "vpc-global" {
		t.Errorf("expected fallback vpc-global, got %s", vpcID)
	}
}

// ── IsRoleVisibleToGroups ────────────────────────────────

func TestCoverageIsRoleVisibleToGroups_AllType(t *testing.T) {
	setupResolveCoverageDB(t)

	model.DB(context.Background()).Create(&model.OpenClawRole{ID: 1, Name: "admin", VisibilityType: "all", Description: "", Soul: ""})

	visible := IsRoleVisibleToGroups(context.Background(), 1, []uint{1, 2})
	if !visible {
		t.Error("expected visible for type=all")
	}
}

func TestCoverageIsRoleVisibleToGroups_GroupType_Visible(t *testing.T) {
	setupResolveCoverageDB(t)

	model.DB(context.Background()).Create(&model.OpenClawRole{ID: 2, Name: "dev", VisibilityType: "group", Description: "", Soul: ""})
	model.DB(context.Background()).Create(&model.RoleVisibilityGroup{OpenClawRoleID: 2, GroupID: 1})

	visible := IsRoleVisibleToGroups(context.Background(), 2, []uint{1})
	if !visible {
		t.Error("expected visible")
	}
}

func TestCoverageIsRoleVisibleToGroups_GroupType_NotVisible(t *testing.T) {
	setupResolveCoverageDB(t)

	model.DB(context.Background()).Create(&model.OpenClawRole{ID: 3, Name: "pm", VisibilityType: "group", Description: "", Soul: ""})
	model.DB(context.Background()).Create(&model.RoleVisibilityGroup{OpenClawRoleID: 3, GroupID: 99})

	visible := IsRoleVisibleToGroups(context.Background(), 3, []uint{1, 2})
	if visible {
		t.Error("expected not visible")
	}
}

func TestCoverageIsRoleVisibleToGroups_EmptyGroups(t *testing.T) {
	setupResolveCoverageDB(t)

	model.DB(context.Background()).Create(&model.OpenClawRole{ID: 4, Name: "role4", VisibilityType: "group", Description: "", Soul: ""})

	visible := IsRoleVisibleToGroups(context.Background(), 4, nil)
	if !visible {
		t.Error("expected visible when groupIDs is empty (no restriction)")
	}
}

// ── parseUintStr ─────────────────────────────────────────

func TestCoverageParseUintStr_Valid(t *testing.T) {
	var out uint
	n, err := parseUintStr("12345", &out)
	if err != nil {
		t.Fatalf("parseUintStr: %v", err)
	}
	if n != 5 || out != 12345 {
		t.Errorf("unexpected: n=%d, out=%d", n, out)
	}
}

func TestCoverageParseUintStr_Invalid(t *testing.T) {
	var out uint
	_, err := parseUintStr("abc", &out)
	if err == nil {
		t.Error("expected error for non-numeric")
	}
}

func TestCoverageParseUintStr_Partial(t *testing.T) {
	var out uint
	n, err := parseUintStr("42abc", &out)
	if err != nil {
		t.Fatalf("parseUintStr partial: %v", err)
	}
	if n != 2 || out != 42 {
		t.Errorf("unexpected: n=%d, out=%d", n, out)
	}
}

// ── IsRoleGloballyVisible ────────────────────────────────

func TestIsRoleGloballyVisible_AllType(t *testing.T) {
	setupResolveCoverageDB(t)
	model.DB(context.Background()).Create(&model.OpenClawRole{
		ID: 1, Name: "全局角色", VisibilityType: "all",
	})
	if !IsRoleGloballyVisible(context.Background(), 1) {
		t.Error("expected visibility_type='all' role to be globally visible")
	}
}

func TestIsRoleGloballyVisible_EmptyType(t *testing.T) {
	setupResolveCoverageDB(t)
	model.DB(context.Background()).Create(&model.OpenClawRole{
		ID: 2, Name: "默认角色", VisibilityType: "",
	})
	if !IsRoleGloballyVisible(context.Background(), 2) {
		t.Error("expected visibility_type='' role to be globally visible")
	}
}

func TestIsRoleGloballyVisible_GroupType(t *testing.T) {
	setupResolveCoverageDB(t)
	model.DB(context.Background()).Create(&model.OpenClawRole{
		ID: 3, Name: "分组角色", VisibilityType: "group",
	})
	if IsRoleGloballyVisible(context.Background(), 3) {
		t.Error("expected visibility_type='group' role to NOT be globally visible")
	}
}

func TestIsRoleGloballyVisible_NotFound(t *testing.T) {
	setupResolveCoverageDB(t)
	if IsRoleGloballyVisible(context.Background(), 999) {
		t.Error("expected non-existent role to return false")
	}
}
