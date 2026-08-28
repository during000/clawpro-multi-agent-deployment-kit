package model

import (
	"context"
	"errors"
	"strings"
	"testing"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupProjectModelTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open project test DB: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	if err := db.AutoMigrate(
		&Project{},
		&ProjectMember{},
		&ProjectConfigBinding{},
		&Instance{},
		&LocalAgentScopeBinding{},
	); err != nil {
		t.Fatalf("migrate project test DB: %v", err)
	}
	return db.WithContext(common.WithSkipIdentifier(context.Background()))
}

func TestAssetBindingConfigType_AllTypes(t *testing.T) {
	tests := []struct {
		assetType string
		wantType  string
		wantOK    bool
	}{
		{AssetTypeSkill, AssetBindingTypeSkill, true},
		{AssetTypeRule, AssetBindingTypeRule, true},
		{"unknown", "", false},
	}
	for _, tt := range tests {
		gotType, gotOK := AssetBindingConfigType(tt.assetType)
		if gotType != tt.wantType || gotOK != tt.wantOK {
			t.Fatalf("AssetBindingConfigType(%q) = (%q, %v), want (%q, %v)",
				tt.assetType, gotType, gotOK, tt.wantType, tt.wantOK)
		}
	}
}

func TestNormalizeProjectName_Validation(t *testing.T) {
	got, err := NormalizeProjectName("  Project A  ")
	if err != nil || got != "Project A" {
		t.Fatalf("valid name = %q, %v; want Project A, nil", got, err)
	}
	for _, name := range []string{"   ", strings.Repeat("项", 192)} {
		if _, err := NormalizeProjectName(name); !errors.Is(err, ErrInvalidProjectName) {
			t.Fatalf("NormalizeProjectName(%q) error = %v, want ErrInvalidProjectName", name, err)
		}
	}
	if got, err := NormalizeProjectName(strings.Repeat("项", 191)); err != nil || len([]rune(got)) != 191 {
		t.Fatalf("191-rune name rejected: len=%d err=%v", len([]rune(got)), err)
	}
}

func TestProjectMemberAndBindingReplacement_DeduplicatesAndClears(t *testing.T) {
	db := setupProjectModelTestDB(t)

	if err := db.Create(&ProjectMember{ProjectID: 1, UserID: 99, CreatedBy: 7}).Error; err != nil {
		t.Fatalf("seed member: %v", err)
	}
	if err := ReplaceProjectMembers(db, 1, []uint{0, 2, 2, 3}, 8); err != nil {
		t.Fatalf("ReplaceProjectMembers: %v", err)
	}
	var members []ProjectMember
	if err := db.Where("project_id = ?", 1).Order("user_id").Find(&members).Error; err != nil {
		t.Fatalf("query members: %v", err)
	}
	if len(members) != 2 || members[0].UserID != 2 || members[1].UserID != 3 || members[0].CreatedBy != 8 {
		t.Fatalf("unexpected members: %#v", members)
	}
	if err := ReplaceProjectMembers(db, 1, nil, 8); err != nil {
		t.Fatalf("clear members: %v", err)
	}
	if err := db.Where("project_id = ?", 1).Find(&members).Error; err != nil || len(members) != 0 {
		t.Fatalf("members not cleared: len=%d err=%v", len(members), err)
	}

	if err := ReplaceProjectConfigBindings(db, 1, AssetBindingTypeSkill,
		[]string{" alpha ", "", "alpha", "beta"}); err != nil {
		t.Fatalf("ReplaceProjectConfigBindings: %v", err)
	}
	var bindings []ProjectConfigBinding
	if err := db.Where("project_id = ? AND config_type = ?", 1, AssetBindingTypeSkill).
		Order("config_key").Find(&bindings).Error; err != nil {
		t.Fatalf("query bindings: %v", err)
	}
	if len(bindings) != 2 || bindings[0].ConfigKey != "alpha" || bindings[1].ConfigKey != "beta" {
		t.Fatalf("unexpected config bindings: %#v", bindings)
	}
	if err := ReplaceProjectConfigBindings(db, 1, AssetBindingTypeSkill, nil); err != nil {
		t.Fatalf("clear config bindings: %v", err)
	}
	if err := db.Where("project_id = ?", 1).Find(&bindings).Error; err != nil || len(bindings) != 0 {
		t.Fatalf("bindings not cleared: len=%d err=%v", len(bindings), err)
	}
}

