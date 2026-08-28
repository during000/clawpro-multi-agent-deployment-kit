package controller

import (
	"context"
	"testing"

	"hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// setupNoticesTestDB 初始化内存 SQLite 数据库，用于 admin_notices 测试
func setupNoticesTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.AIModel{},
		&model.AIChannel{},
		&model.SiteConfig{},
		&model.RuleSet{},
		&model.GroupConfigBinding{},
	); err != nil {
		t.Fatalf("迁移测试数据库失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	// 确保 FixedSnapshot 非 nil，供各测试操作字段
	if common.FixedSnapshot == nil {
		old := common.FixedSnapshot
		common.FixedSnapshot = &common.TenantSnapshot{}
		t.Cleanup(func() { common.FixedSnapshot = old })
	}
}

func TestBuildConfigSteps_StandardMode_Keys(t *testing.T) {
	setupNoticesTestDB(t)
	origSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{
		Identifier:     origSnap.Identifier,
		Domain:         origSnap.Domain,
		Uin:            origSnap.Uin,
		OneIDAccountID: "",
	}
	defer func() { common.FixedSnapshot = origSnap }()

	steps := buildConfigSteps(context.Background())
	if len(steps) != 8 {
		t.Fatalf("标准模式期望 8 个步骤，实际 %d", len(steps))
	}
	expectedKeys := []string{"brand", "default_quota", "users", "model", "channel", "vpc", "security_group", "image"}
	for i, key := range expectedKeys {
		if steps[i].Key != key {
			t.Errorf("步骤 %d: 期望 key=%q，实际 %q", i, key, steps[i].Key)
		}
	}
}

func TestBuildConfigSteps_StandardMode_Labels(t *testing.T) {
	setupNoticesTestDB(t)
	origSnap := common.FixedSnapshot
	common.FixedSnapshot = &common.TenantSnapshot{
		Identifier:     origSnap.Identifier,
		Domain:         origSnap.Domain,
		Uin:            origSnap.Uin,
		OneIDAccountID: "",
	}
	defer func() { common.FixedSnapshot = origSnap }()

	steps := buildConfigSteps(context.Background())
	expectedLabels := []string{
		"设置平台名称与品牌",
		"配置用户默认配额",
		"导入企业用户",
		"配置至少一个模型",
		"配置至少一个通道",
		"配置私有网络",
		"配置安全组",
		"配置至少一个镜像",
	}
	for i, label := range expectedLabels {
		if steps[i].Label != label {
			t.Errorf("步骤 %d: 期望 label=%q，实际 %q", i, label, steps[i].Label)
		}
	}
}

func TestBuildConfigSteps_OneIDMode_Keys(t *testing.T) {
	setupNoticesTestDB(t)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		Identifier:     "",
		OneIDAccountID: "test-tenant",
	})

	steps := buildConfigSteps(ctx)
	if len(steps) != 9 {
		t.Fatalf("OneID 模式期望 9 个步骤，实际 %d", len(steps))
	}
	expectedKeys := []string{"brand", "default_quota", "users", "sso_login", "model", "channel", "vpc", "security_group", "image"}
	for i, key := range expectedKeys {
		if steps[i].Key != key {
			t.Errorf("步骤 %d: 期望 key=%q，实际 %q", i, key, steps[i].Key)
		}
	}
}

func TestBuildConfigSteps_OneIDMode_Labels(t *testing.T) {
	setupNoticesTestDB(t)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		Identifier:     "",
		OneIDAccountID: "test-tenant",
	})
	steps := buildConfigSteps(ctx)
	expectedLabels := []string{
		"设置平台名称与品牌",
		"配置用户默认配额",
		"导入企业用户",
		"设置用户登录方式",
		"配置至少一个模型",
		"配置至少一个通道",
		"配置私有网络",
		"配置安全组",
		"配置至少一个镜像",
	}
	for i, label := range expectedLabels {
		if steps[i].Label != label {
			t.Errorf("步骤 %d: 期望 label=%q，实际 %q", i, label, steps[i].Label)
		}
	}
}

func TestBuildConfigSteps_OneIDMode_SSOLoginPosition(t *testing.T) {
	setupNoticesTestDB(t)
	ctx := common.InjectTenant(context.Background(), common.TenantSnapshot{
		Identifier:     "",
		OneIDAccountID: "test-tenant",
	})

	steps := buildConfigSteps(ctx)
	// sso_login 应在 users 之后、model 之前（索引 3）
	if steps[3].Key != "sso_login" {
		t.Errorf("期望第 4 步（索引 3）为 sso_login，实际 %q", steps[3].Key)
	}
	if steps[4].Key != "model" {
		t.Errorf("期望第 5 步（索引 4）为 model，实际 %q", steps[4].Key)
	}
}

func TestHasEnterpriseUsers_OnlyAdmin(t *testing.T) {
	setupNoticesTestDB(t)
	// 只有一个管理员
	model.DB(context.Background()).Create(&model.User{Username: "admin", Role: "admin", Password: "hash"})
	if hasEnterpriseUsers(context.Background()) {
		t.Error("只有初始管理员，期望 hasEnterpriseUsers()=false")
	}
}

func TestHasEnterpriseUsers_WithImportedUser(t *testing.T) {
	setupNoticesTestDB(t)
	// 一个管理员 + 一个普通用户
	model.DB(context.Background()).Create(&model.User{Username: "admin", Role: "admin", Password: "hash"})
	model.DB(context.Background()).Create(&model.User{Username: "user1", Role: "user", Password: "hash"})
	if !hasEnterpriseUsers(context.Background()) {
		t.Error("有导入用户，期望 hasEnterpriseUsers()=true")
	}
}

func TestHasEnterpriseUsers_TwoAdmins(t *testing.T) {
	setupNoticesTestDB(t)
	// 两个管理员也算有用户（count > 1）
	model.DB(context.Background()).Create(&model.User{Username: "admin1", Role: "admin", Password: "hash"})
	model.DB(context.Background()).Create(&model.User{Username: "admin2", Role: "admin", Password: "hash"})
	if !hasEnterpriseUsers(context.Background()) {
		t.Error("有两个用户（均为管理员），期望 hasEnterpriseUsers()=true")
	}
}
