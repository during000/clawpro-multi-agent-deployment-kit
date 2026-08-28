package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	hcommon "hatchery/common"
	"hatchery/model"
)

const testOfficialImageID = "img-idzg74s9"

func postImageHistoryJSON(t *testing.T, path string, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("编码请求 JSON 失败: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(payload)))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func adminSessionImageHistoryReq(t *testing.T, method, path string) *http.Request {
	t.Helper()
	if err := model.DB(context.Background()).Create(&model.User{Username: "admin", Role: "admin"}).Error; err != nil {
		t.Fatalf("创建管理员用户失败: %v", err)
	}
	seedReq := httptest.NewRequest(method, path, nil)
	seedRR := httptest.NewRecorder()
	session := getSession(seedReq)
	session.Values["username"] = "admin"
	session.Values["role"] = "admin"
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

func decodeImageHistoryResponse(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是 JSON: %v body=%s", err, rr.Body.String())
	}
	return resp
}

func mustParseTestTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("解析时间失败: %v", err)
	}
	return parsed
}

func createTestAIImages(t *testing.T, version string, notice bool) {
	t.Helper()
	imgs := []model.AIImage{
		{Identifier: "tenant-a", ImageId: testOfficialImageID, ImageName: "old", AgentType: model.AgentTypeOpenClaw, AgentVersion: version, UpdateNoticeEnabled: notice},
		{Identifier: "tenant-b", ImageId: testOfficialImageID, ImageName: "old", AgentType: model.AgentTypeOpenClaw, AgentVersion: version, UpdateNoticeEnabled: notice},
	}
	if err := model.DBGlobal(context.Background()).Create(&imgs).Error; err != nil {
		t.Fatalf("创建测试镜像失败: %v", err)
	}
}

func assertAllImagesVersionAndNotice(t *testing.T, version string, notice bool) {
	t.Helper()
	var imgs []model.AIImage
	if err := model.DBGlobal(context.Background()).Where("image_id = ?", testOfficialImageID).Order("identifier asc").Find(&imgs).Error; err != nil {
		t.Fatalf("查询镜像失败: %v", err)
	}
	if len(imgs) != 2 {
		t.Fatalf("镜像数量=%d，期望 2", len(imgs))
	}
	for _, img := range imgs {
		if img.AgentVersion != version {
			t.Fatalf("tenant=%s version=%s，期望 %s", img.Identifier, img.AgentVersion, version)
		}
		if img.UpdateNoticeEnabled != notice {
			t.Fatalf("tenant=%s notice=%v，期望 %v", img.Identifier, img.UpdateNoticeEnabled, notice)
		}
	}
}

func TestImageHistoryPublishLatestChangedSyncsImagesAndClearsNotice(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	createTestAIImages(t, "2026.5.7", true)

	req := postImageHistoryJSON(t, "/admin/images/history/publish", map[string]any{
		"image_id":      testOfficialImageID,
		"agent_version": "2026.5.25",
		"published_at":  "2026-05-25",
	})
	rr := httptest.NewRecorder()
	HandlePublishImageUpdate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	if resp["latest_changed"] != true {
		t.Fatalf("latest_changed=%v，期望 true", resp["latest_changed"])
	}
	if resp["updated_images"].(float64) != 2 {
		t.Fatalf("updated_images=%v，期望 2", resp["updated_images"])
	}
	assertAllImagesVersionAndNotice(t, "2026.5.25", false)
}

func TestImageHistoryPublishOlderDoesNotClearNotice(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	createTestAIImages(t, "2026.5.25", true)
	if err := model.DBGlobal(context.Background()).Create(&model.ImageHistory{
		ImageId:      testOfficialImageID,
		AgentType:    model.AgentTypeOpenClaw,
		AgentVersion: "2026.5.25",
		PublishedAt:  mustParseTestTime(t, "2026-05-25"),
	}).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}

	req := postImageHistoryJSON(t, "/admin/images/history/publish", map[string]any{
		"image_id":      testOfficialImageID,
		"agent_version": "2026.5.1",
		"published_at":  "2026-05-01",
	})
	rr := httptest.NewRecorder()
	HandlePublishImageUpdate(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	if resp["latest_changed"] != false {
		t.Fatalf("latest_changed=%v，期望 false", resp["latest_changed"])
	}
	assertAllImagesVersionAndNotice(t, "2026.5.25", true)
}