func TestProjectResourceBindings_ReplaceCleanupAndValidate(t *testing.T) {
	db := setupProjectModelTestDB(t)
	projects := []Project{
		{Identifier: "p1", Name: "P1", Description: ""},
		{Identifier: "p2", Name: "P2", Description: ""},
	}
	if err := db.Create(&projects).Error; err != nil {
		t.Fatalf("create projects: %v", err)
	}

	if err := ReplaceResourceProjectBindings(db, ProjectConfigTypeSkill, "skill-a",
		[]uint{0, projects[0].ID, projects[0].ID, projects[1].ID}); err != nil {
		t.Fatalf("ReplaceResourceProjectBindings: %v", err)
	}
	var count int64
	db.Model(&ProjectConfigBinding{}).Where("config_type = ? AND config_key = ?", ProjectConfigTypeSkill, "skill-a").Count(&count)
	if count != 2 {
		t.Fatalf("resource binding count=%d, want 2", count)
	}
	if err := ReplaceResourceProjectBindings(db, ProjectConfigTypeSkill, "skill-a", nil); err != nil {
		t.Fatalf("clear resource bindings: %v", err)
	}

	seed := []ProjectConfigBinding{
		{ProjectID: projects[0].ID, ConfigType: ProjectConfigTypeSkill, ConfigKey: "skill-b"},
		{ProjectID: projects[0].ID, ConfigType: AssetBindingTypeSkill, ConfigKey: "skill-b"},
		{ProjectID: projects[0].ID, ConfigType: ProjectConfigTypeRule, ConfigKey: "rule-b"},
		{ProjectID: projects[0].ID, ConfigType: AssetBindingTypeRule, ConfigKey: "rule-b"},
		{ProjectID: projects[1].ID, ConfigType: "other", ConfigKey: "other-b"},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed bindings: %v", err)
	}
	if err := CleanupProjectBindings(db, ProjectConfigTypeSkill, "skill-b"); err != nil {
		t.Fatalf("cleanup skill bindings: %v", err)
	}
	if err := CleanupProjectBindings(db, ProjectConfigTypeRule, "rule-b"); err != nil {
		t.Fatalf("cleanup rule bindings: %v", err)
	}
	if err := CleanupProjectBindings(db, "other", "other-b"); err != nil {
		t.Fatalf("cleanup other binding: %v", err)
	}
	db.Model(&ProjectConfigBinding{}).Count(&count)
	if count != 0 {
		t.Fatalf("bindings remain after cleanup: %d", count)
	}

	if err := ValidateProjectIDs(db, nil); err != nil {
		t.Fatalf("empty IDs: %v", err)
	}
	if err := ValidateProjectIDs(db, []uint{0, projects[0].ID, projects[0].ID, projects[1].ID}); err != nil {
		t.Fatalf("valid IDs: %v", err)
	}
	if err := ValidateProjectIDs(db, []uint{projects[0].ID, 99999}); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("invalid IDs error=%v, want ErrRecordNotFound", err)
	}
	if got := uniqueProjectIDs([]uint{0, 3, 2, 3, 2, 1}); len(got) != 3 || got[0] != 3 || got[1] != 2 || got[2] != 1 {
		t.Fatalf("uniqueProjectIDs returned %v", got)
	}
}

