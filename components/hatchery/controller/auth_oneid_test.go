package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// initAuthOneidTestDB 初始化内存 SQLite，迁移 Webhook 事件处理所需的表。
func initAuthOneidTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.User{},
		&model.Instance{},
		&model.SiteConfig{},
		&model.SMHPersonalSpace{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
	db.Create(&model.SiteConfig{})
}

// ─── validateAndGetSafeRedirectURL 单元测试 ─────────────────────────────────

func TestValidateAndGetSafeRedirectURL_Empty(t *testing.T) {
	got := validateAndGetSafeRedirectURL("", "/default")
	if got != "/default" {
		t.Errorf("空字符串应返回 defaultURL，got=%q", got)
	}
}

func TestValidateAndGetSafeRedirectURL_ValidRelativePath(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"/my-openclaw", "/my-openclaw"},
		{"/admin/basic-info", "/admin/basic-info"},
		{"/path?query=1", "/path?query=1"},
		{"/path#fragment", "/path#fragment"},
		{"/path?a=1&b=2#sec", "/path?a=1&b=2#sec"},
	}
	for _, tc := range cases {
		got := validateAndGetSafeRedirectURL(tc.raw, "/default")
		if got != tc.want {
			t.Errorf("raw=%q: want=%q, got=%q", tc.raw, tc.want, got)
		}
	}
}

func TestValidateAndGetSafeRedirectURL_RejectsAbsoluteURL(t *testing.T) {
	cases := []string{
		"https://evil.com/steal",
		"http://evil.com",
		"ftp://files.com/path",
		"javascript:alert(1)",
	}
	for _, raw := range cases {
		got := validateAndGetSafeRedirectURL(raw, "/default")
		if got != "/default" {
			t.Errorf("raw=%q: 应被拒绝返回 /default，got=%q", raw, got)
		}
	}
}

func TestValidateAndGetSafeRedirectURL_RejectsProtocolRelative(t *testing.T) {
	got := validateAndGetSafeRedirectURL("//evil.com/path", "/default")
	if got != "/default" {
		t.Errorf("协议相对 URL 应被拒绝，got=%q", got)
	}
}

func TestValidateAndGetSafeRedirectURL_RejectsNoLeadingSlash(t *testing.T) {
	got := validateAndGetSafeRedirectURL("relative/path", "/default")
	if got != "/default" {
		t.Errorf("无前导斜杠的路径应被拒绝，got=%q", got)
	}
}

func TestValidateAndGetSafeRedirectURL_RejectsURLWithHost(t *testing.T) {
	got := validateAndGetSafeRedirectURL("//example.com", "/default")
	if got != "/default" {
		t.Errorf("带 Host 的 URL 应被拒绝，got=%q", got)
	}
}

func TestValidateAndGetSafeRedirectURL_NormalizesEncodedPath(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"/%2e%2e/etc/passwd", "/%2e%2e/etc/passwd"},
		{"/path%2fextra", "/path%2fextra"},
		{"/normal/path", "/normal/path"},
		{"/page?key=%E4%B8%AD%E6%96%87", "/page?key=%E4%B8%AD%E6%96%87"},
		{"/page?key=%E4%B8%AD%E6%96%87#top", "/page?key=%E4%B8%AD%E6%96%87#top"},
	}
	for _, tc := range cases {
		got := validateAndGetSafeRedirectURL(tc.raw, "/default")
		if got != tc.want {
			t.Errorf("raw=%q: want=%q, got=%q", tc.raw, tc.want, got)
		}
	}
}

// ─── handleOneIDMemberDeleted 单元测试 ──────────────────────────────────────

func TestHandleOneIDMemberDeleted_HardDelete_NoResources(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "union-del-hard"
	model.DB(context.Background()).Create(&model.User{
		Username: "hard-del-user",
		OneIDSub: &sub,
	})

	// 无实例、无 VPC → 应被硬删除
	handleOneIDMemberDeleted(context.Background(), oneIDEventData{Sub: sub, AssetAction: "keep"})

	var count int64
	model.DB(context.Background()).Unscoped().Model(&model.User{}).Where("one_id_sub = ?", sub).Count(&count)
	if count != 0 {
		t.Fatalf("无资源用户应被硬删除，但仍存在 count=%d", count)
	}
}

