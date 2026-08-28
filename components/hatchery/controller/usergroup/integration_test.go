package usergroup

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建内存 SQLite 数据库，自动建表并注入到 model.DB。
// 返回清理函数（测试结束后调用）。
func setupTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	// 建表
	err = db.AutoMigrate(
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupConfigBinding{},
		&model.GroupClosure{},
		&model.SiteConfig{},
		&model.AIChannel{},
		&model.PluginBundle{},
		&model.McpServer{},
		&model.Instance{},           // CanDeleteUserGroup 会查 instances.group_id
		&model.User{},               // GetGroupMembersPaged JOIN users
		&model.GroupConfigBinding{}, // DeleteUserGroupForOneIDDept 事务内清理配置绑定
		&model.TagVisibilityGroup{},
	)
	if err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	// 注入
	restore := model.UseDBForTest(db)

	return func() {
		restore()
	}
}

// seedGroupHierarchy 种入测试用多层级分组和 closure 数据：
//
//	研发部(1)
//	  └── 后端组(2)
//	        └── Java组(3)
//	设计部(4)
func seedGroupHierarchy(t *testing.T) {
	t.Helper()
	groups := []model.UserGroup{
		{ID: 1, Name: "研发部"},
		{ID: 2, Name: "后端组"},
		{ID: 3, Name: "Java组"},
		{ID: 4, Name: "设计部"},
	}
	for _, g := range groups {
		model.DB(context.Background()).Create(&g)
	}

	// closure: 自指 + 父子 + 祖孙
	closures := []model.GroupClosure{
		{AncestorID: 1, DescendantID: 1, Depth: 0},
		{AncestorID: 2, DescendantID: 2, Depth: 0},
		{AncestorID: 3, DescendantID: 3, Depth: 0},
		{AncestorID: 4, DescendantID: 4, Depth: 0},
		{AncestorID: 1, DescendantID: 2, Depth: 1},
		{AncestorID: 2, DescendantID: 3, Depth: 1},
		{AncestorID: 1, DescendantID: 3, Depth: 2},
	}
	for _, c := range closures {
		model.DB(context.Background()).Create(&c)
	}
}

// TestGetAncestorIDs_WithClosure 测试祖先链查询（有 closure 数据）
func TestGetAncestorIDs_WithClosure(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	seedGroupHierarchy(t)

	// Java组(3) 的祖先链应为 [3, 2, 1]（自己 → 父 → 祖父）
	ancestors, err := GetAncestorIDs(context.Background(), 3)
	if err != nil {
		t.Fatalf("GetAncestorIDs: %v", err)
	}
	if len(ancestors) != 3 {
		t.Fatalf("expected 3 ancestors, got %d: %v", len(ancestors), ancestors)
	}
	// ancestors[0] = 自己(depth=0)
	if ancestors[0] != 3 {
		t.Errorf("ancestors[0] = %d, want 3", ancestors[0])
	}
}

// TestResolvePolicyInt_InheritanceChain 测试策略继承链
func TestResolvePolicyInt_InheritanceChain(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	seedGroupHierarchy(t)

	// 只在研发部(1)配了 token_quota_day = 200000
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: ConfigTypePolicy,
		ConfigKey:  "token_quota_day",
		GroupID:    1,
		ValueJSON:  `{"value": 200000}`,
	})

	// Java组(3) 的祖先链 [3, 2, 1]
	ancestors, _ := GetAncestorIDs(context.Background(), 3)

	val, source, err := ResolvePolicyInt(context.Background(), "token_quota_day", ancestors, 500000)
	if err != nil {
		t.Fatalf("ResolvePolicyInt: %v", err)
	}
	if val != 200000 {
		t.Errorf("value = %d, want 200000", val)
	}
	if source.Type != SourceInherited {
		t.Errorf("source.Type = %q, want %q", source.Type, SourceInherited)
	}
	if source.GroupID != 1 {
		t.Errorf("source.GroupID = %d, want 1", source.GroupID)
	}
}