func TestListLocalAgentInstancesByScope_GroupsBindingsAndFiltersInstances(t *testing.T) {
	db := setupProjectModelTestDB(t)
	local := Instance{InstanceId: "local-1", Name: "Local", Source: InstanceSourceLocal}
	remote := Instance{InstanceId: "remote-1", Name: "Remote", Source: "cvm"}
	if err := db.Create(&local).Error; err != nil {
		t.Fatalf("create local instance: %v", err)
	}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create remote instance: %v", err)
	}
	bindings := []LocalAgentScopeBinding{
		{InstanceID: local.ID, Scope: LocalAgentScopeUser, ScopeKey: "", GroupID: 10},
		{InstanceID: local.ID, Scope: LocalAgentScopeUser, ScopeKey: "extra", GroupID: 10},
		{InstanceID: local.ID, Scope: LocalAgentScopeUser, ScopeKey: "descendant", GroupID: 11},
		{InstanceID: remote.ID, Scope: LocalAgentScopeUser, ScopeKey: "", GroupID: 10},
		{InstanceID: local.ID, Scope: LocalAgentScopeWorkspace, ScopeKey: "/repo", ProjectID: 20},
	}
	if err := db.Create(&bindings).Error; err != nil {
		t.Fatalf("create scope bindings: %v", err)
	}

	items, err := ListLocalAgentInstancesByScopeWithDB(db, LocalAgentScopeUser, 10)
	if err != nil || len(items) != 1 || items[0].Instance.ID != local.ID || len(items[0].ScopeBindings) != 2 {
		t.Fatalf("user scope items=%#v err=%v", items, err)
	}
	items, err = ListLocalAgentInstancesByScopeTargetsWithDB(db, LocalAgentScopeUser, []uint{10, 11, 10, 0})
	if err != nil || len(items) != 1 || len(items[0].ScopeBindings) != 3 {
		t.Fatalf("batched user scope items=%#v err=%v", items, err)
	}
	items, err = ListLocalAgentInstancesByScopeWithDB(db, LocalAgentScopeWorkspace, 20)
	if err != nil || len(items) != 1 || len(items[0].ScopeBindings) != 1 {
		t.Fatalf("workspace scope items=%#v err=%v", items, err)
	}
	for _, tc := range []struct {
		scope  string
		target uint
	}{{LocalAgentScopeUser, 0}, {"invalid", 10}, {LocalAgentScopeUser, 999}} {
		items, err = ListLocalAgentInstancesByScopeWithDB(db, tc.scope, tc.target)
		if err != nil || len(items) != 0 {
			t.Fatalf("scope=%q target=%d items=%#v err=%v", tc.scope, tc.target, items, err)
		}
	}

	restore := UseDBForTest(db)
	items, err = ListLocalAgentInstancesByScope(context.Background(), LocalAgentScopeUser, 10)
	if err != nil || len(items) != 1 {
		t.Fatalf("ListLocalAgentInstancesByScope items=%#v err=%v", items, err)
	}
	items, err = ListLocalAgentInstancesByScopeTargets(context.Background(), LocalAgentScopeUser, []uint{10, 11})
	restore()
	if err != nil || len(items) != 1 || len(items[0].ScopeBindings) != 3 {
		t.Fatalf("ListLocalAgentInstancesByScopeTargets items=%#v err=%v", items, err)
	}
}

func TestListLocalAgentInstancesByScopeTargets_QueryError(t *testing.T) {
	db := setupProjectModelTestDB(t)
	if err := db.Migrator().DropTable(&LocalAgentScopeBinding{}); err != nil {
		t.Fatalf("drop bindings table: %v", err)
	}
	if _, err := ListLocalAgentInstancesByScopeTargetsWithDB(db, LocalAgentScopeUser, []uint{1}); err == nil {
		t.Fatal("expected query error")
	}
}

func TestListUserProjects_ReturnsMembershipOrder(t *testing.T) {
	db := setupProjectModelTestDB(t)
	projects := []Project{
		{Identifier: "user-p1", Name: "User P1", Description: ""},
		{Identifier: "user-p2", Name: "User P2", Description: ""},
		{Identifier: "other-p", Name: "Other P", Description: ""},
	}
	if err := db.Create(&projects).Error; err != nil {
		t.Fatalf("create projects: %v", err)
	}
	if err := db.Create(&[]ProjectMember{
		{ProjectID: projects[1].ID, UserID: 7},
		{ProjectID: projects[0].ID, UserID: 7},
		{ProjectID: projects[2].ID, UserID: 8},
	}).Error; err != nil {
		t.Fatalf("create members: %v", err)
	}
	restore := UseDBForTest(db)
	got, err := ListUserProjects(context.Background(), 7)
	restore()
	if err != nil || len(got) != 2 || got[0].ID != projects[1].ID || got[1].ID != projects[0].ID {
		t.Fatalf("ListUserProjects=%#v err=%v", got, err)
	}
}
