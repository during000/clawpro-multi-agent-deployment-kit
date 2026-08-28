package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

func initOneIDJumpTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试DB失败: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.SiteConfig{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	db.Create(&model.SiteConfig{})
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
}

func TestHandleOneIDJump_Unauthorized(t *testing.T) {
	initOneIDJumpTestDB(t)
	origGW := GatewayURL
	GatewayURL = "https://gateway.example.com"
	defer func() { GatewayURL = origGW }()
	req := httptest.NewRequest(http.MethodGet, "/admin/oneid/jump", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	HandleOneIDJump(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleOneIDJump_Forbidden(t *testing.T) {
	initOneIDJumpTestDB(t)
	origGW := GatewayURL
	GatewayURL = "https://gateway.example.com"
	defer func() { GatewayURL = origGW }()
	db := model.DB(context.Background())
	db.Create(&model.User{Username: "noadmin", Role: "user"})
	req := httptest.NewRequest(http.MethodGet, "/admin/oneid/jump", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "noadmin"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, c := range rr.Result().Cookies() {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	HandleOneIDJump(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}