// TestResolvePolicyInt_LocalOverride 测试本组配置覆盖祖先
func TestResolvePolicyInt_LocalOverride(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	seedGroupHierarchy(t)

	// 研发部(1)配 200000，后端组(2)配 100000
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: ConfigTypePolicy, ConfigKey: "token_quota_day",
		GroupID: 1, ValueJSON: `{"value": 200000}`,
	})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: ConfigTypePolicy, ConfigKey: "token_quota_day",
		GroupID: 2, ValueJSON: `{"value": 100000}`,
	})

	// 后端组(2) 的祖先链 [2, 1]
	ancestors, _ := GetAncestorIDs(context.Background(), 2)

	val, source, err := ResolvePolicyInt(context.Background(), "token_quota_day", ancestors, 500000)
	if err != nil {
		t.Fatalf("ResolvePolicyInt: %v", err)
	}
	if val != 100000 {
		t.Errorf("value = %d, want 100000 (local override)", val)
	}
	if source.Type != SourceLocal {
		t.Errorf("source.Type = %q, want %q", source.Type, SourceLocal)
	}
}

// TestResolvePolicyInt_FallbackToSiteConfig 测试回退到全局默认
func TestResolvePolicyInt_FallbackToSiteConfig(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	seedGroupHierarchy(t)

	// 不配任何策略
	ancestors, _ := GetAncestorIDs(context.Background(), 3)
	val, source, err := ResolvePolicyInt(context.Background(), "token_quota_day", ancestors, 500000)
	if err != nil {
		t.Fatalf("ResolvePolicyInt: %v", err)
	}
	if val != 500000 {
		t.Errorf("value = %d, want 500000 (fallback)", val)
	}
	if source.Type != SourceSiteDefault {
		t.Errorf("source.Type = %q, want %q", source.Type, SourceSiteDefault)
	}
}

// TestResolvePolicyBool_Inheritance 测试布尔策略继承
func TestResolvePolicyBool_Inheritance(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	seedGroupHierarchy(t)

	// 研发部开启终端
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: ConfigTypePolicy, ConfigKey: "agent_terminal",
		GroupID: 1, ValueJSON: `{"enabled": true}`,
	})

	ancestors, _ := GetAncestorIDs(context.Background(), 3) // Java组
	val, source, _ := ResolvePolicyBool(context.Background(), "agent_terminal", ancestors, false)
	if val != true {
		t.Errorf("value = %v, want true (inherited from 研发部)", val)
	}
	if source.Type != SourceInherited {
		t.Errorf("source.Type = %q, want %q", source.Type, SourceInherited)
	}
}

// TestResolveImageTypes_NoRestriction 无限制时所有类型可见
func TestResolveImageTypes_NoRestriction(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	allTypes := []string{"openclaw", "hermes", "lightclaw_ace"}
	result, err := ResolveImageTypes(context.Background(), []uint{1, 2}, allTypes)
	if err != nil {
		t.Fatalf("ResolveImageTypes: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 types, got %d", len(result))
	}
}

// TestResolveImageTypes_WithRestriction 有限制时按组过滤
func TestResolveImageTypes_WithRestriction(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()
	seedGroupHierarchy(t)

	// hermes 仅对 研发部(1) 可见
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: ConfigTypeImageType, ConfigKey: "hermes",
		GroupID: 1, ValueJSON: "{}",
	})

	allTypes := []string{"openclaw", "hermes", "lightclaw_ace"}

	// 设计部(4) 不应看到 hermes
	result, _ := ResolveImageTypes(context.Background(), []uint{4}, allTypes)
	for _, r := range result {
		if r == "hermes" {
			t.Errorf("设计部不应看到 hermes")
		}
	}

	// 研发部(1) 应看到 hermes
	result, _ = ResolveImageTypes(context.Background(), []uint{1}, allTypes)
	found := false
	for _, r := range result {
		if r == "hermes" {
			found = true
		}
	}
	if !found {
		t.Errorf("研发部应看到 hermes")
	}
}
