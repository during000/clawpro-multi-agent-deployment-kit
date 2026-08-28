package usergroup

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAdminQueryCoverageDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.UserGroup{}, &model.GroupClosure{}, &model.UserGroupMember{},
		&model.GroupConfigBinding{}, &model.User{}, &model.AIModel{},
		&model.ModelVisibilityGroup{}, &model.AIChannel{}, &model.AIImage{},
		&model.SiteConfig{}, &model.Instance{}, &model.VpcConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// ── HasGlobalModel ───────────────────────────────────────

func TestCoverageHasGlobalModel_NoModels(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	if HasGlobalModel(context.Background()) {
		t.Error("expected false when no models")
	}
}

func TestCoverageHasGlobalModel_DisabledModel(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.AIModel{Provider: "openai", ModelID: "gpt-4", ModelName: "GPT-4", Enabled: false, VisibilityType: "all"})

	if HasGlobalModel(context.Background()) {
		t.Error("expected false for disabled model")
	}
}

func TestCoverageHasGlobalModel_EnabledAllVisible(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.AIModel{Provider: "openai", ModelID: "gpt-4", ModelName: "GPT-4", Enabled: true, Visible: true, VisibilityType: "all"})

	if !HasGlobalModel(context.Background()) {
		t.Error("expected true for enabled all-visible model")
	}
}

func TestCoverageHasGlobalModel_GroupVisible(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.AIModel{Provider: "openai", ModelID: "gpt-4", ModelName: "GPT-4", Enabled: true, Visible: true, VisibilityType: "group"})

	if HasGlobalModel(context.Background()) {
		t.Error("expected false for group-visible model")
	}
}

func TestCoverageHasGlobalModel_BuiltinExcluded(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	// 内置占位模型不算
	model.DB(context.Background()).Create(&model.AIModel{Provider: "hatchery", ModelID: "custom", ModelName: "Custom", Enabled: true, Visible: true, VisibilityType: "all"})

	if HasGlobalModel(context.Background()) {
		t.Error("expected false for builtin placeholder model")
	}
}

// ── HasGlobalChannel ─────────────────────────────────────

func TestCoverageHasGlobalChannel_NoChannels(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	if HasGlobalChannel(context.Background()) {
		t.Error("expected false when no channels")
	}
}

func TestCoverageHasGlobalChannel_DisabledChannel(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	enabled := false
	model.DB(context.Background()).Create(&model.AIChannel{Name: "WeChat", Enabled: &enabled, VisibilityType: "all"})

	if HasGlobalChannel(context.Background()) {
		t.Error("expected false for disabled channel")
	}
}

func TestCoverageHasGlobalChannel_EnabledAllVisible(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	enabled := true
	model.DB(context.Background()).Create(&model.AIChannel{Name: "WeChat", Enabled: &enabled, VisibilityType: "all"})

	if !HasGlobalChannel(context.Background()) {
		t.Error("expected true for enabled all-visible channel")
	}
}

func TestCoverageHasGlobalChannel_GroupVisible(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	enabled := true
	model.DB(context.Background()).Create(&model.AIChannel{Name: "WeChat", Enabled: &enabled, VisibilityType: "group"})

	if HasGlobalChannel(context.Background()) {
		t.Error("expected false for group-visible channel")
	}
}

// ── HasGlobalNetwork ─────────────────────────────────────

func TestCoverageHasGlobalNetwork_AlwaysTrue(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	// 2026-05-13：VPC 自动分配，HasGlobalNetwork 始终返回 true
	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, VpcId: ""})

	if !HasGlobalNetwork(context.Background()) {
		t.Error("expected true: VPC auto-assigned, always returns true")
	}
}

func TestCoverageHasGlobalNetwork_WithVpc(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, VpcId: "vpc-12345"})

	if !HasGlobalNetwork(context.Background()) {
		t.Error("expected true when VPC configured")
	}
}