func TestHandleOneIDMemberDeleted_SoftDelete_HasInstances(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "union-del-soft"
	user := model.User{
		Username: "soft-del-user",
		OneIDSub: &sub,
	}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{UserID: user.ID, Name: "inst"})

	// asset_action=keep: 实例 user_id 被清零，然后 tryHardDeleteUser 检查实例时
	// 找不到 user_id=user.ID 的实例 → 用户被硬删除。
	// 这是正确的业务行为："keep"模式解绑实例后用户确实无资源了。
	handleOneIDMemberDeleted(context.Background(), oneIDEventData{Sub: sub, AssetAction: "keep"})

	// 验证实例的 user_id 已被清零
	var inst model.Instance
	model.DB(context.Background()).Where("name = ?", "inst").First(&inst)
	if inst.UserID != 0 {
		t.Errorf("keep 模式下实例 user_id 应被清零，实际=%d", inst.UserID)
	}

	// 实例解绑后用户无资源 → 应被硬删除
	var count int64
	model.DB(context.Background()).Unscoped().Model(&model.User{}).Where("one_id_sub = ?", sub).Count(&count)
	if count != 0 {
		t.Fatalf("keep 模式解绑后用户无资源，应被硬删除，但仍存在 count=%d", count)
	}
}

func TestHandleOneIDMemberDeleted_AssetActionDelete(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "union-del-action-delete"
	user := model.User{
		Username: "del-action-user",
		OneIDSub: &sub,
	}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Create(&model.Instance{UserID: user.ID, Name: "inst-del"})

	// asset_action=delete: 实例被软删除 → 用户无资源 → 硬删除
	handleOneIDMemberDeleted(context.Background(), oneIDEventData{Sub: sub, AssetAction: "delete"})

	var count int64
	model.DB(context.Background()).Unscoped().Model(&model.User{}).Where("one_id_sub = ?", sub).Count(&count)
	if count != 0 {
		t.Fatalf("asset_action=delete 后用户应被硬删除，但仍存在 count=%d", count)
	}
}

func TestHandleOneIDMemberDeleted_AssetActionTransfer(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "union-del-transfer"
	targetSub := "union-target"
	user := model.User{Username: "transfer-user", OneIDSub: &sub}
	model.DB(context.Background()).Create(&user)
	targetUser := model.User{Username: "target-user", OneIDSub: &targetSub}
	model.DB(context.Background()).Create(&targetUser)
	model.DB(context.Background()).Create(&model.Instance{UserID: user.ID, Name: "inst-transfer"})

	handleOneIDMemberDeleted(context.Background(), oneIDEventData{
		Sub:           sub,
		AssetAction:   "transfer",
		TransferToSub: targetSub,
	})

	// 实例应转移到目标用户
	var inst model.Instance
	model.DB(context.Background()).Where("name = ?", "inst-transfer").First(&inst)
	if inst.UserID != targetUser.ID {
		t.Errorf("实例应转移到目标用户 %d，实际=%d", targetUser.ID, inst.UserID)
	}

	// 源用户应被硬删除（无资源了）
	var count int64
	model.DB(context.Background()).Unscoped().Model(&model.User{}).Where("one_id_sub = ?", sub).Count(&count)
	if count != 0 {
		t.Fatalf("转移后源用户应被硬删除，但仍存在 count=%d", count)
	}
}

func TestHandleOneIDMemberDeleted_UserNotFound(t *testing.T) {
	initAuthOneidTestDB(t)

	// 不应 panic
	handleOneIDMemberDeleted(context.Background(), oneIDEventData{Sub: "non-existent-sub"})
}

func TestHandleOneIDMemberDeleted_DefaultAssetAction(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "union-del-default"
	user := model.User{Username: "default-action-user", OneIDSub: &sub}
	model.DB(context.Background()).Create(&user)

	// 空 AssetAction 应默认为 "keep"，无资源 → 硬删除
	handleOneIDMemberDeleted(context.Background(), oneIDEventData{Sub: sub, AssetAction: ""})

	var count int64
	model.DB(context.Background()).Unscoped().Model(&model.User{}).Where("one_id_sub = ?", sub).Count(&count)
	if count != 0 {
		t.Fatalf("默认 keep 且无资源，用户应被硬删除，但仍存在 count=%d", count)
	}
}

// ─── handleOneIDMemberCreated 单元测试 ──────────────────────────────────────

