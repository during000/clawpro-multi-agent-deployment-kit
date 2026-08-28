package skillhubclient

import (
	"encoding/json"
	"strings"
	"testing"
)

// ── adapter.go 测试 ──

func TestConvertSkillHubListToHatchery_Success(t *testing.T) {
	resp := &SkillListResponse{
		Total: 2,
		Items: []SkillItem{
			{
				ID:          1,
				DisplayName: "My Skill",
				Slug:        "my-skill",
				Summary:     "A test skill",
				Version:     "1.2.3",
				CreatedAt:   "2026-07-15T10:00:00Z",
				UpdatedAt:   "2026-07-15T12:00:00Z",
			},
			{
				ID:          2,
				DisplayName: "Another Skill",
				Slug:        "another-skill",
				Summary:     "Second skill",
				Version:     "2.0.0",
				CreatedAt:   "2026-07-14T08:00:00Z",
				UpdatedAt:   "",
			},
		},
	}

	result := ConvertSkillHubListToHatchery(resp)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}

	s1 := result[0]
	if s1.ID != 1 {
		t.Errorf("s1.ID = %d", s1.ID)
	}
	if s1.Name != "My Skill" {
		t.Errorf("s1.Name = %q", s1.Name)
	}
	if s1.Slug != "my-skill" {
		t.Errorf("s1.Slug = %q", s1.Slug)
	}
	if s1.Description != "A test skill" {
		t.Errorf("s1.Description = %q", s1.Description)
	}
	if s1.Version != "1.2.3" {
		t.Errorf("s1.Version = %q", s1.Version)
	}
	if s1.VisibilityType != "all" {
		t.Errorf("s1.VisibilityType = %q", s1.VisibilityType)
	}
	if s1.CreatedAt.IsZero() {
		t.Error("s1.CreatedAt is zero")
	}
	if s1.UpdatedAt.IsZero() {
		t.Error("s1.UpdatedAt is zero")
	}
	if s1.Categories == nil || len(s1.Categories) != 0 {
		t.Errorf("s1.Categories = %v, want empty slice", s1.Categories)
	}
	if s1.VisibilityGroups == nil || len(s1.VisibilityGroups) != 0 {
		t.Errorf("s1.VisibilityGroups = %v, want empty slice", s1.VisibilityGroups)
	}
	if s1.LastTask != nil {
		t.Error("s1.LastTask should be nil")
	}
	if s1.SecurityScan != nil {
		t.Error("s1.SecurityScan should be nil")
	}
}

func TestConvertSkillHubListToHatchery_EmptyUpdatedAt(t *testing.T) {
	resp := &SkillListResponse{
		Total: 1,
		Items: []SkillItem{
			{
				ID:        1,
				Slug:      "skill",
				CreatedAt: "2026-07-15T10:00:00Z",
				UpdatedAt: "",
			},
		},
	}

	result := ConvertSkillHubListToHatchery(resp)
	if !result[0].UpdatedAt.Equal(result[0].CreatedAt) {
		t.Errorf("UpdatedAt should equal CreatedAt when UpdatedAt is empty")
	}
}

func TestConvertSkillHubListToHatchery_InvalidTimeFormat(t *testing.T) {
	resp := &SkillListResponse{
		Total: 1,
		Items: []SkillItem{
			{ID: 1, Slug: "skill", CreatedAt: "invalid", UpdatedAt: "also-invalid"},
		},
	}

	result := ConvertSkillHubListToHatchery(resp)
	if result[0].CreatedAt.IsZero() {
		t.Error("CreatedAt should fall back to now(), got zero")
	}
	if result[0].UpdatedAt.IsZero() {
		t.Error("UpdatedAt should fall back to CreatedAt, got zero")
	}
}

func TestConvertSkillHubListToHatchery_EmptyItems(t *testing.T) {
	result := ConvertSkillHubListToHatchery(&SkillListResponse{Total: 0, Items: []SkillItem{}})
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestConvertSkillHubListToHatchery_NilResponse(t *testing.T) {
	result := ConvertSkillHubListToHatchery(nil)
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestConvertSkillHubListToHatchery_NilItems(t *testing.T) {
	result := ConvertSkillHubListToHatchery(&SkillListResponse{Items: nil})
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestHatcherySkill_JSONKeys(t *testing.T) {
	s := HatcherySkill{
		ID:             1,
		Slug:           "test",
		Name:           "Test",
		VisibilityType: "all",
		Categories:     []map[string]interface{}{},
		VisibilityGroups: []map[string]interface{}{},
	}

	data, _ := json.Marshal(s)
	jsonStr := string(data)

	requiredKeys := []string{
		`"id"`, `"slug"`, `"name"`, `"description"`,
		`"version"`, `"visibility_type"`, `"categories"`,
		`"visibility_groups"`, `"last_task"`, `"security_scan"`,
	}
	for _, key := range requiredKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("JSON missing key %q: %s", key, jsonStr)
		}
	}
}
