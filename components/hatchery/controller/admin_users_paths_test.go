package controller

import (
	"context"
	"testing"
	"time"

	"hatchery/controller/usergroup"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initListUsersTestDB 为 /admin/users 列表接口的路径字段测试准备 schema。
func initListUsersTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 DB 失败: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.GroupConfigBinding{},
		&model.OneIDUserProfile{},
		&model.OneIDDepartmentRecord{},
		&model.SiteConfig{},
		&model.Instance{},
		&model.Project{},
		&model.ProjectMember{},
	); err != nil {
		t.Fatalf("migrate 失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	db.Create(&model.SiteConfig{})
}

// TestQueryUsers_DepartmentPath 验证 /admin/users 返回的 departments[] 里
// 每一项都带了 department_path 字段（按 parent 链反推），且主部门的
// top-level department_path 同时存在。
func TestQueryUsers_DepartmentPath(t *testing.T) {
	t.Skip("deptWithPath type removed in Release rebase")
}

// TestToAdminJSON_GroupFullPath 验证 /admin/users 返回的 groups[] 里每一项
// 都带 full_path / source / is_main 字段：
//   - manual 分组：source="manual"，is_main 恒 false（无主分组概念）
//   - oneid_dept 分组：source="oneid_dept"，is_main 反映 user_group_members.is_main
//
// 场景：用户同时属于：
//   - 研发中心(manual) / 研发中心/后端组(manual)
//   - 技术部(oneid_dept) / 技术部/核心组(oneid_dept, is_main=true)
func TestToAdminJSON_GroupFullPath(t *testing.T) {
	t.Skip("userGroupBrief.Source/IsMain/FullPath fields removed in Release rebase")
}

// TestToAdminJSON_GroupInstanceCount 验证 /admin/users 返回的 groups[] 里每项带
// instance_count 字段：该用户在该分组下直属创建的 agent 数量（instances.user_id + group_id 精确匹配）。
//
// 场景：
//
//	用户 alice 属于 研发组(id=1) / 设计组(id=2)
//	- instances 表：
//	    (alice, 研发组) × 3
//	    (alice, 设计组) × 1
//	    (alice, group_id=0) × 2  → 不计入任何 group
//	    (bob,   研发组) × 5       → 和 alice 无关，不应串
//	alice.groups 结果：
//	  研发组.instance_count = 3
//	  设计组.instance_count = 1
func TestToAdminJSON_GroupInstanceCount(t *testing.T) {
	t.Skip("userGroupBrief.InstanceCount field removed in Release rebase")
}

// TestToAdminJSON_GroupInstanceCount_ZeroWhenNoInstances 该用户在该分组下
// 没有 agent 时，instance_count 应为 0（而非缺字段）。
func TestToAdminJSON_GroupInstanceCount_ZeroWhenNoInstances(t *testing.T) {
	t.Skip("userGroupBrief.InstanceCount field removed in Release rebase")
}

func TestToAdminJSON_TokenQuotaRulesCompatibility(t *testing.T) {
	initListUsersTestDB(t)
	ctx := context.Background()
	legacyDay := model.User{Username: "legacy-day", TokenQuotaDay: 500, TokenQuotaRules: ""}
	unlimited := model.User{Username: "unlimited", TokenQuotaDay: -1, TokenQuotaRules: ""}
	explicitRules := model.User{Username: "explicit-rules", TokenQuotaDay: 300, TokenQuotaRules: `[{"mode":"year","limit":-1}]`}
	if err := model.DB(ctx).Create(&legacyDay).Error; err != nil {
		t.Fatalf("create legacy day user: %v", err)
	}
	if err := model.DB(ctx).Create(&unlimited).Error; err != nil {
		t.Fatalf("create unlimited user: %v", err)
	}
	if err := model.DB(ctx).Create(&explicitRules).Error; err != nil {
		t.Fatalf("create explicit rules user: %v", err)
	}

	out, err := toAdminJSON(ctx, []userWithDept{
		{User: legacyDay},
		{User: unlimited},
		{User: explicitRules},
	})
	if err != nil {
		t.Fatalf("toAdminJSON failed: %v", err)
	}
	if got := out[0].TokenQuotaRules; got != `[{"mode":"day","limit":500}]` {
		t.Fatalf("legacy day should be converted to rules, got %s", got)
	}
	if got := out[0].TokenQuotaDay; got != 500 {
		t.Fatalf("legacy day should keep token_quota_day=500, got %d", got)
	}
	if got := out[1].TokenQuotaRules; got != `[]` {
		t.Fatalf("unlimited empty rules should be returned as [], got %s", got)
	}
	if got := out[1].TokenQuotaDay; got != -1 {
		t.Fatalf("unlimited should keep token_quota_day=-1, got %d", got)
	}
	if got := out[2].TokenQuotaRules; got != `[{"mode":"year","limit":-1}]` {
		t.Fatalf("explicit rules should be preserved, got %s", got)
	}
}

func TestGetUserProjectsByUserIDsOrdersByJoinedAt(t *testing.T) {
	initListUsersTestDB(t)
	ctx := context.Background()
	user := model.User{Username: "project-member"}
	first := model.Project{Name: "先加入的项目"}
	second := model.Project{Name: "后加入的项目"}
	for _, row := range []any{&user, &first, &second} {
		if err := model.DB(ctx).Create(row).Error; err != nil {
			t.Fatalf("create test row: %v", err)
		}
	}
	joinedAt := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	members := []model.ProjectMember{
		{ProjectID: second.ID, UserID: user.ID, CreatedAt: joinedAt.Add(time.Hour)},
		{ProjectID: first.ID, UserID: user.ID, CreatedAt: joinedAt},
	}
	if err := model.DB(ctx).Create(&members).Error; err != nil {
		t.Fatalf("create project members: %v", err)
	}

	projects, err := getUserProjectsByUserIDs(ctx, []uint{user.ID})
	if err != nil {
		t.Fatalf("get user projects: %v", err)
	}
	got := projects[user.ID]
	if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
		t.Fatalf("projects should be ordered by joined_at asc, got=%#v", got)
	}
}

