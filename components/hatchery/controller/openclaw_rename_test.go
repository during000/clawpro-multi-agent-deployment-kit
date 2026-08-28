package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	hcommon "hatchery/common"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

func initRenameTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Instance{}, &model.SiteConfig{},
		&model.CustomAgentType{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	t.Cleanup(func() {
		origDB()
		Store = origStore
	})
}

func renameReqWithSession(t *testing.T, method, path, username, body string) *http.Request {
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

// ─── HandleRenameInstance Tests ─────────────────────────────────────────

func TestHandleRenameInstance_MethodNotAllowed(t *testing.T) {
	initRenameTestDB(t)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/rename", nil)
	rr := httptest.NewRecorder()
	HandleRenameInstance(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleRenameInstance_Unauthenticated(t *testing.T) {
	initRenameTestDB(t)

	req := httptest.NewRequest(http.MethodPost, "/openclaw/rename", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleRenameInstance(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestHandleRenameInstance_MissingID(t *testing.T) {
	initRenameTestDB(t)

	model.DB(context.Background()).Create(&model.User{Username: "user1", Password: "x", Role: "user"})

	req := renameReqWithSession(t, http.MethodPost, "/openclaw/rename", "user1", "name=newname")
	rr := httptest.NewRecorder()
	HandleRenameInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if msg, _ := resp["error"].(string); !strings.Contains(msg, "id") && !strings.Contains(msg, "instance_id") {
		t.Errorf("expected error about missing id, got %q", msg)
	}
}

func TestHandleRenameInstance_InstanceNotFound(t *testing.T) {
	initRenameTestDB(t)

	model.DB(context.Background()).Create(&model.User{Username: "user1", Password: "x", Role: "user"})

	req := renameReqWithSession(t, http.MethodPost, "/openclaw/rename?id=999", "user1", "name=newname")
	rr := httptest.NewRecorder()
	HandleRenameInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleRenameInstance_EmptyName(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "user1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{UserID: user.ID, InstanceId: "ins-test123", Name: "old"}
	model.DB(context.Background()).Create(inst)

	req := renameReqWithSession(t, http.MethodPost, "/openclaw/rename?id=1", "user1", "name=")
	rr := httptest.NewRecorder()
	HandleRenameInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if msg, _ := resp["error"].(string); !strings.Contains(msg, "名称") {
		t.Errorf("expected error about name, got %q", msg)
	}
}

func TestHandleRenameInstance_NameTooLong(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "user1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{UserID: user.ID, InstanceId: "ins-test123", Name: "old"}
	model.DB(context.Background()).Create(inst)

	longName := strings.Repeat("a", 129)
	body := "name=" + url.QueryEscape(longName)
	req := renameReqWithSession(t, http.MethodPost, "/openclaw/rename?id=1", "user1", body)
	rr := httptest.NewRecorder()
	HandleRenameInstance(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleRenameInstance_NoCVMInstance(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "user1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	// InstanceId 为空 — 还没创建 CVM
	inst := &model.Instance{UserID: user.ID, InstanceId: "", Name: "old"}
	model.DB(context.Background()).Create(inst)

	req := renameReqWithSession(t, http.MethodPost, "/openclaw/rename?id=1", "user1", "name=newname")
	rr := httptest.NewRecorder()
	HandleRenameInstance(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if msg, _ := resp["error"].(string); !strings.Contains(msg, "就绪") {
		t.Errorf("expected error about not ready, got %q", msg)
	}
}

// TestHandleRenameInstance_LocalInstance 本地 agent 实例 rename 跳过 CVM API，
// 直接改本地 DB。关键断言：不报「实例ID不合要求」，续改名成功。
func TestHandleRenameInstance_LocalInstance(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "local-user", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{
		UserID:     user.ID,
		InstanceId: "local-codebuddy-001", // 非空但不是 CVM 格式
		Source:     model.InstanceSourceLocal,
		Name:       "old-name",
	}
	model.DB(context.Background()).Create(inst)

	req := renameReqWithSession(t, http.MethodPost,
		fmt.Sprintf("/openclaw/rename?id=%d", inst.ID), "local-user", "name=new-name")
	rr := httptest.NewRecorder()
	HandleRenameInstance(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("本地实例 rename 应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "实例ID") || strings.Contains(rr.Body.String(), "不合要求") {
		t.Errorf("本地实例不应报 CVM ID 格式错，实际=%s", rr.Body.String())
	}

	// 验证 DB 中的 name 确实被更新
	var updated model.Instance
	model.DB(context.Background()).First(&updated, inst.ID)
	if updated.Name != "new-name" {
		t.Errorf("应更新为 new-name，实际=%s", updated.Name)
	}
}

// ─── getInstanceByID Tests ─────────────────────────────────────────────

func TestGetInstanceByID_ById(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "user1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{UserID: user.ID, InstanceId: "ins-aaa", Name: "test"}
	model.DB(context.Background()).Create(inst)

	req := httptest.NewRequest(http.MethodGet, "/test?id=1", nil)
	w := http.ResponseWriter(httptest.NewRecorder())
	got, err := getInstanceByID(&w, req, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.InstanceId != "ins-aaa" {
		t.Errorf("expected ins-aaa, got %s", got.InstanceId)
	}
}

func TestGetInstanceByID_ByInstanceId(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "user1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{UserID: user.ID, InstanceId: "ins-bbb", Name: "test2"}
	model.DB(context.Background()).Create(inst)

	req := httptest.NewRequest(http.MethodGet, "/test?instance_id=ins-bbb", nil)
	w := http.ResponseWriter(httptest.NewRecorder())
	got, err := getInstanceByID(&w, req, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "test2" {
		t.Errorf("expected test2, got %s", got.Name)
	}
}

func TestGetInstanceByID_InstanceIdInBody(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "user1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{UserID: user.ID, InstanceId: "ins-ccc", Name: "test3"}
	model.DB(context.Background()).Create(inst)

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("instance_id=ins-ccc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := http.ResponseWriter(httptest.NewRecorder())
	got, err := getInstanceByID(&w, req, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "test3" {
		t.Errorf("expected test3, got %s", got.Name)
	}
}

func TestGetInstanceByID_NoParams(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "user1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := http.ResponseWriter(httptest.NewRecorder())
	_, err := getInstanceByID(&w, req, user)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(hcommon.ErrorMessageWithCtx(context.Background(), err), "id") {
		t.Errorf("expected error about missing params, got %q", err.Error())
	}
}

func TestGetInstanceByID_InstanceIdNotFound(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "user1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)

	req := httptest.NewRequest(http.MethodGet, "/test?instance_id=ins-nonexist", nil)
	w := http.ResponseWriter(httptest.NewRecorder())
	_, err := getInstanceByID(&w, req, user)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "不存在") {
		t.Errorf("expected '不存在' error, got %q", err.Error())
	}
}

func TestGetInstanceByID_OtherUserInstance(t *testing.T) {
	initRenameTestDB(t)

	user1 := &model.User{Username: "user1", Password: "x", Role: "user"}
	user2 := &model.User{Username: "user2", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user1)
	model.DB(context.Background()).Create(user2)

	// 实例属于 user2
	inst := &model.Instance{UserID: user2.ID, InstanceId: "ins-ddd", Name: "other"}
	model.DB(context.Background()).Create(inst)

	// user1 尝试通过 instance_id 访问
	req := httptest.NewRequest(http.MethodGet, "/test?instance_id=ins-ddd", nil)
	w := http.ResponseWriter(httptest.NewRecorder())
	_, err := getInstanceByID(&w, req, user1)
	if err == nil {
		t.Fatal("expected error for other user's instance, got nil")
	}
}

func TestGetInstanceByID_IdTakesPriority(t *testing.T) {
	initRenameTestDB(t)

	user := &model.User{Username: "user1", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(user)
	inst := &model.Instance{UserID: user.ID, InstanceId: "ins-eee", Name: "priority"}
	model.DB(context.Background()).Create(inst)

	// 同时传 id 和 instance_id，id 应优先
	req := httptest.NewRequest(http.MethodGet, "/test?id=1&instance_id=ins-nonexist", nil)
	w := http.ResponseWriter(httptest.NewRecorder())
	got, err := getInstanceByID(&w, req, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.InstanceId != "ins-eee" {
		t.Errorf("expected ins-eee (id takes priority), got %s", got.InstanceId)
	}
}
