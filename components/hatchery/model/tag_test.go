package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTagTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&SiteConfig{}, &Tag{}, &TagVisibilityGroup{}, &UserGroup{}, &GroupClosure{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	oldDB := gdb
	gdb = db
	t.Cleanup(func() { gdb = oldDB })
	return db
}

func TestReplaceGlobalTags_MigratesLegacyAndClearsOldField(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	db.Create(&SiteConfig{DefaultTags: `[{"Key":"legacy","Value":"yes"}]`})

	if err := ReplaceGlobalTags(ctx, []TagItem{{Key: "env", Value: "prod"}}); err != nil {
		t.Fatalf("ReplaceGlobalTags failed: %v", err)
	}

	var cfg SiteConfig
	db.First(&cfg)
	if cfg.DefaultTags != "[]" {
		t.Fatalf("DefaultTags should be cleared, got %q", cfg.DefaultTags)
	}

	items, err := GetGlobalTagItemsForConfig(ctx, cfg.DefaultTags)
	if err != nil {
		t.Fatalf("GetGlobalTagItemsForConfig failed: %v", err)
	}
	if len(items) != 1 || items[0].Key != "env" || items[0].Value != "prod" {
		t.Fatalf("unexpected global tags: %+v", items)
	}
}

func TestResolveTagsForGroup_UsesAncestorBindings(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	db.Create(&SiteConfig{DefaultTags: `[{"Key":"legacy","Value":"yes"}]`})
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 1, Depth: 0})
	db.Create(&GroupClosure{AncestorID: 1, DescendantID: 2, Depth: 1})
	db.Create(&GroupClosure{AncestorID: 2, DescendantID: 2, Depth: 0})
	db.Create(&Tag{TagKey: "global", TagValue: "on", VisibilityType: VisibilityAll})
	scoped := Tag{TagKey: "team", TagValue: "rd", VisibilityType: VisibilityGroup}
	db.Create(&scoped)
	db.Create(&TagVisibilityGroup{TagID: scoped.ID, GroupID: 1})

	items, err := ResolveTagsForGroup(ctx, 2, `[{"Key":"legacy","Value":"yes"}]`)
	if err != nil {
		t.Fatalf("ResolveTagsForGroup failed: %v", err)
	}
	got := map[string]string{}
	for _, item := range items {
		got[item.Key] = item.Value
	}
	if got["legacy"] != "" {
		t.Fatalf("legacy tags should not be used once new rows exist: %+v", items)
	}
	if got["global"] != "on" || got["team"] != "rd" {
		t.Fatalf("unexpected resolved tags: %+v", items)
	}
}

func TestResolveTagsForGroup_FallbackLegacyWhenNewTableEmpty(t *testing.T) {
	setupTagTestDB(t)
	ctx := context.Background()

	items, err := ResolveTagsForGroup(ctx, 10, `[{"Key":"legacy","Value":"yes"}]`)
	if err != nil {
		t.Fatalf("ResolveTagsForGroup failed: %v", err)
	}
	if len(items) != 1 || items[0].Key != "legacy" || items[0].Value != "yes" {
		t.Fatalf("unexpected fallback tags: %+v", items)
	}
}

func TestGetGlobalTagItemsForConfig_FallbackLegacyWhenNewTableEmpty(t *testing.T) {
	setupTagTestDB(t)
	items, err := GetGlobalTagItemsForConfig(context.Background(), `[{"Key":"legacy","Value":"yes"}]`)
	if err != nil {
		t.Fatalf("GetGlobalTagItemsForConfig failed: %v", err)
	}
	if len(items) != 1 || items[0].Key != "legacy" || items[0].Value != "yes" {
		t.Fatalf("unexpected fallback tags: %+v", items)
	}
}

func TestParseAndMarshalTagItems_NormalizesInput(t *testing.T) {
	items := ParseTagItems(`[
		{"Key":" env ","Value":"prod"},
		{"Key":"env","Value":"prod"},
		{"Key":"","Value":"ignored"},
		{"Key":"team","Value":"rd"}
	]`)
	if len(items) != 2 {
		t.Fatalf("expected normalized two tags, got %+v", items)
	}
	if items[0].Key != "env" || items[0].Value != "prod" || items[1].Key != "team" {
		t.Fatalf("unexpected normalized tags: %+v", items)
	}
	if got := MarshalTagItems(nil); got != "[]" {
		t.Fatalf("empty tags should marshal to [], got %q", got)
	}
	if got := MarshalTagItems(items); got != `[{"Key":"env","Value":"prod"},{"Key":"team","Value":"rd"}]` {
		t.Fatalf("unexpected marshaled tags: %s", got)
	}
	if got := ParseTagItems("{bad json"); len(got) != 0 {
		t.Fatalf("invalid json should parse as empty, got %+v", got)
	}
}

