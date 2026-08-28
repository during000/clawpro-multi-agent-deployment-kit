package controller

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initInstanceCountTestDB 初始化 toAdminJSON 测试需要的 DB。
func initInstanceCountTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Instance{}, &model.UserGroup{},
		&model.GroupClosure{}, &model.UserGroupMember{}, &model.SiteConfig{},
		// toAdminJSON → enrichUsersWithProjectInfo 查询 project_members / projects
		&model.ProjectMember{}, &model.Project{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

// TestToAdminJSON_GroupInstanceCount_DoctorNodeExcluded 验证 /admin/users 响应里
// groups[].instance_count 不会把龙虾医生节点算进去。
//
// 场景：alice 在 group G 下有 1 个普通实例 + 2 个龙虾医生节点 → 期望 instance_count=1。
func TestToAdminJSON_GroupInstanceCount_DoctorNodeExcluded(t *testing.T) {
	cleanup := initInstanceCountTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 1) 创建用户 + 分组
	alice := &model.User{Username: "alice", Password: "x", Role: "user"}
	model.DB(ctx).Create(alice)
	g := &model.UserGroup{Name: "G", Source: "manual", FullPath: "/G"}
	model.DB(ctx).Create(g)
	model.DB(ctx).Create(&model.GroupClosure{
		AncestorID: g.ID, DescendantID: g.ID, Depth: 0,
	})
	model.DB(ctx).Create(&model.UserGroupMember{
		UserID: alice.ID, UserGroupID: g.ID,
	})

	// 2) 1 个普通实例 + 2 个龙虾医生节点
	model.DB(ctx).Create(&model.Instance{
		Name: "n1", InstanceId: "ins-n1",
		UserID: alice.ID, GroupID: g.ID,
		AgentType:    model.AgentTypeOpenClaw,
		IsDoctorNode: false,
	})
	for i := 0; i < 2; i++ {
		model.DB(ctx).Create(&model.Instance{
			Name: "d", InstanceId: "ins-d",
			UserID: alice.ID, GroupID: g.ID,
			AgentType:    model.AgentTypeOpenClaw,
			IsDoctorNode: true,
		})
	}

	// 3) 调用 toAdminJSON
	users := []userWithDept{{User: *alice}}
	out, err := toAdminJSON(ctx, users)
	if err != nil {
		t.Fatalf("toAdminJSON 失败: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("应返回 1 个用户，实际=%d", len(out))
	}

	// 4) 找到 alice 的 G group brief，验证 InstanceCount=1
	found := false
	for _, gp := range out[0].Groups {
		if gp.ID == g.ID {
			found = true
			if gp.InstanceCount != 1 {
				t.Errorf("instance_count 应为 1（仅普通实例），实际=%d", gp.InstanceCount)
			}
		}
	}
	if !found {
		t.Errorf("响应未找到 group=%d 的 brief", g.ID)
	}
}
