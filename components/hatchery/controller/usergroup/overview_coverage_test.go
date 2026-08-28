package usergroup

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupOverviewCoverageDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.UserGroup{}, &model.GroupClosure{}, &model.UserGroupMember{},
		&model.GroupConfigBinding{}, &model.User{}, &model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// ── ResolveAdditiveOverview ──────────────────────────────

func TestCoverageResolveAdditiveOverview_EmptyAncestors(t *testing.T) {
	setupOverviewCoverageDB(t)

	result, err := ResolveAdditiveOverview(context.Background(), ConfigTypeChannel, 1, nil, nil)
	if err != nil {
		t.Fatalf("ResolveAdditiveOverview: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0, got %d", result.Total)
	}
}

func TestCoverageResolveAdditiveOverview_LocalBinding(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "10", GroupID: 1, ValueJSON: "{}"})

	result, err := ResolveAdditiveOverview(context.Background(), ConfigTypeChannel, 1, []uint{1}, func(key string) string {
		return "Channel-" + key
	})
	if err != nil {
		t.Fatalf("ResolveAdditiveOverview: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1, got %d", result.Total)
	}
	if result.Items[0].Source.Type != SourceLocal {
		t.Errorf("expected local, got %s", result.Items[0].Source.Type)
	}
	if result.Items[0].ResourceName != "Channel-10" {
		t.Errorf("expected Channel-10, got %s", result.Items[0].ResourceName)
	}
}

func TestCoverageResolveAdditiveOverview_InheritedBinding(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "Child", Source: "manual", FullPath: "Root/Child", ParentID: 1})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "20", GroupID: 1, ValueJSON: "{}"})

	// 从 Child 视角看，祖先是 [2, 1]，binding 在 1 上
	result, err := ResolveAdditiveOverview(context.Background(), ConfigTypeChannel, 2, []uint{2, 1}, nil)
	if err != nil {
		t.Fatalf("ResolveAdditiveOverview: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1, got %d", result.Total)
	}
	if result.Items[0].Source.Type != SourceInherited {
		t.Errorf("expected inherited, got %s", result.Items[0].Source.Type)
	}
}

func TestCoverageResolveAdditiveOverview_LocalOverridesInherited(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "Child", Source: "manual", FullPath: "Root/Child", ParentID: 1})
	// 同一资源在两层都绑定
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "30", GroupID: 1, ValueJSON: "{}"})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "30", GroupID: 2, ValueJSON: "{}"})

	result, _ := ResolveAdditiveOverview(context.Background(), ConfigTypeChannel, 2, []uint{2, 1}, nil)
	if result.Total != 1 {
		t.Errorf("expected 1 (deduped), got %d", result.Total)
	}
	if result.Items[0].Source.Type != SourceLocal {
		t.Errorf("expected local (closer priority), got %s", result.Items[0].Source.Type)
	}
}

func TestCoverageResolveAdditiveOverview_NilResourceNameFn(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{ConfigType: model.ConfigTypeChannel, ConfigKey: "42", GroupID: 1, ValueJSON: "{}"})

	result, _ := ResolveAdditiveOverview(context.Background(), ConfigTypeChannel, 1, []uint{1}, nil)
	if result.Items[0].ResourceName != "42" {
		t.Errorf("expected raw key '42', got '%s'", result.Items[0].ResourceName)
	}
}

// ── ResolvePolicyOverview ────────────────────────────────

func TestCoverageResolvePolicyOverview_NoBindings(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1})

	sc := model.GetSiteConfig(context.Background())
	items, err := ResolvePolicyOverview(context.Background(), 1, []uint{1}, &sc)
	if err != nil {
		t.Fatalf("ResolvePolicyOverview: %v", err)
	}
	// 应该返回所有 policyKeyOrder 中的项，全部为 site_default
	if len(items) != len(policyKeyOrder) {
		t.Errorf("expected %d items, got %d", len(policyKeyOrder), len(items))
	}
	for _, item := range items {
		if item.Source.Type != SourceSiteDefault {
			t.Errorf("expected site_default for %s, got %s", item.Key, item.Source.Type)
		}
	}
}

