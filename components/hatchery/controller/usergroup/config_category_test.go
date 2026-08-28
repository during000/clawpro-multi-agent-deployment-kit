package usergroup

import (
	"testing"
)

// ── ConfigCategoryList ───────────────────────────────────

func TestCoverageConfigCategoryList_Count(t *testing.T) {
	if len(ConfigCategoryList) != 13 {
		t.Errorf("expected 13 categories, got %d", len(ConfigCategoryList))
	}
}

func TestCoverageConfigCategoryList_Keys(t *testing.T) {
	expectedKeys := map[string]bool{
		CategoryKeyChargeType:      true,
		CategoryKeyModel:           true,
		CategoryKeyResourcePolicy:  true,
		CategoryKeyChannel:         true,
		CategoryKeySkill:           true,
		CategoryKeyAgentTool:       true,
		CategoryKeyMemory:          true,
		CategoryKeyDrive:           true,
		CategoryKeyImageType:       true,
		CategoryKeyNetwork:         true,
		CategoryKeyCLS:             true,
		CategoryKeyAIAgentSecurity: true,
		CategoryKeyPlatformPolicy:  true,
	}

	for _, cat := range ConfigCategoryList {
		if !expectedKeys[cat.Key] {
			t.Errorf("unexpected category key: %s", cat.Key)
		}
		if cat.Label == "" {
			t.Errorf("category %s has empty label", cat.Key)
		}
		if cat.Description == "" {
			t.Errorf("category %s has empty description", cat.Key)
		}
		if cat.Icon == "" {
			t.Errorf("category %s has empty icon", cat.Key)
		}
	}
}

// ── ConfigTypeMeta ───────────────────────────────────────

func TestCoverageConfigTypes_Entries(t *testing.T) {
	if len(ConfigTypes) != 6 {
		t.Errorf("expected 6 config types, got %d", len(ConfigTypes))
	}

	// 验证加法型
	for _, ct := range []string{ConfigTypeChannel, ConfigTypePluginBundle, ConfigTypeMCP, ConfigTypeImageType, ConfigTypeCLSCollectScope} {
		meta, ok := ConfigTypes[ct]
		if !ok {
			t.Errorf("missing config type: %s", ct)
			continue
		}
		if meta.Cardinality != CardinalityAdditive {
			t.Errorf("%s should be additive, got %s", ct, meta.Cardinality)
		}
	}

	// 验证策略型
	meta, ok := ConfigTypes[ConfigTypePolicy]
	if !ok {
		t.Fatal("missing policy config type")
	}
	if meta.Cardinality != CardinalityExclusive {
		t.Errorf("policy should be exclusive, got %s", meta.Cardinality)
	}
}

// ── PolicyDefs ───────────────────────────────────────────

func TestCoveragePolicyDefs_Count(t *testing.T) {
	if len(PolicyDefs) != 15 {
		t.Errorf("expected 15 policy defs, got %d", len(PolicyDefs))
	}
}

func TestCoveragePolicyDefs_AllHaveLabel(t *testing.T) {
	for key, def := range PolicyDefs {
		if def.Label == "" {
			t.Errorf("policy %s has empty label", key)
		}
		if def.Key != key {
			t.Errorf("policy %s key mismatch: def.Key=%s", key, def.Key)
		}
	}
}

// ── policyKeyOrder ───────────────────────────────────────

func TestCoveragePolicyKeyOrder_AllValid(t *testing.T) {
	for _, key := range policyKeyOrder {
		if !IsValidPolicyKey(key) {
			t.Errorf("policyKeyOrder contains invalid key: %s", key)
		}
	}
}

// ── Source types ─────────────────────────────────────────

func TestCoverageSourceTypes(t *testing.T) {
	sources := []SourceType{SourceLocal, SourceInherited, SourceAllUsers, SourceSiteDefault, SourceGlobal, SourceUnset}
	for _, s := range sources {
		if s == "" {
			t.Error("source type should not be empty")
		}
	}
}

// ── ConfigCategoryResult struct ──────────────────────────

func TestCoverageConfigCategoryResult_Struct(t *testing.T) {
	result := ConfigCategoryResult{
		Key:         CategoryKeyModel,
		Label:       "模型",
		Description: "用户能使用哪些模型",
		Icon:        "Brain",
		Entries: []ConfigEntry{
			{ID: "1", Label: "GPT-4", Source: Source{Type: SourceLocal, GroupID: 1}},
		},
	}
	if result.Key != CategoryKeyModel {
		t.Error("unexpected key")
	}
	if len(result.Entries) != 1 {
		t.Error("expected 1 entry")
	}
	if result.Entries[0].Source.GroupID != 1 {
		t.Error("unexpected source group_id")
	}
}
