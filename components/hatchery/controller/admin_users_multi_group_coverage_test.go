package controller

import (
	"context"
	"encoding/json"
	"hatchery/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ── HandleAdminMultiGroupStats 测试 ────────────────────────────────────────

func TestCoverageMultiGroupStats_Success(t *testing.T) {
	setupTreeTestDB(t)
	seedTreeData(t)

	// 把 user1 加入两个组以形成多归属
	var user model.User
	model.DB(context.Background()).Where("username = ?", "user1").First(&user)
	var grandchild model.UserGroup
	model.DB(context.Background()).Where("name = ?", "前端组").First(&grandchild)
	model.DB(context.Background()).Create(&model.UserGroupMember{UserGroupID: grandchild.ID, UserID: user.ID, Source: "manual"})

	w := httptest.NewRecorder()
	HandleAdminMultiGroupStats(w, adminTreeReq(http.MethodGet, "/admin/users/multi-group-stats", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["ok"] != true {
		t.Error("期望 ok=true")
	}
	if resp["multi_group_users"].(float64) < 1 {
		t.Error("期望至少 1 个多归属用户")
	}
}

func TestCoverageMultiGroupStats_EmptyDB(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminMultiGroupStats(w, adminTreeReq(http.MethodGet, "/admin/users/multi-group-stats", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实际=%d, body=%s", w.Code, w.Body.String())
	}
}

func TestCoverageMultiGroupStats_MethodNotAllowed(t *testing.T) {
	setupTreeTestDB(t)

	w := httptest.NewRecorder()
	HandleAdminMultiGroupStats(w, adminTreeReq(http.MethodPost, "/admin/users/multi-group-stats", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际=%d", w.Code)
	}
}
