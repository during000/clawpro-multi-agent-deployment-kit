package controller

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupSkillInstallDB 初始化内存 SQLite 数据库，包含 createSkillInstallTasks 需要的全部表
func setupSkillInstallDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.Instance{},
		&model.User{},
		&model.OpenClawRole{},
		&model.OpenClawRoleSkill{},
		&model.SkillBundle{},
		&model.BundleSkill{},
		&model.SkillBundleVisibilityGroup{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.SkillInstallation{},
		&model.SiteConfig{},
		&model.SMHSpace{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

func TestCreateSkillInstallTasks_NoRoleNoBundles(t *testing.T) {
	setupSkillInstallDB(t)

	inst := model.Instance{Name: "empty-inst", InstanceId: "ins-empty", UserID: 1}
	model.DB(context.Background()).Create(&inst)

	// roleID=0, 无技能包 → 不创建任何 SkillInstallation 记录
	createSkillInstallTasks(context.Background(), inst.ID, 0, inst.UserID)

	var count int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("期望 0 条安装记录，实际=%d", count)
	}
}

func TestCreateSkillInstallTasks_RoleSkillsOnly(t *testing.T) {
	setupSkillInstallDB(t)

	// 创建角色和角色技能
	role := model.OpenClawRole{Name: "测试角色", Description: "desc", Soul: "soul"}
	model.DB(context.Background()).Create(&role)
	model.DB(context.Background()).Create(&model.OpenClawRoleSkill{
		OpenClawRoleID: role.ID, Name: "技能A", Slug: "skill-a", Version: "1.0.0", CosZipKey: "/a.zip",
	})
	model.DB(context.Background()).Create(&model.OpenClawRoleSkill{
		OpenClawRoleID: role.ID, Name: "技能B", Slug: "skill-b", Version: "2.0.0", CosZipKey: "/b.zip",
	})

	inst := model.Instance{Name: "role-inst", InstanceId: "ins-role", UserID: 1}
	model.DB(context.Background()).Create(&inst)

	createSkillInstallTasks(context.Background(), inst.ID, role.ID, inst.UserID)

	var installations []model.SkillInstallation
	model.DB(context.Background()).Where("instance_id = ?", inst.ID).Find(&installations)
	if len(installations) != 2 {
		t.Fatalf("期望 2 条安装记录，实际=%d", len(installations))
	}

	slugs := map[string]bool{}
	for _, i := range installations {
		slugs[i.Slug] = true
		if i.InstallStatus != model.SkillInstallNone {
			t.Errorf("期望 install_status=%d, 实际=%d", model.SkillInstallNone, i.InstallStatus)
		}
	}
	if !slugs["skill-a"] || !slugs["skill-b"] {
		t.Errorf("期望包含 skill-a 和 skill-b, 实际=%v", slugs)
	}
}

func TestCreateSkillInstallTasks_AllTypeBundleSkills(t *testing.T) {
	setupSkillInstallDB(t)

	// 创建 all 类型技能包
	bundle := model.SkillBundle{Name: "通用包", Enabled: true, VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.BundleSkill{
		SkillBundleID: bundle.ID, Name: "通用技能", Slug: "common-skill", Version: "1.0.0", CosZipKey: "/common.zip",
	})

	user := model.User{Username: "testuser"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "bundle-inst", InstanceId: "ins-bundle", UserID: user.ID}
	model.DB(context.Background()).Create(&inst)

	createSkillInstallTasks(context.Background(), inst.ID, 0, user.ID)

	var count int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("期望 1 条安装记录（all 类型技能包），实际=%d", count)
	}
}

