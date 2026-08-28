package controller

import (
	"context"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initSkillTestDB 初始化内存 SQLite 数据库用于技能安装测试
func initSkillTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.SkillInstallation{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	origDB := model.UseDBForTest(db)
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

func TestCreateSkillInstallTasks_NoSkillBundle(t *testing.T) {
	cleanup := initSkillTestDB(t)
	defer cleanup()

	// 创建实例
	inst := model.Instance{
		Name:       "test-instance",
		InstanceId: "ins-test-001",
		UserID:     1,
		AgentType:  "openclaw",
	}
	model.DB(context.Background()).Create(&inst)

	// 调用 createSkillInstallTasks（无角色、无技能包场景下不会创建记录）
	createSkillInstallTasks(context.Background(), inst.ID, 0, inst.UserID)

	// 验证：没有技能包时不会创建安装记录
	var count int64
	model.DB(context.Background()).Model(&model.SkillInstallation{}).Where("instance_id = ?", inst.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected no skill installations, got %d", count)
	}
}

func TestCreateSkillInstallTasks_InstanceNotFound(t *testing.T) {
	cleanup := initSkillTestDB(t)
	defer cleanup()

	// 调用不存在的实例 ID，应该不 panic
	createSkillInstallTasks(context.Background(), 99999, 0, 0)
}
