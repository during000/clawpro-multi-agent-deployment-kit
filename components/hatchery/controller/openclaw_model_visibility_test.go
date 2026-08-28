package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initModelVisibilityTestDB 初始化包含分组表的内存数据库
func initModelVisibilityTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Instance{}, &model.AIModel{}, &model.SiteConfig{},
		&model.InstanceModel{}, &model.UserGroup{}, &model.GroupClosure{},
		&model.ModelVisibilityGroup{}, &model.GroupConfigBinding{},
	); err != nil {
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

// TestHandleSetModel_VisibilityAll_ZeroGroupInstance 测试实例未分组时可以设置全局可见模型
func TestHandleSetModel_VisibilityAll_ZeroGroupInstance(t *testing.T) {
	cleanup := initModelVisibilityTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	// 实例 group_id=0（未分组）
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-vis-all",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		GroupID: 0,
	}
	model.DB(context.Background()).Create(inst)

	aiModel := model.AIModel{
		Provider: "p1", ModelID: "m1", APIKey: "k", URL: "http://x",
		ModelType: "openai-completions", Enabled: true, Visible: true, VisibilityType: "all",
	}
	model.DB(context.Background()).Create(&aiModel)

	form := url.Values{}
	form.Set("ai_model_id", fmt.Sprintf("%d", aiModel.ID))
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	// 不应返回 403（可能因 domain 空返回 500，但不是可见性拒绝）
	if rr.Code == http.StatusForbidden {
		t.Errorf("全局可见模型不应被拒绝，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleSetModel_VisibilityGroup_ZeroGroupInstance 测试实例未分组时不能设置分组模型
func TestHandleSetModel_VisibilityGroup_ZeroGroupInstance(t *testing.T) {
	cleanup := initModelVisibilityTestDB(t)
	defer cleanup()

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-vis-grp",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		GroupID: 0,
	}
	model.DB(context.Background()).Create(inst)

	aiModel := model.AIModel{
		Provider: "p1", ModelID: "m-group", APIKey: "k", URL: "http://x",
		ModelType: "openai-completions", Enabled: true, Visible: true, VisibilityType: "group",
	}
	model.DB(context.Background()).Create(&aiModel)
	// 模型绑定到分组 1，但实例不在任何分组
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: aiModel.ID, GroupID: 1})

	form := url.Values{}
	form.Set("ai_model_id", fmt.Sprintf("%d", aiModel.ID))
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("未分组实例应被拒绝使用分组模型，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

// TestHandleSetModel_VisibilityGroup_MatchingGroupInstance 测试实例分组匹配时可以设置
func TestHandleSetModel_VisibilityGroup_MatchingGroupInstance(t *testing.T) {
	cleanup := initModelVisibilityTestDB(t)
	defer cleanup()

	// 创建分组 + closure
	model.DB(context.Background()).Create(&model.UserGroup{ID: 5, Name: "DevGroup", Source: "manual", FullPath: "DevGroup"})
	model.DB(context.Background()).Create(&model.GroupClosure{AncestorID: 5, DescendantID: 5, Depth: 0})

	user := &model.User{Username: "u1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		Name: "inst", InstanceId: "ins-vis-match",
		UserID: user.ID, AgentType: model.AgentTypeOpenClaw,
		GroupID: 5,
	}
	model.DB(context.Background()).Create(inst)

	aiModel := model.AIModel{
		Provider: "p1", ModelID: "m-group", APIKey: "k", URL: "http://x",
		ModelType: "openai-completions", Enabled: true, Visible: true, VisibilityType: "group",
	}
	model.DB(context.Background()).Create(&aiModel)
	model.DB(context.Background()).Create(&model.ModelVisibilityGroup{AIModelID: aiModel.ID, GroupID: 5})

	form := url.Values{}
	form.Set("ai_model_id", fmt.Sprintf("%d", aiModel.ID))
	req := modelReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/model?id=%d", inst.ID), "u1", form.Encode())
	rr := httptest.NewRecorder()
	handleSetModel(rr, req, testCVMFetcher)

	// 不应返回 403
	if rr.Code == http.StatusForbidden {
		t.Errorf("实例分组匹配的模型不应被拒绝，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}