func TestCoverageResolvePolicyOverview_WithLocalBinding(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyTokenQuotaDay,
		GroupID:    1,
		ValueJSON:  `{"value":500}`,
	})

	sc := model.GetSiteConfig(context.Background())
	items, err := ResolvePolicyOverview(context.Background(), 1, []uint{1}, &sc)
	if err != nil {
		t.Fatalf("ResolvePolicyOverview: %v", err)
	}

	// 找到 token_quota_day 项
	found := false
	for _, item := range items {
		if item.Key == PolicyKeyTokenQuotaDay {
			found = true
			if item.Source.Type != SourceLocal {
				t.Errorf("expected local, got %s", item.Source.Type)
			}
			break
		}
	}
	if !found {
		t.Error("token_quota_day not found in results")
	}
}

func TestCoverageResolvePolicyOverview_InheritedBinding(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "Child", Source: "manual", FullPath: "Root/Child", ParentID: 1})
	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyInstanceQuota,
		GroupID:    1,
		ValueJSON:  `{"value":10}`,
	})

	sc := model.GetSiteConfig(context.Background())
	items, err := ResolvePolicyOverview(context.Background(), 2, []uint{2, 1}, &sc)
	if err != nil {
		t.Fatalf("ResolvePolicyOverview: %v", err)
	}

	for _, item := range items {
		if item.Key == PolicyKeyInstanceQuota {
			if item.Source.Type != SourceInherited {
				t.Errorf("expected inherited, got %s", item.Source.Type)
			}
			return
		}
	}
	t.Error("instance_quota not found in results")
}

// ── parseValueJSON ───────────────────────────────────────

func TestCoverageParseValueJSON_Int(t *testing.T) {
	val := parseValueJSON(`{"value":42}`, PolicyValueInt)
	if val != 42 {
		t.Errorf("expected 42, got %v", val)
	}
}

func TestCoverageParseValueJSON_Bool(t *testing.T) {
	val := parseValueJSON(`{"enabled":true}`, PolicyValueBool)
	if val != true {
		t.Errorf("expected true, got %v", val)
	}
}

func TestCoverageParseValueJSON_String(t *testing.T) {
	val := parseValueJSON(`{"value":"hello"}`, PolicyValueString)
	if val != "hello" {
		t.Errorf("expected hello, got %v", val)
	}
}

func TestCoverageParseValueJSON_Invalid(t *testing.T) {
	val := parseValueJSON(`invalid`, PolicyValueInt)
	if val != nil {
		t.Errorf("expected nil for invalid JSON, got %v", val)
	}
}

// ── batchGetGroupNames ───────────────────────────────────

