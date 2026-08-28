package controller

import (
	"context"
	"encoding/json"
	"hatchery/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── HandleOpenClawConfigOverview 测试 ──────────────────────────────────────
// 注意：requireLogin 需要 session 或合法 API Token，
// 这里仅测试方法校验和未授权分支。

func TestCoverageOpenClawConfigOverview_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/openclaw/config-overview", nil)
	req.Header.Set("Accept", "application/json")
	HandleOpenClawConfigOverview(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}

func TestCoverageOpenClawConfigOverview_Unauthorized(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview?group_ids=1", nil)
	req.Header.Set("Accept", "application/json")
	// 不设置 Authorization，session 也为空 → 未登录
	HandleOpenClawConfigOverview(w, req)

	// 未授权应返回非 200（可能 401 或 HX-Redirect）
	if w.Code == http.StatusOK {
		t.Error("期望非 200（未登录）")
	}
}

func TestCoverageOpenClawConfigOverview_InvalidToken(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview?group_ids=1", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer invalid-token-xyz")
	HandleOpenClawConfigOverview(w, req)

	// 未授权（无匹配用户）
	if w.Code == http.StatusOK {
		t.Error("期望非 200（无效 token）")
	}
}

// ── 以 session 方式测试（通过 cookie 设置 username） ──────────────────────

func TestCoverageOpenClawConfigOverview_ByGroupIDs_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	// 创建测试用户
	user := model.User{Username: "overview_user", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	req := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?group_ids="+itoa(group.ID), nil)
	req.Header.Set("Accept", "application/json")

	// 通过 session 设置登录状态
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "overview_user"

	w := httptest.NewRecorder()
	session.Save(req, w)

	// 从 recorder 取出 cookie 并重新设置到新 request
	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?group_ids="+itoa(group.ID), nil)
	req2.Header.Set("Accept", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	HandleOpenClawConfigOverview(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w2.Code, w2.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Error("期望 ok=true")
	}
}

func TestCoverageOpenClawConfigOverview_ByGroupIDs_WithZero(t *testing.T) {
	setupTreeTestDB(t)

	user := model.User{Username: "overview_user_zero", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview?group_ids=0", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "overview_user_zero"
	w := httptest.NewRecorder()
	session.Save(req, w)

	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview?group_ids=0", nil)
	req2.Header.Set("Accept", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	HandleOpenClawConfigOverview(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200（group_id=0 全局配置），实际=%d, body=%s", w2.Code, w2.Body.String())
	}
}

func TestCoverageOpenClawConfigOverview_ByGroupIDs_WithKeys(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	user := model.User{Username: "overview_user_keys", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	req := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?group_ids="+itoa(group.ID)+"&keys=model,channel", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "overview_user_keys"
	w := httptest.NewRecorder()
	session.Save(req, w)

	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?group_ids="+itoa(group.ID)+"&keys=model,channel", nil)
	req2.Header.Set("Accept", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	HandleOpenClawConfigOverview(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w2.Code, w2.Body.String())
	}
}

func TestCoverageOpenClawConfigOverview_MissingParams(t *testing.T) {
	setupTreeTestDB(t)

	user := model.User{Username: "overview_user_missing", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "overview_user_missing"
	w := httptest.NewRecorder()
	session.Save(req, w)

	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview", nil)
	req2.Header.Set("Accept", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	HandleOpenClawConfigOverview(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d, body=%s", w2.Code, w2.Body.String())
	}
}

func TestCoverageOpenClawConfigOverview_InvalidKey(t *testing.T) {
	setupTreeTestDB(t)

	user := model.User{Username: "overview_user_badkey", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview?group_ids=1&keys=bad_key", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "overview_user_badkey"
	w := httptest.NewRecorder()
	session.Save(req, w)

	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview?group_ids=1&keys=bad_key", nil)
	req2.Header.Set("Accept", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	HandleOpenClawConfigOverview(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w2.Code)
	}
}

func TestCoverageOpenClawConfigOverview_InvalidGroupIDsFormat(t *testing.T) {
	setupTreeTestDB(t)

	user := model.User{Username: "overview_user_badfmt", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	req := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview?group_ids=abc", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "overview_user_badfmt"
	w := httptest.NewRecorder()
	session.Save(req, w)

	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet, "/openclaw/config-overview?group_ids=abc", nil)
	req2.Header.Set("Accept", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	HandleOpenClawConfigOverview(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际=%d", w2.Code)
	}
}

func TestCoverageOpenClawConfigOverview_ByInstanceIDs(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	user := model.User{Username: "overview_user_inst", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	inst := model.Instance{Name: "agent-test", UserID: user.ID, GroupID: group.ID}
	model.DB(context.Background()).Create(&inst)

	req := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?ids="+itoa(inst.ID), nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "overview_user_inst"
	w := httptest.NewRecorder()
	session.Save(req, w)

	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?ids="+itoa(inst.ID), nil)
	req2.Header.Set("Accept", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	HandleOpenClawConfigOverview(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w2.Code, w2.Body.String())
	}
}

func TestCoverageOpenClawConfigOverview_ByInstanceIDs_NotOwned(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	user := model.User{Username: "overview_user_notown", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	otherUser := model.User{Username: "other_owner", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&otherUser)

	otherInst := model.Instance{Name: "other-inst", UserID: otherUser.ID, GroupID: 0}
	model.DB(context.Background()).Create(&otherInst)

	req := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?ids="+itoa(otherInst.ID), nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "overview_user_notown"
	w := httptest.NewRecorder()
	session.Save(req, w)

	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?ids="+itoa(otherInst.ID), nil)
	req2.Header.Set("Accept", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	HandleOpenClawConfigOverview(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Errorf("期望 400（非自己的实例），实际=%d", w2.Code)
	}
}

func TestCoverageOpenClawConfigOverview_Dedup(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	user := model.User{Username: "overview_user_dedup", Password: "hash", Role: "user"}
	model.DB(context.Background()).Create(&user)

	var group model.UserGroup
	model.DB(context.Background()).Where("name = ?", "研发组").First(&group)

	req := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?group_ids="+itoa(group.ID)+","+itoa(group.ID), nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "overview_user_dedup"
	w := httptest.NewRecorder()
	session.Save(req, w)

	cookies := w.Result().Cookies()
	req2 := httptest.NewRequest(http.MethodGet,
		"/openclaw/config-overview?group_ids="+itoa(group.ID)+","+itoa(group.ID), nil)
	req2.Header.Set("Accept", "application/json")
	for _, c := range cookies {
		req2.AddCookie(c)
	}

	w2 := httptest.NewRecorder()
	HandleOpenClawConfigOverview(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w2.Code, w2.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&resp)
	results := resp["results"].([]interface{})
	if len(results) != 1 {
		t.Errorf("期望去重后 1 个结果，实际=%d", len(results))
	}
}
