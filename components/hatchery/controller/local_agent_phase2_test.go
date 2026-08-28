package controller

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupPhase2TestDB 初始化包含 rule 表的测试 DB。
func setupPhase2TestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if sqlDB, e := db.DB(); e == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := db.AutoMigrate(
		&model.Instance{},
		&model.User{},
		&model.Skill{},
		&model.SkillDistributionRecord{},
		&model.SkillDistributionTask{},
		&model.SiteConfig{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.LocalInstanceInfo{},
		&model.LocalInstanceSkill{},
		&model.LocalInstanceRule{},
		&model.EnterpriseRule{},
		&model.RuleDistributionRecord{},
		&model.RuleDistributionTask{},
		&model.FeatureAllowlist{},
		&model.SkillVisibilityGroup{},
		&model.RuleVisibilityGroup{},
		// local_agent_scope_bindings：ensureUserLevelGroup / report 路径会写入与查询
		&model.LocalAgentScopeBinding{},
		// catalog 查询依赖：group_closure + group_config_bindings
		&model.GroupClosure{},
		&model.GroupConfigBinding{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	t.Cleanup(useDBForTestWithSafeRestore(db))
	model.DBGlobal(context.Background()).Model(&model.SiteConfig{}).
		Where("1 = 1").Update("local_agent_enabled", true)
	return db
}

// ---- resolveUserPrimaryGroup 测试 ----------------------------------------------------

func TestResolveUserPrimaryGroup_OneIDMain(t *testing.T) {
	db := setupPhase2TestDB(t)
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Email: "u1@t.com"}
	db.Create(user)

	g1 := &model.UserGroup{Name: "G1", Source: model.GroupSourceOneIDDept}
	db.Create(g1)
	g2 := &model.UserGroup{Name: "G2", Source: model.GroupSourceOneIDDept}
	db.Create(g2)

	// g1 是主分组
	db.Create(&model.UserGroupMember{UserGroupID: g1.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept, IsMain: true})
	db.Create(&model.UserGroupMember{UserGroupID: g2.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept, IsMain: false})

	got := resolveUserPrimaryGroup(ctx, db, user.ID)
	if got != g1.ID {
		t.Errorf("resolveUserPrimaryGroup = %d, want %d (OneID main)", got, g1.ID)
	}
}

func TestResolveUserPrimaryGroup_SingleOneIDNoMain(t *testing.T) {
	db := setupPhase2TestDB(t)
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Email: "u1@t.com"}
	db.Create(user)

	g1 := &model.UserGroup{Name: "G1", Source: model.GroupSourceOneIDDept}
	db.Create(g1)
	db.Create(&model.UserGroupMember{UserGroupID: g1.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept, IsMain: false})

	got := resolveUserPrimaryGroup(ctx, db, user.ID)
	if got != g1.ID {
		t.Errorf("resolveUserPrimaryGroup = %d, want %d (single OneID no main)", got, g1.ID)
	}
}

func TestResolveUserPrimaryGroup_MultipleOneIDNoMain(t *testing.T) {
	db := setupPhase2TestDB(t)
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Email: "u1@t.com"}
	db.Create(user)

	g1 := &model.UserGroup{Name: "G1", Source: model.GroupSourceOneIDDept}
	db.Create(g1)
	g2 := &model.UserGroup{Name: "G2", Source: model.GroupSourceOneIDDept}
	db.Create(g2)
	db.Create(&model.UserGroupMember{UserGroupID: g1.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept, IsMain: false})
	db.Create(&model.UserGroupMember{UserGroupID: g2.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept, IsMain: false})

	got := resolveUserPrimaryGroup(ctx, db, user.ID)
	if got != 0 {
		t.Errorf("resolveUserPrimaryGroup = %d, want 0 (multiple OneID no main)", got)
	}
}

func TestResolveUserPrimaryGroup_ManualOnly(t *testing.T) {
	db := setupPhase2TestDB(t)
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Email: "u1@t.com"}
	db.Create(user)

	g1 := &model.UserGroup{Name: "G1", Source: "manual"}
	db.Create(g1)
	db.Create(&model.UserGroupMember{UserGroupID: g1.ID, UserID: user.ID, Source: model.MemberSourceManual, IsMain: false})

	got := resolveUserPrimaryGroup(ctx, db, user.ID)
	if got != 0 {
		t.Errorf("resolveUserPrimaryGroup = %d, want 0 (manual only)", got)
	}
}

func TestResolveUserPrimaryGroup_NoGroups(t *testing.T) {
	db := setupPhase2TestDB(t)
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Email: "u1@t.com"}
	db.Create(user)

	got := resolveUserPrimaryGroup(ctx, db, user.ID)
	if got != 0 {
		t.Errorf("resolveUserPrimaryGroup = %d, want 0 (no groups)", got)
	}
}

