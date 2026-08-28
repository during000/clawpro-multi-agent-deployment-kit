package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hatchery/i18n"
	"hatchery/model"

	"github.com/glebarez/sqlite"
	"github.com/gorilla/sessions"
	"gorm.io/gorm"
)

// initAdminChannelsTestDB 初始化 admin channels 测试 DB。
func initAdminChannelsTestDB(t *testing.T) func() {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.CustomAgentType{}, &model.User{}, &model.AIChannel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	origDB := model.UseDBForTest(db)
	AdminToken = "test-admin-token"
	origStore := Store
	Store = sessions.NewCookieStore([]byte("test-secret-key-32-bytes-long!!!"))

	return func() {
		origDB()
		Store = origStore
	}
}

// adminChannelsReq 构造带 admin token 的请求。
func adminChannelsReq(method, path string, body []byte) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	return req
}

// ─── HandleAdminChannels ────────────────────────────────────────────────

func TestHandleAdminChannels_RequiresAdmin(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/admin/channels", nil)
	req.Header.Set("Accept", "application/json") // 无 admin token
	rr := httptest.NewRecorder()
	HandleAdminChannels(rr, req)

	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("无 admin token 应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleAdminChannels_JSON(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	// 插入一条 channel
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "feishu", Name: "Feishu"})

	req := adminChannelsReq(http.MethodGet, "/admin/channels", nil)
	rr := httptest.NewRecorder()
	HandleAdminChannels(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Channels []map[string]interface{} `json:"channels"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, rr.Body.String())
	}
	if len(resp.Channels) != 1 {
		t.Errorf("应返回 1 个 channel，实际=%d", len(resp.Channels))
	}
	// 验证 agent_types 字段存在
	if _, ok := resp.Channels[0]["agent_types"]; !ok {
		t.Errorf("响应应包含 agent_types 字段，实际=%v", resp.Channels[0])
	}
}

func TestHandleAdminChannels_FiltersOverseasOnlyChannelsBySiteScope(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()
	i18n.SetDefaultLang("zh")
	defer i18n.SetDefaultLang("zh")

	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "feishu", Name: "Feishu"})
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "slack", Name: "Slack"})
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "msteams", Name: "Microsoft Teams"})

	req := adminChannelsReq(http.MethodGet, "/admin/channels", nil)
	rr := httptest.NewRecorder()
	HandleAdminChannels(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Channels []map[string]interface{} `json:"channels"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v body=%s", err, rr.Body.String())
	}
	if hasChannel(resp.Channels, "slack") {
		t.Fatalf("domestic site should hide slack, channels=%v", resp.Channels)
	}
	if !hasChannel(resp.Channels, "msteams") {
		t.Fatalf("domestic site should show msteams (all-scope), channels=%v", resp.Channels)
	}
	if !hasChannel(resp.Channels, "feishu") {
		t.Fatalf("domestic site should keep feishu, channels=%v", resp.Channels)
	}

	i18n.SetDefaultLang("en")
	rr = httptest.NewRecorder()
	HandleAdminChannels(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("海外站点应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}
	resp.Channels = nil
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析海外响应失败: %v body=%s", err, rr.Body.String())
	}
	for _, ch := range []string{"slack", "msteams"} {
		if !hasChannel(resp.Channels, ch) {
			t.Fatalf("overseas site should show %s, channels=%v", ch, resp.Channels)
		}
	}
}

func hasChannel(channels []map[string]interface{}, channelID string) bool {
	for _, ch := range channels {
		if got, _ := ch["ChannelID"].(string); got == channelID {
			return true
		}
	}
	return false
}

// ─── HandleToggleChannel ────────────────────────────────────────────────

