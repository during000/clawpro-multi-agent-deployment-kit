package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCanDeleteGroupTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&UserGroup{},
		&UserGroupMember{},
		&AIModel{},
		&ModelVisibilityGroup{},
		&Skill{},
		&SkillVisibilityGroup{},
		&SkillBundle{},
		&SkillBundleVisibilityGroup{},
		&OpenClawRole{},
		&RoleVisibilityGroup{},
		&Tag{},
		&TagVisibilityGroup{},
		&Instance{},
		&GroupConfigBinding{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	gdb = db
}

func TestCanDeleteUserGroup_NoReferences(t *testing.T) {
	setupCanDeleteGroupTestDB(t)

	group := UserGroup{Name: "空组"}
	gdb.Create(&group)

	ok, err := CanDeleteUserGroup(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("无关联的用户组应允许删除")
	}
}

func TestCanDeleteUserGroup_BlockedByModelVisibility(t *testing.T) {
	setupCanDeleteGroupTestDB(t)

	group := UserGroup{Name: "模型关联组"}
	gdb.Create(&group)
	model := AIModel{ModelName: "test-model", VisibilityType: "group"}
	gdb.Create(&model)
	gdb.Create(&ModelVisibilityGroup{AIModelID: model.ID, GroupID: group.ID})

	ok, err := CanDeleteUserGroup(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("有模型关联的用户组不应允许删除")
	}
}

func TestCanDeleteUserGroup_BlockedBySkillVisibility(t *testing.T) {
	setupCanDeleteGroupTestDB(t)

	group := UserGroup{Name: "技能关联组"}
	gdb.Create(&group)
	skill := Skill{Name: "test", Slug: "test", Version: "1.0.0", VisibilityType: "group"}
	skill.ParseVersion()
	gdb.Create(&skill)
	gdb.Create(&SkillVisibilityGroup{SkillID: skill.ID, GroupID: group.ID})

	ok, err := CanDeleteUserGroup(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("有技能关联的用户组不应允许删除")
	}
}

func TestCanDeleteUserGroup_BlockedByBundleVisibility(t *testing.T) {
	setupCanDeleteGroupTestDB(t)

	group := UserGroup{Name: "技能包关联组"}
	gdb.Create(&group)
	bundle := SkillBundle{Name: "test-bundle", VisibilityType: "group"}
	gdb.Create(&bundle)
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group.ID})

	ok, err := CanDeleteUserGroup(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("有技能包关联的用户组不应允许删除")
	}
}

func TestCanDeleteUserGroup_BlockedByRoleVisibility(t *testing.T) {
	setupCanDeleteGroupTestDB(t)

	group := UserGroup{Name: "角色关联组"}
	gdb.Create(&group)
	role := OpenClawRole{Name: "test-role", Soul: "test", VisibilityType: "group"}
	gdb.Create(&role)
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: group.ID})

	ok, err := CanDeleteUserGroup(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("有角色关联的用户组不应允许删除")
	}
}

func TestCanDeleteUserGroup_MultipleReferences_FirstBlocks(t *testing.T) {
	setupCanDeleteGroupTestDB(t)

	group := UserGroup{Name: "多重关联组"}
	gdb.Create(&group)
	// 同时有模型和技能包关联
	m := AIModel{ModelName: "m1", VisibilityType: "group"}
	gdb.Create(&m)
	gdb.Create(&ModelVisibilityGroup{AIModelID: m.ID, GroupID: group.ID})
	bundle := SkillBundle{Name: "b1", VisibilityType: "group"}
	gdb.Create(&bundle)
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group.ID})

	ok, err := CanDeleteUserGroup(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("有多重关联的用户组不应允许删除")
	}
}

// TestCanDeleteUserGroup_BlockedByDirectInstance 校验 v6.13：分组下直属 Agent
// （instances.group_id）应阻塞删除。
func TestCanDeleteUserGroup_BlockedByDirectInstance(t *testing.T) {
	setupCanDeleteGroupTestDB(t)

	group := UserGroup{Name: "有Agent的组"}
	gdb.Create(&group)
	gdb.Create(&Instance{Name: "agent-1", InstanceId: "ins-abc123", GroupID: group.ID})

	ok, err := CanDeleteUserGroup(context.Background(), group.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("有直属 Agent 的分组不应允许删除")
	}
}

// ── CleanupByGroupID 测试 ──────────────────────────────────────────

func TestCleanupSkillBundleVisibilityByGroupID(t *testing.T) {
	setupCanDeleteGroupTestDB(t)

	group := UserGroup{Name: "待清理组"}
	gdb.Create(&group)
	bundle := SkillBundle{Name: "b1", VisibilityType: "group"}
	gdb.Create(&bundle)
	gdb.Create(&SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: group.ID})

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return CleanupSkillBundleVisibilityByGroupID(tx, group.ID)
	})
	if err != nil {
		t.Fatalf("cleanup error: %v", err)
	}

	var count int64
	gdb.Model(&SkillBundleVisibilityGroup{}).Where("group_id = ?", group.ID).Count(&count)
	if count != 0 {
		t.Errorf("清理后应无关联，实际=%d", count)
	}
}

func TestCleanupRoleVisibilityByGroupID(t *testing.T) {
	setupCanDeleteGroupTestDB(t)

	group := UserGroup{Name: "待清理角色组"}
	gdb.Create(&group)
	role := OpenClawRole{Name: "r1", Soul: "test", VisibilityType: "group"}
	gdb.Create(&role)
	gdb.Create(&RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: group.ID})

	err := gdb.Transaction(func(tx *gorm.DB) error {
		return CleanupRoleVisibilityByGroupID(tx, group.ID)
	})
	if err != nil {
		t.Fatalf("cleanup error: %v", err)
	}

	var count int64
	gdb.Model(&RoleVisibilityGroup{}).Where("group_id = ?", group.ID).Count(&count)
	if count != 0 {
		t.Errorf("清理后应无关联，实际=%d", count)
	}
}