func TestCreateSkillInstallTasks_GroupBundleVisibility(t *testing.T) {
	setupSkillInstallDB(t)

	// 创建两个用户组
	group1 := model.UserGroup{Name: "研发组"}
	model.DB(context.Background()).Create(&group1)
	group2 := model.UserGroup{Name: "产品组"}
	model.DB(context.Background()).Create(&group2)

	// 为扁平组插入 GroupClosure 自引用记录（depth=0）
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group1.ID, DescendantID: group1.ID, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: group2.ID, DescendantID: group2.ID, Depth: 0})

	// 创建用户并加入 group1
	user := model.User{Username: "dev-user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: group1.ID, UserID: user.ID})

	// 创建 group 类型技能包，关联到 group1
	bundle1 := model.SkillBundle{Name: "研发包", Enabled: true, VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundle1)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundle1.ID, GroupID: group1.ID})
	model.DB(context.Background()).Create(&model.BundleSkill{
		SkillBundleID: bundle1.ID, Name: "研发技能", Slug: "dev-skill", Version: "1.0.0", CosZipKey: "/dev.zip",
	})

	// 创建 group 类型技能包，关联到 group2（用户不在此组）
	bundle2 := model.SkillBundle{Name: "产品包", Enabled: true, VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundle2)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundle2.ID, GroupID: group2.ID})
	model.DB(context.Background()).Create(&model.BundleSkill{
		SkillBundleID: bundle2.ID, Name: "产品技能", Slug: "pm-skill", Version: "1.0.0", CosZipKey: "/pm.zip",
	})

	inst := model.Instance{Name: "group-inst", InstanceId: "ins-grp", UserID: user.ID, GroupID: group1.ID}
	model.DB(context.Background()).Create(&inst)

	// 传入 group1.ID 作为 groupID（用户所在组）
	createSkillInstallTasks(context.Background(), inst.ID, 0, group1.ID)

	var installations []model.SkillInstallation
	model.DB(context.Background()).Where("instance_id = ?", inst.ID).Find(&installations)

	// 用户在 group1，只有 bundle1（研发包）的技能可见
	if len(installations) != 1 {
		t.Fatalf("期望 1 条安装记录（仅 group1 的技能包），实际=%d", len(installations))
	}
	if installations[0].Slug != "dev-skill" {
		t.Errorf("期望安装 dev-skill，实际=%s", installations[0].Slug)
	}
}

