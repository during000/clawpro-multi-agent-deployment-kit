package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func initPluginBundleTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试DB失败: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	db.Create(&model.SiteConfig{SMHEnabled: 1})
}

func TestPluginBundleHandlers_MethodNotAllowed(t *testing.T) {
	initPluginBundleTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		method  string
		path    string
	}{
		{"CreatePluginBundle", HandleCreatePluginBundle, http.MethodGet, "/admin/plugin-bundles/create"},
		{"DeletePluginBundle", HandleDeletePluginBundle, http.MethodGet, "/admin/plugin-bundles/delete?id=1"},
		{"TogglePluginBundle", HandleTogglePluginBundle, http.MethodGet, "/admin/plugin-bundles/toggle?id=1"},
		{"UpdatePlugins", HandleUpdatePluginBundlePlugins, http.MethodGet, "/admin/plugin-bundles/update-plugins?id=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", "Bearer test-admin-token")
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s: expected 405, got %d", tt.name, w.Code)
			}
		})
	}
}
