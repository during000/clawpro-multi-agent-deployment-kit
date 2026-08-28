package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hatchery/model"

	"github.com/gorilla/sessions"
)

// TestHandleGetMyUserGroups_NewFields 验证 /user-groups/mine 响应里 groups[] 子项
// 新增的四个字段：full_path / source / is_main / created_at。
//
// 场景：alice 同时属于
//   - manual 组：研发中心/后端组（source=manual, is_main 恒 false）
//   - oneid_dept 组：技术部/核心组（source=oneid_dept, is_main=true，alice 的主部门）
//   - oneid_dept 组：技术部（source=oneid_dept, is_main=false，alice 非主部门）
func TestHandleGetMyUserGroups_NewFields(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	// 切到带 gorilla session 的 cookie store，使 RequestUser 能解析出用户名
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	alice := model.User{Username: "alice", Password: "x", Role: "user"}
	if err := model.DB(context.Background()).Create(&alice).Error; err != nil {
		t.Fatalf("create alice: %v", err)
	}

	// manual 组
	manual := model.UserGroup{
		Name: "后端组", FullPath: "研发中心/后端组", Depth: 1,
		Source: model.GroupSourceManual,
	}
	if err := model.DB(context.Background()).Create(&manual).Error; err != nil {
		t.Fatalf("create manual: %v", err)
	}

	// oneid_dept 组：技术部（父）
	deptParent := model.UserGroup{
		Name: "技术部", FullPath: "技术部", Depth: 0,
		Source: model.GroupSourceOneIDDept, SourceRef: "D100",
	}
	if err := model.DB(context.Background()).Create(&deptParent).Error; err != nil {
		t.Fatalf("create dept parent: %v", err)
	}
	deptChild := model.UserGroup{
		Name: "核心组", FullPath: "技术部/核心组", ParentID: deptParent.ID, Depth: 1,
		Source: model.GroupSourceOneIDDept, SourceRef: "D101",
	}
	if err := model.DB(context.Background()).Create(&deptChild).Error; err != nil {
		t.Fatalf("create dept child: %v", err)
	}

	// 成员关系：alice 在 manual + 技术部(非主) + 核心组(主)
	for _, m := range []model.UserGroupMember{
		{UserGroupID: manual.ID, UserID: alice.ID, Source: model.MemberSourceManual},
		{UserGroupID: deptParent.ID, UserID: alice.ID, Source: model.MemberSourceOneIDDept, IsMain: false},
		{UserGroupID: deptChild.ID, UserID: alice.ID, Source: model.MemberSourceOneIDDept, IsMain: true},
	} {
		if err := model.DB(context.Background()).Create(&m).Error; err != nil {
			t.Fatalf("create member %+v: %v", m, err)
		}
	}

	// 构造带 session 的请求
	req := httptest.NewRequest(http.MethodGet, "/user-groups/mine", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "alice"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}

	w := httptest.NewRecorder()
	HandleGetMyUserGroups(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		OK     bool                     `json:"ok"`
		Groups []map[string]interface{} `json:"groups"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	if len(resp.Groups) != 3 {
		t.Fatalf("期望 3 个组，实际 %d: %+v", len(resp.Groups), resp.Groups)
	}

	byID := map[uint]map[string]interface{}{}
	for _, g := range resp.Groups {
		id := uint(g["id"].(float64))
		byID[id] = g
		// 四个新字段都必须存在
		for _, key := range []string{"full_path", "source", "is_main", "created_at"} {
			if _, ok := g[key]; !ok {
				t.Errorf("组 id=%d 缺字段 %q: %+v", id, key, g)
			}
		}
	}

	// manual 组
	m := byID[manual.ID]
	if got := m["full_path"]; got != "研发中心/后端组" {
		t.Errorf("manual full_path 错: %v", got)
	}
	if got := m["source"]; got != "manual" {
		t.Errorf("manual source 错: %v", got)
	}
	if got := m["is_main"]; got != false {
		t.Errorf("manual is_main 应为 false（manual 无主分组概念），实际 %v", got)
	}
	if s, _ := m["created_at"].(string); s == "" {
		t.Errorf("manual created_at 应非空")
	} else if _, err := time.Parse("2006-01-02T15:04:05Z", s); err != nil {
		t.Errorf("manual created_at 格式错: %q", s)
	}

	// oneid_dept 父：非主
	p := byID[deptParent.ID]
	if got := p["source"]; got != "oneid_dept" {
		t.Errorf("deptParent source 错: %v", got)
	}
	if got := p["is_main"]; got != false {
		t.Errorf("deptParent is_main 应为 false，实际 %v", got)
	}
	if got := p["full_path"]; got != "技术部" {
		t.Errorf("deptParent full_path 错: %v", got)
	}

	// oneid_dept 子：主
	c := byID[deptChild.ID]
	if got := c["source"]; got != "oneid_dept" {
		t.Errorf("deptChild source 错: %v", got)
	}
	if got := c["is_main"]; got != true {
		t.Errorf("deptChild is_main 应为 true（主部门），实际 %v", got)
	}
	if got := c["full_path"]; got != "技术部/核心组" {
		t.Errorf("deptChild full_path 错: %v", got)
	}
}

// TestHandleGetMyUserGroups_IsMainIsolation 确认 is_main 只看"当前用户"的成员行，
// 不会误把其他用户的 is_main=true 串到自己身上。
func TestHandleGetMyUserGroups_IsMainIsolation(t *testing.T) {
	setupUserGroupsUserTestDB(t)

	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))
	defer func() { Store = origStore }()

	alice := model.User{Username: "alice", Password: "x", Role: "user"}
	bob := model.User{Username: "bob", Password: "x", Role: "user"}
	model.DB(context.Background()).Create(&alice)
	model.DB(context.Background()).Create(&bob)

	dept := model.UserGroup{
		Name: "技术部", FullPath: "技术部", Depth: 0,
		Source: model.GroupSourceOneIDDept, SourceRef: "D100",
	}
	model.DB(context.Background()).Create(&dept)

	// bob 把该部门作为主部门，alice 非主
	model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: dept.ID, UserID: bob.ID,
		Source: model.MemberSourceOneIDDept, IsMain: true,
	})
	model.DB(context.Background()).Create(&model.UserGroupMember{
		UserGroupID: dept.ID, UserID: alice.ID,
		Source: model.MemberSourceOneIDDept, IsMain: false,
	})

	// alice 发请求
	req := httptest.NewRequest(http.MethodGet, "/user-groups/mine", nil)
	req.Header.Set("Accept", "application/json")
	session, _ := Store.Get(req, "hatchery-session")
	session.Values["username"] = "alice"
	rr := httptest.NewRecorder()
	session.Save(req, rr)
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	HandleGetMyUserGroups(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d", w.Code)
	}

	var resp struct {
		Groups []map[string]interface{} `json:"groups"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Groups) != 1 {
		t.Fatalf("期望 1 个组，实际 %d", len(resp.Groups))
	}
	if got := resp.Groups[0]["is_main"]; got != false {
		t.Fatalf("alice 非主部门时 is_main 必须 false（不应被 bob 的 is_main=true 串到），实际 %v", got)
	}
}
