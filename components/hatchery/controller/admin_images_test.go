package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initImagesTestDB 初始化内存 SQLite 数据库用于 images 测试
func initImagesTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存数据库失败: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.UserGroup{}, &model.UserGroupMember{}, &model.GroupConfigBinding{}, &model.AIImage{}, &model.ImageHistory{}, &model.SiteConfig{}, &model.Instance{}); err != nil {
		t.Fatalf("数据库迁移失败: %v", err)
	}
	// 如果使用 :memory:，则设置最大连接数为 1
	// 并发连接可能会导致 no such table 错误
	sqlDB, _ := db.DB()
	// 保证单个连接不会因为生命周期或者空闲而断开
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetMaxOpenConns(1)
	origDB := model.UseDBForTestWithDriver(db, "sqlite")
	// 设置 AdminToken 用于测试
	AdminToken = "test-admin-token"

	// 初始化 session store
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

// adminImagesReq 构造携带管理员 Token 的 HTTP 请求
func adminImagesReq(method, path string, body string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// ==================== HandleSetDefaultAgentType Tests ====================

func TestHandleSetDefaultAgentType_MethodNotAllowed(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	req := adminImagesReq(http.MethodGet, "/api/admin/images/set-default-agent-type", "")
	rr := httptest.NewRecorder()

	HandleSetDefaultAgentType(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleSetDefaultAgentType_EmptyAgentType(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	req := adminImagesReq(http.MethodPost, "/api/admin/images/set-default-agent-type", "agent_type=")
	rr := httptest.NewRecorder()

	HandleSetDefaultAgentType(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleSetDefaultAgentType_InvalidAgentType(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("agent_type", "invalid_type")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/set-default-agent-type", form.Encode())
	rr := httptest.NewRecorder()

	HandleSetDefaultAgentType(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleSetDefaultAgentType_NoEnabledImage(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 创建 site config
	config := model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}
	model.DB(context.Background()).Create(&config)

	// 创建一个禁用的镜像
	img := model.AIImage{
		ImageId:   "img-001",
		ImageName: "Hermes Image",
		AgentType: "hermes",
		Enabled:   false,
	}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("agent_type", "hermes")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/set-default-agent-type", form.Encode())
	rr := httptest.NewRecorder()

	HandleSetDefaultAgentType(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if errMsg, ok := resp["error"].(string); !ok || !strings.Contains(errMsg, "没有已启用镜像") {
		t.Errorf("expected error about no enabled image, got %v", resp)
	}
}

func TestHandleSetDefaultAgentType_Success(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 创建 site config
	config := model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}
	model.DB(context.Background()).Create(&config)

	// 创建启用的 hermes 镜像
	img := model.AIImage{
		ImageId:      "img-001",
		ImageName:    "Hermes Image",
		AgentType:    "hermes",
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("agent_type", "hermes")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/set-default-agent-type", form.Encode())
	rr := httptest.NewRecorder()

	HandleSetDefaultAgentType(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp)
	}

	// 验证数据库中的值已更新
	var updated model.SiteConfig
	model.DB(context.Background()).First(&updated)
	if updated.DefaultAgentType != "hermes" {
		t.Errorf("expected default_agent_type=hermes, got %s", updated.DefaultAgentType)
	}
}

// ==================== matchPublicImageType Tests ====================

func TestMatchPublicImageType_Hermes(t *testing.T) {
	tests := []struct {
		osName      string
		wantType    string
		wantVersion string
		wantMatched bool
	}{
		{"HermesAgent-1.0.0", model.AgentTypeHermes, "0.9.0", true},
		{"hermesagent", model.AgentTypeHermes, "0.9.0", true},
		{"HERMES-Server", model.AgentTypeHermes, "0.9.0", true},
		{"ubuntu-20.04", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			agentType, agentVersion, matched := matchPublicImageType(tt.osName)
			if matched != tt.wantMatched {
				t.Errorf("matchPublicImageType(%q) matched = %v, want %v", tt.osName, matched, tt.wantMatched)
			}
			if agentType != tt.wantType {
				t.Errorf("matchPublicImageType(%q) agentType = %q, want %q", tt.osName, agentType, tt.wantType)
			}
			if agentVersion != tt.wantVersion {
				t.Errorf("matchPublicImageType(%q) agentVersion = %q, want %q", tt.osName, agentVersion, tt.wantVersion)
			}
		})
	}
}

func TestMatchPublicImageType_LightclawACE(t *testing.T) {
	tests := []struct {
		osName      string
		wantType    string
		wantVersion string
		wantMatched bool
	}{
		{"LightClawACE-0.1.0", model.AgentTypeLightclawACE, "0.1.1", true},
		{"lightclawace", model.AgentTypeLightclawACE, "0.1.1", true},
		{"lightclaw_ace_v1", model.AgentTypeLightclawACE, "0.1.1", true},
		{"openclaw-2026.1.1", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.osName, func(t *testing.T) {
			agentType, agentVersion, matched := matchPublicImageType(tt.osName)
			if matched != tt.wantMatched {
				t.Errorf("matchPublicImageType(%q) matched = %v, want %v", tt.osName, matched, tt.wantMatched)
			}
			if agentType != tt.wantType {
				t.Errorf("matchPublicImageType(%q) agentType = %q, want %q", tt.osName, agentType, tt.wantType)
			}
			if agentVersion != tt.wantVersion {
				t.Errorf("matchPublicImageType(%q) agentVersion = %q, want %q", tt.osName, agentVersion, tt.wantVersion)
			}
		})
	}
}

func TestMatchPublicImageType_NoMatch(t *testing.T) {
	tests := []string{
		"ubuntu-20.04",
		"centos-7",
		"windows-server-2019",
		"openclaw-2026.1.1",
		"random-image",
	}

	for _, osName := range tests {
		t.Run(osName, func(t *testing.T) {
			agentType, agentVersion, matched := matchPublicImageType(osName)
			if matched {
				t.Errorf("matchPublicImageType(%q) should not match, got type=%q, version=%q", osName, agentType, agentVersion)
			}
		})
	}
}

// ==================== HandleUpdateImage Tests ====================

func TestHandleUpdateImage_MethodNotAllowed(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	req := adminImagesReq(http.MethodGet, "/api/admin/images/update", "")
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleUpdateImage_MissingId(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("agent_type", "hermes")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleUpdateImage_ImageNotFound(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("id", "999")
	form.Set("agent_type", "hermes")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleUpdateImage_InvalidAgentType(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{ImageId: "img-001", ImageName: "Test Image", AgentType: "openclaw", AgentVersion: "2026.1.1"}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_type", "invalid_type")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleUpdateImage_InvalidAgentVersion(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{ImageId: "img-001", ImageName: "Test Image", AgentType: "openclaw", AgentVersion: "2026.1.1"}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_version", "!!!invalid!!!")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleUpdateImage_NoUpdates(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{ImageId: "img-001", ImageName: "Test Image", AgentType: "openclaw", AgentVersion: "2026.1.1"}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandleUpdateImage_Success_AgentType(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 初始为 hermes+semver，改类型到 lightclawace（版本沿用 0.9.0 仍是合法 semver）
	img := model.AIImage{ImageId: "img-001", ImageName: "Test Image", AgentType: "hermes", AgentVersion: "0.9.0"}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_type", "lightclawace")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Errorf("expected ok=true, got %v", resp)
	}

	// 验证数据库更新
	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if updated.AgentType != "lightclawace" {
		t.Errorf("expected agent_type=lightclawace, got %s", updated.AgentType)
	}
}

func TestHandleUpdateImage_Success_AgentVersion(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{ImageId: "img-001", ImageName: "Test Image", AgentType: "openclaw", AgentVersion: "2026.1.1"}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_version", "2026.4.11")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if updated.AgentVersion != "2026.4.11" {
		t.Errorf("expected agent_version=2026.4.11, got %s", updated.AgentVersion)
	}
}

func TestHandleUpdateImage_Success_BothFields(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{ImageId: "img-001", ImageName: "Test Image", AgentType: "openclaw", AgentVersion: "2026.1.1"}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_type", "hermes")
	form.Set("agent_version", "1.0.0")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if updated.AgentType != "hermes" {
		t.Errorf("expected agent_type=hermes, got %s", updated.AgentType)
	}
	if updated.AgentVersion != "1.0.0" {
		t.Errorf("expected agent_version=1.0.0, got %s", updated.AgentVersion)
	}
}

// ==================== HandleEnableImage Tests (V2 新增逻辑) ====================

func TestHandleEnableImage_ImageNotFound(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	req := adminImagesReq(http.MethodPost, "/api/admin/images/enable?id=999", "")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleEnableImage(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandleEnableImage_CannotEnableInvalidType(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 创建一个有无效 agent_type 的镜像（模拟脏数据）
	img := model.AIImage{
		ImageId:      "img-001",
		ImageName:    "Bad Type Image",
		AgentType:    "invalid_type",
		AgentVersion: "1.0.0",
		Enabled:      false,
	}
	model.DB(context.Background()).Create(&img)

	req := adminImagesReq(http.MethodPost, "/api/admin/images/enable?id=1", "")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleEnableImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleEnableImage_CannotEnableWithoutVersion(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 有 agent_type 但没有 agent_version
	img := model.AIImage{
		ImageId:      "img-001",
		ImageName:    "No Version Image",
		AgentType:    "hermes",
		AgentVersion: "",
		Enabled:      false,
	}
	model.DB(context.Background()).Create(&img)

	req := adminImagesReq(http.MethodPost, "/api/admin/images/enable?id=1", "")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleEnableImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleEnableImage_EnableSuccess(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{
		ImageId:      "img-001",
		ImageName:    "OpenClaw Image",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      false,
	}
	model.DB(context.Background()).Create(&img)

	req := adminImagesReq(http.MethodPost, "/api/admin/images/enable?id=1", "")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleEnableImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// 验证数据库已启用
	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if !updated.Enabled {
		t.Error("expected image to be enabled")
	}
}

func TestHandleEnableImage_DisableProtected(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 创建 site config 设置 default agent type
	config := model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}
	model.DB(context.Background()).Create(&config)

	// 创建唯一启用的 openclaw 镜像
	img := model.AIImage{
		ImageId:      "img-001",
		ImageName:    "OpenClaw Image",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(&img)

	// 尝试禁用该唯一启用镜像 — 应被保护
	req := adminImagesReq(http.MethodPost, "/api/admin/images/enable?id=1", "")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleEnableImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 (protected), got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleEnableImage_MutualExclusion(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 创建两个同类型镜像，第一个启用
	img1 := model.AIImage{
		ImageId:      "img-001",
		ImageName:    "OpenClaw Image 1",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      true,
	}
	img2 := model.AIImage{
		ImageId:      "img-002",
		ImageName:    "OpenClaw Image 2",
		AgentType:    "openclaw",
		AgentVersion: "2026.4.11",
		Enabled:      false,
	}
	model.DB(context.Background()).Create(&img1)
	model.DB(context.Background()).Create(&img2)

	// 启用第二个，应自动禁用第一个
	req := adminImagesReq(http.MethodPost, "/api/admin/images/enable?id=2", "")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleEnableImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var first, second model.AIImage
	model.DB(context.Background()).First(&first, 1)
	model.DB(context.Background()).First(&second, 2)
	if first.Enabled {
		t.Error("expected first image to be disabled after enabling second of same type")
	}
	if !second.Enabled {
		t.Error("expected second image to be enabled")
	}
}

func TestHandleEnableImage_LegacyImageMutualExclusion(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 创建一个存量镜像（空类型）和一个 openclaw 镜像
	legacy := model.AIImage{
		ImageId:   "img-legacy",
		ImageName: "Legacy Image",
		AgentType: "",
		Enabled:   true,
	}
	newImg := model.AIImage{
		ImageId:      "img-openclaw",
		ImageName:    "OpenClaw Image",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      false,
	}
	model.DB(context.Background()).Create(&legacy)
	model.DB(context.Background()).Create(&newImg)

	// 启用 openclaw 镜像，应禁用存量镜像
	req := adminImagesReq(http.MethodPost, "/api/admin/images/enable?id=2", "")
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	HandleEnableImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var legacyUpdated model.AIImage
	model.DB(context.Background()).First(&legacyUpdated, 1)
	if legacyUpdated.Enabled {
		t.Error("expected legacy image to be disabled after enabling openclaw image")
	}
}

// ==================== HandleImportImage Tests (V2 前置校验) ====================

func TestHandleImportImage_EmptyImageId(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("image_id", "")
	form.Set("agent_type", "openclaw")
	form.Set("agent_version", "2026.1.1")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/import", form.Encode())
	rr := httptest.NewRecorder()

	HandleImportImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleImportImage_MissingAgentType(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("image_id", "img-test-001")
	form.Set("agent_version", "2026.1.1")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/import", form.Encode())
	rr := httptest.NewRecorder()

	HandleImportImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleImportImage_InvalidAgentType(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("image_id", "img-test-001")
	form.Set("agent_type", "invalid_type")
	form.Set("agent_version", "1.0.0")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/import", form.Encode())
	rr := httptest.NewRecorder()

	HandleImportImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleImportImage_MissingAgentVersion(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("image_id", "img-test-001")
	form.Set("agent_type", "openclaw")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/import", form.Encode())
	rr := httptest.NewRecorder()

	HandleImportImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleImportImage_InvalidVersionFormat(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	form := url.Values{}
	form.Set("image_id", "img-test-001")
	form.Set("agent_type", "openclaw")
	form.Set("agent_version", "bad-version")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/import", form.Encode())
	rr := httptest.NewRecorder()

	HandleImportImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleImportImage_DuplicateImageId(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 先创建一个镜像
	existing := model.AIImage{
		ImageId:      "img-test-001",
		ImageName:    "Existing Image",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
	}
	model.DB(context.Background()).Create(&existing)

	// 尝试导入相同 image_id
	form := url.Values{}
	form.Set("image_id", "img-test-001")
	form.Set("agent_type", "openclaw")
	form.Set("agent_version", "2026.4.11")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/import", form.Encode())
	rr := httptest.NewRecorder()

	HandleImportImage(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected status 409 (conflict), got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// ==================== HandleAdminImages Tests (JSON) ====================

func TestHandleAdminImages_JSON(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 创建测试数据
	config := model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}
	model.DB(context.Background()).Create(&config)

	images := []model.AIImage{
		{ImageId: "img-001", ImageName: "OpenClaw", AgentType: "openclaw", AgentVersion: "2026.1.1", Enabled: true},
		{ImageId: "img-002", ImageName: "Hermes", AgentType: "hermes", AgentVersion: "1.0.0", Enabled: true},
		{ImageId: "img-003", ImageName: "Legacy", AgentType: "", AgentVersion: "", Enabled: false},
	}
	for _, img := range images {
		model.DB(context.Background()).Create(&img)
	}

	req := adminImagesReq(http.MethodGet, "/api/admin/images", "")
	rr := httptest.NewRecorder()

	HandleAdminImages(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	// 验证 images 列表
	imgList, ok := resp["images"].([]interface{})
	if !ok || len(imgList) != 3 {
		t.Errorf("expected 3 images, got %v", resp["images"])
	}

	// 验证 enabled_images_by_type
	enabledMap, ok := resp["enabled_images_by_type"].(map[string]interface{})
	if !ok {
		t.Fatal("expected enabled_images_by_type map")
	}
	if _, ok := enabledMap["openclaw"]; !ok {
		t.Error("expected openclaw in enabled_images_by_type")
	}
	if _, ok := enabledMap["hermes"]; !ok {
		t.Error("expected hermes in enabled_images_by_type")
	}

	// 验证 default_agent_type
	if resp["default_agent_type"] != "openclaw" {
		t.Errorf("expected default_agent_type=openclaw, got %v", resp["default_agent_type"])
	}
}

// ==================== 新增覆盖测试 ====================

// TestHandleEnableImage_DisableNonDefaultType 禁用非首选类型的镜像应成功（不受保护）
func TestHandleEnableImage_DisableNonDefaultType(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 设置首选类型为 openclaw
	config := model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}
	model.DB(context.Background()).Create(&config)

	// 创建一个已启用的 hermes 镜像（非首选类型）
	img := model.AIImage{
		ImageId:      "img-hermes-001",
		ImageName:    "Hermes Image",
		AgentType:    "hermes",
		AgentVersion: "1.0.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(&img)

	// 禁用该 hermes 镜像 — 应成功（count <= 1 但不是首选类型）
	req := adminImagesReq(http.MethodPost, "/api/admin/images/enable?id=1", "")
	rr := httptest.NewRecorder()

	HandleEnableImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// 验证镜像已被禁用
	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if updated.Enabled {
		t.Error("expected hermes image to be disabled (non-default type should not be protected)")
	}
}

// TestHandleEnableImage_EnableLegacyDisablesOpenclaw 启用存量镜像（空类型）应禁用 openclaw 类型
func TestHandleEnableImage_EnableLegacyDisablesOpenclaw(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 创建一个已启用的 openclaw 镜像
	openclawImg := model.AIImage{
		ImageId:      "img-openclaw-001",
		ImageName:    "OpenClaw Image",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(&openclawImg)

	// 创建一个禁用的存量镜像（空类型、空版本 = legacy）
	legacyImg := model.AIImage{
		ImageId:      "img-legacy-001",
		ImageName:    "Legacy Image",
		AgentType:    "",
		AgentVersion: "",
		Enabled:      false,
	}
	model.DB(context.Background()).Create(&legacyImg)

	// 启用存量镜像 — 应同时禁用 openclaw 镜像（空类型视为 openclaw 互斥）
	req := adminImagesReq(http.MethodPost, "/api/admin/images/enable?id=2", "")
	rr := httptest.NewRecorder()

	HandleEnableImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// 验证 openclaw 镜像已被禁用
	var ocImg model.AIImage
	model.DB(context.Background()).First(&ocImg, 1)
	if ocImg.Enabled {
		t.Error("expected openclaw image to be disabled after enabling legacy image")
	}

	// 验证存量镜像已启用
	var lgImg model.AIImage
	model.DB(context.Background()).First(&lgImg, 2)
	if !lgImg.Enabled {
		t.Error("expected legacy image to be enabled")
	}
}

// TestHandleUpdateImage_SuccessWithVersionValidation 更新镜像时 agent_type + agent_version 都合法
func TestHandleUpdateImage_SuccessWithVersionValidation(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{
		ImageId:      "img-001",
		ImageName:    "Test Image",
		AgentType:    "hermes",
		AgentVersion: "0.9.0",
	}
	model.DB(context.Background()).Create(&img)

	// 同时更新 agent_type 为 openclaw，agent_version 为合法的 YYYY.M.D 格式
	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_type", "openclaw")
	form.Set("agent_version", "2026.4.17")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if updated.AgentType != "openclaw" {
		t.Errorf("expected agent_type=openclaw, got %s", updated.AgentType)
	}
	if updated.AgentVersion != "2026.4.17" {
		t.Errorf("expected agent_version=2026.4.17, got %s", updated.AgentVersion)
	}
}

// TestHandleImportImage_ValidateAgentVersionError 导入镜像时版本格式不匹配类型
func TestHandleImportImage_ValidateAgentVersionError(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// agent_type=openclaw 要求 YYYY.M.D 格式，传入 semver 格式应报错
	form := url.Values{}
	form.Set("image_id", "img-test-002")
	form.Set("agent_type", "openclaw")
	form.Set("agent_version", "1.0.0") // semver 格式，不符合 openclaw 的 YYYY.M.D

	req := adminImagesReq(http.MethodPost, "/api/admin/images/import", form.Encode())
	rr := httptest.NewRecorder()

	HandleImportImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// 确认错误信息包含格式提示
	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if errMsg, ok := resp["error"].(string); !ok || !strings.Contains(errMsg, "YYYY.M.D") {
		t.Errorf("expected error about YYYY.M.D format, got %v", resp)
	}
}

// TestHandleSetDefaultAgentType_SuccessReturnsOldDefault 切换首选类型成功并验证旧默认值被替换
func TestHandleSetDefaultAgentType_SuccessReturnsOldDefault(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 初始首选类型为 openclaw
	config := model.SiteConfig{ID: 1, DefaultAgentType: "openclaw"}
	model.DB(context.Background()).Create(&config)

	// 创建启用的 openclaw 镜像（旧首选）
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:      "img-oc-001",
		ImageName:    "OpenClaw Image",
		AgentType:    "openclaw",
		AgentVersion: "2026.1.1",
		Enabled:      true,
	})

	// 创建启用的 hermes 镜像（新首选）
	model.DB(context.Background()).Create(&model.AIImage{
		ImageId:      "img-hermes-001",
		ImageName:    "Hermes Image",
		AgentType:    "hermes",
		AgentVersion: "1.0.0",
		Enabled:      true,
	})

	// 切换到 hermes
	form := url.Values{}
	form.Set("agent_type", "hermes")
	req := adminImagesReq(http.MethodPost, "/api/admin/images/set-default-agent-type", form.Encode())
	rr := httptest.NewRecorder()

	HandleSetDefaultAgentType(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d, body: %s", rr.Code, rr.Body.String())
	}

	// 验证数据库中首选类型已更新
	var updated model.SiteConfig
	model.DB(context.Background()).First(&updated)
	if updated.DefaultAgentType != "hermes" {
		t.Errorf("expected default_agent_type=hermes, got %s", updated.DefaultAgentType)
	}

	// 再切换回 openclaw，验证可以来回切
	form2 := url.Values{}
	form2.Set("agent_type", "openclaw")
	req2 := adminImagesReq(http.MethodPost, "/api/admin/images/set-default-agent-type", form2.Encode())
	rr2 := httptest.NewRecorder()

	HandleSetDefaultAgentType(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("expected status 200 on switch back, got %d, body: %s", rr2.Code, rr2.Body.String())
	}

	model.DB(context.Background()).First(&updated)
	if updated.DefaultAgentType != "openclaw" {
		t.Errorf("expected default_agent_type=openclaw after switch back, got %s", updated.DefaultAgentType)
	}
}

// TestHandleImportImage_CheckAgentVersionValidError 导入镜像时版本号含非法字符（触发 checkAgentVersionValid）
func TestHandleImportImage_CheckAgentVersionValidError(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 版本号含特殊字符，checkAgentVersionValid 通过正则校验拒绝
	form := url.Values{}
	form.Set("image_id", "img-test-003")
	form.Set("agent_type", "hermes")
	form.Set("agent_version", "v1.0.0; rm -rf /")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/import", form.Encode())
	rr := httptest.NewRecorder()

	HandleImportImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid version chars, got %d, body: %s", rr.Code, rr.Body.String())
	}
}

// ==================== HandleUpdateImage 安全防护测试 ====================

// TestHandleUpdateImage_RejectCandidateImage 公共镜像禁止编辑
func TestHandleUpdateImage_RejectCandidateImage(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// img-idzg74s9 是 CandidateImages 中的公共镜像
	img := model.AIImage{
		ImageId:      "img-idzg74s9",
		ImageName:    "OpenClaw on Ubuntu 24.04",
		AgentType:    "openclaw",
		AgentVersion: "2026.4.11",
	}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_version", "2026.5.1")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for candidate image, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if updated.AgentVersion != "2026.4.11" {
		t.Errorf("expected version unchanged, got %s", updated.AgentVersion)
	}
}

// TestHandleUpdateImage_RejectTypeChangeOnEnabled 启用中镜像禁止修改 agent_type
func TestHandleUpdateImage_RejectTypeChangeOnEnabled(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{
		ImageId:      "img-custom-001",
		ImageName:    "Custom Image",
		AgentType:    "hermes",
		AgentVersion: "0.9.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_type", "lightclawace")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for type change on enabled image, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if updated.AgentType != "hermes" {
		t.Errorf("expected agent_type unchanged, got %s", updated.AgentType)
	}
}

// TestHandleUpdateImage_AllowVersionChangeOnEnabled 启用中镜像允许修改 agent_version（同类型版本升级）
func TestHandleUpdateImage_AllowVersionChangeOnEnabled(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{
		ImageId:      "img-custom-001",
		ImageName:    "Custom Image",
		AgentType:    "hermes",
		AgentVersion: "0.9.0",
		Enabled:      true,
	}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_version", "1.0.0")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for version change on enabled image, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if updated.AgentVersion != "1.0.0" {
		t.Errorf("expected agent_version=1.0.0, got %s", updated.AgentVersion)
	}
}

// TestHandleUpdateImage_RejectTypeVersionMismatch type+version 组合格式不匹配时拒绝
func TestHandleUpdateImage_RejectTypeVersionMismatch(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	// 原本是 hermes/semver，单独改类型为 openclaw 但版本仍是 semver，应拒绝
	img := model.AIImage{
		ImageId:      "img-custom-001",
		ImageName:    "Custom Image",
		AgentType:    "hermes",
		AgentVersion: "0.9.0",
	}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_type", "openclaw")
	// 不提供 agent_version，沿用 0.9.0 即 openclaw 类型下的非法版本

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for type/version mismatch, got %d, body: %s", rr.Code, rr.Body.String())
	}

	var updated model.AIImage
	model.DB(context.Background()).First(&updated, 1)
	if updated.AgentType != "hermes" {
		t.Errorf("expected agent_type unchanged (rollback), got %s", updated.AgentType)
	}
}

// TestHandleUpdateImage_NoActualChangeFails 传入与现值相同的字段时应 400（没有实际变更）
func TestHandleUpdateImage_NoActualChangeFails(t *testing.T) {
	cleanup := initImagesTestDB(t)
	defer cleanup()

	img := model.AIImage{
		ImageId:      "img-custom-001",
		ImageName:    "Custom Image",
		AgentType:    "hermes",
		AgentVersion: "0.9.0",
	}
	model.DB(context.Background()).Create(&img)

	form := url.Values{}
	form.Set("id", "1")
	form.Set("agent_type", "hermes")
	form.Set("agent_version", "0.9.0")

	req := adminImagesReq(http.MethodPost, "/api/admin/images/update", form.Encode())
	rr := httptest.NewRecorder()

	HandleUpdateImage(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when no actual change, got %d, body: %s", rr.Code, rr.Body.String())
	}
}