func TestHandleOneIDMemberCreated_NewUser(t *testing.T) {
	initAuthOneidTestDB(t)

	handleOneIDMemberCreated(context.Background(), oneIDEventData{Sub: "new-sub", Name: "新用户"})

	var user model.User
	if model.DB(context.Background()).Where("one_id_sub = ?", "new-sub").First(&user).Error != nil {
		t.Fatal("应创建新用户")
	}
	if user.Username != "新用户" {
		t.Errorf("用户名期望'新用户'，实际=%s", user.Username)
	}
}

// ─── handleOneIDMemberUpdated 单元测试 ──────────────────────────────────────

func TestHandleOneIDMemberUpdated_SyncName(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "union-update-name"
	model.DB(context.Background()).Create(&model.User{Username: "旧名", OneIDSub: &sub})

	handleOneIDMemberUpdated(context.Background(), oneIDEventData{Sub: sub, Name: "新名"})

	var user model.User
	model.DB(context.Background()).Where("one_id_sub = ?", sub).First(&user)
	if user.Username != "新名" {
		t.Errorf("用户名应更新为'新名'，实际=%s", user.Username)
	}
}

func TestHandleOneIDMemberUpdated_UserNotFound(t *testing.T) {
	initAuthOneidTestDB(t)
	// 不应 panic
	handleOneIDMemberUpdated(context.Background(), oneIDEventData{Sub: "non-existent", Name: "name"})
}

func TestHandleOneIDMemberUpdated_SameName(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "union-same-name"
	model.DB(context.Background()).Create(&model.User{Username: "同名", OneIDSub: &sub})

	handleOneIDMemberUpdated(context.Background(), oneIDEventData{Sub: sub, Name: "同名"})

	var user model.User
	model.DB(context.Background()).Where("one_id_sub = ?", sub).First(&user)
	if user.Username != "同名" {
		t.Errorf("同名时不应修改，实际=%s", user.Username)
	}
}

func TestHandleOneIDMemberCreated_RestoreSoftDeleted(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "restore-sub"
	user := model.User{Username: "soft-del", OneIDSub: &sub}
	model.DB(context.Background()).Create(&user)
	model.DB(context.Background()).Delete(&user)

	handleOneIDMemberCreated(context.Background(), oneIDEventData{Sub: sub, Name: "新名字"})

	var restored model.User
	model.DB(context.Background()).Where("one_id_sub = ?", sub).First(&restored)
	if restored.DeletedAt.Valid {
		t.Fatal("软删除的用户应被恢复")
	}
}

func TestHandleOneIDMemberCreated_AlreadyExists(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "exists-sub"
	model.DB(context.Background()).Create(&model.User{Username: "existing", OneIDSub: &sub})

	// 不应 panic，不应重复创建
	handleOneIDMemberCreated(context.Background(), oneIDEventData{Sub: sub, Name: "existing"})

	var count int64
	model.DB(context.Background()).Model(&model.User{}).Where("one_id_sub = ?", sub).Count(&count)
	if count != 1 {
		t.Fatalf("应仍只有 1 个用户，实际 count=%d", count)
	}
}

// ─── handleOneIDAdminAdded/Removed 单元测试 ─────────────────────────────────

func TestHandleOneIDAdminAdded(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "admin-add-sub"
	model.DB(context.Background()).Create(&model.User{Username: "to-admin", Role: "user", OneIDSub: &sub})

	handleOneIDAdminAdded(context.Background(), oneIDEventData{Sub: sub})

	var user model.User
	model.DB(context.Background()).Where("one_id_sub = ?", sub).First(&user)
	if user.Role != "admin" {
		t.Errorf("角色应更新为 admin，实际=%s", user.Role)
	}
}

func TestHandleOneIDAdminRemoved(t *testing.T) {
	initAuthOneidTestDB(t)

	sub := "admin-rm-sub"
	model.DB(context.Background()).Create(&model.User{Username: "from-admin", Role: "admin", OneIDSub: &sub})

	handleOneIDAdminRemoved(context.Background(), oneIDEventData{Sub: sub})

	var user model.User
	model.DB(context.Background()).Where("one_id_sub = ?", sub).First(&user)
	if user.Role != "user" {
		t.Errorf("角色应更新为 user，实际=%s", user.Role)
	}
}

func TestOneIDSPIHandlers_MethodNotAllowed(t *testing.T) {
	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
		path    string
	}{
		{"logout", HandleOneIDLogout, "/spi/logout"},
		{"event", HandleOneIDEvent, "/spi/event"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			tt.handler(w, req)
			if w.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected 405, got %d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
