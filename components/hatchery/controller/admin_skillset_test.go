package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func initSkillSetTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试DB失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.SiteConfig{}, &model.PublicSkillSet{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}

func seedSkillSet(t *testing.T, slug string) *model.PublicSkillSet {
	ss := model.PublicSkillSet{Slug: slug}
	if err := model.DB(context.Background()).Create(&ss).Error; err != nil {
		t.Fatalf("创建 PublicSkillSet 失败: %v", err)
	}
	return &ss
}

// ── 405 测试 ──────────────────────────────────────────────────────────

func TestSkillSetHandlers_MethodNotAllowed(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		method  string
		path    string
	}{
		{"FavoriteSkillSet", HandleFavoriteSkillSet, http.MethodGet, "/admin/skillsets/favorite"},
		{"UnfavoriteSkillSet", HandleUnfavoriteSkillSet, http.MethodGet, "/admin/skillsets/unfavorite?id=1"},
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

// ── Favorite ──────────────────────────────────────────────────────────

func TestHandleFavoriteSkillSet_Success(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	body := `{"slug": "finance-risk-assessment"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/skillsets/favorite", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleFavoriteSkillSet(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Error("expected ok=true")
	}
	if id, ok := resp["skillset_id"].(float64); !ok || id == 0 {
		t.Error("expected skillset_id > 0")
	}
}

func TestHandleFavoriteSkillSet_MissingSlug(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/skillsets/favorite", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	HandleFavoriteSkillSet(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ── Unfavorite ────────────────────────────────────────────────────────

func TestHandleUnfavoriteSkillSet_ById(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	ss := seedSkillSet(t, "test-skillset")

	req := httptest.NewRequest(http.MethodPost, "/admin/skillsets/unfavorite?id="+strconv.FormatUint(uint64(ss.ID), 10), nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleUnfavoriteSkillSet(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 确认已删除
	var count int64
	model.DB(context.Background()).Model(&model.PublicSkillSet{}).Where("id = ?", ss.ID).Count(&count)
	if count != 0 {
		t.Error("expected skillset to be deleted")
	}
}

func TestHandleUnfavoriteSkillSet_BySlug(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	seedSkillSet(t, "test-skillset")

	req := httptest.NewRequest(http.MethodPost, "/admin/skillsets/unfavorite?slug=test-skillset", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleUnfavoriteSkillSet(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int64
	model.DB(context.Background()).Model(&model.PublicSkillSet{}).Where("slug = ?", "test-skillset").Count(&count)
	if count != 0 {
		t.Error("expected skillset to be deleted")
	}
}

func TestHandleUnfavoriteSkillSet_NoParams(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/skillsets/unfavorite", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleUnfavoriteSkillSet(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleUnfavoriteSkillSet_BothParams(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/skillsets/unfavorite?id=1&slug=test", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleUnfavoriteSkillSet(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleUnfavoriteSkillSet_NotFound(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodPost, "/admin/skillsets/unfavorite?slug=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleUnfavoriteSkillSet(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ── Favorited ─────────────────────────────────────────────────────────

func TestHandleAdminFavoritedSkillSets_Empty(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	req := httptest.NewRequest(http.MethodGet, "/admin/skillsets/favorited", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminFavoritedSkillSets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	skillsets, _ := resp["skillsets"].([]interface{})
	if len(skillsets) != 0 {
		t.Errorf("expected empty skillsets, got %d", len(skillsets))
	}
}

func TestHandleAdminFavoritedSkillSets_WithData(t *testing.T) {
	initSkillSetTestDB(t)
	origToken := AdminToken
	AdminToken = "test-admin-token"
	defer func() { AdminToken = origToken }()

	seedSkillSet(t, "skillset-a")
	seedSkillSet(t, "skillset-b")

	req := httptest.NewRequest(http.MethodGet, "/admin/skillsets/favorited", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	HandleAdminFavoritedSkillSets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if total, ok := resp["total"].(float64); !ok || total != 2 {
		t.Errorf("expected total=2, got %v", resp["total"])
	}
	skillsets, _ := resp["skillsets"].([]interface{})
	if len(skillsets) != 2 {
		t.Errorf("expected 2 skillsets, got %d", len(skillsets))
	}
}

