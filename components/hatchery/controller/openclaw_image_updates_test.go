package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"hatchery/model"
)

func userSessionImageNoticeReq(t *testing.T, path string) *http.Request {
	t.Helper()
	return userSessionImageNoticeReqWithMethod(t, http.MethodGet, path)
}

func userSessionImageNoticeReqWithMethod(t *testing.T, method, path string) *http.Request {
	t.Helper()
	if err := model.DB(context.Background()).Create(&model.User{Username: "notice-user", Role: "user"}).Error; err != nil {
		t.Fatalf("创建普通用户失败: %v", err)
	}
	seedReq := httptest.NewRequest(method, path, nil)
	seedRR := httptest.NewRecorder()
	session := getSession(seedReq)
	session.Values["username"] = "notice-user"
	session.Values["role"] = "user"
	if err := session.Save(seedReq, seedRR); err != nil {
		t.Fatalf("保存 session 失败: %v", err)
	}
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Accept", "application/json")
	for _, cookie := range seedRR.Result().Cookies() {
		req.AddCookie(cookie)
	}
	return req
}

func TestUserImageUpdateNoticesMethodNotAllowed(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	req := userSessionImageNoticeReqWithMethod(t, http.MethodPost, "/openclaw/images/update-notices")
	rr := httptest.NewRecorder()
	HandleUserImageUpdateNotices(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("应 405，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestUserImageUpdateNoticesNoNotices(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	req := userSessionImageNoticeReq(t, "/openclaw/images/update-notices")
	rr := httptest.NewRecorder()
	HandleUserImageUpdateNotices(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 0 {
		t.Fatalf("items=%v，期望空列表", resp["items"])
	}
}

func TestUserImageUpdateNoticesOnlyReturnsEnabledImages(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	enabledImageID := "img-nmg7pw1r"
	images := []model.AIImage{
		{ImageId: testOfficialImageID, ImageName: "disabled openclaw", AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.7", Enabled: false, UpdateNoticeEnabled: true},
		{ImageId: enabledImageID, ImageName: "enabled openclaw", AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.28", Enabled: true, UpdateNoticeEnabled: true},
	}
	if err := model.DB(context.Background()).Create(&images).Error; err != nil {
		t.Fatalf("创建镜像失败: %v", err)
	}
	histories := []model.ImageHistory{
		{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.7", PublishedAt: mustParseTestTime(t, "2026-05-07")},
		{ImageId: enabledImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.28", PublishedAt: mustParseTestTime(t, "2026-05-28")},
	}
	if err := model.DBGlobal(context.Background()).Create(&histories).Error; err != nil {
		t.Fatalf("创建镜像历史失败: %v", err)
	}

	req := userSessionImageNoticeReq(t, "/openclaw/images/update-notices")
	rr := httptest.NewRecorder()
	HandleUserImageUpdateNotices(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v，期望只返回 1 条启用镜像通知", resp["items"])
	}
	item := items[0].(map[string]any)
	if item["image_id"] != enabledImageID || item["agent_version"] != "2026.5.28" {
		t.Fatalf("应只返回启用镜像通知，item=%v", item)
	}
}

func TestUserImageUpdateNoticesFiltersInvisibleAgentTypes(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	if err := model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, DefaultAgentType: model.AgentTypeHermes, DisabledAgentTypes: `["openclaw"]`}).Error; err != nil {
		t.Fatalf("创建站点配置失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.AIImage{ImageId: testOfficialImageID, ImageName: "openclaw", AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.28", Enabled: true, UpdateNoticeEnabled: true}).Error; err != nil {
		t.Fatalf("创建镜像失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.28", PublishedAt: mustParseTestTime(t, "2026-05-28")}).Error; err != nil {
		t.Fatalf("创建镜像历史失败: %v", err)
	}

	req := userSessionImageNoticeReq(t, "/openclaw/images/update-notices")
	rr := httptest.NewRecorder()
	HandleUserImageUpdateNotices(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 0 {
		t.Fatalf("items=%v，期望不可见 AgentType 被过滤", resp["items"])
	}
}
