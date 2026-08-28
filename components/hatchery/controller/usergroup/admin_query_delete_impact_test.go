package usergroup

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initDeleteImpactTestDB 为 delete-impact 场景准备最小 schema。
func initDeleteImpactTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 DB 失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.GroupClosure{},
		&model.GroupConfigBinding{},
		&model.Instance{},
		&model.AIModel{},
		&model.ModelVisibilityGroup{},
		&model.Skill{},
		&model.SkillVisibilityGroup{},
		&model.SkillBundle{},
		&model.SkillBundleVisibilityGroup{},
		&model.OpenClawRole{},
		&model.RoleVisibilityGroup{},
	); err != nil {
		t.Fatalf("migrate 失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

// TestGetDeleteImpact_Instances 验证 v6.13：当分组下存在直属 Agent（db.instances
// 中 group_id=X 的记录）时，delete-impact.blockers.instances 返回 instance_id + name，
// 并给出对应的 hint 文案。
func TestGetDeleteImpact_Instances(t *testing.T) {
	initDeleteImpactTestDB(t)

	g := model.UserGroup{Name: "有Agent的研发组", FullPath: "有Agent的研发组", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	// 2 台直属 Agent
	if err := model.DB(context.Background()).Create(&model.Instance{Name: "研发组-测试机-1", InstanceId: "ins-aaa111", GroupID: g.ID}).Error; err != nil {
		t.Fatalf("create instance 1: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.Instance{Name: "研发组-测试机-2", InstanceId: "ins-bbb222", GroupID: g.ID}).Error; err != nil {
		t.Fatalf("create instance 2: %v", err)
	}

	impact, err := GetDeleteImpact(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetDeleteImpact: %v", err)
	}
	if len(impact.Blockers.Instances) != 2 {
		t.Fatalf("instances 长度应为 2，实际 %d: %+v", len(impact.Blockers.Instances), impact.Blockers.Instances)
	}
	byID := map[string]string{}
	for _, inst := range impact.Blockers.Instances {
		byID[inst.InstanceID] = inst.Name
	}
	if byID["ins-aaa111"] != "研发组-测试机-1" {
		t.Fatalf("instance aaa111 name 错: %q", byID["ins-aaa111"])
	}
	if byID["ins-bbb222"] != "研发组-测试机-2" {
		t.Fatalf("instance bbb222 name 错: %q", byID["ins-bbb222"])
	}
	// instances 非空时走直属 Agent 专属 hint
	if impact.Hint == "" || impact.Hint == "此组当前无阻塞项，可直接删除。" {
		t.Fatalf("有直属 Agent 时 hint 应提示迁移/销毁 Agent，实际 %q", impact.Hint)
	}
}

// TestGetDeleteImpact_NoBlockers 验证没有任何阻塞项时 instances 字段为空数组
// （而非 nil，避免前端 null 处理），且 hint 走"可直接删除"分支。
func TestGetDeleteImpact_NoBlockers(t *testing.T) {
	initDeleteImpactTestDB(t)

	g := model.UserGroup{Name: "空组", FullPath: "空组", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	impact, err := GetDeleteImpact(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetDeleteImpact: %v", err)
	}
	if impact.Blockers.Instances == nil {
		t.Fatal("instances 字段初始应为 [] 而非 nil")
	}
	if len(impact.Blockers.Instances) != 0 {
		t.Fatalf("无阻塞时 instances 应为空，实际 %+v", impact.Blockers.Instances)
	}
	if impact.Hint != "此组当前无阻塞项，可直接删除。" {
		t.Fatalf("无阻塞时 hint 错: %q", impact.Hint)
	}
}

// TestGetDeleteImpact_ResourceBindingsCategoryKeys 验证 resource_bindings 的 key
// 对齐 config-overview 的 category key，并正确聚合资源。
func TestGetDeleteImpact_ResourceBindingsCategoryKeys(t *testing.T) {
	initDeleteImpactTestDB(t)

	g := model.UserGroup{Name: "绑定组", FullPath: "绑定组", Source: model.GroupSourceManual}
	if err := model.DB(context.Background()).Create(&g).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}

	// model: 通过旧表 model_visibility_groups 绑定
	m := model.AIModel{ModelID: "gpt-4", ModelName: "GPT-4"}
	model.DB(context.Background()).Create(&m)
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: m.ID, GroupID: g.ID})

	// skill: skill_bundle + role 应聚合到 "skill" key
	sb := model.SkillBundle{Name: "默认技能包", Enabled: true, VisibilityType: "group"}
	model.DB(context.Background()).Create(&sb)
	model.DB(context.Background()).Create(&model.SkillBundleVisibilityGroup{SkillBundleID: sb.ID, GroupID: g.ID})
	role := model.OpenClawRole{Name: "管理员角色", Visible: true, VisibilityType: "group"}
	model.DB(context.Background()).Create(&role)
	model.DB(context.Background()).Create(&model.RoleVisibilityGroup{OpenClawRoleID: role.ID, GroupID: g.ID})

	// agentTool: skill(企业技能) 应聚合到 "agentTool" key
	sk := model.Skill{Name: "企业技能A"}
	model.DB(context.Background()).Create(&sk)
	model.DB(context.Background()).Create(&model.SkillVisibilityGroup{SkillID: sk.ID, GroupID: g.ID})
	// agentTool: plugin_bundle 通过 GroupConfigBinding
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePluginBundle, ConfigKey: "plugin-1", GroupID: g.ID, ValueJSON: "{}",
	})
	// agentTool: mcp 通过 GroupConfigBinding
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypeMCP, ConfigKey: "mcp-1", GroupID: g.ID, ValueJSON: "{}",
	})

	// channel: 通过 GroupConfigBinding
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypeChannel, ConfigKey: "ch-1", GroupID: g.ID, ValueJSON: "{}",
	})

	// imageType: 通过 GroupConfigBinding
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypeImageType, ConfigKey: "img-1", GroupID: g.ID, ValueJSON: "{}",
	})

	// platformPolicy: 通过 GroupConfigBinding
	model.DB(context.Background()).Create(&model.GroupConfigBinding{
		ConfigType: model.ConfigTypePolicy, ConfigKey: "token_quota_day", GroupID: g.ID, ValueJSON: `{"limit":1000}`,
	})

	impact, err := GetDeleteImpact(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("GetDeleteImpact: %v", err)
	}

	bindings := impact.Blockers.ResourceBindings

	// 验证 model key
	if items, ok := bindings[CategoryKeyModel]; !ok || len(items) != 1 {
		t.Fatalf("expected 1 model binding, got %v", bindings[CategoryKeyModel])
	}

	// 验证 channel key
	if items, ok := bindings[CategoryKeyChannel]; !ok || len(items) != 1 {
		t.Fatalf("expected 1 channel binding, got %v", bindings[CategoryKeyChannel])
	}

	// 验证 skill key（技能包 + 角色 = 2）
	if items, ok := bindings[CategoryKeySkill]; !ok || len(items) != 2 {
		t.Fatalf("expected 2 skill bindings (bundle+role), got %v", bindings[CategoryKeySkill])
	}

	// 验证 agentTool key（企业技能 + plugin_bundle + mcp = 3）
	if items, ok := bindings[CategoryKeyAgentTool]; !ok || len(items) != 3 {
		t.Fatalf("expected 3 agentTool bindings (skill+plugin+mcp), got %v", bindings[CategoryKeyAgentTool])
	}

	// 验证 imageType key
	if items, ok := bindings[CategoryKeyImageType]; !ok || len(items) != 1 {
		t.Fatalf("expected 1 imageType binding, got %v", bindings[CategoryKeyImageType])
	}

	// 验证 platformPolicy key
	if items, ok := bindings[CategoryKeyPlatformPolicy]; !ok || len(items) != 1 {
		t.Fatalf("expected 1 platformPolicy binding, got %v", bindings[CategoryKeyPlatformPolicy])
	}

	// 不应有旧的 key
	for _, oldKey := range []string{"skill_bundle", "role", "plugin_bundle", "mcp", "image_type", "policy"} {
		if _, ok := bindings[oldKey]; ok {
			t.Fatalf("should not have old key %q in resource_bindings", oldKey)
		}
	}

	// 有阻塞项，hint 应为"解除绑定"
	if impact.Hint == "此组当前无阻塞项，可直接删除。" {
		t.Fatal("有绑定阻塞项时不应返回'可直接删除' hint")
	}
}