// ---- ensureUserLevelGroup 测试 -------------------------------------------------------

func TestEnsureUserLevelGroup_NewAgentAutoAssign(t *testing.T) {
	db := setupPhase2TestDB(t)
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Email: "u1@t.com"}
	db.Create(user)

	// OneID 主分组
	group := &model.UserGroup{Name: "DevTeam", Source: model.GroupSourceOneIDDept}
	db.Create(group)
	db.Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept, IsMain: true})

	inst := &model.Instance{UserID: user.ID, InstanceId: "local-workbuddy-001", Source: model.InstanceSourceLocal}
	db.Create(inst)

	resources := &model.LocalAgentResources{UserLevel: model.UserLevelResources{}, Workspaces: []model.WorkspaceResource{}}

	changed, err := ensureUserLevelGroup(ctx, db, inst, user, resources)
	if err != nil {
		t.Fatalf("ensureUserLevelGroup error: %v", err)
	}
	if !changed {
		t.Error("expected groupChanged=true for new agent")
	}
	if resources.UserLevel.GroupID != group.ID {
		t.Errorf("GroupID = %d, want %d", resources.UserLevel.GroupID, group.ID)
	}
	if resources.UserLevel.GroupName != "DevTeam" {
		t.Errorf("GroupName = %q, want %q", resources.UserLevel.GroupName, "DevTeam")
	}
}

func TestEnsureUserLevelGroup_ValidGroupKept(t *testing.T) {
	db := setupPhase2TestDB(t)
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Email: "u1@t.com"}
	db.Create(user)

	group := &model.UserGroup{Name: "G1", Source: "manual"}
	db.Create(group)
	db.Create(&model.UserGroupMember{UserGroupID: group.ID, UserID: user.ID, Source: model.MemberSourceManual})

	inst := &model.Instance{UserID: user.ID, InstanceId: "local-workbuddy-001", Source: model.InstanceSourceLocal}
	db.Create(inst)

	// 已有有效分组
	resources := &model.LocalAgentResources{
		UserLevel:  model.UserLevelResources{GroupID: group.ID, GroupName: "G1"},
		Workspaces: []model.WorkspaceResource{},
	}

	changed, err := ensureUserLevelGroup(ctx, db, inst, user, resources)
	if err != nil {
		t.Fatalf("ensureUserLevelGroup error: %v", err)
	}
	if changed {
		t.Error("expected groupChanged=false for valid group")
	}
	if resources.UserLevel.GroupID != group.ID {
		t.Errorf("GroupID = %d, want %d (should not change)", resources.UserLevel.GroupID, group.ID)
	}
	var reloaded model.Instance
	if err := db.First(&reloaded, inst.ID).Error; err != nil {
		t.Fatalf("重新查询本地 Agent 失败: %v", err)
	}
	if reloaded.GroupID != group.ID {
		t.Errorf("存量本地 Agent 的 instances.group_id=%d，应修复为用户级组织=%d", reloaded.GroupID, group.ID)
	}
}

func TestEnsureUserLevelGroup_InvalidGroupAutoSwitch(t *testing.T) {
	db := setupPhase2TestDB(t)
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Email: "u1@t.com"}
	db.Create(user)

	// 新主分组
	newGroup := &model.UserGroup{Name: "NewTeam", Source: model.GroupSourceOneIDDept}
	db.Create(newGroup)
	db.Create(&model.UserGroupMember{UserGroupID: newGroup.ID, UserID: user.ID, Source: model.MemberSourceOneIDDept, IsMain: true})

	inst := &model.Instance{UserID: user.ID, InstanceId: "local-workbuddy-001", Source: model.InstanceSourceLocal}
	db.Create(inst)

	// 旧分组已失效（不在用户分组列表里）
	oldGroupID := uint(99999)
	resources := &model.LocalAgentResources{
		UserLevel:  model.UserLevelResources{GroupID: oldGroupID, GroupName: "OldTeam"},
		Workspaces: []model.WorkspaceResource{},
	}

	changed, err := ensureUserLevelGroup(ctx, db, inst, user, resources)
	if err != nil {
		t.Fatalf("ensureUserLevelGroup error: %v", err)
	}
	if !changed {
		t.Error("expected groupChanged=true for invalid group")
	}
	if resources.UserLevel.GroupID != newGroup.ID {
		t.Errorf("GroupID = %d, want %d (switched to new primary)", resources.UserLevel.GroupID, newGroup.ID)
	}
}