// ── HasGlobalImage ───────────────────────────────────────

func TestCoverageHasGlobalImage_NoImages(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	if HasGlobalImage(context.Background()) {
		t.Error("expected false when no images")
	}
}

func TestCoverageHasGlobalImage_EnabledUnrestricted(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-1", AgentType: "openclaw", Enabled: true})

	if !HasGlobalImage(context.Background()) {
		t.Error("expected true for enabled unrestricted image")
	}
}

func TestCoverageHasGlobalImage_RestrictedOnly(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-1", AgentType: "openclaw", Enabled: true})
	// 限制 openclaw 类型
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeImageType, ConfigKey: "openclaw", GroupID: 1, ValueJSON: "{}"})

	if HasGlobalImage(context.Background()) {
		t.Error("expected false when all enabled images are restricted")
	}
}

func TestCoverageHasGlobalImage_MixedTypes(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-1", AgentType: "openclaw", Enabled: true})
	model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-2", AgentType: "browser", Enabled: true})
	// 只限制 openclaw
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeImageType, ConfigKey: "openclaw", GroupID: 1, ValueJSON: "{}"})

	if !HasGlobalImage(context.Background()) {
		t.Error("expected true: browser is unrestricted and enabled")
	}
}

// ── batchGetAncestorsForHealth ───────────────────────────

func TestCoverageBatchGetAncestorsForHealth_Empty(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	result := batchGetAncestorsForHealth(context.Background(), nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestCoverageBatchGetAncestorsForHealth_WithData(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})

	result := batchGetAncestorsForHealth(context.Background(), []uint{2})
	if len(result[2]) != 2 {
		t.Errorf("expected 2 ancestors for group 2, got %d: %v", len(result[2]), result[2])
	}
}

func TestCoverageBatchGetAncestorsForHealth_MissingGroup(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	// 组 99 没有 closure 行，应该 fallback 为 [99]
	result := batchGetAncestorsForHealth(context.Background(), []uint{99})
	if len(result[99]) != 1 || result[99][0] != 99 {
		t.Errorf("expected [99] for missing group, got %v", result[99])
	}
}

// ── hasModelBinding ──────────────────────────────────────

func TestCoverageHasModelBinding_Empty(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	if hasModelBinding(context.Background(), nil) {
		t.Error("expected false for nil")
	}
	if hasModelBinding(context.Background(), []uint{1, 2}) {
		t.Error("expected false when no bindings")
	}
}

func TestCoverageHasModelBinding_Found(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: 1, GroupID: 5})

	if !hasModelBinding(context.Background(), []uint{5}) {
		t.Error("expected true")
	}
}

// ── hasChannelBinding ────────────────────────────────────

func TestCoverageHasChannelBinding_Empty(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	if hasChannelBinding(context.Background(), nil) {
		t.Error("expected false for nil")
	}
}

func TestCoverageHasChannelBinding_Found(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "1", GroupID: 3, ValueJSON: "{}"})

	if !hasChannelBinding(context.Background(), []uint{3}) {
		t.Error("expected true")
	}
}

func TestCoverageHasChannelBinding_NotFound(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "1", GroupID: 99, ValueJSON: "{}"})

	if hasChannelBinding(context.Background(), []uint{1, 2}) {
		t.Error("expected false when not matching group")
	}
}

// ── hasImageTypeBinding ──────────────────────────────────────

func TestCoverageHasImageBinding_Empty(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	if hasImageTypeBinding(context.Background(), nil) {
		t.Error("expected false for nil")
	}
}

func TestCoverageHasImageBinding_Found(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeImageType, ConfigKey: "openclaw", GroupID: 5, ValueJSON: "{}"})

	if !hasImageTypeBinding(context.Background(), []uint{5}) {
		t.Error("expected true")
	}
}

// ── countDescendantMembers ───────────────────────────────