func TestImageHistoryUpdateLatestChangeSyncsImages(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	createTestAIImages(t, "2026.5.25", true)
	older := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.1", PublishedAt: mustParseTestTime(t, "2026-05-01")}
	newer := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	if err := model.DBGlobal(context.Background()).Create(&older).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&newer).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}

	req := postImageHistoryJSON(t, "/admin/images/history/update", map[string]any{
		"id":            older.ID,
		"published_at":  "2026-06-01",
		"agent_version": "2026.6.1",
	})
	rr := httptest.NewRecorder()
	HandleUpdateImageHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	if resp["latest_changed"] != true {
		t.Fatalf("latest_changed=%v，期望 true", resp["latest_changed"])
	}
	assertAllImagesVersionAndNotice(t, "2026.6.1", false)
}

func TestImageHistoryDeleteLatestFallsBackToPrevious(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	createTestAIImages(t, "2026.5.25", true)
	older := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.1", PublishedAt: mustParseTestTime(t, "2026-05-01")}
	newer := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	if err := model.DBGlobal(context.Background()).Create(&older).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&newer).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}

	req := postImageHistoryJSON(t, "/admin/images/history/delete", map[string]any{
		"id": newer.ID,
	})
	rr := httptest.NewRecorder()
	HandleDeleteImageHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	if resp["latest_changed"] != true {
		t.Fatalf("latest_changed=%v，期望 true", resp["latest_changed"])
	}
	assertAllImagesVersionAndNotice(t, "2026.5.1", false)

	var count int64
	if err := model.DBGlobal(context.Background()).Model(&model.ImageHistory{}).Where("id = ?", newer.ID).Count(&count).Error; err != nil {
		t.Fatalf("查询历史失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("删除后默认查询应看不到记录，count=%d", count)
	}
}

func TestImageHistoryHardDeleteRemovesSoftDeletedRecord(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	createTestAIImages(t, "2026.5.25", true)
	older := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.1", PublishedAt: mustParseTestTime(t, "2026-05-01")}
	newer := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	if err := model.DBGlobal(context.Background()).Create(&older).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&newer).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Delete(&newer).Error; err != nil {
		t.Fatalf("软删除历史失败: %v", err)
	}
	assertAllImagesVersionAndNotice(t, "2026.5.25", true)

	req := postImageHistoryJSON(t, "/admin/images/history/delete", map[string]any{
		"id":   newer.ID,
		"hard": true,
	})
	rr := httptest.NewRecorder()
	HandleDeleteImageHistory(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	if resp["hard"] != true {
		t.Fatalf("hard=%v，期望 true", resp["hard"])
	}
	if resp["latest_changed"] != false {
		t.Fatalf("latest_changed=%v，期望 false", resp["latest_changed"])
	}
	assertAllImagesVersionAndNotice(t, "2026.5.25", true)

	var count int64
	if err := model.DBGlobal(context.Background()).Unscoped().Model(&model.ImageHistory{}).Where("id = ?", newer.ID).Count(&count).Error; err != nil {
		t.Fatalf("查询历史失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("hard 删除后不应存在记录，count=%d", count)
	}
}

func TestImageHistoryRestoreLatestSyncsImages(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	createTestAIImages(t, "2026.5.1", true)
	older := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.1", PublishedAt: mustParseTestTime(t, "2026-05-01")}
	newer := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	if err := model.DBGlobal(context.Background()).Create(&older).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&newer).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Delete(&newer).Error; err != nil {
		t.Fatalf("删除历史失败: %v", err)
	}

	req := postImageHistoryJSON(t, "/admin/images/history/restore", map[string]any{"id": newer.ID})
	rr := httptest.NewRecorder()
	HandleRestoreImageHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	if resp["latest_changed"] != true {
		t.Fatalf("latest_changed=%v，期望 true", resp["latest_changed"])
	}
	assertAllImagesVersionAndNotice(t, "2026.5.25", false)
}

func TestImageHistoryRestoreOlderDoesNotClearNotice(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	createTestAIImages(t, "2026.5.25", true)
	older := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.1", PublishedAt: mustParseTestTime(t, "2026-05-01")}
	newer := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	if err := model.DBGlobal(context.Background()).Create(&older).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&newer).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Delete(&older).Error; err != nil {
		t.Fatalf("删除历史失败: %v", err)
	}

	req := postImageHistoryJSON(t, "/admin/images/history/restore", map[string]any{"id": older.ID})
	rr := httptest.NewRecorder()
	HandleRestoreImageHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	if resp["latest_changed"] != false {
		t.Fatalf("latest_changed=%v，期望 false", resp["latest_changed"])
	}
	assertAllImagesVersionAndNotice(t, "2026.5.25", true)
}