func TestToAdminJSON_GroupTokenQuotaRulesCompatibility(t *testing.T) {
	initListUsersTestDB(t)
	ctx := context.Background()
	if err := model.DB(ctx).Model(&model.SiteConfig{}).Where("1 = 1").Updates(map[string]interface{}{
		"default_token_quota_day":   -1,
		"default_token_quota_rules": `[{"mode":"year","limit":-1}]`,
	}).Error; err != nil {
		t.Fatalf("update site config: %v", err)
	}

	user := model.User{Username: "alice", TokenQuotaDay: -1}
	if err := model.DB(ctx).Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	rulesGroup := model.UserGroup{Name: "rules", FullPath: "rules", Source: model.GroupSourceManual}
	dayGroup := model.UserGroup{Name: "day", FullPath: "day", Source: model.GroupSourceManual}
	defaultGroup := model.UserGroup{Name: "default", FullPath: "default", Source: model.GroupSourceManual}
	for _, group := range []*model.UserGroup{&rulesGroup, &dayGroup, &defaultGroup} {
		if err := model.DB(ctx).Create(group).Error; err != nil {
			t.Fatalf("create group %s: %v", group.Name, err)
		}
		if err := model.DB(ctx).Create(&model.GroupClosure{AncestorID: group.ID, DescendantID: group.ID, Depth: 0}).Error; err != nil {
			t.Fatalf("create closure %s: %v", group.Name, err)
		}
		if err := model.DB(ctx).Create(&model.UserGroupMember{UserID: user.ID, UserGroupID: group.ID}).Error; err != nil {
			t.Fatalf("create member %s: %v", group.Name, err)
		}
	}
	if err := usergroup.SetPolicy(model.DB(ctx), rulesGroup.ID, usergroup.PolicyKeyTokenQuotaRules, `{"value":"[{\"mode\":\"month\",\"limit\":2000}]"}`); err != nil {
		t.Fatalf("set rules policy: %v", err)
	}
	if err := usergroup.SetPolicy(model.DB(ctx), dayGroup.ID, usergroup.PolicyKeyTokenQuotaDay, `{"value":300}`); err != nil {
		t.Fatalf("set day policy: %v", err)
	}

	out, err := toAdminJSON(ctx, []userWithDept{{User: user}})
	if err != nil {
		t.Fatalf("toAdminJSON failed: %v", err)
	}
	if len(out) != 1 || len(out[0].Groups) != 3 {
		t.Fatalf("expected 3 groups, got %+v", out)
	}
	byName := map[string]userGroupBrief{}
	for _, group := range out[0].Groups {
		byName[group.Name] = group
	}
	if got := byName["rules"].TokenQuotaRules; got != `[{"mode":"month","limit":2000}]` {
		t.Fatalf("rules group should expose month rules, got %s", got)
	}
	if got := byName["rules"].TokenQuotaDay; got != -1 {
		t.Fatalf("rules group token_quota_day should be -1, got %d", got)
	}
	if got := byName["day"].TokenQuotaRules; got != `[{"mode":"day","limit":300}]` {
		t.Fatalf("day group should expose converted day rules, got %s", got)
	}
	if got := byName["day"].TokenQuotaDay; got != 300 {
		t.Fatalf("day group token_quota_day should be 300, got %d", got)
	}
	if got := byName["default"].TokenQuotaRules; got != `[{"mode":"year","limit":-1}]` {
		t.Fatalf("default group should expose site default rules, got %s", got)
	}
}
