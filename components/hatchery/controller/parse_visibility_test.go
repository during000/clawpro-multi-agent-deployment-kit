package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// ── parseVisibilityParams 测试 ──────────────────────────────────────

func TestParseVisibilityParams_NoForm(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Form 为 nil
	vt, gids, projectIDs, has, err := parseVisibilityParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Error("无 Form 时 hasScope 应为 false")
	}
	if vt != "" || gids != nil || projectIDs != nil {
		t.Errorf("无 Form 时应返回空值，实际 vt=%s, gids=%v, projectIDs=%v", vt, gids, projectIDs)
	}
}

func TestParseVisibilityParams_NoVisibilityType(t *testing.T) {
	form := url.Values{"name": {"test"}}
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ParseForm()

	vt, gids, projectIDs, has, err := parseVisibilityParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Error("未传范围字段时 hasScope 应为 false")
	}
	if vt != "" || gids != nil || projectIDs != nil {
		t.Errorf("未传 visibility_type 时应返回空值")
	}
}

func TestParseVisibilityParams_AllType(t *testing.T) {
	setupParseVisibilityTestDB(t)

	form := url.Values{
		"visibility_type": {"all"},
	}
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ParseForm()

	vt, gids, projectIDs, has, err := parseVisibilityParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Error("传了范围字段时 hasScope 应为 true")
	}
	if vt != "all" {
		t.Errorf("期望 vt=all，实际=%s", vt)
	}
	if len(gids) != 0 || len(projectIDs) != 0 {
		t.Errorf("all 类型不应有范围关联")
	}
}

func TestParseVisibilityParams_EmptyDefaultsToAll(t *testing.T) {
	setupParseVisibilityTestDB(t)

	form := url.Values{
		"visibility_type": {""},
		"group_ids":       {""},
		"project_ids":     {""},
	}
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ParseForm()

	vt, _, _, has, err := parseVisibilityParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Error("传了范围字段（visibility_type 为空）时 hasScope 应为 true")
	}
	if vt != "all" {
		t.Errorf("空 visibility_type 应默认为 all，实际=%s", vt)
	}
}

func TestParseVisibilityParams_GroupWithIDs(t *testing.T) {
	setupParseVisibilityTestDB(t)

	// 创建分组以通过 validateVisibilityInput 校验
	model.DB(context.Background()).Create(&model.UserGroup{Name: "g1"})
	model.DB(context.Background()).Create(&model.UserGroup{Name: "g2"})

	form := url.Values{
		"visibility_type": {"group"},
		"group_ids":       {"1,2"},
	}
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ParseForm()

	vt, gids, projectIDs, has, err := parseVisibilityParams(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Error("hasScope 应为 true")
	}
	if vt != "group" {
		t.Errorf("期望 vt=group，实际=%s", vt)
	}
	if len(gids) != 2 {
		t.Errorf("期望 2 个 groupIDs，实际=%d", len(gids))
	}
	if len(projectIDs) != 0 {
		t.Errorf("期望没有 projectIDs，实际=%v", projectIDs)
	}
}

func TestParseVisibilityParams_GroupWithProjectIDs(t *testing.T) {
	setupParseVisibilityTestDB(t)
	if err := model.DB(context.Background()).Create(&model.Project{Name: "p1"}).Error; err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}
	form := url.Values{
		"visibility_type": {"group"},
		"project_ids":     {"1"},
	}
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ParseForm()

	vt, groupIDs, projectIDs, has, err := parseVisibilityParams(req)
	if err != nil {
		t.Fatalf("仅 project_ids 应允许更新范围: %v", err)
	}
	if !has || vt != model.VisibilityGroup || len(groupIDs) != 0 || !slices.Equal(projectIDs, []uint{1}) {
		t.Fatalf("解析结果不符合项目范围语义: vt=%q groupIDs=%v projectIDs=%v has=%v", vt, groupIDs, projectIDs, has)
	}
}

func TestParseVisibilityParams_InvalidType(t *testing.T) {
	form := url.Values{
		"visibility_type": {"invalid"},
		"group_ids":       {""},
		"project_ids":     {""},
	}
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ParseForm()

	_, _, _, _, err := parseVisibilityParams(req)
	if err == nil {
		t.Error("无效的 visibility_type 应返回错误")
	}
}

func TestParseVisibilityParams_RejectsGroupWithoutScope(t *testing.T) {
	form := url.Values{
		"visibility_type": {"group"},
	}
	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ParseForm()

	_, _, _, _, err := parseVisibilityParams(req)
	if err == nil {
		t.Error("group 类型未传 group_ids 和 project_ids 时应拒绝")
	}
}

func setupParseVisibilityTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := db.AutoMigrate(
		&model.CustomAgentType{},
		&model.UserGroup{},
		&model.UserGroupMember{},
		&model.Project{},
		&model.SiteConfig{},
	); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	t.Cleanup(model.UseDBForTest(db))
}