func TestImageHistoryListTenantAdminFiltersUnavailableImages(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	visible := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	hidden := model.ImageHistory{ImageId: "img-pf18atu9", AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	if err := model.DBGlobal(context.Background()).Create(&visible).Error; err != nil {
		t.Fatalf("创建可见历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&hidden).Error; err != nil {
		t.Fatalf("创建不可见历史失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.AIImage{ImageId: testOfficialImageID, ImageName: "visible", AgentType: model.AgentTypeOpenClaw}).Error; err != nil {
		t.Fatalf("创建租户镜像失败: %v", err)
	}

	req := adminSessionImageHistoryReq(t, http.MethodGet, "/admin/images/history")
	rr := httptest.NewRecorder()
	HandleImageUpdateHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v，期望只返回 1 条可用镜像历史", resp["items"])
	}
	item := items[0].(map[string]any)
	if item["image_id"] != testOfficialImageID {
		t.Fatalf("应过滤不可用镜像历史，item=%v", item)
	}
}

func TestImageHistoryListEnabledOnlyFiltersDisabledImages(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	enabledImageID := testOfficialImageID
	disabledImageID := "img-pf18atu9"
	disabledAgentTypeImageID := "img-al484uhr"
	enabled := model.ImageHistory{ImageId: enabledImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	disabled := model.ImageHistory{ImageId: disabledImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	disabledAgentType := model.ImageHistory{ImageId: disabledAgentTypeImageID, AgentType: model.AgentTypeHermes, AgentVersion: "0.12.0", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	if err := model.DBGlobal(context.Background()).Create(&enabled).Error; err != nil {
		t.Fatalf("创建启用镜像历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&disabled).Error; err != nil {
		t.Fatalf("创建未启用镜像历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&disabledAgentType).Error; err != nil {
		t.Fatalf("创建禁用类型镜像历史失败: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, DefaultAgentType: model.AgentTypeOpenClaw, DisabledAgentTypes: `["hermes"]`}).Error; err != nil {
		t.Fatalf("创建站点配置失败: %v", err)
	}
	images := []model.AIImage{
		{ImageId: enabledImageID, ImageName: "enabled", AgentType: model.AgentTypeOpenClaw, Enabled: true},
		{ImageId: disabledImageID, ImageName: "disabled", AgentType: model.AgentTypeOpenClaw, Enabled: false},
		{ImageId: disabledAgentTypeImageID, ImageName: "disabled-agent-type", AgentType: model.AgentTypeHermes, Enabled: true},
	}
	for _, img := range images {
		if err := model.DB(context.Background()).Create(&img).Error; err != nil {
			t.Fatalf("创建租户镜像失败: %v", err)
		}
	}

	req := adminImagesReq(http.MethodGet, "/admin/images/history?enabled_only=true", "")
	rr := httptest.NewRecorder()
	HandleImageUpdateHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v，期望只返回 1 条启用镜像历史", resp["items"])
	}
	item := items[0].(map[string]any)
	if item["image_id"] != enabledImageID {
		t.Fatalf("应过滤未启用镜像历史，item=%v", item)
	}
	if item["image_enabled"] != true {
		t.Fatalf("启用镜像历史应返回 image_enabled=true，item=%v", item)
	}
}

func TestImageHistoryListCanSetNoticeOnlyLatestPublishedAt(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	older := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.1", PublishedAt: mustParseTestTime(t, "2026-05-01")}
	newer := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	if err := model.DBGlobal(context.Background()).Create(&older).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&newer).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}

	req := adminImagesReq(http.MethodGet, "/admin/images/history", "")
	rr := httptest.NewRecorder()
	HandleImageUpdateHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items=%v，期望 2 条", resp["items"])
	}
	latest := items[0].(map[string]any)
	if latest["id"].(float64) != float64(newer.ID) || latest["can_set_notice"] != true {
		t.Fatalf("最新历史应 can_set_notice=true，item=%v", latest)
	}
	previous := items[1].(map[string]any)
	if previous["id"].(float64) != float64(older.ID) || previous["can_set_notice"] != false {
		t.Fatalf("非最新历史应 can_set_notice=false，item=%v", previous)
	}
}

func TestImageHistoryListIncludeDeleted(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()
	history := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "2026.5.25", PublishedAt: mustParseTestTime(t, "2026-05-25")}
	if err := model.DBGlobal(context.Background()).Create(&history).Error; err != nil {
		t.Fatalf("创建历史失败: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Delete(&history).Error; err != nil {
		t.Fatalf("删除历史失败: %v", err)
	}

	req := adminImagesReq(http.MethodGet, "/admin/images/history?include_deleted=true", "")
	rr := httptest.NewRecorder()
	HandleImageUpdateHistory(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("应 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeImageHistoryResponse(t, rr)
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v，期望 1 条", resp["items"])
	}
	item := items[0].(map[string]any)
	if item["deleted_at"] == nil {
		t.Fatalf("include_deleted=true 应返回 deleted_at，item=%v", item)
	}
}

func TestImageHistoryHelperFunctions(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	ids := officialImageIDs()
	if len(ids) == 0 {
		t.Fatal("officialImageIDs should not be empty")
	}
	if model.NormalizeAgentType("") != model.AgentTypeOpenClaw {
		t.Fatal("empty agent type should normalize to openclaw")
	}
	if model.NormalizeAgentType(model.AgentTypeHermes) != model.AgentTypeHermes {
		t.Fatal("agent type should be preserved")
	}

	latest, err := model.LatestImageHistoriesByImageID(context.Background(), nil)
	if err != nil || len(latest) != 0 {
		t.Fatalf("empty latest history = %v err=%v", latest, err)
	}
	older := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "old", PublishedAt: mustParseTestTime(t, "2026-05-01")}
	newer := model.ImageHistory{ImageId: testOfficialImageID, AgentType: model.AgentTypeOpenClaw, AgentVersion: "new", PublishedAt: mustParseTestTime(t, "2026-05-02")}
	if err := model.DBGlobal(context.Background()).Create(&older).Error; err != nil {
		t.Fatalf("create older history: %v", err)
	}
	if err := model.DBGlobal(context.Background()).Create(&newer).Error; err != nil {
		t.Fatalf("create newer history: %v", err)
	}
	latest, err = model.LatestImageHistoriesByImageID(context.Background(), []string{testOfficialImageID})
	if err != nil {
		t.Fatalf("latest history: %v", err)
	}
	if latest[testOfficialImageID].AgentVersion != "new" {
		t.Fatalf("latest history = %+v", latest[testOfficialImageID])
	}

	visibleIDs, err := tenantOfficialImageIDs(context.Background())
	if err != nil || len(visibleIDs) != 0 {
		t.Fatalf("tenantOfficialImageIDs before images = %v err=%v", visibleIDs, err)
	}
	if err := model.DB(context.Background()).Create(&model.AIImage{ImageId: testOfficialImageID, AgentType: "", Enabled: true}).Error; err != nil {
		t.Fatalf("create enabled openclaw image: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.AIImage{ImageId: "img-al484uhr", AgentType: model.AgentTypeHermes, Enabled: true}).Error; err != nil {
		t.Fatalf("create enabled hermes image: %v", err)
	}
	if err := model.DB(context.Background()).Create(&model.SiteConfig{ID: 1, DefaultAgentType: model.AgentTypeOpenClaw, DisabledAgentTypes: `["hermes"]`}).Error; err != nil {
		t.Fatalf("create site config: %v", err)
	}
	visibleIDs, err = tenantOfficialImageIDs(context.Background())
	if err != nil || len(visibleIDs) != 2 {
		t.Fatalf("tenantOfficialImageIDs after images = %v err=%v", visibleIDs, err)
	}
	enabledSet, err := enabledOfficialImageIDSet(context.Background(), []string{testOfficialImageID, "img-al484uhr"})
	if err != nil {
		t.Fatalf("enabledOfficialImageIDSet: %v", err)
	}
	if _, ok := enabledSet[testOfficialImageID]; !ok {
		t.Fatalf("enabled set missing openclaw image: %v", enabledSet)
	}
	if _, ok := enabledSet["img-al484uhr"]; ok {
		t.Fatalf("enabled set should exclude disabled hermes type: %v", enabledSet)
	}
	emptySet, err := enabledOfficialImageIDSet(context.Background(), nil)
	if err != nil || len(emptySet) != 0 {
		t.Fatalf("empty enabled set = %v err=%v", emptySet, err)
	}
}

func TestImageHistoryHelperErrorBranches(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	origCandidates := hcommon.CandidateImages
	hcommon.CandidateImages = nil
	if ids, err := tenantOfficialImageIDs(context.Background()); err != nil || ids != nil {
		t.Fatalf("tenantOfficialImageIDs without candidates = %v err=%v", ids, err)
	}
	hcommon.CandidateImages = origCandidates

	if err := model.DB(context.Background()).Migrator().DropTable(&model.ImageHistory{}); err != nil {
		t.Fatalf("drop image_history: %v", err)
	}
	if _, err := model.LatestImageHistoriesByImageID(context.Background(), []string{testOfficialImageID}); err == nil {
		t.Fatal("expected latest history query error")
	}

	if err := model.DB(context.Background()).Migrator().DropTable(&model.AIImage{}); err != nil {
		t.Fatalf("drop ai_images: %v", err)
	}
	if _, err := tenantOfficialImageIDs(context.Background()); err == nil {
		t.Fatal("expected tenant image query error")
	}
	if _, err := enabledOfficialImageIDSet(context.Background(), []string{testOfficialImageID}); err == nil {
		t.Fatal("expected enabled image query error")
	}
}