func TestTagCRUDReplaceListDeleteAndGroupUsage(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	db.Create(&SiteConfig{DefaultTags: `[{"Key":"legacy","Value":"yes"}]`})

	if _, err := CreateTag(ctx, "", "bad", VisibilityAll, nil); err == nil {
		t.Fatal("expected empty key validation error")
	}
	if _, err := CreateTag(ctx, "bad", "value", "team", nil); err == nil {
		t.Fatal("expected invalid visibility validation error")
	}
	if _, err := CreateTag(ctx, "team", "rd", VisibilityGroup, nil); err == nil {
		t.Fatal("expected group visibility without groups validation error")
	}

	row, err := CreateTag(ctx, " team ", "rd", VisibilityGroup, []uint{0, 2, 2, 3})
	if err != nil {
		t.Fatalf("CreateTag failed: %v", err)
	}
	if row.TagKey != "team" {
		t.Fatalf("expected trimmed key, got %q", row.TagKey)
	}

	rows, groupMap, err := ListTags(ctx)
	if err != nil {
		t.Fatalf("ListTags failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected migrated legacy + created tag, got %+v", rows)
	}
	if got := groupMap[row.ID]; len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Fatalf("expected deduped groups [2 3], got %+v", got)
	}
	used, err := IsGroupUsedByTagVisibility(ctx, 2)
	if err != nil || !used {
		t.Fatalf("expected group 2 to be used, used=%v err=%v", used, err)
	}

	updated, err := UpdateTag(ctx, row.ID, "env", "prod", VisibilityAll, []uint{2})
	if err != nil {
		t.Fatalf("UpdateTag failed: %v", err)
	}
	if updated.VisibilityType != VisibilityAll {
		t.Fatalf("expected global tag after update, got %+v", updated)
	}
	used, err = IsGroupUsedByTagVisibility(ctx, 2)
	if err != nil || used {
		t.Fatalf("global update should remove group binding, used=%v err=%v", used, err)
	}
	if _, err := UpdateTag(ctx, 999, "missing", "x", VisibilityAll, nil); err == nil {
		t.Fatal("expected missing tag update error")
	}

	if err := ReplaceTags(ctx, []TagWithScope{
		{Key: " global ", Value: "on"},
		{Key: "team", Value: "rd", VisibilityType: VisibilityGroup, GroupIDs: []uint{5, 5}},
	}); err != nil {
		t.Fatalf("ReplaceTags failed: %v", err)
	}
	rows, groupMap, err = ListTags(ctx)
	if err != nil {
		t.Fatalf("ListTags after replace failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two replaced tags, got %+v", rows)
	}
	var scopedID uint
	for _, r := range rows {
		if r.TagKey == "team" {
			scopedID = r.ID
		}
	}
	if got := groupMap[scopedID]; len(got) != 1 || got[0] != 5 {
		t.Fatalf("expected replaced scoped group [5], got %+v", got)
	}

	if err := DeleteTag(ctx, scopedID); err != nil {
		t.Fatalf("DeleteTag failed: %v", err)
	}
	used, err = IsGroupUsedByTagVisibility(ctx, 5)
	if err != nil || used {
		t.Fatalf("delete should remove group binding, used=%v err=%v", used, err)
	}
}

func TestResolveTagsForGroup_KeyOverrideAndNoGroup(t *testing.T) {
	db := setupTagTestDB(t)
	ctx := context.Background()
	db.Create(&Tag{TagKey: "env", TagValue: "global", VisibilityType: VisibilityAll})
	scoped := Tag{TagKey: "env", TagValue: "group", VisibilityType: VisibilityGroup}
	db.Create(&scoped)
	db.Create(&TagVisibilityGroup{TagID: scoped.ID, GroupID: 7})

	items, err := ResolveTagsForGroup(ctx, 0, `[{"Key":"legacy","Value":"yes"}]`)
	if err != nil {
		t.Fatalf("ResolveTagsForGroup without group failed: %v", err)
	}
	if len(items) != 1 || items[0].Value != "global" {
		t.Fatalf("no group should only return global tag, got %+v", items)
	}

	items, err = ResolveTagsForGroup(ctx, 7, `[{"Key":"legacy","Value":"yes"}]`)
	if err != nil {
		t.Fatalf("ResolveTagsForGroup with group failed: %v", err)
	}
	if len(items) != 1 || items[0].Key != "env" || items[0].Value != "group" {
		t.Fatalf("group tag should override same-key global tag, got %+v", items)
	}
}