func TestHandleToggleChannel_RequiresAdmin(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/channels/toggle?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleToggleChannel(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("无 admin token 应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleToggleChannel_NotFound(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	req := adminChannelsReq(http.MethodPost, "/admin/channels/toggle?id=999", nil)
	rr := httptest.NewRecorder()
	HandleToggleChannel(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("通道不存在应返回 404，实际=%d", rr.Code)
	}
}

func TestHandleToggleChannel_FlipsEnabled(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	// 初始 enabled=false
	enabled := false
	ch := &model.AIChannel{ChannelID: "feishu", Name: "F", Enabled: &enabled}
	model.DB(context.Background()).Create(ch)

	req := adminChannelsReq(http.MethodPost, fmt.Sprintf("/admin/channels/toggle?id=%d", ch.ID), nil)
	rr := httptest.NewRecorder()
	HandleToggleChannel(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 验证 DB 中 enabled 被翻转为 true
	var updated model.AIChannel
	model.DB(context.Background()).First(&updated, ch.ID)
	if updated.Enabled == nil || !*updated.Enabled {
		t.Errorf("enabled 应被翻转为 true，实际=%v", updated.Enabled)
	}
}

// ─── HandleAddChannel ───────────────────────────────────────────────────

func TestHandleAddChannel_RequiresAdmin(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	body := []byte(`{"channel_id":"custom_x","name":"X","custom_config":{"server":{"a":"b"},"cred_fields":[]}}`)
	req := httptest.NewRequest(http.MethodPost, "/admin/channels/add", bytes.NewReader(body))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("无 admin token 应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleAddChannel_MethodNotAllowed(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	req := adminChannelsReq(http.MethodGet, "/admin/channels/add", nil)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleAddChannel_InvalidJSON(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", []byte(`not-a-json`))
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 JSON 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAddChannel_EmptyChannelID(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	body := []byte(`{"channel_id":"","name":"X","custom_config":{"server":{"a":"b"}}}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 channel_id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAddChannel_InvalidChannelID(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	// 含特殊字符
	body := []byte(`{"channel_id":"bad-channel!","name":"X","custom_config":{"server":{"a":"b"}}}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 channel_id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAddChannel_EmptyName(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	body := []byte(`{"channel_id":"good_id","name":"","custom_config":{"server":{"a":"b"}}}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 name 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAddChannel_MissingCustomConfig(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	body := []byte(`{"channel_id":"good_id","name":"X"}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺 custom_config 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleAddChannel_InvalidCustomConfig(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	body := []byte(`{"channel_id":"good_id","name":"X","custom_config":"not-an-object"}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 custom_config 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddChannel_EmptyServer(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	// 空 server 现在允许通过（DefaultsForChannel 会在运行时填充默认值），应返回 200
	body := []byte(`{"channel_id":"good_id","name":"X","custom_config":{"server":null,"cred_fields":[]}}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("空 server 应返回 200（DefaultsForChannel 兜底），实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddChannel_InvalidCredFieldKey(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	body := []byte(`{"channel_id":"good_id","name":"X","custom_config":{"server":{"a":"b"},"cred_fields":[{"key":"bad-key!","label":"L"}]}}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("非法 cred_field key 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddChannel_EmptyCredFieldLabel(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	body := []byte(`{"channel_id":"good_id","name":"X","custom_config":{"server":{"a":"b"},"cred_fields":[{"key":"good","label":""}]}}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("空 label 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddChannel_DuplicateCredFieldKey(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	body := []byte(`{"channel_id":"good_id","name":"X","custom_config":{"server":{"a":"b"},"cred_fields":[{"key":"dup","label":"A"},{"key":"dup","label":"B"}]}}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("重复 cred_field key 应返回 400，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddChannel_Conflict(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	// 预先创建
	model.DB(context.Background()).Create(&model.AIChannel{ChannelID: "exists", Name: "existing"})

	body := []byte(`{"channel_id":"exists","name":"X","custom_config":{"server":{"a":"b"},"cred_fields":[]}}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("已存在应返回 409，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAddChannel_Success(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	body := []byte(`{"channel_id":"newcustom","name":"New Custom","custom_config":{"server":{"endpoint":"https://example.com"},"cred_fields":[{"key":"token","label":"Token"}]}}`)
	req := adminChannelsReq(http.MethodPost, "/admin/channels/add", body)
	rr := httptest.NewRecorder()
	HandleAddChannel(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 验证 DB 中已创建
	var ch model.AIChannel
	if err := model.DB(context.Background()).Where("channel_id = ?", "newcustom").First(&ch).Error; err != nil {
		t.Fatalf("查询新建 channel 失败: %v", err)
	}
	if !ch.Custom {
		t.Errorf("新建 channel 应为 Custom=true，实际=%v", ch.Custom)
	}
	if ch.Enabled == nil || *ch.Enabled {
		t.Errorf("新建 channel 默认应为 Enabled=false（等管理员启用），实际=%v", ch.Enabled)
	}
}

// ─── HandleDeleteChannel ────────────────────────────────────────────────

func TestHandleDeleteChannel_RequiresAdmin(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/admin/channels/delete?id=1", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()
	HandleDeleteChannel(rr, req)
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("无 admin token 应返回 401/403，实际=%d", rr.Code)
	}
}

func TestHandleDeleteChannel_MethodNotAllowed(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	req := adminChannelsReq(http.MethodGet, "/admin/channels/delete", nil)
	rr := httptest.NewRecorder()
	HandleDeleteChannel(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET 应返回 405，实际=%d", rr.Code)
	}
}

func TestHandleDeleteChannel_MissingID(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	req := adminChannelsReq(http.MethodPost, "/admin/channels/delete", nil)
	rr := httptest.NewRecorder()
	HandleDeleteChannel(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("缺 id 应返回 400，实际=%d", rr.Code)
	}
}

func TestHandleDeleteChannel_NotFound(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	req := adminChannelsReq(http.MethodPost, "/admin/channels/delete?id=9999", nil)
	rr := httptest.NewRecorder()
	HandleDeleteChannel(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("通道不存在应返回 404，实际=%d", rr.Code)
	}
}

func TestHandleDeleteChannel_ForbidPredefined(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	// 预定义 channel（Custom=false）
	ch := &model.AIChannel{ChannelID: "feishu", Name: "F", Custom: false}
	model.DB(context.Background()).Create(ch)

	req := adminChannelsReq(http.MethodPost,
		fmt.Sprintf("/admin/channels/delete?id=%d", ch.ID), nil)
	rr := httptest.NewRecorder()
	HandleDeleteChannel(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("预定义通道应返回 403，实际=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDeleteChannel_SuccessCustom(t *testing.T) {
	cleanup := initAdminChannelsTestDB(t)
	defer cleanup()

	// 自定义 channel
	ch := &model.AIChannel{ChannelID: "custom_ok", Name: "C", Custom: true}
	model.DB(context.Background()).Create(ch)

	req := adminChannelsReq(http.MethodPost,
		fmt.Sprintf("/admin/channels/delete?id=%d", ch.ID), nil)
	rr := httptest.NewRecorder()
	HandleDeleteChannel(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("应返回 200，实际=%d body=%s", rr.Code, rr.Body.String())
	}

	// 验证已被 unscoped 删除（不是软删）
	var count int64
	model.DB(context.Background()).Unscoped().Model(&model.AIChannel{}).Where("id = ?", ch.ID).Count(&count)
	if count != 0 {
		t.Errorf("channel 应被硬删除，实际 count=%d", count)
	}
	if !strings.Contains(rr.Body.String(), "ok") {
		t.Errorf("响应应含 ok，实际=%s", rr.Body.String())
	}
}