func TestCoverageBatchGetGroupNames_Empty(t *testing.T) {
	setupOverviewCoverageDB(t)

	result := batchGetGroupNames(context.Background(), nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestCoverageBatchGetGroupNames_Found(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.UserGroup{ID: 2, Name: "Child", Source: "manual", FullPath: "Root/Child"})

	result := batchGetGroupNames(context.Background(), []uint{1, 2})
	if result[1] == "" || result[2] == "" {
		t.Errorf("expected non-empty names, got %v", result)
	}
}

// ── ResolvePolicyOverview token_quota_rules 兼容 ────────────

func TestCoverageResolvePolicyOverview_RulesCompat_DayFromRules(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1})
	// 只有 token_quota_rules binding（无 token_quota_day binding）
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyTokenQuotaRules,
		GroupID:    1,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":200000}]"}`,
	})

	sc := model.GetSiteConfig(context.Background())
	items, err := ResolvePolicyOverview(context.Background(), 1, []uint{1}, &sc)
	if err != nil {
		t.Fatalf("ResolvePolicyOverview: %v", err)
	}

	for _, item := range items {
		if item.Key == PolicyKeyTokenQuotaDay {
			if item.Source.Type != SourceLocal {
				t.Errorf("expected local, got %s", item.Source.Type)
			}
			// value 应为反推出的 day 值
			if v, ok := item.Value.(int); !ok || v != 200000 {
				t.Errorf("expected value 200000, got %v", item.Value)
			}
			return
		}
	}
	t.Error("token_quota_day not found in results")
}

func TestCoverageResolvePolicyOverview_RulesCompat_NoDayRule(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1})
	// token_quota_rules = [] (无 day 规则)
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyTokenQuotaRules,
		GroupID:    1,
		ValueJSON:  `{"value":"[]"}`,
	})

	sc := model.GetSiteConfig(context.Background())
	items, err := ResolvePolicyOverview(context.Background(), 1, []uint{1}, &sc)
	if err != nil {
		t.Fatalf("ResolvePolicyOverview: %v", err)
	}

	for _, item := range items {
		if item.Key == PolicyKeyTokenQuotaDay {
			if item.Source.Type != SourceLocal {
				t.Errorf("expected local (from rules compat), got %s", item.Source.Type)
			}
			// value 应为 -1（无 day 规则）
			if v, ok := item.Value.(int); !ok || v != -1 {
				t.Errorf("expected value -1, got %v", item.Value)
			}
			return
		}
	}
	t.Error("token_quota_day not found in results")
}

func TestCoverageResolvePolicyOverview_RulesCompat_SkipExistingDay(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1})
	// 该组仍有旧 token_quota_day binding，不应从 rules 反推
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyTokenQuotaDay,
		GroupID:    1,
		ValueJSON:  `{"value":100000}`,
	})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyTokenQuotaRules,
		GroupID:    1,
		ValueJSON:  `{"value":"[{\"mode\":\"day\",\"limit\":999999}]"}`,
	})

	sc := model.GetSiteConfig(context.Background())
	items, err := ResolvePolicyOverview(context.Background(), 1, []uint{1}, &sc)
	if err != nil {
		t.Fatalf("ResolvePolicyOverview: %v", err)
	}

	for _, item := range items {
		if item.Key == PolicyKeyTokenQuotaDay {
			// 与 LLM proxy 一致：rules 优先于旧 token_quota_day，day 展示从 rules 反推。
			if v, ok := item.Value.(int); !ok || v != 999999 {
				t.Errorf("expected value 999999 (from rules), got %v", item.Value)
			}
			return
		}
	}
	t.Error("token_quota_day not found in results")
}

func TestCoverageResolvePolicyOverview_RulesCompat_RulesFromLegacyDay(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1})
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy,
		ConfigKey:  PolicyKeyTokenQuotaDay,
		GroupID:    1,
		ValueJSON:  `{"value":123456}`,
	})

	sc := model.GetSiteConfig(context.Background())
	items, err := ResolvePolicyOverview(context.Background(), 1, []uint{1}, &sc)
	if err != nil {
		t.Fatalf("ResolvePolicyOverview: %v", err)
	}

	for _, item := range items {
		if item.Key == PolicyKeyTokenQuotaRules {
			if item.Source.Type != SourceLocal {
				t.Errorf("expected local, got %s", item.Source.Type)
			}
			if v, ok := item.Value.(string); !ok || v != `[{"mode":"day","limit":123456}]` {
				t.Errorf("expected rules from legacy day, got %#v", item.Value)
			}
			return
		}
	}
	t.Error("token_quota_rules not found in results")
}

func TestCoverageResolvePolicyOverview_RulesCompat_SiteDefaultRulesFromLegacyDay(t *testing.T) {
	setupOverviewCoverageDB(t)

	model.DB(context.Background()).Create(&model.UserGroup{ID: 1, Name: "Root", Source: "manual", FullPath: "Root"})
	model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, DefaultTokenQuotaDay: 777, DefaultTokenQuotaRules: ""})

	sc := model.GetSiteConfig(context.Background())
	items, err := ResolvePolicyOverview(context.Background(), 1, []uint{1}, &sc)
	if err != nil {
		t.Fatalf("ResolvePolicyOverview: %v", err)
	}

	for _, item := range items {
		if item.Key == PolicyKeyTokenQuotaRules {
			if item.Source.Type != SourceSiteDefault {
				t.Errorf("expected site default, got %s", item.Source.Type)
			}
			if v, ok := item.Value.(string); !ok || v != `[{"mode":"day","limit":777}]` {
				t.Errorf("expected site default rules from legacy day, got %#v", item.Value)
			}
			return
		}
	}
	t.Error("token_quota_rules not found in results")
}