func TestCreateSkillInstallTasks_VersionDedup(t *testing.T) {
	setupSkillInstallDB(t)

	// 角色中有 skill-x v1.0.0
	role := model.OpenClawRole{Name: "角色", Description: "d", Soul: "s"}
	model.DB(context.Background()).Create(&role)
	model.DB(context.Background()).Create(&model.OpenClawRoleSkill{
		OpenClawRoleID: role.ID, Name: "技能X", Slug: "skill-x", Version: "1.0.0", CosZipKey: "/x1.zip",
	})

	// 技能包中也有 skill-x v2.0.0（更高版本）
	bundle := model.SkillBundle{Name: "包", Enabled: true, VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.BundleSkill{
		SkillBundleID: bundle.ID, Name: "技能X新版", Slug: "skill-x", Version: "2.0.0", CosZipKey: "/x2.zip",
	})

	user := model.User{Username: "dedup-user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "dedup-inst", InstanceId: "ins-dedup", UserID: user.ID}
	model.DB(context.Background()).Create(&inst)

	createSkillInstallTasks(context.Background(), inst.ID, role.ID, user.ID)

	var installations []model.SkillInstallation
	model.DB(context.Background()).Where("instance_id = ?", inst.ID).Find(&installations)

	// 同 slug 只保留高版本
	if len(installations) != 1 {
		t.Fatalf("期望 1 条（去重后），实际=%d", len(installations))
	}
	if installations[0].Version != "2.0.0" {
		t.Errorf("期望版本 2.0.0（高版本覆盖），实际=%s", installations[0].Version)
	}
	if installations[0].CosZipKey != "/x2.zip" {
		t.Errorf("期望 cos_zip_key=/x2.zip，实际=%s", installations[0].CosZipKey)
	}
}

func TestCreateSkillInstallTasks_RolePlusAllBundle(t *testing.T) {
	setupSkillInstallDB(t)

	// 角色技能
	role := model.OpenClawRole{Name: "混合角色", Description: "d", Soul: "s"}
	model.DB(context.Background()).Create(&role)
	model.DB(context.Background()).Create(&model.OpenClawRoleSkill{
		OpenClawRoleID: role.ID, Name: "角色技能", Slug: "role-skill", Version: "1.0.0", CosZipKey: "/r.zip",
	})

	// all 类型技能包
	bundle := model.SkillBundle{Name: "通用", Enabled: true, VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.BundleSkill{
		SkillBundleID: bundle.ID, Name: "通用技能", Slug: "bundle-skill", Version: "1.0.0", CosZipKey: "/b.zip",
	})

	user := model.User{Username: "mix-user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "mix-inst", InstanceId: "ins-mix", UserID: user.ID}
	model.DB(context.Background()).Create(&inst)

	createSkillInstallTasks(context.Background(), inst.ID, role.ID, user.ID)

	var installations []model.SkillInstallation
	model.DB(context.Background()).Where("instance_id = ?", inst.ID).Find(&installations)

	// 角色技能 + 技能包技能 = 2 条
	if len(installations) != 2 {
		t.Fatalf("期望 2 条，实际=%d", len(installations))
	}
	slugs := map[string]bool{}
	for _, i := range installations {
		slugs[i.Slug] = true
	}
	if !slugs["role-skill"] || !slugs["bundle-skill"] {
		t.Errorf("期望包含 role-skill 和 bundle-skill，实际=%v", slugs)
	}
}

func TestCreateSkillInstallTasks_DisabledBundleIgnored(t *testing.T) {
	setupSkillInstallDB(t)

	// 创建禁用的技能包
	bundle := model.SkillBundle{Name: "禁用包", Enabled: false, VisibilityType: "all"}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.BundleSkill{
		SkillBundleID: bundle.ID, Name: "禁用技能", Slug: "disabled-skill", Version: "1.0.0", CosZipKey: "/d.zip",
	})

	user := model.User{Username: "disabled-user"}
	model.DB(context.Background()).Create(&user)
	inst := model.Instance{Name: "disabled-inst", InstanceId: "ins-dis", UserID: user.ID}
	model.DB(context.Background()).Create(&inst)

	createSkillInstallTasks(context.Background(), inst.ID, 0, user.ID)

	var count int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("期望 0 条（禁用包不安装），实际=%d", count)
	}
}

func TestCreateSkillInstallTasks_GroupBundleMultiGroup(t *testing.T) {
	setupSkillInstallDB(t)

	// 创建用户组
	g1 := model.UserGroup{Name: "组1"}
	model.DB(context.Background()).Create(&g1)
	g2 := model.UserGroup{Name: "组2"}
	model.DB(context.Background()).Create(&g2)

	// 为扁平组插入 GroupClosure 自引用记录
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: g1.ID, DescendantID: g1.ID, Depth: 0})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: g2.ID, DescendantID: g2.ID, Depth: 0})

	// 用户同时属于 g1 和 g2
	user := model.User{Username: "multi-group-user"}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: g1.ID, UserID: user.ID})
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: g2.ID, UserID: user.ID})

	// 技能包关联到 g2
	bundle := model.SkillBundle{Name: "组2包", Enabled: true, VisibilityType: "group"}
	model.DB(context.Background()).Create(&bundle)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: bundle.ID, GroupID: g2.ID})
	model.DB(context.Background()).Create(&model.BundleSkill{
		SkillBundleID: bundle.ID, Name: "组2技能", Slug: "g2-skill", Version: "1.0.0", CosZipKey: "/g2.zip",
	})

	inst := model.Instance{Name: "mg-inst", InstanceId: "ins-mg", UserID: user.ID, GroupID: g2.ID}
	model.DB(context.Background()).Create(&inst)

	// 用户实例在 g2，bundle 关联 g2 → 应命中
	createSkillInstallTasks(context.Background(), inst.ID, 0, g2.ID)

	var count int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 1 {
		t.Errorf("期望 1 条（用户在 g2 中），实际=%d", count)
	}
}