func TestEnsureUserLevelGroup_InvalidGroupNoPrimary(t *testing.T) {
	db := setupPhase2TestDB(t)
	ctx := context.Background()

	user := &model.User{Username: "u1", Password: "x", Email: "u1@t.com"}
	db.Create(user)

	// 用户无任何分组
	inst := &model.Instance{UserID: user.ID, InstanceId: "local-workbuddy-001", Source: model.InstanceSourceLocal}
	db.Create(inst)

	// 旧分组已失效
	oldGroupID := uint(99999)
	resources := &model.LocalAgentResources{
		UserLevel:  model.UserLevelResources{GroupID: oldGroupID, GroupName: "OldTeam"},
		Workspaces: []model.WorkspaceResource{},
	}

	changed, err := ensureUserLevelGroup(ctx, db, inst, user, resources)
	if err != nil {
		t.Fatalf("ensureUserLevelGroup error: %v", err)
	}
	if !changed {
		t.Error("expected groupChanged=true")
	}
	if resources.UserLevel.GroupID != 0 {
		t.Errorf("GroupID = %d, want 0 (no primary group)", resources.UserLevel.GroupID)
	}
}

// ---- alignLocalRules 测试 -----------------------------------------------------------

func TestAlignLocalRules_InsertNew(t *testing.T) {
	db := setupPhase2TestDB(t)
	now := time.Now()

	reported := []reportRuleEntry{
		{Slug: "rule-1", Version: "1.0.0", DisplayName: "Rule 1", RuleType: "prompt", Source: "enterprise"},
	}

	synced, err := alignLocalRules(db, 1, model.LocalSkillScopeUser, "", reported, now)
	if err != nil {
		t.Fatalf("alignLocalRules error: %v", err)
	}
	if synced != 1 {
		t.Errorf("synced = %d, want 1", synced)
	}

	var row model.LocalInstanceRule
	if err := db.Where("instance_id = ? AND slug = ?", 1, "rule-1").First(&row).Error; err != nil {
		t.Fatalf("查询插入的 rule 失败: %v", err)
	}
	if row.InstallStatus != model.LocalSkillInstallStatusDistributed {
		t.Errorf("InstallStatus = %q, want %q", row.InstallStatus, model.LocalSkillInstallStatusDistributed)
	}
	if row.Scope != model.LocalSkillScopeUser {
		t.Errorf("Scope = %q, want %q", row.Scope, model.LocalSkillScopeUser)
	}
}

func TestAlignLocalRules_DefaultSourceIsLocal(t *testing.T) {
	db := setupPhase2TestDB(t)
	platformSkill := model.LocalInstanceSkill{InstanceID: 1, Slug: "platform-skill", Scope: model.LocalSkillScopeWorkspace, WorkspacePath: "/platform"}
	if err := db.Create(&platformSkill).Error; err != nil {
		t.Fatalf("创建平台 skill 快照失败: %v", err)
	}
	if platformSkill.Source != model.LocalSkillSourceLocal {
		t.Fatalf("未显式来源的本地资源默认应为 local，实际=%q", platformSkill.Source)
	}
	if _, err := alignLocalRules(db, 1, model.LocalSkillScopeUser, "", []reportRuleEntry{{Slug: "rule-local"}}, time.Now()); err != nil {
		t.Fatalf("alignLocalRules error: %v", err)
	}
	var row model.LocalInstanceRule
	if err := db.Where("instance_id = ? AND slug = ?", 1, "rule-local").First(&row).Error; err != nil {
		t.Fatalf("查询 rule 失败: %v", err)
	}
	if row.Source != model.LocalRuleSourceLocal {
		t.Errorf("未传 source 时应按本地事实上报，实际=%q", row.Source)
	}
	if _, err := alignLocalSkills(db, 1, model.LocalSkillScopeUser, "", []reportSkillEntry{{Slug: "skill-local"}}, time.Now()); err != nil {
		t.Fatalf("alignLocalSkills error: %v", err)
	}
	var localSkill model.LocalInstanceSkill
	if err := db.Where("instance_id = ? AND slug = ?", 1, "skill-local").First(&localSkill).Error; err != nil {
		t.Fatalf("查询 skill 失败: %v", err)
	}
	if localSkill.Source != model.LocalSkillSourceLocal {
		t.Errorf("未传 source 的 report skill 应为 local，实际=%q", localSkill.Source)
	}
}

