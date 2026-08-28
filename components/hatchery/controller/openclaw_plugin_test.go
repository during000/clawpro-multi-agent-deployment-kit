package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initPluginTestDB 初始化内存 SQLite 数据库用于插件安装测试
func initPluginTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.Instance{}, &model.PluginInstallation{}, &model.SiteConfig{}); err != nil {
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

func TestCreatePluginInstallTasks_SkipsUnsupportedType(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	tests := []struct {
		name       string
		agentType  string
		expectSkip bool
	}{
		{"hermes should skip", "hermes", true},
		{"lightclawace should skip", "lightclawace", true},
		{"openclaw should not skip", "openclaw", false},
		{"empty (legacy) should not skip", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.DB(context.Background()).Where("1=1").Delete(&model.Instance{})
			model.DB(context.Background()).Where("1=1").Delete(&model.PluginInstallation{})

			inst := model.Instance{
				Name:       "test-instance",
				InstanceId: "ins-test-001",
				UserID:     1,
				AgentType:  tt.agentType,
			}
			model.DB(context.Background()).Create(&inst)

			createPluginInstallTasks(context.Background(), inst.ID, 0)

			var count int64
			model.DB(context.Background()).Model(&model.PluginInstallation{}).Where("instance_id = ?", inst.ID).Count(&count)
			if count != 0 {
				t.Errorf("expected no plugin installations, got %d", count)
			}
		})
	}
}

func TestCreatePluginInstallTasks_InstanceNotFound(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	// 调用不存在的实例 ID，应该不 panic
	createPluginInstallTasks(context.Background(), 99999, 0)
}

// ─── Handler 级别测试：类型防护 ──────────────────────────────────────────

func pluginReqWithSession(t *testing.T, method, path string, username string, body string) *http.Request {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")

	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = username

	rr := httptest.NewRecorder()
	session.Save(req, rr)

	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	return req
}

func TestHandleAddPlugin_UnsupportedAgentType(t *testing.T) {
	cleanup := initPluginTestDB(t)
	defer cleanup()

	// 创建用户和 hermes 类型实例
	user := &model.User{Username: "testuser", Password: "test", Role: "user"}
	model.DB(context.Background()).Create(user)

	proxyToken := "sk-test"
	inst := model.Instance{
		Name:       "hermes-inst",
		InstanceId: "ins-hermes-001",
		UserID:     user.ID,
		AgentType:  "hermes",
		ProxyToken: &proxyToken,
	}
	model.DB(context.Background()).Create(&inst)

	form := url.Values{}
	form.Set("plugin", "test-plugin")
	req := pluginReqWithSession(t, http.MethodPost, fmt.Sprintf("/openclaw/plugin?id=%d", inst.ID), "testuser", form.Encode())
	rr := httptest.NewRecorder()

	handleAddPlugin(rr, req, testCVMFetcher)

	if rr.Code != http.StatusForbidden {
		t.Errorf("hermes 实例添加插件应返回 403，实际=%d, body=%s", rr.Code, rr.Body.String())
	}
}