func TestCoverageCountDescendantMembers_NoMembers(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual"})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})

	count := countDescendantMembers(context.Background(), 1)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestCoverageCountDescendantMembers_WithMembers(t *testing.T) {
	setupAdminQueryCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "Child", Source: "manual", ParentID: 1})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})

	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: 1, UserID: 10, Source: "manual"})
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: 2, UserID: 20, Source: "manual"})

	count := countDescendantMembers(context.Background(), 1)
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestGetGroupMembersPaged_MarksMembersFromDescendants(t *testing.T) {
	setupAdminQueryCoverageDB(t)
	ctx := context.Background()

	parent := model.UserGroup{ID: 1, Name: "Parent", FullPath: "Parent", Source: model.GroupSourceManual}
	child := model.UserGroup{ID: 2, Name: "Child", FullPath: "Parent/Child", ParentID: 1, Source: model.GroupSourceManual}
	if err := model.DB(ctx).Create(&[]model.UserGroup{parent, child}).Error; err != nil {
		t.Fatalf("create groups: %v", err)
	}
	if err := model.DB(ctx).Create(&[]model.GroupClosure{
		{AncestorID: 1, DescendantID: 1, Depth: 0},
		{AncestorID: 1, DescendantID: 2, Depth: 1},
		{AncestorID: 2, DescendantID: 2, Depth: 0},
	}).Error; err != nil {
		t.Fatalf("create closures: %v", err)
	}
	users := []model.User{{Username: "direct-user"}, {Username: "descendant-user"}}
	if err := model.DB(ctx).Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	directUser, descendantUser := users[0], users[1]
	if err := model.DB(ctx).Create(&[]model.UserGroupMember{
		{UserGroupID: 1, UserID: directUser.ID, Source: model.MemberSourceManual},
		{UserGroupID: 2, UserID: descendantUser.ID, Source: model.MemberSourceManual},
	}).Error; err != nil {
		t.Fatalf("create memberships: %v", err)
	}

	resp, err := GetGroupMembersPaged(ctx, MembersOptions{
		GroupID: 1, Page: 1, PageSize: 200, IncludeDescendants: true,
	})
	if err != nil {
		t.Fatalf("GetGroupMembersPaged: %v", err)
	}
	if len(resp.Members) != 2 {
		t.Fatalf("members count = %d, want 2", len(resp.Members))
	}
	for _, member := range resp.Members {
		switch member.UserID {
		case directUser.ID:
			if member.FromDescendant {
				t.Error("direct member must not be marked as from descendant")
			}
		case descendantUser.ID:
			if !member.FromDescendant {
				t.Error("descendant-only member must be marked as from descendant")
			}
		default:
			t.Fatalf("unexpected member: %d", member.UserID)
		}
	}
}

// ── normalizeSources ─────────────────────────────────────

func TestCoverageNormalizeSources_Empty(t *testing.T) {
	result := normalizeSources(nil)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestCoverageNormalizeSources_WithValues(t *testing.T) {
	result := normalizeSources([]string{"manual"})
	if len(result) != 1 || result[0] != "manual" {
		t.Errorf("expected [manual], got %v", result)
	}
}

func TestCoverageNormalizeSources_InvalidFiltered(t *testing.T) {
	result := normalizeSources([]string{"invalid_source"})
	if result != nil {
		t.Errorf("expected nil for invalid sources, got %v", result)
	}
}

func TestCoverageNormalizeSources_Mixed(t *testing.T) {
	result := normalizeSources([]string{"manual", "invalid", "oneid_dept", "manual"})
	if len(result) != 2 {
		t.Errorf("expected 2 (deduped valid), got %d: %v", len(result), result)
	}
}

func TestCoverageNormalizeSources_TrimAndLower(t *testing.T) {
	result := normalizeSources([]string{" Manual ", "  ONEID_DEPT "})
	if len(result) != 2 {
		t.Errorf("expected 2 after trim/lower, got %d: %v", len(result), result)
	}
}