func TestAlignLocalRules_DisappearDelete(t *testing.T) {
	db := setupPhase2TestDB(t)
	now := time.Now()

	// 预置：1 个 distributed + 1 个 distributing
	db.Create(&model.LocalInstanceRule{InstanceID: 1, Slug: "r-distributed", Version: "1.0", InstallStatus: model.LocalSkillInstallStatusDistributed, Scope: model.LocalSkillScopeUser, InstalledAt: &now, LastSeenAt: &now})
	db.Create(&model.LocalInstanceRule{InstanceID: 1, Slug: "r-distributing", Version: "1.0", InstallStatus: model.LocalSkillInstallStatusDistributing, Scope: model.LocalSkillScopeUser, InstalledAt: &now, LastSeenAt: &now})

	// report 只包含 r-distributed（r-distributing 不在 report 里）
	reported := []reportRuleEntry{
		{Slug: "r-distributed", Version: "1.0", DisplayName: "R Dist"},
	}

	_, err := alignLocalRules(db, 1, model.LocalSkillScopeUser, "", reported, now)
	if err != nil {
		t.Fatalf("alignLocalRules error: %v", err)
	}

	// r-distributed 应保留
	var count int64
	db.Model(&model.LocalInstanceRule{}).Where("instance_id = ? AND slug = ?", 1, "r-distributed").Count(&count)
	if count != 1 {
		t.Errorf("r-distributed count = %d, want 1 (should keep)", count)
	}

	// r-distributing 应保留（不删 distributing）
	db.Model(&model.LocalInstanceRule{}).Where("instance_id = ? AND slug = ?", 1, "r-distributing").Count(&count)
	if count != 1 {
		t.Errorf("r-distributing count = %d, want 1 (should NOT delete distributing)", count)
	}
}

func TestAlignLocalRules_UpdateChanged(t *testing.T) {
	db := setupPhase2TestDB(t)
	now := time.Now()

	// 预置
	db.Create(&model.LocalInstanceRule{InstanceID: 1, Slug: "r1", Version: "1.0", DisplayName: "Old", InstallStatus: model.LocalSkillInstallStatusDistributed, Scope: model.LocalSkillScopeUser, InstalledAt: &now, LastSeenAt: &now})

	// report 带新版本
	reported := []reportRuleEntry{
		{Slug: "r1", Version: "2.0", DisplayName: "New", RuleType: "rule"},
	}

	_, err := alignLocalRules(db, 1, model.LocalSkillScopeUser, "", reported, now)
	if err != nil {
		t.Fatalf("alignLocalRules error: %v", err)
	}

	var row model.LocalInstanceRule
	db.Where("instance_id = ? AND slug = ?", 1, "r1").First(&row)
	if row.Version != "2.0" {
		t.Errorf("Version = %q, want 2.0", row.Version)
	}
	if row.DisplayName != "New" {
		t.Errorf("DisplayName = %q, want New", row.DisplayName)
	}
	// install_status 不应变（report 不改 install_status）
	if row.InstallStatus != model.LocalSkillInstallStatusDistributed {
		t.Errorf("InstallStatus = %q, want %q", row.InstallStatus, model.LocalSkillInstallStatusDistributed)
	}
}

func TestAlignLocalRules_ScopeIsolated(t *testing.T) {
	db := setupPhase2TestDB(t)
	now := time.Now()

	// user scope 有 r1
	db.Create(&model.LocalInstanceRule{InstanceID: 1, Slug: "r1", Version: "1.0", InstallStatus: model.LocalSkillInstallStatusDistributed, Scope: model.LocalSkillScopeUser, InstalledAt: &now, LastSeenAt: &now})
	// project scope 也有 r1
	db.Create(&model.LocalInstanceRule{InstanceID: 1, Slug: "r1", Version: "1.0", InstallStatus: model.LocalSkillInstallStatusDistributed, Scope: model.LocalSkillScopeWorkspace, WorkspacePath: "/ws", InstalledAt: &now, LastSeenAt: &now})

	// report user scope 为空 → user scope 的 r1 应被删
	reported := []reportRuleEntry{}

	_, err := alignLocalRules(db, 1, model.LocalSkillScopeUser, "", reported, now)
	if err != nil {
		t.Fatalf("alignLocalRules error: %v", err)
	}

	// user scope 的 r1 应被删
	var count int64
	db.Model(&model.LocalInstanceRule{}).Where("instance_id = ? AND slug = ? AND scope = ?", 1, "r1", model.LocalSkillScopeUser).Count(&count)
	if count != 0 {
		t.Errorf("user scope r1 count = %d, want 0 (should be deleted)", count)
	}

	// project scope 的 r1 应保留
	db.Model(&model.LocalInstanceRule{}).Where("instance_id = ? AND slug = ? AND scope = ?", 1, "r1", model.LocalSkillScopeWorkspace).Count(&count)
	if count != 1 {
		t.Errorf("project scope r1 count = %d, want 1 (scope isolated)", count)
	}
}
